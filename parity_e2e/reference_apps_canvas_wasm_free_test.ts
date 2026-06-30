import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvasWASMFree } from "./reference_apps_harness";
import {
  formatCanvasRenderEvidence,
  waitForCanvasBoardRenderEvidence,
  WASM_FREE_CANVAS_SELECTOR,
} from "./canvas_render_evidence";

// Live-proof e2e for the WASM-FREE Canvas2D site-map board — the payload payoff.
//
// In wasm-free mode (MUDDY_CANVAS_WASM_FREE=1) the editor renders + drives the
// Canvas2D site-map board with NO gosx client WASM. The decoupling + the proof:
//
//   - The board canvas is muddy-owned: <canvas data-gosx-canvas-wasm-free="true">
//     that DELIBERATELY carries NO data-gosx-surface-kind, so the gosx client
//     bootstrap's canvas discovery
//     (`[data-gosx-surface-kind]:not([data-gosx-engine-bytecode])`) never tries
//     to WASM-hydrate it. The muddy WASM-free scripts (ported painter +
//     interaction client) are the ONLY things painting/handling it.
//   - The server ships the SAME precomputed RenderBundle inline in
//     <script data-gosx-canvas-bundle>; the client paints it via a rAF loop with
//     a JS-side camera (no __gosx_render_canvas).
//   - Selection flows DIRECTLY into the DOM board via
//     window.GoSXStudioSiteMapRuntime.setState (reachable in the LIGHT islands
//     runtime via gosx-studio's EngineRuntimeScript), so the right-rail inspector
//     / selection detail card updates identically to a DOM-board selection — with
//     no signal poll and no WASM canvas event.
//
// This file asserts, end-to-end in headless Chromium:
//   (a) PAINT — the canvas backing store has non-background pixels.
//   (b) NO FULL WASM — the full gosx-runtime.<hash>.wasm is never fetched. A
//       zero-WASM footprint is valid/preferred when the Studio JS runtimes provide
//       the required globals; if any WASM or manifest runtime path is present, it
//       must be islands-only.
//   (c) PAN — a left-drag shifts painted content (the camera moves; pixels change).
//   (d) ZOOM — a wheel gesture changes the JS camera scale.
//   (e) PICK — a click on a node updates the board's selected-node attr AND the
//       inspector's selection-detail label (DOM readback).
//   (f) MARQUEE — a shift-drag over >=2 nodes flips >=2 DOM nodes selected.
//   (g) NAV — an arrow key moves the selection to the spatial neighbor.
//
// Honesty discipline (matching the sibling canvas tests): rects + their screen
// positions are DISCOVERED from the live inline RenderBundle and the live JS
// camera, matched to DOM workspace nodes — nothing is hardcoded. The expected nav
// neighbor is computed here with the SAME cost model the client uses. If any
// interaction genuinely does not work WASM-free, the offending assertion is left
// test.fixme with the observed evidence rather than faked.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The reference-app harness rebuilds
// Muddy's ignored dist assets against the gosx/gosx-studio under test before
// booting the server.

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const NODE_SELECTED_ATTR = "data-studio-site-map-node-selected";
const BG = { r: 0x0f, g: 0x17, b: 0x20 }; // siteMapCanvasDefaultBackground

interface RectInfo {
  id: string;
  // on-screen center (CSS px relative to the canvas top-left)
  screenX: number;
  screenY: number;
  // world-space center (for nav neighbor + marquee computation)
  worldX: number;
  worldY: number;
}

