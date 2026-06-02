import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvas } from "./reference_apps_harness";

// Live-proof e2e for the Canvas2D site-map board (GoSX Studio Canvas2D track).
//
// This is the FIRST real-browser run of the whole canvas2d track. The Go
// renderer (gosx-studio RenderSiteMapCanvasEngine), the browser bootstrap
// discovery (bootstrap-feature-engines.js: querySelectorAll
// [data-gosx-surface-kind=canvas2d] → __gosx_hydrate), and the rAF paint loop
// (__gosx_render_canvas + __gosx_paint_canvas_bundle) are all verified by
// unit/wasm tests but had never run in a real headless browser before this
// test. The Muddy/Noni admin editor renders the canvas board behind a flag
// (MUDDY_SITEMAP_CANVAS=1, default OFF — the DOM board stays the default).
//
// The assertion genuinely verifies PAINT, not just element existence:
//   1. wait for the gosx runtime ready signal (window.__gosx.ready /
//      <html data-gosx-runtime-ready="true">),
//   2. wait for the canvas2d paint hook to be installed
//      (window.__gosx_canvas_board_screen_transform — exported by the
//      full-build canvas painter) AND a board adapter registered,
//   3. assert the <canvas data-gosx-surface-kind="canvas2d"> exists with
//      non-zero CSS size,
//   4. read back the canvas pixels (getImageData) and require at least one
//      non-background pixel — i.e. the rAF loop actually drew the page-graph
//      rects/labels/links onto the 2D context.
//
// Polls with generous timeouts (WASM cold-load is slow); no arbitrary sleeps.
//
// ============================================================================
// STATUS: test.fixme — the canvas HYDRATES but does NOT paint board content.
// ============================================================================
// First real-browser run (2026-06-01, headless Chromium, Muddy editor with
// MUDDY_SITEMAP_CANVAS=1 against local ../gosx @ v0.24.0-50-g8ed0df4) proved:
//   ✓ editor selects the FULL shared WASM runtime (/gosx/runtime.wasm),
//   ✓ bootstrap-feature-engines.js discovers the <canvas
//     data-gosx-surface-kind="canvas2d"> placeholder,
//   ✓ window.__gosx.ready === true, <html data-gosx-runtime-ready="true">,
//   ✓ window.__gosx_canvas_board_screen_transform / __gosx_paint_canvas_bundle
//     / __gosx_render_canvas / __gosx_hydrate are all installed (function),
//   ✓ the canvas hydrates into a CanvasBoardAdapter (registered under the
//     bootstrap's internal id "gosx-engine-surface-1"),
//   ✓ the rAF paint loop runs and clears+fills the board BACKGROUND (#0f1720).
//
// BUT the board CONTENT does not paint. The server emits 102 nodes
// (30 lines + 36 rects + 36 labels) in data-gosx-engine-props, yet
// __gosx_render_canvas("gosx-engine-surface-1", 1280, 720, t) returns a
// bundle with NO drawable objects:
//   {"background":"#0f1720","camera":{"mode":"ortho2d",...},"environment":{}}
// (93 bytes, no objects/lines/labels arrays), so getImageData reads a flat
// #0f1720 canvas: distinctColors=1, nonBackground=0.
//
// ROOT CAUSE (gosx framework, NOT this editor wiring or gosx-studio):
//   vm.CanvasBoardAdapter.RenderBundle() builds objects ONLY from
//   prog.EngineNodes (the COMPILED .gsx engine-node path). A Go-constructed
//   gosx.CanvasBoard (what gosx-studio RenderSiteMapCanvasEngine emits) carries
//   its nodes in the runtime data-gosx-engine-props JSON instead. The browser
//   bootstrap intentionally hydrates that "static primitive" with an EMPTY
//   program ("{}") — see gosx client/bridge/bridge_canvasboard_full.go
//   DecodeCanvasBoardProgram ("a static CanvasBoard is a no-code primitive…
//   To keep that path crash-free, an empty payload is treated as {}"). So
//   prog.EngineNodes is empty and snapshot() returns nil → zero render objects.
//   The props.nodes → render-bundle path is simply NOT IMPLEMENTED in gosx; the
//   passing unit/wasm tests (TestHydrateReconcilerCanvas2DSucceeds) exercise the
//   compiled-EngineNodes path, never the props.nodes path.
//
// Fixing this requires a gosx-framework change (render gosx.CanvasBoard's
// props.nodes, OR compile RenderSiteMapCanvasEngine's nodes into EngineNodes),
// which is out of scope for this editor-flag slice. Flip this back to test()
// once gosx renders CanvasBoard props.nodes — the assertion below already
// verifies genuine paint (non-background pixels) and will go green then.

