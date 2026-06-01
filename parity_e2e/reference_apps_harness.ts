import { expect, type APIRequestContext, type Page } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { createServer } from "node:net";

const workRoot = path.resolve(__dirname, "../..");
const muddyRepo = process.env.GOSX_STUDIO_MUDDY_REPO ?? path.join(workRoot, "muddy-noni-commerce");
const pajaritosRepo = process.env.GOSX_STUDIO_PAJARITOS_REPO ?? path.join(workRoot, "pajaritos-forest-school");

export type ClickAuthoringOptions = {
  noWaitAfter?: boolean;
  reloadAfter?: boolean;
  settleAfter?: boolean;
};

export type ServerHandle = {
  baseURL: string;
  stop: () => Promise<void>;
};

export type StudioAuthoringResultDetail = {
  action?: string;
  method?: string;
  result?: {
    message?: string;
    previewURL?: string;
    refreshPreview?: boolean;
  };
  change?: {
    key?: string;
    kind?: string;
    pageKey?: string;
    component?: string;
    binding?: string;
  };
  selectedCount?: number;
  previewCount?: number;
};

export async function clickIntent(page: Page, intentKey: string) {
  return clickAuthoringButton(page, `[data-studio-composition-intent="${intentKey}"] button`);
}

export async function clickAuthoringPanel(page: Page, panelSelector: string, options?: ClickAuthoringOptions) {
  return clickAuthoringButton(page, `${panelSelector} button`, options);
}

export async function clickAuthoringButton(page: Page, buttonSelector: string, options?: ClickAuthoringOptions) {
  return clickEditorActionButton(page, buttonSelector, "/__actions/authoring", options);
}

export async function clickEditorActionButton(page: Page, buttonSelector: string, actionPathPart: string, options?: ClickAuthoringOptions) {
  const navigationPromise = page.waitForEvent("framenavigated", {
    predicate: (frame) => frame === page.mainFrame(),
    timeout: 3_000,
  }).catch(() => null);
  const responsePromise = page.waitForResponse((response) =>
    response.url().includes(actionPathPart) &&
    response.request().method() === "POST",
  );
  await page.locator(buttonSelector).first().click({ noWaitAfter: options?.noWaitAfter === true });
  const response = await responsePromise;
  await navigationPromise;
  if (options?.settleAfter !== false) {
    await page.waitForLoadState("networkidle");
  }
  if (options?.reloadAfter !== false) {
    await page.goto(page.url(), { waitUntil: "networkidle" });
  }
  return response;
}

export async function expectIntentButtonReceivesPointer(page: Page, intentKey: string) {
  await expectButtonReceivesPointer(page, `[data-studio-composition-intent="${intentKey}"] button`, intentKey);
}

export async function expectPanelButtonReceivesPointer(page: Page, panelSelector: string) {
  await expectButtonReceivesPointer(page, `${panelSelector} button`, panelSelector);
}

export async function expectPajaritosPublishingReadiness(page: Page) {
  const publishing = page.locator("#publishing");
  await expect(publishing).toContainText("Publishing checklist");
  await expect(publishing).toContainText("4/5 clear");
  await expect(publishing).toContainText("Director approval");
  await expect(publishing).toContainText("Publish timing");
  await expect(publishing).toContainText("Family forms");
  await expect(publishing.locator("[data-pajaritos-environments='true']")).toContainText("Publishing environments");
  await expect(publishing.locator("[data-pajaritos-environment='staging']")).toContainText("https://pajaritos.m31labs.dev");
  await expect(publishing.locator("[data-pajaritos-environment='staging']")).toHaveAttribute("data-pajaritos-environment-state", "ready");
  await expect(publishing.locator("[data-pajaritos-environment='production']")).toContainText("TBD");
  await expect(publishing.locator("[data-pajaritos-environment='production']")).toHaveAttribute("data-pajaritos-environment-state", "tbd");
  await expectPanelButtonReceivesPointer(page, ".studio-publish-controls");
}

