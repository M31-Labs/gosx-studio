package studio

import (
	"encoding/json"
	"strconv"
	"strings"

	"m31labs.dev/gosx"
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
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!workbenchViewBool(siteMapView, "hasPages"))),
		), gosx.Text(workbenchViewString(siteMapView, "empty"))),
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
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasPages"))),
	), gosx.Fragment(bodyChildren...)))

	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.Class, "studio-site-map-board")),
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
		group:                 FirstNonEmpty(options.DefaultGroupKey, workbenchViewString(siteMapView, "defaultGroupKey"), "all"),
		detail:                FirstNonEmpty(options.DefaultDetailKey, workbenchViewString(siteMapView, "defaultDetailKey"), "map"),
		palette:               FirstNonEmpty(options.DefaultPaletteKey, "all"),
		focus:                 FirstNonEmpty(options.DefaultFocusKey, workbenchViewString(siteMapView, "defaultFocusKey"), "all"),
		zoom:                  FirstNonEmpty(options.DefaultZoomKey, "fit"),
		density:               FirstNonEmpty(options.DefaultDensityKey, "roomy"),
		grid:                  FirstNonEmpty(options.DefaultGridKey, "on"),
		selectedBlueprintKey:  workbenchViewString(siteMapView, "defaultBlueprintKey"),
		selectedTemplateKey:   workbenchViewString(siteMapView, "defaultTemplateKey"),
		selectedNodeKey:       workbenchViewString(siteMapView, "defaultWorkspaceNodeKey"),
		selectedNodeLabel:     workbenchViewString(siteMapView, "defaultWorkspaceNodeLabel"),
		selectedNodeKind:      workbenchViewString(siteMapView, "defaultWorkspaceNodeKind"),
		selectedNodeSummary:   workbenchViewString(siteMapView, "defaultWorkspaceNodeSummary"),
		selectedNodeRoute:     workbenchViewString(siteMapView, "defaultWorkspaceNodeRoute"),
		selectedNodeSource:    workbenchViewString(siteMapView, "defaultWorkspaceNodeSource"),
		selectedNodeBinding:   workbenchViewString(siteMapView, "defaultWorkspaceNodeBinding"),
		selectedNodeStatus:    workbenchViewString(siteMapView, "defaultWorkspaceNodeStatus"),
		selectedNodeComponent: workbenchViewString(siteMapView, "defaultWorkspaceNodeComponentLabel"),
		inspectorInputID:      FirstNonEmpty(options.InspectorInputID, "studioSiteMapInspectorFocus"),
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
	pages := workbenchViewMapList(siteMapView, "pages")
	choices := make([]siteMapBoardChoice, 0, len(pages))
	for _, page := range pages {
		key := workbenchMapString(page, "key")
		if key == "" {
			continue
		}
		choices = append(choices, siteMapBoardChoice{Label: workbenchMapString(page, "label"), Key: key})
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
				gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "workspaceLayerCountLabel"))),
				gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "workspaceNodeCountLabel"))),
				gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "workspaceLinkCountLabel"))),
			),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-workspace__body"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasWorkspace"))),
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
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!workbenchViewBool(siteMapView, "hasWorkspace"))),
		), gosx.Text("No composition workspace is configured.")),
	)
}

func renderSiteMapBoardCanvasPaths(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	paths := workbenchViewMapList(siteMapView, "workspaceCanvasLinks")
	children := make([]gosx.Node, 0, len(paths))
	for _, link := range paths {
		children = append(children, gosx.El("path", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(link, "class")),
			gosx.Attr("d", workbenchMapString(link, "path")),
			gosx.Attr("data-studio-site-map-workspace-path", workbenchMapString(link, "key")),
			gosx.Attr("data-studio-site-map-link-kind", workbenchMapString(link, "kind")),
			gosx.Attr("data-studio-site-map-link-from", workbenchMapString(link, "fromNodeKey")),
			gosx.Attr("data-studio-site-map-link-to", workbenchMapString(link, "toNodeKey")),
			gosx.Attr("data-studio-site-map-link-from-page", workbenchMapString(link, "fromPageKey")),
			gosx.Attr("data-studio-site-map-link-to-page", workbenchMapString(link, "toPageKey")),
			gosx.Attr("data-studio-site-map-link-from-group", workbenchMapString(link, "fromGroup")),
			gosx.Attr("data-studio-site-map-link-to-group", workbenchMapString(link, "toGroup")),
			gosx.Attr("data-studio-site-map-link-selected", siteMapBoardBoolString(siteMapBoardLinkSelected(link, state.selectedNodeKey))),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardLinkVisible(link, state.group))),
		)))
	}
	return gosx.El("svg", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-workspace__paths"),
		gosx.Attr("viewBox", workbenchViewString(siteMapView, "workspaceCanvasViewBox")),
		gosx.Attr("aria-hidden", "true"),
		gosx.Attr("focusable", "false"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasWorkspaceCanvasLinks"))),
	), gosx.Fragment(children...))
}

