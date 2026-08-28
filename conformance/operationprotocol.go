package conformance

import (
	"encoding/json"
	"testing"

	"m31labs.dev/gosx-studio/authoring"
)

// operationKindFixture is one host-independent, always-valid
// authoring.OperationRequest fixture for one authoring.OperationKind.
type operationKindFixture struct {
	kind   authoring.OperationKind
	target authoring.OperationTarget
	value  string
	// historyOperationID is only set for undo/redo, which address a prior
	// operation id instead of a concrete target/value.
	historyOperationID string
	// present is target's present/absent hint for SetKindForTarget -- true
	// selects the "set" member of a paired family (or a singleton family's
	// only member); false selects the "clear" member of a paired family.
	// Meaningless (left false) for undo/redo, which SetKindForTarget never
	// classifies this way.
	present bool
}

// allOperationKindFixtures returns one fixture per one of the 14
// authoring.OperationKind values Studio defines: the 5 core kinds (set-field,
// set-style, reset-style, undo, redo) plus the 9 durable instance/
// interaction/flow kinds added in HANDOFF-12.
func allOperationKindFixtures() []operationKindFixture {
	coreField := authoring.OperationTarget{Route: "/", Field: "conformance.title"}
	coreStyle := authoring.OperationTarget{Route: "/", ComponentKey: "hero", Property: "color"}
	sharedField := authoring.OperationTarget{Route: "/", Field: authoring.FieldInstancesSharedField, ComponentKey: "conformance-component", ControlKey: "headline"}
	override := authoring.OperationTarget{Route: "/", Field: authoring.FieldInstancesOverride, PageID: "conformance-page", ComponentKey: "conformance-component", ControlKey: "headline"}
	attachment := authoring.OperationTarget{Route: "/", Field: authoring.FieldInstancesAttachment, PageID: "conformance-page", ComponentKey: "conformance-component"}
	interaction := authoring.OperationTarget{Route: "/", Field: authoring.FieldInteractionsEntry, PageID: "conformance-page", ComponentKey: "conformance-component", ControlKey: "reveal"}
	flowField := authoring.OperationTarget{Route: "/", Field: authoring.FieldFlowsField, PageID: "conformance-flow", ComponentKey: "submit", ControlKey: "email"}
	flowAction := authoring.OperationTarget{Route: "/", Field: authoring.FieldFlowsAction, PageID: "conformance-flow", ComponentKey: "submit"}

	return []operationKindFixture{
		{kind: authoring.OperationSetField, target: coreField, value: "Conformance Title", present: true},
		{kind: authoring.OperationSetStyle, target: coreStyle, value: "#111111", present: true},
		{kind: authoring.OperationResetStyle, target: coreStyle, value: "", present: false},
		{kind: authoring.OperationUndo, target: coreField, historyOperationID: "conformance-history-1"},
		{kind: authoring.OperationRedo, target: coreField, historyOperationID: "conformance-history-1"},
		{kind: authoring.OperationSetSharedField, target: sharedField, value: "Shared Value", present: true},
		{kind: authoring.OperationOverrideInstance, target: override, value: "Override Value", present: true},
		{kind: authoring.OperationDetachInstance, target: attachment, value: "", present: true},
		{kind: authoring.OperationRestoreInstance, target: attachment, value: "", present: false},
		{kind: authoring.OperationSetInteraction, target: interaction, value: authoring.EncodeInteractionSettings(defaultConformanceInteraction()), present: true},
		{kind: authoring.OperationRemoveInteraction, target: interaction, value: "", present: false},
		{kind: authoring.OperationSetFlowField, target: flowField, value: authoring.EncodeFlowFieldSettings(defaultConformanceFlowField()), present: true},
		{kind: authoring.OperationRemoveFlowField, target: flowField, value: "", present: false},
		{kind: authoring.OperationSetFlowAction, target: flowAction, value: authoring.EncodeFlowActionSettings("Submit", "conformance.handler"), present: true},
	}
}

