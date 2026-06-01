import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../authoringruntime/island_runtime.js"),
  "utf8",
);

const fields = {
  operation: "gosx_studio_operation",
  intentKey: "gosx_studio_intent_key",
  intentKind: "gosx_studio_intent_kind",
  pageKey: "gosx_studio_page_key",
  pageLabel: "gosx_studio_page_label",
  pageRoute: "gosx_studio_page_route",
  pageBlueprintKey: "gosx_studio_page_blueprint_key",
  componentKey: "gosx_studio_component_key",
  componentTemplateKey: "gosx_studio_component_template_key",
  componentLabel: "gosx_studio_component_label",
  controlKey: "gosx_studio_control_key",
  controlKind: "gosx_studio_control_kind",
  binding: "gosx_studio_binding",
  position: "gosx_studio_position",
  value: "gosx_studio_value",
};

test.describe("@smoke GoSXStudio authoring surface workflows", () => {
  test("clicks visible create/add/duplicate/edit controls and applies managed action feedback", async ({ page }) => {
    await page.setContent(authoringSurfaceHTML());
    await page.addScriptTag({ content: runtimeJS });
    await installManagedActionHarness(page);

    await page.getByRole("button", { name: "Create page" }).click();
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Landing created.");
    await expect(page.locator("[data-gosx-studio-workbench]")).toHaveAttribute("data-gosx-studio-authoring-change-page", "landing");
    await expect(page.locator("iframe")).toHaveAttribute("src", /gosx-studio-refresh=/);

    await page.getByRole("button", { name: "Add section" }).click();
    await expect(page.locator("[data-studio-site-map-component='contact']")).toHaveAttribute("data-gosx-studio-authoring-selected", "true");
    await expect(page.locator("[data-gosx-studio-site-canvas]")).toHaveAttribute("data-gosx-studio-canvas-selected", "home-section-contact");
    await expect(page.locator("[data-gosx-studio-canvas-node='home-section-contact']")).toHaveAttribute("aria-pressed", "true");

    await page.getByRole("button", { name: "Duplicate section" }).click();
    await expect(page.locator("[data-studio-site-map-workspace-node='home-section-hero__copy_2']")).toHaveAttribute("data-gosx-studio-authoring-selected", "true");
    await expect(page.locator("[data-gosx-studio-workbench]")).toHaveAttribute("data-gosx-studio-authoring-change-component", "hero__copy_2");

    await page.getByLabel("Headline").fill("Browser authored headline");
    await page.getByRole("button", { name: "Save headline" }).click();
    await expect(page.getByLabel("Headline")).toHaveAttribute("data-gosx-studio-authoring-selected", "true");
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Hero headline saved.");

    const snapshot = await page.evaluate(() => {
      const w = window as unknown as {
        __authoringStore: {
          pages: string[];
          sections: string[];
          headline: string;
        };
        __authoringEvents: unknown[];
      };
      return {
        store: w.__authoringStore,
        eventCount: w.__authoringEvents.length,
      };
    });

    expect(snapshot.store.pages).toContain("landing");
    expect(snapshot.store.sections).toEqual(["hero", "contact", "hero__copy_2"]);
    expect(snapshot.store.headline).toBe("Browser authored headline");
    expect(snapshot.eventCount).toBe(4);
  });
});

async function installManagedActionHarness(page: import("@playwright/test").Page) {
  await page.evaluate(({ fields }) => {
    type Store = { pages: string[]; sections: string[]; headline: string };
    type Change = { key: string; label: string; kind: string; pageKey?: string; component?: string; binding?: string };

    const w = window as unknown as {
      __authoringStore: Store;
      __authoringEvents: unknown[];
    };
    w.__authoringStore = { pages: [], sections: ["hero"], headline: "Original headline" };
    w.__authoringEvents = [];
    document.addEventListener("gosxstudio:authoring-result", (event) => {
      w.__authoringEvents.push((event as CustomEvent).detail);
    });

    document.addEventListener("submit", (event) => {
      const form = event.target as HTMLFormElement | null;
      if (!form || !form.matches("[data-test-authoring-form]")) return;
      event.preventDefault();

      const payload = Object.fromEntries(new FormData(form).entries()) as Record<string, string>;
      const operation = payload[fields.operation] || "";
      const intentKind = payload[fields.intentKind] || "";
      let message = "Saved.";
      let previewURL = "/";
      let draftID = "draft";
      let change: Change = { key: "authoring-change", label: "Authoring change", kind: "component" };

      if (operation === "apply-intent" && intentKind === "create-page") {
        w.__authoringStore.pages.push("landing");
        message = "Landing created.";
        previewURL = "http://127.0.0.1:4173/pages/landing?preview=1";
        draftID = "page_landing";
        change = { key: "page-landing", label: "Landing", kind: "page", pageKey: "landing" };
      } else if (operation === "apply-intent" && intentKind === "add-component") {
        w.__authoringStore.sections.push("contact");
        message = "Contact form added to Home.";
        draftID = "rev_contact";
        change = { key: "home-section-contact", label: "Contact", kind: "component", pageKey: "home", component: "contact", binding: "home.section.contact" };
      } else if (operation === "duplicate-component") {
        w.__authoringStore.sections.push("hero__copy_2");
        message = "Hero copy 2 duplicated.";
        draftID = "rev_hero_copy";
        change = { key: "home-section-hero__copy_2", label: "Hero copy 2", kind: "component", pageKey: "home", component: "hero__copy_2", binding: "home.section.hero__copy_2" };
      } else if (operation === "save-control") {
        w.__authoringStore.headline = payload[fields.value] || "";
        message = "Hero headline saved.";
        draftID = "rev_headline";
        change = { key: "control-home-hero-headline", label: "Headline", kind: "control", pageKey: "home", component: "hero", binding: payload[fields.binding] || "home.hero.headline" };
      }

      document.dispatchEvent(new CustomEvent("gosx:form:result", {
        detail: {
          action: form.getAttribute("action") || "",
          method: form.getAttribute("method") || "post",
          result: {
            ok: true,
            message,
            data: {
              message,
        previewURL,
              draftID,
              refreshPreview: true,
              changeCount: 1,
              changes: [change],
            },
          },
        },
      }));
    });
  }, { fields });
}

