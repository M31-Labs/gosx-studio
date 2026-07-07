// Phase 3 slice-3 parity tests for GoSXStudioBrandRuntime.
//
// Three scenarios:
//   - @coverage         — confirms the candidate boot exposes both island
//                         globals. Self-skips with a descriptive reason when
//                         the gosx-studio module bump hasn't landed in the
//                         consumer yet (same precondition as slices 1 and 2).
//   - bindLogo parity   — types into the brand-logo URL/width/offset inputs,
//                         asserts the preview CSS custom properties
//                         (--brand-logo-width / --brand-logo-offset-x /
//                         --brand-logo-offset-y) and readout text agree
//                         between baseline and candidate modes.
//   - updateHeaderLogo  — direct call into window.GoSXStudioBrandRuntime
//     parity              .updateHeaderLogo with a known payload; assert
//                         the storefront preview iframe's .brand element
//                         mutations agree between modes. This is the
//                         transitional iframe-delegation path documented in
//                         ADR 0008
//                         (~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md) —
//                         slice 6 swaps the mechanism to a
//                         $preview.brand.headerLogo shared signal write
//                         without revisiting brand ownership.
//
// Each test runs the same user interaction in baseline (legacy JS active)
// and candidate (island runtime active) modes, snapshots observable state,
// and asserts equivalence. See ./harness.ts and the slice plan
// (~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-3-brandruntime.md)
// for the deletion gate this suite feeds.
//
// Per the parity matrix (gosx-studio-runtime-parity-matrix.md), BrandRuntime
// has exactly two methods (bindLogo, updateHeaderLogo). The slice plan calls
// for one parity sub-test per method plus the @coverage prerequisite gate;
// see Section C of the plan.

import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { bootBaseline, bootCandidate, disposeBoot } from "./harness";

// Editor brand-logo authoring surface selectors. These come from the editor
// route's stable structure — see muddy-noni-commerce/app/admin/editor/page.gsx
// around line 1585 (data-editor-brand-preview / data-editor-brand-logo /
// data-editor-logo-* inputs).
const BRAND_PREVIEW_SELECTOR = "[data-editor-brand-preview]";
const BRAND_LOGO_SELECTOR = "[data-editor-brand-logo]";
const BRAND_URL_INPUT_SELECTOR = "[data-editor-logo-url]";
const BRAND_WIDTH_INPUT_SELECTOR = "[data-editor-logo-width]";
const BRAND_OFFSET_X_INPUT_SELECTOR = "[data-editor-logo-offset-x]";
const BRAND_OFFSET_Y_INPUT_SELECTOR = "[data-editor-logo-offset-y]";
const BRAND_READOUT_SELECTOR = "[data-editor-logo-readout]";

interface CandidateFlagSnapshot {
  flagAttrPresent: boolean;
  flagAttrValue: string | null;
  brandRuntimeGlobalShape: { bindLogo: boolean; updateHeaderLogo: boolean };
  islandRuntimeAvailable: { bindLogo: boolean; updateHeaderLogo: boolean };
}

async function snapshotFlag(page: Page): Promise<CandidateFlagSnapshot> {
  return await page.evaluate(() => {
    // The feature flag attribute is surfaced on the workbench root (see
    // lessons/phase-3-slice-1-fieldruntime-deletion-log.md), not on
    // documentElement. Fall back to documentElement for backward compat.
    const attr = "data-gosx-studio-feature-flag-brand-runtime-islands";
    const root =
      document.querySelector(`[${attr}]`) ?? document.documentElement;
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBrandRuntime ?? {}) as Record<string, unknown>;
    return {
      flagAttrPresent: root.hasAttribute(attr),
      flagAttrValue: root.getAttribute(attr),
      brandRuntimeGlobalShape: {
        bindLogo: typeof runtime.bindLogo === "function",
        updateHeaderLogo: typeof runtime.updateHeaderLogo === "function",
      },
      islandRuntimeAvailable: {
        bindLogo: typeof w.__gosx_brand_runtime_island_bindLogo === "function",
        updateHeaderLogo: typeof w.__gosx_brand_runtime_island_updateHeaderLogo === "function",
      },
    };
  });
}

