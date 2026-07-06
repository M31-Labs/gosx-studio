package panels

import (
	"strings"

	"m31labs.dev/gosx-studio/authoring"
)

// StyleControlOption is one selectable value in a swatch or segment widget.
// Value is the CSS value; Label is its display name.
type StyleControlOption struct {
	Value string
	Label string
}

// StyleControlSpec describes one per-section style property control.
// Widget must be one of "swatch", "segment", or "number".
// Options is non-nil for swatch and segment widgets; nil for number.
type StyleControlSpec struct {
	Property string
	Label    string
	Widget   string // "swatch" | "segment" | "number"
	Unit     string // optional unit suffix for display (e.g. "px")
	Options  []StyleControlOption
}

// StyleSection is the host-supplied description of one section (block) in
// the page composition whose per-element styles the inspector may edit.
type StyleSection struct {
	// Key is the component key used in set-style mutations and as the
	// selection scope identifier (data-gosx-component / data-studio-inspector-for).
	Key string
	// Label is the display name for the section in the inspector group header.
	Label string
}

// StyleValueLookup returns the current style value for (sectionKey, property)
// from the host's settings/override store. The host passes a closure so the
// studio builder never touches muddy types.
type StyleValueLookup func(sectionKey, property string) string

// BuildStyleControlView assembles the {hasControls, shells, groups} view-data
// map that page.gsx consumes for data.componentStyle. It is the host-agnostic
// core extracted from muddy's editorComponentStyleView.
//
// The caller (muddy) supplies:
//   - sections: the ordered list of page-composition sections to build groups for.
//   - specs: the property control definitions (property, label, widget, options).
//   - valueLookup: a closure returning the current override value for (key, property).
//   - action: the authoring form action URL.
//   - csrfToken: the CSRF token (empty string omits the hasCsrf flag).
//
// The returned map has the exact shape page.gsx iterates:
//
//	{
//	  "hasControls": bool,
//	  "shells": []map[string]any — one per (section × property),
//	  "groups": []map[string]any — one per section, containing "controls".
//	}
//
// page.gsx contract (frozen — do not change keys):
//
//	shells[i]: formID, action, pageKey, componentKey, property, breakpoint, state, csrf, hasCsrf
//	groups[i]: sectionKey, inspectorFor, label, controls[]
//	controls[j]: formID, property, label, value, unit, isSwatch, isSegment, isNumber, options[]
//	options[k]: value, label, checked, swatchStyle
func BuildStyleControlView(
	sections []StyleSection,
	specs []StyleControlSpec,
	valueLookup StyleValueLookup,
	pageKey, action, csrfToken string,
) map[string]any {
	hasCsrf := strings.TrimSpace(csrfToken) != ""
	shells := make([]map[string]any, 0)
	groups := make([]map[string]any, 0)

	for _, section := range sections {
		key := strings.TrimSpace(section.Key)
		if key == "" {
			continue
		}
		controls := make([]map[string]any, 0, len(specs))
		for _, spec := range specs {
			value := ""
			if valueLookup != nil {
				value = valueLookup(key, spec.Property)
			}
			formID := "editorSetStyle-" + authoring.StyleFormIDPart(key) + "-" + authoring.StyleFormIDPart(spec.Property)
			options := make([]map[string]any, 0, len(spec.Options))
			for _, opt := range spec.Options {
				options = append(options, map[string]any{
					"value":       opt.Value,
					"label":       opt.Label,
					"checked":     opt.Value == value,
					"swatchStyle": "background:" + opt.Value,
				})
			}
			controls = append(controls, map[string]any{
				"formID":    formID,
				"property":  spec.Property,
				"label":     spec.Label,
				"value":     value,
				"unit":      spec.Unit,
				"isSwatch":  spec.Widget == "swatch",
				"isSegment": spec.Widget == "segment",
				"isNumber":  spec.Widget == "number",
				"options":   options,
			})
			shells = append(shells, map[string]any{
				"formID":       formID,
				"action":       action,
				"pageKey":      pageKey,
				"componentKey": key,
				"property":     spec.Property,
				"breakpoint":   authoring.StyleBreakpointBase,
				"state":        authoring.StyleStateDefault,
				"csrf":         csrfToken,
				"hasCsrf":      hasCsrf,
			})
		}
		groups = append(groups, map[string]any{
			"sectionKey":   key,
			"inspectorFor": key,
			"label":        section.Label,
			"controls":     controls,
		})
	}
	return map[string]any{
		"hasControls": len(shells) > 0,
		"shells":      shells,
		"groups":      groups,
	}
}

