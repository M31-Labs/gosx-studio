import { expect, type APIRequestContext, type Locator, type Page } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdtempSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { createServer } from "node:net";

const workRoot = path.resolve(__dirname, "../..");
const muddyRepo = process.env.GOSX_STUDIO_MUDDY_REPO ?? path.join(workRoot, "muddy-noni-commerce");
const pajaritosRepo = process.env.GOSX_STUDIO_PAJARITOS_REPO ?? path.join(workRoot, "pajaritos-forest-school");
const defaultGoBin = "/home/draco/go/bin";

let muddyDistBuildPromise: Promise<void> | null = null;

export type ClickAuthoringOptions = {
  noWaitAfter?: boolean;
  reloadAfter?: boolean;
  settleAfter?: boolean;
};

export type ServerHandle = {
  baseURL: string;
  dataDir?: string;
  stop: () => Promise<void>;
};

export type StudioAuthoringResultDetail = {
  action?: string;
  method?: string;
  result?: {
    message?: string;
    previewURL?: string;
    refreshPreview?: boolean;
    fragments?: Array<{ selector?: string; mode?: string }>;
    fragmentCount?: number;
    values?: Record<string, string>;
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
  fragmentCount?: number;
};

export type StudioFragmentRefreshDetail = {
  url?: string;
  count?: number;
  selectors?: string[];
};

export async function clickIntent(page: Page, intentKey: string, options?: ClickAuthoringOptions) {
  return clickAuthoringButton(page, `[data-studio-composition-intent="${intentKey}"] button`, options);
}

export async function clickAuthoringPanel(page: Page, panelSelector: string, options?: ClickAuthoringOptions) {
  return clickAuthoringButton(page, `${panelSelector} button`, options);
}

export async function clickAuthoringButton(page: Page, buttonSelector: string, options?: ClickAuthoringOptions) {
  return clickEditorActionButton(page, buttonSelector, "/__actions/authoring", options);
}

export async function clickEditorActionButton(page: Page, buttonSelector: string, actionPathPart: string, options?: ClickAuthoringOptions) {
  const shouldWaitForNavigation = options?.reloadAfter !== false;
  const navigationPromise = shouldWaitForNavigation ? page.waitForEvent("framenavigated", {
    predicate: (frame) => frame === page.mainFrame(),
    timeout: 3_000,
  }).catch(() => null) : Promise.resolve(null);
  const responsePromise = page.waitForResponse((response) =>
    response.url().includes(actionPathPart) &&
    response.request().method() === "POST",
  );
  await page.locator(buttonSelector).first().click({ noWaitAfter: options?.noWaitAfter === true || !shouldWaitForNavigation });
  const response = await responsePromise;
  await navigationPromise;
  if (options?.settleAfter !== false) {
    // Best-effort settle: interaction-heavy pages (15 canvas targets) keep
    // background requests trickling and never reach networkidle on slow CI
    // runners — tests must rely on their own element/response assertions,
    // not this wait, so a missed idle is not an error.
    await page.waitForLoadState("networkidle", { timeout: 15_000 }).catch(() => {});
  }
  if (options?.reloadAfter !== false) {
    await page.goto(page.url(), { waitUntil: "domcontentloaded" });
    await page.waitForLoadState("networkidle", { timeout: 10_000 }).catch(() => {});
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

export async function gotoEditor(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("[data-studio-workbench='true']").first()).toBeAttached();
  await expect(page.locator("[data-studio-composition-intent-forms='true']")).toBeAttached();
  await page.waitForFunction(() =>
    (window as unknown as { __gosx_studio_authoring_runtime_bound?: string }).__gosx_studio_authoring_runtime_bound === "true",
  null, { timeout: 20_000 });
}

export async function expectButtonReceivesPointer(page: Page, selector: string, label: string) {
  const button = page.locator(selector).first();
  await expect(button).toBeVisible();
  await button.evaluate((el) => {
    const scrollNodes: HTMLElement[] = [];
    let scrollNode: HTMLElement | null = el.parentElement;
    while (scrollNode) {
      scrollNodes.push(scrollNode);
      scrollNode = scrollNode.parentElement;
    }
    if (document.documentElement instanceof HTMLElement) scrollNodes.push(document.documentElement);
    if (document.body instanceof HTMLElement) scrollNodes.push(document.body);
    const previousScrollBehavior = scrollNodes.map((node) => [node, node.style.scrollBehavior] as const);
    for (const node of scrollNodes) {
      node.style.scrollBehavior = "auto";
    }

    let current: HTMLElement | null = el.parentElement;
    try {
      while (current) {
        if (current.scrollHeight > current.clientHeight || current.scrollWidth > current.clientWidth) {
          const currentRect = current.getBoundingClientRect();
          const targetRect = el.getBoundingClientRect();
          current.scrollTop += targetRect.top - currentRect.top - current.clientHeight / 2 + targetRect.height / 2;
          current.scrollLeft += targetRect.left - currentRect.left - current.clientWidth / 2 + targetRect.width / 2;
        }
        current = current.parentElement;
      }
      el.scrollIntoView({ block: "center", inline: "center" });
      const rect = el.getBoundingClientRect();
      const targetTop = rect.top + rect.height / 2 - window.innerHeight / 2;
      const targetLeft = rect.left + rect.width / 2 - window.innerWidth / 2;
      const scroller = document.scrollingElement;
      if (scroller instanceof HTMLElement && (rect.top < 0 || rect.bottom > window.innerHeight)) {
        scroller.scrollTop += targetTop;
      }
      if (scroller instanceof HTMLElement && (rect.left < 0 || rect.right > window.innerWidth)) {
        scroller.scrollLeft += targetLeft;
      }
      if (rect.top < 0 || rect.bottom > window.innerHeight) window.scrollBy({ top: targetTop, behavior: "auto" });
      if (rect.left < 0 || rect.right > window.innerWidth) window.scrollBy({ left: targetLeft, behavior: "auto" });
    } finally {
      for (const [node, value] of previousScrollBehavior) {
        node.style.scrollBehavior = value;
      }
    }
  });
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
      rect: {
        left: Math.round(rect.left),
        top: Math.round(rect.top),
        right: Math.round(rect.right),
        bottom: Math.round(rect.bottom),
      },
      viewport: { width: window.innerWidth, height: window.innerHeight },
      hitTag: target?.tagName ?? "",
      hitClass: target instanceof HTMLElement ? String(target.className) : "",
      hitText: (target?.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 120),
    };
  }, selector);

  expect(hit.ok, `expected ${label} button to receive pointer events: ${JSON.stringify(hit)}`).toBe(true);
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

