package studio

import (
	"sort"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/engine"
)

type StudioEngineHostsOptions struct {
	Class         string
	EngineSource  string
	RootAttrs     map[string]any
	EngineRuntime StudioEngineRuntime
}

type StudioEngineRuntime interface {
	Engine(engine.Config, gosx.Node) gosx.Node
}

func RenderStudioEngineHosts(hosts []map[string]any, options StudioEngineHostsOptions) gosx.Node {
	children := make([]gosx.Node, 0, len(hosts))
	for _, host := range hosts {
		children = append(children, renderStudioEngineHost(host, options))
	}
	return gosx.El("div", studioEngineHostsRootAttrs(options), gosx.Fragment(children...))
}

func studioEngineHostsRootAttrs(options StudioEngineHostsOptions) gosx.AttrList {
	defaults := map[string]any{
		"class":                                  FirstNonEmpty(options.Class, "studio-engine-hosts"),
		"aria-hidden":                            "true",
		"data-gosx-studio-engines":               "true",
		"data-gosx-studio-engine-hosts-renderer": "gosx-studio",
	}
	ordered := []string{
		"class",
		"aria-hidden",
		"data-gosx-studio-engines",
		"data-gosx-studio-engine-hosts-renderer",
	}

	attrs := make([]any, 0, len(defaults)+len(options.RootAttrs))
	for _, name := range ordered {
		value := defaults[name]
		if override, ok := options.RootAttrs[name]; ok {
			value = override
		}
		attrs = append(attrs, gosx.Attr(name, value))
	}

	extraNames := make([]string, 0, len(options.RootAttrs))
	for name := range options.RootAttrs {
		if _, ok := defaults[name]; ok {
			continue
		}
		extraNames = append(extraNames, name)
	}
	sort.Strings(extraNames)
	for _, name := range extraNames {
		attrs = append(attrs, gosx.Attr(name, options.RootAttrs[name]))
	}
	return gosx.Attrs(attrs...)
}

func renderStudioEngineHost(host map[string]any, options StudioEngineHostsOptions) gosx.Node {
	key := workbenchMapString(host, "key")
	name := workbenchMapString(host, "name")
	mountID := workbenchMapString(host, "mountId")
	className := workbenchMapString(host, "class")
	engineSource := FirstNonEmpty(workbenchMapString(host, "engineSource"), options.EngineSource, "gosx")
	capabilities := siteMapEngineCapabilities(siteMapEngineCapabilityStrings(host["capabilities"]))
	mountAttrs := map[string]any{
		"class":                     className,
		"data-gosx-studio-engine":   key,
		"data-studio-engine-role":   key,
		"data-studio-engine-source": engineSource,
	}
	if options.EngineRuntime != nil {
		return options.EngineRuntime.Engine(engine.Config{
			Name:         name,
			Kind:         engine.KindSurface,
			MountID:      mountID,
			MountAttrs:   mountAttrs,
			Capabilities: capabilities,
		}, gosx.Node{})
	}

	attrs := []any{
		gosx.Attr("id", mountID),
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-engine", name),
		gosx.Attr("data-gosx-engine-id", FirstNonEmpty("static-"+NormalizeKey(key), "static-engine")),
		gosx.Attr("data-gosx-engine-kind", string(engine.KindSurface)),
		gosx.Attr("data-gosx-enhance", "engine"),
		gosx.Attr("data-gosx-enhance-layer", "runtime"),
		gosx.Attr("data-gosx-fallback", "none"),
		gosx.Attr("data-gosx-studio-engine", key),
		gosx.Attr("data-studio-engine-role", key),
		gosx.Attr("data-studio-engine-source", engineSource),
	}
	if len(capabilities) > 0 {
		attrs = append(attrs, gosx.Attr("data-gosx-engine-capabilities", strings.Join(siteMapEngineCapabilityNames(capabilities), " ")))
	}
	return gosx.El("div", gosx.Attrs(attrs...))
}
