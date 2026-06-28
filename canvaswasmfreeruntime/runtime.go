// Package canvaswasmfreeruntime owns the WASM-free CanvasBoard painter and
// browser client runtime assets.
//
// The runtime is host-agnostic: it paints server-provided CanvasBoard render
// bundles, mounts CanvasBoard overlays, and delegates selection/editor behavior
// through public Studio browser runtime APIs. Temporary Muddy compatibility
// globals remain for hosts and tests that have not yet moved to the primary
// Studio names.
package canvaswasmfreeruntime

import (
	_ "embed"
)

//go:embed canvas2d_painter.js
var canvas2DPainterJS []byte

//go:embed canvas_wasm_free_client.js
var canvasWASMFreeClientJS []byte

// Canvas2DPainterScript returns the WASM-free Canvas2D painter. It publishes
// window.GoSXStudioCanvas2DPainterRuntime and a temporary
// window.__muddyCanvas2DPainter compatibility alias.
func Canvas2DPainterScript() []byte {
	return canvas2DPainterJS
}

// CanvasWASMFreeClientScript returns the WASM-free CanvasBoard browser client.
// It publishes window.GoSXStudioCanvasWASMFreeClientRuntime and a temporary
// window.__muddyCanvasWasmFreeClient compatibility alias.
func CanvasWASMFreeClientScript() []byte {
	return canvasWASMFreeClientJS
}
