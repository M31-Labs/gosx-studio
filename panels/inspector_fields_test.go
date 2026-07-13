package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderInspectorFieldListPreservesRepresentativeControls(t *testing.T) {
	fields := []map[string]any{
		{
			"isArea":  true,
			"id":      "customCss",
			"label":   "Custom CSS",
			"value":   ".hero{color:red}",
			"hasHelp": true,
			"help":    "CSS applied to the preview.",
			"rowAttrs": map[string]any{
				"class":                    "field-row field-row--wide",
				"data-studio-field-source": "type.customCss",
			},
			"controlAttrs": map[string]any{
				"id":              "customCss",
				"name":            "customCss",
				"rows":            7,
				"required":        true,
				"data-editor-css": "true",
			},
		},
		{
			"isSelect": true,
			"id":       "themePalette",
			"label":    "Palette",
			"rowAttrs": map[string]any{
				"class": "field-row",
			},
			"controlAttrs": map[string]any{
				"id":   "themePalette",
				"name": "themePalette",
			},
			"options": []map[string]any{
				{
					"label": "Clay",
					"attrs": map[string]any{"value": "clay", "selected": true},
				},
				{
					"label": "Moss",
					"attrs": map[string]any{"value": "moss"},
				},
			},
		},
		{
			"isCheckbox": true,
			"id":         "showAnnouncement",
			"label":      "Show announcement",
			"rowAttrs":   map[string]any{"class": "field-row"},
			"controlAttrs": map[string]any{
				"id":       "showAnnouncement",
				"name":     "showAnnouncement",
				"type":     "checkbox",
				"checked":  true,
				"disabled": true,
			},
		},
		{
			"isInput": true,
			"id":      "logoUrl",
			"label":   "Logo image",
			"rowAttrs": map[string]any{
				"class":                    "field-row",
				"data-studio-field-source": "brand.logoUrl",
			},
			"controlAttrs": map[string]any{
				"id":                   "logoUrl",
				"name":                 "logoUrl",
				"type":                 "url",
				"value":                "/media/logo.png",
				"list":                 "editor-media-urls",
				"data-editor-logo-url": "true",
			},
		},
	}

	html := gosx.RenderHTML(RenderInspectorFieldList(fields, InspectorFieldListOptions{}))
	for _, want := range []string{
		`data-gosx-studio-inspector-field-list-renderer="gosx-studio"`,
		`class="studio-brand-panel__field-list"`,
		`data-studio-field-source="type.customCss"`,
		`data-editor-css="true"`,
		`rows="7"`,
		`required`,
		`.hero{color:red}`,
		`<small class="field-help">CSS applied to the preview.</small>`,
		`<select`,
		`<option selected value="clay">Clay</option>`,
		`type="checkbox"`,
		`checked`,
		`disabled`,
		`data-editor-logo-url="true"`,
		`list="editor-media-urls"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered inspector field list missing %q:\n%s", want, html)
		}
	}
}

func TestRenderInspectorFieldTextareaPreservesExactValueWhitespace(t *testing.T) {
	wantValue := "\n  .hero {\n    color: red;\n  }\n\n"
	field := map[string]any{
		"isArea": true,
		"id":     "customCss",
		"label":  "Custom CSS",
		"value":  wantValue,
		"controlAttrs": map[string]any{
			"id":   "customCss",
			"name": "customCss",
			"rows": 7,
		},
	}

	html := gosx.RenderHTML(RenderInspectorFieldRow(field, InspectorFieldRowOptions{}))
	textareaStart := strings.Index(html, "<textarea")
	if textareaStart < 0 {
		t.Fatalf("rendered field row missing textarea:\n%s", html)
	}
	bodyStart := strings.Index(html[textareaStart:], ">")
	if bodyStart < 0 {
		t.Fatalf("rendered textarea missing body start:\n%s", html)
	}
	bodyStart += textareaStart + 1
	bodyEnd := strings.Index(html[bodyStart:], "</textarea>")
	if bodyEnd < 0 {
		t.Fatalf("rendered textarea missing close tag:\n%s", html)
	}
	bodyEnd += bodyStart

	if got := html[bodyStart:bodyEnd]; got != wantValue {
		t.Fatalf("rendered textarea value mismatch:\n got: %q\nwant: %q\nhtml: %s", got, wantValue, html)
	}
}

func TestRenderInspectorFieldRowRendersCardActions(t *testing.T) {
	field := map[string]any{
		"isCard":    true,
		"label":     "Calendar",
		"cardTitle": "Nature school schedule",
		"hasHelp":   true,
		"help":      "Open the calendar tool to change schedule slots.",
		"rowAttrs": map[string]any{
			"class":                    "field-row field-row--card",
			"data-studio-field-action": "Open schedule",
		},
		"actions": []map[string]any{
			{
				"label": "Open schedule",
				"attrs": map[string]any{
					"class": "button button--secondary",
					"href":  "/admin/calendar",
				},
			},
			{
				"label":         "Refresh",
				"hasFormAction": true,
				"attrs": map[string]any{
					"class":      "button",
					"type":       "submit",
					"formaction": "/admin/editor/__actions/save",
				},
			},
		},
	}

	html := gosx.RenderHTML(RenderInspectorFieldRow(field, InspectorFieldRowOptions{Wrapper: true}))
	for _, want := range []string{
		`data-studio-advanced-field-row="true"`,
		`data-gosx-studio-inspector-field-row-renderer="gosx-studio"`,
		`data-studio-field-action="Open schedule"`,
		`<div class="studio-source-card">`,
		`<strong>Nature school schedule</strong>`,
		`href="/admin/calendar"`,
		`formaction="/admin/editor/__actions/save"`,
		`<button`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered card row missing %q:\n%s", want, html)
		}
	}
}

func TestRenderInspectorFieldListHomeVariantPreservesRowControlAttrsAndMeta(t *testing.T) {
	fields := []map[string]any{
		{
			"isInput":         true,
			"id":              "heroHeadline",
			"label":           "Headline",
			"kindLabel":       "Text",
			"validationLabel": "Ready",
			"reactionLabel":   "Live preview",
			"rowAttrs": map[string]any{
				"class":                      "field-row field-row--wide",
				"data-studio-field-group":    "copy",
				"data-studio-inspector-for":  "hero",
				"data-studio-field-source":   "hero.headline",
				"data-studio-field-editable": "content",
				"tabindex":                   "-1",
				"data-studio-field-invalid":  false,
			},
			"inputName":         "heroHeadline",
			"inputType":         "text",
			"inputValue":        "Mud Relics",
			"rows":              4,
			"inputRequired":     true,
			"inputEditorSource": "hero-headline",
			"inputFrameTarget":  ".hero h1",
			"inputAttrTarget":   ".hero h1",
			"inputAttrName":     "data-preview",
			"inputAttrPrefix":   "prefix-",
			"inputAttrSuffix":   "-suffix",
		},
		{
			"isArea":           true,
			"id":               "heroSubhead",
			"label":            "Subhead",
			"kindLabel":        "Text area",
			"validationLabel":  "Required",
			"reactionLabel":    "Required before publish",
			"hasHelp":          true,
			"help":             "Shown under the headline.",
			"areaName":         "heroSubhead",
			"areaValue":        "\n  Preserve this copy.\n",
			"rows":             5,
			"areaRequired":     true,
			"areaEditorSource": "hero-subhead",
			"areaFrameTarget":  ".hero .subhead",
			"rowAttrs": map[string]any{
				"class":                     "field-row field-row--wide",
				"data-studio-field-group":   "copy",
				"data-studio-inspector-for": "hero",
				"data-studio-field-source":  "hero.subhead",
				"data-studio-field-invalid": true,
			},
		},
	}

	html := gosx.RenderHTML(RenderInspectorFieldList(fields, InspectorFieldListOptions{
		Class: "studio-home-fields-panel__fields",
		RowOptions: InspectorFieldRowOptions{
			Variant:      InspectorFieldRowVariantHome,
			InvalidLabel: "Needs attention",
		},
	}))
	for _, want := range []string{
		`class="studio-home-fields-panel__fields"`,
		`data-studio-field-group="copy"`,
		`data-studio-inspector-for="hero"`,
		`data-studio-field-source="hero.headline"`,
		`data-studio-field-editable="content"`,
		`data-studio-field-invalid`,
		`<div class="studio-home-field__head">`,
		`<label for="heroHeadline">Headline</label>`,
		`<span>Text</span>`,
		`<div class="studio-home-field__meta">`,
		`<output>Ready</output>`,
		`<small>Live preview</small>`,
		`<small class="studio-home-field__invalid">Needs attention</small>`,
		`name="heroHeadline"`,
		`type="text"`,
		`value="Mud Relics"`,
		`required`,
		`data-editor-source="hero-headline"`,
		`data-editor-frame-target=".hero h1"`,
		`data-editor-frame-attr-target=".hero h1"`,
		`data-editor-frame-attr="data-preview"`,
		`data-editor-frame-attr-prefix="prefix-"`,
		`data-editor-frame-attr-suffix="-suffix"`,
		`<textarea`,
		`name="heroSubhead"`,
		`rows="5"`,
		`Shown under the headline.`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("home inspector field list missing %q:\n%s", want, html)
		}
	}
	if !strings.Contains(html, "\n  Preserve this copy.\n") {
		t.Fatalf("home textarea did not preserve raw whitespace:\n%s", html)
	}
}

func TestRenderInspectorFieldRowHomeVariantRendersCardActions(t *testing.T) {
	field := map[string]any{
		"isCard":          true,
		"id":              "source",
		"label":           "Section source",
		"kindLabel":       "Resource",
		"validationLabel": "Linked",
		"reactionLabel":   "Review home content",
		"cardTitle":       "Home page draft",
		"hasHelp":         true,
		"help":            "Saved with the home page draft.",
		"rowAttrs": map[string]any{
			"class":                               "field-row field-row--wide",
			"data-studio-field-group":             "sources",
			"data-studio-inspector-for":           "hero",
			"data-studio-field-source":            "hero.source",
			"data-studio-field-action":            "Review home content",
			"data-studio-field-action-href":       "/admin/pages",
			"data-studio-field-action-formaction": "/admin/editor/__actions/save",
			"tabindex":                            "-1",
		},
		"actions": []map[string]any{
			{
				"label":         "Refresh",
				"class":         "button button--primary",
				"hasFormAction": true,
				"formAction":    "/admin/editor/__actions/save",
				"formMethod":    "post",
				"name":          "refresh",
				"value":         "hero",
				"confirm":       "Refresh source?",
				"submitAction":  "refreshHero",
			},
			{
				"label": "Pages",
				"class": "button button--secondary",
				"href":  "/admin/pages",
			},
		},
	}

	html := gosx.RenderHTML(RenderInspectorFieldRow(field, InspectorFieldRowOptions{
		Variant:      InspectorFieldRowVariantHome,
		InvalidLabel: "Needs attention",
	}))
	for _, want := range []string{
		`class="studio-source-card"`,
		`<strong>Home page draft</strong>`,
		`Saved with the home page draft.`,
		`data-studio-field-action="Review home content"`,
		`data-studio-field-action-href="/admin/pages"`,
		`data-studio-field-action-formaction="/admin/editor/__actions/save"`,
		`formaction="/admin/editor/__actions/save"`,
		`formmethod="post"`,
		// Edit-revert regression (Noni staging incident, handoff-24): this
		// button submits the SHARED websiteEditorForm without its own form=
		// attribute (relies on DOM nesting) — formnovalidate must be present
		// or a hidden invalid control elsewhere in the form silently cancels
		// the submission with no server round trip, indistinguishable from a
		// revert.
		`formnovalidate="formnovalidate"`,
		`name="refresh"`,
		`value="hero"`,
		`data-admin-confirm="Refresh source?"`,
		`data-studio-submit-action="refreshHero"`,
		`href="/admin/pages"`,
		`data-gosx-link="true"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("home card field row missing %q:\n%s", want, html)
		}
	}
}
