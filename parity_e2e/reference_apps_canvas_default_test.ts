import { expect, test, type Page } from "@playwright/test";
import {
  applyCompositionIntentInPlace,
  gotoEditor,
  revealModeIfPresent,
  startMuddyCanvas,
  startMuddyCanvasDefault,
} from "./reference_apps_harness";
import { DEFAULT_CANVAS_SELECTOR, WASM_FREE_CANVAS_SELECTOR } from "./canvas_render_evidence";

// Live-proof e2e for the no-signal Muddy/Noni editor canvas default.
//
// STALE-PREMISE FIX (defect #2): this file previously asserted that, with no
// Muddy canvas env/query override, the production default was the low-WASM
// static WebGPU/Canvas2D-fallback ("wasm-free") path. That premise is stale.
// Per M1 slice 4 (muddy-noni-commerce app/admin/editor/page.server.go,
// editorSiteMapCanvasMode), the no-signal default is now "canvas-default": the
// REAL gosx CanvasBoard (the runtime/engine-WASM, WebGPU-backed site-map engine) is the
// SOLE site-map graph visual. The DOM board's graph sub-tree
// (.studio-site-map-workspace__surface) stays in the markup as the
// selection/authoring sink but is hidden via gated CSS, since the canvas is the
// sole graph. "wasm-free" is reached ONLY via an explicit opt-in
// (MUDDY_CANVAS_WASM_FREE=1) — it is never the no-signal default.
//
// This file now asserts the canvas-default reality: a real CanvasBoard surface
// (data-gosx-surface-kind="canvas2d", WebGPU backend, the actual
// gosx-runtime.<hash>.wasm or gosx-runtime-engine.<hash>.wasm fetched — proving
// a real runtime artifact was selected, not merely the right DOM attribute) with
// the wasm-free marker ABSENT,
// plus the preserved editing basics (a canvas pick drives the inspector,
// create-page, add-component) and a parity check against explicit co-render
// mode (MUDDY_SITEMAP_CANVAS=1).
//
// Honesty discipline matches the sibling canvas2d files: nothing is
// hardcoded — the pick target is discovered from the live RenderBundle
// (window.__gosx_render_canvas) and the expected label is read from the
// matching DOM workspace node.
//
// NOTE: this file deliberately does NOT gate on full 2D/WebGPU paint-pixel
// evidence (canvas_render_evidence.ts's waitForCanvasBoardRenderEvidence). That
// assertion is where the independently-tracked defect #1 (WASM CanvasBoard
// stuck at a 1x1 backing store when hydrated while hidden behind "Advanced")
// lives — out of scope here and being fixed separately. This file's contract is
// the STRUCTURAL canvas-default reality (surface identity/attributes + real
// WASM footprint + non-paint interaction), not paint verification.

const CANVAS_SELECTOR = DEFAULT_CANVAS_SELECTOR;
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const GRAPH_SURFACE_SELECTOR = `${BOARD_SELECTOR} .studio-site-map-workspace__surface`;
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const FORMS_SELECTOR = "[data-studio-composition-intent-forms='true']";
const NEW_NODE_KEY = "component:home:hero-copy-2";