// ConfigControlOption is one selectable value in a config select widget.
type ConfigControlOption struct {
	Value string
	Label string
}

// ConfigControlSpec describes one config field control for a section template.
// Widget must be one of "select", "text", or "number".
type ConfigControlSpec struct {
	Field   string // the field key within the section's Values map
	Label   string
	Widget  string // "select" | "text" | "number"
	Options []ConfigControlOption
}

// ConfigSection carries one section's identity plus the specs applicable to it
// (already filtered by template key in the host).
type ConfigSection struct {
	// Key is the component/section key.
	Key string
	// Label is the display name shown in the inspector group header.
	Label string
	// Specs is the list of config controls for this section's template type.
	Specs []ConfigControlSpec
	// ValueLookup returns the current field value for the given field key.
	ValueLookup func(field string) string
}

// BuildSectionConfigView assembles the {hasControls, shells, groups} view-data
// map that page.gsx consumes for data.componentConfig. It is the host-agnostic
// core extracted from muddy's editorComponentConfigView.
//
// The caller supplies ConfigSection entries pre-filtered to only the sections
// that have applicable config specs. Sections with empty Key or empty Specs
// are skipped.
//
// page.gsx contract (frozen — do not change keys):
//
//	shells[i]: formID, action, pageKey, componentKey, controlKey, binding, csrf, hasCsrf
//	groups[i]: sectionKey, inspectorFor, label, controls[]
//	controls[j]: formID, label, value, isSelect, isNumber, isText, options[]
//	options[k]: value, label, checked
func BuildSectionConfigView(
	sections []ConfigSection,
	pageKey, action, csrfToken string,
) map[string]any {
	empty := map[string]any{
		"hasControls": false,
		"shells":      []map[string]any{},
		"groups":      []map[string]any{},
	}
	hasCsrf := strings.TrimSpace(csrfToken) != ""
	shells := make([]map[string]any, 0)
	groups := make([]map[string]any, 0)

	for _, section := range sections {
		key := strings.TrimSpace(section.Key)
		if key == "" || len(section.Specs) == 0 {
			continue
		}
		controls := make([]map[string]any, 0, len(section.Specs))
		for _, spec := range section.Specs {
			formID := "editorCfg-" + authoring.StyleFormIDPart(key) + "-" + authoring.StyleFormIDPart(spec.Field)
			value := ""
			if section.ValueLookup != nil {
				value = section.ValueLookup(spec.Field)
			}
			options := make([]map[string]any, 0, len(spec.Options))
			for _, opt := range spec.Options {
				options = append(options, map[string]any{
					"value":   opt.Value,
					"label":   opt.Label,
					"checked": opt.Value == value,
				})
			}
			controls = append(controls, map[string]any{
				"formID":   formID,
				"label":    spec.Label,
				"value":    value,
				"isSelect": spec.Widget == "select",
				"isNumber": spec.Widget == "number",
				"isText":   spec.Widget == "text",
				"options":  options,
			})
			shells = append(shells, map[string]any{
				"formID":       formID,
				"action":       action,
				"pageKey":      pageKey,
				"componentKey": key,
				"controlKey":   spec.Field,
				"binding":      authoring.SectionFieldBinding(key, spec.Field),
				"csrf":         csrfToken,
				"hasCsrf":      hasCsrf,
			})
		}
		groups = append(groups, map[string]any{
			"sectionKey":   key,
			"inspectorFor": key,
			"label":        section.Label,
			"controls":     controls,
		})
	}
	if len(shells) == 0 {
		return empty
	}
	return map[string]any{
		"hasControls": true,
		"shells":      shells,
		"groups":      groups,
	}
}

// customCSSFormID is the stable form id for the custom-CSS managed form.
// It matches the id muddy's editorCustomCssView has always emitted and that
// page.gsx references via data.customCssControl.formID.
const customCSSFormID = "editorCustomCssForm"

// BuildCustomCSSView assembles the view-data map that page.gsx consumes for
// data.customCssControl. It is the host-agnostic core extracted from muddy's
// editorCustomCssView.
//
// page.gsx contract (frozen — do not change keys):
//
//	{ formID, action, value, csrf, hasCsrf }
func BuildCustomCSSView(currentCSS, action, csrfToken string) map[string]any {
	return map[string]any{
		"formID":  customCSSFormID,
		"action":  action,
		"value":   currentCSS,
		"csrf":    csrfToken,
		"hasCsrf": strings.TrimSpace(csrfToken) != "",
	}
}
