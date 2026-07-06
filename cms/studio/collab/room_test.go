package collab

import (
	"strings"
	"testing"

	"m31labs.dev/gosx-admin/blockstudio"
	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
)

func TestRoomAppliesOperationAndPersistsDraft(t *testing.T) {
	store := NewMemoryStore()
	resource := Resource{Kind: "page", ID: "home"}
	room, err := NewRoom(Options{
		Resource: resource,
		Document: testDocument(),
		Store:    store,
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := admincollab.Actor{ID: "owner", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityEdit}}
	snapshot, err := room.ApplyOperation(actor, admincollab.Operation{
		ID:      "headline",
		Clock:   "01",
		Kind:    admincollab.OpSetText,
		Target:  admincollab.Target{BlockID: "hero", Field: "headline"},
		Payload: admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: "Forest mornings"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	value := snapshot.Document.Blocks[0].Values["headline"]
	if value.String != "Forest mornings" {
		t.Fatalf("unexpected headline: %#v", value)
	}
	saved, ok, err := store.LoadDraft(resource)
	if err != nil || !ok {
		t.Fatalf("expected saved draft, ok=%v err=%v", ok, err)
	}
	if saved.Blocks[0].Values["headline"].String != "Forest mornings" {
		t.Fatalf("draft was not persisted: %#v", saved.Blocks[0].Values["headline"])
	}
	if len(snapshot.Reviews) != 1 || snapshot.Reviews[0].Summary != "owner prepared 1 operation." {
		t.Fatalf("expected transaction review in snapshot: %#v", snapshot.Reviews)
	}
}

func TestAgentDefaultsToSuggestions(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	agent := admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, Capabilities: []admincollab.Capability{admincollab.CapabilitySuggest, admincollab.CapabilityComment}}
	_, err = room.ApplyOperation(agent, admincollab.Operation{
		ID:      "direct-edit",
		Clock:   "01",
		Kind:    admincollab.OpSetText,
		Target:  admincollab.Target{BlockID: "hero", Field: "headline"},
		Payload: admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: "Agent direct edit"}),
	})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected agent direct edit to be rejected, got %v", err)
	}
	snapshot, err := room.ApplyOperation(agent, admincollab.Operation{
		ID:        "suggestion",
		Clock:     "02",
		Kind:      admincollab.OpSuggest,
		ActorKind: admincollab.ActorAgent,
		Target:    admincollab.Target{BlockID: "hero"},
		Payload: admincollab.Payload(admincollab.SuggestPayload{
			Title: "Try a warmer lead",
			Operations: []admincollab.Operation{{
				ID:      "agent-title",
				Clock:   "02.1",
				Kind:    admincollab.OpSetText,
				Target:  admincollab.Target{BlockID: "hero", Field: "headline"},
				Payload: admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: "A gentler morning"}),
			}},
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Suggestions) != 1 || snapshot.Suggestions[0].Title != "Try a warmer lead" {
		t.Fatalf("unexpected suggestions: %#v", snapshot.Suggestions)
	}
	if len(snapshot.Reviews) != 1 || !snapshot.Reviews[0].RequiresReview || snapshot.Reviews[0].Items[0].Summary != "Review suggestion: Try a warmer lead" {
		t.Fatalf("unexpected review history: %#v", snapshot.Reviews)
	}
	if snapshot.Document.Blocks[0].Values["headline"].String != "Welcome" {
		t.Fatalf("suggestion should not mutate document: %#v", snapshot.Document.Blocks[0].Values["headline"])
	}
}

func TestAcceptSuggestionAppliesProposalAndRecordsDecision(t *testing.T) {
	store := NewMemoryStore()
	resource := Resource{Kind: "page", ID: "home"}
	room, err := NewRoom(Options{Resource: resource, Document: testDocument(), Store: store})
	if err != nil {
		t.Fatal(err)
	}
	agent := admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, Capabilities: []admincollab.Capability{admincollab.CapabilitySuggest}}
	if _, err := room.ApplyOperation(agent, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	owner := admincollab.Actor{ID: "owner", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityEdit}}
	snapshot, err := room.AcceptSuggestion(owner, "suggestion")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Document.Blocks[0].Values["headline"].String != "A gentler morning" {
		t.Fatalf("accepted suggestion did not update draft: %#v", snapshot.Document.Blocks[0].Values["headline"])
	}
	if len(snapshot.ProposalDecisions) != 1 || snapshot.ProposalDecisions[0].Status != ProposalAccepted || snapshot.ProposalDecisions[0].Actor.ID != "owner" {
		t.Fatalf("unexpected proposal decision: %#v", snapshot.ProposalDecisions)
	}
	if snapshot.ProposalDecisions[0].Review.Summary != "owner prepared 1 operation." {
		t.Fatalf("expected accept review summary, got %#v", snapshot.ProposalDecisions[0].Review)
	}
	saved, ok, err := store.LoadDraft(resource)
	if err != nil || !ok {
		t.Fatalf("expected accepted draft to persist, ok=%v err=%v", ok, err)
	}
	if saved.Blocks[0].Values["headline"].String != "A gentler morning" {
		t.Fatalf("accepted draft was not persisted: %#v", saved.Blocks[0].Values["headline"])
	}
	if _, err := room.AcceptSuggestion(owner, "suggestion"); err == nil || !strings.Contains(err.Error(), "not pending") {
		t.Fatalf("expected accepted suggestion not to be accepted twice, got %v", err)
	}
}

