package studio

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
