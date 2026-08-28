import { expect, test, type APIRequestContext, type JSHandle, type Locator, type Page } from "@playwright/test";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import {
  applyCompositionIntentInPlace,
  gotoEditor,
  revealModeIfPresent,
  startMuddyCollaboration,
  startMuddyCanvasDefault,
} from "./reference_apps_harness";

const QA_ROOT = path.resolve(__dirname, "../.tiller/scratch/codex/modern-editor-20260827/qa");
const CREATE_MESSAGE = "Landing page created with 3 starter sections. Edit its sections in Content › Pages.";
const SITE_NAVIGATOR = "[data-studio-site-navigator-panel='true']";
const SECTION_ORDER = "[data-gosx-studio-section-order='true']";
const SECTION_ITEM = "[data-gosx-studio-section-order-item]";
const SECTION_HANDLE = "[data-gosx-studio-section-order-handle='true']";
const SECTION_STATUS = "[data-gosx-studio-section-order-status='true']";
const OPERATION_POST = "/admin/editor/__actions/authoring";
const VIEWPORTS = [
  { label: "1600x900", width: 1600, height: 900 },
  { label: "1280x800", width: 1280, height: 800 },
  { label: "390x844", width: 390, height: 844 },
];
const DESKTOP_EDITOR_MODES = ["home", "look", "preview", "publish", "advanced"] as const;

