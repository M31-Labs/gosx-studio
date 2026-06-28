package studio

import "m31labs.dev/gosx"

type NavigationPanelOptions struct {
	RootAttrs map[string]any
}

func RenderNavigationPanel(view map[string]any, options NavigationPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-panel-key", workbenchMapString(view, "panelKey")),
		gosx.Attr("data-studio-navigation-panel", "true"),
		gosx.Attr("data-gosx-studio-navigation-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	hasItems := workbenchMapBool(view, "hasItems")
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(view, "headerClass"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(workbenchMapString(view, "help"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("hidden", hasItems),
		), gosx.Text(workbenchMapString(view, "empty"))),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "inline-form studio-navigation-panel__form"),
			gosx.Attr("hidden", !hasItems),
		),
			gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list field-list--stacked")),
				gosx.Fragment(renderNavigationPanelRows(workbenchViewMapList(view, "items"))...),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "button button--primary"),
					gosx.Attr("type", "submit"),
					gosx.Attr("form", workbenchMapString(view, "formID")),
					gosx.Attr("formaction", workbenchMapString(view, "action")),
					gosx.Attr("formmethod", "post"),
					gosx.Attr("data-studio-submit-action", "navigation"),
					gosx.Attr("data-studio-field-action-formaction", workbenchMapString(view, "action")),
				), gosx.Text(workbenchMapString(view, "buttonLabel"))),
			),
		),
	)
}

func renderNavigationPanelRows(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("li", gosx.Attrs(
			gosx.Attr("class", "studio-navigation-panel__row"),
			gosx.Attr("data-studio-navigation-row", workbenchMapString(item, "key")),
		),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__field")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(item, "labelID"))), gosx.Text(workbenchMapString(item, "labelFieldText"))),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("id", workbenchMapString(item, "labelID")),
					gosx.Attr("name", workbenchMapString(item, "labelName")),
					gosx.Attr("type", "text"),
					gosx.Attr("value", workbenchMapString(item, "label")),
				)),
				gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(workbenchMapString(item, "linkLabel"))),
			),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "hidden"),
				gosx.Attr("name", workbenchMapString(item, "hrefName")),
				gosx.Attr("value", workbenchMapString(item, "href")),
			)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__controls")),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__toggle")),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("name", workbenchMapString(item, "enabledName")),
						gosx.Attr("type", "checkbox"),
						gosx.Attr("value", workbenchMapString(item, "enabledValue")),
						gosx.Attr("checked", workbenchMapBool(item, "enabled")),
					)),
					gosx.El("span", nil, gosx.Text(workbenchMapString(item, "showLabel"))),
				),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__order")),
					gosx.El("span", nil, gosx.Text(workbenchMapString(item, "positionLabel"))),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("name", workbenchMapString(item, "orderName")),
						gosx.Attr("type", "number"),
						gosx.Attr("min", "1"),
						gosx.Attr("value", workbenchMapString(item, "order")),
					)),
				),
			),
		))
	}
	return nodes
}
