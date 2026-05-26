// island_runtime.js — companion JS for the workbenchruntime .gsx islands.
//
// The .gsx islands (workbench_chrome.gsx, workbench_rails.gsx,
// workbench_viewport.gsx, workbench_zoom.gsx, workbench_toggles.gsx,
// workbench_style_state.gsx) are mount-point markers in the editor DOM.
// This script publishes the fifteen
// window.__gosx_workbench_runtime_island_<method> globals that
// workbenchruntime.BridgeShim delegates to when the
// "workbench-runtime-islands" feature flag is on. It is emitted into the
// studio runtime bundle by workbenchruntime.IslandRuntimeJS() (see
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
// Idempotency guards use distinct dataset keys from the legacy
// implementation so both code paths can run on the same DOM during the
// additive shipping window without double-binding:
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
    if (window.requestAnimationFrame) {
      window.requestAnimationFrame(callback);
      return;
    }
    window.setTimeout(callback, 16);
  }

  // frameTask — coalesces calls into a single rAF dispatch with the last
  // arguments (mirrors studio-engines.js:83 helper). Used by saveLayoutSoon.
  function frameTask(callback) {
    var queued = false;
    var lastArgs = null;
    var lastThis = null;
    return function () {
      lastArgs = arguments;
      lastThis = this;
      if (queued) return;
      queued = true;
      frame(function () {
        queued = false;
        callback.apply(lastThis, lastArgs || []);
      });
    };
  }

  // editorWorkbench — resolve the [data-editor-workbench] form (or root if
  // it matches). Mirrors studio-engines.js:69 helper. Used by bindChrome /
  // bindRailResizers.
  function editorWorkbench(root) {
    if (root && root.matches && root.matches("[data-editor-workbench]")) return root;
    return doc.querySelector("[data-editor-workbench]");
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
  function refreshWorkbenchCanvas() {
    frame(function () {
      window.dispatchEvent(new Event("resize"));
    });
  }

  // reducedMotion — preference probe used by setMode's scrollIntoView.
  function reducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
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
    if (mode === "preview") return "publish";
    if (mode === "manage" || mode === "flows") return "advanced";
    return mode || "home";
  }

  function workbenchModePanel(form, mode) {
    var selector = '[data-studio-mode-panel="' + attrValue(normalizeWorkbenchMode(mode)) + '"]';
    return form.querySelector(".studio-right-rail " + selector) || form.querySelector(".editor-panel" + selector) || form.querySelector(selector);
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
  // Idempotency: each handle gets [data-gosx-studio-resizer-island-bound]
  // distinct from the legacy [data-gosx-studio-resizer-bound] so both
  // paths can coexist during the additive shipping window.
  function bindRailResizersIsland(root) {
    var form = editorWorkbench(root);
    var stage = workbenchStage(form);
    if (!form || !stage) return;
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-resizer]"), function (handle) {
      if (handle.dataset.gosxStudioResizerIslandBound === "true") return;
      handle.dataset.gosxStudioResizerIslandBound = "true";
      handle.addEventListener("pointerdown", function (event) {
        if (event.button !== 0) return;
        event.preventDefault();
        var side = handle.getAttribute("data-studio-resizer");
        var rect = stage.getBoundingClientRect();
        handle.classList.add("is-resizing");
        if (handle.setPointerCapture) handle.setPointerCapture(event.pointerId);
        function widthFor(pointerEvent) {
          return side === "left" ? pointerEvent.clientX - rect.left : rect.right - pointerEvent.clientX;
        }
        function move(pointerEvent) {
          setRailWidthIsland(form, side, widthFor(pointerEvent), handle, false);
        }
        function finish(pointerEvent) {
          handle.classList.remove("is-resizing");
          if (pointerEvent && pointerEvent.clientX !== undefined) {
            setRailWidthIsland(form, side, widthFor(pointerEvent), handle, true);
          } else {
            emitWorkbenchRailWidth(form, side, currentRailWidthIsland(form, side, handle), true);
          }
          doc.removeEventListener("pointermove", move);
          doc.removeEventListener("pointerup", finish);
          doc.removeEventListener("pointercancel", finish);
        }
        setRailWidthIsland(form, side, widthFor(event), handle, false);
        doc.addEventListener("pointermove", move);
        doc.addEventListener("pointerup", finish);
        doc.addEventListener("pointercancel", finish);
      });
      handle.addEventListener("keydown", function (event) {
        if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
        event.preventDefault();
        var side = handle.getAttribute("data-studio-resizer");
        var step = event.shiftKey ? 48 : 24;
        var delta = event.key === "ArrowRight" ? step : -step;
        setRailWidthIsland(form, side, currentRailWidthIsland(form, side, handle) + (side === "left" ? delta : -delta), handle, true);
      });
    });
  }

  window.__gosx_workbench_runtime_island_bindRailResizers = bindRailResizersIsland;

  // setRailWidthIsland and currentRailWidthIsland are forward-declared
  // helpers — their public island publications happen at the end of this
  // IIFE (methods 14/15 + 15/15 in the TDD sequence). Function
  // declarations hoist within scope so the forward references above work
  // without warning. Same pattern as slice 5's bindThemeIsland referencing
  // applyThemeIsland before it's published.
  function setRailWidthIsland(form, side, width, handle, committed) {
    if (!form || (side !== "left" && side !== "right")) return;
    var bounds = railBounds(handle, side);
    var next = clampNumber(Math.round(width), bounds.min, bounds.max);
    form.style.setProperty(railWidthProperty(side), next + "px");
    if (handle) handle.setAttribute("aria-valuenow", String(next));
    emitWorkbenchRailWidth(form, side, next, !!committed);
  }

  function currentRailWidthIsland(form, side, handle) {
    if (!form) return railBounds(handle, side).fallback;
    var custom = form.style.getPropertyValue(railWidthProperty(side));
    var parsed = parseInt(custom, 10);
    if (Number.isFinite(parsed)) return parsed;
    var node = railSidebar(form, side);
    if (node) return Math.round(node.getBoundingClientRect().width);
    return railBounds(handle, side).fallback;
  }

  // bindChrome(root) — mirrors bindWorkbenchChrome at studio-engines.js:516.
  // Binds the workbench form's delegated click handler (fans out to
  // setMode / syncViewport / syncZoom / setStyleState / toggleRail /
  // toggleFocus / toggleActivity), wires saveLayoutSoon to
  // rail-width-change/commit events, applies the persisted layout, and
  // seeds the workbench mode / viewport / zoom on initial bind.
  //
  // Idempotency: per-form [data-gosx-studio-workbench-chrome-island-bound]
  // distinct from the legacy [data-gosx-studio-workbench-chrome-bound] so
  // both paths can coexist during the additive shipping window.
  function bindChromeIsland(root) {
    var form = editorWorkbench(root);
    if (!form || form.dataset.gosxStudioWorkbenchChromeIslandBound === "true") return;
    form.dataset.gosxStudioWorkbenchChromeIslandBound = "true";
    var saveLayoutSoon = frameTask(saveLayoutIsland);
    form.addEventListener("click", function (event) {
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
    doc.addEventListener("gosxstudio:rail-width-change", function (event) {
      if (!doc.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      saveLayoutSoon(form);
      refreshWorkbenchCanvas();
    });
    doc.addEventListener("gosxstudio:rail-width-commit", function (event) {
      if (!doc.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      saveLayoutIsland(form);
      refreshWorkbenchCanvas();
    });
    applyWorkbenchLayout(form);
    setModeIsland(form, form.getAttribute("data-studio-mode") || "home", false);
    if (!form.hasAttribute("data-studio-left")) form.setAttribute("data-studio-left", "open");
    if (!form.hasAttribute("data-studio-right")) form.setAttribute("data-studio-right", "open");
    if (!form.hasAttribute("data-studio-focus")) form.setAttribute("data-studio-focus", "false");
    if (!form.hasAttribute("data-studio-activity-state")) form.setAttribute("data-studio-activity-state", "open");
    syncWorkbenchRailButtons(form);
    syncWorkbenchActivityButtons(form);
    bindCommandPaletteIsland(form);
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
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-mode-control]"), function (button) {
      button.setAttribute("aria-pressed", normalizeWorkbenchMode(button.getAttribute("data-studio-mode-control")) === mode ? "true" : "false");
    });
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-mode-panel]"), function (panel) {
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
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-viewport-label]"), function (node) {
      node.textContent = workbenchViewportLabels[viewport] || viewport.charAt(0).toUpperCase() + viewport.slice(1);
    });
    emitWorkbenchChange("viewport-change", form, { viewport: viewport });
    refreshWorkbenchCanvas();
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

  // bindCommandPaletteIsland — mirrors bindWorkbenchCommandPalette at
  // studio-engines.js:497. Wires the [data-studio-command-palette]
  // gosxstudio:command listener to the same fan-out (setMode /
  // activateViewport / activateZoom / toggleRail / toggleActivity /
  // toggleFocus). Used internally by bindChromeIsland; not exposed as a
  // public island global because the legacy contract didn't expose it.
  function bindCommandPaletteIsland(form) {
    var node = form.querySelector("[data-studio-command-palette]");
    if (!node || node.dataset.gosxStudioWorkbenchCommandsIslandBound === "true") return;
    node.dataset.gosxStudioWorkbenchCommandsIslandBound = "true";
    node.addEventListener("gosxstudio:command", function (event) {
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
    });
  }
})();
