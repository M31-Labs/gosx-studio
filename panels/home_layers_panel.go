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
	Toolbar     gosx.Node
	Body        gosx.Node
	RootClose   gosx.Node
}

func RenderHomeLayersPanelSegments(view map[string]any, options HomeLayersPanelOptions) HomeLayersPanelSegments {
	return HomeLayersPanelSegments{
		RootOpen:    renderHomeLayersPanelRootOpen(view, options),
		HeaderOpen:  renderHomeLayersPanelHeaderOpen(view),
		HeaderClose: gosx.RawHTML("</header>"),
		Toolbar:     renderHomeLayersPanelToolbar(),
		Body:        renderHomeLayersPanelBody(view),
		RootClose:   gosx.RawHTML("</section>"),
	}
}

func RenderHomeLayersPanel(view map[string]any, options HomeLayersPanelOptions) gosx.Node {
	return gosx.El("section", gosx.Attrs(homeLayersPanelRootAttrs(view, options)...),
		renderHomeLayersPanelHeader(view, options.PickerNode),
		renderHomeLayersPanelToolbar(),
		renderHomeLayersPanelBody(view),
	)
}

// renderHomeLayersPanelToolbar renders the section manager's visible "+ Add
// section" affordance. It is a plain same-page anchor to the block library
// ("Sections") panel that RenderBlockLayoutEngine already stacks directly
// below this one in Home mode — no script is required to reveal it, and no
// new insertion flow is introduced: clicking through lands on the exact
// existing "Add"/"Add another" buttons block_library_panel.go renders.
func renderHomeLayersPanelToolbar() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-panel-toolbar")),
		gosx.El("a", gosx.Attrs(
			gosx.Attr("class", "button button--secondary studio-add-section"),
			gosx.Attr("href", "#"+blockLibraryPanelAnchorID),
			gosx.Attr("data-studio-add-section", "true"),
		), gosx.Text("+ Add section")),
	)
}

func homeLayersPanelRootAttrs(view map[string]any, options HomeLayersPanelOptions) []any {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-home-layers-panel-renderer", "gosx-studio"),
		// handoff-4 (item 5 -- left rail section-collapse): see
		// sitemap/site_navigator_panel.go's identical comment.
		gosx.Attr("data-studio-rail-group", "layers"),
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
			renderHomeLayersPanelDelete(item),
		),
	)
}

// renderHomeLayersPanelDelete renders a section row's real, confirmable
// "Delete" action, dispatching the existing delete-component durable
// authoring operation (studio.AuthoringOperationDeleteComponent), separate
// from the "editor-visibility" checkbox above it (which stays a plain
// hide/show toggle bundled into the next Save). It only renders when the
// host supplies a "delete" sub-map with hasDelete=true; hosts that have not
// wired a delete action yet see no button (never a silently-broken one).
//
// This deliberately does NOT render its own <form>: the whole workbench
// (this panel included) already sits inside one shared <form> (see
// shell/workbench_frame.go), and a nested <form> is invalid HTML — the
// browser's parser drops the inner tag and re-associates its children with
// the outer form anyway (the exact failure mode documented on
// renderRevisionHistoryWorkbenchButton). Instead this mirrors that same
// proven pattern: fixed (operation, page key) hidden inputs that are
// identical across every row (harmless if the browser submits the same
// name/value more than once) plus a formaction-overriding submit button
// whose OWN name/value pair carries the one field that truly varies per
// row — this section's component key.
func renderHomeLayersPanelDelete(item map[string]any) gosx.Node {
	del := core.WorkbenchViewMap(item, "delete")
	if !core.WorkbenchViewBool(del, "hasDelete") {
		return gosx.Fragment()
	}
	children := make([]gosx.Node, 0, 4)
	for _, input := range core.WorkbenchViewMapList(del, "hiddenInputs") {
		children = append(children, gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", core.WorkbenchViewString(input, "name")),
			gosx.Attr("value", core.WorkbenchViewString(input, "value")),
		)))
	}
	children = append(children, gosx.El("button", gosx.Attrs(
		gosx.Attr("type", "submit"),
		gosx.Attr("class", "button button--danger studio-delete-section"),
		gosx.Attr("formaction", core.WorkbenchViewString(del, "action")),
		gosx.Attr("formmethod", "post"),
		gosx.Attr("formnovalidate", "formnovalidate"),
		gosx.Attr("name", core.WorkbenchViewString(del, "componentKeyName")),
		gosx.Attr("value", core.WorkbenchViewString(del, "componentKeyValue")),
		gosx.Attr("data-admin-confirm", core.WorkbenchViewString(del, "confirm")),
		gosx.Attr("data-studio-delete-section", "true"),
	), gosx.Text(core.FirstNonEmpty(core.WorkbenchViewString(del, "buttonLabel"), "Delete"))))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__delete")), gosx.Fragment(children...))
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
