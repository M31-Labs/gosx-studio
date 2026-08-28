package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type NavigationPanelOptions struct {
	RootAttrs map[string]any
}

func RenderNavigationPanel(view map[string]any, options NavigationPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(view, "panelKey")),
		gosx.Attr("data-studio-navigation-panel", "true"),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(view, "mode")),
		gosx.Attr("data-gosx-studio-navigation-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	hasItems := core.WorkbenchViewBool(view, "hasItems")
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(view, "headerClass"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(core.WorkbenchViewString(view, "help"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("hidden", hasItems),
		), gosx.Text(core.WorkbenchViewString(view, "empty"))),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "inline-form studio-navigation-panel__form"),
			gosx.Attr("hidden", !hasItems),
		),
			gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list field-list--stacked")),
				gosx.Fragment(renderNavigationPanelRows(core.WorkbenchViewMapList(view, "items"))...),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "button button--primary"),
					gosx.Attr("type", "submit"),
					gosx.Attr("form", core.WorkbenchViewString(view, "formID")),
					gosx.Attr("formaction", core.WorkbenchViewString(view, "action")),
					gosx.Attr("formmethod", "post"),
					// This button submits the SHARED websiteEditorForm (every
					// panel's fields, including ones currently hidden by
					// mode-gating CSS, are "listed" constraint-validation
					// candidates of that one form). Native HTML5 validation
					// runs BEFORE the browser dispatches the form's "submit"
					// event; if ANY control anywhere in the form is invalid,
					// the browser silently cancels the submission with no
					// submit event at all, regardless of which button was
					// clicked. formnovalidate exempts THIS submission from
					// that shared-form validation surface, mirroring every
					// other workbench-form action button (Publish/Discard/
					// Schedule in publish_panel.go, Restore in
					// revision_history_panel.go) — Save navigation must
					// behave the same way or it silently no-ops whenever an
					// unrelated hidden panel happens to carry an invalid
					// value.
					gosx.Attr("formnovalidate", "formnovalidate"),
					gosx.Attr("data-studio-submit-action", "navigation"),
					gosx.Attr("data-studio-field-action-formaction", core.WorkbenchViewString(view, "action")),
				), gosx.Text(core.WorkbenchViewString(view, "buttonLabel"))),
			),
		),
	)
}

func renderNavigationPanelRows(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("li", gosx.Attrs(
			gosx.Attr("class", "studio-navigation-panel__row"),
			gosx.Attr("data-studio-navigation-row", core.WorkbenchViewString(item, "key")),
		),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__field")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", core.WorkbenchViewString(item, "labelID"))), gosx.Text(core.WorkbenchViewString(item, "labelFieldText"))),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("id", core.WorkbenchViewString(item, "labelID")),
					gosx.Attr("name", core.WorkbenchViewString(item, "labelName")),
					gosx.Attr("type", "text"),
					gosx.Attr("value", core.WorkbenchViewString(item, "label")),
				)),
				gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(core.WorkbenchViewString(item, "linkLabel"))),
			),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "hidden"),
				gosx.Attr("name", core.WorkbenchViewString(item, "hrefName")),
				gosx.Attr("value", core.WorkbenchViewString(item, "href")),
			)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__controls")),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__toggle")),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("name", core.WorkbenchViewString(item, "enabledName")),
						gosx.Attr("type", "checkbox"),
						gosx.Attr("value", core.WorkbenchViewString(item, "enabledValue")),
						gosx.Attr("checked", core.WorkbenchViewBool(item, "enabled")),
					)),
					gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "showLabel"))),
				),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-navigation-panel__order")),
					gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "positionLabel"))),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("name", core.WorkbenchViewString(item, "orderName")),
						gosx.Attr("type", "number"),
						gosx.Attr("min", "1"),
						gosx.Attr("value", core.WorkbenchViewString(item, "order")),
					)),
				),
			),
		))
	}
	return nodes
}
