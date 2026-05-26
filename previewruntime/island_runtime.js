// island_runtime.js — editor-side writer for the previewruntime islands.
//
// Phase 3 slice-6 burn-down of GoSXStudioPreviewRuntime (10 methods at
// gosx-studio/assets/preview-runtime.js:1118). This script runs on the
// editor page; it publishes the
// window.__gosx_preview_runtime_island_<method> globals that
// previewruntime.BridgeShim delegates to when the
// "preview-runtime-islands" feature flag is on. Each writer publishes to
// a $preview.* shared signal via window.__gosx_set_shared_signal_json;
// the cross-frame postMessage relay (ADR 0009, shipped in
// gosx/client/bridge/cross_frame.go + gosx/client/js/relay.js) routes the
// write to the storefront iframe, where subscriber_runtime.js applies
// the corresponding DOM mutation locally.
//
// Companion file:
//   - subscriber_runtime.js (preview-side) — reads the same signals and
//     mutates the iframe DOM.
//
// Bundle order: this file is embedded by previewruntime.IslandRuntimeJS();
// previewruntime.Bundle() concatenates this file then BridgeShim, and
// gosx-studio/runtime.go appends Bundle() to EngineRuntimeScript() AFTER
// preview-runtime.js (the legacy 1130-line bridge). At runtime, the legacy
// preview-runtime.js IIFE runs first and installs
// window.GoSXStudioPreviewRuntime; this island_runtime.js then runs and
// publishes the __gosx_preview_runtime_island_* globals; finally
// BridgeShim runs and replaces window.GoSXStudioPreviewRuntime with the
// flag-aware delegating shim.

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  // Set a shared signal value through the WASM bridge. The bridge exposes
  // window.__gosx_set_shared_signal_json(name, valueJSON) once the
  // runtime boots. The cross-frame relay (per ADR 0009 + Bridge.
  // EnableCrossFrameRelay) routes writes whose name starts with "$preview."
  // to the iframe's Bridge, where subscriber_runtime.js's
  // __gosx_observe_shared_signal subscribers fire.
  //
  // If the bridge is not ready, the write is a no-op — but during the
  // additive shipping window, BridgeShim's legacy fallback still keeps the
  // preview live via window.GoSXStudioPreviewRuntime (the IIFE-exported
  // legacy implementation in preview-runtime.js).
  function writeSharedSignal(name, value) {
    try {
      var setter = window.__gosx_set_shared_signal_json;
      if (typeof setter === "function") {
        setter(name, JSON.stringify(value));
      }
    } catch (error) {
      // Bridge not ready yet — legacy fallback keeps the preview live.
    }
  }

  // monotonicCounter — used by requestInlineEdit / cycleField to mint a
  // unique request id every call. The subscriber side observes the id
  // change (not the value content) so repeated identical detail payloads
  // still trigger work.
  var monotonicCounter = 0;
  function nextRequestId() {
    monotonicCounter += 1;
    return monotonicCounter;
  }

  // mountEpoch — incremented every call to the mount island. Subscribers
  // re-run their bindPreview routine when the epoch changes so the editor
  // can force a remount (e.g., after a hot-reload).
  var mountEpoch = 0;

  // ===== Method 1/10: mount(root) =====
  //
  // mount(root) — mirrors window.GoSXStudioPreviewRuntime.mount at
  // preview-runtime.js:1119 (the .mount field on the IIFE-exported
  // object, bound to init at line 1113).
  //
  // The legacy mount() walks every [data-editor-workbench] in root and
  // calls bindPreview(form) to wire the drag/drop/inline-edit machinery
  // for the editor preview overlay. The slice-6 writer just bumps the
  // mount epoch signal; the preview-side subscriber re-runs its
  // bindPreview routine when the epoch changes.
  function mountPreviewIsland(root) {
    mountEpoch += 1;
    writeSharedSignal("$preview.mount.epoch", mountEpoch);
  }

  window.__gosx_preview_runtime_island_mount = mountPreviewIsland;

  // ===== Method 2/10: setBlockVisibility(key, visible) =====
  //
  // setBlockVisibility(key, visible) — mirrors setBlockVisibility at
  // preview-runtime.js:873.
  //
  // Editor-side writer: $preview.block.<key>.visible := !!visible.
  // Subscriber-side: toggles display style on the matching
  // [data-studio-block-key="<key>"] node in the iframe.
  function setBlockVisibilityIsland(key, visible) {
    if (!key) return;
    writeSharedSignal("$preview.block." + key + ".visible", !!visible);
  }

  window.__gosx_preview_runtime_island_setBlockVisibility = setBlockVisibilityIsland;

  // ===== Method 3/10: applyTextUpdate(detail) =====
  //
  // applyTextUpdate(detail) — mirrors applyTextUpdate at preview-
  // runtime.js:912.
  //
  // Editor-side writer: publishes the full detail object to
  // $preview.text.<sourceKey>. The subscriber re-emits all three legacy
  // branches (sourceKey textContent, frameTarget textContent, attrTarget
  // attribute assignment) from the detail payload. When detail.sourceKey
  // is empty, the signal name falls back to a $preview.text._.attr suffix
  // keyed off attrTarget so attribute-only updates still route. See the
  // subscriber side for the exact dispatch.
  function applyTextUpdateIsland(detail) {
    detail = detail || {};
    var key = detail.sourceKey || detail.frameTarget || detail.attrTarget || "_";
    writeSharedSignal("$preview.text." + key, {
      sourceKey: detail.sourceKey || "",
      frameTarget: detail.frameTarget || "",
      attrTarget: detail.attrTarget || "",
      attrName: detail.attrName || "",
      attrPrefix: detail.attrPrefix || "",
      attrSuffix: detail.attrSuffix || "",
      value: detail.value || ""
    });
  }

  window.__gosx_preview_runtime_island_applyTextUpdate = applyTextUpdateIsland;

  // ===== Method 4/10: applyTheme(detail) =====
  //
  // applyTheme(detail) — mirrors applyTheme at preview-runtime.js:938.
  //
  // Editor-side writer: publishes the theme payload to a family of
  // $preview.theme.* signals. The subscriber rebuilds the .site-shell
  // class list (clearing legacy theme--template- / -kit- / -custom- /
  // -nav- / -buttons- / -cards- / -spacing- / -images- / -motion- /
  // -palette- / -image- prefixes) and reapplies the new classes; then
  // sets each color-token CSS variable on documentElement / body /
  // .site-shell.
  function applyThemeIsland(detail) {
    detail = detail || {};
    writeSharedSignal("$preview.theme.kit", detail.kit || "");
    writeSharedSignal("$preview.theme.template", detail.template || "");
    writeSharedSignal("$preview.theme.palette", detail.palette || "");
    writeSharedSignal("$preview.theme.imageRatio", detail.imageRatio || "");
    writeSharedSignal("$preview.theme.customClasses", Array.isArray(detail.customClasses) ? detail.customClasses : []);
    writeSharedSignal("$preview.theme.styleClasses", Array.isArray(detail.styleClasses) ? detail.styleClasses : []);
    writeSharedSignal("$preview.theme.colors", detail.colors || {});
  }

  window.__gosx_preview_runtime_island_applyTheme = applyThemeIsland;

  // ===== Method 5/10: applyStyleImpact(selector) =====
  //
  // applyStyleImpact(selector) — mirrors applyStyleImpact at preview-
  // runtime.js:986.
  //
  // The legacy returns an integer count of marked nodes; the signal-
  // based path is asynchronous (writes propagate via postMessage). To
  // preserve a useful return value we synchronously count nodes that
  // would have matched in the editor's view of the iframe (best-effort
  // via the .editor-preview-frame contentDocument when same-origin), and
  // publish the signal for the iframe-side subscriber to perform the
  // authoritative mutation. When the editor cannot read the iframe
  // (cross-origin or not yet loaded), the count is 0 — consumers should
  // treat the count as advisory.
  function applyStyleImpactIsland(selector) {
    writeSharedSignal("$preview.style.impact.selector", selector || "");
    var count = 0;
    try {
      Array.prototype.forEach.call(doc.querySelectorAll(".editor-preview-frame"), function (frame) {
        var idoc = frame && frame.contentDocument;
        if (!idoc) return;
        var scope = idoc.querySelector(".site-shell") || idoc;
        if (!selector) return;
        Array.prototype.forEach.call(scope.querySelectorAll(selector), function () {
          count += 1;
        });
      });
    } catch (error) {
      // Cross-origin iframe or detached document; advisory count stays 0.
    }
    return count;
  }

  window.__gosx_preview_runtime_island_applyStyleImpact = applyStyleImpactIsland;

  // ===== Method 6/10: applyCSS(cssText) =====
  //
  // applyCSS(cssText) — mirrors applyCSS at preview-runtime.js:1019.
  //
  // Editor-side writer: $preview.theme.cssCustom := cssText.
  // Subscriber-side: ensures a <style data-editor-live-css> element in
  // the iframe head and sets its textContent.
  function applyCSSIsland(cssText) {
    writeSharedSignal("$preview.theme.cssCustom", String(cssText || ""));
  }

  window.__gosx_preview_runtime_island_applyCSS = applyCSSIsland;

  // ===== Method 7/10: applyFonts(cssText) =====
  //
  // applyFonts(cssText) — mirrors applyFonts at preview-runtime.js:1023.
  //
  // Editor-side writer: $preview.theme.fontsCustom := cssText.
  // Subscriber-side: same as applyCSS but with [data-editor-live-fonts]
  // attribute marker on the <style> element.
  function applyFontsIsland(cssText) {
    writeSharedSignal("$preview.theme.fontsCustom", String(cssText || ""));
  }

  window.__gosx_preview_runtime_island_applyFonts = applyFontsIsland;

  // ===== Method 8/10: updateHeaderLogo(detail) =====
  //
  // updateHeaderLogo(detail) — mirrors updateHeaderLogo at preview-
  // runtime.js:1027.
  //
  // Editor-side writer: publishes the header-logo payload as a single
  // $preview.brand.headerLogo signal carrying src / alt / width / x / y.
  // The slice plan lists src / alt / width as the primary sub-keys;
  // offsets x / y are carried in the same payload object to preserve
  // legacy parity (the legacy reads detail.x / detail.y in addition to
  // url / alt / width). The subscriber decomposes the payload onto the
  // .brand DOM (img attributes + CSS custom properties).
  function updateHeaderLogoIsland(detail) {
    detail = detail || {};
    writeSharedSignal("$preview.brand.headerLogo", {
      src: detail.url || "",
      alt: detail.alt || "Site logo",
      width: detail.width || "96",
      x: detail.x || "0",
      y: detail.y || "0"
    });
  }

  window.__gosx_preview_runtime_island_updateHeaderLogo = updateHeaderLogoIsland;

  // ===== Method 9/10: requestInlineEdit(detail) =====
  //
  // requestInlineEdit(detail) — mirrors requestInlineEdit at preview-
  // runtime.js:1049.
  //
  // The legacy returns a `handled` boolean from the first matching
  // preview controller. The signal path is asynchronous, so the
  // editor-side return value is best-effort (true if any
  // .editor-preview-frame exists, false otherwise). The authoritative
  // handling happens on the subscriber side.
  function requestInlineEditIsland(detail) {
    var id = nextRequestId();
    writeSharedSignal("$preview.editor.inlineEdit.requestId", id);
    writeSharedSignal("$preview.editor.inlineEdit.detail." + id, detail || {});
    return !!doc.querySelector(".editor-preview-frame");
  }

  window.__gosx_preview_runtime_island_requestInlineEdit = requestInlineEditIsland;

  // ===== Method 10/10: cycleField(detail) =====
  //
  // cycleField(detail) — mirrors cycleField at preview-runtime.js:1058.
  //
  // Same shape as requestInlineEdit but writes to
  // $preview.editor.fieldCycle.requestId. Subscriber dispatches
  // controller.cycleField(detail) on the first matching preview
  // controller.
  function cycleFieldIsland(detail) {
    var id = nextRequestId();
    writeSharedSignal("$preview.editor.fieldCycle.requestId", id);
    writeSharedSignal("$preview.editor.fieldCycle.detail." + id, detail || {});
    return !!doc.querySelector(".editor-preview-frame");
  }

  window.__gosx_preview_runtime_island_cycleField = cycleFieldIsland;
})();