// openInteractionsTargetDisclosure opens whatever native <details> disclosures
// currently stand between the DOM and one interactions-target card's own
// controls, matching the real human steps wave 3A's inspector-presentation
// pass introduced (panels/interactions_panel.go):
//   1. RenderInteractionsInspector groups every canvas target with ZERO
//      attached interactions behind one shared closed-by-default
//      `[data-studio-interactions-add]` "+ Add interaction" picker. A target
//      that hasn't had anything attached yet (the common starting state for
//      these tests) is nested inside that picker and must have it opened
//      first, or its own card never reaches the layout at all.
//   2. Every target card (attached or not) additionally wraps its own
//      Effect/Duration/Delay rows in a second, per-target closed-by-default
//      `.studio-interactions-target__editor` disclosure (RenderInteractionsPanel).
// Both checks re-read the LIVE DOM at call time (not a one-shot assumption
// about page structure), so this is safe to call again after an in-place
// authoring save/fragment-refresh moves a target between the attached and
// unattached buckets.
export async function openInteractionsTargetDisclosure(page: Page, targetSelector: string) {
  const target = page.locator(targetSelector).first();
  await expect(target, `interactions target ${targetSelector} must render`).toBeAttached();

  const nestedInClosedAddPicker = await target.evaluate((el) => {
    const picker = el.closest("[data-studio-interactions-add]");
    return picker instanceof HTMLDetailsElement && !picker.open;
  });
  if (nestedInClosedAddPicker) {
    await page.locator("[data-studio-interactions-add] > summary").first().click();
  }

  const editor = target.locator(".studio-interactions-target__editor");
  const editorClosed = await editor.evaluate((el) => el instanceof HTMLDetailsElement && !el.open);
  if (editorClosed) {
    await editor.locator("summary").first().click();
  }
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
    }, 45_000);
    handler = (event: Event) => {
      window.clearTimeout(timeout);
      document.removeEventListener("gosxstudio:authoring-result", handler);
      resolve(((event as CustomEvent).detail ?? null) as StudioAuthoringResultDetail | null);
    };
    document.addEventListener("gosxstudio:authoring-result", handler);
  }));
}