export async function expectButtonReceivesPointer(page: Page, selector: string, label: string) {
  const button = page.locator(selector).first();
  await expect(button).toBeVisible();
  await button.scrollIntoViewIfNeeded();
  await page.waitForTimeout(50);

  const hit = await page.evaluate((selector) => {
    const button = document.querySelector(selector);
    if (!(button instanceof HTMLElement)) {
      return { ok: false, hitTag: "", hitClass: "", hitText: "" };
    }
    const rect = button.getBoundingClientRect();
    const target = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
    return {
      ok: target === button || button.contains(target),
      hitTag: target?.tagName ?? "",
      hitClass: target instanceof HTMLElement ? String(target.className) : "",
      hitText: (target?.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 120),
    };
  }, selector);

  expect(hit, `expected ${label} button to receive pointer events`).toMatchObject({ ok: true });
}

export async function revealModeIfPresent(page: Page, mode: string | undefined) {
  if (!mode) return;
  await page.waitForFunction(() =>
    document.documentElement.getAttribute("data-gosx-studio-workbench-runtime-auto-mounted") === "true" ||
    "GoSXStudioWorkbenchRuntime" in window,
  null, { timeout: 5_000 }).catch(() => null);
  await page.evaluate((mode) => {
    const form = document.querySelector("[data-studio-workbench]");
    const runtime = (window as unknown as { GoSXStudioWorkbenchRuntime?: { setMode?: (form: Element | null, mode: string, scroll: boolean) => void } }).GoSXStudioWorkbenchRuntime;
    runtime?.setMode?.(form, mode, false);
  }, mode).catch(() => null);
  const button = page.locator(`[data-studio-mode-control="${mode}"]`).first();
  if (await button.count() === 0) return;
  if (!await button.isVisible()) return;
  await button.click();
  await page.waitForTimeout(150);
}

export async function waitForStudioPreviewRefresh(page: Page, reason: string) {
  return page.evaluate((expectedReason) => new Promise<{ reason: string; route: string } | null>((resolve) => {
    let handler: EventListener;
    const timeout = window.setTimeout(() => {
      document.removeEventListener("gosxstudio:preview-refresh", handler);
      resolve(null);
    }, 5_000);
    handler = (event: Event) => {
      const detail = ((event as CustomEvent).detail ?? {}) as { reason?: string; route?: string };
      if (expectedReason && detail.reason !== expectedReason) return;
      window.clearTimeout(timeout);
      document.removeEventListener("gosxstudio:preview-refresh", handler);
      resolve({
        reason: detail.reason ?? "",
        route: detail.route ?? "",
      });
    };
    document.addEventListener("gosxstudio:preview-refresh", handler);
  }), reason);
}

export async function waitForStudioAuthoringResult(page: Page) {
  return page.evaluate(() => new Promise<StudioAuthoringResultDetail | null>((resolve) => {
    let handler: EventListener;
    const timeout = window.setTimeout(() => {
      document.removeEventListener("gosxstudio:authoring-result", handler);
      resolve(null);
    }, 5_000);
    handler = (event: Event) => {
      window.clearTimeout(timeout);
      document.removeEventListener("gosxstudio:authoring-result", handler);
      resolve(((event as CustomEvent).detail ?? null) as StudioAuthoringResultDetail | null);
    };
    document.addEventListener("gosxstudio:authoring-result", handler);
  }));
}

