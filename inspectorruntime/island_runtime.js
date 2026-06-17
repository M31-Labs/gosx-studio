// island_runtime.js — InspectorRuntime island for GoSX Studio.
//
// This script owns dirty tracking (widget value != initial value),
// enabling/disabling the Save button based on dirty+valid state,
// required-field validation (non-empty), and Cancel → revert to initial value.
//
// It does NOT implement form submission. The managed-form path in
// authoringruntime (data-gosx-studio-authoring-managed) handles all
// fetch+JSON submission. This island is purely a UX-layer concern.
//
// Public API (IIFE, window-global style):
//   window.GoSXStudioInspectorRuntime = { bind, bindAll }
//
//   bind(root)    — scan root for one [data-gosx-studio-inspector-control]
//                   element and attach dirty/cancel/validation listeners.
//   bindAll(root) — scan root for ALL [data-gosx-studio-inspector-control]
//                   elements and bind each.
//
// Attribute contracts expected in the DOM:
//   [data-gosx-studio-inspector-control]  — wrapper element for an inspector field.
//   [data-gosx-studio-inspector-kind]     — the control kind string on the wrapper.
//   [data-gosx-studio-inspector-cancel]   — Cancel button (optional; type=reset also accepted).

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  var CONTROL_ATTR = "data-gosx-studio-inspector-control";
  var KIND_ATTR = "data-gosx-studio-inspector-kind";
  var CANCEL_ATTR = "data-gosx-studio-inspector-cancel";
  var DIRTY_ATTR = "data-gosx-studio-inspector-dirty";

  // Return the primary interactive widget inside a control wrapper.
  // Looks for the first input, textarea, or select that is not hidden.
  function findWidget(wrapper) {
    var els = wrapper.querySelectorAll("input:not([type='hidden']), textarea, select");
    for (var i = 0; i < els.length; i++) {
      var el = els[i];
      if (el.type !== "hidden") return el;
    }
    return null;
  }

  // Return submit button inside a control wrapper.
  function findSave(wrapper) {
    return wrapper.querySelector("button[type='submit'], input[type='submit']");
  }

  // Return cancel button inside a control wrapper.
  function findCancel(wrapper) {
    var byAttr = wrapper.querySelector("[" + CANCEL_ATTR + "]");
    if (byAttr) return byAttr;
    return wrapper.querySelector("button[type='reset'], input[type='reset']");
  }

  function isRequired(wrapper) {
    var widget = findWidget(wrapper);
    if (!widget) return false;
    return widget.required || widget.getAttribute("required") !== null;
  }

  function getWidgetValue(widget) {
    if (!widget) return "";
    if (widget.type === "checkbox") return widget.checked ? "true" : "false";
    return widget.value || "";
  }

  function setDirty(wrapper, dirty, save) {
    if (dirty) {
      wrapper.setAttribute(DIRTY_ATTR, "true");
    } else {
      wrapper.removeAttribute(DIRTY_ATTR);
    }
    if (save) {
      save.disabled = !dirty;
    }
  }

  function isValid(widget) {
    if (!widget) return true;
    // Required non-empty check
    if ((widget.required || widget.getAttribute("required") !== null) && getWidgetValue(widget).trim() === "") {
      return false;
    }
    return true;
  }

  // Bind a single [data-gosx-studio-inspector-control] wrapper.
  function bind(wrapper) {
    if (!wrapper) return;
    var widget = findWidget(wrapper);
    var save = findSave(wrapper);
    var cancel = findCancel(wrapper);

    if (!widget) return;

    // Capture initial value so Cancel can restore it.
    var initialValue = getWidgetValue(widget);
    var initialChecked = widget.type === "checkbox" ? widget.checked : false;

    // Start with Save disabled (not dirty yet).
    if (save) {
      save.disabled = true;
    }

    function update() {
      var current = getWidgetValue(widget);
      var dirty = current !== initialValue;
      var valid = isValid(widget);
      wrapper.setAttribute(DIRTY_ATTR, dirty ? "true" : "false");
      if (save) {
        save.disabled = !(dirty && valid);
      }
    }

    widget.addEventListener("input", update);
    widget.addEventListener("change", update);

    // Cancel: revert to initial value and reset dirty state.
    if (cancel) {
      cancel.addEventListener("click", function (e) {
        if (cancel.type === "reset") {
          // Let the native reset fire, then re-sync.
          setTimeout(function () {
            initialValue = getWidgetValue(widget);
            setDirty(wrapper, false, save);
          }, 0);
        } else {
          e.preventDefault();
          if (widget.type === "checkbox") {
            widget.checked = initialChecked;
          } else {
            widget.value = initialValue;
          }
          setDirty(wrapper, false, save);
          // Dispatch change so any mirrors can react.
          widget.dispatchEvent(new Event("change", { bubbles: true }));
        }
      });
    }
  }

  // Bind ALL inspector controls found under root.
  function bindAll(root) {
    if (!root) root = doc;
    var wrappers = root.querySelectorAll
      ? root.querySelectorAll("[" + CONTROL_ATTR + "]")
      : [];
    for (var i = 0; i < wrappers.length; i++) {
      bind(wrappers[i]);
    }
  }

  window.GoSXStudioInspectorRuntime = {
    bind: bind,
    bindAll: bindAll
  };

  // Auto-mount on document ready. Idempotent via marker attribute.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-inspector-runtime-auto-mounted";
      if (doc.documentElement && doc.documentElement.getAttribute(marker) === "true") return;
      if (doc.documentElement) doc.documentElement.setAttribute(marker, "true");
      window.GoSXStudioInspectorRuntime.bindAll(doc.body || doc);
    } catch (e) {
      // Auto-mount must never break the page.
    }
  }

  if (doc.readyState === "loading") {
    doc.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
