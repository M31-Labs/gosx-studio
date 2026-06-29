;(function () {
  "use strict";

  // Canvas2D painter for the Studio-owned WASM-free site-map board.
  //
  // This is a verbatim behavioral PORT of the gosx client painter
  // (gosx/client/js/bootstrap-src/26b1-canvas2d-painter.js
  // paintCanvasBundle + canvasBoardScreenTransform). It replays a RenderBundle
  // — the SAME JSON the WASM __gosx_render_canvas would return, but precomputed
  // server-side and shipped inline next to the canvas in
  // <script data-gosx-canvas-bundle> — onto a 2D context. No WASM, no
  // syscall/js, no __gosx_* globals: a pure JS renderer.
  //
  // The screen transform replicates render/bundle.OrthoCamera2D exactly so the
  // WASM-free board paints identically to the GPU/WASM path. OrthoCamera2D packs
  // pan into camera.x/.y and zoom into camera.z, and maps the world point
  // (panX, panY) to the viewport center, 1 world unit = 1 CSS pixel at zoom 1.
  // NDC has +Y up; a DOM canvas has +Y down, so the Y axis is flipped:
  //
  //     screenX = (worldX - panX) * zoom + cssWidth / 2
  //     screenY = (cssHeight / 2) - (worldY - panY) * zoom
  //
  // It installs window.GoSXStudioCanvas2DPainterRuntime = { paint,
  // screenTransform } on evaluation so the WASM-free render loop + interaction
  // handlers can reuse the identical forward transform (keeping click → rect
  // alignment exact).
  if (typeof window === "undefined") return;

  // canvasBoardScreenTransform builds the world→screen mapping from a bundle
  // camera. Defaults keep a missing/garbled camera from blanking the board:
  // zoom falls back to 1 and pan to (0, 0).
  function canvasBoardScreenTransform(camera, cssWidth, cssHeight) {
    var cam = camera || {};
    var zoom = typeof cam.z === "number" && cam.z > 0 ? cam.z : 1;
    var panX = typeof cam.x === "number" ? cam.x : 0;
    var panY = typeof cam.y === "number" ? cam.y : 0;
    var halfW = cssWidth / 2;
    var halfH = cssHeight / 2;
    return {
      zoom: zoom,
      x: function (worldX) {
        return (worldX - panX) * zoom + halfW;
      },
      // Y is flipped: world +Y (up in NDC) maps to a smaller canvas Y.
      y: function (worldY) {
        return halfH - (worldY - panY) * zoom;
      },
    };
  }

  function materialColor(bundle, index, fallback) {
    var materials = bundle && bundle.materials;
    if (Array.isArray(materials) && typeof index === "number" && index >= 0 && index < materials.length) {
      var color = materials[index] && materials[index].color;
      if (typeof color === "string" && color !== "") {
        return color;
      }
    }
    return fallback;
  }

  // paintCanvasBundle clears ctx with bundle.background, then draws every
  // rect object, line, and label through the OrthoCamera2D screen transform.
  // Tolerant of missing arrays and fields. All drawing is in CSS-logical
  // pixels: cssWidth/cssHeight are the logical viewport the transform centers
  // on, and the CALLER is responsible for any device-pixel-ratio scaling (the
  // render loop pre-applies ctx.setTransform(dpr, …) before calling). dpr is
  // accepted for API symmetry but does not affect the logical clear/fill region.
  function paintCanvasBundle(ctx, bundle, cssWidth, cssHeight, dpr) {
    if (!ctx || !bundle) return;
    var w = cssWidth > 0 ? cssWidth : 1;
    var h = cssHeight > 0 ? cssHeight : 1;
    void dpr;

    // Clear + paint the logical viewport. The caller's setTransform(dpr, …)
    // scales these CSS-pixel rects to cover the full device backing store.
    ctx.clearRect(0, 0, w, h);

    var bg = bundle.background;
    if (typeof bg === "string" && bg !== "") {
      ctx.fillStyle = bg;
      ctx.fillRect(0, 0, w, h);
    }

    var t = canvasBoardScreenTransform(bundle.camera, w, h);

    // Rect objects. RenderObject carries world-space bounds + a material index;
    // color comes from bundle.materials[materialIndex].color (the same lookup
    // the GPU path uses). Each rect is rendered as a polished page CARD: a dark
    // surface body (rounded when roundRect is available), a header/accent strip
    // in the card's material color (preserving the group-color semantics), and a
    // subtle 1px border. Cards are collected here so labels falling inside their
    // world bounds can be associated to them below (title vs subtitle).
    var objects = Array.isArray(bundle.objects) ? bundle.objects : [];
    var cards = [];
    for (var i = 0; i < objects.length; i++) {
      var obj = objects[i];
      if (!obj || obj.kind !== "rect") continue;
      var b = obj.bounds || {};
      var minX = typeof b.minX === "number" ? b.minX : 0;
      var maxX = typeof b.maxX === "number" ? b.maxX : 0;
      var minY = typeof b.minY === "number" ? b.minY : 0;
      var maxY = typeof b.maxY === "number" ? b.maxY : 0;
      // Top-left of the on-screen rect: smallest screenX (minX) and smallest
      // screenY (which is maxY after the Y flip).
      var sx = t.x(minX);
      var syTop = t.y(maxY);
      var width = (maxX - minX) * t.zoom;
      var height = (maxY - minY) * t.zoom;
      var accentColor = materialColor(bundle, obj.materialIndex, "#888888");

      drawCard(ctx, sx, syTop, width, height, accentColor);

      cards.push({
        minX: minX, maxX: maxX, minY: minY, maxY: maxY,
        sx: sx, syTop: syTop, width: width, height: height,
      });
    }

    // Lines.
    var lines = Array.isArray(bundle.lines) ? bundle.lines : [];
    for (var j = 0; j < lines.length; j++) {
      var line = lines[j];
      if (!line) continue;
      var from = line.from || {};
      var to = line.to || {};
      ctx.strokeStyle = typeof line.color === "string" && line.color !== "" ? line.color : "#cccccc";
      ctx.lineWidth = typeof line.lineWidth === "number" && line.lineWidth > 0 ? line.lineWidth : 1;
      ctx.beginPath();
      ctx.moveTo(t.x(from.x || 0), t.y(from.y || 0));
      ctx.lineTo(t.x(to.x || 0), t.y(to.y || 0));
      ctx.stroke();
    }

    // Sprites: thumbnail sprites are rendered as DOM <img> overlays in the
    // medium-zoom band by renderCanvasBoardThumbnails. At the FAR band the M7
    // enriched cards already stand in for the pages, so nothing extra is painted
    // for sprites on the canvas layer.

    // Labels. Partition into card-labels (whose world point falls inside a
    // card's bounds) and free labels (everything else). For each card, the
    // FIRST associated label is rendered as a prominent title and any further
    // labels as smaller, muted subtitles — stacked under the header strip. Free
    // labels render exactly as before (own color/font at their world position).
    var labels = Array.isArray(bundle.labels) ? bundle.labels : [];
    for (var k = 0; k < labels.length; k++) {
      var label = labels[k];
      if (!label) continue;
      var p = label.position || {};
      var lx = typeof p.x === "number" ? p.x : 0;
      var ly = typeof p.y === "number" ? p.y : 0;

      var card = findCardForPoint(cards, lx, ly);
      if (card) {
        if (!card.labels) card.labels = [];
        card.labels.push(label);
        continue;
      }

      // Free label: unchanged behavior.
      if (typeof label.font === "string" && label.font !== "") {
        ctx.font = label.font;
      }
      ctx.fillStyle = typeof label.color === "string" && label.color !== "" ? label.color : "#e6edf3";
      ctx.fillText(String(label.text == null ? "" : label.text), t.x(lx), t.y(ly));
    }

    // Render each card's associated labels as a title (+ optional subtitles),
    // stacked from just below the header strip. Drawn after the free-label pass
    // so each label is painted exactly once.
    for (var c = 0; c < cards.length; c++) {
      drawCardLabels(ctx, t, cards[c]);
    }
  }

  // CARD_SURFACE is the card body fill: a dark surface a touch lighter than the
  // board background so cards read as raised panels. CARD_BORDER is a subtle
  // hairline. CARD_TITLE / CARD_SUBTITLE are the label colors (subtitle muted).
  var CARD_SURFACE = "#161b22";
  var CARD_BORDER = "rgba(240,246,252,0.12)";
  var CARD_TITLE = "#e6edf3";
  var CARD_SUBTITLE = "#8b949e";
  var CARD_RADIUS = 8; // screen px at zoom 1
  var CARD_HEADER = 24; // screen px at zoom 1

  // drawCard paints one styled page card: a rounded (when available) surface
  // body, a header/accent strip in the card's material color clipped to the
  // rounded top, and a 1px border. All optional ctx APIs are guarded so the
  // node:vm stub and older browsers degrade to plain rects.
  function drawCard(ctx, x, y, w, h, accentColor) {
    if (w <= 0 || h <= 0) return;
    var hasRound = typeof ctx.roundRect === "function";
    // Radius/header scale a little with the card size but stay bounded so they
    // never swallow tiny cards.
    var r = Math.max(0, Math.min(CARD_RADIUS, w / 2, h / 2));
    var header = Math.max(0, Math.min(CARD_HEADER, h));

    // Body surface.
    ctx.fillStyle = CARD_SURFACE;
    if (hasRound) {
      ctx.beginPath();
      ctx.roundRect(x, y, w, h, r);
      ctx.fill();
    } else {
      ctx.fillRect(x, y, w, h);
    }

    // Header / accent strip in the card (group) color. Clip to the rounded card
    // shape when possible so the strip's top corners follow the body radius.
    if (header > 0) {
      ctx.fillStyle = accentColor;
      if (hasRound && typeof ctx.save === "function" && typeof ctx.clip === "function") {
        ctx.save();
        ctx.beginPath();
        ctx.roundRect(x, y, w, h, r);
        ctx.clip();
        ctx.fillRect(x, y, w, header);
        ctx.restore();
      } else {
        ctx.fillRect(x, y, w, header);
      }
    }

    // Border.
    ctx.strokeStyle = CARD_BORDER;
    ctx.lineWidth = 1;
    if (hasRound) {
      ctx.beginPath();
      ctx.roundRect(x, y, w, h, r);
      ctx.stroke();
    } else if (typeof ctx.strokeRect === "function") {
      ctx.strokeRect(x, y, w, h);
    }
  }

  // findCardForPoint returns the first card whose world bounds contain (wx, wy),
  // or null. Bounds are inclusive so labels placed exactly on an edge still
  // associate to their card.
  function findCardForPoint(cards, wx, wy) {
    for (var i = 0; i < cards.length; i++) {
      var c = cards[i];
      if (wx >= c.minX && wx <= c.maxX && wy >= c.minY && wy <= c.maxY) {
        return c;
      }
    }
    return null;
  }

  // drawCardLabels renders a card's associated labels: the first as a prominent
  // title under the header strip, the rest as smaller muted subtitles stacked
  // beneath it. Font sizes scale with zoom but clamp to a legible range. Each
  // label keeps its own color only as a fallback — the title/subtitle roles set
  // the primary/muted palette so cards read consistently.
  function drawCardLabels(ctx, t, card) {
    var labels = card.labels;
    if (!labels || labels.length === 0) return;

    var zoom = t.zoom > 0 ? t.zoom : 1;
    var pad = Math.max(6, 8 * zoom);
    // Place the title baseline a little below the header strip; subtitles follow.
    var titleSize = clamp(16 * zoom, 11, 28);
    var subSize = clamp(12 * zoom, 9, 20);

    var x = card.sx + pad;
    var y = card.syTop + Math.min(CARD_HEADER, card.height) + titleSize;

    for (var i = 0; i < labels.length; i++) {
      var label = labels[i];
      var isTitle = i === 0;
      var size = isTitle ? titleSize : subSize;
      var weight = isTitle ? "600" : "400";
      ctx.font = weight + " " + size.toFixed(0) +
        "px -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif";
      ctx.fillStyle = isTitle
        ? (typeof label.color === "string" && label.color !== "" ? label.color : CARD_TITLE)
        : CARD_SUBTITLE;
      ctx.fillText(String(label.text == null ? "" : label.text), x, y);
      y += size + Math.max(2, 4 * zoom);
    }
  }

  function clamp(v, lo, hi) {
    return v < lo ? lo : (v > hi ? hi : v);
  }

  // CANVAS_LOD_SURFACE_ZOOM is the level-of-detail threshold for live DOM
  // surfaces. Below this zoom the enriched Canvas2D cards stand in for the live
  // pages (a readable overview), so the heavy live-dom surfaces are NOT mounted;
  // at/above it the surfaces mount as usual so the page is editable up close.
  // 0.8 sits inside the live zoom range [ZOOM_MIN=0.1, ZOOM_MAX=8]: cards read at
  // the typical zoomed-out overview, live pages take over a touch below 1:1.
  var CANVAS_LOD_SURFACE_ZOOM = 0.8;

  // CANVAS_LOD_THUMB_ZOOM is the lower bound of the MEDIUM level-of-detail band.
  // The board has a 3-tier LOD: far (enriched cards) → medium (faithful page
  // thumbnail <img>) → near (the live editable dom surface). Thumbnail imgs show
  // only while CANVAS_LOD_THUMB_ZOOM <= camera.z < CANVAS_LOD_SURFACE_ZOOM: below
  // THUMB_ZOOM the cards alone read as the overview (a thumbnail would be a
  // smeared sliver) and at/above SURFACE_ZOOM the live surface owns the page. It
  // must sit strictly below CANVAS_LOD_SURFACE_ZOOM (0.8) so the medium band is
  // non-empty, and inside the live zoom range [ZOOM_MIN=0.1, ZOOM_MAX=8].
  var CANVAS_LOD_THUMB_ZOOM = 0.3;

  // renderCanvasBoardHTML mounts each bundle.html entry as a camera-positioned
  // DOM element in an overlay div that sits above the Canvas2D surface. Each
  // entry is identified by h.id via a [data-gosx-canvas-html] attribute so the
  // helper is idempotent: re-calling with the same bundle updates position only.
  // innerHTML is only written when the markup changes (and not while a descendant
  // is focused) to avoid blowing away in-progress user input. Entries absent from
  // the new bundle are removed. All coordinates use the same OrthoCamera2D
  // forward transform as paintCanvasBundle so DOM and Canvas layers stay aligned.
  //
  // Level-of-detail: when zoomed out below CANVAS_LOD_SURFACE_ZOOM the live
  // surfaces are not mounted (the Canvas2D cards represent the pages instead) and
  // any previously-mounted surfaces are torn down — EXCEPT a surface that
  // currently contains the document's activeElement, which is kept so an
  // in-progress edit is never destroyed (the focus-guard).
  function renderCanvasBoardHTML(overlay, bundle, cssWidth, cssHeight) {
    if (!overlay || !bundle || !Array.isArray(bundle.html)) return;
    var doc0 = overlay.ownerDocument;

    // LOD gate: zoomed out → no live surfaces. Read zoom defensively (a missing
    // camera should not collapse the board to the overview).
    var zoom = bundle.camera && typeof bundle.camera.z === "number" ? bundle.camera.z : 1;
    if (zoom < CANVAS_LOD_SURFACE_ZOOM) {
      var existing = overlay.querySelectorAll('[data-gosx-canvas-html]');
      for (var e = 0; e < existing.length; e++) {
        var node = existing[e];
        // Focus-guard: never tear down a surface the user is actively editing.
        var focused = doc0 && doc0.activeElement && node.contains(doc0.activeElement);
        if (!focused) overlay.removeChild(node);
      }
      return;
    }

    var t = canvasBoardScreenTransform(bundle.camera, cssWidth, cssHeight);
    var doc = overlay.ownerDocument, seen = Object.create(null);
    for (var i = 0; i < bundle.html.length; i++) {
      var h = bundle.html[i]; if (!h || !h.id) continue; seen[h.id] = true;
      var el = overlay.querySelector('[data-gosx-canvas-html="' + h.id + '"]');
      if (!el) {
        el = doc.createElement('div');
        el.setAttribute('data-gosx-canvas-html', h.id);
        el.style.position = 'absolute'; el.style.transformOrigin = '0 0';
        el.style.pointerEvents = (h.pointerEvents === 'none') ? 'none' : 'auto';
        overlay.appendChild(el);
      }
      var editing = doc.activeElement && el.contains(doc.activeElement);
      if (!editing && el.__gosxHTMLMarkup !== h.markup) { el.innerHTML = h.markup; el.__gosxHTMLMarkup = h.markup; }
      var left = t.x(h.x), top = t.y(h.y);
      el.style.transform = 'translate(' + left + 'px,' + top + 'px) scale(' + t.zoom + ')';
      el.style.width = h.width + 'px'; if (h.height) el.style.height = h.height + 'px';
    }
    var kids = overlay.querySelectorAll('[data-gosx-canvas-html]');
    for (var j = 0; j < kids.length; j++) { var id = kids[j].getAttribute('data-gosx-canvas-html'); if (!seen[id]) overlay.removeChild(kids[j]); }
  }

  // renderCanvasBoardThumbnails mounts each bundle.sprite (a faithful per-page
  // thumbnail data URI) as a camera-positioned overlay <img> in the SAME overlay
  // div as the live surfaces. It is the MEDIUM tier of the 3-tier LOD and mirrors
  // renderCanvasBoardHTML's idempotent mount/update/teardown exactly — each img is
  // identified by sprite.id via a [data-gosx-canvas-thumb] attribute so re-calling
  // with the same bundle updates position only; sprites absent from the new bundle
  // (or a band exit) are removed. Coordinates use the identical OrthoCamera2D
  // forward transform as paintCanvasBundle, with the sprite anchored at its
  // bottom-left world point (position) like a rect, so the thumbnail overlays its
  // card pixel-for-pixel.
  //
  // Thumbnails stay as DOM <img> elements for now so the thumbnail LOD tier
  // remains separate from the enriched far-card canvas layer.
  //
  // Level-of-detail: thumbnails mount ONLY in the medium band
  // [CANVAS_LOD_THUMB_ZOOM, CANVAS_LOD_SURFACE_ZOOM). Below THUMB_ZOOM (far) the
  // enriched cards are the overview; at/above SURFACE_ZOOM (near) the live dom
  // surface owns the page — in both out-of-band cases any previously-mounted
  // thumbnails are torn down. Zoom is read defensively: a missing/garbled camera
  // defaults to 1 (near band → no thumbnails), never blanking the board.
  function renderCanvasBoardThumbnails(overlay, bundle, cssWidth, cssHeight) {
    if (!overlay || !bundle) return;

    function teardownAll() {
      if (typeof overlay.querySelectorAll !== "function") return;
      var existing = overlay.querySelectorAll('[data-gosx-canvas-thumb]');
      for (var e = 0; e < existing.length; e++) overlay.removeChild(existing[e]);
    }

    // LOD gate: only the medium band shows thumbnail imgs.
    var zoom = bundle.camera && typeof bundle.camera.z === "number" && bundle.camera.z > 0
      ? bundle.camera.z : 1;
    if (zoom < CANVAS_LOD_THUMB_ZOOM || zoom >= CANVAS_LOD_SURFACE_ZOOM) {
      teardownAll();
      return;
    }

    var sprites = Array.isArray(bundle.sprites) ? bundle.sprites : [];
    var t = canvasBoardScreenTransform(bundle.camera, cssWidth, cssHeight);
    var doc = overlay.ownerDocument, seen = Object.create(null);
    for (var i = 0; i < sprites.length; i++) {
      var sp = sprites[i];
      if (!sp || !sp.id) continue;
      // A sprite with no src has nothing to display — skip it (and let any prior
      // img for this id be torn down as stale below).
      if (typeof sp.src !== "string" || sp.src === "") continue;
      seen[sp.id] = true;

      var img = overlay.querySelector('[data-gosx-canvas-thumb="' + sp.id + '"]');
      if (!img) {
        img = doc.createElement('img');
        img.setAttribute('data-gosx-canvas-thumb', sp.id);
        img.style.position = 'absolute';
        // Thumbnails are a passive preview; clicks fall through to the canvas
        // picker beneath (the overlay's surfaces own pointer events up close).
        img.style.pointerEvents = 'none';
        overlay.appendChild(img);
      }
      if (img.src !== sp.src) img.src = sp.src;

      // Anchor at the sprite's bottom-left world point (position = minX, minY),
      // matching the rect card: left = t.x(minX); top = t.y(minY) - height*zoom
      // (= t.y(maxY)). So the thumbnail covers its card exactly.
      var pos = sp.position || {};
      var w = (typeof sp.width === "number" ? sp.width : 0) * t.zoom;
      var h = (typeof sp.height === "number" ? sp.height : 0) * t.zoom;
      var left = t.x(typeof pos.x === "number" ? pos.x : 0);
      var top = t.y(typeof pos.y === "number" ? pos.y : 0) - h;
      img.style.left = left + 'px';
      img.style.top = top + 'px';
      img.style.width = w + 'px';
      img.style.height = h + 'px';
    }

    if (typeof overlay.querySelectorAll === "function") {
      var kids = overlay.querySelectorAll('[data-gosx-canvas-thumb]');
      for (var j = 0; j < kids.length; j++) {
        var id = kids[j].getAttribute('data-gosx-canvas-thumb');
        if (!seen[id]) overlay.removeChild(kids[j]);
      }
    }
  }

  var api = {
    paint: paintCanvasBundle,
    screenTransform: canvasBoardScreenTransform,
    renderCanvasBoardHTML: renderCanvasBoardHTML,
    renderCanvasBoardThumbnails: renderCanvasBoardThumbnails,
    CANVAS_LOD_SURFACE_ZOOM: CANVAS_LOD_SURFACE_ZOOM,
    CANVAS_LOD_THUMB_ZOOM: CANVAS_LOD_THUMB_ZOOM,
  };
  window.GoSXStudioCanvas2DPainterRuntime = api;
})();