test.describe("@reference-apps canvas-default (no-signal) canvas", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni no-env default uses the real runtime-WASM CanvasBoard and preserves editing basics", async ({ page, request }, testInfo) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const wasmRequests: string[] = [];
    page.on("request", (req) => {
      const url = req.url();
      if (/\.wasm(\?|$)/.test(url)) wasmRequests.push(url);
    });

    const server = await startMuddyCanvasDefault(request);
    try {
      await gotoEditor(page, server.baseURL);
      // The legacy site-map/canvas board under test here now lives inside the
      // "Advanced" mode panel (studio-pagecanvas-handoff moved it there once a
      // PageCanvas surface is present); reveal it before touching the board.
      await revealModeIfPresent(page, "advanced");

      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup").toBeAttached();
      await expect(page.locator(FORMS_SELECTOR), "managed authoring forms must stay present").toBeAttached();

      // ── canvas-default reality: the REAL runtime-WASM CanvasBoard, not wasm-free ──
      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "no-env default must emit the real gosx CanvasBoard surface").toBeAttached({ timeout: 30_000 });
      await expect(canvas, "no-env default canvas must carry the canvas2d surface-kind").toHaveAttribute("data-gosx-surface-kind", "canvas2d");
      await expect(canvas, "no-env default CanvasBoard should request the WebGPU backend").toHaveAttribute("data-gosx-canvas-backend", "webgpu");
      await expect(canvas, "no-env default canvas should be the CanvasBoard engine component").toHaveAttribute("data-gosx-engine-component", "CanvasBoard");
      await expect(
        page.locator(WASM_FREE_CANVAS_SELECTOR),
        "no-env default must NOT emit the wasm-free marker (wasm-free is opt-in only via MUDDY_CANVAS_WASM_FREE)",
      ).toHaveCount(0);

      // Runtime ready, then the full-build canvas painter + selection-bridge
      // globals installed (mirrors reference_apps_canvas_selection_test.ts —
      // the selection bridge mounts for canvas-default same as co-render,
      // per muddy's TestEditorCanvasSelectionBridgeMountsWhenCanvasEnabled).
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
          typeof w.__gosx_paint_canvas_bundle === "function" &&
          typeof w.__gosx_canvas_event === "function" &&
          !!rt && typeof rt.setState === "function" &&
          document.documentElement.getAttribute("data-gosx-studio-canvas-selection-bridge-bound") === "true";
      }, null, { timeout: 120_000 });

      const noWASMFreeGlobals = await page.evaluate(() => {
        const w = window as unknown as Record<string, unknown>;
        return typeof w.GoSXStudioCanvas2DPainterRuntime === "undefined" &&
          document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") !== "true";
      });
      expect(noWASMFreeGlobals, "no-env default must not expose the wasm-free client globals").toBe(true);

      const graphSurface = page.locator(GRAPH_SURFACE_SELECTOR).first();
      await expect(graphSurface, "DOM graph sub-tree still exists").toBeAttached();
      await expect(graphSurface, "DOM graph sub-tree is hidden because the canvas is the sole graph").toBeHidden();

      const box = await canvas.boundingBox();
      expect(box, "default canvas should have a layout box").not.toBeNull();
      expect(box!.width, "canvas CSS width > 0").toBeGreaterThan(0);
      expect(box!.height, "canvas CSS height > 0").toBeGreaterThan(0);

      // ── canvas pick drives the right-rail inspector (editing basics) ──────────
      const target = await pickTargetWithLabel(page);
      expect(
        target,
        "the live RenderBundle should expose a rect whose key matches a DOM workspace node",
      ).not.toBeNull();
      const picked = await clickAndPollForSelection(page, box!, target!);
      expect(picked.boardSelected, `click should set board ${SELECTED_NODE_ATTR}=${target!.id}`).toBe(target!.id);
      expect(picked.label, `click should drive inspector label to ${JSON.stringify(target!.expectedLabel)}`).toBe(target!.expectedLabel);

      // ── real WASM footprint: prove a genuine full-runtime/engine artifact was fetched ──
      await page.waitForTimeout(250);
      const runtimeWasm = wasmRequests.filter(isRealRuntimeWasm);
      await testInfo.attach("canvas-default-wasm-footprint-evidence.json", {
        contentType: "application/json",
        body: JSON.stringify({ wasmRequests, runtimeWasm }, null, 2),
      });
      expect(
        runtimeWasm.length,
        `the no-env default must genuinely fetch a gosx-runtime.<hash>.wasm or gosx-runtime-engine.<hash>.wasm artifact (not just carry the right DOM attribute); requests=${JSON.stringify(wasmRequests)}`,
      ).toBeGreaterThan(0);

      // ── preserved editing basics: create-page + add-component ─────────────────
      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("td:has-text('new-page')"), "landing draft should not exist before create-page").toHaveCount(0);
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");
      const createResult = await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: "Landing page created with 3 starter sections. Edit its sections in Content › Pages.",
        expectedChangeKind: "page",
        requireSelection: false,
        requirePreview: false,
      });
      expect(createResult.detail?.change?.kind, "create-page should report a page change in the no-env default").toBe("page");
      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("tr", { has: page.locator("td:has-text('new-page')") }), "created landing draft should persist").toHaveCount(1);

      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(`[data-studio-site-map-workspace-node='${NEW_NODE_KEY}']`), "the new hero instance must not exist before add-component").toHaveCount(0);
      const addResult = await applyCompositionIntentInPlace(page, "add-component:home:hero", {
        expectedMessage: "Hero added to Home.",
        requirePreview: false,
      });
      expect(addResult.detail?.change?.kind, "add-component should report a component change in the no-env default").toBe("component");
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(
        page.locator(`${BOARD_SELECTOR} [data-studio-site-map-workspace-node='${NEW_NODE_KEY}']`).first(),
        "the added hero instance must persist in the still-present DOM site-map",
      ).toBeAttached({ timeout: 30_000 });

      const clientErrors = consoleErrors.filter((line) => /authoring|sitemap|site-map|canvas|surface|__gosx_render|GoSXStudioSiteMapRuntime/i.test(line));
      expect(clientErrors, `unexpected canvas-default console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);

      await parityAgainstCoRender(page, request, {
        selectKey: target!.id,
        selectedKey: picked.boardSelected,
        selectedLabel: picked.label,
        newNodeKey: NEW_NODE_KEY,
      });
    } finally {
      await server.stop();
    }
  });
});

async function parityAgainstCoRender(
  page: Page,
  request: Parameters<typeof startMuddyCanvas>[0],
  expected: { selectKey: string; selectedKey: string | null; selectedLabel: string | null; newNodeKey: string },
) {
  const coRender = await startMuddyCanvas(request);
  try {
    await gotoEditor(page, coRender.baseURL);
    await revealModeIfPresent(page, "advanced");
    await expect(page.locator(GRAPH_SURFACE_SELECTOR).first(), "co-render keeps the DOM graph visible").toBeVisible();
    await page.waitForFunction(() => {
      const w = window as unknown as Record<string, unknown>;
      const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
      return !!rt && typeof rt.setState === "function";
    }, null, { timeout: 120_000 });
    const coRenderSelection = await page.evaluate(({ board, attr, key }) => {
      const root = document.querySelector(board);
      const rt = (window as unknown as { GoSXStudioSiteMapRuntime?: { setState?: (r: Element, p: unknown) => void } }).GoSXStudioSiteMapRuntime;
      if (!root || !rt || typeof rt.setState !== "function") return { selected: null as string | null, label: null as string | null };
      rt.setState(root, { selectedNode: key, selectedNodes: [] });
      const labelEl = root.querySelector("[data-studio-site-map-selected-label]");
      return {
        selected: root.getAttribute(attr),
        label: labelEl ? (labelEl.textContent || "").trim() : null,
      };
    }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR, key: expected.selectKey });
    expect(coRenderSelection.selected, "co-render selection sink should match the canvas-default default").toBe(expected.selectedKey);
    expect(coRenderSelection.label, "co-render selection-detail label should match the canvas-default default").toBe(expected.selectedLabel);

    await expect(page.locator(`[data-studio-site-map-workspace-node='${expected.newNodeKey}']`), "co-render: new hero instance must not exist before add-component").toHaveCount(0);
    const addResult = await applyCompositionIntentInPlace(page, "add-component:home:hero", {
      expectedMessage: "Hero added to Home.",
      requirePreview: false,
    });
    expect(addResult.detail?.change?.kind, "co-render add-component should report a component change").toBe("component");
    await page.goto(`${coRender.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
    await expect(
      page.locator(`${BOARD_SELECTOR} [data-studio-site-map-workspace-node='${expected.newNodeKey}']`).first(),
      "co-render add-component should persist the same DOM site-map node",
    ).toBeAttached({ timeout: 30_000 });
  } finally {
    await coRender.stop();
  }
}

