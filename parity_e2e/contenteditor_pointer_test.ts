import { expect, type Page, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../contenteditorruntime/content_editor.js"),
  "utf8",
);

const studioCSS = readFileSync(
  path.resolve(__dirname, "../hostruntime/assets/studio.css"),
  "utf8",
);

const initialDocument = {
  version: 1,
  title: "Physical Page CMS fixture",
  blocks: [
    { id: "alpha", type: "paragraph", text: "Alpha block", extra: { keep: "alpha" } },
    { id: "beta", type: "heading", text: "Beta block", level: "3", extra: { keep: "beta" } },
  ],
};

type DragTelemetryEvent = {
  type: string;
  phase: string;
  targetID: string;
  rawTargetTag: string;
  rawTargetID: string;
  rawTargetClass: string;
  defaultPrevented: boolean;
  clientX: number;
  clientY: number;
  effectAllowed: string;
  dropEffect: string;
  dataTransferTypes: string[];
};

type DragPoint = { x: number; y: number };
type DragBox = { x: number; y: number; width: number; height: number };

type PhysicalDragAttempt = {
  ok: boolean;
  beforeSource: string;
  beforeOrder: string[];
  geometry: { start: DragPoint; end: DragPoint; source: DragBox; target: DragBox };
  telemetry: DragTelemetryEvent[];
  markerContent: string;
};

type PhysicalWindow = Window & {
  __physicalContentDndEvents?: DragTelemetryEvent[];
};

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function editorHTML(): string {
  const source = JSON.stringify(initialDocument, null, 2);
  return `
    <!doctype html>
    <html>
      <head>
        <meta charset="utf-8">
        <style>${studioCSS}</style>
        <style>
          html, body { margin: 0; min-height: 100%; }
          body { background: var(--color-canvas); }
          .physical-dnd-fixture { width: min(100%, 1120px); margin: 0 auto; padding: 24px 32px 140px; }
          .physical-dnd-fixture .admin-heading { margin-bottom: 12px; }
          .physical-dnd-fixture .admin-form { display: grid; gap: 16px; }
          #outside-cancel-zone {
            position: fixed;
            z-index: 40;
            top: 20px;
            right: 20px;
            width: 130px;
            min-height: 74px;
            border: 2px dashed var(--color-line-strong);
            border-radius: var(--radius-card);
            background: var(--color-panel);
            color: var(--color-ink-soft);
            padding: 12px;
            font-family: var(--font-mono);
            font-size: var(--text-xs);
          }
        </style>
      </head>
      <body>
        <main class="admin-page editor-page physical-dnd-fixture">
          <header class="admin-heading">
            <p class="kicker">Page CMS</p>
            <h1>Physical content editor</h1>
          </header>
          <form id="page-form" class="admin-form">
            <input type="hidden" name="csrf_token" value="fixture-csrf">
            <div class="field-row field-row--wide">
              <label for="bodyFormat">Content format</label>
              <select id="bodyFormat" name="bodyFormat" data-content-editor-format="true">
                <option value="blocks" selected>GoSX blocks</option>
                <option value="mdpp">MDPP</option>
              </select>
            </div>
            <div class="field-row field-row--wide content-editor" data-content-editor="true">
              <label for="body">Body</label>
              <textarea class="content-editor__source" id="body" name="body" rows="8" data-content-editor-source="true">${escapeHTML(source)}</textarea>
              <div class="content-editor__toolbar">
                <button type="button" data-content-add="paragraph">Paragraph</button>
                <button type="button" data-content-add="heading">Heading</button>
              </div>
              <div class="content-block-list" data-content-editor-list="true"></div>
            </div>
          </form>
          <aside id="outside-cancel-zone" aria-label="Outside drag cancel zone">Outside list</aside>
        </main>
      </body>
    </html>
  `;
}

async function mountFixture(page: Page): Promise<void> {
  await page.setViewportSize({ width: 1280, height: 800 });
  await page.setContent(editorHTML());
  await page.addScriptTag({ content: runtimeJS });
  await expect(page.locator("[data-content-editor]"))
    .toHaveAttribute("data-content-editor-source-state", "ready");
  await installTelemetry(page);
}

