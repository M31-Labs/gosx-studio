package engine

import "m31labs.dev/gosx-studio/cms/lifecycle"

// Actor is the authenticated caller driving a transition. Caps is resolved
// server-side (see CapabilitySet) and is never trusted from the browser.
type Actor struct {
	ID    string
	Label string
	Caps  CapabilitySet
}

// CapabilitySet gates every transition. It must be resolved from the
// authenticated request context by the host, never from client input.
type CapabilitySet struct {
	CanPublish             bool
	CanPromote             bool
	CanRestore             bool
	CanConfigureProduction bool
}

// ReviewCommand opens a review (records a Pending PublishDecision).
type ReviewCommand struct {
	Target TargetRef
	Actor  Actor
	Note   string
}

// ReviewResult reports the recorded decision.
type ReviewResult struct {
	Decision lifecycle.PublishDecision
}

// ReadyCommand approves a review once readiness is clear.
type ReadyCommand struct {
	Target TargetRef
	Actor  Actor
	Note   string
}

// ReadyResult reports the recorded decision and the readiness gate that
// cleared it.
type ReadyResult struct {
	Decision lifecycle.PublishDecision
	Gate     Gate
}

// PublishCommand requests a publish. OperationID shares the descriptor-02
// durable-operation id space so the same submit that failed mid-flight (or
// was resubmitted by an impatient click) replays instead of double-applying.
// ExpectedRevisionID is the optimistic head check: "" skips it.
// RestorePointID is engine-populated right before Host.ApplyPublish; callers
// never set it.
type PublishCommand struct {
	Target             TargetRef
	Actor              Actor
	OperationID        string
	ExpectedRevisionID string
	RestorePointID     string
	Summary            string
}

// PublishResult is idempotent + revision-addressed: the operation id dedupes
// the write, RevisionID + SnapshotDigest address the exact bytes that went
// live.
type PublishResult struct {
	RevisionID     string
	RestorePointID string
	Diff           lifecycle.RevisionDiff
	SnapshotDigest string
	Idempotent     bool
}

// RestoreCommand rolls a target back to RevisionID, which may be a restore
// point minted by an earlier publish/restore, or any other prior content
// revision on the target.
type RestoreCommand struct {
	Target      TargetRef
	Actor       Actor
	RevisionID  string
	OperationID string
	// RestorePointID is engine-populated right before Host.RestoreLive with
	// the id of the fresh pre-restore restore point Engine just minted for
	// this exact transition (Engine.Restore always mints one, so a bad
	// restore is itself reversible -- see RestoreResult.RestorePointID,
	// which carries the same value back out). Callers never set it.
	//
	// Host-adoption semantics (restore-of-restore symmetry): a host that
	// maintains a companion resource keyed by a target's content-revision id
	// (e.g. an interactions ContentRevision that snapshots collaborative
	// editor state alongside a page's content) MUST mint that companion
	// keyed by RestorePointID inside the same RestoreLive transaction that
	// rewrites live -- exactly mirroring how PublishCommand.RestorePointID
	// already lets a host key a companion to a *publish*-minted restore
	// point. Without this, a later restore whose RevisionID equals this
	// RestorePointID (a "restore-of-restore": undoing an earlier restore)
	// finds no companion to restore and silently skips it. A host that has
	// not adopted this yet may leave the field unused; that is the prior
	// (pre-fix) behavior, not a regression.
	RestorePointID string
}

// RestoreResult is the outcome of a restore: RevisionID is the NEW live
// revision minted by the restore; RestorePointID is the pre-restore
// snapshot Engine minted before calling the host (a restore always mints
// one, so a bad restore is itself reversible) -- the same id threaded into
// RestoreCommand.RestorePointID for the host that performed the restore.
// Idempotent mirrors PublishResult.Idempotent: true when this result was
// replayed from a prior ActionRestoreCompleted ledger entry sharing
// OperationID rather than produced by a fresh Host.RestoreLive call.
type RestoreResult struct {
	RevisionID     string
	RestorePointID string
	Diff           lifecycle.RevisionDiff
	Idempotent     bool
}

// DraftPreview is the exact-draft-diff surface for a target: the structural
// diff between live and draft (Diff/HasDiff, from Host.DraftDiff) plus the
// domain-classified, actor-attributed operation-log change set (Changes,
// spec §3 / slice S3) derived from Host.OperationLog since the target's last
// publish. Changes is what Publish is about to promote to live, itemized;
// Diff is the same promotion summarized as net before/after state.
//
// HasChanges is presentation-only: an empty change set does not block
// Engine.Publish (see change_set.go doc comment "Empty-diff publish
// behavior"); it lets a host render "nothing to publish" without adding a
// new server-side gate this slice was not asked to add.
type DraftPreview struct {
	Diff       lifecycle.RevisionDiff
	HasDiff    bool
	Changes    []DraftChange
	HasChanges bool
}
