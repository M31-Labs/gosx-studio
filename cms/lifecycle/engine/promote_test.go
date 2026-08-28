package engine

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	lifecyclesql "m31labs.dev/gosx-studio/cms/lifecycle/sqlstore"
)

// TestPromoteDestructiveMigrationIsBlocked is the HANDOFF-PUBLISH-MASTERING-07
// slice S8 verification bullet: "destructive/unknown migration -> hard STOP
// (ErrDestructiveMigration), actually triggered by a test, not just
// declared". A single Destructive step anywhere in the plan refuses the
// promotion and records NOTHING -- no promote.recorded audit event, no
// PromotionIntent -- so a bad classification can never leak a partially
// recorded intent.
func TestPromoteDestructiveMigrationIsBlocked(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	_, err := e.Promote(ctx, PromoteCommand{
		Target:      ref,
		Actor:       actor,
		Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true},
		OperationID: "promote-destructive",
		Migration: MigrationPlan{Steps: []MigrationStep{
			{Description: "drop legacy column", Class: MigrationDestructive},
		}},
	})
	if !errors.Is(err, ErrDestructiveMigration) {
		t.Fatalf("expected ErrDestructiveMigration, got %v", err)
	}

	events, listErr := e.Ledger.ListAuditEvents(ctx, lifecycle.LedgerFilter{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       ActionPromoteRecorded,
	})
	if listErr != nil {
		t.Fatalf("list audit events: %v", listErr)
	}
	if len(events) != 0 {
		t.Fatalf("expected zero promote.recorded audit events for a refused destructive migration, got %#v", events)
	}
}

// TestPromoteUnknownMigrationIsBlocked proves an UNCLASSIFIED step fails
// closed exactly like a known-destructive one (spec §8: "Any future host
// change that would drop/rewrite persisted fields is out of scope and a
// STOP under this descriptor") -- Promote never treats "I don't know" as
// safe by default.
func TestPromoteUnknownMigrationIsBlocked(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	actor := publisherActor("actor-1")
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	_, err := e.Promote(ctx, PromoteCommand{
		Target:      ref,
		Actor:       actor,
		Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true},
		OperationID: "promote-unknown",
		Migration: MigrationPlan{Steps: []MigrationStep{
			{Description: "unreviewed schema change from a third-party plugin"},
		}},
	})
	if !errors.Is(err, ErrDestructiveMigration) {
		t.Fatalf("expected ErrDestructiveMigration for an unclassified migration step, got %v", err)
	}
}

// TestPromoteSafeOrAbsentMigrationProceeds is the mirror positive case: an
// empty plan (no migration at all -- the common case) and a plan whose every
// step is explicitly Safe both let Promote proceed, so the STOP is scoped to
// genuinely risky/unknown steps, never to "no migration was declared".
func TestPromoteSafeOrAbsentMigrationProceeds(t *testing.T) {
	for _, tc := range []struct {
		name      string
		migration MigrationPlan
	}{
		{name: "no migration declared", migration: MigrationPlan{}},
		{name: "every step explicitly safe", migration: MigrationPlan{Steps: []MigrationStep{
			{Description: "add nullable column", Class: MigrationSafe},
			{Description: "add index", Class: MigrationSafe},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, host := newTestEngine(t)
			ctx := context.Background()
			ref := testTarget()
			actor := publisherActor("actor-1")
			host.setLive(ref, map[string]string{"title": "Original"})
			host.setDraft(ref, map[string]string{"title": "Draft V1"})
			if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
				t.Fatalf("Publish: %v", err)
			}
			intent, err := e.Promote(ctx, PromoteCommand{
				Target:      ref,
				Actor:       actor,
				Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true},
				OperationID: "promote-safe",
				Migration:   tc.migration,
			})
			if err != nil {
				t.Fatalf("expected promote to proceed, got %v", err)
			}
			if intent.SeedData {
				t.Fatal("expected SeedData to always be false")
			}
		})
	}
}

// TestPromoteProductionGateRequiresBothConfiguredAndCapability exercises all
// four combinations of Environment.Configured x Actor.Caps.CanConfigureProduction
// for a production-shaped target: only the case where both are true may
// proceed (spec §7: "production gated by Configured && CanConfigureProduction").
func TestPromoteProductionGateRequiresBothConfiguredAndCapability(t *testing.T) {
	cases := []struct {
		name       string
		configured bool
		canConfig  bool
		wantErr    bool
	}{
		{name: "neither configured nor capable", configured: false, canConfig: false, wantErr: true},
		{name: "configured but not capable", configured: true, canConfig: false, wantErr: true},
		{name: "capable but not configured", configured: false, canConfig: true, wantErr: true},
		{name: "configured and capable", configured: true, canConfig: true, wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, host := newTestEngine(t)
			ctx := context.Background()
			ref := testTarget()
			actor := Actor{ID: "actor-1", Caps: CapabilitySet{CanPublish: true, CanPromote: true, CanConfigureProduction: tc.canConfig}}
			host.setLive(ref, map[string]string{"title": "Original"})
			host.setDraft(ref, map[string]string{"title": "Draft V1"})
			if _, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
				t.Fatalf("Publish: %v", err)
			}

			_, err := e.Promote(ctx, PromoteCommand{
				Target:      ref,
				Actor:       actor,
				Environment: Environment{Key: "production", Kind: EnvProduction, Configured: tc.configured},
				OperationID: "promote-prod-" + tc.name,
			})
			if tc.wantErr {
				if !errors.Is(err, ErrProductionNotConfigured) {
					t.Fatalf("expected ErrProductionNotConfigured, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected promote to succeed when configured+capable, got %v", err)
			}
		})
	}
}

