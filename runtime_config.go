package studio

// This file (originally host_runtime.go) is what remains at the studio root
// after Slice 3 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) moved the
// Mounter/MountRuntimes asset-mounting pair, the runtime/asset bundle code,
// and the public /_gosx/studio/* path constants into
// m31labs.dev/gosx-studio/hostruntime (see compat_hostruntime.go for the
// facade aliases).
//
// RuntimeConfig itself stays here (not in hostruntime) for now: it embeds
// ShellConfig, which is shell-package territory (Slice 8) and still lives
// only at this root package today. hostruntime must not import the root
// package (that would be a compile-time import cycle, since this package
// already imports hostruntime for the Slice 3 facade), and shell is DAG-below
// hostruntime (shell imports hostruntime, never the reverse), so
// RuntimeConfig cannot move until ShellConfig has its own package in Slice 8.
// It is deferred there rather than forced into hostruntime early. It
// continues to compile unchanged today via the compat_hostruntime.go shims
// for AssetHref and the runtime path constants, which resolve as unqualified
// package-level names in this same package.

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

func (c RuntimeConfig) engineHost(key string, className string) map[string]any {
	engine, ok := c.HostConfig().Engine(key)
	if !ok {
		engine, _ = DefaultShellConfig().Engine(key)
	}
	return EngineHostView(engine, className)
}
