// Package previewruntime is the Go-side surface for the Phase 3 slice-6
// architectural burn-down of GoSXStudioPreviewRuntime.
//
// The slice replaces the JavaScript-implemented PreviewRuntime (ten public
// methods declared at the legacy bundle (removed 2026-05-27) —
// mount, setBlockVisibility, applyTextUpdate, applyTheme, applyStyleImpact,
// applyCSS, applyFonts, updateHeaderLogo, requestInlineEdit, cycleField)
// with two coordinated `.gsx`-authored pieces:
//
//  1. **Editor-side writer** — IslandRuntimeJS() publishes
//     window.__gosx_preview_runtime_island_<method> globals that the
//     BridgeShim delegates to when the FeatureFlagKey flag is on. Each
//     writer publishes to a `$preview.*` shared signal (via
//     window.__gosx_set_shared_signal_json) instead of mutating iframe DOM
//     directly.
//
//  2. **Preview-side subscriber** — PreviewSubscriberScript() returns a
//     separate JS bundle that the storefront mounts when loaded in the
//     editor preview iframe. The subscriber reads `$preview.*` shared
//     signals (which arrive cross-frame through gosx's postMessage relay
//     per ADR 0009) and applies the corresponding DOM mutations locally,
//     inside the iframe.
//
// Both implementations ship additively for at least seven days of CI green
// (see ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-6-previewruntime.md
// Section I). After the deletion window closes, the legacy
// the legacy bundle (removed 2026-05-27) (1,130 lines) is DELETED entirely
// — not just shrunk — as a follow-up commit.
//
// # The 10 methods + signal-namespace mapping
//
// The catalog below mirrors the table in the slice plan (which itself
// mirrors the parity matrix PreviewRuntime row). Each method's editor-side
// writer publishes to a $preview.* signal; each preview-side subscriber
// applies the same DOM action the legacy the legacy bundle performed.
//
//  1. mount(root)
//     Signal target: $preview.mount.epoch (number).
//     Subscriber action: re-runs the storefront's preview-readiness
//     initialization (binds the preview workbench, hydrates outlines and
//     field hotspots). The legacy the legacy bundle mount(root) walked
//     every [data-editor-workbench] in root and called bindPreview(form)
//     to wire the drag/drop/inline-edit machinery for the editor preview
//     overlay. Slice 6's subscriber runs the same bindPreview() routine
//     when the mount epoch changes.
//
//  2. setBlockVisibility(key, visible)
//     Signal target: $preview.block.<key>.visible (boolean).
//     Subscriber action: toggles display style of the matching
//     [data-studio-block-key="<key>"] node in the iframe document.
//     Mirrors setPreviewBlockVisibility at the legacy bundle:866.
//
//  3. applyTextUpdate(detail)
//     Signal target: $preview.text.<source> (string + attribute meta).
//     Subscriber action: updates textContent on the matching
//     [data-editor-preview="<source>"] target, and/or sets the attribute
//     payload (frameTarget / attrTarget / attrName / attrPrefix /
//     attrSuffix) on the matching selector. Mirrors applyTextUpdate at
//     the legacy bundle:912.
//
//  4. applyTheme(detail)
//     Signal targets: $preview.theme.{kit, template, palette, imageRatio,
//     customClasses, styleClasses, colors} (all string or string[]).
//     Subscriber action: rewrites the .site-shell class list (clearing
//     prior theme--template- / -kit- / -custom- / -nav- / -buttons- / -
//     cards- / -spacing- / -images- / -motion- / -palette- / -image-
//     prefixes) and reapplies the new theme classes; sets each color-
//     token CSS variable on documentElement / body / .site-shell. Mirrors
//     applyTheme at the legacy bundle:938.
//
//  5. applyStyleImpact(selector)
//     Signal target: $preview.style.impact.selector (string).
//     Subscriber action: clears every existing [data-studio-style-impact-
//     node] attribute in the iframe document; then, if selector is non-
//     empty, sets the attribute on each matching node and counts. Returns
//     the count via a $preview.style.impact.count echo signal so the
//     editor's showImpact readout still has the affected-count value.
//     Mirrors applyStyleImpact at the legacy bundle:986.
//
//  6. applyCSS(cssText)
//     Signal target: $preview.theme.cssCustom (string).
//     Subscriber action: ensures a <style data-editor-live-css> element
//     exists in the iframe document head and sets its textContent to the
//     payload. Mirrors applyCSS / applyPreviewStyle at preview-
//     runtime.js:1013.
//
//  7. applyFonts(cssText)
//     Signal target: $preview.theme.fontsCustom (string).
//     Subscriber action: same as applyCSS but with
//     [data-editor-live-fonts] attribute marker. Mirrors applyFonts at
//     the legacy bundle:1023.
//
//  8. updateHeaderLogo(detail)
//     Signal targets: $preview.brand.headerLogo.{src, alt, width, x, y}
//     (string / number).
//     Subscriber action: toggles the .brand--with-logo class, sets the
//     .brand__logo src + alt + hidden, sets --brand-logo-width / -offset-
//     x / -offset-y CSS variables on .brand. Mirrors updateHeaderLogo at
//     the legacy bundle:1027.
//
//  9. requestInlineEdit(detail)
//     Signal target: $preview.editor.inlineEdit.requestId (number).
//     Subscriber action: walks the registered preview controllers (the
//     bindPreview state) and dispatches requestInlineEdit(detail) on the
//     first matching one. Returns a handled boolean via
//     $preview.editor.inlineEdit.handled.<requestId> echo. Mirrors
//     requestInlineEdit at the legacy bundle:1049.
//
//  10. cycleField(detail)
//      Signal target: $preview.editor.fieldCycle.requestId (number).
//      Subscriber action: same fan-out as requestInlineEdit but calls
//      controller.cycleField(detail). Mirrors cycleField at preview-
//      runtime.js:1058.
//
// Per ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// (Decision section, restored 2026-05-26 by ADR 0009) the $preview.*
// namespace is the only legal cross-frame channel for preview mutations.
// The relay infrastructure that delivers these signals across the iframe
// boundary lives in github.com/odvcencio/gosx — see
// gosx/client/bridge/cross_frame.go (Bridge.EnableCrossFrameRelay) +
// gosx/client/js/relay.js + gosx/island/preview_bootstrap.go
// (island.EnablePreviewBootstrap).
//
// Each method is exercised by its own test() call in
// parity_e2e/previewruntime_test.ts so the parity claim is not collapsed
// to a single boolean.
package previewruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

