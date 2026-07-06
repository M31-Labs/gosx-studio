import { expect, test, type Page } from "@playwright/test";
import { gotoEditor, startMuddyCanvasFullWASM } from "./reference_apps_harness";
import {
  formatCanvasRenderEvidence,
  waitForCanvasBoardRenderEvidence,
} from "./canvas_render_evidence";

// Live-proof e2e for Slice 3 — Canvas2D marquee multi-select + keyboard
// node-navigation, reaching interaction parity with the DOM site-map board.
//
// In canvas-default mode (MUDDY_SITEMAP_CANVAS=canvas-default) the Canvas2D
// board is the SOLE site-map graph visual; the DOM board element + its
// sitemapruntime binding stay live (hidden graph sub-tree) so the muddy
// selection bridge can drive its selection state. This file asserts, end-to-end
// in headless Chromium:
//
//   (a) MARQUEE: a shift+left-drag box over >=2 canvas nodes selects them — the
//       matching DOM workspace nodes flip data-studio-site-map-node-selected to
//       "true" (the multi-select highlight the DOM board renders for
//       selectedNodes), driven by the canvas marquee → $surface.event.selectedIDs
//       → muddy bridge → setState({selectedNodes}) seam.
//
//   (b) KEYBOARD NAV: from a selected node, an arrow key moves
//       data-studio-site-map-selected-node to the SPATIAL neighbor in that
//       direction — the node the adapter's NavFrom (mirroring the DOM board's
//       nearestNodeKey) picks, computed independently here from the live
//       RenderBundle so nothing is hardcoded.
//
// Honesty discipline: target nodes + their screen positions are DISCOVERED from
// the live RenderBundle (window.__gosx_render_canvas) and matched to DOM
// workspace nodes; the expected nav neighbor is computed here with the SAME cost
// model. If marquee/nav genuinely does not work live, the offending assertion is
// left test.fixme with the observed evidence rather than faked.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const CANVAS_SELECTOR = "canvas[data-gosx-surface-kind='canvas2d']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const NODE_SELECTED_ATTR = "data-studio-site-map-node-selected";

interface RectInfo {
  id: string;
  // on-screen center (CSS px relative to the canvas top-left)
  screenX: number;
  screenY: number;
  // world-space center (for nav neighbor computation)
  worldX: number;
  worldY: number;
}

