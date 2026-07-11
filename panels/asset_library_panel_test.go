package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/media"
)

func TestAssetLibraryPanelAccessibleInspectorAndReferenceGuard(t *testing.T) {
	a := media.Asset{ID: "asset_a", Kind: media.AssetKindImage, Filename: "hero.png", Alt: "Hero", Responsive: media.ResponsiveMetadata{"mobile": {Alt: "Mobile"}}}
	b := authoring.AssetBinding{ID: "binding_a", AssetID: a.ID, Target: authoring.AssetBindingTarget{ComponentKey: "hero", Property: "image"}}
	html := gosx.RenderHTML(RenderAssetLibraryPanel(AssetLibraryPanelOptions{Action: "/assets", CSRFToken: "csrf", SelectedAssetID: a.ID, Assets: []media.Asset{a}, Bindings: []authoring.AssetBinding{b}, Target: b.Target, OperationIDs: map[string]string{"bind": "op-bind"}}))
	for _, want := range []string{`type="search"`, `aria-label="Available assets"`, `data-studio-asset-inspector="asset_a"`, `data-studio-asset-breakpoint="base"`, `data-studio-asset-breakpoint="tablet"`, `data-studio-asset-breakpoint="mobile"`, `Alternative text`, `Horizontal focal point`, `disabled="disabled"`, `Used by 1 bindings`, `name="csrf_token" value="csrf"`} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q:\n%s", want, html)
		}
	}
}