test.describe("@reference-apps modern editor acceptance baseline", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("captures Muddy editor and Page CMS baseline screenshots plus UI inventory", async ({ page, request }) => {
    mkdirSync(QA_ROOT, { recursive: true });
    page.on("dialog", (dialog) => { void dialog.accept(); });

    // The acceptance surface must exercise Muddy's actual no-env production
    // default. Explicit canvas/co-render helpers remain useful in their
    // dedicated tests, but this inventory must not force MUDDY_SITEMAP_CANVAS.
    const server = await startMuddyCanvasDefault(request);
    try {
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");
      const createResult = await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: CREATE_MESSAGE,
        expectedChangeKind: "page",
        requireSelection: false,
      });
      const pageID = createResult.detail?.change?.pageKey;
      expect(pageID, "baseline needs a concrete Page CMS detail route").toBeTruthy();
      if (!pageID) throw new Error("baseline needs a concrete Page CMS detail route");
      await expect(page.locator(`${SITE_NAVIGATOR} a[href="/admin/pages/${pageID}"]`).first()).toBeAttached({ timeout: 30_000 });

      const screenshots: Record<string, string> = {};
      const viewportReports = [];
      for (const viewport of VIEWPORTS) {
        await page.setViewportSize({ width: viewport.width, height: viewport.height });
        await gotoEditor(page, server.baseURL);
        await revealModeIfPresent(page, "home");
        await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
        // Let managed navigation and mode selection finish before forcing all
        // actually scrollable editor panes back to their top framing. A
        // reset before this settle can be overwritten by the selected-section
        // reveal effect in the left rail.
        const homeScroll = await resetEditorScroll(page);
        const modebarFit = viewport.width >= 1280 ? await assertDesktopModebarControls(page, viewport.label) : undefined;
        screenshots[`home-${viewport.label}`] = await screenshot(page, `home-editor-${viewport.label}.png`);
        const homeFit = await layoutProbe(page);

        await revealModeIfPresent(page, "advanced");
        await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
        screenshots[`advanced-${viewport.label}`] = await screenshot(page, `advanced-editor-${viewport.label}.png`);
        const advancedFit = await layoutProbe(page);

        await page.goto(`${server.baseURL}/admin/pages/${pageID}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
        await ensurePageCMSControlsVisible(page);
        await selectFirstContentBlock(page);
        await resetEditorScroll(page);
        screenshots[`page-cms-${viewport.label}`] = await screenshot(page, `page-cms-${viewport.label}.png`);
        const firstEditableHeadingUnscrolled = await probeFirstEditableHeading(page);
        await page.locator("[data-content-block-id]").first().scrollIntoViewIfNeeded();
        const firstEditableHeading = await probeFirstEditableHeading(page);
        if (viewport.label === "1280x800") {
          expect(firstEditableHeadingUnscrolled.inputRect.bottom, "the unscrolled first editable Heading input must end within the 1280x800 viewport").toBeLessThanOrEqual(800);
        }
        if (viewport.label === "1280x800") {
          expect(firstEditableHeading.inputInsideViewport, "the first editable Heading input must fit fully inside the 1280x800 viewport after its evidence scroll").toBe(true);
        }
        screenshots[`page-cms-selected-first-block-${viewport.label}`] = await screenshot(page, `page-cms-selected-first-block-${viewport.label}.png`);
        await captureMediaAndDragStates(page, screenshots, viewport.label);
        const pageCMSFit = await layoutProbe(page);
        expect(pageCMSFit.hasHorizontalOverflow, `Page CMS ${viewport.label} must not overflow horizontally`).toBe(false);

        viewportReports.push({
          viewport: viewport.label,
          homeScroll,
          modebarFit,
          homeFit,
          advancedFit,
          pageCMSFit,
          firstEditableHeadingUnscrolled,
          firstEditableHeading,
        });
        expect(homeFit.hasHorizontalOverflow, `Home ${viewport.label} must not overflow horizontally`).toBe(false);
        expect(advancedFit.hasHorizontalOverflow, `Advanced ${viewport.label} must not overflow horizontally`).toBe(false);
      }

      await page.setViewportSize({ width: 1600, height: 900 });
      await page.goto(`${server.baseURL}/admin/pages/${pageID}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await ensurePageCMSControlsVisible(page);
      const inventory = await collectInventory(page, server.baseURL);
      const issues = baselineIssues(inventory, viewportReports);

      const report = {
        generatedAt: new Date().toISOString(),
        baseURL: server.baseURL,
        pageID,
        screenshots,
        inventory,
        viewportReports,
        issues,
      };
      const jsonPath = path.join(QA_ROOT, "baseline-inventory.json");
      const mdPath = path.join(QA_ROOT, "baseline-report.md");
      writeFileSync(jsonPath, JSON.stringify(report, null, 2));
      writeFileSync(mdPath, renderBaselineMarkdown(report));

      expect(Object.values(screenshots).every((file) => existsSync(file))).toBe(true);
      expect(inventory.pageCMS.contentBlocks, "Page CMS baseline should expose editable content blocks").toBeGreaterThan(0);
      expect(inventory.pageCMS.dragHandles, "Page CMS baseline should expose drag handles").toBeGreaterThan(0);
      expect(inventory.pageCMS.hasSearchControl, "Page CMS baseline should expose its local block search control").toBe(true);
      expect(inventory.pageCMS.hasUndoControl, "Page CMS baseline should render an Undo control even before the first edit").toBe(true);
      expect(inventory.pageCMS.hasRedoControl, "Page CMS baseline should render a Redo control even before the first edit").toBe(true);
    } finally {
      await server.stop();
    }
  });

  test("fresh dashboard Content navigation mounts Page and Blog editor runtimes", async ({ page, request }) => {
    const server = await startMuddyCanvasDefault(request);
    const routes: ManagedRoute[] = [
      {
        href: "/admin/pages",
        renderer: "[data-gosx-studio-backend-page-index-renderer='gosx-studio']",
        detailRenderer: "[data-gosx-studio-backend-page-detail-renderer='gosx-studio']",
        kind: "content",
        mediaList: "page-media-urls",
      },
      {
        href: "/admin/blog",
        renderer: "[data-gosx-studio-backend-blog-index-renderer='gosx-studio']",
        detailRenderer: "[data-gosx-studio-backend-blog-detail-renderer='gosx-studio']",
        kind: "content",
        mediaList: "blog-media-urls",
      },
      {
        href: "/admin/media",
        renderer: "[data-gosx-studio-backend-media-library-renderer='gosx-studio']",
        kind: "media",
      },
    ];
    try {
      for (const route of routes) {
        // A hard dashboard load resets the document globals before each
        // managed Content navigation, so a prior route cannot mask a missing
        // body-script mount.
        await page.goto(`${server.baseURL}/admin`, { waitUntil: "domcontentloaded", timeout: 60_000 });
        await page.waitForFunction(() => {
          const runtime = (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime;
          return !!runtime;
        }, undefined, { timeout: 30_000 });
        // Noni's shared admin layout intentionally preloads the media runtime
        // as a plain deferred script. Capture its object identity before the
        // managed route swap: the navigation loader may retain that singleton
        // and leave the route's inert pending placeholder in the body.
        const initialMediaRuntime = await page.evaluateHandle(() => {
          const runtime = (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime;
          return runtime || null;
        });
        const hasInitialMediaRuntime = await initialMediaRuntime.evaluate((runtime) => !!runtime);
        await clickManagedAdminContentRoute(page, server.baseURL, route.href);

        const renderer = page.locator(route.renderer).first();
        await expect(renderer, `${route.href} should render the GoSX-owned index`).toBeVisible();
        await assertManagedRouteRuntime(page, server.baseURL, route, true, hasInitialMediaRuntime ? initialMediaRuntime : undefined);
        const firstContentRuntime = route.kind === "content" ? await page.evaluateHandle(() => {
          const runtime = (window as unknown as { GoSXStudioContentEditorRuntime?: unknown }).GoSXStudioContentEditorRuntime;
          return runtime || null;
        }) : undefined;
        if (firstContentRuntime) {
          const hasContentRuntime = await firstContentRuntime.evaluate((runtime) => !!runtime);
          expect(hasContentRuntime, `${route.href} should expose a content runtime identity after first entry`).toBe(true);
        }

        // The index create form intentionally starts with an empty block
        // document. Exercise the non-persisting block-search and media-picker
        // controls on a real existing detail reached through the managed Edit
        // link, then return through the detail's managed index link. This
        // keeps the singleton/rebind proof grounded in the actual Page/Blog
        // editor rather than manufacturing blocks in an index fixture.
        if (route.kind === "content") {
          const detail = await openManagedContentDetail(page, server.baseURL, route);
          await assertContentRuntimeIdentity(page, firstContentRuntime, `${route.href} detail`);
          await exerciseManagedContentControls(page, detail.renderer, detail.pathname, detail.mediaInput);
          await clickManagedContentIndexLink(page, server.baseURL, route, detail.renderer);
          await expect(page.locator(route.renderer).first()).toBeVisible();
        }

        // A second visit through the real dashboard link must keep the one
        // runtime/global identity and one set of controls. GoSX may remove
        // the dynamically inserted head script on this cached revisit, so
        // only the first entry requires an executed script tag.
        await clickManagedAdminOverview(page, server.baseURL);
        await clickManagedAdminContentRoute(page, server.baseURL, route.href);
        await expect(page.locator(route.renderer).first()).toBeVisible();
        await assertManagedRouteRuntime(page, server.baseURL, route, false, hasInitialMediaRuntime ? initialMediaRuntime : undefined, firstContentRuntime || undefined);
        await initialMediaRuntime.dispose();
        await firstContentRuntime?.dispose();
      }
    } finally {
      await server.stop();
    }
  });

  test("default Home section ordering uses real input, durable history, draft save, and publish boundary", async ({ page, request }) => {
    const server = await startMuddyCanvasDefault(request);
    const operationRequests: string[] = [];
    page.on("dialog", (dialog) => { void dialog.accept(); });
    page.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes(OPERATION_POST)) operationRequests.push(request.url());
    });
    try {
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "home");
      await waitForSectionOrderReady(page);

      const lane = page.locator(SECTION_ORDER).first();
      const initialOrder = await sectionOrderKeys(page);
      expect(initialOrder.length, "the default fixture should expose the full durable section order").toBeGreaterThanOrEqual(4);
      const optionalItems = await lane.locator(`${SECTION_ITEM}[data-gosx-studio-section-order-preview-optional='true']`).count();
      expect(optionalItems, "the default fixture should retain at least one disabled/optional section in durable order").toBeGreaterThan(0);
      const previewKeys = await previewSectionKeys(page);
      expect(previewKeys.length, "the default Home preview should expose server-issued section identities").toBeGreaterThan(0);
      expect(previewKeys.every((key) => initialOrder.includes(key))).toBe(true);

      const publicBefore = await publicSectionKeys(request, server.baseURL);
      expect(publicBefore.length, "the public Home route should expose section order for the draft/public boundary check").toBeGreaterThan(0);

      const movable = await enabledSectionKeys(page);
      expect(movable.length, "at least two rendered Home sections should be movable").toBeGreaterThanOrEqual(2);
      const sourceKey = movable[1];
      const targetKey = movable[0];
      await dragSectionWithMouse(page, sourceKey, targetKey, "top");
      await expect(lane.locator(SECTION_STATUS)).toHaveText("Section order saved.", { timeout: 30_000 });
      await waitForSectionOrderReady(page);
      const afterDrag = await sectionOrderKeys(page);
      expect(afterDrag).not.toEqual(initialOrder);
      expect(operationRequests.length, "a real Home mouse drag should commit exactly one authoring operation").toBeGreaterThan(0);
      expect(await publicSectionKeys(request, server.baseURL), "a draft reorder must not change the public storefront before publish").toEqual(publicBefore);

      // Escape cancellation while a pointer drag is active must not post or
      // change the lane. The focused handle is a real keyboard target; the
      // pointer remains down until after Escape is delivered.
      const beforeEscape = await sectionOrderKeys(page);
      const escapeKey = (await enabledSectionKeys(page))[0];
      const escapeHandle = lane.locator(`${SECTION_ITEM}[data-gosx-studio-section-order-item='${escapeKey}'] ${SECTION_HANDLE}`);
      await beginSectionPointerDrag(page, escapeHandle);
      const requestsBeforeEscape = operationRequests.length;
      await escapeHandle.focus();
      await page.keyboard.press("Escape");
      await page.mouse.up();
      await expect(lane.locator(SECTION_STATUS)).toHaveText("Section move canceled.");
      expect(operationRequests.length, "Escape cancellation must not send an authoring request").toBe(requestsBeforeEscape);
      expect(await sectionOrderKeys(page), "Escape cancellation must preserve the durable lane order").toEqual(beforeEscape);

      // Releasing outside the lane with no target is the other cancel path.
      const beforeOutside = await sectionOrderKeys(page);
      const outsideKey = (await enabledSectionKeys(page))[0];
      const outsideHandle = lane.locator(`${SECTION_ITEM}[data-gosx-studio-section-order-item='${outsideKey}'] ${SECTION_HANDLE}`);
      await beginSectionPointerDrag(page, outsideHandle);
      const requestsBeforeOutside = operationRequests.length;
      await page.mouse.move(1, 1, { steps: 3 });
      await page.mouse.up();
      await expect(lane.locator(SECTION_STATUS)).toHaveText("Section move canceled.");
      expect(operationRequests.length, "dropping outside the section lane must not send an authoring request").toBe(requestsBeforeOutside);
      expect(await sectionOrderKeys(page), "outside cancellation must preserve the durable lane order").toEqual(beforeOutside);

      // Click and keyboard alternatives are exercised on fresh enabled rows,
      // not by dispatching synthetic custom events.
      await waitForSectionOrderReady(page);
      const clickKey = (await enabledSectionKeys(page))[0];
      const clickBefore = await sectionOrderKeys(page);
      const clickRow = lane.locator(`${SECTION_ITEM}[data-gosx-studio-section-order-item='${clickKey}']`);
      await clickRow.locator("[data-gosx-studio-section-order-move='down']").click();
      await expect(lane.locator(SECTION_STATUS)).toHaveText("Section moved down.", { timeout: 30_000 });
      await waitForSectionOrderReady(page);
      const afterClick = await sectionOrderKeys(page);
      expect(afterClick).not.toEqual(clickBefore);

      const keyboardCandidates = await enabledSectionKeys(page);
      const keyboardKey = keyboardCandidates.find((key) => afterClick.indexOf(key) > 0);
      expect(keyboardKey, "keyboard reorder needs an enabled non-first section").toBeTruthy();
      if (!keyboardKey) throw new Error("keyboard reorder needs an enabled non-first section");
      const keyboardBefore = await sectionOrderKeys(page);
      const keyboardHandle = lane.locator(`${SECTION_ITEM}[data-gosx-studio-section-order-item='${keyboardKey}'] ${SECTION_HANDLE}`);
      await keyboardHandle.focus();
      await page.keyboard.press("Space");
      await page.keyboard.press("ArrowUp");
      await page.keyboard.press("Enter");
      await expect(lane.locator(SECTION_STATUS)).toHaveText("Section order saved.", { timeout: 30_000 });
      await waitForSectionOrderReady(page);
      const afterKeyboard = await sectionOrderKeys(page);
      expect(afterKeyboard).not.toEqual(keyboardBefore);

      // Durable Undo/Redo must restore the exact previous order through the
      // real authoring endpoint, then reapply it, without a test response
      // override.
      const undo = lane.locator("[data-gosx-studio-section-order-history='undo']");
      const redo = lane.locator("[data-gosx-studio-section-order-history='redo']");
      await expect(undo, "a committed reorder must enable durable Undo").toBeEnabled({ timeout: 30_000 });
      await undo.click();
      await expect(lane.locator(SECTION_STATUS)).toHaveText("Section reorder history applied.", { timeout: 30_000 });
      await waitForSectionOrderReady(page);
      expect(await sectionOrderKeys(page), "Undo should restore the order before the last keyboard move").toEqual(keyboardBefore);
      await expect(redo, "Undo must expose durable Redo").toBeEnabled({ timeout: 30_000 });
      await redo.click();
      await expect(lane.locator(SECTION_STATUS)).toHaveText("Section reorder history applied.", { timeout: 30_000 });
      await waitForSectionOrderReady(page);
      expect(await sectionOrderKeys(page), "Redo should reapply the keyboard move").toEqual(afterKeyboard);

      // A normal Save changes submit must not clobber the durable section
      // order. It is intentionally a real form submission, not a mocked
      // response or direct store mutation.
      await submitOrdinaryEditorSave(page, server.baseURL);
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "home");
      await waitForSectionOrderReady(page);
      expect(await sectionOrderKeys(page), "ordinary Save changes must preserve the durable reorder").toEqual(afterKeyboard);
      expect(await publicSectionKeys(request, server.baseURL), "ordinary Save changes must not publish the draft reorder").toEqual(publicBefore);

      // /about is a safe read-only route for this Home-only order target. A
      // disabled lane must remain unchanged and send no operation.
      const aboutRoute = page.locator("[data-studio-preview-route-key='about']").first();
      await expect(aboutRoute).toBeVisible();
      await aboutRoute.click();
      await waitForPreviewRoute(page, "/about");
      await expect(lane).toHaveAttribute("data-gosx-studio-section-order-enabled", "false", { timeout: 30_000 });
      await expect(lane.locator(SECTION_STATUS)).toContainText("configured source route", { timeout: 30_000 });
      const aboutBefore = await sectionOrderKeys(page);
      const requestsBeforeAbout = operationRequests.length;
      await expect(lane.locator(SECTION_HANDLE).first()).toBeDisabled();
      await lane.locator("[data-gosx-studio-section-order-move='down']").first().click({ force: true });
      expect(operationRequests.length, "/about read-only mode must not send a reorder operation").toBe(requestsBeforeAbout);
      expect(await sectionOrderKeys(page), "/about read-only mode must preserve the Home order").toEqual(aboutBefore);

      // Publish only after the draft/public assertions. The public route may
      // then adopt the durable order, proving the boundary rather than merely
      // checking that the draft endpoint returned 200.
      await page.locator("[data-studio-preview-route-key='home']").click();
      await waitForSectionOrderReady(page);
      await publishEditorDraft(page);
      const publicAfterPublish = await publicSectionKeys(request, server.baseURL);
      expect(publicAfterPublish).toEqual(afterKeyboard.filter((key) => publicBefore.includes(key)));
    } finally {
      await server.stop();
    }
  });

  test("Page CMS uses real block/media input, truthful offline/validation states, and draft preview boundary", async ({ page, request }) => {
    // Draft preview requires an actual signed admin session; the plain
    // MUDDY_MOCK_AUTH fallback does not satisfy the storefront preview guard.
    // Collaboration mode has no canvas override, so this remains the real
    // default editor path while using the test-only signed-auth endpoint.
    const server = await startMuddyCollaboration(request);
    const saveRequests: string[] = [];
    page.on("request", (request) => {
      if (request.method() === "POST" && (/\/admin\/pages\/[^/]+\/__actions\/save/.test(request.url()) || request.url().endsWith("/admin/media/upload"))) {
        saveRequests.push(request.url());
      }
    });
    try {
      await page.goto(`${server.baseURL}/__test/collaboration/signin/admin`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");
      const createResult = await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: CREATE_MESSAGE,
        expectedChangeKind: "page",
        requireSelection: false,
      });
      const pageID = createResult.detail?.change?.pageKey;
      expect(pageID, "real Page CMS flow needs the created page id").toBeTruthy();
      if (!pageID) throw new Error("real Page CMS flow needs the created page id");

      await page.goto(`${server.baseURL}/admin/pages/${pageID}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await ensurePageCMSControlsVisible(page);
      const editor = page.locator("[data-content-editor]").first();
      await expect(editor).toHaveAttribute("data-content-editor-save-state", "idle");
      const initialIDs = await contentBlockIDs(page);
      expect(initialIDs).toEqual(["heading-1", "paragraph-2", "paragraph-3", "button-4"]);
      await expect(editor.locator("[data-content-editor-action='undo']")).toHaveCount(1);
      await expect(editor.locator("[data-content-editor-action='redo']")).toHaveCount(1);

      // Search is scoped to the Page CMS editor itself. Filtering one known
      // paragraph must change only the visible rows; clearing it must restore
      // the exact source/order without a save or hidden global Find control.
      const blockSearch = editor.locator("[data-content-editor-search='true']");
      await expect(blockSearch, "Page CMS should expose a local Search blocks field").toBeVisible();
      const sourceBeforeSearch = await page.locator("textarea[name='body']").inputValue();
      await blockSearch.fill("warm welcome");
      await expect(editor.locator("[data-content-editor-search-count='true']")).toHaveText("1 of 4 blocks");
      await expect(page.locator("[data-content-block-id]")).toHaveCount(1);
      await expect(page.locator("[data-content-block-type='paragraph']")).toHaveCount(1);
      await blockSearch.fill("");
      await expect(editor.locator("[data-content-editor-search-count='true']")).toHaveText("4 blocks");
      await expect(page.locator("[data-content-block-id]")).toHaveCount(initialIDs.length);
      expect(await contentBlockIDs(page), "clearing block search must restore all rows in their source order").toEqual(initialIDs);
      expect(await page.locator("textarea[name='body']").inputValue(), "search filtering must not rewrite the source").toBe(sourceBeforeSearch);

      // Native input undo is deliberately separate from block-structure
      // history. It must restore the text control while keeping the editor's
      // source in sync.
      const heading = page.locator("[data-content-block-id='heading-1']");
      await heading.click();
      const headingInput = heading.locator(".content-block__fields label", { hasText: "Heading" }).locator("input").first();
      const originalHeading = await headingInput.inputValue();
      const typedHeading = `Luna native undo ${Date.now()}`;
      await headingInput.fill(typedHeading);
      await expect(heading.locator(".content-block__preview h3")).toHaveText(typedHeading);
      await headingInput.press("Control+Z");
      await expect(headingInput, "native Control+Z must restore the heading input").toHaveValue(originalHeading);
      expect(JSON.parse(await page.locator("textarea[name='body']").inputValue()).blocks[0].text).toBe(originalHeading);

      // The HTML5 drag-event/DataTransfer protocol path commits a local
      // reorder; a canceled protocol drag leaves it unchanged and does not
      // POST until the operator chooses Save draft. Physical pointer proof
      // belongs to the separate mouse-drag gate.
      const firstHandle = page.locator(`[data-content-block-id='${initialIDs[0]}'] [data-content-drag-handle='true']`);
      const secondBlock = page.locator(`[data-content-block-id='${initialIDs[1]}']`);
      await dragContentBlock(page, firstHandle, secondBlock, "bottom");
      const afterDrag = await contentBlockIDs(page);
      expect(afterDrag).not.toEqual(initialIDs);
      const beforeCancel = await contentBlockIDs(page);
      await cancelContentDrag(page, afterDrag[0], afterDrag[1]);
      expect(await contentBlockIDs(page), "canceled Page CMS drag must preserve local order").toEqual(beforeCancel);
      expect(saveRequests.filter((url) => url.includes("/admin/pages/")).length, "local Page CMS operations must not save implicitly").toBe(0);

      // Insert, duplicate, delete, and media search/selection all use live
      // controls in the real page detail DOM.
      await page.locator("button[data-content-add='quote']").click();
      await expect(page.locator("[data-content-block-type='quote']").last()).toBeVisible();
      const afterInsert = await contentBlockIDs(page);
      expect(afterInsert).toHaveLength(beforeCancel.length + 1);
      const inserted = page.locator("[data-content-block-type='quote']").last();
      await inserted.locator("[data-content-editor-action='duplicate']").click();
      const afterDuplicate = await contentBlockIDs(page);
      expect(afterDuplicate).toHaveLength(afterInsert.length + 1);
      const duplicateID = afterDuplicate.find((id) => !afterInsert.includes(id));
      expect(duplicateID, "duplicating a Quote should create one new block id").toBeTruthy();
      if (!duplicateID) throw new Error("duplicating a Quote should create one new block id");
      await page.locator(`[data-content-block-id='${duplicateID}'] [data-content-editor-action='delete']`).click();
      expect(await contentBlockIDs(page)).toEqual(afterInsert);

      const metaInput = page.locator("input[name='metaImageUrl']");
      const metaPicker = page.locator("input[name='metaImageUrl'] + .media-picker");
      await expect(metaPicker).toHaveCount(1);
      await metaPicker.locator(".media-picker__trigger").click();
      const mediaSearch = metaPicker.locator(".media-picker__search");
      await mediaSearch.fill("moss");
      await expect(metaPicker.locator(".media-picker__asset")).toHaveCount(1);
      await metaPicker.locator(".media-picker__asset").click();
      // The fixture's seeded media uses the canonical Unsplash URL as its
      // stored value; the human filename is the picker label, not part of
      // the submitted URL.
      await expect(metaInput).toHaveValue(/photo-1565193566173-7a0ee3dbe261/);
      await expect(page.locator("input[name='metaImageAlt']")).not.toHaveValue("");

      // Drag a real media-picker asset into the content list. media-runtime
      // owns the custom DataTransfer payload and content-editor consumes it.
      await page.locator("button[data-content-add='image']").click();
      const imageTarget = page.locator("[data-content-block-id]").first();
      await metaPicker.locator(".media-picker__trigger").click();
      const mediaAsset = metaPicker.locator(".media-picker__asset").first();
      const mediaTargetBox = await imageTarget.boundingBox();
      if (!mediaTargetBox) throw new Error("Page CMS media insertion requires visible content target geometry");
      const mediaTransfer = await page.evaluateHandle(() => new DataTransfer());
      // Playwright's headless Chromium dragTo does not reliably preserve the
      // media runtime's custom DataTransfer payload. Dispatch the same
      // bubbling drag sequence through both real runtime listeners instead.
      await mediaAsset.dispatchEvent("dragstart", { dataTransfer: mediaTransfer });
      await imageTarget.dispatchEvent("dragenter", {
        bubbles: true,
        cancelable: true,
        clientX: mediaTargetBox.x + mediaTargetBox.width / 2,
        clientY: mediaTargetBox.y + 4,
        dataTransfer: mediaTransfer,
      });
      await imageTarget.dispatchEvent("dragover", {
        bubbles: true,
        cancelable: true,
        clientX: mediaTargetBox.x + mediaTargetBox.width / 2,
        clientY: mediaTargetBox.y + 4,
        dataTransfer: mediaTransfer,
      });
      await imageTarget.dispatchEvent("drop", {
        bubbles: true,
        cancelable: true,
        clientX: mediaTargetBox.x + mediaTargetBox.width / 2,
        clientY: mediaTargetBox.y + 4,
        dataTransfer: mediaTransfer,
      });
      await mediaAsset.dispatchEvent("dragend", { dataTransfer: mediaTransfer });
      const mediaBlocks = page.locator("[data-content-block-type='image']");
      await expect(mediaBlocks.last()).toBeVisible();
      const bodyAfterMedia = JSON.parse(await page.locator("textarea[name='body']").inputValue());
      expect(bodyAfterMedia.blocks.some((block: { type?: string; url?: string }) => block.type === "image" && !!block.url)).toBe(true);

      // The actual Media Library drop zone must select a file before upload;
      // dropping alone is not a silent save.
      await page.goto(`${server.baseURL}/admin/media`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const dropzone = page.locator("[data-media-upload-drop-zone='true']").first();
      await expect(dropzone).toBeVisible();
      await page.evaluate(() => {
        const zone = document.querySelector("[data-media-upload-drop-zone='true']");
        if (!(zone instanceof HTMLElement)) throw new Error("media upload drop zone missing");
        const transfer = new DataTransfer();
        // Use a real 1x1 PNG payload: the GoSX upload action validates the
        // submitted bytes, so a text body with an image MIME type is rightly
        // rejected even though the browser picker can preview it.
        const encoded = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=";
        const bytes = Uint8Array.from(atob(encoded), (character) => character.charCodeAt(0));
        transfer.items.add(new File([bytes], "luna-browser-drop.png", { type: "image/png" }));
        zone.dispatchEvent(new DragEvent("dragenter", { bubbles: true, cancelable: true, dataTransfer: transfer }));
        zone.dispatchEvent(new DragEvent("dragover", { bubbles: true, cancelable: true, dataTransfer: transfer }));
        zone.dispatchEvent(new DragEvent("drop", { bubbles: true, cancelable: true, dataTransfer: transfer }));
      });
      await expect(page.locator("[data-media-upload-status]")).toContainText("luna-browser-drop.png selected");
      await expect(page.locator("[data-media-upload]")).toHaveAttribute("data-media-upload-bound", "true");
      const uploadResponse = page.waitForResponse((response) => response.url().endsWith("/admin/media/upload") && response.request().method() === "POST");
      await page.locator("form[action='/admin/media/upload']").getByRole("button", { name: "Upload file", exact: true }).click();
      const upload = await uploadResponse;
      expect([200, 303]).toContain(upload.status());
      await page.waitForURL(/\/admin\/media(?:\?uploaded=1)?$/, { timeout: 30_000 }).catch(() => {});
      await expect(page.locator("body")).toContainText("Upload saved to the media library.");
      // GoSX prefixes/suffixes uploaded filenames for collision-safe storage;
      // retain the fixture stem as the server-side identity assertion.
      await expect(page.locator(".media-card", { hasText: "luna-browser-drop" })).toHaveCount(1);

      // Return to the page detail and exercise truthful offline + real server
      // validation. No mocked fetch or fabricated success response is used.
      await page.goto(`${server.baseURL}/admin/pages/${pageID}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await ensurePageCMSControlsVisible(page);
      const currentHeading = page.locator("[data-content-block-id]").first().locator(".content-block__fields input").first();
      const draftMarker = `Luna offline draft ${Date.now()}`;
      await currentHeading.fill(draftMarker);
      await page.evaluate(() => Object.defineProperty(window.navigator, "onLine", { configurable: true, value: false }));
      await page.locator("button", { hasText: "Save draft" }).first().click();
      await expect(editor).toHaveAttribute("data-content-editor-save-state", "offline");
      await expect(editor).toHaveAttribute("data-content-editor-dirty", "true");
      await expect(editor.locator("[data-content-editor-save-status]"), "offline save must be announced accessibly").toHaveAttribute("role", "alert");
      await expect(page.locator("textarea[name='body']")).toHaveValue(new RegExp(draftMarker));
      expect(saveRequests.filter((url) => url.includes("/admin/pages/")).length, "offline Save draft must not POST").toBe(0);

      await page.evaluate(() => Object.defineProperty(window.navigator, "onLine", { configurable: true, value: true }));
      await page.locator("input[name='title']").fill("");
      const validationResponse = page.waitForResponse((response) => response.url().includes(`/admin/pages/${pageID}/__actions/save`) && response.request().method() === "POST");
      await page.locator("button", { hasText: "Save draft" }).first().click();
      const validation = await validationResponse;
      expect([200, 422, 303]).toContain(validation.status());
      await expect(editor).toHaveAttribute("data-content-editor-save-state", "failed");
      await expect(editor).toHaveAttribute("data-content-editor-dirty", "true");
      await expect(page.locator("textarea[name='body']")).toHaveValue(new RegExp(draftMarker));
      await expect(page.locator("[data-content-editor-save-status]"), "validation failure must remain visible in the editor status").toHaveAttribute("role", "alert");

      const validTitle = `Luna saved ${Date.now()}`;
      await page.locator("input[name='title']").fill(validTitle);
      await page.locator("input[name='published']").uncheck();
      const saveResponse = page.waitForResponse((response) => response.url().includes(`/admin/pages/${pageID}/__actions/save`) && response.request().method() === "POST");
      await page.locator("button", { hasText: "Save draft" }).first().click();
      const saved = await saveResponse;
      expect([200, 303]).toContain(saved.status());
      await page.waitForURL(new RegExp(`/admin/pages/${pageID}$`), { timeout: 30_000 });
      await expect(page.locator("input[name='title']")).toHaveValue(validTitle);
      await expect(page.locator("textarea[name='body']")).toHaveValue(new RegExp(draftMarker));

      const publicPage = await request.get(`${server.baseURL}/pages/new-page`);
      expect(publicPage.status(), "a saved but unpublished Page CMS draft must remain absent from the public route").toBe(404);
      const previewLink = page.locator("form.admin-form a.button", { hasText: "Preview" }).first();
      await expect(previewLink).toBeVisible();
      await Promise.all([
        page.waitForURL(/\/admin\/storefront\?path=/),
        previewLink.click(),
      ]);
      await expect(page.frameLocator("iframe.storefront-frame").first().locator("body")).toContainText(draftMarker);

      // A clean reload proves the successful server response persisted the
      // body marker; the preview assertion above proves the draft route sees
      // it while public remains 404.
      await page.goto(`${server.baseURL}/admin/pages/${pageID}`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("input[name='title']")).toHaveValue(validTitle);
      await expect(page.locator("textarea[name='body']")).toHaveValue(new RegExp(draftMarker));

      // Exercise the managed navigation lifecycle with real Content › Pages
      // links. This catches stale page-canvas listeners/detached forms after
      // Page CMS -> Pages index -> Page CMS -> Release Center transitions.
      const requestsBeforeManagedNavigation = saveRequests.length;
      const detailForm = page.locator("form.admin-form").first();
      const backToPages = detailForm.locator("a[data-gosx-link='true']", { hasText: "Back to pages" }).first();
      await expect(backToPages, "Page CMS should expose a managed Back to pages link").toBeVisible();
      await Promise.all([
        page.waitForURL(/\/admin\/pages$/),
        backToPages.click(),
      ]);
      // GoSX managed navigation canonicalizes hrefs to absolute same-origin
      // URLs. Accept either canonical absolute or ordinary relative markup,
      // but keep the exact pathname/origin assertion so a foreign or wrong
      // page link cannot satisfy this lifecycle check.
      const createdPagePath = `/admin/pages/${pageID}`;
      const createdPageLink = page.locator(`a[href$="${createdPagePath}"]`).first();
      await expect(createdPageLink, "managed Pages index should retain the created page link").toBeVisible();
      const createdPageHref = await createdPageLink.getAttribute("href");
      const createdPageURL = new URL(createdPageHref ?? "", server.baseURL);
      expect(createdPageURL.origin, "managed created-page link must stay same-origin").toBe(new URL(server.baseURL).origin);
      expect(createdPageURL.pathname, "managed created-page link must target the created page").toBe(createdPagePath);
      await Promise.all([
        page.waitForURL(new RegExp(`/admin/pages/${pageID}$`)),
        createdPageLink.click(),
      ]);
      const openEditorToPublish = page.locator("form.admin-form").first().locator("a[data-gosx-link='true']", { hasText: /Open editor to publish|Publish changes/ }).first();
      await expect(openEditorToPublish, "Page CMS should expose the managed Release Center link").toBeVisible();
      await Promise.all([
        page.waitForURL(/\/admin\/editor(?:\?mode=publish)?$/),
        openEditorToPublish.click(),
      ]);
      await revealModeIfPresent(page, "home");
      await waitForSectionOrderReady(page);
      expect(saveRequests.length, "managed editor navigation must not duplicate Page CMS save/upload POSTs").toBe(requestsBeforeManagedNavigation);
    } finally {
      await server.stop();
    }
  });
});