func renderSiteMapBoardWorkspaceLayers(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	layers := workbenchViewMapList(siteMapView, "workspaceLayers")
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
	nodes := siteMapMapList(layer, "nodes")
	nodeChildren := make([]gosx.Node, 0, len(nodes)*3)
	for _, node := range nodes {
		nodeChildren = append(nodeChildren, renderSiteMapBoardWorkspaceNode(node, state)...)
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(layer, "class")),
		gosx.Attr("data-studio-site-map-workspace-layer", workbenchMapString(layer, "key")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardLayerVisible(layer, state.focus))),
	),
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(layer, "label"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!workbenchMapBool(layer, "hasPageComposition")))), gosx.Text(workbenchMapString(layer, "summary"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(layer, "hasPageComposition")))), gosx.Text(workbenchMapString(layer, "pageRoute"))),
			),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "studio-site-map-workspace-layer__composition"),
				gosx.Attr("aria-label", "Page composition"),
				gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(layer, "hasPageComposition"))),
			),
				siteMapBoardFactSpan("Page type", workbenchMapString(layer, "pageComponentLabel")),
				siteMapBoardFactSpan("Sections", workbenchMapString(layer, "sectionCountLabel")),
				siteMapBoardFactSpan("Controls", workbenchMapString(layer, "controlCountLabel")),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(layer, "nodeCountLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-layer__nodes")), gosx.Fragment(nodeChildren...)),
	)
}