type PickTarget = { id: string; screenX: number; screenY: number; expectedLabel: string };

// pickTargetWithLabel reads the live bundle, finds a rect whose on-screen center
// lands inside the visible canvas AND whose id matches a DOM workspace node, and
// returns its on-screen center plus the label the DOM board carries for that
// node. Computing screen coords via the live transform means the click lands on
// the rect the user actually sees — nothing is hardcoded.
async function pickTargetWithLabel(page: Page): Promise<PickTarget | null> {
  return page.evaluate(({ canvasSel, boardSel }) => {
    const w = window as unknown as {
      __gosx_render_canvas?: (id: string, width: number, height: number, t: number) => unknown;
      __gosx_canvas_board_screen_transform?: (camera: unknown, cssW: number, cssH: number) => { x: (wx: number) => number; y: (wy: number) => number };
    };
    const el = document.querySelector(canvasSel) as HTMLElement | null;
    if (!el) return null;
    const id = el.getAttribute("data-gosx-surface-id") || "";
    const cssW = Math.max(1, el.clientWidth || 1);
    const cssH = Math.max(1, el.clientHeight || 1);
    const out = typeof w.__gosx_render_canvas === "function" ? w.__gosx_render_canvas(id, cssW, cssH, 0) : "";
    if (typeof out !== "string" || out[0] === "e") return null;
    const bundle = JSON.parse(out) as {
      camera?: unknown;
      objects?: Array<{ id?: string; kind?: string; pickable?: boolean; bounds?: { minX?: number; maxX?: number; minY?: number; maxY?: number } }>;
    };
    const objects = Array.isArray(bundle.objects) ? bundle.objects : [];
    const t = typeof w.__gosx_canvas_board_screen_transform === "function"
      ? w.__gosx_canvas_board_screen_transform(bundle.camera, cssW, cssH)
      : null;
    if (!t) return null;

    const board = document.querySelector(boardSel);
    const labelOf = (key: string): string | null => {
      if (!board) return null;
      const node = Array.prototype.slice
        .call(board.querySelectorAll("[data-studio-site-map-workspace-node]"))
        .find((n) => (n as Element).getAttribute("data-studio-site-map-workspace-node") === key) as Element | undefined;
      if (!node) return null;
      return node.getAttribute("data-studio-site-map-node-label") || "";
    };

    for (const obj of objects) {
      if (!obj || obj.kind !== "rect" || !obj.id || obj.pickable === false || !obj.bounds) continue;
      const label = labelOf(obj.id);
      if (label == null || label === "") continue; // only pick rects the DOM board can resolve
      const b = obj.bounds;
      const worldCX = ((b.minX ?? 0) + (b.maxX ?? 0)) / 2;
      const worldCY = ((b.minY ?? 0) + (b.maxY ?? 0)) / 2;
      const sx = t.x(worldCX);
      const sy = t.y(worldCY);
      if (sx >= 6 && sx <= cssW - 6 && sy >= 6 && sy <= cssH - 6) {
        return { id: obj.id, screenX: sx, screenY: sy, expectedLabel: label };
      }
    }
    return null;
  }, { canvasSel: CANVAS_SELECTOR, boardSel: BOARD_SELECTOR });
}

