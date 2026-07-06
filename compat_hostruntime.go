package studio

// This file is the deprecated compatibility facade for the symbols that used
// to live in runtime.go, host_runtime.go, plugin_v2.go, and the runtime path
// constants in shell.go before Slice 3 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) extracted them into
// the m31labs.dev/gosx-studio/hostruntime package. The assets/ directory
// (go:embed source for the CSS/JS bundles) moved alongside runtime.go into
// hostruntime/assets, since go:embed paths are package-dir-relative and
// cannot reference a parent directory (risk R2).
//
// Every exported type below is a type ALIAS (not a new type), so struct
// literals, embedding, method sets, and assignability are all unchanged for
// existing callers (muddy-noni-commerce, pajaritos-forest-school,
// gosx-cms/studio). Every exported free function is a thin forwarding
// wrapper. Methods on the aliased types are inherited automatically through
// the alias and do not need wrappers here. Every exported constant keeps its
// exact string value (the /_gosx/studio/* URL paths are part of the
// rendered-page contract and must not change).
//
// Deprecated: import m31labs.dev/gosx-studio/hostruntime directly; this
// facade will be removed after the v0.6.x compatibility window (see spec
// §9).
//
// RuntimeConfig stays defined at the studio root (runtime_config.go) rather
// than moving to hostruntime in this slice: it embeds ShellConfig, which is
// shell-package territory (Slice 8) and still lives only at this root
// package today, and hostruntime must not import the root package.

import (
	"net/http"
	"time"

	"m31labs.dev/gosx-studio/hostruntime"
)

// --- Public runtime asset paths (hostruntime/paths.go) ---

const (
	// Deprecated: use hostruntime.RuntimeRoot.
	RuntimeRoot = hostruntime.RuntimeRoot
	// Deprecated: use hostruntime.StylesheetPath.
	StylesheetPath = hostruntime.StylesheetPath
	// Deprecated: use hostruntime.EngineRuntimePath.
	EngineRuntimePath = hostruntime.EngineRuntimePath
	// Deprecated: use hostruntime.WorkbenchRuntimePath.
	WorkbenchRuntimePath = hostruntime.WorkbenchRuntimePath
	// Deprecated: use hostruntime.CommandRuntimePath.
	CommandRuntimePath = hostruntime.CommandRuntimePath
	// Deprecated: use hostruntime.StateRuntimePath.
	StateRuntimePath = hostruntime.StateRuntimePath
	// Deprecated: use hostruntime.PreviewSubscriberPath.
	PreviewSubscriberPath = hostruntime.PreviewSubscriberPath
	// Deprecated: use hostruntime.CanvasSelectionBridgePath.
	CanvasSelectionBridgePath = hostruntime.CanvasSelectionBridgePath
	// Deprecated: use hostruntime.CanvasInlineEditPath.
	CanvasInlineEditPath = hostruntime.CanvasInlineEditPath
	// Deprecated: use hostruntime.Canvas2DPainterPath.
	Canvas2DPainterPath = hostruntime.Canvas2DPainterPath
	// Deprecated: use hostruntime.CanvasWASMFreeClientPath.
	CanvasWASMFreeClientPath = hostruntime.CanvasWASMFreeClientPath
	// Deprecated: use hostruntime.CanvasContextualPanelPath.
	CanvasContextualPanelPath = hostruntime.CanvasContextualPanelPath
	// Deprecated: use hostruntime.CanvasDefaultInlineInstallerPath.
	CanvasDefaultInlineInstallerPath = hostruntime.CanvasDefaultInlineInstallerPath
)

// --- Asset mounting (hostruntime/host_runtime.go) ---

// Deprecated: use hostruntime.Mounter.
type Mounter = hostruntime.Mounter

// Deprecated: use hostruntime.MountRuntimes.
func MountRuntimes(app Mounter) { hostruntime.MountRuntimes(app) }

// --- Plugin v2 island registry (hostruntime/plugin_v2.go) ---

// Deprecated: use hostruntime.PluginRegistry.
type PluginRegistry = hostruntime.PluginRegistry

// Deprecated: use hostruntime.NewPluginRegistry.
func NewPluginRegistry() *PluginRegistry { return hostruntime.NewPluginRegistry() }

// --- Runtime/asset bundles and handlers (hostruntime/runtime.go) ---

// Deprecated: use hostruntime.EngineRuntimeScript.
func EngineRuntimeScript() []byte { return hostruntime.EngineRuntimeScript() }

// Deprecated: use hostruntime.EngineRuntimeScriptWithPlugins.
func EngineRuntimeScriptWithPlugins(reg *PluginRegistry) []byte {
	return hostruntime.EngineRuntimeScriptWithPlugins(reg)
}

// Deprecated: use hostruntime.Stylesheet.
func Stylesheet() []byte { return hostruntime.Stylesheet() }

// Deprecated: use hostruntime.EngineRuntimeHandler.
func EngineRuntimeHandler() http.Handler { return hostruntime.EngineRuntimeHandler() }