interface BrandLogoSnapshot {
  // Preview element CSS custom-property writes.
  previewLogoWidth: string | null;
  previewLogoOffsetX: string | null;
  previewLogoOffsetY: string | null;
  previewLogoGridSize: string | null;
  // Logo element src/alt.
  logoSrc: string | null;
  logoAlt: string | null;
  // Readout text.
  readoutText: string;
  // The URL input's current value (so we can assert the input actually took
  // the keystroke before comparing rendered effects).
  urlValue: string;
}

async function snapshotBrandLogo(page: Page): Promise<BrandLogoSnapshot> {
  return await page.evaluate((sel) => {
    const preview = document.querySelector(sel.preview) as HTMLElement | null;
    const logo = document.querySelector(sel.logo) as HTMLElement | null;
    const readout = document.querySelector(sel.readout) as HTMLElement | null;
    const urlInput = document.querySelector(sel.urlInput) as HTMLInputElement | null;
    const previewStyle = preview ? preview.style : null;
    return {
      previewLogoWidth: previewStyle ? previewStyle.getPropertyValue("--brand-logo-width") : null,
      previewLogoOffsetX: previewStyle ? previewStyle.getPropertyValue("--brand-logo-offset-x") : null,
      previewLogoOffsetY: previewStyle ? previewStyle.getPropertyValue("--brand-logo-offset-y") : null,
      previewLogoGridSize: previewStyle ? previewStyle.getPropertyValue("--editor-logo-grid-size") : null,
      logoSrc: logo ? logo.getAttribute("src") : null,
      logoAlt: logo ? logo.getAttribute("alt") : null,
      readoutText: readout?.textContent?.trim() ?? "",
      urlValue: urlInput ? urlInput.value : "",
    };
  }, {
    preview: BRAND_PREVIEW_SELECTOR,
    logo: BRAND_LOGO_SELECTOR,
    readout: BRAND_READOUT_SELECTOR,
    urlInput: BRAND_URL_INPUT_SELECTOR,
  });
}

interface HeaderLogoSnapshot {
  // The iframe-side preview-document mutations the legacy and island paths
  // both perform via GoSXStudioPreviewRuntime.updateHeaderLogo. Until slice 6
  // ships the $preview.brand.headerLogo shared-signal portal, we read the
  // preview iframe's .brand DOM directly.
  brandWithLogoClass: boolean;
  brandLogoSrc: string | null;
  brandLogoAlt: string | null;
  brandLogoHidden: boolean;
  brandLogoWidth: string | null;
  brandLogoOffsetX: string | null;
  brandLogoOffsetY: string | null;
  // Whether the iframe was actually reachable. When the preview iframe is
  // cross-origin in CI, the assertion below skips the iframe-DOM check and
  // falls back to verifying the brandruntime delegation took place at all
  // (by reading the bridge signal we wrote alongside the delegation).
  iframeReachable: boolean;
  // The $brand.headerLogo signal value the island writes alongside the
  // PreviewRuntime delegation. This is observable even when the iframe is
  // cross-origin.
  signalSnapshot: {
    url: string;
    width: string;
    x: string;
    y: string;
    alt: string;
  } | null;
}

