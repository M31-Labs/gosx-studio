package studio

import "m31labs.dev/gosx"

type RevisionHistoryPanelOptions struct {
	RootAttrs map[string]any
}

func RenderRevisionHistoryPanel(view map[string]any, options RevisionHistoryPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-panel-key", workbenchMapString(view, "panelKey")),
		gosx.Attr("data-studio-revision-history", "true"),
		gosx.Attr("data-gosx-studio-revision-history-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	hasItems := workbenchMapBool(view, "hasItems")
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(view, "headerClass"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("hidden", hasItems),
		), gosx.Text(workbenchMapString(view, "empty"))),
		gosx.El("ul", gosx.Attrs(
			gosx.Attr("class", "field-list field-list--stacked"),
			gosx.Attr("hidden", !hasItems),
		),
			gosx.Fragment(renderRevisionHistoryItems(workbenchViewMapList(view, "items"))...),
		),
	)
}

func renderRevisionHistoryItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		hasSummary := workbenchMapBool(item, "hasSummary")
		hasDiff := workbenchMapBool(item, "hasDiff")
		nodes = append(nodes, gosx.El("li", gosx.Attrs(gosx.Attr("data-studio-revision", workbenchMapString(item, "key"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(item, "title"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(item, "actionLabel"))),
			gosx.El("time", gosx.Attrs(
				gosx.Attr("datetime", workbenchMapString(item, "createdMachine")),
				gosx.Attr("data-viewer-time", "datetime"),
			), gosx.Text(workbenchMapString(item, "createdLabel"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("hidden", !hasSummary)), gosx.Text(workbenchMapString(item, "summary"))),
			gosx.El("p", gosx.Attrs(
				gosx.Attr("class", "revision-diff-summary"),
				gosx.Attr("hidden", !hasDiff),
			), gosx.Text(workbenchMapString(item, "changeSummary"))),
			gosx.El("ul", gosx.Attrs(
				gosx.Attr("class", "revision-diff-list"),
				gosx.Attr("hidden", !hasDiff),
			),
				gosx.Fragment(renderRevisionHistoryChangeItems(workbenchViewMapList(item, "changeItems"))...),
			),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", "button button--secondary"),
				gosx.Attr("type", "submit"),
				gosx.Attr("form", workbenchMapString(item, "formID")),
				gosx.Attr("formaction", workbenchMapString(item, "restoreAction")),
				gosx.Attr("formmethod", "post"),
				gosx.Attr("name", workbenchMapString(item, "revisionInputName")),
				gosx.Attr("value", workbenchMapString(item, "revisionInputValue")),
				gosx.Attr("data-admin-confirm", workbenchMapString(item, "confirm")),
				gosx.Attr("data-studio-submit-action", "restoreRevision"),
				gosx.Attr("data-studio-field-action-formaction", workbenchMapString(item, "restoreAction")),
			), gosx.Text(workbenchMapString(item, "buttonLabel"))),
		))
	}
	return nodes
}

func renderRevisionHistoryChangeItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("li", nil,
			gosx.El("span", nil, gosx.Text(workbenchMapString(item, "kindLabel"))),
			gosx.El("code", nil, gosx.Text(workbenchMapString(item, "path"))),
		))
	}
	return nodes
}
