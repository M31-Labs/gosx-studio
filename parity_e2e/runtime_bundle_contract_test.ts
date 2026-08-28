/**
 * Semantic contract for the independently-owned CMS and host runtime bundles.
 *
 * The test deliberately uses Playwright's real DOM and event implementation;
 * it is a fixture-only contract and must run in the central browser lane, not
 * from a WSL source/preflight lane. It asserts observable behavior instead of
 * comparing the two workbench bundles byte-for-byte.
 */

import { expect, test, type Page } from "@playwright/test";
import path from "node:path";

type RuntimeVariant = "cms" | "host";

const assetRoot: Record<RuntimeVariant, string> = {
  cms: path.resolve(__dirname, "../cms/studio/assets"),
  host: path.resolve(__dirname, "../hostruntime/assets"),
};
const PREVIEW_RUNTIME = path.resolve(__dirname, "../previewruntime/island_runtime.js");

const observedEvents = [
  "gosxstudio:command",
  "gosxstudio:mode-change",
  "gosxstudio:viewport-change",
  "gosxstudio:zoom-change",
  "gosxstudio:workbench-layout",
  "gosxstudio:save-state",
  "gosxstudio:history-state",
  "gosxstudio:history-restore",
  "gosxstudio:preview-patch",
] as const;

function fixtureHTML(): string {
  return `
    <base href="http://127.0.0.1:4173/">
    <form data-studio-workbench="true" data-gosx-studio-state="true" action="/save" method="post">
      <div data-studio-layout></div>
      <button type="button" data-studio-mode-control="style">Style</button>
      <button type="button" data-studio-mode-control="content">Content</button>
      <section data-studio-mode-panel="structure"></section>
      <section data-studio-mode-panel="style"></section>
      <section data-studio-mode-panel="content"></section>
      <output data-studio-mode-label></output>
      <button type="button" data-studio-viewport="mobile" data-studio-viewport-width="375px">Mobile</button>
      <output data-studio-viewport-label></output>
      <button type="button" data-studio-zoom="80%">80%</button>
      <button type="button" data-gosx-studio-history-undo>Undo</button>
      <button type="button" data-gosx-studio-history-redo>Redo</button>
      <output data-gosx-studio-history-status></output>
      <output data-gosx-studio-save-state>Saved</output>
      <span data-gosx-studio-save-detail>Ready</span>
      <span data-gosx-studio-dirty-count></span>
      <button type="submit" data-gosx-studio-save-button>Save</button>
      <label>Title <input name="title" value="before" data-studio-field-source="title"></label>
      <div data-gosx-studio-preview="true">
        <iframe data-studio-preview-frame="true" data-studio-preview-src="/preview?route=%2Fcontract" src="/preview?route=%2Fcontract"></iframe>
      </div>
      <div data-studio-command-palette data-studio-command-state="closed">
        <button type="button" data-studio-command-open>Commands</button>
        <div data-studio-command-overlay hidden>
          <input type="search" data-studio-command-search>
          <div data-studio-command-list>
            <button type="button" data-studio-command="style-mode" data-studio-command-kind="mode" data-studio-command-target="style" data-studio-command-search-text="Style mode">Style mode</button>
          </div>
          <div data-studio-command-empty hidden></div>
        </div>
      </div>
    </form>`;
}

function previewHTML(): string {
  return `
    <!doctype html>
    <html data-studio-route="/contract">
      <body>
        <main data-studio-page-id="contract">
          <span data-studio-field="title">before</span>
        </main>
      </body>
    </html>`;
}

async function installEventAccounting(page: Page): Promise<void> {
  await page.evaluate((eventNames) => {
    const events: Record<string, unknown[]> = {};
    for (const name of eventNames) {
      events[name] = [];
      document.querySelector("form")?.addEventListener(name, (event) => {
        events[name].push((event as CustomEvent).detail);
      });
    }
    (window as unknown as { __runtimeContractEvents: Record<string, unknown[]> }).__runtimeContractEvents = events;
  }, observedEvents);
}

