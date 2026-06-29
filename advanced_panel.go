package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type AdvancedPanelOptions struct {
	RootAttrs  map[string]any
	GroupNodes map[string]gosx.Node
}

type AdvancedPanelSegments struct {
	RootOpen            gosx.Node
	Controls            gosx.Node
	FlowsSlotOpen       gosx.Node
	FlowsSlotClose      gosx.Node
	ToolsSlotOpen       gosx.Node
	ToolsSlotClose      gosx.Node
	SchemaSlotOpen      gosx.Node
	SchemaSlotClose     gosx.Node
	ScheduleSlotOpen    gosx.Node
	ScheduleSlotClose   gosx.Node
	TypographySlotOpen  gosx.Node
	TypographySlotClose gosx.Node
	RootClose           gosx.Node
}

type AdvancedToolsPanelOptions struct {
	RootAttrs map[string]any
}

type AdvancedFieldPanelOptions struct {
	RootAttrs map[string]any
}

func RenderAdvancedPanel(view map[string]any, options AdvancedPanelOptions) gosx.Node {
	children := []gosx.Node{renderAdvancedPanelControls(view)}
	for _, group := range workbenchViewMapList(view, "groups") {
		children = append(children, renderAdvancedPanelGroupSlot(group, options.GroupNodes))
	}
	return gosx.El("section", gosx.Attrs(advancedPanelRootAttrs(view, options.RootAttrs)...), gosx.Fragment(children...))
}

func RenderAdvancedPanelSegments(view map[string]any, options AdvancedPanelOptions) AdvancedPanelSegments {
	groups := advancedPanelGroupsByKey(view)
	return AdvancedPanelSegments{
		RootOpen:            renderBlockLayoutEngineOpenTag("section", advancedPanelRootAttrs(view, options.RootAttrs)),
		Controls:            renderAdvancedPanelControls(view),
		FlowsSlotOpen:       renderAdvancedPanelGroupSlotOpen(groups["flows"]),
		FlowsSlotClose:      gosx.RawHTML("</div></section>"),
		ToolsSlotOpen:       renderAdvancedPanelGroupSlotOpen(groups["tools"]),
		ToolsSlotClose:      gosx.RawHTML("</div></section>"),
		SchemaSlotOpen:      renderAdvancedPanelGroupSlotOpen(groups["schema"]),
		SchemaSlotClose:     gosx.RawHTML("</div></section>"),
		ScheduleSlotOpen:    renderAdvancedPanelGroupSlotOpen(groups["schedule"]),
		ScheduleSlotClose:   gosx.RawHTML("</div></section>"),
		TypographySlotOpen:  renderAdvancedPanelGroupSlotOpen(groups["typography"]),
		TypographySlotClose: gosx.RawHTML("</div></section>"),
		RootClose:           gosx.RawHTML("</section>"),
	}
}

func RenderAdvancedToolsPanel(view map[string]any, options AdvancedToolsPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-panel-key", workbenchMapString(view, "key")),
		gosx.Attr("data-studio-panel", workbenchMapString(view, "key")),
		gosx.Attr("data-studio-mode-panel", workbenchMapString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-advanced-tools-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
		),
		renderAdvancedToolsResources(view),
	}
	items := workbenchViewMapList(view, "items")
	if len(items) == 0 && !workbenchMapBool(view, "hasItems") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(view, "empty"))))
	}
	if len(items) > 0 || workbenchMapBool(view, "hasItems") {
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(view, "gridClass"))), gosx.Fragment(renderAdvancedToolsItems(items)...)))
	}
	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func RenderAdvancedFieldPanel(view map[string]any, options AdvancedFieldPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-panel-key", workbenchMapString(view, "key")),
		gosx.Attr("data-studio-panel", workbenchMapString(view, "key")),
		gosx.Attr("data-studio-mode-panel", workbenchMapString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-advanced-field-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
		),
	}
	fieldList, ok := view["fieldList"].(gosx.Node)
	if !ok || workbenchNodeEmpty(fieldList) || !workbenchMapBool(view, "hasFields") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(view, "empty"))))
	} else {
		children = append(children, fieldList)
	}
	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func advancedPanelRootAttrs(view map[string]any, rootAttrs map[string]any) []any {
	attrs := []any{
		gosx.Attr("class", "editor-panel editor-panel--advanced-studio studio-advanced-panel"),
		gosx.Attr("data-studio-advanced-panel", "true"),
		gosx.Attr("data-studio-mode-panel", "advanced"),
		gosx.Attr("data-studio-panel", "advanced"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-advanced-group-active", workbenchMapString(view, "defaultGroupKey")),
		gosx.Attr("data-gosx-studio-advanced-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

func renderAdvancedPanelControls(view map[string]any) gosx.Node {
	groups := workbenchViewMapList(view, "groups")
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-advanced-panel__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
				gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
				gosx.El("p", nil, gosx.Text(workbenchMapString(view, "summary"))),
			),
		),
	}
	for _, group := range groups {
		children = append(children, renderAdvancedPanelGroupInput(group))
	}
	children = append(children, renderAdvancedPanelGroups(view, groups))
	return gosx.Fragment(children...)
}

func renderAdvancedPanelGroupInput(group map[string]any) gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("class", "studio-advanced-panel__group-input"),
		gosx.Attr("id", workbenchMapString(group, "inputID")),
		gosx.Attr("type", "radio"),
		gosx.Attr("name", "studioAdvancedGroup"),
		gosx.Attr("value", workbenchMapString(group, "key")),
		gosx.Attr("checked", workbenchMapBool(group, "selected")),
		gosx.Attr("aria-label", workbenchMapString(group, "label")),
	))
}

