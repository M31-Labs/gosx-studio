package panels

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

// RenderInspectorControl renders the right widget for a control map entry
// (keyed by "kind") along with hidden formInputs, a label, a Save submit
// button, and a Cancel (reset) button. It is designed to be embedded inside
// any host's inspector panel — it emits no visible platform copy and no
// wrapping section element (the caller owns the outer section and its
// data-attributes so the existing runtime and tests keep matching).
//
// Required control keys:
//   - "kind"        — raw ControlKind string (e.g. "text", "rich-text")
//   - "value"       — current field value
//
// Optional control keys:
//   - "actionLabel" — Save button label; defaults to "Save field"
//   - "formInputs"  — hidden inputs list ([]map[string]string or []map[string]any)
//   - "options"     — for "choice" kind: []map[string]string with "value"/"label" keys
//   - "label"       — field label shown above the widget (may be empty)
//
// formID and inputName must match the <form id> and the visible widget's name.
func RenderInspectorControl(control map[string]any, formID, inputName string) gosx.Node {
	kind := workbenchMapString(control, "kind")
	value := workbenchMapString(control, "value")
	actionLabel := core.FirstNonEmpty(workbenchMapString(control, "actionLabel"), "Save field")
	hiddenInputs := siteMapHiddenInputs(formID, siteMapInputViews(control, "formInputs"), inputName)

	widget := inspectorWidget(kind, value, formID, inputName, control)

	saveBtn := gosx.El("button", gosx.Attrs(
		gosx.Attr("form", formID),
		gosx.Attr("type", "submit"),
	), gosx.Text(actionLabel))

	cancelBtn := gosx.El("button", gosx.Attrs(
		gosx.Attr("form", formID),
		gosx.Attr("type", "reset"),
		gosx.Attr("data-gosx-studio-inspector-cancel", "true"),
	), gosx.Text("Cancel"))

	// Collect children: hidden inputs, then label+widget, then buttons.
	children := make([]gosx.Node, 0, len(hiddenInputs)+3)
	for _, h := range hiddenInputs {
		children = append(children, h)
	}
	children = append(children, widget, saveBtn, cancelBtn)

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("data-gosx-studio-inspector-control", "true"),
		gosx.Attr("data-gosx-studio-inspector-kind", kind),
	), gosx.Fragment(children...))
}

// inspectorWidget returns the kind-specific interactive widget node.
func inspectorWidget(kind, value, formID, inputName string, control map[string]any) gosx.Node {
	switch kind {
	case string(core.ControlRichText):
		return gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("textarea", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("name", inputName),
			), gosx.Text(value)),
		)

	case string(core.ControlNumber):
		return gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "number"),
				gosx.Attr("name", inputName),
				gosx.Attr("value", value),
			)),
		)

	case string(core.ControlLink):
		return gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "url"),
				gosx.Attr("name", inputName),
				gosx.Attr("value", value),
			)),
		)

	case string(core.ControlColor):
		// Color swatch + hex text mirror. The text mirror does NOT submit
		// (no name attr) — it is a display-only companion. Slice 3 will
		// wire a real swatch picker; this stops a bare text box shipping.
		return gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "color"),
				gosx.Attr("name", inputName),
				gosx.Attr("value", value),
			)),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "text"),
				gosx.Attr("value", value),
				gosx.Attr("data-gosx-studio-inspector-color-mirror", "true"),
			)),
		)

	case string(core.ControlToggle):
		// <select> with Show/Hide options; value "true" selects Show.
		showSelected := ""
		hideSelected := ""
		if value == "true" || value == "1" || value == "yes" {
			showSelected = "selected"
		} else {
			hideSelected = "selected"
		}
		showAttrs := []any{gosx.Attr("value", "true")}
		hideAttrs := []any{gosx.Attr("value", "false")}
		if showSelected != "" {
			showAttrs = append(showAttrs, gosx.Attr("selected", "selected"))
		}
		if hideSelected != "" {
			hideAttrs = append(hideAttrs, gosx.Attr("selected", "selected"))
		}
		return gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("select", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("name", inputName),
			),
				gosx.El("option", gosx.Attrs(showAttrs...), gosx.Text("Show")),
				gosx.El("option", gosx.Attrs(hideAttrs...), gosx.Text("Hide")),
			),
		)

	case string(core.ControlChoice):
		options := inspectorControlOptions(control)
		if len(options) > 0 {
			optionNodes := make([]gosx.Node, 0, len(options))
			for _, opt := range options {
				optValue := opt["value"]
				optLabel := opt["label"]
				optAttrs := []any{gosx.Attr("value", optValue)}
				if optValue == value {
					optAttrs = append(optAttrs, gosx.Attr("selected", "selected"))
				}
				optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(optAttrs...), gosx.Text(optLabel)))
			}
			return gosx.El("label", nil,
				gosx.El("span", nil, gosx.Text("Value")),
				gosx.El("select", gosx.Attrs(
					gosx.Attr("form", formID),
					gosx.Attr("name", inputName),
				), gosx.Fragment(optionNodes...)),
			)
		}
		// Fallback: no options provided → text input.
		return gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "text"),
				gosx.Attr("name", inputName),
				gosx.Attr("value", value),
			)),
		)

	case string(core.ControlMedia):
		// Placeholder slot: read-only value display + disabled picker button.
		// Slice 2 wires the real media picker.
		return gosx.El("div", gosx.Attrs(
			gosx.Attr("data-gosx-studio-inspector-slot", "media"),
		),
			gosx.El("span", nil, gosx.Text(value)),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("disabled", "disabled"),
			), gosx.Text("Choose media…")),
		)

	case string(core.ControlText):
		// Explicit case: keep as single-line text input (existing behaviour).
		return gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "text"),
				gosx.Attr("name", inputName),
				gosx.Attr("value", value),
			)),
		)

	default:
		// source, flow, scene-3d, and any unknown kind → text input with slot attr.
		return gosx.El("div", gosx.Attrs(
			gosx.Attr("data-gosx-studio-inspector-slot", kind),
		),
			gosx.El("label", nil,
				gosx.El("span", nil, gosx.Text("Value")),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("form", formID),
					gosx.Attr("type", "text"),
					gosx.Attr("name", inputName),
					gosx.Attr("value", value),
				)),
			),
		)
	}
}

// inspectorControlOptions extracts options from a control map.
// Accepts []map[string]string or []map[string]any (both shapes appear in practice).
func inspectorControlOptions(control map[string]any) []map[string]string {
	raw, ok := control["options"]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case []map[string]string:
		return typed
	case []map[string]any:
		out := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, map[string]string{
				"value": workbenchMapString(item, "value"),
				"label": workbenchMapString(item, "label"),
			})
		}
		return out
	default:
		return nil
	}
}
