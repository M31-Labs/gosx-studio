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
} from "./canvas_render_evidence";

// Live-proof e2e for the no-signal CanvasBoard default.
//
// With no Muddy canvas env/query override, canvas-default mode makes the
// Canvas2D board the SOLE site-map graph visual: the DOM board's graph SUB-TREE
// (.studio-site-map-workspace__surface — the workspace layers/nodes + the SVG
// link paths) is hidden by a gated CSS rule, while the DOM board element, its
// sitemapruntime binding, the selection detail card, the workspace rail, and
// the host authoring panels all stay LIVE. This is what lets the canvas be the
// only graph visual WITHOUT losing the Slice-1 selection bridge or Slice-2
// authoring (which both target the still-present DOM board).
//
// This file asserts, end-to-end in headless Chromium:
//   (a) the canvas IS the visible graph and the DOM board graph sub-tree is
//       hidden (display:none / 0-size) — not merely absent;
//   (b) clicking a canvas node updates the inspector + selection detail card
//       (reuses the Slice-1 selection seam);
//   (c) create-page and add-component still work (reuses the Slice-2 seam);
//   (d) explicit default-route footprint evidence — today the default
//       CanvasBoard route selects the full runtime, not the low-footprint
//       wasm-free runtime;
//   (e) a PARITY check — the same selection + authoring actions reach the SAME
//       editor state in canvas-default mode as in co-render (DOM-graph-visible)
//       mode: the selected-node the board records for a given key matches, and
//       the persisted added node appears in the reloaded DOM site-map in both.
//
// Honesty discipline: the target rect is DISCOVERED from the live RenderBundle
// (window.__gosx_render_canvas) and the expected label is READ from the matching
// DOM workspace node — nothing is hardcoded. If canvas-default genuinely breaks
// selection/authoring, the offending assertion is left test.fixme with the
// observed evidence rather than faked.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy dist rebuilt against the
// gosx/gosx-studio under test (`gosx build --dev .` in muddy-noni-commerce).

const CANVAS_SELECTOR = "canvas[data-gosx-surface-kind='canvas2d']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const GRAPH_SURFACE_SELECTOR = `${BOARD_SELECTOR} .studio-site-map-workspace__surface`;
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";
const FORMS_SELECTOR = "[data-studio-composition-intent-forms='true']";
// "Add hero" inserts a hero__copy_2 instance (hero is DefaultOn); the workspace
// node key collapses non-alphanumerics to single dashes, so the node/rect key
// is component:home:hero-copy-2 (same as the co-render authoring test).
const NEW_NODE_KEY = "component:home:hero-copy-2";

