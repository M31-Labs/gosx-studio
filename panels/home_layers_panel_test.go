package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/blocklayoutruntime"
)

func TestRenderHomeLayersPanelSegmentsNonEmpty(t *testing.T) {
	view := map[string]any{
		"class":          "studio-nav-panel studio-nav-panel--layers",
		"listClass":      "home-section-list editor-block-list",
		"kicker":         "Home",
		"title":          "Layers",
		"mode":           "home",
		"blockStudioKey": "homepage",
		"hasItems":       true,
		"items": []map[string]any{{
			"statusLabel": "Visible",
			"rowAttrs": map[string]any{
				"class":                    "home-section-card is-active",
				"draggable":                "true",
				"tabindex":                 "-1",
				blocklayoutruntime.RowAttr: "hero",
				"data-studio-block-label":  "Hero",
			},
			"keyInputAttrs":   map[string]any{"type": "hidden", "name": "home_sections[0][key]", "value": "hero"},
			"orderInputAttrs": map[string]any{"class": "home-section-order", "type": "hidden", "name": "home_sections[0][order]", "value": "0", blocklayoutruntime.OrderAttr: "true"},
			"handleAttrs":     map[string]any{"class": "home-section-handle", "type": "button", "aria-label": "Drag Hero", blocklayoutruntime.HandleAttr: "true"},
			"statusAttrs":     map[string]any{"class": "status-pill", blocklayoutruntime.PillAttr: "true"},
			"preview": map[string]any{
				"visualClass": "editor-block__visual editor-block__visual--hero",
				"kicker":      "Hero",
				"title":       "Thrown by hand",
				"titleAttrs":  map[string]any{"data-editor-preview": "hero-headline"},
				"body":        "Small batches",
				"bodyAttrs":   map[string]any{"data-editor-preview": "hero-subhead"},
				"hasActions":  true,
				"actions": []map[string]any{{
					"label": "Shop now",
					"attrs": map[string]any{"class": "button button--secondary", "data-editor-preview": "hero-primary"},
				}},
			},
			"visibilityAttrs":    map[string]any{"type": "checkbox", "name": "home_sections[0][enabled]", blocklayoutruntime.VisibleAttr: "true", "checked": true},
			"hiddenEnabledAttrs": map[string]any{"type": "hidden", "name": "home_sections[0][enabled]", "value": "off"},
			"upButtonAttrs":      map[string]any{"class": "home-section-move", "type": "button", blocklayoutruntime.MoveAttr: "up", "aria-label": "Move Hero up"},
			"downButtonAttrs":    map[string]any{"class": "home-section-move", "type": "button", blocklayoutruntime.MoveAttr: "down", "aria-label": "Move Hero down"},
		}},
	}

	segments := RenderHomeLayersPanelSegments(view, HomeLayersPanelOptions{})
	html := gosx.RenderHTML(gosx.Fragment(segments.RootOpen, segments.HeaderOpen, gosx.Text("SLOT"), segments.HeaderClose, segments.Body, segments.RootClose))

	for _, fragment := range []string{
		`<section class="studio-nav-panel studio-nav-panel--layers" data-studio-mode-panel="home" data-studio-engine-source="gosx" data-gosx-studio-home-layers-panel-renderer="gosx-studio">`,
		`<header class="studio-panel-heading"><div><p class="kicker">Home</p><h2>Layers</h2></div>SLOT</header>`,
		`<div class="home-section-list editor-block-list" data-block-studio="homepage">`,
		`<article class="home-section-card is-active" data-block-studio-block="hero" data-studio-block-label="Hero" draggable="true" tabindex="-1">`,
		`<input name="home_sections[0][key]" type="hidden" value="hero" />`,
		`data-block-studio-order="true"`,
		`data-block-studio-handle="true"`,
		`data-editor-block-pill="true">Visible</span>`,
		`class="editor-block__visual editor-block__visual--hero"`,
		`data-editor-preview="hero-headline">Thrown by hand</h2>`,
		`data-editor-preview="hero-subhead">Small batches</p>`,
		`<div class="button-row"><span class="button button--secondary" data-editor-preview="hero-primary">Shop now</span></div>`,
		`data-editor-block-visible="true"`,
		`checked`,
		`<span data-editor-block-status="true">Visible</span>`,
		`data-block-studio-move="up"`,
		`data-block-studio-move="down"`,
		`</section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered home layers panel missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("home layers panel must not inject visible platform copy:\n%s", html)
	}
}

func TestRenderHomeLayersPanelFullWithPickerSlot(t *testing.T) {
	view := map[string]any{
		"class":          "studio-nav-panel studio-nav-panel--layers",
		"listClass":      "home-section-list editor-block-list",
		"kicker":         "Home",
		"title":          "Layers",
		"mode":           "home",
		"blockStudioKey": "homepage",
		"hasItems":       true,
		"items": []map[string]any{{
			"statusLabel": "Visible",
			"rowAttrs": map[string]any{
				"class":                    "home-section-card is-active",
				blocklayoutruntime.RowAttr: "hero",
				"data-studio-block-label":  "Hero",
			},
			"keyInputAttrs":   map[string]any{"type": "hidden", "name": "home_sections[0][key]", "value": "hero"},
			"orderInputAttrs": map[string]any{"type": "hidden", "name": "home_sections[0][order]", "value": "0"},
			"handleAttrs":     map[string]any{"type": "button", "aria-label": "Drag Hero"},
			"statusAttrs":     map[string]any{"class": "status-pill"},
			"preview": map[string]any{
				"visualClass": "editor-block__visual editor-block__visual--hero",
				"kicker":      "Hero",
				"title":       "Thrown by hand",
				"body":        "Small batches",
			},
			"visibilityAttrs":    map[string]any{"type": "checkbox", "name": "home_sections[0][enabled]", "checked": true},
			"hiddenEnabledAttrs": map[string]any{"type": "hidden", "name": "home_sections[0][enabled]", "value": "off"},
			"upButtonAttrs":      map[string]any{"type": "button", blocklayoutruntime.MoveAttr: "up"},
			"downButtonAttrs":    map[string]any{"type": "button", blocklayoutruntime.MoveAttr: "down"},
		}},
	}

	html := gosx.RenderHTML(RenderHomeLayersPanel(view, HomeLayersPanelOptions{
		PickerNode: gosx.El("div", gosx.Attrs(gosx.Attr("data-studio-home-layer-picker", "true")), gosx.Text("Picker")),
	}))

	for _, fragment := range []string{
		`<section class="studio-nav-panel studio-nav-panel--layers" data-studio-mode-panel="home" data-studio-engine-source="gosx" data-gosx-studio-home-layers-panel-renderer="gosx-studio">`,
		`<header class="studio-panel-heading"><div><p class="kicker">Home</p><h2>Layers</h2></div><div data-studio-home-layer-picker="true">Picker</div></header>`,
		`<div class="home-section-list editor-block-list" data-block-studio="homepage">`,
		`data-block-studio-block="hero"`,
		`data-studio-block-label="Hero"`,
		`data-block-studio-move="up"`,
		`data-block-studio-move="down"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered full home layers panel missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderHomeLayersPanelSegmentsEmpty(t *testing.T) {
	view := map[string]any{
		"class":     "studio-nav-panel studio-nav-panel--layers",
		"kicker":    "Home",
		"title":     "Layers",
		"mode":      "home",
		"empty":     "No homepage sections are configured.",
		"hasItems":  false,
		"listClass": "home-section-list editor-block-list",
	}

	segments := RenderHomeLayersPanelSegments(view, HomeLayersPanelOptions{})
	html := gosx.RenderHTML(gosx.Fragment(segments.RootOpen, segments.HeaderOpen, segments.HeaderClose, segments.Body, segments.RootClose))

	for _, fragment := range []string{
		`data-gosx-studio-home-layers-panel-renderer="gosx-studio"`,
		`<header class="studio-panel-heading"><div><p class="kicker">Home</p><h2>Layers</h2></div></header>`,
		`<p class="empty">No homepage sections are configured.</p>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty home layers panel missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `data-block-studio`) || strings.Contains(html, `editor-block__chrome`) {
		t.Fatalf("empty home layers panel should not render list rows:\n%s", html)
	}
}
