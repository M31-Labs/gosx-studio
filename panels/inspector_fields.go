package panels

import (
	"fmt"
	"sort"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

type InspectorFieldListOptions struct {
	Class      string
	Attrs      map[string]any
	RowOptions InspectorFieldRowOptions
}

type InspectorFieldRowOptions struct {
	Wrapper      bool
	Variant      string
	InvalidLabel string
}

const InspectorFieldRowVariantHome = "home"

func RenderInspectorFieldList(fields []map[string]any, options InspectorFieldListOptions) gosx.Node {
	children := make([]gosx.Node, 0, len(fields))
	for _, field := range fields {
		children = append(children, RenderInspectorFieldRow(field, options.RowOptions))
	}
	attrs := []any{
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-brand-panel__field-list")),
		gosx.Attr("data-gosx-studio-inspector-field-list-renderer", "gosx-studio"),
	}
	attrs = appendInspectorFieldAttrs(attrs, options.Attrs)
	return gosx.El("div", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func RenderInspectorFieldRow(field map[string]any, options InspectorFieldRowOptions) gosx.Node {
	row := renderInspectorFieldRow(field, options)
	if !options.Wrapper {
		return row
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("data-studio-advanced-field-row", "true"),
		gosx.Attr("data-gosx-studio-inspector-field-row-renderer", "gosx-studio"),
	), row)
}

func renderInspectorFieldRow(field map[string]any, options InspectorFieldRowOptions) gosx.Node {
	if options.Variant == InspectorFieldRowVariantHome {
		return renderHomeInspectorFieldRow(field, options)
	}
	children := make([]gosx.Node, 0, 4)
	if core.WorkbenchViewBool(field, "isCard") {
		children = append(children,
			gosx.El("label", nil, gosx.Text(core.WorkbenchViewString(field, "label"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-source-card")),
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(field, "cardTitle"))),
				renderInspectorFieldHelp(field),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(renderInspectorFieldActions(field)...)),
			),
		)
		return gosx.El("div", gosx.Attrs(inspectorFieldMapAttrs(core.WorkbenchViewMap(field, "rowAttrs"))...), gosx.Fragment(children...))
	}

	children = append(children, gosx.El("label", gosx.Attrs(gosx.Attr("for", core.WorkbenchViewString(field, "id"))), gosx.Text(core.WorkbenchViewString(field, "label"))))
	children = append(children, renderInspectorFieldControl(field))
	children = append(children, renderInspectorFieldHelp(field))

	return gosx.El("div", gosx.Attrs(inspectorFieldMapAttrs(core.WorkbenchViewMap(field, "rowAttrs"))...), gosx.Fragment(children...))
}

func renderHomeInspectorFieldRow(field map[string]any, options InspectorFieldRowOptions) gosx.Node {
	return gosx.El("div", gosx.Attrs(inspectorFieldMapAttrs(core.WorkbenchViewMap(field, "rowAttrs"))...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-home-field__head")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", core.WorkbenchViewString(field, "id"))), gosx.Text(core.WorkbenchViewString(field, "label"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(field, "kindLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-home-field__meta")),
			// Wave 3A (vocabulary pruning, fix (e)): a per-field "READY" chip
			// on every valid field was pure noise (162 selects' worth of
			// context also meant dozens of these on one page) — only an
			// actual validation problem is worth a chip here, so this output
			// is hidden whenever the field isn't invalid.
			gosx.El("output", gosx.Attrs(gosx.Attr("hidden", !core.WorkbenchViewBool(field, "invalid"))), gosx.Text(core.WorkbenchViewString(field, "validationLabel"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(field, "reactionLabel"))),
			gosx.El("small", gosx.Attrs(gosx.Attr("class", "studio-home-field__invalid")), gosx.Text(options.InvalidLabel)),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-source-card"),
			gosx.Attr("hidden", !core.WorkbenchViewBool(field, "isCard")),
		),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(field, "cardTitle"))),
			gosx.El("small", gosx.Attrs(
				gosx.Attr("class", "field-help"),
				gosx.Attr("hidden", !core.WorkbenchViewBool(field, "hasHelp")),
			), gosx.Text(core.WorkbenchViewString(field, "help"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(renderHomeInspectorFieldActions(field)...)),
		),
		renderHomeInspectorTextarea(field),
		renderHomeInspectorSelect(field),
		renderHomeInspectorInput(field),
		gosx.El("small", gosx.Attrs(
			gosx.Attr("class", "field-help"),
			gosx.Attr("hidden", core.WorkbenchViewBool(field, "isCard") || !core.WorkbenchViewBool(field, "hasHelp")),
		), gosx.Text(core.WorkbenchViewString(field, "help"))),
	)
}

