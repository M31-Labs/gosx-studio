package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type InspectorChromePanelOptions struct {
	RootAttrs map[string]any
}

func RenderInspectorChromePanel(view map[string]any, options InspectorChromePanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", "studio-inspector-chrome"),
		gosx.Attr("data-studio-inspector-chrome-panel", "true"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-inspector-chrome-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-inspector-head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
				gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-mode-label", "true")), gosx.Text(core.WorkbenchViewString(view, "modeLabel"))),
			),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("data-studio-selection-label", "true"),
				gosx.Attr("aria-live", "polite"),
			), gosx.Text(core.WorkbenchViewString(view, "selectionLabel"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-scope-strip"),
			gosx.Attr("aria-label", core.WorkbenchViewString(view, "scopeLabel")),
		), gosx.Fragment(renderInspectorChromeCrumbs(core.WorkbenchViewMapList(view, "crumbs"))...)),
	)
}

func renderInspectorChromeCrumbs(crumbs []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(crumbs))
	for _, crumb := range crumbs {
		label := core.WorkbenchViewString(crumb, "label")
		switch {
		case core.WorkbenchViewBool(crumb, "dynamicSelection"):
			nodes = append(nodes, gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(label)))
		case core.WorkbenchViewBool(crumb, "dynamicMode"):
			nodes = append(nodes, gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-mode-label", "true")), gosx.Text(label)))
		default:
			nodes = append(nodes, gosx.El("span", nil, gosx.Text(label)))
		}
	}
	return nodes
}
