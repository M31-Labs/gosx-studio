package authoring

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/cms/media"
	"m31labs.dev/gosx-studio/core"
)

func testAsset(id string) media.Asset {
	return media.Asset{ID: id, Kind: media.AssetKindImage, URL: "/assets/a.png", Filename: "a.png", ContentType: "image/png", Size: 4, ContentHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Alt: "Hero", Version: 1, Created: time.Unix(1, 0), Updated: time.Unix(1, 0)}
}
func manage() AssetApplyOptions {
	return AssetApplyOptions{ActorID: "noni", CanManageAssets: true, CanBindAssets: true}
}

func TestAssetLifecycleBindingsReferenceSafetyUndoAndIdempotency(t *testing.T) {
	state := AssetState{}
	asset := testAsset("asset_a")
	r, err := ApplyAssetOperation(state, AssetOperation{ID: "op-import", Kind: AssetImport, Asset: &asset}, manage())
	if err != nil {
		t.Fatal(err)
	}
	state = r.State
	target := AssetBindingTarget{ComponentKey: "hero", Property: "image"}
	binding := AssetBinding{ID: AssetBindingID(target), AssetID: asset.ID, Target: target, Responsive: media.ResponsiveMetadata{"mobile": {Alt: "Mobile hero", Crop: media.CropCover, FocalX: .2, FocalY: .8}}}
	r, err = ApplyAssetOperation(state, AssetOperation{ID: "op-bind", Kind: AssetBind, Binding: &binding}, manage())
	if err != nil {
		t.Fatal(err)
	}
	state = r.State
	if _, _, ok := ResolveAssetBinding(state, ReusableState{}, "", "", "hero", "image"); !ok {
		t.Fatal("binding did not resolve")
	}
	if _, err := ApplyAssetOperation(state, AssetOperation{ID: "op-delete", Kind: AssetDelete, AssetID: asset.ID}, manage()); !errors.Is(err, ErrAssetReferenced) {
		t.Fatalf("delete error=%v", err)
	}
	r, err = ApplyAssetOperation(state, AssetOperation{ID: "op-undo", Kind: AssetUndo, HistoryID: "op-bind"}, manage())
	if err != nil {
		t.Fatal(err)
	}
	state = r.State
	if len(state.Bindings) != 0 {
		t.Fatal("undo did not restore binding snapshot")
	}
	r2, err := ApplyAssetOperation(state, AssetOperation{ID: "op-undo", Kind: AssetUndo, HistoryID: "op-bind"}, manage())
	if err != nil {
		t.Fatal(err)
	}
	if r2.State.Revision != state.Revision {
		t.Fatal("idempotent replay changed state")
	}
}

func TestAssetPermissionsConflictsAndInstanceDefinitionIntegration(t *testing.T) {
	a := testAsset("asset_a")
	r, _ := ApplyAssetOperation(AssetState{}, AssetOperation{ID: "i", Kind: AssetImport, Asset: &a}, manage())
	state := r.State
	if _, err := ApplyAssetOperation(state, AssetOperation{ID: "x", Kind: AssetDelete, AssetID: a.ID}, AssetApplyOptions{ActorID: "viewer"}); !errors.Is(err, ErrAssetUnauthorized) {
		t.Fatalf("permission=%v", err)
	}
	def := core.ComponentDefinition{ID: "def_a", Key: "hero", Version: 1, TemplateKey: "hero", Label: "Hero"}
	inst := core.ComponentInstance{ID: "inst_a", DefinitionID: def.ID, DefinitionVersion: 1, PageKey: "home", Region: "main"}
	reusable := ReusableState{Definitions: map[string]core.ComponentDefinition{def.ID: def}, Instances: map[string]core.ComponentInstance{inst.ID: inst}}
	b := AssetBinding{ID: "binding_a", AssetID: a.ID, Target: AssetBindingTarget{InstanceID: inst.ID, Property: "image"}}
	opt := manage()
	opt.Reusable = reusable
	if _, err := ApplyAssetOperation(state, AssetOperation{ID: "b", Kind: AssetBind, Binding: &b, ExpectedHead: "stale"}, opt); !errors.Is(err, ErrAssetConflict) {
		t.Fatalf("conflict=%v", err)
	}
	if _, err := ApplyAssetOperation(state, AssetOperation{ID: "b", Kind: AssetBind, Binding: &b}, opt); err != nil {
		t.Fatal(err)
	}
}

func TestFileAssetStateStoreCASAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "assets.json")
	store := &FileAssetStateStore{Path: path}
	a := testAsset("asset_a")
	r, err := CommitAssetOperation(context.Background(), store, AssetOperation{ID: "i", Kind: AssetImport, Asset: &a}, manage())
	if err != nil {
		t.Fatal(err)
	}
	restarted := &FileAssetStateStore{Path: path}
	got, err := restarted.LoadAssetState(context.Background())
	if err != nil || got.Revision != r.State.Revision || got.Assets[a.ID].ContentHash != a.ContentHash {
		t.Fatalf("restart: %#v %v", got, err)
	}
	if err := restarted.SaveAssetState(context.Background(), got, 0); !errors.Is(err, ErrAssetConflict) {
		t.Fatalf("CAS=%v", err)
	}
}

func TestAssetRejectsInvalidResponsiveMetadata(t *testing.T) {
	for _, responsive := range []media.ResponsiveMetadata{
		{"watch": {Crop: media.CropCover}},
		{"mobile": {Crop: media.CropMode("stretch")}},
		{"tablet": {FocalX: 2}},
	} {
		a := testAsset("asset_a")
		a.Responsive = responsive
		if _, err := ApplyAssetOperation(AssetState{}, AssetOperation{ID: "i", Kind: AssetImport, Asset: &a}, manage()); !errors.Is(err, ErrAssetInvalid) {
			t.Fatalf("responsive %#v accepted: %v", responsive, err)
		}
	}
}
