import { expect, test, type APIRequestContext, type Frame, type JSHandle, type Locator, type Page } from "@playwright/test";
import { mkdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import {
  applyCompositionIntentInPlace,
  gotoEditor,
  revealModeIfPresent,
  startMuddyCollaboration,
} from "./reference_apps_harness";

const CREATE_MESSAGE = "Landing page created with 3 starter sections. Edit its sections in Content › Pages.";
const PAGE_INDEX_RENDERER = "[data-gosx-studio-backend-page-index-renderer='gosx-studio']";
const PAGE_DETAIL_RENDERER = "[data-gosx-studio-backend-page-detail-renderer='gosx-studio']";
const CONTENT_RUNTIME_PATH = "/_gosx/studio/content-editor.js";
const MEDIA_RUNTIME_PATH = "/_gosx/studio/media-runtime.js";
const DEFAULT_INTEGRATION_ROOT = path.resolve(__dirname, "../.tiller/scratch/codex/enterprise-polish-20260827/integration08");
const INTEGRATION_ROOT = path.resolve(
  process.env.GOSX_STUDIO_ENTERPRISE_ARTIFACT_ROOT?.trim() || DEFAULT_INTEGRATION_ROOT,
);
const EXPECTED_MODULE_VERSION = process.env.GOSX_STUDIO_EXPECTED_MODULE_VERSION?.trim() || "";
const FIXTURE_DEPLOYMENT_VERSION = "test-deployment-20260827";
const CONFLICT_LAYOUT_LIMITS = {
  desktop: { detailMinWidth: 240, maxHeight: 240 },
  mobile: { detailMinWidth: 180, maxHeight: 360 },
} as const;

type PageFormSnapshot = {
  title: string;
  slug: string;
  body: string;
  expectedRevision: string;
  published: boolean;
};

type ActionResponsePayload = {
  ok?: boolean;
  message?: string;
  redirect?: string;
  values?: Record<string, string>;
};

type SaveResult = {
  status: number;
  payload: ActionResponsePayload | null;
  revisionReadback: string | null;
  source: "structured-response" | "remounted-form" | "none";
};

type SaveTrigger = "button" | "keyboard";

type ManagedScriptSnapshot = {
  src: string;
  href: string;
  origin: string;
  pathname: string;
  releaseKey: string;
  loadMode: string;
  loaded: string;
};

type ManagedAssetEvidence = {
  rendered: ManagedScriptSnapshot[];
  fetched: Array<{ path: string; url: string; status: number; releaseKey: string }>;
  capturedRequests: Array<{ path: string; url: string; releaseKey: string }>;
  executed: Record<string, number>;
  pending: Record<string, number>;
};

test.describe("@reference-apps enterprise polish acceptance", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Page CMS rejects a stale two-tab save without destroying the local draft", async ({ browser, page, request }) => {
    const server = await startMuddyCollaboration(request);
    const contextA = await browser.newContext();
    const contextB = await browser.newContext();
    const pageA = await contextA.newPage();
    const pageB = await contextB.newPage();

    try {
      await signIn(pageA, server.baseURL, "admin");
      await signIn(pageB, server.baseURL, "admin");
      const pageID = await createLandingPage(pageA, server.baseURL);
      const detailURL = `${server.baseURL}/admin/pages/${pageID}`;

      // Both browser contexts load the same draft before either one types.
      // Keeping the original token in hand makes the CAS precondition
      // explicit instead of relying on a hidden implementation detail.
      await openPageDetail(pageA, detailURL);
      await openPageDetail(pageB, detailURL);
      const initialA = await readPageForm(pageA);
      const initialB = await readPageForm(pageB);
      expect(initialA.expectedRevision, "Page CMS must render a non-empty revision precondition").toBeTruthy();
      expect(initialB).toEqual(initialA);
      const oldRevision = initialA.expectedRevision;

      const titleA = `Enterprise A ${Date.now()}`;
      const bodyMarkerA = `enterprise-a-${Date.now()}`;
      await replaceWithKeyboard(pageA, pageA.locator("input[name='title']").first(), titleA);
      await replaceWithKeyboard(pageA, await firstContentField(pageA), bodyMarkerA);
      const draftA = await readPageForm(pageA);
      expect(draftA.title).toBe(titleA);
      expect(draftA.body).toContain(bodyMarkerA);

      const acceptedA = await saveDraft(pageA, pageID);
      expect([200, 303], "the first tab's draft must be accepted").toContain(acceptedA.status);
      const acceptedRevision = expectObservedSuccess(acceptedA, "first tab", oldRevision);
      await expect(pageA.locator("[data-content-editor='true']")).toHaveAttribute("data-content-editor-dirty", "false");
      const savedA = await readPageForm(pageA);
      expect(savedA.expectedRevision, "the remounted first tab must keep the structured revision token").toBe(acceptedRevision);
      expect(savedA.title).toBe(titleA);
      expect(savedA.body).toContain(bodyMarkerA);

      const titleB = `Enterprise B ${Date.now()}`;
      const bodyMarkerB = `enterprise-b-${Date.now()}`;
      await replaceWithKeyboard(pageB, pageB.locator("input[name='title']").first(), titleB);
      await replaceWithKeyboard(pageB, await firstContentField(pageB), bodyMarkerB);
      const draftB = await readPageForm(pageB);
      expect(draftB.title).toBe(titleB);
      expect(draftB.body).toContain(bodyMarkerB);
      expect(draftB.expectedRevision, "the second tab must still submit its original precondition").toBe(oldRevision);

      const staleB = await saveDraft(pageB, pageID);
      expect(staleB.status, "a second save with the old revision must return HTTP 409").toBe(409);
      const conflictEditor = pageB.locator("[data-content-editor='true']").first();
      await expect(conflictEditor).toHaveAttribute("data-content-editor-save-state", "conflict");
      const conflictStatus = conflictEditor.locator("[data-content-editor-save-status]");
      await expect(conflictStatus, "the guarded conflict must be announced in the live editor status").toHaveAttribute("role", "alert");
      await expect(conflictStatus).toContainText(/This page changed elsewhere/i);
      const currentVersionLink = conflictEditor.locator("[data-content-editor-save-conflict-link]");
      await expect(currentVersionLink, "guarded conflicts must expose a current-version comparison link").toBeVisible();

      // Treat the conflict surface as a real-host layout gate at both target
      // acceptance widths. The stale tab stays live throughout; resizing is
      // the only viewport change and never reloads or discards its draft.
      const desktopConflictLayout = await assertConflictLayout(pageB, 1280, 800);

      // The failed action must not replace B's title, body, or token. Do not
      // reload this stale tab: preserving its live controls is the behavior
      // under test and avoids silently discarding the operator's draft.
      const retainedB = await readPageForm(pageB);
      expect(retainedB.title).toBe(titleB);
      expect(retainedB.body).toContain(bodyMarkerB);
      expect(retainedB.body).not.toContain(bodyMarkerA);
      expect(retainedB.expectedRevision).toBe(oldRevision);

      const publicURL = `${server.baseURL}/pages/${initialA.slug}`;
      const publicAfterConflict = await request.get(publicURL);
      expect(publicAfterConflict.status(), "a draft-only conflict must not mutate a public route").toBe(404);
      const conflictScreenshot = await captureScreenshot(pageB, "page-cms-conflict-1280x800", 1280, 800, false);

      const mobileConflictLayout = await assertConflictLayout(pageB, 390, 844);
      const mobileConflictScreenshot = await captureScreenshot(pageB, "page-cms-conflict-390x844", 390, 844, false);
      await pageB.setViewportSize({ width: 1280, height: 800 });

      await expect(currentVersionLink).toHaveAttribute("target", "_blank");
      await expect(currentVersionLink).toHaveText("Open current version in new tab");
      const currentHref = await currentVersionLink.getAttribute("href");
      expect(new URL(currentHref ?? "", server.baseURL).pathname).toBe(`/admin/pages/${pageID}`);

      const currentVersionPromise = pageB.waitForEvent("popup");
      await currentVersionLink.click();
      const currentPage = await currentVersionPromise;
      try {
        await currentPage.waitForLoadState("domcontentloaded");
        await expectPageDetailReady(currentPage);
        const currentBeforeReconcile = await readPageForm(currentPage);
        expect(currentBeforeReconcile.title, "the comparison tab must show A's accepted title").toBe(titleA);
        expect(currentBeforeReconcile.body, "the comparison tab must show A's accepted body").toContain(bodyMarkerA);
        expect(currentBeforeReconcile.body).not.toContain(bodyMarkerB);
        expect(currentBeforeReconcile.expectedRevision).toBe(savedA.expectedRevision);

        // Reconcile in the new tab. B remains open and untouched throughout,
        // so this save proves the operator can merge without refreshing away
        // the stale tab's local draft.
        const mergedTitle = `${titleA} + ${titleB}`;
        const mergedBodyMarker = `${bodyMarkerA}+${bodyMarkerB}`;
        await replaceWithKeyboard(currentPage, currentPage.locator("input[name='title']").first(), mergedTitle);
        await replaceWithKeyboard(currentPage, await firstContentField(currentPage), mergedBodyMarker);
        const mergedSave = await saveDraft(currentPage, pageID);
        expect([200, 303], "the reconciled current version must save").toContain(mergedSave.status);
        const mergedRevision = expectObservedSuccess(mergedSave, "reconciled current version", currentBeforeReconcile.expectedRevision);
        await expect(currentPage.locator("[data-content-editor='true']")).toHaveAttribute("data-content-editor-dirty", "false");
        const currentAfterReconcile = await readPageForm(currentPage);
        expect(currentAfterReconcile.title).toBe(mergedTitle);
        expect(currentAfterReconcile.body).toContain(mergedBodyMarker);
        expect(currentAfterReconcile.expectedRevision).toBe(mergedRevision);

        const stillRetainedB = await readPageForm(pageB);
        expect(stillRetainedB.title).toBe(titleB);
        expect(stillRetainedB.body).toContain(bodyMarkerB);
        expect(stillRetainedB.expectedRevision).toBe(oldRevision);

        const publicAfterReconcile = await request.get(publicURL);
        expect(publicAfterReconcile.status(), "reconciling a draft must not publish the page").toBe(404);
        writeEvidence("page-cms-stale-save", {
          pageID,
          oldRevision,
          acceptedRevision,
          acceptedRevisionReadback: acceptedA.revisionReadback,
          acceptedSource: acceptedA.source,
          acceptedStatus: acceptedA.status,
          reconciledRevision: mergedRevision,
          reconciledRevisionReadback: mergedSave.revisionReadback,
          reconciledSource: mergedSave.source,
          reconciledStatus: mergedSave.status,
          staleStatus: staleB.status,
          publicStatuses: [publicAfterConflict.status(), publicAfterReconcile.status()],
          screenshots: [conflictScreenshot, mobileConflictScreenshot],
          conflictLayout: {
            desktop: desktopConflictLayout,
            mobile: mobileConflictLayout,
          },
        });
      } finally {
        await currentPage.close();
      }
    } finally {
      await Promise.all([contextA.close(), contextB.close()]);
      await server.stop();
    }
  });

  test("managed Content navigation keeps Page CMS keyboard editing and media changes draft-only", async ({ page, request }) => {
    const server = await startMuddyCollaboration(request);
    try {
      await signIn(page, server.baseURL, "admin");
      const pageID = await createLandingPage(page, server.baseURL);

      // Start from the dashboard and use the managed Content › Pages link,
      // then the managed Edit link. This exercises the real route transition
      // before touching the Page CMS editor controls.
      await page.goto(`${server.baseURL}/admin`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await clickManagedPages(page, server.baseURL);
      const indexRenderer = page.locator(PAGE_INDEX_RENDERER).first();
      await expect(indexRenderer).toBeVisible();
      const detailLink = indexRenderer.locator(`a[data-gosx-link='true'][href*='/admin/pages/${pageID}']`).first();
      await expect(detailLink, "managed Pages index must expose the created page's Edit link").toBeVisible();
      const detailHref = await detailLink.getAttribute("href");
      const detailURL = new URL(detailHref ?? "", server.baseURL);
      expect(detailURL.origin).toBe(new URL(server.baseURL).origin);
      expect(detailURL.pathname).toBe(`/admin/pages/${pageID}`);
      await Promise.all([
        page.waitForURL((url) => url.origin === new URL(server.baseURL).origin && url.pathname === detailURL.pathname),
        detailLink.click(),
      ]);

      const detail = page.locator(PAGE_DETAIL_RENDERER).first();
      await expect(detail).toBeVisible();
      const editor = detail.locator("[data-content-editor='true']").first();
      await expect(editor).toHaveAttribute("data-content-editor-bound", "true", { timeout: 30_000 });
      const revision = detail.locator("input[name='expectedRevision']").first();
      await expect(revision).toHaveAttribute("data-content-editor-revision", "true");
      const initialRevision = await revision.inputValue();
      expect(initialRevision).toBeTruthy();

      const initialBlockCount = await editor.locator("[data-content-block-id]").count();
      const addHeading = editor.locator("button[data-content-add='heading']").first();
      await expect(addHeading).toBeVisible();
      await addHeading.click();
      const addedBlock = editor.locator("[data-content-block-id]").last();
      await expect(addedBlock, "Page CMS must add a real block through the visible toolbar").toBeVisible();
      await expect(editor.locator("[data-content-block-id]")).toHaveCount(initialBlockCount + 1);
      await expect(addedBlock).toHaveAttribute("data-content-block-type", "heading");
      await expect(addedBlock).toHaveAttribute("data-content-editor-selected", "true");

      // Click a real editable control, type through the browser keyboard,
      // undo natively, then Tab to the next control. No synthetic pointer or
      // DOM event dispatch is part of this acceptance path.
      const addedHeading = addedBlock.locator(".content-block__fields label", { hasText: "Heading" }).locator("input").first();
      const headingBefore = await addedHeading.inputValue();
      await addedHeading.click();
      await page.keyboard.press("ControlOrMeta+A");
      // Keep this as one native edit transaction. Chromium may coalesce a
      // multi-character `keyboard.type` sequence into several undo units;
      // one character proves the browser's real select/type/undo path without
      // making the assertion depend on that implementation detail.
      await page.keyboard.type("x");
      await expect(addedHeading).toHaveValue("x");
      await page.keyboard.press("ControlOrMeta+Z");
      await expect(addedHeading).toHaveValue(headingBefore);
      await page.keyboard.press("Tab");
      await expect(addedBlock.locator("select[aria-label='Heading level']")).toBeFocused();

      const sourceAfterUndo = JSON.parse(await detail.locator("textarea[name='body']").inputValue()) as { blocks?: Array<{ type?: string; text?: string }> };
      expect(sourceAfterUndo.blocks?.some((block) => block.type === "heading")).toBe(true);

      // Exercise the media picker with keyboard interaction: open the real
      // picker, type a query into its search field, focus the result, and
      // activate it with Enter before editing the alt text by keyboard.
      const mediaURL = detail.locator("input[name='metaImageUrl']").first();
      const mediaPicker = detail.locator("input[name='metaImageUrl'] + .media-picker").first();
      await expect(mediaURL).toHaveAttribute("data-media-picker-bound", "true", { timeout: 30_000 });
      const mediaTrigger = mediaPicker.locator(".media-picker__trigger").first();
      await mediaTrigger.click();
      await expect(mediaTrigger).toHaveAttribute("aria-expanded", "true");
      const mediaSearch = mediaPicker.locator(".media-picker__search").first();
      await expect(mediaSearch).toBeFocused();
      await page.keyboard.type("moss");
      const mediaAsset = mediaPicker.locator(".media-picker__asset").first();
      await expect(mediaAsset).toHaveCount(1);
      await mediaAsset.focus();
      await page.keyboard.press("Enter");
      await expect(mediaURL).toHaveValue(/photo-1565193566173-7a0ee3dbe261/);
      const expectedMediaURL = await mediaURL.inputValue();

      const mediaAlt = detail.locator("input[name='metaImageAlt']").first();
      const mediaAltMarker = `keyboard-alt-${Date.now()}`;
      await replaceWithKeyboard(page, mediaAlt, mediaAltMarker);
      await expect(mediaAlt).toHaveValue(mediaAltMarker);

      const published = detail.locator("input[name='published']").first();
      await expect(published, "the new Page CMS page must begin as a draft").not.toBeChecked();
      const draftBeforeSave = await readPageForm(page);
      const save = await saveDraft(page, pageID);
      expect([200, 303], "content and media edits must save through the draft action").toContain(save.status);
      const savedRevision = expectObservedSuccess(save, "keyboard/media draft", initialRevision);
      await expect(editor).toHaveAttribute("data-content-editor-dirty", "false");
      await expect(published).not.toBeChecked();
      const persistedDraft = await readPageForm(page);
      expect(persistedDraft.title, "the remounted Page CMS draft must retain its title").toBe(draftBeforeSave.title);
      expect(persistedDraft.body, "the remounted Page CMS draft must retain its body").toBe(draftBeforeSave.body);
      const persistedRevision = await revision.inputValue();
      expect(persistedRevision, "the remounted draft must keep the observed revision token").toBe(savedRevision);
      const persistedMediaURL = await mediaURL.inputValue();
      const persistedMediaAlt = await mediaAlt.inputValue();
      expect(persistedMediaURL, "the remounted draft must retain the selected media URL").toBe(expectedMediaURL);
      expect(persistedMediaAlt, "the remounted draft must retain the edited media alt text").toBe(mediaAltMarker);

      const slug = await detail.locator("input[name='slug']").inputValue();
      const publicResponse = await request.get(`${server.baseURL}/pages/${slug}`);
      expect(publicResponse.status(), "Save draft must not publish the new page").toBe(404);
      const screenshots = [];
      for (const viewport of [
        { label: "1600x900", width: 1600, height: 900 },
        { label: "1280x800", width: 1280, height: 800 },
        { label: "390x844", width: 390, height: 844 },
      ]) {
        screenshots.push(await captureScreenshot(page, `page-cms-${viewport.label}`, viewport.width, viewport.height));
      }
      writeEvidence("page-cms-keyboard-media", {
        pageID,
        initialRevision,
        saveStatus: save.status,
        saveSource: save.source,
        revisionReadback: save.revisionReadback,
        savedRevision,
        publicStatus: publicResponse.status(),
        blockCount: await editor.locator("[data-content-block-id]").count(),
        expectedMediaURL,
        mediaURL: persistedMediaURL,
        mediaAlt: persistedMediaAlt,
        expectedMediaAlt: mediaAltMarker,
        screenshots,
      });
    } finally {
      await server.stop();
    }
  });

  test("real Page CMS Search Enter stays view-only across clean and dirty drafts", async ({ page, request }) => {
    const server = await startMuddyCollaboration(request);
    const saveRequests: string[] = [];
    const onRequest = (browserRequest: { method: () => string; url: () => string }) => {
      if (browserRequest.method() === "POST" && browserRequest.url().includes("/__actions/save")) {
        saveRequests.push(browserRequest.url());
      }
    };
    page.on("request", onRequest);

    try {
      await signIn(page, server.baseURL, "admin");
      const pageID = await createLandingPage(page, server.baseURL);
      await openPageDetail(page, `${server.baseURL}/admin/pages/${pageID}`);

      const detail = page.locator(PAGE_DETAIL_RENDERER).first();
      const editor = detail.locator("[data-content-editor='true']").first();
      const search = editor.locator("[data-content-editor-search='true']").first();
      const title = detail.locator("input[name='title']").first();
      await expect(search, "Page CMS must expose a real block Search field").toBeVisible();
      await expect(editor).toHaveAttribute("data-content-editor-save-state", "idle");
      await expect(editor).toHaveAttribute("data-content-editor-dirty", "false");

      const cleanBefore = await readPageForm(page);
      const cleanBody = cleanBefore.body;
      await search.fill("paragraph");
      await expect(search).toBeFocused();
      await expect(search).toHaveValue("paragraph");
      await expect(editor).toHaveAttribute("data-content-editor-search-query", "paragraph");
      const cleanVisibleCount = await editor.getAttribute("data-content-editor-search-visible-count");
      expect(Number(cleanVisibleCount), "Search should retain a real filtered result in the clean draft").toBeGreaterThan(0);

      // This is a browser keyboard action on the real host control. The
      // search field is view-only, so Enter must not invoke the form's Save
      // submitter or mutate the clean draft.
      await search.press("Enter");
      await page.waitForTimeout(250);
      await expect(search).toBeFocused();
      await expect(search).toHaveValue("paragraph");
      await expect(editor).toHaveAttribute("data-content-editor-search-query", "paragraph");
      await expect(editor).toHaveAttribute("data-content-editor-search-visible-count", cleanVisibleCount ?? "0");
      await expect(editor).toHaveAttribute("data-content-editor-dirty", "false");
      await expect(editor).toHaveAttribute("data-content-editor-save-state", "idle");
      expect(saveRequests, "clean Search Enter must not issue a save POST").toHaveLength(0);
      expect((await readPageForm(page)).body, "clean Search Enter must not rewrite the draft body").toBe(cleanBody);

      const dirtyTitle = `Search Enter draft ${Date.now()}`;
      await replaceWithKeyboard(page, title, dirtyTitle);
      await expect(editor).toHaveAttribute("data-content-editor-dirty", "true");
      await expect(editor).toHaveAttribute("data-content-editor-save-state", "dirty");
      const dirtyBefore = await readPageForm(page);
      await search.focus();
      await expect(search).toBeFocused();
      await search.press("Enter");
      await page.waitForTimeout(250);
      await expect(search).toBeFocused();
      await expect(search).toHaveValue("paragraph");
      await expect(editor).toHaveAttribute("data-content-editor-search-query", "paragraph");
      await expect(editor).toHaveAttribute("data-content-editor-search-visible-count", cleanVisibleCount ?? "0");
      await expect(editor).toHaveAttribute("data-content-editor-dirty", "true");
      await expect(editor).toHaveAttribute("data-content-editor-save-state", "dirty");
      expect(saveRequests, "dirty Search Enter must not issue a save POST").toHaveLength(0);
      expect(await readPageForm(page), "dirty Search Enter must retain all draft values").toEqual(dirtyBefore);

      // Keep the explicit button path as a positive control: the Search
      // guard must not block the actual Save draft submitter.
      const explicitSave = await saveDraft(page, pageID, "button");
      expect([200, 303], "the explicit Save draft control must still submit").toContain(explicitSave.status);
      const explicitRevision = expectObservedSuccess(explicitSave, "Search Enter explicit save", dirtyBefore.expectedRevision);
      await expect(page.locator(`${PAGE_DETAIL_RENDERER} [data-content-editor='true']`).first())
        .toHaveAttribute("data-content-editor-dirty", "false");
      const afterExplicitSave = await readPageForm(page);
      expect(afterExplicitSave.title).toBe(dirtyTitle);
      expect(afterExplicitSave.expectedRevision).toBe(explicitRevision);
      const publicURL = `${server.baseURL}/pages/${afterExplicitSave.slug}`;
      expect((await request.get(publicURL)).status(), "a draft Save must keep the public route unpublished").toBe(404);

      // Make another real edit and invoke the enhanced save from a focused
      // Search control. ControlOrMeta is Playwright's cross-platform native
      // modifier alias; this run observes the Ctrl+S path on Linux while
      // keeping the test portable to a macOS project.
      const keyboardTitle = `${dirtyTitle} keyboard`;
      await replaceWithKeyboard(page, title, keyboardTitle);
      const keyboardEditor = page.locator(`${PAGE_DETAIL_RENDERER} [data-content-editor='true']`).first();
      await expect(keyboardEditor).toHaveAttribute("data-content-editor-dirty", "true");
      const keyboardBefore = await readPageForm(page);
      const keyboardSearch = keyboardEditor.locator("[data-content-editor-search='true']").first();
      await keyboardSearch.fill("heading");
      await keyboardSearch.focus();
      await expect(keyboardSearch).toBeFocused();
      const keyboardSave = await saveDraft(page, pageID, "keyboard", keyboardSearch);
      expect([200, 303], "ControlOrMeta+S from Search must still submit").toContain(keyboardSave.status);
      const keyboardRevision = expectObservedSuccess(keyboardSave, "Search Enter keyboard save", keyboardBefore.expectedRevision);
      await expect(page.locator(`${PAGE_DETAIL_RENDERER} [data-content-editor='true']`).first())
        .toHaveAttribute("data-content-editor-dirty", "false");
      const persisted = await readPageForm(page);
      expect(persisted.title, "ControlOrMeta+S must persist the edited draft title after remount").toBe(keyboardTitle);
      expect(persisted.expectedRevision).toBe(keyboardRevision);
      expect((await request.get(`${server.baseURL}/pages/${persisted.slug}`)).status(), "keyboard Save must keep the draft route unpublished").toBe(404);

      writeEvidence("page-cms-search-enter", {
        pageID,
        cleanSearch: "paragraph",
        cleanVisibleCount,
        dirtyTitle,
        explicitSaveStatus: explicitSave.status,
        explicitSaveSource: explicitSave.source,
        explicitRevision,
        keyboardTitle,
        keyboardSaveStatus: keyboardSave.status,
        keyboardSaveSource: keyboardSave.source,
        keyboardRevision,
        savePostCount: saveRequests.length,
        publicStatus: 404,
      });
    } finally {
      page.off("request", onRequest);
      await server.stop();
    }
  });

  test("fresh managed Content navigation loads versioned runtime assets once", async ({ page, request }) => {
    const server = await startMuddyCollaboration(request, { MUDDY_ASSET_VERSION: FIXTURE_DEPLOYMENT_VERSION });
    const capturedAssetRequests: string[] = [];
    const onRequest = (browserRequest: { method: () => string; url: () => string }) => {
      if (browserRequest.method() !== "GET") return;
      const url = new URL(browserRequest.url());
      if (url.origin !== new URL(server.baseURL).origin) return;
      if (url.pathname === CONTENT_RUNTIME_PATH || url.pathname === MEDIA_RUNTIME_PATH) {
        capturedAssetRequests.push(browserRequest.url());
      }
    };
    page.on("request", onRequest);
    let firstContentRuntime: JSHandle<unknown> | undefined;
    let firstMediaRuntime: JSHandle<unknown> | undefined;

    try {
      // Keep the listener active through the signed-in fresh dashboard load,
      // the managed Content > Pages transition, and the real Edit link. This
      // records actual browser asset requests rather than reconstructing URLs.
      expect(FIXTURE_DEPLOYMENT_VERSION).not.toBe(EXPECTED_MODULE_VERSION);
      await signIn(page, server.baseURL, "admin");
      const pageID = await createLandingPage(page, server.baseURL);
      await page.goto(`${server.baseURL}/admin`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await page.waitForFunction(() => {
        const runtime = (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime;
        return !!runtime;
      }, undefined, { timeout: 30_000 });
      // Noni's shared admin layout intentionally preloads the plain head-tag
      // media runtime with its deployment key. Capture that singleton before
      // the Pages renderer introduces the separately module-keyed managed tag.
      firstMediaRuntime = await page.evaluateHandle(() => (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime || null);
      await clickManagedPages(page, server.baseURL);
      // The Pages index is the first managed Content entry. Anchor the
      // one-executable assertion and singleton handles here, before a later
      // managed detail swap is allowed to clean up dynamic script nodes.
      const indexRenderer = page.locator(PAGE_INDEX_RENDERER).first();
      await expect(indexRenderer).toBeVisible();
      await expect(indexRenderer.locator("[data-content-editor='true']")).toHaveCount(1);
      await expect(indexRenderer.locator("[data-content-editor='true']"))
        .toHaveAttribute("data-content-editor-bound", "true", { timeout: 30_000 });
      await page.waitForFunction(() => {
        const win = window as unknown as { GoSXStudioContentEditorRuntime?: unknown; GoSXStudioMediaRuntime?: unknown };
        return !!win.GoSXStudioContentEditorRuntime && !!win.GoSXStudioMediaRuntime;
      }, undefined, { timeout: 30_000 });
      const firstAssets = await assertManagedAssetURLs(page, request, server.baseURL, capturedAssetRequests, true);
      await expectRuntimeSingleton(page, undefined, firstMediaRuntime, "managed Pages index");
      firstContentRuntime = await page.evaluateHandle(() => (window as unknown as { GoSXStudioContentEditorRuntime?: unknown }).GoSXStudioContentEditorRuntime || null);

      await openManagedPageFromIndex(page, server.baseURL, pageID);
      const detailAssets = await assertManagedAssetURLs(page, request, server.baseURL, capturedAssetRequests, false);
      await expectRuntimeSingleton(page, firstContentRuntime, firstMediaRuntime, "managed Page detail");

      await clickManagedPageIndex(page, server.baseURL);
      await openManagedPageFromIndex(page, server.baseURL, pageID);
      const revisitAssets = await assertManagedAssetURLs(page, request, server.baseURL, capturedAssetRequests, false);
      await expectRuntimeSingleton(page, firstContentRuntime, firstMediaRuntime, "managed Page revisit");

      writeEvidence("page-cms-managed-assets", {
        pageID,
        expectedModuleVersion: EXPECTED_MODULE_VERSION || null,
        capturedAssetRequests,
        firstAssets,
        detailAssets,
        revisitAssets,
        singleton: true,
      });
    } finally {
      await firstContentRuntime?.dispose();
      await firstMediaRuntime?.dispose();
      page.off("request", onRequest);
      await server.stop();
    }
  });
});

async function signIn(page: Page, baseURL: string, identity: string): Promise<void> {
  await page.goto(`${baseURL}/__test/collaboration/signin/${identity}`, { waitUntil: "networkidle", timeout: 60_000 });
}

async function createLandingPage(page: Page, baseURL: string): Promise<string> {
  await gotoEditor(page, baseURL);
  await revealModeIfPresent(page, "advanced");
  const result = await applyCompositionIntentInPlace(page, "create-page:landing", {
    expectedMessage: CREATE_MESSAGE,
    expectedChangeKind: "page",
    requireSelection: false,
  });
  const pageID = result.detail?.change?.pageKey;
  expect(pageID, "create-page should report a concrete Page CMS id").toBeTruthy();
  if (!pageID) throw new Error("create-page should report a concrete Page CMS id");
  return pageID;
}

async function openPageDetail(page: Page, detailURL: string): Promise<void> {
  await page.goto(detailURL, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expectPageDetailReady(page);
}

async function expectPageDetailReady(page: Page): Promise<void> {
  const detail = page.locator(PAGE_DETAIL_RENDERER).first();
  await expect(detail).toBeVisible();
  await expect(detail.locator("[data-content-editor='true']")).toHaveAttribute("data-content-editor-bound", "true", { timeout: 30_000 });
  await expect(detail.locator("input[name='expectedRevision']")).toHaveAttribute("data-content-editor-revision", "true");
}

async function openManagedPageFromIndex(page: Page, baseURL: string, pageID: string): Promise<void> {
  const indexRenderer = page.locator(PAGE_INDEX_RENDERER).first();
  await expect(indexRenderer, "managed Pages navigation must render the Page index").toBeVisible();
  const detailLink = indexRenderer.locator(`a[data-gosx-link='true'][href*='/admin/pages/${pageID}']`).first();
  await expect(detailLink, "managed Pages index must expose the created page's Edit link").toBeVisible();
  const href = await detailLink.getAttribute("href");
  const detailURL = new URL(href ?? "", baseURL);
  expect(detailURL.origin).toBe(new URL(baseURL).origin);
  expect(detailURL.pathname).toBe(`/admin/pages/${pageID}`);
  await Promise.all([
    page.waitForURL((next) => next.origin === detailURL.origin && next.pathname === detailURL.pathname, { timeout: 30_000 }),
    detailLink.click(),
  ]);
  await expectPageDetailReady(page);
}

async function clickManagedPageIndex(page: Page, baseURL: string): Promise<void> {
  const detail = page.locator(PAGE_DETAIL_RENDERER).first();
  const indexLink = detail.locator("a[data-gosx-link='true']").filter({ hasText: /Back to pages|Cancel/ }).first();
  await expect(indexLink, "Page CMS detail must expose a managed link back to Pages").toBeVisible();
  const href = await indexLink.getAttribute("href");
  const indexURL = new URL(href ?? "", baseURL);
  expect(indexURL.origin).toBe(new URL(baseURL).origin);
  expect(indexURL.pathname).toBe("/admin/pages");
  await Promise.all([
    page.waitForURL((next) => next.origin === indexURL.origin && next.pathname === indexURL.pathname, { timeout: 30_000 }),
    indexLink.click(),
  ]);
}

async function assertManagedAssetURLs(
  page: Page,
  request: APIRequestContext,
  baseURL: string,
  capturedRequestURLs: string[],
  requireFreshRequests: boolean,
): Promise<ManagedAssetEvidence> {
  const origin = new URL(baseURL).origin;
  const rendered = await page.locator("script[data-gosx-script='managed']").evaluateAll((nodes, expectedBaseURL) => {
    const base = new URL(String(expectedBaseURL));
    return nodes.map((node) => {
      const src = node.getAttribute("src") || "";
      let url: URL | null = null;
      try {
        url = new URL(src, base);
      } catch {
        // Preserve malformed source details so the assertion below fails with
        // the actual rendered value rather than hiding a bad script node.
      }
      return {
        src,
        href: url?.href || "",
        origin: url?.origin || "",
        pathname: url?.pathname || "",
        releaseKey: url?.searchParams.get("v") || "",
        loadMode: node.getAttribute("data-gosx-script-load") || "",
        loaded: node.getAttribute("data-gosx-script-loaded") || "",
      };
    });
  }, baseURL);
  const paths = [CONTENT_RUNTIME_PATH, MEDIA_RUNTIME_PATH];
  const selected: Record<string, ManagedScriptSnapshot[]> = {};
  for (const assetPath of paths) {
    const matches = rendered.filter((script) => script.origin === origin && script.pathname === assetPath);
    expect(matches.length, `${assetPath} must render a same-origin canonical managed script`).toBeGreaterThan(0);
    expect(matches.every((script) => script.loadMode === "dom"), `${assetPath} must retain the managed DOM loading marker`).toBe(true);
    expect(matches.every((script) => script.loaded === "pending" || script.loaded === "true"), `${assetPath} must use pending/true executable markers`).toBe(true);
    expect(matches.every((script) => script.releaseKey.length > 0), `${assetPath} must use a nonempty ?v= release key: ${JSON.stringify(rendered)}`).toBe(true);
    if (EXPECTED_MODULE_VERSION) {
      expect(matches.every((script) => script.releaseKey === EXPECTED_MODULE_VERSION), `${assetPath} must match GOSX_STUDIO_EXPECTED_MODULE_VERSION`).toBe(true);
    }
    const executed = matches.filter((script) => script.loaded === "true").length;
    if (requireFreshRequests) {
      expect(executed, `${assetPath} should have one actual executable managed script; pending SSR markers may coexist`).toBe(1);
    } else {
      expect(executed, `${assetPath} must not execute more than once on a managed revisit`).toBeLessThanOrEqual(1);
    }
    selected[assetPath] = matches;
  }

  const renderedKeys = paths.flatMap((assetPath) => selected[assetPath].map((script) => script.releaseKey));
  expect(new Set(renderedKeys).size, "content and media managed scripts must share one release key").toBe(1);
  const releaseKey = renderedKeys[0] || "";
  expect(FIXTURE_DEPLOYMENT_VERSION).not.toBe(releaseKey);
  const fetched: Array<{ path: string; url: string; status: number; releaseKey: string }> = [];
  for (const assetPath of paths) {
    const script = selected[assetPath][0];
    const response = await request.get(script.href);
    const responseURL = new URL(response.url(), baseURL);
    expect(response.status(), `${assetPath} canonical managed asset should be fetchable`).toBe(200);
    expect(responseURL.origin).toBe(origin);
    expect(responseURL.pathname).toBe(assetPath);
    expect(responseURL.searchParams.get("v"), `${assetPath} fetched URL must retain the release key`).toBe(releaseKey);
    fetched.push({ path: assetPath, url: response.url(), status: response.status(), releaseKey: responseURL.searchParams.get("v") || "" });
  }

  const captured = capturedRequestURLs.map((raw) => {
    const url = new URL(raw, baseURL);
    return { path: url.pathname, url: raw, releaseKey: url.searchParams.get("v") || "" };
  });
  if (requireFreshRequests) {
    const contentRequests = captured.filter((entry) => entry.path === CONTENT_RUNTIME_PATH && new URL(entry.url, baseURL).origin === origin);
    expect(contentRequests.length, `${CONTENT_RUNTIME_PATH} must be requested during the fresh signed-in Content navigation`).toBeGreaterThan(0);
    expect(contentRequests.every((entry) => entry.releaseKey.length > 0), `${CONTENT_RUNTIME_PATH} browser requests must not use an unversioned legacy URL`).toBe(true);
    expect(contentRequests.every((entry) => entry.releaseKey === releaseKey), `${CONTENT_RUNTIME_PATH} browser requests must use the module release key`).toBe(true);
    if (EXPECTED_MODULE_VERSION) expect(contentRequests.every((entry) => entry.releaseKey === EXPECTED_MODULE_VERSION)).toBe(true);

    const mediaRequests = captured.filter((entry) => entry.path === MEDIA_RUNTIME_PATH && new URL(entry.url, baseURL).origin === origin);
    const allowedMediaKeys = new Set([releaseKey, FIXTURE_DEPLOYMENT_VERSION]);
    expect(mediaRequests.length, `${MEDIA_RUNTIME_PATH} must be requested during the fresh signed-in Content navigation`).toBeGreaterThan(0);
    expect(mediaRequests.every((entry) => entry.releaseKey.length > 0), `${MEDIA_RUNTIME_PATH} browser requests must not use an unversioned legacy URL`).toBe(true);
    expect(mediaRequests.every((entry) => allowedMediaKeys.has(entry.releaseKey)), `${MEDIA_RUNTIME_PATH} browser requests must use only the managed module or outer deployment release key`).toBe(true);
    expect(mediaRequests.some((entry) => entry.releaseKey === releaseKey), `${MEDIA_RUNTIME_PATH} must request the module-keyed managed runtime on Pages`).toBe(true);
    expect(mediaRequests.some((entry) => entry.releaseKey === FIXTURE_DEPLOYMENT_VERSION), `${MEDIA_RUNTIME_PATH} must request the outer admin deployment-keyed runtime`).toBe(true);
    if (EXPECTED_MODULE_VERSION) expect(mediaRequests.some((entry) => entry.releaseKey === EXPECTED_MODULE_VERSION)).toBe(true);
  }

  return {
    rendered,
    fetched,
    capturedRequests: captured,
    executed: Object.fromEntries(paths.map((assetPath) => [assetPath, selected[assetPath].filter((script) => script.loaded === "true").length])),
    pending: Object.fromEntries(paths.map((assetPath) => [assetPath, selected[assetPath].filter((script) => script.loaded === "pending").length])),
  };
}

async function expectRuntimeSingleton(
  page: Page,
  firstContentRuntime: JSHandle<unknown> | undefined,
  firstMediaRuntime: JSHandle<unknown> | undefined,
  label: string,
): Promise<void> {
  if (firstContentRuntime) {
    await expect.poll(
      () => page.evaluate((initial) => (window as unknown as { GoSXStudioContentEditorRuntime?: unknown }).GoSXStudioContentEditorRuntime === initial, firstContentRuntime),
      { timeout: 30_000, message: `${label} must retain the content editor runtime singleton` },
    ).toBe(true);
  }
  if (firstMediaRuntime) {
    await expect.poll(
      () => page.evaluate((initial) => (window as unknown as { GoSXStudioMediaRuntime?: unknown }).GoSXStudioMediaRuntime === initial, firstMediaRuntime),
      { timeout: 30_000, message: `${label} must retain the media runtime singleton` },
    ).toBe(true);
  }
}

async function readPageForm(page: Page): Promise<PageFormSnapshot> {
  const detail = page.locator(PAGE_DETAIL_RENDERER).first();
  return {
    title: await detail.locator("input[name='title']").inputValue(),
    slug: await detail.locator("input[name='slug']").inputValue(),
    body: await detail.locator("textarea[name='body']").inputValue(),
    expectedRevision: await detail.locator("input[name='expectedRevision']").inputValue(),
    published: await detail.locator("input[name='published']").isChecked(),
  };
}

async function firstContentField(page: Page): Promise<Locator> {
  const field = page.locator(
    "[data-content-editor='true'] .content-block__fields input:not([type='hidden']), " +
    "[data-content-editor='true'] .content-block__fields textarea",
  ).first();
  await expect(field, "the Page CMS editor must expose a keyboard-editable block field").toBeVisible();
  return field;
}

async function replaceWithKeyboard(page: Page, field: Locator, value: string): Promise<void> {
  await field.click();
  await field.press("ControlOrMeta+A");
  await page.keyboard.type(value);
}

async function saveDraft(page: Page, pageID: string, trigger: SaveTrigger = "button", keyboardTarget?: Locator): Promise<SaveResult> {
  let mainFrameNavigated = false;
  const onNavigated = (frame: Frame) => {
    if (frame === page.mainFrame()) mainFrameNavigated = true;
  };
  page.on("framenavigated", onNavigated);
  const responsePromise = page.waitForResponse((response) =>
    response.url().includes(`/admin/pages/${pageID}/__actions/save`) &&
    response.request().method() === "POST",
  { timeout: 30_000 });
  try {
    if (trigger === "keyboard") {
      const target = keyboardTarget ?? page.locator("[data-content-editor-search='true']").first();
      await expect(target, "the keyboard save trigger must be a visible focused control").toBeVisible();
      await target.focus();
      await page.keyboard.press("ControlOrMeta+S");
    } else {
      await page.getByRole("button", { name: "Save draft", exact: true }).first().click();
    }
    const response = await responsePromise;
    let payload: ActionResponsePayload | null = null;
    try {
      payload = await response.json() as ActionResponsePayload;
    } catch {
      // A non-JSON failure body is still useful to the caller through status.
    }
    let revisionReadback: string | null = null;
    let source: SaveResult["source"] = "none";
    const hasStructuredSuccess = payload?.ok === true;
    if (hasStructuredSuccess || response.status() === 303) {
      // RuntimeSAVE07 follows the structured action redirect with a real
      // same-route navigation. Wait for that frame event even though the
      // pathname stays the same, then verify the remounted editor so all
      // following assertions observe persisted values rather than the
      // pre-submit DOM.
      await expect.poll(() => mainFrameNavigated, { timeout: 30_000, message: "a successful Page CMS save must remount through its real redirect" }).toBe(true);
      await page.waitForLoadState("domcontentloaded");
      await expectPageDetailReady(page);

      // GoSX action.Redirect deliberately emits HTTP 303 for a successful
      // form action. Playwright cannot read a redirect response body, even
      // though the browser follows it and the runtime receives the
      // structured result. Keep that response payload untouched when JSON is
      // available; for a 303, record only the verified remounted readback so
      // callers cannot mistake it for response.values.
      revisionReadback = await page.locator(`${PAGE_DETAIL_RENDERER} input[name='expectedRevision']`).first().inputValue();
      source = hasStructuredSuccess ? "structured-response" : "remounted-form";
    }
    return { status: response.status(), payload, revisionReadback, source };
  } finally {
    page.off("framenavigated", onNavigated);
  }
}

function expectObservedSuccess(result: SaveResult, label: string, oldRevision: string): string {
  let nextRevision: string | undefined;
  if (result.source === "structured-response") {
    // JSON action responses are the actual response payload, so validate the
    // structured success and token directly. The real remounted form must
    // agree with that payload as an additional persistence check.
    expect(result.payload?.ok, `${label} response must be a structured success`).toBe(true);
    nextRevision = result.payload?.values?.expectedRevision;
    expect(nextRevision, `${label} response must include values.expectedRevision`).toBeTruthy();
    expect(result.revisionReadback, `${label} remounted form must expose the response revision`).toBe(nextRevision);
  } else {
    // GoSX action.Redirect emits HTTP 303 and Playwright cannot expose its
    // redirect response body. Do not invent a payload: validate the observed
    // redirect, real navigation, and remounted form token instead.
    expect(result.source, `${label} must report an observed save source`).toBe("remounted-form");
    expect(result.status, `${label} redirect must be HTTP 303`).toBe(303);
    expect(result.payload?.ok ?? true, `${label} redirect must not carry a failed JSON payload`).toBe(true);
    nextRevision = result.revisionReadback ?? undefined;
    expect(nextRevision, `${label} remounted form must expose values.expectedRevision`).toBeTruthy();
  }
  expect(nextRevision, `${label} save must expose a next revision`).toBeTruthy();
  expect(nextRevision, `${label} save must rotate values.expectedRevision`).not.toBe(oldRevision);
  if (!nextRevision) throw new Error(`${label} observed save did not expose values.expectedRevision`);
  return nextRevision;
}

async function captureScreenshot(page: Page, label: string, width: number, height: number, fullPage = true): Promise<string> {
  await page.setViewportSize({ width, height });
  const filename = `${label}.png`;
  const screenshotPath = path.join(INTEGRATION_ROOT, "screenshots", filename);
  mkdirSync(path.dirname(screenshotPath), { recursive: true });
  await page.screenshot({ path: screenshotPath, fullPage });
  return screenshotPath;
}

async function readConflictLayout(page: Page): Promise<Record<string, unknown>> {
  return page.evaluate(() => {
    const describe = (element: Element | null): Record<string, unknown> | null => {
      if (!element) return null;
      const rect = element.getBoundingClientRect();
      const style = window.getComputedStyle(element);
      return {
        tag: element.tagName.toLowerCase(),
        id: element.id,
        className: typeof element.className === "string" ? element.className : "",
        rect: {
          left: Math.round(rect.left * 100) / 100,
          top: Math.round(rect.top * 100) / 100,
          right: Math.round(rect.right * 100) / 100,
          bottom: Math.round(rect.bottom * 100) / 100,
          width: Math.round(rect.width * 100) / 100,
          height: Math.round(rect.height * 100) / 100,
        },
        display: style.display,
        position: style.position,
        gridTemplateColumns: style.gridTemplateColumns,
        gridTemplateRows: style.gridTemplateRows,
        gridColumn: style.gridColumn,
        alignSelf: style.alignSelf,
        minWidth: style.minWidth,
        maxWidth: style.maxWidth,
        width: style.width,
        overflow: style.overflow,
      };
    };
    const editor = document.querySelector("[data-content-editor='true']");
    const status = editor?.querySelector("[data-content-editor-save-status]") ?? null;
    return {
      viewport: { width: window.innerWidth, height: window.innerHeight },
      document: {
        clientWidth: document.documentElement.clientWidth,
        scrollWidth: document.documentElement.scrollWidth,
      },
      editor: describe(editor),
      status: describe(status),
      label: describe(status?.querySelector("[data-content-editor-save-label]") ?? null),
      detail: describe(status?.querySelector("[data-content-editor-save-detail-text]") ?? null),
      errors: describe(status?.querySelector("[data-content-editor-save-errors]") ?? null),
      conflictLink: describe(status?.querySelector("[data-content-editor-save-conflict-link]") ?? null),
      statusParent: describe(status?.parentElement ?? null),
      editorParent: describe(editor?.parentElement ?? null),
    };
  });
}

async function assertConflictLayout(page: Page, width: number, height: number): Promise<Record<string, unknown>> {
  await page.setViewportSize({ width, height });
  const layout = await readConflictLayout(page);
  const viewport = layout.viewport as { width?: number; height?: number };
  expect(viewport.width, "conflict layout must be sampled at the requested viewport width").toBe(width);
  expect(viewport.height, "conflict layout must be sampled at the requested viewport height").toBe(height);

  const rectOf = (name: string): { width?: number; height?: number } => {
    const element = layout[name] as { rect?: { width?: number; height?: number } } | null | undefined;
    return element?.rect ?? {};
  };
  const statusRect = rectOf("status");
  const detailRect = rectOf("detail");
  const documentLayout = layout.document as { clientWidth?: number; scrollWidth?: number };
  const limits = width <= 820 ? CONFLICT_LAYOUT_LIMITS.mobile : CONFLICT_LAYOUT_LIMITS.desktop;

  expect(detailRect.width, `conflict detail must have a readable width at ${width}x${height}`).toBeGreaterThanOrEqual(limits.detailMinWidth);
  expect(detailRect.height, `conflict detail must stay bounded at ${width}x${height}`).toBeLessThanOrEqual(limits.maxHeight);
  expect(statusRect.height, `conflict status must stay bounded at ${width}x${height}`).toBeLessThanOrEqual(limits.maxHeight);
  expect(documentLayout.scrollWidth, `conflict surface must not introduce horizontal overflow at ${width}x${height}`).toBeLessThanOrEqual((documentLayout.clientWidth ?? width) + 1);

  const editor = page.locator("[data-content-editor='true']").first();
  await expect(editor.locator("[data-content-editor-save-status]")).toBeVisible();
  await expect(editor.locator("[data-content-editor-save-detail-text]")).toContainText(/Your draft remains local/i);
  await expect(editor.locator("[data-content-editor-save-conflict-link]")).toBeVisible();
  return layout;
}

async function clickManagedPages(page: Page, baseURL: string): Promise<void> {
  const contentSummary = page.locator("nav[aria-label='Main'] details summary", { hasText: "Content" }).first();
  await expect(contentSummary, "the dashboard must expose the managed Content navigation").toBeVisible();
  const contentMenu = contentSummary.locator("xpath=..");
  if ((await contentMenu.getAttribute("open")) === null) await contentSummary.click();

  const pagesLink = page.locator("nav[aria-label='Main'] a[data-gosx-link='true'][href$='/admin/pages']").first();
  await expect(pagesLink).toBeVisible();
  const href = await pagesLink.getAttribute("href");
  const url = new URL(href ?? "", baseURL);
  expect(url.origin).toBe(new URL(baseURL).origin);
  expect(url.pathname).toBe("/admin/pages");
  await Promise.all([
    page.waitForURL((next) => next.origin === url.origin && next.pathname === url.pathname),
    pagesLink.click(),
  ]);
}

function writeEvidence(name: string, value: Record<string, unknown>): void {
  mkdirSync(INTEGRATION_ROOT, { recursive: true });
  writeFileSync(path.join(INTEGRATION_ROOT, `${name}.json`), JSON.stringify({ generatedAt: new Date().toISOString(), ...value }, null, 2));
}
