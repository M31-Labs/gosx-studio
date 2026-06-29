import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M7 ENRICHED CARDS + LEVEL-OF-DETAIL (LOD) swap on the
// WASM-free Canvas2D site-map board.
//
// Builds on the M0/M2 fixture (MUDDY_CANVAS_WASM_FREE=1 + MUDDY_EDITOR_SCENE_DOM=1,
// see reference_apps_canvas_html_surface_test.ts): the host injects ONE real
// Kind:"html" "page:home" full-page surface that the WASM-free client mounts as a
// live DOM element ([data-gosx-canvas-html]) in a camera-positioned overlay above
// the Canvas2D board. M7 adds two things this file proves end-to-end:
//
//   (A) ENRICHED CARDS — gosx-studio (sitemap_canvas.go) now stacks each PAGE
//       card's route as a muted subtitle LABEL a row under the card title. That
//       label flows through the SAME server-precomputed inline RenderBundle that
//       paints the board (<script data-gosx-canvas-bundle>…</script>), landing in
//       bundle.labels with an id ending ":route" and text that looks like a route
//       ("/" or "/something"). Asserting it in the live bundle proves the
//       enriched-card data reached the board.
//
//   (B) LOD SWAP — the muddy painter only mounts live DOM surfaces when
//       camera.z >= CANVAS_LOD_SURFACE_ZOOM (the painter's exported threshold).
//       Below that (zoomed-out overview) the enriched Canvas2D cards stand in for
//       the pages and ZERO live surfaces are mounted; at/above it the live
//       surface(s) mount again so the page is editable up close. We drive the
//       camera across the threshold and assert the [data-gosx-canvas-html]
//       overlay count swaps 0 ⇄ ≥1 accordingly. A focus-guard exception is also
//       asserted: a surface the user is actively editing is never torn down even
//       when zoomed out.
//
// Honesty discipline (matching the sibling canvas tests): the LOD threshold is
// READ FROM THE PAGE (window.GoSXStudioCanvas2DPainterRuntime.CANVAS_LOD_SURFACE_ZOOM) — no
// magic number — and the route label is DISCOVERED in the live bundle, not
// hardcoded. Zoom is driven through the WASM-free client's documented test hook
// (canvas.GoSXStudioCanvasWasmFree.setCamera, the same hook the M0/M2 tests use to
// "drive a deterministic camera without reaching into the closure"); each change
// is followed by a poll on the live overlay surface count with a deadline so the
// assertion only fires after the rAF render has applied. A real wheel gesture over
// the canvas is additionally exercised to confirm the production wheel-zoom path
// moves camera.z in the expected direction.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy built against the
// gosx/gosx-studio under test (the harness `go run`s muddy, so the embedded M7
// painter/client JS and the gosx-studio route-label code compile fresh).

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const BUNDLE_SELECTOR = "[data-gosx-canvas-bundle]";
const OVERLAY_SURFACE = "[data-gosx-canvas-html]";
// M10: the full-page surface root carries data-gosx-html-key="page:home" but is a
// non-editable wrapper; the contenteditable hero heading the focus-guard step
// focuses is the nested <h1 data-studio-field="home.hero.headline"> INSIDE it.
const EDITABLE_HERO = "[data-studio-field='home.hero.headline']";

type CameraHook = {
  camera: () => { x: number; y: number; z: number };
  setCamera: (x: number, y: number, z: number) => void;
  repaint: () => void;
  bundle: () => unknown;
};

