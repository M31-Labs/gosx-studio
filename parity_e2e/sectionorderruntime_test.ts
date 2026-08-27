import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const operationRuntimeJS = readFileSync(
  path.resolve(__dirname, "../operationruntime/island_runtime.js"),
  "utf8",
);

const sectionOrderRuntimeJS = readFileSync(
  path.resolve(__dirname, "../sectionorderruntime/section_order_runtime.js"),
  "utf8",
);

function editorHTML(route = "/", optionalKeys: string[] = []): string {
  return `
    <main class="editor-workbench" data-gosx-studio-workbench="true">
      <form id="editor-main-form">
        <input type="hidden" name="csrf_token" value="csrf-existing" />
        <input id="editor-text" name="editor_text" value="editable text" />
        <section
          class="editor-preview-shell studio-page-canvas"
          data-gosx-studio-preview="true"
          data-gosx-studio-page-canvas="true"
          data-gosx-studio-preview-url="${route}?gosx-preview=1"
          data-gosx-studio-preview-state="ready"
        >
          <aside
            data-gosx-studio-section-order="true"
            data-gosx-studio-section-order-enabled="true"
            data-gosx-studio-section-order-route="/"
            data-gosx-studio-section-order-source-route="/"
            data-gosx-studio-section-order-preview-route="${route}"
          >
            <output data-gosx-studio-section-order-status="true" role="status" aria-live="polite">Drag sections or use Up and Down.</output>
            <ol data-gosx-studio-section-order-list="true">
              ${["hero", "gallery", "contact"].map((key, index) => `
                <li data-gosx-studio-section-order-item="${key}" data-gosx-studio-section-order-label="${key}" ${optionalKeys.includes(key) ? 'data-gosx-studio-section-order-preview-optional="true"' : ""}>
                  <button type="button" data-gosx-studio-section-order-handle="true" aria-label="Drag ${key}" aria-describedby="studio-section-order-guidance">Drag</button>
                  <span data-gosx-studio-section-order-name="true">${key}</span>
                  <button type="button" data-gosx-studio-section-order-move="up" ${index === 0 ? "disabled" : ""}>Up</button>
                  <button type="button" data-gosx-studio-section-order-move="down" ${index === 2 ? "disabled" : ""}>Down</button>
                </li>
              `).join("")}
            </ol>
            <p id="studio-section-order-guidance">Use the drag handle, arrow keys, or Up and Down controls to reorder sections.</p>
            <button type="button" data-gosx-studio-section-order-history="undo" data-gosx-studio-history-operation-id="" disabled>Undo reorder</button>
            <button type="button" data-gosx-studio-section-order-history="redo" data-gosx-studio-history-operation-id="" disabled>Redo reorder</button>
            <div
              data-gosx-studio-section-order-form="true"
              data-gosx-studio-operation-action="/admin/editor/__actions/operation"
              data-studio-document-revision="5"
              data-studio-target-head="head-a"
              data-studio-target-route="/"
              data-studio-target-page-id="home"
              data-studio-target-field="home.sections.order"
              data-studio-target-component=""
              hidden
            ></div>
            <script src="/_gosx/studio/section-order-runtime.js" defer data-gosx-studio-section-order-runtime="true"></script>
          </aside>
          <div data-studio-page-canvas-stage="true">
            <iframe title="preview" data-studio-preview-frame="true" src="${route}?gosx-preview=1"></iframe>
          </div>
        </section>
      </form>
      <form id="ordinary-save">
        <input type="hidden" name="gosx_studio_binding" value="home.sections.order" />
        <input type="hidden" name="gosx_studio_value" value="[&quot;hero&quot;,&quot;gallery&quot;,&quot;contact&quot;]" />
      </form>
    </main>
  `;
}

function previewHTML(keys: string[] = ["hero", "gallery", "contact"]): string {
  return `
    <!doctype html>
    <html>
      <body style="margin:0">
        ${keys.map((key) => `
          <section
            data-gosx-studio-section-key="${key}"
            data-gosx-studio-section-label="${key}"
            style="height:120px;border:1px solid #444;box-sizing:border-box"
          >${key}</section>
        `).join("")}
      </body>
    </html>
  `;
}

