package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBrandMediaPickerFull(t *testing.T) {
	html := gosx.RenderHTML(RenderBrandMediaPicker(brandMediaPickerTestView(), BrandMediaPickerOptions{}))

	for _, fragment := range []string{
		`<section class="studio-brand-media-picker" data-studio-media-picker-island="brand" data-studio-engine-source="gosx" data-studio-media-filter="all" data-gosx-studio-brand-media-picker-renderer="gosx-studio">`,
		`<header class="studio-brand-media-picker__head"><div><p class="kicker">Media</p><h3>Choose brand images</h3><p>Pick uploaded images.</p></div><a href="/admin/media" data-gosx-link="true">Upload</a></header>`,
		`<div class="studio-brand-media-picker__filters" role="toolbar" aria-label="Media filters"><button type="button" aria-pressed="true">All</button><button type="button" aria-pressed="false">Ready</button><button type="button" aria-pressed="false">Needs alt</button></div>`,
		`<p class="studio-brand-media-picker__empty" hidden>Upload an image to use it as a brand mark.</p>`,
		`<div class="studio-brand-media-picker__grid" aria-label="Uploaded brand images">`,
		`<article class="studio-brand-media-card studio-brand-media-card--ready" data-studio-media-asset="logo" data-studio-media-filter-group="ready">`,
		`<img src="/media/logo.png" alt="Logo" />`,
		`<strong>logo.png</strong><span>Ready</span>`,
		`<div class="studio-brand-media-picker__actions" aria-label="Use logo.png">`,
		`<button type="submit" form="editor-workbench" formaction="/admin/editor/__actions/save" name="logoUrl" value="/media/logo.png">Use logo</button>`,
		`<button type="submit" form="editor-workbench" formaction="/admin/editor/__actions/save" name="faviconUrl" value="/media/logo.png">Use icon</button>`,
		`<article class="studio-brand-media-card studio-brand-media-card--missing-alt" data-studio-media-asset="missing-alt" data-studio-media-filter-group="missing-alt">`,
		`<img src="/media/missing.png" alt="" />`,
		`<strong>missing.png</strong><span>Needs alt</span>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered brand media picker missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{"<form", "csrf_token", "data-on-click", "onClick"} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered brand media picker must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderBrandMediaPickerEmptyState(t *testing.T) {
	view := brandMediaPickerTestView()
	view["hasAssets"] = false
	view["assets"] = []map[string]any{}

	html := gosx.RenderHTML(RenderBrandMediaPicker(view, BrandMediaPickerOptions{}))
	for _, fragment := range []string{
		`data-gosx-studio-brand-media-picker-renderer="gosx-studio"`,
		`<p class="studio-brand-media-picker__empty">Upload an image to use it as a brand mark.</p>`,
		`<div class="studio-brand-media-picker__grid" aria-label="Uploaded brand images"></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty brand media picker missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<article`) {
		t.Fatalf("empty brand media picker should not render asset cards:\n%s", html)
	}
}

func TestRenderBrandMediaPickerRootAttrOverride(t *testing.T) {
	html := gosx.RenderHTML(RenderBrandMediaPicker(brandMediaPickerTestView(), BrandMediaPickerOptions{
		RootAttrs: map[string]any{
			"class":              "custom-picker",
			"data-custom-flag":   true,
			"data-studio-panel":  "brand-media",
			"aria-label":         "Brand media",
			"data-studio-source": "test",
		},
	}))

	for _, fragment := range []string{
		`class="custom-picker"`,
		`data-custom-flag="true"`,
		`data-studio-panel="brand-media"`,
		`aria-label="Brand media"`,
		`data-studio-source="test"`,
		`data-gosx-studio-brand-media-picker-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("brand media picker attr override missing %q:\n%s", fragment, html)
		}
	}
}

func brandMediaPickerTestView() map[string]any {
	return map[string]any{
		"kicker":          "Media",
		"title":           "Choose brand images",
		"summary":         "Pick uploaded images.",
		"filterLabel":     "Media filters",
		"assetLabel":      "Uploaded brand images",
		"emptyLabel":      "Upload an image to use it as a brand mark.",
		"uploadHref":      "/admin/media",
		"formID":          "editor-workbench",
		"saveAction":      "/admin/editor/__actions/save",
		"logoPickName":    "logoUrl",
		"faviconPickName": "faviconUrl",
		"hasAssets":       true,
		"assets": []map[string]any{
			{
				"id":          "logo",
				"url":         "/media/logo.png",
				"alt":         "Logo",
				"filename":    "logo.png",
				"cardClass":   "studio-brand-media-card studio-brand-media-card--ready",
				"filterGroup": "ready",
				"statusLabel": "Ready",
				"actionLabel": "Use logo.png",
			},
			{
				"id":          "missing-alt",
				"url":         "/media/missing.png",
				"filename":    "missing.png",
				"cardClass":   "studio-brand-media-card studio-brand-media-card--missing-alt",
				"filterGroup": "missing-alt",
				"statusLabel": "Needs alt",
				"actionLabel": "Use missing.png",
			},
		},
	}
}
