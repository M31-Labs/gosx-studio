package collab_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/studio/collab"
	"m31labs.dev/gosx-studio/cms/studio/collab/sqlstore"
	"m31labs.dev/gosx/hub"
)

type transportFixture struct {
	t          *testing.T
	store      *sqlstore.Store
	service    *collab.Service
	transport  *collab.Transport
	server     *httptest.Server
	principals map[string]collab.Principal
}

func newTransportFixture(t *testing.T) *transportFixture {
	t.Helper()
	store, err := sqlstore.Open(filepath.Join(t.TempDir(), "studio.db"))
	if err != nil {
		t.Fatal(err)
	}
	resource := collab.ResourceKey{TenantID: "tenant", ProjectID: "project", Kind: "site", ID: "main"}
	service, err := collab.NewService(resource, store)
	if err != nil {
		t.Fatal(err)
	}
	f := &transportFixture{t: t, store: store, service: service, principals: map[string]collab.Principal{
		"author":   {ActorID: "actor-author", DisplayName: "Author", Capabilities: map[collab.Capability]bool{collab.CapabilityView: true, collab.CapabilityAuthor: true}},
		"designer": {ActorID: "actor-designer", DisplayName: "Designer", Capabilities: map[collab.Capability]bool{collab.CapabilityView: true, collab.CapabilityDesign: true}},
		"viewer":   {ActorID: "actor-viewer", DisplayName: "Viewer", Capabilities: map[collab.Capability]bool{collab.CapabilityView: true}},
	}}
	transport, err := collab.NewTransport(collab.TransportOptions{Service: service, Authenticate: func(r *http.Request) (collab.Principal, error) {
		p, ok := f.principals[r.Header.Get("X-Test-Session")]
		if !ok {
			return collab.Principal{}, errors.New("no session")
		}
		return p, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	f.transport = transport
	f.server = httptest.NewServer(transport)
	t.Cleanup(func() { f.server.Close(); _ = store.Close() })
	return f
}

func (f *transportFixture) dial(session string, query string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(f.server.URL)
	if err != nil {
		return nil, nil, err
	}
	u.Scheme = "ws"
	u.RawQuery = query
	header := http.Header{}
	header.Set("X-Test-Session", session)
	return websocket.DefaultDialer.Dial(u.String(), header)
}

func writeEvent(t *testing.T, connection *websocket.Conn, event string, data any) {
	t.Helper()
	if err := connection.WriteJSON(hub.Message{Event: event, Data: mustRaw(t, data)}); err != nil {
		t.Fatal(err)
	}
}
func mustRaw(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func readEvent(t *testing.T, connection *websocket.Conn, want string) json.RawMessage {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	for {
		_, raw, err := connection.ReadMessage()
		if err != nil {
			t.Fatalf("read %s: %v", want, err)
		}
		var message hub.Message
		if err := json.Unmarshal(raw, &message); err != nil {
			continue
		}
		if message.Event == want {
			return message.Data
		}
	}
}
func decode[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestTransportRejectsOutsiderBeforeUpgradeAndFixesResource(t *testing.T) {
	f := newTransportFixture(t)
	connection, response, err := f.dial("outsider", "tenant=evil&project=evil")
	if connection != nil {
		connection.Close()
	}
	if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("dial err=%v response=%v", err, response)
	}
	author, _, err := f.dial("author", "tenant=evil&project=evil")
	if err != nil {
		t.Fatal(err)
	}
	defer author.Close()
	hello := decode[collab.Hello](t, readEvent(t, author, collab.EventHello))
	if hello.Resource != f.service.Resource() {
		t.Fatalf("resource selected by query: %#v", hello.Resource)
	}
	if hello.Self.ActorID != "actor-author" || !hello.Permissions.Author || hello.Permissions.Design {
		t.Fatalf("hello=%#v", hello)
	}
}

func TestTransportTwoClientsPermissionsStaleResumePresenceAndBinaryGate(t *testing.T) {
	f := newTransportFixture(t)
	author, _, err := f.dial("author", "")
	if err != nil {
		t.Fatal(err)
	}
	defer author.Close()
	_ = readEvent(t, author, collab.EventHello)
	designer, _, err := f.dial("designer", "")
	if err != nil {
		t.Fatal(err)
	}
	defer designer.Close()
	_ = readEvent(t, designer, collab.EventHello)
	presence := decode[collab.PresenceSnapshot](t, readEvent(t, author, collab.EventPresence))
	for len(presence.Members) < 2 {
		presence = decode[collab.PresenceSnapshot](t, readEvent(t, author, collab.EventPresence))
	}
	if presence.Members[0].ActorID == presence.Members[1].ActorID {
		t.Fatalf("presence=%#v", presence)
	}

	field := authoring.OperationRequest{ID: "field-1", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Route: "/", PageID: "home", Field: "hero.headline"}, Value: "Shared"}
	writeEvent(t, author, collab.EventOperationSubmit, collab.OperationSubmit{Request: field})
	accepted := decode[collab.OperationAck](t, readEvent(t, author, collab.EventOperationAccepted))
	if accepted.Sequence != 1 || accepted.Record.ActorID != "actor-author" {
		t.Fatalf("accepted=%#v", accepted)
	}
	_ = readEvent(t, designer, collab.EventOperationAccepted)

	writeEvent(t, designer, collab.EventOperationSubmit, collab.OperationSubmit{Request: authoring.OperationRequest{ID: "designer-field", Kind: authoring.OperationSetField, Target: field.Target, Value: "no", ExpectedTargetHead: "field-1"}})
	rejected := decode[collab.ProtocolError](t, readEvent(t, designer, collab.EventOperationRejected))
	if rejected.Code != collab.ErrorForbidden {
		t.Fatalf("designer field=%#v", rejected)
	}
	writeEvent(t, author, collab.EventOperationSubmit, collab.OperationSubmit{Request: authoring.OperationRequest{ID: "author-style", Kind: authoring.OperationSetStyle, Target: authoring.OperationTarget{Route: "/", PageID: "home", ComponentKey: "home:hero", Property: "color"}, Value: "red"}})
	rejected = decode[collab.ProtocolError](t, readEvent(t, author, collab.EventOperationRejected))
	if rejected.Code != collab.ErrorForbidden {
		t.Fatalf("author style=%#v", rejected)
	}

	style := authoring.OperationRequest{ID: "style-1", Kind: authoring.OperationSetStyle, Target: authoring.OperationTarget{Route: "/", PageID: "home", ComponentKey: "home:hero", Property: "color"}, Value: "red"}
	writeEvent(t, designer, collab.EventOperationSubmit, collab.OperationSubmit{Request: style})
	styleAck := decode[collab.OperationAck](t, readEvent(t, designer, collab.EventOperationAccepted))
	if styleAck.Sequence != 2 {
		t.Fatalf("style ack=%#v", styleAck)
	}
	_ = readEvent(t, author, collab.EventOperationAccepted)

	stale := field
	stale.ID = "field-stale"
	stale.Value = "Lost"
	writeEvent(t, author, collab.EventOperationSubmit, collab.OperationSubmit{Request: stale})
	rejected = decode[collab.ProtocolError](t, readEvent(t, author, collab.EventOperationRejected))
	if rejected.Code != collab.ErrorStaleField || rejected.CurrentHead != "field-1" {
		t.Fatalf("stale=%#v", rejected)
	}

	selection := collab.SelectionState{Route: "/", PageID: "home", Field: "hero.headline", Viewport: "desktop"}
	writeEvent(t, author, collab.EventSelectionSubmit, selection)
	remoteSelection := decode[collab.SelectionBroadcast](t, readEvent(t, designer, collab.EventSelection))
	if remoteSelection.Source.ActorID != "actor-author" || remoteSelection.Selection.Identity().Field != "hero.headline" {
		t.Fatalf("selection=%#v", remoteSelection)
	}
	writeEvent(t, author, collab.EventCursorSubmit, collab.CursorState{Route: "/", Viewport: "desktop", X: 2, Y: -1})
	remoteCursor := decode[collab.CursorBroadcast](t, readEvent(t, designer, collab.EventCursor))
	if remoteCursor.Cursor.X != 1 || remoteCursor.Cursor.Y != 0 {
		t.Fatalf("cursor=%#v", remoteCursor)
	}

	if err := author.WriteMessage(websocket.BinaryMessage, []byte{1, 2}); err != nil {
		t.Fatal(err)
	}
	binaryError := readEvent(t, author, "__crdt_error")
	if len(binaryError) == 0 {
		t.Fatal("binary path did not fail closed")
	}

	if err := author.Close(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return len(f.service.Presence().Entries()) == 1 })
	reconnected, _, err := f.dial("author", "")
	if err != nil {
		t.Fatal(err)
	}
	defer reconnected.Close()
	_ = readEvent(t, reconnected, collab.EventHello)
	writeEvent(t, reconnected, collab.EventResume, collab.ResumeRequest{AfterSequence: 1})
	tail := decode[collab.TailEnvelope](t, readEvent(t, reconnected, collab.EventTail))
	if len(tail.Operations) != 1 || tail.Operations[0].Sequence != 2 || tail.Operations[0].Record.ID != "style-1" {
		t.Fatalf("tail=%#v", tail)
	}
}

func TestTransportViewerAndSpoofedPrincipalPayloadAreAuthoritativelyRejected(t *testing.T) {
	f := newTransportFixture(t)
	viewer, _, err := f.dial("viewer", "")
	if err != nil {
		t.Fatal(err)
	}
	defer viewer.Close()
	_ = readEvent(t, viewer, collab.EventHello)
	request := authoring.OperationRequest{ID: "viewer-write", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Route: "/", Field: "title"}, Value: "no"}
	writeEvent(t, viewer, collab.EventOperationSubmit, collab.OperationSubmit{Request: request})
	rejected := decode[collab.ProtocolError](t, readEvent(t, viewer, collab.EventOperationRejected))
	if rejected.Code != collab.ErrorForbidden {
		t.Fatalf("viewer=%#v", rejected)
	}
	spoof := json.RawMessage(`{"request":{"schemaVersion":1,"id":"spoof","kind":"set-field","target":{"route":"/","field":"title"},"value":"no"},"actor":"actor-author","capabilities":["author"]}`)
	if err := viewer.WriteJSON(hub.Message{Event: collab.EventOperationSubmit, Data: spoof}); err != nil {
		t.Fatal(err)
	}
	rejected = decode[collab.ProtocolError](t, readEvent(t, viewer, collab.EventOperationRejected))
	if rejected.Code != collab.ErrorInvalidRequest {
		t.Fatalf("spoof=%#v", rejected)
	}
	attempts, err := f.store.Attempts(t.Context(), f.service.Resource())
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 || attempts[0].ActorID != "actor-viewer" || attempts[0].Accepted {
		t.Fatalf("attempts=%#v", attempts)
	}
}

func TestTransportEnforcesServerCursorRateLimit(t *testing.T) {
	f := newTransportFixture(t)
	author, _, err := f.dial("author", "")
	if err != nil {
		t.Fatal(err)
	}
	defer author.Close()
	_ = readEvent(t, author, collab.EventHello)
	for i := 0; i < 31; i++ {
		writeEvent(t, author, collab.EventCursorSubmit, collab.CursorState{Route: "/", Viewport: "desktop", X: float64(i) / 30, Y: .5})
	}
	rejected := decode[collab.ProtocolError](t, readEvent(t, author, collab.EventOperationRejected))
	if rejected.Code != collab.ErrorInvalidRequest || rejected.Message != "cursor update rate exceeded" {
		t.Fatalf("rate rejection=%#v", rejected)
	}
	entries := f.service.Presence().Entries()
	if len(entries) != 1 || entries[0].Cursor.X < 0 || entries[0].Cursor.X > 1 {
		t.Fatalf("presence cursor=%#v", entries)
	}
}

type pagedTransportService struct {
	resource   collab.ResourceKey
	presence   *collab.PresenceRegistry
	operations []collab.OperationAck
}

func (s *pagedTransportService) Resource() collab.ResourceKey       { return s.resource }
func (s *pagedTransportService) Presence() *collab.PresenceRegistry { return s.presence }
func (s *pagedTransportService) Submit(context.Context, collab.Principal, authoring.OperationRequest) (collab.OperationAck, *collab.ProtocolError) {
	return collab.OperationAck{}, collab.NewProtocolError(collab.ErrorForbidden, "unused")
}
func (s *pagedTransportService) Resume(_ context.Context, after uint64, limit int) ([]collab.OperationAck, error) {
	start := int(after)
	if start > len(s.operations) {
		start = len(s.operations)
	}
	end := start + limit
	if end > len(s.operations) {
		end = len(s.operations)
	}
	return append([]collab.OperationAck(nil), s.operations[start:end]...), nil
}

func TestTransportResumePaginatesWithoutDroppingHistory(t *testing.T) {
	service := &pagedTransportService{resource: collab.ResourceKey{TenantID: "t", ProjectID: "p", Kind: "site", ID: "main"}, presence: collab.NewPresenceRegistry()}
	for i := 1; i <= 1001; i++ {
		service.operations = append(service.operations, collab.OperationAck{Sequence: uint64(i), Record: authoring.OperationRecord{ID: fmt.Sprintf("op-%d", i)}})
	}
	transport, err := collab.NewTransport(collab.TransportOptions{Service: service, Authenticate: func(*http.Request) (collab.Principal, error) {
		return collab.Principal{ActorID: "viewer", Capabilities: map[collab.Capability]bool{collab.CapabilityView: true}}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(transport)
	defer server.Close()
	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"
	connection, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_ = readEvent(t, connection, collab.EventHello)
	writeEvent(t, connection, collab.EventResume, collab.ResumeRequest{AfterSequence: 0})
	first := decode[collab.TailEnvelope](t, readEvent(t, connection, collab.EventTail))
	if len(first.Operations) != 1000 || !first.HasMore || first.Operations[999].Sequence != 1000 {
		t.Fatalf("first page len=%d more=%v", len(first.Operations), first.HasMore)
	}
	writeEvent(t, connection, collab.EventResume, collab.ResumeRequest{AfterSequence: 1000})
	second := decode[collab.TailEnvelope](t, readEvent(t, connection, collab.EventTail))
	if len(second.Operations) != 1 || second.HasMore || second.Operations[0].Sequence != 1001 {
		t.Fatalf("second page=%#v", second)
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not reached")
}
