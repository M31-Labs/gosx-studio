// Package selectionruntime is the Go-side surface for
// GoSXStudioSelectionRuntime.
//
// The runtime exposes one public method, bind, through a .gsx-authored island
// living in this package's selection_bind.gsx file. The island publishes itself
// onto a well-known window global (see BridgeShim below); the JS shim emitted by
// BridgeShim delegates window.GoSXStudioSelectionRuntime calls directly to that
// global.
//
// The deleted JavaScript runtime covered five distinct sub-behaviors that the
// island now owns:
//  1. Block selection (data-block-studio-block click / focus / selectRow).
//  2. Workspace target selection (data-studio-panel-link, data-studio-site-page).
//  3. Field focus (data-studio-field-source, studio:field-select event).
//  4. Selection commandbar actions (data-studio-selection-action: reveal /
//     content / style / inline-text / previous-field / next-field /
//     toggle-visibility).
//  5. Style-scope readouts (data-studio-style-scope-*, data-studio-style-state,
//     data-studio-style-breakpoint).
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

// FeatureFlagKey remains the studio.ShellConfig.FeatureFlags key consumers can
// probe or surface as a stable host attribute. SelectionRuntime binding no
// longer has an off path; BridgeShim delegates directly to the island global
// regardless of this flag's value.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1 and 3..7
// each declare their own constant of the same shape.
const FeatureFlagKey = "selection-runtime-islands"

// IslandGlobals lists the window globals the Selection island publishes
// itself at when it mounts. The shim returned by BridgeShim looks these up at
// call time; an unmounted island leaves the public bind call inert.
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

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime bundle
// appends to publish window.GoSXStudioSelectionRuntime. The public runtime
// delegates directly to the corresponding IslandGlobals entry on window.
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
  if (typeof window === "undefined") return;
  function delegate(islandGlobal) {
    return function () {
      var island = window[islandGlobal];
      if (typeof island === "function") {
        return island.apply(null, arguments);
      }
      return undefined;
    };
  }
  // Install the runtime object. The island global is the only binding path.
  // The literal island-global name below MUST match selectionruntime.IslandGlobals
  // in runtime.go — the test TestBridgeShimDelegatesToIslandGlobals enforces this.
  window.GoSXStudioSelectionRuntime = {
    bind: delegate("__gosx_selection_runtime_island_bind")
  };
  // Auto-mount on document ready. Mirrors the v0.4.1 fieldruntime fix: the
  // deleted studio-engines.js bundle ran a top-level init() at DOMContentLoaded
  // that mounted the six sibling runtimes. Keeping the auto-mount inside the
  // shim attaches selection click handlers on initial load and keeps
  // responsibility local to the slice. Idempotent: a per-document marker
  // attribute guards against double-mount if a host also calls .bind explicitly
  // during SSR hydration.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-selection-runtime-auto-mounted";
      if (document.documentElement && document.documentElement.getAttribute(marker) === "true") return;
      if (document.documentElement) document.documentElement.setAttribute(marker, "true");
      window.GoSXStudioSelectionRuntime.bind(document.body || document);
    } catch (e) {
      // Auto-mount must never break the page; the host can always invoke
      // window.GoSXStudioSelectionRuntime.bind manually if it needs to recover.
    }
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
`
