package studio

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/engine"
)

type BlockLayoutEngineRuntime interface {
	Engine(engine.Config, gosx.Node) gosx.Node
}

type BlockLayoutEngineHostOptions struct {
	Key          string
	Name         string
	MountID      string
	Class        string
	Capabilities []string
	EngineSource string
}

type BlockLayoutEngineOptions struct {
	Class        string
	ModePanel    string
	PanelKey     string
	EngineSource string

	Kicker     string
	Title      string
	CountLabel string

	EngineRuntime BlockLayoutEngineRuntime
	EngineHost    BlockLayoutEngineHostOptions

	EngineHostNode gosx.Node
	LayersNode     gosx.Node
	LibraryNode    gosx.Node
}

type BlockLayoutEngineSegments struct {
	RootOpen     gosx.Node
	Header       gosx.Node
	EngineHost   gosx.Node
	LayersOpen   gosx.Node
	LayersClose  gosx.Node
	LibraryOpen  gosx.Node
	LibraryClose gosx.Node
	RootClose    gosx.Node
}

func BlockLayoutEngineHostFromMap(host map[string]any) BlockLayoutEngineHostOptions {
	return BlockLayoutEngineHostOptions{
		Key:          workbenchMapString(host, "key"),
		Name:         workbenchMapString(host, "name"),
		MountID:      workbenchMapString(host, "mountId"),
		Class:        workbenchMapString(host, "class"),
		Capabilities: siteMapEngineCapabilityStrings(host["capabilities"]),
	}
}

func RenderBlockLayoutEngineSegments(view map[string]any, options BlockLayoutEngineOptions) BlockLayoutEngineSegments {
	return BlockLayoutEngineSegments{
		RootOpen:     renderBlockLayoutEngineRootOpen(options),
		Header:       renderBlockLayoutEngineHeader(view, options),
		EngineHost:   RenderBlockLayoutEngineHost(options),
		LayersOpen:   renderBlockLayoutEngineSlotOpen("studio-block-layout-engine__layers", "data-studio-block-layout-layers"),
		LayersClose:  gosx.RawHTML("</div>"),
		LibraryOpen:  renderBlockLayoutEngineSlotOpen("studio-block-layout-engine__library", "data-studio-block-layout-library"),
		LibraryClose: gosx.RawHTML("</div>"),
		RootClose:    gosx.RawHTML("</section>"),
	}
}

func RenderBlockLayoutEngineHost(options BlockLayoutEngineOptions) gosx.Node {
	return renderBlockLayoutEngineHost(options)
}

func RenderBlockLayoutEngine(view map[string]any, options BlockLayoutEngineOptions) gosx.Node {
	segments := RenderBlockLayoutEngineSegments(view, options)
	engineHost := segments.EngineHost
	if !workbenchNodeEmpty(options.EngineHostNode) {
		engineHost = options.EngineHostNode
	}
	return gosx.Fragment(
		segments.RootOpen,
		segments.Header,
		engineHost,
		segments.LayersOpen,
		options.LayersNode,
		segments.LayersClose,
		segments.LibraryOpen,
		options.LibraryNode,
		segments.LibraryClose,
		segments.RootClose,
	)
}

func renderBlockLayoutEngineRootOpen(options BlockLayoutEngineOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", FirstNonEmpty(options.Class, "studio-block-layout-engine")),
		gosx.Attr("data-studio-block-layout-panel", "true"),
		gosx.Attr("data-studio-mode-panel", FirstNonEmpty(options.ModePanel, "home")),
		gosx.Attr("data-studio-panel", FirstNonEmpty(options.PanelKey, "block-layout")),
		gosx.Attr("data-studio-engine-source", FirstNonEmpty(options.EngineSource, "gosx")),
		gosx.Attr("data-gosx-studio-block-layout-engine-renderer", "gosx-studio"),
	}
	return renderBlockLayoutEngineOpenTag("section", attrs)
}

func renderBlockLayoutEngineHeader(view map[string]any, options BlockLayoutEngineOptions) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-block-layout-engine__chrome")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(FirstNonEmpty(options.Kicker, workbenchMapString(view, "kicker"), "Home"))),
			gosx.El("h2", nil, gosx.Text(FirstNonEmpty(options.Title, workbenchMapString(view, "title"), "Sections"))),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", "studio-block-layout-engine__state"),
			gosx.Attr("data-studio-block-layout-state", "true"),
		), gosx.Text(FirstNonEmpty(options.CountLabel, workbenchMapString(view, "countLabel")))),
	)
}

func renderBlockLayoutEngineSlotOpen(className string, flagAttr string) gosx.Node {
	return renderBlockLayoutEngineOpenTag("div", []any{
		gosx.Attr("class", className),
		gosx.Attr(flagAttr, "true"),
	})
}

func renderBlockLayoutEngineOpenTag(tag string, attrs []any) gosx.Node {
	html := gosx.RenderHTML(gosx.El(tag, gosx.Attrs(attrs...)))
	suffix := "</" + tag + ">"
	html = strings.TrimSuffix(html, suffix)
	return gosx.RawHTML(html)
}

func renderBlockLayoutEngineHost(options BlockLayoutEngineOptions) gosx.Node {
	host := normalizeBlockLayoutEngineHost(options.EngineHost, options.EngineSource)
	mountAttrs := map[string]any{
		"class":                                   host.Class,
		"data-gosx-studio-engine":                 host.Key,
		"data-studio-engine-role":                 host.Key,
		"data-studio-engine-source":               host.EngineSource,
		"data-studio-block-layout-engine-surface": "true",
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
		gosx.Attr("data-studio-block-layout-engine-surface", "true"),
	}
	if len(capabilities) > 0 {
		attrs = append(attrs, gosx.Attr("data-gosx-engine-capabilities", strings.Join(siteMapEngineCapabilityNames(capabilities), " ")))
	}
	return gosx.El("div", gosx.Attrs(attrs...))
}

func normalizeBlockLayoutEngineHost(host BlockLayoutEngineHostOptions, engineSource string) BlockLayoutEngineHostOptions {
	defaults := BlockLayoutEngineHostOptions{
		Key:          "block-layout",
		Name:         BlockLayoutEngineName,
		MountID:      "gosx-studio-block-layout-engine",
		Class:        "studio-block-layout-engine-host",
		Capabilities: []string{"pointer", "keyboard", "text-input", "animation"},
		EngineSource: "gosx",
	}
	return BlockLayoutEngineHostOptions{
		Key:          FirstNonEmpty(host.Key, defaults.Key),
		Name:         FirstNonEmpty(host.Name, defaults.Name),
		MountID:      FirstNonEmpty(host.MountID, defaults.MountID),
		Class:        FirstNonEmpty(host.Class, defaults.Class),
		Capabilities: siteMapEngineHostCapabilities(host.Capabilities, defaults.Capabilities),
		EngineSource: FirstNonEmpty(host.EngineSource, engineSource, defaults.EngineSource),
	}
}
