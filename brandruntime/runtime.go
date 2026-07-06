// Package brandruntime is the Go-side surface for GoSXStudioBrandRuntime.
//
// BrandRuntime has two public methods, bindLogo and updateHeaderLogo. The
// .gsx-authored islands in this package publish well-known window globals
// (see BridgeShim below); the JS shim emitted by BridgeShim delegates
// window.GoSXStudioBrandRuntime calls directly to those globals. An unmounted
// island returns undefined.
//
// # Method fan-out
//
//  1. bindLogo (editor-side only) — wires the editor's brand-logo authoring
//     surface: URL/alt/width/X/Y/snap inputs, the drag-handle, the reset
//     button, and the keyboard nudges. Writes both DOM (preview style vars,
//     handle state, readouts) and a $brand.logo signal. No iframe crossing.
//
//  2. updateHeaderLogo — publishes $preview.brand.headerLogo for the
//     storefront preview document and $brand.headerLogo for editor-frame
//     consumers. The preview subscriber applies the .brand / .brand__logo DOM
//     updates inside the preview frame.
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

// FeatureFlagKey is retained as the studio.ShellConfig.FeatureFlags key that
// host shells may probe or emit as a stable data attribute. BrandRuntime no
// longer branches on this key; the island path is the runtime path.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1, 2, 4..7
// each declare their own constant of the same shape.
const FeatureFlagKey = "brand-runtime-islands"

// IslandGlobals lists the window globals each Brand island publishes itself
// at when it mounts. The shim returned by BridgeShim looks these up at call
// time so unmounted islands return undefined.
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
// bundle appends to publish window.GoSXStudioBrandRuntime.
//
// Each public method dispatches directly to its corresponding IslandGlobals
// entry on window. If the island has not mounted, the method returns
// undefined.
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
  // BrandRuntime island bridge.
  // See gosx-studio/brandruntime/runtime.go for the contract.
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
  // Install the runtime object. The literal island-global names below MUST
  // match brandruntime.IslandGlobals in runtime.go; the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  window.GoSXStudioBrandRuntime = {
    bindLogo: delegate("__gosx_brand_runtime_island_bindLogo"),
    updateHeaderLogo: delegate("__gosx_brand_runtime_island_updateHeaderLogo")
  };
  // Auto-mount on document ready. Mirrors the v0.4.1 fieldruntime fix: the
  // deleted studio-engines.js bundle ran a top-level init() at
  // DOMContentLoaded that called bindBrandLogo(document) along with the six
  // sibling runtime binds. When the bundle was deleted the per-slice
  // BridgeShim correctly re-published window.GoSXStudioBrandRuntime but the
  // auto-mount was lost — the brand logo URL/width/offset input listeners
  // were never attached on initial load, so authoring the brand logo from a
  // cold boot did nothing. updateHeaderLogo is NOT auto-mounted because the
  // legacy init() only called bindBrandLogo at boot; updateHeaderLogo is a
  // host-driven preview-side write triggered by user edits. Idempotent: a
  // per-document marker attribute guards against double-mount if a host also
  // calls .bindLogo explicitly during SSR hydration.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-brand-runtime-auto-mounted";
      if (document.documentElement && document.documentElement.getAttribute(marker) === "true") return;
      if (document.documentElement) document.documentElement.setAttribute(marker, "true");
      window.GoSXStudioBrandRuntime.bindLogo(document.body || document);
    } catch (e) {
      // Auto-mount must never break the page; the host can always invoke
      // window.GoSXStudioBrandRuntime.bindLogo manually if it needs to recover.
    }
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
`