test.describe("@reference-apps canvas2d site-map WASM-free enriched cards + LOD", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni wasm-free: page cards carry a route subtitle label in the bundle, and the live-surface overlay swaps to cards-only when zoomed out below the LOD threshold (with a focus-guard)", async ({ page, request }) => {
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
      await expect(canvas, "the muddy WASM-free canvas must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

      // The WASM-free client + painter must mount, the painter must export the LOD
      // threshold (M7), and the client must expose the camera test hook. A missing
      // CANVAS_LOD_SURFACE_ZOOM export means the served bundle predates M7 (stale
      // dist): fail loud rather than fake.
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
        const painter = w.GoSXStudioCanvas2DPainterRuntime as { paint?: unknown; renderCanvasBoardHTML?: unknown; CANVAS_LOD_SURFACE_ZOOM?: unknown } | undefined;
        const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { setCamera?: unknown; camera?: unknown } }) | null;
        return !!painter && typeof painter.paint === "function" && typeof painter.renderCanvasBoardHTML === "function" &&
          typeof painter.CANVAS_LOD_SURFACE_ZOOM === "number" &&
          !!rt && typeof rt.setState === "function" &&
          document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
          !!el && !!el.GoSXStudioCanvasWasmFree &&
          typeof el.GoSXStudioCanvasWasmFree.setCamera === "function" && typeof el.GoSXStudioCanvasWasmFree.camera === "function";
      }, null, { timeout: 120_000 });

      // Read the LOD threshold straight off the page so the assertions gate on the
      // REAL constant the painter ships — no magic number.
      const lodZoom = await page.evaluate(() => {
        const w = window as unknown as Record<string, unknown>;
        const painter = w.GoSXStudioCanvas2DPainterRuntime as { CANVAS_LOD_SURFACE_ZOOM?: number } | undefined;
        return painter && typeof painter.CANVAS_LOD_SURFACE_ZOOM === "number" ? painter.CANVAS_LOD_SURFACE_ZOOM : NaN;
      });
      expect(Number.isFinite(lodZoom) && lodZoom > 0, `painter must export a numeric LOD threshold; got ${JSON.stringify(lodZoom)}`).toBe(true);

      // ── (A) ENRICHED-CARD DATA: route subtitle label in the bundle ────────────
      // Read the inline server bundle JSON and prove a page card's route reached
      // the board as a muted subtitle label (id ends ":route", text looks like a
      // route). This is the enriched-card data flowing to the board.
      const routeLabel = await page.evaluate((sel) => {
        const el = document.querySelector(sel);
        const raw = el ? (el.textContent || "").trim() : "";
        if (!raw) return { parsed: false, labels: [] as Array<{ id?: string; text?: string; kind?: string }>, match: null as null | { id?: string; text?: string } };
        let bundle: { labels?: Array<{ id?: string; text?: string; kind?: string }> } | null = null;
        try { bundle = JSON.parse(raw); } catch { return { parsed: false, labels: [], match: null }; }
        const labels = Array.isArray(bundle?.labels) ? bundle!.labels! : [];
        // A route subtitle: id ends with ":route" AND its text reads as a route
        // ("/" or "/something"). The id gate keeps this off the title labels; the
        // text shape proves it is a real route value, not an arbitrary string.
        const match = labels.find((l) =>
          typeof l?.id === "string" && /:route$/.test(l.id) &&
          typeof l?.text === "string" && /^\/(\S*)$/.test(l.text.trim()),
        ) ?? null;
        return { parsed: true, labels, match };
      }, BUNDLE_SELECTOR);
      expect(routeLabel.parsed, "the inline [data-gosx-canvas-bundle] JSON must parse").toBe(true);
      expect(
        routeLabel.match,
        `a page card must carry a route subtitle label (id ~ ":route", text ~ "/…") in bundle.labels; labels=${JSON.stringify(routeLabel.labels)}`,
      ).toBeTruthy();
      expect(
        (routeLabel.match!.text || "").trim(),
        "the route subtitle label text must read as a route path",
      ).toMatch(/^\/(\S*)$/);

      // The live surface must mount at the default (>= threshold) camera before we
      // start swapping LOD — otherwise there is nothing to tear down.
      await expect(
        page.locator(EDITABLE_HERO).first(),
        "the injected hero html surface must mount as a live DOM surface at the default camera",
      ).toBeAttached({ timeout: 60_000 });
      const baselineCount = await surfaceCount(page);
      expect(baselineCount, "at the default camera (>= LOD threshold) at least one live surface must be mounted").toBeGreaterThanOrEqual(1);

      // ── (B) LOD SWAP — ZOOM OUT below threshold → cards-only (zero surfaces) ───
      // Drive camera.z below the LOD threshold via the documented test hook, then
      // poll the live overlay until the painter has torn the surfaces down.
      const belowZoom = lodZoom / 2;
      await setCameraZoom(page, belowZoom);
      const appliedBelow = await cameraZoom(page);
      expect(appliedBelow, `camera.z must be set below the LOD threshold; want ~${belowZoom}, got ${appliedBelow}`).toBeLessThan(lodZoom);
      const countBelow = await pollSurfaceCount(page, (n) => n === 0);
      expect(
        countBelow,
        `zoomed OUT below LOD (z=${appliedBelow} < ${lodZoom}) the overlay must hold ZERO live surfaces (cards stand in); got ${countBelow}`,
      ).toBe(0);

      // ── (B) LOD SWAP — ZOOM IN to/above threshold → live surface mounts again ──
      const aboveZoom = Math.max(lodZoom, lodZoom + 0.5);
      await setCameraZoom(page, aboveZoom);
      const appliedAbove = await cameraZoom(page);
      expect(appliedAbove, `camera.z must be set at/above the LOD threshold; want >=${lodZoom}, got ${appliedAbove}`).toBeGreaterThanOrEqual(lodZoom);
      const countAbove = await pollSurfaceCount(page, (n) => n >= 1);
      expect(
        countAbove,
        `zoomed IN at/above LOD (z=${appliedAbove} >= ${lodZoom}) at least one live surface must mount; got ${countAbove}`,
      ).toBeGreaterThanOrEqual(1);
      await expect(
        page.locator(EDITABLE_HERO).first(),
        "the hero surface's contenteditable must be back in the DOM once re-mounted above the LOD threshold",
      ).toBeAttached({ timeout: 30_000 });

      // ── PRODUCTION WHEEL PATH — a real wheel gesture moves camera.z AND swaps LOD
      // Confirm the production wheel-zoom handler (not just the test hook) drives
      // the camera across the LOD threshold. We start zoomed OUT below the
      // threshold (cards-only: ZERO overlay surfaces) so no mounted surface sits
      // over the canvas to intercept the wheel, then dispatch a real wheel IN over
      // the canvas center: camera.z must rise above the threshold AND a live
      // surface must re-mount — the production wheel path driving the LOD swap.
      await setCameraZoom(page, lodZoom / 2);
      await pollSurfaceCount(page, (n) => n === 0);
      const box = await canvas.boundingBox();
      expect(box, "the canvas must have a layout box to dispatch wheel events over").toBeTruthy();
      // Wheel over a near-corner of the canvas rather than dead-center: once a live
      // surface re-mounts (above threshold) it overlays the board and would
      // intercept a centered wheel, but the corner stays clear of the surface so
      // subsequent wheel notches keep reaching the canvas's own handler.
      const cx = box!.x + box!.width * 0.1;
      const cy = box!.y + box!.height * 0.1;
      await page.mouse.move(cx, cy);
      const zBeforeWheelIn = await cameraZoom(page);
      // A real wheel IN (deltaY < 0) over the canvas. Repeat notches until the
      // production handler has carried camera.z across the LOD threshold (each
      // notch is multiplicative + clamped, so several may be needed from a low z).
      let zAfterWheelIn = zBeforeWheelIn;
      for (let i = 0; i < 12 && zAfterWheelIn < lodZoom; i++) {
        await page.mouse.wheel(0, -240); // deltaY < 0 → zoom IN
        zAfterWheelIn = await pollCameraZoom(page, (z) => z > zAfterWheelIn);
      }
      expect(
        zAfterWheelIn,
        `a real wheel-in gesture over the canvas must raise camera.z above the LOD threshold (production wheel path); before=${zBeforeWheelIn}, after=${zAfterWheelIn}, lod=${lodZoom}`,
      ).toBeGreaterThanOrEqual(lodZoom);
      const wheelMountedCount = await pollSurfaceCount(page, (n) => n >= 1);
      expect(
        wheelMountedCount,
        `the production wheel-in path must drive the LOD swap so a live surface re-mounts; got ${wheelMountedCount}`,
      ).toBeGreaterThanOrEqual(1);
      // And a real wheel OUT (deltaY > 0) must lower camera.z back below the
      // threshold, swapping the overlay back to cards-only. Re-anchor the cursor at
      // the clear corner first (the re-mounted surface must not intercept).
      await page.mouse.move(cx, cy);
      const zBeforeWheelOut = await cameraZoom(page);
      let zAfterWheelOut = zBeforeWheelOut;
      for (let i = 0; i < 12 && zAfterWheelOut >= lodZoom; i++) {
        await page.mouse.wheel(0, 240); // deltaY > 0 → zoom OUT
        zAfterWheelOut = await pollCameraZoom(page, (z) => z < zAfterWheelOut);
      }
      expect(
        zAfterWheelOut,
        `a real wheel-out gesture over the canvas must lower camera.z below the LOD threshold; before=${zBeforeWheelOut}, after=${zAfterWheelOut}, lod=${lodZoom}`,
      ).toBeLessThan(lodZoom);
      const wheelCardsOnlyCount = await pollSurfaceCount(page, (n) => n === 0);
      expect(
        wheelCardsOnlyCount,
        `the production wheel-out path must drive the LOD swap back to cards-only (zero surfaces); got ${wheelCardsOnlyCount}`,
      ).toBe(0);

      // ── FOCUS-GUARD — an actively-edited surface is never torn down on zoom-out ─
      // Re-mount the surface (zoom in), focus its contenteditable, then zoom out
      // below the threshold. The painter must KEEP the focused surface mounted even
      // though every other surface would be torn down at this zoom.
      await setCameraZoom(page, aboveZoom);
      await pollSurfaceCount(page, (n) => n >= 1);
      const editable = page.locator(EDITABLE_HERO).first();
      await expect(editable, "the hero surface must be re-mounted before focusing it").toBeAttached({ timeout: 30_000 });
      await editable.click();
      await page.waitForFunction((sel) => {
        const el = document.querySelector(sel);
        return !!el && document.activeElement === el;
      }, EDITABLE_HERO, { timeout: 10_000 });
      // Zoom out below the threshold while the surface holds focus.
      await setCameraZoom(page, belowZoom);
      const guardedZoom = await cameraZoom(page);
      expect(guardedZoom, "camera.z must be below the LOD threshold for the focus-guard assertion").toBeLessThan(lodZoom);
      // Hold a few rAF frames so any tear-down would have run, then assert the
      // focused surface survived and still holds focus.
      await page.waitForTimeout(400);
      const guard = await page.evaluate((sel) => {
        const el = document.querySelector(sel);
        const surface = el ? el.closest("[data-gosx-canvas-html]") : null;
        return {
          editableStillAttached: !!el && document.body.contains(el),
          surfaceStillAttached: !!surface && document.body.contains(surface),
          stillFocused: !!el && document.activeElement === el,
        };
      }, EDITABLE_HERO);
      expect(
        guard.surfaceStillAttached,
        `the focus-guard must keep the actively-edited surface mounted while zoomed out below LOD; guard=${JSON.stringify(guard)}`,
      ).toBe(true);
      expect(guard.editableStillAttached, "the focused contenteditable must survive the zoom-out tear-down").toBe(true);
      expect(guard.stillFocused, "the guarded surface must retain focus across the zoom-out").toBe(true);

      // No WASM-free client console errors should have leaked. (Mirrors the sibling
      // wasm-free tests' filter; the dev-mode "gosx/relay.js" MIME warning that
      // `go run` emits without a served dist is unrelated and intentionally not
      // matched.)
      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

