// Package fieldruntime is the Go-side surface for the Phase 3 slice-1
// burn-down of GoSXStudioFieldRuntime.
//
// The slice replaces the JavaScript-implemented FieldRuntime (three methods:
// bind / bindMirroring / bindClipboard, declared at
// gosx-studio/assets/studio-engines.js:2332) with .gsx-authored islands
// living in this package's *.gsx files. Each island publishes itself onto a
// well-known window global (see BridgeShim below); the JS shim emitted by
// BridgeShim delegates window.GoSXStudioFieldRuntime calls to those globals
// when the FeatureFlagKey flag is on in studio.ShellConfig.FeatureFlags.
//
// Both implementations ship additively for at least seven days of CI green
// (see ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-25-phase-3-slice-1-fieldruntime.md
// Section E). After the deletion window closes, the legacy JS implementation
// (bindFieldRuntime / bindFieldMirroring / bindFieldClipboard) is removed
// and the shim collapses to a thin pass-through.
//
// Mirroring writes go through $preview.field.<name> shared signals per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// rather than reaching across the preview iframe directly.
package fieldruntime

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key consumers set to
// activate the .gsx-island FieldRuntime path. Default value (omitted from a
// host's flag map) keeps the legacy JS implementation active.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 2..7 each
// declare their own constant of the same shape.
const FeatureFlagKey = "field-runtime-islands"

// IslandGlobals lists the window globals each Field island publishes itself
// at when it mounts. The shim returned by BridgeShim looks these up at call
// time so unmounted islands fall back to the legacy implementation.
//
// Naming convention: window.__gosx_field_runtime_island_<methodName>.
var IslandGlobals = struct {
	Bind           string
	BindMirroring  string
	BindClipboard  string
}{
	Bind:          "__gosx_field_runtime_island_bind",
	BindMirroring: "__gosx_field_runtime_island_bindMirroring",
	BindClipboard: "__gosx_field_runtime_island_bindClipboard",
}

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime
// bundle appends to gate window.GoSXStudioFieldRuntime through the
// FeatureFlagKey flag.
//
// When the host has marked FeatureFlagKey true (read by the consumer and
// surfaced to the page via the data-gosx-studio-feature-flag-<flag>
// attribute on <html> or any element with that attribute), the shim
// dispatches each method to its corresponding IslandGlobals entry on
// window. When the flag is off or the island global is missing (island
// never mounted), the shim falls back to the legacy JS implementation that
// already lives in studio-engines.js.
//
// The shim itself never imports the legacy implementation directly; it
// references the existing in-bundle functions by name (bindFieldRuntime,
// bindFieldMirroring, bindFieldClipboard) so the bundle remains a single
// self-contained IIFE. After Section E deletes the legacy functions, the
// shim is rewritten to remove the fallback branch.
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

const bridgeShimJS = `;(function () {
  // Phase 3 slice-1 FieldRuntime island bridge.
  // See gosx-studio/fieldruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  var FLAG = "` + FeatureFlagKey + `";
  function flagEnabled() {
    // The consumer surfaces ShellConfig.FeatureFlags via a
    // data-gosx-studio-feature-flag-<key>="true" attribute on the
    // document root (or any ancestor of the editor mount). This keeps the
    // flag decision out of cookies / query params at the bundle level so
    // SSR and CSR observations agree.
    try {
      var attr = "data-gosx-studio-feature-flag-" + FLAG;
      if (document.documentElement && document.documentElement.getAttribute(attr) === "true") return true;
      var marked = document.querySelector("[" + attr + "='true']");
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
  // Replace the runtime object emitted by studio-engines.js with shims
  // that consult the feature flag on every call. Legacy functions
  // (bindFieldRuntime / bindFieldMirroring / bindFieldClipboard) remain
  // defined in the bundle and are passed in as the fallback path. The
  // literal island-global names below MUST match
  // fieldruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  var legacy = window.GoSXStudioFieldRuntime || {};
  window.GoSXStudioFieldRuntime = {
    bind: delegate(
      "__gosx_field_runtime_island_bind",
      typeof bindFieldRuntime === "function" ? bindFieldRuntime : legacy.bind
    ),
    bindMirroring: delegate(
      "__gosx_field_runtime_island_bindMirroring",
      typeof bindFieldMirroring === "function" ? bindFieldMirroring : legacy.bindMirroring
    ),
    bindClipboard: delegate(
      "__gosx_field_runtime_island_bindClipboard",
      typeof bindFieldClipboard === "function" ? bindFieldClipboard : legacy.bindClipboard
    )
  };
})();
`
