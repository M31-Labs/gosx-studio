package studio

import "testing"

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
