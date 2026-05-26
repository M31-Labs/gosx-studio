// island_runtime.js — companion JS for the brandruntime .gsx islands.
//
// The .gsx islands (brand_logo.gsx, brand_header_logo.gsx) are mount-point
// markers in the editor DOM. This script publishes the
// window.__gosx_brand_runtime_island_bindLogo and
// window.__gosx_brand_runtime_island_updateHeaderLogo globals that
// brandruntime.BridgeShim delegates to when the "brand-runtime-islands"
// feature flag is on. It is emitted into the studio runtime bundle by
// brandruntime.IslandRuntimeJS() (see runtime.go) and runs before
// BridgeShim() at bundle init time.
//
// The two functions below replace window.GoSXStudioBrandRuntime.bindLogo
// and window.GoSXStudioBrandRuntime.updateHeaderLogo while preserving exact
// observable behavior of the legacy bindBrandLogo and updateHeaderLogo
// (assets/studio-engines.js:1264 / 1269).
//
// # bindLogo (editor-side only)
//
// Scans the editor authoring surface for the brand-logo controls
// (data-editor-brand-preview / data-editor-brand-logo / data-editor-brand-handle
// / data-editor-logo-url|alt|width|offset-x|offset-y|snap|snap-size|reset
// / data-editor-logo-readout) and wires up:
//
//   - input/change listeners on every form control to recompute the preview
//     CSS custom properties (--brand-logo-width / --brand-logo-offset-x /
//     --brand-logo-offset-y / --editor-logo-grid-size) and the readout text.
//   - pointer drag on the handle (with snap-aware grid stepping and an alt-
//     key bypass for fine adjustments).
//   - keyboard nudges on the handle (Arrow / Shift+Arrow / Alt+Arrow / Home).
//   - reset button that zeroes both offsets and refocuses the handle.
//   - frame-load hook (bindStudioFrameLoad) so the preview iframe re-runs
//     the update when its document load fires.
//
// Every change calls into updateHeaderLogo (below) so the storefront
// preview iframe header stays in sync with the editor authoring state.
// Writes a $brand.logo signal for the island's own bookkeeping.
//
// # updateHeaderLogo (transitional iframe delegation)
//
// Per ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md,
// long-term cross-frame mutations route through $preview.* shared signals.
// The slice 6 PreviewRuntime burn-down will ship a $preview.brand.headerLogo
// subscriber on the preview side. Until then, slice 3's updateHeaderLogo
// island keeps the legacy iframe contract by delegating to
// window.GoSXStudioPreviewRuntime.updateHeaderLogo — same DOM mutation, just
// owned by the brand island. Slice 6 will flip the mechanism to a shared-
// signal write without revisiting brand ownership.
//
// # Idempotency
//
// The island uses gosxStudioBrandLogoIslandBound (and per-input
// gosxStudioBrandLogoIslandInputBound, gosxStudioBrandLogoIslandHandleBound,
// gosxStudioBrandLogoIslandResetBound, gosxStudioBrandLogoIslandFrameBound)
// dataset keys distinct from the legacy gosxStudioBrandLogoBound so both
// implementations can coexist on the same DOM during the additive shipping
// window without double-binding event listeners.

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

  function clampNumber(value, min, max) {
    value = Number(value);
    if (!isFinite(value)) value = 0;
    if (min !== undefined && value < min) value = min;
    if (max !== undefined && value > max) value = max;
    return value;
  }

  function numericValue(input, fallback) {
    if (!input) return fallback;
    var raw = input.value;
    if (raw === "" || raw === null || raw === undefined) return fallback;
    var num = Number(raw);
    return isFinite(num) ? num : fallback;
  }

  function numericLimit(input, attr, fallback) {
    if (!input) return fallback;
    var raw = input.getAttribute(attr);
    if (raw === "" || raw === null || raw === undefined) return fallback;
    var num = Number(raw);
    return isFinite(num) ? num : fallback;
  }

  function snapValue(value, step) {
    step = step || 1;
    return Math.round(value / step) * step;
  }

  function setInputNumber(input, value, commit) {
    if (!input) return;
    var next = String(value);
    if (input.value === next) return;
    input.value = next;
    input.dispatchEvent(new Event("input", { bubbles: true }));
    if (commit) input.dispatchEvent(new Event("change", { bubbles: true }));
  }

  function brandPreviewRuntime() {
    return window.GoSXStudioPreviewRuntime || {};
  }

  // Set a shared signal value through the WASM bridge, if available. The
  // bridge exposes window.__gosx_set_shared_signal_json(name, valueJSON)
  // after the runtime boots; before then the call is a no-op (the legacy
  // attribute writes / preview-runtime delegation below still keep the UI
  // live).
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

  // Transitional iframe delegation. Slice 6 will replace the
  // brandPreviewRuntime() call below with a shared-signal write to
  // $preview.brand.headerLogo; the editor-side contract stays the same.
  function updateHeaderLogoIsland(payload) {
    payload = payload || {};
    var preview = brandPreviewRuntime();
    if (preview && typeof preview.updateHeaderLogo === "function") {
      preview.updateHeaderLogo(payload);
    }
    // Also write the $brand.headerLogo signal for island bookkeeping. This
    // is the editor-side observable; the preview-side observable is the
    // iframe DOM mutation done by PreviewRuntime above.
    writeSharedSignal("$brand.headerLogo", {
      url: payload.url || "",
      width: payload.width || "96",
      x: payload.x || "0",
      y: payload.y || "0",
      alt: payload.alt || "Site logo"
    });
  }

  // Frame-load helper mirroring legacy bindStudioFrameLoad — used so the
  // preview iframe re-runs the update when its document load fires.
  function bindStudioFrameLoadIsland(datasetKey, update) {
    var frames = doc.querySelectorAll("iframe");
    Array.prototype.forEach.call(frames, function (iframe) {
      if (iframe.dataset[datasetKey] === "true") return;
      iframe.dataset[datasetKey] = "true";
      iframe.addEventListener("load", function () {
        try {
          update();
        } catch (error) {
          // Cross-origin or detached frame; harmless.
        }
      });
    });
  }

  function bindBrandLogoIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    var preview = scope.querySelector("[data-editor-brand-preview]");
    var logo = scope.querySelector("[data-editor-brand-logo]");
    var handle = scope.querySelector("[data-editor-brand-handle]");
    var urlInput = scope.querySelector("[data-editor-logo-url]");
    var altInput = scope.querySelector("[data-editor-logo-alt]");
    var widthInput = scope.querySelector("[data-editor-logo-width]");
    var xInput = scope.querySelector("[data-editor-logo-offset-x]");
    var yInput = scope.querySelector("[data-editor-logo-offset-y]");
    var snapInput = scope.querySelector("[data-editor-logo-snap]");
    var snapSizeInput = scope.querySelector("[data-editor-logo-snap-size]");
    var resetButton = scope.querySelector("[data-editor-logo-reset]");
    var readout = scope.querySelector("[data-editor-logo-readout]");
    if (!preview || !logo || !handle || !urlInput) return;
    // Idempotency guard — distinct from legacy gosxStudioBrandLogoBound so
    // both implementations can coexist on the same DOM during the additive
    // shipping window without double-binding.
    if (preview.dataset && preview.dataset.gosxStudioBrandLogoIslandBound === "true") return;
    if (preview.dataset) preview.dataset.gosxStudioBrandLogoIslandBound = "true";

    function gridStep() {
      return clampNumber(numericValue(snapSizeInput, 8), 1, 32);
    }

    function snapped(value, bypassSnap) {
      if (bypassSnap || !snapInput || !snapInput.checked) return Math.round(value);
      return snapValue(value, gridStep());
    }

    function clampFor(input, value, fallbackMin, fallbackMax) {
      return clampNumber(value, numericLimit(input, "min", fallbackMin), numericLimit(input, "max", fallbackMax));
    }

    function applyPosition(x, y, commit, bypassSnap) {
      setInputNumber(xInput, clampFor(xInput, snapped(x, bypassSnap), -80, 320), commit);
      setInputNumber(yInput, clampFor(yInput, snapped(y, bypassSnap), -80, 320), commit);
      update();
    }

    var updateNow = function () {
      logo.setAttribute("src", urlInput.value || "");
      logo.setAttribute("alt", (altInput && altInput.value) || "Site logo");
      var width = widthInput && widthInput.value ? widthInput.value : "96";
      var x = xInput && xInput.value ? xInput.value : "0";
      var y = yInput && yInput.value ? yInput.value : "0";
      preview.style.setProperty("--brand-logo-width", width + "px");
      preview.style.setProperty("--brand-logo-offset-x", x + "px");
      preview.style.setProperty("--brand-logo-offset-y", y + "px");
      preview.style.setProperty("--editor-logo-grid-size", gridStep() + "px");
      handle.classList.toggle("is-empty", !urlInput.value);
      if (readout) readout.textContent = "X " + x + " / Y " + y + " / W " + width;
      var payload = {
        url: urlInput.value,
        width: width,
        x: x,
        y: y,
        alt: (altInput && altInput.value) || "Site logo"
      };
      updateHeaderLogoIsland(payload);
      // $brand.logo is the editor-side signal recording the authoring state.
      // The cross-frame mutation is done by updateHeaderLogoIsland above.
      writeSharedSignal("$brand.logo", payload);
    };
    var update = frameTask(updateNow);

    [urlInput, altInput, widthInput, xInput, yInput, snapInput, snapSizeInput].forEach(function (input) {
      if (!input || input.dataset.gosxStudioBrandLogoIslandInputBound === "true") return;
      input.dataset.gosxStudioBrandLogoIslandInputBound = "true";
      input.addEventListener("input", function () {
        if (input === snapInput || input === snapSizeInput) applyPosition(numericValue(xInput, 0), numericValue(yInput, 0), false, false);
        else update();
      });
      input.addEventListener("change", function () {
        if (input === snapInput || input === snapSizeInput) applyPosition(numericValue(xInput, 0), numericValue(yInput, 0), true, false);
        else update();
      });
    });

    if (resetButton && resetButton.dataset.gosxStudioBrandLogoIslandResetBound !== "true") {
      resetButton.dataset.gosxStudioBrandLogoIslandResetBound = "true";
      resetButton.addEventListener("click", function () {
        applyPosition(0, 0, true, true);
        handle.focus();
      });
    }

    if (handle.dataset.gosxStudioBrandLogoIslandHandleBound !== "true") {
      handle.dataset.gosxStudioBrandLogoIslandHandleBound = "true";
      var drag = null;
      handle.addEventListener("pointerdown", function (event) {
        if (event.button !== undefined && event.button !== 0) return;
        if (!urlInput.value) return;
        drag = {
          pointerId: event.pointerId,
          startX: event.clientX,
          startY: event.clientY,
          valueX: numericValue(xInput, 0),
          valueY: numericValue(yInput, 0)
        };
        handle.classList.add("is-dragging");
        handle.setPointerCapture(event.pointerId);
        event.preventDefault();
      });
      handle.addEventListener("pointermove", function (event) {
        if (!drag || drag.pointerId !== event.pointerId) return;
        applyPosition(drag.valueX + event.clientX - drag.startX, drag.valueY + event.clientY - drag.startY, false, event.altKey);
        event.preventDefault();
      });
      handle.addEventListener("pointerup", function (event) {
        if (!drag || drag.pointerId !== event.pointerId) return;
        handle.classList.remove("is-dragging");
        applyPosition(numericValue(xInput, 0), numericValue(yInput, 0), true, event.altKey);
        drag = null;
      });
      handle.addEventListener("pointercancel", function () {
        handle.classList.remove("is-dragging");
        drag = null;
      });
      handle.addEventListener("keydown", function (event) {
        var x = numericValue(xInput, 0);
        var y = numericValue(yInput, 0);
        var step = event.altKey ? 1 : gridStep();
        if (event.shiftKey) step = step * 4;
        if (event.key === "ArrowLeft") x -= step;
        else if (event.key === "ArrowRight") x += step;
        else if (event.key === "ArrowUp") y -= step;
        else if (event.key === "ArrowDown") y += step;
        else if (event.key === "Home") {
          x = 0;
          y = 0;
        } else {
          return;
        }
        applyPosition(x, y, true, event.altKey);
        event.preventDefault();
      });
    }
    bindStudioFrameLoadIsland("gosxStudioBrandLogoIslandFrameBound", update);
    updateNow();
  }

  // Publish the island globals. The names are the contracts referenced by
  // brandruntime.IslandGlobals in runtime.go; the BridgeShim there
  // delegates window.GoSXStudioBrandRuntime.bindLogo /
  // window.GoSXStudioBrandRuntime.updateHeaderLogo calls here when the
  // feature flag is on.
  window.__gosx_brand_runtime_island_bindLogo = bindBrandLogoIsland;
  window.__gosx_brand_runtime_island_updateHeaderLogo = updateHeaderLogoIsland;
})();