async function snapshotHeaderLogo(page: Page): Promise<HeaderLogoSnapshot> {
  return await page.evaluate(() => {
    const w = window as unknown as Record<string, unknown>;
    // Try to reach into every iframe's preview document (matches legacy
    // editorDocuments() behavior in the legacy bundle).
    const frames = Array.prototype.slice.call(document.querySelectorAll("iframe")) as HTMLIFrameElement[];
    let target: { doc: Document; brand: Element } | null = null;
    let reachable = false;
    for (const frame of frames) {
      let doc: Document | null = null;
      try {
        doc = frame.contentDocument;
      } catch (_) {
        // Cross-origin — skip.
        continue;
      }
      if (!doc) continue;
      reachable = true;
      const brand = doc.querySelector(".brand");
      if (brand) {
        target = { doc, brand };
        break;
      }
    }
    let logoEl: HTMLImageElement | null = null;
    let brandStyle: CSSStyleDeclaration | null = null;
    if (target) {
      logoEl = target.brand.querySelector(".brand__logo") as HTMLImageElement | null;
      brandStyle = (target.brand as HTMLElement).style;
    }
    // Read the $brand.headerLogo signal the island writes alongside the
    // PreviewRuntime delegation, if the bridge exposes a getter. The signal
    // is editor-side observable so it works even when the iframe is cross-
    // origin.
    let signalSnapshot: HeaderLogoSnapshot["signalSnapshot"] = null;
    try {
      const getter = w.__gosx_get_shared_signal_json as ((name: string) => string | null) | undefined;
      if (typeof getter === "function") {
        const raw = getter("$brand.headerLogo");
        if (raw) {
          const parsed = JSON.parse(raw) as Partial<NonNullable<HeaderLogoSnapshot["signalSnapshot"]>>;
          signalSnapshot = {
            url: parsed.url ?? "",
            width: parsed.width ?? "",
            x: parsed.x ?? "",
            y: parsed.y ?? "",
            alt: parsed.alt ?? "",
          };
        }
      }
    } catch (_) {
      // Bridge not booted yet; leave signalSnapshot null.
    }
    return {
      brandWithLogoClass: target ? target.brand.classList.contains("brand--with-logo") : false,
      brandLogoSrc: logoEl ? logoEl.getAttribute("src") : null,
      brandLogoAlt: logoEl ? logoEl.getAttribute("alt") : null,
      brandLogoHidden: logoEl ? Boolean(logoEl.hidden) : false,
      brandLogoWidth: brandStyle ? brandStyle.getPropertyValue("--brand-logo-width") : null,
      brandLogoOffsetX: brandStyle ? brandStyle.getPropertyValue("--brand-logo-offset-x") : null,
      brandLogoOffsetY: brandStyle ? brandStyle.getPropertyValue("--brand-logo-offset-y") : null,
      iframeReachable: reachable,
      signalSnapshot,
    };
  });
}