export async function waitForStudioFragmentRefresh(page: Page) {
  return page.evaluate(() => new Promise<StudioFragmentRefreshDetail | null>((resolve) => {
    let handler: EventListener;
    const timeout = window.setTimeout(() => {
      document.removeEventListener("gosxstudio:fragments-refresh", handler);
      resolve(null);
    }, 45_000);
    handler = (event: Event) => {
      window.clearTimeout(timeout);
      document.removeEventListener("gosxstudio:fragments-refresh", handler);
      resolve(((event as CustomEvent).detail ?? null) as StudioFragmentRefreshDetail | null);
    };
    document.addEventListener("gosxstudio:fragments-refresh", handler);
  }));
}

export async function waitForStudioAuthoringCycle(page: Page) {
  return page.evaluate(() => new Promise<{ detail: StudioAuthoringResultDetail | null; fragments: StudioFragmentRefreshDetail | null }>((resolve) => {
    let detail: StudioAuthoringResultDetail | null = null;
    let fragments: StudioFragmentRefreshDetail | null = null;
    let resolved = false;
    let authoringHandler: EventListener;
    let fragmentsHandler: EventListener;
    const done = () => {
      if (resolved) return;
      resolved = true;
      window.clearTimeout(timeout);
      document.removeEventListener("gosxstudio:authoring-result", authoringHandler);
      document.removeEventListener("gosxstudio:fragments-refresh", fragmentsHandler);
      resolve({ detail, fragments });
    };
    const timeout = window.setTimeout(done, 45_000);
    authoringHandler = (event: Event) => {
      detail = ((event as CustomEvent).detail ?? null) as StudioAuthoringResultDetail | null;
      window.setTimeout(done, 0);
    };
    fragmentsHandler = (event: Event) => {
      fragments = ((event as CustomEvent).detail ?? null) as StudioFragmentRefreshDetail | null;
    };
    document.addEventListener("gosxstudio:authoring-result", authoringHandler);
    document.addEventListener("gosxstudio:fragments-refresh", fragmentsHandler);
  }));
}

export async function applyCompositionIntentInPlace(page: Page, intentKey: string, options?: { expectedMessage?: string; expectedChangeKind?: string; requireSelection?: boolean; requirePreview?: boolean }) {
  await expectIntentButtonReceivesPointer(page, intentKey);
  const authoringCyclePromise = waitForStudioAuthoringCycle(page);
  const response = await clickIntent(page, intentKey, { reloadAfter: false, settleAfter: false, noWaitAfter: true });
  expect(response.status()).toBe(200);
  const { detail, fragments } = await authoringCyclePromise;
  expect(detail, `${intentKey} should emit authoring result detail`).toBeTruthy();
  expect(detail?.result?.message).toBe(options?.expectedMessage);
  expect(detail?.change?.kind).toBe(options?.expectedChangeKind ?? "component");
  if (options?.requireSelection !== false) {
    expect(detail?.selectedCount ?? 0, `${intentKey} should select the changed surface`).toBeGreaterThan(0);
  }
  if (options?.requirePreview !== false) {
    expect(detail?.previewCount ?? 0, `${intentKey} should refresh preview`).toBeGreaterThan(0);
  }
  expect(detail?.fragmentCount ?? 0, `${intentKey} should refresh structural fragments`).toBeGreaterThan(0);
  expect(fragments?.count ?? detail?.fragmentCount ?? detail?.result?.fragmentCount ?? 0, `${intentKey} should replace at least one fragment`).toBeGreaterThan(0);
  await expect(page.locator("[data-gosx-studio-save-detail]").first()).toHaveText(options?.expectedMessage ?? /./);
  return { response, detail, fragments };
}

