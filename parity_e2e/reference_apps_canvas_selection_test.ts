import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvas } from "./reference_apps_harness";

// Live-proof e2e for the Canvas2D site-map SELECTION BRIDGE (Slice 1 of the
// "Canvas2D board → default editor" parity push).
//
// The sibling reference_apps_canvas_interaction_test.ts proves a canvas click
// writes the picked node key into $surface.event.selectedID. THIS file proves
// the keystone integration: that pick must drive the right-rail inspector /
// selection detail card EXACTLY as a DOM-board selection does.
//
// The seam, pinned end-to-end:
//   - Canvas pick → gosx canvasBoardApplyPick writes $surface.event.selectedID
//     = the picked workspace node key (gosx-studio sitemap_canvas.go: id == key),
//     and store.Set fans out to JS subscribers via __gosx_notify_shared_signal.
//   - The muddy selection bridge (internal/gosxstudio/canvas_selection_bridge.js,
//     mounted ONLY under MUDDY_SITEMAP_CANVAS=1) reads that signal and calls
//     window.GoSXStudioSiteMapRuntime.setState(boardRoot, { selectedNode: key }),
//     the SAME call a DOM-board node click makes (gosx-studio
//     sitemapruntime/island_runtime.js handleClick → setState).
//   - setState → sync → writeSelectionDetails populates the selection detail
//     card ([data-studio-site-map-selected-label] etc.) and writes
//     data-studio-site-map-selected-node on the board root.
//
// Honesty discipline (matching the interaction test): the target rect is
// DISCOVERED from the live RenderBundle (window.__gosx_render_canvas) and the
// expected label is READ from the matching DOM workspace node — nothing is
// hardcoded. The click is synthesized at the rect's live on-screen center via
// the same screen transform the painter uses. The assertion is driven off the
// inspector's observable text, so it passes only if the bridge truly updated
// the inspector live.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy dist rebuilt against the
// gosx/gosx-studio under test (`gosx build --dev .` in muddy-noni-commerce).

const CANVAS_SELECTOR = "canvas[data-gosx-surface-kind='canvas2d']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const SELECTED_NODE_ATTR = "data-studio-site-map-selected-node";

