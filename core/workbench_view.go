package core

import (
	"strings"

	"m31labs.dev/gosx"
)

// This file is the single shared home for the small, dependency-free
// view-map coercion helpers that every layer of the studio composition
// pipeline needs (canvas, sitemap, panels, and the shell workbench
// renderers). Before Slice 8 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) these lived as
// byte-identical unexported duplicates named workbenchView*/workbenchMap*/
// workbenchNodeEmpty in workbench_toolbar.go and workbench_frame.go, and were
// frozen-copied into canvas/support.go, sitemap/support.go, and
// panels/support.go because the DAG forbade those leaf/peer packages from
// importing the shell files that owned them. core imports nothing internal and
// is legal from every layer, so it is the correct convergence point the spec's
// "Text helpers" seam (§1) anticipated. All former callers now reference these
// exported names; the frozen copies were deleted this slice.

// WorkbenchViewMap reads view[key] as a map[string]any, converting a
// map[string]string shape as needed.
func WorkbenchViewMap(view map[string]any, key string) map[string]any {
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

// WorkbenchViewBool reads view[key] as a bool, accepting a case-insensitive
// "true" string as truthy.
func WorkbenchViewBool(view map[string]any, key string) bool {
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

// WorkbenchViewString reads view[key] as a trimmed string, formatting
// non-string values with FmtAny.
func WorkbenchViewString(view map[string]any, key string) string {
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
		return strings.TrimSpace(FmtAny(typed))
	}
}

// WorkbenchViewMapList reads view[key] as a []map[string]any, converting a
// []map[string]string shape as needed.
func WorkbenchViewMapList(view map[string]any, key string) []map[string]any {
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
			for k, v := range item {
				next[k] = v
			}
			out = append(out, next)
		}
		return out
	default:
		return nil
	}
}

// WorkbenchNodeEmpty reports whether a rendered gosx.Node is the empty/zero
// fragment.
func WorkbenchNodeEmpty(node gosx.Node) bool {
	html := gosx.RenderHTML(node)
	return html == "" || html == "<></>"
}
