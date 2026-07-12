import { expect, test, type Page } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the editor's default main surface (defect #8 fix).
//
// SUPERSEDED CONTRACT: this file previously asserted the M9 "pure canvas-first"
// shape — the Canvas2D site-map board (PageArtboardsOnly) as the editor's ONE
// main surface, with the legacy iframe-preview panel entirely gone
// ([data-studio-preview-frame] etc. asserted ABSENT). The studio-pagecanvas-
// handoff branch has since superseded that contract: the editor's default Home
// surface is now the PageCanvas ([data-gosx-studio-page-canvas="true"]), a real
// same-origin iframe of the live storefront ([data-studio-preview-frame]) — NOT
// the Canvas2D site-map board. The Canvas2D board still exists (moved behind
// the "Advanced" mode panel, [data-studio-advanced-canvas-board="true"]) and
// keeps its own still-true facts (page-artboards-only, no spiderweb, one
// surface, no embedded iframe of its own) — those assertions are preserved
// below, just scoped to Advanced mode instead of the removed default.
//
// This file now asserts the NEW default-surface contract end-to-end in
// headless Chromium:
//
//   (1) DEFAULT HOME SURFACE IS PAGECANVAS — with no mode reveal, the
//       PageCanvas section is attached AND visible, and its
//       [data-studio-preview-frame] iframe is attached, visible, and
//       FUNCTIONAL (it loads the live storefront route — proven by polling the
//       frame's own contentDocument for real rendered text, not just
//       DOM/attribute presence).
//   (2) OLD CENTER-FORM CANVAS IS NOT THE DEFAULT — the legacy Canvas2D
//       site-map board ([data-studio-advanced-canvas-board]) is hidden by
//       default (gated CSS, .editor-workbench:not([data-studio-mode="advanced"])
//       .studio-advanced-canvas-board { display:none }), the mirror image of
//       PageCanvas being hidden once Advanced is revealed
//       (.editor-workbench[data-studio-mode="advanced"] .studio-page-canvas
//       { display:none }) — the two are mutually exclusive by design.
//   (3) STILL-VALID CANVAS2D FACTS (scoped to Advanced mode): page artboards
//       only (bundle.lines empty — no inter-node "spiderweb"), exactly one
//       board surface + one bundle script, no component/resource node labels
//       leaked, and the legacy board does not embed its own iframe.
//
// Honesty discipline (unchanged from the sibling canvas tests): the bundle is
// the live JSON the server precomputed and the client paints — nothing is
// hardcoded beyond the canonical seeded site-map node titles the editor
// authors. The preview-frame "functional" proof reads the frame's OWN
// contentDocument, not a hardcoded string.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const WORKBENCH_SELECTOR = "[data-studio-workbench='true']";
const PAGE_CANVAS_SELECTOR = "[data-gosx-studio-page-canvas='true']";
const PREVIEW_FRAME_SELECTOR = "[data-studio-preview-frame]";
const ADVANCED_BOARD_SELECTOR = "[data-studio-advanced-canvas-board='true']";
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

type CanvasBundle = {
  parsed: boolean;
  scriptCount: number;
  rawLen: number;
  lines: unknown[];
  objects: Array<{ id?: string; kind?: string }>;
  labels: Array<{ text?: string }>;
  reason: string;
};

