;(function () {
  // GoSX Studio full-WASM CanvasBoard selection bridge (opt-in).
  //
  // When the opt-in Canvas2D site-map board is mounted (MUDDY_SITEMAP_CANVAS=1
  // / ?sitemap=canvas), a click on a canvas node hit-tests in the gosx
  // CanvasBoard pick path and writes the picked workspace node key into the
  // shared signal $surface.event.selectedID (gosx
  // client/bridge/bridge_canvasboard_event_full.go canvasBoardApplyPick →
  // store.Set, which fans out to JS subscribers via __gosx_notify_shared_signal).
  //
  // The DOM site-map board exposes a public runtime
  // (window.GoSXStudioSiteMapRuntime, gosx-studio
  // sitemapruntime/island_runtime.js) whose setState(root, { selectedNode })
  // is exactly what a DOM-board node click invokes: it writes
  // data-studio-site-map-selected-node on the board root, runs sync →
  // syncSelection → writeSelectionDetails (which populates the right-rail
  // inspector / selection detail card), and dispatches gosxstudio:site-map-change.
  //
  // This bridge subscribes to $surface.event.selectedID and feeds the picked
  // key into that SAME setState path, so picking a canvas node updates the
  // inspector identically to selecting a DOM-board node. It NEVER touches the
  // DOM-board markup or the canvas pick path — it only reads the canvas signal
  // and calls the DOM board runtime's public API. The canvas rect id IS the
  // workspace node key (gosx-studio sitemap_canvas.go: "id == node key"), the
  // same key the DOM board carries on data-studio-site-map-workspace-node, so
  // no key remapping is required.
  if (typeof window === "undefined") return;

  var BOARD = "[data-studio-site-map-board='true']";
  var SIGNAL = "$surface.event.selectedID";
  // Marquee multi-select signal (gosx bridge_canvasboard_event_full.go
  // canvasBoardApplyMarquee → $surface.event.selectedIDs, comma-joined ids). A
  // canvas shift-drag marquee writes this; a single pick / arrow-key nav writes
  // only selectedID, leaving selectedIDs untouched.
  var SIGNAL_IDS = "$surface.event.selectedIDs";
  var BOUND = "data-gosx-studio-canvas-selection-bridge-bound";
  var SELECTED_ATTR = "data-studio-site-map-selected-node";
  // Marks the most recent selection the bridge itself applied, so it never
  // re-applies an unchanged value and never fights the DOM board when the user
  // selects on the DOM board directly (then selectedID may be stale/empty).
  var lastApplied = null;
  // Tracks the most recent selectedIDs string the bridge applied so a marquee
  // multi-selection drives selectedNodes exactly once, and a later single pick
  // (which does not touch selectedIDs) is not mistaken for a marquee change.
  var lastAppliedIDs = null;
  var stopped = false;

  function boardRoot() {
    return document.querySelector(BOARD);
  }

  function runtime() {
    var rt = window.GoSXStudioSiteMapRuntime;
    return rt && typeof rt.setState === "function" ? rt : null;
  }

  // readSignalString returns the current value of a shared string signal, or ""
  // when unset / the getter is unavailable. The getter returns JSON, so a string
  // value comes back quoted (mirrors the canvas interaction e2e's readback).
  function readSignalString(name) {
    var get = window.__gosx_get_shared_signal;
    if (typeof get !== "function") return null;
    var raw;
    try {
      raw = get(name);
    } catch (e) {
      return null;
    }
    if (typeof raw !== "string") return "";
    try {
      var parsed = JSON.parse(raw);
      if (parsed == null) return "";
      return typeof parsed === "string" ? parsed : String(parsed);
    } catch (e) {
      return raw;
    }
  }

  // readSelectedID returns the current picked node key (primary selection).
  function readSelectedID() {
    return readSignalString(SIGNAL);
  }

  // readSelectedIDs returns the raw comma-joined marquee selection string ("" =
  // nothing / cleared). Parsing into a key list happens in apply.
  function readSelectedIDs() {
    return readSignalString(SIGNAL_IDS);
  }

  // parseKeyList splits a comma-joined id string into normalized node keys,
  // dropping empties. Mirrors the DOM board runtime's parseKeyList.
  function parseKeyList(value) {
    if (typeof value !== "string" || value === "") return [];
    var out = [];
    var parts = value.split(",");
    for (var i = 0; i < parts.length; i++) {
      var key = normalizeKey(parts[i].trim());
      if (key) out.push(key);
    }
    return out;
  }

  // Defensive: the canvas only picks rect nodes (whose id == the node key), but
  // strip a trailing ":label" just in case a label id ever arrives, so the key
  // always maps to the DOM board's workspace node.
  function normalizeKey(key) {
    if (typeof key !== "string") return "";
    if (key.length > 6 && key.slice(-6) === ":label") return key.slice(0, -6);
    return key;
  }

  // apply feeds the picked key into the DOM board's selection state via the
  // SAME public setState a DOM-board click uses. A single pick / arrow-key nav
  // is a single-selection, so it clears any prior marquee multi-selection.
  // Returns true when it ran.
  function apply(key) {
    var rt = runtime();
    var root = boardRoot();
    if (!rt || !root) return false;
    rt.setState(root, { selectedNode: key, selectedNodes: [] });
    return true;
  }

  // applyMulti feeds a marquee's node-key list into the DOM board's multi-select
  // state via the SAME public setState — the DOM board renders each as
  // data-studio-site-map-node-selected="true". The primary (first) key also
  // becomes selectedNode so the inspector / selection detail card reflects the
  // lead node, matching the canvas bridge's single-pick behavior. Returns true
  // when it ran.
  function applyMulti(keys) {
    var rt = runtime();
    var root = boardRoot();
    if (!rt || !root) return false;
    var primary = keys.length ? keys[0] : "";
    rt.setState(root, { selectedNode: primary, selectedNodes: keys });
    return true;
  }

  // tick reconciles the canvas selection signals into the DOM board.
  //
  // A marquee writes BOTH selectedIDs (comma list) and selectedID (primary) in
  // the same revision; a single pick / arrow-key nav writes ONLY selectedID.
  // So selectedIDs CHANGING is the reliable marquee discriminator: when it
  // changes this tick we drive selectedNodes (multi) and skip the single-select
  // branch (which would otherwise immediately clear the multi-selection). When
  // selectedIDs is unchanged, a changed non-empty selectedID drives single
  // selection. An empty signal (deselect / not-yet-picked) is left alone so the
  // bridge never clears a selection the DOM board made on its own.
  function tick() {
    if (stopped) return;
    var rawIDs = readSelectedIDs();
    // A marquee changed the multi-selection set this tick.
    if (rawIDs !== null && rawIDs !== lastAppliedIDs) {
      lastAppliedIDs = rawIDs;
      var keys = parseKeyList(rawIDs);
      // Only drive a non-empty marquee here. An empty selectedIDs that arrives
      // alongside a single pick is the marquee being superseded — let the
      // single-select branch below own that transition so a lone pick still
      // selects exactly one node.
      if (keys.length) {
        if (applyMulti(keys)) lastApplied = keys[0];
        return;
      }
    }
    var key = normalizeKey(readSelectedID());
    if (key && key !== lastApplied) {
      var root = boardRoot();
      var current = root ? root.getAttribute(SELECTED_ATTR) : null;
      if (key !== current) {
        if (apply(key)) lastApplied = key;
      } else {
        lastApplied = key;
      }
    }
  }

  // The pick signal is written from the gosx WASM side (store.Set →
  // __gosx_notify_shared_signal), which fires synchronously after a pick; we
  // poll the authoritative getter on a short interval (the same getter the
  // canvas interaction e2e trusts) so a missing/late JS subscriber registry is
  // never a dependency. Polling is cheap (one signal read) and idempotent.
  var timer = window.setInterval(tick, 120);

  // Re-sync once the runtime + canvas are up, and after structural fragment
  // refreshes re-bind the board (the DOM board re-binds on the same event).
  document.addEventListener("gosxstudio:fragments-refreshed", function () {
    lastApplied = null;
    lastAppliedIDs = null;
  });

  window.addEventListener("beforeunload", function () {
    stopped = true;
    if (timer) window.clearInterval(timer);
  });

  // Expose a tiny status hook so live e2e/status checks can assert the bridge
  // mounted (and trigger a synchronous reconcile) without reaching into the
  // closure.
  document.documentElement.setAttribute(BOUND, "true");
  var api = {
    tick: tick,
    selectedID: readSelectedID,
    selectedIDs: readSelectedIDs,
    applied: function () { return lastApplied; },
    appliedIDs: function () { return lastAppliedIDs; },
  };
  window.GoSXStudioCanvasSelectionBridgeRuntime = api;

  // Kick an initial reconcile in case a pick already landed before this script
  // ran (e.g. server-default selection echoed into the signal).
  tick();
})();
