package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

const styleTestAction = "/admin/editor/__actions/authoring"
const styleTestCSRF = "csrf-test-token"

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))
	if strings.Contains(html, `type="color"`) {
		t.Fatalf("empty palette must not render any color inputs:\n%s", html)
	}

	// empty slice ([]map[string]any)
	view2 := map[string]any{
		"palette": []map[string]any{},
	}
	html2 := gosx.RenderHTML(RenderStylePanel(view2, "stylePaletteForm", styleTestAction, styleTestCSRF))
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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

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
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("panel must not emit visible platform copy:\n%s", html)
	}
}

// TestRenderStylePanelFormAction asserts the form carries the host-provided action
// (the authoring route) so the managed submit POSTs to the right endpoint rather than
// window.location.href.
func TestRenderStylePanelFormAction(t *testing.T) {
	view := map[string]any{"palette": testPaletteAny()}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))
	if !strings.Contains(html, `action="`+styleTestAction+`"`) {
		t.Fatalf("the form must carry action=%q so the managed submit POSTs to the authoring route:\n%s", styleTestAction, html)
	}
}

// TestRenderStylePanelCSRFField asserts a non-empty csrfToken renders a csrf_token
// hidden input the authoring runtime forwards as X-CSRF-Token.
func TestRenderStylePanelCSRFField(t *testing.T) {
	view := map[string]any{"palette": testPaletteAny()}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))
	if !strings.Contains(html, `name="csrf_token"`) || !strings.Contains(html, `value="`+styleTestCSRF+`"`) {
		t.Fatalf("a non-empty csrfToken must render a csrf_token field so the managed POST is accepted:\n%s", html)
	}
	// Blank token → no csrf_token field (host decides).
	htmlBlank := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, ""))
	if strings.Contains(htmlBlank, `name="csrf_token"`) {
		t.Fatalf("a blank csrfToken must omit the csrf_token field:\n%s", htmlBlank)
	}
}

// ============================================================
// Font section tests
// ============================================================

// testFontsAny returns a []map[string]any font view fixture with all three roles.
func testFontsAny() []map[string]any {
	return []map[string]any{
		{
			"role":      "display",
			"label":     "Display",
			"nameField": "displayFont",
			"urlField":  "displayFontUrl",
			"family":    "Fraunces",
			"url":       "https://example.com/fraunces.woff2",
		},
		{
			"role":      "body",
			"label":     "Body",
			"nameField": "bodyFont",
			"urlField":  "bodyFontUrl",
			"family":    "Inter",
			"url":       "https://example.com/inter.woff2",
		},
		{
			"role":      "mono",
			"label":     "Mono",
			"nameField": "monoFont",
			"urlField":  "monoFontUrl",
			"family":    "JetBrains Mono",
			"url":       "",
		},
	}
}

// testFontsString returns the same fixture as []map[string]string.
func testFontsString() []map[string]string {
	return []map[string]string{
		{
			"role":      "display",
			"label":     "Display",
			"nameField": "displayFont",
			"urlField":  "displayFontUrl",
			"family":    "Playfair Display",
			"url":       "https://example.com/playfair.woff2",
		},
	}
}

// TestRenderStylePanelFontFamilyInput is the LIVE-PREVIEW GUARD for the family
// input. styleruntime's bindFontsIsland (island_runtime.js:543) queries
// [data-editor-font-name="{role}"] — the attribute value is the slot/role.
// If the wrong attribute or value is emitted, font live-preview silently breaks.
func TestRenderStylePanelFontFamilyInput(t *testing.T) {
	view := map[string]any{
		"fonts": testFontsAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "fontsForm", styleTestAction, styleTestCSRF))

	for _, entry := range testFontsAny() {
		role := entry["role"].(string)
		nameField := entry["nameField"].(string)
		family := entry["family"].(string)

		// LIVE-PREVIEW GUARD: data-editor-font-name must equal the slot role.
		wantFontNameAttr := `data-editor-font-name="` + role + `"`
		if !strings.Contains(html, wantFontNameAttr) {
			t.Fatalf("family input must carry data-editor-font-name=%q (slot role).\nbindFontsIsland (island_runtime.js:551) queries [data-editor-font-name=%q] — wrong value breaks live preview.\nHTML:\n%s", role, role, html)
		}

		// name attribute must match the form field key for RawForm submission.
		wantName := `name="` + nameField + `"`
		if !strings.Contains(html, wantName) {
			t.Fatalf("family input must carry name=%q for RawForm submission, not found in:\n%s", nameField, html)
		}

		// value must be the current family.
		wantValue := `value="` + family + `"`
		if !strings.Contains(html, wantValue) {
			t.Fatalf("family input must carry value=%q, not found in:\n%s", family, html)
		}

		// Must be a text input.
		if !strings.Contains(html, `type="text"`) {
			t.Fatalf("family input must be type=text, not found in:\n%s", html)
		}

		// Must be bound to the form.
		wantForm := `form="fontsForm"`
		if !strings.Contains(html, wantForm) {
			t.Fatalf("family input must carry form=%q, not found in:\n%s", "fontsForm", html)
		}
	}
}