func renderSiteMapBoardWorkspaceNode(node map[string]any, state siteMapBoardState) []gosx.Node {
	selected := state.selectedNodeKey != "" && state.selectedNodeKey == workbenchMapString(node, "key")
	visible := siteMapBoardMatchesGroup(state.group, workbenchMapString(node, "group"))
	labelAttrs := []any{
		gosx.Attr("class", workbenchMapString(node, "class")),
		gosx.Attr("for", workbenchMapString(node, "inputID")),
		gosx.Attr("data-studio-site-map-workspace-node", workbenchMapString(node, "key")),
		gosx.Attr("data-studio-site-map-node-kind", workbenchMapString(node, "kind")),
		gosx.Attr("data-studio-site-map-node-kind-label", workbenchMapString(node, "kindLabel")),
		gosx.Attr("data-studio-site-map-node-label", workbenchMapString(node, "label")),
		gosx.Attr("data-studio-site-map-node-summary", workbenchMapString(node, "summary")),
		gosx.Attr("data-studio-site-map-node-source-label", workbenchMapString(node, "sourceLabel")),
		gosx.Attr("data-studio-site-map-node-binding-label", workbenchMapString(node, "bindingLabel")),
		gosx.Attr("data-studio-site-map-node-status-label", workbenchMapString(node, "statusLabel")),
		gosx.Attr("data-studio-site-map-node-component-label", workbenchMapString(node, "componentLabel")),
		gosx.Attr("data-studio-site-map-group", workbenchMapString(node, "group")),
		gosx.Attr("data-studio-site-map-page", workbenchMapString(node, "pageKey")),
		gosx.Attr("data-studio-site-map-route", workbenchMapString(node, "route")),
		gosx.Attr("data-studio-site-map-source", workbenchMapString(node, "source")),
		gosx.Attr("data-studio-site-map-binding", workbenchMapString(node, "binding")),
		gosx.Attr("data-studio-gosx-component", workbenchMapString(node, "gosxComponent")),
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
			gosx.Attr("id", workbenchMapString(node, "inputID")),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", workbenchMapString(node, "inputName")),
			gosx.Attr("value", workbenchMapString(node, "key")),
			gosx.Attr("aria-label", workbenchMapString(node, "label")),
			gosx.Attr("checked", selected),
		)),
		gosx.El("label", gosx.Attrs(labelAttrs...),
			gosx.El("span", nil, gosx.Text(workbenchMapString(node, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(node, "label"))),
			gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(node, "hasSummary")))), gosx.Text(workbenchMapString(node, "summary"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-node__facts")),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(node, "hasRoute")))), gosx.Text(workbenchMapString(node, "route"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(node, "hasSource")))), gosx.Text(workbenchMapString(node, "sourceLabel"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(node, "hasBindingLabel")))), gosx.Text(workbenchMapString(node, "bindingLabel"))),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(node, "statusLabel"))),
		),
		gosx.El("aside", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-workspace-node-card"),
			gosx.Attr("aria-label", "Selected site map item"),
			gosx.Attr("data-studio-site-map-workspace-node-card", workbenchMapString(node, "key")),
			gosx.Attr("data-studio-site-map-group", workbenchMapString(node, "group")),
			gosx.Attr("data-studio-site-map-node-selected", siteMapBoardBoolString(selected)),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(visible)),
		),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-node-card__main")),
				gosx.El("span", nil, gosx.Text(workbenchMapString(node, "kindLabel"))),
				gosx.El("strong", nil, gosx.Text(workbenchMapString(node, "label"))),
				gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(node, "hasSummary")))), gosx.Text(workbenchMapString(node, "summary"))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-workspace-node-card__facts")),
				siteMapBoardOptionalFact("Page", workbenchMapString(node, "route"), workbenchMapBool(node, "hasRoute")),
				siteMapBoardOptionalFact("Content", workbenchMapString(node, "sourceLabel"), workbenchMapBool(node, "hasSource")),
				gosx.El("output", nil, gosx.Text(workbenchMapString(node, "statusLabel"))),
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
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasDefaultWorkspaceNode"))),
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
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasDefaultWorkspaceNode"))),
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
	links := workbenchViewMapList(siteMapView, "workspaceLinks")
	children := make([]gosx.Node, 0, len(links))
	for _, link := range links {
		children = append(children, gosx.El("span", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(link, "class")),
			gosx.Attr("data-studio-site-map-workspace-link", workbenchMapString(link, "key")),
			gosx.Attr("data-studio-site-map-link-kind", workbenchMapString(link, "kind")),
			gosx.Attr("data-studio-site-map-link-from", workbenchMapString(link, "fromNodeKey")),
			gosx.Attr("data-studio-site-map-link-to", workbenchMapString(link, "toNodeKey")),
			gosx.Attr("data-studio-site-map-link-from-page", workbenchMapString(link, "fromPageKey")),
			gosx.Attr("data-studio-site-map-link-to-page", workbenchMapString(link, "toPageKey")),
			gosx.Attr("data-studio-site-map-link-from-group", workbenchMapString(link, "fromGroup")),
			gosx.Attr("data-studio-site-map-link-to-group", workbenchMapString(link, "toGroup")),
			gosx.Attr("data-studio-site-map-link-selected", siteMapBoardBoolString(siteMapBoardLinkSelected(link, state.selectedNodeKey))),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardLinkVisible(link, state.group))),
		),
			gosx.El("small", nil, gosx.Text(workbenchMapString(link, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(link, "fromLabel"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(link, "label"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(link, "toLabel"))),
		))
	}
	return gosx.El("aside", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-workspace__connections"),
		gosx.Attr("aria-label", "Composition connections"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasWorkspaceLinks"))),
	),
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text("Connections")),
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "workspaceLinkCountLabel"))),
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
	intents := workbenchViewMapList(siteMapView, "compositionIntents")
	children := make([]gosx.Node, 0, len(intents))
	for _, intent := range intents {
		children = append(children, renderSiteMapBoardIntent(intent, state))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-intents"),
		gosx.Attr("data-studio-composition-intents", "true"),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasCompositionIntents"))),
	),
		gosx.El("header", nil,
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text("Ready to insert")),
				gosx.El("small", nil, gosx.Text(workbenchViewString(siteMapView, "selectedPageLabel"))),
			),
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "activeCompositionIntentCountLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-intents__grid")), gosx.Fragment(children...)),
	)
}

