package canvas

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/engine"

	"m31labs.dev/gosx-studio/core"
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
		Key:          core.WorkbenchViewString(host, "key"),
		Name:         core.WorkbenchViewString(host, "name"),
		MountID:      core.WorkbenchViewString(host, "mountId"),
		Class:        core.WorkbenchViewString(host, "class"),
		Capabilities: engineCapabilityStrings(host["capabilities"]),
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
	if !core.WorkbenchNodeEmpty(options.EngineHostNode) {
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
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-block-layout-engine")),
		gosx.Attr("data-studio-block-layout-panel", "true"),
		gosx.Attr("data-studio-mode-panel", core.FirstNonEmpty(options.ModePanel, "home")),
		gosx.Attr("data-studio-panel", core.FirstNonEmpty(options.PanelKey, "block-layout")),
		gosx.Attr("data-studio-engine-source", core.FirstNonEmpty(options.EngineSource, "gosx")),
		gosx.Attr("data-gosx-studio-block-layout-engine-renderer", "gosx-studio"),
	}
	return RenderBlockLayoutEngineOpenTag("section", attrs)
}

func renderBlockLayoutEngineHeader(view map[string]any, options BlockLayoutEngineOptions) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-block-layout-engine__chrome")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.FirstNonEmpty(options.Kicker, core.WorkbenchViewString(view, "kicker"), "Home"))),
			gosx.El("h2", nil, gosx.Text(core.FirstNonEmpty(options.Title, core.WorkbenchViewString(view, "title"), "Sections"))),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", "studio-block-layout-engine__state"),
			gosx.Attr("data-studio-block-layout-state", "true"),
		), gosx.Text(core.FirstNonEmpty(options.CountLabel, core.WorkbenchViewString(view, "countLabel")))),
	)
}

func renderBlockLayoutEngineSlotOpen(className string, flagAttr string) gosx.Node {
	return RenderBlockLayoutEngineOpenTag("div", []any{
		gosx.Attr("class", className),
		gosx.Attr(flagAttr, "true"),
	})
}

// RenderBlockLayoutEngineOpenTag renders tag with attrs and strips its
// closing tag, producing an "open tag" fragment. It is exported (rather than
// package-private, matching its original studio-root name) because three
// still-root-resident panel files (brand_panel.go, home_layers_panel.go,
// advanced_panel.go — panels package territory, Slice 6) call it by its
// original unqualified name via the compat_canvas.go shim var; see that
// file's "Unexported shims" section.
func RenderBlockLayoutEngineOpenTag(tag string, attrs []any) gosx.Node {
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
	capabilities := engineCapabilities(host.Capabilities)
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
		attrs = append(attrs, gosx.Attr("data-gosx-engine-capabilities", strings.Join(engineCapabilityNames(capabilities), " ")))
	}
	return gosx.El("div", gosx.Attrs(attrs...))
}

func normalizeBlockLayoutEngineHost(host BlockLayoutEngineHostOptions, engineSource string) BlockLayoutEngineHostOptions {
	defaults := BlockLayoutEngineHostOptions{
		Key:          "block-layout",
		Name:         blockLayoutEngineDefaultName,
		MountID:      "gosx-studio-block-layout-engine",
		Class:        "studio-block-layout-engine-host",
		Capabilities: []string{"pointer", "keyboard", "text-input", "animation"},
		EngineSource: "gosx",
	}
	return BlockLayoutEngineHostOptions{
		Key:          core.FirstNonEmpty(host.Key, defaults.Key),
		Name:         core.FirstNonEmpty(host.Name, defaults.Name),
		MountID:      core.FirstNonEmpty(host.MountID, defaults.MountID),
		Class:        core.FirstNonEmpty(host.Class, defaults.Class),
		Capabilities: engineHostCapabilities(host.Capabilities, defaults.Capabilities),
		EngineSource: core.FirstNonEmpty(host.EngineSource, engineSource, defaults.EngineSource),
	}
}
