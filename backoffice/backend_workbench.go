package backoffice

import "m31labs.dev/gosx"

type BackendWorkbenchProps struct {
	Class     string
	Resources []BackendWorkbenchResource
	Tools     []BackendWorkbenchTool
}

type BackendWorkbenchResource struct {
	Label          string
	Description    string
	Route          string
	MutableLabel   string
	GeneratedLabel string
	CountLabel     string
	Fields         []BackendWorkbenchField
	Actions        []BackendWorkbenchAction
}

type BackendWorkbenchField struct {
	Label string
	Kind  string
}

type BackendWorkbenchAction struct {
	Label string
	Kind  string
}

type BackendWorkbenchTool struct {
	Kind        string
	Label       string
	Description string
	Route       string
}

func RenderBackendWorkbench(props BackendWorkbenchProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-workbench-renderer", "gosx-studio"),
	),
		RenderBackendWorkbenchContent(props),
	)
}

func RenderBackendWorkbenchContent(props BackendWorkbenchProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendWorkbenchHeading(),
		RenderBackendWorkbenchResources(props.Resources),
		RenderBackendWorkbenchTools(props.Tools),
	)
}

func RenderBackendWorkbenchHeading() gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Workbench")),
		gosx.El("h1", nil, gosx.Text("GoSX internal app surface")),
		gosx.El("p", nil, gosx.Text("Resources, fields, server actions, and API tools are defined in Go and rendered through GoSX.")),
	)
}

func RenderBackendWorkbenchResources(resources []BackendWorkbenchResource) gosx.Node {
	nodes := make([]gosx.Node, 0, len(resources))
	for _, resource := range resources {
		nodes = append(nodes, renderBackendWorkbenchResource(resource))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "workbench-grid")), gosx.Fragment(nodes...))
}

func RenderBackendWorkbenchTools(tools []BackendWorkbenchTool) gosx.Node {
	nodes := make([]gosx.Node, 0, len(tools))
	for _, tool := range tools {
		nodes = append(nodes, gosx.El("article", nil,
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(tool.Kind)),
			gosx.El("strong", nil, gosx.Text(tool.Label)),
			gosx.El("p", nil, gosx.Text(tool.Description)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "field-note")), gosx.Text(tool.Route)),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Runtime tools")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "tool-list")), gosx.Fragment(nodes...)),
	)
}

func renderBackendWorkbenchResource(resource BackendWorkbenchResource) gosx.Node {
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", "panel resource-detail")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("div", nil,
				gosx.El("h2", nil, gosx.Text(resource.Label)),
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "field-note")), gosx.Text(resource.Description)),
			),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "button button--secondary"),
				gosx.Attr("href", resource.Route),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text("Open")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "capability-row")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(resource.MutableLabel)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(resource.GeneratedLabel)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(resource.CountLabel)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "schema-columns")),
			gosx.El("div", nil,
				gosx.El("h3", nil, gosx.Text("Fields")),
				gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list")), gosx.Fragment(renderBackendWorkbenchFields(resource.Fields)...)),
			),
			gosx.El("div", nil,
				gosx.El("h3", nil, gosx.Text("Actions")),
				renderBackendWorkbenchActions(resource.Actions),
			),
		),
	)
}

func renderBackendWorkbenchFields(fields []BackendWorkbenchField) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(fields))
	for _, field := range fields {
		nodes = append(nodes, gosx.El("li", nil,
			gosx.El("strong", nil, gosx.Text(field.Label)),
			gosx.El("span", nil, gosx.Text(field.Kind)),
		))
	}
	return nodes
}

func renderBackendWorkbenchActions(actions []BackendWorkbenchAction) gosx.Node {
	if len(actions) == 0 {
		return gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text("No direct actions."))
	}
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		nodes = append(nodes, gosx.El("li", nil,
			gosx.El("strong", nil, gosx.Text(action.Label)),
			gosx.El("span", nil, gosx.Text(action.Kind)),
		))
	}
	return gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list")), gosx.Fragment(nodes...))
}
