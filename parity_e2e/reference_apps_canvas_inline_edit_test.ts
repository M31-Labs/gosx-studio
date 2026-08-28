import { expect, test, type Page, type Request } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface, startMuddyCanvasHTMLSurfaceCollaboration } from "./reference_apps_harness";

// Live-proof e2e for the M2 canvas inline-edit LOOP — edit a contenteditable HTML
// surface field on the WASM-free Canvas2D board and prove it PERSISTS across a
// full page reload.
//
// Builds directly on the M0 fixture (MUDDY_CANVAS_WASM_FREE=1 +
// MUDDY_EDITOR_SCENE_DOM=1, see reference_apps_canvas_html_surface_test.ts): the
// host injects ONE Kind:"html" "page:home" full-page surface whose nested <h1>
// shows the REAL current store hero headline and carries
// data-studio-field="home.hero.headline" + contenteditable. M2 adds the muddy-side commit bridge (canvas_inline_edit.js,
// window.GoSXStudioCanvasInlineEditRuntime): on blur / Enter it POSTs an edit-commit to
// /admin/editor/__actions/authoring, which the muddy authoring adapter routes
// (saveHeroHeadline) into store.SaveSettings + a settings checkpoint revision.
// Because the SAME running server reads store.Settings().Hero.Headline when it
// re-renders the hero surface markup (editorSiteMapCanvasExtraNodes), a reload
// re-serves the persisted value — that round-trip is the core assertion here.
//
// This file asserts, end-to-end in headless Chromium:
//   (1) committing an inline edit fires a POST to the authoring action whose
//       x-www-form-urlencoded body carries gosx_studio_binding=home.hero.headline
//       and the new (unique) value;
//   (2) after the POST resolves AND the page is reloaded, the re-served hero
//       surface shows the UNIQUE marker — i.e. the edit was persisted to the cms
//       store and re-rendered from it, not just left in the live DOM.
//
// Wave 5 (handoff-31, "one product path"): canvas/page_surface.go's
// RenderHomePageSurfaceMarkup now renders durable-history opt-in attributes
// on this SAME hero <h1> (inlineeditruntime/island_runtime.js's
// data-gosx-studio-durable-history + data-gosx-studio-durable-field
// ="hero.headline" — the exact durable Field literal
// editorDirectEditPanel's own studioDirectEditForm targets), so a commit
// with a live, capability-granting collaboration transport now dispatches
// through the durable operation ledger instead of the legacy save-control
// POST above — the SAME "ONE path when durable, honest fallback when not"
// discipline every other durable-history form already uses. The third test
// below proves that path end-to-end (durable commit event, absence of the
// legacy POST, Release Center change-set visibility, AND — since
// handoff-34's actor-identity-namespace unification — a live, browser-driven
// Undo of that same collab-made edit through studioDirectEditForm), and the
// fourth proves the stale-write conflict path across two sessions. The
// first two tests above are UNCHANGED and keep proving the legacy-fallback
// path (startMuddyCanvasHTMLSurface has no live collaboration transport, so
// collaboration.available() stays false and commit() falls through to the
// same save-control POST it always did).
//
// Honesty discipline (matching the sibling canvas tests): the overlay element +
// its editable child are DISCOVERED from the live DOM the client renders; the
// marker is freshly generated per run (timestamp + random nonce) so the reload
// assertion cannot false-pass on stale/seed content. The marker is kept to
// HTML-safe characters (letters/digits/spaces/hyphens) so it survives the server's
// html.EscapeString round-trip byte-for-byte in textContent.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const OVERLAY_HERO = "[data-gosx-canvas-html='page:home']";
// M10: the full-page surface root carries data-gosx-html-key="page:home" but is a
// non-editable wrapper; the store-bound, contenteditable hero heading is the nested
// <h1 data-studio-field="home.hero.headline"> INSIDE that surface.
const EDITABLE_HERO = "[data-studio-field='home.hero.headline']";
const FIELD_HERO = "[data-studio-field='home.hero.headline']";
const AUTHORING_ROUTE = "/admin/editor/__actions/authoring";
const BINDING = "home.hero.headline";

