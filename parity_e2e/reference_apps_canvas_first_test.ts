import { expect, test, type Page } from "@playwright/test";
import { startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M9 "pure canvas-first" editor shape.
//
// M9 made the Muddy/Noni admin editor canvas-first: the Canvas2D site-map board
// is built with SiteMapCanvasOptions.PageArtboardsOnly=true (so it renders ONLY
// page artboards — no component/resource node cards, no inter-node link
// "spiderweb" lines) AND the legacy iframe-preview panel (CanvasEnginePanel /
// StudioPreviewFrame / the editor-preview-frame <iframe> + studio-preview-overlay)
// is gone, leaving the Canvas2D board as the SINGLE main editing surface.
//
// This file asserts that shape end-to-end in headless Chromium, against the live
// server-rendered editor (same boot as the sibling canvas tests:
// startMuddyCanvasHTMLSurface → MUDDY_CANVAS_WASM_FREE=1 + MUDDY_EDITOR_SCENE_DOM=1):
//
//   (1) NO SPIDERWEB — parse the inline [data-gosx-canvas-bundle] JSON the canvas
//       engine emits and assert bundle.lines is empty/absent (the RenderLine[]
//       json tag is "lines,omitempty", so PageArtboardsOnly → zero lines → key
//       absent), AND that page cards ARE present (bundle.objects carries rect
//       page-card objects). Only page artboards, no connecting lines.
//   (2) ONE SURFACE — the editor DOM has exactly one main canvas board surface
//       (one [data-gosx-canvas-bundle] script + one canvas board) and NO iframe
//       preview: editor-preview-frame / .studio-preview-overlay /
//       data-gosx-studio-preview / any <iframe> markup is ABSENT.
//   (3) NO COMPONENT/RESOURCE LABELS — derived from the bundle: bundle.labels
//       carry only page card titles + routes, never the canonical
//       component/resource node titles in the seeded site map.
//
// Honesty discipline (matching the sibling canvas tests): the bundle is the live
// JSON the server precomputed and the client paints — nothing is hardcoded beyond
// the canonical seeded site-map node titles the editor authors. If the board is
// NOT pages-only or the iframe is still present, the offending assertion fails
// with the observed evidence rather than being weakened.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy dist rebuilt against the
// gosx/gosx-studio under test (`gosx build .` in muddy-noni-commerce).

const BUNDLE_SELECTOR = "[data-gosx-canvas-bundle]";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";

// Canonical non-page node titles in the editor's seeded site map (components +
// resources + flows). Under PageArtboardsOnly NONE of these may appear in the
// bundle labels — the board shows page artboards only. Mirrors the muddy
// TestEditorSiteMapCanvasIsPageArtboardsOnly forbidden list.
const FORBIDDEN_NON_PAGE_LABELS = [
  "Hero content", "Products content", "Product grid", "Category filter",
  "Product collection", "Product categories", "Checkout Handoff flow",
  "Studio notes", "Contact band", "Gallery grid", "Post list",
];

// Legacy iframe-preview panel markers (M9 removed all of these from the editor).
const IFRAME_PREVIEW_MARKERS = [
  "[data-studio-preview-overlay]",
  ".studio-preview-overlay",
  "[data-studio-preview-frame]",
  "iframe[data-studio-preview-frame]",
  "[data-gosx-studio-preview]",
  ".editor-preview-frame",
  ".editor-preview-shell",
];

type CanvasBundle = {
  parsed: boolean;
  scriptCount: number;
  rawLen: number;
  lines: unknown[];
  objects: Array<{ id?: string; kind?: string }>;
  labels: Array<{ text?: string }>;
  reason: string;
};

test.describe("@reference-apps canvas2d editor is pure canvas-first (pages-only + single surface)", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni editor: the Canvas2D board renders page artboards only (no spiderweb), is the single main surface, and the legacy iframe preview is gone", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });

      // The editor shell must be present (proves we got the editor route).
      await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();

      // The canvas board surface must be emitted (the single main editing surface).
      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(
        canvas,
        "the WASM-free Canvas2D board surface must be emitted as the single main editing surface",
      ).toBeAttached({ timeout: 30_000 });

      // The inline server-precomputed bundle <script> must be present (the canvas
      // engine emits exactly one). Read it straight off the live DOM.
      const bundleScript = page.locator(BUNDLE_SELECTOR).first();
      await expect(
        bundleScript,
        "the inline [data-gosx-canvas-bundle] server bundle script must be emitted",
      ).toBeAttached({ timeout: 30_000 });

      // ── (2a) ONE SURFACE: exactly one canvas board + exactly one bundle ──────────
      const bundleCount = await page.locator(BUNDLE_SELECTOR).count();
      expect(
        bundleCount,
        `the editor main area must render exactly one canvas board bundle; found ${bundleCount}`,
      ).toBe(1);
      const canvasCount = await page.locator(CANVAS_SELECTOR).count();
      expect(
        canvasCount,
        `the editor main area must render exactly one canvas board surface; found ${canvasCount}`,
      ).toBe(1);
      // The DOM site-map board element (selection sink) stays — single, not zero.
      await expect(page.locator(BOARD_SELECTOR).first()).toBeAttached();

      // ── (2b) NO IFRAME PREVIEW: every legacy iframe-preview marker absent ────────
      for (const marker of IFRAME_PREVIEW_MARKERS) {
        const count = await page.locator(marker).count();
        expect(
          count,
          `legacy iframe-preview marker ${marker} must be ABSENT (M9 removed the iframe preview panel), found ${count}`,
        ).toBe(0);
      }
      // No <iframe> at all in the editor (the only engine surface is the canvas board).
      const iframeCount = await page.locator("iframe").count();
      expect(
        iframeCount,
        `the editor must contain NO <iframe> (the canvas board is the single surface), found ${iframeCount}`,
      ).toBe(0);

      // ── (1) NO SPIDERWEB + page cards present: parse the inline bundle JSON ──────
      const bundle = await readCanvasBundle(page);
      expect(
        bundle.parsed,
        `the inline ${BUNDLE_SELECTOR} JSON must parse; reason=${bundle.reason}, rawLen=${bundle.rawLen}`,
      ).toBe(true);
      // Exactly one bundle script carried the JSON (cross-check of single surface).
      expect(
        bundle.scriptCount,
        `exactly one inline canvas bundle script must be present; found ${bundle.scriptCount}`,
      ).toBe(1);

      // ZERO link lines: PageArtboardsOnly drops the inter-node "spiderweb".
      // The marshaled "lines" key is omitempty, so zero lines == absent/empty.
      expect(
        bundle.lines.length,
        `pure canvas-first board must carry ZERO spiderweb lines (no inter-node connectors); got ${bundle.lines.length}: ${JSON.stringify(bundle.lines)}`,
      ).toBe(0);

      // Page cards present: the bundle must carry page-card rect objects (otherwise
      // an empty board would trivially satisfy "no lines").
      const rectCards = bundle.objects.filter((o) => (o?.kind ?? "") === "rect");
      expect(
        rectCards.length,
        `pure canvas-first board must still render page artboard cards (rect objects); got ${rectCards.length} of ${bundle.objects.length} objects`,
      ).toBeGreaterThan(0);

      // ── (3) NO COMPONENT/RESOURCE LABELS (derived from the bundle) ───────────────
      const labelTexts = bundle.labels.map((l) => (l?.text ?? "").trim());
      expect(
        labelTexts.length,
        "the bundle must still carry page card labels; got none",
      ).toBeGreaterThan(0);
      const leakedNonPageLabels = labelTexts.filter((t) => FORBIDDEN_NON_PAGE_LABELS.includes(t));
      expect(
        leakedNonPageLabels,
        `pure canvas-first board must drop all component/resource node labels; leaked: ${JSON.stringify(leakedNonPageLabels)} (all labels: ${JSON.stringify(labelTexts)})`,
      ).toEqual([]);
      // Sanity: a page card label survives (we filtered to pages, not all nodes).
      // Page artboard labels are titles (e.g. "Shop") and/or routes (e.g. "/shop").
      const sawPageLabel = labelTexts.some((t) => t === "Shop" || t === "/shop" || /^\/(?:[a-z0-9-]+)?$/.test(t));
      expect(
        sawPageLabel,
        `pure canvas-first board must keep page card labels (a title/route survives); labels=${JSON.stringify(labelTexts)}`,
      ).toBe(true);

      // No unexpected console errors that mention the canvas path.
      const canvasErrors = consoleErrors.filter((line) => /canvas|surface|hydrate|sitemap|site-map|painter|__gosx_render/i.test(line) && !/relay\.js/i.test(line));
      expect(canvasErrors, `unexpected canvas console errors: ${JSON.stringify(canvasErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// readCanvasBundle parses every inline [data-gosx-canvas-bundle] script off the
// live DOM, returning the (single) parsed bundle's lines/objects/labels plus a
// count so the caller can prove single-surface and inspect the shape.
async function readCanvasBundle(page: Page): Promise<CanvasBundle> {
  return page.evaluate((sel) => {
    const scripts = Array.from(document.querySelectorAll(sel));
    const empty = {
      parsed: false,
      scriptCount: scripts.length,
      rawLen: 0,
      lines: [] as unknown[],
      objects: [] as Array<{ id?: string; kind?: string }>,
      labels: [] as Array<{ text?: string }>,
      reason: "no-bundle-script",
    };
    if (scripts.length === 0) return empty;
    const raw = scripts[0].textContent || "";
    empty.rawLen = raw.length;
    let parsed: {
      lines?: unknown[];
      objects?: Array<{ id?: string; kind?: string }>;
      labels?: Array<{ text?: string }>;
    } | null = null;
    try {
      parsed = JSON.parse(raw);
    } catch (error) {
      return { ...empty, reason: `json-parse-threw:${String(error)}` };
    }
    if (!parsed || typeof parsed !== "object") {
      return { ...empty, reason: "bundle-not-object" };
    }
    return {
      parsed: true,
      scriptCount: scripts.length,
      rawLen: raw.length,
      lines: Array.isArray(parsed.lines) ? parsed.lines : [],
      objects: Array.isArray(parsed.objects) ? parsed.objects : [],
      labels: Array.isArray(parsed.labels) ? parsed.labels : [],
      reason: "parsed",
    };
  }, BUNDLE_SELECTOR);
}
