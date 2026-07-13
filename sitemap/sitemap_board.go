package sitemap

import (
	"encoding/json"
	"strconv"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type SiteMapBoardOptions struct {
	Class             string
	InspectorInputID  string
	DefaultGroupKey   string
	DefaultDetailKey  string
	DefaultFocusKey   string
	DefaultPaletteKey string
	DefaultZoomKey    string
	DefaultDensityKey string
	DefaultGridKey    string
	HideMinimap       bool
}

type siteMapBoardState struct {
	group                 string
	detail                string
	palette               string
	focus                 string
	zoom                  string
	density               string
	grid                  string
	selectedBlueprintKey  string
	selectedTemplateKey   string
	selectedNodeKey       string
	selectedNodeLabel     string
	selectedNodeKind      string
	selectedNodeSummary   string
	selectedNodeRoute     string
	selectedNodeSource    string
	selectedNodeBinding   string
	selectedNodeStatus    string
	selectedNodeComponent string
	inspectorInputID      string
}

type siteMapBoardChoice struct {
	Label string
	Key   string
}

func RenderSiteMapBoard(siteMapView map[string]any, options SiteMapBoardOptions) gosx.Node {
	state := siteMapBoardInitialState(siteMapView, options)
	children := []gosx.Node{
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-board__inspector-input"),
			gosx.Attr("id", state.inspectorInputID),
			gosx.Attr("type", "checkbox"),
		)),
		renderSiteMapBoardToolbar(siteMapView, state),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!core.WorkbenchViewBool(siteMapView, "hasPages"))),
		), gosx.Text(core.WorkbenchViewString(siteMapView, "empty"))),
	}
	bodyChildren := []gosx.Node{
		renderSiteMapBoardWorkspace(siteMapView, state),
		renderSiteMapBoardBuilder(siteMapView, state),
		renderSiteMapBoardViewport(siteMapView, state),
	}
	if !options.HideMinimap {
		bodyChildren = append(bodyChildren, renderSiteMapBoardMinimap(siteMapView))
	}
	children = append(children, gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-board__body"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasPages"))),
	), gosx.Fragment(bodyChildren...)))

	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-site-map-board")),
		gosx.Attr("data-studio-site-map-board", "true"),
		gosx.Attr("data-gosx-studio-site-map-board-renderer", "gosx-studio"),
		gosx.Attr("data-studio-site-map-filter", state.group),
		gosx.Attr("data-studio-site-map-detail-control", "runtime"),
		gosx.Attr("data-studio-site-map-detail", state.detail),
		gosx.Attr("data-studio-site-map-palette", state.palette),
		gosx.Attr("data-studio-site-map-focus-control", "runtime"),
		gosx.Attr("data-studio-site-map-focus", state.focus),
		gosx.Attr("data-studio-site-map-zoom", state.zoom),
		gosx.Attr("data-studio-site-map-density", state.density),
		gosx.Attr("data-studio-site-map-grid", state.grid),
		gosx.Attr("data-studio-site-map-scale", "1"),
		gosx.Attr("data-studio-site-map-pan-x", "0"),
		gosx.Attr("data-studio-site-map-pan-y", "0"),
		gosx.Attr("data-studio-site-map-selected-node", state.selectedNodeKey),
		gosx.Attr("data-studio-site-map-selected-blueprint", state.selectedBlueprintKey),
		gosx.Attr("data-studio-site-map-selected-template", state.selectedTemplateKey),
	), gosx.Fragment(children...))
}

func siteMapBoardInitialState(siteMapView map[string]any, options SiteMapBoardOptions) siteMapBoardState {
	return siteMapBoardState{
		group:                 core.FirstNonEmpty(options.DefaultGroupKey, core.WorkbenchViewString(siteMapView, "defaultGroupKey"), "all"),
		detail:                core.FirstNonEmpty(options.DefaultDetailKey, core.WorkbenchViewString(siteMapView, "defaultDetailKey"), "map"),
		palette:               core.FirstNonEmpty(options.DefaultPaletteKey, "all"),
		focus:                 core.FirstNonEmpty(options.DefaultFocusKey, core.WorkbenchViewString(siteMapView, "defaultFocusKey"), "all"),
		zoom:                  core.FirstNonEmpty(options.DefaultZoomKey, "fit"),
		density:               core.FirstNonEmpty(options.DefaultDensityKey, "roomy"),
		grid:                  core.FirstNonEmpty(options.DefaultGridKey, "on"),
		selectedBlueprintKey:  core.WorkbenchViewString(siteMapView, "defaultBlueprintKey"),
		selectedTemplateKey:   core.WorkbenchViewString(siteMapView, "defaultTemplateKey"),
		selectedNodeKey:       core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeKey"),
		selectedNodeLabel:     core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeLabel"),
		selectedNodeKind:      core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeKind"),
		selectedNodeSummary:   core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeSummary"),
		selectedNodeRoute:     core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeRoute"),
		selectedNodeSource:    core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeSource"),
		selectedNodeBinding:   core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeBinding"),
		selectedNodeStatus:    core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeStatus"),
		selectedNodeComponent: core.WorkbenchViewString(siteMapView, "defaultWorkspaceNodeComponentLabel"),
		inspectorInputID:      core.FirstNonEmpty(options.InspectorInputID, "studioSiteMapInspectorFocus"),
	}
}

