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
	if workbenchMapBool(field, "isCard") {
		children = append(children,
			gosx.El("label", nil, gosx.Text(workbenchMapString(field, "label"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-source-card")),
				gosx.El("strong", nil, gosx.Text(workbenchMapString(field, "cardTitle"))),
				renderInspectorFieldHelp(field),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(renderInspectorFieldActions(field)...)),
			),
		)
		return gosx.El("div", gosx.Attrs(inspectorFieldMapAttrs(workbenchViewMap(field, "rowAttrs"))...), gosx.Fragment(children...))
	}

	children = append(children, gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(field, "id"))), gosx.Text(workbenchMapString(field, "label"))))
	children = append(children, renderInspectorFieldControl(field))
	children = append(children, renderInspectorFieldHelp(field))

	return gosx.El("div", gosx.Attrs(inspectorFieldMapAttrs(workbenchViewMap(field, "rowAttrs"))...), gosx.Fragment(children...))
}

func renderHomeInspectorFieldRow(field map[string]any, options InspectorFieldRowOptions) gosx.Node {
	return gosx.El("div", gosx.Attrs(inspectorFieldMapAttrs(workbenchViewMap(field, "rowAttrs"))...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-home-field__head")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(field, "id"))), gosx.Text(workbenchMapString(field, "label"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(field, "kindLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-home-field__meta")),
			gosx.El("output", nil, gosx.Text(workbenchMapString(field, "validationLabel"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(field, "reactionLabel"))),
			gosx.El("small", gosx.Attrs(gosx.Attr("class", "studio-home-field__invalid")), gosx.Text(options.InvalidLabel)),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-source-card"),
			gosx.Attr("hidden", !workbenchMapBool(field, "isCard")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(field, "cardTitle"))),
			gosx.El("small", gosx.Attrs(
				gosx.Attr("class", "field-help"),
				gosx.Attr("hidden", !workbenchMapBool(field, "hasHelp")),
			), gosx.Text(workbenchMapString(field, "help"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(renderHomeInspectorFieldActions(field)...)),
		),
		renderHomeInspectorTextarea(field),
		renderHomeInspectorSelect(field),
		renderHomeInspectorInput(field),
		gosx.El("small", gosx.Attrs(
			gosx.Attr("class", "field-help"),
			gosx.Attr("hidden", workbenchMapBool(field, "isCard") || !workbenchMapBool(field, "hasHelp")),
		), gosx.Text(workbenchMapString(field, "help"))),
	)
}

func renderHomeInspectorFieldActions(field map[string]any) []gosx.Node {
	actions := siteMapMapList(field, "actions")
	nodes := make([]gosx.Node, 0, len(actions)*2)
	for _, action := range actions {
		hasFormAction := workbenchMapBool(action, "hasFormAction")
		nodes = append(nodes,
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", workbenchMapString(action, "class")),
				gosx.Attr("type", "submit"),
				gosx.Attr("formaction", workbenchMapString(action, "formAction")),
				gosx.Attr("formmethod", workbenchMapString(action, "formMethod")),
				gosx.Attr("name", workbenchMapString(action, "name")),
				gosx.Attr("value", inspectorFieldRawString(action, "value")),
				gosx.Attr("data-admin-confirm", workbenchMapString(action, "confirm")),
				gosx.Attr("data-studio-submit-action", workbenchMapString(action, "submitAction")),
				gosx.Attr("data-studio-field-action-formaction", workbenchMapString(action, "formAction")),
				gosx.Attr("hidden", !hasFormAction),
			), gosx.Text(workbenchMapString(action, "label"))),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", workbenchMapString(action, "class")),
				gosx.Attr("href", workbenchMapString(action, "href")),
				gosx.Attr("data-gosx-link", "true"),
				gosx.Attr("hidden", hasFormAction),
			), gosx.Text(workbenchMapString(action, "label"))),
		)
	}
	return nodes
}

func renderHomeInspectorTextarea(field map[string]any) gosx.Node {
	return gosx.El("textarea", gosx.Attrs(
		gosx.Attr("id", workbenchMapString(field, "id")),
		gosx.Attr("name", workbenchMapString(field, "areaName")),
		gosx.Attr("rows", inspectorFieldValue(field, "rows")),
		gosx.Attr("placeholder", workbenchMapString(field, "placeholder")),
		gosx.Attr("required", workbenchMapBool(field, "areaRequired")),
		gosx.Attr("disabled", workbenchMapBool(field, "areaDisabled")),
		gosx.Attr("data-editor-source", workbenchMapString(field, "areaEditorSource")),
		gosx.Attr("data-editor-frame-target", workbenchMapString(field, "areaFrameTarget")),
		gosx.Attr("data-editor-frame-attr-target", workbenchMapString(field, "areaAttrTarget")),
		gosx.Attr("data-editor-frame-attr", workbenchMapString(field, "areaAttrName")),
		gosx.Attr("data-editor-frame-attr-prefix", workbenchMapString(field, "areaAttrPrefix")),
		gosx.Attr("data-editor-frame-attr-suffix", workbenchMapString(field, "areaAttrSuffix")),
		gosx.Attr("hidden", !workbenchMapBool(field, "isArea")),
	), gosx.Text(inspectorFieldRawString(field, "areaValue")))
}

