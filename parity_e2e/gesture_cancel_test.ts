import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const blockLayoutRuntimeJS = readFileSync(
  path.resolve(__dirname, "../blocklayoutruntime/island_runtime.js"),
  "utf8",
);
const siteMapRuntimeJS = readFileSync(
  path.resolve(__dirname, "../sitemapruntime/island_runtime.js"),
  "utf8",
);

type PointerOptions = {
  pointerId: number;
  clientX: number;
  clientY: number;
  button?: number;
  shiftKey?: boolean;
};

async function dispatchPointer(
  page: import("@playwright/test").Page,
  selector: string,
  type: "pointerdown" | "pointermove" | "pointerup" | "pointercancel",
  options: PointerOptions,
): Promise<void> {
  await page.locator(selector).evaluate((target, payload) => {
    target.dispatchEvent(new PointerEvent(payload.type, {
      bubbles: true,
      cancelable: true,
      pointerId: payload.pointerId,
      clientX: payload.clientX,
      clientY: payload.clientY,
      button: payload.button,
      shiftKey: payload.shiftKey,
      pointerType: "mouse",
    }));
  }, { type, ...options });
}

async function dispatchDocumentPointer(
  page: import("@playwright/test").Page,
  type: "pointermove" | "pointerup" | "pointercancel",
  options: PointerOptions,
): Promise<void> {
  await page.evaluate(({ type, options }) => {
    document.dispatchEvent(new PointerEvent(type, {
      bubbles: true,
      cancelable: true,
      pointerId: options.pointerId,
      clientX: options.clientX,
      clientY: options.clientY,
      button: options.button,
      shiftKey: options.shiftKey,
      pointerType: "mouse",
    }));
  }, { type, options });
}

const blockLayoutFixture = `
  <style>
    html, body { margin: 0; padding: 0; }
    #block-list { width: 320px; }
    [data-block-studio-block] {
      display: block;
      box-sizing: border-box;
      width: 320px;
      height: 72px;
      border: 1px solid #888;
      padding: 12px;
    }
  </style>
  <main>
    <div id="block-list">
      <div data-block-studio-block="alpha">
        <button type="button" data-block-studio-handle aria-label="Drag Alpha">Drag</button>
        <input data-block-studio-order value="1" />
      </div>
      <div data-block-studio-block="beta">
        <button type="button" data-block-studio-handle aria-label="Drag Beta">Drag</button>
        <input data-block-studio-order value="2" />
      </div>
      <div data-block-studio-block="gamma">
        <button type="button" data-block-studio-handle aria-label="Drag Gamma">Drag</button>
        <input data-block-studio-order value="3" />
      </div>
    </div>
  </main>
`;

const siteMapFixture = `
  <style>
    html, body { margin: 0; padding: 0; }
    #board { width: 640px; }
    [data-studio-site-map-canvas] {
      position: relative;
      width: 600px;
      height: 400px;
      overflow: hidden;
      border: 1px solid #888;
    }
    [data-studio-site-map-pan-surface] { position: absolute; inset: 0; }
    [data-studio-site-map-workspace-node] {
      display: block;
      position: absolute;
      width: 88px;
      height: 48px;
      border: 1px solid #888;
      box-sizing: border-box;
    }
  </style>
  <section id="board"
    data-studio-site-map-board="true"
    tabindex="0"
    data-studio-site-map-filter="all"
    data-studio-site-map-detail="map"
    data-studio-site-map-palette="all"
    data-studio-site-map-focus="all"
    data-studio-site-map-scale="1"
    data-studio-site-map-pan-x="10"
    data-studio-site-map-pan-y="20"
    data-studio-site-map-selected-node="beta"
    data-studio-site-map-selected-nodes="beta">
    <button id="toolbar" type="button" data-studio-site-map-detail-target="build">Toolbar</button>
    <div data-studio-site-map-workspace="true" data-studio-site-map-panel-visible="true">
      <div id="canvas" data-studio-site-map-canvas="true">
        <div data-studio-site-map-pan-surface="true">
          <label id="alpha" data-studio-site-map-workspace-node="alpha"
            data-studio-site-map-node-x="40" data-studio-site-map-node-y="40"
            style="left:40px;top:40px;">Alpha</label>
          <label id="beta" data-studio-site-map-workspace-node="beta"
            data-studio-site-map-node-x="240" data-studio-site-map-node-y="180"
            style="left:240px;top:180px;">Beta</label>
        </div>
      </div>
    </div>
  </section>
`;

