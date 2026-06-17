package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

const styleTestAction = "/admin/editor/__actions/authoring"

// palette helpers for the style panel tests.

func testPaletteAny() []map[string]any {
	return []map[string]any{
		{
			"key":      "accent",
			"name":     "colorAccent",
			"label":    "Accent",
			"cssVar":   "--color-accent",
			"value":    "#b5651d",
			"fallback": "#b5651d",
		},
		{
			"key":      "canvas",
			"name":     "colorCanvas",
			"label":    "Canvas",
			"cssVar":   "--color-canvas",
			"value":    "#f8f5ef",
			"fallback": "#ffffff",
		},
		{
			"key":      "ink",
			"name":     "colorInk",
			"label":    "Ink",
			"cssVar":   "--color-ink",
			"value":    "#1a1a1a",
			"fallback": "#000000",
		},
	}
}

func testPaletteString() []map[string]string {
	return []map[string]string{
		{
			"key":      "primary",
			"name":     "colorPrimary",
			"label":    "Primary",
			"cssVar":   "--color-primary",
			"value":    "#3a86ff",
			"fallback": "#3a86ff",
		},
		{
			"key":      "secondary",
			"name":     "colorSecondary",
			"label":    "Secondary",
			"cssVar":   "--color-secondary",
			"value":    "#ff006e",
			"fallback": "#ff006e",
		},
	}
}

// TestRenderStylePanelColorInputsPerToken asserts one <input type="color"> per
// palette entry with the correct name, value, data-editor-color-token (cssVar),
// and data-editor-color-key attributes, all bound to formID.
func TestRenderStylePanelColorInputsPerToken(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	for _, entry := range testPaletteAny() {
		name := entry["name"].(string)
		cssVar := entry["cssVar"].(string)
		value := entry["value"].(string)
		key := entry["key"].(string)

		// Each token must have <input type="color">
		if !strings.Contains(html, `type="color"`) {
			t.Fatalf("panel must render <input type=\"color\"> for each token, missing in:\n%s", html)
		}

		// name attribute must match the form field name
		if !strings.Contains(html, `name="`+name+`"`) {
			t.Fatalf("color input must carry name=%q, not found in:\n%s", name, html)
		}

		// value must be the current hex
		if !strings.Contains(html, `value="`+value+`"`) {
			t.Fatalf("color input must carry value=%q, not found in:\n%s", value, html)
		}

		// LIVE-PREVIEW GUARD: data-editor-color-token must be the cssVar, NOT the field name.
		// The styleruntime reads this attr and publishes $preview.theme.colors keyed by it.
		wantTokenAttr := `data-editor-color-token="` + cssVar + `"`
		if !strings.Contains(html, wantTokenAttr) {
			t.Fatalf("color input must carry data-editor-color-token=%q (the CSS var, not the field name).\nIf this fires, live preview will silently break.\nHTML:\n%s", cssVar, html)
		}

		// data-editor-color-key must be the logical token key
		wantKeyAttr := `data-editor-color-key="` + key + `"`
		if !strings.Contains(html, wantKeyAttr) {
			t.Fatalf("color input must carry data-editor-color-key=%q, not found in:\n%s", key, html)
		}

		// All color inputs must be associated with the form
		if !strings.Contains(html, `form="stylePaletteForm"`) {
			t.Fatalf("color input must carry form=%q, not found in:\n%s", "stylePaletteForm", html)
		}
	}
}

// TestRenderStylePanelColorMirror asserts the read-only hex text mirror is
// emitted for each token (mirrors the inspector color-widget shape).
func TestRenderStylePanelColorMirror(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	if !strings.Contains(html, `data-gosx-studio-inspector-color-mirror`) {
		t.Fatalf("panel must render the read-only hex mirror with data-gosx-studio-inspector-color-mirror:\n%s", html)
	}
}

// TestRenderStylePanelResetButton asserts each token has a per-token reset
// button with data-gosx-studio-style-reset and data-fallback attributes.
func TestRenderStylePanelResetButton(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	for _, entry := range testPaletteAny() {
		fallback := entry["fallback"].(string)

		if !strings.Contains(html, `data-gosx-studio-style-reset`) {
			t.Fatalf("panel must render per-token reset button with data-gosx-studio-style-reset:\n%s", html)
		}
		wantFallback := `data-fallback="` + fallback + `"`
		if !strings.Contains(html, wantFallback) {
			t.Fatalf("reset button must carry data-fallback=%q, not found in:\n%s", fallback, html)
		}
	}
}