function authoringSurfaceHTML(): string {
  return `
    <main class="editor-workbench" data-gosx-studio-workbench="true">
      <output data-gosx-studio-save-state="true">Dirty</output>
      <span data-gosx-studio-save-detail="true">Unsaved</span>

      <form id="intent-create-landing" action="/admin/editor/__actions/authoring" method="post" data-test-authoring-form="true">
        ${hidden(fields.operation, "apply-intent")}
        ${hidden(fields.intentKey, "create-page:landing")}
        ${hidden(fields.intentKind, "create-page")}
        ${hidden(fields.pageKey, "landing")}
        ${hidden(fields.pageLabel, "Landing")}
        ${hidden(fields.pageRoute, "/pages/landing")}
        ${hidden(fields.pageBlueprintKey, "landing")}
      </form>

      <form id="intent-add-contact" action="/admin/editor/__actions/authoring" method="post" data-test-authoring-form="true">
        ${hidden(fields.operation, "apply-intent")}
        ${hidden(fields.intentKey, "add-component:home:contact")}
        ${hidden(fields.intentKind, "add-component")}
        ${hidden(fields.pageKey, "home")}
        ${hidden(fields.componentTemplateKey, "contact")}
        ${hidden(fields.componentLabel, "Contact form")}
        ${hidden(fields.binding, "home.section.contact")}
      </form>

      <section aria-label="Ready to apply">
        <button type="submit" form="intent-create-landing">Create page</button>
        <button type="submit" form="intent-add-contact">Add section</button>
      </section>

      <form id="duplicate-hero" action="/admin/editor/__actions/authoring" method="post" data-test-authoring-form="true">
        ${hidden(fields.operation, "duplicate-component")}
        ${hidden(fields.pageKey, "home")}
        ${hidden(fields.componentKey, "hero")}
        ${hidden(fields.componentTemplateKey, "hero")}
        ${hidden(fields.componentLabel, "Hero")}
        ${hidden(fields.position, "1")}
        <button type="submit">Duplicate section</button>
      </form>

      <form id="edit-headline" action="/admin/editor/__actions/authoring" method="post" data-test-authoring-form="true">
        ${hidden(fields.operation, "save-control")}
        ${hidden(fields.pageKey, "home")}
        ${hidden(fields.componentKey, "hero")}
        ${hidden(fields.controlKey, "headline")}
        ${hidden(fields.controlKind, "rich-text")}
        ${hidden(fields.binding, "home.hero.headline")}
        <label>
          Headline
          <input
            name="${fields.value}"
            value="Original headline"
            data-gosx-studio-authoring-binding="home.hero.headline"
            data-studio-field="home.hero.headline"
          />
        </label>
        <button type="submit">Save headline</button>
      </form>

      <section data-gosx-studio-preview="true" data-gosx-studio-preview-url="http://127.0.0.1:4173/?gosx-preview=1">
        <iframe title="preview" src="http://127.0.0.1:4173/?gosx-preview=1"></iframe>
      </section>

      <section data-gosx-studio-site-canvas="true">
        <button type="button" data-gosx-studio-canvas-node="page-landing" data-gosx-studio-canvas-node-kind="page" data-gosx-studio-canvas-node-label="Landing">Landing</button>
        <button type="button" data-gosx-studio-canvas-node="home-section-hero" data-gosx-studio-canvas-node-kind="section" data-gosx-studio-canvas-node-label="Hero">Hero</button>
        <button type="button" data-gosx-studio-canvas-node="home-section-contact" data-gosx-studio-canvas-node-kind="section" data-gosx-studio-canvas-node-label="Contact">Contact</button>
        <button type="button" data-gosx-studio-canvas-node="home-section-hero__copy_2" data-gosx-studio-canvas-node-kind="section" data-gosx-studio-canvas-node-label="Hero copy 2">Hero copy 2</button>
      </section>

      <article data-studio-site-map-page="landing" data-studio-site-map-workspace-node="page-landing">Landing page</article>
      <article data-studio-site-map-page="home">Home</article>
      <article data-studio-site-map-component="hero" data-studio-site-map-binding="home.section.hero">Hero section</article>
      <article data-studio-site-map-component="contact" data-studio-site-map-binding="home.section.contact">Contact section</article>
      <article data-studio-site-map-workspace-node="home-section-hero__copy_2">Hero copy 2 section</article>
    </main>
  `;
}

function hidden(name: string, value: string): string {
  return `<input type="hidden" name="${name}" value="${value}" />`;
}