func TestRejectSuggestionKeepsDraftUnchanged(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	agent := admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, Capabilities: []admincollab.Capability{admincollab.CapabilitySuggest}}
	if _, err := room.ApplyOperation(agent, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	viewer := admincollab.Actor{ID: "viewer", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityComment}}
	if _, err := room.RejectSuggestion(viewer, "suggestion", "No"); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected viewer rejection to be denied, got %v", err)
	}
	owner := admincollab.Actor{ID: "owner", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityEdit}}
	snapshot, err := room.RejectSuggestion(owner, "suggestion", "Keep the original lead.")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Document.Blocks[0].Values["headline"].String != "Welcome" {
		t.Fatalf("rejected suggestion should not mutate draft: %#v", snapshot.Document.Blocks[0].Values["headline"])
	}
	if len(snapshot.ProposalDecisions) != 1 || snapshot.ProposalDecisions[0].Status != ProposalRejected || snapshot.ProposalDecisions[0].Reason != "Keep the original lead." {
		t.Fatalf("unexpected rejection decision: %#v", snapshot.ProposalDecisions)
	}
	if _, err := room.AcceptSuggestion(owner, "suggestion"); err == nil || !strings.Contains(err.Error(), "not pending") {
		t.Fatalf("expected rejected suggestion not to be accepted, got %v", err)
	}
}

func TestRoomAddsResolvesAndReopensComments(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	viewer := admincollab.Actor{ID: "reviewer", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityComment}}
	snapshot, err := room.AddComment(viewer, admincollab.Target{BlockID: "hero", Field: "headline"}, "Can we make this warmer?")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Comments) != 1 || snapshot.Comments[0].Body != "Can we make this warmer?" || snapshot.Comments[0].Target.Field != "headline" {
		t.Fatalf("unexpected comment snapshot: %#v", snapshot.Comments)
	}
	commentID := snapshot.Comments[0].ID
	snapshot, err = room.ResolveComment(viewer, commentID, "Handled in copy pass.")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.CommentDecisions) != 1 || snapshot.CommentDecisions[0].Status != CommentResolved || snapshot.CommentDecisions[0].Reason != "Handled in copy pass." {
		t.Fatalf("unexpected comment decision: %#v", snapshot.CommentDecisions)
	}
	if _, err := room.ResolveComment(viewer, commentID, "again"); err == nil || !strings.Contains(err.Error(), "already resolved") {
		t.Fatalf("expected duplicate resolve to be rejected, got %v", err)
	}
	snapshot, err = room.ReopenComment(viewer, commentID, "Need another look.")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.CommentDecisions) != 2 || snapshot.CommentDecisions[1].Status != CommentOpen {
		t.Fatalf("unexpected reopen decision: %#v", snapshot.CommentDecisions)
	}
	readonly := admincollab.Actor{ID: "visitor", Kind: admincollab.ActorHuman}
	if _, err := room.ReopenComment(readonly, commentID, "No"); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("expected readonly comment status update to be denied, got %v", err)
	}
}

func TestPresenceIncludesHumanAndAgent(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	room.Join(admincollab.Actor{ID: "owner", Kind: admincollab.ActorHuman, DisplayName: "Owner"}, PresenceEditing)
	room.SetPresence(admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, DisplayName: "Cedar"}, PresenceRunning, admincollab.Target{BlockID: "hero"})
	presence := room.Snapshot().Presence
	if len(presence) != 2 || presence[0].Actor.ID != "cedar" || presence[1].Actor.ID != "owner" {
		t.Fatalf("unexpected sorted presence: %#v", presence)
	}
	if presence[0].State != PresenceRunning || presence[0].Selection.BlockID != "hero" {
		t.Fatalf("unexpected agent presence: %#v", presence[0])
	}
}

func TestRoomLoadsExistingDraft(t *testing.T) {
	store := NewMemoryStore()
	resource := Resource{Kind: "page", ID: "home"}
	doc := testDocument()
	doc.Blocks[0].Values["headline"] = blockstudio.Value{Kind: blockstudio.FieldText, String: "Saved draft"}
	if err := store.SaveDraft(resource, doc); err != nil {
		t.Fatal(err)
	}
	room, err := NewRoom(Options{Resource: resource, Store: store, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	if room.Snapshot().Document.Blocks[0].Values["headline"].String != "Saved draft" {
		t.Fatalf("room did not load saved draft: %#v", room.Snapshot().Document.Blocks[0].Values["headline"])
	}
}

func warmerLeadSuggestion() admincollab.Operation {
	return admincollab.Operation{
		ID:        "suggestion",
		Clock:     "02",
		Kind:      admincollab.OpSuggest,
		ActorKind: admincollab.ActorAgent,
		Target:    admincollab.Target{BlockID: "hero"},
		Payload: admincollab.Payload(admincollab.SuggestPayload{
			Title: "Try a warmer lead",
			Operations: []admincollab.Operation{{
				ID:      "agent-title",
				Clock:   "02.1",
				Kind:    admincollab.OpSetText,
				Target:  admincollab.Target{BlockID: "hero", Field: "headline"},
				Payload: admincollab.Payload(admincollab.SetTextPayload{BlockID: "hero", Field: "headline", Text: "A gentler morning"}),
			}},
		}),
	}
}

func testDocument() blockstudio.Document {
	return blockstudio.Document{
		Version: 1,
		Kind:    "page",
		Blocks: []blockstudio.BlockInstance{{
			ID:      "hero",
			Key:     "hero",
			Enabled: true,
			Order:   1,
			Values: blockstudio.Values{
				"headline": {Kind: blockstudio.FieldText, String: "Welcome"},
			},
		}},
	}
}
