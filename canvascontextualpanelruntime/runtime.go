// Package canvascontextualpanelruntime owns the WASM-free CanvasBoard
// contextual inspector panel runtime.
//
// The panel is host-agnostic: it renders selected CanvasBoard fields and
// persists edits through the Studio Canvas inline-edit runtime API. A temporary
// legacy Muddy global remains for hosts that have not yet moved their client
// runtime to the Studio primary global.
package canvascontextualpanelruntime

import (
	_ "embed"
)

//go:embed canvas_contextual_panel.js
var canvasContextualPanelJS []byte

// CanvasContextualPanelScript returns the CanvasBoard contextual inspector
// panel. It publishes window.GoSXStudioCanvasContextualPanelRuntime and a
// temporary window.__muddyCanvasContextualPanel compatibility alias.
func CanvasContextualPanelScript() []byte {
	return canvasContextualPanelJS
}
