import { expect, test } from "@playwright/test";
import { gotoEditor, revealModeIfPresent, startMuddy } from "./reference_apps_harness";

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
      // /admin/media renders TWO forms sharing the class "admin-form
      // admin-form--single" (RenderBackendMediaLibraryUploadForm's file
      // upload form comes first in DOM order, then
      // RenderBackendMediaLibraryRemoteAssetForm's #url/#alt/#filename
      // form) -- see gosx-studio/backoffice/backend_media_library.go. A bare
      // ".first()" over that shared class picks the WRONG (upload) form's
      // submit button. Scope to the form that actually contains #url.
      await page.locator("form:has(#url) button[type=submit]").first().click();

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

  test("real seeded media ContentType reaches the editor datalist and picker", async ({ page, request }) => {
    // This is a source-backed host proof: startMuddy seeds the isolated
    // temporary CMS store, Noni's editor route projects that store through
    // view.MediaAssetList/editorBackendEditorMediaAssets, and Studio renders
    // the resulting option metadata. No fixture DOM or host state is added.
    const seededAsset = {
      label: "moss-handle-cup.jpg",
      urlFragment: "photo-1565193566173-7a0ee3dbe261",
      contentType: "image/jpeg",
      alt: "Ceramic cup with earthy glaze",
    } as const;
    const pageErrors: string[] = [];
    const externalImageRequests: string[] = [];
    const postRequests: string[] = [];
    const tinyPNG = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=", "base64");

    // Seeded records use Unsplash URLs. Fulfill only that external origin
    // locally; application responses from the real temporary host remain
    // untouched.
    await page.route(/^https:\/\/images\.unsplash\.com\//, async (route) => {
      await route.fulfill({ status: 200, contentType: "image/png", body: tinyPNG });
    });
    page.on("pageerror", (error) => pageErrors.push(error.message));
    page.on("request", (browserRequest) => {
      if (browserRequest.method() === "POST") postRequests.push(browserRequest.url());
      if (browserRequest.resourceType() === "image" && browserRequest.url().startsWith("https://images.unsplash.com/")) {
        externalImageRequests.push(browserRequest.url());
      }
    });

    const server = await startMuddy(request);
    try {
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");

      const editor = page.locator("[data-gosx-studio-backend-editor-renderer='gosx-studio']").first();
      await expect(editor, "real Noni editor must expose the Studio renderer").toBeVisible();
      const datalist = editor.locator("#editor-media-urls").first();
      const option = datalist.locator(`option[label='${seededAsset.label}']`).first();
      await expect(datalist, "real editor must render its media datalist").toBeAttached();
      await expect(option, "the seeded media record must reach the editor datalist").toHaveCount(1);
      await expect(option).toHaveAttribute("data-media-content-type", seededAsset.contentType);
      await expect(option).toHaveAttribute("data-media-alt", seededAsset.alt);
      const seededURL = await option.getAttribute("value");
      expect(seededURL, "the seeded media option must retain its source URL").toContain(seededAsset.urlFragment);
      if (!seededURL) throw new Error("the seeded media option must retain its source URL");

      const mediaInput = editor.locator("input[name='metaImageUrl']").first();
      const mediaPicker = editor.locator("input[name='metaImageUrl'] + .media-picker").first();
      const mediaTrigger = mediaPicker.locator(".media-picker__trigger").first();
      const mediaPanel = mediaPicker.locator(".media-picker__panel").first();
      const mediaSearch = mediaPicker.locator(".media-picker__search").first();
      const mediaAsset = mediaPicker.locator(`.media-picker__asset[aria-label='Use ${seededAsset.label}']`).first();
      const mediaAlt = editor.locator("input[name='metaImageAlt']").first();
      await expect(mediaInput).toHaveAttribute("list", "editor-media-urls");
      await expect(mediaInput, "real editor media input must bind to the metadata-backed picker").toHaveAttribute("data-media-picker-bound", "true", { timeout: 30_000 });
      await expect(mediaPicker).toHaveClass(/media-picker/);

      // Advanced mode contains a second, native group selector. The settings
      // fields are not visible until the real SEO radio is selected; activate
      // that associated label before starting the keyboard traversal below.
      const advancedPanel = editor.locator("[data-gosx-studio-advanced-panel-renderer='gosx-studio']").first();
      const seoGroup = advancedPanel.locator("label[data-studio-advanced-group-label='settings']").first();
      const seoRadio = advancedPanel.locator("#studioAdvancedGroupSettings").first();
      const settingsSlot = advancedPanel.locator("[data-studio-advanced-group-slot='settings']").first();
      await expect(advancedPanel, "real Noni editor must expose the Studio Advanced panel").toBeVisible();
      await expect(seoGroup, "the Advanced SEO group must expose its native associated label").toBeVisible();
      await seoGroup.click();
      await expect(seoRadio, "the SEO group must be selected through its native radio").toBeChecked();
      await expect(settingsSlot, "selecting SEO must reveal the real settings slot").toBeVisible();

      // Start from a real non-interactive panel heading and reach the chooser
      // by browser Tab traversal, rather than focusing the trigger in script.
      const settingsHeading = editor.getByRole("heading", { name: "SEO and integrations", exact: true }).first();
      await expect(settingsHeading, "the real Advanced settings heading must be available for keyboard traversal").toBeVisible();
      await settingsHeading.click();
      let triggerReachedByTab = false;
      for (let step = 0; step < 96; step += 1) {
        if (await mediaTrigger.evaluate((element) => document.activeElement === element)) {
          triggerReachedByTab = true;
          break;
        }
        await page.keyboard.press("Tab");
      }
      expect(triggerReachedByTab, "the real media-picker trigger must be keyboard reachable").toBe(true);
      await expect(mediaTrigger).toBeFocused();

      const postCountBeforePicker = postRequests.length;
      await page.keyboard.press("Enter");
      await expect(mediaSearch).toBeFocused();
      await expect(mediaPanel).not.toBeHidden();
      await page.keyboard.press("Escape");
      await expect(mediaPanel).toBeHidden();
      await expect(mediaTrigger).toBeFocused();

      await page.keyboard.press("Enter");
      await expect(mediaSearch).toBeFocused();
      await mediaSearch.fill(seededAsset.label);
      await expect(mediaAsset, "the picker must classify the MIME-backed seeded asset as an image").toHaveCount(1);
      await expect(mediaAsset.locator("img")).toHaveCount(1);
      await expect(mediaAsset.locator("img")).toHaveAttribute("src", seededURL);
      await expect(mediaAsset.locator(".media-picker__file")).toHaveCount(0);

      // Search is view-only; the asset's native button Enter is the deliberate
      // selection. Studio should close the panel and return focus to the input.
      await mediaSearch.press("Tab");
      await expect(mediaAsset).toBeFocused();
      await page.keyboard.press("Enter");
      await expect(mediaPanel).toBeHidden();
      await expect(mediaInput).toHaveValue(seededURL);
      await expect(mediaAlt).toHaveValue(seededAsset.alt);
      await expect(mediaInput).toBeFocused();
      expect(postRequests.slice(postCountBeforePicker), "picker search/selection must not submit the editor form").toEqual([]);
      await expect.poll(
        () => externalImageRequests.some((url) => url.includes(seededAsset.urlFragment)),
        { timeout: 10_000 },
      ).toBe(true);
      expect(pageErrors, "real host media picker flow must not raise page errors").toEqual([]);
    } finally {
      await server.stop();
    }
  });
});
