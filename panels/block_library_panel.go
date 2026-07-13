package panels

import (
	"sort"
	"strings"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

type BlockLibraryPanelOptions struct {
	RootAttrs map[string]any
}

// blockLibraryPanelAnchorID is the stable DOM id the section manager's "+ Add
// section" link (panels/home_layers_panel.go) jumps to. It is a plain anchor
// (href="#id"), so revealing this panel needs no script — the browser
// scrolls it into view natively. There is exactly one home-mode block
// library panel per rendered page, so a fixed id is safe.
const blockLibraryPanelAnchorID = "gosx-studio-block-library"

func RenderBlockLibraryPanel(view map[string]any, options BlockLibraryPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("id", blockLibraryPanelAnchorID),
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(view, "key")),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(view, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-gosx-studio-block-library-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Add a section")),
		gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
	}
	items := core.WorkbenchViewMapList(view, "items")
	if len(items) == 0 && !core.WorkbenchViewBool(view, "hasItems") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(core.WorkbenchViewString(view, "empty"))))
	}
	if len(items) > 0 || core.WorkbenchViewBool(view, "hasItems") {
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(view, "listClass"))), gosx.Fragment(renderBlockLibraryPanelItems(items)...)))
	}
	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func renderBlockLibraryPanelItems(items []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		attrs := core.WorkbenchViewMap(item, "attrs")
		children := []gosx.Node{}
		// A leading "+" only reads as an add affordance when the section isn't
		// already active (aria-pressed=true means "added"/toggled-on, where the
		// button's own label already communicates the current state).
		if !core.WorkbenchViewBool(attrs, "aria-pressed") {
			children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", "icon"), gosx.Attr("aria-hidden", "true")), gosx.Text("+")))
		}
		children = append(children,
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(item, "buttonLabel"))),
		)
		nodes = append(nodes, gosx.El("button", gosx.Attrs(BlockLibraryPanelMapAttrs(attrs)...), gosx.Fragment(children...)))
	}
	return nodes
}

func appendBlockLibraryPanelAttrs(attrs []any, values map[string]any) []any {
	return append(attrs, BlockLibraryPanelMapAttrs(values)...)
}

// BlockLibraryPanelMapAttrs is exported (rather than package-private,
// matching its original studio-root name) because workbench_frame.go
// (shell package territory, Slice 8) still calls it by its original
// unqualified name via the compat_panels.go shim var; see that file's
// "Unexported shims" section.
func BlockLibraryPanelMapAttrs(values map[string]any) []any {
	attrs := make([]any, 0, len(values))
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		value := values[key]
		if value == nil {
			continue
		}
		if typed, ok := value.(bool); ok && (strings.HasPrefix(name, "aria-") || strings.HasPrefix(name, "data-")) {
			value = core.BoolAttr(typed)
		}
		attrs = append(attrs, gosx.Attr(name, value))
	}
	return attrs
}
