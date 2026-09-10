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
  title: "Pointer content editor fixture",
  blocks: [
    { id: "alpha", type: "paragraph", text: "Alpha", extra: { keep: "alpha" } },
    { id: "beta", type: "heading", text: "Beta", level: "3", extra: { keep: "beta" } },
    { id: "gamma", type: "quote", text: "Gamma", extra: { keep: "gamma" } },
  ],
};

type PointerType = "touch" | "pen" | "mouse";
type PointerEventType = "pointerdown" | "pointermove" | "pointerup" | "pointercancel";
type PointerPoint = { pointerId: number; pointerType: PointerType; x: number; y: number; button?: number };
type FixtureInputMode = "synthetic-pointer" | "native-mouse";
type MountOptions = { scrollHeight?: number; inputMode?: FixtureInputMode };

type PointerWindow = Window & {
  __contentPointerSourceInputs?: number;
  __contentPointerRenders?: number;
};

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function editorHTML(): string {
  return `
    <!doctype html>
    <html>
      <head>
        <meta charset="utf-8">
        <style>${studioCSS}</style>
        <style>
          html, body { margin: 0; min-height: 100%; }
          body { background: var(--color-canvas); }
          .admin-page { width: min(100%, 720px); margin: 0 auto; padding: 16px; }
          .admin-form { display: grid; gap: 12px; }
          .content-editor-scroll {
            width: 360px;
            height: 220px;
            overflow: auto;
            border: 1px solid var(--color-line-strong);
            padding: 8px;
          }
          .content-block-list { min-width: 640px; }
          .content-block { min-height: 116px; }
          #outside { width: 140px; height: 80px; margin-top: 16px; border: 2px dashed var(--color-line-strong); }
        </style>
      </head>
      <body>
        <main class="admin-page">
          <form id="pointer-form" class="admin-form">
            <select id="pointer-format" name="bodyFormat" data-content-editor-format="true">
              <option value="blocks" selected>GoSX blocks</option>
              <option value="mdpp">MDPP</option>
            </select>
            <div class="content-editor" data-content-editor="true" data-content-editor-test-input-mode="synthetic-pointer">
              <label for="pointer-source">Body</label>
              <textarea id="pointer-source" name="body" data-content-editor-source="true">${escapeHTML(JSON.stringify(initialDocument, null, 2))}</textarea>
              <div class="content-editor__toolbar">
                <button type="button" data-content-add="paragraph">Paragraph</button>
                <button type="button" data-content-add="heading">Heading</button>
              </div>
              <div class="content-editor-scroll">
                <div class="content-block-list" data-content-editor-list="true"></div>
              </div>
            </div>
          </form>
          <aside id="outside">Outside the editor</aside>
        </main>
      </body>
    </html>
  `;
}

async function mount(page: Page, options: MountOptions = {}): Promise<void> {
  const scrollHeight = options.scrollHeight ?? 420;
  const inputMode = options.inputMode ?? "synthetic-pointer";
  await page.setViewportSize({ width: 900, height: 700 });
  await page.setContent(editorHTML());
  await page.locator(".content-editor-scroll").evaluate((node, height) => {
    node.setAttribute("data-content-editor-fixture-scroll-height", String(height));
    (node as HTMLElement).style.height = `${height}px`;
  }, scrollHeight);
  await page.addScriptTag({ content: runtimeJS });
  await expect(page.locator("[data-content-editor]"))
    .toHaveAttribute("data-content-editor-source-state", "ready");
  await page.evaluate((mode) => {
    const editor = document.querySelector("[data-content-editor]");
    const source = document.querySelector("[data-content-editor-source]");
    if (!editor || !source) throw new Error("pointer fixture did not mount");
    editor.setAttribute("data-content-editor-test-input-mode", mode);
    const browserWindow = window as PointerWindow;
    browserWindow.__contentPointerSourceInputs = 0;
    browserWindow.__contentPointerRenders = 0;
    source.addEventListener("input", () => {
      browserWindow.__contentPointerSourceInputs = (browserWindow.__contentPointerSourceInputs ?? 0) + 1;
    });
    editor.addEventListener("gosxstudio:content-editor-render", () => {
      browserWindow.__contentPointerRenders = (browserWindow.__contentPointerRenders ?? 0) + 1;
    });
  }, inputMode);
  await test.info().attach("content-editor-fixture", {
    body: JSON.stringify({ inputMode, scrollHeight }),
    contentType: "application/json",
  });
}

