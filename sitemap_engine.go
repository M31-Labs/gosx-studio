package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type SiteMapEngineOptions struct {
	Class        string
	ModePanel    string
	PanelKey     string
	EngineSource string
}

type SiteMapEngineSegments struct {
	RootOpen     gosx.Node
	Header       gosx.Node
	SourceLegend gosx.Node
	RootClose    gosx.Node
}

func RenderSiteMapEngineSegments(siteMapView map[string]any, options SiteMapEngineOptions) SiteMapEngineSegments {
	return SiteMapEngineSegments{
		RootOpen:     renderSiteMapEngineRootOpen(options),
		Header:       renderSiteMapEngineHeader(siteMapView),
		SourceLegend: renderSiteMapEngineSourceLegend(siteMapView),
		RootClose:    gosx.RawHTML("</section>"),
	}
}

func renderSiteMapEngineRootOpen(options SiteMapEngineOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", FirstNonEmpty(options.Class, "studio-site-map-engine")),
		gosx.Attr("data-studio-site-map-panel", "true"),
		gosx.Attr("data-studio-mode-panel", FirstNonEmpty(options.ModePanel, "home")),
		gosx.Attr("data-studio-panel", FirstNonEmpty(options.PanelKey, "site-map")),
		gosx.Attr("data-studio-engine-source", FirstNonEmpty(options.EngineSource, "gosx")),
		gosx.Attr("data-gosx-studio-site-map-engine-renderer", "gosx-studio"),
	}
	return renderSiteMapEngineOpenTag("section", attrs)
}

func renderSiteMapEngineHeader(siteMapView map[string]any) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-site-map-engine__chrome")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchViewString(siteMapView, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchViewString(siteMapView, "title"))),
			gosx.El("p", nil, gosx.Text(workbenchViewString(siteMapView, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-engine__stats"),
			gosx.Attr("aria-label", "Site map summary"),
		),
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "pageCountLabel"))),
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "componentCountLabel"))),
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "controlCountLabel"))),
			gosx.El("output", nil, gosx.Text(workbenchViewString(siteMapView, "selectedPageLabel"))),
		),
	)
}

func renderSiteMapEngineSourceLegend(siteMapView map[string]any) gosx.Node {
	sources := workbenchViewMapList(siteMapView, "sources")
	if len(sources) == 0 {
		return gosx.Fragment()
	}
	items := make([]gosx.Node, 0, len(sources))
	for _, source := range sources {
		items = append(items, gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-source"),
			gosx.Attr("data-studio-site-map-source-kind", workbenchMapString(source, "key")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(source, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(source, "countLabel"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-site-map-engine__legend"),
		gosx.Attr("aria-label", "Component sources"),
		gosx.Attr("data-gosx-studio-site-map-source-legend-renderer", "gosx-studio"),
	), gosx.Fragment(items...))
}

func renderSiteMapEngineOpenTag(tag string, attrs []any) gosx.Node {
	html := gosx.RenderHTML(gosx.El(tag, gosx.Attrs(attrs...)))
	suffix := "</" + tag + ">"
	html = strings.TrimSuffix(html, suffix)
	return gosx.RawHTML(html)
}