async function waitForSectionOrderReady(page: Page) {
  const lane = page.locator(SECTION_ORDER).first();
  await expect(lane, "Home should render one durable section-order lane").toBeAttached({ timeout: 30_000 });
  await expect(lane).toHaveAttribute("data-gosx-studio-section-order-enabled", "true", { timeout: 30_000 });
  await expect.poll(
    () => lane.locator(`${SECTION_ITEM} ${SECTION_HANDLE}:not([disabled])`).count(),
    { timeout: 30_000, message: "at least two rendered Home sections should have usable drag handles" },
  ).toBeGreaterThanOrEqual(2);
}

async function clickManagedAdminContentRoute(page: Page, baseURL: string, href: string) {
  const contentSummary = page.locator("nav[aria-label='Main'] details summary", { hasText: "Content" }).first();
  if (await contentSummary.count()) {
    const menu = contentSummary.locator("xpath=..");
    if ((await menu.getAttribute("open")) === null) await contentSummary.click();
  }
  await clickManagedAdminRoute(page, baseURL, href);
}

async function clickManagedAdminOverview(page: Page, baseURL: string) {
  await clickManagedAdminRoute(page, baseURL, "/admin");
}

async function clickManagedAdminRoute(page: Page, baseURL: string, href: string) {
  // GoSX canonicalizes managed links to absolute same-origin URLs. Select by
  // pathname suffix, then validate both origin and pathname before clicking;
  // a foreign or wrong absolute URL must never satisfy this navigation gate.
  const link = page.locator(`nav[aria-label='Main'] a[data-gosx-link='true'][href$='${href}']`).first();
  await expect(link, `dashboard should expose managed ${href} navigation`).toBeVisible();
  const linkHref = await link.getAttribute("href");
  const linkURL = new URL(linkHref ?? "", baseURL);
  expect(linkURL.origin, `${href} navigation must stay same-origin`).toBe(new URL(baseURL).origin);
  expect(linkURL.pathname, `${href} navigation must retain its canonical pathname`).toBe(href);
  await Promise.all([
    page.waitForURL((url) => url.origin === new URL(baseURL).origin && url.pathname === href, { timeout: 30_000 }),
    link.click(),
  ]);
}

