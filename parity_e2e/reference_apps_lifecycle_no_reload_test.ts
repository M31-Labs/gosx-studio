import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { installEditorDocumentContinuityProbe } from "./reference_apps_document_continuity";
import { gotoEditor, revealModeIfPresent, startPajaritos } from "./reference_apps_harness";

const DIRECT_EDIT_FORM = "#pajaritosDirectEditForm";
const DIRECT_EDIT_VALUE = "[data-studio-operation-value]";
const DIRECT_EDIT_SAVE = "[data-gosx-studio-operation-kind='set-field']";
const PUBLISH_CONTROLS = ".studio-publish-controls";
const SCHEDULE_INPUT = `${PUBLISH_CONTROLS} input[name='publishAt']`;
const APPROVE_BUTTON = `${PUBLISH_CONTROLS} button[formaction*='approvePublish']`;
const SCHEDULE_BUTTON = `${PUBLISH_CONTROLS} button[formaction*='schedulePublish']`;
const PROCESS_DUE_BUTTON = `${PUBLISH_CONTROLS} button[formaction*='processDuePublishes']`;
const RESTORE_BUTTONS = "[data-pajaritos-restore-points='true'] [data-pajaritos-restore-revision]";
const COLLAB_PANEL = "[data-gosx-studio-collaboration='true']";
const PENDING_DIFF = "[data-pajaritos-pending-diff='true']";
const PENDING_CHANGE = `${PENDING_DIFF} [data-pajaritos-pending-change]`;

async function waitForAuthoringResult(page: Page) {
  return page.evaluate(
    () => new Promise<{ fragmentCount?: number; previewCount?: number; result?: { message?: string } }>((resolve, reject) => {
      const timeout = window.setTimeout(() => reject(new Error("timed out waiting for gosxstudio:authoring-result")), 30_000);
      document.addEventListener("gosxstudio:authoring-result", (event) => {
        window.clearTimeout(timeout);
        resolve((event as CustomEvent).detail || {});
      }, { once: true });
    }),
  );
}

async function waitForActionResult(page: Page) {
  return page.evaluate(
    () => new Promise<{ ok?: boolean; error?: string; status?: number; fieldErrors?: Record<string, string>; body?: { fieldErrors?: Record<string, string> }; authoringHandled?: boolean; result?: { message?: string; data?: { message?: string } } }>((resolve, reject) => {
      const timeout = window.setTimeout(() => reject(new Error("timed out waiting for gosxstudio:action-result")), 30_000);
      document.addEventListener("gosxstudio:action-result", (event) => {
        window.clearTimeout(timeout);
        resolve((event as CustomEvent).detail || {});
      }, { once: true });
    }),
  );
}

async function expectCollaborationReady(page: Page) {
  const panel = page.locator(COLLAB_PANEL).first();
  if (await panel.count() === 0) return;
  await expect(
    panel,
    "Pajaritos collaboration runtime must grant capabilities before durable operation buttons enable",
  ).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
}

async function commitDirectEdit(page: Page, marker: string) {
  const form = page.locator(DIRECT_EDIT_FORM);
  await expect(form, "Pajaritos durable direct-edit form must render").toBeAttached({ timeout: 30_000 });
  await expect(form.locator(DIRECT_EDIT_SAVE), "durable direct-edit save must be enabled by the collaboration capability handshake").toBeEnabled({ timeout: 20_000 });
  const committed = page.evaluate(
    () => new Promise<{ kind: "committed" | "error"; detail: unknown }>((resolve) => {
      document.addEventListener("gosxstudio:operation-committed", (event) => resolve({ kind: "committed", detail: (event as CustomEvent).detail }), { once: true });
      document.addEventListener("gosxstudio:operation-error", (event) => resolve({ kind: "error", detail: (event as CustomEvent).detail }), { once: true });
    }),
  );
  await form.locator(DIRECT_EDIT_VALUE).fill(marker);
  await form.locator(DIRECT_EDIT_SAVE).click();
  const outcome = await committed;
  expect(outcome.kind, `direct edit durable operation should commit: ${JSON.stringify(outcome)}`).toBe("committed");
}

