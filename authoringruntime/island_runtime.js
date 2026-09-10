;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  var SELECTED_ATTR = "data-gosx-studio-authoring-selected";
  var STATE_ATTR = "data-gosx-studio-authoring-state";
  var MESSAGE_ATTR = "data-gosx-studio-authoring-message";
  var PREVIEW_ATTR = "data-gosx-studio-authoring-preview-url";
  var DRAFT_ATTR = "data-gosx-studio-authoring-draft-id";
  var CHANGE_KEY_ATTR = "data-gosx-studio-authoring-change-key";
  var CHANGE_KIND_ATTR = "data-gosx-studio-authoring-change-kind";
  var CHANGE_PAGE_ATTR = "data-gosx-studio-authoring-change-page";
  var CHANGE_COMPONENT_ATTR = "data-gosx-studio-authoring-change-component";
  var CHANGE_BINDING_ATTR = "data-gosx-studio-authoring-change-binding";
  var SELECTED_COUNT_ATTR = "data-gosx-studio-authoring-selected-count";
  var MANAGED_FORM_ATTR = "data-gosx-studio-authoring-managed";
  var FORM_STATE_ATTR = "data-gosx-form-state";
  var FORM_PENDING_ATTR = "data-gosx-pending";

  function toObject(value) {
    if (!value) return {};
    if (typeof value === "string") {
      try {
        var parsed = JSON.parse(value);
        return parsed && typeof parsed === "object" ? parsed : {};
      } catch (e) {
        return {};
      }
    }
    return typeof value === "object" ? value : {};
  }

  function hasKeys(value) {
    return value && typeof value === "object" && Object.keys(value).length > 0;
  }

  function looksLikeAuthoringData(data, meta) {
    if (!data || typeof data !== "object") return false;
    if (data.refreshPreview !== undefined) return true;
    if (data.previewURL || data.draftID || data.changeCount !== undefined) return true;
    if (Array.isArray(data.changes)) return true;
    var action = String(meta && meta.action || "").toLowerCase();
    return action.indexOf("authoring") !== -1;
  }

  function dataFromResult(result, meta) {
    result = toObject(result);
    var data = toObject(result.data);
    if (!hasKeys(data) && looksLikeAuthoringData(result, meta)) {
      data = result;
    }
    if (!hasKeys(data)) return null;
    if (!looksLikeAuthoringData(data, meta)) return null;
    if (result.message && !data.message) data.message = String(result.message);
    return data;
  }

  function queryAll(selector, root) {
    try {
      return Array.prototype.slice.call((root || doc).querySelectorAll(selector));
    } catch (e) {
      return [];
    }
  }

  function closest(node, selector) {
    try {
      return node && node.closest ? node.closest(selector) : null;
    } catch (e) {
      return null;
    }
  }

  function selectorValue(value) {
    value = String(value || "");
    if (window.CSS && typeof window.CSS.escape === "function") {
      return window.CSS.escape(value);
    }
    return value.replace(/\\/g, "\\\\").replace(/"/g, "\\\"");
  }

  function attrSelector(attr, value) {
    value = String(value || "").trim();
    if (!value) return "";
    return "[" + attr + "=\"" + selectorValue(value) + "\"]";
  }

  function pushAttr(selectors, attr, value) {
    var selector = attrSelector(attr, value);
    if (selector) selectors.push(selector);
  }

  function firstChange(data) {
    var changes = Array.isArray(data && data.changes) ? data.changes : [];
    return toObject(changes[0]);
  }

  function roots() {
    var seen = [];
    [
      "[data-gosx-studio-workbench]",
      "[data-studio-workbench]",
      ".gosx-studio",
      ".editor-workbench",
      ".studio-workbench"
    ].forEach(function (selector) {
      queryAll(selector).forEach(function (node) {
        if (seen.indexOf(node) === -1) seen.push(node);
      });
    });
    return seen;
  }

  function setOptionalAttr(node, attr, value) {
    value = String(value || "").trim();
    if (value) {
      node.setAttribute(attr, value);
    } else {
      node.removeAttribute(attr);
    }
  }

  function changeSelectors(change) {
    change = toObject(change);
    var selectors = [];
    pushAttr(selectors, "data-studio-site-map-component", change.component);
    pushAttr(selectors, "data-studio-block-key", change.component);
    pushAttr(selectors, "data-block-studio-block", change.component);
    pushAttr(selectors, "data-studio-component", change.component);
    pushAttr(selectors, "data-studio-site-map-page", change.pageKey);
    pushAttr(selectors, "data-studio-site-page", change.pageKey);
    pushAttr(selectors, "data-studio-site-map-workspace-node", change.pageKey ? "page:" + change.pageKey : "");
    pushAttr(selectors, "data-gosx-studio-canvas-node", change.pageKey);
    pushAttr(selectors, "data-studio-site-map-binding", change.binding);
    pushAttr(selectors, "data-studio-field", change.binding);
    pushAttr(selectors, "data-studio-field-source", change.binding);
    pushAttr(selectors, "data-editor-source", change.binding);
    pushAttr(selectors, "data-gosx-studio-authoring-binding", change.binding);
    pushAttr(selectors, "data-studio-site-map-workspace-node", change.key);
    pushAttr(selectors, "data-studio-site-map-control", change.key);
    pushAttr(selectors, "data-gosx-studio-canvas-node", change.key);
    pushAttr(selectors, "data-studio-flow-card", change.key);
    pushAttr(selectors, "data-studio-flow-editor", change.key);
    return selectors;
  }

  function clearSelected() {
    queryAll("[" + SELECTED_ATTR + "='true']").forEach(function (node) {
      node.removeAttribute(SELECTED_ATTR);
    });
  }

  function syncCanvasSelection(selected) {
    var canvasNode = null;
    selected.some(function (node) {
      if (node && node.getAttribute && node.getAttribute("data-gosx-studio-canvas-node")) {
        canvasNode = node;
        return true;
      }
      return false;
    });
    if (!canvasNode) return;
    var canvas = closest(canvasNode, "[data-gosx-studio-site-canvas]") || closest(canvasNode, "[data-gosx-studio-canvas]");
    if (!canvas) return;
    var key = canvasNode.getAttribute("data-gosx-studio-canvas-node") || "";
    queryAll("[data-gosx-studio-canvas-node]", canvas).forEach(function (node) {
      var active = node === canvasNode;
      if (node.classList) node.classList.toggle("is-selected", active);
      node.setAttribute("aria-pressed", active ? "true" : "false");
    });
    if (key) canvas.setAttribute("data-gosx-studio-canvas-selected", key);
    if (typeof window.CustomEvent === "function") {
      canvas.dispatchEvent(new CustomEvent("gosxstudio:canvas-select", {
        bubbles: true,
        detail: {
          key: key,
          kind: canvasNode.getAttribute("data-gosx-studio-canvas-node-kind") || "",
          label: canvasNode.getAttribute("data-gosx-studio-canvas-node-label") || canvasNode.textContent.trim(),
          href: canvasNode.getAttribute("data-gosx-studio-canvas-node-href") || "",
          reason: "authoring-result"
        }
      }));
    }
  }

  function selectChange(change) {
    clearSelected();
    var selected = [];
    changeSelectors(change).forEach(function (selector) {
      queryAll(selector).forEach(function (node) {
        if (selected.indexOf(node) === -1) selected.push(node);
      });
    });
    selected.forEach(function (node) {
      node.setAttribute(SELECTED_ATTR, "true");
    });
    syncCanvasSelection(selected);
    if (selected[0] && typeof selected[0].scrollIntoView === "function") {
      try {
        selected[0].scrollIntoView({ block: "nearest", inline: "nearest" });
      } catch (e) {
        selected[0].scrollIntoView();
      }
    }
    return selected.length;
  }

  function previewFrames() {
    var frames = [];
    [
      "[data-gosx-studio-preview='true'] iframe",
      "iframe[data-gosx-studio-preview-frame]",
      "iframe.editor-preview-frame"
    ].forEach(function (selector) {
      queryAll(selector).forEach(function (frame) {
        if (frame && String(frame.tagName || "").toLowerCase() === "iframe" && frames.indexOf(frame) === -1) {
          frames.push(frame);
        }
      });
    });
    return frames;
  }

  function previewContainer(frame) {
    return closest(frame, "[data-gosx-studio-preview='true']") || closest(frame, "[data-gosx-studio-preview]");
  }

  function previewURLForFrame(frame, rawURL) {
    var container = previewContainer(frame);
    var target = String(rawURL || "").trim();
    if (!target && container) target = String(container.getAttribute("data-gosx-studio-preview-url") || "").trim();
    if (!target) target = String(frame.getAttribute("src") || "").trim();
    if (!target) return "";
    try {
      var url = new URL(target, window.location.href);
      var current = new URL(String(frame.getAttribute("src") || target), window.location.href);
      if (!url.searchParams.has("gosx-preview") && current.searchParams.has("gosx-preview")) {
        url.searchParams.set("gosx-preview", current.searchParams.get("gosx-preview") || "1");
      }
      url.searchParams.set("gosx-studio-refresh", String(Date.now()));
      return url.href;
    } catch (e) {
      return target;
    }
  }

  function remountPreviewRuntime() {
    var runtime = window.GoSXStudioPreviewRuntime;
    if (runtime && typeof runtime.mount === "function") {
      try {
        runtime.mount(doc);
      } catch (e) {
        return;
      }
    }
  }

  function remountEditorRuntimes(root) {
    root = root || doc;
    remountPreviewRuntime();
    var workbench = window.GoSXStudioWorkbenchRuntime || {};
    [
      ["bindRailResizers", root],
      ["bindChrome", root]
    ].forEach(function (call) {
      if (typeof workbench[call[0]] === "function") {
        try { workbench[call[0]](call[1]); } catch (e) {}
      }
    });
    var block = window.GoSXStudioBlockLayoutRuntime || {};
    [
      ["bindLibrary", root],
      ["bindVisibility", root],
      ["bindList", root],
      ["bindHandleDrag", root],
      ["updateBlockLibraryState", root]
    ].forEach(function (call) {
      if (typeof block[call[0]] === "function") {
        try { block[call[0]](call[1]); } catch (e) {}
      }
    });
    var operation = window.GoSXStudioOperationRuntime || {};
    if (typeof operation.bind === "function") {
      try { operation.bind(root); } catch (e) {}
    }
    var content = window.GoSXStudioContentEditorRuntime || {};
    if (typeof content.init === "function") {
      try { content.init(root); } catch (e) {}
    }
    var sitemap = window.GoSXStudioSiteMapRuntime || {};
    if (typeof sitemap.bindAll === "function") {
      try { sitemap.bindAll(root); } catch (e) {}
    }
    var media = window.GoSXStudioMediaRuntime || {};
    if (typeof media.init === "function") {
      try { media.init(root); } catch (e) {}
    }
    var style = window.GoSXStudioStyleRuntime || {};
    [
      ["bindTheme", root],
      ["bindWorkbench", root],
      ["bindCSS", root],
      ["bindFonts", root]
    ].forEach(function (call) {
      if (typeof style[call[0]] === "function") {
        try { style[call[0]](call[1]); } catch (e) {}
      }
    });
    var selection = window.GoSXStudioSelectionRuntime || {};
    if (typeof selection.bind === "function") {
      try { selection.bind(root); } catch (e) {}
    }
  }

  function refreshPreview(data) {
    if (!data || (!data.refreshPreview && !data.previewURL)) return 0;
    var frames = previewFrames();
    frames.forEach(function (frame) {
      var container = previewContainer(frame);
      var url = previewURLForFrame(frame, data.previewURL);
      if (!url) return;
      if (container) {
        container.setAttribute("data-gosx-studio-preview-state", "refreshing");
        container.setAttribute("data-gosx-studio-preview-url", data.previewURL || url);
      }
      try {
        frame.addEventListener("load", function () {
          if (container) container.setAttribute("data-gosx-studio-preview-state", "ready");
          remountPreviewRuntime();
        }, { once: true });
      } catch (e) {
        remountPreviewRuntime();
      }
      frame.setAttribute("src", url);
    });
    return frames.length;
  }

  function writeSaveFeedback(message) {
    message = String(message || "").trim();
    var blocked = roots().some(function (root) {
      var state = root.getAttribute && root.getAttribute(STATE_ATTR);
      return state === "dirty" || state === "pending" || state === "error";
    });
    if (blocked) return;
    if (message) {
      queryAll("[data-gosx-studio-save-detail]").forEach(function (node) {
        node.textContent = message;
      });
    }
    queryAll("[data-gosx-studio-save-state='true']").forEach(function (node) {
      node.textContent = "Saved";
    });
  }

  function fragmentSpecs(data) {
    var raw = data && (data.fragments || data.refreshFragments || data.fragmentSelectors || data.refreshFragmentSelectors);
    if (!Array.isArray(raw)) return [];
    var specs = [];
    raw.forEach(function (item) {
      var spec = typeof item === "string" ? { selector: item } : toObject(item);
      var selector = String(spec.selector || spec.target || "").trim();
      if (!selector) return;
      var mode = String(spec.mode || "replace").trim();
      if (mode !== "inner") mode = "replace";
      specs.push({
        key: String(spec.key || selector).trim(),
        selector: selector,
        mode: mode
      });
    });
    return specs;
  }

  function fragmentRefreshURL(data) {
    var target = String(data && (data.fragmentURL || data.refreshURL) || "").trim();
    if (!target) target = window.location.href;
    try {
      return new URL(target, window.location.href).href;
    } catch (e) {
      return window.location.href;
    }
  }

  function replaceFragment(current, fresh, mode) {
    if (!current || !fresh) return false;
    if (mode === "inner") {
      current.innerHTML = fresh.innerHTML;
      return true;
    }
    current.replaceWith(fresh.cloneNode(true));
    return true;
  }

  function focusSelector(node) {
    if (!node || !node.getAttribute) return "";
    var id = String(node.getAttribute("id") || "").trim();
    if (id) return "#" + selectorValue(id);
    var binding = String(node.getAttribute("data-gosx-studio-authoring-binding") || node.getAttribute("data-studio-field") || "").trim();
    if (binding) return attrSelector("data-gosx-studio-authoring-binding", binding) + ", " + attrSelector("data-studio-field", binding);
    var name = String(node.getAttribute("name") || "").trim();
    if (name) return "[name=\"" + selectorValue(name) + "\"]";
    return "";
  }

  function captureFocus() {
    var active = doc.activeElement;
    if (!active || active === doc.body || active === doc.documentElement) return null;
    var selector = focusSelector(active);
    if (!selector) return null;
    return {
      selector: selector,
      start: typeof active.selectionStart === "number" ? active.selectionStart : null,
      end: typeof active.selectionEnd === "number" ? active.selectionEnd : null
    };
  }

  function restoreFocus(snapshot) {
    if (!snapshot || !snapshot.selector) return;
    var target = queryAll(snapshot.selector, doc)[0];
    if (!target || typeof target.focus !== "function") return;
    try {
      target.focus({ preventScroll: true });
    } catch (e) {
      target.focus();
    }
    if (snapshot.start !== null && typeof target.setSelectionRange === "function") {
      try {
        target.setSelectionRange(snapshot.start, snapshot.end === null ? snapshot.start : snapshot.end);
      } catch (e) {}
    }
  }

  function mutableControlSelector(node) {
    if (!node || !node.getAttribute) return "";
    var id = String(node.getAttribute("id") || "").trim();
    if (id) return "#" + selectorValue(id);
    var binding = String(node.getAttribute("data-gosx-studio-authoring-binding") || node.getAttribute("data-studio-field") || "").trim();
    if (binding) return attrSelector("data-gosx-studio-authoring-binding", binding) + ", " + attrSelector("data-studio-field", binding);
    var name = String(node.getAttribute("name") || "").trim();
    if (name) return "[name=\"" + selectorValue(name) + "\"]";
    return "";
  }

  function mutableControlState(node) {
    if (!node || node.disabled) return null;
    var tag = String(node.tagName || "").toLowerCase();
    if (tag !== "input" && tag !== "textarea" && tag !== "select") return null;
    var type = String(node.type || "").toLowerCase();
    if (type === "hidden" || type === "submit" || type === "button" || type === "reset" || type === "file") return null;
    var selector = mutableControlSelector(node);
    if (!selector) return null;
    return {
      selector: selector,
      value: "value" in node ? String(node.value) : "",
      checked: "checked" in node ? !!node.checked : null
    };
  }

  function captureMutableControls(root) {
    var snapshot = {};
    queryAll("input, textarea, select", root || doc).forEach(function (node) {
      var state = mutableControlState(node);
      if (state) snapshot[state.selector] = state;
    });
    return snapshot;
  }

  function changedMutableControls(root, baseline) {
    baseline = baseline || {};
    var changed = {};
    Object.keys(baseline).forEach(function (selector) {
      var current = mutableControlState(queryAll(selector, root || doc)[0]);
      var before = baseline[selector];
      if (!current || !before) return;
      if (current.value !== before.value || current.checked !== before.checked) changed[selector] = current;
    });
    return changed;
  }

  function restoreMutableControls(snapshot) {
    snapshot = snapshot || {};
    var count = 0;
    Object.keys(snapshot).forEach(function (selector) {
      var state = snapshot[selector];
      var node = queryAll(selector, doc)[0];
      if (!node) return;
      if ("checked" in node && state.checked !== null) node.checked = !!state.checked;
      if ("value" in node) {
        node.value = state.value;
        node.setAttribute("value", state.value);
      }
      var form = closest(node, "form");
      if (form && form.setAttribute) {
        form.setAttribute(FORM_STATE_ATTR, "dirty");
        if (form.removeAttribute) form.removeAttribute(FORM_PENDING_ATTR);
      }
      count += 1;
    });
    return count;
  }

  function markPreservedEditsDirty() {
    var write = function () {
      roots().forEach(function (root) {
        root.setAttribute(STATE_ATTR, "dirty");
        setOptionalAttr(root, MESSAGE_ATTR, "Unsaved changes");
      });
      queryAll("[data-gosx-studio-save-state='true']").forEach(function (node) {
        node.textContent = "Unsaved";
      });
      queryAll("[data-gosx-studio-save-detail]").forEach(function (node) {
        node.textContent = "Unsaved changes";
      });
    };
    write();
    if (typeof window.setTimeout === "function") {
      window.setTimeout(write, 0);
      window.setTimeout(write, 100);
    }
  }

  function applyFragmentDocument(sourceDoc, specs, meta) {
    meta = meta || {};
    var focus = captureFocus();
    var changedControls = changedMutableControls(doc, meta.submittedControls);
    var count = 0;
    specs.forEach(function (spec) {
      var current = queryAll(spec.selector, doc);
      var fresh = queryAll(spec.selector, sourceDoc);
      var limit = Math.min(current.length, fresh.length);
      for (var index = 0; index < limit; index += 1) {
        if (replaceFragment(current[index], fresh[index], spec.mode)) count += 1;
      }
    });
    if (count > 0) {
      remountEditorRuntimes(doc);
      meta.preservedEditCount = restoreMutableControls(changedControls);
      restoreFocus(focus);
    }
    return count;
  }

  function emitFragmentRefresh(data, specs, url, count) {
    if (typeof window.CustomEvent !== "function" || typeof doc.dispatchEvent !== "function") return;
    var detail = {
      url: url,
      count: count,
      selectors: specs.map(function (spec) { return spec.selector; }),
      result: data
    };
    doc.dispatchEvent(new CustomEvent("gosxstudio:fragments-refresh", { detail: detail }));
    doc.dispatchEvent(new CustomEvent("gosxstudio:fragments-refreshed", { detail: detail }));
  }

  function refreshFragments(data, meta) {
    meta = meta || {};
    var specs = fragmentSpecs(data);
    if (!specs.length) return Promise.resolve(0);
    var url = fragmentRefreshURL(data);
    return fetch(url, {
      method: "GET",
      credentials: "same-origin",
      headers: {
        Accept: "text/html",
        "X-Requested-With": "XMLHttpRequest"
      }
    }).then(function (response) {
      if (!response || !response.ok) return "";
      return response.text();
    }).then(function (html) {
      if (!html) {
        emitFragmentRefresh(data, specs, url, 0);
        return 0;
      }
      if (meta.isCurrent && !meta.isCurrent()) return 0;
      var parsed = new DOMParser().parseFromString(html, "text/html");
      var count = applyFragmentDocument(parsed, specs, meta);
      emitFragmentRefresh(data, specs, url, count);
      return count;
    }, function () {
      emitFragmentRefresh(data, specs, url, 0);
      return 0;
    });
  }

  function markWorkbench(data, change, selectedCount) {
    var message = String(data.message || "").trim();
    roots().forEach(function (root) {
      root.setAttribute(STATE_ATTR, "saved");
      root.setAttribute(SELECTED_COUNT_ATTR, String(selectedCount || 0));
      setOptionalAttr(root, MESSAGE_ATTR, message);
      setOptionalAttr(root, PREVIEW_ATTR, data.previewURL);
      setOptionalAttr(root, DRAFT_ATTR, data.draftID);
      setOptionalAttr(root, CHANGE_KEY_ATTR, change.key);
      setOptionalAttr(root, CHANGE_KIND_ATTR, change.kind);
      setOptionalAttr(root, CHANGE_PAGE_ATTR, change.pageKey);
      setOptionalAttr(root, CHANGE_COMPONENT_ATTR, change.component);
      setOptionalAttr(root, CHANGE_BINDING_ATTR, change.binding);
    });
    writeSaveFeedback(message);
    if (typeof window.setTimeout === "function") {
      window.setTimeout(function () { writeSaveFeedback(message); }, 0);
      window.setTimeout(function () { writeSaveFeedback(message); }, 100);
    }
  }

  function emitResult(data, meta, change, selectedCount, previewCount, fragmentCount) {
    if (typeof window.CustomEvent !== "function" || typeof doc.dispatchEvent !== "function") return;
    doc.dispatchEvent(new CustomEvent("gosxstudio:authoring-result", {
      detail: {
        action: meta && meta.action || "",
        method: meta && meta.method || "",
        result: data,
        change: change,
        selectedCount: selectedCount,
        previewCount: previewCount,
        fragmentCount: fragmentCount || 0
      }
    }));
  }

  function handlePayload(result, meta) {
    meta = meta || {};
    if (meta.ok === false) return null;
    if (meta.isCurrent && !meta.isCurrent()) return null;
    result = toObject(result);
    if (result.ok === false) return null;
    var data = dataFromResult(result, meta);
    if (!data) return null;
    var change = firstChange(data);
    var previewCount = refreshPreview(data);
    var finish = function (fragmentCount) {
      if (meta.isCurrent && !meta.isCurrent()) return null;
      var selectedCount = selectChange(change);
      markWorkbench(data, change, selectedCount);
      markSourcePanel(data, change, meta.sourcePanel);
      if (meta.preservedEditCount > 0) markPreservedEditsDirty();
      emitResult(data, meta, change, selectedCount, previewCount, fragmentCount);
      return {
        result: data,
        change: change,
        selectedCount: selectedCount,
        previewCount: previewCount,
        fragmentCount: fragmentCount || 0
      };
    };
    var specs = fragmentSpecs(data);
    if (specs.length) {
      return refreshFragments(data, meta).then(finish);
    }
    return finish(0);
  }

  function handleManagedFormResult(event) {
    var detail = event && event.detail || {};
    return handlePayload(detail.result, detail);
  }

  function resultValues(data) {
    return toObject(data && data.values);
  }

  function setFirstText(root, selector, value) {
    value = String(value || "").trim();
    if (!root || !value) return;
    var nodes = queryAll(selector, root);
    if (nodes[0]) nodes[0].textContent = value;
  }

  function updatePanelInputs(root, name, value) {
    if (!root || !name) return;
    queryAll("input[name=\"" + selectorValue(name) + "\"]", root).forEach(function (input) {
      input.setAttribute("value", value);
      input.value = value;
    });
  }

  function updateVisibilityPanel(data, panel) {
    if (!panel || !panel.matches || !panel.matches("[data-gosx-studio-component-visibility], [data-studio-site-map-component-visibility]")) return;
    var values = resultValues(data);
    var submitted = String(values.gosx_studio_visible || "").trim().toLowerCase();
    if (submitted !== "true" && submitted !== "false") return;
    var visible = submitted === "true";
    panel.setAttribute("data-gosx-studio-authoring-visibility", visible ? "visible" : "hidden");
    setFirstText(panel, "output", visible ? "Visible" : "Hidden");
    setFirstText(panel, "button", visible ? "Hide section" : "Show section");
    updatePanelInputs(panel, "gosx_studio_visible", visible ? "false" : "true");
  }

  function firstPanelInputValue(panel, name) {
    var inputs = queryAll("input[name=\"" + selectorValue(name) + "\"]", panel);
    if (!inputs[0]) return "";
    return String(inputs[0].value || inputs[0].getAttribute("value") || "");
  }

  function panelPosition(panel) {
    var attr = String(panel && panel.getAttribute && panel.getAttribute("data-gosx-studio-authoring-position") || "").trim();
    if (attr) {
      var parsedAttr = parseInt(attr, 10);
      if (!isNaN(parsedAttr)) return parsedAttr;
    }
    var output = queryAll("output", panel)[0];
    var text = String(output && output.textContent || "").replace(/[^0-9-]/g, "");
    var parsed = parseInt(text, 10);
    return isNaN(parsed) ? -1 : parsed - 1;
  }

  function updateReorderPanel(data, panel) {
    if (!panel || !panel.matches || !panel.matches("[data-gosx-studio-component-reorder], [data-studio-site-map-component-reorder]")) return;
    var values = resultValues(data);
    var submitted = String(values.gosx_studio_position || firstPanelInputValue(panel, "gosx_studio_position")).trim();
    if (!submitted) return;
    var target = parseInt(submitted, 10);
    if (isNaN(target) || target < 0) return;
    var previous = panelPosition(panel);
    var nextTarget = target + 1;
    var nextLabel = "Move down section";
    if (previous < 0 || target > previous) {
      nextTarget = Math.max(0, target - 1);
      nextLabel = "Move up section";
    }
    panel.setAttribute("data-gosx-studio-authoring-position", String(target));
    panel.setAttribute("data-gosx-studio-authoring-next-position", String(nextTarget));
    setFirstText(panel, "output", "#" + String(target + 1));
    setFirstText(panel, "button", nextLabel);
    updatePanelInputs(panel, "gosx_studio_position", String(nextTarget));
  }

  function markSourcePanel(data, change, panel) {
    if (!panel || !panel.setAttribute) return;
    panel.setAttribute(STATE_ATTR, "saved");
    setOptionalAttr(panel, MESSAGE_ATTR, data && data.message);
    setOptionalAttr(panel, CHANGE_KEY_ATTR, change && change.key);
    setOptionalAttr(panel, CHANGE_KIND_ATTR, change && change.kind);
    setOptionalAttr(panel, CHANGE_PAGE_ATTR, change && change.pageKey);
    setOptionalAttr(panel, CHANGE_COMPONENT_ATTR, change && change.component);
    setOptionalAttr(panel, CHANGE_BINDING_ATTR, change && change.binding);
    updateVisibilityPanel(data, panel);
    updateReorderPanel(data, panel);
  }

  function formSubmissionMethod(form, submitter) {
    return String(
      submitter && submitter.getAttribute && submitter.getAttribute("formmethod")
      || form && form.getAttribute && form.getAttribute("method")
      || "post"
    ).trim().toUpperCase();
  }

  function formSubmissionAction(form, submitter) {
    return String(
      submitter && submitter.getAttribute && submitter.getAttribute("formaction")
      || form && form.getAttribute && form.getAttribute("action")
      || window.location.href
    );
  }

  function isSameOrigin(value) {
    try {
      return new URL(value, window.location.href).origin === window.location.origin;
    } catch (e) {
      return false;
    }
  }

  function serializeForm(form, submitter) {
    var formData;
    var capturedSubmitter = false;
    try {
      formData = submitter ? new FormData(form, submitter) : new FormData(form);
      capturedSubmitter = !!submitter;
    } catch (e) {
      formData = new FormData(form);
    }
    var submitterName = submitter && (submitter.name || (submitter.getAttribute && submitter.getAttribute("name")));
    var submitterValue = submitter && (submitter.value || (submitter.getAttribute && submitter.getAttribute("value")) || "");
    if (submitterName && !capturedSubmitter) {
      formData.append(submitterName, submitterValue);
    }
    return formData;
  }

  function formCSRFToken(formData) {
    if (!formData || typeof formData.get !== "function") return "";
    var token = formData.get("csrf_token");
    return token == null ? "" : String(token);
  }

  function captureFormState(form) {
    if (!form || !form.getAttribute) return { pending: null, state: null };
    return {
      pending: form.getAttribute(FORM_PENDING_ATTR),
      state: form.getAttribute(FORM_STATE_ATTR)
    };
  }

  function submitBaselineState(form) {
    if (!form) return { pending: null, state: null };
    if (!form.__gosxStudioAuthoringSubmitBaseline) {
      form.__gosxStudioAuthoringSubmitBaseline = captureFormState(form);
    }
    return form.__gosxStudioAuthoringSubmitBaseline;
  }

  function clearSubmitBaselineState(form) {
    if (form) form.__gosxStudioAuthoringSubmitBaseline = null;
  }

  function setFormPending(form) {
    if (!form || !form.setAttribute) return;
    clearFieldErrors(form);
    if (form.removeAttribute) {
      form.removeAttribute("data-gosx-studio-authoring-error-message");
      form.removeAttribute("data-gosx-studio-authoring-error-status");
    }
    form.setAttribute(FORM_PENDING_ATTR, "true");
    form.setAttribute(FORM_STATE_ATTR, "pending");
  }

  function setFormDirty(form) {
    if (!form || !form.setAttribute) return;
    form.setAttribute(FORM_STATE_ATTR, "dirty");
    if (form.removeAttribute) form.removeAttribute(FORM_PENDING_ATTR);
  }

  function restoreFormState(form, previous) {
    if (!form) return;
    previous = previous || { pending: null, state: null };
    if (previous.pending == null) {
      if (form.removeAttribute) form.removeAttribute(FORM_PENDING_ATTR);
    } else if (form.setAttribute) {
      form.setAttribute(FORM_PENDING_ATTR, previous.pending);
    }
    if (previous.state == null) {
      if (form.setAttribute) form.setAttribute(FORM_STATE_ATTR, "idle");
    } else if (form.setAttribute) {
      form.setAttribute(FORM_STATE_ATTR, previous.state);
    }
  }

  function setFormError(form) {
    if (!form || !form.setAttribute) return;
    form.setAttribute(FORM_STATE_ATTR, "error");
    if (form.removeAttribute) form.removeAttribute(FORM_PENDING_ATTR);
  }

  function errorMessage(result, status) {
    result = toObject(result);
    var message = String(result.message || (result.data && result.data.message) || "").trim();
    if (message) return message;
    if (status === 409) return "This edit conflicts with a newer change.";
    if (status === 422) return "Please correct the highlighted fields.";
    return "This edit could not be saved.";
  }

  function errorFieldErrors(result) {
    result = toObject(result);
    var errors = toObject(result.fieldErrors);
    if (hasKeys(errors)) return errors;
    return toObject(result.data && result.data.fieldErrors);
  }

  function clearFieldErrors(form) {
    if (!form || !form.querySelectorAll) return;
    queryAll("[data-gosx-studio-authoring-field-error]", form).forEach(function (node) {
      node.removeAttribute("data-gosx-studio-authoring-field-error");
      node.removeAttribute("aria-invalid");
    });
    queryAll("[data-gosx-studio-field-error-for]", form).forEach(function (node) {
      node.textContent = "";
      node.setAttribute("hidden", "hidden");
    });
    if (form.removeAttribute) form.removeAttribute("data-gosx-studio-authoring-field-errors");
  }

  function markFieldErrors(form, fieldErrors) {
    fieldErrors = toObject(fieldErrors);
    clearFieldErrors(form);
    if (!form || !hasKeys(fieldErrors)) return;
    try {
      form.setAttribute("data-gosx-studio-authoring-field-errors", JSON.stringify(fieldErrors));
    } catch (e) {}
    Object.keys(fieldErrors).forEach(function (name) {
      var message = String(fieldErrors[name] || "").trim();
      queryAll("[name=\"" + selectorValue(name) + "\"]", form).forEach(function (node) {
        node.setAttribute("aria-invalid", "true");
        node.setAttribute("data-gosx-studio-authoring-field-error", message);
      });
      queryAll("[data-gosx-studio-field-error-for=\"" + selectorValue(name) + "\"]", form).forEach(function (node) {
        node.textContent = message;
        node.removeAttribute("hidden");
      });
    });
  }

  function applySubmitError(form, result, meta) {
    meta = meta || {};
    var message = errorMessage(result, meta.status || 0);
    var fieldErrors = errorFieldErrors(result);
    setFormError(form);
    if (form && form.setAttribute) {
      form.setAttribute("data-gosx-studio-authoring-error-message", message);
      form.setAttribute("data-gosx-studio-authoring-error-status", String(meta.status || 0));
    }
    roots().forEach(function (root) {
      root.setAttribute(STATE_ATTR, "error");
      setOptionalAttr(root, MESSAGE_ATTR, message);
    });
    queryAll("[data-gosx-studio-save-state='true']").forEach(function (node) {
      node.textContent = "Error";
    });
    queryAll("[data-gosx-studio-save-detail]").forEach(function (node) {
      node.textContent = message;
    });
    markFieldErrors(form, fieldErrors);
    if (typeof window.CustomEvent === "function" && typeof doc.dispatchEvent === "function") {
      doc.dispatchEvent(new CustomEvent("gosxstudio:authoring-error", {
        detail: {
          action: meta.action || "",
          method: meta.method || "",
          status: meta.status || 0,
          message: message,
          fieldErrors: fieldErrors,
          result: toObject(result)
        }
      }));
    }
  }

  function nextSubmitSequence(form) {
    if (!form) return 0;
    form.__gosxStudioAuthoringSubmitSequence = (form.__gosxStudioAuthoringSubmitSequence || 0) + 1;
    return form.__gosxStudioAuthoringSubmitSequence;
  }

  function currentSubmitSequence(form, sequence) {
    return !form || form.__gosxStudioAuthoringSubmitSequence === sequence;
  }

  function formNavigationURL(url, formData) {
    var next = new URL(url.href);
    var params = new URLSearchParams();
    if (formData && typeof formData.forEach === "function") {
      formData.forEach(function (value, key) {
        params.append(String(key), value == null ? "" : String(value));
      });
    }
    next.search = params.toString();
    return next;
  }

  function parseJSONResponse(response) {
    return response.json().catch(function () {
      return null;
    });
  }

  function sourcePanelForSubmitter(submitter) {
    return closest(submitter, [
      "[data-gosx-studio-editable-control]",
      "[data-gosx-studio-page-metadata]",
      "[data-gosx-studio-component-visibility]",
      "[data-gosx-studio-component-reorder]",
      "[data-gosx-studio-component-duplicate]",
      "[data-gosx-studio-component-delete]",
      "[data-studio-composition-intent-apply]",
      "[data-studio-site-map-page-edit]",
      "[data-studio-site-map-component-visibility]",
      "[data-studio-site-map-component-reorder]",
      "[data-studio-site-map-component-duplicate]",
      "[data-studio-site-map-component-delete]"
    ].join(", "));
  }

  function submitAuthoringManagedForm(form, submitter) {
    var method = formSubmissionMethod(form, submitter);
    var action = formSubmissionAction(form, submitter) || window.location.href;
    var url;
    try {
      url = new URL(action, window.location.href);
    } catch (e) {
      setFormError(form);
      return;
    }
    var formData = serializeForm(form, submitter);
    var previous = submitBaselineState(form);
    var submittedControls = captureMutableControls(form);
    var csrfToken = formCSRFToken(formData);
    var sourcePanel = sourcePanelForSubmitter(submitter);
    var sequence = nextSubmitSequence(form);

    setFormPending(form);

    if ((method !== "GET" && method !== "POST") || !isSameOrigin(url.href)) {
      applySubmitError(form, {
        ok: false,
        message: "This edit cannot be submitted from inside the editor."
      }, { action: url.href, method: method, status: 0 });
      clearSubmitBaselineState(form);
      return;
    }

    if (method === "GET") {
      if (!currentSubmitSequence(form, sequence)) return;
      handlePayload({
        ok: true,
        data: {
          message: "",
          previewURL: formNavigationURL(url, formData).href,
          refreshPreview: true
        }
      }, {
        action: url.href,
        method: method,
        ok: true,
        sourcePanel: sourcePanel,
        submittedControls: submittedControls,
        isCurrent: function () { return currentSubmitSequence(form, sequence); }
      });
      if (hasKeys(changedMutableControls(form, submittedControls))) {
        setFormDirty(form);
        markPreservedEditsDirty();
      } else {
        restoreFormState(form, previous);
      }
      clearSubmitBaselineState(form);
      return;
    }

    fetch(url.href, {
      method: method,
      headers: {
        Accept: "application/json",
        "X-Requested-With": "XMLHttpRequest",
        "X-CSRF-Token": csrfToken
      },
      body: formData,
      credentials: "same-origin",
      redirect: "follow"
    }).then(function (response) {
      return parseJSONResponse(response).then(function (result) {
        return { response: response, result: result };
      });
    }).then(function (payload) {
      if (!currentSubmitSequence(form, sequence)) return;
      if (!payload.response || !payload.response.ok) {
        applySubmitError(form, payload.result, {
          action: url.href,
          method: method,
          status: payload.response ? payload.response.status : 0
        });
        clearSubmitBaselineState(form);
        return;
      }
      var meta = {
        action: url.href,
        method: method,
        ok: true,
        status: payload.response ? payload.response.status : 0,
        sourcePanel: sourcePanel,
        submittedControls: submittedControls,
        isCurrent: function () { return currentSubmitSequence(form, sequence); }
      };
      return Promise.resolve(handlePayload(payload.result, meta)).then(function (handled) {
        if (!currentSubmitSequence(form, sequence)) return;
        if (!handled) {
          applySubmitError(form, {
            ok: false,
            message: "Studio action failed; no structured authoring response.",
            result: payload.result
          }, { action: url.href, method: method, status: payload.response ? payload.response.status : 0 });
          clearSubmitBaselineState(form);
          return;
        }
        if (meta.preservedEditCount > 0 || hasKeys(changedMutableControls(form, submittedControls))) {
          setFormDirty(form);
          markPreservedEditsDirty();
        } else {
          restoreFormState(form, previous);
        }
        clearSubmitBaselineState(form);
      });
    }, function () {
      if (!currentSubmitSequence(form, sequence)) return;
      setFormError(form);
      clearSubmitBaselineState(form);
    });
  }

  function shouldHandleAuthoringForm(form, event) {
    if (!form || !form.hasAttribute || !form.hasAttribute(MANAGED_FORM_ATTR)) return false;
    if (event && event.defaultPrevented) return false;
    return true;
  }

  function handleAuthoringFormSubmit(event) {
    var form = event && event.target;
    if (!shouldHandleAuthoringForm(form, event)) return;
    event.preventDefault();
    if (typeof event.stopImmediatePropagation === "function") {
      event.stopImmediatePropagation();
    } else if (typeof event.stopPropagation === "function") {
      event.stopPropagation();
    }
    submitAuthoringManagedForm(form, event.submitter || null);
  }

  window.GoSXStudioAuthoringRuntime = {
    handleEvent: handleManagedFormResult,
    handleResult: function (result, meta) {
      return handlePayload(result, meta || {});
    },
    handleSubmit: handleAuthoringFormSubmit,
    refreshPreview: refreshPreview,
    selectChange: selectChange
  };

  if (window.__gosx_studio_authoring_runtime_bound !== "true") {
    window.__gosx_studio_authoring_runtime_bound = "true";
    doc.addEventListener("submit", handleAuthoringFormSubmit, true);
    doc.addEventListener("gosx:form:result", handleManagedFormResult);
  }
}());
