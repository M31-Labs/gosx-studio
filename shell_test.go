package studio

import (
	"testing"

	"m31labs.dev/gosx-studio/blocklayoutruntime"
	"m31labs.dev/gosx-studio/brandruntime"
	"m31labs.dev/gosx-studio/fieldruntime"
	"m31labs.dev/gosx-studio/previewruntime"
	"m31labs.dev/gosx-studio/selectionruntime"
	"m31labs.dev/gosx-studio/styleruntime"
	"m31labs.dev/gosx-studio/workbenchruntime"
)

func TestDefaultShellConfigOwnsReusableStudioChrome(t *testing.T) {
	config := DefaultShellConfig().Normalize()

	if config.Labels.EditorTitle != "Website editor" || config.Labels.PreviewLabel != "Preview" {
		t.Fatalf("labels = %#v", config.Labels)
	}
	if config.Canvas.PreviewShell.OverlayAttr != "data-studio-preview-overlay" {
		t.Fatalf("preview shell = %#v", config.Canvas.PreviewShell)
	}
	if len(config.Modes) != 5 || config.Modes[0].Key != "home" || config.Modes[4].Key != "advanced" {
		t.Fatalf("modes = %#v", config.Modes)
	}
	if len(config.Panels) != 6 || config.Panels[0].Key != "site" || !config.Panels[5].Advanced {
		t.Fatalf("panels = %#v", config.Panels)
	}
	if len(config.Engines) != 5 {
		t.Fatalf("engines = %#v", config.Engines)
	}
	if !config.FeatureEnabled("brand-media-picker") || !config.FeatureEnabled("publish-review") {
		t.Fatalf("feature flags = %#v", config.FeatureFlags)
	}
	// Phase 3 burn-down: the per-slice island flags must be registered
	// and default to true post-2026-05-27 — the legacy JS bundles
	// (studio-engines.js and preview-runtime.js) were deleted then. The
	// flags remain in the registry so host probes via FeatureEnabled keep
	// returning a stable shape. See
	// ~/.hyphae/spaces/m31labs-gosx/plans/gosx-vm-unification-and-editor-bridge-burn-down.md
	// Phase 3 Section E.
	for _, flag := range []string{
		fieldruntime.FeatureFlagKey,
		selectionruntime.FeatureFlagKey,
		brandruntime.FeatureFlagKey,
		blocklayoutruntime.FeatureFlagKey,
		styleruntime.FeatureFlagKey,
		workbenchruntime.FeatureFlagKey,
		previewruntime.FeatureFlagKey,
	} {
		if _, ok := config.FeatureFlags[flag]; !ok {
			t.Fatalf("default shell config must register %q flag (got %#v)", flag, config.FeatureFlags)
		}
		if !config.FeatureEnabled(flag) {
			t.Fatalf("%q must default to true post-legacy-deletion; got false", flag)
		}
	}
	if _, ok := config.Adapter(ResourceProducts); !ok {
		t.Fatalf("expected product resource adapter in %#v", config.Adapters)
	}
}

func TestShellConfigNormalizesHostOverrides(t *testing.T) {
	config := ShellConfig{
		Labels: HostLabels{EditorTitle: " Custom studio "},
		Canvas: CanvasConfig{PreviewShell: CanvasPreviewShell{
			OverlayAttr:     "data-custom-overlay",
			InlineEditClass: "custom-inline",
		}},
		Modes: []ModeConfig{
			{Key: " custom ", Label: " Custom ", Summary: " Host mode "},
			{Key: " ", Label: "Skipped"},
		},
		Resources: []ResourceConfig{
			{Key: " products ", Label: " Products ", Kind: " cms ", Href: " /admin/products ", Summary: " Store records "},
			{Key: "", Label: "Skipped"},
		},
		Panels: []PanelConfig{
			{Key: " custom-panel ", Label: " Custom panel ", Mode: " custom ", Summary: " Host-owned "},
		},
		Engines: []EngineConfig{
			{Key: " custom-engine ", Name: " CustomEngine ", MountID: " custom-mount ", Capabilities: []string{" pointer ", "", " keyboard "}},
		},
		Actions: []ActionConfig{
			{Key: " preview ", Label: " Preview ", Href: " /preview ", Method: " get "},
		},
		FeatureFlags: map[string]bool{
			" brand-media-picker ": false,
			" showcase-3d ":        true,
		},
	}.Normalize()

	if config.Labels.EditorTitle != "Custom studio" || config.Labels.ProductName != "Website editor" {
		t.Fatalf("labels = %#v", config.Labels)
	}
	if config.Canvas.PreviewShell.OverlayAttr != "data-custom-overlay" || config.Canvas.PreviewShell.DropLineAttr != "data-studio-preview-drop-line" || config.Canvas.PreviewShell.InlineEditClass != "custom-inline" {
		t.Fatalf("preview shell = %#v", config.Canvas.PreviewShell)
	}
	if len(config.Modes) != 1 || config.Modes[0].Key != "custom" || config.Modes[0].Label != "Custom" {
		t.Fatalf("modes = %#v", config.Modes)
	}
	if len(config.Resources) != 1 || config.Resources[0].Key != "products" || config.Resources[0].Href != "/admin/products" {
		t.Fatalf("resources = %#v", config.Resources)
	}
	if len(config.Panels) != 1 || config.Panels[0].Key != "custom-panel" {
		t.Fatalf("panels = %#v", config.Panels)
	}
	if len(config.Engines) != 1 || config.Engines[0].Name != "CustomEngine" || len(config.Engines[0].Capabilities) != 2 {
		t.Fatalf("engines = %#v", config.Engines)
	}
	if len(config.Actions) != 1 || config.Actions[0].Method != "GET" || config.Actions[0].Href != "/preview" {
		t.Fatalf("actions = %#v", config.Actions)
	}
	if config.Permissions != (PermissionConfig{CanEdit: true, CanPublish: true, CanManage: true}) {
		t.Fatalf("permissions = %#v", config.Permissions)
	}
	if config.FeatureEnabled("brand-media-picker") || !config.FeatureEnabled("showcase-3d") {
		t.Fatalf("feature flags = %#v", config.FeatureFlags)
	}
}