func renderSiteMapBoardIntent(intent map[string]any, state siteMapBoardState) gosx.Node {
	steps := siteMapMapList(intent, "steps")
	stepChildren := make([]gosx.Node, 0, len(steps))
	for _, step := range steps {
		stepChildren = append(stepChildren, gosx.El("li", gosx.Attrs(
			gosx.Attr("data-studio-composition-step", workbenchMapString(step, "key")),
			gosx.Attr("data-studio-gosx-component", workbenchMapString(step, "gosxComponent")),
			gosx.Attr("data-studio-composition-binding", workbenchMapString(step, "binding")),
		),
			gosx.El("span", nil, gosx.Text(workbenchMapString(step, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(step, "summary"))),
		))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(intent, "class")),
		gosx.Attr("data-studio-composition-intent", workbenchMapString(intent, "key")),
		gosx.Attr("data-studio-composition-kind", workbenchMapString(intent, "kind")),
		gosx.Attr("data-studio-composition-target", workbenchMapString(intent, "targetPageKey")),
		gosx.Attr("data-studio-composition-route", workbenchMapString(intent, "targetRoute")),
		gosx.Attr("data-studio-gosx-component", workbenchMapString(intent, "gosxComponent")),
		gosx.Attr("data-studio-composition-binding", workbenchMapString(intent, "binding")),
		gosx.Attr("data-studio-composition-blueprint", workbenchMapString(intent, "pageBlueprintKey")),
		gosx.Attr("data-studio-composition-template", workbenchMapString(intent, "componentTemplateKey")),
		gosx.Attr("data-studio-composition-active", siteMapBoardBoolString(siteMapBoardIntentActive(intent, state))),
	),
		gosx.El("header", nil,
			gosx.El("span", nil, gosx.Text(workbenchMapString(intent, "kindLabel"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(intent, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(intent, "summary"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-intent__facts")),
			siteMapBoardFactSpan("Target", workbenchMapString(intent, "targetLabel")),
			siteMapBoardFactSpan("Route", workbenchMapString(intent, "targetRoute")),
			siteMapBoardFactSpan("Scope", workbenchMapString(intent, "targetRegion")),
		),
		gosx.El("ol", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-intent__steps"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(intent, "hasSteps"))),
		), gosx.Fragment(stepChildren...)),
		gosx.El("footer", nil,
			gosx.El("output", nil, gosx.Text(workbenchMapString(intent, "statusLabel"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(intent, "stepCountLabel"))),
		),
	)
}

func renderSiteMapBoardBlueprints(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	blueprints := workbenchViewMapList(siteMapView, "blueprints")
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
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "blueprintCountLabel"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!workbenchViewBool(siteMapView, "hasBlueprints"))),
		), gosx.Text("No page blueprints are configured.")),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-blueprints"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasBlueprints"))),
		), gosx.Fragment(children...)),
	)
}

