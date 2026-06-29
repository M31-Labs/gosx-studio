import { expect, test, type Page, type Request } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M3 DRAFT → PUBLISH → DISCARD loop on the muddy website
// editor — proven end-to-end in a real headless Chromium against ONE running
// muddy server (so live/draft state persists in-process across navigations and
// across the editor/public-storefront split).
//
// Builds on the M2 inline-edit fixture (MUDDY_CANVAS_WASM_FREE=1 +
// MUDDY_EDITOR_SCENE_DOM=1, see reference_apps_canvas_inline_edit_test.ts): the
// host injects ONE Kind:"html" "page:home" full-page surface whose nested
// contenteditable <h1> is bound to home.hero.headline. M3 changed where an inline-edit commit lands: the muddy
// authoring adapter (saveHeroHeadline) now writes store.SaveDraftSettings — an
// editor-scoped DRAFT — instead of touching live settings. Two render paths read
// two different sources, which is the entire point of this test:
//   • the EDITOR renders store.SettingsForEditor() (draft-when-present, else live)
//   • the PUBLIC storefront ("/") renders store.Settings() (LIVE only)
// A Publish button promotes draft→live (store.PublishSettingsDraft, recording a
// versioned editor.published revision + a live→draft changelog); a Discard button
// drops the draft (store.DiscardSettingsDraft), leaving live untouched.
//
// This file asserts, in order:
//   (1) DRAFT ISOLATION — after an inline edit autosaves to the draft:
//         (a) reloading /admin/editor STILL shows the marker (editor is
//             draft-aware), and
//         (b) a fresh GET of the public storefront "/" does NOT yet contain the
//             marker (still the seed/live headline) — the draft is invisible live.
//   (2) PUBLISH — triggering the publish action (the REAL Publish button is
//         revealed + asserted present/visible, then the action is POSTed; see the
//         triggering note below) promotes draft→live; afterwards "/" NOW contains
//         the marker, and the editor's pending-draft indicator clears
//         (data-studio-has-draft="false", the draft-status line goes hidden, and
//         the Discard button disappears — it is only rendered while a draft exists).
//   (3) DISCARD — a SECOND inline edit autosaves a different marker to a new draft
//         (editor shows it after reload); triggering the discard action (the REAL
//         Discard button is likewise asserted present, then the action is POSTed)
//         drops the draft, so the editor reverts to the PUBLISHED first marker (not
//         the discarded second), and "/" is unchanged (still the first marker).
//
// Publish/Discard: the REAL publish-panel buttons are revealed, asserted present +
// visible, and triggered via a NATIVE button CLICK. Each button is a submit
// targeting the workbench form (form="websiteEditorForm"); that form is enhanced by
// the studio STATE runtime (data-gosx-studio-state + data-gosx-studio-client),
// which re-sends the submit as a multipart/form-data fetch carrying the hidden
// csrf_token input but no X-CSRF-Token header. The gosx CSRF guard now reads the
// csrf_token out of the multipart body (r.FormValue), so the native click validates
// and the action responds non-403 — see triggerPublishPanelAction, which asserts
// status != 403 on every publish/discard.
//
// Honesty discipline (matching the sibling canvas tests): the overlay element +
// its editable child are DISCOVERED from the live DOM the client renders; both
// markers are freshly generated per run (timestamp + random nonce) so neither the
// reload nor the public-storefront assertions can false-pass on stale/seed
// content. Markers use HTML-safe chars only (letters/digits/spaces/hyphens) so
// they survive the server's html escaping round-trip byte-for-byte in both the
// contenteditable textContent and the public storefront HTML.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy build rebuilt against the
// gosx/gosx-studio under test (`go run ./cmd/muddy-noni` via the shared harness).

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const OVERLAY_HERO = "[data-gosx-canvas-html='page:home']";
// M10: the full-page surface root carries data-gosx-html-key="page:home" but is a
// non-editable wrapper; the home.hero.headline-bound contenteditable <h1> is nested
// INSIDE that surface.
const EDITABLE_HERO = "[data-studio-field='home.hero.headline']";
const FIELD_HERO = "[data-studio-field='home.hero.headline']";
const AUTHORING_ROUTE = "/admin/editor/__actions/authoring";
const PUBLISH_ROUTE = "/admin/editor/__actions/publish";
const DISCARD_ROUTE = "/admin/editor/__actions/discardDraft";
const BINDING = "home.hero.headline";

