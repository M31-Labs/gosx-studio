package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// fakeHost is a minimal in-memory HostLifecycle used to exercise Engine
// transitions without depending on any concrete host repo (Noni/Pajaritos
// adapters land in later slices, S5/S6). It shares Engine's RevisionStore,
// matching how a real host adapter resolves RestoreCommand.RevisionID
// against the store it already owns (spec §5/§2).
type fakeHost struct {
	mu        sync.Mutex
	revisions cmsstore.RevisionStore

	live     map[TargetRef]map[string]string
	draft    map[TargetRef]map[string]string
	blockers map[TargetRef][]Blocker
	ops      map[TargetRef][]authoring.OperationRecord

	// companion models a host-owned resource keyed by a target's content
	// revision id -- e.g. Noni's interactions ContentRevision, which
	// snapshots collaborative editor state alongside a page's published/
	// restored content. companionLive is the current value; companionByRev
	// captures a companion snapshot the moment RestoreLive mints a fresh
	// pre-restore restore point (keyed by RestoreCommand.RestorePointID),
	// mirroring how Engine captures the pre-restore *content* snapshot into
	// the RevisionStore under that same id. This exists purely to prove the
	// restore-of-restore companion round trip (see
	// TestRestoreOfRestoreRoundTripsHostCompanion in engine_test.go).
	companionLive  map[TargetRef]string
	companionByRev map[TargetRef]map[string]string

	// publishedOpRevision is the fake host's answer to
	// PublishedOperationRevision: the durable operation-log DocumentRevision
	// high-water mark as of the target's last publish. ApplyPublish advances
	// it automatically to the highest DocumentRevision among the ops seeded
	// for that target so far (mirroring how a real host would correlate its
	// content revision with the durable log at publish time); tests may also
	// set it directly for scenarios that do not go through a full Publish.
	publishedOpRevision map[TargetRef]uint64

	// applied gives the fake host its own defense-in-depth idempotency by
	// operation id, mirroring the "MUST be idempotent on cmd.OperationID"
	// requirement on HostLifecycle.ApplyPublish.
	applied map[string]PublishOutcome

	revSeq int

	applyPublishCalls int
	restoreLiveCalls  int
	snapshotLiveCalls int
}

func newFakeHost(revisions cmsstore.RevisionStore) *fakeHost {
	return &fakeHost{
		revisions:           revisions,
		live:                map[TargetRef]map[string]string{},
		draft:               map[TargetRef]map[string]string{},
		blockers:            map[TargetRef][]Blocker{},
		ops:                 map[TargetRef][]authoring.OperationRecord{},
		applied:             map[string]PublishOutcome{},
		publishedOpRevision: map[TargetRef]uint64{},
		companionLive:       map[TargetRef]string{},
		companionByRev:      map[TargetRef]map[string]string{},
	}
}

// setLiveCompanion sets the current value of the fake host's companion
// resource for ref (see the companionLive/companionByRev field doc).
func (h *fakeHost) setLiveCompanion(ref TargetRef, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.companionLive[ref] = value
}

// companionSnapshotFor reports the companion value RestoreLive captured
// under revisionOrRestorePointID, if any.
func (h *fakeHost) companionSnapshotFor(ref TargetRef, revisionOrRestorePointID string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	value, ok := h.companionByRev[ref][revisionOrRestorePointID]
	return value, ok
}

// currentLiveCompanion reports the fake host's current companion value.
func (h *fakeHost) currentLiveCompanion(ref TargetRef) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.companionLive[ref]
}

// addOp appends one durable operation-log record for ref, as a real host's
// collab-backed OperationLog would report it. Tests use this to seed a mixed
// operation sequence for the change-set surface (spec §3 / S3) without
// depending on the actual cms/studio/collab persistence layer.
func (h *fakeHost) addOp(ref TargetRef, op authoring.OperationRecord) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ops[ref] = append(h.ops[ref], op.Normalize())
}

func (h *fakeHost) setLive(ref TargetRef, attrs map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.live[ref] = cloneAttrs(attrs)
}

func (h *fakeHost) setDraft(ref TargetRef, attrs map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.draft[ref] = cloneAttrs(attrs)
}

func (h *fakeHost) setBlockers(ref TargetRef, blockers ...Blocker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.blockers[ref] = append([]Blocker(nil), blockers...)
}

func (h *fakeHost) currentLive(ref TargetRef) map[string]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return cloneAttrs(h.live[ref])
}

func (h *fakeHost) SnapshotLive(ctx context.Context, ref TargetRef) (any, bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.snapshotLiveCalls++
	live, ok := h.live[ref]
	if !ok {
		return nil, false, nil
	}
	return cloneAttrs(live), true, nil
}

func (h *fakeHost) DraftDiff(ctx context.Context, ref TargetRef) (lifecycle.RevisionDiff, bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	draft, hasDraft := h.draft[ref]
	if !hasDraft {
		return lifecycle.RevisionDiff{}, false, nil
	}
	diff, err := diffAttrs(ref, h.live[ref], draft)
	return diff, true, err
}

func (h *fakeHost) Readiness(ctx context.Context, ref TargetRef) ([]Blocker, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]Blocker(nil), h.blockers[ref]...), nil
}