test.describe("GoSXStudioBrandRuntime parity", () => {
  // The harness boots both modes against the same consumer process via the
  // ?gosx-studio-parity= query parameter. Until the consumer bumps its
  // gosx-studio module dependency to include the brandruntime island
  // bundle, both modes load the same legacy bundle and the assertions
  // trivially succeed. The "@coverage" test below detects this and warns
  // so the team doesn't ship a false-positive parity claim.

  test("@coverage candidate boot exposes brand island runtime globals", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      const snap = await snapshotFlag(handle.page);
      expect(snap.flagAttrPresent).toBe(true);
      expect(snap.flagAttrValue).toBe("true");
      // The next assertion is the "real" coverage check. Until the
      // muddy-noni-commerce gosx-studio bump lands, the island runtime
      // globals are absent and the test marks itself skipped with a
      // descriptive reason so CI surfaces the prerequisite plainly.
      test.skip(
        !snap.islandRuntimeAvailable.bindLogo || !snap.islandRuntimeAvailable.updateHeaderLogo,
        "gosx-studio module bump not yet landed in muddy-noni-commerce: " +
          "__gosx_brand_runtime_island_bindLogo or __gosx_brand_runtime_island_updateHeaderLogo " +
          "is undefined. The parity tests below still run but only verify " +
          "legacy-path stability. Land the dependency bump and rerun to get " +
          "true parity coverage.",
      );
      expect(snap.islandRuntimeAvailable.bindLogo).toBe(true);
      expect(snap.islandRuntimeAvailable.updateHeaderLogo).toBe(true);
      expect(snap.brandRuntimeGlobalShape.bindLogo).toBe(true);
      expect(snap.brandRuntimeGlobalShape.updateHeaderLogo).toBe(true);
    } finally {
      await disposeBoot(handle);
    }
  });

  // Sentinel for v0.5.0 BridgeShim auto-mount restoration (all 7 islands).
  //
  // The other tests in this file explicitly drive the runtime — they type
  // into inputs (relying on .bindLogo having attached listeners at boot)
  // or call runtime methods directly via callRuntime. Both paths silently
  // still pass if the binding happens for an unrelated reason (e.g., the
  // host calls .bindLogo manually). This sentinel verifies the v0.5.0
  // contract end-to-end: navigate the editor route in candidate mode, do
  // NOTHING else, and assert that the per-slice BridgeShim's autoMount
  // IIFE wrote its idempotency marker on documentElement for every island
  // that owns one. If any island's autoMount silently regresses, this is
  // the first test to fail — every other test would still pass because
  // they invoke the runtime methods themselves. Pre-v0.5.0 (post legacy-
  // bundle deletion), all six markers were null because the deleted
  // studio-engines.js bundle's init() was the only thing that invoked the
  // boot binds; v0.4.1 restored fieldruntime, v0.5.0 restored the rest.
  //
  // The island-side bound-attr (e.g., data-gosx-studio-brand-logo-island-
  // bound) is NOT asserted here because individual islands' bind() early-
  // return when required DOM fixtures aren't present in the editor route
  // (e.g., the brand-logo URL input lives in an inspector panel that's
  // not always rendered). The shim's autoMount marker is the right
  // observable — it proves the BridgeShim's auto-mount IIFE ran and
  // invoked the runtime method, regardless of what the island binder did
  // with the call.
  test("BridgeShim auto-mount sentinel: every island marks documentElement on boot", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      // Wait a tick past assertEditorRoute's GoSXStudioFieldRuntime
      // presence-check so every slice's autoMount IIFE has run (each
      // either fires inline or on the already-dispatched DOMContentLoaded
      // event).
      await handle.page.waitForTimeout(100);
      const markers = await handle.page.evaluate(() => ({
        field: document.documentElement.getAttribute("data-gosx-studio-field-runtime-auto-mounted"),
        selection: document.documentElement.getAttribute("data-gosx-studio-selection-runtime-auto-mounted"),
        brand: document.documentElement.getAttribute("data-gosx-studio-brand-runtime-auto-mounted"),
        blocklayout: document.documentElement.getAttribute("data-gosx-studio-blocklayout-runtime-auto-mounted"),
        style: document.documentElement.getAttribute("data-gosx-studio-style-runtime-auto-mounted"),
        preview: document.documentElement.getAttribute("data-gosx-studio-preview-runtime-auto-mounted"),
        workbench: document.documentElement.getAttribute("data-gosx-studio-workbench-runtime-auto-mounted"),
      }));
      // Each marker is set unconditionally by the autoMount IIFE before it
      // dispatches the runtime call (see <slice>/runtime.go bridgeShimJS).
      // A null marker means the IIFE either never ran or threw before the
      // setAttribute — both regressions.
      expect(markers.field).toBe("true");
      expect(markers.selection).toBe("true");
      expect(markers.brand).toBe("true");
      expect(markers.blocklayout).toBe("true");
      expect(markers.style).toBe("true");
      expect(markers.preview).toBe("true");
      expect(markers.workbench).toBe("true");
    } finally {
      await disposeBoot(handle);
    }
  });

  // Method 1 of 2: bindLogo (editor-side only).
  test("brandruntime.bindLogo parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      await typeBrandLogoFixture(baseline.page);
      await typeBrandLogoFixture(candidate.page);

      const baselineSnap = await snapshotBrandLogo(baseline.page);
      const candidateSnap = await snapshotBrandLogo(candidate.page);

      // Both modes must agree on the URL the input actually holds (sanity
      // check that the typing landed before comparing rendered effects).
      expect(candidateSnap.urlValue).toBe(baselineSnap.urlValue);
      // Preview CSS custom-property writes — the editor-side observable
      // for the bindLogo binding.
      expect(candidateSnap.previewLogoWidth).toBe(baselineSnap.previewLogoWidth);
      expect(candidateSnap.previewLogoOffsetX).toBe(baselineSnap.previewLogoOffsetX);
      expect(candidateSnap.previewLogoOffsetY).toBe(baselineSnap.previewLogoOffsetY);
      expect(candidateSnap.previewLogoGridSize).toBe(baselineSnap.previewLogoGridSize);
      // Logo element src/alt — written on every update.
      expect(candidateSnap.logoSrc).toBe(baselineSnap.logoSrc);
      expect(candidateSnap.logoAlt).toBe(baselineSnap.logoAlt);
      // Readout text — the human-visible authoring feedback.
      expect(candidateSnap.readoutText).toBe(baselineSnap.readoutText);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 2 of 2: updateHeaderLogo (transitional iframe delegation).
  // Until slice 6 swaps to the $preview.brand.headerLogo shared signal,
  // both legacy and candidate paths call into
  // window.GoSXStudioPreviewRuntime.updateHeaderLogo. The candidate path
  // just owns the call from the brand island instead of from the legacy
  // updateHeaderLogo function in the legacy bundle. The observable
  // behavior — iframe .brand DOM mutations — must be identical.
  test("brandruntime.updateHeaderLogo parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      const payload = {
        url: "https://example.com/logo.png",
        width: "128",
        x: "16",
        y: "8",
        alt: "Parity logo",
      };
      await callUpdateHeaderLogo(baseline.page, payload);
      await callUpdateHeaderLogo(candidate.page, payload);
      // Allow rAF / iframe-style settling.
      await baseline.page.waitForTimeout(50);
      await candidate.page.waitForTimeout(50);

      const baselineSnap = await snapshotHeaderLogo(baseline.page);
      const candidateSnap = await snapshotHeaderLogo(candidate.page);

      // When the preview iframe is reachable, the .brand DOM mutations
      // must match exactly between baseline and candidate. This is the
      // ADR 0008 transitional contract — same iframe-crossing mechanism,
      // just owned by the brand island in candidate mode.
      if (baselineSnap.iframeReachable && candidateSnap.iframeReachable) {
        expect(candidateSnap.brandWithLogoClass).toBe(baselineSnap.brandWithLogoClass);
        expect(candidateSnap.brandLogoSrc).toBe(baselineSnap.brandLogoSrc);
        expect(candidateSnap.brandLogoAlt).toBe(baselineSnap.brandLogoAlt);
        expect(candidateSnap.brandLogoHidden).toBe(baselineSnap.brandLogoHidden);
        expect(candidateSnap.brandLogoWidth).toBe(baselineSnap.brandLogoWidth);
        expect(candidateSnap.brandLogoOffsetX).toBe(baselineSnap.brandLogoOffsetX);
        expect(candidateSnap.brandLogoOffsetY).toBe(baselineSnap.brandLogoOffsetY);
        return;
      }
      // Fallback: when the iframe is cross-origin or absent in CI, fall
      // back to verifying the brandruntime delegation took place at all by
      // comparing the $brand.headerLogo signal. (Legacy path doesn't write
      // this signal — the legacy updateHeaderLogo is a pure PreviewRuntime
      // delegate — so on the legacy side this snapshot will be null. We
      // skip the full equality assertion in that case and only verify the
      // candidate side wrote the signal.)
      if (candidateSnap.signalSnapshot) {
        expect(candidateSnap.signalSnapshot.url).toBe(payload.url);
        expect(candidateSnap.signalSnapshot.alt).toBe(payload.alt);
      } else {
        test.skip(
          true,
          "iframe not reachable and $brand.headerLogo signal not yet readable; " +
            "deletion gate cannot certify updateHeaderLogo parity until either " +
            "the iframe is same-origin in CI or the shared-signal getter ships. " +
            "Slice 6's PreviewRuntime burn-down resolves this by replacing the " +
            "iframe mutation with a $preview.brand.headerLogo subscriber.",
        );
      }
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });
});

