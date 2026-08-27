import { expect, test, type Page } from "@playwright/test";
import {
  applyCompositionIntentInPlace,
  gotoEditor,
  revealModeIfPresent,
  startMuddyCanvas,
} from "./reference_apps_harness";

const CREATE_MESSAGE = "Landing page created with 3 starter sections. Edit its sections in Content › Pages.";
const PUBLISH_BUTTON = "[data-studio-submit-action='publish']";
const PUBLISH_ROUTE = "/admin/editor/__actions/publish";
const SITE_NAVIGATOR = "[data-studio-site-navigator-panel='true']";

test.describe("@reference-apps Muddy/Noni page CMS lifecycle", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("owner can create a CMS page, discover it in the editor rail, save a draft in Page CMS, preview it, then publish it live", async ({ page, request }) => {
    page.on("dialog", (dialog) => { void dialog.accept(); });

    const nonce = Math.random().toString(36).slice(2, 10);
    const marker = `Friday CMS handoff ${Date.now()} ${nonce}`;
    const title = `Friday Turnover ${nonce}`;
    const slug = "new-page";

    const server = await startMuddyCanvas(request);
    try {
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");

      const createResult = await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: CREATE_MESSAGE,
        expectedChangeKind: "page",
        requireSelection: false,
      });
      const pageID = createResult.detail?.change?.pageKey;
      expect(pageID, "create-page should report the new CMS page id as change.pageKey").toBeTruthy();

      const railLink = page.locator(`${SITE_NAVIGATOR} a[href="/admin/pages/${pageID}"]`).first();
      await expect(railLink, "the refreshed editor rail should expose the new CMS page as a direct Content > Pages link").toBeAttached({ timeout: 30_000 });
      await expect(railLink).toContainText("Landing");

      const pagesIndexURL = `${server.baseURL}/admin/pages`;
      await page.goto(pagesIndexURL, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const canonicalRow = page.locator("tr", { has: page.locator(`a[href="/admin/pages/${pageID}"]`) });
      await expect(canonicalRow, "the canonical CMS page list should include the created draft page").toHaveCount(1);
      await expect(canonicalRow).toContainText("new-page");

      await page.goto(pagesIndexURL, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await canonicalRow.locator(`a[href="/admin/pages/${pageID}"]`).first().click();
      await expect(page).toHaveURL(new RegExp(`/admin/pages/${pageID}$`));
      await expect(page.locator("[data-gosx-studio-backend-page-detail-renderer='gosx-studio']").first()).toBeAttached();
      const detailForm = page.locator("form.admin-form").first();

      await expect(detailForm.locator("button", { hasText: "Save draft" })).toBeVisible();
      await expect(detailForm.locator("label", { hasText: "Include this page in the next publish" })).toBeVisible();
      await expect(detailForm.locator("a.button", { hasText: "Preview" })).toBeVisible();
      await expect(detailForm.locator("a.button", { hasText: "Cancel" })).toBeVisible();
      await expect(detailForm.locator("a.button", { hasText: "Open editor to publish" })).toBeVisible();
      await expect(page.locator("body")).toContainText("Save creates a page draft. Use the Release Center in the editor to publish it.");

      const unsavedTitle = `Unsaved ${title}`;
      await detailForm.locator("input[name='title']").fill(unsavedTitle);
      await detailForm.locator("a.button", { hasText: "Cancel" }).first().click();
      await expect(page).toHaveURL(/\/admin\/pages$/);
      await expect(page.locator("body"), "Cancel should leave the unsaved title out of the CMS page list").not.toContainText(unsavedTitle);

      await page.goto(`${server.baseURL}/admin/pages/${pageID}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await page.locator("input[name='title']").fill(title);
      await expect(page.locator("input[name='slug']")).toHaveValue(slug);
      await page.locator("select[name='bodyFormat']").selectOption("blocks");
      await addVisibleHeadingBlock(page, marker);
      await page.locator("input[name='published']").check();

      const saveResponsePromise = page.waitForResponse((response) =>
        response.url().includes(`/admin/pages/${pageID}/__actions/save`) &&
        response.request().method() === "POST",
      { timeout: 30_000 });
      await page.locator("button", { hasText: "Save draft" }).first().click();
      const saveResponse = await saveResponsePromise;
      expect([200, 303].includes(saveResponse.status()), `Save draft should be accepted/redirected; got ${saveResponse.status()}`).toBe(true);
      await page.waitForURL(new RegExp(`/admin/pages/${pageID}$`), { timeout: 30_000 }).catch(() => {});
      await expect(page.locator("input[name='title']")).toHaveValue(title);
      await expect(page.locator("textarea[name='body']")).toHaveValue(new RegExp(marker));

      const detailFormAfterSave = page.locator("form.admin-form").first();
      const previewLink = detailFormAfterSave.locator("a.button", { hasText: "Preview" }).first();
      await expect(previewLink, "Page CMS should expose a visible Preview link").toBeVisible();
      const publicURL = `${server.baseURL}/pages/${slug}`;

      const publicDuringDraft = await request.get(publicURL);
      expect(publicDuringDraft.status(), "before Release Center publish the public page route should still be unpublished").toBe(404);

      await previewLink.click();
      await expect(page).toHaveURL(/\/admin\/storefront\?path=%2Fpages%2Fnew-page%3Fpreview%3D1$/);
      const previewFrame = page.frameLocator("iframe.storefront-frame").first();
      await expect(previewFrame.locator("body"), "the visible Preview affordance should render the draft heading block").toContainText(marker);
      await expect(previewFrame.locator("body"), "the visible Preview affordance should render the saved draft title").toContainText(title);

      await page.goto(`${server.baseURL}/admin/pages/${pageID}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await page.locator("form.admin-form").first().locator("a.button", { hasText: "Open editor to publish" }).first().click();
      await expect(page).toHaveURL(/\/admin\/editor$/);
      const publishStatus = await triggerPublish(page);
      expect([200, 303].includes(publishStatus), `Release Center publish should be accepted/redirected; got ${publishStatus}`).toBe(true);

      const publicAfterPublish = await request.get(publicURL);
      expect(publicAfterPublish.status(), "after Release Center publish the public page route should be live").toBe(200);
      const publicHTML = await publicAfterPublish.text();
      expect(publicHTML).toContain(marker);
      expect(publicHTML).toContain(title);
    } finally {
      await server.stop();
    }
  });
});