test.describe("@reference-apps canvas2d site-map WASM-free", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni wasm-free: canvas paints + pans/zooms/picks/marquees/navs + drives the inspector with the FULL WASM provably absent", async ({ page, request }, testInfo) => {
    const consoleErrors: string[] = [];
    const consoleWarnings: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
      if (message.type() === "warning") consoleWarnings.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    // Record every WASM request the page makes so we can prove the full runtime
    // is never fetched. Zero requests are valid on the current JS-runtime path.
    const wasmRequests: string[] = [];
    page.on("request", (req) => {
      const url = req.url();
      if (/\.wasm(\?|$)/.test(url)) wasmRequests.push(url);
    });

    const server = await startMuddyCanvasWASMFree(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (selection sink)").toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "the muddy WASM-free canvas (no data-gosx-surface-kind) must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

      // The WASM-free client + painter must mount, and the DOM board runtime
      // (the selection sink) must be present — all WITHOUT any __gosx canvas
      // entrypoint. Any missing piece means the served bundle predates this
      // work — fail loud, do not fake.
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
        const painter = w.GoSXStudioCanvas2DPainterRuntime as { paint?: unknown } | undefined;
        const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: unknown }) | null;
        return !!painter && typeof painter.paint === "function" &&
          !!rt && typeof rt.setState === "function" &&
          document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
          !!el && !!el.GoSXStudioCanvasWasmFree &&
          el.getAttribute("data-gosx-canvas-wasm-free-bound") === "true";
      }, null, { timeout: 120_000 });

      // The WASM-free path must NOT install the WASM canvas entrypoints (those
      // live only in the full runtime). Their absence is part of the proof.
      const noWasmCanvasEntrypoints = await page.evaluate(() => {
        const w = window as unknown as Record<string, unknown>;
        return typeof w.__gosx_render_canvas !== "function" &&
          typeof w.__gosx_canvas_event !== "function" &&
          typeof w.__gosx_canvas_set_backend !== "function";
      });
      expect(
        noWasmCanvasEntrypoints,
        "the full-WASM canvas entrypoints (__gosx_render_canvas / __gosx_canvas_event / __gosx_canvas_set_backend) must be ABSENT in wasm-free mode",
      ).toBe(true);

      await canvas.scrollIntoViewIfNeeded();
      const box = await canvas.boundingBox();
      expect(box, "canvas should have a layout box").not.toBeNull();
      expect(box!.width, "canvas CSS width > 0").toBeGreaterThan(0);
      expect(box!.height, "canvas CSS height > 0").toBeGreaterThan(0);

      const renderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings, {
        selector: WASM_FREE_CANVAS_SELECTOR,
      });
      await testInfo.attach("wasm-free-static-webgpu-route-evidence.json", {
        contentType: "application/json",
        body: formatCanvasRenderEvidence(renderEvidence),
      });
      expect(
        renderEvidence.webgpuLoaded,
        `wasm-free route must load the scene3d-webgpu chunk; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);
      expect(
        renderEvidence.webgpuAPI,
        `wasm-free route must expose window.__gosx_scene3d_webgpu_api.createRenderer; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);
      expect(
        renderEvidence.webgpuRoute || renderEvidence.fallback2D,
        `wasm-free route must either publish WebGPU frame attrs or explicitly fall back to Canvas2D; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);
      if (!renderEvidence.webgpuRoute) {
        const fallbackReason = renderEvidence.hostAttrs["data-gosx-canvas-wasm-free-fallback-reason"];
        expect(
          fallbackReason,
          `Canvas2D fallback must expose a stable fallback reason; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
        ).toMatch(/\S/);
      }

      // ── (a) PAINT ───────────────────────────────────────────────────────────
      const paintedCount = await pollForPaint(page);
      expect(paintedCount, "the WASM-free board must paint non-background pixels").toBeGreaterThan(0);

      // ── (b) NO FULL WASM — zero or islands-only runtime footprint ───────────
      // Give the network a beat to settle, then inspect.
      await page.waitForTimeout(250);
      const manifestRuntime = await runtimeManifestPath(page);
      const fullWasm = wasmRequests.filter(isFullRuntimeWasm);
      const islandsWasm = wasmRequests.filter(isIslandsRuntimePath);
      const zeroWasmNoManifestRuntime = wasmRequests.length === 0 && manifestRuntime === "";
      const islandsOnlyFootprint = fullWasm.length === 0 &&
        wasmRequests.every(isIslandsRuntimePath) &&
        (manifestRuntime === "" || isIslandsRuntimePath(manifestRuntime));
      const footprintEvidence = { manifestRuntime, wasmRequests, fullWasm, islandsWasm, zeroWasmNoManifestRuntime };
      await testInfo.attach("wasm-free-low-wasm-footprint-evidence.json", {
        contentType: "application/json",
        body: JSON.stringify(footprintEvidence, null, 2),
      });
      expect(
        fullWasm,
        `the FULL gosx-runtime.<hash>.wasm must never be fetched in wasm-free mode; observed wasm requests=${JSON.stringify(wasmRequests)}`,
      ).toEqual([]);
      expect(
        zeroWasmNoManifestRuntime || islandsOnlyFootprint,
        `wasm-free mode must have either zero WASM with no manifest runtime path, or an islands-only WASM footprint; footprint=${JSON.stringify(footprintEvidence)}`,
      ).toBe(true);

      // Discover the rects the DOM board can resolve, with on-screen + world
      // centers, from the live inline bundle + the live JS camera.
      const rects = await pollForRects(page);
      expect(rects.length, `need >=2 wasm-free canvas rects resolvable to DOM nodes; got ${rects.length}`).toBeGreaterThanOrEqual(2);

      // ── (c) PAN — drag shifts painted content ────────────────────────────────
      const sig0 = await paintSignature(page);
      const cam0 = await readCamera(page);
      await page.mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2);
      await page.mouse.down();
      await page.mouse.move(box!.x + box!.width / 2 + 90, box!.y + box!.height / 2 + 60, { steps: 6 });
      await page.mouse.up();
      const cam1 = await readCamera(page);
      expect(
        cam1.x !== cam0.x || cam1.y !== cam0.y,
        `pan should move the JS camera; before=${JSON.stringify(cam0)} after=${JSON.stringify(cam1)}`,
      ).toBe(true);
      const sig1 = await pollForPaintChange(page, sig0);
      expect(
        sig1.changed,
        `pan should shift painted content (the rendered pixels should change); console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toBe(true);
      // Reset the camera to a known state for the remaining, position-sensitive
      // interactions (use the client's setCamera test hook).
      await resetCamera(page);
      await page.waitForTimeout(60);

      // ── (d) ZOOM — wheel changes the JS camera scale ─────────────────────────
      const camZ0 = (await readCamera(page)).z;
      await page.mouse.move(box!.x + box!.width / 2, box!.y + box!.height / 2);
      await page.mouse.wheel(0, -600); // scroll up = zoom in (factor > 1)
      const camZ1 = (await readCamera(page)).z;
      expect(
        camZ1,
        `wheel-up should increase the JS camera zoom; before=${camZ0} after=${camZ1}`,
      ).toBeGreaterThan(camZ0);
      await resetCamera(page);
      await page.waitForTimeout(60);

      // Re-discover rects at the reset camera so screen coords match the view.
      const rectsAtReset = await pollForRects(page);
      expect(rectsAtReset.length, "rects should resolve at the reset camera").toBeGreaterThanOrEqual(2);

      // ── (e) PICK — click a node → board + inspector update ──────────────────
      const pickTarget = rectsAtReset.find((r) => true)!;
      const expectedLabel = await labelForKey(page, pickTarget.id);
      expect(expectedLabel, `the picked node ${pickTarget.id} should carry a DOM label`).toMatch(/\S/);
      const labelBefore = await selectedLabelText(page);
      await page.mouse.move(box!.x + pickTarget.screenX, box!.y + pickTarget.screenY);
      await page.mouse.down();
      await page.mouse.up();
      const picked = await pollForSelection(page, pickTarget.id, expectedLabel!);
      expect(
        picked.boardSelected,
        `click should set board ${SELECTED_NODE_ATTR}=${pickTarget.id}; got ${JSON.stringify(picked.boardSelected)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toBe(pickTarget.id);
      expect(
        picked.label,
        `click should drive the inspector selection-detail label to ${JSON.stringify(expectedLabel)}; before=${JSON.stringify(labelBefore)}, got ${JSON.stringify(picked.label)}`,
      ).toBe(expectedLabel);

      // ── (f) MARQUEE — shift-drag over >=2 nodes selects them ────────────────
      // Re-discover rects + box AFTER the pick (selection can relayout the grid),
      // so the marquee box coordinates match the live view.
      const rectsForMarquee = await pollForRects(page);
      expect(rectsForMarquee.length, "rects should resolve before marquee").toBeGreaterThanOrEqual(2);
      await canvas.scrollIntoViewIfNeeded();
      const boxM = (await canvas.boundingBox())!;
      const pair = closestPair(rectsForMarquee);
      const targetIds = [pair.a.id, pair.b.id];
      const pad = 16;
      const minX = Math.min(pair.a.screenX, pair.b.screenX) - pad;
      const minY = Math.min(pair.a.screenY, pair.b.screenY) - pad;
      const maxX = Math.max(pair.a.screenX, pair.b.screenX) + pad;
      const maxY = Math.max(pair.a.screenY, pair.b.screenY) + pad;
      await page.keyboard.down("Shift");
      await page.mouse.move(boxM.x + minX, boxM.y + minY);
      await page.mouse.down();
      await page.mouse.move(boxM.x + (minX + maxX) / 2, boxM.y + (minY + maxY) / 2, { steps: 4 });
      await page.mouse.move(boxM.x + maxX, boxM.y + maxY, { steps: 4 });
      await page.mouse.up();
      await page.keyboard.up("Shift");
      const marqueeResult = await pollForMultiSelect(page, targetIds);
      expect(
        marqueeResult.selectedCount,
        `shift-drag marquee should select >=2 DOM workspace nodes; targets=${JSON.stringify(targetIds)}, selected=${JSON.stringify(marqueeResult.selectedKeys)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toBeGreaterThanOrEqual(2);
      for (const id of targetIds) {
        expect(
          marqueeResult.selectedKeys.includes(id),
          `marquee target ${id} should be flagged selected; selected=${JSON.stringify(marqueeResult.selectedKeys)}`,
        ).toBe(true);
      }

      // ── (g) NAV — arrow key moves the selection to the spatial neighbor ─────
      // Re-discover rects + box after the marquee for the same layout-stability
      // reason, and compute the nav plan from the live geometry.
      const rectsForNav = await pollForRects(page);
      expect(rectsForNav.length, "rects should resolve before nav").toBeGreaterThanOrEqual(2);
      await canvas.scrollIntoViewIfNeeded();
      const boxN = (await canvas.boundingBox())!;
      const navPlan = chooseNavTarget(rectsForNav);
      expect(navPlan, `need a node with a computable spatial neighbor for nav; rects=${rectsForNav.length}`).not.toBeNull();
      const { start, dir, expectedNeighbor } = navPlan!;
      const startRect = rectsForNav.find((r) => r.id === start)!;
      // Seed selection on `start` via a click; the click also focuses the canvas
      // so the keydown nav targets it.
      await page.mouse.move(boxN.x + startRect.screenX, boxN.y + startRect.screenY);
      await page.mouse.down();
      await page.mouse.up();
      const seeded = await pollForSelectedNode(page, start);
      expect(seeded, `seeding selection on ${start} via click should set ${SELECTED_NODE_ATTR}; got ${JSON.stringify(seeded)}`).toBe(start);
      await canvas.press(dir);
      const moved = await pollForSelectedNode(page, expectedNeighbor);
      expect(
        moved,
        `arrow-key ${dir} from ${start} should move ${SELECTED_NODE_ATTR} to the spatial neighbor ${expectedNeighbor}; got ${JSON.stringify(moved)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
      ).toBe(expectedNeighbor);

      // No WASM-free client console errors should have leaked.
      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|GoSXStudioSiteMapRuntime/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// readInlineBundleAndCamera reads the inline RenderBundle JSON + the live JS
// camera from the WASM-free client test hook, returning both so the caller can
// build the SAME world→screen transform the painter applies.
async function readBundleRects(page: Page): Promise<RectInfo[]> {
  return page.evaluate(({ canvasSel, boardSel }) => {
    const el = document.querySelector(canvasSel) as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { camera: () => { x: number; y: number; z: number }; bundle: () => unknown } }) | null;
    if (!el || !el.GoSXStudioCanvasWasmFree) return [];
    const hook = el.GoSXStudioCanvasWasmFree;
    const bundle = hook.bundle() as { objects?: Array<{ id?: string; kind?: string; pickable?: boolean; bounds?: { minX?: number; maxX?: number; minY?: number; maxY?: number } }> } | null;
    if (!bundle) return [];
    const cam = hook.camera();
    const cssW = Math.max(1, el.clientWidth || 1);
    const cssH = Math.max(1, el.clientHeight || 1);
    const zoom = cam.z > 0 ? cam.z : 1;
    // OrthoCamera2D forward transform (Y flipped) — identical to the painter.
    const sx = (wx: number) => (wx - cam.x) * zoom + cssW / 2;
    const sy = (wy: number) => cssH / 2 - (wy - cam.y) * zoom;
    const board = document.querySelector(boardSel);
    const resolvable = (key: string): boolean => {
      if (!board) return false;
      const node = Array.prototype.slice
        .call(board.querySelectorAll("[data-studio-site-map-workspace-node]"))
        .find((n) => (n as Element).getAttribute("data-studio-site-map-workspace-node") === key) as Element | undefined;
      return !!node;
    };
    const objects = Array.isArray(bundle.objects) ? bundle.objects : [];
    const out: RectInfo[] = [];
    for (const obj of objects) {
      if (!obj || obj.kind !== "rect" || !obj.id || obj.pickable === false || !obj.bounds) continue;
      if (!resolvable(obj.id)) continue;
      const b = obj.bounds;
      const worldX = ((b.minX ?? 0) + (b.maxX ?? 0)) / 2;
      const worldY = ((b.minY ?? 0) + (b.maxY ?? 0)) / 2;
      const screenX = sx(worldX);
      const screenY = sy(worldY);
      if (screenX >= 6 && screenX <= cssW - 6 && screenY >= 6 && screenY <= cssH - 6) {
        out.push({ id: obj.id, screenX, screenY, worldX, worldY });
      }
    }
    return out;
  }, { canvasSel: CANVAS_SELECTOR, boardSel: BOARD_SELECTOR });
}

async function pollForRects(page: Page): Promise<RectInfo[]> {
  const deadline = Date.now() + 30_000;
  let last: RectInfo[] = [];
  while (Date.now() < deadline) {
    last = await readBundleRects(page);
    if (last.length >= 2) return last;
    await page.waitForTimeout(250);
  }
  return last;
}

async function readCamera(page: Page): Promise<{ x: number; y: number; z: number }> {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel) as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { camera: () => { x: number; y: number; z: number } } }) | null;
    if (!el || !el.GoSXStudioCanvasWasmFree) return { x: NaN, y: NaN, z: NaN };
    return el.GoSXStudioCanvasWasmFree.camera();
  }, CANVAS_SELECTOR);
}

