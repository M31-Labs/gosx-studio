import { expect, test, type Page } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the WASM-FREE Canvas2D in-surface DOM editing path — the M0
// "HTML surface" payoff.
//
// In wasm-free mode (MUDDY_CANVAS_WASM_FREE=1) the editor renders + drives the
// Canvas2D site-map board with NO gosx client WASM. With MUDDY_EDITOR_SCENE_DOM=1
// the host also injects ONE real Kind:"html" CanvasBoardNode — the M10 full-page
// home surface (node id "page:home") — onto the board. That node serializes through
// the SAME server-precomputed RenderBundle path as the site-map rects (gosx
// vm.canvasBoardHTML → RenderBundle.HTML), so it lands in bundle.html. The WASM-free
// client mounts each bundle.html entry as a camera-positioned DOM element in an
// overlay above the canvas ([data-gosx-canvas-html="page:home"]), with the host
// full-page markup inside it: a non-editable surface root
// (<div data-gosx-html-key="page:home">) wrapping the editable hero heading
// (<h1 data-studio-field="home.hero.headline" contenteditable>…</h1>).
//
// This file asserts, end-to-end in headless Chromium, the two M0 in-surface
// behaviors:
//
//   (1) CLICK-TO-SELECT — a click on the rendered surface (the contenteditable hero
//       heading) walks up to the nearest data-gosx-html-key ("page:home") and calls
//       GoSXStudioSiteMapRuntime.setState({selectedNode:"page:home"}) (the SAME
//       public API a DOM-board click uses), so the DOM site-map board's selection
//       attr (data-studio-site-map-selected-node) reflects "page:home".
//   (2) CONTENTEDITABLE — focusing the contenteditable inside the surface, typing
//       text, lands that text in the surface DOM and is NOT clobbered by the rAF
//       paint loop (renderCanvasBoardHTML skips innerHTML rewrites while a
//       descendant is focused).
//
// Honesty discipline (matching the sibling canvas tests): the overlay element +
// its editable child are DISCOVERED from the live DOM the client renders — nothing
// is hardcoded beyond the host-authored "page:home" id/markup the fixture injects. If
// either behavior genuinely does not work WASM-free, the offending assertion is
// left with the observed evidence rather than faked.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const OVERLAY_HERO = "[data-gosx-canvas-html='page:home']";
// M10: the full-page surface root carries data-gosx-html-key="page:home" but is a
// non-editable wrapper <div>; the editable hero heading is the nested
// contenteditable <h1 data-studio-field="home.hero.headline"> INSIDE that surface.
const EDITABLE_HERO = "[data-studio-field='home.hero.headline']";

