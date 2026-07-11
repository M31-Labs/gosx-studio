package collab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx/hub"
)

const (
	EventHello             = "studio.hello"
	EventPresence          = "studio.presence"
	EventCursorSubmit      = "studio.cursor.submit"
	EventCursor            = "studio.cursors"
	EventSelectionSubmit   = "studio.selection.submit"
	EventSelection         = "studio.selection"
	EventOperationSubmit   = "studio.operation.submit"
	EventOperationAccepted = "studio.operation.accepted"
	EventOperationRejected = "studio.operation.rejected"
	EventResume            = "studio.sync.resume"
	EventTail              = "studio.sync.tail"
)

const (
	metadataActor   = "gosx.studio.actor"
	metadataLabel   = "gosx.studio.label"
	metadataView    = "gosx.studio.capability.view"
	metadataAuthor  = "gosx.studio.capability.author"
	metadataDesign  = "gosx.studio.capability.design"
	metadataPublish = "gosx.studio.capability.publish"
)

type PrincipalResolver func(*http.Request) (Principal, error)

// TransportService lets a host wrap the canonical Service with an outbox
// projection gate. Noni's adapter can therefore withhold accepted broadcast
// until its draft projection succeeds, while still delegating authorization,
// conflict, and sequence semantics to the 03B service.
type TransportService interface {
	Resource() ResourceKey
	Submit(context.Context, Principal, authoring.OperationRequest) (OperationAck, *ProtocolError)
	Resume(context.Context, uint64, int) ([]OperationAck, error)
	Presence() *PresenceRegistry
}

type TransportOptions struct {
	Service      TransportService
	Authenticate PrincipalResolver
	MaxClients   int
}

type Transport struct {
	service      TransportService
	hub          *hub.Hub
	authenticate PrincipalResolver
	rateMu       sync.Mutex
	cursorRates  map[string]cursorRate
}

type cursorRate struct {
	window time.Time
	count  int
}

type SelfPermissions struct {
	View    bool `json:"view"`
	Author  bool `json:"author"`
	Design  bool `json:"design"`
	Publish bool `json:"publish"`
}
type Collaborator struct {
	ConnectionID string `json:"connectionId"`
	ActorID      string `json:"actorId"`
	DisplayName  string `json:"displayName"`
}
type Hello struct {
	Protocol     string          `json:"protocol"`
	Version      int             `json:"version"`
	ConnectionID string          `json:"connectionId"`
	Self         Collaborator    `json:"self"`
	Permissions  SelfPermissions `json:"permissions"`
	Resource     ResourceKey     `json:"resource"`
}
type PresenceSnapshot struct {
	Members []Collaborator `json:"members"`
}
type SelectionBroadcast struct {
	Source    Collaborator   `json:"source"`
	Selection SelectionState `json:"selection"`
}
type CursorBroadcast struct {
	Source Collaborator `json:"source"`
	Cursor CursorState  `json:"cursor"`
}
type TailEnvelope struct {
	Operations []OperationAck `json:"operations"`
	HasMore    bool           `json:"hasMore,omitempty"`
}

func NewTransport(options TransportOptions) (*Transport, error) {
	if options.Service == nil {
		return nil, fmt.Errorf("collaboration service is required")
	}
	if options.Authenticate == nil {
		return nil, fmt.Errorf("collaboration principal resolver is required")
	}
	h := hub.New("studio:" + options.Service.Resource().StableKey())
	h.MaxClients = options.MaxClients
	// This transport registers no SyncDoc. If one is ever added accidentally,
	// inbound binary mutation still fails closed for every connection.
	h.SetBinaryAuthorizer(func(*hub.Client, string) bool { return false })
	t := &Transport{service: options.Service, hub: h, authenticate: options.Authenticate, cursorRates: map[string]cursorRate{}}
	t.register()
	return t, nil
}

func (t *Transport) Hub() *hub.Hub { return t.hub }

func (t *Transport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	principal, err := t.authenticate(r)
	if err != nil || principal.Validate() != nil || !canJoin(principal) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	principal = sanitizePrincipal(principal)
	t.hub.ServeHTTPWithMetadata(w, r, principalMetadata(principal))
}