test.describe("@reference-apps canvas2d site-map selection bridge", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni canvas pick drives the right-rail inspector like a DOM-board selection", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvas(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board should be present (the bridge feeds its selection state)").toBeAttached();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "canvas2d surface should be emitted under MUDDY_SITEMAP_CANVAS=1").toBeAttached({ timeout: 30_000 });

      // Runtime ready + the canvas interaction entry (__gosx_canvas_event) AND
      // the DOM board runtime (GoSXStudioSiteMapRuntime.setState — the bridge's
      // sink) AND the muddy bridge itself must be installed. Any missing piece
      // means the served bundle predates this work — fail loud, do not fake.
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
      expect(box!.width, "canvas CSS width > 0").toBeGreaterThan(0);
      expect(box!.height, "canvas CSS height > 0").toBeGreaterThan(0);

      // Wait for the board to actually paint before probing — otherwise there's
      // nothing to pick.
      const painted = await pollForPaint(page);
      expect(painted, "board must paint before selection").toBe(true);

      // Discover a pickable rect from the live bundle AND read the label the DOM
      // board carries for that same node key — the inspector should end up
      // showing exactly this label after the canvas pick.
      const target = await pickTargetWithLabel(page);
      expect(
        target,
        "the rendered bundle should expose a rect whose key matches a DOM workspace node (so the inspector can resolve it)",
      ).not.toBeNull();
      expect(target!.expectedLabel, "the matched DOM node should carry a non-empty label").toMatch(/\S/);

      // Record what the inspector showed BEFORE the pick so we can prove it
      // actually CHANGED to the picked node (not coincidentally already there).
      const labelBefore = await selectedLabelText(page);
      const selectedBefore = await boardSelectedNode(page);

      // Click the rect at its live on-screen center.
      await page.mouse.move(box!.x + target!.screenX, box!.y + target!.screenY);
      await page.mouse.down();
      await page.mouse.up();

      // The bridge subscribes to the canvas pick signal and feeds it into the
      // DOM board's setState. Assert the inspector / selection detail card now
      // reflects the picked node: (a) the board root records the picked key, and
      // (b) the selection detail label reads the picked node's label.
      const settled = await pollForSelection(page, target!.id, target!.expectedLabel);
      expect(
        settled.boardSelected,
        `board root ${SELECTED_NODE_ATTR} should equal the picked key: ` +
          `expected ${target!.id}, before=${JSON.stringify(selectedBefore)}, ` +
          `got ${JSON.stringify(settled.boardSelected)} ` +
          `(signal=${JSON.stringify(settled.signal)}, console=${JSON.stringify(consoleErrors.slice(-8))})`,
      ).toBe(target!.id);

      expect(
        settled.label,
        `inspector selection-detail label should update to the picked node: ` +
          `expected ${JSON.stringify(target!.expectedLabel)}, before=${JSON.stringify(labelBefore)}, ` +
          `got ${JSON.stringify(settled.label)}`,
      ).toBe(target!.expectedLabel);

      // The matched DOM workspace node must also be flagged selected (this is the
      // same visual-selection contract a DOM-board click drives).
      const nodeSelected = await page.evaluate(({ board, key }) => {
        const root = document.querySelector(board);
        if (!root) return null;
        const node = Array.prototype.slice
          .call(root.querySelectorAll("[data-studio-site-map-workspace-node]"))
          .find((el) => (el as Element).getAttribute("data-studio-site-map-workspace-node") === key) as Element | undefined;
        return node ? node.getAttribute("data-studio-site-map-node-selected") : null;
      }, { board: BOARD_SELECTOR, key: target!.id });
      expect(nodeSelected, `the picked DOM workspace node (${target!.id}) should be marked selected`).toBe("true");

      const bridgeErrors = consoleErrors.filter((line) => /selection|sitemap|site-map|__gosx_get_shared_signal|GoSXStudioSiteMapRuntime/i.test(line));
      expect(bridgeErrors, `unexpected selection-bridge console errors: ${JSON.stringify(bridgeErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// pickTargetWithLabel reads the live bundle, finds a rect whose on-screen center
// lands inside the visible canvas AND whose id matches a DOM workspace node, and
// returns its on-screen center plus the label the DOM board carries for that
// node. Computing screen coords via the live transform means the click lands on
// the rect the user actually sees.
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

// selectedLabelText returns the current text of the selection-detail label in
// the DOM board (the right-rail inspector / selection card target).
async function selectedLabelText(page: Page): Promise<string | null> {
  return page.evaluate((board) => {
    const root = document.querySelector(board);
    if (!root) return null;
    const el = root.querySelector("[data-studio-site-map-selected-label]");
    return el ? (el.textContent || "").trim() : null;
  }, BOARD_SELECTOR);
}

// boardSelectedNode returns the board root's recorded selected node key.
async function boardSelectedNode(page: Page): Promise<string | null> {
  return page.evaluate(({ board, attr }) => {
    const root = document.querySelector(board);
    return root ? root.getAttribute(attr) : null;
  }, { board: BOARD_SELECTOR, attr: SELECTED_NODE_ATTR });
}

// pollForSelection waits until the board reflects the picked node both in its
// recorded selection attribute and in the selection-detail label, returning the
// last observed values (and the raw signal) for diagnostics on timeout.
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

// pollForPaint waits until the board draws at least one non-background pixel.
async function pollForPaint(page: Page): Promise<boolean> {
  const deadline = Date.now() + 60_000;
  while (Date.now() < deadline) {
    const nonBackground = await page.evaluate((selector) => {
      const canvas = document.querySelector(selector);
      if (!(canvas instanceof HTMLCanvasElement)) return 0;
      const ctx = canvas.getContext("2d");
      if (!ctx) return 0;
      const wpx = canvas.width;
      const hpx = canvas.height;
      if (wpx === 0 || hpx === 0) return 0;
      let data: Uint8ClampedArray;
      try {
        data = ctx.getImageData(0, 0, wpx, hpx).data;
      } catch {
        return 0;
      }
      const bg = { r: 0x0f, g: 0x17, b: 0x20 };
      const tol = 10;
      let count = 0;
      const stride = 4 * Math.max(1, Math.floor((wpx * hpx) / 50000));
      for (let i = 0; i < data.length; i += stride) {
        const r = data[i], g = data[i + 1], b = data[i + 2], a = data[i + 3];
        if (a === 0) continue;
        if (Math.abs(r - bg.r) > tol || Math.abs(g - bg.g) > tol || Math.abs(b - bg.b) > tol) count++;
      }
      return count;
    }, CANVAS_SELECTOR);
    if (nonBackground > 0) return true;
    await page.waitForTimeout(250);
  }
  return false;
}