test.describe("@reference-apps PageCanvas is the default Home surface (canvas2d board lives behind Advanced)", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni editor: PageCanvas + functional preview frame are the default, the legacy Canvas2D board is Advanced-only", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });

      // The editor shell must be present (proves we got the editor route).
      await expect(page.locator(WORKBENCH_SELECTOR).first()).toBeAttached();

      // ── (1) DEFAULT HOME SURFACE IS PAGECANVAS ────────────────────────────────
      const pageCanvas = page.locator(PAGE_CANVAS_SELECTOR).first();
      await expect(pageCanvas, "PageCanvas must be emitted as the editor's default Home surface").toBeAttached({ timeout: 30_000 });
      await expect(pageCanvas, "PageCanvas must be VISIBLE by default (no mode reveal)").toBeVisible();

      const previewFrame = page.locator(PREVIEW_FRAME_SELECTOR).first();
      await expect(previewFrame, "the preview iframe must be attached inside the default PageCanvas surface").toBeAttached({ timeout: 30_000 });
      await hydrateDeferredPreviewFrame(page);
      await previewFrame.scrollIntoViewIfNeeded();
      await expect(previewFrame, "the preview iframe must be visible by default").toBeVisible();

      // Functional, not just present: the frame must have actually navigated
      // to and rendered the live storefront route (read straight off the
      // frame's own contentDocument — nothing hardcoded).
      await page.waitForFunction(() => {
        const frame = document.querySelector("[data-studio-preview-frame]");
        if (!(frame instanceof HTMLIFrameElement)) return false;
        const body = frame.contentDocument?.body;
        return !!body && body.innerText.trim().length > 10;
      }, null, { timeout: 45_000 });
      const previewState = await previewFrame.getAttribute("data-studio-preview-state");
      expect(previewState, "the preview iframe must report a ready/loaded state after navigating").toBe("ready");

      // ── (2) OLD CENTER-FORM CANVAS IS NOT THE DEFAULT ─────────────────────────
      // The legacy Canvas2D site-map board now lives behind "Advanced" and must
      // NOT be the visible default surface (mirrors the mutual-exclusivity CSS:
      // studio.css hides .studio-advanced-canvas-board unless
      // data-studio-mode="advanced", and hides .studio-page-canvas WHEN it is).
      await expect(
        page.locator(ADVANCED_BOARD_SELECTOR),
        "the legacy Canvas2D board must be hidden by default — PageCanvas is the default surface now",
      ).toBeHidden();

      // ── (3) STILL-VALID CANVAS2D FACTS, scoped to Advanced mode ──────────────
      await revealModeIfPresent(page, "advanced");

      // Revealing Advanced hides PageCanvas (mutually exclusive by CSS) —
      // cross-checking the mirror-image rule from step (2) above.
      await expect(pageCanvas, "revealing Advanced must hide PageCanvas (mutually exclusive panels)").toBeHidden();

      const advancedBoard = page.locator(ADVANCED_BOARD_SELECTOR).first();
      await expect(advancedBoard, "the legacy Canvas2D board must be visible once Advanced is revealed").toBeVisible();

      const canvas = page.locator(CANVAS_SELECTOR).first();
      await expect(
        canvas,
        "the WASM-free Canvas2D board surface must be emitted inside the Advanced board",
      ).toBeAttached({ timeout: 30_000 });

      const bundleScript = page.locator(BUNDLE_SELECTOR).first();
      await expect(
        bundleScript,
        "the inline [data-gosx-canvas-bundle] server bundle script must be emitted",
      ).toBeAttached({ timeout: 30_000 });

      // ONE SURFACE within the Advanced board (still true: one board, one bundle).
      const bundleCount = await page.locator(BUNDLE_SELECTOR).count();
      expect(
        bundleCount,
        `the Advanced board must render exactly one canvas board bundle; found ${bundleCount}`,
      ).toBe(1);
      const canvasCount = await page.locator(CANVAS_SELECTOR).count();
      expect(
        canvasCount,
        `the Advanced board must render exactly one canvas board surface; found ${canvasCount}`,
      ).toBe(1);
      await expect(page.locator(BOARD_SELECTOR).first()).toBeAttached();

      // The legacy board does not embed its own iframe (narrower, still-true
      // version of the old blanket "editor has no iframe" claim — that claim
      // is false at the document level now that PageCanvas legitimately has
      // one, but it stays true SCOPED to the legacy board itself).
      const nestedIframeCount = await page.locator(`${ADVANCED_BOARD_SELECTOR} iframe`).count();
      expect(
        nestedIframeCount,
        `the legacy Canvas2D board must not embed its own iframe, found ${nestedIframeCount}`,
      ).toBe(0);

      // NO SPIDERWEB + page cards present: parse the inline bundle JSON.
      const bundle = await readCanvasBundle(page);
      expect(
        bundle.parsed,
        `the inline ${BUNDLE_SELECTOR} JSON must parse; reason=${bundle.reason}, rawLen=${bundle.rawLen}`,
      ).toBe(true);
      expect(
        bundle.scriptCount,
        `exactly one inline canvas bundle script must be present; found ${bundle.scriptCount}`,
      ).toBe(1);
      expect(
        bundle.lines.length,
        `pure canvas-first board must carry ZERO spiderweb lines (no inter-node connectors); got ${bundle.lines.length}: ${JSON.stringify(bundle.lines)}`,
      ).toBe(0);

      const rectCards = bundle.objects.filter((o) => (o?.kind ?? "") === "rect");
      expect(
        rectCards.length,
        `Advanced board must still render page artboard cards (rect objects); got ${rectCards.length} of ${bundle.objects.length} objects`,
      ).toBeGreaterThan(0);

      const labelTexts = bundle.labels.map((l) => (l?.text ?? "").trim());
      expect(
        labelTexts.length,
        "the bundle must still carry page card labels; got none",
      ).toBeGreaterThan(0);
      const leakedNonPageLabels = labelTexts.filter((t) => FORBIDDEN_NON_PAGE_LABELS.includes(t));
      expect(
        leakedNonPageLabels,
        `Advanced board must drop all component/resource node labels; leaked: ${JSON.stringify(leakedNonPageLabels)} (all labels: ${JSON.stringify(labelTexts)})`,
      ).toEqual([]);
      const sawPageLabel = labelTexts.some((t) => t === "Shop" || t === "/shop" || /^\/(?:[a-z0-9-]+)?$/.test(t));
      expect(
        sawPageLabel,
        `Advanced board must keep page card labels (a title/route survives); labels=${JSON.stringify(labelTexts)}`,
      ).toBe(true);

      // No unexpected console errors that mention the canvas/preview paths.
      const canvasErrors = consoleErrors.filter((line) => /canvas|surface|hydrate|sitemap|site-map|painter|preview|__gosx_render/i.test(line) && !/relay\.js/i.test(line));
      expect(canvasErrors, `unexpected canvas/preview console errors: ${JSON.stringify(canvasErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// hydrateDeferredPreviewFrame sets the iframe's src from its
// data-studio-preview-src/host data-gosx-studio-preview-url fallback if the
// server rendered it deferred (no src yet) — mirrors
// reference_apps_performance_test.ts's identical helper.
async function hydrateDeferredPreviewFrame(page: Page) {
  await page.evaluate(() => {
    const node = document.querySelector("[data-studio-preview-frame]");
    if (!(node instanceof HTMLIFrameElement)) return;
    if (node.getAttribute("src")) return;
    const container = node.closest("[data-gosx-studio-preview]");
    const nextURL = node.getAttribute("data-studio-preview-src") ||
      container?.getAttribute("data-gosx-studio-preview-url") ||
      "/";
    node.setAttribute("src", nextURL);
  });
}

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
