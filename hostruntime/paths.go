package hostruntime

// RuntimeRoot and the path constants below are the public, stable
// /_gosx/studio/* URL contract for Studio-owned runtime assets. They moved
// here from shell.go in Slice 3 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) alongside the
// runtime/asset mounting code that serves them; their string values are
// unchanged (part of the rendered-page contract — see the spec's WATCH FOR
// note (a)).
const (
	RuntimeRoot                      = "/_gosx/studio"
	StylesheetPath                   = RuntimeRoot + "/studio.css"
	EngineRuntimePath                = RuntimeRoot + "/studio-engines.js"
	WorkbenchRuntimePath             = RuntimeRoot + "/workbench-runtime.js"
	CommandRuntimePath               = RuntimeRoot + "/command-palette.js"
	StateRuntimePath                 = RuntimeRoot + "/state-runtime.js"
	ContentEditorRuntimePath         = RuntimeRoot + "/content-editor.js"
	PreviewSubscriberPath            = RuntimeRoot + "/preview-subscriber.js"
	CanvasSelectionBridgePath        = RuntimeRoot + "/canvas-selection-bridge.js"
	CanvasInlineEditPath             = RuntimeRoot + "/canvas-inline-edit.js"
	Canvas2DPainterPath              = RuntimeRoot + "/canvas2d-painter.js"
	CanvasWASMFreeClientPath         = RuntimeRoot + "/canvas-wasm-free-client.js"
	CanvasContextualPanelPath        = RuntimeRoot + "/canvas-contextual-panel.js"
	CanvasDefaultInlineInstallerPath = RuntimeRoot + "/canvas-default-inline-installer.js"
)
