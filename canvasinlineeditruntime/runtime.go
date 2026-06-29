// Package canvasinlineeditruntime owns the CanvasBoard inline-edit bridge
// assets that sit above the host-agnostic GoSXStudioInlineEditRuntime.
//
// The bridge is still host-agnostic: it adds CanvasBoard repaint-safe bundle
// updates and exposes the default CanvasBoard installer, while action/CSRF
// resolution remains delegated to the generic inline-edit runtime contract or
// host-provided forms/options.
package canvasinlineeditruntime

import (
	_ "embed"
)

//go:embed canvas_inline_edit.js
var canvasInlineEditJS []byte

//go:embed canvas_default_inline_installer.js
var canvasDefaultInlineInstallerJS []byte

// CanvasInlineEditScript returns the CanvasBoard inline-edit bridge. It
// publishes window.GoSXStudioCanvasInlineEditRuntime.
func CanvasInlineEditScript() []byte {
	return canvasInlineEditJS
}

// CanvasDefaultInlineInstallerScript returns the default CanvasBoard installer.
// It publishes window.GoSXStudioCanvasDefaultInlineInstallerRuntime.
func CanvasDefaultInlineInstallerScript() []byte {
	return canvasDefaultInlineInstallerJS
}
