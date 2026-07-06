package collab

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
)

func TestRenderProposalPanelWithPendingActions(t *testing.T) {
	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	agent := admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, Capabilities: []admincollab.Capability{admincollab.CapabilitySuggest}}
	snapshot, err := room.ApplyOperation(agent, warmerLeadSuggestion())
	if err != nil {
		t.Fatal(err)
	}
	html := gosx.RenderHTML(RenderProposalPanel(snapshot, RenderProposalOptions{}))
	for _, want := range []string{
		`data-studio-proposals="true"`,
		`data-studio-proposal="suggestion"`,
		`data-studio-proposal-status="pending"`,
		`data-studio-proposal-action="accept"`,
		`data-studio-proposal-event="studio.acceptSuggestion"`,
		`data-studio-proposal-action="reject"`,
		`Try a warmer lead`,
		`1 pending`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in proposal panel html: %s", want, html)
		}
	}
}

func TestRenderProposalPanelWithDecisionAndEmptyState(t *testing.T) {
	empty := gosx.RenderHTML(RenderProposalPanel(Snapshot{}, RenderProposalOptions{Class: "studio-review", EmptyTitle: "All clear"}))
	if !strings.Contains(empty, `class="studio-review"`) || !strings.Contains(empty, "All clear") {
		t.Fatalf("unexpected empty proposal html: %s", empty)
	}

	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	agent := admincollab.Actor{ID: "cedar", Kind: admincollab.ActorAgent, Capabilities: []admincollab.Capability{admincollab.CapabilitySuggest}}
	if _, err := room.ApplyOperation(agent, warmerLeadSuggestion()); err != nil {
		t.Fatal(err)
	}
	owner := admincollab.Actor{ID: "owner", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityEdit}}
	snapshot, err := room.RejectSuggestion(owner, "suggestion", "Keep the original.")
	if err != nil {
		t.Fatal(err)
	}
	html := gosx.RenderHTML(RenderProposalPanel(snapshot, RenderProposalOptions{}))
	if !strings.Contains(html, `data-studio-proposal-status="rejected"`) || !strings.Contains(html, "Keep the original.") || strings.Contains(html, `data-studio-proposal-action="accept"`) {
		t.Fatalf("unexpected rejected proposal html: %s", html)
	}
}

func TestRenderCommentPanelWithActionsAndEmptyState(t *testing.T) {
	empty := gosx.RenderHTML(RenderCommentPanel(Snapshot{}, RenderCommentOptions{Class: "studio-comments", EmptyTitle: "No notes"}))
	if !strings.Contains(empty, `data-studio-comments="true"`) || !strings.Contains(empty, "No notes") {
		t.Fatalf("unexpected empty comment html: %s", empty)
	}

	room, err := NewRoom(Options{Resource: Resource{Kind: "page", ID: "home"}, Document: testDocument()})
	if err != nil {
		t.Fatal(err)
	}
	reviewer := admincollab.Actor{ID: "reviewer", Kind: admincollab.ActorHuman, Capabilities: []admincollab.Capability{admincollab.CapabilityComment}}
	snapshot, err := room.AddComment(reviewer, admincollab.Target{BlockID: "hero", Field: "headline"}, "Can we make this warmer?")
	if err != nil {
		t.Fatal(err)
	}
	html := gosx.RenderHTML(RenderCommentPanel(snapshot, RenderCommentOptions{}))
	for _, want := range []string{
		`data-studio-comment=`,
		`data-studio-comment-status="open"`,
		`data-studio-comment-action="resolve"`,
		`data-studio-comment-event="studio.resolveComment"`,
		`hero / headline`,
		`1 open`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in comment panel html: %s", want, html)
		}
	}
	snapshot, err = room.ResolveComment(reviewer, snapshot.Comments[0].ID, "Handled.")
	if err != nil {
		t.Fatal(err)
	}
	html = gosx.RenderHTML(RenderCommentPanel(snapshot, RenderCommentOptions{}))
	if !strings.Contains(html, `data-studio-comment-status="resolved"`) || !strings.Contains(html, `data-studio-comment-action="reopen"`) || strings.Contains(html, `data-studio-comment-action="resolve"`) {
		t.Fatalf("unexpected resolved comment html: %s", html)
	}
}