async function openManagedContentDetail(page: Page, baseURL: string, route: ManagedRoute): Promise<{ renderer: Locator; mediaInput: Locator; pathname: string }> {
  if (!route.detailRenderer || !route.mediaList) throw new Error(`${route.href} content route is missing its detail renderer or media datalist id`);
  const indexRenderer = page.locator(route.renderer).first();
  const detailLink = indexRenderer.locator(`a[data-gosx-link='true'][href*='${route.href}/']`).first();
  await expect(detailLink, `${route.href} index should expose an existing managed Edit link`).toBeVisible();
  const linkHref = await detailLink.getAttribute("href");
  const detailURL = new URL(linkHref ?? "", baseURL);
  expect(detailURL.origin, `${route.href} detail navigation must stay same-origin`).toBe(new URL(baseURL).origin);
  expect(detailURL.pathname.startsWith(`${route.href}/`), `${route.href} Edit link must target a detail pathname`).toBe(true);
  await Promise.all([
    page.waitForURL((url) => url.origin === new URL(baseURL).origin && url.pathname === detailURL.pathname, { timeout: 30_000 }),
    detailLink.click(),
  ]);

  const detailRenderer = page.locator(route.detailRenderer).first();
  await expect(detailRenderer, `${route.href} managed Edit link should render the GoSX-owned detail`).toBeVisible();
  const editor = detailRenderer.locator("[data-content-editor='true']");
  await expect(editor, `${detailURL.pathname} should expose one detail content editor`).toHaveCount(1);
  await expect(editor.first(), `${detailURL.pathname} detail editor should bind exactly once`).toHaveAttribute("data-content-editor-bound", "true", { timeout: 30_000 });
  const mediaInput = detailRenderer.locator(`input[name='metaImageUrl'][list='${route.mediaList}']`).first();
  await expect(mediaInput, `${detailURL.pathname} should expose its detail media picker input`).toHaveCount(1);
  await expect(mediaInput, `${detailURL.pathname} detail media picker should bind`).toHaveAttribute("data-media-picker-bound", "true", { timeout: 30_000 });
  return { renderer: detailRenderer, mediaInput, pathname: detailURL.pathname };
}