async function loadFixture(
  page: import("@playwright/test").Page,
  route = "/",
  options: { optionalKeys?: string[]; previewKeys?: string[]; previewDelayMs?: number } = {},
) {
  await page.route("http://127.0.0.1:4173/_gosx/studio/section-order-runtime.js", async (r) =>
    r.fulfill({ contentType: "text/javascript", body: sectionOrderRuntimeJS }));
  await page.route("http://127.0.0.1:4173/editor", async (r) =>
    r.fulfill({ contentType: "text/html", body: editorHTML(route, options.optionalKeys) }));
  await page.route("http://127.0.0.1:4173/**?gosx-preview=1**", async (r) =>
    {
      if (options.previewDelayMs) await new Promise((resolve) => setTimeout(resolve, options.previewDelayMs));
      await r.fulfill({ contentType: "text/html", body: previewHTML(options.previewKeys) });
    });
  await page.goto("http://127.0.0.1:4173/editor");
  await page.addScriptTag({ content: operationRuntimeJS });
}

type CapturedSectionOrderCommit = {
  kind: string;
  value: string;
  expectedTargetValue?: { present: boolean; value: string };
};

async function captureSectionOrderCommits(page: import("@playwright/test").Page) {
  await page.evaluate(() => {
    type OperationExtra = { expectedTargetValue?: { present: boolean; value: string } };
    type Operation = {
      commit: (kind: string, value: string, extra?: OperationExtra) => Promise<Response>;
    };
    type OperationRuntime = {
      create: (form: HTMLFormElement) => Operation;
    };
    type BrowserWindow = Window & {
      GoSXStudioOperationRuntime?: OperationRuntime;
      __sectionOrderCommitCaptures?: CapturedSectionOrderCommit[];
    };
    const browserWindow = window as unknown as BrowserWindow;
    const runtime = browserWindow.GoSXStudioOperationRuntime;
    if (!runtime) throw new Error("operation runtime is not mounted");
    const originalCreate = runtime.create;
    browserWindow.__sectionOrderCommitCaptures = [];
    runtime.create = (form) => {
      const operation = originalCreate(form);
      const originalCommit = operation.commit;
      operation.commit = (kind, value, extra) => {
        browserWindow.__sectionOrderCommitCaptures?.push({
          kind,
          value,
          expectedTargetValue: extra?.expectedTargetValue,
        });
        return originalCommit(kind, value, extra);
      };
      return operation;
    };
  });
}

async function capturedSectionOrderCommits(page: import("@playwright/test").Page): Promise<CapturedSectionOrderCommit[]> {
  return page.evaluate(() => {
    const browserWindow = window as unknown as { __sectionOrderCommitCaptures?: CapturedSectionOrderCommit[] };
    return browserWindow.__sectionOrderCommitCaptures ?? [];
  });
}

function multipartField(body: string, name: string): string {
  const escapedName = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`name="${escapedName}"\\r?\\n\\r?\\n([\\s\\S]*?)\\r?\\n--`).exec(body)?.[1] ?? "";
}

async function laneOrder(page: import("@playwright/test").Page): Promise<string[]> {
  return page.locator("[data-gosx-studio-section-order-item]").evaluateAll((nodes) =>
    nodes.map((node) => node.getAttribute("data-gosx-studio-section-order-item") ?? ""));
}

async function dragHandleToItem(
  page: import("@playwright/test").Page,
  key: string,
  targetKey: string,
  targetHalf: "top" | "bottom",
) {
  const handle = page.locator(`[data-gosx-studio-section-order-item="${key}"] [data-gosx-studio-section-order-handle]`);
  const target = page.locator(`[data-gosx-studio-section-order-item="${targetKey}"]`);
  const handleBox = await handle.boundingBox();
  const targetBox = await target.boundingBox();
  if (!handleBox || !targetBox) throw new Error("missing drag geometry");
  await page.mouse.move(handleBox.x + handleBox.width / 2, handleBox.y + handleBox.height / 2);
  await page.mouse.down();
  await page.mouse.move(targetBox.x + targetBox.width / 2, targetBox.y + (targetHalf === "top" ? 4 : targetBox.height - 4), { steps: 6 });
  await page.mouse.up();
}

