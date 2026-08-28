package conformance

import (
	"testing"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

// DraftProjector is the collaboration ledger's draft-projection seam: it
// exposes a target's current draft value (for the ledger's Seed step),
// applies one already-accepted canonical ledger record to draft-only host
// state, dry-run validates a not-yet-accepted request against the same
// apply-time preconditions, and durably quarantines an accepted operation
// that turned out to be unprojectable.
//
// This is not (yet) declared inside cms/studio/collab itself -- see doc.go's
// package comment. It is declared here, once, as the canonical Go shape both
// current host adapters (Noni's internal/studiohost.DraftProjector,
// Pajaritos' *site.DurableAuthoringStore) already independently implement
// identically, so a real host adapter type satisfies this interface with no
// wrapper: `var _ conformance.DraftProjector = (*mypkg.Store)(nil)`.
type DraftProjector interface {
	// StudioTargetValue returns target's current draft value. A target that
	// has never been written reports OperationValue{Present:false} (or, for a
	// host whose content-field model always has a value, Present:true with
	// whatever default it holds) -- callers must not assume absence, they
	// must read it.
	StudioTargetValue(authoring.OperationTarget) (authoring.OperationValue, error)
	// ProjectStudioOperation applies one already-accepted canonical ledger
	// record to draft-only state. It MUST be idempotent by record.ID: a
	// replay of the exact same record is a no-op returning nil error; the
	// same ID with DIFFERENT content (a different Kind/Target/Before/After)
	// must return a non-nil error rather than silently re-applying. It MUST
	// reject (non-nil error, no mutation) a record whose Before does not
	// match the target's actual current value -- this is the divergence
	// class a caller (e.g. cms/studio/collab-consuming ProjectingService)
	// treats as quarantine-worthy rather than retryable.
	ProjectStudioOperation(authoring.OperationRecord) error
	// CanApplyStudioOperation reports whether a not-yet-accepted request
	// would satisfy ProjectStudioOperation's apply-time preconditions,
	// without mutating any state. It shares its validation with
	// ProjectStudioOperation so pre-accept validation and post-accept
	// projection can never diverge. Per the real-host precedent, it is
	// primarily meaningful for requests that carry a concrete Target and
	// Value directly (set-field, set-style, reset-style, and this package's
	// generalization to the other target-shaped kinds); undo/redo resolve
	// their concrete target/value from ledger history and have nothing
	// concrete to dry-run.
	CanApplyStudioOperation(authoring.OperationRequest) error
	// QuarantineStudioOperation durably records an accepted operation that
	// could never be projected (most commonly a before-value divergence from
	// a concurrent direct edit), for audit, WITHOUT mutating draft state. It
	// must not silently apply record's After value.
	QuarantineStudioOperation(record authoring.OperationRecord, reason string) error
}

// DraftProjectorFixture is what a host's factory builds for one conformance
// scenario.
type DraftProjectorFixture struct {
	// Projector is the host's real DraftProjector adapter under test.
	Projector DraftProjector
	// Cleanup releases anything the factory opened. Optional.
	Cleanup func()
}

// DraftProjectorFactory builds one fresh DraftProjectorFixture.
// RunDraftProjectorConformance calls it once per t.Run subtest.
type DraftProjectorFactory func(t *testing.T) DraftProjectorFixture

// TargetCase is one family fixture the DraftProjector conformance suite
// drives through Project/CanApply/Quarantine. Target must already address an
// identity the host's real projector will accept (a fake/in-memory projector
// can accept any Target; a real host's projector typically requires the
// definition/instance/flow/interaction Target addresses to already exist --
// the factory is responsible for seeding that identity before returning the
// fixture, e.g. creating the shared-component definition Target.ComponentKey
// names).
//
// The suite derives which OperationKind(s) to drive from Target's own shape
// via authoring.SetKindForTarget/authoring.ClearsTarget -- the exact same
// resolution cms/studio/collab/sqlstore and every DraftProjector
// implementation already use -- so one TargetCase per family exercises both
// members of a paired family (e.g. set-style + reset-style) or the single
// member of a singleton family (e.g. set-shared-field).
type TargetCase struct {
	// Name labels the subtest (e.g. "style", "shared-field", "interaction").
	Name string
	// Target is a valid, host-recognized addressing for this family.
	Target authoring.OperationTarget
	// Value is the wire value the family's "set" member's After carries.
	Value string
}

// CoreTargetCases returns TargetCase fixtures for the 3 core kinds a
// DraftProjector implementation always supports: a plain content field
// (set-field) and a style property (set-style/reset-style, a paired family).
// authoring.OperationUndo/OperationRedo are intentionally not represented
// here -- see RunOperationProtocolConformance's doc comment for why they
// never reach a DraftProjector directly.
func CoreTargetCases() []TargetCase {
	return []TargetCase{
		{Name: "content-field", Target: authoring.OperationTarget{Route: "/", Field: "conformance.title"}, Value: "Conformance Title"},
		{Name: "style", Target: authoring.OperationTarget{Route: "/", ComponentKey: "hero", Property: "color"}, Value: "#111111"},
	}
}

// ExtendedTargetCases returns TargetCase fixtures for the 9 instance/
// interaction/flow durable kinds added in HANDOFF-12 (shared-field,
// override-instance, attachment [paired], interaction [paired], flow-field
// [paired], flow-action). A host that has not yet adopted HANDOFF-12's
// durable-kind ledger wiring for these domains (see the host-adoption TODO
// list in that slice's report) should not pass these to
// RunDraftProjectorConformance yet -- pass only CoreTargetCases() until it
// has. Placeholder ComponentKey/ControlKey/PageID identities here are
// generic; a real host's factory must override Target with identities that
// actually exist in its own fixtures (a permissive in-memory fake accepts any
// identity; a real store typically does not).
func ExtendedTargetCases() []TargetCase {
	return []TargetCase{
		{Name: "shared-field", Target: authoring.OperationTarget{Route: "/", Field: authoring.FieldInstancesSharedField, ComponentKey: "conformance-component", ControlKey: "headline"}, Value: "Shared Value"},
		{Name: "override-instance", Target: authoring.OperationTarget{Route: "/", Field: authoring.FieldInstancesOverride, PageID: "conformance-page", ComponentKey: "conformance-component", ControlKey: "headline"}, Value: "Override Value"},
		{Name: "attachment", Target: authoring.OperationTarget{Route: "/", Field: authoring.FieldInstancesAttachment, PageID: "conformance-page", ComponentKey: "conformance-component"}, Value: ""},
		{Name: "interaction", Target: authoring.OperationTarget{Route: "/", Field: authoring.FieldInteractionsEntry, PageID: "conformance-page", ComponentKey: "conformance-component", ControlKey: "reveal"}, Value: authoring.EncodeInteractionSettings(defaultConformanceInteraction())},
		{Name: "flow-field", Target: authoring.OperationTarget{Route: "/", Field: authoring.FieldFlowsField, PageID: "conformance-flow", ComponentKey: "submit", ControlKey: "email"}, Value: authoring.EncodeFlowFieldSettings(defaultConformanceFlowField())},
		{Name: "flow-action", Target: authoring.OperationTarget{Route: "/", Field: authoring.FieldFlowsAction, PageID: "conformance-flow", ComponentKey: "submit"}, Value: authoring.EncodeFlowActionSettings("Submit", "conformance.handler")},
	}
}

// defaultConformanceInteraction is a minimal valid core.Interaction for
// building an "interaction" TargetCase's Value. EncodeInteractionSettings
// drops Key/Target (the operation's own Target already carries that
// identity), so the zero values there are fine.
func defaultConformanceInteraction() core.Interaction {
	return core.Interaction{Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFade, DurationMS: 400, Once: true}
}

// defaultConformanceFlowField is a minimal valid core.FlowField for building
// a "flow-field" TargetCase's Value. EncodeFlowFieldSettings drops Name (the
// operation's own Target.ControlKey already carries it).
func defaultConformanceFlowField() core.FlowField {
	return core.FlowField{Name: "email", Label: "Email", Kind: core.ControlText, Required: true}
}

// RunDraftProjectorConformance proves a DraftProjector adapter satisfies the
// contract cms/studio/collab's ProjectingService convention relies on (the
// "P1" outbox-wedge fix both current host adapters implement):
//
//   - ProjectStudioOperation applies an accepted record's After value,
//     readable back via StudioTargetValue.
//   - ProjectStudioOperation is idempotent by record.ID (an exact replay is a
//     silent no-op) and rejects (non-nil error) an ID reused with different
//     content.
//   - ProjectStudioOperation rejects (non-nil error, no mutation) a record
//     whose Before diverges from the target's actual current value.
//   - For a paired family (Target resolves ClearsTarget's sibling kind),
//     projecting the "clear" member's record removes the value
//     (StudioTargetValue reports Present:false afterward).
//   - CanApplyStudioOperation dry-runs a request without mutating draft
//     state, and does not itself commit anything even when it reports the
//     request would succeed.
//   - QuarantineStudioOperation durably records a record + reason without
//     mutating the target's draft value.
//
// cases must be non-empty; pass CoreTargetCases() (always supported) plus
// ExtendedTargetCases() (only once the host has adopted HANDOFF-12's 9
// durable kinds for its own draft store -- see ExtendedTargetCases' doc
// comment).
func RunDraftProjectorConformance(t *testing.T, factory DraftProjectorFactory, cases []TargetCase) {
	t.Helper()
	if len(cases) == 0 {
		t.Fatal("RunDraftProjectorConformance: cases must be non-empty (see CoreTargetCases/ExtendedTargetCases)")
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			fx := factory(t)
			if fx.Cleanup != nil {
				t.Cleanup(fx.Cleanup)
			}
			runTargetCaseConformance(t, fx.Projector, tc)
		})
	}

	t.Run("QuarantineDoesNotMutateDraft", func(t *testing.T) {
		fx := factory(t)
		if fx.Cleanup != nil {
			t.Cleanup(fx.Cleanup)
		}
		tc := cases[0]
		before, err := fx.Projector.StudioTargetValue(tc.Target)
		if err != nil {
			t.Fatalf("StudioTargetValue (baseline): %v", err)
		}
		record := authoring.OperationRecord{
			ID: "conformance-quarantine-1", ActorID: "conformance-actor",
			Kind: authoring.SetKindForTarget(tc.Target, true), Target: tc.Target,
			Before: before, After: authoring.PresentValue(tc.Value),
			DocumentRevision: 1, TargetHead: "conformance-head-1",
		}.Normalize()
		if err := fx.Projector.QuarantineStudioOperation(record, "conformance: forced quarantine"); err != nil {
			t.Fatalf("QuarantineStudioOperation: %v", err)
		}
		after, err := fx.Projector.StudioTargetValue(tc.Target)
		if err != nil {
			t.Fatalf("StudioTargetValue (post-quarantine): %v", err)
		}
		if after != before {
			t.Fatalf("expected quarantining a record to leave the draft value untouched, before=%#v after=%#v", before, after)
		}
	})
}

