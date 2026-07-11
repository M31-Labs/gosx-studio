import { expect, test } from "@playwright/test";
import { startMuddy } from "./reference_apps_harness";

// Descriptor HANDOFF-COMPONENTS-ASSETS-05: proves the media-manager
// referenced-asset protection end to end in a real headless browser against
// the real Muddy/Noni admin — "Delete permanently" only appears (and only
// succeeds) for an asset with no usage; a referenced asset must never be
// silently deletable. The negative (referenced-asset) path is covered at the
// Go level (delete_action_test.go, media_test.go in both repos); this test
// proves the browser-facing control itself is correctly gated and wired to
// the guarded delete route.
test.describe("@reference-apps Muddy/Noni media library delete guard", () => {
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("an unreferenced remote asset can be added, then permanently deleted from its detail page", async ({ page, request }) => {
    // The Delete button carries data-admin-confirm; auto-accept so the
    // confirm() prompt never stalls the test.
    page.on("dialog", (dialog) => { void dialog.accept(); });

    const server = await startMuddy(request);
    try {
      const nonce = Math.random().toString(36).slice(2, 10);
      const url = `https://example.test/media-e2e-${nonce}.jpg`;
      const filename = `media-e2e-${nonce}.jpg`;

      await page.goto(`${server.baseURL}/admin/media`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("#url")).toBeAttached();

      await page.locator("#url").fill(url);
      await page.locator("#alt").fill(`Media e2e asset ${nonce}`);
      await page.locator("#filename").fill(filename);
      await page.locator("form.admin-form--single button[type=submit]").first().click();

      // createAction redirects to /admin/media/<id> on success.
      await page.waitForURL(/\/admin\/media\/[^/?]+$/, { timeout: 30_000 });

      const deleteButton = page.locator("[data-gosx-studio-media-delete='true']");
      await expect(deleteButton, "an unreferenced asset must offer the guarded Delete control").toBeAttached({ timeout: 10_000 });
      expect(await deleteButton.getAttribute("formaction")).toMatch(/\/__actions\/delete$/);

      await deleteButton.click();
      await page.waitForURL(/\/admin\/media\?deleted=1$/, { timeout: 30_000 });

      await expect(page.locator("body")).not.toContainText(filename);
    } finally {
      await server.stop();
    }
  });
});
