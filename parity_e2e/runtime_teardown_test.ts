import { expect, test } from "@playwright/test";
import type { Page } from "@playwright/test";
import path from "node:path";

const SELECTION_RUNTIME = path.resolve(__dirname, "../selectionruntime/island_runtime.js");
const WORKBENCH_RUNTIME = path.resolve(__dirname, "../workbenchruntime/island_runtime.js");
const CANVAS_RUNTIME = path.resolve(__dirname, "../canvaswasmfreeruntime/canvas_wasm_free_client.js");

const REPLACEMENT_CYCLES = 3;
const TEARDOWN_TEST_ORIGIN = "http://runtime-teardown.test";

async function establishTestOrigin(page: Page): Promise<void> {
  await page.route(`${TEARDOWN_TEST_ORIGIN}/**`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "text/html",
      body: "<!doctype html><html><head></head><body></body></html>",
    });
  });
  await page.goto(`${TEARDOWN_TEST_ORIGIN}/`);
}

async function installObserverAccounting(page: Page): Promise<void> {
  await page.evaluate(() => {
    type ObserverOwner = "runtime" | "tool" | "unknown";
    const stats = {
      activeObservers: 0,
      toolOwnedObservers: 0,
      unknownObservers: 0,
      createdObservers: 0,
      runtimeCreatedObservers: 0,
      toolCreatedObservers: 0,
      unknownCreatedObservers: 0,
    };
    const OriginalMutationObserver = window.MutationObserver;
    if (!OriginalMutationObserver) throw new Error("MutationObserver is required for teardown coverage");
    const observedTargets = new WeakMap<MutationObserver, Map<Node, ObserverOwner>>();
    const activeOwners = new WeakMap<MutationObserver, ObserverOwner>();
    const createdOwners = new WeakMap<MutationObserver, ObserverOwner>();
    const unknownCreated = new WeakSet<MutationObserver>();

    function isRuntimeObserver(target: Node, options?: MutationObserverInit): boolean {
      if (target === document.documentElement && options?.childList === true && options.subtree === true &&
        options.attributes !== true && options.characterData !== true && options.attributeOldValue !== true &&
        options.characterDataOldValue !== true && options.attributeFilter === undefined) return true;
      if (!(target instanceof HTMLElement) || !target.matches("[data-editor-workbench], [data-studio-workbench]")) return false;
      const filters = options?.attributeFilter;
      return options?.attributes === true && options.childList !== true && options.characterData !== true &&
        options.subtree !== true && options.attributeOldValue !== true && options.characterDataOldValue !== true &&
        Array.isArray(filters) && filters.length === 1 && filters[0] === "data-studio-selection";
    }

    function isToolObserver(target: Node, options?: MutationObserverInit): boolean {
      if (target !== document || !options || options.attributeFilter !== undefined) return false;
      if (options.attributes === true && options.subtree === true && options.childList !== true &&
        options.characterData !== true && options.attributeOldValue !== true && options.characterDataOldValue !== true) return true;
      return options.childList === true && options.attributes !== true && options.characterData !== true &&
        options.subtree !== true && options.attributeOldValue !== true && options.characterDataOldValue !== true;
    }

    function observerOwner(target: Node, options?: MutationObserverInit): ObserverOwner {
      if (isRuntimeObserver(target, options)) return "runtime";
      if (isToolObserver(target, options)) return "tool";
      return "unknown";
    }

    function adjustActive(owner: ObserverOwner, delta: 1 | -1): void {
      if (owner === "runtime") stats.activeObservers += delta;
      else if (owner === "tool") stats.toolOwnedObservers += delta;
      else stats.unknownObservers += delta;
    }

    function recordCreated(owner: ObserverOwner): void {
      if (owner === "runtime") stats.runtimeCreatedObservers += 1;
      else if (owner === "tool") stats.toolCreatedObservers += 1;
      else stats.unknownCreatedObservers += 1;
    }

    function recordUnknownCreation(observer: MutationObserver): void {
      if (unknownCreated.has(observer)) return;
      unknownCreated.add(observer);
      stats.unknownCreatedObservers += 1;
    }

    function activeOwnerFor(targets: Map<Node, ObserverOwner>): ObserverOwner {
      if (targets.size !== 1) return "unknown";
      return targets.values().next().value as ObserverOwner;
    }

    class CountingMutationObserver extends OriginalMutationObserver {
      constructor(callback: MutationCallback) {
        super(callback);
        stats.createdObservers += 1;
      }

      override observe(target: Node, options?: MutationObserverInit): void {
        super.observe(target, options);
        const owner = observerOwner(target, options);
        let targets = observedTargets.get(this);
        if (!targets) {
          targets = new Map<Node, ObserverOwner>();
          observedTargets.set(this, targets);
        }
        targets.set(target, owner);
        if (!createdOwners.has(this)) {
          createdOwners.set(this, owner);
          recordCreated(owner);
          if (owner === "unknown") unknownCreated.add(this);
        }
        const nextOwner = activeOwnerFor(targets);
        if (nextOwner === "unknown") recordUnknownCreation(this);
        const previousOwner = activeOwners.get(this);
        if (previousOwner === nextOwner) return;
        if (previousOwner) adjustActive(previousOwner, -1);
        activeOwners.set(this, nextOwner);
        adjustActive(nextOwner, 1);
      }

      override disconnect(): void {
        super.disconnect();
        const owner = activeOwners.get(this);
        if (owner) {
          activeOwners.delete(this);
          adjustActive(owner, -1);
        }
        observedTargets.delete(this);
      }
    }
    (window as unknown as { MutationObserver: typeof MutationObserver }).MutationObserver = CountingMutationObserver;
    (window as unknown as { __teardownObserverStats: typeof stats }).__teardownObserverStats = stats;
  });
}

async function observerStats(page: Page): Promise<{
  activeObservers: number;
  toolOwnedObservers: number;
  unknownObservers: number;
  createdObservers: number;
  runtimeCreatedObservers: number;
  toolCreatedObservers: number;
  unknownCreatedObservers: number;
}> {
  return await page.evaluate(() => {
    const stats = (window as unknown as {
      __teardownObserverStats?: {
        activeObservers: number;
        toolOwnedObservers: number;
        unknownObservers: number;
        createdObservers: number;
        runtimeCreatedObservers: number;
        toolCreatedObservers: number;
        unknownCreatedObservers: number;
      };
    }).__teardownObserverStats;
    if (!stats) throw new Error("observer accounting stats missing");
    if (stats.unknownObservers !== 0 || stats.unknownCreatedObservers !== 0) {
      throw new Error(`unknown MutationObserver ownership: active=${stats.unknownObservers}, created=${stats.unknownCreatedObservers}`);
    }
    return stats;
  });
}

function selectionMarkup(cycle: number): string {
  return `
    <form data-studio-workbench="true" data-studio-mode="home" data-cycle="${cycle}">
      <button type="button" data-studio-selection-action="reveal">Reveal</button>
      <output data-studio-selection-label></output>
      <output data-studio-selection-status></output>
      <div data-block-studio-block="hero-${cycle}" data-studio-block-label="Hero ${cycle}" tabindex="0">Hero ${cycle}</div>
      <div data-block-studio-block="footer-${cycle}" data-studio-block-label="Footer ${cycle}" tabindex="0">Footer ${cycle}</div>
      <div data-studio-command-palette></div>
    </form>`;
}