async function resetCamera(page: Page): Promise<void> {
  await page.evaluate((sel) => {
    const el = document.querySelector(sel) as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { setCamera: (x: number, y: number, z: number) => void } }) | null;
    el?.GoSXStudioCanvasWasmFree?.setCamera(0, 0, 1);
  }, CANVAS_SELECTOR);
}

// paintSignature samples a coarse fingerprint of the painted backing store so we
// can detect that a pan actually changed the rendered pixels.
async function paintSignature(page: Page): Promise<number[]> {
  return page.evaluate((selector) => {
    const canvas = document.querySelector(selector);
    if (!(canvas instanceof HTMLCanvasElement)) return [];
    const ctx = canvas.getContext("2d");
    if (!ctx) return [];
    const wpx = canvas.width, hpx = canvas.height;
    if (!wpx || !hpx) return [];
    let data: Uint8ClampedArray;
    try { data = ctx.getImageData(0, 0, wpx, hpx).data; } catch { return []; }
    // Bucket non-background pixel counts into a 4x4 grid → a 16-number signature.
    const grid = new Array(16).fill(0);
    const cellW = wpx / 4, cellH = hpx / 4;
    const tol = 10;
    const stride = 4 * Math.max(1, Math.floor((wpx * hpx) / 40000));
    const bg = { r: 0x0f, g: 0x17, b: 0x20 };
    for (let i = 0; i < data.length; i += stride) {
      const px = (i / 4) % wpx;
      const py = Math.floor((i / 4) / wpx);
      const r = data[i], g = data[i + 1], b = data[i + 2], a = data[i + 3];
      if (a === 0) continue;
      if (Math.abs(r - bg.r) > tol || Math.abs(g - bg.g) > tol || Math.abs(b - bg.b) > tol) {
        const cell = Math.min(3, Math.floor(px / cellW)) + 4 * Math.min(3, Math.floor(py / cellH));
        grid[cell]++;
      }
    }
    return grid;
  }, CANVAS_SELECTOR);
}