// shellHookHost embeds a real HostLifecycle and additionally exposes a
// shell/exec-shaped method that is NOT part of the HostLifecycle interface.
// Because Engine.Host is statically typed as HostLifecycle, nothing in the
// engine package can call Exec through that field -- this test makes that
// structural guarantee an executable assertion rather than an unenforced
// comment: if Engine.Promote (or any future engine code) ever gained a way
// to invoke a shell/exec hook, this counter would move.
type shellHookHost struct {
	HostLifecycle
	execCalls int
}

func (h *shellHookHost) Exec(cmd string) error {
	h.execCalls++
	return nil
}

// TestPromoteNeverInvokesHostOrShellHooks is the S8 verification bullet:
// "promote never invokes shell/k8s (no exec calls -- test via a fake host
// asserting no such hook exists on the interface)". Two independent proofs:
//  1. Promote succeeds with Engine.Host == nil entirely -- it never
//     dereferences Host at all, so there is no way for it to reach a
//     shell/k8s call through the host seam.
//  2. A decorated fake host exposing an Exec method never has that method
//     invoked by Promote (it structurally cannot be, since Engine.Host's
//     static type is HostLifecycle, which has no such method).
func TestPromoteNeverInvokesHostOrShellHooks(t *testing.T) {
	t.Run("Promote succeeds with a nil Host", func(t *testing.T) {
		ledger, closeLedger, err := lifecyclesql.Open(":memory:")
		if err != nil {
			t.Fatalf("open ledger: %v", err)
		}
		t.Cleanup(func() { _ = closeLedger() })
		e := &Engine{Revisions: ledger, Ledger: ledger, Host: nil}
		ref := testTarget()
		actor := publisherActor("actor-1")

		// Promote only reads the ledger's LatestPublishDecision -- seed one
		// directly so a nil Host never needs to be touched to reach it.
		if _, err := ledger.SavePublishDecision(context.Background(), lifecycle.PublishDecisionInput{
			ResourceKind: ref.ResourceKind,
			ResourceID:   ref.ResourceID,
			RevisionID:   "seed-revision",
			Status:       lifecycle.DecisionApproved,
			ActorID:      "actor-1",
		}); err != nil {
			t.Fatalf("seed publish decision: %v", err)
		}

		if _, err := e.Promote(context.Background(), PromoteCommand{
			Target:      ref,
			Actor:       actor,
			Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true},
			OperationID: "promote-nil-host",
		}); err != nil {
			t.Fatalf("expected Promote to succeed without ever touching Host, got %v", err)
		}
	})

	t.Run("a decorated fake host's Exec hook is never invoked", func(t *testing.T) {
		e, host := newTestEngine(t)
		ref := testTarget()
		actor := publisherActor("actor-1")
		host.setLive(ref, map[string]string{"title": "Original"})
		host.setDraft(ref, map[string]string{"title": "Draft V1"})
		if _, err := e.Publish(context.Background(), PublishCommand{Target: ref, Actor: actor, OperationID: "op-1"}); err != nil {
			t.Fatalf("Publish: %v", err)
		}

		shellHost := &shellHookHost{HostLifecycle: host}
		e.Host = shellHost
		if _, err := e.Promote(context.Background(), PromoteCommand{
			Target:      ref,
			Actor:       actor,
			Environment: Environment{Key: "staging", Kind: EnvStaging, Configured: true},
			OperationID: "promote-exec-hook",
		}); err != nil {
			t.Fatalf("Promote: %v", err)
		}
		if shellHost.execCalls != 0 {
			t.Fatalf("expected Promote to never invoke the host's Exec hook, got %d calls", shellHost.execCalls)
		}
	})
}

// TestHostLifecycleInterfaceHasNoExecOrShellHook is a structural, grep-proof
// guard on the HostLifecycle interface itself: it reflects over every method
// HostLifecycle declares and fails if any method name suggests a shell/exec/
// k8s/deploy hook. This is the "no such hook exists on the interface" half
// of the S8 verification bullet -- if HostLifecycle ever grew such a method,
// this test (not just a doc comment) would catch it, independent of whether
// Promote happens to call it.
func TestHostLifecycleInterfaceHasNoExecOrShellHook(t *testing.T) {
	typ := reflect.TypeOf((*HostLifecycle)(nil)).Elem()
	banned := []string{"exec", "shell", "kubectl", "k8s", "deploy", "rollout", "command"}
	for i := 0; i < typ.NumMethod(); i++ {
		name := strings.ToLower(typ.Method(i).Name)
		for _, b := range banned {
			if strings.Contains(name, b) {
				t.Fatalf("HostLifecycle must not expose a shell/exec/k8s hook, found method %q (matches banned substring %q)", typ.Method(i).Name, b)
			}
		}
	}
}
