import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../authoringruntime/island_runtime.js"),
  "utf8",
);

test.describe("@smoke GoSXStudioAuthoringRuntime feedback", () => {
  test("selects changed component, refreshes preview, and emits result detail", async ({ page }) => {
    await page.setContent(`
      <main class="editor-workbench" data-gosx-studio-workbench="true">
        <output data-gosx-studio-save-state="true">Dirty</output>
        <span data-gosx-studio-save-detail="true">Unsaved</span>
        <section data-gosx-studio-preview="true" data-gosx-studio-preview-url="http://127.0.0.1:4173/?gosx-preview=1">
          <iframe title="preview" src="http://127.0.0.1:4173/?gosx-preview=1"></iframe>
        </section>
        <section data-gosx-studio-site-canvas="true">
          <button
            type="button"
            data-gosx-studio-canvas-node="home-section-contact"
            data-gosx-studio-canvas-node-kind="section"
            data-gosx-studio-canvas-node-label="Contact"
          >Contact</button>
        </section>
        <article data-studio-site-map-page="home">Home</article>
        <article data-studio-site-map-component="contact" data-studio-site-map-binding="home.section.contact">Contact section</article>
      </main>
    `);
    await page.addScriptTag({ content: runtimeJS });

    const snapshot = await page.evaluate(() => {
      const details: unknown[] = [];
      document.addEventListener("gosxstudio:authoring-result", (event) => {
        details.push((event as CustomEvent).detail);
      });
      document.addEventListener("gosxstudio:canvas-select", (event) => {
        details.push({ canvas: (event as CustomEvent).detail });
      });

      const runtime = (window as unknown as {
        GoSXStudioAuthoringRuntime: {
          handleResult: (result: unknown, meta: unknown) => unknown;
        };
      }).GoSXStudioAuthoringRuntime;

      runtime.handleResult({
        ok: true,
        message: "Contact section shown.",
        data: {
          message: "Contact section shown.",
          previewURL: "http://127.0.0.1:4173/?gosx-preview=1",
          refreshPreview: true,
          draftID: "rev_contact",
          changes: [{
            key: "home-section-contact",
            label: "Contact",
            kind: "component",
            pageKey: "home",
            component: "contact",
            binding: "home.section.contact",
          }],
        },
      }, {
        action: "/admin/editor/__actions/authoring",
        method: "POST",
      });

      const workbench = document.querySelector("[data-gosx-studio-workbench]") as HTMLElement;
      const component = document.querySelector("[data-studio-site-map-component='contact']") as HTMLElement;
      const canvas = document.querySelector("[data-gosx-studio-site-canvas]") as HTMLElement;
      const canvasNode = document.querySelector("[data-gosx-studio-canvas-node='home-section-contact']") as HTMLElement;
      const preview = document.querySelector("[data-gosx-studio-preview]") as HTMLElement;
      const frame = document.querySelector("iframe") as HTMLIFrameElement;
      const saveState = document.querySelector("[data-gosx-studio-save-state]") as HTMLElement;
      const saveDetail = document.querySelector("[data-gosx-studio-save-detail]") as HTMLElement;

      return {
        workbenchState: workbench.getAttribute("data-gosx-studio-authoring-state"),
        selectedCount: workbench.getAttribute("data-gosx-studio-authoring-selected-count"),
        changeComponent: workbench.getAttribute("data-gosx-studio-authoring-change-component"),
        componentSelected: component.getAttribute("data-gosx-studio-authoring-selected"),
        canvasSelected: canvas.getAttribute("data-gosx-studio-canvas-selected"),
        canvasPressed: canvasNode.getAttribute("aria-pressed"),
        previewState: preview.getAttribute("data-gosx-studio-preview-state"),
        frameSrc: frame.getAttribute("src") ?? "",
        saveState: saveState.textContent?.trim() ?? "",
        saveDetail: saveDetail.textContent?.trim() ?? "",
        details,
      };
    });

    expect(snapshot.workbenchState).toBe("saved");
    expect(snapshot.selectedCount).toBe("3");
    expect(snapshot.changeComponent).toBe("contact");
    expect(snapshot.componentSelected).toBe("true");
    expect(snapshot.canvasSelected).toBe("home-section-contact");
    expect(snapshot.canvasPressed).toBe("true");
    expect(snapshot.previewState).toBe("refreshing");
    expect(snapshot.frameSrc).toContain("gosx-studio-refresh=");
    expect(snapshot.saveState).toBe("Saved");
    expect(snapshot.saveDetail).toBe("Contact section shown.");
    expect(JSON.stringify(snapshot.details)).toContain("authoring-result");
    expect(JSON.stringify(snapshot.details)).toContain("home-section-contact");
  });
});