func runTargetCaseConformance(t *testing.T, projector DraftProjector, tc TargetCase) {
	t.Helper()

	setKind := authoring.SetKindForTarget(tc.Target, true)
	clearKind := authoring.SetKindForTarget(tc.Target, false)
	paired := clearKind != setKind

	initial, err := projector.StudioTargetValue(tc.Target)
	if err != nil {
		t.Fatalf("StudioTargetValue (initial): %v", err)
	}

	setRecord := authoring.OperationRecord{
		ID: "conformance-op-1", ActorID: "conformance-actor",
		Kind: setKind, Target: tc.Target,
		Before: initial, After: authoring.PresentValue(tc.Value),
		DocumentRevision: 1, TargetHead: "conformance-head-1",
	}.Normalize()

	if err := projector.ProjectStudioOperation(setRecord); err != nil {
		t.Fatalf("ProjectStudioOperation (set): %v", err)
	}
	got, err := projector.StudioTargetValue(tc.Target)
	if err != nil {
		t.Fatalf("StudioTargetValue (after set): %v", err)
	}
	if !got.Present || got.Value != tc.Value {
		t.Fatalf("expected the projected After value to be readable back, got %#v", got)
	}

	// Idempotent replay: the exact same record again must be a silent no-op.
	if err := projector.ProjectStudioOperation(setRecord); err != nil {
		t.Fatalf("ProjectStudioOperation (idempotent replay): %v", err)
	}
	if got2, err := projector.StudioTargetValue(tc.Target); err != nil || got2 != got {
		t.Fatalf("expected the idempotent replay to leave the value unchanged, got %#v err=%v", got2, err)
	}

	// Same ID, different content: must be rejected, not silently re-applied.
	divergent := setRecord
	divergent.After = authoring.PresentValue(tc.Value + "-different")
	if err := projector.ProjectStudioOperation(divergent); err == nil {
		t.Fatal("expected reusing an operation id with different content to fail")
	}
	if got3, err := projector.StudioTargetValue(tc.Target); err != nil || got3 != got {
		t.Fatalf("expected the rejected id-reuse attempt to leave the value unchanged, got %#v err=%v", got3, err)
	}

	// Before-value divergence: a record whose Before does not match the
	// target's actual current value must be rejected without mutating state.
	staleBefore := authoring.OperationRecord{
		ID: "conformance-op-stale", ActorID: "conformance-actor",
		Kind: setKind, Target: tc.Target,
		Before: authoring.OperationValue{Present: false}, After: authoring.PresentValue(tc.Value + "-stale"),
		DocumentRevision: 2, TargetHead: "conformance-head-stale",
	}.Normalize()
	if got.Present {
		// got is Present, so an explicitly-absent Before is guaranteed to
		// diverge from it; this is the scenario ProjectingService quarantines.
		if err := projector.ProjectStudioOperation(staleBefore); err == nil {
			t.Fatal("expected a before-value divergence to be rejected")
		}
		if got4, err := projector.StudioTargetValue(tc.Target); err != nil || got4 != got {
			t.Fatalf("expected a rejected divergent projection to leave the value unchanged, got %#v err=%v", got4, err)
		}
	}

	// CanApplyStudioOperation: dry-run a fresh request shaped like the set
	// operation above and confirm it does not itself mutate state.
	req := authoring.OperationRequest{ID: "conformance-canapply-1", Kind: setKind, Target: tc.Target, Value: tc.Value}.Normalize()
	beforeCanApply, err := projector.StudioTargetValue(tc.Target)
	if err != nil {
		t.Fatalf("StudioTargetValue (before CanApply): %v", err)
	}
	_ = projector.CanApplyStudioOperation(req) // result (accept/reject) is host-specific; only the no-mutation invariant is generically assertable here.
	if afterCanApply, err := projector.StudioTargetValue(tc.Target); err != nil || afterCanApply != beforeCanApply {
		t.Fatalf("expected CanApplyStudioOperation to be a pure dry run, before=%#v after=%#v err=%v", beforeCanApply, afterCanApply, err)
	}

	if !paired {
		return
	}

	// Paired family: projecting the "clear" member must remove the value.
	current, err := projector.StudioTargetValue(tc.Target)
	if err != nil {
		t.Fatalf("StudioTargetValue (before clear): %v", err)
	}
	clearRecord := authoring.OperationRecord{
		ID: "conformance-op-clear", ActorID: "conformance-actor",
		Kind: clearKind, Target: tc.Target,
		Before: current, After: authoring.OperationValue{Present: false},
		DocumentRevision: 3, TargetHead: "conformance-head-clear",
	}.Normalize()
	if err := projector.ProjectStudioOperation(clearRecord); err != nil {
		t.Fatalf("ProjectStudioOperation (clear): %v", err)
	}
	if cleared, err := projector.StudioTargetValue(tc.Target); err != nil || cleared.Present {
		t.Fatalf("expected the clear member to leave the target absent, got %#v err=%v", cleared, err)
	}
}
