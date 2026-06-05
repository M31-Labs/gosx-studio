import { expect, test, type Page, type Request } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M3 RESTORE-REVISION loop on the muddy website editor —
// proven end-to-end in a real headless Chromium against ONE running muddy server
// (so live/draft/revision state persists in-process across navigations and across
// the editor/public-storefront split).
//
// This guards a real in-browser bug we just fixed: the RevisionHistoryPanel used
// to render each revision's "Restore this version" submit inside a NESTED
// <form method="post" action={revision.restoreAction}>, and that panel sits inside
// the workbench <form id="websiteEditorForm" data-gosx-studio-state>. Nested forms
// are invalid HTML → the browser parser DROPS the inner form, so the Restore button
// re-associated to the workbench form and POSTed to the wrong (save) action; restore
// did NOT work in-browser. The fix mirrors the proven workbench-form button pattern
// (Navigation/Checkout/Publish panels): the Restore control is now a real native
//   <button type="submit" form="websiteEditorForm"
//           formaction={restoreRevision} formmethod="post"
//           name="revisionId" value={revision.id} data-studio-submit-action="restoreRevision">
// Because the button submits the WHOLE workbench form (not a per-row nested form),
// the revision id travels via the submitter button's own name/value (only the
// clicked submit button's name/value is sent). The restoreRevision handler reads
// ctx.FormData["revisionId"] (a single field) and derives the resource from the
// revision id, so name=revisionId value=<id> is sufficient.
//
// Flow (mirrors reference_apps_canvas_draft_publish_test.ts):
//   (1) Edit the hero to a unique per-run marker V1, Publish → V1 goes live and an
//       editor.published revision (snapshotting V1) is recorded.
//   (2) Edit the hero to a unique per-run marker V2, Publish → V2 goes live; a SECOND
//       editor.published revision is recorded. V1 is now a PRIOR revision in the
//       history panel.
//   (3) In the RevisionHistoryPanel, NATIVELY CLICK the real "Restore this version"
//       button for the V1 revision row (the now-fixed native button). The revision
//       ordering is most-recent-first: row[0] snapshots the current live (V2) and
//       row[1] snapshots the prior live (V1) — verified by a muddy Go probe — so we
//       click the SECOND restore button to restore V1. We await the restoreRevision
//       action response and assert it is NOT 403 (the native multipart submit's
//       csrf_token now validates) and is 2xx/3xx.
//   (4) Assert the restore took effect: the live public storefront "/" AND the editor
//       both reflect V1 again (restored), not V2. Markers are freshly generated per
//       run (timestamp + random nonce, HTML-safe chars only) so no assertion can
//       false-pass on stale/seed content; the restore assertion polls/awaits.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy build rebuilt against the
// gosx/gosx-studio under test (`go run ./cmd/muddy-noni` via the shared harness).

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const OVERLAY_HERO = "[data-gosx-canvas-html='hero']";
const EDITABLE_HERO = "[data-gosx-html-key='hero']";
const FIELD_HERO = "[data-studio-field='home.hero.headline']";
const AUTHORING_ROUTE = "/admin/editor/__actions/authoring";
const PUBLISH_ROUTE = "/admin/editor/__actions/publish";
const RESTORE_ROUTE = "/admin/editor/__actions/restoreRevision";
const BINDING = "home.hero.headline";

const REVISION_HISTORY = "[data-studio-revision-history='true']";
const RESTORE_BUTTON = "[data-studio-submit-action='restoreRevision']";
const PUBLISH_BUTTON = "[data-studio-submit-action='publish']";

const SEED_HEADLINE = "Ceramic pieces with evidence of the hand";

