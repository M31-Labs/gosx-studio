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
        data-studio-site-map-detail-control="runtime"
        data-studio-site-map-focus-control="runtime"
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

  test("rebinds boards that already carry the public bound attribute", async ({ page }) => {
    await page.setContent(`
      <section
        data-studio-site-map-board="true"
        data-gosx-studio-site-map-runtime-bound="true"
        data-studio-site-map-detail="map"
        data-studio-site-map-detail-control="runtime"
        data-studio-site-map-palette="all"
        data-studio-site-map-focus="all"
        data-studio-site-map-focus-control="runtime"
      >
        <button type="button" data-studio-site-map-detail-target="build" aria-pressed="false">Add</button>
        <div data-studio-site-map-workspace="true" data-studio-site-map-panel-visible="true"></div>
        <div data-studio-site-map-builder="true" data-studio-site-map-panel-visible="false"></div>
      </section>
    `);
    await page.addScriptTag({ content: runtimeJS });

    const board = page.locator("[data-studio-site-map-board='true']");
    await page.locator("[data-studio-site-map-detail-target='build']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-detail", "build");
    await expect(board).toHaveAttribute("data-studio-site-map-palette", "page-sections");
    await expect(page.locator("[data-studio-site-map-workspace='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "false");
    await expect(page.locator("[data-studio-site-map-builder='true']")).toHaveAttribute("data-studio-site-map-panel-visible", "true");
  });
});

const panZoomBoard = `
  <section
    data-studio-site-map-board="true"
    tabindex="0"
    data-studio-site-map-scale="1"
    data-studio-site-map-pan-x="0"
    data-studio-site-map-pan-y="0"
  >
    <div class="studio-site-map-board__view" role="group">
      <button type="button" data-studio-site-map-zoom-action="out">-</button>
      <button type="button" data-studio-site-map-zoom-action="reset">Reset</button>
      <button type="button" data-studio-site-map-zoom-action="in">+</button>
      <output data-studio-site-map-zoom-readout>100%</output>
    </div>
    <div data-studio-site-map-workspace="true" data-studio-site-map-panel-visible="true">
      <div data-studio-site-map-canvas="true" style="width:600px;height:400px;position:relative;overflow:hidden;">
        <div data-studio-site-map-pan-surface="true">
          <label
            data-studio-site-map-workspace-node="home"
            data-studio-site-map-group="site"
            data-studio-site-map-node-label="Home"
          >Home</label>
        </div>
      </div>
    </div>
  </section>
`;

test.describe("@smoke GoSXStudioSiteMapRuntime infinite-canvas pan and zoom", () => {
  test("applies the initial viewport transform on bind", async ({ page }) => {
    await page.setContent(panZoomBoard);
    await page.addScriptTag({ content: runtimeJS });

    const surface = page.locator("[data-studio-site-map-pan-surface='true']");
    await expect(surface).toHaveAttribute("style", /scale\(1\)/);
    await expect(page.locator("[data-studio-site-map-zoom-readout]")).toHaveText("100%");
  });

  test("zoom action buttons scale about the viewport center", async ({ page }) => {
    await page.setContent(panZoomBoard);
    await page.addScriptTag({ content: runtimeJS });

    const board = page.locator("[data-studio-site-map-board='true']");
    const surface = page.locator("[data-studio-site-map-pan-surface='true']");
    const readout = page.locator("[data-studio-site-map-zoom-readout]");

    await page.locator("[data-studio-site-map-zoom-action='in']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-scale", "1.25");
    await expect(readout).toHaveText("125%");
    await expect(surface).toHaveAttribute("style", /scale\(1\.25\)/);

    await page.locator("[data-studio-site-map-zoom-action='out']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-scale", "1");

    await page.locator("[data-studio-site-map-zoom-action='out']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-scale", "0.8");
    await expect(readout).toHaveText("80%");

    await page.locator("[data-studio-site-map-zoom-action='reset']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-scale", "1");
    await expect(board).toHaveAttribute("data-studio-site-map-pan-x", "0");
    await expect(board).toHaveAttribute("data-studio-site-map-pan-y", "0");
    await expect(readout).toHaveText("100%");
  });

  test("setState applies a pan translation deterministically", async ({ page }) => {
    await page.setContent(panZoomBoard);
    await page.addScriptTag({ content: runtimeJS });

    await page.evaluate(() => {
      const board = document.querySelector("[data-studio-site-map-board='true']");
      window.GoSXStudioSiteMapRuntime.setState(board, { panX: 48, panY: 24, scale: 1.5 });
    });

    const board = page.locator("[data-studio-site-map-board='true']");
    const surface = page.locator("[data-studio-site-map-pan-surface='true']");
    await expect(board).toHaveAttribute("data-studio-site-map-pan-x", "48");
    await expect(board).toHaveAttribute("data-studio-site-map-pan-y", "24");
    await expect(surface).toHaveAttribute("style", /translate\(48px, 24px\) scale\(1\.5\)/);
  });

  test("wheel over the canvas zooms toward the pointer", async ({ page }) => {
    await page.setContent(panZoomBoard);
    await page.addScriptTag({ content: runtimeJS });

    const canvas = page.locator("[data-studio-site-map-canvas='true']");
    const box = await canvas.boundingBox();
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.wheel(0, -100);

    const scale = await page.locator("[data-studio-site-map-board='true']").getAttribute("data-studio-site-map-scale");
    expect(Number(scale)).toBeGreaterThan(1);
  });

  test("keyboard +/-/0 zooms in, out, and resets", async ({ page }) => {
    await page.setContent(panZoomBoard);
    await page.addScriptTag({ content: runtimeJS });

    const board = page.locator("[data-studio-site-map-board='true']");
    await board.focus();

    await page.keyboard.press("=");
    await expect(board).toHaveAttribute("data-studio-site-map-scale", "1.25");

    await page.keyboard.press("0");
    await expect(board).toHaveAttribute("data-studio-site-map-scale", "1");
  });

  test("dragging the canvas background pans the surface", async ({ page }) => {
    await page.setContent(panZoomBoard);
    await page.addScriptTag({ content: runtimeJS });

    const canvas = page.locator("[data-studio-site-map-canvas='true']");
    const box = await canvas.boundingBox();
    // Start in the empty bottom-right of the canvas, away from the node.
    const startX = box.x + box.width - 24;
    const startY = box.y + box.height - 24;
    await page.mouse.move(startX, startY);
    await page.mouse.down();
    await page.mouse.move(startX - 60, startY - 40, { steps: 4 });
    await page.mouse.up();

    const board = page.locator("[data-studio-site-map-board='true']");
    expect(Number(await board.getAttribute("data-studio-site-map-pan-x"))).toBeLessThan(0);
    expect(Number(await board.getAttribute("data-studio-site-map-pan-y"))).toBeLessThan(0);
  });
});

