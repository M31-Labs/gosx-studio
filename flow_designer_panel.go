package studio

import "m31labs.dev/gosx"

type FlowDesignerPanelOptions struct {
	RootAttrs map[string]any
}

func RenderFlowDesignerPanel(view map[string]any, options FlowDesignerPanelOptions) gosx.Node {
	children := []gosx.Node{
		RenderFlowDesignerHeader(),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__grid")),
			renderFlowDesignerList(view),
			renderFlowDesignerEditors(view),
		),
	}
	return gosx.El("section", gosx.Attrs(flowDesignerPanelRootAttrs(view, options.RootAttrs)...), gosx.Fragment(children...))
}

func RenderFlowDesignerHeader() gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Advanced")),
		gosx.El("h2", nil, gosx.Text("Form flows")),
		gosx.El("p", nil, gosx.Text("Build and publish the site forms people use to contact, request, subscribe, and continue checkout.")),
	)
}

func RenderFlowDesignerReadiness(flow map[string]any) gosx.Node {
	readiness := workbenchViewMap(flow, "readiness")
	nodes := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "label"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(readiness, "summary"))),
		),
	}
	checks := workbenchViewMapList(flow, "readinessChecks")
	checkNodes := make([]gosx.Node, 0, len(checks))
	for _, check := range checks {
		checkNodes = append(checkNodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(check, "class")),
			gosx.Attr("role", "listitem"),
			gosx.Attr("tabindex", workbenchMapString(check, "tabIndex")),
			gosx.Attr("data-studio-flow-check", workbenchMapString(check, "key")),
			gosx.Attr("data-studio-flow-check-status", workbenchMapString(check, "status")),
		),
			gosx.El("span", nil, gosx.Text(workbenchMapString(check, "statusLabel"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(check, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(check, "summary"))),
		))
	}
	nodes = append(nodes, gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-flow-checks"),
		gosx.Attr("role", "list"),
		gosx.Attr("aria-label", "Flow readiness checks"),
	), gosx.Fragment(checkNodes...)))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(readiness, "class")),
		gosx.Attr("data-studio-flow-readiness", workbenchMapString(flow, "key")),
	), gosx.Fragment(nodes...))
}

func RenderFlowDesignerGraph(flow map[string]any) gosx.Node {
	nodes := workbenchViewMapList(flow, "nodes")
	children := make([]gosx.Node, 0, len(nodes))
	for _, node := range nodes {
		children = append(children, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(node, "class")),
			gosx.Attr("role", "listitem"),
			gosx.Attr("tabindex", workbenchMapString(node, "tabIndex")),
			gosx.Attr("data-studio-flow-node", workbenchMapString(node, "key")),
			gosx.Attr("data-studio-flow-node-kind", workbenchMapString(node, "kind")),
			gosx.Attr("data-studio-flow-node-status", workbenchMapString(node, "status")),
		),
			gosx.El("span", nil, gosx.Text(workbenchMapString(node, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(node, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(node, "summary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-flow-graph"),
		gosx.Attr("role", "list"),
		gosx.Attr("aria-label", "Flow map"),
	), gosx.Fragment(children...))
}

func RenderFlowDesignerRoute(flow map[string]any, formID string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__route")),
		gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "field"),
			gosx.Attr("for", workbenchMapString(flow, "routeInputID")),
		),
			gosx.El("span", nil, gosx.Text("Public page")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", workbenchMapString(flow, "routeInputID")),
				gosx.Attr("name", workbenchMapString(flow, "routeInputName")),
				gosx.Attr("value", workbenchMapString(flow, "route")),
				gosx.Attr("form", formID),
				gosx.Attr("data-studio-field-source", "flow."+workbenchMapString(flow, "key")+".route"),
				gosx.Attr("data-studio-field-editable", "routing"),
				gosx.Attr("readonly", true),
			)),
		),
		gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "field"),
			gosx.Attr("for", workbenchMapString(flow, "embedInputID")),
		),
			gosx.El("span", nil, gosx.Text("Appears in")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", workbenchMapString(flow, "embedInputID")),
				gosx.Attr("name", workbenchMapString(flow, "embedInputName")),
				gosx.Attr("value", workbenchMapString(flow, "embedTarget")),
				gosx.Attr("form", formID),
				gosx.Attr("data-studio-field-source", "flow."+workbenchMapString(flow, "key")+".embedTarget"),
				gosx.Attr("data-studio-field-editable", "routing"),
				gosx.Attr("readonly", true),
			)),
		),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("id", workbenchMapString(flow, "handlerInputID")),
			gosx.Attr("name", workbenchMapString(flow, "handlerInputName")),
			gosx.Attr("value", workbenchMapString(flow, "handlerRef")),
			gosx.Attr("form", formID),
			gosx.Attr("data-studio-field-source", workbenchMapString(flow, "handlerSource")),
			gosx.Attr("data-studio-field-editable", "flow"),
			gosx.Attr("data-studio-flow-handler-ref", "true"),
			gosx.Attr("type", "hidden"),
		)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(flow, "handlerClass")),
			gosx.Attr("data-studio-flow-handler-summary", workbenchMapString(flow, "key")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(flow, "handlerStatusLabel"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(flow, "handlerDetail"))),
		),
	)
}

