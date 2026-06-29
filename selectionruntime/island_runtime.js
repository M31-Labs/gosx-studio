// island_runtime.js — companion JS for the selectionruntime .gsx island.
//
// The .gsx island (selection_bind.gsx) is a mount-point marker in the
// editor DOM. This script publishes the
// window.__gosx_selection_runtime_island_bind global that
// selectionruntime.BridgeShim delegates to when the
// "selection-runtime-islands" feature flag is on. It is emitted into the
// studio runtime bundle by selectionruntime.IslandRuntimeJS() (see
// runtime.go) and runs before BridgeShim() at bundle init time.
//
// The single function below replaces window.GoSXStudioSelectionRuntime.bind
// while preserving exact observable behavior of the legacy
// bindSelectionSurface (assets/studio-engines.js:713). The fan-out covers
// five sub-behaviors enumerated in selection_bind.gsx:
//
//   1. Block selection         — clicks / focusin on data-block-studio-block
//                                rows update data-studio-selection,
//                                data-studio-selection-kind="block", toggle
//                                .is-selected, dispatch
//                                CustomEvent("blockstudio:select"), and
//                                write $selection.block.
//
//   2. Workspace target select — clicks / focusin on data-studio-panel-link
//                                or data-studio-site-page elements update
//                                data-studio-workspace-selection and
//                                data-studio-selection-kind="workspace",
//                                toggle .is-selected on the active link,
//                                set aria-current="true", call
//                                workbench.setMode("advanced") on reveal,
//                                scroll the inspector node into view, and
//                                write $selection.workspaceTarget.
//
//   3. Field focus             — focusin on data-studio-field-source inputs
//                                or studio:field-select / studio:inline-
//                                edit-start CustomEvents set
//                                data-studio-field-selection,
//                                data-studio-field-editable, toggle
//                                .is-studio-field-active on the row, update
//                                inline-text button labels, and write
//                                $selection.fieldFocus.
//
//   4. Selection commandbar    — clicks on data-studio-selection-action
//                                buttons (reveal / content / style /
//                                inline-text / previous-field / next-field
//                                / toggle-visibility) call into the
//                                selected row, dispatch previewRuntime
//                                methods (requestInlineEdit / cycleField)
//                                where applicable, and toggle visibility
//                                checkboxes via dispatched input/change
//                                events. Also bind the command-palette
//                                gosxstudio:command listener.
//
//   5. Style-scope readouts    — listen for gosxstudio:workbench-mode-change,
//                                gosxstudio:workbench-viewport-change,
//                                gosxstudio:workbench-style-state-change
//                                events and refresh data-studio-style-scope-*
//                                attributes + readouts; write $selection.styleScope.
//
// Cross-iframe state is NOT required for selection — all five sub-behaviors
// operate on editor-side DOM only. Selection writes use plain Bridge signals
// (not $preview.*). The preview-side fan-out (inline-edit, field-cycle)
// still routes through GoSXStudioPreviewRuntime, which slice 6 will
// replace; until then the legacy preview runtime keeps those paths live.

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

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
  var selectionOperationCounter = 0;

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

  function selectionReducedMotion() {
    return window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }

  function selectionWorkbenchRuntime() {
    return window.GoSXStudioWorkbenchRuntime || {};
  }

  function selectionPreviewRuntime() {
    return window.GoSXStudioPreviewRuntime || {};
  }

  function selectionBlockRow(root, key) {
    var runtime = window.GoSXStudioBlockLayoutRuntime;
    if (runtime && typeof runtime.rowForKey === "function") {
      return runtime.rowForKey(root || doc, key);
    }
    // Fallback when slice 4 (BlockLayoutRuntime burn-down) hasn't shipped
    // yet — query the row directly. Mirrors the legacy blockRowForKey path.
    var scope = root && root.querySelector ? root : doc;
    if (!key) return null;
    return scope.querySelector('[data-block-studio-block="' + cssAttrValue(key) + '"]');
  }

  function cssAttrValue(value) {
    return String(value).replace(/"/g, '\\"');
  }

  function attrValue(value) {
    return String(value).replace(/"/g, '\\"');
  }

  function compactText(value) {
    return String(value || "").replace(/\s+/g, " ").trim();
  }

  function editorWorkbench(root) {
    var scope = root && root.querySelector ? root : doc;
    return scope.querySelector ? scope.querySelector("[data-studio-workbench]") : null;
  }

  // Set a shared signal value through the WASM bridge, if available. The
  // bridge exposes window.__gosx_set_shared_signal_json(name, valueJSON)
  // after the runtime boots; before then the call is a no-op (the legacy
  // attribute writes below still keep the UI live).
  function writeSharedSignal(name, value) {
    try {
      var setter = window.__gosx_set_shared_signal_json;
      if (typeof setter === "function") {
        setter(name, JSON.stringify(value));
      }
    } catch (error) {
      // Shared-signal channel not ready yet. Attribute writes keep state live.
    }
  }

  function bindSelectionSurfaceIsland(root) {
    var form = editorWorkbench(root);
    if (!form || form.dataset.gosxStudioSelectionIslandBound === "true") return;
    form.dataset.gosxStudioSelectionIslandBound = "true";

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

    // Sub-behavior 5: style-scope readouts.
    function updateStyleScope() {
      var row = selectedRow();
      var blockLabel = row ? row.getAttribute("data-studio-block-label") || row.getAttribute("data-block-studio-block") || "Block" : "No selection";
      var field = form.getAttribute("data-studio-field-selection") || "";
      var state = normalizeStyleState(form.getAttribute("data-studio-style-state") || "default");
      var breakpoint = currentBreakpoint();
      var valid = form.getAttribute("data-studio-style-valid") !== "false";
      var system = form.getAttribute("data-studio-style-system") || "Local";
      form.setAttribute("data-studio-style-state", state);
      form.setAttribute("data-studio-style-breakpoint", breakpoint);
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
      writeSharedSignal("$selection.styleScope", {
        block: blockLabel,
        field: field,
        state: state,
        breakpoint: breakpoint,
        valid: valid,
        system: system
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
      return String(key || "").replace(/^tool-/, "");
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

    // Sub-behavior 3: field focus management.
    function clearFieldFocus() {
      form.removeAttribute("data-studio-field-selection");
      form.removeAttribute("data-studio-field-editable");
      form.removeAttribute("data-studio-field-action-label");
      form.removeAttribute("data-studio-field-action-href");
      form.removeAttribute("data-studio-field-action-formaction");
      Array.prototype.forEach.call(form.querySelectorAll(".is-studio-field-active"), function (row) {
        row.classList.remove("is-studio-field-active");
      });
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-field-selection-label]"), function (readout) {
        readout.textContent = "Block";
      });
      updateFieldActionLabels();
      updateStyleScope();
      writeSharedSignal("$selection.fieldFocus", { field: "", editable: "", label: "" });
    }

    // Sub-behavior 2: workspace target selection.
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
        if (node && node.scrollIntoView) {
          node.scrollIntoView({ behavior: selectionReducedMotion() ? "auto" : "smooth", block: "center" });
        }
      }
      updateStyleScope();
      writeSharedSignal("$selection.workspaceTarget", { scope: scope, key: key, label: label });
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
      return form.querySelector('[data-studio-field-source="' + attrValue(field) + '"], [data-editor-source="' + attrValue(field) + '"]');
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

    function inferPreviewEditableKind(source, control) {
      if (!source) return "";
      var explicit = source.getAttribute("data-studio-field-editable") || source.getAttribute("data-studio-editable") || "";
      if (explicit) return explicit;
      var tag = control && control.tagName ? control.tagName.toLowerCase() : "";
      var type = control && control.type ? String(control.type).toLowerCase() : "";
      if (tag === "textarea") return "text";
      if (tag === "input" && (type === "url" || type === "email")) return "url";
      if (tag === "input" || tag === "select") return "text";
      return "";
    }

    function previewReadableFieldName(field, editable) {
      var value = compactText(field || "field").split(".").pop() || field || "Field";
      value = value.replace(/([a-z])([A-Z])/g, "$1 $2").replace(/[-_]/g, " ");
      value = value.charAt(0).toUpperCase() + value.slice(1);
      if (editable === "media" || editable === "image") return value === "Media" ? value : value + " media";
      if (editable === "source") return value + " source";
      if (editable === "flow") return value + " flow";
      if (editable === "url" || editable === "link") return value + " URL";
      return value || "Field";
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

    function resolvePreviewSelectionDetail(event) {
      var envelope = event.detail || {};
      var node = envelope.target;
      if (!node || !node.closest) {
        envelope.result = {};
        return;
      }
      var fieldNode = node.closest("[data-studio-field], [data-editor-preview], [data-studio-field-source]");
      var blockNode = node.closest("[data-studio-block-key], [data-studio-node-id]");
      var field = fieldNode ? (fieldNode.getAttribute("data-studio-field") || fieldNode.getAttribute("data-editor-preview") || fieldNode.getAttribute("data-studio-field-source") || "") : "";
      var editable = fieldNode ? (fieldNode.getAttribute("data-studio-editable") || fieldNode.getAttribute("data-studio-field-editable") || "") : "";
      var source = fieldSource(field);
      var control = fieldControl(source);
      if (!editable) editable = inferPreviewEditableKind(source, control);
      var label = fieldNode ? (fieldNode.getAttribute("data-studio-field-label") || previewReadableFieldName(field, editable)) : "";
      var action = fieldNode ? (fieldNode.getAttribute("data-studio-field-action") || "") : "";
      var actionHref = fieldNode ? (fieldNode.getAttribute("data-studio-field-action-href") || "") : "";
      var actionFormAction = fieldNode ? (fieldNode.getAttribute("data-studio-field-action-formaction") || "") : "";
      var blockKey = blockNode ? (blockNode.getAttribute("data-studio-block-key") || "") : "";
      var nodeID = blockNode ? (blockNode.getAttribute("data-studio-node-id") || "") : "";
      var blockLabel = blockNode ? previewBlockLabel(blockNode, blockKey || nodeID || "") : "";
      if (!label) label = blockLabel || "Preview selection";
      envelope.result = {
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

    function applyFieldActionMetadata(source, detail) {
      detail = detail || {};
      var actionLabel = detail.action || (source && source.getAttribute("data-studio-field-action")) || "";
      var actionHref = detail.actionHref || (source && source.getAttribute("data-studio-field-action-href")) || "";
      var actionFormAction = detail.actionFormAction || (source && source.getAttribute("data-studio-field-action-formaction")) || "";
      if (actionLabel) form.setAttribute("data-studio-field-action-label", actionLabel);
      else form.removeAttribute("data-studio-field-action-label");
      if (actionHref) form.setAttribute("data-studio-field-action-href", actionHref);
      else form.removeAttribute("data-studio-field-action-href");
      if (actionFormAction) form.setAttribute("data-studio-field-action-formaction", actionFormAction);
      else form.removeAttribute("data-studio-field-action-formaction");
      return {
        action: actionLabel,
        actionHref: actionHref,
        actionFormAction: actionFormAction
      };
    }

    function clearPreviewInspectorSelection() {
      Array.prototype.forEach.call(form.querySelectorAll("[data-gosx-studio-inspector-selected]"), function (target) {
        target.removeAttribute("data-gosx-studio-inspector-selected");
        if (target.classList) target.classList.remove("is-studio-field-active", "is-preview-selected");
      });
    }

    function markPreviewInspectorSelection(source, control) {
      var row = source && source.closest ? source.closest(".field-row, [data-studio-field-row]") : null;
      [source, row, control].forEach(function (target) {
        if (!target) return;
        target.setAttribute("data-gosx-studio-inspector-selected", "true");
        if (target.classList) target.classList.add(target === row ? "is-studio-field-active" : "is-preview-selected");
      });
    }

    function previewSelectionResult(detail) {
      detail = detail || {};
      var source = fieldSource(detail.field || "");
      var control = fieldControl(source);
      var editable = detail.editable || inferPreviewEditableKind(source, control) || "";
      var actions = applyFieldActionMetadata(source, detail);
      var selectionKey = detail.blockKey || detail.nodeID || detail.field || "";
      var kind = detail.field ? "preview-field" : "preview";
      return {
        field: detail.field || "",
        editable: editable,
        label: detail.label || "",
        blockLabel: detail.blockLabel || "",
        action: actions.action || "",
        actionHref: actions.actionHref || "",
        actionFormAction: actions.actionFormAction || "",
        selectionKey: selectionKey,
        kind: kind,
        blockKey: detail.blockKey || "",
        nodeID: detail.nodeID || ""
      };
    }

    function previewFieldActionEnvelope(detail) {
      detail = detail || {};
      return {
        field: detail.field || "",
        editable: detail.editable || "",
        label: detail.label || "",
        blockKey: detail.blockKey || "",
        action: detail.action || "",
        actionHref: detail.actionHref || "",
        actionFormAction: detail.actionFormAction || ""
      };
    }

    function previewDockActionEnvelope(action, detail) {
      detail = detail || {};
      return {
        action: action || "",
        field: detail.field || "",
        editable: detail.editable || "",
        label: detail.label || "",
        blockKey: detail.blockKey || "",
        blockLabel: detail.blockLabel || "",
        actionLabel: detail.actionLabel || detail.action || "",
        actionHref: detail.actionHref || "",
        actionFormAction: detail.actionFormAction || ""
      };
    }

    function previewFieldNavigationEnvelope(detail, navigation) {
      detail = detail || {};
      navigation = navigation || {};
      return {
        detail: {
          field: detail.field || "",
          editable: detail.editable || "",
          label: detail.label || "",
          blockKey: detail.blockKey || ""
        },
        navigation: {
          direction: navigation.direction || "",
          fieldIndex: navigation.fieldIndex || 0,
          fieldCount: navigation.fieldCount || 0,
          reason: navigation.reason || "field-navigation"
        }
      };
    }

    function resolvePreviewFieldAction(event) {
      var envelope = event.detail || {};
      var detail = previewFieldActionEnvelope(envelope.detail || envelope);
      var field = envelope.field || "";
      var editable = envelope.editable || "";
      var label = envelope.label || "";
      var blockKey = envelope.blockKey || "";
      var action = envelope.action || "";
      var actionHref = envelope.actionHref || "";
      var actionFormAction = envelope.actionFormAction || "";
      field = detail.field;
      editable = detail.editable;
      label = detail.label;
      blockKey = detail.blockKey;
      action = detail.action;
      actionHref = detail.actionHref;
      actionFormAction = detail.actionFormAction;
      envelope.result = {
        inlineText: editable === "text",
        submit: !!actionFormAction,
        navigate: !!actionHref,
        href: actionHref || "",
        reveal: !!field,
        field: field,
        editable: editable,
        label: label,
        blockKey: blockKey,
        action: action,
        actionHref: actionHref,
        actionFormAction: actionFormAction
      };
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
      var submitters = Array.prototype.slice.call(source.querySelectorAll(selector));
      if (!formAction) return submitters[0] || null;
      for (var i = 0; i < submitters.length; i += 1) {
        var candidate = submitters[i];
        if (!isFormSubmitControl(candidate)) continue;
        if ((candidate.getAttribute("data-studio-field-action-formaction") || candidate.getAttribute("formaction") || "") === formAction) return candidate;
      }
      return submitters.filter(isFormSubmitControl)[0] || null;
    }

    function submitPreviewFieldAction(event) {
      var envelope = event.detail || {};
      var detail = previewFieldActionEnvelope(envelope.detail || envelope);
      envelope.result = false;
      var source = fieldSource(detail.field || "");
      var submitter = fieldActionSubmitter(source, detail.actionFormAction || "");
      var action = detail.actionFormAction || (submitter && (submitter.getAttribute("data-studio-field-action-formaction") || submitter.getAttribute("formaction"))) || "";
      if (!action) return;
      var confirmMessage = (submitter && submitter.getAttribute("data-admin-confirm")) || (source && source.getAttribute("data-admin-confirm")) || "";
      if (confirmMessage && !window.confirm(confirmMessage)) return;
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
        envelope.result = true;
      } catch (error) {
        if (!submitter || !submitter.click) return;
        try {
          submitter.click();
          envelope.result = true;
        } catch (clickError) {
          envelope.result = false;
        }
      }
    }

    function resolvePreviewDockAction(event) {
      var envelope = event.detail || {};
      var detail = previewDockActionEnvelope(envelope.action || "", envelope.detail || envelope);
      var action = detail.action || "";
      var field = detail.field || "";
      var editable = detail.editable || "";
      var label = detail.label || "";
      var blockKey = detail.blockKey || "";
      var blockLabel = detail.blockLabel || "";
      var actionLabel = detail.actionLabel || "";
      var actionHref = detail.actionHref || "";
      var actionFormAction = detail.actionFormAction || "";
      envelope.result = {
        mode: action === "content" ? "content" : action === "style" ? "style" : "",
        reveal: action === "content" && !!field,
        action: action,
        field: field,
        editable: editable,
        label: label,
        blockKey: blockKey,
        blockLabel: blockLabel,
        actionLabel: actionLabel,
        actionHref: actionHref,
        actionFormAction: actionFormAction
      };
    }

    function applyPreviewSelectionState(event) {
      var envelope = event.detail || {};
      var detail = envelope.detail || {};
      var options = envelope.options || {};
      if (!detail.field && !detail.blockKey && !detail.nodeID) return;
      clearPreviewInspectorSelection();
      clearFieldFocus();
      var result = previewSelectionResult(detail);
      var source = fieldSource(result.field);
      var control = fieldControl(source);
      if (result.field) {
        form.setAttribute("data-studio-field-selection", result.field);
        if (result.editable) form.setAttribute("data-studio-field-editable", result.editable);
        else form.removeAttribute("data-studio-field-editable");
        markPreviewInspectorSelection(source, control);
      } else {
        form.removeAttribute("data-studio-field-selection");
        form.removeAttribute("data-studio-field-editable");
      }
      if (result.selectionKey) form.setAttribute("data-studio-selection", result.selectionKey);
      else form.removeAttribute("data-studio-selection");
      form.setAttribute("data-studio-selection-kind", result.kind);
      setSelectionReadout(result.label || result.field || result.blockKey || "Preview selection");
      setReadout("[data-studio-selection-status]", result.field ? "Preview field" : "Preview selection");
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-field-selection-label]"), function (readout) {
        readout.textContent = result.field ? (result.label || fieldLabel(result.field)) : "Block";
      });
      updateFieldActionLabels();
      updateStyleScope();
      writeSharedSignal("$selection.fieldFocus", { field: result.field, editable: result.editable, label: result.label || fieldLabel(result.field) });
      envelope.result = result;
      selectionOperationCounter += 1;
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-operation", {
        bubbles: true,
        detail: {
          id: "studio-op-" + Date.now() + "-" + selectionOperationCounter,
          kind: "select_preview",
          source: "gosx-studio",
          mutation: false,
          reason: options.reason || "preview",
          target: {
            field: result.field || detail.field || "",
            editable: result.editable || detail.editable || "",
            blockKey: result.blockKey || detail.blockKey || "",
            nodeID: result.nodeID || detail.nodeID || "",
            selection: result.selectionKey || detail.blockKey || detail.nodeID || detail.field || "",
            kind: result.kind || (detail.field ? "preview-field" : "preview")
          },
          payload: {
            label: result.label || detail.label || "",
            blockLabel: result.blockLabel || detail.blockLabel || "",
            action: result.action || "",
            actionHref: result.actionHref || "",
            actionFormAction: result.actionFormAction || ""
          }
        }
      }));
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-select", {
        bubbles: true,
        detail: {
          field: result.field,
          source: detail.source || result.field,
          editable: result.editable,
          label: result.label,
          blockLabel: result.blockLabel,
          action: result.action,
          actionHref: result.actionHref,
          actionFormAction: result.actionFormAction,
          blockKey: result.blockKey,
          nodeID: result.nodeID,
          reason: options.reason || "preview"
        }
      }));
    }

    function clearPreviewSelectionState(event) {
      var envelope = event.detail || {};
      clearPreviewInspectorSelection();
      form.removeAttribute("data-studio-selection");
      form.removeAttribute("data-studio-selection-kind");
      form.removeAttribute("data-studio-workspace-selection");
      updateWorkspaceSelectionLinks("");
      syncHomeLayerPicker("");
      clearFieldFocus();
      setSelectionReadout("No selection");
      updateSelectionStatus(null);
      Array.prototype.forEach.call(form.querySelectorAll("[data-studio-field-selection-label]"), function (readout) {
        readout.textContent = "Block";
      });
      updateFieldActionLabels();
      updateStyleScope();
      writeSharedSignal("$selection.block", { key: "", label: "" });
      writeSharedSignal("$selection.workspaceTarget", { scope: "", key: "", label: "" });
      envelope.result = { cleared: true };
    }

    function emitPreviewFieldNavigation(event) {
      var envelope = event.detail || {};
      var normalized = previewFieldNavigationEnvelope(envelope.detail || envelope, envelope.navigation || {});
      var detail = normalized.detail;
      var navigation = normalized.navigation;
      var field = detail.field || "";
      var editable = detail.editable || "";
      var label = detail.label || "";
      var blockKey = detail.blockKey || "";
      var direction = navigation.direction || "";
      var fieldIndex = navigation.fieldIndex || 0;
      var fieldCount = navigation.fieldCount || 0;
      var reason = navigation.reason || "field-navigation";
      selectionOperationCounter += 1;
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-operation", {
        bubbles: true,
        detail: {
          id: "studio-op-" + Date.now() + "-" + selectionOperationCounter,
          kind: "preview_field_navigate",
          source: "gosx-studio",
          reason: reason,
          mutation: false,
          target: {
            field: field,
            editable: editable,
            blockKey: blockKey,
            selection: form.getAttribute("data-studio-selection") || field,
            kind: form.getAttribute("data-studio-selection-kind") || "preview-field"
          },
          payload: {
            direction: direction,
            fieldIndex: fieldIndex,
            fieldCount: fieldCount,
            label: label
          }
        }
      }));
      form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-navigate", {
        bubbles: true,
        detail: {
          field: field,
          editable: editable,
          label: label,
          blockKey: blockKey,
          direction: direction,
          fieldIndex: fieldIndex,
          fieldCount: fieldCount,
          reason: reason
        }
      }));
      envelope.result = { emitted: true };
    }

    function revealFieldControl(source, control) {
      var reduced = selectionReducedMotion();
      setMode("home", true);
      if (source && source.scrollIntoView) {
        source.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });
      }
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
      var row = source && source.closest ? source.closest(".field-row") : null;
      if (row) row.classList.add("is-studio-field-active");
      if (reveal && source) revealFieldControl(source, control);
      updateFieldActionLabels();
      updateStyleScope();
      writeSharedSignal("$selection.fieldFocus", { field: field, editable: editable, label: label });
    }

    // Sub-behavior 1: block selection.
    function selectRow(row) {
      if (!row) return;
      Array.prototype.forEach.call(form.querySelectorAll("[data-block-studio-block]"), function (item) {
        item.classList.toggle("is-selected", item === row);
      });
      updateSelection(row.getAttribute("data-block-studio-block") || "");
      doc.dispatchEvent(new CustomEvent("blockstudio:select", {
        bubbles: true,
        detail: { key: row.getAttribute("data-block-studio-block") || "" }
      }));
    }

    function revealSelection(row) {
      if (!row) return;
      var reduced = selectionReducedMotion();
      setMode("home", true);
      selectRow(row);
      if (row.scrollIntoView) {
        row.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });
      }
      window.setTimeout(function () {
        if (row.focus) row.focus({ preventScroll: true });
      }, reduced ? 0 : 120);
    }

    // Sub-behavior 4: selection commandbar actions.
    function selectionActionLabel(action, labelOverride) {
      var button = action ? form.querySelector('[data-studio-selection-action="' + attrValue(action) + '"]') : null;
      return compactText(button && (button.getAttribute("aria-label") || button.getAttribute("title") || button.textContent)) || labelOverride || action || "";
    }

    function emitSelectionAction(action, labelOverride) {
      var detail = {
        action: action || "",
        label: selectionActionLabel(action, labelOverride),
        selection: form.getAttribute("data-studio-selection") || form.getAttribute("data-gosx-studio-canvas-selected") || "",
        kind: form.getAttribute("data-studio-selection-kind") || ""
      };
      form.dispatchEvent(new CustomEvent("gosxstudio:selection-action", { bubbles: true, detail: detail }));
      form.dispatchEvent(new CustomEvent("gosxstudio:workbench-action", { bubbles: true, detail: detail }));
    }

    function mirrorPreviewActionSelection(event) {
      var envelope = event.detail || {};
      var detail = previewDockActionEnvelope(envelope.action || "", envelope.detail || envelope);
      var reason = envelope.reason || "preview-dock";
      selectionOperationCounter += 1;
      form.dispatchEvent(new CustomEvent("gosxstudio:editor-operation", {
        bubbles: true,
        detail: {
          id: "studio-op-" + Date.now() + "-" + selectionOperationCounter,
          kind: "preview_action",
          source: "gosx-studio",
          mutation: false,
          reason: reason,
          target: {
            field: detail.field || "",
            editable: detail.editable || "",
            blockKey: detail.blockKey || "",
            selection: form.getAttribute("data-studio-selection") || detail.blockKey || detail.field || "",
            kind: form.getAttribute("data-studio-selection-kind") || "preview"
          },
          payload: {
            action: detail.action || "",
            label: detail.label || "",
            blockLabel: detail.blockLabel || "",
            actionLabel: detail.actionLabel || "",
            actionHref: detail.actionHref || "",
            actionFormAction: detail.actionFormAction || ""
          }
        }
      }));
      form.dispatchEvent(new CustomEvent("gosxstudio:selection-action", { bubbles: true, detail: {
        action: detail.action || "",
        label: detail.actionLabel || detail.label || detail.action || "",
        selection: form.getAttribute("data-studio-selection") || detail.blockKey || detail.field || "",
        kind: form.getAttribute("data-studio-selection-kind") || "preview"
      } }));
    }

    function runSelectionAction(action, labelOverride) {
      emitSelectionAction(action, labelOverride);
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
        var preview = selectionPreviewRuntime();
        if (preview.requestInlineEdit) preview.requestInlineEdit({ key: key, field: field });
        return;
      }
      if (action === "previous-field" || action === "next-field") {
        selectRow(row);
        var fieldPreview = selectionPreviewRuntime();
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

    function insertActionLabel(button, fallback) {
      return compactText(button && (button.getAttribute("aria-label") || button.textContent)) || fallback || "";
    }

    function runInsert(button) {
      var target = button.getAttribute("data-studio-insert-block") || button.getAttribute("data-editor-add-block") || "";
      var detail = { target: target, label: insertActionLabel(button, target), button: button };
      form.dispatchEvent(new CustomEvent("gosxstudio:insert-block", { bubbles: true, detail: detail }));
      form.dispatchEvent(new CustomEvent("gosxstudio:workbench-action", {
        bubbles: true,
        detail: { action: "insert-block", target: target, label: detail.label }
      }));
      var add = target ? form.querySelector('[data-editor-add-block="' + attrValue(target) + '"]') : null;
      if (add && add !== button && add.click) add.click();
      return true;
    }

    function runInsertTarget(target, label) {
      target = target || "";
      if (!target) return false;
      var button = form.querySelector('[data-studio-insert-block="' + attrValue(target) + '"], [data-editor-add-block="' + attrValue(target) + '"]');
      if (button) return runInsert(button);
      form.dispatchEvent(new CustomEvent("gosxstudio:insert-block", {
        bubbles: true,
        detail: { target: target, label: label || target }
      }));
      form.dispatchEvent(new CustomEvent("gosxstudio:workbench-action", {
        bubbles: true,
        detail: { action: "insert-block", target: target, label: label || target }
      }));
      return true;
    }

    function bindCommandPaletteCommands() {
      var node = form.querySelector("[data-studio-command-palette]");
      if (!node || node.dataset.gosxStudioSelectionIslandCommandsBound === "true") return;
      node.dataset.gosxStudioSelectionIslandCommandsBound = "true";
      node.addEventListener("gosxstudio:command", function (event) {
        var detail = event.detail || {};
        var kind = detail.kind || "";
        var target = detail.target || "";
        if (kind === "selection-action") {
          runSelectionAction(target, detail.label);
          if (event.preventDefault) event.preventDefault();
        }
        if (kind === "insert" && runInsertTarget(detail.target, detail.label)) {
          if (event.preventDefault) event.preventDefault();
        }
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
        writeSharedSignal("$selection.block", { key: "", label: "" });
        return;
      }
      var label = row.getAttribute("data-studio-block-label") || row.getAttribute("data-block-studio-block") || "Selection";
      var rowKey = row.getAttribute("data-block-studio-block") || "";
      form.setAttribute("data-studio-selection", rowKey);
      form.removeAttribute("data-studio-workspace-selection");
      form.setAttribute("data-studio-selection-kind", "block");
      Array.prototype.forEach.call(form.querySelectorAll("[data-block-studio-block]"), function (item) {
        item.classList.toggle("is-selected", item === row);
      });
      setSelectionReadout(label);
      updateWorkspaceSelectionLinks("");
      syncHomeLayerPicker(rowKey);
      updateSelectionStatus(row);
      syncInspectorForSelection(rowKey);
      clearFieldFocus();
      updateStyleScope();
      writeSharedSignal("$selection.block", { key: rowKey, label: label });
    }

    // Wire up all five sub-behaviors against the workbench form.
    form.addEventListener("click", function (event) {
      var selectionAction = event.target.closest && event.target.closest("[data-studio-selection-action]");
      if (selectionAction && form.contains(selectionAction)) {
        event.preventDefault();
        runSelectionAction(selectionAction.getAttribute("data-studio-selection-action"));
        return;
      }
      var insert = event.target.closest && event.target.closest("[data-studio-insert-block]");
      if (insert && form.contains(insert)) {
        event.preventDefault();
        runInsert(insert);
        return;
      }
      var workspaceTarget = workspaceTargetFromEvent(event);
      if (workspaceTarget && setWorkspaceSelection(workspaceTarget, true)) {
        if (!event.metaKey && !event.ctrlKey && !event.shiftKey) event.preventDefault();
        return;
      }
      var layerPick = event.target.closest && event.target.closest("[data-studio-home-layer-pick]");
      if (layerPick && form.contains(layerPick)) {
        event.preventDefault();
        var layerKey = layerPick.getAttribute("data-studio-home-layer-pick") || "";
        var layerRow = selectionBlockRow(form, layerKey);
        if (layerRow) {
          updateSelection(layerKey);
          if (layerRow.scrollIntoView) {
            layerRow.scrollIntoView({ behavior: selectionReducedMotion() ? "auto" : "smooth", block: "center" });
          }
        }
        return;
      }
    });

    form.addEventListener("click", function (event) {
      var row = event.target.closest && event.target.closest("[data-block-studio-block]");
      if (row && form.contains(row)) updateSelection(row.getAttribute("data-block-studio-block"));
    });
    form.addEventListener("focusin", function (event) {
      var row = event.target.closest && event.target.closest("[data-block-studio-block]");
      if (row && form.contains(row)) updateSelection(row.getAttribute("data-block-studio-block"));
      var workspaceTarget = workspaceTargetFromEvent(event);
      if (workspaceTarget) setWorkspaceSelection(workspaceTarget, false);
    });
    doc.addEventListener("blockstudio:select", function (event) {
      if (doc.contains(form)) updateSelection(event.detail && event.detail.key);
    });
    doc.addEventListener("studio:field-select", function (event) {
      if (!doc.contains(form)) return;
      setFieldFocus(event.detail, event.detail && event.detail.source !== "preview");
    });
    doc.addEventListener("studio:inline-edit-start", function (event) {
      if (!doc.contains(form)) return;
      setFieldFocus(event.detail, false);
    });
    doc.addEventListener("gosxstudio:workbench-mode-change", function (event) {
      if (!doc.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      var mode = event.detail && event.detail.mode || form.getAttribute("data-studio-mode") || "";
      if (mode === "advanced") ensureWorkspaceSelection();
      else if (form.getAttribute("data-studio-workspace-selection")) updateSelection("");
      updateStyleScope();
    });
    doc.addEventListener("gosxstudio:workbench-viewport-change", function (event) {
      if (!doc.contains(form)) return;
      if (event.detail && event.detail.form && event.detail.form !== form) return;
      updateStyleScope();
    });
    doc.addEventListener("gosxstudio:workbench-style-state-change", function (event) {
      if (!doc.contains(form)) return;
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
    form.addEventListener("gosxstudio:preview-action", mirrorPreviewActionSelection);
    form.addEventListener("gosxstudio:preview-selection-detail-resolve", resolvePreviewSelectionDetail);
    form.addEventListener("gosxstudio:preview-selection-apply", applyPreviewSelectionState);
    form.addEventListener("gosxstudio:preview-selection-clear", clearPreviewSelectionState);
    form.addEventListener("gosxstudio:preview-field-navigation-commit", emitPreviewFieldNavigation);
    form.addEventListener("gosxstudio:preview-field-action-resolve", resolvePreviewFieldAction);
    form.addEventListener("gosxstudio:preview-field-action-submit", submitPreviewFieldAction);
    form.addEventListener("gosxstudio:preview-dock-action-resolve", resolvePreviewDockAction);

    bindCommandPaletteCommands();
    if (form.getAttribute("data-studio-mode") === "advanced") ensureWorkspaceSelection();
    else updateSelection("");
  }

  // Publish the island global. The name is the contract referenced by
  // selectionruntime.IslandGlobals in runtime.go; the BridgeShim there
  // delegates window.GoSXStudioSelectionRuntime.bind calls here when the
  // feature flag is on.
  window.__gosx_selection_runtime_island_bind = bindSelectionSurfaceIsland;
})();