async function addVisibleHeadingBlock(page: Page, marker: string) {
  await page.locator("button[data-content-add='heading']").click();
  const block = page.locator("[data-content-block-id]").last();
  await expect(block, "the visible Page CMS block list should add a heading row").toBeVisible();
  await expect(block.locator("select[aria-label='Block type']")).toHaveValue("heading");

  const headingInput = block.locator(".content-block__fields label", { hasText: "Heading" }).locator("input").first();
  await expect(headingInput, "the new heading block should expose a visible Heading field").toBeVisible();
  await headingInput.fill(marker);
  await expect(block.locator(".content-block__preview h3"), "the visible block preview should update as the owner types").toHaveText(marker);
  await expect(page.locator("textarea[name='body']")).toHaveValue(new RegExp(marker));
}

async function triggerPublish(page: Page): Promise<number> {
  await revealModeIfPresent(page, "publish");
  const button = page.locator(PUBLISH_BUTTON).first();
  await expect(button, "the Release Center Publish button should be visible").toBeVisible({ timeout: 30_000 });
  const responsePromise = page.waitForResponse((response) =>
    response.url().includes(PUBLISH_ROUTE) &&
    response.request().method() === "POST",
  { timeout: 30_000 });
  await button.click();
  const response = await responsePromise;
  await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
  return response.status();
}