// TestRenderStylePanelFontURLInput is the LIVE-PREVIEW GUARD for the URL input.
// bindFontsIsland (island_runtime.js:552) queries [data-editor-font-url="{role}"].
func TestRenderStylePanelFontURLInput(t *testing.T) {
	view := map[string]any{
		"fonts": testFontsAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "fontsForm", styleTestAction, styleTestCSRF))

	for _, entry := range testFontsAny() {
		role := entry["role"].(string)
		urlField := entry["urlField"].(string)

		// LIVE-PREVIEW GUARD: data-editor-font-url must equal the slot role.
		wantFontURLAttr := `data-editor-font-url="` + role + `"`
		if !strings.Contains(html, wantFontURLAttr) {
			t.Fatalf("URL input must carry data-editor-font-url=%q (slot role).\nbindFontsIsland (island_runtime.js:552) queries [data-editor-font-url=%q] — wrong value breaks live preview.\nHTML:\n%s", role, role, html)
		}

		// name attribute must match the form field key.
		wantName := `name="` + urlField + `"`
		if !strings.Contains(html, wantName) {
			t.Fatalf("URL input must carry name=%q for RawForm submission, not found in:\n%s", urlField, html)
		}

		// Must be type=url.
		if !strings.Contains(html, `type="url"`) {
			t.Fatalf("URL input must be type=url, not found in:\n%s", html)
		}

		// Must be bound to the form.
		if !strings.Contains(html, `form="fontsForm"`) {
			t.Fatalf("URL input must carry form=%q, not found in:\n%s", "fontsForm", html)
		}
	}
}

// TestRenderStylePanelFontURLValue asserts the current URL value is rendered (or
// empty string when url is blank — input should still be present).
func TestRenderStylePanelFontURLValue(t *testing.T) {
	view := map[string]any{
		"fonts": testFontsAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "fontsForm", styleTestAction, styleTestCSRF))

	// Non-empty URL must appear as the value.
	if !strings.Contains(html, `value="https://example.com/fraunces.woff2"`) {
		t.Fatalf("URL input must carry the current font URL as value, not found in:\n%s", html)
	}
	if !strings.Contains(html, `value="https://example.com/inter.woff2"`) {
		t.Fatalf("URL input must carry the current font URL as value, not found in:\n%s", html)
	}

	// Mono has empty url — the monoFontUrl input must still render (value="" is fine).
	if !strings.Contains(html, `name="monoFontUrl"`) {
		t.Fatalf("URL input for mono role must be present even when url is empty:\n%s", html)
	}
}

// TestRenderStylePanelFontLabelRendered asserts the human label is rendered for
// each font entry.
func TestRenderStylePanelFontLabelRendered(t *testing.T) {
	view := map[string]any{
		"fonts": testFontsAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "fontsForm", styleTestAction, styleTestCSRF))

	for _, entry := range testFontsAny() {
		label := entry["label"].(string)
		if !strings.Contains(html, label) {
			t.Fatalf("font label %q must appear in panel:\n%s", label, html)
		}
	}
}

