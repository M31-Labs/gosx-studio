// island_runtime.js — companion JS for the workbenchruntime .gsx islands.
//
// The .gsx islands (workbench_chrome.gsx, workbench_rails.gsx,
// workbench_viewport.gsx, workbench_zoom.gsx, workbench_toggles.gsx,
// workbench_style_state.gsx) are mount-point markers in the editor DOM.
// This script publishes the fifteen
// window.__gosx_workbench_runtime_island_<method> globals that
// workbenchruntime.BridgeShim delegates to directly. It is emitted into
// the studio runtime bundle by workbenchruntime.IslandRuntimeJS() (see
// runtime.go) and runs before BridgeShim() at bundle init time.
//
// The fifteen functions below replace window.GoSXStudioWorkbenchRuntime.{
// bindRailResizers, bindChrome, setMode, syncViewport, activateViewport,
// currentBreakpoint, setStyleState, syncZoom, activateZoom, toggleRail,
// toggleFocus, toggleActivity, saveLayout, currentRailWidth, setRailWidth }
// while preserving exact observable behavior of the legacy implementations
// (bindWorkbenchRailResizers / bindWorkbenchChrome / setWorkbenchMode /
// syncWorkbenchViewport / activateWorkbenchViewport /
// currentWorkbenchBreakpoint / setWorkbenchStyleState / syncWorkbenchZoom /
// activateWorkbenchZoom / toggleWorkbenchRail / toggleWorkbenchFocus /
// toggleWorkbenchActivity / saveWorkbenchLayout / currentWorkbenchRailWidth /
// setWorkbenchRailWidth at assets/studio-engines.js:165–586).
//
// # Idempotency
//
// Idempotency guards use dataset keys scoped to the island implementation
// so repeated host calls do not double-bind:
//
//   gosxStudioResizerIslandBound        — per resizer handle (bindRailResizers)
//   gosxStudioWorkbenchChromeIslandBound — per editor-workbench form (bindChrome)
//   gosxStudioWorkbenchCommandsIslandBound — per command-palette form (bindChrome)
//
// The other twelve methods are pure DOM mutations / reads — they do not
// attach event listeners and require no idempotency guards.

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  // ===== Shared helpers (mirror the legacy private helpers in studio-engines.js) =====

  // attrValue — escapes backslashes and double quotes so an arbitrary string
  // can be embedded inside a CSS attribute selector double-quoted value.
  // Mirrors the legacy attrValue helper at studio-engines.js:2158.
  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  // frame — schedule a callback at next animation frame (mirrors
  // studio-engines.js:75 helper).
  function frame(callback) {
    if (typeof window.requestAnimationFrame === "function") {
      return { id: window.requestAnimationFrame(callback), raf: true };
    }
    return { id: window.setTimeout(callback, 16), raf: false };
  }

  // frameTask — coalesces calls into a single rAF dispatch with the last
  // arguments (mirrors studio-engines.js:83 helper). Used by saveLayoutSoon.
  function frameTask(callback) {
    var queued = false;
    var lastArgs = null;
    var lastThis = null;
    var active = true;
    var pendingFrame = null;
    var task = function () {
      if (!active) return;
      lastArgs = arguments;
      lastThis = this;
      if (queued) return;
      queued = true;
      pendingFrame = frame(function () {
        pendingFrame = null;
        queued = false;
        if (!active) return;
        callback.apply(lastThis, lastArgs || []);
      });
    };
    task.cancel = function () {
      active = false;
      if (pendingFrame) {
        if (pendingFrame.raf && typeof window.cancelAnimationFrame === "function") {
          window.cancelAnimationFrame(pendingFrame.id);
        } else if (!pendingFrame.raf && typeof window.clearTimeout === "function") {
          window.clearTimeout(pendingFrame.id);
        }
        pendingFrame = null;
      }
      queued = false;
      lastArgs = null;
      lastThis = null;
    };
    return task;
  }

  // editorWorkbench — resolve the [data-editor-workbench] form (or root if
  // it matches). Mirrors studio-engines.js:69 helper. Used by bindChrome /
  // bindRailResizers.
  function editorWorkbench(root) {
    if (root && root.matches && root.matches("[data-editor-workbench]")) return root;
    return doc.querySelector("[data-editor-workbench]");
  }

  function workbenchContains(node) {
    return !doc.contains || doc.contains(node);
  }

  // Each workbench form can be serviced by both public bind methods. Keep one
  // lifecycle observer per form and let each method register its own cleanup;
  // a fragment replacement then disposes every controller exactly once.
  var workbenchLifecycleProperty = "__gosxStudioWorkbenchLifecycle";

  function activeRailGestureFor(form) {
    var lifecycle = form && form[workbenchLifecycleProperty];
    var gesture = lifecycle && lifecycle.activeRailGesture;
    if (!lifecycle || lifecycle.disposed || !gesture || gesture.done || gesture.form !== form) return null;
    return gesture;
  }

  function workbenchLifecycle(form) {
    var existing = form[workbenchLifecycleProperty];
    if (existing && !existing.disposed) {
      if (!workbenchContains(form)) existing.dispose();
      else return existing;
    }
    var lifecycle = {
      disposed: false,
      controllers: Object.create(null),
      observer: null,
      canvasRefreshTask: null,
      activeRailGesture: null,
      register: function (name, controller) {
        if (this.disposed || this.controllers[name]) return false;
        this.controllers[name] = controller;
        return true;
      },
      get: function (name) {
        return this.disposed ? null : this.controllers[name] || null;
      },
      dispose: function () {
        if (this.disposed) return;
        this.disposed = true;
        var controllers = this.controllers;
        this.controllers = Object.create(null);
        Object.keys(controllers).forEach(function (name) {
          try {
            var controller = controllers[name];
            if (controller && typeof controller.dispose === "function") controller.dispose();
            else if (typeof controller === "function") controller();
          } catch (error) { /* best effort teardown */ }
        });
        if (this.canvasRefreshTask && this.canvasRefreshTask.cancel) this.canvasRefreshTask.cancel();
        this.canvasRefreshTask = null;
        if (this.observer) this.observer.disconnect();
        if (form[workbenchLifecycleProperty] === this) delete form[workbenchLifecycleProperty];
      }
    };
    form[workbenchLifecycleProperty] = lifecycle;
    if (window.MutationObserver && doc.documentElement) {
      lifecycle.observer = new MutationObserver(function () {
        if (!workbenchContains(form)) {
          lifecycle.dispose();
          return;
        }
        // Keep one observer per form, but let each registered controller
        // reconcile its own child bindings after an inner fragment swap.
        Object.keys(lifecycle.controllers).forEach(function (name) {
          var controller = lifecycle.controllers[name];
          if (controller && typeof controller.refresh === "function") {
            try { controller.refresh(); } catch (error) { /* best effort refresh */ }
          }
        });
      });
      lifecycle.observer.observe(doc.documentElement, { childList: true, subtree: true });
    }
    return lifecycle;
  }

  // workbenchStage — resolve the [data-studio-stage] surface inside the
  // workbench form. Mirrors studio-engines.js stage lookup; used by
  // bindRailResizers to compute pointer-relative widths.
  function workbenchStage(form) {
    if (!form) return null;
    return form.querySelector("[data-studio-stage]") || form.querySelector(".studio-workbench-stage") || form;
  }

  // clampNumber — clamp a number between min and max (mirrors legacy helper
  // used by setWorkbenchRailWidth).
  function clampNumber(value, min, max) {
    if (typeof value !== "number" || !Number.isFinite(value)) return min;
    if (value < min) return min;
    if (value > max) return max;
    return value;
  }

  // railWidthProperty — CSS custom property name for the rail side.
  function railWidthProperty(side) {
    return side === "left" ? "--studio-left-width" : "--studio-right-width";
  }

  // railBounds — read the handle's min/max/fallback width attributes. Used
  // by setRailWidth + currentRailWidth.
  function railBounds(handle, side) {
    var min = 200;
    var max = 720;
    var fallback = side === "left" ? 320 : 360;
    if (handle) {
      var minAttr = parseInt(handle.getAttribute("aria-valuemin"), 10);
      var maxAttr = parseInt(handle.getAttribute("aria-valuemax"), 10);
      var fallbackAttr = parseInt(handle.getAttribute("data-studio-resizer-default"), 10);
      if (Number.isFinite(minAttr)) min = minAttr;
      if (Number.isFinite(maxAttr)) max = maxAttr;
      if (Number.isFinite(fallbackAttr)) fallback = fallbackAttr;
    }
    return { min: min, max: max, fallback: fallback };
  }

  function inlineStylePropertyPresent(element, property) {
    if (!element || !element.style) return false;
    for (var i = 0; i < element.style.length; i++) {
      if (element.style.item(i) === property) return true;
    }
    return false;
  }

  // railSidebar — resolve the [data-studio-sidebar='<side>'] element inside
  // the form. Used by currentRailWidth as a getBoundingClientRect fallback
  // when the CSS variable hasn't been set yet.
  function railSidebar(form, side) {
    return form.querySelector(side === "left" ? "[data-studio-sidebar='left']" : "[data-studio-sidebar='right']");
  }

  // emitWorkbenchRailWidth — dispatch the rail-width-change /
  // rail-width-commit custom events used by bindChrome's saveLayoutSoon
  // wiring.
  function emitWorkbenchRailWidth(form, side, width, committed) {
    doc.dispatchEvent(new CustomEvent(committed ? "gosxstudio:rail-width-commit" : "gosxstudio:rail-width-change", {
      bubbles: true,
      detail: {
        form: form,
        side: side,
        width: width
      }
    }));
  }

  // emitWorkbenchChange — dispatch the gosxstudio:workbench-<name> custom
  // event the toolbar / command palette listens for.
  function emitWorkbenchChange(name, form, detail) {
    detail = detail || {};
    detail.form = form;
    doc.dispatchEvent(new CustomEvent("gosxstudio:workbench-" + name, {
      bubbles: true,
      detail: detail
    }));
  }

  // refreshWorkbenchCanvas — dispatch a window resize on the next rAF so
  // canvas engines recompute their layout. Used by syncViewport / syncZoom /
  // setStyleState / toggleRail / toggleFocus / toggleActivity / setMode.
  function refreshWorkbenchCanvas(form) {
    if (!form || !workbenchContains(form)) return;
    var lifecycle = workbenchLifecycle(form);
    if (!lifecycle || lifecycle.disposed) return;
    if (!lifecycle.canvasRefreshTask) {
      lifecycle.canvasRefreshTask = frameTask(function () {
        if (lifecycle.disposed || !workbenchContains(form)) return;
        window.dispatchEvent(new Event("resize"));
      });
    }
    lifecycle.canvasRefreshTask();
  }

  // reducedMotion — preference probe used by setMode's scrollIntoView.
  function reducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  // workbenchWorkingStateStorageKey — sessionStorage (NOT localStorage: this
  // is per-tab-session working context, not a durable layout preference,
  // and should not resurrect a stale mode/selection days later) key holding
  // the operator's in-progress working context: the active mode tab, the
  // current data-studio-selection key, and the canvas stage's scroll
  // position. A host form POST (Save) navigates the browser, which tears
  // down all in-memory JS state; without this, every save silently resets
  // the editor back to the default mode with no selection and a scrolled-
  // to-top canvas, discarding the operator's place in the document. Restored
  // on the next bindChromeIsland() call (i.e. after the post-save page
  // reload) — see restoreWorkbenchWorkingState.
  var workbenchWorkingStateStorageKey = "gosx-studio-editor-working-state";

  // readWorkbenchWorkingState / writeWorkbenchWorkingState — guarded
  // sessionStorage read/merge-write, mirroring readWorkbenchLayout /
  // saveLayoutIsland's error-swallowing convention (a lost working-state
  // restore is a UX regression, never a correctness break).
  function readWorkbenchWorkingState() {
    try {
      return JSON.parse(window.sessionStorage.getItem(workbenchWorkingStateStorageKey) || "{}") || {};
    } catch (error) {
      return {};
    }
  }

  function writeWorkbenchWorkingState(patch) {
    try {
      var current = readWorkbenchWorkingState();
      for (var key in patch) {
        if (Object.prototype.hasOwnProperty.call(patch, key)) current[key] = patch[key];
      }
      window.sessionStorage.setItem(workbenchWorkingStateStorageKey, JSON.stringify(current));
    } catch (error) {
      return;
    }
  }

  // restoreWorkbenchWorkingState — applies a persisted mode/selection/scroll
  // position onto the freshly bound form + stage, then rebinds the
  // persistence listeners (selection attribute + stage scroll) so future
  // changes within this tab session keep the snapshot current. Called once
  // per bindChromeIsland() call, before setModeIsland seeds the mode from
  // the server-rendered default.
  function restoreWorkbenchWorkingState(form, stage) {
    var state = readWorkbenchWorkingState();
    var disposed = false;
    var selectionObserver = null;
    var boundStage = null;
    var saveScrollSoon = null;
    var stageScrollHandler = null;
    var restoreScroll = null;
    if (state.selection) {
      form.setAttribute("data-studio-selection", state.selection);
    }

    if (window.MutationObserver) {
      selectionObserver = new MutationObserver(function () {
        if (disposed || !workbenchContains(form)) return;
        var selection = form.getAttribute("data-studio-selection") || "";
        state.selection = selection;
        writeWorkbenchWorkingState({ selection: state.selection });
      });
      selectionObserver.observe(form, { attributes: true, attributeFilter: ["data-studio-selection"] });
    }

    function unbindStage() {
      if (boundStage && stageScrollHandler) {
        boundStage.removeEventListener("scroll", stageScrollHandler, { passive: true });
      }
      if (saveScrollSoon && saveScrollSoon.cancel) saveScrollSoon.cancel();
      if (restoreScroll && restoreScroll.cancel) restoreScroll.cancel();
      boundStage = null;
      saveScrollSoon = null;
      stageScrollHandler = null;
      restoreScroll = null;
    }

    function bindStage(nextStage) {
      if (disposed || boundStage === nextStage) return;
      unbindStage();
      boundStage = nextStage || null;
      if (!boundStage) return;
      saveScrollSoon = frameTask(function () {
        if (disposed || !workbenchContains(form) || nextStage.isConnected === false || boundStage !== nextStage || workbenchStage(form) !== nextStage) return;
        state.scrollTop = nextStage.scrollTop;
        state.scrollLeft = nextStage.scrollLeft;
        writeWorkbenchWorkingState({ scrollTop: state.scrollTop, scrollLeft: state.scrollLeft });
      });
      stageScrollHandler = function () {
        if (disposed || !workbenchContains(form) || nextStage.isConnected === false || boundStage !== nextStage || workbenchStage(form) !== nextStage) return;
        saveScrollSoon();
      };
      boundStage.addEventListener("scroll", stageScrollHandler, { passive: true });
      if (state.scrollTop || state.scrollLeft) {
        restoreScroll = frameTask(function () {
          if (disposed || !workbenchContains(form) || boundStage !== nextStage) return;
          if (state.scrollTop) nextStage.scrollTop = state.scrollTop;
          if (state.scrollLeft) nextStage.scrollLeft = state.scrollLeft;
        });
        restoreScroll();
      }
    }

    bindStage(stage);
    return {
      state: state,
      refresh: bindStage,
      dispose: function () {
        if (disposed) return;
        disposed = true;
        unbindStage();
        if (selectionObserver) selectionObserver.disconnect();
      }
    };
  }

  // Workbench label maps — exact mirror of studio-engines.js:242-265.
  var workbenchLayoutStorageKey = "gosx-studio-editor-layout";
  var workbenchModeLabels = {
    home: "Home",
    look: "Look",
    brand: "Brand",
    publish: "Publish",
    advanced: "Advanced",
    structure: "Home",
    content: "Home",
    style: "Look",
    manage: "Advanced",
    commerce: "Advanced",
    flows: "Advanced",
    preview: "Publish"
  };
  var workbenchViewportLabels = {
    desktop: "Desktop",
    tablet: "Tablet",
    mobile: "Mobile"
  };
  var workbenchStyleStateLabels = {
    default: "Default",
    hover: "Hover",
    focus: "Focus"
  };

  function normalizeWorkbenchMode(mode) {
    if (mode === "structure" || mode === "content") return "home";
    if (mode === "style") return "look";
    if (mode === "manage" || mode === "flows") return "advanced";
    return mode || "home";
  }

  // workbenchModeGateRoot — the scope setMode searches for
  // [data-studio-mode-panel] elements. Most mode-gated panels render inside
  // the workbench <form> itself (the left/right rails), but a host's own
  // companion panels that need their OWN <form> (e.g. Muddy/Noni's direct-
  // edit panel, interactions inspector, flow field editor, shared-
  // components palette — a <form> cannot legally nest inside the workbench's
  // own <form id="websiteEditorForm">, so hosts render them as siblings
  // instead) live OUTSIDE the form. Search the nearest
  // [data-gosx-studio-backend-editor-renderer] ancestor (the whole backend
  // editor page — see shell.RenderBackendEditorPage) when present, so those
  // sibling panels are reachable too; fall back to the form itself so
  // fixtures/hosts that mount the form standalone (no such ancestor) keep
  // their existing form-scoped behavior unchanged.
  function workbenchModeGateRoot(form) {
    if (form.closest) {
      var root = form.closest("[data-gosx-studio-backend-editor-renderer]");
      if (root) return root;
    }
    return form;
  }

  function workbenchModePanel(form, mode) {
    var root = workbenchModeGateRoot(form);
    var selector = '[data-studio-mode-panel="' + attrValue(normalizeWorkbenchMode(mode)) + '"]';
    return root.querySelector(".studio-right-rail " + selector) || root.querySelector(".editor-panel" + selector) || root.querySelector(selector);
  }

  function workbenchPanelFollowsMode(panel) {
    return Boolean(panel.closest(".studio-right-rail") || panel.classList.contains("editor-panel"));
  }

  function setWorkbenchReadout(form, selector, value) {
    Array.prototype.forEach.call(form.querySelectorAll(selector), function (node) {
      node.textContent = value;
    });
  }

  function updateWorkbenchModeLabel(form, mode) {
    setWorkbenchReadout(form, "[data-studio-mode-label]", workbenchModeLabels[mode] || mode || "Home");
  }

  function workbenchRailState(form, side) {
    return form.getAttribute("data-studio-" + side) || "open";
  }

  function syncWorkbenchRailButtons(form) {
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-rail-toggle]"), function (button) {
      var side = button.getAttribute("data-studio-rail-toggle");
      button.setAttribute("aria-pressed", workbenchRailState(form, side) === "open" ? "true" : "false");
    });
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-focus-toggle]"), function (button) {
      button.setAttribute("aria-pressed", form.getAttribute("data-studio-focus") === "true" ? "true" : "false");
    });
  }

  function workbenchActivityState(form) {
    return form.getAttribute("data-studio-activity-state") || "open";
  }

  function syncWorkbenchActivityButtons(form) {
    var open = workbenchActivityState(form) === "open";
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-activity-toggle]"), function (button) {
      button.setAttribute("aria-pressed", open ? "true" : "false");
      if (button.closest("[data-studio-activity-drawer]")) button.textContent = open ? "Hide" : "Show";
    });
  }

  function readWorkbenchLayout() {
    try {
      return JSON.parse(window.localStorage.getItem(workbenchLayoutStorageKey) || "{}") || {};
    } catch (error) {
      return {};
    }
  }

  function applyWorkbenchLayout(form) {
    var layout = readWorkbenchLayout();
    if (layout.left) form.style.setProperty("--studio-left-width", layout.left);
    if (layout.right) form.style.setProperty("--studio-right-width", layout.right);
    if (layout.activity) form.setAttribute("data-studio-activity-state", layout.activity);
  }

  function normalizeWorkbenchStyleState(state) {
    return workbenchStyleStateLabels[state] ? state : "default";
  }

  // ===== Method implementations (15 globals published below as TDD progresses) =====

  // bindRailResizers(root) — mirrors bindWorkbenchRailResizers at
  // studio-engines.js:194. Binds the [data-studio-resizer] pointer drag +
  // ArrowLeft/ArrowRight keyboard nudges on rail handles.
  //
  // Idempotency: each handle gets
  // [data-gosx-studio-resizer-island-bound] so repeated host calls do not
  // stack pointer or keyboard listeners.
  function bindRailResizersIsland(root) {
    var form = editorWorkbench(root);
    if (!form || !workbenchContains(form)) return;
    var lifecycle = workbenchLifecycle(form);
    var existing = lifecycle.get("railResizers");
    if (existing) {
      existing.refresh();
      return;
    }
    var disposed = false;
    var handleBindings = [];
    var activeGesture = null;

    function listen(target, type, handler, options) {
      target.addEventListener(type, handler, options);
      return function () {
        target.removeEventListener(type, handler, options);
      };
    }

    function removeGestureListeners(gesture) {
      doc.removeEventListener("pointermove", gesture.move);
      doc.removeEventListener("pointerup", gesture.finish);
      doc.removeEventListener("pointercancel", gesture.cancel);
      doc.removeEventListener("keydown", gesture.escape);
      window.removeEventListener("blur", gesture.cancelWindow);
      doc.removeEventListener("visibilitychange", gesture.cancelVisibility);
    }

    function restoreGestureWidth(gesture) {
      if (!gesture || !gesture.form) return;
      var property = railWidthProperty(gesture.side);
      var inlineWidth = gesture.startInlineWidth;
      if (inlineWidth && inlineWidth.present) {
        gesture.form.style.setProperty(property, inlineWidth.value, inlineWidth.priority);
      } else {
        gesture.form.style.removeProperty(property);
      }
      var handle = gesture.handle;
      if (!handle || !gesture.form.contains(handle)) {
        handle = gesture.form.querySelector("[data-studio-resizer='" + attrValue(gesture.side) + "']");
      }
      if (!handle) return;
      if (gesture.startAriaValue && gesture.startAriaValue.present) {
        handle.setAttribute("aria-valuenow", gesture.startAriaValue.value || "");
      } else {
        handle.removeAttribute("aria-valuenow");
      }
    }

    function gestureHasCurrentOwnership(gesture) {
      if (!gesture || !gesture.form || !gesture.stage || !gesture.handle) return false;
      if (!workbenchContains(gesture.form) || !gesture.form.contains(gesture.stage) || !gesture.form.contains(gesture.handle)) return false;
      if (gesture.stage.isConnected === false || gesture.handle.isConnected === false) return false;
      if (workbenchStage(gesture.form) !== gesture.stage) return false;
      return gesture.form.querySelector("[data-studio-resizer='" + attrValue(gesture.side) + "']") === gesture.handle;
    }

    function finishGesture(gesture, committed, pointerEvent) {
      if (!gesture || gesture.done || activeGesture !== gesture) return;
      if (pointerEvent && pointerEvent.pointerId !== undefined && pointerEvent.pointerId !== gesture.pointerId) return;
      gesture.done = true;
      // Clear ownership before releasePointerCapture: a synchronous
      // lostpointercapture callback must observe an already-finished gesture.
      if (activeGesture === gesture) activeGesture = null;
      if (lifecycle.activeRailGesture === gesture) lifecycle.activeRailGesture = null;
      removeGestureListeners(gesture);
      if (gesture.handle.classList) gesture.handle.classList.remove("is-resizing");
      try {
        if (gesture.handle.releasePointerCapture) gesture.handle.releasePointerCapture(gesture.pointerId);
      } catch (error) { /* tolerate stale pointer capture during teardown */ }
      if (committed) {
        var width = pointerEvent && pointerEvent.clientX !== undefined
          ? gesture.widthFor(pointerEvent)
          : currentRailWidthIsland(gesture.form, gesture.side, gesture.handle);
        setRailWidthIsland(gesture.form, gesture.side, width, gesture.handle, true);
      } else {
        // Cancellation is deliberately silent: the in-progress change is
        // rolled back and no commit event reaches saveLayout. Recompute the
        // live canvas after restoring the width so a refresh that already ran
        // during the drag cannot leave the canvas laid out for stale geometry.
        restoreGestureWidth(gesture);
        if (!disposed && workbenchContains(gesture.form)) refreshWorkbenchCanvas(gesture.form);
      }
    }

    function cancelGesture(gesture) {
      finishGesture(gesture, false, null);
    }

    function dispose() {
      if (disposed) return;
      disposed = true;
      if (activeGesture) cancelGesture(activeGesture);
      for (var i = handleBindings.length - 1; i >= 0; i--) {
        handleBindings[i].dispose();
      }
      handleBindings = [];
      form.removeAttribute("data-gosx-studio-resizers-island-bound");
    }

    function bindHandle(handle, stage) {
      for (var existingIndex = 0; existingIndex < handleBindings.length; existingIndex++) {
        if (handleBindings[existingIndex].handle === handle) return;
      }
      if (handle.dataset.gosxStudioResizerIslandBound === "true") {
        // A marker without an entry belongs to a previous controller that
        // was interrupted before its expando cleanup. Reclaim it safely.
        handle.removeAttribute("data-gosx-studio-resizer-island-bound");
      }
      handle.dataset.gosxStudioResizerIslandBound = "true";
      var pointerdown = function (event) {
        if (disposed || !workbenchContains(form) || !form.contains(handle) || activeGesture || event.button !== 0) return;
        if (!stage || workbenchStage(form) !== stage) return;
        event.preventDefault();
        var side = handle.getAttribute("data-studio-resizer");
        var rect = stage.getBoundingClientRect();
        var gesture;
        function widthFor(pointerEvent) {
          return side === "left" ? pointerEvent.clientX - rect.left : rect.right - pointerEvent.clientX;
        }
        gesture = {
          form: form,
          handle: handle,
          side: side,
          stage: stage,
          rect: rect,
          pointerId: event.pointerId,
          widthFor: widthFor,
          startWidth: currentRailWidthIsland(form, side, handle),
          startInlineWidth: {
            present: inlineStylePropertyPresent(form, railWidthProperty(side)),
            value: form.style.getPropertyValue(railWidthProperty(side)),
            priority: form.style.getPropertyPriority(railWidthProperty(side))
          },
          startAriaValue: {
            present: handle.hasAttribute("aria-valuenow"),
            value: handle.getAttribute("aria-valuenow")
          },
          done: false,
        };
        activeGesture = gesture;
        lifecycle.activeRailGesture = gesture;
        handle.classList.add("is-resizing");
        function move(pointerEvent) {
          if (disposed || gesture.done || activeGesture !== gesture || pointerEvent.pointerId !== gesture.pointerId) return;
          if (!gestureHasCurrentOwnership(gesture)) {
            cancelGesture(gesture);
            return;
          }
          setRailWidthIsland(form, side, widthFor(pointerEvent), handle, false);
        }
        function finish(pointerEvent) {
          if (disposed) return;
          if (!gestureHasCurrentOwnership(gesture)) {
            cancelGesture(gesture);
            return;
          }
          finishGesture(gesture, true, pointerEvent);
        }
        function cancel(pointerEvent) {
          if (disposed) return;
          finishGesture(gesture, false, pointerEvent);
        }
        function escape(escapeEvent) {
          if (disposed || gesture.done || activeGesture !== gesture || escapeEvent.key !== "Escape") return;
          escapeEvent.preventDefault();
          cancelGesture(gesture);
        }
        function cancelWindow() {
          if (disposed || gesture.done || activeGesture !== gesture) return;
          cancelGesture(gesture);
        }
        function cancelVisibility() {
          if (disposed || gesture.done || activeGesture !== gesture || doc.visibilityState !== "hidden") return;
          cancelGesture(gesture);
        }
        gesture.move = move;
        gesture.finish = finish;
        gesture.cancel = cancel;
        gesture.escape = escape;
        gesture.cancelWindow = cancelWindow;
        gesture.cancelVisibility = cancelVisibility;
        setRailWidthIsland(form, side, widthFor(event), handle, false);
        doc.addEventListener("pointermove", move);
        doc.addEventListener("pointerup", finish);
        doc.addEventListener("pointercancel", cancel);
        doc.addEventListener("keydown", escape);
        window.addEventListener("blur", cancelWindow);
        doc.addEventListener("visibilitychange", cancelVisibility);
        try {
          if (handle.setPointerCapture) handle.setPointerCapture(event.pointerId);
        } catch (error) {
          // Synthetic or stale pointer IDs may not be capturable. Document
          // lifecycle listeners remain authoritative for the active gesture.
        }
      };
      var removePointerDown = listen(handle, "pointerdown", pointerdown);
      var keydown = function (event) {
        if (disposed || !workbenchContains(form) || !form.contains(handle)) return;
        if (event.key === "Escape") {
          if (activeGesture && activeGesture.handle === handle) {
            event.preventDefault();
            cancelGesture(activeGesture);
          }
          return;
        }
        if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
        event.preventDefault();
        var side = handle.getAttribute("data-studio-resizer");
        var step = event.shiftKey ? 48 : 24;
        var delta = event.key === "ArrowRight" ? step : -step;
        setRailWidthIsland(form, side, currentRailWidthIsland(form, side, handle) + (side === "left" ? delta : -delta), handle, true);
      };
      var removeKeydown = listen(handle, "keydown", keydown);
      var lostPointerCapture = function (event) {
        if (!activeGesture || activeGesture.handle !== handle) return;
        if (event.pointerId !== undefined && event.pointerId !== activeGesture.pointerId) return;
        cancelGesture(activeGesture);
      };
      var removeLostPointerCapture = listen(handle, "lostpointercapture", lostPointerCapture);
      var binding = {
        handle: handle,
        stage: stage,
        dispose: function () {
          if (activeGesture && activeGesture.handle === handle) cancelGesture(activeGesture);
          removePointerDown();
          removeKeydown();
          removeLostPointerCapture();
          handle.removeAttribute("data-gosx-studio-resizer-island-bound");
        }
      };
      handleBindings.push(binding);
    }

    function refresh() {
      if (disposed || !workbenchContains(form)) return;
      var stage = workbenchStage(form);
      for (var i = handleBindings.length - 1; i >= 0; i--) {
        var binding = handleBindings[i];
        if (!form.contains(binding.handle) || !workbenchContains(binding.handle) || binding.stage !== stage) {
          binding.dispose();
          handleBindings.splice(i, 1);
        }
      }
      if (!stage) return;
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-resizer]"), function (handle) {
        bindHandle(handle, stage);
      });
    }

    var controller = { dispose: dispose, refresh: refresh };
    if (!lifecycle.register("railResizers", controller)) return;
    form.dataset.gosxStudioResizersIslandBound = "true";
    refresh();
  }

  window.__gosx_workbench_runtime_island_bindRailResizers = bindRailResizersIsland;

  // setRailWidthIsland and currentRailWidthIsland are forward-declared
  // helpers — their public island publications happen at the end of this
  // IIFE (methods 14/15 + 15/15 in the TDD sequence). Function
  // declarations hoist within scope so the forward references above work
  // without warning. Same pattern as slice 5's bindThemeIsland referencing
  // applyThemeIsland before it's published.
  // setRailWidth(form, side, width, handle, committed) — mirrors
  // setWorkbenchRailWidth at studio-engines.js:185. Clamps width to the
  // handle's min/max bounds, writes --studio-{left,right}-width on the
  // form, updates the handle's aria-valuenow, emits
  // gosxstudio:rail-width-change or gosxstudio:rail-width-commit
  // depending on the committed flag.
  function setRailWidthIsland(form, side, width, handle, committed) {
    if (!form || (side !== "left" && side !== "right")) return;
    var bounds = railBounds(handle, side);
    var next = clampNumber(Math.round(width), bounds.min, bounds.max);
    form.style.setProperty(railWidthProperty(side), next + "px");
    if (handle) handle.setAttribute("aria-valuenow", String(next));
    emitWorkbenchRailWidth(form, side, next, !!committed);
  }

  window.__gosx_workbench_runtime_island_setRailWidth = setRailWidthIsland;

  // currentRailWidth(form, side, handle) — mirrors currentWorkbenchRailWidth
  // at studio-engines.js:165. Pure read: returns the form's
  // --studio-{left,right}-width custom property as an integer (or the
  // sidebar's bounding-rect width fallback, or the handle's bounds
  // fallback). Per the slice plan, this method ships as a
  // signal-derivation rather than a mutator — it reads the same DOM state
  // setRailWidth writes (--studio-{left,right}-width CSS custom
  // properties) and returns it unmodified. No event dispatch, no canvas
  // refresh.
  function currentRailWidthIsland(form, side, handle) {
    if (!form) return railBounds(handle, side).fallback;
    var custom = form.style.getPropertyValue(railWidthProperty(side));
    var parsed = parseInt(custom, 10);
    if (Number.isFinite(parsed)) return parsed;
    var node = railSidebar(form, side);
    if (node) return Math.round(node.getBoundingClientRect().width);
    return railBounds(handle, side).fallback;
  }

  window.__gosx_workbench_runtime_island_currentRailWidth = currentRailWidthIsland;

  // bindChrome(root) — mirrors bindWorkbenchChrome at studio-engines.js:516.
  // Binds the workbench form's delegated click handler (fans out to
  // setMode / syncViewport / syncZoom / setStyleState / toggleRail /
  // toggleFocus / toggleActivity), wires saveLayoutSoon to
  // rail-width-change/commit events, applies the persisted layout, and
  // seeds the workbench mode / viewport / zoom on initial bind.
  //
  // Idempotency: one chrome controller per form; its child command-palette
  // binding is reconciled independently after inner fragment replacement so
  // repeated host calls do not stack delegated click or layout listeners.
  function bindChromeIsland(root) {
    var form = editorWorkbench(root);
    if (!form || !workbenchContains(form)) return;
    var lifecycle = workbenchLifecycle(form);
    var existing = lifecycle.get("chrome");
    if (existing) {
      existing.refresh();
      return;
    }
    var disposed = false;
    var cleanupFns = [];
    var saveLayoutSoon = null;
    var workingStateBinding = null;
    var commandPaletteBinding = null;
    function listen(target, type, handler, options) {
      target.addEventListener(type, handler, options);
      cleanupFns.push(function () {
        target.removeEventListener(type, handler, options);
      });
    }
    function refresh() {
      if (disposed || !workbenchContains(form)) return;
      if (workingStateBinding) workingStateBinding.refresh(workbenchStage(form));
      if (commandPaletteBinding) commandPaletteBinding.refresh();
    }
    function dispose() {
      if (disposed) return;
      disposed = true;
      if (saveLayoutSoon && saveLayoutSoon.cancel) saveLayoutSoon.cancel();
      if (commandPaletteBinding) commandPaletteBinding.dispose();
      if (workingStateBinding) workingStateBinding.dispose();
      for (var i = cleanupFns.length - 1; i >= 0; i--) cleanupFns[i]();
      cleanupFns = [];
      form.removeAttribute("data-gosx-studio-workbench-chrome-island-bound");
    }
    if (!lifecycle.register("chrome", { dispose: dispose, refresh: refresh })) return;
    form.dataset.gosxStudioWorkbenchChromeIslandBound = "true";
    saveLayoutSoon = frameTask(function (targetForm) {
      if (disposed || !workbenchContains(targetForm)) return;
      saveLayoutIsland(targetForm);
    });
    listen(form, "click", function (event) {
      if (disposed || !workbenchContains(form)) return;
      var mode = event.target.closest("[data-studio-mode-control]");
      if (mode && form.contains(mode)) {
        event.preventDefault();
        setModeIsland(form, mode.getAttribute("data-studio-mode-control"), true);
        return;
      }
      var viewport = event.target.closest("[data-studio-viewport]");
      if (viewport && form.contains(viewport)) {
        event.preventDefault();
        syncViewportIsland(form, viewport.getAttribute("data-studio-viewport"));
        return;
      }
      var zoom = event.target.closest("button[data-studio-zoom], [role='button'][data-studio-zoom]");
      if (zoom && form.contains(zoom)) {
        event.preventDefault();
        syncZoomIsland(form, zoom.getAttribute("data-studio-zoom"));
        return;
      }
      var styleState = event.target.closest("button[data-studio-style-state], [role='button'][data-studio-style-state]");
      if (styleState && form.contains(styleState)) {
        event.preventDefault();
        setStyleStateIsland(form, styleState.getAttribute("data-studio-style-state"));
        return;
      }
      var rail = event.target.closest("[data-studio-rail-toggle]");
      if (rail && form.contains(rail)) {
        event.preventDefault();
        toggleRailIsland(form, rail.getAttribute("data-studio-rail-toggle"));
        return;
      }
      var focus = event.target.closest("[data-studio-focus-toggle]");
      if (focus && form.contains(focus)) {
        event.preventDefault();
        toggleFocusIsland(form);
        return;
      }
      var activity = event.target.closest("[data-studio-activity-toggle]");
      if (activity && form.contains(activity)) {
        event.preventDefault();
        toggleActivityIsland(form);
      }
    });
    listen(doc, "gosxstudio:rail-width-change", function (event) {
      if (disposed || !workbenchContains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      // A pointer gesture owns its active-side width until it commits. The
      // live change still refreshes the canvas, but serializing it would make
      // a later cancel durable. Independent layout saves use the captured
      // pre-drag width in saveLayoutIsland instead.
      var activeGesture = activeRailGestureFor(form);
      if (!activeGesture || !event.detail || event.detail.side !== activeGesture.side) saveLayoutSoon(form);
      refreshWorkbenchCanvas(form);
    });
    listen(doc, "gosxstudio:rail-width-commit", function (event) {
      if (disposed || !workbenchContains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      saveLayoutIsland(form);
      refreshWorkbenchCanvas(form);
    });
    applyWorkbenchLayout(form);
    var stage = workbenchStage(form);
    workingStateBinding = restoreWorkbenchWorkingState(form, stage);
    var workingState = workingStateBinding.state;
    setModeIsland(form, workingState.mode || form.getAttribute("data-studio-mode") || "home", false);
    if (!form.hasAttribute("data-studio-left")) form.setAttribute("data-studio-left", "open");
    if (!form.hasAttribute("data-studio-right")) form.setAttribute("data-studio-right", "open");
    if (!form.hasAttribute("data-studio-focus")) form.setAttribute("data-studio-focus", "false");
    if (!form.hasAttribute("data-studio-activity-state")) form.setAttribute("data-studio-activity-state", "open");
    syncWorkbenchRailButtons(form);
    syncWorkbenchActivityButtons(form);
    commandPaletteBinding = bindCommandPaletteIsland(form);
    commandPaletteBinding.refresh();
    syncViewportIsland(form, "desktop");
    var canvas = form.querySelector("[data-studio-canvas]");
    syncZoomIsland(form, canvas ? canvas.getAttribute("data-studio-canvas-zoom") || "fit" : "fit");
  }

  window.__gosx_workbench_runtime_island_bindChrome = bindChromeIsland;

  // setMode(form, mode, scroll) — mirrors setWorkbenchMode at
  // studio-engines.js:343. Sets data-studio-mode on the form, toggles
  // aria-pressed on [data-studio-mode-control] buttons, toggles
  // is-mode-active / hidden / aria-hidden on [data-studio-mode-panel]
  // siblings, scrolls active panel into view when requested, updates
  // [data-studio-mode-label] readouts, emits gosxstudio:workbench-mode-change.
  function setModeIsland(form, mode, scroll) {
    if (!form) return;
    mode = normalizeWorkbenchMode(mode);
    form.setAttribute("data-studio-mode", mode);
    writeWorkbenchWorkingState({ mode: mode });
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-mode-control]"), function (button) {
      button.setAttribute("aria-pressed", normalizeWorkbenchMode(button.getAttribute("data-studio-mode-control")) === mode ? "true" : "false");
    });
    // Widened to workbenchModeGateRoot (not just the form): a host's own
    // companion panels needing their own <form> (direct-edit, interactions,
    // flow field editor, shared components...) render as siblings of the
    // workbench form, not descendants of it — see workbenchModeGateRoot.
    Array.prototype.forEach.call(workbenchModeGateRoot(form).querySelectorAll("[data-studio-mode-panel]"), function (panel) {
      var active = normalizeWorkbenchMode(panel.getAttribute("data-studio-mode-panel")) === mode;
      panel.classList.toggle("is-mode-active", active);
      if (workbenchPanelFollowsMode(panel)) {
        panel.hidden = !active;
        panel.setAttribute("aria-hidden", active ? "false" : "true");
      } else {
        panel.hidden = false;
        panel.setAttribute("aria-hidden", "false");
      }
    });
    var target = workbenchModePanel(form, mode);
    if (scroll && target && target.scrollIntoView) {
      target.scrollIntoView({ behavior: reducedMotion() ? "auto" : "smooth", block: "nearest" });
    }
    updateWorkbenchModeLabel(form, mode);
    emitWorkbenchChange("mode-change", form, { mode: mode, scroll: !!scroll });
  }

  window.__gosx_workbench_runtime_island_setMode = setModeIsland;

  // syncViewport(form, viewport) — mirrors syncWorkbenchViewport at
  // studio-engines.js:369. Sets data-studio-breakpoint on the form,
  // data-studio-preview-viewport on the .editor-preview-shell (only when
  // the viewport island is not already managing it via
  // data-studio-viewport-island="true"), updates
  // [data-studio-viewport-label] readouts, emits
  // gosxstudio:workbench-viewport-change, refreshes the canvas.
  function syncViewportIsland(form, viewport) {
    if (!form) return;
    viewport = viewport || "desktop";
    var shell = form.querySelector(".editor-preview-shell");
    if (shell && shell.getAttribute("data-studio-viewport-island") !== "true") {
      shell.setAttribute("data-studio-preview-viewport", viewport);
    }
    form.setAttribute("data-studio-breakpoint", viewport);
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-viewport-current]"), function (root) {
      root.setAttribute("data-studio-viewport-current", viewport);
    });
    Array.prototype.forEach.call(form.querySelectorAll("button[data-studio-viewport], [role='button'][data-studio-viewport]"), function (button) {
      button.setAttribute("aria-pressed", button.getAttribute("data-studio-viewport") === viewport ? "true" : "false");
    });
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-viewport-label]"), function (node) {
      node.textContent = workbenchViewportLabels[viewport] || viewport.charAt(0).toUpperCase() + viewport.slice(1);
    });
    emitWorkbenchChange("viewport-change", form, { viewport: viewport });
    refreshWorkbenchCanvas(form);
  }

  window.__gosx_workbench_runtime_island_syncViewport = syncViewportIsland;

  // activateViewport(form, viewport) — mirrors activateWorkbenchViewport at
  // studio-engines.js:382. Clicks the matching [data-studio-viewport="<x>"]
  // button if present; otherwise falls through to syncViewportIsland.
  function activateViewportIsland(form, viewport) {
    if (!form) return;
    viewport = viewport || "desktop";
    var button = form.querySelector('[data-studio-viewport="' + attrValue(viewport) + '"]');
    if (button && button.click) {
      button.click();
      return;
    }
    syncViewportIsland(form, viewport);
  }

  window.__gosx_workbench_runtime_island_activateViewport = activateViewportIsland;

  // currentBreakpoint(form) — mirrors currentWorkbenchBreakpoint at
  // studio-engines.js:393. Pure read: derives the form's
  // data-studio-breakpoint attribute (or the preview-shell's
  // data-studio-preview-viewport fallback, or "desktop"). Per the slice
  // plan, this method ships as a signal-derivation rather than a mutator —
  // it reads the same DOM state syncViewport writes (data-studio-breakpoint
  // / data-studio-preview-viewport) and returns it unmodified. No event
  // dispatch, no canvas refresh. The BridgeShim routes through it
  // uniformly so future $workbench.viewport signal consumers can substitute
  // a signal-read for the attribute-read without changing the contract.
  function currentBreakpointIsland(form) {
    if (!form) return "desktop";
    var viewport = form.getAttribute("data-studio-breakpoint");
    if (viewport) return viewport;
    var shell = form.querySelector(".editor-preview-shell");
    return shell ? shell.getAttribute("data-studio-preview-viewport") || "desktop" : "desktop";
  }

  window.__gosx_workbench_runtime_island_currentBreakpoint = currentBreakpointIsland;

  // setStyleState(form, state) — mirrors setWorkbenchStyleState at
  // studio-engines.js:405. Sets data-studio-style-state on the form,
  // toggles aria-pressed on [data-studio-style-state] buttons, writes
  // data-style-state / data-style-breakpoint / data-style-valid on
  // [data-studio-style-scope] wrappers, emits
  // gosxstudio:workbench-style-state-change, refreshes the canvas.
  //
  // data-style-breakpoint is derived via currentBreakpointIsland so the
  // attribute stays in sync with the viewport state without re-reading
  // the raw form attribute. Per the slice plan, currentBreakpoint is a
  // pure derivation — this method's call into it has no side effects.
  function setStyleStateIsland(form, state) {
    if (!form) return;
    state = normalizeWorkbenchStyleState(state || "default");
    form.setAttribute("data-studio-style-state", state);
    Array.prototype.forEach.call(form.querySelectorAll("button[data-studio-style-state], [role='button'][data-studio-style-state]"), function (button) {
      button.setAttribute("aria-pressed", button.getAttribute("data-studio-style-state") === state ? "true" : "false");
    });
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-style-scope]"), function (scope) {
      scope.setAttribute("data-style-state", state);
      scope.setAttribute("data-style-breakpoint", currentBreakpointIsland(form));
      scope.setAttribute("data-style-valid", form.getAttribute("data-studio-style-valid") !== "false" ? "true" : "false");
    });
    emitWorkbenchChange("style-state-change", form, { state: state });
    refreshWorkbenchCanvas(form);
  }

  window.__gosx_workbench_runtime_island_setStyleState = setStyleStateIsland;

  // syncZoom(form, zoom) — mirrors syncWorkbenchZoom at
  // studio-engines.js:421. Sets data-studio-canvas-zoom on the
  // [data-studio-canvas] element, syncs zoom toolbar state, emits
  // gosxstudio:workbench-zoom-change, refreshes the canvas via a resize event.
  function syncZoomIsland(form, zoom) {
    if (!form) return;
    zoom = zoom || "fit";
    var canvas = form.querySelector("[data-studio-canvas]");
    if (canvas) canvas.setAttribute("data-studio-canvas-zoom", zoom);
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-zoom-island]"), function (root) {
      root.setAttribute("data-studio-zoom-current", zoom);
    });
    Array.prototype.forEach.call(form.querySelectorAll("button[data-studio-zoom], [role='button'][data-studio-zoom]"), function (button) {
      button.setAttribute("aria-pressed", button.getAttribute("data-studio-zoom") === zoom ? "true" : "false");
    });
    emitWorkbenchChange("zoom-change", form, { zoom: zoom });
    refreshWorkbenchCanvas(form);
  }

  window.__gosx_workbench_runtime_island_syncZoom = syncZoomIsland;

  // activateZoom(form, zoom) — mirrors activateWorkbenchZoom at
  // studio-engines.js:430. Clicks the matching [data-studio-zoom="<x>"]
  // button if present; otherwise falls through to syncZoomIsland.
  function activateZoomIsland(form, zoom) {
    if (!form) return;
    zoom = zoom || "fit";
    var button = form.querySelector('button[data-studio-zoom="' + attrValue(zoom) + '"], [role="button"][data-studio-zoom="' + attrValue(zoom) + '"]');
    if (button && button.click) {
      button.click();
      return;
    }
    syncZoomIsland(form, zoom);
  }

  window.__gosx_workbench_runtime_island_activateZoom = activateZoomIsland;

  // toggleRail(form, side) — mirrors toggleWorkbenchRail at
  // studio-engines.js:480. Flips data-studio-{left,right} between "open"
  // and "collapsed", sets data-studio-focus to "false" (any rail toggle
  // exits focus mode), re-syncs the rail toggle aria-pressed states,
  // emits gosxstudio:workbench-rail-change, refreshes the canvas.
  function toggleRailIsland(form, side) {
    if (!form || (side !== "left" && side !== "right")) return;
    form.setAttribute("data-studio-focus", "false");
    form.setAttribute("data-studio-" + side, workbenchRailState(form, side) === "open" ? "collapsed" : "open");
    syncWorkbenchRailButtons(form);
    emitWorkbenchChange("rail-change", form, { side: side, state: workbenchRailState(form, side) });
    refreshWorkbenchCanvas(form);
  }

  window.__gosx_workbench_runtime_island_toggleRail = toggleRailIsland;

  // toggleFocus(form) — mirrors toggleWorkbenchFocus at studio-engines.js:489.
  // Flips data-studio-focus between "true" and "false", re-syncs the rail
  // toggle aria-pressed states (the rail buttons share the focus pressed
  // state — see syncWorkbenchRailButtons in studio-engines.js:450), emits
  // gosxstudio:workbench-focus-change, refreshes the canvas.
  function toggleFocusIsland(form) {
    if (!form) return;
    form.setAttribute("data-studio-focus", form.getAttribute("data-studio-focus") === "true" ? "false" : "true");
    syncWorkbenchRailButtons(form);
    emitWorkbenchChange("focus-change", form, { focus: form.getAttribute("data-studio-focus") === "true" });
    refreshWorkbenchCanvas(form);
  }

  window.__gosx_workbench_runtime_island_toggleFocus = toggleFocusIsland;

  // setActivityIsland — internal helper mirroring setWorkbenchActivity at
  // studio-engines.js:467. Writes data-studio-activity-state, re-syncs
  // activity-toggle buttons, persists via saveLayout, emits the change
  // event, refreshes the canvas. Used by toggleActivityIsland — not
  // published as a public island global because the legacy contract
  // doesn't expose it.
  function setActivityIsland(form, state) {
    if (!form) return;
    form.setAttribute("data-studio-activity-state", state === "collapsed" ? "collapsed" : "open");
    syncWorkbenchActivityButtons(form);
    saveLayoutIsland(form);
    emitWorkbenchChange("activity-change", form, { state: workbenchActivityState(form) });
    refreshWorkbenchCanvas(form);
  }

  // toggleActivity(form) — mirrors toggleWorkbenchActivity at
  // studio-engines.js:476. Flips data-studio-activity-state between "open"
  // and "collapsed" via setActivityIsland, which handles the readout
  // updates / persistence / event dispatch / canvas refresh.
  function toggleActivityIsland(form) {
    setActivityIsland(form, workbenchActivityState(form) === "open" ? "collapsed" : "open");
  }

  window.__gosx_workbench_runtime_island_toggleActivity = toggleActivityIsland;

  // saveLayout(form) — mirrors saveWorkbenchLayout at
  // studio-engines.js:294. Persistence wrapper: serializes the form's
  // --studio-left-width / --studio-right-width custom properties and
  // data-studio-activity-state attribute to localStorage under the
  // gosx-studio-editor-layout key (workbenchLayoutStorageKey declared
  // above). Swallows storage errors silently — the layout is a UX
  // convenience, not a correctness invariant.
  function saveLayoutIsland(form) {
    if (!form) return;
    try {
      var leftWidth = form.style.getPropertyValue("--studio-left-width");
      var rightWidth = form.style.getPropertyValue("--studio-right-width");
      var activeGesture = activeRailGestureFor(form);
      if (activeGesture) {
        if (activeGesture.side === "left") leftWidth = activeGesture.startInlineWidth.value;
        else if (activeGesture.side === "right") rightWidth = activeGesture.startInlineWidth.value;
      }
      window.localStorage.setItem(workbenchLayoutStorageKey, JSON.stringify({
        left: leftWidth,
        right: rightWidth,
        activity: form.getAttribute("data-studio-activity-state")
      }));
    } catch (error) {
      return;
    }
  }

  window.__gosx_workbench_runtime_island_saveLayout = saveLayoutIsland;

  // bindCommandPaletteIsland — mirrors bindWorkbenchCommandPalette at
  // studio-engines.js:497. Wires the [data-studio-command-palette]
  // gosxstudio:command listener to the same fan-out (setMode /
  // activateViewport / activateZoom / toggleRail / toggleActivity /
  // toggleFocus). Used internally by bindChromeIsland; not exposed as a
  // public island global because the legacy contract didn't expose it.
  function bindCommandPaletteIsland(form) {
    var disposed = false;
    var node = null;
    var handler = null;

    function clear() {
      if (node && handler) node.removeEventListener("gosxstudio:command", handler);
      if (node) node.removeAttribute("data-gosx-studio-workbench-commands-island-bound");
      node = null;
      handler = null;
    }

    function refresh() {
      if (disposed || !workbenchContains(form)) return;
      var next = form.querySelector("[data-studio-command-palette]");
      if (next === node && handler) return;
      clear();
      if (!next) return;
      node = next;
      node.dataset.gosxStudioWorkbenchCommandsIslandBound = "true";
      handler = function (event) {
        if (disposed || !workbenchContains(form) || !node || event.currentTarget !== node ||
          node.isConnected === false || form.querySelector("[data-studio-command-palette]") !== node) return;
        var detail = event.detail || {};
        var kind = detail.kind || "";
        var target = detail.target || "";
        if (kind === "mode") setModeIsland(form, target, true);
        else if (kind === "viewport") activateViewportIsland(form, target);
        else if (kind === "zoom") activateZoomIsland(form, target);
        else if (kind === "toggle") {
          if (target === "left" || target === "right") toggleRailIsland(form, target);
          else if (target === "activity") toggleActivityIsland(form);
          else if (target === "focus") toggleFocusIsland(form);
        }
      };
      node.addEventListener("gosxstudio:command", handler);
    }

    return {
      refresh: refresh,
      dispose: function () {
        if (disposed) return;
        disposed = true;
        clear();
      }
    };
  }
})();
