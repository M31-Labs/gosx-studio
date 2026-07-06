// island_runtime.js — editor-side writer for the previewruntime islands.
//
// Phase 3 slice-6 burn-down of GoSXStudioPreviewRuntime (10 methods at
// gosx-studio/assets/preview-runtime.js:1118). This script runs on the
// editor page; it publishes the
// window.__gosx_preview_runtime_island_<method> globals that
// previewruntime.BridgeShim delegates to directly. Each writer publishes
// to a $preview.* shared signal via window.__gosx_set_shared_signal_json;
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
// gosx-studio/runtime.go appends Bundle() to EngineRuntimeScript(). This
// island_runtime.js runs first and publishes the
// __gosx_preview_runtime_island_* globals; finally BridgeShim runs and
// exposes window.GoSXStudioPreviewRuntime as a direct delegating shim.

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
  // If the bridge is not ready, the write is a no-op. The next host call
  // can publish once the shared-signal bridge is available.
  function writeSharedSignal(name, value) {
    try {
      var setter = window.__gosx_set_shared_signal_json;
      if (typeof setter === "function") {
        setter(name, JSON.stringify(value));
      }
    } catch (error) {
      // Bridge not ready yet; defer to the next host-driven publish.
    }
  }

  function queryAll(root, selector) {
    if (!root || !root.querySelectorAll) return [];
    return Array.prototype.slice.call(root.querySelectorAll(selector));
  }

  function editorPreviewClamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function compactText(value) {
    return String(value || "").replace(/\s+/g, " ").trim();
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

  function editorPreviewShellForFrame(frame) {
    return frame && frame.closest ? frame.closest("[data-gosx-studio-preview], [data-studio-preview-shell], [data-studio-canvas]") : null;
  }

  function editorPreviewDockForFrame(frame) {
    var shell = editorPreviewShellForFrame(frame);
    if (!shell) return { dock: null, shell: null };
    return {
      dock: shell.querySelector("[data-studio-preview-dock], [data-gosx-studio-preview-dock]"),
      shell: shell
    };
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

  function editorPreviewSelectableNode(target) {
    if (!target || !target.closest) return null;
    return target.closest("[data-studio-field], [data-editor-preview], [data-studio-field-source], [data-studio-block-key], [data-studio-node-id]");
  }

  function editorPreviewFieldKeyForTarget(target) {
    if (!target || !target.getAttribute) return "";
    return target.getAttribute("data-studio-field") || target.getAttribute("data-editor-preview") || target.getAttribute("data-studio-field-source") || "";
  }

  function editorPreviewFieldKey(target) {
    return editorPreviewFieldKeyForTarget(target);
  }

  function editorPreviewFieldNavigationScope(frame, target, detail) {
    var frameDoc = editorPreviewFrameDocument(frame);
    if (!frameDoc) return null;
    if (target && target.closest) {
      var targetBlock = target.closest("[data-studio-block-key], [data-studio-node-id]");
      if (targetBlock) return targetBlock;
    }
    var blockKey = detail && (detail.blockKey || detail.nodeID);
    if (blockKey) {
      var selector = '[data-studio-block-key="' + attrValue(blockKey) + '"], [data-studio-node-id="' + attrValue(blockKey) + '"]';
      return frameDoc.querySelector(selector) || frameDoc.body || frameDoc.documentElement;
    }
    return frameDoc.body || frameDoc.documentElement;
  }

  function editorPreviewFieldNodesForSelection(frame, target, detail) {
    var scope = editorPreviewFieldNavigationScope(frame, target, detail);
    if (!scope) return [];
    var seen = {};
    return queryAll(scope, "[data-studio-field], [data-editor-preview], [data-studio-field-source]").filter(function (candidate) {
      var key = editorPreviewFieldKeyForTarget(candidate);
      if (!key || seen[key]) return false;
      seen[key] = true;
      return true;
    });
  }

  function editorPreviewFieldNavigationState(frame, target, detail) {
    var scope = editorPreviewFieldNavigationScope(frame, target, detail);
    if (!frame || !scope) return { handled: false, fields: [], count: 0, index: -1, current: "" };
    var fields = editorPreviewFieldNodesForSelection(frame, target, detail);
    var current = detail && detail.field ? detail.field : editorPreviewFieldKeyForTarget(target);
    var index = -1;
    fields.forEach(function (candidate, candidateIndex) {
      if (index < 0 && editorPreviewFieldKeyForTarget(candidate) === current) index = candidateIndex;
    });
    return {
      handled: true,
      fields: fields,
      count: fields.length,
      index: index,
      current: current
    };
  }

  function runEditorPreviewFieldNavigation(detail, event) {
    detail = detail || {};
    var frame = detail.frame;
    var direction = Number(detail.direction) || 0;
    var reason = detail.reason || "field-navigation";
    var host = detail.host || {};
    var dock = frame && frame.__gosxStudioPreviewDock;
    var target = frame && frame.__gosxStudioPreviewDockTarget;
    if (!dock || dock.hidden || !target) return { handled: false, navigated: false };
    editorPreviewHostCall(host, "finishInlineTextEdit", false, frame, true, "field-navigation");
    var selection = editorPreviewHostCall(host, "previewSelectionDetail", {}, target) || {};
    var state = editorPreviewFieldNavigationState(frame, target, selection);
    if (!state.count) return { handled: false, navigated: false, state: state };
    var currentIndex = state.index >= 0 ? state.index : (direction > 0 ? -1 : 0);
    var nextIndex = (currentIndex + direction + state.count) % state.count;
    var nextTarget = state.fields[nextIndex];
    var nextDetail = editorPreviewHostCall(host, "previewSelectionDetail", {}, nextTarget) || {};
    if (!nextTarget || !nextDetail.field) return { handled: false, navigated: false, state: state };
    if (!editorPreviewHostCall(host, "applyPreviewSelection", false, frame, nextTarget, nextDetail, { reveal: true, reason: reason })) {
      return { handled: false, navigated: false, state: state, detail: nextDetail };
    }
    var navigation = {
      direction: direction > 0 ? "next" : "previous",
      fieldIndex: nextIndex + 1,
      fieldCount: state.count,
      reason: reason
    };
    editorPreviewHostCall(host, "dispatchPreviewFieldNavigation", null, nextDetail, navigation);
    return { handled: true, navigated: true, detail: nextDetail, navigation: navigation, state: state };
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

  function clearEditorPreviewSelections(form, host) {
    if (!form) return { handled: false, frames: 0, fieldMaps: 0, markers: 0, chrome: null, docks: null };
    var frames = editorPreviewFrames(form);
    var fieldMaps = 0;
    var markers = 0;
    frames.forEach(function (frame) {
      editorPreviewHostCall(host, "finishInlineTextEdit", null, frame, true, "clear-selection");
    });
    frames.forEach(function (frame) {
      var fieldMap = clearEditorPreviewFieldMap(frame);
      var marker = clearEditorPreviewSelectionMarker(frame);
      fieldMaps += fieldMap && fieldMap.count ? fieldMap.count : 0;
      markers += marker && marker.count ? marker.count : 0;
    });
    var chrome = clearEditorPreviewSelectionChrome(form);
    var docks = hideEditorPreviewDocks(form);
    return { handled: true, frames: frames.length, fieldMaps: fieldMaps, markers: markers, chrome: chrome, docks: docks };
  }

  function applyEditorPreviewSelectionSync(detail, event) {
    detail = detail || {};
    var form = editorPreviewForm(detail, event);
    var frame = detail.frame;
    var target = detail.target;
    var selection = detail.selection || detail.detail || {};
    var options = detail.options || {};
    var host = detail.host || {};
    if (!selection.field && !selection.blockKey && !selection.nodeID) {
      return { handled: false, applied: null, targets: [] };
    }
    var cleared = clearEditorPreviewSelections(form, host);
    var selectedTargets = selection.field ? editorPreviewPatchTargets(frame, { field: { source: selection.field, name: selection.field } }) : [];
    if (!selectedTargets.length && target) selectedTargets = [target];
    var marker = applyEditorPreviewSelectionMarker(frame, selectedTargets, true);
    var applied = editorPreviewHostCall(host, "dispatchPreviewSelectionApply", null, selection, options);
    if (!applied) {
      return { handled: false, applied: null, targets: selectedTargets, marker: marker, cleared: cleared };
    }
    var chrome = applyEditorPreviewSelectionChrome(form, frame, selection.field || selection.blockKey || selection.nodeID || "");
    var dockSelection = editorPreviewDockSelection(selection, applied);
    var dock = syncEditorPreviewDockSelection(form, frame, selectedTargets[0] || target, dockSelection, host);
    return { handled: true, applied: applied, targets: selectedTargets, marker: marker, chrome: chrome, dock: dock, cleared: cleared };
  }

  function normalizeEditorPreviewDock(dock) {
    if (!dock || !dock.setAttribute || !dock.querySelector) return { handled: false, commands: 0 };
    if (!dock.hasAttribute("data-gosx-studio-preview-dock")) {
      dock.setAttribute("data-gosx-studio-preview-dock", "true");
    }
    var title = dock.querySelector("[data-studio-preview-dock-title]");
    if (title && !title.hasAttribute("data-gosx-studio-preview-dock-label")) {
      title.setAttribute("data-gosx-studio-preview-dock-label", "true");
    }
    var fieldCount = dock.querySelector("[data-studio-preview-field-count]");
    if (fieldCount && !fieldCount.hasAttribute("data-gosx-studio-preview-field-meter")) {
      fieldCount.setAttribute("data-gosx-studio-preview-field-meter", "true");
    }
    var actions = queryAll(dock, "[data-studio-preview-action]");
    actions.forEach(function (button) {
      if (!button.hasAttribute("data-gosx-studio-preview-command")) {
        button.setAttribute("data-gosx-studio-preview-command", button.getAttribute("data-studio-preview-action") || "");
      }
    });
    return { handled: true, commands: actions.length };
  }

  function runEditorPreviewDockAction(detail, event) {
    detail = detail || {};
    var form = editorPreviewForm(detail, event);
    var frame = detail.frame;
    var action = detail.action || "";
    var host = detail.host || {};
    var dock = frame && frame.__gosxStudioPreviewDock;
    if (!dock || !action) return { handled: false, action: action, detail: {}, result: false };
    var dockDetail = editorPreviewDockDetail(form, dock);
    if (action === "clear") {
      editorPreviewHostCall(host, "clearPreviewSelections", null);
      editorPreviewHostCall(host, "dispatchPreviewSelectionClear", null, "preview-dock");
      editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail);
      return { handled: true, action: action, detail: dockDetail, result: true };
    }
    if (action === "content" || action === "style") {
      var dockIntent = editorPreviewHostCall(host, "dispatchPreviewDockActionResolve", null, action, dockDetail) || { mode: action, reveal: action === "content" && !!dockDetail.field };
      if (dockIntent.mode) editorPreviewHostCall(host, "setMode", null, dockIntent.mode, { scroll: true, reason: "preview-dock" });
      if (dockIntent.reveal) {
        editorPreviewHostCall(host, "revealPreviewField", null, dockDetail, "preview-dock");
      }
      editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail);
      return { handled: true, action: action, detail: dockDetail, result: true };
    }
    if (action === "prev-field" || action === "next-field") {
      var navigation = runEditorPreviewFieldNavigation({
        form: form,
        frame: frame,
        direction: action === "next-field" ? 1 : -1,
        reason: "preview-dock",
        host: host
      }, event);
      if (navigation && navigation.navigated) {
        dockDetail = editorPreviewDockDetail(form, dock);
        editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail);
        return { handled: true, action: action, detail: dockDetail, result: true, navigation: navigation };
      }
      editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail);
      return { handled: true, action: action, detail: dockDetail, result: false, navigation: navigation };
    }
    if (action === "field-action") {
      var intent = editorPreviewHostCall(host, "dispatchPreviewFieldActionResolve", null, dockDetail) || { reveal: !!dockDetail.field };
      if (intent.inlineText && editorPreviewHostCall(host, "startInlineTextFromDetail", false, frame, dockDetail, "preview-dock")) {
        return { handled: true, action: action, detail: dockDetail, result: true };
      }
      if (intent.submit && editorPreviewHostCall(host, "submitPreviewFieldAction", false, dockDetail)) {
        editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail);
        return { handled: true, action: action, detail: dockDetail, result: true };
      }
      if (intent.navigate) {
        if (editorPreviewHostCall(host, "navigateToHref", null, intent.href || "") === null) {
          window.location.href = intent.href || "";
        }
      } else if (intent.reveal) {
        editorPreviewHostCall(host, "revealPreviewField", null, dockDetail, "preview-dock");
      }
      editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail);
      return { handled: true, action: action, detail: dockDetail, result: true };
    }
    editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail);
    return { handled: true, action: action, detail: dockDetail, result: true };
  }

  function bindEditorPreviewDock(form, frame, dock, host) {
    var normalize = normalizeEditorPreviewDock(dock);
    if (!dock || !dock.addEventListener) {
      return { handled: false, bound: false, commands: normalize.commands || 0 };
    }
    dock.__gosxStudioPreviewDockFrame = frame;
    dock.__gosxStudioPreviewDockHost = host;
    if (!dock.__gosxStudioPreviewDockHandler) {
      dock.__gosxStudioPreviewDockHandler = true;
      dock.addEventListener("click", function (event) {
        var button = event.target && event.target.closest ? event.target.closest("[data-gosx-studio-preview-command], [data-studio-preview-action]") : null;
        if (!button || !dock.contains(button)) return;
        event.preventDefault();
        var command = button.getAttribute("data-gosx-studio-preview-command") || button.getAttribute("data-studio-preview-action") || "";
        var actionHost = dock.__gosxStudioPreviewDockHost || host;
        runEditorPreviewDockAction({ form: form, frame: dock.__gosxStudioPreviewDockFrame || frame, action: command, host: actionHost }, event);
      });
      return { handled: true, bound: true, commands: normalize.commands || 0 };
    }
    return { handled: true, bound: false, commands: normalize.commands || 0 };
  }

  function editorPreviewDockKindLabel(selection) {
    if (!selection || !selection.field) return "Block";
    var editable = selection.editable || "";
    if (editable === "media" || editable === "image") return "Media field";
    if (editable === "source") return "Source field";
    if (editable === "flow") return "Flow field";
    if (editable === "url" || editable === "link") return "Link field";
    if (editable === "text") return "Text field";
    return "Field";
  }

  function editorPreviewDockSelection(selection, applied) {
    selection = selection || {};
    applied = applied || {};
    return {
      field: applied.field || selection.field || "",
      editable: applied.editable || selection.editable || "",
      label: applied.label || selection.label || "",
      blockLabel: applied.blockLabel || selection.blockLabel || "",
      action: applied.action || selection.action || "",
      actionHref: applied.actionHref || selection.actionHref || "",
      actionFormAction: applied.actionFormAction || selection.actionFormAction || "",
      blockKey: applied.blockKey || selection.blockKey || "",
      nodeID: applied.nodeID || selection.nodeID || ""
    };
  }

  function syncEditorPreviewDock(dock, selection) {
    if (!dock || !dock.setAttribute) return { handled: false };
    selection = editorPreviewDockSelection(selection);
    dock.hidden = false;
    dock.setAttribute("data-gosx-studio-preview-field", selection.field || "");
    dock.setAttribute("data-gosx-studio-preview-block", selection.blockKey || selection.nodeID || "");
    dock.setAttribute("data-gosx-studio-preview-block-label", selection.blockLabel || "");
    dock.setAttribute("data-gosx-studio-preview-action-label", selection.action || "");
    dock.setAttribute("data-gosx-studio-preview-action-href", selection.actionHref || "");
    dock.setAttribute("data-gosx-studio-preview-action-formaction", selection.actionFormAction || "");
    var label = dock.querySelector("[data-gosx-studio-preview-dock-label]");
    if (label) label.textContent = selection.label || selection.field || selection.blockKey || "Preview selection";
    var breadcrumb = dock.querySelector("[data-gosx-studio-preview-breadcrumb]");
    if (breadcrumb) {
      breadcrumb.hidden = !selection.blockLabel || !selection.field;
      breadcrumb.textContent = selection.blockLabel && selection.field ? selection.blockLabel + " / " + (selection.label || selection.field) : "";
    }
    var kind = dock.querySelector("[data-gosx-studio-preview-dock-kind]");
    if (kind) kind.textContent = editorPreviewDockKindLabel(selection);
    var action = dock.querySelector('[data-gosx-studio-preview-command="field-action"]');
    if (action) {
      action.textContent = selection.action || (selection.editable === "text" ? "Edit text" : selection.editable === "media" || selection.editable === "image" ? "Media" : selection.editable === "flow" ? "Flow" : selection.editable === "source" ? "Source" : "Open");
      action.disabled = !selection.field && !selection.action && !selection.actionHref && !selection.actionFormAction;
    }
    return { handled: true };
  }

  function editorPreviewDockDetail(form, dock) {
    if (!dock || !dock.getAttribute || !dock.querySelector) {
      return { field: "", blockKey: "", blockLabel: "", label: "", action: "", actionHref: "", actionFormAction: "", editable: "" };
    }
    var label = dock.querySelector("[data-gosx-studio-preview-dock-label]");
    return {
      field: dock.getAttribute("data-gosx-studio-preview-field") || "",
      blockKey: dock.getAttribute("data-gosx-studio-preview-block") || "",
      blockLabel: dock.getAttribute("data-gosx-studio-preview-block-label") || "",
      label: compactText(label && label.textContent),
      action: dock.getAttribute("data-gosx-studio-preview-action-label") || "",
      actionHref: dock.getAttribute("data-gosx-studio-preview-action-href") || "",
      actionFormAction: dock.getAttribute("data-gosx-studio-preview-action-formaction") || "",
      editable: form ? form.getAttribute("data-studio-field-editable") || "" : ""
    };
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

  function syncEditorPreviewDockPosition(frame, dock, target, shell) {
    shell = shell || editorPreviewShellForFrame(frame);
    if (!frame || !dock || !target || !shell || dock.hidden) return { handled: false, placement: "" };
    var frameRect = frame.getBoundingClientRect();
    var shellRect = shell.getBoundingClientRect();
    var targetRect = target.getBoundingClientRect();
    var left = frameRect.left - shellRect.left + targetRect.left + (targetRect.width / 2);
    var top = frameRect.top - shellRect.top + targetRect.top;
    var maxLeft = Math.max(8, shellRect.width - 8);
    left = editorPreviewClamp(left, 8, maxLeft);
    var placement = top < 52 ? "bottom" : "top";
    dock.setAttribute("data-gosx-studio-preview-dock-placement", placement);
    dock.style.top = Math.round(placement === "bottom" ? frameRect.top - shellRect.top + targetRect.bottom + 10 : top - 10) + "px";
    dock.style.left = Math.round(left) + "px";
    return { handled: true, placement: placement };
  }

  function syncEditorPreviewDockPositions(form) {
    if (!form) return { handled: false, count: 0 };
    var count = 0;
    editorPreviewFrames(form).forEach(function (frame) {
      var dock = frame && frame.__gosxStudioPreviewDock;
      var target = frame && frame.__gosxStudioPreviewDockTarget;
      if (!dock || !target || dock.hidden) return;
      var result = syncEditorPreviewDockPosition(frame, dock, target);
      if (result && result.handled) count += 1;
    });
    return { handled: true, count: count };
  }

  function syncEditorPreviewDockPositionForFrame(frame) {
    var dock = frame && frame.__gosxStudioPreviewDock;
    var target = frame && frame.__gosxStudioPreviewDockTarget;
    if (!dock || !target || dock.hidden) return { handled: false, placement: "" };
    return syncEditorPreviewDockPosition(frame, dock, target);
  }

  window.__gosx_preview_runtime_island_syncDockPosition = syncEditorPreviewDockPositionForFrame;

  function syncEditorPreviewDockSelection(form, frame, target, selection, host) {
    var dockLookup = editorPreviewDockForFrame(frame);
    var dock = dockLookup.dock;
    var shell = dockLookup.shell;
    if (!dock || !target) return { handled: false, dock: dock || null, state: null, bound: false, positioned: false };
    var normalizedSelection = editorPreviewDockSelection(selection);
    var bindResult = bindEditorPreviewDock(form, frame, dock, host);
    frame.__gosxStudioPreviewDock = dock;
    frame.__gosxStudioPreviewDockTarget = target;
    syncEditorPreviewDock(dock, normalizedSelection);
    var state = editorPreviewFieldNavigationState(frame, target, normalizedSelection);
    syncEditorPreviewDockFieldNavigation(dock, state.count, state.index);
    syncEditorPreviewFieldMap(frame, state.fields, state.current, state.count);
    var position = syncEditorPreviewDockPosition(frame, dock, target, shell);
    return { handled: true, dock: dock, state: state, bound: !!(bindResult && bindResult.bound), positioned: !!(position && position.handled) };
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

  function editorPreviewFieldPatch(field) {
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

  function editorPreviewPatchEnvelope(reason, detail, field) {
    return {
      type: "gosxstudio:preview-patch",
      source: "gosx-studio",
      reason: reason || "patch",
      detail: detail || {},
      field: editorPreviewFieldPatch(field)
    };
  }

  function syncEditorPreviewFrame(form, frame, reason, shouldTransport) {
    if (!form || !editorPreviewFrameDocument(frame)) return { handled: false, count: 0 };
    reason = reason || "sync";
    var count = 0;
    queryAll(form, "[data-studio-field-source], [data-editor-source], input[name], textarea[name], select[name]").forEach(function (field) {
      if (typeof shouldTransport === "function" && shouldTransport(field, reason) === false) return;
      var patch = {
        type: "gosxstudio:preview-patch",
        source: "gosx-studio",
        reason: reason,
        detail: {},
        field: editorPreviewFieldPatch(field)
      };
      count += applyEditorPreviewPatch(frame, patch).count || 0;
    });
    if (count) emitEditorPreview(form, "gosxstudio:preview-sync", { count: count, reason: reason });
    return { handled: true, count: count };
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

  function editorPreviewFrameTask(fn) {
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

  function editorPreviewHostCall(host, name, fallback) {
    if (!host || typeof host[name] !== "function") return fallback;
    return host[name].apply(host, Array.prototype.slice.call(arguments, 3));
  }

  function editorPreviewInlineEditContains(frame, target) {
    var edit = frame && frame.__gosxStudioInlineEdit;
    var editTarget = edit && edit.target;
    if (!editTarget || !target) return false;
    return target === editTarget || !!(editTarget.contains && editTarget.contains(target));
  }

  function editorPreviewPointerEventEligible(event) {
    if (!event || event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return false;
    if (event.button > 0) return false;
    return true;
  }

  function editorPreviewKeyboardIntent(event) {
    if (!event || event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return null;
    var focusedControl = event.target && event.target.closest ? event.target.closest("input, textarea, select, button, a[href], [contenteditable='true'], [contenteditable='plaintext-only']") : null;
    if (focusedControl) return null;
    if (event.key === "[") return { action: "prev-field", reason: "keyboard-prev-field" };
    if (event.key === "]") return { action: "next-field", reason: "keyboard-next-field" };
    if (event.key === "F2") return { action: "inline-text", reason: "keyboard-f2" };
    if (event.key === "Enter") return { action: "inline-text", reason: "keyboard-enter" };
    return null;
  }

  var editorPreviewDocumentBindings = typeof WeakMap !== "undefined" ? new WeakMap() : null;

  function editorPreviewBoundDocument(frame) {
    if (!frame) return null;
    return editorPreviewDocumentBindings ? editorPreviewDocumentBindings.get(frame) : frame.__gosxStudioPreviewRuntimeDocument;
  }

  function setEditorPreviewBoundDocument(frame, frameDoc) {
    if (!frame) return;
    if (editorPreviewDocumentBindings) editorPreviewDocumentBindings.set(frame, frameDoc);
    else frame.__gosxStudioPreviewRuntimeDocument = frameDoc;
  }

  function bindEditorPreviewDocument(form, frame, host) {
    var frameDoc = editorPreviewFrameDocument(frame);
    if (!frame || !frameDoc || editorPreviewBoundDocument(frame) === frameDoc) return { handled: false, bound: false };
    setEditorPreviewBoundDocument(frame, frameDoc);
    if (frameDoc.documentElement) frameDoc.documentElement.setAttribute("data-gosx-studio-preview-selectable", "true");
    var repositionDock = editorPreviewFrameTask(function () {
      syncEditorPreviewDockPositionForFrame(frame);
    });
    frameDoc.addEventListener("click", function (event) {
      if (editorPreviewInlineEditContains(frame, event.target)) return;
      if (!editorPreviewPointerEventEligible(event)) return;
      var target = editorPreviewSelectableNode(event.target);
      if (!target) return;
      var detail = editorPreviewHostCall(host, "previewSelectionDetail", {}, target) || {};
      if (editorPreviewHostCall(host, "applyPreviewSelection", false, frame, target, detail, { reveal: true, reason: "click" })) {
        event.preventDefault();
        event.stopPropagation();
      }
    }, true);
    frameDoc.addEventListener("dblclick", function (event) {
      if (editorPreviewInlineEditContains(frame, event.target)) return;
      if (!editorPreviewPointerEventEligible(event)) return;
      var target = editorPreviewSelectableNode(event.target);
      if (!target) return;
      var detail = editorPreviewHostCall(host, "previewSelectionDetail", {}, target) || {};
      if (detail.editable !== "text") return;
      if (editorPreviewHostCall(host, "applyPreviewSelection", false, frame, target, detail, { reveal: true, reason: "double-click" }) && editorPreviewHostCall(host, "startInlineTextFromDetail", false, frame, detail, "double-click")) {
        event.preventDefault();
        event.stopPropagation();
      }
    }, true);
    frameDoc.addEventListener("focusin", function (event) {
      if (editorPreviewInlineEditContains(frame, event.target)) return;
      var target = editorPreviewSelectableNode(event.target);
      if (target) {
        var detail = editorPreviewHostCall(host, "previewSelectionDetail", {}, target) || {};
        editorPreviewHostCall(host, "applyPreviewSelection", false, frame, target, detail, { reveal: false, reason: "focus" });
      }
    }, true);
    frameDoc.addEventListener("input", function (event) {
      editorPreviewHostCall(host, "handleInlineTextInput", false, frame, event);
    });
    frameDoc.addEventListener("keydown", function (event) {
      editorPreviewHostCall(host, "handleInlineTextKeyEvent", false, frame, event);
    });
    frameDoc.addEventListener("keydown", function (event) {
      if (frame.__gosxStudioInlineEdit) return;
      var intent = editorPreviewKeyboardIntent(event);
      if (!intent) return;
      if (intent.action === "prev-field" || intent.action === "next-field") {
        var result = runEditorPreviewFieldNavigation({
          frame: frame,
          direction: intent.action === "next-field" ? 1 : -1,
          reason: intent.reason,
          host: host
        }, event);
        if (result && result.navigated) {
          event.preventDefault();
          event.stopPropagation();
        }
        return;
      }
      if (intent.action !== "inline-text") return;
      if (startEditorPreviewInlineTextFromSelection(form, frame, host, intent.reason)) {
        event.preventDefault();
        event.stopPropagation();
      }
    });
    frameDoc.addEventListener("paste", function (event) {
      editorPreviewHostCall(host, "handleInlineTextPaste", false, frame, event);
    });
    frameDoc.addEventListener("blur", function (event) {
      editorPreviewHostCall(host, "handleInlineTextBlur", false, frame, event);
    }, true);
    frameDoc.addEventListener("scroll", repositionDock, true);
    if (frame.contentWindow) frame.contentWindow.addEventListener("resize", repositionDock);
    return { handled: true, bound: true };
  }

  function startEditorPreviewInlineTextFromSelection(form, frame, host, reason) {
    var dock = frame && frame.__gosxStudioPreviewDock;
    if (!dock || dock.hidden) return false;
    var detail = editorPreviewDockDetail(form, dock);
    return editorPreviewHostCall(host, "startInlineTextFromDetail", false, frame, detail, reason || "keyboard");
  }

  function bindEditorPreviewFrameDocument(form, frame, host) {
    return bindEditorPreviewDocument(form, frame, host);
  }

  function bindEditorPreviewFrames(form, host) {
    if (!form) return { handled: false, frames: 0, bound: 0 };
    var frames = editorPreviewFrames(form);
    var bound = 0;
    frames.forEach(function (frame) {
      if (!frame || !frame.addEventListener) return;
      if (frame.dataset && frame.dataset.gosxStudioPreviewBound === "true") return;
      if (frame.dataset) frame.dataset.gosxStudioPreviewBound = "true";
      if (!frame.getAttribute("data-studio-preview-src")) {
        frame.setAttribute("data-studio-preview-src", frame.getAttribute("src") || "");
      }
      frame.addEventListener("load", function () {
        setEditorPreviewStatus(form, "ready", "Ready", "load");
        syncEditorPreviewFrame(form, frame, "load", typeof host.shouldTransportPreviewPatch === "function" ? host.shouldTransportPreviewPatch : null);
        bindEditorPreviewFrameDocument(form, frame, host);
        var route = editorPreviewURL(frame) || frame.getAttribute("src") || "";
        postEditorPreviewPatch(form, "load-sync", editorPreviewPatchEnvelope("load-sync", { route: route }, null));
      });
      frame.addEventListener("error", function () {
        setEditorPreviewStatus(form, "error", "Preview failed", "error");
      });
      bindEditorPreviewFrameDocument(form, frame, host);
      bound += 1;
    });
    return { handled: true, frames: frames.length, bound: bound };
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

  window.__gosx_preview_runtime_island_setStatus = function (form, state, label, reason) {
    return setEditorPreviewStatus(form, state, label, reason);
  };
  window.__gosx_preview_runtime_island_syncRoute = function (form, route, reason) {
    return syncEditorPreviewRoute(form, route, reason);
  };
  window.__gosx_preview_runtime_island_refreshNow = function (form, reason, route) {
    return refreshEditorPreviewNow(form, reason, route);
  };
  window.__gosx_preview_runtime_island_scheduleRefresh = function (form, reason, route) {
    return scheduleEditorPreviewRefresh(form, reason, route);
  };
  window.__gosx_preview_runtime_island_postPatch = function (form, reason, detail, field) {
    return postEditorPreviewPatch(form, reason, editorPreviewPatchEnvelope(reason, detail, field));
  };
  window.__gosx_preview_runtime_island_bindFrames = function (form, host) {
    return bindEditorPreviewFrames(form, host);
  };
  window.__gosx_preview_runtime_island_clearSelections = function (form, host) {
    return clearEditorPreviewSelections(form, host);
  };
  window.__gosx_preview_runtime_island_applySelection = function (detail) {
    return applyEditorPreviewSelectionSync(detail, null);
  };
  window.__gosx_preview_runtime_island_runDockAction = function (detail) {
    return runEditorPreviewDockAction(detail, null);
  };

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
  // The legacy mount() walks every workbench form in root and calls
  // bindPreview(form) to wire the drag/drop/inline-edit machinery for the
  // editor preview overlay. The slice-6 writer just bumps the mount epoch
  // signal; the preview-side subscriber re-runs its bindPreview routine
  // when the epoch changes.
  function editorPreviewWorkbenchForms(root) {
    var scope = root || doc;
    var forms = [];
    if (scope && scope.matches && scope.matches("form[data-studio-workbench], form[data-editor-workbench]")) {
      forms.push(scope);
    }
    if (scope && scope.querySelectorAll) {
      Array.prototype.forEach.call(scope.querySelectorAll("form[data-studio-workbench], form[data-editor-workbench]"), function (form) {
        forms.push(form);
      });
    }
    return forms;
  }

  function mountPreviewIsland(root) {
    mountEpoch += 1;
    writeSharedSignal("$preview.mount.epoch", mountEpoch);
    editorPreviewWorkbenchForms(root).forEach(function (form) {
      syncEditorPreviewDockPositions(form);
    });
  }

  window.__gosx_preview_runtime_island_mount = mountPreviewIsland;

  window.addEventListener("resize", function () {
    editorPreviewWorkbenchForms(doc).forEach(function (form) {
      syncEditorPreviewDockPositions(form);
    });
  });

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
