import { expect, test, type APIRequestContext, type Page } from "@playwright/test";
import {
  expectButtonReceivesPointer,
  expectPajaritosPublishingReadiness,
  revealModeIfPresent,
  startMuddy,
  startPajaritos,
  type ServerHandle,
} from "./reference_apps_harness";

type ReferenceAppGate = {
  name: string;
  start: (request: APIRequestContext) => Promise<ServerHandle>;
  authoringIntentKey: string;
  canvasSelector: string;
  publishSelector: string;
  canvasMode?: string;
  previewMode?: string;
  publishMode?: string;
  assertPublishing?: (page: Page) => Promise<void>;
};

const referenceApps: ReferenceAppGate[] = [
  {
    name: "Muddy/Noni",
    start: startMuddy,
    authoringIntentKey: "create-page:landing",
    canvasSelector: "[data-studio-site-map-workspace='true'], [data-studio-site-map-canvas='true']",
    publishSelector: "[data-studio-publish-panel='true']",
    // studio-pagecanvas-handoff moved the legacy site-map/canvas board (and, for
    // Pajaritos, the publishing panel) behind the "Advanced" mode panel once a
    // PageCanvas surface is present; "home" no longer reveals this content.
    canvasMode: "advanced",
    previewMode: "preview",
    publishMode: "publish",
  },
  {
    name: "Pajaritos",
    start: startPajaritos,
    authoringIntentKey: "create-page:program-page",
    canvasSelector: "#website-map",
    publishSelector: "#publishing",
    canvasMode: "advanced",
    // The preview frame lives in the page-canvas ("Home") stage, which
    // studio.css hides outright while data-studio-mode="advanced"
    // (`.editor-workbench[data-studio-mode="advanced"] .studio-page-canvas`);
    // switch back to a non-advanced mode before the preview-frame gate.
    previewMode: "home",
    publishMode: "advanced",
    assertPublishing: expectPajaritosPublishingReadiness,
  },
];

