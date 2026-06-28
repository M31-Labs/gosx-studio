package studio

import (
	"sort"

	"m31labs.dev/gosx"
)

type InspectorFieldListOptions struct {
	Class      string
	Attrs      map[string]any
	RowOptions InspectorFieldRowOptions
}

type InspectorFieldRowOptions struct {
	Wrapper bool
}

func RenderInspectorFieldList(fields []map[string]any, options InspectorFieldListOptions) gosx.Node {
	children := make([]gosx.Node, 0, len(fields))
	for _, field := range fields {
		children = append(children, RenderInspectorFieldRow(field, options.RowOptions))
	}
	attrs := []any{
		gosx.Attr("class", FirstNonEmpty(options.Class, "studio-brand-panel__field-list")),
		gosx.Attr("data-gosx-studio-inspector-field-list-renderer", "gosx-studio"),
	}
	attrs = appendInspectorFieldAttrs(attrs, options.Attrs)
	return gosx.El("div", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func RenderInspectorFieldRow(field map[string]any, options InspectorFieldRowOptions) gosx.Node {
	row := renderInspectorFieldRow(field)
	if !options.Wrapper {
		return row
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("data-studio-advanced-field-row", "true"),
		gosx.Attr("data-gosx-studio-inspector-field-row-renderer", "gosx-studio"),
	), row)
}

func renderInspectorFieldRow(field map[string]any) gosx.Node {
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

func renderInspectorFieldControl(field map[string]any) gosx.Node {
	attrs := gosx.Attrs(inspectorFieldMapAttrs(workbenchViewMap(field, "controlAttrs"))...)
	switch {
	case workbenchMapBool(field, "isArea"):
		return gosx.El("textarea", attrs, gosx.Text(workbenchMapString(field, "value")))
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
