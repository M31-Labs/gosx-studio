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
    var previewRefreshTimer = 0;
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
      return queryAll(form, "[data-gosx-studio-preview]");
    }

    function previewFrames() {
      return queryAll(form, "[data-studio-preview-frame]");
    }

    function previewURL(frame) {
      return frame.getAttribute("data-studio-preview-src") || frame.getAttribute("src") || "";
    }

    function frameDocument(frame) {
      try {
        return frame.contentDocument || (frame.contentWindow && frame.contentWindow.document) || null;
      } catch (error) {
        return null;
      }
    }

    function previewPatchSelector(source) {
      source = attrValue(source);
      return [
        '[data-studio-field="' + source + '"]',
        '[data-editor-preview="' + source + '"]',
        '[data-studio-field-source="' + source + '"]'
      ].join(",");
    }

    function ensurePreviewPatchStyles(doc) {
      if (!doc || doc.getElementById("gosx-studio-preview-patch-style")) return;
      var style = doc.createElement("style");
      style.id = "gosx-studio-preview-patch-style";
      style.textContent = [
        "[data-gosx-studio-preview-patched]{outline:2px solid currentColor;outline-offset:3px;transition:outline-offset 180ms ease,filter 180ms ease;}",
        "[data-gosx-studio-preview-patched='fresh']{outline-offset:6px;filter:brightness(1.03);}",
        "[data-gosx-studio-preview-selectable='true'] [data-studio-field],[data-gosx-studio-preview-selectable='true'] [data-editor-preview],[data-gosx-studio-preview-selectable='true'] [data-studio-field-source],[data-gosx-studio-preview-selectable='true'] [data-studio-block-key],[data-gosx-studio-preview-selectable='true'] [data-studio-node-id]{cursor:pointer;}",
        "[data-gosx-studio-preview-field-scope]{outline:1px dashed currentColor;outline-offset:3px;}",
        "[data-gosx-studio-preview-field-current]{outline:3px solid currentColor;outline-offset:6px;filter:saturate(1.08);}",
        "[data-gosx-studio-preview-selected]{outline:3px solid currentColor;outline-offset:6px;filter:saturate(1.08);}",
        "[data-gosx-studio-inline-editing]{outline:3px solid currentColor;outline-offset:5px;caret-color:currentColor;}",
        "[data-gosx-studio-inline-editing]:focus{outline-offset:7px;}"
      ].join("");
      (doc.head || doc.documentElement).appendChild(style);
    }

    function previewTargets(frame, patch) {
      var doc = frameDocument(frame);
      var field = patch && patch.field;
      var source = field && (field.source || field.name);
      if (!doc || !source) return [];
      return queryAll(doc, previewPatchSelector(source));
    }

    function readableFieldName(field, editable) {
      var value = compactText(field || "field").split(".").pop() || field || "Field";
      value = value.replace(/([a-z])([A-Z])/g, "$1 $2").replace(/[-_]/g, " ");
      value = value.charAt(0).toUpperCase() + value.slice(1);
      if (editable === "media" || editable === "image") return value === "Media" ? value : value + " media";
      if (editable === "source") return value + " source";
      if (editable === "flow") return value + " flow";
      if (editable === "url" || editable === "link") return value + " URL";
      return value || "Field";
    }

    function inspectorSource(field) {
      if (!field) return null;
      var source = attrValue(field);
      return form.querySelector('[data-studio-field-source="' + source + '"], [data-editor-source="' + source + '"]');
    }

    function inspectorControl(source) {
      if (!source) return null;
      if (source.matches && source.matches("input, textarea, select, button, a[href], [tabindex]")) return source;
      return source.querySelector ? source.querySelector("input, textarea, select, button, a[href], [tabindex]") : null;
    }

    function inferInspectorEditable(source, control) {
      var explicit = source && (source.getAttribute("data-studio-field-editable") || source.getAttribute("data-studio-editable"));
      if (explicit) return explicit;
      var tag = control && control.tagName ? String(control.tagName).toLowerCase() : "";
      var type = control && control.type ? String(control.type).toLowerCase() : "";
      if (tag === "textarea") return "text";
      if (tag === "input" && (type === "url" || type === "email")) return "url";
      if (tag === "input" || tag === "select") return "text";
      return "";
    }

    function previewBlockLabel(node, fallback) {
      if (!node) return fallback || "Preview selection";
      var explicit = node.getAttribute("data-studio-block-label") || node.getAttribute("data-studio-node-label") || node.getAttribute("aria-label") || "";
      if (explicit) return explicit;
      var heading = node.querySelector && node.querySelector("[data-studio-block-title], h1, h2, h3, strong");
      var text = compactText(heading ? heading.textContent : node.textContent);
      if (text) return text.length > 72 ? text.slice(0, 69) + "..." : text;
      return fallback || "Preview selection";
    }

    function previewSelectionDetail(node) {
      if (!node || !node.closest) return {};
      var fieldNode = node.closest("[data-studio-field], [data-editor-preview], [data-studio-field-source]");
      var blockNode = node.closest("[data-studio-block-key], [data-studio-node-id]");
      var field = fieldNode ? (fieldNode.getAttribute("data-studio-field") || fieldNode.getAttribute("data-editor-preview") || fieldNode.getAttribute("data-studio-field-source") || "") : "";
      var editable = fieldNode ? (fieldNode.getAttribute("data-studio-editable") || fieldNode.getAttribute("data-studio-field-editable") || "") : "";
      var source = inspectorSource(field);
      var control = inspectorControl(source);
      if (!editable) editable = inferInspectorEditable(source, control);
      var label = fieldNode ? (fieldNode.getAttribute("data-studio-field-label") || readableFieldName(field, editable)) : "";
      var action = fieldNode ? (fieldNode.getAttribute("data-studio-field-action") || "") : "";
      var actionHref = fieldNode ? (fieldNode.getAttribute("data-studio-field-action-href") || "") : "";
      var actionFormAction = fieldNode ? (fieldNode.getAttribute("data-studio-field-action-formaction") || "") : "";
      var blockKey = blockNode ? (blockNode.getAttribute("data-studio-block-key") || "") : "";
      var nodeID = blockNode ? (blockNode.getAttribute("data-studio-node-id") || "") : "";
      var blockLabel = blockNode ? previewBlockLabel(blockNode, blockKey || nodeID || "") : "";
      if (!label) label = blockLabel || "Preview selection";
      return {
        field: field,
        source: field,
        editable: editable,
        label: label,
        blockLabel: blockLabel,
        action: action,
        actionHref: actionHref,
        actionFormAction: actionFormAction,
        blockKey: blockKey,
        nodeID: nodeID
      };
    }

    function previewSelectableNode(target) {
      if (!target || !target.closest) return null;
      return target.closest("[data-studio-field], [data-editor-preview], [data-studio-field-source], [data-studio-block-key], [data-studio-node-id]");
    }

    function clearPreviewFieldMap(frame) {
      var doc = frameDocument(frame);
      if (!doc) return;
      queryAll(doc, "[data-gosx-studio-preview-field-scope], [data-gosx-studio-preview-field-current]").forEach(function (target) {
        target.removeAttribute("data-gosx-studio-preview-field-scope");
        target.removeAttribute("data-gosx-studio-preview-field-current");
        target.removeAttribute("data-gosx-studio-preview-field-position");
        target.removeAttribute("data-gosx-studio-preview-field-total");
      });
    }

    function clearPreviewSelections() {
      previewFrames().forEach(function (frame) {
        finishInlineTextEdit(frame, true, "clear-selection");
      });
      previewFrames().forEach(function (frame) {
        var doc = frameDocument(frame);
        if (!doc) return;
        clearPreviewFieldMap(frame);
        queryAll(doc, "[data-gosx-studio-preview-selected]").forEach(function (target) {
          target.removeAttribute("data-gosx-studio-preview-selected");
        });
      });
      previewShells().forEach(function (shell) {
        shell.removeAttribute("data-gosx-studio-preview-selection");
      });
      previewFrames().forEach(function (frame) {
        frame.removeAttribute("data-studio-preview-selection");
      });
      hidePreviewDocks();
    }

    function clearInspectorSelection() {
      queryAll(form, "[data-gosx-studio-inspector-selected]").forEach(function (target) {
        target.removeAttribute("data-gosx-studio-inspector-selected");
        if (target.classList) target.classList.remove("is-studio-field-active", "is-preview-selected");
      });
    }

    function ensurePreviewDockStyles() {
      if (document.getElementById("gosx-studio-preview-dock-style")) return;
      var style = document.createElement("style");
      style.id = "gosx-studio-preview-dock-style";
      style.textContent = [
        "[data-gosx-studio-preview]{position:relative;}",
        "[data-gosx-studio-preview-dock]{position:absolute;z-index:30;display:flex;align-items:center;gap:var(--gosx-studio-space-2,0.5rem);max-width:min(34rem,calc(100% - 1rem));padding:var(--gosx-studio-space-2,0.5rem);border:1px solid var(--gosx-studio-border,currentColor);border-radius:var(--gosx-studio-radius-md,0.5rem);background:var(--gosx-studio-surface-raised,Canvas);color:var(--gosx-studio-text,CanvasText);font:inherit;box-shadow:0 14px 40px color-mix(in srgb,currentColor 18%,transparent);transform:translate(-50%,-100%);transition:opacity var(--gosx-studio-duration-base,180ms) var(--gosx-studio-ease-out,ease),transform var(--gosx-studio-duration-base,180ms) var(--gosx-studio-ease-out,ease);}",
        "[data-gosx-studio-preview-dock][hidden]{display:none;}",
        "[data-gosx-studio-preview-dock][data-gosx-studio-preview-dock-placement='bottom']{transform:translate(-50%,0);}",
        "[data-gosx-studio-preview-dock-label]{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-weight:700;}",
        "[data-gosx-studio-preview-breadcrumb]{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--gosx-studio-muted,currentColor);font-size:0.8125em;}",
        "[data-gosx-studio-preview-dock-kind]{color:var(--gosx-studio-muted,currentColor);font-size:0.8125em;white-space:nowrap;}",
        "[data-gosx-studio-preview-field-meter]{color:var(--gosx-studio-muted,currentColor);font-size:0.8125em;white-space:nowrap;}",
        "[data-gosx-studio-preview-dock-actions]{display:flex;align-items:center;gap:var(--gosx-studio-space-1,0.25rem);margin-left:auto;}",
        "[data-gosx-studio-preview-command]{appearance:none;border:1px solid var(--gosx-studio-border,currentColor);border-radius:var(--gosx-studio-radius-sm,0.375rem);background:var(--gosx-studio-surface,Canvas);color:inherit;font:inherit;font-size:0.8125em;line-height:1;padding:var(--gosx-studio-space-2,0.5rem);cursor:pointer;}",
        "[data-gosx-studio-preview-command]:hover,[data-gosx-studio-preview-command]:focus-visible{border-color:var(--gosx-studio-accent,Highlight);outline:2px solid var(--gosx-studio-accent,Highlight);outline-offset:2px;}",
        "@media (prefers-reduced-motion: reduce){[data-gosx-studio-preview-dock]{transition:none;}}"
      ].join("");
      (document.head || document.documentElement).appendChild(style);
    }

    function previewShellForFrame(frame) {
      return frame && frame.closest ? (frame.closest("[data-gosx-studio-preview]") || frame.parentElement) : null;
    }

    function createDockButton(action, label) {
      var button = document.createElement("button");
      button.type = "button";
      button.textContent = label;
      button.setAttribute("data-gosx-studio-preview-command", action);
      return button;
    }

    function previewDockForFrame(frame) {
      var shell = previewShellForFrame(frame);
      if (!shell) return null;
      ensurePreviewDockStyles();
      var dock = shell.querySelector("[data-gosx-studio-preview-dock]");
      if (dock) return dock;
      dock = document.createElement("div");
      dock.hidden = true;
      dock.setAttribute("data-gosx-studio-preview-dock", "true");
      dock.setAttribute("role", "toolbar");
      dock.setAttribute("aria-label", "Preview selection actions");
      var label = document.createElement("strong");
      label.setAttribute("data-gosx-studio-preview-dock-label", "true");
      var breadcrumb = document.createElement("span");
      breadcrumb.hidden = true;
      breadcrumb.setAttribute("data-gosx-studio-preview-breadcrumb", "true");
      var kind = document.createElement("span");
      kind.setAttribute("data-gosx-studio-preview-dock-kind", "true");
      var meter = document.createElement("span");
      meter.hidden = true;
      meter.setAttribute("data-gosx-studio-preview-field-meter", "true");
      var actions = document.createElement("div");
      actions.setAttribute("data-gosx-studio-preview-dock-actions", "true");
      actions.appendChild(createDockButton("content", "Content"));
      actions.appendChild(createDockButton("style", "Style"));
      actions.appendChild(createDockButton("prev-field", "Prev field"));
      actions.appendChild(createDockButton("next-field", "Next field"));
      actions.appendChild(createDockButton("field-action", "Open"));
      actions.appendChild(createDockButton("clear", "Clear"));
      dock.appendChild(label);
      dock.appendChild(breadcrumb);
      dock.appendChild(kind);
      dock.appendChild(meter);
      dock.appendChild(actions);
      shell.appendChild(dock);
      dock.addEventListener("click", function (event) {
        var button = event.target && event.target.closest ? event.target.closest("[data-gosx-studio-preview-command]") : null;
        if (!button || !dock.contains(button)) return;
        event.preventDefault();
        runPreviewDockAction(frame, button.getAttribute("data-gosx-studio-preview-command") || "");
      });
      return dock;
    }

    function hidePreviewDocks() {
      queryAll(form, "[data-gosx-studio-preview-dock]").forEach(function (dock) {
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
    }

    function dockKindLabel(detail) {
      if (!detail || !detail.field) return "Block";
      var editable = detail.editable || "";
      if (editable === "media" || editable === "image") return "Media field";
      if (editable === "source") return "Source field";
      if (editable === "flow") return "Flow field";
      if (editable === "url" || editable === "link") return "Link field";
      if (editable === "text") return "Text field";
      return "Field";
    }

    function updatePreviewDockPosition(frame) {
      var dock = frame && frame.__gosxStudioPreviewDock;
      var target = frame && frame.__gosxStudioPreviewDockTarget;
      var shell = previewShellForFrame(frame);
      if (!dock || !target || !shell || dock.hidden) return;
      var frameRect = frame.getBoundingClientRect();
      var shellRect = shell.getBoundingClientRect();
      var targetRect = target.getBoundingClientRect();
      var left = frameRect.left - shellRect.left + targetRect.left + (targetRect.width / 2);
      var top = frameRect.top - shellRect.top + targetRect.top;
      var maxLeft = Math.max(8, shellRect.width - 8);
      left = clamp(left, 8, maxLeft);
      if (top < 52) {
        dock.setAttribute("data-gosx-studio-preview-dock-placement", "bottom");
        dock.style.top = Math.round(frameRect.top - shellRect.top + targetRect.bottom + 10) + "px";
      } else {
        dock.setAttribute("data-gosx-studio-preview-dock-placement", "top");
        dock.style.top = Math.round(top - 10) + "px";
      }
      dock.style.left = Math.round(left) + "px";
    }

    function fieldKeyForTarget(target) {
      if (!target) return "";
      return target.getAttribute("data-studio-field") || target.getAttribute("data-editor-preview") || target.getAttribute("data-studio-field-source") || "";
    }

    function fieldNavigationScope(frame, target, detail) {
      var doc = frameDocument(frame);
      if (!doc) return null;
      if (target && target.closest) {
        var targetBlock = target.closest("[data-studio-block-key], [data-studio-node-id]");
        if (targetBlock) return targetBlock;
      }
      var blockKey = detail && (detail.blockKey || detail.nodeID);
      if (blockKey) {
        var selector = '[data-studio-block-key="' + attrValue(blockKey) + '"], [data-studio-node-id="' + attrValue(blockKey) + '"]';
        return doc.querySelector(selector) || doc.body || doc.documentElement;
      }
      return doc.body || doc.documentElement;
    }

    function previewFieldNodesForSelection(frame, target, detail) {
      var scope = fieldNavigationScope(frame, target, detail);
      if (!scope) return [];
      var seen = {};
      return queryAll(scope, "[data-studio-field], [data-editor-preview], [data-studio-field-source]").filter(function (candidate) {
        var key = fieldKeyForTarget(candidate);
        if (!key || seen[key]) return false;
        seen[key] = true;
        return true;
      });
    }

    function previewFieldNavigationState(frame, target, detail) {
      var fields = previewFieldNodesForSelection(frame, target, detail);
      var current = detail && detail.field ? detail.field : fieldKeyForTarget(target);
      var index = -1;
      fields.forEach(function (candidate, candidateIndex) {
        if (index < 0 && fieldKeyForTarget(candidate) === current) index = candidateIndex;
      });
      return {
        fields: fields,
        count: fields.length,
        index: index
      };
    }

    function updatePreviewFieldNavigation(frame, dock, target, detail) {
      var state = previewFieldNavigationState(frame, target, detail);
      dock.setAttribute("data-gosx-studio-preview-field-count", String(state.count));
      dock.setAttribute("data-gosx-studio-preview-field-index", state.index >= 0 ? String(state.index + 1) : "");
      var meter = dock.querySelector("[data-gosx-studio-preview-field-meter]");
      if (meter) {
        meter.hidden = state.count < 2 || state.index < 0;
        meter.textContent = state.count > 1 && state.index >= 0 ? "Field " + (state.index + 1) + " of " + state.count : "";
      }
      ["prev-field", "next-field"].forEach(function (action) {
        var button = dock.querySelector('[data-gosx-studio-preview-command="' + action + '"]');
        if (!button) return;
        button.hidden = state.count === 0;
        button.disabled = state.count < 2;
        button.setAttribute("aria-label", (action === "prev-field" ? "Previous" : "Next") + " editable field");
      });
      return state;
    }

    function syncPreviewFieldMap(frame, target, detail) {
      clearPreviewFieldMap(frame);
      var state = previewFieldNavigationState(frame, target, detail);
      var current = detail && detail.field ? detail.field : fieldKeyForTarget(target);
      state.fields.forEach(function (candidate, index) {
        candidate.setAttribute("data-gosx-studio-preview-field-scope", "true");
        candidate.setAttribute("data-gosx-studio-preview-field-position", String(index + 1));
        candidate.setAttribute("data-gosx-studio-preview-field-total", String(state.count));
        if (fieldKeyForTarget(candidate) === current) {
          candidate.setAttribute("data-gosx-studio-preview-field-current", "true");
        }
      });
      return state;
    }

    function syncPreviewDock(frame, target, detail) {
      var dock = previewDockForFrame(frame);
      if (!dock || !target) return;
      frame.__gosxStudioPreviewDock = dock;
      frame.__gosxStudioPreviewDockTarget = target;
      dock.hidden = false;
      dock.setAttribute("data-gosx-studio-preview-field", detail.field || "");
      dock.setAttribute("data-gosx-studio-preview-block", detail.blockKey || detail.nodeID || "");
      dock.setAttribute("data-gosx-studio-preview-block-label", detail.blockLabel || "");
      dock.setAttribute("data-gosx-studio-preview-action-label", detail.action || "");
      dock.setAttribute("data-gosx-studio-preview-action-href", detail.actionHref || "");
      dock.setAttribute("data-gosx-studio-preview-action-formaction", detail.actionFormAction || "");
      dock.querySelector("[data-gosx-studio-preview-dock-label]").textContent = detail.label || detail.field || detail.blockKey || "Preview selection";
      var breadcrumb = dock.querySelector("[data-gosx-studio-preview-breadcrumb]");
      if (breadcrumb) {
        breadcrumb.hidden = !detail.blockLabel || !detail.field;
        breadcrumb.textContent = detail.blockLabel && detail.field ? detail.blockLabel + " / " + (detail.label || detail.field) : "";
      }
      dock.querySelector("[data-gosx-studio-preview-dock-kind]").textContent = dockKindLabel(detail);
      var action = dock.querySelector('[data-gosx-studio-preview-command="field-action"]');
      if (action) {
        action.textContent = detail.action || (detail.editable === "text" ? "Edit text" : detail.editable === "media" || detail.editable === "image" ? "Media" : detail.editable === "flow" ? "Flow" : detail.editable === "source" ? "Source" : "Open");
        action.disabled = !detail.field && !detail.action && !detail.actionHref && !detail.actionFormAction;
      }
      updatePreviewFieldNavigation(frame, dock, target, detail);
      syncPreviewFieldMap(frame, target, detail);
      updatePreviewDockPosition(frame);
    }

    function previewDockDetail(dock) {
      return {
        field: dock.getAttribute("data-gosx-studio-preview-field") || "",
        blockKey: dock.getAttribute("data-gosx-studio-preview-block") || "",
        blockLabel: dock.getAttribute("data-gosx-studio-preview-block-label") || "",
        label: compactText(dock.querySelector("[data-gosx-studio-preview-dock-label]") && dock.querySelector("[data-gosx-studio-preview-dock-label]").textContent),
        action: dock.getAttribute("data-gosx-studio-preview-action-label") || "",
        actionHref: dock.getAttribute("data-gosx-studio-preview-action-href") || "",
        actionFormAction: dock.getAttribute("data-gosx-studio-preview-action-formaction") || "",
        editable: form.getAttribute("data-studio-field-editable") || ""
      };
    }

    function emitPreviewDockAction(action, detail) {
      detail = detail || {};
      emitEditorOperation("preview_action", {
        mutation: false,
        reason: "preview-dock",
        target: {
          field: detail.field || "",
          editable: detail.editable || "",
          blockKey: detail.blockKey || "",
          selection: form.getAttribute("data-studio-selection") || detail.blockKey || detail.field || "",
          kind: form.getAttribute("data-studio-selection-kind") || "preview"
        },
        payload: {
          action: action,
          label: detail.label || "",
          blockLabel: detail.blockLabel || "",
          actionLabel: detail.action || "",
          actionHref: detail.actionHref || "",
          actionFormAction: detail.actionFormAction || ""
        }
      });
      emit(form, "gosxstudio:preview-action", {
        action: action,
        field: detail.field || "",
        editable: detail.editable || "",
        label: detail.label || "",
        blockLabel: detail.blockLabel || "",
        blockKey: detail.blockKey || "",
        actionLabel: detail.action || "",
        actionHref: detail.actionHref || "",
        actionFormAction: detail.actionFormAction || "",
        reason: "preview-dock"
      });
      emit(form, "gosxstudio:selection-action", {
        action: action,
        label: detail.action || detail.label || action,
        selection: form.getAttribute("data-studio-selection") || detail.blockKey || detail.field || "",
        kind: form.getAttribute("data-studio-selection-kind") || "preview"
      });
    }

    function isFormSubmitControl(node) {
      if (!node) return false;
      var tag = String(node.tagName || "").toLowerCase();
      var type = String(node.getAttribute("type") || "submit").toLowerCase();
      if (tag === "button") return type !== "button" && type !== "reset";
      if (tag === "input") return type === "submit" || type === "image";
      return false;
    }

    function fieldActionSubmitter(source, formAction) {
      if (!source || !source.querySelector) return null;
      var selector = "button[type='submit'], input[type='submit'], button[formaction], input[formaction], button[data-studio-field-action-formaction], input[data-studio-field-action-formaction]";
      var submitters = queryAll(source, selector);
      if (!formAction) return submitters[0] || null;
      for (var i = 0; i < submitters.length; i += 1) {
        var candidate = submitters[i];
        if (!isFormSubmitControl(candidate)) continue;
        if ((candidate.getAttribute("data-studio-field-action-formaction") || candidate.getAttribute("formaction") || "") === formAction) return candidate;
      }
      return submitters.filter(isFormSubmitControl)[0] || null;
    }

    function submitPreviewFieldAction(detail) {
      detail = detail || {};
      var source = inspectorSource(detail.field);
      var submitter = fieldActionSubmitter(source, detail.actionFormAction || "");
      var action = detail.actionFormAction || (submitter && (submitter.getAttribute("data-studio-field-action-formaction") || submitter.getAttribute("formaction"))) || "";
      if (!action) return false;
      var confirmMessage = (submitter && submitter.getAttribute("data-admin-confirm")) || (source && source.getAttribute("data-admin-confirm")) || "";
      if (confirmMessage && !window.confirm(confirmMessage)) return false;
      form.dataset.gosxStudioPendingAction = action;
      form.dataset.gosxStudioPendingActionLabel = detail.action || detail.label || "Field action";
      try {
        if (submitter && form.requestSubmit) {
          form.requestSubmit(submitter);
        } else if (form.requestSubmit) {
          var button = document.createElement("button");
          button.type = "submit";
          button.hidden = true;
          button.setAttribute("formaction", action);
          form.appendChild(button);
          form.requestSubmit(button);
          form.removeChild(button);
        } else {
          form.setAttribute("action", action);
          form.submit();
        }
      } catch (error) {
        if (submitter && submitter.click) submitter.click();
        else return false;
      }
      return true;
    }

    function navigatePreviewField(frame, direction, reason) {
      var dock = frame && frame.__gosxStudioPreviewDock;
      var target = frame && frame.__gosxStudioPreviewDockTarget;
      if (!dock || dock.hidden || !target) return false;
      finishInlineTextEdit(frame, true, "field-navigation");
      var detail = previewSelectionDetail(target);
      var state = previewFieldNavigationState(frame, target, detail);
      if (!state.count) return false;
      var currentIndex = state.index >= 0 ? state.index : (direction > 0 ? -1 : 0);
      var nextIndex = (currentIndex + direction + state.count) % state.count;
      var nextTarget = state.fields[nextIndex];
      var nextDetail = previewSelectionDetail(nextTarget);
      if (!nextTarget || !nextDetail.field) return false;
      if (!applyPreviewSelection(frame, nextTarget, nextDetail, { reveal: true, reason: reason || "field-navigation" })) return false;
      emitEditorOperation("preview_field_navigate", {
        mutation: false,
        reason: reason || "field-navigation",
        target: {
          field: nextDetail.field || "",
          editable: nextDetail.editable || "",
          blockKey: nextDetail.blockKey || "",
          selection: form.getAttribute("data-studio-selection") || nextDetail.field || "",
          kind: form.getAttribute("data-studio-selection-kind") || "preview-field"
        },
        payload: {
          direction: direction > 0 ? "next" : "previous",
          fieldIndex: nextIndex + 1,
          fieldCount: state.count,
          label: nextDetail.label || ""
        }
      });
      emit(form, "gosxstudio:preview-field-navigate", {
        field: nextDetail.field || "",
        editable: nextDetail.editable || "",
        label: nextDetail.label || "",
        blockKey: nextDetail.blockKey || "",
        direction: direction > 0 ? "next" : "previous",
        fieldIndex: nextIndex + 1,
        fieldCount: state.count,
        reason: reason || "field-navigation"
      });
      return true;
    }

    function runPreviewDockAction(frame, action) {
      var dock = frame && frame.__gosxStudioPreviewDock;
      if (!dock || !action) return false;
      var detail = previewDockDetail(dock);
      if (action === "clear") {
        clearPreviewSelections();
        clearInspectorSelection();
        form.removeAttribute("data-studio-selection");
        form.removeAttribute("data-studio-selection-kind");
        form.removeAttribute("data-studio-field-selection");
        form.removeAttribute("data-studio-field-editable");
        form.removeAttribute("data-studio-field-action-label");
        form.removeAttribute("data-studio-field-action-href");
        form.removeAttribute("data-studio-field-action-formaction");
        setReadout("[data-studio-selection-label]", "No selection");
        setReadout("[data-studio-selection-status]", "No selection");
        setReadout("[data-studio-field-selection-label]", "Block");
        emitPreviewDockAction(action, detail);
        return true;
      }
      if (action === "content") {
        setMode("content", { scroll: true, reason: "preview-dock" });
        var source = inspectorSource(detail.field);
        if (source) revealInspectorSelection(source, inspectorControl(source));
        emitPreviewDockAction(action, detail);
        return true;
      }
      if (action === "style") {
        setMode("style", { scroll: true, reason: "preview-dock" });
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
        if (detail.editable === "text" && startInlineTextFromDetail(frame, detail, "preview-dock")) return true;
        if (detail.actionFormAction && submitPreviewFieldAction(detail)) {
          emitPreviewDockAction(action, detail);
          return true;
        }
        if (detail.actionHref) {
          window.location.href = detail.actionHref;
        } else {
          var fieldSourceNode = inspectorSource(detail.field);
          if (fieldSourceNode) revealInspectorSelection(fieldSourceNode, inspectorControl(fieldSourceNode));
        }
        emitPreviewDockAction(action, detail);
        return true;
      }
      emitPreviewDockAction(action, detail);
      return true;
    }

    function markInspectorSelection(source) {
      if (!source) return null;
      var row = source.closest ? source.closest(".field-row, [data-studio-field-row]") : null;
      var control = inspectorControl(source);
      [source, row, control].forEach(function (target) {
        if (!target) return;
        target.setAttribute("data-gosx-studio-inspector-selected", "true");
        if (target.classList) target.classList.add(target === row ? "is-studio-field-active" : "is-preview-selected");
      });
      return control || source;
    }

    function revealInspectorSelection(source, control) {
      if (!source) return;
      if (form.querySelector('[data-studio-mode-control="content"], [data-studio-mode-panel="content"]')) {
        setMode("content", { reason: "preview-select" });
      }
      var target = (source.closest && source.closest(".field-row, [data-studio-field-row]")) || source;
      if (target.scrollIntoView) target.scrollIntoView({ block: "center", behavior: "smooth" });
      window.setTimeout(function () {
        if (control && control.focus) control.focus({ preventScroll: true });
      }, 120);
    }

    function applyPreviewSelection(frame, target, detail, options) {
      detail = detail || previewSelectionDetail(target);
      options = options || {};
      if (!detail.field && !detail.blockKey && !detail.nodeID) return false;
      clearPreviewSelections();
      clearInspectorSelection();
      var selectedTargets = detail.field ? previewTargets(frame, { field: { source: detail.field, name: detail.field } }) : [];
      if (!selectedTargets.length && target) selectedTargets = [target];
      selectedTargets.forEach(function (candidate) {
        candidate.setAttribute("data-gosx-studio-preview-selected", "true");
      });
      var source = inspectorSource(detail.field);
      var control = markInspectorSelection(source);
      var selectedEditable = detail.editable || inferInspectorEditable(source, control) || "";
      if (detail.field) {
        form.setAttribute("data-studio-field-selection", detail.field);
        if (selectedEditable) form.setAttribute("data-studio-field-editable", selectedEditable);
        else form.removeAttribute("data-studio-field-editable");
      } else {
        form.removeAttribute("data-studio-field-selection");
        form.removeAttribute("data-studio-field-editable");
      }
      var actionLabel = detail.action || (source && source.getAttribute("data-studio-field-action")) || "";
      if (actionLabel) form.setAttribute("data-studio-field-action-label", actionLabel);
      else form.removeAttribute("data-studio-field-action-label");
      var actionHref = detail.actionHref || (source && source.getAttribute("data-studio-field-action-href")) || "";
      if (actionHref) {
        form.setAttribute("data-studio-field-action-href", actionHref);
      } else {
        form.removeAttribute("data-studio-field-action-href");
      }
      var actionFormAction = detail.actionFormAction || (source && source.getAttribute("data-studio-field-action-formaction")) || "";
      if (actionFormAction) {
        form.setAttribute("data-studio-field-action-formaction", actionFormAction);
      } else {
        form.removeAttribute("data-studio-field-action-formaction");
      }
      var selectionKey = detail.blockKey || detail.nodeID || detail.field || "";
      if (selectionKey) form.setAttribute("data-studio-selection", selectionKey);
      else form.removeAttribute("data-studio-selection");
      form.setAttribute("data-studio-selection-kind", detail.field ? "preview-field" : "preview");
      setReadout("[data-studio-selection-label]", detail.label || detail.field || detail.blockKey || "Preview selection");
      setReadout("[data-studio-selection-status]", detail.field ? "Preview field" : "Preview selection");
      setReadout("[data-studio-field-selection-label]", detail.field ? (detail.label || readableFieldName(detail.field, selectedEditable)) : "Block");
      previewShells().forEach(function (shell) {
        shell.setAttribute("data-gosx-studio-preview-selection", detail.field || detail.blockKey || detail.nodeID || "");
      });
      frame.setAttribute("data-studio-preview-selection", detail.field || detail.blockKey || detail.nodeID || "");
      syncPreviewDock(frame, selectedTargets[0] || target, {
        field: detail.field || "",
        editable: selectedEditable,
        label: detail.label || "",
        blockLabel: detail.blockLabel || "",
        action: actionLabel || "",
        actionHref: actionHref || "",
        actionFormAction: actionFormAction || "",
        blockKey: detail.blockKey || "",
        nodeID: detail.nodeID || ""
      });
      if (options.reveal && source) revealInspectorSelection(source, control);
      emit(form, "gosxstudio:preview-select", {
        field: detail.field || "",
        source: detail.source || detail.field || "",
        editable: selectedEditable,
        label: detail.label || "",
        blockLabel: detail.blockLabel || "",
        action: actionLabel || "",
        actionHref: actionHref || "",
        actionFormAction: actionFormAction || "",
        blockKey: detail.blockKey || "",
        nodeID: detail.nodeID || "",
        reason: options.reason || "preview"
      });
      emitEditorOperation("select_preview", {
        mutation: false,
        reason: options.reason || "preview",
        target: {
          field: detail.field || "",
          editable: selectedEditable,
          blockKey: detail.blockKey || "",
          nodeID: detail.nodeID || "",
          selection: selectionKey,
          kind: detail.field ? "preview-field" : "preview"
        },
        payload: {
          label: detail.label || "",
          blockLabel: detail.blockLabel || "",
          action: actionLabel || "",
          actionHref: actionHref || "",
          actionFormAction: actionFormAction || ""
        }
      });
      return true;
    }

    function updatePreviewTarget(target, field) {
      if (!target || !field) return;
      var value = field.value == null ? "" : String(field.value);
      var tag = String(target.tagName || "").toLowerCase();
      var editable = target.getAttribute("data-studio-editable") || target.getAttribute("data-studio-field-editable") || "";
      if (tag === "input" || tag === "textarea" || tag === "select") {
        target.value = value;
        if (field.type === "checkbox" || field.type === "radio") target.checked = !!field.checked;
      } else if (tag === "img" || editable === "media" || editable === "image") {
        if (value) target.setAttribute("src", value);
      } else if (tag === "a" && (editable === "url" || field.name.toLowerCase().indexOf("url") >= 0)) {
        target.setAttribute("href", value || "#");
      } else {
        target.textContent = value;
      }
      target.setAttribute("data-gosx-studio-preview-patched", "fresh");
      window.setTimeout(function () {
        if (target && target.setAttribute) target.setAttribute("data-gosx-studio-preview-patched", "true");
      }, 220);
    }

    function applyPreviewPatch(frame, patch) {
      var doc = frameDocument(frame);
      if (!doc || !patch || !patch.field) return 0;
      ensurePreviewPatchStyles(doc);
      var targets = previewTargets(frame, patch);
      targets.forEach(function (target) {
        updatePreviewTarget(target, patch.field);
      });
      if (targets.length) {
        frame.setAttribute("data-studio-preview-patched-count", String(targets.length));
      }
      return targets.length;
    }

    function syncPreviewFrame(frame, reason) {
      if (!frameDocument(frame)) return 0;
      var count = 0;
      queryAll(form, "[data-studio-field-source], [data-editor-source]").forEach(function (field) {
        var patch = {
          type: "gosxstudio:preview-patch",
          source: "gosx-studio",
          reason: reason || "sync",
          detail: {},
          field: fieldPatch(field)
        };
        count += applyPreviewPatch(frame, patch);
      });
      if (count) emit(form, "gosxstudio:preview-sync", { count: count, reason: reason || "sync" });
      return count;
    }

    function setPreviewStatus(state, label, reason) {
      previewShells().forEach(function (shell) {
        shell.setAttribute("data-gosx-studio-preview-state", state);
        shell.setAttribute("data-gosx-studio-preview-reason", reason || "");
        queryAll(shell, "[data-studio-preview-status]").forEach(function (node) {
          node.textContent = label;
        });
      });
      previewFrames().forEach(function (frame) {
        frame.setAttribute("data-studio-preview-state", state);
      });
    }

    function cacheBustURL(url, reason) {
      try {
        var next = new URL(url || window.location.href, window.location.href);
        next.searchParams.set("_gosx_preview", String(Date.now()));
        if (reason) next.searchParams.set("_gosx_preview_reason", reason);
        return next.pathname + next.search + next.hash;
      } catch (error) {
        return url || "";
      }
    }

    function openLinks() {
      return queryAll(form, "[data-studio-open-preview]");
    }

    function syncPreviewRoute(route, reason) {
      route = route || "";
      if (!route) return;
      previewShells().forEach(function (shell) {
        shell.setAttribute("data-gosx-studio-preview-url", route);
      });
      previewFrames().forEach(function (frame) {
        frame.setAttribute("data-studio-preview-src", route);
      });
      openLinks().forEach(function (link) {
        if (link.getAttribute("aria-disabled") === "true") return;
        link.setAttribute("href", route);
      });
      queryAll(form, "[data-studio-selected-flow-route]").forEach(function (node) {
        node.textContent = route;
      });
      emit(form, "gosxstudio:preview-route", { route: route, reason: reason || "" });
    }

    function refreshPreviewNow(reason, route) {
      if (route) syncPreviewRoute(route, reason);
      var frames = previewFrames();
      if (!frames.length) return;
      setPreviewStatus("loading", "Refreshing preview", reason || "refresh");
      frames.forEach(function (frame) {
        var base = route || previewURL(frame) || frame.getAttribute("src") || "/";
        frame.setAttribute("src", cacheBustURL(base, reason || "refresh"));
      });
      emit(form, "gosxstudio:preview-refresh", { route: route || "", reason: reason || "refresh" });
    }

    function schedulePreviewRefresh(reason, route) {
      window.clearTimeout(previewRefreshTimer);
      previewRefreshTimer = window.setTimeout(function () {
        refreshPreviewNow(reason, route);
      }, 180);
    }

    function fieldPatch(field) {
      if (!field || !field.name || field.disabled) return null;
      var type = String(field.type || "").toLowerCase();
      if (field.name === "csrf_token" || type === "button" || type === "submit" || type === "reset" || type === "file") return null;
      return {
        name: field.name,
        source: field.getAttribute("data-studio-field-source") || field.getAttribute("data-editor-source") || field.name,
        editable: field.getAttribute("data-studio-field-editable") || "",
        value: type === "checkbox" || type === "radio" ? (field.checked ? (field.value || "on") : "") : (field.value || ""),
        checked: !!field.checked,
        type: type,
        tag: String(field.tagName || "").toLowerCase()
      };
    }

    function emitFieldOperation(reason, field) {
      var patch = fieldPatch(field);
      if (!patch) return null;
      return emitEditorOperation("set_field", {
        mutation: true,
        reason: reason || "field",
        target: {
          field: patch.source || patch.name || "",
          name: patch.name || "",
          editable: patch.editable || "",
          selection: form.getAttribute("data-studio-selection") || "",
          kind: form.getAttribute("data-studio-selection-kind") || ""
        },
        payload: {
          field: patch
        }
      });
    }

    function textControlForField(field) {
      var source = inspectorSource(field);
      var control = inspectorControl(source);
      if (control && "value" in control) return control;
      if (source && "value" in source) return source;
      return null;
    }

    function placeCaretAtEnd(doc, node) {
      try {
        node.focus();
        var selection = doc.defaultView && doc.defaultView.getSelection ? doc.defaultView.getSelection() : null;
        if (!selection || !doc.createRange) return;
        var range = doc.createRange();
        range.selectNodeContents(node);
        range.collapse(false);
        selection.removeAllRanges();
        selection.addRange(range);
      } catch (error) {
        return;
      }
    }

    function inlineTextPayload(edit, text) {
      return {
        text: text,
        field: edit.field || "",
        previous: edit.originalValue || "",
        label: edit.label || ""
      };
    }

    function emitInlineTextEvent(name, edit, reason, text) {
      emit(form, name, {
        field: edit.field || "",
        editable: "text",
        blockKey: edit.blockKey || "",
        label: edit.label || "",
        text: text == null ? (edit.target && edit.target.textContent) || "" : text,
        reason: reason || ""
      });
    }

    function syncInlineTextEdit(frame, reason) {
      var edit = frame && frame.__gosxStudioInlineEdit;
      if (!edit || !edit.target) return false;
      var text = edit.target.textContent || "";
      if (edit.control && "value" in edit.control) edit.control.value = text;
      if (edit.lastText === text) return true;
      edit.lastText = text;
      setPreviewStatus("dirty", "Draft changed", reason || "inline-text");
      emitEditorOperation("set_text", {
        mutation: true,
        reason: reason || "inline-text",
        target: {
          field: edit.field || "",
          editable: "text",
          blockKey: edit.blockKey || "",
          selection: form.getAttribute("data-studio-selection") || edit.blockKey || edit.field || "",
          kind: form.getAttribute("data-studio-selection-kind") || "preview-field"
        },
        payload: inlineTextPayload(edit, text)
      });
      emitInlineTextEvent("gosxstudio:inline-text", edit, reason || "inline-text", text);
      return true;
    }

    function startInlineTextEdit(frame, detail, reason) {
      var target = frame && frame.__gosxStudioPreviewDockTarget;
      detail = detail || (frame && frame.__gosxStudioPreviewDock ? previewDockDetail(frame.__gosxStudioPreviewDock) : {});
      if (!frame || !target || !detail.field || detail.editable !== "text") return false;
      finishInlineTextEdit(frame, true, "restart-inline-text");
      var doc = frameDocument(frame);
      if (!doc) return false;
      var startReason = reason || "preview-dock";
      var control = textControlForField(detail.field);
      var text = target.textContent || "";
      frame.__gosxStudioInlineEdit = {
        target: target,
        field: detail.field || "",
        blockKey: detail.blockKey || "",
        label: detail.label || "",
        control: control,
        originalText: text,
        originalValue: control && "value" in control ? control.value || "" : text,
        lastText: text
      };
      target.setAttribute("contenteditable", "plaintext-only");
      target.setAttribute("spellcheck", "true");
      target.setAttribute("data-gosx-studio-inline-editing", "true");
      form.setAttribute("data-gosx-studio-inline-field", detail.field);
      placeCaretAtEnd(doc, target);
      emitEditorOperation("inline_text_start", {
        mutation: false,
        reason: startReason,
        target: {
          field: detail.field || "",
          editable: "text",
          blockKey: detail.blockKey || "",
          selection: form.getAttribute("data-studio-selection") || detail.blockKey || detail.field || "",
          kind: form.getAttribute("data-studio-selection-kind") || "preview-field"
        },
        payload: {
          label: detail.label || ""
        }
      });
      emitInlineTextEvent("gosxstudio:inline-text-start", frame.__gosxStudioInlineEdit, startReason, text);
      return true;
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
      var edit = frame && frame.__gosxStudioInlineEdit;
      if (!edit || !edit.target) return false;
      if (edit.finishing) return false;
      edit.finishing = true;
      if (commit) {
        syncInlineTextEdit(frame, reason || "commit");
        emitInlineTextEvent("gosxstudio:inline-text-commit", edit, reason || "commit", edit.target.textContent || "");
      } else {
        edit.target.textContent = edit.originalText || "";
        if (edit.control && "value" in edit.control) edit.control.value = edit.originalValue || "";
        emitEditorOperation("inline_text_cancel", {
          mutation: false,
          reason: reason || "cancel",
          target: {
            field: edit.field || "",
            editable: "text",
            blockKey: edit.blockKey || ""
          },
          payload: inlineTextPayload(edit, edit.originalValue || "")
        });
        emitInlineTextEvent("gosxstudio:inline-text-cancel", edit, reason || "cancel", edit.originalValue || "");
      }
      edit.target.removeAttribute("contenteditable");
      edit.target.removeAttribute("data-gosx-studio-inline-editing");
      form.removeAttribute("data-gosx-studio-inline-field");
      frame.__gosxStudioInlineEdit = null;
      updatePreviewDockPosition(frame);
      return true;
    }

    function postPreviewPatch(reason, detail, field) {
      var frames = previewFrames();
      if (!frames.length) return;
      var patch = {
        type: "gosxstudio:preview-patch",
        source: "gosx-studio",
        reason: reason || "patch",
        detail: detail || {},
        field: fieldPatch(field)
      };
      frames.forEach(function (frame) {
        if (!frame.contentWindow || !frame.getAttribute("src")) return;
        applyPreviewPatch(frame, patch);
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
      if (reason !== "load-sync") setPreviewStatus("dirty", "Live preview pending", reason || "patch");
      emit(form, "gosxstudio:preview-patch", patch);
    }

    function bindPreviewFrames() {
      previewFrames().forEach(function (frame) {
        if (frame.dataset.gosxStudioPreviewBound === "true") return;
        frame.dataset.gosxStudioPreviewBound = "true";
        if (!frame.getAttribute("data-studio-preview-src")) {
          frame.setAttribute("data-studio-preview-src", frame.getAttribute("src") || "");
        }
        frame.addEventListener("load", function () {
          setPreviewStatus("ready", "Ready", "load");
          syncPreviewFrame(frame, "load");
          bindPreviewDocument(frame);
          postPreviewPatch("load-sync", { route: previewURL(frame) || frame.getAttribute("src") || "" }, null);
        });
        frame.addEventListener("error", function () {
          setPreviewStatus("error", "Preview failed", "error");
        });
        bindPreviewDocument(frame);
      });
    }

    function bindPreviewDocument(frame) {
      var doc = frameDocument(frame);
      if (!doc || frame.__gosxStudioPreviewDocument === doc) return;
      frame.__gosxStudioPreviewDocument = doc;
      ensurePreviewPatchStyles(doc);
      if (doc.documentElement) doc.documentElement.setAttribute("data-gosx-studio-preview-selectable", "true");
      var repositionDock = frameTask(function () {
        updatePreviewDockPosition(frame);
      });
      doc.addEventListener("click", function (event) {
        var edit = frame.__gosxStudioInlineEdit;
        if (edit && edit.target && (event.target === edit.target || (edit.target.contains && edit.target.contains(event.target)))) return;
        if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button > 0) return;
        var target = previewSelectableNode(event.target);
        if (!target) return;
        if (applyPreviewSelection(frame, target, previewSelectionDetail(target), { reveal: true, reason: "click" })) {
          event.preventDefault();
          event.stopPropagation();
        }
      }, true);
      doc.addEventListener("dblclick", function (event) {
        var edit = frame.__gosxStudioInlineEdit;
        if (edit && edit.target && (event.target === edit.target || (edit.target.contains && edit.target.contains(event.target)))) return;
        if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button > 0) return;
        var target = previewSelectableNode(event.target);
        if (!target) return;
        var detail = previewSelectionDetail(target);
        if (detail.editable !== "text") return;
        if (applyPreviewSelection(frame, target, detail, { reveal: true, reason: "double-click" }) && startInlineTextFromDetail(frame, detail, "double-click")) {
          event.preventDefault();
          event.stopPropagation();
        }
      }, true);
      doc.addEventListener("focusin", function (event) {
        var edit = frame.__gosxStudioInlineEdit;
        if (edit && event.target === edit.target) return;
        var target = previewSelectableNode(event.target);
        if (target) applyPreviewSelection(frame, target, previewSelectionDetail(target), { reveal: false, reason: "focus" });
      }, true);
      doc.addEventListener("input", function (event) {
        var edit = frame.__gosxStudioInlineEdit;
        if (edit && event.target === edit.target) syncInlineTextEdit(frame, "input");
      });
      doc.addEventListener("keydown", function (event) {
        var edit = frame.__gosxStudioInlineEdit;
        if (!edit || event.target !== edit.target) return;
        if (event.key === "Escape") {
          event.preventDefault();
          finishInlineTextEdit(frame, false, "escape");
          return;
        }
        if (event.key === "Enter" && !event.shiftKey) {
          event.preventDefault();
          finishInlineTextEdit(frame, true, "enter");
        }
        return;
      });
      doc.addEventListener("keydown", function (event) {
        if (frame.__gosxStudioInlineEdit) return;
        if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return;
        var focusedControl = event.target && event.target.closest ? event.target.closest("input, textarea, select, button, a[href], [contenteditable='true'], [contenteditable='plaintext-only']") : null;
        if (focusedControl) return;
        if (event.key === "[" || event.key === "]") {
          if (navigatePreviewField(frame, event.key === "]" ? 1 : -1, event.key === "]" ? "keyboard-next-field" : "keyboard-prev-field")) {
            event.preventDefault();
            event.stopPropagation();
          }
          return;
        }
        if (event.key !== "Enter" && event.key !== "F2") return;
        if (startInlineTextFromSelection(frame, event.key === "F2" ? "keyboard-f2" : "keyboard-enter")) {
          event.preventDefault();
          event.stopPropagation();
        }
      });
      doc.addEventListener("paste", function (event) {
        var edit = frame.__gosxStudioInlineEdit;
        if (!edit || event.target !== edit.target) return;
        var text = event.clipboardData && event.clipboardData.getData ? event.clipboardData.getData("text/plain") : "";
        if (!text) return;
        event.preventDefault();
        try {
          var selection = doc.defaultView && doc.defaultView.getSelection ? doc.defaultView.getSelection() : null;
          if (!selection || !selection.rangeCount) return;
          selection.deleteFromDocument();
          selection.getRangeAt(0).insertNode(doc.createTextNode(text));
          selection.collapseToEnd();
          syncInlineTextEdit(frame, "paste");
        } catch (error) {
          return;
        }
      });
      doc.addEventListener("blur", function (event) {
        var edit = frame.__gosxStudioInlineEdit;
        if (edit && event.target === edit.target) finishInlineTextEdit(frame, true, "blur");
      }, true);
      doc.addEventListener("scroll", repositionDock, true);
      if (frame.contentWindow) frame.contentWindow.addEventListener("resize", repositionDock);
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

    function runInsert(button) {
      var target = button.getAttribute("data-studio-insert-block") || button.getAttribute("data-editor-add-block") || "";
      var detail = { target: target, label: labelFromButton(button, target), button: button };
      emit(form, "gosxstudio:insert-block", detail);
      emit(form, "gosxstudio:workbench-action", { action: "insert-block", target: target, label: detail.label });
      var add = target ? form.querySelector('[data-editor-add-block="' + attrValue(target) + '"]') : null;
      if (add && add !== button && add.click) add.click();
    }

    function runSelectionAction(button) {
      var action = button.getAttribute("data-studio-selection-action") || "";
      var detail = {
        action: action,
        label: labelFromButton(button, action),
        selection: form.getAttribute("data-studio-selection") || form.getAttribute("data-gosx-studio-canvas-selected") || "",
        kind: form.getAttribute("data-studio-selection-kind") || ""
      };
      emit(form, "gosxstudio:selection-action", detail);
      emit(form, "gosxstudio:workbench-action", detail);
    }

    function runInsertTarget(target, label) {
      target = target || "";
      if (!target) return false;
      var button = form.querySelector('[data-studio-insert-block="' + attrValue(target) + '"], [data-editor-add-block="' + attrValue(target) + '"]');
      if (button) {
        runInsert(button);
        return true;
      }
      emit(form, "gosxstudio:insert-block", { target: target, label: label || target });
      emit(form, "gosxstudio:workbench-action", { action: "insert-block", target: target, label: label || target });
      return true;
    }

    function runSelectionTarget(target, label) {
      target = target || "";
      if (!target) return false;
      var button = form.querySelector('[data-studio-selection-action="' + attrValue(target) + '"]');
      if (button) {
        runSelectionAction(button);
        return true;
      }
      emit(form, "gosxstudio:selection-action", {
        action: target,
        label: label || target,
        selection: form.getAttribute("data-studio-selection") || "",
        kind: form.getAttribute("data-studio-selection-kind") || ""
      });
      emit(form, "gosxstudio:workbench-action", { action: target, label: label || target });
      return true;
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
      if (detail.kind === "insert") return runInsertTarget(detail.target, detail.label);
      if (detail.kind === "selection-action") return runSelectionTarget(detail.target, detail.label);
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
      var insert = event.target.closest("[data-studio-insert-block]");
      if (insert && form.contains(insert)) {
        runInsert(insert);
        return;
      }
      var selectionAction = event.target.closest("[data-studio-selection-action]");
      if (selectionAction && form.contains(selectionAction)) {
        event.preventDefault();
        runSelectionAction(selectionAction);
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
      postPreviewPatch("input", {}, event.target);
    });

    form.addEventListener("change", function (event) {
      if (!event.target || !form.contains(event.target)) return;
      emitFieldOperation("change", event.target);
      postPreviewPatch("change", {}, event.target);
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
        clearInspectorSelection();
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
