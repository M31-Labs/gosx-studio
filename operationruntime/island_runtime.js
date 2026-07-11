(function () {
  "use strict";
  function qs(root, selector) { return (root || document).querySelector(selector); }
  function post(form, payload) {
    var body = new FormData(form);
    Object.keys(payload || {}).forEach(function (key) { body.set(key, payload[key] == null ? "" : String(payload[key])); });
    var action = form.getAttribute("data-gosx-studio-operation-action") || form.action || window.location.href;
    return fetch(action, { method: "POST", body: body, credentials: "same-origin", headers: { "X-Requested-With": "XMLHttpRequest" } });
  }
  function resultData(payload) {
    var data = payload || {};
    if (data.data && typeof data.data === "object") data = data.data;
    if (data.result && typeof data.result === "object") data = data.result;
    if (data.data && typeof data.data === "object") data = data.data;
    return data || {};
  }
  function updateCursors(form, payload) {
    var data = resultData(payload);
    var revision = data.documentRevision;
    var head = data.targetHead;
    if (revision !== undefined && revision !== null && revision !== "") {
      revision = String(revision);
      form.setAttribute("data-studio-document-revision", revision);
      form.querySelectorAll("[name='gosx_studio_expected_revision']").forEach(function (node) { node.value = revision; });
    }
    if (head !== undefined && head !== null) {
      head = String(head);
      form.setAttribute("data-studio-target-head", head);
      form.querySelectorAll("[name='gosx_studio_expected_target_head']").forEach(function (node) { node.value = head; });
    }
  }
  function updateHistoryButtons(form, kind, payload) {
    var data = resultData(payload);
    var operationID = data.operationID || "";
    var undo = form.querySelector("[data-gosx-studio-operation-kind='undo']");
    var redo = form.querySelector("[data-gosx-studio-operation-kind='redo']");
    if (kind === "undo" && undo && redo) {
      undo.setAttribute("disabled", "disabled");
      redo.removeAttribute("disabled");
      redo.setAttribute("data-gosx-studio-history-operation-id", operationID);
    } else if (kind === "redo" && undo && redo) {
      redo.setAttribute("disabled", "disabled");
      undo.removeAttribute("disabled");
      undo.setAttribute("data-gosx-studio-history-operation-id", operationID);
    }
  }
  function runtime(form) {
    var selected = { route: form.getAttribute("data-studio-target-route") || "/", pageId: form.getAttribute("data-studio-target-page-id") || "", field: form.getAttribute("data-studio-target-field") || "", componentKey: form.getAttribute("data-studio-target-component") || "" };
    var targetHeads = {};
    function targetKey(target, extra) {
      extra = extra || {};
      return [target.route || "/", target.pageId || "", target.field || "", extra.componentKey || target.componentKey || "", extra.property || target.property || "", extra.breakpoint || target.breakpoint || "base", extra.state || target.state || "default"].join("|");
    }
    function setSelection(detail) { selected = detail || null; form.dispatchEvent(new CustomEvent("gosxstudio:operation-selection", { bubbles: true, detail: selected })); }
    function request(kind, value, extra) {
      extra = extra || {};
      var target = selected || {};
      var effectiveKey = targetKey(target, extra);
      var expectedHead = extra.head;
      if (expectedHead === undefined) expectedHead = targetHeads[effectiveKey] || "";
      // The form cursor belongs to its selected content target. Style/reset
      // scopes have independent target heads and must start empty (or use the
      // per-target map), otherwise a prior headline save falsely conflicts.
      if (expectedHead === "" && kind === "set-field" && effectiveKey === targetKey(target, {})) expectedHead = form.getAttribute("data-studio-target-head") || "";
      var payload = { gosx_studio_operation: kind, gosx_studio_operation_id: extra.id || (crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + Math.random()), gosx_studio_page_route: extra.route || target.route || "/", gosx_studio_page_key: extra.pageId || target.pageId || target.page || "", gosx_studio_component_key: extra.componentKey || target.componentKey || target.blockKey || "", gosx_studio_binding: extra.field || target.field || "", gosx_studio_value: value || "", gosx_studio_style_property: extra.property || target.property || "", gosx_studio_style_value: value || "", gosx_studio_breakpoint: extra.breakpoint || target.breakpoint || "base", gosx_studio_state: extra.state || target.state || "default", gosx_studio_expected_revision: extra.revision || form.getAttribute("data-studio-document-revision") || "0", gosx_studio_expected_target_head: expectedHead, gosx_studio_history_operation_id: extra.historyOperationId || "" };
      return post(form, payload).then(function (response) {
        if (!response.ok) throw new Error("Operation failed (" + response.status + ")");
        return response.clone().json().catch(function () { return {}; }).then(function (body) {
          updateCursors(form, body);
          var bodyData = resultData(body);
          if (bodyData.targetHead !== undefined) targetHeads[effectiveKey] = String(bodyData.targetHead || "");
          updateHistoryButtons(form, kind, body);
          form.dispatchEvent(new CustomEvent("gosxstudio:operation-committed", { bubbles: true, detail: { kind: kind, response: response, body: body } }));
          return response;
        });
      }).catch(function (error) { form.dispatchEvent(new CustomEvent("gosxstudio:operation-error", { bubbles: true, detail: { error: error } })); throw error; });
    }
    return { select: setSelection, commit: request, cancel: function () { form.dispatchEvent(new CustomEvent("gosxstudio:operation-cancel", { bubbles: true })); }, undo: function (id) { return request("undo", "", { historyOperationId: id }); }, redo: function (id) { return request("redo", "", { historyOperationId: id }); } };
  }
  function bind(root) {
    (root || document).querySelectorAll("form[data-gosx-studio-durable-history='true']").forEach(function (form) {
      if (form.__gosxOperationRuntime) return;
      form.__gosxOperationRuntime = runtime(form);
      form.addEventListener("gosxstudio:preview-selection", function (event) { form.__gosxOperationRuntime.select(event.detail); });
      form.addEventListener("click", function (event) {
        var button = event.target.closest && event.target.closest("[data-gosx-studio-operation-kind]");
        if (!button) return;
        event.preventDefault();
        var kind = button.getAttribute("data-gosx-studio-operation-kind");
        var value = button.getAttribute("data-gosx-studio-operation-value") || "";
        if (!value && kind === "set-field") {
          var editorValue = form.querySelector("[data-studio-operation-value]");
          if (editorValue) value = "value" in editorValue ? editorValue.value : (editorValue.textContent || "");
        }
        form.__gosxOperationRuntime.select({ route: button.getAttribute("data-gosx-studio-route") || form.getAttribute("data-studio-target-route") || "/", pageId: button.getAttribute("data-gosx-studio-page-id") || form.getAttribute("data-studio-target-page-id") || "", field: button.getAttribute("data-gosx-studio-field") || form.getAttribute("data-studio-target-field") || "", componentKey: button.getAttribute("data-gosx-studio-component") || form.getAttribute("data-studio-target-component") || "", property: button.getAttribute("data-gosx-studio-style-property") || "" });
        form.__gosxOperationRuntime.commit(kind, value, { componentKey: button.getAttribute("data-gosx-studio-component"), property: button.getAttribute("data-gosx-studio-style-property"), breakpoint: button.getAttribute("data-gosx-studio-breakpoint"), state: button.getAttribute("data-gosx-studio-state"), historyOperationId: button.getAttribute("data-gosx-studio-history-operation-id") });
      });
    });
    if (!document.__gosxStudioOperationSelectionBridge) {
      document.__gosxStudioOperationSelectionBridge = true;
      document.addEventListener("gosxstudio:preview-select", function (event) { bridgeSelection(event.detail || {}); });
      document.addEventListener("gosxstudio:editor-operation", function (event) {
        var detail = event.detail || {};
        if (detail.mutation === false && detail.target) bridgeSelection(detail.target);
      });
    }
  }
  function bridgeSelection(detail) {
    detail = detail || {};
    var field = detail.field || "";
    var routeNode = document.querySelector("[data-studio-preview-route-current]");
    var route = detail.route || detail.targetRoute || (routeNode && routeNode.getAttribute("data-studio-preview-route-current")) || "/";
    var pageId = detail.pageID || detail.pageId || (route === "/about" ? "page:about" : "page:home");
    var component = detail.componentKey || detail.blockKey || "";
    if (component && component.indexOf(":") < 0) component = (route === "/about" ? "about:" : "home:") + component;
    if (!component && field === "pages.about.title") component = "about:content";
    var forms = document.querySelectorAll("form[data-gosx-studio-durable-history='true']");
    Array.prototype.forEach.call(forms, function (form) {
      var formRoute = form.getAttribute("data-studio-target-route") || "/";
      var formField = form.getAttribute("data-studio-target-field") || "";
      if (field && formField !== field) return;
      if (route && formRoute !== route) return;
      form.setAttribute("data-studio-target-route", route);
      form.setAttribute("data-studio-target-page-id", pageId);
      form.setAttribute("data-studio-target-component", component);
      ["gosx_studio_page_route", "gosx_studio_page_key", "gosx_studio_component_key", "gosx_studio_binding"].forEach(function (name, index) {
        var value = [route, pageId, component, field][index];
        form.querySelectorAll("[name='" + name + "']").forEach(function (node) { node.value = value; });
      });
      if (detail.value !== undefined || detail.text !== undefined) {
        var valueNode = form.querySelector("[data-studio-operation-value]");
        if (valueNode) valueNode.value = String(detail.value !== undefined ? detail.value : detail.text);
      }
      if (form.__gosxOperationRuntime) form.__gosxOperationRuntime.select({ route: route, pageId: pageId, field: field, componentKey: component });
    });
  }
  window.GoSXStudioOperationRuntime = { bind: bind, create: runtime };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { bind(document); }, { once: true }); else bind(document);
})();