test.describe("@smoke GoSXStudio section order runtime", () => {
  test("production HTML auto-loads the optional runtime without nesting a form", async ({ page }) => {
    await loadFixture(page);

    await expect(page.locator("script[data-gosx-studio-section-order-runtime='true']")).toHaveCount(1);
    await expect(page.locator("[data-gosx-studio-section-order]")).toHaveAttribute("data-gosx-studio-section-order-enabled", "true");
    expect(await page.locator("#editor-main-form").locator("form").count()).toBe(0);
    expect(await page.evaluate(() => typeof (window as unknown as { GoSXStudioSectionOrderRuntime?: { mount?: unknown } }).GoSXStudioSectionOrderRuntime?.mount === "function")).toBe(true);
  });

  test("iframe load revalidation enables ordering after a delayed preview response", async ({ page }) => {
    await loadFixture(page, "/", { previewDelayMs: 150 });
    await expect(page.locator("[data-gosx-studio-section-order]"))
      .toHaveAttribute("data-gosx-studio-section-order-enabled", "true");
    await expect(page.locator(`[data-gosx-studio-section-order-item="hero"] [data-gosx-studio-section-order-handle]`))
      .toBeEnabled();
  });

  test("mouse drag reorders sections through durable set-field and updates cursors", async ({ page }) => {
    let posted = "";
    await loadFixture(page);
    await captureSectionOrderCommits(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      posted = route.request().postData() ?? "";
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: 6,
            targetHead: "head-b",
            operationID: "op-reorder",
            value: JSON.stringify(["contact", "hero", "gallery"]),
          },
        }),
      });
    });

    await dragHandleToItem(page, "contact", "hero", "top");

    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section order saved.");
    expect(await laneOrder(page)).toEqual(["contact", "hero", "gallery"]);
    expect(posted).toContain("set-field");
    expect(posted).toContain("home.sections.order");
    expect(posted).toContain(JSON.stringify(["contact", "hero", "gallery"]));
    expect(posted).toContain("csrf-existing");
    expect(posted).toContain("gosx_studio_operation_id");
    expect(multipartField(posted, "gosx_studio_expected_target_value")).toBe(JSON.stringify({
      present: true,
      value: JSON.stringify(["hero", "gallery", "contact"]),
    }));
    expect(await capturedSectionOrderCommits(page)).toEqual([{
      kind: "set-field",
      value: JSON.stringify(["contact", "hero", "gallery"]),
      expectedTargetValue: { present: true, value: JSON.stringify(["hero", "gallery", "contact"]) },
    }]);
    await expect(page.locator("[data-gosx-studio-section-order-form]")).toHaveAttribute("data-studio-document-revision", "6");
    await expect(page.locator("[data-gosx-studio-section-order-form]")).toHaveAttribute("data-studio-target-head", "head-b");
    await expect(page.locator("[data-gosx-studio-section-order-history='undo']")).toHaveAttribute("data-gosx-studio-history-operation-id", "op-reorder");
    await expect(page.locator("#ordinary-save [name='gosx_studio_value']")).toHaveValue(JSON.stringify(["contact", "hero", "gallery"]));
    await expect(page.locator("form[data-gosx-studio-section-order-detached-form]")).toHaveCount(1);
  });

  test("keyboard handle Space Arrow Enter commits a reorder", async ({ page }) => {
    await loadFixture(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ data: { documentRevision: 6, targetHead: "head-b", operationID: "op-key", value: JSON.stringify(["gallery", "hero", "contact"]) } }),
      });
    });

    await page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-handle]`).focus();
    await page.keyboard.press("Space");
    await page.keyboard.press("ArrowUp");
    await page.keyboard.press("Enter");

    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section order saved.");
    expect(await laneOrder(page)).toEqual(["gallery", "hero", "contact"]);
  });

  test("focused section controls route keyboard history without stealing text undo", async ({ page }) => {
    let requests = 0;
    const operationIDs: string[] = [];
    await loadFixture(page);
    await captureSectionOrderCommits(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      requests += 1;
      const postData = route.request().postData() ?? "";
      const operationID = /name="gosx_studio_operation_id"\r?\n\r?\n([^\r\n]+)/.exec(postData)?.[1] ?? "";
      operationIDs.push(operationID);
      const values = [
        ["gallery", "hero", "contact"],
        ["hero", "gallery", "contact"],
        ["gallery", "hero", "contact"],
        ["hero", "gallery", "contact"],
      ];
      const value = values[Math.min(requests - 1, values.length - 1)];
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: 5 + requests,
            targetHead: `head-${requests}`,
            operationID: `op-history-${requests}`,
            value: JSON.stringify(value),
          },
        }),
      });
    });

    await page.locator(`[data-gosx-studio-section-order-item="hero"] [data-gosx-studio-section-order-move="down"]`).click();
    const heroHandle = page.locator(`[data-gosx-studio-section-order-item="hero"] [data-gosx-studio-section-order-handle]`);
    await expect(heroHandle).toBeEnabled();
    await heroHandle.focus();
    await expect(heroHandle).toBeFocused();
    await page.keyboard.press("Control+z");
    await expect(page.locator("[data-gosx-studio-section-order-status]"))
      .toHaveText("Section reorder history applied.");
    expect(await laneOrder(page)).toEqual(["hero", "gallery", "contact"]);
    await expect(page.locator("[data-gosx-studio-section-order-history='undo']")).toBeDisabled();
    await expect(page.locator("[data-gosx-studio-section-order-history='redo']")).toBeEnabled();

    await expect(heroHandle).toBeEnabled();
    await heroHandle.focus();
    await page.keyboard.press("Control+Shift+z");
    await expect(page.locator("[data-gosx-studio-section-order-status]"))
      .toHaveText("Section reorder history applied.");
    expect(await laneOrder(page)).toEqual(["gallery", "hero", "contact"]);
    await expect(page.locator("[data-gosx-studio-section-order-history='undo']")).toBeEnabled();
    await expect(page.locator("[data-gosx-studio-section-order-history='redo']")).toBeDisabled();

    const galleryDown = page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-move="down"]`);
    await expect(galleryDown).toBeEnabled();
    await galleryDown.click();
    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section moved down.");
    expect(await laneOrder(page)).toEqual(["hero", "gallery", "contact"]);
    expect(await capturedSectionOrderCommits(page)).toEqual([
      {
        kind: "set-field",
        value: JSON.stringify(["gallery", "hero", "contact"]),
        expectedTargetValue: { present: true, value: JSON.stringify(["hero", "gallery", "contact"]) },
      },
      {
        kind: "set-field",
        value: JSON.stringify(["hero", "gallery", "contact"]),
        expectedTargetValue: { present: true, value: JSON.stringify(["gallery", "hero", "contact"]) },
      },
    ]);

    await page.locator("#editor-text").focus();
    await page.keyboard.press("Control+z");
    expect(await laneOrder(page)).toEqual(["hero", "gallery", "contact"]);
    expect(requests).toBe(4);
    expect(operationIDs).toHaveLength(4);
    expect(operationIDs.every(Boolean)).toBe(true);
    expect(new Set(operationIDs).size).toBe(4);
  });

  test("pending section save disables a second action until the first response settles", async ({ page }) => {
    let requests = 0;
    const postedExpectedValues: string[] = [];
    let release: (() => void) | undefined;
    const delayed = new Promise<void>((resolve) => { release = resolve; });
    await loadFixture(page);
    await captureSectionOrderCommits(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      requests += 1;
      postedExpectedValues.push(multipartField(route.request().postData() ?? "", "gosx_studio_expected_target_value"));
      if (requests === 1) await delayed;
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: 5 + requests,
            targetHead: `head-pending-${requests}`,
            operationID: `op-pending-${requests}`,
            value: JSON.stringify(requests === 1 ? ["gallery", "hero", "contact"] : ["hero", "gallery", "contact"]),
          },
        }),
      });
    });

    const first = page.locator(`[data-gosx-studio-section-order-item="hero"] [data-gosx-studio-section-order-move="down"]`);
    const galleryDown = page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-move="down"]`);
    await first.click();
    await expect(first).toBeDisabled();
    await expect(galleryDown).toBeDisabled();
    expect(requests).toBe(1);
    release?.();
    await expect(page.locator("[data-gosx-studio-section-order-status]"))
      .toHaveText("Section moved down.");
    expect(await laneOrder(page)).toEqual(["gallery", "hero", "contact"]);

    await expect(galleryDown).toBeEnabled();
    await galleryDown.click();
    await expect.poll(() => requests).toBe(2);
    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section moved down.");
    expect(await capturedSectionOrderCommits(page)).toEqual([
      {
        kind: "set-field",
        value: JSON.stringify(["gallery", "hero", "contact"]),
        expectedTargetValue: { present: true, value: JSON.stringify(["hero", "gallery", "contact"]) },
      },
      {
        kind: "set-field",
        value: JSON.stringify(["hero", "gallery", "contact"]),
        expectedTargetValue: { present: true, value: JSON.stringify(["gallery", "hero", "contact"]) },
      },
    ]);
    expect(postedExpectedValues).toEqual([
      JSON.stringify({ present: true, value: JSON.stringify(["hero", "gallery", "contact"]) }),
      JSON.stringify({ present: true, value: JSON.stringify(["gallery", "hero", "contact"]) }),
    ]);
  });

  test("Escape cancels drag without a request", async ({ page }) => {
    let requests = 0;
    await loadFixture(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      requests += 1;
      await route.fulfill({ contentType: "application/json", body: JSON.stringify({ data: {} }) });
    });

    const handle = page.locator(`[data-gosx-studio-section-order-item="contact"] [data-gosx-studio-section-order-handle]`);
    const box = await handle.boundingBox();
    if (!box) throw new Error("missing handle geometry");
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();
    await page.keyboard.press("Escape");
    await page.mouse.up();

    expect(requests).toBe(0);
    expect(await laneOrder(page)).toEqual(["hero", "gallery", "contact"]);
    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section move canceled.");
  });

  test("unsupported route stays read-only and sends no request", async ({ page }) => {
    let requests = 0;
    await loadFixture(page, "/about");
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      requests += 1;
      await route.fulfill({ contentType: "application/json", body: JSON.stringify({ data: {} }) });
    });

    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section ordering applies only to the configured source route.");
    await expect(page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-handle]`)).toBeDisabled();
    await page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-move="up"]`).click({ force: true });
    expect(requests).toBe(0);
    expect(await laneOrder(page)).toEqual(["hero", "gallery", "contact"]);
  });

  test("server-declared optional hidden section stays in durable order while rendered controls remain usable", async ({ page }) => {
    await loadFixture(page, "/", { optionalKeys: ["contact"], previewKeys: ["hero", "gallery"] });

    await expect(page.locator("[data-gosx-studio-section-order-status]"))
      .toHaveText("Some optional sections are hidden in this preview; rendered sections can be reordered.");
    await expect(page.locator(`[data-gosx-studio-section-order-item="contact"] [data-gosx-studio-section-order-handle]`)).toBeDisabled();
    await expect(page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-move="up"]`)).toBeEnabled();
  });

  test("complete-order CAS includes an optional hidden member before the optimistic move", async ({ page }) => {
    let posted = "";
    await loadFixture(page, "/", { optionalKeys: ["contact"], previewKeys: ["hero", "gallery"] });
    await captureSectionOrderCommits(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      posted = route.request().postData() ?? "";
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: 6,
            targetHead: "head-cas-optional",
            operationID: "op-cas-optional",
            value: JSON.stringify(["hero", "contact", "gallery"]),
          },
        }),
      });
    });

    await page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-move="down"]`).click();
    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section moved down.");
    expect(multipartField(posted, "gosx_studio_expected_target_value")).toBe(JSON.stringify({
      present: true,
      value: JSON.stringify(["hero", "gallery", "contact"]),
    }));
    expect(await capturedSectionOrderCommits(page)).toEqual([{
      kind: "set-field",
      value: JSON.stringify(["hero", "contact", "gallery"]),
      expectedTargetValue: { present: true, value: JSON.stringify(["hero", "gallery", "contact"]) },
    }]);
  });

  test("missing required preview identity keeps the lane read-only", async ({ page }) => {
    await loadFixture(page, "/", { previewKeys: ["hero", "gallery"] });

    await expect(page.locator("[data-gosx-studio-section-order-status]"))
      .toHaveText("This preview omits one or more required sections, so ordering is read-only until all sections are rendered.");
    await expect(page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-handle]`)).toBeDisabled();
  });

  test("managed navigation disposes the old root and isolates a pending response", async ({ page }) => {
    let requests = 0;
    let releaseFirst: () => void = () => {};
    const firstResponse = new Promise<void>((resolve) => { releaseFirst = resolve; });
    await loadFixture(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      requests += 1;
      if (requests === 1) await firstResponse;
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: 6 + requests,
            targetHead: `head-navigation-${requests}`,
            operationID: `op-navigation-${requests}`,
            value: JSON.stringify(["gallery", "hero", "contact"]),
          },
        }),
      });
    });

    const oldHandle = page.locator(`[data-gosx-studio-section-order-item="hero"] [data-gosx-studio-section-order-move="down"]`);
    await oldHandle.click();
    await expect.poll(() => requests).toBe(1);
    await expect(page.locator("form[data-gosx-studio-section-order-detached-form]")).toHaveCount(1);

    await page.evaluate(() => {
      const current = document.querySelector<HTMLElement>("[data-gosx-studio-section-order]");
      if (!current) throw new Error("missing current section-order root");
      const oldRuntime = (current as HTMLElement & { __gosxSectionOrderRuntime?: { isDisposed?: () => boolean } }).__gosxSectionOrderRuntime;
      const next = current.cloneNode(true) as HTMLElement;
      const list = next.querySelector<HTMLOListElement>("[data-gosx-studio-section-order-list]");
      if (!list) throw new Error("missing replacement section-order list");
      const byKey = new Map(Array.from(list.children).map((item) => [item.getAttribute("data-gosx-studio-section-order-item"), item]));
      ["hero", "gallery", "contact"].forEach((key) => {
        const item = byKey.get(key);
        if (item) list.appendChild(item);
      });
      next.querySelector<HTMLElement>("[data-gosx-studio-section-order-status]")!.textContent = "Drag sections or use Up and Down.";
      next.querySelectorAll<HTMLElement>("[data-gosx-studio-section-order-history]").forEach((button) => {
        button.setAttribute("data-gosx-studio-history-operation-id", "");
        button.setAttribute("disabled", "disabled");
      });
      current.replaceWith(next);
      document.dispatchEvent(new Event("gosx:navigate"));
      const nextWithRuntime = next as HTMLElement & { __gosxSectionOrderRuntime?: unknown };
      const mountedOnce = nextWithRuntime.__gosxSectionOrderRuntime;
      document.dispatchEvent(new Event("gosx:navigate"));
      (window as unknown as { __oldSectionOrderDisposed?: boolean }).__oldSectionOrderDisposed = !!oldRuntime?.isDisposed?.();
      (window as unknown as { __newSectionOrderMountedOnce?: boolean }).__newSectionOrderMountedOnce = mountedOnce === nextWithRuntime.__gosxSectionOrderRuntime;
    });

    await expect(page.locator("form[data-gosx-studio-section-order-detached-form]")).toHaveCount(0);
    await expect(page.locator("[data-gosx-studio-section-order-overlay]")).toHaveCount(1);
    expect(await page.evaluate(() => (window as unknown as { __oldSectionOrderDisposed?: boolean }).__oldSectionOrderDisposed)).toBe(true);
    expect(await page.evaluate(() => (window as unknown as { __newSectionOrderMountedOnce?: boolean }).__newSectionOrderMountedOnce)).toBe(true);
    expect(await page.evaluate(() => {
      const root = document.querySelector<HTMLElement>("[data-gosx-studio-section-order]");
      return !!root && !!(root as HTMLElement & { __gosxSectionOrderRuntime?: { isDisposed?: () => boolean } }).__gosxSectionOrderRuntime;
    })).toBe(true);

    releaseFirst();
    await expect.poll(() => requests).toBe(1);
    expect(await laneOrder(page)).toEqual(["hero", "gallery", "contact"]);
    await expect(page.locator("#ordinary-save [name='gosx_studio_value']")).toHaveValue(JSON.stringify(["hero", "gallery", "contact"]));
    await expect(page.locator("form[data-gosx-studio-section-order-detached-form]")).toHaveCount(0);

    await page.locator(`[data-gosx-studio-section-order-item="hero"] [data-gosx-studio-section-order-move="down"]`).click();
    await expect.poll(() => requests).toBe(2);
    await expect(page.locator("form[data-gosx-studio-section-order-detached-form]")).toHaveCount(1);
  });

  test("CAS conflict preserves unsaved intent without advancing the committed baseline", async ({ page }) => {
    let requests = 0;
    const postedExpectedValues: string[] = [];
    await loadFixture(page);
    await captureSectionOrderCommits(page);
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      requests += 1;
      postedExpectedValues.push(multipartField(route.request().postData() ?? "", "gosx_studio_expected_target_value"));
      if (requests === 1) {
        await route.fulfill({ status: 409, contentType: "application/json", body: JSON.stringify({ error: "stale" }) });
        return;
      }
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: 7,
            targetHead: "head-after-conflict",
            operationID: "op-after-conflict",
            value: JSON.stringify(["hero", "gallery", "contact"]),
          },
        }),
      });
    });

    await page.locator(`[data-gosx-studio-section-order-item="hero"] [data-gosx-studio-section-order-move="down"]`).click();

    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section order changed on the server; your reorder remains unsaved.");
    expect(await laneOrder(page)).toEqual(["gallery", "hero", "contact"]);
    await expect(page.locator("#ordinary-save [name='gosx_studio_value']")).toHaveValue(JSON.stringify(["hero", "gallery", "contact"]));

    await page.locator(`[data-gosx-studio-section-order-item="gallery"] [data-gosx-studio-section-order-move="down"]`).click();
    await expect.poll(() => requests).toBe(2);
    await expect(page.locator("[data-gosx-studio-section-order-status]")).toHaveText("Section moved down.");
    expect(await laneOrder(page)).toEqual(["hero", "gallery", "contact"]);
    expect(await capturedSectionOrderCommits(page)).toEqual([
      {
        kind: "set-field",
        value: JSON.stringify(["gallery", "hero", "contact"]),
        expectedTargetValue: { present: true, value: JSON.stringify(["hero", "gallery", "contact"]) },
      },
      {
        kind: "set-field",
        value: JSON.stringify(["hero", "gallery", "contact"]),
        expectedTargetValue: { present: true, value: JSON.stringify(["hero", "gallery", "contact"]) },
      },
    ]);
    expect(postedExpectedValues).toEqual([
      JSON.stringify({ present: true, value: JSON.stringify(["hero", "gallery", "contact"]) }),
      JSON.stringify({ present: true, value: JSON.stringify(["hero", "gallery", "contact"]) }),
    ]);
    await expect(page.locator("#ordinary-save [name='gosx_studio_value']")).toHaveValue(JSON.stringify(["hero", "gallery", "contact"]));
  });
});
