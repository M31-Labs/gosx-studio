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

  var csrfMeta = document.querySelector('meta[name="csrf-token"]');
  var CSRF = csrfMeta ? csrfMeta.getAttribute("content") : "";
  /* This tab's name in the room, so its own saves don't bounce back. */
  var clientId = Math.random().toString(36).slice(2, 10);
  var pageId = root.getAttribute("data-page-id");
  var saveURL = root.getAttribute("data-save-url") || "/admin/api/pages/" + encodeURIComponent(pageId);
  var publishURL = root.getAttribute("data-publish-url") || saveURL + "/publish";
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
    var payload = { kind: kind, text: textNode ? serializeText(textNode) : "" };
    if (el.getAttribute("data-phone") === "hide") payload.phone = "hide";
    if (el.getAttribute("data-locked") === "true") payload.locked = "true";
    if (kind === "section") {
      var style = el.querySelector("[data-section-style]");
      payload.style = style ? style.value : "plain";
      var salign = el.querySelector("[data-section-align]");
      var swidth = el.querySelector("[data-section-width]");
      var sspace = el.querySelector("[data-section-space]");
      var simg = el.querySelector("[data-section-src]");
      payload.align = salign ? salign.value : "left";
      payload.width = swidth ? swidth.value : "normal";
      payload.space = sspace ? sspace.value : "normal";
      payload.url = simg ? simg.value.trim() : "";
    }
    var composite = el.querySelector("[data-composite]");
    if (composite) {
      payload.text = "";
      payload.fields = {};
      payload.items = [];
      var variant = composite.querySelector("[data-variant]");
      payload.variant = variant ? variant.value : "";
      Array.prototype.forEach.call(composite.querySelectorAll("[data-field]"), function (node) {
        if (node.closest("[data-item]")) return;
        payload.fields[node.getAttribute("data-field")] = fieldValue2(node);
      });
      Array.prototype.forEach.call(composite.querySelectorAll("[data-item]"), function (item) {
        var values = {};
        Array.prototype.forEach.call(item.querySelectorAll("[data-field]"), function (node) {
          values[node.getAttribute("data-field")] = fieldValue2(node);
        });
        payload.items.push(values);
      });
    }
    if (kind === "video") {
      var vurl = el.querySelector("[data-video-url]");
      payload.url = vurl ? vurl.value.trim() : "";
      payload.text = "";
    }
    if (kind === "columns") {
      var c1 = el.querySelector('[data-col="1"]');
      var c2 = el.querySelector('[data-col="2"]');
      payload.text = c1 ? serializeText(c1) : "";
      payload.text2 = c2 ? serializeText(c2) : "";
    }
    if (kind === "gallery") {
      payload.text = "";
      payload.images = [];
      var gitems = el.querySelectorAll(".ed-gallery__item");
      for (var g = 0; g < gitems.length; g++) {
        var gimg = gitems[g].querySelector("[data-gimg]");
        var galt = gitems[g].querySelector("[data-galt]");
        if (gimg && gimg.getAttribute("src")) payload.images.push({ url: gimg.getAttribute("src"), alt: galt ? galt.value.trim() : "" });
      }
    }

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
    if (kind === "form") {
      payload.text = "";
      var fsel = el.querySelector("[data-form-select]");
      payload.form = fsel ? fsel.value : "";
    }
    if (kind === "product") {
      payload.text = "";
      var psel = el.querySelector("[data-product-select]");
      payload.product = psel ? psel.value : "";
    }
    return payload;
  }

  /* A field's value: text with markers, an input's value, or a checkbox. */
  function fieldValue2(node) {
    if (node.tagName === "INPUT") {
      if (node.type === "checkbox") return node.checked ? "yes" : "";
      return node.value.trim();
    }
    return serializeText(node);
  }

  /* ---------- inline formatting <-> markers ---------- */

  /* The document stores **bold**, _italic_, and [label](address); the
     browser edits <strong>, <em>, and <a>. This walks the editable DOM back
     into markers. Lists serialize one item per line. */
  function serializeInline(node) {
    var out = "";
    var children = node.childNodes;
    for (var i = 0; i < children.length; i++) {
      var child = children[i];
      if (child.nodeType === 3) { out += child.nodeValue; continue; }
      if (child.nodeType !== 1) continue;
      var tag = child.tagName;
      if (tag === "BR") { out += "\n"; continue; }
      if (child.getAttribute && child.getAttribute("contenteditable") === "false") continue;
      var inner = serializeInline(child);
      if (tag === "STRONG" || tag === "B") out += inner ? "**" + inner + "**" : "";
      else if (tag === "EM" || tag === "I") out += inner ? "_" + inner + "_" : "";
      else if (tag === "A") out += inner ? "[" + inner + "](" + (child.getAttribute("href") || "") + ")" : "";
      else if (tag === "DIV" || tag === "P") out += (out && !/\n$/.test(out) ? "\n" : "") + inner;
      else out += inner;
    }
    return out;
  }

  function serializeText(textNode) {
    if (textNode.hasAttribute("data-list")) {
      var lines = [];
      var items = textNode.querySelectorAll("li");
      for (var i = 0; i < items.length; i++) {
        var line = serializeInline(items[i]).replace(/\u00a0/g, " ").trim();
        if (line) lines.push(line);
      }
      return lines.join("\n");
    }
    return serializeInline(textNode).replace(/\u00a0/g, " ").trim();
  }

  function serialize() {
    var blocks = [];
    var nodes = article.querySelectorAll(".ed-block");
    for (var i = 0; i < nodes.length; i++) blocks.push(blockPayload(nodes[i]));
    return {
      title: titleNode ? titleNode.textContent.trim() : "",
      slug: fieldValue("pageSlug"),
      description: fieldValue("pageDescription"),
      excerpt: fieldValue("pageExcerpt"),
      tags: fieldValue("pageTags"),
      author: fieldValue("pageAuthor"),
      publishAt: localToISO(fieldValue("pagePublishAt")),
      navParent: fieldValue("pageParent"),
      blocks: blocks,
    };
  }

  function fieldValue(id) {
    var el = document.getElementById(id);
    return el ? el.value.trim() : "";
  }

  /* Date pickers: the server stores UTC, the owner thinks in local time. */
  function localToISO(local) {
    if (!local) return "";
    var d = new Date(local);
    return isNaN(d.getTime()) ? "" : d.toISOString();
  }

  function pad(n) { return (n < 10 ? "0" : "") + n; }

  function isoToLocal(iso) {
    if (!iso) return "";
    var d = new Date(iso);
    if (isNaN(d.getTime())) return "";
    return d.getFullYear() + "-" + pad(d.getMonth() + 1) + "-" + pad(d.getDate()) + "T" + pad(d.getHours()) + ":" + pad(d.getMinutes());
  }

  Array.prototype.forEach.call(root.querySelectorAll("input[type=datetime-local][data-iso]"), function (input) {
    input.value = isoToLocal(input.getAttribute("data-iso"));
  });

  /* The line under a post's title mirrors the sidebar as the owner types. */
  var MONTHS = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
  function refreshPostMeta() {
    var line = article && article.querySelector(".ed-post-meta");
    if (!line) return;
    var when = fieldValue("pagePublishAt");
    var dateText = line.getAttribute("data-default-date") || "";
    if (when) {
      var d = new Date(when);
      if (!isNaN(d.getTime())) dateText = d.getDate() + " " + MONTHS[d.getMonth()] + " " + d.getFullYear();
    }
    var text = dateText;
    var author = fieldValue("pageAuthor");
    if (author) text += " · by " + author;
    var tags = fieldValue("pageTags").split(",").map(function (t) { return t.trim(); }).filter(Boolean);
    if (tags.length) text += " · in " + tags.join(", ");
    line.textContent = text;
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

    fetch(saveURL, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-CSRF-Token": CSRF, "X-Editor-Client": clientId },
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
        renderChecks(result.checks);
        if (pending) {
          pending = false;
          save();
        } else if (refreshWanted && !editingHere()) {
          refreshCanvas();
        }
      })
      .catch(function () {
        saving = false;
        status("error", "Couldn't reach the server. Your text is still here — it'll save when you're back.");
      });
  }

  function renderChecks(checks) {
    var list = root.querySelector("[data-checks-list]");
    if (!list || !Array.isArray(checks)) return;
    list.textContent = "";
    if (!checks.length) {
      var ok = document.createElement("li");
      ok.className = "ed-check ed-check--ok";
      ok.textContent = "Looks good. Nothing to fix.";
      list.appendChild(ok);
      return;
    }
    checks.forEach(function (text) {
      var li = document.createElement("li");
      li.className = "ed-check";
      li.textContent = text;
      list.appendChild(li);
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
    wrapper.appendChild(makeBadge());
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
    tools.appendChild(toolBtn("phone", "📱", "Hide on phones"));
    tools.appendChild(toolBtn("lock", "🔒", "Lock: only admins can change this"));
    tools.appendChild(toolBtn("delete", "✕", "Delete"));
    return tools;
  }

  /* Editors see locked blocks but cannot touch them. */
  var canLock = root.getAttribute("data-can-lock") === "true";
  function freezeLocked() {
    if (canLock) return;
    Array.prototype.forEach.call(root.querySelectorAll('.ed-block[data-locked="true"]'), function (blockEl) {
      Array.prototype.forEach.call(blockEl.querySelectorAll("[contenteditable=true]"), function (el) { el.setAttribute("contenteditable", "false"); });
      Array.prototype.forEach.call(blockEl.querySelectorAll("input, select, textarea, button.ed-tool, .ed-insert, .ed-level"), function (el) { el.disabled = true; });
      blockEl.classList.add("ed-block--frozen");
    });
  }
  freezeLocked();

  function makeBadge() {
    var badge = document.createElement("span");
    badge.className = "ed-block__badge";
    badge.setAttribute("contenteditable", "false");
    badge.textContent = "Hidden on phones";
    return badge;
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
      fig.appendChild(libraryButton());
      fig.appendChild(inlineInput("src", "", "or paste a link to one", "Image link"));
      fig.appendChild(inlineInput("alt", "", "Describe the picture for people who can't see it", "Image description"));
      return fig;
    }
    if (kind === "video") {
      var vid = document.createElement("div");
      vid.className = "ed-video";
      vid.innerHTML =
        '<div class="ed-image-empty" data-video-preview="true">Paste a YouTube or Vimeo link below</div>' +
        '<input class="ed-inline-input" type="text" data-video-url="true" value="" placeholder="https://youtube.com/watch?v=…" aria-label="Video link" contenteditable="false">';
      return vid;
    }
    if (kind === "columns") {
      var cols = document.createElement("div");
      cols.className = "site-columns ed-columns";
      var left = document.createElement("div"); left.className = "site-columns__col"; markText(left); left.setAttribute("data-col", "1"); left.textContent = "Left column";
      var right = document.createElement("div"); right.className = "site-columns__col"; markText(right); right.setAttribute("data-col", "2"); right.textContent = "Right column";
      cols.appendChild(left); cols.appendChild(right);
      return cols;
    }
    if (kind === "gallery") {
      var gal = document.createElement("div");
      gal.className = "ed-gallery";
      gal.setAttribute("data-picker-target", "gallery");
      gal.setAttribute("contenteditable", "false");
      gal.innerHTML =
        '<div class="site-gallery ed-gallery__grid" data-gallery-items="true"></div>' +
        '<div class="ed-gallery__controls">' +
        '<label class="ed-upload"><input type="file" multiple accept="image/png,image/jpeg,image/gif,image/webp" data-upload="true" aria-label="Add pictures"><span>Add pictures</span></label>' +
        '<button type="button" class="ed-library-btn" data-library="true">Choose from your pictures</button>' +
        '</div>';
      return gal;
    }
    if (kind === "list") {
      var ul = document.createElement("ul");
      ul.className = "site-list";
      markText(ul);
      ul.setAttribute("data-list", "true");
      var li = document.createElement("li");
      li.textContent = "First point";
      ul.appendChild(li);
      return ul;
    }
    if (kind === "divider") {
      var hr = document.createElement("hr");
      hr.className = "site-divider";
      hr.setAttribute("contenteditable", "false");
      return hr;
    }
    if (kind === "section") {
      var bar = document.createElement("div");
      bar.className = "ed-section-bar";
      bar.setAttribute("data-section", "plain");
      bar.setAttribute("contenteditable", "false");
      bar.innerHTML =
        '<span class="ed-section-bar__label">New section</span>' +
        '<label class="ed-section-bar__style"><span>Background</span>' +
        '<select data-section-style="true" aria-label="Section background">' +
        '<option value="plain" selected>Plain</option><option value="tinted">Tinted</option><option value="accent">Accent colour</option>' +
        '</select></label>';
      return bar;
    }
    if (kind === "product") {
      var pbox = document.createElement("div");
      pbox.className = "ed-product";
      pbox.setAttribute("contenteditable", "false");
      if (!PRODUCTS.length) {
        pbox.className = "ed-image-empty";
        pbox.textContent = "No products yet. Add one under Shop, then pick it here.";
        return pbox;
      }
      var card = document.createElement("ul");
      card.className = "site-products site-products--single";
      card.setAttribute("data-product-card", "true");
      var pbar = document.createElement("div");
      pbar.className = "ed-form__bar";
      var plabel = document.createElement("label");
      plabel.textContent = "Which product: ";
      var pselect = document.createElement("select");
      pselect.className = "ed-inline-select";
      pselect.setAttribute("data-product-select", "true");
      pselect.setAttribute("aria-label", "Which product");
      PRODUCTS.forEach(function (preset) {
        var opt = document.createElement("option");
        opt.value = preset.ref; opt.textContent = preset.name;
        pselect.appendChild(opt);
      });
      plabel.appendChild(pselect);
      var pedit = document.createElement("a");
      pedit.setAttribute("data-product-edit", "true"); pedit.target = "_blank"; pedit.rel = "noopener"; pedit.textContent = "Edit the product";
      pbar.appendChild(plabel); pbar.appendChild(pedit);
      pbox.appendChild(card); pbox.appendChild(pbar);
      refreshProductPreview(pbox);
      return pbox;
    }
    if (kind === "form") {
      var box = document.createElement("div");
      box.className = "ed-form";
      box.setAttribute("contenteditable", "false");
      var fields = document.createElement("div");
      fields.className = "site-form ed-form-preview";
      fields.setAttribute("data-form-fields", "true");
      var bar = document.createElement("div");
      bar.className = "ed-form__bar";
      var label = document.createElement("label");
      label.textContent = "Which form: ";
      var select = document.createElement("select");
      select.className = "ed-inline-select";
      select.setAttribute("data-form-select", "true");
      select.setAttribute("aria-label", "Which form");
      FORMS.forEach(function (preset) {
        var opt = document.createElement("option");
        opt.value = preset.ref; opt.textContent = preset.name;
        select.appendChild(opt);
      });
      label.appendChild(select);
      var edit = document.createElement("a");
      edit.setAttribute("data-form-edit", "true"); edit.target = "_blank"; edit.rel = "noopener"; edit.textContent = "Change the questions";
      var build = document.createElement("a");
      build.href = "/admin/forms"; build.target = "_blank"; build.rel = "noopener"; build.textContent = "Build a new form";
      bar.appendChild(label); bar.appendChild(edit); bar.appendChild(build);
      box.appendChild(fields); box.appendChild(bar);
      refreshFormPreview(box);
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

  function libraryButton() {
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "ed-library-btn";
    btn.setAttribute("data-library", "true");
    btn.setAttribute("contenteditable", "false");
    btn.textContent = "Choose from your pictures";
    return btn;
  }

  function galleryItem(url, alt) {
    var fig = document.createElement("figure");
    fig.className = "site-gallery__item ed-gallery__item";
    fig.innerHTML =
      '<img src="" alt="" data-gimg="true" loading="lazy">' +
      '<input class="ed-inline-input" type="text" data-galt="true" value="" placeholder="Describe this picture" aria-label="Picture description">' +
      '<button type="button" class="ed-tool ed-gallery__remove" data-gremove="true" aria-label="Remove this picture">✕</button>';
    fig.querySelector("[data-gimg]").setAttribute("src", url);
    fig.querySelector("[data-galt]").value = alt || "";
    return fig;
  }

  function addToGallery(gallery, url) {
    var grid = gallery.querySelector("[data-gallery-items]");
    if (grid) grid.appendChild(galleryItem(url, ""));
  }

  /* Mirrors videoEmbedURL on the server, so the canvas can preview a link
     the moment it is pasted. */
  function videoEmbed(raw) {
    raw = (raw || "").trim();
    if (!raw) return "";
    if (raw.indexOf("://") < 0) raw = "https://" + raw;
    var a;
    try { a = new URL(raw); } catch (e) { return ""; }
    var host = a.hostname.toLowerCase().replace(/^www\./, "");
    var path = a.pathname.replace(/^\/+|\/+$/g, "");
    var yt = /^[A-Za-z0-9_-]{6,20}$/;
    if (host === "youtube.com" || host === "m.youtube.com" || host === "youtube-nocookie.com") {
      var id = a.searchParams.get("v") || "";
      var parts = path.split("/");
      if (!id && parts.length === 2 && ["shorts", "embed", "live"].indexOf(parts[0]) >= 0) id = parts[1];
      if (yt.test(id)) return "https://www.youtube-nocookie.com/embed/" + id;
    }
    if (host === "youtu.be" && yt.test(path)) return "https://www.youtube-nocookie.com/embed/" + path;
    if (host === "vimeo.com" || host === "player.vimeo.com") {
      var vid = path.split("/").pop();
      if (/^[0-9]{5,15}$/.test(vid)) return "https://player.vimeo.com/video/" + vid;
    }
    return "";
  }

  function refreshVideo(input) {
    var box = input.closest(".ed-video");
    if (!box) return;
    var preview = box.querySelector("[data-video-preview]");
    var embed = videoEmbed(input.value);
    var next;
    if (embed) {
      next = document.createElement("div");
      next.className = "site-video";
      next.setAttribute("data-video-preview", "true");
      var frame = document.createElement("iframe");
      frame.src = embed; frame.title = "Video"; frame.loading = "lazy"; frame.setAttribute("allowfullscreen", "allowfullscreen");
      next.appendChild(frame);
    } else {
      next = document.createElement("div");
      next.className = "ed-image-empty";
      next.setAttribute("data-video-preview", "true");
      next.textContent = input.value.trim() ? "That doesn't look like a YouTube or Vimeo link" : "Paste a YouTube or Vimeo link below";
    }
    if (preview) preview.parentNode.replaceChild(next, preview);
  }

  /* ---------- forms on the canvas ---------- */

  var FORMS = [];
  try {
    var formsNode = document.querySelector("[data-forms-presets]");
    FORMS = formsNode ? JSON.parse(formsNode.textContent) : [];
  } catch (e) { FORMS = []; }

  function findForm(ref) {
    for (var i = 0; i < FORMS.length; i++) if (FORMS[i].ref === ref) return FORMS[i];
    return FORMS[0] || { ref: "contact", name: "Contact form", button: "Send", fields: [], edit: "/admin/messages" };
  }

  /* Redraws a form block's disabled preview from the chosen form. */
  function refreshFormPreview(box) {
    var select = box.querySelector("[data-form-select]");
    var holder = box.querySelector("[data-form-fields]");
    if (!select || !holder) return;
    var preset = findForm(select.value);
    holder.textContent = "";
    (preset.fields || []).forEach(function (field) {
      var label = document.createElement("label");
      var text = field.label + (field.required ? " *" : "");
      if (field.kind === "checkbox") {
        label.className = "site-form__check";
        var cb = document.createElement("input"); cb.type = "checkbox"; cb.disabled = true;
        var span = document.createElement("span"); span.textContent = text;
        label.appendChild(cb); label.appendChild(span);
      } else {
        label.className = "site-form__field";
        var span2 = document.createElement("span"); span2.textContent = text;
        var control;
        if (field.kind === "textarea") { control = document.createElement("textarea"); control.rows = 3; }
        else if (field.kind === "select") {
          control = document.createElement("select");
          var opt = document.createElement("option"); opt.textContent = "Choose…"; control.appendChild(opt);
        } else { control = document.createElement("input"); control.type = field.kind === "email" ? "email" : field.kind === "date" ? "date" : field.kind === "phone" ? "tel" : "text"; }
        control.disabled = true;
        label.appendChild(span2); label.appendChild(control);
      }
      holder.appendChild(label);
    });
    var button = document.createElement("span");
    button.className = "site-button"; button.setAttribute("aria-hidden", "true"); button.textContent = preset.button || "Send";
    holder.appendChild(button);
    var edit = box.querySelector("[data-form-edit]");
    if (edit) edit.href = preset.edit || "/admin/forms";
  }

  root.addEventListener("change", function (event) {
    if (event.target.matches("[data-form-select]")) {
      snapshot();
      refreshFormPreview(event.target.closest(".ed-form"));
      queueSave();
    }
    if (event.target.matches("[data-product-select]")) {
      snapshot();
      refreshProductPreview(event.target.closest(".ed-product"));
      queueSave();
    }
  });

  /* ---------- products on the canvas ---------- */

  var PRODUCTS = [];
  try {
    var productsNode = document.querySelector("[data-products-presets]");
    PRODUCTS = productsNode ? JSON.parse(productsNode.textContent) : [];
  } catch (e) { PRODUCTS = []; }

  function findProduct(ref) {
    for (var i = 0; i < PRODUCTS.length; i++) if (PRODUCTS[i].ref === ref) return PRODUCTS[i];
    return PRODUCTS[0];
  }

  function refreshProductPreview(box) {
    var select = box.querySelector("[data-product-select]");
    var card = box.querySelector("[data-product-card]");
    if (!select || !card) return;
    var preset = findProduct(select.value);
    if (!preset) return;
    card.textContent = "";
    var li = document.createElement("li");
    li.className = "site-product-card site-product-card--inline";
    var a = document.createElement("a");
    a.className = "site-product-card__link"; a.href = preset.href; a.tabIndex = -1;
    var pic = document.createElement("span");
    pic.className = "site-product-card__picture";
    if (preset.image) { var img = document.createElement("img"); img.src = preset.image; img.alt = ""; pic.appendChild(img); }
    else { var blank = document.createElement("span"); blank.className = "site-product-card__blank"; pic.appendChild(blank); }
    var name = document.createElement("span"); name.className = "site-product-card__name"; name.textContent = preset.name;
    var price = document.createElement("span"); price.className = "site-product-card__price"; price.textContent = preset.price;
    a.appendChild(pic); a.appendChild(name); a.appendChild(price);
    li.appendChild(a); card.appendChild(li);
    var edit = box.querySelector("[data-product-edit]");
    if (edit) edit.href = preset.edit || "/admin/shop";
  }

  /* ---------- the picture picker ---------- */

  var picker = null;
  var pickerTarget = null;

  function closePicker() {
    if (picker) picker.remove();
    picker = null;
    pickerTarget = null;
  }

  function openPicker(button) {
    closePicker();
    pickerTarget = button.closest(".ed-figure") || button.closest("[data-picker-target]");
    picker = document.createElement("div");
    picker.className = "ed-menu ed-picker";
    picker.setAttribute("data-picker", "true");
    picker.innerHTML = '<p class="ed-picker__hint">Loading your pictures…</p>';
    var rect = button.getBoundingClientRect();
    picker.style.top = window.scrollY + rect.bottom + 6 + "px";
    picker.style.left = window.scrollX + rect.left + "px";
    // Inside the editor root: the click listener that applies a pick is
    // bound there, and a picker parked on <body> would never reach it.
    root.appendChild(picker);

    fetch("/admin/api/media", { credentials: "same-origin" })
      .then(function (r) { return r.json(); })
      .then(function (result) {
        if (!picker) return;
        picker.innerHTML = "";
        var pictures = (result && result.pictures) || [];
        if (!pictures.length) {
          picker.innerHTML = '<p class="ed-picker__hint">No pictures yet. Upload one and it will be here next time.</p>';
          return;
        }
        pictures.forEach(function (pic) {
          var item = document.createElement("button");
          item.type = "button";
          item.className = "ed-picker__item";
          item.setAttribute("data-pick", pic.url);
          item.title = pic.width + " × " + pic.height;
          var img = document.createElement("img");
          img.src = pic.thumb;
          img.alt = "";
          img.loading = "lazy";
          item.appendChild(img);
          picker.appendChild(item);
        });
      })
      .catch(function () {
        if (picker) picker.innerHTML = '<p class="ed-picker__hint">Couldn\'t load your pictures. Try again.</p>';
      });
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

  var SERVER_KINDS = (root.getAttribute("data-server-kinds") || "").split(",");

  function placeBlock(el, atIndex) {
    var nodes = article.querySelectorAll(".ed-block");
    if (atIndex === null || atIndex === undefined || atIndex >= nodes.length) {
      article.appendChild(el);
    } else {
      article.insertBefore(el, nodes[atIndex]);
    }
    reindex();
    freezeLocked();
    focusText(el);
    el.scrollIntoView({ block: "nearest", behavior: "smooth" });
    queueSave();
  }

  /* Ready-made sections come from the server, rendered exactly as they
     will look; the browser only wraps them in the usual tools. */
  function addBlock(kind, atIndex) {
    if (SERVER_KINDS.indexOf(kind) >= 0) {
      status("dirty", "Adding…");
      fetch("/admin/api/blocks/" + encodeURIComponent(kind), { credentials: "same-origin" })
        .then(function (r) { if (!r.ok) throw new Error("no block"); return r.text(); })
        .then(function (html) {
          snapshot();
          var wrapper = document.createElement("div");
          wrapper.className = "ed-block";
          wrapper.setAttribute("data-block", kind);
          wrapper.setAttribute("tabindex", "0");
          wrapper.appendChild(makeTools());
          wrapper.appendChild(makeBadge());
          var holder = document.createElement("div");
          holder.innerHTML = html;
          while (holder.firstChild) wrapper.appendChild(holder.firstChild);
          wrapper.appendChild(makeInsertPoint());
          placeBlock(wrapper, atIndex);
        })
        .catch(function () { status("error", "Couldn't add that. Try again."); });
      return;
    }
    snapshot();
    placeBlock(makeBlock(kind), atIndex);
  }

  function addItem(button) {
    var kind = button.getAttribute("data-item-add");
    var composite = button.closest("[data-composite]");
    var list = composite && composite.querySelector("[data-items]");
    if (!list) return;
    fetch("/admin/api/blocks/" + encodeURIComponent(kind) + "/item", { credentials: "same-origin" })
      .then(function (r) { if (!r.ok) throw new Error("no item"); return r.text(); })
      .then(function (html) {
        snapshot();
        var holder = document.createElement("div");
        holder.innerHTML = html;
        var item = holder.firstElementChild;
        if (!item) return;
        list.appendChild(item);
        var first = item.querySelector("[data-text]");
        if (first) first.focus();
        queueSave();
      })
      .catch(function () { status("error", "Couldn't add that. Try again."); });
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

    var itemAdd = event.target.closest("[data-item-add]");
    if (itemAdd) {
      event.preventDefault();
      addItem(itemAdd);
      return;
    }

    var itemRemove = event.target.closest("[data-item-remove]");
    if (itemRemove) {
      event.preventDefault();
      var item = itemRemove.closest("[data-item]");
      if (item) {
        snapshot();
        item.parentNode.removeChild(item);
        queueSave();
      }
      return;
    }

    var library = event.target.closest("[data-library]");
    if (library) {
      event.preventDefault();
      openPicker(library);
      return;
    }

    var pick = event.target.closest("[data-pick]");
    if (pick && pickerTarget) {
      event.preventDefault();
      snapshot();
      if (pickerTarget.classList.contains("ed-gallery")) {
        addToGallery(pickerTarget, pick.getAttribute("data-pick"));
      } else {
        var src = pickerTarget.querySelector("[data-src]");
        if (src) { src.value = pick.getAttribute("data-pick"); refreshImage(src); }
      }
      queueSave();
      closePicker();
      return;
    }

    var gremove = event.target.closest("[data-gremove]");
    if (gremove) {
      event.preventDefault();
      snapshot();
      var item = gremove.closest(".ed-gallery__item");
      if (item) item.remove();
      queueSave();
      return;
    }
    if (picker && !event.target.closest("[data-picker]")) closePicker();

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
    } else if (action === "lock") {
      if (root.getAttribute("data-can-lock") !== "true") return;
      var wasLocked = blockEl.getAttribute("data-locked") === "true";
      if (wasLocked) blockEl.removeAttribute("data-locked"); else blockEl.setAttribute("data-locked", "true");
      tool.setAttribute("aria-pressed", wasLocked ? "false" : "true");
      tool.title = wasLocked ? "Lock: only admins can change this" : "Unlock for editors";
      tool.setAttribute("aria-label", tool.title);
      status("dirty", wasLocked ? "Unlocked: editors can change this again" : "Locked: editors can see this but not change it");
    } else if (action === "phone") {
      var hidden = blockEl.getAttribute("data-phone") === "hide";
      if (hidden) blockEl.removeAttribute("data-phone"); else blockEl.setAttribute("data-phone", "hide");
      tool.setAttribute("aria-pressed", hidden ? "false" : "true");
      tool.title = hidden ? "Hide on phones" : "Show on phones again";
      tool.setAttribute("aria-label", tool.title);
      status("dirty", hidden ? "Shown on phones again" : "Hidden on phones. Visitors on a computer still see it.");
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
  root.addEventListener("change", function (event) {
    var style = event.target.closest("[data-section-style]");
    if (style) {
      snapshot();
      var bar = style.closest(".ed-section-bar");
      if (bar) bar.setAttribute("data-section", style.value);
      queueSave();
      return;
    }
    if (event.target.matches("[data-section-align],[data-section-width],[data-section-space]")) { snapshot(); queueSave(); return; }
    var variant = event.target.closest("[data-variant]");
    if (variant) {
      snapshot();
      var composite = variant.closest("[data-composite]");
      if (composite) {
        var kind = composite.getAttribute("data-composite");
        composite.className = composite.className.replace(new RegExp("\\bsite-" + kind + "--[a-z]+"), "site-" + kind + "--" + variant.value);
        composite.setAttribute("data-variant-value", variant.value);
      }
      queueSave();
      return;
    }
    if (event.target.matches("input[type=checkbox][data-field]")) {
      snapshot();
      var card = event.target.closest("[data-item]");
      if (card) card.classList.toggle("site-pricing__card--highlight", event.target.checked);
      queueSave();
    }
  });

  root.addEventListener("input", function (event) {
    if (event.target.matches("[data-video-url]")) { refreshVideo(event.target); queueSave(); return; }
    if (event.target.matches("[data-galt]")) { queueSave(); return; }
    var field = event.target.closest("[data-text]") || (event.target.matches("[data-href],[data-src],[data-alt],input[data-field]") ? event.target : null);
    if (field) {
      if (typingSession !== field) {
        typingSession = field;
        preTypingSnapshot(field);
      }
      if (event.target.matches("[data-src]")) refreshImage(event.target);
      queueSave();
      return;
    }
    if (event.target.matches("[data-meta]")) { refreshPostMeta(); queueSave(); }
  });

  /* A date picker or a drop-down commits on change, not on every keystroke. */
  root.addEventListener("change", function (event) {
    if (event.target.matches("input[type=datetime-local][data-meta]")) { refreshPostMeta(); queueSave(); }
    if (event.target.matches("select[data-meta]")) { snapshot(); queueSave(); }
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
  function uploadOne(file) {
    var form = new FormData();
    form.append("file", file);
    return fetch("/admin/api/upload", { method: "POST", credentials: "same-origin", headers: { "X-CSRF-Token": CSRF }, body: form })
      .then(function (r) { return r.json(); });
  }

  root.addEventListener("change", function (event) {
    var input = event.target.closest("[data-upload]");
    if (!input || !input.files || !input.files[0]) return;
    var label = input.closest(".ed-upload");
    var gallery = input.closest(".ed-gallery");
    var fig = input.closest(".ed-figure");
    var src = fig && fig.querySelector("[data-src]");
    var files = Array.prototype.slice.call(input.files);
    if (label) label.setAttribute("data-busy", "true");
    status("dirty", files.length > 1 ? "Uploading " + files.length + " pictures…" : "Uploading picture…");

    var chain = Promise.resolve();
    var failed = "";
    files.forEach(function (file) {
      chain = chain.then(function () {
        return uploadOne(file).then(function (result) {
          if (!result.ok) { failed = result.message || "A picture didn't upload."; return; }
          snapshot();
          if (gallery) addToGallery(gallery, result.url);
          else if (src) { src.value = result.url; refreshImage(src); }
        });
      });
    });
    chain
      .then(function () {
        if (label) label.removeAttribute("data-busy");
        input.value = "";
        if (failed) { status("error", failed); }
        queueSave();
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
      if (text && text.tagName !== "A" && !text.hasAttribute("data-list")) {
        event.preventDefault();
        var blockEl = text.closest(".ed-block");
        var nodes = Array.prototype.slice.call(article.querySelectorAll(".ed-block"));
        addBlock("paragraph", nodes.indexOf(blockEl) + 1);
        return;
      }
    }
    if (event.key === "Escape") { closeInsertMenu(); closePicker(); }
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "z") {
      event.preventDefault();
      if (event.shiftKey) redo(); else undo();
    }
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "y") {
      event.preventDefault();
      redo();
    }
  });

  /* ---------- the formatting bubble ---------- */

  /* Select some text and a small bar appears above it: bold, italic, link.
     execCommand is old but it is what every browser's contenteditable
     understands, and the result is serialized to markers, not stored as
     HTML, so what it produces can never reach the page directly. */
  var bubble = document.createElement("div");
  bubble.className = "ed-bubble";
  bubble.hidden = true;
  bubble.setAttribute("contenteditable", "false");
  bubble.innerHTML =
    '<button type="button" data-fmt="bold" title="Bold (Ctrl+B)"><b>B</b></button>' +
    '<button type="button" data-fmt="italic" title="Italic (Ctrl+I)"><i>I</i></button>' +
    '<button type="button" data-fmt="link" title="Link">Link</button>' +
    '<button type="button" data-fmt="clear" title="Remove formatting">Clear</button>' +
    '<span class="ed-bubble__link" hidden><input type="url" placeholder="https:// or /page" aria-label="Link address"><button type="button" data-fmt="apply-link">Add</button></span>';
  root.appendChild(bubble);
  var bubbleLink = bubble.querySelector(".ed-bubble__link");
  var bubbleInput = bubble.querySelector("input");
  var savedRange = null;

  function selectionInText() {
    var sel = window.getSelection();
    if (!sel || sel.rangeCount === 0 || sel.isCollapsed) return null;
    var node = sel.anchorNode;
    if (!node) return null;
    var el = node.nodeType === 1 ? node : node.parentElement;
    var text = el && el.closest("[data-text]");
    if (!text || text.tagName === "A" || !article.contains(text)) return null;
    return sel.getRangeAt(0);
  }

  function placeBubble(range) {
    var rect = range.getBoundingClientRect();
    bubble.hidden = false;
    bubble.style.top = window.scrollY + rect.top - bubble.offsetHeight - 8 + "px";
    bubble.style.left = Math.max(8, window.scrollX + rect.left + rect.width / 2 - bubble.offsetWidth / 2) + "px";
  }

  document.addEventListener("selectionchange", function () {
    if (!bubbleLink.hidden) return; // typing a link address
    var range = selectionInText();
    if (!range) { bubble.hidden = true; return; }
    savedRange = range.cloneRange();
    placeBubble(range);
  });

  function restoreSelection() {
    if (!savedRange) return;
    var sel = window.getSelection();
    sel.removeAllRanges();
    sel.addRange(savedRange);
  }

  bubble.addEventListener("mousedown", function (event) { event.preventDefault(); });
  bubble.addEventListener("click", function (event) {
    var btn = event.target.closest("[data-fmt]");
    if (!btn) return;
    var fmt = btn.getAttribute("data-fmt");
    if (fmt === "link") {
      bubbleLink.hidden = false;
      bubbleInput.value = "";
      bubbleInput.focus();
      return;
    }
    restoreSelection();
    snapshot();
    if (fmt === "bold") document.execCommand("bold");
    else if (fmt === "italic") document.execCommand("italic");
    else if (fmt === "clear") { document.execCommand("unlink"); document.execCommand("removeFormat"); }
    else if (fmt === "apply-link") {
      var href = bubbleInput.value.trim();
      if (href) {
        if (!/^(https?:\/\/|\/|mailto:|tel:|#)/.test(href)) href = "https://" + href;
        document.execCommand("createLink", false, href);
      }
      bubbleLink.hidden = true;
    }
    queueSave();
  });
  bubbleInput.addEventListener("keydown", function (event) {
    if (event.key === "Enter") { event.preventDefault(); bubble.querySelector('[data-fmt="apply-link"]').click(); }
    if (event.key === "Escape") { bubbleLink.hidden = true; bubble.hidden = true; }
  });

  /* Ctrl+B / Ctrl+I in a text block. */
  root.addEventListener("keydown", function (event) {
    if (!(event.metaKey || event.ctrlKey)) return;
    var key = event.key.toLowerCase();
    if (key !== "b" && key !== "i") return;
    if (!event.target.closest("[data-text]")) return;
    event.preventDefault();
    snapshot();
    document.execCommand(key === "b" ? "bold" : "italic");
    queueSave();
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
    if (grab && grab.closest(".ed-block--frozen")) return;
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

  /* ---------- phone / desktop preview ---------- */

  var frameEl = root.querySelector("[data-frame]");
  Array.prototype.forEach.call(root.querySelectorAll("[data-device]"), function (button) {
    button.addEventListener("click", function () {
      var phone = button.getAttribute("data-device") === "phone";
      if (frameEl) frameEl.classList.toggle("ed-frame--phone", phone);
      Array.prototype.forEach.call(root.querySelectorAll("[data-device]"), function (other) {
        other.setAttribute("aria-pressed", other === button ? "true" : "false");
      });
    });
  });

  var mustRequest = root.getAttribute("data-must-request") === "true";
  var reviewURL = root.getAttribute("data-review-url") || "";

  if (publishBtn && mustRequest && reviewURL) {
    publishBtn.addEventListener("click", function () {
      var note = window.prompt("Anything the reviewer should know? (optional)", "") || "";
      publishBtn.disabled = true;
      status("dirty", "Sending for review…");
      save();
      setTimeout(function () {
        fetch(reviewURL, {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json", "X-CSRF-Token": CSRF },
          body: JSON.stringify({ note: note }),
        })
          .then(function (r) { return r.json(); })
          .then(function (result) {
            publishBtn.disabled = false;
            if (!result.ok) { status("error", result.message || "That didn't send. Try again."); return; }
            status("saved", result.message || "Sent for review");
            renderChecks(result.checks);
            if (chip) { chip.setAttribute("data-live", "false"); chip.textContent = result.chip || "Waiting for review"; }
          })
          .catch(function () { publishBtn.disabled = false; status("error", "Couldn't reach the server. Try again."); });
      }, 400);
    });
  } else if (publishBtn) {
    publishBtn.addEventListener("click", function () {
      publishBtn.disabled = true;
      status("dirty", "Publishing…");
      save();
      setTimeout(function () {
        fetch(publishURL, {
          method: "POST",
          credentials: "same-origin",
          headers: { "X-CSRF-Token": CSRF, "X-Editor-Client": clientId },
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
            status("saved", result.message || "Published — your page is live");
            renderChecks(result.checks);
            publishBtn.textContent = "Publish changes";
            if (chip) {
              chip.setAttribute("data-live", result.live ? "true" : "false");
              chip.textContent = result.chip || "Live";
            }
          })
          .catch(function () {
            publishBtn.disabled = false;
            status("error", "Couldn't reach the server. Try publishing again.");
          });
      }, 400);
    });
  }

  /* ---------- preview links ---------- */

  var shareMake = root.querySelector("[data-share-make]");
  if (shareMake) {
    shareMake.addEventListener("click", function () {
      var previewURL = root.getAttribute("data-preview-url");
      if (!previewURL) return;
      shareMake.disabled = true;
      save();
      fetch(previewURL, { method: "POST", credentials: "same-origin", headers: { "X-CSRF-Token": CSRF } })
        .then(function (r) { return r.json(); })
        .then(function (result) {
          shareMake.disabled = false;
          if (!result.ok) { status("error", result.message || "Couldn't make a link."); return; }
          var box = root.querySelector("[data-share-result]");
          var input = root.querySelector("[data-share-link]");
          var expires = root.querySelector("[data-share-expires]");
          if (input) input.value = result.url;
          if (expires) expires.textContent = "Works until " + result.expires + ".";
          if (box) box.hidden = false;
          if (input) { input.focus(); input.select(); }
        })
        .catch(function () { shareMake.disabled = false; status("error", "Couldn't reach the server. Try again."); });
    });
    var copy = root.querySelector("[data-share-copy]");
    if (copy) copy.addEventListener("click", function () {
      var input = root.querySelector("[data-share-link]");
      if (!input || !input.value) return;
      input.select();
      try { navigator.clipboard.writeText(input.value); copy.textContent = "Copied"; setTimeout(function () { copy.textContent = "Copy"; }, 1500); } catch (e) { document.execCommand("copy"); }
    });
  }

  /* ---------- other people on this page ---------- */

  var noun = root.getAttribute("data-kind") === "post" ? "post" : "page";
  var peopleNode = root.querySelector("[data-people]");
  var refreshWanted = false;
  var toastTimer = null;

  function editingHere() {
    var active = document.activeElement;
    if (active && article.contains(active)) return true;
    if (saveNode && saveNode.getAttribute("data-save-status") === "dirty") return true;
    return saving;
  }

  function toast(text) {
    var node = root.querySelector(".ed-toast");
    if (!node) {
      node = document.createElement("div");
      node.className = "ed-toast";
      node.setAttribute("role", "status");
      root.appendChild(node);
    }
    node.textContent = text;
    node.hidden = false;
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { node.hidden = true; }, 7000);
  }

  function initials(name) {
    return String(name || "?").split(/\s+/).map(function (word) { return word.charAt(0); }).join("").slice(0, 2).toUpperCase();
  }

  function hue(name) {
    var n = 0;
    for (var i = 0; i < name.length; i++) n = (n * 31 + name.charCodeAt(i)) % 360;
    return String(n);
  }

  function renderPeople(list) {
    if (!peopleNode || !Array.isArray(list)) return;
    var others = list.filter(function (person) { return person.client !== clientId; });
    peopleNode.textContent = "";
    peopleNode.hidden = others.length === 0;
    others.forEach(function (person) {
      var dot = document.createElement("span");
      dot.className = "ed-person";
      dot.title = person.name + " is also editing this " + noun;
      dot.setAttribute("aria-label", dot.title);
      dot.textContent = initials(person.name);
      dot.style.setProperty("--person", hue(person.name || ""));
      peopleNode.appendChild(dot);
    });
  }

  /* Swap in the blocks as the server now has them. Nothing here touches
     the caret: it only runs when nobody is typing in this tab. */
  function refreshCanvas() {
    refreshWanted = false;
    fetch(saveURL + "/canvas", { credentials: "same-origin" })
      .then(function (r) { return r.json(); })
      .then(function (result) {
        if (!result.ok) return;
        if (editingHere()) { refreshWanted = true; return; }
        var holder = document.createElement("div");
        holder.innerHTML = result.html;
        Array.prototype.forEach.call(article.querySelectorAll(".ed-block"), function (el) {
          if (el.parentNode) el.parentNode.removeChild(el);
        });
        while (holder.firstChild) article.appendChild(holder.firstChild);
        if (titleNode && document.activeElement !== titleNode) titleNode.textContent = result.title;
        if (chip && result.chip) {
          chip.setAttribute("data-live", result.live ? "true" : "false");
          chip.textContent = result.chip;
        }
        reindex();
        history = [];
        future = [];
        updateHistoryButtons();
      })
      .catch(function () {});
  }

  if (window.EventSource && saveURL) {
    var stream = new EventSource(saveURL + "/events?client=" + encodeURIComponent(clientId));
    stream.addEventListener("presence", function (event) {
      try { renderPeople(JSON.parse(event.data)); } catch (e) {}
    });
    stream.addEventListener("changed", function (event) {
      var change = {};
      try { change = JSON.parse(event.data); } catch (e) {}
      if (change.client === clientId) return;
      var who = change.by || "Someone";
      if (editingHere()) {
        refreshWanted = true;
        toast(who + " just changed this " + noun + ". You'll see it when you pause. The last save wins, and History keeps every version.");
      } else {
        toast(who + " just changed this " + noun + ".");
        refreshCanvas();
      }
    });
    article.addEventListener("focusout", function () {
      setTimeout(function () { if (refreshWanted && !editingHere()) refreshCanvas(); }, 900);
    });
    window.addEventListener("pagehide", function () { stream.close(); });
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

  var csrfMeta = document.querySelector('meta[name="csrf-token"]');
  var CSRF = csrfMeta ? csrfMeta.getAttribute("content") : "";
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
    var buttons = look.querySelector("[data-look-buttons]:checked");
    var spacing = look.querySelector("[data-look-spacing]:checked");
    return {
      palette: palette ? palette.value : (presets.palettes[0] || {}).key,
      fonts: fonts ? fonts.value : (presets.fonts[0] || {}).key,
      accent: accentInput ? accentInput.value : "",
      buttons: buttons ? buttons.value : "",
      spacing: spacing ? spacing.value : "",
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
      var shape = find(presets.buttons || [], state.buttons);
      var space = find(presets.spacing || [], state.spacing);
      if (shape) canvas.style.setProperty("--site-radius", shape.value);
      if (space) canvas.style.setProperty("--site-space", space.value);
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
      headers: { "Content-Type": "application/json", "X-CSRF-Token": CSRF },
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
