package studio

import "m31labs.dev/gosx"

type BackendStorefrontPreviewProps struct {
	Kicker       string
	Title        string
	Summary      string
	Class        string
	RefreshHref  string
	PublicHref   string
	PathInput    string
	RouteSummary string
	PreviewPath  string
	Routes       []BackendStorefrontPreviewRoute
}

type BackendStorefrontPreviewRoute struct {
	Label    string
	Href     string
	Class    string
	GOSXLink bool
}

func RenderBackendStorefrontPreview(props BackendStorefrontPreviewProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-storefront-preview-renderer", "gosx-studio"),
	),
		RenderBackendStorefrontPreviewContent(props),
	)
}

func RenderBackendStorefrontPreviewContent(props BackendStorefrontPreviewProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendStorefrontPreviewHeading(props),
		RenderBackendStorefrontPreviewPanel(props),
	)
}

func RenderBackendStorefrontPreviewHeading(props BackendStorefrontPreviewProps) gosx.Node {
	children := []gosx.Node{
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
		gosx.El("h1", nil, gosx.Text(props.Title)),
	}
	if props.Summary != "" {
		children = append(children, gosx.El("p", nil, gosx.Text(props.Summary)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Fragment(children...))
}

func RenderBackendStorefrontPreviewPanel(props BackendStorefrontPreviewProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel storefront-preview")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "storefront-preview__toolbar")),
			gosx.El("nav", gosx.Attrs(
				gosx.Attr("class", "storefront-tabs"),
				gosx.Attr("aria-label", "Storefront preview routes"),
			), gosx.Fragment(renderBackendStorefrontPreviewRoutes(props.Routes)...)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
				gosx.El("a", gosx.Attrs(
					gosx.Attr("class", "button button--secondary"),
					gosx.Attr("href", props.RefreshHref),
					gosx.Attr("data-gosx-link", "true"),
				), gosx.Text("Refresh")),
				gosx.El("a", gosx.Attrs(
					gosx.Attr("class", "button button--primary"),
					gosx.Attr("href", props.PublicHref),
					gosx.Attr("target", "_blank"),
					gosx.Attr("rel", "noreferrer"),
				), gosx.Text("Open public")),
			),
		),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "storefront-preview__form"),
			gosx.Attr("method", "get"),
			gosx.Attr("action", "/admin/storefront"),
		),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "storefront-path")), gosx.Text("Path")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "storefront-path"),
				gosx.Attr("name", "path"),
				gosx.Attr("value", props.PathInput),
				gosx.Attr("class", "storefront-preview__path"),
			)),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", "button button--secondary"),
				gosx.Attr("type", "submit"),
			), gosx.Text("Load")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "storefront-frame-shell")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "storefront-frame-toolbar")),
				gosx.El("span", nil, gosx.Text(props.RouteSummary)),
				gosx.El("code", nil, gosx.Text(props.PreviewPath)),
			),
			gosx.El("iframe", gosx.Attrs(
				gosx.Attr("class", "storefront-frame"),
				gosx.Attr("title", "Storefront preview"),
				gosx.Attr("src", props.PreviewPath),
			)),
		),
	)
}

func renderBackendStorefrontPreviewRoutes(routes []BackendStorefrontPreviewRoute) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(routes))
	for _, route := range routes {
		attrs := []any{
			gosx.Attr("class", route.Class),
			gosx.Attr("href", route.Href),
		}
		if route.GOSXLink {
			attrs = append(attrs, gosx.Attr("data-gosx-link", "true"))
		}
		nodes = append(nodes, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(route.Label)))
	}
	return nodes
}
