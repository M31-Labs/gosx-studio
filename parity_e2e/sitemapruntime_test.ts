import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../sitemapruntime/island_runtime.js"),
  "utf8",
);

test.describe("@smoke GoSXStudioSiteMapRuntime interactions", () => {
  test("drives board tabs, filters, palette state, and node selection", async ({ page }) => {
    await page.setContent(`
      <section
        data-studio-site-map-board="true"
        data-studio-site-map-filter="all"
        data-studio-site-map-detail="map"
        data-studio-site-map-palette="all"
        data-studio-site-map-focus="all"
        data-studio-site-map-zoom="fit"
        data-studio-site-map-density="roomy"
        data-studio-site-map-grid="on"
        data-studio-site-map-selected-node="home-section-hero"
      >
        <button type="button" data-studio-site-map-filter-control="content" aria-pressed="false">Content</button>
        <button type="button" data-studio-site-map-detail-target="build" aria-pressed="false">Add</button>
        <button type="button" data-studio-site-map-detail-target="components" aria-pressed="false">Components</button>
        <button type="button" data-studio-site-map-palette-control="content" aria-pressed="false">Content blocks</button>
        <button type="button" data-studio-site-map-zoom-control="wide" aria-pressed="false">Wide</button>
        <button type="button" data-studio-site-map-density-control="dense" aria-pressed="false">Dense</button>
        <button type="button" data-studio-site-map-grid-control="off" aria-pressed="false">Plain</button>

        <div data-studio-site-map-workspace="true" data-studio-site-map-panel-visible="true">
          <section data-studio-site-map-workspace-layer="home" data-studio-site-map-visible="true">
            <input class="studio-site-map-workspace-node__input" value="home-section-hero" />
            <label
              data-studio-site-map-workspace-node="home-section-hero"
              data-studio-site-map-group="site"
              data-studio-site-map-node-kind="component"
              data-studio-site-map-node-kind-label="Section"
              data-studio-site-map-node-label="Hero"
              data-studio-site-map-node-summary="Lead story"
              data-studio-site-map-route="/"
              data-studio-site-map-node-source-label="Site"
              data-studio-site-map-node-binding-label="home.section.hero"
              data-studio-site-map-node-status-label="Editable"
              data-studio-site-map-node-component-label="HomeHero"
            >Hero</label>
            <label
              data-studio-site-map-workspace-node="journal-list"
              data-studio-site-map-group="content"
              data-studio-site-map-node-kind="component"
              data-studio-site-map-node-kind-label="Section"
              data-studio-site-map-node-label="Journal"
              data-studio-site-map-node-summary="Recent posts"
              data-studio-site-map-route="/blog"
              data-studio-site-map-node-source-label="Content"
              data-studio-site-map-node-binding-label="blog.collection"
              data-studio-site-map-node-status-label="Draftable"
              data-studio-site-map-node-component-label="PostList"
            >Journal</label>
          </section>
          <span
            data-studio-site-map-workspace-link="hero-to-journal"
            data-studio-site-map-link-from="home-section-hero"
            data-studio-site-map-link-to="journal-list"
            data-studio-site-map-link-from-group="site"
            data-studio-site-map-link-to-group="content"
            data-studio-site-map-link-selected="true"
          >Link</span>
        </div>

        <div data-studio-site-map-builder="true" data-studio-site-map-panel-visible="false">
          <label data-studio-site-map-blueprint="landing">Landing</label>
          <article data-studio-composition-intent="create-landing" data-studio-composition-kind="create-page" data-studio-composition-blueprint="landing" data-studio-composition-active="false">Create landing</article>
          <label data-studio-site-map-component-template="hero" data-studio-site-map-template-category="page-sections">Hero template</label>
          <label data-studio-site-map-component-template="post-list" data-studio-site-map-template-category="content">Post list</label>
          <article data-studio-composition-intent="add-post-list" data-studio-composition-kind="add-component" data-studio-composition-template="post-list" data-studio-composition-active="false">Add posts</article>
        </div>
        <div data-studio-site-map-viewport="true" data-studio-site-map-panel-visible="false"></div>

        <aside data-studio-site-map-selection-card="true">
          <span data-studio-site-map-selected-kind></span>
          <strong data-studio-site-map-selected-label></strong>
          <small data-studio-site-map-selected-summary-row><span data-studio-site-map-selected-summary></span></small>
          <span data-studio-site-map-selected-route-row><strong data-studio-site-map-selected-route></strong></span>
          <span data-studio-site-map-selected-source-row><strong data-studio-site-map-selected-source></strong></span>
          <span data-studio-site-map-selected-binding-row><strong data-studio-site-map-selected-binding></strong></span>
          <span data-studio-site-map-selected-component-row><strong data-studio-site-map-selected-component></strong></span>
          <output data-studio-site-map-selected-status></output>
        </aside>
      </section>
    `);
    await page.addScriptTag({ content: runtimeJS });

    const board = page.locator("[data-studio-site-map-board='true']");
    await expect(board).toHaveAttribute("data-gosx-studio-site-map-runtime-bound", "true");

    await page.locator("[data-studio-site-map-detail-target='build']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-detail", "build");
    await expect(board).toHaveAttribute("data-studio-site-map-palette", "page-sections");
    await expect(page.locator("[data-studio-site-map-workspace='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "false");
    await expect(page.locator("[data-studio-site-map-builder='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "true");

    await page.locator("[data-studio-site-map-palette-control='content']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-palette", "content");
    await expect(page.locator("[data-studio-site-map-component-template='hero']")).toHaveAttribute("data-studio-site-map-visible", "false");
    await expect(page.locator("[data-studio-site-map-component-template='post-list']")).toHaveAttribute("data-studio-site-map-visible", "true");

    await page.locator("[data-studio-site-map-filter-control='content']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-filter", "content");
    await expect(page.locator("[data-studio-site-map-workspace-node='home-section-hero']")).toHaveAttribute("data-studio-site-map-visible", "false");
    await expect(page.locator("[data-studio-site-map-workspace-node='journal-list']")).toHaveAttribute("data-studio-site-map-visible", "true");

    await page.locator("[data-studio-site-map-workspace-node='journal-list']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-selected-node", "journal-list");
    await expect(page.locator("[data-studio-site-map-selected-label]")).toHaveText("Journal");
    await expect(page.locator("[data-studio-site-map-selected-route]")).toHaveText("/blog");
    await expect(page.locator("[data-studio-site-map-selected-status]")).toHaveText("Draftable");

    await page.locator("[data-studio-site-map-blueprint='landing']").click();
    await expect(page.locator("[data-studio-composition-intent='create-landing']")).toHaveAttribute("data-studio-composition-active", "true");

    await page.locator("[data-studio-site-map-component-template='post-list']").click();
    await expect(page.locator("[data-studio-composition-intent='add-post-list']")).toHaveAttribute("data-studio-composition-active", "true");

    await page.locator("[data-studio-site-map-zoom-control='wide']").click();
    await page.locator("[data-studio-site-map-density-control='dense']").click();
    await page.locator("[data-studio-site-map-grid-control='off']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-zoom", "wide");
    await expect(board).toHaveAttribute("data-studio-site-map-density", "dense");
    await expect(board).toHaveAttribute("data-studio-site-map-grid", "off");
  });
});
