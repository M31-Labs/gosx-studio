// Package styleruntime is the Go-side surface for the Phase 3 slice-5
// burn-down of GoSXStudioStyleRuntime.
//
// The slice replaces the JavaScript-implemented StyleRuntime (ten public
// methods declared at the legacy bundle (removed 2026-05-27) —
// bindTheme, bindWorkbench, bindCSS, bindFonts, applyTheme,
// syncControlButtons, showImpact, restoreImpact, setControlValue,
// resetControlValue) with .gsx-authored islands living in this package's
// *.gsx files. Each island publishes itself onto a well-known window global
// (see BridgeShim below); the JS shim emitted by BridgeShim delegates
// window.GoSXStudioStyleRuntime calls to those globals when the
// FeatureFlagKey flag is on in studio.ShellConfig.FeatureFlags.
//
// Both implementations ship additively for at least seven days of CI green
// (see ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-5-styleruntime.md
// Section G). After the deletion window closes, the legacy JS implementation
// (bindTheme / bindStyleWorkbench / bindCSS / bindFonts / updateTheme /
// syncStyleControlButtons / showStyleImpact / restoreStyleImpact /
// setStyleControlValue / resetStyleControlValue) is removed and the shim
// collapses to a thin pass-through.
//
// # Method fan-out
//
// The ten methods divide into three groups:
//
//  1. Editor-side binding (4 methods — bind input controls to Bridge signals):
//
//   - bindTheme(root) — binds theme kit / template / palette / image-ratio /
//     custom-template / style-class / color-token controls; on input/change
//     dispatches updateTheme (which calls into PreviewRuntime.applyTheme).
//   - bindWorkbench(root) — binds the editor-workbench form's click / pointer /
//     focus listeners so style controls / resets fire setControlValue /
//     resetControlValue and the impact panel previews / restores on hover /
//     focus.
//   - bindCSS(root) — binds the [data-editor-css] textarea so input dispatches
//     PreviewRuntime.applyCSS(value).
//   - bindFonts(root) — binds the three [data-editor-font-name] /
//     [data-editor-font-url] slot pairs so input/change recomputes the
//     combined @font-face + :root font-variable CSS and dispatches
//     PreviewRuntime.applyFonts(css).
//
//  2. Editor-side state (3 methods — toggle button states / signal writes):
//
//   - syncControlButtons(root) — for every [data-studio-style-control],
//     [data-studio-style-readout], [data-studio-style-reset] in root, sync
//     aria-pressed / .is-selected / textContent against the live form values
//     and kit defaults.
//   - setControlValue(name, value) — sets the named control's value (radio /
//     input / select), dispatches input/change events, then runs
//     syncControlButtons + showImpact(name, value, true).
//   - resetControlValue(name) — looks up kit default for name and calls
//     setControlValue(name, default).
//
//  3. Transitional iframe-crossing (3 methods):
//
//   - applyTheme() — reads theme form state (kit / template / palette / ratio /
//     custom-template / style classes / colors), then calls
//     window.GoSXStudioPreviewRuntime.applyTheme(payload) to apply the
//     storefront preview's class+token mutation, then updates theme swatches /
//     kit cards / template cards / custom-template builder, then runs
//     syncControlButtons(document).
//   - showImpact(name, value, committed) — looks up impact metadata for the
//     named control, calls window.GoSXStudioPreviewRuntime.applyStyleImpact
//     (selector) for the count, then writes the editor-side impact-readout
//     labels (label / summary / count / scope / state) and toggles the
//     impact panel's is-live / has-change classes.
//   - restoreImpact() — restores the last committed style impact via
//     showImpact, or clears the readouts to a "No active recipe" state and
//     calls window.GoSXStudioPreviewRuntime.applyStyleImpact("").
//
// Per ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// the three iframe-crossing methods follow the transitional pattern —
// ownership has moved to the styleruntime islands in slice 5, but the
// mechanism (iframe DOM mutation via PreviewRuntime) stays until slice 6
// collapses it to $preview.theme.* / $preview.style.impact.* shared-signal
// writes. The slice 5 islands keep the same iframe contract by delegating
// to the legacy PreviewRuntime functions — slice 6 will swap the mechanism
// without revisiting style ownership. Same shape as slice 3's
// updateHeaderLogo and slice 4's updateVisibilityState.
//
// Each method is exercised by its own test() call in
// parity_e2e/styleruntime_test.ts so the parity claim is not collapsed to a
// single boolean.
package styleruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key consumers set to
// activate the .gsx-island StyleRuntime path. Default value (omitted from a
// host's flag map) keeps the legacy JS implementation active.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1, 2, 3, 4,
// 6, 7 each declare their own constant of the same shape.
const FeatureFlagKey = "style-runtime-islands"

