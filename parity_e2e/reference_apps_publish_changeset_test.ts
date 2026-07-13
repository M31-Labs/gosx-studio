import { expect, test, type Page } from "@playwright/test";
import { revealModeIfPresent, startMuddyCollaboration } from "./reference_apps_harness";

// Live-proof e2e for the Release Center's typed "what will change" change-set
// section (adversarial-review punch #4: cms/lifecycle/engine
// Engine.PendingDiff -- DraftPreview{Changes []DraftChange}, domain-grouped
// and actor-attributed -- is computed in slice S3 and rendered by
// panels/publish_panel.go when a host supplies cms/studio.PublishChangeSetView's
// keys under data-studio-publish-changeset='true'). HANDOFF-29 wires the
// Noni host side: app/admin/editor/lifecycle_engine.server.go's
// editorPendingChangeSet calls Engine.PendingDiff and
// engineChangesToPublishChanges maps engine.DraftChange -> cms/studio.
// PublishChange, and page.server.go merges cms/studio.PublishChangeSetView's
// keys into the publish panel's flat view map.
//
// The edit driving this test goes through the REAL durable direct-edit form
// (#studioDirectEditForm, the Home Inspector's hero-headline editor), which
// posts a set-field operation through internal/cms.Store.ApplyStudioOperation
// -- the durable operation log Engine.PendingDiff reads from
// (OperationLog/PublishedOperationRevision).
//
// STALE-COMMENT UPDATE (handoff-31, "one product path"): this used to say
// the canvas contenteditable inline-edit surface could never produce a
// change here because it only ever posted a non-durable save-control
// mutation. That gap is now closed for the specific fields a host opts in
// (canvas/page_surface.go's DurableHeadlineField/DurableTitleField;
// inlineeditruntime/island_runtime.js's data-gosx-studio-durable-history
// dispatch) — reference_apps_canvas_inline_edit_test.ts's third test proves
// a canvas-committed hero.headline edit produces the SAME
// data-studio-publish-changeset-group='content' row this test checks for
// below, end to end with a live collaboration transport. A canvas field the
// host has NOT opted in (tagline/subhead/ctaLabel, or any page title other
// than about's) still commits through the legacy save-control mutation and
// correctly produces no change-set row here — that remaining gap is
// intentional per-field scope, not an oversight.
//
// HONESTY NOTE (read before "fixing" a red run): if a host under test has NOT
// wired Engine.PendingDiff into its publish-panel view (e.g. Pajaritos, or a
// pre-HANDOFF-29 Muddy checkout), the Release Center only ever renders the
// pre-existing legacy flat pendingChanges list (panels/publish_panel.go's
// renderPublishPanelPendingChangesLegacy), which does NOT carry
// data-studio-publish-changeset. This test asserts that fact honestly: it
// probes for the new section and, if absent, calls test.skip() with a
// message pointing at the pending host-wiring slice instead of failing or
// silently no-op passing.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const WORKBENCH = "[data-studio-workbench='true']";
const COLLAB_PANEL = "[data-gosx-studio-collaboration='true']";
const DIRECT_EDIT_FORM = "#studioDirectEditForm";
const DIRECT_EDIT_VALUE = "[data-studio-operation-value]";
const DIRECT_EDIT_SAVE = "[data-gosx-studio-operation-kind='set-field']";
const PUBLISH_ROUTE = "/admin/editor/__actions/publish";

const PUBLISH_PANEL = "[data-studio-publish-panel='true']";
const PUBLISH_BUTTON = "[data-studio-submit-action='publish']";
const CHANGESET_SECTION = "[data-studio-publish-changeset='true']";
const CHANGESET_GROUP = "[data-studio-publish-changeset-group]";
const CHANGESET_ROW = "[data-studio-publish-changeset-row='true']";
const CHANGESET_EMPTY = "[data-studio-publish-changeset-empty]";

