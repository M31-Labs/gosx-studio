package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type CheckoutPanelOptions struct {
	RootAttrs map[string]any
}

func RenderCheckoutPanel(view map[string]any, options CheckoutPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(view, "panelKey")),
		gosx.Attr("data-studio-checkout-panel", "true"),
		gosx.Attr("data-gosx-studio-checkout-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	hasProducts := core.WorkbenchViewBool(view, "hasProducts")
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(view, "headerClass"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(core.WorkbenchViewString(view, "help"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "inline-form studio-checkout-panel__form")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-checkout-panel__field")),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-checkout-panel__toggle")),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("name", core.WorkbenchViewString(view, "enabledName")),
						gosx.Attr("type", "checkbox"),
						gosx.Attr("value", core.WorkbenchViewString(view, "enabledValue")),
						gosx.Attr("checked", core.WorkbenchViewBool(view, "enabled")),
					)),
					gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(view, "enabledLabel"))),
				),
			),
			gosx.El("p", gosx.Attrs(
				gosx.Attr("class", "empty"),
				gosx.Attr("hidden", hasProducts),
			), gosx.Text(core.WorkbenchViewString(view, "empty"))),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "studio-checkout-panel__field"),
				gosx.Attr("hidden", !hasProducts),
			),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", core.WorkbenchViewString(view, "productID"))), gosx.Text(core.WorkbenchViewString(view, "productLabel"))),
				gosx.El("select", gosx.Attrs(
					gosx.Attr("id", core.WorkbenchViewString(view, "productID")),
					gosx.Attr("name", core.WorkbenchViewString(view, "productName")),
					gosx.Attr("data-studio-checkout-product", "true"),
				),
					gosx.Fragment(renderCheckoutPanelOptions(core.WorkbenchViewMapList(view, "products"))...),
				),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-checkout-panel__field")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", core.WorkbenchViewString(view, "labelID"))), gosx.Text(core.WorkbenchViewString(view, "labelFieldText"))),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("id", core.WorkbenchViewString(view, "labelID")),
					gosx.Attr("name", core.WorkbenchViewString(view, "labelName")),
					gosx.Attr("type", "text"),
					gosx.Attr("value", core.WorkbenchViewString(view, "label")),
				)),
				gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(core.WorkbenchViewString(view, "labelHelp"))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "button button--primary"),
					gosx.Attr("type", "submit"),
					gosx.Attr("form", core.WorkbenchViewString(view, "formID")),
					gosx.Attr("formaction", core.WorkbenchViewString(view, "action")),
					gosx.Attr("formmethod", "post"),
					gosx.Attr("data-studio-submit-action", "checkout"),
					gosx.Attr("data-studio-field-action-formaction", core.WorkbenchViewString(view, "action")),
				), gosx.Text(core.WorkbenchViewString(view, "buttonLabel"))),
			),
		),
	)
}

func renderCheckoutPanelOptions(options []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", core.WorkbenchViewString(option, "value")),
			gosx.Attr("selected", core.WorkbenchViewBool(option, "selected")),
		), gosx.Text(core.WorkbenchViewString(option, "label"))))
	}
	return nodes
}
