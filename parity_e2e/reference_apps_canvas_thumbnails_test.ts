import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M8 FAITHFUL THUMBNAILS + 3-TIER LEVEL-OF-DETAIL (LOD)
// on the WASM-free Canvas2D site-map board.
//
// Builds on the M7 fixture (MUDDY_CANVAS_WASM_FREE=1 + MUDDY_EDITOR_SCENE_DOM=1,
// see reference_apps_canvas_cards_lod_test.ts): the host paints the Canvas2D
// board from one server-precomputed inline RenderBundle
// (<script data-gosx-canvas-bundle>…</script>) and, with MUDDY_EDITOR_SCENE_DOM=1,
// injects ONE real Kind:"html" "page:home" full-page surface that the WASM-free
// client mounts as a live DOM element ([data-gosx-canvas-html]) in a camera-positioned overlay
// above the board. M8 adds the FAITHFUL THUMBNAIL middle tier this file proves
// end-to-end:
//
//   (A) FAITHFUL-THUMBNAIL DATA — muddy generates a self-contained
//       raster PNG data URI (themed hero/title content) per page card and
//       gosx-studio attaches it as an "image" node at the card bounds, which rides
//       the SAME inline RenderBundle as a `sprite`
//       ({id, src:"data:image/png;base64,…", position, width, height}).
//       Asserting ≥1 bundle.sprites entry whose src starts "data:image/png"
//       proves the faithful-thumbnail data reaches the board.
//
//   (B) 3-TIER LOD — the painter renders these as overlay
//       <img data-gosx-canvas-thumb="<id>"> ONLY in the MEDIUM zoom band
//       (CANVAS_LOD_THUMB_ZOOM <= camera.z < CANVAS_LOD_SURFACE_ZOOM), giving a
//       three-tier hand-off:
//         far    (z <  THUMB_ZOOM)              → enriched cards only
//         medium (THUMB_ZOOM <= z < SURFACE)    → faithful thumbnail <img>
//         near   (z >= SURFACE_ZOOM)            → live editable dom surface
//       Both thresholds are exported on window.GoSXStudioCanvas2DPainterRuntime. We drive the
//       camera across all three bands via the WASM-free client's documented test
//       hook (canvas.GoSXStudioCanvasWasmFree.setCamera + .repaint — the same hook the
//       M7 cards-LOD test uses) and a real wheel gesture, and assert the
//       thumbnail-img / live-surface counts swap as the tier changes.
//
// Honesty discipline (matching the sibling canvas tests): BOTH LOD thresholds are
// READ FROM THE PAGE (window.GoSXStudioCanvas2DPainterRuntime) — no magic numbers — and the
// sprite/thumbnail is DISCOVERED in the live bundle/overlay, not hardcoded. Each
// camera change is followed by a poll on the live overlay counts with a deadline
// so the assertion only fires after the rAF render has applied.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy built against the
// gosx/gosx-studio under test (the harness `go run`s muddy, so the embedded M8
// painter/client JS and the gosx-studio sprite code compile fresh).

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const BUNDLE_SELECTOR = "[data-gosx-canvas-bundle]";
const OVERLAY_SURFACE = "[data-gosx-canvas-html]";
const THUMB_IMG = "[data-gosx-canvas-thumb]";
// M10: the full-page surface root carries data-gosx-html-key="page:home" but is a
// non-editable wrapper; the contenteditable hero heading is the nested
// <h1 data-studio-field="home.hero.headline"> INSIDE that surface.
const EDITABLE_HERO = "[data-studio-field='home.hero.headline']";
const PNG_DATA_PREFIX = "data:image/png";

