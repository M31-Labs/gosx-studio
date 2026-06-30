import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvas } from "./reference_apps_harness";
import {
  canvasFingerprint2D,
  formatCanvasRenderEvidence,
  readCanvasWebGPUFrameSeq,
  waitForCanvasBoardRenderEvidence,
  waitForCanvasWebGPUFrameAdvance,
} from "./canvas_render_evidence";

// Live-proof e2e for Canvas2D site-map board INTERACTION (drag-to-pan,
// wheel-to-zoom toward cursor, click-to-pick).
//
// The sibling reference_apps_canvas_test.ts proves the board PAINTS. This file
// proves it RESPONDS: the bootstrap input handlers (_bridgeCanvasBoardEvents in
// gosx client/js/bootstrap-src/26b-feature-engines-prefix.js) dispatch real DOM
// pointer/wheel events to __gosx_canvas_event, which mutates the adapter's
// runtime camera (pan/zoom) and writes pick results into $surface.event.*.
//
// Honesty discipline: every assertion is driven off observable browser state —
// the rendered RenderBundle JSON (window.__gosx_render_canvas), the live screen
// transform (window.__gosx_canvas_board_screen_transform), route evidence from
// WebGPU host diagnostics or explicit fallback paint, and the shared-signal
// getter (window.__gosx_get_shared_signal).
// On-screen target positions are DISCOVERED from the live bundle, never
// hardcoded, so the test follows whatever the editor actually renders. Each
// interaction either passes against live evidence or is reported honestly.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const CANVAS_SELECTOR = "canvas[data-gosx-surface-kind='canvas2d']";