func renderSiteMapBoardToolbar(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-board__toolbar"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Site map board filters"),
	),
		renderSiteMapBoardChoiceGroup("studio-site-map-board__filter", "group", "Page groups", "data-studio-site-map-filter-control", state.group, []siteMapBoardChoice{
			{Label: "All", Key: "all"},
			{Label: "Site", Key: "site"},
			{Label: "Store", Key: "commerce"},
			{Label: "Content", Key: "content"},
			{Label: "Flows", Key: "flows"},
		}),
		renderSiteMapBoardChoiceGroup("studio-site-map-board__focus", "group", "Page focus", "data-studio-site-map-focus-control", state.focus, append([]siteMapBoardChoice{{Label: "All pages", Key: "all"}}, siteMapBoardPageChoices(siteMapView)...)),
		renderSiteMapBoardChoiceGroup("studio-site-map-board__view", "group", "Board view", "", "", nil,
			renderSiteMapBoardActionControl("Zoom out", "data-studio-site-map-zoom-action", "out"),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("class", "studio-site-map-board__zoom-readout"),
				gosx.Attr("data-studio-site-map-zoom-readout", "true"),
				gosx.Attr("aria-label", "Zoom level"),
			), gosx.Text("100%")),
			renderSiteMapBoardActionControl("Zoom in", "data-studio-site-map-zoom-action", "in"),
			renderSiteMapBoardActionControl("Reset view", "data-studio-site-map-zoom-action", "reset"),
			renderSiteMapBoardControl("Fit", "data-studio-site-map-zoom-control", "fit", state.zoom),
			renderSiteMapBoardControl("Wide", "data-studio-site-map-zoom-control", "wide", state.zoom),
			renderSiteMapBoardControl("Dense", "data-studio-site-map-density-control", "dense", state.density),
			renderSiteMapBoardControl("Roomy", "data-studio-site-map-density-control", "roomy", state.density),
			renderSiteMapBoardControl("Grid", "data-studio-site-map-grid-control", "on", state.grid),
			renderSiteMapBoardControl("Plain", "data-studio-site-map-grid-control", "off", state.grid),
		),
		renderSiteMapBoardChoiceGroup("studio-site-map-board__detail", "radiogroup", "Component detail", "data-studio-site-map-detail-target", state.detail, []siteMapBoardChoice{
			{Label: "Map", Key: "map"},
			{Label: "Add", Key: "build"},
			{Label: "Components", Key: "components"},
			{Label: "Controls", Key: "controls"},
			{Label: "Sources", Key: "sources"},
		}),
	)
}

func siteMapBoardPageChoices(siteMapView map[string]any) []siteMapBoardChoice {
	pages := core.WorkbenchViewMapList(siteMapView, "pages")
	choices := make([]siteMapBoardChoice, 0, len(pages))
	for _, page := range pages {
		key := core.WorkbenchViewString(page, "key")
		if key == "" {
			continue
		}
		choices = append(choices, siteMapBoardChoice{Label: core.WorkbenchViewString(page, "label"), Key: key})
	}
	return choices
}

func renderSiteMapBoardChoiceGroup(className, role, label, attrName, current string, choices []siteMapBoardChoice, extra ...gosx.Node) gosx.Node {
	children := make([]gosx.Node, 0, len(choices)+len(extra))
	for _, choice := range choices {
		children = append(children, renderSiteMapBoardControl(choice.Label, attrName, choice.Key, current))
	}
	children = append(children, extra...)
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("role", role),
		gosx.Attr("aria-label", label),
	), gosx.Fragment(children...))
}

func renderSiteMapBoardControl(label, attrName, key, current string) gosx.Node {
	attrs := []any{
		gosx.Attr("type", "button"),
		gosx.Attr(attrName, key),
		gosx.Attr("aria-pressed", siteMapBoardBoolString(key == current)),
	}
	return gosx.El("button", gosx.Attrs(attrs...), gosx.Text(label))
}

// renderSiteMapBoardActionControl renders a momentary action button (no pressed
// state) such as the continuous zoom in/out/reset controls driven by the
// site-map runtime's infinite-canvas viewport.
func renderSiteMapBoardActionControl(label, attrName, key string) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("type", "button"),
		gosx.Attr(attrName, key),
	), gosx.Text(label))
}

func renderSiteMapBoardWorkspace(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-workspace"),
		gosx.Attr("data-studio-site-map-workspace", "true"),
		gosx.Attr("data-studio-site-map-panel-visible", siteMapBoardBoolString(state.detail == "map")),
	),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace__chrome")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text("Site structure")),
				gosx.El("small", nil, gosx.Text("Pages, sections, content sources, and reusable blocks.")),
			),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "studio-site-map-workspace__stats"),
				gosx.Attr("aria-label", "Composition board summary"),
			),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "workspaceLayerCountLabel"))),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "workspaceNodeCountLabel"))),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "workspaceLinkCountLabel"))),
			),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-workspace__body"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasWorkspace"))),
		),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "studio-site-map-workspace__canvas"),
				gosx.Attr("data-studio-site-map-canvas", "true"),
				gosx.Attr("tabindex", "0"),
				gosx.Attr("role", "group"),
				gosx.Attr("aria-label", "Site map canvas. Drag to pan, scroll to zoom, press plus or minus to zoom and zero to reset."),
			),
				gosx.El("div", gosx.Attrs(
					gosx.Attr("class", "studio-site-map-workspace__surface"),
					gosx.Attr("data-studio-site-map-pan-surface", "true"),
					// position:relative anchors absolutely-positioned nodes
					// (drag-to-reposition) to the pan surface. Nodes without a
					// saved position keep flowing in lanes, so this is
					// layout-neutral for the default view.
					gosx.Attr("style", "position:relative"),
				),
					renderSiteMapBoardCanvasPaths(siteMapView, state),
					renderSiteMapBoardWorkspaceLayers(siteMapView, state),
				),
				renderSiteMapBoardSelectionCard(siteMapView, state),
			),
			renderSiteMapBoardWorkspaceRail(siteMapView, state),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!core.WorkbenchViewBool(siteMapView, "hasWorkspace"))),
		), gosx.Text("No composition workspace is configured.")),
	)
}

