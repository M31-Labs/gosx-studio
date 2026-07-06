package shell

import (
	"m31labs.dev/gosx-studio/core"
	"m31labs.dev/gosx-studio/hostruntime"
)

// RuntimeConfig (originally host_runtime.go) landed in the shell package in
// Slice 8 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md). Slice 3 had deferred
// it: it embeds ShellConfig (shell territory) and its methods use the
// hostruntime /_gosx/studio/* path constants and AssetHref, so it could not
// live in hostruntime (that would invert the DAG — shell imports hostruntime,
// never the reverse) nor stay stranded at the root once shell existed. Now
// that shell is a real package below hostruntime in the DAG, RuntimeConfig
// imports hostruntime directly for the canonical path constants and AssetHref
// (no facade shim needed). The root compat_hostruntime.go continues to
// re-export DefaultRuntimeConfig/RuntimeConfig via compat_shell.go for the
// v0.6.x compatibility window.

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
		"engineRuntime":    hostruntime.AssetHref(hostruntime.EngineRuntimePath, c.AssetVersion),
		"workbenchRuntime": hostruntime.AssetHref(hostruntime.WorkbenchRuntimePath, c.AssetVersion),
		"commandRuntime":   hostruntime.AssetHref(hostruntime.CommandRuntimePath, c.AssetVersion),
		"stateRuntime":     hostruntime.AssetHref(hostruntime.StateRuntimePath, c.AssetVersion),
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

func (c RuntimeConfig) CanvasPreviewShell() core.CanvasPreviewShell {
	return c.HostConfig().Canvas.PreviewShell
}

func (c RuntimeConfig) engineHost(key string, className string) map[string]any {
	engine, ok := c.HostConfig().Engine(key)
	if !ok {
		engine, _ = DefaultShellConfig().Engine(key)
	}
	return EngineHostView(engine, className)
}
