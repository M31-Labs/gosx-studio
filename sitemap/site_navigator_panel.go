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

	NewPageLabel string
	NewPageItems []SiteNavigatorNewPageItem
}

// SiteNavigatorNewPageItem is one create-page-from-blueprint choice offered by
// the "New page" disclosure in the Pages panel. FormID must match the id of
// the (already-rendered, always-present) composition-intent form the site map
// engine emits for the same blueprint (see sitemap_authoring_forms.go /
// authoringSiteMapIntentView's "formID"), so clicking the button here submits
// that existing create-page intent — no new backend wiring required. Because
// the button also carries the block-layout "add" data attribute, the command
// palette's existing insert-kind command (Ctrl+K → "New page: <label>") can
// click it too.
type SiteNavigatorNewPageItem struct {
	Key     string
	Label   string
	FormID  string
	Summary string
}

type SiteNavigatorPanelOptions struct {
	RootAttrs map[string]any
}

func SiteNavigatorPropsFromMap(view map[string]any) SiteNavigatorProps {
	return SiteNavigatorProps{
		Mode:         core.WorkbenchViewString(view, "mode"),
		Kicker:       core.WorkbenchViewString(view, "kicker"),
		Title:        core.WorkbenchViewString(view, "title"),
		Label:        core.WorkbenchViewString(view, "label"),
		Empty:        core.WorkbenchViewString(view, "empty"),
		HasItems:     core.WorkbenchViewBool(view, "hasItems"),
		Items:        siteNavigatorItemsFromMap(view),
		NewPageLabel: core.WorkbenchViewString(view, "newPageLabel"),
		NewPageItems: siteNavigatorNewPageItemsFromMap(view),
	}
}

func RenderSiteNavigatorPanel(props SiteNavigatorProps, options SiteNavigatorPanelOptions) gosx.Node {
	view := map[string]any{
		"mode":         props.Mode,
		"kicker":       props.Kicker,
		"title":        props.Title,
		"label":        props.Label,
		"empty":        props.Empty,
		"hasItems":     props.HasItems,
		"items":        siteNavigatorItemViews(props.Items),
		"newPageLabel": props.NewPageLabel,
		"newPageItems": siteNavigatorNewPageItemViews(props.NewPageItems),
	}
	attrs := []any{
		gosx.Attr("class", "studio-nav-panel"),
		gosx.Attr("data-studio-site-navigator-panel", "true"),
		gosx.Attr("data-studio-mode-panel", props.Mode),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-site-filter", "all"),
		gosx.Attr("data-gosx-studio-site-navigator-renderer", "gosx-studio"),
		// handoff-4 (item 5 -- left rail section-collapse): a stable,
		// unambiguous group key the runtime's collapseLeftRailGroups()
		// (hostruntime/assets/workbench_runtime.js) uses to add a
		// persisted collapse toggle to this section without guessing at
		// one of this panel's several "-renderer" marker attributes
		// (those repeat on nested sub-elements, not just this root).
		gosx.Attr("data-studio-rail-group", "site-navigator"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return gosx.El("section", gosx.Attrs(attrs...),
		RenderSiteNavigatorHeader(view, SiteNavigatorPanelOptions{}),
		RenderSiteNavigatorNewPage(view, SiteNavigatorPanelOptions{}),
		renderSiteNavigatorFilterControls(),
		RenderSiteNavigatorList(view, SiteNavigatorPanelOptions{}),
	)
}

// RenderSiteNavigatorNewPage renders the visible "+ New page" disclosure
// listing every create-page-from-blueprint choice as its own submit button.
// Each button submits the SAME hidden composition-intent form the ADVANCED →
// Site map builder already renders for that blueprint (matched by formID), so
// creating a page from here does not require any new authoring plumbing. The
// disclosure uses the native <details>/<summary> pair, so it needs no script
// to open.
func RenderSiteNavigatorNewPage(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	items := core.WorkbenchViewMapList(view, "newPageItems")
	if len(items) == 0 {
		return gosx.Fragment()
	}
	label := firstNonEmpty(core.WorkbenchViewString(view, "newPageLabel"), "New page")
	attrs := []any{
		gosx.Attr("class", "studio-new-page"),
		gosx.Attr("data-studio-new-page", "true"),
		gosx.Attr("data-gosx-studio-site-navigator-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return gosx.El("details", gosx.Attrs(attrs...),
		gosx.El("summary", nil, gosx.Text("+ "+label)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-new-page__options")), gosx.Fragment(renderSiteNavigatorNewPageItems(items)...)),
	)
}

func renderSiteNavigatorNewPageItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		key := core.WorkbenchViewString(item, "key")
		formID := core.WorkbenchViewString(item, "formID")
		if key == "" || formID == "" {
			continue
		}
		label := firstNonEmpty(core.WorkbenchViewString(item, "label"), key)
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "submit"),
			gosx.Attr("form", formID),
			gosx.Attr("class", "button button--secondary studio-new-page-option"),
			gosx.Attr("data-editor-add-block", "page-"+key),
			gosx.Attr("data-studio-new-page-blueprint", key),
			gosx.Attr("title", core.WorkbenchViewString(item, "summary")),
		), gosx.Text("New page: "+label)))
	}
	return nodes
}

func siteNavigatorNewPageItemViews(items []SiteNavigatorNewPageItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"key":     item.Key,
			"label":   item.Label,
			"formID":  item.FormID,
			"summary": item.Summary,
		})
	}
	return out
}

func siteNavigatorNewPageItemsFromMap(view map[string]any) []SiteNavigatorNewPageItem {
	items := core.WorkbenchViewMapList(view, "newPageItems")
	out := make([]SiteNavigatorNewPageItem, 0, len(items))
	for _, item := range items {
		out = append(out, SiteNavigatorNewPageItem{
			Key:     core.WorkbenchViewString(item, "key"),
			Label:   core.WorkbenchViewString(item, "label"),
			FormID:  core.WorkbenchViewString(item, "formID"),
			Summary: core.WorkbenchViewString(item, "summary"),
		})
	}
	return out
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
