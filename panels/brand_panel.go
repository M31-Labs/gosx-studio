package panels

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

type BrandPanelOptions struct {
	RootAttrs       map[string]any
	MediaPickerNode gosx.Node
}

type BrandPanelSegments struct {
	RootOpen             gosx.Node
	Body                 gosx.Node
	MediaPickerSlotOpen  gosx.Node
	MediaPickerSlotClose gosx.Node
	RootClose            gosx.Node
}

func RenderBrandPanel(view map[string]any, fields map[string]any, options BrandPanelOptions) gosx.Node {
	children := []gosx.Node{
		renderBrandPanelBody(view, fields),
		renderBrandPanelMediaPickerSlot(options.MediaPickerNode),
	}

	return gosx.El("section", gosx.Attrs(brandPanelRootAttrs(view, options)...), gosx.Fragment(children...))
}

func RenderBrandPanelSegments(view map[string]any, fields map[string]any, options BrandPanelOptions) BrandPanelSegments {
	return BrandPanelSegments{
		RootOpen:             renderBlockLayoutEngineOpenTag("section", brandPanelRootAttrs(view, options)),
		Body:                 renderBrandPanelBody(view, fields),
		MediaPickerSlotOpen:  renderBlockLayoutEngineOpenTag("section", brandPanelMediaPickerSlotAttrs()),
		MediaPickerSlotClose: gosx.RawHTML("</section>"),
		RootClose:            gosx.RawHTML("</section>"),
	}
}

func brandPanelRootAttrs(view map[string]any, options BrandPanelOptions) []any {
	attrs := []any{
		gosx.Attr("class", "editor-panel editor-panel--brand-studio studio-brand-panel"),
		gosx.Attr("data-studio-brand-panel", "true"),
		gosx.Attr("data-studio-mode-panel", core.FirstNonEmpty(core.WorkbenchViewString(view, "mode"), "brand")),
		gosx.Attr("data-studio-panel", "brand"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-brand-group-active", core.WorkbenchViewString(view, "defaultGroupKey")),
		gosx.Attr("data-gosx-studio-brand-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return attrs
}

func renderBrandPanelBody(view map[string]any, fields map[string]any) gosx.Node {
	children := []gosx.Node{
		renderBrandPanelHead(view),
		renderBrandPanelPreview(view, fields),
	}
	groups := core.WorkbenchViewMapList(view, "groups")
	for _, group := range groups {
		children = append(children, renderBrandPanelGroupInput(group))
	}
	children = append(children, renderBrandPanelGroups(view, groups))
	for _, group := range groups {
		children = append(children, renderBrandPanelGroupSlot(group, fields))
	}

	return gosx.Fragment(children...)
}

func renderBrandPanelHead(view map[string]any) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(view, "summary"))),
		),
	)
}

func renderBrandPanelPreview(view map[string]any, fields map[string]any) gosx.Node {
	preview := core.WorkbenchViewMap(fields, "preview")
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__preview-shell")),
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(view, "previewLabel"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-brand-panel__preview-slot"),
			gosx.Attr("data-studio-brand-preview-slot", "true"),
		),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "editor-brand-preview"),
				gosx.Attr("data-editor-brand-preview", "true"),
			),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", "editor-brand-preview__corner")), gosx.Text(core.WorkbenchViewString(preview, "cornerLabel"))),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "editor-brand-handle"),
					gosx.Attr("type", "button"),
					gosx.Attr("aria-label", core.WorkbenchViewString(preview, "buttonLabel")),
					gosx.Attr("data-editor-brand-handle", "true"),
				),
					gosx.El("img", gosx.Attrs(
						gosx.Attr("src", core.WorkbenchViewString(preview, "logoURL")),
						gosx.Attr("alt", core.WorkbenchViewString(preview, "logoAlt")),
						gosx.Attr("data-editor-brand-logo", "true"),
					)),
				),
				gosx.El("output", gosx.Attrs(
					gosx.Attr("class", "editor-brand-readout"),
					gosx.Attr("data-editor-logo-readout", "true"),
					gosx.Attr("aria-live", "polite"),
				)),
			),
		),
	)
}