async function clickManagedContentIndexLink(page: Page, baseURL: string, route: ManagedRoute, detailRenderer: Locator) {
  const link = detailRenderer.locator("a[data-gosx-link='true']").filter({ hasText: /Back to pages|Back to blog|Cancel/ }).first();
  await expect(link, `${route.href} detail should expose a managed link back to its index`).toBeVisible();
  const linkHref = await link.getAttribute("href");
  const indexURL = new URL(linkHref ?? "", baseURL);
  expect(indexURL.origin, `${route.href} detail return must stay same-origin`).toBe(new URL(baseURL).origin);
  expect(indexURL.pathname, `${route.href} detail return must target its index`).toBe(route.href);
  await Promise.all([
    page.waitForURL((url) => url.origin === new URL(baseURL).origin && url.pathname === route.href, { timeout: 30_000 }),
    link.click(),
  ]);
}

async function assertDesktopModebarControls(page: Page, viewportLabel: string) {
  const modebar = page.locator(".studio-modebar").first();
  await expect(modebar, `${viewportLabel} should render the shared editor modebar`).toBeVisible();
  await expect(modebar.locator("[data-studio-mode-control]")).toHaveCount(DESKTOP_EDITOR_MODES.length);
  const bar = await modebar.boundingBox();
  if (!bar) throw new Error(`${viewportLabel} modebar should expose measurable geometry`);

  const controls = [];
  for (const mode of DESKTOP_EDITOR_MODES) {
    const button = modebar.locator(`[data-studio-mode-control='${mode}']`).first();
    await expect(button, `${viewportLabel} ${mode} mode control should be visible`).toBeVisible();
    const geometry = await button.evaluate((element) => {
      const rect = element.getBoundingClientRect();
      const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
      const hitByButton = hit === element || element.contains(hit);
      return {
        rect: { left: rect.left, top: rect.top, right: rect.right, bottom: rect.bottom, width: rect.width, height: rect.height },
        hitByButton,
        hitTag: hit?.tagName || "",
        hitText: (hit?.textContent || "").replace(/\s+/g, " ").trim().slice(0, 80),
      };
    });
    expect(geometry.rect.left, `${viewportLabel} ${mode} control must not begin offscreen`).toBeGreaterThanOrEqual(0);
    expect(geometry.rect.top, `${viewportLabel} ${mode} control must not begin offscreen`).toBeGreaterThanOrEqual(0);
    expect(geometry.rect.right, `${viewportLabel} ${mode} control must fit the viewport width`).toBeLessThanOrEqual(page.viewportSize()?.width ?? 0);
    expect(geometry.rect.bottom, `${viewportLabel} ${mode} control must fit the viewport height`).toBeLessThanOrEqual(page.viewportSize()?.height ?? 0);
    expect(geometry.rect.left, `${viewportLabel} ${mode} control must fit inside the modebar`).toBeGreaterThanOrEqual(bar.x);
    expect(geometry.rect.top, `${viewportLabel} ${mode} control must fit inside the modebar`).toBeGreaterThanOrEqual(bar.y);
    expect(geometry.rect.right, `${viewportLabel} ${mode} control must fit inside the modebar`).toBeLessThanOrEqual(bar.x + bar.width);
    expect(geometry.rect.bottom, `${viewportLabel} ${mode} control must fit inside the modebar`).toBeLessThanOrEqual(bar.y + bar.height);
    expect(geometry.hitByButton, `${viewportLabel} ${mode} control center must not be obscured (hit=${geometry.hitTag}:${geometry.hitText})`).toBe(true);
    controls.push({ mode, ...geometry });
  }
  return {
    viewport: viewportLabel,
    modebar: { left: bar.x, top: bar.y, right: bar.x + bar.width, bottom: bar.y + bar.height, width: bar.width, height: bar.height },
    controls,
  };
}

type ManagedRoute = {
  href: string;
  renderer: string;
  detailRenderer?: string;
  kind: "content" | "media";
  mediaList?: string;
};

