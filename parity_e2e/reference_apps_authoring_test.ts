import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { createServer } from "node:net";

const workRoot = path.resolve(__dirname, "../..");
const muddyRepo = process.env.GOSX_STUDIO_MUDDY_REPO ?? path.join(workRoot, "muddy-noni-commerce");
const pajaritosRepo = process.env.GOSX_STUDIO_PAJARITOS_REPO ?? path.join(workRoot, "pajaritos-forest-school");

test.describe("@reference-apps browser authoring workflows", () => {
  test.describe.configure({ timeout: 90_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni visible authoring controls submit through the real admin editor", async ({ page, request }) => {
    const server = await startMuddy(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle" });
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await expectIntentButtonReceivesPointer(page, "create-page:landing");
      const createResponse = await clickIntent(page, "create-page:landing");
      expect(createResponse.status()).toBe(303);
      await expect(page.locator("body")).toContainText("6 pages");

      await expectIntentButtonReceivesPointer(page, "add-component:home:contact-flow");
      const addResponse = await clickIntent(page, "add-component:home:contact-flow");
      expect(addResponse.status()).toBe(303);
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await expectPanelButtonReceivesPointer(page, "[data-studio-site-map-component-duplicate='true']");
      const duplicateResponse = await clickAuthoringPanel(page, "[data-studio-site-map-component-duplicate='true']");
      expect(duplicateResponse.status()).toBe(303);
      await expect(page.locator("body")).toContainText("Hero copy 2");

      await saveEditableControl(page, "Surface-tested clay rituals");
    } finally {
      await server.stop();
    }
  });

  test("Pajaritos visible authoring controls submit through the real admin editor", async ({ page, request }) => {
    const server = await startPajaritos(request);
    try {
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle" });
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();

      await expectIntentButtonReceivesPointer(page, "create-page:program-page");
      const createResponse = await clickIntent(page, "create-page:program-page");
      expect(createResponse.status()).toBe(303);
      expect(authoringParam(createResponse, "gosx_studio_operation")).toBe("apply-intent");
      expect(authoringParam(createResponse, "gosx_studio_intent_kind")).toBe("create-page");
      await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();
      await expect(page.locator("[data-gosx-studio-page-list='true']")).toContainText("6 pages");
      await expect(page.locator("[data-gosx-studio-page-list='true']")).toContainText("Program Page");
      await expect(page.locator("body")).toContainText("Program Page created.");

      await expectPanelButtonReceivesPointer(page, "[data-gosx-studio-component-duplicate='true']");
      const duplicateResponse = await clickAuthoringPanel(page, "[data-gosx-studio-component-duplicate='true']", { reloadAfter: false });
      expect(duplicateResponse.status()).toBe(303);
      expect(authoringParam(duplicateResponse, "gosx_studio_operation")).toBe("duplicate-component");
      expect(authoringParam(duplicateResponse, "gosx_studio_component_key")).toBe("hero");
      await expect(page.locator("body")).toContainText("Hero copy 2 duplicated.");

      await saveEditableControl(page, "Small boots, big questions.", {
        reloadAfter: false,
        expectedMessage: "Hero headline saved.",
      });

      await expectIntentButtonReceivesPointer(page, "add-component:home:hero");
      const addResponse = await clickIntent(page, "add-component:home:hero");
      expect(addResponse.status()).toBe(303);
      expect(authoringParam(addResponse, "gosx_studio_operation")).toBe("apply-intent");
      expect(authoringParam(addResponse, "gosx_studio_intent_kind")).toBe("add-component");
      expect(authoringParam(addResponse, "gosx_studio_component_template_key")).toBe("hero");
      await expect(page.locator("body")).toContainText("Hero section added to Home.");
      await expect(page.locator("[data-studio-site-map-component='hero__copy_3']").first()).toBeAttached();

      const restoreButtonSelector = "[data-pajaritos-restore-index='1'] button";
      await expectPanelButtonReceivesPointer(page, "[data-pajaritos-restore-index='1']");
      const restoreButton = page.locator(restoreButtonSelector).first();
      const restoreRevisionID = await restoreButton.getAttribute("value");
      expect(await restoreButton.getAttribute("name")).toBe("revisionId");
      expect(restoreRevisionID).toBeTruthy();
      const restoreResponse = await clickEditorActionButton(page, restoreButtonSelector, "/__actions/restoreRevision", {
        noWaitAfter: true,
        reloadAfter: false,
        settleAfter: false,
      });
      expect(restoreResponse.status()).toBe(303);
      expect(restoreResponse.headers()["location"]).toContain("Restore+point+applied.");
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "networkidle" });
      await expect(page.locator("[data-studio-site-map-component='hero__copy_2']").first()).toBeAttached();
      await expect(page.locator("[data-studio-site-map-component='hero__copy_3']")).toHaveCount(0);
    } finally {
      await server.stop();
    }
  });
});

