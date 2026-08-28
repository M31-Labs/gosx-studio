import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";
import { startMuddyCollaboration } from "./reference_apps_harness";

test.describe("Muddy Noni authenticated collaboration", () => {
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1");
  test.setTimeout(120_000);

  test("two signed browser contexts converge without touching live or protected data", async ({ browser, request }) => {
    const server = await startMuddyCollaboration(request);
    const dataPath = path.join(server.dataDir!, "cms.json");
    const contextA = await browser.newContext();
    const contextB = await browser.newContext();
    const viewerContext = await browser.newContext();
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

      const scopes = await Promise.all([pageA, pageB].map((page) => page.evaluate(() => ({
        route: document.querySelector("[data-studio-preview-route-current]")?.getAttribute("data-studio-preview-route-current") || new URL(document.querySelector("[data-studio-preview-route][aria-pressed='true']")?.getAttribute("data-studio-preview-route") || "/", location.origin).pathname,
        viewport: document.querySelector("[data-studio-preview-viewport][aria-pressed='true'], [data-studio-viewport][aria-pressed='true']")?.getAttribute("data-studio-preview-viewport") || document.querySelector("[data-studio-preview-viewport][aria-pressed='true'], [data-studio-viewport][aria-pressed='true']")?.getAttribute("data-studio-viewport"),
      }))));
      expect(scopes).toEqual([{ route: "/", viewport: "desktop" }, { route: "/", viewport: "desktop" }]);

      const remoteSelection = pageB.evaluate(() => new Promise<any>((resolve) => document.addEventListener("gosxstudio:remote-selection", (event) => resolve((event as CustomEvent).detail), { once: true })));
      await pageA.evaluate(() => document.dispatchEvent(new CustomEvent("gosxstudio:preview-select", { detail: { field: "hero.headline", componentKey: "home:hero" } })));
      await expect.poll(async () => (await remoteSelection).selection?.field).toBe("hero.headline");
      expect((await remoteSelection).selection).toMatchObject({ route: "/", pageId: "page:home", blockKey: "home:hero", viewport: "desktop" });
      await expect(panelB.locator("[data-studio-collab-live]")).toContainText("selected hero.headline");

      await pageA.evaluate(() => {
        const target = document.querySelector("[data-studio-preview-frame], [data-studio-canvas]");
        target?.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, clientX: 120, clientY: 160 }));
      });
      await expect(pageB.locator("[data-studio-remote-cursor]")).toHaveCount(1);

      const before = JSON.parse(readFileSync(dataPath, "utf8"));
      const acceptedOnB = pageB.evaluate(() => new Promise<any>((resolve) => document.addEventListener("gosxstudio:collaboration-operation-accepted", (event) => resolve((event as CustomEvent).detail), { once: true })));
      const ack = await pageA.evaluate(async () => {
        const runtime = (window as any).GoSXStudioCollaborationRuntime;
        const operation = { schemaVersion: 1, id: "browser-author-a", kind: "set-field", target: { route: "/", pageId: "home", field: "hero.headline", breakpoint: "base", state: "default" }, value: "Browser collaboration wins", expectedDocumentRevision: 0, expectedTargetHead: "" };
        operation.expectedTargetHead = runtime.expectedHead(operation);
        return runtime.submit(operation);
      });
      expect(ack.sequence).toBe(1);
      expect((await acceptedOnB).record.id).toBe("browser-author-a");

      const stale = await pageB.evaluate(async () => {
        try {
          await (window as any).GoSXStudioCollaborationRuntime.submit({ schemaVersion: 1, id: "browser-author-b-stale", kind: "set-field", target: { route: "/", pageId: "home", field: "hero.headline", breakpoint: "base", state: "default" }, value: "Must not overwrite", expectedDocumentRevision: 0, expectedTargetHead: "" });
          return null;
        } catch (error: any) {
          return error.collaborationError ?? { message: error.message };
        }
      });
      expect(stale?.code).toBe("stale_field");
      expect(stale?.currentHead).toBe("browser-author-a");

      const after = JSON.parse(readFileSync(dataPath, "utf8"));
      expect(after.settings.hero.headline).toBe(before.settings.hero.headline);
      expect(after.draftSettings.hero.headline).toBe("Browser collaboration wins");
      expect(protectedView(after)).toEqual(protectedView(before));

      await pageA.reload({ waitUntil: "networkidle" });
      await expect(pageA.locator("[data-gosx-studio-collaboration='true']")).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      const resumedSequence = await pageA.evaluate(() => (window as any).GoSXStudioCollaborationRuntime ? Number(sessionStorage.getItem(Object.keys(sessionStorage).find((key) => key.startsWith("gosxstudio:collab-sequence:")) ?? "") || "0") : 0);
      expect(resumedSequence).toBeGreaterThanOrEqual(1);

      const viewer = await viewerContext.newPage();
      await viewer.goto(`${server.baseURL}/__test/collaboration/signin/viewer`, { waitUntil: "networkidle" });
      const viewerPanel = viewer.locator("[data-gosx-studio-collaboration='true']");
      await expect(viewerPanel).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      const operationButtons = viewer.locator("[data-gosx-studio-operation-kind]");
      expect(await operationButtons.count()).toBeGreaterThan(0);
      await expect(operationButtons.first()).toBeDisabled();
      const overflow = await viewer.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
      expect(overflow).toBeLessThanOrEqual(1);
    } finally {
      await Promise.all([contextA.close(), contextB.close(), viewerContext.close()]);
      await server.stop();
    }
  });
});

function protectedView(data: any) {
  return {
    products: data.products,
    gallery: data.gallery,
    pages: data.pages,
    blog: data.blog,
    media: data.media,
    categories: data.categories,
    orders: data.orders,
    contacts: data.contacts,
    webhooks: data.webhooks,
    secrets: data.secrets,
  };
}