const marqueeBoard = `
  <section
    data-studio-site-map-board="true"
    tabindex="0"
    data-studio-site-map-scale="1"
    data-studio-site-map-pan-x="0"
    data-studio-site-map-pan-y="0"
    data-studio-site-map-selected-node=""
    data-studio-site-map-selected-nodes=""
    data-studio-site-map-filter="all"
  >
    <div data-studio-site-map-workspace="true" data-studio-site-map-panel-visible="true">
      <div data-studio-site-map-canvas="true" style="width:600px;height:400px;position:relative;overflow:hidden;">
        <div data-studio-site-map-pan-surface="true" style="position:absolute;inset:0;">
          <label data-studio-site-map-workspace-node="alpha" data-studio-site-map-group="site"
                 style="position:absolute;left:40px;top:40px;width:80px;height:40px;">Alpha</label>
          <label data-studio-site-map-workspace-node="beta" data-studio-site-map-group="site"
                 style="position:absolute;left:440px;top:320px;width:80px;height:40px;">Beta</label>
        </div>
      </div>
    </div>
  </section>
`;

test.describe("@smoke GoSXStudioSiteMapRuntime infinite-canvas marquee and multi-select", () => {
  test("shift-drag rubber-band selects intersecting nodes only", async ({ page }) => {
    await page.setContent(marqueeBoard);
    await page.addScriptTag({ content: runtimeJS });

    const canvas = page.locator("[data-studio-site-map-canvas='true']");
    const box = await canvas.boundingBox();
    await page.keyboard.down("Shift");
    await page.mouse.move(box.x + 16, box.y + 16);
    await page.mouse.down();
    await page.mouse.move(box.x + 180, box.y + 140, { steps: 5 });
    await page.mouse.up();
    await page.keyboard.up("Shift");

    const board = page.locator("[data-studio-site-map-board='true']");
    await expect(board).toHaveAttribute("data-studio-site-map-selected-nodes", "alpha");
    await expect(page.locator("[data-studio-site-map-workspace-node='alpha']")).toHaveAttribute("data-studio-site-map-node-selected", "true");
    await expect(page.locator("[data-studio-site-map-workspace-node='beta']")).toHaveAttribute("data-studio-site-map-node-selected", "false");
  });

  test("shift-click toggles a node in and out of the multi-selection", async ({ page }) => {
    await page.setContent(marqueeBoard);
    await page.addScriptTag({ content: runtimeJS });

    const board = page.locator("[data-studio-site-map-board='true']");
    const beta = page.locator("[data-studio-site-map-workspace-node='beta']");

    await page.locator("[data-studio-site-map-workspace-node='alpha']").click();
    await expect(board).toHaveAttribute("data-studio-site-map-selected-node", "alpha");

    await beta.click({ modifiers: ["Shift"] });
    await expect(board).toHaveAttribute("data-studio-site-map-selected-nodes", "beta");
    await expect(beta).toHaveAttribute("data-studio-site-map-node-selected", "true");

    await beta.click({ modifiers: ["Shift"] });
    await expect(board).toHaveAttribute("data-studio-site-map-selected-nodes", "");
    await expect(beta).toHaveAttribute("data-studio-site-map-node-selected", "false");
  });

  test("Escape clears the multi-selection", async ({ page }) => {
    await page.setContent(marqueeBoard);
    await page.addScriptTag({ content: runtimeJS });

    await page.evaluate(() => {
      const board = document.querySelector("[data-studio-site-map-board='true']");
      window.GoSXStudioSiteMapRuntime.setState(board, { selectedNodes: ["alpha", "beta"] });
    });
    const board = page.locator("[data-studio-site-map-board='true']");
    await expect(board).toHaveAttribute("data-studio-site-map-selected-nodes", "alpha,beta");

    await board.focus();
    await page.keyboard.press("Escape");
    await expect(board).toHaveAttribute("data-studio-site-map-selected-nodes", "");
  });

  test("multi-selection survives a sync triggered by another state change", async ({ page }) => {
    await page.setContent(marqueeBoard);
    await page.addScriptTag({ content: runtimeJS });

    await page.evaluate(() => {
      const board = document.querySelector("[data-studio-site-map-board='true']");
      window.GoSXStudioSiteMapRuntime.setState(board, { selectedNodes: ["alpha"] });
      window.GoSXStudioSiteMapRuntime.setState(board, { filter: "site" });
    });

    await expect(page.locator("[data-studio-site-map-workspace-node='alpha']")).toHaveAttribute("data-studio-site-map-node-selected", "true");
  });
});
