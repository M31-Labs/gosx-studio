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

      // 3) A DIFFERENT field/form, still no reload.
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
