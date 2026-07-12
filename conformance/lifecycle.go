package conformance

import (
	"context"
	"errors"
	"testing"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	"m31labs.dev/gosx-studio/cms/lifecycle/engine"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// CompanionFixture is the optional seam a LifecycleFixture supplies only when
// its HostLifecycle implementation keys a companion resource (e.g. Noni's
// interactions ContentRevision) to RestoreCommand.RestorePointID, per that
// field's doc comment. Set, Current, and SnapshotFor must all operate on the
// same TargetRef the enclosing LifecycleFixture.Target names.
type CompanionFixture struct {
	// Set replaces the target's current companion value.
	Set func(value string)
	// Current reads the target's current companion value.
	Current func() string
	// SnapshotFor reports the companion value a prior RestoreLive call
	// captured under revisionOrRestorePointID (see HostLifecycle.RestoreLive's
	// doc comment), and whether one was captured at all.
	SnapshotFor func(revisionOrRestorePointID string) (value string, ok bool)
}

// LifecycleFixture is what a host's factory builds for one conformance
// scenario: a fully wired *engine.Engine (Host set to the real adapter under
// test, Revisions/Ledger set to whatever store the host really shares with
// its adapter -- typically cms/lifecycle/sqlstore.Store) plus narrow setup
// hooks the suite uses to arrange each scenario's starting state without
// reaching into host-internal storage.
//
// A factory is called once per subtest (see LifecycleFactory) and must
// return fresh, isolated state every time -- scenarios never share a Target
// or a store across subtests.
type LifecycleFixture struct {
	// Engine is wired against the host's real HostLifecycle implementation.
	Engine *engine.Engine
	// Target is the one TargetRef every scenario in this fixture run runs
	// against.
	Target engine.TargetRef
	// SeedLive marshals attrs as Target's current live state, so
	// Host.SnapshotLive reports (attrs, true, nil) and Host.ApplyPublish/
	// Host.RestoreLive have a real "before" state to diff against.
	SeedLive func(attrs map[string]string)
	// SeedDraft marshals attrs as Target's pending draft, so Host.DraftDiff
	// reports (diff-against-live, true, nil).
	SeedDraft func(attrs map[string]string)
	// SeedBlockers replaces Target's current readiness blockers (Host.Readiness).
	SeedBlockers func(blockers ...engine.Blocker)
	// CurrentLive reads back Target's current live state, for asserting what
	// Publish/Restore actually promoted.
	CurrentLive func() map[string]string
	// Companion is optional (nil is valid): only set by a fixture whose
	// HostLifecycle implementation adopted the RestorePointID companion
	// contract. When nil, RunHostLifecycleConformance skips (not fails) the
	// restore-of-restore companion scenario -- not every host has a
	// companion resource to prove this against.
	Companion *CompanionFixture
	// Cleanup releases anything the factory opened (temp files, DB handles).
	// Optional.
	Cleanup func()
}

// LifecycleFactory builds one fresh LifecycleFixture. RunHostLifecycleConformance
// calls it once per t.Run subtest.
type LifecycleFactory func(t *testing.T) LifecycleFixture

// publisherActor returns an Actor with every capability granted -- every
// scenario in this suite exercises the happy-path capability gate; a host
// that wants dedicated authorization-boundary coverage keeps that in its own
// test suite (this package intentionally does not assert on host-specific
// authorization wiring, only on Engine's own contract with HostLifecycle).
func publisherActor(id string) engine.Actor {
	return engine.Actor{ID: id, Label: id, Caps: engine.CapabilitySet{CanPublish: true, CanRestore: true, CanPromote: true}}
}

func restorePointRevisions(t *testing.T, e *engine.Engine, ref engine.TargetRef) []lifecycle.Revision {
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

// RunHostLifecycleConformance proves a HostLifecycle adapter satisfies the
// contract cms/lifecycle/engine.Engine relies on:
//
//   - Publish mints a restore point capturing the PRE-publish live state
//     (not the merged post-publish state) before ever calling Host.ApplyPublish.
//   - Publish is idempotent on PublishCommand.OperationID: a resubmit makes
//     no additional Host.ApplyPublish call and returns the same revision/
//     restore-point ids with Idempotent=true.
//   - Publish never calls the host (no ApplyPublish, no SnapshotLive, no
//     restore point minted) when Host.Readiness reports a blocking Blocker.
//   - Restore round-trips Target back to a prior revision's content and
//     itself mints a fresh pre-restore restore point capturing what was live
//     immediately before the restore.
//   - Restore is idempotent on RestoreCommand.OperationID exactly like Publish.
//   - When the fixture supplies a CompanionFixture, a "restore of a restore"
//     (a second Restore whose RevisionID equals the first Restore's own
//     minted RestorePointID) finds and restores the companion value that
//     first restore captured -- proving the host adopted
//     RestoreCommand.RestorePointID's companion contract. Fixtures with no
//     companion resource skip this one scenario via t.Skip, not a failure.
func RunHostLifecycleConformance(t *testing.T, factory LifecycleFactory) {
	t.Helper()

	t.Run("PublishMintsRestorePointCapturingPrePublishLive", func(t *testing.T) {
		fx := factory(t)
		if fx.Cleanup != nil {
			t.Cleanup(fx.Cleanup)
		}
		ctx := context.Background()
		fx.SeedLive(map[string]string{"title": "Original"})
		fx.SeedDraft(map[string]string{"title": "Draft V1"})

		result, err := fx.Engine.Publish(ctx, engine.PublishCommand{
			Target: fx.Target, Actor: publisherActor("conformance-actor"), OperationID: "op-1", Summary: "conformance publish",
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
		if got := fx.CurrentLive(); got["title"] != "Draft V1" {
			t.Fatalf("expected live to reflect the published draft, got %#v", got)
		}

		points := restorePointRevisions(t, fx.Engine, fx.Target)
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
	})

	t.Run("PublishIsIdempotentOnOperationID", func(t *testing.T) {
		fx := factory(t)
		if fx.Cleanup != nil {
			t.Cleanup(fx.Cleanup)
		}
		ctx := context.Background()
		fx.SeedLive(map[string]string{"title": "Original"})
		fx.SeedDraft(map[string]string{"title": "Draft V1"})

		cmd := engine.PublishCommand{Target: fx.Target, Actor: publisherActor("conformance-actor"), OperationID: "op-dup", Summary: "dup"}
		first, err := fx.Engine.Publish(ctx, cmd)
		if err != nil {
			t.Fatalf("first Publish: %v", err)
		}
		second, err := fx.Engine.Publish(ctx, cmd)
		if err != nil {
			t.Fatalf("second Publish: %v", err)
		}
		if !second.Idempotent {
			t.Fatal("expected the resubmitted operation id to report Idempotent=true")
		}
		if second.RevisionID != first.RevisionID || second.RestorePointID != first.RestorePointID {
			t.Fatalf("expected the replay to return the same ids, got first=%#v second=%#v", first, second)
		}
		if points := restorePointRevisions(t, fx.Engine, fx.Target); len(points) != 1 {
			t.Fatalf("expected a single minted restore point across both calls, got %d", len(points))
		}
	})

	t.Run("PublishBlockedByReadinessNeverCallsHost", func(t *testing.T) {
		fx := factory(t)
		if fx.Cleanup != nil {
			t.Cleanup(fx.Cleanup)
		}
		ctx := context.Background()
		fx.SeedLive(map[string]string{"title": "Original"})
		fx.SeedDraft(map[string]string{"title": "Draft V1"})
		fx.SeedBlockers(engine.Blocker{Key: "conformance-blocker", Scope: "asset", Summary: "Missing image", Blocking: true})

		liveBefore := fx.CurrentLive()
		_, err := fx.Engine.Publish(ctx, engine.PublishCommand{Target: fx.Target, Actor: publisherActor("conformance-actor"), OperationID: "op-blocked"})
		var readinessErr engine.ErrReadiness
		if !errors.As(err, &readinessErr) {
			t.Fatalf("expected ErrReadiness, got %v", err)
		}
		if !readinessErr.Gate.Blocking() {
			t.Fatal("expected the returned gate to still report blocking")
		}
		if got := fx.CurrentLive(); !mapsEqual(got, liveBefore) {
			t.Fatalf("expected live to be untouched when readiness blocks publish, before=%#v after=%#v", liveBefore, got)
		}
		if points := restorePointRevisions(t, fx.Engine, fx.Target); len(points) != 0 {
			t.Fatalf("expected no restore point minted when publish is blocked, got %d", len(points))
		}
	})

	t.Run("RestoreRoundTripsToPriorRevisionAndMintsPreRestoreRestorePoint", func(t *testing.T) {
		fx := factory(t)
		if fx.Cleanup != nil {
			t.Cleanup(fx.Cleanup)
		}
		ctx := context.Background()
		actor := publisherActor("conformance-actor")
		fx.SeedLive(map[string]string{"title": "Original"})
		fx.SeedDraft(map[string]string{"title": "Draft V1"})

		first, err := fx.Engine.Publish(ctx, engine.PublishCommand{Target: fx.Target, Actor: actor, OperationID: "op-1"})
		if err != nil {
			t.Fatalf("first Publish: %v", err)
		}
		originalRestorePointID := first.RestorePointID

		fx.SeedDraft(map[string]string{"title": "Draft V2"})
		if _, err := fx.Engine.Publish(ctx, engine.PublishCommand{Target: fx.Target, Actor: actor, OperationID: "op-2"}); err != nil {
			t.Fatalf("second Publish: %v", err)
		}
		if got := fx.CurrentLive(); got["title"] != "Draft V2" {
			t.Fatalf("expected live to be Draft V2 before restore, got %#v", got)
		}

		restoreResult, err := fx.Engine.Restore(ctx, engine.RestoreCommand{Target: fx.Target, Actor: actor, RevisionID: originalRestorePointID, OperationID: "restore-1"})
		if err != nil {
			t.Fatalf("Restore: %v", err)
		}
		if restoreResult.RestorePointID == "" || restoreResult.RestorePointID == originalRestorePointID {
			t.Fatalf("expected a fresh pre-restore restore point, got %q (original %q)", restoreResult.RestorePointID, originalRestorePointID)
		}
		if got := fx.CurrentLive(); got["title"] != "Original" {
			t.Fatalf("expected round trip to restore the original live content, got %#v", got)
		}

		preRestore, ok := fx.Engine.Revisions.RevisionByID(fx.Target.ResourceKind, fx.Target.ResourceID, restoreResult.RestorePointID)
		if !ok {
			t.Fatal("expected the restore's own pre-restore restore point to be addressable by id")
		}
		preRestoreSnapshot, err := lifecycle.DecodeSnapshot[map[string]string](preRestore)
		if err != nil {
			t.Fatalf("decode pre-restore snapshot: %v", err)
		}
		if preRestoreSnapshot["title"] != "Draft V2" {
			t.Fatalf("expected the restore's pre-restore restore point to capture Draft V2 (live immediately before the restore), got %#v", preRestoreSnapshot)
		}
	})

	t.Run("RestoreIsIdempotentOnOperationID", func(t *testing.T) {
		fx := factory(t)
		if fx.Cleanup != nil {
			t.Cleanup(fx.Cleanup)
		}
		ctx := context.Background()
		actor := publisherActor("conformance-actor")
		fx.SeedLive(map[string]string{"title": "Original"})
		fx.SeedDraft(map[string]string{"title": "Draft V1"})

		published, err := fx.Engine.Publish(ctx, engine.PublishCommand{Target: fx.Target, Actor: actor, OperationID: "op-1"})
		if err != nil {
			t.Fatalf("Publish: %v", err)
		}

		cmd := engine.RestoreCommand{Target: fx.Target, Actor: actor, RevisionID: published.RestorePointID, OperationID: "restore-dup"}
		first, err := fx.Engine.Restore(ctx, cmd)
		if err != nil {
			t.Fatalf("first Restore: %v", err)
		}
		second, err := fx.Engine.Restore(ctx, cmd)
		if err != nil {
			t.Fatalf("second Restore: %v", err)
		}
		if !second.Idempotent {
			t.Fatal("expected the resubmitted restore operation id to report Idempotent=true")
		}
		if second.RevisionID != first.RevisionID || second.RestorePointID != first.RestorePointID {
			t.Fatalf("expected the replay to return the same ids, got first=%#v second=%#v", first, second)
		}
	})

	t.Run("RestoreOfRestoreRoundTripsHostCompanion", func(t *testing.T) {
		fx := factory(t)
		if fx.Cleanup != nil {
			t.Cleanup(fx.Cleanup)
		}
		if fx.Companion == nil {
			t.Skip("fixture has no companion resource (RestoreCommand.RestorePointID companion contract not adopted) -- not a failure")
		}
		ctx := context.Background()
		actor := publisherActor("conformance-actor")

		priorRevision, err := fx.Engine.Revisions.SaveRevision(lifecycle.RevisionInput{
			ResourceKind: fx.Target.ResourceKind, ResourceID: fx.Target.ResourceID, Action: "conformance.seed", Snapshot: map[string]string{"title": "v0"},
		})
		if err != nil {
			t.Fatalf("seed prior revision: %v", err)
		}

		fx.SeedLive(map[string]string{"title": "Draft V2 Live"})
		fx.Companion.Set("companion-v2")

		restore1, err := fx.Engine.Restore(ctx, engine.RestoreCommand{Target: fx.Target, Actor: actor, RevisionID: priorRevision.ID, OperationID: "restore-1"})
		if err != nil {
			t.Fatalf("first Restore: %v", err)
		}
		if fx.Companion.Current() != "" {
			t.Fatalf("expected the restore target's own companion to be replaced by the prior (companion-less) revision's absence, got %q", fx.Companion.Current())
		}
		if snapshot, ok := fx.Companion.SnapshotFor(restore1.RestorePointID); !ok || snapshot != "companion-v2" {
			t.Fatalf("expected restore1's fresh restore point (%q) to capture the pre-restore companion value, got %q ok=%v", restore1.RestorePointID, snapshot, ok)
		}

		// Restore of a restore: undo restore1 by restoring to restore1's own
		// minted restore point. The companion snapshot restore1 captured under
		// that exact id must come back.
		restore2, err := fx.Engine.Restore(ctx, engine.RestoreCommand{Target: fx.Target, Actor: actor, RevisionID: restore1.RestorePointID, OperationID: "restore-2"})
		if err != nil {
			t.Fatalf("second (restore-of-restore) Restore: %v", err)
		}
		if got := fx.Companion.Current(); got != "companion-v2" {
			t.Fatalf("expected restore-of-restore to bring back the companion value captured at restore1's restore point, got %q", got)
		}
		_ = restore2
	})
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
