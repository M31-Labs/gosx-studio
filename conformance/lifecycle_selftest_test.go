package conformance_test

// This file proves conformance.RunHostLifecycleConformance actually runs,
// end to end, against a tiny in-memory engine.HostLifecycle implementation
// standing in for a real host adapter -- exactly the pattern a host's own
// `_test.go` file (in ITS repo, importing this package) would follow. See
// conformance/doc.go for the wiring pattern a real host uses instead of this
// self-test's in-memory fake.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/lifecycle"
	"m31labs.dev/gosx-studio/cms/lifecycle/engine"
	lifecyclesql "m31labs.dev/gosx-studio/cms/lifecycle/sqlstore"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/conformance"
)

// selfTestHost is a minimal in-memory engine.HostLifecycle. It is
// independently written (not imported) from cms/lifecycle/engine's own
// unexported fakehost_test.go fixture -- that type is test-only and
// unexported, so a real host and this self-test both write their own.
type selfTestHost struct {
	mu             sync.Mutex
	revisions      cmsstore.RevisionStore
	live           map[engine.TargetRef]map[string]string
	draft          map[engine.TargetRef]map[string]string
	blockers       map[engine.TargetRef][]engine.Blocker
	companionLive  map[engine.TargetRef]string
	companionByRev map[engine.TargetRef]map[string]string
	applied        map[string]engine.PublishOutcome
	revSeq         int
}

func newSelfTestHost(revisions cmsstore.RevisionStore) *selfTestHost {
	return &selfTestHost{
		revisions:      revisions,
		live:           map[engine.TargetRef]map[string]string{},
		draft:          map[engine.TargetRef]map[string]string{},
		blockers:       map[engine.TargetRef][]engine.Blocker{},
		companionLive:  map[engine.TargetRef]string{},
		companionByRev: map[engine.TargetRef]map[string]string{},
		applied:        map[string]engine.PublishOutcome{},
	}
}

func cloneStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (h *selfTestHost) SnapshotLive(ctx context.Context, ref engine.TargetRef) (any, bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	live, ok := h.live[ref]
	if !ok {
		return nil, false, nil
	}
	return cloneStringMap(live), true, nil
}

func (h *selfTestHost) DraftDiff(ctx context.Context, ref engine.TargetRef) (lifecycle.RevisionDiff, bool, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	draft, hasDraft := h.draft[ref]
	if !hasDraft {
		return lifecycle.RevisionDiff{}, false, nil
	}
	diff, err := selfTestDiff(ref, h.live[ref], draft)
	return diff, true, err
}

func (h *selfTestHost) Readiness(ctx context.Context, ref engine.TargetRef) ([]engine.Blocker, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]engine.Blocker(nil), h.blockers[ref]...), nil
}

func (h *selfTestHost) ApplyPublish(ctx context.Context, cmd engine.PublishCommand) (engine.PublishOutcome, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cmd.OperationID != "" {
		if out, ok := h.applied[cmd.OperationID]; ok {
			return out, nil
		}
	}
	live := h.live[cmd.Target]
	draft := h.draft[cmd.Target]
	diff, err := selfTestDiff(cmd.Target, live, draft)
	if err != nil {
		return engine.PublishOutcome{}, err
	}
	merged := cloneStringMap(live)
	for k, v := range draft {
		merged[k] = v
	}
	h.live[cmd.Target] = merged
	h.revSeq++
	outcome := engine.PublishOutcome{RevisionID: fmt.Sprintf("live-rev-%d", h.revSeq), SnapshotDigest: selfTestDigest(merged), Diff: diff}
	if cmd.OperationID != "" {
		h.applied[cmd.OperationID] = outcome
	}
	return outcome, nil
}

