package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestComposeWorkbenchBuildsSharedEditorBundle(t *testing.T) {
	shell := New(Options{
		Title:      "Test Studio",
		PreviewURL: "/",
		SaveAction: "/admin/editor/__actions/save",
		Metrics: []Metric{
			NewMetric("flows", "Flows", 2),
		},
		Canvas: CanvasSurface{RouteLabel: "Website map", SelectionLabel: "Home", Zoom: "fit", Focus: true},
	})
	commands := []Command{{Key: "save", Label: "Save", Kind: CommandSubmit, Target: "save"}}
	commandNode := RenderCommandPalette(CommandPaletteOptions{Class: "studio-command-palette", Commands: commands})
	saveNode := RenderSaveStatus(SaveStatusOptions{Class: "studio-save-status"})
	composition := ComposeWorkbench(shell, WorkbenchCompositionOptions{
		Commands: commands,
		CommandPalette: CommandPaletteOptions{
			Class:    "studio-command-palette",
			Launcher: "Commands",
			Title:    "Test commands",
		},
		SaveStatus: SaveStatusOptions{Class: "studio-save-status"},
		Health: NewHealthReport(
			NewHealthCheck("site", "Site", "Studio", ReadinessReady, "Ready", "Ready to publish."),
		),
		PublishReview: NewPublishReview(
			NewPublishCheck("site", "Site", "Studio", ReadinessReady, "Ready", "Ready to publish."),
		),
		IncludeHealthPanel:        true,
		IncludePublishReviewPanel: true,
		IncludeRuntimeNodes:       true,
		Workbench: WorkbenchOptions{
			Class:                   "studio-workbench-shell",
			FormClass:               "studio-workbench",
			CSRFToken:               "csrf",
			Toolbar:                 []gosx.Node{RenderWorkbenchSummaryToolbar(shell, WorkbenchSummaryToolbarOptions{CommandPaletteNode: commandNode, SaveStatusNode: saveNode})},
			Board:                   []gosx.Node{RenderMetricCards(shell.Metrics, MetricCardsOptions{})},
			IncludeWorkbenchRuntime: true,
			IncludeCommandRuntime:   true,
			IncludeStateRuntime:     true,
			IncludeCanvasRuntime:    true,
			IncludeFlowRuntime:      true,
		},
	})
	view := composition.View()
	for _, key := range []string{"commandPalette", "saveStatus", "workbench", "commandRuntime", "stateRuntime", "siteCanvasRuntime", "flowRuntime"} {
		if strings.TrimSpace(view[key].(string)) == "" {
			t.Fatalf("expected %s html in composition view: %#v", key, view)
		}
	}
	html := view["workbench"].(string)
	for _, check := range []string{
		`data-gosx-studio-workbench="true"`,
		`data-gosx-studio-workbench-runtime="true"`,
		`data-gosx-studio-flow-editor-runtime="true"`,
		`data-gosx-studio-history-controls="true"`,
		`data-studio-health="true"`,
		`data-studio-publish-review=`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in workbench composition: %s", check, html)
		}
	}
}

func TestRenderReusableWorkbenchPanels(t *testing.T) {
	shell := New(Options{
		Title:      "Panel Studio",
		PreviewURL: "/",
		Metrics:    []Metric{NewMetric("flows", "Flows", 4)},
		Navigation: []Section{{Key: "site", Label: "Site", Summary: "Pages"}},
		Left:       []Panel{NewPanel("layers", "Layers", "Page layers")},
		Right:      []Panel{NewPanel("inspector", "Inspector", "Selection fields")},
	})
	html := gosx.RenderHTML(gosx.Fragment(
		RenderMetricCards(shell.Metrics, MetricCardsOptions{}),
		RenderFlowStoragePanel(FlowStorageSummary{Kind: "file", Class: "studio-storage", Label: "File store", StatusLabel: "Durable", Detail: "Stored on disk.", Path: "data/flows.json", DocumentCount: 4}, FlowStoragePanelOptions{}),
		RenderLifecycleControlsPanel(LifecycleReviewState{}, LifecycleControlsPanelOptions{ApproveAction: "/approve", ScheduleAction: "/schedule", ProcessDueAction: "/due", Activity: []LifecycleActivityItem{{Summary: "Approved", Action: "approve", Created: "now"}}}),
		RenderStudioMapPanel(shell, StudioMapPanelOptions{}),
		RenderWorkbenchPreviewPanel(shell, WorkbenchPreviewPanelOptions{}),
	))
	for _, check := range []string{
		`data-studio-metric`,
		`data-studio-flow-storage="file"`,
		`formaction="/approve"`,
		`formaction="/schedule"`,
		`formaction="/due"`,
		`Studio map`,
		`data-studio-preview-frame="true"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in reusable panel html: %s", check, html)
		}
	}
	if MetricValue(shell.Metrics, "flows") != "4" {
		t.Fatalf("expected metric value helper to read flows")
	}
}

func TestRenderWorkbenchSectionNavigation(t *testing.T) {
	html := gosx.RenderHTML(gosx.Fragment(
		RenderWorkbenchSectionNavigator([]WorkbenchSectionNavItem{
			{Key: "map", Label: "Website map", Target: "website-map", Summary: "Pages and forms", Active: true},
			{Key: "forms", Label: "Family forms", Target: "family-forms"},
		}, WorkbenchSectionNavigatorOptions{Class: "workspace-nav", Title: "Move around"}),
		RenderWorkbenchSection(WorkbenchSectionOptions{
			Key:     "map",
			ID:      "website-map",
			Title:   "Website map",
			Summary: "Pages and forms",
			Children: []gosx.Node{
				gosx.El("p", nil, gosx.Text("Canvas")),
			},
		}),
	))
	for _, check := range []string{
		`data-studio-workbench-section-nav="true"`,
		`class="is-active"`,
		`href="#website-map"`,
		`data-studio-workbench-section-link="map"`,
		`id="website-map"`,
		`data-studio-workbench-section="map"`,
		`<h2>Website map</h2>`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in workbench section navigation: %s", check, html)
		}
	}
}
