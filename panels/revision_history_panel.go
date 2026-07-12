package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type RevisionHistoryPanelOptions struct {
	RootAttrs       map[string]any
	StandaloneForms bool
}

func RenderRevisionHistoryPanel(view map[string]any, options RevisionHistoryPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(view, "panelKey")),
		gosx.Attr("data-studio-revision-history", "true"),
		gosx.Attr("data-gosx-studio-revision-history-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	hasItems := core.WorkbenchViewBool(view, "hasItems")
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(view, "headerClass"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("hidden", hasItems),
		), gosx.Text(core.WorkbenchViewString(view, "empty"))),
		gosx.El("ul", gosx.Attrs(
			gosx.Attr("class", "field-list field-list--stacked"),
			gosx.Attr("hidden", !hasItems),
		),
			gosx.Fragment(renderRevisionHistoryItems(view, core.WorkbenchViewMapList(view, "items"), options)...),
		),
	)
}

func renderRevisionHistoryItems(view map[string]any, items []map[string]any, options RevisionHistoryPanelOptions) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		hasSummary := core.WorkbenchViewBool(item, "hasSummary")
		hasDiff := core.WorkbenchViewBool(item, "hasDiff")
		children := []gosx.Node{
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(item, "title"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "actionLabel"))),
			gosx.El("time", gosx.Attrs(
				gosx.Attr("datetime", core.WorkbenchViewString(item, "createdMachine")),
				gosx.Attr("data-viewer-time", "datetime"),
			), gosx.Text(core.WorkbenchViewString(item, "createdLabel"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("hidden", !hasSummary)), gosx.Text(core.WorkbenchViewString(item, "summary"))),
			gosx.El("p", gosx.Attrs(
				gosx.Attr("class", "revision-diff-summary"),
				gosx.Attr("hidden", !hasDiff),
			), gosx.Text(core.WorkbenchViewString(item, "changeSummary"))),
			gosx.El("ul", gosx.Attrs(
				gosx.Attr("class", "revision-diff-list"),
				gosx.Attr("hidden", !hasDiff),
			),
				gosx.Fragment(renderRevisionHistoryChangeItems(core.WorkbenchViewMapList(item, "changeItems"))...),
			),
		}
		if options.StandaloneForms {
			children = append(children, renderRevisionHistoryStandaloneForm(view, item))
		} else {
			children = append(children, renderRevisionHistoryWorkbenchButton(item))
		}
		nodes = append(nodes, gosx.El("li", gosx.Attrs(gosx.Attr("data-studio-revision", core.WorkbenchViewString(item, "key"))), gosx.Fragment(children...)))
	}
	return nodes
}

func renderRevisionHistoryWorkbenchButton(item map[string]any) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "button button--secondary"),
		gosx.Attr("type", "submit"),
		gosx.Attr("form", core.WorkbenchViewString(item, "formID")),
		gosx.Attr("formaction", core.WorkbenchViewString(item, "restoreAction")),
		gosx.Attr("formmethod", "post"),
		// The workbench-button variant submits the SHARED websiteEditorForm
		// (every panel's fields, including ones currently hidden by mode-gating
		// CSS, are "listed" constraint-validation candidates of that one form).
		// Native HTML5 validation runs BEFORE the browser dispatches the form's
		// "submit" event (which is what studio's state_runtime.js listens on to
		// intercept and fetch this action) — if ANY control anywhere in the form
		// is invalid (e.g. a type=url field carrying a non-URL seeded value in a
		// panel that isn't the active mode), the browser silently cancels the
		// submission with no submit event at all, regardless of which button was
		// clicked. formnovalidate exempts THIS submission from that shared-form
		// validation surface, mirroring every other workbench-form action button
		// (Publish/Discard/Schedule in publish_panel.go) — restore must behave
		// the same way or it silently no-ops whenever an unrelated hidden panel
		// happens to carry an invalid value.
		gosx.Attr("formnovalidate", "formnovalidate"),
		gosx.Attr("name", core.WorkbenchViewString(item, "revisionInputName")),
		gosx.Attr("value", core.WorkbenchViewString(item, "revisionInputValue")),
		gosx.Attr("data-admin-confirm", core.WorkbenchViewString(item, "confirm")),
		gosx.Attr("data-studio-submit-action", "restoreRevision"),
		gosx.Attr("data-studio-field-action-formaction", core.WorkbenchViewString(item, "restoreAction")),
	), gosx.Text(core.WorkbenchViewString(item, "buttonLabel")))
}

func renderRevisionHistoryStandaloneForm(view, item map[string]any) gosx.Node {
	hiddenInputs := revisionHistoryInputViews(view, "hiddenInputs")
	hiddenInputs = append(hiddenInputs, revisionHistoryInputViews(item, "hiddenInputs")...)
	token := core.WorkbenchViewString(item, "csrfToken")
	if token == "" {
		token = core.WorkbenchViewString(view, "csrfToken")
	}
	children := renderRevisionHistoryHiddenInputs(hiddenInputs, token)
	children = append(children, gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "button button--secondary"),
		gosx.Attr("type", "submit"),
		gosx.Attr("data-admin-confirm", core.WorkbenchViewString(item, "confirm")),
	), gosx.Text(core.WorkbenchViewString(item, "buttonLabel"))))
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "inline-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", core.WorkbenchViewString(item, "restoreAction")),
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
				"name":  core.WorkbenchViewString(item, "name"),
				"value": core.WorkbenchViewString(item, "value"),
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
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "kindLabel"))),
			gosx.El("code", nil, gosx.Text(core.WorkbenchViewString(item, "path"))),
		))
	}
	return nodes
}