async function installTelemetry(page: Page): Promise<void> {
  await page.evaluate(() => {
    const browserWindow = window as unknown as PhysicalWindow;
    const events: DragTelemetryEvent[] = [];
    const eventNames = ["dragstart", "dragenter", "dragover", "dragleave", "drop", "dragend"];
    const record = (event: Event, phase: string) => {
      const rawTarget = event.target instanceof Element ? event.target : null;
      const target = rawTarget
        ? rawTarget.closest("[data-content-block-id]")
        : null;
      const pointerEvent = event as DragEvent;
      const dataTransfer = pointerEvent.dataTransfer;
      events.push({
        type: event.type,
        phase,
        targetID: target?.getAttribute("data-content-block-id") ?? "",
        rawTargetTag: rawTarget?.tagName.toLowerCase() ?? "",
        rawTargetID: rawTarget?.id ?? "",
        rawTargetClass: typeof rawTarget?.className === "string" ? rawTarget.className : "",
        defaultPrevented: event.defaultPrevented,
        clientX: pointerEvent.clientX ?? 0,
        clientY: pointerEvent.clientY ?? 0,
        effectAllowed: dataTransfer?.effectAllowed ?? "",
        dropEffect: dataTransfer?.dropEffect ?? "",
        dataTransferTypes: dataTransfer ? Array.from(dataTransfer.types ?? []) : [],
      });
    };
    eventNames.forEach((name) => {
      document.addEventListener(name, (event) => record(event, "capture"), true);
      document.addEventListener(name, (event) => record(event, "bubble"));
    });
    browserWindow.__physicalContentDndEvents = events;
  });
}

async function readTelemetry(page: Page): Promise<DragTelemetryEvent[]> {
  return page.evaluate(() => {
    const browserWindow = window as unknown as PhysicalWindow;
    return browserWindow.__physicalContentDndEvents?.slice() ?? [];
  });
}

async function blockOrder(page: Page): Promise<string[]> {
  return page.locator("[data-content-block-id]").evaluateAll((rows) =>
    rows.map((row) => row.getAttribute("data-content-block-id") ?? ""));
}

async function beginPhysicalDrag(
  page: Page,
  sourceID: string,
  targetID: string,
): Promise<PhysicalDragAttempt> {
  const sourceRow = page.locator(`[data-content-block-id="${sourceID}"]`);
  const sourceHandle = sourceRow.locator("[data-content-drag-handle]");
  const targetRow = page.locator(`[data-content-block-id="${targetID}"]`);
  const sourceBox = await sourceRow.boundingBox();
  const handleBox = await sourceHandle.boundingBox();
  const targetBox = await targetRow.boundingBox();
  if (!sourceBox || !handleBox || !targetBox) {
    throw new Error(`missing physical drag geometry source=${sourceID} target=${targetID}`);
  }

  const beforeSource = await page.locator("#body").inputValue();
  const beforeOrder = await blockOrder(page);
  const start = {
    x: handleBox.x + handleBox.width / 2,
    y: handleBox.y + handleBox.height / 2,
  };
  const end = {
    x: targetBox.x + targetBox.width / 2,
    y: targetBox.y + Math.min(12, Math.max(4, targetBox.height * 0.2)),
  };

  await page.mouse.move(start.x, start.y);
  await page.mouse.down();
  await page.mouse.move(end.x, end.y, { steps: 12 });
  // Keep the pointer over the actual drop target for a second native move.
  // Playwright's manual-drag guidance uses two moves over the drop element so
  // engines that retarget descendants during a path still dispatch drop.
  await page.mouse.move(end.x, end.y);

  let markerContent = "";
  let ok = true;
  try {
    await expect(sourceRow).toHaveAttribute("data-content-editor-drag-preview", "true", { timeout: 1_500 });
    await expect(targetRow).toHaveAttribute("data-content-editor-drop-before", "true", { timeout: 1_500 });
    markerContent = await targetRow.evaluate((row) => getComputedStyle(row, "::before").content);
  } catch {
    ok = false;
  }

  if (!ok) await page.mouse.up();
  return {
    ok,
    beforeSource,
    beforeOrder,
    geometry: { start, end, source: sourceBox, target: targetBox },
    telemetry: await readTelemetry(page),
    markerContent,
  };
}

async function beginPhysicalDragWithRetry(
  page: Page,
  sourceID: string,
  targetID: string,
): Promise<PhysicalDragAttempt & { attempt: number }> {
  const failures: Array<PhysicalDragAttempt & { attempt: number }> = [];
  for (let attempt = 1; attempt <= 2; attempt += 1) {
    if (attempt > 1) await mountFixture(page);
    const result = await beginPhysicalDrag(page, sourceID, targetID);
    const withAttempt = { ...result, attempt };
    if (result.ok) return withAttempt;
    failures.push(withAttempt);
  }
  throw new Error(`native physical drag did not expose drag markers after two attempts: ${JSON.stringify(failures)}`);
}

