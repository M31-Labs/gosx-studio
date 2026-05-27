// Package brandruntime is the Go-side surface for the Phase 3 slice-3
// burn-down of GoSXStudioBrandRuntime.
//
// The slice replaces the JavaScript-implemented BrandRuntime (two public
// methods: bindLogo / updateHeaderLogo, declared at
// the legacy bundle (removed 2026-05-27) and implemented as
// bindBrandLogo / updateHeaderLogo at lines 1269 / 1264) with .gsx-authored
// islands living in this package's *.gsx files. Each island publishes itself
// onto a well-known window global (see BridgeShim below); the JS shim emitted
// by BridgeShim delegates window.GoSXStudioBrandRuntime calls to those
// globals when the FeatureFlagKey flag is on in studio.ShellConfig.FeatureFlags.
//
// Both implementations ship additively for at least seven days of CI green
// (see ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-3-brandruntime.md
// Section E). After the deletion window closes, the legacy JS implementation
// (bindBrandLogo / updateHeaderLogo) is removed and the shim collapses to a
// thin pass-through.
//
// # Method fan-out
//
//  1. bindLogo (editor-side only) — wires the editor's brand-logo authoring
//     surface: URL/alt/width/X/Y/snap inputs, the drag-handle, the reset
//     button, and the keyboard nudges. Writes both DOM (preview style vars,
//     handle state, readouts) and a $brand.logo signal. No iframe crossing.
//
//  2. updateHeaderLogo (transitional iframe delegation) — the legacy JS
//     implementation reaches across the storefront preview iframe to mutate
//     the .brand / .brand__logo DOM directly (see the legacy bundle (removed 2026-05-27)
//     line 1027). Per
//     ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
//     the long-term destination is a $preview.brand.headerLogo shared signal
//     subscribed by the preview document. That subscriber ships in slice 6
//     (GoSXStudioPreviewRuntime burn-down). Until then, slice 3's island
//     keeps the same iframe contract by delegating to the legacy
//     window.GoSXStudioPreviewRuntime.updateHeaderLogo — ownership moves to
//     the brand island, the mechanism stays. Slice 6 will swap the
//     mechanism to the shared-signal write without revisiting brand
//     ownership.
//
// Each method is exercised by its own test() call in
// parity_e2e/brandruntime_test.ts so the parity claim is not collapsed to a
// single boolean.
package brandruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key consumers set to
// activate the .gsx-island BrandRuntime path. Default value (omitted from a
// host's flag map) keeps the legacy JS implementation active.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1, 2, 4..7
// each declare their own constant of the same shape.
const FeatureFlagKey = "brand-runtime-islands"

// IslandGlobals lists the window globals each Brand island publishes itself
// at when it mounts. The shim returned by BridgeShim looks these up at call
// time so unmounted islands fall back to the legacy implementation.
//
// Naming convention: window.__gosx_brand_runtime_island_<methodName>.
// BrandRuntime declares two public methods (bindLogo, updateHeaderLogo), so
// the struct holds two fields. Slices with single-method contracts (e.g.,
// selectionruntime) hold one; slices with larger fan-outs (e.g.,
// fieldruntime) hold one per method.
var IslandGlobals = struct {
	BindLogo         string
	UpdateHeaderLogo string
}{
	BindLogo:         "__gosx_brand_runtime_island_bindLogo",
	UpdateHeaderLogo: "__gosx_brand_runtime_island_updateHeaderLogo",
}

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime
// bundle appends to gate window.GoSXStudioBrandRuntime through the
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
// references the existing in-bundle functions by name (bindBrandLogo,
// updateHeaderLogo) so the bundle remains a single self-contained IIFE.
// After Section E deletes the legacy functions, the shim is rewritten to
// remove the fallback branch.
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

// IslandRuntimeJS returns the JS that publishes the
// window.__gosx_brand_runtime_island_* globals BridgeShim delegates to.
// Sourced from island_runtime.js (embedded). Companion to the .gsx islands
// in this package: the .gsx files are the mount-point markers in the editor
// DOM; this script is the actual behavior layer for slice 3, sized so the
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
  // Phase 3 slice-3 BrandRuntime island bridge.
  // See gosx-studio/brandruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  // The consumer surfaces ShellConfig.FeatureFlags via a
  // data-gosx-studio-feature-flag-<key>="true" attribute on the document
  // root (or any ancestor of the editor mount). This keeps the flag
  // decision out of cookies / query params at the bundle level so SSR and
  // CSR observations agree. The attribute name below MUST be the literal
  // "data-gosx-studio-feature-flag-brand-runtime-islands" so a grep over
  // the bundle finds it; the test
  // TestEngineRuntimeIncludesBrandRuntimeIslandBundle enforces this.
  var FLAG_ATTR = "data-gosx-studio-feature-flag-brand-runtime-islands";
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
  // (bindBrandLogo / updateHeaderLogo) remain defined in the bundle and
  // are passed in as the fallback path. The literal island-global names
  // below MUST match brandruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  var legacy = window.GoSXStudioBrandRuntime || {};
  window.GoSXStudioBrandRuntime = {
    bindLogo: delegate(
      "__gosx_brand_runtime_island_bindLogo",
      typeof bindBrandLogo === "function" ? bindBrandLogo : legacy.bindLogo
    ),
    updateHeaderLogo: delegate(
      "__gosx_brand_runtime_island_updateHeaderLogo",
      typeof updateHeaderLogo === "function" ? updateHeaderLogo : legacy.updateHeaderLogo
    )
  };
})();
`