func RenderFlowDesignerActions(flow map[string]any) gosx.Node {
	actions := workbenchViewMapList(flow, "actions")
	actionNodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		fields := workbenchViewMapList(action, "fields")
		fieldNodes := make([]gosx.Node, 0, len(fields))
		for _, field := range fields {
			fieldNodes = append(fieldNodes, gosx.El("span", gosx.Attrs(
				gosx.Attr("class", "studio-flow-field"),
				gosx.Attr("data-studio-flow-field", workbenchMapString(field, "name")),
				gosx.Attr("tabindex", "0"),
			),
				gosx.El("strong", nil, gosx.Text(workbenchMapString(field, "label"))),
				gosx.El("small", nil, gosx.Text(workbenchMapString(field, "requiredLabel"))),
			))
		}
		actionNodes = append(actionNodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "studio-flow-action"),
			gosx.Attr("data-studio-flow-action", workbenchMapString(action, "key")),
			gosx.Attr("tabindex", "0"),
		),
			gosx.El("header", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(action, "label"))),
				gosx.El("span", nil,
					gosx.Text(workbenchMapString(action, "fieldCount")),
					gosx.Text(" fields"),
				),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-field-list")), gosx.Fragment(fieldNodes...)),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-flow-editor__actions"),
		gosx.Attr("aria-label", "Flow actions"),
	), gosx.Fragment(actionNodes...))
}

func flowDesignerPanelRootAttrs(view map[string]any, rootAttrs map[string]any) []any {
	attrs := []any{
		gosx.Attr("class", "editor-panel editor-panel--flow-designer studio-flow-designer"),
		gosx.Attr("data-studio-mode-panel", "advanced"),
		gosx.Attr("data-studio-panel", "flow-designer"),
		gosx.Attr("data-studio-flow-designer-panel", "true"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-flow-selected", workbenchMapString(view, "defaultFlowKey")),
		gosx.Attr("data-gosx-studio-flow-designer-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

func renderFlowDesignerList(view map[string]any) gosx.Node {
	flows := workbenchViewMapList(view, "flows")
	nodes := make([]gosx.Node, 0, len(flows))
	selected := workbenchMapString(view, "defaultFlowKey")
	for _, flow := range flows {
		key := workbenchMapString(flow, "key")
		isSelected := key == selected
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("class", workbenchMapString(flow, "cardClass")),
			gosx.Attr("data-editor-flow", key),
			gosx.Attr("data-studio-flow-card", key),
			gosx.Attr("data-studio-flow-card-selected", BoolAttr(isSelected)),
			gosx.Attr("data-studio-flow-route", workbenchMapString(flow, "route")),
			gosx.Attr("data-studio-flow-label", workbenchMapString(flow, "label")),
			gosx.Attr("data-studio-flow-status", workbenchMapString(flow, "statusLabel")),
			gosx.Attr("aria-pressed", BoolAttr(isSelected)),
		),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__head")),
				gosx.El("strong", nil, gosx.Text(workbenchMapString(flow, "label"))),
				gosx.El("output", nil, gosx.Text(workbenchMapString(flow, "statusLabel"))),
			),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__body")), gosx.Text(workbenchMapString(flow, "description"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__meta")),
				gosx.El("span", nil, gosx.Text(workbenchMapString(flow, "summary"))),
				gosx.El("span", nil, gosx.Text(workbenchMapString(flow, "handlerStatusLabel"))),
			),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("class", "studio-flow-summary__dirty"),
				gosx.Attr("data-studio-flow-dirty-badge", key),
				gosx.Attr("data-studio-flow-dirty-visible", "false"),
			), gosx.Text("Unsaved")),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-flow-designer__list"),
		gosx.Attr("role", "list"),
		gosx.Attr("aria-label", "Form flows"),
	), gosx.Fragment(nodes...))
}

