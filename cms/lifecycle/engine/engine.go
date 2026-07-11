package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// Ledger audit actions minted by the engine. These are new AuditEvent.Action
// values; LedgerStore's interface is unchanged (additive, see spec §8).
const (
	// ActionPublishCompleted records a successful (non-idempotent) publish.
	// Publish scans for this action + metadata["operationId"] to replay.
	ActionPublishCompleted = "lifecycle.publish_completed"
	// ActionRestoreCompleted records a successful restore.
	ActionRestoreCompleted = "lifecycle.restore_completed"
	// ActionReviewOpened records Review opening a PublishDecision.
	ActionReviewOpened = "lifecycle.review_opened"
	// ActionReadyMarked records MarkReady approving a PublishDecision.
	ActionReadyMarked = "lifecycle.ready_marked"
	// ActionPromoteRecorded records a promotion intent (spec §7).
	ActionPromoteRecorded = "promote.recorded"
)

// ErrUnauthorized is returned by every transition's first statement when the
// actor lacks the required capability. There is no code path to live that
// bypasses this check (spec §6/§9).
var ErrUnauthorized = errors.New("actor lacks the required capability")

// ErrConflict is returned by Publish when the caller's optimistic
// ExpectedRevisionID no longer matches the current live head. It composes
// with the descriptor-02 durable operation protocol's TargetHead concept at
// the publish layer: the ledger's latest PublishDecision.RevisionID is the
// lifecycle-level "head".
type ErrConflict struct {
	Target             TargetRef
	ExpectedRevisionID string
	ActualRevisionID   string
}

func (e ErrConflict) Error() string {
	return fmt.Sprintf("publish conflict on %s/%s: expected head %q, current head %q",
		e.Target.ResourceKind, e.Target.ResourceID, e.ExpectedRevisionID, e.ActualRevisionID)
}

// Engine is the only place lifecycle transitions are sequenced. Hosts
// contribute persistence via HostLifecycle (host.go); Revisions and Ledger
// are the existing cms/store RevisionStore and cms/lifecycle LedgerStore --
// no new storage backend is introduced.
type Engine struct {
	// Revisions is the existing RevisionStore restore points (and every
	// other revision) live in. Hosts share the same store instance so
	// HostLifecycle implementations can resolve restore-point/revision ids
	// Engine hands them (e.g. RestoreCommand.RevisionID).
	Revisions cmsstore.RevisionStore
	// Ledger is the existing decisions+audit store.
	Ledger lifecycle.LedgerStore
	// Host is the narrow per-host persistence seam (host.go).
	Host HostLifecycle
	// Now defaults to time.Now.
	Now func() time.Time
	// NewID defaults to a time-based id; hosts typically inject their own
	// id generator for deterministic tests/ids.
	NewID func(prefix string) string
	// Retain is the restore-point-aware retention policy (restorepoint.go).
	Retain RetentionPolicy
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Engine) newID(prefix string) string {
	if e.NewID != nil {
		return e.NewID(prefix)
	}
	return fmt.Sprintf("%s_%d", prefix, e.now().UnixNano())
}

// requireCap returns ErrUnauthorized when ok is false.
func (e *Engine) requireCap(ok bool) error {
	if !ok {
		return ErrUnauthorized
	}
	return nil
}

// Readiness reports the host-computed publish blockers for ref. Publish
// enforces this gate server-side; the panel's button state is advisory
// only.
func (e *Engine) Readiness(ctx context.Context, ref TargetRef) (Gate, error) {
	if e.Host == nil {
		return Gate{}, nil
	}
	blockers, err := e.Host.Readiness(ctx, ref)
	if err != nil {
		return Gate{}, err
	}
	return Gate{Blockers: blockers}, nil
}

// PendingDiff returns the structural draft-diff surface for ref. The
// operation-log change-set classification (studio.DraftChange) lands in a
// later slice (spec §3 / S3).
func (e *Engine) PendingDiff(ctx context.Context, ref TargetRef) (DraftPreview, error) {
	if e.Host == nil {
		return DraftPreview{}, nil
	}
	diff, hasDraft, err := e.Host.DraftDiff(ctx, ref)
	if err != nil {
		return DraftPreview{}, err
	}
	return DraftPreview{Diff: diff, HasDiff: hasDraft}, nil
}

