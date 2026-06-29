package studio

import "net/http"

// Mounter is the minimal host-app contract needed to serve Studio runtime
// assets from an HTTP router.
type Mounter interface {
	Mount(pattern string, handler http.Handler)
}

// RuntimeConfig is the reusable host-facing runtime/config bundle for Studio
// browser assets and shell defaults.
type RuntimeConfig struct {
	AssetVersion string
	Host         ShellConfig
}

// DefaultRuntimeConfig returns a RuntimeConfig with Studio shell defaults.
func DefaultRuntimeConfig(assetVersion string) RuntimeConfig {
	return RuntimeConfig{
		AssetVersion: assetVersion,
		Host:         DefaultShellConfig(),
	}
}

// HostConfig returns the normalized Studio host shell config.
func (c RuntimeConfig) HostConfig() ShellConfig {
	return c.Host.Normalize()
}

// Scripts returns the canonical Studio runtime script paths for host views.
func (c RuntimeConfig) Scripts() map[string]any {
	return map[string]any{
		"engineRuntime":    AssetHref(EngineRuntimePath, c.AssetVersion),
		"workbenchRuntime": AssetHref(WorkbenchRuntimePath, c.AssetVersion),
		"commandRuntime":   AssetHref(CommandRuntimePath, c.AssetVersion),
		"stateRuntime":     AssetHref(StateRuntimePath, c.AssetVersion),
	}
}

// EngineHosts returns the generic Studio engine host views.
func (c RuntimeConfig) EngineHosts() []map[string]any {
	return []map[string]any{
		c.engineHost("flow-designer", "studio-engine-host studio-engine-host--flow"),
	}
}

func (c RuntimeConfig) CanvasEngineHost() map[string]any {
	return c.engineHost("canvas", "studio-canvas-engine-host")
}

func (c RuntimeConfig) SiteMapEngineHost() map[string]any {
	return c.engineHost("site-map", "studio-site-map-engine-host")
}

func (c RuntimeConfig) BlockLayoutEngineHost() map[string]any {
	return c.engineHost("block-layout", "studio-block-layout-engine-host")
}

func (c RuntimeConfig) Showcase3DEngineHost() map[string]any {
	return c.engineHost("showcase-3d", "studio-showcase3d-engine-host")
}

func (c RuntimeConfig) CanvasPreviewShell() CanvasPreviewShell {
	return c.HostConfig().Canvas.PreviewShell
}

// MountRuntimes mounts all Studio-owned runtime assets at their public paths.
func MountRuntimes(app Mounter) {
	if app == nil {
		return
	}
	app.Mount("GET "+StylesheetPath, StylesheetHandler())
	app.Mount("GET "+EngineRuntimePath, EngineRuntimeHandler())
	app.Mount("GET "+WorkbenchRuntimePath, WorkbenchRuntimeHandler())
	app.Mount("GET "+CommandRuntimePath, CommandRuntimeHandler())
	app.Mount("GET "+StateRuntimePath, StateRuntimeHandler())
	app.Mount("GET "+PreviewSubscriberPath, PreviewSubscriberHandler())
	app.Mount("GET "+CanvasSelectionBridgePath, CanvasSelectionBridgeHandler())
	app.Mount("GET "+Canvas2DPainterPath, Canvas2DPainterHandler())
	app.Mount("GET "+CanvasWASMFreeClientPath, CanvasWASMFreeClientHandler())
	app.Mount("GET "+CanvasInlineEditPath, CanvasInlineEditHandler())
	app.Mount("GET "+CanvasContextualPanelPath, CanvasContextualPanelHandler())
	app.Mount("GET "+CanvasDefaultInlineInstallerPath, CanvasDefaultInlineInstallerHandler())
}

func (c RuntimeConfig) engineHost(key string, className string) map[string]any {
	engine, ok := c.HostConfig().Engine(key)
	if !ok {
		engine, _ = DefaultShellConfig().Engine(key)
	}
	return EngineHostView(engine, className)
}
