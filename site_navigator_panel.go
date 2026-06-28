package studio

import "m31labs.dev/gosx"

type SiteNavigatorPanelOptions struct {
	RootAttrs map[string]any
}

func RenderSiteNavigatorHeader(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", "studio-panel-heading"),
		gosx.Attr("data-gosx-studio-site-navigator-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return gosx.El("div", gosx.Attrs(attrs...),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
		gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
	)
}

func RenderSiteNavigatorList(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", "studio-page-list"),
		gosx.Attr("aria-label", workbenchMapString(view, "label")),
		gosx.Attr("data-gosx-studio-site-navigator-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	items := workbenchViewMapList(view, "items")
	if len(items) == 0 && !workbenchMapBool(view, "hasItems") {
		return gosx.El("nav", gosx.Attrs(attrs...),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(view, "empty"))),
		)
	}
	return gosx.El("nav", gosx.Attrs(attrs...), gosx.Fragment(renderSiteNavigatorItems(items)...))
}

func renderSiteNavigatorItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(
			gosx.Attr("href", workbenchMapString(item, "href")),
			gosx.Attr("class", workbenchMapString(item, "class")),
			gosx.Attr("data-gosx-link", "true"),
			gosx.Attr("data-studio-site-page", workbenchMapString(item, "key")),
			gosx.Attr("data-studio-site-group", workbenchMapString(item, "group")),
			gosx.Attr("title", workbenchMapString(item, "summary")),
		), gosx.El("span", nil, gosx.Text(workbenchMapString(item, "label")))))
	}
	return nodes
}
