package collab

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"m31labs.dev/gosx-admin/blockstudio"
	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
	"m31labs.dev/gosx/hub"
)

func TestRoomServeHTTPRequiresValidActorResolver(t *testing.T) {
	tests := []struct {
		name     string
		resolver ActorResolver
	}{
		{name: "nil resolver"},
		{name: "resolver error", resolver: func(*http.Request) (admincollab.Actor, error) {
			return admincollab.Actor{}, errors.New("authentication unavailable")
		}},
		{name: "missing actor id", resolver: func(*http.Request) (admincollab.Actor, error) {
			return admincollab.Actor{Kind: admincollab.ActorHuman}, nil
		}},
		{name: "unsupported actor kind", resolver: func(*http.Request) (admincollab.Actor, error) {
			return admincollab.Actor{ID: "trusted", Kind: admincollab.ActorKind("root")}, nil
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			room, err := NewRoom(Options{
				Resource:      Resource{Kind: "page", ID: "home"},
				Document:      testDocument(),
				ActorResolver: test.resolver,
			})
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(room)
			defer server.Close()

			connection, response, err := dialRoomAuth(server.URL)
			if connection != nil {
				_ = connection.Close()
			}
			if err == nil || response == nil || response.StatusCode != http.StatusForbidden {
				t.Fatalf("dial err=%v response=%v, want HTTP 403", err, response)
			}
		})
	}
}

func TestRoomRejectsSuppliedHubBeforeLoadingDraft(t *testing.T) {
	foreign := hub.New("foreign-shared-hub")
	foreign.Latch("studio.snapshot")
	foreign.Broadcast("studio.snapshot", Snapshot{Document: testDocument()})
	store := &roomAuthStoreProbe{}

	for attempt := 0; attempt < 2; attempt++ {
		room, err := NewRoom(Options{
			Resource: Resource{Kind: "page", ID: "home"},
			Document: testDocument(),
			Store:    store,
			Hub:      foreign,
		})
		if room != nil {
			t.Fatalf("attempt %d returned a Room for a supplied Hub", attempt)
		}
		if err == nil || !strings.Contains(err.Error(), "Options.Hub") || !strings.Contains(err.Error(), "private Hub") {
			t.Fatalf("attempt %d error=%v, want actionable supplied-Hub rejection", attempt, err)
		}
	}
	if store.loadCalls != 0 || store.saveCalls != 0 {
		t.Fatalf("supplied-Hub rejection touched Store: loads=%d saves=%d", store.loadCalls, store.saveCalls)
	}
}