export async function applyAuthoringPanelInPlace(page: Page, panelSelector: string, options?: { expectedMessage?: string; expectedChangeKind?: string }) {
  await expectPanelButtonReceivesPointer(page, panelSelector);
  const authoringCyclePromise = waitForStudioAuthoringCycle(page);
  const response = await clickAuthoringPanel(page, panelSelector, { reloadAfter: false, settleAfter: false, noWaitAfter: true });
  expect(response.status()).toBe(200);
  const { detail, fragments } = await authoringCyclePromise;
  expect(detail, `${panelSelector} should emit authoring result detail`).toBeTruthy();
  expect(detail?.result?.message).toBe(options?.expectedMessage);
  expect(detail?.change?.kind).toBe(options?.expectedChangeKind ?? "component");
  expect(detail?.selectedCount ?? 0, `${panelSelector} should select the changed surface`).toBeGreaterThan(0);
  expect(detail?.previewCount ?? 0, `${panelSelector} should refresh preview`).toBeGreaterThan(0);
  expect(detail?.fragmentCount ?? 0, `${panelSelector} should refresh structural fragments`).toBeGreaterThan(0);
  expect(fragments?.count ?? detail?.fragmentCount ?? detail?.result?.fragmentCount ?? 0, `${panelSelector} should replace at least one fragment`).toBeGreaterThan(0);
  await expect(page.locator("[data-gosx-studio-save-detail]").first()).toHaveText(options?.expectedMessage ?? /./);
  return { response, detail, fragments };
}

export async function saveEditableControl(page: Page, value: string, options?: ClickAuthoringOptions & { expectedMessage?: string }) {
  const panel = page.locator("[data-gosx-studio-editable-control='true']").first();
  await expect(panel).toBeVisible();
  // The value field renders as <input> for plain controls and <textarea> for
  // rich-text controls (e.g. Muddy's hero headline is studio.ControlRichText
  // — app/admin/editor/site_map.server.go — so it always renders a
  // textarea); both share the stable name="gosx_studio_value", so match
  // either widget kind rather than assuming a plain <input>.
  const input = panel.locator("input[name='gosx_studio_value'], textarea[name='gosx_studio_value']");
  await expect(input).toBeVisible();
  await setFieldValue(input, value);
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
    await expect(page.locator("[data-gosx-studio-save-detail]").first()).toHaveText(options.expectedMessage ?? /./);
    return;
  }
  await expect(page.locator("[data-gosx-studio-editable-control='true'] input[name='gosx_studio_value'], [data-gosx-studio-editable-control='true'] textarea[name='gosx_studio_value']").first()).toHaveValue(value);
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
  await setFieldValue(titleInput, title);
  await setFieldValue(routeInput, route);
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

export async function toggleComponentVisibility(page: Page, options?: ClickAuthoringOptions) {
  const panelSelector = "[data-gosx-studio-component-visibility='true'], [data-studio-site-map-component-visibility='true']";
  const buttonSelector = "[data-gosx-studio-component-visibility='true'] button, [data-studio-site-map-component-visibility='true'] button";
  const panel = page.locator(panelSelector).first();
  await expect(panel).toBeVisible();
  const visibleInput = panel.locator("input[name='gosx_studio_visible']");
  await expect(visibleInput).toHaveCount(1);
  const submittedVisible = (await visibleInput.first().getAttribute("value")) === "true";
  await expectButtonReceivesPointer(page, buttonSelector, "component visibility");
  const authoringResultPromise = options?.reloadAfter === false ? waitForStudioAuthoringResult(page) : null;
  const clickOptions = options?.reloadAfter === false ? { ...options, settleAfter: false } : options;
  const response = await clickEditorActionButton(page, buttonSelector, "/__actions/authoring", clickOptions);
  if (options?.reloadAfter === false) {
    expect(response.status()).toBe(200);
    const detail = await authoringResultPromise;
    expect(detail, "real editor should emit component visibility authoring result detail").toBeTruthy();
    expect(detail?.change?.kind).toBe("component");
    expect(detail?.selectedCount ?? 0, "visibility save should select the changed component").toBeGreaterThan(0);
    expect(detail?.previewCount ?? 0, "visibility save should refresh at least one preview frame").toBeGreaterThan(0);
    expect(detail?.result?.values?.gosx_studio_visible).toBe(submittedVisible ? "true" : "false");
    await expect(page.locator("[data-gosx-studio-save-detail]").first()).toHaveText(detail?.result?.message ?? /./);
    return;
  }
  expect(response.status()).toBe(303);
  expect(authoringParam(response, "gosx_studio_operation")).toBe("toggle-visibility");
}

