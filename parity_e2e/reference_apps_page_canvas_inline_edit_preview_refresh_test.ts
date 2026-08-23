import { expect, test, type Page } from "@playwright/test";
import { startMuddy } from "./reference_apps_harness";

// Live-proof e2e for a production defect on the muddy-noni website editor:
// inline canvas text editing silently lost typing.
//
// REPRO (live Playwright walkthrough, ~150ms/keystroke, MUDDY_EDITOR_SCENE_DOM=1):
// double-click the hero headline on the PageCanvas (the real same-origin
// [data-studio-preview-frame] iframe — see shell/workbench_page_canvas.go)
// starts a contenteditable inline-edit session (inlineeditruntime/
// island_runtime.js's startPreviewTextEdit). While typing, the preview
// iframe RELOADED mid-edit — each reload destroys the contenteditable
// session (a fresh document, a fresh window), discarding whatever had been
// typed. Escape still cancelled cleanly (finishPreviewTextEdit's cancel path
// never touches the frame), and editing the SAME field via the right-panel
// input still worked (a plain, same-document <textarea>, never subject to an
// iframe navigation) — both of which is why this defect read as "canvas
// editing" specifically being broken.
//
// ROOT CAUSE (confirmed empirically below, not just by inspection — this
// file was proven red against the pre-fix previewruntime/island_runtime.js
// and green after): on EVERY keystroke, inlineeditruntime's
// syncPreviewTextEdit fires a gosxstudio:editor-operation "set_text" event.
// hostruntime/assets/state_runtime.js's listener always calls updateFrame()
// on that event, which recomputes and dispatches gosxstudio:save-state with
// state:"saved" whenever the tracked workbench form's own signature isn't
// currently dirty (state_runtime.js's update()) — true on essentially every
// keystroke here, since the PageCanvas default surface's inline-edit session
// has no bound, VISIBLE shadow form control to mirror into (the legacy
// [data-gosx-studio-editable-control] textarea that WOULD receive it lives
// inside the Advanced-mode-only site-map board panel, which stays unbound to
// whatever's newly selected while that panel is hidden — a separate,
// pre-existing gap, not fixed here; see the caveat below).
// hostruntime/assets/workbench_runtime.js's listener on gosxstudio:save-state
// turns EVERY "saved" dispatch into schedulePreviewRefresh(...) unconditionally
// (`else if (detail.state === "saved") schedulePreviewRefresh(...)`), and
// previewruntime/island_runtime.js's scheduleEditorPreviewRefresh only
// debounces that for 180ms before calling refreshEditorPreviewNow, which
// reassigns frame.setAttribute("src", cacheBustPreviewURL(...)) — a full
// navigation, with zero awareness of inlineeditruntime's own
// frame.__gosxStudioInlineEdit session flag. Net effect: ANY inter-keystroke
// gap over ~180ms (trivially common at a natural ~150ms/keystroke cadence —
// word boundaries, capitalization, punctuation) re-arms and then fires that
// reload mid-session, discarding the in-progress edit. (A slower, secondary
// contributor funnels through the exact same choke point: when a field DOES
// have a live mirrored control, state_runtime.js's own autosave debounce —
// default 1400ms, reset every keystroke via the same "set_text" event —
// eventually completes and dispatches its own "saved"/"autosave", which hits
// the identical unconditional schedulePreviewRefresh call.)
//
// THE FIX (previewruntime/island_runtime.js only): refreshEditorPreviewNow
// now checks frame.__gosxStudioInlineEdit per frame before navigating it; if
// a session is active it queues the {reason, route} instead, and a short
// poll (keyed off the SAME flag inlineeditruntime already clears on
// commit/cancel) applies the queued refresh once the session ends — the
// preview document is never reloaded/replaced while an inline edit session
// is active, regardless of which upstream signal (per-keystroke status
// churn or a real autosave completion) requested the refresh.
//
// WHY THE EXISTING e2e MISSED THIS: reference_apps_canvas_inline_edit_test.ts
// drives a COMPLETELY different editable surface — the WASM-free Canvas2D
// site-map board's contenteditable overlay (canvasinlineeditruntime +
// inlineeditruntime's install()/commit() engine), which persists directly on
// blur via a single fetch() POST with no gosxstudio:editor-operation /
// autosave / schedulePreviewRefresh wiring in its path AT ALL, and types an
// 8ms/char, well-under-a-second marker — nowhere near a 180ms gap. This file
// drives the PageCanvas [data-studio-preview-frame] iframe instead — the
// surface the live repro actually broke on.
//
// CAVEAT (out of scope for this fix, flagged for follow-up): typing on the
// PageCanvas default surface's inline-edit session does not appear to reach
// the draft-persistence pipeline at all in this configuration (confirmed
// below: the field's resolved shadow value control never mirrors the typed
// text, because it belongs to the Advanced-mode-only legacy site-map board's
// editable-control panel, which stays unbound while that panel is hidden by
// default). This file therefore proves the reload-mid-session defect and its
// fix precisely — it does not additionally assert server-side persistence of
// the typed value, which is a separate, pre-existing gap.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const WORKBENCH_SELECTOR = "[data-studio-workbench='true']";
const PAGE_CANVAS_SELECTOR = "[data-gosx-studio-page-canvas='true']";
const PREVIEW_FRAME_SELECTOR = "[data-studio-preview-frame]";
const HERO_FIELD_SELECTOR = "[data-studio-field='hero.headline']";

