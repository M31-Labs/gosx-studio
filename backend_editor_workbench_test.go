package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendEditorWorkbenchOwnsFrameAndSlotComposition(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title:      "Website editor",
		SaveAction: "/admin/editor/__actions/save",
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero selected",
			Zoom:           "100",
		},
	}, WorkbenchShellViewOptions{
		CSRFToken: "csrf-token",
	})

	html := gosx.RenderHTML(RenderBackendEditorWorkbench(BackendEditorWorkbenchProps{
		View:         view,
		AuthoringURL: "/admin/editor/__actions/authoring",
		Toolbar:      gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-toolbar", "true")), gosx.Text("Toolbar")),
		CanvasBar:    gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-bar", "true")), gosx.Text("Canvas bar")),
		CanvasStatus: gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-status", "true")), gosx.Text("Ready")),
		LeftRail: []gosx.Node{
			gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
			gosx.El("section", gosx.Attrs(gosx.Attr("data-block-layout", "true")), gosx.Text("Blocks")),
		},
		Board: []gosx.Node{
			gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-engine", "true")), gosx.Text("Site map")),
			gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-canvas", "true")), gosx.Text("Canvas")),
		},
		RightRail: []gosx.Node{
			gosx.El("section", gosx.Attrs(gosx.Attr("data-inspector", "true")), gosx.Text("Inspector")),
			gosx.El("section", gosx.Attrs(gosx.Attr("data-publish", "true")), gosx.Text("Publish")),
		},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-frame-slot="true"`,
		`data-gosx-studio-workbench="true"`,
		`data-gosx-studio-frame-renderer="gosx-studio"`,
		`data-gosx-studio-authoring-url="/admin/editor/__actions/authoring"`,
		`data-gosx-studio-stage-renderer="gosx-studio"`,
		`data-gosx-studio-rail-resizer-renderer="gosx-studio"`,
		`data-gosx-studio-left-rail-slot="true"`,
		`data-site-navigator="true"`,
		`data-block-layout="true"`,
		`data-gosx-studio-board-slot="true"`,
		`data-site-map-engine="true"`,
		`data-site-map-canvas="true"`,
		`data-studio-preview-shell="true"`,
		`data-studio-preview-dock="true"`,
		`data-gosx-studio-preview-dock="true"`,
		`data-gosx-studio-preview-dock-renderer="gosx-studio"`,
		`data-studio-preview-dock-title="true"`,
		`data-gosx-studio-preview-dock-label="true"`,
		`data-studio-preview-field-count="true"`,
		`data-gosx-studio-preview-field-meter="true"`,
		`data-studio-preview-action="content"`,
		`data-studio-preview-action="style"`,
		`data-gosx-studio-preview-command="prev-field"`,
		`data-gosx-studio-preview-command="next-field"`,
		`data-gosx-studio-preview-command="field-action"`,
		`data-gosx-studio-preview-command="clear"`,
		`data-gosx-studio-right-rail-slot="true"`,
		`data-inspector="true"`,
		`data-publish="true"`,
		`data-editor-toolbar="true"`,
		`data-editor-canvas-bar="true"`,
		`data-editor-canvas-status="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor workbench missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`data-editor-toolbar="true"`,
		`data-gosx-studio-left-rail-slot="true"`,
		`data-gosx-studio-board-slot="true"`,
		`data-gosx-studio-right-rail-slot="true"`,
	)
}

func TestRenderBackendEditorWorkbenchContentOwnsStandardContentOrder(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title:      "Website editor",
		SaveAction: "/admin/editor/__actions/save",
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero selected",
			Zoom:           "100",
		},
	}, WorkbenchShellViewOptions{
		CSRFToken: "csrf-token",
	})

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchContent(BackendEditorWorkbenchContentProps{
		View:            view,
		AuthoringURL:    "/admin/editor/__actions/authoring",
		Toolbar:         gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-toolbar", "true")), gosx.Text("Toolbar")),
		CanvasBar:       gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-bar", "true")), gosx.Text("Canvas bar")),
		CanvasStatus:    gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-status", "true")), gosx.Text("Ready")),
		SiteNavigator:   gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		BlockLayout:     gosx.El("section", gosx.Attrs(gosx.Attr("data-block-layout", "true")), gosx.Text("Blocks")),
		SiteMapEngine:   gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-engine", "true")), gosx.Text("Site map")),
		SiteMapCanvas:   gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-canvas", "true")), gosx.Text("Canvas")),
		InspectorChrome: gosx.El("section", gosx.Attrs(gosx.Attr("data-inspector-chrome", "true")), gosx.Text("Inspector")),
		HomeInspector:   gosx.El("section", gosx.Attrs(gosx.Attr("data-home-inspector", "true")), gosx.Text("Home")),
		LookPanel:       gosx.El("section", gosx.Attrs(gosx.Attr("data-look-panel", "true")), gosx.Text("Look")),
		BrandPanel:      gosx.El("section", gosx.Attrs(gosx.Attr("data-brand-panel", "true")), gosx.Text("Brand")),
		NavigationPanel: gosx.El("section", gosx.Attrs(gosx.Attr("data-navigation-panel", "true")), gosx.Text("Navigation")),
		CheckoutPanel:   gosx.El("section", gosx.Attrs(gosx.Attr("data-checkout-panel", "true")), gosx.Text("Checkout")),
		PublishPanel:    gosx.El("section", gosx.Attrs(gosx.Attr("data-publish-panel", "true")), gosx.Text("Publish")),
		AdvancedPanel:   gosx.El("section", gosx.Attrs(gosx.Attr("data-advanced-panel", "true")), gosx.Text("Advanced")),
	}))

	for _, fragment := range []string{
		`data-gosx-studio-frame-slot="true"`,
		`data-gosx-studio-frame-renderer="gosx-studio"`,
		`data-gosx-studio-authoring-url="/admin/editor/__actions/authoring"`,
		`data-gosx-studio-left-rail-slot="true"`,
		`data-gosx-studio-board-slot="true"`,
		`data-gosx-studio-right-rail-slot="true"`,
		`data-editor-toolbar="true"`,
		`data-editor-canvas-bar="true"`,
		`data-editor-canvas-status="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor workbench content missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`data-gosx-studio-left-rail-slot="true"`,
		`data-site-navigator="true"`,
		`data-block-layout="true"`,
		`data-gosx-studio-board-slot="true"`,
		`data-site-map-engine="true"`,
		`data-site-map-canvas="true"`,
		`data-gosx-studio-right-rail-slot="true"`,
		`data-inspector-chrome="true"`,
		`data-home-inspector="true"`,
		`data-look-panel="true"`,
		`data-brand-panel="true"`,
		`data-navigation-panel="true"`,
		`data-checkout-panel="true"`,
		`data-publish-panel="true"`,
		`data-advanced-panel="true"`,
	)
}

func TestRenderBackendEditorWorkbenchPanelStackOwnsNestedPanelComposition(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title:      "Website editor",
		SaveAction: "/admin/editor/__actions/save",
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero selected",
			Zoom:           "100",
		},
	}, WorkbenchShellViewOptions{
		CSRFToken: "csrf-token",
	})
	brandPanel := map[string]any{
		"kicker":          "Brand",
		"title":           "Identity",
		"summary":         "Brand system",
		"previewLabel":    "Preview",
		"groupLabel":      "Brand groups",
		"defaultGroupKey": "files",
		"groups": []map[string]any{{
			"key":      "files",
			"inputID":  "brandFiles",
			"label":    "Files",
			"summary":  "Media",
			"selected": true,
		}},
	}
	brandFields := map[string]any{
		"preview": map[string]any{
			"cornerLabel": "Logo",
			"buttonLabel": "Adjust logo",
			"logoURL":     "/logo.png",
			"logoAlt":     "Logo",
		},
	}
	brandMediaPicker := map[string]any{
		"kicker":          "Assets",
		"title":           "Brand media",
		"summary":         "Choose brand assets.",
		"filterLabel":     "Filter media",
		"assetLabel":      "Brand assets",
		"emptyLabel":      "No media yet.",
		"uploadHref":      "/admin/media",
		"formID":          "editorWorkbenchForm",
		"saveAction":      "/admin/editor/__actions/save",
		"logoPickName":    "studioPickLogoUrl",
		"faviconPickName": "studioPickFaviconUrl",
		"hasAssets":       true,
		"assets": []map[string]any{{
			"id":          "asset-logo",
			"url":         "/logo.png",
			"alt":         "Logo",
			"filename":    "logo.png",
			"cardClass":   "studio-brand-media-card",
			"filterGroup": "ready",
			"statusLabel": "Ready",
			"actionLabel": "Use logo.png",
		}},
	}
	advancedPanel := map[string]any{
		"kicker":          "Advanced",
		"title":           "Controls",
		"summary":         "Advanced controls",
		"groupLabel":      "Advanced groups",
		"defaultGroupKey": "flows",
		"groups": []map[string]any{
			{"key": "flows", "inputID": "advancedFlows", "label": "Flows", "summary": "Automation", "selected": true},
			{"key": "tools", "inputID": "advancedTools", "label": "Tools", "summary": "Tools"},
			{"key": "schema", "inputID": "advancedSchema", "label": "Schema", "summary": "Workspace"},
			{"key": "schedule", "inputID": "advancedSchedule", "label": "Schedule", "summary": "Calendar"},
			{"key": "typography", "inputID": "advancedType", "label": "Type", "summary": "Typography"},
			{"key": "settings", "inputID": "advancedSettings", "label": "SEO", "summary": "Settings"},
		},
	}

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                  view,
		AuthoringURL:          "/admin/editor/__actions/authoring",
		Toolbar:               gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-toolbar", "true")), gosx.Text("Toolbar")),
		CanvasBar:             gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-bar", "true")), gosx.Text("Canvas bar")),
		CanvasStatus:          gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-status", "true")), gosx.Text("Ready")),
		SiteNavigator:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:            map[string]any{"kicker": "Home", "title": "Sections", "empty": "No sections."},
		HomeLayerSelection:    gosx.El("select", gosx.Attrs(gosx.Attr("data-home-layer-selection", "true"))),
		BlockLayoutEngineHost: gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		BlockLibraryPanel:     gosx.El("section", gosx.Attrs(gosx.Attr("data-block-library", "true")), gosx.Text("Library")),
		SiteMapEngine:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-engine", "true")), gosx.Text("Site map")),
		SiteMapCanvas:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-canvas", "true")), gosx.Text("Canvas")),
		InspectorChrome:       gosx.El("section", gosx.Attrs(gosx.Attr("data-inspector-chrome", "true")), gosx.Text("Inspector")),
		HomeInspector:         gosx.El("section", gosx.Attrs(gosx.Attr("data-home-inspector", "true")), gosx.Text("Home")),
		LookPanel:             gosx.El("section", gosx.Attrs(gosx.Attr("data-look-panel", "true")), gosx.Text("Look")),
		BrandPanel:            brandPanel,
		BrandFields:           brandFields,
		BrandMediaPickerView:  brandMediaPicker,
		NavigationPanel:       gosx.El("section", gosx.Attrs(gosx.Attr("data-navigation-panel", "true")), gosx.Text("Navigation")),
		CheckoutPanel:         gosx.El("section", gosx.Attrs(gosx.Attr("data-checkout-panel", "true")), gosx.Text("Checkout")),
		PublishPanel:          gosx.El("section", gosx.Attrs(gosx.Attr("data-publish-panel", "true")), gosx.Text("Publish")),
		AdvancedPanel:         advancedPanel,
		FlowDesignerView:      map[string]any{"defaultFlowKey": "contact"},
		AdvancedToolsPanelView: map[string]any{
			"class": "editor-panel editor-panel--advanced-tools",
			"key":   "advanced-tools",
			"mode":  "advanced",
			"title": "Tools",
			"empty": "No tools.",
		},
		AdvancedWorkspaceFieldPanelView: map[string]any{
			"class":     "editor-panel editor-panel--advanced-fields",
			"key":       "workspace-fields",
			"mode":      "advanced",
			"kicker":    "Workspace",
			"title":     "Workspace fields",
			"empty":     "No workspace fields.",
			"hasFields": true,
			"fieldList": gosx.El("div", gosx.Attrs(gosx.Attr("data-workspace-field-list", "true")), gosx.Text("Workspace fields")),
		},
		AdvancedCalendarFieldPanelView: map[string]any{
			"class":     "editor-panel editor-panel--advanced-fields",
			"key":       "calendar-fields",
			"mode":      "advanced",
			"kicker":    "Calendar",
			"title":     "Calendar fields",
			"empty":     "No calendar fields.",
			"hasFields": true,
			"fieldList": gosx.El("div", gosx.Attrs(gosx.Attr("data-calendar-field-list", "true")), gosx.Text("Calendar fields")),
		},
		AdvancedTypeFieldPanelView: map[string]any{
			"class":     "editor-panel editor-panel--advanced-fields",
			"key":       "type-fields",
			"mode":      "advanced",
			"kicker":    "Type",
			"title":     "Type fields",
			"empty":     "No type fields.",
			"hasFields": true,
			"fieldList": gosx.El("div", gosx.Attrs(gosx.Attr("data-type-field-list", "true")), gosx.Text("Type fields")),
		},
		AdvancedSettingsPanelView: map[string]any{
			"class":     "editor-panel editor-panel--advanced-settings",
			"key":       "advanced-settings",
			"mode":      "advanced",
			"kicker":    "Settings",
			"title":     "Advanced settings",
			"empty":     "No settings.",
			"hasFields": true,
			"fieldList": gosx.El("div", gosx.Attrs(gosx.Attr("data-settings-field-list", "true")), gosx.Text("Settings fields")),
		},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-frame-slot="true"`,
		`data-gosx-studio-frame-renderer="gosx-studio"`,
		`data-gosx-studio-left-rail-slot="true"`,
		`data-gosx-studio-board-slot="true"`,
		`data-gosx-studio-right-rail-slot="true"`,
		`data-gosx-studio-home-layers-panel-renderer="gosx-studio"`,
		`data-gosx-studio-block-layout-engine-renderer="gosx-studio"`,
		`data-gosx-studio-brand-panel-renderer="gosx-studio"`,
		`data-gosx-studio-brand-media-picker-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-panel-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor panel stack missing %q:\n%s", fragment, html)
		}
	}

	for _, fragment := range []string{
		`<header class="studio-panel-heading"><div><p class="kicker">Home</p><h2>Sections</h2></div><select data-home-layer-selection="true"></select></header>`,
		`<div class="studio-block-layout-engine__layers" data-studio-block-layout-layers="true"><section class="" data-studio-mode-panel="" data-studio-engine-source="gosx" data-gosx-studio-home-layers-panel-renderer="gosx-studio">`,
		`<div class="studio-block-layout-engine__library" data-studio-block-layout-library="true"><section data-block-library="true">Library</section></div>`,
		`<section class="studio-brand-panel__group-slot studio-brand-media-picker-slot" data-studio-brand-group-slot="files" data-studio-brand-group-selected="false"><section class="studio-brand-media-picker"`,
		`data-studio-media-picker-island="brand"`,
		`data-studio-media-asset="asset-logo"`,
		`data-gosx-studio-flow-designer-panel-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-tools-panel-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-settings-panel-renderer="gosx-studio"`,
		`data-workspace-field-list="true"`,
		`data-calendar-field-list="true"`,
		`data-type-field-list="true"`,
		`data-settings-field-list="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor panel stack did not nest %q:\n%s", fragment, html)
		}
	}
	if got := strings.Count(html, `data-gosx-studio-advanced-field-panel-renderer="gosx-studio"`); got != 3 {
		t.Fatalf("backend editor panel stack should render three Studio advanced field panels, got %d:\n%s", got, html)
	}

	assertOrder(t, html,
		`data-gosx-studio-left-rail-slot="true"`,
		`data-site-navigator="true"`,
		`data-gosx-studio-block-layout-engine-renderer="gosx-studio"`,
		`data-studio-block-layout-layers="true"`,
		`data-home-layer-selection="true"`,
		`data-studio-block-layout-library="true"`,
		`data-block-library="true"`,
		`data-gosx-studio-board-slot="true"`,
		`data-site-map-engine="true"`,
		`data-site-map-canvas="true"`,
		`data-gosx-studio-right-rail-slot="true"`,
		`data-gosx-studio-brand-panel-renderer="gosx-studio"`,
		`data-gosx-studio-brand-media-picker-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-panel-renderer="gosx-studio"`,
		`data-gosx-studio-flow-designer-panel-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-tools-panel-renderer="gosx-studio"`,
		`data-workspace-field-list="true"`,
		`data-calendar-field-list="true"`,
		`data-type-field-list="true"`,
		`data-gosx-studio-advanced-settings-panel-renderer="gosx-studio"`,
	)
}

