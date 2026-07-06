(function () {
  "use strict";

  function ready(fn) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", fn, { once: true });
      return;
    }
    fn();
  }

  function number(value, fallback) {
    var parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : fallback;
  }

  function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function round(value) {
    return Math.round(value * 1000) / 1000;
  }

  function cssNumber(element, name, fallback) {
    if (!element || !window.getComputedStyle) return fallback;
    var value = window.getComputedStyle(element).getPropertyValue(name);
    return number(String(value || "").replace("px", ""), fallback);
  }

  function emit(canvas, name, detail) {
    canvas.dispatchEvent(new CustomEvent(name, { bubbles: true, detail: detail || {} }));
  }

  function selectorValue(value) {
    if (window.CSS && CSS.escape) return CSS.escape(String(value || ""));
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, "\\\"");
  }

  function initCanvas(canvas) {
    if (!canvas || canvas.dataset.gosxStudioSiteCanvasBound === "true") return;
    canvas.dataset.gosxStudioSiteCanvasBound = "true";
    var viewport = canvas.querySelector("[data-gosx-studio-canvas-viewport]");
    var surface = canvas.querySelector("[data-gosx-studio-canvas-surface]");
    if (!viewport || !surface) return;

    var state = {
      zoom: number(canvas.getAttribute("data-gosx-studio-canvas-zoom"), 1),
      panX: number(canvas.getAttribute("data-gosx-studio-canvas-pan-x"), 0),
      panY: number(canvas.getAttribute("data-gosx-studio-canvas-pan-y"), 0),
      selected: canvas.getAttribute("data-gosx-studio-canvas-selected") || ""
    };

    function nodes() {
      return Array.prototype.slice.call(canvas.querySelectorAll("[data-gosx-studio-canvas-node]"));
    }

    function edges() {
      return Array.prototype.slice.call(canvas.querySelectorAll("[data-gosx-studio-canvas-edge]"));
    }

    function nodeRect(node) {
      var x = cssNumber(node, "--gosx-studio-node-x", number(node.getAttribute("data-gosx-studio-canvas-node-x"), node.offsetLeft || 0));
      var y = cssNumber(node, "--gosx-studio-node-y", number(node.getAttribute("data-gosx-studio-canvas-node-y"), node.offsetTop || 0));
      var width = cssNumber(node, "--gosx-studio-node-width", node.offsetWidth || 240);
      var height = cssNumber(node, "--gosx-studio-node-height", node.offsetHeight || 132);
      return { x: x, y: y, width: width, height: height };
    }

    function nodeByKey(key) {
      if (!key) return null;
      if (window.CSS && CSS.escape) return canvas.querySelector("[data-gosx-studio-canvas-node=\"" + CSS.escape(key) + "\"]");
      return nodes().filter(function (node) {
        return node.getAttribute("data-gosx-studio-canvas-node") === key;
      })[0] || null;
    }

    function selectedNode() {
      return nodeByKey(state.selected);
    }

    function positionFields(key, axis) {
      return Array.prototype.slice.call(canvas.querySelectorAll(
        "[data-gosx-studio-canvas-node-position-key=\"" + selectorValue(key) + "\"][data-gosx-studio-canvas-node-position-axis=\"" + selectorValue(axis) + "\"]"
      ));
    }

    function notifyField(field) {
      field.dispatchEvent(new Event("input", { bubbles: true }));
      field.dispatchEvent(new Event("change", { bubbles: true }));
    }

    function syncPositionFields(detail, notify) {
      if (!detail || !detail.key) return;
      [["x", detail.x], ["y", detail.y]].forEach(function (entry) {
        positionFields(detail.key, entry[0]).forEach(function (field) {
          var value = String(round(entry[1]));
          if (field.value !== value) field.value = value;
          field.setAttribute("value", value);
          if (notify) notifyField(field);
        });
      });
    }

    function emitEditorTransaction(kind, detail) {
      var payload = {};
      Object.keys(detail || {}).forEach(function (key) {
        payload[key] = detail[key];
      });
      payload.kind = kind;
      emit(canvas, "gosxstudio:editor-transaction", payload);
    }

    function contentBounds() {
      var minX = 0;
      var minY = 0;
      var maxX = number(surface.style.getPropertyValue("--gosx-studio-canvas-width").replace("px", ""), surface.offsetWidth || 960);
      var maxY = number(surface.style.getPropertyValue("--gosx-studio-canvas-height").replace("px", ""), surface.offsetHeight || 640);
      nodes().forEach(function (node) {
        var rect = nodeRect(node);
        minX = Math.min(minX, rect.x);
        minY = Math.min(minY, rect.y);
        maxX = Math.max(maxX, rect.x + rect.width + 96);
        maxY = Math.max(maxY, rect.y + rect.height + 96);
      });
      return { minX: minX, minY: minY, width: Math.max(1, maxX - minX), height: Math.max(1, maxY - minY) };
    }

    function updateEdges() {
      edges().forEach(function (edge) {
        var from = nodeByKey(edge.getAttribute("data-gosx-studio-canvas-edge-from") || "");
        var to = nodeByKey(edge.getAttribute("data-gosx-studio-canvas-edge-to") || "");
        if (!from || !to) return;
        var fromRect = nodeRect(from);
        var toRect = nodeRect(to);
        var x1 = fromRect.x + fromRect.width;
        var y1 = fromRect.y + fromRect.height / 2;
        var x2 = toRect.x;
        var y2 = toRect.y + toRect.height / 2;
        if (toRect.x < fromRect.x) {
          x1 = fromRect.x;
          x2 = toRect.x + toRect.width;
        }
        edge.setAttribute("d", "M " + round(x1) + " " + round(y1) + " C " + round(x1 + 80) + " " + round(y1) + ", " + round(x2 - 80) + " " + round(y2) + ", " + round(x2) + " " + round(y2));
      });
    }

    function apply(reason) {
      state.zoom = clamp(state.zoom, 0.25, 2.8);
      surface.style.setProperty("--gosx-studio-canvas-pan-x", state.panX + "px");
      surface.style.setProperty("--gosx-studio-canvas-pan-y", state.panY + "px");
      surface.style.setProperty("--gosx-studio-canvas-zoom", state.zoom);
      canvas.setAttribute("data-gosx-studio-canvas-zoom", String(state.zoom));
      canvas.setAttribute("data-gosx-studio-canvas-pan-x", String(Math.round(state.panX)));
      canvas.setAttribute("data-gosx-studio-canvas-pan-y", String(Math.round(state.panY)));
      emit(canvas, "gosxstudio:canvas-viewport", {
        zoom: state.zoom,
        panX: state.panX,
        panY: state.panY,
        reason: reason || ""
      });
    }

    function nodeDetail(node) {
      var rect = nodeRect(node);
      return {
        key: node.getAttribute("data-gosx-studio-canvas-node") || "",
        kind: node.getAttribute("data-gosx-studio-canvas-node-kind") || "",
        label: node.getAttribute("data-gosx-studio-canvas-node-label") || node.textContent.trim(),
        href: node.getAttribute("data-gosx-studio-canvas-node-href") || "",
        x: round(rect.x),
        y: round(rect.y),
        width: round(rect.width),
        height: round(rect.height)
      };
    }

    function syncSelection(detail) {
      var selected = detail || { key: "", kind: "", label: "", href: "" };
      canvas.querySelectorAll("[data-gosx-studio-canvas-selection-detail]").forEach(function (target) {
        target.dispatchEvent(new CustomEvent("gosxstudio:canvas-selection-detail", { bubbles: true, detail: selected }));
      });
      ["key", "kind", "label", "href", "x", "y", "width", "height"].forEach(function (field) {
        canvas.querySelectorAll("[data-gosx-studio-canvas-selection-" + field + "]").forEach(function (target) {
          var value = selected[field] == null ? "" : String(selected[field]);
          if ("value" in target && target.tagName !== "OUTPUT") target.value = value;
          else target.textContent = value;
          target.setAttribute("data-gosx-studio-canvas-selection-value", value);
        });
      });
    }

    function select(node, reason) {
      nodes().forEach(function (candidate) {
        var active = candidate === node;
        candidate.classList.toggle("is-selected", active);
        candidate.setAttribute("aria-pressed", active ? "true" : "false");
      });
      var detail = nodeDetail(node);
      state.selected = detail.key;
      canvas.setAttribute("data-gosx-studio-canvas-selected", state.selected);
      detail.reason = reason || "";
      syncSelection(detail);
      emit(canvas, "gosxstudio:canvas-select", detail);
    }

    function clearSelection(reason) {
      nodes().forEach(function (node) {
        node.classList.remove("is-selected");
        node.setAttribute("aria-pressed", "false");
      });
      state.selected = "";
      canvas.removeAttribute("data-gosx-studio-canvas-selected");
      syncSelection({ key: "", reason: reason || "clear" });
      emit(canvas, "gosxstudio:canvas-select", { key: "", reason: reason || "clear" });
    }

    function centerNode(node, reason) {
      if (!node) return false;
      var viewportRect = viewport.getBoundingClientRect();
      var rect = nodeRect(node);
      state.panX = Math.round(viewportRect.width / 2 - (rect.x + rect.width / 2) * state.zoom);
      state.panY = Math.round(viewportRect.height / 2 - (rect.y + rect.height / 2) * state.zoom);
      apply(reason || "center-node");
      return true;
    }

    function selectAdjacent(delta, reason) {
      var all = nodes();
      if (!all.length) return false;
      var current = selectedNode();
      var index = current ? all.indexOf(current) : delta > 0 ? -1 : 0;
      var next = all[(index + delta + all.length) % all.length];
      select(next, reason || "command");
      centerNode(next, "select-node");
      emitAction(delta > 0 ? "select-next-node" : "select-previous-node", nodeDetail(next));
      return true;
    }

    function fit(reason) {
      var rect = viewport.getBoundingClientRect();
      var bounds = contentBounds();
      var padding = number(canvas.getAttribute("data-gosx-studio-canvas-fit-padding"), 48);
      var availableWidth = Math.max(1, rect.width - padding * 2);
      var availableHeight = Math.max(1, rect.height - padding * 2);
      state.zoom = clamp(Math.min(availableWidth / bounds.width, availableHeight / bounds.height), 0.25, 1.4);
      state.panX = Math.round((rect.width - bounds.width * state.zoom) / 2 - bounds.minX * state.zoom);
      state.panY = Math.round((rect.height - bounds.height * state.zoom) / 2 - bounds.minY * state.zoom);
      apply(reason || "fit");
    }

    function zoomAt(clientX, clientY, nextZoom, reason) {
      var rect = viewport.getBoundingClientRect();
      var x = clientX - rect.left;
      var y = clientY - rect.top;
      var worldX = (x - state.panX) / state.zoom;
      var worldY = (y - state.panY) / state.zoom;
      state.zoom = clamp(nextZoom, 0.25, 2.8);
      state.panX = x - worldX * state.zoom;
      state.panY = y - worldY * state.zoom;
      apply(reason || "zoom-at");
    }

    function zoomBy(delta, reason, anchor) {
      if (anchor && Number.isFinite(anchor.clientX) && Number.isFinite(anchor.clientY)) {
        zoomAt(anchor.clientX, anchor.clientY, state.zoom + delta, reason || "zoom");
        return;
      }
      state.zoom = clamp(state.zoom + delta, 0.25, 2.8);
      apply(reason || "zoom");
    }

    function setZoom(value, reason) {
      var text = String(value || "").toLowerCase();
      if (text === "fit") {
        fit(reason || "workbench-zoom");
        return;
      }
      if (text.indexOf("%") > 0) text = text.replace("%", "");
      var scale = number(text, 0);
      if (scale > 10) scale = scale / 100;
      if (scale <= 0) return;
      state.zoom = clamp(scale, 0.25, 2.8);
      apply(reason || "workbench-zoom");
    }

    function moveNode(node, x, y, reason) {
      node.style.setProperty("--gosx-studio-node-x", x + "px");
      node.style.setProperty("--gosx-studio-node-y", y + "px");
      node.setAttribute("data-gosx-studio-canvas-node-x", String(round(x)));
      node.setAttribute("data-gosx-studio-canvas-node-y", String(round(y)));
      updateEdges();
      var detail = nodeDetail(node);
      detail.reason = reason || "move";
      syncPositionFields(detail, false);
      emit(canvas, "gosxstudio:canvas-node-move", detail);
      if (state.selected === detail.key) syncSelection(detail);
      return detail;
    }

    function nudgeNode(node, dx, dy, reason, action) {
      if (!node || (!dx && !dy)) return false;
      var rect = nodeRect(node);
      var detail = moveNode(node, rect.x + dx, rect.y + dy, reason || "nudge");
      detail.moved = true;
      detail.fromX = round(rect.x);
      detail.fromY = round(rect.y);
      detail.toX = detail.x;
      detail.toY = detail.y;
      syncPositionFields(detail, true);
      emit(canvas, "gosxstudio:canvas-node-moved", detail);
      emitEditorTransaction("move-node", detail);
      emitAction(action || "nudge-node", detail);
      return true;
    }

    function keyboardNudge(event) {
      var node = selectedNode();
      if (!node) return false;
      var step = number(canvas.getAttribute("data-gosx-studio-canvas-keyboard-nudge"), 8);
      if (event.altKey) step = 1;
      if (event.shiftKey) step *= 10;
      var dx = event.key === "ArrowLeft" ? -step : event.key === "ArrowRight" ? step : 0;
      var dy = event.key === "ArrowUp" ? -step : event.key === "ArrowDown" ? step : 0;
      return nudgeNode(node, dx, dy, "keyboard-nudge", "nudge-node");
    }

    function openSelected(reason) {
      var node = selectedNode();
      if (!node) return false;
      var detail = nodeDetail(node);
      detail.reason = reason || "command";
      emit(canvas, "gosxstudio:canvas-open", detail);
      emitAction("open-node", detail);
      return true;
    }

    function handleCanvasCommand(detail) {
      detail = detail || {};
      if (detail.kind !== "canvas") return false;
      var target = detail.target || detail.key || "";
      var step = number(canvas.getAttribute("data-gosx-studio-canvas-keyboard-nudge"), 8);
      if (target === "fit" || target === "reset") {
        fit("command");
        emitAction("fit", { reason: "command" });
        return true;
      }
      if (target === "select-next") return selectAdjacent(1, "command");
      if (target === "select-previous") return selectAdjacent(-1, "command");
      if (target === "center-selected") {
        if (!centerNode(selectedNode(), "command")) return false;
        emitAction("center-node", { key: state.selected, reason: "command" });
        return true;
      }
      if (target === "open-selected") return openSelected("command");
      if (target === "clear-selection") {
        clearSelection("command");
        return true;
      }
      if (target === "nudge-left") return nudgeNode(selectedNode(), -step, 0, "command-nudge", "nudge-node");
      if (target === "nudge-right") return nudgeNode(selectedNode(), step, 0, "command-nudge", "nudge-node");
      if (target === "nudge-up") return nudgeNode(selectedNode(), 0, -step, "command-nudge", "nudge-node");
      if (target === "nudge-down") return nudgeNode(selectedNode(), 0, step, "command-nudge", "nudge-node");
      return false;
    }

    function emitAction(action, detail) {
      var payload = {};
      Object.keys(detail || {}).forEach(function (key) {
        payload[key] = detail[key];
      });
      payload.action = action;
      emit(canvas, "gosxstudio:canvas-action", payload);
    }

    var panDrag = null;
    var nodeDrag = null;
    viewport.addEventListener("pointerdown", function (event) {
      var node = event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-node]");
      if (node && canvas.contains(node)) {
        var rect = nodeRect(node);
        nodeDrag = {
          node: node,
          x: event.clientX,
          y: event.clientY,
          nodeX: rect.x,
          nodeY: rect.y,
          moved: false
        };
        select(node, "pointer");
        node.setAttribute("data-gosx-studio-canvas-node-dragging", "true");
        viewport.setPointerCapture(event.pointerId);
        return;
      }
      if (event.target && event.target.closest && event.target.closest("button, a, input, textarea, select")) return;
      panDrag = {
        x: event.clientX,
        y: event.clientY,
        panX: state.panX,
        panY: state.panY
      };
      viewport.setPointerCapture(event.pointerId);
      viewport.setAttribute("data-gosx-studio-canvas-dragging", "true");
    });
    viewport.addEventListener("pointermove", function (event) {
      var rect = viewport.getBoundingClientRect();
      emit(canvas, "gosxstudio:canvas-cursor", {
        clientX: event.clientX,
        clientY: event.clientY,
        x: (event.clientX - rect.left - state.panX) / state.zoom,
        y: (event.clientY - rect.top - state.panY) / state.zoom,
        selected: state.selected
      });
      if (nodeDrag) {
        nodeDrag.moved = true;
        moveNode(nodeDrag.node, nodeDrag.nodeX + (event.clientX - nodeDrag.x) / state.zoom, nodeDrag.nodeY + (event.clientY - nodeDrag.y) / state.zoom, "drag");
        return;
      }
      if (!panDrag) return;
      state.panX = panDrag.panX + event.clientX - panDrag.x;
      state.panY = panDrag.panY + event.clientY - panDrag.y;
      apply("pan");
    });
    viewport.addEventListener("pointerup", function (event) {
      if (nodeDrag) {
        var detail = nodeDetail(nodeDrag.node);
        detail.reason = "drag-end";
        detail.moved = nodeDrag.moved;
        detail.fromX = round(nodeDrag.nodeX);
        detail.fromY = round(nodeDrag.nodeY);
        detail.toX = detail.x;
        detail.toY = detail.y;
        nodeDrag.node.removeAttribute("data-gosx-studio-canvas-node-dragging");
        syncPositionFields(detail, nodeDrag.moved);
        nodeDrag = null;
        emit(canvas, "gosxstudio:canvas-node-moved", detail);
        if (detail.moved) emitEditorTransaction("move-node", detail);
        emitAction("move-node", detail);
      }
      if (panDrag) panDrag = null;
      viewport.removeAttribute("data-gosx-studio-canvas-dragging");
      try {
        viewport.releasePointerCapture(event.pointerId);
      } catch (error) {
        return;
      }
    });
    viewport.addEventListener("wheel", function (event) {
      event.preventDefault();
      if (event.ctrlKey || event.metaKey) {
        zoomBy(event.deltaY > 0 ? -0.08 : 0.08, "wheel", { clientX: event.clientX, clientY: event.clientY });
        return;
      }
      state.panX -= event.deltaX;
      state.panY -= event.deltaY;
      apply("wheel-pan");
    }, { passive: false });

    canvas.addEventListener("click", function (event) {
      var node = event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-node]");
      if (node && canvas.contains(node)) select(node, "click");
      if (event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-reset]")) {
        fit("reset");
        emitAction("fit", { reason: "button" });
      }
      if (event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-zoom-in]")) {
        zoomBy(0.12, "button");
        emitAction("zoom-in", { zoom: state.zoom, reason: "button" });
      }
      if (event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-zoom-out]")) {
        zoomBy(-0.12, "button");
        emitAction("zoom-out", { zoom: state.zoom, reason: "button" });
      }
      var action = event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-action]");
      if (action && canvas.contains(action)) {
        emitAction(action.getAttribute("data-gosx-studio-canvas-action") || "custom", { reason: "click" });
      }
    });
    canvas.addEventListener("dblclick", function (event) {
      var node = event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-node]");
      if (!node || !canvas.contains(node)) return;
      var detail = nodeDetail(node);
      emit(canvas, "gosxstudio:canvas-open", detail);
      emitAction("open-node", detail);
    });
    canvas.addEventListener("keydown", function (event) {
      if (event.key === "+" || event.key === "=") {
        event.preventDefault();
        zoomBy(0.12, "keyboard");
        emitAction("zoom-in", { zoom: state.zoom, reason: "keyboard" });
      } else if (event.key === "-") {
        event.preventDefault();
        zoomBy(-0.12, "keyboard");
        emitAction("zoom-out", { zoom: state.zoom, reason: "keyboard" });
      } else if (event.key === "0") {
        event.preventDefault();
        fit("reset");
        emitAction("fit", { reason: "keyboard" });
      } else if (event.key === "ArrowLeft" || event.key === "ArrowRight" || event.key === "ArrowUp" || event.key === "ArrowDown") {
        event.preventDefault();
        if (keyboardNudge(event)) return;
        state.panX += event.key === "ArrowLeft" ? 42 : event.key === "ArrowRight" ? -42 : 0;
        state.panY += event.key === "ArrowUp" ? 42 : event.key === "ArrowDown" ? -42 : 0;
        apply("keyboard-pan");
      } else if (event.key === "Escape") {
        clearSelection("clear");
      }
    });
    canvas.addEventListener("input", function (event) {
      var field = event.target && event.target.closest && event.target.closest("[data-gosx-studio-canvas-node-position]");
      if (!field || !canvas.contains(field)) return;
      var key = field.getAttribute("data-gosx-studio-canvas-node-position-key") || "";
      var node = nodeByKey(key);
      if (!node) return;
      var rect = nodeRect(node);
      var xFields = positionFields(key, "x");
      var yFields = positionFields(key, "y");
      var x = xFields.length ? number(xFields[0].value, rect.x) : rect.x;
      var y = yFields.length ? number(yFields[0].value, rect.y) : rect.y;
      moveNode(node, x, y, "field");
    });
    document.addEventListener("gosxstudio:workbench-zoom", function (event) {
      var detail = event.detail || {};
      if (detail.form && detail.form.contains && !detail.form.contains(canvas)) return;
      setZoom(detail.zoom || detail.scale, detail.reason || "workbench");
    });
    var ownerForm = canvas.closest && canvas.closest("form");
    if (ownerForm) {
      ownerForm.addEventListener("gosxstudio:command", function (event) {
        if (event.defaultPrevented) return;
        if (handleCanvasCommand(event.detail || {})) event.preventDefault();
      });
    }
    updateEdges();
    if (state.selected) {
      var selected = nodeByKey(state.selected);
      if (selected) select(selected, "init");
    }
    apply("init");
  }

  function initAll(root) {
    var scope = root && root.querySelectorAll ? root : document;
    scope.querySelectorAll("[data-gosx-studio-site-canvas]").forEach(initCanvas);
  }

  ready(function () { initAll(document); });
  document.addEventListener("gosx:navigate", function () { initAll(document); });
  document.addEventListener("gosx:render", function () { initAll(document); });
})();