test.describe("@smoke shared content editor physical mouse drag", () => {
  test("reorders a page block with real mouse input and exposes the CSS drop marker", async ({ page }) => {
    await mountFixture(page);
    const drag = await beginPhysicalDragWithRetry(page, "beta", "alpha");

    expect(drag.attempt).toBeLessThanOrEqual(2);
    expect(drag.markerContent).toBe('"Drop before"');
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-drop-position", "before");

    await page.mouse.up();
    try {
      await expect(page.locator("[data-content-block-id]").first())
        .toHaveAttribute("data-content-block-id", "beta");
    } catch (error) {
      const postReleaseTelemetry = await readTelemetry(page);
      throw new Error(`physical drop geometry=${JSON.stringify(drag.geometry)} pre-release telemetry=${JSON.stringify(drag.telemetry)} post-release telemetry=${JSON.stringify(postReleaseTelemetry)}: ${String(error)}`);
    }
    await expect(page.locator("[data-content-block-id='beta']"))
      .not.toHaveAttribute("data-content-editor-drag-preview");
    await expect(page.locator("[data-content-block-id='alpha']"))
      .not.toHaveAttribute("data-content-editor-drop-before");

    expect(await blockOrder(page)).toEqual(["beta", "alpha"]);
    const parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.blocks.map((block: { id?: string }) => block.id)).toEqual(["beta", "alpha"]);
    expect(parsed.blocks[0]).toMatchObject({ id: "beta", text: "Beta block", extra: { keep: "beta" } });
    expect(parsed.blocks[1]).toMatchObject({ id: "alpha", text: "Alpha block", extra: { keep: "alpha" } });

    const telemetry = await readTelemetry(page);
    expect(telemetry.some((event) => event.type === "dragstart" && event.targetID === "beta")).toBe(true);
    expect(telemetry.some((event) => event.type === "dragover" && event.targetID === "alpha")).toBe(true);
    expect(telemetry.some((event) => event.type === "drop" && event.targetID === "alpha")).toBe(true);
  });

  test("cancels a real mouse drag with Escape and outside release without changing source", async ({ page }) => {
    await mountFixture(page);
    const escapeDrag = await beginPhysicalDragWithRetry(page, "beta", "alpha");
    await page.keyboard.press("Escape");
    await page.mouse.up();

    await expect(page.locator("#body")).toHaveValue(escapeDrag.beforeSource);
    expect(await blockOrder(page)).toEqual(escapeDrag.beforeOrder);
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-drop-position");
    await expect(page.locator("[data-content-block-id='beta']"))
      .not.toHaveAttribute("data-content-editor-drag-preview");
    await expect(page.locator("[data-content-block-id='alpha']"))
      .not.toHaveAttribute("data-content-editor-drop-before");

    const escapeTelemetry = await readTelemetry(page);
    expect(escapeTelemetry.some((event) => event.type === "dragstart" && event.targetID === "beta")).toBe(true);
    expect(escapeTelemetry.some((event) => event.type === "dragend" && event.targetID === "beta")).toBe(true);
    expect(escapeTelemetry.some((event) => event.type === "drop" && event.targetID === "alpha")).toBe(false);

    await mountFixture(page);
    const outsideDrag = await beginPhysicalDragWithRetry(page, "beta", "alpha");
    const outside = page.locator("#outside-cancel-zone");
    const outsideBox = await outside.boundingBox();
    if (!outsideBox) throw new Error("outside cancel zone has no geometry");
    await page.mouse.move(outsideBox.x + outsideBox.width / 2, outsideBox.y + outsideBox.height / 2, { steps: 12 });
    await page.mouse.up();

    await expect(page.locator("#body")).toHaveValue(outsideDrag.beforeSource);
    expect(await blockOrder(page)).toEqual(outsideDrag.beforeOrder);
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-drop-position");
    await expect(page.locator("[data-content-block-id='beta']"))
      .not.toHaveAttribute("data-content-editor-drag-preview");
    await expect(page.locator("[data-content-block-id='alpha']"))
      .not.toHaveAttribute("data-content-editor-drop-before");

    const outsideTelemetry = await readTelemetry(page);
    expect(outsideTelemetry.some((event) => event.type === "dragstart" && event.targetID === "beta")).toBe(true);
    expect(outsideTelemetry.some((event) => event.type === "dragend" && event.targetID === "beta")).toBe(true);
    expect(outsideTelemetry.some((event) => event.type === "drop" && event.targetID === "alpha")).toBe(false);
  });
});
