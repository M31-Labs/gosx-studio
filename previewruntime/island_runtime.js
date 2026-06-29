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

  function queryAll(root, selector) {
    if (!root || !root.querySelectorAll) return [];
    return Array.prototype.slice.call(root.querySelectorAll(selector));
  }

  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function closestWorkbenchForm(node) {
    return node && node.closest ? node.closest("form[data-studio-workbench], form[data-editor-workbench]") : null;
  }

  function editorPreviewForm(detail, event) {
    detail = detail || {};
    if (detail.form && detail.form.querySelectorAll) return detail.form;
    return closestWorkbenchForm(event && event.target) || doc.querySelector("form[data-studio-workbench], form[data-editor-workbench]");
  }

  function editorPreviewShells(form) {
    return queryAll(form, "[data-gosx-studio-preview], [data-studio-preview-shell], [data-studio-canvas]");
  }

  function editorPreviewFrames(form) {
    return queryAll(form, ".editor-preview-frame, [data-studio-preview-frame]");
  }

  function editorPreviewFrameDocument(frame) {
    try {
      return frame.contentDocument || (frame.contentWindow && frame.contentWindow.document) || null;
    } catch (error) {
      return null;
    }
  }

  function editorPreviewURL(frame) {
    return frame.getAttribute("data-studio-preview-src") || frame.getAttribute("src") || "";
  }

  function editorPreviewPatchSelector(source) {
    source = attrValue(source);
    return [
      '[data-studio-field="' + source + '"]',
      '[data-editor-preview="' + source + '"]',
      '[data-studio-field-source="' + source + '"]'
    ].join(",");
  }

  function editorPreviewPatchTargets(frame, patch) {
    var frameDoc = editorPreviewFrameDocument(frame);
    var field = patch && patch.field;
    var source = field && (field.source || field.name);
    if (!frameDoc || !source) return [];
    return queryAll(frameDoc, editorPreviewPatchSelector(source));
  }

  function editorPreviewFieldKey(target) {
    if (!target || !target.getAttribute) return "";
    return target.getAttribute("data-studio-field") || target.getAttribute("data-editor-preview") || target.getAttribute("data-studio-field-source") || "";
  }

  function clearEditorPreviewFieldMap(frame) {
    var frameDoc = editorPreviewFrameDocument(frame);
    if (!frameDoc) return { handled: false, count: 0 };
    var count = 0;
    queryAll(frameDoc, "[data-gosx-studio-preview-field-scope], [data-gosx-studio-preview-field-current]").forEach(function (target) {
      target.removeAttribute("data-gosx-studio-preview-field-scope");
      target.removeAttribute("data-gosx-studio-preview-field-current");
      target.removeAttribute("data-gosx-studio-preview-field-position");
      target.removeAttribute("data-gosx-studio-preview-field-total");
      count += 1;
    });
    return { handled: true, count: count };
  }

  function syncEditorPreviewFieldMap(frame, fields, current, count) {
    var result = clearEditorPreviewFieldMap(frame);
    fields = Array.isArray(fields) ? fields : [];
    count = Number.isFinite(Number(count)) ? Number(count) : fields.length;
    fields.forEach(function (candidate, index) {
      if (!candidate || !candidate.setAttribute) return;
      candidate.setAttribute("data-gosx-studio-preview-field-scope", "true");
      candidate.setAttribute("data-gosx-studio-preview-field-position", String(index + 1));
      candidate.setAttribute("data-gosx-studio-preview-field-total", String(count));
      if (editorPreviewFieldKey(candidate) === current) {
        candidate.setAttribute("data-gosx-studio-preview-field-current", "true");
      }
    });
    return { handled: true, count: fields.length, cleared: result.count || 0 };
  }

  function clearEditorPreviewSelectionMarker(frame) {
    var frameDoc = editorPreviewFrameDocument(frame);
    if (!frameDoc) return { handled: false, count: 0 };
    var count = 0;
    queryAll(frameDoc, "[data-gosx-studio-preview-selected]").forEach(function (target) {
      target.removeAttribute("data-gosx-studio-preview-selected");
      count += 1;
    });
    return { handled: true, count: count };
  }

  function applyEditorPreviewSelectionMarker(frame, targets, clear) {
    var result = clear === false ? { count: 0 } : clearEditorPreviewSelectionMarker(frame);
    targets = Array.isArray(targets) ? targets : [];
    targets.forEach(function (target) {
      if (!target || !target.setAttribute) return;
      target.setAttribute("data-gosx-studio-preview-selected", "true");
    });
    return { handled: true, count: targets.length, cleared: result.count || 0 };
  }

  function clearEditorPreviewSelectionChrome(form) {
    if (!form) return { handled: false, shells: 0, frames: 0 };
    var shells = editorPreviewShells(form);
    var frames = editorPreviewFrames(form);
    shells.forEach(function (shell) {
      shell.removeAttribute("data-gosx-studio-preview-selection");
    });
    frames.forEach(function (frame) {
      frame.removeAttribute("data-studio-preview-selection");
    });
    return { handled: true, shells: shells.length, frames: frames.length };
  }

  function applyEditorPreviewSelectionChrome(form, frame, selection) {
    if (!form) return { handled: false, shells: 0, frames: 0 };
    selection = selection || "";
    var shells = editorPreviewShells(form);
    shells.forEach(function (shell) {
      shell.setAttribute("data-gosx-studio-preview-selection", selection);
    });
    if (frame && frame.setAttribute) {
      frame.setAttribute("data-studio-preview-selection", selection);
      return { handled: true, shells: shells.length, frames: 1 };
    }
    return { handled: true, shells: shells.length, frames: 0 };
  }

  function hideEditorPreviewDocks(form) {
    if (!form) return { handled: false, count: 0 };
    var docks = queryAll(form, "[data-gosx-studio-preview-dock]");
    docks.forEach(function (dock) {
      dock.hidden = true;
      dock.removeAttribute("data-gosx-studio-preview-field");
      dock.removeAttribute("data-gosx-studio-preview-block");
      dock.removeAttribute("data-gosx-studio-preview-action-label");
      dock.removeAttribute("data-gosx-studio-preview-action-href");
      dock.removeAttribute("data-gosx-studio-preview-action-formaction");
      dock.removeAttribute("data-gosx-studio-preview-block-label");
      dock.removeAttribute("data-gosx-studio-preview-field-count");
      dock.removeAttribute("data-gosx-studio-preview-field-index");
    });
    return { handled: true, count: docks.length };
  }

  function syncEditorPreviewDockFieldNavigation(dock, count, index) {
    if (!dock || !dock.setAttribute) return { handled: false, count: 0, index: -1 };
    count = Number.isFinite(Number(count)) ? Number(count) : 0;
    index = Number.isFinite(Number(index)) ? Number(index) : -1;
    dock.setAttribute("data-gosx-studio-preview-field-count", String(count));
    dock.setAttribute("data-gosx-studio-preview-field-index", index >= 0 ? String(index + 1) : "");
    var meter = dock.querySelector("[data-gosx-studio-preview-field-meter]");
    if (meter) {
      meter.hidden = count < 2 || index < 0;
      meter.textContent = count > 1 && index >= 0 ? "Field " + (index + 1) + " of " + count : "";
    }
    ["prev-field", "next-field"].forEach(function (action) {
      var button = dock.querySelector('[data-gosx-studio-preview-command="' + action + '"]');
      if (!button) return;
      button.hidden = count === 0;
      button.disabled = count < 2;
      button.setAttribute("aria-label", (action === "prev-field" ? "Previous" : "Next") + " editable field");
    });
    return { handled: true, count: count, index: index };
  }

  function updateEditorPreviewPatchTarget(target, field) {
    if (!target || !field) return;
    var value = field.value == null ? "" : String(field.value);
    var tag = String(target.tagName || "").toLowerCase();
    var editable = target.getAttribute("data-studio-editable") || target.getAttribute("data-studio-field-editable") || "";
    if (tag === "input" || tag === "textarea" || tag === "select") {
      target.value = value;
      if (field.type === "checkbox" || field.type === "radio") target.checked = !!field.checked;
    } else if (tag === "img" || tag === "video" || tag === "audio" || editable === "media" || editable === "image") {
      if (value) target.setAttribute("src", value);
    } else if (tag === "a" && (editable === "url" || editable === "link" || String(field.name || "").toLowerCase().indexOf("url") >= 0)) {
      target.setAttribute("href", value || "#");
    } else {
      target.textContent = value;
    }
    target.setAttribute("data-gosx-studio-preview-patched", "fresh");
    window.setTimeout(function () {
      if (target && target.setAttribute) target.setAttribute("data-gosx-studio-preview-patched", "true");
    }, 220);
  }

  function applyEditorPreviewPatch(frame, patch) {
    if (!frame || !patch || !patch.field) return { handled: false, count: 0 };
    var targets = editorPreviewPatchTargets(frame, patch);
    targets.forEach(function (target) {
      updateEditorPreviewPatchTarget(target, patch.field);
    });
    if (targets.length) {
      frame.setAttribute("data-studio-preview-patched-count", String(targets.length));
    }
    return { handled: true, count: targets.length };
  }

  function postEditorPreviewPatch(form, reason, patch) {
    var frames = editorPreviewFrames(form);
    var count = 0;
    if (!form || !frames.length || !patch) return { handled: false, count: 0 };
    if (!patch.field) {
      if (reason !== "load-sync") setEditorPreviewStatus(form, "dirty", "Live preview pending", reason || "patch");
      emitEditorPreview(form, "gosxstudio:preview-patch", patch);
      return { handled: true, count: 0 };
    }
    frames.forEach(function (frame) {
      if (!frame.contentWindow || !frame.getAttribute("src")) return;
      count += applyEditorPreviewPatch(frame, patch).count || 0;
      try {
        frame.contentWindow.postMessage(patch, new URL(frame.getAttribute("src"), window.location.href).origin);
      } catch (error) {
        try {
          frame.contentWindow.postMessage(patch, window.location.origin);
        } catch (ignored) {
          return;
        }
      }
    });
    if (reason !== "load-sync") setEditorPreviewStatus(form, "dirty", "Live preview pending", reason || "patch");
    emitEditorPreview(form, "gosxstudio:preview-patch", patch);
    return { handled: true, count: count };
  }

  function cacheBustPreviewURL(url, reason) {
    try {
      var next = new URL(url || window.location.href, window.location.href);
      next.searchParams.set("_gosx_preview", String(Date.now()));
      if (reason) next.searchParams.set("_gosx_preview_reason", reason);
      return next.pathname + next.search + next.hash;
    } catch (error) {
      return url || "";
    }
  }

  function emitEditorPreview(form, name, detail) {
    form.dispatchEvent(new CustomEvent(name, { bubbles: true, detail: detail || {} }));
  }

  function setEditorPreviewResult(detail, result) {
    if (detail) detail.result = result;
    return result;
  }

  function setEditorPreviewStatus(form, state, label, reason) {
    if (!form) return { handled: false };
    editorPreviewShells(form).forEach(function (shell) {
      shell.setAttribute("data-gosx-studio-preview-state", state || "");
      shell.setAttribute("data-gosx-studio-preview-reason", reason || "");
      queryAll(shell, "[data-studio-preview-status]").forEach(function (node) {
        node.textContent = label || "";
      });
    });
    editorPreviewFrames(form).forEach(function (frame) {
      frame.setAttribute("data-studio-preview-state", state || "");
    });
    return { handled: true };
  }

  function syncEditorPreviewRoute(form, route, reason) {
    route = route || "";
    if (!form || !route) return { handled: false };
    editorPreviewShells(form).forEach(function (shell) {
      shell.setAttribute("data-gosx-studio-preview-url", route);
    });
    editorPreviewFrames(form).forEach(function (frame) {
      frame.setAttribute("data-studio-preview-src", route);
    });
    queryAll(form, "[data-studio-open-preview]").forEach(function (link) {
      if (link.getAttribute("aria-disabled") === "true") return;
      link.setAttribute("href", route);
    });
    queryAll(form, "[data-studio-selected-flow-route]").forEach(function (node) {
      node.textContent = route;
    });
    emitEditorPreview(form, "gosxstudio:preview-route", { route: route, reason: reason || "" });
    return { handled: true };
  }

  function refreshEditorPreviewNow(form, reason, route) {
    if (!form) return { handled: false };
    reason = reason || "refresh";
    route = route || "";
    if (route) syncEditorPreviewRoute(form, route, reason);
    var frames = editorPreviewFrames(form);
    if (!frames.length) return { handled: false };
    setEditorPreviewStatus(form, "loading", "Refreshing preview", reason);
    frames.forEach(function (frame) {
      var base = route || editorPreviewURL(frame) || frame.getAttribute("src") || "/";
      frame.setAttribute("src", cacheBustPreviewURL(base, reason));
    });
    emitEditorPreview(form, "gosxstudio:preview-refresh", { route: route, reason: reason });
    return { handled: true };
  }

  var editorPreviewRefreshTimers = typeof WeakMap !== "undefined" ? new WeakMap() : null;
  function editorPreviewRefreshTimer(form) {
    return editorPreviewRefreshTimers ? editorPreviewRefreshTimers.get(form) : form.__gosxStudioEditorPreviewRefreshTimer;
  }

  function setEditorPreviewRefreshTimer(form, timer) {
    if (editorPreviewRefreshTimers) editorPreviewRefreshTimers.set(form, timer);
    else form.__gosxStudioEditorPreviewRefreshTimer = timer;
  }

  function scheduleEditorPreviewRefresh(form, reason, route) {
    if (!form) return { handled: false };
    window.clearTimeout(editorPreviewRefreshTimer(form));
    setEditorPreviewRefreshTimer(form, window.setTimeout(function () {
      refreshEditorPreviewNow(form, reason, route);
    }, 180));
    return { handled: true };
  }

  function bindEditorPreviewChromeEvents() {
    doc.addEventListener("gosxstudio:editor-preview-status-set", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, setEditorPreviewStatus(form, detail.state, detail.label, detail.reason));
    });
    doc.addEventListener("gosxstudio:editor-preview-route-sync", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, syncEditorPreviewRoute(form, detail.route, detail.reason));
    });
    doc.addEventListener("gosxstudio:editor-preview-refresh-now", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, refreshEditorPreviewNow(form, detail.reason, detail.route));
    });
    doc.addEventListener("gosxstudio:editor-preview-refresh-schedule", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, scheduleEditorPreviewRefresh(form, detail.reason, detail.route));
    });
    doc.addEventListener("gosxstudio:editor-preview-patch-apply", function (event) {
      var detail = event.detail || {};
      setEditorPreviewResult(detail, applyEditorPreviewPatch(detail.frame, detail.patch));
    });
    doc.addEventListener("gosxstudio:editor-preview-patch-post", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, postEditorPreviewPatch(form, detail.reason, detail.patch));
    });
    doc.addEventListener("gosxstudio:editor-preview-field-map-clear", function (event) {
      var detail = event.detail || {};
      setEditorPreviewResult(detail, clearEditorPreviewFieldMap(detail.frame));
    });
    doc.addEventListener("gosxstudio:editor-preview-field-map-sync", function (event) {
      var detail = event.detail || {};
      setEditorPreviewResult(detail, syncEditorPreviewFieldMap(detail.frame, detail.fields, detail.current, detail.count));
    });
    doc.addEventListener("gosxstudio:editor-preview-selection-marker-clear", function (event) {
      var detail = event.detail || {};
      setEditorPreviewResult(detail, clearEditorPreviewSelectionMarker(detail.frame));
    });
    doc.addEventListener("gosxstudio:editor-preview-selection-marker-apply", function (event) {
      var detail = event.detail || {};
      setEditorPreviewResult(detail, applyEditorPreviewSelectionMarker(detail.frame, detail.targets, detail.clear));
    });
    doc.addEventListener("gosxstudio:editor-preview-selection-chrome-clear", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, clearEditorPreviewSelectionChrome(form));
    });
    doc.addEventListener("gosxstudio:editor-preview-selection-chrome-apply", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, applyEditorPreviewSelectionChrome(form, detail.frame, detail.selection));
    });
    doc.addEventListener("gosxstudio:editor-preview-dock-hide", function (event) {
      var detail = event.detail || {};
      var form = editorPreviewForm(detail, event);
      setEditorPreviewResult(detail, hideEditorPreviewDocks(form));
    });
    doc.addEventListener("gosxstudio:editor-preview-dock-field-navigation-sync", function (event) {
      var detail = event.detail || {};
      setEditorPreviewResult(detail, syncEditorPreviewDockFieldNavigation(detail.dock, detail.count, detail.index));
    });
  }

  bindEditorPreviewChromeEvents();

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
