// subscriber_runtime.js — preview-side subscriber for the previewruntime
// $preview.* shared-signal namespace.
//
// Phase 3 slice-6 architectural piece. This script runs INSIDE the
// storefront iframe (when loaded via the editor preview's
// ?gosx-preview=1 query param, which causes
// island.EnablePreviewBootstrap() in the storefront layout to emit the
// gosx tiny WASM build + relay.js). The cross-frame postMessage relay
// (ADR 0009) routes $preview.* signal writes from the editor frame to
// the iframe's Bridge; the Bridge fans the writes out to in-frame
// __gosx_observe_shared_signal subscribers; THIS file is those
// subscribers, applying each signal change to the local iframe DOM.
//
// See:
//   - ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
//   - ~/.hyphae/spaces/m31labs-gosx/decisions/0009-iframe-transport-postmessage-relay.md
//   - ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-6-previewruntime.md
//
// The 10 subscribers below mirror the 10 legacy GoSXStudioPreviewRuntime
// methods in gosx-studio/assets/preview-runtime.js — applied locally
// instead of via cross-frame contentDocument reach-across.

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  // observe registers a subscriber against the in-frame shared-signal
  // store. The store fans out writes from the cross-frame relay (per
  // ADR 0009) so signals written in the editor frame fire this callback
  // in the iframe frame. When the relay hasn't booted yet, queue the
  // subscription on a pending list and replay once the observer hook
  // appears.
  var pending = [];
  function observe(name, callback) {
    try {
      var obs = window.__gosx_observe_shared_signal;
      if (typeof obs === "function") {
        obs(name, callback);
        return;
      }
    } catch (error) {
      // fall through to queue
    }
    pending.push([name, callback]);
  }

  // Drain the pending queue once the observer hook appears. The relay
  // installs __gosx_observe_shared_signal on bridge boot.
  function drainPending() {
    if (!pending.length) return;
    try {
      var obs = window.__gosx_observe_shared_signal;
      if (typeof obs !== "function") return;
      var snapshot = pending.slice();
      pending.length = 0;
      snapshot.forEach(function (entry) {
        try {
          obs(entry[0], entry[1]);
        } catch (error) {
          // observer rejected the registration; drop it
        }
      });
    } catch (error) {
      // observer not ready yet; will retry on next pulse
    }
  }
  // Poll for the observer hook every 16ms during boot, then stop once
  // pending is drained. Bounded retries so we don't poll forever if the
  // relay never appears.
  var bootRetries = 0;
  var bootPoll = window.setInterval(function () {
    drainPending();
    bootRetries += 1;
    if (!pending.length || bootRetries > 120) {
      window.clearInterval(bootPoll);
    }
  }, 16);

  // parseSignalJSON — defensive parse of the valueJSON callbacks
  // receive from the shared-signal observer.
  function parseSignalJSON(json, fallback) {
    if (json === undefined || json === null) return fallback;
    if (typeof json !== "string") return json;
    try {
      return JSON.parse(json);
    } catch (error) {
      return fallback;
    }
  }

  // ===== Subscriber 1/10: $preview.mount.epoch =====
  //
  // The editor calls mount() to seed preview-readiness. Iframe-side
  // response: re-runs the preview's local hydration (currently a no-op
  // because storefront pages don't carry the editor-preview overlay
  // chrome — the legacy mount() walked [data-editor-workbench] which
  // only exists in the editor frame). The subscriber emits a
  // mount-acknowledged class on documentElement so the editor can
  // observe readiness via cross-frame DOM polling if needed.
  observe("$preview.mount.epoch", function (valueJSON) {
    var epoch = parseSignalJSON(valueJSON, 0);
    if (doc.documentElement) {
      doc.documentElement.setAttribute("data-gosx-preview-mount-epoch", String(epoch));
    }
  });

  // ===== Subscriber 2/10: $preview.block.<key>.visible =====
  //
  // Mirrors setPreviewBlockVisibility at preview-runtime.js:866.
  //
  // The signal name is dynamic (one per block key). The relay forwards
  // every $preview.* write; the in-frame fan-out passes us the full name
  // through a prefix subscription. We use the wildcard hook
  // __gosx_observe_shared_signal_prefix if available; otherwise fall
  // back to a regular observe on a wildcard pattern the bridge supports.
  function observePrefix(prefix, callback) {
    try {
      var obs = window.__gosx_observe_shared_signal_prefix;
      if (typeof obs === "function") {
        obs(prefix, callback);
        return true;
      }
    } catch (error) {
      // fall through
    }
    return false;
  }

  // Block visibility uses a prefix subscription on $preview.block. so
  // every key's writes route through one callback. Each callback re-
  // extracts the key from the name and applies the toggle.
  observePrefix("$preview.block.", function (name, valueJSON) {
    var rest = name.slice("$preview.block.".length);
    var dot = rest.lastIndexOf(".visible");
    if (dot < 0) return;
    var key = rest.slice(0, dot);
    if (!key) return;
    var visible = parseSignalJSON(valueJSON, true);
    var node = doc.querySelector('[data-studio-block-key="' + attrValue(key) + '"]');
    if (node) node.style.display = visible ? "" : "none";
  });

  // ===== Subscriber 3/10: $preview.text.<source> =====
  //
  // Mirrors applyTextUpdate at preview-runtime.js:912.
  //
  // The payload is a JSON object carrying sourceKey / frameTarget /
  // attrTarget / attrName / attrPrefix / attrSuffix / value. The
  // subscriber dispatches all three legacy branches against the iframe
  // document.
  observePrefix("$preview.text.", function (name, valueJSON) {
    var detail = parseSignalJSON(valueJSON, null);
    if (!detail) return;
    var value = detail.value || "";
    if (detail.sourceKey) {
      Array.prototype.forEach.call(doc.querySelectorAll('[data-editor-preview="' + attrValue(detail.sourceKey) + '"]'), function (target) {
        target.textContent = value;
      });
    }
    if (detail.frameTarget) {
      Array.prototype.forEach.call(doc.querySelectorAll(detail.frameTarget), function (target) {
        target.textContent = value;
      });
    }
    if (detail.attrTarget && detail.attrName) {
      Array.prototype.forEach.call(doc.querySelectorAll(detail.attrTarget), function (target) {
        var normalized = String(value || "").trim();
        if (!normalized) target.removeAttribute(detail.attrName);
        else target.setAttribute(detail.attrName, (detail.attrPrefix || "") + normalized + (detail.attrSuffix || ""));
      });
    }
  });

  // ===== Subscribers 4a-g/10: $preview.theme.* family =====
  //
  // Mirrors applyTheme at preview-runtime.js:938.
  //
  // The legacy applies the entire payload atomically (clear classes,
  // add new classes, set color tokens). The signal family is decomposed
  // (one signal per sub-field) but we reassemble the latest values in
  // a module-scope buffer and apply on every write so any sub-field
  // change re-renders the theme correctly.
  var themeBuffer = {
    kit: "",
    template: "",
    palette: "",
    imageRatio: "",
    customClasses: [],
    styleClasses: [],
    colors: {}
  };

  function applyThemeFromBuffer() {
    var shell = doc.querySelector(".site-shell");
    if (!shell) return;
    var classes = [];
    if (themeBuffer.template) classes.push("theme--template-" + themeBuffer.template);
    if (themeBuffer.kit) classes.push("theme--kit-" + themeBuffer.kit);
    if (Array.isArray(themeBuffer.customClasses)) {
      themeBuffer.customClasses.forEach(function (className) {
        if (className) classes.push(className);
      });
    }
    if (Array.isArray(themeBuffer.styleClasses)) {
      themeBuffer.styleClasses.forEach(function (className) {
        if (className) classes.push(className);
      });
    }
    if (themeBuffer.palette) classes.push("theme--palette-" + themeBuffer.palette);
    if (themeBuffer.imageRatio) classes.push("theme--image-" + themeBuffer.imageRatio);

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
    var colors = themeBuffer.colors || {};
    Object.keys(colors).forEach(function (token) {
      doc.documentElement.style.setProperty(token, colors[token]);
      if (doc.body) doc.body.style.setProperty(token, colors[token]);
      shell.style.setProperty(token, colors[token]);
    });
  }

  observe("$preview.theme.kit", function (valueJSON) {
    themeBuffer.kit = parseSignalJSON(valueJSON, "");
    applyThemeFromBuffer();
  });
  observe("$preview.theme.template", function (valueJSON) {
    themeBuffer.template = parseSignalJSON(valueJSON, "");
    applyThemeFromBuffer();
  });
  observe("$preview.theme.palette", function (valueJSON) {
    themeBuffer.palette = parseSignalJSON(valueJSON, "");
    applyThemeFromBuffer();
  });
  observe("$preview.theme.imageRatio", function (valueJSON) {
    themeBuffer.imageRatio = parseSignalJSON(valueJSON, "");
    applyThemeFromBuffer();
  });
  observe("$preview.theme.customClasses", function (valueJSON) {
    themeBuffer.customClasses = parseSignalJSON(valueJSON, []);
    applyThemeFromBuffer();
  });
  observe("$preview.theme.styleClasses", function (valueJSON) {
    themeBuffer.styleClasses = parseSignalJSON(valueJSON, []);
    applyThemeFromBuffer();
  });
  observe("$preview.theme.colors", function (valueJSON) {
    themeBuffer.colors = parseSignalJSON(valueJSON, {});
    applyThemeFromBuffer();
  });

  // ===== Subscriber 5/10: $preview.style.impact.selector =====
  //
  // Mirrors applyStyleImpact at preview-runtime.js:986.
  //
  // Clears any existing [data-studio-style-impact-node] attributes,
  // then if selector is non-empty applies the attribute to every
  // matching node under .site-shell (or document) in the iframe.
  // The legacy returned a count; the signal-based path can't return
  // synchronously to the editor, so the editor maintains an advisory
  // count by querying its own view of the iframe (see island_runtime.js
  // applyStyleImpactIsland).
  observe("$preview.style.impact.selector", function (valueJSON) {
    var selector = parseSignalJSON(valueJSON, "");
    Array.prototype.forEach.call(doc.querySelectorAll("[data-studio-style-impact-node]"), function (node) {
      node.removeAttribute("data-studio-style-impact-node");
    });
    if (!selector) return;
    var scope = doc.querySelector(".site-shell") || doc;
    Array.prototype.forEach.call(scope.querySelectorAll(selector), function (node) {
      if (node.hasAttribute("data-studio-style-impact-node")) return;
      node.setAttribute("data-studio-style-impact-node", "true");
    });
  });

  // ===== Subscriber 6/10: $preview.theme.cssCustom =====
  //
  // Mirrors applyCSS / applyPreviewStyle at preview-runtime.js:1013.
  // Ensures a <style data-editor-live-css> element in the iframe head
  // and sets its textContent.
  function ensureStyle(attr) {
    var style = doc.querySelector("style[" + attr + "]");
    if (!style) {
      style = doc.createElement("style");
      style.setAttribute(attr, "true");
      (doc.head || doc.documentElement).appendChild(style);
    }
    return style;
  }
  observe("$preview.theme.cssCustom", function (valueJSON) {
    var cssText = parseSignalJSON(valueJSON, "");
    ensureStyle("data-editor-live-css").textContent = cssText || "";
  });

  // ===== Subscriber 7/10: $preview.theme.fontsCustom =====
  observe("$preview.theme.fontsCustom", function (valueJSON) {
    var cssText = parseSignalJSON(valueJSON, "");
    ensureStyle("data-editor-live-fonts").textContent = cssText || "";
  });

  // ===== Subscriber 8/10: $preview.brand.headerLogo =====
  //
  // Mirrors updateHeaderLogo at preview-runtime.js:1027.
  // Applies the payload to the iframe's .brand element.
  observe("$preview.brand.headerLogo", function (valueJSON) {
    var payload = parseSignalJSON(valueJSON, null);
    if (!payload) return;
    var brand = doc.querySelector(".brand");
    if (!brand) return;
    var logo = brand.querySelector(".brand__logo");
    if (!logo) return;
    var src = payload.src || "";
    brand.classList.toggle("brand--with-logo", !!src);
    logo.hidden = !src;
    logo.src = src;
    logo.alt = payload.alt || "Site logo";
    brand.style.setProperty("--brand-logo-width", (payload.width || "96") + "px");
    brand.style.setProperty("--brand-logo-offset-x", (payload.x || "0") + "px");
    brand.style.setProperty("--brand-logo-offset-y", (payload.y || "0") + "px");
  });

  // ===== Subscribers 9-10/10: $preview.editor.inlineEdit + fieldCycle =====
  //
  // The legacy requestInlineEdit / cycleField fan out to previewControllers
  // (the bindPreview machinery). In the iframe-side subscriber, the
  // preview controllers don't exist — they're an editor-frame concept
  // attached to [data-editor-workbench] forms. The subscriber records
  // the most-recent request id on documentElement so editor-side
  // observers can confirm delivery; the actual editing UI continues to
  // run in the editor frame.
  observe("$preview.editor.inlineEdit.requestId", function (valueJSON) {
    var id = parseSignalJSON(valueJSON, 0);
    if (doc.documentElement) {
      doc.documentElement.setAttribute("data-gosx-preview-inline-edit-request", String(id));
    }
  });
  observe("$preview.editor.fieldCycle.requestId", function (valueJSON) {
    var id = parseSignalJSON(valueJSON, 0);
    if (doc.documentElement) {
      doc.documentElement.setAttribute("data-gosx-preview-field-cycle-request", String(id));
    }
  });
})();
