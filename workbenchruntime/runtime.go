// Package workbenchruntime is the Go-side surface for the Phase 3 slice-7
// burn-down of GoSXStudioWorkbenchRuntime.
//
// The slice replaces the JavaScript-implemented WorkbenchRuntime (fifteen
// public methods declared at the legacy bundle (removed 2026-05-27) —
// bindRailResizers, bindChrome, setMode, syncViewport, activateViewport,
// currentBreakpoint, setStyleState, syncZoom, activateZoom, toggleRail,
// toggleFocus, toggleActivity, saveLayout, currentRailWidth, setRailWidth)
// with .gsx-authored islands living in this package's *.gsx files. Each
// island publishes itself onto a well-known window global (see BridgeShim
// below); the JS shim emitted by BridgeShim delegates
// window.GoSXStudioWorkbenchRuntime calls to those globals when the
// FeatureFlagKey flag is on in studio.ShellConfig.FeatureFlags.
//
// Both implementations ship additively for at least seven days of CI green
// (see ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// Section G). After the deletion window closes, the legacy JS implementation
// (bindWorkbenchRailResizers / bindWorkbenchChrome / setWorkbenchMode /
// syncWorkbenchViewport / activateWorkbenchViewport / currentWorkbenchBreakpoint /
// setWorkbenchStyleState / syncWorkbenchZoom / activateWorkbenchZoom /
// toggleWorkbenchRail / toggleWorkbenchFocus / toggleWorkbenchActivity /
// saveWorkbenchLayout / currentWorkbenchRailWidth / setWorkbenchRailWidth)
// is removed and the shim collapses to a thin pass-through.
//
// # Method fan-out
//
// All fifteen methods are editor-chrome (no iframe crossing — this slice is
// independent of the cross-frame transport block from slice 6). They divide
// into six groups:
//
//  1. Chrome binding (2 methods):
//
//   - bindRailResizers(root) — binds the [data-studio-resizer] pointer drag
//     + arrow-key handlers on the left/right rail handles; writes
//     --studio-{left,right}-width on the workbench form and dispatches
//     gosxstudio:rail-width-change / gosxstudio:rail-width-commit.
//   - bindChrome(root) — binds the workbench form's click delegate that
//     fans out to setMode / syncViewport / syncZoom / setStyleState /
//     toggleRail / toggleFocus / toggleActivity; binds the rail-width
//     change/commit listeners that persist via saveLayout; applies the
//     stored layout; binds the command palette.
//
//  2. Mode + state (3 methods):
//
//   - setMode(form, mode, scroll) — sets data-studio-mode on the form,
//     toggles is-mode-active / hidden / aria-hidden on
//     [data-studio-mode-panel] siblings, scrolls active panel into view
//     when requested, updates [data-studio-mode-label] readouts, emits
//     gosxstudio:workbench-mode-change.
//   - setStyleState(form, state) — sets data-studio-style-state on the
//     form, toggles aria-pressed on style-state buttons, writes
//     data-style-state / data-style-breakpoint / data-style-valid on
//     style-scope wrappers, emits gosxstudio:workbench-style-state-change.
//   - saveLayout(form) — server-action wrapper: writes the form's left/
//     right rail widths + activity state to localStorage under
//     gosx-studio-editor-layout key.
//
//  3. Viewport (3 methods):
//
//   - syncViewport(form, viewport) — sets data-studio-breakpoint on the
//     form, data-studio-preview-viewport on the .editor-preview-shell (only
//     when the viewport island is not already managing it), updates
//     [data-studio-viewport-label] readouts, emits
//     gosxstudio:workbench-viewport-change, refreshes the canvas via a
//     resize event.
//   - activateViewport(form, viewport) — clicks the matching
//     [data-studio-viewport="<x>"] button if present; otherwise calls
//     syncViewport directly. Routes user-initiated viewport activation so
//     command-palette and toolbar both go through the same dispatch.
//   - currentBreakpoint(form) — pure read: returns the form's
//     data-studio-breakpoint attribute (or the preview-shell's
//     data-studio-preview-viewport fallback, or "desktop"). Slice 7
//     implements this as a derivation from form state, not a method on the
//     runtime — see the slice plan and the parity matrix note.
//
//  4. Zoom (2 methods):
//
//   - syncZoom(form, zoom) — sets data-studio-canvas-zoom on the
//     [data-studio-canvas] element, emits gosxstudio:workbench-zoom-change,
//     refreshes the canvas.
//   - activateZoom(form, zoom) — clicks the matching
//     [data-studio-zoom="<x>"] button if present; otherwise calls syncZoom
//     directly. Routes user-initiated zoom activation through the same
//     dispatch as the toolbar/command-palette.
//
//  5. Toggles (3 methods):
//
//   - toggleRail(form, side) — flips data-studio-{left,right} between
//     "open" and "collapsed", sets data-studio-focus="false", re-syncs the
//     rail toggle aria-pressed states, emits gosxstudio:workbench-rail-change,
//     refreshes the canvas.
//   - toggleFocus(form) — flips data-studio-focus between "true" and
//     "false", re-syncs the rail toggle aria-pressed states, emits
//     gosxstudio:workbench-focus-change, refreshes the canvas.
//   - toggleActivity(form) — flips data-studio-activity-state between
//     "open" and "collapsed", re-syncs activity-toggle buttons (Show/Hide
//     text + aria-pressed), persists via saveLayout, emits
//     gosxstudio:workbench-activity-change, refreshes the canvas.
//
//  6. Rail width (2 methods):
//
//   - currentRailWidth(form, side, handle) — pure read: returns the form's
//     --studio-{left,right}-width custom property as an integer (or the
//     sidebar's bounding-rect width fallback, or the handle's bounds
//     fallback). Slice 7 implements this as a derivation from form state.
//   - setRailWidth(form, side, width, handle, committed) — clamps width to
//     the handle's min/max bounds, writes --studio-{left,right}-width on
//     the form, updates the handle's aria-valuenow, emits
//     gosxstudio:rail-width-change or gosxstudio:rail-width-commit based on
//     the committed flag.
//
// Per the slice plan, currentBreakpoint and currentRailWidth are READS that
// derive from existing form state — they ship as island globals so the
// BridgeShim can route through them uniformly, but their implementations
// are pure derivations rather than mutators.
//
// Each method is exercised by its own test() call in
// parity_e2e/workbenchruntime_test.ts so the parity claim is not collapsed
// to a single boolean.
package workbenchruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key consumers set to
// activate the .gsx-island WorkbenchRuntime path. Default value (omitted
// from a host's flag map) keeps the legacy JS implementation active.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1, 2, 3,
// 4, 5, 6 each declare their own constant of the same shape.
const FeatureFlagKey = "workbench-runtime-islands"

