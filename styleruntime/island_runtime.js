// island_runtime.js — companion JS for the styleruntime .gsx islands.
//
// The .gsx islands (style_theme.gsx, style_workbench.gsx, style_css.gsx,
// style_impact.gsx, style_controls.gsx) are mount-point markers in the
// editor DOM. This script publishes the ten
// window.__gosx_style_runtime_island_<method> globals that
// styleruntime.BridgeShim delegates to when the "style-runtime-islands"
// feature flag is on. It is emitted into the studio runtime bundle by
// styleruntime.IslandRuntimeJS() (see runtime.go) and runs before
// BridgeShim() at bundle init time.
//
// The ten functions below replace window.GoSXStudioStyleRuntime.{
// bindTheme, bindWorkbench, bindCSS, bindFonts, applyTheme,
// syncControlButtons, showImpact, restoreImpact, setControlValue,
// resetControlValue } while preserving exact observable behavior of the
// legacy implementations (bindTheme / bindStyleWorkbench / bindCSS /
// bindFonts / updateTheme / syncStyleControlButtons / showStyleImpact /
// restoreStyleImpact / setStyleControlValue / resetStyleControlValue at
// assets/studio-engines.js:1396–1940).
//
// # Method fan-out (group 1: editor-side binding)
//
// bindTheme(root)     — binds theme kit / template / palette / image-ratio /
//                       custom-template / style-class / color-token controls;
//                       on input/change dispatches the internal applyTheme +
//                       applies kit/preset side effects.
// bindWorkbench(root) — binds the editor-workbench form's click / pointer /
//                       focus listeners; dispatches setControlValue /
//                       resetControlValue and previews / restores impact.
// bindCSS(root)       — binds the [data-editor-css] textarea so input
//                       dispatches PreviewRuntime.applyCSS(value).
// bindFonts(root)     — binds the three [data-editor-font-name] /
//                       [data-editor-font-url] slot pairs so input/change
//                       recomputes the combined @font-face + :root font-var
//                       CSS and dispatches PreviewRuntime.applyFonts(css).
//
// # Method fan-out (group 2: editor-side state)
//
// syncControlButtons(root) — for every [data-studio-style-control],
//                            [data-studio-style-readout],
//                            [data-studio-style-reset] in root, sync
//                            aria-pressed / .is-selected / textContent
//                            against live form values and kit defaults.
// setControlValue(name, value) — sets the named control's value (radio /
//                                input / select), dispatches input/change
//                                events, then runs syncControlButtons +
//                                showImpact(name, value, true).
// resetControlValue(name)      — looks up kit default for name and calls
//                                setControlValue(name, default).
//
// # Method fan-out (group 3: iframe-crossing transitional)
//
// applyTheme()                       — reads theme form state, calls
//                                      PreviewRuntime.applyTheme(payload),
//                                      updates editor-side swatches / kit /
//                                      template / custom-template builder,
//                                      then syncControlButtons.
// showImpact(name, value, committed) — looks up impact metadata, calls
//                                      PreviewRuntime.applyStyleImpact for
//                                      count, writes editor-side readouts,
//                                      toggles impact-panel classes.
// restoreImpact()                    — restores last committed impact via
//                                      showImpact, or clears readouts + calls
//                                      PreviewRuntime.applyStyleImpact("").
//
// # Idempotency
//
// Idempotency guards use distinct dataset keys from the legacy
// implementation so both code paths can run on the same DOM during the
// additive shipping window without double-binding (compare to slice 3's
// bindBrandLogo pattern):
//
//   gosxStudioThemeIslandBound          — per theme control input
//   gosxStudioStyleWorkbenchIslandBound — per editor-workbench form
//   gosxStudioCSSIslandBound            — per CSS textarea
//   gosxStudioFontIslandBound           — per font name/url input
//   gosxStudioThemeIslandFrameBound     — per preview iframe (theme load)
//   gosxStudioCSSIslandFrameBound       — per preview iframe (CSS load)
//   gosxStudioFontIslandFrameBound      — per preview iframe (font load)
//   gosxStudioStyleImpactIslandFrameBound — per preview iframe (impact restore)

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  // ===== Shared helpers (mirror the legacy private helpers) =====

  // attrValue — escapes backslashes and double quotes so an arbitrary string
  // can be embedded inside a CSS attribute selector double-quoted value.
  // Mirrors the legacy attrValue helper at studio-engines.js:2158. Used by
  // namedControl / styleControlValue / setControlValue.
  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  // frame — schedule a callback at next animation frame (mirrors
  // studio-engines.js:75 helper). Used by frameTask.
  function frame(callback) {
    if (window.requestAnimationFrame) {
      window.requestAnimationFrame(callback);
      return;
    }
    window.setTimeout(callback, 16);
  }

  // frameTask — coalesces calls into a single rAF dispatch with the last
  // arguments (mirrors studio-engines.js:83 helper). Used by the bind*
  // methods to debounce input/change handlers.
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

  // bindStudioFrameLoad — attach an iframe load listener that fires update on
  // next rAF (mirrors studio-engines.js:99 helper). The attr argument is the
  // per-iframe idempotency guard. Used by bindTheme / bindCSS / bindFonts /
  // bindWorkbench so the editor re-applies styling when the storefront
  // iframe finishes loading.
  function bindStudioFrameLoad(attr, update) {
    Array.prototype.forEach.call(doc.querySelectorAll(".editor-preview-frame"), function (f) {
      if (f.getAttribute(attr) === "true") return;
      f.setAttribute(attr, "true");
      f.addEventListener("load", function () {
        window.requestAnimationFrame(update);
      });
    });
  }

  // stylePreviewRuntime — accessor for the legacy preview runtime the three
  // transitional iframe-crossing methods delegate to. Slice 6 will replace
  // these delegations with $preview.theme.* / $preview.style.impact.*
  // shared-signal writes per
  // ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md.
  function stylePreviewRuntime() {
    return window.GoSXStudioPreviewRuntime || {};
  }

  // editorWorkbench — resolve the [data-editor-workbench] form (or root if it
  // matches). Mirrors studio-engines.js:69 helper. Used by bindWorkbench /
  // showImpact / restoreImpact for scope path readouts.
  function editorWorkbench(root) {
    if (root && root.matches && root.matches("[data-editor-workbench]")) return root;
    return doc.querySelector("[data-editor-workbench]");
  }

  // ===== Theme form readers (mirror studio-engines.js helpers) =====

  function themeKitValue() {
    var radio = doc.querySelector('input[name="themeKit"]:checked');
    return radio ? radio.value : "";
  }

  function themeTemplateValue() {
    var radio = doc.querySelector('input[name="themeTemplate"]:checked');
    return radio ? radio.value : "studio";
  }

  function themePaletteValue() {
    var select = doc.querySelector("select[name='themePalette']");
    if (select) return select.value;
    var radio = doc.querySelector('input[name="themePalette"]:checked');
    return radio ? radio.value : "clay";
  }

  function themeColors() {
    var out = {};
    Array.prototype.forEach.call(doc.querySelectorAll("[data-editor-color-token]"), function (input) {
      var name = input.getAttribute("data-editor-color-token");
      if (name && input.value) out[name] = input.value;
    });
    return out;
  }

  function customValue(name, fallback) {
    var input = doc.querySelector('[name="' + name + '"]');
    return input && input.value ? input.value : fallback;
  }

  function customTemplateClasses() {
    return [
      "theme--custom-hero-" + customValue("customHeroLayout", "overlay"),
      "theme--custom-width-" + customValue("customContentWidth", "standard"),
      "theme--custom-density-" + customValue("customProductDensity", "balanced"),
      "theme--custom-height-" + customValue("customHeroHeight", "standard")
    ];
  }

  function styleClasses() {
    return [
      "theme--nav-" + customValue("styleNav", "links"),
      "theme--buttons-" + customValue("styleButtons", "pill"),
      "theme--cards-" + customValue("styleCards", "quiet"),
      "theme--spacing-" + customValue("styleSpacing", "balanced"),
      "theme--images-" + customValue("styleImageFrame", "sharp"),
      "theme--motion-" + customValue("styleMotion", "subtle")
    ];
  }

  function namedControl(name) {
    return doc.querySelector('[name="' + attrValue(name) + '"]');
  }

  function setInputValue(name, value) {
    var input = doc.querySelector('[name="' + name + '"]');
    if (!input || !value) return;
    input.value = value;
  }

  function setSelectValue(name, value) {
    var select = doc.querySelector('[name="' + name + '"]');
    if (!select || !value) return;
    select.value = value;
  }

  function setRadioValue(name, value) {
    if (!value) return;
    var radio = doc.querySelector('input[name="' + name + '"][value="' + value + '"]');
    if (radio) radio.checked = true;
  }

  function selectedKitCard() {
    var input = doc.querySelector('input[name="themeKit"]:checked');
    if (input && input.closest) return input.closest("[data-editor-kit-card]");
    return null;
  }

  function applyKit(input) {
    var card = input && input.closest ? input.closest("[data-editor-kit-card]") : null;
    if (!card) return;
    setRadioValue("themeTemplate", card.getAttribute("data-kit-template"));
    setSelectValue("themePalette", card.getAttribute("data-kit-palette"));
    setRadioValue("themeImageRatio", card.getAttribute("data-kit-image-ratio"));
    setInputValue("customTemplateName", card.getAttribute("data-kit-custom-name"));
    setSelectValue("customHeroLayout", card.getAttribute("data-kit-custom-hero-layout"));
    setSelectValue("customContentWidth", card.getAttribute("data-kit-custom-content-width"));
    setSelectValue("customProductDensity", card.getAttribute("data-kit-custom-product-density"));
    setSelectValue("customHeroHeight", card.getAttribute("data-kit-custom-hero-height"));
    setSelectValue("styleNav", card.getAttribute("data-kit-style-nav"));
    setSelectValue("styleButtons", card.getAttribute("data-kit-style-buttons"));
    setSelectValue("styleCards", card.getAttribute("data-kit-style-cards"));
    setSelectValue("styleSpacing", card.getAttribute("data-kit-style-spacing"));
    setSelectValue("styleImageFrame", card.getAttribute("data-kit-style-image-frame"));
    setSelectValue("styleMotion", card.getAttribute("data-kit-style-motion"));
    applyPresetDefaults(doc.querySelector("select[name='themePalette']"));
  }

  function applyPresetDefaults(select) {
    var option = select && select.options ? select.options[select.selectedIndex] : null;
    if (!option) return;
    Array.prototype.forEach.call(doc.querySelectorAll("[data-editor-color-token]"), function (input) {
      var key = input.getAttribute("data-editor-color-key");
      var attr = "data-color-" + String(key || "").replace(/[A-Z]/g, function (letter) { return "-" + letter.toLowerCase(); });
      var value = option.getAttribute(attr);
      if (value) {
        input.value = value;
        input.dispatchEvent(new Event("input", { bubbles: true }));
      }
    });
  }

  // ===== Group 1: editor-side binding =====

  // bindTheme(root) — mirrors bindTheme at studio-engines.js:1421.
  // Binds the seven theme input selectors so input/change dispatches the
  // internal applyTheme via frameTask (coalesced to one rAF). Also wires
  // applyKit when a [data-editor-kit-input] changes, and applyPresetDefaults
  // when a [data-editor-theme-preset] changes. Re-binds on preview-frame
  // load via bindStudioFrameLoad. Calls applyTheme on bind to populate the
  // preview with the current form state.
  //
  // Idempotency: each control gets [data-gosx-studio-theme-island-bound]
  // distinct from the legacy [data-gosx-studio-theme-bound] so both paths
  // can coexist during the additive shipping window.
  function bindThemeIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    var controls = scope.querySelectorAll('[name="themeKit"], [name="themeTemplate"], [name="themePalette"], [name="customTemplateName"], [data-editor-custom-class], [data-editor-style-class], input[name="themeImageRatio"], [data-editor-color-token]');
    var updateThemeFrame = frameTask(applyThemeIsland);
    Array.prototype.forEach.call(controls, function (control) {
      if (control.dataset.gosxStudioThemeIslandBound === "true") return;
      control.dataset.gosxStudioThemeIslandBound = "true";
      control.addEventListener("input", updateThemeFrame);
      control.addEventListener("change", function () {
        if (control.matches("[data-editor-kit-input]")) applyKit(control);
        if (control.matches("[data-editor-theme-preset]")) applyPresetDefaults(control);
        updateThemeFrame();
      });
    });
    bindStudioFrameLoad("data-editor-theme-island-frame-bound", updateThemeFrame);
    applyThemeIsland();
  }

  window.__gosx_style_runtime_island_bindTheme = bindThemeIsland;

  // ===== Group 3: iframe-crossing transitional (forward-declared so bindTheme can reference) =====

  // applyTheme() — mirrors updateTheme at studio-engines.js:1396.
  // Reads the theme form state (kit / template / palette / image-ratio /
  // custom-template / style classes / color tokens), calls
  // PreviewRuntime.applyTheme(payload) for the cross-frame DOM mutation,
  // updates editor-side swatches / kit cards / template cards / custom-
  // template builder, then runs syncControlButtonsIsland(document).
  //
  // Per ADR 0008 transitional pattern (same shape as slice 4's
  // updateVisibilityState), ownership has moved to the styleruntime island
  // but the iframe-crossing mechanism stays the same — slice 6 will swap
  // to a $preview.theme.* shared-signal write subscribed by the preview
  // document without revisiting styleruntime ownership.
  function applyThemeIsland() {
    var kit = themeKitValue();
    var template = themeTemplateValue();
    var palette = themePaletteValue();
    var ratio = doc.querySelector('input[name="themeImageRatio"]:checked');
    var colors = themeColors();
    var preview = stylePreviewRuntime();
    if (typeof preview.applyTheme === "function") {
      preview.applyTheme({
        kit: kit,
        template: template,
        palette: palette,
        imageRatio: ratio ? ratio.value : "",
        customClasses: template === "custom" ? customTemplateClasses() : [],
        styleClasses: styleClasses(),
        colors: colors
      });
    }
    updateThemeSwatchesIsland(colors);
    updateKitCardsIsland(kit);
    updateTemplateCardsIsland(template);
    updateCustomTemplateBuilderIsland(template);
    if (typeof syncControlButtonsIsland === "function") syncControlButtonsIsland(doc);
  }

  function updateThemeSwatchesIsland(colors) {
    var swatches = doc.querySelectorAll("[data-editor-theme-swatches] .theme-swatch");
    var values = Object.keys(colors).map(function (key) { return colors[key]; });
    Array.prototype.forEach.call(swatches, function (swatch, index) {
      if (values[index]) swatch.style.background = values[index];
    });
  }

  function updateKitCardsIsland(active) {
    Array.prototype.forEach.call(doc.querySelectorAll("[data-editor-kit-card]"), function (card) {
      card.classList.toggle("is-selected", card.getAttribute("data-editor-kit-card") === active);
    });
  }

  function updateTemplateCardsIsland(active) {
    Array.prototype.forEach.call(doc.querySelectorAll("[data-editor-template-card]"), function (card) {
      card.classList.toggle("is-selected", card.getAttribute("data-editor-template-card") === active);
    });
    var customName = doc.querySelector("[data-editor-custom-template-name]");
    var customCard = doc.querySelector('[data-editor-template-card="custom"] strong');
    if (customName && customCard && customName.value.trim()) customCard.textContent = customName.value.trim();
  }

  function updateCustomTemplateBuilderIsland(active) {
    var builder = doc.querySelector("[data-custom-template-builder]");
    if (builder) builder.classList.toggle("is-active", active === "custom");
  }

  // applyTheme is also published as the public island global so the
  // BridgeShim can dispatch GoSXStudioStyleRuntime.applyTheme() calls into
  // it. Defined above as applyThemeIsland (shared with bindTheme's update
  // dispatch).
  window.__gosx_style_runtime_island_applyTheme = applyThemeIsland;

  // bindWorkbench(root) — mirrors bindStyleWorkbench at studio-engines.js:1884.
  // Resolves the [data-editor-workbench] form (or root if it matches),
  // guards re-entry via [data-gosx-studio-style-workbench-island-bound]
  // distinct from the legacy [data-gosx-studio-style-workbench-bound], and
  // binds the click / pointer / focus / input / change listeners that fan
  // out to setControlValue / resetControlValue / showImpact / restoreImpact.
  // bindStudioFrameLoad("data-editor-style-impact-island-frame-bound") so
  // restoreImpact re-runs after the storefront iframe reloads. Calls
  // syncControlButtons(form) + restoreImpact() on bind to seed the panel.
  function bindWorkbenchIsland(root) {
    var form = editorWorkbench(root);
    if (!form || form.dataset.gosxStudioStyleWorkbenchIslandBound === "true") return;
    form.dataset.gosxStudioStyleWorkbenchIslandBound = "true";
    var previewImpact = frameTask(showImpactIsland);
    var restoreImpactFrame = frameTask(restoreImpactIsland);
    var syncControlsFrame = frameTask(function () {
      syncControlButtonsIsland(form);
      restoreImpactIsland();
    });
    form.addEventListener("click", function (event) {
      var resetButton = event.target.closest("[data-studio-style-reset]");
      if (resetButton && form.contains(resetButton)) {
        event.preventDefault();
        resetControlValueIsland(resetButton.getAttribute("data-studio-style-reset"));
        return;
      }
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      event.preventDefault();
      setControlValueIsland(button.getAttribute("data-studio-style-control"), button.getAttribute("data-studio-style-value"));
    });
    form.addEventListener("pointerover", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      previewImpact(button.getAttribute("data-studio-style-control"), button.getAttribute("data-studio-style-value"), false);
    });
    form.addEventListener("pointerout", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      if (button.contains(event.relatedTarget)) return;
      restoreImpactFrame();
    });
    form.addEventListener("focusin", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      previewImpact(button.getAttribute("data-studio-style-control"), button.getAttribute("data-studio-style-value"), false);
    });
    form.addEventListener("focusout", function (event) {
      var button = event.target.closest("[data-studio-style-control]");
      if (!button || !form.contains(button)) return;
      restoreImpactFrame();
    });
    form.addEventListener("input", function (event) {
      if (event.target && event.target.matches && event.target.matches('[name="themeTemplate"], [name="themeImageRatio"], [data-editor-custom-class], [data-editor-style-class]')) {
        syncControlsFrame();
      }
    });
    form.addEventListener("change", function (event) {
      if (event.target && event.target.matches && event.target.matches('[name="themeTemplate"], [name="themeImageRatio"], [data-editor-custom-class], [data-editor-style-class]')) {
        syncControlsFrame();
      }
    });
    bindStudioFrameLoad("data-editor-style-impact-island-frame-bound", restoreImpactFrame);
    syncControlButtonsIsland(form);
    restoreImpactIsland();
  }

  window.__gosx_style_runtime_island_bindWorkbench = bindWorkbenchIsland;

  // bindCSS(root) — mirrors bindCSS at studio-engines.js:1570.
  // Binds the [data-editor-css] textarea: input -> frameTask -> apply via
  // PreviewRuntime.applyCSS(value). Re-binds on preview-frame load. Calls
  // applyCSS once on bind so the preview starts in sync with the textarea.
  //
  // Idempotency: per-textarea [data-gosx-studio-css-island-bound] distinct
  // from legacy [data-gosx-studio-css-bound] so both paths coexist.
  function bindCSSIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    var input = scope.querySelector("[data-editor-css]");
    if (!input || input.dataset.gosxStudioCSSIslandBound === "true") return;
    input.dataset.gosxStudioCSSIslandBound = "true";
    var updateNow = function () {
      var preview = stylePreviewRuntime();
      if (typeof preview.applyCSS === "function") preview.applyCSS(input.value || "");
    };
    var update = frameTask(updateNow);
    input.addEventListener("input", update);
    bindStudioFrameLoad("data-editor-css-island-frame-bound", update);
    updateNow();
  }

  window.__gosx_style_runtime_island_bindCSS = bindCSSIsland;

  // cssString / cssURL / fontFormat — mirror the helpers at
  // studio-engines.js:1585 / 1589 / 1593. cssString escapes a font family
  // name for safe embedding in a CSS string literal; cssURL escapes a font
  // file URL; fontFormat picks the appropriate format() hint from the URL
  // file extension.
  function cssString(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"').replace(/[\n\r]/g, " ").trim();
  }

  function cssURL(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, "%22").replace(/[\n\r]/g, "").trim();
  }

  function fontFormat(url) {
    var clean = String(url || "").split("?")[0].toLowerCase();
    if (clean.endsWith(".woff2")) return ' format("woff2")';
    if (clean.endsWith(".woff")) return ' format("woff")';
    if (clean.endsWith(".ttf")) return ' format("truetype")';
    if (clean.endsWith(".otf")) return ' format("opentype")';
    return "";
  }

  // bindFonts(root) — mirrors bindFonts at studio-engines.js:1602.
  // For each of the three slots (display / body / mono) reads the matching
  // [data-editor-font-name] + [data-editor-font-url] inputs, composes the
  // @font-face rule (when both family and source are present) and the
  // :root font-variable assignment (when family is present), and dispatches
  // PreviewRuntime.applyFonts(combined-css). frameTask-coalesced. Re-binds
  // on preview-frame load and calls updateNow once on bind to seed the
  // preview with the current state.
  //
  // Idempotency: per-input [data-gosx-studio-font-island-bound] distinct
  // from the legacy [data-gosx-studio-font-bound] so both paths coexist.
  function bindFontsIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    var inputs = scope.querySelectorAll("[data-editor-font-name], [data-editor-font-url]");
    if (!inputs.length) return;
    var updateNow = function () {
      var css = [];
      var vars = [];
      ["display", "body", "mono"].forEach(function (slot) {
        var name = scope.querySelector('[data-editor-font-name="' + slot + '"]');
        var url = scope.querySelector('[data-editor-font-url="' + slot + '"]');
        var family = name && name.value ? name.value.trim() : "";
        var source = url && url.value ? url.value.trim() : "";
        if (family && source) {
          css.push('@font-face{font-family:"' + cssString(family) + '";src:url("' + cssURL(source) + '")' + fontFormat(source) + ';font-display:swap;}');
        }
        if (family) {
          var token = slot === "display" ? "--font-display" : slot === "body" ? "--font-body" : "--font-mono";
          vars.push(token + ':"' + cssString(family) + '";');
        }
      });
      if (vars.length) css.push(":root{" + vars.join("") + "}");
      var preview = stylePreviewRuntime();
      if (typeof preview.applyFonts === "function") preview.applyFonts(css.join("\n"));
    };
    var update = frameTask(updateNow);
    Array.prototype.forEach.call(inputs, function (input) {
      if (input.dataset.gosxStudioFontIslandBound === "true") return;
      input.dataset.gosxStudioFontIslandBound = "true";
      input.addEventListener("input", update);
      input.addEventListener("change", update);
    });
    bindStudioFrameLoad("data-editor-font-island-frame-bound", update);
    updateNow();
  }

  window.__gosx_style_runtime_island_bindFonts = bindFontsIsland;

  // ===== Group 2: editor-side state =====

  // ownerLabel — mirror studio-engines.js:1637. Convert kebab/snake_case
  // value into a Title-cased label for the editor readouts.
  function ownerLabel(value) {
    value = String(value || "").replace(/[-_]/g, " ");
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  // styleControlValue — mirror studio-engines.js:1652. Return the live form
  // value for the named control (checked radio takes precedence; else the
  // namedControl's value).
  function styleControlValue(name) {
    var checked = doc.querySelector('input[name="' + attrValue(name) + '"]:checked');
    if (checked) return checked.value;
    var control = namedControl(name);
    return control ? control.value || "" : "";
  }

  // kitDefaultForControl — mirror studio-engines.js:1659. For named style
  // controls, look up the active kit card's per-control default attribute;
  // fall back to the per-control hardcoded fallback when no card matches.
  function kitDefaultForControl(name) {
    var attrs = {
      customHeroLayout: "data-kit-custom-hero-layout",
      customContentWidth: "data-kit-custom-content-width",
      customProductDensity: "data-kit-custom-product-density",
      customHeroHeight: "data-kit-custom-hero-height",
      styleNav: "data-kit-style-nav",
      styleButtons: "data-kit-style-buttons",
      styleCards: "data-kit-style-cards",
      styleSpacing: "data-kit-style-spacing",
      styleImageFrame: "data-kit-style-image-frame",
      styleMotion: "data-kit-style-motion",
      themeImageRatio: "data-kit-image-ratio"
    };
    var fallbacks = {
      customHeroLayout: "overlay",
      customContentWidth: "standard",
      customProductDensity: "balanced",
      customHeroHeight: "standard",
      styleNav: "links",
      styleButtons: "pill",
      styleCards: "quiet",
      styleSpacing: "balanced",
      styleImageFrame: "sharp",
      styleMotion: "subtle",
      themeImageRatio: "landscape"
    };
    var card = selectedKitCard();
    var attr = attrs[name];
    var value = card && attr ? card.getAttribute(attr) : "";
    return value || fallbacks[name] || "";
  }

  // styleImpactFor — mirror studio-engines.js:1692. Returns the {label,
  // summary, selector} metadata for a named style control. Used by
  // showImpact for the preview overlay + readouts and by syncControlButtons
  // for the reset-button aria-label.
  function styleImpactFor(name) {
    var catalog = {
      customHeroLayout: {
        label: "Page shape",
        summary: "Hero composition and above-the-fold frame",
        selector: ".hero, .hero__copy, .hero__media"
      },
      customContentWidth: {
        label: "Page width",
        summary: "Site frame, sections, and reading measure",
        selector: ".site-shell, .hero, .section"
      },
      customHeroHeight: {
        label: "Hero height",
        summary: "Hero depth and first viewport balance",
        selector: ".hero"
      },
      styleSpacing: {
        label: "Section rhythm",
        summary: "Vertical rhythm, grids, and section breathing room",
        selector: ".hero, .section, .product-grid, .post-grid, .gallery-strip"
      },
      styleCards: {
        label: "Cards",
        summary: "Product, post, and gallery card treatment",
        selector: ".product-card, .post-card, .gallery-card"
      },
      customProductDensity: {
        label: "Product rhythm",
        summary: "Product grid density and commerce scan rhythm",
        selector: ".product-grid, .product-card"
      },
      styleNav: {
        label: "Navigation",
        summary: "Header, brand, and primary navigation treatment",
        selector: ".site-header, .brand, nav, .nav"
      },
      styleButtons: {
        label: "Buttons",
        summary: "Primary and secondary action treatment",
        selector: ".button"
      },
      styleImageFrame: {
        label: "Image frame",
        summary: "Media corner shape and card image treatment",
        selector: ".hero__media, img, .product-card, .post-card, .gallery-card"
      },
      themeImageRatio: {
        label: "Image crop",
        summary: "Image aspect rhythm across product, blog, and gallery media",
        selector: ".hero__media, img, .product-card img, .post-card img, .gallery-card img"
      },
      styleMotion: {
        label: "Motion",
        summary: "Interactive movement across links, cards, and actions",
        selector: ".site-shell, .hero, .section, .button, .product-card, .post-card, .gallery-card"
      }
    };
    return catalog[name] || {
      label: ownerLabel(name),
      summary: "Scoped style surface",
      selector: ".site-shell"
    };
  }

  // syncControlButtons(root) — mirrors syncStyleControlButtons at
  // studio-engines.js:1848. Walks the three sub-control families in scope
  // (controls / readouts / resets) and syncs each against the live form
  // value + kit defaults. Used by bindWorkbench / applyTheme /
  // setControlValue to keep panel state coherent after any input change.
  function syncControlButtonsIsland(root) {
    root = root || doc;
    Array.prototype.forEach.call(root.querySelectorAll("[data-studio-style-control]"), function (button) {
      var name = button.getAttribute("data-studio-style-control") || "";
      var value = button.getAttribute("data-studio-style-value") || "";
      var active = !!name && styleControlValue(name) === value;
      button.setAttribute("aria-pressed", active ? "true" : "false");
      button.classList.toggle("is-selected", active);
    });
    Array.prototype.forEach.call(root.querySelectorAll("[data-studio-style-readout]"), function (readout) {
      var name = readout.getAttribute("data-studio-style-readout") || "";
      var value = styleControlValue(name);
      var defaultValue = kitDefaultForControl(name);
      var inherited = !!defaultValue && value === defaultValue;
      readout.textContent = value ? ownerLabel(value) : "Inherited";
      readout.setAttribute("data-studio-style-inherited", inherited ? "true" : "false");
    });
    Array.prototype.forEach.call(root.querySelectorAll("[data-studio-style-reset]"), function (button) {
      var name = button.getAttribute("data-studio-style-reset") || "";
      var value = styleControlValue(name);
      var defaultValue = kitDefaultForControl(name);
      var inherited = !!defaultValue && value === defaultValue;
      var group = button.closest(".studio-style-control-group");
      var meta = styleImpactFor(name);
      if (group) group.setAttribute("data-studio-style-inherited", inherited ? "true" : "false");
      button.disabled = inherited;
      button.textContent = inherited ? "Kit" : "Reset";
      button.setAttribute(
        "aria-label",
        inherited
          ? meta.label + " is using the starter kit"
          : "Reset " + meta.label + " to " + ownerLabel(defaultValue)
      );
    });
  }

  window.__gosx_style_runtime_island_syncControlButtons = syncControlButtonsIsland;

  // ===== Group 3: iframe-crossing transitional (continued) =====

  // lastStyleImpact — module-scope state tracking the most-recent committed
  // impact, mirroring the legacy bridge's `lastStyleImpact` variable at
  // studio-engines.js:4. restoreImpact uses this to repaint the panel when
  // pointer leaves a control after a commit.
  var lastStyleImpact = null;

  function setImpactReadout(selector, value) {
    Array.prototype.forEach.call(doc.querySelectorAll(selector), function (node) {
      node.textContent = value;
    });
  }

  function styleScopePath(form) {
    var bits = ["site", "home"];
    if (form) {
      var selection = form.getAttribute("data-studio-selection") || "";
      var field = form.getAttribute("data-studio-field-selection") || "";
      if (selection) bits.push(selection);
      if (field) bits.push(field);
    }
    return bits.join(" > ");
  }

  function styleStatePath(form) {
    var state = form && form.getAttribute("data-studio-style-state") || "default";
    var breakpoint = form && form.getAttribute("data-studio-breakpoint") || "desktop";
    return state + " / " + breakpoint;
  }

  function clearStyleImpactNodes() {
    var preview = stylePreviewRuntime();
    if (typeof preview.applyStyleImpact === "function") preview.applyStyleImpact("");
  }

  // showImpact(name, value, committed) — mirrors showStyleImpact at
  // studio-engines.js:1785. The legacy uses a single ternary for the
  // affected-count label, "1 affected" vs "N affected" — both branches
  // emit " affected" with a space prefix so the singular and plural read
  // identically; preserved verbatim for parity.
  //
  // Transitional iframe delegation per ADR 0008 — slice 6 will swap the
  // PreviewRuntime.applyStyleImpact call for a $preview.style.impact.selector
  // shared-signal write subscribed by the preview document.
  function showImpactIsland(name, value, committed) {
    var meta = styleImpactFor(name);
    var form = editorWorkbench(doc);
    var preview = stylePreviewRuntime();
    var count = typeof preview.applyStyleImpact === "function" ? preview.applyStyleImpact(meta.selector) : 0;
    var label = meta.label + " / " + ownerLabel(value || styleControlValue(name));
    setImpactReadout("[data-studio-style-impact-label]", label);
    setImpactReadout("[data-studio-style-impact-summary]", meta.summary);
    setImpactReadout("[data-studio-style-impact-count]", count + (count === 1 ? " affected" : " affected"));
    setImpactReadout("[data-studio-style-impact-scope]", styleScopePath(form));
    setImpactReadout("[data-studio-style-impact-state]", styleStatePath(form));
    Array.prototype.forEach.call(doc.querySelectorAll("[data-studio-style-impact-panel]"), function (panel) {
      panel.classList.toggle("is-live", !committed);
      panel.classList.toggle("has-change", !!committed);
    });
    if (committed) lastStyleImpact = { name: name, value: value };
  }

  window.__gosx_style_runtime_island_showImpact = showImpactIsland;

  // restoreImpact() — mirrors restoreStyleImpact at studio-engines.js:1803.
  // Replays the last committed impact when present; otherwise clears the
  // editor-side readouts to the no-active-recipe state and removes the
  // preview overlay markers via PreviewRuntime.applyStyleImpact("").
  //
  // Transitional iframe delegation per ADR 0008 — slice 6 will swap the
  // PreviewRuntime.applyStyleImpact("") call for a $preview.style.impact
  // .selector = "" shared-signal write subscribed by the preview document.
  function restoreImpactIsland() {
    if (lastStyleImpact) {
      showImpactIsland(lastStyleImpact.name, lastStyleImpact.value, true);
      return;
    }
    clearStyleImpactNodes();
    setImpactReadout("[data-studio-style-impact-label]", "No active recipe");
    setImpactReadout("[data-studio-style-impact-summary]", "Awaiting scoped change.");
    setImpactReadout("[data-studio-style-impact-count]", "0 affected");
    var form = editorWorkbench(doc);
    setImpactReadout("[data-studio-style-impact-scope]", styleScopePath(form));
    setImpactReadout("[data-studio-style-impact-state]", styleStatePath(form));
    Array.prototype.forEach.call(doc.querySelectorAll("[data-studio-style-impact-panel]"), function (panel) {
      panel.classList.remove("is-live");
      panel.classList.remove("has-change");
    });
  }

  window.__gosx_style_runtime_island_restoreImpact = restoreImpactIsland;
})();