func renderHomeInspectorFieldActions(field map[string]any) []gosx.Node {
	actions := siteMapMapList(field, "actions")
	nodes := make([]gosx.Node, 0, len(actions)*2)
	for _, action := range actions {
		hasFormAction := core.WorkbenchViewBool(action, "hasFormAction")
		nodes = append(nodes,
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", core.WorkbenchViewString(action, "class")),
				gosx.Attr("type", "submit"),
				gosx.Attr("formaction", core.WorkbenchViewString(action, "formAction")),
				gosx.Attr("formmethod", core.WorkbenchViewString(action, "formMethod")),
				// This button submits the SHARED websiteEditorForm (every
				// Home-mode field row renders inside that one <form>; every
				// panel's fields — including ones currently hidden by
				// mode-gating CSS — are "listed" constraint-validation
				// candidates of it). Native HTML5 validation runs BEFORE the
				// browser dispatches the form's "submit" event; if ANY
				// control anywhere in the form is invalid, the browser
				// silently cancels the submission with no submit event at
				// all, regardless of which button was clicked. formnovalidate
				// exempts THIS submission from that shared-form validation
				// surface, mirroring every other workbench-form action button
				// (Publish/Discard/Schedule in publish_panel.go, Restore in
				// revision_history_panel.go, Save navigation in
				// navigation_panel.go, Save checkout in checkout_panel.go,
				// Publish flow in flow_designer_panel.go) — a Home field's
				// formaction-driven action (e.g. a future "detach shared
				// component"/"reset to default" button) must behave the same
				// way or it would silently no-op whenever an unrelated hidden
				// panel happens to carry an invalid value. Present
				// unconditionally alongside formaction/formmethod above
				// (which are also always rendered, gated only by the sibling
				// hidden attribute below), so it costs nothing when
				// hasFormAction is false.
				gosx.Attr("formnovalidate", "formnovalidate"),
				gosx.Attr("name", core.WorkbenchViewString(action, "name")),
				gosx.Attr("value", inspectorFieldRawString(action, "value")),
				gosx.Attr("data-admin-confirm", core.WorkbenchViewString(action, "confirm")),
				gosx.Attr("data-studio-submit-action", core.WorkbenchViewString(action, "submitAction")),
				gosx.Attr("data-studio-field-action-formaction", core.WorkbenchViewString(action, "formAction")),
				gosx.Attr("hidden", !hasFormAction),
			), gosx.Text(core.WorkbenchViewString(action, "label"))),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", core.WorkbenchViewString(action, "class")),
				gosx.Attr("href", core.WorkbenchViewString(action, "href")),
				gosx.Attr("data-gosx-link", "true"),
				gosx.Attr("hidden", hasFormAction),
			), gosx.Text(core.WorkbenchViewString(action, "label"))),
		)
	}
	return nodes
}

