package panels

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

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
	readiness := core.WorkbenchViewMap(flow, "readiness")
	nodes := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(readiness, "label"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(readiness, "summary"))),
		),
	}
	checks := core.WorkbenchViewMapList(flow, "readinessChecks")
	checkNodes := make([]gosx.Node, 0, len(checks))
	for _, check := range checks {
		checkNodes = append(checkNodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(check, "class")),
			gosx.Attr("role", "listitem"),
			gosx.Attr("tabindex", core.WorkbenchViewString(check, "tabIndex")),
			gosx.Attr("data-studio-flow-check", core.WorkbenchViewString(check, "key")),
			gosx.Attr("data-studio-flow-check-status", core.WorkbenchViewString(check, "status")),
		),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(check, "statusLabel"))),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(check, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(check, "summary"))),
		))
	}
	nodes = append(nodes, gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-flow-checks"),
		gosx.Attr("role", "list"),
		gosx.Attr("aria-label", "Flow readiness checks"),
	), gosx.Fragment(checkNodes...)))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(readiness, "class")),
		gosx.Attr("data-studio-flow-readiness", core.WorkbenchViewString(flow, "key")),
	), gosx.Fragment(nodes...))
}

func RenderFlowDesignerGraph(flow map[string]any) gosx.Node {
	nodes := core.WorkbenchViewMapList(flow, "nodes")
	children := make([]gosx.Node, 0, len(nodes))
	for _, node := range nodes {
		children = append(children, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(node, "class")),
			gosx.Attr("role", "listitem"),
			gosx.Attr("tabindex", core.WorkbenchViewString(node, "tabIndex")),
			gosx.Attr("data-studio-flow-node", core.WorkbenchViewString(node, "key")),
			gosx.Attr("data-studio-flow-node-kind", core.WorkbenchViewString(node, "kind")),
			gosx.Attr("data-studio-flow-node-status", core.WorkbenchViewString(node, "status")),
		),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(node, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(node, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(node, "summary"))),
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
			gosx.Attr("for", core.WorkbenchViewString(flow, "routeInputID")),
		),
			gosx.El("span", nil, gosx.Text("Public page")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", core.WorkbenchViewString(flow, "routeInputID")),
				gosx.Attr("name", core.WorkbenchViewString(flow, "routeInputName")),
				gosx.Attr("value", core.WorkbenchViewString(flow, "route")),
				gosx.Attr("form", formID),
				gosx.Attr("data-studio-field-source", "flow."+core.WorkbenchViewString(flow, "key")+".route"),
				gosx.Attr("data-studio-field-editable", "routing"),
				gosx.Attr("readonly", true),
			)),
		),
		gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "field"),
			gosx.Attr("for", core.WorkbenchViewString(flow, "embedInputID")),
		),
			gosx.El("span", nil, gosx.Text("Appears in")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", core.WorkbenchViewString(flow, "embedInputID")),
				gosx.Attr("name", core.WorkbenchViewString(flow, "embedInputName")),
				gosx.Attr("value", core.WorkbenchViewString(flow, "embedTarget")),
				gosx.Attr("form", formID),
				gosx.Attr("data-studio-field-source", "flow."+core.WorkbenchViewString(flow, "key")+".embedTarget"),
				gosx.Attr("data-studio-field-editable", "routing"),
				gosx.Attr("readonly", true),
			)),
		),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("id", core.WorkbenchViewString(flow, "handlerInputID")),
			gosx.Attr("name", core.WorkbenchViewString(flow, "handlerInputName")),
			gosx.Attr("value", core.WorkbenchViewString(flow, "handlerRef")),
			gosx.Attr("form", formID),
			gosx.Attr("data-studio-field-source", core.WorkbenchViewString(flow, "handlerSource")),
			gosx.Attr("data-studio-field-editable", "flow"),
			gosx.Attr("data-studio-flow-handler-ref", "true"),
			gosx.Attr("type", "hidden"),
		)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(flow, "handlerClass")),
			gosx.Attr("data-studio-flow-handler-summary", core.WorkbenchViewString(flow, "key")),
		),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(flow, "handlerStatusLabel"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(flow, "handlerDetail"))),
		),
	)
}

