import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";
import { startMuddyCollaboration } from "./reference_apps_harness";

// HANDOFF-21 regression: aeeb86d (collabruntime/island_runtime.js) fixed the
// two-SESSION stale-write bypass (a live broadcast of ANOTHER actor's
// accepted operation used to rebase this connection's local head cache,
// letting a stale form's blank-vs-live head check pass). That fix must not
// regress the single-session sequential-edit case: ONE authenticated editor,
// with ONE already-connected/synced collaboration socket, editing and saving
// the SAME field twice in a row, then a DIFFERENT field, all in the same
// page load (no reload between saves). Each of those saves is this
// connection's own accepted operation resolving isOwnPending in accepted(),
// so the local head cache must keep advancing correctly and never falsely
// reject a subsequent save from the very session that just committed it.
//
// PRODUCT REGRESSION, NOW FIXED (osier found it in wave-3B follow-up, see
// .tiller/scratch/codex/handoff-3b2-sequential-fix-report.md; fixed in
// handoff-4 -- see .tiller/scratch/codex/handoff-4-polish-tail-report.md):
// step 3 below (the aboutForm save) used to time out because
// #studioAboutDirectEditForm was permanently hidden by wave 3B's dedupe
// guard (hostruntime/assets/workbench_runtime.js scopeDirectEditPanels).
// Two bugs compounded: (a) scopeSupportSelectionPanels only ever re-ran
// inside the "gosxstudio:canvas-select" CustomEvent handler, but NO real
// selection-changing interaction (Home's own layer-row block selection,
// the site-map board, or the live preview's own field-click selection)
// ever dispatched that event -- they all write the data-studio-selection
// attribute directly; (b) even once re-triggered, the live preview's field
// click writes data-studio-selection as the FIELD PATH
// ("pages.about.title"), not the panel's ComponentKey ("about:content"),
// so the two would still never have matched. Fixed by (a) a
// MutationObserver on the workbench form's data-studio-selection attribute
// (mirrors workbenchruntime/island_runtime.js's own sessionStorage-persist
// pattern for the same attribute) that re-runs scopeSupportSelectionPanels
// on every mutation, and (b) scopeSelectionPanels accepting an optional
// fieldAttr parameter so scopeDirectEditPanels also matches a panel's
// existing data-studio-target-field attribute (already emitted by
// panels/direct_edit_panel.go, no new Go-side attribute needed) against
// the live selection. This test now drives the real, intended human
// gesture -- switching the PageCanvas route to "About" and clicking its
// <h1 data-studio-field="pages.about.title"> in the live preview -- before
// step 3, proving the panel is genuinely reachable, not forced open with
// an evaluate()-driven selection hack.
test.describe("Muddy Noni authenticated collaboration: single-session sequential edits", () => {
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1");
  test.setTimeout(120_000);

  test("edit, save, edit same field again, save, edit a different field, save -- all succeed", async ({ browser, request }) => {
    const server = await startMuddyCollaboration(request);
    const dataPath = path.join(server.dataDir!, "cms.json");
    const context = await browser.newContext();
    try {
      const page = await context.newPage();
      await page.goto(`${server.baseURL}/__test/collaboration/signin/author-a`, { waitUntil: "networkidle" });

      const panel = page.locator("[data-gosx-studio-collaboration='true']");
      await expect(panel).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });

      const heroForm = page.locator("#studioDirectEditForm");
      const aboutForm = page.locator("#studioAboutDirectEditForm");

      async function saveAndAwait(form: import("@playwright/test").Locator, value: string) {
        const outcome = page.evaluate(
          () => new Promise<{ kind: "committed" | "error"; detail: any }>((resolve) => {
            document.addEventListener("gosxstudio:operation-committed", (e) => resolve({ kind: "committed", detail: (e as CustomEvent).detail }), { once: true });
            document.addEventListener("gosxstudio:operation-error", (e) => resolve({ kind: "error", detail: (e as CustomEvent).detail }), { once: true });
          }),
        );
        await form.locator("[data-studio-operation-value]").fill(value);
        await form.locator("[data-gosx-studio-operation-kind='set-field']").click();
        return outcome;
      }

      // 1) First edit + save of the hero headline.
      const first = await saveAndAwait(heroForm, "SEQ-EDIT-ONE");
      expect(first.kind).toBe("committed");
      await expect(panel.locator("[data-studio-collab-conflict]")).toBeEmpty();

      // 2) SAME field, second edit + save, no reload in between. This is the
      // gap aeeb86d's own-ack test coverage never exercised: this
      // connection's own immediately-preceding accepted operation must have
      // already advanced its local head cache so this save is NOT
      // considered stale against itself.
      const second = await saveAndAwait(heroForm, "SEQ-EDIT-TWO");
      expect(second.kind).toBe("committed");
      await expect(panel.locator("[data-studio-collab-conflict]")).toBeEmpty();

      // 3) A DIFFERENT field/form, still no reload -- but first, a REAL human
      // selection gesture must reveal the About direct-edit panel: switch the
      // PageCanvas route to "About" and click its live-preview headline. This
      // is the exact path handoff-3b2 proved live updates data-studio-selection
      // to "pages.about.title" (previewruntime + selectionruntime's
      // applyPreviewSelectionState), which scopeDirectEditPanels' fieldAttr
      // bridge now matches against about:content's data-studio-target-field.
      const workbench = page.locator("[data-studio-workbench]");
      await page.locator('[data-studio-preview-route-key="about"]').click();
      const aboutFrame = page.frameLocator("iframe[data-studio-preview-frame]");
      await aboutFrame.locator('[data-studio-field="pages.about.title"]').click();
      await expect(workbench).toHaveAttribute("data-studio-selection", "pages.about.title", { timeout: 15_000 });
      await expect(aboutForm, "the About direct-edit panel must become visible once its own field is selected in the live preview").toBeVisible({ timeout: 15_000 });

      const third = await saveAndAwait(aboutForm, "SEQ-EDIT-ABOUT");
      expect(third.kind).toBe("committed");
      await expect(panel.locator("[data-studio-collab-conflict]")).toBeEmpty();

      const after = JSON.parse(readFileSync(dataPath, "utf8"));
      expect(after.draftSettings.hero.headline).toBe("SEQ-EDIT-TWO");
      expect(after.draftPages?.about?.title).toBe("SEQ-EDIT-ABOUT");
    } finally {
      await context.close();
      await server.stop();
    }
  });
});