// IslandGlobals lists the window globals each Style island publishes itself
// at when it mounts. The shim returned by BridgeShim looks these up at call
// time so unmounted islands fall back to the legacy implementation.
//
// Naming convention: window.__gosx_style_runtime_island_<methodName>.
// StyleRuntime declares ten public methods (bindTheme, bindWorkbench,
// bindCSS, bindFonts, applyTheme, syncControlButtons, showImpact,
// restoreImpact, setControlValue, resetControlValue), so the struct holds
// ten fields.
var IslandGlobals = struct {
	BindTheme          string
	BindWorkbench      string
	BindCSS            string
	BindFonts          string
	ApplyTheme         string
	SyncControlButtons string
	ShowImpact         string
	RestoreImpact      string
	SetControlValue    string
	ResetControlValue  string
}{
	BindTheme:          "__gosx_style_runtime_island_bindTheme",
	BindWorkbench:      "__gosx_style_runtime_island_bindWorkbench",
	BindCSS:            "__gosx_style_runtime_island_bindCSS",
	BindFonts:          "__gosx_style_runtime_island_bindFonts",
	ApplyTheme:         "__gosx_style_runtime_island_applyTheme",
	SyncControlButtons: "__gosx_style_runtime_island_syncControlButtons",
	ShowImpact:         "__gosx_style_runtime_island_showImpact",
	RestoreImpact:      "__gosx_style_runtime_island_restoreImpact",
	SetControlValue:    "__gosx_style_runtime_island_setControlValue",
	ResetControlValue:  "__gosx_style_runtime_island_resetControlValue",
}

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime
// bundle appends to gate window.GoSXStudioStyleRuntime through the
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
// references the existing in-bundle functions by name (bindTheme /
// bindStyleWorkbench / bindCSS / bindFonts / updateTheme /
// syncStyleControlButtons / showStyleImpact / restoreStyleImpact /
// setStyleControlValue / resetStyleControlValue) so the bundle remains a
// single self-contained IIFE. After Section G deletes the legacy
// functions, the shim is rewritten to remove the fallback branch.
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