func renderSiteMapBoardBlueprint(blueprint map[string]any, state siteMapBoardState) gosx.Node {
	components := siteMapMapList(blueprint, "components")
	componentChildren := make([]gosx.Node, 0, len(components))
	for _, component := range components {
		componentChildren = append(componentChildren, gosx.El("small", gosx.Attrs(gosx.Attr("data-studio-gosx-component", workbenchMapString(component, "gosxComponent"))), gosx.Text(workbenchMapString(component, "label"))))
	}
	selected := state.selectedBlueprintKey != "" && state.selectedBlueprintKey == workbenchMapString(blueprint, "key")
	return gosx.El("label", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-blueprint"),
		gosx.Attr("data-studio-site-map-blueprint", workbenchMapString(blueprint, "key")),
		gosx.Attr("data-studio-site-map-route-pattern", workbenchMapString(blueprint, "routePattern")),
		gosx.Attr("data-studio-gosx-component", workbenchMapString(blueprint, "gosxComponent")),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-builder__input"),
			gosx.Attr("id", workbenchMapString(blueprint, "inputID")),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", workbenchMapString(blueprint, "inputName")),
			gosx.Attr("value", workbenchMapString(blueprint, "key")),
			gosx.Attr("checked", selected),
			gosx.Attr("aria-label", workbenchMapString(blueprint, "label")),
		)),
		gosx.El("span", nil, gosx.Text(workbenchMapString(blueprint, "label"))),
		gosx.El("small", nil, gosx.Text(workbenchMapString(blueprint, "summary"))),
		gosx.El("output", nil, gosx.Text(workbenchMapString(blueprint, "componentCountLabel"))),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-blueprint__plan"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(blueprint, "hasComponents"))),
		),
			gosx.El("strong", nil, gosx.Text("Page structure")),
			gosx.Fragment(componentChildren...),
		),
	)
}

func renderSiteMapBoardPalette(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	templates := workbenchViewMapList(siteMapView, "palette")
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
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "paletteCountLabel"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(!workbenchViewBool(siteMapView, "hasPalette"))),
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
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchViewBool(siteMapView, "hasPalette"))),
		), gosx.Fragment(children...)),
	)
}

