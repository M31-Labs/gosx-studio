(function () {
  "use strict";

  var states = {
    saved: "Saved",
    dirty: "Unsaved",
    saving: "Saving",
    autosaving: "Autosaving",
    error: "Save failed"
  };

  function ready(fn) {
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", fn, { once: true });
      return;
    }
    fn();
  }

  function frameTask(fn) {
    var pending = false;
    return function () {
      if (pending) return;
      pending = true;
      window.requestAnimationFrame(function () {
        pending = false;
        fn();
      });
    };
  }

  function skipField(field) {
    if (!field || !field.name || field.disabled) return true;
    if (field.name === "csrf_token") return true;
    if (field.matches && field.matches("[data-gosx-studio-state-ignore]")) return true;
    var type = String(field.type || "").toLowerCase();
    return type === "button" || type === "submit" || type === "reset" || type === "file";
  }

  function fieldValues(field) {
    var type = String(field.type || "").toLowerCase();
    if (type === "checkbox" || type === "radio") {
      return [field.checked ? (field.value || "on") : ""];
    }
    if (field.tagName === "SELECT" && field.multiple) {
      return Array.prototype.slice.call(field.options)
        .filter(function (option) { return option.selected; })
        .map(function (option) { return option.value; });
    }
    return [field.value || ""];
  }

  function fieldSnapshot(field) {
    var type = String(field.type || "").toLowerCase();
    var tag = String(field.tagName || "").toUpperCase();
    var snapshot = {
      name: field.name || "",
      type: type,
      tag: tag,
      value: field.value || "",
      checked: !!field.checked,
      values: []
    };
    if (tag === "SELECT" && field.multiple) {
      snapshot.values = Array.prototype.slice.call(field.options)
        .filter(function (option) { return option.selected; })
        .map(function (option) { return option.value; });
    }
    return snapshot;
  }

  function formState(form) {
    var fields = [];
    Array.prototype.forEach.call(form.elements, function (field) {
      if (skipField(field)) return;
      fields.push(fieldSnapshot(field));
    });
    return {
      signature: formSignature(form),
      fields: fields
    };
  }

  function restoreField(field, snapshot) {
    if (!snapshot) return;
    var type = String(field.type || "").toLowerCase();
    if (type === "checkbox" || type === "radio") {
      field.checked = !!snapshot.checked;
    } else if (field.tagName === "SELECT" && field.multiple) {
      var values = snapshot.values || [];
      Array.prototype.forEach.call(field.options, function (option) {
        option.selected = values.indexOf(option.value) >= 0;
      });
    } else {
      field.value = snapshot.value || "";
    }
    field.dispatchEvent(new Event("input", { bubbles: true }));
    field.dispatchEvent(new Event("change", { bubbles: true }));
  }

  function restoreFormState(form, state) {
    var buckets = {};
    (state.fields || []).forEach(function (field) {
      if (!buckets[field.name]) buckets[field.name] = [];
      buckets[field.name].push(field);
    });
    Array.prototype.forEach.call(form.elements, function (field) {
      if (skipField(field)) return;
      restoreField(field, (buckets[field.name] || []).shift());
    });
  }

  function formSignature(form) {
    var parts = [];
    Array.prototype.forEach.call(form.elements, function (field) {
      if (skipField(field)) return;
      fieldValues(field).forEach(function (value) {
        parts.push(encodeURIComponent(field.name) + "=" + encodeURIComponent(value));
      });
    });
    return parts.join("&");
  }

  function dirtyCount(savedSignature, currentSignature) {
    if (savedSignature === currentSignature) return 0;
    var saved = savedSignature ? savedSignature.split("&") : [];
    var current = currentSignature ? currentSignature.split("&") : [];
    var total = Math.max(saved.length, current.length);
    var count = 0;
    for (var index = 0; index < total; index += 1) {
      if (saved[index] !== current[index]) count += 1;
    }
    return Math.max(count, 1);
  }

  function statusNodes(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-save-state], [data-editor-save-state]"));
  }

  function detailNodes(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-save-detail]"));
  }

  function lastSavedNodes(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-last-saved]"));
  }

  function dirtyCountNodes(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-dirty-count]"));
  }

  function historyUndoButtons(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-history-undo]"));
  }

  function historyRedoButtons(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-history-redo]"));
  }

  function historyStatusNodes(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-history-status]"));
  }

  function saveButtons(form) {
    return Array.prototype.slice.call(form.querySelectorAll("[data-gosx-studio-save-button], [data-editor-save-button]"));
  }

  function baseClass(node) {
    if (!node.dataset.gosxStudioSaveStateBaseClass) {
      var keep = Array.prototype.slice.call(node.classList).filter(function (className) {
        return className.indexOf("gosx-studio-save-state--") !== 0 && className.indexOf("editor-save-state--") !== 0;
      });
      node.dataset.gosxStudioSaveStateBaseClass = keep.join(" ");
    }
    return node.dataset.gosxStudioSaveStateBaseClass;
  }

  function setStatusClass(node, state) {
    var classes = baseClass(node);
    if (classes) classes += " ";
    classes += "gosx-studio-save-state--" + state;
    if (node.classList.contains("editor-save-state") || classes.indexOf("editor-save-state") >= 0) {
      classes += " editor-save-state--" + state;
    }
    node.className = classes;
  }

  function dispatchState(form, state, dirty, reason, options) {
    options = options || {};
    form.dispatchEvent(new CustomEvent("gosxstudio:save-state", {
      bubbles: true,
      detail: {
        state: state,
        dirty: dirty,
        reason: reason || "",
        dirtyCount: options.dirtyCount || 0,
        savedAt: options.savedAt || ""
      }
    }));
  }

  function timeLabel(date) {
    try {
      return date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
    } catch (error) {
      return "";
    }
  }

  function saveDetail(state, options) {
    options = options || {};
    if (state === "dirty") {
      return options.dirtyCount === 1 ? "1 change waiting" : String(options.dirtyCount || 0) + " changes waiting";
    }
    if (state === "autosaving") return "Syncing changes";
    if (state === "saving" && options.actionLabel) return options.actionLabel + " in progress";
    if (state === "saving") return "Sending checkpoint";
    if (state === "error" && options.actionLabel) return options.actionLabel + " failed";
    if (state === "error") return "Autosave could not reach the server";
    if (state === "saved" && options.actionLabel) return options.actionLabel + " complete";
    if (options.savedAtLabel) return "Saved at " + options.savedAtLabel;
    return "Ready";
  }

  function updateLastSaved(form, savedAt) {
    lastSavedNodes(form).forEach(function (node) {
      if (!savedAt) {
        node.textContent = node.getAttribute("data-gosx-studio-last-saved-empty") || "Not saved this session";
        node.removeAttribute("datetime");
        return;
      }
      var date = new Date(savedAt);
      var label = timeLabel(date);
      node.textContent = label ? "Last saved " + label : "Last saved";
      if (node.tagName === "TIME") node.setAttribute("datetime", savedAt);
    });
  }

  function updateDirtyCount(form, count) {
    form.setAttribute("data-gosx-studio-dirty-count", String(count || 0));
    dirtyCountNodes(form).forEach(function (node) {
      node.hidden = !count;
      node.textContent = count === 1 ? "1 change" : String(count || 0) + " changes";
    });
  }

  function setState(form, state, reason, options) {
    options = options || {};
    var dirty = state === "dirty" || state === "autosaving" || state === "error";
    var count = options.dirtyCount || 0;
    form.setAttribute("data-gosx-studio-save-state", state);
    form.setAttribute("data-studio-dirty-state", dirty ? "dirty" : "clean");
    statusNodes(form).forEach(function (node) {
      node.textContent = states[state] || states.saved;
      node.setAttribute("data-gosx-studio-save-state-value", state);
      setStatusClass(node, state);
    });
    detailNodes(form).forEach(function (node) {
      node.textContent = saveDetail(state, options);
    });
    updateDirtyCount(form, count);
    updateLastSaved(form, options.savedAt || form.getAttribute("data-gosx-studio-last-saved-at") || "");
    saveButtons(form).forEach(function (button) {
      button.classList.toggle("button--attention", state === "dirty" || state === "error");
      button.setAttribute("data-gosx-studio-save-button-state", state);
      button.setAttribute("aria-busy", state === "saving" || state === "autosaving" ? "true" : "false");
    });
    dispatchState(form, state, dirty, reason, options);
  }

  function autosaveEnabled(form) {
    return form.getAttribute("data-gosx-studio-autosave") === "true";
  }

  function autosaveDelay(form) {
    var delay = Number(form.getAttribute("data-gosx-studio-autosave-delay") || 1400);
    if (!Number.isFinite(delay) || delay < 250) return 1400;
    return delay;
  }

  function autosaveURL(form) {
    return form.getAttribute("data-gosx-studio-autosave-url") || form.getAttribute("action") || window.location.href;
  }

  function autosaveMethod(form) {
    return String(form.getAttribute("method") || "post").toUpperCase();
  }

  function clientActionsEnabled(form) {
    return form.getAttribute("data-gosx-studio-client") === "true";
  }

  function submitterField(submitter) {
    if (!submitter || !submitter.name || submitter.disabled) return null;
    var type = String(submitter.type || "").toLowerCase();
    if (type === "button" || type === "reset") return null;
    return {
      name: submitter.name,
      value: submitter.value || ""
    };
  }

  function actionFormData(form, submitter) {
    try {
      if (submitter) return new FormData(form, submitter);
    } catch (error) {
      // Older browsers ignore the submitter argument; append it below.
    }
    var data = new FormData(form);
    var field = submitterField(submitter);
    if (field) data.append(field.name, field.value);
    return data;
  }

  function submitActionURL(form, submitter, pendingAction) {
    if (pendingAction) return pendingAction;
    if (submitter) {
      var attr = submitter.getAttribute("formaction");
      if (attr) return attr;
      if (submitter.formAction) return submitter.formAction;
    }
    return form.getAttribute("action") || window.location.href;
  }

  function submitMethod(form, submitter) {
    if (submitter) {
      var attr = submitter.getAttribute("formmethod");
      if (attr) return String(attr).toUpperCase();
      if (submitter.formMethod) return String(submitter.formMethod).toUpperCase();
    }
    return String(form.getAttribute("method") || "post").toUpperCase();
  }

  function methodHasBody(method) {
    return method !== "GET" && method !== "HEAD";
  }

  function actionURLWithData(action, data) {
    var url = new URL(action, window.location.href);
    data.forEach(function (value, key) {
      url.searchParams.append(key, value);
    });
    return url.toString();
  }

  function submitterDetail(submitter) {
    if (!submitter) return null;
    return {
      name: submitter.getAttribute("name") || "",
      value: submitter.getAttribute("value") || "",
      action: submitter.getAttribute("formaction") || "",
      method: submitter.getAttribute("formmethod") || ""
    };
  }

  function compactText(text) {
    return String(text || "").replace(/\s+/g, " ").trim();
  }

  function actionLabel(submitter, pendingLabel) {
    if (pendingLabel) return pendingLabel;
    if (!submitter) return "Action";
    return compactText(submitter.getAttribute("data-gosx-studio-action-label") || submitter.getAttribute("aria-label") || submitter.textContent) || "Action";
  }

  function dispatchActionResult(form, detail) {
    form.dispatchEvent(new CustomEvent("gosxstudio:action-result", {
      bubbles: true,
      detail: detail
    }));
  }

  function initForm(form) {
    if (!form || form.dataset.gosxStudioStateBound === "true") return;
    form.dataset.gosxStudioStateBound = "true";
    var saved = formSignature(form);
    var submitting = false;
    var autosaving = false;
    var autosaveTimer = 0;
    var lastSavedAt = "";
    var applyingHistory = false;
    var historyTimer = 0;
    var history = [];
    var historyIndex = -1;
    var historyLimit = Number(form.getAttribute("data-gosx-studio-history-limit") || 100);
    if (!Number.isFinite(historyLimit) || historyLimit < 2) historyLimit = 100;

    function isDirty() {
      return formSignature(form) !== saved;
    }

    function stateOptions(extra) {
      var current = formSignature(form);
      var options = {
        dirtyCount: dirtyCount(saved, current),
        savedAt: lastSavedAt,
        savedAtLabel: lastSavedAt ? timeLabel(new Date(lastSavedAt)) : ""
      };
      if (extra) {
        Object.keys(extra).forEach(function (key) {
          options[key] = extra[key];
        });
      }
      return options;
    }

    function update(reason) {
      if (submitting || autosaving) return;
      setState(form, isDirty() ? "dirty" : "saved", reason, stateOptions());
    }

    var updateFrame = frameTask(function () {
      update("field");
    });

    function canUndo() {
      return historyIndex > 0;
    }

    function canRedo() {
      return historyIndex >= 0 && historyIndex < history.length - 1;
    }

    function historyStatusLabel() {
      var edits = Math.max(0, historyIndex);
      if (!edits) return "No local edits";
      return edits === 1 ? "1 local edit" : String(edits) + " local edits";
    }

    function dispatchHistoryState(reason) {
      var detail = {
        index: historyIndex,
        length: history.length,
        canUndo: canUndo(),
        canRedo: canRedo(),
        reason: reason || ""
      };
      form.setAttribute("data-gosx-studio-history-index", String(historyIndex));
      form.setAttribute("data-gosx-studio-history-length", String(history.length));
      form.setAttribute("data-gosx-studio-can-undo", detail.canUndo ? "true" : "false");
      form.setAttribute("data-gosx-studio-can-redo", detail.canRedo ? "true" : "false");
      historyUndoButtons(form).forEach(function (button) {
        button.disabled = !detail.canUndo;
        button.setAttribute("aria-disabled", detail.canUndo ? "false" : "true");
      });
      historyRedoButtons(form).forEach(function (button) {
        button.disabled = !detail.canRedo;
        button.setAttribute("aria-disabled", detail.canRedo ? "false" : "true");
      });
      historyStatusNodes(form).forEach(function (node) {
        node.textContent = historyStatusLabel();
      });
      form.dispatchEvent(new CustomEvent("gosxstudio:history-state", {
        bubbles: true,
        detail: detail
      }));
    }

    function pushHistory(reason) {
      if (applyingHistory) return;
      var state = formState(form);
      if (historyIndex >= 0 && history[historyIndex] && history[historyIndex].signature === state.signature) {
        dispatchHistoryState(reason);
        return;
      }
      if (historyIndex < history.length - 1) history = history.slice(0, historyIndex + 1);
      history.push(state);
      if (history.length > historyLimit) history.shift();
      historyIndex = history.length - 1;
      dispatchHistoryState(reason);
    }

    function scheduleHistory(reason) {
      if (applyingHistory) return;
      window.clearTimeout(historyTimer);
      historyTimer = window.setTimeout(function () {
        pushHistory(reason || "field");
      }, 350);
    }

    function flushHistory(reason) {
      window.clearTimeout(historyTimer);
      pushHistory(reason || "flush");
    }

    function restoreHistory(index, reason) {
      if (index < 0 || index >= history.length || index === historyIndex) return;
      applyingHistory = true;
      historyIndex = index;
      restoreFormState(form, history[historyIndex]);
      applyingHistory = false;
      dispatchHistoryState(reason);
      update(reason || "history");
      scheduleAutosave();
      form.dispatchEvent(new CustomEvent("gosxstudio:history-restore", {
        bubbles: true,
        detail: {
          index: historyIndex,
          length: history.length,
          reason: reason || "history"
        }
      }));
    }

    function runHistory(action, reason) {
      if (action === "undo") {
        flushHistory("history-before-undo");
        if (!canUndo()) return false;
        restoreHistory(historyIndex - 1, reason || "undo");
        return true;
      }
      if (action === "redo") {
        if (!canRedo()) return false;
        restoreHistory(historyIndex + 1, reason || "redo");
        return true;
      }
      return false;
    }

    function scheduleAutosave() {
      if (!autosaveEnabled(form)) return;
      window.clearTimeout(autosaveTimer);
      autosaveTimer = window.setTimeout(runAutosave, autosaveDelay(form));
    }

    function runAutosave() {
      if (!autosaveEnabled(form) || submitting || autosaving || !isDirty()) return;
      var signature = formSignature(form);
      var target = autosaveURL(form);
      if (!target) return;
      autosaving = true;
      setState(form, "autosaving", "autosave", stateOptions());
      window.fetch(target, {
        method: autosaveMethod(form),
        body: new FormData(form),
        credentials: "same-origin",
        headers: {
          "X-GoSX-Studio-Autosave": "true"
        }
      }).then(function (response) {
        if (!response.ok) throw new Error("Autosave failed");
        saved = signature;
        lastSavedAt = new Date().toISOString();
        form.setAttribute("data-gosx-studio-last-saved-at", lastSavedAt);
        autosaving = false;
        if (isDirty()) {
          setState(form, "dirty", "autosave-stale", stateOptions());
          scheduleAutosave();
          return;
        }
        setState(form, "saved", "autosave", stateOptions({ dirtyCount: 0 }));
      }, function () {
        autosaving = false;
        setState(form, "error", "autosave", stateOptions());
      });
    }

    form.addEventListener("input", function () {
      scheduleHistory("field");
      updateFrame();
      scheduleAutosave();
    });
    form.addEventListener("change", function () {
      scheduleHistory("field");
      updateFrame();
      scheduleAutosave();
    });
    form.addEventListener("click", function (event) {
      if (event.target && event.target.closest && event.target.closest("[data-block-studio-move], [data-gosx-studio-state-dirty-action]")) {
        scheduleHistory("action");
        updateFrame();
        scheduleAutosave();
      }
    });
    form.addEventListener("drop", function () {
      scheduleHistory("drop");
      updateFrame();
      scheduleAutosave();
    });
    form.addEventListener("dragend", function () {
      scheduleHistory("drag");
      updateFrame();
      scheduleAutosave();
    });
    form.addEventListener("gosxstudio:editor-transaction", function (event) {
      var detail = event.detail || {};
      scheduleHistory(detail.kind || detail.reason || "transaction");
      updateFrame();
      scheduleAutosave();
    });
    form.addEventListener("gosxstudio:editor-operation", function (event) {
      var detail = event.detail || {};
      if (detail.mutation === false) return;
      scheduleHistory(detail.kind || detail.reason || "operation");
      updateFrame();
      scheduleAutosave();
    });
    form.addEventListener("click", function (event) {
      var undo = event.target && event.target.closest && event.target.closest("[data-gosx-studio-history-undo]");
      var redo = event.target && event.target.closest && event.target.closest("[data-gosx-studio-history-redo]");
      if (undo && form.contains(undo)) {
        event.preventDefault();
        runHistory("undo", "button");
      } else if (redo && form.contains(redo)) {
        event.preventDefault();
        runHistory("redo", "button");
      }
    });
    form.addEventListener("gosxstudio:command", function (event) {
      var detail = event.detail || {};
      if (detail.kind !== "history") return;
      if (runHistory(detail.target || detail.key, "command")) event.preventDefault();
    });
    function runClientAction(pendingAction, submitter, pendingLabel) {
      if (submitting) return;
      var action = submitActionURL(form, submitter, pendingAction);
      var method = submitMethod(form, submitter);
      var label = actionLabel(submitter, pendingLabel);
      var data = actionFormData(form, submitter);
      var requestURL = methodHasBody(method) ? action : actionURLWithData(action, data);
      var request = {
        method: method,
        credentials: "same-origin",
        headers: {
          "X-GoSX-Studio-Action": "true",
          "X-GoSX-Studio-Client-Action": "true"
        }
      };
      if (methodHasBody(method)) request.body = data;
      var signature = formSignature(form);
      submitting = true;
      window.clearTimeout(autosaveTimer);
      setState(form, "saving", "action", stateOptions({ actionLabel: label }));
      window.fetch(requestURL, request).then(function (response) {
        if (!response.ok) throw new Error("Studio action failed");
        saved = signature;
        lastSavedAt = new Date().toISOString();
        form.setAttribute("data-gosx-studio-last-saved-at", lastSavedAt);
        submitting = false;
        setState(form, "saved", "action", stateOptions({ dirtyCount: 0, actionLabel: label }));
        dispatchActionResult(form, {
          ok: true,
          action: action,
          label: label,
          method: method,
          status: response.status,
          redirected: response.redirected,
          url: response.url || "",
          submitter: submitterDetail(submitter)
        });
      }, function (error) {
        submitting = false;
        setState(form, "error", "action", stateOptions({ actionLabel: label }));
        dispatchActionResult(form, {
          ok: false,
          action: action,
          label: label,
          method: method,
          status: 0,
          redirected: false,
          url: "",
          error: error && error.message ? error.message : "Studio action failed",
          submitter: submitterDetail(submitter)
        });
      });
    }

    form.addEventListener("submit", function (event) {
      var pendingAction = form.dataset.gosxStudioPendingAction || "";
      var pendingLabel = form.dataset.gosxStudioPendingActionLabel || "";
      if (pendingAction) delete form.dataset.gosxStudioPendingAction;
      if (pendingLabel) delete form.dataset.gosxStudioPendingActionLabel;
      if (clientActionsEnabled(form)) {
        event.preventDefault();
        runClientAction(pendingAction, event.submitter || null, pendingLabel);
        return;
      }
      submitting = true;
      window.clearTimeout(autosaveTimer);
      setState(form, "saving", "submit", stateOptions());
    });

    document.addEventListener("keydown", function (event) {
      if (event.defaultPrevented || !document.contains(form)) return;
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "z") {
        event.preventDefault();
        runHistory(event.shiftKey ? "redo" : "undo", "keyboard");
        return;
      }
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "y") {
        event.preventDefault();
        runHistory("redo", "keyboard");
        return;
      }
      if (!(event.metaKey || event.ctrlKey) || event.key.toLowerCase() !== "s") return;
      event.preventDefault();
      if (form.requestSubmit) form.requestSubmit();
      else if (clientActionsEnabled(form)) runClientAction("", null, "");
      else form.submit();
    });

    document.addEventListener("click", function (event) {
      var link = event.target && event.target.closest && event.target.closest("a[href]");
      if (!link || !document.contains(form) || !isDirty() || submitting) return;
      if (link.target && link.target !== "_self") return;
      var next = new URL(link.href, window.location.href);
      if (next.origin === window.location.origin && next.pathname === window.location.pathname && next.search === window.location.search) return;
      if (!window.confirm("Discard unsaved editor changes?")) event.preventDefault();
    }, true);

    window.addEventListener("beforeunload", function (event) {
      if (!document.contains(form) || !isDirty() || submitting) return;
      event.preventDefault();
      event.returnValue = "";
    });

    pushHistory("init");
    update("init");
  }

  function initAll(root) {
    var scope = root && root.querySelectorAll ? root : document;
    Array.prototype.forEach.call(scope.querySelectorAll("[data-gosx-studio-state]"), initForm);
  }

  ready(function () { initAll(document); });
  document.addEventListener("gosx:navigate", function () { initAll(document); });
  document.addEventListener("gosx:render", function () { initAll(document); });
})();
