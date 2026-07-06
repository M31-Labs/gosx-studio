package sitemap

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type SiteNavigatorItem struct {
	Key     string
	Group   string
	Label   string
	Href    string
	Class   string
	Summary string
}

type SiteNavigatorProps struct {
	Mode     string
	Kicker   string
	Title    string
	Label    string
	Empty    string
	HasItems bool
	Items    []SiteNavigatorItem
}

type SiteNavigatorPanelOptions struct {
	RootAttrs map[string]any
}

func SiteNavigatorPropsFromMap(view map[string]any) SiteNavigatorProps {
	return SiteNavigatorProps{
		Mode:     core.WorkbenchViewString(view, "mode"),
		Kicker:   core.WorkbenchViewString(view, "kicker"),
		Title:    core.WorkbenchViewString(view, "title"),
		Label:    core.WorkbenchViewString(view, "label"),
		Empty:    core.WorkbenchViewString(view, "empty"),
		HasItems: core.WorkbenchViewBool(view, "hasItems"),
		Items:    siteNavigatorItemsFromMap(view),
	}
}

func RenderSiteNavigatorPanel(props SiteNavigatorProps, options SiteNavigatorPanelOptions) gosx.Node {
	view := map[string]any{
		"mode":     props.Mode,
		"kicker":   props.Kicker,
		"title":    props.Title,
		"label":    props.Label,
		"empty":    props.Empty,
		"hasItems": props.HasItems,
		"items":    siteNavigatorItemViews(props.Items),
	}
	attrs := []any{
		gosx.Attr("class", "studio-nav-panel"),
		gosx.Attr("data-studio-site-navigator-panel", "true"),
		gosx.Attr("data-studio-mode-panel", props.Mode),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-site-filter", "all"),
		gosx.Attr("data-gosx-studio-site-navigator-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return gosx.El("section", gosx.Attrs(attrs...),
		RenderSiteNavigatorHeader(view, SiteNavigatorPanelOptions{}),
		renderSiteNavigatorFilterControls(),
		RenderSiteNavigatorList(view, SiteNavigatorPanelOptions{}),
	)
}

func RenderSiteNavigatorHeader(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", "studio-panel-heading"),
		gosx.Attr("data-gosx-studio-site-navigator-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return gosx.El("div", gosx.Attrs(attrs...),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
		gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
	)
}

func renderSiteNavigatorFilterControls() gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-page-filter"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Site area filter"),
	),
		renderSiteNavigatorFilterButton("All", true),
		renderSiteNavigatorFilterButton("Site", false),
		renderSiteNavigatorFilterButton("Content", false),
		renderSiteNavigatorFilterButton("Store", false),
		renderSiteNavigatorFilterButton("Tools", false),
	)
}

func renderSiteNavigatorFilterButton(label string, pressed bool) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("type", "button"),
		gosx.Attr("aria-pressed", pressed),
	), gosx.Text(label))
}

func RenderSiteNavigatorList(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", "studio-page-list"),
		gosx.Attr("aria-label", core.WorkbenchViewString(view, "label")),
		gosx.Attr("data-gosx-studio-site-navigator-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	items := core.WorkbenchViewMapList(view, "items")
	if len(items) == 0 && !core.WorkbenchViewBool(view, "hasItems") {
		return gosx.El("nav", gosx.Attrs(attrs...),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(core.WorkbenchViewString(view, "empty"))),
		)
	}
	return gosx.El("nav", gosx.Attrs(attrs...), gosx.Fragment(renderSiteNavigatorItems(items)...))
}

func renderSiteNavigatorItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(
			gosx.Attr("href", core.WorkbenchViewString(item, "href")),
			gosx.Attr("class", core.WorkbenchViewString(item, "class")),
			gosx.Attr("data-gosx-link", "true"),
			gosx.Attr("data-studio-site-page", core.WorkbenchViewString(item, "key")),
			gosx.Attr("data-studio-site-group", core.WorkbenchViewString(item, "group")),
			gosx.Attr("title", core.WorkbenchViewString(item, "summary")),
		), gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "label")))))
	}
	return nodes
}

func siteNavigatorItemViews(items []SiteNavigatorItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"key":     item.Key,
			"group":   item.Group,
			"label":   item.Label,
			"href":    item.Href,
			"class":   item.Class,
			"summary": item.Summary,
		})
	}
	return out
}

func siteNavigatorItemsFromMap(view map[string]any) []SiteNavigatorItem {
	items := core.WorkbenchViewMapList(view, "items")
	out := make([]SiteNavigatorItem, 0, len(items))
	for _, item := range items {
		out = append(out, SiteNavigatorItem{
			Key:     core.WorkbenchViewString(item, "key"),
			Group:   core.WorkbenchViewString(item, "group"),
			Label:   core.WorkbenchViewString(item, "label"),
			Href:    core.WorkbenchViewString(item, "href"),
			Class:   core.WorkbenchViewString(item, "class"),
			Summary: core.WorkbenchViewString(item, "summary"),
		})
	}
	return out
}
