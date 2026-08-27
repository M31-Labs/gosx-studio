;(function () {
  if (typeof window === "undefined" || typeof document === "undefined") return;

  function markManagedScriptLoaded() {
    var script = document.currentScript;
    if (!script || script.tagName !== "SCRIPT" || script.getAttribute("data-gosx-script") !== "managed") return;
    script.setAttribute("data-gosx-script-loaded", "true");
  }

  markManagedScriptLoaded();
  var existingRuntime = window.GoSXStudioContentEditorRuntime;
  if (existingRuntime && typeof existingRuntime.init === "function") {
    existingRuntime.init(document);
    return;
  }

  var BLOCK_TYPES = ["heading", "paragraph", "image", "gallery", "button", "product", "quote", "flow"];
  var TYPE_LABELS = {
    heading: "Heading",
    paragraph: "Paragraph",
    image: "Image",
    gallery: "Gallery",
    button: "Button",
    product: "Product",
    quote: "Quote",
    flow: "Flow"
  };
  var HISTORY_LIMIT = 80;
  var editorStates = new WeakMap();

  function uid() {
    return "blk_" + Math.random().toString(36).slice(2, 9);
  }

  function clone(value) {
    if (value == null) return value;
    try {
      return JSON.parse(JSON.stringify(value));
    } catch (error) {
      return value;
    }
  }

  function isObject(value) {
    return value && typeof value === "object" && !Array.isArray(value);
  }

  function typeSupported(type) {
    return BLOCK_TYPES.indexOf(type) >= 0;
  }

  function normalizeImage(image) {
    var out = isObject(image) ? clone(image) : {};
    out.url = String(out.url || "");
    out.alt = String(out.alt || "");
    return out;
  }

  function normalizeBlock(block) {
    var out = isObject(block) ? clone(block) : {};
    out.id = String(out.id || uid());
    out.type = String(out.type || "paragraph");
    if (out.type === "heading") {
      out.text = String(out.text || "");
      out.level = normalizeHeadingLevel(out.level);
    } else if (out.type === "paragraph" || out.type === "quote") {
      out.text = String(out.text || "");
    } else if (out.type === "image") {
      out.url = String(out.url || "");
      out.alt = String(out.alt || "");
    } else if (out.type === "gallery") {
      out.images = Array.isArray(out.images) ? out.images.map(normalizeImage).filter(function (image) {
        return image.url.trim();
      }) : [];
    } else if (out.type === "button") {
      out.label = String(out.label || "");
      out.href = String(out.href || "");
    } else if (out.type === "product") {
      out.productRef = String(out.productRef || out.ref || "");
    } else if (out.type === "flow") {
      out.flowKey = String(out.flowKey || "");
    }
    return out;
  }

  function duplicateBlockIDs(blocks) {
    var seen = Object.create(null);
    var duplicates = [];
    (blocks || []).forEach(function (block) {
      var id = String(block && block.id || "");
      if (Object.prototype.hasOwnProperty.call(seen, id)) {
        if (duplicates.indexOf(id) < 0) duplicates.push(id);
        return;
      }
      seen[id] = true;
    });
    return duplicates;
  }

  function duplicateBlockIDError(ids) {
    var labels = (ids || []).map(function (id) {
      return id ? '"' + id + '"' : "(empty)";
    });
    return "Duplicate block IDs detected" + (labels.length ? ": " + labels.join(", ") : "") + ". Fix each block ID in the source before using structured editing.";
  }

  function normalizeHeadingLevel(level) {
    level = String(level || "2");
    return level === "3" || level === "4" ? level : "2";
  }

  function defaultBlock(type) {
    type = typeSupported(type) ? type : "paragraph";
    var block = normalizeBlock({ type: type });
    if (type === "heading") {
      block.text = "Section heading";
      block.level = "2";
    } else if (type === "paragraph") {
      block.text = "Write a paragraph...";
    } else if (type === "quote") {
      block.text = "Add a quote...";
    } else if (type === "button") {
      block.label = "Call to action";
      block.href = "/";
    } else if (type === "flow") {
      block.flowKey = "flow-key";
    }
    return block;
  }

  function parseImageLine(line) {
    var parts = String(line || "").split("|");
    return { url: (parts[0] || "").trim(), alt: (parts[1] || "").trim() };
  }

  function parseBlocks(value) {
    value = String(value || "").trim();
    if (!value) return { ok: true, mode: "empty", envelope: { version: 1, blocks: [] }, blocks: [] };
    if (value.charAt(0) === "{") {
      try {
        var parsed = JSON.parse(value);
        if (isObject(parsed) && Array.isArray(parsed.blocks)) {
          var blocks = parsed.blocks.map(normalizeBlock);
          var duplicateIDs = duplicateBlockIDs(blocks);
          if (duplicateIDs.length) {
            return {
              ok: false,
              mode: "duplicate-id",
              error: duplicateBlockIDError(duplicateIDs),
              envelope: null,
              blocks: [],
              duplicateIDs: duplicateIDs
            };
          }
          return {
            ok: true,
            mode: "json",
            envelope: clone(parsed),
            blocks: blocks
          };
        }
        return { ok: false, mode: "invalid", error: "JSON source must contain a blocks array.", envelope: null, blocks: [] };
      } catch (error) {
        return { ok: false, mode: "invalid", error: error && error.message ? error.message : "Invalid JSON source.", envelope: null, blocks: [] };
      }
    }
    return { ok: true, mode: "legacy", envelope: { version: 1, blocks: [] }, blocks: legacyBlocks(value) };
  }

  function legacyBlocks(value) {
    var blocks = [];
    String(value || "").split(/\n\s*\n/g).forEach(function (part) {
      var text = part.trim();
      var lower = text.toLowerCase();
      var match;
      if (!text) return;
      if (text.indexOf("## ") === 0) blocks.push(normalizeBlock({ type: "heading", text: text.slice(3).trim(), level: "2" }));
      else if (text.indexOf("> ") === 0) blocks.push(normalizeBlock({ type: "quote", text: text.slice(2).trim() }));
      else if ((match = text.match(/^\[image:\s*([^\]|]+)(?:\|\s*([^\]]+))?\]$/i))) blocks.push(normalizeBlock({ type: "image", url: match[1].trim(), alt: (match[2] || "").trim() }));
      else if ((match = text.match(/^\[button:\s*([^\]|]+)\|\s*([^\]]+)\]$/i))) blocks.push(normalizeBlock({ type: "button", label: match[1].trim(), href: match[2].trim() }));
      else if ((match = text.match(/^\[product:\s*([^\]]+)\]$/i))) blocks.push(normalizeBlock({ type: "product", productRef: match[1].trim() }));
      else if ((match = text.match(/^\[flow:\s*([^\]]+)\]$/i))) blocks.push(normalizeBlock({ type: "flow", flowKey: match[1].trim() }));
      else if (lower.indexOf("[gallery:") === 0 && text.charAt(text.length - 1) === "]") {
        blocks.push(normalizeBlock({ type: "gallery", images: text.slice(9, -1).trim().split(/[;\n]/g).map(parseImageLine).filter(function (image) { return image.url; }) }));
      } else {
        blocks.push(normalizeBlock({ type: "paragraph", text: text.replace(/\s+/g, " ") }));
      }
    });
    return blocks;
  }

  function serializeBlocks(state) {
    var envelope = isObject(state.envelope) ? clone(state.envelope) : { version: 1 };
    envelope.version = envelope.version || 1;
    envelope.blocks = state.blocks.map(normalizeBlock);
    return JSON.stringify(envelope, null, 2);
  }

  function safePreviewURL(value) {
    value = String(value || "").trim();
    if (!value) return "";
    if (value.charAt(0) === "/" && value.charAt(1) !== "/") return value;
    try {
      var url = new URL(value, window.location.href);
      if (url.protocol === "http:" || url.protocol === "https:" || (url.protocol === "data:" && /^data:image\//i.test(value))) return value;
    } catch (error) {
      return "";
    }
    return "";
  }

  function el(tag, attrs) {
    var node = document.createElement(tag);
    Object.keys(attrs || {}).forEach(function (key) {
      var value = attrs[key];
      if (value === false || value == null) return;
      if (key === "class") node.className = value;
      else if (key === "text") node.textContent = value;
      else if (key === "value") node.value = value;
      else if (value === true) node.setAttribute(key, "true");
      else node.setAttribute(key, value);
    });
    for (var i = 2; i < arguments.length; i++) {
      if (arguments[i]) node.appendChild(arguments[i]);
    }
    return node;
  }

  function control(label, input) {
    return el("label", { class: "content-block__field" }, el("span", { text: label }), input);
  }

  function isEphemeralFormField(name) {
    name = String(name || "");
    return name === "csrf_token" || name === "_csrf" || name === "gorilla.csrf.Token";
  }

  function formValueSignature(value) {
    if (value && typeof value === "object" && typeof value.name === "string" && typeof value.size === "number") {
      return "file:" + value.name + ":" + value.size + ":" + String(value.lastModified || 0) + ":" + String(value.type || "");
    }
    return "value:" + String(value == null ? "" : value);
  }

  function formCSRFToken(formData) {
    if (!formData || typeof formData.get !== "function") return "";
    var token = formData.get("csrf_token");
    return typeof token === "string" ? token.trim() : "";
  }

  function formSignature(form) {
    if (!form || typeof FormData !== "function") return "";
    var data;
    try {
      data = new FormData(form);
    } catch (error) {
      return "";
    }
    var parts = [];
    var append = function (value, name) {
      if (isEphemeralFormField(name)) return;
      parts.push(encodeURIComponent(String(name)) + "=" + encodeURIComponent(formValueSignature(value)));
    };
    if (typeof data.forEach === "function") {
      data.forEach(append);
    } else if (typeof data.entries === "function") {
      var entries = data.entries();
      var next;
      while (!(next = entries.next()).done) append(next.value[1], next.value[0]);
    }
    return parts.join("&");
  }

  function formIsDirty(state) {
    return formSignature(state.form) !== state.initialFormSignature;
  }

  function textInput(value, placeholder, onInput, list, attrs) {
    var input = el("input", Object.assign({ type: "text", value: value || "", placeholder: placeholder || "" }, attrs || {}));
    if (list) input.setAttribute("list", list);
    input.addEventListener("input", function () { onInput(input.value); });
    return input;
  }

  function textArea(value, placeholder, onInput, rows, attrs) {
    var input = el("textarea", Object.assign({ rows: rows || "3", placeholder: placeholder || "" }, attrs || {}));
    input.value = value || "";
    input.addEventListener("input", function () { onInput(input.value); });
    return input;
  }

  function renderPreview(block) {
    var preview = el("div", { class: "content-block__preview" });
    var type = block.type;
    if (type === "heading") {
      preview.appendChild(el("h3", { text: block.text || "Heading" }));
    } else if (type === "quote") {
      preview.appendChild(el("blockquote", { text: block.text || "Quote" }));
    } else if (type === "image") {
      var url = safePreviewURL(block.url);
      preview.appendChild(url ? el("img", { src: url, alt: block.alt || "" }) : el("p", { text: "Image preview" }));
    } else if (type === "gallery") {
      var strip = el("div", { class: "content-block__preview-strip" });
      (block.images || []).slice(0, 4).forEach(function (image) {
        var url = safePreviewURL(image.url);
        if (url) strip.appendChild(el("img", { src: url, alt: image.alt || "" }));
      });
      preview.appendChild(strip.childNodes.length ? strip : el("p", { text: "Gallery preview" }));
    } else if (type === "button") {
      preview.appendChild(el("span", { class: "button button--primary", text: block.label || "Button" }));
    } else if (type === "product") {
      preview.appendChild(el("p", { text: "Product: " + (block.productRef || "product-reference") }));
    } else if (type === "flow") {
      preview.appendChild(el("p", { text: "Flow: " + (block.flowKey || "flow-key") }));
    } else if (type === "paragraph") {
      preview.appendChild(el("p", { text: block.text || "Paragraph" }));
    } else {
      preview.appendChild(el("p", { text: "Unsupported block: " + type }));
    }
    return preview;
  }

  function initEditor(root) {
    if (!root || root.getAttribute("data-content-editor-bound") === "true") return editorStates.get(root) || null;
    var source = root.querySelector("[data-content-editor-source]");
    var list = root.querySelector("[data-content-editor-list]");
    var toolbar = root.querySelector(".content-editor__toolbar");
    var form = root.closest("form") || document;
    var format = form.querySelector("[data-content-editor-format]");
    if (!source || !list || !format) return null;

    var parsed = parseBlocks(source.value);
    var state = {
      root: root,
      source: source,
      list: list,
      toolbar: toolbar,
      form: form,
      format: format,
      mediaList: root.getAttribute("data-content-editor-media-list") || "",
      initialSource: source.value,
      initialFormSignature: formSignature(form),
      envelope: parsed.envelope || { version: 1 },
      blocks: parsed.blocks,
      sourceState: parsed.ok ? "ready" : "invalid",
      sourceError: parsed.error || "",
      saveState: "idle",
      saveDetail: "",
      selectedID: "",
      dragging: null,
      keyboardDragID: "",
      internalSourceWrite: false,
      history: [],
      redo: [],
      suppressHistory: false,
      intentionalDiscard: false,
      createCompleted: false,
      createRedirect: "",
      keyboardDragPreviewIndex: -1,
      searchQuery: "",
      collapsedByID: Object.create(null),
      collapseBodyPrefix: uid() + "-body-",
      createFormLocked: false,
      createLockRecords: null,
      saveErrorsNode: null,
      saveErrorsID: "",
      sourceDiagnosticNode: null
    };
    root.setAttribute("data-content-editor-bound", "true");
    editorStates.set(root, state);
    bindEditor(state);
    render(state);
    syncStatus(state);
    return state;
  }

  function bindEditor(state) {
    var saveStatus = ensureSaveStatus(state);
    ensureSearchControls(state);
    saveStatus.retry.addEventListener("click", function (event) {
      event.preventDefault();
      handleSubmit(state, {
        submitter: saveSubmitter(state),
        preventDefault: function () {}
      });
    });

    state.source.addEventListener("input", function () {
      if (state.internalSourceWrite) {
        syncStatus(state);
        return;
      }
      if (state.format.value !== "blocks") {
        syncStatus(state);
        return;
      }
      var parsed = parseBlocks(state.source.value);
      if (!parsed.ok) {
        state.sourceState = "invalid";
        state.sourceError = parsed.error || "Invalid source.";
        render(state);
        syncStatus(state);
        return;
      }
      state.envelope = parsed.envelope || { version: 1 };
      state.blocks = parsed.blocks;
      state.sourceState = "ready";
      state.sourceError = "";
      render(state);
      syncStatus(state);
    });

    state.format.addEventListener("change", function () {
      if (state.format.value === "blocks") {
        var parsed = parseBlocks(state.source.value);
        state.sourceState = parsed.ok ? "ready" : "invalid";
        state.sourceError = parsed.error || "";
        if (parsed.ok) {
          state.envelope = parsed.envelope || { version: 1 };
          state.blocks = parsed.blocks;
        }
      }
      render(state);
      syncStatus(state);
    });

    state.form.addEventListener("input", function (event) {
      if (event.target === state.source || event.target === state.format) return;
      syncStatus(state);
    });
    state.form.addEventListener("change", function (event) {
      if (event.target === state.source || event.target === state.format) return;
      syncStatus(state);
    });

    ["click", "pointerdown", "mousedown", "keydown", "beforeinput", "drop", "focusin"].forEach(function (eventName) {
      state.form.addEventListener(eventName, function (event) {
        if (!state.createFormLocked || !createLockTarget(state, event.target)) return;
        event.preventDefault();
        if (typeof event.stopImmediatePropagation === "function") event.stopImmediatePropagation();
        if (eventName === "focusin" && event.target && typeof event.target.blur === "function") event.target.blur();
      }, true);
    });

    if (state.toolbar) {
      state.toolbar.addEventListener("click", function (event) {
        var button = event.target.closest("[data-content-add]");
        if (!button || !state.toolbar.contains(button)) return;
        event.preventDefault();
        insertBlock(state, button.getAttribute("data-content-add"));
      });
      state.toolbar.addEventListener("dragstart", function (event) {
        if (String(state.searchQuery || "").trim()) {
          event.preventDefault();
          return;
        }
        var button = event.target.closest("[data-content-add]");
        if (!button || !state.toolbar.contains(button)) return;
        state.dragging = { type: "palette", blockType: button.getAttribute("data-content-add") || "paragraph" };
        if (event.dataTransfer) {
          event.dataTransfer.effectAllowed = "copy";
          event.dataTransfer.setData("text/plain", "content-editor-palette:" + state.dragging.blockType);
        }
      });
      Array.prototype.forEach.call(state.toolbar.querySelectorAll("[data-content-add]"), function (button) {
        button.setAttribute("draggable", "true");
        button.setAttribute("data-content-editor-palette-item", "true");
      });
    }

    state.list.addEventListener("dragstart", function (event) {
      if (String(state.searchQuery || "").trim()) {
        event.preventDefault();
        return;
      }
      var handle = event.target.closest("[data-content-drag-handle]");
      if (!handle || !state.list.contains(handle)) {
        event.preventDefault();
        return;
      }
      var row = handle.closest("[data-content-block-id]");
      if (!row) return;
      var id = row.getAttribute("data-content-block-id");
      state.dragging = { type: "move", id: id };
      row.setAttribute("data-content-editor-drag-preview", "true");
      if (event.dataTransfer) {
        event.dataTransfer.effectAllowed = "move";
        event.dataTransfer.setData("text/plain", id);
      }
    });
    state.list.addEventListener("dragover", function (event) {
      if (String(state.searchQuery || "").trim()) {
        clearDropIndicators(state);
        return;
      }
      if (!state.dragging && hasMediaDrag(event)) state.dragging = { type: "media" };
      if (!state.dragging) return;
      var target = dropTarget(state, event);
      if (!target.valid) return;
      event.preventDefault();
      applyDropIndicator(state, target);
    });
    state.list.addEventListener("dragleave", function (event) {
      if (!state.list.contains(event.relatedTarget)) clearDropIndicators(state);
      if (state.dragging && state.dragging.type === "media" && !state.list.contains(event.relatedTarget)) state.dragging = null;
    });
    state.list.addEventListener("drop", function (event) {
      if (String(state.searchQuery || "").trim()) {
        cancelDrag(state);
        return;
      }
      if (!state.dragging && hasMediaDrag(event)) state.dragging = { type: "media" };
      if (!state.dragging) return;
      var target = dropTarget(state, event);
      if (!target.valid) {
        cancelDrag(state);
        return;
      }
      event.preventDefault();
      if (state.dragging.type === "move") {
        reorderBlock(state, state.dragging.id, target.id, target.position);
      } else if (state.dragging.type === "palette") {
        insertBlock(state, state.dragging.blockType, target.id, target.position);
      } else if (state.dragging.type === "media") {
        insertMediaBlock(state, mediaPayloadFromDrop(event), target.id, target.position);
      }
      endDrag(state);
    });
    state.list.addEventListener("dragend", function () {
      endDrag(state);
    });

    state.root.addEventListener("click", function (event) {
      var row = event.target.closest("[data-content-block-id]");
      var collapse = event.target.closest("[data-content-editor-collapse]");
      if (collapse && state.root.contains(collapse)) {
        event.preventDefault();
        if (row) state.selectedID = row.getAttribute("data-content-block-id") || "";
        toggleCollapse(state, collapse.getAttribute("data-content-editor-block") || "");
        return;
      }
      var action = event.target.closest("[data-content-editor-action]");
      if (action && state.root.contains(action)) {
        event.preventDefault();
        if (row) state.selectedID = row.getAttribute("data-content-block-id") || "";
        runAction(state, action);
        return;
      }
      if (row && state.root.contains(row)) selectBlock(state, row.getAttribute("data-content-block-id"), false);
    });

    state.root.addEventListener("keydown", function (event) {
      if (isSaveShortcut(event)) {
        if (!event.defaultPrevented && state.form !== document && state.form.getAttribute("data-content-editor-enhanced-submit") === "true") {
          var submitter = saveSubmitter(state);
          if (submitter) {
            event.preventDefault();
            handleSubmit(state, { submitter: submitter, preventDefault: function () {} });
          }
        }
        return;
      }
      if (isTextEditingTarget(event.target)) return;
      if ((event.ctrlKey || event.metaKey) && !event.shiftKey && event.key.toLowerCase() === "z") {
        event.preventDefault();
        undo(state);
      } else if ((event.ctrlKey || event.metaKey) && (event.key.toLowerCase() === "y" || (event.shiftKey && event.key.toLowerCase() === "z"))) {
        event.preventDefault();
        redo(state);
      }
    });

    state.root.addEventListener("keydown", function (event) {
      var handle = event.target.closest("[data-content-drag-handle]");
      if (!handle || !state.root.contains(handle)) return;
      var row = handle.closest("[data-content-block-id]");
      if (!row) return;
      var id = row.getAttribute("data-content-block-id");
      if (event.key === " " || event.key === "Enter") {
        event.preventDefault();
        if (state.keyboardDragID === id) commitKeyboardDrag(state, id);
        else beginKeyboardDrag(state, id);
      } else if (event.key === "Escape") {
        event.preventDefault();
        cancelKeyboardDrag(state, id);
      } else if (event.key === "ArrowUp" || event.key === "ArrowDown") {
        event.preventDefault();
        if (state.keyboardDragID === id) {
          previewKeyboardMove(state, id, event.key === "ArrowUp" ? -1 : 1);
        }
      }
    });

    state.form.addEventListener("submit", function (event) {
      handleSubmit(state, event);
    });
    state.form.addEventListener("keydown", function (event) {
      if (event.defaultPrevented || !isSaveShortcut(event)) return;
      if (state.form.getAttribute("data-content-editor-enhanced-submit") !== "true") return;
      var submitter = saveSubmitter(state);
      if (!submitter) return;
      event.preventDefault();
      handleSubmit(state, { submitter: submitter, preventDefault: function () {} });
    });
    state.form.addEventListener("click", function (event) {
      var link = event.target.closest("a");
      if (link && link.getAttribute("data-content-editor-discard") === "true") state.intentionalDiscard = true;
    });
  }

  function isTextEditingTarget(target) {
    if (!target || !target.matches) return false;
    return target.matches("input, textarea, select, [contenteditable], [contenteditable='true']");
  }

  function isSaveShortcut(event) {
    return !!event && (event.ctrlKey || event.metaKey) && !event.altKey && !event.shiftKey && String(event.key || "").toLowerCase() === "s";
  }

  function beginKeyboardDrag(state, id) {
    var index = blockIndex(state, id);
    if (index < 0) return;
    state.keyboardDragID = id;
    state.keyboardDragPreviewIndex = index;
    render(state);
    focusHandle(state, id);
  }

  function previewKeyboardMove(state, id, delta) {
    var from = blockIndex(state, id);
    if (from < 0) return;
    var current = state.keyboardDragPreviewIndex >= 0 ? state.keyboardDragPreviewIndex : from;
    var target = Math.max(0, Math.min(state.blocks.length - 1, current + delta));
    if (target === current) return;
    state.keyboardDragPreviewIndex = target;
    render(state);
    focusHandle(state, id);
  }

  function commitKeyboardDrag(state, id) {
    var from = blockIndex(state, id);
    var target = state.keyboardDragPreviewIndex;
    state.keyboardDragID = "";
    state.keyboardDragPreviewIndex = -1;
    if (from >= 0 && target >= 0 && target < state.blocks.length && target !== from) {
      pushHistory(state);
      var item = state.blocks.splice(from, 1)[0];
      state.blocks.splice(target, 0, item);
      state.selectedID = id;
      writeSource(state);
    }
    render(state);
    focusHandle(state, id);
  }

  function cancelKeyboardDrag(state, id) {
    if (!state.keyboardDragID) return;
    state.keyboardDragID = "";
    state.keyboardDragPreviewIndex = -1;
    render(state);
    focusHandle(state, id);
  }

  function keyboardPreviewBlocks(state) {
    if (!state.keyboardDragID || state.keyboardDragPreviewIndex < 0) return state.blocks;
    var from = blockIndex(state, state.keyboardDragID);
    var target = state.keyboardDragPreviewIndex;
    if (from < 0 || target < 0 || target >= state.blocks.length || from === target) return state.blocks;
    var blocks = state.blocks.slice();
    var item = blocks.splice(from, 1)[0];
    blocks.splice(target, 0, item);
    return blocks;
  }

  function bindKeyboardHandle(state, handle, id) {
    handle.addEventListener("blur", function () {
      window.setTimeout(function () {
        if (state.keyboardDragID !== id) return;
        var current = state.root.querySelector('[data-content-block-id="' + cssEscape(id) + '"] [data-content-drag-handle]');
        if (document.activeElement !== current) cancelKeyboardDrag(state, id);
      }, 0);
    });
  }

  function shouldEnhanceSubmit(state, event) {
    if (state.form.getAttribute("data-content-editor-enhanced-submit") !== "true") return false;
    var submitter = event && event.submitter;
    if (submitter && submitter.getAttribute && submitter.getAttribute("data-admin-confirm")) return false;
    if (submitter && submitter.getAttribute && submitter.getAttribute("data-content-editor-native-submit") === "true") return false;
    if (submitter && submitter.hasAttribute && submitter.hasAttribute("formaction")) return false;
    var method = String((submitter && submitter.getAttribute && submitter.getAttribute("formmethod")) || state.form.getAttribute("method") || "get").toLowerCase();
    if (method !== "post") return false;
    if (typeof state.form.checkValidity === "function" && !state.form.checkValidity()) return false;
    return true;
  }

  function saveSubmitter(state) {
    var candidates = state.form.querySelectorAll ? state.form.querySelectorAll("button, input") : [];
    for (var index = 0; index < candidates.length; index += 1) {
      var candidate = candidates[index];
      if (!candidate || candidate.disabled) continue;
      if (candidate.getAttribute("data-admin-confirm")) continue;
      if (candidate.getAttribute("data-content-editor-native-submit") === "true") continue;
      if (candidate.hasAttribute && candidate.hasAttribute("formaction")) continue;
      var type = String(candidate.type || "").toLowerCase();
      if (type === "submit" || type === "image" || (candidate.tagName === "BUTTON" && !type)) return candidate;
    }
    return null;
  }

  function actionIsCreate(action) {
    try {
      var url = new URL(String(action || ""), window.location.href);
      return /(?:^|\/)create\/?$/i.test(url.pathname);
    } catch (error) {
      return /(?:^|\/)create(?:[/?#]|$)/i.test(String(action || ""));
    }
  }

  function pathLooksUnauthenticated(path) {
    path = String(path || "").toLowerCase();
    return path === "/login" || path.indexOf("/login/") === 0 || path === "/signin" || path === "/sign-in" || path.indexOf("/auth/") === 0;
  }

  function redirectLooksUnauthenticated(redirect) {
    try {
      var url = new URL(String(redirect || ""), window.location.href);
      return url.origin === window.location.origin && pathLooksUnauthenticated(url.pathname);
    } catch (error) {
      return false;
    }
  }

  function saveStateLabel(state) {
    var dirty = formIsDirty(state);
    switch (state.saveState) {
      case "saving": return "Saving";
      case "saved": return "Saved";
      case "dirty": return "Unsaved";
      case "offline": return "Offline";
      case "failed": return "Save failed";
      case "unconfirmed": return "Save unconfirmed";
      case "auth": return "Sign-in required";
      case "conflict": return "Save conflict";
      case "idle": return dirty ? "Unsaved" : "Ready";
      default: return dirty ? "Unsaved" : "Ready";
    }
  }

  function ensureSaveStatus(state) {
    var status = state.saveStatusNode || state.root.querySelector("[data-content-editor-save-status]");
    if (!status) {
      status = el("div", {
        class: "content-editor__save-status",
        "data-content-editor-save-status": "true",
        role: "status",
        "aria-live": "polite",
        "aria-atomic": "true"
      });
      state.root.insertBefore(status, state.root.firstChild || null);
    }
    var label = status.querySelector("[data-content-editor-save-label]");
    var detail = status.querySelector("[data-content-editor-save-detail-text]");
    var retry = status.querySelector("[data-content-editor-save-retry]");
    if (!label || !detail || !retry) {
      status.textContent = "";
      label = el("span", { class: "content-editor__save-label", "data-content-editor-save-label": "true" });
      detail = el("span", { class: "content-editor__save-detail", "data-content-editor-save-detail-text": "true" });
      retry = el("button", {
        type: "button",
        class: "content-editor__save-retry",
        "data-content-editor-save-retry": "true",
        hidden: true,
        text: "Retry save"
      });
      status.appendChild(label);
      status.appendChild(detail);
      status.appendChild(retry);
    }
    var errors = status.querySelector("[data-content-editor-save-errors]");
    if (!errors) {
      if (!state.saveErrorsID) state.saveErrorsID = uid() + "-save-errors";
      errors = el("div", {
        class: "content-editor__save-errors",
        "data-content-editor-save-errors": "true",
        "aria-label": "Save errors",
        id: state.saveErrorsID,
        hidden: true
      });
      status.appendChild(errors);
    } else if (!state.saveErrorsID) {
      state.saveErrorsID = errors.id || uid() + "-save-errors";
      if (!errors.id) errors.id = state.saveErrorsID;
    }
    var createdLink = status.querySelector("[data-content-editor-save-created-link]");
    if (!createdLink) {
      createdLink = el("a", {
        class: "content-editor__save-created-link",
        "data-content-editor-save-created-link": "true",
        "data-content-editor-discard": "true",
        hidden: true,
        text: "Open saved item"
      });
      status.appendChild(createdLink);
    }
    state.saveStatusNode = status;
    state.saveLabelNode = label;
    state.saveDetailNode = detail;
    state.saveRetryNode = retry;
    state.saveErrorsNode = errors;
    state.saveCreatedLinkNode = createdLink;
    return { status: status, label: label, detail: detail, retry: retry, errors: errors, createdLink: createdLink };
  }

  function clearFieldErrors(state) {
    Array.prototype.forEach.call(state.form.querySelectorAll ? state.form.querySelectorAll("[data-content-editor-field-error]") : [], function (control) {
      control.removeAttribute("data-content-editor-field-error");
      control.removeAttribute("aria-invalid");
      var describedBy = String(control.getAttribute("aria-describedby") || "").split(/\s+/).filter(function (id) {
        return id && id !== state.saveErrorsID;
      });
      if (describedBy.length) control.setAttribute("aria-describedby", describedBy.join(" "));
      else control.removeAttribute("aria-describedby");
    });
    if (!state.saveErrorsNode) return;
    state.saveErrorsNode.replaceChildren();
    state.saveErrorsNode.hidden = true;
  }

  function fieldErrorText(value) {
    if (Array.isArray(value)) return value.map(fieldErrorText).filter(Boolean).join("; ");
    if (isObject(value) && value.message) return String(value.message);
    return String(value == null ? "" : value);
  }

  function syncFieldErrors(state, fieldErrors) {
    clearFieldErrors(state);
    if (!isObject(fieldErrors)) return;
    var entries = Object.keys(fieldErrors).map(function (name) {
      return { name: String(name), message: fieldErrorText(fieldErrors[name]) };
    }).filter(function (entry) { return entry.message; });
    if (!entries.length) return;
    var nodes = ensureSaveStatus(state);
    var list = el("ul");
    entries.forEach(function (entry) {
      list.appendChild(el("li", { text: entry.name + ": " + entry.message }));
      Array.prototype.forEach.call(state.form.querySelectorAll ? state.form.querySelectorAll("[name]") : [], function (control) {
        if (String(control.name || "") !== entry.name) return;
        control.setAttribute("data-content-editor-field-error", "true");
        control.setAttribute("aria-invalid", "true");
        var describedBy = String(control.getAttribute("aria-describedby") || "").split(/\s+/).filter(Boolean);
        if (describedBy.indexOf(state.saveErrorsID) < 0) describedBy.push(state.saveErrorsID);
        control.setAttribute("aria-describedby", describedBy.join(" "));
      });
    });
    nodes.errors.appendChild(list);
    nodes.errors.hidden = false;
  }

  function createLockExempt(state, control) {
    if (!control || !control.closest) return false;
    if (state.saveStatusNode && state.saveStatusNode.contains(control)) return true;
    if (control.closest("[data-content-editor-discard='true']")) return true;
    return false;
  }

  function createLockTarget(state, target) {
    if (!state.createFormLocked || !target || !target.closest) return null;
    var control = target.closest("[data-content-editor-create-locked]");
    return control && state.form.contains(control) ? control : null;
  }

  function setCreateFormLocked(state, locked) {
    if (locked) {
      if (state.createFormLocked) return;
      state.createFormLocked = true;
      state.createLockRecords = [];
      var controls = state.form.querySelectorAll ? state.form.querySelectorAll("input, textarea, select, button, [contenteditable]") : [];
      Array.prototype.forEach.call(controls, function (control) {
        if (!control || createLockExempt(state, control)) return;
        var record = {
          control: control,
          disabled: "disabled" in control ? control.disabled : null,
          readOnly: "readOnly" in control ? control.readOnly : null,
          readOnlyAttribute: control.getAttribute("readonly"),
          inert: "inert" in control ? control.inert : null,
          inertAttribute: control.getAttribute("inert"),
          tabIndexAttribute: control.getAttribute("tabindex"),
          contenteditable: control.getAttribute("contenteditable"),
          ariaDisabled: control.getAttribute("aria-disabled"),
          lockMarker: control.getAttribute("data-content-editor-create-locked")
        };
        if ("inert" in control) control.inert = true;
        control.setAttribute("inert", "");
        if ("readOnly" in control) control.readOnly = true;
        if ("readOnly" in control) control.setAttribute("readonly", "");
        if (control.hasAttribute("contenteditable")) control.setAttribute("contenteditable", "false");
        control.setAttribute("tabindex", "-1");
        control.setAttribute("aria-disabled", "true");
        control.setAttribute("data-content-editor-create-locked", "true");
        state.createLockRecords.push(record);
      });
      state.root.setAttribute("data-content-editor-create-lock", "true");
      state.root.setAttribute("aria-busy", "true");
      return;
    }
    if (!state.createFormLocked) return;
    (state.createLockRecords || []).forEach(function (record) {
      var control = record.control;
      if (!control) return;
      if (record.disabled !== null) control.disabled = record.disabled;
      if (record.readOnly !== null) control.readOnly = record.readOnly;
      if (record.readOnlyAttribute == null) control.removeAttribute("readonly");
      else control.setAttribute("readonly", record.readOnlyAttribute);
      if (record.inert !== null) control.inert = record.inert;
      if (record.inertAttribute == null) control.removeAttribute("inert");
      else control.setAttribute("inert", record.inertAttribute);
      if (record.tabIndexAttribute == null) control.removeAttribute("tabindex");
      else control.setAttribute("tabindex", record.tabIndexAttribute);
      if (record.contenteditable == null) control.removeAttribute("contenteditable");
      else control.setAttribute("contenteditable", record.contenteditable);
      if (record.ariaDisabled == null) control.removeAttribute("aria-disabled");
      else control.setAttribute("aria-disabled", record.ariaDisabled);
      if (record.lockMarker == null) control.removeAttribute("data-content-editor-create-locked");
      else control.setAttribute("data-content-editor-create-locked", record.lockMarker);
    });
    state.createFormLocked = false;
    state.createLockRecords = null;
    state.root.removeAttribute("data-content-editor-create-lock");
    state.root.removeAttribute("aria-busy");
  }

  function blockSearchText(block) {
    var parts = [TYPE_LABELS[block.type] || block.type, block.type];
    Object.keys(block || {}).forEach(function (key) {
      if (key === "id") return;
      var value = block[key];
      if (value == null) return;
      parts.push(typeof value === "string" ? value : JSON.stringify(value));
    });
    return parts.join(" ").toLowerCase();
  }

  function blockMatchesSearch(state, block) {
    var query = String(state.searchQuery || "").trim().toLowerCase();
    return !query || blockSearchText(block).indexOf(query) >= 0;
  }

  function ensureSearchControls(state) {
    var controls = state.searchControlsNode || state.root.querySelector("[data-content-editor-search-controls]");
    if (!controls) {
      controls = el("div", {
        class: "content-editor__search-controls",
        "data-content-editor-search-controls": "true"
      });
      state.root.insertBefore(controls, state.saveStatusNode || state.root.firstChild || null);
    }
    var input = controls.querySelector("[data-content-editor-search]");
    var clear = controls.querySelector("[data-content-editor-search-clear]");
    var count = controls.querySelector("[data-content-editor-search-count]");
    if (!input || !clear || !count) {
      controls.replaceChildren();
      input = el("input", {
        type: "search",
        class: "content-editor__search-input",
        "data-content-editor-search": "true",
        placeholder: "Search blocks"
      });
      clear = el("button", {
        type: "button",
        class: "content-editor__search-clear",
        "data-content-editor-search-clear": "true",
        hidden: true,
        text: "Clear search"
      });
      count = el("span", {
        class: "content-editor__search-count",
        "data-content-editor-search-count": "true",
        role: "status",
        "aria-live": "polite",
        "aria-atomic": "true"
      });
      controls.appendChild(el("label", { class: "content-editor__search-label" }, el("span", { text: "Search blocks" }), input));
      controls.appendChild(clear);
      controls.appendChild(count);
      input.addEventListener("input", function () {
        state.searchQuery = input.value;
        render(state);
      });
      clear.addEventListener("click", function () {
        input.value = "";
        state.searchQuery = "";
        render(state);
        input.focus();
      });
    }
    state.searchControlsNode = controls;
    state.searchInputNode = input;
    state.searchClearNode = clear;
    state.searchCountNode = count;
    // Search is view state only; never let a host-supplied name add it to the
    // form payload or make it participate in the save dirty signature.
    input.removeAttribute("name");
    input.value = state.searchQuery || "";
    return { controls: controls, input: input, clear: clear, count: count };
  }

  function syncSearchControls(state, total, visible) {
    var nodes = ensureSearchControls(state);
    var query = String(state.searchQuery || "").trim();
    nodes.clear.hidden = !query;
    nodes.count.textContent = query ? visible + " of " + total + " blocks" : total + " blocks";
    if (query) state.root.setAttribute("data-content-editor-search-query", query);
    else state.root.removeAttribute("data-content-editor-search-query");
    state.root.setAttribute("data-content-editor-search-visible-count", String(visible));
  }

  function syncSaveStatus(state) {
    var nodes = ensureSaveStatus(state);
    var failure = ["offline", "failed", "unconfirmed", "auth", "conflict"].indexOf(state.saveState) >= 0;
    var retryable = (failure || state.saveState === "dirty") && !state.createCompleted;
    nodes.status.setAttribute("role", failure ? "alert" : "status");
    nodes.status.setAttribute("aria-live", failure ? "assertive" : "polite");
    nodes.label.textContent = saveStateLabel(state);
    nodes.detail.textContent = state.saveDetail || (formIsDirty(state) ? "Changes waiting to be saved." : "Ready");
    nodes.retry.hidden = !retryable;
    nodes.retry.disabled = state.saveState === "saving";
    var createdTarget = "";
    if (state.createRedirect) {
      try {
        var target = new URL(state.createRedirect, window.location.href);
        createdTarget = target.origin === window.location.origin ? target.href : "";
      } catch (error) {
        createdTarget = "";
      }
    }
    nodes.createdLink.hidden = !createdTarget;
    if (createdTarget) {
      nodes.createdLink.href = createdTarget;
    } else {
      nodes.createdLink.removeAttribute("href");
    }
  }

  function handleSubmit(state, event) {
    if (!shouldEnhanceSubmit(state, event)) return;
    event.preventDefault();
    if (state.saveState === "saving") return;
    if (state.createCompleted) {
      setSaveState(state, "dirty", "This new item was already created. Open the saved item before retrying newer edits.");
      return;
    }
    if (window.navigator && window.navigator.onLine === false) {
      setSaveState(state, "offline", "Offline. Your edits are still local; retry when the connection returns.");
      return;
    }
    var submitter = event.submitter || null;
    var action = state.form.action || window.location.href;
    if (submitter && submitter.hasAttribute && submitter.hasAttribute("formaction")) {
      action = submitter.getAttribute("formaction") || window.location.href;
    }
    var actionURL;
    try {
      actionURL = new URL(action, window.location.href);
    } catch (error) {
      setSaveState(state, "failed", "Save target is invalid. Your edits are still local; retry save.");
      return;
    }
    if (actionURL.origin !== window.location.origin) {
      setSaveState(state, "failed", "Save target must stay on this site. Your edits are still local; retry save.");
      return;
    }
    action = actionURL.href;
    var body;
    try {
      body = new FormData(state.form);
      if (submitter && submitter.name && !body.has(submitter.name)) body.append(submitter.name, submitter.value || "");
    } catch (error) {
      setSaveState(state, "failed", "Could not prepare the save request. Your edits are still local; retry save.");
      return;
    }
    var csrfToken = formCSRFToken(body);
    if (!csrfToken) {
      setSaveState(state, "failed", "This form has no CSRF token. Refresh the page and retry; your edits are still local.");
      return;
    }
    var submittedSource = state.source.value;
    var submittedFormSignature = formSignature(state.form);
    clearFieldErrors(state);
    if (actionIsCreate(action)) setCreateFormLocked(state, true);
    setSaveState(state, "saving", "Saving draft...");
    fetch(action, {
      method: "POST",
      body: body,
      credentials: "same-origin",
      headers: {
        "Accept": "application/json",
        "X-Requested-With": "XMLHttpRequest",
        "X-CSRF-Token": csrfToken
      }
    }).then(function (response) {
      if (!editorStillActive(state)) return null;
      var contentType = response.headers && response.headers.get ? (response.headers.get("Content-Type") || "") : "";
      if (response.status === 401 || response.status === 403 || responseLooksUnauthenticated(response)) {
        setSaveState(state, "auth", "Save needs a fresh session. Your edits are still local; sign in and retry.");
        return null;
      }
      var payloadPromise = contentType.toLowerCase().indexOf("json") >= 0 && typeof response.json === "function" ? response.json().catch(function () { return null; }) : Promise.resolve(null);
      return payloadPromise.then(function (payload) {
        if (!editorStillActive(state)) return;
        if (!payload) {
          setSaveState(state, response.ok ? "unconfirmed" : "failed", "Save was not confirmed by a structured action response. Your edits are still local; retry save.");
          return;
        }
        var structuredSuccess = payload && typeof payload === "object" && !Array.isArray(payload) && payload.ok === true && response.status >= 200 && response.status < 400;
        if (!structuredSuccess) {
          if (payload.fieldErrors) syncFieldErrors(state, payload.fieldErrors);
          var failureState = response.status === 409 || payload.conflict === true ? "conflict" : "failed";
          setSaveState(state, failureState, payload && payload.message ? String(payload.message) + " Your edits are still local; retry save." : "Save was not confirmed. Your edits are still local; retry save.");
          return;
        }
        if (payload.redirect && redirectLooksUnauthenticated(payload.redirect)) {
          setSaveState(state, "auth", "Save needs a fresh session. Your edits are still local; sign in and retry.");
          return;
        }
        var newerEdits = formSignature(state.form) !== submittedFormSignature;
        state.initialSource = submittedSource;
        state.initialFormSignature = submittedFormSignature;
        if (newerEdits) {
          if (actionIsCreate(action)) {
            state.createCompleted = true;
            state.createRedirect = payload.redirect ? String(payload.redirect) : "";
            setSaveState(state, "dirty", state.createRedirect ? "New item created. Newer edits remain local; open the saved item before continuing." : "New item created. Newer edits remain local; open it from the item list before continuing.");
          } else {
            setSaveState(state, "dirty", "Earlier edits saved. Newer edits remain local; retry save.");
          }
          syncStatus(state);
          return;
        }
        if (actionIsCreate(action)) state.createCompleted = true;
        setSaveState(state, "saved", payload.message ? String(payload.message) : "Saved.");
        syncStatus(state);
        if (payload.redirect) navigateAfterSave(state, payload.redirect);
      });
    }).catch(function () {
      if (!editorStillActive(state)) return;
      setSaveState(state, "failed", "Save failed. Your edits are still local; retry save.");
    });
  }

  function navigateAfterSave(state, redirect) {
    if (!editorStillActive(state)) return;
    var target = String(redirect || "").trim();
    if (!target) return;
    var event = new CustomEvent("gosxstudio:content-editor-navigate", {
      bubbles: true,
      cancelable: true,
      detail: { root: state.root, href: target }
    });
    state.root.dispatchEvent(event);
    if (event.defaultPrevented) return;
    try {
      var url = new URL(target, window.location.href);
      if (url.origin !== window.location.origin) return;
      window.location.href = url.href;
    } catch (error) {
      return;
    }
  }

  function responseLooksUnauthenticated(response) {
    if (!response || !response.redirected || !response.url) return false;
    try {
      var url = new URL(response.url, window.location.href);
      return url.origin === window.location.origin && pathLooksUnauthenticated(url.pathname);
    } catch (error) {
      return false;
    }
  }

  function editorStillActive(state) {
    if (!state || state.intentionalDiscard || !state.root || !state.form) return false;
    if (state.root.isConnected === false) return false;
    if (document.documentElement && !document.documentElement.contains(state.root)) return false;
    if (state.form !== document && state.form.contains && !state.form.contains(state.root)) return false;
    return true;
  }

  function setSaveState(state, value, detail) {
    if (!editorStillActive(state)) return;
    if (value !== "saving") setCreateFormLocked(state, false);
    state.saveState = value;
    state.saveDetail = detail || "";
    state.root.setAttribute("data-content-editor-save-state", value);
    if (detail) state.root.setAttribute("data-content-editor-save-detail", detail);
    else state.root.removeAttribute("data-content-editor-save-detail");
    syncSaveStatus(state);
    state.root.dispatchEvent(new CustomEvent("gosxstudio:content-editor-save-state", {
      bubbles: true,
      detail: { state: value, message: state.saveDetail, root: state.root, dirty: formIsDirty(state) }
    }));
  }

  function pushHistory(state) {
    if (state.suppressHistory) return;
    state.history.push({ source: state.source.value, blocks: clone(state.blocks), envelope: clone(state.envelope), selectedID: state.selectedID });
    if (state.history.length > HISTORY_LIMIT) state.history.shift();
    state.redo = [];
    syncStatus(state);
  }

  function restoreSnapshot(state, snap) {
    if (!snap) return;
    state.suppressHistory = true;
    state.source.value = snap.source;
    state.blocks = clone(snap.blocks) || [];
    state.envelope = clone(snap.envelope) || { version: 1 };
    state.selectedID = snap.selectedID || "";
    state.sourceState = "ready";
    state.sourceError = "";
    state.suppressHistory = false;
    state.source.dispatchEvent(new Event("input", { bubbles: true }));
    render(state);
    syncStatus(state);
    focusHandle(state, state.selectedID);
  }

  function undo(state) {
    if (!state.history.length) return;
    state.redo.push({ source: state.source.value, blocks: clone(state.blocks), envelope: clone(state.envelope), selectedID: state.selectedID });
    restoreSnapshot(state, state.history.pop());
  }

  function redo(state) {
    if (!state.redo.length) return;
    state.history.push({ source: state.source.value, blocks: clone(state.blocks), envelope: clone(state.envelope), selectedID: state.selectedID });
    restoreSnapshot(state, state.redo.pop());
  }

  function commit(state, options) {
    options = options || {};
    if (state.format.value !== "blocks" || state.sourceState === "invalid") return;
    if (options.history !== false) pushHistory(state);
    writeSource(state);
    syncStatus(state);
  }

  function blockIndex(state, id) {
    for (var i = 0; i < state.blocks.length; i++) {
      if (state.blocks[i].id === id) return i;
    }
    return -1;
  }

  function selectBlock(state, id, focus) {
    state.selectedID = id || "";
    render(state);
    if (focus) focusHandle(state, id);
  }

  function focusCollapseHandle(state, id) {
    if (!id) return;
    var row = state.root.querySelector('[data-content-block-id="' + cssEscape(id) + '"]');
    var button = row && row.querySelector("[data-content-editor-collapse]");
    if (button) button.focus();
  }

  function toggleCollapse(state, id) {
    if (!id || blockIndex(state, id) < 0) return;
    state.collapsedByID[id] = state.collapsedByID[id] !== true;
    render(state);
    focusCollapseHandle(state, id);
  }

  function focusHandle(state, id) {
    if (!id) return;
    var row = state.list.querySelector('[data-content-block-id="' + cssEscape(id) + '"]');
    var handle = row && row.querySelector("[data-content-drag-handle]");
    if (handle) handle.focus();
  }

  function replaceBlock(state, id, patch) {
    var index = blockIndex(state, id);
    if (index < 0) return;
    pushHistory(state);
    Object.keys(patch).forEach(function (key) { state.blocks[index][key] = patch[key]; });
    writeSource(state);
    renderBlockPreview(state, id);
    syncStatus(state);
  }

  function removeBlock(state, id) {
    var index = blockIndex(state, id);
    if (index < 0) return;
    pushHistory(state);
    state.blocks.splice(index, 1);
    state.selectedID = state.blocks[Math.min(index, state.blocks.length - 1)] ? state.blocks[Math.min(index, state.blocks.length - 1)].id : "";
    writeSource(state);
    render(state);
    focusHandle(state, state.selectedID);
  }

  function duplicateBlock(state, id) {
    var index = blockIndex(state, id);
    if (index < 0) return;
    pushHistory(state);
    var copy = normalizeBlock(clone(state.blocks[index]));
    copy.id = uid();
    state.blocks.splice(index + 1, 0, copy);
    state.selectedID = copy.id;
    writeSource(state);
    render(state);
    focusHandle(state, copy.id);
  }

  function moveBlock(state, id, delta) {
    var index = blockIndex(state, id);
    var target = index + delta;
    if (index < 0 || target < 0 || target >= state.blocks.length) return;
    pushHistory(state);
    var item = state.blocks.splice(index, 1)[0];
    state.blocks.splice(target, 0, item);
    state.selectedID = id;
    writeSource(state);
    render(state);
    focusHandle(state, id);
  }

  function insertBlock(state, type, targetID, position) {
    if (state.sourceState === "invalid") return;
    pushHistory(state);
    var block = defaultBlock(type);
    var index = typeof targetID === "string" ? blockIndex(state, targetID) : -1;
    if (index < 0) index = state.selectedID ? blockIndex(state, state.selectedID) + 1 : state.blocks.length;
    else if (position === "after") index += 1;
    state.blocks.splice(Math.max(0, index), 0, block);
    state.selectedID = block.id;
    writeSource(state);
    render(state);
    focusHandle(state, block.id);
  }

  function insertMediaBlock(state, media, targetID, position) {
    if (!media || !media.url) return;
    pushHistory(state);
    var block = normalizeBlock({ type: "image", url: media.url, alt: media.alt || media.label || "" });
    var index = typeof targetID === "string" ? blockIndex(state, targetID) : -1;
    if (index < 0) index = state.selectedID ? blockIndex(state, state.selectedID) + 1 : state.blocks.length;
    else if (position === "after") index += 1;
    state.blocks.splice(Math.max(0, index), 0, block);
    state.selectedID = block.id;
    writeSource(state);
    render(state);
    focusHandle(state, block.id);
  }

  function reorderBlock(state, id, targetID, position) {
    var from = blockIndex(state, id);
    var to = blockIndex(state, targetID);
    if (from < 0 || to < 0 || id === targetID) return;
    if (position === "after") to += 1;
    if (from < to) to -= 1;
    if (from === to) return;
    pushHistory(state);
    var item = state.blocks.splice(from, 1)[0];
    state.blocks.splice(Math.max(0, Math.min(to, state.blocks.length)), 0, item);
    state.selectedID = id;
    writeSource(state);
    render(state);
    focusHandle(state, id);
  }

  function runAction(state, action) {
    var id = action.getAttribute("data-content-editor-block") || "";
    var kind = action.getAttribute("data-content-editor-action");
    if (kind === "move-up") moveBlock(state, id, -1);
    else if (kind === "move-down") moveBlock(state, id, 1);
    else if (kind === "duplicate") duplicateBlock(state, id);
    else if (kind === "delete") removeBlock(state, id);
    else if (kind === "undo") undo(state);
    else if (kind === "redo") redo(state);
  }

  function writeSource(state) {
    state.internalSourceWrite = true;
    state.source.value = serializeBlocks(state);
    state.source.dispatchEvent(new Event("input", { bubbles: true }));
    state.internalSourceWrite = false;
  }

  function dropTarget(state, event) {
    var row = event.target.closest ? event.target.closest("[data-content-block-id]") : null;
    if (!row || !state.list.contains(row)) {
      if (!state.blocks.length) return { valid: true, id: "", position: "after" };
      return { valid: false };
    }
    var id = row.getAttribute("data-content-block-id");
    if (state.dragging && state.dragging.type === "move" && state.dragging.id === id) return { valid: false };
    var box = row.getBoundingClientRect();
    var position = event.clientY > box.top + box.height / 2 ? "after" : "before";
    return { valid: true, id: id, position: position, row: row };
  }

  function hasMediaDrag(event) {
    var types = event.dataTransfer && event.dataTransfer.types;
    if (!types) return false;
    return Array.prototype.indexOf.call(types, "application/x-gosx-studio-media") >= 0;
  }

  function mediaPayloadFromDrop(event) {
    if (!event.dataTransfer) return null;
    var raw = event.dataTransfer.getData("application/x-gosx-studio-media");
    if (!raw) return null;
    try {
      var parsed = JSON.parse(raw);
      var url = safePreviewURL(parsed && parsed.url);
      if (!url) return null;
      return {
        url: url,
        alt: String((parsed && parsed.alt) || ""),
        label: String((parsed && parsed.label) || ""),
        kind: String((parsed && parsed.kind) || "")
      };
    } catch (error) {
      return null;
    }
  }

  function clearDropIndicators(state) {
    state.root.removeAttribute("data-content-editor-drop-position");
    Array.prototype.forEach.call(state.list.querySelectorAll("[data-content-editor-drop-before], [data-content-editor-drop-after]"), function (row) {
      row.removeAttribute("data-content-editor-drop-before");
      row.removeAttribute("data-content-editor-drop-after");
    });
  }

  function applyDropIndicator(state, target) {
    clearDropIndicators(state);
    if (!target.valid || !target.row) return;
    var attr = target.position === "after" ? "data-content-editor-drop-after" : "data-content-editor-drop-before";
    target.row.setAttribute(attr, "true");
    state.root.setAttribute("data-content-editor-drop-position", target.position);
  }

  function cancelDrag(state) {
    endDrag(state);
  }

  function endDrag(state) {
    clearDropIndicators(state);
    Array.prototype.forEach.call(state.list.querySelectorAll("[data-content-editor-drag-preview]"), function (row) {
      row.removeAttribute("data-content-editor-drag-preview");
    });
    state.dragging = null;
  }

  function renderBlockPreview(state, id) {
    var row = state.list.querySelector('[data-content-block-id="' + cssEscape(id) + '"]');
    var block = state.blocks[blockIndex(state, id)];
    if (!row || !block) return;
    var previous = row.querySelector(".content-block__preview");
    if (previous) previous.replaceWith(renderPreview(block));
  }

  function renderFields(state, block) {
    var fields = el("div", { class: "content-block__fields" });
    if (block.type === "heading") {
      fields.appendChild(control("Heading", textInput(block.text, "Section heading", function (value) { replaceBlock(state, block.id, { text: value }); })));
      var level = el("select", { "aria-label": "Heading level" });
      ["2", "3", "4"].forEach(function (value) {
        var option = el("option", { value: value, text: "H" + value });
        if (value === normalizeHeadingLevel(block.level)) option.selected = true;
        level.appendChild(option);
      });
      level.addEventListener("change", function () { replaceBlock(state, block.id, { level: level.value }); });
      fields.appendChild(control("Level", level));
    } else if (block.type === "quote") {
      fields.appendChild(control("Quote", textArea(block.text, "Quote text", function (value) { replaceBlock(state, block.id, { text: value }); }, 3)));
    } else if (block.type === "image") {
      var altID = block.id + "-alt";
      fields.appendChild(control("Image URL", textInput(block.url, "/media/uploads/image.jpg", function (value) { replaceBlock(state, block.id, { url: value }); }, state.mediaList, { "data-media-alt-target": "#" + altID })));
      fields.appendChild(control("Alt text", textInput(block.alt, "Describe the image", function (value) { replaceBlock(state, block.id, { alt: value }); }, "", { id: altID })));
    } else if (block.type === "gallery") {
      fields.appendChild(control("Images", textArea((block.images || []).map(function (image) {
        return image.url + (image.alt ? " | " + image.alt : "");
      }).join("\n"), "One image per line: /media/a.jpg | Alt text", function (value) {
        replaceBlock(state, block.id, { images: value.split("\n").map(parseImageLine).filter(function (image) { return image.url; }) });
      }, 4, state.mediaList ? { "data-media-lines-list": state.mediaList } : null)));
    } else if (block.type === "button") {
      fields.appendChild(control("Label", textInput(block.label, "Call to action", function (value) { replaceBlock(state, block.id, { label: value }); })));
      fields.appendChild(control("URL", textInput(block.href, "/", function (value) { replaceBlock(state, block.id, { href: value }); })));
    } else if (block.type === "product") {
      fields.appendChild(control("Product reference", textInput(block.productRef, "product-slug-or-id", function (value) { replaceBlock(state, block.id, { productRef: value }); })));
    } else if (block.type === "flow") {
      fields.appendChild(control("Flow key", textInput(block.flowKey, "flow-key", function (value) { replaceBlock(state, block.id, { flowKey: value }); })));
    } else if (block.type === "paragraph") {
      fields.appendChild(control("Paragraph", textArea(block.text, "Paragraph text", function (value) { replaceBlock(state, block.id, { text: value }); }, 4)));
    } else {
      fields.appendChild(el("p", { class: "field-help", text: "This block type is not editable here. Its saved data will be preserved." }));
    }
    return fields;
  }

  function renderSafeSource(state) {
    syncSourceDiagnostic(state);
    state.source.hidden = false;
    state.list.hidden = true;
    if (state.toolbar) state.toolbar.hidden = true;
    state.root.setAttribute("data-content-editor-source-state", state.sourceState);
    state.root.setAttribute("data-content-editor-block-count", String(state.blocks.length));
  }

  function syncSourceDiagnostic(state) {
    var diagnostic = state.sourceDiagnosticNode || state.root.querySelector("[data-content-editor-source-diagnostic]");
    if (!diagnostic && !state.sourceError) {
      state.root.removeAttribute("data-content-editor-source-error");
      return;
    }
    if (!diagnostic) {
      diagnostic = el("p", {
        class: "field-help",
        "data-content-editor-source-diagnostic": "true",
        role: "alert",
        "aria-live": "assertive"
      });
      var parent = state.source.parentNode || state.root;
      parent.insertBefore(diagnostic, state.source.parentNode ? state.source : parent.firstChild || null);
    }
    state.sourceDiagnosticNode = diagnostic;
    diagnostic.textContent = state.sourceError || "";
    diagnostic.hidden = !state.sourceError;
    if (state.sourceError) state.root.setAttribute("data-content-editor-source-error", state.sourceError);
    else state.root.removeAttribute("data-content-editor-source-error");
  }

  function render(state) {
    var blockMode = state.format.value === "blocks";
    state.root.setAttribute("data-content-editor-source-state", state.sourceState);
    state.root.setAttribute("data-content-editor-block-count", String(state.blocks.length));
    state.root.setAttribute("data-content-editor-save-state", state.saveState);
    state.root.setAttribute("data-content-editor-history-can-undo", state.history.length ? "true" : "false");
    state.root.setAttribute("data-content-editor-history-can-redo", state.redo.length ? "true" : "false");
    syncSourceDiagnostic(state);
    if (state.keyboardDragID && state.keyboardDragPreviewIndex >= 0) state.root.setAttribute("data-content-editor-keyboard-preview-index", String(state.keyboardDragPreviewIndex));
    else state.root.removeAttribute("data-content-editor-keyboard-preview-index");
    syncSaveStatus(state);
    if (!blockMode || state.sourceState === "invalid") {
      syncSearchControls(state, state.blocks.length, state.blocks.length);
      renderSafeSource(state);
      return;
    }
    state.source.hidden = true;
    state.list.hidden = false;
    if (state.toolbar) state.toolbar.hidden = false;
    state.list.replaceChildren();
    state.blocks = state.blocks.map(normalizeBlock);
    var renderBlocks = keyboardPreviewBlocks(state);
    var visibleBlocks = renderBlocks.filter(function (block) { return blockMatchesSearch(state, block); });
    syncSearchControls(state, state.blocks.length, visibleBlocks.length);
    var history = el("div", { class: "content-editor__history", "data-content-editor-history": "true" },
      el("button", { type: "button", class: "home-section-move", "data-content-editor-action": "undo", disabled: state.history.length === 0, text: "Undo" }),
      el("button", { type: "button", class: "home-section-move", "data-content-editor-action": "redo", disabled: state.redo.length === 0, text: "Redo" }),
      el("span", { class: "content-editor__status", "data-content-editor-status": "true", text: state.blocks.length + " block" + (state.blocks.length === 1 ? "" : "s") })
    );
    state.list.appendChild(history);
    if (!visibleBlocks.length) {
      state.list.appendChild(el("p", { class: "field-help", "data-content-editor-empty": "true", "data-content-editor-search-empty": String(state.searchQuery || "").trim() ? "true" : false, role: "status", text: String(state.searchQuery || "").trim() ? "No blocks match your search." : "No blocks yet." }));
      return;
    }
    visibleBlocks.forEach(function (block, index) {
      var actualIndex = blockIndex(state, block.id);
      var known = typeSupported(block.type);
      var typeSelect = el("select", { "aria-label": "Block type" });
      if (!known) {
        typeSelect.appendChild(el("option", { value: block.type, text: "Unsupported: " + block.type }));
        typeSelect.disabled = true;
      } else {
        BLOCK_TYPES.forEach(function (type) {
          var option = el("option", { value: type, text: TYPE_LABELS[type] || type });
          if (type === block.type) option.selected = true;
          typeSelect.appendChild(option);
        });
        typeSelect.addEventListener("change", function () {
          var patch = { type: typeSelect.value };
          if (typeSelect.value === "heading") patch.level = normalizeHeadingLevel(block.level);
          replaceBlock(state, block.id, patch);
          render(state);
          focusHandle(state, block.id);
        });
      }
      var collapsed = state.collapsedByID[block.id] === true;
      var bodyID = state.collapseBodyPrefix + String(block.id).replace(/[^A-Za-z0-9_-]/g, "_");
      var collapseButton = el("button", { type: "button", class: "home-section-move", "aria-label": collapsed ? "Expand block" : "Collapse block", "aria-expanded": collapsed ? "false" : "true", "aria-controls": bodyID, "data-content-editor-collapse": "true", "data-content-editor-block": block.id, text: collapsed ? "Expand" : "Collapse" });
      var handle = el("button", { type: "button", class: "home-section-handle", "aria-label": "Move block", "aria-pressed": state.keyboardDragID === block.id ? "true" : "false", "data-content-drag-handle": "true", draggable: "true", text: "Drag" });
      bindKeyboardHandle(state, handle, block.id);
      var header = el("div", { class: "content-block__header" },
        handle,
        typeSelect,
        collapseButton,
        el("button", { type: "button", class: "home-section-move", "aria-label": "Move block up", "data-content-editor-action": "move-up", "data-content-editor-block": block.id, "data-content-editor-touch-fallback": "true", disabled: actualIndex === 0, text: "Up" }),
        el("button", { type: "button", class: "home-section-move", "aria-label": "Move block down", "data-content-editor-action": "move-down", "data-content-editor-block": block.id, "data-content-editor-touch-fallback": "true", disabled: actualIndex === state.blocks.length - 1, text: "Down" }),
        el("button", { type: "button", class: "home-section-move", "data-content-editor-action": "duplicate", "data-content-editor-block": block.id, text: "Duplicate" }),
        el("button", { type: "button", class: "home-section-move", "data-content-editor-action": "delete", "data-content-editor-block": block.id, text: "Remove" })
      );
      var row = el("article", { class: "content-block", tabindex: "0", "data-content-block-id": block.id, "data-content-block-type": block.type, "data-content-editor-selected": state.selectedID === block.id ? "true" : "false", "data-content-editor-collapsed": collapsed ? "true" : "false", "data-content-editor-keyboard-dragging": state.keyboardDragID === block.id ? "true" : "false", "data-content-editor-keyboard-preview": state.keyboardDragID === block.id ? "true" : "false" },
        header,
        el("div", { class: "content-block__body", id: bodyID, hidden: collapsed }, renderFields(state, block), renderPreview(block))
      );
      state.list.appendChild(row);
    });
    state.root.dispatchEvent(new CustomEvent("gosxstudio:content-editor-render", {
      bubbles: true,
      detail: { root: state.root, source: state.source, blockCount: state.blocks.length }
    }));
  }

  function syncStatus(state) {
    var dirty = formIsDirty(state);
    state.root.setAttribute("data-content-editor-dirty", dirty ? "true" : "false");
    state.root.setAttribute("data-content-editor-block-count", String(state.blocks.length));
    if (dirty && (state.saveState === "saved" || state.saveState === "idle")) setSaveState(state, "dirty", "Changes waiting to be saved.");
    syncSaveStatus(state);
  }

  function cssEscape(value) {
    if (window.CSS && typeof window.CSS.escape === "function") return window.CSS.escape(value);
    return String(value).replace(/["\\]/g, "\\$&");
  }

  function dirtyEditorStates() {
    return Array.prototype.filter.call(document.querySelectorAll("[data-content-editor]"), function (root) {
      var state = editorStates.get(root);
      return state && !state.intentionalDiscard && formIsDirty(state);
    });
  }

  function confirmEditorNavigation() {
    if (typeof window.confirm !== "function") return false;
    try {
      return window.confirm("You have unsaved editor changes. Leave this page?");
    } catch (error) {
      return false;
    }
  }

  function markEditorNavigationDiscard(states) {
    states.forEach(function (state) { state.intentionalDiscard = true; });
  }

  function navigationAnchor(event) {
    var target = event && event.target;
    return target && target.closest ? target.closest("a[href]") : null;
  }

  function sameOriginNavigationAnchor(anchor) {
    if (!anchor || !anchor.getAttribute || anchor.getAttribute("data-content-editor-discard") === "true") return false;
    if (anchor.getAttribute("target") || anchor.hasAttribute("download")) return false;
    var href = String(anchor.getAttribute("href") || "");
    if (!href || href.charAt(0) === "#") return false;
    try {
      var url = new URL(href, window.location.href);
      return url.origin === window.location.origin;
    } catch (error) {
      return false;
    }
  }

  function guardEditorLinkNavigation(event) {
    if (!event || event.defaultPrevented) return;
    if (event.button != null && event.button !== 0) return;
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    var anchor = navigationAnchor(event);
    if (!sameOriginNavigationAnchor(anchor)) return;
    var dirty = dirtyEditorStates();
    if (!dirty.length) return;
    if (confirmEditorNavigation()) {
      markEditorNavigationDiscard(dirty);
      return;
    }
    event.preventDefault();
    if (typeof event.stopImmediatePropagation === "function") event.stopImmediatePropagation();
  }

  function guardEditorBeforeNavigation(event) {
    if (!event || event.defaultPrevented) return;
    var detail = event.detail || {};
    if (detail.intentionalDiscard === true || detail.discard === true) return;
    var dirty = dirtyEditorStates();
    if (!dirty.length) return;
    if (confirmEditorNavigation()) {
      markEditorNavigationDiscard(dirty);
      return;
    }
    event.preventDefault();
  }

  var lastCommittedNavigationURL = String(window.location && window.location.href || "");

  function noteCommittedNavigation(event) {
    var detail = event && event.detail;
    if (detail && detail.url) lastCommittedNavigationURL = String(detail.url);
    else lastCommittedNavigationURL = String(window.location && window.location.href || lastCommittedNavigationURL);
  }

  function guardEditorHistoryNavigation(event) {
    var dirty = dirtyEditorStates();
    var nextURL = String(window.location && window.location.href || "");
    if (!dirty.length || !nextURL || nextURL === lastCommittedNavigationURL) return;
    if (confirmEditorNavigation()) {
      markEditorNavigationDiscard(dirty);
      lastCommittedNavigationURL = nextURL;
      return;
    }
    if (event && typeof event.preventDefault === "function") event.preventDefault();
    if (event && typeof event.stopImmediatePropagation === "function") event.stopImmediatePropagation();
    try {
      if (window.history && typeof window.history.pushState === "function") window.history.pushState(window.history.state, "", lastCommittedNavigationURL);
    } catch (error) {
      return;
    }
  }

  function init(root) {
    Array.prototype.forEach.call((root || document).querySelectorAll("[data-content-editor]"), initEditor);
  }

  window.GoSXStudioContentEditorRuntime = {
    init: init,
    initEditor: initEditor,
    parseBlocks: parseBlocks,
    serializeBlocks: serializeBlocks
  };

  document.addEventListener("click", guardEditorLinkNavigation, true);
  document.addEventListener("gosx:before-navigate", guardEditorBeforeNavigation);
  document.addEventListener("gosx:navigation:before", guardEditorBeforeNavigation);
  document.addEventListener("gosx:navigate", noteCommittedNavigation);
  window.addEventListener("popstate", guardEditorHistoryNavigation, true);

  window.addEventListener("beforeunload", function (event) {
    var dirty = dirtyEditorStates().length > 0;
    if (!dirty) return;
    event.preventDefault();
    event.returnValue = "";
    return "";
  });

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () { init(document); });
  } else {
    init(document);
  }
  document.addEventListener("gosx:navigate", function () { init(document); });
  document.addEventListener("gosx:render", function () { init(document); });
})();