async function installHostPreviewInstrumentation(page: Page): Promise<void> {
  await page.evaluate(() => {
    const calls = {
      bind: 0,
      bindResults: [] as Array<{ handled: boolean; frames: number; bound: number }>,
      postPatches: [] as Array<{ reason: string; field: string }>,
      postResults: [] as Array<{ handled: boolean; count: number }>,
    };
    const win = window as unknown as {
      __runtimeContractPreviewCalls: typeof calls;
      __runtimeContractPreviewActual: { bind: boolean; postPatch: boolean };
      __gosx_preview_runtime_island_bindFrames?: (form: Element, host: unknown) => unknown;
      __gosx_preview_runtime_island_postPatch?: (form: Element, reason: string, detail: unknown, field: { name?: string } | null) => unknown;
    };
    const realBind = win.__gosx_preview_runtime_island_bindFrames;
    const realPostPatch = win.__gosx_preview_runtime_island_postPatch;
    if (typeof realBind !== "function" || typeof realPostPatch !== "function") {
      throw new Error("real preview runtime island globals must load before host instrumentation");
    }
    win.__runtimeContractPreviewCalls = calls;
    win.__runtimeContractPreviewActual = { bind: true, postPatch: true };
    win.__gosx_preview_runtime_island_bindFrames = (form, host) => {
      calls.bind += 1;
      const result = realBind(form, host) as { handled?: boolean; frames?: number; bound?: number } | undefined;
      calls.bindResults.push({
        handled: result?.handled === true,
        frames: Number(result?.frames || 0),
        bound: Number(result?.bound || 0),
      });
      return result;
    };
    win.__gosx_preview_runtime_island_postPatch = (form, reason, detail, field) => {
      calls.postPatches.push({ reason, field: field?.name ?? "" });
      const result = realPostPatch(form, reason, detail, field) as { handled?: boolean; count?: number } | undefined;
      calls.postResults.push({ handled: result?.handled === true, count: Number(result?.count || 0) });
      return result;
    };
  });
}

async function mountVariant(page: Page, variant: RuntimeVariant): Promise<void> {
  await page.route("http://127.0.0.1:4173/runtime-contract", (route) => route.fulfill({
    status: 200,
    contentType: "text/html",
    body: fixtureHTML(),
  }));
  await page.route("http://127.0.0.1:4173/preview**", (route) => route.fulfill({
    status: 200,
    contentType: "text/html",
    body: previewHTML(),
  }));
  await page.goto("http://127.0.0.1:4173/runtime-contract", { waitUntil: "load" });
  await page.waitForFunction(() => {
    const frame = document.querySelector("iframe[data-studio-preview-frame]") as HTMLIFrameElement | null;
    return !!frame?.contentDocument?.querySelector("[data-studio-field='title']");
  });
  await installEventAccounting(page);
  for (const script of ["command_palette.js", "state_runtime.js", "workbench_runtime.js"]) {
    await page.addScriptTag({ path: path.join(assetRoot[variant], script) });
  }
  if (variant === "host") {
    // hostruntime/runtime.go places previewruntime.Bundle immediately after
    // workbenchruntime.Bundle. Load the real editor-side island at that same
    // boundary; the wrapper below records calls while delegating unchanged.
    await page.addScriptTag({ path: PREVIEW_RUNTIME });
    await installHostPreviewInstrumentation(page);
    await page.evaluate(() => {
      const win = window as unknown as {
        __gosx_preview_runtime_island_bindFrames?: (form: Element, host: unknown) => unknown;
      };
      const form = document.querySelector("form[data-studio-workbench]");
      const bindFrames = win.__gosx_preview_runtime_island_bindFrames;
      if (!form || typeof bindFrames !== "function") throw new Error("preview runtime bindFrames is unavailable");
      const result = bindFrames(form, {});
      if (!(result as { handled?: boolean } | undefined)?.handled) throw new Error("preview runtime frame binding failed");
    });
  }
}

async function events(page: Page): Promise<Record<string, unknown[]>> {
  return await page.evaluate(() =>
    (window as unknown as { __runtimeContractEvents: Record<string, unknown[]> }).__runtimeContractEvents,
  );
}