test.describe("@reference-apps canvas2d site-map WASM-free restore-revision loop", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni wasm-free: Publish V1, Publish V2 (live=V2), then the REAL 'Restore this version' button on the V1 revision row restores V1 to live (storefront + editor) — proving the workbench-form restore button reaches restoreRevision in-browser", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    // The Restore/Publish buttons carry data-admin-confirm; cms-actions.js fires
    // window.confirm on any submit. Auto-accept so no confirm prompt stalls the test.
    page.on("dialog", (dialog) => { void dialog.accept(); });

    // Freshly generated per run so neither the storefront nor editor assertions can
    // false-pass on stale/seed content. HTML-safe chars only (survives html escaping
    // in textContent + raw HTML byte-for-byte).
    const nonce = Math.random().toString(36).slice(2, 10);
    const markerV1 = `M3 restore v1 ${Date.now()} ${nonce}`;
    const markerV2 = `M3 restore v2 ${Date.now()} ${nonce}b`;

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await openEditorToHeroSurface(page, server.baseURL, consoleErrors);

      const seedText = (await editableText(page)).trim();
      expect(markerV1, "the V1 marker must differ from the seeded headline").not.toBe(seedText);
      expect(markerV2, "the V2 marker must differ from the seeded headline").not.toBe(seedText);

      const storefrontSeed = await fetchStorefront(request, server.baseURL);
      expect(storefrontSeed, "the public storefront should start on the seed headline").toContain(SEED_HEADLINE);
      expect(storefrontSeed, "no marker should exist on the storefront before any edit").not.toContain(markerV1);
      expect(storefrontSeed, "no marker should exist on the storefront before any edit").not.toContain(markerV2);

      // ── (1) EDIT → V1 → PUBLISH (V1 live, revision recorded) ───────────────────
      await commitInlineEdit(page, markerV1, consoleErrors);
      const publishV1Status = await triggerPublish(page, server.baseURL);
      expect([200, 303].includes(publishV1Status), `Publish V1 should be accepted/redirected; got ${publishV1Status}`).toBe(true);

      const storefrontAfterV1 = await fetchStorefront(request, server.baseURL);
      expect(storefrontAfterV1, "after Publish V1 the storefront must show V1 (now live)").toContain(markerV1);

      // ── (2) EDIT → V2 → PUBLISH (V2 live; V1 becomes a prior revision) ─────────
      await commitInlineEdit(page, markerV2, consoleErrors);
      const publishV2Status = await triggerPublish(page, server.baseURL);
      expect([200, 303].includes(publishV2Status), `Publish V2 should be accepted/redirected; got ${publishV2Status}`).toBe(true);

      const storefrontAfterV2 = await fetchStorefront(request, server.baseURL);
      expect(storefrontAfterV2, "after Publish V2 the storefront must show V2 (now live)").toContain(markerV2);
      expect(storefrontAfterV2, "after Publish V2 the storefront must NOT still show V1").not.toContain(markerV1);

      // ── (3) RESTORE V1 via the REAL fixed restore button ──────────────────────
      // Reveal the publish-mode area (the RevisionHistoryPanel lives there) and
      // assert the history panel + at least two revision rows (V2 most-recent, V1
      // prior) rendered. The restore button is a native workbench-form submit; if
      // the OLD nested-form bug were present the buttons would not carry
      // data-studio-submit-action='restoreRevision' or would post to the save action.
      await revealModeIfPresent(page, "publish");
      await expect(
        page.locator(REVISION_HISTORY).first(),
        "the revision history panel must render after two publishes",
      ).toBeAttached({ timeout: 30_000 });

      const restoreButtons = page.locator(`${REVISION_HISTORY} ${RESTORE_BUTTON}`);
      await expect(
        restoreButtons,
        "at least two revision rows (V2 then V1) must each carry a native restore button",
      ).toHaveCount(2, { timeout: 30_000 });

      // Revision ordering is most-recent-first: row[0] restores the current live (V2),
      // row[1] restores the prior live (V1). Click row[1] to restore V1.
      const restoreV1Button = restoreButtons.nth(1);
      // The history panel lives in a scrollable publish-mode column; bring the button
      // (and every scrollable ancestor) into view so the real click can land.
      await restoreV1Button.evaluate((el) => {
        let node: HTMLElement | null = el as HTMLElement;
        while (node) {
          if (node.scrollHeight > node.clientHeight || node.scrollWidth > node.clientWidth) {
            node.scrollTop = node.scrollTop; // touch to ensure layout
          }
          node = node.parentElement;
        }
        (el as HTMLElement).scrollIntoView({ block: "center", inline: "center" });
      });
      await restoreV1Button.scrollIntoViewIfNeeded().catch(() => undefined);
      await expect(restoreV1Button, "the V1-row restore button must be present to click").toBeAttached({ timeout: 30_000 });

      // The fixed button is a real <button type=submit form=websiteEditorForm
      // formaction=...restoreRevision name=revisionId value=<id>>: verify the submitter
      // contract before clicking (this is exactly what made the in-browser restore work).
      expect(await restoreV1Button.getAttribute("type"), "restore control must be a submit button").toBe("submit");
      expect(await restoreV1Button.getAttribute("name"), "restore submitter must carry name=revisionId").toBe("revisionId");
      const formaction = await restoreV1Button.getAttribute("formaction");
      expect(formaction ?? "", "restore submitter must formaction the restoreRevision action").toContain("restoreRevision");
      const v1RevisionID = await restoreV1Button.getAttribute("value");
      expect(v1RevisionID ?? "", "restore submitter must carry the revision id in value").toMatch(/\S/);

      // Native click submits websiteEditorForm; the studio state runtime re-sends it
      // as a multipart fetch carrying csrf_token (read via r.FormValue). Await the
      // restoreRevision response and assert it reaches the handler non-403.
      const restoreResponsePromise = page.waitForResponse(
        (response) => response.url().includes(RESTORE_ROUTE) && response.request().method() === "POST",
        { timeout: 30_000 },
      );
      // The publish-mode RevisionHistoryPanel can sit in an off-viewport mode column
      // (the workbench mode layout positions inactive/overflowing panels off-screen),
      // so a pointer click cannot land. We instead invoke the REAL button's native
      // HTMLElement.click() in-page: this dispatches a genuine click on the actual
      // <button type="submit" form="websiteEditorForm">, firing the form's submit
      // event so the studio STATE runtime intercepts it and re-sends the workbench
      // form (with this submitter's name=revisionId value=<id>) as the multipart
      // restoreRevision fetch — the exact code path a user click exercises.
      await restoreV1Button.evaluate((el) => (el as HTMLButtonElement).click());
      const restoreResponse = await restoreResponsePromise;
      const restoreStatus = restoreResponse.status();
      expect(
        restoreStatus,
        `the native Restore click must reach restoreRevision and NOT 403 (multipart csrf_token now validates); got ${restoreStatus}`,
      ).not.toBe(403);
      expect(
        [200, 303].includes(restoreStatus),
        `the Restore action must be accepted/redirected (2xx/3xx); got ${restoreStatus}`,
      ).toBe(true);

      // The clicked submitter must have carried the V1 revision id (only the clicked
      // submit button's name/value is sent — the whole point of the fix).
      const restoreBody = restoreResponse.request().postData() ?? "";
      expect(
        restoreBody.includes(`revisionId`) && restoreBody.includes(v1RevisionID ?? "__none__"),
        `the restore POST body must carry revisionId=${v1RevisionID} from the submitter; body-head=${restoreBody.slice(0, 300)}`,
      ).toBe(true);

      // ── (4) ASSERT RESTORE TOOK EFFECT: live is V1 again, not V2 ───────────────
      const storefrontAfterRestore = await pollForStorefront(request, server.baseURL, markerV1);
      expect(
        storefrontAfterRestore,
        `after Restore the public storefront "/" must show V1 again (restored to live); html-head=${storefrontAfterRestore.slice(0, 200)}`,
      ).toContain(markerV1);
      expect(
        storefrontAfterRestore,
        "after Restore the storefront must NOT show V2 (it was replaced by the restored V1)",
      ).not.toContain(markerV2);

      // The editor (reads live after restore, no draft) reflects V1 too.
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await waitForHeroEditable(page);
      const editorAfterRestore = await pollForEditableText(page, markerV1);
      expect(
        editorAfterRestore,
        `after Restore the editor hero must show the restored V1 value; got=${JSON.stringify(editorAfterRestore)}`,
      ).toContain(markerV1);
      expect(
        editorAfterRestore,
        "after Restore the editor hero must NOT show V2",
      ).not.toContain(markerV2);

      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|inline-edit|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// openEditorToHeroSurface boots the editor route and waits for the full WASM-free
// canvas + inline-edit bridge + the contenteditable hero overlay to mount.
async function openEditorToHeroSurface(page: Page, baseURL: string, consoleErrors: string[]) {
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
  await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (selection sink)").toBeAttached();

  const canvas = page.locator(CANVAS_SELECTOR).first();
  await expect(canvas, "the muddy WASM-free canvas must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

  await page.waitForFunction(() => {
    const w = window as unknown as Record<string, unknown>;
    const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
    const painter = w.__muddyCanvas2DPainter as { paint?: unknown; renderCanvasBoardHTML?: unknown } | undefined;
    const inlineEdit = w.__muddyCanvasInlineEdit as { install?: unknown; persist?: unknown } | undefined;
    const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { __muddyCanvasWasmFree?: unknown }) | null;
    return !!painter && typeof painter.paint === "function" && typeof painter.renderCanvasBoardHTML === "function" &&
      !!rt && typeof rt.setState === "function" &&
      !!inlineEdit && typeof inlineEdit.install === "function" && typeof inlineEdit.persist === "function" &&
      document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
      !!el && !!el.__muddyCanvasWasmFree;
  }, null, { timeout: 120_000 });

  await waitForHeroEditable(page);
  await expect(
    page.locator(`${EDITABLE_HERO}${FIELD_HERO}`).first(),
    "the editable hero must carry the store binding data-studio-field='home.hero.headline'",
  ).toBeAttached();
  expect(await page.locator(EDITABLE_HERO).first().getAttribute("contenteditable"), "the hero element must be contenteditable").toBe("true");
  void consoleErrors;
}

async function waitForHeroEditable(page: Page) {
  const overlayHero = page.locator(OVERLAY_HERO).first();
  await expect(
    overlayHero,
    "the painter must mount the html surface as an overlay element [data-gosx-canvas-html='hero'] above the canvas",
  ).toBeAttached({ timeout: 90_000 });
  const editable = page.locator(EDITABLE_HERO).first();
  await expect(editable, "the host markup's contenteditable hero must render inside the overlay").toBeAttached({ timeout: 30_000 });
  await expect(editable, "the editable hero element must be visible (overlay sized + positioned)").toBeVisible({ timeout: 30_000 });
}

// commitInlineEdit selects the hero surface, replaces its text with the marker,
// blurs to commit, and waits for the autosave POST (draft save) to resolve.
async function commitInlineEdit(page: Page, marker: string, consoleErrors: string[]) {
  const editable = page.locator(EDITABLE_HERO).first();
  await editable.click();
  await pollForSelectedNode(page, "hero");

  const authoringPostPromise = page.waitForRequest(
    (req: Request) => req.url().includes(AUTHORING_ROUTE) && req.method() === "POST",
    { timeout: 30_000 },
  );
  const authoringResponsePromise = page.waitForResponse(
    (resp) => resp.url().includes(AUTHORING_ROUTE) && resp.request().method() === "POST",
    { timeout: 30_000 },
  );

  await editable.click();
  await page.waitForFunction((sel) => {
    const el = document.querySelector(sel);
    return !!el && document.activeElement === el;
  }, EDITABLE_HERO, { timeout: 10_000 });
  await page.keyboard.press("ControlOrMeta+A");
  await page.keyboard.type(marker, { delay: 8 });

  const landed = await pollForEditableText(page, marker);
  expect(
    landed,
    `typed marker must land in the surface DOM before commit; got=${JSON.stringify(landed)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
  ).toContain(marker);

  await page.evaluate((sel) => {
    const el = document.querySelector(sel) as HTMLElement | null;
    if (el && typeof el.blur === "function") el.blur();
    const board = document.querySelector("[data-studio-site-map-board='true']") as HTMLElement | null;
    if (board && typeof board.focus === "function") {
      try { board.focus(); } catch { /* tolerate */ }
    }
  }, EDITABLE_HERO);

  const postRequest = await authoringPostPromise;
  const body = postRequest.postData() ?? "";
  const params = new URLSearchParams(body);
  expect(
    params.get("gosx_studio_binding"),
    `commit POST body must carry gosx_studio_binding=${BINDING}; body=${JSON.stringify(body)}`,
  ).toBe(BINDING);
  expect(
    params.get("gosx_studio_value"),
    `commit POST body must carry the new marker as gosx_studio_value; body=${JSON.stringify(body)}`,
  ).toBe(marker);

  const postResponse = await authoringResponsePromise;
  expect(
    [200, 303].includes(postResponse.status()),
    `draft autosave POST should be accepted by the authoring action; got status ${postResponse.status()}`,
  ).toBe(true);
}

// triggerPublish fires the REAL publish-panel button (native click), returns the
// action's HTTP status (asserting non-403), then reloads the editor so its
// server-rendered state reflects the result.
async function triggerPublish(page: Page, baseURL: string): Promise<number> {
  void baseURL;
  await revealModeIfPresent(page, "publish");
  const button = page.locator(PUBLISH_BUTTON).first();
  await expect(button, "the Publish button must be present + visible to trigger publish").toBeVisible({ timeout: 30_000 });

  const responsePromise = page.waitForResponse(
    (response) => response.url().includes(PUBLISH_ROUTE) && response.request().method() === "POST",
    { timeout: 30_000 },
  );
  await button.click();
  const response = await responsePromise;
  const status = response.status();
  expect(
    status,
    `the native Publish click must NOT 403 (gosx reads the multipart csrf_token); got ${status}`,
  ).not.toBe(403);

  await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
  await waitForHeroEditable(page);
  return status;
}

// fetchStorefront GETs the PUBLIC storefront "/" (renders store.Settings() — LIVE).
async function fetchStorefront(request: { get: (url: string, opts?: { failOnStatusCode?: boolean; timeout?: number }) => Promise<{ text(): Promise<string> }> }, baseURL: string): Promise<string> {
  const response = await request.get(`${baseURL}/`, { failOnStatusCode: false, timeout: 60_000 });
  return response.text();
}

// pollForStorefront re-fetches the live storefront until it contains `want` (or a
// deadline), making the restore-took-effect assertion robust against any redirect/
// re-render latency after the restore action.
async function pollForStorefront(request: { get: (url: string, opts?: { failOnStatusCode?: boolean; timeout?: number }) => Promise<{ text(): Promise<string> }> }, baseURL: string, want: string): Promise<string> {
  const deadline = Date.now() + 15_000;
  let last = "";
  while (Date.now() < deadline) {
    last = await fetchStorefront(request, baseURL);
    if (last.includes(want)) return last;
    await new Promise((resolve) => setTimeout(resolve, 250));
  }
  return last;
}

async function readSelectedNode(page: Page): Promise<string | null> {
  return page.evaluate(({ board, attr }) => {
    const root = document.querySelector(board);
    return root ? root.getAttribute(attr) : null;
  }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR });
}

async function pollForSelectedNode(page: Page, want: string): Promise<string | null> {
  const deadline = Date.now() + 10_000;
  let last: string | null = null;
  while (Date.now() < deadline) {
    last = await readSelectedNode(page);
    if (last === want) return last;
    await page.waitForTimeout(120);
  }
  return last;
}

async function editableText(page: Page): Promise<string> {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel);
    return el ? (el.textContent || "") : "";
  }, EDITABLE_HERO);
}

async function pollForEditableText(page: Page, want: string): Promise<string> {
  const deadline = Date.now() + 15_000;
  let last = "";
  while (Date.now() < deadline) {
    last = await editableText(page);
    if (last.includes(want)) return last;
    await page.waitForTimeout(120);
  }
  return last;
}