test.describe("@reference-apps canvas2d site-map WASM-free inline-edit loop", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni wasm-free: committing an inline edit on the hero surface POSTs the bound value and PERSISTS across a full reload", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    // Freshly generated per run so the reload assertion can't false-pass on stale
    // or seed content. HTML-safe chars only (survives html.EscapeString in
    // textContent byte-for-byte).
    const marker = `M2 inline edit ${Date.now()} ${Math.random().toString(36).slice(2, 10)}`;

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
      // The legacy site-map/canvas board now lives inside the "Advanced" mode
      // panel (studio-pagecanvas-handoff moved it there once a PageCanvas
      // surface is present); reveal it before touching the board below.
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (selection sink)").toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "the muddy WASM-free canvas must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

      // The WASM-free client + painter must mount, the DOM board runtime (selection
      // sink) must be present, AND the M2 inline-edit commit bridge must be installed
      // (window.GoSXStudioCanvasInlineEditRuntime). A missing inline-edit module means the
      // served bundle predates the M2 work (stale dist): fail loud rather than fake.
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

      // The injected html surface must mount into the camera-positioned overlay and
      // expose the contenteditable, store-bound hero field.
      const overlayHero = page.locator(OVERLAY_HERO).first();
      await expect(
        overlayHero,
        "the painter must mount the html surface as an overlay element [data-gosx-canvas-html='page:home'] above the canvas",
      ).toBeAttached({ timeout: 60_000 });

      const editable = page.locator(EDITABLE_HERO).first();
      await expect(editable, "the host markup's contenteditable hero must render inside the overlay").toBeAttached({ timeout: 30_000 });
      await expect(editable, "the editable hero element must be visible (overlay sized + positioned)").toBeVisible({ timeout: 30_000 });
      await expect(
        page.locator(`${EDITABLE_HERO}${FIELD_HERO}`).first(),
        "the editable hero must carry the store binding data-studio-field='home.hero.headline'",
      ).toBeAttached();
      expect(await editable.getAttribute("contenteditable"), "the hero element must be contenteditable").toBe("true");

      // The seed value the server rendered — the reload assertion proves the marker
      // REPLACED this rather than matching pre-existing content.
      const seedText = (await editableText(page)).trim();
      expect(marker, "the per-run marker must differ from the seeded headline").not.toBe(seedText);

      // ── (1) SELECT then EDIT ──────────────────────────────────────────────────
      // Click selects the surface (setState → board selection attr), then we focus,
      // select-all, and type the unique marker.
      await editable.click();
      await pollForSelectedNode(page, "page:home");

      // Arm network capture for the commit POST BEFORE we trigger the blur, so the
      // request can't slip past us.
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

      // Commit by blurring (focusout → commit() → persist() POST). Move focus to a
      // neutral element so the contenteditable loses focus.
      await page.evaluate((sel) => {
        const el = document.querySelector(sel) as HTMLElement | null;
        if (el && typeof el.blur === "function") el.blur();
        const board = document.querySelector("[data-studio-site-map-board='true']") as HTMLElement | null;
        if (board && typeof board.focus === "function") {
          try { board.focus(); } catch { /* tolerate */ }
        }
      }, EDITABLE_HERO);

      // ── (1) ASSERT THE COMMIT POST ────────────────────────────────────────────
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
        `commit POST should be accepted by the authoring action; got status ${postResponse.status()}`,
      ).toBe(true);

      // ── (2) RELOAD AND ASSERT PERSISTENCE ─────────────────────────────────────
      // The SAME running server re-renders the hero surface from
      // store.Settings().Hero.Headline. A reload must therefore re-serve the marker
      // — proving the edit reached the cms store, not just the live DOM.
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "advanced");

      const reloadedEditable = page.locator(EDITABLE_HERO).first();
      await expect(
        reloadedEditable,
        "the hero surface must re-mount in the overlay after reload",
      ).toBeAttached({ timeout: 90_000 });
      await expect(reloadedEditable, "the re-mounted hero surface must be visible").toBeVisible({ timeout: 30_000 });

      const persisted = await pollForEditableText(page, marker);
      expect(
        persisted,
        `after reload, the re-served hero surface must show the persisted marker (store round-trip); seed=${JSON.stringify(seedText)}, got=${JSON.stringify(persisted)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toContain(marker);

      // Cross-check via the public storefront-facing render path too: the saved
      // headline must be the live store value, independent of the canvas overlay.
      const editorHTML = await request.get(`${server.baseURL}/admin/editor`, { failOnStatusCode: false, timeout: 60_000 }).then((r) => r.text());
      expect(
        editorHTML,
        "a fresh server render of the editor must embed the persisted marker (store-backed, not client-only)",
      ).toContain(marker);

      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|inline-edit|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });

  // #5 (UX wave 2A) — Escape must cancel a canvas inline edit instead of
  // silently persisting it on the next blur. Before the fix,
  // inlineeditruntime/island_runtime.js's install() had no Escape handling at
  // all, so nothing reverted the DOM and a later blur committed the typed
  // text via a real POST — a silent-commit trust wound. This proves: (1) no
  // authoring POST fires for the duration of the Escape press, (2) the
  // surface text reverts to the pre-edit value, and (3) a reload still shows
  // the ORIGINAL seeded headline, proving nothing reached the store.
  test("Muddy/Noni wasm-free: Escape cancels an inline edit — reverts the value and never POSTs", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const marker = `M2 escape cancel ${Date.now()} ${Math.random().toString(36).slice(2, 10)}`;
    let authoringPostSeen = false;

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "advanced");

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas).toBeAttached({ timeout: 30_000 });
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        const inlineEdit = w.GoSXStudioCanvasInlineEditRuntime as { install?: unknown } | undefined;
        return !!inlineEdit && typeof inlineEdit.install === "function";
      }, null, { timeout: 120_000 });

      const overlayHero = page.locator(OVERLAY_HERO).first();
      await expect(overlayHero).toBeAttached({ timeout: 60_000 });
      const editable = page.locator(EDITABLE_HERO).first();
      await expect(editable).toBeAttached({ timeout: 30_000 });
      await expect(editable).toBeVisible({ timeout: 30_000 });

      const seedText = (await editableText(page)).trim();
      expect(marker, "the per-run marker must differ from the seeded headline").not.toBe(seedText);

      // Any authoring POST during this test would prove the silent-commit
      // regression; watch for the entire window the edit is in progress.
      page.on("request", (req) => {
        if (req.url().includes(AUTHORING_ROUTE) && req.method() === "POST") authoringPostSeen = true;
      });

      await editable.click();
      await pollForSelectedNode(page, "page:home");
      await editable.click();
      await page.waitForFunction((sel) => {
        const el = document.querySelector(sel);
        return !!el && document.activeElement === el;
      }, EDITABLE_HERO, { timeout: 10_000 });
      await page.keyboard.press("ControlOrMeta+A");
      await page.keyboard.type(marker, { delay: 8 });

      const typed = await pollForEditableText(page, marker);
      expect(typed, `typed marker must land before Escape; got=${JSON.stringify(typed)}`).toContain(marker);

      await page.keyboard.press("Escape");
      // Give any (incorrect) trailing commit a full beat to fire before
      // asserting its absence — this must stay a negative result.
      await page.waitForTimeout(500);

      const reverted = await editableText(page);
      expect(
        reverted.trim(),
        `Escape must revert the surface text to its pre-edit value; seed=${JSON.stringify(seedText)}, got=${JSON.stringify(reverted)}`,
      ).toBe(seedText);

      expect(authoringPostSeen, "Escape must never POST the cancelled value to the authoring route").toBe(false);

      // Reload and confirm the store itself was never touched — the
      // strongest possible proof no silent commit reached the server.
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "advanced");
      const reloadedEditable = page.locator(EDITABLE_HERO).first();
      await expect(reloadedEditable).toBeAttached({ timeout: 90_000 });
      const afterReload = await pollForEditableText(page, seedText);
      expect(
        afterReload,
        `reload must still show the original seeded headline, not the cancelled marker; got=${JSON.stringify(afterReload)}`,
      ).toContain(seedText);
      expect(afterReload).not.toContain(marker);

      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|inline-edit|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });

  // handoff-31 ("one product path"): the durable-dispatch proof. Requires a
  // LIVE, authenticated collaboration transport (startMuddyCanvasHTMLSurfaceCollaboration)
  // — under the plain startMuddyCanvasHTMLSurface harness the two tests
  // above use, collaboration.available() never becomes true (no
  // MUDDY_COLLAB_TEST_AUTH-signed session), so commit() correctly falls
  // back to the legacy save-control POST instead — that fallback discipline
  // is exactly what the two tests above already prove.
  test("Muddy/Noni wasm-free: committing an inline edit with a live collaboration transport dispatches the durable operation (never the legacy POST) and appears in the Release Center change-set", async ({ page, request }) => {
    const marker = `M2 durable inline edit ${Date.now()} ${Math.random().toString(36).slice(2, 10)}`;
    const CHANGESET_SECTION = "[data-studio-publish-changeset='true']";
    const CHANGESET_ROW_LABEL = `${CHANGESET_SECTION} code.revision-diff-row__label`;
    const COLLAB_PANEL = "[data-gosx-studio-collaboration='true']";

    const server = await startMuddyCanvasHTMLSurfaceCollaboration(request);
    try {
      await page.goto(`${server.baseURL}/__test/collaboration/signin/author-a`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const collabPanel = page.locator(COLLAB_PANEL).first();
      await expect(
        collabPanel,
        "the collaboration runtime must connect before the durable commit path is available",
      ).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });

      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(collabPanel).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(BOARD_SELECTOR).first()).toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas).toBeAttached({ timeout: 30_000 });
      const overlayHero = page.locator(OVERLAY_HERO).first();
      await expect(overlayHero).toBeAttached({ timeout: 60_000 });
      const editable = page.locator(EDITABLE_HERO).first();
      await expect(editable).toBeAttached({ timeout: 30_000 });
      await expect(editable).toBeVisible({ timeout: 30_000 });
      expect(await editable.getAttribute("data-gosx-studio-durable-history")).toBe("true");
      // Captured BEFORE the marker overwrite below so the undo-reverts
      // assertion (handoff-34) has a real baseline to compare against,
      // rather than a hardcoded seed string.
      const preEditHeadline = (await editable.textContent())?.trim() ?? "";

      // Arm the durable-commit event listener BEFORE the blur that fires it,
      // and watch for the LEGACY save-control POST too — this is the
      // absence assertion: a durable-enabled commit must never also fire it.
      let legacyPostSeen = false;
      page.on("request", (req) => {
        if (req.url().includes(AUTHORING_ROUTE) && req.method() === "POST") {
          const body = req.postData() ?? "";
          if (new URLSearchParams(body).get("gosx_studio_operation") === "save-control") legacyPostSeen = true;
        }
      });
      const committed = page.evaluate(
        () => new Promise<{ kind: "committed" | "error"; detail: unknown }>((resolve) => {
          document.addEventListener("gosxstudio:collaboration-operation-accepted", (event) => resolve({ kind: "committed", detail: (event as CustomEvent).detail }), { once: true });
          document.addEventListener("gosxstudio:collaboration-operation-rejected", (event) => resolve({ kind: "error", detail: (event as CustomEvent).detail }), { once: true });
        }),
      );

      await editable.click();
      await page.waitForFunction((sel) => {
        const el = document.querySelector(sel);
        return !!el && document.activeElement === el;
      }, EDITABLE_HERO, { timeout: 10_000 });
      await page.keyboard.press("ControlOrMeta+A");
      await page.keyboard.type(marker, { delay: 8 });
      await pollForEditableText(page, marker);

      await page.evaluate((sel) => {
        const el = document.querySelector(sel) as HTMLElement | null;
        if (el && typeof el.blur === "function") el.blur();
      }, EDITABLE_HERO);

      const outcome = await committed;
      expect(outcome.kind, `the durable set-field commit must succeed; got ${JSON.stringify(outcome)}`).toBe("committed");
      expect(legacyPostSeen, "a durable-enabled commit must never also fire the legacy save-control POST (absence assertion)").toBe(false);

      // Consequence (a): the Release Center change-set now carries this
      // commit — the SAME "hero.headline" durable target
      // reference_apps_publish_changeset_test.ts already proves for the
      // direct-edit-panel form, now reached from the canvas surface instead.
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "publish");
      await expect(page.locator(CHANGESET_SECTION).first(), "the change-set section must render").toBeVisible({ timeout: 30_000 });
      await expect(
        page.locator(CHANGESET_ROW_LABEL, { hasText: "hero.headline" }).first(),
        "the canvas-committed hero.headline edit must appear as a Content change-set row",
      ).toBeAttached();

      // Consequence (b), durable undo (handoff-34): studioDirectEditForm's
      // Undo button targets the SAME (route "/", page:home, hero.headline,
      // home:hero) target the canvas surface just committed to. Before
      // handoff-34 this button rendered enabled with a BLANK
      // historyOperationId here — muddy's editorHistoryIDs filtered
      // store.StudioAuthoringSnapshot().Operations by
      // editorActorIDFromRequest's plain HTTP-session actor id
      // (auth.User.ID, e.g. "test:author-a"), but the record the LIVE
      // COLLABORATION transport just accepted above carries
      // internal/studiohost.CollaborationPrincipal's hashed, namespaced
      // ActorID ("noni:<sha256(...)>") for the SAME real user — two
      // different identity strings for one person, so the button could never
      // find its own history. handoff-34 unified both paths onto
      // studiohost.ActorID's single derivation (see
      // internal/studiohost/collaboration.go and app/admin/editor's
      // authoringAction/editorHistoryIDs/editorActorIdentitiesFromRequest in
      // muddy-noni-commerce) — this is the live, browser-driven proof that
      // the fix actually closes the gap for a real signed-in user's own
      // collab-made edit, not just at the Go/ledger layer (see
      // TestActorScopedUndoMatchesRecordsAcrossCollaborationAndHTTPActionIdentityNamespaces
      // in both hosts for the lower-layer, actor-isolation-focused proof).
      await revealModeIfPresent(page, "home");
      const directEditForm = page.locator("#studioDirectEditForm");
      const undoButton = directEditForm.locator("[data-gosx-studio-operation-kind='undo']");
      await expect(undoButton, "the Undo button must render enabled for the SAME actor who made the collab commit").toBeEnabled({ timeout: 20_000 });
      await expect
        .poll(async () => (await undoButton.getAttribute("data-gosx-studio-history-operation-id")) ?? "", {
          timeout: 20_000,
          message: "the Undo button must carry a real historyOperationId, not the blank cursor the pre-fix identity mismatch left it with",
        })
        .not.toBe("");

      const undoOutcome = page.evaluate(
        () => new Promise<{ kind: "committed" | "error"; detail: unknown }>((resolve) => {
          document.addEventListener("gosxstudio:operation-committed", (event) => resolve({ kind: "committed", detail: (event as CustomEvent).detail }), { once: true });
          document.addEventListener("gosxstudio:operation-error", (event) => resolve({ kind: "error", detail: (event as CustomEvent).detail }), { once: true });
        }),
      );
      await undoButton.click();
      const undone = await undoOutcome;
      expect(undone.kind, `undo of the collab-made edit must succeed for the SAME actor; got ${JSON.stringify(undone)}`).toBe("committed");

      // The revert reaches the SAME durable store the canvas surface reads
      // from — a full reload re-serves the pre-marker headline, not the
      // collab-committed marker.
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "advanced");
      const revertedEditable = page.locator(EDITABLE_HERO).first();
      await expect(revertedEditable).toBeVisible({ timeout: 30_000 });
      await expect(revertedEditable, "undo must revert the canvas-rendered headline back to its pre-collab-edit value").toHaveText(preEditHeadline, { timeout: 20_000 });
    } finally {
      await server.stop();
    }
  });

  // handoff-31, consequence (c): the collab conflict path for inline edit.
  // Mirrors reference_apps_collaboration_direct_edit_form_test.ts's
  // "stale same-field form save is honestly rejected" scenario exactly, but
  // driving the CANVAS CONTENTEDITABLE surface on both sides instead of the
  // direct-edit-panel FORM: two authenticated sessions, both viewing the
  // canvas board, session A commits first; session B never reloads (so its
  // collabruntime head cache for hero.headline never learns A's new head —
  // collabruntime's accepted() deliberately never rebases an already-synced
  // session's cache from another actor's live broadcast, HANDOFF-09 Defect
  // 1) and its second, different-value commit is honestly rejected instead
  // of silently last-write-winning.
  test("Muddy/Noni wasm-free: stale same-field canvas inline edit between two sessions is honestly rejected, not silently applied", async ({ browser, request }) => {
    const server = await startMuddyCanvasHTMLSurfaceCollaboration(request);
    const contextA = await browser.newContext();
    const contextB = await browser.newContext();
    try {
      const pageA = await contextA.newPage();
      const pageB = await contextB.newPage();

      await pageA.goto(`${server.baseURL}/__test/collaboration/signin/author-a`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await pageB.goto(`${server.baseURL}/__test/collaboration/signin/author-b`, { waitUntil: "domcontentloaded", timeout: 60_000 });

      const panelA = pageA.locator("[data-gosx-studio-collaboration='true']");
      const panelB = pageB.locator("[data-gosx-studio-collaboration='true']");
      await expect(panelA).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      await expect(panelB).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });

      await pageA.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await pageB.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(panelA).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      await expect(panelB).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      await revealModeIfPresent(pageA, "advanced");
      await revealModeIfPresent(pageB, "advanced");
      await expect(pageA.locator(OVERLAY_HERO).first()).toBeAttached({ timeout: 60_000 });
      await expect(pageB.locator(OVERLAY_HERO).first()).toBeAttached({ timeout: 60_000 });
      const editableA = pageA.locator(EDITABLE_HERO).first();
      const editableB = pageB.locator(EDITABLE_HERO).first();
      await expect(editableA).toBeAttached({ timeout: 30_000 });
      await expect(editableB).toBeAttached({ timeout: 30_000 });

      // A commits first.
      const committedA = pageA.evaluate(
        () => new Promise<{ kind: "committed" | "error" }>((resolve) => {
          document.addEventListener("gosxstudio:collaboration-operation-accepted", () => resolve({ kind: "committed" }), { once: true });
          document.addEventListener("gosxstudio:collaboration-operation-rejected", () => resolve({ kind: "error" }), { once: true });
        }),
      );
      await editableA.click();
      await pageA.keyboard.press("ControlOrMeta+A");
      await pageA.keyboard.type("COLLAB-A-CANVAS-marker", { delay: 8 });
      await pageA.evaluate((sel) => { (document.querySelector(sel) as HTMLElement | null)?.blur(); }, EDITABLE_HERO);
      expect((await committedA).kind, "A's commit must succeed").toBe("committed");

      // B never reloads: B's overlay still shows the pre-A content, and B's
      // own collaboration head cache for this exact target never learned
      // A's just-accepted head. Give B's already-open socket a beat to
      // receive A's broadcast (which must NOT rebase B's cache).
      await pageB.waitForTimeout(1000);

      const outcomeB = pageB.evaluate(
        () => new Promise<{ kind: "committed" | "error"; detail: unknown }>((resolve) => {
          document.addEventListener("gosxstudio:collaboration-operation-accepted", () => resolve({ kind: "committed", detail: null }), { once: true });
          document.addEventListener("gosxstudio:collaboration-operation-rejected", (event) => resolve({ kind: "error", detail: (event as CustomEvent).detail }), { once: true });
        }),
      );
      await editableB.click();
      await pageB.keyboard.press("ControlOrMeta+A");
      await pageB.keyboard.type("STALE-B-CANVAS-marker", { delay: 8 });
      await pageB.evaluate((sel) => { (document.querySelector(sel) as HTMLElement | null)?.blur(); }, EDITABLE_HERO);
      const resultB = await outcomeB;

      expect(resultB.kind, `B's stale commit must be honestly rejected, not silently applied; got ${JSON.stringify(resultB)}`).toBe("error");

      // Reload and confirm the store holds A's value, never B's.
      await pageA.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(pageA, "advanced");
      const persisted = await pollForEditableText(pageA, "COLLAB-A-CANVAS-marker");
      expect(persisted, `expected A's marker to persist; got=${JSON.stringify(persisted)}`).toContain("COLLAB-A-CANVAS-marker");
      expect(persisted).not.toContain("STALE-B-CANVAS-marker");
    } finally {
      await Promise.all([contextA.close(), contextB.close()]);
      await server.stop();
    }
  });
});

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