func renderHomeInspectorSelect(field map[string]any) gosx.Node {
	optionNodes := []gosx.Node{}
	for _, option := range siteMapMapList(field, "options") {
		optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", workbenchMapString(option, "value")),
			gosx.Attr("selected", workbenchMapBool(option, "selected")),
		), gosx.Text(workbenchMapString(option, "label"))))
	}
	return gosx.El("select", gosx.Attrs(
		gosx.Attr("id", workbenchMapString(field, "id")),
		gosx.Attr("name", workbenchMapString(field, "selectName")),
		gosx.Attr("required", workbenchMapBool(field, "selectRequired")),
		gosx.Attr("disabled", workbenchMapBool(field, "selectDisabled")),
		gosx.Attr("data-editor-source", workbenchMapString(field, "selectEditorSource")),
		gosx.Attr("data-editor-frame-target", workbenchMapString(field, "selectFrameTarget")),
		gosx.Attr("data-editor-frame-attr-target", workbenchMapString(field, "selectAttrTarget")),
		gosx.Attr("data-editor-frame-attr", workbenchMapString(field, "selectAttrName")),
		gosx.Attr("data-editor-frame-attr-prefix", workbenchMapString(field, "selectAttrPrefix")),
		gosx.Attr("data-editor-frame-attr-suffix", workbenchMapString(field, "selectAttrSuffix")),
		gosx.Attr("hidden", !workbenchMapBool(field, "isSelect")),
	), gosx.Fragment(optionNodes...))
}

func renderHomeInspectorInput(field map[string]any) gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("id", workbenchMapString(field, "id")),
		gosx.Attr("name", workbenchMapString(field, "inputName")),
		gosx.Attr("type", workbenchMapString(field, "inputType")),
		gosx.Attr("value", inspectorFieldRawString(field, "inputValue")),
		gosx.Attr("placeholder", workbenchMapString(field, "placeholder")),
		gosx.Attr("required", workbenchMapBool(field, "inputRequired")),
		gosx.Attr("disabled", workbenchMapBool(field, "inputDisabled")),
		gosx.Attr("checked", workbenchMapBool(field, "checked")),
		gosx.Attr("data-editor-source", workbenchMapString(field, "inputEditorSource")),
		gosx.Attr("data-editor-frame-target", workbenchMapString(field, "inputFrameTarget")),
		gosx.Attr("data-editor-frame-attr-target", workbenchMapString(field, "inputAttrTarget")),
		gosx.Attr("data-editor-frame-attr", workbenchMapString(field, "inputAttrName")),
		gosx.Attr("data-editor-frame-attr-prefix", workbenchMapString(field, "inputAttrPrefix")),
		gosx.Attr("data-editor-frame-attr-suffix", workbenchMapString(field, "inputAttrSuffix")),
		gosx.Attr("hidden", workbenchMapBool(field, "isArea") || workbenchMapBool(field, "isSelect") || workbenchMapBool(field, "isCard")),
	))
}

func renderInspectorFieldControl(field map[string]any) gosx.Node {
	attrs := gosx.Attrs(inspectorFieldMapAttrs(workbenchViewMap(field, "controlAttrs"))...)
	switch {
	case workbenchMapBool(field, "isArea"):
		return gosx.El("textarea", attrs, gosx.Text(inspectorFieldRawString(field, "value")))
	case workbenchMapBool(field, "isSelect"):
		optionNodes := []gosx.Node{}
		for _, option := range siteMapMapList(field, "options") {
			optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(inspectorFieldMapAttrs(workbenchViewMap(option, "attrs"))...), gosx.Text(workbenchMapString(option, "label"))))
		}
		return gosx.El("select", attrs, gosx.Fragment(optionNodes...))
	case workbenchMapBool(field, "isCheckbox"), workbenchMapBool(field, "isInput"):
		return gosx.El("input", attrs)
	default:
		return gosx.Fragment()
	}
}

func renderInspectorFieldHelp(field map[string]any) gosx.Node {
	if !workbenchMapBool(field, "hasHelp") {
		return gosx.Fragment()
	}
	return gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(workbenchMapString(field, "help")))
}

func renderInspectorFieldActions(field map[string]any) []gosx.Node {
	actions := siteMapMapList(field, "actions")
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		attrs := gosx.Attrs(inspectorFieldMapAttrs(workbenchViewMap(action, "attrs"))...)
		if workbenchMapBool(action, "hasFormAction") {
			nodes = append(nodes, gosx.El("button", attrs, gosx.Text(workbenchMapString(action, "label"))))
			continue
		}
		nodes = append(nodes, gosx.El("a", attrs, gosx.Text(workbenchMapString(action, "label"))))
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