async function pollForPaintChange(page: Page, before: number[]): Promise<{ changed: boolean; after: number[] }> {
  const deadline = Date.now() + 5_000;
  let after = before;
  while (Date.now() < deadline) {
    after = await paintSignature(page);
    const changed = after.length === before.length && after.some((v, i) => Math.abs(v - before[i]) > 2);
    if (changed) return { changed: true, after };
    await page.waitForTimeout(120);
  }
  return { changed: false, after };
}

function closestPair(rects: RectInfo[]): { a: RectInfo; b: RectInfo } {
  let best = { a: rects[0], b: rects[1] };
  let bestD = Infinity;
  for (let i = 0; i < rects.length; i++) {
    for (let j = i + 1; j < rects.length; j++) {
      const dx = rects[i].screenX - rects[j].screenX;
      const dy = rects[i].screenY - rects[j].screenY;
      const d = dx * dx + dy * dy;
      if (d < bestD) { bestD = d; best = { a: rects[i], b: rects[j] }; }
    }
  }
  return best;
}

// chooseNavTarget mirrors the client's navFrom cost model (cost = primary +
// 2*perpendicular, half-plane filtered; world +Y is UP, so ArrowUp => larger
// world-Y). Returns the first start node with an unambiguous neighbor.
function chooseNavTarget(rects: RectInfo[]): { start: string; dir: string; expectedNeighbor: string } | null {
  const dirs: Array<{ key: string; pick: (dx: number, dy: number) => { primary: number; perp: number } | null }> = [
    { key: "ArrowRight", pick: (dx, dy) => (dx > 0 ? { primary: dx, perp: Math.abs(dy) } : null) },
    { key: "ArrowLeft", pick: (dx, dy) => (dx < 0 ? { primary: -dx, perp: Math.abs(dy) } : null) },
    { key: "ArrowUp", pick: (dx, dy) => (dy > 0 ? { primary: dy, perp: Math.abs(dx) } : null) },
    { key: "ArrowDown", pick: (dx, dy) => (dy < 0 ? { primary: -dy, perp: Math.abs(dx) } : null) },
  ];
  for (const from of rects) {
    for (const d of dirs) {
      let best: { id: string; cost: number } | null = null;
      let secondCost = Infinity;
      for (const to of rects) {
        if (to.id === from.id) continue;
        const dx = to.worldX - from.worldX;
        const dy = to.worldY - from.worldY;
        const r = d.pick(dx, dy);
        if (!r) continue;
        const cost = r.primary + r.perp * 2;
        if (!best || cost < best.cost) { secondCost = best ? best.cost : secondCost; best = { id: to.id, cost }; }
        else if (cost < secondCost) { secondCost = cost; }
      }
      if (best && secondCost - best.cost > 8) {
        return { start: from.id, dir: d.key, expectedNeighbor: best.id };
      }
    }
  }
  return null;
}

