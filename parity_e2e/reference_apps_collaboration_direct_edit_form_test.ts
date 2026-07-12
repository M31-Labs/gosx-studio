import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";
import { startMuddyCollaboration } from "./reference_apps_harness";

// HANDOFF-09 Defect 1 regression: the acceptance rehearsal found that two
// authenticated sessions editing the SAME field, with the second session's
// durable direct-edit FORM still showing stale content, resulted in a
// silent last-write-wins instead of the honest rejection the collaboration
// contract requires. reference_apps_collaboration_test.ts already proves the
// ledger honestly rejects a stale RAW GoSXStudioCollaborationRuntime.submit()
// call. This test instead drives the real durable direct-edit FORM UI (the
// actual editor surface end users click "Save" on), which routes through
// operationruntime/island_runtime.js's submit() -- the wrapper that used to
// silently replace a stale form-computed expectedTargetHead with whatever
// collabruntime/island_runtime.js's live broadcast listener had cached from
// ANY actor's accepted operation, defeating the head check before the
// request ever reached the wire.
test.describe("Muddy Noni authenticated collaboration: durable direct-edit form", () => {
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1");
  test.setTimeout(120_000);

  test("stale same-field form save is honestly rejected, not silently applied", async ({ browser, request }) => {
    const server = await startMuddyCollaboration(request);
    const dataPath = path.join(server.dataDir!, "cms.json");
    const contextA = await browser.newContext();
    const contextB = await browser.newContext();
    try {
      const pageA = await contextA.newPage();
      const pageB = await contextB.newPage();

      await pageA.goto(`${server.baseURL}/__test/collaboration/signin/author-a`, { waitUntil: "networkidle" });
      await pageB.goto(`${server.baseURL}/__test/collaboration/signin/author-b`, { waitUntil: "networkidle" });

      const panelA = pageA.locator("[data-gosx-studio-collaboration='true']");
      const panelB = pageB.locator("[data-gosx-studio-collaboration='true']");
      await expect(panelA).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      await expect(panelB).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      await expect(panelA.locator("[data-studio-collaborator]")).toHaveCount(2);
      await expect(panelB.locator("[data-studio-collaborator]")).toHaveCount(2);

      const formA = pageA.locator("#studioDirectEditForm");
      const formB = pageB.locator("#studioDirectEditForm");

      // A edits + saves home.hero.headline via the real durable direct-edit
      // form UI (not a raw runtime.submit() call).
      const committedA = pageA.evaluate(
        () => new Promise<any>((resolve) => document.addEventListener("gosxstudio:operation-committed", (e) => resolve((e as CustomEvent).detail), { once: true })),
      );
      await formA.locator("[data-studio-operation-value]").fill("COLLAB-A-marker");
      await formA.locator("[data-gosx-studio-operation-kind='set-field']").click();
      await committedA;

      // B never reloads: B's form still displays the pre-A content and its
      // own hidden expected-head field is still whatever was baked in at
      // B's original page render (stale relative to A's just-accepted edit).
      const staleHeadBeforeSubmit = await formB.getAttribute("data-studio-target-head");
      await pageB.waitForTimeout(1000); // let B's already-open socket receive A's broadcast

      // B edits (a DIFFERENT value), SAME field, without reloading, and
      // saves via the same real form UI.
      const outcomeB = pageB.evaluate(
        () => new Promise<{ kind: "committed" | "error"; detail: any }>((resolve) => {
          document.addEventListener("gosxstudio:operation-committed", (e) => resolve({ kind: "committed", detail: (e as CustomEvent).detail }), { once: true });
          document.addEventListener("gosxstudio:operation-error", (e) => resolve({ kind: "error", detail: (e as CustomEvent).detail }), { once: true });
        }),
      );
      await formB.locator("[data-studio-operation-value]").fill("STALE-B-marker");
      await formB.locator("[data-gosx-studio-operation-kind='set-field']").click();
      const resultB = await outcomeB;

      expect(resultB.kind).toBe("error");
      await expect(panelB.locator("[data-studio-collab-conflict]")).not.toBeEmpty();

      const after = JSON.parse(readFileSync(dataPath, "utf8"));
      expect(after.draftSettings.hero.headline).toBe("COLLAB-A-marker");
      expect(after.draftSettings.hero.headline).not.toBe("STALE-B-marker");
      void staleHeadBeforeSubmit;
    } finally {
      await Promise.all([contextA.close(), contextB.close()]);
      await server.stop();
    }
  });
});