//go:embed subscriber_runtime.js
var subscriberRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key consumers set
// to activate the .gsx-island PreviewRuntime path. Default value (omitted
// from a host's flag map) keeps the legacy JS implementation active in
// the legacy bundle (removed 2026-05-27).
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1, 2, 3,
// 4, 5, 7 each declare their own constant of the same shape.
const FeatureFlagKey = "preview-runtime-islands"

// IslandGlobals lists the window globals the editor-side island writer
// publishes itself at when it loads. The shim returned by BridgeShim looks
// these up at call time so an absent global falls back to the legacy
// implementation in the legacy bundle.
//
// Naming convention: window.__gosx_preview_runtime_island_<methodName>.
// PreviewRuntime declares ten public methods (mount, setBlockVisibility,
// applyTextUpdate, applyTheme, applyStyleImpact, applyCSS, applyFonts,
// updateHeaderLogo, requestInlineEdit, cycleField), so the struct holds
// ten fields.
var IslandGlobals = struct {
	Mount             string
	SetBlockVisibility string
	ApplyTextUpdate   string
	ApplyTheme        string
	ApplyStyleImpact  string
	ApplyCSS          string
	ApplyFonts        string
	UpdateHeaderLogo  string
	RequestInlineEdit string
	CycleField        string
}{
	Mount:              "__gosx_preview_runtime_island_mount",
	SetBlockVisibility: "__gosx_preview_runtime_island_setBlockVisibility",
	ApplyTextUpdate:    "__gosx_preview_runtime_island_applyTextUpdate",
	ApplyTheme:         "__gosx_preview_runtime_island_applyTheme",
	ApplyStyleImpact:   "__gosx_preview_runtime_island_applyStyleImpact",
	ApplyCSS:           "__gosx_preview_runtime_island_applyCSS",
	ApplyFonts:         "__gosx_preview_runtime_island_applyFonts",
	UpdateHeaderLogo:   "__gosx_preview_runtime_island_updateHeaderLogo",
	RequestInlineEdit:  "__gosx_preview_runtime_island_requestInlineEdit",
	CycleField:         "__gosx_preview_runtime_island_cycleField",
}

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime
// bundle appends to gate window.GoSXStudioPreviewRuntime through the
// FeatureFlagKey flag.
//
// When the host has marked FeatureFlagKey true (read by the consumer and
// surfaced to the page via the data-gosx-studio-feature-flag-preview-
// runtime-islands="true" attribute), the shim dispatches each method to
// its IslandGlobals entry on window. When the flag is off or the island
// global is missing (the writer never loaded), the shim falls back to the
// legacy implementation that lived in the now-deleted legacy bundle.
//
// The shim references the legacy implementations by their preview-
// runtime.js IIFE-exported names (window.GoSXStudioPreviewRuntime.mount /
// .setBlockVisibility / etc.) — the IIFE has already run by the time the
// shim runs, so window.GoSXStudioPreviewRuntime is the legacy object the
// shim wraps. Slice-6 deletion (after the 7-day window) removes the
// legacy fallback branch.
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

