(function () {
  "use strict";
  function qs(root, selector) { return (root || document).querySelector(selector); }
  function post(form, payload) {
    var body = new FormData(form);
    Object.keys(payload || {}).forEach(function (key) { body.set(key, payload[key] == null ? "" : String(payload[key])); });
    var action = form.getAttribute("data-gosx-studio-operation-action") || form.action || window.location.href;
    return fetch(action, { method: "POST", body: body, credentials: "same-origin", headers: { "X-Requested-With": "XMLHttpRequest" } });
  }
  // Field values for the instance/interaction/flow durable operation
  // families (see authoring/operations.go's Field* constants -- these
  // literal strings must match exactly). A kind absent from this map keeps
  // using gosx_studio_binding as its field, matching set-field/undo/redo's
  // existing behavior.
  var DURABLE_FIELD_BY_KIND = {
    "set-shared-field": "instances.shared-field",
    "override-instance": "instances.override",
    "detach-instance": "instances.attachment",
    "restore-instance": "instances.attachment",
    "set-interaction": "interactions.entry",
    "remove-interaction": "interactions.entry",
    "set-flow-field": "flows.field",
    "remove-flow-field": "flows.field",
    "set-flow-action": "flows.action"
  };
  // Flow operations address their flow key/action key through the same
  // page/component payload fields SetField-family operations already use
  // (gosx_studio_page_key/gosx_studio_component_key), following
  // flow_handlers.go's own AuthoringChange precedent (PageKey carries
  // FlowKey). set-shared-field has no page target -- its "component" IS the
  // shared definition key.
  function durablePageId(kind, payload) {
    if (kind === "set-flow-field" || kind === "remove-flow-field" || kind === "set-flow-action") {
      return payload.gosx_studio_flow_key || payload.gosx_studio_page_key || "";
    }
    return payload.gosx_studio_page_key || "";
  }
  function durableComponentKey(kind, payload) {
    if (kind === "set-shared-field") return payload.gosx_studio_component_template_key || payload.gosx_studio_component_key || "";
    if (kind === "set-flow-field" || kind === "remove-flow-field" || kind === "set-flow-action") {
      return payload.gosx_studio_flow_action_key || payload.gosx_studio_component_key || "";
    }
    return payload.gosx_studio_component_key || "";
  }
  function durableControlKey(kind, payload) {
    if (kind === "set-interaction" || kind === "remove-interaction") return payload.gosx_studio_interaction_key || "";
    if (kind === "set-flow-field" || kind === "remove-flow-field") return payload.gosx_studio_flow_field_name || "";
    return payload.gosx_studio_control_key || "";
  }
  // durableValue builds the OperationRequest.Value the ledger expects for
  // kind: a plain string for content/style/instance-field kinds, or a JSON
  // settings payload for the interaction/flow-field/flow-action kinds (see
  // authoring.InteractionSettings/FlowFieldSettings/FlowActionSettings --
  // these JSON keys must match those Go structs' json tags exactly).
  function durableValue(kind, payload) {
    if (kind === "set-style") return payload.gosx_studio_style_value;
    if (kind === "set-interaction") {
      return JSON.stringify({
        kind: payload.gosx_studio_interaction_kind || "",
        effect: payload.gosx_studio_interaction_effect || "",
        durationMs: Number(payload.gosx_studio_interaction_duration_ms) || 0,
        delayMs: Number(payload.gosx_studio_interaction_delay_ms) || 0,
        once: payload.gosx_studio_interaction_once === "true" || payload.gosx_studio_interaction_once === true
      });
    }
    if (kind === "set-flow-field") {
      return JSON.stringify({
        label: payload.gosx_studio_flow_field_label || "",
        kind: payload.gosx_studio_control_kind || "",
        required: payload.gosx_studio_flow_field_required === "true" || payload.gosx_studio_flow_field_required === true
      });
    }
    if (kind === "set-flow-action") {
      return JSON.stringify({ label: payload.gosx_studio_flow_action_label || "", handlerRef: payload.gosx_studio_binding || "" });
    }
    return payload.gosx_studio_value;
  }
  function collaborationRequest(payload, target, extra) {
    extra = extra || {};
    var revision = Number(payload.gosx_studio_expected_revision || "0");
    if (!Number.isFinite(revision) || revision < 0) revision = 0;
    var kind = payload.gosx_studio_operation;
    var durableField = DURABLE_FIELD_BY_KIND[kind];
    var request = {
      schemaVersion: 1,
      id: payload.gosx_studio_operation_id,
      kind: kind,
      target: {
        route: payload.gosx_studio_page_route || "/",
        pageId: durablePageId(kind, payload),
        field: durableField !== undefined ? durableField : (payload.gosx_studio_binding || ""),
        componentKey: durableComponentKey(kind, payload),
        controlKey: durableControlKey(kind, payload),
        nodeId: extra.nodeId || target.nodeId || "",
        property: payload.gosx_studio_style_property || "",
        breakpoint: payload.gosx_studio_breakpoint || "base",
        state: payload.gosx_studio_state || "default"
      },
      value: durableValue(kind, payload),
      expectedDocumentRevision: revision,
      expectedTargetHead: payload.gosx_studio_expected_target_head || "",
      historyOperationId: payload.gosx_studio_history_operation_id || ""
    };
    if (payload.gosx_studio_expected_target_value !== undefined) {
      request.expectedTargetValue = JSON.parse(payload.gosx_studio_expected_target_value);
    }
    return request;
  }
  function collaborationResponse(ack) {
    var record = (ack && ack.record) || {}, after = record.after || {};
    var body = { data: { documentRevision: record.documentRevision || ack.sequence || 0, targetHead: record.targetHead || "", operationID: record.id || "", value: after.present ? (after.value || "") : "" } };
    return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
  }
  function submit(form, payload, target, extra) {
    var collaboration = window.GoSXStudioCollaborationRuntime;
    if (collaboration && typeof collaboration.available === "function" && collaboration.available() && typeof collaboration.submit === "function") {
      var request = collaborationRequest(payload, target, extra);
      if (typeof collaboration.expectedHead === "function") request.expectedTargetHead = collaboration.expectedHead(request);
      return collaboration.submit(request).then(collaborationResponse);
    }
    return post(form, payload);
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
    var layout = form.hasAttribute("data-studio-layout-control");
    if ((kind === "set-field" || (layout && (kind === "set-style" || kind === "reset-style"))) && undo && redo) {
      undo.removeAttribute("disabled");
      undo.setAttribute("data-gosx-studio-history-operation-id", operationID);
      redo.setAttribute("disabled", "disabled");
      redo.setAttribute("data-gosx-studio-history-operation-id", "");
    } else if (kind === "undo" && undo && redo) {
      undo.setAttribute("disabled", "disabled");
      undo.setAttribute("data-gosx-studio-history-operation-id", "");
      redo.removeAttribute("disabled");
      redo.setAttribute("data-gosx-studio-history-operation-id", operationID);
    } else if (kind === "redo" && undo && redo) {
      redo.setAttribute("disabled", "disabled");
      redo.setAttribute("data-gosx-studio-history-operation-id", "");
      undo.removeAttribute("disabled");
      undo.setAttribute("data-gosx-studio-history-operation-id", operationID);
    }
    // These controls represent content history. Style targets have their own
    // target heads and must never replace the selected content value or claim
    // its Undo button.
    if ((kind === "set-field" || kind === "undo" || kind === "redo") && data.value !== undefined && data.value !== null) {
      var valueNode = form.querySelector("[data-studio-operation-value]");
      if (valueNode) valueNode.value = String(data.value);
    }
    if (layout) {
      var select = form.querySelector("[data-studio-layout-value]");
      var presence = form.querySelector("[data-studio-layout-presence]");
      var reset = form.querySelector("[data-gosx-studio-operation-kind='reset-style']");
      if (select && data.value !== undefined && data.value !== null && String(data.value) !== "") select.value = String(data.value);
      if (presence && (kind === "set-style" || kind === "reset-style")) {
        var explicit = kind === "set-style";
        presence.setAttribute("data-studio-layout-presence", explicit ? "true" : "false");
        presence.textContent = explicit ? "Explicit override" : "Inherited after reload";
        if (reset) explicit ? reset.removeAttribute("disabled") : reset.setAttribute("disabled", "disabled");
      }
    }
  }
  function runtime(form) {
    var selected = { route: form.getAttribute("data-studio-target-route") || "/", pageId: form.getAttribute("data-studio-target-page-id") || "", field: form.getAttribute("data-studio-target-field") || "", componentKey: form.getAttribute("data-studio-target-component") || "" };
    var targetHeads = {};
    function targetKey(target, extra) {
      extra = extra || {};
      // controlKey addresses one named sub-key WITHIN a componentKey/pageId
      // container (an interaction's key, a shared/override instance's
      // control key, a flow field's name). Omitting it here (handoff-31
      // finding) let every interaction attached to the SAME component
      // collide onto one local-cursor cache entry -- see the matching fix
      // in collabruntime/island_runtime.js's own targetKey(), which is what
      // actually gates a live-collaboration submit's expectedTargetHead;
      // this local cache only matters for the plain-POST fallback (no
      // collaboration) and this form's own updateHistoryButtons/undo-redo
      // association, but must stay consistent with the same target-identity
      // discipline.
      return [target.route || "/", target.pageId || "", target.field || "", extra.componentKey || target.componentKey || "", extra.controlKey || target.controlKey || "", extra.property || target.property || "", extra.breakpoint || target.breakpoint || "base", extra.state || target.state || "default"].join("|");
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
      // set-interaction/remove-interaction share this same form-level seed
      // (handoff-31): an already-attached interaction's Save/Remove after a
      // fresh page load must start from the host-rendered current head, not
      // a blank in-memory cursor, or it would falsely conflict against its
      // own real state the first time either button is clicked post-load.
      if (expectedHead === "" && (kind === "set-field" || kind === "set-interaction" || kind === "remove-interaction") && effectiveKey === targetKey(target, {})) expectedHead = form.getAttribute("data-studio-target-head") || "";
      if (expectedHead === "" && form.hasAttribute("data-studio-layout-control")) expectedHead = form.getAttribute("data-studio-target-head") || "";
      var payload = { gosx_studio_operation: kind, gosx_studio_operation_id: extra.id || (crypto.randomUUID ? crypto.randomUUID() : String(Date.now()) + Math.random()), gosx_studio_page_route: extra.route || target.route || "/", gosx_studio_page_key: extra.pageId || target.pageId || target.page || "", gosx_studio_component_key: extra.componentKey || target.componentKey || target.blockKey || "", gosx_studio_binding: extra.field || target.field || "", gosx_studio_value: value || "", gosx_studio_style_property: extra.property || target.property || "", gosx_studio_style_value: value || "", gosx_studio_breakpoint: extra.breakpoint || target.breakpoint || "base", gosx_studio_state: extra.state || target.state || "default", gosx_studio_expected_revision: extra.revision || form.getAttribute("data-studio-document-revision") || "0", gosx_studio_expected_target_head: expectedHead, gosx_studio_history_operation_id: extra.historyOperationId || "",
        // Instance/interaction/flow durable operation addressing + typed
        // settings (see DURABLE_FIELD_BY_KIND / durableValue above). Unused
        // (left "") by every kind that predates those families.
        gosx_studio_control_key: extra.controlKey || "", gosx_studio_component_template_key: extra.definitionKey || "",
        gosx_studio_flow_key: extra.flowKey || "", gosx_studio_flow_action_key: extra.flowActionKey || "", gosx_studio_flow_action_label: extra.flowActionLabel || "",
        gosx_studio_flow_field_name: extra.flowFieldName || "", gosx_studio_flow_field_label: extra.flowFieldLabel || "", gosx_studio_flow_field_required: extra.required ? "true" : "",
        gosx_studio_control_kind: extra.controlKind || "",
        gosx_studio_interaction_key: extra.interactionKey || "", gosx_studio_interaction_kind: extra.interactionKind || "", gosx_studio_interaction_effect: extra.interactionEffect || "",
        gosx_studio_interaction_duration_ms: extra.durationMs || "", gosx_studio_interaction_delay_ms: extra.delayMs || "", gosx_studio_interaction_once: extra.once ? "true" : "" };
      if (extra.expectedTargetValue !== undefined && extra.expectedTargetValue !== null) {
        payload.gosx_studio_expected_target_value = JSON.stringify(extra.expectedTargetValue);
      }
      return submit(form, payload, target, extra).then(function (response) {
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
        var valueSource = button.getAttribute("data-gosx-studio-operation-value-source");
        if (valueSource) {
          var sourceNode = form.querySelector(valueSource) || document.querySelector(valueSource);
          if (sourceNode) value = "value" in sourceNode ? sourceNode.value : (sourceNode.textContent || "");
        }
        if (!value && kind === "set-field") {
          var editorValue = form.querySelector("[data-studio-operation-value]");
          if (editorValue) value = "value" in editorValue ? editorValue.value : (editorValue.textContent || "");
        }
        form.__gosxOperationRuntime.select({ route: button.getAttribute("data-gosx-studio-route") || form.getAttribute("data-studio-target-route") || "/", pageId: button.getAttribute("data-gosx-studio-page-id") || form.getAttribute("data-studio-target-page-id") || "", field: button.getAttribute("data-gosx-studio-field") || form.getAttribute("data-studio-target-field") || "", componentKey: button.getAttribute("data-gosx-studio-component") || form.getAttribute("data-studio-target-component") || "", property: button.getAttribute("data-gosx-studio-style-property") || "" });
        // set-interaction reads its Effect/Duration/Delay/Once from the
        // form's own live named controls (the operator's current dropdown
        // picks), NOT from any static button attribute — the button never
        // carries one, and even if a host did render one it would go stale
        // the instant the operator changed a select without a page reload.
        // The field names are the exact authoring.AuthoringField* literals
        // panels/interactions_panel.go's <select>/<input> elements render.
        var interactionEffect = button.getAttribute("data-gosx-studio-interaction-effect");
        var durationMs = button.getAttribute("data-gosx-studio-duration-ms");
        var delayMs = button.getAttribute("data-gosx-studio-delay-ms");
        var once = button.getAttribute("data-gosx-studio-once") === "true";
        if (kind === "set-interaction") {
          var effectNode = form.querySelector("[name='gosx_studio_interaction_effect']");
          var durationNode = form.querySelector("[name='gosx_studio_interaction_duration_ms']");
          var delayNode = form.querySelector("[name='gosx_studio_interaction_delay_ms']");
          var onceNode = form.querySelector("[name='gosx_studio_interaction_once']");
          if (effectNode) interactionEffect = effectNode.value;
          if (durationNode) durationMs = durationNode.value;
          if (delayNode) delayMs = delayNode.value;
          if (onceNode) once = !!onceNode.checked;
        }
        form.__gosxOperationRuntime.commit(kind, value, {
          componentKey: button.getAttribute("data-gosx-studio-component"), property: button.getAttribute("data-gosx-studio-style-property"),
          breakpoint: button.getAttribute("data-gosx-studio-breakpoint"), state: button.getAttribute("data-gosx-studio-style-state"),
          historyOperationId: button.getAttribute("data-gosx-studio-history-operation-id"),
          // Instance/interaction/flow durable operation addressing (see
          // DURABLE_FIELD_BY_KIND above). A panel opts into one of these
          // kinds by rendering the matching data-gosx-studio-* attribute on
          // its operation button; every kind that predates these families
          // simply never renders them, so these are always null there.
          controlKey: button.getAttribute("data-gosx-studio-control-key"), definitionKey: button.getAttribute("data-gosx-studio-definition-key"),
          flowKey: button.getAttribute("data-gosx-studio-flow-key"), flowActionKey: button.getAttribute("data-gosx-studio-flow-action-key"),
          flowActionLabel: button.getAttribute("data-gosx-studio-flow-action-label"), flowFieldName: button.getAttribute("data-gosx-studio-flow-field-name"),
          flowFieldLabel: button.getAttribute("data-gosx-studio-flow-field-label"), controlKind: button.getAttribute("data-gosx-studio-control-kind"),
          required: button.getAttribute("data-gosx-studio-required") === "true",
          interactionKey: button.getAttribute("data-gosx-studio-interaction-key"), interactionKind: button.getAttribute("data-gosx-studio-interaction-kind"),
          interactionEffect: interactionEffect, durationMs: durationMs, delayMs: delayMs, once: once
        });
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