func (h *selfTestHost) RestoreLive(ctx context.Context, cmd engine.RestoreCommand) (engine.RestoreOutcome, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	revision, ok := h.revisions.RevisionByID(cmd.Target.ResourceKind, cmd.Target.ResourceID, cmd.RevisionID)
	if !ok {
		return engine.RestoreOutcome{}, fmt.Errorf("selfTestHost: revision %q not found", cmd.RevisionID)
	}
	attrs, err := lifecycle.DecodeSnapshot[map[string]string](revision)
	if err != nil {
		return engine.RestoreOutcome{}, err
	}
	live := h.live[cmd.Target]
	diff, err := selfTestDiff(cmd.Target, live, attrs)
	if err != nil {
		return engine.RestoreOutcome{}, err
	}

	// Companion round trip, mirroring HostLifecycle.RestoreLive's doc
	// comment: mint the companion snapshot for the fresh pre-restore restore
	// point Engine just handed us, then restore whatever companion snapshot
	// (if any) was captured under the revision we're restoring to.
	if cmd.RestorePointID != "" {
		if h.companionByRev[cmd.Target] == nil {
			h.companionByRev[cmd.Target] = map[string]string{}
		}
		h.companionByRev[cmd.Target][cmd.RestorePointID] = h.companionLive[cmd.Target]
	}
	if companion, ok := h.companionByRev[cmd.Target][cmd.RevisionID]; ok {
		h.companionLive[cmd.Target] = companion
	} else {
		h.companionLive[cmd.Target] = ""
	}

	h.live[cmd.Target] = cloneStringMap(attrs)
	h.revSeq++
	newRevID := fmt.Sprintf("live-rev-%d", h.revSeq)
	if h.revisions != nil {
		if _, err := h.revisions.SaveRevision(cmsstore.RevisionInput{
			ResourceKind: cmd.Target.ResourceKind, ResourceID: cmd.Target.ResourceID,
			Action: cmsstore.ActionRestoreApplied, Summary: "self-test restore", Snapshot: attrs,
		}); err != nil {
			return engine.RestoreOutcome{}, err
		}
	}
	return engine.RestoreOutcome{RevisionID: newRevID, Diff: diff}, nil
}

func (h *selfTestHost) OperationLog(ctx context.Context, ref engine.TargetRef, sinceRevision uint64) ([]authoring.OperationRecord, error) {
	return nil, nil
}

func (h *selfTestHost) PublishedOperationRevision(ctx context.Context, ref engine.TargetRef) (uint64, error) {
	return 0, nil
}

func selfTestDiff(ref engine.TargetRef, from, to map[string]string) (lifecycle.RevisionDiff, error) {
	fromRev, err := lifecycle.NewRevision(lifecycle.RevisionInput{ResourceKind: ref.ResourceKind, ResourceID: ref.ResourceID, Action: "conformance.selftest", Snapshot: from}, time.Time{})
	if err != nil {
		return lifecycle.RevisionDiff{}, err
	}
	toRev, err := lifecycle.NewRevision(lifecycle.RevisionInput{ResourceKind: ref.ResourceKind, ResourceID: ref.ResourceID, Action: "conformance.selftest", Snapshot: to}, time.Time{})
	if err != nil {
		return lifecycle.RevisionDiff{}, err
	}
	return lifecycle.DiffRevisions(fromRev, toRev)
}

func selfTestDigest(attrs map[string]string) string {
	data, _ := json.Marshal(attrs)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestSelfHostLifecycleConformance(t *testing.T) {
	conformance.RunHostLifecycleConformance(t, func(t *testing.T) conformance.LifecycleFixture {
		store, closeStore, err := lifecyclesql.Open(":memory:")
		if err != nil {
			t.Fatalf("open lifecycle sqlstore: %v", err)
		}
		host := newSelfTestHost(store)
		target := engine.TargetRef{ResourceKind: cmsstore.ResourceKindSiteSettings, ResourceID: "conformance-selftest"}
		idSeq := 0
		return conformance.LifecycleFixture{
			Engine: &engine.Engine{
				Revisions: store,
				Ledger:    store,
				Host:      host,
				NewID: func(prefix string) string {
					idSeq++
					return fmt.Sprintf("%s_%d", prefix, idSeq)
				},
			},
			Target:       target,
			SeedLive:     func(attrs map[string]string) { host.live[target] = cloneStringMap(attrs) },
			SeedDraft:    func(attrs map[string]string) { host.draft[target] = cloneStringMap(attrs) },
			SeedBlockers: func(blockers ...engine.Blocker) { host.blockers[target] = append([]engine.Blocker(nil), blockers...) },
			CurrentLive:  func() map[string]string { return cloneStringMap(host.live[target]) },
			Companion: &conformance.CompanionFixture{
				Set:     func(value string) { host.companionLive[target] = value },
				Current: func() string { return host.companionLive[target] },
				SnapshotFor: func(revisionOrRestorePointID string) (string, bool) {
					v, ok := host.companionByRev[target][revisionOrRestorePointID]
					return v, ok
				},
			},
			Cleanup: func() { _ = closeStore() },
		}
	})
}
