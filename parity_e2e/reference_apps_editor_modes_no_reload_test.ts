import { expect, test, type Page } from "@playwright/test";
import { installEditorDocumentContinuityProbe } from "./reference_apps_document_continuity";
import { gotoEditor, startPajaritos } from "./reference_apps_harness";

const RENDERER = "[data-gosx-studio-backend-editor-renderer='gosx-studio']";
const ADVANCED_PANEL = "[data-pajaritos-editor-advanced-panel='true']";
const LOOK_PANEL = "[data-pajaritos-editor-look-panel='true']";
const PREVIEW_FRAME = "[data-studio-preview-frame].editor-preview-frame";
const HERO_LAYER_PICK = "[data-studio-home-layer-pick='hero']";
const PROGRAMS_LAYER_PICK = "[data-studio-home-layer-pick='programs']";
const WORKBENCH = "form[data-studio-workbench]";

async function modeButton(page: Page, mode: string) {
  return page.locator(`[data-studio-mode-control='${mode}']`).first();
}

async function setMode(page: Page, mode: string) {
  const button = await modeButton(page, mode);
  await expect(button, `${mode} mode control`).toBeVisible();
  await button.click();
  await expect(button, `${mode} mode control pressed`).toHaveAttribute("aria-pressed", "true");
}

async function expectNoPageOverflow(page: Page) {
  const metrics = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    innerWidth: window.innerWidth,
  }));
  expect(metrics.scrollWidth, `page should not overflow horizontally: ${JSON.stringify(metrics)}`).toBeLessThanOrEqual(metrics.innerWidth + 2);
}

async function expectDefaultEditorCompact(page: Page, maxScrollHeight: number) {
  const metrics = await page.evaluate(() => ({
    scrollHeight: document.documentElement.scrollHeight,
    innerHeight: window.innerHeight,
  }));
  expect(metrics.scrollHeight, `default editor should not dump support controls below workbench: ${JSON.stringify(metrics)}`).toBeLessThan(maxScrollHeight);
}

async function expectUsablePreviewFrame(page: Page, minHeight: number) {
  const metrics = await page.locator(PREVIEW_FRAME).first().evaluate((frame: HTMLIFrameElement) => {
    const rect = frame.getBoundingClientRect();
    const doc = frame.contentDocument;
    return {
      height: rect.height,
      width: rect.width,
      readyState: doc?.readyState ?? null,
      documentHeight: doc?.documentElement?.scrollHeight ?? 0,
      text: doc?.body?.innerText ?? "",
    };
  });
  expect(metrics.height, `preview iframe should expose a usable editing viewport: ${JSON.stringify(metrics)}`).toBeGreaterThanOrEqual(minHeight);
  expect(metrics.documentHeight, `preview iframe should contain loaded page content: ${JSON.stringify(metrics)}`).toBeGreaterThan(1_000);
  expect(metrics.text).toContain("A forest school where curiosity gets muddy.");
}

async function expectPolishedSmallControls(page: Page) {
  const controls = page.locator(`${HERO_LAYER_PICK}, [data-studio-mode-control='home']`);
  await expect(controls.first()).toBeVisible();
  const styles = await controls.evaluateAll((nodes) => nodes.map((node) => {
    const style = getComputedStyle(node as HTMLElement);
    return {
      appearance: style.getPropertyValue("appearance"),
      borderWidth: style.getPropertyValue("border-top-width"),
      borderRadius: style.getPropertyValue("border-radius"),
      boxShadow: style.getPropertyValue("box-shadow"),
      background: style.getPropertyValue("background-color"),
    };
  }));
  for (const style of styles) {
    expect(style.appearance, `control should opt out of native browser appearance: ${JSON.stringify(style)}`).toBe("none");
    expect(style.borderWidth, `control should have a visible styled border: ${JSON.stringify(style)}`).not.toBe("0px");
    expect(style.borderRadius, `control should have Paper & Ink pill shape: ${JSON.stringify(style)}`).not.toBe("0px");
    expect(style.boxShadow, `control should use styled control shadow: ${JSON.stringify(style)}`).not.toBe("none");
  }
}

async function expectReadablePropertiesScope(page: Page) {
  const scopeStrip = page.locator(".studio-inspector-chrome .studio-scope-strip").first();
  await expect(scopeStrip).toBeVisible();
  const styles = await scopeStrip.evaluate((node) => {
    const strip = getComputedStyle(node as HTMLElement);
    const second = node.querySelector("span + span") as HTMLElement | null;
    return {
      display: strip.display,
      gap: strip.columnGap,
      separator: second ? getComputedStyle(second, "::before").content : "",
      text: (node as HTMLElement).innerText,
    };
  });
  expect(styles.display).toBe("flex");
  expect(styles.separator, `scope strip should visually separate adjacent breadcrumbs: ${JSON.stringify(styles)}`).toContain("/");
  expect(styles.text).toContain("Site");
}

