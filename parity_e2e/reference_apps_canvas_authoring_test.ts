import { expect, test, type Page } from "@playwright/test";
import {
  applyCompositionIntentInPlace,
  gotoEditor,
  revealModeIfPresent,
  startMuddyCanvas,
} from "./reference_apps_harness";
import {
  formatCanvasRenderEvidence,
  waitForCanvasBoardRenderEvidence,
} from "./canvas_render_evidence";

// Live-proof e2e for AUTHORING PARITY with the Canvas2D site-map board active
// (Slice 2 of the "Canvas2D board → default editor" parity push).
//
// Slice 1 proved a canvas pick drives the inspector. THIS file proves the
// no-code AUTHORING workflow stays reachable AND functional when the editor is
// booted with the Canvas2D board as the active visual (MUDDY_SITEMAP_CANVAS=1),
// so nothing authoring-related is lost when the DOM board graph is hidden later
// (Slice 4). The authoring controls (composition-intent "Create page" / "Add
// section" applies and the per-component mutation panels) render as host chrome
// alongside the canvas — they are siblings of the DOM board graph
// ([data-studio-site-map-board]), not nested inside it (locked by the muddy
// unit guard TestEditorSiteMapAuthoringControlsSurviveBoardHide).
//
// The two authoring paths assert DIFFERENT, HONEST persistence facts:
//
//   create-page (from a blueprint): persists a draft CMS page. Muddy's editor
//   site-map page list is a fixed topology, so a created CMS page does NOT add a
//   new site-map/canvas node — therefore we prove persistence the truthful way:
//   the draft page's public route (404 before) serves 200 after the visible
//   "Create page" control submits. We do NOT claim it surfaces in the canvas
//   graph, because for Muddy it does not.
//
//   add-component (a home section whose base is already enabled): genuinely adds
//   a NEW workspace node (hero is DefaultOn, so "Add hero" inserts a
//   hero__copy_2 instance). After the visible "Add" control submits, we RELOAD
//   the canvas-active editor and assert the new node component:home:hero__copy_2
//   appears in BOTH the reloaded DOM site-map AND the LIVE canvas RenderBundle
//   (a rect whose id == the node key). That is the dispatch's exact requirement:
//   the added section appears in the reloaded canvas graph.
//
// Honesty discipline: the canvas must be genuinely active (canvas2d surface
// attached, route evidence observed (true WebGPU or explicit 2D fallback paint))
// before any authoring is driven, and the canvas-graph assertion reads the live
// __gosx_render_canvas bundle — nothing is hardcoded.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const CANVAS_SELECTOR = "canvas[data-gosx-surface-kind='canvas2d']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const FORMS_SELECTOR = "[data-studio-composition-intent-forms='true']";
// The added hero instance's home section key is hero__copy_2; the workspace
// node key collapses non-alphanumerics to single dashes (gosx-studio
// workspaceToken), so the node/rect key is component:home:hero-copy-2.
const NEW_NODE_KEY = "component:home:hero-copy-2";

