import { expect, test } from "@playwright/test";
import { startMuddy } from "./reference_apps_harness";

// This spec is the "clutter regression" test called for by the Noni
// declutter slice: the owner's complaint was that the editor sidebars
// stacked every panel (Look, Brand, Publish, Advanced, Navigation,
// Checkout, the flow field editor, the shared-components palette...)
// simultaneously regardless of the selected mode tab, because
// workbenchruntime's Bundle() (the JS that actually hides/shows
// [data-studio-mode-panel] elements based on the active mode) was never
// linked into the editor page — the mode tabs were decorative buttons with
// no bound behavior. app/admin/editor/page.server.go now inlines
// workbenchruntime.Bundle() and tags every previously-unconditional panel
// with the mode it belongs to; this spec proves both the "clean default"
// composition AND (the owner's related "interaction fails after an edit"
// complaint) that a save's full-page reload no longer discards the
// operator's mode, selection, and scroll position.
test.describe("@reference-apps Muddy/Noni editor declutter", () => {
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("default (Home) mode composes exactly three clean zones", async ({ page, request }) => {
    const server = await startMuddy(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const form = page.locator("[data-studio-workbench='true']").first();
      await expect(form).toBeAttached();
      await expect(form).toHaveAttribute("data-studio-mode", "home");

      // The workbench runtime that makes mode tabs actually hide/show
      // panels must be bound (not just present as inert markup).
      await page.waitForFunction(() => "GoSXStudioWorkbenchRuntime" in window, null, { timeout: 20_000 });

      // Zone 1 (left): pages/layers navigation is visible.
      await expect(page.locator("[data-studio-site-navigator-panel='true']").first()).toBeVisible();
      // Zone 2 (center): the rendered page canvas is visible.
      await expect(page.locator("[data-gosx-studio-page-canvas='true']").first()).toBeVisible();
      // Zone 3 (right): the Home contextual inspector is visible.
      await expect(page.locator("[data-studio-home-inspector-panel='true']").first()).toBeVisible();

      // Everything that is NOT part of the default Home composition must
      // actually be hidden, not merely mode-tagged in markup — this is the
      // regression: previously every one of these rendered unconditionally.
      for (const selector of [
        "[data-studio-look-panel='true']",
        "[data-studio-brand-panel='true']", // folds into Look; hidden by default too
        "[data-studio-publish-panel='true']",
        "[data-studio-advanced-panel='true']",
        "[data-studio-navigation-panel='true']",
        "[data-studio-checkout-panel='true']",
        "[data-studio-flow-field-editor='true']",
        "[data-studio-shared-components-panel='true']",
      ]) {
        await expect(page.locator(selector).first(), `${selector} must be hidden in the default Home mode`).toBeHidden();
      }

      // No separate "Brand" mode tab.
      await expect(page.locator('[data-studio-mode-control="brand"]')).toHaveCount(0);

      // Switching to Advanced reveals Advanced-homed panels (navigation,
      // checkout, flow field editor, shared components) and hides Home's.
      await page.locator('[data-studio-mode-control="advanced"]').first().click();
      await expect(form).toHaveAttribute("data-studio-mode", "advanced");
      await expect(page.locator("[data-studio-navigation-panel='true']").first()).toBeVisible();
      await expect(page.locator("[data-studio-checkout-panel='true']").first()).toBeVisible();
      await expect(page.locator("[data-studio-home-inspector-panel='true']").first()).toBeHidden();

      // Switching to Look reveals the merged Brand+Look surface.
      await page.locator('[data-studio-mode-control="look"]').first().click();
      await expect(form).toHaveAttribute("data-studio-mode", "look");
      await expect(page.locator("[data-studio-look-panel='true']").first()).toBeVisible();
      await expect(page.locator("[data-studio-brand-panel='true']").first()).toBeVisible();
    } finally {
      await server.stop();
    }
  });

  test("a save's full-page reload preserves the active mode and selection (not just rail widths)", async ({ page, request }) => {
    const server = await startMuddy(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const form = page.locator("[data-studio-workbench='true']").first();
      await page.waitForFunction(() => "GoSXStudioWorkbenchRuntime" in window, null, { timeout: 20_000 });

      // Switch to a non-default mode/panel.
      await page.locator('[data-studio-mode-control="advanced"]').first().click();
      await expect(form).toHaveAttribute("data-studio-mode", "advanced");
      const navigationPanel = page.locator("[data-studio-navigation-panel='true']").first();
      await expect(navigationPanel).toBeVisible();

      // Select an object. Canvas click-to-select (selectionruntime) is a
      // separate, not-yet-wired concern (see this slice's report) — this
      // simulates "an object is selected" by setting the same
      // data-studio-selection attribute selectionruntime would set, purely
      // to prove the working-state PERSISTENCE contract this slice adds.
      await form.evaluate((node) => node.setAttribute("data-studio-selection", "home:hero"));

      // Make a real edit so the shared form is genuinely dirty. #12/#13 (UX
      // wave 2A) made the primary Save button visible only while dirty (see
      // hostruntime/assets/studio.css [data-gosx-studio-save-button-state]
      // and state_runtime.js setState()) instead of a static always-on
      // button — a synthetic data-studio-selection attribute alone (above)
      // does not dirty the form, so type into a real, visible Advanced-mode
      // field first, exactly like an operator would before saving.
      const navigationField = navigationPanel.locator("input[type='text']").first();
      await expect(navigationField).toBeVisible();
      await navigationField.fill("Shop the collection");
      await expect(page.locator("[data-gosx-studio-save-button]").first()).toBeVisible();

      // Save an edit: the toolbar's primary Save button submits the whole
      // shared workbench form via a real (unintercepted) POST to the editor
      // route itself (gosx's file-action POST target, not a distinct
      // "/__actions/save" sub-path) + full-page navigation, exactly like
      // the owner's "interaction fails after an edit" report.
      const [response] = await Promise.all([
        page.waitForResponse((res) =>
          res.request().method() === "POST" &&
          res.url().includes("/admin/editor") &&
          !res.url().includes("/__actions/") &&
          !res.url().includes("client-events")),
        page.locator("[data-gosx-studio-save-button]").first().click(),
      ]);
      expect(response.status()).toBeLessThan(400);
      await page.waitForLoadState("domcontentloaded");

      // Post-reload: mode, panel visibility, and selection must all survive
      // — not reset back to the server's default Home/no-selection state.
      const reloadedForm = page.locator("[data-studio-workbench='true']").first();
      await page.waitForFunction(() => "GoSXStudioWorkbenchRuntime" in window, null, { timeout: 20_000 });
      await expect(reloadedForm).toHaveAttribute("data-studio-mode", "advanced");
      await expect(page.locator("[data-studio-navigation-panel='true']").first()).toBeVisible();
      await expect(page.locator("[data-studio-home-inspector-panel='true']").first()).toBeHidden();
      await expect(reloadedForm).toHaveAttribute("data-studio-selection", "home:hero");
    } finally {
      await server.stop();
    }
  });
});
