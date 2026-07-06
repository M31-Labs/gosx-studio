package sitemap

// This file carries small, dependency-free helpers and one larger widget
// renderer that sitemap_board.go, sitemap_authoring_panels.go,
// sitemap_authoring_forms.go, and site_navigator_panel.go need but that
// still live as originals in not-yet-migrated root files
// (workbench_toolbar.go, workbench_frame.go, block_library_panel.go,
// inspector.go — see .tiller/scratch/gosx-studio-restructure-spec-v0.1.md,
// panels/shell rows in §1). sitemap may import only core and authoring (§1
// Import DAG), and those root files belong to the panels (Slice 6) and
// shell (Slice 8) packages, so sitemap cannot reach their implementations
// directly without an import that the DAG forbids (a peer/upward import).
// Rather than force that import, this slice carries its own copy, matching
// the existing precedent of duplicated helpers the spec itself documents
// (§1 "Text helpers" seam, and canvas/support.go's identical treatment of
// this same map-shape family for Slice 4) until a later slice promotes a
// single shared home and every copy converges.

import (
	"sort"
	"strings"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

// workbenchViewMap reads values[key] as a map[string]any, converting a
// map[string]string shape as needed. Mirrors workbench_frame.go's
// workbenchViewMap.
func workbenchViewMap(view map[string]any, key string) map[string]any {
	if view == nil {
		return nil
	}
	switch typed := view[key].(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = value
		}
		return out
	default:
		return nil
	}
}

// workbenchViewBool reads values[key] as a bool, accepting a case-insensitive
// "true" string as truthy. Mirrors workbench_frame.go's workbenchViewBool
// (identical to workbench_toolbar.go's workbenchMapBool).
func workbenchViewBool(view map[string]any, key string) bool {
	if view == nil {
		return false
	}
	switch typed := view[key].(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

// workbenchMapBool is the workbench_toolbar.go-named twin of
// workbenchViewBool; sitemap_board.go calls both bare names.
func workbenchMapBool(values map[string]any, key string) bool {
	return workbenchViewBool(values, key)
}

// workbenchViewString reads values[key] as a trimmed string, formatting
// non-string values with core.FmtAny. Mirrors workbench_toolbar.go's
// workbenchViewString.
func workbenchViewString(view map[string]any, key string) string {
	if view == nil {
		return ""
	}
	value, ok := view[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(core.FmtAny(typed))
	}
}

// workbenchMapString is the workbench_toolbar.go-named twin of
// workbenchViewString; sitemap_board.go/sitemap_authoring_panels.go call
// both bare names.
func workbenchMapString(values map[string]any, key string) string {
	return workbenchViewString(values, key)
}

// workbenchViewMapList reads values[key] as a []map[string]any, converting a
// []map[string]string shape as needed. Mirrors workbench_toolbar.go's
// workbenchViewMapList (identical to sitemap_authoring_panels.go's
// SiteMapMapList, which stays local to this package).
func workbenchViewMapList(view map[string]any, key string) []map[string]any {
	if view == nil {
		return nil
	}
	switch typed := view[key].(type) {
	case []map[string]any:
		return typed
	case []map[string]string:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			next := make(map[string]any, len(item))
			for key, value := range item {
				next[key] = value
			}
			out = append(out, next)
		}
		return out
	default:
		return nil
	}
}

// workbenchNodeEmpty reports whether a rendered gosx.Node is the empty/zero
// fragment. Mirrors workbench_toolbar.go's workbenchNodeEmpty.
func workbenchNodeEmpty(node gosx.Node) bool {
	html := gosx.RenderHTML(node)
	return html == "" || html == "<></>"
}

// appendBlockLibraryPanelAttrs appends the caller-supplied root attribute map
// (already rendered as gosx.Attr values) onto attrs. Mirrors
// block_library_panel.go's appendBlockLibraryPanelAttrs.
func appendBlockLibraryPanelAttrs(attrs []any, values map[string]any) []any {
	return append(attrs, blockLibraryPanelMapAttrs(values)...)
}

// blockLibraryPanelMapAttrs renders a caller-supplied attribute map into
// gosx.Attr values in deterministic (sorted-key) order, coercing bool values
// on aria-*/data-* names via core.BoolAttr. Mirrors block_library_panel.go's
// blockLibraryPanelMapAttrs.
func blockLibraryPanelMapAttrs(values map[string]any) []any {
	attrs := make([]any, 0, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		value := values[key]
		if value == nil {
			continue
		}
		if typed, ok := value.(bool); ok && (strings.HasPrefix(name, "aria-") || strings.HasPrefix(name, "data-")) {
			value = core.BoolAttr(typed)
		}
		attrs = append(attrs, gosx.Attr(name, value))
	}
	return attrs
}

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
// formID and inputName must match the <form id> and the visible widget's
// name.
//
// This is a frozen, byte-identical copy of inspector.go's
// RenderInspectorControl (panels package territory, Slice 6):
// renderSiteMapEditableControlPanel (sitemap_authoring_panels.go) renders the
// same inspector widget the editor's inspector rail uses, but sitemap may
// import only core and authoring (§1 Import DAG) — panels sits beside
// sitemap, not below it, so importing it would be a forbidden peer import.
// Reconcile when a later slice gives this widget a shared home.
func RenderInspectorControl(control map[string]any, formID, inputName string) gosx.Node {
	kind := workbenchMapString(control, "kind")
	value := workbenchMapString(control, "value")
	actionLabel := core.FirstNonEmpty(workbenchMapString(control, "actionLabel"), "Save field")
	hiddenInputs := SiteMapHiddenInputs(formID, SiteMapInputViews(control, "formInputs"), inputName)

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

// inspectorWidget returns the kind-specific interactive widget node. Frozen
// copy of inspector.go's inspectorWidget; see RenderInspectorControl's doc
// comment.
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
		// (no name attr) — it is a display-only companion.
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
// Accepts []map[string]string or []map[string]any (both shapes appear in
// practice). Frozen copy of inspector.go's inspectorControlOptions; see
// RenderInspectorControl's doc comment.
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

// siteMapEngineDefaultName is the default engine name rendered into the
// site-map engine host's Name field when a caller omits an explicit one. It
// mirrors the root package's SiteMapEngineName constant (shell.go, still
// Slice-8 territory: it is declared alongside CanvasEngineName/
// BlockLayoutEngineName/FlowDesignerName/Showcase3DEngineName). sitemap
// cannot import shell (peer import forbidden per the DAG), so this is a
// frozen, byte-identical copy of the same literal value rather than a shared
// reference; keep it in sync with shell.go until a later slice unifies the
// engine-name registry (matches canvas/support.go's identical treatment of
// BlockLayoutEngineName in Slice 4).
const siteMapEngineDefaultName = "GoSXStudioSiteMap"