export async function reorderComponent(page: Page, options?: ClickAuthoringOptions) {
  const panelSelector = "[data-gosx-studio-component-reorder='true'], [data-studio-site-map-component-reorder='true']";
  const buttonSelector = "[data-gosx-studio-component-reorder='true'] button, [data-studio-site-map-component-reorder='true'] button";
  const panel = page.locator(panelSelector).first();
  await expect(panel).toBeVisible();
  const positionInput = panel.locator("input[name='gosx_studio_position']");
  await expect(positionInput).toHaveCount(1);
  const submittedPosition = Number(await positionInput.first().getAttribute("value"));
  expect(Number.isFinite(submittedPosition), "reorder form should carry a numeric target position").toBeTruthy();
  const expectedPosition = submittedPosition;
  await expectButtonReceivesPointer(page, buttonSelector, "component reorder");
  const authoringResultPromise = options?.reloadAfter === false ? waitForStudioAuthoringResult(page) : null;
  const clickOptions = options?.reloadAfter === false ? { ...options, settleAfter: false } : options;
  const response = await clickEditorActionButton(page, buttonSelector, "/__actions/authoring", clickOptions);
  if (options?.reloadAfter === false) {
    expect(response.status()).toBe(200);
    const detail = await authoringResultPromise;
    expect(detail, "real editor should emit component reorder authoring result detail").toBeTruthy();
    expect(detail?.change?.kind).toBe("component");
    expect(detail?.selectedCount ?? 0, "reorder save should select the changed component").toBeGreaterThan(0);
    expect(detail?.previewCount ?? 0, "reorder save should refresh at least one preview frame").toBeGreaterThan(0);
    expect(detail?.result?.values?.gosx_studio_position).toBe(String(expectedPosition));
    expect(detail?.result?.message ?? "", "reorder save should carry a result message").toMatch(/\S/);
    return;
  }
  expect(response.status()).toBe(303);
  expect(authoringParam(response, "gosx_studio_operation")).toBe("reorder-component");
}

async function setFieldValue(input: Locator, value: string) {
  await input.evaluate((el, nextValue) => {
    if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) {
      el.value = nextValue;
    }
  }, value);
  await expect(input).toHaveValue(value);
}

export function authoringParam(response: { request(): { postData(): string | null } }, key: string) {
  const body = response.request().postData() ?? "";
  const encoded = new URLSearchParams(body).get(key);
  if (encoded !== null) return encoded;
  const escaped = key.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const match = new RegExp(`name="${escaped}"\\s*\\r?\\n\\r?\\n([^\\r\\n]*)`).exec(body);
  return match?.[1] ?? null;
}

export async function startMuddy(request: APIRequestContext, extraEnv?: Record<string, string>): Promise<ServerHandle> {
  await ensureMuddyDist();
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
      ...(extraEnv ?? {}),
    },
    tempDir,
  });
}

