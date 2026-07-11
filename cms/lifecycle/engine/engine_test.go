package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	lifecyclesql "m31labs.dev/gosx-studio/cms/lifecycle/sqlstore"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// newTestEngine wires an Engine against the real (SQLite in-memory)
// cms/lifecycle/sqlstore.Store for both Ledger and Revisions -- the
// "existing RevisionStore"/"existing LedgerStore" the spec composes with --
// and a fresh fakeHost (the only new/fake seam for this slice).
func newTestEngine(t *testing.T) (*Engine, *fakeHost) {
	t.Helper()
	store, closeStore, err := lifecyclesql.Open(":memory:")
	if err != nil {
		t.Fatalf("open ledger/revision store: %v", err)
	}
	t.Cleanup(func() {
		if err := closeStore(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	host := newFakeHost(store)
	clock := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	idSeq := 0
	e := &Engine{
		Revisions: store,
		Ledger:    store,
		Host:      host,
		Now: func() time.Time {
			clock = clock.Add(time.Second)
			return clock
		},
		NewID: func(prefix string) string {
			idSeq++
			return fmt.Sprintf("%s_%d", prefix, idSeq)
		},
	}
	return e, host
}

func testTarget() TargetRef {
	return TargetRef{ResourceKind: cmsstore.ResourceKindSiteSettings, ResourceID: "site"}
}

func publisherActor(id string) Actor {
	return Actor{ID: id, Label: id, Caps: CapabilitySet{CanPublish: true, CanRestore: true, CanPromote: true}}
}

func restorePointRevisions(t *testing.T, e *Engine, ref TargetRef) []lifecycle.Revision {
	t.Helper()
	all := e.Revisions.ListRevisions(lifecycle.RevisionFilter{ResourceKind: ref.ResourceKind, ResourceID: ref.ResourceID})
	out := make([]lifecycle.Revision, 0, len(all))
	for _, revision := range all {
		if revision.Action == cmsstore.ActionRestorePoint {
			out = append(out, revision)
		}
	}
	return out
}

func TestPublishHappyPathMintsRestorePointAndRecordsLedger(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})

	result, err := e.Publish(ctx, PublishCommand{
		Target:      ref,
		Actor:       publisherActor("actor-1"),
		OperationID: "op-1",
		Summary:     "first publish",
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Idempotent {
		t.Fatal("expected first publish to not be idempotent")
	}
	if result.RevisionID == "" || result.RestorePointID == "" {
		t.Fatalf("expected populated revision/restore-point ids, got %#v", result)
	}
	if host.applyPublishCalls != 1 {
		t.Fatalf("expected exactly one host write, got %d", host.applyPublishCalls)
	}
	if got := host.currentLive(ref); got["title"] != "Draft V1" {
		t.Fatalf("expected live to reflect published draft, got %#v", got)
	}

	// The restore point must capture live BEFORE publish, not the merged
	// post-publish state.
	points := restorePointRevisions(t, e, ref)
	if len(points) != 1 {
		t.Fatalf("expected exactly one restore point, got %d", len(points))
	}
	if points[0].ID != result.RestorePointID {
		t.Fatalf("expected restore point id %q, got %q", result.RestorePointID, points[0].ID)
	}
	snapshot, err := lifecycle.DecodeSnapshot[map[string]string](points[0])
	if err != nil {
		t.Fatalf("decode restore point snapshot: %v", err)
	}
	if snapshot["title"] != "Original" {
		t.Fatalf("expected restore point to capture pre-publish live state, got %#v", snapshot)
	}

	// Ledger must carry the audit trail the idempotent replay scans.
	events, err := e.Ledger.ListAuditEvents(ctx, lifecycle.LedgerFilter{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       ActionPublishCompleted,
	})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected exactly one publish-completed audit event, got %d", len(events))
	}
	event := events[0]
	if event.Metadata["operationId"] != "op-1" {
		t.Fatalf("expected operationId metadata, got %#v", event.Metadata)
	}
	if event.Metadata["publishedRevisionId"] != result.RevisionID {
		t.Fatalf("expected publishedRevisionId to match result, got %#v", event.Metadata)
	}
	if event.Metadata["restorePointId"] != result.RestorePointID {
		t.Fatalf("expected restorePointId to match result, got %#v", event.Metadata)
	}
	if event.ActorID != "actor-1" {
		t.Fatalf("expected editor-driven audit actor to be the real actor, not system, got %q", event.ActorID)
	}
}

func TestPublishIsIdempotentOnOperationID(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})

	cmd := PublishCommand{Target: ref, Actor: publisherActor("actor-1"), OperationID: "op-dup", Summary: "dup"}

	first, err := e.Publish(ctx, cmd)
	if err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	if first.Idempotent {
		t.Fatal("expected first publish to not be idempotent")
	}

	second, err := e.Publish(ctx, cmd)
	if err != nil {
		t.Fatalf("second Publish: %v", err)
	}
	if !second.Idempotent {
		t.Fatal("expected second publish with the same operation id to be idempotent")
	}
	if second.RevisionID != first.RevisionID {
		t.Fatalf("expected the same revision id on replay, got %q vs %q", second.RevisionID, first.RevisionID)
	}
	if second.RestorePointID != first.RestorePointID {
		t.Fatalf("expected the same restore point id on replay, got %q vs %q", second.RestorePointID, first.RestorePointID)
	}

	if host.applyPublishCalls != 1 {
		t.Fatalf("expected zero additional host writes on replay, ApplyPublish called %d times", host.applyPublishCalls)
	}
	if points := restorePointRevisions(t, e, ref); len(points) != 1 {
		t.Fatalf("expected a single minted restore point across both calls, got %d", len(points))
	}
}

