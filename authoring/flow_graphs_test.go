package authoring

import (
	"context"
	"errors"
	"m31labs.dev/gosx-studio/cms/flows"
	"m31labs.dev/gosx-studio/core"
	"path/filepath"
	"testing"
)

func flowFixture() flows.Graph {
	return flows.Graph{ID: "signup", Name: "Signup", InitialStepID: "form", Steps: []flows.FlowStep{{ID: "form", Label: "Form", Kind: flows.StepForm, DefinitionID: "def"}, {ID: "done", Label: "Done", Kind: flows.StepSuccess}}, Transitions: []flows.FlowTransition{{ID: "ok", FromStepID: "form", ToStepID: "done", Kind: flows.TransitionSuccess}}}
}
func flowOptions() FlowApplyOptions {
	return FlowApplyOptions{ActorID: "noni", CanManageFlows: true, Reusable: ReusableState{Definitions: map[string]core.ComponentDefinition{"def": {ID: "def", Key: "def", Version: 1, TemplateKey: "x", Label: "x"}}}}
}
func TestFlowHistoryReferencesRestartAndCAS(t *testing.T) {
	g := flowFixture()
	r, e := ApplyFlowOperation(FlowState{}, FlowOperation{ID: "up", Kind: FlowUpsert, Flow: &g}, flowOptions())
	if e != nil {
		t.Fatal(e)
	}
	if e = ValidateFlowReferenceDeletion(r.State, "definition", "def"); !errors.Is(e, ErrFlowReferenced) {
		t.Fatalf("ref=%v", e)
	}
	replay, e := ApplyFlowOperation(r.State, FlowOperation{ID: "up", Kind: FlowUpsert, Flow: &g}, flowOptions())
	if e != nil || replay.State.Revision != r.State.Revision {
		t.Fatalf("idempotency=%v", e)
	}
	u, e := ApplyFlowOperation(r.State, FlowOperation{ID: "undo", Kind: FlowUndo, HistoryID: "up"}, flowOptions())
	if e != nil || len(u.State.Flows) != 0 {
		t.Fatalf("undo=%v", e)
	}
	path := filepath.Join(t.TempDir(), "flows.json")
	store := &FileFlowStateStore{Path: path}
	r, e = CommitFlowOperation(context.Background(), store, FlowOperation{ID: "save", Kind: FlowUpsert, Flow: &g}, flowOptions())
	if e != nil {
		t.Fatal(e)
	}
	got, e := (&FileFlowStateStore{Path: path}).LoadFlowState(context.Background())
	if e != nil || got.Revision != r.State.Revision {
		t.Fatalf("restart %v", e)
	}
	if e = store.SaveFlowState(context.Background(), got, 0); !errors.Is(e, ErrFlowConflict) {
		t.Fatalf("cas=%v", e)
	}
	if _, e = ApplyFlowOperation(got, FlowOperation{ID: "del", Kind: FlowDelete, FlowID: g.ID}, FlowApplyOptions{ActorID: "viewer"}); !errors.Is(e, ErrFlowUnauthorized) {
		t.Fatalf("permission=%v", e)
	}
}

func TestFlowRejectsMissingAssetReference(t *testing.T) {
	g := flowFixture()
	g.Steps[0].DefinitionID = ""
	g.Steps[0].AssetID = "missing"
	if _, err := ApplyFlowOperation(FlowState{}, FlowOperation{ID: "up", Kind: FlowUpsert, Flow: &g}, flowOptions()); !errors.Is(err, ErrFlowNotFound) {
		t.Fatalf("asset=%v", err)
	}
}