async function labelForKey(page: Page, key: string): Promise<string | null> {
  return page.evaluate(({ board, k }) => {
    const root = document.querySelector(board);
    if (!root) return null;
    const node = Array.prototype.slice
      .call(root.querySelectorAll("[data-studio-site-map-workspace-node]"))
      .find((n) => (n as Element).getAttribute("data-studio-site-map-workspace-node") === k) as Element | undefined;
    return node ? (node.getAttribute("data-studio-site-map-node-label") || "") : null;
  }, { board: BOARD_SELECTOR, k: key });
}

async function selectedLabelText(page: Page): Promise<string | null> {
  return page.evaluate((board) => {
    const root = document.querySelector(board);
    if (!root) return null;
    const el = root.querySelector("[data-studio-site-map-selected-label]");
    return el ? (el.textContent || "").trim() : null;
  }, BOARD_SELECTOR);
}

async function pollForSelection(page: Page, wantKey: string, wantLabel: string): Promise<{ boardSelected: string | null; label: string | null }> {
  const deadline = Date.now() + 10_000;
  let last: { boardSelected: string | null; label: string | null } = { boardSelected: null, label: null };
  while (Date.now() < deadline) {
    last = await page.evaluate(({ board, attr }) => {
      const root = document.querySelector(board);
      const labelEl = root ? root.querySelector("[data-studio-site-map-selected-label]") : null;
      return {
        boardSelected: root ? root.getAttribute(attr) : null,
        label: labelEl ? (labelEl.textContent || "").trim() : null,
      };
    }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR });
    if (last.boardSelected === wantKey && last.label === wantLabel) return last;
    await page.waitForTimeout(120);
  }
  return last;
}

