;(function () {
  "use strict";

  // Studio-owned WASM-free Canvas2D site-map client.
  //
  // This is the payload payoff: it renders + drives the editor's Canvas2D
  // site-map board WITHOUT the gosx client WASM. It reads the server-precomputed
  // RenderBundle that ships inline (<script data-gosx-canvas-bundle>), maintains a
  // JS-side camera, paints via requestAnimationFrame using the ported painter
  // (window.GoSXStudioCanvas2DPainterRuntime), and owns all interaction
  // (pan/zoom/pick/marquee/nav) in pure JS - no __gosx_* calls, no signal poll. Selection flows
  // directly into the DOM board via window.GoSXStudioSiteMapRuntime.setState, the
  // SAME public API a DOM-board click uses, so the right-rail inspector / selection
  // detail card updates identically.
  //
  // The interaction math is a verbatim port of gosx's WASM bridge
  // (client/bridge/bridge_canvasboard_event_full.go) + the nav-cost algorithm
  // (client/vm/canvas_board_adapter.go NavFrom), so the WASM-free board behaves
  // identically to the WASM board:
  //   pan:   panX -= dxScreen/zoom;  panY += dyScreen/zoom
  //   zoom:  newZoom = clamp(zoom*factor), re-pin world point under cursor
  //   s→w:   worldX = (sx - cssW/2)/zoom + panX;  worldY = panY - (sy - cssH/2)/zoom
  //   pick:  AABB test vs objects[].bounds (topmost wins)
  //   nav:   nearest-in-direction, cost = primary + 2*perpendicular (half-plane gate)
  //
  // The canvas it owns is a host-rendered <canvas data-gosx-canvas-wasm-free="true">
  // that DELIBERATELY carries NO data-gosx-surface-kind, so the gosx bootstrap's
  // canvas discovery ([data-gosx-surface-kind]:not([data-gosx-engine-bytecode]))
  // never WASM-hydrates it. This script is the ONLY thing painting/handling it.
  if (typeof window === "undefined" || typeof document === "undefined") return;

  var CANVAS = "canvas[data-gosx-canvas-wasm-free='true']";
  var BUNDLE = "[data-gosx-canvas-bundle]";
  var WEBGPU_BUNDLE = "[data-gosx-canvas-webgpu-bundle]";
  var BOARD = "[data-studio-site-map-board='true']";
  var BOUND_FLAG = "data-gosx-canvas-wasm-free-bound";
  var WEBGPU_ROUTE_ATTR = "data-gosx-canvas-wasm-free-webgpu";
  var FALLBACK_ATTR = "data-gosx-canvas-wasm-free-fallback";
  var FALLBACK_REASON_ATTR = "data-gosx-canvas-wasm-free-fallback-reason";
  // Match gosx vm.ClampCanvasBoardZoom bounds so zoom feels identical to WASM.
  var ZOOM_MIN = 0.1;
  var ZOOM_MAX = 8;
  // Wheel zoom step per notch; matches the gosx bootstrap's wheel→factor mapping
  // (a positive deltaY zooms out). exp keeps zoom multiplicative + smooth.
  var WHEEL_ZOOM_BASE = 1.0015;
  // A drag shorter than this (CSS px) is treated as a click (pick), not a pan.
  var CLICK_SLOP = 4;
  // A `click` arriving within this window (ms) after a pan/marquee drag is the
  // browser's trailing synthetic click for that drag — ignore it so a drag never
  // also picks.
  var CLICK_AFTER_DRAG_MS = 320;

  function clampZoom(z) {
    if (!(z > 0)) return 1;
    if (z < ZOOM_MIN) return ZOOM_MIN;
    if (z > ZOOM_MAX) return ZOOM_MAX;
    return z;
  }

  function now() {
    return (typeof performance !== "undefined" && performance.now) ? performance.now() : Date.now();
  }

  // parseJSONScript reads an inline RenderBundle JSON sibling of the canvas.
  // Returns null when absent/invalid.
  function parseJSONScript(root, selector) {
    var el = root ? root.querySelector(selector) : document.querySelector(selector);
    if (!el) return null;
    var raw = (el.textContent || "").trim();
    if (!raw) return null;
    try {
      return JSON.parse(raw);
    } catch (e) {
      return null;
    }
  }

  function parseBundle(root) {
    return parseJSONScript(root, BUNDLE);
  }

  function parseWebGPUBundle(root) {
    return parseJSONScript(root, WEBGPU_BUNDLE);
  }

  // boardRoot resolves the DOM site-map board (the selection sink).
  function boardRoot() {
    return document.querySelector(BOARD);
  }

  function runtime() {
    var rt = window.GoSXStudioSiteMapRuntime;
    return rt && typeof rt.setState === "function" ? rt : null;
  }

  function painter() {
    var p = window.GoSXStudioCanvas2DPainterRuntime;
    return p && typeof p.paint === "function" ? p : null;
  }

  function webgpuAPI() {
    var api = window.__gosx_scene3d_webgpu_api;
    return api && typeof api.createRenderer === "function" ? api : null;
  }

  // normalizeKey strips a trailing ":label" so a label id maps to its rect's
  // node key (the DOM board's workspace node key). Mirrors the bridge.
  function normalizeKey(key) {
    if (typeof key !== "string") return "";
    if (key.length > 6 && key.slice(-6) === ":label") return key.slice(0, -6);
    return key;
  }

  // boundsOf reads an object's world-space AABB, tolerant of missing fields.
  function boundsOf(obj) {
    var b = (obj && obj.bounds) || {};
    return {
      minX: typeof b.minX === "number" ? b.minX : 0,
      maxX: typeof b.maxX === "number" ? b.maxX : 0,
      minY: typeof b.minY === "number" ? b.minY : 0,
      maxY: typeof b.maxY === "number" ? b.maxY : 0,
    };
  }

  // pickableRects returns the rect objects that carry an id (the node key) and
  // are pickable (default true when unset — the server bundle marks rects
  // pickable). Drawn back-to-front, so the LAST hit wins (topmost).
  function pickableRects(bundle) {
    var out = [];
    var objects = (bundle && Array.isArray(bundle.objects)) ? bundle.objects : [];
    for (var i = 0; i < objects.length; i++) {
      var o = objects[i];
      if (!o || o.kind !== "rect" || !o.id) continue;
      if (o.pickable === false) continue;
      out.push(o);
    }
    return out;
  }

  // ---- Interaction math (ported from gosx WASM bridge) ----

  // screenToWorld inverts the OrthoCamera2D screen transform (Y flipped). Kept
  // identical to the painter's forward transform so a click lands on the rect
  // the user actually sees.
  function screenToWorld(sx, sy, cam, cssW, cssH) {
    var zoom = cam.z > 0 ? cam.z : 1;
    return {
      x: (sx - cssW / 2) / zoom + cam.x,
      y: cam.y - (sy - cssH / 2) / zoom,
    };
  }

  // applyPan translates the camera by a screen-space drag delta. Dragging the
  // canvas moves content with the pointer, so the camera origin moves opposite:
  // panX -= dx/zoom; screen-Y grows down while world-Y grows up so panY += dy/zoom.
  function applyPan(cam, dxScreen, dyScreen) {
    var zoom = cam.z > 0 ? cam.z : 1;
    cam.x -= dxScreen / zoom;
    cam.y += dyScreen / zoom;
  }

  // applyZoom scales the camera toward the cursor: the world point under the
  // cursor stays pinned to the same screen pixel. Exact port of
  // canvasBoardApplyZoom.
  function applyZoom(cam, factor, cursorX, cursorY, cssW, cssH) {
    var oldZoom = cam.z > 0 ? cam.z : 1;
    if (!(factor > 0)) factor = 1;
    var world = screenToWorld(cursorX, cursorY, { x: cam.x, y: cam.y, z: oldZoom }, cssW, cssH);
    var newZoom = clampZoom(oldZoom * factor);
    cam.x = world.x - (cursorX - cssW / 2) / newZoom;
    cam.y = world.y + (cursorY - cssH / 2) / newZoom;
    cam.z = newZoom;
  }

  // pickWorld hit-tests the topmost pickable rect whose AABB contains (wx, wy).
  // Back-to-front draw order → iterate forward, keep the last hit. Mirrors
  // CanvasBoardAdapter.PickWorld.
  function pickWorld(bundle, wx, wy) {
    var rects = pickableRects(bundle);
    var hit = "";
    for (var i = 0; i < rects.length; i++) {
      var b = boundsOf(rects[i]);
      if (wx >= b.minX && wx <= b.maxX && wy >= b.minY && wy <= b.maxY) {
        hit = rects[i].id;
      }
    }
    return hit;
  }

  // pickWorldRect returns the ids of every pickable rect overlapping the
  // world-space rectangle, back-to-front. Mirrors PickWorldRect (AABB overlap).
  function pickWorldRect(bundle, x0, y0, x1, y1) {
    var minX = Math.min(x0, x1), maxX = Math.max(x0, x1);
    var minY = Math.min(y0, y1), maxY = Math.max(y0, y1);
    var rects = pickableRects(bundle);
    var ids = [];
    for (var i = 0; i < rects.length; i++) {
      var b = boundsOf(rects[i]);
      if (b.minX <= maxX && b.maxX >= minX && b.minY <= maxY && b.maxY >= minY) {
        ids.push(rects[i].id);
      }
    }
    return ids;
  }

  function rectCenter(obj) {
    var b = boundsOf(obj);
    return { x: (b.minX + b.maxX) / 2, y: (b.minY + b.maxY) / 2 };
  }

  // topmostLeftmost returns the id of the topmost-leftmost pickable rect: largest
  // world-Y (visually highest after the Y flip), ties → smallest world-X. Mirrors
  // canvasBoardTopmostLeftmost.
  function topmostLeftmost(bundle) {
    var rects = pickableRects(bundle);
    var best = "", bestY = 0, bestX = 0, have = false;
    for (var i = 0; i < rects.length; i++) {
      var c = rectCenter(rects[i]);
      if (!have || c.y > bestY + 0.5 || (Math.abs(c.y - bestY) <= 0.5 && c.x < bestX)) {
        best = rects[i].id;
        bestY = c.y;
        bestX = c.x;
        have = true;
      }
    }
    return best;
  }

  // navFrom returns the id of the pickable rect spatially nearest to currentID
  // in direction dir. Cost = primary-axis distance + 2*perpendicular-axis
  // distance, candidates behind the pressed direction are gated out, lowest cost
  // wins. Exact port of CanvasBoardAdapter.NavFrom (screen-up = larger world-Y).
  function navFrom(bundle, currentID, dir) {
    var rects = pickableRects(bundle);
    if (!rects.length) return "";
    currentID = normalizeKey((currentID || "").trim());
    if (currentID === "") return topmostLeftmost(bundle);

    var origin = null;
    for (var i = 0; i < rects.length; i++) {
      if (rects[i].id === currentID) {
        origin = rectCenter(rects[i]);
        break;
      }
    }
    if (!origin) return "";

    dir = (dir || "").toLowerCase().trim();
    var best = "", bestCost = -1;
    for (var j = 0; j < rects.length; j++) {
      var id = rects[j].id;
      if (id === currentID) continue;
      var c = rectCenter(rects[j]);
      var dx = c.x - origin.x;
      var dy = c.y - origin.y;
      var primary, perpendicular;
      if (dir === "left") {
        if (dx >= 0) continue;
        primary = -dx; perpendicular = Math.abs(dy);
      } else if (dir === "right") {
        if (dx <= 0) continue;
        primary = dx; perpendicular = Math.abs(dy);
      } else if (dir === "up") {
        // Screen-up = larger world-Y (Y flip).
        if (dy <= 0) continue;
        primary = dy; perpendicular = Math.abs(dx);
      } else if (dir === "down") {
        if (dy >= 0) continue;
        primary = -dy; perpendicular = Math.abs(dx);
      } else {
        return "";
      }
      var cost = primary + perpendicular * 2;
      if (bestCost < 0 || cost < bestCost) {
        bestCost = cost;
        best = id;
      }
    }
    return best;
  }

  // ---- Selection sink (direct setState, no signal) ----

  function applySingle(key) {
    var rt = runtime();
    var root = boardRoot();
    if (!rt || !root) return false;
    rt.setState(root, { selectedNode: key, selectedNodes: [] });
    return true;
  }

  function applyMulti(keys) {
    var rt = runtime();
    var root = boardRoot();
    if (!rt || !root) return false;
    var primary = keys.length ? keys[0] : "";
    rt.setState(root, { selectedNode: primary, selectedNodes: keys });
    return true;
  }

  // currentSelection reads the board's recorded selected node (the nav origin),
  // so arrow-key nav walks from wherever the selection currently is.
  function currentSelection() {
    var root = boardRoot();
    return root ? (root.getAttribute("data-studio-site-map-selected-node") || "") : "";
  }

  // normalizeToPageKey reduces any workspace node key ("component:home:hero",
  // "resource:shop:banner", "page:home") down to the owning PAGE-granularity
  // key ("page:<pageToken>"), by workspace-key convention (kind:pageToken:slug
  // for non-page kinds; page:pageToken for page kind itself). Returns "" if it
  // cannot be parsed as a workspace key at all.
  function normalizeToPageKey(key) {
    if (!key) return "";
    if (key.indexOf("page:") === 0) return key;
    var parts = key.split(":");
    return parts.length >= 2 ? "page:" + parts[1] : "";
  }

  // ---- Per-canvas controller ----

  function mountCanvas(canvas) {
    if (!(canvas instanceof HTMLCanvasElement)) return;
    if (canvas.getAttribute(BOUND_FLAG) === "true") return;
    canvas.setAttribute(BOUND_FLAG, "true");

    var section = canvas.closest("section") || canvas.parentElement || document;
    var host = canvas.parentElement || section;
    var ctx = null;
    var renderer = null;
    var renderRoute = "pending";
    var routeReason = "";
    var ROUTE_DEADLINE_MS = 2500;
    var routeDeadline = now() + ROUTE_DEADLINE_MS;
    // Timer-based safety net for the WebGPU->Canvas2D fallback. The rAF render
    // loop only evaluates routeDeadline when it runs, but requestAnimationFrame
    // callbacks are driven by the compositor's BeginFrame — and a hung GPU /
    // WebGPU adapter probe stalls the compositor, so rAF stops firing. Without
    // this timer, markFallback() (reached only inside the rAF loop) never runs
    // and the canvas is left permanently blank — neither WebGPU nor Canvas2D.
    // setTimeout runs on the main-thread event loop independent of the
    // compositor, so it forces the Canvas2D fallback even when rAF is dead. The
    // guard makes a late fire (route already resolved) a no-op.
    setTimeout(function() {
      if (disposed || renderRoute !== "pending") return;
      markFallback("webgpu-init-timeout");
      markDirty();
      render();
    }, ROUTE_DEADLINE_MS);

    // JS-side camera, seeded from the inline bundle's authored camera so the
    // first frame matches the server-precomputed view (pan 0,0 / zoom 1).
    var seed = parseBundle(section);
    var seedGPU = parseWebGPUBundle(section);
    var seedCam = (seed && seed.camera) || {};
    var cam = {
      x: typeof seedCam.x === "number" ? seedCam.x : 0,
      y: typeof seedCam.y === "number" ? seedCam.y : 0,
      z: typeof seedCam.z === "number" && seedCam.z > 0 ? seedCam.z : 1,
    };

    var dirty = true;
    var disposed = false;
    var selectedID = "";
    var htmlOverlay = null;
    var panelHost = null;

    function bundleNow() {
      // Re-read each paint so an authoring fragment refresh that swaps the
      // inline bundle is picked up live (cheap: a single querySelector + parse
      // only happens on the rAF cadence, and the JSON is small).
      return parseBundle(section) || seed;
    }

    function webgpuBundleNow() {
      return parseWebGPUBundle(section) || seedGPU;
    }

    function cssSize() {
      var w = Math.max(1, Math.round(canvas.clientWidth || canvas.width || 1));
      var h = Math.max(1, Math.round(canvas.clientHeight || canvas.height || 1));
      return { w: w, h: h };
    }

    function cloneWithCamera(b) {
      if (!b) return null;
      var out = {};
      for (var key in b) {
        if (Object.prototype.hasOwnProperty.call(b, key)) out[key] = b[key];
      }
      var camera = { mode: (b.camera && b.camera.mode) || "ortho2d", x: cam.x, y: cam.y, z: cam.z };
      out.camera = camera;
      return out;
    }

    // composeBundle returns the live fallback bundle with the JS camera spliced
    // in, so hit-testing, overlays, and the Canvas2D painter apply the current
    // pan/zoom rather than the authored camera.
    function composeBundle() {
      return cloneWithCamera(bundleNow());
    }

    function composeWebGPUBundle() {
      return cloneWithCamera(webgpuBundleNow());
    }

    function resizeBackingStore(size) {
      var dpr = window.devicePixelRatio || 1;
      var wantW = Math.round(size.w * dpr);
      var wantH = Math.round(size.h * dpr);
      if (canvas.width !== wantW) canvas.width = wantW;
      if (canvas.height !== wantH) canvas.height = wantH;
      return dpr;
    }

    function setRouteAttr(name, value) {
      if (canvas && typeof canvas.setAttribute === "function") canvas.setAttribute(name, value);
      if (host && host !== document && typeof host.setAttribute === "function") host.setAttribute(name, value);
      if (document && document.documentElement && typeof document.documentElement.setAttribute === "function") {
        document.documentElement.setAttribute(name, value);
      }
    }

    function clearRouteAttr(name) {
      if (canvas && typeof canvas.removeAttribute === "function") canvas.removeAttribute(name);
      if (host && host !== document && typeof host.removeAttribute === "function") host.removeAttribute(name);
      if (document && document.documentElement && typeof document.documentElement.removeAttribute === "function") {
        document.documentElement.removeAttribute(name);
      }
    }

    function markWebGPURoute() {
      renderRoute = "webgpu";
      routeReason = "";
      canvas.setAttribute("data-gosx-canvas-backend", "webgpu");
      setRouteAttr(WEBGPU_ROUTE_ATTR, "true");
      clearRouteAttr(FALLBACK_ATTR);
      clearRouteAttr(FALLBACK_REASON_ATTR);
    }

    function markFallback(reason) {
      if (renderRoute === "2d-fallback") return;
      renderRoute = "2d-fallback";
      routeReason = reason || "unknown";
      canvas.setAttribute("data-gosx-canvas-backend", "canvas2d");
      setRouteAttr(WEBGPU_ROUTE_ATTR, "false");
      setRouteAttr(FALLBACK_ATTR, "canvas2d");
      setRouteAttr(FALLBACK_REASON_ATTR, routeReason);
      console.warn("[gosx] wasm-free static WebGPU canvas unavailable (" + routeReason + "); falling back to Canvas2D painter.");
    }

    function routePending(reason) {
      routeReason = reason || "pending";
      setRouteAttr(WEBGPU_ROUTE_ATTR, "pending");
      setRouteAttr(FALLBACK_REASON_ATTR, routeReason);
    }

    function tryInitWebGPU() {
      if (renderer) return true;
      if (!seedGPU && !webgpuBundleNow()) {
        markFallback("missing-static-webgpu-bundle");
        return false;
      }
      if (typeof navigator === "undefined" || !navigator.gpu) {
        markFallback("navigator-gpu-unavailable");
        return false;
      }
      var api = webgpuAPI();
      if (!api) {
        if (now() < routeDeadline) {
          routePending("scene3d-webgpu-factory-pending");
          return false;
        }
        markFallback("scene3d-webgpu-factory-unavailable");
        return false;
      }
      if (typeof api.available === "function") {
        var available = false;
        try { available = api.available() === true; } catch (e) { available = false; }
        if (!available && now() < routeDeadline) {
          routePending("scene3d-webgpu-probe-pending");
          return false;
        }
      }
      try {
        renderer = api.createRenderer(canvas, { source: "gosx-studio-wasm-free-static-board" });
      } catch (e2) {
        markFallback("factory-threw:" + (e2 && e2.message ? e2.message : String(e2)));
        return false;
      }
      if (!renderer || typeof renderer.render !== "function") {
        markFallback("factory:" + (window.__gosx_scene3d_webgpu_factory_error || "renderer-unavailable"));
        return false;
      }
      var gpuBundle = composeWebGPUBundle();
      if (renderer && typeof renderer.supportsBundle === "function" && !renderer.supportsBundle(gpuBundle)) {
        markFallback("unsupported-static-webgpu-bundle");
        return false;
      }
      markWebGPURoute();
      return true;
    }

    function ensure2DContext() {
      if (ctx) return ctx;
      ctx = canvas.getContext("2d");
      if (!ctx) {
        setRouteAttr(FALLBACK_REASON_ATTR, "canvas2d-context-unavailable");
        return null;
      }
      return ctx;
    }

    function ensureOverlay(size) {
      if (!htmlOverlay) {
        var ov = document.createElement("div");
        ov.style.position = "absolute";
        ov.style.pointerEvents = "none";
        ov.style.overflow = "hidden";
        ov.style.zIndex = "1";
        // Insert as a sibling immediately after the canvas so it shares the
        // same offset parent. The canvas's own parent must be
        // position:relative/absolute for this to align — the board wrapper
        // already carries that style from the DOM site-map layout.
        if (canvas.parentElement) {
          canvas.parentElement.insertBefore(ov, canvas.nextSibling);
        }
        // Delegate click-to-select: walk up from the clicked child to the
        // nearest element carrying data-gosx-html-key, then call applySingle.
        ov.addEventListener("click", function (ev) {
          var el = ev.target;
          while (el && el !== ov) {
            var key = el.getAttribute("data-gosx-html-key");
            if (key) {
              selectedID = key;
              applySingle(key);
              markDirty(); // refresh the contextual panel for the new selection
              return;
            }
            el = el.parentElement;
          }
        });
        // Wire inline-edit persistence: a contenteditable surface field
        // (data-studio-field=<binding>) commits its edit to the authoring save
        // route on blur / Enter, so an inline edit survives a board repaint.
        // install() is idempotent, but guard the lookup so a missing module
        // (script load order) never throws.
        var inlineEdit = window.GoSXStudioCanvasInlineEditRuntime;
        if (inlineEdit && typeof inlineEdit.install === "function") {
          inlineEdit.install(ov, {});
        }
        htmlOverlay = ov;
      }
      // Keep the overlay box in sync with the canvas CSS box each frame.
      htmlOverlay.style.left   = canvas.offsetLeft + "px";
      htmlOverlay.style.top    = canvas.offsetTop  + "px";
      htmlOverlay.style.width  = size.w + "px";
      htmlOverlay.style.height = size.h + "px";

      // Contextual inspector panel host: a separate sibling layer above the
      // surface overlay so its inputs sit on top and the panel module can mount
      // exactly one positioned panel inside it. The host itself is click-through
      // (pointerEvents none); the panel re-enables pointer events for its inputs.
      if (!panelHost) {
        var ph = document.createElement("div");
        // Retained compatibility marker for existing contextual-panel CSS/tests;
        // Studio owns this WASM-free runtime despite the legacy muddy prefix.
        ph.setAttribute("data-muddy-contextual-panel-host", "true");
        ph.style.position = "absolute";
        ph.style.pointerEvents = "none";
        ph.style.zIndex = "2";
        if (canvas.parentElement) {
          canvas.parentElement.insertBefore(ph, htmlOverlay.nextSibling);
        }
        panelHost = ph;
      }
      panelHost.style.left   = canvas.offsetLeft + "px";
      panelHost.style.top    = canvas.offsetTop  + "px";
      panelHost.style.width  = size.w + "px";
      panelHost.style.height = size.h + "px";
    }

    // humanizeBinding turns a dotted binding tail into a readable fallback label
    // (e.g. "home.hero.headline" → "Headline") when the surface field carries no
    // explicit data-studio-field-label. Mirrors the panel's own fallback so the
    // label is stable regardless of which side computes it.
    function humanizeBinding(binding) {
      var parts = String(binding || "").split(".").filter(function (p) { return p !== ""; });
      var last = parts.length ? parts[parts.length - 1] : "";
      if (!last) return "Field";
      var spaced = last.replace(/[_-]+/g, " ").replace(/([a-z])([A-Z])/g, "$1 $2");
      return spaced.charAt(0).toUpperCase() + spaced.slice(1);
    }

    // buildPanelSelection reads the currently selected surface straight from the
    // overlay DOM (the markup already carries each binding, optional label, and
    // the current value as element text) and pairs it with the surface's
    // world-space bounds from the matching bundle.html entry. Returns null when
    // nothing is selected or the surface has no editable fields — which hides the
    // panel. Client-side only: no server round-trip.
    function buildPanelSelection(composed, key) {
      if (!key || !htmlOverlay || typeof htmlOverlay.querySelector !== "function") return null;
      var surface = htmlOverlay.querySelector('[data-gosx-html-key="' + key + '"]');
      // The painter mounts surfaces under [data-gosx-canvas-html="<id>"]; the
      // authored field carries data-gosx-html-key. Fall back to the container.
      var scope = surface;
      if (!scope) scope = htmlOverlay.querySelector('[data-gosx-canvas-html="' + key + '"]');
      if (!scope || typeof scope.querySelectorAll !== "function") return null;

      var fieldEls = scope.querySelectorAll("[data-studio-field]");
      // The selected element may itself be a field (data-studio-field on it).
      var collected = [];
      if (typeof scope.getAttribute === "function" && scope.getAttribute("data-studio-field")) {
        collected.push(scope);
      }
      for (var i = 0; i < fieldEls.length; i++) collected.push(fieldEls[i]);
      if (!collected.length) return null;

      var fields = [];
      for (var j = 0; j < collected.length; j++) {
        var el = collected[j];
        var binding = el.getAttribute("data-studio-field");
        if (!binding) continue;
        var label = el.getAttribute("data-studio-field-label") || humanizeBinding(binding);
        var value = typeof el.textContent === "string" ? el.textContent : "";
        fields.push({
          label: label,
          binding: binding,
          value: value,
          element: el,
          operation: el.getAttribute("data-studio-operation") || "",
          pageKey: el.getAttribute("data-studio-page-key") || "",
          pageRoute: el.getAttribute("data-studio-page-route") || "",
        });
      }
      if (!fields.length) return null;

      // Bounds from the matching bundle.html entry (world space), so the panel
      // anchors via the same transform the painter uses for the surface.
      var bounds = { x: 0, y: 0, width: 0, height: 0 };
      var html = (composed && Array.isArray(composed.html)) ? composed.html : [];
      for (var k = 0; k < html.length; k++) {
        if (html[k] && html[k].id === key) {
          bounds = {
            x: html[k].x, y: html[k].y,
            width: html[k].width, height: html[k].height,
          };
          break;
        }
      }
      return { key: key, bounds: bounds, fields: fields };
    }

    // renderPanel positions the contextual panel for the current selection using
    // the same OrthoCamera2D transform as the painter. Cheap + idempotent: the
    // panel module reuses its single element across frames.
    function renderPanel(composed, size) {
      var panel = window.GoSXStudioCanvasContextualPanelRuntime;
      if (!panel || typeof panel.render !== "function" || !panelHost) return;
      var p = painter();
      var key = selectedID || currentSelection();
      var selection = buildPanelSelection(composed, normalizeKey(key));
      var transform = (p && typeof p.screenTransform === "function")
        ? p.screenTransform(composed.camera, size.w, size.h) : null;
      panel.render(panelHost, { selection: selection, transform: transform });
    }

    function render() {
      if (disposed) return;
      var size = cssSize();
      if (dirty) {
        var composed = composeBundle();
        var p = painter();
        var paintedWebGPU = false;

        if (renderRoute !== "2d-fallback") {
          var ready = tryInitWebGPU();
          if (ready) {
            var gpuBundle = composeWebGPUBundle();
            if (gpuBundle) {
              var dprGPU = window.devicePixelRatio || 1;
              var wantW = Math.max(1, Math.round(size.w * dprGPU));
              var wantH = Math.max(1, Math.round(size.h * dprGPU));
              if (canvas.width !== wantW) canvas.width = wantW;
              if (canvas.height !== wantH) canvas.height = wantH;
              // 16a's ortho2d camera uses viewport width/height in CSS pixels;
              // only the canvas backing store is DPR-scaled.
              renderer.render(gpuBundle, { width: size.w, height: size.h, cssWidth: size.w, cssHeight: size.h, dpr: dprGPU });
              paintedWebGPU = true;
            } else {
              markFallback("missing-static-webgpu-bundle");
            }
          } else if (renderRoute !== "2d-fallback") {
            requestAnimationFrame(render);
            return;
          }
        }

        if (!paintedWebGPU) {
          if (!p) {
            // Painter script not yet evaluated — retry next frame.
            requestAnimationFrame(render);
            return;
          }
          var activeCtx = ensure2DContext();
          if (!activeCtx) {
            requestAnimationFrame(render);
            return;
          }
          var dpr = resizeBackingStore(size);
          activeCtx.setTransform(dpr, 0, 0, dpr, 0, 0);
          if (composed) {
            p.paint(activeCtx, composed, size.w, size.h, dpr);
          }
        }

        if (composed) {
          ensureOverlay(size);
          if (!p) p = painter();
          if (p && typeof p.renderCanvasBoardHTML === "function") {
            // Resolve which page-granularity surface should stay
            // pointer-interactive: selectedID (this closure's own
            // click-to-select tracker) first, since it is exact and
            // page-granularity already; it resets to "" on every fresh
            // load/reload though, so fall back to the board's
            // SERVER-PERSISTED selection (currentSelection(), survives
            // reloads), normalized from possible component/resource
            // granularity down to the owning page key. See
            // canvas2d_painter.js's cross-page pointer-interception fix.
            var activePageKey = selectedID || normalizeToPageKey(currentSelection());
            p.renderCanvasBoardHTML(htmlOverlay, composed, size.w, size.h, activePageKey);
          }
          // Medium-tier LOD: faithful per-page thumbnail <img> overlays. Gated
          // internally to [THUMB_ZOOM, SURFACE_ZOOM) and idempotent, so it can be
          // called every frame alongside the live-surface mount.
          if (p && typeof p.renderCanvasBoardThumbnails === "function") {
            p.renderCanvasBoardThumbnails(htmlOverlay, composed, size.w, size.h);
          }
          renderPanel(composed, size);
        }
        dirty = false;
      }
      requestAnimationFrame(render);
    }

    function markDirty() {
      dirty = true;
    }

    // ---- Pointer (pan + shift-drag-marquee); pick is a separate click handler ----
    var dragging = false;
    var marquee = false;
    var pointerId = null;
    var startX = 0, startY = 0;
    var lastX = 0, lastY = 0;
    var moved = 0;
    // Timestamp of the last drag (pan/marquee) that consumed the gesture. The
    // browser's click-after-pointerup behavior under setPointerCapture is
    // INCONSISTENT (a pan often emits no trailing click, a marquee sometimes
    // does), so a sticky boolean would leak. Instead the click handler ignores
    // any click arriving within CLICK_AFTER_DRAG_MS of a drag end — robust to
    // both "click fires" and "click suppressed" without leaking to a later tap.
    var lastDragEndAt = 0;

    function localPoint(ev) {
      var rect = canvas.getBoundingClientRect();
      return { x: ev.clientX - rect.left, y: ev.clientY - rect.top };
    }

    canvas.addEventListener("pointerdown", function (ev) {
      if (ev.button !== 0) return;
      var pt = localPoint(ev);
      dragging = true;
      marquee = ev.shiftKey === true;
      pointerId = ev.pointerId;
      startX = pt.x; startY = pt.y;
      lastX = pt.x; lastY = pt.y;
      moved = 0;
      try { canvas.setPointerCapture(ev.pointerId); } catch (e) { /* tolerate */ }
      // Keyboard nav targets a focused canvas. preventScroll is REQUIRED: a
      // plain focus() scrolls a partially-offscreen canvas into view, which
      // shifts the element mid-gesture and makes the click land off-target.
      try { canvas.focus({ preventScroll: true }); } catch (e) { try { canvas.focus(); } catch (e2) { /* tolerate */ } }
      ev.preventDefault();
    });

    canvas.addEventListener("pointermove", function (ev) {
      if (!dragging || (pointerId !== null && ev.pointerId !== pointerId)) return;
      var pt = localPoint(ev);
      var dx = pt.x - lastX;
      var dy = pt.y - lastY;
      moved += Math.abs(dx) + Math.abs(dy);
      lastX = pt.x; lastY = pt.y;
      if (marquee) {
        // Marquee preview is not painted (the inline bundle owns the visuals);
        // the rectangle is resolved on pointerup. Nothing to do per-move.
        return;
      }
      // Plain drag = pan.
      applyPan(cam, dx, dy);
      markDirty();
      ev.preventDefault();
    });

    function endPointer(ev) {
      if (!dragging) return;
      if (pointerId !== null && ev.pointerId !== pointerId) return;
      dragging = false;
      try { canvas.releasePointerCapture(pointerId); } catch (e) { /* tolerate */ }
      var pt = localPoint(ev);
      var size = cssSize();
      var b = bundleNow();

      if (marquee) {
        marquee = false;
        if (!b) return;
        // A marquee that barely moved is really a shift-click → let the click
        // handler own it (a single pick), so a stray shift never clears.
        if (moved <= CLICK_SLOP) return;
        lastDragEndAt = now();
        var w0 = screenToWorld(startX, startY, cam, size.w, size.h);
        var w1 = screenToWorld(pt.x, pt.y, cam, size.w, size.h);
        var ids = pickWorldRect(b, w0.x, w0.y, w1.x, w1.y).map(normalizeKey);
        selectedID = ids.length ? ids[0] : "";
        applyMulti(ids);
        markDirty(); // refresh the contextual panel for the new selection
        return;
      }
      // A plain drag (pan) was already applied per pointermove; record the
      // drag-end so the trailing synthetic click (if the browser fires one) is
      // ignored and a pan never also picks.
      if (moved > CLICK_SLOP) {
        lastDragEndAt = now();
      }
    }

    canvas.addEventListener("pointerup", endPointer);
    canvas.addEventListener("pointercancel", function (ev) {
      if (!dragging) return;
      if (pointerId !== null && ev.pointerId !== pointerId) return;
      dragging = false;
      marquee = false;
      try { canvas.releasePointerCapture(pointerId); } catch (e) { /* tolerate */ }
    });

    // Pick on a real click. A `click` fires only when pointerdown+up land on the
    // same element WITHOUT an intervening drag (the browser suppresses click
    // after a drag), so this cleanly captures the "tap a node" gesture while
    // pan/marquee (drags) are owned by the pointer handlers above. Using click
    // (rather than deriving the pick inside pointerup) sidesteps pointer-capture
    // quirks under synthesized input and matches native click semantics. A
    // shift-click with no drag falls through here too → a single pick, which is
    // the felt-right behavior (an empty marquee is a no-op, not a clear).
    canvas.addEventListener("click", function (ev) {
      // Ignore the trailing synthetic click the browser emits right after a
      // pan/marquee drag (so a drag never also picks).
      if (now() - lastDragEndAt < CLICK_AFTER_DRAG_MS) return;
      var b = bundleNow();
      if (!b) return;
      var pt = localPoint(ev);
      var size = cssSize();
      var world = screenToWorld(pt.x, pt.y, cam, size.w, size.h);
      var id = normalizeKey(pickWorld(b, world.x, world.y));
      selectedID = id;
      // A miss clears selection (click on empty space deselects), matching
      // canvasBoardApplyPick writing selectedID="".
      applySingle(id);
      markDirty(); // refresh the contextual panel for the new selection
    });

    // ---- Wheel (zoom toward cursor) ----
    canvas.addEventListener("wheel", function (ev) {
      var pt = localPoint(ev);
      var size = cssSize();
      // deltaY > 0 (scroll down / pinch out) zooms OUT → factor < 1.
      var factor = Math.pow(WHEEL_ZOOM_BASE, -ev.deltaY);
      applyZoom(cam, factor, pt.x, pt.y, size.w, size.h);
      markDirty();
      ev.preventDefault();
    }, { passive: false });

    // ---- Keyboard (arrow nav + escape clear) ----
    canvas.addEventListener("keydown", function (ev) {
      var dir = "";
      switch (ev.key) {
        case "ArrowUp": dir = "up"; break;
        case "ArrowDown": dir = "down"; break;
        case "ArrowLeft": dir = "left"; break;
        case "ArrowRight": dir = "right"; break;
        case "Escape":
          selectedID = "";
          applyMulti([]);
          markDirty(); // hide the contextual panel on deselect
          ev.preventDefault();
          return;
        default:
          return;
      }
      var b = bundleNow();
      if (!b) return;
      var origin = selectedID || currentSelection();
      var next = navFrom(b, origin, dir);
      if (next) {
        selectedID = next;
        applySingle(next);
        markDirty(); // refresh the contextual panel for the new selection
      }
      ev.preventDefault();
    });

    // Re-paint after an authoring fragment refresh (inline bundle may have been
    // swapped); also re-seed nothing — the JS camera is preserved across refreshes.
    document.addEventListener("gosxstudio:fragments-refreshed", markDirty);
    window.addEventListener("resize", markDirty);
    window.addEventListener("beforeunload", function () { disposed = true; });

    // Status hook so the live e2e can assert this client mounted and drive a
    // deterministic camera/selection without reaching into the closure.
    var controlAPI = {
      camera: function () { return { x: cam.x, y: cam.y, z: cam.z }; },
      setCamera: function (x, y, z) { cam.x = x; cam.y = y; cam.z = clampZoom(z); markDirty(); },
      screenToWorld: function (sx, sy) { var s = cssSize(); return screenToWorld(sx, sy, cam, s.w, s.h); },
      bundle: bundleNow,
      pickAt: function (sx, sy) { var s = cssSize(); var w = screenToWorld(sx, sy, cam, s.w, s.h); return pickWorld(bundleNow(), w.x, w.y); },
      selected: function () { return selectedID; },
      repaint: markDirty,
    };
    canvas.__gosxStudioCanvasWASMFree = controlAPI;
    canvas.GoSXStudioCanvasWasmFree = controlAPI;

    render();
  }

  function mountAll() {
    var canvases = document.querySelectorAll(CANVAS);
    for (var i = 0; i < canvases.length; i++) {
      mountCanvas(canvases[i]);
    }
  }

  // Mark the document so the e2e can assert the WASM-free client is the one
  // owning the canvas (vs the WASM canvas path).
  document.documentElement.setAttribute("data-gosx-canvas-wasm-free-client", "true");
  var api = { mountAll: mountAll };
  window.GoSXStudioCanvasWASMFreeClientRuntime = api;

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", mountAll);
  } else {
    mountAll();
  }
  // Re-scan after structural fragment refreshes that may re-insert the canvas.
  document.addEventListener("gosxstudio:fragments-refreshed", mountAll);
})();
