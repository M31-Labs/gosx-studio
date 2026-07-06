import { expect, test, type Page } from "@playwright/test";
import {
  applyCompositionIntentInPlace,
  gotoEditor,
  startMuddyCanvas,
  startMuddyCanvasDefault,
} from "./reference_apps_harness";
import {
  formatCanvasRenderEvidence,
  waitForCanvasBoardRenderEvidence,
  WASM_FREE_CANVAS_SELECTOR,
} from "./canvas_render_evidence";

// Live-proof e2e for the no-signal Muddy/Noni editor canvas default.
//
// With no Muddy canvas env/query override, the production default is now the
// low-WASM static WebGPU/Canvas2D-fallback path: a muddy-owned canvas with no
// data-gosx-surface-kind, no full CanvasBoard globals, and no full
// gosx-runtime.<hash>.wasm fetch. Zero WASM requests are valid/preferred when
// Studio JS runtimes provide the needed globals; if any WASM or manifest runtime
// path is present, it must be islands-only. The still-present DOM board remains
// the selection/authoring sink while its graph sub-tree is hidden by the
// sole-graph marker.

const CANVAS_SELECTOR = WASM_FREE_CANVAS_SELECTOR;
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const GRAPH_SURFACE_SELECTOR = `${BOARD_SELECTOR} .studio-site-map-workspace__surface`;
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const FORMS_SELECTOR = "[data-studio-composition-intent-forms='true']";
const NEW_NODE_KEY = "component:home:hero-copy-2";

type RectInfo = {
  id: string;
  screenX: number;
  screenY: number;
};

