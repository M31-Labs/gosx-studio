/* GoSX site editor.
 *
 * The canvas DOM is the source of truth in the browser. Text edits happen in
 * place, structural edits move real nodes, and every change serializes the
 * canvas back into a block document and posts it. Nothing reloads, so the
 * caret, the scroll position, and the user's train of thought all survive.
 */
(function () {
  "use strict";

  var root = document.querySelector("[data-editor]");
  if (!root) return;

  var pageId = root.getAttribute("data-page-id");
  var article = root.querySelector("[data-blocks]");
  var titleNode = root.querySelector("[data-page-title]");
  var saveNode = root.querySelector("[data-save-status]");
  var publishBtn = root.querySelector("[data-publish]");
  var undoBtn = root.querySelector("[data-undo]");
  var insertMenu = root.querySelector("[data-insert-menu]");
  var chip = root.querySelector(".ed-chip");

  var saveTimer = null;
  var saving = false;
  var pending = false;
  var history = [];
  var future = [];
  var redoBtn = root.querySelector("[data-redo]");
  var insertIndex = null;

  /* ---------- status ---------- */

  function status(state, text) {
    if (!saveNode) return;
    saveNode.setAttribute("data-save-status", state);
    saveNode.textContent = text;
  }

  /* ---------- serialize ---------- */

  function blockPayload(el) {
    var kind = el.getAttribute("data-block") || "paragraph";
    var textNode = el.querySelector("[data-text]");
    var payload = { kind: kind, text: textNode ? textNode.textContent.trim() : "" };

    if (kind === "heading") {
      var h = el.querySelector("[data-text]");
      payload.level = (h && h.getAttribute("data-level")) || "2";
    }
    if (kind === "button") {
      var href = el.querySelector("[data-href]");
      payload.url = href ? href.value.trim() : "";
    }
    if (kind === "image") {
      var src = el.querySelector("[data-src]");
      var alt = el.querySelector("[data-alt]");
      payload.url = src ? src.value.trim() : "";
      payload.alt = alt ? alt.value.trim() : "";
      payload.text = "";
    }
    if (kind === "form") payload.text = "";
    return payload;
  }

  function serialize() {
    var blocks = [];
    var nodes = article.querySelectorAll(".ed-block");
    for (var i = 0; i < nodes.length; i++) blocks.push(blockPayload(nodes[i]));
    return {
      title: titleNode ? titleNode.textContent.trim() : "",
      slug: fieldValue("pageSlug"),
      description: fieldValue("pageDescription"),
      blocks: blocks,
    };
  }

  function fieldValue(id) {
    var el = document.getElementById(id);
    return el ? el.value.trim() : "";
  }

  /* ---------- save ---------- */

  function queueSave() {
    status("dirty", "Saving…");
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(save, 700);
  }

  function save() {
    if (saving) {
      pending = true;
      return;
    }
    saving = true;
    var payload = serialize();

    fetch("/admin/api/pages/" + encodeURIComponent(pageId), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify(payload),
    })
      .then(function (response) {
        return response.json().catch(function () {
          throw new Error("unreadable");
        });
      })
      .then(function (result) {
        saving = false;
        if (!result.ok) {
          status("error", result.message || "That didn't save. Your text is still here — try again.");
          return;
        }
        status("saved", "All changes saved");
        if (result.slug) syncSlugHint(result.slug);
        if (pending) {
          pending = false;
          save();
        }
      })
      .catch(function () {
        saving = false;
        status("error", "Couldn't reach the server. Your text is still here — it'll save when you're back.");
      });
  }

  function syncSlugHint(slug) {
    var field = document.getElementById("pageSlug");
    if (field && field.value.trim() !== slug && document.activeElement !== field) {
      field.value = slug;
    }
  }

  /* ---------- history ---------- */

  function updateHistoryButtons() {
    if (undoBtn) undoBtn.disabled = history.length === 0;
    if (redoBtn) redoBtn.disabled = future.length === 0;
  }

  /* A new edit invalidates anything that was undone: that is what every
     editor a person has used does, and the alternative — a redo that
     resurrects a state from a different branch — feels like the page moved
     on its own. */
  function snapshot() {
    history.push(article.innerHTML);
    if (history.length > 50) history.shift();
    future = [];
    updateHistoryButtons();
  }

  function endTypingSession() { typingSession = null; }

  function undo() {
    if (!history.length) return;
    endTypingSession();
    future.push(article.innerHTML);
    article.innerHTML = history.pop();
    reindex();
    updateHistoryButtons();
    queueSave();
  }

  function redo() {
    if (!future.length) return;
    endTypingSession();
    history.push(article.innerHTML);
    article.innerHTML = future.pop();
    reindex();
    updateHistoryButtons();
    queueSave();
  }

  /* ---------- block construction ---------- */

  var TEMPLATES = {
    heading: 'Section heading',
    paragraph: 'Write something here.',
    quote: 'Something a customer said about you.',
    button: 'Get in touch',
    image: '',
  };

  function makeBlock(kind) {
    var wrapper = document.createElement("div");
    wrapper.className = "ed-block";
    wrapper.setAttribute("data-block", kind);
    wrapper.setAttribute("tabindex", "0");

    wrapper.appendChild(makeTools());
    if (kind === "heading") wrapper.appendChild(makeLevels("2"));
    wrapper.appendChild(makeInner(kind));
    wrapper.appendChild(makeInsertPoint());
    return wrapper;
  }

  function makeTools() {
    var tools = document.createElement("div");
    tools.className = "ed-block__tools";
    tools.setAttribute("contenteditable", "false");
    tools.appendChild(toolBtn("grab", "⠿", "Drag to move"));
    tools.appendChild(toolBtn("up", "↑", "Move up"));
    tools.appendChild(toolBtn("down", "↓", "Move down"));
    tools.appendChild(toolBtn("duplicate", "⧉", "Make a copy"));
    tools.appendChild(toolBtn("delete", "✕", "Delete"));
    return tools;
  }

  function toolBtn(action, glyph, label) {
    var btn = document.createElement("button");
    btn.className = "ed-tool";
    btn.type = "button";
    btn.setAttribute("data-tool", action);
    btn.title = label;
    btn.setAttribute("aria-label", label);
    btn.textContent = glyph;
    return btn;
  }

  function makeLevels(active) {
    var wrap = document.createElement("div");
    wrap.className = "ed-levels";
    wrap.setAttribute("contenteditable", "false");
    ["2", "3", "4"].forEach(function (level) {
      var btn = document.createElement("button");
      btn.className = "ed-level";
      btn.type = "button";
      btn.setAttribute("data-level", level);
      btn.setAttribute("aria-label", "Heading level " + level);
      if (level === active) btn.setAttribute("aria-pressed", "true");
      btn.textContent = "H" + level;
      wrap.appendChild(btn);
    });
    return wrap;
  }

  function makeInner(kind) {
    if (kind === "heading") {
      var h = document.createElement("h2");
      markText(h);
      h.setAttribute("data-level", "2");
      h.textContent = TEMPLATES.heading;
      return h;
    }
    if (kind === "quote") {
      var q = document.createElement("blockquote");
      markText(q);
      q.textContent = TEMPLATES.quote;
      return q;
    }
    if (kind === "button") {
      var row = document.createElement("span");
      row.className = "ed-button-row";
      var a = document.createElement("a");
      a.className = "site-button";
      a.href = "#";
      markText(a);
      a.textContent = TEMPLATES.button;
      row.appendChild(a);
      row.appendChild(inlineInput("href", "", "/contact", "Where this button goes"));
      return row;
    }
    if (kind === "image") {
      var fig = document.createElement("figure");
      fig.className = "ed-figure";
      var empty = document.createElement("div");
      empty.className = "ed-image-empty";
      empty.setAttribute("data-img", "true");
      empty.textContent = "No picture yet — upload one, or paste a link";
      fig.appendChild(empty);
      fig.appendChild(uploadControl());
      fig.appendChild(inlineInput("src", "", "or paste a link to one", "Image link"));
      fig.appendChild(inlineInput("alt", "", "Describe the picture for people who can't see it", "Image description"));
      return fig;
    }
    if (kind === "form") {
      var box = document.createElement("div");
      box.className = "site-form ed-form-preview";
      box.setAttribute("contenteditable", "false");
      box.innerHTML =
        '<label class="site-form__field"><span>Your name</span><input type="text" disabled></label>' +
        '<label class="site-form__field"><span>Your email</span><input type="email" disabled></label>' +
        '<label class="site-form__field"><span>Message</span><textarea rows="3" disabled></textarea></label>' +
        '<span class="site-button" aria-hidden="true">Send message</span>' +
        '<p class="ed-form-preview__note">Messages people send here arrive in Messages, in your admin area.</p>';
      return box;
    }
    var p = document.createElement("p");
    markText(p);
    p.textContent = TEMPLATES.paragraph;
    return p;
  }

  function markText(el) {
    el.setAttribute("data-text", "true");
    el.setAttribute("contenteditable", "true");
    el.setAttribute("spellcheck", "true");
  }

  function inlineInput(flag, value, placeholder, label) {
    var input = document.createElement("input");
    input.className = "ed-inline-input";
    input.type = "text";
    input.setAttribute("data-" + flag, "true");
    input.setAttribute("contenteditable", "false");
    input.value = value;
    input.placeholder = placeholder;
    input.setAttribute("aria-label", label);
    return input;
  }

  function uploadControl() {
    var label = document.createElement("label");
    label.className = "ed-upload";
    label.setAttribute("contenteditable", "false");
    var input = document.createElement("input");
    input.type = "file";
    input.accept = "image/png,image/jpeg,image/gif,image/webp";
    input.setAttribute("data-upload", "true");
    input.setAttribute("aria-label", "Upload a picture");
    var text = document.createElement("span");
    text.textContent = "Upload a picture";
    label.appendChild(input);
    label.appendChild(text);
    return label;
  }

  function makeInsertPoint() {
    var btn = document.createElement("button");
    btn.className = "ed-insert";
    btn.type = "button";
    btn.setAttribute("contenteditable", "false");
    btn.setAttribute("aria-label", "Add a section here");
    btn.textContent = "+";
    return btn;
  }

  function reindex() {
    var nodes = article.querySelectorAll(".ed-block");
    for (var i = 0; i < nodes.length; i++) {
      nodes[i].setAttribute("data-index", String(i));
      var insert = nodes[i].querySelector(".ed-insert");
      if (insert) insert.setAttribute("data-insert-at", String(i));
    }
  }

  function focusText(blockEl) {
    var target = blockEl.querySelector("[data-text]") || blockEl.querySelector("input");
    if (!target) return;
    target.focus();
    if (target.tagName === "INPUT") return;
    var range = document.createRange();
    range.selectNodeContents(target);
    var sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(range);
  }

  function addBlock(kind, atIndex) {
    snapshot();
    var el = makeBlock(kind);
    var nodes = article.querySelectorAll(".ed-block");
    if (atIndex === null || atIndex === undefined || atIndex >= nodes.length) {
      article.appendChild(el);
    } else {
      article.insertBefore(el, nodes[atIndex]);
    }
    reindex();
    focusText(el);
    queueSave();
  }

  /* ---------- interactions ---------- */

  root.addEventListener("click", function (event) {
    var tool = event.target.closest("[data-tool]");
    if (tool && tool.getAttribute("data-tool") !== "grab") {
      event.preventDefault();
      handleTool(tool);
      return;
    }

    var level = event.target.closest("[data-level][class*=ed-level]");
    if (level) {
      event.preventDefault();
      setLevel(level);
      return;
    }

    var insert = event.target.closest(".ed-insert");
    if (insert) {
      event.preventDefault();
      openInsertMenu(insert);
      return;
    }

    var add = event.target.closest("[data-add]");
    if (add) {
      event.preventDefault();
      addBlock(add.getAttribute("data-add"), null);
      return;
    }

    var choose = event.target.closest("[data-insert]");
    if (choose) {
      event.preventDefault();
      var kind = choose.getAttribute("data-insert");
      closeInsertMenu();
      addBlock(kind, insertIndex);
      return;
    }

    if (insertMenu && !insertMenu.hidden) closeInsertMenu();
  });

  function handleTool(tool) {
    var blockEl = tool.closest(".ed-block");
    if (!blockEl) return;
    var action = tool.getAttribute("data-tool");
    snapshot();

    if (action === "delete") {
      var next = blockEl.nextElementSibling || blockEl.previousElementSibling;
      blockEl.remove();
      if (next && next.classList.contains("ed-block")) focusText(next);
    } else if (action === "up") {
      var prev = blockEl.previousElementSibling;
      if (prev && prev.classList.contains("ed-block")) article.insertBefore(blockEl, prev);
    } else if (action === "down") {
      var after = blockEl.nextElementSibling;
      if (after && after.classList.contains("ed-block")) article.insertBefore(after, blockEl);
    } else if (action === "duplicate") {
      var copy = blockEl.cloneNode(true);
      blockEl.parentNode.insertBefore(copy, blockEl.nextSibling);
      focusText(copy);
    }
    reindex();
    queueSave();
  }

  function setLevel(button) {
    var blockEl = button.closest(".ed-block");
    if (!blockEl) return;
    var level = button.getAttribute("data-level");
    var current = blockEl.querySelector("[data-text]");
    if (!current || current.getAttribute("data-level") === level) return;

    snapshot();
    var replacement = document.createElement("h" + level);
    markText(replacement);
    replacement.setAttribute("data-level", level);
    replacement.textContent = current.textContent;
    current.parentNode.replaceChild(replacement, current);

    var buttons = blockEl.querySelectorAll(".ed-level");
    for (var i = 0; i < buttons.length; i++) {
      if (buttons[i].getAttribute("data-level") === level) {
        buttons[i].setAttribute("aria-pressed", "true");
      } else {
        buttons[i].removeAttribute("aria-pressed");
      }
    }
    focusText(blockEl);
    queueSave();
  }

  function openInsertMenu(button) {
    if (!insertMenu) return;
    var blockEl = button.closest(".ed-block");
    var nodes = Array.prototype.slice.call(article.querySelectorAll(".ed-block"));
    insertIndex = nodes.indexOf(blockEl) + 1;

    var rect = button.getBoundingClientRect();
    insertMenu.hidden = false;
    insertMenu.style.top = window.scrollY + rect.bottom + 6 + "px";
    insertMenu.style.left = window.scrollX + rect.left + "px";
  }

  function closeInsertMenu() {
    if (insertMenu) insertMenu.hidden = true;
    insertIndex = null;
  }

  /* text edits. A snapshot is taken at the START of a typing session — the
     first keystroke after focus lands in a field — not on every keystroke,
     so Undo steps back over what was just typed in one go rather than one
     character at a time, and history never fills with 50 near-identical
     states from a single sentence. */
  var typingSession = null;
  root.addEventListener("focusin", function (event) {
    if (event.target.closest("[data-text]") || event.target.matches("[data-href],[data-src],[data-alt]")) {
      typingSession = null;
    }
  });
  root.addEventListener("input", function (event) {
    var field = event.target.closest("[data-text]") || (event.target.matches("[data-href],[data-src],[data-alt]") ? event.target : null);
    if (field) {
      if (typingSession !== field) {
        typingSession = field;
        preTypingSnapshot(field);
      }
      if (event.target.matches("[data-src]")) refreshImage(event.target);
      queueSave();
      return;
    }
    if (event.target.matches("[data-meta]")) queueSave();
  });

  /* beforeinput fires before the DOM changes, which is the only moment the
     pre-edit markup can still be captured. */
  var pendingTypingField = null;
  root.addEventListener("beforeinput", function (event) {
    var field = event.target.closest("[data-text]");
    if (!field) return;
    if (typingSession !== field) {
      snapshot();
      pendingTypingField = field;
    }
  });
  function preTypingSnapshot(field) {
    // Inputs (href/src/alt) do not fire beforeinput on the article; take
    // the snapshot here for them. Contenteditable fields were snapshotted
    // in beforeinput, before the DOM changed.
    if (pendingTypingField === field) { pendingTypingField = null; return; }
    if (field.tagName === "INPUT") snapshot();
  }

  function refreshImage(input) {
    var fig = input.closest(".ed-figure");
    if (!fig) return;
    var current = fig.querySelector("[data-img]");
    var url = input.value.trim();
    var replacement;
    if (url) {
      replacement = document.createElement("img");
      replacement.setAttribute("data-img", "true");
      replacement.src = url;
      replacement.alt = "";
    } else {
      replacement = document.createElement("div");
      replacement.className = "ed-image-empty";
      replacement.setAttribute("data-img", "true");
      replacement.textContent = "No picture yet — upload one, or paste a link";
    }
    if (current) current.parentNode.replaceChild(replacement, current);
  }

  /* Uploads: the file goes up, the returned URL goes into the link field,
     and the preview updates — the same path a pasted link takes. */
  root.addEventListener("change", function (event) {
    var input = event.target.closest("[data-upload]");
    if (!input || !input.files || !input.files[0]) return;
    var label = input.closest(".ed-upload");
    var fig = input.closest(".ed-figure");
    var src = fig && fig.querySelector("[data-src]");
    var form = new FormData();
    form.append("file", input.files[0]);
    if (label) label.setAttribute("data-busy", "true");
    status("dirty", "Uploading picture…");
    fetch("/admin/api/upload", { method: "POST", credentials: "same-origin", body: form })
      .then(function (r) { return r.json(); })
      .then(function (result) {
        if (label) label.removeAttribute("data-busy");
        input.value = "";
        if (!result.ok) { status("error", result.message || "That picture didn't upload. Try again."); return; }
        if (src) {
          snapshot();
          src.value = result.url;
          refreshImage(src);
          queueSave();
        }
      })
      .catch(function () {
        if (label) label.removeAttribute("data-busy");
        status("error", "Couldn't reach the server. Try the upload again.");
      });
  });

  /* Enter inside a text block makes a new paragraph rather than a <div> soup. */
  root.addEventListener("keydown", function (event) {
    if (event.key === "Enter" && !event.shiftKey) {
      var text = event.target.closest("[data-text]");
      if (text && text.tagName !== "A") {
        event.preventDefault();
        var blockEl = text.closest(".ed-block");
        var nodes = Array.prototype.slice.call(article.querySelectorAll(".ed-block"));
        addBlock("paragraph", nodes.indexOf(blockEl) + 1);
        return;
      }
    }
    if (event.key === "Escape") closeInsertMenu();
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "z") {
      event.preventDefault();
      if (event.shiftKey) redo(); else undo();
    }
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "y") {
      event.preventDefault();
      redo();
    }
  });

  /* Paste as plain text, so pasted formatting never contaminates the page. */
  root.addEventListener("paste", function (event) {
    var text = event.target.closest("[data-text]");
    if (!text) return;
    event.preventDefault();
    var plain = (event.clipboardData || window.clipboardData).getData("text/plain");
    document.execCommand("insertText", false, plain);
  });

  if (undoBtn) {
    undoBtn.disabled = true;
    undoBtn.addEventListener("click", function () {
      undo();
    });
  }
  if (redoBtn) {
    redoBtn.disabled = true;
    redoBtn.addEventListener("click", function () {
      redo();
    });
  }

  /* ---------- drag to reorder ---------- */

  /* Pointer events rather than HTML5 drag-and-drop: the native API does not
     fire on most touch browsers, and a phone is where a section is most
     likely to be moved with a thumb. The grip captures the pointer, the
     block under the pointer is found with elementsFromPoint (skipping the
     one being dragged), and an accent line shows where it will land. */
  var dragging = null;
  var dropTarget = null;
  var dropBefore = false;
  var dropLine = null;

  function ensureDropLine() {
    if (dropLine) return dropLine;
    dropLine = document.createElement("div");
    dropLine.className = "ed-drop-line";
    dropLine.hidden = true;
    article.appendChild(dropLine);
    return dropLine;
  }

  function blockAtPoint(x, y) {
    var stack = document.elementsFromPoint(x, y);
    for (var i = 0; i < stack.length; i++) {
      var blockEl = stack[i].closest && stack[i].closest(".ed-block");
      if (blockEl && blockEl !== dragging && article.contains(blockEl)) return blockEl;
    }
    return null;
  }

  function showDropLine(target, before) {
    var line = ensureDropLine();
    dropTarget = target;
    dropBefore = before;
    line.style.top = (before ? target.offsetTop : target.offsetTop + target.offsetHeight) - 1 + "px";
    line.hidden = false;
  }

  function hideDropLine() {
    dropTarget = null;
    if (dropLine) dropLine.hidden = true;
  }

  function endDrag(commit) {
    if (!dragging) return;
    var moved = false;
    if (commit && dropTarget && dropTarget !== dragging) {
      if (dropBefore) article.insertBefore(dragging, dropTarget);
      else article.insertBefore(dragging, dropTarget.nextSibling);
      moved = true;
    }
    dragging.classList.remove("is-dragging");
    var el = dragging;
    dragging = null;
    hideDropLine();
    document.body.classList.remove("ed-is-dragging");
    if (moved) {
      reindex();
      el.focus();
      queueSave();
    } else {
      // Nothing changed, so the snapshot taken at pickup is noise.
      history.pop();
      updateHistoryButtons();
    }
  }

  root.addEventListener("pointerdown", function (event) {
    var grab = event.target.closest('[data-tool="grab"]');
    if (!grab) return;
    var blockEl = grab.closest(".ed-block");
    if (!blockEl) return;
    event.preventDefault();
    snapshot();
    dragging = blockEl;
    blockEl.classList.add("is-dragging");
    document.body.classList.add("ed-is-dragging");
    try { grab.setPointerCapture(event.pointerId); } catch (e) {}
  });

  root.addEventListener("pointermove", function (event) {
    if (!dragging) return;
    var target = blockAtPoint(event.clientX, event.clientY);
    if (!target) { hideDropLine(); return; }
    var rect = target.getBoundingClientRect();
    showDropLine(target, event.clientY < rect.top + rect.height / 2);
  });

  root.addEventListener("pointerup", function () { endDrag(true); });
  root.addEventListener("pointercancel", function () { endDrag(false); });
  window.addEventListener("blur", function () { endDrag(false); });

  if (publishBtn) {
    publishBtn.addEventListener("click", function () {
      publishBtn.disabled = true;
      status("dirty", "Publishing…");
      save();
      setTimeout(function () {
        fetch("/admin/api/pages/" + encodeURIComponent(pageId) + "/publish", {
          method: "POST",
          credentials: "same-origin",
        })
          .then(function (r) {
            return r.json();
          })
          .then(function (result) {
            publishBtn.disabled = false;
            if (!result.ok) {
              status("error", result.message || "That didn't publish. Try again.");
              return;
            }
            status("saved", "Published — your page is live");
            publishBtn.textContent = "Publish changes";
            if (chip) {
              chip.setAttribute("data-live", "true");
              chip.textContent = "Live";
            }
          })
          .catch(function () {
            publishBtn.disabled = false;
            status("error", "Couldn't reach the server. Try publishing again.");
          });
      }, 400);
    });
  }

  window.addEventListener("beforeunload", function (event) {
    if (saveNode && saveNode.getAttribute("data-save-status") === "dirty") {
      event.preventDefault();
      event.returnValue = "";
    }
  });

  reindex();
  status("saved", "All changes saved");
})();