func TestShellConfigEngineLooksUpDefaultEngines(t *testing.T) {
	config := DefaultShellConfig()

	checks := map[string]struct {
		name    string
		mountID string
	}{
		"canvas":        {name: CanvasEngineName, mountID: "gosx-studio-canvas-engine"},
		"site-map":      {name: SiteMapEngineName, mountID: "gosx-studio-site-map-engine"},
		"flow-designer": {name: FlowDesignerName, mountID: "gosx-studio-flow-engine"},
		"block-layout":  {name: BlockLayoutEngineName, mountID: "gosx-studio-block-layout-engine"},
		"showcase-3d":   {name: Showcase3DEngineName, mountID: "gosx-studio-showcase-3d-engine"},
	}
	for key, want := range checks {
		engine, ok := config.Engine(" " + key + " ")
		if !ok {
			t.Fatalf("missing default engine %q", key)
		}
		if engine.Key != key || engine.Name != want.name || engine.MountID != want.mountID {
			t.Fatalf("%s engine = %#v", key, engine)
		}
		if len(engine.Capabilities) == 0 {
			t.Fatalf("%s engine missing capabilities", key)
		}
	}
}

func TestEngineHostViewProjectsNormalizedEngine(t *testing.T) {
	view := EngineHostView(EngineConfig{
		Key:          " canvas ",
		Name:         " GoSXStudioCanvas ",
		MountID:      " gosx-studio-canvas-engine ",
		Capabilities: []string{" canvas ", "", " pointer ", " keyboard ", " text-input ", " animation "},
	}, " studio-canvas-engine-host ")

	if view["key"] != "canvas" {
		t.Fatalf("key = %v", view["key"])
	}
	if view["name"] != "GoSXStudioCanvas" {
		t.Fatalf("name = %v", view["name"])
	}
	if view["mountId"] != "gosx-studio-canvas-engine" {
		t.Fatalf("mountId = %v", view["mountId"])
	}
	if view["capabilities"] != "canvas pointer keyboard text-input animation" {
		t.Fatalf("capabilities = %v", view["capabilities"])
	}
	if view["class"] != "studio-canvas-engine-host" {
		t.Fatalf("class = %v", view["class"])
	}

	if _, ok := EngineHostView(EngineConfig{Key: "x", Name: "X", MountID: "x"}, " ")["class"]; ok {
		t.Fatalf("blank class should be omitted")
	}
}

func TestStudioRuntimePathsArePublicContracts(t *testing.T) {
	if RuntimeRoot != "/_gosx/studio" {
		t.Fatalf("runtime root = %q", RuntimeRoot)
	}
	if StylesheetPath != RuntimeRoot+"/studio.css" || EngineRuntimePath != RuntimeRoot+"/studio-engines.js" {
		t.Fatalf("runtime paths = %q %q", StylesheetPath, EngineRuntimePath)
	}
	if WorkbenchRuntimePath == "" || CommandRuntimePath == "" || StateRuntimePath == "" {
		t.Fatalf("script paths should be declared")
	}
	if CanvasEngineName != "GoSXStudioCanvas" || SiteMapEngineName != "GoSXStudioSiteMap" || FlowDesignerName != "GoSXStudioFlowDesigner" || BlockLayoutEngineName != "GoSXStudioBlockLayout" || Showcase3DEngineName != "GoSXStudioShowcase3D" {
		t.Fatalf("engine globals changed")
	}
}
