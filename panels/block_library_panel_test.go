package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/blocklayoutruntime"
)

func TestRenderBlockLibraryPanelNonEmpty(t *testing.T) {
	view := map[string]any{
		"class":     "editor-panel editor-panel--library",
		"key":       "blocks",
		"mode":      "home",
		"title":     "Sections",
		"listClass": "editor-block-library",
		"hasItems":  true,
		"items": []map[string]any{{
			"label":       "Hero",
			"buttonLabel": "Added",
			"attrs": map[string]any{
				"class":                         "button button--secondary is-active",
				"type":                          "button",
				"data-editor-button-base":       "button button--secondary",
				blocklayoutruntime.AddBlockAttr: "hero",
				"aria-pressed":                  true,
			},
		}},
	}

	html := gosx.RenderHTML(RenderBlockLibraryPanel(view, BlockLibraryPanelOptions{}))

	for _, fragment := range []string{
		`<section class="editor-panel editor-panel--library" data-panel-key="blocks" data-studio-mode-panel="home" data-studio-engine-source="gosx" data-gosx-studio-block-library-panel-renderer="gosx-studio">`,
		`<h2>Sections</h2>`,
		`<div class="editor-block-library">`,
		`<button aria-pressed="true" class="button button--secondary is-active" data-editor-add-block="hero" data-editor-button-base="button button--secondary" type="button">`,
		`<span>Hero</span>`,
		`<small>Added</small>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered block library panel missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `class="empty"`) {
		t.Fatalf("non-empty block library panel should not render empty state:\n%s", html)
	}
}

func TestRenderBlockLibraryPanelEmpty(t *testing.T) {
	view := map[string]any{
		"class":     "editor-panel editor-panel--library",
		"key":       "blocks",
		"mode":      "home",
		"title":     "Sections",
		"listClass": "editor-block-library",
		"empty":     "No sections are available.",
	}

	html := gosx.RenderHTML(RenderBlockLibraryPanel(view, BlockLibraryPanelOptions{}))

	for _, fragment := range []string{
		`data-gosx-studio-block-library-panel-renderer="gosx-studio"`,
		`<h2>Sections</h2>`,
		`<p class="empty">No sections are available.</p>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty block library panel missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `editor-block-library`) {
		t.Fatalf("empty block library panel should not render list:\n%s", html)
	}
}
