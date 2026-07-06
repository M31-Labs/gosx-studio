// Package hostruntime owns the embedded runtime/asset bundles Studio serves
// over HTTP: the concatenated engine-runtime JS, the workbench/command/state
// runtime scripts, the CSS stylesheet, the canvas island scripts, and the
// public /_gosx/studio/* URL paths and mounting helper for all of them.
//
// This package moved here from the studio root in Slice 3 of the package
// restructure (see .tiller/scratch/gosx-studio-restructure-spec-v0.1.md,
// risk R2). The assets/ directory MUST live alongside this file because
// go:embed paths are package-dir-relative and cannot reference a parent
// directory.
package hostruntime

import (
	"bytes"
	"embed"
	"net/http"
	"net/url"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/authoringruntime"
	"m31labs.dev/gosx-studio/blocklayoutruntime"
	"m31labs.dev/gosx-studio/brandruntime"
	"m31labs.dev/gosx-studio/canvascontextualpanelruntime"
	"m31labs.dev/gosx-studio/canvasinlineeditruntime"
	"m31labs.dev/gosx-studio/canvasselectionbridgeruntime"
	"m31labs.dev/gosx-studio/canvaswasmfreeruntime"
	"m31labs.dev/gosx-studio/fieldruntime"
	"m31labs.dev/gosx-studio/inlineeditruntime"
	"m31labs.dev/gosx-studio/inspectorruntime"
	"m31labs.dev/gosx-studio/previewruntime"
	"m31labs.dev/gosx-studio/selectionruntime"
	"m31labs.dev/gosx-studio/sitemapruntime"
	"m31labs.dev/gosx-studio/styleruntime"
	"m31labs.dev/gosx-studio/workbenchruntime"
)

//go:embed assets/studio.css assets/workbench_runtime.js assets/command_palette.js assets/state_runtime.js
var runtimeFS embed.FS

// EngineRuntimeScript returns the concatenated island runtime bundles for
// every Phase 3 slice. The legacy monolithic JS bundles (studio-engines.js
// and preview-runtime.js) were deleted on 2026-05-27 — the island paths
// now own every runtime contract by default. See
// ~/.hyphae/spaces/m31labs-gosx/plans/gosx-vm-unification-and-editor-bridge-burn-down.md
// Phase 3 Section E for the burn-down history.
func EngineRuntimeScript() []byte {
	return concatRuntimeSlices(builtinRuntimeSlices())
}

// EngineRuntimeScriptWithPlugins returns the built-in island bundle followed by
// every client island contributed by plugins registered in reg, in registration
// order. With a nil or empty registry the output is byte-identical to
// EngineRuntimeScript(), so non-plugin hosts are unaffected. A plugin-aware host
// serves this instead of EngineRuntimeScript() and mounts reg.IslandRoutes().
func EngineRuntimeScriptWithPlugins(reg *PluginRegistry) []byte {
	return concatRuntimeSlices(append(builtinRuntimeSlices(), reg.IslandBundles()...))
}

// builtinRuntimeSlices returns the ordered built-in island bundles. Extracted so
// EngineRuntimeScript and EngineRuntimeScriptWithPlugins share one source of
// truth for the built-in set and its order.
func builtinRuntimeSlices() [][]byte {
	return [][]byte{
		fieldruntime.Bundle(),
		selectionruntime.Bundle(),
		brandruntime.Bundle(),
		blocklayoutruntime.Bundle(),
		styleruntime.Bundle(),
		workbenchruntime.Bundle(),
		previewruntime.Bundle(),
		authoringruntime.Bundle(),
		sitemapruntime.Bundle(),
		inspectorruntime.Bundle(),
		inlineeditruntime.Bundle(),
	}
}

// concatRuntimeSlices joins non-empty island bundles, each terminated by a
// newline (the historical engine-bundle framing).
func concatRuntimeSlices(slices [][]byte) []byte {
	total := 0
	for _, slice := range slices {
		if len(slice) > 0 {
			total += len(slice) + 1
		}
	}
	out := make([]byte, 0, total)
	for _, slice := range slices {
		if len(slice) == 0 {
			continue
		}
		out = append(out, slice...)
		out = append(out, '\n')
	}
	return out
}

func Stylesheet() []byte {
	return readRuntime("assets/studio.css")
}

func EngineRuntimeHandler() http.Handler {
	return ScriptHandler("studio-engines.js", EngineRuntimeScript())
}

// EngineRuntimeHandlerWithPlugins serves the engine bundle including the islands
// contributed by plugins registered in reg. A plugin-aware host mounts this in
// place of EngineRuntimeHandler() and also mounts reg.IslandRoutes() so each
// island is additionally reachable on its own URL.
func EngineRuntimeHandlerWithPlugins(reg *PluginRegistry) http.Handler {
	return ScriptHandler("studio-engines.js", EngineRuntimeScriptWithPlugins(reg))
}

// CanvasInlineEditScript returns the CanvasBoard inline-edit bridge. The bridge
// delegates commit behavior to GoSXStudioInlineEditRuntime and owns only the
// CanvasBoard repaint-safe source-of-truth refresh.
func CanvasInlineEditScript() []byte {
	return canvasinlineeditruntime.CanvasInlineEditScript()
}

// CanvasDefaultInlineInstallerScript returns the default CanvasBoard inline-edit
// installer that scopes installation to the marked default editor CanvasBoard.
func CanvasDefaultInlineInstallerScript() []byte {
	return canvasinlineeditruntime.CanvasDefaultInlineInstallerScript()
}

