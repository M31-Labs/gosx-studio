package studio

import "m31labs.dev/gosx"

type RevisionHistoryPanelOptions struct {
	RootAttrs       map[string]any
	StandaloneForms bool
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
			gosx.Fragment(renderRevisionHistoryItems(view, workbenchViewMapList(view, "items"), options)...),
		),
	)
}

func renderRevisionHistoryItems(view map[string]any, items []map[string]any, options RevisionHistoryPanelOptions) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		hasSummary := workbenchMapBool(item, "hasSummary")
		hasDiff := workbenchMapBool(item, "hasDiff")
		children := []gosx.Node{
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
		}
		if options.StandaloneForms {
			children = append(children, renderRevisionHistoryStandaloneForm(view, item))
		} else {
			children = append(children, renderRevisionHistoryWorkbenchButton(item))
		}
		nodes = append(nodes, gosx.El("li", gosx.Attrs(gosx.Attr("data-studio-revision", workbenchMapString(item, "key"))), gosx.Fragment(children...)))
	}
	return nodes
}

func renderRevisionHistoryWorkbenchButton(item map[string]any) gosx.Node {
	return gosx.El("button", gosx.Attrs(
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
	), gosx.Text(workbenchMapString(item, "buttonLabel")))
}

func renderRevisionHistoryStandaloneForm(view, item map[string]any) gosx.Node {
	hiddenInputs := revisionHistoryInputViews(view, "hiddenInputs")
	hiddenInputs = append(hiddenInputs, revisionHistoryInputViews(item, "hiddenInputs")...)
	token := workbenchMapString(item, "csrfToken")
	if token == "" {
		token = workbenchMapString(view, "csrfToken")
	}
	children := renderRevisionHistoryHiddenInputs(hiddenInputs, token)
	children = append(children, gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "button button--secondary"),
		gosx.Attr("type", "submit"),
		gosx.Attr("data-admin-confirm", workbenchMapString(item, "confirm")),
	), gosx.Text(workbenchMapString(item, "buttonLabel"))))
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "inline-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", workbenchMapString(item, "restoreAction")),
	), gosx.Fragment(children...))
}

func renderRevisionHistoryHiddenInputs(inputs []map[string]string, csrfToken string) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(inputs)+1)
	if csrfToken != "" && !revisionHistoryHasHiddenInput(inputs, "csrf_token") {
		nodes = append(nodes, renderRevisionHistoryHiddenInput("csrf_token", csrfToken))
	}
	for _, input := range inputs {
		name := input["name"]
		if name == "" {
			continue
		}
		nodes = append(nodes, renderRevisionHistoryHiddenInput(name, input["value"]))
	}
	return nodes
}

func renderRevisionHistoryHiddenInput(name, value string) gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("type", "hidden"),
		gosx.Attr("name", name),
		gosx.Attr("value", value),
	))
}

func revisionHistoryHasHiddenInput(inputs []map[string]string, name string) bool {
	for _, input := range inputs {
		if input["name"] == name {
			return true
		}
	}
	return false
}

func revisionHistoryInputViews(values map[string]any, key string) []map[string]string {
	switch typed := values[key].(type) {
	case []map[string]string:
		return typed
	case []map[string]any:
		out := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, map[string]string{
				"name":  workbenchMapString(item, "name"),
				"value": workbenchMapString(item, "value"),
			})
		}
		return out
	default:
		return nil
	}
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
