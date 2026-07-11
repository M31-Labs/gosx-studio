package authoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"m31labs.dev/gosx-studio/core"
)

func reusableOptions(actor string) ReusableApplyOptions {
	return ReusableApplyOptions{ActorID: actor, ActorLabel: "Editor " + actor, Templates: []core.ComponentTemplate{{Key: "hero", Label: "Hero"}}, Regions: []string{"main", "footer"}, AllowDefinitionMutations: true}
}

func reusableDefinition(headline string) core.ComponentDefinition {
	return core.ComponentDefinition{ID: "def-hero", Key: "shared-hero", TemplateKey: "hero", Label: "Shared hero", Controls: []core.Control{{Key: "headline", Label: "Headline", Kind: core.ControlRichText, Value: headline}}, Styles: core.DefinitionStyles{"base/default": {"background-color": "var(--color-canvas)"}}, Layout: core.DefinitionLayout{"base/default": {"display": "grid"}}}
}

func applyReusable(t *testing.T, state ReusableState, operation ReusableOperation, actor string) ReusableOperationResult {
	t.Helper()
	result, err := ApplyReusableOperation(state, operation, reusableOptions(actor))
	if err != nil {
		t.Fatalf("apply %s: %v", operation.Kind, err)
	}
	return result
}

func TestReusableDefinitionPropagationOverridesDetachAndRestore(t *testing.T) {
	created := applyReusable(t, ReusableState{}, ReusableOperation{ID: "op-create-def", Kind: ReusableCreateDefinition, Definition: ptrDefinition(reusableDefinition("Original"))}, "actor-a")
	if created.Definition == nil || created.Definition.Version != 1 || created.Record.ActorID != "actor-a" || created.Record.TargetHead == "" {
		t.Fatalf("definition create did not produce durable identity/history: %#v", created)
	}
	state := created.State
	for index, id := range []string{"inst-a", "inst-b"} {
		instance := core.ComponentInstance{ID: id, DefinitionID: "def-hero", PageKey: "home", Region: "main", Position: index}
		result := applyReusable(t, state, ReusableOperation{ID: "op-create-" + id, Kind: ReusableCreateInstance, Instance: &instance}, "actor-a")
		state = result.State
	}
	definition := reusableDefinition("Definition v2")
	updated := applyReusable(t, state, ReusableOperation{ID: "op-update-v2", Kind: ReusableUpdateDefinition, Definition: &definition, ExpectedHead: state.Heads["definition:def-hero"]}, "actor-a")
	state = updated.State
	if state.Definitions["def-hero"].Version != 2 || state.Instances["inst-a"].DefinitionVersion != 2 || state.Instances["inst-b"].DefinitionVersion != 2 || len(updated.Record.AffectedIDs) != 2 {
		t.Fatalf("definition update must fan out to attached instances: %#v", updated.Record)
	}
	override := applyReusable(t, state, ReusableOperation{ID: "op-override", Kind: ReusableSetInstanceOverride, InstanceID: "inst-a", OverrideKey: "control:headline", Override: core.ExplicitOverride{Present: true, Value: "Only A"}, ExpectedHead: state.Heads["instance:inst-a"]}, "actor-a")
	state = override.State
	definition = reusableDefinition("Definition v3")
	updated = applyReusable(t, state, ReusableOperation{ID: "op-update-v3", Kind: ReusableUpdateDefinition, Definition: &definition, ExpectedHead: state.Heads["definition:def-hero"]}, "actor-a")
	state = updated.State
	a, err := core.ResolveComponentInstance(state.Instances["inst-a"], state.Definitions)
	if err != nil {
		t.Fatal(err)
	}
	b, err := core.ResolveComponentInstance(state.Instances["inst-b"], state.Definitions)
	if err != nil {
		t.Fatal(err)
	}
	if a.Controls[0].Value != "Only A" || b.Controls[0].Value != "Definition v3" {
		t.Fatalf("definition propagation must preserve explicit instance overrides: a=%q b=%q", a.Controls[0].Value, b.Controls[0].Value)
	}
	cleared := applyReusable(t, state, ReusableOperation{ID: "op-clear", Kind: ReusableClearOverride, InstanceID: "inst-a", OverrideKey: "control:headline", ExpectedHead: state.Heads["instance:inst-a"]}, "actor-a")
	state = cleared.State
	a, _ = core.ResolveComponentInstance(state.Instances["inst-a"], state.Definitions)
	if a.Controls[0].Value != "Definition v3" {
		t.Fatalf("clear override must restore definition inheritance, got %q", a.Controls[0].Value)
	}
	detached := applyReusable(t, state, ReusableOperation{ID: "op-detach", Kind: ReusableDetachInstance, InstanceID: "inst-a", ExpectedHead: state.Heads["instance:inst-a"]}, "actor-a")
	state = detached.State
	definition = reusableDefinition("Definition v4")
	updated = applyReusable(t, state, ReusableOperation{ID: "op-update-v4", Kind: ReusableUpdateDefinition, Definition: &definition, ExpectedHead: state.Heads["definition:def-hero"]}, "actor-a")
	state = updated.State
	a, _ = core.ResolveComponentInstance(state.Instances["inst-a"], state.Definitions)
	b, _ = core.ResolveComponentInstance(state.Instances["inst-b"], state.Definitions)
	if a.Controls[0].Value != "Definition v3" || b.Controls[0].Value != "Definition v4" || state.Instances["inst-a"].DefinitionVersion == state.Instances["inst-b"].DefinitionVersion {
		t.Fatalf("detach must preserve materialization while attached instance follows update: a=%q b=%q", a.Controls[0].Value, b.Controls[0].Value)
	}
	restored := applyReusable(t, state, ReusableOperation{ID: "op-restore", Kind: ReusableRestoreInstance, InstanceID: "inst-a", ExpectedHead: state.Heads["instance:inst-a"]}, "actor-a")
	a, _ = core.ResolveComponentInstance(restored.State.Instances["inst-a"], restored.State.Definitions)
	if a.Controls[0].Value != "Definition v4" || restored.State.Instances["inst-a"].Detached || restored.State.Instances["inst-a"].Materialized != nil {
		t.Fatalf("restore must reattach to latest definition: %#v resolved=%q", restored.State.Instances["inst-a"], a.Controls[0].Value)
	}
}

