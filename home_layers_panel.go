package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type HomeLayersPanelOptions struct {
	RootAttrs map[string]any
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

func renderHomeLayersPanelRootOpen(view map[string]any, options HomeLayersPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-studio-mode-panel", workbenchMapString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-home-layers-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return renderBlockLayoutEngineOpenTag("section", attrs)
}

func renderHomeLayersPanelHeaderOpen(view map[string]any) gosx.Node {
	html := gosx.RenderHTML(gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
		),
	))
	return gosx.RawHTML(strings.TrimSuffix(html, "</header>"))
}

func renderHomeLayersPanelBody(view map[string]any) gosx.Node {
	items := workbenchViewMapList(view, "items")
	if len(items) == 0 && !workbenchMapBool(view, "hasItems") {
		return gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(view, "empty")))
	}
	if len(items) == 0 && workbenchMapBool(view, "hasItems") {
		return gosx.El("div", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(view, "listClass")),
			gosx.Attr("data-block-studio", workbenchMapString(view, "blockStudioKey")),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(view, "listClass")),
		gosx.Attr("data-block-studio", workbenchMapString(view, "blockStudioKey")),
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
	preview := workbenchViewMap(item, "preview")
	return gosx.El("article", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "rowAttrs"))...),
		gosx.El("input", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "keyInputAttrs"))...)),
		gosx.El("input", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "orderInputAttrs"))...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__chrome")),
			gosx.El("button", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "handleAttrs"))...), gosx.Text("Drag")),
			gosx.El("span", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "statusAttrs"))...), gosx.Text(workbenchMapString(item, "statusLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__preview")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(preview, "visualClass"))),
				gosx.El("span", nil),
				gosx.El("span", nil),
				gosx.El("span", nil),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__copy")),
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(preview, "kicker"))),
				gosx.El("h2", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(preview, "titleAttrs"))...), gosx.Text(workbenchMapString(preview, "title"))),
				gosx.El("p", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(preview, "bodyAttrs"))...), gosx.Text(workbenchMapString(preview, "body"))),
				renderHomeLayersPanelPreviewActions(preview),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__actions")),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "editor-visibility")),
				gosx.El("input", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "visibilityAttrs"))...)),
				gosx.Text(" "),
				gosx.El("span", gosx.Attrs(gosx.Attr("data-editor-block-status", "true")), gosx.Text(workbenchMapString(item, "statusLabel"))),
			),
			gosx.El("input", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "hiddenEnabledAttrs"))...)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-order-actions")),
				gosx.El("button", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "upButtonAttrs"))...), gosx.Text("Move up")),
				gosx.El("button", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(item, "downButtonAttrs"))...), gosx.Text("Move down")),
			),
		),
	)
}

func renderHomeLayersPanelPreviewActions(preview map[string]any) gosx.Node {
	if !workbenchMapBool(preview, "hasActions") {
		return gosx.Fragment()
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
		gosx.Fragment(renderHomeLayersPanelPreviewActionItems(workbenchViewMapList(preview, "actions"))...),
	)
}

func renderHomeLayersPanelPreviewActionItems(actions []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		nodes = append(nodes, gosx.El("span", gosx.Attrs(blockLibraryPanelMapAttrs(workbenchViewMap(action, "attrs"))...), gosx.Text(workbenchMapString(action, "label"))))
	}
	return nodes
}