// IslandGlobals lists the window globals each Workbench island publishes
// itself at when it mounts. The shim returned by BridgeShim looks these up
// at call time so unmounted islands fall back to the legacy implementation.
//
// Naming convention: window.__gosx_workbench_runtime_island_<methodName>.
// WorkbenchRuntime declares fifteen public methods, so the struct holds
// fifteen fields.
var IslandGlobals = struct {
	BindRailResizers   string
	BindChrome         string
	SetMode            string
	SyncViewport       string
	ActivateViewport   string
	CurrentBreakpoint  string
	SetStyleState      string
	SyncZoom           string
	ActivateZoom       string
	ToggleRail         string
	ToggleFocus        string
	ToggleActivity     string
	SaveLayout         string
	CurrentRailWidth   string
	SetRailWidth       string
}{
	BindRailResizers:  "__gosx_workbench_runtime_island_bindRailResizers",
	BindChrome:        "__gosx_workbench_runtime_island_bindChrome",
	SetMode:           "__gosx_workbench_runtime_island_setMode",
	SyncViewport:      "__gosx_workbench_runtime_island_syncViewport",
	ActivateViewport:  "__gosx_workbench_runtime_island_activateViewport",
	CurrentBreakpoint: "__gosx_workbench_runtime_island_currentBreakpoint",
	SetStyleState:     "__gosx_workbench_runtime_island_setStyleState",
	SyncZoom:          "__gosx_workbench_runtime_island_syncZoom",
	ActivateZoom:      "__gosx_workbench_runtime_island_activateZoom",
	ToggleRail:        "__gosx_workbench_runtime_island_toggleRail",
	ToggleFocus:       "__gosx_workbench_runtime_island_toggleFocus",
	ToggleActivity:    "__gosx_workbench_runtime_island_toggleActivity",
	SaveLayout:        "__gosx_workbench_runtime_island_saveLayout",
	CurrentRailWidth:  "__gosx_workbench_runtime_island_currentRailWidth",
	SetRailWidth:      "__gosx_workbench_runtime_island_setRailWidth",
}

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime
// bundle appends to gate window.GoSXStudioWorkbenchRuntime through the
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
// references the existing in-bundle functions by name (bindWorkbenchRailResizers
// / bindWorkbenchChrome / setWorkbenchMode / syncWorkbenchViewport /
// activateWorkbenchViewport / currentWorkbenchBreakpoint /
// setWorkbenchStyleState / syncWorkbenchZoom / activateWorkbenchZoom /
// toggleWorkbenchRail / toggleWorkbenchFocus / toggleWorkbenchActivity /
// saveWorkbenchLayout / currentWorkbenchRailWidth / setWorkbenchRailWidth)
// so the bundle remains a single self-contained IIFE. After Section G
// deletes the legacy functions, the shim is rewritten to remove the
// fallback branch.
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

