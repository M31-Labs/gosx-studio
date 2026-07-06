package panels

// This file carries small, dependency-free helpers that
// activity_panel.go, advanced_panel.go, brand_panel.go, checkout_panel.go,
// flow_designer_panel.go, home_inspector_panel.go, home_layers_panel.go,
// inspector.go, inspector_fields.go, look_panel.go, navigation_panel.go,
// preview_share_panel.go, publish_panel.go, revision_history_panel.go, and
// style_panel.go need but that still live as originals in not-yet-migrated
// root files (workbench_toolbar.go, workbench_frame.go — see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md, shell row in §1) or
// that now live as exported originals in the sitemap package
// (sitemap_authoring_panels.go's SiteMapHiddenInputs/SiteMapInputViews/
// SiteMapMapList, Slice 5). panels may import only core and authoring (§1
// Import DAG); shell is Slice 8 territory and sitemap sits beside panels as
// a peer, so importing either would be a forbidden peer/upward import.
// Rather than force that import, this slice carries its own copy, matching
// the existing precedent of duplicated helpers the spec itself documents
// (§1 "Text helpers" seam, and canvas/support.go + sitemap/support.go's
// identical treatment of this same map-shape family for Slices 4 and 5)
// until a later slice promotes a single shared home and every copy
// converges.

import (
	"strings"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

// workbenchViewMap reads values[key] as a map[string]any, converting a
// map[string]string shape as needed. Mirrors workbench_frame.go's
// workbenchViewMap.
func workbenchViewMap(view map[string]any, key string) map[string]any {
	if view == nil {
		return nil
	}
	switch typed := view[key].(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = value
		}
		return out
	default:
		return nil
	}
}

// workbenchViewBool reads values[key] as a bool, accepting a case-insensitive
// "true" string as truthy. Mirrors workbench_frame.go's workbenchViewBool
// (identical to workbench_toolbar.go's workbenchMapBool).
func workbenchViewBool(view map[string]any, key string) bool {
	if view == nil {
		return false
	}
	switch typed := view[key].(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

// workbenchMapBool is the workbench_toolbar.go-named twin of
// workbenchViewBool; the panel files call both bare names.
func workbenchMapBool(values map[string]any, key string) bool {
	return workbenchViewBool(values, key)
}

// workbenchViewString reads values[key] as a trimmed string, formatting
// non-string values with core.FmtAny. Mirrors workbench_toolbar.go's
// workbenchViewString.
func workbenchViewString(view map[string]any, key string) string {
	if view == nil {
		return ""
	}
	value, ok := view[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(core.FmtAny(typed))
	}
}

// workbenchMapString is the workbench_toolbar.go-named twin of
// workbenchViewString; the panel files call both bare names.
func workbenchMapString(values map[string]any, key string) string {
	return workbenchViewString(values, key)
}

// workbenchViewMapList reads values[key] as a []map[string]any, converting a
// []map[string]string shape as needed. Mirrors workbench_toolbar.go's
// workbenchViewMapList.
func workbenchViewMapList(view map[string]any, key string) []map[string]any {
	if view == nil {
		return nil
	}
	switch typed := view[key].(type) {
	case []map[string]any:
		return typed
	case []map[string]string:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			next := make(map[string]any, len(item))
			for key, value := range item {
				next[key] = value
			}
			out = append(out, next)
		}
		return out
	default:
		return nil
	}
}

// workbenchNodeEmpty reports whether a rendered gosx.Node is the empty/zero
// fragment. Mirrors workbench_toolbar.go's workbenchNodeEmpty.
func workbenchNodeEmpty(node gosx.Node) bool {
	html := gosx.RenderHTML(node)
	return html == "" || html == "<></>"
}

// renderBlockLayoutEngineOpenTag is a frozen, byte-identical copy of
// canvas.RenderBlockLayoutEngineOpenTag (canvas/block_layout_engine.go,
// Slice 4). advanced_panel.go, brand_panel.go, and home_layers_panel.go call
// it by its original unqualified name, but panels sits beside canvas (not
// below it) in the Import DAG, so importing it would be a forbidden peer
// import. Reconcile when a later slice gives this a shared home (mirrors
// compat_canvas.go's identical shim treatment of this exact function at the
// root facade layer).
func renderBlockLayoutEngineOpenTag(tag string, attrs []any) gosx.Node {
	html := gosx.RenderHTML(gosx.El(tag, gosx.Attrs(attrs...)))
	suffix := "</" + tag + ">"
	html = strings.TrimSuffix(html, suffix)
	return gosx.RawHTML(html)
}

// siteMapHiddenInputs, siteMapInputViews, and siteMapMapList are frozen,
// byte-identical-in-behavior copies of sitemap.SiteMapHiddenInputs,
// sitemap.SiteMapInputViews, and sitemap.SiteMapMapList
// (sitemap/sitemap_authoring_panels.go, Slice 5). inspector.go's
// RenderInspectorControl and inspector_fields.go need the same hidden-input
// and map-shape helpers the site-map's editable-control panel uses, but
// panels sits beside sitemap (not below it) in the Import DAG, so importing
// it would be a forbidden peer import. Reconcile when a later slice gives
// these a shared home (mirrors sitemap/support.go's identical treatment of
// this exact seam, in reverse, from Slice 5).

func siteMapHiddenInputs(formID string, inputs []map[string]string, exclude ...string) []gosx.Node {
	excluded := map[string]bool{}
	for _, name := range exclude {
		name = strings.TrimSpace(name)
		if name != "" {
			excluded[name] = true
		}
	}
	nodes := make([]gosx.Node, 0, len(inputs))
	for _, input := range inputs {
		name := strings.TrimSpace(input["name"])
		if name == "" || excluded[name] {
			continue
		}
		nodes = append(nodes, gosx.El("input", gosx.Attrs(
			gosx.Attr("form", formID),
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", name),
			gosx.Attr("value", input["value"]),
		)))
	}
	return nodes
}

func siteMapInputViews(values map[string]any, key string) []map[string]string {
	switch typed := values[key].(type) {
	case []map[string]string:
		return typed
	case []map[string]any:
		out := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, map[string]string{
				"name":  workbenchMapString(item, "name"),
				"value": workbenchMapString(item, "value"),
			})
		}
		return out
	default:
		return nil
	}
}

func siteMapMapList(values map[string]any, key string) []map[string]any {
	if values == nil {
		return nil
	}
	switch typed := values[key].(type) {
	case []map[string]any:
		return typed
	case []map[string]string:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			next := make(map[string]any, len(item))
			for key, value := range item {
				next[key] = value
			}
			out = append(out, next)
		}
		return out
	default:
		return nil
	}
}
