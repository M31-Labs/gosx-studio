;(function () {
  if (typeof window === "undefined") return;

  var BOARD = "[data-studio-site-map-board='true']";
  var LAYOUT_ACTION = "data-gosx-studio-site-map-layout-action";
  var BOUND = "data-gosx-studio-site-map-runtime-bound";
  var BOUND_PROP = "__gosxStudioSiteMapRuntimeBound";

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

  // ---- Infinite-canvas viewport (continuous pan + zoom) ----
  var ZOOM_STEP = 1.25;
  var WHEEL_STEP = 1.1;
  var MIN_SCALE = 0.25;
  var MAX_SCALE = 3;

  // ---- Node drag-to-reposition ----
  // Movement (in screen px) a pointer must travel before a press on a node
  // becomes a drag; below it the press stays a click (select). Snap grid is in
  // pan-surface-LOCAL px (the units left/top are written in).
  var DRAG_THRESHOLD = 4;
  var NODE_SNAP_GRID = 20;

  function num(el, name, fallback) {
    var value = parseFloat(attr(el, name, ""));
    return isFinite(value) ? value : fallback;
  }

  function snapTo(value, grid) {
    if (!grid) return value;
    return Math.round(value / grid) * grid;
  }

  function styleNum(el, prop) {
    var value = el && el.style ? parseFloat(el.style[prop]) : NaN;
    return isFinite(value) ? value : null;
  }

  function clampScale(value) {
    if (!isFinite(value)) return 1;
    return Math.min(MAX_SCALE, Math.max(MIN_SCALE, value));
  }

  function trimNum(value) {
    return String(Math.round(value * 10000) / 10000);
  }

  function canvasEl(root) {
    return root.querySelector("[data-studio-site-map-canvas]");
  }

  function canvasPoint(root, clientX, clientY) {
    var canvas = canvasEl(root);
    if (!canvas || !canvas.getBoundingClientRect) return { x: 0, y: 0 };
    var rect = canvas.getBoundingClientRect();
    return { x: clientX - rect.left, y: clientY - rect.top };
  }

  function canvasCenter(root) {
    var canvas = canvasEl(root);
    if (!canvas || !canvas.getBoundingClientRect) return { x: 0, y: 0 };
    var rect = canvas.getBoundingClientRect();
    return { x: rect.width / 2, y: rect.height / 2 };
  }

  function viewportTransform(next) {
    return "translate(" + trimNum(next.panX) + "px, " + trimNum(next.panY) + "px) scale(" + trimNum(clampScale(next.scale)) + ")";
  }

  function applyViewport(root, next) {
    all(root, "[data-studio-site-map-pan-surface]").forEach(function (surface) {
      surface.style.transformOrigin = "0 0";
      surface.style.transform = viewportTransform(next);
    });
    writeText(root, "[data-studio-site-map-zoom-readout]", Math.round(clampScale(next.scale) * 100) + "%");
  }

  // Zoom keeps the world point under `point` (canvas-local px) fixed.
  function zoomAbout(root, newScale, point) {
    var current = state(root);
    var scale = clampScale(newScale);
    var worldX = (point.x - current.panX) / current.scale;
    var worldY = (point.y - current.panY) / current.scale;
    return setState(root, {
      scale: scale,
      panX: point.x - worldX * scale,
      panY: point.y - worldY * scale,
    });
  }

  function zoomAction(root, action) {
    if (action === "reset" || action === "fit") {
      return setState(root, { scale: 1, panX: 0, panY: 0 });
    }
    var current = state(root);
    var factor = action === "in" ? ZOOM_STEP : action === "out" ? 1 / ZOOM_STEP : 1;
    return zoomAbout(root, current.scale * factor, canvasCenter(root));
  }

  function handleWheel(root, event) {
    if (!closest(event.target, "[data-studio-site-map-canvas]")) return;
    if (event.cancelable) event.preventDefault();
    var current = state(root);
    var factor = event.deltaY < 0 ? WHEEL_STEP : 1 / WHEEL_STEP;
    zoomAbout(root, current.scale * factor, canvasPoint(root, event.clientX, event.clientY));
  }

  // Left-drag pans only from empty canvas; middle-drag pans from anywhere.
  function viewportDragAllowed(target) {
    return !closest(target, "button, a, input, textarea, select, label, [data-studio-site-map-workspace-node], [data-studio-site-map-workspace-node-card]");
  }

  function handlePointerDown(root, event) {
    if (event.button !== 0 && event.button !== 1) return;
    if (!closest(event.target, "[data-studio-site-map-canvas]")) return;
    var onEmpty = viewportDragAllowed(event.target);
    // Plain left-press on a node = potential drag-to-reposition. Shift is
    // reserved for marquee/multi-select, so it never starts a node drag.
    if (event.button === 0 && !event.shiftKey && !onEmpty) {
      var node = closest(event.target, "[data-studio-site-map-workspace-node]");
      if (node) return startNodeDrag(root, node, event);
    }
    // Shift + left-drag on empty canvas = marquee multi-select.
    if (event.button === 0 && event.shiftKey && onEmpty) return startMarquee(root, event);
    // Plain left-drag must start on empty canvas; middle-drag pans from anywhere.
    if (event.button === 0 && !onEmpty) return;
    startPan(root, event);
  }

  // Live-move a node on the pan surface. Screen-pixel deltas are converted to
  // pan-surface-LOCAL units by dividing by the current scale, so a node tracks
  // the pointer at any zoom. A press that never crosses DRAG_THRESHOLD stays a
  // click (the click handler selects it) and emits no move event.
  function startNodeDrag(root, node, event) {
    var key = attr(node, "data-studio-site-map-workspace-node", "");
    var startClientX = event.clientX;
    var startClientY = event.clientY;
    var startLeft = styleNum(node, "left");
    if (startLeft == null) startLeft = node.offsetLeft || 0;
    var startTop = styleNum(node, "top");
    if (startTop == null) startTop = node.offsetTop || 0;
    var dragging = false;
    function move(ev) {
      var dxScreen = ev.clientX - startClientX;
      var dyScreen = ev.clientY - startClientY;
      if (!dragging && Math.abs(dxScreen) < DRAG_THRESHOLD && Math.abs(dyScreen) < DRAG_THRESHOLD) return;
      if (!dragging) {
        dragging = true;
        setAttr(root, "data-studio-site-map-node-dragging", "true");
      }
      var scale = state(root).scale || 1;
      node.style.position = "absolute";
      node.style.left = startLeft + dxScreen / scale + "px";
      node.style.top = startTop + dyScreen / scale + "px";
    }
    function up() {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", up);
      if (!dragging) return; // sub-threshold press: leave it to click-to-select.
      setAttr(root, "data-studio-site-map-node-dragging", "false");
      var x = snapTo(styleNum(node, "left") || 0, NODE_SNAP_GRID);
      var y = snapTo(styleNum(node, "top") || 0, NODE_SNAP_GRID);
      node.style.left = x + "px";
      node.style.top = y + "px";
      setAttr(node, "data-studio-site-map-node-x", x);
      setAttr(node, "data-studio-site-map-node-y", y);
      root.dispatchEvent(new CustomEvent("gosxstudio:sitemap-node-moved", {
        bubbles: true,
        detail: { key: key, x: x, y: y },
      }));
    }
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", up);
  }

  function startPan(root, event) {
    var origin = state(root);
    var startX = event.clientX;
    var startY = event.clientY;
    setAttr(root, "data-studio-site-map-panning", "true");
    function move(ev) {
      setState(root, {
        panX: origin.panX + (ev.clientX - startX),
        panY: origin.panY + (ev.clientY - startY),
      });
    }
    function up() {
      setAttr(root, "data-studio-site-map-panning", "false");
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", up);
    }
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", up);
  }

  function parseKeyList(value) {
    return String(value || "").split(",").map(function (item) {
      return item.trim();
    }).filter(Boolean);
  }

  function marqueeRect(startX, startY, ev) {
    return {
      x: Math.min(startX, ev.clientX),
      y: Math.min(startY, ev.clientY),
      w: Math.abs(ev.clientX - startX),
      h: Math.abs(ev.clientY - startY),
    };
  }

  function rectsIntersect(box, nodeRect) {
    return !(nodeRect.right < box.x || nodeRect.left > box.x + box.w || nodeRect.bottom < box.y || nodeRect.top > box.y + box.h);
  }

  function ensureMarqueeOverlay(canvas) {
    var overlay = canvas.querySelector("[data-studio-site-map-marquee]");
    if (!overlay) {
      overlay = document.createElement("div");
      overlay.setAttribute("data-studio-site-map-marquee", "true");
      overlay.className = "studio-site-map-marquee";
      overlay.style.position = "absolute";
      overlay.style.pointerEvents = "none";
      overlay.style.display = "none";
      canvas.appendChild(overlay);
    }
    return overlay;
  }

  function startMarquee(root, event) {
    var canvas = canvasEl(root);
    if (!canvas) return;
    var overlay = ensureMarqueeOverlay(canvas);
    var startX = event.clientX;
    var startY = event.clientY;
    var moved = false;
    setAttr(root, "data-studio-site-map-marqueeing", "true");
    function move(ev) {
      var box = marqueeRect(startX, startY, ev);
      if (box.w > 3 || box.h > 3) moved = true;
      var canvasRect = canvas.getBoundingClientRect();
      overlay.style.display = "block";
      overlay.style.left = box.x - canvasRect.left + "px";
      overlay.style.top = box.y - canvasRect.top + "px";
      overlay.style.width = box.w + "px";
      overlay.style.height = box.h + "px";
    }
    function up(ev) {
      document.removeEventListener("pointermove", move);
      document.removeEventListener("pointerup", up);
      overlay.style.display = "none";
      setAttr(root, "data-studio-site-map-marqueeing", "false");
      if (!moved) return;
      var box = marqueeRect(startX, startY, ev);
      var selected = [];
      all(root, "[data-studio-site-map-workspace-node]").forEach(function (node) {
        if (rectsIntersect(box, node.getBoundingClientRect())) {
          selected.push(attr(node, "data-studio-site-map-workspace-node", ""));
        }
      });
      setState(root, { selectedNodes: selected });
    }
    document.addEventListener("pointermove", move);
    document.addEventListener("pointerup", up);
  }

  // ---- Keyboard node navigation (spatial arrows + activate) ----
  function nodeCenter(node) {
    var rect = node.getBoundingClientRect();
    return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
  }

  function visibleNodes(root) {
    return all(root, "[data-studio-site-map-workspace-node]").filter(function (node) {
      return attr(node, "data-studio-site-map-visible", "true") !== "false";
    });
  }

  // Topmost-leftmost visible node (smallest top, then smallest left).
  function firstNodeKey(root) {
    var best = null;
    var bestCenter = null;
    visibleNodes(root).forEach(function (node) {
      var c = nodeCenter(node);
      if (!best || c.y < bestCenter.y - 0.5 || (Math.abs(c.y - bestCenter.y) <= 0.5 && c.x < bestCenter.x)) {
        best = node;
        bestCenter = c;
      }
    });
    return best ? attr(best, "data-studio-site-map-workspace-node", "") : "";
  }

  // Nearest visible node strictly in `dir` from the current node. Cost is the
  // primary-axis distance plus a 2x perpendicular-axis penalty, so we prefer
  // the node most directly in the pressed direction.
  function nearestNodeKey(root, fromKey, dir) {
    var fromNode = nodeByKey(root, fromKey);
    if (!fromNode) return "";
    var origin = nodeCenter(fromNode);
    var best = null;
    var bestCost = Infinity;
    visibleNodes(root).forEach(function (node) {
      var key = attr(node, "data-studio-site-map-workspace-node", "");
      if (key === fromKey) return;
      var c = nodeCenter(node);
      var dx = c.x - origin.x;
      var dy = c.y - origin.y;
      var primary;
      var perpendicular;
      if (dir === "ArrowLeft") {
        if (dx >= 0) return;
        primary = -dx;
        perpendicular = Math.abs(dy);
      } else if (dir === "ArrowRight") {
        if (dx <= 0) return;
        primary = dx;
        perpendicular = Math.abs(dy);
      } else if (dir === "ArrowUp") {
        if (dy >= 0) return;
        primary = -dy;
        perpendicular = Math.abs(dx);
      } else { // ArrowDown
        if (dy <= 0) return;
        primary = dy;
        perpendicular = Math.abs(dx);
      }
      var cost = primary + perpendicular * 2;
      if (cost < bestCost) {
        bestCost = cost;
        best = key;
      }
    });
    return best || "";
  }

  function navigateNodes(root, dir) {
    var current = state(root).selectedNode;
    var next = current ? nearestNodeKey(root, current, dir) : firstNodeKey(root);
    if (!next || next === current) return;
    setState(root, { selectedNode: next });
  }

  function isArrowKey(key) {
    return key === "ArrowUp" || key === "ArrowDown" || key === "ArrowLeft" || key === "ArrowRight";
  }

  // True when focus is on an interactive control (button/link/editable). Node
  // navigation and activate apply to the board/canvas surface only — they must
  // never hijack a focused control's native keyboard activation (Enter/Space).
  function isInteractiveControl(target) {
    return !!closest(target, "button, a, [role='button'], [contenteditable=''], [contenteditable='true']");
  }

  function handleKeydown(root, event) {
    var target = event.target;
    if (target && target.matches && target.matches("input, textarea, select")) return;
    if (event.key === "+" || event.key === "=") {
      event.preventDefault();
      zoomAction(root, "in");
    } else if (event.key === "-" || event.key === "_") {
      event.preventDefault();
      zoomAction(root, "out");
    } else if (event.key === "0") {
      event.preventDefault();
      zoomAction(root, "reset");
    } else if (isArrowKey(event.key)) {
      if (isInteractiveControl(target)) return;
      event.preventDefault();
      navigateNodes(root, event.key);
    } else if (event.key === "Enter" || event.key === " " || event.key === "Spacebar") {
      if (isInteractiveControl(target)) return;
      var selected = state(root).selectedNode;
      if (!selected) return;
      event.preventDefault();
      root.dispatchEvent(new CustomEvent("gosxstudio:site-map-activate", {
        bubbles: true,
        detail: { key: selected },
      }));
    } else if (event.key === "Escape") {
      if (state(root).selectedNodes.length) {
        event.preventDefault();
        setState(root, { selectedNodes: [] });
      }
    }
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
      scale: clampScale(num(root, "data-studio-site-map-scale", 1)),
      panX: num(root, "data-studio-site-map-pan-x", 0),
      panY: num(root, "data-studio-site-map-pan-y", 0),
      selectedNodes: parseKeyList(attr(root, "data-studio-site-map-selected-nodes", "")),
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
    setAttr(root, "data-studio-site-map-scale", trimNum(clampScale(next.scale)));
    setAttr(root, "data-studio-site-map-pan-x", trimNum(next.panX));
    setAttr(root, "data-studio-site-map-pan-y", trimNum(next.panY));
    setAttr(root, "data-studio-site-map-selected-nodes", (next.selectedNodes || []).join(","));
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
    var multi = next.selectedNodes || [];
    function isSelected(key) {
      return key === next.selectedNode || multi.indexOf(key) >= 0;
    }
    all(root, "[data-studio-site-map-workspace-node]").forEach(function (node) {
      setAttr(node, "data-studio-site-map-node-selected", isSelected(attr(node, "data-studio-site-map-workspace-node", "")) ? "true" : "false");
    });
    all(root, "[data-studio-site-map-workspace-node-card]").forEach(function (card) {
      setAttr(card, "data-studio-site-map-node-selected", isSelected(attr(card, "data-studio-site-map-workspace-node-card", "")) ? "true" : "false");
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
    applyViewport(root, next);
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
    if (control && control.matches && control.matches(BOARD)) return "";
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

    value = controlValue(target, "data-studio-site-map-zoom-action");
    if (value) return zoomAction(root, value);

    value = controlValue(target, "data-studio-site-map-density-control");
    if (value) return setState(root, { density: value });

    value = controlValue(target, "data-studio-site-map-grid-control");
    if (value) return setState(root, { grid: value });

    value = controlValue(target, "data-studio-site-map-palette-control");
    if (value) return setState(root, { palette: value });

    var node = closest(target, "[data-studio-site-map-workspace-node]");
    if (node) {
      var nodeKey = attr(node, "data-studio-site-map-workspace-node", "");
      if (event.shiftKey) {
        var selected = state(root).selectedNodes.slice();
        var at = selected.indexOf(nodeKey);
        if (at >= 0) selected.splice(at, 1);
        else selected.push(nodeKey);
        return setState(root, { selectedNodes: selected });
      }
      return setState(root, { selectedNode: nodeKey, selectedNodes: [] });
    }

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

  function finiteNumber(value) {
    var number = Number(value);
    return isFinite(number) ? number : null;
  }

  function csrfToken() {
    var input = document.querySelector('input[name="csrf_token"]');
    return input ? input.value : "";
  }

  function layoutAction(root) {
    var source = attr(root, LAYOUT_ACTION, "");
    if (source) return source;
    var host = closest(root, "[" + LAYOUT_ACTION + "]");
    return host ? attr(host, LAYOUT_ACTION, "") : "";
  }

  function postLayoutPosition(action, key, x, y) {
    var body = new URLSearchParams();
    body.set("siteMapNodeKey", key);
    body.set("siteMapNodeX", String(x));
    body.set("siteMapNodeY", String(y));
    var token = csrfToken();
    if (token) body.set("csrf_token", token);
    return fetch(action, {
      method: "POST",
      credentials: "same-origin",
      headers: {
        "Accept": "application/json",
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8",
      },
      body: body.toString(),
    }).catch(function () {});
  }

  function bindLayoutPersistence(root) {
    var action = layoutAction(root);
    if (!action || typeof fetch !== "function" || typeof URLSearchParams !== "function") return;
    var pending = Object.create(null);
    var timers = Object.create(null);
    function flush(key) {
      var position = pending[key];
      delete pending[key];
      if (!position) return;
      postLayoutPosition(action, key, position.x, position.y);
    }
    root.addEventListener("gosxstudio:sitemap-node-moved", function (event) {
      var detail = event && event.detail ? event.detail : {};
      var key = typeof detail.key === "string" ? detail.key.trim() : "";
      var x = finiteNumber(detail.x);
      var y = finiteNumber(detail.y);
      if (!key || x === null || y === null) return;
      pending[key] = { x: x, y: y };
      window.clearTimeout(timers[key]);
      timers[key] = window.setTimeout(function () {
        flush(key);
      }, 120);
    });
  }

  function bind(root) {
    if (!root || root[BOUND_PROP] === true) return root;
    root[BOUND_PROP] = true;
    setAttr(root, BOUND, "true");
    root.addEventListener("click", function (event) {
      handleClick(root, event);
    });
    root.addEventListener("change", function (event) {
      handleChange(root, event);
    });
    root.addEventListener("wheel", function (event) {
      handleWheel(root, event);
    }, { passive: false });
    root.addEventListener("pointerdown", function (event) {
      handlePointerDown(root, event);
    });
    root.addEventListener("keydown", function (event) {
      handleKeydown(root, event);
    });
    bindLayoutPersistence(root);
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