func renderSiteMapBoardTemplate(template map[string]any, state siteMapBoardState) gosx.Node {
	controls := siteMapMapList(template, "controls")
	controlChildren := make([]gosx.Node, 0, len(controls))
	for _, control := range controls {
		controlChildren = append(controlChildren, gosx.El("small", gosx.Attrs(
			gosx.Attr("data-studio-site-map-control", workbenchMapString(control, "key")),
			gosx.Attr("data-studio-site-map-control-kind", workbenchMapString(control, "kind")),
		), gosx.Text(workbenchMapString(control, "label"))))
	}
	selected := state.selectedTemplateKey != "" && state.selectedTemplateKey == workbenchMapString(template, "key")
	return gosx.El("label", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-palette-card"),
		gosx.Attr("data-studio-site-map-component-template", workbenchMapString(template, "key")),
		gosx.Attr("data-studio-site-map-template-category", workbenchMapString(template, "categoryKey")),
		gosx.Attr("data-studio-site-map-template-source", workbenchMapString(template, "source")),
		gosx.Attr("data-studio-site-map-default-binding", workbenchMapString(template, "defaultBinding")),
		gosx.Attr("data-studio-gosx-component", workbenchMapString(template, "gosxComponent")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardMatchesPalette(state.palette, workbenchMapString(template, "categoryKey")))),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-builder__input"),
			gosx.Attr("id", workbenchMapString(template, "inputID")),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", workbenchMapString(template, "inputName")),
			gosx.Attr("value", workbenchMapString(template, "key")),
			gosx.Attr("checked", selected),
			gosx.Attr("aria-label", workbenchMapString(template, "label")),
		)),
		gosx.El("span", nil, gosx.Text(workbenchMapString(template, "label"))),
		gosx.El("small", nil, gosx.Text(workbenchMapString(template, "summary"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-palette-card__meta")),
			gosx.El("output", nil, gosx.Text(workbenchMapString(template, "statusLabel"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(template, "controlLabel"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-palette-card__plan"),
			gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(template, "hasControls"))),
		),
			gosx.El("strong", nil, gosx.Text("No-code controls")),
			gosx.Fragment(controlChildren...),
		),
		gosx.El("strong", gosx.Attrs(gosx.Attr("class", "studio-site-map-palette-card__add")), gosx.Text(workbenchMapString(template, "addLabel"))),
	)
}

func renderSiteMapBoardViewport(siteMapView map[string]any, state siteMapBoardState) gosx.Node {
	pages := workbenchViewMapList(siteMapView, "pages")
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
	components := siteMapMapList(page, "components")
	componentNodes := make([]gosx.Node, 0, len(components))
	sourceNodes := make([]gosx.Node, 0, len(components))
	controlNodes := make([]gosx.Node, 0, len(components))
	for _, component := range components {
		componentNodes = append(componentNodes, renderSiteMapBoardComponent(component))
		sourceNodes = append(sourceNodes, renderSiteMapBoardSourceRow(component))
		controlNodes = append(controlNodes, renderSiteMapBoardControlCard(component))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(page, "class")),
		gosx.Attr("data-studio-site-map-page", workbenchMapString(page, "key")),
		gosx.Attr("data-studio-site-map-route", workbenchMapString(page, "route")),
		gosx.Attr("data-studio-site-map-group", workbenchMapString(page, "group")),
		gosx.Attr("data-studio-gosx-component", workbenchMapString(page, "gosxComponent")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(siteMapBoardMatchesGroup(state.group, workbenchMapString(page, "group")))),
	),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-site-map-page__head")),
			gosx.El("label", gosx.Attrs(
				gosx.Attr("class", "studio-site-map-page__select"),
				gosx.Attr("data-studio-site-map-select", workbenchMapString(page, "key")),
			),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("class", "studio-site-map-page__input"),
					gosx.Attr("type", "radio"),
					gosx.Attr("name", "studioSiteMapPage"),
					gosx.Attr("value", workbenchMapString(page, "key")),
					gosx.Attr("checked", workbenchMapBool(page, "selected")),
					gosx.Attr("aria-label", workbenchMapString(page, "label")),
				)),
				gosx.El("span", nil, gosx.Text(workbenchMapString(page, "label"))),
				gosx.El("small", nil, gosx.Text(workbenchMapString(page, "route"))),
			),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("href", workbenchMapString(page, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text("Open")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-page__meta")),
			gosx.El("output", nil, gosx.Text(workbenchMapString(page, "statusLabel"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(page, "typeLabel"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(page, "groupLabel"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-components"),
			gosx.Attr("aria-label", workbenchMapString(page, "componentCountLabel")),
		), gosx.Fragment(componentNodes...)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-source-stack"),
			gosx.Attr("aria-label", workbenchMapString(page, "componentCountLabel")),
		), gosx.Fragment(sourceNodes...)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-control-stack"),
			gosx.Attr("aria-label", workbenchMapString(page, "controlCountLabel")),
		), gosx.Fragment(controlNodes...)),
		gosx.El("footer", gosx.Attrs(gosx.Attr("class", "studio-site-map-page__foot")),
			gosx.El("span", nil, gosx.Text(workbenchMapString(page, "componentCountLabel"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(page, "controlCountLabel"))),
		),
	)
}

func renderSiteMapBoardComponent(component map[string]any) gosx.Node {
	return gosx.El("label", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-component"),
		gosx.Attr("data-studio-site-map-component", workbenchMapString(component, "key")),
		gosx.Attr("data-studio-gosx-component", workbenchMapString(component, "gosxComponent")),
		gosx.Attr("data-studio-site-map-source", workbenchMapString(component, "source")),
		gosx.Attr("data-studio-site-map-binding", workbenchMapString(component, "binding")),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-component__input"),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", "studioSiteMapComponent"),
			gosx.Attr("value", workbenchMapString(component, "selectionKey")),
			gosx.Attr("aria-label", workbenchMapString(component, "label")),
		)),
		gosx.El("span", nil, gosx.Text(workbenchMapString(component, "label"))),
		gosx.El("small", nil, gosx.Text(workbenchMapString(component, "sourceLabel"))),
		gosx.El("small", gosx.Attrs(gosx.Attr("class", "studio-site-map-component__binding")), gosx.Text(workbenchMapString(component, "summary"))),
		gosx.El("output", nil, gosx.Text(workbenchMapString(component, "statusLabel"))),
		renderSiteMapBoardControls(component),
	)
}

