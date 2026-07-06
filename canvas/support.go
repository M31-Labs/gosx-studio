package canvas

import (
	"net/url"
	"strings"

	"m31labs.dev/gosx/engine"

	"m31labs.dev/gosx-studio/core"
)

// This file carries small, dependency-free engine-capability helpers plus the
// frozen rendered-page-contract constants (hostruntime asset paths, the
// block-layout default engine name, and assetHref) that engine_hosts.go,
// block_layout_engine.go, and sitemap_canvas.go need. canvas may import only
// core (§1 Import DAG) and stays a leaf package, so it cannot reach
// hostruntime (a peer) directly. The former workbenchView*/mapString/mapList/
// nodeEmpty duplicates in this file converged into core in Slice 8 (see
// core/workbench_view.go); callers now use core.Workbench*. The remaining
// frozen values below are guarded against drift by a shell sync-guard test
// (shell/frozen_copy_sync_test.go), which is the only package that may import
// both canvas and hostruntime.

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