// IslandRuntimeJS returns the JS that publishes the
// window.__gosx_style_runtime_island_* globals BridgeShim delegates to.
// Sourced from island_runtime.js (embedded). Companion to the .gsx islands
// in this package: the .gsx files are the mount-point markers in the editor
// DOM; this script is the actual behavior layer for slice 5, sized so the
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
  // Phase 3 slice-5 StyleRuntime island bridge.
  // See gosx-studio/styleruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  // The consumer surfaces ShellConfig.FeatureFlags via a
  // data-gosx-studio-feature-flag-<key>="true" attribute on the document
  // root (or any ancestor of the editor mount). This keeps the flag
  // decision out of cookies / query params at the bundle level so SSR and
  // CSR observations agree. The attribute name below MUST be the literal
  // "data-gosx-studio-feature-flag-style-runtime-islands" so a grep over
  // the bundle finds it; the test
  // TestEngineRuntimeIncludesStyleRuntimeIslandBundle enforces this.
  var FLAG_ATTR = "data-gosx-studio-feature-flag-style-runtime-islands";
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
  // that consults the feature flag on every call. The legacy functions
  // (bindTheme / bindStyleWorkbench / bindCSS / bindFonts / updateTheme /
  // syncStyleControlButtons / showStyleImpact / restoreStyleImpact /
  // setStyleControlValue / resetStyleControlValue) remain defined in the
  // bundle and are passed in as the fallback path. The literal
  // island-global names below MUST match styleruntime.IslandGlobals in
  // runtime.go — the test TestBridgeShimDelegatesToIslandGlobals enforces
  // this.
  var legacy = window.GoSXStudioStyleRuntime || {};
  window.GoSXStudioStyleRuntime = {
    bindTheme: delegate(
      "__gosx_style_runtime_island_bindTheme",
      typeof bindTheme === "function" ? bindTheme : legacy.bindTheme
    ),
    bindWorkbench: delegate(
      "__gosx_style_runtime_island_bindWorkbench",
      typeof bindStyleWorkbench === "function" ? bindStyleWorkbench : legacy.bindWorkbench
    ),
    bindCSS: delegate(
      "__gosx_style_runtime_island_bindCSS",
      typeof bindCSS === "function" ? bindCSS : legacy.bindCSS
    ),
    bindFonts: delegate(
      "__gosx_style_runtime_island_bindFonts",
      typeof bindFonts === "function" ? bindFonts : legacy.bindFonts
    ),
    applyTheme: delegate(
      "__gosx_style_runtime_island_applyTheme",
      typeof updateTheme === "function" ? updateTheme : legacy.applyTheme
    ),
    syncControlButtons: delegate(
      "__gosx_style_runtime_island_syncControlButtons",
      typeof syncStyleControlButtons === "function" ? syncStyleControlButtons : legacy.syncControlButtons
    ),
    showImpact: delegate(
      "__gosx_style_runtime_island_showImpact",
      typeof showStyleImpact === "function" ? showStyleImpact : legacy.showImpact
    ),
    restoreImpact: delegate(
      "__gosx_style_runtime_island_restoreImpact",
      typeof restoreStyleImpact === "function" ? restoreStyleImpact : legacy.restoreImpact
    ),
    setControlValue: delegate(
      "__gosx_style_runtime_island_setControlValue",
      typeof setStyleControlValue === "function" ? setStyleControlValue : legacy.setControlValue
    ),
    resetControlValue: delegate(
      "__gosx_style_runtime_island_resetControlValue",
      typeof resetStyleControlValue === "function" ? resetStyleControlValue : legacy.resetControlValue
    )
  };
  // Auto-mount on document ready. Mirrors the v0.4.1 fieldruntime fix: the
  // deleted studio-engines.js bundle ran a top-level init() at
  // DOMContentLoaded that called four StyleRuntime binds — bindTheme,
  // bindStyleWorkbench, bindCSS, bindFonts — against document. When the
  // bundle was deleted the per-slice BridgeShim correctly re-published
  // window.GoSXStudioStyleRuntime but the four auto-mounts were lost — none
  // of the style control listeners (theme picker, workbench control buttons,
  // CSS editor, font picker) were attached on initial load, so editing styles
  // from a cold boot did nothing. The remaining six methods (applyTheme,
  // syncControlButtons, showImpact, restoreImpact, setControlValue,
  // resetControlValue) are NOT auto-mounted because the legacy init() only
  // called the four bind* methods at boot; the rest are host-driven writes.
  // Idempotent: a per-document marker attribute guards against double-mount
  // if a host also calls .bind* explicitly during SSR hydration.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-style-runtime-auto-mounted";
      if (document.documentElement && document.documentElement.getAttribute(marker) === "true") return;
      if (document.documentElement) document.documentElement.setAttribute(marker, "true");
      var root = document.body || document;
      window.GoSXStudioStyleRuntime.bindTheme(root);
      window.GoSXStudioStyleRuntime.bindWorkbench(root);
      window.GoSXStudioStyleRuntime.bindCSS(root);
      window.GoSXStudioStyleRuntime.bindFonts(root);
    } catch (e) {
      // Auto-mount must never break the page; the host can always invoke
      // window.GoSXStudioStyleRuntime.bind* manually if it needs to recover.
    }
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
`
