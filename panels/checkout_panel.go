package panels

import "m31labs.dev/gosx"

type CheckoutPanelOptions struct {
	RootAttrs map[string]any
}

func RenderCheckoutPanel(view map[string]any, options CheckoutPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-panel-key", workbenchMapString(view, "panelKey")),
		gosx.Attr("data-studio-checkout-panel", "true"),
		gosx.Attr("data-gosx-studio-checkout-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	hasProducts := workbenchMapBool(view, "hasProducts")
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(view, "headerClass"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(workbenchMapString(view, "help"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "inline-form studio-checkout-panel__form")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-checkout-panel__field")),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-checkout-panel__toggle")),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("name", workbenchMapString(view, "enabledName")),
						gosx.Attr("type", "checkbox"),
						gosx.Attr("value", workbenchMapString(view, "enabledValue")),
						gosx.Attr("checked", workbenchMapBool(view, "enabled")),
					)),
					gosx.El("span", nil, gosx.Text(workbenchMapString(view, "enabledLabel"))),
				),
			),
			gosx.El("p", gosx.Attrs(
				gosx.Attr("class", "empty"),
				gosx.Attr("hidden", hasProducts),
			), gosx.Text(workbenchMapString(view, "empty"))),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "studio-checkout-panel__field"),
				gosx.Attr("hidden", !hasProducts),
			),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(view, "productID"))), gosx.Text(workbenchMapString(view, "productLabel"))),
				gosx.El("select", gosx.Attrs(
					gosx.Attr("id", workbenchMapString(view, "productID")),
					gosx.Attr("name", workbenchMapString(view, "productName")),
					gosx.Attr("data-studio-checkout-product", "true"),
				),
					gosx.Fragment(renderCheckoutPanelOptions(workbenchViewMapList(view, "products"))...),
				),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-checkout-panel__field")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(view, "labelID"))), gosx.Text(workbenchMapString(view, "labelFieldText"))),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("id", workbenchMapString(view, "labelID")),
					gosx.Attr("name", workbenchMapString(view, "labelName")),
					gosx.Attr("type", "text"),
					gosx.Attr("value", workbenchMapString(view, "label")),
				)),
				gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(workbenchMapString(view, "labelHelp"))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "button button--primary"),
					gosx.Attr("type", "submit"),
					gosx.Attr("form", workbenchMapString(view, "formID")),
					gosx.Attr("formaction", workbenchMapString(view, "action")),
					gosx.Attr("formmethod", "post"),
					gosx.Attr("data-studio-submit-action", "checkout"),
					gosx.Attr("data-studio-field-action-formaction", workbenchMapString(view, "action")),
				), gosx.Text(workbenchMapString(view, "buttonLabel"))),
			),
		),
	)
}

func renderCheckoutPanelOptions(options []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", workbenchMapString(option, "value")),
			gosx.Attr("selected", workbenchMapBool(option, "selected")),
		), gosx.Text(workbenchMapString(option, "label"))))
	}
	return nodes
}
