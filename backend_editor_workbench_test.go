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
		},
	}

	html := gosx.RenderHTML(RenderBackendEditorWorkbenchPanelStack(BackendEditorWorkbenchPanelStackProps{
		View:                        view,
		AuthoringURL:                "/admin/editor/__actions/authoring",
		Toolbar:                     gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-toolbar", "true")), gosx.Text("Toolbar")),
		CanvasBar:                   gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-bar", "true")), gosx.Text("Canvas bar")),
		CanvasStatus:                gosx.El("div", gosx.Attrs(gosx.Attr("data-editor-canvas-status", "true")), gosx.Text("Ready")),
		SiteNavigator:               gosx.El("section", gosx.Attrs(gosx.Attr("data-site-navigator", "true")), gosx.Text("Pages")),
		HomeLayers:                  map[string]any{"kicker": "Home", "title": "Sections", "empty": "No sections."},
		HomeLayerSelection:          gosx.El("select", gosx.Attrs(gosx.Attr("data-home-layer-selection", "true"))),
		BlockLayoutEngineHost:       gosx.El("div", gosx.Attrs(gosx.Attr("data-block-layout-host", "true"))),
		BlockLibraryPanel:           gosx.El("section", gosx.Attrs(gosx.Attr("data-block-library", "true")), gosx.Text("Library")),
		SiteMapEngine:               gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-engine", "true")), gosx.Text("Site map")),
		SiteMapCanvas:               gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map-canvas", "true")), gosx.Text("Canvas")),
		InspectorChrome:             gosx.El("section", gosx.Attrs(gosx.Attr("data-inspector-chrome", "true")), gosx.Text("Inspector")),
		HomeInspector:               gosx.El("section", gosx.Attrs(gosx.Attr("data-home-inspector", "true")), gosx.Text("Home")),
		LookPanel:                   gosx.El("section", gosx.Attrs(gosx.Attr("data-look-panel", "true")), gosx.Text("Look")),
		BrandPanel:                  brandPanel,
		BrandFields:                 brandFields,
		BrandMediaPicker:            gosx.El("section", gosx.Attrs(gosx.Attr("data-brand-media-picker", "true")), gosx.Text("Media")),
		NavigationPanel:             gosx.El("section", gosx.Attrs(gosx.Attr("data-navigation-panel", "true")), gosx.Text("Navigation")),
		CheckoutPanel:               gosx.El("section", gosx.Attrs(gosx.Attr("data-checkout-panel", "true")), gosx.Text("Checkout")),
		PublishPanel:                gosx.El("section", gosx.Attrs(gosx.Attr("data-publish-panel", "true")), gosx.Text("Publish")),
		AdvancedPanel:               advancedPanel,
		FlowDesigner:                gosx.El("section", gosx.Attrs(gosx.Attr("data-flow-designer", "true")), gosx.Text("Flows")),
		AdvancedToolsPanel:          gosx.El("section", gosx.Attrs(gosx.Attr("data-advanced-tools", "true")), gosx.Text("Tools")),
		AdvancedWorkspaceFieldPanel: gosx.El("section", gosx.Attrs(gosx.Attr("data-advanced-workspace", "true")), gosx.Text("Workspace")),
		AdvancedCalendarFieldPanel:  gosx.El("section", gosx.Attrs(gosx.Attr("data-advanced-calendar", "true")), gosx.Text("Calendar")),
		AdvancedTypeFieldPanel:      gosx.El("section", gosx.Attrs(gosx.Attr("data-advanced-type", "true")), gosx.Text("Type")),
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
		`<section class="studio-brand-panel__group-slot studio-brand-media-picker-slot" data-studio-brand-group-slot="files" data-studio-brand-group-selected="false"><section data-brand-media-picker="true">Media</section></section>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="flows"><section data-flow-designer="true">Flows</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="tools"><section data-advanced-tools="true">Tools</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="schema"><section data-advanced-workspace="true">Workspace</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="schedule"><section data-advanced-calendar="true">Calendar</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="typography"><section data-advanced-type="true">Type</section></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("backend editor panel stack did not nest %q:\n%s", fragment, html)
		}
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
		`data-brand-media-picker="true"`,
		`data-gosx-studio-advanced-panel-renderer="gosx-studio"`,
		`data-flow-designer="true"`,
		`data-advanced-tools="true"`,
		`data-advanced-workspace="true"`,
		`data-advanced-calendar="true"`,
		`data-advanced-type="true"`,
	)
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