// IslandRuntimeJS returns the JS that publishes the
// window.__gosx_workbench_runtime_island_* globals BridgeShim delegates to.
// Sourced from island_runtime.js (embedded). Companion to the .gsx islands
// in this package: the .gsx files are the mount-point markers in the editor
// DOM; this script is the actual behavior layer for slice 7.
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
  // Phase 3 slice-7 WorkbenchRuntime island bridge.
  // See gosx-studio/workbenchruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  // The consumer surfaces ShellConfig.FeatureFlags via a
  // data-gosx-studio-feature-flag-<key>="true" attribute on the document
  // root (or any ancestor of the editor mount). This keeps the flag
  // decision out of cookies / query params at the bundle level so SSR and
  // CSR observations agree. The attribute name below MUST be the literal
  // "data-gosx-studio-feature-flag-workbench-runtime-islands" so a grep over
  // the bundle finds it; the test
  // TestEngineRuntimeIncludesWorkbenchRuntimeIslandBundle enforces this.
  var FLAG_ATTR = "data-gosx-studio-feature-flag-workbench-runtime-islands";
  function flagEnabled() {
    try {
      if (document.documentElement && document.documentElement.getAttribute(FLAG_ATTR) === "true") return true;
      var marked = document.querySelector("[" + FLAG_ATTR + "='true']");
      return !!marked;
    } catch (e) {
      return false;
    }
  }
  function delegate(islandGlobal) {
    return function () {
      if (!flagEnabled()) return undefined;
      var island = window[islandGlobal];
      if (typeof island === "function") {
        return island.apply(null, arguments);
      }
      return undefined;
    };
  }
  // Install the runtime object. Pre-2026-05-27 the BridgeShim wrapped the
  // legacy bundle's fifteen workbench functions (bindWorkbenchRailResizers
  // / bindWorkbenchChrome / setWorkbenchMode / syncWorkbenchViewport /
  // activateWorkbenchViewport / currentWorkbenchBreakpoint /
  // setWorkbenchStyleState / syncWorkbenchZoom / activateWorkbenchZoom /
  // toggleWorkbenchRail / toggleWorkbenchFocus / toggleWorkbenchActivity /
  // saveWorkbenchLayout / currentWorkbenchRailWidth / setWorkbenchRailWidth)
  // as fallback paths; with the legacy bundle deleted in Phase 3 Section E
  // those identifiers are undefined and the fallback branches were dead
  // code. v0.5.0 removes the dead branches — the island global is the only
  // path. The literal island-global names below MUST match
  // workbenchruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  window.GoSXStudioWorkbenchRuntime = {
    bindRailResizers: delegate("__gosx_workbench_runtime_island_bindRailResizers"),
    bindChrome: delegate("__gosx_workbench_runtime_island_bindChrome"),
    setMode: delegate("__gosx_workbench_runtime_island_setMode"),
    syncViewport: delegate("__gosx_workbench_runtime_island_syncViewport"),
    activateViewport: delegate("__gosx_workbench_runtime_island_activateViewport"),
    currentBreakpoint: delegate("__gosx_workbench_runtime_island_currentBreakpoint"),
    setStyleState: delegate("__gosx_workbench_runtime_island_setStyleState"),
    syncZoom: delegate("__gosx_workbench_runtime_island_syncZoom"),
    activateZoom: delegate("__gosx_workbench_runtime_island_activateZoom"),
    toggleRail: delegate("__gosx_workbench_runtime_island_toggleRail"),
    toggleFocus: delegate("__gosx_workbench_runtime_island_toggleFocus"),
    toggleActivity: delegate("__gosx_workbench_runtime_island_toggleActivity"),
    saveLayout: delegate("__gosx_workbench_runtime_island_saveLayout"),
    currentRailWidth: delegate("__gosx_workbench_runtime_island_currentRailWidth"),
    setRailWidth: delegate("__gosx_workbench_runtime_island_setRailWidth")
  };
  // Auto-mount on document ready. Mirrors the v0.4.1 fieldruntime fix: the
  // deleted studio-engines.js bundle ran a top-level init() at
  // DOMContentLoaded that called bindWorkbenchRailResizers(document) and
  // bindWorkbenchChrome(document). When the bundle was deleted the per-slice
  // BridgeShim correctly re-published window.GoSXStudioWorkbenchRuntime but
  // the two auto-mounts were lost — the rail-resizer drag handles and
  // workbench chrome (toolbar mode/viewport buttons, command palette
  // listener) were never wired on initial load. The remaining thirteen
  // methods on the WorkbenchRuntime contract are host-driven writes (mode
  // switches, viewport changes, zoom updates, layout save, etc.), not boot
  // bindings, so they're NOT auto-mounted. Idempotent: a per-document marker
  // attribute guards against double-mount if a host also calls .bind*
  // explicitly during SSR hydration.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-workbench-runtime-auto-mounted";
      if (document.documentElement && document.documentElement.getAttribute(marker) === "true") return;
      if (document.documentElement) document.documentElement.setAttribute(marker, "true");
      var root = document.body || document;
      window.GoSXStudioWorkbenchRuntime.bindRailResizers(root);
      window.GoSXStudioWorkbenchRuntime.bindChrome(root);
    } catch (e) {
      // Auto-mount must never break the page; the host can always invoke
      // window.GoSXStudioWorkbenchRuntime.bind* manually if it needs to recover.
    }
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
`