export async function startMuddyCollaboration(request: APIRequestContext): Promise<ServerHandle> {
  await ensureMuddyDist();
  const port = await freePort();
  const tempDir = mkdtempSync(path.join(tmpdir(), "gosx-studio-muddy-collab-e2e-"));
  mkdirSync(path.join(tempDir, "media"), { recursive: true });
  return startGoServer(request, {
    cwd: muddyRepo,
    command: "./cmd/muddy-noni",
    baseURL: `http://127.0.0.1:${port}`,
    env: {
      PORT: String(port),
      PUBLIC_SITE_URL: `http://127.0.0.1:${port}`,
      SESSION_SECRET: "noni-collaboration-playwright-signed-session-secret",
      MUDDY_ENVIRONMENT: "test",
      MUDDY_COLLAB_TEST_AUTH: "1",
      MUDDY_DATA_PATH: path.join(tempDir, "cms.json"),
      MUDDY_FLOW_DATA_PATH: path.join(tempDir, "flows.json"),
      MUDDY_LIFECYCLE_DB_PATH: path.join(tempDir, "lifecycle.db"),
      MUDDY_STUDIO_DB_PATH: path.join(tempDir, "studio.db"),
      MUDDY_MEDIA_PATH: path.join(tempDir, "media"),
      MUDDY_LIFECYCLE_WORKER: "0",
      MUDDY_MOCK_CHECKOUT: "1",
    },
    tempDir,
  });
}

function ensureMuddyDist(): Promise<void> {
  if (!muddyDistBuildPromise) {
    muddyDistBuildPromise = buildMuddyDist().catch((error) => {
      muddyDistBuildPromise = null;
      throw error;
    });
  }
  return muddyDistBuildPromise;
}

async function buildMuddyDist(): Promise<void> {
  const gosxBin = process.env.GOSX_STUDIO_GOSX_BIN ?? "gosx";
  const env = {
    ...process.env,
    GOWORK: "off",
    PATH: process.env.GOSX_STUDIO_GOSX_BIN ? process.env.PATH : withPathEntry(process.env.PATH, defaultGoBin),
  };
  await runLoggedCommand(gosxBin, ["build", "--dev", "."], {
    cwd: muddyRepo,
    env,
    label: "Muddy GoSX dist build",
  });
}

function withPathEntry(currentPath: string | undefined, entry: string): string {
  const parts = (currentPath ?? "").split(path.delimiter).filter(Boolean);
  if (parts.includes(entry)) return currentPath ?? entry;
  return [entry, ...parts].join(path.delimiter);
}

// startMuddyCanvas boots Muddy/Noni with explicit co-render mode
// (MUDDY_SITEMAP_CANVAS=1). The editor renders the canvas board surface in
// addition to the DOM board for parity/debug comparisons.
export async function startMuddyCanvas(request: APIRequestContext): Promise<ServerHandle> {
  return startMuddy(request, { MUDDY_SITEMAP_CANVAS: "1" });
}

// startMuddyCanvasDefault boots Muddy/Noni with no canvas env override, proving
// the production default. The low-WASM static WebGPU/Canvas2D-fallback board
// becomes the SOLE site-map graph visual and the DOM board's graph sub-tree
// (.studio-site-map-workspace__surface — the workspace layers/nodes + SVG link
// paths) is hidden via gated CSS, while the DOM board element, its
// sitemapruntime binding, the selection detail card, and the host authoring
// panels stay live.
export async function startMuddyCanvasDefault(request: APIRequestContext): Promise<ServerHandle> {
  return startMuddy(request);
}

// startMuddyCanvasFullWASM boots Muddy/Noni in explicit full-WASM CanvasBoard
// mode (MUDDY_SITEMAP_CANVAS=canvas-default). This preserves compatibility
// coverage for the gosx CanvasBoard surface, shared runtime globals, and
// selection bridge without treating that mode as the production default.
export async function startMuddyCanvasFullWASM(request: APIRequestContext): Promise<ServerHandle> {
  return startMuddy(request, { MUDDY_SITEMAP_CANVAS: "canvas-default" });
}

// startMuddyCanvasWASMFree boots Muddy/Noni in WASM-free Canvas2D mode
// (MUDDY_CANVAS_WASM_FREE=1): the same low-WASM static WebGPU/Canvas2D-fallback
// path as the production default. Kept as an explicit forcing helper for parity
// and regression tests.
export async function startMuddyCanvasWASMFree(request: APIRequestContext): Promise<ServerHandle> {
  return startMuddy(request, { MUDDY_CANVAS_WASM_FREE: "1" });
}

