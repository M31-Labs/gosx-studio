package shell

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderWorkbenchToolbarUsesShellView(t *testing.T) {
	shell := New(Options{
		Title:      "Pajaritos Website",
		PreviewURL: "/",
		SaveAction: "/admin/editor/__actions/save",
		Metrics: []Metric{
			NewMetric("sections", "Sections", 4),
		},
		Modes: []Mode{
			NewMode("home", "Home", true),
			NewMode("preview", "Preview", false),
		},
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "4 editable sections",
		},
	})
	view := WorkbenchShellViewForShell(shell, WorkbenchShellViewOptions{
		ToolbarKicker: "Website",
		PreviewLabel:  "Preview site",
		SaveLabel:     "Save changes",
	})

	html := gosx.RenderHTML(RenderWorkbenchToolbar(view, WorkbenchToolbarOptions{
		Class:        "studio-toolbar",
		ActionsClass: "studio-toolbar__actions",
		Summary:      "4 editable sections / 3 family forms",
		CommandPaletteNode: gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-command-open", "true"),
		), gosx.Text("Commands")),
		Controls: []gosx.Node{
			RenderWorkbenchModebar(view, WorkbenchModebarOptions{Class: "studio-modebar"}),
			RenderWorkbenchMetricStrip(view, WorkbenchMetricStripOptions{Class: "studio-context-strip"}),
		},
		SaveStatusNode: RenderWorkbenchSaveStatus(WorkbenchSaveStatusOptions{
			Class:           "studio-save-status",
			StateClass:      "studio-save-state",
			DetailClass:     "studio-save-detail",
			LastSavedClass:  "studio-save-time",
			DirtyCountClass: "studio-save-count",
		}),
	}))

	for _, want := range []string{
		`class="studio-toolbar"`,
		`data-studio-toolbar="true"`,
		`data-gosx-studio-toolbar-renderer="gosx-studio"`,
		`data-gosx-studio-toolbar-actions="true"`,
		"Website",
		"Pajaritos Website",
		"4 editable sections / 3 family forms",
		`class="studio-modebar"`,
		`data-studio-mode-control="home"`,
		`aria-pressed="true"`,
		`class="studio-context-strip"`,
		`data-studio-metric="sections"`,
		`data-studio-command-open="true"`,
		`data-gosx-studio-save-status="true"`,
		`data-gosx-studio-history-controls="true"`,
		`data-gosx-studio-preview-link="true"`,
		`href="/"`,
		"Preview site",
		`data-gosx-studio-save-button="true"`,
		"Save changes",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in toolbar html: %s", want, html)
		}
	}
	if commandIndex, actionsIndex := strings.Index(html, `data-studio-command-open="true"`), strings.Index(html, `data-gosx-studio-toolbar-actions="true"`); commandIndex == -1 || actionsIndex == -1 || commandIndex > actionsIndex {
		t.Fatalf("expected command palette to render as top-level toolbar chrome before actions: %s", html)
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("host toolbar should not inject product copy into client UI: %s", html)
	}
}

