import { expect, test, type Page } from "@playwright/test";
import { startMuddy } from "./reference_apps_harness";

// Live-proof e2e for the PageCanvas turnover safety contract. The default
// Muddy/Noni PageCanvas can expose text-selectable iframe nodes whose typed
// content is not backed by a live, value-bearing editor control. Those nodes
// must not enter contenteditable inline mode: doing so creates owner-visible
// typing that looks like an edit but never reaches durable CMS persistence.
//
// The Friday behavior is deliberately conservative. Double-click still selects
// the preview text and reveals the field editor, but it does not set
// contenteditable, does not mark the editor dirty, and does not emit inline
// text operations unless the standard Workbench host can resolve a real backing
// control for the field.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const WORKBENCH_SELECTOR = "[data-studio-workbench='true']";
const PAGE_CANVAS_SELECTOR = "[data-gosx-studio-page-canvas='true']";
const PREVIEW_FRAME_SELECTOR = "[data-studio-preview-frame]";
const HERO_FIELD_SELECTOR = "[data-studio-field='hero.headline']";

test.describe("@reference-apps PageCanvas inline text edit fails safe without a backing control", () => {
  test.describe.configure({ timeout: 120_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni editor: double-clicking an unbacked PageCanvas text field opens the durable field panel instead of contenteditable", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddy(request, { MUDDY_EDITOR_SCENE_DOM: "1" });
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const workbench = page.locator(WORKBENCH_SELECTOR).first();
      await expect(workbench).toBeAttached();

      const pageCanvas = page.locator(PAGE_CANVAS_SELECTOR).first();
      await expect(pageCanvas, "PageCanvas must be the default Home surface").toBeVisible({ timeout: 30_000 });

      const previewFrame = page.locator(PREVIEW_FRAME_SELECTOR).first();
      await expect(previewFrame, "the preview iframe must be attached").toBeAttached({ timeout: 30_000 });
      await hydrateDeferredPreviewFrame(page);
      await previewFrame.scrollIntoViewIfNeeded();
      await expect(previewFrame, "the preview iframe must be visible").toBeVisible();

      await page.waitForFunction(() => {
        const frame = document.querySelector("[data-studio-preview-frame]");
        if (!(frame instanceof HTMLIFrameElement)) return false;
        const body = frame.contentDocument?.body;
        return !!body && body.innerText.trim().length > 10;
      }, null, { timeout: 45_000 });
      await expect(previewFrame, "the preview iframe must report ready after its first load").toHaveAttribute("data-studio-preview-state", "ready", { timeout: 30_000 });

      const seedText = (await editableHeroText(page)).trim();
      expect(seedText, "the live storefront's hero headline must have seeded text").not.toBe("");

      await page.evaluate(() => {
        const w = window as unknown as { __pageCanvasFallbackEvents: unknown[] };
        w.__pageCanvasFallbackEvents = [];
        document.addEventListener("gosxstudio:editor-operation", (event) => {
          const detail = (event as CustomEvent).detail ?? {};
          w.__pageCanvasFallbackEvents.push({
            kind: detail.kind ?? detail.type ?? "",
            reason: detail.operation?.reason ?? detail.reason ?? "",
            mutation: detail.mutation ?? detail.operation?.mutation ?? false,
          });
        });
        document.addEventListener("gosxstudio:inline-text-start", (event) => {
          const detail = (event as CustomEvent).detail ?? {};
          w.__pageCanvasFallbackEvents.push({ type: "inline-start", field: detail.field ?? "" });
        });
      });

      const frameLocator = page.frameLocator(PREVIEW_FRAME_SELECTOR);
      const hero = frameLocator.locator(HERO_FIELD_SELECTOR).first();
      await expect(hero, "the live storefront's hero headline must render inside the preview iframe").toBeVisible({ timeout: 30_000 });
      await hero.dblclick();

      await expect(hero, "unbacked PageCanvas text must not enter inline contenteditable mode").not.toHaveAttribute("data-gosx-studio-inline-editing", "true", { timeout: 1000 });
      await expect(hero, "unbacked PageCanvas text must not become browser-editable").not.toHaveAttribute("contenteditable", "plaintext-only");
      await expect(workbench, "fallback should reveal the normalized Home mode that owns Muddy direct-edit panels").toHaveAttribute("data-studio-mode", "home", { timeout: 10_000 });
      const directEditForm = page.locator("#studioDirectEditForm").first();
      await expect(directEditForm, "fallback should reveal Muddy's durable direct-edit form").toBeVisible({ timeout: 10_000 });
      const directEditControl = directEditForm.locator("textarea[name='hero.headline'], [data-studio-field-source='hero.headline'] textarea, textarea").first();
      await expect(directEditControl, "fallback should expose the durable textarea for the selected hero headline").toBeVisible({ timeout: 10_000 });
      await expect(workbench, "fallback should keep the selected field bound to the durable editor path").toHaveAttribute("data-studio-field-selection", "hero.headline");
      await expect(directEditControl, "fallback should expose the selected preview text through the durable textarea").toHaveValue(seedText);

      const actionText = (await page.locator("[data-gosx-studio-preview-command='field-action']").first().textContent())?.trim() ?? "";
      expect(actionText, "the preview dock action must not promise canvas typing for an unbacked field").toBe("Open field");

      const fallbackEvents = await page.evaluate(() => (window as unknown as { __pageCanvasFallbackEvents: unknown[] }).__pageCanvasFallbackEvents);
      const forbiddenEvents = fallbackEvents.filter((event) => {
        const detail = event as { kind?: string; type?: string; mutation?: boolean };
        const kind = detail.kind ?? detail.type ?? "";
        return detail.mutation === true || kind === "set_text" || kind === "inline_text_start" || kind === "inline-start";
      });
      expect(forbiddenEvents, `unbacked fallback may select/reveal but must not dirty the editor or emit inline text operations; saw=${JSON.stringify(fallbackEvents)}`).toEqual([]);
      expect(await editableHeroText(page), "fallback must leave preview text unchanged").toBe(seedText);

      const clientErrors = consoleErrors.filter((line) => /preview|inline|canvas|hydrate/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected preview/inline-edit console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// editableHeroText reads the hero headline's live text straight off the
// preview iframe's own contentDocument (same-origin) so the assertion tracks
// the real rendered preview, not a fixture string.
async function editableHeroText(page: Page): Promise<string> {
  return page.evaluate((sel) => {
    const frame = document.querySelector("[data-studio-preview-frame]");
    if (!(frame instanceof HTMLIFrameElement)) return "";
    const el = frame.contentDocument?.querySelector(sel);
    return el ? (el.textContent || "") : "";
  }, HERO_FIELD_SELECTOR);
}

// hydrateDeferredPreviewFrame sets the iframe's src from its
// data-studio-preview-src/host data-gosx-studio-preview-url fallback if the
// server rendered it deferred. This mirrors reference_apps_canvas_first_test.ts.
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