func TestPublishBlockedByReadiness(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	host.setBlockers(ref, Blocker{Key: "asset-missing", Scope: "asset", Summary: "Missing image", Blocking: true})

	_, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: publisherActor("actor-1"), OperationID: "op-blocked"})
	var readinessErr ErrReadiness
	if !errors.As(err, &readinessErr) {
		t.Fatalf("expected ErrReadiness, got %v", err)
	}
	if !readinessErr.Gate.Blocking() {
		t.Fatal("expected the returned gate to still report blocking")
	}
	if host.applyPublishCalls != 0 {
		t.Fatalf("expected no host write when readiness blocks publish, got %d calls", host.applyPublishCalls)
	}
	if host.snapshotLiveCalls != 0 {
		t.Fatalf("expected no restore point snapshot attempt when readiness blocks publish, got %d calls", host.snapshotLiveCalls)
	}
	if points := restorePointRevisions(t, e, ref); len(points) != 0 {
		t.Fatalf("expected no restore point minted when publish is blocked, got %d", len(points))
	}
}

func TestPublishUnauthorized(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})

	actor := Actor{ID: "actor-2", Caps: CapabilitySet{CanPublish: false}}
	_, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-unauth"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if host.applyPublishCalls != 0 || host.snapshotLiveCalls != 0 {
		t.Fatalf("expected zero host interaction for an unauthorized publish, got applyPublishCalls=%d snapshotLiveCalls=%d", host.applyPublishCalls, host.snapshotLiveCalls)
	}
}

func TestPublishOptimisticHeadConflict(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})

	first, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"})
	if err != nil {
		t.Fatalf("first Publish: %v", err)
	}

	host.setDraft(ref, map[string]string{"title": "Draft V2"})
	_, err = e.Publish(ctx, PublishCommand{
		Target:             ref,
		Actor:              actor,
		OperationID:        "op-2",
		ExpectedRevisionID: "wrong-head",
	})
	var conflict ErrConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if conflict.ActualRevisionID != first.RevisionID {
		t.Fatalf("expected conflict to report the real current head %q, got %q", first.RevisionID, conflict.ActualRevisionID)
	}
	if host.applyPublishCalls != 1 {
		t.Fatalf("expected the conflicting publish to make no additional host write, got %d calls", host.applyPublishCalls)
	}

	second, err := e.Publish(ctx, PublishCommand{
		Target:             ref,
		Actor:              actor,
		OperationID:        "op-3",
		ExpectedRevisionID: first.RevisionID,
	})
	if err != nil {
		t.Fatalf("expected publish with the correct expected head to succeed, got %v", err)
	}
	if second.RevisionID == first.RevisionID {
		t.Fatal("expected a new revision id for the successful conflict-free publish")
	}
}