// TestRenderWorkbenchToolbarTitleSummaryIsReactiveSelectionReadout is the
// regression test for HANDOFF-19 (owner report: clicking anywhere in the
// center canvas never visibly selects anything — the editor's most prominent
// selection status, next to the "Website editor" title, read "No selection"
// the whole time).
//
// Root cause: the toolbar title's summary node (fed by selectionLabel,
// falling back to routeLabel — see RenderWorkbenchToolbar above) was rendered
// as a completely bare <span> with no attributes at all. Every OTHER
// selection-label readout in this file (RenderWorkbenchCanvasBar,
// RenderWorkbenchCanvasStatus) and in shell/workbench_page_canvas.go carries
// data-studio-selection-label="true", which
// hostruntime/assets/workbench_runtime.js's setReadout() targets via
// queryAll(form, "[data-studio-selection-label]") on every real selection
// change. Lacking that attribute, the toolbar title span was invisible to
// that update path and stayed frozen at whatever string the server baked in
// on the FIRST render (in muddy-noni-commerce, literally the hardcoded
// string "No selection" — see app/admin/editor/page.server.go's
// editorStudioShell) no matter how many real, successful clicks the user
// made on the canvas underneath it.
//
// This test asserts the summary span now carries that attribute, so a
// regression here (dropping the attribute again) fails loudly at the render
// layer without needing a live browser.
func TestRenderWorkbenchToolbarTitleSummaryIsReactiveSelectionReadout(t *testing.T) {
	shell := New(Options{
		Title:      "Pajaritos Website",
		PreviewURL: "/",
		SaveAction: "/admin/editor/__actions/save",
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero",
		},
	})
	view := WorkbenchShellViewForShell(shell, WorkbenchShellViewOptions{ToolbarKicker: "Website"})

	html := gosx.RenderHTML(RenderWorkbenchToolbar(view, WorkbenchToolbarOptions{}))

	want := `<span data-studio-selection-label="true">Hero</span>`
	if !strings.Contains(html, want) {
		t.Fatalf("expected reactive selection-label span %q in toolbar title html, got: %s", want, html)
	}
	if strings.Contains(html, "<span>Hero</span>") {
		t.Fatalf("toolbar title summary span rendered WITHOUT a data-studio-selection-label hook — client-side setReadout() can never update it, reproducing the frozen 'No selection' defect: %s", html)
	}
}

func TestRenderWorkbenchCommandPaletteUsesShellViewCommands(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title: "Client Site",
	}, WorkbenchShellViewOptions{
		CommandLauncher:   "Find",
		CommandTitle:      "Editor tools",
		CommandSearchHint: "Find editing areas",
		Commands: []map[string]any{{
			"key":         "preview",
			"label":       "Preview",
			"summary":     "Open preview",
			"group":       "Navigate",
			"kind":        "link",
			"target":      "preview",
			"href":        "/preview",
			"shortcut":    "P",
			"hasSummary":  true,
			"hasShortcut": true,
			"searchText":  "preview open",
		}},
	})
	html := gosx.RenderHTML(RenderWorkbenchCommandPalette(view, WorkbenchCommandPaletteOptions{}))
	for _, want := range []string{
		`data-gosx-studio-command-renderer="gosx-studio"`,
		`data-studio-command-open="true"`,
		`aria-label="Editor tools"`,
		`placeholder="Find editing areas"`,
		`data-studio-command="preview"`,
		`data-studio-command-kind="link"`,
		`data-studio-command-href="/preview"`,
		"Open preview",
		"<kbd>P</kbd>",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in command palette html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchZoomControlsUsesDefaultShellView(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{
		Zoom: "100",
	})
	html := gosx.RenderHTML(RenderWorkbenchZoomControls(view, WorkbenchZoomControlsOptions{}))
	for _, want := range []string{
		`class="studio-zoombar"`,
		`role="toolbar"`,
		`aria-label="Canvas zoom"`,
		`data-studio-zoom-island="true"`,
		`data-studio-zoom-current="100"`,
		`data-gosx-studio-zoom-controls-renderer="gosx-studio"`,
		`type="button" data-studio-zoom="fit" aria-pressed="false">Fit</button>`,
		`type="button" data-studio-zoom="75" aria-pressed="false">75%</button>`,
		`type="button" data-studio-zoom="100" aria-pressed="true">100%</button>`,
		`type="button" data-studio-zoom="125" aria-pressed="false">125%</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in zoom controls html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchZoomControlsHonorsCustomLevelsAndClass(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{
		Zoom: "fit",
		ZoomLevels: []WorkbenchZoomLevel{
			{Key: "fit", Label: "Fit"},
			{Key: "200", Label: "200%", Active: true},
		},
	})
	html := gosx.RenderHTML(RenderWorkbenchZoomControls(view, WorkbenchZoomControlsOptions{
		Class: "custom-zoombar",
		Label: "Preview zoom",
	}))
	for _, want := range []string{
		`class="custom-zoombar"`,
		`aria-label="Preview zoom"`,
		`data-studio-zoom-current="200"`,
		`data-gosx-studio-zoom-controls-renderer="gosx-studio"`,
		`data-studio-zoom="fit" aria-pressed="false">Fit</button>`,
		`data-studio-zoom="200" aria-pressed="true">200%</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in custom zoom controls html: %s", want, html)
		}
	}
	if strings.Contains(html, `data-studio-zoom="100"`) {
		t.Fatalf("custom zoom controls should not render default levels: %s", html)
	}
}

