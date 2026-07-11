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
		revisions: revisions,
		live:      map[TargetRef]map[string]string{},
		draft:     map[TargetRef]map[string]string{},
		blockers:  map[TargetRef][]Blocker{},
		ops:       map[TargetRef][]authoring.OperationRecord{},
		applied:   map[string]PublishOutcome{},
	}
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