func renderAdvancedPanelGroups(view map[string]any, groups []map[string]any) gosx.Node {
	children := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "studio-advanced-panel__group"),
			gosx.Attr("for", workbenchMapString(group, "inputID")),
			gosx.Attr("data-studio-advanced-group-label", workbenchMapString(group, "key")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(group, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(group, "summary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-advanced-panel__groups"),
		gosx.Attr("role", "radiogroup"),
		gosx.Attr("aria-label", workbenchMapString(view, "groupLabel")),
	), gosx.Fragment(children...))
}

func renderAdvancedPanelGroupSlot(group map[string]any, groupNodes map[string]gosx.Node) gosx.Node {
	key := workbenchMapString(group, "key")
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text(workbenchMapString(group, "label"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(group, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-advanced-panel__group-body"),
			gosx.Attr("data-studio-advanced-group-body", key),
		), renderAdvancedPanelGroupNode(key, groupNodes)),
	}
	return gosx.El("section", gosx.Attrs(advancedPanelGroupSlotAttrs(group)...), gosx.Fragment(children...))
}

func renderAdvancedPanelGroupNode(key string, groupNodes map[string]gosx.Node) gosx.Node {
	if groupNodes == nil {
		return gosx.Fragment()
	}
	node, ok := groupNodes[key]
	if !ok {
		return gosx.Fragment()
	}
	if workbenchNodeEmpty(node) {
		return gosx.Fragment()
	}
	return node
}

func renderAdvancedPanelGroupSlotOpen(group map[string]any) gosx.Node {
	key := workbenchMapString(group, "key")
	html := gosx.RenderHTML(gosx.El("section", gosx.Attrs(advancedPanelGroupSlotAttrs(group)...),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text(workbenchMapString(group, "label"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(group, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-advanced-panel__group-body"),
			gosx.Attr("data-studio-advanced-group-body", key),
		)),
	))
	html = strings.TrimSuffix(html, "</div></section>")
	return gosx.RawHTML(html)
}

func advancedPanelGroupSlotAttrs(group map[string]any) []any {
	return []any{
		gosx.Attr("class", "studio-advanced-panel__group-slot"),
		gosx.Attr("data-studio-advanced-group-slot", workbenchMapString(group, "key")),
		gosx.Attr("data-studio-advanced-group-selected", BoolAttr(workbenchMapBool(group, "selected"))),
	}
}

func advancedPanelGroupsByKey(view map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, group := range workbenchViewMapList(view, "groups") {
		out[workbenchMapString(group, "key")] = group
	}
	return out
}

func renderAdvancedToolsResources(view map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(view, "resourcesTitle"))),
				gosx.El("p", nil, gosx.Text(workbenchMapString(view, "resourcesSummary"))),
			),
		),
	}
	adapters := workbenchViewMapList(view, "adapters")
	if len(adapters) == 0 && !workbenchMapBool(view, "hasAdapters") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(view, "resourceEmpty"))))
	}
	if len(adapters) > 0 || workbenchMapBool(view, "hasAdapters") {
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(view, "resourceGridClass"))), gosx.Fragment(renderAdvancedToolsAdapters(adapters)...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-resource-adapters"),
		gosx.Attr("data-studio-resource-adapters", "true"),
	), gosx.Fragment(children...))
}

func renderAdvancedToolsAdapters(adapters []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(adapters))
	for _, adapter := range adapters {
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", "studio-resource-adapter-card"),
			gosx.Attr("data-studio-resource-adapter", workbenchMapString(adapter, "kind")),
			gosx.Attr("data-studio-resource-surface", workbenchMapString(adapter, "surface")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(adapter, "label"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(adapter, "summary"))),
			gosx.El("div", nil,
				gosx.El("small", nil, gosx.Text(workbenchMapString(adapter, "surface"))),
				gosx.El("small", nil, gosx.Text(workbenchMapString(adapter, "capabilityLabel"))),
				gosx.El("small", nil, gosx.Text(workbenchMapString(adapter, "bindingLabel"))),
			),
		))
	}
	return nodes
}

func renderAdvancedToolsItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "attrs"))...),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(item, "label"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(item, "summary"))),
		))
	}
	return nodes
}