func TestReusableOperationsRejectStaleUnauthorizedInvalidAndIDReuse(t *testing.T) {
	created := applyReusable(t, ReusableState{}, ReusableOperation{ID: "op-create", Kind: ReusableCreateDefinition, Definition: ptrDefinition(reusableDefinition("Original"))}, "actor-a")
	instance := core.ComponentInstance{ID: "inst-a", DefinitionID: "def-hero", PageKey: "home", Region: "main"}
	createdInstance := applyReusable(t, created.State, ReusableOperation{ID: "op-instance", Kind: ReusableCreateInstance, Instance: &instance}, "actor-a")
	state := createdInstance.State
	_, err := ApplyReusableOperation(state, ReusableOperation{ID: "op-stale", Kind: ReusableDetachInstance, InstanceID: "inst-a", ExpectedHead: "stale"}, reusableOptions("actor-a"))
	if !errors.Is(err, ErrReusableConflict) {
		t.Fatalf("stale head must conflict, got %v", err)
	}
	_, err = ApplyReusableOperation(state, ReusableOperation{ID: "op-invalid", Kind: ReusableSetInstanceOverride, InstanceID: "inst-a", OverrideKey: "raw-html", Override: core.ExplicitOverride{Present: true, Value: "<script>"}}, reusableOptions("actor-a"))
	if !errors.Is(err, core.ErrReusableInvalid) {
		t.Fatalf("invalid override key must be rejected, got %v", err)
	}
	_, err = ApplyReusableOperation(state, ReusableOperation{ID: "op-css-injection", Kind: ReusableSetInstanceOverride, InstanceID: "inst-a", OverrideKey: "style:base/default:background-color", Override: core.ExplicitOverride{Present: true, Value: "red;display:none"}}, reusableOptions("actor-a"))
	if !errors.Is(err, core.ErrReusableInvalid) {
		t.Fatalf("unsafe style override must be rejected, got %v", err)
	}
	_, err = ApplyReusableOperation(state, ReusableOperation{ID: "op-instance", Kind: ReusableDetachInstance, InstanceID: "inst-a"}, reusableOptions("actor-a"))
	if !errors.Is(err, ErrReusableIdempotency) {
		t.Fatalf("operation id reuse with different request must fail, got %v", err)
	}
	_, err = ApplyReusableOperation(state, ReusableOperation{ID: "op-anon", Kind: ReusableDetachInstance, InstanceID: "inst-a"}, reusableOptions(""))
	if !errors.Is(err, ErrReusableUnauthorized) {
		t.Fatalf("anonymous operation must fail, got %v", err)
	}
	denied := reusableOptions("actor-designer")
	denied.AllowDefinitionMutations = false
	_, err = ApplyReusableOperation(state, ReusableOperation{ID: "op-denied-definition", Kind: ReusableUpdateDefinition, Definition: ptrDefinition(reusableDefinition("Denied")), ExpectedHead: state.Heads["definition:def-hero"]}, denied)
	if !errors.Is(err, ErrReusableUnauthorized) {
		t.Fatalf("actor without definition capability must be rejected, got %v", err)
	}
}

