package authoring

import (
	"context"
	"errors"
	"m31labs.dev/gosx-studio/cms/media"
	"m31labs.dev/gosx-studio/core"
	"path/filepath"
	"testing"
)

func interactionGraph() core.InteractionGraph {
	return core.InteractionGraph{ID: "graph_a", Target: core.InteractionTarget{DefinitionID: "def_a"}, Event: core.InteractionHover, Enabled: true, Actions: []core.InteractionAction{{ID: "show", Kind: core.InteractionShow, TargetID: "details"}}}
}
func interactionOptions() InteractionApplyOptions {
	return InteractionApplyOptions{ActorID: "noni", CanManageInteractions: true, Reusable: ReusableState{Definitions: map[string]core.ComponentDefinition{"def_a": {ID: "def_a", Key: "hero", Version: 1, TemplateKey: "hero", Label: "Hero"}}, Instances: map[string]core.ComponentInstance{"inst_a": {ID: "inst_a", DefinitionID: "def_a", PageKey: "home", Region: "main"}}}, CanvasIDs: map[string]bool{"hero": true}, AssetState: AssetState{Assets: map[string]media.Asset{"asset_a": {ID: "asset_a"}}}}
}
func TestInteractionOperationsPermissionsIdempotencyUndoRedoAndReferences(t *testing.T) {
	g := interactionGraph()
	r, e := ApplyInteractionOperation(InteractionState{}, InteractionOperation{ID: "up", Kind: InteractionUpsert, Graph: &g}, interactionOptions())
	if e != nil {
		t.Fatal(e)
	}
	state := r.State
	if e := ValidateInteractionReferenceDeletion(state, "definition", "def_a"); !errors.Is(e, ErrInteractionReferenced) {
		t.Fatalf("reference=%v", e)
	}
	replay, e := ApplyInteractionOperation(state, InteractionOperation{ID: "up", Kind: InteractionUpsert, Graph: &g}, interactionOptions())
	if e != nil || replay.State.Revision != state.Revision {
		t.Fatalf("replay %v", e)
	}
	u, e := ApplyInteractionOperation(state, InteractionOperation{ID: "undo", Kind: InteractionUndo, HistoryID: "up"}, interactionOptions())
	if e != nil || len(u.State.Graphs) != 0 {
		t.Fatalf("undo %v", e)
	}
	rr, e := ApplyInteractionOperation(u.State, InteractionOperation{ID: "redo", Kind: InteractionRedo, HistoryID: "undo"}, interactionOptions())
	if e != nil || len(rr.State.Graphs) != 1 {
		t.Fatalf("redo %v", e)
	}
	if _, e = ApplyInteractionOperation(state, InteractionOperation{ID: "delete", Kind: InteractionDelete, GraphID: g.ID}, InteractionApplyOptions{ActorID: "viewer"}); !errors.Is(e, ErrInteractionUnauthorized) {
		t.Fatalf("permission=%v", e)
	}
}
func TestInteractionReferencesAndCASRestart(t *testing.T) {
	g := interactionGraph()
	g.Actions[0] = core.InteractionAction{ID: "play", Kind: core.InteractionPlayMedia, TargetID: "video", AssetID: "missing"}
	if _, e := ApplyInteractionOperation(InteractionState{}, InteractionOperation{ID: "up", Kind: InteractionUpsert, Graph: &g}, interactionOptions()); !errors.Is(e, ErrInteractionNotFound) {
		t.Fatalf("asset ref=%v", e)
	}
	g = interactionGraph()
	path := filepath.Join(t.TempDir(), "state", "interactions.json")
	store := &FileInteractionStateStore{Path: path}
	r, e := CommitInteractionOperation(context.Background(), store, InteractionOperation{ID: "up", Kind: InteractionUpsert, Graph: &g}, interactionOptions())
	if e != nil {
		t.Fatal(e)
	}
	restart := &FileInteractionStateStore{Path: path}
	got, e := restart.LoadInteractionState(context.Background())
	if e != nil || got.Revision != r.State.Revision || got.Graphs[g.ID].Event != g.Event {
		t.Fatalf("restart %#v %v", got, e)
	}
	if e = restart.SaveInteractionState(context.Background(), got, 0); !errors.Is(e, ErrInteractionConflict) {
		t.Fatalf("CAS=%v", e)
	}
}

func TestAssetDeleteHonorsInteractionReferences(t *testing.T) {
	asset := testAsset("asset_a")
	imported, err := ApplyAssetOperation(AssetState{}, AssetOperation{ID: "asset-import", Kind: AssetImport, Asset: &asset}, manage())
	if err != nil {
		t.Fatal(err)
	}
	g := interactionGraph()
	g.Actions[0] = core.InteractionAction{ID: "play", Kind: core.InteractionPlayMedia, TargetID: "video", AssetID: asset.ID}
	interactions, err := ApplyInteractionOperation(InteractionState{}, InteractionOperation{ID: "graph-up", Kind: InteractionUpsert, Graph: &g}, interactionOptions())
	if err != nil {
		t.Fatal(err)
	}
	opt := manage()
	opt.Interactions = interactions.State
	if _, err = ApplyAssetOperation(imported.State, AssetOperation{ID: "asset-delete", Kind: AssetDelete, AssetID: asset.ID}, opt); !errors.Is(err, ErrInteractionReferenced) {
		t.Fatalf("delete=%v", err)
	}
}