async function assertCommonBehavior(page: Page, variant: RuntimeVariant, pageErrors: string[]): Promise<void> {
  const form = page.locator("form[data-studio-workbench]");
  const frame = form.locator("[data-studio-preview-frame]");
  const previewTitle = page.frameLocator("iframe[data-studio-preview-frame]").locator("[data-studio-field='title']");
  await expect(form).toHaveAttribute("data-studio-mode", "structure");
  await expect(form).toHaveAttribute("data-studio-breakpoint", "desktop");
  await expect(form).toHaveAttribute("data-studio-zoom", "fit");
  await expect(frame).toHaveAttribute("src", "/preview?route=%2Fcontract");
  await expect(frame).toHaveAttribute("data-studio-preview-src", "/preview?route=%2Fcontract");

  await page.keyboard.press("Control+K");
  await expect(page.locator("[data-studio-command-palette]")).toHaveAttribute("data-studio-command-state", "open");
  await page.locator("[data-studio-command='style-mode']").click();
  await expect(form).toHaveAttribute("data-studio-mode", "style");

  await page.locator("[data-studio-viewport='mobile']").click();
  await expect(form).toHaveAttribute("data-studio-breakpoint", "mobile");
  await expect(form.locator("[data-studio-viewport='mobile']")).toHaveAttribute("aria-pressed", "true");
  await expect(form.locator("[data-studio-preview-frame]")).toHaveAttribute("data-studio-preview-viewport", "mobile");
  await expect(form.locator("[data-studio-preview-frame]")).toHaveCSS("width", "375px");

  await page.locator("[data-studio-zoom='80%']").click();
  await expect(form).toHaveAttribute("data-studio-zoom", "80%");
  await expect(form.locator("[data-gosx-studio-preview]")).toHaveAttribute("data-studio-canvas-zoom", "80%");

  await page.getByLabel("Title").fill("after");
  await expect(previewTitle).toHaveText("after");
  await expect(previewTitle).toHaveAttribute("data-gosx-studio-preview-patched", "true");
  await expect(form).toHaveAttribute("data-gosx-studio-save-state", "dirty");
  await expect(form).toHaveAttribute("data-studio-dirty-state", "dirty");
  await expect(form).toHaveAttribute("data-gosx-studio-history-length", "2");
  await expect(form.locator("[data-gosx-studio-history-undo]")).toBeEnabled();

  await form.locator("[data-gosx-studio-history-undo]").click();
  await expect(page.getByLabel("Title")).toHaveValue("before");
  await expect(previewTitle).toHaveText("before");
  await expect(previewTitle).toHaveAttribute("data-gosx-studio-preview-patched", "true");
  await expect(form).toHaveAttribute("data-gosx-studio-save-state", "saved");

  const seen = await events(page);
  expect(seen["gosxstudio:command"]?.some((detail) => (detail as { kind?: string }).kind === "mode")).toBe(true);
  expect(seen["gosxstudio:mode-change"]?.some((detail) => (detail as { mode?: string }).mode === "style")).toBe(true);
  expect(seen["gosxstudio:viewport-change"]?.some((detail) => (detail as { viewport?: string; width?: string }).viewport === "mobile" && (detail as { width?: string }).width === "375px")).toBe(true);
  expect(seen["gosxstudio:zoom-change"]?.some((detail) => (detail as { zoom?: string; scale?: number }).zoom === "80%" && (detail as { scale?: number }).scale === 0.8)).toBe(true);
  expect(seen["gosxstudio:workbench-layout"]?.length ?? 0).toBeGreaterThan(0);
  expect(seen["gosxstudio:save-state"]?.some((detail) => (detail as { state?: string }).state === "dirty")).toBe(true);
  expect(seen["gosxstudio:history-state"]?.some((detail) => (detail as { length?: number }).length === 2)).toBe(true);
  expect(seen["gosxstudio:history-restore"]?.length ?? 0).toBeGreaterThan(0);

  if (variant === "cms") {
    expect(seen["gosxstudio:preview-patch"]?.length ?? 0).toBeGreaterThan(0);
  } else {
    expect(seen["gosxstudio:preview-patch"]?.length ?? 0).toBeGreaterThan(0);
    const calls = await page.evaluate(() =>
      (window as unknown as {
        __runtimeContractPreviewActual: { bind: boolean; postPatch: boolean };
        __runtimeContractPreviewCalls: {
          bind: number;
          bindResults: Array<{ handled: boolean; frames: number; bound: number }>;
          postPatches: Array<{ reason: string; field: string }>;
          postResults: Array<{ handled: boolean; count: number }>;
        };
      }).__runtimeContractPreviewCalls,
    );
    const actual = await page.evaluate(() =>
      (window as unknown as { __runtimeContractPreviewActual: { bind: boolean; postPatch: boolean } }).__runtimeContractPreviewActual,
    );
    expect(actual).toEqual({ bind: true, postPatch: true });
    expect(calls.bind).toBeGreaterThan(0);
    expect(calls.bindResults.some((result) => result.handled && result.frames === 1)).toBe(true);
    expect(calls.postPatches.some((call) => call.reason === "input" && call.field === "title")).toBe(true);
    expect(calls.postPatches.some((call) => call.reason === "history-restore")).toBe(true);
    expect(calls.postResults.some((result) => result.handled && result.count > 0)).toBe(true);
  }
  expect(pageErrors).toEqual([]);
}

for (const variant of ["cms", "host"] as const) {
  test(`${variant} runtime bundles preserve the shared semantic contract`, async ({ page }) => {
    const pageErrors: string[] = [];
    page.on("pageerror", (error) => pageErrors.push(error.message));
    await mountVariant(page, variant);
    await assertCommonBehavior(page, variant, pageErrors);
  });
}