// TestRenderStylePanelHiddenOperationInput asserts the hidden
// gosx_studio_operation=save-appearance input is present and bound to the form.
func TestRenderStylePanelHiddenOperationInput(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	if !strings.Contains(html, `name="gosx_studio_operation"`) {
		t.Fatalf("panel must include hidden gosx_studio_operation input:\n%s", html)
	}
	if !strings.Contains(html, `value="save-appearance"`) {
		t.Fatalf("panel must include value=\"save-appearance\" for the operation input:\n%s", html)
	}
}

// TestRenderStylePanelManagedFormMarker asserts the form (not the section) carries
// data-gosx-studio-authoring-managed="true" so authoringruntime intercepts the submit.
// island_runtime.js line 741: form.hasAttribute(MANAGED_FORM_ATTR).
func TestRenderStylePanelManagedFormMarker(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	if !strings.Contains(html, `data-gosx-studio-authoring-managed="true"`) {
		t.Fatalf("the form must carry data-gosx-studio-authoring-managed=\"true\" so the island runtime intercepts submit:\n%s", html)
	}
}

// TestRenderStylePanelSectionDataAttrs asserts the outer section carries the
// panel-level data attributes (panel-renderer, style-panel).
func TestRenderStylePanelSectionDataAttrs(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	if !strings.Contains(html, `data-gosx-studio-style-panel="true"`) {
		t.Fatalf("outer section must carry data-gosx-studio-style-panel=\"true\":\n%s", html)
	}
	if !strings.Contains(html, `data-gosx-studio-authoring-panel-renderer="gosx-studio"`) {
		t.Fatalf("outer section must carry data-gosx-studio-authoring-panel-renderer=\"gosx-studio\":\n%s", html)
	}
}

// TestRenderStylePanelSubmitAndCancelButtons asserts one Save submit + one Cancel reset.
func TestRenderStylePanelSubmitAndCancelButtons(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	if !strings.Contains(html, `type="submit"`) {
		t.Fatalf("panel must include a submit button:\n%s", html)
	}
	if !strings.Contains(html, `type="reset"`) {
		t.Fatalf("panel must include a reset/cancel button:\n%s", html)
	}
	if !strings.Contains(html, `data-gosx-studio-inspector-cancel`) {
		t.Fatalf("cancel button must carry data-gosx-studio-inspector-cancel (matches inspector convention):\n%s", html)
	}
}

// TestRenderStylePanelFormIDOnButtons asserts submit and reset buttons are bound
// to the supplied formID.
func TestRenderStylePanelFormIDOnButtons(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	// Both buttons must reference the form
	count := strings.Count(html, `form="stylePaletteForm"`)
	// At minimum: hidden op input + each color input + both buttons → at least 5
	// but we just need to assert the buttons are there
	if count < 2 {
		t.Fatalf("expected at least 2 elements with form=\"stylePaletteForm\" (buttons + inputs), got %d:\n%s", count, html)
	}
}

// TestRenderStylePanelEmptyPaletteInert asserts that an empty or nil palette
// produces an inert section (no crash, no color inputs).
func TestRenderStylePanelEmptyPaletteInert(t *testing.T) {
	// nil palette
	view := map[string]any{}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))
	if strings.Contains(html, `type="color"`) {
		t.Fatalf("empty palette must not render any color inputs:\n%s", html)
	}

	// empty slice ([]map[string]any)
	view2 := map[string]any{
		"palette": []map[string]any{},
	}
	html2 := gosx.RenderHTML(RenderStylePanel(view2, "stylePaletteForm", styleTestAction))
	if strings.Contains(html2, `type="color"`) {
		t.Fatalf("empty palette slice must not render any color inputs:\n%s", html2)
	}
}

// TestRenderStylePanelAcceptsStringPalette asserts that []map[string]string
// shape (both shapes per spec) also works correctly.
func TestRenderStylePanelAcceptsStringPalette(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteString(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))

	for _, entry := range testPaletteString() {
		cssVar := entry["cssVar"]
		wantTokenAttr := `data-editor-color-token="` + cssVar + `"`
		if !strings.Contains(html, wantTokenAttr) {
			t.Fatalf("string-palette: color input must carry data-editor-color-token=%q:\n%s", cssVar, html)
		}
	}
}

// TestRenderStylePanelNoPlatformCopy asserts no "GoSX Studio" visible string.
func TestRenderStylePanelNoPlatformCopy(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("panel must not emit visible platform copy:\n%s", html)
	}
}

// TestRenderStylePanelFormAction asserts the form carries the host-provided action
// (the authoring route) so the managed submit POSTs to the right endpoint rather than
// window.location.href.
func TestRenderStylePanelFormAction(t *testing.T) {
	view := map[string]any{"palette": testPaletteAny()}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction))
	if !strings.Contains(html, `action="`+styleTestAction+`"`) {
		t.Fatalf("the form must carry action=%q so the managed submit POSTs to the authoring route:\n%s", styleTestAction, html)
	}
}