func TestRoomRawHubCannotReplayOrReceiveDocuments(t *testing.T) {
	actor := roomAuthActor("trusted-editor", admincollab.ActorHuman, admincollab.CapabilityEdit, admincollab.CapabilityComment)
	room, authenticatedServer := newRoomAuthServer(t, actor, nil)

	if _, err := room.ApplyOperation(actor, roomAuthSetTextOperation("before-wire", "Before raw Hub")); err != nil {
		t.Fatal(err)
	}

	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		room.Hub().ServeHTTP(w, req)
	}))
	defer rawServer.Close()

	passive, _, err := dialRoomAuth(rawServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer passive.Close()
	_ = readRoomAuthEvent(t, passive, "__welcome")
	passiveSnapshots := observeRoomAuthSnapshots(t, passive)
	// The Room no longer creates a global snapshot latch, so a raw Hub route
	// cannot replay the document that was changed before this connection.
	assertNoRoomAuthSnapshotEvent(t, passiveSnapshots, 120*time.Millisecond)

	trusted, _, err := dialRoomAuth(authenticatedServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer trusted.Close()
	_ = readRoomAuthEvent(t, trusted, "__welcome")
	roomAuthWrite(t, trusted, "studio.join", roomAuthJoinMessage{Actor: roomAuthForgedSystemActor(), State: PresenceEditing})
	_ = readRoomAuthEvent(t, trusted, "studio.presence")
	snapshot := readRoomAuthSnapshot(t, trusted)
	if snapshot.Document.Blocks[0].Values["headline"].String != "Before raw Hub" {
		t.Fatalf("authenticated join snapshot=%q, want prior update", snapshot.Document.Blocks[0].Values["headline"].String)
	}
	assertRoomAuthActor(t, snapshot.Presence[0].Actor, actor)

	roomAuthWrite(t, trusted, "studio.operation", roomAuthOperationMessage{
		Actor:     roomAuthForgedSystemActor(),
		Operation: roomAuthSetTextOperation("after-wire", "After trusted wire"),
	})
	_ = readRoomAuthSnapshot(t, trusted)
	assertNoRoomAuthSnapshotEvent(t, passiveSnapshots, 160*time.Millisecond)

	activeRaw, _, err := dialRoomAuth(rawServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer activeRaw.Close()
	_ = readRoomAuthEvent(t, activeRaw, "__welcome")
	roomAuthWrite(t, activeRaw, "studio.join", roomAuthJoinMessage{Actor: roomAuthForgedSystemActor(), State: PresenceEditing})
	assertRoomAuthDisconnectedWithoutSnapshot(t, activeRaw)
	if got := room.Snapshot().Presence; len(got) != 1 || got[0].Actor.ID != actor.ID {
		t.Fatalf("raw Hub join changed presence: %#v", got)
	}
}

func TestRoomInstancesUsePrivateHubsAndIsolateResources(t *testing.T) {
	actor := roomAuthActor("same-actor", admincollab.ActorHuman, admincollab.CapabilityEdit)
	roomA, serverA := newRoomAuthPolicyServerForResource(t, Resource{Kind: "page", ID: "alpha"}, actor, nil, nil)
	roomB, serverB := newRoomAuthPolicyServerForResource(t, Resource{Kind: "page", ID: "beta"}, actor, nil, nil)
	if roomA.Hub() == roomB.Hub() {
		t.Fatal("distinct Room instances unexpectedly share a Hub")
	}

	connectionA, _, err := dialRoomAuth(serverA.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer connectionA.Close()
	_ = readRoomAuthEvent(t, connectionA, "__welcome")
	roomAuthWrite(t, connectionA, "studio.join", roomAuthJoinMessage{Actor: roomAuthForgedSystemActor(), State: PresenceEditing})
	_ = readRoomAuthEvent(t, connectionA, "studio.presence")
	joinedA := readRoomAuthSnapshot(t, connectionA)
	if joinedA.Resource.ID != "alpha" {
		t.Fatalf("Room A resource=%q", joinedA.Resource.ID)
	}

	connectionB, _, err := dialRoomAuth(serverB.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer connectionB.Close()
	welcomeB := readRoomAuthEvent(t, connectionB, "__welcome")
	clientBID := roomAuthWelcomeClientID(t, welcomeB)
	roomAuthWrite(t, connectionB, "studio.join", roomAuthJoinMessage{Actor: roomAuthForgedSystemActor(), State: PresenceEditing})
	_ = readRoomAuthEvent(t, connectionB, "studio.presence")
	joinedB := readRoomAuthSnapshot(t, connectionB)
	if joinedB.Resource.ID != "beta" {
		t.Fatalf("Room B resource=%q", joinedB.Resource.ID)
	}
	roomAuthWrite(t, connectionA, "studio.operation", roomAuthOperationMessage{
		Actor:     roomAuthForgedSystemActor(),
		Operation: roomAuthSetTextOperation("alpha-write", "Only alpha changes"),
	})
	updatedA := readRoomAuthSnapshot(t, connectionA)
	if got := updatedA.Document.Blocks[0].Values["headline"].String; got != "Only alpha changes" {
		t.Fatalf("Room A headline=%q", got)
	}
	roomB.Hub().Send(clientBID, "room-auth.barrier", map[string]bool{"ready": true})
	assertNoRoomAuthSnapshotBeforeBarrier(t, connectionB, "room-auth.barrier")
	if got := roomB.Snapshot().Document.Blocks[0].Values["headline"].String; got != "Welcome" {
		t.Fatalf("Room B was mutated by Room A operation: headline=%q", got)
	}
}

func TestRoomWireUsesAuthenticatedActorForAllActions(t *testing.T) {
	actor := roomAuthActor("trusted-agent", admincollab.ActorAgent, admincollab.CapabilityEdit, admincollab.CapabilitySuggest, admincollab.CapabilityComment)
	_, server := newRoomAuthServer(t, actor, nil)
	connection, _, err := dialRoomAuth(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_ = readRoomAuthEvent(t, connection, "__welcome")
	forged := roomAuthForgedSystemActor()

	roomAuthWrite(t, connection, "studio.join", roomAuthJoinMessage{Actor: forged, State: PresenceEditing})
	presence := decodeRoomAuth[[]Presence](t, readRoomAuthEvent(t, connection, "studio.presence").Data)
	if len(presence) != 1 {
		t.Fatalf("join presence=%#v", presence)
	}
	assertRoomAuthActor(t, presence[0].Actor, actor)
	snapshot := readRoomAuthSnapshot(t, connection)
	assertRoomAuthActor(t, snapshot.Presence[0].Actor, actor)

	roomAuthWrite(t, connection, "studio.presence", roomAuthPresenceMessage{
		Actor: forged, State: PresenceViewing, Selection: admincollab.Target{BlockID: "hero", Field: "headline"},
	})
	presence = decodeRoomAuth[[]Presence](t, readRoomAuthEvent(t, connection, "studio.presence").Data)
	assertRoomAuthActor(t, presence[0].Actor, actor)
	if presence[0].Selection.Field != "headline" || presence[0].State != PresenceViewing {
		t.Fatalf("presence payload=%#v", presence[0])
	}

	roomAuthWrite(t, connection, "studio.operation", roomAuthOperationMessage{
		Actor: forged,
		Operation: admincollab.Operation{
			ID:        "wire-operation",
			ActorID:   "mallory",
			ActorKind: admincollab.ActorSystem,
			Kind:      admincollab.OpSetText,
			Target:    admincollab.Target{BlockID: "hero", Field: "headline"},
			Payload:   admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: "Operation value"}),
		},
	})
	snapshot = readRoomAuthSnapshot(t, connection)
	if got := snapshot.Document.Blocks[0].Values["headline"].String; got != "Operation value" {
		t.Fatalf("operation value=%q", got)
	}
	assertRoomAuthActor(t, snapshot.Reviews[len(snapshot.Reviews)-1].Actor, actor)

	roomAuthWrite(t, connection, "studio.transaction", admincollab.Transaction{
		ID:    "wire-transaction",
		Actor: forged,
		Operations: []admincollab.Operation{{
			ID:        "wire-transaction-operation",
			ActorID:   "mallory",
			ActorKind: admincollab.ActorSystem,
			Kind:      admincollab.OpSetText,
			Target:    admincollab.Target{BlockID: "hero", Field: "headline"},
			Payload:   admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: "Transaction value"}),
		}},
	})
	snapshot = readRoomAuthSnapshot(t, connection)
	if got := snapshot.Document.Blocks[0].Values["headline"].String; got != "Transaction value" {
		t.Fatalf("transaction value=%q", got)
	}
	assertRoomAuthActor(t, snapshot.Reviews[len(snapshot.Reviews)-1].Actor, actor)

	suggestion := admincollab.Operation{
		ID:        "wire-suggestion",
		ActorID:   "mallory",
		ActorKind: admincollab.ActorSystem,
		Kind:      admincollab.OpSuggest,
		Target:    admincollab.Target{BlockID: "hero"},
		Payload: admincollab.Payload(admincollab.SuggestPayload{
			Title: "Trusted suggestion",
			Operations: []admincollab.Operation{{
				ID:        "nested-forged-operation",
				ActorID:   "mallory",
				ActorKind: admincollab.ActorSystem,
				Kind:      admincollab.OpSetText,
				Target:    admincollab.Target{BlockID: "hero", Field: "headline"},
				Payload:   admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: "Accepted suggestion"}),
			}},
		}),
	}
	roomAuthWrite(t, connection, "studio.operation", roomAuthOperationMessage{Actor: forged, Operation: suggestion})
	snapshot = readRoomAuthSnapshot(t, connection)
	if len(snapshot.Suggestions) != 1 {
		t.Fatalf("suggestions=%#v", snapshot.Suggestions)
	}
	assertRoomAuthActorIdentity(t, admincollab.Actor{ID: snapshot.Suggestions[0].ActorID, Kind: snapshot.Suggestions[0].ActorKind}, actor)
	if len(snapshot.Suggestions[0].Operations) != 1 {
		t.Fatalf("suggestion operations=%#v", snapshot.Suggestions[0].Operations)
	}
	if snapshot.Suggestions[0].Operations[0].ActorID != actor.ID || snapshot.Suggestions[0].Operations[0].ActorKind != actor.Kind {
		t.Fatalf("nested operation identity=%#v, want trusted actor", snapshot.Suggestions[0].Operations[0])
	}
	assertRoomAuthActor(t, snapshot.Reviews[len(snapshot.Reviews)-1].Actor, actor)

	roomAuthWrite(t, connection, "studio.acceptSuggestion", roomAuthSuggestionDecisionMessage{Actor: forged, SuggestionID: "wire-suggestion"})
	snapshot = readRoomAuthSnapshot(t, connection)
	if got := snapshot.Document.Blocks[0].Values["headline"].String; got != "Accepted suggestion" {
		t.Fatalf("accepted suggestion value=%q", got)
	}
	if len(snapshot.ProposalDecisions) != 1 {
		t.Fatalf("proposal decisions=%#v", snapshot.ProposalDecisions)
	}
	assertRoomAuthActor(t, snapshot.ProposalDecisions[0].Actor, actor)
	assertRoomAuthActor(t, snapshot.ProposalDecisions[0].Review.Actor, actor)

	secondSuggestion := suggestion
	secondSuggestion.ID = "wire-rejected-suggestion"
	secondSuggestion.Payload = admincollab.Payload(admincollab.SuggestPayload{Title: "Rejected suggestion"})
	roomAuthWrite(t, connection, "studio.transaction", admincollab.Transaction{
		ID: "wire-second-suggestion", Actor: forged, Operations: []admincollab.Operation{secondSuggestion},
	})
	_ = readRoomAuthSnapshot(t, connection)
	roomAuthWrite(t, connection, "studio.rejectSuggestion", roomAuthSuggestionDecisionMessage{Actor: forged, SuggestionID: secondSuggestion.ID, Reason: "Keep current copy"})
	snapshot = readRoomAuthSnapshot(t, connection)
	if len(snapshot.ProposalDecisions) != 2 || snapshot.ProposalDecisions[1].Status != ProposalRejected {
		t.Fatalf("rejected proposal decisions=%#v", snapshot.ProposalDecisions)
	}
	assertRoomAuthActor(t, snapshot.ProposalDecisions[1].Actor, actor)

	roomAuthWrite(t, connection, "studio.comment", roomAuthCommentMessage{
		Actor: forged, Target: admincollab.Target{BlockID: "hero", Field: "headline"}, Body: "Trusted comment",
	})
	snapshot = readRoomAuthSnapshot(t, connection)
	if len(snapshot.Comments) != 1 || snapshot.Comments[0].Body != "Trusted comment" {
		t.Fatalf("comments=%#v", snapshot.Comments)
	}
	assertRoomAuthActorIdentity(t, admincollab.Actor{ID: snapshot.Comments[0].ActorID, Kind: snapshot.Comments[0].ActorKind}, actor)
	assertRoomAuthActor(t, snapshot.Reviews[len(snapshot.Reviews)-1].Actor, actor)
	commentID := snapshot.Comments[0].ID

	roomAuthWrite(t, connection, "studio.resolveComment", roomAuthCommentDecisionMessage{Actor: forged, CommentID: commentID, Reason: "Resolved by trusted actor"})
	snapshot = readRoomAuthSnapshot(t, connection)
	if len(snapshot.CommentDecisions) != 1 || snapshot.CommentDecisions[0].Status != CommentResolved {
		t.Fatalf("resolved comments=%#v", snapshot.CommentDecisions)
	}
	assertRoomAuthActor(t, snapshot.CommentDecisions[0].Actor, actor)

	roomAuthWrite(t, connection, "studio.reopenComment", roomAuthCommentDecisionMessage{Actor: forged, CommentID: commentID, Reason: "Reopened by trusted actor"})
	snapshot = readRoomAuthSnapshot(t, connection)
	if len(snapshot.CommentDecisions) != 2 || snapshot.CommentDecisions[1].Status != CommentOpen {
		t.Fatalf("reopened comments=%#v", snapshot.CommentDecisions)
	}
	assertRoomAuthActor(t, snapshot.CommentDecisions[1].Actor, actor)
}