func renderFlowDesignerEditors(view map[string]any) gosx.Node {
	flows := workbenchViewMapList(view, "flows")
	nodes := make([]gosx.Node, 0, len(flows))
	selected := workbenchMapString(view, "defaultFlowKey")
	formID := workbenchMapString(view, "formID")
	publishAction := workbenchMapString(view, "publishAction")
	for _, flow := range flows {
		nodes = append(nodes, renderFlowDesignerEditor(flow, formID, publishAction, workbenchMapString(flow, "key") == selected))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__editors")), gosx.Fragment(nodes...))
}

func renderFlowDesignerEditor(flow map[string]any, formID, publishAction string, selected bool) gosx.Node {
	key := workbenchMapString(flow, "key")
	readiness := workbenchViewMap(flow, "readiness")
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", "studio-flow-editor"),
		gosx.Attr("data-editor-flow", key),
		gosx.Attr("data-studio-flow-editor", key),
		gosx.Attr("data-studio-flow-valid", BoolAttr(workbenchMapBool(readiness, "canPublish"))),
		gosx.Attr("data-studio-flow-readiness-status", workbenchMapString(readiness, "status")),
		gosx.Attr("data-studio-flow-dirty", "false"),
		gosx.Attr("data-studio-flow-editor-visible", BoolAttr(selected)),
	),
		renderFlowDesignerEditorHead(flow, formID, publishAction),
		RenderFlowDesignerReadiness(flow),
		RenderFlowDesignerGraph(flow),
		RenderFlowDesignerRoute(flow, formID),
		renderFlowDesignerSteps(flow, formID),
		RenderFlowDesignerActions(flow),
	)
}

func renderFlowDesignerEditorHead(flow map[string]any, formID, publishAction string) gosx.Node {
	key := workbenchMapString(flow, "key")
	readiness := workbenchViewMap(flow, "readiness")
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(flow, "statusLabel"))),
			gosx.El("h3", nil, gosx.Text(workbenchMapString(flow, "label"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(flow, "summary"))),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(readiness, "badgeClass")),
			gosx.Attr("data-studio-flow-editor-state", key),
			gosx.Attr("data-studio-flow-editor-state-visible", "true"),
		), gosx.Text(workbenchMapString(readiness, "label"))),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", "studio-flow-editor__state studio-flow-editor__state--watch"),
			gosx.Attr("data-studio-flow-editor-state", key),
			gosx.Attr("data-studio-flow-editor-state-visible", "false"),
		), gosx.Text("Unsaved changes")),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("form", formID),
			gosx.Attr("formaction", publishAction),
			gosx.Attr("formmethod", "post"),
			gosx.Attr("name", "flowKey"),
			gosx.Attr("value", key),
			gosx.Attr("data-studio-submit-action", "publish-flow"),
		), gosx.Text("Publish flow")),
	)
}

func renderFlowDesignerSteps(flow map[string]any, formID string) gosx.Node {
	steps := workbenchViewMapList(flow, "steps")
	stepNodes := make([]gosx.Node, 0, len(steps))
	for _, step := range steps {
		stepNodes = append(stepNodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "studio-flow-step"),
			gosx.Attr("data-studio-flow-step", workbenchMapString(step, "key")),
			gosx.Attr("tabindex", "0"),
		),
			gosx.El("label", gosx.Attrs(
				gosx.Attr("class", "field"),
				gosx.Attr("for", workbenchMapString(step, "labelInputID")),
			),
				gosx.El("span", nil, gosx.Text(workbenchMapString(step, "label"))),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("id", workbenchMapString(step, "labelInputID")),
					gosx.Attr("name", workbenchMapString(step, "labelInputName")),
					gosx.Attr("value", workbenchMapString(step, "label")),
					gosx.Attr("form", formID),
					gosx.Attr("data-studio-field-source", workbenchMapString(step, "labelSource")),
					gosx.Attr("data-studio-field-editable", "flow"),
				)),
			),
			gosx.El("p", gosx.Attrs(
				gosx.Attr("class", "studio-flow-step__body"),
				gosx.Attr("data-studio-field-source", workbenchMapString(step, "bodySource")),
				gosx.Attr("data-studio-field-editable", "flow"),
			), gosx.Text(workbenchMapString(step, "bodySummary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-flow-editor__steps"),
		gosx.Attr("aria-label", "Flow steps"),
	), gosx.Fragment(stepNodes...))
}
