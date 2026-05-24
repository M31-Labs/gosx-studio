(function () {
  "use strict";

  var PREVIEW_DRAG_THRESHOLD = 7;
  var PREVIEW_SCROLL_ZONE = 64;
  var PREVIEW_SCROLL_MAX = 28;
  var previewControllers = [];

  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function blockLayoutRuntime() {
    return window.GoSXStudioBlockLayoutRuntime || {};
  }

  function rows(list) {
    var runtime = blockLayoutRuntime();
    return runtime.rows ? runtime.rows(list) : [];
  }

  function rowKey(row) {
    var runtime = blockLayoutRuntime();
    return runtime.rowKey ? runtime.rowKey(row) : "";
  }

  function rowForKey(root, key) {
    var runtime = blockLayoutRuntime();
    return runtime.rowForKey ? runtime.rowForKey(root, key) : null;
  }

  function moveRow(list, row, direction) {
    var runtime = blockLayoutRuntime();
    if (runtime.moveRow) runtime.moveRow(list, row, direction);
  }

  function selectRow(root, key) {
    var runtime = blockLayoutRuntime();
    if (runtime.selectRow) runtime.selectRow(root, key);
  }

  function commitReorder(list, key, targetKey, position) {
    var runtime = blockLayoutRuntime();
    return runtime.commitReorder ? runtime.commitReorder(list, key, targetKey, position) : false;
  }

  function cloneTemplateElement(template) {
    var node = template && template.content ? template.content.firstElementChild : null;
    return node ? node.cloneNode(true) : null;
  }

  function setTemplateText(root, selector, value) {
    var node = root.querySelector(selector);
    if (node) node.textContent = value || "";
  }

  function bindPreview(form) {
    if (!form || form.dataset.gosxStudioPreviewBound === "true") return;
    form.dataset.gosxStudioPreviewBound = "true";
    var frame = form.querySelector(".editor-preview-frame");
    var shell = form.querySelector(".editor-preview-shell");
    var list = form.querySelector("[data-block-studio]");
    if (!frame || !shell || !list) return;
    shell.classList.add("editor-preview-shell--studio");
    var overlay = shell.querySelector("[data-studio-preview-overlay]");
    if (!overlay) return;
    var dropLine = overlay.querySelector("[data-studio-preview-drop-line]");
    if (!dropLine) return;
    var dock = overlay.querySelector("[data-studio-preview-dock]");
    if (!dock) return;
    var outlineTemplate = overlay.querySelector("[data-studio-preview-outline-template]");
    var fieldTemplate = overlay.querySelector("[data-studio-preview-field-template]");
    if (!outlineTemplate || !fieldTemplate) return;
    var drag = null;
    var inlineEdit = null;
    var scrollFrame = 0;
    var renderRetries = 0;

    function render() {
      Array.prototype.slice.call(overlay.querySelectorAll(".studio-preview-outline, .studio-preview-field-hotspot")).forEach(function (node) {
        node.remove();
      });
      var doc = frame.contentDocument;
      if (!doc) {
        scheduleRenderRetry(render);
        return;
      }
      var shellBox = shell.getBoundingClientRect();
      var frameBox = frame.getBoundingClientRect();
      var rendered = 0;
      Array.prototype.forEach.call(doc.querySelectorAll("[data-studio-node-id]"), function (node) {
        var key = node.getAttribute("data-studio-block-key");
        if (!key) return;
        var label = labelForKey(key);
        var rect = node.getBoundingClientRect();
        if (rect.width <= 0 || rect.height <= 0) return;
        var outline = cloneTemplateElement(outlineTemplate);
        if (!outline) return;
        outline.setAttribute("data-studio-preview-key", key);
        outline.setAttribute("data-studio-preview-label", label);
        outline.setAttribute("aria-label", "Select and reorder " + label + " block");
        outline.style.left = (frameBox.left - shellBox.left + rect.left) + "px";
        outline.style.top = (frameBox.top - shellBox.top + rect.top) + "px";
        outline.style.width = Math.max(0, rect.width) + "px";
        outline.style.height = Math.max(0, rect.height) + "px";
        if (drag && drag.key === key) outline.classList.add(drag.active ? "is-dragging" : "is-pressing");
        setTemplateText(outline, "[data-studio-preview-outline-label]", label);
        overlay.appendChild(outline);
        rendered += 1;
      });
      syncOverlaySelection(list, overlay, dock);
      syncPreviewDockMeta();
      renderFieldHotspots(selectedKey());
      if (rendered === 0 && renderRetries < 20) {
        scheduleRenderRetry(render);
      } else if (rendered > 0) {
        renderRetries = 0;
      }
    }

    function scheduleRenderRetry(fn) {
      if (renderRetries >= 20) return;
      renderRetries += 1;
      window.setTimeout(fn, 120);
    }

    overlay.addEventListener("click", function (event) {
      var action = event.target.closest("[data-studio-preview-action]");
      if (action && overlay.contains(action)) {
        event.preventDefault();
        runDockAction(action.getAttribute("data-studio-preview-action"));
        return;
      }
      var fieldHotspot = event.target.closest("[data-studio-preview-field]");
      if (fieldHotspot && overlay.contains(fieldHotspot)) {
        event.preventDefault();
        runFieldHotspot(fieldHotspot);
        return;
      }
      var outline = event.target.closest("[data-studio-preview-key]");
      if (!outline || drag) return;
      var key = outline.getAttribute("data-studio-preview-key");
      selectRow(list, key);
      syncOverlaySelection(list, overlay, dock);
      syncPreviewDockMeta();
      if (form.getAttribute("data-studio-mode") === "content") {
        startInlineEdit(editableAtPoint(event), event);
      }
    });

    overlay.addEventListener("dblclick", function (event) {
      var outline = event.target.closest("[data-studio-preview-key]");
      if (!outline || drag) return;
      event.preventDefault();
      var key = outline.getAttribute("data-studio-preview-key");
      selectRow(list, key);
      syncOverlaySelection(list, overlay, dock);
      syncPreviewDockMeta();
      startInlineEdit(editableAtPoint(event) || firstEditableForKey(key), event);
    });

    overlay.addEventListener("pointerdown", function (event) {
      if (event.target.closest("[data-studio-preview-field]")) return;
      var outline = event.target.closest("[data-studio-preview-key]");
      if (!outline || event.button !== 0) return;
      finishInlineEdit(true);
      event.preventDefault();
      var key = outline.getAttribute("data-studio-preview-key");
      selectRow(list, key);
      syncOverlaySelection(list, overlay, dock);
      syncPreviewDockMeta();
      drag = {
        key: key,
        outline: outline,
        pointerId: event.pointerId,
        startX: event.clientX,
        startY: event.clientY,
        active: false,
        targetKey: "",
        position: "before",
        scrollDelta: 0
      };
      outline.classList.add("is-pressing");
      outline.setPointerCapture && outline.setPointerCapture(event.pointerId);
    });
    overlay.addEventListener("pointermove", function (event) {
      if (!drag) return;
      var moved = Math.hypot(event.clientX - drag.startX, event.clientY - drag.startY);
      if (!drag.active && moved < PREVIEW_DRAG_THRESHOLD) return;
      if (!drag.active) startPreviewDrag();
      updateDropTarget(event.clientY);
      updateFrameAutoScroll(event.clientY);
      event.preventDefault();
    });
    overlay.addEventListener("pointerup", finishDrag);
    overlay.addEventListener("pointercancel", cancelDrag);
    overlay.addEventListener("wheel", function (event) {
      if (drag && drag.active) return;
      if (!pointInsideFrame(event.clientX, event.clientY)) return;
      var win = frame.contentWindow;
      if (!win) return;
      win.scrollBy(event.deltaX, event.deltaY);
      renderSoon(render);
      event.preventDefault();
    }, { passive: false });

    overlay.addEventListener("keydown", function (event) {
      var action = event.target.closest("[data-studio-preview-action]");
      if (action && (event.key === "Enter" || event.key === " ")) {
        event.preventDefault();
        runDockAction(action.getAttribute("data-studio-preview-action"));
        return;
      }
      var fieldHotspot = event.target.closest("[data-studio-preview-field]");
      if (fieldHotspot && (event.key === "Enter" || event.key === " ")) {
        event.preventDefault();
        runFieldHotspot(fieldHotspot);
        return;
      }
      var outline = event.target.closest("[data-studio-preview-key]");
      if (!outline) return;
      var key = outline.getAttribute("data-studio-preview-key");
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        selectRow(list, key);
        syncOverlaySelection(list, overlay, dock);
        syncPreviewDockMeta();
      }
      if (event.key === "ArrowUp" || event.key === "ArrowDown") {
        var row = rowForKey(list, key);
        if (!row) return;
        event.preventDefault();
        moveRow(list, row, event.key === "ArrowUp" ? "up" : "down");
      }
    });

    function startPreviewDrag() {
      if (!drag || drag.active) return;
      drag.active = true;
      overlay.classList.add("is-preview-dragging");
      if (drag.outline) {
        drag.outline.classList.remove("is-pressing");
        drag.outline.classList.add("is-dragging");
      }
    }

    function selectedKey() {
      var selected = list.querySelector("[data-block-studio-block].is-selected");
      return rowKey(selected);
    }

    function selectedRow() {
      var key = selectedKey();
      return key ? rowForKey(list, key) : null;
    }

    function labelForKey(key) {
      var row = rowForKey(list, key);
      return row ? row.getAttribute("data-studio-block-label") || key : key;
    }

    function runDockAction(action) {
      var row = selectedRow();
      if (!row) return;
      var key = rowKey(row);
      if (action === "move-up" || action === "move-down") {
        moveRow(list, row, action === "move-up" ? "up" : "down");
        syncPreviewDOMFromList(frame, list);
        renderSoon(render);
        return;
      }
      if (action === "content" || action === "style") {
        setStudioMode(action);
        selectRow(list, key);
        return;
      }
      if (action === "inline-text") {
        setStudioMode("content");
        selectRow(list, key);
        var primaryButton = dock.querySelector('[data-studio-preview-action="inline-text"]');
        var field = primaryButton ? primaryButton.getAttribute("data-studio-preview-primary-field") || "" : "";
        var editable = primaryButton ? primaryButton.getAttribute("data-studio-preview-primary-editable") || "" : "";
        if (field && editable && editable !== "text") {
          announceFieldSelection(key, field, editable, "", "dock", primaryButton.textContent || "");
          runFieldInspectorAction(field, editable, true);
          return;
        }
        if (field && editable === "text") {
          announceFieldSelection(key, field, editable, "", "dock", primaryButton.textContent || "");
          if (startInlineEdit(editableNodeByField(key, field), null, true)) return;
        }
        startInlineEditForKey(key, true);
        return;
      }
      if (action === "toggle-visibility") {
        var checkbox = row.querySelector("[data-editor-block-visible]");
        if (!checkbox) return;
        checkbox.checked = !checkbox.checked;
        checkbox.dispatchEvent(new Event("input", { bubbles: true }));
        checkbox.dispatchEvent(new Event("change", { bubbles: true }));
        setPreviewBlockVisibility(frame, key, checkbox.checked);
        selectRow(list, key);
        syncOverlaySelection(list, overlay, dock);
        syncPreviewDockMeta();
        renderSoon(render);
      }
    }

    function setStudioMode(mode) {
      var button = form.querySelector('[data-studio-mode-control="' + attrValue(mode) + '"]');
      if (button) button.click();
    }

    function fieldSource(field) {
      if (!field) return null;
      return form.querySelector('[data-studio-field-source="' + attrValue(field) + '"]');
    }

    function fieldControl(source) {
      if (source.matches("input, textarea, select, button, a, [tabindex]")) return source;
      return source.querySelector("input, textarea, select, button, a[href], [tabindex]") || source;
    }

    function fieldInput(field) {
      var source = fieldSource(field);
      if (!source) return null;
      return fieldControl(source);
    }

    function fieldActionTarget(field) {
      var source = fieldSource(field);
      if (!source) return { source: null, control: null, href: "" };
      var control = fieldControl(source);
      var href = source.getAttribute("data-studio-field-action-href") || "";
      if (!href) {
        var link = source.querySelector("a[href]");
        if (link) href = link.getAttribute("href") || "";
      }
      return { source: source, control: control, href: href };
    }

    function selectedPreviewBlock(key) {
      var doc = frame.contentDocument;
      if (!doc || !key) return null;
      return doc.querySelector('[data-studio-block-key="' + attrValue(key) + '"]');
    }

    function fieldLabel(field, editable, explicit) {
      if (explicit) return explicit;
      var value = String(field || "field").split(".").pop();
      value = value.replace(/([a-z])([A-Z])/g, "$1 $2").replace(/[-_]/g, " ");
      value = value.charAt(0).toUpperCase() + value.slice(1);
      if (editable === "media") return "Media";
      if (editable === "source") return value + " source";
      if (editable === "flow") return value + " flow";
      if (editable === "url" || editable === "link") return value + " URL";
      return value || "Field";
    }

    function fieldActionLabel(editable, explicit) {
      if (explicit) return explicit;
      if (editable === "text") return "Edit text";
      if (editable === "media") return "Manage media";
      if (editable === "source") return "Manage source";
      if (editable === "flow") return "Configure flow";
      if (editable === "url" || editable === "link") return "Edit URL";
      return "Open field";
    }

    function fieldMetaFromNode(node) {
      var field = node.getAttribute("data-studio-field") || "";
      var editable = node.getAttribute("data-studio-editable") || "";
      return {
        field: field,
        editable: editable,
        label: fieldLabel(field, editable, node.getAttribute("data-studio-field-label") || ""),
        action: fieldActionLabel(editable, node.getAttribute("data-studio-field-action") || "")
      };
    }

    function fieldEntriesForKey(key) {
      var block = selectedPreviewBlock(key);
      if (!block) return [];
      return Array.prototype.map.call(block.querySelectorAll("[data-studio-field][data-studio-editable]"), function (node) {
        return fieldMetaFromNode(node);
      }).filter(function (entry) {
        return entry.field;
      });
    }

    function primaryFieldEntry(fields) {
      if (!fields.length) return null;
      var current = form.getAttribute("data-studio-field-selection") || "";
      if (current) {
        var selected = fields.find(function (entry) { return entry.field === current; });
        if (selected) return selected;
      }
      return fields.find(function (entry) { return entry.editable === "text"; }) || fields[0];
    }

    function syncPreviewDockMeta() {
      var key = selectedKey();
      var fields = fieldEntriesForKey(key);
      var primary = primaryFieldEntry(fields);
      var count = dock.querySelector("[data-studio-preview-field-count]");
      var inline = dock.querySelector('[data-studio-preview-action="inline-text"]');
      if (count) {
        count.textContent = fields.length ? fields.length + (fields.length === 1 ? " field" : " fields") : "No direct fields";
      }
      dock.setAttribute("data-studio-preview-fields", String(fields.length));
      if (inline) {
        inline.disabled = !primary;
        inline.textContent = primary ? primary.action : "No fields";
        if (primary) {
          inline.setAttribute("data-studio-preview-primary-field", primary.field);
          inline.setAttribute("data-studio-preview-primary-editable", primary.editable);
        } else {
          inline.removeAttribute("data-studio-preview-primary-field");
          inline.removeAttribute("data-studio-preview-primary-editable");
        }
      }
    }

    function setActiveFieldHotspot(field) {
      Array.prototype.forEach.call(overlay.querySelectorAll(".studio-preview-field-hotspot.is-active"), function (node) {
        node.classList.remove("is-active");
      });
      if (!field) return;
      var hotspot = overlay.querySelector('[data-studio-preview-field="' + attrValue(field) + '"]');
      if (hotspot) hotspot.classList.add("is-active");
    }

    function announceFieldSelection(key, field, editable, label, source, action) {
      if (key) selectRow(list, key);
      setActiveFieldHotspot(field);
      document.dispatchEvent(new CustomEvent("studio:field-select", {
        bubbles: true,
        detail: { key: key, field: field, label: label || fieldLabel(field, editable), editable: editable, action: action || fieldActionLabel(editable, ""), source: source || "preview" }
      }));
      syncPreviewDockMeta();
    }

    function editableNodeByField(key, field) {
      var block = selectedPreviewBlock(key);
      if (!block) return null;
      return block.querySelector('[data-studio-field="' + attrValue(field) + '"][data-studio-editable]');
    }

    function runFieldHotspot(hotspot) {
      var key = hotspot.getAttribute("data-studio-preview-key") || selectedKey();
      var field = hotspot.getAttribute("data-studio-preview-field") || "";
      var editable = hotspot.getAttribute("data-studio-preview-editable") || "";
      var label = hotspot.getAttribute("data-studio-preview-label") || fieldLabel(field, editable);
      var action = hotspot.getAttribute("data-studio-preview-action-label") || "";
      announceFieldSelection(key, field, editable, label, "preview", action);
      if (editable === "text") {
        setStudioMode("content");
        if (startInlineEdit(editableNodeByField(key, field), null, true)) return;
      }
      runFieldInspectorAction(field, editable, false);
    }

    function runFieldInspectorAction(field, editable, activate) {
      var target = fieldActionTarget(field);
      if (target.source) {
        setStudioMode("content");
        target.source.scrollIntoView({ behavior: "smooth", block: "center" });
        if (activate && target.href) {
          window.location.href = target.href;
          return true;
        }
        window.setTimeout(function () {
          if (target.control && target.control.focus) target.control.focus();
        }, 120);
        return true;
      }
      setStudioMode(editable === "media" || editable === "source" || editable === "flow" || editable === "url" || editable === "link" ? "content" : "style");
      return false;
    }

    function cycleField(direction) {
      var key = selectedKey();
      var fields = fieldEntriesForKey(key);
      if (!fields.length) return;
      var current = form.getAttribute("data-studio-field-selection") || "";
      var currentIndex = fields.findIndex(function (entry) { return entry.field === current; });
      var nextIndex = 0;
      if (currentIndex < 0) {
        nextIndex = direction < 0 ? fields.length - 1 : 0;
      } else {
        nextIndex = (currentIndex + direction + fields.length) % fields.length;
      }
      var next = fields[nextIndex];
      announceFieldSelection(key, next.field, next.editable, next.label, "cycle", next.action);
      renderSoon(render);
    }

    function requestInlineEdit(detail) {
      detail = detail || {};
      var key = detail.key || selectedKey();
      var field = detail.field || "";
      if (key && field) return startInlineEdit(editableNodeByField(key, field), null, true);
      if (key) return startInlineEditForKey(key, true);
      return false;
    }

    function cycleSelectedField(detail) {
      detail = detail || {};
      cycleField(detail.direction < 0 ? -1 : 1);
      return true;
    }

    function editableAtPoint(event) {
      var doc = frame.contentDocument;
      if (!doc || !event) return null;
      var rect = frame.getBoundingClientRect();
      var node = doc.elementFromPoint(event.clientX - rect.left, event.clientY - rect.top);
      return editableNode(node);
    }

    function editableNode(node) {
      if (!node || !node.closest) return null;
      return node.closest('[data-studio-editable="text"][data-studio-field]');
    }

    function firstEditableForKey(key) {
      var doc = frame.contentDocument;
      if (!doc || !key) return null;
      var block = doc.querySelector('[data-studio-block-key="' + attrValue(key) + '"]');
      if (!block) return null;
      if (block.matches && block.matches('[data-studio-editable="text"][data-studio-field]')) return block;
      return block.querySelector('[data-studio-field$=".headline"][data-studio-editable="text"], h1[data-studio-editable="text"][data-studio-field], [data-studio-editable="text"][data-studio-field]');
    }

    function startInlineEditForKey(key, selectAll) {
      return startInlineEdit(firstEditableForKey(key), null, selectAll);
    }

    function startInlineEdit(node, event, selectAll) {
      node = editableNode(node);
      if (!node) return false;
      var field = node.getAttribute("data-studio-field") || "";
      var input = fieldInput(field);
      if (!input) return false;
      var block = node.closest("[data-studio-block-key]");
      var key = block ? block.getAttribute("data-studio-block-key") || "" : "";
      if (key) selectRow(list, key);
      if (inlineEdit && inlineEdit.node !== node) finishInlineEdit(true);
      inlineEdit = {
        node: node,
        input: input,
        field: field,
        initial: node.textContent || ""
      };
      shell.classList.add("is-inline-editing");
      overlay.classList.add("is-inline-editing");
      form.setAttribute("data-studio-inline-field", field);
      node.classList.add("is-studio-inline-editing");
      node.setAttribute("contenteditable", "plaintext-only");
      node.setAttribute("spellcheck", "true");
      node.focus({ preventScroll: true });
      if (event && selectAll !== true) placeCaretFromPoint(node.ownerDocument, node, event);
      else selectEditableText(node.ownerDocument, node);
      document.dispatchEvent(new CustomEvent("studio:inline-edit-start", {
        bubbles: true,
        detail: { key: key, field: field, label: fieldLabel(field, "text"), editable: "text" }
      }));
      return true;
    }

    function syncInlineInput(commit) {
      if (!inlineEdit || !inlineEdit.node || !inlineEdit.input) return;
      var value = inlineEdit.node.textContent || "";
      if (inlineEdit.input.value !== value) inlineEdit.input.value = value;
      inlineEdit.input.dispatchEvent(new Event("input", { bubbles: true }));
      if (commit) inlineEdit.input.dispatchEvent(new Event("change", { bubbles: true }));
    }

    function finishInlineEdit(commit) {
      if (!inlineEdit) return;
      var node = inlineEdit.node;
      if (commit) syncInlineInput(true);
      node.classList.remove("is-studio-inline-editing");
      node.removeAttribute("contenteditable");
      shell.classList.remove("is-inline-editing");
      overlay.classList.remove("is-inline-editing");
      form.removeAttribute("data-studio-inline-field");
      inlineEdit = null;
      renderSoon(render);
      document.dispatchEvent(new CustomEvent("studio:inline-edit-end", { bubbles: true }));
    }

    function cancelInlineEdit() {
      if (!inlineEdit) return;
      inlineEdit.node.textContent = inlineEdit.initial;
      if (inlineEdit.input.value !== inlineEdit.initial) inlineEdit.input.value = inlineEdit.initial;
      inlineEdit.input.dispatchEvent(new Event("input", { bubbles: true }));
      finishInlineEdit(false);
    }

    function selectEditableText(doc, node) {
      var selection = doc.getSelection && doc.getSelection();
      if (!selection || !doc.createRange) return;
      var range = doc.createRange();
      range.selectNodeContents(node);
      selection.removeAllRanges();
      selection.addRange(range);
    }

    function placeCaretFromPoint(doc, node, event) {
      var rect = frame.getBoundingClientRect();
      var x = event.clientX - rect.left;
      var y = event.clientY - rect.top;
      var range = null;
      if (doc.caretRangeFromPoint) {
        range = doc.caretRangeFromPoint(x, y);
      } else if (doc.caretPositionFromPoint) {
        var position = doc.caretPositionFromPoint(x, y);
        if (position && doc.createRange) {
          range = doc.createRange();
          range.setStart(position.offsetNode, position.offset);
        }
      }
      if (!range || !node.contains(range.startContainer)) {
        selectEditableText(doc, node);
        return;
      }
      var selection = doc.getSelection && doc.getSelection();
      if (!selection) return;
      range.collapse(true);
      selection.removeAllRanges();
      selection.addRange(range);
    }

    function bindInlineDocument(doc) {
      if (!doc || !doc.documentElement || doc.documentElement.dataset.studioInlineEditBound === "true") return;
      doc.documentElement.dataset.studioInlineEditBound = "true";
      doc.addEventListener("input", function (event) {
        if (inlineEdit && event.target === inlineEdit.node) syncInlineInput(false);
      });
      doc.addEventListener("keydown", function (event) {
        if (!inlineEdit || event.target !== inlineEdit.node) return;
        if (event.key === "Escape") {
          event.preventDefault();
          cancelInlineEdit();
          return;
        }
        if (event.key === "Enter" && !event.shiftKey) {
          event.preventDefault();
          finishInlineEdit(true);
        }
      });
      doc.addEventListener("click", function (event) {
        if (inlineEdit && event.target && event.target.closest && event.target.closest(".is-studio-inline-editing[href]")) {
          event.preventDefault();
        }
      }, true);
      doc.addEventListener("blur", function (event) {
        if (inlineEdit && event.target === inlineEdit.node) finishInlineEdit(true);
      }, true);
    }

    function renderFieldHotspots(selectedKeyValue) {
      Array.prototype.slice.call(overlay.querySelectorAll(".studio-preview-field-hotspot")).forEach(function (node) {
        node.remove();
      });
      if (!selectedKeyValue || form.getAttribute("data-studio-mode") === "preview" || inlineEdit) return;
      var doc = frame.contentDocument;
      if (!doc) return;
      var block = doc.querySelector('[data-studio-block-key="' + attrValue(selectedKeyValue) + '"]');
      if (!block) return;
      var shellBox = shell.getBoundingClientRect();
      var frameBox = frame.getBoundingClientRect();
      Array.prototype.forEach.call(block.querySelectorAll("[data-studio-field][data-studio-editable]"), function (node) {
        var rect = node.getBoundingClientRect();
        if (rect.width <= 0 || rect.height <= 0) return;
        var meta = fieldMetaFromNode(node);
        var field = meta.field;
        var editable = meta.editable;
        var hotspot = cloneTemplateElement(fieldTemplate);
        if (!hotspot) return;
        hotspot.setAttribute("data-studio-preview-key", selectedKeyValue);
        hotspot.setAttribute("data-studio-preview-field", field);
        hotspot.setAttribute("data-studio-preview-editable", editable);
        hotspot.setAttribute("data-studio-preview-label", meta.label);
        hotspot.setAttribute("data-studio-preview-action-label", meta.action);
        hotspot.setAttribute("aria-label", meta.action + ": " + meta.label);
        hotspot.style.left = (frameBox.left - shellBox.left + rect.left) + "px";
        hotspot.style.top = (frameBox.top - shellBox.top + rect.top) + "px";
        hotspot.style.width = Math.max(0, rect.width) + "px";
        hotspot.style.height = Math.max(0, rect.height) + "px";
        setTemplateText(hotspot, "[data-studio-preview-field-action-text]", meta.action);
        if (form.getAttribute("data-studio-field-selection") === field) hotspot.classList.add("is-active");
        if (editable !== "text") hotspot.classList.add("is-reference-field");
        if (editable !== "text" && !fieldSource(field)) hotspot.classList.add("is-inspector-only");
        overlay.appendChild(hotspot);
      });
    }

    function updateDropTarget(y) {
      if (!drag || !drag.active) return;
      var outlines = Array.prototype.slice.call(overlay.querySelectorAll(".studio-preview-outline")).filter(function (outline) {
        return outline.getAttribute("data-studio-preview-key") !== drag.key;
      }).sort(function (a, b) {
        return a.getBoundingClientRect().top - b.getBoundingClientRect().top;
      });
      Array.prototype.forEach.call(overlay.querySelectorAll(".is-drop-target"), function (node) {
        node.classList.remove("is-drop-target");
      });
      if (outlines.length === 0) {
        dropLine.classList.remove("is-visible");
        drag.targetKey = "";
        return;
      }
      var target = outlines[outlines.length - 1];
      var position = "after";
      for (var index = 0; index < outlines.length; index += 1) {
        var candidateRect = outlines[index].getBoundingClientRect();
        if (y < candidateRect.top + candidateRect.height / 2) {
          target = outlines[index];
          position = "before";
          break;
        }
      }
      var rect = target.getBoundingClientRect();
      var shellBox = shell.getBoundingClientRect();
      drag.targetKey = target.getAttribute("data-studio-preview-key");
      drag.position = position;
      target.classList.add("is-drop-target");
      dropLine.style.top = ((position === "after" ? rect.bottom : rect.top) - shellBox.top) + "px";
      dropLine.style.left = (rect.left - shellBox.left) + "px";
      dropLine.style.width = rect.width + "px";
      dropLine.classList.add("is-visible");
    }

    function finishDrag() {
      if (!drag) return;
      if (!drag.active) {
        selectRow(list, drag.key);
        cancelDrag();
        return;
      }
      var changed = drag.targetKey && commitReorder(list, drag.key, drag.targetKey, drag.position);
      if (changed) reorderPreviewDOM(frame, drag.key, drag.targetKey, drag.position);
      cancelDrag();
      renderSoon(render);
    }

    function cancelDrag() {
      Array.prototype.forEach.call(overlay.querySelectorAll(".is-dragging, .is-pressing, .is-drop-target"), function (node) {
        node.classList.remove("is-dragging");
        node.classList.remove("is-pressing");
        node.classList.remove("is-drop-target");
      });
      overlay.classList.remove("is-preview-dragging");
      dropLine.classList.remove("is-visible");
      stopFrameAutoScroll();
      drag = null;
    }

    function pointInsideFrame(x, y) {
      var rect = frame.getBoundingClientRect();
      return x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom;
    }

    function updateFrameAutoScroll(y) {
      if (!drag || !drag.active) return;
      var rect = frame.getBoundingClientRect();
      var nextDelta = 0;
      if (y > rect.top && y < rect.bottom) {
        if (y - rect.top < PREVIEW_SCROLL_ZONE) {
          nextDelta = -scrollDeltaForDistance(y - rect.top);
        } else if (rect.bottom - y < PREVIEW_SCROLL_ZONE) {
          nextDelta = scrollDeltaForDistance(rect.bottom - y);
        }
      }
      drag.scrollDelta = nextDelta;
      if (nextDelta !== 0 && !scrollFrame) {
        scrollFrame = requestAnimationFrame(scrollPreviewFrame);
      }
    }

    function scrollDeltaForDistance(distance) {
      var intensity = Math.max(0, Math.min(1, (PREVIEW_SCROLL_ZONE - distance) / PREVIEW_SCROLL_ZONE));
      return Math.max(3, Math.round(PREVIEW_SCROLL_MAX * intensity));
    }

    function scrollPreviewFrame() {
      scrollFrame = 0;
      if (!drag || !drag.active || !drag.scrollDelta) return;
      var win = frame.contentWindow;
      if (win) {
        win.scrollBy(0, drag.scrollDelta);
        renderSoon(render);
      }
      scrollFrame = requestAnimationFrame(scrollPreviewFrame);
    }

    function stopFrameAutoScroll() {
      if (scrollFrame) cancelAnimationFrame(scrollFrame);
      scrollFrame = 0;
    }

    function bindFrameDocument() {
      var doc = frame.contentDocument;
      if (!doc || !doc.documentElement) return;
      if (doc.documentElement.dataset.studioPreviewScrollBound !== "true") {
        doc.documentElement.dataset.studioPreviewScrollBound = "true";
        doc.addEventListener("scroll", function () { renderSoon(render); }, true);
      }
      bindInlineDocument(doc);
    }

    frame.addEventListener("load", function () {
      finishInlineEdit(false);
      bindFrameDocument();
      renderSoon(render);
    });
    window.addEventListener("resize", function () { renderSoon(render); });
    list.addEventListener("blockstudio:reorder", function () {
      if (!drag || !drag.active) syncPreviewDOMFromList(frame, list);
      renderSoon(render);
    });
    document.addEventListener("blockstudio:select", function () {
      syncOverlaySelection(list, overlay, dock);
      syncPreviewDockMeta();
      renderSoon(render);
    });
    previewControllers.push({
      form: form,
      requestInlineEdit: requestInlineEdit,
      cycleField: cycleSelectedField
    });
    bindFrameDocument();
    renderSoon(render);
    window.setTimeout(render, 120);
  }

  function reorderPreviewDOM(frame, key, targetKey, position) {
    var doc = frame.contentDocument;
    if (!doc) return;
    var node = doc.querySelector('[data-studio-block-key="' + attrValue(key) + '"]');
    var target = doc.querySelector('[data-studio-block-key="' + attrValue(targetKey) + '"]');
    if (!node || !target || !target.parentNode) return;
    target.parentNode.insertBefore(node, position === "after" ? target.nextSibling : target);
  }

  function syncPreviewDOMFromList(frame, list) {
    var doc = frame.contentDocument;
    if (!doc) return;
    var nodesByKey = {};
    Array.prototype.forEach.call(doc.querySelectorAll("[data-studio-block-key]"), function (node) {
      nodesByKey[node.getAttribute("data-studio-block-key")] = node;
    });
    var ordered = rows(list).map(function (row) {
      return nodesByKey[rowKey(row)];
    }).filter(Boolean);
    if (ordered.length < 2) return;
    var parent = ordered[0].parentNode;
    if (!parent) return;
    ordered.forEach(function (node) {
      if (node.parentNode === parent) parent.appendChild(node);
    });
  }

  function setPreviewBlockVisibility(frame, key, visible) {
    var doc = frame.contentDocument;
    if (!doc) return;
    var node = doc.querySelector('[data-studio-block-key="' + attrValue(key) + '"]');
    if (node) node.style.display = visible ? "" : "none";
  }

  function setBlockVisibility(key, visible) {
    if (!key) return;
    Array.prototype.forEach.call(document.querySelectorAll(".editor-preview-frame"), function (frame) {
      try {
        setPreviewBlockVisibility(frame, key, visible);
      } catch (error) {
        return;
      }
    });
  }

  function editorDocuments() {
    var docs = [document];
    Array.prototype.forEach.call(document.querySelectorAll(".editor-preview-frame"), function (frame) {
      try {
        if (frame.contentDocument) docs.push(frame.contentDocument);
      } catch (error) {
        return;
      }
    });
    return docs;
  }

  function previewDocuments() {
    var docs = [];
    Array.prototype.forEach.call(document.querySelectorAll(".editor-preview-frame"), function (frame) {
      try {
        if (frame.contentDocument) docs.push(frame.contentDocument);
      } catch (error) {
        return;
      }
    });
    return docs;
  }

  function textTargets(root, key) {
    return Array.prototype.slice.call(root.querySelectorAll('[data-editor-preview="' + attrValue(key) + '"]'));
  }

  function applyTextUpdate(detail) {
    detail = detail || {};
    var value = detail.value || "";
    if (detail.sourceKey) {
      textTargets(document, detail.sourceKey).forEach(function (target) {
        target.textContent = value;
      });
    }
    if (detail.frameTarget) {
      editorDocuments().forEach(function (doc) {
        Array.prototype.forEach.call(doc.querySelectorAll(detail.frameTarget), function (target) {
          target.textContent = value;
        });
      });
    }
    if (detail.attrTarget && detail.attrName) {
      editorDocuments().forEach(function (doc) {
        Array.prototype.forEach.call(doc.querySelectorAll(detail.attrTarget), function (target) {
          var normalized = String(value || "").trim();
          if (!normalized) target.removeAttribute(detail.attrName);
          else target.setAttribute(detail.attrName, (detail.attrPrefix || "") + normalized + (detail.attrSuffix || ""));
        });
      });
    }
  }

  function applyTheme(detail) {
    detail = detail || {};
    var colors = detail.colors || {};
    var classes = [];
    if (detail.template) classes.push("theme--template-" + detail.template);
    if (detail.kit) classes.push("theme--kit-" + detail.kit);
    if (Array.isArray(detail.customClasses)) {
      detail.customClasses.forEach(function (className) {
        if (className) classes.push(className);
      });
    }
    if (Array.isArray(detail.styleClasses)) {
      detail.styleClasses.forEach(function (className) {
        if (className) classes.push(className);
      });
    }
    if (detail.palette) classes.push("theme--palette-" + detail.palette);
    if (detail.imageRatio) classes.push("theme--image-" + detail.imageRatio);

    editorDocuments().forEach(function (doc) {
      var shell = doc.querySelector(".site-shell");
      if (!shell) return;
      Array.prototype.slice.call(shell.classList).forEach(function (className) {
        if (className.indexOf("theme--template-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--kit-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--custom-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--nav-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--buttons-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--cards-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--spacing-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--images-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--motion-") === 0) shell.classList.remove(className);
        if (className.indexOf("theme--palette-") === 0) shell.classList.remove(className);
      });
      ["landscape", "portrait", "square"].forEach(function (value) {
        shell.classList.remove("theme--image-" + value);
      });
      classes.forEach(function (className) {
        shell.classList.add(className);
      });
      Object.keys(colors).forEach(function (name) {
        doc.documentElement.style.setProperty(name, colors[name]);
        doc.body && doc.body.style.setProperty(name, colors[name]);
        shell.style.setProperty(name, colors[name]);
      });
    });
  }

  function applyStyleImpact(selector) {
    var count = 0;
    previewDocuments().forEach(function (doc) {
      Array.prototype.forEach.call(doc.querySelectorAll("[data-studio-style-impact-node]"), function (node) {
        node.removeAttribute("data-studio-style-impact-node");
      });
      if (!selector) return;
      var scope = doc.querySelector(".site-shell") || doc;
      Array.prototype.forEach.call(scope.querySelectorAll(selector), function (node) {
        if (node.hasAttribute("data-studio-style-impact-node")) return;
        node.setAttribute("data-studio-style-impact-node", "true");
        count += 1;
      });
    });
    return count;
  }

  function ensureStyle(doc, attr) {
    var style = doc.querySelector("style[" + attr + "]");
    if (!style) {
      style = doc.createElement("style");
      style.setAttribute(attr, "true");
      doc.head.appendChild(style);
    }
    return style;
  }

  function applyPreviewStyle(attr, cssText) {
    editorDocuments().forEach(function (doc) {
      ensureStyle(doc, attr).textContent = cssText || "";
    });
  }

  function applyCSS(cssText) {
    applyPreviewStyle("data-editor-live-css", cssText);
  }

  function applyFonts(cssText) {
    applyPreviewStyle("data-editor-live-fonts", cssText);
  }

  function updateHeaderLogo(detail) {
    detail = detail || {};
    var url = detail.url || "";
    var width = detail.width || "96";
    var x = detail.x || "0";
    var y = detail.y || "0";
    var alt = detail.alt || "Site logo";
    editorDocuments().forEach(function (doc) {
      var brand = doc.querySelector(".brand");
      if (!brand) return;
      var logo = brand.querySelector(".brand__logo");
      if (!logo) return;
      brand.classList.toggle("brand--with-logo", !!url);
      logo.hidden = !url;
      logo.src = url;
      logo.alt = alt;
      brand.style.setProperty("--brand-logo-width", width + "px");
      brand.style.setProperty("--brand-logo-offset-x", x + "px");
      brand.style.setProperty("--brand-logo-offset-y", y + "px");
    });
  }

  function requestInlineEdit(detail) {
    var handled = false;
    previewControllers.forEach(function (controller) {
      if (!controller || !document.contains(controller.form)) return;
      if (controller.requestInlineEdit(detail || {})) handled = true;
    });
    return handled;
  }

  function cycleField(detail) {
    var handled = false;
    previewControllers.forEach(function (controller) {
      if (!controller || !document.contains(controller.form)) return;
      if (controller.cycleField(detail || {})) handled = true;
    });
    return handled;
  }

  function syncOverlaySelection(list, overlay, dock) {
    var selected = list.querySelector("[data-block-studio-block].is-selected");
    var key = rowKey(selected);
    var selectedOutline = null;
    Array.prototype.forEach.call(overlay.querySelectorAll("[data-studio-preview-key]"), function (outline) {
      var match = outline.getAttribute("data-studio-preview-key") === key;
      outline.classList.toggle("is-selected", match);
      if (match) selectedOutline = outline;
    });
    positionDock(list, overlay, dock, selected, selectedOutline);
  }

  function positionDock(list, overlay, dock, selectedRow, selectedOutline) {
    if (!dock) return;
    if (!selectedRow || !selectedOutline) {
      dock.hidden = true;
      return;
    }
    var title = dock.querySelector("[data-studio-preview-dock-title]");
    var visibility = dock.querySelector('[data-studio-preview-action="toggle-visibility"]');
    var up = dock.querySelector('[data-studio-preview-action="move-up"]');
    var down = dock.querySelector('[data-studio-preview-action="move-down"]');
    var checkbox = selectedRow.querySelector("[data-editor-block-visible]");
    var label = selectedRow.getAttribute("data-studio-block-label") || rowKey(selectedRow);
    var rowsNow = rows(list);
    var index = rowsNow.indexOf(selectedRow);
    if (title) title.textContent = label || "Block";
    if (visibility) visibility.textContent = checkbox && checkbox.checked ? "Hide" : "Show";
    if (up) up.disabled = index <= 0;
    if (down) down.disabled = index < 0 || index >= rowsNow.length - 1;
    var rect = selectedOutline.getBoundingClientRect();
    var overlayBox = overlay.getBoundingClientRect();
    dock.hidden = false;
    dock.style.left = Math.max(8, rect.left - overlayBox.left + 8) + "px";
    dock.style.top = Math.max(8, rect.top - overlayBox.top + 8) + "px";
  }

  var renderTimer = 0;
  function renderSoon(fn) {
    if (renderTimer) cancelAnimationFrame(renderTimer);
    renderTimer = requestAnimationFrame(function () {
      renderTimer = 0;
      fn();
    });
  }

  function init(root) {
    var scope = root && root.querySelectorAll ? root : document;
    Array.prototype.forEach.call(scope.querySelectorAll("[data-editor-workbench]"), bindPreview);
  }

  window.GoSXStudioPreviewRuntime = {
    mount: init,
    setBlockVisibility: setBlockVisibility,
    applyTextUpdate: applyTextUpdate,
    applyTheme: applyTheme,
    applyStyleImpact: applyStyleImpact,
    applyCSS: applyCSS,
    applyFonts: applyFonts,
    updateHeaderLogo: updateHeaderLogo,
    requestInlineEdit: requestInlineEdit,
    cycleField: cycleField
  };
})();
