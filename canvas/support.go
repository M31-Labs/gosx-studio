package canvas

import (
	"net/url"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/engine"

	"m31labs.dev/gosx-studio/core"
)

// This file carries small, dependency-free helpers that engine_hosts.go,
// block_layout_engine.go, and sitemap_canvas.go need but that still live as
// unexported duplicates in not-yet-migrated root files (workbench_toolbar.go,
// sitemap_authoring_panels.go, sitemap_engine.go — see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md, sitemap/shell rows in
// §1). canvas may import only core (§1 Import DAG), and those root files
// belong to the sitemap (Slice 5) and shell (Slice 8) packages, so canvas
// cannot reach their implementations directly without an import that the DAG
// forbids (a peer/upward import). Rather than force that import, this slice
// carries its own copy, matching the existing precedent of duplicated
// helpers the spec itself documents (§1 "Text helpers" seam) until a later
// slice promotes a single shared home (likely core) and every copy converges.
//
// mapString reads values[key] as a trimmed string, formatting non-string
// values with core.FmtAny. Mirrors workbench_toolbar.go's workbenchMapString.
func mapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
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

// mapList reads values[key] as a []map[string]any, converting a
// []map[string]string shape as needed. Mirrors workbench_toolbar.go's
// workbenchViewMapList and sitemap_authoring_panels.go's siteMapMapList
// (identical logic; that duplication predates this slice).
func mapList(values map[string]any, key string) []map[string]any {
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

// nodeEmpty reports whether a rendered gosx.Node is the empty/zero fragment.
// Mirrors workbench_toolbar.go's workbenchNodeEmpty.
func nodeEmpty(node gosx.Node) bool {
	html := gosx.RenderHTML(node)
	return html == "" || html == "<></>"
}

// engineCapabilityStrings normalizes a loosely-typed capability value (a
// comma/space-separated string, []string, []engine.Capability, []any, or any
// other value formatted via core.FmtAny) into a deduplicated, ordered list of
// capability name strings. Mirrors sitemap_engine.go's
// siteMapEngineCapabilityStrings.
func engineCapabilityStrings(value any) []string {
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

// engineCapabilities converts a loosely-typed capability value into
// []engine.Capability. Mirrors sitemap_engine.go's siteMapEngineCapabilities.
func engineCapabilities(values []string) []engine.Capability {
	names := engineCapabilityStrings(values)
	capabilities := make([]engine.Capability, 0, len(names))
	for _, name := range names {
		capabilities = append(capabilities, engine.Capability(name))
	}
	return capabilities
}

// engineCapabilityNames renders []engine.Capability back to trimmed, non-empty
// strings. Mirrors sitemap_engine.go's siteMapEngineCapabilityNames.
func engineCapabilityNames(values []engine.Capability) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		if name := strings.TrimSpace(string(value)); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// engineHostCapabilities normalizes an engine host's configured capabilities,
// falling back to the supplied default list when none are configured. Mirrors
// sitemap_engine.go's siteMapEngineHostCapabilities.
func engineHostCapabilities(values []string, fallback []string) []string {
	normalized := engineCapabilityStrings(values)
	if len(normalized) > 0 {
		return normalized
	}
	return append([]string(nil), fallback...)
}

// blockLayoutEngineDefaultName is the default engine name rendered into
// data-gosx-engine when a block-layout engine host omits an explicit Name.
// It mirrors the root package's BlockLayoutEngineName constant (shell.go,
// still Slice-8 territory: it is declared alongside ShellConfig and the other
// engine name constants CanvasEngineName/SiteMapEngineName/FlowDesignerName/
// Showcase3DEngineName). canvas cannot import shell (peer import forbidden
// per the DAG), so this is a frozen, byte-identical copy of the same literal
// value rather than a shared reference; keep it in sync with shell.go until a
// later slice unifies the engine-name registry.
const blockLayoutEngineDefaultName = "GoSXStudioBlockLayout"

// The six constants below mirror m31labs.dev/gosx-studio/hostruntime's public
// /_gosx/studio/* runtime script path contract (hostruntime/paths.go):
// CanvasSelectionBridgePath, CanvasInlineEditPath, Canvas2DPainterPath,
// CanvasWASMFreeClientPath, CanvasContextualPanelPath, and
// CanvasDefaultInlineInstallerPath. RenderSiteMapCanvasSurface renders
// <script src=...> tags pointing at these Studio-owned runtime bundles, but
// canvas may import only core (§1 Import DAG) — hostruntime sits beside
// canvas, not below it, so importing it would be a forbidden peer import
// (only shell, Slice 8, is allowed to import both canvas and hostruntime).
// These are frozen rendered-page-contract values (spec risk R5: rendered
// attribute/path names must not change); if hostruntime/paths.go ever
// changes these strings, update this file to match.
const (
	canvasRuntimeRoot                = "/_gosx/studio"
	canvasSelectionBridgePath        = canvasRuntimeRoot + "/canvas-selection-bridge.js"
	canvasInlineEditPath             = canvasRuntimeRoot + "/canvas-inline-edit.js"
	canvas2DPainterPath              = canvasRuntimeRoot + "/canvas2d-painter.js"
	canvasWASMFreeClientPath         = canvasRuntimeRoot + "/canvas-wasm-free-client.js"
	canvasContextualPanelPath        = canvasRuntimeRoot + "/canvas-contextual-panel.js"
	canvasDefaultInlineInstallerPath = canvasRuntimeRoot + "/canvas-default-inline-installer.js"
)

// assetHref appends a cache-busting "?v=" (or "&v=" if the path already
// carries a query string) query parameter to path when version is non-blank,
// mirroring hostruntime.AssetHref. Duplicated locally for the same DAG reason
// as the path constants above: canvas must not import hostruntime.
func assetHref(path string, version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return path
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "v=" + url.QueryEscape(version)
}
