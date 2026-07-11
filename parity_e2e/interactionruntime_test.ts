import { test, expect } from "@playwright/test";
import fs from "node:fs";
import path from "node:path";
import vm from "node:vm";

const runtime = fs.readFileSync(path.join(__dirname, "..", "interactionruntime", "runtime.js"), "utf8");

test("public interaction runtime preserves keyboard parity and ordered actions", async ({ page }) => {
  const graphs = JSON.stringify([{ id: "g", event: "click", enabled: true, conditions: [{ kind: "always" }], actions: [
    { id: "show", kind: "show", targetId: "details" },
    { id: "class", kind: "toggle-class", targetId: "details", value: "open" },
    { id: "focus", kind: "focus", targetId: "details" },
  ] }]);
  await page.setContent(`<div id="trigger" data-gosx-interactions='${graphs}'></div><button id="details" hidden>Details</button>`);
  await page.addScriptTag({ content: runtime });
  await page.locator("#trigger").focus();
  await page.keyboard.press("Enter");
  await expect(page.locator("#details")).toBeVisible();
  await expect(page.locator("#details")).toHaveClass(/open/);
  await expect(page.locator("#details")).toBeFocused();
});

test("hover has focus parity and reduced motion removes delays", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  const graphs = JSON.stringify([{ id: "g", event: "hover", enabled: true, actions: [{ id: "show", kind: "show", targetId: "details", delayMs: 60000 }] }]);
  await page.setContent(`<div id="trigger" tabindex="0" data-gosx-interactions='${graphs}'></div><div id="details" hidden>Details</div>`);
  await page.addScriptTag({ content: runtime });
  await page.locator("#trigger").focus();
  await expect(page.locator("#details")).toBeVisible({ timeout: 1000 });
});

test("malformed graphs and unsafe runtime navigation are inert", async ({ page }) => {
  await page.setContent(`<button id="bad" data-gosx-interactions='not json'>Bad</button><button id="nav" data-gosx-interactions='[{"event":"click","enabled":true,"actions":[{"kind":"navigate","value":"javascript:alert(1)"}]}]'>Nav</button>`);
  await page.addScriptTag({ content: runtime });
  await page.locator("#bad").click();
  await page.locator("#nav").click();
  await expect(page).toHaveURL("about:blank");
});

test("runtime is SSR inert and introduces no responsive overflow", async ({ page }) => {
  expect(() => vm.runInNewContext(runtime, {})).not.toThrow();
  const graphs = JSON.stringify([{ id: "g", event: "click", enabled: true, actions: [{ id: "show", kind: "show", targetId: "details" }] }]);
  await page.setContent(`<main style="max-width:100%"><button data-gosx-interactions='${graphs}'>Open</button><div id="details" hidden>Details</div></main>`);
  await page.addScriptTag({ content: runtime });
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 800 });
    const dimensions = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth, client: document.documentElement.clientWidth }));
    expect(dimensions.scroll).toBeLessThanOrEqual(dimensions.client + 1);
  }
});
