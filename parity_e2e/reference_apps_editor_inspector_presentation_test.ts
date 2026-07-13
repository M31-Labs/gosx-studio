import { expect, test, type APIRequestContext } from "@playwright/test";
import { mkdtempSync, copyFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { openInteractionsTargetDisclosure, revealModeIfPresent, startMuddy, startMuddyCollaboration, type ServerHandle } from "./reference_apps_harness";

// Wave 3A (inspector-presentation): the post-wave-2 re-score (6/10) found the
// entire gap to 7-8 was the Home inspector's presentation once real content
// volume is on the page, not a missing feature. A dev-seed HOME page (one or
// two canvas targets) never reproduces this — the defect only shows up with
// a realistic content volume (multiple pages, each with a real canvas
// target), which is why this spec seeds MUDDY_DATA_PATH from
// fixtures/noni_staging_seed.json (the staging content snapshot the review
// was run against: 6 pages, 15 canvas targets) instead of the harness's
// default empty tempDir.
//
// Verified against a live scratch build before writing these assertions
// (not just static CSS/Go reasoning — see the handoff report): with this
// seed, HOME went from a 16,540px page with 114 always-visible selects and
// 2 visible "Responsive layout" headings down to a 5,158px page with 0
// visible selects (all tucked behind closed-by-default <details>) and 1
// visible "Responsive layout" heading. The numeric budgets below are set
// with headroom above the fixed (~5,200px / 0 select) result so the guard
// fails on a real regression, not on every unrelated content tweak.
async function startMuddyWithStagingSeed(request: APIRequestContext): Promise<ServerHandle> {
  const seedDir = mkdtempSync(path.join(tmpdir(), "gosx-studio-inspector-presentation-seed-"));
  const seedPath = path.join(seedDir, "cms.json");
  copyFileSync(path.join(__dirname, "fixtures", "noni_staging_seed.json"), seedPath);
  const server = await startMuddy(request, { MUDDY_DATA_PATH: seedPath });
  const stop = server.stop.bind(server);
  return {
    ...server,
    stop: async () => {
      await stop();
      rmSync(seedDir, { force: true, recursive: true });
    },
  };
}

// startMuddyCollaborationWithStagingSeed mirrors startMuddyWithStagingSeed but
// boots with a live, authenticated collaboration transport (handoff-31): the
// interactions row's Save/Add button now dispatches through the durable
// operation protocol (operationruntime/island_runtime.js's
// [data-gosx-studio-durable-history]), and collabruntime's permissions() gate
// disables that button entirely until a real websocket handshake grants
// capabilities — the plain startMuddy harness above never establishes one
// (MUDDY_MOCK_AUTH only covers plain HTTP action handlers, not the
// collaboration websocket's own auth; see
// reference_apps_interactions_test.ts's header comment for the full
// rationale), so any test that clicks that button needs this instead.
async function startMuddyCollaborationWithStagingSeed(request: APIRequestContext): Promise<ServerHandle> {
  const seedDir = mkdtempSync(path.join(tmpdir(), "gosx-studio-inspector-presentation-collab-seed-"));
  const seedPath = path.join(seedDir, "cms.json");
  copyFileSync(path.join(__dirname, "fixtures", "noni_staging_seed.json"), seedPath);
  const server = await startMuddyCollaboration(request, { MUDDY_DATA_PATH: seedPath });
  const stop = server.stop.bind(server);
  return {
    ...server,
    stop: async () => {
      await stop();
      rmSync(seedDir, { force: true, recursive: true });
    },
  };
}

// isRendered mirrors Element.checkVisibility(): a real regression guard for
// this class of defect cannot use a plain getBoundingClientRect()+display
// check (the "raw wall" fix's own verification found that a closed native
// <details> keeps a non-zero bounding rect for its collapsed content in
// headless Chromium — only checkVisibility()/Playwright's own
// locator.isVisible() correctly resolve it, confirmed against an isolated
// <details><p>/<select> repro plus a screenshot proving the content isn't
// actually painted).
const isRenderedFn = `(el) => typeof el.checkVisibility === "function" ? el.checkVisibility() : el.offsetParent !== null`;

test.describe("@reference-apps Muddy/Noni inspector presentation (wave 3A)", () => {
  // The second test below drives a real authoring round trip (attach +
  // in-place fragment refresh, then a full networkidle reload) against the
  // 15-target staging seed, on top of the editor boot itself — the default
  // 30s Playwright test timeout is too tight for that combination on this
  // fixture size. Matches the longer budget the sibling interaction specs
  // (reference_apps_interactions_test.ts et al.) already use for the same
  // reason.
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("HOME with realistic content volume stays under a height and visible-select budget", async ({ page, request }) => {
    const server = await startMuddyWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
      await page.waitForTimeout(1_500);

      const metrics = await page.evaluate(([isRenderedSrc]) => {
        // eslint-disable-next-line no-eval
        const isRendered = eval(isRenderedSrc) as (el: Element) => boolean;
        const visibleSelects = [...document.querySelectorAll("select")].filter(isRendered).length;
        const respLayoutVisibleHeadings = [...document.querySelectorAll("h1,h2,h3")].filter(
          (h) => /responsive layout/i.test(h.textContent || "") && isRendered(h),
        ).length;
        const interactionTargetsRendered = document.querySelectorAll('[data-studio-interactions-target="true"]').length;
        const interactionEditorsOpenByDefault = document.querySelectorAll(".studio-interactions-target__editor[open]").length;
        const readyFieldChipsVisible = [...document.querySelectorAll(".studio-home-field__meta output")].filter(isRendered).length;
        return {
          homeHeightPx: document.body.scrollHeight,
          visibleSelects,
          respLayoutVisibleHeadings,
          interactionTargetsRendered,
          interactionEditorsOpenByDefault,
          readyFieldChipsVisible,
        };
      }, [isRenderedFn] as const);

      // Fix (c)/(a): the whole point of collapsing per-target cards is a
      // bounded HOME height regardless of how many canvas targets the site
      // has. The pre-fix baseline on this exact seed was 16,540px; budget at
      // well under half that, matching the task's "well under 6000px" target.
      expect(metrics.homeHeightPx, "HOME height budget (collapsed-by-default inspector sections)").toBeLessThan(6_000);

      // Fix (a): every interaction target's Effect/Duration/Delay selects
      // must be closed-by-default, not pre-expanded. The pre-fix baseline
      // was 114 always-visible selects; a real regression back to
      // "raw wall" would blow well past a single-digit budget.
      expect(metrics.visibleSelects, "visible <select> budget (interactions/responsive-layout collapsed by default)").toBeLessThan(10);

      // Fix (b): the studio-side idempotence guard must leave exactly one
      // "Responsive layout" heading visible even though the host still
      // composes one per flagship target.
      expect(metrics.respLayoutVisibleHeadings, "exactly one visible \"Responsive layout\" heading").toBe(1);

      // Fix (a): with 15 canvas targets in this seed, every one of them
      // must render as a compact card (not silently dropped).
      expect(metrics.interactionTargetsRendered, "every canvas target renders as a compact card").toBeGreaterThanOrEqual(15);
      expect(metrics.interactionEditorsOpenByDefault, "no interaction target's editor is pre-expanded").toBe(0);

      // Fix (e): no per-field "READY" vocabulary chip on a valid field.
      expect(metrics.readyFieldChipsVisible, "no visible per-field READY-style chips").toBe(0);
    } finally {
      await server.stop();
    }
  });

  test("an interaction target's disclosure expands independently without affecting its siblings", async ({ page, request }) => {
    const server = await startMuddyCollaborationWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/__test/collaboration/signin/designer`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const collabPanel = page.locator("[data-gosx-studio-collaboration='true']");
      await expect(
        collabPanel,
        "the collaboration runtime must connect before the durable operation button enables",
      ).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });

      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
      await expect(collabPanel).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      await page.waitForTimeout(1_500);

      // This fixture's staging seed has ZERO pre-attached interactions (see
      // the file header comment above), and RenderInteractionsInspector
      // nests every target with zero attached interactions behind one
      // shared "+ Add interaction" picker by design (fix (a)/(d) — the whole
      // point is not pre-expanding N empty control sets at once). Proving
      // "one target's disclosure expands independently" against the real
      // wiring therefore means first attaching one interaction the same way
      // an operator would (the exact flow
      // reference_apps_interactions_test.ts already drives end to end),
      // which promotes that target out of the shared picker into the
      // top-level `.studio-interactions__list` — the realistic shape this
      // assertion is actually about.
      const heroPanel = "#studio-interactions-home-hero";
      const revealForm = `${heroPanel} article[data-studio-interaction-card='reveal-on-scroll'] form[data-studio-interaction-row='reveal-on-scroll']`;
      await expect(page.locator(revealForm), "the hero reveal-on-scroll interaction row must render").toBeAttached({ timeout: 15_000 });
      await openInteractionsTargetDisclosure(page, heroPanel);
      // Commit through the durable operation protocol (handoff-31) instead
      // of the retired plain-AJAX POST clickAuthoringPanel waited on —
      // reload explicitly afterward with the same domcontentloaded +
      // generous-timeout pattern gotoEditor/reference_apps_interactions_test.ts
      // already use (this 15-target staging seed page has enough ongoing
      // background asset activity that a strict networkidle wait isn't
      // reliable within the config's 30s navigationTimeout).
      const committed = page.evaluate(
        () => new Promise<{ kind: "committed" | "error" }>((resolve) => {
          document.addEventListener("gosxstudio:operation-committed", () => resolve({ kind: "committed" }), { once: true });
          document.addEventListener("gosxstudio:operation-error", () => resolve({ kind: "error" }), { once: true });
        }),
      );
      await page.locator(`${revealForm} [data-gosx-studio-operation-kind='set-interaction']`).click();
      const outcome = await committed;
      expect(outcome.kind, "attaching the hero interaction must not fail server-side").toBe("committed");
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await page.waitForTimeout(1_500);

      const targets = page.locator('[data-studio-interactions-target="true"]');
      const count = await targets.count();
      expect(count, "staging seed should produce multiple canvas targets").toBeGreaterThanOrEqual(15);

      // RenderInteractionsInspector lists every attached target ahead of the
      // shared add picker, so the just-attached hero target (now a
      // top-level card, no longer nested inside any outer disclosure) is
      // the first match.
      const first = targets.first();
      await expect(first, "the newly-attached hero target should render as the first top-level card").toHaveAttribute("id", "studio-interactions-home-hero");
      const firstEditor = first.locator(".studio-interactions-target__editor");
      const firstSelects = firstEditor.locator("select");
      await expect(firstSelects.first()).toBeHidden();

      await firstEditor.locator("summary").first().click();
      await expect(firstSelects.first()).toBeVisible();

      // A sibling target's controls must stay collapsed — expanding one
      // target's disclosure is not a "expand everything" global toggle. This
      // sibling is still unattached (nested inside the shared "+ Add
      // interaction" picker), which is itself further proof the two kinds
      // of disclosure (the shared picker and each target's own editor) are
      // independent of one another.
      const second = targets.nth(1);
      await expect(second.locator(".studio-interactions-target__editor select").first()).toBeHidden();
    } finally {
      await server.stop();
    }
  });
});

// Wave 3B (tamarind): the magnolia-3 re-score (7/10) found the presentation
// gap wave 3A fixed for HOME's own inspector repeated across the rest of the
// editor surface: ADVANCED measured 9,497px with 18 always-visible <select>
// elements and 4 duplicate "Fields, validation, and binding" panels; HOME's
// support-node region below the workbench stacked 5 duplicate "Selected
// content" direct-edit forms across 3 different, accidental panel widths
// (483px/1182px/1440px) plus an always-composed (never mode-gated)
// color-token/font editor and one completely blank bordered box. This spec
// extends the wave 3A budget guard with the ADVANCED-mode equivalent plus
// the new fixes: direct-edit dedupe (generalizing wave 3A's
// scopeResponsiveLayoutInspectors into scopeSelectionPanels/
// scopeDirectEditPanels, hostruntime/assets/workbench_runtime.js), one
// consistent support-panel width (--size-studio-support-panel, 62.5rem),
// and the LOOK-only style/theme panel mode-gate (shell/backend_editor_page.go).
//
// Root-cause note on the "one empty bordered box" finding (verified live,
// not guessed): it was NOT a Go-side "render nothing when empty" gap. It was
// muddy-noni-commerce's own CSS (public/site.css) applying
// `content-visibility: auto; contain-intrinsic-size: 8rem` to `.editor-panel`
// — the exact class the mode-gated Interactions/direct-edit/flow-field-editor
// support-node wrappers carry. `contain-intrinsic-size: 8rem` (128px) is far
// smaller than these panels' real rendered height (the Interactions panel
// alone measures ~185px on this seed), and a panel far enough down the page
// (Interactions starts around y=2476 on the realistic 15-canvas-target seed)
// never gets promoted "relevant to the user" before a full-page capture,
// painting as a totally blank rounded-rect despite
// "Interactions"/"Attach safe, typed behavior..." genuinely being in the DOM
// (confirmed: `textContent` had the real copy, `innerText` was `""` —
// Chromium's content-visibility "skipped" signature — and a pixel crop of
// the rendered page exactly matched a blank box). The fix removed
// `.editor-panel` from that content-visibility rule (composition-level,
// muddy-side; left uncommitted per the handoff, see the report). This spec
// guards the OUTCOME (the panel actually paints its own text), not the CSS
// property directly, since the property lives in the host, not gosx-studio.
test.describe("@reference-apps Muddy/Noni editor surface presentation (wave 3B)", () => {
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("ADVANCED with realistic content volume stays under a height and visible-select budget", async ({ page, request }) => {
    const server = await startMuddyWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
      await page.waitForTimeout(1_500);
      await revealModeIfPresent(page, "advanced");
      await page.waitForTimeout(1_500);

      const metrics = await page.evaluate(([isRenderedSrc]) => {
        // eslint-disable-next-line no-eval
        const isRendered = eval(isRenderedSrc) as (el: Element) => boolean;
        const visibleSelects = [...document.querySelectorAll("select")].filter(isRendered).length;
        const flowFieldHeaderCount = [...document.querySelectorAll("h3")].filter(
          (h) => (h.textContent || "").trim() === "Fields, validation, and binding" && isRendered(h),
        ).length;
        const flowFieldCardCount = document.querySelectorAll(".studio-flow-field-editor").length;
        const flowFieldEditorsOpenByDefault = document.querySelectorAll(".studio-flow-field-editor__editor[open]").length;
        return {
          advancedHeightPx: document.body.scrollHeight,
          visibleSelects,
          flowFieldHeaderCount,
          flowFieldCardCount,
          flowFieldEditorsOpenByDefault,
        };
      }, [isRenderedFn] as const);

      // Fix #5 (ADVANCED discipline): the pre-fix baseline on this exact
      // seed was 9,497px; budget well under the task's 5,000px target.
      expect(metrics.advancedHeightPx, "ADVANCED height budget (collapsed-by-default flow field editor)").toBeLessThan(5_000);

      // Fix #5: the pre-fix baseline was 18 always-visible selects (4 flow
      // actions x [4-5 Kind selects each]). One legitimate select remains on
      // this seed regardless of this fix: the Stripe checkout product
      // picker (`#checkoutProductId`), an ADVANCED sitemap/commerce control
      // unrelated to the flow field editor and out of this fix's scope —
      // budget with headroom above that honest floor, not zero.
      expect(metrics.visibleSelects, "visible <select> budget (flow field editor collapsed by default)").toBeLessThan(6);

      // Fix #5 (dedupe): exactly one "Fields, validation, and binding"
      // header for the whole aggregate inspector, not one per flow action.
      expect(metrics.flowFieldHeaderCount, "exactly one visible \"Fields, validation, and binding\" header").toBe(1);

      // This seed's flow catalog (contact, purchase-request,
      // checkout-handoff, newsletter) — every action still renders as its
      // own compact card, just not one per header.
      expect(metrics.flowFieldCardCount, "every flow action renders as a compact card").toBeGreaterThanOrEqual(4);
      expect(metrics.flowFieldEditorsOpenByDefault, "no flow action's editor is pre-expanded").toBe(0);
    } finally {
      await server.stop();
    }
  });

  test("HOME's direct-edit panels are scoped to the active selection (dedupe)", async ({ page, request }) => {
    const server = await startMuddyWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
      await page.waitForTimeout(1_500);

      const metrics = await page.evaluate(([isRenderedSrc]) => {
        // eslint-disable-next-line no-eval
        const isRendered = eval(isRenderedSrc) as (el: Element) => boolean;
        const panels = [...document.querySelectorAll("[data-studio-direct-edit-panel]")];
        return {
          totalPanels: panels.length,
          visiblePanels: panels.filter(isRendered).length,
          // The label's own textContent also picks up its sibling
          // <textarea>'s value text (they share one <label> element — see
          // panels/direct_edit_panel.go), so match the inner <span> alone,
          // not the whole label.
          visibleLabels: [...document.querySelectorAll("[data-studio-direct-edit-field] span")].filter(
            (s) => /^selected content$/i.test((s.textContent || "").trim()) && isRendered(s),
          ).length,
        };
      }, [isRenderedFn] as const);

      // The host (editorDirectEditPanels) still composes all 5 (home:hero,
      // about:content, and 3 contact:contact-links fields) unconditionally —
      // this is the studio-side idempotence guard's job, not a host rewrite.
      expect(metrics.totalPanels, "host still composes every target's direct-edit panel").toBeGreaterThanOrEqual(5);

      // Fix #2: with no explicit canvas selection yet, only the
      // first-composed target's panel(s) — home:hero, a single panel — are
      // visible. The magnolia-3 re-score's baseline was 5 simultaneously
      // visible "Selected content" forms.
      expect(metrics.visiblePanels, "only the active selection's direct-edit panel(s) are visible").toBe(1);
      expect(metrics.visibleLabels, "only one visible \"Selected content\" label").toBe(1);
    } finally {
      await server.stop();
    }
  });

  test("support-node panels share one consistent width", async ({ page, request }) => {
    const server = await startMuddyWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
      await page.waitForTimeout(1_500);

      const widths = await page.evaluate(() => {
        const isRendered = (el: Element) => (el as HTMLElement).checkVisibility ? (el as HTMLElement).checkVisibility() : true;
        return [...document.querySelectorAll("[data-studio-mode-gate-wrapper]")]
          .filter(isRendered)
          .map((el) => Math.round(el.getBoundingClientRect().width));
      });

      // Fix #1 (width system): the magnolia-3 re-score measured 483px
      // (direct-edit stack) vs ~1182-1244px (Interactions) vs full-bleed
      // elsewhere on HOME alone. Every visible support-node wrapper must now
      // share one width (--size-studio-support-panel).
      expect(widths.length, "at least one support-node panel is visible on HOME").toBeGreaterThan(0);
      const distinct = [...new Set(widths)];
      expect(distinct.length, `expected one consistent support-panel width, got ${JSON.stringify(widths)}`).toBe(1);
    } finally {
      await server.stop();
    }
  });

  test("the color-token/font style panel only renders in LOOK mode", async ({ page, request }) => {
    const server = await startMuddyWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
      await page.waitForTimeout(1_500);

      const isStylePanelVisible = () => page.evaluate(() => {
        const panel = document.querySelector("[data-gosx-studio-style-panel]");
        if (!panel) return false;
        return (panel as HTMLElement).checkVisibility ? (panel as HTMLElement).checkVisibility() : true;
      });

      // Fix #3: previously composed with no data-studio-mode-panel at all,
      // so it rendered unconditionally (confirmed live: visible under
      // HOME/LOOK/PREVIEW/PUBLISH/ADVANCED alike before this fix).
      expect(await isStylePanelVisible(), "style panel must be hidden on HOME").toBe(false);

      await revealModeIfPresent(page, "look");
      await page.waitForTimeout(500);
      expect(await isStylePanelVisible(), "style panel must be visible on LOOK").toBe(true);

      await revealModeIfPresent(page, "publish");
      await page.waitForTimeout(500);
      expect(await isStylePanelVisible(), "style panel must be hidden on PUBLISH").toBe(false);
    } finally {
      await server.stop();
    }
  });

  // Regression guard for the "one empty bordered box" root cause (see the
  // file-level comment above): a mode-gated support panel far down the page
  // must actually paint its own header text, not just have it present but
  // unrendered in the DOM (content-visibility:auto's "skipped" state).
  test("the Interactions panel actually paints its header text (content-visibility regression guard)", async ({ page, request }) => {
    const server = await startMuddyWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
      await page.waitForTimeout(1_500);

      const rendered = await page.evaluate(() => {
        const h3 = [...document.querySelectorAll("h3")].find((h) => (h.textContent || "").trim() === "Interactions");
        if (!h3) return { found: false };
        return { found: true, innerText: (h3 as HTMLElement).innerText, textContent: h3.textContent };
      });
      expect(rendered.found, "an \"Interactions\" header must exist").toBe(true);
      expect(rendered.textContent, "textContent must have the real copy").toBe("Interactions");
      // innerText requires layout; it is empty when content-visibility:auto
      // has skipped rendering this element's contents (the exact defect this
      // guards against). A real fix keeps both in sync.
      expect(rendered.innerText, "innerText must match textContent (not content-visibility-skipped)").toBe("Interactions");
    } finally {
      await server.stop();
    }
  });
});