func renderSiteMapBoardCanvasPaths(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	paths := core.WorkbenchViewMapList(siteMapView, "workspaceCanvasLinks")
	children := make([]gosx.Node, 0, len(paths))
	for _, link := range paths {
		children = append(children, gosx.El("path", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(link, "class")),
			gosx.Attr("d", core.WorkbenchViewString(link, "path")),
			gosx.Attr("data-studio-site-map-workspace-path", core.WorkbenchViewString(link, "key")),
			gosx.Attr("data-studio-site-map-link-kind", core.WorkbenchViewString(link, "kind")),
			gosx.Attr("data-studio-site-map-link-from", core.WorkbenchViewString(link, "fromNodeKey")),
			gosx.Attr("data-studio-site-map-link-to", core.WorkbenchViewString(link, "toNodeKey")),
			gosx.Attr("data-studio-site-map-link-from-page", core.WorkbenchViewString(link, "fromPageKey")),
			gosx.Attr("data-studio-site-map-link-to-page", core.WorkbenchViewString(link, "toPageKey")),
			gosx.Attr("data-studio-site-map-link-from-group", core.WorkbenchViewString(link, "fromGroup")),
			gosx.Attr("data-studio-site-map-link-to-group", core.WorkbenchViewString(link, "toGroup")),
			gosx.Attr("data-studio-site-map-link-selected", siteMapBoardBoolString(siteMapBoardLinkSelected(link, state.selectedNodeKey))),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardLinkVisible(link, state.group))),
		)))
	}
	return gosx.El("svg", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-workspace__paths"),
		gosx.Attr("viewBox", core.WorkbenchViewString(siteMapView, "workspaceCanvasViewBox")),
		gosx.Attr("aria-hidden", "true"),
		gosx.Attr("focusable", "false"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasWorkspaceCanvasLinks"))),
	), gosx.Fragment(children...))
}

func renderSiteMapBoardWorkspaceLayers(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	layers := core.WorkbenchViewMapList(siteMapView, "workspaceLayers")
	children := make([]gosx.Node, 0, len(layers))
	for _, layer := range layers {
		children = append(children, renderSiteMapBoardWorkspaceLayer(layer, state))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-workspace__layers"),
		gosx.Attr("aria-label", "Editable page composition lanes"),
	), gosx.Fragment(children...))
}

func renderSiteMapBoardWorkspaceLayer(layer map[string]any, state siteMapBoardState) gosx.Node {
	nodes := SiteMapMapList(layer, "nodes")
	nodeChildren := make([]gosx.Node, 0, len(nodes)*3)
	for _, node := range nodes {
		nodeChildren = append(nodeChildren, renderSiteMapBoardWorkspaceNode(node, state)...)
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(layer, "class")),
		gosx.Attr("data-studio-site-map-workspace-layer", core.WorkbenchViewString(layer, "key")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardLayerVisible(layer, state.focus))),
	),
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(layer, "label"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!core.WorkbenchViewBool(layer, "hasPageComposition")))), gosx.Text(core.WorkbenchViewString(layer, "summary"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(layer, "hasPageComposition")))), gosx.Text(core.WorkbenchViewString(layer, "pageRoute"))),
			),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "studio-site-map-workspace-layer__composition"),
				gosx.Attr("aria-label", "Page composition"),
				gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(layer, "hasPageComposition"))),
			),
				siteMapBoardFactSpan("Page type", core.WorkbenchViewString(layer, "pageComponentLabel")),
				siteMapBoardFactSpan("Sections", core.WorkbenchViewString(layer, "sectionCountLabel")),
				siteMapBoardFactSpan("Controls", core.WorkbenchViewString(layer, "controlCountLabel")),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(layer, "nodeCountLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-layer__nodes")), gosx.Fragment(nodeChildren...)),
	)
}