test.describe("@reference-apps canvas2d site-map board live paint", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test.fixme("Muddy/Noni editor canvas2d site-map board hydrates and paints in headless Chromium", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvas(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });

      // The editor shell must be present (proves we got the editor route).
      await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();

      // The canvas2d placeholder must be emitted server-side under the flag.
      const canvas = page.locator("canvas[data-gosx-surface-kind='canvas2d']").first();
      await expect(canvas, "canvas2d placeholder should be emitted under MUDDY_SITEMAP_CANVAS=1").toBeAttached({ timeout: 30_000 });
      await expect(canvas).toHaveAttribute("data-gosx-engine-component", "CanvasBoard");

      // Wait for the gosx runtime to come ready. The canvas discovery
      // (nn() in bootstrap-feature-engines.js) runs inside runtimeReady, so
      // once the runtime reports ready the surface mount has been attempted.
      await page.waitForFunction(() => {
        const w = window as unknown as { __gosx?: { ready?: boolean } };
        return w.__gosx?.ready === true ||
          document.documentElement.getAttribute("data-gosx-runtime-ready") === "true";
      }, null, { timeout: 120_000 });

      // The full-build canvas painter installs __gosx_canvas_board_screen_transform
      // and __gosx_paint_canvas_bundle on evaluation. If these are missing the
      // served bundle is the islands-only / stale build (no canvas2d reconciler).
      await page.waitForFunction(() => {
        const w = window as unknown as {
          __gosx_canvas_board_screen_transform?: unknown;
          __gosx_paint_canvas_bundle?: unknown;
          __gosx_render_canvas?: unknown;
        };
        return typeof w.__gosx_canvas_board_screen_transform === "function" &&
          typeof w.__gosx_paint_canvas_bundle === "function" &&
          typeof w.__gosx_render_canvas === "function";
      }, null, { timeout: 120_000 });

      // The canvas must have a non-zero rendered (CSS) size, otherwise the
      // paint loop has nothing to draw into and getImageData is meaningless.
      const box = await canvas.boundingBox();
      expect(box, "canvas should have a layout box").not.toBeNull();
      expect(box!.width, "canvas CSS width should be non-zero").toBeGreaterThan(0);
      expect(box!.height, "canvas CSS height should be non-zero").toBeGreaterThan(0);

      // PAINT PROOF: poll the canvas backing store until the rAF loop has
      // drawn at least one non-background pixel. We compare against the board
      // background (#0f1720, the gosx-studio canvas default). A blank/transparent
      // canvas — or one filled only with the background color — fails.
      const paint = await pollForPaint(page);
      expect(
        paint.painted,
        `canvas should paint non-background pixels (last sample: ${JSON.stringify(paint)}; console: ${JSON.stringify(consoleErrors.slice(-12))})`,
      ).toBe(true);

      // Sanity: zero unexpected console errors that mention the canvas path.
      const canvasErrors = consoleErrors.filter((line) => /canvas|surface|hydrate|__gosx_render/i.test(line));
      expect(canvasErrors, `unexpected canvas2d console errors: ${JSON.stringify(canvasErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

type PaintSample = {
  painted: boolean;
  width: number;
  height: number;
  nonBackground: number;
  distinctColors: number;
  reason: string;
};

// pollForPaint repeatedly reads the canvas 2D backing store and reports
// whether the rAF paint loop has drawn anything beyond the flat background.
// Returns as soon as paint is detected, or after the deadline with the last
// sample (so the failure message carries real evidence).
async function pollForPaint(page: Page): Promise<PaintSample> {
  const deadline = Date.now() + 60_000;
  let last: PaintSample = { painted: false, width: 0, height: 0, nonBackground: 0, distinctColors: 0, reason: "not-sampled" };
  while (Date.now() < deadline) {
    last = await page.evaluate(() => {
      const canvas = document.querySelector("canvas[data-gosx-surface-kind='canvas2d']");
      if (!(canvas instanceof HTMLCanvasElement)) {
        return { painted: false, width: 0, height: 0, nonBackground: 0, distinctColors: 0, reason: "no-canvas" };
      }
      const ctx = canvas.getContext("2d");
      if (!ctx) {
        return { painted: false, width: 0, height: 0, nonBackground: 0, distinctColors: 0, reason: "no-2d-context" };
      }
      const w = canvas.width;
      const h = canvas.height;
      if (w === 0 || h === 0) {
        return { painted: false, width: w, height: h, nonBackground: 0, distinctColors: 0, reason: "zero-backing-store" };
      }
      let data: Uint8ClampedArray;
      try {
        data = ctx.getImageData(0, 0, w, h).data;
      } catch (error) {
        return { painted: false, width: w, height: h, nonBackground: 0, distinctColors: 0, reason: `getImageData-threw:${String(error)}` };
      }
      // Board background is #0f1720 (gosx-studio canvas default). Count pixels
      // that differ from it (with a small tolerance) and that are opaque —
      // those are the painted rects/labels/links.
      const bg = { r: 0x0f, g: 0x17, b: 0x20 };
      const tol = 10;
      let nonBackground = 0;
      const colors = new Set<number>();
      // Sample on a stride to keep the readback cheap on large canvases.
      const stride = 4 * Math.max(1, Math.floor((w * h) / 200000));
      for (let i = 0; i < data.length; i += stride) {
        const r = data[i];
        const g = data[i + 1];
        const b = data[i + 2];
        const a = data[i + 3];
        if (a === 0) continue;
        colors.add((r << 16) | (g << 8) | b);
        if (Math.abs(r - bg.r) > tol || Math.abs(g - bg.g) > tol || Math.abs(b - bg.b) > tol) {
          nonBackground++;
        }
      }
      return {
        painted: nonBackground > 0 && colors.size > 1,
        width: w,
        height: h,
        nonBackground,
        distinctColors: colors.size,
        reason: nonBackground > 0 ? "painted" : "background-only",
      };
    });
    if (last.painted) return last;
    await page.waitForTimeout(250);
  }
  return last;
}