func TestRoomWireIgnoresForgedAuthorityClaims(t *testing.T) {
	actor := roomAuthActor("trusted-viewer", admincollab.ActorHuman)
	room, server := newRoomAuthServer(t, actor, nil)
	connection, _, err := dialRoomAuth(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	_ = readRoomAuthEvent(t, connection, "__welcome")
	forged := roomAuthForgedSystemActor()
	roomAuthWrite(t, connection, "studio.join", roomAuthJoinMessage{Actor: forged, State: PresenceEditing})
	_ = readRoomAuthEvent(t, connection, "studio.presence")
	_ = readRoomAuthSnapshot(t, connection)

	serverSuggestor := roomAuthActor("server-agent", admincollab.ActorAgent, admincollab.CapabilitySuggest)
	if _, err := room.ApplyOperation(serverSuggestor, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	_ = readRoomAuthSnapshot(t, connection)
	serverCommenter := roomAuthActor("server-commenter", admincollab.ActorHuman, admincollab.CapabilityComment)
	commentSnapshot, err := room.AddComment(serverCommenter, admincollab.Target{BlockID: "hero", Field: "headline"}, "Server-seeded comment")
	if err != nil {
		t.Fatal(err)
	}
	_ = readRoomAuthSnapshot(t, connection)
	commentID := commentSnapshot.Comments[0].ID
	before := room.Snapshot()
	forgedSnapshots := observeRoomAuthSnapshots(t, connection)

	roomAuthWrite(t, connection, "studio.operation", roomAuthOperationMessage{Actor: forged, Operation: roomAuthSetTextOperation("forged-operation", "must not apply")})
	roomAuthWrite(t, connection, "studio.transaction", admincollab.Transaction{ID: "forged-transaction", Actor: forged, Operations: []admincollab.Operation{roomAuthSetTextOperation("forged-transaction-op", "must not apply")}})
	roomAuthWrite(t, connection, "studio.acceptSuggestion", roomAuthSuggestionDecisionMessage{Actor: forged, SuggestionID: "suggestion"})
	roomAuthWrite(t, connection, "studio.rejectSuggestion", roomAuthSuggestionDecisionMessage{Actor: forged, SuggestionID: "suggestion", Reason: "must not reject"})
	roomAuthWrite(t, connection, "studio.comment", roomAuthCommentMessage{Actor: forged, Target: admincollab.Target{BlockID: "hero"}, Body: "must not comment"})
	roomAuthWrite(t, connection, "studio.resolveComment", roomAuthCommentDecisionMessage{Actor: forged, CommentID: commentID, Reason: "must not resolve"})
	roomAuthWrite(t, connection, "studio.reopenComment", roomAuthCommentDecisionMessage{Actor: forged, CommentID: commentID, Reason: "must not reopen"})
	assertNoRoomAuthSnapshotEvent(t, forgedSnapshots, 200*time.Millisecond)

	if after := room.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("forged authority changed Room state:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestRoomLegacyAuthorizeCannotApproveDecisions(t *testing.T) {
	actor := roomAuthActor("authorized-editor", admincollab.ActorHuman, admincollab.CapabilityEdit, admincollab.CapabilitySuggest, admincollab.CapabilityComment)
	// This legacy policy intentionally permits the operation kinds used to
	// create suggestions/comments. Without an explicit decision hook it must
	// not become an implicit grant for state transitions.
	authorize := func(_ admincollab.Actor, op admincollab.Operation) bool {
		return op.Kind == admincollab.OpSuggest || op.Kind == admincollab.OpComment
	}
	room, _ := newRoomAuthServer(t, actor, authorize)
	if _, err := room.ApplyOperation(actor, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	commentSnapshot, err := room.AddComment(actor, admincollab.Target{BlockID: "hero", Field: "headline"}, "A comment")
	if err != nil {
		t.Fatal(err)
	}
	commentID := commentSnapshot.Comments[0].ID
	before := room.Snapshot()

	if _, err := room.AcceptSuggestion(actor, "suggestion"); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("legacy suggestion policy unexpectedly approved accept: %v", err)
	}
	if _, err := room.ResolveComment(actor, commentID, "must be denied"); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("legacy comment policy unexpectedly approved resolve: %v", err)
	}
	if after := room.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("legacy operation policy changed decision state:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestRoomAuthorizeDecisionReceivesExactTargetAndCanDeny(t *testing.T) {
	actor := roomAuthActor("authorized-editor", admincollab.ActorHuman, admincollab.CapabilityEdit, admincollab.CapabilitySuggest, admincollab.CapabilityComment)
	type observedLegacy struct {
		actor     admincollab.Actor
		operation admincollab.Operation
	}
	legacyCalls := make([]observedLegacy, 0, 8)
	authorize := func(got admincollab.Actor, op admincollab.Operation) bool {
		legacyCalls = append(legacyCalls, observedLegacy{actor: got, operation: op})
		if got.Kind == admincollab.ActorSystem {
			return false
		}
		return true
	}
	type observedDecision struct {
		actor    admincollab.Actor
		decision Decision
	}
	decisionCalls := make([]observedDecision, 0, 4)
	authorizeDecision := func(got admincollab.Actor, decision Decision) bool {
		decisionCalls = append(decisionCalls, observedDecision{actor: got, decision: decision})
		return decision.Kind != DecisionRejectSuggestion
	}
	room, _ := newRoomAuthPolicyServer(t, actor, authorize, authorizeDecision)

	if _, err := room.ApplyOperation(actor, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	accepted, err := room.AcceptSuggestion(actor, "suggestion")
	if err != nil {
		t.Fatal(err)
	}
	if len(accepted.ProposalDecisions) != 1 || accepted.ProposalDecisions[0].Status != ProposalAccepted {
		t.Fatalf("accepted suggestion decision=%#v", accepted.ProposalDecisions)
	}

	rejectedSuggestion := warmerLeadSuggestion()
	rejectedSuggestion.ID = "reject-suggestion"
	if _, err := room.ApplyOperation(actor, rejectedSuggestion); err != nil {
		t.Fatal(err)
	}
	beforeReject := room.Snapshot()
	if _, err := room.RejectSuggestion(actor, rejectedSuggestion.ID, "policy denied"); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("explicit decision policy unexpectedly approved reject: %v", err)
	}
	if after := room.Snapshot(); !reflect.DeepEqual(after, beforeReject) {
		t.Fatalf("denied rejection changed Room state:\nbefore=%#v\nafter=%#v", beforeReject, after)
	}

	commentSnapshot, err := room.AddComment(actor, admincollab.Target{BlockID: "hero", Field: "headline"}, "A comment")
	if err != nil {
		t.Fatal(err)
	}
	commentID := commentSnapshot.Comments[0].ID
	if _, err := room.ResolveComment(actor, commentID, "resolved"); err != nil {
		t.Fatal(err)
	}
	if _, err := room.ReopenComment(actor, commentID, "reopened"); err != nil {
		t.Fatal(err)
	}

	wantDecisions := []Decision{
		{Kind: DecisionAcceptSuggestion, TargetID: "suggestion"},
		{Kind: DecisionRejectSuggestion, TargetID: "reject-suggestion"},
		{Kind: DecisionResolveComment, TargetID: commentID},
		{Kind: DecisionReopenComment, TargetID: commentID},
	}
	if len(decisionCalls) != len(wantDecisions) {
		t.Fatalf("decision callback calls=%#v, want %d calls", decisionCalls, len(wantDecisions))
	}
	for index, call := range decisionCalls {
		assertRoomAuthActor(t, call.actor, actor)
		if call.decision != wantDecisions[index] {
			t.Errorf("decision call %d=%#v, want %#v", index, call.decision, wantDecisions[index])
		}
	}
	var legacyDecisionCalls []observedLegacy
	for _, call := range legacyCalls {
		assertRoomAuthActor(t, call.actor, actor)
		if call.operation.ID == "" {
			legacyDecisionCalls = append(legacyDecisionCalls, call)
		}
	}
	if len(legacyDecisionCalls) != len(wantDecisions) {
		t.Fatalf("legacy veto decision calls=%#v, want %d calls", legacyDecisionCalls, len(wantDecisions))
	}
	for index, call := range legacyDecisionCalls {
		wantKind := admincollab.OpSuggest
		if wantDecisions[index].Kind == DecisionResolveComment || wantDecisions[index].Kind == DecisionReopenComment {
			wantKind = admincollab.OpComment
		}
		if call.operation.Kind != wantKind || call.operation.Target.BlockID != wantDecisions[index].TargetID {
			t.Errorf("legacy veto call %d=%#v, want kind=%q target=%q", index, call.operation, wantKind, wantDecisions[index].TargetID)
		}
	}
}

func TestRoomAuthorizeDecisionLegacyVetoCoversSystemActor(t *testing.T) {
	seedActor := roomAuthActor("seed-editor", admincollab.ActorHuman, admincollab.CapabilitySuggest)
	systemActor := roomAuthActor("trusted-system", admincollab.ActorSystem)
	authorize := func(got admincollab.Actor, _ admincollab.Operation) bool {
		return got.Kind != admincollab.ActorSystem
	}
	decisionCalls := 0
	authorizeDecision := func(_ admincollab.Actor, _ Decision) bool {
		decisionCalls++
		return true
	}
	room, _ := newRoomAuthPolicyServer(t, seedActor, authorize, authorizeDecision)
	if _, err := room.ApplyOperation(seedActor, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	before := room.Snapshot()
	if _, err := room.AcceptSuggestion(systemActor, "suggestion"); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("legacy veto unexpectedly allowed system decision: %v", err)
	}
	if decisionCalls != 0 {
		t.Fatalf("explicit decision callback ran after legacy veto: %d calls", decisionCalls)
	}
	if after := room.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("system veto changed Room state:\nbefore=%#v\nafter=%#v", before, after)
	}
}

type roomAuthJoinMessage struct {
	Actor admincollab.Actor `json:"actor"`
	State PresenceState     `json:"state,omitempty"`
}

type roomAuthPresenceMessage struct {
	Actor     admincollab.Actor  `json:"actor"`
	State     PresenceState      `json:"state,omitempty"`
	Selection admincollab.Target `json:"selection,omitempty"`
}

type roomAuthOperationMessage struct {
	Actor     admincollab.Actor     `json:"actor"`
	Operation admincollab.Operation `json:"operation"`
}

type roomAuthSuggestionDecisionMessage struct {
	Actor        admincollab.Actor `json:"actor"`
	SuggestionID string            `json:"suggestionId"`
	Reason       string            `json:"reason,omitempty"`
}

type roomAuthCommentMessage struct {
	Actor  admincollab.Actor  `json:"actor"`
	Target admincollab.Target `json:"target,omitempty"`
	Body   string             `json:"body"`
}

type roomAuthCommentDecisionMessage struct {
	Actor     admincollab.Actor `json:"actor"`
	CommentID string            `json:"commentId"`
	Reason    string            `json:"reason,omitempty"`
}

func newRoomAuthServer(t *testing.T, actor admincollab.Actor, authorize AuthorizeFunc) (*Room, *httptest.Server) {
	return newRoomAuthPolicyServerForResource(t, Resource{Kind: "page", ID: "home"}, actor, authorize, nil)
}

func newRoomAuthPolicyServer(t *testing.T, actor admincollab.Actor, authorize AuthorizeFunc, authorizeDecision AuthorizeDecisionFunc) (*Room, *httptest.Server) {
	return newRoomAuthPolicyServerForResource(t, Resource{Kind: "page", ID: "home"}, actor, authorize, authorizeDecision)
}

func newRoomAuthPolicyServerForResource(t *testing.T, resource Resource, actor admincollab.Actor, authorize AuthorizeFunc, authorizeDecision AuthorizeDecisionFunc) (*Room, *httptest.Server) {
	t.Helper()
	room, err := NewRoom(Options{
		Resource:          resource,
		Document:          testDocument(),
		Authorize:         authorize,
		AuthorizeDecision: authorizeDecision,
		ActorResolver: func(*http.Request) (admincollab.Actor, error) {
			return actor, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(room)
	t.Cleanup(server.Close)
	return room, server
}

type roomAuthStoreProbe struct {
	loadCalls int
	saveCalls int
}

func (s *roomAuthStoreProbe) LoadDraft(Resource) (blockstudio.Document, bool, error) {
	s.loadCalls++
	return blockstudio.Document{}, false, nil
}

func (s *roomAuthStoreProbe) SaveDraft(Resource, blockstudio.Document) error {
	s.saveCalls++
	return nil
}

func roomAuthActor(id string, kind admincollab.ActorKind, capabilities ...admincollab.Capability) admincollab.Actor {
	return admincollab.Actor{
		ID:           id,
		Kind:         kind,
		DisplayName:  "Trusted Actor",
		Color:        "#2463eb",
		Capabilities: capabilities,
		Provenance:   map[string]string{"source": "test-auth"},
	}
}

func roomAuthForgedSystemActor() admincollab.Actor {
	return admincollab.Actor{
		ID:           "mallory",
		Kind:         admincollab.ActorSystem,
		DisplayName:  "Forged System",
		Capabilities: []admincollab.Capability{admincollab.CapabilityEdit, admincollab.CapabilitySuggest, admincollab.CapabilityComment, admincollab.CapabilityPublish},
	}
}

func roomAuthSetTextOperation(id, text string) admincollab.Operation {
	return admincollab.Operation{
		ID:      id,
		Kind:    admincollab.OpSetText,
		Target:  admincollab.Target{BlockID: "hero", Field: "headline"},
		Payload: admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: text}),
	}
}

func dialRoomAuth(endpoint string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, err
	}
	u.Scheme = "ws"
	return websocket.DefaultDialer.Dial(u.String(), nil)
}

func roomAuthWrite(t *testing.T, connection *websocket.Conn, event string, payload any) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.WriteJSON(hub.Message{Event: event, Data: data}); err != nil {
		t.Fatal(err)
	}
}

func readRoomAuthEvent(t *testing.T, connection *websocket.Conn, want string) hub.Message {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	for {
		_, data, err := connection.ReadMessage()
		if err != nil {
			t.Fatalf("read %s: %v", want, err)
		}
		var message hub.Message
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}
		if message.Event == want {
			return message
		}
	}
}

