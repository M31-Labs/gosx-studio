// Package selectionruntime is the Go-side surface for the Phase 3 slice-2
// burn-down of GoSXStudioSelectionRuntime.
//
// The slice replaces the JavaScript-implemented SelectionRuntime (single
// public method: bind, declared at the legacy bundle (removed 2026-05-27)
// and implemented as bindSelectionSurface at line 713) with a .gsx-authored
// island living in this package's selection_bind.gsx file. The island
// publishes itself onto a well-known window global (see BridgeShim below);
// the JS shim emitted by BridgeShim delegates window.GoSXStudioSelectionRuntime
// calls to that global when the FeatureFlagKey flag is on in
// studio.ShellConfig.FeatureFlags.
//
// Both implementations ship additively for at least seven days of CI green
// (see ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-2-selectionruntime.md
// Section E). After the deletion window closes, the legacy JS implementation
// (bindSelectionSurface) is removed and the shim collapses to a thin
// pass-through.
//
// The legacy bindSelectionSurface fan-out covers five distinct sub-behaviors:
//   1. Block selection (data-block-studio-block click / focus / selectRow).
//   2. Workspace target selection (data-studio-panel-link, data-studio-site-page).
//   3. Field focus (data-studio-field-source, studio:field-select event).
//   4. Selection commandbar actions (data-studio-selection-action: reveal /
//      content / style / inline-text / previous-field / next-field /
//      toggle-visibility).
//   5. Style-scope readouts (data-studio-style-scope-*, data-studio-style-state,
//      data-studio-style-breakpoint).
//
// Each sub-behavior is exercised by its own test() call in
// parity_e2e/selectionruntime_test.ts so the parity claim is not
// collapsed to a single boolean.
//
// Any cross-iframe state (none expected for selection — all five
// sub-behaviors operate on editor-side DOM only) would go through
// $preview.* shared signals per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md.
// Selection state is editor-only; the island writes plain Bridge signals
// ($selection.block, $selection.workspaceTarget, $selection.fieldFocus,
// $selection.styleScope) for the island runtime's own bookkeeping.
package selectionruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key consumers set to
// activate the .gsx-island SelectionRuntime path. Default value (omitted from
// a host's flag map) keeps the legacy JS implementation active.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1 and 3..7
// each declare their own constant of the same shape.
const FeatureFlagKey = "selection-runtime-islands"

// IslandGlobals lists the window globals the Selection island publishes
// itself at when it mounts. The shim returned by BridgeShim looks these up
// at call time so an unmounted island falls back to the legacy
// implementation.
//
// Naming convention: window.__gosx_selection_runtime_island_<methodName>.
// SelectionRuntime declares a single public method (bind), so the struct
// holds a single field. Slices 3..7 with multi-method contracts add one
// field per method (see fieldruntime.IslandGlobals for the 3-method shape).
var IslandGlobals = struct {
	Bind string
}{
	Bind: "__gosx_selection_runtime_island_bind",
}

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime
// bundle appends to gate window.GoSXStudioSelectionRuntime through the
// FeatureFlagKey flag.
//
// When the host has marked FeatureFlagKey true (read by the consumer and
// surfaced to the page via the data-gosx-studio-feature-flag-<flag>
// attribute on <html> or any element with that attribute), the shim
// dispatches each method to its corresponding IslandGlobals entry on
// window. When the flag is off or the island global is missing (island
// never mounted), the shim falls back to the legacy JS implementation that
// lived in the now-deleted legacy bundle.
//
// The shim itself never imports the legacy implementation directly; it
// references the existing in-bundle function by name (bindSelectionSurface)
// so the bundle remains a single self-contained IIFE. After Section E
// deletes the legacy function, the shim is rewritten to remove the
// fallback branch.
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

// IslandRuntimeJS returns the JS that publishes the
// window.__gosx_selection_runtime_island_* global BridgeShim delegates to.
// Sourced from island_runtime.js (embedded). Companion to the .gsx island
// in this package: the .gsx file is the mount-point marker in the editor
// DOM; this script is the actual behavior layer for slice 2, sized so the
// slice ships independently of the inspector form being signal-driven
// (which is a separate workstream).
func IslandRuntimeJS() []byte {
	return islandRuntimeJS
}

// Bundle returns the IslandRuntimeJS + BridgeShim concatenation that the
// studio runtime asset pipeline serves. Order matters:
// the island runtime publishes globals; the shim consults them.
func Bundle() []byte {
	island := IslandRuntimeJS()
	shim := BridgeShim()
	out := make([]byte, 0, len(island)+1+len(shim))
	out = append(out, island...)
	out = append(out, '\n')
	out = append(out, shim...)
	return out
}

const bridgeShimJS = `;(function () {
  // Phase 3 slice-2 SelectionRuntime island bridge.
  // See gosx-studio/selectionruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  // The consumer surfaces ShellConfig.FeatureFlags via a
  // data-gosx-studio-feature-flag-<key>="true" attribute on the document
  // root (or any ancestor of the editor mount). This keeps the flag
  // decision out of cookies / query params at the bundle level so SSR and
  // CSR observations agree. The attribute name below MUST be the literal
  // "data-gosx-studio-feature-flag-selection-runtime-islands" so a grep
  // over the bundle finds it; the test
  // TestEngineRuntimeIncludesSelectionRuntimeIslandBundle enforces this.
  var FLAG_ATTR = "data-gosx-studio-feature-flag-selection-runtime-islands";
  function flagEnabled() {
    try {
      if (document.documentElement && document.documentElement.getAttribute(FLAG_ATTR) === "true") return true;
      var marked = document.querySelector("[" + FLAG_ATTR + "='true']");
      return !!marked;
    } catch (e) {
      return false;
    }
  }
  function delegate(islandGlobal, legacyFn) {
    return function () {
      if (flagEnabled()) {
        var island = window[islandGlobal];
        if (typeof island === "function") {
          return island.apply(null, arguments);
        }
      }
      if (typeof legacyFn === "function") {
        return legacyFn.apply(null, arguments);
      }
      return undefined;
    };
  }
  // Install the runtime object (the legacy bundle that previously emitted it was removed 2026-05-27) with a shim
  // that consults the feature flag on every call. The legacy function
  // (bindSelectionSurface) remains defined in the bundle and is passed in
  // as the fallback path. The literal island-global name below MUST match
  // selectionruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  var legacy = window.GoSXStudioSelectionRuntime || {};
  window.GoSXStudioSelectionRuntime = {
    bind: delegate(
      "__gosx_selection_runtime_island_bind",
      typeof bindSelectionSurface === "function" ? bindSelectionSurface : legacy.bind
    )
  };
})();
`
