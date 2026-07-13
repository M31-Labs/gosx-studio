package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderLookPanelFull(t *testing.T) {
	html := gosx.RenderHTML(RenderLookPanel(lookPanelTestView(), LookPanelOptions{}))

	for _, fragment := range []string{
		`<section class="editor-panel editor-panel--look-studio studio-look-panel" data-studio-look-panel="true" data-studio-mode-panel="look" data-studio-panel="style" data-studio-engine-source="gosx" data-studio-look-group-active="theme" data-gosx-studio-look-panel-renderer="gosx-studio">`,
		`<input class="studio-look-panel__group-input" id="studioLookGroupTheme" type="radio" name="studioLookGroup" value="theme" checked aria-label="Theme" />`,
		`<label class="studio-look-panel__group" for="studioLookGroupDetails" data-studio-look-group-label="details"><strong>Details</strong><small>Navigation and motion.</small></label>`,
		`<div class="studio-look-panel__scope" data-studio-look-scope="true">`,
		`data-studio-style-scope="true" data-style-valid="true"`,
		`data-studio-style-scope-block="true"`,
		`data-studio-style-validity="true"`,
		`data-studio-style-system-label="true"`,
		`data-studio-style-scope-field="true"`,
		`data-studio-style-breakpoint-label="true"`,
		`data-studio-style-state="hover"`,
		`data-studio-style-state-label="true"`,
		`data-studio-style-workbench="true"`,
		`data-studio-style-impact-panel="true"`,
		`data-studio-style-impact-label="true"`,
		`data-studio-style-impact-summary="true"`,
		`data-studio-style-impact-count="true"`,
		`data-studio-style-impact-scope="true"`,
		`data-studio-style-impact-state="true"`,
		`data-studio-style-recipe="layout"`,
		`data-studio-style-readout="heroWidth"`,
		`data-studio-style-mark="1"`,
		`data-studio-style-reset="heroWidth"`,
		`<button aria-pressed="true" class="is-selected" data-studio-style-control="heroWidth" data-studio-style-value="wide" type="button">Wide</button>`,
		`<section class="studio-look-panel__group-slot" data-studio-look-group-slot="theme" data-studio-look-group-selected="true">`,
		`data-studio-look-group-body="theme"`,
		`class="look-choice-card" data-kit-card="starter"`,
		`<input checked name="themeKit" type="radio" value="starter" />`,
		`data-studio-look-group-slot="layout" data-studio-look-group-selected="false"`,
		`class="custom-template-builder" data-custom-template-builder="true"`,
		`data-editor-custom-template-name="true"`,
		`<select data-editor-source="theme.heroShape" id="heroShape" name="heroShape">`,
		`data-studio-look-group-slot="color" data-studio-look-group-selected="false"`,
		`<div class="theme-swatch-row" data-editor-theme-swatches="true"><span class="theme-swatch" style="--swatch: #111" title="Ink"></span></div>`,
		`data-studio-look-group-slot="details" data-studio-look-group-selected="false"`,
		`<fieldset class="look-radio-group" data-image-crop="true"><legend>Image crop</legend><label><input checked name="imageCrop" type="radio" value="cover" /> Cover</label></fieldset>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("look panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(html, notWant) {
			t.Fatalf("look panel must not include %q:\n%s", notWant, html)
		}
	}
}

// TestRenderLookPanelImpactCardCarriesTitleTooltips guards wave 3B's
// residual #6 fix (LOOK Recipes card crowding): the impact card's
// label/summary/count/scope/state values must carry a title tooltip when
// non-empty (the #15 pattern renderLookScopeField already established) so
// the full value stays discoverable on hover/focus once the card's own
// width-constrained CSS (host-side, see public/site.css
// .studio-style-impact/.studio-style-impact__meta) truncates it.
func TestRenderLookPanelImpactCardCarriesTitleTooltips(t *testing.T) {
	html := gosx.RenderHTML(RenderLookPanel(lookPanelTestView(), LookPanelOptions{}))
	for _, want := range []string{
		`data-studio-style-impact-label="true" title="Hero width"`,
		`data-studio-style-impact-summary="true" title="One field affected."`,
		`data-studio-style-impact-count="true" title="1 affected"`,
		`data-studio-style-impact-scope="true" title="site &gt; home"`,
		`data-studio-style-impact-state="true" title="default / desktop"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("look panel impact card missing tooltip %q:\n%s", want, html)
		}
	}
}