test.describe("@reference-apps low-WASM default canvas", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni no-env default uses low-WASM static WebGPU/fallback canvas and preserves editing basics", async ({ page, request }, testInfo) => {
    const consoleErrors: string[] = [];
    const consoleWarnings: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
      if (message.type() === "warning") consoleWarnings.push(message.text());
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

      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup").toBeAttached();
      await expect(page.locator(FORMS_SELECTOR), "managed authoring forms must stay present").toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "no-env default must emit the muddy-owned WASM-free canvas").toBeAttached({ timeout: 30_000 });
      await expect(canvas, "default canvas must not be a gosx full-WASM surface").not.toHaveAttribute("data-gosx-surface-kind", "canvas2d");

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

      const noFullCanvasBoardGlobals = await page.evaluate(() => {
        const w = window as unknown as Record<string, unknown>;
        return typeof w.__gosx_render_canvas !== "function" &&
          typeof w.__gosx_canvas_event !== "function" &&
          typeof w.__gosx_canvas_set_backend !== "function";
      });
      expect(noFullCanvasBoardGlobals, "no-env default must not expose full CanvasBoard globals").toBe(true);

      const graphSurface = page.locator(GRAPH_SURFACE_SELECTOR).first();
      await expect(graphSurface, "DOM graph sub-tree still exists").toBeAttached();
      await expect(graphSurface, "DOM graph sub-tree is hidden because the canvas is the sole graph").toBeHidden();

      await canvas.scrollIntoViewIfNeeded();
      const box = await canvas.boundingBox();
      expect(box, "default canvas should have a layout box").not.toBeNull();
      expect(box!.width, "canvas CSS width > 0").toBeGreaterThan(0);
      expect(box!.height, "canvas CSS height > 0").toBeGreaterThan(0);

      const renderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings, {
        selector: CANVAS_SELECTOR,
      });
      await testInfo.attach("default-low-wasm-static-webgpu-route-evidence.json", {
        contentType: "application/json",
        body: formatCanvasRenderEvidence(renderEvidence),
      });
      expect(
        renderEvidence.webgpuLoaded,
        `default route must load the scene3d-webgpu chunk; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);
      expect(
        renderEvidence.webgpuAPI,
        `default route must expose window.__gosx_scene3d_webgpu_api.createRenderer; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);
      expect(
        renderEvidence.webgpuRoute || renderEvidence.fallback2D,
        `default route must either publish WebGPU frame attrs or explicitly fall back to Canvas2D; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
      ).toBe(true);
      if (!renderEvidence.webgpuRoute) {
        expect(
          renderEvidence.hostAttrs["data-gosx-canvas-wasm-free-fallback-reason"],
          `Canvas2D fallback must expose a stable fallback reason; evidence=${formatCanvasRenderEvidence(renderEvidence)}`,
        ).toMatch(/\S/);
      }

      await page.waitForTimeout(250);
      const manifestRuntime = await runtimeManifestPath(page);
      const fullWasm = wasmRequests.filter(isFullRuntimeWasm);
      const islandsWasm = wasmRequests.filter(isIslandsRuntimePath);
      const zeroWasmNoManifestRuntime = wasmRequests.length === 0 && manifestRuntime === "";
      const islandsOnlyFootprint = fullWasm.length === 0 &&
        wasmRequests.every(isIslandsRuntimePath) &&
        (manifestRuntime === "" || isIslandsRuntimePath(manifestRuntime));
      const footprintEvidence = { manifestRuntime, wasmRequests, fullWasm, islandsWasm, zeroWasmNoManifestRuntime, renderEvidence };
      await testInfo.attach("default-low-wasm-footprint-evidence.json", {
        contentType: "application/json",
        body: JSON.stringify(footprintEvidence, null, 2),
      });
      expect(
        fullWasm,
        `the full gosx-runtime.<hash>.wasm must never be fetched by the no-env default; footprint=${JSON.stringify(footprintEvidence)}`,
      ).toEqual([]);
      expect(
        zeroWasmNoManifestRuntime || islandsOnlyFootprint,
        `the no-env default must have either zero WASM with no manifest runtime path, or an islands-only WASM footprint; footprint=${JSON.stringify(footprintEvidence)}`,
      ).toBe(true);

      const rects = await pollForRects(page);
      expect(rects.length, `need at least one resolvable default canvas rect; got ${rects.length}`).toBeGreaterThanOrEqual(1);
      const pickTarget = rects[0];
      const expectedLabel = await labelForKey(page, pickTarget.id);
      expect(expectedLabel, `picked node ${pickTarget.id} should carry a DOM label`).toMatch(/\S/);
      await page.mouse.move(box!.x + pickTarget.screenX, box!.y + pickTarget.screenY);
      await page.mouse.down();
      await page.mouse.up();
      const picked = await pollForSelection(page, pickTarget.id, expectedLabel!);
      expect(picked.boardSelected, `click should set board ${SELECTED_NODE_ATTR}=${pickTarget.id}`).toBe(pickTarget.id);
      expect(picked.label, `click should drive inspector label to ${JSON.stringify(expectedLabel)}`).toBe(expectedLabel);

      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("td:has-text('new-page')"), "landing draft should not exist before create-page").toHaveCount(0);
      await gotoEditor(page, server.baseURL);
      const createResult = await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: "Landing created.",
        expectedChangeKind: "page",
        requireSelection: false,
        requirePreview: false,
      });
      expect(createResult.detail?.change?.kind, "create-page should report a page change in the no-env default").toBe("page");
      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("tr", { has: page.locator("td:has-text('new-page')") }), "created landing draft should persist").toHaveCount(1);

      await gotoEditor(page, server.baseURL);
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

      const clientErrors = consoleErrors.filter((line) => /authoring|sitemap|site-map|canvas|surface|wasm-free|GoSXStudioSiteMapRuntime/i.test(line));
      expect(clientErrors, `unexpected low-WASM default console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);

      await parityAgainstCoRender(page, request, {
        selectKey: pickTarget.id,
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
    expect(coRenderSelection.selected, "co-render selection sink should match the low-WASM default").toBe(expected.selectedKey);
    expect(coRenderSelection.label, "co-render selection-detail label should match the low-WASM default").toBe(expected.selectedLabel);

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

async function pollForRects(page: Page): Promise<RectInfo[]> {
  const deadline = Date.now() + 30_000;
  let last: RectInfo[] = [];
  while (Date.now() < deadline) {
    last = await page.evaluate(({ canvasSel, boardSel }) => {
      const el = document.querySelector(canvasSel) as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: { camera: () => { x: number; y: number; z: number }; bundle: () => unknown } }) | null;
      if (!el || !el.GoSXStudioCanvasWasmFree) return [];
      const bundle = el.GoSXStudioCanvasWasmFree.bundle() as { objects?: Array<{ id?: string; kind?: string; pickable?: boolean; bounds?: { minX?: number; maxX?: number; minY?: number; maxY?: number } }> } | null;
      if (!bundle) return [];
      const cam = el.GoSXStudioCanvasWasmFree.camera();
      const cssW = Math.max(1, el.clientWidth || 1);
      const cssH = Math.max(1, el.clientHeight || 1);
      const zoom = cam.z > 0 ? cam.z : 1;
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
          out.push({ id: obj.id, screenX, screenY });
        }
      }
      return out;
    }, { canvasSel: CANVAS_SELECTOR, boardSel: BOARD_SELECTOR });
    if (last.length > 0) return last;
    await page.waitForTimeout(250);
  }
  return last;
}

async function labelForKey(page: Page, key: string): Promise<string | null> {
  return page.evaluate(({ board, key }) => {
    const root = document.querySelector(board);
    if (!root) return null;
    const node = Array.prototype.slice
      .call(root.querySelectorAll("[data-studio-site-map-workspace-node]"))
      .find((el) => (el as Element).getAttribute("data-studio-site-map-workspace-node") === key) as Element | undefined;
    return node ? node.getAttribute("data-studio-site-map-node-label") || "" : null;
  }, { board: BOARD_SELECTOR, key });
}

async function pollForSelection(page: Page, wantKey: string, wantLabel: string): Promise<{ boardSelected: string | null; label: string | null }> {
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
    if (last.boardSelected === wantKey && last.label === wantLabel) return last;
    await page.waitForTimeout(120);
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