func canJoin(p Principal) bool {
	return p.Can(CapabilityView) || p.Can(CapabilityAuthor) || p.Can(CapabilityDesign) || p.Can(CapabilityPublish)
}
func sanitizePrincipal(p Principal) Principal {
	p = p.Clone()
	p.ActorID = bounded(strings.TrimSpace(p.ActorID), 128)
	p.DisplayName = bounded(strings.TrimSpace(p.DisplayName), 80)
	if p.DisplayName == "" {
		p.DisplayName = "Collaborator"
	}
	return p
}
func bounded(value string, max int) string {
	r := []rune(value)
	if len(r) > max {
		return string(r[:max])
	}
	return value
}
func principalMetadata(p Principal) hub.ConnectionMetadata {
	return hub.ConnectionMetadata{metadataActor: p.ActorID, metadataLabel: p.DisplayName, metadataView: flag(p.Can(CapabilityView)), metadataAuthor: flag(p.Can(CapabilityAuthor)), metadataDesign: flag(p.Can(CapabilityDesign)), metadataPublish: flag(p.Can(CapabilityPublish))}
}
func flag(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
func principalFromClient(client *hub.Client) (Principal, error) {
	actor, ok := client.Metadata(metadataActor)
	if !ok || strings.TrimSpace(actor) == "" {
		return Principal{}, errors.New("trusted actor metadata is missing")
	}
	label, _ := client.Metadata(metadataLabel)
	p := Principal{ActorID: actor, DisplayName: label, Capabilities: map[Capability]bool{}}
	for capability, key := range map[Capability]string{CapabilityView: metadataView, CapabilityAuthor: metadataAuthor, CapabilityDesign: metadataDesign, CapabilityPublish: metadataPublish} {
		value, _ := client.Metadata(key)
		p.Capabilities[capability] = value == "1"
	}
	return sanitizePrincipal(p), nil
}
func collaborator(client *hub.Client) Collaborator {
	p, _ := principalFromClient(client)
	return Collaborator{ConnectionID: client.ID, ActorID: p.ActorID, DisplayName: p.DisplayName}
}

func (t *Transport) register() {
	t.hub.On("join", func(ctx *hub.Context) {
		p, err := principalFromClient(ctx.Client)
		if err != nil {
			return
		}
		t.service.Presence().Join(ctx.Client.ID, p)
		ctx.Hub.Send(ctx.Client.ID, EventHello, Hello{Protocol: ProtocolName, Version: ProtocolVersion, ConnectionID: ctx.Client.ID, Self: collaborator(ctx.Client), Permissions: permissions(p), Resource: t.service.Resource()})
		t.broadcastPresence()
	})
	t.hub.On("leave", func(ctx *hub.Context) {
		t.service.Presence().Leave(ctx.Client.ID)
		t.rateMu.Lock()
		delete(t.cursorRates, ctx.Client.ID)
		t.rateMu.Unlock()
		t.broadcastPresence()
	})
	t.hub.On(EventOperationSubmit, t.handleOperation)
	t.hub.On(EventResume, t.handleResume)
	t.hub.On(EventSelectionSubmit, t.handleSelection)
	t.hub.On(EventCursorSubmit, t.handleCursor)
}

func permissions(p Principal) SelfPermissions {
	return SelfPermissions{View: p.Can(CapabilityView), Author: p.Can(CapabilityAuthor), Design: p.Can(CapabilityDesign), Publish: p.Can(CapabilityPublish)}
}

func (t *Transport) handleOperation(ctx *hub.Context) {
	p, err := principalFromClient(ctx.Client)
	if err != nil {
		t.reject(ctx, NewProtocolError(ErrorForbidden, "trusted principal metadata is unavailable"))
		return
	}
	submit, protocolErr := DecodeOperationSubmit(ctx.Data)
	if protocolErr != nil {
		t.reject(ctx, protocolErr)
		return
	}
	ack, protocolErr := t.service.Submit(context.Background(), p, submit.Request)
	if protocolErr != nil {
		if protocolErr.OperationID == "" {
			protocolErr.OperationID = submit.Request.ID
		}
		t.reject(ctx, protocolErr)
		return
	}
	ctx.Hub.Broadcast(EventOperationAccepted, ack)
}

func (t *Transport) handleResume(ctx *hub.Context) {
	var request ResumeRequest
	if protocolErr := decodeStrict(ctx.Data, &request); protocolErr != nil {
		t.reject(ctx, protocolErr)
		return
	}
	operations, err := t.service.Resume(context.Background(), request.AfterSequence, 1000)
	if err != nil {
		protocolErr := NewProtocolError(ErrorStoreUnavailable, "collaboration history is unavailable")
		protocolErr.Retryable = true
		t.reject(ctx, protocolErr)
		return
	}
	hasMore := false
	if len(operations) == 1000 {
		probe, probeErr := t.service.Resume(context.Background(), operations[len(operations)-1].Sequence, 1)
		if probeErr != nil {
			protocolErr := NewProtocolError(ErrorStoreUnavailable, "collaboration history is unavailable")
			protocolErr.Retryable = true
			t.reject(ctx, protocolErr)
			return
		}
		hasMore = len(probe) > 0
	}
	ctx.Hub.Send(ctx.Client.ID, EventTail, TailEnvelope{Operations: operations, HasMore: hasMore})
}

func (t *Transport) handleSelection(ctx *hub.Context) {
	var selection SelectionState
	if protocolErr := decodeStrict(ctx.Data, &selection); protocolErr != nil {
		t.reject(ctx, protocolErr)
		return
	}
	selection = selection.Normalize()
	if protocolErr := selection.Validate(); protocolErr != nil {
		t.reject(ctx, protocolErr)
		return
	}
	if !t.service.Presence().SetSelection(ctx.Client.ID, selection) {
		t.reject(ctx, NewProtocolError(ErrorForbidden, "connection is not present"))
		return
	}
	ctx.Hub.Broadcast(EventSelection, SelectionBroadcast{Source: collaborator(ctx.Client), Selection: selection})
}

func (t *Transport) handleCursor(ctx *hub.Context) {
	var cursor CursorState
	if protocolErr := decodeStrict(ctx.Data, &cursor); protocolErr != nil {
		t.reject(ctx, protocolErr)
		return
	}
	if protocolErr := cursor.Validate(); protocolErr != nil {
		t.reject(ctx, protocolErr)
		return
	}
	if !t.allowCursor(ctx.Client.ID, time.Now()) {
		protocolErr := NewProtocolError(ErrorInvalidRequest, "cursor update rate exceeded")
		t.reject(ctx, protocolErr)
		return
	}
	cursor = cursor.Normalize()
	if !t.service.Presence().SetCursor(ctx.Client.ID, cursor) {
		t.reject(ctx, NewProtocolError(ErrorForbidden, "connection is not present"))
		return
	}
	ctx.Hub.Broadcast(EventCursor, CursorBroadcast{Source: collaborator(ctx.Client), Cursor: cursor})
}

func (t *Transport) allowCursor(connectionID string, now time.Time) bool {
	t.rateMu.Lock()
	defer t.rateMu.Unlock()
	rate := t.cursorRates[connectionID]
	if rate.window.IsZero() || now.Sub(rate.window) >= time.Second {
		rate = cursorRate{window: now}
	}
	if rate.count >= 30 {
		return false
	}
	rate.count++
	t.cursorRates[connectionID] = rate
	return true
}

func (t *Transport) reject(ctx *hub.Context, protocolErr *ProtocolError) {
	if protocolErr == nil {
		return
	}
	public := *protocolErr
	if public.Code == ErrorStoreUnavailable {
		public.Message = "collaboration storage is temporarily unavailable"
	}
	ctx.Hub.Send(ctx.Client.ID, EventOperationRejected, &public)
}

func (t *Transport) broadcastPresence() {
	entries := t.service.Presence().Entries()
	members := make([]Collaborator, 0, len(entries))
	for _, entry := range entries {
		members = append(members, Collaborator{ConnectionID: entry.ConnectionID, ActorID: entry.Principal.ActorID, DisplayName: entry.Principal.DisplayName})
	}
	sort.Slice(members, func(i, j int) bool { return members[i].ConnectionID < members[j].ConnectionID })
	t.hub.Broadcast(EventPresence, PresenceSnapshot{Members: members})
}

func decodeStrict(raw []byte, target any) *ProtocolError {
	if len(raw) == 0 || len(raw) > MaxFrameBytes {
		return NewProtocolError(ErrorInvalidRequest, "message payload is empty or exceeds maximum size")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return NewProtocolError(ErrorInvalidRequest, "malformed collaboration payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return NewProtocolError(ErrorInvalidRequest, "collaboration payload must contain one object")
	}
	return nil
}