test.describe("@reference-apps canvas2d site-map WASM-free HTML surface", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni wasm-free: an injected html surface paints in the canvas overlay, click selects it, and contenteditable input lands in the surface DOM", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
      // HAZEL PROOF-OF-FIX: the legacy site-map/canvas board (incl. the HTML
      // surface overlay under test) now lives inside the "Advanced" mode panel.
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (selection sink)").toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "the muddy WASM-free canvas (no data-gosx-surface-kind) must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

      // The WASM-free client + painter must mount, and the DOM board runtime (the
      // selection sink) must be present — all WITHOUT any __gosx canvas entrypoint.
      // Any missing piece means the served bundle predates this work (stale dist):
      // fail loud rather than fake.
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
        const painter = w.GoSXStudioCanvas2DPainterRuntime as { paint?: unknown; renderCanvasBoardHTML?: unknown } | undefined;
        const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: unknown }) | null;
        return !!painter && typeof painter.paint === "function" && typeof painter.renderCanvasBoardHTML === "function" &&
          !!rt && typeof rt.setState === "function" &&
          document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
          !!el && !!el.GoSXStudioCanvasWasmFree;
      }, null, { timeout: 120_000 });

      // The injected html surface must serialize into the inline bundle's html
      // records — otherwise the fixture (MUDDY_EDITOR_SCENE_DOM=1 → ExtraNodes)
      // never reached bundle.html. Read it straight off the live bundle so a
      // failure here pinpoints the fixture seam vs. the overlay rendering.
      const heroInBundle = await page.evaluate((sel) => {
        const el = document.querySelector(sel) as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { bundle: () => unknown } }) | null;
        const hook = el && el.GoSXStudioCanvasWasmFree;
        if (!hook) return { hasHook: false, html: [] as unknown[] };
        const bundle = hook.bundle() as { html?: Array<{ id?: string; markup?: string }> } | null;
        const html = (bundle && Array.isArray(bundle.html)) ? bundle.html : [];
        return { hasHook: true, html };
      }, CANVAS_SELECTOR);
      expect(heroInBundle.hasHook, "the WASM-free client test hook must expose the live bundle").toBe(true);
      const heroEntry = heroInBundle.html.find((h) => (h as { id?: string }).id === "page:home") as { id?: string; markup?: string } | undefined;
      expect(
        heroEntry,
        `the injected full-page Kind:"html" node must land in bundle.html as id "page:home"; bundle.html=${JSON.stringify(heroInBundle.html)}`,
      ).toBeTruthy();
      expect(
        heroEntry!.markup || "",
        "the html surface markup must carry the contenteditable hero heading",
      ).toMatch(/data-studio-field="home\.hero\.headline"/);

      // The WASM-free client + painter must mount the html surface into a
      // camera-positioned overlay element above the canvas.
      const overlayHero = page.locator(OVERLAY_HERO).first();
      await expect(
        overlayHero,
        "the painter must mount the html surface as an overlay element [data-gosx-canvas-html='page:home'] above the canvas",
      ).toBeAttached({ timeout: 60_000 });

      const editable = page.locator(EDITABLE_HERO).first();
      await expect(
        editable,
        "the host markup's contenteditable [data-studio-field='home.hero.headline'] must render inside the overlay",
      ).toBeAttached({ timeout: 30_000 });
      await expect(editable, "the editable hero element must be visible (overlay sized + positioned)").toBeVisible({ timeout: 30_000 });
      // Sanity: the injected element is the host markup we authored.
      const contentEditableAttr = await editable.getAttribute("contenteditable");
      expect(contentEditableAttr, "the hero element must be contenteditable").toBe("true");

      // ── (1) CLICK-TO-SELECT ──────────────────────────────────────────────────
      // Click the rendered surface element; the overlay click-delegate walks to
      // data-gosx-html-key ("page:home") and calls setState({selectedNode:"page:home"}).
      const selectedBefore = await readSelectedNode(page);
      await editable.click();
      const selected = await pollForSelectedNode(page, "page:home");
      expect(
        selected,
        `clicking the html surface should set board ${SELECTED_NODE_ATTR}="page:home"; before=${JSON.stringify(selectedBefore)}, got=${JSON.stringify(selected)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toBe("page:home");
      // Cross-check the runtime exposes the same selection (parity with a DOM-board
      // click that flows through the identical setState API).
      const runtimeSelected = await page.evaluate(() => {
        const w = window as unknown as Record<string, unknown>;
        const rt = w.GoSXStudioSiteMapRuntime as { getState?: (root: Element) => { selectedNode?: string } } | undefined;
        const board = document.querySelector("[data-studio-site-map-board='true']");
        if (!rt || typeof rt.getState !== "function" || !board) return undefined;
        try {
          return rt.getState(board)?.selectedNode;
        } catch {
          return undefined;
        }
      });
      if (runtimeSelected !== undefined) {
        expect(runtimeSelected, "GoSXStudioSiteMapRuntime selection should also reflect page:home").toBe("page:home");
      }

      // ── (2) CONTENTEDITABLE ──────────────────────────────────────────────────
      // Focus the contenteditable, select-all, and type new text. The rAF paint
      // loop must NOT clobber the in-progress edit (renderCanvasBoardHTML skips
      // innerHTML rewrites while a descendant is focused).
      const typed = "Edited hero copy";
      await editable.click();
      await page.waitForFunction((sel) => {
        const el = document.querySelector(sel);
        return !!el && document.activeElement === el;
      }, EDITABLE_HERO, { timeout: 10_000 });
      // Select all existing content, then replace it.
      await page.keyboard.press("ControlOrMeta+A");
      await page.keyboard.type(typed, { delay: 12 });

      const landed = await pollForEditableText(page, typed);
      expect(
        landed,
        `typed text must land in the surface DOM and survive the rAF paint loop; got=${JSON.stringify(landed)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toContain(typed);

      // Hold for a few rAF frames, then re-read: the painter must not have wiped
      // the user's input on a subsequent repaint.
      await page.waitForTimeout(400);
      const stillThere = await editableText(page);
      expect(
        stillThere,
        `the rAF paint loop must not clobber contenteditable input after typing; got=${JSON.stringify(stillThere)}`,
      ).toContain(typed);

      // No WASM-free client console errors should have leaked. (Mirrors the
      // sibling wasm-free test's filter; the unrelated dev-mode "gosx/relay.js"
      // MIME warning that `go run` emits without a served dist is not part of this
      // path and is intentionally not matched.)
      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
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
  const deadline = Date.now() + 10_000;
  let last = "";
  while (Date.now() < deadline) {
    last = await editableText(page);
    if (last.includes(want)) return last;
    await page.waitForTimeout(120);
  }
  return last;
}
