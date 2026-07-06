package panels

import (
	"strings"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
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
	SettingsSlotOpen    gosx.Node
	SettingsSlotClose   gosx.Node
	RootClose           gosx.Node
}

type AdvancedToolsPanelOptions struct {
	RootAttrs map[string]any
}

type AdvancedFieldPanelOptions struct {
	RootAttrs map[string]any
}

type AdvancedSettingsPanelOptions struct {
	RootAttrs map[string]any
}

func RenderAdvancedPanel(view map[string]any, options AdvancedPanelOptions) gosx.Node {
	children := []gosx.Node{renderAdvancedPanelControls(view)}
	for _, group := range core.WorkbenchViewMapList(view, "groups") {
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
		SettingsSlotOpen:    renderAdvancedPanelGroupSlotOpen(groups["settings"]),
		SettingsSlotClose:   gosx.RawHTML("</div></section>"),
		RootClose:           gosx.RawHTML("</section>"),
	}
}

func RenderAdvancedToolsPanel(view map[string]any, options AdvancedToolsPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(view, "key")),
		gosx.Attr("data-studio-panel", core.WorkbenchViewString(view, "key")),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-advanced-tools-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
		),
		renderAdvancedToolsResources(view),
	}
	items := core.WorkbenchViewMapList(view, "items")
	if len(items) == 0 && !core.WorkbenchViewBool(view, "hasItems") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(core.WorkbenchViewString(view, "empty"))))
	}
	if len(items) > 0 || core.WorkbenchViewBool(view, "hasItems") {
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(view, "gridClass"))), gosx.Fragment(renderAdvancedToolsItems(items)...)))
	}
	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func RenderAdvancedFieldPanel(view map[string]any, options AdvancedFieldPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(view, "key")),
		gosx.Attr("data-studio-panel", core.WorkbenchViewString(view, "key")),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-advanced-field-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
		),
	}
	fieldList, ok := view["fieldList"].(gosx.Node)
	if !ok || core.WorkbenchNodeEmpty(fieldList) || !core.WorkbenchViewBool(view, "hasFields") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(core.WorkbenchViewString(view, "empty"))))
	} else {
		children = append(children, fieldList)
	}
	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func RenderAdvancedSettingsPanel(view map[string]any, options AdvancedSettingsPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(view, "key")),
		gosx.Attr("data-studio-panel", core.WorkbenchViewString(view, "key")),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-advanced-settings-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
		),
	}
	if summary := core.WorkbenchViewString(view, "summary"); summary != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-advanced-settings-panel__summary")), gosx.Text(summary)))
	}
	fieldList, ok := view["fieldList"].(gosx.Node)
	if ok && !core.WorkbenchNodeEmpty(fieldList) && core.WorkbenchViewBool(view, "hasFields") {
		children = append(children, fieldList)
	} else {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(core.WorkbenchViewString(view, "empty"))))
	}
	if statusCards := renderAdvancedSettingsStatusCards(view); !core.WorkbenchNodeEmpty(statusCards) {
		children = append(children, statusCards)
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
		gosx.Attr("data-studio-advanced-group-active", core.WorkbenchViewString(view, "defaultGroupKey")),
		gosx.Attr("data-gosx-studio-advanced-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

func renderAdvancedSettingsStatusCards(view map[string]any) gosx.Node {
	cards := core.WorkbenchViewMapList(view, "statusCards")
	if len(cards) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(cards)+1)
	for _, card := range cards {
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(card, "class")),
			gosx.Attr("data-studio-advanced-settings-status", core.WorkbenchViewString(card, "key")),
		),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(card, "title"))),
				gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(card, "summary"))),
			),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(card, "status"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(core.WorkbenchViewString(view, "statusGridClass"), "studio-advanced-settings-panel__status-grid")),
		gosx.Attr("data-studio-advanced-settings-status-grid", "true"),
	), gosx.Fragment(nodes...))
}

func renderAdvancedPanelControls(view map[string]any) gosx.Node {
	groups := core.WorkbenchViewMapList(view, "groups")
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-advanced-panel__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
				gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
				gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(view, "summary"))),
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
		gosx.Attr("id", core.WorkbenchViewString(group, "inputID")),
		gosx.Attr("type", "radio"),
		gosx.Attr("name", "studioAdvancedGroup"),
		gosx.Attr("value", core.WorkbenchViewString(group, "key")),
		gosx.Attr("checked", core.WorkbenchViewBool(group, "selected")),
		gosx.Attr("aria-label", core.WorkbenchViewString(group, "label")),
	))
}

func renderAdvancedPanelGroups(view map[string]any, groups []map[string]any) gosx.Node {
	children := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "studio-advanced-panel__group"),
			gosx.Attr("for", core.WorkbenchViewString(group, "inputID")),
			gosx.Attr("data-studio-advanced-group-label", core.WorkbenchViewString(group, "key")),
		),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(group, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(group, "summary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-advanced-panel__groups"),
		gosx.Attr("role", "radiogroup"),
		gosx.Attr("aria-label", core.WorkbenchViewString(view, "groupLabel")),
	), gosx.Fragment(children...))
}

func renderAdvancedPanelGroupSlot(group map[string]any, groupNodes map[string]gosx.Node) gosx.Node {
	key := core.WorkbenchViewString(group, "key")
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text(core.WorkbenchViewString(group, "label"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(group, "summary"))),
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
	if core.WorkbenchNodeEmpty(node) {
		return gosx.Fragment()
	}
	return node
}

func renderAdvancedPanelGroupSlotOpen(group map[string]any) gosx.Node {
	key := core.WorkbenchViewString(group, "key")
	html := gosx.RenderHTML(gosx.El("section", gosx.Attrs(advancedPanelGroupSlotAttrs(group)...),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text(core.WorkbenchViewString(group, "label"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(group, "summary"))),
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
		gosx.Attr("data-studio-advanced-group-slot", core.WorkbenchViewString(group, "key")),
		gosx.Attr("data-studio-advanced-group-selected", core.BoolAttr(core.WorkbenchViewBool(group, "selected"))),
	}
}

func advancedPanelGroupsByKey(view map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, group := range core.WorkbenchViewMapList(view, "groups") {
		out[core.WorkbenchViewString(group, "key")] = group
	}
	return out
}

func renderAdvancedToolsResources(view map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(view, "resourcesTitle"))),
				gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(view, "resourcesSummary"))),
			),
		),
	}
	adapters := core.WorkbenchViewMapList(view, "adapters")
	if len(adapters) == 0 && !core.WorkbenchViewBool(view, "hasAdapters") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(core.WorkbenchViewString(view, "resourceEmpty"))))
	}
	if len(adapters) > 0 || core.WorkbenchViewBool(view, "hasAdapters") {
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(view, "resourceGridClass"))), gosx.Fragment(renderAdvancedToolsAdapters(adapters)...)))
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
			gosx.Attr("data-studio-resource-adapter", core.WorkbenchViewString(adapter, "kind")),
			gosx.Attr("data-studio-resource-surface", core.WorkbenchViewString(adapter, "surface")),
		),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(adapter, "label"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(adapter, "summary"))),
			gosx.El("div", nil,
				gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(adapter, "surface"))),
				gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(adapter, "capabilityLabel"))),
				gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(adapter, "bindingLabel"))),
			),
		))
	}
	return nodes
}

func renderAdvancedToolsItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "attrs"))...),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(item, "label"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "summary"))),
		))
	}
	return nodes
}