async function assertManagedRouteRuntime(page: Page, baseURL: string, route: ManagedRoute, requireExecutedScript: boolean, initialMediaRuntime?: JSHandle<unknown>, firstContentRuntime?: JSHandle<unknown>) {
  if (route.kind === "content") {
    const renderer = page.locator(route.renderer).first();
    const editors = renderer.locator("[data-content-editor='true']");
    await expect(editors, `${route.href} should expose one shared content editor`).toHaveCount(1);
    await expect(editors.first(), `${route.href} content editor should bind exactly once`).toHaveAttribute("data-content-editor-bound", "true", { timeout: 30_000 });
    await page.waitForFunction(() => {
      const runtime = (window as unknown as { GoSXStudioContentEditorRuntime?: unknown }).GoSXStudioContentEditorRuntime;
      return !!runtime;
    }, undefined, { timeout: 30_000 });
    await assertContentRuntimeIdentity(page, firstContentRuntime, route.href);
    await assertManagedRuntimeScript(page, baseURL, "/_gosx/studio/content-editor.js", requireExecutedScript);

    const mediaList = route.mediaList;
    if (!mediaList) throw new Error(`${route.href} content route is missing its media datalist id`);
    const mediaInputs = renderer.locator(`input[name='metaImageUrl'][list='${mediaList}']`);
    await expect(mediaInputs, `${route.href} should expose one media picker input`).toHaveCount(1);
    await expect(mediaInputs.first(), `${route.href} should mount the media picker after managed navigation`).toHaveAttribute("data-media-picker-bound", "true", { timeout: 30_000 });
    await expect(mediaInputs.first().locator("xpath=following-sibling::*[1]")).toHaveClass(/media-picker/);
    await page.waitForFunction(() => {
      const runtime = (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime;
      return !!runtime;
    }, undefined, { timeout: 30_000 });
    await assertMediaRuntimeIdentity(page, initialMediaRuntime, route.href);
    if (!initialMediaRuntime) {
      await assertManagedRuntimeScript(page, baseURL, "/_gosx/studio/media-runtime.js", requireExecutedScript);
    }
    return;
  }

  const renderer = page.locator(route.renderer).first();
  const uploadRoots = renderer.locator("[data-media-upload='true']");
  await expect(uploadRoots, "Media should expose one upload binding root").toHaveCount(1);
  await expect(uploadRoots.first(), "Media upload root should bind after managed navigation").toHaveAttribute("data-media-upload-bound", "true", { timeout: 30_000 });
  await expect(renderer.locator("input[data-media-upload-input='true'], input[type='file']")).toHaveCount(1);
    await page.waitForFunction(() => {
      const runtime = (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime;
      return !!runtime;
    }, undefined, { timeout: 30_000 });
    await assertMediaRuntimeIdentity(page, initialMediaRuntime, route.href);
  if (!initialMediaRuntime) {
    await assertManagedRuntimeScript(page, baseURL, "/_gosx/studio/media-runtime.js", requireExecutedScript);
  }
}

async function exerciseManagedContentControls(page: Page, renderer: Locator, routeHref: string, mediaInput: Locator) {
  const editor = renderer.locator("[data-content-editor='true']").first();
  const initialIDs = await contentBlockIDs(page);
  const paragraphBlocks = editor.locator("[data-content-block-type='paragraph']");
  const paragraphCount = await paragraphBlocks.count();
  expect(paragraphCount, `${routeHref} detail should expose an existing paragraph block for the local search proof`).toBeGreaterThan(0);
  const search = editor.locator("[data-content-editor-search='true']").first();
  await expect(search, `${routeHref} should expose the local block search input`).toHaveCount(1);
  await search.fill("paragraph");
  await expect(editor, `${routeHref} local search should update query state`).toHaveAttribute("data-content-editor-search-query", "paragraph");
  await expect(editor, `${routeHref} local search should report a filtered block count`).toHaveAttribute("data-content-editor-search-visible-count", String(paragraphCount));
  await search.fill("");
  await expect(editor).not.toHaveAttribute("data-content-editor-search-query");
  expect(await contentBlockIDs(page), `${routeHref} search/clear must not alter local block order or source IDs`).toEqual(initialIDs);

  const picker = mediaInput.locator("xpath=following-sibling::*[1]");
  const trigger = picker.locator(".media-picker__trigger").first();
  await expect(trigger, `${routeHref} should expose a live media-picker trigger`).toBeVisible();
  const panelID = await trigger.getAttribute("aria-controls");
  expect(panelID, `${routeHref} media-picker trigger should identify its panel`).toBeTruthy();
  if (!panelID) throw new Error(`${routeHref} media-picker trigger is missing aria-controls`);
  await trigger.click();
  await expect(trigger).toHaveAttribute("aria-expanded", "true");
  await expect(page.locator(`#${panelID}`)).toBeVisible();
  await trigger.click();
  await expect(trigger).toHaveAttribute("aria-expanded", "false");
}

async function assertMediaRuntimeIdentity(page: Page, initialMediaRuntime: JSHandle<unknown> | undefined, routeHref: string) {
  if (!initialMediaRuntime) return;
  await expect.poll(
    () => page.evaluate((initial) => {
      const current = (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime;
      return current === initial;
    }, initialMediaRuntime),
    { timeout: 30_000, message: `${routeHref} should retain the preloaded media runtime singleton across managed navigation` },
  ).toBe(true);
}

async function assertContentRuntimeIdentity(page: Page, initialContentRuntime: JSHandle<unknown> | undefined, routeHref: string) {
  if (!initialContentRuntime) return;
  await expect.poll(
    () => page.evaluate((initial) => {
      const current = (window as unknown as { GoSXStudioContentEditorRuntime?: unknown }).GoSXStudioContentEditorRuntime;
      return current === initial;
    }, initialContentRuntime),
    { timeout: 30_000, message: `${routeHref} should retain the content editor runtime singleton across managed navigation` },
  ).toBe(true);
}

async function assertManagedRuntimeScript(page: Page, baseURL: string, scriptPath: string, requireExecutedScript: boolean) {
  const scripts = await page.locator("script[data-gosx-script='managed']").evaluateAll((nodes, expectedBaseURL) => {
    const base = new URL(String(expectedBaseURL));
    return nodes.map((node) => {
      const raw = node.getAttribute("src") || "";
      let url: URL | null = null;
      try {
        url = new URL(raw, base);
      } catch {
        // Keep malformed source details in the diagnostic result below.
      }
      return {
        src: raw,
        origin: url?.origin || "",
        pathname: url?.pathname || "",
        loaded: node.getAttribute("data-gosx-script-loaded") || "",
      };
    });
  }, baseURL);
  const origin = new URL(baseURL).origin;
  const matching = scripts.filter((script) => script.origin === origin && script.pathname === scriptPath);
  expect(matching.length, `${scriptPath} should be represented by a managed same-origin script`).toBeGreaterThan(0);
  const executed = matching.filter((script) => script.loaded === "true");
  if (requireExecutedScript) {
    expect(executed, `${scriptPath} should have exactly one executed managed script on first entry; scripts=${JSON.stringify(scripts)}`).toHaveLength(1);
  } else {
    expect(executed.length, `${scriptPath} must not execute more than once on a managed revisit; scripts=${JSON.stringify(scripts)}`).toBeLessThanOrEqual(1);
  }
}

async function sectionOrderKeys(page: Page): Promise<string[]> {
  return page.locator(`${SECTION_ORDER} ${SECTION_ITEM}`).evaluateAll((nodes) =>
    nodes.map((node) => node.getAttribute("data-gosx-studio-section-order-item") || ""),
  );
}

async function enabledSectionKeys(page: Page): Promise<string[]> {
  return page.locator(`${SECTION_ORDER} ${SECTION_ITEM}`).evaluateAll((nodes) =>
    nodes.filter((node) => {
      const handle = node.querySelector("[data-gosx-studio-section-order-handle='true']");
      return handle instanceof HTMLButtonElement && !handle.disabled;
    }).map((node) => node.getAttribute("data-gosx-studio-section-order-item") || "").filter(Boolean),
  );
}

async function previewSectionKeys(page: Page): Promise<string[]> {
  await page.waitForFunction(() => {
    const frame = document.querySelector("iframe[data-studio-preview-frame='true'], iframe[data-gosx-studio-preview-frame]");
    if (!(frame instanceof HTMLIFrameElement)) return false;
    try {
      return !!frame.contentDocument?.querySelector("[data-gosx-studio-section-key]");
    } catch {
      return false;
    }
  }, undefined, { timeout: 30_000 });
  return page.evaluate(() => {
    const frame = document.querySelector("iframe[data-studio-preview-frame='true'], iframe[data-gosx-studio-preview-frame]");
    if (!(frame instanceof HTMLIFrameElement) || !frame.contentDocument) return [];
    return Array.from(frame.contentDocument.querySelectorAll("[data-gosx-studio-section-key]"))
      .map((node) => node.getAttribute("data-gosx-studio-section-key") || "")
      .filter(Boolean);
  });
}

async function publicSectionKeys(request: APIRequestContext, baseURL: string): Promise<string[]> {
  const response = await request.get(`${baseURL}/`);
  expect(response.status(), "public Home should remain reachable during draft/public boundary checks").toBe(200);
  const html = await response.text();
  const keys: string[] = [];
  const sectionKeyPattern = /data-gosx-studio-section-key=(?:"([^"]+)"|'([^']+)')/g;
  for (const match of html.matchAll(sectionKeyPattern)) {
    const key = match[1] || match[2] || "";
    if (key && !keys.includes(key)) keys.push(key);
  }
  return keys;
}

async function dragSectionWithMouse(page: Page, sourceKey: string, targetKey: string, targetHalf: "top" | "bottom") {
  const source = page.locator(`${SECTION_ITEM}[data-gosx-studio-section-order-item='${sourceKey}']`);
  const target = page.locator(`${SECTION_ITEM}[data-gosx-studio-section-order-item='${targetKey}']`);
  const handle = source.locator(SECTION_HANDLE).first();
  await expect(handle).toBeEnabled();
  await handle.scrollIntoViewIfNeeded();
  await target.scrollIntoViewIfNeeded();
  const handleBox = await handle.boundingBox();
  const targetBox = await target.boundingBox();
  if (!handleBox || !targetBox) throw new Error("Home section drag requires visible source and target geometry");
  await page.mouse.move(handleBox.x + handleBox.width / 2, handleBox.y + handleBox.height / 2);
  await page.mouse.down();
  const y = targetHalf === "top" ? targetBox.y + Math.min(8, targetBox.height / 4) : targetBox.y + targetBox.height - Math.min(8, targetBox.height / 4);
  await page.mouse.move(targetBox.x + targetBox.width / 2, y, { steps: 10 });
  await page.mouse.up();
}

async function dragContentBlock(page: Page, sourceHandle: Locator, target: Locator, targetHalf: "top" | "bottom") {
  await expect(sourceHandle).toBeVisible();
  await expect(target).toBeVisible();
  const targetBox = await target.boundingBox();
  if (!targetBox) throw new Error("Page CMS block drag requires visible target geometry");
  const y = targetHalf === "top" ? targetBox.y + Math.min(8, targetBox.height / 4) : targetBox.y + targetBox.height - Math.min(8, targetBox.height / 4);
  const transfer = await page.evaluateHandle(() => new DataTransfer());
  // Chromium's headless drag gesture does not consistently synthesize HTML5
  // drag events for a draggable button. Deliver the same bubbling DragEvent
  // sequence that a user gesture produces, with one real DataTransfer shared
  // by the runtime's dragstart/dragover/drop handlers.
  await sourceHandle.dispatchEvent("dragstart", { dataTransfer: transfer });
  await target.dispatchEvent("dragenter", {
    bubbles: true,
    cancelable: true,
    clientX: targetBox.x + targetBox.width / 2,
    clientY: y,
    dataTransfer: transfer,
  });
  await target.dispatchEvent("dragover", {
    bubbles: true,
    cancelable: true,
    clientX: targetBox.x + targetBox.width / 2,
    clientY: y,
    dataTransfer: transfer,
  });
  await target.dispatchEvent("drop", {
    bubbles: true,
    cancelable: true,
    clientX: targetBox.x + targetBox.width / 2,
    clientY: y,
    dataTransfer: transfer,
  });
  await sourceHandle.dispatchEvent("dragend", { dataTransfer: transfer });
}

async function beginSectionPointerDrag(page: Page, handle: Locator) {
  await expect(handle).toBeEnabled();
  await handle.scrollIntoViewIfNeeded();
  const box = await handle.boundingBox();
  if (!box) throw new Error("section pointer drag requires visible handle geometry");
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
  await page.mouse.down();
  await expect(handle).toHaveAttribute("aria-describedby", "studio-section-order-guidance");
}

async function waitForPreviewRoute(page: Page, expectedPath: string) {
  await page.waitForFunction((path) => {
    const frame = document.querySelector("iframe[data-studio-preview-frame='true'], iframe[data-gosx-studio-preview-frame]");
    if (!(frame instanceof HTMLIFrameElement)) return false;
    try {
      return new URL(frame.contentWindow?.location.href || frame.src, window.location.href).pathname === path;
    } catch {
      return false;
    }
  }, expectedPath, { timeout: 30_000 });
}

async function submitOrdinaryEditorSave(page: Page, _baseURL?: string) {
  // The primary Save button is intentionally hidden while the whole editor
  // form is clean. Make one real, disposable Advanced-mode edit first so the
  // save below exercises the ordinary form POST rather than a hidden command
  // palette option. The server is a fresh fixture for this test, so the probe
  // value cannot touch customer or staging data.
  await revealModeIfPresent(page, "advanced");
  const probeField = page.locator("[data-studio-checkout-panel='true'] input[type='text']").first();
  await expect(probeField, "the default editor should expose a real Advanced text control for the Save probe").toBeVisible();
  await probeField.fill(`Luna ordinary save probe ${Date.now()}`);

  // The command palette also contains a hidden "Save changes" option. Scope
  // this to the actual form submit control so DOM order cannot select that
  // hidden option before the visible toolbar button.
  const button = page.locator("button[data-gosx-studio-save-button='true']:visible, button[data-editor-save-button='true']:visible", { hasText: "Save changes" }).first();
  await expect(button, "the ordinary editor Save changes control should be visible").toBeVisible();
  const responsePromise = page.waitForResponse((response) =>
    response.request().method() === "POST" &&
    response.url().includes("/admin/editor/__actions/save"),
  );
  await button.click();
  const response = await responsePromise;
  expect([200, 303]).toContain(response.status());
  await page.waitForLoadState("domcontentloaded").catch(() => {});
}

async function publishEditorDraft(page: Page) {
  await revealModeIfPresent(page, "publish");
  const button = page.locator("[data-studio-submit-action='publish']").first();
  await expect(button, "Release Center should expose a real publish control").toBeVisible();
  const responsePromise = page.waitForResponse((response) =>
    response.request().method() === "POST" && response.url().includes("/admin/editor/__actions/publish"),
  );
  await button.click();
  const response = await responsePromise;
  expect([200, 303]).toContain(response.status());
  await page.waitForLoadState("domcontentloaded").catch(() => {});
}

async function cancelContentDrag(page: Page, sourceID: string, targetID: string) {
  const sourceHandle = page.locator(`[data-content-block-id='${sourceID}'] [data-content-drag-handle='true']`).first();
  const target = page.locator(`[data-content-block-id='${targetID}']`).first();
  await expect(sourceHandle).toBeVisible();
  await expect(target).toBeVisible();
  const transfer = await page.evaluateHandle(() => new DataTransfer());
  await sourceHandle.dispatchEvent("dragstart", { dataTransfer: transfer });
  const targetBox = await target.boundingBox();
  if (!targetBox) throw new Error("Page CMS cancel drag requires visible target geometry");
  await target.dispatchEvent("dragover", {
    clientX: targetBox.x + targetBox.width / 2,
    clientY: targetBox.y + 4,
    dataTransfer: transfer,
  });
  await sourceHandle.dispatchEvent("dragend");
}

async function screenshot(page: Page, filename: string): Promise<string> {
  const target = path.join(QA_ROOT, filename);
  await page.screenshot({ path: target, fullPage: false });
  return target;
}

async function resetEditorScroll(page: Page) {
  return page.evaluate(() => new Promise<Array<{
    tag: string;
    id: string;
    className: string;
    scrollTop: number;
    scrollLeft: number;
    scrollHeight: number;
    clientHeight: number;
    rect: { left: number; top: number; right: number; bottom: number; width: number; height: number };
    inSiteNavigator: boolean;
  }>>((resolve) => {
    window.scrollTo(0, 0);
    const scrollNodes = new Set<HTMLElement>();
    if (document.scrollingElement instanceof HTMLElement) scrollNodes.add(document.scrollingElement);
    if (document.body instanceof HTMLElement) scrollNodes.add(document.body);
    document.querySelectorAll("*").forEach((candidate) => {
      if (!(candidate instanceof HTMLElement)) return;
      if (candidate.scrollHeight > candidate.clientHeight + 1 || candidate.scrollWidth > candidate.clientWidth + 1) {
        scrollNodes.add(candidate);
      }
    });
    scrollNodes.forEach((node) => {
      node.scrollTop = 0;
      node.scrollLeft = 0;
    });

    // Let nested pane layout/scroll restoration settle before screenshotting;
    // the returned diagnostics distinguish a real CSS clip from stale rail
    // scroll framing when a control remains outside the viewport.
    requestAnimationFrame(() => requestAnimationFrame(() => {
      resolve(Array.from(scrollNodes).map((node) => {
        const rect = node.getBoundingClientRect();
        return {
          tag: node.tagName.toLowerCase(),
          id: node.id,
          className: String(node.className || "").slice(0, 160),
          scrollTop: node.scrollTop,
          scrollLeft: node.scrollLeft,
          scrollHeight: node.scrollHeight,
          clientHeight: node.clientHeight,
          rect: {
            left: Math.round(rect.left),
            top: Math.round(rect.top),
            right: Math.round(rect.right),
            bottom: Math.round(rect.bottom),
            width: Math.round(rect.width),
            height: Math.round(rect.height),
          },
          inSiteNavigator: !!node.closest("[data-studio-site-navigator-panel='true']"),
        };
      }));
    }));
  }));
}

async function ensurePageCMSControlsVisible(page: Page) {
  const editor = page.locator("[data-content-editor]").first();
  await editor.scrollIntoViewIfNeeded();
  await expect(page.locator("button[data-content-add='heading']").first()).toBeVisible({ timeout: 30_000 });
  if (await page.locator("[data-content-block-id]").count() === 0) {
    await page.locator("button[data-content-add='heading']").first().click();
  }
  await expect(page.locator("[data-content-block-id]").last(), "Page CMS should expose at least one visible sortable block row").toBeVisible({ timeout: 30_000 });
  await page.evaluate(() => window.scrollTo(0, 0));
}

async function selectFirstContentBlock(page: Page) {
  const first = page.locator("[data-content-block-id]").first();
  await expect(first, "Page CMS should render a first sortable block").toBeVisible({ timeout: 30_000 });
  await first.click();
  await expect(first).toHaveAttribute("data-content-editor-selected", "true");
}

async function captureMediaAndDragStates(page: Page, screenshots: Record<string, string>, viewportLabel: string) {
  const blocks = page.locator("[data-content-block-id]");
  await expect(blocks).toHaveCount(4);
  const source = blocks.nth(0);
  const target = blocks.nth(1);
  await source.scrollIntoViewIfNeeded();
  const sourceHandle = source.locator("[data-content-drag-handle='true']").first();
  const sourceBox = await sourceHandle.boundingBox();
  const targetBox = await target.boundingBox();
  if (!sourceBox || !targetBox) throw new Error("Page CMS drag screenshot requires visible source and target geometry");
  // Keep the drag in its active, inspectable state long enough to capture the
  // before-drop affordance. The functional suite below also performs a full
  // Playwright mouse drag; this frozen event sequence is only for deterministic
  // visual evidence while preserving the real runtime's drag handlers.
  const transfer = await page.evaluateHandle(() => new DataTransfer());
  await sourceHandle.dispatchEvent("dragstart", { dataTransfer: transfer });
  await target.dispatchEvent("dragover", { clientY: targetBox.y + 4, dataTransfer: transfer });
  await expect(target, "a valid drag should expose a before-drop indicator").toHaveAttribute("data-content-editor-drop-before", "true");
  screenshots[`page-cms-drag-active-${viewportLabel}`] = await screenshot(page, `page-cms-drag-active-${viewportLabel}.png`);
  await sourceHandle.dispatchEvent("dragend");
  await expect(target).not.toHaveAttribute("data-content-editor-drop-before", "true");
  expect(await contentBlockIDs(page)).toEqual(["heading-1", "paragraph-2", "paragraph-3", "button-4"]);

  // Add a real image block so the screenshot proves the media editing
  // affordance. It remains local to this isolated browser page; the next
  // viewport navigates back to the server's unchanged draft fixture.
  const addImage = page.locator("button[data-content-add='image']").first();
  await expect(addImage, "Page CMS should expose the Image authoring control").toBeVisible();
  await addImage.click();
  const imageBlock = page.locator("[data-content-block-type='image']").last();
  await expect(imageBlock, "Image insertion should render a media block").toBeVisible();
  await imageBlock.scrollIntoViewIfNeeded();
  screenshots[`page-cms-media-${viewportLabel}`] = await screenshot(page, `page-cms-media-${viewportLabel}.png`);
}

async function probeFirstEditableHeading(page: Page) {
  const headingInput = page.locator("[data-content-block-type='heading'] input[placeholder='Section heading']").first();
  await expect(headingInput, "the selected first content block should expose an editable Heading input").toBeVisible({ timeout: 30_000 });
  return headingInput.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    const primary = Array.from(document.querySelectorAll("form.admin-form button.button--primary"))
      .find((candidate) => /save draft/i.test(candidate.textContent || "")) as HTMLButtonElement | undefined;
    const style = primary ? window.getComputedStyle(primary) : null;
    const parseRGB = (value: string) => {
      const match = value.match(/^rgba?\(([^)]+)\)$/i);
      if (!match) return null;
      const channels = match[1].split(",").slice(0, 3).map((channel) => Number.parseFloat(channel.trim()));
      return channels.length === 3 && channels.every((channel) => Number.isFinite(channel)) ? channels : null;
    };
    const luminance = (value: number[]) => value.map((channel) => channel / 255).map((channel) => channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4)
      .reduce((sum, channel, index) => sum + channel * [0.2126, 0.7152, 0.0722][index], 0);
    const foreground = style ? parseRGB(style.color) : null;
    const background = style ? parseRGB(style.backgroundColor) : null;
    const foregroundLuminance = foreground ? luminance(foreground) : null;
    const backgroundLuminance = background ? luminance(background) : null;
    const contrastRatio = foregroundLuminance !== null && backgroundLuminance !== null
      ? (Math.max(foregroundLuminance, backgroundLuminance) + 0.05) / (Math.min(foregroundLuminance, backgroundLuminance) + 0.05)
      : null;
    return {
      inputRect: {
        left: rect.left,
        top: rect.top,
        right: rect.right,
        bottom: rect.bottom,
        width: rect.width,
        height: rect.height,
      },
      inputInsideViewport: rect.left >= 0 && rect.top >= 0 && rect.right <= window.innerWidth && rect.bottom <= window.innerHeight,
      primarySave: primary && style ? {
        text: (primary.textContent || "").replace(/\s+/g, " ").trim(),
        color: style.color,
        backgroundColor: style.backgroundColor,
        borderColor: style.borderColor,
        boxShadow: style.boxShadow,
        fontWeight: style.fontWeight,
        minHeight: style.minHeight,
        padding: style.padding,
        contrastRatio,
      } : null,
    };
  });
}