func renderSiteMapBoardSourceRow(component map[string]any) gosx.Node {
	return gosx.El("label", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-source-row"),
		gosx.Attr("data-studio-site-map-component", workbenchMapString(component, "key")),
		gosx.Attr("data-studio-gosx-component", workbenchMapString(component, "gosxComponent")),
		gosx.Attr("data-studio-site-map-source", workbenchMapString(component, "source")),
		gosx.Attr("data-studio-site-map-binding", workbenchMapString(component, "binding")),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-component__input"),
			gosx.Attr("type", "radio"),
			gosx.Attr("name", "studioSiteMapComponent"),
			gosx.Attr("value", workbenchMapString(component, "selectionKey")),
			gosx.Attr("aria-label", workbenchMapString(component, "label")),
		)),
		gosx.El("strong", nil, gosx.Text(workbenchMapString(component, "sourceLabel"))),
		gosx.El("span", nil, gosx.Text(workbenchMapString(component, "sourceSummary"))),
		gosx.El("small", nil, gosx.Text(workbenchMapString(component, "summary"))),
		gosx.El("small", nil, gosx.Text(workbenchMapString(component, "controlLabel"))),
	)
}

func renderSiteMapBoardControlCard(component map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-control-card"),
		gosx.Attr("data-studio-site-map-component", workbenchMapString(component, "key")),
	),
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(component, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(component, "controlLabel"))),
		),
		renderSiteMapBoardControls(component),
	)
}

func renderSiteMapBoardControls(component map[string]any) gosx.Node {
	controls := siteMapMapList(component, "controls")
	children := make([]gosx.Node, 0, len(controls))
	for _, control := range controls {
		children = append(children, gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-control"),
			gosx.Attr("data-studio-site-map-control", workbenchMapString(control, "key")),
			gosx.Attr("data-studio-site-map-control-kind", workbenchMapString(control, "kind")),
			gosx.Attr("data-gosx-studio-authoring-operation", workbenchMapString(control, "authoringOperation")),
			gosx.Attr("data-gosx-studio-authoring-page", workbenchMapString(control, "authoringPageKey")),
			gosx.Attr("data-gosx-studio-authoring-component", workbenchMapString(control, "authoringComponent")),
			gosx.Attr("data-gosx-studio-authoring-control", workbenchMapString(control, "authoringControl")),
			gosx.Attr("data-gosx-studio-authoring-binding", workbenchMapString(control, "authoringBinding")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(control, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(control, "kindLabel"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-controls"),
		gosx.Attr("aria-label", workbenchMapString(component, "controlLabel")),
		gosx.Attr("data-studio-site-map-visible", siteMapBoardBoolString(workbenchMapBool(component, "hasControls"))),
	), gosx.Fragment(children...))
}

func renderSiteMapBoardMinimap(siteMapView map[string]any) gosx.Node {
	return gosx.El("aside", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-minimap"),
		gosx.Attr("aria-label", "Site map overview"),
	),
		gosx.El("strong", nil, gosx.Text("Board")),
		gosx.El("span", nil, gosx.Text(workbenchViewString(siteMapView, "pageCountLabel"))),
		gosx.El("span", nil, gosx.Text(workbenchViewString(siteMapView, "componentCountLabel"))),
		gosx.El("span", nil, gosx.Text(workbenchViewString(siteMapView, "controlCountLabel"))),
		gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "selectedPageLabel"))),
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
	key := workbenchMapString(layer, "key")
	focus = strings.TrimSpace(focus)
	return focus == "" || focus == "all" || key == focus || key == "resources"
}

func siteMapBoardLinkVisible(link map[string]any, group string) bool {
	group = strings.TrimSpace(group)
	if group == "" || group == "all" {
		return true
	}
	return workbenchMapString(link, "fromGroup") == group || workbenchMapString(link, "toGroup") == group
}

func siteMapBoardLinkSelected(link map[string]any, selectedNodeKey string) bool {
	return selectedNodeKey != "" && (workbenchMapString(link, "fromNodeKey") == selectedNodeKey || workbenchMapString(link, "toNodeKey") == selectedNodeKey)
}

func siteMapBoardIntentActive(intent map[string]any, state siteMapBoardState) bool {
	switch workbenchMapString(intent, "kind") {
	case "create-page":
		return state.selectedBlueprintKey != "" && state.selectedBlueprintKey == workbenchMapString(intent, "pageBlueprintKey")
	case "add-component":
		return state.selectedTemplateKey != "" && state.selectedTemplateKey == workbenchMapString(intent, "componentTemplateKey")
	default:
		return false
	}
}
