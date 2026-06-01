import { expect, test, type APIRequestContext, type Page, type Response, type TestInfo } from "@playwright/test";
import {
  expectPajaritosPublishingReadiness,
  revealModeIfPresent,
  startMuddy,
  startPajaritos,
  type ServerHandle,
} from "./reference_apps_harness";

type ResourceMetric = {
  url: string;
  resourceType: string;
  status: number;
  bytes: number;
};

type PerformanceBudgets = {
  shellReadyMs: number;
  previewReadyMs: number;
  publishReadyMs: number;
  editorRuntimePayloadBytes: number;
  largestRuntimeResourceBytes: number;
  previewFrameCount: number;
  canvasNodeCount: number;
  canvasOverlayCount: number;
  canvasLayoutSampleMs: number;
  publishCheckCount: number;
};

type ReferenceAppPerformanceGate = {
  name: string;
  start: (request: APIRequestContext) => Promise<ServerHandle>;
  canvasSelector: string;
  publishSelector: string;
  canvasMode?: string;
  previewMode?: string;
  publishMode?: string;
  assertPublishing?: (page: Page) => Promise<void>;
  budgets: PerformanceBudgets;
};

const defaultBudgets: PerformanceBudgets = {
  shellReadyMs: 3_500,
  previewReadyMs: 4_000,
  publishReadyMs: 1_500,
  editorRuntimePayloadBytes: 5_000_000,
  largestRuntimeResourceBytes: 2_500_000,
  previewFrameCount: 2,
  canvasNodeCount: 450,
  canvasOverlayCount: 220,
  canvasLayoutSampleMs: 150,
  publishCheckCount: 80,
};

const referenceApps: ReferenceAppPerformanceGate[] = [
  {
    name: "Muddy/Noni",
    start: startMuddy,
    canvasSelector: "[data-studio-site-map-workspace='true'], [data-studio-site-map-canvas='true']",
    publishSelector: "[data-studio-publish-panel='true']",
    canvasMode: "home",
    previewMode: "preview",
    publishMode: "publish",
    budgets: {
      ...defaultBudgets,
      editorRuntimePayloadBytes: 6_000_000,
      largestRuntimeResourceBytes: 5_000_000,
      canvasNodeCount: 260,
      publishCheckCount: 60,
    },
  },
  {
    name: "Pajaritos",
    start: startPajaritos,
    canvasSelector: "[data-gosx-studio-site-canvas='true']",
    publishSelector: "#publishing",
    assertPublishing: expectPajaritosPublishingReadiness,
    budgets: {
      ...defaultBudgets,
      shellReadyMs: 2_500,
      previewReadyMs: 3_000,
      editorRuntimePayloadBytes: 3_500_000,
      canvasNodeCount: 120,
      publishCheckCount: 60,
    },
  },
];