test.describe("@reference-apps PageCanvas inline text edit survives an in-session preview refresh (silent-typing-loss fix)", () => {
  test.describe.configure({ timeout: 120_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni editor: typing a full headline on the PageCanvas preview iframe survives a mid-session pause — no reload while editing, refresh only after commit", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    // Two distinct, freshly generated halves so "did the WHOLE sentence
    // land" cannot false-pass on only the part typed before the mid-session
    // pause.
    const firstHalf = `Slow-roasted single-origin ${Date.now()}`;
    const secondHalf = ` arriving fresh weekly ${Math.random().toString(36).slice(2, 10)}`;
    const fullMarker = firstHalf + secondHalf;

    const server = await startMuddy(request, { MUDDY_EDITOR_SCENE_DOM: "1" });
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator(WORKBENCH_SELECTOR).first()).toBeAttached();

      // PageCanvas is the default Home surface under this env (no mode
      // reveal needed — mirrors reference_apps_canvas_first_test.ts).
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
      expect(fullMarker, "the per-run marker must differ from the seeded headline").not.toBe(seedText);

      // Arm an audit trail of every gosxstudio:preview-refresh event BEFORE
      // touching the field, so nothing can slip past unobserved.
      await page.evaluate(() => {
        const w = window as unknown as { __reproRefresh: unknown[] };
        w.__reproRefresh = [];
        document.addEventListener("gosxstudio:preview-refresh", (event) => {
          const detail = (event as CustomEvent).detail ?? {};
          w.__reproRefresh.push({ reason: detail.reason ?? "", route: detail.route ?? "", t: Date.now() });
        });
      });

      // ── START THE SESSION: double-click the hero headline inside the iframe ──
      const frameLocator = page.frameLocator(PREVIEW_FRAME_SELECTOR);
      const hero = frameLocator.locator(HERO_FIELD_SELECTOR).first();
      await expect(hero, "the live storefront's hero headline must render inside the preview iframe").toBeVisible({ timeout: 30_000 });
      await hero.dblclick();
      await expect(hero, "double-click must start an inline-edit session on the hero field").toHaveAttribute("data-gosx-studio-inline-editing", "true", { timeout: 10_000 });

      const srcAtSessionStart = await previewFrame.getAttribute("src");

      // ── TYPE THE FIRST HALF at a realistic ~150ms/char cadence ──────────────
      await page.keyboard.press("ControlOrMeta+A");
      await page.keyboard.type(firstHalf, { delay: 150 });
      const afterFirstHalf = await pollForHeroText(page, firstHalf);
      expect(afterFirstHalf, `first half must land before the mid-session pause; got=${JSON.stringify(afterFirstHalf)}`).toContain(firstHalf);

      // ── MID-SESSION PAUSE ─────────────────────────────────────────────────
      // A realistic human beat mid-headline (composing the next word) — well
      // over the 180ms preview-refresh debounce
      // (previewruntime/island_runtime.js's scheduleEditorPreviewRefresh),
      // which is all it takes to fire a would-be reload on every keystroke
      // gap this size. This is the live repro's actual trigger, reproduced
      // deterministically rather than hoping natural typing jitter happens
      // to exceed 180ms during the run.
      await page.waitForTimeout(900);

      // ── THE CORE REGRESSION ASSERTIONS ───────────────────────────────────────
      // (1) the preview iframe must NOT have navigated mid-session — this is
      //     exactly the destructive step that discarded keystrokes live.
      const srcDuringSession = await previewFrame.getAttribute("src");
      expect(srcDuringSession, "the preview iframe must not reload while an inline-edit session is active, even after the preview-refresh debounce elapses").toBe(srcAtSessionStart);
      // (2) no preview-refresh may have applied while the session stays open.
      const refreshEventsDuringSession = await page.evaluate(() => (window as unknown as { __reproRefresh: unknown[] }).__reproRefresh);
      expect(refreshEventsDuringSession, `no preview refresh may apply while the inline-edit session is still open; saw=${JSON.stringify(refreshEventsDuringSession)}`).toEqual([]);
      // (3) the session itself must have survived — still contenteditable,
      //     still carrying everything typed so far.
      await expect(hero, "the inline-edit session must survive the debounce that fired mid-edit").toHaveAttribute("data-gosx-studio-inline-editing", "true");
      const textStillThere = await editableHeroText(page);
      expect(textStillThere, `the first half must still be present after the mid-session refresh was deferred; got=${JSON.stringify(textStillThere)}`).toContain(firstHalf);

      // ── TYPE THE SECOND HALF, continuing the SAME session ───────────────────
      await page.keyboard.type(secondHalf, { delay: 150 });
      const afterSecondHalf = await pollForHeroText(page, fullMarker);
      expect(afterSecondHalf, `the FULL sentence (both halves) must land in the live DOM; got=${JSON.stringify(afterSecondHalf)}, console=${JSON.stringify(consoleErrors.slice(-6))}`).toContain(fullMarker);

      // Still nothing applied — proves the deferral holds for the WHOLE
      // session, not just its first pause.
      const refreshEventsBeforeCommit = await page.evaluate(() => (window as unknown as { __reproRefresh: unknown[] }).__reproRefresh);
      expect(refreshEventsBeforeCommit, `no preview refresh may apply before commit; saw=${JSON.stringify(refreshEventsBeforeCommit)}`).toEqual([]);

      // ── COMMIT (blur) — this ends the session ────────────────────────────────
      await frameLocator.locator("body").click({ position: { x: 4, y: 4 } });
      await expect(hero, "blurring the field must end the inline-edit session").not.toHaveAttribute("data-gosx-studio-inline-editing", "true", { timeout: 10_000 });

      // ── THE DEFERRED REFRESH MUST NOW APPLY ─────────────────────────────────
      // Once the session has ended, the SAME refresh request that was queued
      // (or the next one commit's own final sync schedules) is now free to
      // apply — proving "queue then apply after session end", not "queue
      // forever".
      await expect
        .poll(async () => previewFrame.getAttribute("src"), {
          message: "the preview iframe must eventually refresh once the session has ended (queued-then-applied contract)",
          timeout: 10_000,
        })
        .not.toBe(srcAtSessionStart);
      const refreshEventsAfterCommit = await page.evaluate(() => (window as unknown as { __reproRefresh: unknown[] }).__reproRefresh);
      expect(refreshEventsAfterCommit.length, `at least one preview refresh must have fired once the session ended; saw=${JSON.stringify(refreshEventsAfterCommit)}`).toBeGreaterThan(0);

      const clientErrors = consoleErrors.filter((line) => /preview|inline|canvas|hydrate/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected preview/inline-edit console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// editableHeroText reads the hero headline's live text straight off the
// preview iframe's own contentDocument (same-origin) — nothing hardcoded.
async function editableHeroText(page: Page): Promise<string> {
  return page.evaluate((sel) => {
    const frame = document.querySelector("[data-studio-preview-frame]");
    if (!(frame instanceof HTMLIFrameElement)) return "";
    const el = frame.contentDocument?.querySelector(sel);
    return el ? (el.textContent || "") : "";
  }, HERO_FIELD_SELECTOR);
}

async function pollForHeroText(page: Page, want: string): Promise<string> {
  const deadline = Date.now() + 15_000;
  let last = "";
  while (Date.now() < deadline) {
    last = await editableHeroText(page);
    if (last.includes(want)) return last;
    await page.waitForTimeout(120);
  }
  return last;
}

// hydrateDeferredPreviewFrame sets the iframe's src from its
// data-studio-preview-src/host data-gosx-studio-preview-url fallback if the
// server rendered it deferred (no src yet) — mirrors
// reference_apps_canvas_first_test.ts's identical helper.
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
