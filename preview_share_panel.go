package studio

import "m31labs.dev/gosx"

type PreviewSharePanelOptions struct {
	RootAttrs map[string]any
}

func RenderPreviewSharePanel(view map[string]any, options PreviewSharePanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-studio-preview-share", "true"),
		gosx.Attr("data-studio-preview-share-state", workbenchMapString(view, "state")),
		gosx.Attr("data-gosx-studio-preview-share-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-preview-share__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-preview-share__kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
				gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(view, "status"))),
		),
		gosx.Fragment(renderPreviewSharePanelBody(view)...),
	)
}

func renderPreviewSharePanelBody(view map[string]any) []gosx.Node {
	if !workbenchMapBool(view, "hasHref") {
		return []gosx.Node{
			gosx.El("article", gosx.Attrs(gosx.Attr("class", "studio-preview-share__empty")),
				gosx.El("strong", nil, gosx.Text(workbenchMapString(view, "emptyTitle"))),
				gosx.El("p", nil, gosx.Text(workbenchMapString(view, "emptyDetail"))),
			),
		}
	}

	return []gosx.Node{
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-preview-share__detail")), gosx.Text(workbenchMapString(view, "detail"))),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-preview-share__field")),
			gosx.El("span", nil, gosx.Text(workbenchMapString(view, "inputLabel"))),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "url"),
				gosx.Attr("readonly", "readonly"),
				gosx.Attr("value", workbenchMapString(view, "href")),
				gosx.Attr("data-studio-preview-url", "true"),
				gosx.Attr("aria-label", workbenchMapString(view, "inputLabel")),
			)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-preview-share__actions")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("data-studio-copy-target", "[data-studio-preview-url]"),
			), gosx.Text(workbenchMapString(view, "copyLabel"))),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("href", workbenchMapString(view, "href")),
				gosx.Attr("target", "_blank"),
				gosx.Attr("rel", "noreferrer"),
			), gosx.Text(workbenchMapString(view, "openLabel"))),
		),
		gosx.El("dl", gosx.Attrs(gosx.Attr("class", "studio-preview-share__meta")),
			gosx.El("dt", nil, gosx.Text("Resource")),
			gosx.El("dd", nil, gosx.Text(workbenchMapString(view, "resource"))),
			gosx.El("dt", nil, gosx.Text("Audience")),
			gosx.El("dd", nil, gosx.Text(workbenchMapString(view, "audience"))),
			gosx.El("dt", nil, gosx.Text("Expires")),
			gosx.El("dd", nil, gosx.Text(workbenchMapString(view, "expiresLabel"))),
		),
	}
}