test.describe("@reference-apps canvas2d authoring parity", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni create-page and add-component work with the canvas board active", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    const consoleWarnings: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
      if (message.type() === "warning") consoleWarnings.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvas(request);
    try {
      await gotoEditor(page, server.baseURL);
      // The legacy site-map/canvas board under test here now lives inside the
      // "Advanced" mode panel (studio-pagecanvas-handoff moved it there once a
      // PageCanvas surface is present); reveal it before touching the board.
      await revealModeIfPresent(page, "advanced");

      // The canvas must be the genuinely-active visual: surface attached, the
      // full-build canvas painter installed, the board painting real content,
      // and the host authoring chrome (managed forms) present alongside it.
      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(canvas, "canvas2d surface should be emitted under MUDDY_SITEMAP_CANVAS=1").toBeAttached({ timeout: 30_000 });
      await expect(canvas, "full-runtime CanvasBoard should request the WebGPU backend").toHaveAttribute("data-gosx-canvas-backend", "webgpu");
      await expect(page.locator(BOARD_SELECTOR).first(), "DOM site-map board renders alongside the canvas today").toBeAttached();
      await expect(page.locator(FORMS_SELECTOR), "managed authoring forms (host chrome) must be present with the canvas active").toBeAttached();

      await page.waitForFunction(() => {
        const w = window as unknown as { __gosx?: { ready?: boolean } };
        return w.__gosx?.ready === true ||
          document.documentElement.getAttribute("data-gosx-runtime-ready") === "true";
      }, null, { timeout: 120_000 });
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        return typeof w.__gosx_render_canvas === "function" &&
          typeof w.__gosx_canvas_board_screen_transform === "function" &&
          typeof w.__gosx_paint_canvas_bundle === "function";
      }, null, { timeout: 120_000 });
      const initialRenderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings);
      expect(
        initialRenderEvidence.webgpuRoute || initialRenderEvidence.fallback2D,
        `canvas board must render before authoring is driven; evidence=${formatCanvasRenderEvidence(initialRenderEvidence)}`,
      ).toBe(true);

      // ── create-page from a blueprint ───────────────────────────────────────
      // The CMS pages list must NOT already carry the landing draft, so the
      // post-create appearance is a real persistence signal.
      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator("table"), "CMS pages list should render").toBeAttached();
      await expect(page.locator("td:has-text('new-page')"), "landing draft should not exist before create-page").toHaveCount(0);
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(CANVAS_SELECTOR).first()).toBeAttached({ timeout: 30_000 });

      const createResult = await applyCompositionIntentInPlace(page, "create-page:landing", {
        expectedMessage: "Landing created.",
        expectedChangeKind: "page",
        requireSelection: false,
      });
      expect(createResult.detail?.change?.kind, "create-page should report a page change").toBe("page");
      expect(createResult.detail?.result?.previewURL ?? "", "create-page should report the persisted draft preview URL").toMatch(/\/pages\/.+preview=1/);

      // Persistence proof: the created draft now appears in the canonical CMS
      // pages list (title + slug). (Muddy's editor site-map page list is a fixed
      // topology, so a created CMS page does not add a site-map/canvas node — we
      // assert what is actually true: the draft persisted as a real CMS page.)
      await page.goto(`${server.baseURL}/admin/pages`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const landingRow = page.locator("tr", { has: page.locator("td:has-text('new-page')") });
      await expect(landingRow, "created landing draft should persist in the CMS pages list after the canvas-active create-page").toHaveCount(1);
      await expect(landingRow).toContainText("Landing");

      // Return to the canvas-active editor for the add-component path.
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(CANVAS_SELECTOR).first(), "canvas stays active after returning to the editor").toBeAttached({ timeout: 30_000 });

      // ── add-component: a NEW section that appears in the reloaded canvas graph ─
      // "Add hero" inserts a hero__copy_2 instance (hero is already enabled), a
      // genuinely new workspace node component:home:hero__copy_2.
      await expect(page.locator(`[data-studio-site-map-workspace-node='${NEW_NODE_KEY}']`), "the new hero instance must not exist before add-component").toHaveCount(0);

      const addResult = await applyCompositionIntentInPlace(page, "add-component:home:hero", {
        expectedMessage: "Hero added to Home.",
      });
      expect(addResult.detail?.change?.kind, "add-component should report a component change").toBe("component");

      // RELOAD the canvas-active editor and prove the added section persisted
      // AND now appears in the reloaded canvas graph.
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(CANVAS_SELECTOR).first(), "canvas stays active after reload").toBeAttached({ timeout: 30_000 });
      await expect(
        page.locator(`${BOARD_SELECTOR} [data-studio-site-map-workspace-node='${NEW_NODE_KEY}']`).first(),
        "the added hero instance must persist in the reloaded DOM site-map graph",
      ).toBeAttached({ timeout: 30_000 });

      // The canvas graph (live RenderBundle) must carry a rect for the new node.
      await page.waitForFunction(() => {
        const w = window as unknown as Record<string, unknown>;
        return typeof w.__gosx_render_canvas === "function" &&
          (w.__gosx as { ready?: boolean } | undefined)?.ready === true ||
          document.documentElement.getAttribute("data-gosx-runtime-ready") === "true";
      }, null, { timeout: 120_000 });
      const reloadRenderEvidence = await waitForCanvasBoardRenderEvidence(page, consoleWarnings);
      expect(
        reloadRenderEvidence.webgpuRoute || reloadRenderEvidence.fallback2D,
        `canvas must render after reload; evidence=${formatCanvasRenderEvidence(reloadRenderEvidence)}`,
      ).toBe(true);

      const found = await pollForCanvasNode(page, NEW_NODE_KEY);
      expect(
        found,
        `the added section ${NEW_NODE_KEY} should appear as a rect in the reloaded canvas RenderBundle ` +
          `(console: ${JSON.stringify(consoleErrors.slice(-8))})`,
      ).toBe(true);

      const authoringErrors = consoleErrors.filter((line) => /authoring|sitemap|site-map|canvas|surface|__gosx_render/i.test(line));
      expect(authoringErrors, `unexpected canvas-authoring console errors: ${JSON.stringify(authoringErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// pollForCanvasNode reads the live canvas RenderBundle and reports whether it
// contains a rect whose id matches the given workspace node key. The bundle is
// the SAME source the painter draws from, so a hit proves the node is genuinely
// on the canvas graph (not just in the DOM).
async function pollForCanvasNode(page: Page, key: string): Promise<boolean> {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    const hit = await page.evaluate(({ canvasSel, nodeKey }) => {
      const w = window as unknown as {
        __gosx_render_canvas?: (id: string, width: number, height: number, t: number) => unknown;
      };
      const el = document.querySelector(canvasSel) as HTMLElement | null;
      if (!el || typeof w.__gosx_render_canvas !== "function") return false;
      const id = el.getAttribute("data-gosx-surface-id") || "";
      const cssW = Math.max(1, el.clientWidth || 1);
      const cssH = Math.max(1, el.clientHeight || 1);
      const out = w.__gosx_render_canvas(id, cssW, cssH, 0);
      if (typeof out !== "string" || out[0] === "e") return false;
      let bundle: { objects?: Array<{ id?: string; kind?: string }> };
      try {
        bundle = JSON.parse(out);
      } catch {
        return false;
      }
      const objects = Array.isArray(bundle.objects) ? bundle.objects : [];
      return objects.some((obj) => obj && obj.kind === "rect" && obj.id === nodeKey);
    }, { canvasSel: CANVAS_SELECTOR, nodeKey: key });
    if (hit) return true;
    await page.waitForTimeout(250);
  }
  return false;
}