async function clickIntent(page: Page, intentKey: string) {
  return clickAuthoringButton(page, `[data-studio-composition-intent="${intentKey}"] button`);
}

type ClickAuthoringOptions = {
  noWaitAfter?: boolean;
  reloadAfter?: boolean;
  settleAfter?: boolean;
};

async function clickAuthoringPanel(page: Page, panelSelector: string, options?: ClickAuthoringOptions) {
  return clickAuthoringButton(page, `${panelSelector} button`, options);
}

async function clickAuthoringButton(page: Page, buttonSelector: string, options?: ClickAuthoringOptions) {
  return clickEditorActionButton(page, buttonSelector, "/__actions/authoring", options);
}

async function clickEditorActionButton(page: Page, buttonSelector: string, actionPathPart: string, options?: ClickAuthoringOptions) {
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

async function expectIntentButtonReceivesPointer(page: Page, intentKey: string) {
  await expectButtonReceivesPointer(page, `[data-studio-composition-intent="${intentKey}"] button`, intentKey);
}

async function expectPanelButtonReceivesPointer(page: Page, panelSelector: string) {
  await expectButtonReceivesPointer(page, `${panelSelector} button`, panelSelector);
}

async function expectButtonReceivesPointer(page: Page, selector: string, label: string) {
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

async function saveEditableControl(page: Page, value: string, options?: ClickAuthoringOptions & { expectedMessage?: string }) {
  const panel = page.locator("[data-gosx-studio-editable-control='true']").first();
  await expect(panel).toBeVisible();
  const input = panel.locator("input[name='gosx_studio_value']");
  await expect(input).toBeVisible();
  await input.fill(value);
  await expectPanelButtonReceivesPointer(page, "[data-gosx-studio-editable-control='true']");
  const response = await clickAuthoringPanel(page, "[data-gosx-studio-editable-control='true']", options);
  expect(response.status()).toBe(303);
  expect(authoringParam(response, "gosx_studio_operation")).toBe("save-control");
  expect(authoringParam(response, "gosx_studio_value")).toBe(value);
  if (options?.reloadAfter === false) {
    if (options.expectedMessage) {
      await expect(page.locator("body")).toContainText(options.expectedMessage);
    }
    await expect(page.locator("[data-gosx-studio-editable-control='true']")).toBeVisible();
    return;
  }
  await expect(page.locator("[data-gosx-studio-editable-control='true'] input[name='gosx_studio_value']").first()).toHaveValue(value);
}

function authoringParam(response: { request(): { postData(): string | null } }, key: string) {
  return new URLSearchParams(response.request().postData() ?? "").get(key);
}

async function startMuddy(request: APIRequestContext): Promise<ServerHandle> {
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

async function startPajaritos(request: APIRequestContext): Promise<ServerHandle> {
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

type ServerHandle = {
  baseURL: string;
  stop: () => Promise<void>;
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