async function contentBlockIDs(page: Page): Promise<string[]> {
  return page.locator("[data-content-block-id]").evaluateAll((nodes) =>
    nodes.map((node) => node.getAttribute("data-content-block-id") || ""));
}

async function layoutProbe(page: Page) {
  return page.evaluate(() => ({
    viewport: { width: window.innerWidth, height: window.innerHeight },
    scroll: {
      width: document.documentElement.scrollWidth,
      height: document.documentElement.scrollHeight,
    },
    hasHorizontalOverflow: document.documentElement.scrollWidth > window.innerWidth + 1,
    regions: [
      "[data-studio-workbench='true']",
      "[data-gosx-studio-workbench='true']",
      "[data-studio-site-navigator-panel='true']",
      "[data-content-editor='true']",
      "[data-content-editor-list='true']",
      "[data-studio-publish-panel='true']",
    ].map((selector) => {
      const el = document.querySelector(selector);
      if (!el) return { selector, present: false };
      const rect = el.getBoundingClientRect();
      return {
        selector,
        present: true,
        visible: rect.width > 0 && rect.height > 0,
        rect: {
          left: Math.round(rect.left),
          top: Math.round(rect.top),
          right: Math.round(rect.right),
          bottom: Math.round(rect.bottom),
          width: Math.round(rect.width),
          height: Math.round(rect.height),
        },
      };
    }),
    visibleControls: Array.from(document.querySelectorAll("button, a, input, textarea, select")).filter((el) => {
      const rect = el.getBoundingClientRect();
      const style = window.getComputedStyle(el);
      return rect.width > 0 && rect.height > 0 && style.visibility !== "hidden" && style.display !== "none";
    }).length,
    clippedControls: Array.from(document.querySelectorAll("button, a, input, textarea, select")).filter((el) => {
      const rect = el.getBoundingClientRect();
      const style = window.getComputedStyle(el);
      if (!(rect.width > 0 && rect.height > 0) || style.visibility === "hidden" || style.display === "none") return false;
      return rect.left < -1 || rect.top < -1 || rect.right > window.innerWidth + 1 || rect.bottom > window.innerHeight + 1;
    }).slice(0, 20).map((el) => {
      const rect = el.getBoundingClientRect();
      return {
        tag: el.tagName.toLowerCase(),
        text: (el.textContent || (el as HTMLInputElement).value || "").replace(/\s+/g, " ").trim().slice(0, 80),
        name: (el as HTMLInputElement).name || "",
        selectorHint: el.getAttribute("data-content-add") || el.getAttribute("data-studio-submit-action") || el.getAttribute("aria-label") || "",
        rect: {
          left: Math.round(rect.left),
          top: Math.round(rect.top),
          right: Math.round(rect.right),
          bottom: Math.round(rect.bottom),
          width: Math.round(rect.width),
          height: Math.round(rect.height),
        },
      };
    }),
  }));
}