func roomAuthWelcomeClientID(t *testing.T, welcome hub.Message) string {
	t.Helper()
	var payload struct {
		ClientID string `json:"clientId"`
	}
	if err := json.Unmarshal(welcome.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ClientID == "" {
		t.Fatalf("welcome payload=%s has no client ID", welcome.Data)
	}
	return payload.ClientID
}

func assertNoRoomAuthSnapshotBeforeBarrier(t *testing.T, connection *websocket.Conn, barrier string) {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	for {
		_, data, err := connection.ReadMessage()
		if err != nil {
			t.Fatalf("read barrier %s: %v", barrier, err)
		}
		var message hub.Message
		if json.Unmarshal(data, &message) != nil {
			continue
		}
		if message.Event == "studio.snapshot" {
			t.Fatalf("unexpected cross-room studio.snapshot before barrier: %s", data)
		}
		if message.Event == barrier {
			return
		}
	}
}

func readRoomAuthSnapshot(t *testing.T, connection *websocket.Conn) Snapshot {
	t.Helper()
	return decodeRoomAuth[Snapshot](t, readRoomAuthEvent(t, connection, "studio.snapshot").Data)
}

func observeRoomAuthSnapshots(t *testing.T, connection *websocket.Conn) <-chan Snapshot {
	t.Helper()
	snapshots := make(chan Snapshot, 8)
	go func() {
		defer close(snapshots)
		if err := connection.SetReadDeadline(time.Time{}); err != nil {
			return
		}
		for {
			_, data, err := connection.ReadMessage()
			if err != nil {
				return
			}
			var message hub.Message
			if json.Unmarshal(data, &message) != nil || message.Event != "studio.snapshot" {
				continue
			}
			var snapshot Snapshot
			if json.Unmarshal(message.Data, &snapshot) != nil {
				continue
			}
			select {
			case snapshots <- snapshot:
			default:
			}
		}
	}()
	return snapshots
}

func assertNoRoomAuthSnapshotEvent(t *testing.T, snapshots <-chan Snapshot, timeout time.Duration) {
	t.Helper()
	select {
	case snapshot, ok := <-snapshots:
		if !ok {
			t.Fatal("snapshot observer connection closed unexpectedly")
		}
		t.Fatalf("unexpected studio.snapshot: %#v", snapshot)
	case <-time.After(timeout):
	}
}

func decodeRoomAuth[T any](t *testing.T, data json.RawMessage) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func assertRoomAuthDisconnectedWithoutSnapshot(t *testing.T, connection *websocket.Conn) {
	t.Helper()
	if err := connection.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	for {
		_, data, err := connection.ReadMessage()
		if err != nil {
			return
		}
		var message hub.Message
		if json.Unmarshal(data, &message) == nil && message.Event == "studio.snapshot" {
			t.Fatalf("raw Hub received snapshot before disconnect: %s", data)
		}
	}
}

func assertRoomAuthActor(t *testing.T, got, want admincollab.Actor) {
	t.Helper()
	got = admincollab.NormalizeActor(got)
	want = admincollab.NormalizeActor(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("actor=%#v, want trusted %#v", got, want)
	}
}

func assertRoomAuthActorIdentity(t *testing.T, got, want admincollab.Actor) {
	t.Helper()
	got = admincollab.NormalizeActor(got)
	want = admincollab.NormalizeActor(want)
	if got.ID != want.ID || got.Kind != want.Kind {
		t.Fatalf("actor identity=%#v, want trusted identity=%#v", got, want)
	}
}