const PUBLISH_PANEL = "[data-studio-publish-panel='true']";
const DRAFT_STATUS = "[data-studio-publish-draft-status='true']";
// The gsx renderer emits a true bool attribute as a BARE attribute (no ="true"
// value) and OMITS it entirely when false. So the editor's pending-draft state is
// signalled by the PRESENCE of data-studio-has-draft on the draft-status line:
//   • draft pending → [data-studio-has-draft] is present (and the Discard button,
//     gated on hasDraft, is rendered);
//   • no draft      → the attribute is absent (and no Discard button).
const DRAFT_STATUS_HAS_DRAFT = `${DRAFT_STATUS}[data-studio-has-draft]`;
const PUBLISH_BUTTON = "[data-studio-submit-action='publish']";
const DISCARD_BUTTON = "[data-studio-submit-action='discard']";

const SEED_HEADLINE = "Ceramic pieces with evidence of the hand";

test.describe("@reference-apps canvas2d site-map WASM-free draft → publish → discard loop", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni wasm-free: an inline hero edit stays draft-isolated from the public storefront, Publish promotes it to live + clears the draft, and Discard reverts a second edit to the published value", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    // The Publish/Discard buttons carry data-admin-confirm; cms-actions.js fires
    // window.confirm on any submit. We trigger the action via a scoped POST (see
    // triggerPublishPanelAction), so this dialog auto-accept is defensive — it
    // guarantees no confirm prompt (e.g. from any incidental submit) can stall the
    // test.
    page.on("dialog", (dialog) => { void dialog.accept(); });

    // Freshly generated per run so neither the reload nor the public-storefront
    // assertions can false-pass on stale/seed content. HTML-safe chars only
    // (survives html escaping in textContent + raw HTML byte-for-byte).
    const nonce = Math.random().toString(36).slice(2, 10);
    const marker1 = `M3 draft one ${Date.now()} ${nonce}`;
    const marker2 = `M3 draft two ${Date.now()} ${nonce}b`;

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await openEditorToHeroSurface(page, server.baseURL, consoleErrors);

      // The seed value the server rendered live — every later assertion proves the
      // markers REPLACED (or did not reach) this rather than matching pre-existing
      // content.
      const seedText = (await editableText(page)).trim();
      expect(marker1, "the per-run marker must differ from the seeded headline").not.toBe(seedText);
      expect(marker2, "the per-run second marker must differ from the seeded headline").not.toBe(seedText);

      // Sanity-check the live storefront baseline up front: "/" shows the seed
      // headline and neither marker yet.
      const storefrontSeed = await fetchStorefront(request, server.baseURL);
      expect(storefrontSeed, "the public storefront should start on the seed headline").toContain(SEED_HEADLINE);
      expect(storefrontSeed, "no marker should exist on the storefront before any edit").not.toContain(marker1);

      // ── (1) EDIT → DRAFT AUTOSAVE ─────────────────────────────────────────────
      await commitInlineEdit(page, marker1, consoleErrors);

      // ── (1a) DRAFT ISOLATION: editor is draft-aware across a reload ────────────
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await waitForHeroEditable(page);
      const draftReload = await pollForEditableText(page, marker1);
      expect(
        draftReload,
        `after reload the editor must re-serve the DRAFT marker (SettingsForEditor); seed=${JSON.stringify(seedText)}, got=${JSON.stringify(draftReload)}`,
      ).toContain(marker1);

      // The reloaded editor must show a pending-draft indicator + a Discard button
      // (only rendered while a draft exists) — server-rendered from HasSettingsDraft.
      await expect(page.locator(PUBLISH_PANEL).first(), "the publish panel must render").toBeAttached({ timeout: 30_000 });
      await expect(
        page.locator(DRAFT_STATUS_HAS_DRAFT).first(),
        "with a draft pending the editor must mark the draft-status line with data-studio-has-draft (HasSettingsDraft=true)",
      ).toBeAttached({ timeout: 30_000 });
      await expect(
        page.locator(DISCARD_BUTTON).first(),
        "the Discard button must be present while a draft is pending",
      ).toBeAttached({ timeout: 30_000 });

      // ── (1b) DRAFT ISOLATION: public storefront still LIVE (no marker) ─────────
      const storefrontDuringDraft = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontDuringDraft,
        `the public storefront "/" must NOT contain the draft marker before publish (draft is editor-only / live untouched); html-head=${storefrontDuringDraft.slice(0, 200)}`,
      ).not.toContain(marker1);
      expect(
        storefrontDuringDraft,
        "the public storefront must still render the live seed headline while the edit is only a draft",
      ).toContain(SEED_HEADLINE);

      // ── (2) PUBLISH → promote draft → live ────────────────────────────────────
      const publishStatus = await triggerPublishPanelAction(page, server.baseURL, PUBLISH_BUTTON, PUBLISH_ROUTE);
      expect(
        [200, 303].includes(publishStatus),
        `Publish action POST should be accepted/redirected; got status ${publishStatus}`,
      ).toBe(true);

      // ── (2a) PUBLISHED: storefront NOW carries the marker (promoted to live) ───
      const storefrontAfterPublish = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontAfterPublish,
        `after Publish the public storefront "/" must contain the promoted marker (draft→live); html-head=${storefrontAfterPublish.slice(0, 200)}`,
      ).toContain(marker1);
      expect(
        storefrontAfterPublish,
        "after Publish the seed headline must no longer be the live storefront headline",
      ).not.toContain(SEED_HEADLINE);

      // ── (2b) PUBLISHED: editor draft indicator cleared + a version recorded ────
      // After publish the draft is gone: HasSettingsDraft=false, so the data-studio-
      // has-draft attribute is omitted from the (still-present) draft-status line and
      // the Discard button is not rendered.
      await expect(page.locator(PUBLISH_PANEL).first(), "the publish panel must still render after publish").toBeAttached({ timeout: 30_000 });
      await expect(
        page.locator(DRAFT_STATUS_HAS_DRAFT),
        "after Publish the draft is cleared, so data-studio-has-draft must be absent (HasSettingsDraft=false)",
      ).toHaveCount(0);
      await expect(
        page.locator(DISCARD_BUTTON),
        "after Publish there is no pending draft, so the Discard button must be gone",
      ).toHaveCount(0);
      // The published value is the live editor value too (editor now reads live).
      const editorAfterPublish = await pollForEditableText(page, marker1);
      expect(
        editorAfterPublish,
        `after Publish the editor hero must show the published value (now live); got=${JSON.stringify(editorAfterPublish)}`,
      ).toContain(marker1);

      // ── (3) SECOND EDIT → NEW DRAFT ───────────────────────────────────────────
      await commitInlineEdit(page, marker2, consoleErrors);
      // The new draft must be editor-visible after reload, and must NOT have touched
      // the live storefront (still the first/published marker).
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await waitForHeroEditable(page);
      const secondDraftReload = await pollForEditableText(page, marker2);
      expect(
        secondDraftReload,
        `the second edit must autosave to a new draft the editor re-serves; got=${JSON.stringify(secondDraftReload)}`,
      ).toContain(marker2);
      await expect(
        page.locator(DISCARD_BUTTON).first(),
        "the Discard button must reappear once the second draft exists",
      ).toBeAttached({ timeout: 30_000 });
      const storefrontDuringSecondDraft = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontDuringSecondDraft,
        "the second draft must stay isolated: the storefront still shows the PUBLISHED first marker, not the new draft",
      ).toContain(marker1);
      expect(
        storefrontDuringSecondDraft,
        "the second (unpublished) marker must NOT appear on the public storefront",
      ).not.toContain(marker2);

      // ── (3) DISCARD → revert to published value ────────────────────────────────
      const discardStatus = await triggerPublishPanelAction(page, server.baseURL, DISCARD_BUTTON, DISCARD_ROUTE);
      expect(
        [200, 303].includes(discardStatus),
        `Discard action POST should be accepted/redirected; got status ${discardStatus}`,
      ).toBe(true);

      // The editor must REVERT to the published first marker, not the discarded
      // second, and the draft indicator must be gone again.
      const editorAfterDiscard = await pollForEditableText(page, marker1);
      expect(
        editorAfterDiscard,
        `after Discard the editor hero must revert to the PUBLISHED value (first marker); got=${JSON.stringify(editorAfterDiscard)}`,
      ).toContain(marker1);
      expect(
        editorAfterDiscard,
        "after Discard the editor must NOT show the discarded second marker",
      ).not.toContain(marker2);
      await expect(
        page.locator(DRAFT_STATUS_HAS_DRAFT),
        "after Discard there is no draft, so data-studio-has-draft must be absent",
      ).toHaveCount(0);
      await expect(
        page.locator(DISCARD_BUTTON),
        "after Discard there is no draft, so the Discard button must be gone",
      ).toHaveCount(0);

      // The public storefront is unchanged by a discard: still the published first
      // marker, never the discarded second.
      const storefrontAfterDiscard = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontAfterDiscard,
        "after Discard the public storefront must be unchanged: still the published first marker",
      ).toContain(marker1);
      expect(
        storefrontAfterDiscard,
        "after Discard the discarded second marker must never have reached the live storefront",
      ).not.toContain(marker2);

      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|inline-edit|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// openEditorToHeroSurface boots the editor route and waits for the full WASM-free