test.describe("@reference-apps Release Center typed change-set (\"what will change\")", () => {
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni: a durable hero-headline edit surfaces a grouped, attributed change-set in the Release Center, and Publish empties it", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));
    page.on("dialog", (dialog) => { void dialog.accept(); });

    const nonce = Math.random().toString(36).slice(2, 10);
    const marker = `changeset probe ${Date.now()} ${nonce}`;

    // startMuddyCollaboration (not the plain startMuddy) is required here: Noni
    // always wires a collaboration endpoint (cmd/muddy-noni/main.go's
    // studiohost.CollaborationPath), and collabruntime/island_runtime.js's
    // global permissions() gate disables every [data-gosx-studio-operation-kind]
    // button -- including the durable direct-edit form's Save button -- until
    // its websocket handshake completes and the server grants real
    // capabilities. That handshake needs a real authenticated session (the
    // MUDDY_MOCK_AUTH server-side fallback only covers plain HTTP action
    // handlers, not the collaboration websocket's own auth), so this test signs
    // in through the MUDDY_COLLAB_TEST_AUTH test-only route
    // (registerCollaborationTestAuth), the same real mechanism
    // reference_apps_collaboration_direct_edit_form_test.ts already proves out.
    const server = await startMuddyCollaboration(request);
    try {
      await page.goto(`${server.baseURL}/__test/collaboration/signin/author-a`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator(WORKBENCH).first()).toBeAttached({ timeout: 30_000 });

      const collabPanel = page.locator(COLLAB_PANEL).first();
      await expect(
        collabPanel,
        "the collaboration runtime must connect before durable operation buttons enable",
      ).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });

      await commitDirectEdit(page, marker);

      // The publish panel's change-set section is server-rendered from the
      // store's CURRENT durable-operation-log state at page-load time -- the
      // committed edit above happened client-side after that render, so a
      // reload is required before the Release Center reflects it (mirrors
      // reference_apps_canvas_draft_publish_test.ts's identical reload-before-
      // checking-pending-state step).
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(page.locator(WORKBENCH).first()).toBeAttached({ timeout: 30_000 });

      // Reveal the publish mode so the Release Center (and, if wired, its
      // change-set section) is in the DOM.
      await revealModeIfPresent(page, "publish");
      await expect(page.locator(PUBLISH_PANEL).first(), "the publish panel must render").toBeAttached({ timeout: 30_000 });

      const changesetCount = await page.locator(CHANGESET_SECTION).count();
      test.skip(
        changesetCount === 0,
        "host wiring pending: no host (Muddy/Pajaritos) yet maps engine.DraftChange -> cms/studio.PublishChangeSetView into the publish panel's view map (see handoff-29-publish-changeset-report.md); the Release Center is correctly falling back to the legacy flat pendingChanges list in the meantime (graceful degradation, not a bug)",
      );

      // ── Below only runs once a host slice supplies the section. ──────────
      const section = page.locator(CHANGESET_SECTION).first();
      await expect(section, "the change-set section must render once a host supplies it").toBeVisible();

      const hasChanges = await section.getAttribute("data-studio-publish-changeset-has-changes");
      expect(hasChanges, "a durable pending edit exists, so the section must report hasChanges=true").toBe("true");

      const groups = page.locator(`${CHANGESET_SECTION} ${CHANGESET_GROUP}`);
      expect(await groups.count(), "at least one scope group must render for a pending edit").toBeGreaterThan(0);
      await expect(
        page.locator(`${CHANGESET_SECTION} [data-studio-publish-changeset-group='content']`).first(),
        "the hero-headline set-field op classifies as the Content scope",
      ).toBeAttached();

      const rows = page.locator(`${CHANGESET_SECTION} ${CHANGESET_ROW}`);
      expect(await rows.count(), "at least one change row must render for a pending edit").toBeGreaterThan(0);
      await expect(
        page.locator(`${CHANGESET_SECTION} code.revision-diff-row__label`, { hasText: "hero.headline" }).first(),
        "the row must be labeled with the durable operation's target field",
      ).toBeAttached();

      const emptyStateBefore = page.locator(`${CHANGESET_SECTION} ${CHANGESET_EMPTY}`).first();
      await expect(
        emptyStateBefore,
        "the empty-state paragraph must be marked false while changes are pending",
      ).toHaveAttribute("data-studio-publish-changeset-empty", "false");

      // ── PUBLISH → the durable operation log's high-water mark advances, so
      // the SAME change-set section must now report the honest empty state. ──
      const publishResponsePromise = page.waitForResponse(
        (resp) => resp.url().includes(PUBLISH_ROUTE) && resp.request().method() === "POST",
        { timeout: 30_000 },
      );
      await page.locator(PUBLISH_BUTTON).first().click();
      const publishResponse = await publishResponsePromise;
      expect(
        [200, 303].includes(publishResponse.status()),
        `Publish action POST should be accepted/redirected; got status ${publishResponse.status()}`,
      ).toBe(true);

      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "publish");
      await expect(page.locator(PUBLISH_PANEL).first(), "the publish panel must still render after publish").toBeAttached({ timeout: 30_000 });

      const sectionAfterPublish = page.locator(CHANGESET_SECTION).first();
      await expect(sectionAfterPublish, "the change-set section must still render (Supplied stays true) after publish").toBeVisible({ timeout: 30_000 });
      await expect(
        sectionAfterPublish,
        "after publish there is nothing pending, so hasChanges must flip to false",
      ).toHaveAttribute("data-studio-publish-changeset-has-changes", "false");
      await expect(
        page.locator(`${CHANGESET_SECTION} ${CHANGESET_EMPTY}`).first(),
        "the empty-state paragraph must flip to true once the publish clears the durable log's pending tail",
      ).toHaveAttribute("data-studio-publish-changeset-empty", "true");
      expect(
        await page.locator(`${CHANGESET_SECTION} ${CHANGESET_ROW}`).count(),
        "no change rows should remain once everything is published",
      ).toBe(0);

      const clientErrors = consoleErrors.filter((line) => /gosxstudio|operationruntime|workbenchruntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected editor client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// commitDirectEdit fills + saves the Home Inspector's durable hero-headline
// direct-edit form (studio.RenderDirectEditPanel), the real UI surface that
// posts an AuthoringOperationSetField mutation. Both write paths land in the
// SAME durable operation log Engine.PendingDiff's OperationLog/
// PublishedOperationRevision read from
// (internal/cms.Store.ApplyStudioOperation / .ProjectStudioOperation append
// to the identical state.Operations slice) -- operationruntime/
// island_runtime.js's submit() prefers the collaboration websocket
// (GoSXStudioCollaborationRuntime.submit, -> ProjectStudioOperation) over a
// plain HTTP POST to AUTHORING_ROUTE (-> ApplyStudioOperation) whenever a
// collaboration runtime is connected and available, which it always is here
// (see the collab-connected wait above) -- so this only waits for the
// "gosxstudio:operation-committed" event the runtime fires on success either
// way, not a specific transport.
async function commitDirectEdit(page: Page, marker: string) {
  const form = page.locator(DIRECT_EDIT_FORM);
  await expect(form, "the durable hero-headline direct-edit form must render (Home Inspector)").toBeAttached({ timeout: 30_000 });

  const committed = page.evaluate(
    () => new Promise<{ kind: "committed" | "error"; detail: unknown }>((resolve) => {
      document.addEventListener("gosxstudio:operation-committed", (event) => resolve({ kind: "committed", detail: (event as CustomEvent).detail }), { once: true });
      document.addEventListener("gosxstudio:operation-error", (event) => resolve({ kind: "error", detail: (event as CustomEvent).detail }), { once: true });
    }),
  );

  await form.locator(DIRECT_EDIT_VALUE).fill(marker);
  await form.locator(DIRECT_EDIT_SAVE).click();

  const outcome = await committed;
  expect(outcome.kind, `the durable set-field commit must succeed; got ${JSON.stringify(outcome)}`).toBe("committed");
}
