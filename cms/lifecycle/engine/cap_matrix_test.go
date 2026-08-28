package engine

import (
	"context"
	"errors"
	"testing"
)

// TestCapabilityMatrixDeniesWithoutMatchingCapability is the
// HANDOFF-PUBLISH-MASTERING-07 slice S7 verification bullet: "cap matrix
// table test (publish/promote/restore each deny without the matching
// capability)". Each row grants every OTHER capability so the only possible
// reason for denial is the one capability under test -- proving the gate is
// exactly the one requireCap check the spec's normative sequence names, not
// an accidental all-or-nothing check.
func TestCapabilityMatrixDeniesWithoutMatchingCapability(t *testing.T) {
	full := CapabilitySet{CanPublish: true, CanRestore: true, CanPromote: true, CanConfigureProduction: true}

	cases := []struct {
		name string
		caps CapabilitySet
		run  func(t *testing.T, e *Engine, host *fakeHost, ref TargetRef, actor Actor) error
	}{
		{
			name: "publish denies without CanPublish",
			caps: withoutCap(full, "publish"),
			run: func(t *testing.T, e *Engine, host *fakeHost, ref TargetRef, actor Actor) error {
				host.setLive(ref, map[string]string{"title": "Original"})
				host.setDraft(ref, map[string]string{"title": "Draft"})
				_, err := e.Publish(context.Background(), PublishCommand{Target: ref, Actor: actor, OperationID: "op-publish"})
				return err
			},
		},
		{
			name: "restore denies without CanRestore",
			caps: withoutCap(full, "restore"),
			run: func(t *testing.T, e *Engine, host *fakeHost, ref TargetRef, actor Actor) error {
				_, err := e.Restore(context.Background(), RestoreCommand{Target: ref, Actor: actor, RevisionID: "does-not-matter", OperationID: "op-restore"})
				return err
			},
		},
		{
			name: "promote denies without CanPromote",
			caps: withoutCap(full, "promote"),
			run: func(t *testing.T, e *Engine, host *fakeHost, ref TargetRef, actor Actor) error {
				_, err := e.Promote(context.Background(), PromoteCommand{
					Target:      ref,
					Actor:       actor,
					Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true},
					OperationID: "op-promote",
				})
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, host := newTestEngine(t)
			ref := testTarget()
			actor := Actor{ID: "actor-under-test", Label: "Actor Under Test", Caps: tc.caps}
			err := tc.run(t, e, host, ref, actor)
			if !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("%s: expected ErrUnauthorized with every OTHER capability granted, got %v (caps=%#v)", tc.name, err, tc.caps)
			}
			if host.applyPublishCalls != 0 || host.restoreLiveCalls != 0 || host.snapshotLiveCalls != 0 {
				t.Fatalf("%s: expected zero host interaction when the matching capability is missing, got apply=%d restore=%d snapshot=%d",
					tc.name, host.applyPublishCalls, host.restoreLiveCalls, host.snapshotLiveCalls)
			}
		})
	}
}

// TestCapabilityMatrixSucceedsWithMatchingCapability is the positive half of
// the matrix: each transition succeeds when its own capability IS granted
// (with the others granted too, isolating the assertion to "this capability
// governs this transition").
func TestCapabilityMatrixSucceedsWithMatchingCapability(t *testing.T) {
	full := CapabilitySet{CanPublish: true, CanRestore: true, CanPromote: true, CanConfigureProduction: true}

	t.Run("publish succeeds with CanPublish", func(t *testing.T) {
		e, host := newTestEngine(t)
		ref := testTarget()
		host.setLive(ref, map[string]string{"title": "Original"})
		host.setDraft(ref, map[string]string{"title": "Draft"})
		actor := Actor{ID: "actor-1", Caps: full}
		if _, err := e.Publish(context.Background(), PublishCommand{Target: ref, Actor: actor, OperationID: "op-publish-ok"}); err != nil {
			t.Fatalf("expected publish to succeed with CanPublish granted, got %v", err)
		}
	})

	t.Run("restore succeeds with CanRestore", func(t *testing.T) {
		e, host := newTestEngine(t)
		ref := testTarget()
		host.setLive(ref, map[string]string{"title": "Original"})
		host.setDraft(ref, map[string]string{"title": "Draft"})
		actor := Actor{ID: "actor-1", Caps: full}
		published, err := e.Publish(context.Background(), PublishCommand{Target: ref, Actor: actor, OperationID: "op-publish-seed"})
		if err != nil {
			t.Fatalf("seed publish: %v", err)
		}
		if _, err := e.Restore(context.Background(), RestoreCommand{Target: ref, Actor: actor, RevisionID: published.RestorePointID, OperationID: "op-restore-ok"}); err != nil {
			t.Fatalf("expected restore to succeed with CanRestore granted, got %v", err)
		}
	})

	t.Run("promote succeeds with CanPromote", func(t *testing.T) {
		e, host := newTestEngine(t)
		ref := testTarget()
		host.setLive(ref, map[string]string{"title": "Original"})
		host.setDraft(ref, map[string]string{"title": "Draft"})
		actor := Actor{ID: "actor-1", Caps: full}
		if _, err := e.Publish(context.Background(), PublishCommand{Target: ref, Actor: actor, OperationID: "op-publish-seed"}); err != nil {
			t.Fatalf("seed publish: %v", err)
		}
		if _, err := e.Promote(context.Background(), PromoteCommand{
			Target:      ref,
			Actor:       actor,
			Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true},
			OperationID: "op-promote-ok",
		}); err != nil {
			t.Fatalf("expected promote to succeed with CanPromote granted, got %v", err)
		}
	})
}

// withoutCap returns a copy of caps with exactly one capability cleared,
// identified by name ("publish", "restore", "promote"). Keeping this as a
// small named helper (rather than inline struct literals per case) is what
// makes each cap_matrix_test.go case above legible as "everything else
// granted, this one capability missing".
func withoutCap(caps CapabilitySet, name string) CapabilitySet {
	switch name {
	case "publish":
		caps.CanPublish = false
	case "restore":
		caps.CanRestore = false
	case "promote":
		caps.CanPromote = false
	}
	return caps
}
