package studio

import "m31labs.dev/gosx"

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
		gosx.Attr("data-studio-mode-panel", "brand"),
		gosx.Attr("data-studio-panel", "brand"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-brand-group-active", workbenchMapString(view, "defaultGroupKey")),
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
	groups := workbenchViewMapList(view, "groups")
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
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(view, "summary"))),
		),
	)
}

func renderBrandPanelPreview(view map[string]any, fields map[string]any) gosx.Node {
	preview := workbenchViewMap(fields, "preview")
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__preview-shell")),
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(view, "previewLabel"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-brand-panel__preview-slot"),
			gosx.Attr("data-studio-brand-preview-slot", "true"),
		),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", "editor-brand-preview"),
				gosx.Attr("data-editor-brand-preview", "true"),
			),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", "editor-brand-preview__corner")), gosx.Text(workbenchMapString(preview, "cornerLabel"))),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "editor-brand-handle"),
					gosx.Attr("type", "button"),
					gosx.Attr("aria-label", workbenchMapString(preview, "buttonLabel")),
					gosx.Attr("data-editor-brand-handle", "true"),
				),
					gosx.El("img", gosx.Attrs(
						gosx.Attr("src", workbenchMapString(preview, "logoURL")),
						gosx.Attr("alt", workbenchMapString(preview, "logoAlt")),
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
		gosx.Attr("id", workbenchMapString(group, "inputID")),
		gosx.Attr("type", "radio"),
		gosx.Attr("name", "studioBrandGroup"),
		gosx.Attr("value", workbenchMapString(group, "key")),
		gosx.Attr("checked", workbenchMapBool(group, "selected")),
		gosx.Attr("aria-label", workbenchMapString(group, "label")),
	))
}

func renderBrandPanelGroups(view map[string]any, groups []map[string]any) gosx.Node {
	children := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "studio-brand-panel__group"),
			gosx.Attr("for", workbenchMapString(group, "inputID")),
			gosx.Attr("data-studio-brand-group-label", workbenchMapString(group, "key")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(group, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(group, "summary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-brand-panel__groups"),
		gosx.Attr("role", "radiogroup"),
		gosx.Attr("aria-label", workbenchMapString(view, "groupLabel")),
	), gosx.Fragment(children...))
}

func renderBrandPanelGroupSlot(group map[string]any, fields map[string]any) gosx.Node {
	key := workbenchMapString(group, "key")
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text(workbenchMapString(group, "label"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(group, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-brand-panel__group-body"),
			gosx.Attr("data-studio-brand-group-body", key),
		), gosx.Fragment(renderBrandPanelGroupBody(key, fields)...)),
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-brand-panel__group-slot"),
		gosx.Attr("data-studio-brand-group-slot", key),
		gosx.Attr("data-studio-brand-group-selected", BoolAttr(workbenchMapBool(group, "selected"))),
	), gosx.Fragment(children...))
}

func renderBrandPanelGroupBody(key string, fields map[string]any) []gosx.Node {
	switch key {
	case "logo":
		return brandPanelOptionalNode(fields, "logoFieldList")
	case "placement":
		nodes := brandPanelOptionalNode(fields, "layoutFieldList")
		nodes = append(nodes, renderBrandPanelTools(workbenchViewMap(fields, "tools")))
		return nodes
	case "files":
		nodes := brandPanelOptionalNode(fields, "fileFieldList")
		nodes = append(nodes, renderBrandPanelMediaLink(workbenchViewMap(fields, "media")))
		return nodes
	default:
		return nil
	}
}

func brandPanelOptionalNode(fields map[string]any, key string) []gosx.Node {
	node, ok := fields[key].(gosx.Node)
	if !ok || workbenchNodeEmpty(node) {
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
				gosx.Attr("checked", workbenchMapBool(tools, "snapChecked")),
			)),
			gosx.El("span", nil, gosx.Text("Snap")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--compact")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "logoSnapSize")), gosx.Text(workbenchMapString(tools, "gridLabel"))),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "logoSnapSize"),
				gosx.Attr("type", "number"),
				gosx.Attr("min", "1"),
				gosx.Attr("max", "32"),
				gosx.Attr("value", workbenchMapString(tools, "snapSize")),
				gosx.Attr("data-editor-logo-snap-size", "true"),
			)),
		),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "home-section-move"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-editor-logo-reset", "true"),
		), gosx.Text(workbenchMapString(tools, "resetLabel"))),
	)
}

func renderBrandPanelMediaLink(media map[string]any) gosx.Node {
	return gosx.El("a", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(media, "class")),
		gosx.Attr("href", workbenchMapString(media, "href")),
		gosx.Attr("data-gosx-link", "true"),
	), gosx.Text(workbenchMapString(media, "label")))
}

func renderBrandPanelMediaPickerSlot(node gosx.Node) gosx.Node {
	children := []gosx.Node{}
	if !workbenchNodeEmpty(node) {
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
