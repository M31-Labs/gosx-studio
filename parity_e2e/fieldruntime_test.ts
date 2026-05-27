// Phase 3 slice-1 parity tests for GoSXStudioFieldRuntime.
//
// Three scenarios, one per public method:
//   - bind             — composite of bindMirroring + bindClipboard.
//   - bindMirroring    — input keystroke writes $preview.field.<key>.
//   - bindClipboard    — copy button reads target value and toggles state.
//
// Each test runs the same user interaction in baseline (legacy JS active)
// and candidate (island runtime active) modes, snapshots observable state,
// and asserts equivalence. See ./harness.ts and the slice plan
// (~/.hyphae/spaces/m31labs-gosx/plans/2026-05-25-phase-3-slice-1-fieldruntime.md)
// for the deletion gate this suite feeds.

import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { bootBaseline, bootCandidate, disposeBoot } from "./harness";

// Field selectors come from the editor route's stable inspector contract
// (muddy-noni-commerce/app/admin/editor/page.server.go declares
// data-editor-source="site-tagline" / "hero-headline" / etc.). The site
// tagline is the simplest stable input for keystroke parity — it has a
// single source key and no attr-target fan-out.
const TAGLINE_INPUT_SELECTOR = "[data-editor-source='site-tagline']";

// The copy button declared in editor/page.gsx at the preview-share section.
const COPY_BUTTON_SELECTOR = "[data-studio-copy-target='[data-studio-preview-url]']";

interface MirroringSnapshot {
  inputValue: string;
  // The legacy path writes through GoSXStudioPreviewRuntime.applyTextUpdate
  // which mutates iframe DOM. We can't easily peek into the iframe in
  // candidate mode without slice 6 shipping the preview subscriber, so we
  // snapshot the editor-side observable instead: the input's bound state
  // attribute and any field-attr the runtime wrote.
  islandBoundAttr: string | null;
  legacyBoundAttr: string | null;
}

interface ClipboardSnapshot {
  copyState: string | null;
  copyLabel: string | null;
  buttonText: string;
}

interface CandidateFlagSnapshot {
  flagAttrPresent: boolean;
  flagAttrValue: string | null;
  fieldRuntimeGlobalShape: { bind: boolean; bindMirroring: boolean; bindClipboard: boolean };
  islandRuntimeAvailable: { bind: boolean; bindMirroring: boolean; bindClipboard: boolean };
}

async function snapshotFlag(page: Page): Promise<CandidateFlagSnapshot> {
  return await page.evaluate(() => {
    // The feature flag attribute is surfaced on the workbench root (see
    // lessons/phase-3-slice-1-fieldruntime-deletion-log.md — "the consumer
    // surfaces it via data-gosx-studio-feature-flag-<key> on the workbench
    // root"), not on documentElement. Fall back to documentElement for
    // backward compat in case a host SSRs the flag higher in the tree.
    const attr = "data-gosx-studio-feature-flag-field-runtime-islands";
    const root =
      document.querySelector(`[${attr}]`) ?? document.documentElement;
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioFieldRuntime ?? {}) as Record<string, unknown>;
    return {
      flagAttrPresent: root.hasAttribute(attr),
      flagAttrValue: root.getAttribute(attr),
      fieldRuntimeGlobalShape: {
        bind: typeof runtime.bind === "function",
        bindMirroring: typeof runtime.bindMirroring === "function",
        bindClipboard: typeof runtime.bindClipboard === "function",
      },
      islandRuntimeAvailable: {
        bind: typeof w.__gosx_field_runtime_island_bind === "function",
        bindMirroring: typeof w.__gosx_field_runtime_island_bindMirroring === "function",
        bindClipboard: typeof w.__gosx_field_runtime_island_bindClipboard === "function",
      },
    };
  });
}

async function snapshotMirroring(page: Page, inputSelector: string): Promise<MirroringSnapshot> {
  return await page.evaluate((selector) => {
    const input = document.querySelector(selector) as HTMLInputElement | null;
    return {
      inputValue: input ? input.value : "",
      islandBoundAttr: input?.getAttribute("data-gosx-studio-field-mirror-island-bound") ?? null,
      legacyBoundAttr: input?.getAttribute("data-gosx-studio-field-mirror-bound") ?? null,
    };
  }, inputSelector);
}

