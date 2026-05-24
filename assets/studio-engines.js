(function () {
  "use strict";

  var lastStyleImpact = null;

  var studioEngineFactories = {
    GoSXStudioCanvas: function (context) {
      mountStudioEngine(context, "canvas");
      mountCanvasEngine(context);
    },
    GoSXStudioSiteMap: function (context) {
      mountStudioEngine(context, "site-map");
    },
    GoSXStudioFlowDesigner: function (context) {
      mountStudioEngine(context, "flow-designer");
    },
    GoSXStudioBlockLayout: function (context) {
      mountStudioEngine(context, "block-layout");
      mountBlockLayoutEngine(context);
    },
    GoSXStudioShowcase3D: function (context) {
      mountStudioEngine(context, "showcase-3d");
      mountShowcase3DViewer(context);
    }
  };

  function mountStudioEngine(context, role) {
    var mount = context && context.mount;
    if (!mount || mount.dataset.gosxStudioEngineMounted === "true") return;
    mount.dataset.gosxStudioEngineMounted = "true";
    mount.dataset.gosxStudioEngineRole = role;
    mount.dataset.gosxStudioEngineId = context.id || "";
    if (mount.closest(".studio-engine-hosts") || mount.getAttribute("data-studio-engine-hidden") === "true") {
      mount.setAttribute("aria-hidden", "true");
    } else {
      mount.removeAttribute("aria-hidden");
    }
    document.dispatchEvent(new CustomEvent("gosxstudio:engine-mounted", {
      bubbles: true,
      detail: {
        id: context.id || "",
        component: context.component || "",
        role: role,
        props: context.props || {},
        capabilities: context.capabilities || [],
        programRef: context.programRef || ""
      }
    }));
  }

  function mountBlockLayoutEngine(context) {
    var mount = context && context.mount;
    if (mount) mount.dataset.gosxStudioBlockLayoutEngine = "true";
    Array.prototype.forEach.call(document.querySelectorAll("[data-block-studio]"), bindBlockLayoutList);
    bindBlockLayoutLibrary(document);
    bindBlockLayoutVisibility(document);
  }

  function mountCanvasEngine(context) {
    var mount = context && context.mount;
    if (mount) mount.dataset.gosxStudioCanvasEngine = "true";
    var preview = window.GoSXStudioPreviewRuntime;
    if (preview && typeof preview.mount === "function") {
      preview.mount(document);
    }
    bindSelectionSurface(document);
  }

  function editorWorkbench(root) {
    return (root && root.matches && root.matches("[data-editor-workbench]"))
      ? root
      : document.querySelector("[data-editor-workbench]");
  }

  function frame(callback) {
    if (window.requestAnimationFrame) {
      window.requestAnimationFrame(callback);
      return;
    }
    window.setTimeout(callback, 16);
  }

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

  function bindStudioFrameLoad(attr, update) {
    Array.prototype.forEach.call(document.querySelectorAll(".editor-preview-frame"), function (frame) {
      if (frame.getAttribute(attr) === "true") return;
      frame.setAttribute(attr, "true");
      frame.addEventListener("load", function () {
        window.requestAnimationFrame(update);
      });
    });
  }

  function numericValue(input, fallback) {
    if (!input) return fallback;
    var parsed = parseInt(input.value, 10);
    return Number.isFinite(parsed) ? parsed : fallback;
  }

  function numericLimit(input, attr, fallback) {
    if (!input) return fallback;
    var parsed = parseInt(input.getAttribute(attr), 10);
    return Number.isFinite(parsed) ? parsed : fallback;
  }

  function snapValue(value, step) {
    step = Math.max(1, parseInt(step, 10) || 1);
    return Math.round(value / step) * step;
  }

  function setInputNumber(input, value, commit) {
    if (!input) return;
    value = String(Math.round(value));
    if (input.value !== value) {
      input.value = value;
      input.dispatchEvent(new Event("input", { bubbles: true }));
      if (commit) input.dispatchEvent(new Event("change", { bubbles: true }));
    }
  }

  function workbenchStage(form) {
    return form && form.querySelector ? form.querySelector("[data-studio-layout]") : null;
  }

  function clampNumber(value, min, max) {
    return Math.min(Math.max(value, min), max);
  }

  function integerAttr(node, attr, fallback) {
    var parsed = parseInt(node && node.getAttribute(attr), 10);
    return Number.isFinite(parsed) ? parsed : fallback;
  }

  function railBounds(handle, side) {
    return {
      min: integerAttr(handle, "data-studio-rail-min", side === "left" ? 256 : 320),
      max: integerAttr(handle, "data-studio-rail-max", side === "left" ? 448 : 544),
      fallback: integerAttr(handle, "data-studio-rail-default", side === "left" ? 320 : 416)
    };
  }

  function railWidthProperty(side) {
    return side === "left" ? "--studio-left-width" : "--studio-right-width";
  }

  function railSidebar(form, side) {
    return form.querySelector(side === "left" ? "[data-studio-sidebar='left']" : "[data-studio-sidebar='right']");
  }

  function currentWorkbenchRailWidth(form, side, handle) {
    var custom = form.style.getPropertyValue(railWidthProperty(side));
    var parsed = parseInt(custom, 10);
    if (Number.isFinite(parsed)) return parsed;
    var node = railSidebar(form, side);
    if (node) return Math.round(node.getBoundingClientRect().width);
    return railBounds(handle, side).fallback;
  }

  function emitWorkbenchRailWidth(form, side, width, committed) {
    document.dispatchEvent(new CustomEvent(committed ? "gosxstudio:rail-width-commit" : "gosxstudio:rail-width-change", {
      bubbles: true,
      detail: {
        form: form,
        side: side,
        width: width
      }
    }));
  }

  function setWorkbenchRailWidth(form, side, width, handle, committed) {
    if (!form || (side !== "left" && side !== "right")) return;
    var bounds = railBounds(handle, side);
    var next = clampNumber(Math.round(width), bounds.min, bounds.max);
    form.style.setProperty(railWidthProperty(side), next + "px");
    if (handle) handle.setAttribute("aria-valuenow", String(next));
    emitWorkbenchRailWidth(form, side, next, !!committed);
  }

  function bindWorkbenchRailResizers(root) {
    var form = editorWorkbench(root);
    var stage = workbenchStage(form);
    if (!form || !stage) return;
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-resizer]"), function (handle) {
      if (handle.dataset.gosxStudioResizerBound === "true") return;
      handle.dataset.gosxStudioResizerBound = "true";
      handle.addEventListener("pointerdown", function (event) {
        if (event.button !== 0) return;
        event.preventDefault();
        var side = handle.getAttribute("data-studio-resizer");
        var rect = stage.getBoundingClientRect();
        handle.classList.add("is-resizing");
        handle.setPointerCapture && handle.setPointerCapture(event.pointerId);
        function widthFor(pointerEvent) {
          return side === "left" ? pointerEvent.clientX - rect.left : rect.right - pointerEvent.clientX;
        }
        function move(pointerEvent) {
          setWorkbenchRailWidth(form, side, widthFor(pointerEvent), handle, false);
        }
        function finish(pointerEvent) {
          handle.classList.remove("is-resizing");
          if (pointerEvent && pointerEvent.clientX !== undefined) {
            setWorkbenchRailWidth(form, side, widthFor(pointerEvent), handle, true);
          } else {
            emitWorkbenchRailWidth(form, side, currentWorkbenchRailWidth(form, side, handle), true);
          }
          document.removeEventListener("pointermove", move);
          document.removeEventListener("pointerup", finish);
          document.removeEventListener("pointercancel", finish);
        }
        setWorkbenchRailWidth(form, side, widthFor(event), handle, false);
        document.addEventListener("pointermove", move);
        document.addEventListener("pointerup", finish);
        document.addEventListener("pointercancel", finish);
      });
      handle.addEventListener("keydown", function (event) {
        if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
        event.preventDefault();
        var side = handle.getAttribute("data-studio-resizer");
        var step = event.shiftKey ? 48 : 24;
        var delta = event.key === "ArrowRight" ? step : -step;
        setWorkbenchRailWidth(form, side, currentWorkbenchRailWidth(form, side, handle) + (side === "left" ? delta : -delta), handle, true);
      });
    });
  }

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

  function reducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function emitWorkbenchChange(name, form, detail) {
    detail = detail || {};
    detail.form = form;
    document.dispatchEvent(new CustomEvent("gosxstudio:workbench-" + name, {
      bubbles: true,
      detail: detail
    }));
  }

  function refreshWorkbenchCanvas() {
    frame(function () {
      window.dispatchEvent(new Event("resize"));
    });
  }

  function readWorkbenchLayout() {
    try {
      return JSON.parse(window.localStorage.getItem(workbenchLayoutStorageKey) || "{}") || {};
    } catch (error) {
      return {};
    }
  }

  function saveWorkbenchLayout(form) {
    if (!form) return;
    try {
      window.localStorage.setItem(workbenchLayoutStorageKey, JSON.stringify({
        left: form.style.getPropertyValue("--studio-left-width"),
        right: form.style.getPropertyValue("--studio-right-width"),
        activity: form.getAttribute("data-studio-activity-state")
      }));
    } catch (error) {
      return;
    }
  }

  var saveWorkbenchLayoutSoon = frameTask(saveWorkbenchLayout);

  function applyWorkbenchLayout(form) {
    var layout = readWorkbenchLayout();
    if (layout.left) form.style.setProperty("--studio-left-width", layout.left);
    if (layout.right) form.style.setProperty("--studio-right-width", layout.right);
    if (layout.activity) form.setAttribute("data-studio-activity-state", layout.activity);
  }

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

  function setWorkbenchMode(form, mode, scroll) {
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

  function syncWorkbenchViewport(form, viewport) {
    if (!form) return;
    viewport = viewport || "desktop";
    var shell = form.querySelector(".editor-preview-shell");
    if (shell && shell.getAttribute("data-studio-viewport-island") !== "true") shell.setAttribute("data-studio-preview-viewport", viewport);
    form.setAttribute("data-studio-breakpoint", viewport);
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-viewport-label]"), function (node) {
      node.textContent = workbenchViewportLabels[viewport] || viewport.charAt(0).toUpperCase() + viewport.slice(1);
    });
    emitWorkbenchChange("viewport-change", form, { viewport: viewport });
    refreshWorkbenchCanvas();
  }

  function activateWorkbenchViewport(form, viewport) {
    if (!form) return;
    viewport = viewport || "desktop";
    var button = form.querySelector('[data-studio-viewport="' + attrValue(viewport) + '"]');
    if (button && button.click) {
      button.click();
      return;
    }
    syncWorkbenchViewport(form, viewport);
  }

  function currentWorkbenchBreakpoint(form) {
    if (!form) return "desktop";
    var viewport = form.getAttribute("data-studio-breakpoint");
    if (viewport) return viewport;
    var shell = form.querySelector(".editor-preview-shell");
    return shell ? shell.getAttribute("data-studio-preview-viewport") || "desktop" : "desktop";
  }

  function normalizeWorkbenchStyleState(state) {
    return workbenchStyleStateLabels[state] ? state : "default";
  }

  function setWorkbenchStyleState(form, state) {
    if (!form) return;
    state = normalizeWorkbenchStyleState(state || "default");
    form.setAttribute("data-studio-style-state", state);
    Array.prototype.forEach.call(form.querySelectorAll("button[data-studio-style-state], [role='button'][data-studio-style-state]"), function (button) {
      button.setAttribute("aria-pressed", button.getAttribute("data-studio-style-state") === state ? "true" : "false");
    });
    Array.prototype.forEach.call(form.querySelectorAll("[data-studio-style-scope]"), function (scope) {
      scope.setAttribute("data-style-state", state);
      scope.setAttribute("data-style-breakpoint", currentWorkbenchBreakpoint(form));
      scope.setAttribute("data-style-valid", form.getAttribute("data-studio-style-valid") !== "false" ? "true" : "false");
    });
    emitWorkbenchChange("style-state-change", form, { state: state });
    refreshWorkbenchCanvas();
  }

  function syncWorkbenchZoom(form, zoom) {
    if (!form) return;
    zoom = zoom || "fit";
    var canvas = form.querySelector("[data-studio-canvas]");
    if (canvas) canvas.setAttribute("data-studio-canvas-zoom", zoom);
    emitWorkbenchChange("zoom-change", form, { zoom: zoom });
    refreshWorkbenchCanvas();
  }

  function activateWorkbenchZoom(form, zoom) {
    if (!form) return;
    zoom = zoom || "fit";
    var button = form.querySelector('button[data-studio-zoom="' + attrValue(zoom) + '"], [role="button"][data-studio-zoom="' + attrValue(zoom) + '"]');
    if (button && button.click) {
      button.click();
      return;
    }
    syncWorkbenchZoom(form, zoom);
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

  function setWorkbenchActivity(form, state) {
    if (!form) return;
    form.setAttribute("data-studio-activity-state", state === "collapsed" ? "collapsed" : "open");
    syncWorkbenchActivityButtons(form);
    saveWorkbenchLayout(form);
    emitWorkbenchChange("activity-change", form, { state: workbenchActivityState(form) });
    refreshWorkbenchCanvas();
  }

  function toggleWorkbenchActivity(form) {
    setWorkbenchActivity(form, workbenchActivityState(form) === "open" ? "collapsed" : "open");
  }

  function toggleWorkbenchRail(form, side) {
    if (!form || (side !== "left" && side !== "right")) return;
    form.setAttribute("data-studio-focus", "false");
    form.setAttribute("data-studio-" + side, workbenchRailState(form, side) === "open" ? "collapsed" : "open");
    syncWorkbenchRailButtons(form);
    emitWorkbenchChange("rail-change", form, { side: side, state: workbenchRailState(form, side) });
    refreshWorkbenchCanvas();
  }

  function toggleWorkbenchFocus(form) {
    if (!form) return;
    form.setAttribute("data-studio-focus", form.getAttribute("data-studio-focus") === "true" ? "false" : "true");
    syncWorkbenchRailButtons(form);
    emitWorkbenchChange("focus-change", form, { focus: form.getAttribute("data-studio-focus") === "true" });
    refreshWorkbenchCanvas();
  }

  function bindWorkbenchCommandPalette(form) {
    var node = form.querySelector("[data-studio-command-palette]");
    if (!node || node.dataset.gosxStudioWorkbenchCommandsBound === "true") return;
    node.dataset.gosxStudioWorkbenchCommandsBound = "true";
    node.addEventListener("gosxstudio:command", function (event) {
      var detail = event.detail || {};
      var kind = detail.kind || "";
      var target = detail.target || "";
      if (kind === "mode") setWorkbenchMode(form, target, true);
      else if (kind === "viewport") activateWorkbenchViewport(form, target);
      else if (kind === "zoom") activateWorkbenchZoom(form, target);
      else if (kind === "toggle") {
        if (target === "left" || target === "right") toggleWorkbenchRail(form, target);
        else if (target === "activity") toggleWorkbenchActivity(form);
        else if (target === "focus") toggleWorkbenchFocus(form);
      }
    });
  }

  function bindWorkbenchChrome(root) {
    var form = editorWorkbench(root);
    if (!form || form.dataset.gosxStudioWorkbenchChromeBound === "true") return;
    form.dataset.gosxStudioWorkbenchChromeBound = "true";
    form.addEventListener("click", function (event) {
      var mode = event.target.closest("[data-studio-mode-control]");
      if (mode && form.contains(mode)) {
        event.preventDefault();
        setWorkbenchMode(form, mode.getAttribute("data-studio-mode-control"), true);
        return;
      }
      var viewport = event.target.closest("[data-studio-viewport]");
      if (viewport && form.contains(viewport)) {
        event.preventDefault();
        syncWorkbenchViewport(form, viewport.getAttribute("data-studio-viewport"));
        return;
      }
      var zoom = event.target.closest("button[data-studio-zoom], [role='button'][data-studio-zoom]");
      if (zoom && form.contains(zoom)) {
        event.preventDefault();
        syncWorkbenchZoom(form, zoom.getAttribute("data-studio-zoom"));
        return;
      }
      var styleState = event.target.closest("button[data-studio-style-state], [role='button'][data-studio-style-state]");
      if (styleState && form.contains(styleState)) {
        event.preventDefault();
        setWorkbenchStyleState(form, styleState.getAttribute("data-studio-style-state"));
        return;
      }
      var rail = event.target.closest("[data-studio-rail-toggle]");
      if (rail && form.contains(rail)) {
        event.preventDefault();
        toggleWorkbenchRail(form, rail.getAttribute("data-studio-rail-toggle"));
        return;
      }
      var focus = event.target.closest("[data-studio-focus-toggle]");
      if (focus && form.contains(focus)) {
        event.preventDefault();
        toggleWorkbenchFocus(form);
        return;
      }
      var activity = event.target.closest("[data-studio-activity-toggle]");
      if (activity && form.contains(activity)) {
        event.preventDefault();
        toggleWorkbenchActivity(form);
      }
    });
    document.addEventListener("gosxstudio:rail-width-change", function (event) {
      if (!document.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      saveWorkbenchLayoutSoon(form);
      refreshWorkbenchCanvas();
    });
    document.addEventListener("gosxstudio:rail-width-commit", function (event) {
      if (!document.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      saveWorkbenchLayout(form);
      refreshWorkbenchCanvas();
    });
    applyWorkbenchLayout(form);
    setWorkbenchMode(form, form.getAttribute("data-studio-mode") || "home", false);
    if (!form.hasAttribute("data-studio-left")) form.setAttribute("data-studio-left", "open");
    if (!form.hasAttribute("data-studio-right")) form.setAttribute("data-studio-right", "open");
    if (!form.hasAttribute("data-studio-focus")) form.setAttribute("data-studio-focus", "false");
    if (!form.hasAttribute("data-studio-activity-state")) form.setAttribute("data-studio-activity-state", "open");
    syncWorkbenchRailButtons(form);
    syncWorkbenchActivityButtons(form);
    bindWorkbenchCommandPalette(form);
    syncWorkbenchViewport(form, "desktop");
    syncWorkbenchZoom(form, form.querySelector("[data-studio-canvas]") ? form.querySelector("[data-studio-canvas]").getAttribute("data-studio-canvas-zoom") || "fit" : "fit");
  }

  function previewRuntime() {
    return window.GoSXStudioPreviewRuntime || {};
  }

  function bindFieldMirroring(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var inputs = scope.querySelectorAll("[data-editor-source], [data-editor-frame-attr-target]");
    Array.prototype.forEach.call(inputs, function (input) {
      if (input.dataset.gosxStudioFieldMirrorBound === "true") return;
      input.dataset.gosxStudioFieldMirrorBound = "true";
      var key = input.getAttribute("data-editor-source");
      var frameTarget = input.getAttribute("data-editor-frame-target");
      var attrTarget = input.getAttribute("data-editor-frame-attr-target");
      var attrName = input.getAttribute("data-editor-frame-attr");
      var attrPrefix = input.getAttribute("data-editor-frame-attr-prefix") || "";
      var attrSuffix = input.getAttribute("data-editor-frame-attr-suffix") || "";
      var updateNow = function () {
        var preview = previewRuntime();
        if (preview.applyTextUpdate) {
          preview.applyTextUpdate({
            sourceKey: key,
            frameTarget: frameTarget,
            attrTarget: attrTarget,
            attrName: attrName,
            attrPrefix: attrPrefix,
            attrSuffix: attrSuffix,
            value: input.value
          });
        }
      };
      var update = frameTask(updateNow);
      input.addEventListener("input", update);
      if (frameTarget || attrTarget) {
        var loadKey = String(key || input.name || attrName || "field").replace(/[^a-z0-9_-]/gi, "-").toLowerCase();
        bindStudioFrameLoad("data-editor-frame-text-" + loadKey, update);
      }
      updateNow();
    });
  }

  function bindFieldClipboard(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var marker = document.documentElement;
    if (marker.dataset.gosxStudioFieldClipboardBound === "true") return;
    marker.dataset.gosxStudioFieldClipboardBound = "true";

    function setCopyState(button, state) {
      if (!button) return;
      var previous = button.getAttribute("data-studio-copy-label") || button.textContent || "Copy";
      button.setAttribute("data-studio-copy-label", previous);
      button.setAttribute("data-studio-copy-state", state);
      button.textContent = state === "ok" ? "Copied" : state === "error" ? "Copy failed" : previous;
      window.setTimeout(function () {
        if (!document.contains(button)) return;
        button.setAttribute("data-studio-copy-state", "idle");
        button.textContent = button.getAttribute("data-studio-copy-label") || previous;
      }, 1600);
    }

    function fallbackCopy(value, target) {
      var input = target && "select" in target ? target : null;
      if (!input) {
        input = document.createElement("textarea");
        input.value = value;
        input.setAttribute("readonly", "readonly");
        input.style.position = "fixed";
        input.style.left = "-9999px";
        document.body.appendChild(input);
      }
      input.focus();
      input.select();
      var ok = false;
      try {
        ok = document.execCommand("copy");
      } catch (error) {
        ok = false;
      }
      if (input.parentNode === document.body && target !== input) document.body.removeChild(input);
      return ok;
    }

    document.addEventListener("click", function (event) {
      var button = event.target.closest("[data-studio-copy-target]");
      if (!button || !scope.contains(button)) return;
      event.preventDefault();
      var selector = button.getAttribute("data-studio-copy-target");
      var target = selector ? document.querySelector(selector) : null;
      var value = target ? (target.value || target.textContent || "") : "";
      if (!value) {
        setCopyState(button, "error");
        return;
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(value).then(function () {
          setCopyState(button, "ok");
        }, function () {
          setCopyState(button, fallbackCopy(value, target) ? "ok" : "error");
        });
      } else {
        setCopyState(button, fallbackCopy(value, target) ? "ok" : "error");
      }
    });
  }

  function bindFieldRuntime(root) {
    bindFieldMirroring(root);
    bindFieldClipboard(root);
  }

  function selectionWorkbenchRuntime() {
    return window.GoSXStudioWorkbenchRuntime || {};
  }

  function selectionReducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function selectionBlockRow(root, key) {
    var runtime = window.GoSXStudioBlockLayoutRuntime;
    if (runtime && typeof runtime.rowForKey === "function") {
      return runtime.rowForKey(root || document, key);
    }
    return blockRowForKey(root || document, key);
  }

  function bindSelectionSurface(root) {
    var form = editorWorkbench(root);
    if (!form || form.dataset.gosxStudioSelectionBound === "true") return;
    form.dataset.gosxStudioSelectionBound = "true";
    var viewportLabels = {
      desktop: "Desktop",
      tablet: "Tablet",
      mobile: "Mobile"
    };
    var styleStateLabels = {
      default: "Default",
      hover: "Hover",
      focus: "Focus"
    };
    var refreshCanvas = frameTask(function () {
      window.dispatchEvent(new Event("resize"));
    });

    function setReadout(selector, value) {
      Array.prototype.forEach.call(form.querySelectorAll(selector), function (node) {
        node.textContent = value;
      });
    }

    function setMode(mode, scroll) {
      var runtime = selectionWorkbenchRuntime();
      if (runtime.setMode) runtime.setMode(form, mode, !!scroll);
    }

    function currentBreakpoint() {
      var runtime = selectionWorkbenchRuntime();
      if (runtime.currentBreakpoint) return runtime.currentBreakpoint(form);
      var viewport = form.getAttribute("data-studio-breakpoint");
      if (viewport) return viewport;
      var shell = form.querySelector(".editor-preview-shell");
      return shell ? shell.getAttribute("data-studio-preview-viewport") || "desktop" : "desktop";
    }

    function normalizeStyleState(state) {
      return styleStateLabels[state] ? state : "default";
    }

    function selectedRow() {
      var key = form.getAttribute("data-studio-selection");
      var row = key ? selectionBlockRow(form, key) : null;
      if (row) return row;
      if (key && form.getAttribute("data-studio-workspace-selection") === key) return null;
      row = form.querySelector("[data-block-studio-block].is-selected");
      if (row) return row;
      return form.querySelector("[data-block-studio-block]");
    }

    function selectedKey() {
      var row = selectedRow();
      return row ? row.getAttribute("data-block-studio-block") || "" : "";
    }

    function selectionIsVisible(row) {
      var checkbox = row && row.querySelector("[data-editor-block-visible]");
      return !checkbox || checkbox.checked;
    }

    function updateSelectionStatus(row, label) {
      var hasSelection = !!row || !!label;
      var visible = row ? selectionIsVisible(row) : false;
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-selection-status]"), function (status) {
        status.textContent = label || (hasSelection ? (visible ? "Visible" : "Hidden") : "No selection");
        status.classList.toggle("is-hidden", !!row && !visible);
      });
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-selection-action]"), function (button) {
        button.disabled = !row;
        if (button.getAttribute("data-studio-selection-action") === "toggle-visibility") {
          button.textContent = row && visible ? "Hide" : "Show";
        }
      });
    }

    function fieldLabel(field) {
      var value = String(field || "block").split(".").pop();
      value = value.replace(/([a-z])([A-Z])/g, "$1 $2").replace(/[-_]/g, " ");
      return value.charAt(0).toUpperCase() + value.slice(1);
    }

    function updateStyleScope() {
      var row = selectedRow();
      var blockLabel = row ? row.getAttribute("data-studio-block-label") || row.getAttribute("data-block-studio-block") || "Block" : "No selection";
      var field = form.getAttribute("data-studio-field-selection") || "";
      var state = normalizeStyleState(form.getAttribute("data-studio-style-state") || "default");
      var breakpoint = currentBreakpoint();
      var valid = form.getAttribute("data-studio-style-valid") !== "false";
      var system = form.getAttribute("data-studio-style-system") || "Local";
      form.setAttribute("data-studio-style-state", state);
      setReadout("[data-studio-style-scope-block]", blockLabel);
      setReadout("[data-studio-style-scope-field]", field ? fieldLabel(field) : "Block");
      setReadout("[data-studio-style-breakpoint-label]", viewportLabels[breakpoint] || fieldLabel(breakpoint));
      setReadout("[data-studio-style-state-label]", styleStateLabels[state]);
      setReadout("[data-studio-style-validity]", valid ? "Ready" : "Needs review");
      setReadout("[data-studio-style-system-label]", system);
      Array.prototype.forEach.call(form.querySelectorAll("button[data-studio-style-state], [role='button'][data-studio-style-state]"), function (button) {
        button.setAttribute("aria-pressed", button.getAttribute("data-studio-style-state") === state ? "true" : "false");
      });
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-style-scope]"), function (scope) {
        scope.setAttribute("data-style-state", state);
        scope.setAttribute("data-style-breakpoint", breakpoint);
        scope.setAttribute("data-style-valid", valid ? "true" : "false");
      });
    }

    function inspectorMatchesSelection(scope, key) {
      scope = String(scope || "").trim();
      if (!scope) return true;
      if (!key) return false;
      return scope.split(/[\s,]+/).filter(Boolean).indexOf(key) >= 0;
    }

    function setSelectionReadout(label) {
      label = label || "No selection";
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-selection-label]"), function (readout) {
        readout.textContent = label;
      });
    }

    function syncInspectorForSelection(key) {
      key = key || selectedKey();
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-inspector-for]"), function (node) {
        var visible = inspectorMatchesSelection(node.getAttribute("data-studio-inspector-for"), key);
        node.hidden = !visible;
        node.classList.toggle("is-inspector-hidden", !visible);
      });
    }

    function workspaceScopeForKey(key) {
      key = String(key || "").trim();
      if (!key || key === "home") return "";
      if (key.indexOf("tool-") === 0) return key;
      return "resource-" + key;
    }

    function workspaceTargetKey(target) {
      if (!target) return "";
      return target.getAttribute("data-studio-panel-link") || target.getAttribute("data-studio-site-page") || "";
    }

    function workspaceTargetLabel(target, key) {
      var label = target && target.querySelector ? target.querySelector("strong, span") : null;
      label = label ? label.textContent : "";
      if (label && label.trim()) return label.trim();
      return ownerLabel(String(key || "").replace(/^tool-/, ""));
    }

    function workspaceInspectorNode(scope) {
      var match = null;
      Array.prototype.some.call(form.querySelectorAll("[data-studio-inspector-for]"), function (node) {
        if (!inspectorMatchesSelection(node.getAttribute("data-studio-inspector-for"), scope)) return false;
        match = node;
        return true;
      });
      return match;
    }

    function firstWorkspaceTarget() {
      var match = null;
      Array.prototype.some.call(form.querySelectorAll("[data-studio-panel-link], [data-studio-site-page]"), function (link) {
        var scope = workspaceScopeForKey(workspaceTargetKey(link));
        if (!scope || !workspaceInspectorNode(scope)) return false;
        match = link;
        return true;
      });
      return match;
    }

    function updateWorkspaceSelectionLinks(scope) {
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-panel-link], [data-studio-site-page]"), function (link) {
        var linkScope = workspaceScopeForKey(workspaceTargetKey(link));
        link.classList.toggle("is-selected", !!scope && linkScope === scope);
        if (linkScope && linkScope === scope) link.setAttribute("aria-current", "true");
        else link.removeAttribute("aria-current");
      });
    }

    function syncHomeLayerPicker(key) {
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-home-layer-pick]"), function (button) {
        button.setAttribute("aria-pressed", button.getAttribute("data-studio-home-layer-pick") === key ? "true" : "false");
      });
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-home-layer-selected]"), function (picker) {
        picker.setAttribute("data-studio-home-layer-selected", key || "");
      });
    }

    function clearFieldFocus() {
      form.removeAttribute("data-studio-field-selection");
      form.removeAttribute("data-studio-field-editable");
      form.removeAttribute("data-studio-field-action-label");
      form.removeAttribute("data-studio-field-action-href");
      Array.prototype.forEach.call(form.querySelectorAll(".is-studio-field-active"), function (row) {
        row.classList.remove("is-studio-field-active");
      });
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-field-selection-label]"), function (readout) {
        readout.textContent = "Block";
      });
      updateFieldActionLabels();
      updateStyleScope();
    }

    function setWorkspaceSelection(target, reveal) {
      var key = workspaceTargetKey(target);
      var scope = workspaceScopeForKey(key);
      if (!scope || !workspaceInspectorNode(scope)) return false;
      var label = workspaceTargetLabel(target, key);
      form.setAttribute("data-studio-selection", scope);
      form.setAttribute("data-studio-workspace-selection", scope);
      form.setAttribute("data-studio-selection-kind", "workspace");
      Array.prototype.forEach.call(form.querySelectorAll("[data-block-studio-block]"), function (item) {
        item.classList.remove("is-selected");
      });
      setSelectionReadout(label);
      updateSelectionStatus(null, "Surface");
      syncInspectorForSelection(scope);
      clearFieldFocus();
      setSelectionReadout(label);
      updateWorkspaceSelectionLinks(scope);
      syncHomeLayerPicker("");
      if (reveal) {
        setMode("advanced", true);
        var node = workspaceInspectorNode(scope);
        if (node) node.scrollIntoView({ behavior: selectionReducedMotion() ? "auto" : "smooth", block: "center" });
      }
      updateStyleScope();
      return true;
    }

    function ensureWorkspaceSelection() {
      var scope = form.getAttribute("data-studio-workspace-selection") || "";
      if (scope && workspaceInspectorNode(scope)) {
        syncInspectorForSelection(scope);
        updateWorkspaceSelectionLinks(scope);
        return;
      }
      var target = firstWorkspaceTarget();
      if (target) setWorkspaceSelection(target, false);
    }

    function workspaceTargetFromEvent(event) {
      var target = event.target && event.target.closest
        ? event.target.closest("[data-studio-panel-link], [data-studio-site-page]")
        : null;
      return target && form.contains(target) ? target : null;
    }

    function updateFieldActionLabels() {
      var field = form.getAttribute("data-studio-field-selection") || "";
      var editable = form.getAttribute("data-studio-field-editable") || "";
      var explicit = form.getAttribute("data-studio-field-action-label") || "";
      Array.prototype.forEach.call(form.querySelectorAll('[data-studio-selection-action="inline-text"]'), function (button) {
        if (!field) {
          button.textContent = "Edit";
          button.disabled = !selectedRow();
          return;
        }
        button.textContent = explicit || (editable === "media" ? "Manage media" :
          editable === "source" ? "Manage source" :
          editable === "flow" ? "Configure flow" :
          editable === "url" || editable === "link" ? "Edit URL" :
          editable === "text" ? "Edit text" : "Open field");
      });
    }

    function fieldSource(field) {
      if (!field) return null;
      return form.querySelector('[data-studio-field-source="' + attrValue(field) + '"]');
    }

    function fieldControl(source) {
      if (!source) return null;
      if (source.matches("input, textarea, select, button, a, [tabindex]")) return source;
      return source.querySelector("input, textarea, select, button, a[href], [tabindex]") || source;
    }

    function fieldActionTarget(field) {
      var source = fieldSource(field);
      if (!source) return { source: null, control: null, href: "" };
      var control = fieldControl(source);
      var href = source.getAttribute("data-studio-field-action-href") || "";
      if (!href) {
        var link = source.querySelector("a[href]");
        if (link) href = link.getAttribute("href") || "";
      }
      return { source: source, control: control, href: href };
    }

    function inferEditableKind(source, control) {
      if (!source) return "";
      var explicit = source.getAttribute("data-studio-field-editable") || "";
      if (explicit) return explicit;
      var tag = control && control.tagName ? control.tagName.toLowerCase() : "";
      var type = control && control.type ? String(control.type).toLowerCase() : "";
      if (tag === "textarea") return "text";
      if (tag === "input" && (type === "url" || type === "email")) return "url";
      if (tag === "input") return "text";
      return "field";
    }

    function applyFieldActionMetadata(source, detail) {
      if (!source) return;
      var actionLabel = detail.action || source.getAttribute("data-studio-field-action") || "";
      var actionHref = source.getAttribute("data-studio-field-action-href") || "";
      if (actionLabel) form.setAttribute("data-studio-field-action-label", actionLabel);
      if (actionHref) form.setAttribute("data-studio-field-action-href", actionHref);
    }

    function revealFieldControl(source, control) {
      var reduced = selectionReducedMotion();
      setMode("home", true);
      source.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });
      window.setTimeout(function () {
        if (control && control.focus) control.focus({ preventScroll: true });
      }, reduced ? 0 : 120);
    }

    function runSelectedFieldAction(field, editable) {
      var target = fieldActionTarget(field);
      if (!target.source) return false;
      if (target.href && editable !== "text" && editable !== "url" && editable !== "link") {
        window.location.href = target.href;
        return true;
      }
      revealFieldControl(target.source, target.control);
      return true;
    }

    function setFieldFocus(detail, reveal) {
      detail = detail || {};
      var field = detail.field || "";
      var label = detail.label || fieldLabel(field);
      var editable = detail.editable || "";
      clearFieldFocus();
      if (!field) return;
      form.setAttribute("data-studio-field-selection", field);
      form.setAttribute("data-studio-field-editable", editable);
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-field-selection-label]"), function (readout) {
        readout.textContent = label;
      });
      var source = fieldSource(field);
      var control = fieldControl(source);
      if (!editable) {
        editable = inferEditableKind(source, control);
        form.setAttribute("data-studio-field-editable", editable);
      }
      applyFieldActionMetadata(source, detail);
      var row = source && source.closest(".field-row");
      if (row) row.classList.add("is-studio-field-active");
      if (reveal && source) revealFieldControl(source, control);
      updateFieldActionLabels();
      updateStyleScope();
    }

    function selectRow(row) {
      if (!row) return;
      Array.prototype.forEach.call(form.querySelectorAll("[data-block-studio-block]"), function (item) {
        item.classList.toggle("is-selected", item === row);
      });
      updateSelection(row.getAttribute("data-block-studio-block") || "");
      document.dispatchEvent(new CustomEvent("blockstudio:select", {
        bubbles: true,
        detail: { key: row.getAttribute("data-block-studio-block") || "" }
      }));
    }

    function revealSelection(row) {
      if (!row) return;
      var reduced = selectionReducedMotion();
      setMode("home", true);
      selectRow(row);
      row.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });
      window.setTimeout(function () {
        if (row.focus) row.focus({ preventScroll: true });
      }, reduced ? 0 : 120);
    }

    function runSelectionAction(action) {
      var row = selectedRow();
      if (!row) return;
      var key = row.getAttribute("data-block-studio-block") || "";
      if (action === "reveal") {
        revealSelection(row);
        return;
      }
      if (action === "content") {
        setMode("home", true);
        selectRow(row);
        return;
      }
      if (action === "style") {
        setMode("look", true);
        selectRow(row);
        return;
      }
      if (action === "inline-text") {
        setMode("home", true);
        selectRow(row);
        var field = form.getAttribute("data-studio-field-selection") || "";
        var editable = form.getAttribute("data-studio-field-editable") || "";
        if (field && editable && editable !== "text") {
          if (runSelectedFieldAction(field, editable)) return;
          setFieldFocus({ field: field, editable: editable }, true);
          return;
        }
        var preview = previewRuntime();
        if (preview.requestInlineEdit) preview.requestInlineEdit({ key: key, field: field });
        return;
      }
      if (action === "previous-field" || action === "next-field") {
        selectRow(row);
        var fieldPreview = previewRuntime();
        if (fieldPreview.cycleField) fieldPreview.cycleField({ key: key, direction: action === "previous-field" ? -1 : 1 });
        return;
      }
      if (action === "toggle-visibility") {
        var checkbox = row.querySelector("[data-editor-block-visible]");
        if (!checkbox) return;
        checkbox.checked = !checkbox.checked;
        checkbox.dispatchEvent(new Event("input", { bubbles: true }));
        checkbox.dispatchEvent(new Event("change", { bubbles: true }));
        selectRow(row);
        refreshCanvas();
      }
    }

    function bindCommandPaletteCommands() {
      var node = form.querySelector("[data-studio-command-palette]");
      if (!node || node.dataset.gosxStudioSelectionCommandsBound === "true") return;
      node.dataset.gosxStudioSelectionCommandsBound = "true";
      node.addEventListener("gosxstudio:command", function (event) {
        var detail = event.detail || {};
        var kind = detail.kind || "";
        var target = detail.target || "";
        if (kind === "selection-action") runSelectionAction(target);
      });
    }

    function updateSelection(key) {
      var row = key ? selectionBlockRow(form, key) : selectedRow();
      if (!row) row = form.querySelector("[data-block-studio-block]");
      if (!row) {
        form.removeAttribute("data-studio-selection");
        form.removeAttribute("data-studio-workspace-selection");
        form.removeAttribute("data-studio-selection-kind");
        setSelectionReadout("No selection");
        updateSelectionStatus(null);
        updateWorkspaceSelectionLinks("");
        syncHomeLayerPicker("");
        clearFieldFocus();
        return;
      }
      var label = row.getAttribute("data-studio-block-label") || row.getAttribute("data-block-studio-block") || "Selection";
      form.setAttribute("data-studio-selection", row.getAttribute("data-block-studio-block") || "");
      form.removeAttribute("data-studio-workspace-selection");
      form.setAttribute("data-studio-selection-kind", "block");
      Array.prototype.forEach.call(form.querySelectorAll("[data-block-studio-block]"), function (item) {
        item.classList.toggle("is-selected", item === row);
      });
      setSelectionReadout(label);
      updateWorkspaceSelectionLinks("");
      syncHomeLayerPicker(row.getAttribute("data-block-studio-block") || "");
      updateSelectionStatus(row);
      syncInspectorForSelection(row.getAttribute("data-block-studio-block") || "");
      clearFieldFocus();
      updateStyleScope();
    }

    form.addEventListener("click", function (event) {
      var selectionAction = event.target.closest("[data-studio-selection-action]");
      if (selectionAction && form.contains(selectionAction)) {
        event.preventDefault();
        runSelectionAction(selectionAction.getAttribute("data-studio-selection-action"));
        return;
      }
      var workspaceTarget = workspaceTargetFromEvent(event);
      if (workspaceTarget && setWorkspaceSelection(workspaceTarget, true)) {
        if (!event.metaKey && !event.ctrlKey && !event.shiftKey) event.preventDefault();
        return;
      }
      var layerPick = event.target.closest("[data-studio-home-layer-pick]");
      if (layerPick && form.contains(layerPick)) {
        event.preventDefault();
        var layerKey = layerPick.getAttribute("data-studio-home-layer-pick") || "";
        var layerRow = selectionBlockRow(form, layerKey);
        if (layerRow) {
          updateSelection(layerKey);
          layerRow.scrollIntoView({ behavior: selectionReducedMotion() ? "auto" : "smooth", block: "center" });
        }
        return;
      }
    });

    form.addEventListener("click", function (event) {
      var row = event.target.closest("[data-block-studio-block]");
      if (row && form.contains(row)) updateSelection(row.getAttribute("data-block-studio-block"));
    });
    form.addEventListener("focusin", function (event) {
      var row = event.target.closest("[data-block-studio-block]");
      if (row && form.contains(row)) updateSelection(row.getAttribute("data-block-studio-block"));
      var workspaceTarget = workspaceTargetFromEvent(event);
      if (workspaceTarget) setWorkspaceSelection(workspaceTarget, false);
    });
    document.addEventListener("blockstudio:select", function (event) {
      if (document.contains(form)) updateSelection(event.detail && event.detail.key);
    });
    document.addEventListener("studio:field-select", function (event) {
      if (!document.contains(form)) return;
      setFieldFocus(event.detail, event.detail && event.detail.source !== "preview");
    });
    document.addEventListener("studio:inline-edit-start", function (event) {
      if (!document.contains(form)) return;
      setFieldFocus(event.detail, false);
    });
    document.addEventListener("gosxstudio:workbench-mode-change", function (event) {
      if (!document.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      var mode = event.detail && event.detail.mode || form.getAttribute("data-studio-mode") || "";
      if (mode === "advanced") ensureWorkspaceSelection();
      else if (form.getAttribute("data-studio-workspace-selection")) updateSelection("");
      updateStyleScope();
    });
    document.addEventListener("gosxstudio:workbench-viewport-change", function (event) {
      if (!document.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      updateStyleScope();
    });
    document.addEventListener("gosxstudio:workbench-style-state-change", function (event) {
      if (!document.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      updateStyleScope();
    });
    form.addEventListener("focusin", function (event) {
      var input = event.target && event.target.closest && event.target.closest("[data-studio-field-source]");
      if (input && form.contains(input)) {
        setFieldFocus({ field: input.getAttribute("data-studio-field-source") || "" }, false);
      }
    });
    form.addEventListener("change", function (event) {
      if (event.target && event.target.closest && event.target.closest("[data-editor-block-visible]")) {
        updateSelection(selectedKey());
      }
    });

    bindCommandPaletteCommands();
    if (form.getAttribute("data-studio-mode") === "advanced") ensureWorkspaceSelection();
    else updateSelection("");
  }

  function updateHeaderLogo(payload) {
    var preview = previewRuntime();
    if (preview.updateHeaderLogo) preview.updateHeaderLogo(payload);
  }

  function bindBrandLogo(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var preview = scope.querySelector("[data-editor-brand-preview]");
    var logo = scope.querySelector("[data-editor-brand-logo]");
    var handle = scope.querySelector("[data-editor-brand-handle]");
    var urlInput = scope.querySelector("[data-editor-logo-url]");
    var altInput = scope.querySelector("[data-editor-logo-alt]");
    var widthInput = scope.querySelector("[data-editor-logo-width]");
    var xInput = scope.querySelector("[data-editor-logo-offset-x]");
    var yInput = scope.querySelector("[data-editor-logo-offset-y]");
    var snapInput = scope.querySelector("[data-editor-logo-snap]");
    var snapSizeInput = scope.querySelector("[data-editor-logo-snap-size]");
    var resetButton = scope.querySelector("[data-editor-logo-reset]");
    var readout = scope.querySelector("[data-editor-logo-readout]");
    if (!preview || !logo || !handle || !urlInput) return;

    function gridStep() {
      return clampNumber(numericValue(snapSizeInput, 8), 1, 32);
    }

    function snapped(value, bypassSnap) {
      if (bypassSnap || !snapInput || !snapInput.checked) return Math.round(value);
      return snapValue(value, gridStep());
    }

    function clampFor(input, value, fallbackMin, fallbackMax) {
      return clampNumber(value, numericLimit(input, "min", fallbackMin), numericLimit(input, "max", fallbackMax));
    }

    function applyPosition(x, y, commit, bypassSnap) {
      setInputNumber(xInput, clampFor(xInput, snapped(x, bypassSnap), -80, 320), commit);
      setInputNumber(yInput, clampFor(yInput, snapped(y, bypassSnap), -80, 320), commit);
      update();
    }

    var updateNow = function () {
      logo.setAttribute("src", urlInput.value || "");
      logo.setAttribute("alt", (altInput && altInput.value) || "Site logo");
      var width = widthInput && widthInput.value ? widthInput.value : "96";
      var x = xInput && xInput.value ? xInput.value : "0";
      var y = yInput && yInput.value ? yInput.value : "0";
      preview.style.setProperty("--brand-logo-width", width + "px");
      preview.style.setProperty("--brand-logo-offset-x", x + "px");
      preview.style.setProperty("--brand-logo-offset-y", y + "px");
      preview.style.setProperty("--editor-logo-grid-size", gridStep() + "px");
      handle.classList.toggle("is-empty", !urlInput.value);
      if (readout) readout.textContent = "X " + x + " / Y " + y + " / W " + width;
      updateHeaderLogo({ url: urlInput.value, width: width, x: x, y: y, alt: (altInput && altInput.value) || "Site logo" });
    };
    var update = frameTask(updateNow);

    [urlInput, altInput, widthInput, xInput, yInput, snapInput, snapSizeInput].forEach(function (input) {
      if (!input || input.dataset.gosxStudioBrandLogoBound === "true") return;
      input.dataset.gosxStudioBrandLogoBound = "true";
      input.addEventListener("input", function () {
        if (input === snapInput || input === snapSizeInput) applyPosition(numericValue(xInput, 0), numericValue(yInput, 0), false, false);
        else update();
      });
      input.addEventListener("change", function () {
        if (input === snapInput || input === snapSizeInput) applyPosition(numericValue(xInput, 0), numericValue(yInput, 0), true, false);
        else update();
      });
    });

    if (resetButton && resetButton.dataset.gosxStudioBrandLogoBound !== "true") {
      resetButton.dataset.gosxStudioBrandLogoBound = "true";
      resetButton.addEventListener("click", function () {
        applyPosition(0, 0, true, true);
        handle.focus();
      });
    }

    if (handle.dataset.gosxStudioBrandLogoBound !== "true") {
      handle.dataset.gosxStudioBrandLogoBound = "true";
      var drag = null;
      handle.addEventListener("pointerdown", function (event) {
        if (event.button !== undefined && event.button !== 0) return;
        if (!urlInput.value) return;
        drag = {
          pointerId: event.pointerId,
          startX: event.clientX,
          startY: event.clientY,
          valueX: numericValue(xInput, 0),
          valueY: numericValue(yInput, 0)
        };
        handle.classList.add("is-dragging");
        handle.setPointerCapture(event.pointerId);
        event.preventDefault();
      });
      handle.addEventListener("pointermove", function (event) {
        if (!drag || drag.pointerId !== event.pointerId) return;
        applyPosition(drag.valueX + event.clientX - drag.startX, drag.valueY + event.clientY - drag.startY, false, event.altKey);
        event.preventDefault();
      });
      handle.addEventListener("pointerup", function (event) {
        if (!drag || drag.pointerId !== event.pointerId) return;
        handle.classList.remove("is-dragging");
        applyPosition(numericValue(xInput, 0), numericValue(yInput, 0), true, event.altKey);
        drag = null;
      });
      handle.addEventListener("pointercancel", function () {
        handle.classList.remove("is-dragging");
        drag = null;
      });
      handle.addEventListener("keydown", function (event) {
        var x = numericValue(xInput, 0);
        var y = numericValue(yInput, 0);
        var step = event.altKey ? 1 : gridStep();
        if (event.shiftKey) step = step * 4;
        if (event.key === "ArrowLeft") x -= step;
        else if (event.key === "ArrowRight") x += step;
        else if (event.key === "ArrowUp") y -= step;
        else if (event.key === "ArrowDown") y += step;
        else if (event.key === "Home") {
          x = 0;
          y = 0;
        } else {
          return;
        }
        applyPosition(x, y, true, event.altKey);
        event.preventDefault();
      });
    }
    bindStudioFrameLoad("data-editor-logo-frame-bound", update);
    updateNow();
  }

  function updateTheme() {
    var kit = themeKitValue();
    var template = themeTemplateValue();
    var palette = themePaletteValue();
    var ratio = document.querySelector('input[name="themeImageRatio"]:checked');
    var colors = themeColors();
    var preview = previewRuntime();
    if (preview.applyTheme) {
      preview.applyTheme({
        kit: kit,
        template: template,
        palette: palette,
        imageRatio: ratio ? ratio.value : "",
        customClasses: template === "custom" ? customTemplateClasses() : [],
        styleClasses: styleClasses(),
        colors: colors
      });
    }
    updateThemeSwatches(colors);
    updateKitCards(kit);
    updateTemplateCards(template);
    updateCustomTemplateBuilder(template);
    syncStyleControlButtons(document);
  }

  function bindTheme(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var controls = scope.querySelectorAll('[name="themeKit"], [name="themeTemplate"], [name="themePalette"], [name="customTemplateName"], [data-editor-custom-class], [data-editor-style-class], input[name="themeImageRatio"], [data-editor-color-token]');
    var updateThemeFrame = frameTask(updateTheme);
    Array.prototype.forEach.call(controls, function (control) {
      if (control.dataset.gosxStudioThemeBound === "true") return;
      control.dataset.gosxStudioThemeBound = "true";
      control.addEventListener("input", updateThemeFrame);
      control.addEventListener("change", function () {
        if (control.matches("[data-editor-kit-input]")) applyKit(control);
        if (control.matches("[data-editor-theme-preset]")) applyPresetDefaults(control);
        updateThemeFrame();
      });
    });
    bindStudioFrameLoad("data-editor-theme-frame-bound", updateThemeFrame);
    updateTheme();
  }

  function themeKitValue() {
    var radio = document.querySelector('input[name="themeKit"]:checked');
    return radio ? radio.value : "";
  }

  function themeTemplateValue() {
    var radio = document.querySelector('input[name="themeTemplate"]:checked');
    return radio ? radio.value : "studio";
  }

  function customTemplateClasses() {
    return [
      "theme--custom-hero-" + customValue("customHeroLayout", "overlay"),
      "theme--custom-width-" + customValue("customContentWidth", "standard"),
      "theme--custom-density-" + customValue("customProductDensity", "balanced"),
      "theme--custom-height-" + customValue("customHeroHeight", "standard")
    ];
  }

  function customValue(name, fallback) {
    var input = document.querySelector('[name="' + name + '"]');
    return input && input.value ? input.value : fallback;
  }

  function styleClasses() {
    return [
      "theme--nav-" + customValue("styleNav", "links"),
      "theme--buttons-" + customValue("styleButtons", "pill"),
      "theme--cards-" + customValue("styleCards", "quiet"),
      "theme--spacing-" + customValue("styleSpacing", "balanced"),
      "theme--images-" + customValue("styleImageFrame", "sharp"),
      "theme--motion-" + customValue("styleMotion", "subtle")
    ];
  }

  function themePaletteValue() {
    var select = document.querySelector("select[name='themePalette']");
    if (select) return select.value;
    var radio = document.querySelector('input[name="themePalette"]:checked');
    return radio ? radio.value : "clay";
  }

  function themeColors() {
    var out = {};
    Array.prototype.forEach.call(document.querySelectorAll("[data-editor-color-token]"), function (input) {
      var name = input.getAttribute("data-editor-color-token");
      if (name && input.value) out[name] = input.value;
    });
    return out;
  }

  function applyPresetDefaults(select) {
    var option = select && select.options ? select.options[select.selectedIndex] : null;
    if (!option) return;
    Array.prototype.forEach.call(document.querySelectorAll("[data-editor-color-token]"), function (input) {
      var key = input.getAttribute("data-editor-color-key");
      var attr = "data-color-" + String(key || "").replace(/[A-Z]/g, function (letter) { return "-" + letter.toLowerCase(); });
      var value = option.getAttribute(attr);
      if (value) {
        input.value = value;
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });
  }

  function applyKit(input) {
    var card = input && input.closest ? input.closest("[data-editor-kit-card]") : null;
    if (!card) return;
    setRadioValue("themeTemplate", card.getAttribute("data-kit-template"));
    setSelectValue("themePalette", card.getAttribute("data-kit-palette"));
    setRadioValue("themeImageRatio", card.getAttribute("data-kit-image-ratio"));
    setInputValue("customTemplateName", card.getAttribute("data-kit-custom-name"));
    setSelectValue("customHeroLayout", card.getAttribute("data-kit-custom-hero-layout"));
    setSelectValue("customContentWidth", card.getAttribute("data-kit-custom-content-width"));
    setSelectValue("customProductDensity", card.getAttribute("data-kit-custom-product-density"));
    setSelectValue("customHeroHeight", card.getAttribute("data-kit-custom-hero-height"));
    setSelectValue("styleNav", card.getAttribute("data-kit-style-nav"));
    setSelectValue("styleButtons", card.getAttribute("data-kit-style-buttons"));
    setSelectValue("styleCards", card.getAttribute("data-kit-style-cards"));
    setSelectValue("styleSpacing", card.getAttribute("data-kit-style-spacing"));
    setSelectValue("styleImageFrame", card.getAttribute("data-kit-style-image-frame"));
    setSelectValue("styleMotion", card.getAttribute("data-kit-style-motion"));
    applyPresetDefaults(document.querySelector("select[name='themePalette']"));
  }

  function setInputValue(name, value) {
    var input = document.querySelector('[name="' + name + '"]');
    if (!input || !value) return;
    input.value = value;
  }

  function setSelectValue(name, value) {
    var select = document.querySelector('[name="' + name + '"]');
    if (!select || !value) return;
    select.value = value;
  }

  function setRadioValue(name, value) {
    if (!value) return;
    var radio = document.querySelector('input[name="' + name + '"][value="' + value + '"]');
    if (radio) radio.checked = true;
  }

  function updateThemeSwatches(colors) {
    var swatches = document.querySelectorAll("[data-editor-theme-swatches] .theme-swatch");
    var values = Object.keys(colors).map(function (key) { return colors[key]; });
    Array.prototype.forEach.call(swatches, function (swatch, index) {
      if (values[index]) swatch.style.background = values[index];
    });
  }

  function updateKitCards(active) {
    Array.prototype.forEach.call(document.querySelectorAll("[data-editor-kit-card]"), function (card) {
      card.classList.toggle("is-selected", card.getAttribute("data-editor-kit-card") === active);
    });
  }

  function updateTemplateCards(active) {
    Array.prototype.forEach.call(document.querySelectorAll("[data-editor-template-card]"), function (card) {
      card.classList.toggle("is-selected", card.getAttribute("data-editor-template-card") === active);
    });
    var customName = document.querySelector("[data-editor-custom-template-name]");
    var customCard = document.querySelector('[data-editor-template-card="custom"] strong');
    if (customName && customCard && customName.value.trim()) customCard.textContent = customName.value.trim();
  }

  function updateCustomTemplateBuilder(active) {
    var builder = document.querySelector("[data-custom-template-builder]");
    if (builder) builder.classList.toggle("is-active", active === "custom");
  }

  function bindCSS(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var input = scope.querySelector("[data-editor-css]");
    if (!input || input.dataset.gosxStudioCSSBound === "true") return;
    input.dataset.gosxStudioCSSBound = "true";
    var updateNow = function () {
      var preview = previewRuntime();
      if (preview.applyCSS) preview.applyCSS(input.value || "");
    };
    var update = frameTask(updateNow);
    input.addEventListener("input", update);
    bindStudioFrameLoad("data-editor-css-frame-bound", update);
    updateNow();
  }

  function cssString(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"').replace(/[\n\r]/g, " ").trim();
  }

  function cssURL(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, "%22").replace(/[\n\r]/g, "").trim();
  }

  function fontFormat(url) {
    var clean = String(url || "").split("?")[0].toLowerCase();
    if (clean.endsWith(".woff2")) return ' format("woff2")';
    if (clean.endsWith(".woff")) return ' format("woff")';
    if (clean.endsWith(".ttf")) return ' format("truetype")';
    if (clean.endsWith(".otf")) return ' format("opentype")';
    return "";
  }

  function bindFonts(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var inputs = scope.querySelectorAll("[data-editor-font-name], [data-editor-font-url]");
    if (!inputs.length) return;
    var updateNow = function () {
      var css = [];
      var vars = [];
      ["display", "body", "mono"].forEach(function (slot) {
        var name = scope.querySelector('[data-editor-font-name="' + slot + '"]');
        var url = scope.querySelector('[data-editor-font-url="' + slot + '"]');
        var family = name && name.value ? name.value.trim() : "";
        var source = url && url.value ? url.value.trim() : "";
        if (family && source) {
          css.push('@font-face{font-family:"' + cssString(family) + '";src:url("' + cssURL(source) + '")' + fontFormat(source) + ';font-display:swap;}');
        }
        if (family) {
          var token = slot === "display" ? "--font-display" : slot === "body" ? "--font-body" : "--font-mono";
          vars.push(token + ':"' + cssString(family) + '";');
        }
      });
      if (vars.length) css.push(":root{" + vars.join("") + "}");
      var preview = previewRuntime();
      if (preview.applyFonts) preview.applyFonts(css.join("\n"));
    };
    var update = frameTask(updateNow);
    Array.prototype.forEach.call(inputs, function (input) {
      if (input.dataset.gosxStudioFontBound === "true") return;
      input.dataset.gosxStudioFontBound = "true";
      input.addEventListener("input", update);
      input.addEventListener("change", update);
    });
    bindStudioFrameLoad("data-editor-font-frame-bound", update);
    updateNow();
  }

  function ownerLabel(value) {
    value = String(value || "").replace(/[-_]/g, " ");
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  function namedControl(name) {
    return document.querySelector('[name="' + attrValue(name) + '"]');
  }

  function selectedKitCard() {
    var input = document.querySelector('input[name="themeKit"]:checked');
    if (input && input.closest) return input.closest("[data-editor-kit-card]");
    return null;
  }

  function styleControlValue(name) {
    var checked = document.querySelector('input[name="' + attrValue(name) + '"]:checked');
    if (checked) return checked.value;
    var control = namedControl(name);
    return control ? control.value || "" : "";
  }

  function kitDefaultForControl(name) {
    var attrs = {
      customHeroLayout: "data-kit-custom-hero-layout",
      customContentWidth: "data-kit-custom-content-width",
      customProductDensity: "data-kit-custom-product-density",
      customHeroHeight: "data-kit-custom-hero-height",
      styleNav: "data-kit-style-nav",
      styleButtons: "data-kit-style-buttons",
      styleCards: "data-kit-style-cards",
      styleSpacing: "data-kit-style-spacing",
      styleImageFrame: "data-kit-style-image-frame",
      styleMotion: "data-kit-style-motion",
      themeImageRatio: "data-kit-image-ratio"
    };
    var fallbacks = {
      customHeroLayout: "overlay",
      customContentWidth: "standard",
      customProductDensity: "balanced",
      customHeroHeight: "standard",
      styleNav: "links",
      styleButtons: "pill",
      styleCards: "quiet",
      styleSpacing: "balanced",
      styleImageFrame: "sharp",
      styleMotion: "subtle",
      themeImageRatio: "landscape"
    };
    var card = selectedKitCard();
    var attr = attrs[name];
    var value = card && attr ? card.getAttribute(attr) : "";
    return value || fallbacks[name] || "";
  }

  function styleImpactFor(name) {
    var catalog = {
      customHeroLayout: {
        label: "Page shape",
        summary: "Hero composition and above-the-fold frame",
        selector: ".hero, .hero__copy, .hero__media"
      },
      customContentWidth: {
        label: "Page width",
        summary: "Site frame, sections, and reading measure",
        selector: ".site-shell, .hero, .section"
      },
      customHeroHeight: {
        label: "Hero height",
        summary: "Hero depth and first viewport balance",
        selector: ".hero"
      },
      styleSpacing: {
        label: "Section rhythm",
        summary: "Vertical rhythm, grids, and section breathing room",
        selector: ".hero, .section, .product-grid, .post-grid, .gallery-strip"
      },
      styleCards: {
        label: "Cards",
        summary: "Product, post, and gallery card treatment",
        selector: ".product-card, .post-card, .gallery-card"
      },
      customProductDensity: {
        label: "Product rhythm",
        summary: "Product grid density and commerce scan rhythm",
        selector: ".product-grid, .product-card"
      },
      styleNav: {
        label: "Navigation",
        summary: "Header, brand, and primary navigation treatment",
        selector: ".site-header, .brand, nav, .nav"
      },
      styleButtons: {
        label: "Buttons",
        summary: "Primary and secondary action treatment",
        selector: ".button"
      },
      styleImageFrame: {
        label: "Image frame",
        summary: "Media corner shape and card image treatment",
        selector: ".hero__media, img, .product-card, .post-card, .gallery-card"
      },
      themeImageRatio: {
        label: "Image crop",
        summary: "Image aspect rhythm across product, blog, and gallery media",
        selector: ".hero__media, img, .product-card img, .post-card img, .gallery-card img"
      },
      styleMotion: {
        label: "Motion",
        summary: "Interactive movement across links, cards, and actions",
        selector: ".site-shell, .hero, .section, .button, .product-card, .post-card, .gallery-card"
      }
    };
    return catalog[name] || {
      label: ownerLabel(name),
      summary: "Scoped style surface",
      selector: ".site-shell"
    };
  }

  function clearStyleImpactNodes() {
    var preview = previewRuntime();
    if (preview.applyStyleImpact) preview.applyStyleImpact("");
  }

  function styleScopePath(form) {
    var bits = ["site", "home"];
    if (form) {
      var selection = form.getAttribute("data-studio-selection") || "";
      var field = form.getAttribute("data-studio-field-selection") || "";
      if (selection) bits.push(selection);
      if (field) bits.push(field);
    }
    return bits.join(" > ");
  }

  function styleStatePath(form) {
    var state = form && form.getAttribute("data-studio-style-state") || "default";
    var breakpoint = form && form.getAttribute("data-studio-breakpoint") || "desktop";
    return state + " / " + breakpoint;
  }

  function setImpactReadout(selector, value) {
    Array.prototype.forEach.call(document.querySelectorAll(selector), function (node) {
      node.textContent = value;
    });
  }

  function showStyleImpact(name, value, committed) {
    var meta = styleImpactFor(name);
    var form = editorWorkbench(document);
    var preview = previewRuntime();
    var count = preview.applyStyleImpact ? preview.applyStyleImpact(meta.selector) : 0;
    var label = meta.label + " / " + ownerLabel(value || styleControlValue(name));
    setImpactReadout("[data-studio-style-impact-label]", label);
    setImpactReadout("[data-studio-style-impact-summary]", meta.summary);
    setImpactReadout("[data-studio-style-impact-count]", count + (count === 1 ? " affected" : " affected"));
    setImpactReadout("[data-studio-style-impact-scope]", styleScopePath(form));
    setImpactReadout("[data-studio-style-impact-state]", styleStatePath(form));
    Array.prototype.forEach.call(document.querySelectorAll("[data-studio-style-impact-panel]"), function (panel) {
      panel.classList.toggle("is-live", !committed);
      panel.classList.toggle("has-change", !!committed);
    });
    if (committed) lastStyleImpact = { name: name, value: value };
  }

  function restoreStyleImpact() {
    if (lastStyleImpact) {
      showStyleImpact(lastStyleImpact.name, lastStyleImpact.value, true);
      return;
    }
    clearStyleImpactNodes();
    setImpactReadout("[data-studio-style-impact-label]", "No active recipe");
    setImpactReadout("[data-studio-style-impact-summary]", "Awaiting scoped change.");
    setImpactReadout("[data-studio-style-impact-count]", "0 affected");
    var form = editorWorkbench(document);
    setImpactReadout("[data-studio-style-impact-scope]", styleScopePath(form));
    setImpactReadout("[data-studio-style-impact-state]", styleStatePath(form));
    Array.prototype.forEach.call(document.querySelectorAll("[data-studio-style-impact-panel]"), function (panel) {
      panel.classList.remove("is-live");
      panel.classList.remove("has-change");
    });
  }

  function setStyleControlValue(name, value) {
    if (!name || !value) return;
    if (name.indexOf("custom") === 0 && name !== "customTemplateName") {
      var customTemplate = document.querySelector('input[name="themeTemplate"][value="custom"]');
      if (customTemplate && !customTemplate.checked) {
        customTemplate.checked = true;
        customTemplate.dispatchEvent(new Event("input", { bubbles: true }));
        customTemplate.dispatchEvent(new Event("change", { bubbles: true }));
      }
    }
    var radio = document.querySelector('input[name="' + attrValue(name) + '"][value="' + attrValue(value) + '"]');
    var control = radio || namedControl(name);
    if (!control) return;
    if (radio) radio.checked = true;
    else control.value = value;
    control.dispatchEvent(new Event("input", { bubbles: true }));
    control.dispatchEvent(new Event("change", { bubbles: true }));
    syncStyleControlButtons(document);
    showStyleImpact(name, value, true);
  }

  function resetStyleControlValue(name) {
    var value = kitDefaultForControl(name);
    if (!value) return;
    setStyleControlValue(name, value);
  }

  function syncStyleControlButtons(root) {
    root = root || document;
    Array.prototype.forEach.call(root.querySelectorAll("[data-studio-style-control]"), function (button) {
      var name = button.getAttribute("data-studio-style-control") || "";
      var value = button.getAttribute("data-studio-style-value") || "";
      var active = !!name && styleControlValue(name) === value;
      button.setAttribute("aria-pressed", active ? "true" : "false");
      button.classList.toggle("is-selected", active);
    });
    Array.prototype.forEach.call(root.querySelectorAll("[data-studio-style-readout]"), function (readout) {
      var name = readout.getAttribute("data-studio-style-readout") || "";
      var value = styleControlValue(name);
      var defaultValue = kitDefaultForControl(name);
      var inherited = !!defaultValue && value === defaultValue;
      readout.textContent = value ? ownerLabel(value) : "Inherited";
      readout.setAttribute("data-studio-style-inherited", inherited ? "true" : "false");
    });
    Array.prototype.forEach.call(root.querySelectorAll("[data-studio-style-reset]"), function (button) {
      var name = button.getAttribute("data-studio-style-reset") || "";
      var value = styleControlValue(name);
      var defaultValue = kitDefaultForControl(name);
      var inherited = !!defaultValue && value === defaultValue;
      var group = button.closest(".studio-style-control-group");
      var meta = styleImpactFor(name);
      if (group) group.setAttribute("data-studio-style-inherited", inherited ? "true" : "false");
      button.disabled = inherited;
      button.textContent = inherited ? "Kit" : "Reset";
      button.setAttribute(
        "aria-label",
        inherited
          ? meta.label + " is using the starter kit"
          : "Reset " + meta.label + " to " + ownerLabel(defaultValue)
      );
    });
  }

  function bindStyleWorkbench(root) {
    var form = editorWorkbench(root);
    if (!form || form.dataset.gosxStudioStyleWorkbenchBound === "true") return;
    form.dataset.gosxStudioStyleWorkbenchBound = "true";
    var previewImpact = frameTask(showStyleImpact);
    var restoreImpact = frameTask(restoreStyleImpact);
    var syncControls = frameTask(function () {
      syncStyleControlButtons(form);
      restoreStyleImpact();
    });
    form.addEventListener("click", function (event) {
      var resetButton = event.target.closest("[data-studio-style-reset]");
      if (resetButton && form.contains(resetButton)) {
        event.preventDefault();
        resetStyleControlValue(resetButton.getAttribute("data-studio-style-reset"));
        return;
      }
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      event.preventDefault();
      setStyleControlValue(button.getAttribute("data-studio-style-control"), button.getAttribute("data-studio-style-value"));
    });
    form.addEventListener("pointerover", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      previewImpact(button.getAttribute("data-studio-style-control"), button.getAttribute("data-studio-style-value"), false);
    });
    form.addEventListener("pointerout", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      if (button.contains(event.relatedTarget)) return;
      restoreImpact();
    });
    form.addEventListener("focusin", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      previewImpact(button.getAttribute("data-studio-style-control"), button.getAttribute("data-studio-style-value"), false);
    });
    form.addEventListener("focusout", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      restoreImpact();
    });
    form.addEventListener("input", function (event) {
      if (event.target && event.target.matches && event.target.matches('[name="themeTemplate"], [name="themeImageRatio"], [data-editor-custom-class], [data-editor-style-class]')) {
        syncControls();
      }
    });
    form.addEventListener("change", function (event) {
      if (event.target && event.target.matches && event.target.matches('[name="themeTemplate"], [name="themeImageRatio"], [data-editor-custom-class], [data-editor-style-class]')) {
        syncControls();
      }
    });
    bindStudioFrameLoad("data-editor-style-impact-frame-bound", restoreImpact);
    syncStyleControlButtons(form);
    restoreStyleImpact();
  }

  function bindBlockLayoutList(list) {
    if (!list || list.dataset.gosxStudioBlockLayoutBound === "true") return;
    list.dataset.gosxStudioBlockLayoutBound = "true";
    blockRows(list).forEach(function (row) { row.draggable = false; });
    list.addEventListener("click", function (event) {
      var move = event.target.closest("[data-block-studio-move]");
      if (move && list.contains(move)) {
        event.preventDefault();
        moveBlockLayoutRow(list, move.closest("[data-block-studio-block]"), move.getAttribute("data-block-studio-move"));
        return;
      }
      var row = event.target.closest("[data-block-studio-block]");
      if (row && list.contains(row)) selectBlockLayoutRow(list, blockRowKey(row));
    });
    bindBlockLayoutHandleDrag(list);
    renumberBlockLayoutList(list, "engine-init");
  }

  function blockRows(list) {
    return Array.prototype.slice.call(list.querySelectorAll("[data-block-studio-block]"));
  }

  function blockRowKey(row) {
    return row ? row.getAttribute("data-block-studio-block") || "" : "";
  }

  function blockRowForKey(root, key) {
    return root.querySelector('[data-block-studio-block="' + attrValue(key) + '"]');
  }

  function updateBlockLayoutLibraryState(root) {
    var scope = root && root.querySelectorAll ? root : document;
    Array.prototype.forEach.call(scope.querySelectorAll("[data-editor-add-block]"), function (button) {
      var key = button.getAttribute("data-editor-add-block");
      var row = blockRowForKey(document, key);
      var checkbox = row && row.querySelector("[data-editor-block-visible]");
      var enabled = !!(checkbox && checkbox.checked);
      var base = button.getAttribute("data-editor-button-base") || "button";
      button.className = base + " " + (enabled ? "button--ghost is-active" : "button--secondary");
      button.setAttribute("aria-pressed", enabled ? "true" : "false");
      var label = button.querySelector("small");
      if (label) label.textContent = enabled ? "On page" : "Add";
    });
  }

  function setPreviewBlockVisibility(key, visible) {
    var preview = window.GoSXStudioPreviewRuntime;
    if (preview && typeof preview.setBlockVisibility === "function") {
      preview.setBlockVisibility(key, visible);
    }
  }

  function updateBlockLayoutVisibilityState(check) {
    var row = check && check.closest ? check.closest("[data-block-studio-block]") : null;
    if (!row) return;
    var visible = !!check.checked;
    var key = blockRowKey(row);
    var statuses = row.querySelectorAll("[data-editor-block-status], [data-editor-block-pill]");
    row.classList.toggle("editor-block--hidden", !visible);
    Array.prototype.forEach.call(statuses, function (status) {
      status.textContent = visible ? "Visible" : "Hidden";
    });
    var pill = row.querySelector("[data-editor-block-pill]");
    if (pill) pill.className = visible ? "status status--ready" : "status";
    setPreviewBlockVisibility(key, visible);
    updateBlockLayoutLibraryState(document);
  }

  function bindBlockLayoutVisibility(root) {
    var scope = root && root.querySelectorAll ? root : document;
    Array.prototype.forEach.call(scope.querySelectorAll("[data-editor-block-visible]"), function (check) {
      if (check.dataset.gosxStudioBlockVisibilityBound === "true") return;
      check.dataset.gosxStudioBlockVisibilityBound = "true";
      check.addEventListener("change", function () {
        updateBlockLayoutVisibilityState(check);
      });
      updateBlockLayoutVisibilityState(check);
    });
  }

  function bindBlockLayoutLibrary(root) {
    var form = editorWorkbench(root);
    if (!form || form.dataset.gosxStudioBlockLibraryBound === "true") return;
    form.dataset.gosxStudioBlockLibraryBound = "true";
    form.addEventListener("click", function (event) {
      var button = event.target.closest("[data-editor-add-block]");
      if (!button || !form.contains(button)) return;
      var key = button.getAttribute("data-editor-add-block");
      var row = blockRowForKey(form, key);
      if (!row) return;
      var checkbox = row.querySelector("[data-editor-block-visible]");
      if (checkbox && !checkbox.checked) {
        checkbox.checked = true;
        checkbox.dispatchEvent(new Event("input", { bubbles: true }));
        checkbox.dispatchEvent(new Event("change", { bubbles: true }));
      }
      selectBlockLayoutRow(form, key);
      var reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      row.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });
      window.setTimeout(function () { row.focus(); }, 180);
      updateBlockLayoutLibraryState(document);
    });
    updateBlockLayoutLibraryState(document);
  }

  function renumberBlockLayoutList(list, source) {
    blockRows(list).forEach(function (row, index) {
      var input = row.querySelector("[data-block-studio-order]");
      if (input) input.value = String(index + 1);
      row.setAttribute("data-block-studio-index", String(index));
    });
    updateBlockMoveButtons(list);
    list.dispatchEvent(new CustomEvent("blockstudio:reorder", {
      bubbles: true,
      detail: { source: source || "block-layout-engine" }
    }));
    list.dispatchEvent(new Event("input", { bubbles: true }));
    list.dispatchEvent(new Event("change", { bubbles: true }));
  }

  function updateBlockMoveButtons(list) {
    var all = blockRows(list);
    all.forEach(function (row, index) {
      var up = row.querySelector('[data-block-studio-move="up"]');
      var down = row.querySelector('[data-block-studio-move="down"]');
      if (up) up.disabled = index === 0;
      if (down) down.disabled = index === all.length - 1;
    });
  }

  function moveBlockLayoutRow(list, row, direction) {
    if (!row) return;
    if (direction === "up" && row.previousElementSibling) {
      list.insertBefore(row, row.previousElementSibling);
    }
    if (direction === "down" && row.nextElementSibling) {
      list.insertBefore(row.nextElementSibling, row);
    }
    renumberBlockLayoutList(list, "engine-buttons");
    selectBlockLayoutRow(list, blockRowKey(row));
  }

  function commitBlockLayoutReorder(list, key, targetKey, position) {
    if (!key || !targetKey || key === targetKey) return false;
    var row = blockRowForKey(list, key);
    var target = blockRowForKey(list, targetKey);
    if (!row || !target) return false;
    if (position === "after") {
      list.insertBefore(row, target.nextElementSibling);
    } else {
      list.insertBefore(row, target);
    }
    renumberBlockLayoutList(list, "engine-preview");
    selectBlockLayoutRow(list, key);
    return true;
  }

  function selectBlockLayoutRow(root, key) {
    if (!key) return;
    blockRows(root).forEach(function (row) {
      row.classList.toggle("is-selected", blockRowKey(row) === key);
    });
    var row = blockRowForKey(root, key);
    if (row && row.scrollIntoView) row.scrollIntoView({ block: "nearest", behavior: "smooth" });
    document.dispatchEvent(new CustomEvent("blockstudio:select", {
      bubbles: true,
      detail: { key: key }
    }));
  }

  function bindBlockLayoutHandleDrag(list) {
    var drag = null;
    list.addEventListener("pointerdown", function (event) {
      var handle = event.target.closest("[data-block-studio-handle]");
      if (!handle || !list.contains(handle)) return;
      var row = handle.closest("[data-block-studio-block]");
      if (!row) return;
      event.preventDefault();
      drag = { row: row, key: blockRowKey(row), y: event.clientY };
      row.classList.add("is-dragging");
      handle.setPointerCapture && handle.setPointerCapture(event.pointerId);
    });
    list.addEventListener("pointermove", function (event) {
      if (!drag) return;
      event.preventDefault();
      var target = closestBlockLayoutRowByY(list, event.clientY, drag.row);
      if (!target) return;
      var rect = target.getBoundingClientRect();
      list.insertBefore(drag.row, event.clientY > rect.top + rect.height / 2 ? target.nextElementSibling : target);
    });
    list.addEventListener("pointerup", finish);
    list.addEventListener("pointercancel", finish);
    function finish() {
      if (!drag) return;
      drag.row.classList.remove("is-dragging");
      renumberBlockLayoutList(list, "engine-list");
      selectBlockLayoutRow(list, drag.key);
      drag = null;
    }
  }

  function closestBlockLayoutRowByY(list, y, dragging) {
    var best = null;
    var bestDistance = Number.POSITIVE_INFINITY;
    blockRows(list).forEach(function (row) {
      if (row === dragging) return;
      var rect = row.getBoundingClientRect();
      var distance = Math.abs(y - (rect.top + rect.height / 2));
      if (distance < bestDistance) {
        best = row;
        bestDistance = distance;
      }
    });
    return best;
  }

  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function mountShowcase3DViewer(context) {
    var mount = context && context.mount;
    if (!mount || mount.dataset.gosxStudioShowcase3dMounted === "true") return;
    mount.dataset.gosxStudioShowcase3dMounted = "true";
    var props = normalizeShowcase3DProps((context && context.props) || {});
    mount.dataset.showcase3dState = props.modelURI ? "model-ready" : "poster-only";
    mount.dataset.showcase3dModelUri = props.modelURI;
    mount.dataset.showcase3dPopout = props.popout ? "true" : "false";
    mount.replaceChildren(showcase3DShell(props));
  }

  function normalizeShowcase3DProps(props) {
    var modelURI = stringProp(props, "modelURI", "modelUri", "modelURL", "modelUrl", "model");
    return {
      label: stringProp(props, "label", "title", "name") || "3D showcase",
      summary: stringProp(props, "summary", "description") || "Approved model viewer.",
      modelURI: modelURI,
      posterURI: stringProp(props, "posterURI", "posterUri", "posterURL", "posterUrl", "poster"),
      posterAlt: stringProp(props, "posterAlt", "alt") || "3D showcase poster",
      fallbackText: stringProp(props, "fallbackText", "fallback") || "3D viewer unavailable.",
      popout: boolProp(props, "popout", "popoutEnabled", "PopoutEnabled") && !!modelURI
    };
  }

  function showcase3DShell(props) {
    var shell = element("div", "showcase3d-engine__shell");
    var frame = element("div", "showcase3d-engine__frame");
    if (props.posterURI) {
      var img = document.createElement("img");
      img.src = props.posterURI;
      img.alt = props.posterAlt;
      img.loading = "lazy";
      frame.appendChild(img);
    }
    var status = element("output", "showcase3d-engine__status");
    status.textContent = props.modelURI ? "Model asset connected" : props.fallbackText;
    frame.appendChild(status);
    shell.appendChild(frame);

    var copy = element("div", "showcase3d-engine__copy");
    var title = document.createElement("strong");
    title.textContent = props.label;
    var summary = document.createElement("span");
    summary.textContent = props.summary;
    copy.appendChild(title);
    copy.appendChild(summary);
    if (props.popout) {
      var link = document.createElement("a");
      link.href = props.modelURI;
      link.target = "_blank";
      link.rel = "noopener";
      link.textContent = "Open 3D";
      copy.appendChild(link);
    }
    shell.appendChild(copy);
    return shell;
  }

  function element(tag, className) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    return node;
  }

  function stringProp(props, names) {
    for (var i = 1; i < arguments.length; i += 1) {
      var value = props[arguments[i]];
      if (value == null) continue;
      value = String(value).trim();
      if (value) return value;
    }
    return "";
  }

  function boolProp(props) {
    for (var i = 1; i < arguments.length; i += 1) {
      var value = props[arguments[i]];
      if (value === true) return true;
      if (typeof value === "string" && /^(1|true|yes|on)$/i.test(value.trim())) return true;
    }
    return false;
  }

  function ensureStudioEngineRegistry() {
    window.__gosx_engine_factories = window.__gosx_engine_factories || Object.create(null);
    if (typeof window.__gosx_register_engine_factory !== "function") {
      window.__gosx_register_engine_factory = function (name, factory) {
        window.__gosx_engine_factories[name] = factory;
      };
    }
    Object.keys(studioEngineFactories).forEach(function (name) {
      window.__gosx_register_engine_factory(name, studioEngineFactories[name]);
    });
  }

  function manifestEntries() {
    var entries = [];
    Array.prototype.forEach.call(document.querySelectorAll('script[type="application/json"]'), function (script) {
      var manifest = null;
      try {
        manifest = JSON.parse(script.textContent || "{}");
      } catch (error) {
        return;
      }
      if (!manifest || !manifest.engines) return;
      manifest.engines.forEach(function (entry) {
        entries.push(entry);
      });
    });
    return entries;
  }

  function mountDeclaredStudioEngines() {
    ensureStudioEngineRegistry();
    manifestEntries().forEach(function (entry) {
      var registry = window.__gosx_engine_factories || studioEngineFactories;
      var factory = registry[entry.component];
      var mount = entry.mountId ? document.getElementById(entry.mountId) : null;
      if (!factory || !mount) return;
      factory({
        mount: mount,
        id: entry.id,
        kind: entry.kind,
        component: entry.component,
        props: entry.props || {},
        capabilities: entry.capabilities || [],
        programRef: entry.programRef || "",
        runtime: null,
        emit: function () {}
      });
    });
  }

  function init() {
    bindWorkbenchRailResizers(document);
    bindWorkbenchChrome(document);
    bindFieldRuntime(document);
    bindSelectionSurface(document);
    bindBrandLogo(document);
    bindTheme(document);
    bindStyleWorkbench(document);
    bindCSS(document);
    bindFonts(document);
    mountDeclaredStudioEngines();
  }

  window.GoSXStudioEngineRuntime = {
    factories: studioEngineFactories,
    mountDeclared: mountDeclaredStudioEngines
  };
  window.GoSXStudioWorkbenchRuntime = {
    bindRailResizers: bindWorkbenchRailResizers,
    bindChrome: bindWorkbenchChrome,
    setMode: setWorkbenchMode,
    syncViewport: syncWorkbenchViewport,
    activateViewport: activateWorkbenchViewport,
    currentBreakpoint: currentWorkbenchBreakpoint,
    setStyleState: setWorkbenchStyleState,
    syncZoom: syncWorkbenchZoom,
    activateZoom: activateWorkbenchZoom,
    toggleRail: toggleWorkbenchRail,
    toggleFocus: toggleWorkbenchFocus,
    toggleActivity: toggleWorkbenchActivity,
    saveLayout: saveWorkbenchLayout,
    currentRailWidth: currentWorkbenchRailWidth,
    setRailWidth: setWorkbenchRailWidth
  };
  window.GoSXStudioSelectionRuntime = {
    bind: bindSelectionSurface
  };
  window.GoSXStudioFieldRuntime = {
    bind: bindFieldRuntime,
    bindMirroring: bindFieldMirroring,
    bindClipboard: bindFieldClipboard
  };
  window.GoSXStudioBrandRuntime = {
    bindLogo: bindBrandLogo,
    updateHeaderLogo: updateHeaderLogo
  };
  window.GoSXStudioStyleRuntime = {
    bindTheme: bindTheme,
    bindWorkbench: bindStyleWorkbench,
    bindCSS: bindCSS,
    bindFonts: bindFonts,
    applyTheme: updateTheme,
    syncControlButtons: syncStyleControlButtons,
    showImpact: showStyleImpact,
    restoreImpact: restoreStyleImpact,
    setControlValue: setStyleControlValue,
    resetControlValue: resetStyleControlValue
  };
  window.GoSXStudioBlockLayoutRuntime = {
    rows: blockRows,
    rowKey: blockRowKey,
    rowForKey: blockRowForKey,
    moveRow: moveBlockLayoutRow,
    renumber: renumberBlockLayoutList,
    selectRow: selectBlockLayoutRow,
    commitReorder: commitBlockLayoutReorder,
    updateBlockLibraryState: updateBlockLayoutLibraryState,
    updateVisibilityState: updateBlockLayoutVisibilityState
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
  document.addEventListener("gosx:navigate", init);
  document.addEventListener("gosx:render", init);
})();
