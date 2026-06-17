// Package inspectorruntime is the Go-side surface for the InspectorIsland
// runtime — the browser-side dirty tracking, Cancel/revert, and
// required-field validation layer for the editable control inspector panel.
//
// Submit / persistence is intentionally absent: the authoringruntime
// managed-form path (data-gosx-studio-authoring-managed) owns that
// concern. This island is purely UX-layer.
//
// Public API:
//
//   InspectorRuntimeScript() []byte — the IIFE that publishes
//     window.GoSXStudioInspectorRuntime = { bind, bindAll }.
//
//   Bundle() []byte — alias for InspectorRuntimeScript(), kept parallel
//     with every other *runtime package so EngineRuntimeScript() can
//     call Bundle() uniformly.
//
// Phase 3 naming: FeatureFlagKey ends with "-runtime-islands".
package inspectorruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key for the
// InspectorRuntime island path. Follows the Phase 3 convention:
// "<contract>-runtime-islands".
const FeatureFlagKey = "inspector-runtime-islands"

// InspectorRuntimeScript returns the embedded inspector island JS.
// It publishes window.GoSXStudioInspectorRuntime = { bind, bindAll }.
func InspectorRuntimeScript() []byte {
	return islandRuntimeJS
}

// Bundle returns the complete inspector runtime served by the Studio
// engine asset. Parallel to every other *runtime package's Bundle().
func Bundle() []byte {
	return islandRuntimeJS
}
