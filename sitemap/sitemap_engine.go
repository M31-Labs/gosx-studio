package sitemap

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
	"m31labs.dev/gosx/engine"
)

type SiteMapEngineRuntime interface {
	Engine(engine.Config, gosx.Node) gosx.Node
}

type SiteMapEngineHostOptions struct {
	Key          string
	Name         string
	MountID      string
	Class        string
	Capabilities []string
	EngineSource string
}

type SiteMapEngineOptions struct {
	Class        string
	ModePanel    string
	PanelKey     string
	EngineSource string
	LayoutAction string

	EngineRuntime SiteMapEngineRuntime
	EngineHost    SiteMapEngineHostOptions

	Board           SiteMapBoardOptions
	AuthoringPanels SiteMapAuthoringPanelsOptions
	AuthoringForms  SiteMapAuthoringFormsOptions

	HideEngineHost      bool
	HideBoard           bool
	HideAuthoringPanels bool
	HideAuthoringForms  bool
}

type SiteMapEngineSegments struct {
	RootOpen     gosx.Node
	Header       gosx.Node
	SourceLegend gosx.Node
	RootClose    gosx.Node
}

type SiteMapEngineRender struct {
	Surface gosx.Node
	Forms   gosx.Node
}

func SiteMapEngineHostFromMap(host map[string]any) SiteMapEngineHostOptions {
	return SiteMapEngineHostOptions{
		Key:          workbenchMapString(host, "key"),
		Name:         workbenchMapString(host, "name"),
		MountID:      workbenchMapString(host, "mountId"),
		Class:        workbenchMapString(host, "class"),
		Capabilities: siteMapEngineCapabilityStrings(host["capabilities"]),
	}
}

