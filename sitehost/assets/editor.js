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

  function snapshot() {
    history.push(article.innerHTML);
    if (history.length > 50) history.shift();
    if (undoBtn) undoBtn.disabled = false;
  }

  function undo() {
    if (!history.length) return;
    article.innerHTML = history.pop();
    reindex();
    if (undoBtn) undoBtn.disabled = history.length === 0;
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
      empty.textContent = "No picture yet — paste a link below";
      fig.appendChild(empty);
      fig.appendChild(inlineInput("src", "", "Paste an image link", "Image link"));
      fig.appendChild(inlineInput("alt", "", "Describe the picture for people who can't see it", "Image description"));
      return fig;
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
    if (tool) {
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

  /* text edits */
  root.addEventListener("input", function (event) {
    if (event.target.closest("[data-text]") || event.target.matches("[data-href],[data-src],[data-alt],[data-meta]")) {
      if (event.target.matches("[data-src]")) refreshImage(event.target);
      queueSave();
    }
  });

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
      replacement.textContent = "No picture yet — paste a link below";
    }
    if (current) current.parentNode.replaceChild(replacement, current);
  }

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
      undo();
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