async function expectHeroLayerSelection(page: Page) {
  await expect(page.locator(WORKBENCH), "Hero layer click should set the shared block selection on the workbench").toHaveAttribute("data-studio-selection", "hero");
  await expect(page.locator(WORKBENCH), "Hero layer click should mark the shared selection as a block").toHaveAttribute("data-studio-selection-kind", "block");
  await expect(page.locator(`${HERO_LAYER_PICK}.is-selected`), "Hero layer chip should double as the selected block row").toBeAttached();
  await expect(page.locator(HERO_LAYER_PICK), "Hero layer chip should stay pressed after the shared selection update").toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("[data-studio-selection-label]").first(), "selection readout should update from the Hero row label").toContainText(/hero/i);
  await expect(page.locator("[data-studio-selection-status]").first(), "Properties/preview selection status should leave the idle No selection state").toContainText(/visible|preview selection/i);
  await expect(page.locator("[data-studio-inspector-for~='hero']").first(), "Properties inspector for Hero should remain visible after chip selection").toBeVisible();
  await expect(page.frameLocator("iframe[title='Pajaritos home page authoring canvas']").locator("[data-studio-block-key='hero']").first(), "preview document should expose the matching Hero block identity").toBeAttached({ timeout: 30_000 });
}

test.describe("@reference-apps Pajaritos editor mode panels", () => {
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("default editor stays compact and Advanced/Look panels are reachable without main-document reload", async ({ page, request }) => {
    const server = await startPajaritos(request);
    try {
      await page.setViewportSize({ width: 1440, height: 1000 });
      await gotoEditor(page, server.baseURL);

      await expect(page.locator(RENDERER), "Pajaritos editor must expose the Studio backend renderer boundary").toBeAttached();
      await expect(page.locator(ADVANCED_PANEL), "Advanced support panel must render").toBeAttached();
      await expect(page.locator(LOOK_PANEL), "Look component-style panel must render").toBeAttached();
      await expect(page.locator(ADVANCED_PANEL)).toBeHidden();
      await expect(page.locator(LOOK_PANEL)).toBeHidden();
      await expectDefaultEditorCompact(page, 4_400);
      await expectNoPageOverflow(page);
      await expectUsablePreviewFrame(page, 440);
      await expectPolishedSmallControls(page);
      await expectReadablePropertiesScope(page);

      const probe = await installEditorDocumentContinuityProbe(page, { settleMs: 125 });
      try {
        await expect(page.locator(HERO_LAYER_PICK), "Hero layer chip should remain visible and styled as the selection affordance").toBeVisible();
        await expect(page.locator(PROGRAMS_LAYER_PICK), "Programs layer chip should be available to prove Hero click changes shared selection").toBeVisible();
        await page.locator(PROGRAMS_LAYER_PICK).click();
        await expect(page.locator(WORKBENCH), "Programs chip should move shared selection away from Hero").toHaveAttribute("data-studio-selection", "programs");
        await expect(page.locator(PROGRAMS_LAYER_PICK), "Programs chip should become the pressed layer").toHaveAttribute("aria-pressed", "true");
        await page.locator(HERO_LAYER_PICK).click();
        await expectHeroLayerSelection(page);
        await probe.assertStillOnSameDocument();

        await setMode(page, "advanced");
        await expect(page.locator(ADVANCED_PANEL)).toBeVisible();
        await expect(page.locator(`${ADVANCED_PANEL} [data-gosx-studio-collaboration='true']`)).toBeAttached();
        await expect(page.locator(`${ADVANCED_PANEL} [data-studio-flow-field-editor='true']`)).toBeAttached();
        await expect(page.locator(`${ADVANCED_PANEL} [data-studio-shared-components-panel='true']`)).toBeAttached();
        await expect(page.locator(`${ADVANCED_PANEL} [data-studio-media-summary='true']`)).toBeAttached();
        await expect(page.locator(`${ADVANCED_PANEL} [data-studio-quarantine-panel='true']`)).toBeAttached();
        await expect(page.locator(".studio-publish-controls")).toBeAttached();
        await probe.assertStillOnSameDocument();
        await expectNoPageOverflow(page);

        await setMode(page, "look");
        await expect(page.locator(LOOK_PANEL)).toBeVisible();
        await expect(page.locator(`${LOOK_PANEL} [data-gosx-studio-component-styles='true']`)).toBeVisible();
        await expect(page.locator(`${LOOK_PANEL} .studio-style-apply`).first()).toBeVisible();
        await probe.assertStillOnSameDocument();
        await expectNoPageOverflow(page);

        await setMode(page, "home");
        await expect(page.locator(ADVANCED_PANEL)).toBeHidden();
        await expect(page.locator(LOOK_PANEL)).toBeHidden();
        await expect(page.locator(`${ADVANCED_PANEL} :focus, ${LOOK_PANEL} :focus`), "focus should not remain inside a hidden supplemental mode panel").toHaveCount(0);
        await probe.assertStillOnSameDocument();
        await expectDefaultEditorCompact(page, 4_400);
      } finally {
        probe.dispose();
      }

      await page.setViewportSize({ width: 390, height: 844 });
      await page.reload({ waitUntil: "domcontentloaded" });
      await expect(page.locator(RENDERER)).toBeAttached();
      await expectDefaultEditorCompact(page, 4_400);
      await expectNoPageOverflow(page);
      await expectUsablePreviewFrame(page, 400);
      await expectPolishedSmallControls(page);
      await setMode(page, "advanced");
      await expect(page.locator(ADVANCED_PANEL)).toBeVisible();
      await expectNoPageOverflow(page);
      await setMode(page, "look");
      await expect(page.locator(LOOK_PANEL)).toBeVisible();
      await expectNoPageOverflow(page);
    } finally {
      await server.stop();
    }
  });
});