async function pollForMultiSelect(page: Page, wantIds: string[]): Promise<{ selectedCount: number; selectedKeys: string[] }> {
  const deadline = Date.now() + 10_000;
  let last = { selectedCount: 0, selectedKeys: [] as string[] };
  while (Date.now() < deadline) {
    last = await page.evaluate(({ board, attr }) => {
      const root = document.querySelector(board);
      const keys: string[] = [];
      if (root) {
        Array.prototype.slice
          .call(root.querySelectorAll("[data-studio-site-map-workspace-node]"))
          .forEach((n) => {
            const el = n as Element;
            if (el.getAttribute(attr) === "true") keys.push(el.getAttribute("data-studio-site-map-workspace-node") || "");
          });
      }
      return { selectedCount: keys.length, selectedKeys: keys };
    }, { board: BOARD_SELECTOR, attr: NODE_SELECTED_ATTR });
    if (last.selectedCount >= 2 && wantIds.every((id) => last.selectedKeys.includes(id))) return last;
    await page.waitForTimeout(120);
  }
  return last;
}

async function pollForSelectedNode(page: Page, want: string): Promise<string | null> {
  const deadline = Date.now() + 10_000;
  let last: string | null = null;
  while (Date.now() < deadline) {
    last = await page.evaluate(({ board, attr }) => {
      const root = document.querySelector(board);
      return root ? root.getAttribute(attr) : null;
    }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR });
    if (last === want) return last;
    await page.waitForTimeout(120);
  }
  return last;
}