func TestRenderWorkbenchViewportControlsUsesDefaultShellView(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{
		ViewportKey: "tablet",
	})
	html := gosx.RenderHTML(RenderWorkbenchViewportControls(view, WorkbenchViewportControlsOptions{}))
	for _, want := range []string{
		`class="studio-viewport-switcher"`,
		`role="toolbar"`,
		`aria-label="Preview viewport"`,
		`data-studio-viewport-current="tablet"`,
		`data-gosx-studio-viewport-controls-renderer="gosx-studio"`,
		`type="button" data-studio-viewport="desktop" data-studio-viewport-width="100%" aria-pressed="false">Desktop</button>`,
		`type="button" data-studio-viewport="tablet" data-studio-viewport-width="48rem" aria-pressed="true">Tablet</button>`,
		`type="button" data-studio-viewport="mobile" data-studio-viewport-width="24rem" aria-pressed="false">Mobile</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in viewport controls html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchViewportControlsHonorsCustomViewportsAndClass(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title: "Client Site",
		Viewports: []Viewport{
			NewViewport("desktop", "Desktop", "100%", false),
			NewViewport("wide", "Wide", "72rem", true),
		},
	}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderWorkbenchViewportControls(view, WorkbenchViewportControlsOptions{
		Class: "custom-viewportbar",
		Label: "Preview size",
	}))
	for _, want := range []string{
		`class="custom-viewportbar"`,
		`aria-label="Preview size"`,
		`data-studio-viewport-current="wide"`,
		`data-gosx-studio-viewport-controls-renderer="gosx-studio"`,
		`data-studio-viewport="desktop" data-studio-viewport-width="100%" aria-pressed="false">Desktop</button>`,
		`data-studio-viewport="wide" data-studio-viewport-width="72rem" aria-pressed="true">Wide</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in custom viewport controls html: %s", want, html)
		}
	}
	if strings.Contains(html, `data-studio-viewport="mobile"`) {
		t.Fatalf("custom viewport controls should not render default viewports: %s", html)
	}
}

