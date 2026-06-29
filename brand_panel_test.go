package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBrandPanelFull(t *testing.T) {
	html := gosx.RenderHTML(RenderBrandPanel(brandPanelTestView(), brandPanelTestFields(), BrandPanelOptions{
		MediaPickerNode: gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-media-picker-island", "brand")), gosx.Text("Media picker")),
	}))

	for _, fragment := range []string{
		`<section class="editor-panel editor-panel--brand-studio studio-brand-panel" data-studio-brand-panel="true" data-studio-mode-panel="brand" data-studio-panel="brand" data-studio-engine-source="gosx" data-studio-brand-group-active="logo" data-gosx-studio-brand-panel-renderer="gosx-studio">`,
		`<header class="studio-brand-panel__head"><div><p class="kicker">Brand</p><h2>Logo and identity</h2><p>Manage public brand marks.</p></div></header>`,
		`<div class="studio-brand-panel__preview-slot" data-studio-brand-preview-slot="true"><div class="editor-brand-preview" data-editor-brand-preview="true">`,
		`<button class="editor-brand-handle" type="button" aria-label="Position logo" data-editor-brand-handle="true"><img src="/logo.png" alt="Site logo" data-editor-brand-logo="true" /></button>`,
		`<output class="editor-brand-readout" data-editor-logo-readout="true" aria-live="polite"></output>`,
		`<input class="studio-brand-panel__group-input" id="studioBrandGroupLogo" type="radio" name="studioBrandGroup" value="logo" checked aria-label="Logo" />`,
		`<label class="studio-brand-panel__group" for="studioBrandGroupFiles" data-studio-brand-group-label="files"><strong>Files</strong><small>Favicon and assets.</small></label>`,
		`<section class="studio-brand-panel__group-slot" data-studio-brand-group-slot="logo" data-studio-brand-group-selected="true">`,
		`<div class="studio-brand-panel__group-body" data-studio-brand-group-body="logo"><div data-field-list="logo">Logo fields</div></div>`,
		`<section class="studio-brand-panel__group-slot" data-studio-brand-group-slot="placement" data-studio-brand-group-selected="false">`,
		`<div data-field-list="layout">Layout fields</div><div class="editor-brand-tools">`,
		`<input type="checkbox" data-editor-logo-snap="true" checked />`,
		`<input id="logoSnapSize" type="number" min="1" max="32" value="8" data-editor-logo-snap-size="true" />`,
		`<button class="home-section-move" type="button" data-editor-logo-reset="true">Reset</button>`,
		`<div class="studio-brand-panel__group-body" data-studio-brand-group-body="files"><div data-field-list="files">File fields</div><a class="button button--secondary" href="/admin/media" data-gosx-link="true">Upload brand assets</a></div>`,
		`<section class="studio-brand-panel__group-slot studio-brand-media-picker-slot" data-studio-brand-group-slot="files" data-studio-brand-group-selected="false"><section data-studio-media-picker-island="brand">Media picker</section></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered brand panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered brand panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderBrandPanelMinimalKeepsEmptySlots(t *testing.T) {
	fields := brandPanelTestFields()
	fields["logoFieldList"] = "not a node"
	fields["layoutFieldList"] = nil
	fields["fileFieldList"] = "not a node"

	html := gosx.RenderHTML(RenderBrandPanel(brandPanelTestView(), fields, BrandPanelOptions{}))

	for _, fragment := range []string{
		`data-gosx-studio-brand-panel-renderer="gosx-studio"`,
		`<div class="studio-brand-panel__group-body" data-studio-brand-group-body="logo"></div>`,
		`<div class="studio-brand-panel__group-body" data-studio-brand-group-body="placement"><div class="editor-brand-tools">`,
		`<div class="studio-brand-panel__group-body" data-studio-brand-group-body="files"><a class="button button--secondary" href="/admin/media" data-gosx-link="true">Upload brand assets</a></div>`,
		`<section class="studio-brand-panel__group-slot studio-brand-media-picker-slot" data-studio-brand-group-slot="files" data-studio-brand-group-selected="false"></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("minimal brand panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`data-field-list="logo"`,
		`data-field-list="layout"`,
		`data-field-list="files"`,
		"<form",
		"csrf_token",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("minimal brand panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderBrandPanelRootAttrOverride(t *testing.T) {
	html := gosx.RenderHTML(RenderBrandPanel(brandPanelTestView(), brandPanelTestFields(), BrandPanelOptions{
		RootAttrs: map[string]any{
			"class":                  "custom-brand-panel",
			"data-studio-panel":      "custom",
			"data-custom-flag":       true,
			"aria-label":             "Brand override",
			"data-studio-extra-mode": "test",
		},
	}))

	for _, fragment := range []string{
		`class="custom-brand-panel"`,
		`data-studio-panel="custom"`,
		`data-custom-flag="true"`,
		`aria-label="Brand override"`,
		`data-studio-extra-mode="test"`,
		`data-gosx-studio-brand-panel-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("root attr override missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBrandPanelSegmentsWrapIslandInsideRoot(t *testing.T) {
	segments := RenderBrandPanelSegments(brandPanelTestView(), brandPanelTestFields(), BrandPanelOptions{})
	html := gosx.RenderHTML(gosx.Fragment(
		segments.RootOpen,
		segments.Body,
		segments.MediaPickerSlotOpen,
		gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-media-picker-island", "brand")), gosx.Text("Media picker")),
		segments.MediaPickerSlotClose,
		segments.RootClose,
	))

	for _, fragment := range []string{
		`<section class="editor-panel editor-panel--brand-studio studio-brand-panel" data-studio-brand-panel="true" data-studio-mode-panel="brand" data-studio-panel="brand" data-studio-engine-source="gosx" data-studio-brand-group-active="logo" data-gosx-studio-brand-panel-renderer="gosx-studio">`,
		`class="studio-brand-panel__group-input"`,
		`<section class="studio-brand-panel__group-slot studio-brand-media-picker-slot" data-studio-brand-group-slot="files" data-studio-brand-group-selected="false"><section data-studio-media-picker-island="brand">Media picker</section></section></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("brand panel segments missing %q:\n%s", fragment, html)
		}
	}

	rootIndex := strings.Index(html, `data-gosx-studio-brand-panel-renderer="gosx-studio"`)
	inputIndex := strings.Index(html, `class="studio-brand-panel__group-input"`)
	islandIndex := strings.Index(html, `data-studio-media-picker-island="brand"`)
	if rootIndex < 0 || inputIndex < 0 || islandIndex < 0 || !(rootIndex < inputIndex && inputIndex < islandIndex) {
		t.Fatalf("brand panel segment order should keep island inside root after group inputs:\n%s", html)
	}
}

func brandPanelTestView() map[string]any {
	return map[string]any{
		"kicker":          "Brand",
		"title":           "Logo and identity",
		"summary":         "Manage public brand marks.",
		"groupLabel":      "Brand groups",
		"defaultGroupKey": "logo",
		"previewLabel":    "Live logo placement",
		"groups": []map[string]any{
			{"key": "logo", "inputID": "studioBrandGroupLogo", "label": "Logo", "summary": "Logo image.", "selected": true},
			{"key": "placement", "inputID": "studioBrandGroupPlacement", "label": "Placement", "summary": "Logo size."},
			{"key": "files", "inputID": "studioBrandGroupFiles", "label": "Files", "summary": "Favicon and assets."},
		},
	}
}

func brandPanelTestFields() map[string]any {
	return map[string]any{
		"preview": map[string]any{
			"cornerLabel": "Top left",
			"buttonLabel": "Position logo",
			"logoURL":     "/logo.png",
			"logoAlt":     "Site logo",
		},
		"tools": map[string]any{
			"snapChecked": true,
			"snapSize":    8,
			"gridLabel":   "Grid",
			"resetLabel":  "Reset",
		},
		"media": map[string]any{
			"href":  "/admin/media",
			"label": "Upload brand assets",
			"class": "button button--secondary",
		},
		"logoFieldList":   gosx.El("div", gosx.Attrs(gosx.Attr("data-field-list", "logo")), gosx.Text("Logo fields")),
		"layoutFieldList": gosx.El("div", gosx.Attrs(gosx.Attr("data-field-list", "layout")), gosx.Text("Layout fields")),
		"fileFieldList":   gosx.El("div", gosx.Attrs(gosx.Attr("data-field-list", "files")), gosx.Text("File fields")),
	}
}