test.describe("@reference-apps canvas2d site-map board live interaction", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni canvas2d board pans, zooms, and picks in headless Chromium", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    const consoleWarnings: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
      if (message.type() === "warning") consoleWarnings.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvas(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "canvas2d placeholder should be emitted under MUDDY_SITEMAP_CANVAS=1").toBeAttached({ timeout: 30_000 });
      await expect(canvas, "full-runtime CanvasBoard should request the WebGPU backend").toHaveAttribute("data-gosx-canvas-backend", "webgpu");

      // Runtime ready + the full-build canvas painter AND the interaction entry
      // (__gosx_canvas_event) must be installed. If __gosx_canvas_event is
      // missing, the served bundle/WASM predate the interaction work — fail loud.
      await page.waitForFunction(() => {
        const w = window as unknown as { __gosx?: { ready?: boolean } };
        return w.__gosx?.ready === true ||
          document.documentElement.getAttribute("data-gosx-runtime-ready") === "true";
      }, null, { timeout: 120_000 });

      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        return typeof w.__gosx_canvas_board_screen_transform === "function" &&
          typeof w.__gosx_render_canvas === "function" &&
          typeof w.__gosx_canvas_event === "function" &&
          typeof w.__gosx_get_shared_signal === "function";
      }, null, { timeout: 120_000 });

      const box = await canvas.boundingBox();
      expect(box, "canvas should have a layout box").not.toBeNull();
      expect(box!.width, "canvas CSS width > 0").toBeGreaterThan(0);
      expect(box!.height, "canvas CSS height > 0").toBeGreaterThan(0);

      const renderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings);
      expect(
        renderEvidence.webgpuRoute || renderEvidence.fallback2D,
        `board must render before interaction; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);

      // ----------------------------------------------------------------------
      // (c) CLICK-TO-PICK (run FIRST, against the pristine camera where the
      // rects are on-screen): discover a real rect from the live bundle, compute
      // its on-screen center via the live screen transform, click there, and
      // assert $surface.event.selectedID equals that rect's id. Running pick
      // before pan/zoom keeps the target inside the viewport.
      // ----------------------------------------------------------------------
      const target = await pickTarget(page);
      expect(target, "the rendered bundle should expose at least one rect to pick").not.toBeNull();

      await page.mouse.move(box!.x + target!.screenX, box!.y + target!.screenY);
      await page.mouse.down();
      await page.mouse.up();

      const selected = await pollForSelectedID(page, target!.id);
      expect(
        selected,
        `click should select the rect under the cursor: expected ${target!.id}, ` +
          `got ${JSON.stringify(selected)} (console: ${JSON.stringify(consoleErrors.slice(-8))})`,
      ).toBe(target!.id);

      // ----------------------------------------------------------------------
      // (a) DRAG-TO-PAN: drag the canvas by a known screen delta and assert the
      // route produced new evidence (WebGPU frame seq or fallback 2D pixels).
      // The pan is verified two independent ways: the rendered camera.x/.y moved
      // by the expected world delta, AND the route-specific render evidence
      // advanced.
      // ----------------------------------------------------------------------
      const camBefore = await readCamera(page);
      const before = renderEvidence.fallback2D ? await canvasFingerprint2D(page, CANVAS_SELECTOR) : null;
      const frameBeforePan = renderEvidence.webgpuRoute ? await readCanvasWebGPUFrameSeq(page, CANVAS_SELECTOR) : 0;

      const cx = box!.x + box!.width / 2;
      const cy = box!.y + box!.height / 2;
      const dragDX = 120;
      const dragDY = 60;
      await page.mouse.move(cx, cy);
      await page.mouse.down();
      // Several incremental moves so pointermove deltas accumulate like a real drag.
      for (let s = 1; s <= 6; s++) {
        await page.mouse.move(cx + (dragDX * s) / 6, cy + (dragDY * s) / 6);
      }
      await page.mouse.up();

      const camAfterPan = await pollForCameraChange(page, camBefore);
      // Dragging right/down moves content with the pointer → camera origin moves
      // opposite, scaled by 1/zoom. We don't assume zoom===1; just assert the
      // sign and that it moved meaningfully.
      expect(camAfterPan, "pan should move the rendered camera").not.toBeNull();
      expect(camAfterPan!.x, "drag right should decrease camera.x (content moves right)").toBeLessThan(camBefore.x - 1);
      expect(camAfterPan!.y, "drag down should increase camera.y (Y flipped)").toBeGreaterThan(camBefore.y + 1);

      if (renderEvidence.fallback2D && before) {
        const afterPan = await canvasFingerprint2D(page, CANVAS_SELECTOR);
        expect(
          afterPan.signature !== before.signature,
          `fallback 2D painted pixels should shift after pan (before=${before.signature}, after=${afterPan.signature})`,
        ).toBe(true);
      } else {
        const frameAfterPan = await waitForCanvasWebGPUFrameAdvance(page, frameBeforePan, { selector: CANVAS_SELECTOR });
        expect(
          frameAfterPan,
          `WebGPU frame sequence should advance after pan; before=${frameBeforePan}, evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
        ).not.toBeNull();
      }

      // ----------------------------------------------------------------------
      // (b) WHEEL-TO-ZOOM (toward cursor): wheel up over the canvas and assert
      // the rendered scale (camera.z) grew. camera.z is the authoritative scale
      // the OrthoCamera2D transform applies, so a strictly-larger z proves the
      // wheel zoomed in. We ALSO require route-specific render evidence to
      // advance (the zoom actually re-rendered) and verify the world point under
      // the cursor stays pinned (the zoom-toward-cursor property), reading the
      // live transform.
      // (We deliberately do NOT assert a visible-bbox area grows: zooming in
      // pushes content past the viewport edges, so the bbox of *visible* painted
      // pixels is not monotonic with zoom — camera.z is the honest signal.)
      // ----------------------------------------------------------------------
      const camBeforeZoom = await readCamera(page);
      const zoomBefore = camBeforeZoom.z;
      const fpBeforeZoom = renderEvidence.fallback2D ? await canvasFingerprint2D(page, CANVAS_SELECTOR) : null;
      const frameBeforeZoom = renderEvidence.webgpuRoute ? await readCanvasWebGPUFrameSeq(page, CANVAS_SELECTOR) : 0;
      // World point currently under the canvas center, via the live transform.
      const worldUnderCursorBefore = await worldAtScreen(page, box!.width / 2, box!.height / 2);

      await page.mouse.move(cx, cy);
      // Negative deltaY = zoom in. Several notches to clear any per-frame easing.
      for (let n = 0; n < 8; n++) {
        await page.mouse.wheel(0, -120);
      }
      const zoomAfter = await pollForZoomChange(page, zoomBefore);
      expect(zoomAfter, "wheel should change the rendered zoom").not.toBeNull();
      expect(zoomAfter!, "wheel up should increase zoom (camera.z)").toBeGreaterThan(zoomBefore + 0.01);

      if (renderEvidence.fallback2D && fpBeforeZoom) {
        const fpAfterZoom = await canvasFingerprint2D(page, CANVAS_SELECTOR);
        expect(
          fpAfterZoom.signature !== fpBeforeZoom.signature,
          "fallback 2D zoom should re-render the painted pixels",
        ).toBe(true);
      } else {
        const frameAfterZoom = await waitForCanvasWebGPUFrameAdvance(page, frameBeforeZoom, { selector: CANVAS_SELECTOR });
        expect(
          frameAfterZoom,
          `WebGPU frame sequence should advance after zoom; before=${frameBeforeZoom}, evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
        ).not.toBeNull();
      }

      // Zoom-toward-cursor: the world point that was under the canvas center
      // before the zoom must still map back to (approximately) the canvas center.
      const worldUnderCursorAfter = await worldAtScreen(page, box!.width / 2, box!.height / 2);
      if (worldUnderCursorBefore && worldUnderCursorAfter) {
        expect(
          Math.abs(worldUnderCursorAfter.worldX - worldUnderCursorBefore.worldX) < 2 &&
            Math.abs(worldUnderCursorAfter.worldY - worldUnderCursorBefore.worldY) < 2,
          `zoom should pin the world point under the cursor ` +
            `(before=${JSON.stringify(worldUnderCursorBefore)}, after=${JSON.stringify(worldUnderCursorAfter)})`,
        ).toBe(true);
      }

      // No unexpected canvas-path console errors throughout the interaction.
      const canvasErrors = consoleErrors.filter((line) => /canvas|surface|__gosx_canvas_event|__gosx_render/i.test(line));
      expect(canvasErrors, `unexpected canvas2d console errors: ${JSON.stringify(canvasErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// readCamera returns the current rendered camera (pan x/y, zoom z) by asking
// the live render entry for the board's bundle at its CSS size.
async function readCamera(page: Page): Promise<{ x: number; y: number; z: number }> {
  return page.evaluate((selector) => {
    const w = window as unknown as { __gosx_render_canvas?: (id: string, width: number, height: number, t: number) => unknown };
    const el = document.querySelector(selector) as HTMLElement | null;
    const id = el?.getAttribute("data-gosx-surface-id") || "";
    const cssW = Math.max(1, (el as HTMLElement)?.clientWidth || 1);
    const cssH = Math.max(1, (el as HTMLElement)?.clientHeight || 1);
    const out = typeof w.__gosx_render_canvas === "function" ? w.__gosx_render_canvas(id, cssW, cssH, 0) : "";
    if (typeof out !== "string" || out[0] === "e") return { x: 0, y: 0, z: 1 };
    const bundle = JSON.parse(out) as { camera?: { x?: number; y?: number; z?: number } };
    const cam = bundle.camera || {};
    return { x: cam.x ?? 0, y: cam.y ?? 0, z: cam.z ?? 1 };
  }, CANVAS_SELECTOR);
}

// pollForCameraChange waits for the camera pan to move away from `from`,
// returning the new camera or null on timeout.
async function pollForCameraChange(page: Page, from: { x: number; y: number; z: number }): Promise<{ x: number; y: number; z: number } | null> {
  const deadline = Date.now() + 8_000;
  while (Date.now() < deadline) {
    const cam = await readCamera(page);
    if (Math.abs(cam.x - from.x) > 0.5 || Math.abs(cam.y - from.y) > 0.5) return cam;
    await page.waitForTimeout(100);
  }
  return null;
}

// pollForZoomChange waits for camera.z to move away from `from`.
async function pollForZoomChange(page: Page, from: number): Promise<number | null> {
  const deadline = Date.now() + 8_000;
  while (Date.now() < deadline) {
    const cam = await readCamera(page);
    if (Math.abs(cam.z - from) > 0.005) return cam.z;
    await page.waitForTimeout(100);
  }
  return null;
}

// pollForSelectedID waits for $surface.event.selectedID to reach `want`,
// returning the last observed value (parsed out of the getter's JSON).
async function pollForSelectedID(page: Page, want: string): Promise<string | null> {
  const deadline = Date.now() + 8_000;
  let last: string | null = null;
  while (Date.now() < deadline) {
    last = await page.evaluate(() => {
      const w = window as unknown as { __gosx_get_shared_signal?: (name: string) => unknown };
      if (typeof w.__gosx_get_shared_signal !== "function") return null;
      const raw = w.__gosx_get_shared_signal("$surface.event.selectedID");
      if (typeof raw !== "string") return null;
      // The getter returns JSON; a string value comes back quoted.
      try {
        const parsed = JSON.parse(raw);
        return typeof parsed === "string" ? parsed : String(parsed);
      } catch {
        return raw;
      }
    });
    if (last === want) return last;
    await page.waitForTimeout(100);
  }
  return last;
}

// worldAtScreen returns the world point currently under a CSS-pixel screen
// position, using the live camera and the SAME OrthoCamera2D inverse the Go
// pick path uses. Used to verify zoom-toward-cursor pins the point under the
// cursor.
async function worldAtScreen(page: Page, screenX: number, screenY: number): Promise<{ worldX: number; worldY: number } | null> {
  const cam = await readCamera(page);
  return page.evaluate(({ selector, sx, sy, camera }) => {
    const el = document.querySelector(selector) as HTMLElement | null;
    if (!el) return null;
    const cssW = Math.max(1, el.clientWidth || 1);
    const cssH = Math.max(1, el.clientHeight || 1);
    const zoom = camera.z > 0 ? camera.z : 1;
    // Inverse of the painter transform: worldX = (sx - cssW/2)/zoom + panX,
    // worldY = panY - (sy - cssH/2)/zoom.
    const worldX = (sx - cssW / 2) / zoom + camera.x;
    const worldY = camera.y - (sy - cssH / 2) / zoom;
    return { worldX, worldY };
  }, { selector: CANVAS_SELECTOR, sx: screenX, sy: screenY, camera: cam });
}

// pickTarget reads the live bundle, picks a rect, and computes its on-screen
// center using the SAME screen transform the painter uses — so the returned
// (screenX, screenY) is exactly where that rect is drawn right now.
async function pickTarget(page: Page): Promise<{ id: string; screenX: number; screenY: number } | null> {
  return page.evaluate((selector) => {
    const w = window as unknown as {
      __gosx_render_canvas?: (id: string, width: number, height: number, t: number) => unknown;
      __gosx_canvas_board_screen_transform?: (camera: unknown, cssW: number, cssH: number) => { x: (wx: number) => number; y: (wy: number) => number };
    };
    const el = document.querySelector(selector) as HTMLElement | null;
    if (!el) return null;
    const id = el.getAttribute("data-gosx-surface-id") || "";
    const cssW = Math.max(1, el.clientWidth || 1);
    const cssH = Math.max(1, el.clientHeight || 1);
    const out = typeof w.__gosx_render_canvas === "function" ? w.__gosx_render_canvas(id, cssW, cssH, 0) : "";
    if (typeof out !== "string" || out[0] === "e") return null;
    const bundle = JSON.parse(out) as {
      camera?: unknown;
      objects?: Array<{ id?: string; kind?: string; bounds?: { minX?: number; maxX?: number; minY?: number; maxY?: number } }>;
    };
    const objects = Array.isArray(bundle.objects) ? bundle.objects : [];
    // Prefer a rect whose on-screen center lands inside the visible canvas, so
    // the synthesized click actually hits it.
    const t = typeof w.__gosx_canvas_board_screen_transform === "function"
      ? w.__gosx_canvas_board_screen_transform(bundle.camera, cssW, cssH)
      : null;
    if (!t) return null;
    for (const obj of objects) {
      if (!obj || obj.kind !== "rect" || !obj.id || !obj.bounds) continue;
      const b = obj.bounds;
      const worldCX = ((b.minX ?? 0) + (b.maxX ?? 0)) / 2;
      const worldCY = ((b.minY ?? 0) + (b.maxY ?? 0)) / 2;
      const sx = t.x(worldCX);
      const sy = t.y(worldCY);
      if (sx >= 2 && sx <= cssW - 2 && sy >= 2 && sy <= cssH - 2) {
        return { id: obj.id, screenX: sx, screenY: sy };
      }
    }
    return null;
  }, CANVAS_SELECTOR);
}
