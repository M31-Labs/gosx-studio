// Package inlineeditruntime is the Go-side surface for the InlineEditRuntime
// island — a host-agnostic browser runtime that persists in-canvas
// contenteditable edits by POSTing a save-control mutation on blur / Enter.
//
// The island is ported from muddy-noni-commerce/internal/gosxstudio/
// canvas_inline_edit.js but made fully host-agnostic: no muddy-specific route,
// no inline-bundle rewriting. Any host wires inline editing by calling
// window.GoSXStudioInlineEditRuntime.install(overlayEl, opts) with its action
// URL and CSRF token.
//
// Public API:
//
//	InlineEditRuntimeScript() []byte — the IIFE that publishes
//	  window.GoSXStudioInlineEditRuntime = { install, deriveKeys }.
//
//	Bundle() []byte — alias for InlineEditRuntimeScript(), kept parallel
//	  with every other *runtime package so EngineRuntimeScript() can
//	  call Bundle() uniformly.
//
// Phase 3 naming: FeatureFlagKey ends with "-runtime-islands".
package inlineeditruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key for the
// InlineEditRuntime island path. Follows the Phase 3 convention:
// "<contract>-runtime-islands".
const FeatureFlagKey = "inline-edit-runtime-islands"

// InlineEditRuntimeScript returns the embedded inline-edit island JS.
// It publishes window.GoSXStudioInlineEditRuntime = { install, deriveKeys }.
func InlineEditRuntimeScript() []byte {
	return islandRuntimeJS
}

// Bundle returns the complete inline-edit runtime served by the Studio
// engine asset. Parallel to every other *runtime package's Bundle().
func Bundle() []byte {
	return islandRuntimeJS
}
