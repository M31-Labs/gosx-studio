package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderWorkbenchComposesBrowserEditorShell(t *testing.T) {
	shell := New(Options{
		Title:      "Commerce Studio",
		PreviewURL: "/preview",
		SaveAction: "/admin/editor/__actions/save",
		Actions: []Action{
			LinkAction("preview", "Preview storefront", "/admin/storefront"),
		},
		Metrics: []Metric{NewMetric("blocks", "Blocks", 4)},
		Canvas:  CanvasSurface{RouteLabel: "Home", SelectionLabel: "Hero", Zoom: "fit", Focus: true},
		Left: []Panel{
			NewPanel("layers", "Layers", "Page sections."),
		},
		Right: []Panel{
			NewPanel("inspector", "Inspector", "Selected controls."),
		},
	})

	html := gosx.RenderHTML(RenderWorkbench(shell, WorkbenchOptions{
		CSRFToken:     "token-123",
		Autosave:      true,
		AutosaveDelay: 900,
		FormAttrs: []FieldAttribute{
			{Name: "data-studio-style-system", Value: "theme-1"},
			{Name: "data-studio-style-valid", Value: "true"},
		},
		ToolbarTitle:         "Canvas",
		ToolbarSummary:       "4 reusable blocks",
		SaveButtonLabel:      "Save checkpoint",
		ResizableRails:       true,
		IncludeScripts:       true,
		IncludeCanvasRuntime: true,
		IncludeFlowRuntime:   true,
		Commands: []Command{
			{Kind: CommandSubmit, Key: "save", Label: "Save checkpoint", Target: "save", Primary: true},
		},
		Insertions: []InsertOption{
			{Key: "hero", Label: "Hero", Target: "hero"},
		},
		CanvasFooter: []gosx.Node{
			gosx.El("aside", gosx.Attrs(gosx.Attr("data-extra-canvas-panel", "true")), gosx.Text("Activity")),
		},
	}))

	for _, want := range []string{
		`data-gosx-studio-workbench="true"`,
		`data-studio-workbench="true"`,
		`data-editor-workbench="true"`,
		`data-gosx-studio-state="true"`,
		`data-gosx-studio-client="true"`,
		`data-gosx-studio-autosave="true"`,
		`data-gosx-studio-autosave-delay="900"`,
		`data-gosx-studio-autosave-url="/admin/editor/__actions/save"`,
		`name="csrf_token"`,
		`value="token-123"`,
		`data-studio-style-system="theme-1"`,
		`data-studio-style-valid="true"`,
		`data-gosx-studio-toolbar="true"`,
		`data-studio-command-palette="true"`,
		`data-gosx-studio-save-status="true"`,
		`data-gosx-studio-history-controls="true"`,
		`data-gosx-studio-history-undo="true"`,
		`data-gosx-studio-history-redo="true"`,
		`Preview storefront`,
		`data-studio-layout="true"`,
		`data-studio-sidebar="left"`,
		`data-panel-key="layers"`,
		`data-studio-sidebar="right"`,
		`data-panel-key="inspector"`,
		`data-studio-canvas="true"`,
		`data-studio-insert-shelf="true"`,
		`data-studio-selection-commandbar="true"`,
		`data-gosx-studio-preview="true"`,
		`data-extra-canvas-panel="true"`,
		`data-studio-resizer="left"`,
		`data-studio-resizer="right"`,
		`role="separator"`,
		`aria-valuemin="256"`,
		`aria-valuemax="544"`,
		`data-studio-rail-default="416"`,
		`data-gosx-studio-workbench-runtime="true"`,
		`data-gosx-studio-command-runtime="true"`,
		`data-gosx-studio-state-runtime="true"`,
		`data-gosx-studio-site-canvas-runtime="true"`,
		`data-gosx-studio-flow-editor-runtime="true"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in workbench markup: %s", want, html)
		}
	}
}

func TestRenderWorkbenchAcceptsAppOwnedMainAndSlots(t *testing.T) {
	shell := New(Options{
		Title:      "School Studio",
		SaveAction: "/save",
		Canvas:     CanvasSurface{RouteLabel: "Website map"},
	})

	html := gosx.RenderHTML(RenderWorkbench(shell, WorkbenchOptions{
		Class:                "studio-shell",
		FormClass:            "studio-form",
		DisableClientActions: true,
		DisableCanvasStatus:  true,
		Toolbar: []gosx.Node{
			gosx.El("header", gosx.Attrs(gosx.Attr("data-custom-toolbar", "true")), gosx.Text("Toolbar")),
		},
		Main: []gosx.Node{
			gosx.El("section", gosx.Attrs(gosx.Attr("data-site-map", "true")), gosx.Text("Canvas")),
		},
		AfterForm: []gosx.Node{
			gosx.El("output", gosx.Attrs(gosx.Attr("data-after-form", "true")), gosx.Text("Done")),
		},
		Scripts: []gosx.Node{
			gosx.RawHTML(`<script data-app-runtime="true"></script>`),
		},
	}))

	for _, want := range []string{
		`class="studio-shell"`,
		`class="studio-form"`,
		`data-custom-toolbar="true"`,
		`data-site-map="true"`,
		`data-after-form="true"`,
		`data-gosx-studio-scripts="true"`,
		`data-app-runtime="true"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in custom workbench markup: %s", want, html)
		}
	}
	for _, unwanted := range []string{
		`data-gosx-studio-client="true"`,
		`data-studio-canvas="true"`,
		`data-gosx-studio-command-runtime="true"`,
	} {
		if strings.Contains(html, unwanted) {
			t.Fatalf("did not expect %q in custom workbench markup: %s", unwanted, html)
		}
	}
}