// clearingKinds is the exact set authoring.ClearsTarget documents: the
// "clear" member of every paired set/reset family.
func clearingKinds() map[authoring.OperationKind]bool {
	return map[authoring.OperationKind]bool{
		authoring.OperationResetStyle:        true,
		authoring.OperationRestoreInstance:   true,
		authoring.OperationRemoveInteraction: true,
		authoring.OperationRemoveFlowField:   true,
	}
}

// RunOperationProtocolConformance proves the durable authoring operation
// protocol (authoring.OperationRequest / authoring.OperationRecord) is
// JSON-stable and internally consistent for all 14 authoring.OperationKind
// values Studio defines -- the 5 core kinds (set-field, set-style,
// reset-style, undo, redo) plus the 9 instance/interaction/flow kinds added
// in HANDOFF-12 (set-shared-field, override-instance, detach-instance,
// restore-instance, set-interaction, remove-interaction, set-flow-field,
// remove-flow-field, set-flow-action).
//
// It takes no factory: every OperationKind's Validate/Normalize/JSON/
// SetKindForTarget/ClearsTarget behavior lives entirely in Studio's own
// authoring package, independent of any host adapter. A host that wants to
// prove its OWN OperationRequest construction (e.g. a panel's client-side
// payload, or a persisted historical record) decodes correctly against this
// same protocol calls this from its own test with its own payload bytes
// instead of (or in addition to) this suite's generic fixtures.
//
// authoring.OperationUndo/OperationRedo are deliberately excluded from
// RunDraftProjectorConformance's TargetCase-driven scenarios (see that
// function's doc comment): the collaboration ledger (cms/studio/collab/
// sqlstore's resolveHistory) resolves an accepted undo/redo into a concrete
// target-shaped record (via authoring.OperationRecord.CanonicalRequest,
// itself built from SetKindForTarget) BEFORE a DraftProjector ever sees it --
// a DraftProjector implementation never receives a record whose Kind is
// literally "undo" or "redo". This suite still proves undo/redo's own
// Validate/JSON/history-addressing contract, since a host's ledger-adjacent
// code (or a persisted audit trail) may still construct or decode one
// directly.
func RunOperationProtocolConformance(t *testing.T) {
	t.Helper()
	clearing := clearingKinds()

	for _, fx := range allOperationKindFixtures() {
		fx := fx
		t.Run(string(fx.kind), func(t *testing.T) {
			req := authoring.OperationRequest{
				ID: "conformance-" + string(fx.kind), Kind: fx.kind, Target: fx.target,
				Value: fx.value, HistoryOperationID: fx.historyOperationID,
			}.Normalize()
			if err := req.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}

			// JSON round trip: marshal, unmarshal into a fresh value, and
			// confirm re-marshaling produces byte-identical output -- the
			// wire contract every host's persisted record must agree on.
			raw, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var decoded authoring.OperationRequest
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			rerraw, err := json.Marshal(decoded.Normalize())
			if err != nil {
				t.Fatalf("re-Marshal: %v", err)
			}
			if string(rerraw) != string(raw) {
				t.Fatalf("expected JSON round trip to be byte-stable, got %s then %s", raw, rerraw)
			}

			// Normalize is idempotent.
			twice := req.Normalize()
			again, err := json.Marshal(twice)
			if err != nil {
				t.Fatalf("Marshal (re-normalized): %v", err)
			}
			if string(again) != string(raw) {
				t.Fatalf("expected Normalize to be idempotent, got %s then %s", raw, again)
			}

			if fx.kind == authoring.OperationUndo || fx.kind == authoring.OperationRedo {
				return
			}

			// Target-shaped kinds: SetKindForTarget must round-trip to
			// exactly this fixture's kind given its present hint, and
			// ClearsTarget must classify it exactly per the documented set.
			if got := authoring.SetKindForTarget(fx.target, fx.present); got != fx.kind {
				t.Fatalf("expected SetKindForTarget(target, %v) to resolve to %s, got %s", fx.present, fx.kind, got)
			}
			if got, want := authoring.ClearsTarget(fx.kind), clearing[fx.kind]; got != want {
				t.Fatalf("expected ClearsTarget(%s)=%v, got %v", fx.kind, want, got)
			}
		})
	}
}