// CanvasContextualPanelScript returns the CanvasBoard contextual inspector
// panel. The panel persists field edits through the Studio Canvas inline-edit
// runtime API.
func CanvasContextualPanelScript() []byte {
	return canvascontextualpanelruntime.CanvasContextualPanelScript()
}

// CanvasSelectionBridgeScript returns the opt-in full-WASM CanvasBoard
// selection bridge. It publishes window.GoSXStudioCanvasSelectionBridgeRuntime.
func CanvasSelectionBridgeScript() []byte {
	return canvasselectionbridgeruntime.CanvasSelectionBridgeScript()
}

// Canvas2DPainterScript returns the WASM-free Canvas2D painter. It publishes
// window.GoSXStudioCanvas2DPainterRuntime.
func Canvas2DPainterScript() []byte {
	return canvaswasmfreeruntime.Canvas2DPainterScript()
}

// CanvasWASMFreeClientScript returns the WASM-free CanvasBoard browser client.
// It publishes window.GoSXStudioCanvasWASMFreeClientRuntime.
func CanvasWASMFreeClientScript() []byte {
	return canvaswasmfreeruntime.CanvasWASMFreeClientScript()
}

func CanvasInlineEditHandler() http.Handler {
	return ScriptHandler("canvas-inline-edit.js", CanvasInlineEditScript())
}

func CanvasDefaultInlineInstallerHandler() http.Handler {
	return ScriptHandler("canvas-default-inline-installer.js", CanvasDefaultInlineInstallerScript())
}

func CanvasContextualPanelHandler() http.Handler {
	return ScriptHandler("canvas-contextual-panel.js", CanvasContextualPanelScript())
}

func CanvasSelectionBridgeHandler() http.Handler {
	return ScriptHandler("canvas-selection-bridge.js", CanvasSelectionBridgeScript())
}

func Canvas2DPainterHandler() http.Handler {
	return ScriptHandler("canvas2d-painter.js", Canvas2DPainterScript())
}

func CanvasWASMFreeClientHandler() http.Handler {
	return ScriptHandler("canvas-wasm-free-client.js", CanvasWASMFreeClientScript())
}

// PreviewSubscriberScript returns the JS bundle the storefront mounts
// when loaded inside the editor preview iframe. Unlike the editor-side
// engine bundle, this script is intended for the storefront page —
// public storefront visitors never load it (the iframe's preview-mode
// bootstrap per ADR 0009 is gated on ?gosx-preview=1 query param).
//
// The bundle wires window.__gosx_observe_shared_signal subscribers
// against the $preview.* signal namespace; the cross-frame postMessage
// relay (gosx/client/bridge/cross_frame.go) delivers writes from the
// editor frame, and the subscribers apply the corresponding DOM
// mutations locally inside the iframe.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-6-previewruntime.md
// Section F and the package docstring at
// gosx-studio/previewruntime/runtime.go for the contract.
func PreviewSubscriberScript() []byte {
	return previewruntime.PreviewSubscriberScript()
}

// WorkbenchRuntimeScript returns the legacy Studio workbench chrome runtime.
// It remains separate from EngineRuntimeScript because existing hosts mount it
// through WorkbenchRuntimePath while backend admin/editor ownership moves into
// this module.
func WorkbenchRuntimeScript() []byte {
	return readRuntime("assets/workbench_runtime.js")
}

// CommandRuntimeScript returns the Studio command palette runtime.
func CommandRuntimeScript() []byte {
	return readRuntime("assets/command_palette.js")
}

// StateRuntimeScript returns the Studio save/state indicator runtime.
func StateRuntimeScript() []byte {
	return readRuntime("assets/state_runtime.js")
}

// AuthoringRuntimeScript returns the browser runtime that reacts to successful
// Studio authoring action results. It is included in EngineRuntimeScript for
// hosts that serve the Studio engine bundle, and is exposed separately for
// hosts that still compose their editor chrome from gosx-cms/studio helpers.
func AuthoringRuntimeScript() []byte {
	return authoringruntime.Bundle()
}

// SiteMapRuntimeScript returns the browser runtime that owns site-map board
// filters, detail tabs, palette state, and selection synchronization for hosts
// that have not yet switched to the full Studio engine bundle.
func SiteMapRuntimeScript() []byte {
	return sitemapruntime.Bundle()
}

// PreviewSubscriberHandler serves PreviewSubscriberScript() at
// shell.PreviewSubscriberPath. Storefront pages reference this URL in
// a <script defer> tag emitted by the preview-mode bootstrap.
func PreviewSubscriberHandler() http.Handler {
	return ScriptHandler("preview-subscriber.js", PreviewSubscriberScript())
}

func WorkbenchRuntimeHandler() http.Handler {
	return ScriptHandler("workbench-runtime.js", WorkbenchRuntimeScript())
}

func CommandRuntimeHandler() http.Handler {
	return ScriptHandler("command-palette.js", CommandRuntimeScript())
}

func StateRuntimeHandler() http.Handler {
	return ScriptHandler("state-runtime.js", StateRuntimeScript())
}

func StylesheetHandler() http.Handler {
	return AssetHandler("studio.css", Stylesheet(), "text/css; charset=utf-8")
}

func ScriptHandler(name string, data []byte) http.Handler {
	return AssetHandler(name, data, "text/javascript; charset=utf-8")
}

func AssetHandler(name string, data []byte, contentType string) http.Handler {
	modtime := RuntimeAssetModTime()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, name, modtime, bytes.NewReader(data))
	})
}

func RuntimeAssetModTime() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func AssetHref(path string, version string) string {
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

func readRuntime(path string) []byte {
	data, err := runtimeFS.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}