// The brand-logo URL/width/offset inputs live in the right-rail "Brand"
// inspector panel (data-studio-mode-panel="brand"), which is not the active
// panel by default — the editor boots into the "Home" mode
// (see editorStudioShellModes in muddy-noni-commerce/app/admin/editor/
// page.server.go). The inputs are present in the DOM but `hidden` (the
// WorkbenchRuntime island sets `panel.hidden = true` on every
// [data-studio-mode-panel] that isn't the active mode — see
// workbenchruntime/island_runtime.js setModeIsland). Switch to the Brand
// tab first so the fixture actually exercises bindLogo instead of timing
// out on a hidden input. Workbench tab-switching is fully islands-based
// already (see the "BridgeShim auto-mount sentinel" test above — the
// workbench marker is unconditional), so clicking this control behaves
// identically in baseline and candidate modes and doesn't affect what this
// test is actually asserting (brandruntime.bindLogo parity).
async function activateBrandInspectorPanel(page: Page): Promise<void> {
  const brandTab = page.locator('[data-studio-mode-control="brand"]').first();
  if ((await brandTab.count()) === 0) return;
  await brandTab.click();
  // Wait for the panel to actually become visible before returning; the
  // WorkbenchRuntime island toggles `hidden` synchronously on click, but
  // give it a moment in case a future revision defers the toggle (e.g. to
  // a rAF or transition-end handler).
  await page
    .locator(BRAND_URL_INPUT_SELECTOR)
    .first()
    .waitFor({ state: "visible", timeout: 2_000 })
    .catch(() => {
      // If it's still not visible, let typeBrandLogoFixture's own
      // count()/fill() surface the failure with its usual diagnostics.
    });
}