test.describe("shared runtime fragment teardown", () => {
  test.describe.configure({ timeout: 30_000 });

  test("selection replacement disconnects document handlers and keeps one active selection binding", async ({ page }, testInfo) => {
    await page.setContent(selectionMarkup(0));
    await installObserverAccounting(page);
    await page.addScriptTag({ path: SELECTION_RUNTIME });

    await page.evaluate(() => {
      const bind = (window as unknown as { __gosx_selection_runtime_island_bind?: (root: Document) => void }).__gosx_selection_runtime_island_bind;
      if (!bind) throw new Error("selection island bind global missing");
      bind(document);
      bind(document);
      (window as unknown as { __selectionActionCount: number }).__selectionActionCount = 0;
      document.addEventListener("gosxstudio:selection-action", () => {
        (window as unknown as { __selectionActionCount: number }).__selectionActionCount += 1;
      });
    });

    const initialForm = page.locator("[data-studio-workbench]").first();
    await initialForm.locator("[data-studio-selection-action='reveal']").click();
    await expect.poll(() => page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount)).toBe(1);
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);

    // Keep the form node alive while replacing only its command palette. The
    // old child must become inert and the replacement must receive one
    // listener through the form-owned lifecycle observer.
    const selectionCountBeforePalette = await page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount);
    await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-studio-workbench]");
      const oldPalette = form?.querySelector<HTMLElement>("[data-studio-command-palette]");
      if (!form || !oldPalette) throw new Error("selection command palette missing before inner replacement");
      const nextPalette = document.createElement("div");
      nextPalette.setAttribute("data-studio-command-palette", "true");
      oldPalette.replaceWith(nextPalette);
      (window as unknown as { __oldSelectionPalette: HTMLElement }).__oldSelectionPalette = oldPalette;
      (window as unknown as { __newSelectionPalette: HTMLElement }).__newSelectionPalette = nextPalette;
      oldPalette.dispatchEvent(new CustomEvent("gosxstudio:command", { bubbles: true, detail: { kind: "selection-action", target: "reveal" } }));
    });
    expect(await page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount)).toBe(selectionCountBeforePalette);
    await page.waitForFunction(() => {
      const oldPalette = (window as unknown as { __oldSelectionPalette?: HTMLElement }).__oldSelectionPalette;
      const newPalette = (window as unknown as { __newSelectionPalette?: HTMLElement }).__newSelectionPalette;
      return !!oldPalette && !!newPalette && !oldPalette.hasAttribute("data-gosx-studio-selection-island-commands-bound") && newPalette.hasAttribute("data-gosx-studio-selection-island-commands-bound");
    });
    await page.evaluate(() => {
      const oldPalette = (window as unknown as { __oldSelectionPalette: HTMLElement }).__oldSelectionPalette;
      oldPalette.dispatchEvent(new CustomEvent("gosxstudio:command", { bubbles: true, detail: { kind: "selection-action", target: "reveal" } }));
    });
    expect(await page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount)).toBe(selectionCountBeforePalette);
    await page.evaluate(() => {
      const newPalette = (window as unknown as { __newSelectionPalette: HTMLElement }).__newSelectionPalette;
      newPalette.dispatchEvent(new CustomEvent("gosxstudio:command", { bubbles: true, detail: { kind: "selection-action", target: "reveal" } }));
    });
    await expect.poll(() => page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount)).toBe(selectionCountBeforePalette + 1);
    let expectedSelectionActions = selectionCountBeforePalette + 1;

    for (let cycle = 1; cycle <= REPLACEMENT_CYCLES; cycle += 1) {
      await page.evaluate((nextCycle) => {
        const current = document.querySelector<HTMLElement>("[data-studio-workbench]");
        if (!current) throw new Error("selection form missing before replacement");
        const next = current.cloneNode(true) as HTMLElement;
        next.setAttribute("data-cycle", String(nextCycle));
        const blocks = next.querySelectorAll<HTMLElement>("[data-block-studio-block]");
        blocks[0]?.setAttribute("data-block-studio-block", `hero-${nextCycle}`);
        blocks[0]?.setAttribute("data-studio-block-label", `Hero ${nextCycle}`);
        blocks[1]?.setAttribute("data-block-studio-block", `footer-${nextCycle}`);
        blocks[1]?.setAttribute("data-studio-block-label", `Footer ${nextCycle}`);
        const old = current;
        old.replaceWith(next);
        (window as unknown as { __oldSelectionForm: HTMLElement }).__oldSelectionForm = old;
        const bind = (window as unknown as { __gosx_selection_runtime_island_bind?: (root: Document) => void }).__gosx_selection_runtime_island_bind;
        if (!bind) throw new Error("selection island bind global missing after replacement");
        bind(document);
        bind(document);
      }, cycle);

      await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
      await page.waitForFunction(() => {
        const old = (window as unknown as { __oldSelectionForm?: HTMLElement }).__oldSelectionForm;
        return !!old && !old.hasAttribute("data-gosx-studio-selection-island-bound");
      });

      await page.evaluate(() => {
        const old = (window as unknown as { __oldSelectionForm: HTMLElement }).__oldSelectionForm;
        old.querySelector<HTMLElement>("[data-studio-selection-action]")?.click();
      });
      expect(await page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount)).toBe(expectedSelectionActions);

      await page.locator("[data-studio-workbench] [data-studio-selection-action]").click();
      expectedSelectionActions += 1;
      await expect.poll(() => page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount)).toBe(expectedSelectionActions);
      await page.locator("[data-studio-workbench] [data-block-studio-block]").last().click();
      await expect(page.locator("[data-studio-workbench]")).toHaveAttribute("data-studio-selection", new RegExp(`footer-${cycle}`));
    }

    await testInfo.attach("selection-runtime-teardown-evidence.json", {
      contentType: "application/json",
      body: Buffer.from(JSON.stringify({
        cycles: REPLACEMENT_CYCLES,
        selectionActionCount: await page.evaluate(() => (window as unknown as { __selectionActionCount: number }).__selectionActionCount),
        observerStats: await observerStats(page),
        oldNodeMarkerRemoved: await page.evaluate(() => !(window as unknown as { __oldSelectionForm: HTMLElement }).__oldSelectionForm.hasAttribute("data-gosx-studio-selection-island-bound")),
      }, null, 2)),
    });
  });

  test("workbench replacement disconnects rail/scroll controllers and keeps keyboard + resize dispatch singular", async ({ page }, testInfo) => {
    const workbenchMarkup = (cycle: number) => `
      <form data-editor-workbench="true" data-studio-mode="home" data-cycle="${cycle}" style="--studio-left-width: 240px !important">
        <button type="button" data-studio-mode-control="home">Home</button>
        <button type="button" data-studio-mode-control="advanced">Advanced</button>
        <div data-studio-mode-panel="home"></div>
        <div data-studio-stage style="width: 900px; height: 320px; overflow: auto">
          <div style="height: 800px"></div>
        </div>
        <div data-studio-resizer="left" data-studio-resizer-min="120" data-studio-resizer-max="480" tabindex="0"></div>
        <div data-studio-command-palette></div>
      </form>`;

    await establishTestOrigin(page);
    await page.setContent(workbenchMarkup(0));
    const pageErrors: string[] = [];
    page.on("pageerror", (error) => pageErrors.push(error.message));
    await installObserverAccounting(page);
    await page.addScriptTag({ path: WORKBENCH_RUNTIME });
    await page.evaluate(() => {
      const w = window as unknown as {
        __gosx_workbench_runtime_island_bindChrome?: (root: Document) => void;
        __gosx_workbench_runtime_island_bindRailResizers?: (root: Document) => void;
        __railCommitCount: number;
        __resizeCount: number;
        __modeChangeCount: number;
      };
      if (!w.__gosx_workbench_runtime_island_bindChrome || !w.__gosx_workbench_runtime_island_bindRailResizers) {
        throw new Error("workbench island bind globals missing");
      }
      w.__gosx_workbench_runtime_island_bindChrome(document);
      w.__gosx_workbench_runtime_island_bindRailResizers(document);
      w.__gosx_workbench_runtime_island_bindChrome(document);
      w.__gosx_workbench_runtime_island_bindRailResizers(document);
      w.__railCommitCount = 0;
      w.__resizeCount = 0;
      w.__modeChangeCount = 0;
      document.addEventListener("gosxstudio:rail-width-commit", () => { w.__railCommitCount += 1; });
      document.addEventListener("gosxstudio:workbench-mode-change", () => { w.__modeChangeCount += 1; });
      window.addEventListener("resize", () => { w.__resizeCount += 1; });
    });

    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(2);
    await page.evaluate(() => new Promise<void>((resolve) => {
      if (typeof window.requestAnimationFrame === "function") {
        window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()));
      } else {
        window.setTimeout(resolve, 40);
      }
    }));
    const initialResizeBaseline = await page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount);
    const initialRailRollbackState = await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      const handle = form?.querySelector<HTMLElement>("[data-studio-resizer]");
      if (!form || !handle) throw new Error("workbench controls missing before rollback baseline");
      const widthProperty = "--studio-left-width";
      return {
        widthValue: form.style.getPropertyValue(widthProperty),
        widthPriority: form.style.getPropertyPriority(widthProperty),
        widthPresent: Array.from(form.style).includes(widthProperty),
        ariaPresent: handle.hasAttribute("aria-valuenow"),
        ariaValue: handle.getAttribute("aria-valuenow"),
      };
    });
    await page.locator("[data-studio-resizer]").dispatchEvent("keydown", { key: "ArrowRight", bubbles: true });
    await expect.poll(() => page.evaluate(() => (window as unknown as { __railCommitCount: number }).__railCommitCount)).toBe(1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount)).toBe(initialResizeBaseline + 1);

    // Replace stage, rail handle, and command palette independently while the
    // form remains connected. Each old child must lose its subscription, and
    // each replacement must become interactive through the shared lifecycle.
    const savedWorkingStateBeforeInnerSwap = await page.evaluate(() => window.sessionStorage.getItem("gosx-studio-editor-working-state"));
    const innerSwapCountsBefore = await page.evaluate(() => ({
      commits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
      resizes: (window as unknown as { __resizeCount: number }).__resizeCount,
      modes: (window as unknown as { __modeChangeCount: number }).__modeChangeCount,
    }));
    await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      const oldStage = form?.querySelector<HTMLElement>("[data-studio-stage]");
      const oldHandle = form?.querySelector<HTMLElement>("[data-studio-resizer]");
      const oldPalette = form?.querySelector<HTMLElement>("[data-studio-command-palette]");
      if (!form || !oldStage || !oldHandle || !oldPalette) throw new Error("workbench child missing before inner replacement");
      const newStage = oldStage.cloneNode(true) as HTMLElement;
      const newHandle = oldHandle.cloneNode(true) as HTMLElement;
      const newPalette = oldPalette.cloneNode(true) as HTMLElement;
      newHandle.removeAttribute("data-gosx-studio-resizer-island-bound");
      newPalette.removeAttribute("data-gosx-studio-workbench-commands-island-bound");
      oldStage.replaceWith(newStage);
      oldHandle.replaceWith(newHandle);
      oldPalette.replaceWith(newPalette);
      oldPalette.dispatchEvent(new CustomEvent("gosxstudio:command", { bubbles: true, detail: { kind: "mode", target: "advanced" } }));
      (window as unknown as {
        __oldInnerStage: HTMLElement;
        __oldInnerHandle: HTMLElement;
        __oldInnerPalette: HTMLElement;
        __newInnerStage: HTMLElement;
        __newInnerHandle: HTMLElement;
        __newInnerPalette: HTMLElement;
      }).__oldInnerStage = oldStage;
      (window as unknown as { __oldInnerHandle: HTMLElement }).__oldInnerHandle = oldHandle;
      (window as unknown as { __oldInnerPalette: HTMLElement }).__oldInnerPalette = oldPalette;
      (window as unknown as { __newInnerStage: HTMLElement }).__newInnerStage = newStage;
      (window as unknown as { __newInnerHandle: HTMLElement }).__newInnerHandle = newHandle;
      (window as unknown as { __newInnerPalette: HTMLElement }).__newInnerPalette = newPalette;
    });
    expect(await page.evaluate(() => ({
      commits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
      resizes: (window as unknown as { __resizeCount: number }).__resizeCount,
      modes: (window as unknown as { __modeChangeCount: number }).__modeChangeCount,
    }))).toEqual(innerSwapCountsBefore);
    await page.waitForFunction(() => {
      const oldHandle = (window as unknown as { __oldInnerHandle?: HTMLElement }).__oldInnerHandle;
      const oldPalette = (window as unknown as { __oldInnerPalette?: HTMLElement }).__oldInnerPalette;
      const newHandle = (window as unknown as { __newInnerHandle?: HTMLElement }).__newInnerHandle;
      const newPalette = (window as unknown as { __newInnerPalette?: HTMLElement }).__newInnerPalette;
      return !!oldHandle && !!oldPalette && !!newHandle && !!newPalette &&
        !oldHandle.hasAttribute("data-gosx-studio-resizer-island-bound") &&
        !oldPalette.hasAttribute("data-gosx-studio-workbench-commands-island-bound") &&
        newHandle.hasAttribute("data-gosx-studio-resizer-island-bound") &&
        newPalette.hasAttribute("data-gosx-studio-workbench-commands-island-bound");
    });
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(2);
    const innerSwapCounts = await page.evaluate(() => ({
      commits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
      resizes: (window as unknown as { __resizeCount: number }).__resizeCount,
      modes: (window as unknown as { __modeChangeCount: number }).__modeChangeCount,
    }));
    await page.evaluate(() => {
      const oldHandle = (window as unknown as { __oldInnerHandle: HTMLElement }).__oldInnerHandle;
      const oldPalette = (window as unknown as { __oldInnerPalette: HTMLElement }).__oldInnerPalette;
      const oldStage = (window as unknown as { __oldInnerStage: HTMLElement }).__oldInnerStage;
      oldHandle.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }));
      oldPalette.dispatchEvent(new CustomEvent("gosxstudio:command", { bubbles: true, detail: { kind: "mode", target: "advanced" } }));
      oldStage.scrollTop = 777;
      oldStage.dispatchEvent(new Event("scroll", { bubbles: true }));
    });
    await page.waitForTimeout(40);
    expect(await page.evaluate(() => ({
      commits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
      modes: (window as unknown as { __modeChangeCount: number }).__modeChangeCount,
      workingState: window.sessionStorage.getItem("gosx-studio-editor-working-state"),
    }))).toEqual({
      commits: innerSwapCounts.commits,
      modes: innerSwapCounts.modes,
      workingState: savedWorkingStateBeforeInnerSwap,
    });
    await page.evaluate(() => {
      const newHandle = (window as unknown as { __newInnerHandle: HTMLElement }).__newInnerHandle;
      const newPalette = (window as unknown as { __newInnerPalette: HTMLElement }).__newInnerPalette;
      newHandle.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }));
      newPalette.dispatchEvent(new CustomEvent("gosxstudio:command", { bubbles: true, detail: { kind: "mode", target: "advanced" } }));
    });
    await expect.poll(() => page.evaluate(() => (window as unknown as { __railCommitCount: number }).__railCommitCount)).toBe(innerSwapCounts.commits + 1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount)).toBe(innerSwapCounts.resizes + 1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __modeChangeCount: number }).__modeChangeCount)).toBe(innerSwapCounts.modes + 1);

    // A live pointer move must not make the active rail width durable. Let a
    // frame boundary pass while the gesture is still active, then perform an
    // independent activity save: it must retain the captured active-side
    // width and the other rail while updating activity. Cancellation keeps
    // that legitimate independent update, and a cancel-only gesture with an
    // absent layout entry must leave the entry absent.
    const persistenceResult = await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      const handle = (window as unknown as { __newInnerHandle: HTMLElement }).__newInnerHandle;
      const layoutKey = "gosx-studio-editor-layout";
      const widthProperty = "--studio-left-width";
      if (!form || !handle) throw new Error("workbench controls missing before persistence rollback");
      form.style.setProperty(widthProperty, "240px", "important");
      form.style.setProperty("--studio-right-width", "376px");
      form.setAttribute("data-studio-activity-state", "open");
      handle.removeAttribute("aria-valuenow");
      const beforeStorage = JSON.stringify({ left: "240px", right: "376px", activity: "open" });
      window.localStorage.setItem(layoutKey, beforeStorage);
      const rect = handle.getBoundingClientRect();
      const beforeCommits = (window as unknown as { __railCommitCount: number }).__railCommitCount;
      const beforeWidth = form.style.getPropertyValue(widthProperty);
      handle.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 41, clientX: rect.left + 100, clientY: rect.top + 2 }));
      document.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 41, clientX: rect.left + 420, clientY: rect.top + 2 }));
      const duringWidth = form.style.getPropertyValue(widthProperty);
      return {
        beforeStorage,
        beforeWidth,
        duringWidth,
        beforeCommits,
      };
    });
    await page.evaluate(() => new Promise<void>((resolve) => {
      if (typeof window.requestAnimationFrame === "function") {
        window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()));
      } else {
        window.setTimeout(resolve, 40);
      }
    }));
    const delayedLivePersistence = await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      if (!form) throw new Error("workbench form missing during delayed persistence check");
      return {
        storage: window.localStorage.getItem("gosx-studio-editor-layout"),
        width: form.style.getPropertyValue("--studio-left-width"),
      };
    });
    expect(delayedLivePersistence.storage).toBe(persistenceResult.beforeStorage);
    expect(delayedLivePersistence.width).toBe(persistenceResult.duringWidth);
    expect(delayedLivePersistence.width).not.toBe(persistenceResult.beforeWidth);

    const otherRailDuringGesture = await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      const w = window as unknown as {
        __gosx_workbench_runtime_island_setRailWidth?: (target: HTMLElement, side: string, width: number, handle: HTMLElement | null, committed: boolean) => void;
      };
      if (!form || !w.__gosx_workbench_runtime_island_setRailWidth) throw new Error("rail width global missing during persistence check");
      w.__gosx_workbench_runtime_island_setRailWidth(form, "right", 412, null, false);
      return form.style.getPropertyValue("--studio-right-width");
    });
    await page.evaluate(() => new Promise<void>((resolve) => {
      if (typeof window.requestAnimationFrame === "function") {
        window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()));
      } else {
        window.setTimeout(resolve, 40);
      }
    }));
    const otherRailPersistence = await page.evaluate(() => window.localStorage.getItem("gosx-studio-editor-layout"));
    expect(otherRailDuringGesture).toBe("412px");
    expect(JSON.parse(otherRailPersistence || "null")).toEqual({ left: "240px", right: "412px", activity: "open" });

    const independentSaveDuringGesture = await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      const w = window as unknown as {
        __gosx_workbench_runtime_island_toggleActivity?: (target: HTMLElement) => void;
      };
      if (!form || !w.__gosx_workbench_runtime_island_toggleActivity) throw new Error("activity toggle global missing during persistence check");
      w.__gosx_workbench_runtime_island_toggleActivity(form);
      return {
        activity: form.getAttribute("data-studio-activity-state"),
        storage: window.localStorage.getItem("gosx-studio-editor-layout"),
      };
    });
    const independentLayout = JSON.parse(independentSaveDuringGesture.storage || "null") as {
      left?: string;
      right?: string;
      activity?: string;
    } | null;
    expect(independentSaveDuringGesture.activity).toBe("collapsed");
    expect(independentLayout).toEqual({ left: "240px", right: "412px", activity: "collapsed" });

    await page.evaluate(() => new Promise<void>((resolve) => {
      if (typeof window.requestAnimationFrame === "function") {
        window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()));
      } else {
        window.setTimeout(resolve, 40);
      }
    }));
    const cancelResizeBaseline = await page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount);
    const persistenceCancel = await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      const handle = (window as unknown as { __newInnerHandle: HTMLElement }).__newInnerHandle;
      if (!form || !handle) throw new Error("workbench controls missing during persistence cancel");
      document.dispatchEvent(new PointerEvent("pointercancel", { bubbles: true, pointerId: 41 }));
      return {
        width: form.style.getPropertyValue("--studio-left-width"),
        widthPriority: form.style.getPropertyPriority("--studio-left-width"),
        ariaPresent: handle.hasAttribute("aria-valuenow"),
        storage: window.localStorage.getItem("gosx-studio-editor-layout"),
        commits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
      };
    });
    await page.evaluate(() => new Promise<void>((resolve) => {
      if (typeof window.requestAnimationFrame === "function") {
        window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()));
      } else {
        window.setTimeout(resolve, 40);
      }
    }));
    const persistenceCancelRefresh = await page.evaluate(() => ({
      resizeCount: (window as unknown as { __resizeCount: number }).__resizeCount,
      width: document.querySelector<HTMLElement>("[data-editor-workbench]")?.style.getPropertyValue("--studio-left-width"),
    }));
    expect(persistenceCancel.width).toBe("240px");
    expect(persistenceCancel.widthPriority).toBe("important");
    expect(persistenceCancel.ariaPresent).toBe(false);
    expect(persistenceCancel.storage).toBe(independentSaveDuringGesture.storage);
    expect(persistenceCancel.commits).toBe(persistenceResult.beforeCommits);
    expect(persistenceCancelRefresh.resizeCount).toBe(cancelResizeBaseline + 1);
    expect(persistenceCancelRefresh.width).toBe(persistenceCancel.width);

    // Replace the stage and handle in the same task as an active gesture,
    // before the shared MutationObserver can reconcile the child bindings.
    // The stale pointerup must cancel rather than commit, and rollback must
    // target the replacement handle as well as the still-connected form.
    const staleGestureResult = await page.evaluate(() => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]");
      const oldStage = form?.querySelector<HTMLElement>("[data-studio-stage]");
      const oldHandle = form?.querySelector<HTMLElement>("[data-studio-resizer]");
      if (!form || !oldStage || !oldHandle) throw new Error("workbench controls missing before stale gesture swap");
      const rect = oldStage.getBoundingClientRect();
      const beforeCommits = (window as unknown as { __railCommitCount: number }).__railCommitCount;
      const beforeWidth = form.style.getPropertyValue("--studio-left-width");
      oldHandle.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 42, clientX: rect.left + 100, clientY: rect.top + 2 }));
      document.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 42, clientX: rect.left + 420, clientY: rect.top + 2 }));
      const duringWidth = form.style.getPropertyValue("--studio-left-width");
      const newStage = oldStage.cloneNode(true) as HTMLElement;
      const newHandle = oldHandle.cloneNode(true) as HTMLElement;
      newHandle.removeAttribute("data-gosx-studio-resizer-island-bound");
      oldStage.replaceWith(newStage);
      oldHandle.replaceWith(newHandle);
      document.dispatchEvent(new PointerEvent("pointerup", { bubbles: true, pointerId: 42, clientX: rect.left + 420, clientY: rect.top + 2 }));
      (window as unknown as {
        __oldInnerStage: HTMLElement;
        __oldInnerHandle: HTMLElement;
        __newInnerStage: HTMLElement;
        __newInnerHandle: HTMLElement;
      }).__oldInnerStage = oldStage;
      (window as unknown as { __oldInnerHandle: HTMLElement }).__oldInnerHandle = oldHandle;
      (window as unknown as { __newInnerStage: HTMLElement }).__newInnerStage = newStage;
      (window as unknown as { __newInnerHandle: HTMLElement }).__newInnerHandle = newHandle;
      return {
        beforeCommits,
        afterCommits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
        beforeWidth,
        duringWidth,
        afterWidth: form.style.getPropertyValue("--studio-left-width"),
        afterPriority: form.style.getPropertyPriority("--studio-left-width"),
        newAriaPresent: newHandle.hasAttribute("aria-valuenow"),
      };
    });
    expect(staleGestureResult.duringWidth).not.toBe(staleGestureResult.beforeWidth);
    expect(staleGestureResult.afterCommits).toBe(staleGestureResult.beforeCommits);
    expect(staleGestureResult.afterWidth).toBe(staleGestureResult.beforeWidth);
    expect(staleGestureResult.afterPriority).toBe("important");
    expect(staleGestureResult.newAriaPresent).toBe(false);
    await page.waitForFunction(() => {
      const oldStage = (window as unknown as { __oldInnerStage?: HTMLElement }).__oldInnerStage;
      const oldHandle = (window as unknown as { __oldInnerHandle?: HTMLElement }).__oldInnerHandle;
      const newStage = (window as unknown as { __newInnerStage?: HTMLElement }).__newInnerStage;
      const newHandle = (window as unknown as { __newInnerHandle?: HTMLElement }).__newInnerHandle;
      return !!oldStage && !!oldHandle && !!newStage && !!newHandle &&
        !oldHandle.hasAttribute("data-gosx-studio-resizer-island-bound") &&
        newHandle.hasAttribute("data-gosx-studio-resizer-island-bound") &&
        newStage.isConnected;
    });

    // Pointer cancellation rolls back the in-progress width and emits no
    // commit. Force capture to fail so document lifecycle ownership is
    // exercised even when a synthetic/stale pointer cannot be captured. The
    // Escape gesture keeps focus on an unrelated field and then a valid
    // release proves the cancellation listeners were removed exactly once.
    const cancellationResult = await page.evaluate((seed) => {
      const form = document.querySelector<HTMLElement>("[data-editor-workbench]")!;
      const handle = (window as unknown as { __newInnerHandle: HTMLElement }).__newInnerHandle;
      const widthProperty = "--studio-left-width";
      // Restore the authored baseline captured before the keyboard smoke check
      // so this block exercises exact present/priority/ARIA rollback rather
      // than depending on that earlier committed nudge's normalized style.
      if (seed.widthPresent) form.style.setProperty(widthProperty, seed.widthValue, seed.widthPriority);
      else form.style.removeProperty(widthProperty);
      if (seed.ariaPresent) handle.setAttribute("aria-valuenow", seed.ariaValue || "");
      else handle.removeAttribute("aria-valuenow");
      const beforeWidth = form.style.getPropertyValue(widthProperty);
      const beforeWidthPriority = form.style.getPropertyPriority(widthProperty);
      const beforeWidthPresent = Array.from(form.style).includes(widthProperty);
      const beforeAriaPresent = handle.hasAttribute("aria-valuenow");
      const beforeAriaValue = handle.getAttribute("aria-valuenow");
      const beforeCommits = (window as unknown as { __railCommitCount: number }).__railCommitCount;
      const layoutKey = "gosx-studio-editor-layout";
      const beforeStorage = window.localStorage.getItem(layoutKey);
      const rect = handle.getBoundingClientRect();
      const originalCaptureDescriptor = Object.getOwnPropertyDescriptor(handle, "setPointerCapture");
      let captureAttempts = 0;
      Object.defineProperty(handle, "setPointerCapture", {
        configurable: true,
        writable: true,
        value: () => {
          captureAttempts += 1;
          throw new Error("synthetic capture failure");
        },
      });
      const focusSentinel = document.createElement("input");
      focusSentinel.type = "text";
      focusSentinel.setAttribute("aria-label", "focus sentinel");
      document.body.appendChild(focusSentinel);
      focusSentinel.focus();

      handle.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 17, clientX: rect.left + 100, clientY: rect.top + 2 }));
      document.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 17, clientX: rect.left + 420, clientY: rect.top + 2 }));
      const duringCancelWidth = form.style.getPropertyValue(widthProperty);
      document.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 99, clientX: rect.left + 1800, clientY: rect.top + 2 }));
      const wrongPointerWidth = form.style.getPropertyValue(widthProperty);
      document.dispatchEvent(new PointerEvent("pointercancel", { bubbles: true, pointerId: 99 }));
      const afterWrongPointerCancelWidth = form.style.getPropertyValue(widthProperty);
      document.dispatchEvent(new PointerEvent("pointercancel", { bubbles: true, pointerId: 17 }));
      const afterCancelWidth = form.style.getPropertyValue(widthProperty);
      const afterCancelWidthPriority = form.style.getPropertyPriority(widthProperty);
      const afterCancelWidthPresent = Array.from(form.style).includes(widthProperty);
      const afterCancelAriaPresent = handle.hasAttribute("aria-valuenow");
      const afterCancelAriaValue = handle.getAttribute("aria-valuenow");
      const afterCancelCommits = (window as unknown as { __railCommitCount: number }).__railCommitCount;
      const afterCancelStorage = window.localStorage.getItem(layoutKey);

      handle.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 18, clientX: rect.left + 100, clientY: rect.top + 2 }));
      document.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 18, clientX: rect.left + 420, clientY: rect.top + 2 }));
      const duringEscapeWidth = form.style.getPropertyValue(widthProperty);
      const focusStayedElsewhere = document.activeElement === focusSentinel;
      document.dispatchEvent(new KeyboardEvent("keydown", { bubbles: true, key: "Escape" }));
      const afterEscapeWidth = form.style.getPropertyValue(widthProperty);
      const afterEscapeWidthPriority = form.style.getPropertyPriority(widthProperty);
      const afterEscapeWidthPresent = Array.from(form.style).includes(widthProperty);
      const afterEscapeAriaPresent = handle.hasAttribute("aria-valuenow");
      const afterEscapeAriaValue = handle.getAttribute("aria-valuenow");
      const afterEscapeCommits = (window as unknown as { __railCommitCount: number }).__railCommitCount;
      const afterEscapeStorage = window.localStorage.getItem(layoutKey);

      if (originalCaptureDescriptor) {
        Object.defineProperty(handle, "setPointerCapture", originalCaptureDescriptor);
      } else {
        delete (handle as unknown as { setPointerCapture?: unknown }).setPointerCapture;
      }
      form.style.removeProperty(widthProperty);
      handle.setAttribute("aria-valuenow", "seed");
      const absentBeforeWidth = form.style.getPropertyValue(widthProperty);
      const absentBeforeWidthPriority = form.style.getPropertyPriority(widthProperty);
      const absentBeforeWidthPresent = Array.from(form.style).includes(widthProperty);
      const presentBeforeAria = handle.hasAttribute("aria-valuenow");
      const presentBeforeAriaValue = handle.getAttribute("aria-valuenow");
      window.localStorage.removeItem(layoutKey);
      const absentBeforeStorage = window.localStorage.getItem(layoutKey);
      handle.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 20, clientX: rect.left + 100, clientY: rect.top + 2 }));
      document.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 20, clientX: rect.left + 420, clientY: rect.top + 2 }));
      const duringAbsentCancelWidth = form.style.getPropertyValue(widthProperty);
      document.dispatchEvent(new PointerEvent("pointercancel", { bubbles: true, pointerId: 20 }));
      const afterAbsentCancelWidth = form.style.getPropertyValue(widthProperty);
      const afterAbsentCancelWidthPriority = form.style.getPropertyPriority(widthProperty);
      const afterAbsentCancelWidthPresent = Array.from(form.style).includes(widthProperty);
      const afterAbsentCancelAriaPresent = handle.hasAttribute("aria-valuenow");
      const afterAbsentCancelAriaValue = handle.getAttribute("aria-valuenow");
      const afterAbsentCancelStorage = window.localStorage.getItem(layoutKey);
      const beforeValidStorage = afterAbsentCancelStorage;
      handle.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 19, clientX: rect.left + 100, clientY: rect.top + 2 }));
      document.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 19, clientX: rect.left + 360, clientY: rect.top + 2 }));
      const duringValidWidth = form.style.getPropertyValue("--studio-left-width");
      document.dispatchEvent(new PointerEvent("pointerup", { bubbles: true, pointerId: 19, clientX: rect.left + 360, clientY: rect.top + 2 }));
      const afterValidWidth = form.style.getPropertyValue("--studio-left-width");
      const afterValidStorage = window.localStorage.getItem(layoutKey);
      focusSentinel.remove();
      return {
        beforeWidth,
        beforeWidthPriority,
        beforeWidthPresent,
        beforeAriaPresent,
        beforeAriaValue,
        duringCancelWidth,
        wrongPointerWidth,
        afterWrongPointerCancelWidth,
        afterCancelWidth,
        afterCancelWidthPriority,
        afterCancelWidthPresent,
        afterCancelAriaPresent,
        afterCancelAriaValue,
        afterCancelCommits,
        beforeStorage,
        afterCancelStorage,
        duringEscapeWidth,
        focusStayedElsewhere,
        afterEscapeWidth,
        afterEscapeWidthPriority,
        afterEscapeWidthPresent,
        afterEscapeAriaPresent,
        afterEscapeAriaValue,
        afterEscapeCommits,
        afterEscapeStorage,
        absentBeforeWidth,
        absentBeforeWidthPriority,
        absentBeforeWidthPresent,
        presentBeforeAria,
        presentBeforeAriaValue,
        duringAbsentCancelWidth,
        afterAbsentCancelWidth,
        afterAbsentCancelWidthPriority,
        afterAbsentCancelWidthPresent,
        afterAbsentCancelAriaPresent,
        afterAbsentCancelAriaValue,
        absentBeforeStorage,
        afterAbsentCancelStorage,
        beforeValidStorage,
        duringValidWidth,
        afterValidWidth,
        afterValidStorage,
        captureAttempts,
        commits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
        beforeCommits,
      };
    }, initialRailRollbackState);
    expect(pageErrors).toEqual([]);
    expect(cancellationResult.captureAttempts).toBe(2);
    expect(cancellationResult.beforeWidthPresent).toBe(true);
    expect(cancellationResult.beforeWidthPriority).toBe("important");
    expect(cancellationResult.beforeAriaPresent).toBe(false);
    expect(cancellationResult.beforeAriaValue).toBeNull();
    expect(cancellationResult.duringCancelWidth).not.toBe(cancellationResult.beforeWidth);
    expect(cancellationResult.wrongPointerWidth).toBe(cancellationResult.duringCancelWidth);
    expect(cancellationResult.afterWrongPointerCancelWidth).toBe(cancellationResult.duringCancelWidth);
    expect(cancellationResult.afterCancelWidth).toBe(cancellationResult.beforeWidth);
    expect(cancellationResult.afterCancelWidthPresent).toBe(cancellationResult.beforeWidthPresent);
    expect(cancellationResult.afterCancelWidthPriority).toBe(cancellationResult.beforeWidthPriority);
    expect(cancellationResult.afterCancelAriaPresent).toBe(cancellationResult.beforeAriaPresent);
    expect(cancellationResult.afterCancelAriaValue).toBe(cancellationResult.beforeAriaValue);
    expect(cancellationResult.afterCancelCommits).toBe(cancellationResult.beforeCommits);
    expect(cancellationResult.beforeStorage).toBe(independentSaveDuringGesture.storage);
    expect(cancellationResult.afterCancelStorage).toBe(cancellationResult.beforeStorage);
    expect(cancellationResult.duringEscapeWidth).not.toBe(cancellationResult.beforeWidth);
    expect(cancellationResult.focusStayedElsewhere).toBe(true);
    expect(cancellationResult.afterEscapeWidth).toBe(cancellationResult.beforeWidth);
    expect(cancellationResult.afterEscapeWidthPresent).toBe(cancellationResult.beforeWidthPresent);
    expect(cancellationResult.afterEscapeWidthPriority).toBe(cancellationResult.beforeWidthPriority);
    expect(cancellationResult.afterEscapeAriaPresent).toBe(cancellationResult.beforeAriaPresent);
    expect(cancellationResult.afterEscapeAriaValue).toBe(cancellationResult.beforeAriaValue);
    expect(cancellationResult.afterEscapeCommits).toBe(cancellationResult.beforeCommits);
    expect(cancellationResult.afterEscapeStorage).toBe(cancellationResult.beforeStorage);
    expect(cancellationResult.absentBeforeWidth).toBe("");
    expect(cancellationResult.absentBeforeWidthPresent).toBe(false);
    expect(cancellationResult.absentBeforeWidthPriority).toBe("");
    expect(cancellationResult.presentBeforeAria).toBe(true);
    expect(cancellationResult.presentBeforeAriaValue).toBe("seed");
    expect(cancellationResult.duringAbsentCancelWidth).not.toBe(cancellationResult.absentBeforeWidth);
    expect(cancellationResult.afterAbsentCancelWidth).toBe(cancellationResult.absentBeforeWidth);
    expect(cancellationResult.afterAbsentCancelWidthPresent).toBe(cancellationResult.absentBeforeWidthPresent);
    expect(cancellationResult.afterAbsentCancelWidthPriority).toBe(cancellationResult.absentBeforeWidthPriority);
    expect(cancellationResult.afterAbsentCancelAriaPresent).toBe(cancellationResult.presentBeforeAria);
    expect(cancellationResult.afterAbsentCancelAriaValue).toBe(cancellationResult.presentBeforeAriaValue);
    expect(cancellationResult.absentBeforeStorage).toBeNull();
    expect(cancellationResult.afterAbsentCancelStorage).toBeNull();
    expect(cancellationResult.beforeValidStorage).toBeNull();
    expect(cancellationResult.duringValidWidth).not.toBe(cancellationResult.beforeWidth);
    expect(cancellationResult.afterValidWidth).not.toBe(cancellationResult.beforeWidth);
    expect(cancellationResult.commits).toBe(cancellationResult.beforeCommits + 1);
    const validPointerLayout = JSON.parse(cancellationResult.afterValidStorage || "null") as {
      left?: string;
      right?: string;
      activity?: string;
    } | null;
    expect(validPointerLayout).toEqual({
      left: cancellationResult.afterValidWidth,
      right: "412px",
      activity: "collapsed",
    });
    await page.waitForTimeout(40);
    const postGestureCounts = await page.evaluate(() => ({
      commits: (window as unknown as { __railCommitCount: number }).__railCommitCount,
      resizes: (window as unknown as { __resizeCount: number }).__resizeCount,
    }));
    const persistedPointerLayout = JSON.parse(cancellationResult.afterValidStorage || "null") as {
      left?: string;
      right?: string;
      activity?: string;
    } | null;
    expect(persistedPointerLayout?.left).toBe(cancellationResult.afterValidWidth);
    let expectedRailCommits = postGestureCounts.commits;
    let expectedResizeEvents = postGestureCounts.resizes;

    for (let cycle = 1; cycle <= REPLACEMENT_CYCLES; cycle += 1) {
      const resizeBeforeFreshBind = expectedResizeEvents;
      await page.evaluate((nextCycle) => {
        const current = document.querySelector<HTMLElement>("[data-editor-workbench]");
        if (!current) throw new Error("workbench form missing before replacement");
        const old = current;
        const next = current.cloneNode(true) as HTMLElement;
        next.setAttribute("data-cycle", String(nextCycle));
        if (nextCycle === 1) next.style.setProperty("--studio-left-width", "120px");
        old.replaceWith(next);
        (window as unknown as { __oldWorkbenchForm: HTMLElement; __oldWorkbenchHandle: HTMLElement }).__oldWorkbenchForm = old;
        (window as unknown as { __oldWorkbenchHandle: HTMLElement }).__oldWorkbenchHandle = old.querySelector<HTMLElement>("[data-studio-resizer]")!;
        const w = window as unknown as {
          __gosx_workbench_runtime_island_bindChrome?: (root: Document) => void;
          __gosx_workbench_runtime_island_bindRailResizers?: (root: Document) => void;
        };
        w.__gosx_workbench_runtime_island_bindChrome?.(document);
        w.__gosx_workbench_runtime_island_bindRailResizers?.(document);
        w.__gosx_workbench_runtime_island_bindChrome?.(document);
        w.__gosx_workbench_runtime_island_bindRailResizers?.(document);
      }, cycle);

      await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(2);
      await page.waitForFunction(() => {
        const old = (window as unknown as { __oldWorkbenchForm?: HTMLElement }).__oldWorkbenchForm;
        return !!old && !old.hasAttribute("data-gosx-studio-workbench-chrome-island-bound");
      });
      // A fresh form bind legitimately schedules one coalesced canvas refresh.
      // Drain it before measuring the next user delta so an initial layout
      // frame cannot satisfy the replacement's keyboard-resize assertion.
      await page.evaluate(() => new Promise<void>((resolve) => {
        if (typeof window.requestAnimationFrame === "function") {
          window.requestAnimationFrame(() => window.requestAnimationFrame(() => resolve()));
        } else {
          window.setTimeout(resolve, 40);
        }
      }));
      const postBindResizeBaseline = await page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount);
      // A fresh form bind schedules exactly one coalesced startup refresh;
      // record that settled baseline before testing detached-old inertness or
      // the replacement's user-driven keyboard delta.
      expect(postBindResizeBaseline).toBe(resizeBeforeFreshBind + 1);
      if (cycle === 1) {
        // Deliberately seed a stale inline width before the first fresh bind;
        // applyWorkbenchLayout must restore the committed pointer width from
        // storage, proving the successful commit survives a reload-like bind.
        expect(await page.evaluate(() => document.querySelector<HTMLElement>("[data-editor-workbench]")?.style.getPropertyValue("--studio-left-width"))).toBe(persistedPointerLayout?.left);
      }
      expectedResizeEvents = postBindResizeBaseline;
      await page.evaluate(() => {
        const oldHandle = (window as unknown as { __oldWorkbenchHandle: HTMLElement }).__oldWorkbenchHandle;
        oldHandle.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }));
      });
      expect(await page.evaluate(() => (window as unknown as { __railCommitCount: number }).__railCommitCount)).toBe(expectedRailCommits);
      expect(await page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount)).toBe(expectedResizeEvents);

      await page.locator("[data-editor-workbench] [data-studio-resizer]").dispatchEvent("keydown", { key: "ArrowRight", bubbles: true });
      expectedRailCommits += 1;
      expectedResizeEvents += 1;
      await expect.poll(() => page.evaluate(() => (window as unknown as { __railCommitCount: number }).__railCommitCount)).toBe(expectedRailCommits);
      await expect.poll(() => page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount)).toBe(expectedResizeEvents);
    }

    expect(pageErrors).toEqual([]);
    await testInfo.attach("workbench-runtime-teardown-evidence.json", {
      contentType: "application/json",
      body: Buffer.from(JSON.stringify({
        cycles: REPLACEMENT_CYCLES,
        railCommitCount: await page.evaluate(() => (window as unknown as { __railCommitCount: number }).__railCommitCount),
        resizeCount: await page.evaluate(() => (window as unknown as { __resizeCount: number }).__resizeCount),
        observerStats: await observerStats(page),
        persistence: {
          delayedLivePersistence,
          otherRailDuringGesture,
          otherRailPersistence,
          independentSaveDuringGesture,
          persistenceCancel,
          cancelResizeBaseline,
          persistenceCancelRefresh,
          validPointerLayout: persistedPointerLayout,
        },
      }, null, 2)),
    });
  });

  test("WASM-free canvas replacement disposes global callbacks and keeps resize + keyboard navigation singular", async ({ page }, testInfo) => {
    const bundle = JSON.stringify({
      camera: { x: 0, y: 0, z: 1 },
      objects: [
        { kind: "rect", id: "page:a", pickable: true, bounds: { minX: -220, maxX: -20, minY: -80, maxY: 80 } },
        { kind: "rect", id: "page:b", pickable: true, bounds: { minX: 20, maxX: 220, minY: -80, maxY: 80 } },
      ],
    });
    const canvasMarkup = (cycle: number) => `
      <div data-studio-site-map-board="true" data-studio-site-map-selected-node="page:a"></div>
      <section data-cycle="${cycle}" style="position: relative; width: 600px; height: 320px">
        <script type="application/json" data-gosx-canvas-bundle>${bundle}</script>
        <canvas data-gosx-canvas-wasm-free="true" width="600" height="320" tabindex="0" style="width: 600px; height: 320px"></canvas>
      </section>`;

    await page.setContent(canvasMarkup(0));
    await installObserverAccounting(page);
    await page.evaluate(() => {
      const stats = { activeResizeBindings: 0, paintCount: 0, stateCount: 0 };
      const originalAdd = window.addEventListener.bind(window);
      const originalRemove = window.removeEventListener.bind(window);
      (window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings = true;
      window.addEventListener = ((type: string, listener: EventListenerOrEventListenerObject, options?: boolean | AddEventListenerOptions) => {
        if ((window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings && type === "resize") stats.activeResizeBindings += 1;
        return originalAdd(type, listener, options);
      }) as typeof window.addEventListener;
      window.removeEventListener = ((type: string, listener: EventListenerOrEventListenerObject, options?: boolean | EventListenerOptions) => {
        if ((window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings && type === "resize") stats.activeResizeBindings = Math.max(0, stats.activeResizeBindings - 1);
        return originalRemove(type, listener, options);
      }) as typeof window.removeEventListener;
      (window as unknown as {
        __canvasTeardownStats: typeof stats;
        GoSXStudioCanvas2DPainterRuntime: { paint: () => void; renderCanvasBoardHTML: () => void; renderCanvasBoardThumbnails: () => void };
        GoSXStudioSiteMapRuntime: { setState: (root: HTMLElement, state: { selectedNode: string }) => void };
      }).__canvasTeardownStats = stats;
      (window as unknown as { GoSXStudioCanvas2DPainterRuntime: unknown }).GoSXStudioCanvas2DPainterRuntime = {
        paint: () => { stats.paintCount += 1; },
        renderCanvasBoardHTML: () => {},
        renderCanvasBoardThumbnails: () => {},
      };
      (window as unknown as { GoSXStudioSiteMapRuntime: unknown }).GoSXStudioSiteMapRuntime = {
        setState: (root: HTMLElement, state: { selectedNode: string }) => {
          stats.stateCount += 1;
          root.setAttribute("data-studio-site-map-selected-node", state.selectedNode);
        },
      };
    });
    await page.addScriptTag({ path: CANVAS_RUNTIME });
    await page.evaluate(() => {
      const runtime = (window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime?: { mountAll: () => void } }).GoSXStudioCanvasWASMFreeClientRuntime;
      if (!runtime) throw new Error("WASM-free client global missing");
      runtime.mountAll();
      runtime.mountAll();
    });
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasTeardownStats: { activeResizeBindings: number } }).__canvasTeardownStats.activeResizeBindings)).toBe(1);

    // Re-evaluating the script must reuse the original registry/API, not
    // install a second global listener or lose the mounted canvas controller.
    await page.addScriptTag({ path: CANVAS_RUNTIME });
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasTeardownStats: { activeResizeBindings: number } }).__canvasTeardownStats.activeResizeBindings)).toBe(1);

    for (let cycle = 1; cycle <= REPLACEMENT_CYCLES; cycle += 1) {
      await page.evaluate((nextCycle) => {
        const current = document.querySelector<HTMLElement>("section[data-cycle]");
        if (!current) throw new Error("canvas section missing before replacement");
        const oldCanvas = current.querySelector<HTMLCanvasElement>("canvas[data-gosx-canvas-wasm-free]");
        if (!oldCanvas) throw new Error("canvas missing before replacement");
        const next = current.cloneNode(true) as HTMLElement;
        next.setAttribute("data-cycle", String(nextCycle));
        current.replaceWith(next);
        (window as unknown as { __oldCanvas: HTMLCanvasElement }).__oldCanvas = oldCanvas;
        const runtime = (window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime?: { mountAll: () => void } }).GoSXStudioCanvasWASMFreeClientRuntime;
        (window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings = true;
        runtime?.mountAll();
        runtime?.mountAll();
        (window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings = false;
      }, cycle);

      await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
      await expect.poll(async () => (await page.evaluate(() => {
        const old = (window as unknown as { __oldCanvas?: HTMLCanvasElement }).__oldCanvas;
        const stats = (window as unknown as { __canvasTeardownStats: { activeResizeBindings: number } }).__canvasTeardownStats;
        return { oldBound: old?.hasAttribute("data-gosx-canvas-wasm-free-bound") ?? true, activeResizeBindings: stats.activeResizeBindings };
      }))).toEqual({ oldBound: false, activeResizeBindings: 1 });

      await page.evaluate(() => {
        const board = document.querySelector<HTMLElement>("[data-studio-site-map-board]");
        if (!board) throw new Error("canvas board missing before navigation precondition");
        board.setAttribute("data-studio-site-map-selected-node", "page:a");
      });
      const beforeOldInput = await page.evaluate(() => (window as unknown as { __canvasTeardownStats: { stateCount: number } }).__canvasTeardownStats.stateCount);
      await page.evaluate(() => {
        const old = (window as unknown as { __oldCanvas: HTMLCanvasElement }).__oldCanvas;
        old.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }));
        old.dispatchEvent(new Event("resize"));
      });
      await page.waitForTimeout(20);
      expect(await page.evaluate(() => (window as unknown as { __canvasTeardownStats: { stateCount: number } }).__canvasTeardownStats.stateCount)).toBe(beforeOldInput);

      await page.locator("canvas[data-gosx-canvas-wasm-free]").dispatchEvent("keydown", { key: "ArrowRight", bubbles: true });
      await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasTeardownStats: { stateCount: number } }).__canvasTeardownStats.stateCount)).toBe(beforeOldInput + 1);
      await expect.poll(() => page.evaluate(() => document.querySelector<HTMLElement>("[data-studio-site-map-board]")?.getAttribute("data-studio-site-map-selected-node"))).toBe("page:b");
      await page.evaluate(() => window.dispatchEvent(new Event("resize")));
      await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasTeardownStats: { activeResizeBindings: number } }).__canvasTeardownStats.activeResizeBindings)).toBe(1);
      await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasTeardownStats: { paintCount: number } }).__canvasTeardownStats.paintCount)).toBeGreaterThan(0);
    }

    const focusLossResult = await page.evaluate(() => {
      const canvas = document.querySelector<HTMLCanvasElement>("canvas[data-gosx-canvas-wasm-free]");
      if (!canvas) throw new Error("canvas missing before focus-loss cancellation");
      const control = (canvas as unknown as { __gosxStudioCanvasWASMFree?: { camera: () => { x: number; y: number; z: number } } }).__gosxStudioCanvasWASMFree;
      if (!control) throw new Error("canvas control API missing before focus-loss cancellation");
      const rect = canvas.getBoundingClientRect();
      const start = control.camera();
      canvas.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 31, clientX: rect.left + 100, clientY: rect.top + 2 }));
      canvas.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 31, clientX: rect.left + 130, clientY: rect.top + 2 }));
      const afterBlurMove = control.camera();
      window.dispatchEvent(new Event("blur"));
      canvas.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 31, clientX: rect.left + 230, clientY: rect.top + 2 }));
      const afterBlur = control.camera();

      canvas.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 32, clientX: rect.left + 100, clientY: rect.top + 2 }));
      canvas.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 32, clientX: rect.left + 130, clientY: rect.top + 2 }));
      const afterLostCaptureMove = control.camera();
      canvas.dispatchEvent(new PointerEvent("lostpointercapture", { bubbles: true, pointerId: 32 }));
      canvas.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 32, clientX: rect.left + 230, clientY: rect.top + 2 }));
      const afterLostCapture = control.camera();

      canvas.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, pointerId: 33, clientX: rect.left + 100, clientY: rect.top + 2 }));
      canvas.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 33, clientX: rect.left + 130, clientY: rect.top + 2 }));
      const afterHiddenMove = control.camera();
      const visibilityDescriptor = Object.getOwnPropertyDescriptor(document, "visibilityState");
      Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });
      document.dispatchEvent(new Event("visibilitychange"));
      if (visibilityDescriptor) Object.defineProperty(document, "visibilityState", visibilityDescriptor);
      else delete (document as unknown as { visibilityState?: unknown }).visibilityState;
      canvas.dispatchEvent(new PointerEvent("pointermove", { bubbles: true, pointerId: 33, clientX: rect.left + 230, clientY: rect.top + 2 }));
      const afterHidden = control.camera();
      return {
        start,
        afterBlurMove,
        afterBlur,
        afterLostCaptureMove,
        afterLostCapture,
        afterHiddenMove,
        afterHidden,
      };
    });
    expect(focusLossResult.afterBlurMove).not.toEqual(focusLossResult.start);
    expect(focusLossResult.afterBlur).toEqual(focusLossResult.afterBlurMove);
    expect(focusLossResult.afterLostCaptureMove).not.toEqual(focusLossResult.afterBlur);
    expect(focusLossResult.afterLostCapture).toEqual(focusLossResult.afterLostCaptureMove);
    expect(focusLossResult.afterHiddenMove).not.toEqual(focusLossResult.afterLostCapture);
    expect(focusLossResult.afterHidden).toEqual(focusLossResult.afterHiddenMove);

    await testInfo.attach("canvas-wasm-free-teardown-evidence.json", {
      contentType: "application/json",
      body: Buffer.from(JSON.stringify({
        cycles: REPLACEMENT_CYCLES,
        canvasStats: await page.evaluate(() => (window as unknown as { __canvasTeardownStats: unknown }).__canvasTeardownStats),
        observerStats: await observerStats(page),
      }, null, 2)),
    });
  });

  test("WASM-free canvas lifecycle stays bounded across multiple canvases, regrowth, and script re-execution", async ({ page }, testInfo) => {
    const bundle = JSON.stringify({
      camera: { x: 0, y: 0, z: 1 },
      objects: [{ kind: "rect", id: "page:a", pickable: true, bounds: { minX: -220, maxX: 220, minY: -80, maxY: 80 } }],
    });
    const sectionMarkup = (slot: string) => `
      <section data-canvas-slot="${slot}" style="position: relative; width: 600px; height: 320px">
        <script type="application/json" data-gosx-canvas-bundle>${bundle}</script>
        <canvas data-gosx-canvas-wasm-free="true" width="600" height="320" tabindex="0" style="width: 600px; height: 320px"></canvas>
      </section>`;

    await page.setContent(`<div data-studio-site-map-board="true"></div>${sectionMarkup("a")}${sectionMarkup("b")}`);
    await installObserverAccounting(page);
    await page.evaluate(() => {
      const stats = { activeResizeBindings: 0, stateCount: 0 };
      const originalAdd = window.addEventListener.bind(window);
      const originalRemove = window.removeEventListener.bind(window);
      (window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings = true;
      window.addEventListener = ((type: string, listener: EventListenerOrEventListenerObject, options?: boolean | AddEventListenerOptions) => {
        if ((window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings && type === "resize") stats.activeResizeBindings += 1;
        return originalAdd(type, listener, options);
      }) as typeof window.addEventListener;
      window.removeEventListener = ((type: string, listener: EventListenerOrEventListenerObject, options?: boolean | EventListenerOptions) => {
        if ((window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings && type === "resize") stats.activeResizeBindings = Math.max(0, stats.activeResizeBindings - 1);
        return originalRemove(type, listener, options);
      }) as typeof window.removeEventListener;
      (window as unknown as { __canvasMultiStats: typeof stats }).__canvasMultiStats = stats;
      (window as unknown as { GoSXStudioCanvas2DPainterRuntime: unknown }).GoSXStudioCanvas2DPainterRuntime = {
        paint: () => {}, renderCanvasBoardHTML: () => {}, renderCanvasBoardThumbnails: () => {},
      };
      (window as unknown as { GoSXStudioSiteMapRuntime: unknown }).GoSXStudioSiteMapRuntime = {
        setState: (root: HTMLElement, state: { selectedNode: string }) => {
          stats.stateCount += 1;
          root.setAttribute("data-studio-site-map-selected-node", state.selectedNode);
        },
      };
      (window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings = false;
      window.addEventListener("resize", () => {});
      (window as unknown as { __countRuntimeBindings: boolean }).__countRuntimeBindings = true;
    });
    await page.addScriptTag({ path: CANVAS_RUNTIME });
    await page.evaluate(() => {
      const w = window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime?: { mountAll: () => void }; __multiRuntimeRef?: unknown };
      if (!w.GoSXStudioCanvasWASMFreeClientRuntime) throw new Error("WASM-free client global missing");
      w.GoSXStudioCanvasWASMFreeClientRuntime.mountAll();
      w.GoSXStudioCanvasWASMFreeClientRuntime.mountAll();
      w.__multiRuntimeRef = w.GoSXStudioCanvasWASMFreeClientRuntime;
    });
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasMultiStats: { activeResizeBindings: number } }).__canvasMultiStats.activeResizeBindings)).toBe(2);

    await page.addScriptTag({ path: CANVAS_RUNTIME });
    await expect.poll(() => page.evaluate(() => {
      const w = window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime?: unknown; __multiRuntimeRef?: unknown };
      return w.GoSXStudioCanvasWASMFreeClientRuntime === w.__multiRuntimeRef;
    })).toBe(true);
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasMultiStats: { activeResizeBindings: number } }).__canvasMultiStats.activeResizeBindings)).toBe(2);

    const reparentBundle = JSON.stringify({
      camera: { x: 0, y: 0, z: 1 },
      objects: [{ kind: "rect", id: "page:b", pickable: true, bounds: { minX: -220, maxX: 220, minY: -80, maxY: 80 } }],
    });
    await page.evaluate((nextBundle) => {
      const oldSection = document.querySelector<HTMLElement>("section[data-canvas-slot='a']");
      const canvas = oldSection?.querySelector<HTMLCanvasElement>("canvas[data-gosx-canvas-wasm-free]");
      if (!oldSection || !canvas) throw new Error("primary canvas missing before connected reparent");
      const nextSection = document.createElement("section");
      nextSection.setAttribute("data-canvas-slot", "reparented");
      nextSection.style.position = "relative";
      nextSection.style.width = "600px";
      nextSection.style.height = "320px";
      const bundleScript = document.createElement("script");
      bundleScript.type = "application/json";
      bundleScript.setAttribute("data-gosx-canvas-bundle", "");
      bundleScript.textContent = nextBundle;
      nextSection.appendChild(bundleScript);
      nextSection.appendChild(canvas);
      document.body.appendChild(nextSection);
      (window as unknown as { __reparentedCanvas: HTMLCanvasElement }).__reparentedCanvas = canvas;
      (window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime: { mountAll: () => void } }).GoSXStudioCanvasWASMFreeClientRuntime.mountAll();
    }, reparentBundle);
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasMultiStats: { activeResizeBindings: number } }).__canvasMultiStats.activeResizeBindings)).toBe(2);
    await expect.poll(() => page.evaluate(() => {
      const canvas = (window as unknown as { __reparentedCanvas?: HTMLCanvasElement }).__reparentedCanvas;
      const control = canvas && (canvas as unknown as { __gosxStudioCanvasWASMFree?: { bundle: () => { objects?: Array<{ id: string }> } | null } }).__gosxStudioCanvasWASMFree;
      return {
        sameCanvas: !!canvas && canvas.closest("section")?.getAttribute("data-canvas-slot") === "reparented",
        bundleIds: control?.bundle()?.objects?.map((object) => object.id) ?? [],
      };
    })).toEqual({ sameCanvas: true, bundleIds: ["page:b"] });

    await page.evaluate(() => {
      const oldSection = document.querySelector<HTMLElement>("section[data-canvas-slot='b']");
      const oldCanvas = oldSection?.querySelector<HTMLCanvasElement>("canvas[data-gosx-canvas-wasm-free]");
      if (!oldSection || !oldCanvas) throw new Error("secondary canvas missing before removal");
      oldSection.remove();
      (window as unknown as { __oldMultiCanvas: HTMLCanvasElement }).__oldMultiCanvas = oldCanvas;
      (window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime: { mountAll: () => void } }).GoSXStudioCanvasWASMFreeClientRuntime.mountAll();
    });
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    await expect.poll(() => page.evaluate(() => {
      const old = (window as unknown as { __oldMultiCanvas?: HTMLCanvasElement }).__oldMultiCanvas;
      const stats = (window as unknown as { __canvasMultiStats: { activeResizeBindings: number } }).__canvasMultiStats;
      return { oldBound: old?.hasAttribute("data-gosx-canvas-wasm-free-bound") ?? true, activeResizeBindings: stats.activeResizeBindings };
    })).toEqual({ oldBound: false, activeResizeBindings: 1 });

    await page.evaluate((replacement) => {
      document.body.insertAdjacentHTML("beforeend", replacement);
      const runtime = (window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime: { mountAll: () => void } }).GoSXStudioCanvasWASMFreeClientRuntime;
      runtime.mountAll();
      runtime.mountAll();
    }, sectionMarkup("c"));
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __canvasMultiStats: { activeResizeBindings: number } }).__canvasMultiStats.activeResizeBindings)).toBe(2);
    const beforeOldInput = await page.evaluate(() => (window as unknown as { __canvasMultiStats: { stateCount: number } }).__canvasMultiStats.stateCount);
    await page.evaluate(() => {
      const old = (window as unknown as { __oldMultiCanvas: HTMLCanvasElement }).__oldMultiCanvas;
      old.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }));
      old.dispatchEvent(new Event("resize"));
    });
    expect(await page.evaluate(() => (window as unknown as { __canvasMultiStats: { stateCount: number } }).__canvasMultiStats.stateCount)).toBe(beforeOldInput);

    await testInfo.attach("canvas-wasm-free-multi-teardown-evidence.json", {
      contentType: "application/json",
      body: Buffer.from(JSON.stringify({
        activeCanvases: 2,
        observerStats: await observerStats(page),
        canvasStats: await page.evaluate(() => (window as unknown as { __canvasMultiStats: unknown }).__canvasMultiStats),
      }, null, 2)),
    });
  });

  test("WASM-free canvas teardown cancels queued RAF and fallback timer work", async ({ page }, testInfo) => {
    const bundle = JSON.stringify({ camera: { x: 0, y: 0, z: 1 }, objects: [] });
    const markup = `<section data-canvas-slot="pending" style="width: 600px; height: 320px">
      <script type="application/json" data-gosx-canvas-bundle>${bundle}</script>
      <canvas data-gosx-canvas-wasm-free="true" width="600" height="320" tabindex="0"></canvas>
    </section>`;

    async function setupPendingAccounting(useRAF: boolean): Promise<void> {
      await page.setContent(markup);
      await installObserverAccounting(page);
      await page.evaluate((useAnimationFrame) => {
        const stats = { pendingRAF: 0, cancelledRAF: 0, pendingTimers: 0 };
        const rafs = new Set<number>();
        let nextRAF = 1;
        if (useAnimationFrame) {
          window.requestAnimationFrame = ((callback: FrameRequestCallback) => {
            const id = nextRAF++;
            rafs.add(id);
            stats.pendingRAF += 1;
            return id;
          }) as typeof window.requestAnimationFrame;
          window.cancelAnimationFrame = ((id: number) => {
            if (rafs.delete(id)) {
              stats.pendingRAF -= 1;
              stats.cancelledRAF += 1;
            }
          }) as typeof window.cancelAnimationFrame;
        } else {
          window.requestAnimationFrame = undefined as unknown as typeof window.requestAnimationFrame;
        }
        const originalSetTimeout = window.setTimeout.bind(window);
        const originalClearTimeout = window.clearTimeout.bind(window);
        const timers = new Set<number>();
        window.setTimeout = ((handler: TimerHandler, timeout?: number) => {
          let id = 0;
          id = originalSetTimeout(() => {
            if (timers.delete(id)) stats.pendingTimers -= 1;
            if (typeof handler === "function") handler();
          }, timeout) as unknown as number;
          timers.add(id);
          stats.pendingTimers += 1;
          return id;
        }) as typeof window.setTimeout;
        window.clearTimeout = ((id: number) => {
          if (timers.delete(id)) stats.pendingTimers -= 1;
          originalClearTimeout(id);
        }) as typeof window.clearTimeout;
        (window as unknown as { __pendingCanvasStats: typeof stats }).__pendingCanvasStats = stats;
        (window as unknown as { GoSXStudioCanvas2DPainterRuntime: unknown }).GoSXStudioCanvas2DPainterRuntime = {
          paint: () => {}, renderCanvasBoardHTML: () => {}, renderCanvasBoardThumbnails: () => {},
        };
        (window as unknown as { GoSXStudioSiteMapRuntime: unknown }).GoSXStudioSiteMapRuntime = {
          setState: () => {},
        };
      }, useRAF);
      await page.addScriptTag({ path: CANVAS_RUNTIME });
      await page.evaluate(() => {
        (window as unknown as { GoSXStudioCanvasWASMFreeClientRuntime: { mountAll: () => void } }).GoSXStudioCanvasWASMFreeClientRuntime.mountAll();
      });
      await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(1);
    }

    await setupPendingAccounting(true);
    await expect.poll(() => page.evaluate(() => {
      const stats = (window as unknown as { __pendingCanvasStats: { pendingRAF: number; pendingTimers: number } }).__pendingCanvasStats;
      return { pendingRAF: stats.pendingRAF > 0, pendingTimers: stats.pendingTimers > 0 };
    })).toEqual({ pendingRAF: true, pendingTimers: true });
    await page.locator("section[data-canvas-slot='pending']").evaluate((section) => section.remove());
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(0);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __pendingCanvasStats: { pendingRAF: number; cancelledRAF: number; pendingTimers: number } }).__pendingCanvasStats)).toEqual({ pendingRAF: 0, cancelledRAF: 1, pendingTimers: 0 });

    await setupPendingAccounting(false);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __pendingCanvasStats: { pendingTimers: number } }).__pendingCanvasStats.pendingTimers)).toBeGreaterThan(0);
    await page.locator("section[data-canvas-slot='pending']").evaluate((section) => section.remove());
    await expect.poll(async () => (await observerStats(page)).activeObservers).toBe(0);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __pendingCanvasStats: { pendingTimers: number } }).__pendingCanvasStats.pendingTimers)).toBe(0);

    await testInfo.attach("canvas-wasm-free-pending-teardown-evidence.json", {
      contentType: "application/json",
      body: Buffer.from(JSON.stringify({
        queuedRAF: "cancelled",
        fallbackTimer: "cleared",
        observerStats: await observerStats(page),
        pendingStats: await page.evaluate(() => (window as unknown as { __pendingCanvasStats: unknown }).__pendingCanvasStats),
      }, null, 2)),
    });
  });
});