test.describe("@reference-apps canvas2d site-map WASM-free faithful thumbnails + 3-tier LOD", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni wasm-free: faithful per-page thumbnails ride the bundle as PNG sprites, and the overlay swaps cards → thumbnail img → live surface across the 3-tier LOD bands", async ({ page, request }) => {
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

      // The WASM-free client + painter must mount, the painter must export BOTH LOD
      // thresholds (M8), and the client must expose the camera test hook. A missing
      // CANVAS_LOD_THUMB_ZOOM export means the served bundle predates M8 (stale
      // dist): fail loud rather than fake.
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
        const painter = w.GoSXStudioCanvas2DPainterRuntime as {
          paint?: unknown;
          renderCanvasBoardHTML?: unknown;
          renderCanvasBoardThumbnails?: unknown;
          CANVAS_LOD_SURFACE_ZOOM?: unknown;
          CANVAS_LOD_THUMB_ZOOM?: unknown;
        } | undefined;
        const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { setCamera?: unknown; camera?: unknown } }) | null;
        return !!painter && typeof painter.paint === "function" &&
          typeof painter.renderCanvasBoardHTML === "function" &&
          typeof painter.renderCanvasBoardThumbnails === "function" &&
          typeof painter.CANVAS_LOD_SURFACE_ZOOM === "number" &&
          typeof painter.CANVAS_LOD_THUMB_ZOOM === "number" &&
          !!rt && typeof rt.setState === "function" &&
          document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
          !!el && !!el.GoSXStudioCanvasWasmFree &&
          typeof el.GoSXStudioCanvasWasmFree.setCamera === "function" && typeof el.GoSXStudioCanvasWasmFree.camera === "function";
      }, null, { timeout: 120_000 });

      // Read BOTH LOD thresholds straight off the page so the assertions gate on the
      // REAL constants the painter ships — no magic numbers. THUMB_ZOOM must be the
      // lower bound of a non-empty medium band, i.e. strictly below SURFACE_ZOOM.
      const lod = await page.evaluate(() => {
        const w = window as unknown as Record<string, unknown>;
        const painter = w.GoSXStudioCanvas2DPainterRuntime as { CANVAS_LOD_SURFACE_ZOOM?: number; CANVAS_LOD_THUMB_ZOOM?: number } | undefined;
        return {
          thumb: painter && typeof painter.CANVAS_LOD_THUMB_ZOOM === "number" ? painter.CANVAS_LOD_THUMB_ZOOM : NaN,
          surface: painter && typeof painter.CANVAS_LOD_SURFACE_ZOOM === "number" ? painter.CANVAS_LOD_SURFACE_ZOOM : NaN,
        };
      });
      expect(Number.isFinite(lod.thumb) && lod.thumb > 0, `painter must export a numeric THUMB LOD threshold; got ${JSON.stringify(lod.thumb)}`).toBe(true);
      expect(Number.isFinite(lod.surface) && lod.surface > 0, `painter must export a numeric SURFACE LOD threshold; got ${JSON.stringify(lod.surface)}`).toBe(true);
      expect(
        lod.surface,
        `SURFACE_ZOOM (${lod.surface}) must sit strictly above THUMB_ZOOM (${lod.thumb}) for a non-empty medium band`,
      ).toBeGreaterThan(lod.thumb);

      // ── (A) FAITHFUL-THUMBNAIL DATA: PNG sprite in the bundle ──────────────────
      // Read the inline server bundle JSON and prove a faithful per-page thumbnail
      // reached the board as a sprite whose src is a self-contained
      // "data:image/png…" data URI. This is the faithful-thumbnail data flowing
      // to the board (decoding the PNG is the browser's job at render time).
      const sprite = await page.evaluate(({ sel, prefix }) => {
        const el = document.querySelector(sel);
        const raw = el ? (el.textContent || "").trim() : "";
        if (!raw) return { parsed: false, sprites: [] as Array<{ id?: string; src?: string }>, match: null as null | { id?: string; src?: string } };
        let bundle: { sprites?: Array<{ id?: string; src?: string }> } | null = null;
        try { bundle = JSON.parse(raw); } catch { return { parsed: false, sprites: [], match: null }; }
        const sprites = Array.isArray(bundle?.sprites) ? bundle!.sprites! : [];
        const match = sprites.find((s) =>
          typeof s?.src === "string" && s.src.indexOf(prefix) === 0,
        ) ?? null;
        // Don't ship the (potentially large) data URIs back wholesale — just ids +
        // src prefixes for diagnostics.
        const summary = sprites.map((s) => ({ id: s?.id, src: typeof s?.src === "string" ? s.src.slice(0, 32) : s?.src }));
        return { parsed: true, sprites: summary, match: match ? { id: match.id, src: match.src!.slice(0, 64) } : null };
      }, { sel: BUNDLE_SELECTOR, prefix: PNG_DATA_PREFIX });
      expect(sprite.parsed, "the inline [data-gosx-canvas-bundle] JSON must parse").toBe(true);
      expect(
        sprite.match,
        `a page card must carry a faithful thumbnail as a sprite (src ~ "${PNG_DATA_PREFIX}…") in bundle.sprites; sprites=${JSON.stringify(sprite.sprites)}`,
      ).toBeTruthy();
      expect(
        sprite.match!.src ?? "",
        "the sprite src must be a self-contained PNG data URI",
      ).toMatch(/^data:image\/png/);

      // The live surface must mount at the default (>= SURFACE_ZOOM) camera before
      // we start swapping LOD — it is the near tier we hand off to.
      await expect(
        page.locator(EDITABLE_HERO).first(),
        "the injected hero html surface must mount as a live DOM surface at the default camera",
      ).toBeAttached({ timeout: 60_000 });
      const baselineSurfaces = await surfaceCount(page);
      expect(baselineSurfaces, "at the default camera (>= SURFACE_ZOOM) at least one live surface must be mounted").toBeGreaterThanOrEqual(1);

      // ── (B) 3-TIER LOD ─────────────────────────────────────────────────────────
      // FAR band (z < THUMB_ZOOM): enriched CARDS ONLY — zero thumbnail imgs AND
      // zero live surfaces. Pick a z safely below THUMB_ZOOM.
      const farZoom = Math.max(lod.thumb / 2, 0.05);
      await setCameraZoom(page, farZoom);
      const appliedFar = await cameraZoom(page);
      expect(appliedFar, `camera.z must be below THUMB_ZOOM for the far band; want ~${farZoom}, got ${appliedFar}`).toBeLessThan(lod.thumb);
      const farThumbs = await pollCount(page, THUMB_IMG, (n) => n === 0);
      expect(
        farThumbs,
        `far band (z=${appliedFar} < THUMB ${lod.thumb}) must hold ZERO thumbnail imgs (cards only); got ${farThumbs}`,
      ).toBe(0);
      const farSurfaces = await pollCount(page, OVERLAY_SURFACE, (n) => n === 0);
      expect(
        farSurfaces,
        `far band (z=${appliedFar} < SURFACE ${lod.surface}) must hold ZERO live surfaces (cards only); got ${farSurfaces}`,
      ).toBe(0);

      // MEDIUM band (THUMB_ZOOM <= z < SURFACE_ZOOM): faithful thumbnail IMG mounts,
      // and NO live surface yet. Pick the midpoint of the band.
      const mediumZoom = (lod.thumb + lod.surface) / 2;
      await setCameraZoom(page, mediumZoom);
      const appliedMedium = await cameraZoom(page);
      expect(appliedMedium, `camera.z must be in [THUMB, SURFACE) for the medium band; want ~${mediumZoom}, got ${appliedMedium}`).toBeGreaterThanOrEqual(lod.thumb);
      expect(appliedMedium, `camera.z must be below SURFACE_ZOOM for the medium band; want ~${mediumZoom}, got ${appliedMedium}`).toBeLessThan(lod.surface);
      const mediumThumbs = await pollCount(page, THUMB_IMG, (n) => n >= 1);
      expect(
        mediumThumbs,
        `medium band (THUMB ${lod.thumb} <= z=${appliedMedium} < SURFACE ${lod.surface}) must mount at least one thumbnail img; got ${mediumThumbs}`,
      ).toBeGreaterThanOrEqual(1);
      // Still no live dom surface in the medium band — the thumbnail stands in.
      const mediumSurfaces = await pollCount(page, OVERLAY_SURFACE, (n) => n === 0);
      expect(
        mediumSurfaces,
        `medium band (z=${appliedMedium} < SURFACE ${lod.surface}) must hold ZERO live surfaces (the thumbnail stands in); got ${mediumSurfaces}`,
      ).toBe(0);
      // Faithfulness sanity: the mounted thumbnail img carries the PNG data URI src
      // (the same self-contained faithful data the bundle shipped; decoding it is
      // the browser's job).
      const thumbSrc = await page.evaluate((sel) => {
        const img = document.querySelector(sel) as HTMLImageElement | null;
        return img ? img.getAttribute("src") : null;
      }, THUMB_IMG);
      expect(thumbSrc ?? "", `the mounted thumbnail img src must be a faithful PNG data URI; got ${JSON.stringify(thumbSrc)}`).toMatch(/^data:image\/png/);

      // NEAR band (z >= SURFACE_ZOOM): the live editable dom surface owns the page —
      // zero thumbnail imgs AND at least one live surface (the LOD handed off).
      const nearZoom = Math.max(lod.surface, lod.surface + 0.5);
      await setCameraZoom(page, nearZoom);
      const appliedNear = await cameraZoom(page);
      expect(appliedNear, `camera.z must be at/above SURFACE_ZOOM for the near band; want >=${lod.surface}, got ${appliedNear}`).toBeGreaterThanOrEqual(lod.surface);
      const nearThumbs = await pollCount(page, THUMB_IMG, (n) => n === 0);
      expect(
        nearThumbs,
        `near band (z=${appliedNear} >= SURFACE ${lod.surface}) must hold ZERO thumbnail imgs (the live surface owns it); got ${nearThumbs}`,
      ).toBe(0);
      const nearSurfaces = await pollCount(page, OVERLAY_SURFACE, (n) => n >= 1);
      expect(
        nearSurfaces,
        `near band (z=${appliedNear} >= SURFACE ${lod.surface}) must mount at least one live surface (the LOD handed off); got ${nearSurfaces}`,
      ).toBeGreaterThanOrEqual(1);
      await expect(
        page.locator(EDITABLE_HERO).first(),
        "the hero surface's contenteditable must be in the DOM in the near tier",
      ).toBeAttached({ timeout: 30_000 });

      // ── PRODUCTION WHEEL PATH — a real wheel gesture walks back through the bands ─
      // Confirm the production wheel-zoom handler (not just the test hook) drives the
      // camera across the LOD thresholds. Start in the NEAR tier (live surface);
      // wheel OUT into the MEDIUM band and assert the thumbnail img re-appears and
      // the live surface tears down. Wheel over a near-corner of the canvas so a
      // mounted live surface never sits over the cursor to intercept the wheel.
      const box = await canvas.boundingBox();
      expect(box, "the canvas must have a layout box to dispatch wheel events over").toBeTruthy();
      const cx = box!.x + box!.width * 0.1;
      const cy = box!.y + box!.height * 0.1;
      await page.mouse.move(cx, cy);
      // Wheel OUT (deltaY > 0) until camera.z drops into the medium band.
      let zWheel = await cameraZoom(page);
      for (let i = 0; i < 16 && zWheel >= lod.surface; i++) {
        await page.mouse.wheel(0, 240); // deltaY > 0 → zoom OUT
        zWheel = await pollCameraZoom(page, (z) => z < zWheel);
      }
      expect(
        zWheel,
        `a real wheel-out gesture must lower camera.z below SURFACE_ZOOM (production wheel path); after=${zWheel}, surface=${lod.surface}`,
      ).toBeLessThan(lod.surface);
      expect(
        zWheel,
        `the wheel-out must land in (or above) the medium band, not below THUMB_ZOOM; after=${zWheel}, thumb=${lod.thumb}`,
      ).toBeGreaterThanOrEqual(lod.thumb);
      const wheelMediumThumbs = await pollCount(page, THUMB_IMG, (n) => n >= 1);
      expect(
        wheelMediumThumbs,
        `the production wheel-out path must drive the LOD swap so a thumbnail img re-mounts in the medium band; got ${wheelMediumThumbs}`,
      ).toBeGreaterThanOrEqual(1);
      const wheelMediumSurfaces = await pollCount(page, OVERLAY_SURFACE, (n) => n === 0);
      expect(
        wheelMediumSurfaces,
        `the production wheel-out path must tear down the live surface in the medium band; got ${wheelMediumSurfaces}`,
      ).toBe(0);

      // And a real wheel IN (deltaY < 0) back to the near tier: thumbnail img tears
      // down and the live surface re-mounts. Re-anchor at the clear corner first.
      await page.mouse.move(cx, cy);
      let zWheelIn = await cameraZoom(page);
      for (let i = 0; i < 16 && zWheelIn < lod.surface; i++) {
        await page.mouse.wheel(0, -240); // deltaY < 0 → zoom IN
        zWheelIn = await pollCameraZoom(page, (z) => z > zWheelIn);
      }
      expect(
        zWheelIn,
        `a real wheel-in gesture must raise camera.z to/above SURFACE_ZOOM (production wheel path); after=${zWheelIn}, surface=${lod.surface}`,
      ).toBeGreaterThanOrEqual(lod.surface);
      const wheelNearThumbs = await pollCount(page, THUMB_IMG, (n) => n === 0);
      expect(
        wheelNearThumbs,
        `the production wheel-in path must tear down the thumbnail img in the near tier; got ${wheelNearThumbs}`,
      ).toBe(0);
      const wheelNearSurfaces = await pollCount(page, OVERLAY_SURFACE, (n) => n >= 1);
      expect(
        wheelNearSurfaces,
        `the production wheel-in path must re-mount the live surface in the near tier; got ${wheelNearSurfaces}`,
      ).toBeGreaterThanOrEqual(1);

      // No WASM-free client console errors should have leaked. (Mirrors the sibling
      // wasm-free tests' filter; the dev-mode "gosx/relay.js" MIME warning that
      // `go run` emits without a served dist is unrelated and intentionally not
      // matched.)
      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|thumb|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

async function elementCount(page: Page, selector: string): Promise<number> {
  return page.locator(selector).count();
}

async function surfaceCount(page: Page): Promise<number> {
  return elementCount(page, OVERLAY_SURFACE);
}

async function pollCount(page: Page, selector: string, predicate: (n: number) => boolean): Promise<number> {
  const deadline = Date.now() + 10_000;
  let last = await elementCount(page, selector);
  while (Date.now() < deadline) {
    last = await elementCount(page, selector);
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