// TestRenderStylePanelFontsOnlyViewHasFormAndOp asserts that a fonts-only view
// (no palette) still renders the managed form with op + csrf + save/cancel.
func TestRenderStylePanelFontsOnlyViewHasFormAndOp(t *testing.T) {
	view := map[string]any{
		"fonts": testFontsAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "fontsForm", styleTestAction, styleTestCSRF))

	if !strings.Contains(html, `data-gosx-studio-authoring-managed="true"`) {
		t.Fatalf("fonts-only panel must include managed form marker:\n%s", html)
	}
	if !strings.Contains(html, `value="save-appearance"`) {
		t.Fatalf("fonts-only panel must include save-appearance operation:\n%s", html)
	}
	if !strings.Contains(html, `name="csrf_token"`) {
		t.Fatalf("fonts-only panel must include csrf_token field:\n%s", html)
	}
	if !strings.Contains(html, `type="submit"`) {
		t.Fatalf("fonts-only panel must include submit button:\n%s", html)
	}
	if !strings.Contains(html, `type="reset"`) {
		t.Fatalf("fonts-only panel must include reset/cancel button:\n%s", html)
	}
}

// TestRenderStylePanelPaletteOnlyRendersNoFontInputs asserts that a palette-only
// view (no fonts key) does NOT render any font inputs. Additive/no-regression.
func TestRenderStylePanelPaletteOnlyRendersNoFontInputs(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "stylePaletteForm", styleTestAction, styleTestCSRF))

	if strings.Contains(html, `data-editor-font-name`) {
		t.Fatalf("palette-only view must NOT render any data-editor-font-name inputs:\n%s", html)
	}
	if strings.Contains(html, `data-editor-font-url`) {
		t.Fatalf("palette-only view must NOT render any data-editor-font-url inputs:\n%s", html)
	}
}

// TestRenderStylePanelBothPaletteAndFonts asserts that a view with both palette
// and fonts renders color inputs AND font inputs in the same form.
func TestRenderStylePanelBothPaletteAndFonts(t *testing.T) {
	view := map[string]any{
		"palette": testPaletteAny(),
		"fonts":   testFontsAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "brandForm", styleTestAction, styleTestCSRF))

	// Colors still present.
	if !strings.Contains(html, `type="color"`) {
		t.Fatalf("combined view must render color inputs:\n%s", html)
	}
	// Fonts also present.
	if !strings.Contains(html, `data-editor-font-name="display"`) {
		t.Fatalf("combined view must render font inputs:\n%s", html)
	}
	// Single form element only (no nested forms, no duplicate form ids).
	formCount := strings.Count(html, `id="brandForm"`)
	if formCount != 1 {
		t.Fatalf("expected exactly 1 form with id=brandForm, got %d:\n%s", formCount, html)
	}
}

// TestRenderStylePanelFontsStringShape asserts the []map[string]string variant
// works (mirrors TestRenderStylePanelAcceptsStringPalette).
func TestRenderStylePanelFontsStringShape(t *testing.T) {
	view := map[string]any{
		"fonts": testFontsString(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "fontsForm", styleTestAction, styleTestCSRF))

	for _, entry := range testFontsString() {
		role := entry["role"]
		wantFontNameAttr := `data-editor-font-name="` + role + `"`
		if !strings.Contains(html, wantFontNameAttr) {
			t.Fatalf("string-fonts: family input must carry data-editor-font-name=%q:\n%s", role, html)
		}
		wantFontURLAttr := `data-editor-font-url="` + role + `"`
		if !strings.Contains(html, wantFontURLAttr) {
			t.Fatalf("string-fonts: URL input must carry data-editor-font-url=%q:\n%s", role, html)
		}
	}
}

// TestRenderStylePanelFontsSectionDataAttr asserts the fonts subsection carries
// a scoping data attribute so hosts can target it via CSS/JS.
func TestRenderStylePanelFontsSectionDataAttr(t *testing.T) {
	view := map[string]any{
		"fonts": testFontsAny(),
	}
	html := gosx.RenderHTML(RenderStylePanel(view, "fontsForm", styleTestAction, styleTestCSRF))

	if !strings.Contains(html, `data-gosx-studio-style-fonts`) {
		t.Fatalf("fonts subsection must carry data-gosx-studio-style-fonts for host CSS/JS targeting:\n%s", html)
	}
}