func TestRenderBackendEditorWorkbenchPanelStackKeepsExplicitBrandMediaPickerOverride(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Website editor"}, WorkbenchShellViewOptions{})
	brandPanel := map[string]any{
		"kicker":          "Brand",
		"title":           "Identity",
		"summary":         "Brand system",
		"previewLabel":    "Preview",
		"groupLabel":      "Brand groups",
		"defaultGroupKey": "files",
		"groups": []map[string]any{{
			"key":      "files",
			"inputID":  "brandFiles",
			"label":    "Files",
			"summary":  "Media",
			"selected": true,
		}},
	}
	brandFields := map[string]any{
		"preview": map[string]any{
			"cornerLabel": "Logo",
			"buttonLabel": "Adjust logo",
			"logoURL":     "/logo.png",
			"logoAlt":     "Logo",
		},
	}

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                  view,
		SiteNavigator:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:            map[string]any{"kicker": "Home", "title": "Sections"},
		BlockLayoutEngineHost: gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		BrandPanel:            brandPanel,
		BrandFields:           brandFields,
		BrandMediaPicker:      gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-brand-media-picker", "true")), gosx.Text("Custom media")),
		BrandMediaPickerView: map[string]any{
			"kicker":     "Assets",
			"title":      "Brand media",
			"summary":    "Choose brand assets.",
			"uploadHref": "/admin/media",
		},
	}))

	if !strings.Contains(html, `data-custom-brand-media-picker="true"`) {
		t.Fatalf("explicit brand media picker override missing:\n%s", html)
	}
	if strings.Contains(html, `data-gosx-studio-brand-media-picker-renderer="gosx-studio"`) {
		t.Fatalf("explicit brand media picker override should not render default picker:\n%s", html)
	}
}