async function collectInventory(page: Page, baseURL: string) {
  const pageCMS = await page.evaluate(() => {
    const texts = (selector: string) => Array.from(document.querySelectorAll(selector))
      .map((el) => (el.textContent || "").replace(/\s+/g, " ").trim())
      .filter(Boolean);
    const scripts = Array.from(document.querySelectorAll("script[src]")).map((script) => script.getAttribute("src") || "");
    const blocks = Array.from(document.querySelectorAll("[data-content-block-id]"));
    return {
      scripts,
      usesSharedRuntimeScript: scripts.some((src) => src.includes("/_gosx/studio/content-editor.js")),
      // Only the historical root alias counts as host legacy. Do not use
      // endsWith: /_gosx/studio/content-editor.js is the canonical shared
      // endpoint and also ends with the same filename.
      usesHostLegacyScript: scripts.some((src) => src === "/content-editor.js"),
      contentBlocks: blocks.length,
      blockIDs: blocks.map((el) => el.getAttribute("data-content-block-id") || ""),
      draggableBlocks: blocks.filter((el) => el.getAttribute("draggable") === "true").length,
      dragHandles: document.querySelectorAll("[data-content-drag-handle='true']").length,
      addButtons: texts("[data-content-add]"),
      moveButtons: texts("[aria-label='Move block up'], [aria-label='Move block down']"),
      duplicateButtons: texts("[data-content-block-id] button").filter((text) => text === "Duplicate").length,
      removeButtons: texts("[data-content-block-id] button").filter((text) => text === "Remove").length,
      // Search is intentionally scoped to the Page CMS block editor. The
      // global editor command launcher also says "Find", but that is not a
      // block search control and must not make this inventory pass by accident.
      hasSearchControl: !!document.querySelector("[data-content-editor-search='true']"),
      hasCommandSearchControl: !!document.querySelector("[data-studio-command-search]"),
      hasUndoControl: !!document.querySelector("[data-content-editor-history] [data-content-editor-action='undo']"),
      hasRedoControl: !!document.querySelector("[data-content-editor-history] [data-content-editor-action='redo']"),
      focusedControls: Array.from(document.querySelectorAll("button, a, input, textarea, select")).length,
    };
  });

  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await revealModeIfPresent(page, "advanced");
  const homeEditor = await page.evaluate(() => ({
    hasWorkbench: !!document.querySelector("[data-studio-workbench='true'], [data-gosx-studio-workbench='true']"),
    hasCanvas: !!document.querySelector("canvas[data-gosx-surface-kind='canvas2d'], canvas[data-gosx-canvas-wasm-free='true']"),
    siteNavigatorLinks: document.querySelectorAll("[data-studio-site-navigator-panel='true'] a").length,
    compositionIntents: document.querySelectorAll("[data-studio-composition-intent]").length,
    siteMapNodes: document.querySelectorAll("[data-studio-site-map-workspace-node]").length,
    publishButtons: document.querySelectorAll("[data-studio-submit-action='publish']").length,
  }));

  return {
    pageCMS,
    homeEditor,
  };
}

function baselineIssues(inventory: Awaited<ReturnType<typeof collectInventory>>, viewportReports: Array<Record<string, unknown>>) {
  const issues: string[] = [];
  if (!inventory.pageCMS.usesSharedRuntimeScript) {
    issues.push("Page CMS still uses the host legacy /content-editor.js script; shared /_gosx/studio/content-editor.js is not integrated yet.");
  }
  if (!inventory.pageCMS.hasSearchControl) {
    issues.push("No visible Page CMS block search command was detected.");
  }
  if (!inventory.pageCMS.hasUndoControl || !inventory.pageCMS.hasRedoControl) {
    issues.push("No visible Page CMS undo/redo controls were detected.");
  }
  if (viewportReports.some((report) => Boolean((report.pageCMSFit as { hasHorizontalOverflow?: boolean }).hasHorizontalOverflow))) {
    issues.push("At least one Page CMS viewport reported page-level horizontal overflow.");
  }
  return issues;
}

function renderBaselineMarkdown(report: {
  generatedAt: string;
  baseURL: string;
  pageID: string;
  screenshots: Record<string, string>;
  inventory: Awaited<ReturnType<typeof collectInventory>>;
  viewportReports: Array<Record<string, unknown>>;
  issues: string[];
}) {
  return [
    "# Modern Editor Baseline",
    "",
    `Generated: ${report.generatedAt}`,
    `Isolated server: ${report.baseURL}`,
    `Baseline Page CMS id: ${report.pageID}`,
    "",
    "## Screenshots",
    "",
    ...Object.entries(report.screenshots).map(([key, value]) => `- ${key}: ${value}`),
    "",
    "## Inventory",
    "",
    "```json",
    JSON.stringify(report.inventory, null, 2),
    "```",
    "",
    "## Viewport Fit",
    "",
    "```json",
    JSON.stringify(report.viewportReports, null, 2),
    "```",
    "",
    "## Biggest Usability Issues",
    "",
    ...(report.issues.length ? report.issues.map((issue) => `- ${issue}`) : ["- No baseline issues detected by this automated inventory."]),
    "",
  ].join("\n");
}
