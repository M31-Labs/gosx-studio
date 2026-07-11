(function () {
  "use strict";
  var runtimes = [];
  function qs(root, selector) { return (root || document).querySelector(selector); }
  function qsa(root, selector) { return Array.prototype.slice.call((root || document).querySelectorAll(selector)); }
  function socketURL(value) {
    var url = new URL(value, window.location.href);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    return url.toString();
  }
  function canonicalRoute(value) {
    try { return new URL(value || "/", window.location.origin).pathname || "/"; } catch (_) { return String(value || "/").split(/[?#]/)[0] || "/"; }
  }
  function activeRouteNode() {
    return qs(document, "[data-studio-preview-route-current], [data-studio-preview-route][aria-pressed='true'], [data-studio-preview-frame]");
  }
  function currentRoute() {
    var node = activeRouteNode();
    var value = node && (node.getAttribute("data-studio-preview-route-current") || node.getAttribute("data-studio-preview-route"));
    return canonicalRoute(value || window.location.pathname || "/");
  }
  function currentPageID() {
    var node = qs(document, "[data-studio-preview-route][aria-pressed='true']");
    var key = node && node.getAttribute("data-studio-preview-route-key");
    return key ? (key.indexOf("page:") === 0 ? key : "page:" + key) : "";
  }
  function currentViewport() {
    var node = qs(document, "[data-studio-preview-viewport][aria-pressed='true'], [data-studio-viewport][aria-pressed='true']");
    return (node && (node.getAttribute("data-studio-preview-viewport") || node.getAttribute("data-studio-viewport"))) || "desktop";
  }
  function stableSelection(detail) {
    detail = detail || {};
    var selection = {
      route: canonicalRoute(detail.route || detail.targetRoute || currentRoute()),
      pageId: detail.pageId || detail.pageID || currentPageID(),
      field: detail.field || "",
      blockKey: detail.blockKey || detail.componentKey || "",
      nodeId: detail.nodeId || detail.nodeID || "",
      kind: detail.kind || "",
      viewport: detail.viewport || currentViewport()
    };
    return selection.route && (selection.field || selection.blockKey || selection.nodeId || selection.pageId) ? selection : null;
  }
  function storageGet(key) { try { return sessionStorage.getItem(key); } catch (_) { return null; } }
  function storageSet(key, value) { try { sessionStorage.setItem(key, value); } catch (_) {} }
  function create(panel) {
    var endpoint = panel.getAttribute("data-studio-collab-endpoint") || "";
    var resource = panel.getAttribute("data-studio-collab-resource") || "default";
    var status = qs(panel, "[data-studio-collab-status]");
    var live = qs(panel, "[data-studio-collab-live]");
    var facepile = qs(panel, "[data-studio-collab-facepile]");
    var conflict = qs(panel, "[data-studio-collab-conflict]");
    var socket = null, connectionID = "", retry = 0, reconnectTimer = 0, stopped = false, lastCursor = 0, pending = {}, synced = false, serverPermissions = {};
    var sequenceKey = "gosxstudio:collab-sequence:" + resource;
    var headsKey = "gosxstudio:collab-heads:" + resource;
    var acceptedSequence = Number(storageGet(sequenceKey) || "0") || 0;
    var heads = {}; try { heads = JSON.parse(storageGet(headsKey) || "{}") || {}; } catch (_) { heads = {}; }
    function targetKey(target) { target = target || {}; return [target.route || "/", target.pageId || "", target.field || "", target.componentKey || target.blockKey || "", target.nodeId || "", target.property || "", target.breakpoint || "base", target.state || "default"].join("|"); }
    function storeHeads() { storageSet(headsKey, JSON.stringify(heads)); }
    function setStatus(value, message) {
      panel.setAttribute("data-studio-collab-state", value);
      if (status) status.textContent = message;
    }
    function announce(message) { if (live) live.textContent = message || ""; }
    function send(event, data) {
      if (!socket || socket.readyState !== WebSocket.OPEN) return false;
      socket.send(JSON.stringify({ event: event, data: data || {} }));
      return true;
    }
    function permissions(value) {
      value = value || {};
      panel.setAttribute("data-studio-collab-can-author", value.author ? "true" : "false");
      panel.setAttribute("data-studio-collab-can-design", value.design ? "true" : "false");
      qsa(document, "[data-studio-requires-author]").forEach(function (node) { node.disabled = !value.author; node.setAttribute("aria-disabled", value.author ? "false" : "true"); });
      qsa(document, "[data-studio-requires-design]").forEach(function (node) { node.disabled = !value.design; node.setAttribute("aria-disabled", value.design ? "false" : "true"); });
      qsa(document, "[data-gosx-studio-operation-kind]").forEach(function (node) { var design = !!node.getAttribute("data-gosx-studio-style-property") || !!node.closest("[data-studio-layout-control]"); var allowed = design ? !!value.design : !!value.author; node.disabled = !allowed; node.setAttribute("aria-disabled", allowed ? "false" : "true"); });
    }
    function initials(name) {
      return String(name || "?").trim().split(/\s+/).slice(0, 2).map(function (part) { return part.charAt(0).toUpperCase(); }).join("") || "?";
    }
    function renderPresence(payload) {
      var members = (payload && payload.members) || [];
      if (facepile) {
        facepile.textContent = "";
        members.forEach(function (member) {
          var item = document.createElement("li"), avatar = document.createElement("span");
          item.setAttribute("data-studio-collaborator", member.connectionId || "");
          avatar.className = "studio-collaboration-avatar";
          avatar.textContent = initials(member.displayName);
          avatar.title = member.displayName || "Collaborator";
          avatar.setAttribute("aria-label", member.displayName || "Collaborator");
          item.appendChild(avatar); facepile.appendChild(item);
        });
      }
      var liveIDs = {};
      members.forEach(function (member) { liveIDs[member.connectionId] = true; });
      qsa(document, "[data-studio-remote-cursor]").forEach(function (node) { if (!liveIDs[node.getAttribute("data-studio-remote-cursor")]) node.remove(); });
      qsa(document, "[data-studio-remote-selection-source]").forEach(function (node) { if (!liveIDs[node.getAttribute("data-studio-remote-selection-source")]) { node.removeAttribute("data-studio-remote-selection-source"); node.removeAttribute("data-studio-remote-route"); node.removeAttribute("data-studio-remote-viewport"); node.classList.remove("studio-remote-selection"); } });
      announce(members.length + (members.length === 1 ? " collaborator connected" : " collaborators connected"));
    }
    function scoped(route, viewport) { return (!route || route === currentRoute()) && (!viewport || viewport === currentViewport()); }
    function renderCursor(payload) {
      if (!payload || !payload.source || payload.source.connectionId === connectionID || !scoped(payload.cursor && payload.cursor.route, payload.cursor && payload.cursor.viewport)) return;
      var id = payload.source.connectionId, node = qs(document, "[data-studio-remote-cursor='" + CSS.escape(id) + "']");
      if (!node) { node = document.createElement("span"); node.className = "studio-remote-cursor"; node.setAttribute("data-studio-remote-cursor", id); node.setAttribute("aria-hidden", "true"); document.body.appendChild(node); }
      node.style.left = Math.round((payload.cursor.x || 0) * window.innerWidth) + "px";
      node.style.top = Math.round((payload.cursor.y || 0) * window.innerHeight) + "px";
      node.title = payload.source.displayName || "Collaborator";
      node.setAttribute("data-studio-remote-route", payload.cursor.route || "");
      node.setAttribute("data-studio-remote-viewport", payload.cursor.viewport || "");
    }
    function locatorSelector(selection) {
      if (selection.field) return "[data-studio-field='" + CSS.escape(selection.field) + "']";
      if (selection.blockKey) return "[data-studio-block-key='" + CSS.escape(selection.blockKey) + "'],[data-block-studio-block='" + CSS.escape(selection.blockKey) + "']";
      if (selection.nodeId) return "[data-studio-node-id='" + CSS.escape(selection.nodeId) + "']";
      if (selection.pageId) return "[data-studio-page-id='" + CSS.escape(selection.pageId) + "']";
      return "";
    }
    function renderSelection(payload) {
      if (!payload || !payload.source || payload.source.connectionId === connectionID || !scoped(payload.selection && payload.selection.route, payload.selection && payload.selection.viewport)) return;
      qsa(document, "[data-studio-remote-selection-source='" + CSS.escape(payload.source.connectionId) + "']").forEach(function (node) { node.removeAttribute("data-studio-remote-selection-source"); node.removeAttribute("data-studio-remote-route"); node.removeAttribute("data-studio-remote-viewport"); node.classList.remove("studio-remote-selection"); });
      var selector = locatorSelector(payload.selection || {});
      if (selector) qsa(document, selector).forEach(function (node) { node.classList.add("studio-remote-selection"); node.setAttribute("data-studio-remote-selection-source", payload.source.connectionId); node.setAttribute("data-studio-remote-route", payload.selection.route || ""); node.setAttribute("data-studio-remote-viewport", payload.selection.viewport || ""); });
      document.dispatchEvent(new CustomEvent("gosxstudio:remote-selection", { detail: payload }));
      announce((payload.source.displayName || "A collaborator") + " selected " + ((payload.selection && (payload.selection.field || payload.selection.blockKey || payload.selection.pageId)) || "an element"));
    }
    function pruneRemote() {
      qsa(document, "[data-studio-remote-route]").forEach(function (node) {
        if (!scoped(node.getAttribute("data-studio-remote-route"), node.getAttribute("data-studio-remote-viewport"))) {
          if (node.hasAttribute("data-studio-remote-cursor")) node.remove();
          else { node.removeAttribute("data-studio-remote-selection-source"); node.removeAttribute("data-studio-remote-route"); node.removeAttribute("data-studio-remote-viewport"); node.classList.remove("studio-remote-selection"); }
        }
      });
    }
    function accepted(ack) {
      var sequence = Number(ack && ack.sequence || 0);
      if (sequence > acceptedSequence) { acceptedSequence = sequence; storageSet(sequenceKey, String(sequence)); }
      document.dispatchEvent(new CustomEvent("gosxstudio:collaboration-operation-accepted", { detail: ack || {} }));
      var operationID = ack && ack.record && ack.record.id;
      if (ack && ack.record && ack.record.target) { heads[targetKey(ack.record.target)] = ack.record.targetHead || ""; storeHeads(); }
      if (operationID && pending[operationID]) { pending[operationID].resolve(ack); clearTimeout(pending[operationID].timer); delete pending[operationID]; }
    }
    function rejected(error) {
      if (conflict) { conflict.hidden = false; conflict.textContent = (error && error.message) || "The shared edit was rejected."; }
      announce((error && error.message) || "The shared edit was rejected.");
      document.dispatchEvent(new CustomEvent("gosxstudio:collaboration-operation-rejected", { detail: error || {} }));
      var operationID = error && error.operationId;
      if (operationID && pending[operationID]) { if (error.currentHead !== undefined && pending[operationID].request) { heads[targetKey(pending[operationID].request.target)] = error.currentHead || ""; storeHeads(); } var failure = new Error(error.message || "Shared edit rejected"); failure.collaborationError = error; pending[operationID].reject(failure); clearTimeout(pending[operationID].timer); delete pending[operationID]; }
    }
    function rejectPending(message) { Object.keys(pending).forEach(function (id) { clearTimeout(pending[id].timer); pending[id].reject(new Error(message)); delete pending[id]; }); }
    function message(event) {
      var envelope; try { envelope = JSON.parse(event.data); } catch (_) { return; }
      var data = envelope.data || {};
      if (envelope.event === "studio.hello") { connectionID = data.connectionId || ""; synced = false; serverPermissions = data.permissions || {}; permissions({}); retry = 0; setStatus("syncing", "Syncing…"); send("studio.sync.resume", { afterSequence: acceptedSequence }); }
      else if (envelope.event === "studio.presence") renderPresence(data);
      else if (envelope.event === "studio.cursors") renderCursor(data);
      else if (envelope.event === "studio.selection") renderSelection(data);
      else if (envelope.event === "studio.operation.accepted") accepted(data);
      else if (envelope.event === "studio.operation.rejected") rejected(data);
      else if (envelope.event === "studio.sync.tail") { (data.operations || []).forEach(accepted); if (data.hasMore) send("studio.sync.resume", { afterSequence: acceptedSequence }); else { synced = true; permissions(serverPermissions); setStatus("connected", "Live"); } }
      else if (envelope.event === "__crdt_error") rejected({ message: "Binary collaboration is not supported." });
    }
    function connect() {
      if (stopped || !endpoint) return;
      clearTimeout(reconnectTimer); setStatus("connecting", "Connecting…");
      socket = new WebSocket(socketURL(endpoint));
      socket.addEventListener("message", message);
      socket.addEventListener("close", function () { connectionID = ""; synced = false; permissions({}); rejectPending("Collaboration connection closed"); setStatus("reconnecting", "Reconnecting…"); if (!stopped) reconnectTimer = setTimeout(connect, Math.min(5000, 500 * Math.pow(2, retry++))); });
      socket.addEventListener("error", function () { setStatus("error", "Connection interrupted"); });
    }
    function selectionEvent(event) { var selection = stableSelection(event.detail); if (selection) send("studio.selection.submit", selection); }
    function cursorEvent(event) {
      var now = Date.now(); if (now - lastCursor < 50) return;
      var canvas = event.target && event.target.closest && event.target.closest("[data-studio-canvas], [data-gosx-studio-canvas], [data-studio-preview-frame]"); if (!canvas) return;
      lastCursor = now; send("studio.cursor.submit", { route: currentRoute(), viewport: currentViewport(), x: Math.max(0, Math.min(1, event.clientX / Math.max(1, window.innerWidth))), y: Math.max(0, Math.min(1, event.clientY / Math.max(1, window.innerHeight))) });
    }
    document.addEventListener("gosxstudio:preview-select", selectionEvent);
    document.addEventListener("pointermove", cursorEvent, { passive: true });
    document.addEventListener("gosxstudio:preview-route-change", pruneRemote);
    document.addEventListener("gosxstudio:preview-viewport-change", pruneRemote);
    document.addEventListener("gosxstudio:collaboration-submit", function (event) { if (event.detail && event.detail.request) send("studio.operation.submit", { request: event.detail.request }); });
    var retryButton = qs(panel, "[data-studio-collab-reconnect]"); if (retryButton) retryButton.addEventListener("click", connect);
    function submit(request) {
      return new Promise(function (resolve, reject) {
        var id = request && request.id; if (!id || !send("studio.operation.submit", { request: request })) { reject(new Error("Collaboration is not connected")); return; }
        var timer = setTimeout(function () { if (pending[id]) { pending[id].reject(new Error("Shared edit timed out")); delete pending[id]; } }, 15000);
        pending[id] = { resolve: resolve, reject: reject, timer: timer, request: request };
      });
    }
    permissions({}); connect();
    return { submit: submit, expectedHead: function (request) { var key = targetKey(request && request.target); if (!Object.prototype.hasOwnProperty.call(heads, key)) { heads[key] = ""; storeHeads(); } return heads[key]; }, available: function () { return synced && !!connectionID && !!socket && socket.readyState === WebSocket.OPEN; }, resume: function () { synced = false; permissions({}); return send("studio.sync.resume", { afterSequence: acceptedSequence }); }, close: function () { stopped = true; synced = false; clearTimeout(reconnectTimer); rejectPending("Collaboration closed"); if (socket) socket.close(1000, "closed"); }, sequence: function () { return acceptedSequence; } };
  }
  function bind(root) {
    qsa(root || document, "[data-gosx-studio-collaboration='true']").forEach(function (panel) { if (panel.__gosxCollaborationRuntime) return; panel.__gosxCollaborationRuntime = create(panel); runtimes.push(panel.__gosxCollaborationRuntime); });
  }
  window.GoSXStudioCollaborationRuntime = { bind: bind, create: create, available: function () { return !!(runtimes.length && runtimes[0].available()); }, expectedHead: function (request) { return runtimes.length ? runtimes[0].expectedHead(request) : ""; }, submit: function (request) { return runtimes.length ? runtimes[0].submit(request) : Promise.reject(new Error("Collaboration is not mounted")); } };
  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", function () { bind(document); }, { once: true }); else bind(document);
})();
