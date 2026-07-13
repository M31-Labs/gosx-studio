import { expect, test, type APIRequestContext } from "@playwright/test";
import { mkdtempSync, copyFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { clickAuthoringPanel, openInteractionsTargetDisclosure, startMuddy, type ServerHandle } from "./reference_apps_harness";

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
    const server = await startMuddyWithStagingSeed(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle", timeout: 60_000 });
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
      // reloadAfter/settleAfter: false — this 15-target staging seed page
      // has enough ongoing background asset activity that a strict
      // networkidle wait (the harness's own default post-click behavior)
      // isn't reliable within the config's 30s navigationTimeout. Reload
      // explicitly below with the same domcontentloaded + generous-timeout
      // pattern gotoEditor/reference_apps_interactions_test.ts already use
      // for this reason.
      const attachResponse = await clickAuthoringPanel(page, revealForm, { reloadAfter: false, settleAfter: false });
      expect(attachResponse.status(), "attaching the hero interaction must not fail server-side").toBe(200);
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
