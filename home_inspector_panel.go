package studio

import "m31labs.dev/gosx"

type HomeInspectorPanelOptions struct {
	RootAttrs map[string]any
}

func RenderHomeInspectorPanel(view map[string]any, contentFields map[string]any, options HomeInspectorPanelOptions) gosx.Node {
	return gosx.El("section", gosx.Attrs(homeInspectorPanelRootAttrs(view, options.RootAttrs)...),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-home-inspector__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
				gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
			),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("class", "studio-home-inspector__state"),
				gosx.Attr("data-studio-inspector-state", workbenchMapString(view, "cleanLabel")),
			), gosx.Text(workbenchMapString(view, "cleanLabel"))),
			gosx.El("label", gosx.Attrs(
				gosx.Attr("class", "studio-home-inspector__close"),
				gosx.Attr("for", "studioSiteMapInspectorFocus"),
				gosx.Attr("role", "button"),
				gosx.Attr("tabindex", "0"),
			), gosx.Text("Close")),
		),
		renderHomeInspectorTarget(view),
		renderHomeInspectorGroups(view),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-home-inspector__fields"),
			gosx.Attr("data-studio-home-inspector-fields", "true"),
		), renderHomeInspectorFieldPanel(contentFields)),
	)
}

func homeInspectorPanelRootAttrs(view map[string]any, rootAttrs map[string]any) []any {
	attrs := []any{
		gosx.Attr("id", "studioHomeInspector"),
		gosx.Attr("class", "editor-panel editor-panel--home-inspector studio-home-inspector"),
		gosx.Attr("data-studio-home-inspector-panel", "true"),
		gosx.Attr("data-studio-mode-panel", "home"),
		gosx.Attr("data-studio-panel", "inspector"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-inspector-group-active", workbenchMapString(view, "defaultGroupKey")),
		gosx.Attr("data-studio-inspector-dirty", "false"),
		gosx.Attr("data-studio-inspector-clean-label", workbenchMapString(view, "cleanLabel")),
		gosx.Attr("data-studio-inspector-dirty-label", workbenchMapString(view, "dirtyLabel")),
		gosx.Attr("data-gosx-studio-home-inspector-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

func renderHomeInspectorTarget(view map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-home-inspector__target"),
		gosx.Attr("aria-label", workbenchMapString(view, "targetLabel")),
	),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "targetKicker"))),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(workbenchMapString(view, "selectionLabel"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(view, "targetSummary"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-home-inspector__target-status")),
			gosx.El("output", nil, gosx.Text(workbenchMapString(view, "previewLabel"))),
			gosx.El("output", nil, gosx.Text(workbenchMapString(view, "draftLabel"))),
		),
	)
}

func renderHomeInspectorGroups(view map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-home-inspector__groups"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", workbenchMapString(view, "groupLabel")),
	),
		renderHomeInspectorGroupButton(view, "copy", "copyLabel"),
		renderHomeInspectorGroupButton(view, "media", "mediaLabel"),
		renderHomeInspectorGroupButton(view, "sources", "sourcesLabel"),
		renderHomeInspectorGroupButton(view, "all", "allLabel"),
	)
}

func renderHomeInspectorGroupButton(view map[string]any, key, labelKey string) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("type", "button"),
		gosx.Attr("class", "studio-home-inspector__group"),
		gosx.Attr("data-studio-inspector-group", key),
		gosx.Attr("aria-pressed", BoolAttr(workbenchMapString(view, "defaultGroupKey") == key)),
	), gosx.El("strong", nil, gosx.Text(workbenchMapString(view, labelKey))))
}

func renderHomeInspectorFieldPanel(contentFields map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("hidden", workbenchMapBool(contentFields, "hasFields")),
		), gosx.Text(workbenchMapString(contentFields, "empty"))),
	}
	if fieldList, ok := contentFields["fieldList"].(gosx.Node); ok && workbenchMapBool(contentFields, "hasFields") {
		children = append(children, fieldList)
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(contentFields, "class")),
		gosx.Attr("data-studio-home-fields-panel", "true"),
		gosx.Attr("data-panel-key", workbenchMapString(contentFields, "key")),
		gosx.Attr("data-studio-panel", workbenchMapString(contentFields, "key")),
		gosx.Attr("data-studio-mode-panel", workbenchMapString(contentFields, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
	), gosx.Fragment(children...))
}