// pollForPaint waits until the canvas backing store has at least one
// non-background pixel — i.e. the rAF loop actually drew the board content.
async function pollForPaint(page: Page): Promise<number> {
  const deadline = Date.now() + 60_000;
  let last = 0;
  while (Date.now() < deadline) {
    last = await page.evaluate(({ selector, bg }) => {
      const canvas = document.querySelector(selector);
      if (!(canvas instanceof HTMLCanvasElement)) return 0;
      const ctx = canvas.getContext("2d");
      if (!ctx) return 0;
      const wpx = canvas.width, hpx = canvas.height;
      if (wpx === 0 || hpx === 0) return 0;
      let data: Uint8ClampedArray;
      try { data = ctx.getImageData(0, 0, wpx, hpx).data; } catch { return 0; }
      const tol = 10;
      let count = 0;
      const stride = 4 * Math.max(1, Math.floor((wpx * hpx) / 50000));
      for (let i = 0; i < data.length; i += stride) {
        const r = data[i], g = data[i + 1], b = data[i + 2], a = data[i + 3];
        if (a === 0) continue;
        if (Math.abs(r - bg.r) > tol || Math.abs(g - bg.g) > tol || Math.abs(b - bg.b) > tol) count++;
      }
      return count;
    }, { selector: CANVAS_SELECTOR, bg: BG });
    if (last > 0) return last;
    await page.waitForTimeout(250);
  }
  return last;
}

async function runtimeManifestPath(page: Page): Promise<string> {
  return page.evaluate(() => {
    const el = document.getElementById("gosx-manifest");
    if (!el) return "";
    try {
      const m = JSON.parse(el.textContent || "") as { runtime?: { path?: string } };
      return m?.runtime?.path || "";
    } catch {
      return "";
    }
  });
}

function isFullRuntimeWasm(url: string): boolean {
  return /gosx-runtime\.[0-9a-f]+\.wasm/i.test(url) || /\/gosx\/runtime\.wasm/i.test(url);
}

function isIslandsRuntimePath(url: string): boolean {
  return /gosx-runtime-islands/i.test(url) || /runtime-islands\.wasm/i.test(url);
}