func TestReusableInverseIsActorScopedAndDurable(t *testing.T) {
	created := applyReusable(t, ReusableState{}, ReusableOperation{ID: "op-create", Kind: ReusableCreateDefinition, Definition: ptrDefinition(reusableDefinition("Original"))}, "actor-a")
	instance := core.ComponentInstance{ID: "inst-a", DefinitionID: "def-hero", PageKey: "home", Region: "main"}
	createdInstance := applyReusable(t, created.State, ReusableOperation{ID: "op-instance", Kind: ReusableCreateInstance, Instance: &instance}, "actor-a")
	state := createdInstance.State
	set := applyReusable(t, state, ReusableOperation{ID: "op-set", Kind: ReusableSetInstanceOverride, InstanceID: "inst-a", OverrideKey: "control:headline", Override: core.ExplicitOverride{Present: true, Value: "Override"}, ExpectedHead: state.Heads["instance:inst-a"]}, "actor-a")
	_, err := ApplyReusableInverse(set.State, set.Record.ID, "op-undo-b", set.Record.TargetHead, reusableOptions("actor-b"))
	if !errors.Is(err, ErrReusableUnauthorized) {
		t.Fatalf("actor B must not invert actor A history, got %v", err)
	}
	undo, err := ApplyReusableInverse(set.State, set.Record.ID, "op-undo-a", set.Record.TargetHead, reusableOptions("actor-a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, present := undo.State.Instances["inst-a"].Overrides["control:headline"]; present || undo.Record.UndoOf != "op-set" || undo.Record.ActorID != "actor-a" || undo.Record.TargetHead == set.Record.TargetHead {
		t.Fatalf("inverse must clear prior override and append actor history: %#v", undo.Record)
	}
	retry, err := ApplyReusableInverse(undo.State, set.Record.ID, "op-undo-a", set.Record.TargetHead, reusableOptions("actor-a"))
	if err != nil || retry.Record.ID != undo.Record.ID || retry.State.Revision != undo.State.Revision {
		t.Fatalf("inverse retry must be idempotent: result=%#v err=%v", retry.Record, err)
	}
	_, err = ApplyReusableInverse(undo.State, set.Record.ID, "op-old-undo", set.Record.TargetHead, reusableOptions("actor-a"))
	if !errors.Is(err, ErrReusableConflict) {
		t.Fatalf("consumed history head must reject another inverse, got %v", err)
	}
	redo, err := ApplyReusableInverse(undo.State, undo.Record.ID, "op-redo-a", undo.Record.TargetHead, reusableOptions("actor-a"))
	if err != nil {
		t.Fatal(err)
	}
	if value := redo.State.Instances["inst-a"].Overrides["control:headline"]; !value.Present || value.Value != "Override" || redo.Record.UndoOf != undo.Record.ID {
		t.Fatalf("inverse of inverse must restore the explicit override: %#v", redo)
	}
}

func TestDefinitionInverseCreatesMonotonicCompensatingVersion(t *testing.T) {
	created := applyReusable(t, ReusableState{}, ReusableOperation{ID: "op-create-def", Kind: ReusableCreateDefinition, Definition: ptrDefinition(reusableDefinition("Version one"))}, "actor-a")
	instance := core.ComponentInstance{ID: "inst-a", DefinitionID: "def-hero", PageKey: "home", Region: "main"}
	withInstance := applyReusable(t, created.State, ReusableOperation{ID: "op-instance", Kind: ReusableCreateInstance, Instance: &instance}, "actor-a")
	updatedDefinition := reusableDefinition("Version two")
	updated := applyReusable(t, withInstance.State, ReusableOperation{ID: "op-update", Kind: ReusableUpdateDefinition, Definition: &updatedDefinition, ExpectedHead: withInstance.State.Heads["definition:def-hero"]}, "actor-a")
	undo, err := ApplyReusableInverse(updated.State, updated.Record.ID, "op-update-undo", updated.Record.TargetHead, reusableOptions("actor-a"))
	if err != nil {
		t.Fatal(err)
	}
	definition := undo.State.Definitions["def-hero"]
	resolved, _ := core.ResolveComponentInstance(undo.State.Instances["inst-a"], undo.State.Definitions)
	if definition.Version != 3 || definition.Controls[0].Value != "Version one" || definition.UpdatedRevision != undo.State.Revision || undo.State.Instances["inst-a"].DefinitionVersion != 3 || resolved.Controls[0].Value != "Version one" {
		t.Fatalf("definition inverse must be a monotonic propagated version: definition=%#v instance=%#v", definition, undo.State.Instances["inst-a"])
	}
}

func TestReusableOperationFromFormKeepsTypedInstancePayload(t *testing.T) {
	operation, err := ReusableOperationFromForm(map[string]string{
		ReusableFieldOperation:    string(ReusableCreateInstance),
		ReusableFieldOperationID:  "op-form-create",
		ReusableFieldDefinitionID: "def-hero",
		ReusableFieldInstanceID:   "inst-form",
		ReusableFieldPageKey:      "home",
		ReusableFieldRegion:       "main",
		ReusableFieldPosition:     "2",
		ReusableFieldExpectedHead: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if operation.Instance == nil || operation.Instance.ID != "inst-form" || operation.Instance.Position != 2 || operation.Instance.Region != "main" {
		t.Fatalf("typed create-instance form was not preserved: %#v", operation)
	}
	override, err := ReusableOperationFromForm(map[string]string{
		ReusableFieldOperation:     string(ReusableSetInstanceOverride),
		ReusableFieldOperationID:   "op-form-override",
		ReusableFieldInstanceID:    "inst-form",
		ReusableFieldOverrideKey:   "control:headline",
		ReusableFieldOverrideValue: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !override.Override.Present || override.Override.Value != "" {
		t.Fatalf("authored empty override must remain explicitly present: %#v", override.Override)
	}
	if _, err := ReusableOperationFromForm(map[string]string{ReusableFieldOperation: string(ReusableUpdateDefinition), ReusableFieldOperationID: "unsafe-json"}); err == nil {
		t.Fatal("definition JSON must not be accepted from the client form")
	}
}

func TestReusableStateRoundTripPreservesDefinitionsInstancesHeadsAndHistory(t *testing.T) {
	created := applyReusable(t, ReusableState{}, ReusableOperation{ID: "op-create", Kind: ReusableCreateDefinition, Definition: ptrDefinition(reusableDefinition("Durable"))}, "actor-a")
	instance := core.ComponentInstance{ID: "inst-durable", DefinitionID: "def-hero", PageKey: "home", Region: "main", Overrides: map[string]core.ExplicitOverride{"control:headline": {Present: true, Value: "Persisted override"}}}
	result := applyReusable(t, created.State, ReusableOperation{ID: "op-instance", Kind: ReusableCreateInstance, Instance: &instance}, "actor-a")
	data, err := json.Marshal(result.State)
	if err != nil {
		t.Fatal(err)
	}
	var restored ReusableState
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	resolved, err := core.ResolveComponentInstance(restored.Instances["inst-durable"], restored.Definitions)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Revision != result.State.Revision || restored.Heads["instance:inst-durable"] == "" || len(restored.Operations) != 2 || resolved.Controls[0].Value != "Persisted override" {
		t.Fatalf("durable state did not survive JSON round trip: %#v", restored)
	}
}

type reusableMemoryStore struct {
	state         ReusableState
	savedExpected []uint64
}

func (store *reusableMemoryStore) LoadReusableState(context.Context) (ReusableState, error) {
	return cloneReusableState(store.state), nil
}

func (store *reusableMemoryStore) SaveReusableState(_ context.Context, state ReusableState, expected uint64) error {
	if store.state.Revision != expected {
		return fmt.Errorf("%w: store revision changed", ErrReusableConflict)
	}
	store.savedExpected = append(store.savedExpected, expected)
	store.state = cloneReusableState(state)
	return nil
}

func TestCommitReusableOperationPersistsStateAndHistoryWithCAS(t *testing.T) {
	store := &reusableMemoryStore{}
	created, err := CommitReusableOperation(context.Background(), store, ReusableOperation{ID: "op-store-definition", Kind: ReusableCreateDefinition, Definition: ptrDefinition(reusableDefinition("Stored"))}, reusableOptions("actor-a"))
	if err != nil {
		t.Fatal(err)
	}
	instance := core.ComponentInstance{ID: "inst-store", DefinitionID: "def-hero", PageKey: "home", Region: "main"}
	createdInstance, err := CommitReusableOperation(context.Background(), store, ReusableOperation{ID: "op-store-instance", Kind: ReusableCreateInstance, Instance: &instance}, reusableOptions("actor-a"))
	if err != nil {
		t.Fatal(err)
	}
	if store.state.Revision != 2 || len(store.state.Operations) != 2 || len(store.savedExpected) != 2 || store.savedExpected[0] != 0 || store.savedExpected[1] != 1 || created.State.Revision != 1 || createdInstance.State.Heads["instance:inst-store"] == "" {
		t.Fatalf("commit seam did not persist state/history with CAS: %#v expected=%v", store.state, store.savedExpected)
	}
}

func ptrDefinition(value core.ComponentDefinition) *core.ComponentDefinition { return &value }