func TestRenderWorkbenchCanvasToolsUsesDefaultShellView(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderWorkbenchCanvasTools(view, WorkbenchCanvasToolsOptions{}))
	for _, want := range []string{
		`class="studio-canvas-tools"`,
		`role="toolbar"`,
		`aria-label="Canvas tools"`,
		`data-gosx-studio-canvas-tools-renderer="gosx-studio"`,
		`type="button" data-studio-rail-toggle="left" aria-pressed="true">Layers</button>`,
		`type="button" data-studio-rail-toggle="right" aria-pressed="true">Inspector</button>`,
		`type="button" data-studio-activity-toggle="true" aria-pressed="true">Activity</button>`,
		`type="button" data-studio-focus-toggle="true" aria-pressed="false">Focus</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in canvas tools html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchCanvasToolsHonorsShellLabelsAndClass(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{
		LeftRailLabel:  "Pages",
		RightRailLabel: "Fields",
		ActivityLabel:  "Timeline",
		FocusLabel:     "Inspect",
	})
	html := gosx.RenderHTML(RenderWorkbenchCanvasTools(view, WorkbenchCanvasToolsOptions{
		Class: "custom-canvas-tools",
		Label: "Editor tools",
	}))
	for _, want := range []string{
		`class="custom-canvas-tools"`,
		`aria-label="Editor tools"`,
		`data-gosx-studio-canvas-tools-renderer="gosx-studio"`,
		`data-studio-rail-toggle="left" aria-pressed="true">Pages</button>`,
		`data-studio-rail-toggle="right" aria-pressed="true">Fields</button>`,
		`data-studio-activity-toggle="true" aria-pressed="true">Timeline</button>`,
		`data-studio-focus-toggle="true" aria-pressed="false">Inspect</button>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in custom canvas tools html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchCanvasBarUsesDefaultShellView(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{
		RouteLabel:     "Home",
		SelectionLabel: "Hero selected",
	})
	html := gosx.RenderHTML(RenderWorkbenchCanvasBar(view, WorkbenchCanvasBarOptions{
		Controls: []gosx.Node{
			gosx.El("span", gosx.Attrs(gosx.Attr("data-test-control", "one")), gosx.Text("Control")),
		},
	}))
	for _, want := range []string{
		`class="studio-canvas-bar"`,
		`data-gosx-studio-canvas-bar-renderer="gosx-studio"`,
		`<p class="kicker">Canvas</p>`,
		`<strong>Home</strong>`,
		`class="studio-breadcrumbs"`,
		`aria-label="Canvas selection"`,
		`<span>Site</span>`,
		`<output data-studio-selection-label="true">Hero selected</output>`,
		`data-test-control="one"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in canvas bar html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchCanvasBarHonorsOverrides(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderWorkbenchCanvasBar(view, WorkbenchCanvasBarOptions{
		Class:           "custom-canvas-bar",
		TitleClass:      "custom-title",
		KickerClass:     "custom-kicker",
		Kicker:          "Preview",
		RouteLabel:      "Products",
		BreadcrumbClass: "custom-breadcrumbs",
		BreadcrumbLabel: "Preview route",
		BreadcrumbRoot:  "Store",
		SelectionLabel:  "Product grid selected",
		Controls: []gosx.Node{
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-studio-zoom", "fit")), gosx.Text("Fit")),
		},
	}))
	for _, want := range []string{
		`class="custom-canvas-bar"`,
		`data-gosx-studio-canvas-bar-renderer="gosx-studio"`,
		`class="custom-title"`,
		`<p class="custom-kicker">Preview</p>`,
		`<strong>Products</strong>`,
		`class="custom-breadcrumbs"`,
		`aria-label="Preview route"`,
		`<span>Store</span>`,
		`<output data-studio-selection-label="true">Product grid selected</output>`,
		`data-studio-zoom="fit"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in custom canvas bar html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchCanvasStatusUsesDefaultShellView(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{
		RouteLabel:     "Home",
		ViewportLabel:  "Tablet",
		SelectionLabel: "Hero selected",
	})
	html := gosx.RenderHTML(RenderWorkbenchCanvasStatus(view, WorkbenchCanvasStatusOptions{}))
	for _, want := range []string{
		`class="studio-canvas-status"`,
		`data-gosx-studio-canvas-status-renderer="gosx-studio"`,
		`<span>Home</span>`,
		`<span data-studio-viewport-label="true">Tablet</span>`,
		`<output data-studio-selection-label="true">Hero selected</output>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in canvas status html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchCanvasStatusHonorsOverrides(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site"}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderWorkbenchCanvasStatus(view, WorkbenchCanvasStatusOptions{
		Class:          "custom-canvas-status",
		RouteLabel:     "Products",
		ViewportLabel:  "Mobile",
		SelectionLabel: "Product grid selected",
	}))
	for _, want := range []string{
		`class="custom-canvas-status"`,
		`data-gosx-studio-canvas-status-renderer="gosx-studio"`,
		`<span>Products</span>`,
		`<span data-studio-viewport-label="true">Mobile</span>`,
		`<output data-studio-selection-label="true">Product grid selected</output>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in custom canvas status html: %s", want, html)
		}
	}
}

func TestRenderWorkbenchToolbarHonorsDisabledActions(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client Site", PreviewURL: "/"}, WorkbenchShellViewOptions{})
	html := gosx.RenderHTML(RenderWorkbenchToolbar(view, WorkbenchToolbarOptions{
		DisablePreviewAction:   true,
		DisableSaveButton:      true,
		DisableHistoryControls: true,
	}))
	for _, blocked := range []string{
		`data-gosx-studio-preview-link="true"`,
		`data-gosx-studio-save-button="true"`,
		`data-gosx-studio-history-controls="true"`,
	} {
		if strings.Contains(html, blocked) {
			t.Fatalf("expected disabled toolbar action %q to be omitted: %s", blocked, html)
		}
	}
}
