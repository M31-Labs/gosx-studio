package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderInspectorChromePanel(t *testing.T) {
	view := map[string]any{
		"kicker":         "Properties",
		"modeLabel":      "Home",
		"selectionLabel": "Hero",
		"scopeLabel":     "Inspector scope",
		"crumbs": []map[string]any{
			{"label": "Site"},
			{"label": "Homepage"},
			{"label": "Home", "dynamicMode": true},
			{"label": "Hero", "dynamicSelection": true},
		},
	}

	html := gosx.RenderHTML(RenderInspectorChromePanel(view, InspectorChromePanelOptions{}))

	for _, fragment := range []string{
		`<section class="studio-inspector-chrome" data-studio-inspector-chrome-panel="true" data-studio-engine-source="gosx" data-gosx-studio-inspector-chrome-renderer="gosx-studio">`,
		`<div class="studio-inspector-head">`,
		`<p class="kicker">Properties</p>`,
		`<strong data-studio-mode-label="true">Home</strong>`,
		`<output data-studio-selection-label="true" aria-live="polite">Hero</output>`,
		`<div class="studio-scope-strip" aria-label="Inspector scope">`,
		`<span>Site</span><span>Homepage</span><span data-studio-mode-label="true">Home</span><output data-studio-selection-label="true">Hero</output>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("inspector chrome render missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("inspector chrome panel must not inject visible platform copy:\n%s", html)
	}
}

func TestRenderInspectorChromePanelPartialView(t *testing.T) {
	html := gosx.RenderHTML(RenderInspectorChromePanel(map[string]any{}, InspectorChromePanelOptions{
		RootAttrs: map[string]any{"data-extra": "ok"},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-inspector-chrome-renderer="gosx-studio"`,
		`data-extra="ok"`,
		`<strong data-studio-mode-label="true"></strong>`,
		`<output data-studio-selection-label="true" aria-live="polite"></output>`,
		`<div class="studio-scope-strip" aria-label=""></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("partial inspector chrome render missing %q:\n%s", fragment, html)
		}
	}
}