async function surfaceCount(page: Page): Promise<number> {
  return page.locator(OVERLAY_SURFACE).count();
}

async function pollSurfaceCount(page: Page, predicate: (n: number) => boolean): Promise<number> {
  const deadline = Date.now() + 10_000;
  let last = await surfaceCount(page);
  while (Date.now() < deadline) {
    last = await surfaceCount(page);
    if (predicate(last)) return last;
    await page.waitForTimeout(100);
  }
  return last;
}

async function cameraZoom(page: Page): Promise<number> {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel) as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { camera: () => { z: number } } }) | null;
    const hook = el && el.GoSXStudioCanvasWasmFree;
    return hook ? hook.camera().z : NaN;
  }, CANVAS_SELECTOR);
}

async function pollCameraZoom(page: Page, predicate: (z: number) => boolean): Promise<number> {
  const deadline = Date.now() + 5_000;
  let last = await cameraZoom(page);
  while (Date.now() < deadline) {
    last = await cameraZoom(page);
    if (predicate(last)) return last;
    await page.waitForTimeout(60);
  }
  return last;
}

async function setCameraZoom(page: Page, z: number): Promise<void> {
  await page.evaluate(({ sel, zoom }) => {
    const el = document.querySelector(sel) as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { camera: () => { x: number; y: number; z: number }; setCamera: (x: number, y: number, z: number) => void; repaint: () => void } }) | null;
    const hook = el && el.GoSXStudioCanvasWasmFree;
    if (!hook) return;
    const cam = hook.camera();
    hook.setCamera(cam.x, cam.y, zoom);
    hook.repaint();
  }, { sel: CANVAS_SELECTOR, zoom: z });
}
