// Package fieldruntime is the Go-side surface for the Phase 3 slice-1
// island implementation of GoSXStudioFieldRuntime.
//
// FieldRuntime (three methods: bind / bindMirroring / bindClipboard) is
// implemented by the .gsx-authored islands living in this package's *.gsx
// files. Each island publishes itself onto a well-known window global (see
// BridgeShim below); the JS shim emitted by BridgeShim installs
// window.GoSXStudioFieldRuntime and dispatches each method to its island
// global at call time.
//
// The legacy monolithic JS implementation (bindFieldRuntime,
// bindFieldMirroring, bindFieldClipboard from the now-deleted
// studio-engines.js bundle) is gone as of 2026-05-27. The FeatureFlagKey
// flag remains in the registry for host-probe stability but no longer
// gates anything — the island path always runs.
//
// Mirroring writes go through $preview.field.<name> shared signals per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// rather than reaching across the preview iframe directly.
package fieldruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key for the
// FieldRuntime island path. Post-2026-05-27 the legacy fallback is gone
// and the flag is kept only for host-probe API stability — the island
// path runs regardless of its value.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 2..7 each
// declare their own constant of the same shape.
const FeatureFlagKey = "field-runtime-islands"

// IslandGlobals lists the window globals each Field island publishes itself
// at when it mounts. The shim returned by BridgeShim looks these up at call
// time.
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
// bundle appends to install window.GoSXStudioFieldRuntime. Each method
// dispatches to the corresponding IslandGlobals entry on window. If the
// island is not yet mounted, the shim returns undefined (the legacy JS
// fallback was deleted on 2026-05-27).
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

// IslandRuntimeJS returns the JS that publishes the
// window.__gosx_field_runtime_island_* globals BridgeShim delegates to.
// Sourced from island_runtime.js (embedded). Companion to the .gsx islands
// in this package: the .gsx files are the mount-point markers in the
// editor DOM; this script is the actual behavior layer for slice 1, sized
// so the slice ships independently of the inspector form being
// signal-driven (which is a separate workstream).
func IslandRuntimeJS() []byte {
	return islandRuntimeJS
}

// Bundle returns the IslandRuntimeJS + BridgeShim concatenation that the
// studio runtime asset pipeline serves. Order matters: the island runtime
// publishes globals; the shim consults them.
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
  // Phase 3 slice-1 FieldRuntime island bridge.
  // See gosx-studio/fieldruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  // The consumer surfaces ShellConfig.FeatureFlags via a
  // data-gosx-studio-feature-flag-<key>="true" attribute on the document
  // root (or any ancestor of the editor mount). This keeps the flag
  // decision out of cookies / query params at the bundle level so SSR and
  // CSR observations agree. The attribute name below MUST be the literal
  // "data-gosx-studio-feature-flag-field-runtime-islands" so a grep over
  // the bundle finds it; the test
  // TestEngineRuntimeIncludesFieldRuntimeIslandBundle enforces this.
  var FLAG_ATTR = "data-gosx-studio-feature-flag-field-runtime-islands";
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
  // legacy bundle's bindFieldRuntime / bindFieldMirroring / bindFieldClipboard
  // functions as a fallback path; with the legacy bundle deleted in Phase 3
  // Section E those identifiers are undefined and the fallback branch was
  // dead code. v0.5.0 removes the dead branch — the island global is the
  // only path. The literal island-global names below MUST match
  // fieldruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  window.GoSXStudioFieldRuntime = {
    bind: delegate("__gosx_field_runtime_island_bind"),
    bindMirroring: delegate("__gosx_field_runtime_island_bindMirroring"),
    bindClipboard: delegate("__gosx_field_runtime_island_bindClipboard")
  };
  // Auto-mount on document ready. The deleted studio-engines.js bundle ran a
  // top-level init() at DOMContentLoaded that called bindFieldRuntime(document)
  // (plus the six sibling runtime binds). When the bundle was deleted the per-
  // slice BridgeShims correctly re-published window.GoSXStudio*Runtime but the
  // auto-mount was lost — the field-mirror parity test caught the gap because
  // it asserts the input's bound-attr marker exists after page load, which
  // only happens if .bind ran. Restoring the auto-mount inside the shim keeps
  // the responsibility local to the slice and matches the legacy contract.
  // Idempotent: a per-document marker attribute guards against double-mount
  // if a host also calls .bind explicitly during SSR hydration.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-field-runtime-auto-mounted";
      if (document.documentElement && document.documentElement.getAttribute(marker) === "true") return;
      if (document.documentElement) document.documentElement.setAttribute(marker, "true");
      window.GoSXStudioFieldRuntime.bind(document.body || document);
    } catch (e) {
      // Auto-mount must never break the page; the host can always invoke
      // window.GoSXStudioFieldRuntime.bind manually if it needs to recover.
    }
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
`
