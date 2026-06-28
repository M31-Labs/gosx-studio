// Package canvasselectionbridgeruntime owns the opt-in full-WASM CanvasBoard
// selection bridge.
//
// The bridge is host-agnostic: it reads GoSX shared CanvasBoard selection
// signals and forwards them through the public Studio site-map runtime API.
// A temporary Muddy compatibility global remains for legacy e2e/status hooks.
package canvasselectionbridgeruntime

import (
	_ "embed"
)

//go:embed canvas_selection_bridge.js
var canvasSelectionBridgeJS []byte

// CanvasSelectionBridgeScript returns the opt-in full-WASM CanvasBoard
// selection bridge. It publishes window.GoSXStudioCanvasSelectionBridgeRuntime
// and a temporary window.__muddyCanvasSelectionBridge compatibility alias.
func CanvasSelectionBridgeScript() []byte {
	return canvasSelectionBridgeJS
}