test.describe("@reference-apps visual and accessibility gates", () => {
  // 240s: the Muddy editor now ships the full-WASM CanvasBoard by default
  // (~30MB runtime) and this spec walks shell + canvas + publish + three
  // responsive breakpoints with an a11y scan at each stop — CI's 2-core
  // runners routinely need >120s for the Muddy entry while local runs
  // finish in about half that. Assertions are unchanged.
  test.describe.configure({ timeout: 240_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  for (const app of referenceApps) {
    test(`${app.name} editor shell, canvas, publish, and responsive preview stay usable`, async ({ page, request }) => {
      const server = await app.start(request);
      try {
        await runVisualA11yGate(page, server.baseURL, app);
      } finally {
        await server.stop();
      }
    });
  }
});

async function runVisualA11yGate(page: Page, baseURL: string, app: ReferenceAppGate) {
  await page.setViewportSize({ width: 1280, height: 820 });
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expectEditorShell(page, app);
  await expectCanvasGate(page, app);
  await expectPublishingGate(page, app);
  await expectPreviewGate(page, app);
  await expectEditorAccessibilityBasics(page, app);
  await expectVisualIntegrity(page, app, "desktop");

  await page.setViewportSize({ width: 390, height: 844 });
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.waitForTimeout(150);
  await expectEditorShell(page, app);
  await expectCanvasGate(page, app);
  await expectPreviewGate(page, app);
  await expectEditorAccessibilityBasics(page, app);
  await expectVisualIntegrity(page, app, "mobile");
}

async function expectEditorShell(page: Page, app: ReferenceAppGate) {
  await expect(page.locator("[data-gosx-studio-workbench='true']").first()).toBeAttached();
  await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
  // The composition-intent forms below are part of the legacy site-map engine,
  // which now lives inside the "Advanced" mode panel (studio-pagecanvas-handoff
  // moved it there once a PageCanvas surface is present); reveal it before the
  // pointer-receives assertion, which requires the button to be visible.
  await revealModeIfPresent(page, "advanced");
  await expect(page.locator("[data-studio-toolbar='true']").first()).toBeVisible();
  await expect(page.locator("[data-studio-composition-intent-forms='true']").first()).toBeAttached();
  await expect(page.locator("h1, [data-studio-toolbar='true'] strong").first()).toBeVisible();
  await expectButtonReceivesPointer(
    page,
    `[data-studio-composition-intent="${app.authoringIntentKey}"] button`,
    `${app.name} create-page intent`,
  );
}

async function expectCanvasGate(page: Page, app: ReferenceAppGate) {
  await revealModeIfPresent(page, app.canvasMode);
  const canvas = page.locator(app.canvasSelector).first();
  await expect(canvas).toBeAttached();
  await canvas.scrollIntoViewIfNeeded();
  await expect(canvas).toBeVisible();
  await expectSurfaceWithinViewport(page, app.canvasSelector, `${app.name} canvas`);

  const nodeCount = await page.locator([
    `${app.canvasSelector} [data-gosx-studio-canvas-node]`,
    `${app.canvasSelector} [data-studio-site-map-workspace-node]`,
    `${app.canvasSelector} [data-studio-site-map-workspace-node-card]`,
  ].join(", ")).count();
  expect(nodeCount, `${app.name} should expose visible canvas/site-map nodes`).toBeGreaterThan(0);
}

async function expectPublishingGate(page: Page, app: ReferenceAppGate) {
  await revealModeIfPresent(page, app.publishMode);
  const publishing = page.locator(app.publishSelector).first();
  await expect(publishing).toBeAttached();
  await publishing.scrollIntoViewIfNeeded();
  await expect(publishing).toBeVisible();
  await expectSurfaceWithinViewport(page, app.publishSelector, `${app.name} publishing`);
  if (app.assertPublishing) {
    await app.assertPublishing(page);
  }
}

async function expectPreviewGate(page: Page, app: ReferenceAppGate) {
  await revealModeIfPresent(page, app.previewMode);
  const frame = page.locator("[data-studio-preview-frame]").first();
  await expect(frame).toBeAttached();
  await hydrateDeferredPreviewFrame(frame);
  await frame.scrollIntoViewIfNeeded();
  await expect(frame).toBeVisible();
  await expect(frame).toHaveAttribute("src", /.+/);
  await expectSurfaceWithinViewport(page, "[data-studio-preview-frame]", `${app.name} responsive preview`);

  await page.waitForFunction(() => {
    const frame = document.querySelector("[data-studio-preview-frame]");
    if (!(frame instanceof HTMLIFrameElement)) return false;
    const body = frame.contentDocument?.body;
    return !!body && body.innerText.trim().length > 10;
  }, null, { timeout: 10_000 });

  const box = await frame.boundingBox();
  expect(box, `${app.name} preview frame should have a visible box`).toBeTruthy();
  expect(box?.width ?? 0).toBeGreaterThan(240);
  expect(box?.height ?? 0).toBeGreaterThan(120);
  const screenshot = await frame.screenshot();
  expect(screenshot.length, `${app.name} preview frame should render non-empty pixels`).toBeGreaterThan(1_000);
}

async function hydrateDeferredPreviewFrame(frame: ReturnType<Page["locator"]>) {
  await frame.evaluate((node) => {
    if (!(node instanceof HTMLIFrameElement)) return;
    if (node.getAttribute("src")) return;
    const container = node.closest("[data-gosx-studio-preview]");
    const nextURL = node.getAttribute("data-studio-preview-src") ||
      container?.getAttribute("data-gosx-studio-preview-url") ||
      "/";
    node.setAttribute("src", nextURL);
  });
}

async function expectEditorAccessibilityBasics(page: Page, app: ReferenceAppGate) {
  const missingNames = await page.evaluate(() => {
    const issues: string[] = [];
    const visible = (element: Element) => {
      if (!(element instanceof HTMLElement)) return false;
      if (element.closest("[hidden], [aria-hidden='true']")) return false;
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return style.display !== "none" &&
        style.visibility !== "hidden" &&
        Number(style.opacity) > 0.01 &&
        rect.width > 1 &&
        rect.height > 1;
    };
    const textFor = (element: HTMLElement): string => {
      const labelledBy = element.getAttribute("aria-labelledby");
      const labelledByText = labelledBy
        ?.split(/\s+/)
        .map((id) => document.getElementById(id)?.textContent?.trim() ?? "")
        .filter(Boolean)
        .join(" ");
      return [
        element.getAttribute("aria-label"),
        labelledByText,
        element.getAttribute("title"),
        element instanceof HTMLInputElement ? element.value : "",
        element.textContent,
      ].join(" ").trim();
    };
    const describe = (element: HTMLElement) => {
      const selectorHint = element.getAttribute("data-studio-command-open") ||
        element.getAttribute("data-studio-mode-control") ||
        element.getAttribute("data-studio-composition-intent") ||
        element.getAttribute("class") ||
        element.tagName.toLowerCase();
      return `${element.tagName.toLowerCase()} ${String(selectorHint).replace(/\s+/g, ".")}`.trim();
    };

    document.querySelectorAll("button, [role='button'], a[href]").forEach((element) => {
      if (!(element instanceof HTMLElement) || !visible(element)) return;
      if (!textFor(element)) issues.push(`missing accessible name: ${describe(element)}`);
    });

    document.querySelectorAll("input:not([type='hidden']), textarea, select").forEach((element) => {
      if (!(element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement) || !visible(element)) return;
      const labels = "labels" in element && element.labels ? Array.from(element.labels) : [];
      const hasName = !!element.getAttribute("aria-label") ||
        !!element.getAttribute("aria-labelledby") ||
        labels.some((label) => label.textContent?.trim()) ||
        !!element.closest("label")?.textContent?.trim();
      if (!hasName) issues.push(`missing form label: ${describe(element)}`);
    });
    return issues.slice(0, 20);
  });
  expect(missingNames, `${app.name} visible controls should have accessible names`).toEqual([]);

  await page.keyboard.press("Tab");
  await page.waitForTimeout(50);
  const focus = await page.evaluate(() => {
    const element = document.activeElement;
    if (!(element instanceof HTMLElement)) return { ok: false, tag: "" };
    const rect = element.getBoundingClientRect();
    return {
      ok: ["A", "BUTTON", "INPUT", "SELECT", "TEXTAREA"].includes(element.tagName) &&
        rect.width > 0 &&
        rect.height > 0,
      tag: element.tagName,
      text: (element.textContent ?? element.getAttribute("aria-label") ?? "").trim().slice(0, 80),
    };
  });
  expect(focus, `${app.name} should expose keyboard focus to visible controls`).toMatchObject({ ok: true });

  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.waitForTimeout(100);
  const runningAnimations = await page.evaluate(() =>
    document.getAnimations()
      .filter((animation) => {
        const timing = animation.effect?.getComputedTiming();
        return animation.playState === "running" && Number(timing?.duration ?? 0) > 200;
      })
      .map((animation) => String((animation.effect as KeyframeEffect | null)?.target ?? animation.playState))
      .slice(0, 10),
  );
  expect(runningAnimations, `${app.name} should not run long animations under reduced motion`).toEqual([]);
  await page.emulateMedia({ reducedMotion: "no-preference" });
}

async function expectVisualIntegrity(page: Page, app: ReferenceAppGate, viewport: "desktop" | "mobile") {
  // The composition forms + canvas live inside the "Advanced" mode panel,
  // while the preview frame lives in the page-canvas ("Home") stage, which
  // studio.css hides outright while data-studio-mode="advanced". The two are
  // mutually exclusive, so reveal each mode immediately before the checks
  // that depend on it (mirrors expectCanvasGate / expectPreviewGate above).
  await revealModeIfPresent(page, app.canvasMode);
  const advancedChecks = [
    { label: "toolbar", selector: "[data-studio-toolbar='true']" },
    { label: "composition forms", selector: "[data-studio-composition-intent-forms='true']" },
    { label: "canvas", selector: app.canvasSelector },
  ];
  for (const check of advancedChecks) {
    await expectSurfaceWithinViewport(page, check.selector, `${app.name} ${viewport} ${check.label}`);
  }
  await revealModeIfPresent(page, app.previewMode);
  await expectSurfaceWithinViewport(page, "[data-studio-preview-frame]", `${app.name} ${viewport} preview`);

  const clippedText = await page.evaluate(() => {
    const issues: string[] = [];
    const candidates = Array.from(document.querySelectorAll<HTMLElement>(
      "button, [role='button'], a.button, .button",
    ));
    for (const element of candidates) {
      if (issues.length >= 20) break;
      if (element.closest("[hidden], [aria-hidden='true']")) continue;
      const rect = element.getBoundingClientRect();
      const style = window.getComputedStyle(element);
      if (style.display === "none" || style.display === "inline" || style.visibility === "hidden" || rect.width <= 1 || rect.height <= 1) continue;
      if (element.scrollWidth > Math.ceil(element.clientWidth) + 3 || element.scrollHeight > Math.ceil(element.clientHeight) + 3) {
        issues.push(`${element.tagName.toLowerCase()} "${(element.textContent || element.getAttribute("aria-label") || element.getAttribute("name") || "").trim().replace(/\s+/g, " ").slice(0, 80)}"`);
      }
    }
    return issues;
  });
  expect(clippedText, `${app.name} ${viewport} controls should not clip visible text`).toEqual([]);

  const contrastIssues = await page.evaluate(() => {
    type RGB = { r: number; g: number; b: number; a: number };
    const parseRGB = (value: string): RGB | null => {
      const match = value.match(/rgba?\(([^)]+)\)/);
      if (!match) return null;
      const parts = match[1].split(",").map((part) => Number(part.trim()));
      if (parts.length < 3) return null;
      return { r: parts[0], g: parts[1], b: parts[2], a: parts[3] ?? 1 };
    };
    const luminance = (rgb: RGB) => {
      const channel = (value: number) => {
        const normalized = value / 255;
        return normalized <= 0.03928 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
      };
      return 0.2126 * channel(rgb.r) + 0.7152 * channel(rgb.g) + 0.0722 * channel(rgb.b);
    };
    const ratio = (a: RGB, b: RGB) => {
      const light = Math.max(luminance(a), luminance(b));
      const dark = Math.min(luminance(a), luminance(b));
      return (light + 0.05) / (dark + 0.05);
    };
    const visible = (element: HTMLElement) => {
      if (element.closest("[hidden], [aria-hidden='true']")) return false;
      const style = window.getComputedStyle(element);
      const rect = element.getBoundingClientRect();
      return style.display !== "none" && style.visibility !== "hidden" && Number(style.opacity) > 0.5 && rect.width > 1 && rect.height > 1;
    };
    const backgroundFor = (element: HTMLElement): RGB => {
      let current: HTMLElement | null = element;
      while (current) {
        const rgb = parseRGB(window.getComputedStyle(current).backgroundColor);
        if (rgb && rgb.a > 0.2) return rgb;
        current = current.parentElement;
      }
      return { r: 255, g: 255, b: 255, a: 1 };
    };
    const issues: string[] = [];
    const root = document.querySelector("[data-studio-workbench='true']") ?? document.body;
    const candidates = Array.from(root.querySelectorAll<HTMLElement>("button, a, label, strong, small, p, span, output, h1, h2, h3"));
    for (const element of candidates) {
      if (issues.length >= 15) break;
      if (!visible(element)) continue;
      const text = (element.textContent ?? "").trim().replace(/\s+/g, " ");
      if (text.length < 2) continue;
      const color = parseRGB(window.getComputedStyle(element).color);
      if (!color || color.a < 0.5) continue;
      const value = ratio(color, backgroundFor(element));
      if (value < 3) issues.push(`${element.tagName.toLowerCase()} "${text.slice(0, 70)}" ratio ${value.toFixed(2)}`);
    }
    return issues;
  });
  expect(contrastIssues, `${app.name} ${viewport} sampled text should keep usable contrast`).toEqual([]);
}

async function expectSurfaceWithinViewport(page: Page, selector: string, label: string) {
  const issue = await page.evaluate((selector) => {
    const element = document.querySelector(selector);
    if (!(element instanceof HTMLElement)) return `missing ${selector}`;
    if (element.closest("[hidden], [aria-hidden='true']")) return null;
    const style = window.getComputedStyle(element);
    if (style.display === "none" || style.visibility === "hidden") return null;
    const rect = element.getBoundingClientRect();
    const tolerance = 8;
    if (rect.width <= 1 || rect.height <= 1) return `empty rect ${Math.round(rect.width)}x${Math.round(rect.height)}`;
    if (rect.left < -tolerance) return `left overflow ${Math.round(rect.left)}`;
    if (rect.right > window.innerWidth + tolerance) return `right overflow ${Math.round(rect.right)} > ${window.innerWidth}`;
    return null;
  }, selector);
  expect(issue, `${label} should fit the viewport`).toBeNull();
}