// TestRenderLookPanelImpactCardOmitsTooltipWhenValueEmpty guards the
// no-crash/no-noise empty path: an empty value must not emit a bare
// title="" attribute.
func TestRenderLookPanelImpactCardOmitsTooltipWhenValueEmpty(t *testing.T) {
	view := lookPanelTestView()
	settings := view["settings"].(map[string]any)
	workbench := settings["workbench"].(map[string]any)
	workbench["impact"] = map[string]any{"kicker": "Preview impact"}
	html := gosx.RenderHTML(RenderLookPanel(view, LookPanelOptions{}))
	if strings.Contains(html, `data-studio-style-impact-label="true" title=`) {
		t.Fatalf("empty impact label must not carry a title tooltip:\n%s", html)
	}
}

func TestRenderLookPanelMinimalKeepsRootAndSlots(t *testing.T) {
	html := gosx.RenderHTML(RenderLookPanel(map[string]any{}, LookPanelOptions{}))
	for _, fragment := range []string{
		`data-studio-look-panel="true"`,
		`data-gosx-studio-look-panel-renderer="gosx-studio"`,
		`<div class="studio-look-panel__groups" role="radiogroup" aria-label=""></div>`,
		`data-studio-look-scope="true"`,
		`data-studio-look-group-slot="theme"`,
		`data-studio-look-group-body="theme"`,
		`data-studio-look-group-slot="layout"`,
		`data-studio-look-group-slot="color"`,
		`data-studio-look-group-slot="details"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("minimal look panel missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderLookPanelRootAttrOverride(t *testing.T) {
	html := gosx.RenderHTML(RenderLookPanel(lookPanelTestView(), LookPanelOptions{
		RootAttrs: map[string]any{
			"class":                  "custom-look-panel",
			"data-studio-panel":      "custom-style",
			"data-custom-look":       true,
			"aria-label":             "Look override",
			"data-studio-look-extra": "test",
		},
	}))

	for _, fragment := range []string{
		`class="custom-look-panel"`,
		`data-studio-panel="custom-style"`,
		`data-custom-look="true"`,
		`aria-label="Look override"`,
		`data-studio-look-extra="test"`,
		`data-gosx-studio-look-panel-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("look root attr override missing %q:\n%s", fragment, html)
		}
	}
}

