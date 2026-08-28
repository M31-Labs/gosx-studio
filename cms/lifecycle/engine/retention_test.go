package engine

import (
	"context"
	"testing"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// TestRetentionTrimsOldestRestorePointsAtConfiguredCap is the S2 boundary
// test: with a small configured RetentionPolicy, publishing repeatedly mints
// one restore point per publish, and once the count exceeds the cap the
// oldest restore points are trimmed down to exactly the keep window --
// never the newest ones.
func TestRetentionTrimsOldestRestorePointsAtConfiguredCap(t *testing.T) {
	e, host := newTestEngine(t)
	e.Retain = RetentionPolicy{MaxRevisions: 3, MinRestorePointKeep: 1}
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")

	host.setLive(ref, map[string]string{"title": "v0"})
	var restorePointIDs []string
	for i := 1; i <= 5; i++ {
		host.setDraft(ref, map[string]string{"title": versionLabel(i)})
		result, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: versionLabel(i)})
		if err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
		restorePointIDs = append(restorePointIDs, result.RestorePointID)
	}

	points := restorePointRevisions(t, e, ref)
	if len(points) != 3 {
		t.Fatalf("expected retention to trim down to the configured cap of 3, got %d", len(points))
	}

	remaining := map[string]bool{}
	for _, point := range points {
		remaining[point.ID] = true
	}
	// The 5 restore points were minted in order restorePointIDs[0..4]
	// (oldest to newest); the cap of 3 must keep exactly the newest 3.
	for i, id := range restorePointIDs {
		wantKept := i >= len(restorePointIDs)-3
		if remaining[id] != wantKept {
			t.Fatalf("restore point %d (%q): expected kept=%v, got kept=%v", i, id, wantKept, remaining[id])
		}
	}

	// Trimmed restore points are gone from RevisionByID too, not just the
	// ActionRestorePoint-filtered list.
	if _, ok := e.Revisions.RevisionByID(ref.ResourceKind, ref.ResourceID, restorePointIDs[0]); ok {
		t.Fatalf("expected the oldest restore point %q to have been physically deleted", restorePointIDs[0])
	}
	if _, ok := e.Revisions.RevisionByID(ref.ResourceKind, ref.ResourceID, restorePointIDs[len(restorePointIDs)-1]); !ok {
		t.Fatal("expected the newest restore point to still be addressable")
	}
}

// TestRetentionNeverTrimsRestorePointBackingInFlightRestore proves the
// negative case the spec calls out explicitly: an old restore point that
// would otherwise fall outside the retention window must survive if this
// exact transition is restoring FROM it -- trimming can never delete the
// revision a live Restore call is reading from.
func TestRetentionNeverTrimsRestorePointBackingInFlightRestore(t *testing.T) {
	e, host := newTestEngine(t)
	e.Retain = RetentionPolicy{MaxRevisions: 2, MinRestorePointKeep: 1}
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")

	host.setLive(ref, map[string]string{"title": "v0"})
	host.setDraft(ref, map[string]string{"title": "v1"})
	first, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"})
	if err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	oldestRestorePointID := first.RestorePointID

	for i := 2; i <= 4; i++ {
		host.setDraft(ref, map[string]string{"title": versionLabel(i)})
		if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: versionLabel(i)}); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}
	// Sanity: with cap 2 and no protection, the oldest restore point from
	// publish 1 would normally already be gone after 4 total publishes.
	if _, ok := e.Revisions.RevisionByID(ref.ResourceKind, ref.ResourceID, oldestRestorePointID); ok {
		t.Fatal("test setup invariant broken: expected the oldest restore point to already be trimmed before the in-flight restore")
	}

	// Re-seed it directly through the store so we can restore from an id
	// that is, at the moment Restore runs, already outside the keep window
	// -- exactly the scenario retention must protect against deleting mid-
	// transaction.
	reseeded, err := e.Revisions.SaveRevision(revisionInputFor(ref, "v0"))
	if err != nil {
		t.Fatalf("reseed outside-window revision: %v", err)
	}

	restoreResult, err := e.Restore(ctx, RestoreCommand{
		Target:      ref,
		Actor:       actor,
		RevisionID:  reseeded.ID,
		OperationID: "restore-in-flight",
	})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}

	if _, ok := e.Revisions.RevisionByID(ref.ResourceKind, ref.ResourceID, reseeded.ID); !ok {
		t.Fatalf("expected the restore-point backing the in-flight restore (%q) to survive retention", reseeded.ID)
	}
	if got := host.currentLive(ref); got["title"] != "v0" {
		t.Fatalf("expected restore to still succeed against the protected revision, got %#v", got)
	}
	if restoreResult.RestorePointID == "" {
		t.Fatal("expected the restore's own pre-restore restore point to be minted")
	}
}

func versionLabel(i int) string {
	return "v" + string(rune('0'+i))
}

// revisionInputFor builds a RevisionInput stamped with cmsstore.ActionRestorePoint
// (so it counts toward retention exactly like an engine-minted one) for
// directly seeding the store in a retention test. Created is pinned to a
// fixed, unambiguously-oldest timestamp (rather than left zero, which would
// fall back to the store's real wall clock and could non-deterministically
// race the fake engine clock used elsewhere in this test) so the revision is
// deterministically outside the retention window regardless of when the
// test happens to run.
func revisionInputFor(ref TargetRef, title string) cmsstore.RevisionInput {
	return cmsstore.RevisionInput{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       cmsstore.ActionRestorePoint,
		Summary:      "Reseeded for retention test.",
		Snapshot:     map[string]string{"title": title},
		Created:      time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}
