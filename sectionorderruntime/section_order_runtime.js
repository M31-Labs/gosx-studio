;(function () {
  "use strict";
  if (typeof window === "undefined" || !window.document) return;

  var doc = window.document;
  var ROOT = "[data-gosx-studio-section-order='true']";
  var LIST = "[data-gosx-studio-section-order-list='true']";
  var ITEM = "[data-gosx-studio-section-order-item]";
  var HANDLE = "[data-gosx-studio-section-order-handle='true']";
  var PREVIEW_HANDLE = "[data-gosx-studio-section-order-preview-handle='true']";
  var MOVE = "[data-gosx-studio-section-order-move]";
  var HISTORY = "[data-gosx-studio-section-order-history]";
  var FORM = "[data-gosx-studio-section-order-form='true']";
  var SECTION = "[data-gosx-studio-section-key]";
  var GLOBAL_STATE = "__gosxStudioSectionOrderRuntimeState";

  // A managed-navigation fragment can re-evaluate this asset. Reuse the
  // existing controller in that case so document/window listeners remain
  // singleton-owned and the navigation event does not double-mount a root.
  var existingState = window[GLOBAL_STATE];
  if (existingState && typeof existingState.mount === "function") {
    window.GoSXStudioSectionOrderRuntime = existingState.api || { mount: existingState.mount };
    existingState.mount(doc);
    return;
  }

  function queryAll(root, selector) {
    try { return Array.prototype.slice.call((root || doc).querySelectorAll(selector)); } catch (e) { return []; }
  }

  function closest(node, selector) {
    try { return node && node.closest ? node.closest(selector) : null; } catch (e) { return null; }
  }

  function text(value) {
    return String(value == null ? "" : value).trim();
  }

  function routePath(value) {
    value = text(value) || "/";
    try { return new URL(value, window.location.href).pathname || "/"; } catch (e) { return value.split("?")[0] || "/"; }
  }

  function sameSet(a, b) {
    if (a.length !== b.length) return false;
    var seen = Object.create(null);
    a.forEach(function (key) { seen[key] = (seen[key] || 0) + 1; });
    for (var index = 0; index < b.length; index += 1) {
      if (!seen[b[index]]) return false;
      seen[b[index]] -= 1;
    }
    return true;
  }

  function uniqueKeys(items) {
    var out = [];
    items.forEach(function (node) {
      var key = text(node.getAttribute("data-gosx-studio-section-order-item"));
      if (key && out.indexOf(key) === -1) out.push(key);
    });
    return out;
  }

  function moveKey(order, key, targetIndex) {
    var current = order.indexOf(key);
    if (current < 0) return order.slice();
    var next = order.slice();
    next.splice(current, 1);
    if (targetIndex > current) targetIndex -= 1;
    targetIndex = Math.max(0, Math.min(next.length, targetIndex));
    next.splice(targetIndex, 0, key);
    return next;
  }

  function setDisabled(button, disabled) {
    if (!button) return;
    if (disabled) button.setAttribute("disabled", "disabled");
    else button.removeAttribute("disabled");
  }

  function hiddenInput(form, name, value, attrs) {
    var input = doc.createElement("input");
    input.type = "hidden";
    input.name = name;
    input.value = text(value);
    Object.keys(attrs || {}).forEach(function (key) { input.setAttribute(key, attrs[key]); });
    form.appendChild(input);
    return input;
  }

  function createRuntime(root) {
    var list = root.querySelector(LIST);
    var adapter = root.querySelector(FORM);
    // The adapter is deliberately a non-form metadata node. PageCanvas is
    // normally rendered inside the workbench's primary authoring form, so
    // reusing an adapter <form> here would create invalid nested HTML. Every
    // durable operation gets its own detached form appended to <body>.
    var form = null;
    var status = root.querySelector("[data-gosx-studio-section-order-status='true']");
    var frame = closest(root, "[data-gosx-studio-preview='true']") && closest(root, "[data-gosx-studio-preview='true']").querySelector("iframe[data-studio-preview-frame='true'], iframe[data-gosx-studio-preview-frame], iframe");
    var operation = null;
    var frameDoc = null;
    var overlay = null;
    var drag = null;
    var selectedKey = "";
    var listenersBound = false;
    var frameReady = false;
    var operationSequence = 0;
    var previewPresent = Object.create(null);
    var pending = false;
    var disposed = false;
    var listeners = [];
    var order = list ? uniqueKeys(queryAll(list, ITEM)) : [];
    // `order` is the visible editor intent. Keep the last acknowledged full
    // order separately so a set-field operation can carry a value CAS for the
    // value that existed before its optimistic reorder.
    var committedOrder = order.slice();
    var enabled = root.getAttribute("data-gosx-studio-section-order-enabled") === "true";
    var configuredEnabled = enabled;
    var serverDisabledReason = text(root.getAttribute("data-gosx-studio-section-order-disabled-reason"));
    var route = routePath(root.getAttribute("data-gosx-studio-section-order-route") || "/");
    var sourceRoute = routePath(root.getAttribute("data-gosx-studio-section-order-source-route") || route);
    var previewRoute = routePath(root.getAttribute("data-gosx-studio-section-order-preview-route") || sourceRoute);

    function isConnected() {
      if (disposed) return false;
      if (root.isConnected !== undefined && !root.isConnected) return false;
      return !!(doc.documentElement && doc.documentElement.contains(root));
    }

    function ownsNode(node) {
      return !!node && (root.contains(node) || (overlay && overlay.contains(node)));
    }

    function listen(target, type, handler, options) {
      if (!target || !target.addEventListener) return;
      target.addEventListener(type, handler, options);
      listeners.push({ target: target, type: type, handler: handler, options: options });
    }

    function removeListeners() {
      listeners.forEach(function (entry) {
        entry.target.removeEventListener(entry.type, entry.handler, entry.options);
      });
      listeners = [];
    }

    function setStatus(message, state) {
      if (status) status.textContent = message;
      if (state) root.setAttribute("data-gosx-studio-section-order-state", state);
    }

    function setReadOnly(reason) {
      enabled = false;
      root.setAttribute("data-gosx-studio-section-order-enabled", "false");
      if (reason) {
        root.setAttribute("data-gosx-studio-section-order-disabled-reason", reason);
        setStatus(reason, "readonly");
      }
      removeOverlay();
      syncMoveButtons();
    }

    function clearRuntimeDisabledReason() {
      if (!serverDisabledReason) root.removeAttribute("data-gosx-studio-section-order-disabled-reason");
    }

    function readFrameDocument() {
      if (!frame) return null;
      try {
        if (frame.contentWindow && frame.contentWindow.location && frame.contentWindow.location.origin !== window.location.origin) return null;
        return frame.contentDocument || (frame.contentWindow && frame.contentWindow.document) || null;
      } catch (e) {
        return null;
      }
    }

    function frameIsSameOrigin() {
      if (!frame) return false;
      try {
        var frameOrigin = frame.contentWindow && frame.contentWindow.location && frame.contentWindow.location.origin;
        return !frameOrigin || frameOrigin === window.location.origin;
      } catch (e) {
        return false;
      }
    }

    function frameSections() {
      frameDoc = readFrameDocument();
      if (!frameDoc) return [];
      return queryAll(frameDoc, SECTION);
    }

    function currentPreviewRoute() {
      if (!frame) return previewRoute;
      try {
        if (frame.contentWindow && frame.contentWindow.location) return routePath(frame.contentWindow.location.href);
      } catch (e) {}
      return routePath(frame.getAttribute("src") || previewRoute);
    }

    function syncMoveButtons() {
      if (disposed) return;
      var items = list ? queryAll(list, ITEM) : [];
      items.forEach(function (item, index) {
        var locked = item.getAttribute("data-gosx-studio-section-order-locked") === "true";
        var key = text(item.getAttribute("data-gosx-studio-section-order-item"));
        var optionalUnavailable = item.getAttribute("data-gosx-studio-section-order-preview-optional") === "true" && frameReady && !previewPresent[key];
        item.setAttribute("data-gosx-studio-section-order-preview-present", optionalUnavailable ? "false" : "true");
        setDisabled(item.querySelector("[data-gosx-studio-section-order-move='up']"), pending || !enabled || locked || optionalUnavailable || index === 0);
        setDisabled(item.querySelector("[data-gosx-studio-section-order-move='down']"), pending || !enabled || locked || optionalUnavailable || index === items.length - 1);
        setDisabled(item.querySelector(HANDLE), pending || !enabled || locked || optionalUnavailable);
        item.setAttribute("aria-selected", text(item.getAttribute("data-gosx-studio-section-order-item")) === selectedKey ? "true" : "false");
      });
      syncHistoryButtons();
    }

    function setPreviewPresence(keys) {
      previewPresent = Object.create(null);
      (keys || []).forEach(function (key) { previewPresent[key] = true; });
    }

    function syncHistoryButtons() {
      var undo = root.querySelector("[data-gosx-studio-section-order-history='undo']");
      var redo = root.querySelector("[data-gosx-studio-section-order-history='redo']");
      if (undo) setDisabled(undo, pending || !enabled || !text(undo.getAttribute("data-gosx-studio-history-operation-id")));
      if (redo) setDisabled(redo, pending || !enabled || !text(redo.getAttribute("data-gosx-studio-history-operation-id")));
    }

    function reorderLane(nextOrder) {
      if (disposed || !list) return;
      var byKey = Object.create(null);
      queryAll(list, ITEM).forEach(function (item) {
        byKey[text(item.getAttribute("data-gosx-studio-section-order-item"))] = item;
        item.removeAttribute("data-gosx-studio-section-order-drop");
        item.removeAttribute("data-gosx-studio-section-order-dragging");
      });
      nextOrder.forEach(function (key) {
        if (byKey[key]) list.appendChild(byKey[key]);
      });
      order = nextOrder.slice();
      syncMoveButtons();
      positionOverlay();
    }

    function sectionKey(node) {
      return text(node && node.getAttribute && node.getAttribute("data-gosx-studio-section-key"));
    }

    function labelForSection(section, key) {
      return text(section.getAttribute("data-gosx-studio-section-label")) || text(root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(key) + "'] [data-gosx-studio-section-order-name]") && root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(key) + "'] [data-gosx-studio-section-order-name]").textContent) || key;
    }

    function cssEscape(value) {
      if (window.CSS && typeof window.CSS.escape === "function") return window.CSS.escape(value);
      return String(value).replace(/\\/g, "\\\\").replace(/'/g, "\\'");
    }

    function completeOrderValue(candidate) {
      if (!Array.isArray(candidate) || candidate.length === 0 || candidate.length !== order.length) return false;
      var normalized = candidate.map(function (key) { return text(key); });
      if (normalized.some(function (key, index) { return !key || normalized.indexOf(key) !== index; })) return false;
      return sameSet(order, normalized);
    }

    function expectedTargetValueFor(candidate) {
      if (!completeOrderValue(candidate)) return null;
      return { present: true, value: JSON.stringify(candidate.map(String)) };
    }

    function acknowledgedOrder(data, fallback) {
      data = (data && data.data) || (data && data.result) || data || {};
      if (data.data) data = data.data;
      if (data.value === undefined || data.value === null || data.value === "") return fallback.slice();
      var candidate = data.value;
      if (typeof candidate === "string") {
        try { candidate = JSON.parse(candidate); } catch (e) { return fallback.slice(); }
      }
      if (!Array.isArray(candidate)) return fallback.slice();
      candidate = candidate.map(String);
      return candidate.length === fallback.length && sameSet(fallback, candidate) ? candidate : fallback.slice();
    }

    function isConflictError(error) {
      var message = text(error && error.message ? error.message : error);
      return /\b(?:409|412)\b/.test(message) || /\b(?:conflict|stale)\b/i.test(message);
    }

    function validateSupport() {
      if (!configuredEnabled) {
        enabled = false;
        if (serverDisabledReason) setStatus(serverDisabledReason, "readonly");
        else if (!status || !text(status.textContent)) setStatus("Section ordering is read-only for this route.", "readonly");
        removeOverlay();
        syncMoveButtons();
        return false;
      }
      if (!list || !adapter) {
        setReadOnly("Section ordering is unavailable in this editor.");
        return false;
      }
      if (!frame) {
        setReadOnly("Section ordering is unavailable because this editor has no preview.");
        return false;
      }
      // A defer-loaded editor script can run before the iframe's initial
      // document exists. Keep controls disabled with an honest loading state,
      // but leave the load listener attached so the lane can become usable.
      if (!frameReady) {
        enabled = false;
        root.setAttribute("data-gosx-studio-section-order-enabled", "false");
        clearRuntimeDisabledReason();
        setStatus("Loading the same-origin preview before enabling section ordering.", "loading");
        removeOverlay();
        syncMoveButtons();
        return false;
      }
      var activePreviewRoute = currentPreviewRoute();
      if (activePreviewRoute !== sourceRoute || route !== sourceRoute) {
        setReadOnly("Section ordering applies only to the configured source route.");
        return false;
      }
      if (!frameIsSameOrigin()) {
        setReadOnly("Section ordering requires a same-origin preview.");
        return false;
      }
      var sections = frameSections();
      if (!sections.length) {
        setReadOnly("This preview has no server-issued section identity, so ordering is read-only.");
        return false;
      }
      var previewOrder = sections.map(sectionKey);
      if (previewOrder.some(function (key, index) { return !key || previewOrder.indexOf(key) !== index; })) {
        setReadOnly("This preview has duplicate or incomplete section identities, so ordering is read-only.");
        return false;
      }
      var unknownPreviewSection = previewOrder.some(function (key) { return order.indexOf(key) < 0; });
      var missingRequiredSection = order.some(function (key) {
        if (previewOrder.indexOf(key) >= 0) return false;
        var item = root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(key) + "']");
        return !item || item.getAttribute("data-gosx-studio-section-order-preview-optional") !== "true";
      });
      if (unknownPreviewSection || missingRequiredSection) {
        var reason = missingRequiredSection && !unknownPreviewSection
          ? "This preview omits one or more required sections, so ordering is read-only until all sections are rendered."
          : "Section order is read-only until preview identities match the host order.";
        setReadOnly(reason);
        return false;
      }
      setPreviewPresence(previewOrder);
      enabled = true;
      root.setAttribute("data-gosx-studio-section-order-enabled", "true");
      clearRuntimeDisabledReason();
      if (root.getAttribute("data-gosx-studio-section-order-state") !== "saved") {
        var optionalMissing = order.some(function (key) { return previewOrder.indexOf(key) < 0; });
        setStatus(optionalMissing ? "Some optional sections are hidden in this preview; rendered sections can be reordered." : "Drag sections or use Up and Down.", "ready");
      }
      syncMoveButtons();
      return true;
    }

    function ensureOperation() {
      if (operation) return operation;
      if (!form) form = createDetachedForm();
      if (!form) return null;
      if (!hasCSRFToken(form)) return null;
      // GoSXStudioOperationRuntime.commit supplies gosx_studio_operation_id for
      // every durable reorder request; the detached form only carries stable
      // host context such as CSRF and current target cursors.
      var runtime = window.GoSXStudioOperationRuntime;
      if (form.__gosxOperationRuntime) operation = form.__gosxOperationRuntime;
      else if (runtime && typeof runtime.create === "function") operation = runtime.create(form);
      if (operation && typeof operation.select === "function") {
        operation.select({
          route: route,
          pageId: form.getAttribute("data-studio-target-page-id") || "home",
          field: form.getAttribute("data-studio-target-field") || "home.sections.order",
          componentKey: form.getAttribute("data-studio-target-component") || ""
        });
      }
      return operation;
    }

    function copyCSRFToken(targetForm) {
      if (!targetForm) return false;
      var existing = targetForm.querySelector("[name='csrf_token'], [name='_csrf'], [name='gorilla.csrf.Token']");
      if (existing && text(existing.value || existing.getAttribute("value"))) return true;
      var sourceForm = closest(root, "form");
      var token = sourceForm && sourceForm.querySelector("[name='csrf_token'], [name='_csrf'], [name='gorilla.csrf.Token']");
      if (!token) token = doc.querySelector("[name='csrf_token'], [name='_csrf'], [name='gorilla.csrf.Token']");
      if (!token || !token.name || !text(token.value || token.getAttribute("value"))) return false;
      hiddenInput(targetForm, token.name, token.value || token.getAttribute("value") || "");
      return true;
    }

    function hasCSRFToken(targetForm) {
      if (!targetForm) return false;
      var token = targetForm.querySelector("[name='csrf_token'], [name='_csrf'], [name='gorilla.csrf.Token']");
      return !!(token && text(token.value || token.getAttribute("value")));
    }

    function uniqueOperationID() {
      operationSequence += 1;
      try {
        if (window.crypto && typeof window.crypto.randomUUID === "function") return window.crypto.randomUUID();
      } catch (e) {}
      return "section-order-" + String(Date.now()) + "-" + String(operationSequence);
    }

    function syncAdapterCursors() {
      if (!isConnected() || !adapter || !form) return;
      ["data-studio-document-revision", "data-studio-target-head"].forEach(function (attr) {
        var value = form.getAttribute(attr);
        if (value !== null) adapter.setAttribute(attr, value);
      });
    }

    function createDetachedForm() {
      if (!isConnected() || !adapter) return null;
      var action = text(adapter.getAttribute("data-gosx-studio-operation-action"));
      if (!action) return null;
      var detached = doc.createElement("form");
      detached.method = "post";
      detached.action = action;
      detached.hidden = true;
      detached.noValidate = true;
      detached.setAttribute("data-gosx-studio-durable-history", "true");
      detached.setAttribute("data-gosx-studio-section-order-detached-form", "true");
      detached.setAttribute("data-gosx-studio-operation-action", action);
      detached.setAttribute("data-studio-document-revision", adapter.getAttribute("data-studio-document-revision") || "0");
      detached.setAttribute("data-studio-target-head", adapter.getAttribute("data-studio-target-head") || "");
      detached.setAttribute("data-studio-target-route", route);
      detached.setAttribute("data-studio-target-page-id", adapter.getAttribute("data-studio-target-page-id") || "home");
      detached.setAttribute("data-studio-target-field", adapter.getAttribute("data-studio-target-field") || "home.sections.order");
      detached.setAttribute("data-studio-target-component", adapter.getAttribute("data-studio-target-component") || "");
      copyCSRFToken(detached);
      hiddenInput(detached, "gosx_studio_value", "", { "data-studio-operation-value": "true" });
      hiddenInput(detached, "gosx_studio_expected_revision", adapter.getAttribute("data-studio-document-revision") || "0");
      hiddenInput(detached, "gosx_studio_expected_target_head", adapter.getAttribute("data-studio-target-head") || "");
      hiddenInput(detached, "gosx_studio_operation_id", "", { "data-studio-operation-id": "true" });
      (doc.body || doc.documentElement).appendChild(detached);
      detached.addEventListener("gosxstudio:operation-committed", syncAdapterCursors);
      return detached;
    }

    function updateHistory(data, kind) {
      data = (data && data.data) || (data && data.result) || data || {};
      if (data.data) data = data.data;
      if (form) {
        if (data.documentRevision !== undefined && data.documentRevision !== null && data.documentRevision !== "") {
          form.setAttribute("data-studio-document-revision", String(data.documentRevision));
        }
        if (data.targetHead !== undefined && data.targetHead !== null) {
          form.setAttribute("data-studio-target-head", String(data.targetHead));
        }
      }
      var undo = root.querySelector("[data-gosx-studio-section-order-history='undo']");
      var redo = root.querySelector("[data-gosx-studio-section-order-history='redo']");
      if (kind === "undo") {
        if (undo) {
          undo.setAttribute("data-gosx-studio-history-operation-id", "");
          undo.setAttribute("disabled", "disabled");
        }
        if (redo && data.operationID) {
          redo.setAttribute("data-gosx-studio-history-operation-id", String(data.operationID));
          redo.removeAttribute("disabled");
        }
      } else if (kind === "redo") {
        if (redo) {
          redo.setAttribute("data-gosx-studio-history-operation-id", "");
          redo.setAttribute("disabled", "disabled");
        }
        if (undo && data.operationID) {
          undo.setAttribute("data-gosx-studio-history-operation-id", String(data.operationID));
          undo.removeAttribute("disabled");
        }
      } else if (data.operationID && undo) {
        undo.setAttribute("data-gosx-studio-history-operation-id", String(data.operationID));
        undo.removeAttribute("disabled");
        if (redo) {
          redo.setAttribute("data-gosx-studio-history-operation-id", "");
          redo.setAttribute("disabled", "disabled");
        }
      }
      syncAdapterCursors();
    }

    function syncOrderBindings(nextOrder) {
      if (!isConnected()) return;
      var value = JSON.stringify(nextOrder);
      var targetField = form && (form.getAttribute("data-studio-target-field") || "home.sections.order");
      queryAll(doc, "form").forEach(function (candidate) {
        var binding = candidate.querySelector("[name='gosx_studio_binding']");
        var candidateField = binding
          ? String(binding.value || binding.getAttribute("value") || "")
          : text(candidate.getAttribute("data-studio-target-field"));
        if (!candidateField || candidateField !== targetField) return;
        queryAll(candidate, "[data-studio-operation-value], [name='gosx_studio_value']").forEach(function (node) {
          if ("value" in node) node.value = value;
          node.setAttribute("value", value);
        });
      });
      queryAll(doc, "[data-gosx-studio-section-order-value]").forEach(function (node) {
        node.textContent = value;
      });
    }

    function refreshFrame() {
      if (!isConnected() || !frame) return;
      frameReady = false;
      enabled = false;
      root.setAttribute("data-gosx-studio-section-order-enabled", "false");
      syncMoveButtons();
      removeOverlay();
      try {
        var url = new URL(frame.getAttribute("src") || window.location.href, window.location.href);
        url.searchParams.set("gosx-studio-refresh", String(Date.now()));
        frame.setAttribute("src", url.href);
      } catch (e) {
        if (frame.contentWindow && frame.contentWindow.location) frame.contentWindow.location.reload();
      }
    }

    function commit(nextOrder, message) {
      if (!isConnected() || pending || !enabled || !sameSet(order, nextOrder) || order.join("\u0000") === nextOrder.join("\u0000")) {
        clearDrop();
        return Promise.resolve(false);
      }
      var previous = order.slice();
      var expectedOrder = committedOrder.slice();
      var expectedTargetValue = expectedTargetValueFor(expectedOrder);
      var committedKey = drag && drag.key || selectedKey;
      pending = true;
      reorderLane(nextOrder);
      setStatus("Saving section order...", "saving");
      syncMoveButtons();
      var op = null;
      try { op = ensureOperation(); } catch (error) {}
      if (!op || typeof op.commit !== "function") {
        pending = false;
        reorderLane(previous);
        setStatus(hasCSRFToken(form) ? "Section ordering is unavailable until the durable operation runtime loads." : "Section ordering is unavailable until CSRF protection is ready.", "error");
        syncMoveButtons();
        return Promise.resolve(false);
      }
      var operationID = uniqueOperationID();
      var operationIDNode = form && form.querySelector("[data-studio-operation-id]");
      if (operationIDNode) operationIDNode.value = operationID;
      var commitOptions = {
        id: operationID,
        route: route,
        pageId: form.getAttribute("data-studio-target-page-id") || "home",
        field: form.getAttribute("data-studio-target-field") || "home.sections.order",
        componentKey: form.getAttribute("data-studio-target-component") || ""
      };
      // This lane always submits a complete order set-field. Capture the
      // expected value before the optimistic move; it includes optional
      // hidden members retained in the durable lane order.
      if (expectedTargetValue) commitOptions.expectedTargetValue = expectedTargetValue;
      return Promise.resolve().then(function () {
        return op.commit("set-field", JSON.stringify(nextOrder), commitOptions);
      }).then(function (response) {
        if (!isConnected()) return false;
        return response.clone().json().catch(function () { return {}; }).then(function (body) {
          if (!isConnected()) return false;
          var acknowledged = acknowledgedOrder(body, nextOrder);
          if (order.join("\u0000") !== acknowledged.join("\u0000")) reorderLane(acknowledged);
          updateHistory(body, "set-field");
          syncOrderBindings(acknowledged);
          committedOrder = acknowledged.slice();
          selectedKey = committedKey;
          syncMoveButtons();
          setStatus(message || "Section order saved.", "saved");
          refreshFrame();
          root.dispatchEvent(new CustomEvent("gosxstudio:section-order-committed", { bubbles: true, detail: { order: acknowledged.slice(), result: body } }));
          return true;
        });
      }).catch(function (error) {
        if (!isConnected()) return false;
        var conflict = isConflictError(error);
        if (!conflict) reorderLane(previous);
        setStatus(conflict ? "Section order changed on the server; your reorder remains unsaved." : (error && error.message ? error.message : "Section order could not be saved."), "error");
        root.dispatchEvent(new CustomEvent("gosxstudio:section-order-error", { bubbles: true, detail: { error: error, conflict: conflict, order: order.slice(), committedOrder: committedOrder.slice() } }));
        return false;
      }).finally(function () {
        if (!isConnected()) return;
        pending = false;
        clearDrop();
        syncMoveButtons();
      });
    }

    function clearDrop() {
      if (disposed) return;
      queryAll(root, ITEM).forEach(function (item) {
        item.removeAttribute("data-gosx-studio-section-order-drop");
        item.removeAttribute("data-gosx-studio-section-order-dragging");
      });
      (overlay ? queryAll(overlay, PREVIEW_HANDLE) : []).forEach(function (handle) {
        handle.removeAttribute("data-gosx-studio-section-order-drop");
        handle.removeAttribute("data-gosx-studio-section-order-dragging");
      });
      drag = null;
    }

    function targetFromPoint(clientX, clientY) {
      if (!isConnected()) return null;
      var node = doc.elementFromPoint(clientX, clientY);
      var item = closest(node, ITEM);
      if (item && root.contains(item)) {
        var rect = item.getBoundingClientRect();
        return { key: text(item.getAttribute("data-gosx-studio-section-order-item")), index: order.indexOf(text(item.getAttribute("data-gosx-studio-section-order-item"))) + (clientY > rect.top + rect.height / 2 ? 1 : 0), item: item };
      }
      var handle = closest(node, PREVIEW_HANDLE);
      if (handle && (!overlay || !overlay.contains(handle))) handle = null;
      if (handle) {
        var key = text(handle.getAttribute("data-gosx-studio-section-key"));
        var rectHandle = handle.getBoundingClientRect();
        return { key: key, index: order.indexOf(key) + (clientY > rectHandle.top + rectHandle.height / 2 ? 1 : 0), item: root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(key) + "']"), handle: handle };
      }
      if (frame && frame.getBoundingClientRect) {
        var frameRect = frame.getBoundingClientRect();
        if (clientX >= frameRect.left && clientX <= frameRect.right && clientY >= frameRect.top && clientY <= frameRect.bottom && frameDoc) {
          var inner = frameDoc.elementFromPoint(clientX - frameRect.left, clientY - frameRect.top);
          var section = closest(inner, SECTION);
          var sectionKeyValue = sectionKey(section);
          if (sectionKeyValue) return { key: sectionKeyValue, index: order.indexOf(sectionKeyValue), item: root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(sectionKeyValue) + "']") };
        }
      }
      return null;
    }

    function markDrop(target) {
      if (!isConnected()) return;
      queryAll(root, ITEM).forEach(function (item) { item.removeAttribute("data-gosx-studio-section-order-drop"); });
      (overlay ? queryAll(overlay, PREVIEW_HANDLE) : []).forEach(function (handle) { handle.removeAttribute("data-gosx-studio-section-order-drop"); });
      if (!target || !drag || target.index < 0 || target.key === drag.key) return;
      if (target.item && (target.item.getAttribute("data-gosx-studio-section-order-locked") === "true" || (target.item.getAttribute("data-gosx-studio-section-order-preview-optional") === "true" && !previewPresent[target.key]))) return;
      var currentIndex = order.indexOf(target.key);
      if (currentIndex < 0) return;
      var pos = target.index <= currentIndex ? "before" : "after";
      if (target.item) target.item.setAttribute("data-gosx-studio-section-order-drop", pos);
      if (target.handle) target.handle.setAttribute("data-gosx-studio-section-order-drop", pos);
      drag.targetIndex = target.index;
    }

    function startDrag(key, pointerId, origin) {
      if (!isConnected() || pending || !enabled || !key) return false;
      selectedKey = key;
      drag = { key: key, pointerId: pointerId, targetIndex: -1, origin: origin || "lane" };
      var item = root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(key) + "']");
      if (!item || item.getAttribute("data-gosx-studio-section-order-locked") === "true" || (item.getAttribute("data-gosx-studio-section-order-preview-optional") === "true" && !previewPresent[key])) {
        drag = null;
        return false;
      }
      if (item) item.setAttribute("data-gosx-studio-section-order-dragging", "true");
      setStatus("Choose where to drop " + key + ".", "dragging");
      syncMoveButtons();
      return true;
    }

    function handlePointerDown(event) {
      if (!isConnected()) return;
      var handle = closest(event.target, HANDLE + ", " + PREVIEW_HANDLE);
      if (!handle || !ownsNode(handle) || handle.disabled) return;
      var key = text(handle.getAttribute("data-gosx-studio-section-order-key") || handle.getAttribute("data-gosx-studio-section-key") || closest(handle, ITEM).getAttribute("data-gosx-studio-section-order-item"));
      if (!startDrag(key, event.pointerId, closest(handle, PREVIEW_HANDLE) ? "preview" : "lane")) return;
      event.preventDefault();
      try { handle.setPointerCapture(event.pointerId); } catch (e) {}
    }

    function handlePointerMove(event) {
      if (!isConnected() || !drag || drag.pointerId !== event.pointerId) return;
      markDrop(targetFromPoint(event.clientX, event.clientY));
      var edge = 48;
      if (event.clientY < edge) window.scrollBy(0, -16);
      else if (event.clientY > window.innerHeight - edge) window.scrollBy(0, 16);
    }

    function finishPointer(event) {
      if (!isConnected() || !drag || drag.pointerId !== event.pointerId) return;
      event.preventDefault();
      var next = drag.targetIndex >= 0 ? moveKey(order, drag.key, drag.targetIndex) : null;
      if (!next) {
        setStatus("Section move canceled.", "ready");
        clearDrop();
        return;
      }
      commit(next, "Section order saved.");
    }

    function cancelDrag() {
      if (!isConnected() || !drag) return;
      setStatus("Section move canceled.", "ready");
      clearDrop();
    }

    function clickMove(event) {
      if (!isConnected()) return;
      var button = closest(event.target, MOVE);
      if (!button || !ownsNode(button) || button.disabled) return;
      var item = closest(button, ITEM);
      var key = text(item && item.getAttribute("data-gosx-studio-section-order-item"));
      var index = order.indexOf(key);
      var direction = button.getAttribute("data-gosx-studio-section-order-move");
      var target = direction === "up" ? index - 1 : index + 2;
      event.preventDefault();
      commit(moveKey(order, key, target), direction === "up" ? "Section moved up." : "Section moved down.");
    }

    function clickHistory(event) {
      if (!isConnected()) return;
      var button = closest(event.target, HISTORY);
      if (!button || !ownsNode(button) || button.disabled || pending || !enabled) return;
      var op = null;
      try { op = ensureOperation(); } catch (error) {}
      if (!op) {
        setStatus(hasCSRFToken(form) ? "Section ordering is unavailable until the durable operation runtime loads." : "Section ordering is unavailable until CSRF protection is ready.", "error");
        return;
      }
      event.preventDefault();
      var historyKind = button.getAttribute("data-gosx-studio-section-order-history") === "undo" ? "undo" : "redo";
      pending = true;
      setStatus(historyKind === "undo" ? "Undoing reorder..." : "Redoing reorder...", "saving");
      syncMoveButtons();
      var id = button.getAttribute("data-gosx-studio-history-operation-id") || "";
      var call;
      try { call = historyKind === "undo" ? op.undo(id) : op.redo(id); } catch (error) { call = Promise.reject(error); }
      Promise.resolve(call).then(function (response) {
        if (!isConnected()) return;
        return response.clone().json().catch(function () { return {}; }).then(function (body) {
          if (!isConnected()) return;
          var data = (body && body.data) || (body && body.result) || body || {};
          if (data.data) data = data.data;
          if (data.value) {
            try {
              var nextOrder = JSON.parse(data.value);
              if (Array.isArray(nextOrder) && sameSet(order, nextOrder.map(String))) {
                nextOrder = nextOrder.map(String);
                reorderLane(nextOrder);
                syncOrderBindings(nextOrder);
                committedOrder = nextOrder.slice();
              }
            } catch (e) {}
          }
          updateHistory(body, historyKind);
          setStatus("Section reorder history applied.", "saved");
          refreshFrame();
        });
      }).catch(function (error) {
        if (!isConnected()) return;
        setStatus(error && error.message ? error.message : "Section reorder history failed.", "error");
      }).finally(function () {
        if (!isConnected()) return;
        pending = false;
        syncMoveButtons();
      });
    }

    function editableShortcutTarget(target) {
      if (!target) return false;
      if (target.isContentEditable) return true;
      if (closest(target, "[contenteditable='true'], [contenteditable='']")) return true;
      var tagName = String(target.tagName || "").toLowerCase();
      return tagName === "input" || tagName === "textarea" || tagName === "select" || tagName === "option";
    }

    function stopHistoryShortcut(event) {
      event.preventDefault();
      event.stopPropagation();
      if (typeof event.stopImmediatePropagation === "function") event.stopImmediatePropagation();
    }

    function historyShortcut(event) {
      if (!isConnected() || !event || event.isComposing || event.altKey || !(event.ctrlKey || event.metaKey)) return;
      if (editableShortcutTarget(event.target)) return;
      var inLane = root.contains(event.target) || (overlay && overlay.contains(event.target));
      if (!inLane) return;
      var control = closest(event.target, HANDLE + ", " + PREVIEW_HANDLE + ", " + MOVE + ", " + HISTORY);
      if (!control || (!root.contains(control) && !(overlay && overlay.contains(control)))) return;
      var key = String(event.key || "").toLowerCase();
      var historyKind = "";
      if (key === "z" && !event.shiftKey) historyKind = "undo";
      else if (key === "z" && event.shiftKey) historyKind = "redo";
      else if (key === "y" && !event.shiftKey) historyKind = "redo";
      if (!historyKind) return;
      var button = root.querySelector("[data-gosx-studio-section-order-history='" + historyKind + "']");
      // A shortcut focused on the section lane belongs to this lane even when
      // history is unavailable or a request is in flight. Prevent the global
      // editor-history listener from treating it as an ordinary form edit.
      stopHistoryShortcut(event);
      if (!button || button.disabled || pending || !enabled) return;
      clickHistory({ target: button, preventDefault: function () {} });
    }

    function keydown(event) {
      if (!isConnected()) return;
      var handle = closest(event.target, HANDLE);
      if (!handle || handle.disabled) return;
      var item = closest(handle, ITEM);
      var key = text(item && item.getAttribute("data-gosx-studio-section-order-item"));
      if (event.key === " " || event.key === "Spacebar") {
        event.preventDefault();
        if (drag && drag.key === key) cancelDrag();
        else startDrag(key, 0, "keyboard");
      } else if (event.key === "ArrowUp" && drag && drag.key === key) {
        event.preventDefault();
        drag.targetIndex = Math.max(0, order.indexOf(key) - 1);
        markDrop({ key: order[drag.targetIndex], index: drag.targetIndex, item: root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(order[drag.targetIndex]) + "']") });
      } else if (event.key === "ArrowDown" && drag && drag.key === key) {
        event.preventDefault();
        drag.targetIndex = Math.min(order.length, order.indexOf(key) + 2);
        var markKey = order[Math.min(order.length - 1, drag.targetIndex - 1)];
        markDrop({ key: markKey, index: drag.targetIndex, item: root.querySelector("[data-gosx-studio-section-order-item='" + cssEscape(markKey) + "']") });
      } else if (event.key === "Enter" && drag && drag.key === key) {
        event.preventDefault();
        if (drag.targetIndex >= 0) commit(moveKey(order, key, drag.targetIndex), "Section order saved.");
        else cancelDrag();
      } else if (event.key === "Escape") {
        event.preventDefault();
        cancelDrag();
      }
    }

    function removeOverlay() {
      if (overlay && overlay.parentNode) overlay.parentNode.removeChild(overlay);
      overlay = null;
    }

    function positionOverlay() {
      if (!isConnected() || !frame || !enabled) {
        removeOverlay();
        return;
      }
      frameDoc = readFrameDocument();
      if (!frameDoc) {
        removeOverlay();
        return;
      }
      if (!overlay) {
        overlay = doc.createElement("div");
        overlay.setAttribute("data-gosx-studio-section-order-overlay", "true");
        overlay.style.position = "absolute";
        overlay.style.inset = "0";
        overlay.style.pointerEvents = "none";
        overlay.style.zIndex = "6";
        var stage = closest(frame, "[data-studio-page-canvas-stage='true']") || frame.parentElement;
        if (stage) {
          if (window.getComputedStyle(stage).position === "static") stage.style.position = "relative";
          stage.appendChild(overlay);
        }
      }
      overlay.textContent = "";
      var frameRect = frame.getBoundingClientRect();
      var stageRect = overlay.getBoundingClientRect();
      frameSections().forEach(function (section) {
        var key = sectionKey(section);
        if (order.indexOf(key) < 0 || section.getAttribute("data-gosx-studio-section-locked") === "true") return;
        var rect = section.getBoundingClientRect();
        var button = doc.createElement("button");
        button.type = "button";
        button.textContent = "Drag";
        button.setAttribute("data-gosx-studio-section-order-preview-handle", "true");
        button.setAttribute("data-gosx-studio-section-key", key);
        button.setAttribute("aria-label", "Drag " + labelForSection(section, key));
        button.setAttribute("aria-describedby", "studio-section-order-guidance");
        button.style.position = "absolute";
        button.style.left = String(Math.max(0, frameRect.left - stageRect.left + rect.left + 8)) + "px";
        button.style.top = String(Math.max(0, frameRect.top - stageRect.top + rect.top + 8)) + "px";
        button.style.minWidth = "44px";
        button.style.minHeight = "32px";
        button.style.pointerEvents = "auto";
        overlay.appendChild(button);
      });
    }

    function handleFrameLoad() {
      if (!isConnected()) return;
      frameReady = true;
      frameDoc = readFrameDocument();
      validateSupport();
      positionOverlay();
    }

    function dispose() {
      if (disposed) return;
      disposed = true;
      pending = false;
      drag = null;
      removeListeners();
      removeOverlay();
      if (form && form.parentNode) form.parentNode.removeChild(form);
      form = null;
      operation = null;
      frameDoc = null;
      enabled = false;
    }

    function bind() {
      if (disposed) return;
      if (!listenersBound) {
        listenersBound = true;
        // Register the frame listener before the first validation. The initial
        // defer script commonly runs while the iframe is still about:blank;
        // returning early here used to strand the lane in permanent readonly
        // mode.
        if (frame) listen(frame, "load", handleFrameLoad);
        listen(doc, "pointerdown", handlePointerDown);
        listen(root, "click", clickMove);
        listen(root, "click", clickHistory);
        listen(root, "keydown", keydown);
        listen(doc, "keydown", historyShortcut, true);
        listen(doc, "pointermove", handlePointerMove);
        listen(doc, "pointerup", finishPointer);
        listen(doc, "pointercancel", cancelDrag);
        listen(doc, "keydown", handleDocumentEscape);
        listen(window, "resize", positionOverlay);
        listen(window, "scroll", positionOverlay, true);
        var existingFrameDoc = readFrameDocument();
        if (existingFrameDoc && existingFrameDoc.readyState === "complete") frameReady = true;
      }
      if (!isConnected()) return;
      validateSupport();
      syncMoveButtons();
      positionOverlay();
    }

    function handleDocumentEscape(event) {
      if (event.key === "Escape") cancelDrag();
    }

    return {
      bind: bind,
      dispose: dispose,
      isDisposed: function () { return disposed; },
      order: function () { return order.slice(); },
      refresh: function () {
        if (!isConnected()) return;
        validateSupport();
        positionOverlay();
      }
    };
  }

  var mountedRoots = [];

  function rootIsLive(root) {
    if (!root) return false;
    if (root.isConnected !== undefined && !root.isConnected) return false;
    if (root.matches && !root.matches(ROOT)) return false;
    return !!(doc.documentElement && doc.documentElement.contains(root));
  }

  function pruneMountedRoots() {
    mountedRoots = mountedRoots.filter(function (entry) {
      if (rootIsLive(entry.root) && !entry.runtime.isDisposed()) return true;
      entry.runtime.dispose();
      return false;
    });
  }

  function rootsInScope(scope) {
    scope = scope || doc;
    var nodes = [];
    if (scope.matches && scope.matches(ROOT)) nodes.push(scope);
    queryAll(scope, ROOT).forEach(function (node) {
      if (nodes.indexOf(node) < 0) nodes.push(node);
    });
    return nodes;
  }

  function mount(scope) {
    pruneMountedRoots();
    rootsInScope(scope).forEach(function (node) {
      var runtime = node.__gosxSectionOrderRuntime;
      if (runtime && !runtime.isDisposed()) {
        runtime.refresh();
        return;
      }
      if (runtime) runtime.dispose();
      runtime = createRuntime(node);
      node.__gosxSectionOrderRuntime = runtime;
      mountedRoots.push({ root: node, runtime: runtime });
      runtime.bind();
    });
  }

  var api = { mount: mount };
  window[GLOBAL_STATE] = { api: api, mount: mount };
  window.GoSXStudioSectionOrderRuntime = api;
  function handleNavigation() { mount(doc); }
  if (doc.readyState === "loading") doc.addEventListener("DOMContentLoaded", handleNavigation, { once: true });
  else mount(doc);
  doc.addEventListener("gosx:navigate", handleNavigation);
  doc.addEventListener("gosx:render", handleNavigation);
})();