func RenderFlowDesignerActions(flow map[string]any) gosx.Node {
	actions := core.WorkbenchViewMapList(flow, "actions")
	actionNodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		fields := core.WorkbenchViewMapList(action, "fields")
		fieldNodes := make([]gosx.Node, 0, len(fields))
		for _, field := range fields {
			fieldNodes = append(fieldNodes, gosx.El("span", gosx.Attrs(
				gosx.Attr("class", "studio-flow-field"),
				gosx.Attr("data-studio-flow-field", core.WorkbenchViewString(field, "name")),
				gosx.Attr("tabindex", "0"),
			),
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(field, "label"))),
				gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(field, "requiredLabel"))),
			))
		}
		actionNodes = append(actionNodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "studio-flow-action"),
			gosx.Attr("data-studio-flow-action", core.WorkbenchViewString(action, "key")),
			gosx.Attr("tabindex", "0"),
		),
			gosx.El("header", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(action, "label"))),
				gosx.El("span", nil,
					gosx.Text(core.WorkbenchViewString(action, "fieldCount")),
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
		gosx.Attr("data-studio-flow-selected", core.WorkbenchViewString(view, "defaultFlowKey")),
		gosx.Attr("data-gosx-studio-flow-designer-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

func renderFlowDesignerList(view map[string]any) gosx.Node {
	flows := core.WorkbenchViewMapList(view, "flows")
	nodes := make([]gosx.Node, 0, len(flows))
	selected := core.WorkbenchViewString(view, "defaultFlowKey")
	for _, flow := range flows {
		key := core.WorkbenchViewString(flow, "key")
		isSelected := key == selected
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("class", core.WorkbenchViewString(flow, "cardClass")),
			gosx.Attr("data-editor-flow", key),
			gosx.Attr("data-studio-flow-card", key),
			gosx.Attr("data-studio-flow-card-selected", core.BoolAttr(isSelected)),
			gosx.Attr("data-studio-flow-route", core.WorkbenchViewString(flow, "route")),
			gosx.Attr("data-studio-flow-label", core.WorkbenchViewString(flow, "label")),
			gosx.Attr("data-studio-flow-status", core.WorkbenchViewString(flow, "statusLabel")),
			gosx.Attr("aria-pressed", core.BoolAttr(isSelected)),
		),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__head")),
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(flow, "label"))),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(flow, "statusLabel"))),
			),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__body")), gosx.Text(core.WorkbenchViewString(flow, "description"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__meta")),
				gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(flow, "summary"))),
				gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(flow, "handlerStatusLabel"))),
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
	flows := core.WorkbenchViewMapList(view, "flows")
	nodes := make([]gosx.Node, 0, len(flows))
	selected := core.WorkbenchViewString(view, "defaultFlowKey")
	formID := core.WorkbenchViewString(view, "formID")
	publishAction := core.WorkbenchViewString(view, "publishAction")
	for _, flow := range flows {
		nodes = append(nodes, renderFlowDesignerEditor(flow, formID, publishAction, core.WorkbenchViewString(flow, "key") == selected))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__editors")), gosx.Fragment(nodes...))
}

func renderFlowDesignerEditor(flow map[string]any, formID, publishAction string, selected bool) gosx.Node {
	key := core.WorkbenchViewString(flow, "key")
	readiness := core.WorkbenchViewMap(flow, "readiness")
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", "studio-flow-editor"),
		gosx.Attr("data-editor-flow", key),
		gosx.Attr("data-studio-flow-editor", key),
		gosx.Attr("data-studio-flow-valid", core.BoolAttr(core.WorkbenchViewBool(readiness, "canPublish"))),
		gosx.Attr("data-studio-flow-readiness-status", core.WorkbenchViewString(readiness, "status")),
		gosx.Attr("data-studio-flow-dirty", "false"),
		gosx.Attr("data-studio-flow-editor-visible", core.BoolAttr(selected)),
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
	key := core.WorkbenchViewString(flow, "key")
	readiness := core.WorkbenchViewMap(flow, "readiness")
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(flow, "statusLabel"))),
			gosx.El("h3", nil, gosx.Text(core.WorkbenchViewString(flow, "label"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(flow, "summary"))),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(readiness, "badgeClass")),
			gosx.Attr("data-studio-flow-editor-state", key),
			gosx.Attr("data-studio-flow-editor-state-visible", "true"),
		), gosx.Text(core.WorkbenchViewString(readiness, "label"))),
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
	steps := core.WorkbenchViewMapList(flow, "steps")
	stepNodes := make([]gosx.Node, 0, len(steps))
	for _, step := range steps {
		stepNodes = append(stepNodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "studio-flow-step"),
			gosx.Attr("data-studio-flow-step", core.WorkbenchViewString(step, "key")),
			gosx.Attr("tabindex", "0"),
		),
			gosx.El("label", gosx.Attrs(
				gosx.Attr("class", "field"),
				gosx.Attr("for", core.WorkbenchViewString(step, "labelInputID")),
			),
				gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(step, "label"))),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("id", core.WorkbenchViewString(step, "labelInputID")),
					gosx.Attr("name", core.WorkbenchViewString(step, "labelInputName")),
					gosx.Attr("value", core.WorkbenchViewString(step, "label")),
					gosx.Attr("form", formID),
					gosx.Attr("data-studio-field-source", core.WorkbenchViewString(step, "labelSource")),
					gosx.Attr("data-studio-field-editable", "flow"),
				)),
			),
			gosx.El("p", gosx.Attrs(
				gosx.Attr("class", "studio-flow-step__body"),
				gosx.Attr("data-studio-field-source", core.WorkbenchViewString(step, "bodySource")),
				gosx.Attr("data-studio-field-editable", "flow"),
			), gosx.Text(core.WorkbenchViewString(step, "bodySummary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-flow-editor__steps"),
		gosx.Attr("aria-label", "Flow steps"),
	), gosx.Fragment(stepNodes...))
}