function duePublishValue(): string {
  const due = new Date();
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${due.getFullYear()}-${pad(due.getMonth() + 1)}-${pad(due.getDate())}T${pad(due.getHours())}:${pad(due.getMinutes())}`;
}

async function clickLifecycleButton(page: Page, selector: string, expectedMessage: RegExp, expectedPreviewRefresh: boolean) {
  const actionResultPromise = waitForActionResult(page);
  const authoringResultPromise = waitForAuthoringResult(page);
  await page.locator(selector).first().evaluate((element) => (element as HTMLButtonElement).click());
  const [actionResult, authoringResult] = await Promise.all([actionResultPromise, authoringResultPromise]);
  expect(actionResult.ok, `lifecycle action failed: ${JSON.stringify(actionResult)}`).toBe(true);
  expect(actionResult.authoringHandled, `state runtime must bridge lifecycle JSON into authoring fragment refresh: ${JSON.stringify(actionResult)}`).toBe(true);
  expect(actionResult.result?.message ?? actionResult.result?.data?.message ?? "").toMatch(expectedMessage);
  expect(authoringResult.result?.message ?? "").toMatch(expectedMessage);
  expect(authoringResult.fragmentCount ?? 0, `lifecycle action should refresh host fragments: ${JSON.stringify(authoringResult)}`).toBeGreaterThan(0);
  if (expectedPreviewRefresh) {
    expect(authoringResult.previewCount ?? 0, `content-changing lifecycle action should refresh preview: ${JSON.stringify(authoringResult)}`).toBeGreaterThan(0);
  } else {
    expect(authoringResult.previewCount ?? 0, `approval/schedule should not refresh preview: ${JSON.stringify(authoringResult)}`).toBe(0);
  }
}

async function expectPendingDraftContains(page: Page, marker: string) {
  await expect(page.locator(PENDING_DIFF), "pending-diff fragment should stay fresh after lifecycle actions").toHaveAttribute("data-pajaritos-has-changes", "true", { timeout: 30_000 });
  await expect(page.locator(`${PENDING_CHANGE}[data-after="${marker}"]`).first(), `pending-diff should include data-after=${marker}`).toBeAttached({ timeout: 30_000 });
}

async function publishDueDraft(page: Page, request: APIRequestContext, baseURL: string, draftMarker: string, liveMarker: string) {
  await revealModeIfPresent(page, "advanced");
  await expect(page.locator(PUBLISH_CONTROLS)).toBeAttached({ timeout: 30_000 });
  await clickLifecycleButton(page, APPROVE_BUTTON, /Publish approved\./, false);
  await expectPendingDraftContains(page, draftMarker);
  await expectStorefrontContains(request, baseURL, liveMarker, draftMarker);
  await page.locator(SCHEDULE_INPUT).fill(duePublishValue());
  await clickLifecycleButton(page, SCHEDULE_BUTTON, /Publish scheduled\./, false);
  await expectPendingDraftContains(page, draftMarker);
  await expectStorefrontContains(request, baseURL, liveMarker, draftMarker);
  await clickLifecycleButton(page, PROCESS_DUE_BUTTON, /Processed 1 scheduled publish\./, true);
  await expect(page.locator(PENDING_DIFF), "publish should refresh pending-diff to empty without editor reload").toHaveAttribute("data-pajaritos-has-changes", "false", { timeout: 30_000 });
}

async function restoreRevisionIDForHeadline(page: Page, headline: string): Promise<string> {
  const checkpoint = page.locator(`[data-pajaritos-restore-points='true'] [data-pajaritos-restore-point][data-pajaritos-restore-headline="${headline}"]`).first();
  await expect(checkpoint, `restore point for ${headline} should render`).toBeAttached({ timeout: 30_000 });
  const id = await checkpoint.getAttribute("data-pajaritos-restore-point");
  expect(id, "restore point must expose a durable revision id").toBeTruthy();
  return id || "";
}

async function expectPreviewFrameContains(page: Page, present: string, absent?: string) {
  const body = page.frameLocator("iframe[title='Pajaritos home page authoring canvas']").locator("body");
  await expect(body, `preview frame should contain ${present}`).toContainText(present, { timeout: 30_000 });
  if (absent) await expect(body, `preview frame should not contain ${absent}`).not.toContainText(absent);
}

async function expectStorefrontContains(request: APIRequestContext, baseURL: string, present: string, absent?: string) {
  await expect.poll(async () => {
    const response = await request.get(`${baseURL}/`);
    return await response.text();
  }, { timeout: 30_000, intervals: [250, 500, 1000] }).toContain(present);
  if (absent) {
    const response = await request.get(`${baseURL}/`);
    const html = await response.text();
    expect(html).not.toContain(absent);
  }
}

test.describe("@reference-apps Pajaritos lifecycle no-reload host workflow", () => {
  test.describe.configure({ timeout: 240_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Pajaritos save, publish, structured schedule error, and restore stay on the same editor document", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));
    page.on("dialog", (dialog) => { void dialog.accept(); });

    const server = await startPajaritos(request);
    try {
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");
      await expectCollaborationReady(page);

      const nonce = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      const markerV1 = `Pajaritos lifecycle V1 ${nonce}`;
      const markerV2 = `Pajaritos lifecycle V2 ${nonce}`;

      const probe = await installEditorDocumentContinuityProbe(page, { settleMs: 125 });
      try {
        await page.locator(SCHEDULE_INPUT).fill("");
        const responsePromise = page.waitForResponse((response) => response.url().includes("/__actions/schedulePublish") && response.request().method() === "POST");
        const resultPromise = waitForActionResult(page);
        await page.locator(SCHEDULE_BUTTON).first().evaluate((element) => (element as HTMLButtonElement).click());
        const [response, result] = await Promise.all([responsePromise, resultPromise]);
        expect(response.status()).toBe(422);
        expect(result.ok, `empty schedule should surface structured action failure: ${JSON.stringify(result)}`).toBe(false);
        expect(result.status, `empty schedule should preserve HTTP 422 status: ${JSON.stringify(result)}`).toBe(422);
        expect(result.error ?? "").toContain("Choose a valid publish time.");
        expect(result.fieldErrors?.publishAt ?? result.body?.fieldErrors?.publishAt ?? "", `empty schedule should preserve fieldErrors: ${JSON.stringify(result)}`).toContain("publish time");
        await probe.assertStillOnSameDocument();

        await commitDirectEdit(page, markerV1);
        await probe.assertStillOnSameDocument();
        await expectStorefrontContains(request, server.baseURL, "A forest school where curiosity gets muddy.", markerV1);
        await publishDueDraft(page, request, server.baseURL, markerV1, "A forest school where curiosity gets muddy.");
        await probe.assertStillOnSameDocument();
        await expectStorefrontContains(request, server.baseURL, markerV1);

        await commitDirectEdit(page, markerV2);
        await probe.assertStillOnSameDocument();
        await expectStorefrontContains(request, server.baseURL, markerV1, markerV2);
        await publishDueDraft(page, request, server.baseURL, markerV2, markerV1);
        await probe.assertStillOnSameDocument();
        await expectStorefrontContains(request, server.baseURL, markerV2, markerV1);
        const markerV1RestoreRevisionID = await restoreRevisionIDForHeadline(page, markerV1);

        await expect(page.locator(RESTORE_BUTTONS).first(), "two publishes should expose restore controls without reloading").toBeAttached({ timeout: 30_000 });
        await expect.poll(async () => page.locator(RESTORE_BUTTONS).count(), { timeout: 30_000 }).toBeGreaterThanOrEqual(2);
        const markerV1RestoreButton = page.locator(`[data-pajaritos-restore-point="${markerV1RestoreRevisionID}"] button[data-pajaritos-restore-revision]`).first();
        await expect(markerV1RestoreButton, "restore should target the durable revision captured immediately after the V1 publish").toBeAttached({ timeout: 30_000 });
        await clickLifecycleButton(page, `[data-pajaritos-restore-point="${markerV1RestoreRevisionID}"] button[data-pajaritos-restore-revision]`, /Restore point applied\./, true);
        await probe.assertStillOnSameDocument();
        await expect(page.locator(DIRECT_EDIT_FORM).locator(DIRECT_EDIT_VALUE), "visible durable direct-edit control should refresh to restored V1 before any document reload").toHaveValue(markerV1, { timeout: 30_000 });
        await expectPreviewFrameContains(page, markerV1, markerV2);
        await expectStorefrontContains(request, server.baseURL, markerV1, markerV2);
      } finally {
        probe.dispose();
      }

      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator(DIRECT_EDIT_FORM).locator(DIRECT_EDIT_VALUE)).toHaveValue(markerV1, { timeout: 30_000 });
      await expectStorefrontContains(request, server.baseURL, markerV1, markerV2);

      const relevantErrors = consoleErrors.filter((line) => /gosxstudio|authoring|state_runtime|workbench|pajaritos/i.test(line) && !/relay\.js/i.test(line));
      expect(relevantErrors, `unexpected editor console errors: ${JSON.stringify(relevantErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});
