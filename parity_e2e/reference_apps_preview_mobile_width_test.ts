import { expect, test, type Page } from "@playwright/test";
import { startMuddy } from "./reference_apps_harness";

// Closes the staging-handoff "could not be confirmed" item: does a saved
// phone-only (mobile-breakpoint) layout override actually RENDER inside the
// editor's live preview pane when the operator switches the preview viewport
// to Mobile?
//
// The mechanism under proof: authoring/component_styles.go emits mobile
// overrides behind `@media (max-width: 47.99rem)` (767.84px), and
// workbench_runtime.js's setViewport() narrows [data-studio-preview-frame]
// to the Mobile viewport button's data-studio-viewport-width (default
// 24rem = 384px, shell/options.go NormalizeViewports). Media queries inside
// an iframe respond to the IFRAME's own viewport, so the whole question
// reduces to: at Mobile, does the preview iframe's inner width genuinely
// drop below the 768px breakpoint? This spec measures the frame's OWN
// contentWindow.innerWidth — nothing hardcoded, no CSS re-derivation.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 (shared harness boots muddy).

const WORKBENCH_SELECTOR = "[data-studio-workbench='true']";
const PREVIEW_FRAME_SELECTOR = "[data-studio-preview-frame]";
const MOBILE_BREAKPOINT_PX = 768;

test.describe("@reference-apps preview pane viewport honesty: Mobile really is below the phone-only breakpoint", () => {
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni: switching the preview viewport to Mobile narrows the preview iframe below 768px so phone-only overrides fire; Desktop restores a wider pane", async ({ page, request }) => {
    const server = await startMuddy(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator(WORKBENCH_SELECTOR).first()).toBeAttached({ timeout: 30_000 });
      await page.waitForFunction(() => "GoSXStudioWorkbenchRuntime" in window, null, { timeout: 20_000 });

      const previewFrame = page.locator(PREVIEW_FRAME_SELECTOR).first();
      await expect(previewFrame, "the preview iframe must be attached").toBeAttached({ timeout: 30_000 });
      await hydrateDeferredPreviewFrame(page);
      await previewFrame.scrollIntoViewIfNeeded();
      await expect(previewFrame).toBeVisible();

      const frameInnerWidth = () =>
        page.evaluate(() => {
          const node = document.querySelector("[data-studio-preview-frame]");
          if (!(node instanceof HTMLIFrameElement) || !node.contentWindow) return -1;
          return node.contentWindow.innerWidth;
        });

      // Mobile: the pane must be genuinely narrower than the phone-only
      // media-query breakpoint, or saved mobile overrides can never render
      // in the preview.
      await page.locator('[data-studio-viewport="mobile"]').first().click();
      await expect(page.locator(WORKBENCH_SELECTOR).first()).toHaveAttribute("data-studio-breakpoint", "mobile");
      await expect
        .poll(frameInnerWidth, { message: "the preview iframe's own viewport must drop below the 768px phone breakpoint at Mobile", timeout: 15_000 })
        .toBeLessThan(MOBILE_BREAKPOINT_PX);
      await expect
        .poll(frameInnerWidth, { message: "a collapsed/unloaded frame (width<=0) must not be what passes the Mobile check" })
        .toBeGreaterThan(0);
      const mobileWidth = await frameInnerWidth();

      // Tablet: below the desktop range, still at or under the 1024px
      // tablet media boundary used by overrideBreakpointMedia (63.99rem).
      await page.locator('[data-studio-viewport="tablet"]').first().click();
      await expect
        .poll(frameInnerWidth, { message: "the Tablet preview pane must sit at or under the 1024px tablet boundary", timeout: 15_000 })
        .toBeLessThanOrEqual(1024);

      // Desktop: "desktop" is width:100% of the available pane, so its
      // absolute px depends on the window and open rails (at a 1280px test
      // window the chrome leaves well under 768px — that is expected, and
      // why the operator docs point at "Open public" for a true full-width
      // desktop check). The contract here is relative: Desktop must restore
      // a pane meaningfully wider than the Mobile one, so the operator is
      // never stuck previewing phone rules after switching back.
      await page.locator('[data-studio-viewport="desktop"]').first().click();
      await expect
        .poll(frameInnerWidth, { message: "the Desktop preview pane must be wider than the Mobile pane", timeout: 15_000 })
        .toBeGreaterThan(mobileWidth);
    } finally {
      await server.stop();
    }
  });
});

// hydrateDeferredPreviewFrame sets the iframe's src from its
// data-studio-preview-src/host data-gosx-studio-preview-url fallback if the
// server rendered it deferred (no src yet) — mirrors
// reference_apps_canvas_first_test.ts's identical helper.
async function hydrateDeferredPreviewFrame(page: Page) {
  await page.evaluate(() => {
    const node = document.querySelector("[data-studio-preview-frame]");
    if (!(node instanceof HTMLIFrameElement)) return;
    if (node.getAttribute("src")) return;
    const container = node.closest("[data-gosx-studio-preview]");
    const nextURL = node.getAttribute("data-studio-preview-src") ||
      container?.getAttribute("data-gosx-studio-preview-url") ||
      "/";
    node.setAttribute("src", nextURL);
  });
}
