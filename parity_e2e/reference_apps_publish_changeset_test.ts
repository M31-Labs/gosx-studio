import { expect, test, type Page } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the Release Center's typed "what will change" change-set
// section (adversarial-review punch #4: cms/lifecycle/engine
// Engine.PendingDiff -- DraftPreview{Changes []DraftChange}, domain-grouped
// and actor-attributed -- was computed in slice S3 but never surfaced in
// panels/ or shell/; panels/publish_panel.go now renders it when a host
// supplies cms/studio.PublishChangeSetView's keys under
// data-studio-publish-changeset='true').
//
// HONESTY NOTE (read before "fixing" a red run): as of this writing NEITHER
// Muddy nor Pajaritos has wired Engine.PendingDiff into their publish-panel
// view (see internal/cms/lifecycle_adapter.go / app/admin/editor/
// lifecycle_engine.server.go in muddy-noni-commerce -- the adapter fully
// implements HostLifecycle.OperationLog/PublishedOperationRevision, i.e.
// Engine.PendingDiff is ALREADY COMPUTABLE, but nothing calls it or maps its
// DraftPreview.Changes into cms/studio.PublishChangeSet). Until that host
// slice lands, the Release Center only ever renders the pre-existing legacy
// flat pendingChanges list (panels/publish_panel.go's
// renderPublishPanelPendingChangesLegacy), which does NOT carry
// data-studio-publish-changeset. This test asserts that fact honestly: it
// probes for the new section and, if absent, calls test.skip() with a
// message pointing at the pending host-wiring slice instead of failing or
// silently no-op passing. Once a follow-up Noni slice maps
// engine.DraftChange -> cms/studio.PublishChange (see the handoff report's
// exact mapping sketch) and merges cms/studio.PublishChangeSetView's keys
// into the publish panel's view map, this spec starts exercising the real
// assertions below. Per the task's instructions this file is deliberately
// NOT added to package.json's test:reference-apps list until that host
// wiring exists (running it today would only ever produce a skip, which is
// not useful CI signal yet).
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";
const BOARD_SELECTOR = "[data-studio-site-map-board='true']";
const OVERLAY_HERO = "[data-gosx-canvas-html='page:home']";
const EDITABLE_HERO = "[data-studio-field='home.hero.headline']";
const FIELD_HERO = "[data-studio-field='home.hero.headline']";
const AUTHORING_ROUTE = "/admin/editor/__actions/authoring";

const PUBLISH_PANEL = "[data-studio-publish-panel='true']";
const CHANGESET_SECTION = "[data-studio-publish-changeset='true']";
const CHANGESET_GROUP = "[data-studio-publish-changeset-group]";
const CHANGESET_ROW = "[data-studio-publish-changeset-row='true']";
const CHANGESET_EMPTY = "[data-studio-publish-changeset-empty]";