// canvas + inline-edit bridge + the contenteditable hero overlay to mount. Mirrors
// the M2 inline-edit setup so the draft/publish flow starts from a known-live
// surface.
async function openEditorToHeroSurface(page: Page, baseURL: string, consoleErrors: string[]) {
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
  await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (selection sink)").toBeAttached();

  const canvas = page.locator(CANVAS_SELECTOR).first();
  await expect(canvas, "the muddy WASM-free canvas must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

  // Stale-dist guard: the painter, the DOM board runtime, AND the M2 inline-edit
  // commit bridge must all be installed. A missing inline-edit module means the
  // served bundle predates the M2/M3 work — fail loud rather than fake.
  await page.waitForFunction(() => {
    const w = window as unknown as Record<string, unknown>;
    const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
    const painter = w.GoSXStudioCanvas2DPainterRuntime as { paint?: unknown; renderCanvasBoardHTML?: unknown } | undefined;
    const inlineEdit = w.GoSXStudioCanvasInlineEditRuntime as { install?: unknown; persist?: unknown } | undefined;
    const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: unknown }) | null;
    return !!painter && typeof painter.paint === "function" && typeof painter.renderCanvasBoardHTML === "function" &&
      !!rt && typeof rt.setState === "function" &&
      !!inlineEdit && typeof inlineEdit.install === "function" && typeof inlineEdit.persist === "function" &&
      document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
      !!el && !!el.GoSXStudioCanvasWasmFree;
  }, null, { timeout: 120_000 });

  await waitForHeroEditable(page);
  await expect(
    page.locator(`${EDITABLE_HERO}${FIELD_HERO}`).first(),
    "the editable hero must carry the store binding data-studio-field='home.hero.headline'",
  ).toBeAttached();
  expect(await page.locator(EDITABLE_HERO).first().getAttribute("contenteditable"), "the hero element must be contenteditable").toBe("true");
  void consoleErrors;
}

