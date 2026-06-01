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

  test("captures marked authoring forms without replacing the editor page", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main class="editor-workbench" data-gosx-studio-workbench="true" data-gosx-studio-save-state="dirty">
            <output data-gosx-studio-save-state="true">Dirty</output>
            <span data-gosx-studio-save-detail="true">Unsaved</span>
            <section data-gosx-studio-editable-control="true">
              <form
                id="edit-headline"
                action="/admin/editor/__actions/authoring"
                method="post"
                data-gosx-studio-authoring-managed="true"
                data-gosx-form="true"
                data-gosx-form-state="idle"
              >
                <input type="hidden" name="csrf_token" value="test-csrf" />
                <input type="hidden" name="gosx_studio_operation" value="save-control" />
                <input type="hidden" name="gosx_studio_binding" value="home.hero.headline" />
                <label>
                  Headline
                  <input
                    name="gosx_studio_value"
                    value="Original headline"
                    data-gosx-studio-authoring-binding="home.hero.headline"
                    data-studio-field="home.hero.headline"
                  />
                </label>
                <button type="submit">Save headline</button>
              </form>
            </section>
            <section data-gosx-studio-preview="true" data-gosx-studio-preview-url="http://127.0.0.1:4173/?gosx-preview=1">
              <iframe title="preview" src="http://127.0.0.1:4173/?gosx-preview=1"></iframe>
            </section>
          </main>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      const request = route.request();
      expect(request.headers()["accept"]).toContain("application/json");
      expect(request.headers()["x-requested-with"]).toBe("XMLHttpRequest");
      expect(request.headers()["x-csrf-token"]).toBe("test-csrf");
      expect(request.postData() ?? "").toContain(`name="gosx_studio_value"`);
      expect(request.postData() ?? "").toContain("Browser authored headline");
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          ok: true,
          message: "Hero headline saved.",
          data: {
            message: "Hero headline saved.",
            previewURL: "http://127.0.0.1:4173/?gosx-preview=1",
            refreshPreview: true,
            draftID: "rev_headline",
            changes: [{
              key: "control-home-hero-headline",
              label: "Headline",
              kind: "control",
              pageKey: "home",
              component: "hero",
              binding: "home.hero.headline",
            }],
          },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/editor");
    await page.addScriptTag({ content: runtimeJS });

    const resultPromise = page.evaluate(() => new Promise<unknown>((resolve) => {
      document.addEventListener("gosxstudio:authoring-result", (event) => {
        resolve((event as CustomEvent).detail);
      }, { once: true });
    }));

    await page.getByLabel("Headline").fill("Browser authored headline");
    await page.getByRole("button", { name: "Save headline" }).click();
    const detail = await resultPromise as {
      result?: { message?: string };
      change?: { kind?: string; binding?: string };
      selectedCount?: number;
      previewCount?: number;
    };

    await expect(page).toHaveURL("http://127.0.0.1:4173/editor");
    await expect(page.locator("[data-gosx-studio-editable-control='true']")).toBeVisible();
    await expect(page.locator("#edit-headline")).toHaveAttribute("data-gosx-form-state", "idle");
    await expect(page.locator("#edit-headline")).not.toHaveAttribute("data-gosx-pending", "true");
    await expect(page.getByLabel("Headline")).toHaveValue("Browser authored headline");
    await expect(page.getByLabel("Headline")).toHaveAttribute("data-gosx-studio-authoring-selected", "true");
    await expect(page.locator("[data-gosx-studio-save-state='true']")).toHaveText("Saved");
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Hero headline saved.");
    await expect(page.locator("[data-gosx-studio-workbench]")).toHaveAttribute("data-gosx-studio-authoring-change-kind", "control");
    await expect(page.locator("iframe")).toHaveAttribute("src", /gosx-studio-refresh=/);
    expect(detail.result?.message).toBe("Hero headline saved.");
    expect(detail.change).toMatchObject({ kind: "control", binding: "home.hero.headline" });
    expect(detail.selectedCount ?? 0).toBeGreaterThan(0);
    expect(detail.previewCount ?? 0).toBeGreaterThan(0);
  });
});
