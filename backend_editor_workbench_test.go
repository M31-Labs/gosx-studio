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
