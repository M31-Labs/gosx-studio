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

  test("keeps targeted managed submits in-page and applies only the newest response", async ({ page }) => {
    let requestCount = 0;
    const requests: { csrf?: string; postData: string }[] = [];
    await page.route("http://127.0.0.1:4173/editor-targeted-submit", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main class="editor-workbench" data-gosx-studio-workbench="true">
            <span data-gosx-studio-save-detail="true">Unsaved</span>
            <section data-gosx-studio-editable-control="true">
              <form
                id="targeted-authoring"
                action="/admin/editor/__actions/authoring"
                method="post"
                target="_blank"
                data-gosx-studio-authoring-managed="true"
                data-gosx-form-state="idle"
              >
                <input type="hidden" name="csrf_token" value="csrf-latest" />
                <input type="hidden" name="publish" value="field-value" />
                <input name="gosx_studio_value" value="Original" data-gosx-studio-authoring-binding="home.hero.title" />
                <button type="submit" name="publish" value="draft" formtarget="_blank">Save draft</button>
              </form>
            </section>
          </main>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      requestCount += 1;
      const index = requestCount;
      const request = route.request();
      requests.push({
        csrf: request.headers()["x-csrf-token"],
        postData: request.postData() ?? "",
      });
      const body = JSON.stringify({
        ok: true,
        message: index === 1 ? "Older response" : "Newest response",
        data: {
          message: index === 1 ? "Older response" : "Newest response",
          refreshPreview: false,
          changes: [{
            key: index === 1 ? "old-title" : "new-title",
            kind: "control",
            binding: "home.hero.title",
          }],
        },
      });
      if (index === 1) {
        await new Promise((resolve) => setTimeout(resolve, 150));
      }
      await route.fulfill({ contentType: "application/json", body });
    });

    await page.goto("http://127.0.0.1:4173/editor-targeted-submit");
    await page.addScriptTag({ content: runtimeJS });

    const detailsPromise = page.evaluate(() => new Promise<unknown[]>((resolve) => {
      const details: unknown[] = [];
      document.addEventListener("gosxstudio:authoring-result", (event) => {
        details.push((event as CustomEvent).detail);
        if (details.length === 1) window.setTimeout(() => resolve(details), 40);
      });
    }));
    const popupPromise = page.waitForEvent("popup", { timeout: 600 }).then(() => "popup", () => "none");

    await page.getByRole("button", { name: "Save draft" }).click();
    await page.getByRole("button", { name: "Save draft" }).click();

    const [details, popup] = await Promise.all([detailsPromise, popupPromise]);
    await expect(page).toHaveURL("http://127.0.0.1:4173/editor-targeted-submit");
    await expect(page.locator("#targeted-authoring")).toHaveAttribute("data-gosx-form-state", "idle");
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Newest response");
    expect(popup).toBe("none");
    expect(requestCount).toBe(2);
    expect(requests.map((request) => request.csrf)).toEqual(["csrf-latest", "csrf-latest"]);
    expect(requests.every((request) => (request.postData.match(/name="publish"/g) ?? []).length >= 2)).toBe(true);
    expect(JSON.stringify(details)).toContain("Newest response");
    expect(JSON.stringify(details)).not.toContain("Older response");
  });

  test("refreshes GET managed controls through preview instead of document navigation", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor-get-submit", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main class="editor-workbench" data-gosx-studio-workbench="true">
            <section data-gosx-studio-preview="true" data-gosx-studio-preview-url="http://127.0.0.1:4173/preview?gosx-preview=1">
              <iframe title="preview" src="http://127.0.0.1:4173/preview?gosx-preview=1"></iframe>
            </section>
            <form
              id="filter-preview"
              action="/preview"
              method="post"
              target="_blank"
              data-gosx-studio-authoring-managed="true"
              data-gosx-form-state="idle"
            >
              <input name="section" value="gallery" />
              <button type="submit" formmethod="get" formtarget="_blank">Preview filter</button>
            </form>
          </main>
        `,
      });
    });
    await page.goto("http://127.0.0.1:4173/editor-get-submit");
    await page.addScriptTag({ content: runtimeJS });

    const popupPromise = page.waitForEvent("popup", { timeout: 600 }).then(() => "popup", () => "none");
    await page.getByRole("button", { name: "Preview filter" }).click();

    await expect(page).toHaveURL("http://127.0.0.1:4173/editor-get-submit");
    await expect(page.locator("#filter-preview")).toHaveAttribute("data-gosx-form-state", "idle");
    await expect(page.locator("iframe")).toHaveAttribute("src", /section=gallery/);
    await expect(page.locator("iframe")).toHaveAttribute("src", /gosx-studio-refresh=/);
    expect(await popupPromise).toBe("none");
  });

  test("keeps validation errors in-page with field details", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor-validation", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main class="editor-workbench" data-gosx-studio-workbench="true">
            <output data-gosx-studio-save-state="true">Ready</output>
            <span data-gosx-studio-save-detail="true">Waiting</span>
            <form
              id="validation-authoring"
              action="/admin/editor/__actions/authoring"
              method="post"
              data-gosx-studio-authoring-managed="true"
              data-gosx-form-state="idle"
            >
              <input type="hidden" name="csrf_token" value="csrf-error" />
              <label>Headline <input name="gosx_studio_value" value="" /></label>
              <p data-gosx-studio-field-error-for="gosx_studio_value" hidden></p>
              <button type="submit">Save invalid</button>
            </form>
          </main>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      await route.fulfill({
        status: 422,
        contentType: "application/json",
        body: JSON.stringify({
          ok: false,
          message: "Headline is required.",
          fieldErrors: { gosx_studio_value: "Enter a headline." },
          values: { gosx_studio_value: "" },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/editor-validation");
    await page.addScriptTag({ content: runtimeJS });
    const errorPromise = page.evaluate(() => new Promise<unknown>((resolve) => {
      document.addEventListener("gosxstudio:authoring-error", (event) => {
        resolve((event as CustomEvent).detail);
      }, { once: true });
    }));

    await page.getByRole("button", { name: "Save invalid" }).click();
    const detail = await errorPromise as {
      status?: number;
      message?: string;
      fieldErrors?: Record<string, string>;
    };

    await expect(page).toHaveURL("http://127.0.0.1:4173/editor-validation");
    await expect(page.locator("#validation-authoring")).toHaveAttribute("data-gosx-form-state", "error");
    await expect(page.locator("#validation-authoring")).not.toHaveAttribute("data-gosx-pending", "true");
    await expect(page.locator("#validation-authoring")).toHaveAttribute("data-gosx-studio-authoring-error-status", "422");
    await expect(page.locator("[name='gosx_studio_value']")).toHaveAttribute("aria-invalid", "true");
    await expect(page.locator("[name='gosx_studio_value']")).toHaveAttribute("data-gosx-studio-authoring-field-error", "Enter a headline.");
    await expect(page.locator("[data-gosx-studio-field-error-for='gosx_studio_value']")).toHaveText("Enter a headline.");
    await expect(page.locator("[data-gosx-studio-save-state='true']")).toHaveText("Error");
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Headline is required.");
    expect(detail.status).toBe(422);
    expect(detail.message).toBe("Headline is required.");
    expect(detail.fieldErrors?.gosx_studio_value).toBe("Enter a headline.");
  });

  test("preserves typing that happens while a fragment save is in flight", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor-preserve-typing", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main class="editor-workbench" data-gosx-studio-workbench="true">
            <output data-gosx-studio-save-state="true">Ready</output>
            <span data-gosx-studio-save-detail="true">Waiting</span>
            <section data-gosx-studio-fragment="editor-form">
              <form
                id="typing-authoring"
                action="/admin/editor/__actions/authoring"
                method="post"
                data-gosx-studio-authoring-managed="true"
                data-gosx-form-state="idle"
              >
                <input type="hidden" name="csrf_token" value="csrf-typing" />
                <label>Headline <input id="typing-headline" name="gosx_studio_value" value="Submitted" data-gosx-studio-authoring-binding="home.hero.headline" /></label>
                <button type="submit">Save headline</button>
              </form>
            </section>
          </main>
        `,
      });
    });
    await page.route("http://127.0.0.1:4173/editor-preserve-fragment", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main>
            <section data-gosx-studio-fragment="editor-form">
              <form
                id="typing-authoring"
                action="/admin/editor/__actions/authoring"
                method="post"
                data-gosx-studio-authoring-managed="true"
                data-gosx-form-state="idle"
              >
                <input type="hidden" name="csrf_token" value="csrf-typing" />
                <label>Headline <input id="typing-headline" name="gosx_studio_value" value="Submitted" data-gosx-studio-authoring-binding="home.hero.headline" /></label>
                <button type="submit">Save headline</button>
              </form>
            </section>
          </main>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 150));
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          ok: true,
          message: "Saved submitted value.",
          data: {
            message: "Saved submitted value.",
            fragmentURL: "http://127.0.0.1:4173/editor-preserve-fragment",
            fragments: [{ selector: "[data-gosx-studio-fragment='editor-form']" }],
            changes: [{ kind: "control", binding: "home.hero.headline" }],
          },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/editor-preserve-typing");
    await page.addScriptTag({ content: runtimeJS });
    const resultPromise = page.evaluate(() => new Promise<unknown>((resolve) => {
      document.addEventListener("gosxstudio:authoring-result", (event) => {
        resolve((event as CustomEvent).detail);
      }, { once: true });
    }));

    await page.getByRole("button", { name: "Save headline" }).click();
    await page.getByLabel("Headline").fill("Still typing");
    await resultPromise;

    await expect(page.getByLabel("Headline")).toHaveValue("Still typing");
    await expect(page.locator("#typing-authoring")).toHaveAttribute("data-gosx-form-state", "dirty");
    await expect(page.locator("[data-gosx-studio-save-state='true']")).toHaveText("Unsaved");
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Unsaved changes");
    await page.waitForTimeout(150);
    await expect(page.locator("[data-gosx-studio-save-state='true']")).toHaveText("Unsaved");
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Unsaved changes");
  });

  test("treats missing structured authoring JSON as an in-page contract error", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor-contract-error", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main class="editor-workbench" data-gosx-studio-workbench="true">
            <output data-gosx-studio-save-state="true">Ready</output>
            <span data-gosx-studio-save-detail="true">Waiting</span>
            <form
              id="contract-authoring"
              action="/admin/editor/__actions/authoring"
              method="post"
              data-gosx-studio-authoring-managed="true"
              data-gosx-form-state="idle"
            >
              <input type="hidden" name="csrf_token" value="csrf-contract" />
              <input name="gosx_studio_value" value="Headline" />
              <button type="submit">Save contract</button>
            </form>
          </main>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      await route.fulfill({ contentType: "application/json", body: "{}" });
    });

    await page.goto("http://127.0.0.1:4173/editor-contract-error");
    await page.addScriptTag({ content: runtimeJS });
    const errorPromise = page.evaluate(() => new Promise<unknown>((resolve) => {
      document.addEventListener("gosxstudio:authoring-error", (event) => {
        resolve((event as CustomEvent).detail);
      }, { once: true });
    }));
    await page.getByRole("button", { name: "Save contract" }).click();
    const detail = await errorPromise as { status?: number; message?: string };

    await expect(page).toHaveURL("http://127.0.0.1:4173/editor-contract-error");
    await expect(page.locator("#contract-authoring")).toHaveAttribute("data-gosx-form-state", "error");
    await expect(page.locator("#contract-authoring")).not.toHaveAttribute("data-gosx-pending", "true");
    await expect(page.locator("[data-gosx-studio-save-state='true']")).toHaveText("Error");
    expect(detail.status).toBe(200);
    expect(detail.message).toBe("Studio action failed; no structured authoring response.");
  });

  test("refreshes structural fragments before emitting authoring result", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <main class="editor-workbench" data-gosx-studio-workbench="true">
            <section data-gosx-studio-fragment="site-map">
              <label>Title <input id="fragment-title" name="title" value="Hero copy 2" data-gosx-studio-authoring-binding="home.section.hero__copy_2" /></label>
              <article data-studio-site-map-component="hero__copy_2">Hero copy 2</article>
            </section>
          </main>
        `,
      });
    });
    await page.setContent(`
        <main class="editor-workbench" data-gosx-studio-workbench="true">
          <span data-gosx-studio-save-detail="true">Unsaved</span>
          <section data-gosx-studio-fragment="site-map">
            <label>Title <input id="fragment-title" name="title" value="Hero" data-gosx-studio-authoring-binding="home.section.hero__copy_2" /></label>
            <article data-studio-site-map-component="hero">Hero</article>
          </section>
        </main>
    `);
    await page.addScriptTag({ content: runtimeJS });

    const snapshot = await page.evaluate(async () => {
      const details: unknown[] = [];
      document.addEventListener("gosxstudio:fragments-refresh", (event) => {
        details.push({ fragments: (event as CustomEvent).detail });
      });
      document.addEventListener("gosxstudio:authoring-result", (event) => {
        details.push({ authoring: (event as CustomEvent).detail });
      });

      const runtime = (window as unknown as {
        GoSXStudioAuthoringRuntime: {
          handleResult: (result: unknown, meta: unknown) => Promise<unknown>;
        };
      }).GoSXStudioAuthoringRuntime;
      const input = document.querySelector("#fragment-title") as HTMLInputElement;
      input.focus();
      input.setSelectionRange(2, 2);

      await runtime.handleResult({
        ok: true,
        message: "Hero copy 2 duplicated.",
        data: {
          message: "Hero copy 2 duplicated.",
          fragmentURL: "http://127.0.0.1:4173/editor",
          fragments: [{ selector: "[data-gosx-studio-fragment='site-map']" }],
          changes: [{
            key: "home-section-hero__copy_2",
            label: "Hero copy 2",
            kind: "component",
            pageKey: "home",
            component: "hero__copy_2",
            binding: "home.section.hero__copy_2",
          }],
        },
      }, {
        action: "/admin/editor/__actions/authoring",
        method: "POST",
      });

      const root = document.querySelector("[data-gosx-studio-fragment='site-map']") as HTMLElement;
      const copied = document.querySelector("[data-studio-site-map-component='hero__copy_2']") as HTMLElement;
      return {
        html: root.innerHTML,
        selected: copied?.getAttribute("data-gosx-studio-authoring-selected") ?? "",
        activeID: document.activeElement?.getAttribute("id") ?? "",
        selectionStart: (document.activeElement as HTMLInputElement | null)?.selectionStart ?? -1,
        saveDetail: document.querySelector("[data-gosx-studio-save-detail]")?.textContent?.trim() ?? "",
        details,
      };
    });

    expect(snapshot.html).toContain("Hero copy 2");
    expect(snapshot.selected).toBe("true");
    expect(snapshot.activeID).toBe("fragment-title");
    expect(snapshot.selectionStart).toBe(2);
    expect(snapshot.saveDetail).toBe("Hero copy 2 duplicated.");
    expect(JSON.stringify(snapshot.details)).toContain("fragments");
    expect(JSON.stringify(snapshot.details)).toContain("\"fragmentCount\":1");
  });
});