// Deprecated: use hostruntime.EngineRuntimeHandlerWithPlugins.
func EngineRuntimeHandlerWithPlugins(reg *PluginRegistry) http.Handler {
	return hostruntime.EngineRuntimeHandlerWithPlugins(reg)
}

// Deprecated: use hostruntime.CanvasInlineEditScript.
func CanvasInlineEditScript() []byte { return hostruntime.CanvasInlineEditScript() }

// Deprecated: use hostruntime.CanvasDefaultInlineInstallerScript.
func CanvasDefaultInlineInstallerScript() []byte {
	return hostruntime.CanvasDefaultInlineInstallerScript()
}

// Deprecated: use hostruntime.CanvasContextualPanelScript.
func CanvasContextualPanelScript() []byte { return hostruntime.CanvasContextualPanelScript() }

// Deprecated: use hostruntime.CanvasSelectionBridgeScript.
func CanvasSelectionBridgeScript() []byte { return hostruntime.CanvasSelectionBridgeScript() }

// Deprecated: use hostruntime.Canvas2DPainterScript.
func Canvas2DPainterScript() []byte { return hostruntime.Canvas2DPainterScript() }

// Deprecated: use hostruntime.CanvasWASMFreeClientScript.
func CanvasWASMFreeClientScript() []byte { return hostruntime.CanvasWASMFreeClientScript() }

// Deprecated: use hostruntime.CanvasInlineEditHandler.
func CanvasInlineEditHandler() http.Handler { return hostruntime.CanvasInlineEditHandler() }

// Deprecated: use hostruntime.CanvasDefaultInlineInstallerHandler.
func CanvasDefaultInlineInstallerHandler() http.Handler {
	return hostruntime.CanvasDefaultInlineInstallerHandler()
}

// Deprecated: use hostruntime.CanvasContextualPanelHandler.
func CanvasContextualPanelHandler() http.Handler {
	return hostruntime.CanvasContextualPanelHandler()
}

// Deprecated: use hostruntime.CanvasSelectionBridgeHandler.
func CanvasSelectionBridgeHandler() http.Handler {
	return hostruntime.CanvasSelectionBridgeHandler()
}

// Deprecated: use hostruntime.Canvas2DPainterHandler.
func Canvas2DPainterHandler() http.Handler { return hostruntime.Canvas2DPainterHandler() }

// Deprecated: use hostruntime.CanvasWASMFreeClientHandler.
func CanvasWASMFreeClientHandler() http.Handler {
	return hostruntime.CanvasWASMFreeClientHandler()
}

// Deprecated: use hostruntime.PreviewSubscriberScript.
func PreviewSubscriberScript() []byte { return hostruntime.PreviewSubscriberScript() }

// Deprecated: use hostruntime.WorkbenchRuntimeScript.
func WorkbenchRuntimeScript() []byte { return hostruntime.WorkbenchRuntimeScript() }

// Deprecated: use hostruntime.CommandRuntimeScript.
func CommandRuntimeScript() []byte { return hostruntime.CommandRuntimeScript() }

// Deprecated: use hostruntime.StateRuntimeScript.
func StateRuntimeScript() []byte { return hostruntime.StateRuntimeScript() }

// Deprecated: use hostruntime.AuthoringRuntimeScript.
func AuthoringRuntimeScript() []byte { return hostruntime.AuthoringRuntimeScript() }

// Deprecated: use hostruntime.SiteMapRuntimeScript.
func SiteMapRuntimeScript() []byte { return hostruntime.SiteMapRuntimeScript() }

// Deprecated: use hostruntime.PreviewSubscriberHandler.
func PreviewSubscriberHandler() http.Handler { return hostruntime.PreviewSubscriberHandler() }

// Deprecated: use hostruntime.WorkbenchRuntimeHandler.
func WorkbenchRuntimeHandler() http.Handler { return hostruntime.WorkbenchRuntimeHandler() }

// Deprecated: use hostruntime.CommandRuntimeHandler.
func CommandRuntimeHandler() http.Handler { return hostruntime.CommandRuntimeHandler() }

// Deprecated: use hostruntime.StateRuntimeHandler.
func StateRuntimeHandler() http.Handler { return hostruntime.StateRuntimeHandler() }

// Deprecated: use hostruntime.StylesheetHandler.
func StylesheetHandler() http.Handler { return hostruntime.StylesheetHandler() }

// Deprecated: use hostruntime.ScriptHandler.
func ScriptHandler(name string, data []byte) http.Handler {
	return hostruntime.ScriptHandler(name, data)
}

// Deprecated: use hostruntime.AssetHandler.
func AssetHandler(name string, data []byte, contentType string) http.Handler {
	return hostruntime.AssetHandler(name, data, contentType)
}

// Deprecated: use hostruntime.RuntimeAssetModTime.
func RuntimeAssetModTime() time.Time { return hostruntime.RuntimeAssetModTime() }

// Deprecated: use hostruntime.AssetHref.
func AssetHref(path string, version string) string { return hostruntime.AssetHref(path, version) }