test.describe("@reference-apps canvas2d default canvas-default mode", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni default route boots CanvasBoard as sole graph with selection + authoring parity", async ({ page, request }, testInfo) => {
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

      // The DOM board element must still be present (the bridge targets its root
      // + the selection detail card is DOM-board markup), and the managed
      // authoring forms (host chrome) must be present alongside the canvas.
      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board element must stay in the markup (bridge + selection card depend on it)").toBeAttached();
      await expect(page.locator(FORMS_SELECTOR), "managed authoring forms (host chrome) must be present in canvas-default mode").toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "canvas2d surface must be emitted in canvas-default mode").toBeAttached({ timeout: 30_000 });
      await expect(canvas, "canvas-default CanvasBoard should request the WebGPU backend").toHaveAttribute("data-gosx-canvas-backend", "webgpu");

      // Runtime + canvas entrypoints + DOM board runtime (bridge sink) + the
      // muddy bridge must all be installed. A missing piece means the served
      // bundle predates this work — fail loud, do not fake.
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

      // ── (a) canvas is the visible graph; DOM graph sub-tree is hidden ───────
      const box = await canvas.boundingBox();
      expect(box, "canvas should have a layout box (it is the visible graph)").not.toBeNull();
      expect(box!.width, "canvas CSS width > 0").toBeGreaterThan(0);
      expect(box!.height, "canvas CSS height > 0").toBeGreaterThan(0);

      const graphSurface = page.locator(GRAPH_SURFACE_SELECTOR).first();
      await expect(graphSurface, "the DOM board graph sub-tree element must still EXIST (only hidden, so the bridge can resolve node attrs)").toBeAttached();
      await expect(graphSurface, "the DOM board graph sub-tree must be HIDDEN in canvas-default mode (canvas is the sole graph)").toBeHidden();
      const graphBox = await graphSurface.boundingBox();
      expect(graphBox, "the hidden DOM graph sub-tree must paint to 0-size / null box").toBeNull();

      const initialRenderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings);
      expect(
        initialRenderEvidence.webgpuRoute || initialRenderEvidence.fallback2D,
        `the canvas board must render before selection; evidence=${formatCanvasRenderEvidence(initialRenderEvidence)}`,
      ).toBe(true);

      const manifestRuntime = await runtimeManifestPath(page);
      await page.waitForTimeout(250);
      const fullWasm = wasmRequests.filter(isFullRuntimeWasm);
      const islandsWasm = wasmRequests.filter((u) => /gosx-runtime-islands/i.test(u) || /runtime-islands\.wasm/i.test(u));
      const defaultFootprintEvidence = {
        manifestRuntime,
        wasmRequests,
        fullWasm,
        islandsWasm,
        renderEvidence: initialRenderEvidence,
      };
      await testInfo.attach("default-canvasboard-footprint-evidence.json", {
        contentType: "application/json",
        body: JSON.stringify(defaultFootprintEvidence, null, 2),
      });
      expect(
        manifestRuntime && !/islands/i.test(manifestRuntime) && (/gosx-runtime/i.test(manifestRuntime) || /runtime\.wasm/i.test(manifestRuntime)),
        `default CanvasBoard route currently selects the full runtime; footprint evidence=${JSON.stringify(defaultFootprintEvidence)}`,
      ).toBe(true);
      expect(
        fullWasm.length,
        `default CanvasBoard route should fetch the full runtime WASM today; footprint evidence=${JSON.stringify(defaultFootprintEvidence)}`,
      ).toBeGreaterThanOrEqual(1);

      // ── (b) clicking a canvas node updates the inspector + selection card ───
      const target = await pickTargetWithLabel(page);
      expect(target, "the rendered bundle should expose a rect whose key matches a DOM workspace node").not.toBeNull();
      expect(target!.expectedLabel, "the matched DOM node should carry a non-empty label").toMatch(/\S/);

      const labelBefore = await selectedLabelText(page);
      const selectedBefore = await boardSelectedNode(page);

      await page.mouse.move(box!.x + target!.screenX, box!.y + target!.screenY);
      await page.mouse.down();
      await page.mouse.up();

      const settled = await pollForSelection(page, target!.id, target!.expectedLabel);
      expect(
        settled.boardSelected,
        `board root ${SELECTED_NODE_ATTR} should equal the picked key in canvas-default mode: ` +
          `expected ${target!.id}, before=${JSON.stringify(selectedBefore)}, got ${JSON.stringify(settled.boardSelected)} ` +
          `(signal=${JSON.stringify(settled.signal)}, console=${JSON.stringify(consoleErrors.slice(-8))})`,
      ).toBe(target!.id);
      expect(
        settled.label,
        `selection-detail label should update to the picked node in canvas-default mode: ` +
          `expected ${JSON.stringify(target!.expectedLabel)}, before=${JSON.stringify(labelBefore)}, got ${JSON.stringify(settled.label)}`,
      ).toBe(target!.expectedLabel);

      // The matched (hidden) DOM workspace node must still be flagged selected —
      // proves the runtime resolves attrs on the display:none node.
      const nodeSelected = await page.evaluate(({ board, key }) => {
        const root = document.querySelector(board);
        if (!root) return null;
        const node = Array.prototype.slice
          .call(root.querySelectorAll("[data-studio-site-map-workspace-node]"))
          .find((el) => (el as Element).getAttribute("data-studio-site-map-workspace-node") === key) as Element | undefined;
        return node ? node.getAttribute("data-studio-site-map-node-selected") : null;
      }, { board: BOARD_SELECTOR, key: target!.id });
      expect(nodeSelected, `the picked (hidden) DOM workspace node (${target!.id}) should be marked selected`).toBe("true");

      // Record the canvas-default selection result for the parity check (d).
      const canvasDefaultSelectedKey = settled.boardSelected;
      const canvasDefaultSelectedLabel = settled.label;

      // ── (c) create-page and add-component still work in canvas-default ──────
      // create-page persists a CMS page (Muddy's site-map topology is fixed, so
      // it does not add a canvas node — assert the truthful persistence fact).
      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("table"), "CMS pages list should render").toBeAttached();
      await expect(page.locator("td:has-text('new-page')"), "landing draft should not exist before create-page").toHaveCount(0);
      await gotoEditor(page, server.baseURL);
      await expect(page.locator(CANVAS_SELECTOR).first()).toBeAttached({ timeout: 30_000 });

      const createResult = await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: "Landing created.",
        expectedChangeKind: "page",
        requireSelection: false,
        requirePreview: false,
      });
      expect(createResult.detail?.change?.kind, "create-page should report a page change in canvas-default").toBe("page");

      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const landingRow = page.locator("tr", { has: page.locator("td:has-text('new-page')") });
      await expect(landingRow, "created landing draft should persist in the CMS pages list in canvas-default mode").toHaveCount(1);
      await expect(landingRow).toContainText("Landing");

      // add-component: a NEW workspace node must appear in the still-present
      // DOM board graph sub-tree. Muddy's single-surface CanvasBoard is
      // intentionally page artboards only, so component nodes do not become
      // canvas rects.
      await gotoEditor(page, server.baseURL);
      await expect(page.locator(CANVAS_SELECTOR).first()).toBeAttached({ timeout: 30_000 });
      await expect(page.locator(`[data-studio-site-map-workspace-node='${NEW_NODE_KEY}']`), "the new hero instance must not exist before add-component").toHaveCount(0);

      const addResult = await applyCompositionIntentInPlace(page, "add-component:home:hero", {
        expectedMessage: "Hero added to Home.",
        requirePreview: false,
      });
      expect(addResult.detail?.change?.kind, "add-component should report a component change in canvas-default").toBe("component");

      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator(CANVAS_SELECTOR).first(), "canvas stays active after reload").toBeAttached({ timeout: 30_000 });
      // The added node must persist in the (hidden but present) DOM site-map.
      await expect(
        page.locator(`${BOARD_SELECTOR} [data-studio-site-map-workspace-node='${NEW_NODE_KEY}']`).first(),
        "the added hero instance must persist in the reloaded DOM site-map graph (present though hidden)",
      ).toBeAttached({ timeout: 30_000 });
      // And the DOM graph sub-tree must STILL be hidden after the reload.
      await expect(page.locator(GRAPH_SURFACE_SELECTOR).first(), "DOM graph sub-tree must stay hidden after reload").toBeHidden();

      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        return typeof w.__gosx_render_canvas === "function" &&
          ((w.__gosx as { ready?: boolean } | undefined)?.ready === true ||
            document.documentElement.getAttribute("data-gosx-runtime-ready") === "true");
      }, null, { timeout: 120_000 });
      const reloadRenderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings);
      expect(
        reloadRenderEvidence.webgpuRoute || reloadRenderEvidence.fallback2D,
        `canvas must render after reload; evidence=${formatCanvasRenderEvidence(reloadRenderEvidence)}`,
      ).toBe(true);

      const authoringErrors = consoleErrors.filter((line) => /authoring|sitemap|site-map|canvas|surface|__gosx_render|selection|GoSXStudioSiteMapRuntime/i.test(line));
      expect(authoringErrors, `unexpected canvas-default console errors: ${JSON.stringify(authoringErrors)}`).toEqual([]);

      // ── (e) PARITY: the same actions reach the same state as co-render mode ─
      // Boot a SECOND Muddy in co-render mode (DOM graph visible), drive the
      // SAME selection + add-component, and compare the resulting editor state.
      await parityAgainstCoRender(page, request, {
        selectKey: target!.id,
        selectLabel: target!.expectedLabel,
        canvasDefaultSelectedKey,
        canvasDefaultSelectedLabel,
        newNodeKey: NEW_NODE_KEY,
      });
    } finally {
      await server.stop();
    }
  });
});

