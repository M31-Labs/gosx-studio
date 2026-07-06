(function () {
  "use strict";

  function ready(fn) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", fn, { once: true });
      return;
    }
    fn();
  }

  function normalize(value) {
    return String(value || "").toLowerCase().replace(/\s+/g, " ").trim();
  }

  function selectorValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  function fuzzyScore(haystack, query) {
    haystack = normalize(haystack);
    query = normalize(query);
    if (!query) return 0;
    var direct = haystack.indexOf(query);
    if (direct >= 0) return 1000 - direct;
    var words = haystack.split(" ");
    for (var wordIndex = 0; wordIndex < words.length; wordIndex += 1) {
      if (words[wordIndex].indexOf(query) === 0) return 850 - wordIndex;
    }
    var last = -1;
    var gaps = 0;
    for (var i = 0; i < query.length; i += 1) {
      var next = haystack.indexOf(query.charAt(i), last + 1);
      if (next < 0) return -1;
      if (last >= 0) gaps += next - last - 1;
      last = next;
    }
    return 500 - gaps;
  }

  function isEditableTarget(target) {
    if (!target || !target.closest) return false;
    var editable = target.closest("input, textarea, select, [contenteditable]");
    return !!(editable && editable.getAttribute("contenteditable") !== "false");
  }

  function shortcutParts(shortcut) {
    return normalize(shortcut).split(/[+\s]+/).filter(Boolean);
  }

  function shortcutHasModifier(shortcut) {
    return shortcutParts(shortcut).some(function (part) {
      return part === "ctrl" || part === "control" || part === "cmd" || part === "meta" || part === "command" || part === "mod" || part === "shift" || part === "alt" || part === "option";
    });
  }

  function shortcutMatches(shortcut, event) {
    var parts = shortcutParts(shortcut);
    if (!parts.length) return false;
    var key = String(event.key || "").toLowerCase();
    var wantsCtrl = false;
    var wantsMeta = false;
    var wantsShift = false;
    var wantsAlt = false;
    var wantsMod = false;
    var keyPart = "";
    parts.forEach(function (part) {
      if (part === "ctrl" || part === "control") wantsCtrl = true;
      else if (part === "cmd" || part === "meta" || part === "command") wantsMeta = true;
      else if (part === "mod") wantsMod = true;
      else if (part === "shift") wantsShift = true;
      else if (part === "alt" || part === "option") wantsAlt = true;
      else keyPart = part;
    });
    if (!keyPart || keyPart !== key) return false;
    if (wantsMod && !event.ctrlKey && !event.metaKey) return false;
    if (wantsCtrl && !event.ctrlKey && !event.metaKey) return false;
    if (wantsMeta && !event.metaKey) return false;
    if (wantsShift !== !!event.shiftKey) return false;
    if (wantsAlt !== !!event.altKey) return false;
    if (!wantsCtrl && !wantsMeta && !wantsMod && (event.ctrlKey || event.metaKey)) return false;
    return true;
  }

  function initPalette(node) {
    if (!node || node.dataset.gosxStudioCommandRuntime === "true") return;
    node.dataset.gosxStudioCommandRuntime = "true";
    var overlay = node.querySelector("[data-studio-command-overlay]");
    var search = node.querySelector("[data-studio-command-search]");
    var list = node.querySelector("[data-studio-command-list]");
    var empty = node.querySelector("[data-studio-command-empty]");
    if (!overlay || !list) return;
    var lastFocused = null;

    function buttons() {
      return Array.prototype.slice.call(node.querySelectorAll("[data-studio-command]"));
    }

    function visibleButtons() {
      return buttons().filter(function (button) { return !button.hidden; });
    }

    function selectedButton() {
      return visibleButtons().filter(function (button) {
        return button.getAttribute("aria-selected") === "true";
      })[0] || visibleButtons()[0] || null;
    }

    function buttonID(button) {
      if (!button) return "";
      if (!button.id) {
        button.id = "studio-command-" + String(button.getAttribute("data-studio-command") || "item").replace(/[^a-z0-9_-]/gi, "-").toLowerCase();
      }
      return button.id;
    }

    function setSelection(button) {
      buttons().forEach(function (candidate) {
        candidate.setAttribute("aria-selected", candidate === button ? "true" : "false");
      });
      if (!search) return;
      if (!button) {
        search.removeAttribute("aria-activedescendant");
        return;
      }
      search.setAttribute("aria-activedescendant", buttonID(button));
      if (button.scrollIntoView) button.scrollIntoView({ block: "nearest" });
    }

    function sortButtons(all, query) {
      all.sort(function (a, b) {
        var scoreA = Number(a.dataset.studioCommandScore || 0);
        var scoreB = Number(b.dataset.studioCommandScore || 0);
        var orderA = Number(a.dataset.studioCommandOrder || 0);
        var orderB = Number(b.dataset.studioCommandOrder || 0);
        if (query && scoreA !== scoreB) return scoreB - scoreA;
        return orderA - orderB;
      });
      all.forEach(function (button) { list.appendChild(button); });
    }

    function syncFilter() {
      var query = search ? normalize(search.value) : "";
      var visible = 0;
      var all = buttons();
      all.forEach(function (button, index) {
        if (!button.dataset.studioCommandOrder) button.dataset.studioCommandOrder = String(index);
        var haystack = button.getAttribute("data-studio-command-search-text") || button.textContent || "";
        var score = fuzzyScore(haystack, query);
        var match = !query || score >= 0;
        button.dataset.studioCommandScore = String(score);
        button.hidden = !match;
        if (match) visible += 1;
      });
      sortButtons(all, query);
      setSelection(visibleButtons()[0] || null);
      if (empty) empty.hidden = visible !== 0;
    }

    function focusableElements() {
      return Array.prototype.slice.call(overlay.querySelectorAll([
        "a[href]",
        "button:not([disabled])",
        "input:not([disabled])",
        "select:not([disabled])",
        "textarea:not([disabled])",
        "[tabindex]:not([tabindex='-1'])"
      ].join(","))).filter(function (element) {
        return !element.hidden && element.getClientRects().length > 0;
      });
    }

    function restoreFocus() {
      var target = lastFocused;
      lastFocused = null;
      if (!target || !target.focus || !document.contains(target)) return;
      window.setTimeout(function () {
        target.focus({ preventScroll: true });
      }, 0);
    }

    function trapFocus(event) {
      if (!isOpen()) return false;
      var focusable = focusableElements();
      if (!focusable.length) return false;
      var first = focusable[0];
      var last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus({ preventScroll: true });
        return true;
      }
      if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus({ preventScroll: true });
        return true;
      }
      return false;
    }

    function open() {
      lastFocused = document.activeElement;
      node.setAttribute("data-studio-command-state", "open");
      overlay.hidden = false;
      node.querySelectorAll("[data-studio-command-open]").forEach(function (button) {
        button.setAttribute("aria-expanded", "true");
      });
      if (search) {
        search.value = "";
        search.setAttribute("aria-expanded", "true");
      }
      syncFilter();
      window.setTimeout(function () {
        if (search && search.focus) search.focus({ preventScroll: true });
      }, 0);
    }

    function close(options) {
      options = options || {};
      node.setAttribute("data-studio-command-state", "closed");
      overlay.hidden = true;
      node.querySelectorAll("[data-studio-command-open]").forEach(function (button) {
        button.setAttribute("aria-expanded", "false");
      });
      if (search) {
        search.setAttribute("aria-expanded", "false");
        search.removeAttribute("aria-activedescendant");
      }
      if (options.restoreFocus !== false) restoreFocus();
    }

    function isOpen() {
      return node.getAttribute("data-studio-command-state") === "open";
    }

    function submitTo(action, label) {
      var form = node.closest("form");
      if (!form) return;
      if (action) form.dataset.gosxStudioPendingAction = action;
      if (label) form.dataset.gosxStudioPendingActionLabel = label;
      if (form.requestSubmit) form.requestSubmit();
      else if (typeof Event === "function") {
        var event = new Event("submit", { bubbles: true, cancelable: true });
        if (form.dispatchEvent(event)) form.submit();
      }
      else form.submit();
    }

    function run(button) {
      if (!button) return;
      var detail = {
        key: button.getAttribute("data-studio-command") || "",
        kind: button.getAttribute("data-studio-command-kind") || "",
        target: button.getAttribute("data-studio-command-target") || "",
        href: button.getAttribute("data-studio-command-href") || ""
      };
      var commandEvent = new CustomEvent("gosxstudio:command", { bubbles: true, cancelable: true, detail: detail });
      node.dispatchEvent(commandEvent);
      if (commandEvent.defaultPrevented) {
        close();
        return;
      }
      if (detail.kind === "submit") {
        close({ restoreFocus: false });
        submitTo(detail.href, button.textContent);
        return;
      }
      if (detail.kind === "insert" && detail.target) {
        var form = node.closest("form") || document;
        var add = form.querySelector('[data-editor-add-block="' + selectorValue(detail.target) + '"]');
        if (add) {
          close();
          add.click();
          return;
        }
      }
      if (detail.href) {
        close({ restoreFocus: false });
        window.location.href = detail.href;
        return;
      }
      close();
    }

    function shortcutCommand(event) {
      var match = null;
      buttons().some(function (button) {
        var shortcut = button.getAttribute("data-studio-command-shortcut") || "";
        if (!shortcut || !shortcutMatches(shortcut, event)) return false;
        if (isEditableTarget(event.target) && !shortcutHasModifier(shortcut)) return false;
        match = button;
        return true;
      });
      return match;
    }

    function move(delta) {
      var visible = visibleButtons();
      if (!visible.length) {
        setSelection(null);
        return;
      }
      var current = selectedButton();
      var index = visible.indexOf(current);
      if (index < 0) index = delta > 0 ? -1 : 0;
      setSelection(visible[(index + delta + visible.length) % visible.length]);
    }

    node.addEventListener("input", function (event) {
      if (event.target && event.target.matches("[data-studio-command-search]")) syncFilter();
    });
    node.addEventListener("click", function (event) {
      var opener = event.target.closest("[data-studio-command-open]");
      if (opener && node.contains(opener)) {
        event.preventDefault();
        open();
        return;
      }
      var closer = event.target.closest("[data-studio-command-close], [data-studio-command-overlay]");
      if (closer && (closer.matches("[data-studio-command-close]") || closer === event.target)) {
        event.preventDefault();
        close();
        return;
      }
      var command = event.target.closest("[data-studio-command]");
      if (command && node.contains(command)) {
        event.preventDefault();
        setSelection(command);
        run(command);
      }
    });
    node.addEventListener("pointermove", function (event) {
      var command = event.target.closest("[data-studio-command]");
      if (command && node.contains(command) && !command.hidden) setSelection(command);
    });
    node.addEventListener("keydown", function (event) {
      if (event.key === "Escape") {
        event.preventDefault();
        close();
      } else if (event.key === "Tab") {
        trapFocus(event);
      } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        move(event.key === "ArrowDown" ? 1 : -1);
      } else if (event.key === "Home" || event.key === "End") {
        event.preventDefault();
        var visible = visibleButtons();
        setSelection(event.key === "Home" ? visible[0] : visible.slice(-1)[0]);
      } else if (event.key === "Enter" && isOpen()) {
        event.preventDefault();
        run(selectedButton());
      }
    });
    document.addEventListener("keydown", function (event) {
      if (!document.contains(node)) return;
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        if (isOpen()) close();
        else open();
        return;
      }
      var command = shortcutCommand(event);
      if (command) {
        event.preventDefault();
        run(command);
      }
    });
    syncFilter();
  }

  ready(function () {
    document.querySelectorAll("[data-studio-command-palette]").forEach(initPalette);
  });
})();