// clickAndPollForSelection synthesizes a real mouse click at the picked rect's
// live on-screen center and polls the DOM board's selection sink + inspector
// label until they reflect the pick (or the deadline elapses).
async function clickAndPollForSelection(
  page: Page,
  box: { x: number; y: number },
  target: PickTarget,
): Promise<{ boardSelected: string | null; label: string | null }> {
  await page.mouse.move(box.x + target.screenX, box.y + target.screenY);
  await page.mouse.down();
  await page.mouse.up();

  const deadline = Date.now() + 10_000;
  let last = { boardSelected: null as string | null, label: null as string | null };
  while (Date.now() < deadline) {
    last = await page.evaluate(({ board, attr }) => {
      const root = document.querySelector(board);
      const labelEl = root ? root.querySelector("[data-studio-site-map-selected-label]") : null;
      return {
        boardSelected: root ? root.getAttribute(attr) : null,
        label: labelEl ? (labelEl.textContent || "").trim() : null,
      };
    }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR });
    if (last.boardSelected === target.id && last.label === target.expectedLabel) return last;
    await page.waitForTimeout(120);
  }
  return last;
}

function isRealRuntimeWasm(url: string): boolean {
  return /(?:^|\/)gosx-runtime(?:-engine)?\.[0-9a-f]+\.wasm(?:\?|$)/i.test(url) || /\/gosx\/runtime\.wasm(?:\?|$)/i.test(url);
}