func TestRenderBackendEditorWorkbenchPanelStackKeepsExplicitAdvancedGroupOverrides(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Website editor"}, WorkbenchShellViewOptions{})
	advancedPanel := map[string]any{
		"kicker":          "Advanced",
		"title":           "Controls",
		"summary":         "Advanced controls",
		"groupLabel":      "Advanced groups",
		"defaultGroupKey": "flows",
		"groups": []map[string]any{
			{"key": "flows", "inputID": "advancedFlows", "label": "Flows", "summary": "Automation", "selected": true},
			{"key": "tools", "inputID": "advancedTools", "label": "Tools", "summary": "Tools"},
			{"key": "schema", "inputID": "advancedSchema", "label": "Schema", "summary": "Workspace"},
			{"key": "schedule", "inputID": "advancedSchedule", "label": "Schedule", "summary": "Calendar"},
			{"key": "typography", "inputID": "advancedType", "label": "Type", "summary": "Typography"},
			{"key": "settings", "inputID": "advancedSettings", "label": "SEO", "summary": "Settings"},
		},
	}

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                            view,
		SiteNavigator:                   gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:                      map[string]any{"kicker": "Home", "title": "Sections"},
		BlockLayoutEngineHost:           gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		AdvancedPanel:                   advancedPanel,
		FlowDesigner:                    gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-flow-designer", "true")), gosx.Text("Custom flows")),
		FlowDesignerView:                map[string]any{"defaultFlowKey": "contact"},
		AdvancedToolsPanel:              gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-advanced-tools", "true")), gosx.Text("Custom tools")),
		AdvancedToolsPanelView:          map[string]any{"class": "editor-panel editor-panel--advanced-tools"},
		AdvancedWorkspaceFieldPanel:     gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-advanced-workspace", "true")), gosx.Text("Custom workspace")),
		AdvancedWorkspaceFieldPanelView: map[string]any{"class": "editor-panel editor-panel--advanced-fields"},
		AdvancedCalendarFieldPanel:      gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-advanced-calendar", "true")), gosx.Text("Custom calendar")),
		AdvancedCalendarFieldPanelView:  map[string]any{"class": "editor-panel editor-panel--advanced-fields"},
		AdvancedTypeFieldPanel:          gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-advanced-type", "true")), gosx.Text("Custom type")),
		AdvancedTypeFieldPanelView:      map[string]any{"class": "editor-panel editor-panel--advanced-fields"},
		AdvancedSettingsPanel:           gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-advanced-settings", "true")), gosx.Text("Custom settings")),
		AdvancedSettingsPanelView:       map[string]any{"class": "editor-panel editor-panel--advanced-settings"},
	}))

	for _, fragment := range []string{
		`data-custom-flow-designer="true"`,
		`data-custom-advanced-tools="true"`,
		`data-custom-advanced-workspace="true"`,
		`data-custom-advanced-calendar="true"`,
		`data-custom-advanced-type="true"`,
		`data-custom-advanced-settings="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("explicit advanced group override missing %q:\n%s", fragment, html)
		}
	}
	for _, fragment := range []string{
		`data-gosx-studio-flow-designer-panel-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-tools-panel-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-field-panel-renderer="gosx-studio"`,
		`data-gosx-studio-advanced-settings-panel-renderer="gosx-studio"`,
	} {
		if strings.Contains(html, fragment) {
			t.Fatalf("explicit advanced group override should not render default panel %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendEditorWorkbenchPanelStackOwnsPublishPanelComposition(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title:      "Website editor",
		SaveAction: "/admin/editor/__actions/save",
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero selected",
			Zoom:           "100",
		},
	}, WorkbenchShellViewOptions{
		CSRFToken: "csrf-token",
	})
	publishPanel := publishPanelTestView()
	previewShare := map[string]any{
		"class":        "studio-preview-share",
		"kicker":       "Preview",
		"title":        "Share draft",
		"status":       "Ready",
		"state":        "ready",
		"detail":       "Copy a signed preview for review.",
		"inputLabel":   "Signed preview URL",
		"copyLabel":    "Copy",
		"openLabel":    "Open preview",
		"href":         "https://noni.example.test/?preview=token",
		"hasHref":      true,
		"resource":     "settings/site",
		"audience":     "reviewer",
		"expiresLabel": "Jun 28, 2026 10:00 UTC",
	}
	activityPanel := map[string]any{
		"class":         "studio-activity-drawer",
		"kicker":        "Activity",
		"title":         "Readiness",
		"score":         "1/1 ready",
		"toggleLabel":   "Hide",
		"togglePressed": true,
		"readiness": map[string]any{
			"readyCount": "1",
			"watchCount": "0",
			"nextCount":  "0",
			"items": []map[string]any{{
				"key":         "content",
				"class":       "studio-readiness-card studio-readiness-card--ready",
				"label":       "Content",
				"summary":     "Homepage ready",
				"statusLabel": "Ready",
			}},
		},
		"comments": map[string]any{
			"class":      "studio-comments",
			"kicker":     "Discuss",
			"title":      "Comments",
			"countLabel": "0 open",
			"emptyTitle": "No comments",
			"empty":      "Review notes will appear here.",
		},
		"proposals": map[string]any{
			"class":      "studio-proposals",
			"kicker":     "Review",
			"title":      "Proposals",
			"countLabel": "0 pending",
			"emptyTitle": "No proposals",
			"empty":      "Suggestions will appear here.",
		},
	}
	revisionHistory := map[string]any{
		"class":       "editor-panel",
		"panelKey":    "versions",
		"headerClass": "editor-panel__header",
		"title":       "Version history",
		"empty":       "No previous versions yet.",
		"hasItems":    false,
	}

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                  view,
		AuthoringURL:          "/admin/editor/__actions/authoring",
		SiteNavigator:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:            map[string]any{"kicker": "Home", "title": "Sections", "empty": "No sections."},
		HomeLayerSelection:    gosx.El("select", gosx.Attrs(gosx.Attr("data-home-layer-selection", "true"))),
		BlockLayoutEngineHost: gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		BlockLibraryPanel:     gosx.El("section", gosx.Attrs(gosx.Attr("data-block-library", "true")), gosx.Text("Library")),
		SiteMapEngine:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-engine", "true")), gosx.Text("Site map")),
		SiteMapCanvas:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-canvas", "true")), gosx.Text("Canvas")),
		InspectorChrome:       gosx.El("section", gosx.Attrs(gosx.Attr("data-inspector-chrome", "true")), gosx.Text("Inspector")),
		PublishPanelView:      publishPanel,
		PreviewSharePanel:     previewShare,
		ActivityPanel:         activityPanel,
		RevisionHistory:       revisionHistory,
	}))

	for _, fragment := range []string{
		`data-gosx-studio-publish-panel-renderer="gosx-studio"`,
		`data-gosx-studio-preview-share-renderer="gosx-studio"`,
		`data-gosx-studio-activity-panel-renderer="gosx-studio"`,
		`data-gosx-studio-revision-history-renderer="gosx-studio"`,
		`data-studio-publish-preview-share-slot="true"><section class="studio-preview-share"`,
		`data-studio-publish-activity-slot="true"><section class="studio-activity-drawer"`,
		`data-studio-publish-history-slot="true"><section class="editor-panel"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor publish stack missing %q:\n%s", fragment, html)
		}
	}
	assertOrder(t, html,
		`data-gosx-studio-publish-panel-renderer="gosx-studio"`,
		`data-gosx-studio-preview-share-renderer="gosx-studio"`,
		`data-gosx-studio-activity-panel-renderer="gosx-studio"`,
		`data-gosx-studio-revision-history-renderer="gosx-studio"`,
	)
}