// parityAgainstCoRender boots Muddy in co-render mode (the DOM board graph is
// VISIBLE), drives the SAME selection key and the SAME add-component, and asserts
// the resulting editor state matches what canvas-default produced: the board
// records the same selected node + label for a given key, and the added node
// persists in the reloaded DOM site-map in BOTH modes. This is the parity claim:
// canvas-default changes only WHICH graph is visible, not the editor state the
// selection/authoring actions reach.
async function parityAgainstCoRender(
  page: Page,
  request: Parameters<typeof startMuddyCanvas>[0],
  expected: { selectKey: string; selectLabel: string; canvasDefaultSelectedKey: string | null; canvasDefaultSelectedLabel: string | null; newNodeKey: string },
) {
  const coRender = await startMuddyCanvas(request);
  try {
    await gotoEditor(page, coRender.baseURL);
    await expect(page.locator(BOARD_SELECTOR).first()).toBeAttached();
    // In co-render the DOM graph sub-tree is VISIBLE (the contrast that proves
    // canvas-default is genuinely hiding it, not a build artifact).
    await expect(page.locator(GRAPH_SURFACE_SELECTOR).first(), "co-render keeps the DOM board graph sub-tree VISIBLE").toBeVisible();
    await page.waitForFunction(() => {
      const w = window as unknown as Record<string, unknown>;
      const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
      return !!rt && typeof rt.setState === "function";
    }, null, { timeout: 120_000 });

    // Selection parity: drive the SAME node key through the DOM board runtime's
    // public setState (the same sink the canvas bridge uses) and compare the
    // recorded selected-node + selection-detail label to canvas-default's.
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

    expect(
      coRenderSelection.selected,
      `selection parity: co-render board should record the same selected node as canvas-default ` +
        `(canvas-default=${JSON.stringify(expected.canvasDefaultSelectedKey)}, co-render=${JSON.stringify(coRenderSelection.selected)})`,
    ).toBe(expected.canvasDefaultSelectedKey);
    expect(
      coRenderSelection.label,
      `selection parity: co-render selection-detail label should match canvas-default ` +
        `(canvas-default=${JSON.stringify(expected.canvasDefaultSelectedLabel)}, co-render=${JSON.stringify(coRenderSelection.label)})`,
    ).toBe(expected.canvasDefaultSelectedLabel);

    // Authoring parity: the SAME add-component persists the SAME new node in the
    // reloaded DOM site-map (canvas-default proved this above; co-render must too).
    await expect(page.locator(`[data-studio-site-map-workspace-node='${expected.newNodeKey}']`), "co-render: new hero instance must not exist before add-component").toHaveCount(0);
    const addResult = await applyCompositionIntentInPlace(page, "add-component:home:hero", {
      expectedMessage: "Hero added to Home.",
      requirePreview: false,
    });
    expect(addResult.detail?.change?.kind, "co-render add-component should report a component change").toBe("component");
    await page.goto(`${coRender.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
    await expect(
      page.locator(`${BOARD_SELECTOR} [data-studio-site-map-workspace-node='${expected.newNodeKey}']`).first(),
      `authoring parity: the added node ${expected.newNodeKey} must persist in the reloaded DOM site-map in co-render mode TOO (same state both modes)`,
    ).toBeAttached({ timeout: 30_000 });
  } finally {
    await coRender.stop();
  }
}

// pickTargetWithLabel reads the live bundle, finds a rect whose on-screen center
// lands inside the visible canvas AND whose id matches a DOM workspace node, and
// returns its on-screen center plus the label the DOM board carries for that
// node. (Shared shape with the Slice-1 selection test.)
async function pickTargetWithLabel(page: Page): Promise<{ id: string; screenX: number; screenY: number; expectedLabel: string } | null> {
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
      objects?: Array<{ id?: string; kind?: string; bounds?: { minX?: number; maxX?: number; minY?: number; maxY?: number } }>;
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
      if (!obj || obj.kind !== "rect" || !obj.id || !obj.bounds) continue;
      const label = labelOf(obj.id);
      if (label == null || label === "") continue; // only pick rects the DOM board can resolve
      const b = obj.bounds;
      const worldCX = ((b.minX ?? 0) + (b.maxX ?? 0)) / 2;
      const worldCY = ((b.minY ?? 0) + (b.maxY ?? 0)) / 2;
      const sx = t.x(worldCX);
      const sy = t.y(worldCY);
      if (sx >= 2 && sx <= cssW - 2 && sy >= 2 && sy <= cssH - 2) {
        return { id: obj.id, screenX: sx, screenY: sy, expectedLabel: label };
      }
    }
    return null;
  }, { canvasSel: CANVAS_SELECTOR, boardSel: BOARD_SELECTOR });
}

async function selectedLabelText(page: Page): Promise<string | null> {
  return page.evaluate((board) => {
    const root = document.querySelector(board);
    if (!root) return null;
    const el = root.querySelector("[data-studio-site-map-selected-label]");
    return el ? (el.textContent || "").trim() : null;
  }, BOARD_SELECTOR);
}

async function boardSelectedNode(page: Page): Promise<string | null> {
  return page.evaluate(({ board, attr }) => {
    const root = document.querySelector(board);
    return root ? root.getAttribute(attr) : null;
  }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR });
}

async function pollForSelection(page: Page, wantKey: string, wantLabel: string): Promise<{ boardSelected: string | null; label: string | null; signal: string | null }> {
  const deadline = Date.now() + 10_000;
  let last: { boardSelected: string | null; label: string | null; signal: string | null } = { boardSelected: null, label: null, signal: null };
  while (Date.now() < deadline) {
    last = await page.evaluate(({ board, attr, signalName }) => {
      const root = document.querySelector(board);
      const get = (window as unknown as { __gosx_get_shared_signal?: (n: string) => unknown }).__gosx_get_shared_signal;
      let signal: string | null = null;
      if (typeof get === "function") {
        const raw = get(signalName);
        if (typeof raw === "string") {
          try { const p = JSON.parse(raw); signal = typeof p === "string" ? p : String(p); } catch { signal = raw; }
        }
      }
      const labelEl = root ? root.querySelector("[data-studio-site-map-selected-label]") : null;
      return {
        boardSelected: root ? root.getAttribute(attr) : null,
        label: labelEl ? (labelEl.textContent || "").trim() : null,
        signal,
      };
    }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR, signalName: "$surface.event.selectedID" });
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
