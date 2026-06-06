import { expect, test, type Page, type Request } from "@playwright/test";
import { startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M2 canvas inline-edit LOOP — edit a contenteditable HTML
// surface field on the WASM-free Canvas2D board and prove it PERSISTS across a
// full page reload.
//
// Builds directly on the M0 fixture (MUDDY_CANVAS_WASM_FREE=1 +
// MUDDY_EDITOR_SCENE_DOM=1, see reference_apps_canvas_html_surface_test.ts): the
// host injects ONE Kind:"html" "page:home" full-page surface whose nested <h1>
// shows the REAL current store hero headline and carries
// data-studio-field="home.hero.headline" + contenteditable. M2 adds the muddy-side commit bridge (canvas_inline_edit.js,
// window.__muddyCanvasInlineEdit): on blur / Enter it POSTs an edit-commit to
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
// Honesty discipline (matching the sibling canvas tests): the overlay element +
// its editable child are DISCOVERED from the live DOM the client renders; the
// marker is freshly generated per run (timestamp + random nonce) so the reload
// assertion cannot false-pass on stale/seed content. The marker is kept to
// HTML-safe characters (letters/digits/spaces/hyphens) so it survives the server's
// html.EscapeString round-trip byte-for-byte in textContent.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy dist rebuilt against the
// gosx/gosx-studio under test (`gosx build .` in muddy-noni-commerce).

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
      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (selection sink)").toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "the muddy WASM-free canvas must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

      // The WASM-free client + painter must mount, the DOM board runtime (selection
      // sink) must be present, AND the M2 inline-edit commit bridge must be installed
      // (window.__muddyCanvasInlineEdit). A missing inline-edit module means the
      // served bundle predates the M2 work (stale dist): fail loud rather than fake.
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