func TestRenderBackendEditorWorkbenchPanelStackOwnsNavigationAndCheckoutComposition(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Website editor"}, WorkbenchShellViewOptions{})
	navigationPanel := map[string]any{
		"class":       "panel editor-panel editor-panel--navigation studio-navigation-panel",
		"panelKey":    "navigation",
		"headerClass": "editor-panel__header",
		"kicker":      "Structure",
		"title":       "Navigation",
		"help":        "Choose storefront links.",
		"empty":       "No links yet.",
		"hasItems":    false,
	}
	checkoutPanel := map[string]any{
		"class":          "panel editor-panel editor-panel--checkout studio-checkout-panel",
		"panelKey":       "checkout",
		"headerClass":    "editor-panel__header",
		"kicker":         "Commerce",
		"title":          "Checkout",
		"help":           "Choose checkout behavior.",
		"enabledName":    "checkoutEnabled",
		"enabledValue":   "true",
		"enabledLabel":   "Enable checkout",
		"labelID":        "checkoutLabel",
		"labelName":      "checkoutLabel",
		"labelFieldText": "Button label",
		"buttonLabel":    "Save checkout",
		"empty":          "No products yet.",
		"hasProducts":    false,
	}

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                  view,
		SiteNavigator:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:            map[string]any{"kicker": "Home", "title": "Sections"},
		BlockLayoutEngineHost: gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		NavigationPanelView:   navigationPanel,
		CheckoutPanelView:     checkoutPanel,
	}))

	for _, fragment := range []string{
		`data-gosx-studio-navigation-panel-renderer="gosx-studio"`,
		`data-gosx-studio-checkout-panel-renderer="gosx-studio"`,
		`data-studio-navigation-panel="true"`,
		`data-studio-checkout-panel="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor panel stack missing Studio-owned panel fragment %q:\n%s", fragment, html)
		}
	}
	assertOrder(t, html,
		`data-gosx-studio-navigation-panel-renderer="gosx-studio"`,
		`data-gosx-studio-checkout-panel-renderer="gosx-studio"`,
	)
}

func TestRenderBackendEditorWorkbenchPanelStackOwnsBlockLookAndHomeInspectorComposition(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Website editor"}, WorkbenchShellViewOptions{})

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                  view,
		SiteNavigator:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:            map[string]any{"kicker": "Home", "title": "Sections"},
		BlockLayoutEngineHost: gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		BlockLibraryPanelView: map[string]any{
			"class":     "editor-panel editor-panel--library",
			"key":       "blocks",
			"mode":      "home",
			"title":     "Sections",
			"listClass": "editor-block-library",
			"hasItems":  false,
			"empty":     "No sections are available.",
		},
		HomeInspectorView: map[string]any{
			"class":           "editor-panel editor-panel--home-inspector studio-home-inspector",
			"id":              "studioHomeInspector",
			"panelKey":        "inspector",
			"mode":            "home",
			"defaultGroupKey": "copy",
			"groups": []map[string]any{{
				"key":      "copy",
				"label":    "Copy",
				"selected": true,
			}},
		},
		HomeInspectorContentFields: map[string]any{"fields": []map[string]any{}, "hasFields": false},
		LookPanelView: map[string]any{
			"class":           "editor-panel editor-panel--look studio-look-panel",
			"panelKey":        "look",
			"defaultGroupKey": "theme",
			"groups": []map[string]any{{
				"key":      "theme",
				"label":    "Theme",
				"selected": true,
			}},
		},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-block-library-panel-renderer="gosx-studio"`,
		`data-gosx-studio-look-panel-renderer="gosx-studio"`,
		`data-gosx-studio-home-inspector-panel-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor panel stack missing Studio-owned default panel %q:\n%s", fragment, html)
		}
	}
	assertOrder(t, html,
		`data-gosx-studio-block-layout-engine-renderer="gosx-studio"`,
		`data-gosx-studio-block-library-panel-renderer="gosx-studio"`,
		`data-gosx-studio-home-inspector-panel-renderer="gosx-studio"`,
		`data-gosx-studio-look-panel-renderer="gosx-studio"`,
	)
}

func TestRenderBackendEditorWorkbenchPanelStackKeepsExplicitNavigationAndCheckoutOverrides(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Website editor"}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                  view,
		SiteNavigator:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:            map[string]any{"kicker": "Home", "title": "Sections"},
		BlockLayoutEngineHost: gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		NavigationPanel:       gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-navigation-panel", "true")), gosx.Text("Custom navigation")),
		NavigationPanelView:   map[string]any{"class": "studio-navigation-panel", "panelKey": "navigation"},
		CheckoutPanel:         gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-checkout-panel", "true")), gosx.Text("Custom checkout")),
		CheckoutPanelView:     map[string]any{"class": "studio-checkout-panel", "panelKey": "checkout"},
	}))

	for _, fragment := range []string{
		`data-custom-navigation-panel="true"`,
		`data-custom-checkout-panel="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("explicit panel override missing %q:\n%s", fragment, html)
		}
	}
	for _, fragment := range []string{
		`data-gosx-studio-navigation-panel-renderer="gosx-studio"`,
		`data-gosx-studio-checkout-panel-renderer="gosx-studio"`,
	} {
		if strings.Contains(html, fragment) {
			t.Fatalf("explicit panel override should not render default panel %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendEditorWorkbenchPanelStackKeepsExplicitBlockLookAndHomeInspectorOverrides(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Website editor"}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                       view,
		SiteNavigator:              gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:                 map[string]any{"kicker": "Home", "title": "Sections"},
		BlockLayoutEngineHost:      gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		BlockLibraryPanel:          gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-block-library-panel", "true")), gosx.Text("Custom library")),
		BlockLibraryPanelView:      map[string]any{"class": "editor-panel editor-panel--library", "key": "blocks"},
		HomeInspector:              gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-home-inspector-panel", "true")), gosx.Text("Custom inspector")),
		HomeInspectorView:          map[string]any{"class": "studio-home-inspector", "panelKey": "inspector"},
		HomeInspectorContentFields: map[string]any{"hasFields": false},
		LookPanel:                  gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-look-panel", "true")), gosx.Text("Custom look")),
		LookPanelView:              map[string]any{"class": "studio-look-panel", "panelKey": "look"},
	}))

	for _, fragment := range []string{
		`data-custom-block-library-panel="true"`,
		`data-custom-home-inspector-panel="true"`,
		`data-custom-look-panel="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("explicit panel override missing %q:\n%s", fragment, html)
		}
	}
	for _, fragment := range []string{
		`data-gosx-studio-block-library-panel-renderer="gosx-studio"`,
		`data-gosx-studio-look-panel-renderer="gosx-studio"`,
		`data-gosx-studio-home-inspector-panel-renderer="gosx-studio"`,
	} {
		if strings.Contains(html, fragment) {
			t.Fatalf("explicit panel override should not render default panel %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendEditorWorkbenchPanelStackKeepsExplicitPublishPanelOverride(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Website editor"}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                  view,
		SiteNavigator:         gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:            map[string]any{"kicker": "Home", "title": "Sections"},
		BlockLayoutEngineHost: gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		PublishPanel:          gosx.El("section", gosx.Attrs(gosx.Attr("data-custom-publish-panel", "true")), gosx.Text("Custom publish")),
		PublishPanelView:      publishPanelTestView(),
		PreviewSharePanel: map[string]any{
			"class":   "studio-preview-share",
			"hasHref": false,
		},
	}))

	if !strings.Contains(html, `data-custom-publish-panel="true"`) {
		t.Fatalf("explicit publish panel override missing:\n%s", html)
	}
	if strings.Contains(html, `data-gosx-studio-publish-panel-renderer="gosx-studio"`) {
		t.Fatalf("explicit publish panel override should not render default publish stack:\n%s", html)
	}
}

func TestRenderBackendEditorWorkbenchDefaultChrome(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title:      "Website editor",
		SaveAction: "/admin/editor/__actions/save",
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero selected",
			Zoom:           "100",
		},
		Modes: []Mode{NewMode("home", "Home", true)},
		Metrics: []Metric{{
			Key:   "sections",
			Label: "sections",
			Value: "4",
		}},
		Viewports: []Viewport{
			NewViewport("desktop", "Desktop", "100%", false),
			NewViewport("tablet", "Tablet", "48rem", true),
		},
	}, WorkbenchShellViewOptions{
		LeftRailLabel:  "Pages",
		RightRailLabel: "Fields",
		ActivityLabel:  "Timeline",
		FocusLabel:     "Inspect",
		CSRFToken:      "csrf-token",
	})

	html := gosx.RenderHTML(RenderBackendEditorWorkbench(BackendEditorWorkbenchProps{
		View:         view,
		AuthoringURL: "/admin/editor/__actions/authoring",
		LeftRail:     []gosx.Node{gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages"))},
		Board:        []gosx.Node{gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-engine", "true")), gosx.Text("Site map"))},
		RightRail:    []gosx.Node{gosx.El("section", gosx.Attrs(gosx.Attr("data-inspector", "true")), gosx.Text("Inspector"))},
	}))

	for _, fragment := range []string{
		`class="editor-toolbar"`,
		`class="button-row" data-gosx-studio-toolbar-actions="true"`,
		`class="studio-modebar"`,
		`class="studio-context-strip"`,
		`class="studio-command-palette"`,
		`class="editor-save-status"`,
		`class="editor-save-state"`,
		`class="editor-save-detail"`,
		`class="editor-save-time"`,
		`class="editor-save-count"`,
		`data-gosx-studio-save-button="true"`,
		`class="studio-canvas-bar"`,
		`data-gosx-studio-canvas-bar-renderer="gosx-studio"`,
		`class="studio-canvas-tools"`,
		`data-gosx-studio-canvas-tools-renderer="gosx-studio"`,
		`class="studio-viewport-switcher"`,
		`data-gosx-studio-viewport-controls-renderer="gosx-studio"`,
		`class="studio-zoombar"`,
		`data-gosx-studio-zoom-controls-renderer="gosx-studio"`,
		`class="studio-canvas-status"`,
		`data-gosx-studio-canvas-status-renderer="gosx-studio"`,
		`data-studio-mode-control="home"`,
		`data-studio-metric="sections"`,
		`data-studio-command-palette="true"`,
		`data-studio-rail-toggle="left" aria-pressed="true">Pages</button>`,
		`data-studio-viewport-current="tablet"`,
		`data-studio-zoom-current="100"`,
		`<output data-studio-selection-label="true">Hero selected</output>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor default chrome missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendEditorWorkbenchSlotHelpers(t *testing.T) {
	left := gosx.RenderHTML(RenderBackendEditorLeftRail(gosx.El("span", nil, gosx.Text("left"))))
	board := gosx.RenderHTML(RenderBackendEditorBoard(gosx.El("span", nil, gosx.Text("board"))))
	right := gosx.RenderHTML(RenderBackendEditorRightRail(gosx.El("span", nil, gosx.Text("right"))))

	for html, want := range map[string]string{
		left:  `data-gosx-studio-left-rail-slot="true"`,
		board: `data-gosx-studio-board-slot="true"`,
		right: `data-gosx-studio-right-rail-slot="true"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("slot helper missing %q: %s", want, html)
		}
	}
}
