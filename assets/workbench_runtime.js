(function () {
  "use strict";

  function ready(fn) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", fn, { once: true });
      return;
    }
    fn();
  }

  function frameTask(fn) {
    var pending = false;
    return function () {
      if (pending) return;
      pending = true;
      window.requestAnimationFrame(function () {
        pending = false;
        fn();
      });
    };
  }

  function compactText(value) {
    return String(value || "").replace(/\s+/g, " ").trim();
  }

  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function number(value, fallback) {
    var parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : fallback;
  }

  function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function labelFromButton(button, fallback) {
    return compactText(button && (button.getAttribute("aria-label") || button.textContent)) || fallback || "";
  }

  function emit(form, name, detail) {
    form.dispatchEvent(new CustomEvent(name, { bubbles: true, detail: detail || {} }));
  }

  function queryAll(root, selector) {
    return Array.prototype.slice.call(root.querySelectorAll(selector));
  }

  function storageKey(form) {
    return "gosx-studio-workbench:" + compactText(form.getAttribute("data-studio-shell") || form.getAttribute("action") || window.location.pathname || "studio");
  }

  function railVar(side) {
    return side === "left" ? "--gosx-studio-left-rail-width" : "--gosx-studio-right-rail-width";
  }

  function legacyRailVar(side) {
    return side === "left" ? "--studio-left-width" : "--studio-right-width";
  }

  function railStyleValue(form, side) {
    return form.style.getPropertyValue(railVar(side)) || form.style.getPropertyValue(legacyRailVar(side));
  }

  function applyRailWidth(form, side, width) {
    var value = Math.round(width) + "px";
    form.style.setProperty(railVar(side), value);
    form.style.setProperty(legacyRailVar(side), value);
  }

  function workbenchFormFor(control) {
    return control && control.closest ? control.closest("form[data-studio-workbench], form[data-editor-workbench]") : null;
  }

  function shellRailState(form, side) {
    return form.getAttribute("data-studio-" + side) || "open";
  }

  function shellActivityState(form) {
    return form.getAttribute("data-studio-activity-state") || "open";
  }

  function syncShellButtons(form) {
    queryAll(form, "[data-studio-rail-toggle]").forEach(function (button) {
      var side = button.getAttribute("data-studio-rail-toggle");
      button.setAttribute("aria-pressed", shellRailState(form, side) === "open" ? "true" : "false");
    });
    queryAll(form, "[data-studio-focus-toggle]").forEach(function (button) {
      button.setAttribute("aria-pressed", form.getAttribute("data-studio-focus") === "true" ? "true" : "false");
    });
    var activityOpen = shellActivityState(form) === "open";
    queryAll(form, "[data-studio-activity-toggle]").forEach(function (button) {
      button.setAttribute("aria-pressed", activityOpen ? "true" : "false");
    });
  }

  function emitShellLayout(form, reason) {
    emit(form, "gosxstudio:workbench-layout", {
      left: shellRailState(form, "left"),
      right: shellRailState(form, "right"),
      focus: form.getAttribute("data-studio-focus") === "true",
      activity: shellActivityState(form),
      viewport: form.getAttribute("data-studio-breakpoint") || "",
      zoom: form.getAttribute("data-studio-zoom") || "",
      reason: reason || ""
    });
    writeLayout(form);
    window.requestAnimationFrame(function () {
      window.dispatchEvent(new Event("resize"));
    });
  }

  function claimShellControl(event) {
    if (!event.target || !event.target.closest) return false;
    var rail = event.target.closest("[data-studio-rail-toggle]");
    var focus = event.target.closest("[data-studio-focus-toggle]");
    var activity = event.target.closest("[data-studio-activity-toggle]");
    var control = rail || focus || activity;
    var form = workbenchFormFor(control);
    if (!form) return false;
    event.preventDefault();
    event.stopImmediatePropagation();
    if (rail) {
      var side = rail.getAttribute("data-studio-rail-toggle");
      if (side === "left" || side === "right") {
        form.setAttribute("data-studio-focus", "false");
        form.setAttribute("data-studio-" + side, shellRailState(form, side) === "open" ? "collapsed" : "open");
        syncShellButtons(form);
        emit(form, "gosxstudio:rail-change", { side: side, state: shellRailState(form, side), reason: "toggle" });
        emitShellLayout(form, "rail-toggle");
      }
      return true;
    }
    if (focus) {
      var enabled = form.getAttribute("data-studio-focus") !== "true";
      form.setAttribute("data-studio-focus", enabled ? "true" : "false");
      syncShellButtons(form);
      emit(form, "gosxstudio:focus-change", { focus: enabled, reason: "toggle" });
      emitShellLayout(form, "focus-toggle");
      return true;
    }
    form.setAttribute("data-studio-activity-state", shellActivityState(form) === "open" ? "collapsed" : "open");
    syncShellButtons(form);
    emit(form, "gosxstudio:activity-change", { state: shellActivityState(form), reason: "toggle" });
    emitShellLayout(form, "activity-toggle");
    return true;
  }

  function readLayout(form) {
    try {
      return JSON.parse(window.localStorage.getItem(storageKey(form)) || "{}") || {};
    } catch (error) {
      return {};
    }
  }

  function writeLayout(form) {
    try {
      window.localStorage.setItem(storageKey(form), JSON.stringify({
        left: railStyleValue(form, "left"),
        right: railStyleValue(form, "right"),
        focus: form.getAttribute("data-studio-focus") || "false",
        leftState: form.getAttribute("data-studio-left") || "open",
        rightState: form.getAttribute("data-studio-right") || "open",
        activity: form.getAttribute("data-studio-activity-state") || "open",
        viewport: form.getAttribute("data-studio-breakpoint") || "",
        zoom: form.getAttribute("data-studio-zoom") || ""
      }));
    } catch (error) {
      return;
    }
  }

  function restoreLayout(form) {
    var layout = readLayout(form);
    if (layout.left) {
      form.style.setProperty(railVar("left"), layout.left);
      form.style.setProperty(legacyRailVar("left"), layout.left);
    }
    if (layout.right) {
      form.style.setProperty(railVar("right"), layout.right);
      form.style.setProperty(legacyRailVar("right"), layout.right);
    }
    if (layout.leftState) form.setAttribute("data-studio-left", layout.leftState);
    if (layout.rightState) form.setAttribute("data-studio-right", layout.rightState);
    if (layout.activity) form.setAttribute("data-studio-activity-state", layout.activity);
    if (layout.focus) form.setAttribute("data-studio-focus", layout.focus === "true" ? "true" : "false");
    if (layout.viewport) form.setAttribute("data-studio-breakpoint", layout.viewport);
    if (layout.zoom) form.setAttribute("data-studio-zoom", layout.zoom);
  }

  function zoomScale(value) {
    value = String(value || "fit").toLowerCase();
    if (value === "fit") return 0;
    if (value.indexOf("%") > 0) return number(value.replace("%", ""), 100) / 100;
    var parsed = number(value, 0);
    if (parsed > 10) return parsed / 100;
    return parsed > 0 ? parsed : 1;
  }

  function initWorkbench(form) {
    if (!form || (form.dataset.gosxStudioWorkbenchBound === "true" && form.__gosxStudioWorkbenchRuntime)) return;
    form.dataset.gosxStudioWorkbenchBound = "true";
    form.__gosxStudioWorkbenchRuntime = { version: 1 };
    var stage = form.querySelector("[data-studio-layout]");
    var saveLayout = frameTask(function () { writeLayout(form); });
    var operationCounter = 0;
    var refresh = frameTask(function () {
      emit(form, "gosxstudio:workbench-layout", {
        left: form.getAttribute("data-studio-left") || "open",
        right: form.getAttribute("data-studio-right") || "open",
        focus: form.getAttribute("data-studio-focus") === "true",
        activity: form.getAttribute("data-studio-activity-state") || "open",
        viewport: form.getAttribute("data-studio-breakpoint") || "",
        zoom: form.getAttribute("data-studio-zoom") || ""
      });
      window.dispatchEvent(new Event("resize"));
    });

    function setReadout(selector, value) {
      queryAll(form, selector).forEach(function (node) {
        node.textContent = value;
      });
    }

    function emitEditorOperation(kind, detail) {
      detail = detail || {};
      operationCounter += 1;
      var operation = {
        id: "studio-op-" + Date.now() + "-" + operationCounter,
        kind: kind || "operation",
        source: "gosx-studio",
        reason: detail.reason || "",
        mutation: detail.mutation !== false,
        target: detail.target || {},
        payload: detail.payload || {}
      };
      emit(form, "gosxstudio:editor-operation", operation);
      return operation;
    }

    function previewShells() {
      return queryAll(form, "[data-gosx-studio-preview], [data-studio-preview-shell], [data-studio-canvas]");
    }

    function previewFrames() {
      return queryAll(form, "[data-studio-preview-frame]");
    }

    function previewSelectionDetail(node) {
      var payload = { target: node };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-detail-resolve", { bubbles: true, detail: payload }));
      if (payload.result) return payload.result;
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        payload = { target: node };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-detail-resolve", { bubbles: true, detail: payload }));
        if (payload.result) return payload.result;
      }
      return {};
    }

    function clearPreviewSelections() {
      var detail = {
        form: form,
        host: {
          finishInlineTextEdit: finishInlineTextEdit
        }
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selection-clear-sync", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function updatePreviewDockPosition(frame) {
      var dock = frame && frame.__gosxStudioPreviewDock;
      var target = frame && frame.__gosxStudioPreviewDockTarget;
      if (!dock || !target || dock.hidden) return null;
      var detail = { form: form, frame: frame, dock: dock, target: target };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-position-sync", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function previewDockDetail(dock) {
      var detail = { form: form, dock: dock };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-detail-resolve", { bubbles: true, detail: detail }));
      return detail.result || {};
    }

    function emitPreviewDockAction(action, detail) {
      emit(form, "gosxstudio:preview-action", {
        action: action || "",
        detail: detail || {},
        reason: "preview-dock"
      });
    }

    function dispatchPreviewFieldActionResolve(detail) {
      var payload = {
        detail: detail || {}
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-action-resolve", { bubbles: true, detail: payload }));
      if (payload.result) return payload.result;
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        payload = {
          detail: detail || {}
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-action-resolve", { bubbles: true, detail: payload }));
        if (payload.result) return payload.result;
      }
      return null;
    }

    function dispatchPreviewDockActionResolve(action, detail) {
      var payload = {
        action: action || "",
        detail: detail || {}
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-dock-action-resolve", { bubbles: true, detail: payload }));
      if (payload.result) return payload.result;
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        payload = {
          action: action || "",
          detail: detail || {}
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-dock-action-resolve", { bubbles: true, detail: payload }));
        if (payload.result) return payload.result;
      }
      return null;
    }

    function bindSelectionRuntime() {
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        return true;
      }
      return false;
    }

    function previewFieldTarget(detailOrField) {
      var detail = typeof detailOrField === "string" ? { field: detailOrField } : (detailOrField || {});
      var payload = {
        detail: detail
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-target-resolve", { bubbles: true, detail: payload }));
      if (payload.result) return payload.result;
      if (bindSelectionRuntime()) {
        payload = {
          detail: detail
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-target-resolve", { bubbles: true, detail: payload }));
        if (payload.result) return payload.result;
      }
      return { field: detail.field || "", source: null, control: null, found: false };
    }

    function revealPreviewField(detail, reason) {
      detail = detail || {};
      if (!detail.field) return false;
      if (form.querySelector('[data-studio-mode-control="content"], [data-studio-mode-panel="content"]')) {
        setMode("content", { reason: reason || "preview-select" });
      }
      var payload = {
        detail: detail,
        reason: reason || ""
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-reveal", { bubbles: true, detail: payload }));
      if (payload.result) return !!payload.result.revealed;
      if (bindSelectionRuntime()) {
        payload = {
          detail: detail,
          reason: reason || ""
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-reveal", { bubbles: true, detail: payload }));
        if (payload.result) return !!payload.result.revealed;
      }
      return false;
    }

    function submitPreviewFieldAction(detail) {
      var payload = {
        detail: detail || {}
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-action-submit", { bubbles: true, detail: payload }));
      if (payload.result) return !!payload.result;
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        payload = {
          detail: detail || {}
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-action-submit", { bubbles: true, detail: payload }));
        if (payload.result) return !!payload.result;
      }
      return false;
    }

    function navigatePreviewField(frame, direction, reason) {
      var payload = {
        form: form,
        frame: frame,
        direction: direction,
        reason: reason || "field-navigation",
        host: {
          finishInlineTextEdit: finishInlineTextEdit,
          previewSelectionDetail: previewSelectionDetail,
          applyPreviewSelection: applyPreviewSelection,
          dispatchPreviewFieldNavigation: dispatchPreviewFieldNavigation
        }
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-field-navigation-run", { bubbles: true, detail: payload }));
      return !!(payload.result && payload.result.navigated);
    }

    function dispatchPreviewFieldNavigation(detail, navigation) {
      var payload = {
        detail: detail || {},
        navigation: navigation || {}
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-navigation-commit", { bubbles: true, detail: payload }));
      if (payload.result) return payload.result;
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        payload = {
          detail: detail || {},
          navigation: navigation || {}
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-navigation-commit", { bubbles: true, detail: payload }));
        if (payload.result) return payload.result;
      }
      return null;
    }

    function runPreviewDockAction(frame, action) {
      var dock = frame && frame.__gosxStudioPreviewDock;
      if (!dock || !action) return false;
      var detail = previewDockDetail(dock);
      if (action === "clear") {
        clearPreviewSelections();
        dispatchPreviewSelectionClear("preview-dock");
        emitPreviewDockAction(action, detail);
        return true;
      }
      if (action === "content" || action === "style") {
        var dockIntent = dispatchPreviewDockActionResolve(action, detail) || { mode: action, reveal: action === "content" && !!detail.field };
        if (dockIntent.mode) setMode(dockIntent.mode, { scroll: true, reason: "preview-dock" });
        if (dockIntent.reveal) {
          revealPreviewField(detail, "preview-dock");
        }
        emitPreviewDockAction(action, detail);
        return true;
      }
      if (action === "prev-field" || action === "next-field") {
        if (navigatePreviewField(frame, action === "next-field" ? 1 : -1, "preview-dock")) {
          emitPreviewDockAction(action, previewDockDetail(dock));
          return true;
        }
        emitPreviewDockAction(action, detail);
        return false;
      }
      if (action === "field-action") {
        var intent = dispatchPreviewFieldActionResolve(detail) || { reveal: !!detail.field };
        if (intent.inlineText && startInlineTextFromDetail(frame, detail, "preview-dock")) return true;
        if (intent.submit && submitPreviewFieldAction(detail)) {
          emitPreviewDockAction(action, detail);
          return true;
        }
        if (intent.navigate) {
          window.location.href = intent.href || "";
        } else if (intent.reveal) {
          revealPreviewField(detail, "preview-dock");
        }
        emitPreviewDockAction(action, detail);
        return true;
      }
      emitPreviewDockAction(action, detail);
      return true;
    }

    function dispatchPreviewSelectionApply(detail, options) {
      var payload = {
        detail: detail || {},
        options: options || {}
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-apply", { bubbles: true, detail: payload }));
      if (payload.result) return payload.result;
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        payload = {
          detail: detail || {},
          options: options || {}
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-apply", { bubbles: true, detail: payload }));
        if (payload.result) return payload.result;
      }
      return null;
    }

    function dispatchPreviewSelectionClear(reason) {
      var payload = {
        reason: reason || ""
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-clear", { bubbles: true, detail: payload }));
      if (payload.result) return true;
      var runtime = window.GoSXStudioSelectionRuntime;
      if (runtime && typeof runtime.bind === "function") {
        runtime.bind(document.body || document);
        payload = {
          reason: reason || ""
        };
        form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-clear", { bubbles: true, detail: payload }));
        if (payload.result) return true;
      }
      return false;
    }

    function applyPreviewSelection(frame, target, detail, options) {
      detail = detail || previewSelectionDetail(target);
      options = options || {};
      if (!detail.field && !detail.blockKey && !detail.nodeID) return false;
      var payload = {
        form: form,
        frame: frame,
        target: target,
        selection: detail,
        detail: detail,
        options: options,
        host: {
          finishInlineTextEdit: finishInlineTextEdit,
          dispatchPreviewSelectionApply: dispatchPreviewSelectionApply,
          runPreviewDockAction: runPreviewDockAction
        }
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selection-apply-sync", { bubbles: true, detail: payload }));
      if (!payload.result || !payload.result.handled || !payload.result.applied) return false;
      if (options.reveal) revealPreviewField(detail, options.reason || "preview-select");
      return true;
    }

    function syncPreviewFrame(frame, reason) {
      var detail = { form: form, frame: frame, reason: reason || "sync", shouldTransport: shouldTransportPreviewPatch };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-frame-sync", { bubbles: true, detail: detail }));
      return detail.result && detail.result.count ? detail.result.count : 0;
    }

    function setPreviewStatus(state, label, reason) {
      var detail = { form: form, state: state, label: label, reason: reason || "" };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-status-set", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function syncPreviewRoute(route, reason) {
      var detail = { form: form, route: route || "", reason: reason || "" };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-route-sync", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function refreshPreviewNow(reason, route) {
      var detail = { form: form, route: route || "", reason: reason || "refresh" };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-refresh-now", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function schedulePreviewRefresh(reason, route) {
      var detail = { form: form, route: route || "", reason: reason || "refresh" };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-refresh-schedule", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function resolvePreviewPatchTransport(field, reason) {
      var envelope = {
        field: field || null,
        reason: reason || "patch",
        result: null
      };
      var target = field && field.dispatchEvent ? field : form;
      target.dispatchEvent(new CustomEvent("gosxstudio:field-preview-patch-resolve", {
        bubbles: true,
        detail: envelope
      }));
      return envelope.result || null;
    }

    function bindFieldRuntimeForPreviewPatchResolution() {
      var runtime = window.GoSXStudioFieldRuntime;
      var root = document.body || document;
      if (!runtime) return false;
      try {
        if (typeof runtime.bindMirroring === "function") {
          runtime.bindMirroring(root);
          return true;
        }
        if (typeof runtime.bind === "function") {
          runtime.bind(root);
          return true;
        }
      } catch (error) {
        return false;
      }
      return false;
    }

    function resolveFieldOperationEnvelope(reason, field) {
      var payload = {
        form: form,
        reason: reason || "field",
        field: field || null,
        result: null
      };
      var target = field && field.dispatchEvent ? field : form;
      target.dispatchEvent(new CustomEvent("gosxstudio:field-operation-build", {
        bubbles: true,
        detail: payload
      }));
      return payload.result || null;
    }

    function fieldOperationEnvelope(reason, field) {
      var envelope = resolveFieldOperationEnvelope(reason, field);
      if (!envelope && bindFieldRuntimeForPreviewPatchResolution()) {
        envelope = resolveFieldOperationEnvelope(reason, field);
      }
      return envelope || null;
    }

    function shouldTransportPreviewPatch(field, reason) {
      if (!field) return true;
      var result = resolvePreviewPatchTransport(field, reason);
      if (!result && bindFieldRuntimeForPreviewPatchResolution()) {
        result = resolvePreviewPatchTransport(field, reason);
      }
      return !(result && result.transport === false);
    }

    function emitFieldOperation(reason, field) {
      var envelope = fieldOperationEnvelope(reason, field);
      if (!envelope) return null;
      return emitEditorOperation("set_field", envelope);
    }

    function textControlForField(field) {
      var target = previewFieldTarget(field);
      if (target.control && "value" in target.control) return target.control;
      if (target.source && "value" in target.source) return target.source;
      return null;
    }

    function previewInlineTextRuntime(method) {
      var runtime = window.GoSXStudioInlineEditRuntime;
      return runtime && typeof runtime[method] === "function" ? runtime : null;
    }

    function emitPreviewInlineTextEvent(name, detail) {
      emit(form, name, detail);
    }

    function previewInlineTextHost(frame) {
      return {
        form: form,
        target: frame && frame.__gosxStudioPreviewDockTarget,
        controlForField: textControlForField,
        setStatus: setPreviewStatus,
        emitEditorOperation: emitEditorOperation,
        emitEvent: emitPreviewInlineTextEvent,
        updateDock: updatePreviewDockPosition
      };
    }

    function syncInlineTextEdit(frame, reason) {
      var runtime = previewInlineTextRuntime("syncPreviewTextSession");
      if (!runtime) return false;
      return runtime.syncPreviewTextSession(frame, reason, previewInlineTextHost(frame));
    }

    function startInlineTextEdit(frame, detail, reason) {
      var runtime = previewInlineTextRuntime("startPreviewTextSession");
      if (!runtime) return false;
      detail = detail || (frame && frame.__gosxStudioPreviewDock ? previewDockDetail(frame.__gosxStudioPreviewDock) : {});
      return runtime.startPreviewTextSession(frame, detail, reason || "preview-dock", previewInlineTextHost(frame));
    }

    function startInlineTextFromDetail(frame, detail, reason) {
      if (!detail || detail.editable !== "text") return false;
      if (!startInlineTextEdit(frame, detail, reason || "preview-dock")) return false;
      emitPreviewDockAction("field-action", detail);
      return true;
    }

    function startInlineTextFromSelection(frame, reason) {
      var dock = frame && frame.__gosxStudioPreviewDock;
      if (!dock || dock.hidden) return false;
      return startInlineTextFromDetail(frame, previewDockDetail(dock), reason || "keyboard");
    }

    function finishInlineTextEdit(frame, commit, reason) {
      var runtime = previewInlineTextRuntime("finishPreviewTextSession");
      if (!runtime) return false;
      return runtime.finishPreviewTextSession(frame, commit, reason, previewInlineTextHost(frame));
    }

    function handleInlineTextInput(frame, event) {
      var runtime = previewInlineTextRuntime("handlePreviewTextInputEvent");
      if (!runtime) return false;
      return runtime.handlePreviewTextInputEvent(frame, event, previewInlineTextHost(frame));
    }

    function handleInlineTextKeyEvent(frame, event) {
      var runtime = previewInlineTextRuntime("handlePreviewTextKeyEvent");
      if (!runtime) return false;
      return runtime.handlePreviewTextKeyEvent(frame, event, previewInlineTextHost(frame));
    }

    function handleInlineTextPaste(frame, event) {
      var runtime = previewInlineTextRuntime("handlePreviewTextPasteEvent");
      if (!runtime) return false;
      return runtime.handlePreviewTextPasteEvent(frame, event, previewInlineTextHost(frame));
    }

    function handleInlineTextBlur(frame, event) {
      var runtime = previewInlineTextRuntime("handlePreviewTextBlurEvent");
      if (!runtime) return false;
      return runtime.handlePreviewTextBlurEvent(frame, event, previewInlineTextHost(frame));
    }

    function postPreviewPatch(reason, detail, field) {
      var frames = previewFrames();
      if (!frames.length) return;
      var payload = { form: form, reason: reason || "patch", detail: detail || {}, field: field || null };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-patch-build-post", { bubbles: true, detail: payload }));
      return payload.result || null;
    }

    function bindPreviewFrames() {
      var detail = {
        form: form,
        host: {
          setPreviewStatus: setPreviewStatus,
          syncPreviewFrame: syncPreviewFrame,
          bindPreviewDocument: bindPreviewDocument,
          postPreviewLoadSyncPatch: function (route) {
            return postPreviewPatch("load-sync", { route: route || "" }, null);
          }
        }
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-frames-bind", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function bindPreviewDocument(frame) {
      var detail = {
        form: form,
        frame: frame,
        host: {
          previewSelectionDetail: previewSelectionDetail,
          applyPreviewSelection: applyPreviewSelection,
          finishInlineTextEdit: finishInlineTextEdit,
          dispatchPreviewFieldNavigation: dispatchPreviewFieldNavigation,
          startInlineTextFromDetail: startInlineTextFromDetail,
          startInlineTextFromSelection: startInlineTextFromSelection,
          handleInlineTextInput: handleInlineTextInput,
          handleInlineTextKeyEvent: handleInlineTextKeyEvent,
          handleInlineTextPaste: handleInlineTextPaste,
          handleInlineTextBlur: handleInlineTextBlur,
          updatePreviewDockPosition: updatePreviewDockPosition
        }
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-document-bind", { bubbles: true, detail: detail }));
      return detail.result || null;
    }

    function modeLabel(mode) {
      var button = form.querySelector('[data-studio-mode-control="' + attrValue(mode) + '"]');
      return labelFromButton(button, mode ? mode.charAt(0).toUpperCase() + mode.slice(1) : "Structure");
    }

    function viewportButton(viewport) {
      return form.querySelector('[data-studio-viewport="' + attrValue(viewport) + '"]');
    }

    function viewportLabel(viewport) {
      return labelFromButton(viewportButton(viewport), viewport ? viewport.charAt(0).toUpperCase() + viewport.slice(1) : "Desktop");
    }

    function setMode(mode, options) {
      options = options || {};
      mode = mode || "structure";
      form.setAttribute("data-studio-mode", mode);
      queryAll(form, "[data-studio-mode-control]").forEach(function (button) {
        button.setAttribute("aria-pressed", button.getAttribute("data-studio-mode-control") === mode ? "true" : "false");
      });
      queryAll(form, "[data-studio-mode-panel]").forEach(function (panel) {
        panel.classList.toggle("is-mode-active", panel.getAttribute("data-studio-mode-panel") === mode);
      });
      setReadout("[data-studio-mode-label]", modeLabel(mode));
      if (options.scroll) {
        var panel = form.querySelector('[data-studio-mode-panel="' + attrValue(mode) + '"]');
        if (panel && panel.scrollIntoView) panel.scrollIntoView({ block: "nearest", behavior: "smooth" });
      }
      emit(form, "gosxstudio:mode-change", { mode: mode, label: modeLabel(mode), reason: options.reason || "" });
      saveLayout();
    }

    function setViewport(viewport, options) {
      options = options || {};
      viewport = viewport || "desktop";
      var button = viewportButton(viewport);
      var width = button ? button.getAttribute("data-studio-viewport-width") || "" : "";
      form.setAttribute("data-studio-breakpoint", viewport);
      queryAll(form, "[data-studio-viewport]").forEach(function (candidate) {
        candidate.setAttribute("aria-pressed", candidate.getAttribute("data-studio-viewport") === viewport ? "true" : "false");
      });
      setReadout("[data-studio-viewport-label]", viewportLabel(viewport));
      queryAll(form, "[data-gosx-studio-preview], [data-studio-preview-frame]").forEach(function (node) {
        node.setAttribute("data-studio-preview-viewport", viewport);
        if (node.matches("[data-studio-preview-frame]") && width) {
          node.style.width = width;
          node.style.maxWidth = "100%";
        }
      });
      emit(form, "gosxstudio:viewport-change", {
        viewport: viewport,
        label: viewportLabel(viewport),
        width: width,
        reason: options.reason || ""
      });
      saveLayout();
      refresh();
    }

    function setZoom(zoom, options) {
      options = options || {};
      zoom = zoom || "fit";
      var scale = zoomScale(zoom);
      form.setAttribute("data-studio-zoom", zoom);
      queryAll(form, "button[data-studio-zoom], [role='button'][data-studio-zoom]").forEach(function (button) {
        button.setAttribute("aria-pressed", button.getAttribute("data-studio-zoom") === zoom ? "true" : "false");
      });
      queryAll(form, "[data-studio-canvas], [data-gosx-studio-preview]").forEach(function (node) {
        node.setAttribute("data-studio-canvas-zoom", zoom);
        node.style.setProperty("--gosx-studio-preview-zoom", scale ? String(scale) : "1");
      });
      document.dispatchEvent(new CustomEvent("gosxstudio:workbench-zoom", {
        bubbles: true,
        detail: { zoom: zoom, scale: scale, reason: options.reason || "", form: form }
      }));
      emit(form, "gosxstudio:zoom-change", { zoom: zoom, scale: scale, reason: options.reason || "" });
      saveLayout();
      refresh();
    }

    function railState(side) {
      return form.getAttribute("data-studio-" + side) || "open";
    }

    function syncRailButtons() {
      queryAll(form, "[data-studio-rail-toggle]").forEach(function (button) {
        var side = button.getAttribute("data-studio-rail-toggle");
        button.setAttribute("aria-pressed", railState(side) === "open" ? "true" : "false");
      });
      queryAll(form, "[data-studio-focus-toggle]").forEach(function (button) {
        button.setAttribute("aria-pressed", form.getAttribute("data-studio-focus") === "true" ? "true" : "false");
      });
    }

    function setRail(side, state, options) {
      options = options || {};
      if (side !== "left" && side !== "right") return;
      form.setAttribute("data-studio-focus", "false");
      form.setAttribute("data-studio-" + side, state === "collapsed" ? "collapsed" : "open");
      syncRailButtons();
      emit(form, "gosxstudio:rail-change", { side: side, state: railState(side), reason: options.reason || "" });
      saveLayout();
      refresh();
    }

    function toggleRail(side) {
      setRail(side, railState(side) === "open" ? "collapsed" : "open", { reason: "toggle" });
    }

    function setFocus(enabled, reason) {
      form.setAttribute("data-studio-focus", enabled ? "true" : "false");
      syncRailButtons();
      emit(form, "gosxstudio:focus-change", { focus: enabled, reason: reason || "" });
      saveLayout();
      refresh();
    }

    function activityState() {
      return form.getAttribute("data-studio-activity-state") || "open";
    }

    function syncActivityButtons() {
      var open = activityState() === "open";
      queryAll(form, "[data-studio-activity-toggle]").forEach(function (button) {
        button.setAttribute("aria-pressed", open ? "true" : "false");
      });
    }

    function setActivity(state, reason) {
      form.setAttribute("data-studio-activity-state", state === "collapsed" ? "collapsed" : "open");
      syncActivityButtons();
      emit(form, "gosxstudio:activity-change", { state: activityState(), reason: reason || "" });
      saveLayout();
      refresh();
    }

    function currentRailWidth(side) {
      var custom = railStyleValue(form, side);
      var parsed = parseInt(custom, 10);
      if (Number.isFinite(parsed)) return parsed;
      var node = form.querySelector(side === "left" ? "[data-studio-sidebar='left']" : "[data-studio-sidebar='right']");
      return node ? Math.round(node.getBoundingClientRect().width) : (side === "left" ? 320 : 416);
    }

    function resizerHandle(side) {
      return form.querySelector('[data-studio-resizer="' + attrValue(side) + '"]');
    }

    function railLimit(side, name, fallback) {
      var handle = resizerHandle(side);
      return number(handle && handle.getAttribute(name), fallback);
    }

    function updateResizerValue(side, width) {
      var handle = resizerHandle(side);
      if (handle) handle.setAttribute("aria-valuenow", String(Math.round(width)));
    }

    function emitRailResize(name, side, width, reason) {
      emit(form, name, { side: side, width: Math.round(width), reason: reason || "" });
      if (name === "gosxstudio:workbench-rail-resize") {
        emit(form, "gosxstudio:rail-resize", { side: side, width: Math.round(width), reason: reason || "" });
      }
    }

    function setRailWidth(side, width, reason) {
      if (side !== "left" && side !== "right") return;
      var min = railLimit(side, "data-studio-rail-min", side === "left" ? 256 : 320);
      var max = railLimit(side, "data-studio-rail-max", side === "left" ? 448 : 544);
      var next = clamp(Math.round(width), min, max);
      applyRailWidth(form, side, next);
      updateResizerValue(side, next);
      emitRailResize("gosxstudio:workbench-rail-resize", side, next, reason);
      saveLayout();
      refresh();
    }

    function finishRailResize(side, reason) {
      var width = currentRailWidth(side);
      updateResizerValue(side, width);
      emitRailResize("gosxstudio:workbench-rail-resized", side, width, reason);
    }

    function bindResizers() {
      if (!stage) return;
      queryAll(form, "[data-studio-resizer]").forEach(function (handle) {
        if (handle.dataset.gosxStudioResizerBound === "true") return;
        handle.dataset.gosxStudioResizerBound = "true";
        handle.addEventListener("pointerdown", function (event) {
          if (event.button !== 0) return;
          event.preventDefault();
          var side = handle.getAttribute("data-studio-resizer");
          var rect = stage.getBoundingClientRect();
          handle.classList.add("is-resizing");
          if (handle.setPointerCapture) handle.setPointerCapture(event.pointerId);
          function move(pointerEvent) {
            setRailWidth(side, side === "left" ? pointerEvent.clientX - rect.left : rect.right - pointerEvent.clientX, "drag");
          }
          function finish() {
            handle.classList.remove("is-resizing");
            finishRailResize(handle.getAttribute("data-studio-resizer"), "drag");
            writeLayout(form);
            document.removeEventListener("pointermove", move);
            document.removeEventListener("pointerup", finish);
            document.removeEventListener("pointercancel", finish);
          }
          move(event);
          document.addEventListener("pointermove", move);
          document.addEventListener("pointerup", finish);
          document.addEventListener("pointercancel", finish);
        });
        handle.addEventListener("keydown", function (event) {
          if (event.key !== "ArrowLeft" && event.key !== "ArrowRight" && event.key !== "Home" && event.key !== "End") return;
          event.preventDefault();
          var side = handle.getAttribute("data-studio-resizer");
          if (event.key === "Home") {
            setRailWidth(side, railLimit(side, "data-studio-rail-min", side === "left" ? 256 : 320), "keyboard");
            finishRailResize(side, "keyboard");
            return;
          }
          if (event.key === "End") {
            setRailWidth(side, railLimit(side, "data-studio-rail-max", side === "left" ? 448 : 544), "keyboard");
            finishRailResize(side, "keyboard");
            return;
          }
          var step = event.shiftKey ? 48 : 24;
          var delta = event.key === "ArrowRight" ? step : -step;
          setRailWidth(side, currentRailWidth(side) + (side === "left" ? delta : -delta), "keyboard");
          finishRailResize(side, "keyboard");
        });
      });
    }

    function handleCommand(detail) {
      detail = detail || {};
      if (detail.kind === "mode") {
        setMode(detail.target, { scroll: true, reason: "command" });
        return true;
      }
      if (detail.kind === "viewport") {
        setViewport(detail.target, { reason: "command" });
        return true;
      }
      if (detail.kind === "zoom") {
        setZoom(detail.target, { reason: "command" });
        return true;
      }
      if (detail.kind === "toggle") {
        if (detail.target === "left" || detail.target === "right") toggleRail(detail.target);
        else if (detail.target === "activity") setActivity(activityState() === "open" ? "collapsed" : "open", "command");
        else if (detail.target === "focus") setFocus(form.getAttribute("data-studio-focus") !== "true", "command");
        else return false;
        return true;
      }
      return false;
    }

    form.addEventListener("gosxstudio:command", function (event) {
      if (handleCommand(event.detail || {})) event.preventDefault();
    });

    form.addEventListener("click", function (event) {
      var mode = event.target.closest("[data-studio-mode-control]");
      if (mode && form.contains(mode)) {
        event.preventDefault();
        setMode(mode.getAttribute("data-studio-mode-control"), { scroll: true, reason: "click" });
        return;
      }
      var viewport = event.target.closest("[data-studio-viewport]");
      if (viewport && form.contains(viewport)) {
        event.preventDefault();
        setViewport(viewport.getAttribute("data-studio-viewport"), { reason: "click" });
        return;
      }
      var zoom = event.target.closest("button[data-studio-zoom], [role='button'][data-studio-zoom]");
      if (zoom && form.contains(zoom)) {
        event.preventDefault();
        setZoom(zoom.getAttribute("data-studio-zoom"), { reason: "click" });
        return;
      }
      var rail = event.target.closest("[data-studio-rail-toggle]");
      if (rail && form.contains(rail)) {
        event.preventDefault();
        event.stopImmediatePropagation();
        toggleRail(rail.getAttribute("data-studio-rail-toggle"));
        return;
      }
      var focus = event.target.closest("[data-studio-focus-toggle]");
      if (focus && form.contains(focus)) {
        event.preventDefault();
        event.stopImmediatePropagation();
        setFocus(form.getAttribute("data-studio-focus") !== "true", "toggle");
        return;
      }
      var activity = event.target.closest("[data-studio-activity-toggle]");
      if (activity && form.contains(activity)) {
        event.preventDefault();
        event.stopImmediatePropagation();
        setActivity(activityState() === "open" ? "collapsed" : "open", "toggle");
        return;
      }
    });

    form.addEventListener("gosxstudio:canvas-select", function (event) {
      var detail = event.detail || {};
      if (detail.key) form.setAttribute("data-studio-selection", detail.key);
      else form.removeAttribute("data-studio-selection");
      form.setAttribute("data-studio-selection-kind", detail.kind || "");
      setReadout("[data-studio-selection-label]", detail.label || "No selection");
      setReadout("[data-studio-selection-status]", detail.key ? (detail.kind || "Selected") : "No selection");
    });

    form.addEventListener("input", function (event) {
      if (!event.target || !form.contains(event.target)) return;
      emitFieldOperation("input", event.target);
      if (shouldTransportPreviewPatch(event.target, "input")) postPreviewPatch("input", {}, event.target);
    });

    form.addEventListener("change", function (event) {
      if (!event.target || !form.contains(event.target)) return;
      emitFieldOperation("change", event.target);
      if (shouldTransportPreviewPatch(event.target, "change")) postPreviewPatch("change", {}, event.target);
    });

    form.addEventListener("gosxstudio:editor-transaction", function (event) {
      postPreviewPatch("transaction", event.detail || {}, null);
    });

    form.addEventListener("gosxstudio:history-restore", function (event) {
      postPreviewPatch("history-restore", event.detail || {}, null);
    });

    form.addEventListener("gosxstudio:save-state", function (event) {
      var detail = event.detail || {};
      if (detail.state === "dirty") setPreviewStatus("dirty", "Draft changed", detail.reason || "dirty");
      else if (detail.state === "autosaving" || detail.state === "saving") setPreviewStatus("syncing", "Syncing preview", detail.reason || "saving");
      else if (detail.state === "saved") schedulePreviewRefresh(detail.reason || "saved", "");
      else if (detail.state === "error") setPreviewStatus("error", "Preview waiting on save", detail.reason || "error");
    });

    form.addEventListener("gosxstudio:action-result", function (event) {
      var detail = event.detail || {};
      if (detail.ok) schedulePreviewRefresh("action", "");
      else setPreviewStatus("error", "Preview waiting on action", "action-error");
    });

    form.addEventListener("gosxstudio:flow-preview", function (event) {
      var detail = event.detail || {};
      if (detail.route) {
        syncPreviewRoute(detail.route, "flow");
        schedulePreviewRefresh("flow", detail.route);
      }
    });

    document.addEventListener("keydown", function (event) {
      if (!document.contains(form) || event.defaultPrevented) return;
      if ((event.metaKey || event.ctrlKey) && event.key === "\\") {
        event.preventDefault();
        setFocus(form.getAttribute("data-studio-focus") !== "true", "keyboard");
      } else if (event.key === "Escape" && form.getAttribute("data-studio-focus") === "true") {
        setFocus(false, "keyboard");
      } else if (event.key === "Escape" && form.querySelector("[data-gosx-studio-preview-dock]:not([hidden])")) {
        clearPreviewSelections();
        dispatchPreviewSelectionClear("keyboard-escape");
      }
    });

    window.addEventListener("resize", function () {
      previewFrames().forEach(updatePreviewDockPosition);
    });

    restoreLayout(form);
    bindResizers();
    bindPreviewFrames();
    updateResizerValue("left", currentRailWidth("left"));
    updateResizerValue("right", currentRailWidth("right"));
    setMode(form.getAttribute("data-studio-mode") || "", { reason: "init" });
    setViewport(form.getAttribute("data-studio-breakpoint") || "", { reason: "init" });
    setZoom(form.getAttribute("data-studio-zoom") || "", { reason: "init" });
    syncRailButtons();
    syncActivityButtons();
  }

  function initAll(root) {
    var scope = root && root.querySelectorAll ? root : document;
    queryAll(scope, "form[data-studio-workbench], form[data-editor-workbench]").forEach(initWorkbench);
  }

  document.addEventListener("click", claimShellControl, true);
  ready(function () { initAll(document); });
  document.addEventListener("gosx:navigate", function () { initAll(document); });
  document.addEventListener("gosx:render", function () { initAll(document); });
})();
