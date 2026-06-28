;(function () {
  "use strict";

  // CanvasBoard inline-edit commit bridge (WASM-free Canvas2D editor loop).
  //
  // The WASM-free site-map board paints each page as an HTML "surface" inside an
  // absolutely-positioned overlay (see canvas_wasm_free_client.js ensureOverlay).
  // Some elements on those surfaces are authored fields: they carry
  // `contenteditable` + `data-studio-field="<binding>"` (e.g. "home.hero.headline").
  // Editing one is purely visual until committed — and a board repaint would lose
  // it. This module persists an inline edit by delegating to the host-agnostic
  // GoSXStudioInlineEditRuntime (already bundled via runtime.go:EngineRuntimeScript)
  // which handles focusin/focusout/keydown, the baseline WeakMap, double-POST guard,
  // and the POST body. CanvasBoard-specific repaint-safe bundle rewriting is
  // injected through the onCommit hook.
  //
  // It is intentionally dependency-free and attaches to `window` (no bundler),
  // matching its sibling files (canvas_wasm_free_client.js, canvas2d_painter.js).
  //
  // Public API:
  //   window.GoSXStudioCanvasInlineEditRuntime = { install, deriveKeys, persist, persistRepaintSafe, csrfToken }
  //   window.__muddyCanvasInlineEdit = window.GoSXStudioCanvasInlineEditRuntime
  //
  // The __muddyCanvasInlineEdit alias is temporary compatibility for hosts that
  // have not yet moved painter/client/contextual-panel ownership into Studio.
  //
  //   install(overlay, opts)
  //     overlay : the HTML overlay element whose [contenteditable][data-studio-field]
  //               descendants are editable surfaces.
  //     opts.fetch : injectable fetch (default window.fetch) — for tests.
  //     opts.route/action : POST target override.
  //
  // The commit payload mirrors the server's AuthoringMutationFromForm
  // (mutations.go): an x-www-form-urlencoded body with
  //   gosx_studio_operation=save-control
  //   gosx_studio_binding=<binding>
  //   gosx_studio_value=<new text>
  //   gosx_studio_page_key / _component_key / _control_key   (derived from binding)

  if (typeof window === "undefined") return;

  // The server-precomputed inline RenderBundle script the WASM-free client re-reads
  // every frame (canvas_wasm_free_client.js parseBundle/bundleNow).
  var BUNDLE_SEL = "[data-gosx-canvas-bundle]";
  // The painter's per-container markup cache key (canvas2d_painter.js).
  var PAINTER_CACHE = "__gosxHTMLMarkup";
  // The editable element's surface key, matching both the inline bundle entry's
  // `id` and the painter container's data-gosx-canvas-html (canvasboard side).
  var HTML_KEY_ATTR = "data-gosx-html-key";
  // The painter mounts each surface in [data-gosx-canvas-html="<id>"].
  var CANVAS_HTML_ATTR = "data-gosx-canvas-html";

  // csrfToken resolves the session CSRF token the authoring action requires on
  // unsafe (POST) requests. The gosx session middleware (session.Protect) rejects
  // a tokenless POST with 403, so an inline-edit commit MUST carry it. We send it
  // as the X-CSRF-Token header (session.Protect checks the header first, then the
  // csrf_token form field). Resolution order, all best-effort:
  //   1. an explicit opts.csrfToken (tests / future callers),
  //   2. a <meta name="csrf-token"> if the host ever adds one,
  //   3. the value of any rendered input[name="csrf_token"] — the editor page
  //      already emits one per authoring form (mirrors public-site.js), so the
  //      same live token is reused without threading new server plumbing.
  // Returns "" when no token can be found (the POST then proceeds tokenless and
  // the server decides — never throws).
  function csrfToken(explicit) {
    var t = (typeof explicit === "string") ? explicit.trim() : "";
    if (t) return t;
    if (typeof document === "undefined") return "";
    try {
      var meta = document.querySelector
        ? document.querySelector('meta[name="csrf-token"], meta[name="csrf_token"]')
        : null;
      if (meta && typeof meta.getAttribute === "function") {
        var mv = (meta.getAttribute("content") || "").trim();
        if (mv) return mv;
      }
      var input = document.querySelector
        ? document.querySelector('input[name="csrf_token"]')
        : null;
      if (input && typeof input.value === "string") {
        var iv = input.value.trim();
        if (iv) return iv;
      }
    } catch (e) { /* tolerate a missing/odd DOM */ }
    return "";
  }

  // deriveKeys delegates to the engine's canonical implementation so there is one
  // source of truth. Falls back to a local copy if the engine is not yet loaded
  // (should not happen in production — engine is bundled first via runtime.go).
  function deriveKeys(binding) {
    var rt = (typeof window !== "undefined") ? window.GoSXStudioInlineEditRuntime : null;
    if (rt && typeof rt.deriveKeys === "function") {
      return rt.deriveKeys(binding);
    }
    // Fallback (engine not yet available): replicate the same algorithm.
    var parts = String(binding || "").split(".").filter(function (p) { return p !== ""; });
    if (parts.length && parts[0] === "pages") parts = parts.slice(1);
    var tail = parts.slice(-3);
    var keys = { page: "", component: "", control: "" };
    if (tail.length >= 1) keys.control = tail[tail.length - 1];
    if (tail.length >= 2) keys.component = tail[tail.length - 2];
    if (tail.length >= 3) keys.page = tail[tail.length - 3];
    return keys;
  }

  function actionFromDocument() {
    if (typeof document === "undefined" || !document.querySelector) return "";
    try {
      var form = document.querySelector("[data-gosx-studio-authoring-managed]");
      if (form && typeof form.getAttribute === "function") {
        var action = (form.getAttribute("action") || "").trim();
        if (action) return action;
      }
      if (form && typeof form.action === "string" && form.action.trim()) return form.action.trim();
    } catch (e) { /* tolerate a missing/odd DOM */ }
    return "";
  }

  function resolveAction(opts) {
    opts = opts || {};
    var explicit = (typeof opts.action === "string" && opts.action.trim()) ||
      (typeof opts.route === "string" && opts.route.trim()) ||
      "";
    if (explicit) return explicit;
    var fromForm = actionFromDocument();
    if (fromForm) return fromForm;
    if (typeof window !== "undefined" && window.location && typeof window.location.href === "string") {
      return window.location.href;
    }
    return "";
  }

  // buildBody assembles the urlencoded payload for a direct persist() call.
  // (Used only by persist() below — install() delegates commit POSTs to the engine.)
  function buildBody(binding, value, meta) {
    meta = meta || {};
    var operation = meta.operation || "save-control";
    if (operation === "update-page") {
      var pageKey = meta.pageKey || "";
      var pageRoute = meta.pageRoute || "";
      var UpdateUSP = (typeof URLSearchParams !== "undefined")
        ? URLSearchParams
        : (window.URLSearchParams || null);
      var updateParams = new UpdateUSP();
      updateParams.set("gosx_studio_operation", "update-page");
      updateParams.set("gosx_studio_page_key", pageKey);
      updateParams.set("gosx_studio_page_label", value);
      updateParams.set("gosx_studio_page_route", pageRoute);
      return updateParams.toString();
    }
    var keys = deriveKeys(binding);
    var USP = (typeof URLSearchParams !== "undefined")
      ? URLSearchParams
      : (window.URLSearchParams || null);
    var params = new USP();
    params.set("gosx_studio_operation", "save-control");
    params.set("gosx_studio_binding", binding);
    params.set("gosx_studio_value", value);
    params.set("gosx_studio_page_key", keys.page);
    params.set("gosx_studio_component_key", keys.component);
    params.set("gosx_studio_control_key", keys.control);
    return params.toString();
  }

  // persist is the single POST path for a direct authored save-control edit.
  // The contextual panel (canvas_contextual_panel.js) calls this so field-name /
  // body-building logic lives in exactly ONE place (DRY). install() now delegates
  // inline-edit commits to the engine, but persist() is still the panel's path.
  //
  //   persist({ binding, value, fetch, route }) → the fetch result (or undefined
  //   when no fetch is available). `fetch` defaults to window.fetch and `route`
  //   to the resolved authoring action. Defensive: a blank binding or missing fetch is a
  //   no-op (returns undefined) rather than throwing.
  function persist(opts) {
    opts = opts || {};
    var binding = opts.binding;
    if (!binding) return undefined;
    var value = opts.value;
    if (value === undefined || value === null) value = "";
    var route = resolveAction(opts);
    var doFetch = opts.fetch || (typeof window !== "undefined" ? window.fetch : null);
    if (typeof doFetch !== "function") return undefined;
    var headers = { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8" };
    var token = csrfToken(opts.csrfToken);
    if (token) headers["X-CSRF-Token"] = token;
    return doFetch(route, {
      method: "POST",
      headers: headers,
      body: buildBody(binding, value, opts),
      credentials: "same-origin",
    });
  }

  // surfaceContainerFor walks up from the committed element to the painter's
  // mount container [data-gosx-canvas-html=...]. Returns the container or null.
  function surfaceContainerFor(el, root) {
    var node = el;
    while (node && node !== root) {
      if (typeof node.getAttribute === "function" &&
          node.getAttribute(CANVAS_HTML_ATTR) !== null) {
        return node;
      }
      node = node.parentElement || null;
    }
    return null;
  }

  // surfaceKeyFor resolves the surface id for an edited element.
  function surfaceKeyFor(el, container) {
    if (el && typeof el.getAttribute === "function") {
      var k = el.getAttribute(HTML_KEY_ATTR);
      if (k) return k;
    }
    if (container && typeof container.getAttribute === "function") {
      var ck = container.getAttribute(CANVAS_HTML_ATTR);
      if (ck) return ck;
    }
    return "";
  }

  // persistRepaintSafe makes a just-committed edit survive the next board repaint.
  // The WASM-free client re-reads the inline bundle JSON every frame and the
  // painter rewrites the surface container's innerHTML from bundle.html[i].markup
  // whenever it differs from the container's cached markup (and nothing inside is
  // focused). After a blur the element is no longer focused, so unless the inline
  // bundle now carries the saved markup the very next frame reverts the surface.
  //
  // It (1) rewrites the matching inline-bundle html entry's `markup` to the
  // container's CURRENT innerHTML (which already reflects the edit, byte-identical
  // to what the painter would serialize/compare), and (2) seeds the container's
  // painter cache to that same string so even an immediate repaint is a no-op.
  //
  // It must run synchronously from the edited DOM state, before the async
  // authoring POST resolves (called via onCommit in install() and directly from
  // the panel commit).
  //
  // Signature: persistRepaintSafe(overlay, el, committedMarkup)
  //   overlay        : the root overlay element (used for querySelector fallback).
  //   el             : the just-committed editable field element.
  //   committedMarkup: optional pre-captured markup string (avoids re-reading DOM).
  //
  // Fully defensive: any missing node / parse failure is swallowed — a failed
  // cache update must never break the (already-visually-correct) edit.
  function persistRepaintSafe(overlay, el, committedMarkup) {
    try {
      if (!overlay || !el) return;
      var container = surfaceContainerFor(el, overlay);
      if (!container) return;
      var key = surfaceKeyFor(el, container);
      if (!key) return;
      var newMarkup = (typeof committedMarkup === "string") ? committedMarkup : container.innerHTML;
      if (typeof newMarkup !== "string") return;

      // Seed the painter cache so an immediate repaint treats this as a no-op.
      container[PAINTER_CACHE] = newMarkup;
      container._gosxMarkup = newMarkup;
      patchCanvasBoardRuntime(container, key, newMarkup);

      // Rewrite the inline bundle's matching html entry.
      var node = (typeof overlay.querySelector === "function"
        ? overlay.querySelector(BUNDLE_SEL) : null)
        || (typeof document !== "undefined" && document.querySelector
          ? document.querySelector(BUNDLE_SEL) : null);
      if (!node) return;
      var raw = (node.textContent || "").trim();
      if (!raw) return;
      var bundle = JSON.parse(raw);
      if (!bundle || !Array.isArray(bundle.html)) return;
      var changed = false;
      for (var i = 0; i < bundle.html.length; i++) {
        var h = bundle.html[i];
        if (h && h.id === key) { h.markup = newMarkup; changed = true; break; }
      }
      if (changed) node.textContent = JSON.stringify(bundle);
    } catch (e) {
      // Swallow: source-of-truth refresh is best-effort; the edit already shows.
    }
  }

  function patchCanvasBoardRuntime(container, key, markup) {
    try {
      var host = container ? container.parentElement : null;
      while (host && !host.__gosxBoardHTMLLayer) host = host.parentElement;
      var canvas = host && typeof host.querySelector === "function"
        ? host.querySelector("[data-gosx-surface-id]") : null;
      var boardID = canvas && typeof canvas.getAttribute === "function"
        ? canvas.getAttribute("data-gosx-surface-id") : "";
      var fn = (typeof window !== "undefined") ? window.__gosx_canvas_update_html : null;
      if (boardID && key && typeof fn === "function") fn(boardID, key, markup);
    } catch (e) {
      // Best-effort: old runtimes lack the hook; persistence still succeeds.
    }
  }

  // install wires the overlay for inline editing by delegating to the host-agnostic
  // GoSXStudioInlineEditRuntime engine (already in the bundle via runtime.go). The
  // engine handles focusin/focusout/keydown, the baseline WeakMap, and the commit
  // POST. CanvasBoard repaint-safe bundle surgery is injected via onCommit so the
  // engine remains host-agnostic.
  //
  // Fail-safe: if the engine is absent (script load order problem), this is a no-op
  // rather than throwing — consistent with the previous guard pattern in the call sites.
  function install(overlay, opts) {
    if (!overlay || typeof overlay.addEventListener !== "function") return;
    opts = opts || {};
    var rt = (typeof window !== "undefined") ? window.GoSXStudioInlineEditRuntime : null;
    if (!rt || typeof rt.install !== "function") return; // engine missing — fail safe
    rt.install(overlay, {
      action: opts.action || opts.route || undefined,
      // Pass the getter (not its result) so the engine re-reads the live token at
      // each commit — matches the pre-refactor commit-time read, and this
      // csrfToken() checks <meta> first (which the engine's own resolver can't see).
      csrfToken: csrfToken,
      fetch: opts.fetch || undefined,
      onCommit: function (el, value) {
        // persistRepaintSafe runs synchronously from the current DOM state, before
        // the engine fires the async POST. This replicates the original timing:
        // the old commit() called persistRepaintSafe(overlay, el) before persist().
        persistRepaintSafe(overlay, el);
      },
    });
  }

  var api = { install: install, deriveKeys: deriveKeys, persist: persist, persistRepaintSafe: persistRepaintSafe, csrfToken: csrfToken };
  window.GoSXStudioCanvasInlineEditRuntime = api;
  window.__muddyCanvasInlineEdit = api;
})();
