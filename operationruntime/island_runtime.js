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
  function runtime(form) {
    var selected = { route: form.getAttribute("data-studio-target-route") || "/", pageId: form.getAttribute("data-studio-target-page-id") || "", field: form.getAttribute("data-studio-target-field") || "", componentKey: form.getAttribute("data-studio-target-component") || "" };
    function setSelection(detail) { selected = detail || null; form.dispatchEvent(new CustomEvent("gosxstudio:operation-selection", { bubbles: true, detail: selected })); }
    function request(kind, value, extra) {
      extra = extra || {};
      var target = selected || {};
      var payload = { gosx_studio_operation: kind, gosx_studio_operation_id: extra.id || (crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + Math.random()), gosx_studio_page_route: extra.route || target.route || "/", gosx_studio_page_key: extra.pageId || target.pageId || target.page || "", gosx_studio_component_key: extra.componentKey || target.componentKey || target.blockKey || "", gosx_studio_binding: extra.field || target.field || "", gosx_studio_value: value || "", gosx_studio_style_property: extra.property || target.property || "", gosx_studio_style_value: value || "", gosx_studio_breakpoint: extra.breakpoint || target.breakpoint || "base", gosx_studio_state: extra.state || target.state || "default", gosx_studio_expected_revision: extra.revision || form.getAttribute("data-studio-document-revision") || "0", gosx_studio_expected_target_head: extra.head || target.targetHead || "", gosx_studio_history_operation_id: extra.historyOperationId || "" };
      return post(form, payload).then(function (response) {
        if (!response.ok) throw new Error("Operation failed (" + response.status + ")");
        return response.clone().json().catch(function () { return {}; }).then(function (body) {
          updateCursors(form, body);
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
  }
  window.GoSXStudioOperationRuntime = { bind: bind, create: runtime };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { bind(document); }, { once: true }); else bind(document);
})();