func TestRestoreRoundTripToRestorePoint(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")

	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	first, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"})
	if err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	originalRestorePointID := first.RestorePointID

	host.setDraft(ref, map[string]string{"title": "Draft V2"})
	second, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-2"})
	if err != nil {
		t.Fatalf("second Publish: %v", err)
	}
	if got := host.currentLive(ref); got["title"] != "Draft V2" {
		t.Fatalf("expected live to be Draft V2 before restore, got %#v", got)
	}

	restoreResult, err := e.Restore(ctx, RestoreCommand{
		Target:      ref,
		Actor:       actor,
		RevisionID:  originalRestorePointID,
		OperationID: "restore-1",
	})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restoreResult.RevisionID == "" {
		t.Fatal("expected a new live revision id minted by the restore")
	}
	if restoreResult.RestorePointID == "" || restoreResult.RestorePointID == originalRestorePointID {
		t.Fatalf("expected a fresh pre-restore restore point, got %q (original %q)", restoreResult.RestorePointID, originalRestorePointID)
	}

	if got := host.currentLive(ref); got["title"] != "Original" {
		t.Fatalf("expected round trip to restore the original live content, got %#v", got)
	}

	// The restore's own pre-restore restore point must capture what was live
	// immediately before the restore (Draft V2), not the restored content.
	preRestore, ok := e.Revisions.RevisionByID(ref.ResourceKind, ref.ResourceID, restoreResult.RestorePointID)
	if !ok {
		t.Fatal("expected the restore's pre-restore restore point to be addressable by id")
	}
	preRestoreSnapshot, err := lifecycle.DecodeSnapshot[map[string]string](preRestore)
	if err != nil {
		t.Fatalf("decode pre-restore snapshot: %v", err)
	}
	if preRestoreSnapshot["title"] != "Draft V2" {
		t.Fatalf("expected pre-restore restore point to capture Draft V2, got %#v", preRestoreSnapshot)
	}

	if host.restoreLiveCalls != 1 {
		t.Fatalf("expected exactly one host restore write, got %d", host.restoreLiveCalls)
	}
	if second.RevisionID == restoreResult.RevisionID {
		t.Fatal("expected the restore to mint a distinct new live revision id")
	}

	events, err := e.Ledger.ListAuditEvents(ctx, lifecycle.LedgerFilter{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       ActionRestoreCompleted,
	})
	if err != nil {
		t.Fatalf("list restore audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected exactly one restore-completed audit event, got %d", len(events))
	}
	if events[0].Metadata["restoredRevisionId"] != originalRestorePointID {
		t.Fatalf("expected restoredRevisionId to reference the requested restore point, got %#v", events[0].Metadata)
	}
	if events[0].Metadata["newRevisionId"] != restoreResult.RevisionID {
		t.Fatalf("expected newRevisionId to match the result, got %#v", events[0].Metadata)
	}
}

func TestRestoreUnauthorized(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := Actor{ID: "actor-3", Caps: CapabilitySet{CanRestore: false}}

	_, err := e.Restore(ctx, RestoreCommand{Target: ref, Actor: actor, RevisionID: "does-not-matter", OperationID: "restore-x"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if host.restoreLiveCalls != 0 || host.snapshotLiveCalls != 0 {
		t.Fatalf("expected zero host interaction for an unauthorized restore, got restoreLiveCalls=%d snapshotLiveCalls=%d", host.restoreLiveCalls, host.snapshotLiveCalls)
	}
}

func TestReviewThenMarkReady(t *testing.T) {
	e, _ := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")

	reviewResult, err := e.Review(ctx, ReviewCommand{Target: ref, Actor: actor, Note: "please review"})
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if reviewResult.Decision.Status != lifecycle.DecisionPending {
		t.Fatalf("expected pending decision, got %#v", reviewResult.Decision)
	}

	readyResult, err := e.MarkReady(ctx, ReadyCommand{Target: ref, Actor: actor, Note: "looks good"})
	if err != nil {
		t.Fatalf("MarkReady: %v", err)
	}
	if readyResult.Decision.Status != lifecycle.DecisionApproved {
		t.Fatalf("expected approved decision, got %#v", readyResult.Decision)
	}
}

func TestMarkReadyBlockedByReadiness(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")
	host.setBlockers(ref, Blocker{Key: "flow-unbound", Scope: "interaction", Blocking: true})

	_, err := e.MarkReady(ctx, ReadyCommand{Target: ref, Actor: actor})
	var readinessErr ErrReadiness
	if !errors.As(err, &readinessErr) {
		t.Fatalf("expected ErrReadiness, got %v", err)
	}
}

func TestPromoteRecordsIntentWithSeedDataFalse(t *testing.T) {
	e, _ := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")
	host := e.Host.(*fakeHost)
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	intent, err := e.Promote(ctx, PromoteCommand{
		Target:      ref,
		Actor:       actor,
		Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true, ImageDigest: "sha256:from"},
		ToDigest:    "sha256:to",
		OperationID: "promote-1",
	})
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if intent.SeedData {
		t.Fatal("expected SeedData to always be false")
	}
	if intent.FromDigest != "sha256:from" || intent.ToDigest != "sha256:to" {
		t.Fatalf("expected digests to round trip, got %#v", intent)
	}

	events, err := e.Ledger.ListAuditEvents(ctx, lifecycle.LedgerFilter{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       ActionPromoteRecorded,
	})
	if err != nil {
		t.Fatalf("list promote audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected exactly one promote.recorded audit event, got %d", len(events))
	}
	if events[0].Metadata["seedData"] != "false" {
		t.Fatalf("expected seedData=false metadata, got %#v", events[0].Metadata)
	}
}

func TestPromoteToUnconfiguredProductionIsBlocked(t *testing.T) {
	e, _ := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")
	host := e.Host.(*fakeHost)
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	_, err := e.Promote(ctx, PromoteCommand{
		Target:      ref,
		Actor:       actor,
		Environment: Environment{Key: "production", Kind: EnvProduction, Configured: false},
		OperationID: "promote-prod",
	})
	if !errors.Is(err, ErrProductionNotConfigured) {
		t.Fatalf("expected ErrProductionNotConfigured, got %v", err)
	}
}