func renderSiteMapBoardWorkspaceNode(node map[string]any, state siteMapBoardState) []gosx.Node {
	selected := state.selectedNodeKey != "" && state.selectedNodeKey == core.WorkbenchViewString(node, "key")
	visible := siteMapBoardMatchesGroup(state.group, core.WorkbenchViewString(node, "group"))
	labelAttrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(node, "class")),
		gosx.Attr("for", core.WorkbenchViewString(node, "inputID")),
		gosx.Attr("data-studio-site-map-workspace-node", core.WorkbenchViewString(node, "key")),
		gosx.Attr("data-studio-site-map-node-kind", core.WorkbenchViewString(node, "kind")),
		gosx.Attr("data-studio-site-map-node-kind-label", core.WorkbenchViewString(node, "kindLabel")),
		gosx.Attr("data-studio-site-map-node-label", core.WorkbenchViewString(node, "label")),
		gosx.Attr("data-studio-site-map-node-summary", core.WorkbenchViewString(node, "summary")),
		gosx.Attr("data-studio-site-map-node-source-label", core.WorkbenchViewString(node, "sourceLabel")),
		gosx.Attr("data-studio-site-map-node-binding-label", core.WorkbenchViewString(node, "bindingLabel")),
		gosx.Attr("data-studio-site-map-node-status-label", core.WorkbenchViewString(node, "statusLabel")),
		gosx.Attr("data-studio-site-map-node-component-label", core.WorkbenchViewString(node, "componentLabel")),
		gosx.Attr("data-studio-site-map-group", core.WorkbenchViewString(node, "group")),
		gosx.Attr("data-studio-site-map-page", core.WorkbenchViewString(node, "pageKey")),
		gosx.Attr("data-studio-site-map-route", core.WorkbenchViewString(node, "route")),
		gosx.Attr("data-studio-site-map-source", core.WorkbenchViewString(node, "source")),
		gosx.Attr("data-studio-site-map-binding", core.WorkbenchViewString(node, "binding")),
		gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(node, "gosxComponent")),
		gosx.Attr("data-studio-site-map-node-selected", siteMapBoardBoolString(selected)),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(visible)),
	}
	// A node carrying numeric x/y is free-placed on the pan surface (the runtime
	// live-moves it on drag and snaps on release). Nodes WITHOUT a saved
	// position render exactly as before (lane flow), keeping the default view
	// and the reference editors visually unchanged.
	if x, y, ok := siteMapBoardNodePosition(node); ok {
		labelAttrs = append(labelAttrs,
			gosx.Attr("data-studio-site-map-node-x", workspaceCoord(x)),
			gosx.Attr("data-studio-site-map-node-y", workspaceCoord(y)),
			gosx.Attr("style", "position:absolute;left:"+workspaceCoord(x)+"px;top:"+workspaceCoord(y)+"px"),
		)
	}
	return []gosx.Node{
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-workspace-node__input"),
			gosx.Attr("id", core.WorkbenchViewString(node, "inputID")),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", core.WorkbenchViewString(node, "inputName")),
			gosx.Attr("value", core.WorkbenchViewString(node, "key")),
			gosx.Attr("aria-label", core.WorkbenchViewString(node, "label")),
			gosx.Attr("checked", selected),
		)),
		gosx.El("label", gosx.Attrs(labelAttrs...),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(node, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(node, "label"))),
			gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(node, "hasSummary")))), gosx.Text(core.WorkbenchViewString(node, "summary"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-node__facts")),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(node, "hasRoute")))), gosx.Text(core.WorkbenchViewString(node, "route"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(node, "hasSource")))), gosx.Text(core.WorkbenchViewString(node, "sourceLabel"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(node, "hasBindingLabel")))), gosx.Text(core.WorkbenchViewString(node, "bindingLabel"))),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(node, "statusLabel"))),
		),
		gosx.El("aside", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-workspace-node-card"),
			gosx.Attr("aria-label", "Selected site map item"),
			gosx.Attr("data-studio-site-map-workspace-node-card", core.WorkbenchViewString(node, "key")),
			gosx.Attr("data-studio-site-map-group", core.WorkbenchViewString(node, "group")),
			gosx.Attr("data-studio-site-map-node-selected", siteMapBoardBoolString(selected)),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(visible)),
		),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-node-card__main")),
				gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(node, "kindLabel"))),
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(node, "label"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(node, "hasSummary")))), gosx.Text(core.WorkbenchViewString(node, "summary"))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-node-card__facts")),
				siteMapBoardOptionalFact("Page", core.WorkbenchViewString(node, "route"), core.WorkbenchViewBool(node, "hasRoute")),
				siteMapBoardOptionalFact("Content", core.WorkbenchViewString(node, "sourceLabel"), core.WorkbenchViewBool(node, "hasSource")),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(node, "statusLabel"))),
			),
			renderSiteMapBoardSelectionActions("studio-site-map-workspace-node-card__actions", state.inspectorInputID),
		),
	}
}

func renderSiteMapBoardSelectionCard(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	return gosx.El("aside", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-selection-card"),
		gosx.Attr("aria-label", "Selected site map item"),
		gosx.Attr("data-studio-site-map-selection-card", "true"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasDefaultWorkspaceNode"))),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-selection-card__main")),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-kind", "true")), gosx.Text(state.selectedNodeKind)),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-label", "true")), gosx.Text(state.selectedNodeLabel)),
			gosx.El("small", gosx.Attrs(
				gosx.Attr("data-studio-site-map-selected-summary-row", "true"),
				gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(state.selectedNodeSummary != "")),
			), gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-summary", "true")), gosx.Text(state.selectedNodeSummary))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-selection-card__facts")),
			siteMapBoardSelectedFact("Page", "route", state.selectedNodeRoute),
			siteMapBoardSelectedFact("Content", "source", state.selectedNodeSource),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-status", "true")), gosx.Text(state.selectedNodeStatus)),
		),
		renderSiteMapBoardSelectionActions("studio-site-map-selection-card__actions", state.inspectorInputID),
	)
}

func renderSiteMapBoardSelectionActions(className, inspectorInputID string) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("aria-label", "Selected item actions"),
	),
		gosx.El("label", gosx.Attrs(
			gosx.Attr("for", inspectorInputID),
			gosx.Attr("role", "button"),
			gosx.Attr("tabindex", "0"),
		), gosx.Text("Edit fields")),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-site-map-detail-target", "build"),
		), gosx.Text("Add section")),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-site-map-detail-target", "sources"),
		), gosx.Text("Content")),
	)
}

func renderSiteMapBoardWorkspaceRail(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace__rail")),
		gosx.El("aside", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-workspace__selection"),
			gosx.Attr("aria-label", "Selected composition node"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasDefaultWorkspaceNode"))),
		),
			gosx.El("header", nil,
				gosx.El("div", nil,
					gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-kind", "true")), gosx.Text(state.selectedNodeKind)),
					gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-label", "true")), gosx.Text(state.selectedNodeLabel)),
				),
				gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-status", "true")), gosx.Text(state.selectedNodeStatus)),
			),
			gosx.El("p", gosx.Attrs(
				gosx.Attr("data-studio-site-map-selected-summary-row", "true"),
				gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(state.selectedNodeSummary != "")),
			), gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-summary", "true")), gosx.Text(state.selectedNodeSummary))),
			gosx.El("dl", nil,
				siteMapBoardSelectedDefinition("Route", "route", state.selectedNodeRoute),
				siteMapBoardSelectedDefinition("Component", "component", state.selectedNodeComponent),
				siteMapBoardSelectedDefinition("Source", "source", state.selectedNodeSource),
				siteMapBoardSelectedDefinition("Binding", "binding", state.selectedNodeBinding),
			),
		),
		renderSiteMapBoardConnections(siteMapView, state),
	)
}