// handoff-4 (text truncation sweep): the Home inspector's textarea/input
// value previews (Tagline/Headline/Subhead/CTA label/CTA URL, etc.) render
// at a fixed right-rail width and clip long saved values with no way to
// read the rest ("Small-batch ceramics for daily ritua…") — a title
// attribute mirroring the current value gives every truncated preview a
// native hover/focus tooltip with the full text, at zero layout cost.
func renderHomeInspectorTextarea(field map[string]any) gosx.Node {
	value := inspectorFieldRawString(field, "areaValue")
	attrs := []any{
		gosx.Attr("id", core.WorkbenchViewString(field, "id")),
		gosx.Attr("name", core.WorkbenchViewString(field, "areaName")),
		gosx.Attr("rows", inspectorFieldValue(field, "rows")),
		gosx.Attr("placeholder", core.WorkbenchViewString(field, "placeholder")),
		gosx.Attr("required", core.WorkbenchViewBool(field, "areaRequired")),
		gosx.Attr("disabled", core.WorkbenchViewBool(field, "areaDisabled")),
		gosx.Attr("data-editor-source", core.WorkbenchViewString(field, "areaEditorSource")),
		gosx.Attr("data-editor-frame-target", core.WorkbenchViewString(field, "areaFrameTarget")),
		gosx.Attr("data-editor-frame-attr-target", core.WorkbenchViewString(field, "areaAttrTarget")),
		gosx.Attr("data-editor-frame-attr", core.WorkbenchViewString(field, "areaAttrName")),
		gosx.Attr("data-editor-frame-attr-prefix", core.WorkbenchViewString(field, "areaAttrPrefix")),
		gosx.Attr("data-editor-frame-attr-suffix", core.WorkbenchViewString(field, "areaAttrSuffix")),
		gosx.Attr("hidden", !core.WorkbenchViewBool(field, "isArea")),
	}
	if value != "" {
		attrs = append(attrs, gosx.Attr("title", value))
	}
	return gosx.El("textarea", gosx.Attrs(attrs...), gosx.Text(value))
}

func renderHomeInspectorSelect(field map[string]any) gosx.Node {
	optionNodes := []gosx.Node{}
	for _, option := range siteMapMapList(field, "options") {
		optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", core.WorkbenchViewString(option, "value")),
			gosx.Attr("selected", core.WorkbenchViewBool(option, "selected")),
		), gosx.Text(core.WorkbenchViewString(option, "label"))))
	}
	return gosx.El("select", gosx.Attrs(
		gosx.Attr("id", core.WorkbenchViewString(field, "id")),
		gosx.Attr("name", core.WorkbenchViewString(field, "selectName")),
		gosx.Attr("required", core.WorkbenchViewBool(field, "selectRequired")),
		gosx.Attr("disabled", core.WorkbenchViewBool(field, "selectDisabled")),
		gosx.Attr("data-editor-source", core.WorkbenchViewString(field, "selectEditorSource")),
		gosx.Attr("data-editor-frame-target", core.WorkbenchViewString(field, "selectFrameTarget")),
		gosx.Attr("data-editor-frame-attr-target", core.WorkbenchViewString(field, "selectAttrTarget")),
		gosx.Attr("data-editor-frame-attr", core.WorkbenchViewString(field, "selectAttrName")),
		gosx.Attr("data-editor-frame-attr-prefix", core.WorkbenchViewString(field, "selectAttrPrefix")),
		gosx.Attr("data-editor-frame-attr-suffix", core.WorkbenchViewString(field, "selectAttrSuffix")),
		gosx.Attr("hidden", !core.WorkbenchViewBool(field, "isSelect")),
	), gosx.Fragment(optionNodes...))
}

func renderHomeInspectorInput(field map[string]any) gosx.Node {
	value := inspectorFieldRawString(field, "inputValue")
	attrs := []any{
		gosx.Attr("id", core.WorkbenchViewString(field, "id")),
		gosx.Attr("name", core.WorkbenchViewString(field, "inputName")),
		gosx.Attr("type", core.WorkbenchViewString(field, "inputType")),
		gosx.Attr("value", value),
		gosx.Attr("placeholder", core.WorkbenchViewString(field, "placeholder")),
		gosx.Attr("required", core.WorkbenchViewBool(field, "inputRequired")),
		gosx.Attr("disabled", core.WorkbenchViewBool(field, "inputDisabled")),
		gosx.Attr("checked", core.WorkbenchViewBool(field, "checked")),
		gosx.Attr("data-editor-source", core.WorkbenchViewString(field, "inputEditorSource")),
		gosx.Attr("data-editor-frame-target", core.WorkbenchViewString(field, "inputFrameTarget")),
		gosx.Attr("data-editor-frame-attr-target", core.WorkbenchViewString(field, "inputAttrTarget")),
		gosx.Attr("data-editor-frame-attr", core.WorkbenchViewString(field, "inputAttrName")),
		gosx.Attr("data-editor-frame-attr-prefix", core.WorkbenchViewString(field, "inputAttrPrefix")),
		gosx.Attr("data-editor-frame-attr-suffix", core.WorkbenchViewString(field, "inputAttrSuffix")),
		gosx.Attr("hidden", core.WorkbenchViewBool(field, "isArea") || core.WorkbenchViewBool(field, "isSelect") || core.WorkbenchViewBool(field, "isCard")),
	}
	// Checkboxes carry "on"/"true" in this same value slot; a hover tooltip
	// repeating that is noise, not a truncation fix, so only text-like
	// inputs get one.
	if value != "" && core.WorkbenchViewString(field, "inputType") != "checkbox" {
		attrs = append(attrs, gosx.Attr("title", value))
	}
	return gosx.El("input", gosx.Attrs(attrs...))
}

