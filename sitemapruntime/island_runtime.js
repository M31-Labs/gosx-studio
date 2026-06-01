;(function () {
  if (typeof window === "undefined") return;

  var BOARD = "[data-studio-site-map-board='true']";
  var BOUND = "data-gosx-studio-site-map-runtime-bound";

  function attr(el, name, fallback) {
    if (!el || !el.getAttribute) return fallback || "";
    var value = el.getAttribute(name);
    return value == null || value === "" ? (fallback || "") : value;
  }

  function setAttr(el, name, value) {
    if (!el || !el.setAttribute) return;
    el.setAttribute(name, value == null ? "" : String(value));
  }

  function setVisible(el, visible) {
    if (!el) return;
    setAttr(el, "data-studio-site-map-visible", visible ? "true" : "false");
  }

  function setPanelVisible(el, visible) {
    if (!el) return;
    setAttr(el, "data-studio-site-map-panel-visible", visible ? "true" : "false");
  }

  function all(root, selector) {
    return Array.prototype.slice.call(root.querySelectorAll(selector));
  }

  function closest(target, selector) {
    return target && target.closest ? target.closest(selector) : null;
  }

  function state(root) {
    return {
      filter: attr(root, "data-studio-site-map-filter", "all"),
      detail: attr(root, "data-studio-site-map-detail", "map"),
      palette: attr(root, "data-studio-site-map-palette", "all"),
      focus: attr(root, "data-studio-site-map-focus", "all"),
      zoom: attr(root, "data-studio-site-map-zoom", "fit"),
      density: attr(root, "data-studio-site-map-density", "roomy"),
      grid: attr(root, "data-studio-site-map-grid", "on"),
      selectedNode: attr(root, "data-studio-site-map-selected-node", ""),
      selectedBlueprint: attr(root, "data-studio-site-map-selected-blueprint", ""),
      selectedTemplate: attr(root, "data-studio-site-map-selected-template", ""),
    };
  }

  function writeState(root, next) {
    setAttr(root, "data-studio-site-map-filter", next.filter);
    setAttr(root, "data-studio-site-map-detail", next.detail);
    setAttr(root, "data-studio-site-map-palette", next.palette);
    setAttr(root, "data-studio-site-map-focus", next.focus);
    setAttr(root, "data-studio-site-map-zoom", next.zoom);
    setAttr(root, "data-studio-site-map-density", next.density);
    setAttr(root, "data-studio-site-map-grid", next.grid);
    setAttr(root, "data-studio-site-map-selected-node", next.selectedNode);
    setAttr(root, "data-studio-site-map-selected-blueprint", next.selectedBlueprint);
    setAttr(root, "data-studio-site-map-selected-template", next.selectedTemplate);
  }

  function syncPressed(root, selector, current) {
    all(root, selector).forEach(function (button) {
      var target = attr(button, selector.slice(1, -1), "");
      setAttr(button, "aria-pressed", target === current ? "true" : "false");
    });
  }

  function syncToolbar(root, next) {
    syncPressed(root, "[data-studio-site-map-filter-control]", next.filter);
    syncPressed(root, "[data-studio-site-map-focus-control]", next.focus);
    syncPressed(root, "[data-studio-site-map-detail-target]", next.detail);
    syncPressed(root, "[data-studio-site-map-palette-control]", next.palette);
    syncPressed(root, "[data-studio-site-map-zoom-control]", next.zoom);
    syncPressed(root, "[data-studio-site-map-density-control]", next.density);
    syncPressed(root, "[data-studio-site-map-grid-control]", next.grid);
  }

  function syncPanels(root, next) {
    all(root, "[data-studio-site-map-workspace='true']").forEach(function (panel) {
      setPanelVisible(panel, next.detail === "map");
    });
    all(root, "[data-studio-site-map-builder='true']").forEach(function (panel) {
      setPanelVisible(panel, next.detail === "build");
    });
    all(root, "[data-studio-site-map-viewport='true']").forEach(function (panel) {
      setPanelVisible(panel, next.detail !== "map" && next.detail !== "build");
    });
  }

  function matchesGroup(group, candidate) {
    return group === "" || group === "all" || candidate === group;
  }

  function syncFilter(root, next) {
    all(root, "[data-studio-site-map-page][data-studio-site-map-group]").forEach(function (page) {
      setVisible(page, matchesGroup(next.filter, attr(page, "data-studio-site-map-group", "")));
    });
    all(root, "[data-studio-site-map-workspace-node][data-studio-site-map-group]").forEach(function (node) {
      setVisible(node, matchesGroup(next.filter, attr(node, "data-studio-site-map-group", "")));
    });
    all(root, "[data-studio-site-map-workspace-node-card][data-studio-site-map-group]").forEach(function (card) {
      setVisible(card, matchesGroup(next.filter, attr(card, "data-studio-site-map-group", "")));
    });
    all(root, "[data-studio-site-map-workspace-path][data-studio-site-map-link-from-group]").forEach(function (path) {
      var fromGroup = attr(path, "data-studio-site-map-link-from-group", "");
      var toGroup = attr(path, "data-studio-site-map-link-to-group", "");
      setVisible(path, next.filter === "all" || fromGroup === next.filter || toGroup === next.filter);
    });
    all(root, "[data-studio-site-map-workspace-link][data-studio-site-map-link-from-group]").forEach(function (link) {
      var fromGroup = attr(link, "data-studio-site-map-link-from-group", "");
      var toGroup = attr(link, "data-studio-site-map-link-to-group", "");
      setVisible(link, next.filter === "all" || fromGroup === next.filter || toGroup === next.filter);
    });
  }

  function syncFocus(root, next) {
    all(root, "[data-studio-site-map-workspace-layer]").forEach(function (layer) {
      var key = attr(layer, "data-studio-site-map-workspace-layer", "");
      setVisible(layer, next.focus === "" || next.focus === "all" || key === next.focus || key === "resources");
    });
  }

  function syncPalette(root, next) {
    all(root, "[data-studio-site-map-component-template][data-studio-site-map-template-category]").forEach(function (card) {
      var category = attr(card, "data-studio-site-map-template-category", "");
      setVisible(card, next.palette === "" || next.palette === "all" || category === next.palette);
    });
  }

  function nodeByKey(root, key) {
    if (!key) return null;
    var nodes = all(root, "[data-studio-site-map-workspace-node]");
    for (var i = 0; i < nodes.length; i += 1) {
      if (attr(nodes[i], "data-studio-site-map-workspace-node", "") === key) return nodes[i];
    }
    return null;
  }

  function syncSelection(root, next) {
    all(root, "[data-studio-site-map-workspace-node]").forEach(function (node) {
      setAttr(node, "data-studio-site-map-node-selected", attr(node, "data-studio-site-map-workspace-node", "") === next.selectedNode ? "true" : "false");
    });
    all(root, "[data-studio-site-map-workspace-node-card]").forEach(function (card) {
      setAttr(card, "data-studio-site-map-node-selected", attr(card, "data-studio-site-map-workspace-node-card", "") === next.selectedNode ? "true" : "false");
    });
    all(root, "[data-studio-site-map-workspace-path], [data-studio-site-map-workspace-link]").forEach(function (link) {
      var selected = attr(link, "data-studio-site-map-link-from", "") === next.selectedNode || attr(link, "data-studio-site-map-link-to", "") === next.selectedNode;
      setAttr(link, "data-studio-site-map-link-selected", selected ? "true" : "false");
    });

    var node = nodeByKey(root, next.selectedNode);
    if (!node) return;
    writeSelectionDetails(root, node);
  }

  function textAttr(node, name, fallback) {
    return attr(node, name, fallback || "");
  }

  function writeText(root, selector, value) {
    all(root, selector).forEach(function (target) {
      target.textContent = value || "";
    });
  }

  function writeOptional(root, rowSelector, textSelector, value) {
    all(root, rowSelector).forEach(function (row) {
      setVisible(row, !!value);
    });
    writeText(root, textSelector, value);
  }

  function writeSelectionDetails(root, node) {
    var kind = textAttr(node, "data-studio-site-map-node-kind-label", textAttr(node, "data-studio-site-map-node-kind", ""));
    var label = textAttr(node, "data-studio-site-map-node-label", node.textContent || "");
    var summary = textAttr(node, "data-studio-site-map-node-summary", "");
    var route = textAttr(node, "data-studio-site-map-route", "");
    var source = textAttr(node, "data-studio-site-map-node-source-label", textAttr(node, "data-studio-site-map-source", ""));
    var binding = textAttr(node, "data-studio-site-map-node-binding-label", textAttr(node, "data-studio-site-map-binding", ""));
    var status = textAttr(node, "data-studio-site-map-node-status-label", "");
    var component = textAttr(node, "data-studio-site-map-node-component-label", textAttr(node, "data-studio-gosx-component", ""));

    writeText(root, "[data-studio-site-map-selected-kind]", kind);
    writeText(root, "[data-studio-site-map-selected-label]", label);
    writeText(root, "[data-studio-site-map-selected-status]", status);
    writeOptional(root, "[data-studio-site-map-selected-summary-row]", "[data-studio-site-map-selected-summary]", summary);
    writeOptional(root, "[data-studio-site-map-selected-route-row]", "[data-studio-site-map-selected-route]", route);
    writeOptional(root, "[data-studio-site-map-selected-source-row]", "[data-studio-site-map-selected-source]", source);
    writeOptional(root, "[data-studio-site-map-selected-binding-row]", "[data-studio-site-map-selected-binding]", binding);
    writeOptional(root, "[data-studio-site-map-selected-component-row]", "[data-studio-site-map-selected-component]", component);
  }

  function syncIntentActive(root, next) {
    all(root, "[data-studio-composition-intent]").forEach(function (intent) {
      var kind = attr(intent, "data-studio-composition-kind", "");
      var active = false;
      if (kind === "create-page") {
        active = attr(intent, "data-studio-composition-blueprint", "") === next.selectedBlueprint;
      } else if (kind === "add-component") {
        active = attr(intent, "data-studio-composition-template", "") === next.selectedTemplate;
      }
      setAttr(intent, "data-studio-composition-active", active ? "true" : "false");
    });
  }

  function sync(root) {
    var next = state(root);
    writeState(root, next);
    syncToolbar(root, next);
    syncPanels(root, next);
    syncFilter(root, next);
    syncFocus(root, next);
    syncPalette(root, next);
    syncSelection(root, next);
    syncIntentActive(root, next);
    return next;
  }

  function setState(root, patch) {
    var next = state(root);
    Object.keys(patch || {}).forEach(function (key) {
      next[key] = patch[key];
    });
    writeState(root, next);
    next = sync(root);
    root.dispatchEvent(new CustomEvent("gosxstudio:site-map-change", { bubbles: true, detail: next }));
    return next;
  }

  function controlValue(target, attrName) {
    var control = closest(target, "[" + attrName + "]");
    return control ? attr(control, attrName, "") : "";
  }

  function handleClick(root, event) {
    var target = event.target;
    var value = "";

    value = controlValue(target, "data-studio-site-map-filter-control");
    if (value) return setState(root, { filter: value });

    value = controlValue(target, "data-studio-site-map-focus-control");
    if (value) return setState(root, { focus: value });

    value = controlValue(target, "data-studio-site-map-detail-target");
    if (value) {
      var patch = { detail: value };
      if (value === "build") patch.palette = "page-sections";
      return setState(root, patch);
    }

    value = controlValue(target, "data-studio-site-map-zoom-control");
    if (value) return setState(root, { zoom: value });

    value = controlValue(target, "data-studio-site-map-density-control");
    if (value) return setState(root, { density: value });

    value = controlValue(target, "data-studio-site-map-grid-control");
    if (value) return setState(root, { grid: value });

    value = controlValue(target, "data-studio-site-map-palette-control");
    if (value) return setState(root, { palette: value });

    var node = closest(target, "[data-studio-site-map-workspace-node]");
    if (node) return setState(root, { selectedNode: attr(node, "data-studio-site-map-workspace-node", "") });

    var blueprint = closest(target, "[data-studio-site-map-blueprint]");
    if (blueprint) return setState(root, { selectedBlueprint: attr(blueprint, "data-studio-site-map-blueprint", "") });

    var template = closest(target, "[data-studio-site-map-component-template]");
    if (template) return setState(root, { selectedTemplate: attr(template, "data-studio-site-map-component-template", "") });
  }

  function handleChange(root, event) {
    var target = event.target;
    if (!target || !target.matches) return;
    if (target.matches(".studio-site-map-workspace-node__input")) {
      setState(root, { selectedNode: attr(target, "value", target.value || "") });
      return;
    }
    var blueprint = closest(target, "[data-studio-site-map-blueprint]");
    if (blueprint) {
      setState(root, { selectedBlueprint: attr(blueprint, "data-studio-site-map-blueprint", "") });
      return;
    }
    var template = closest(target, "[data-studio-site-map-component-template]");
    if (template) {
      setState(root, { selectedTemplate: attr(template, "data-studio-site-map-component-template", "") });
    }
  }

  function bind(root) {
    if (!root || attr(root, BOUND, "") === "true") return root;
    setAttr(root, BOUND, "true");
    root.addEventListener("click", function (event) {
      handleClick(root, event);
    });
    root.addEventListener("change", function (event) {
      handleChange(root, event);
    });
    sync(root);
    return root;
  }

  function bindAll(scope) {
    all(scope || document, BOARD).forEach(bind);
  }

  window.GoSXStudioSiteMapRuntime = {
    bind: bind,
    bindAll: bindAll,
    setState: setState,
    sync: sync,
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () { bindAll(document); });
  } else {
    bindAll(document);
  }
  document.addEventListener("gosxstudio:fragments-refreshed", function () {
    bindAll(document);
  });
})();