func renderSiteMapBoardConnections(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	links := core.WorkbenchViewMapList(siteMapView, "workspaceLinks")
	children := make([]gosx.Node, 0, len(links))
	for _, link := range links {
		children = append(children, gosx.El("span", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(link, "class")),
			gosx.Attr("data-studio-site-map-workspace-link", core.WorkbenchViewString(link, "key")),
			gosx.Attr("data-studio-site-map-link-kind", core.WorkbenchViewString(link, "kind")),
			gosx.Attr("data-studio-site-map-link-from", core.WorkbenchViewString(link, "fromNodeKey")),
			gosx.Attr("data-studio-site-map-link-to", core.WorkbenchViewString(link, "toNodeKey")),
			gosx.Attr("data-studio-site-map-link-from-page", core.WorkbenchViewString(link, "fromPageKey")),
			gosx.Attr("data-studio-site-map-link-to-page", core.WorkbenchViewString(link, "toPageKey")),
			gosx.Attr("data-studio-site-map-link-from-group", core.WorkbenchViewString(link, "fromGroup")),
			gosx.Attr("data-studio-site-map-link-to-group", core.WorkbenchViewString(link, "toGroup")),
			gosx.Attr("data-studio-site-map-link-selected", siteMapBoardBoolString(siteMapBoardLinkSelected(link, state.selectedNodeKey))),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardLinkVisible(link, state.group))),
		),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(link, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(link, "fromLabel"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(link, "label"))),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(link, "toLabel"))),
		))
	}
	return gosx.El("aside", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-workspace__connections"),
		gosx.Attr("aria-label", "Composition connections"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasWorkspaceLinks"))),
	),
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text("Connections")),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "workspaceLinkCountLabel"))),
		),
		gosx.El("div", nil, gosx.Fragment(children...)),
	)
}

func renderSiteMapBoardBuilder(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-builder"),
		gosx.Attr("data-studio-site-map-builder", "true"),
		gosx.Attr("data-studio-site-map-panel-visible", siteMapBoardBoolString(state.detail == "build")),
	),
		renderSiteMapBoardIntents(siteMapView, state),
		renderSiteMapBoardBlueprints(siteMapView, state),
		renderSiteMapBoardPalette(siteMapView, state),
	)
}

func renderSiteMapBoardIntents(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	intents := core.WorkbenchViewMapList(siteMapView, "compositionIntents")
	children := make([]gosx.Node, 0, len(intents))
	for _, intent := range intents {
		children = append(children, renderSiteMapBoardIntent(intent, state))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-intents"),
		gosx.Attr("data-studio-composition-intents", "true"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasCompositionIntents"))),
	),
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text("Ready to insert")),
				gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "selectedPageLabel"))),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "activeCompositionIntentCountLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-intents__grid")), gosx.Fragment(children...)),
	)
}

func renderSiteMapBoardIntent(intent map[string]any, state siteMapBoardState) gosx.Node {
	steps := SiteMapMapList(intent, "steps")
	stepChildren := make([]gosx.Node, 0, len(steps))
	for _, step := range steps {
		stepChildren = append(stepChildren, gosx.El("li", gosx.Attrs(
			gosx.Attr("data-studio-composition-step", core.WorkbenchViewString(step, "key")),
			gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(step, "gosxComponent")),
			gosx.Attr("data-studio-composition-binding", core.WorkbenchViewString(step, "binding")),
		),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(step, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(step, "summary"))),
		))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(intent, "class")),
		gosx.Attr("data-studio-composition-intent", core.WorkbenchViewString(intent, "key")),
		gosx.Attr("data-studio-composition-kind", core.WorkbenchViewString(intent, "kind")),
		gosx.Attr("data-studio-composition-target", core.WorkbenchViewString(intent, "targetPageKey")),
		gosx.Attr("data-studio-composition-route", core.WorkbenchViewString(intent, "targetRoute")),
		gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(intent, "gosxComponent")),
		gosx.Attr("data-studio-composition-binding", core.WorkbenchViewString(intent, "binding")),
		gosx.Attr("data-studio-composition-blueprint", core.WorkbenchViewString(intent, "pageBlueprintKey")),
		gosx.Attr("data-studio-composition-template", core.WorkbenchViewString(intent, "componentTemplateKey")),
		gosx.Attr("data-studio-composition-active", siteMapBoardBoolString(siteMapBoardIntentActive(intent, state))),
	),
		gosx.El("header", nil,
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(intent, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(intent, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(intent, "summary"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-intent__facts")),
			siteMapBoardFactSpan("Target", core.WorkbenchViewString(intent, "targetLabel")),
			siteMapBoardFactSpan("Route", core.WorkbenchViewString(intent, "targetRoute")),
			siteMapBoardFactSpan("Scope", core.WorkbenchViewString(intent, "targetRegion")),
		),
		gosx.El("ol", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-intent__steps"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(intent, "hasSteps"))),
		), gosx.Fragment(stepChildren...)),
		gosx.El("footer", nil,
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(intent, "statusLabel"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(intent, "stepCountLabel"))),
		),
	)
}

func renderSiteMapBoardBlueprints(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	blueprints := core.WorkbenchViewMapList(siteMapView, "blueprints")
	children := make([]gosx.Node, 0, len(blueprints))
	for _, blueprint := range blueprints {
		children = append(children, renderSiteMapBoardBlueprint(blueprint, state))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-builder__panel"),
		gosx.Attr("data-studio-site-map-blueprints", "true"),
	),
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text("New page starters")),
				gosx.El("small", nil, gosx.Text("Start a page from a reusable structure.")),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "blueprintCountLabel"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!core.WorkbenchViewBool(siteMapView, "hasBlueprints"))),
		), gosx.Text("No page blueprints are configured.")),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-blueprints"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasBlueprints"))),
		), gosx.Fragment(children...)),
	)
}