// Review opens a review by recording a Pending PublishDecision.
func (e *Engine) Review(ctx context.Context, cmd ReviewCommand) (ReviewResult, error) {
	decision, err := e.Ledger.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		Status:       lifecycle.DecisionPending,
		ActorID:      cmd.Actor.ID,
		Note:         cmd.Note,
		Created:      e.now(),
	})
	if err != nil {
		return ReviewResult{}, err
	}
	if _, err := e.Ledger.SaveAuditEvent(ctx, lifecycle.AuditEventInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		Action:       ActionReviewOpened,
		ActorID:      cmd.Actor.ID,
		Summary:      cmd.Note,
		Created:      e.now(),
	}); err != nil {
		return ReviewResult{}, err
	}
	return ReviewResult{Decision: decision}, nil
}

// MarkReady approves a pending review once readiness is clear. It fails
// closed with ErrReadiness when a blocking blocker remains.
func (e *Engine) MarkReady(ctx context.Context, cmd ReadyCommand) (ReadyResult, error) {
	gate, err := e.Readiness(ctx, cmd.Target)
	if err != nil {
		return ReadyResult{}, err
	}
	if gate.Blocking() {
		return ReadyResult{Gate: gate}, ErrReadiness{Gate: gate}
	}
	decision, err := e.Ledger.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		Status:       lifecycle.DecisionApproved,
		ActorID:      cmd.Actor.ID,
		Note:         cmd.Note,
		Created:      e.now(),
	})
	if err != nil {
		return ReadyResult{}, err
	}
	if _, err := e.Ledger.SaveAuditEvent(ctx, lifecycle.AuditEventInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		Action:       ActionReadyMarked,
		ActorID:      cmd.Actor.ID,
		Summary:      cmd.Note,
		Created:      e.now(),
	}); err != nil {
		return ReadyResult{}, err
	}
	return ReadyResult{Decision: decision, Gate: gate}, nil
}

// Publish promotes the current draft to live. Normative sequence (spec
// §1.2):
//  1. requireCap(actor.Caps.CanPublish) -> else ErrUnauthorized.
//  2. gate := Readiness(...); if gate.Blocking() -> ErrReadiness{gate}.
//  3. Idempotency replay: scan ledger audit for
//     metadata["operationId"]==cmd.OperationID with Action ==
//     ActionPublishCompleted; if found, return the recorded PublishResult
//     with Idempotent:true and no host write.
//  4. Optimistic head check: if cmd.ExpectedRevisionID != "" and it != the
//     current live head (the ledger's latest PublishDecision.RevisionID)
//     -> ErrConflict.
//  5. Mint restore point: SnapshotLive + Revisions.SaveRevision.
//  6. Host.ApplyPublish promotes draft->live in one host transaction.
//  7. Record ledger: SavePublishDecision + SaveAuditEvent(ActionPublishCompleted).
//  8. Return PublishResult{..., Idempotent:false}.
func (e *Engine) Publish(ctx context.Context, cmd PublishCommand) (PublishResult, error) {
	if err := e.requireCap(cmd.Actor.Caps.CanPublish); err != nil {
		return PublishResult{}, err
	}

	gate, err := e.Readiness(ctx, cmd.Target)
	if err != nil {
		return PublishResult{}, err
	}
	if gate.Blocking() {
		return PublishResult{}, ErrReadiness{Gate: gate}
	}

	if cmd.OperationID != "" {
		if result, found, err := e.replayPublish(ctx, cmd); err != nil {
			return PublishResult{}, err
		} else if found {
			return result, nil
		}
	}

	if cmd.ExpectedRevisionID != "" {
		actual := ""
		if latest, ok, err := e.Ledger.LatestPublishDecision(ctx, cmd.Target.ResourceKind, cmd.Target.ResourceID); err != nil {
			return PublishResult{}, err
		} else if ok {
			actual = latest.RevisionID
		}
		if actual != cmd.ExpectedRevisionID {
			return PublishResult{}, ErrConflict{
				Target:             cmd.Target,
				ExpectedRevisionID: cmd.ExpectedRevisionID,
				ActualRevisionID:   actual,
			}
		}
	}

	restorePointID, err := e.mintRestorePoint(ctx, cmd.Target, cmd.OperationID, "")
	if err != nil {
		return PublishResult{}, err
	}

	cmd.RestorePointID = restorePointID
	if e.Host == nil {
		return PublishResult{}, fmt.Errorf("engine: no host configured")
	}
	outcome, err := e.Host.ApplyPublish(ctx, cmd)
	if err != nil {
		return PublishResult{}, err
	}

	if _, err := e.Ledger.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		RevisionID:   outcome.RevisionID,
		Status:       lifecycle.DecisionApproved,
		ActorID:      cmd.Actor.ID,
		Note:         cmd.Summary,
		Created:      e.now(),
	}); err != nil {
		return PublishResult{}, err
	}

	result := PublishResult{
		RevisionID:     outcome.RevisionID,
		RestorePointID: restorePointID,
		Diff:           outcome.Diff,
		SnapshotDigest: outcome.SnapshotDigest,
		Idempotent:     false,
	}
	if err := e.recordPublishCompleted(ctx, cmd, result); err != nil {
		return PublishResult{}, err
	}
	// Retention (spec §2) runs only after every ledger/host write has
	// already succeeded, so a failed publish never loses restore-point
	// history it didn't need to trim. The restore point just minted and the
	// optimistic-head revision id (if any) are protected even though, as the
	// newest entries, the retention window would already keep them.
	if _, err := e.enforceRetention(cmd.Target, restorePointID, cmd.ExpectedRevisionID); err != nil {
		return PublishResult{}, err
	}
	return result, nil
}

