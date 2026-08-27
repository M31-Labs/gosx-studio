import { expect, test, type Frame, type Locator, type Page } from "@playwright/test";
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
const INTEGRATION_ROOT = path.resolve(__dirname, "../.tiller/scratch/codex/enterprise-polish-20260827/integration08");
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

async function saveDraft(page: Page, pageID: string): Promise<SaveResult> {
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
    await page.getByRole("button", { name: "Save draft", exact: true }).first().click();
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