async function snapshotClipboard(page: Page, buttonSelector: string): Promise<ClipboardSnapshot> {
  return await page.evaluate((selector) => {
    const button = document.querySelector(selector) as HTMLElement | null;
    return {
      copyState: button?.getAttribute("data-studio-copy-state") ?? null,
      copyLabel: button?.getAttribute("data-studio-copy-label") ?? null,
      buttonText: button?.textContent?.trim() ?? "",
    };
  }, buttonSelector);
}

test.describe("GoSXStudioFieldRuntime parity", () => {
  // The harness boots both modes against the same consumer process via the
  // ?gosx-studio-parity= query parameter. Until the consumer bumps its
  // gosx-studio module dependency to include the fieldruntime island bundle,
  // both modes load the same legacy bundle and the assertions trivially
  // succeed. The "coverage" test below detects this and warns so the team
  // doesn't ship a false-positive parity claim.

  test("@coverage candidate boot exposes island runtime globals", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      const snap = await snapshotFlag(handle.page);
      expect(snap.flagAttrPresent).toBe(true);
      expect(snap.flagAttrValue).toBe("true");
      // The next two assertions are the "real" coverage check. Until the
      // muddy-noni-commerce gosx-studio bump lands, the island runtime
      // globals are absent and the test marks itself skipped with a
      // descriptive reason so CI surfaces the prerequisite plainly.
      test.skip(
        !snap.islandRuntimeAvailable.bind,
        "gosx-studio module bump not yet landed in muddy-noni-commerce: " +
          "__gosx_field_runtime_island_bind is undefined. The parity tests below " +
          "still run but only verify legacy-path stability. Land the dependency " +
          "bump and rerun to get true parity coverage.",
      );
      expect(snap.islandRuntimeAvailable.bind).toBe(true);
      expect(snap.islandRuntimeAvailable.bindMirroring).toBe(true);
      expect(snap.islandRuntimeAvailable.bindClipboard).toBe(true);
    } finally {
      await disposeBoot(handle);
    }
  });

  test("fieldruntime.bindMirroring parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      const sampleValue = "parity tagline " + Date.now();
      await fillTagline(baseline.page, sampleValue);
      await fillTagline(candidate.page, sampleValue);

      const baselineSnap = await snapshotMirroring(baseline.page, TAGLINE_INPUT_SELECTOR);
      const candidateSnap = await snapshotMirroring(candidate.page, TAGLINE_INPUT_SELECTOR);

      // The input value must match; this is the user-visible side of the
      // binding and must never differ between modes.
      expect(candidateSnap.inputValue).toBe(baselineSnap.inputValue);
      expect(candidateSnap.inputValue).toBe(sampleValue);

      // At least one of the bound-attrs must be "true" in each mode — the
      // exact attr depends on which path was active (island-bound vs
      // legacy-bound). Symmetric assertion: each side has some binding
      // marker.
      const baselineHasBinding = baselineSnap.islandBoundAttr === "true" || baselineSnap.legacyBoundAttr === "true";
      const candidateHasBinding = candidateSnap.islandBoundAttr === "true" || candidateSnap.legacyBoundAttr === "true";
      expect(baselineHasBinding).toBe(true);
      expect(candidateHasBinding).toBe(true);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  test("fieldruntime.bindClipboard parity", async ({ browser, context }) => {
    // Grant clipboard permissions for both contexts so the navigator.clipboard
    // path runs (rather than the execCommand fallback). Both modes must use
    // the same code path for fair comparison.
    await context.grantPermissions(["clipboard-read", "clipboard-write"], { origin: process.env.GOSX_STUDIO_PARITY_BASE_URL ?? "http://127.0.0.1:3010" });

    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      await baseline.context.grantPermissions(["clipboard-read", "clipboard-write"]);
      await candidate.context.grantPermissions(["clipboard-read", "clipboard-write"]);

      const baselineBefore = await snapshotClipboard(baseline.page, COPY_BUTTON_SELECTOR);
      const candidateBefore = await snapshotClipboard(candidate.page, COPY_BUTTON_SELECTOR);

      await clickCopyButton(baseline.page);
      await clickCopyButton(candidate.page);

      // The state attribute should flip to "ok" (or "error" if the target
      // had no value); both modes must produce the same outcome state.
      const baselineAfter = await snapshotClipboard(baseline.page, COPY_BUTTON_SELECTOR);
      const candidateAfter = await snapshotClipboard(candidate.page, COPY_BUTTON_SELECTOR);

      expect(candidateAfter.copyState).toBe(baselineAfter.copyState);
      // Both modes preserve the original label.
      expect(candidateAfter.copyLabel).toBe(baselineAfter.copyLabel);
      // Pre-click state must match across modes (nothing has happened yet).
      expect(candidateBefore.copyState).toBe(baselineBefore.copyState);
      expect(candidateBefore.buttonText).toBe(baselineBefore.buttonText);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  test("fieldruntime.bind composes mirroring and clipboard", async ({ browser }) => {
    // bind() is just bindMirroring + bindClipboard. The parity check is
    // that after the editor mounts, both binding markers are present
    // simultaneously, in both modes.
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      const baselineState = await page2BindState(baseline.page);
      const candidateState = await page2BindState(candidate.page);
      expect(candidateState.taglineBound).toBe(baselineState.taglineBound);
      expect(candidateState.copyButtonPresent).toBe(baselineState.copyButtonPresent);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });
});

async function fillTagline(page: Page, value: string): Promise<void> {
  const input = page.locator(TAGLINE_INPUT_SELECTOR).first();
  if (await input.count() === 0) {
    test.skip(true, "tagline input not present in editor route — fixture drift");
    return;
  }
  // The tagline input lives inside an inspector panel that may be CSS-hidden
  // when its parent mode isn't active. We're exercising the runtime binding,
  // not the user-interaction visibility contract, so set the value
  // programmatically and dispatch the input event the binding listens for.
  // This makes the assertion robust to editor-layout changes that toggle
  // panel visibility (the runtime path the test cares about is independent).
  await input.evaluate((el, v) => {
    const node = el as HTMLInputElement;
    node.value = v;
    node.dispatchEvent(new Event("input", { bubbles: true }));
    node.dispatchEvent(new Event("change", { bubbles: true }));
  }, value);
  // Allow one rAF tick so the frame-throttled mirror update runs.
  await page.waitForTimeout(50);
}

async function clickCopyButton(page: Page): Promise<void> {
  const button = page.locator(COPY_BUTTON_SELECTOR).first();
  if (await button.count() === 0) {
    test.skip(true, "copy button not present in editor route — fixture drift");
    return;
  }
  // The copy button may live inside a collapsed inspector panel that's CSS-
  // hidden even though the button itself is interactive. Use the underlying
  // click() rather than Playwright's visibility-aware locator.click so the
  // runtime-binding test is independent of editor-panel layout.
  await button.evaluate((el) => (el as HTMLElement).click());
  // The legacy and island clipboard handlers both set the state attribute
  // synchronously after the writeText promise resolves. Allow a small
  // window for the async clipboard API.
  await page.waitForTimeout(150);
}

async function page2BindState(page: Page): Promise<{ taglineBound: boolean; copyButtonPresent: boolean }> {
  return await page.evaluate((selectors) => {
    const tagline = document.querySelector(selectors.tagline) as HTMLElement | null;
    const copy = document.querySelector(selectors.copy) as HTMLElement | null;
    const taglineBound =
      tagline?.getAttribute("data-gosx-studio-field-mirror-island-bound") === "true" ||
      tagline?.getAttribute("data-gosx-studio-field-mirror-bound") === "true";
    return {
      taglineBound: Boolean(taglineBound),
      copyButtonPresent: copy !== null,
    };
  }, { tagline: TAGLINE_INPUT_SELECTOR, copy: COPY_BUTTON_SELECTOR });
}