async function blockSnapshot(page: import("@playwright/test").Page): Promise<unknown> {
  return page.evaluate(() => {
    const list = document.querySelector("#block-list");
    if (!list) throw new Error("block fixture list missing");
    return {
      keys: Array.from(list.querySelectorAll("[data-block-studio-block]"), (row) => row.getAttribute("data-block-studio-block")),
      orders: Array.from(list.querySelectorAll("[data-block-studio-block]"), (row) => (row.querySelector("[data-block-studio-order]") as HTMLInputElement | null)?.value ?? ""),
      indexes: Array.from(list.querySelectorAll("[data-block-studio-block]"), (row) => row.getAttribute("data-block-studio-index")),
      dragging: Array.from(list.querySelectorAll("[data-block-studio-block]"), (row) => row.classList.contains("is-dragging")),
    };
  });
}

async function blockEventCounts(page: import("@playwright/test").Page): Promise<Record<string, number>> {
  return page.evaluate(() => ({
    ...(window as unknown as { __blockGestureEvents?: Record<string, number> }).__blockGestureEvents,
  }));
}

async function installBlockEventCounters(page: import("@playwright/test").Page): Promise<void> {
  await page.evaluate(() => {
    const events = { reorder: 0, input: 0, change: 0, select: 0 };
    const list = document.querySelector("#block-list");
    if (!list) throw new Error("block fixture list missing");
    list.addEventListener("blockstudio:reorder", () => { events.reorder += 1; });
    list.addEventListener("input", () => { events.input += 1; });
    list.addEventListener("change", () => { events.change += 1; });
    document.addEventListener("blockstudio:select", () => { events.select += 1; });
    (window as unknown as { __blockGestureEvents: Record<string, number> }).__blockGestureEvents = events;
  });
}

async function bootBlockFixture(page: import("@playwright/test").Page): Promise<void> {
  await page.setContent(blockLayoutFixture);
  await page.addScriptTag({ content: blockLayoutRuntimeJS });
  await page.evaluate(() => {
    const w = window as unknown as Record<string, (root: Document) => void>;
    w.__gosx_blocklayout_runtime_island_bindList(document);
    w.__gosx_blocklayout_runtime_island_bindHandleDrag(document);
  });
  await installBlockEventCounters(page);
}

async function siteMapSnapshot(page: import("@playwright/test").Page): Promise<unknown> {
  return page.evaluate(() => {
    const board = document.querySelector("#board");
    const alpha = document.querySelector("#alpha") as HTMLElement | null;
    if (!board || !alpha) throw new Error("site-map fixture missing");
    return {
      panX: board.getAttribute("data-studio-site-map-pan-x"),
      panY: board.getAttribute("data-studio-site-map-pan-y"),
      detail: board.getAttribute("data-studio-site-map-detail"),
      selectedNode: board.getAttribute("data-studio-site-map-selected-node"),
      selectedNodes: board.getAttribute("data-studio-site-map-selected-nodes"),
      nodePosition: alpha.style.position,
      nodeLeft: alpha.style.left,
      nodeTop: alpha.style.top,
      nodeX: alpha.getAttribute("data-studio-site-map-node-x"),
      nodeY: alpha.getAttribute("data-studio-site-map-node-y"),
      nodeDragging: board.getAttribute("data-studio-site-map-node-dragging"),
      panning: board.getAttribute("data-studio-site-map-panning"),
      marqueeing: board.getAttribute("data-studio-site-map-marqueeing"),
      marqueeCount: board.querySelectorAll("[data-studio-site-map-marquee]").length,
    };
  });
}

async function installSiteMapMoveCounter(page: import("@playwright/test").Page): Promise<void> {
  await page.evaluate(() => {
    let count = 0;
    const board = document.querySelector("#board");
    if (!board) throw new Error("site-map fixture board missing");
    board.addEventListener("gosxstudio:sitemap-node-moved", () => { count += 1; });
    (window as unknown as { __siteMapMoveCount: () => number }).__siteMapMoveCount = () => count;
  });
}

