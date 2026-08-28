(function () {
  "use strict";

  function markManagedScriptLoaded() {
    var script = document.currentScript;
    if (!script || script.tagName !== "SCRIPT" || script.getAttribute("data-gosx-script") !== "managed") return;
    script.setAttribute("data-gosx-script-loaded", "true");
  }

  markManagedScriptLoaded();
  var existingRuntime = window.GoSXStudioMediaRuntime;
  if (existingRuntime && typeof existingRuntime.init === "function") {
    existingRuntime.init(document);
    return;
  }

  var pickerID = 0;
  var mediaItemID = 0;
  var mediaDragType = "application/x-gosx-studio-media";
  var uploadControllers = [];

  function toArray(list) {
    return Array.prototype.slice.call(list || []);
  }

  function controls(root, selector) {
    var out = [];
    if (root && root.nodeType === 1 && root.matches && root.matches(selector)) out.push(root);
    return out.concat(toArray((root || document).querySelectorAll(selector)));
  }

  function mediaListFor(input) {
    var id = input && input.getAttribute("list");
    if (!id || id.indexOf("media-urls") < 0) return null;
    return document.getElementById(id);
  }

  function mediaLinesListFor(textarea) {
    var id = textarea && textarea.getAttribute("data-media-lines-list");
    if (!id || id.indexOf("media-urls") < 0) return null;
    return document.getElementById(id);
  }

  function mediaAssets(list) {
    var seen = {};
    return toArray(list.querySelectorAll("option")).map(function (option) {
      var url = safeMediaURL(String(option.getAttribute("value") || "").trim());
      if (!url || seen[url]) return null;
      seen[url] = true;
      return {
        url: url,
        label: String(option.getAttribute("label") || option.textContent || filename(url)).trim() || filename(url),
        alt: String(option.getAttribute("data-media-alt") || "").trim(),
        image: imageURL(url),
        kind: fileKind(url)
      };
    }).filter(Boolean);
  }

  function filename(url) {
    var clean = String(url || "").split("?")[0].split("#")[0];
    return clean.slice(clean.lastIndexOf("/") + 1) || clean || "Asset";
  }

  function fileKind(url) {
    var clean = String(url || "").split("?")[0].toLowerCase();
    var ext = clean.slice(clean.lastIndexOf(".") + 1);
    return ext && ext.length <= 5 ? ext : "file";
  }

  function imageURL(url) {
    var clean = String(url || "").split("?")[0].toLowerCase();
    return /\.(avif|gif|jpe?g|png|svg|webp)$/.test(clean);
  }

  function safeMediaURL(url) {
    url = String(url || "").trim();
    if (!url) return "";
    if (/^(?:javascript|data|vbscript):/i.test(url)) return "";
    if (/^(?:https?:)?\/\//i.test(url) || url.charAt(0) === "/" || url.indexOf("./") === 0 || url.indexOf("../") === 0) return url;
    return /^[A-Za-z0-9._~!$&'()*+,;=:@/-]+(?:[?#][^\s]*)?$/.test(url) ? url : "";
  }

  function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
  }

  function numberValue(input, fallback) {
    if (!input) return fallback;
    var value = parseFloat(input.value);
    return Number.isFinite(value) ? value : fallback;
  }

  function focalString(value) {
    return String(Math.round(clamp(value, 0, 1) * 1000) / 1000);
  }

  function element(tag, attrs) {
    var node = document.createElement(tag);
    Object.keys(attrs || {}).forEach(function (key) {
      if (key === "class") node.className = attrs[key];
      else if (key === "text") node.textContent = attrs[key];
      else if (key === "hidden") node.hidden = !!attrs[key];
      else if (key === "disabled") node.disabled = !!attrs[key];
      else node.setAttribute(key, attrs[key]);
    });
    for (var i = 2; i < arguments.length; i++) {
      if (arguments[i]) node.appendChild(arguments[i]);
    }
    return node;
  }

  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function namedControl(form, name) {
    if (!name) return null;
    if (/^[#.[]/.test(name)) {
      var scope = form || document;
      try {
        return (scope && scope.querySelector && scope.querySelector(name)) || document.querySelector(name);
      } catch (error) {
        return null;
      }
    }
    var selector = '[name="' + attrValue(name) + '"]';
    return (form && form.querySelector && form.querySelector(selector)) || document.querySelector(selector);
  }

  function renderThumb(asset) {
    if (asset.image) return element("img", { src: asset.url, alt: "" });
    return element("span", { class: "media-picker__file", text: asset.kind });
  }

  function setExpanded(trigger, panel, expanded) {
    trigger.setAttribute("aria-expanded", expanded ? "true" : "false");
    panel.hidden = !expanded;
  }

  function dispatchNativeChange(control) {
    control.dispatchEvent(new Event("input", { bubbles: true }));
    control.dispatchEvent(new Event("change", { bubbles: true }));
  }

  function mediaDragPayload(asset) {
    return JSON.stringify({
      url: asset.url,
      alt: asset.alt || "",
      label: asset.label || filename(asset.url),
      kind: asset.kind || fileKind(asset.url)
    });
  }

  function attachAssetDrag(button, asset) {
    button.draggable = true;
    button.setAttribute("data-gosx-studio-media-draggable", "true");
    button.addEventListener("dragstart", function (event) {
      if (!event.dataTransfer) return;
      var payload = mediaDragPayload(asset);
      event.dataTransfer.effectAllowed = "copy";
      event.dataTransfer.setData(mediaDragType, payload);
      event.dataTransfer.setData("text/uri-list", asset.url);
      event.dataTransfer.setData("text/plain", asset.url);
    });
  }

  function parseMediaLines(value) {
    return String(value || "").split("\n").map(function (line) {
      var parts = line.split("|");
      return mediaImage(parts[0], parts.slice(1).join("|"));
    }).filter(function (image) { return image.url; });
  }

  function serializeMediaLines(images) {
    return images.map(function (image) {
      var url = String((image && image.url) || "").trim();
      var alt = String((image && image.alt) || "").trim();
      return url + (alt ? " | " + alt : "");
    }).filter(Boolean).join("\n");
  }

  function mediaImage(url, alt) {
    return {
      id: "media-item-" + (++mediaItemID),
      url: String(url || "").trim(),
      alt: String(alt || "").trim()
    };
  }

  function ensureMediaItemID(image) {
    if (image && !image.id) image.id = "media-item-" + (++mediaItemID);
    return image && image.id;
  }

  function mediaResultsMessage(count, query) {
    if (!count) return query ? "No matching media for \"" + query + "\"." : "No media available.";
    return "Showing " + count + " media " + (count === 1 ? "result" : "results") + (query ? " for \"" + query + "\"" : "") + ".";
  }

  function preventFilterSubmit(event) {
    if (event.key !== "Enter") return;
    event.preventDefault();
  }

  function upgradeInput(input) {
    if (!input || input.dataset.mediaPickerBound === "true") return;
    var list = mediaListFor(input);
    if (!list) return;
    var assets = mediaAssets(list);
    if (!assets.length) return;
    input.dataset.mediaPickerBound = "true";

    var id = "media-picker-" + (++pickerID);
    var preview = element("div", { class: "media-picker__preview", "aria-hidden": "true" });
    var trigger = element("button", {
      type: "button",
      class: "button button--secondary media-picker__trigger",
      "aria-expanded": "false",
      "aria-controls": id,
      text: "Choose media"
    });
    var search = element("input", {
      type: "search",
      class: "media-picker__search",
      "aria-label": "Filter media",
      placeholder: "Filter media"
    });
    var resultStatus = element("output", {
      class: "media-picker__status",
      "data-media-picker-status": "true",
      role: "status",
      "aria-live": "polite",
      "aria-atomic": "true",
      "aria-label": "Media search results"
    });
    var grid = element("div", { class: "media-picker__grid" });
    var panel = element("div", { class: "media-picker__panel", id: id, hidden: true }, search, resultStatus, grid);
    var picker = element("div", { class: "media-picker" },
      element("div", { class: "media-picker__toolbar" }, preview, trigger),
      panel
    );

    function syncGridSelection(value) {
      toArray(grid.querySelectorAll(".media-picker__asset")).forEach(function (button) {
        var current = button.getAttribute("data-media-asset-url") === value;
        button.setAttribute("aria-current", current ? "true" : "false");
        button.setAttribute("data-media-asset-selected", current ? "true" : "false");
      });
    }

    function updatePreview() {
      var value = String(input.value || "").trim();
      syncGridSelection(value);
      preview.textContent = "";
      if (!value) {
        preview.appendChild(element("span", { class: "media-picker__empty", text: "No asset selected" }));
        return;
      }
      var current = assets.filter(function (asset) { return asset.url === value; })[0] || {
        url: safeMediaURL(value),
        label: filename(value),
        image: imageURL(value),
        kind: fileKind(value)
      };
      if (!current.url) return;
      preview.appendChild(renderThumb(current));
      preview.appendChild(element("span", { text: current.label }));
    }

    function renderGrid() {
      var query = search.value.trim();
      var normalizedQuery = query.toLowerCase();
      var value = String(input.value || "").trim();
      grid.textContent = "";
      var matches = assets.filter(function (asset) {
        return !normalizedQuery || asset.label.toLowerCase().indexOf(normalizedQuery) >= 0 || asset.url.toLowerCase().indexOf(normalizedQuery) >= 0 || asset.alt.toLowerCase().indexOf(normalizedQuery) >= 0;
      });
      matches.forEach(function (asset) {
        var button = element("button", {
          type: "button",
          class: "media-picker__asset",
          "aria-label": "Use " + asset.label,
          "aria-current": asset.url === value ? "true" : "false",
          "data-media-asset-selected": asset.url === value ? "true" : "false",
          "data-media-asset-url": asset.url
        }, renderThumb(asset), element("span", { text: asset.label }));
        attachAssetDrag(button, asset);
        button.addEventListener("click", function () {
          input.value = asset.url;
          dispatchNativeChange(input);
          var altInput = namedControl(input.form, input.getAttribute("data-media-alt-target"));
          if (altInput && asset.alt) {
            altInput.value = asset.alt;
            dispatchNativeChange(altInput);
          }
          updatePreview();
          setExpanded(trigger, panel, false);
          input.focus();
        });
        grid.appendChild(button);
      });
      if (!grid.children.length) grid.appendChild(element("p", { class: "empty", text: "No matching media." }));
      resultStatus.textContent = mediaResultsMessage(matches.length, query);
    }

    trigger.addEventListener("click", function () {
      var next = panel.hidden;
      setExpanded(trigger, panel, next);
      if (next) {
        renderGrid();
        search.focus();
      }
    });
    search.addEventListener("input", renderGrid);
    search.addEventListener("keydown", preventFilterSubmit);
    panel.addEventListener("keydown", function (event) {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        setExpanded(trigger, panel, false);
        trigger.focus();
      }
    });
    input.addEventListener("input", updatePreview);
    input.insertAdjacentElement("afterend", picker);
    updatePreview();
    renderGrid();
  }

  function upgradeMediaLines(textarea) {
    if (!textarea || textarea.dataset.mediaLinesBound === "true") return;
    var list = mediaLinesListFor(textarea);
    if (!list) return;
    var assets = mediaAssets(list).filter(function (asset) { return asset.image; });
    if (!assets.length) return;
    textarea.dataset.mediaLinesBound = "true";

    var id = "media-lines-" + (++pickerID);
    var images = parseMediaLines(textarea.value);
    var items = element("div", { class: "media-list-editor__items" });
    var rawID = textarea.id || id + "-raw";
    if (!textarea.id) textarea.id = rawID;
    var trigger = element("button", {
      type: "button",
      class: "button button--secondary media-picker__trigger",
      "aria-expanded": "false",
      "aria-controls": id,
      text: "Add media"
    });
    var rawToggle = element("button", {
      type: "button",
      class: "button button--ghost media-list-editor__raw-toggle",
      "aria-controls": rawID,
      "aria-expanded": "false",
      text: "Raw lines"
    });
    var search = element("input", {
      type: "search",
      class: "media-picker__search",
      "aria-label": "Filter media",
      placeholder: "Filter media"
    });
    var resultStatus = element("output", {
      class: "media-picker__status",
      "data-media-picker-status": "true",
      role: "status",
      "aria-live": "polite",
      "aria-atomic": "true",
      "aria-label": "Media search results"
    });
    var grid = element("div", { class: "media-picker__grid" });
    var panel = element("div", { class: "media-picker__panel", id: id, hidden: true }, search, resultStatus, grid);
    var editor = element("div", { class: "media-list-editor" },
      items,
      element("div", { class: "media-list-editor__actions" }, trigger, rawToggle),
      panel
    );
    var syncing = false;

    function syncTextarea() {
      syncing = true;
      textarea.value = serializeMediaLines(images);
      dispatchNativeChange(textarea);
      syncing = false;
    }

    function syncGridSelection() {
      toArray(grid.querySelectorAll(".media-picker__asset")).forEach(function (button) {
        var assetURL = button.getAttribute("data-media-asset-url");
        var selected = images.some(function (image) { return image.url === assetURL; });
        var label = button.getAttribute("data-media-asset-label") || "media";
        button.setAttribute("data-media-asset-selected", selected ? "true" : "false");
        button.setAttribute("aria-label", "Add " + label + (selected ? " (already in gallery)" : ""));
      });
    }

    function focusItemControl(itemID, action) {
      var item = toArray(items.querySelectorAll("[data-media-item-id]")).filter(function (candidate) {
        return candidate.getAttribute("data-media-item-id") === itemID;
      })[0];
      if (!item) return;
      var control = action ? item.querySelector('[data-media-list-action="' + action + '"]') : null;
      if (!control || control.disabled) control = item.querySelector("input[type=\"text\"]");
      if (control && typeof control.focus === "function") control.focus();
    }

    function move(index, delta, action) {
      var target = index + delta;
      if (target < 0 || target >= images.length) return;
      var itemID = ensureMediaItemID(images[index]);
      var image = images.splice(index, 1)[0];
      images.splice(target, 0, image);
      syncTextarea();
      renderItems();
      focusItemControl(itemID, action);
    }

    function remove(index) {
      var nextImage = images[index + 1] || images[index - 1];
      var nextItemID = nextImage && ensureMediaItemID(nextImage);
      images.splice(index, 1);
      syncTextarea();
      renderItems();
      if (nextItemID) focusItemControl(nextItemID, "remove");
      else trigger.focus();
    }

    function assetFor(image) {
      return assets.filter(function (asset) { return asset.url === image.url; })[0] || {
        url: image.url,
        label: filename(image.url),
        alt: image.alt,
        image: imageURL(image.url),
        kind: fileKind(image.url)
      };
    }

    function renderItems() {
      items.textContent = "";
      if (!images.length) {
        items.appendChild(element("p", { class: "empty", text: "No images selected." }));
        syncGridSelection();
        return;
      }
      images.forEach(function (image, index) {
        var itemID = ensureMediaItemID(image);
        var asset = assetFor(image);
        var altInput = element("input", {
          type: "text",
          value: image.alt || "",
          placeholder: "Alt text",
          "aria-label": "Alt text for " + asset.label + " (item " + (index + 1) + ")"
        });
        altInput.addEventListener("input", function () {
          images[index].alt = altInput.value;
          syncTextarea();
        });
        var up = element("button", { type: "button", class: "home-section-move", text: "Up", "aria-label": "Move image " + (index + 1) + " up", "data-media-list-action": "up" });
        var down = element("button", { type: "button", class: "home-section-move", text: "Down", "aria-label": "Move image " + (index + 1) + " down", "data-media-list-action": "down" });
        var removeButton = element("button", { type: "button", class: "home-section-move", text: "Remove", "aria-label": "Remove image " + (index + 1), "data-media-list-action": "remove" });
        up.disabled = index === 0;
        down.disabled = index === images.length - 1;
        up.addEventListener("click", function () { move(index, -1, "up"); });
        down.addEventListener("click", function () { move(index, 1, "down"); });
        removeButton.addEventListener("click", function () { remove(index); });
        items.appendChild(element("article", { class: "media-list-editor__item" },
          renderThumb(asset),
          element("div", { class: "media-list-editor__fields" },
            element("strong", { text: asset.label }),
            altInput,
            element("small", { text: image.url })
          ),
          element("div", { class: "media-list-editor__item-actions" }, up, down, removeButton)
        ));
        var item = items.lastElementChild;
        item.setAttribute("data-media-item-id", itemID);
        item.setAttribute("aria-label", asset.label + " media item " + (index + 1));
      });
      syncGridSelection();
    }

    function renderGrid() {
      var query = search.value.trim();
      var normalizedQuery = query.toLowerCase();
      grid.textContent = "";
      var matches = assets.filter(function (asset) {
        return !normalizedQuery || asset.label.toLowerCase().indexOf(normalizedQuery) >= 0 || asset.url.toLowerCase().indexOf(normalizedQuery) >= 0 || asset.alt.toLowerCase().indexOf(normalizedQuery) >= 0;
      });
      matches.forEach(function (asset) {
        var button = element("button", {
          type: "button",
          class: "media-picker__asset",
          "aria-label": "Add " + asset.label + (images.some(function (image) { return image.url === asset.url; }) ? " (already in gallery)" : ""),
          "data-media-asset-selected": images.some(function (image) { return image.url === asset.url; }) ? "true" : "false",
          "data-media-asset-label": asset.label,
          "data-media-asset-url": asset.url
        }, renderThumb(asset), element("span", { text: asset.label }));
        attachAssetDrag(button, asset);
        button.addEventListener("click", function () {
          images.push(mediaImage(asset.url, asset.alt || asset.label));
          syncTextarea();
          renderItems();
          setExpanded(trigger, panel, false);
          trigger.focus();
        });
        grid.appendChild(button);
      });
      if (!grid.children.length) grid.appendChild(element("p", { class: "empty", text: "No matching media." }));
      resultStatus.textContent = mediaResultsMessage(matches.length, query);
    }

    trigger.addEventListener("click", function () {
      var next = panel.hidden;
      setExpanded(trigger, panel, next);
      if (next) {
        renderGrid();
        search.focus();
      }
    });
    rawToggle.addEventListener("click", function () {
      var expanded = textarea.hidden;
      textarea.hidden = !expanded;
      rawToggle.setAttribute("aria-expanded", expanded ? "true" : "false");
      if (expanded) textarea.focus();
    });
    textarea.addEventListener("input", function () {
      if (syncing) return;
      images = parseMediaLines(textarea.value);
      renderItems();
    });
    search.addEventListener("input", renderGrid);
    search.addEventListener("keydown", preventFilterSubmit);
    panel.addEventListener("keydown", function (event) {
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
        setExpanded(trigger, panel, false);
        trigger.focus();
      }
    });

    textarea.insertAdjacentElement("afterend", editor);
    textarea.hidden = true;
    renderItems();
    renderGrid();
  }

  function upgradeFocalPreview(preview) {
    if (!preview || preview.dataset.mediaFocalBound === "true") return;
    preview.dataset.mediaFocalBound = "true";
    var form = preview.closest("form") || document;
    var xInput = form.querySelector("[data-media-focal-x]");
    var yInput = form.querySelector("[data-media-focal-y]");
    var marker = preview.querySelector("[data-media-focal-marker]");
    if (!xInput || !yInput || !marker) return;

    function updateMarker() {
      marker.style.left = (clamp(numberValue(xInput, 0), 0, 1) * 100) + "%";
      marker.style.top = (clamp(numberValue(yInput, 0), 0, 1) * 100) + "%";
    }

    function commit(x, y, emitChange) {
      xInput.value = focalString(x);
      yInput.value = focalString(y);
      xInput.dispatchEvent(new Event("input", { bubbles: true }));
      yInput.dispatchEvent(new Event("input", { bubbles: true }));
      if (emitChange) {
        xInput.dispatchEvent(new Event("change", { bubbles: true }));
        yInput.dispatchEvent(new Event("change", { bubbles: true }));
      }
      updateMarker();
    }

    function positionFromEvent(event) {
      var box = preview.getBoundingClientRect();
      return {
        x: box.width ? clamp((event.clientX - box.left) / box.width, 0, 1) : 0,
        y: box.height ? clamp((event.clientY - box.top) / box.height, 0, 1) : 0
      };
    }

    var dragging = false;
    var dragStart = null;
    preview.addEventListener("pointerdown", function (event) {
      if (event.button !== undefined && event.button !== 0) return;
      dragging = true;
      dragStart = { x: numberValue(xInput, 0), y: numberValue(yInput, 0) };
      preview.setPointerCapture(event.pointerId);
      var point = positionFromEvent(event);
      commit(point.x, point.y, false);
      event.preventDefault();
    });
    preview.addEventListener("pointermove", function (event) {
      if (!dragging) return;
      var point = positionFromEvent(event);
      commit(point.x, point.y, false);
      event.preventDefault();
    });
    preview.addEventListener("pointerup", function (event) {
      if (!dragging) return;
      dragging = false;
      dragStart = null;
      var point = positionFromEvent(event);
      commit(point.x, point.y, true);
      event.preventDefault();
    });
    preview.addEventListener("pointercancel", function (event) {
      if (dragging && dragStart) commit(dragStart.x, dragStart.y, false);
      dragging = false;
      dragStart = null;
      if (event && event.preventDefault) event.preventDefault();
    });
    preview.addEventListener("keydown", function (event) {
      if (event.key === "Escape" && dragging) {
        if (dragStart) commit(dragStart.x, dragStart.y, false);
        dragging = false;
        dragStart = null;
        event.preventDefault();
        return;
      }
      var step = event.shiftKey ? 0.05 : 0.01;
      var x = numberValue(xInput, 0);
      var y = numberValue(yInput, 0);
      if (event.key === "ArrowLeft") x -= step;
      else if (event.key === "ArrowRight") x += step;
      else if (event.key === "ArrowUp") y -= step;
      else if (event.key === "ArrowDown") y += step;
      else if (event.key === "Home") {
        x = 0.5;
        y = 0.5;
      } else {
        return;
      }
      commit(x, y, true);
      event.preventDefault();
    });
    xInput.addEventListener("input", updateMarker);
    yInput.addEventListener("input", updateMarker);
    updateMarker();
  }

  function acceptTokens(input) {
    return String(input.getAttribute("accept") || "").split(",").map(function (token) {
      return token.trim().toLowerCase();
    }).filter(Boolean);
  }

  function fileAccepted(input, file) {
    var tokens = acceptTokens(input);
    if (!tokens.length) return true;
    var type = String(file.type || "").toLowerCase();
    var name = String(file.name || "").toLowerCase();
    return tokens.some(function (token) {
      if (token.charAt(0) === ".") return name.endsWith(token);
      if (token.slice(-2) === "/*") return type.indexOf(token.slice(0, -1)) === 0;
      return type === token;
    });
  }

  function maxSize(input) {
    var value = parseInt(input.getAttribute("data-media-upload-max-size") || "", 10);
    return Number.isFinite(value) && value > 0 ? value : 0;
  }

  function formatBytes(size) {
    if (!Number.isFinite(size)) return "";
    if (size < 1024) return size + " B";
    if (size < 1024 * 1024) return Math.round(size / 102.4) / 10 + " KB";
    return Math.round(size / 1024 / 102.4) / 10 + " MB";
  }

  function assignFile(input, file) {
    var transfer = new DataTransfer();
    transfer.items.add(file);
    input.files = transfer.files;
    dispatchNativeChange(input);
  }

  function trackUploadController(controller) {
    uploadControllers.push(controller);
  }

  function cleanupUploads(detachedOnly, keepAttached) {
    uploadControllers = uploadControllers.filter(function (controller) {
      if (!controller || typeof controller.cleanup !== "function") return false;
      var detached = !controller.root || !document.documentElement.contains(controller.root);
      if (!detachedOnly || detached) {
        controller.cleanup();
        return !!keepAttached && !detached;
      }
      return true;
    });
  }

  function upgradeUpload(root) {
    if (!root || root.dataset.mediaUploadBound === "true") return;
    var input = root.querySelector('input[type="file"]');
    var dropzone = root.querySelector("[data-media-upload-drop-zone], [data-media-upload-dropzone]");
    var status = root.querySelector("[data-media-upload-status]");
    var error = root.querySelector("[data-media-upload-error]");
    var preview = root.querySelector("[data-media-upload-preview]");
    var clear = root.querySelector("[data-media-upload-clear]");
    if (!input || !dropzone || !status) return;
    root.dataset.mediaUploadBound = "true";
    var previewURL = "";
    var controller = {
      root: root,
      cleanup: function () {
        revokePreview();
        if (preview) {
          preview.hidden = true;
          preview.removeAttribute("src");
          preview.removeAttribute("alt");
        }
      }
    };
    trackUploadController(controller);

    function setStatus(message, kind) {
      status.textContent = message || "";
      status.setAttribute("data-media-upload-status-kind", kind || "");
      if (error) {
        error.textContent = kind === "error" ? (message || "") : "";
        error.hidden = kind !== "error";
      }
    }

    function revokePreview() {
      if (previewURL) URL.revokeObjectURL(previewURL);
      previewURL = "";
    }

    function renderSelection(file) {
      revokePreview();
      if (!file) {
        setStatus("No file selected.", "");
        if (preview) {
          preview.hidden = true;
          preview.removeAttribute("src");
          preview.removeAttribute("alt");
        }
        if (clear) clear.hidden = true;
        return;
      }
      setStatus(file.name + " selected (" + formatBytes(file.size) + "). Use Upload file to save it.", "ok");
      if (preview) {
        if (String(file.type || "").indexOf("image/") === 0) {
          previewURL = URL.createObjectURL(file);
          preview.src = previewURL;
          preview.alt = "";
          preview.hidden = false;
        } else {
          preview.hidden = true;
          preview.removeAttribute("src");
          preview.removeAttribute("alt");
        }
      }
      if (clear) clear.hidden = false;
    }

    function validate(fileList) {
      var files = toArray(fileList);
      if (!files.length) return null;
      if (files.length > 1) return { error: "Choose one file at a time." };
      var file = files[0];
      if (!fileAccepted(input, file)) return { error: "That file type is not accepted here." };
      var limit = maxSize(input);
      if (limit && file.size > limit) return { error: "That file is larger than the allowed size." };
      return { file: file };
    }

    function choose(files) {
      var result = validate(files);
      if (!result) {
        renderSelection(input.files && input.files[0]);
        return;
      }
      if (result.error) {
        setStatus(result.error, "error");
        return;
      }
      assignFile(input, result.file);
      renderSelection(result.file);
    }

    dropzone.addEventListener("dragenter", function (event) {
      event.preventDefault();
      root.setAttribute("data-media-upload-dragging", "true");
      root.setAttribute("data-media-drop-active", "true");
    });
    dropzone.addEventListener("dragover", function (event) {
      event.preventDefault();
      if (event.dataTransfer) event.dataTransfer.dropEffect = "copy";
      root.setAttribute("data-media-upload-dragging", "true");
      root.setAttribute("data-media-drop-active", "true");
    });
    dropzone.addEventListener("dragleave", function (event) {
      if (!dropzone.contains(event.relatedTarget)) {
        root.removeAttribute("data-media-upload-dragging");
        root.removeAttribute("data-media-drop-active");
      }
    });
    dropzone.addEventListener("drop", function (event) {
      event.preventDefault();
      root.removeAttribute("data-media-upload-dragging");
      root.removeAttribute("data-media-drop-active");
      choose(event.dataTransfer ? event.dataTransfer.files : null);
    });
    dropzone.addEventListener("click", function () {
      input.click();
    });
    input.addEventListener("change", function () {
      var result = validate(input.files);
      if (result && result.error) {
        input.value = "";
        renderSelection(null);
        setStatus(result.error, "error");
        return;
      }
      renderSelection(input.files && input.files[0]);
    });
    if (clear) {
      clear.addEventListener("click", function () {
        input.value = "";
        renderSelection(null);
        input.focus();
      });
    }
    renderSelection(input.files && input.files[0]);
  }

  function init(root) {
    cleanupUploads(true, false);
    var scope = root || document;
    controls(scope, "input[list]").forEach(upgradeInput);
    controls(scope, "textarea[data-media-lines-list]").forEach(upgradeMediaLines);
    controls(scope, "[data-media-focal-preview]").forEach(upgradeFocalPreview);
    controls(scope, "[data-media-upload]").forEach(upgradeUpload);
  }

  window.GoSXStudioMediaRuntime = {
    init: init,
    mediaDragType: mediaDragType
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function () { init(document); });
  } else {
    init(document);
  }
  ["gosx:navigate", "gosx:render", "gosxstudio:content-editor-render"].forEach(function (name) {
    document.addEventListener(name, function (event) {
      cleanupUploads(name === "gosx:navigate" ? false : true, name === "gosx:navigate");
      init((event && event.target) || document);
    });
  });
  window.addEventListener("pagehide", function () { cleanupUploads(false, false); });
  new MutationObserver(function (mutations) {
    cleanupUploads(true, false);
    mutations.forEach(function (mutation) {
      toArray(mutation.addedNodes).forEach(function (node) {
        if (node && node.nodeType === 1) init(node);
      });
    });
  }).observe(document.documentElement, { childList: true, subtree: true });
})();