func lookPanelTestView() map[string]any {
	return map[string]any{
		"kicker":          "Look",
		"title":           "Visual editor",
		"summary":         "Grouped choices.",
		"groupLabel":      "Look groups",
		"defaultGroupKey": "theme",
		"groups": []map[string]any{
			{"key": "theme", "inputID": "studioLookGroupTheme", "label": "Theme", "summary": "Starter kit.", "selected": true},
			{"key": "layout", "inputID": "studioLookGroupLayout", "label": "Layout", "summary": "Hero shape."},
			{"key": "color", "inputID": "studioLookGroupColor", "label": "Color", "summary": "Palette."},
			{"key": "details", "inputID": "studioLookGroupDetails", "label": "Details", "summary": "Navigation and motion."},
		},
		"settings": map[string]any{
			"scope": map[string]any{
				"class":           "studio-style-scope",
				"valid":           true,
				"blockLabel":      "Hero",
				"validityLabel":   "Ready",
				"systemID":        "site",
				"pageLabel":       "Home",
				"fieldLabel":      "Hero title",
				"breakpointLabel": "Desktop",
				"activeState":     "Default",
				"states": []map[string]any{
					{"key": "default", "label": "Default", "active": true},
					{"key": "hover", "label": "Hover"},
				},
			},
			"workbench": map[string]any{
				"class":  "studio-style-workbench",
				"kicker": "Recipes",
				"title":  "Visual system",
				"impact": map[string]any{
					"kicker":     "Preview impact",
					"label":      "Hero width",
					"summary":    "One field affected.",
					"countLabel": "1 affected",
					"scopeLabel": "site > home",
					"stateLabel": "default / desktop",
				},
				"groups": []map[string]any{
					{
						"key":             "layout",
						"label":           "Layout",
						"visualFullClass": "studio-style-visual studio-style-visual--layout",
						"readout":         map[string]any{"field": "heroWidth", "label": "Wide"},
						"marks":           []map[string]any{{"key": "0"}, {"key": "1"}},
						"controls": []map[string]any{
							{
								"fieldName": "heroWidth",
								"label":     "Hero width",
								"options": []map[string]any{
									{
										"label": "Wide",
										"attrs": map[string]any{
											"type":                      "button",
											"data-studio-style-control": "heroWidth",
											"data-studio-style-value":   "wide",
											"aria-pressed":              true,
											"class":                     "is-selected",
										},
									},
								},
							},
						},
					},
				},
			},
			"kitGroup": map[string]any{
				"class":     "look-choice-group",
				"legend":    "Starter kit",
				"gridClass": "look-choice-grid",
				"cards": []map[string]any{
					{
						"label":     "Starter",
						"summary":   "Basic kit",
						"cardAttrs": map[string]any{"class": "look-choice-card", "data-kit-card": "starter"},
						"inputAttrs": map[string]any{
							"type":    "radio",
							"name":    "themeKit",
							"value":   "starter",
							"checked": true,
						},
					},
				},
			},
			"templateGroup": map[string]any{
				"class":     "look-choice-group",
				"legend":    "Template",
				"gridClass": "look-choice-grid",
				"cards":     []map[string]any{},
			},
			"customBuilderClass": "custom-template-builder",
			"customNameField": map[string]any{
				"id":       "customTemplateName",
				"label":    "Custom name",
				"value":    "Landing",
				"isInput":  true,
				"rowAttrs": map[string]any{"class": "field-row field-row--wide"},
				"controlAttrs": map[string]any{
					"id":                               "customTemplateName",
					"name":                             "customTemplateName",
					"type":                             "text",
					"value":                            "Landing",
					"data-editor-custom-template-name": "true",
				},
			},
			"customControls": []map[string]any{
				lookPanelSelectControl("heroShape", "Hero shape", "theme.heroShape", "split"),
			},
			"palette": lookPanelSelectControl("palette", "Palette", "theme.palette", "earth"),
			"swatches": []map[string]any{
				{"style": "--swatch: #111", "label": "Ink"},
			},
			"styleControls": []map[string]any{
				lookPanelSelectControl("buttonStyle", "Button style", "theme.buttonStyle", "rounded"),
			},
			"imageCrop": map[string]any{
				"legend": "Image crop",
				"attrs":  map[string]any{"class": "look-radio-group", "data-image-crop": "true"},
				"options": []map[string]any{
					{
						"label": "Cover",
						"attrs": map[string]any{"name": "imageCrop", "type": "radio", "value": "cover", "checked": true},
					},
				},
			},
		},
	}
}

func lookPanelSelectControl(id, label, source, value string) map[string]any {
	return map[string]any{
		"id":       id,
		"label":    label,
		"rowAttrs": map[string]any{"class": "field-row"},
		"controlAttrs": map[string]any{
			"id":                 id,
			"name":               id,
			"data-editor-source": source,
		},
		"options": []map[string]any{
			{"label": value, "attrs": map[string]any{"value": value, "selected": true}},
		},
	}
}