func renderSiteMapBoardBlueprint(blueprint map[string]any, state siteMapBoardState) gosx.Node {
	components := SiteMapMapList(blueprint, "components")
	componentChildren := make([]gosx.Node, 0, len(components))
	for _, component := range components {
		componentChildren = append(componentChildren, gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(component, "gosxComponent"))), gosx.Text(core.WorkbenchViewString(component, "label"))))
	}
	selected := state.selectedBlueprintKey != "" && state.selectedBlueprintKey == core.WorkbenchViewString(blueprint, "key")
	return gosx.El("label", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-blueprint"),
		gosx.Attr("data-studio-site-map-blueprint", core.WorkbenchViewString(blueprint, "key")),
		gosx.Attr("data-studio-site-map-route-pattern", core.WorkbenchViewString(blueprint, "routePattern")),
		gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(blueprint, "gosxComponent")),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-builder__input"),
			gosx.Attr("id", core.WorkbenchViewString(blueprint, "inputID")),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", core.WorkbenchViewString(blueprint, "inputName")),
			gosx.Attr("value", core.WorkbenchViewString(blueprint, "key")),
			gosx.Attr("checked", selected),
			gosx.Attr("aria-label", core.WorkbenchViewString(blueprint, "label")),
		)),
		gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(blueprint, "label"))),
		gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(blueprint, "summary"))),
		gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(blueprint, "componentCountLabel"))),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-blueprint__plan"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(blueprint, "hasComponents"))),
		),
			gosx.El("strong", nil, gosx.Text("Page structure")),
			gosx.Fragment(componentChildren...),
		),
	)
}

func renderSiteMapBoardPalette(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	templates := core.WorkbenchViewMapList(siteMapView, "palette")
	children := make([]gosx.Node, 0, len(templates))
	for _, template := range templates {
		children = append(children, renderSiteMapBoardTemplate(template, state))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-builder__panel"),
		gosx.Attr("data-studio-site-map-palette", "true"),
	),
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text("Add section")),
				gosx.El("small", nil, gosx.Text("Pick a section to place on the selected page.")),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "paletteCountLabel"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!core.WorkbenchViewBool(siteMapView, "hasPalette"))),
		), gosx.Text("No building blocks are configured.")),
		renderSiteMapBoardChoiceGroup("studio-site-map-palette-filter", "toolbar", "Building block categories", "data-studio-site-map-palette-control", state.palette, []siteMapBoardChoice{
			{Label: "All", Key: "all"},
			{Label: "Sections", Key: "page-sections"},
			{Label: "Content", Key: "content"},
			{Label: "Store", Key: "store"},
			{Label: "Media", Key: "media"},
			{Label: "Flows", Key: "flows"},
			{Label: "Plugins", Key: "plugins"},
		}),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-palette"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(siteMapView, "hasPalette"))),
		), gosx.Fragment(children...)),
	)
}

func renderSiteMapBoardTemplate(template map[string]any, state siteMapBoardState) gosx.Node {
	controls := SiteMapMapList(template, "controls")
	controlChildren := make([]gosx.Node, 0, len(controls))
	for _, control := range controls {
		controlChildren = append(controlChildren, gosx.El("small", gosx.Attrs(
			gosx.Attr("data-studio-site-map-control", core.WorkbenchViewString(control, "key")),
			gosx.Attr("data-studio-site-map-control-kind", core.WorkbenchViewString(control, "kind")),
		), gosx.Text(core.WorkbenchViewString(control, "label"))))
	}
	selected := state.selectedTemplateKey != "" && state.selectedTemplateKey == core.WorkbenchViewString(template, "key")
	isShared := core.WorkbenchViewBool(template, "isShared")
	metaChildren := []gosx.Node{
		gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(template, "statusLabel"))),
		gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(template, "controlLabel"))),
	}
	// A shared ComponentDefinition entry (AuthoringSiteMapComponentDefinitionPaletteViews)
	// additionally shows how many instances are currently attached, so the
	// browse/filter board matches what the real "place another instance"
	// action (renderSiteMapSharedComponentPalettePanel) reports.
	if isShared {
		metaChildren = append(metaChildren, gosx.El("small", gosx.Attrs(
			gosx.Attr("data-studio-shared-component-instance-count", "true"),
		), gosx.Text(core.WorkbenchViewString(template, "instanceCountLabel"))))
	}
	cardAttrs := []any{
		gosx.Attr("class", "studio-site-map-palette-card"),
		gosx.Attr("data-studio-site-map-component-template", core.WorkbenchViewString(template, "key")),
		gosx.Attr("data-studio-site-map-template-category", core.WorkbenchViewString(template, "categoryKey")),
		gosx.Attr("data-studio-site-map-template-source", core.WorkbenchViewString(template, "source")),
		gosx.Attr("data-studio-site-map-default-binding", core.WorkbenchViewString(template, "defaultBinding")),
		gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(template, "gosxComponent")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardMatchesPalette(state.palette, core.WorkbenchViewString(template, "categoryKey")))),
	}
	if isShared {
		cardAttrs = append(cardAttrs, gosx.Attr("data-studio-shared-component-definition", core.WorkbenchViewString(template, "definitionKey")))
	}
	return gosx.El("label", gosx.Attrs(cardAttrs...),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-builder__input"),
			gosx.Attr("id", core.WorkbenchViewString(template, "inputID")),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", core.WorkbenchViewString(template, "inputName")),
			gosx.Attr("value", core.WorkbenchViewString(template, "key")),
			gosx.Attr("checked", selected),
			gosx.Attr("aria-label", core.WorkbenchViewString(template, "label")),
		)),
		gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(template, "label"))),
		gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(template, "summary"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-palette-card__meta")), gosx.Fragment(metaChildren...)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-palette-card__plan"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(template, "hasControls"))),
		),
			gosx.El("strong", nil, gosx.Text("No-code controls")),
			gosx.Fragment(controlChildren...),
		),
		gosx.El("strong", gosx.Attrs(gosx.Attr("class", "studio-site-map-palette-card__add")), gosx.Text(core.WorkbenchViewString(template, "addLabel"))),
	)
}