func renderInspectorFieldControl(field map[string]any) gosx.Node {
	controlAttrs := core.WorkbenchViewMap(field, "controlAttrs")
	switch {
	case core.WorkbenchViewBool(field, "isArea"):
		value := inspectorFieldRawString(field, "value")
		return gosx.El("textarea", gosx.Attrs(inspectorFieldMapAttrsWithTitle(controlAttrs, value)...), gosx.Text(value))
	case core.WorkbenchViewBool(field, "isSelect"):
		optionNodes := []gosx.Node{}
		for _, option := range siteMapMapList(field, "options") {
			optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(inspectorFieldMapAttrs(core.WorkbenchViewMap(option, "attrs"))...), gosx.Text(core.WorkbenchViewString(option, "label"))))
		}
		return gosx.El("select", gosx.Attrs(inspectorFieldMapAttrs(controlAttrs)...), gosx.Fragment(optionNodes...))
	case core.WorkbenchViewBool(field, "isCheckbox"), core.WorkbenchViewBool(field, "isInput"):
		value, _ := controlAttrs["value"].(string)
		return gosx.El("input", gosx.Attrs(inspectorFieldMapAttrsWithTitle(controlAttrs, value)...))
	default:
		return gosx.Fragment()
	}
}

// inspectorFieldMapAttrsWithTitle builds the attrs list from values (see
// inspectorFieldMapAttrs) plus a "title" attribute mirroring value, unless
// values already carries its own title, the control's type is "checkbox"
// (whose value slot holds "on"/"true", not preview text worth a tooltip),
// or value is empty. handoff-4 (text truncation sweep): the ADVANCED/
// generic field controls this feeds (BlockFieldInspectorFields,
// WorkbenchFieldInspectorFields) render at fixed rail widths and clip long
// values the same way the Home inspector's own inputs did (see
// renderHomeInspectorInput/renderHomeInspectorTextarea's identical fix) —
// a title gives every one of them a native hover/focus fallback.
func inspectorFieldMapAttrsWithTitle(values map[string]any, value string) []any {
	attrs := inspectorFieldMapAttrs(values)
	if value == "" {
		return attrs
	}
	if _, hasTitle := values["title"]; hasTitle {
		return attrs
	}
	if typeValue, _ := values["type"].(string); typeValue == "checkbox" {
		return attrs
	}
	return append(attrs, gosx.Attr("title", value))
}

func renderInspectorFieldHelp(field map[string]any) gosx.Node {
	if !core.WorkbenchViewBool(field, "hasHelp") {
		return gosx.Fragment()
	}
	return gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(core.WorkbenchViewString(field, "help")))
}

func renderInspectorFieldActions(field map[string]any) []gosx.Node {
	actions := siteMapMapList(field, "actions")
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		attrs := gosx.Attrs(inspectorFieldMapAttrs(core.WorkbenchViewMap(action, "attrs"))...)
		if core.WorkbenchViewBool(action, "hasFormAction") {
			nodes = append(nodes, gosx.El("button", attrs, gosx.Text(core.WorkbenchViewString(action, "label"))))
			continue
		}
		nodes = append(nodes, gosx.El("a", attrs, gosx.Text(core.WorkbenchViewString(action, "label"))))
	}
	return nodes
}

func inspectorFieldMapAttrs(values map[string]any) []any {
	attrs := make([]any, 0, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := values[key]
		if value == nil {
			continue
		}
		attrs = append(attrs, gosx.Attr(key, value))
	}
	return attrs
}

func appendInspectorFieldAttrs(attrs []any, values map[string]any) []any {
	return append(attrs, inspectorFieldMapAttrs(values)...)
}

func inspectorFieldRawString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func inspectorFieldValue(values map[string]any, key string) any {
	if values == nil {
		return nil
	}
	return values[key]
}
