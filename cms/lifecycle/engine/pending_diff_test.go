package engine

import (
	"context"
	"testing"

	"m31labs.dev/gosx-studio/authoring"
)

// TestPendingDiffShowsOpsSincePublishAndHidesPublishedOnes exercises the
// engine-level wiring end to end (not just the pure OperationChangeSet
// function): ops recorded before a real Engine.Publish must not show in a
// later PendingDiff; ops recorded after that same publish must show.
func TestPendingDiffShowsOpsSincePublishAndHidesPublishedOnes(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")

	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	host.addOp(ref, fieldOp("op-before-1", 1, "actor-1", "site.title", authoring.PresentValue("Original"), authoring.PresentValue("Draft V1")))

	if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Nothing pending yet: the only recorded op predates (or is exactly at)
	// the publish cursor the fake host just advanced to.
	preview, err := e.PendingDiff(ctx, ref)
	if err != nil {
		t.Fatalf("PendingDiff (post-publish, no new ops): %v", err)
	}
	if preview.HasChanges || len(preview.Changes) != 0 {
		t.Fatalf("expected no pending changes immediately after publish, got %#v", preview)
	}

	// A new op recorded after the publish must show.
	host.addOp(ref, fieldOp("op-after-1", 2, "actor-2", "site.subtitle", authoring.PresentValue("Old sub"), authoring.PresentValue("New sub")))

	preview, err = e.PendingDiff(ctx, ref)
	if err != nil {
		t.Fatalf("PendingDiff (post-publish, one new op): %v", err)
	}
	if !preview.HasChanges || len(preview.Changes) != 1 {
		t.Fatalf("expected exactly one pending change, got %#v", preview)
	}
	if preview.Changes[0].OpIDs[0] != "op-after-1" {
		t.Fatalf("expected the post-publish op to be the pending change, got %#v", preview.Changes[0])
	}
	if preview.Changes[0].ActorID != "actor-2" {
		t.Fatalf("expected actor attribution preserved through the engine wiring, got %#v", preview.Changes[0])
	}
}

// TestPendingDiffReportsNoChangesForCleanDraft covers the "empty diff"
// case at the engine level: HasDiff/HasChanges are both false when nothing
// is pending, and PendingDiff never errors just because there is nothing to
// show.
func TestPendingDiffReportsNoChangesForCleanDraft(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	host.setLive(ref, map[string]string{"title": "Original"})
	// No draft set, no ops recorded.

	preview, err := e.PendingDiff(ctx, ref)
	if err != nil {
		t.Fatalf("PendingDiff: %v", err)
	}
	if preview.HasDiff {
		t.Fatalf("expected HasDiff=false for a clean draft, got %#v", preview)
	}
	if preview.HasChanges || len(preview.Changes) != 0 {
		t.Fatalf("expected HasChanges=false and an empty change set for a clean draft, got %#v", preview)
	}
}

// TestPublishSucceedsWithEmptyChangeSet documents and locks in this slice's
// conservative choice on the spec §3 open question "should publishing an
// empty diff be blocked or allowed": ALLOWED, matching Engine.Publish's
// existing (S1/S2/S4) normative sequence, which has no gate on an empty
// diff/change-set today. This slice deliberately does not add one (see
// change_set.go's "Empty-diff publish behavior" doc comment) -- Publish must
// keep succeeding even when PendingDiff would report HasChanges=false.
func TestPublishSucceedsWithEmptyChangeSet(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")
	host.setLive(ref, map[string]string{"title": "Original"})
	// No draft, no ops: PendingDiff would report nothing to publish.

	preview, err := e.PendingDiff(ctx, ref)
	if err != nil {
		t.Fatalf("PendingDiff: %v", err)
	}
	if preview.HasChanges {
		t.Fatalf("expected no pending changes as the empty-diff precondition, got %#v", preview)
	}

	result, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-empty"})
	if err != nil {
		t.Fatalf("expected Publish to succeed on an empty change set (documented choice: publishing nothing is allowed), got error: %v", err)
	}
	if result.RevisionID == "" {
		t.Fatalf("expected a real published revision even for an empty change set, got %#v", result)
	}
	if host.applyPublishCalls != 1 {
		t.Fatalf("expected Publish to still call the host exactly once, got %d", host.applyPublishCalls)
	}
}
