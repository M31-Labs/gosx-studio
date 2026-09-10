import { expect, test, type Page } from "@playwright/test";
import {
  expectEditorDocumentContinuity,
  installEditorDocumentContinuityProbe,
} from "./reference_apps_document_continuity";

const ORIGIN = "http://127.0.0.1:4173";

async function routeDocumentFixtures(page: Page) {
  await page.route(`${ORIGIN}/editor`, async (route) => {
    await route.fulfill({
      contentType: "text/html",
      body: `
        <main data-gosx-studio-workbench="true">
          <button id="fetch-save" type="button">Save</button>
          <button id="iframe-refresh" type="button">Refresh Preview</button>
          <button id="history-change" type="button">History</button>
          <a id="reload-link" href="/editor-reloaded">Reload</a>
          <iframe id="preview-frame" title="preview" src="/preview"></iframe>
          <script>
            document.querySelector("#fetch-save").addEventListener("click", () => {
              fetch("/admin/editor/__actions/save", { method: "POST" });
            });
            document.querySelector("#iframe-refresh").addEventListener("click", () => {
              document.querySelector("#preview-frame").src = "/preview?refresh=1";
            });
            document.querySelector("#history-change").addEventListener("click", () => {
              history.pushState({ editor: true }, "", "/editor#changed");
            });
          </script>
        </main>
      `,
    });
  });
  await page.route(`${ORIGIN}/editor-reloaded`, async (route) => {
    await route.fulfill({
      contentType: "text/html",
      body: `<main data-gosx-studio-workbench="true">Reloaded editor document</main>`,
    });
  });
  await page.route(`${ORIGIN}/preview**`, async (route) => {
    await route.fulfill({
      contentType: "text/html",
      body: `<p>Preview ${new URL(route.request().url()).searchParams.get("refresh") ?? "initial"}</p>`,
    });
  });
  await page.route(`${ORIGIN}/admin/editor/__actions/save`, async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ ok: true }),
    });
  });
}

test.describe("editor document continuity helper", () => {
  test.beforeEach(async ({ page }) => {
    await routeDocumentFixtures(page);
    await page.goto(`${ORIGIN}/editor`, { waitUntil: "domcontentloaded" });
  });

  test("catches real editor main-document reloads", async ({ page }) => {
    await expect(
      expectEditorDocumentContinuity(page, async () => {
        await page.locator("#reload-link").click();
        await page.waitForURL(`${ORIGIN}/editor-reloaded`);
      }),
    ).rejects.toThrow(/editor main-frame document requests/);
  });

  test("permits fetch saves, iframe refresh, and same-document history changes", async ({ page }) => {
    await expectEditorDocumentContinuity(page, async () => {
      const saveResponse = page.waitForResponse(
        (response) => response.url() === `${ORIGIN}/admin/editor/__actions/save`,
      );
      await page.locator("#fetch-save").click();
      await saveResponse;

      await page.locator("#iframe-refresh").click();
      await page.frameLocator("#preview-frame").locator("text=Preview 1").waitFor();

      await page.locator("#history-change").click();
      await expect(page).toHaveURL(`${ORIGIN}/editor#changed`);
    });
  });

  test("disposes request listeners after assertion failures", async ({ page }) => {
    const probe = await installEditorDocumentContinuityProbe(page);
    try {
      await page.locator("#reload-link").click();
      await page.waitForURL(`${ORIGIN}/editor-reloaded`);
      await expect(probe.assertStillOnSameDocument()).rejects.toThrow(/editor main-frame document requests/);
    } finally {
      probe.dispose();
    }

    const recordedAfterDispose = probe.mainDocumentRequests();
    await page.goto(`${ORIGIN}/editor`);
    expect(probe.mainDocumentRequests()).toEqual(recordedAfterDispose);
  });
});