/* ---------- Look: site-wide theme, live on the canvas ---------- */
(function () {
  "use strict";
  var root = document.querySelector("[data-editor]");
  var look = root && root.querySelector("[data-look]");
  if (!look) return;

  var presetsNode = look.querySelector("[data-look-presets]");
  var presets = { palettes: [], fonts: [] };
  try { presets = JSON.parse(presetsNode.textContent); } catch (e) {}

  var canvas = root.querySelector(".ed-canvas");
  var themeStyle = document.querySelector("style[data-site-theme]");
  var saveNode = root.querySelector("[data-save-status]");
  var accentInput = look.querySelector("[data-look-accent]");
  var timer = null;

  function current() {
    var palette = look.querySelector("[data-look-palette]:checked");
    var fonts = look.querySelector("[data-look-fonts]:checked");
    return {
      palette: palette ? palette.value : (presets.palettes[0] || {}).key,
      fonts: fonts ? fonts.value : (presets.fonts[0] || {}).key,
      accent: accentInput ? accentInput.value : "",
    };
  }

  function find(list, key) {
    for (var i = 0; i < list.length; i++) if (list[i].key === key) return list[i];
    return list[0];
  }

  /* Restyle the canvas immediately: the same custom properties the server
     emits, written straight onto the canvas root, so the page changes under
     the cursor before the save returns. */
  function apply(state) {
    var p = find(presets.palettes, state.palette) || {};
    var f = find(presets.fonts, state.fonts) || {};
    var accent = state.accent || p.accent;
    if (canvas) {
      canvas.style.setProperty("color-scheme", p.scheme || "light");
      canvas.style.setProperty("--site-ground", p.ground || "");
      canvas.style.setProperty("--site-surface", p.surface || "");
      canvas.style.setProperty("--site-ink", p.ink || "");
      canvas.style.setProperty("--site-muted", p.muted || "");
      canvas.style.setProperty("--site-rule", p.rule || "");
      canvas.style.setProperty("--site-accent", accent || "");
      canvas.style.setProperty("--site-font-display", f.display || "");
      canvas.style.setProperty("--site-font-body", f.body || "");
    }
    if (f.fontsUrl) ensureFontLink(f.fontsUrl);
  }

  function ensureFontLink(href) {
    if (document.querySelector('link[href="' + href + '"]')) return;
    var link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = href;
    document.head.appendChild(link);
  }

  function status(state, text) {
    if (!saveNode) return;
    saveNode.setAttribute("data-save-status", state);
    saveNode.textContent = text;
  }

  function save() {
    var state = current();
    status("dirty", "Saving the look…");
    fetch("/admin/api/theme", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify(state),
    })
      .then(function (r) { return r.json(); })
      .then(function (result) {
        if (!result.ok) { status("error", result.message || "The look didn't save. Try again."); return; }
        if (themeStyle && result.css) themeStyle.textContent = result.css;
        status("saved", "Look saved — it's live on every page");
      })
      .catch(function () { status("error", "Couldn't reach the server. Try again."); });
  }

  function changed() {
    apply(current());
    if (timer) clearTimeout(timer);
    timer = setTimeout(save, 500);
  }

  look.addEventListener("change", function (event) {
    if (event.target.matches("[data-look-palette]")) {
      // A new palette brings its own accent unless the owner has chosen one.
      var p = find(presets.palettes, event.target.value);
      if (accentInput && p && !accentInput.dataset.custom) accentInput.value = p.accent;
    }
    if (event.target.matches("[data-look-accent]")) accentInput.dataset.custom = "1";
    changed();
  });
  look.addEventListener("input", function (event) {
    if (event.target.matches("[data-look-accent]")) { accentInput.dataset.custom = "1"; apply(current()); }
  });
  var reset = look.querySelector("[data-look-accent-reset]");
  if (reset) reset.addEventListener("click", function () {
    var p = find(presets.palettes, current().palette);
    if (accentInput && p) { accentInput.value = p.accent; delete accentInput.dataset.custom; }
    changed();
  });
})();