export async function saveEditableControl(page: Page, value: string, options?: ClickAuthoringOptions & { expectedMessage?: string }) {
  const panel = page.locator("[data-gosx-studio-editable-control='true']").first();
  await expect(panel).toBeVisible();
  const input = panel.locator("input[name='gosx_studio_value']");
  await expect(input).toBeVisible();
  await input.fill(value);
  await expectPanelButtonReceivesPointer(page, "[data-gosx-studio-editable-control='true']");
  const authoringResultPromise = options?.reloadAfter === false ? waitForStudioAuthoringResult(page) : null;
  const clickOptions = options?.reloadAfter === false ? { ...options, settleAfter: false } : options;
  const response = await clickAuthoringPanel(page, "[data-gosx-studio-editable-control='true']", clickOptions);
  if (options?.reloadAfter === false) {
    expect([200, 303]).toContain(response.status());
  } else {
    expect(response.status()).toBe(303);
    expect(authoringParam(response, "gosx_studio_operation")).toBe("save-control");
    expect(authoringParam(response, "gosx_studio_value")).toBe(value);
  }
  if (options?.reloadAfter === false) {
    const detail = await authoringResultPromise;
    expect(detail, "real editor should emit host-backed authoring result detail").toBeTruthy();
    expect(detail?.result?.message).toBe(options.expectedMessage);
    expect(detail?.change?.kind).toBe("control");
    expect(detail?.selectedCount ?? 0, "authoring result should select the changed object").toBeGreaterThan(0);
    expect(detail?.previewCount ?? 0, "authoring result should refresh at least one preview frame").toBeGreaterThan(0);
    const workbench = page.locator("[data-gosx-studio-workbench], [data-studio-workbench]").first();
    await expect(workbench).toHaveAttribute("data-gosx-studio-authoring-change-kind", "control");
    await expect(workbench).toHaveAttribute("data-gosx-studio-authoring-selected-count", /^[1-9]/);
    await expect(workbench).toHaveAttribute("data-gosx-studio-authoring-preview-url", /./);
    await expect(page.locator("[data-gosx-studio-save-detail]").first()).toHaveText(options.expectedMessage ?? /./);
    await expect(page.locator("[data-gosx-studio-editable-control='true']").first()).toBeVisible();
    await expect(page.locator("[data-gosx-studio-editable-control='true'] input[name='gosx_studio_value']").first()).toHaveValue(value);
    return;
  }
  await expect(page.locator("[data-gosx-studio-editable-control='true'] input[name='gosx_studio_value']").first()).toHaveValue(value);
}

export async function savePageMetadata(page: Page, title: string, route: string, options?: ClickAuthoringOptions & { expectedMessage?: string }) {
  const panelSelector = "[data-gosx-studio-page-metadata='true'], [data-studio-site-map-page-edit='true']";
  const buttonSelector = "[data-gosx-studio-page-metadata='true'] button, [data-studio-site-map-page-edit='true'] button";
  const panel = page.locator(panelSelector).first();
  await expect(panel).toBeVisible();
  const titleInput = panel.locator("input[name='gosx_studio_page_label']");
  const routeInput = panel.locator("input[name='gosx_studio_page_route']");
  await expect(titleInput).toBeVisible();
  await expect(routeInput).toBeVisible();
  await titleInput.fill(title);
  await routeInput.fill(route);
  await expectButtonReceivesPointer(page, buttonSelector, "page metadata");
  const authoringResultPromise = options?.reloadAfter === false ? waitForStudioAuthoringResult(page) : null;
  const clickOptions = options?.reloadAfter === false ? { ...options, settleAfter: false } : options;
  const response = await clickEditorActionButton(page, buttonSelector, "/__actions/authoring", clickOptions);
  if (options?.reloadAfter === false) {
    expect(response.status()).toBe(200);
    const detail = await authoringResultPromise;
    expect(detail, "real editor should emit page metadata authoring result detail").toBeTruthy();
    expect(detail?.result?.message).toBe(options.expectedMessage);
    expect(detail?.change?.kind).toBe("page");
    expect(detail?.selectedCount ?? 0, "page metadata save should select the changed page").toBeGreaterThan(0);
    expect(detail?.previewCount ?? 0, "page metadata save should refresh at least one preview frame").toBeGreaterThan(0);
    const workbench = page.locator("[data-gosx-studio-workbench], [data-studio-workbench]").first();
    await expect(workbench).toHaveAttribute("data-gosx-studio-authoring-change-kind", "page");
    await expect(workbench).toHaveAttribute("data-gosx-studio-authoring-selected-count", /^[1-9]/);
    await expect(page.locator("[data-gosx-studio-save-detail]").first()).toHaveText(options.expectedMessage ?? /./);
    await expect(panel).toBeVisible();
    await expect(titleInput).toHaveValue(title);
    await expect(routeInput).toHaveValue(route);
    return;
  }
  expect(response.status()).toBe(303);
  expect(authoringParam(response, "gosx_studio_operation")).toBe("update-page");
  expect(authoringParam(response, "gosx_studio_page_label")).toBe(title);
  expect(authoringParam(response, "gosx_studio_page_route")).toBe(route);
}

export function authoringParam(response: { request(): { postData(): string | null } }, key: string) {
  return new URLSearchParams(response.request().postData() ?? "").get(key);
}

