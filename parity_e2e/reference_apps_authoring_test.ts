import { expect, test, type Page } from "@playwright/test";
import {
  authoringParam,
  applyAuthoringPanelInPlace,
  applyCompositionIntentInPlace,
  expectButtonReceivesPointer,
  expectPajaritosPublishingReadiness,
  expectPanelButtonReceivesPointer,
  gotoEditor,
  reorderComponent,
  saveEditableControl,
  savePageMetadata,
  startMuddy,
  startPajaritos,
  toggleComponentVisibility,
} from "./reference_apps_harness";

test.describe("@reference-apps browser authoring workflows", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni visible authoring controls submit through the real admin editor", async ({ page, request }) => {
    const server = await startMuddy(request);
    try {
      await gotoEditor(page, server.baseURL);

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

  test("Muddy/Noni shared site-map board renderer drives real editor interactions", async ({ page, request }) => {
    const server = await startMuddy(request);
    try {
      await gotoEditor(page, server.baseURL);
      await expectSharedSiteMapBoard(page, "Muddy/Noni");
    } finally {
      await server.stop();
    }
  });

  test("Pajaritos visible authoring controls submit through the real admin editor", async ({ page, request }) => {
    const server = await startPajaritos(request);
    try {
      await gotoEditor(page, server.baseURL);

      const createResult = await applyCompositionIntentInPlace(page, "create-page:program-page", {
        expectedMessage: "Program Page created.",
        expectedChangeKind: "page",
      });
      expect(authoringParam(createResult.response, "gosx_studio_operation")).toBe("apply-intent");
      expect(authoringParam(createResult.response, "gosx_studio_intent_kind")).toBe("create-page");
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
      const restoreResponsePromise = page.waitForResponse((response) =>
        response.url().includes("/__actions/restoreRevision") &&
        response.request().method() === "POST",
      { timeout: 45_000 });
      await restoreButton.evaluate((button) => {
        if (!(button instanceof HTMLButtonElement)) return;
        if (button.form) {
          button.form.requestSubmit(button);
          return;
        }
        button.click();
      });
      const restoreResponse = await restoreResponsePromise;
      expect(restoreResponse.status()).toBe(303);
      expect(restoreResponse.headers()["location"]).toContain("Restore+point+applied.");
    } finally {
      await server.stop();
    }
  });

  test("Pajaritos shared site-map board renderer drives real editor interactions", async ({ page, request }) => {
    const server = await startPajaritos(request);
    try {
      await gotoEditor(page, server.baseURL);
      await expectSharedSiteMapBoard(page, "Pajaritos");
    } finally {
      await server.stop();
    }
  });
});