func renderSiteMapBoardViewport(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	pages := core.WorkbenchViewMapList(siteMapView, "pages")
	children := make([]gosx.Node, 0, len(pages))
	for _, page := range pages {
		children = append(children, renderSiteMapBoardPage(page, state))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-engine__viewport"),
		gosx.Attr("data-studio-site-map-viewport", "true"),
		gosx.Attr("aria-label", "Editable site map"),
		gosx.Attr("data-studio-site-map-panel-visible", siteMapBoardBoolString(state.detail != "map" && state.detail != "build")),
	), gosx.Fragment(children...))
}

func renderSiteMapBoardPage(page map[string]any, state siteMapBoardState) gosx.Node {
	components := SiteMapMapList(page, "components")
	componentNodes := make([]gosx.Node, 0, len(components))
	sourceNodes := make([]gosx.Node, 0, len(components))
	controlNodes := make([]gosx.Node, 0, len(components))
	for _, component := range components {
		componentNodes = append(componentNodes, renderSiteMapBoardComponent(component))
		sourceNodes = append(sourceNodes, renderSiteMapBoardSourceRow(component))
		controlNodes = append(controlNodes, renderSiteMapBoardControlCard(component))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(page, "class")),
		gosx.Attr("data-studio-site-map-page", core.WorkbenchViewString(page, "key")),
		gosx.Attr("data-studio-site-map-route", core.WorkbenchViewString(page, "route")),
		gosx.Attr("data-studio-site-map-group", core.WorkbenchViewString(page, "group")),
		gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(page, "gosxComponent")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardMatchesGroup(state.group, core.WorkbenchViewString(page, "group")))),
	),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-site-map-page__head")),
			gosx.El("label", gosx.Attrs(
				gosx.Attr("class", "studio-site-map-page__select"),
				gosx.Attr("data-studio-site-map-select", core.WorkbenchViewString(page, "key")),
			),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("class", "studio-site-map-page__input"),
					gosx.Attr("type", "radio"),
					gosx.Attr("name", "studioSiteMapPage"),
					gosx.Attr("value", core.WorkbenchViewString(page, "key")),
					gosx.Attr("checked", core.WorkbenchViewBool(page, "selected")),
					gosx.Attr("aria-label", core.WorkbenchViewString(page, "label")),
				)),
				gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(page, "label"))),
				gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(page, "route"))),
			),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("href", core.WorkbenchViewString(page, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text("Open")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-page__meta")),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(page, "statusLabel"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(page, "typeLabel"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(page, "groupLabel"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-components"),
			gosx.Attr("aria-label", core.WorkbenchViewString(page, "componentCountLabel")),
		), gosx.Fragment(componentNodes...)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-source-stack"),
			gosx.Attr("aria-label", core.WorkbenchViewString(page, "componentCountLabel")),
		), gosx.Fragment(sourceNodes...)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-control-stack"),
			gosx.Attr("aria-label", core.WorkbenchViewString(page, "controlCountLabel")),
		), gosx.Fragment(controlNodes...)),
		gosx.El("footer", gosx.Attrs(gosx.Attr("class", "studio-site-map-page__foot")),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(page, "componentCountLabel"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(page, "controlCountLabel"))),
		),
	)
}

func renderSiteMapBoardComponent(component map[string]any) gosx.Node {
	return gosx.El("label", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-component"),
		gosx.Attr("data-studio-site-map-component", core.WorkbenchViewString(component, "key")),
		gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(component, "gosxComponent")),
		gosx.Attr("data-studio-site-map-source", core.WorkbenchViewString(component, "source")),
		gosx.Attr("data-studio-site-map-binding", core.WorkbenchViewString(component, "binding")),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-component__input"),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", "studioSiteMapComponent"),
			gosx.Attr("value", core.WorkbenchViewString(component, "selectionKey")),
			gosx.Attr("aria-label", core.WorkbenchViewString(component, "label")),
		)),
		gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(component, "label"))),
		gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(component, "sourceLabel"))),
		gosx.El("small", gosx.Attrs(gosx.Attr("class", "studio-site-map-component__binding")), gosx.Text(core.WorkbenchViewString(component, "summary"))),
		gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(component, "statusLabel"))),
		renderSiteMapBoardControls(component),
	)
}

func renderSiteMapBoardSourceRow(component map[string]any) gosx.Node {
	return gosx.El("label", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-source-row"),
		gosx.Attr("data-studio-site-map-component", core.WorkbenchViewString(component, "key")),
		gosx.Attr("data-studio-gosx-component", core.WorkbenchViewString(component, "gosxComponent")),
		gosx.Attr("data-studio-site-map-source", core.WorkbenchViewString(component, "source")),
		gosx.Attr("data-studio-site-map-binding", core.WorkbenchViewString(component, "binding")),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-component__input"),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", "studioSiteMapComponent"),
			gosx.Attr("value", core.WorkbenchViewString(component, "selectionKey")),
			gosx.Attr("aria-label", core.WorkbenchViewString(component, "label")),
		)),
		gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(component, "sourceLabel"))),
		gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(component, "sourceSummary"))),
		gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(component, "summary"))),
		gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(component, "controlLabel"))),
	)
}