// Restore rolls the target back to a previously recorded revision (a
// restore point or any prior content revision). It always mints a
// pre-restore restore point of its own before calling the host, so a bad
// restore is itself reversible (spec §9 TestRestoreMintsRestorePointFirst).
func (e *Engine) Restore(ctx context.Context, cmd RestoreCommand) (RestoreResult, error) {
	if err := e.requireCap(cmd.Actor.Caps.CanRestore); err != nil {
		return RestoreResult{}, err
	}

	summary := fmt.Sprintf("Automatic restore point before restoring to revision %s.", cmd.RevisionID)
	restorePointID, err := e.mintRestorePoint(ctx, cmd.Target, cmd.OperationID, summary)
	if err != nil {
		return RestoreResult{}, err
	}

	if e.Host == nil {
		return RestoreResult{}, fmt.Errorf("engine: no host configured")
	}
	outcome, err := e.Host.RestoreLive(ctx, cmd)
	if err != nil {
		return RestoreResult{}, err
	}

	if _, err := e.Ledger.SaveAuditEvent(ctx, lifecycle.AuditEventInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		RevisionID:   outcome.RevisionID,
		Action:       ActionRestoreCompleted,
		ActorID:      cmd.Actor.ID,
		Metadata: map[string]string{
			"operationId":        cmd.OperationID,
			"restorePointId":     restorePointID,
			"restoredRevisionId": cmd.RevisionID,
			"newRevisionId":      outcome.RevisionID,
		},
		Created: e.now(),
	}); err != nil {
		return RestoreResult{}, err
	}

	// Retention (spec §2) must never delete cmd.RevisionID: it is the
	// restore point (or other prior revision) this exact transition is
	// reading from -- an "in-flight restore" source -- even if it would
	// otherwise fall outside the retention window. The freshly minted
	// pre-restore restore point is protected too, though as the newest entry
	// the window already keeps it.
	if _, err := e.enforceRetention(cmd.Target, restorePointID, cmd.RevisionID); err != nil {
		return RestoreResult{}, err
	}

	return RestoreResult{
		RevisionID:     outcome.RevisionID,
		RestorePointID: restorePointID,
		Diff:           outcome.Diff,
	}, nil
}

