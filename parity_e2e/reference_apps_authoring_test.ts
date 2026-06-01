import { expect, test } from "@playwright/test";
import {
  authoringParam,
  applyAuthoringPanelInPlace,
  applyCompositionIntentInPlace,
  clickEditorActionButton,
  expectPajaritosPublishingReadiness,
  expectPanelButtonReceivesPointer,
  reorderComponent,
  saveEditableControl,
  savePageMetadata,
  startMuddy,
  startPajaritos,
  toggleComponentVisibility,
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

      await savePageMetadata(page, "Studio Test Notes", "/pages/studio-test-notes", {
        reloadAfter: false,
        expectedMessage: "Studio Test Notes saved.",
      });

      await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: "Landing created.",
        expectedChangeKind: "page",
        requireSelection: false,
      });
      await expect(page.locator("body")).toContainText("6 pages");

      await applyCompositionIntentInPlace(page, "add-component:home:contact-flow", {
        expectedMessage: "Contact form added to Home.",
      });
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await applyAuthoringPanelInPlace(page, "[data-studio-site-map-component-duplicate='true']", {
        expectedMessage: "Hero copy 2 duplicated.",
      });
      await expect(page.locator("body")).toContainText("Hero copy 2");

      await saveEditableControl(page, "Surface-tested clay rituals", {
        reloadAfter: false,
        expectedMessage: "Hero headline saved.",
      });

      await toggleComponentVisibility(page, { reloadAfter: false });
      await reorderComponent(page, { reloadAfter: false });
    } finally {
      await server.stop();
    }
  });

  test("Pajaritos visible authoring controls submit through the real admin editor", async ({ page, request }) => {
    const server = await startPajaritos(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle" });
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await savePageMetadata(page, "Forest Notes", "/pages/forest-notes", {
        reloadAfter: false,
        expectedMessage: "Forest Notes saved.",
      });

      const createResult = await applyCompositionIntentInPlace(page, "create-page:program-page", {
        expectedMessage: "Program Page created.",
        expectedChangeKind: "page",
      });
      expect(authoringParam(createResult.response, "gosx_studio_operation")).toBe("apply-intent");
      expect(authoringParam(createResult.response, "gosx_studio_intent_kind")).toBe("create-page");
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();
      await expect(page.locator("[data-gosx-studio-page-list='true']")).toContainText("6 pages");
      await expect(page.locator("[data-gosx-studio-page-list='true']")).toContainText("Program Page");

      const duplicateResult = await applyAuthoringPanelInPlace(page, "[data-gosx-studio-component-duplicate='true']", {
        expectedMessage: "Hero copy 2 duplicated.",
      });
      expect(authoringParam(duplicateResult.response, "gosx_studio_operation")).toBe("duplicate-component");
      expect(authoringParam(duplicateResult.response, "gosx_studio_component_key")).toBe("hero");

      await saveEditableControl(page, "Small boots, big questions.", {
        reloadAfter: false,
        expectedMessage: "Hero headline saved.",
      });

      await toggleComponentVisibility(page, { reloadAfter: false });
      await reorderComponent(page, { reloadAfter: false });

      const addResult = await applyCompositionIntentInPlace(page, "add-component:home:hero", {
        expectedMessage: "Hero section added to Home.",
      });
      expect(authoringParam(addResult.response, "gosx_studio_operation")).toBe("apply-intent");
      expect(authoringParam(addResult.response, "gosx_studio_intent_kind")).toBe("add-component");
      expect(authoringParam(addResult.response, "gosx_studio_component_template_key")).toBe("hero");
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
