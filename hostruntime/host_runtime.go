package hostruntime

import "net/http"

// Mounter is the minimal host-app contract needed to serve Studio runtime
// assets from an HTTP router.
type Mounter interface {
	Mount(pattern string, handler http.Handler)
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
	app.Mount("GET "+ContentEditorRuntimePath, ContentEditorRuntimeHandler())
	app.Mount("GET "+MediaRuntimePath, MediaRuntimeHandler())
	app.Mount("GET "+SectionOrderRuntimePath, SectionOrderRuntimeHandler())
	app.Mount("GET "+PreviewSubscriberPath, PreviewSubscriberHandler())
	app.Mount("GET "+CanvasSelectionBridgePath, CanvasSelectionBridgeHandler())
	app.Mount("GET "+Canvas2DPainterPath, Canvas2DPainterHandler())
	app.Mount("GET "+CanvasWASMFreeClientPath, CanvasWASMFreeClientHandler())
	app.Mount("GET "+CanvasInlineEditPath, CanvasInlineEditHandler())
	app.Mount("GET "+CanvasContextualPanelPath, CanvasContextualPanelHandler())
	app.Mount("GET "+CanvasDefaultInlineInstallerPath, CanvasDefaultInlineInstallerHandler())
}