// Promote records a promotion intent (spec §7). It never touches k8s/shell:
// it verifies caps + (for production) configuration, verifies the target
// has a published revision, and records the intent as ledger audit
// metadata with SeedData permanently false. Migration classification
// (ErrDestructiveMigration) lands in a later slice (S8).
func (e *Engine) Promote(ctx context.Context, cmd PromoteCommand) (PromotionIntent, error) {
	if err := e.requireCap(cmd.Actor.Caps.CanPromote); err != nil {
		return PromotionIntent{}, err
	}
	if cmd.Environment.Kind == EnvProduction {
		if !cmd.Environment.Configured || !cmd.Actor.Caps.CanConfigureProduction {
			return PromotionIntent{}, ErrProductionNotConfigured
		}
	}

	latest, ok, err := e.Ledger.LatestPublishDecision(ctx, cmd.Target.ResourceKind, cmd.Target.ResourceID)
	if err != nil {
		return PromotionIntent{}, err
	}
	if !ok || latest.RevisionID == "" {
		return PromotionIntent{}, fmt.Errorf("promote: target %s/%s has no published revision", cmd.Target.ResourceKind, cmd.Target.ResourceID)
	}

	intent := PromotionIntent{
		ID:             e.newID("promote"),
		Target:         cmd.Target,
		FromDigest:     cmd.Environment.ImageDigest,
		ToDigest:       cmd.ToDigest,
		RestorePointID: cmd.Environment.DataRestorePointID,
		Actor:          cmd.Actor.ID,
		SeedData:       false,
		Created:        e.now(),
	}

	if _, err := e.Ledger.SaveAuditEvent(ctx, lifecycle.AuditEventInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		RevisionID:   latest.RevisionID,
		Action:       ActionPromoteRecorded,
		ActorID:      cmd.Actor.ID,
		Metadata: map[string]string{
			"promotionId":    intent.ID,
			"fromDigest":     intent.FromDigest,
			"toDigest":       intent.ToDigest,
			"restorePointId": intent.RestorePointID,
			"seedData":       "false",
		},
		Created: e.now(),
	}); err != nil {
		return PromotionIntent{}, err
	}

	return intent, nil
}

// mintRestorePoint captures the pre-transition live snapshot for ref (if the
// host reports one) into the existing RevisionStore, distinguished by
// cmsstore.ActionRestorePoint. It returns "" (not an error) when the host
// has no live snapshot yet (e.g. a target that has never been published).
func (e *Engine) mintRestorePoint(ctx context.Context, ref TargetRef, operationID, summary string) (string, error) {
	if e.Host == nil || e.Revisions == nil {
		return "", nil
	}
	snapshot, ok, err := e.Host.SnapshotLive(ctx, ref)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	rp, err := e.Revisions.SaveRevision(restorePointInput(ref, snapshot, operationID, summary, e.now(), e.newID))
	if err != nil {
		return "", err
	}
	return rp.ID, nil
}

// recordPublishCompleted writes the ActionPublishCompleted audit event a
// later replay scans for. The diff is JSON-encoded into metadata so a
// replay can return byte-identical PublishResult.Diff without touching the
// host.
func (e *Engine) recordPublishCompleted(ctx context.Context, cmd PublishCommand, result PublishResult) error {
	diffJSON, err := json.Marshal(result.Diff)
	if err != nil {
		return err
	}
	_, err = e.Ledger.SaveAuditEvent(ctx, lifecycle.AuditEventInput{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		RevisionID:   result.RevisionID,
		Action:       ActionPublishCompleted,
		ActorID:      cmd.Actor.ID,
		Summary:      cmd.Summary,
		Metadata: map[string]string{
			"operationId":         cmd.OperationID,
			"restorePointId":      result.RestorePointID,
			"publishedRevisionId": result.RevisionID,
			"digest":              result.SnapshotDigest,
			"diff":                string(diffJSON),
		},
		Created: e.now(),
	})
	return err
}

// replayPublish scans the ledger audit for a prior ActionPublishCompleted
// event carrying cmd.OperationID. When found it reconstructs PublishResult
// entirely from ledger metadata: no host method is called.
func (e *Engine) replayPublish(ctx context.Context, cmd PublishCommand) (PublishResult, bool, error) {
	events, err := e.Ledger.ListAuditEvents(ctx, lifecycle.LedgerFilter{
		ResourceKind: cmd.Target.ResourceKind,
		ResourceID:   cmd.Target.ResourceID,
		Action:       ActionPublishCompleted,
	})
	if err != nil {
		return PublishResult{}, false, err
	}
	for _, event := range events {
		if event.Metadata["operationId"] == "" || event.Metadata["operationId"] != cmd.OperationID {
			continue
		}
		var diff lifecycle.RevisionDiff
		_ = json.Unmarshal([]byte(event.Metadata["diff"]), &diff)
		return PublishResult{
			RevisionID:     event.Metadata["publishedRevisionId"],
			RestorePointID: event.Metadata["restorePointId"],
			Diff:           diff,
			SnapshotDigest: event.Metadata["digest"],
			Idempotent:     true,
		}, true, nil
	}
	return PublishResult{}, false, nil
}
