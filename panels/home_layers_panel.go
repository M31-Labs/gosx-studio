package panels

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type HomeLayersPanelOptions struct {
	RootAttrs  map[string]any
	PickerNode gosx.Node
}

type HomeLayersPanelSegments struct {
	RootOpen    gosx.Node
	HeaderOpen  gosx.Node
	HeaderClose gosx.Node
	Body        gosx.Node
	RootClose   gosx.Node
}

func RenderHomeLayersPanelSegments(view map[string]any, options HomeLayersPanelOptions) HomeLayersPanelSegments {
	return HomeLayersPanelSegments{
		RootOpen:    renderHomeLayersPanelRootOpen(view, options),
		HeaderOpen:  renderHomeLayersPanelHeaderOpen(view),
		HeaderClose: gosx.RawHTML("</header>"),
		Body:        renderHomeLayersPanelBody(view),
		RootClose:   gosx.RawHTML("</section>"),
	}
}

func RenderHomeLayersPanel(view map[string]any, options HomeLayersPanelOptions) gosx.Node {
	return gosx.El("section", gosx.Attrs(homeLayersPanelRootAttrs(view, options)...),
		renderHomeLayersPanelHeader(view, options.PickerNode),
		renderHomeLayersPanelBody(view),
	)
}

func homeLayersPanelRootAttrs(view map[string]any, options HomeLayersPanelOptions) []any {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-home-layers-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return attrs
}

func renderHomeLayersPanelRootOpen(view map[string]any, options HomeLayersPanelOptions) gosx.Node {
	attrs := homeLayersPanelRootAttrs(view, options)
	return renderBlockLayoutEngineOpenTag("section", attrs)
}

func renderHomeLayersPanelHeader(view map[string]any, pickerNode gosx.Node) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
		),
		pickerNode,
	)
}

func renderHomeLayersPanelHeaderOpen(view map[string]any) gosx.Node {
	html := gosx.RenderHTML(renderHomeLayersPanelHeader(view, gosx.Fragment()))
	return gosx.RawHTML(strings.TrimSuffix(html, "</header>"))
}

func renderHomeLayersPanelBody(view map[string]any) gosx.Node {
	items := core.WorkbenchViewMapList(view, "items")
	if len(items) == 0 && !core.WorkbenchViewBool(view, "hasItems") {
		return gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(core.WorkbenchViewString(view, "empty")))
	}
	if len(items) == 0 && core.WorkbenchViewBool(view, "hasItems") {
		return gosx.El("div", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(view, "listClass")),
			gosx.Attr("data-block-studio", core.WorkbenchViewString(view, "blockStudioKey")),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(view, "listClass")),
		gosx.Attr("data-block-studio", core.WorkbenchViewString(view, "blockStudioKey")),
	), gosx.Fragment(renderHomeLayersPanelItems(items)...))
}

func renderHomeLayersPanelItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, renderHomeLayersPanelItem(item))
	}
	return nodes
}

func renderHomeLayersPanelItem(item map[string]any) gosx.Node {
	preview := core.WorkbenchViewMap(item, "preview")
	return gosx.El("article", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "rowAttrs"))...),
		gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "keyInputAttrs"))...)),
		gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "orderInputAttrs"))...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__chrome")),
			gosx.El("button", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "handleAttrs"))...), gosx.Text("Drag")),
			gosx.El("span", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "statusAttrs"))...), gosx.Text(core.WorkbenchViewString(item, "statusLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__preview")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(preview, "visualClass"))),
				gosx.El("span", nil),
				gosx.El("span", nil),
				gosx.El("span", nil),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__copy")),
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(preview, "kicker"))),
				gosx.El("h2", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(preview, "titleAttrs"))...), gosx.Text(core.WorkbenchViewString(preview, "title"))),
				gosx.El("p", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(preview, "bodyAttrs"))...), gosx.Text(core.WorkbenchViewString(preview, "body"))),
				renderHomeLayersPanelPreviewActions(preview),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__actions")),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "editor-visibility")),
				gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "visibilityAttrs"))...)),
				gosx.Text(" "),
				gosx.El("span", gosx.Attrs(gosx.Attr("data-editor-block-status", "true")), gosx.Text(core.WorkbenchViewString(item, "statusLabel"))),
			),
			gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "hiddenEnabledAttrs"))...)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-order-actions")),
				gosx.El("button", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "upButtonAttrs"))...), gosx.Text("Move up")),
				gosx.El("button", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(item, "downButtonAttrs"))...), gosx.Text("Move down")),
			),
		),
	)
}

func renderHomeLayersPanelPreviewActions(preview map[string]any) gosx.Node {
	if !core.WorkbenchViewBool(preview, "hasActions") {
		return gosx.Fragment()
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
		gosx.Fragment(renderHomeLayersPanelPreviewActionItems(core.WorkbenchViewMapList(preview, "actions"))...),
	)
}

func renderHomeLayersPanelPreviewActionItems(actions []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		nodes = append(nodes, gosx.El("span", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(action, "attrs"))...), gosx.Text(core.WorkbenchViewString(action, "label"))))
	}
	return nodes
}