// startMuddyCanvasHTMLSurface boots Muddy/Noni on the low-WASM canvas route and
// injects one real Kind:"html" CanvasBoardNode onto the board via
// MUDDY_EDITOR_SCENE_DOM=1. The editor then paints a DOM surface (the "hero"
// node) over the Canvas2D board through the same server-precomputed RenderBundle
// path — landing in bundle.html and rendered into the canvas overlay by the
// WASM-free client with no full CanvasBoard WASM. The HTML-surface injection is
// opt-in; only the canvas-html-surface parity e2e flips it on.
export async function startMuddyCanvasHTMLSurface(request: APIRequestContext): Promise<ServerHandle> {
  return startMuddy(request, { MUDDY_CANVAS_WASM_FREE: "1", MUDDY_EDITOR_SCENE_DOM: "1" });
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
      PAJARITOS_MOCK_AUTH: "1",
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

type LoggedCommandOptions = {
  cwd: string;
  env: NodeJS.ProcessEnv;
  label: string;
};

// ── Orphaned-process-group prevention ───────────────────────────────────────
//
// `go run <pkg>` does not exec(2) into the compiled binary — it forks a real
// child process for it and stays alive as a supervisor/relay. On a graceful
// SIGINT, `go run` forwards the signal to that child and waits for it, but a
// SIGKILL (our 5s-timeout fallback below) or an abrupt external kill of the
// Node harness itself (mid-test Ctrl-C/SIGINT on the Playwright process, a
// timed-out test, an afterAll that never runs) kills `go run` with no chance
// to relay anything — its exec'd `muddy-noni`/`pajaritos` binary is simply
// reparented to init and keeps running forever, accumulating across runs
// (proven: a `muddy-noni` binary from an earlier ad hoc session was still
// alive, PPID reparented to /init, when this fix started).
//
// The fix is to always start these processes `detached: true`. On POSIX,
// that makes each spawned process its own process-group LEADER (pgid ===
// its own pid) instead of joining the harness's group. `go run`'s exec'd
// child inherits `go run`'s (new, detached) pgid by default (Go's os/exec
// does not set a new group for its own children), so `process.kill(-pgid,
// signal)` — the negative-pid form targets the whole process GROUP, not
// just one process — reaches `go run` AND its compiled-binary child in one
// signal, however the group leader dies (graceful or SIGKILL).
//
// `activeProcessGroups` tracks every live group so a process-level SIGINT/
// SIGTERM/exit on the Playwright/Node runner itself (a mid-test abort) also
// sweeps every still-tracked group as a last resort, even if a test's own
// `finally`/`afterAll` never gets to call `stop()`.
const activeProcessGroups = new Set<number>();

function trackProcessGroup(pid: number | undefined) {
  if (pid !== undefined) activeProcessGroups.add(pid);
}

function untrackProcessGroup(pid: number | undefined) {
  if (pid !== undefined) activeProcessGroups.delete(pid);
}

function killProcessGroup(pid: number, signal: NodeJS.Signals) {
  try {
    process.kill(-pid, signal);
  } catch {
    // ESRCH: the group is already gone (process exited/reaped on its own,
    // or this is a stale entry from a group we already killed). Cleanup is
    // best-effort here — never throw out of a shutdown path.
  }
}

let processGroupCleanupInstalled = false;

// installProcessGroupCleanupOnce registers a last-resort sweep of every
// still-tracked process group on process exit and on SIGINT/SIGTERM
// delivered to this Node process (e.g. Ctrl-C on `npx playwright test`).
// It intentionally does not call process.exit() itself — Playwright installs
// its own SIGINT handling for graceful worker teardown/reporting; we only
// need to piggyback cleanup, not own the exit sequence.
function installProcessGroupCleanupOnce() {
  if (processGroupCleanupInstalled) return;
  processGroupCleanupInstalled = true;
  const sweep = () => {
    for (const pid of activeProcessGroups) killProcessGroup(pid, "SIGKILL");
    activeProcessGroups.clear();
  };
  process.on("exit", sweep);
  process.on("SIGINT", sweep);
  process.on("SIGTERM", sweep);
}

async function runLoggedCommand(command: string, args: string[], options: LoggedCommandOptions): Promise<void> {
  installProcessGroupCleanupOnce();
  const proc = spawn(command, args, {
    cwd: options.cwd,
    env: options.env,
    stdio: ["ignore", "pipe", "pipe"],
    detached: true,
  });
  trackProcessGroup(proc.pid);
  const stdout: string[] = [];
  const stderr: string[] = [];
  proc.stdout?.on("data", (chunk: Buffer) => stdout.push(chunk.toString()));
  proc.stderr?.on("data", (chunk: Buffer) => stderr.push(chunk.toString()));

  return new Promise((resolve, reject) => {
    proc.on("error", (error) => {
      untrackProcessGroup(proc.pid);
      reject(new Error(`${options.label} failed to start: ${error.message}\n\nstdout:\n${stdout.join("")}\n\nstderr:\n${stderr.join("")}`));
    });
    proc.on("exit", (code, signal) => {
      untrackProcessGroup(proc.pid);
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`${options.label} failed with ${signal ? `signal ${signal}` : `exit code ${code}`}\n\nCommand: ${command} ${args.join(" ")}\nCwd: ${options.cwd}\n\nstdout:\n${stdout.join("")}\n\nstderr:\n${stderr.join("")}`));
    });
  });
}

