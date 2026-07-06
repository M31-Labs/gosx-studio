package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderModebarMetricStripAndViewports(t *testing.T) {
	modes := []Mode{NewMode("Structure", "Structure", true), NewMode("Style", "Style", false)}
	metrics := []Metric{NewMetric("Blocks", "Blocks", 4), NewMetric("Media", "Media", 2)}
	viewports := []Viewport{NewViewport("Desktop", "Desktop", "100%", true), NewViewport("Mobile", "Mobile", "24rem", false)}

	modebar := gosx.RenderHTML(RenderModebar(modes, ModebarOptions{Class: "studio-modebar"}))
	if !strings.Contains(modebar, `class="studio-modebar"`) || !strings.Contains(modebar, `data-studio-mode-control="structure"`) || !strings.Contains(modebar, `aria-pressed="true"`) {
		t.Fatalf("unexpected modebar markup: %s", modebar)
	}

	metricStrip := gosx.RenderHTML(RenderMetricStrip(metrics, MetricStripOptions{Class: "studio-context-strip"}))
	if !strings.Contains(metricStrip, `class="studio-context-strip"`) || !strings.Contains(metricStrip, `<strong>4</strong> Blocks`) || !strings.Contains(metricStrip, `data-studio-metric="media"`) {
		t.Fatalf("unexpected metric strip markup: %s", metricStrip)
	}

	switcher := gosx.RenderHTML(RenderViewportSwitcher(viewports, ViewportSwitcherOptions{Class: "studio-viewport-switcher"}))
	if !strings.Contains(switcher, `data-studio-viewport="mobile"`) || !strings.Contains(switcher, `data-studio-viewport-width="24rem"`) || !strings.Contains(switcher, `aria-pressed="false"`) {
		t.Fatalf("unexpected viewport switcher markup: %s", switcher)
	}
}

func TestRenderCanvasChromeControls(t *testing.T) {
	tools := gosx.RenderHTML(RenderCanvasTools(CanvasToolsOptions{
		Class:       "studio-canvas-tools",
		FocusActive: true,
	}))
	if !strings.Contains(tools, `data-studio-rail-toggle="left"`) || !strings.Contains(tools, `data-studio-activity-toggle="true"`) || !strings.Contains(tools, `data-studio-focus-toggle="true"`) {
		t.Fatalf("unexpected canvas tools markup: %s", tools)
	}

	zoom := gosx.RenderHTML(RenderZoomControls(ZoomControlsOptions{Class: "studio-zoombar", Active: "100"}))
	if !strings.Contains(zoom, `data-studio-zoom="100"`) || !strings.Contains(zoom, `aria-pressed="true"`) {
		t.Fatalf("unexpected zoom markup: %s", zoom)
	}

	selection := gosx.RenderHTML(RenderSelectionCommandbar(SelectionCommandOptions{
		Class:          "studio-selection-commandbar",
		SelectionLabel: "Hero",
		StatusLabel:    "Visible",
		FieldLabel:     "hero.headline",
	}))
	if !strings.Contains(selection, `data-studio-selection-commandbar="true"`) || !strings.Contains(selection, `data-studio-selection-action="inline-text"`) || !strings.Contains(selection, `hero.headline`) {
		t.Fatalf("unexpected selection commandbar markup: %s", selection)
	}

	status := gosx.RenderHTML(RenderCanvasStatus(CanvasStatusOptions{
		Class:          "studio-canvas-status",
		RouteLabel:     "Home",
		ViewportLabel:  "Desktop",
		SelectionLabel: "Hero",
	}))
	if !strings.Contains(status, `Page / Home`) || !strings.Contains(status, `data-studio-viewport-label="true"`) || !strings.Contains(status, `Selection /`) {
		t.Fatalf("unexpected canvas status markup: %s", status)
	}
}

func TestRenderInsertShelfKeepsStudioAndLegacyHooks(t *testing.T) {
	html := gosx.RenderHTML(RenderInsertShelf([]InsertOption{
		{Key: "hero", Label: "Hero", Target: "hero", ButtonLabel: "On page", ButtonClass: "button studio-insert-chip button--ghost", ButtonBaseClass: "button studio-insert-chip"},
		{Key: "skip"},
	}, InsertShelfOptions{Class: "studio-insert-shelf"}))
	if !strings.Contains(html, `data-studio-insert-shelf="true"`) || !strings.Contains(html, `data-studio-insert-block="hero"`) || !strings.Contains(html, `data-editor-add-block="hero"`) || !strings.Contains(html, `data-editor-button-base="button studio-insert-chip"`) {
		t.Fatalf("expected insert shelf hooks, got: %s", html)
	}
	if strings.Contains(html, `skip`) {
		t.Fatalf("expected incomplete insert options to be skipped: %s", html)
	}
}
