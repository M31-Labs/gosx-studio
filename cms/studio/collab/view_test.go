package collab

import (
	"testing"

	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
)

func TestSnapshotViewExposesProposalActionsAndCounts(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	agent := admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, Capabilities: []admincollab.Capability{admincollab.CapabilitySuggest}}
	snapshot, err := room.ApplyOperation(agent, warmerLeadSuggestion())
	if err != nil {
		t.Fatal(err)
	}
	view := SnapshotView(snapshot)
	if view["proposalCount"] != 1 || view["pendingCount"] != 1 || view["acceptedCount"] != 0 || view["rejectedCount"] != 0 {
		t.Fatalf("unexpected snapshot counts: %#v", view)
	}
	proposals := view["proposals"].([]map[string]any)
	if len(proposals) != 1 || proposals[0]["status"] != "pending" || proposals[0]["canAccept"] != true || proposals[0]["acceptEvent"] != "studio.acceptSuggestion" {
		t.Fatalf("unexpected proposal view: %#v", proposals)
	}
	items := proposals[0]["items"].([]map[string]any)
	if len(items) != 1 || items[0]["summary"] != "Review suggestion: Try a warmer lead" {
		t.Fatalf("unexpected proposal review items: %#v", items)
	}
}

func TestProposalViewsReflectAcceptedAndRejectedDecisions(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	agent := admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, Capabilities: []admincollab.Capability{admincollab.CapabilitySuggest}}
	if _, err := room.ApplyOperation(agent, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	owner := admincollab.Actor{ID: "owner", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityEdit}}
	accepted, err := room.AcceptSuggestion(owner, "suggestion")
	if err != nil {
		t.Fatal(err)
	}
	views := ProposalViews(accepted)
	if len(views) != 1 || views[0].Status != ProposalAccepted || views[0].CanAccept || views[0].DecisionActorID != "owner" {
		t.Fatalf("unexpected accepted proposal view: %#v", views)
	}
	if views[0].ReviewSummary != "owner prepared 1 operation." || len(views[0].Items) != 1 || views[0].Items[0].Summary != "Edit headline on hero block" {
		t.Fatalf("unexpected accepted review details: %#v", views[0])
	}

	room, err = NewRoom(Options{Resource: Resource{Kind: "page", ID: "home-2"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := room.ApplyOperation(agent, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	rejected, err := room.RejectSuggestion(owner, "suggestion", "Not this route.")
	if err != nil {
		t.Fatal(err)
	}
	views = ProposalViews(rejected)
	if len(views) != 1 || views[0].Status != ProposalRejected || views[0].CanReject || views[0].DecisionReason != "Not this route." {
		t.Fatalf("unexpected rejected proposal view: %#v", views)
	}
}

func TestCommentViewsExposeStatusActionsAndCounts(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	reviewer := admincollab.Actor{ID: "reviewer", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityComment}}
	snapshot, err := room.AddComment(reviewer, admincollab.Target{BlockID: "hero", Field: "headline"}, "Can we make this warmer?")
	if err != nil {
		t.Fatal(err)
	}
	views := CommentViews(snapshot)
	if len(views) != 1 || views[0].Status != CommentOpen || !views[0].CanResolve || views[0].CanReopen || views[0].TargetLabel != "hero / headline" {
		t.Fatalf("unexpected open comment view: %#v", views)
	}
	view := SnapshotView(snapshot)
	if view["commentCount"] != 1 || view["openCommentCount"] != 1 || view["resolvedCommentCount"] != 0 {
		t.Fatalf("unexpected comment counts: %#v", view)
	}
	comments := view["comments"].([]map[string]any)
	if comments[0]["resolveEvent"] != "studio.resolveComment" || comments[0]["statusLabel"] != "Open" {
		t.Fatalf("unexpected comment view map: %#v", comments)
	}
	snapshot, err = room.ResolveComment(reviewer, views[0].ID, "Done.")
	if err != nil {
		t.Fatal(err)
	}
	views = CommentViews(snapshot)
	if len(views) != 1 || views[0].Status != CommentResolved || views[0].CanResolve || !views[0].CanReopen || views[0].DecisionReason != "Done." {
		t.Fatalf("unexpected resolved comment view: %#v", views)
	}
}
