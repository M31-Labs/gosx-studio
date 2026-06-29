// island_runtime.js — companion JS for the fieldruntime .gsx islands.
//
// The .gsx islands (field_bind.gsx, field_mirror.gsx, field_clipboard.gsx)
// are mount-point markers in the editor DOM. This script publishes the
// window.__gosx_field_runtime_island_* globals that fieldruntime.BridgeShim
// delegates to when the "field-runtime-islands" feature flag is on. It is
// emitted into the studio runtime bundle by fieldruntime.IslandRuntimeJS()
// (see runtime.go) and runs before BridgeShim() at bundle init time.
//
// Each function below replaces a method on window.GoSXStudioFieldRuntime
// while preserving exact observable behavior:
//
//   bindMirroring(root)
//     Scans root for [data-editor-source], [data-studio-field-source],
//     [data-editor-frame-attr-target];
//     attaches an input listener that, on every keystroke (frame-throttled
//     via rAF), writes the value to the shared signal
//     $preview.field.<sourceKey>. Slice 6's preview subscriber receives the
//     signal via the cross-frame relay (per ADR 0009) and applies the
//     textContent / attribute update inside the iframe. Slice 6's
//     transitional cleanup (Section G.1 of the slice-6 plan) removed the
//     legacy GoSXStudioPreviewRuntime.applyTextUpdate fallback — the
//     signal write is the sole delivery mechanism now.
//
//   bindClipboard(root)
//     Idempotently installs the document-level click handler for
//     [data-studio-copy-target]; copies via navigator.clipboard.writeText
//     with an execCommand fallback; manages the 1600ms button-state
//     animation. Mirrors the legacy bindFieldClipboard behavior exactly.
//
//   bind(root)
//     bindMirroring(root) + bindClipboard(root). Matches legacy
//     bindFieldRuntime(root).

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  function frame(callback) {
    if (window.requestAnimationFrame) {
      window.requestAnimationFrame(callback);
      return;
    }
    window.setTimeout(callback, 16);
  }

  function frameTask(callback) {
    var queued = false;
    var lastArgs = null;
    var lastThis = null;
    return function () {
      lastArgs = arguments;
      lastThis = this;
      if (queued) return;
      queued = true;
      frame(function () {
        queued = false;
        callback.apply(lastThis, lastArgs || []);
      });
    };
  }

  function resolveScope(root) {
    if (root && root.querySelectorAll) return root;
    return doc;
  }

  // Set a shared signal value through the WASM bridge. The bridge exposes
  // window.__gosx_set_shared_signal_json(name, valueJSON) after the
  // runtime boots; the cross-frame relay (per ADR 0009 +
  // Bridge.EnableCrossFrameRelay) routes writes whose name starts with
  // "$preview." to the storefront iframe's Bridge, where slice 6's
  // preview_subscriber observes them and applies the matching DOM
  // mutation. Before the bridge boots, the call is a no-op — but the
  // BridgeShim's flag gate ensures we only run this path when the host
  // has opted in (which implies the bridge is mounted).
  function writeSharedSignal(name, value) {
    try {
      var setter = window.__gosx_set_shared_signal_json;
      if (typeof setter === "function") {
        setter(name, JSON.stringify(value));
      }
    } catch (error) {
      // Shared-signal channel not ready yet.
    }
  }

  function bindFieldMirroringIsland(root) {
    var scope = resolveScope(root);
    var inputs = scope.querySelectorAll("[data-editor-source], [data-studio-field-source], [data-editor-frame-attr-target]");
    Array.prototype.forEach.call(inputs, function (input) {
      if (input.dataset.gosxStudioFieldMirrorIslandBound === "true") return;
      input.dataset.gosxStudioFieldMirrorIslandBound = "true";
      var key = input.getAttribute("data-editor-source") || input.getAttribute("data-studio-field-source");
      var frameTarget = input.getAttribute("data-editor-frame-target");
      var attrTarget = input.getAttribute("data-editor-frame-attr-target");
      var attrName = input.getAttribute("data-editor-frame-attr");
      var attrPrefix = input.getAttribute("data-editor-frame-attr-prefix") || "";
      var attrSuffix = input.getAttribute("data-editor-frame-attr-suffix") || "";

      var updateNow = function () {
        // Per ADR 0008 + 0009: mirroring writes go through $preview.field
        // .<key> and $preview.text.<key> (the latter carries the full
        // applyTextUpdate detail object so the preview subscriber can
        // dispatch all three legacy fan-out branches: sourceKey
        // textContent, frameTarget textContent, attrTarget attribute).
        // Slice 6's transitional cleanup (Section G.1) removed the
        // legacy window.GoSXStudioPreviewRuntime.applyTextUpdate
        // delegation — the cross-frame relay (per ADR 0009) delivers
        // the signals to the iframe's subscriber where the mutation
        // happens locally.
        if (key) {
          writeSharedSignal("$preview.field." + key, input.value);
        }
        var detailKey = key || frameTarget || attrTarget || "_";
        writeSharedSignal("$preview.text." + detailKey, {
          sourceKey: key || "",
          frameTarget: frameTarget || "",
          attrTarget: attrTarget || "",
          attrName: attrName || "",
          attrPrefix: attrPrefix || "",
          attrSuffix: attrSuffix || "",
          value: input.value
        });
      };
      var update = frameTask(updateNow);
      input.addEventListener("input", update);
      if (frameTarget || attrTarget) {
        // Re-trigger on iframe load so the preview gets the current value
        // after a navigation. Matches legacy bindStudioFrameLoad behavior.
        Array.prototype.forEach.call(doc.querySelectorAll(".editor-preview-frame"), function (iframe) {
          var attr = "data-editor-frame-text-island-" + String(key || input.name || attrName || "field")
            .replace(/[^a-z0-9_-]/gi, "-")
            .toLowerCase();
          if (iframe.getAttribute(attr) === "true") return;
          iframe.setAttribute(attr, "true");
          iframe.addEventListener("load", function () { window.requestAnimationFrame(update); });
        });
      }
      updateNow();
    });
  }

  function bindFieldClipboardIsland(root) {
    var scope = resolveScope(root);
    var marker = doc.documentElement;
    if (marker.dataset.gosxStudioFieldClipboardIslandBound === "true") return;
    marker.dataset.gosxStudioFieldClipboardIslandBound = "true";

    function setCopyState(button, state) {
      if (!button) return;
      var previous = button.getAttribute("data-studio-copy-label") || button.textContent || "Copy";
      button.setAttribute("data-studio-copy-label", previous);
      button.setAttribute("data-studio-copy-state", state);
      button.textContent = state === "ok" ? "Copied" : state === "error" ? "Copy failed" : previous;
      window.setTimeout(function () {
        if (!doc.contains(button)) return;
        button.setAttribute("data-studio-copy-state", "idle");
        button.textContent = button.getAttribute("data-studio-copy-label") || previous;
      }, 1600);
    }

    function fallbackCopy(value, target) {
      var input = target && "select" in target ? target : null;
      if (!input) {
        input = doc.createElement("textarea");
        input.value = value;
        input.setAttribute("readonly", "readonly");
        input.style.position = "fixed";
        input.style.left = "-9999px";
        doc.body.appendChild(input);
      }
      input.focus();
      input.select();
      var ok = false;
      try {
        ok = doc.execCommand("copy");
      } catch (error) {
        ok = false;
      }
      if (input.parentNode === doc.body && target !== input) doc.body.removeChild(input);
      return ok;
    }

    doc.addEventListener("click", function (event) {
      var button = event.target.closest("[data-studio-copy-target]");
      if (!button || !scope.contains(button)) return;
      event.preventDefault();
      var selector = button.getAttribute("data-studio-copy-target");
      var target = selector ? doc.querySelector(selector) : null;
      var value = target ? (target.value || target.textContent || "") : "";
      if (!value) {
        setCopyState(button, "error");
        return;
      }
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(value).then(function () {
          setCopyState(button, "ok");
        }, function () {
          setCopyState(button, fallbackCopy(value, target) ? "ok" : "error");
        });
      } else {
        setCopyState(button, fallbackCopy(value, target) ? "ok" : "error");
      }
    });
  }

  function bindFieldRuntimeIsland(root) {
    bindFieldMirroringIsland(root);
    bindFieldClipboardIsland(root);
  }

  // Publish the island globals. The names are the contract referenced by
  // fieldruntime.IslandGlobals in runtime.go; the BridgeShim there delegates
  // window.GoSXStudioFieldRuntime calls here when the feature flag is on.
  window.__gosx_field_runtime_island_bind = bindFieldRuntimeIsland;
  window.__gosx_field_runtime_island_bindMirroring = bindFieldMirroringIsland;
  window.__gosx_field_runtime_island_bindClipboard = bindFieldClipboardIsland;
})();