test.describe("gesture cancellation contracts", () => {
  test("BlockLayout restores a canceled reorder, ignores the wrong pointer, and commits the next real drop once", async ({ page }) => {
    await bootBlockFixture(page);
    const before = await blockSnapshot(page);
    const gammaBox = await page.locator("[data-block-studio-block='gamma']").boundingBox();
    if (!gammaBox) throw new Error("gamma row has no geometry");

    // Secondary mouse buttons must never start a reorder gesture.
    for (const button of [1, 2]) {
      await dispatchPointer(page, "[data-block-studio-block='alpha'] [data-block-studio-handle]", "pointerdown", {
        pointerId: 5 + button, clientX: 20, clientY: 20, button,
      });
      await dispatchPointer(page, "#block-list", "pointermove", {
        pointerId: 5 + button, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
      });
      await dispatchPointer(page, "#block-list", "pointerup", {
        pointerId: 5 + button, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
      });
    }
    expect(await blockSnapshot(page)).toEqual(before);
    expect(await blockEventCounts(page)).toEqual({ reorder: 0, input: 0, change: 0, select: 0 });

    await dispatchPointer(page, "[data-block-studio-block='alpha'] [data-block-studio-handle]", "pointerdown", {
      pointerId: 7, clientX: 20, clientY: 20,
    });
    await dispatchPointer(page, "#block-list", "pointermove", {
      pointerId: 7, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
    });
    expect(await blockSnapshot(page)).toMatchObject({ keys: ["beta", "gamma", "alpha"] });

    await dispatchPointer(page, "#block-list", "pointercancel", {
      pointerId: 8, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
    });
    expect(await blockSnapshot(page)).toMatchObject({ keys: ["beta", "gamma", "alpha"], dragging: [false, false, true] });
    await dispatchPointer(page, "#block-list", "pointercancel", {
      pointerId: 7, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
    });
    expect(await blockSnapshot(page)).toEqual(before);
    expect(await blockEventCounts(page)).toEqual({ reorder: 0, input: 0, change: 0, select: 0 });

    // A stale document/list event after cancellation must not keep moving the
    // old row, and a no-op press/release must not report a reorder.
    await dispatchPointer(page, "#block-list", "pointermove", {
      pointerId: 7, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
    });
    await dispatchPointer(page, "[data-block-studio-block='alpha'] [data-block-studio-handle]", "pointerdown", {
      pointerId: 9, clientX: 20, clientY: 20,
    });
    await dispatchPointer(page, "#block-list", "pointerup", {
      pointerId: 9, clientX: 20, clientY: 20,
    });
    expect(await blockSnapshot(page)).toEqual(before);
    expect(await blockEventCounts(page)).toEqual({ reorder: 0, input: 0, change: 0, select: 0 });

    const gammaBoxAgain = await page.locator("[data-block-studio-block='gamma']").boundingBox();
    if (!gammaBoxAgain) throw new Error("gamma row has no geometry after cancel");
    await dispatchPointer(page, "[data-block-studio-block='alpha'] [data-block-studio-handle]", "pointerdown", {
      pointerId: 10, clientX: 20, clientY: 20,
    });
    await dispatchPointer(page, "#block-list", "pointermove", {
      pointerId: 10, clientX: 20, clientY: gammaBoxAgain.y + gammaBoxAgain.height - 4,
    });
    await dispatchPointer(page, "#block-list", "pointerup", {
      pointerId: 10, clientX: 20, clientY: gammaBoxAgain.y + gammaBoxAgain.height - 4,
    });
    expect(await blockSnapshot(page)).toMatchObject({
      keys: ["beta", "gamma", "alpha"],
      orders: ["1", "2", "3"],
      indexes: ["0", "1", "2"],
      dragging: [false, false, false],
    });
    expect(await blockEventCounts(page)).toEqual({ reorder: 1, input: 1, change: 1, select: 1 });
  });

  test("BlockLayout real mouse Escape restores a moved row without a commit", async ({ page }) => {
    await bootBlockFixture(page);
    const before = await blockSnapshot(page);
    const alphaHandleBox = await page.locator("[data-block-studio-block='alpha'] [data-block-studio-handle]").boundingBox();
    const betaBox = await page.locator("[data-block-studio-block='beta']").boundingBox();
    if (!alphaHandleBox || !betaBox) throw new Error("block fixture has no geometry");
    await page.mouse.move(alphaHandleBox.x + alphaHandleBox.width / 2, alphaHandleBox.y + alphaHandleBox.height / 2);
    await page.mouse.down();
    await page.mouse.move(betaBox.x + betaBox.width / 2, betaBox.y + betaBox.height - 4);
    expect(await blockSnapshot(page)).toMatchObject({ keys: ["beta", "alpha", "gamma"] });
    await page.keyboard.press("Escape");
    await page.mouse.up();
    expect(await blockSnapshot(page)).toEqual(before);
    expect(await blockEventCounts(page)).toEqual({ reorder: 0, input: 0, change: 0, select: 0 });

    // The same browser mouse path remains usable after Escape cancellation;
    // this real drop must commit exactly once.
    const alphaHandleAgain = await page.locator("[data-block-studio-block='alpha'] [data-block-studio-handle]").boundingBox();
    const betaAgain = await page.locator("[data-block-studio-block='beta']").boundingBox();
    if (!alphaHandleAgain || !betaAgain) throw new Error("block fixture lost geometry after Escape");
    await page.mouse.move(alphaHandleAgain.x + alphaHandleAgain.width / 2, alphaHandleAgain.y + alphaHandleAgain.height / 2);
    await page.mouse.down();
    await page.mouse.move(betaAgain.x + betaAgain.width / 2, betaAgain.y + betaAgain.height - 4);
    await page.mouse.up();
    expect(await blockSnapshot(page)).toMatchObject({
      keys: ["beta", "alpha", "gamma"],
      dragging: [false, false, false],
    });
    expect(await blockEventCounts(page)).toEqual({ reorder: 1, input: 1, change: 1, select: 1 });
  });

  test("BlockLayout cancel does not resurrect a detached row or overwrite another owner", async ({ page }) => {
    await bootBlockFixture(page);
    const gammaBox = await page.locator("[data-block-studio-block='gamma']").boundingBox();
    if (!gammaBox) throw new Error("gamma row has no geometry");
    await dispatchPointer(page, "[data-block-studio-block='alpha'] [data-block-studio-handle]", "pointerdown", {
      pointerId: 51, clientX: 20, clientY: 20,
    });
    await dispatchPointer(page, "#block-list", "pointermove", {
      pointerId: 51, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
    });
    await page.evaluate(() => {
      document.querySelector("[data-block-studio-block='alpha']")?.remove();
      const beta = document.querySelector("[data-block-studio-block='beta']");
      beta?.querySelector("[data-block-studio-order]")?.setAttribute("value", "external");
      const betaInput = beta?.querySelector("[data-block-studio-order]") as HTMLInputElement | null;
      if (betaInput) betaInput.value = "external";
      beta?.setAttribute("data-block-studio-index", "external");
    });
    // A stale move after external removal must not reinsert the detached row.
    await dispatchPointer(page, "#block-list", "pointermove", {
      pointerId: 51, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
    });
    expect(await page.locator("[data-block-studio-block='alpha']").count()).toBe(0);
    await dispatchPointer(page, "#block-list", "pointercancel", {
      pointerId: 51, clientX: 20, clientY: gammaBox.y + gammaBox.height - 4,
    });
    expect(await page.locator("[data-block-studio-block='alpha']").count()).toBe(0);
    expect(await blockSnapshot(page)).toMatchObject({
      keys: ["beta", "gamma"],
      orders: ["external", "3"],
      indexes: ["external", "2"],
      dragging: [false, false],
    });
    expect(await blockEventCounts(page)).toEqual({ reorder: 0, input: 0, change: 0, select: 0 });
  });

  test("Sitemap node cancel restores inline state, filters pointers, and leaves the next drag committable", async ({ page }) => {
    await page.setContent(siteMapFixture);
    await page.addScriptTag({ content: siteMapRuntimeJS });
    await installSiteMapMoveCounter(page);
    const before = await siteMapSnapshot(page);
    const alphaBox = await page.locator("#alpha").boundingBox();
    if (!alphaBox) throw new Error("alpha node has no geometry");
    const start = { clientX: alphaBox.x + 20, clientY: alphaBox.y + 20 };

    await dispatchPointer(page, "#alpha", "pointerdown", { pointerId: 21, ...start });
    await dispatchDocumentPointer(page, "pointermove", { pointerId: 21, clientX: start.clientX + 80, clientY: start.clientY + 32 });
    expect((await siteMapSnapshot(page) as { nodeLeft: string }).nodeLeft).not.toBe("40px");
    await dispatchDocumentPointer(page, "pointercancel", { pointerId: 22, ...start });
    expect((await siteMapSnapshot(page) as { nodeDragging: string }).nodeDragging).toBe("true");
    expect(await page.evaluate(() => (window as unknown as { __siteMapMoveCount: () => number }).__siteMapMoveCount())).toBe(0);
    await dispatchDocumentPointer(page, "pointercancel", { pointerId: 21, ...start });
    expect(await siteMapSnapshot(page)).toEqual(before);
    expect(await page.evaluate(() => (window as unknown as { __siteMapMoveCount: () => number }).__siteMapMoveCount())).toBe(0);

    // Escape/cancel cleanup must not leave a document move listener behind.
    await dispatchDocumentPointer(page, "pointermove", { pointerId: 21, clientX: start.clientX + 120, clientY: start.clientY + 40 });
    expect(await siteMapSnapshot(page)).toEqual(before);

    // Keyboard activation is not a trailing pointer click and must remain
    // usable even while the one-shot guard is armed.
    await page.evaluate(() => {
      document.querySelector("#alpha")?.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 0 }));
    });
    expect((await siteMapSnapshot(page) as { selectedNode: string }).selectedNode).toBe("alpha");
    await page.evaluate(() => {
      const board = document.querySelector("#board");
      (window as unknown as {
        GoSXStudioSiteMapRuntime: { setState: (target: Element | null, state: Record<string, unknown>) => void };
      }).GoSXStudioSiteMapRuntime.setState(board, { selectedNode: "beta" });
    });

    // A real keyboard activation must pass through while the canceled drag's
    // trailing-click guard is armed; Escape/cancel must not break toolbar use.
    await page.locator("#toolbar").focus();
    await page.keyboard.press("Enter");
    expect((await siteMapSnapshot(page) as { detail: string }).detail).toBe("build");

    // A synthetic trailing click from the canceled drag is ignored. A fresh
    // pointerdown outside the canvas clears that one-shot guard, preserving
    // normal toolbar/tap paths.
    await page.evaluate(() => {
      document.querySelector("#alpha")?.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 }));
    });
    expect((await siteMapSnapshot(page) as { selectedNode: string }).selectedNode).toBe("beta");
    await page.evaluate(() => {
      const board = document.querySelector("#board");
      (window as unknown as {
        GoSXStudioSiteMapRuntime: { setState: (target: Element | null, state: Record<string, unknown>) => void };
      }).GoSXStudioSiteMapRuntime.setState(board, { detail: "map" });
    });
    await page.locator("#toolbar").click();
    expect((await siteMapSnapshot(page) as { detail: string }).detail).toBe("build");
    await page.evaluate(() => {
      document.querySelector("#alpha")?.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 }));
    });
    expect((await siteMapSnapshot(page) as { selectedNode: string }).selectedNode).toBe("alpha");
    await dispatchPointer(page, "#alpha", "pointerdown", { pointerId: 23, ...start });
    await dispatchPointer(page, "#alpha", "pointerup", { pointerId: 23, ...start });
    await page.locator("#alpha").click();
    expect((await siteMapSnapshot(page) as { selectedNode: string }).selectedNode).toBe("alpha");

    const alphaAfterTap = await page.locator("#alpha").boundingBox();
    if (!alphaAfterTap) throw new Error("alpha node has no geometry after tap");
    const nextStart = { clientX: alphaAfterTap.x + 20, clientY: alphaAfterTap.y + 20 };
    // Moving out and back to the exact start is a no-op: restore the original
    // inline values and do not emit a persistence/move event.
    await dispatchPointer(page, "#alpha", "pointerdown", { pointerId: 24, ...nextStart });
    await dispatchDocumentPointer(page, "pointermove", { pointerId: 24, clientX: nextStart.clientX + 64, clientY: nextStart.clientY });
    await dispatchDocumentPointer(page, "pointermove", { pointerId: 24, ...nextStart });
    await dispatchDocumentPointer(page, "pointerup", { pointerId: 24, ...nextStart });
    expect(await page.evaluate(() => (window as unknown as { __siteMapMoveCount: () => number }).__siteMapMoveCount())).toBe(0);
    await dispatchPointer(page, "#alpha", "pointerdown", { pointerId: 25, ...nextStart });
    await dispatchDocumentPointer(page, "pointermove", { pointerId: 25, clientX: nextStart.clientX + 64, clientY: nextStart.clientY });
    await dispatchDocumentPointer(page, "pointerup", { pointerId: 25, clientX: nextStart.clientX + 64, clientY: nextStart.clientY });
    expect(await page.evaluate(() => (window as unknown as { __siteMapMoveCount: () => number }).__siteMapMoveCount())).toBe(1);
    await expect(page.locator("#alpha")).toHaveAttribute("data-studio-site-map-node-x", "100");
  });

  test("Sitemap pan and marquee pointercancel restore their original state", async ({ page }) => {
    await page.setContent(siteMapFixture);
    await page.addScriptTag({ content: siteMapRuntimeJS });
    await installSiteMapMoveCounter(page);
    const before = await siteMapSnapshot(page);
    const canvasBox = await page.locator("#canvas").boundingBox();
    if (!canvasBox) throw new Error("canvas has no geometry");

    const panStart = { clientX: canvasBox.x + 520, clientY: canvasBox.y + 320 };
    await dispatchPointer(page, "#canvas", "pointerdown", { pointerId: 31, ...panStart });
    await dispatchDocumentPointer(page, "pointermove", { pointerId: 31, clientX: panStart.clientX + 90, clientY: panStart.clientY + 20 });
    await dispatchDocumentPointer(page, "pointercancel", { pointerId: 32, ...panStart });
    expect((await siteMapSnapshot(page) as { panX: string }).panX).toBe("100");
    await dispatchDocumentPointer(page, "pointercancel", { pointerId: 31, ...panStart });
    expect(await siteMapSnapshot(page)).toEqual(before);

    // Pan out-and-back is also a no-op: once a gesture has moved, the return
    // move must still write the exact origin before pointerup releases it.
    const panNoopStart = { clientX: canvasBox.x + 520, clientY: canvasBox.y + 320 };
    await dispatchPointer(page, "#canvas", "pointerdown", { pointerId: 33, ...panNoopStart });
    await dispatchDocumentPointer(page, "pointermove", {
      pointerId: 33,
      clientX: panNoopStart.clientX + 90,
      clientY: panNoopStart.clientY + 20,
    });
    await dispatchDocumentPointer(page, "pointermove", { pointerId: 33, ...panNoopStart });
    await dispatchDocumentPointer(page, "pointerup", { pointerId: 33, ...panNoopStart });
    expect(await siteMapSnapshot(page)).toMatchObject({ panX: "10", panY: "20", panning: "false" });

    // A press/release that never moves must restore the original panning flag
    // as well as coordinates, without leaving a gesture marker behind.
    const beforeEmptyPan = await siteMapSnapshot(page);
    await dispatchPointer(page, "#canvas", "pointerdown", { pointerId: 34, ...panNoopStart });
    await dispatchDocumentPointer(page, "pointerup", { pointerId: 34, ...panNoopStart });
    expect(await siteMapSnapshot(page)).toEqual(beforeEmptyPan);

    const beforeMarquee = await siteMapSnapshot(page);
    const emptyMarqueeStart = { clientX: canvasBox.x + 300, clientY: canvasBox.y + 180 };
    await dispatchPointer(page, "#canvas", "pointerdown", { pointerId: 40, ...emptyMarqueeStart, shiftKey: true });
    await dispatchDocumentPointer(page, "pointerup", { pointerId: 40, ...emptyMarqueeStart, shiftKey: true });
    expect(await siteMapSnapshot(page)).toEqual(beforeMarquee);

    const marqueeStart = { clientX: canvasBox.x + 8, clientY: canvasBox.y + 8 };
    await dispatchPointer(page, "#canvas", "pointerdown", { pointerId: 41, ...marqueeStart, shiftKey: true });
    await dispatchDocumentPointer(page, "pointermove", {
      pointerId: 41,
      clientX: canvasBox.x + 180,
      clientY: canvasBox.y + 140,
      shiftKey: true,
    });
    await page.keyboard.press("Escape");
    await dispatchDocumentPointer(page, "pointerup", { pointerId: 41, ...marqueeStart });
    expect(await siteMapSnapshot(page)).toEqual(beforeMarquee);
    expect(await page.evaluate(() => (window as unknown as { __siteMapMoveCount: () => number }).__siteMapMoveCount())).toBe(0);
  });
});