// IslandRuntimeJS returns the JS that publishes the
// window.__gosx_preview_runtime_island_* globals BridgeShim delegates to.
// Sourced from island_runtime.js (embedded). This is the EDITOR-side
// writer that publishes to $preview.* shared signals. The preview-side
// subscriber that consumes those signals ships separately as
// PreviewSubscriberScript().
func IslandRuntimeJS() []byte {
	return islandRuntimeJS
}

// PreviewSubscriberScript returns the JS that the storefront mounts when
// loaded in the editor preview iframe (per ADR 0009). It subscribes to
// $preview.* shared signals via window.__gosx_observe_shared_signal (the
// in-frame fan-out hook the cross-frame relay invokes when the relay
// delivers a peer message) and applies the corresponding DOM mutations
// locally inside the iframe.
//
// Sourced from subscriber_runtime.js (embedded). Served separately from
// the editor bundle so storefront pages don't ship the editor-side island
// writer code.
func PreviewSubscriberScript() []byte {
	return subscriberRuntimeJS
}

// Bundle returns the IslandRuntimeJS + BridgeShim concatenation that the
// studio runtime asset pipeline serves. Order
// matters: the island runtime publishes globals; the shim consults them.
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
  // Phase 3 slice-6 PreviewRuntime island bridge.
  // See gosx-studio/previewruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  // The consumer surfaces ShellConfig.FeatureFlags via a
  // data-gosx-studio-feature-flag-<key>="true" attribute on the document
  // root (or any ancestor of the editor mount). This keeps the flag
  // decision out of cookies / query params at the bundle level so SSR and
  // CSR observations agree. The attribute name below MUST be the literal
  // "data-gosx-studio-feature-flag-preview-runtime-islands" so a grep over
  // the bundle finds it; the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  var FLAG_ATTR = "data-gosx-studio-feature-flag-preview-runtime-islands";
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
  // that consults the feature flag on every call. the legacy bundle's
  // IIFE has already run by the time this shim runs (the legacy bundle
  // is concatenated into the bundle BEFORE the slice bundles — see
  // gosx-studio/runtime.go EngineRuntimeScript()), so
  // window.GoSXStudioPreviewRuntime is the legacy object whose methods
  // we wrap as the fallback path. The island-global names below MUST
  // match previewruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  var legacy = window.GoSXStudioPreviewRuntime || {};
  window.GoSXStudioPreviewRuntime = {
    mount: delegate(
      "__gosx_preview_runtime_island_mount",
      typeof legacy.mount === "function" ? legacy.mount : undefined
    ),
    setBlockVisibility: delegate(
      "__gosx_preview_runtime_island_setBlockVisibility",
      typeof legacy.setBlockVisibility === "function" ? legacy.setBlockVisibility : undefined
    ),
    applyTextUpdate: delegate(
      "__gosx_preview_runtime_island_applyTextUpdate",
      typeof legacy.applyTextUpdate === "function" ? legacy.applyTextUpdate : undefined
    ),
    applyTheme: delegate(
      "__gosx_preview_runtime_island_applyTheme",
      typeof legacy.applyTheme === "function" ? legacy.applyTheme : undefined
    ),
    applyStyleImpact: delegate(
      "__gosx_preview_runtime_island_applyStyleImpact",
      typeof legacy.applyStyleImpact === "function" ? legacy.applyStyleImpact : undefined
    ),
    applyCSS: delegate(
      "__gosx_preview_runtime_island_applyCSS",
      typeof legacy.applyCSS === "function" ? legacy.applyCSS : undefined
    ),
    applyFonts: delegate(
      "__gosx_preview_runtime_island_applyFonts",
      typeof legacy.applyFonts === "function" ? legacy.applyFonts : undefined
    ),
    updateHeaderLogo: delegate(
      "__gosx_preview_runtime_island_updateHeaderLogo",
      typeof legacy.updateHeaderLogo === "function" ? legacy.updateHeaderLogo : undefined
    ),
    requestInlineEdit: delegate(
      "__gosx_preview_runtime_island_requestInlineEdit",
      typeof legacy.requestInlineEdit === "function" ? legacy.requestInlineEdit : undefined
    ),
    cycleField: delegate(
      "__gosx_preview_runtime_island_cycleField",
      typeof legacy.cycleField === "function" ? legacy.cycleField : undefined
    )
  };
  // Auto-mount on document ready. Mirrors the v0.4.1 fieldruntime fix: the
  // deleted preview-runtime.js bundle's CanvasEngine factory called
  // GoSXStudioPreviewRuntime.mount(document) on engine mount. When the
  // bundle and the engine factory were both deleted, the per-slice
  // BridgeShim correctly re-published window.GoSXStudioPreviewRuntime but
  // the auto-mount call was lost — nothing invoked .mount(document) on page
  // load anymore, so the preview epoch signal ($preview.mount.epoch) was
  // never bumped at boot and the preview iframe's subscriber never wired
  // its bindPreview pass on initial load. The other nine methods on the
  // PreviewRuntime contract (setBlockVisibility, applyTextUpdate,
  // applyTheme, applyStyleImpact, applyCSS, applyFonts, updateHeaderLogo,
  // requestInlineEdit, cycleField) are host-driven writes, not boot
  // bindings, so they're NOT auto-mounted. Idempotent: a per-document
  // marker attribute guards against double-mount if a host also calls
  // .mount explicitly during SSR hydration.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-preview-runtime-auto-mounted";
      if (document.documentElement && document.documentElement.getAttribute(marker) === "true") return;
      if (document.documentElement) document.documentElement.setAttribute(marker, "true");
      window.GoSXStudioPreviewRuntime.mount(document.body || document);
    } catch (e) {
      // Auto-mount must never break the page; the host can always invoke
      // window.GoSXStudioPreviewRuntime.mount manually if it needs to recover.
    }
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
`