async function order(page: Page): Promise<string[]> {
  return page.locator("[data-content-block-id]").evaluateAll((rows) =>
    rows.map((row) => row.getAttribute("data-content-block-id") ?? ""));
}

async function sourceValue(page: Page): Promise<string> {
  return page.locator("#pointer-source").inputValue();
}

async function dispatchPointer(
  page: Page,
  selector: string,
  type: PointerEventType,
  point: PointerPoint,
): Promise<boolean> {
  return page.locator(selector).evaluate((target, payload) => {
    const event = new PointerEvent(payload.type, {
      bubbles: true,
      cancelable: true,
      pointerId: payload.pointerId,
      pointerType: payload.pointerType,
      clientX: payload.x,
      clientY: payload.y,
      button: payload.button ?? 0,
    });
    target.dispatchEvent(event);
    return event.defaultPrevented;
  }, { type, ...point });
}

async function dispatchDocumentPointer(
  page: Page,
  type: Exclude<PointerEventType, "pointerdown">,
  point: PointerPoint,
): Promise<boolean> {
  return page.evaluate(({ type, point }) => {
    const event = new PointerEvent(type, {
      bubbles: true,
      cancelable: true,
      pointerId: point.pointerId,
      pointerType: point.pointerType,
      clientX: point.x,
      clientY: point.y,
      button: point.button ?? 0,
    });
    document.dispatchEvent(event);
    return event.defaultPrevented;
  }, { type, point });
}

async function dispatchLostPointerCapture(page: Page, selector: string, pointerId: number): Promise<void> {
  await page.locator(selector).evaluate((target, id) => {
    target.dispatchEvent(new PointerEvent("lostpointercapture", {
      bubbles: false,
      cancelable: false,
      pointerId: id,
      pointerType: "touch",
    }));
  }, pointerId);
}

async function rowPoint(page: Page, id: string, edge: "top" | "middle" | "bottom"): Promise<PointerPoint> {
  const row = page.locator(`[data-content-block-id='${id}']`);
  const handle = row.locator("[data-content-drag-handle]");
  const rowBox = await row.boundingBox();
  const handleBox = await handle.boundingBox();
  if (!rowBox || !handleBox) throw new Error(`missing pointer geometry for ${id}`);
  const y = edge === "top"
    ? rowBox.y + 4
    : edge === "bottom"
      ? rowBox.y + rowBox.height - 4
      : rowBox.y + rowBox.height / 2;
  return {
    pointerId: 1,
    pointerType: "touch",
    x: handleBox.x + handleBox.width / 2,
    y,
  };
}

async function beginPointerReorder(
  page: Page,
  sourceID: string,
  targetID: string,
  pointerType: PointerType,
  pointerId = 1,
  edge: "top" | "middle" | "bottom" = "top",
): Promise<{ beforeSource: string; source: PointerPoint; target: PointerPoint }> {
  const source = await rowPoint(page, sourceID, "middle");
  const target = await rowPoint(page, targetID, edge);
  source.pointerId = pointerId;
  source.pointerType = pointerType;
  target.pointerId = pointerId;
  target.pointerType = pointerType;
  const beforeSource = await sourceValue(page);
  await dispatchPointer(page, `[data-content-block-id='${sourceID}'] [data-content-drag-handle]`, "pointerdown", source);
  await dispatchDocumentPointer(page, "pointermove", {
    ...target,
    x: target.x + 2,
    y: target.y + 2,
  });
  return { beforeSource, source, target };
}

async function pointerCounts(page: Page): Promise<{ inputs: number; renders: number }> {
  return page.evaluate(() => {
    const browserWindow = window as PointerWindow;
    return {
      inputs: browserWindow.__contentPointerSourceInputs ?? 0,
      renders: browserWindow.__contentPointerRenders ?? 0,
    };
  });
}