test.describe("@reference-apps Release Center typed change-set (\"what will change\")", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni: an unpublished hero edit surfaces a grouped, attributed change-set in the Release Center once the host wires Engine.PendingDiff into the publish panel", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));
    page.on("dialog", (dialog) => { void dialog.accept(); });

    const nonce = Math.random().toString(36).slice(2, 10);
    const marker = `changeset probe ${Date.now()} ${nonce}`;

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      await openEditorToHeroSurface(page, consoleErrors);
      await commitInlineEdit(page, marker, consoleErrors);

      // Reveal the publish mode so the Release Center (and, if wired, its
      // change-set section) is in the DOM.
      await revealModeIfPresent(page, "publish");
      await expect(page.locator(PUBLISH_PANEL).first(), "the publish panel must render").toBeAttached({ timeout: 30_000 });

      const changesetCount = await page.locator(CHANGESET_SECTION).count();
      test.skip(
        changesetCount === 0,
        "host wiring pending: no host (Muddy/Pajaritos) yet maps engine.DraftChange -> cms/studio.PublishChangeSetView into the publish panel's view map (see the handoff report's host-contract mapping sketch); the Release Center is correctly falling back to the legacy flat pendingChanges list in the meantime (graceful degradation, not a bug)",
      );

      // Below only runs once a host slice supplies the section.
      const section = page.locator(CHANGESET_SECTION).first();
      await expect(section, "the change-set section must render once a host supplies it").toBeVisible();

      const hasChanges = await section.getAttribute("data-studio-publish-changeset-has-changes");
      expect(hasChanges, "an uncommitted draft edit exists, so the section must report hasChanges=true").toBe("true");

      const groups = page.locator(`${CHANGESET_SECTION} ${CHANGESET_GROUP}`);
      expect(await groups.count(), "at least one scope group must render for a pending edit").toBeGreaterThan(0);

      const rows = page.locator(`${CHANGESET_SECTION} ${CHANGESET_ROW}`);
      expect(await rows.count(), "at least one change row must render for a pending edit").toBeGreaterThan(0);

      const emptyState = page.locator(`${CHANGESET_SECTION} ${CHANGESET_EMPTY}`).first();
      await expect(
        emptyState,
        "the empty-state paragraph must be marked false while changes are pending",
      ).toHaveAttribute("data-studio-publish-changeset-empty", "false");

      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|inline-edit|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

async function openEditorToHeroSurface(page: Page, consoleErrors: string[]) {
  await page.goto("/admin/editor", { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
  await expect(page.locator(BOARD_SELECTOR).first()).toBeAttached();

  const canvas = page.locator(CANVAS_SELECTOR).first();
  await expect(canvas, "the muddy WASM-free canvas must be emitted under MUDDY_CANVAS_WASM_FREE=1").toBeAttached({ timeout: 30_000 });

  await page.waitForFunction(() => {
    const w = window as unknown as Record<string, unknown>;
    const rt = w.GoSXStudioSiteMapRuntime as { setState?: unknown } | undefined;
    const painter = w.GoSXStudioCanvas2DPainterRuntime as { paint?: unknown; renderCanvasBoardHTML?: unknown } | undefined;
    const inlineEdit = w.GoSXStudioCanvasInlineEditRuntime as { install?: unknown; persist?: unknown } | undefined;
    const el = document.querySelector("canvas[data-gosx-canvas-wasm-free='true']") as (HTMLCanvasElement & { GoSXStudioCanvasWasmFree?: unknown }) | null;
    return !!painter && typeof painter.paint === "function" && typeof painter.renderCanvasBoardHTML === "function" &&
      !!rt && typeof rt.setState === "function" &&
      !!inlineEdit && typeof inlineEdit.install === "function" && typeof inlineEdit.persist === "function" &&
      document.documentElement.getAttribute("data-gosx-canvas-wasm-free-client") === "true" &&
      !!el && !!el.GoSXStudioCanvasWasmFree;
  }, null, { timeout: 120_000 });

  await waitForHeroEditable(page);
  await expect(
    page.locator(`${EDITABLE_HERO}${FIELD_HERO}`).first(),
    "the editable hero must carry the store binding data-studio-field='home.hero.headline'",
  ).toBeAttached();
  void consoleErrors;
}

async function waitForHeroEditable(page: Page) {
  await revealModeIfPresent(page, "advanced");
  const overlayHero = page.locator(OVERLAY_HERO).first();
  await expect(overlayHero, "the painter must mount the html surface overlay").toBeAttached({ timeout: 90_000 });
  const editable = page.locator(EDITABLE_HERO).first();
  await expect(editable, "the contenteditable hero must render inside the overlay").toBeAttached({ timeout: 30_000 });
  await expect(editable, "the editable hero element must be visible").toBeVisible({ timeout: 30_000 });
}

async function commitInlineEdit(page: Page, marker: string, consoleErrors: string[]) {
  const editable = page.locator(EDITABLE_HERO).first();
  await editable.click();

  const authoringResponsePromise = page.waitForResponse(
    (resp) => resp.url().includes(AUTHORING_ROUTE) && resp.request().method() === "POST",
    { timeout: 30_000 },
  );

  await editable.click();
  await page.waitForFunction((sel) => {
    const el = document.querySelector(sel);
    return !!el && document.activeElement === el;
  }, EDITABLE_HERO, { timeout: 10_000 });
  await page.keyboard.press("ControlOrMeta+A");
  await page.keyboard.type(marker, { delay: 8 });

  const landed = await pollForEditableText(page, marker);
  expect(
    landed,
    `typed marker must land in the surface DOM before commit; got=${JSON.stringify(landed)}, console=${JSON.stringify(consoleErrors.slice(-6))}`,
  ).toContain(marker);

  await page.evaluate((sel) => {
    const el = document.querySelector(sel) as HTMLElement | null;
    if (el && typeof el.blur === "function") el.blur();
  }, EDITABLE_HERO);

  const postResponse = await authoringResponsePromise;
  expect(
    [200, 303].includes(postResponse.status()),
    `draft autosave POST should be accepted by the authoring action; got status ${postResponse.status()}`,
  ).toBe(true);
}

async function editableText(page: Page): Promise<string> {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel);
    return el ? (el.textContent || "") : "";
  }, EDITABLE_HERO);
}

async function pollForEditableText(page: Page, want: string): Promise<string> {
  const deadline = Date.now() + 15_000;
  let last = "";
  while (Date.now() < deadline) {
    last = await editableText(page);
    if (last.includes(want)) return last;
    await page.waitForTimeout(120);
  }
  return last;
}
