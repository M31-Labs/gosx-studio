(function () {
  "use strict";

  var STATE_KEY = "__gosxStudioFlowRuntime";

  function ready(fn) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", fn, { once: true });
      return;
    }
    fn();
  }

  function text(node, value) {
    if (node) node.textContent = value;
  }

  function compact(value) {
    return String(value || "").replace(/\s+/g, " ").trim();
  }

  function storageKey(scope) {
    return "gosx-studio-flow:" + compact(scope.getAttribute("data-studio-flow-storage-key") || scope.getAttribute("data-studio-shell") || window.location.pathname || "studio");
  }

  function storedFlowKey(scope) {
    try {
      return window.sessionStorage ? window.sessionStorage.getItem(storageKey(scope)) || "" : "";
    } catch (error) {
      return "";
    }
  }

  function storeFlowKey(scope, key) {
    try {
      if (window.sessionStorage) window.sessionStorage.setItem(storageKey(scope), key);
    } catch (error) {
      return;
    }
  }

  function hashFlowKey(flowByKey) {
    if (!window.location.hash) return "";
    var value = "";
    try {
      value = decodeURIComponent(window.location.hash.slice(1));
    } catch (error) {
      value = window.location.hash.slice(1);
    }
    if (value.indexOf("flow=") === 0) value = value.slice("flow=".length);
    else if (value.indexOf("flow-") === 0) value = value.slice("flow-".length);
    return flowByKey.has(value) ? value : "";
  }

  function queryAll(root, selector) {
    return Array.prototype.slice.call(root.querySelectorAll(selector));
  }

  function emit(scope, name, detail) {
    scope.dispatchEvent(new CustomEvent(name, { bubbles: true, detail: detail || {} }));
  }

  function flowState(scope) {
    if (!scope[STATE_KEY]) {
      scope[STATE_KEY] = {
        bound: false,
        hashBound: false,
        selectedKey: "",
        cards: [],
        editors: [],
        flows: [],
        flowByKey: new Map(),
        editorByKey: new Map()
      };
    }
    return scope[STATE_KEY];
  }

  function setPreviewState(state, value) {
    var label = "Ready";
    if (value === "loading") label = "Loading";
    else if (value === "error") label = "Preview failed";
    if (state.preview) state.preview.setAttribute("data-studio-preview-state", value);
    text(state.previewStatus, label);
  }

  function setPreviewMeta(state, flow) {
    text(state.selectedLabel, flow.label);
    text(state.selectedRoute, flow.route || "No public route");
    if (!state.openPreview) return;
    if (flow.route) {
      state.openPreview.setAttribute("href", flow.route);
      state.openPreview.removeAttribute("aria-disabled");
      state.openPreview.classList.remove("is-disabled");
    } else {
      state.openPreview.setAttribute("href", "#");
      state.openPreview.setAttribute("aria-disabled", "true");
      state.openPreview.classList.add("is-disabled");
    }
  }

  function setPreview(state, route) {
    if (!state.preview || !route) return;
    if (state.preview.getAttribute("src") === route) {
      setPreviewState(state, "ready");
      return;
    }
    setPreviewState(state, "loading");
    state.preview.setAttribute("src", route);
  }

  function flowForCard(state, card) {
    if (!card) return null;
    return state.flowByKey.get(card.getAttribute("data-studio-flow-card") || "") || null;
  }

  function selectFlow(state, key, route, options) {
    if (!key || !state.flowByKey.has(key)) return;
    options = options || {};
    var flow = state.flowByKey.get(key);
    state.flows.forEach(function (item) {
      var active = item.key === key;
      item.card.classList.toggle("is-selected", active);
      item.card.setAttribute("aria-selected", active ? "true" : "false");
      item.card.tabIndex = active ? 0 : -1;
      if (item.editor) {
        item.editor.classList.toggle("is-selected", active);
        item.editor.hidden = !active;
        item.editor.setAttribute("aria-hidden", active ? "false" : "true");
      }
    });
    state.selectedKey = key;
    state.scope.setAttribute("data-studio-selected-flow", key);
    setPreviewMeta(state, flow);
    if (options.persist !== false) storeFlowKey(state.scope, key);
    if (options.focus && flow.card.focus) flow.card.focus();
    if (options.preview !== false) setPreview(state, route || flow.route);
    emit(state.scope, "gosxstudio:flow-select", {
      key: key,
      label: flow.label,
      route: route || flow.route,
      status: flow.status,
      reason: options.reason || ""
    });
  }

  function selectByIndex(state, index) {
    var flow = state.flows[index];
    if (flow) selectFlow(state, flow.key, flow.route, { focus: true, reason: "keyboard" });
  }

  function findFlowKeyByRoute(state, route) {
    for (var index = 0; index < state.flows.length; index += 1) {
      if (state.flows[index].route === route) return state.flows[index].key;
    }
    return "";
  }

  function updateDirtyState(state, editor) {
    var flow = editor && state.flowByKey.get(editor.getAttribute("data-studio-flow-editor") || "");
    if (!flow) return;
    var dirtyCount = 0;
    queryAll(editor, "[data-studio-initial-value]").forEach(function (input) {
      if (input.value !== input.getAttribute("data-studio-initial-value")) dirtyCount += 1;
    });
    var dirtyLabel = dirtyCount === 1 ? "1 unsaved" : String(dirtyCount) + " unsaved";
    editor.classList.toggle("is-dirty", dirtyCount > 0);
    flow.card.classList.toggle("is-dirty", dirtyCount > 0);
    flow.card.setAttribute("data-studio-dirty-count", String(dirtyCount));
    queryAll(flow.card, "[data-studio-flow-dirty-badge]").forEach(function (badge) {
      badge.hidden = dirtyCount === 0;
      text(badge, dirtyLabel);
    });
    queryAll(editor, "[data-studio-flow-editor-dirty]").forEach(function (badge) {
      badge.hidden = dirtyCount === 0;
      text(badge, dirtyLabel);
    });
    emit(state.scope, "gosxstudio:flow-dirty", {
      key: flow.key,
      label: flow.label,
      dirtyCount: dirtyCount
    });
  }

  function syncSavedFlowState(state) {
    state.editors.forEach(function (editor) {
      queryAll(editor, "input, textarea, select").forEach(function (input) {
        input.setAttribute("data-studio-initial-value", input.value || "");
      });
      updateDirtyState(state, editor);
    });
  }

  function bindCard(scope, card) {
    if (card.__gosxStudioFlowCardBound) return;
    card.__gosxStudioFlowCardBound = true;
    card.addEventListener("click", function () {
      var state = flowState(scope);
      var flow = flowForCard(state, card);
      if (flow) selectFlow(state, flow.key, flow.route, { reason: "click" });
    });
    card.addEventListener("keydown", function (event) {
      var state = flowState(scope);
      var flow = flowForCard(state, card);
      if (!flow) return;
      if (event.key === "ArrowRight" || event.key === "ArrowDown") {
        event.preventDefault();
        selectByIndex(state, (flow.index + 1) % state.flows.length);
      } else if (event.key === "ArrowLeft" || event.key === "ArrowUp") {
        event.preventDefault();
        selectByIndex(state, (flow.index - 1 + state.flows.length) % state.flows.length);
      } else if (event.key === "Home") {
        event.preventDefault();
        selectByIndex(state, 0);
      } else if (event.key === "End") {
        event.preventDefault();
        selectByIndex(state, state.flows.length - 1);
      }
    });
  }

  function bindPreview(scope, preview) {
    if (!preview || preview.__gosxStudioFlowPreviewBound) return;
    preview.__gosxStudioFlowPreviewBound = true;
    preview.addEventListener("load", function () { setPreviewState(flowState(scope), "ready"); });
    preview.addEventListener("error", function () { setPreviewState(flowState(scope), "error"); });
  }

  function bindScope(scope, state) {
    if (!state.bound) {
      state.bound = true;
      scope.addEventListener("click", function (event) {
        var current = flowState(scope);
        if (current.openPreview && event.target && event.target.closest && event.target.closest("[data-studio-open-preview]") === current.openPreview && current.openPreview.getAttribute("aria-disabled") === "true") {
          event.preventDefault();
          return;
        }
        var button = event.target && event.target.closest && event.target.closest("[data-studio-preview-flow]");
        if (!button || !scope.contains(button)) return;
        var route = button.getAttribute("data-studio-preview-flow") || "";
        var owningEditor = button.closest("[data-studio-flow-editor]");
        var flowKey = owningEditor ? owningEditor.getAttribute("data-studio-flow-editor") : findFlowKeyByRoute(current, route);
        if (flowKey) selectFlow(current, flowKey, route, { preview: false, reason: "preview-button" });
        setPreview(current, route);
        emit(scope, "gosxstudio:flow-preview", { key: flowKey || "", route: route });
      });
      scope.addEventListener("input", function (event) {
        var editor = event.target && event.target.closest && event.target.closest("[data-studio-flow-editor]");
        if (editor && scope.contains(editor)) updateDirtyState(flowState(scope), editor);
      });
      scope.addEventListener("gosxstudio:command", function (event) {
        var current = flowState(scope);
        var detail = event.detail || {};
        if (detail.href && detail.href.indexOf("#flow=") === 0) {
          var key = detail.href.slice("#flow=".length);
          if (current.flowByKey.has(key)) selectFlow(current, key, current.flowByKey.get(key).route, { preview: true, persist: true, reason: "command" });
        }
      });
      scope.addEventListener("gosxstudio:save-state", function (event) {
        if (event.detail && event.detail.state === "saved") syncSavedFlowState(flowState(scope));
      });
    }

    if (!state.hashBound) {
      state.hashBound = true;
      window.addEventListener("hashchange", function () {
        var current = flowState(scope);
        var key = hashFlowKey(current.flowByKey);
        if (key) selectFlow(current, key, current.flowByKey.get(key).route, { preview: true, persist: true, reason: "hash" });
      });
    }
  }

  function collectScopes(root) {
    var scope = root && root.querySelectorAll ? root : document;
    var scopes = [];
    if (scope.matches && scope.matches("form[data-studio-workbench], [data-studio-flow-runtime]")) scopes.push(scope);
    return scopes.concat(queryAll(scope, "form[data-studio-workbench], [data-studio-flow-runtime]"));
  }

  function initFlowEditor(scope) {
    if (!scope) return;
    var cards = queryAll(scope, "[data-studio-flow-card]");
    var editors = queryAll(scope, "[data-studio-flow-editor]");
    if (!cards.length || !editors.length) return;

    var state = flowState(scope);
    state.scope = scope;
    state.cards = cards;
    state.editors = editors;
    state.preview = scope.querySelector("[data-studio-preview-frame]");
    state.selectedLabel = scope.querySelector("[data-studio-selected-flow-label]");
    state.selectedRoute = scope.querySelector("[data-studio-selected-flow-route]");
    state.previewStatus = scope.querySelector("[data-studio-preview-status]");
    state.openPreview = scope.querySelector("[data-studio-open-preview]");
    state.editorByKey = new Map();
    editors.forEach(function (editor) {
      state.editorByKey.set(editor.getAttribute("data-studio-flow-editor") || "", editor);
    });

    state.flows = cards.map(function (card, index) {
      var key = card.getAttribute("data-studio-flow-card") || "";
      return {
        key: key,
        index: index,
        card: card,
        editor: state.editorByKey.get(key),
        route: card.getAttribute("data-studio-flow-route") || "",
        label: card.getAttribute("data-studio-flow-label") || compact(card.textContent),
        status: card.getAttribute("data-studio-flow-status") || ""
      };
    }).filter(function (flow) {
      return !!flow.key;
    });
    if (!state.flows.length) return;

    state.flowByKey = new Map();
    state.flows.forEach(function (flow) { state.flowByKey.set(flow.key, flow); });
    scope.dataset.gosxStudioFlowRuntimeBound = "true";
    bindScope(scope, state);
    bindPreview(scope, state.preview);
    cards.forEach(function (card) { bindCard(scope, card); });

    editors.forEach(function (editor) {
      queryAll(editor, "input, textarea, select").forEach(function (input) {
        if (!input.hasAttribute("data-studio-initial-value")) {
          input.setAttribute("data-studio-initial-value", input.value || "");
        }
      });
      updateDirtyState(state, editor);
    });

    var initialKey = hashFlowKey(state.flowByKey) || storedFlowKey(scope) || state.selectedKey;
    if (!state.flowByKey.has(initialKey)) initialKey = state.flows[0].key;
    selectFlow(state, initialKey, state.flowByKey.get(initialKey).route, { preview: true, persist: true, reason: "init" });
  }

  function initAll(root) {
    collectScopes(root).forEach(initFlowEditor);
  }

  ready(function () { initAll(document); });
  document.addEventListener("gosx:navigate", function () { initAll(document); });
  document.addEventListener("gosx:render", function (event) { initAll(event.target || document); });
})();