test.describe("@smoke shared content editor touch/pointer reorder", () => {
  test("commits touch, pen, and synthetic mouse drags once with stable IDs, indicator state, undo, and focus", async ({ page }) => {
    for (const [pointerType, pointerId] of [["touch", 11], ["pen", 12], ["mouse", 13]] as const) {
      await mount(page);
      const drag = await beginPointerReorder(page, "gamma", "alpha", pointerType, pointerId, "top");

      await expect(page.locator("[data-content-editor]"))
        .toHaveAttribute("data-content-editor-pointer-dragging", "true");
      await expect(page.locator("[data-content-editor]"))
        .toHaveAttribute("data-content-editor-pointer-type", pointerType);
      await expect(page.locator("[data-content-block-id='gamma']"))
        .toHaveAttribute("data-content-editor-drag-preview", "true");
      await expect(page.locator("[data-content-block-id='alpha']"))
        .toHaveAttribute("data-content-editor-drop-before", "true");
      await expect(page.locator("[data-content-editor]"))
        .toHaveAttribute("data-content-editor-drop-position", "before");
      expect(await sourceValue(page)).toBe(drag.beforeSource);
      expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
      expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });

      await dispatchDocumentPointer(page, "pointerup", drag.target);
      expect(await order(page)).toEqual(["gamma", "alpha", "beta"]);
      const parsed = JSON.parse(await sourceValue(page)) as typeof initialDocument;
      expect(parsed.blocks.map((block) => block.id)).toEqual(["gamma", "alpha", "beta"]);
      expect(parsed.blocks[0]).toMatchObject({ id: "gamma", extra: { keep: "gamma" } });
      expect(parsed.blocks[1]).toMatchObject({ id: "alpha", extra: { keep: "alpha" } });
      expect(await pointerCounts(page)).toEqual({ inputs: 1, renders: 1 });
      await expect(page.locator("[data-content-editor]"))
        .not.toHaveAttribute("data-content-editor-pointer-dragging");
      await expect(page.locator("[data-content-editor]"))
        .not.toHaveAttribute("data-content-editor-drop-position");
      await expect(page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")).toBeFocused();
      await expect(page.locator("[data-content-editor]"))
        .toHaveAttribute("data-content-editor-history-can-undo", "true");

      await page.locator("[data-content-editor-action='undo']").click();
      expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
      await expect(page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")).toBeFocused();
      expect(JSON.parse(await sourceValue(page)).blocks.map((block: { id: string }) => block.id))
        .toEqual(["alpha", "beta", "gamma"]);
    }
  });

  test("exposes the after-drop indicator for a valid touch target", async ({ page }) => {
    await mount(page);
    const scroll = page.locator(".content-editor-scroll");
    const targetRow = page.locator("[data-content-block-id='beta']");
    await targetRow.evaluate((node) => {
      node.scrollIntoView({ block: "center", inline: "nearest" });
    });
    const scrollBox = await scroll.boundingBox();
    const targetBox = await targetRow.boundingBox();
    if (!scrollBox || !targetBox) throw new Error("after-drop fixture has no geometry");
    expect(targetBox.y).toBeGreaterThanOrEqual(scrollBox.y);
    expect(targetBox.y + targetBox.height).toBeLessThanOrEqual(scrollBox.y + scrollBox.height);
    const dropY = targetBox.y + targetBox.height - 4;
    expect(dropY).toBeGreaterThan(scrollBox.y + 40);
    expect(dropY).toBeLessThan(scrollBox.y + scrollBox.height - 42);
    const drag = await beginPointerReorder(page, "alpha", "beta", "touch", 14, "bottom");
    await expect(page.locator("[data-content-block-id='beta']"))
      .toHaveAttribute("data-content-editor-drop-after", "true");
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-drop-position", "after");
    await dispatchDocumentPointer(page, "pointercancel", drag.target);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    expect(await sourceValue(page)).toBe(drag.beforeSource);
  });

  test("cancels Escape, pointercancel, invalid/outside release, blur, and no-op without draft/order/history changes", async ({ page }) => {
    await mount(page);
    const before = await sourceValue(page);

    const escape = await beginPointerReorder(page, "gamma", "alpha", "touch", 21, "top");
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-pointer-dragging", "true");
    await page.keyboard.press("Escape");
    await dispatchDocumentPointer(page, "pointerup", escape.target);
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-pointer-dragging");
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-drop-position");
    await expect(page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")).toBeFocused();

    const wrongCancel = await beginPointerReorder(page, "gamma", "alpha", "pen", 22, "top");
    await dispatchDocumentPointer(page, "pointercancel", { ...wrongCancel.target, pointerId: 999 });
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-pointer-dragging", "true");
    await dispatchDocumentPointer(page, "pointercancel", wrongCancel.target);
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    await expect(page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")).toBeFocused();

    const lostCapture = await beginPointerReorder(page, "gamma", "alpha", "touch", 27, "top");
    await dispatchLostPointerCapture(page, "[data-content-block-id='gamma'] [data-content-drag-handle]", 27);
    await dispatchDocumentPointer(page, "pointerup", lostCapture.target);
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    await expect(page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")).toBeFocused();

    const outside = await beginPointerReorder(page, "gamma", "alpha", "touch", 23, "top");
    const outsideBox = await page.locator("#outside").boundingBox();
    if (!outsideBox) throw new Error("outside pointer cancel zone has no geometry");
    await dispatchDocumentPointer(page, "pointerup", {
      ...outside.target,
      x: outsideBox.x + outsideBox.width / 2,
      y: outsideBox.y + outsideBox.height / 2,
    });
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-drop-position");
    await expect(page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")).toBeFocused();

    const invalid = await beginPointerReorder(page, "gamma", "alpha", "pen", 24, "top");
    await dispatchDocumentPointer(page, "pointerup", { ...invalid.target, x: 1, y: 1 });
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);

    const blurred = await beginPointerReorder(page, "gamma", "alpha", "touch", 25, "top");
    await page.evaluate(() => window.dispatchEvent(new Event("blur")));
    await dispatchDocumentPointer(page, "pointerup", blurred.target);
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);

    // Gamma before Beta is the current position after removing gamma from the
    // end only when the target is the same row; a same-row release is always a
    // no-op and must not create an undo entry.
    const noOp = await beginPointerReorder(page, "beta", "beta", "touch", 26, "middle");
    await dispatchDocumentPointer(page, "pointerup", noOp.target);
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-history-can-undo", "false");
    await expect(page.locator("[data-content-block-id='beta'] [data-content-drag-handle]")).toBeFocused();
    expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });
  });

  test("does not steal focus when a pointer drag is canceled by focus departure", async ({ page }) => {
    for (const departure of ["window", "visibility", "handle"] as const) {
      await mount(page);
      const outside = page.locator("#outside");
      await outside.evaluate((node) => {
        (node as HTMLElement).tabIndex = -1;
        (node as HTMLElement).focus();
      });
      await expect(outside).toBeFocused();

      const beforeSource = await sourceValue(page);
      const beforeOrder = await order(page);
      const drag = await beginPointerReorder(page, "gamma", "alpha", "touch", 60, "top");
      await expect(page.locator("[data-content-editor]"))
        .toHaveAttribute("data-content-editor-pointer-dragging", "true");

      await page.evaluate((kind) => {
        const outsideNode = document.querySelector("#outside") as HTMLElement | null;
        if (!outsideNode) throw new Error("focus departure target is missing");
        outsideNode.focus();
        if (kind === "window") {
          window.dispatchEvent(new Event("blur"));
        } else if (kind === "visibility") {
          document.dispatchEvent(new Event("visibilitychange"));
        } else {
          const handle = document.querySelector("[data-content-block-id='gamma'] [data-content-drag-handle]");
          if (!handle) throw new Error("focus departure handle is missing");
          // Model the browser's already-completed focus transition while
          // keeping focus on the outside control, so refocus theft is visible.
          handle.dispatchEvent(new FocusEvent("blur"));
        }
      }, departure);

      await expect(page.locator("[data-content-editor]"))
        .not.toHaveAttribute("data-content-editor-pointer-dragging");
      await expect(outside).toBeFocused();
      expect(await page.evaluate(() => document.activeElement?.id ?? "")).toBe("outside");
      expect(await sourceValue(page)).toBe(beforeSource);
      expect(await order(page)).toEqual(beforeOrder);
      expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });
      await expect(page.locator("[data-content-editor]"))
        .toHaveAttribute("data-content-editor-history-can-undo", "false");
      await dispatchDocumentPointer(page, "pointerup", drag.target);
      await expect(outside).toBeFocused();
    }
  });

  test("does not hijack row scrolling, keeps handle touch-action scoped, honors reduced motion, and keeps move buttons available", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await mount(page, { scrollHeight: 220 });
    const editor = page.locator("[data-content-editor]");
    const scroll = page.locator(".content-editor-scroll");
    const handle = page.locator("[data-content-block-id='beta'] [data-content-drag-handle]");
    await expect(handle).toHaveCSS("touch-action", "none");
    await expect(page.locator("[data-content-block-id='beta']")).toHaveCSS("transition-duration", "0s");
    const moveUp = page.locator("[data-content-block-id='beta'] [data-content-editor-action='move-up']");
    const handleBox = await handle.boundingBox();
    const moveUpBox = await moveUp.boundingBox();
    if (!handleBox || !moveUpBox) throw new Error("pointer targets have no geometry");
    expect(handleBox.width).toBeGreaterThanOrEqual(24);
    expect(handleBox.height).toBeGreaterThanOrEqual(24);
    expect(moveUpBox.width).toBeGreaterThanOrEqual(24);
    expect(moveUpBox.height).toBeGreaterThanOrEqual(24);

    const scrollBox = await scroll.boundingBox();
    if (!scrollBox) throw new Error("scroll fixture has no geometry");
    const edgeDrag = await beginPointerReorder(page, "alpha", "gamma", "touch", 30, "bottom");
    await dispatchDocumentPointer(page, "pointermove", {
      ...edgeDrag.target,
      x: scrollBox.x + scrollBox.width - 4,
      y: scrollBox.y + scrollBox.height - 4,
    });
    await expect.poll(() => scroll.evaluate((node) => node.scrollTop)).toBeGreaterThan(0);
    await expect(editor).toHaveAttribute("data-content-editor-pointer-reduced-motion", "true");
    await dispatchDocumentPointer(page, "pointercancel", edgeDrag.target);
    await expect(editor).not.toHaveAttribute("data-content-editor-pointer-dragging");
    await scroll.evaluate((node) => { node.scrollTop = 0; node.scrollLeft = 0; });

    const xOutsideYEdge = await beginPointerReorder(page, "alpha", "beta", "touch", 33, "middle");
    await dispatchDocumentPointer(page, "pointermove", {
      ...xOutsideYEdge.target,
      x: scrollBox.x + scrollBox.width + 240,
      y: scrollBox.y + scrollBox.height - 4,
    });
    await page.waitForTimeout(60);
    expect(await scroll.evaluate((node) => node.scrollTop)).toBe(0);
    await dispatchDocumentPointer(page, "pointercancel", xOutsideYEdge.target);

    const yOutsideXEdge = await beginPointerReorder(page, "alpha", "beta", "pen", 34, "middle");
    await dispatchDocumentPointer(page, "pointermove", {
      ...yOutsideXEdge.target,
      x: scrollBox.x + scrollBox.width - 4,
      y: scrollBox.y + scrollBox.height + 240,
    });
    await page.waitForTimeout(60);
    expect(await scroll.evaluate((node) => node.scrollLeft)).toBe(0);
    await dispatchDocumentPointer(page, "pointercancel", yOutsideXEdge.target);

    const horizontalDrag = await beginPointerReorder(page, "alpha", "beta", "pen", 31, "middle");
    await dispatchDocumentPointer(page, "pointermove", {
      ...horizontalDrag.target,
      x: scrollBox.x + scrollBox.width - 4,
      y: scrollBox.y + scrollBox.height / 2,
    });
    await expect.poll(() => scroll.evaluate((node) => node.scrollLeft)).toBeGreaterThan(0);
    await dispatchDocumentPointer(page, "pointercancel", horizontalDrag.target);

    const outsideScroll = await beginPointerReorder(page, "alpha", "beta", "touch", 32, "middle");
    await dispatchDocumentPointer(page, "pointermove", {
      ...outsideScroll.target,
      x: scrollBox.x + scrollBox.width - 4,
      y: scrollBox.y + scrollBox.height - 4,
    });
    await dispatchDocumentPointer(page, "pointermove", {
      ...outsideScroll.target,
      x: scrollBox.x + scrollBox.width + 240,
      y: scrollBox.y + scrollBox.height + 240,
    });
    const stoppedScrollTop = await scroll.evaluate((node) => node.scrollTop);
    await page.waitForTimeout(60);
    expect(await scroll.evaluate((node) => node.scrollTop)).toBe(stoppedScrollTop);
    await dispatchDocumentPointer(page, "pointerup", {
      ...outsideScroll.target,
      x: scrollBox.x + scrollBox.width + 240,
      y: scrollBox.y + scrollBox.height + 240,
    });
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });

    await page.evaluate(() => {
      const scrollNode = document.querySelector(".content-editor-scroll");
      if (!scrollNode) throw new Error("scroll fixture missing");
      scrollNode.addEventListener("pointermove", (event) => {
        const browserWindow = window as PointerWindow & { __unhandledPointerDefaults?: boolean[] };
        browserWindow.__unhandledPointerDefaults ??= [];
        browserWindow.__unhandledPointerDefaults.push(event.defaultPrevented);
      }, true);
    });
    const body = page.locator("[data-content-block-id='beta'] .content-block__body");
    const bodyBox = await body.boundingBox();
    if (!bodyBox) throw new Error("content block body has no geometry");
    const touchBody = { pointerId: 31, pointerType: "touch" as const, x: bodyBox.x + 100, y: bodyBox.y + 40 };
    expect(await dispatchPointer(page, "[data-content-block-id='beta'] .content-block__body", "pointerdown", touchBody)).toBe(false);
    expect(await dispatchPointer(page, ".content-editor-scroll", "pointermove", { ...touchBody, x: touchBody.x + 180, y: touchBody.y + 80 })).toBe(false);
    expect(await dispatchPointer(page, ".content-editor-scroll", "pointerup", { ...touchBody, x: touchBody.x + 180, y: touchBody.y + 80 })).toBe(false);
    await expect(editor).not.toHaveAttribute("data-content-editor-pointer-dragging");
    expect(await page.evaluate(() => (window as PointerWindow & { __unhandledPointerDefaults?: boolean[] }).__unhandledPointerDefaults ?? []))
      .toEqual([false]);

    // The explicit tap alternative remains usable under the same reduced
    // motion media preference and produces one ordinary undoable mutation.
    await moveUp.click();
    expect(await order(page)).toEqual(["beta", "alpha", "gamma"]);
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-history-can-undo", "true");
    await expect(page.locator("[data-content-block-id='beta'] [data-content-drag-handle]")).toBeFocused();
    await expect(scroll).toBeVisible();
  });

  test("does not resurrect a removed source row or mutate a replacement owner after a stale pointer release", async ({ page }) => {
    await mount(page);
    const before = await sourceValue(page);
    const drag = await beginPointerReorder(page, "gamma", "alpha", "pen", 41, "top");
    await page.evaluate(() => {
      const root = document.querySelector("[data-content-editor]");
      if (!root) throw new Error("editor root missing");
      (window as Window & { __removedEditorRoot?: Element }).__removedEditorRoot = root;
      root.remove();
    });
    await page.waitForTimeout(0);
    await page.evaluate(() => {
      const root = (window as Window & { __removedEditorRoot?: Element }).__removedEditorRoot;
      const form = document.querySelector("#pointer-form");
      if (!root || !form) throw new Error("removed editor fixture missing");
      form.appendChild(root);
    });
    await dispatchDocumentPointer(page, "pointerup", drag.target);
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-pointer-dragging");
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-drop-position");

    const rowRemoval = await beginPointerReorder(page, "gamma", "alpha", "pen", 42, "top");
    await page.locator("[data-content-block-id='gamma']").evaluate((row) => row.remove());
    await page.waitForTimeout(0);
    await dispatchDocumentPointer(page, "pointerup", rowRemoval.target);
    expect(await sourceValue(page)).toBe(before);
    expect(await order(page)).toEqual(["alpha", "beta"]);
    expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });
  });

  test("cancels when a same-ID source row or handle node is replaced during a pointer drag", async ({ page }) => {
    await mount(page);
    const beforeRowReplacement = await sourceValue(page);
    const rowDrag = await beginPointerReorder(page, "gamma", "alpha", "pen", 43, "top");
    await page.locator("[data-content-block-id='gamma']").evaluate((row) => {
      row.replaceWith(row.cloneNode(true));
    });
    await page.waitForTimeout(0);
    await dispatchDocumentPointer(page, "pointerup", rowDrag.target);
    expect(await sourceValue(page)).toBe(beforeRowReplacement);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-pointer-dragging");

    await mount(page);
    const beforeHandleReplacement = await sourceValue(page);
    const handleDrag = await beginPointerReorder(page, "gamma", "alpha", "touch", 44, "top");
    await page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]").evaluate((handle) => {
      handle.replaceWith(handle.cloneNode(true));
    });
    await page.waitForTimeout(0);
    await dispatchDocumentPointer(page, "pointerup", handleDrag.target);
    expect(await sourceValue(page)).toBe(beforeHandleReplacement);
    expect(await order(page)).toEqual(["alpha", "beta", "gamma"]);
    expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-pointer-dragging");
  });

  test("keeps a synthetic mouse pointer fallback from doubling with a dispatched native-style dragstart", async ({ page }) => {
    await mount(page, { inputMode: "synthetic-pointer" });
    const before = await sourceValue(page);
    const drag = await beginPointerReorder(page, "gamma", "alpha", "mouse", 51, "top");
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-pointer-dragging", "true");

    const transfer = await page.evaluateHandle(() => new DataTransfer());
    await page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")
      .dispatchEvent("dragstart", { dataTransfer: transfer });
    await expect(page.locator("[data-content-editor]"))
      .not.toHaveAttribute("data-content-editor-pointer-dragging");
    expect(await pointerCounts(page)).toEqual({ inputs: 0, renders: 0 });

    await page.locator("[data-content-block-id='alpha']").dispatchEvent("dragover", {
      clientY: drag.target.y,
      dataTransfer: transfer,
    });
    await page.locator("[data-content-block-id='alpha']").dispatchEvent("drop", {
      clientY: drag.target.y,
      dataTransfer: transfer,
    });
    await page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]").dispatchEvent("dragend");
    expect(await sourceValue(page)).not.toBe(before);
    expect(await order(page)).toEqual(["gamma", "alpha", "beta"]);
    expect(await pointerCounts(page)).toEqual({ inputs: 1, renders: 1 });
    await expect(page.locator("[data-content-block-id='gamma'] [data-content-drag-handle]")).toBeFocused();
  });

  test("completes a browser-native mouse drag with source and target visible in the scroll pane", async ({ page }) => {
    await mount(page, { inputMode: "native-mouse" });
    const editor = page.locator("[data-content-editor]");
    const scroll = page.locator(".content-editor-scroll");
    const sourceHandle = page.locator("[data-content-block-id='beta'] [data-content-drag-handle]");
    const targetRow = page.locator("[data-content-block-id='alpha']");
    await expect(editor).toHaveAttribute("data-content-editor-test-input-mode", "native-mouse");
    await expect(scroll).toHaveAttribute("data-content-editor-fixture-scroll-height", "420");

    const scrollBox = await scroll.boundingBox();
    const sourceBox = await sourceHandle.boundingBox();
    const targetBox = await targetRow.boundingBox();
    if (!scrollBox || !sourceBox || !targetBox) throw new Error("native drag fixture has no geometry");
    expect(sourceBox.y).toBeGreaterThanOrEqual(scrollBox.y);
    expect(sourceBox.y + sourceBox.height).toBeLessThanOrEqual(scrollBox.y + scrollBox.height);
    expect(targetBox.y).toBeGreaterThanOrEqual(scrollBox.y);
    expect(targetBox.y + targetBox.height).toBeLessThanOrEqual(scrollBox.y + scrollBox.height);

    const before = await sourceValue(page);
    await sourceHandle.hover();
    await page.mouse.move(sourceBox.x + sourceBox.width / 2, sourceBox.y + sourceBox.height / 2);
    await page.mouse.down();
    await page.mouse.move(targetBox.x + targetBox.width / 2, targetBox.y + 4, { steps: 12 });
    await page.mouse.up();

    expect(await sourceValue(page)).not.toBe(before);
    expect(await order(page)).toEqual(["beta", "alpha", "gamma"]);
    expect(await pointerCounts(page)).toEqual({ inputs: 1, renders: 1 });
    await expect(editor).not.toHaveAttribute("data-content-editor-pointer-dragging");
    await expect(page.locator("[data-content-block-id='beta'] [data-content-drag-handle]")).toBeFocused();
  });
});
