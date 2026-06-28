package studio

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