export async function startMuddy(request: APIRequestContext): Promise<ServerHandle> {
  const port = await freePort();
  const tempDir = mkdtempSync(path.join(tmpdir(), "gosx-studio-muddy-e2e-"));
  mkdirSync(path.join(tempDir, "media"), { recursive: true });
  return startGoServer(request, {
    cwd: muddyRepo,
    command: "./cmd/muddy-noni",
    baseURL: `http://127.0.0.1:${port}`,
    env: {
      PORT: String(port),
      PUBLIC_SITE_URL: `http://127.0.0.1:${port}`,
      MUDDY_DATA_PATH: path.join(tempDir, "cms.json"),
      MUDDY_FLOW_DATA_PATH: path.join(tempDir, "flows.json"),
      MUDDY_LIFECYCLE_DB_PATH: path.join(tempDir, "lifecycle.db"),
      MUDDY_MEDIA_PATH: path.join(tempDir, "media"),
      MUDDY_MOCK_AUTH: "1",
      MUDDY_MOCK_CHECKOUT: "1",
    },
    tempDir,
  });
}

export async function startPajaritos(request: APIRequestContext): Promise<ServerHandle> {
  const port = await freePort();
  const tempDir = mkdtempSync(path.join(tmpdir(), "gosx-studio-pajaritos-e2e-"));
  return startGoServer(request, {
    cwd: pajaritosRepo,
    command: "./cmd/pajaritos",
    baseURL: `http://127.0.0.1:${port}`,
    env: {
      PORT: String(port),
      PAJARITOS_DATA_DIR: tempDir,
      PAJARITOS_LIFECYCLE_DB_PATH: path.join(tempDir, "lifecycle.db"),
      PAJARITOS_FLOW_STORE_PATH: path.join(tempDir, "flows.json"),
      PAJARITOS_LIFECYCLE_WORKER: "0",
    },
    tempDir,
  });
}

type ServerOptions = {
  cwd: string;
  command: string;
  baseURL: string;
  env: Record<string, string>;
  tempDir: string;
};

async function startGoServer(request: APIRequestContext, options: ServerOptions): Promise<ServerHandle> {
  const proc = spawn("go", ["run", options.command], {
    cwd: options.cwd,
    env: { ...process.env, ...options.env },
    stdio: ["ignore", "pipe", "pipe"],
  });
  const logs: string[] = [];
  proc.stdout?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));
  proc.stderr?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));

  try {
    await waitForEditor(request, options.baseURL);
  } catch (error) {
    await stopProcess(proc);
    cleanupTempDir(options.tempDir);
    throw new Error(`${String(error)}\n\nServer logs:\n${logs.join("")}`);
  }

  return {
    baseURL: options.baseURL,
    stop: async () => {
      await stopProcess(proc);
      cleanupTempDir(options.tempDir);
    },
  };
}

async function waitForEditor(request: APIRequestContext, baseURL: string) {
  const deadline = Date.now() + 60_000;
  let lastError = "";
  while (Date.now() < deadline) {
    try {
      const response = await request.get(`${baseURL}/admin/editor`, {
        failOnStatusCode: false,
        timeout: 1500,
      });
      if (response.status() >= 200 && response.status() < 500) {
        return;
      }
      lastError = `status ${response.status()}`;
    } catch (error) {
      lastError = String(error);
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error(`timed out waiting for ${baseURL}/admin/editor (${lastError})`);
}

async function freePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = createServer();
    server.on("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close(() => reject(new Error("could not allocate TCP port")));
        return;
      }
      const port = address.port;
      server.close(() => resolve(port));
    });
  });
}

async function stopProcess(proc: ChildProcess) {
  if (proc.exitCode !== null || proc.signalCode !== null) return;
  const exited = new Promise<void>((resolve) => proc.once("exit", () => resolve()));
  proc.kill("SIGINT");
  const timeout = new Promise<void>((resolve) => {
    setTimeout(() => {
      if (proc.exitCode === null && proc.signalCode === null) proc.kill("SIGKILL");
      resolve();
    }, 5_000);
  });
  await Promise.race([exited, timeout]);
}

function cleanupTempDir(tempDir: string) {
  rmSync(tempDir, { force: true, recursive: true });
}
