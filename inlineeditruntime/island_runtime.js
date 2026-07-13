;(function () {
  "use strict";

  // GoSXStudioInlineEditRuntime — host-agnostic inline-edit island.
  //
  // Persists in-canvas contenteditable edits by POSTing a save-control mutation
  // the moment an edit is committed (blur / Enter). Ported from the muddy-side
  // canvas_inline_edit.js but made fully host-agnostic: no muddy-specific route,
  // no bundle-rewriting machinery. Any host gets inline editing by calling
  // window.GoSXStudioInlineEditRuntime.install(overlayEl, opts).
  //
  // Public API:
  //   window.GoSXStudioInlineEditRuntime = {
  //     install, deriveKeys,
  //     startPreviewTextEdit, syncPreviewTextEdit, finishPreviewTextEdit,
  //     startPreviewTextSession, syncPreviewTextSession, finishPreviewTextSession,
  //     handlePreviewTextInputEvent, handlePreviewTextKeyEvent,
  //     handlePreviewTextPasteEvent, handlePreviewTextBlurEvent,
  //     previewTextEditState
  //   }
  //
  //   install(root, opts)
  //     root : the overlay container element whose [contenteditable][data-studio-field]
  //            descendants are editable canvas surfaces.
  //     opts.action     : POST target URL — string or getter () => string (see resolution order below).
  //     opts.csrfToken  : CSRF token — string or getter () => string. A getter is
  //                       re-evaluated at each commit so a rotating token stays live (see resolution order below).
  //     opts.fetch      : injectable fetch (default window.fetch) — for tests.
  //     opts.onCommit   : optional callback(field, value) called just BEFORE the
  //                       POST so hosts can apply repaint-safe logic (e.g. muddy's
  //                       persistRepaintSafe) without baking muddy logic here.
  //
  // Action / CSRF resolution order (first non-empty value wins):
  //   1. opts.action / opts.csrfToken passed to install().
  //   2. data-gosx-studio-inline-edit-action / data-gosx-studio-inline-edit-csrf
  //      attribute on the root element.
  //   3. A [data-gosx-studio-authoring-managed] form's action + csrf_token input
  //      found anywhere in the document.
  //   4. window.location.href (action) / "" (csrf — POST proceeds tokenless).
  //
  // After a successful POST the JSON response is handed to
  // window.GoSXStudioAuthoringRuntime.handleResult(result, meta) when that
  // global is present, so preview refresh, change selection, and save-chrome
  // all update consistently.
  //
  // Muddy-specific bundle-rewriting (persistRepaintSafe) is intentionally NOT
  // ported here. Supply opts.onCommit to attach host-specific behaviour before
  // the POST.

  if (typeof window === "undefined") return;

  var FIELD_ATTR = "data-studio-field";
  var OPERATION_ATTR = "data-studio-operation";
  var PAGE_KEY_ATTR = "data-studio-page-key";
  var PAGE_ROUTE_ATTR = "data-studio-page-route";
  var ROOT_ACTION_ATTR = "data-gosx-studio-inline-edit-action";
  var ROOT_CSRF_ATTR = "data-gosx-studio-inline-edit-csrf";
  var MANAGED_FORM_ATTR = "data-gosx-studio-authoring-managed";
  // Marks an overlay we have already wired so a second install() is a no-op.
  var INSTALLED_FLAG = "__gosxInlineEditInstalled";

  // isEditableField returns true when el is a non-null element that has a
  // non-empty data-studio-field attribute and contenteditable !== "false".
  function isEditableField(el) {
    if (!el || typeof el.getAttribute !== "function") return false;
    var binding = el.getAttribute(FIELD_ATTR);
    if (!binding) return false;
    var ce = el.getAttribute("contenteditable");
    if (ce === null || ce === undefined) return false;
    if (String(ce).toLowerCase() === "false") return false;
    return true;
  }

  // editableField walks up from a node to the nearest ancestor that satisfies
  // isEditableField, stopping at root. Returns the element or null.
  function editableField(node, root) {
    var el = node;
    while (el && el !== root) {
      if (isEditableField(el)) return el;
      el = el.parentElement || null;
    }
    return null;
  }

  // textOf reads el.textContent defensively.
  function textOf(el) {
    if (!el) return "";
    var t = el.textContent;
    return typeof t === "string" ? t : "";
  }

  function frameDocument(frame) {
    try {
      return frame && (frame.contentDocument || (frame.contentWindow && frame.contentWindow.document)) || null;
    } catch (error) {
      return null;
    }
  }

  function placeCaretAtEnd(doc, node) {
    try {
      node.focus();
      var selection = doc.defaultView && doc.defaultView.getSelection ? doc.defaultView.getSelection() : null;
      if (!selection || !doc.createRange) return;
      var range = doc.createRange();
      range.selectNodeContents(node);
      range.collapse(false);
      selection.removeAllRanges();
      selection.addRange(range);
    } catch (error) {
      return;
    }
  }

  function previewTextEditState(frame) {
    return frame && frame.__gosxStudioInlineEdit || null;
  }

  function previewTextSelection(edit, opts) {
    var selection = {};
    if (opts && typeof opts.selection === "function") {
      try { selection = opts.selection(edit) || {}; } catch (error) { selection = {}; }
    }
    return {
      selection: selection.selection || edit.blockKey || edit.field || "",
      kind: selection.kind || "preview-field"
    };
  }

  function previewTextTarget(edit, opts, includeSelection) {
    var target = {
      field: edit.field || "",
      editable: "text",
      blockKey: edit.blockKey || ""
    };
    if (includeSelection) {
      var selection = previewTextSelection(edit, opts);
      target.selection = selection.selection;
      target.kind = selection.kind;
    }
    return target;
  }

  function inlineTextPayload(edit, text) {
    return {
      text: text,
      field: edit.field || "",
      previous: edit.originalValue || "",
      label: edit.label || ""
    };
  }

  function emitPreviewTextOperation(opts, type, operation) {
    opts = opts || {};
    if (typeof opts.emitOperation === "function") {
      try { return opts.emitOperation(type, operation); } catch (error) { return null; }
    }
    if (typeof opts.emitEditorOperation === "function") {
      var detail = operation || {};
      try { return opts.emitEditorOperation(type, detail); } catch (error) { return null; }
    }
    return null;
  }

  function inlineTextEventDetail(edit, reason, text) {
    return {
      field: edit.field || "",
      editable: "text",
      blockKey: edit.blockKey || "",
      label: edit.label || "",
      text: text == null ? textOf(edit.target) : text,
      reason: reason || ""
    };
  }

  function emitPreviewTextEvent(opts, name, edit, reason, text) {
    opts = opts || {};
    if (typeof opts.emitEvent !== "function") return;
    try { opts.emitEvent(name, inlineTextEventDetail(edit, reason, text)); } catch (error) { return; }
  }

  function previewTextControl(field, opts) {
    if (!opts || typeof opts.controlForField !== "function") return null;
    try {
      var control = opts.controlForField(field);
      return control && "value" in control ? control : null;
    } catch (error) {
      return null;
    }
  }

  function previewTextSessionOptions(frame, host) {
    host = host || {};
    return {
      form: host.form || null,
      target: host.target || (frame && frame.__gosxStudioPreviewDockTarget),
      controlForField: host.controlForField,
      selection: function (edit) {
        var form = host.form || null;
        var selection = "";
        var kind = "";
        if (form && typeof form.getAttribute === "function") {
          selection = form.getAttribute("data-studio-selection") || "";
          kind = form.getAttribute("data-studio-selection-kind") || "";
        }
        return {
          selection: selection || edit.blockKey || edit.field || "",
          kind: kind || "preview-field"
        };
      },
      setDirty: function (reason) {
        if (typeof host.setStatus === "function") {
          try { host.setStatus("dirty", "Draft changed", reason || "inline-text"); } catch (error) { /* tolerate */ }
        }
        if (typeof host.setDirty === "function") {
          try { host.setDirty(reason || "inline-text"); } catch (error) { /* tolerate */ }
        }
      },
      emitOperation: function (type, operation) {
        if (typeof host.emitOperation === "function") {
          try { return host.emitOperation(type, operation); } catch (error) { return null; }
        }
        if (typeof host.emitEditorOperation === "function") {
          try { return host.emitEditorOperation(type, operation); } catch (error) { return null; }
        }
        return null;
      },
      emitEvent: function (name, detail) {
        if (typeof host.emitEvent !== "function") return;
        try { host.emitEvent(name, detail); } catch (error) { return; }
      },
      onFinish: function (finishedFrame, edit, reason, commit) {
        if (typeof host.onFinish === "function") {
          try { return host.onFinish(finishedFrame, edit, reason, commit); } catch (error) { return null; }
        }
        if (typeof host.updateDock === "function") {
          try { return host.updateDock(finishedFrame, edit, reason, commit); } catch (error) { return null; }
        }
        return null;
      }
    };
  }

  function startPreviewTextSession(frame, detail, reason, host) {
    return startPreviewTextEdit(frame, detail, reason, previewTextSessionOptions(frame, host));
  }

  function syncPreviewTextSession(frame, reason, host) {
    return syncPreviewTextEdit(frame, reason, previewTextSessionOptions(frame, host));
  }

  function finishPreviewTextSession(frame, commit, reason, host) {
    return finishPreviewTextEdit(frame, commit, reason, previewTextSessionOptions(frame, host));
  }

  function previewTextEventEdit(frame, event) {
    var edit = previewTextEditState(frame);
    if (!edit || !edit.target || !event || event.target !== edit.target) return null;
    return edit;
  }

  function handlePreviewTextInputEvent(frame, event, host) {
    if (!previewTextEventEdit(frame, event)) return false;
    return syncPreviewTextSession(frame, "input", host);
  }

  function handlePreviewTextKeyEvent(frame, event, host) {
    if (!previewTextEventEdit(frame, event)) return false;
    if (event.key === "Escape") {
      event.preventDefault();
      finishPreviewTextSession(frame, false, "escape", host);
      return true;
    }
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      finishPreviewTextSession(frame, true, "enter", host);
      return true;
    }
    return false;
  }

  function handlePreviewTextPasteEvent(frame, event, host) {
    if (!previewTextEventEdit(frame, event)) return false;
    var text = event.clipboardData && event.clipboardData.getData ? event.clipboardData.getData("text/plain") : "";
    if (!text) return false;
    event.preventDefault();
    try {
      var doc = frameDocument(frame);
      var selection = doc && doc.defaultView && doc.defaultView.getSelection ? doc.defaultView.getSelection() : null;
      if (!doc || !selection || !selection.rangeCount) return false;
      selection.deleteFromDocument();
      selection.getRangeAt(0).insertNode(doc.createTextNode(text));
      selection.collapseToEnd();
      syncPreviewTextSession(frame, "paste", host);
      return true;
    } catch (error) {
      return false;
    }
  }

  function handlePreviewTextBlurEvent(frame, event, host) {
    if (!previewTextEventEdit(frame, event)) return false;
    return finishPreviewTextSession(frame, true, "blur", host);
  }

  function startPreviewTextEdit(frame, detail, reason, opts) {
    opts = opts || {};
    detail = detail || {};
    var target = opts.target || (frame && frame.__gosxStudioPreviewDockTarget);
    if (!frame || !target || !detail.field || detail.editable !== "text") return false;
    finishPreviewTextEdit(frame, true, "restart-inline-text", opts);
    var doc = frameDocument(frame);
    if (!doc) return false;
    var startReason = reason || "preview-dock";
    var control = previewTextControl(detail.field, opts);
    var text = textOf(target);
    var edit = {
      target: target,
      field: detail.field || "",
      blockKey: detail.blockKey || "",
      label: detail.label || "",
      control: control,
      originalText: text,
      originalValue: control && "value" in control ? control.value || "" : text,
      lastText: text,
      hadHref: target.hasAttribute && target.hasAttribute("href"),
      originalHref: target.hasAttribute && target.hasAttribute("href") ? target.getAttribute("href") || "" : ""
    };
    frame.__gosxStudioInlineEdit = edit;
    if (edit.hadHref) target.removeAttribute("href");
    target.setAttribute("contenteditable", "plaintext-only");
    target.setAttribute("spellcheck", "true");
    target.setAttribute("data-gosx-studio-inline-editing", "true");
    if (opts.form && typeof opts.form.setAttribute === "function") {
      opts.form.setAttribute("data-gosx-studio-inline-field", detail.field);
    }
    placeCaretAtEnd(doc, target);
    emitPreviewTextOperation(opts, "inline_text_start", {
      mutation: false,
      reason: startReason,
      target: previewTextTarget(edit, opts, true),
      payload: { label: detail.label || "" }
    });
    emitPreviewTextEvent(opts, "gosxstudio:inline-text-start", edit, startReason, text);
    return true;
  }

  function syncPreviewTextEdit(frame, reason, opts) {
    opts = opts || {};
    var edit = previewTextEditState(frame);
    if (!edit || !edit.target) return false;
    var text = textOf(edit.target);
    if (edit.control && "value" in edit.control) edit.control.value = text;
    if (edit.lastText === text) return true;
    edit.lastText = text;
    if (typeof opts.setDirty === "function") {
      try { opts.setDirty(reason || "inline-text"); } catch (error) { /* tolerate */ }
    }
    emitPreviewTextOperation(opts, "set_text", {
      mutation: true,
      reason: reason || "inline-text",
      target: previewTextTarget(edit, opts, true),
      payload: inlineTextPayload(edit, text)
    });
    emitPreviewTextEvent(opts, "gosxstudio:inline-text", edit, reason || "inline-text", text);
    return true;
  }

  function finishPreviewTextEdit(frame, commit, reason, opts) {
    opts = opts || {};
    var edit = previewTextEditState(frame);
    if (!edit || !edit.target) return false;
    if (edit.finishing) return false;
    edit.finishing = true;
    if (commit) {
      syncPreviewTextEdit(frame, reason || "commit", opts);
      emitPreviewTextEvent(opts, "gosxstudio:inline-text-commit", edit, reason || "commit", textOf(edit.target));
    } else {
      edit.target.textContent = edit.originalText || "";
      if (edit.control && "value" in edit.control) edit.control.value = edit.originalValue || "";
      emitPreviewTextOperation(opts, "inline_text_cancel", {
        mutation: false,
        reason: reason || "cancel",
        target: previewTextTarget(edit, opts, false),
        payload: inlineTextPayload(edit, edit.originalValue || "")
      });
      emitPreviewTextEvent(opts, "gosxstudio:inline-text-cancel", edit, reason || "cancel", edit.originalValue || "");
    }
    if (edit.hadHref) edit.target.setAttribute("href", edit.originalHref);
    edit.target.removeAttribute("contenteditable");
    edit.target.removeAttribute("data-gosx-studio-inline-editing");
    if (opts.form && typeof opts.form.removeAttribute === "function") {
      opts.form.removeAttribute("data-gosx-studio-inline-field");
    }
    frame.__gosxStudioInlineEdit = null;
    if (typeof opts.onFinish === "function") {
      try { opts.onFinish(frame, edit, reason || (commit ? "commit" : "cancel"), !!commit); } catch (error) { /* tolerate */ }
    }
    return true;
  }

  // deriveKeys maps a binding string to { page, component, control } by
  // splitting on ".", dropping a leading "pages" segment, and trailing-aligning
  // the last three segments. Fewer than three segments fill what they can,
  // leaving the rest empty.
  //
  //   "pages.home.hero.headline" → { page:"home", component:"hero", control:"headline" }
  //   "home.hero.headline"       → { page:"home", component:"hero", control:"headline" }
  //   "hero.headline"            → { page:"",     component:"hero", control:"headline" }
  //   "headline"                 → { page:"",     component:"",     control:"headline" }
  function deriveKeys(binding) {
    var parts = String(binding || "").split(".").filter(function (p) { return p !== ""; });
    if (parts.length && parts[0] === "pages") parts = parts.slice(1);
    var tail = parts.slice(-3);
    var keys = { page: "", component: "", control: "" };
    if (tail.length >= 1) keys.control = tail[tail.length - 1];
    if (tail.length >= 2) keys.component = tail[tail.length - 2];
    if (tail.length >= 3) keys.page = tail[tail.length - 3];
    return keys;
  }

  // resolveAction resolves the POST target URL from the four-step priority chain.
  function resolveAction(root, optsAction) {
    // Step 1: explicit opt (string, or a getter () => string evaluated now).
    var rawAction = typeof optsAction === "function" ? optsAction() : optsAction;
    var v = typeof rawAction === "string" ? rawAction.trim() : "";
    if (v) return v;
    // Step 2: attribute on root.
    if (root && typeof root.getAttribute === "function") {
      v = (root.getAttribute(ROOT_ACTION_ATTR) || "").trim();
      if (v) return v;
    }
    // Step 3: managed authoring form in the document.
    if (typeof document !== "undefined" && document.querySelector) {
      try {
        var form = document.querySelector("[" + MANAGED_FORM_ATTR + "]");
        if (form) {
          v = (form.getAttribute("action") || "").trim();
          if (v) return v;
        }
      } catch (e) { /* tolerate */ }
    }
    // Step 4: current page URL.
    return (typeof window !== "undefined" && window.location) ? window.location.href : "";
  }

  // resolveCSRF resolves the CSRF token from the four-step priority chain.
  function resolveCSRF(root, optsCSRF) {
    // Step 1: explicit opt (string, or a getter () => string evaluated now so a
    // host can re-read a rotating token at each commit, not just at install).
    var rawCSRF = typeof optsCSRF === "function" ? optsCSRF() : optsCSRF;
    var v = typeof rawCSRF === "string" ? rawCSRF.trim() : "";
    if (v) return v;
    // Step 2: attribute on root.
    if (root && typeof root.getAttribute === "function") {
      v = (root.getAttribute(ROOT_CSRF_ATTR) || "").trim();
      if (v) return v;
    }
    // Step 3: managed authoring form's csrf_token input in the document.
    if (typeof document !== "undefined" && document.querySelector) {
      try {
        var form = document.querySelector("[" + MANAGED_FORM_ATTR + "]");
        if (form) {
          var input = form.querySelector ? form.querySelector("input[name='csrf_token']") : null;
          if (input && typeof input.value === "string") {
            v = input.value.trim();
            if (v) return v;
          }
        }
      } catch (e) { /* tolerate */ }
    }
    // Step 4: no token found — POST proceeds tokenless.
    return "";
  }

  // editMetaFromElement extracts optional per-field operation overrides.
  function editMetaFromElement(el) {
    if (!el || typeof el.getAttribute !== "function") return {};
    return {
      operation: el.getAttribute(OPERATION_ATTR) || "",
      pageKey: el.getAttribute(PAGE_KEY_ATTR) || "",
      pageRoute: el.getAttribute(PAGE_ROUTE_ATTR) || "",
    };
  }

  // buildBody assembles the application/x-www-form-urlencoded POST payload.
  function buildBody(binding, value, meta) {
    meta = meta || {};
    var operation = meta.operation || "save-control";
    var USP = (typeof URLSearchParams !== "undefined") ? URLSearchParams : null;
    if (!USP) return "";

    if (operation === "update-page") {
      var up = new USP();
      up.set("gosx_studio_operation", "update-page");
      up.set("gosx_studio_page_key", meta.pageKey || "");
      up.set("gosx_studio_page_label", value);
      up.set("gosx_studio_page_route", meta.pageRoute || "");
      return up.toString();
    }

    var keys = deriveKeys(binding);
    var p = new USP();
    p.set("gosx_studio_operation", "save-control");
    p.set("gosx_studio_binding", binding);
    p.set("gosx_studio_value", value);
    p.set("gosx_studio_page_key", keys.page);
    p.set("gosx_studio_component_key", keys.component);
    p.set("gosx_studio_control_key", keys.control);
    return p.toString();
  }

  // install wires delegated focusin/focusout/keydown listeners on root.
  // Idempotent: a second call on the same root is a no-op.
  function install(root, opts) {
    if (!root || typeof root.addEventListener !== "function") return;
    if (root[INSTALLED_FLAG] === true) return; // idempotent guard
    root[INSTALLED_FLAG] = true;

    opts = opts || {};

    // Snapshot-baseline storage. WeakMap when available, single-slot fallback.
    var hasWeakMap = (typeof WeakMap !== "undefined");
    var starts = hasWeakMap ? new WeakMap() : null;
    var fallbackEl = null;
    var fallbackStart = "";

    function rememberStart(el) {
      var v = textOf(el);
      if (hasWeakMap) starts.set(el, v);
      else { fallbackEl = el; fallbackStart = v; }
    }

    function startOf(el) {
      if (hasWeakMap) return starts.has(el) ? starts.get(el) : undefined;
      return (fallbackEl === el) ? fallbackStart : undefined;
    }

    function clearStart(el) {
      if (hasWeakMap) starts.delete(el);
      else if (fallbackEl === el) { fallbackEl = null; fallbackStart = ""; }
    }

    // cancelEdit reverts el's in-progress text back to the value captured at
    // focusin (a no-op if it was never focused) and ends the edit without
    // committing. Escape must never persist a change (see #5: it silently
    // committed via the trailing blur → commit() before this fix). Clearing
    // the baseline before blur means the subsequent focusout's commit() sees
    // "never focused / already committed" and is a guaranteed no-op even if a
    // caller's own blur handling races this one. Blurring afterward exits
    // edit mode and returns focus/selection to their pre-edit state without
    // this module touching any host-owned "selected canvas object" state.
    function cancelEdit(el) {
      if (!el) return;
      var original = startOf(el);
      if (original === undefined) return; // never focused — nothing to cancel
      if (textOf(el) !== original) el.textContent = original;
      clearStart(el);
      if (typeof el.blur === "function") el.blur();
    }

    // commit persists el's current text if it differs from the captured start.
    // After a commit the new value becomes the baseline so the Enter→blur
    // sequence cannot double-POST the same value.
    function commit(el) {
      if (!el) return;
      var binding = el.getAttribute(FIELD_ATTR);
      if (!binding) return;
      var prev = startOf(el);
      var value = textOf(el);
      if (prev === undefined) return; // never focused or already committed
      if (value === prev) { clearStart(el); return; } // unchanged — no-op

      // Rebase the baseline BEFORE firing the async POST so a trailing blur
      // after Enter→blur() does not double-POST.
      if (hasWeakMap) starts.set(el, value);
      else { fallbackEl = el; fallbackStart = value; }

      // Resolve action + CSRF lazily at commit time so opts can be set after
      // install() and so the managed-form token is always the live value.
      var action = resolveAction(root, opts.action);
      var csrf = resolveCSRF(root, opts.csrfToken);

      // Optional pre-POST hook: lets hosts apply repaint-safe markup surgery
      // (e.g. muddy's persistRepaintSafe) without that logic living here.
      if (typeof opts.onCommit === "function") {
        try { opts.onCommit(el, value); } catch (e) { /* never break the commit */ }
      }

      var doFetch = opts.fetch || (typeof window !== "undefined" ? window.fetch : null);
      if (typeof doFetch !== "function") return;

      var meta = editMetaFromElement(el);
      var body = buildBody(binding, value, meta);
      var headers = { "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8" };
      if (csrf) headers["X-CSRF-Token"] = csrf;

      var fetchPromise = doFetch(action, {
        method: "POST",
        headers: headers,
        body: body,
        credentials: "same-origin",
      });

      // After a successful POST, hand the JSON result to the authoring runtime
      // so preview refresh, change selection, and save-chrome all update. No-op
      // gracefully when GoSXStudioAuthoringRuntime is absent.
      if (fetchPromise && typeof fetchPromise.then === "function") {
        fetchPromise.then(function (response) {
          if (!response || !response.ok) return;
          return response.json().catch(function () { return null; });
        }).then(function (result) {
          if (!result) return;
          var ar = typeof window !== "undefined" ? window.GoSXStudioAuthoringRuntime : null;
          if (ar && typeof ar.handleResult === "function") {
            try {
              ar.handleResult(result, { action: action, method: "POST", ok: true });
            } catch (e) { /* defensive */ }
          }
        }).catch(function () { /* network or parse errors are non-fatal */ });
      }
    }

    root.addEventListener("focusin", function (ev) {
      var el = editableField(ev.target, root);
      if (el) rememberStart(el);
    });

    root.addEventListener("focusout", function (ev) {
      var el = editableField(ev.target, root);
      if (el) commit(el);
    });

    root.addEventListener("keydown", function (ev) {
      var el = editableField(ev.target, root);
      if (!el) return;
      // #5 — Escape cancels the in-progress edit: revert to the pre-edit
      // value and end editing WITHOUT committing (previously nothing handled
      // Escape here, so the typed text silently persisted on the next blur —
      // "Escape doesn't cancel an inline edit" / a silent-commit trust wound).
      if (ev.key === "Escape") {
        if (typeof ev.preventDefault === "function") ev.preventDefault();
        cancelEdit(el);
        return;
      }
      if (ev.key !== "Enter" || ev.shiftKey === true) return;
      // Single-line field: Enter commits rather than inserting a newline.
      if (typeof ev.preventDefault === "function") ev.preventDefault();
      if (typeof el.blur === "function") el.blur(); // blur → focusout → commit()
      else commit(el);
    });
  }

  window.GoSXStudioInlineEditRuntime = {
    install: install,
    deriveKeys: deriveKeys,
    startPreviewTextEdit: startPreviewTextEdit,
    syncPreviewTextEdit: syncPreviewTextEdit,
    finishPreviewTextEdit: finishPreviewTextEdit,
    startPreviewTextSession: startPreviewTextSession,
    syncPreviewTextSession: syncPreviewTextSession,
    finishPreviewTextSession: finishPreviewTextSession,
    handlePreviewTextInputEvent: handlePreviewTextInputEvent,
    handlePreviewTextKeyEvent: handlePreviewTextKeyEvent,
    handlePreviewTextPasteEvent: handlePreviewTextPasteEvent,
    handlePreviewTextBlurEvent: handlePreviewTextBlurEvent,
    previewTextEditState: previewTextEditState
  };
})();