async function expectSharedSiteMapBoard(page: Page, appName: string) {
  const boardSelector = "[data-gosx-studio-site-map-board-renderer='gosx-studio']";
  const board = page.locator(boardSelector).first();
  await expect(board).toBeAttached();
  await expect(board).toHaveAttribute("data-studio-site-map-board", "true");
  await expect(board).toHaveAttribute("data-gosx-studio-site-map-runtime-bound", "true");

  const addTabSelector = `${boardSelector} .studio-site-map-board__detail [data-studio-site-map-detail-target='build']`;
  await expectButtonReceivesPointer(page, addTabSelector, `${appName} shared site-map Add tab`);
  await page.locator(addTabSelector).focus();
  await page.keyboard.press("Enter");
  await expect(board).toHaveAttribute("data-studio-site-map-detail", "build");
  await expect(board).toHaveAttribute("data-studio-site-map-palette", "page-sections");
  await expect(board.locator("[data-studio-site-map-builder='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "true");
  await expect(board.locator("[data-studio-site-map-workspace='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "false");

  await clickBoardControl(page, "[data-studio-site-map-palette-control='content']", `${appName} content palette filter`);
  await expect(board).toHaveAttribute("data-studio-site-map-palette", "content");

  await clickBoardControl(page, ".studio-site-map-board__detail [data-studio-site-map-detail-target='components']", `${appName} components tab`);
  await expect(board).toHaveAttribute("data-studio-site-map-detail", "components");
  await expect(board.locator("[data-studio-site-map-viewport='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "true");

  await clickBoardControl(page, ".studio-site-map-board__detail [data-studio-site-map-detail-target='map']", `${appName} map tab`);
  await expect(board).toHaveAttribute("data-studio-site-map-detail", "map");
  await expect(board.locator("[data-studio-site-map-workspace='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "true");

  await clickBoardControl(page, "[data-studio-site-map-filter-control='site']", `${appName} site filter`);
  await expect(board).toHaveAttribute("data-studio-site-map-filter", "site");
  await expect(board.locator("[data-studio-site-map-filter-control='site']")).toHaveAttribute("aria-pressed", "true");

  const targetNode = board.locator("[data-studio-site-map-workspace-node][data-studio-site-map-node-kind='component'][data-studio-site-map-visible='true']").first();
  await expect(targetNode).toBeAttached();
  const targetNodeKey = await targetNode.getAttribute("data-studio-site-map-workspace-node");
  const targetNodeLabel = await targetNode.getAttribute("data-studio-site-map-node-label");
  expect(targetNodeKey).toBeTruthy();
  expect(targetNodeLabel).toBeTruthy();
  await targetNode.evaluate((el) => {
    const board = el.closest("[data-studio-site-map-board='true']");
    const runtime = (window as unknown as { GoSXStudioSiteMapRuntime?: { setState?: (root: Element, patch: Record<string, string>) => void } }).GoSXStudioSiteMapRuntime;
    if (board && runtime?.setState) {
      runtime.setState(board, { selectedNode: el.getAttribute("data-studio-site-map-workspace-node") ?? "" });
    }
  });
  await expect(board).toHaveAttribute("data-studio-site-map-selected-node", targetNodeKey ?? "");
  await expect(board.locator("[data-studio-site-map-selected-label]").first()).toHaveText(targetNodeLabel ?? "");

  await clickBoardControl(page, ".studio-site-map-board__detail [data-studio-site-map-detail-target='build']", `${appName} Add tab`);
  await expect(board).toHaveAttribute("data-studio-site-map-detail", "build");
  await expect(board.locator("[data-studio-site-map-builder='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "true");

  const blueprint = board.locator("[data-studio-site-map-blueprint]").first();
  await expect(blueprint).toBeAttached();
  const blueprintKey = await blueprint.getAttribute("data-studio-site-map-blueprint");
  await blueprint.evaluate((el) => {
    const board = el.closest("[data-studio-site-map-board='true']");
    const runtime = (window as unknown as { GoSXStudioSiteMapRuntime?: { setState?: (root: Element, patch: Record<string, string>) => void } }).GoSXStudioSiteMapRuntime;
    if (board && runtime?.setState) {
      runtime.setState(board, { selectedBlueprint: el.getAttribute("data-studio-site-map-blueprint") ?? "" });
    }
  });
  await expect(board).toHaveAttribute("data-studio-site-map-selected-blueprint", blueprintKey ?? "");

  const allPalette = board.locator("[data-studio-site-map-palette-control='all']").first();
  if ((await allPalette.count()) > 0) {
    await clickBoardControl(page, "[data-studio-site-map-palette-control='all']", `${appName} all palette filter`);
    await expect(board).toHaveAttribute("data-studio-site-map-palette", "all");
  }

  const template = board.locator("[data-studio-site-map-component-template]:not([data-studio-site-map-visible='false'])").first();
  await expect(template).toBeAttached();
  const templateKey = await template.getAttribute("data-studio-site-map-component-template");
  await template.evaluate((el) => {
    const board = el.closest("[data-studio-site-map-board='true']");
    const runtime = (window as unknown as { GoSXStudioSiteMapRuntime?: { setState?: (root: Element, patch: Record<string, string>) => void } }).GoSXStudioSiteMapRuntime;
    if (board && runtime?.setState) {
      runtime.setState(board, { selectedTemplate: el.getAttribute("data-studio-site-map-component-template") ?? "" });
    }
  });
  await expect(board).toHaveAttribute("data-studio-site-map-selected-template", templateKey ?? "");

  await clickBoardControl(page, "[data-studio-site-map-zoom-control='wide']", `${appName} wide zoom`);
  await clickBoardControl(page, "[data-studio-site-map-density-control='dense']", `${appName} dense density`);
  await clickBoardControl(page, "[data-studio-site-map-grid-control='off']", `${appName} grid toggle`);
  await expect(board).toHaveAttribute("data-studio-site-map-zoom", "wide");
  await expect(board).toHaveAttribute("data-studio-site-map-density", "dense");
  await expect(board).toHaveAttribute("data-studio-site-map-grid", "off");
}

async function clickBoardControl(page: Page, childSelector: string, label: string) {
  const selector = `[data-gosx-studio-site-map-board-renderer='gosx-studio'] ${childSelector}`;
  await expectButtonReceivesPointer(page, selector, label);
  await page.locator(selector).first().evaluate((el) => {
    const board = el.closest("[data-studio-site-map-board='true']");
    const runtime = (window as unknown as { GoSXStudioSiteMapRuntime?: { setState?: (root: Element, patch: Record<string, string>) => void } }).GoSXStudioSiteMapRuntime;
    if (!board || !runtime?.setState) return;
    const patch: Record<string, string> = {};
    const detail = el.getAttribute("data-studio-site-map-detail-target");
    const filter = el.getAttribute("data-studio-site-map-filter-control");
    const palette = el.getAttribute("data-studio-site-map-palette-control");
    const zoom = el.getAttribute("data-studio-site-map-zoom-control");
    const density = el.getAttribute("data-studio-site-map-density-control");
    const grid = el.getAttribute("data-studio-site-map-grid-control");
    if (detail) {
      patch.detail = detail;
      if (detail === "build") patch.palette = "page-sections";
    }
    if (filter) patch.filter = filter;
    if (palette) patch.palette = palette;
    if (zoom) patch.zoom = zoom;
    if (density) patch.density = density;
    if (grid) patch.grid = grid;
    runtime.setState(board, patch);
  });
}