func renderBrandPanelGroupInput(group map[string]any) gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("class", "studio-brand-panel__group-input"),
		gosx.Attr("id", core.WorkbenchViewString(group, "inputID")),
		gosx.Attr("type", "radio"),
		gosx.Attr("name", "studioBrandGroup"),
		gosx.Attr("value", core.WorkbenchViewString(group, "key")),
		gosx.Attr("checked", core.WorkbenchViewBool(group, "selected")),
		gosx.Attr("aria-label", core.WorkbenchViewString(group, "label")),
	))
}

func renderBrandPanelGroups(view map[string]any, groups []map[string]any) gosx.Node {
	children := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "studio-brand-panel__group"),
			gosx.Attr("for", core.WorkbenchViewString(group, "inputID")),
			gosx.Attr("data-studio-brand-group-label", core.WorkbenchViewString(group, "key")),
		),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(group, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(group, "summary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-brand-panel__groups"),
		gosx.Attr("role", "radiogroup"),
		gosx.Attr("aria-label", core.WorkbenchViewString(view, "groupLabel")),
	), gosx.Fragment(children...))
}

func renderBrandPanelGroupSlot(group map[string]any, fields map[string]any) gosx.Node {
	key := core.WorkbenchViewString(group, "key")
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text(core.WorkbenchViewString(group, "label"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(group, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-brand-panel__group-body"),
			gosx.Attr("data-studio-brand-group-body", key),
		), gosx.Fragment(renderBrandPanelGroupBody(key, fields)...)),
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-brand-panel__group-slot"),
		gosx.Attr("data-studio-brand-group-slot", key),
		gosx.Attr("data-studio-brand-group-selected", core.BoolAttr(core.WorkbenchViewBool(group, "selected"))),
	), gosx.Fragment(children...))
}

func renderBrandPanelGroupBody(key string, fields map[string]any) []gosx.Node {
	switch key {
	case "logo":
		return brandPanelOptionalNode(fields, "logoFieldList")
	case "placement":
		nodes := brandPanelOptionalNode(fields, "layoutFieldList")
		nodes = append(nodes, renderBrandPanelTools(core.WorkbenchViewMap(fields, "tools")))
		return nodes
	case "files":
		nodes := brandPanelOptionalNode(fields, "fileFieldList")
		nodes = append(nodes, renderBrandPanelMediaLink(core.WorkbenchViewMap(fields, "media")))
		return nodes
	default:
		return nil
	}
}

func brandPanelOptionalNode(fields map[string]any, key string) []gosx.Node {
	node, ok := fields[key].(gosx.Node)
	if !ok || core.WorkbenchNodeEmpty(node) {
		return nil
	}
	return []gosx.Node{node}
}

func renderBrandPanelTools(tools map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-brand-tools")),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "editor-brand-snap")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "checkbox"),
				gosx.Attr("data-editor-logo-snap", "true"),
				gosx.Attr("checked", core.WorkbenchViewBool(tools, "snapChecked")),
			)),
			gosx.El("span", nil, gosx.Text("Snap")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--compact")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "logoSnapSize")), gosx.Text(core.WorkbenchViewString(tools, "gridLabel"))),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "logoSnapSize"),
				gosx.Attr("type", "number"),
				gosx.Attr("min", "1"),
				gosx.Attr("max", "32"),
				gosx.Attr("value", core.WorkbenchViewString(tools, "snapSize")),
				gosx.Attr("data-editor-logo-snap-size", "true"),
			)),
		),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "home-section-move"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-editor-logo-reset", "true"),
		), gosx.Text(core.WorkbenchViewString(tools, "resetLabel"))),
	)
}

func renderBrandPanelMediaLink(media map[string]any) gosx.Node {
	return gosx.El("a", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(media, "class")),
		gosx.Attr("href", core.WorkbenchViewString(media, "href")),
		gosx.Attr("data-gosx-link", "true"),
	), gosx.Text(core.WorkbenchViewString(media, "label")))
}

func renderBrandPanelMediaPickerSlot(node gosx.Node) gosx.Node {
	children := []gosx.Node{}
	if !core.WorkbenchNodeEmpty(node) {
		children = append(children, node)
	}
	return gosx.El("section", gosx.Attrs(brandPanelMediaPickerSlotAttrs()...), gosx.Fragment(children...))
}

func brandPanelMediaPickerSlotAttrs() []any {
	return []any{
		gosx.Attr("class", "studio-brand-panel__group-slot studio-brand-media-picker-slot"),
		gosx.Attr("data-studio-brand-group-slot", "files"),
		gosx.Attr("data-studio-brand-group-selected", "false"),
	}
}
