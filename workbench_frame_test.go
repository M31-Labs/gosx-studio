package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderWorkbenchFrameUsesShellViewChrome(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title:         "Client website",
		PreviewURL:    "/",
		SaveAction:    "/admin/editor/__actions/save",
		BlockCount:    3,
		MediaCount:    2,
		RevisionCount: 1,
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero selected",
			Zoom:           "100",
		},
	}, WorkbenchShellViewOptions{
		Class:            "client-workbench-shell",
		FormClass:        "client-workbench",
		FormID:           "clientWorkbench",
		CSRFToken:        "csrf-token",
		Mode:             "home",
		StyleSystemID:    "brand-system",
		StyleSystemValid: true,
		FeatureFlagAttrs: map[string]any{"data-gosx-studio-feature-flag-workbench-runtime-islands": "true"},
		Extras:           map[string]any{"authoringURL": "/admin/editor/__actions/authoring"},
	})
	html := gosx.RenderHTML(RenderWorkbenchFrame(view, WorkbenchFrameOptions{
		StageClass:     "client-stage",
		LeftRailClass:  "client-left-rail",
		MainClass:      "client-main",
		RightRailClass: "client-right-rail",
		ResizableRails: true,
		Toolbar: gosx.El("div", gosx.Attrs(gosx.Attr("data-test-toolbar", "true")),
			gosx.Text("Website editor")),
		LeftRail: []gosx.Node{
			gosx.El("section", gosx.Attrs(gosx.Attr("data-test-left-panel", "true")), gosx.Text("Pages")),
		},
		CanvasBar: []gosx.Node{
			gosx.El("strong", nil, gosx.Text("Canvas")),
		},
		Board: []gosx.Node{
			gosx.El("section", gosx.Attrs(gosx.Attr("data-test-board", "true")), gosx.Text("Preview board")),
		},
		CanvasStatus: []gosx.Node{
			gosx.El("output", gosx.Attrs(gosx.Attr("data-test-status", "true")), gosx.Text("Ready")),
		},
		RightRail: []gosx.Node{
			gosx.El("section", gosx.Attrs(gosx.Attr("data-test-right-panel", "true")), gosx.Text("Inspector")),
		},
	}))

	for _, want := range []string{
		`class="client-workbench-shell"`,
		`data-gosx-studio-frame-renderer="gosx-studio"`,
		`data-gosx-studio-feature-flag-workbench-runtime-islands="true"`,
		`id="clientWorkbench"`,
		`class="client-workbench"`,
		`action="/admin/editor/__actions/save"`,
		`data-gosx-studio-client="true"`,
		`data-gosx-studio-autosave-url="/admin/editor/__actions/save"`,
		`data-gosx-studio-authoring-url="/admin/editor/__actions/authoring"`,
		`data-gosx-csrf-token="csrf-token"`,
		`data-studio-style-system="brand-system"`,
		`data-studio-style-valid="true"`,
		`type="hidden"`,
		`name="csrf_token"`,
		`data-test-toolbar="true"`,
		`class="client-stage"`,
		`data-gosx-studio-stage-renderer="gosx-studio"`,
		`class="client-left-rail"`,
		`data-gosx-studio-rail-renderer="gosx-studio"`,
		`data-gosx-studio-rail-resizer-renderer="gosx-studio"`,
		`class="client-main"`,
		`aria-label="Home"`,
		`data-panel-key="home"`,
		`data-studio-canvas-zoom="100"`,
		`data-gosx-studio-canvas-shell-renderer="gosx-studio"`,
		`data-gosx-studio-board-renderer="gosx-studio"`,
		`data-test-board="true"`,
		`class="client-right-rail"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in workbench frame html: %s", want, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("host frame should not inject product copy into client UI: %s", html)
	}
}

func TestRenderWorkbenchRailResizerUsesShellViewDefaultsAndOverrides(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{Title: "Client website"}, WorkbenchShellViewOptions{
		RightResizer: WorkbenchRailResizer{
			Label:   "Resize tools",
			Min:     360,
			Max:     640,
			Default: 480,
		},
	})

	right := gosx.RenderHTML(RenderWorkbenchRailResizer(view, "right", WorkbenchRailResizerOptions{}))
	for _, want := range []string{
		`class="studio-rail-resizer studio-rail-resizer--right"`,
		`aria-label="Resize tools"`,
		`aria-valuemin="360"`,
		`aria-valuemax="640"`,
		`aria-valuenow="480"`,
		`data-studio-resizer="right"`,
		`data-gosx-studio-rail-resizer-renderer="gosx-studio"`,
	} {
		if !strings.Contains(right, want) {
			t.Fatalf("expected %q in right resizer html: %s", want, right)
		}
	}

	left := gosx.RenderHTML(RenderWorkbenchRailResizer(view, "left", WorkbenchRailResizerOptions{
		Class:   "custom-resizer",
		Label:   "Resize pages",
		Default: "360",
	}))
	for _, want := range []string{
		`class="custom-resizer"`,
		`aria-label="Resize pages"`,
		`aria-valuemin="256"`,
		`aria-valuemax="448"`,
		`aria-valuenow="360"`,
		`data-studio-resizer="left"`,
	} {
		if !strings.Contains(left, want) {
			t.Fatalf("expected %q in left resizer html: %s", want, left)
		}
	}
}