// Inside the Brand panel itself, the URL/alt fields and the width/offset
// fields are further split into their own pure-CSS radio-tab groups
// (`studio-brand-panel__group-*` in muddy-noni-commerce/app/admin/editor —
// see the `:checked ~ [data-studio-brand-group-slot=...]` sibling-selector
// rules in public/site.css). "Logo" (url/alt) is selected by default;
// "Placement" (width/offsetX/offsetY) is not, so the width/offset inputs
// are visually hidden until the "Placement" tab label is clicked. Checking
// the radio via its <label> mirrors real authoring — a user must click the
// "Placement" tab to reach those fields too.
async function activateBrandGroup(page: Page, group: string): Promise<void> {
  const groupTab = page.locator(`[data-studio-brand-group-label="${group}"]`).first();
  if ((await groupTab.count()) === 0) return;
  await groupTab.click();
}

async function typeBrandLogoFixture(page: Page): Promise<void> {
  const url = page.locator(BRAND_URL_INPUT_SELECTOR).first();
  if ((await url.count()) === 0) {
    test.skip(true, "no brand logo URL input present in editor route — fixture drift");
    return;
  }
  await activateBrandInspectorPanel(page);
  // "Logo" is the default-selected group, but assert it explicitly in case
  // a prior scenario in the same page left a different group selected.
  await activateBrandGroup(page, "logo");
  // Fill the URL, width, and offsets with a deterministic fixture. fill()
  // dispatches input + change events the binding listens for.
  await url.fill("https://example.com/parity-logo.png");
  await activateBrandGroup(page, "placement");
  const width = page.locator(BRAND_WIDTH_INPUT_SELECTOR).first();
  if ((await width.count()) > 0) await width.fill("128");
  const offsetX = page.locator(BRAND_OFFSET_X_INPUT_SELECTOR).first();
  if ((await offsetX.count()) > 0) await offsetX.fill("16");
  const offsetY = page.locator(BRAND_OFFSET_Y_INPUT_SELECTOR).first();
  if ((await offsetY.count()) > 0) await offsetY.fill("8");
  // Allow the rAF-throttled update to flush.
  await page.waitForTimeout(50);
}

async function callUpdateHeaderLogo(
  page: Page,
  payload: { url: string; width: string; x: string; y: string; alt: string },
): Promise<void> {
  await page.evaluate((p) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBrandRuntime ?? {}) as { updateHeaderLogo?: (payload: unknown) => void };
    if (typeof runtime.updateHeaderLogo === "function") {
      runtime.updateHeaderLogo(p);
    }
  }, payload);
}
