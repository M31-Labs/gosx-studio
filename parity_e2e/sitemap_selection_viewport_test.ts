import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtime = readFileSync(path.join(__dirname, "../sitemapruntime/island_runtime.js"), "utf8");

for (const detached of [false, true]) {
  test(`site-map selection ${detached ? "leaves a detached viewport alone" : "preserves the viewport across host mutation bindings"}`, async ({ page }) => {
    await page.setContent(`<style>body{min-height:2400px}</style>
      <section data-studio-site-map-board="true">
        <article data-studio-site-map-workspace-node="page:home" data-studio-site-map-node-label="Home"></article>
        <span data-studio-site-map-selected-label>Hero</span>
      </section>`);
    await page.addScriptTag({ content: runtime });
    await page.evaluate((detach) => {
      const root = document.querySelector("[data-studio-site-map-board]")!;
      const label = root.querySelector("[data-studio-site-map-selected-label]")!;
      const observer = new MutationObserver(() => window.scrollTo(0, 0));
      observer.observe(label, { childList: true });
      window.scrollTo(0, 200);
      (window as any).GoSXStudioSiteMapRuntime.setState(root, { selectedNode: "page:home" });
      if (detach) root.remove();
    }, detached);
    await page.evaluate(() => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve()))));
    expect(await page.evaluate(() => scrollY)).toBe(detached ? 0 : 200);
    if (!detached) await expect(page.locator("[data-studio-site-map-selected-label]")).toHaveText("Home");
  });
}