test.describe("@reference-apps canvas2d marquee + keyboard nav", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni canvas-default: shift-drag marquee multi-selects + arrow-key nav moves to the spatial neighbor", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    const consoleWarnings: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
      if (message.type() === "warning") consoleWarnings.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvasFullWASM(request);
    try {
      await gotoEditor(page, server.baseURL);

      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (bridge target)").toBeAttached();
      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "canvas2d surface must be emitted in canvas-default mode").toBeAttached({ timeout: 30_000 });
      await expect(canvas, "canvas-default CanvasBoard should request the WebGPU backend").toHaveAttribute("data-gosx-canvas-backend", "webgpu");

      // Runtime + canvas entrypoints + the new marquee/nav event surface + the
      // DOM board runtime (bridge sink) + the muddy bridge must all be present.
      await page.waitForFunction(() => {
        const w = window as unknown as { __gosx?: { ready?: boolean } };
        return w.__gosx?.ready === true ||
          document.documentElement.getAttribute("data-gosx-runtime-ready") === "true";
      }, null, { timeout: 120_000 });
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
        return typeof w.__gosx_render_canvas === "function" &&
          typeof w.__gosx_canvas_board_screen_transform === "function" &&
          typeof w.__gosx_canvas_event === "function" &&
          typeof w.__gosx_get_shared_signal === "function" &&
          !!rt && typeof rt.setState === "function" &&
          document.documentElement.getAttribute("data-gosx-studio-canvas-selection-bridge-bound") === "true";
      }, null, { timeout: 120_000 });

      const box = await canvas.boundingBox();
      expect(box, "canvas should have a layout box").not.toBeNull();

      const renderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings);
      expect(
        renderEvidence.webgpuRoute || renderEvidence.fallback2D,
        `the canvas board must render before interaction; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);

      // Discover the rects the DOM board can resolve, with on-screen + world
      // centers, from the live RenderBundle. Need >=2 fully on-screen to marquee.
      const rects = await pollForRects(page);
      expect(rects.length, `need >=2 canvas rects resolvable to DOM nodes for a marquee; got ${rects.length}`).toBeGreaterThanOrEqual(2);

      // ── (a) MARQUEE: shift-drag a box that covers >=2 nodes ─────────────────
      // Pick the two closest-together rects so a compact box covers exactly them
      // (and ideally not the whole graph), then drag a marquee enclosing both.
      const pair = closestPair(rects);
      const targetIds = [pair.a.id, pair.b.id];
      const pad = 14; // CSS px padding so the box comfortably encloses the nodes
      const minX = Math.min(pair.a.screenX, pair.b.screenX) - pad;
      const minY = Math.min(pair.a.screenY, pair.b.screenY) - pad;
      const maxX = Math.max(pair.a.screenX, pair.b.screenX) + pad;
      const maxY = Math.max(pair.a.screenY, pair.b.screenY) + pad;

      // Shift+left-drag: hold Shift, press at one corner, move to the opposite
      // corner (with intermediate steps so the slop threshold is crossed), then
      // release. The bootstrap canvas2d branch sends CANVAS_EVENT_MARQUEE.
      await page.keyboard.down("Shift");
      await page.mouse.move(box!.x + minX, box!.y + minY);
      await page.mouse.down();
      await page.mouse.move(box!.x + (minX + maxX) / 2, box!.y + (minY + maxY) / 2, { steps: 4 });
      await page.mouse.move(box!.x + maxX, box!.y + maxY, { steps: 4 });
      await page.mouse.up();
      await page.keyboard.up("Shift");

      const marqueeResult = await pollForMultiSelect(page, targetIds);
      expect(
        marqueeResult.selectedCount,
        `shift-drag marquee should select >=2 DOM workspace nodes (${NODE_SELECTED_ATTR}="true"); ` +
          `targets=${JSON.stringify(targetIds)}, selected=${JSON.stringify(marqueeResult.selectedKeys)}, ` +
          `signalIDs=${JSON.stringify(marqueeResult.signalIDs)}, console=${JSON.stringify(consoleErrors.slice(-8))}`,
      ).toBeGreaterThanOrEqual(2);
      // The specific pair we boxed must be among the selected.
      for (const id of targetIds) {
        expect(
          marqueeResult.selectedKeys.includes(id),
          `marquee target ${id} should be flagged selected; selected=${JSON.stringify(marqueeResult.selectedKeys)}`,
        ).toBe(true);
      }

      // ── (b) KEYBOARD NAV: from a selected node, arrow → spatial neighbor ─────
      // Choose a start node that HAS a neighbor in some direction, compute the
      // expected neighbor with the adapter's cost model, press that arrow, and
      // assert the board's selected-node moves there.
      const navPlan = chooseNavTarget(rects);
      expect(navPlan, `need a node with a computable spatial neighbor for nav; rects=${rects.length}`).not.toBeNull();
      const { start, dir, expectedNeighbor } = navPlan!;

      // Seed the single-selection on `start` via a canvas click (the pick path),
      // and focus the canvas so the keydown nav applies to the board surface.
      const startRect = rects.find((r) => r.id === start)!;
      await page.mouse.move(box!.x + startRect.screenX, box!.y + startRect.screenY);
      await page.mouse.down();
      await page.mouse.up();
      const seeded = await pollForSelectedNode(page, start);
      expect(
        seeded,
        `seeding selection on ${start} via canvas click should set ${SELECTED_NODE_ATTR}; got ${JSON.stringify(seeded)}`,
      ).toBe(start);

      // Press the arrow. The bootstrap sends CANVAS_EVENT_NAV; the bridge writes
      // selectedID; the muddy bridge mirrors it into selectedNode.
      await page.keyboard.press(dir);

      const moved = await pollForSelectedNode(page, expectedNeighbor);
      expect(
        moved,
        `arrow-key ${dir} from ${start} should move ${SELECTED_NODE_ATTR} to the spatial neighbor ${expectedNeighbor}; ` +
          `got ${JSON.stringify(moved)}, signal=${JSON.stringify(await readSelectedSignal(page))}, ` +
          `console=${JSON.stringify(consoleErrors.slice(-8))}`,
      ).toBe(expectedNeighbor);
    } finally {
      await server.stop();
    }
  });
});

// pollForRects reads the live RenderBundle and returns every rect whose id
// matches a DOM workspace node, with both on-screen (CSS px relative to canvas)
// and world-space centers. Filters to rects whose center is comfortably inside
// the visible canvas (so a marquee/click lands on real pixels).
async function pollForRects(page: Page): Promise<RectInfo[]> {
  const deadline = Date.now() + 30_000;
  let last: RectInfo[] = [];
  while (Date.now() < deadline) {
    last = await page.evaluate(({ canvasSel, boardSel }) => {
      const w = window as unknown as {
        __gosx_render_canvas?: (id: string, width: number, height: number, t: number) => unknown;
        __gosx_canvas_board_screen_transform?: (camera: unknown, cssW: number, cssH: number) => { x: (wx: number) => number; y: (wy: number) => number };
      };
      const el = document.querySelector(canvasSel) as HTMLElement | null;
      if (!el) return [];
      const id = el.getAttribute("data-gosx-surface-id") || "";
      const cssW = Math.max(1, el.clientWidth || 1);
      const cssH = Math.max(1, el.clientHeight || 1);
      const out = typeof w.__gosx_render_canvas === "function" ? w.__gosx_render_canvas(id, cssW, cssH, 0) : "";
      if (typeof out !== "string" || out[0] === "e") return [];
      let bundle: {
        camera?: unknown;
        objects?: Array<{ id?: string; kind?: string; bounds?: { minX?: number; maxX?: number; minY?: number; maxY?: number } }>;
      };
      try { bundle = JSON.parse(out); } catch { return []; }
      const objects = Array.isArray(bundle.objects) ? bundle.objects : [];
      const t = typeof w.__gosx_canvas_board_screen_transform === "function"
        ? w.__gosx_canvas_board_screen_transform(bundle.camera, cssW, cssH)
        : null;
      if (!t) return [];
      const board = document.querySelector(boardSel);
      const resolvable = (key: string): boolean => {
        if (!board) return false;
        const node = Array.prototype.slice
          .call(board.querySelectorAll("[data-studio-site-map-workspace-node]"))
          .find((n) => (n as Element).getAttribute("data-studio-site-map-workspace-node") === key) as Element | undefined;
        return !!node;
      };
      const result: RectInfo[] = [];
      for (const obj of objects) {
        if (!obj || obj.kind !== "rect" || !obj.id || !obj.bounds) continue;
        if (!resolvable(obj.id)) continue;
        const b = obj.bounds;
        const worldX = ((b.minX ?? 0) + (b.maxX ?? 0)) / 2;
        const worldY = ((b.minY ?? 0) + (b.maxY ?? 0)) / 2;
        const sx = t.x(worldX);
        const sy = t.y(worldY);
        if (sx >= 6 && sx <= cssW - 6 && sy >= 6 && sy <= cssH - 6) {
          result.push({ id: obj.id, screenX: sx, screenY: sy, worldX, worldY });
        }
      }
      return result;
    }, { canvasSel: CANVAS_SELECTOR, boardSel: BOARD_SELECTOR });
    if (last.length >= 2) return last;
    await page.waitForTimeout(250);
  }
  return last;
}

// closestPair returns the two rects with the smallest on-screen separation, so
// the marquee box is compact (less likely to also clip a third node — though
// clipping more is fine, the assertion only requires >=2 incl. the pair).
function closestPair(rects: RectInfo[]): { a: RectInfo; b: RectInfo } {
  let best = { a: rects[0], b: rects[1] };
  let bestD = Infinity;
  for (let i = 0; i < rects.length; i++) {
    for (let j = i + 1; j < rects.length; j++) {
      const dx = rects[i].screenX - rects[j].screenX;
      const dy = rects[i].screenY - rects[j].screenY;
      const d = dx * dx + dy * dy;
      if (d < bestD) {
        bestD = d;
        best = { a: rects[i], b: rects[j] };
      }
    }
  }
  return best;
}

// chooseNavTarget finds a (start, direction, expectedNeighbor) triple using the
// SAME cost model as the adapter's NavFrom / the DOM board's nearestNodeKey:
// cost = primary-axis distance + 2x perpendicular, half-plane filtered. World
// +Y is UP (the canvas Y flip), so "ArrowUp" => larger world-Y. Returns the
// first start node that has a unique, unambiguous neighbor in some direction.
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
        if (!best || cost < best.cost) {
          secondCost = best ? best.cost : secondCost;
          best = { id: to.id, cost };
        } else if (cost < secondCost) {
          secondCost = cost;
        }
      }
      // Require an unambiguous winner (clear margin) so a flaky tie can't make
      // the test assert the wrong neighbor.
      if (best && secondCost - best.cost > 8) {
        return { start: from.id, dir: d.key, expectedNeighbor: best.id };
      }
    }
  }
  return null;
}

// pollForMultiSelect waits until the DOM board flips >=2 workspace nodes to
// selected, returning the count + the selected keys + the raw signal for diag.
async function pollForMultiSelect(page: Page, wantIds: string[]): Promise<{ selectedCount: number; selectedKeys: string[]; signalIDs: string | null }> {
  const deadline = Date.now() + 10_000;
  let last = { selectedCount: 0, selectedKeys: [] as string[], signalIDs: null as string | null };
  while (Date.now() < deadline) {
    last = await page.evaluate(({ board, attr }) => {
      const root = document.querySelector(board);
      const get = (window as unknown as { __gosx_get_shared_signal?: (n: string) => unknown }).__gosx_get_shared_signal;
      let signalIDs: string | null = null;
      if (typeof get === "function") {
        const raw = get("$surface.event.selectedIDs");
        if (typeof raw === "string") {
          try { const p = JSON.parse(raw); signalIDs = typeof p === "string" ? p : String(p); } catch { signalIDs = raw; }
        }
      }
      const keys: string[] = [];
      if (root) {
        Array.prototype.slice
          .call(root.querySelectorAll("[data-studio-site-map-workspace-node]"))
          .forEach((n) => {
            const el = n as Element;
            if (el.getAttribute(attr) === "true") {
              keys.push(el.getAttribute("data-studio-site-map-workspace-node") || "");
            }
          });
      }
      return { selectedCount: keys.length, selectedKeys: keys, signalIDs };
    }, { board: BOARD_SELECTOR, attr: NODE_SELECTED_ATTR });
    if (last.selectedCount >= 2 && wantIds.every((id) => last.selectedKeys.includes(id))) return last;
    await page.waitForTimeout(120);
  }
  return last;
}

// pollForSelectedNode waits until the board root's selected-node attr equals the
// wanted key, returning the observed value.
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

async function readSelectedSignal(page: Page): Promise<string | null> {
  return page.evaluate(() => {
    const get = (window as unknown as { __gosx_get_shared_signal?: (n: string) => unknown }).__gosx_get_shared_signal;
    if (typeof get !== "function") return null;
    const raw = get("$surface.event.selectedID");
    if (typeof raw !== "string") return null;
    try { const p = JSON.parse(raw); return typeof p === "string" ? p : String(p); } catch { return raw; }
  });
}