func renderSiteMapBoardControlCard(component map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-control-card"),
		gosx.Attr("data-studio-site-map-component", core.WorkbenchViewString(component, "key")),
	),
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(component, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(component, "controlLabel"))),
		),
		renderSiteMapBoardControls(component),
	)
}

func renderSiteMapBoardControls(component map[string]any) gosx.Node {
	controls := SiteMapMapList(component, "controls")
	children := make([]gosx.Node, 0, len(controls))
	for _, control := range controls {
		children = append(children, gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-control"),
			gosx.Attr("data-studio-site-map-control", core.WorkbenchViewString(control, "key")),
			gosx.Attr("data-studio-site-map-control-kind", core.WorkbenchViewString(control, "kind")),
			gosx.Attr("data-gosx-studio-authoring-operation", core.WorkbenchViewString(control, "authoringOperation")),
			gosx.Attr("data-gosx-studio-authoring-page", core.WorkbenchViewString(control, "authoringPageKey")),
			gosx.Attr("data-gosx-studio-authoring-component", core.WorkbenchViewString(control, "authoringComponent")),
			gosx.Attr("data-gosx-studio-authoring-control", core.WorkbenchViewString(control, "authoringControl")),
			gosx.Attr("data-gosx-studio-authoring-binding", core.WorkbenchViewString(control, "authoringBinding")),
		),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(control, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(control, "kindLabel"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-controls"),
		gosx.Attr("aria-label", core.WorkbenchViewString(component, "controlLabel")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(core.WorkbenchViewBool(component, "hasControls"))),
	), gosx.Fragment(children...))
}

func renderSiteMapBoardMinimap(siteMapView map[string]any) gosx.Node {
	return gosx.El("aside", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-minimap"),
		gosx.Attr("aria-label", "Site map overview"),
	),
		gosx.El("strong", nil, gosx.Text("Board")),
		gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "pageCountLabel"))),
		gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "componentCountLabel"))),
		gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "controlCountLabel"))),
		gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(siteMapView, "selectedPageLabel"))),
	)
}

func siteMapBoardFactSpan(label, value string) gosx.Node {
	return gosx.El("span", nil,
		gosx.El("small", nil, gosx.Text(label)),
		gosx.El("strong", nil, gosx.Text(value)),
	)
}

func siteMapBoardOptionalFact(label, value string, visible bool) gosx.Node {
	return gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(visible))),
		gosx.El("small", nil, gosx.Text(label)),
		gosx.El("strong", nil, gosx.Text(value)),
	)
}

func siteMapBoardSelectedFact(label, kind, value string) gosx.Node {
	return gosx.El("span", gosx.Attrs(
		gosx.Attr("data-studio-site-map-selected-"+kind+"-row", "true"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(value != "")),
	),
		gosx.El("small", nil, gosx.Text(label)),
		gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-"+kind, "true")), gosx.Text(value)),
	)
}

func siteMapBoardSelectedDefinition(label, kind, value string) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("data-studio-site-map-selected-"+kind+"-row", "true"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(value != "")),
	),
		gosx.El("dt", nil, gosx.Text(label)),
		gosx.El("dd", gosx.Attrs(gosx.Attr("data-studio-site-map-selected-"+kind, "true")), gosx.Text(value)),
	)
}

func siteMapBoardBoolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// siteMapBoardNodePosition reports a node's saved canvas position. It returns
// ok=true only when BOTH x and y resolve to finite numbers; today most node
// maps carry no coordinates (ok=false) and render in lane flow.
func siteMapBoardNodePosition(node map[string]any) (x, y float64, ok bool) {
	x, xok := siteMapBoardNodeCoord(node, "x")
	y, yok := siteMapBoardNodeCoord(node, "y")
	return x, y, xok && yok
}

func siteMapBoardNodeCoord(node map[string]any, key string) (float64, bool) {
	if node == nil {
		return 0, false
	}
	switch typed := node[key].(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		if value, err := typed.Float64(); err == nil {
			return value, true
		}
	case string:
		if value, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return value, true
		}
	}
	return 0, false
}

func siteMapBoardMatchesGroup(group, candidate string) bool {
	group = strings.TrimSpace(group)
	return group == "" || group == "all" || strings.TrimSpace(candidate) == group
}

func siteMapBoardMatchesPalette(palette, candidate string) bool {
	palette = strings.TrimSpace(palette)
	return palette == "" || palette == "all" || strings.TrimSpace(candidate) == palette
}

func siteMapBoardLayerVisible(layer map[string]any, focus string) bool {
	key := core.WorkbenchViewString(layer, "key")
	focus = strings.TrimSpace(focus)
	return focus == "" || focus == "all" || key == focus || key == "resources"
}

func siteMapBoardLinkVisible(link map[string]any, group string) bool {
	group = strings.TrimSpace(group)
	if group == "" || group == "all" {
		return true
	}
	return core.WorkbenchViewString(link, "fromGroup") == group || core.WorkbenchViewString(link, "toGroup") == group
}

func siteMapBoardLinkSelected(link map[string]any, selectedNodeKey string) bool {
	return selectedNodeKey != "" && (core.WorkbenchViewString(link, "fromNodeKey") == selectedNodeKey || core.WorkbenchViewString(link, "toNodeKey") == selectedNodeKey)
}

func siteMapBoardIntentActive(intent map[string]any, state siteMapBoardState) bool {
	switch core.WorkbenchViewString(intent, "kind") {
	case "create-page":
		return state.selectedBlueprintKey != "" && state.selectedBlueprintKey == core.WorkbenchViewString(intent, "pageBlueprintKey")
	case "add-component":
		return state.selectedTemplateKey != "" && state.selectedTemplateKey == core.WorkbenchViewString(intent, "componentTemplateKey")
	default:
		return false
	}
}
