// Package engine is the Studio-owned lifecycle orchestrator described in
// HANDOFF-PUBLISH-MASTERING-07. It owns the draft->review->ready->publish->
// promote->rollback/restore state machine and is the only place transitions
// are sequenced: hosts contribute persistence through the narrow
// HostLifecycle seam and never promote a draft to live on their own.
package engine

import (
	"context"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

// TargetRef addresses one lifecycle target: a resource kind (e.g.
// cmsstore.ResourceKindSiteSettings, cmsstore.ResourceKindPage) plus its
// resource id ("site", a page id, ...).
type TargetRef struct {
	ResourceKind string
	ResourceID   string
}

// HostLifecycle is the single seam every host implements. Hosts only
// marshal their own state and run their own transaction; every transition
// sequencing/gating/restore-point/ledger decision stays in Engine.
type HostLifecycle interface {
	// SnapshotLive is read-only. It marshals the current live state for ref
	// (and every dependent resource the transition touches) so Engine can
	// mint a restore point before mutating anything.
	SnapshotLive(ctx context.Context, ref TargetRef) (snapshot any, ok bool, err error)
	// DraftDiff is read-only. It returns the live->draft change set,
	// typically built from lifecycle.DiffRevisions internally.
	DraftDiff(ctx context.Context, ref TargetRef) (diff lifecycle.RevisionDiff, hasDraft bool, err error)
	// Readiness is read-only. It returns host-computed publish blockers.
	Readiness(ctx context.Context, ref TargetRef) ([]Blocker, error)
	// ApplyPublish is mutating, a single host transaction. It MUST be
	// idempotent on cmd.OperationID. It promotes the pending draft to live
	// and returns the published revision id and digest.
	ApplyPublish(ctx context.Context, cmd PublishCommand) (PublishOutcome, error)
	// RestoreLive is mutating, a single host transaction. It restores
	// cmd.RevisionID to live and mints a new live revision. Engine has
	// already minted the pre-restore restore point before calling this.
	RestoreLive(ctx context.Context, cmd RestoreCommand) (RestoreOutcome, error)
	// OperationLog is read-only. It returns the operation-log tail for the
	// draft change set (see spec §3 / slice S3).
	OperationLog(ctx context.Context, ref TargetRef, sinceRevision uint64) ([]authoring.OperationRecord, error)
	// PublishedOperationRevision is read-only. It reports the durable
	// operation log's DocumentRevision high-water mark as of ref's current
	// live (published) state, so Engine.PendingDiff can ask OperationLog for
	// exactly the tail applied since the last publish (spec §3 / slice S3).
	// Only the host can answer this: it is the one place a target's content
	// revision and its durable operation-log sequence are correlated. A host
	// that has not wired that correlation yet may safely return 0 -- every
	// operation in the log then shows as still-pending, which is the
	// conservative/safe default (never hides a real pending change).
	PublishedOperationRevision(ctx context.Context, ref TargetRef) (uint64, error)
}

// PublishOutcome is what a host reports after ApplyPublish promotes a draft
// to live.
type PublishOutcome struct {
	RevisionID     string
	SnapshotDigest string
	Diff           lifecycle.RevisionDiff
}

// RestoreOutcome is what a host reports after RestoreLive rewrites live from
// a prior revision.
type RestoreOutcome struct {
	RevisionID string
	Diff       lifecycle.RevisionDiff
}