func (h *fakeHost) ApplyPublish(ctx context.Context, cmd PublishCommand) (PublishOutcome, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.applyPublishCalls++
	if cmd.OperationID != "" {
		if out, ok := h.applied[cmd.OperationID]; ok {
			return out, nil
		}
	}
	live := h.live[cmd.Target]
	draft := h.draft[cmd.Target]
	diff, err := diffAttrs(cmd.Target, live, draft)
	if err != nil {
		return PublishOutcome{}, err
	}
	merged := cloneAttrs(live)
	for key, value := range draft {
		merged[key] = value
	}
	h.live[cmd.Target] = merged
	h.revSeq++
	outcome := PublishOutcome{
		RevisionID:     fmt.Sprintf("live-rev-%d", h.revSeq),
		SnapshotDigest: digestOf(merged),
		Diff:           diff,
	}
	if cmd.OperationID != "" {
		h.applied[cmd.OperationID] = outcome
	}
	h.advancePublishedOpRevisionLocked(cmd.Target)
	return outcome, nil
}

func (h *fakeHost) RestoreLive(ctx context.Context, cmd RestoreCommand) (RestoreOutcome, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.restoreLiveCalls++
	revision, ok := h.revisions.RevisionByID(cmd.Target.ResourceKind, cmd.Target.ResourceID, cmd.RevisionID)
	if !ok {
		return RestoreOutcome{}, fmt.Errorf("fake host: revision %q not found for %s/%s", cmd.RevisionID, cmd.Target.ResourceKind, cmd.Target.ResourceID)
	}
	attrs, err := lifecycle.DecodeSnapshot[map[string]string](revision)
	if err != nil {
		return RestoreOutcome{}, err
	}
	live := h.live[cmd.Target]
	diff, err := diffAttrs(cmd.Target, live, attrs)
	if err != nil {
		return RestoreOutcome{}, err
	}

	// Companion round trip (proves RestoreCommand.RestorePointID closes the
	// restore-of-restore gap, see host-adoption notes on that field): mint
	// the companion snapshot for the fresh pre-restore restore point Engine
	// just handed us, keyed the same way the RevisionStore keys the content
	// snapshot for it. A later restore whose RevisionID targets this exact
	// RestorePointID can then find and restore the companion below.
	if cmd.RestorePointID != "" {
		if h.companionByRev[cmd.Target] == nil {
			h.companionByRev[cmd.Target] = map[string]string{}
		}
		h.companionByRev[cmd.Target][cmd.RestorePointID] = h.companionLive[cmd.Target]
	}
	if companion, ok := h.companionByRev[cmd.Target][cmd.RevisionID]; ok {
		h.companionLive[cmd.Target] = companion
	}

	h.live[cmd.Target] = cloneAttrs(attrs)
	h.revSeq++
	newRevID := fmt.Sprintf("live-rev-%d", h.revSeq)
	if h.revisions != nil {
		if _, err := h.revisions.SaveRevision(cmsstore.RevisionInput{
			ResourceKind: cmd.Target.ResourceKind,
			ResourceID:   cmd.Target.ResourceID,
			Action:       cmsstore.ActionRestoreApplied,
			Summary:      fmt.Sprintf("Restored live from revision %s.", cmd.RevisionID),
			Snapshot:     attrs,
		}); err != nil {
			return RestoreOutcome{}, err
		}
	}
	return RestoreOutcome{RevisionID: newRevID, Diff: diff}, nil
}

func (h *fakeHost) OperationLog(ctx context.Context, ref TargetRef, sinceRevision uint64) ([]authoring.OperationRecord, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]authoring.OperationRecord, 0)
	for _, op := range h.ops[ref] {
		if op.DocumentRevision > sinceRevision {
			out = append(out, op)
		}
	}
	return out, nil
}

func (h *fakeHost) PublishedOperationRevision(ctx context.Context, ref TargetRef) (uint64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.publishedOpRevision[ref], nil
}

// advancePublishedOpRevisionLocked mirrors what a real host would do at
// publish time: correlate the content revision it just promoted with the
// durable operation log's current high-water mark for ref, so a later
// PendingDiff only reports ops applied after this publish. Callers must hold
// h.mu.
func (h *fakeHost) advancePublishedOpRevisionLocked(ref TargetRef) {
	var max uint64
	for _, op := range h.ops[ref] {
		if op.DocumentRevision > max {
			max = op.DocumentRevision
		}
	}
	if max > h.publishedOpRevision[ref] {
		h.publishedOpRevision[ref] = max
	}
}

func cloneAttrs(attrs map[string]string) map[string]string {
	out := make(map[string]string, len(attrs))
	for key, value := range attrs {
		out[key] = value
	}
	return out
}

func diffAttrs(ref TargetRef, from, to map[string]string) (lifecycle.RevisionDiff, error) {
	fromRev, err := lifecycle.NewRevision(lifecycle.RevisionInput{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       "fake.snapshot",
		Snapshot:     from,
	}, time.Time{})
	if err != nil {
		return lifecycle.RevisionDiff{}, err
	}
	toRev, err := lifecycle.NewRevision(lifecycle.RevisionInput{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       "fake.snapshot",
		Snapshot:     to,
	}, time.Time{})
	if err != nil {
		return lifecycle.RevisionDiff{}, err
	}
	return lifecycle.DiffRevisions(fromRev, toRev)
}

func digestOf(attrs map[string]string) string {
	data, _ := json.Marshal(attrs)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
