;(function () {
  "use strict";

  // Contextual inspector panel for the WASM-free Canvas2D editor loop
  // (Studio-owned runtime).
  //
  // When a surface on the site-map board is selected, this renders a small,
  // Figma-inspector-style panel anchored near that surface, listing its editable
  // field(s) — each as a label + a prefilled <input>. Editing a field persists
  // through the SAME server save path as inline editing
  // (window.GoSXStudioCanvasInlineEditRuntime.persist) and reflects the new value
  // straight back into the live surface element, so the canvas board and the
  // panel agree.
  //
  // It is intentionally dependency-free, window-attached (no bundler), and a
  // sibling of the absolutely-positioned overlay — matching its sibling files
  // (canvas_wasm_free_client.js, canvas2d_painter.js, canvas_inline_edit.js).
  //
  // Public API:
  //   window.GoSXStudioCanvasContextualPanelRuntime.render(host, state)
  //     host  : an element the panel is (singly) mounted into. The caller owns
  //             positioning the host itself; this module positions the panel
  //             absolutely within it using the supplied transform.
  //     state :
  //       { selection: null }                       → hides/removes the panel.
  //       { selection: { key, bounds:{x,y,width,height},
  //                       fields:[{label,binding,value[,element]}] },
  //         transform: { zoom, x(worldX), y(worldY) } }
  //         → builds/updates exactly one panel anchored near the surface.
  //
  // render is idempotent: re-rendering reuses the one mounted panel, so a
  // per-frame call from the render loop never accumulates DOM.

  if (typeof window === "undefined") return;

  // The single panel element marker (one panel per host) + field part markers.
  var PANEL_ATTR = "data-muddy-contextual-panel";
  var LABEL_ATTR = "data-muddy-panel-field-label";
  var INPUT_ATTR = "data-muddy-panel-field-input";
  var FIELD_ATTR = "data-studio-field";
  // Offset (CSS px) from the surface's top-left so the panel hugs it without
  // covering the surface's own first line of content.
  var OFFSET_X = 8;
  var OFFSET_Y = 8;

  function doc() {
    return (typeof document !== "undefined") ? document : null;
  }

  function inlineEdit() {
    var ie = window.GoSXStudioCanvasInlineEditRuntime;
    return ie && typeof ie.persist === "function" ? ie : null;
  }

  // existingPanel returns the host's single mounted panel, or null.
  function existingPanel(host) {
    if (!host || typeof host.querySelector !== "function") return null;
    return host.querySelector("[" + PANEL_ATTR + "]");
  }

  function removePanel(host) {
    var panel = existingPanel(host);
    if (panel && panel.parentElement && typeof panel.parentElement.removeChild === "function") {
      panel.parentElement.removeChild(panel);
    }
  }

  // makePanelShell creates the panel container (absolute, unobtrusive). Pointer
  // events are enabled so its inputs are usable even though the surrounding
  // overlay is click-through.
  function makePanelShell(d) {
    var panel = d.createElement("div");
    panel.setAttribute(PANEL_ATTR, "");
    var s = panel.style;
    s.position = "absolute";
    s.zIndex = "2";
    s.pointerEvents = "auto";
    s.minWidth = "180px";
    s.maxWidth = "260px";
    s.padding = "8px 10px";
    s.borderRadius = "8px";
    s.background = "#11151c";
    s.color = "#e6edf3";
    s.border = "1px solid rgba(255,255,255,0.12)";
    s.boxShadow = "0 6px 20px rgba(0,0,0,0.35)";
    s.font = "12px/1.4 system-ui, -apple-system, Segoe UI, sans-serif";
    return panel;
  }

  // makeFieldRow builds a label + input for one field and wires its commit.
  function makeFieldRow(d, field) {
    var row = d.createElement("div");
    row.style.marginBottom = "8px";

    var label = d.createElement("label");
    label.setAttribute(LABEL_ATTR, "");
    label.textContent = field.label || humanize(field.binding);
    var ls = label.style;
    ls.display = "block";
    ls.marginBottom = "3px";
    ls.fontSize = "11px";
    ls.letterSpacing = "0.02em";
    ls.color = "rgba(230,237,243,0.65)";

    var input = d.createElement("input");
    input.setAttribute(INPUT_ATTR, "");
    input.setAttribute(FIELD_ATTR, field.binding || "");
    input.setAttribute("type", "text");
    input.value = (field.value === undefined || field.value === null) ? "" : String(field.value);
    var is = input.style;
    is.width = "100%";
    is.boxSizing = "border-box";
    is.padding = "5px 7px";
    is.borderRadius = "6px";
    is.border = "1px solid rgba(255,255,255,0.16)";
    is.background = "#0b0e13";
    is.color = "#e6edf3";
    is.font = "inherit";

    // commit persists the input's current value through the shared inline-edit
    // POST path and mirrors it into the live surface element so the board agrees.
    // Repaint safety runs immediately after reflect(), before the async persist
    // resolves, because an unfocused CanvasBoard HTML overlay may repaint stale
    // initial markup on the next frame.
    function commit() {
      var binding = input.getAttribute(FIELD_ATTR);
      if (!binding) return;
      var value = input.value;
      var ie = inlineEdit();
      reflect(field, value);
      if (!ie) return;
      var el = field && field.element;
      if (el && typeof ie.persistRepaintSafe === "function") {
        var ownerDoc = el.ownerDocument || doc();
        ie.persistRepaintSafe(ownerDoc, el);
      }
      var persistOpts = {
        binding: binding,
        value: value,
        operation: field && field.operation,
        pageKey: field && field.pageKey,
        pageRoute: field && field.pageRoute,
      };
      ie.persist(persistOpts);
    }

    if (typeof input.addEventListener === "function") {
      input.addEventListener("change", function () { commit(); });
      input.addEventListener("keydown", function (ev) {
        if (ev.key !== "Enter" || ev.shiftKey === true) return;
        if (typeof ev.preventDefault === "function") ev.preventDefault();
        commit();
      });
    }

    row.appendChild(label);
    row.appendChild(input);
    return row;
  }

  // reflect writes the committed value back into the live surface field element so
  // the canvas board renders the same text the panel shows. The element is taken
  // from the field when provided (the WASM-free client supplies it from the DOM);
  // otherwise it falls back to a document lookup by html key + binding.
  function reflect(field, value) {
    try {
      var el = field && field.element ? field.element : findSurfaceField(field);
      if (el) el.textContent = value;
    } catch (e) {
      // Best-effort: a failed reflect must never break the (already persisted) edit.
    }
  }

  function findSurfaceField(field) {
    var d = doc();
    if (!d || typeof d.querySelector !== "function") return null;
    var binding = field && field.binding;
    if (!binding) return null;
    // Prefer a field scoped to its surface container when a key is known.
    return d.querySelector('[' + FIELD_ATTR + '="' + cssEscape(binding) + '"]');
  }

  function cssEscape(s) {
    return String(s).replace(/["\\]/g, "\\$&");
  }

  // humanize turns a dotted binding tail into a readable fallback label, e.g.
  // "home.hero.headline" → "Headline".
  function humanize(binding) {
    var parts = String(binding || "").split(".").filter(function (p) { return p !== ""; });
    var last = parts.length ? parts[parts.length - 1] : "";
    if (!last) return "Field";
    var spaced = last.replace(/[_-]+/g, " ").replace(/([a-z])([A-Z])/g, "$1 $2");
    return spaced.charAt(0).toUpperCase() + spaced.slice(1);
  }

  // anchor positions the panel near the surface's top-left using the transform's
  // world→screen mapping, plus a small offset so it hugs the surface.
  function anchor(panel, selection, transform) {
    var bounds = (selection && selection.bounds) || {};
    var wx = typeof bounds.x === "number" ? bounds.x : 0;
    var wy = typeof bounds.y === "number" ? bounds.y : 0;
    var left = OFFSET_X, top = OFFSET_Y;
    if (transform && typeof transform.x === "function" && typeof transform.y === "function") {
      left = transform.x(wx) + OFFSET_X;
      top = transform.y(wy) + OFFSET_Y;
    }
    panel.style.left = left + "px";
    panel.style.top = top + "px";
  }

  // render is the single entry point. Hides the panel when nothing is selected,
  // otherwise (re)builds the one panel for the current selection + transform.
  function render(host, state) {
    if (!host || typeof host.appendChild !== "function") return;
    var selection = state && state.selection;
    if (!selection) { removePanel(host); return; }
    var fields = Array.isArray(selection.fields) ? selection.fields : [];
    if (!fields.length) { removePanel(host); return; }

    var d = doc();
    if (!d || typeof d.createElement !== "function") return;

    // Idempotent: reuse the mounted panel, rebuilding its body for the current
    // selection. A fresh panel is created only when none exists yet.
    var panel = existingPanel(host);
    if (!panel) {
      panel = makePanelShell(d);
      host.appendChild(panel);
    } else {
      // Clear the previous selection's rows before rebuilding.
      panel.innerHTML = "";
    }
    panel.setAttribute("data-muddy-panel-key", selection.key || "");

    for (var i = 0; i < fields.length; i++) {
      panel.appendChild(makeFieldRow(d, fields[i]));
    }

    anchor(panel, selection, state && state.transform);
  }

  var api = { render: render };
  window.GoSXStudioCanvasContextualPanelRuntime = api;
})();