func RenderSiteMapEngine(siteMapView map[string]any, options SiteMapEngineOptions) SiteMapEngineRender {
	segments := RenderSiteMapEngineSegments(siteMapView, options)
	surfaceNodes := []gosx.Node{
		segments.RootOpen,
		segments.Header,
	}
	if !options.HideAuthoringPanels {
		surfaceNodes = append(surfaceNodes, RenderSiteMapAuthoringPanels(siteMapView, options.AuthoringPanels))
	}
	surfaceNodes = append(surfaceNodes, segments.SourceLegend)
	if !options.HideEngineHost {
		surfaceNodes = append(surfaceNodes, renderSiteMapEngineHost(options))
	}
	if !options.HideBoard {
		surfaceNodes = append(surfaceNodes, RenderSiteMapBoard(siteMapView, options.Board))
	}
	surfaceNodes = append(surfaceNodes, segments.RootClose)

	forms := gosx.Fragment()
	if !options.HideAuthoringForms {
		forms = RenderSiteMapAuthoringForms(siteMapView, options.AuthoringForms)
	}

	return SiteMapEngineRender{
		Surface: gosx.Fragment(surfaceNodes...),
		Forms:   forms,
	}
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
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-site-map-engine")),
		gosx.Attr("data-studio-site-map-panel", "true"),
		gosx.Attr("data-studio-mode-panel", core.FirstNonEmpty(options.ModePanel, "home")),
		gosx.Attr("data-studio-panel", core.FirstNonEmpty(options.PanelKey, "site-map")),
		gosx.Attr("data-studio-engine-source", core.FirstNonEmpty(options.EngineSource, "gosx")),
		gosx.Attr("data-gosx-studio-site-map-engine-renderer", "gosx-studio"),
	}
	if strings.TrimSpace(options.LayoutAction) != "" {
		attrs = append(attrs, gosx.Attr("data-gosx-studio-site-map-layout-action", strings.TrimSpace(options.LayoutAction)))
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

func renderSiteMapEngineHost(options SiteMapEngineOptions) gosx.Node {
	host := normalizeSiteMapEngineHost(options.EngineHost, options.EngineSource)
	mountAttrs := map[string]any{
		"class":                               host.Class,
		"data-gosx-studio-engine":             host.Key,
		"data-studio-engine-role":             host.Key,
		"data-studio-engine-source":           host.EngineSource,
		"data-studio-site-map-engine-surface": "true",
	}
	capabilities := siteMapEngineCapabilities(host.Capabilities)
	if options.EngineRuntime != nil {
		return options.EngineRuntime.Engine(engine.Config{
			Name:         host.Name,
			Kind:         engine.KindSurface,
			MountID:      host.MountID,
			MountAttrs:   mountAttrs,
			Capabilities: capabilities,
		}, gosx.Node{})
	}

	attrs := []any{
		gosx.Attr("id", host.MountID),
		gosx.Attr("class", host.Class),
		gosx.Attr("data-gosx-engine", host.Name),
		gosx.Attr("data-gosx-engine-kind", string(engine.KindSurface)),
		gosx.Attr("data-gosx-studio-engine", host.Key),
		gosx.Attr("data-studio-engine-role", host.Key),
		gosx.Attr("data-studio-engine-source", host.EngineSource),
		gosx.Attr("data-studio-site-map-engine-surface", "true"),
	}
	if len(capabilities) > 0 {
		attrs = append(attrs, gosx.Attr("data-gosx-engine-capabilities", strings.Join(SiteMapEngineCapabilityNames(capabilities), " ")))
	}
	return gosx.El("div", gosx.Attrs(attrs...))
}

func normalizeSiteMapEngineHost(host SiteMapEngineHostOptions, engineSource string) SiteMapEngineHostOptions {
	defaults := SiteMapEngineHostOptions{
		Key:          "site-map",
		Name:         siteMapEngineDefaultName,
		MountID:      "gosx-studio-site-map-engine",
		Class:        "studio-site-map-engine-host",
		Capabilities: []string{"canvas", "pointer", "keyboard", "text-input", "animation"},
		EngineSource: "gosx",
	}
	return SiteMapEngineHostOptions{
		Key:          core.FirstNonEmpty(host.Key, defaults.Key),
		Name:         core.FirstNonEmpty(host.Name, defaults.Name),
		MountID:      core.FirstNonEmpty(host.MountID, defaults.MountID),
		Class:        core.FirstNonEmpty(host.Class, defaults.Class),
		Capabilities: siteMapEngineHostCapabilities(host.Capabilities, defaults.Capabilities),
		EngineSource: core.FirstNonEmpty(host.EngineSource, engineSource, defaults.EngineSource),
	}
}

func siteMapEngineHostCapabilities(values []string, fallback []string) []string {
	normalized := siteMapEngineCapabilityStrings(values)
	if len(normalized) > 0 {
		return normalized
	}
	return append([]string(nil), fallback...)
}

func siteMapEngineCapabilities(values []string) []engine.Capability {
	names := siteMapEngineCapabilityStrings(values)
	capabilities := make([]engine.Capability, 0, len(names))
	for _, name := range names {
		capabilities = append(capabilities, engine.Capability(name))
	}
	return capabilities
}

func SiteMapEngineCapabilityNames(values []engine.Capability) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		if name := strings.TrimSpace(string(value)); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func siteMapEngineCapabilityStrings(value any) []string {
	out := []string{}
	seen := map[string]bool{}
	appendField := func(field string) {
		field = strings.TrimSpace(field)
		if field == "" || seen[field] {
			return
		}
		seen[field] = true
		out = append(out, field)
	}
	appendFields := func(raw string) {
		raw = strings.ReplaceAll(raw, ",", " ")
		for _, field := range strings.Fields(raw) {
			appendField(field)
		}
	}

	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		appendFields(typed)
	case []string:
		for _, item := range typed {
			appendFields(item)
		}
	case []engine.Capability:
		for _, item := range typed {
			appendField(string(item))
		}
	case []any:
		for _, item := range typed {
			appendFields(core.FmtAny(item))
		}
	default:
		appendFields(core.FmtAny(typed))
	}
	return out
}
