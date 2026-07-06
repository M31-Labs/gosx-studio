package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderHomeInspectorPanelFull(t *testing.T) {
	html := gosx.RenderHTML(RenderHomeInspectorPanel(homeInspectorPanelTestView(), homeInspectorPanelTestContentFields(), HomeInspectorPanelOptions{}))

	for _, fragment := range []string{
		`<section id="studioHomeInspector" class="editor-panel editor-panel--home-inspector studio-home-inspector" data-studio-home-inspector-panel="true" data-studio-mode-panel="home" data-studio-panel="inspector" data-studio-engine-source="gosx" data-studio-inspector-group-active="copy" data-studio-inspector-dirty="false" data-studio-inspector-clean-label="Saved" data-studio-inspector-dirty-label="Unsaved changes" data-gosx-studio-home-inspector-panel-renderer="gosx-studio">`,
		`<header class="studio-home-inspector__head"><div><p class="kicker">Home</p><h2>Inspector</h2></div><output class="studio-home-inspector__state" data-studio-inspector-state="Saved">Saved</output><label class="studio-home-inspector__close" for="studioSiteMapInspectorFocus" role="button" tabindex="0">Close</label></header>`,
		`<section class="studio-home-inspector__target" aria-label="Selected section">`,
		`<strong data-studio-selection-label="true">Hero</strong>`,
		`<div class="studio-home-inspector__target-status"><output>Preview ready</output><output>Draft saved</output></div>`,
		`<div class="studio-home-inspector__groups" role="toolbar" aria-label="Inspector groups">`,
		`<button type="button" class="studio-home-inspector__group" data-studio-inspector-group="copy" aria-pressed="true"><strong>Copy</strong></button>`,
		`<button type="button" class="studio-home-inspector__group" data-studio-inspector-group="media" aria-pressed="false"><strong>Media</strong></button>`,
		`<button type="button" class="studio-home-inspector__group" data-studio-inspector-group="sources" aria-pressed="false"><strong>Sources</strong></button>`,
		`<button type="button" class="studio-home-inspector__group" data-studio-inspector-group="all" aria-pressed="false"><strong>All</strong></button>`,
		`<div class="studio-home-inspector__fields" data-studio-home-inspector-fields="true">`,
		`<section class="editor-panel studio-home-fields-panel" data-studio-home-fields-panel="true" data-panel-key="inspector-fields" data-studio-panel="inspector-fields" data-studio-mode-panel="home" data-studio-engine-source="gosx">`,
		`<p class="empty" hidden>No fields are available.</p>`,
		`<div class="studio-home-fields-panel__fields"`,
		`data-gosx-studio-inspector-field-list-renderer="gosx-studio"`,
		`data-field-list-sentinel="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("home inspector panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(html, notWant) {
			t.Fatalf("home inspector panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderHomeInspectorPanelMinimalKeepsRootAndFields(t *testing.T) {
	html := gosx.RenderHTML(RenderHomeInspectorPanel(map[string]any{}, map[string]any{}, HomeInspectorPanelOptions{}))

	for _, fragment := range []string{
		`id="studioHomeInspector"`,
		`data-studio-home-inspector-panel="true"`,
		`data-gosx-studio-home-inspector-panel-renderer="gosx-studio"`,
		`data-studio-inspector-dirty="false"`,
		`<div class="studio-home-inspector__fields" data-studio-home-inspector-fields="true">`,
		`<section class="" data-studio-home-fields-panel="true" data-panel-key="" data-studio-panel="" data-studio-mode-panel="" data-studio-engine-source="gosx"><p class="empty"></p></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("minimal home inspector panel missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderHomeInspectorPanelRootAttrOverride(t *testing.T) {
	html := gosx.RenderHTML(RenderHomeInspectorPanel(homeInspectorPanelTestView(), homeInspectorPanelTestContentFields(), HomeInspectorPanelOptions{
		RootAttrs: map[string]any{
			"id":                          "customHomeInspector",
			"class":                       "custom-home-panel",
			"data-studio-panel":           "custom-inspector",
			"aria-label":                  "Home inspector override",
			"data-home-inspector-extra":   "test",
			"data-studio-inspector-dirty": true,
		},
	}))

	for _, fragment := range []string{
		`id="customHomeInspector"`,
		`class="custom-home-panel"`,
		`data-studio-panel="custom-inspector"`,
		`aria-label="Home inspector override"`,
		`data-home-inspector-extra="test"`,
		`data-studio-inspector-dirty="true"`,
		`data-gosx-studio-home-inspector-panel-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("home inspector root attr override missing %q:\n%s", fragment, html)
		}
	}
}

func homeInspectorPanelTestView() map[string]any {
	return map[string]any{
		"kicker":          "Home",
		"title":           "Inspector",
		"defaultGroupKey": "copy",
		"cleanLabel":      "Saved",
		"dirtyLabel":      "Unsaved changes",
		"targetLabel":     "Selected section",
		"targetKicker":    "Selection",
		"selectionLabel":  "Hero",
		"targetSummary":   "Hero copy and media",
		"previewLabel":    "Preview ready",
		"draftLabel":      "Draft saved",
		"groupLabel":      "Inspector groups",
		"copyLabel":       "Copy",
		"mediaLabel":      "Media",
		"sourcesLabel":    "Sources",
		"allLabel":        "All",
	}
}

func homeInspectorPanelTestContentFields() map[string]any {
	return map[string]any{
		"class":     "editor-panel studio-home-fields-panel",
		"key":       "inspector-fields",
		"mode":      "home",
		"empty":     "No fields are available.",
		"hasFields": true,
		"fieldList": gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-home-fields-panel__fields"),
			gosx.Attr("data-gosx-studio-inspector-field-list-renderer", "gosx-studio"),
			gosx.Attr("data-field-list-sentinel", "true"),
		)),
	}
}