async function startGoServer(request: APIRequestContext, options: ServerOptions): Promise<ServerHandle> {
  installProcessGroupCleanupOnce();
  const proc = spawn("go", ["run", options.command], {
    cwd: options.cwd,
    env: { ...process.env, GOWORK: "off", ...options.env },
    stdio: ["ignore", "pipe", "pipe"],
    detached: true,
  });
  trackProcessGroup(proc.pid);
  const logs: string[] = [];
  proc.stdout?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));
  proc.stderr?.on("data", (chunk: Buffer) => logs.push(chunk.toString()));

  try {
    await waitForServer(request, options.baseURL);
    await warmEditor(request, options.baseURL);
  } catch (error) {
    await stopProcess(proc);
    cleanupTempDir(options.tempDir);
    throw new Error(`${String(error)}\n\nServer logs:\n${logs.join("")}`);
  }

  return {
    baseURL: options.baseURL,
    dataDir: options.tempDir,
    stop: async () => {
      await stopProcess(proc);
      cleanupTempDir(options.tempDir);
    },
  };
}

async function waitForServer(request: APIRequestContext, baseURL: string) {
  const deadline = Date.now() + 60_000;
  let lastError = "";
  while (Date.now() < deadline) {
    try {
      const response = await request.get(baseURL, {
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
  throw new Error(`timed out waiting for ${baseURL} (${lastError})`);
}

async function warmEditor(request: APIRequestContext, baseURL: string) {
  const response = await request.get(`${baseURL}/admin/editor`, {
    failOnStatusCode: false,
    timeout: 90_000,
  });
  if (response.status() < 200 || response.status() >= 500) {
    throw new Error(`editor warmup returned status ${response.status()}`);
  }
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
  const pid = proc.pid;
  try {
    if (proc.exitCode !== null || proc.signalCode !== null) return;
    const exited = new Promise<void>((resolve) => proc.once("exit", () => resolve()));
    // Kill the whole process GROUP (negative pid), not just the immediate
    // child: `go run` does not exec(2) into its compiled binary, so a plain
    // proc.kill() only reaches the `go run` supervisor and leaves the
    // muddy-noni/pajaritos binary it forked running as an orphan. Both
    // processes share this pgid because we spawned with `detached: true`
    // (see installProcessGroupCleanupOnce's comment above).
    if (pid !== undefined) killProcessGroup(pid, "SIGINT");
    else proc.kill("SIGINT");
    const timeout = new Promise<void>((resolve) => {
      setTimeout(() => {
        if (proc.exitCode === null && proc.signalCode === null) {
          if (pid !== undefined) killProcessGroup(pid, "SIGKILL");
          else proc.kill("SIGKILL");
        }
        resolve();
      }, 5_000);
    });
    await Promise.race([exited, timeout]);
  } finally {
    untrackProcessGroup(pid);
  }
}

function cleanupTempDir(tempDir: string) {
  rmSync(tempDir, { force: true, recursive: true });
}
