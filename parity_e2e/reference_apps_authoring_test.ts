import { expect, test } from "@playwright/test";
import {
  authoringParam,
  clickAuthoringPanel,
  clickEditorActionButton,
  clickIntent,
  expectIntentButtonReceivesPointer,
  expectPajaritosPublishingReadiness,
  expectPanelButtonReceivesPointer,
  saveEditableControl,
  startMuddy,
  startPajaritos,
  waitForStudioPreviewRefresh,
} from "./reference_apps_harness";

test.describe("@reference-apps browser authoring workflows", () => {
  test.describe.configure({ timeout: 90_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni visible authoring controls submit through the real admin editor", async ({ page, request }) => {
    const server = await startMuddy(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle" });
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await expectIntentButtonReceivesPointer(page, "create-page:landing");
      const createResponse = await clickIntent(page, "create-page:landing");
      expect(createResponse.status()).toBe(303);
      await expect(page.locator("body")).toContainText("6 pages");

      await expectIntentButtonReceivesPointer(page, "add-component:home:contact-flow");
      const addResponse = await clickIntent(page, "add-component:home:contact-flow");
      expect(addResponse.status()).toBe(303);
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await expectPanelButtonReceivesPointer(page, "[data-studio-site-map-component-duplicate='true']");
      const duplicateResponse = await clickAuthoringPanel(page, "[data-studio-site-map-component-duplicate='true']");
      expect(duplicateResponse.status()).toBe(303);
      await expect(page.locator("body")).toContainText("Hero copy 2");

      await saveEditableControl(page, "Surface-tested clay rituals", {
        reloadAfter: false,
        expectedMessage: "Hero headline saved.",
      });
    } finally {
      await server.stop();
    }
  });

  test("Pajaritos visible authoring controls submit through the real admin editor", async ({ page, request }) => {
    const server = await startPajaritos(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle" });
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await expectIntentButtonReceivesPointer(page, "create-page:program-page");
      const createResponse = await clickIntent(page, "create-page:program-page");
      expect(createResponse.status()).toBe(303);
      expect(authoringParam(createResponse, "gosx_studio_operation")).toBe("apply-intent");
      expect(authoringParam(createResponse, "gosx_studio_intent_kind")).toBe("create-page");
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();
      await expect(page.locator("[data-gosx-studio-page-list='true']")).toContainText("6 pages");
      await expect(page.locator("[data-gosx-studio-page-list='true']")).toContainText("Program Page");
      await expect(page.locator("body")).toContainText("Program Page created.");

      await expectPanelButtonReceivesPointer(page, "[data-gosx-studio-component-duplicate='true']");
      const duplicateResponse = await clickAuthoringPanel(page, "[data-gosx-studio-component-duplicate='true']", { reloadAfter: false });
      expect(duplicateResponse.status()).toBe(303);
      expect(authoringParam(duplicateResponse, "gosx_studio_operation")).toBe("duplicate-component");
      expect(authoringParam(duplicateResponse, "gosx_studio_component_key")).toBe("hero");
      await expect(page.locator("body")).toContainText("Hero copy 2 duplicated.");

      await saveEditableControl(page, "Small boots, big questions.", {
        reloadAfter: false,
        expectedMessage: "Hero headline saved.",
      });

      await expectIntentButtonReceivesPointer(page, "add-component:home:hero");
      const addResponse = await clickIntent(page, "add-component:home:hero");
      expect(addResponse.status()).toBe(303);
      expect(authoringParam(addResponse, "gosx_studio_operation")).toBe("apply-intent");
      expect(authoringParam(addResponse, "gosx_studio_intent_kind")).toBe("add-component");
      expect(authoringParam(addResponse, "gosx_studio_component_template_key")).toBe("hero");
      await expect(page.locator("body")).toContainText("Hero section added to Home.");
      await expect(page.locator("[data-studio-site-map-component='hero__copy_3']").first()).toBeAttached();
      await expectPajaritosPublishingReadiness(page);

      const restoreButtonSelector = "[data-pajaritos-restore-index='1'] button";
      await expectPanelButtonReceivesPointer(page, "[data-pajaritos-restore-index='1']");
      const restoreButton = page.locator(restoreButtonSelector).first();
      const restoreRevisionID = await restoreButton.getAttribute("value");
      expect(await restoreButton.getAttribute("name")).toBe("revisionId");
      expect(restoreRevisionID).toBeTruthy();
      const previewRefreshPromise = waitForStudioPreviewRefresh(page, "action");
      const restoreResponse = await clickEditorActionButton(page, restoreButtonSelector, "/__actions/restoreRevision", {
        noWaitAfter: true,
        reloadAfter: false,
        settleAfter: false,
      });
      expect(restoreResponse.status()).toBe(303);
      expect(restoreResponse.headers()["location"]).toContain("Restore+point+applied.");
      await expect(previewRefreshPromise).resolves.toMatchObject({ reason: "action" });
      await expect(page.locator("[data-studio-preview-frame]").first()).toHaveAttribute("src", /_gosx_preview_reason=action/);
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle" });
      await expect(page.locator("[data-studio-site-map-component='hero__copy_2']").first()).toBeAttached();
      await expect(page.locator("[data-studio-site-map-component='hero__copy_3']")).toHaveCount(0);
    } finally {
      await server.stop();
    }
  });
});