test.describe("@reference-apps performance budgets", () => {
  test.describe.configure({ timeout: 120_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  for (const app of referenceApps) {
    test(`${app.name} editor stays inside runtime, preview, canvas, and publish budgets`, async ({ page, request }, testInfo) => {
      const server = await app.start(request);
      try {
        const metrics = await measurePerformanceBudgets(page, server.baseURL, app);
        await attachMetrics(testInfo, app.name, metrics);
        expectBudgets(app, metrics);
      } finally {
        await server.stop();
      }
    });
  }
});

type PerformanceMetrics = {
  shellReadyMs: number;
  previewReadyMs: number;
  publishReadyMs: number;
  editorRuntimePayloadBytes: number;
  largestRuntimeResourceBytes: number;
  runtimeResourceCount: number;
  previewFrameCount: number;
  canvasNodeCount: number;
  canvasOverlayCount: number;
  canvasLayoutSampleMs: number;
  publishCheckCount: number;
  navigation: {
    domContentLoadedMs: number;
    loadEventMs: number;
    responseEndMs: number;
  };
  largestResources: Array<{
    url: string;
    resourceType: string;
    bytes: number;
  }>;
};

async function measurePerformanceBudgets(page: Page, baseURL: string, app: ReferenceAppPerformanceGate): Promise<PerformanceMetrics> {
  await page.setViewportSize({ width: 1280, height: 820 });
  const resourcePromises: Promise<ResourceMetric | null>[] = [];
  page.on("response", (response) => {
    if (shouldCaptureResource(response, baseURL)) {
      resourcePromises.push(captureResponse(response));
    }
  });

  const shellStart = Date.now();
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded" });
  await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
  await expect(page.locator("[data-studio-toolbar='true']").first()).toBeVisible();
  const shellReadyMs = Date.now() - shellStart;
  await page.waitForLoadState("networkidle");

  await revealModeIfPresent(page, app.canvasMode);
  const canvasMetrics = await measureCanvasCost(page, app.canvasSelector);

  const previewStart = Date.now();
  await revealModeIfPresent(page, app.previewMode);
  await waitForPreviewFrame(page);
  const previewReadyMs = Date.now() - previewStart;

  const publishStart = Date.now();
  await revealModeIfPresent(page, app.publishMode);
  const publish = page.locator(app.publishSelector).first();
  await expect(publish).toBeAttached();
  await publish.scrollIntoViewIfNeeded();
  await expect(publish).toBeVisible();
  if (app.assertPublishing) {
    await app.assertPublishing(page);
  }
  const publishReadyMs = Date.now() - publishStart;
  const publishCheckCount = await countPublishChecks(page, app.publishSelector);

  await page.waitForLoadState("networkidle");
  const resources = (await Promise.all(resourcePromises)).filter((metric): metric is ResourceMetric => metric !== null);
  const runtimeResources = resources.filter((resource) => isRuntimeResource(resource));
  const editorRuntimePayloadBytes = runtimeResources.reduce((sum, resource) => sum + resource.bytes, 0);
  const largestRuntimeResourceBytes = Math.max(0, ...runtimeResources.map((resource) => resource.bytes));
  const largestResources = runtimeResources
    .slice()
    .sort((a, b) => b.bytes - a.bytes)
    .slice(0, 8)
    .map((resource) => ({
      url: pathOnly(resource.url),
      resourceType: resource.resourceType,
      bytes: resource.bytes,
    }));

  return {
    shellReadyMs,
    previewReadyMs,
    publishReadyMs,
    editorRuntimePayloadBytes,
    largestRuntimeResourceBytes,
    runtimeResourceCount: runtimeResources.length,
    previewFrameCount: await page.locator("[data-studio-preview-frame]").count(),
    canvasNodeCount: canvasMetrics.nodeCount,
    canvasOverlayCount: canvasMetrics.overlayCount,
    canvasLayoutSampleMs: canvasMetrics.layoutSampleMs,
    publishCheckCount,
    navigation: await navigationMetrics(page),
    largestResources,
  };
}

function shouldCaptureResource(response: Response, baseURL: string) {
  if (!response.url().startsWith(baseURL)) return false;
  const type = response.request().resourceType();
  if (["script", "stylesheet", "document"].includes(type)) return true;
  return /\.(?:js|css|wasm)(?:\?|$)/.test(response.url()) || response.url().includes("/_gosx/");
}

async function captureResponse(response: Response): Promise<ResourceMetric | null> {
  const status = response.status();
  if (status >= 300 && status < 400) return null;
  const headers = response.headers();
  let bytes = Number(headers["content-length"] ?? 0);
  if (!Number.isFinite(bytes) || bytes < 0) bytes = 0;
  if (bytes === 0) {
    try {
      bytes = (await response.body()).length;
    } catch {
      bytes = 0;
    }
  }
  return {
    url: response.url(),
    resourceType: response.request().resourceType(),
    status,
    bytes,
  };
}

function isRuntimeResource(resource: ResourceMetric) {
  if (resource.bytes <= 0) return false;
  if (resource.resourceType === "script" || resource.resourceType === "stylesheet") return true;
  return /\.(?:js|css|wasm)(?:\?|$)/.test(resource.url) || resource.url.includes("/_gosx/");
}

async function waitForPreviewFrame(page: Page) {
  const frame = page.locator("[data-studio-preview-frame]").first();
  await expect(frame).toBeAttached();
  await frame.scrollIntoViewIfNeeded();
  await expect(frame).toBeVisible();
  await page.waitForFunction(() => {
    const frame = document.querySelector("[data-studio-preview-frame]");
    if (!(frame instanceof HTMLIFrameElement)) return false;
    const body = frame.contentDocument?.body;
    return !!body && body.innerText.trim().length > 10;
  }, null, { timeout: 10_000 });
}

async function measureCanvasCost(page: Page, canvasSelector: string) {
  const canvas = page.locator(canvasSelector).first();
  await expect(canvas).toBeAttached();
  await canvas.scrollIntoViewIfNeeded();
  await expect(canvas).toBeVisible();
  return page.evaluate((selector) => {
    const root = document.querySelector(selector);
    if (!(root instanceof HTMLElement)) {
      return { nodeCount: 0, overlayCount: 0, layoutSampleMs: 0 };
    }
    const nodeSelector = [
      "[data-gosx-studio-canvas-node]",
      "[data-studio-site-map-workspace-node]",
      "[data-studio-site-map-workspace-node-card]",
    ].join(", ");
    const overlaySelector = [
      "[data-studio-preview-overlay]",
      "[data-studio-preview-drop-line]",
      "[data-studio-preview-dock]",
      "[data-studio-preview-outline-template]",
      "[data-studio-preview-field-template]",
      "[data-gosx-studio-canvas-node-position]",
    ].join(", ");
    const nodes = Array.from(root.querySelectorAll<HTMLElement>(nodeSelector));
    const overlays = Array.from(document.querySelectorAll<HTMLElement>(overlaySelector));
    const started = performance.now();
    for (const element of nodes.slice(0, 250)) {
      element.getBoundingClientRect();
    }
    for (const element of overlays.slice(0, 250)) {
      element.getBoundingClientRect();
    }
    return {
      nodeCount: nodes.length,
      overlayCount: overlays.length,
      layoutSampleMs: performance.now() - started,
    };
  }, canvasSelector);
}

async function countPublishChecks(page: Page, publishSelector: string) {
  return page.locator([
    `${publishSelector} [data-readiness-key]`,
    `${publishSelector} [data-pajaritos-environment]`,
    `${publishSelector} [data-studio-flow-readiness]`,
    `${publishSelector} .studio-publish-review__card`,
    `${publishSelector} article`,
    `${publishSelector} li`,
  ].join(", ")).count();
}

async function navigationMetrics(page: Page) {
  return page.evaluate(() => {
    const navigation = performance.getEntriesByType("navigation")[0] as PerformanceNavigationTiming | undefined;
    if (!navigation) {
      return { domContentLoadedMs: 0, loadEventMs: 0, responseEndMs: 0 };
    }
    return {
      domContentLoadedMs: navigation.domContentLoadedEventEnd - navigation.startTime,
      loadEventMs: navigation.loadEventEnd - navigation.startTime,
      responseEndMs: navigation.responseEnd - navigation.startTime,
    };
  });
}

function expectBudgets(app: ReferenceAppPerformanceGate, metrics: PerformanceMetrics) {
  const budgets = app.budgets;
  expect(metrics.shellReadyMs, `${app.name} shell ready latency`).toBeLessThanOrEqual(budgets.shellReadyMs);
  expect(metrics.previewReadyMs, `${app.name} preview iframe ready latency`).toBeLessThanOrEqual(budgets.previewReadyMs);
  expect(metrics.publishReadyMs, `${app.name} publish readiness latency`).toBeLessThanOrEqual(budgets.publishReadyMs);
  expect(metrics.runtimeResourceCount, `${app.name} should load measurable runtime resources`).toBeGreaterThan(0);
  expect(metrics.editorRuntimePayloadBytes, `${app.name} editor runtime payload`).toBeLessThanOrEqual(budgets.editorRuntimePayloadBytes);
  expect(metrics.largestRuntimeResourceBytes, `${app.name} largest runtime resource`).toBeLessThanOrEqual(budgets.largestRuntimeResourceBytes);
  expect(metrics.previewFrameCount, `${app.name} active preview frame count`).toBeLessThanOrEqual(budgets.previewFrameCount);
  expect(metrics.canvasNodeCount, `${app.name} canvas/site-map node count`).toBeGreaterThan(0);
  expect(metrics.canvasNodeCount, `${app.name} canvas/site-map node count`).toBeLessThanOrEqual(budgets.canvasNodeCount);
  expect(metrics.canvasOverlayCount, `${app.name} canvas overlay DOM count`).toBeLessThanOrEqual(budgets.canvasOverlayCount);
  expect(metrics.canvasLayoutSampleMs, `${app.name} canvas layout sample`).toBeLessThanOrEqual(budgets.canvasLayoutSampleMs);
  expect(metrics.publishCheckCount, `${app.name} publish readiness item count`).toBeGreaterThan(0);
  expect(metrics.publishCheckCount, `${app.name} publish readiness item count`).toBeLessThanOrEqual(budgets.publishCheckCount);
}

async function attachMetrics(testInfo: TestInfo, appName: string, metrics: PerformanceMetrics) {
  await testInfo.attach(`${appName.toLowerCase().replace(/[^a-z0-9]+/g, "-")}-performance-budget.json`, {
    contentType: "application/json",
    body: Buffer.from(JSON.stringify(metrics, null, 2)),
  });
  console.info(`${appName} performance budgets`, JSON.stringify({
    shellReadyMs: Math.round(metrics.shellReadyMs),
    previewReadyMs: Math.round(metrics.previewReadyMs),
    publishReadyMs: Math.round(metrics.publishReadyMs),
    editorRuntimePayloadBytes: metrics.editorRuntimePayloadBytes,
    largestRuntimeResourceBytes: metrics.largestRuntimeResourceBytes,
    runtimeResourceCount: metrics.runtimeResourceCount,
    previewFrameCount: metrics.previewFrameCount,
    canvasNodeCount: metrics.canvasNodeCount,
    canvasOverlayCount: metrics.canvasOverlayCount,
    canvasLayoutSampleMs: Number(metrics.canvasLayoutSampleMs.toFixed(2)),
    publishCheckCount: metrics.publishCheckCount,
    largestResources: metrics.largestResources,
  }));
}

function pathOnly(rawURL: string) {
  try {
    const url = new URL(rawURL);
    return `${url.pathname}${url.search}`;
  } catch {
    return rawURL;
  }
}
