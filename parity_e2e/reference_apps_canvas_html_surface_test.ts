import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the WASM-FREE Canvas2D in-surface DOM editing path — the M0
// "HTML surface" payoff.
//
// In wasm-free mode (MUDDY_CANVAS_WASM_FREE=1) the editor renders + drives the
// Canvas2D site-map board with NO gosx client WASM. With MUDDY_EDITOR_SCENE_DOM=1
// the host also injects ONE real Kind:"html" CanvasBoardNode (the "hero" node)
// onto the board. That node serializes through the SAME server-precomputed
// RenderBundle path as the site-map rects (gosx vm.canvasBoardHTML →
// RenderBundle.HTML), so it lands in bundle.html. The WASM-free client mounts each
// bundle.html entry as a camera-positioned DOM element in an overlay above the
// canvas ([data-gosx-canvas-html="hero"]), with the host markup inside it
// (<h1 data-gosx-html-key="hero" contenteditable>Hello</h1>).
//
// This file asserts, end-to-end in headless Chromium, the two M0 in-surface
// behaviors:
//
//   (1) CLICK-TO-SELECT — a click on the rendered [data-gosx-html-key="hero"]
//       element walks to its data-gosx-html-key and calls
//       GoSXStudioSiteMapRuntime.setState({selectedNode:"hero"}) (the SAME public
//       API a DOM-board click uses), so the DOM site-map board's selection attr
//       (data-studio-site-map-selected-node) reflects "hero".
//   (2) CONTENTEDITABLE — focusing the contenteditable inside the surface, typing
//       text, lands that text in the surface DOM and is NOT clobbered by the rAF
//       paint loop (renderCanvasBoardHTML skips innerHTML rewrites while a
//       descendant is focused).
//
// Honesty discipline (matching the sibling canvas tests): the overlay element +
// its editable child are DISCOVERED from the live DOM the client renders — nothing
// is hardcoded beyond the host-authored "hero" id/markup the fixture injects. If
// either behavior genuinely does not work WASM-free, the offending assertion is
// left with the observed evidence rather than faked.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy dist rebuilt against the
// gosx/gosx-studio under test (`gosx build .` in muddy-noni-commerce).

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const OVERLAY_HERO = "[data-gosx-canvas-html='hero']";
const EDITABLE_HERO = "[data-gosx-html-key='hero']";

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
        const painter = w.__muddyCanvas2DPainter as { paint?: unknown; renderCanvasBoardHTML?: unknown } | undefined;
        const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { __muddyCanvasWasmFree?: unknown }) | null;
        return !!painter && typeof painter.paint === "function" && typeof painter.renderCanvasBoardHTML === "function" &&
          !!rt && typeof rt.setState === "function" &&
          document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
          !!el && !!el.__muddyCanvasWasmFree;
      }, null, { timeout: 120_000 });

      // The injected html surface must serialize into the inline bundle's html
      // records — otherwise the fixture (MUDDY_EDITOR_SCENE_DOM=1 → ExtraNodes)
      // never reached bundle.html. Read it straight off the live bundle so a
      // failure here pinpoints the fixture seam vs. the overlay rendering.
      const heroInBundle = await page.evaluate((sel) => {
        const el = document.querySelector(sel) as (HTMLCanvasElement & { __muddyCanvasWasmFree?: { bundle: () => unknown } }) | null;
        const hook = el && el.__muddyCanvasWasmFree;
        if (!hook) return { hasHook: false, html: [] as unknown[] };
        const bundle = hook.bundle() as { html?: Array<{ id?: string; markup?: string }> } | null;
        const html = (bundle && Array.isArray(bundle.html)) ? bundle.html : [];
        return { hasHook: true, html };
      }, CANVAS_SELECTOR);
      expect(heroInBundle.hasHook, "the WASM-free client test hook must expose the live bundle").toBe(true);
      const heroEntry = heroInBundle.html.find((h) => (h as { id?: string }).id === "hero") as { id?: string; markup?: string } | undefined;
      expect(
        heroEntry,
        `the injected Kind:"html" node must land in bundle.html as id "hero"; bundle.html=${JSON.stringify(heroInBundle.html)}`,
      ).toBeTruthy();
      expect(
        heroEntry!.markup || "",
        "the html surface markup must carry the contenteditable hero heading",
      ).toMatch(/data-gosx-html-key="hero"/);

      // The WASM-free client + painter must mount the html surface into a
      // camera-positioned overlay element above the canvas.
      const overlayHero = page.locator(OVERLAY_HERO).first();
      await expect(
        overlayHero,
        "the painter must mount the html surface as an overlay element [data-gosx-canvas-html='hero'] above the canvas",
      ).toBeAttached({ timeout: 60_000 });

      const editable = page.locator(EDITABLE_HERO).first();
      await expect(
        editable,
        "the host markup's contenteditable [data-gosx-html-key='hero'] must render inside the overlay",
      ).toBeAttached({ timeout: 30_000 });
      await expect(editable, "the editable hero element must be visible (overlay sized + positioned)").toBeVisible({ timeout: 30_000 });
      // Sanity: the injected element is the host markup we authored.
      const contentEditableAttr = await editable.getAttribute("contenteditable");
      expect(contentEditableAttr, "the hero element must be contenteditable").toBe("true");

      // ── (1) CLICK-TO-SELECT ──────────────────────────────────────────────────
      // Click the rendered surface element; the overlay click-delegate walks to
      // data-gosx-html-key and calls setState({selectedNode:"hero"}).
      const selectedBefore = await readSelectedNode(page);
      await editable.click();
      const selected = await pollForSelectedNode(page, "hero");
      expect(
        selected,
        `clicking the html surface should set board ${SELECTED_NODE_ATTR}="hero"; before=${JSON.stringify(selectedBefore)}, got=${JSON.stringify(selected)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toBe("hero");
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
        expect(runtimeSelected, "GoSXStudioSiteMapRuntime selection should also reflect hero").toBe("hero");
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