// waitForHeroEditable waits for the injected hero surface to (re)mount into the
// camera-positioned overlay and become visible.
async function waitForHeroEditable(page: Page) {
  const overlayHero = page.locator(OVERLAY_HERO).first();
  await expect(
    overlayHero,
    "the painter must mount the html surface as an overlay element [data-gosx-canvas-html='page:home'] above the canvas",
  ).toBeAttached({ timeout: 90_000 });
  const editable = page.locator(EDITABLE_HERO).first();
  await expect(editable, "the host markup's contenteditable hero must render inside the overlay").toBeAttached({ timeout: 30_000 });
  await expect(editable, "the editable hero element must be visible (overlay sized + positioned)").toBeVisible({ timeout: 30_000 });
}

// commitInlineEdit selects the hero surface, replaces its text with the marker,
// blurs to commit, and waits for the autosave POST (draft save) to resolve.
// Mirrors the M2 inline-edit commit, asserting the bound value reached the
// authoring action.
async function commitInlineEdit(page: Page, marker: string, consoleErrors: string[]) {
  const editable = page.locator(EDITABLE_HERO).first();
  await editable.click();
  await pollForSelectedNode(page, "page:home");

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

  // Commit by blurring (focusout → commit() → persist() POST).
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

// triggerPublishPanelAction fires a real publish-panel action (publish /
// discardDraft) by NATIVELY CLICKING the real editor button, returns the action's
// HTTP status, then reloads the editor so its server-rendered draft state reflects
// the result.
//
// Triggering: a real headless-Chromium CLICK of the actual <button type="submit">
// publish-panel control. Each button targets the editor workbench form via
// form="websiteEditorForm" (muddy app/admin/editor/page.gsx PublishPanel) + a
// formaction of the action. That form (gosx-cms studio/workbench.go
// workbenchFormAttrs) carries data-gosx-studio-state + data-gosx-studio-client, so
// the studio STATE runtime (gosx-cms studio/assets/state_runtime.js initForm)
// intercepts the submit and re-sends it via runClientAction → fetch(action, {
// body: new FormData(form, submitter) }) — a multipart/form-data body that carries
// the hidden csrf_token input but NO X-CSRF-Token header.
//
// This used to 403: the gosx session guard read csrf_token from the X-CSRF-Token
// header, else (for non-JSON requests) from r.Form via r.ParseForm() — and
// r.ParseForm() does NOT parse a multipart body, so the csrf_token sitting in the
// multipart body was never read. The guard now uses r.FormValue(csrf_token), which
// parses multipart bodies (ParseMultipartForm) as well as urlencoded ones, so the
// native click's multipart submit validates. We assert below the action response
// is NOT 403 — the whole point of this upgrade.
//
// cms-actions.js fires window.confirm on submit of buttons carrying
// data-admin-confirm; the test-level page.on("dialog", d => d.accept()) handles it.
async function triggerPublishPanelAction(page: Page, baseURL: string, buttonSelector: string, actionRoute: string): Promise<number> {
  void baseURL;
  // Faithful UI step: surface the publish panel + assert the real button exists.
  await revealModeIfPresent(page, "publish");
  const button = page.locator(buttonSelector).first();
  await expect(button, `the ${buttonSelector} button must be present + visible to trigger the action`).toBeVisible({ timeout: 30_000 });

  // Native click submits websiteEditorForm; the studio state runtime re-sends it
  // as a multipart/form-data fetch whose csrf_token the gosx guard now reads.
  const responsePromise = page.waitForResponse(
    (response) => response.url().includes(actionRoute) && response.request().method() === "POST",
    { timeout: 30_000 },
  );
  await button.click();
  const response = await responsePromise;
  const status = response.status();
  expect(
    status,
    `the native ${buttonSelector} click must NOT 403 (gosx now reads the multipart csrf_token); got ${status}`,
  ).not.toBe(403);

  // Reload so the editor re-renders from the post-action store state (draft-aware).
  await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
  await waitForHeroEditable(page);
  return status;
}

// fetchStorefront GETs the PUBLIC storefront "/" from the SAME running server via
// a server-side request (no client/editor DOM involved) and returns its HTML. The
// storefront renders store.Settings() (LIVE), so this is the ground truth for
// draft isolation.
async function fetchStorefront(request: { get: (url: string, opts?: { failOnStatusCode?: boolean; timeout?: number }) => Promise<{ text(): Promise<string> }> }, baseURL: string): Promise<string> {
  const response = await request.get(`${baseURL}/`, { failOnStatusCode: false, timeout: 60_000 });
  return response.text();
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
