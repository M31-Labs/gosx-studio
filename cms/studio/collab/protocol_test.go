package collab

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/authoring"
)

func TestDecodeEnvelopeRejectsUnknownOversizedBinaryAndSpoofFields(t *testing.T) {
	valid := `{"protocol":"gosx.studio.collaboration","version":1,"type":"studio.hello"}`
	if _, protocolErr := DecodeEnvelope([]byte(valid), false); protocolErr != nil {
		t.Fatalf("valid hello: %v", protocolErr)
	}
	cases := []struct {
		name   string
		frame  []byte
		binary bool
		code   ErrorCode
	}{
		{"empty", nil, false, ErrorInvalidRequest},
		{"unknown", []byte(`{"protocol":"gosx.studio.collaboration","version":1,"type":"studio.magic"}`), false, ErrorInvalidRequest},
		{"spoof actor", []byte(`{"protocol":"gosx.studio.collaboration","version":1,"type":"studio.hello","actor":"mallory"}`), false, ErrorInvalidRequest},
		{"spoof capabilities", []byte(`{"protocol":"gosx.studio.collaboration","version":1,"type":"studio.hello","capabilities":["publish"]}`), false, ErrorInvalidRequest},
		{"oversize", []byte(strings.Repeat("x", MaxFrameBytes+1)), false, ErrorInvalidRequest},
		{"binary", []byte{1, 2, 3}, true, ErrorBinaryUnsupported},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, got := DecodeEnvelope(tc.frame, tc.binary)
			if got == nil || got.Code != tc.code {
				t.Fatalf("got %#v, want %s", got, tc.code)
			}
		})
	}
}

func TestOperationSubmitRequiresCanonicalCanvasIdentity(t *testing.T) {
	request := authoring.OperationRequest{ID: "op", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Field: "hero.headline"}}
	if got := (OperationSubmit{Request: request}).Validate(); got != nil {
		t.Fatalf("field locator should normalize to root route: %v", got)
	}
	request = authoring.OperationRequest{ID: "op", Kind: authoring.OperationSetStyle, Target: authoring.OperationTarget{Route: "/", ComponentKey: "home:hero", Property: "color"}}
	if got := (OperationSubmit{Request: request}).Validate(); got != nil {
		t.Fatalf("style block locator should be stable: %v", got)
	}
}

func TestOperationSubmitRejectsOversizedContent(t *testing.T) {
	request := authoring.OperationRequest{ID: "large", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Field: "title"}, Value: strings.Repeat("x", MaxFrameBytes)}
	if got := (OperationSubmit{Request: request}).Validate(); got == nil || got.Code != ErrorInvalidRequest {
		t.Fatalf("oversized operation=%#v", got)
	}
}

func TestServiceFixesResourceOutsideOperationWire(t *testing.T) {
	resource := ResourceKey{TenantID: "tenant", ProjectID: "project", Kind: "site", ID: "main"}
	store := &captureOperationStore{}
	service, err := NewService(resource, store)
	if err != nil {
		t.Fatal(err)
	}
	_, pe := service.Submit(t.Context(), Principal{ActorID: "actor"}, authoring.OperationRequest{ID: "op", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Field: "title"}})
	if pe != nil {
		t.Fatal(pe)
	}
	if store.command.Resource != resource.Normalize() {
		t.Fatalf("resource=%#v", store.command.Resource)
	}
}

type captureOperationStore struct{ command ApplyCommand }

func (*captureOperationStore) SeedHeads(context.Context, ResourceKey, []SeedHead) error { return nil }

func (s *captureOperationStore) Apply(_ context.Context, c ApplyCommand) (OperationAck, *ProtocolError) {
	s.command = c
	return OperationAck{}, nil
}
func (*captureOperationStore) Tail(context.Context, ResourceKey, uint64, int) ([]OperationAck, error) {
	return nil, nil
}
func (*captureOperationStore) Attempts(context.Context, ResourceKey) ([]Attempt, error) {
	return nil, nil
}
func (*captureOperationStore) PendingOutbox(context.Context, ResourceKey, int) ([]OutboxEntry, error) {
	return nil, nil
}
func (*captureOperationStore) MarkProjected(context.Context, ResourceKey, int64, time.Time) error {
	return nil
}
func (*captureOperationStore) Close() error { return nil }

func TestOperationWireHasNoPrincipalFields(t *testing.T) {
	raw, err := json.Marshal(OperationSubmit{Request: authoring.OperationRequest{ID: "op", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Field: "title"}}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, forbidden := range []string{"actor", "capabilit", "tenant", "project"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("wire payload leaks %q: %s", forbidden, text)
		}
	}
	principalRaw, err := json.Marshal(Principal{ActorID: "secret", DisplayName: "Secret", Capabilities: map[Capability]bool{CapabilityPublish: true}})
	if err != nil {
		t.Fatal(err)
	}
	if string(principalRaw) != "{}" {
		t.Fatalf("principal must be non-serializable: %s", principalRaw)
	}
	spoof := `{"request":{"schemaVersion":1,"id":"op","kind":"set-field","target":{"field":"title"}},"actor":"mallory"}`
	if _, got := DecodeOperationSubmit([]byte(spoof)); got == nil || got.Code != ErrorInvalidRequest {
		t.Fatalf("spoof payload=%#v", got)
	}
	nestedSpoof := `{"request":{"schemaVersion":1,"id":"op","kind":"set-field","target":{"field":"title"},"capabilities":["publish"]}}`
	if _, got := DecodeOperationSubmit([]byte(nestedSpoof)); got == nil || got.Code != ErrorInvalidRequest {
		t.Fatalf("nested spoof payload=%#v", got)
	}
}

func TestPresenceRegistryIsConnectionKeyedAndSorted(t *testing.T) {
	p := NewPresenceRegistry()
	principal := Principal{ActorID: "same"}
	p.Join("b", principal)
	p.Join("a", principal)
	entries := p.Entries()
	if len(entries) != 2 || entries[0].ConnectionID != "a" || entries[1].ConnectionID != "b" {
		t.Fatalf("entries=%#v", entries)
	}
	entries[0].Principal.Capabilities[CapabilityPublish] = true
	if p.Entries()[0].Principal.Can(CapabilityPublish) {
		t.Fatal("presence snapshots must clone principal capability maps")
	}
	p.Leave("a")
	if entries = p.Entries(); len(entries) != 1 || entries[0].ConnectionID != "b" {
		t.Fatalf("leave entries=%#v", entries)
	}
	if len(NewPresenceRegistry().Entries()) != 0 {
		t.Fatal("fresh registry must be empty after restart")
	}
}

func TestPresenceRegistryConcurrentConnections(t *testing.T) {
	p := NewPresenceRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := string(rune('a' + i))
			p.Join(id, Principal{ActorID: "actor", Capabilities: map[Capability]bool{CapabilityView: true}})
			if i%2 == 0 {
				p.Leave(id)
			}
			_ = p.Entries()
		}(i)
	}
	wg.Wait()
}
