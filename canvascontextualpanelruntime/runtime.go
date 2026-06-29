// Package canvascontextualpanelruntime owns the WASM-free CanvasBoard
// contextual inspector panel runtime.
//
// The panel is host-agnostic: it renders selected CanvasBoard fields and
// persists edits through the Studio Canvas inline-edit runtime API.
package canvascontextualpanelruntime

import (
	_ "embed"
)

//go:embed canvas_contextual_panel.js
var canvasContextualPanelJS []byte

// CanvasContextualPanelScript returns the CanvasBoard contextual inspector
// panel. It publishes window.GoSXStudioCanvasContextualPanelRuntime.
func CanvasContextualPanelScript() []byte {
	return canvasContextualPanelJS
}
