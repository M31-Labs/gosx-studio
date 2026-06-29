package studio

import "m31labs.dev/gosx"

type BrandMediaPickerOptions struct {
	RootAttrs map[string]any
}

func RenderBrandMediaPicker(view map[string]any, options BrandMediaPickerOptions) gosx.Node {
	children := []gosx.Node{
		renderBrandMediaPickerHead(view),
		renderBrandMediaPickerFilters(view),
		renderBrandMediaPickerEmpty(view),
		renderBrandMediaPickerGrid(view),
	}
	return gosx.El("section", gosx.Attrs(brandMediaPickerRootAttrs(view, options)...), gosx.Fragment(children...))
}

func brandMediaPickerRootAttrs(view map[string]any, options BrandMediaPickerOptions) []any {
	attrs := []any{
		gosx.Attr("class", "studio-brand-media-picker"),
		gosx.Attr("data-studio-media-picker-island", "brand"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-media-filter", "all"),
		gosx.Attr("data-gosx-studio-brand-media-picker-renderer", "gosx-studio"),
	}
	return appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
}

func renderBrandMediaPickerHead(view map[string]any) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h3", nil, gosx.Text(workbenchMapString(view, "title"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(view, "summary"))),
		),
		gosx.El("a", gosx.Attrs(
			gosx.Attr("href", workbenchMapString(view, "uploadHref")),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text("Upload")),
	)
}

func renderBrandMediaPickerFilters(view map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-brand-media-picker__filters"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", workbenchMapString(view, "filterLabel")),
	),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("aria-pressed", "true")), gosx.Text("All")),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("aria-pressed", "false")), gosx.Text("Ready")),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("aria-pressed", "false")), gosx.Text("Needs alt")),
	)
}

func renderBrandMediaPickerEmpty(view map[string]any) gosx.Node {
	return gosx.El("p", gosx.Attrs(
		gosx.Attr("class", "studio-brand-media-picker__empty"),
		gosx.Attr("hidden", workbenchMapBool(view, "hasAssets")),
	), gosx.Text(workbenchMapString(view, "emptyLabel")))
}

func renderBrandMediaPickerGrid(view map[string]any) gosx.Node {
	assets := workbenchViewMapList(view, "assets")
	children := make([]gosx.Node, 0, len(assets))
	for _, asset := range assets {
		children = append(children, renderBrandMediaPickerAsset(view, asset))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-brand-media-picker__grid"),
		gosx.Attr("aria-label", workbenchMapString(view, "assetLabel")),
	), gosx.Fragment(children...))
}

func renderBrandMediaPickerAsset(view, asset map[string]any) gosx.Node {
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(asset, "cardClass")),
		gosx.Attr("data-studio-media-asset", workbenchMapString(asset, "id")),
		gosx.Attr("data-studio-media-filter-group", workbenchMapString(asset, "filterGroup")),
	),
		gosx.El("img", gosx.Attrs(
			gosx.Attr("src", workbenchMapString(asset, "url")),
			gosx.Attr("alt", workbenchMapString(asset, "alt")),
		)),
		gosx.El("div", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(asset, "filename"))),
			gosx.El("span", nil, gosx.Text(workbenchMapString(asset, "statusLabel"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-brand-media-picker__actions"),
			gosx.Attr("aria-label", workbenchMapString(asset, "actionLabel")),
		),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "submit"),
				gosx.Attr("form", workbenchMapString(view, "formID")),
				gosx.Attr("formaction", workbenchMapString(view, "saveAction")),
				gosx.Attr("name", workbenchMapString(view, "logoPickName")),
				gosx.Attr("value", workbenchMapString(asset, "url")),
			), gosx.Text("Use logo")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "submit"),
				gosx.Attr("form", workbenchMapString(view, "formID")),
				gosx.Attr("formaction", workbenchMapString(view, "saveAction")),
				gosx.Attr("name", workbenchMapString(view, "faviconPickName")),
				gosx.Attr("value", workbenchMapString(asset, "url")),
			), gosx.Text("Use icon")),
		),
	)
}
