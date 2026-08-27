import { expect, type Page, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../contenteditorruntime/content_editor.js"),
  "utf8",
);

const initialBlocks = {
  version: 1,
  title: "Opaque envelope",
  blocks: [
    { id: "a", type: "paragraph", text: "Alpha", extra: { keep: true } },
    { id: "b", type: "heading", text: "Beta", level: "3" },
    { id: "c", type: "custom-card", payload: { headline: "Opaque" } },
  ],
};

function editorHTML(value: string, id = "body"): string {
  return `
    <form>
      <select id="${id}Format" name="bodyFormat" data-content-editor-format="true">
        <option value="blocks" selected>GoSX blocks</option>
        <option value="mdpp">MDPP</option>
      </select>
      <div class="content-editor" data-content-editor="true" data-content-editor-media-list="media-list">
        <textarea id="${id}" name="body" data-content-editor-source="true">${escapeHTML(value)}</textarea>
        <div class="content-editor__toolbar">
          <button type="button" data-content-add="paragraph">Paragraph</button>
          <button type="button" data-content-add="heading">Heading</button>
          <button type="button" data-content-add="image">Image</button>
          <button type="button" data-content-add="flow">Flow</button>
        </div>
        <div class="content-block-list" data-content-editor-list="true"></div>
      </div>
      <datalist id="media-list"></datalist>
    </form>
  `;
}

function enhancedEditorHTML(value: string, action = "/save", extraFields = "", csrfToken: string | null = "test-csrf"): string {
  const csrfField = csrfToken === null
    ? ""
    : `<input type="hidden" name="csrf_token" value="${escapeHTML(csrfToken)}">`;
  return editorHTML(value)
    .replace("<form>", `<form action="${action}" method="post" data-content-editor-enhanced-submit="true">${csrfField}`)
    .replace("</form>", `${extraFields}<button type="submit">Save page</button><button type="submit" formaction="/archive" data-admin-confirm="Archive?">Archive</button></form>`);
}

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

async function mount(page: Page, html: string): Promise<void> {
  await page.setContent(html);
  await page.addScriptTag({ content: runtimeJS });
}

async function appendManagedRuntimeScript(page: Page, id: string): Promise<void> {
  await page.evaluate(({ id, source }) => {
    const script = document.createElement("script");
    script.id = id;
    script.setAttribute("data-gosx-script", "managed");
    script.setAttribute("data-gosx-script-load", "dom");
    script.setAttribute("data-gosx-script-loaded", "pending");
    script.textContent = source;
    document.head.appendChild(script);
  }, { id, source: runtimeJS });
}

async function mountAtURL(page: Page, html: string): Promise<void> {
  await page.route("http://127.0.0.1:4173/editor", async (route) => {
    await route.fulfill({ contentType: "text/html", body: html });
  });
  await page.goto("http://127.0.0.1:4173/editor");
  await page.addScriptTag({ content: runtimeJS });
}

test.describe("@smoke GoSXStudioContentEditorRuntime", () => {
  test("initializes without dirtying source and preserves unsupported JSON data", async ({ page }) => {
    const source = JSON.stringify(initialBlocks, null, 2);
    await mount(page, editorHTML(source));

    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "false");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-block-count", "3");
    await expect(page.locator("[data-content-editor-save-status]")).toBeVisible();
    await expect(page.locator("[data-content-editor-save-status]")).toHaveAttribute("role", "status");
    await expect(page.locator("[data-content-editor-save-label]")).toHaveText("Ready");
    await expect(page.locator("[data-content-block-type='custom-card']")).toBeVisible();
    await expect(page.locator("[data-content-block-type='custom-card'] select")).toBeDisabled();
    await expect(page.locator("#body")).toHaveValue(source);

    await page.getByRole("button", { name: "Duplicate" }).first().click();
    const parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.title).toBe("Opaque envelope");
    expect(parsed.blocks[0].extra).toEqual({ keep: true });
    expect(parsed.blocks.find((block: { id?: string }) => block.id === "c")?.payload).toEqual({ headline: "Opaque" });
    expect(parsed.blocks.find((block: { id?: string }) => block.id === "b")?.level).toBe("3");
  });

  test("fails closed on duplicate block IDs until the raw source is repaired", async ({ page }) => {
    const duplicate = {
      version: 1,
      title: "Duplicate envelope",
      envelopeExtra: { keep: true },
      blocks: [
        { id: "shared", type: "paragraph", text: "First", extra: { keep: "first" }, payload: { one: 1 } },
        { id: "shared", type: "paragraph", text: "Second", extra: { keep: "second" }, payload: { two: 2 } },
      ],
    };
    const source = JSON.stringify(duplicate, null, 2);
    await mount(page, editorHTML(source));

    const root = page.locator("[data-content-editor]");
    await expect(root).toHaveAttribute("data-content-editor-source-state", "invalid");
    await expect(root).toHaveAttribute("data-content-editor-source-error", /Duplicate block IDs detected/);
    await expect(root.locator("[data-content-editor-source-diagnostic]")).toContainText("Duplicate block IDs detected");
    await expect(page.locator("[data-content-editor-list]")).toBeHidden();
    await expect(page.locator("#body")).toHaveValue(source);
    expect(await page.locator("[data-content-block-id]").count()).toBe(0);

    const repaired = {
      ...duplicate,
      blocks: duplicate.blocks.map((block, index) => ({ ...block, id: index === 0 ? "first" : "second" })),
    };
    const repairedSource = JSON.stringify(repaired, null, 2);
    await page.locator("#body").fill(repairedSource);
    await expect(root).toHaveAttribute("data-content-editor-source-state", "ready");
    await expect(root).not.toHaveAttribute("data-content-editor-source-error");
    await expect(root.locator("[data-content-editor-source-diagnostic]")).toBeHidden();
    await expect(page.locator("[data-content-editor-list]")).toBeVisible();
    await expect(page.locator("#body")).toHaveValue(repairedSource);

    await page.locator("[data-content-block-id='second'] textarea").fill("Second changed");
    let edited = JSON.parse(await page.locator("#body").inputValue());
    expect(edited.title).toBe("Duplicate envelope");
    expect(edited.envelopeExtra).toEqual({ keep: true });
    expect(edited.blocks[0]).toMatchObject({ id: "first", text: "First", extra: { keep: "first" }, payload: { one: 1 } });
    expect(edited.blocks[1]).toMatchObject({ id: "second", text: "Second changed", extra: { keep: "second" }, payload: { two: 2 } });

    await page.locator("[data-content-block-id='second'] [data-content-editor-action='move-up']").click();
    edited = JSON.parse(await page.locator("#body").inputValue());
    expect(edited.blocks.map((block: { id?: string }) => block.id)).toEqual(["second", "first"]);
    expect(edited.blocks[0]).toMatchObject({ id: "second", text: "Second changed", extra: { keep: "second" }, payload: { two: 2 } });
    expect(edited.blocks[1]).toMatchObject({ id: "first", text: "First", extra: { keep: "first" }, payload: { one: 1 } });
  });

  test("searches blocks by type and content without changing draft state", async ({ page }) => {
    const source = JSON.stringify(initialBlocks);
    await mount(page, editorHTML(source));
    const search = page.locator("[data-content-editor-search]");
    await expect(search).not.toHaveAttribute("name");

    await search.fill("heading");
    expect(await page.locator("[data-content-block-id]").evaluateAll((rows) => rows.map((row) => row.getAttribute("data-content-block-id")))).toEqual(["b"]);
    await expect(page.locator("[data-content-editor-search-count]")).toHaveText("1 of 3 blocks");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "false");
    expect(await page.locator("#body").inputValue()).toBe(source);

    await search.fill("opaque");
    expect(await page.locator("[data-content-block-id]").evaluateAll((rows) => rows.map((row) => row.getAttribute("data-content-block-id")))).toEqual(["c"]);
    await search.fill("no such block");
    await expect(page.locator("[data-content-editor-search-empty]")).toBeVisible();
    await expect(page.locator("[data-content-editor-search-count]")).toHaveText("0 of 3 blocks");

    await page.getByRole("button", { name: "Clear search" }).click();
    expect(await page.locator("[data-content-block-id]").count()).toBe(3);
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "false");
  });

  test("keeps Search blocks Enter view-only without blocking Save or Ctrl/Cmd+S", async ({ page }) => {
    const source = JSON.stringify({
      version: 1,
      blocks: [
        { id: "heading", type: "heading", text: "Heading" },
        { id: "paragraph", type: "paragraph", text: "Paragraph" },
      ],
    });
    await mountAtURL(page, enhancedEditorHTML(source));
    await page.evaluate(() => {
      const win = window as unknown as { saveCount?: number; submitCount?: number };
      win.saveCount = 0;
      win.submitCount = 0;
      window.fetch = async () => {
        win.saveCount = (win.saveCount || 0) + 1;
        return new Response(JSON.stringify({ ok: true }), {
          headers: { "Content-Type": "application/json" },
        });
      };
      document.querySelector("form")?.addEventListener("submit", () => {
        win.submitCount = (win.submitCount || 0) + 1;
      }, true);
    });

    const root = page.locator("[data-content-editor]");
    const search = page.locator("[data-content-editor-search]");
    const save = page.getByRole("button", { name: "Save page" });
    await search.click();
    await search.fill("heading");
    await expect(search).toBeFocused();
    await expect(search).toHaveValue("heading");
    await expect(page.locator("[data-content-block-id]")).toHaveCount(1);
    await expect(root).toHaveAttribute("data-content-editor-search-query", "heading");
    await expect(root).toHaveAttribute("data-content-editor-search-visible-count", "1");
    await expect(root).toHaveAttribute("data-content-editor-dirty", "false");
    await expect(root).toHaveAttribute("data-content-editor-save-state", "idle");

    const composingEnterPrevented = await search.evaluate((node) => {
      const event = new KeyboardEvent("keydown", {
        bubbles: true,
        cancelable: true,
        isComposing: true,
        key: "Enter",
      });
      node.dispatchEvent(event);
      return event.defaultPrevented;
    });
    expect(composingEnterPrevented).toBe(true);
    await search.press("Enter");
    await expect(search).toBeFocused();
    await expect(search).toHaveValue("heading");
    await expect(page.locator("[data-content-block-id]")).toHaveCount(1);
    await expect(root).toHaveAttribute("data-content-editor-dirty", "false");
    await expect(root).toHaveAttribute("data-content-editor-save-state", "idle");
    expect(await page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(0);
    expect(await page.evaluate(() => (window as unknown as { submitCount?: number }).submitCount)).toBe(0);

    const heading = page.locator("[data-content-block-id='heading'] input");
    await heading.fill("Changed heading");
    await expect(root).toHaveAttribute("data-content-editor-dirty", "true");
    await search.click();
    await search.press("Enter");
    await expect(search).toBeFocused();
    await expect(search).toHaveValue("heading");
    await expect(page.locator("[data-content-block-id]")).toHaveCount(1);
    await expect(root).toHaveAttribute("data-content-editor-search-query", "heading");
    await expect(root).toHaveAttribute("data-content-editor-dirty", "true");
    await expect(root).toHaveAttribute("data-content-editor-save-state", "dirty");
    expect(await page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(0);
    expect(await page.evaluate(() => (window as unknown as { submitCount?: number }).submitCount)).toBe(0);

    await save.click();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saved");
    expect(await page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(1);

    await heading.fill("Changed heading again");
    await page.keyboard.press(process.platform === "darwin" ? "Meta+S" : "Control+S");
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saved");
    expect(await page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(2);
  });

  test("collapses blocks as view state and retains an accessible focus target", async ({ page }) => {
    const source = JSON.stringify(initialBlocks);
    await mount(page, editorHTML(source));
    const collapse = page.locator("[data-content-block-id='a'] [data-content-editor-collapse]");

    await collapse.click();
    await expect(page.locator("[data-content-block-id='a']")).toHaveAttribute("data-content-editor-collapsed", "true");
    await expect(page.locator("[data-content-block-id='a'] .content-block__body")).toBeHidden();
    await expect(page.locator("[data-content-block-id='a'] [data-content-editor-collapse]")).toHaveAttribute("aria-expanded", "false");
    await expect(page.locator("[data-content-block-id='a'] [data-content-editor-collapse]")).toBeFocused();
    expect(await page.locator("#body").inputValue()).toBe(source);
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "false");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-history-can-undo", "false");

    await page.locator("[data-content-block-id='a'] [data-content-editor-collapse]").click();
    await expect(page.locator("[data-content-block-id='a'] .content-block__body")).toBeVisible();
    await expect(page.locator("[data-content-block-id='a'] [data-content-editor-collapse]")).toHaveAttribute("aria-expanded", "true");
  });

  test("keeps tap-safe Up and Down controls available as the touch reorder fallback", async ({ page }) => {
    await mount(page, editorHTML(JSON.stringify(initialBlocks)));
    const row = page.locator("[data-content-block-id='b']");
    await expect(row.locator("[data-content-editor-touch-fallback]")).toHaveCount(2);
    await row.locator("[data-content-editor-action='move-up']").click();
    const parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.blocks.map((block: { id?: string }) => block.id)).toEqual(["b", "a", "c"]);
  });

  test("commits valid handle drag and leaves canceled drags unchanged", async ({ page }) => {
    await mount(page, editorHTML(JSON.stringify(initialBlocks)));

    const beforeCancel = await page.locator("#body").inputValue();
    await page.locator("[data-content-block-id='a'] [data-content-drag-handle]").dispatchEvent("dragstart", { dataTransfer: await page.evaluateHandle(() => new DataTransfer()) });
    await page.locator("[data-content-block-id='b']").dispatchEvent("dragover", { clientY: 1, dataTransfer: await page.evaluateHandle(() => new DataTransfer()) });
    await expect(page.locator("[data-content-block-id='b']")).toHaveAttribute("data-content-editor-drop-before", "true");
    await page.locator("[data-content-block-id='a'] [data-content-drag-handle]").dispatchEvent("dragend");
    await expect(page.locator("#body")).toHaveValue(beforeCancel);

    const transfer = await page.evaluateHandle(() => new DataTransfer());
    await page.locator("[data-content-block-id='a'] [data-content-drag-handle]").dispatchEvent("dragstart", { dataTransfer: transfer });
    await page.locator("[data-content-block-id='b']").dispatchEvent("dragover", { clientY: 1, dataTransfer: transfer });
    await page.locator("[data-content-block-id='b']").dispatchEvent("drop", { clientY: 1, dataTransfer: transfer });
    const parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.blocks.map((block: { id?: string }) => block.id)).toEqual(["a", "b", "c"]);

    const transferAfter = await page.evaluateHandle(() => new DataTransfer());
    await page.locator("[data-content-block-id='a'] [data-content-drag-handle]").dispatchEvent("dragstart", { dataTransfer: transferAfter });
    const box = await page.locator("[data-content-block-id='b']").boundingBox();
    await page.locator("[data-content-block-id='b']").dispatchEvent("dragover", { clientY: (box?.y ?? 0) + (box?.height ?? 100) - 1, dataTransfer: transferAfter });
    await page.locator("[data-content-block-id='b']").dispatchEvent("drop", { clientY: (box?.y ?? 0) + (box?.height ?? 100) - 1, dataTransfer: transferAfter });
    const moved = JSON.parse(await page.locator("#body").inputValue());
    expect(moved.blocks.map((block: { id?: string }) => block.id)).toEqual(["b", "a", "c"]);
  });

  test("inserts from palette and media drag at the selected or valid drop position", async ({ page }) => {
    await mount(page, editorHTML(JSON.stringify(initialBlocks)));

    await page.locator("[data-content-block-id='a']").click();
    await page.getByRole("button", { name: "Flow" }).click();
    let parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.blocks.map((block: { type?: string }) => block.type).slice(0, 3)).toEqual(["paragraph", "flow", "heading"]);

    await page.evaluate(() => {
      const transfer = new DataTransfer();
      transfer.setData("application/x-gosx-studio-media", JSON.stringify({ url: "/media/image.jpg", alt: "Alt", label: "Label", kind: "image" }));
      const handle = document.querySelector("[data-content-block-id='b'] [data-content-drag-handle]");
      const row = document.querySelector("[data-content-block-id='b']");
      row?.dispatchEvent(new DragEvent("dragover", { bubbles: true, cancelable: true, clientY: 9999, dataTransfer: transfer }));
      row?.dispatchEvent(new DragEvent("drop", { bubbles: true, cancelable: true, clientY: 9999, dataTransfer: transfer }));
      handle?.dispatchEvent(new DragEvent("dragend", { bubbles: true, cancelable: true }));
    });
    parsed = JSON.parse(await page.locator("#body").inputValue());
    const image = parsed.blocks.find((block: { type?: string }) => block.type === "image");
    expect(image).toMatchObject({ url: "/media/image.jpg", alt: "Alt" });
  });

  test("previews keyboard drag moves and only commits or cancels explicitly", async ({ page }) => {
    await mount(page, editorHTML(JSON.stringify(initialBlocks)));

    const sourceBeforeKeyboard = await page.locator("#body").inputValue();
    await page.locator("[data-content-block-id='b'] [data-content-drag-handle]").focus();
    await page.keyboard.press("Space");
    await page.keyboard.press("ArrowUp");
    await expect(page.locator("[data-content-block-id='b'] [data-content-drag-handle]")).toBeFocused();
    expect(await page.locator("#body").inputValue()).toBe(sourceBeforeKeyboard);
    expect(await page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-history-can-undo", "false");
    expect(await page.locator("[data-content-block-id]").evaluateAll((rows) => rows.map((row) => row.getAttribute("data-content-block-id")))).toEqual(["b", "a", "c"]);
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-keyboard-preview-index", "0");

    await page.keyboard.press("Escape");
    expect(await page.locator("#body").inputValue()).toBe(sourceBeforeKeyboard);
    expect(await page.locator("[data-content-block-id]").evaluateAll((rows) => rows.map((row) => row.getAttribute("data-content-block-id")))).toEqual(["a", "b", "c"]);
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-history-can-undo", "false");

    await page.locator("[data-content-block-id='b'] [data-content-drag-handle]").focus();
    await page.keyboard.press("Enter");
    await page.keyboard.press("ArrowUp");
    await page.keyboard.press("Space");
    let parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.blocks.map((block: { id?: string }) => block.id)).toEqual(["b", "a", "c"]);
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-history-can-undo", "true");

    await page.keyboard.press(process.platform === "darwin" ? "Meta+Z" : "Control+Z");
    parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.blocks.map((block: { id?: string }) => block.id)).toEqual(["a", "b", "c"]);

    await page.locator("[data-content-block-id='b'] [data-content-drag-handle]").focus();
    await page.keyboard.press("Space");
    await page.keyboard.press("ArrowUp");
    await page.locator("#bodyFormat").focus();
    await page.waitForTimeout(10);
    expect(await page.locator("#body").inputValue()).toBe(sourceBeforeKeyboard);
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-history-can-undo", "false");

    await page.locator("[data-content-block-id='a'] textarea").fill("Alpha typed");
    await page.keyboard.press(process.platform === "darwin" ? "Meta+Z" : "Control+Z");
    parsed = JSON.parse(await page.locator("#body").inputValue());
    expect(parsed.blocks[0].text).toBe("Alpha");
    await expect(page.locator("[data-content-block-id='a'] textarea")).toBeFocused();
  });

  test("keeps invalid and MDPP source in explicit safe source mode", async ({ page }) => {
    await mount(page, editorHTML("{ not json"));

    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-source-state", "invalid");
    await expect(page.locator("[data-content-editor-list]")).toBeHidden();
    await expect(page.locator("#body")).toHaveValue("{ not json");

    await page.locator("#body").fill("## MDPP heading");
    await page.locator("#bodyFormat").selectOption("mdpp");
    await expect(page.locator("#body")).toHaveValue("## MDPP heading");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "true");
  });

  test("keeps multiple editors isolated and reinit idempotent", async ({ page }) => {
    await mount(page, editorHTML(JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }), "one") + editorHTML(JSON.stringify({ version: 1, blocks: [{ id: "b1", type: "paragraph", text: "Two" }] }), "two"));

    await page.locator("[data-content-editor]").nth(0).locator("[data-content-add='heading']").click();
    expect(JSON.parse(await page.locator("#one").inputValue()).blocks).toHaveLength(2);
    expect(JSON.parse(await page.locator("#two").inputValue()).blocks).toHaveLength(1);

    await page.evaluate(() => document.dispatchEvent(new Event("gosx:render")));
    await expect(page.locator("[data-content-editor]").nth(0).locator("[data-content-block-id='a1']")).toHaveCount(1);
    await expect(page.locator("[data-content-editor]").nth(1).locator("[data-content-block-id='b1']")).toHaveCount(1);
  });

  test("reuses one runtime install while mounting a later soft root", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
    ));
    await page.addScriptTag({ content: runtimeJS });
    await page.evaluate(() => {
      const win = window as unknown as { saveCount?: number };
      win.saveCount = 0;
      window.fetch = async () => {
        win.saveCount = (win.saveCount || 0) + 1;
        return new Response(JSON.stringify({ ok: true }), {
          headers: { "Content-Type": "application/json" },
        });
      };
    });

    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed once");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]").first())
      .toHaveAttribute("data-content-editor-save-state", "saved");
    expect(await page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(1);

    await page.getByRole("button", { name: "Duplicate" }).first().click();
    expect(JSON.parse(await page.locator("#body").inputValue()).blocks).toHaveLength(2);

    await page.evaluate((html) => {
      document.body.insertAdjacentHTML("beforeend", html);
      document.dispatchEvent(new Event("gosx:render"));
    }, editorHTML(JSON.stringify({ version: 1, blocks: [{ id: "soft-a", type: "paragraph", text: "Soft root" }] }), "soft"));
    await expect(page.locator("[data-content-editor]")).toHaveCount(2);
    await page.locator("[data-content-editor]").nth(1).getByRole("button", { name: "Duplicate" }).click();
    expect(JSON.parse(await page.locator("#soft").inputValue()).blocks).toHaveLength(2);
  });

  test("marks managed scripts loaded and keeps one content runtime identity", async ({ page }) => {
    const source = JSON.stringify(initialBlocks, null, 2);
    await page.setContent(editorHTML(source));
    await appendManagedRuntimeScript(page, "content-runtime-first");

    await expect(page.locator("#content-runtime-first")).toHaveAttribute("data-gosx-script-loaded", "true");
    await expect(page.locator("[data-content-block-id]")).toHaveCount(3);
    await expect(page.locator("#body")).toHaveValue(source);
    await page.evaluate(() => {
      const win = window as unknown as { firstContentRuntime?: unknown; GoSXStudioContentEditorRuntime?: unknown };
      win.firstContentRuntime = win.GoSXStudioContentEditorRuntime;
    });

    await appendManagedRuntimeScript(page, "content-runtime-second");
    await expect(page.locator("#content-runtime-second")).toHaveAttribute("data-gosx-script-loaded", "true");
    expect(await page.evaluate(() => {
      const win = window as unknown as { firstContentRuntime?: unknown; GoSXStudioContentEditorRuntime?: unknown };
      return win.firstContentRuntime === win.GoSXStudioContentEditorRuntime;
    })).toBe(true);
    await expect(page.locator("[data-content-block-id]")).toHaveCount(3);

    await page.evaluate((html) => {
      document.body.insertAdjacentHTML("beforeend", html);
      document.dispatchEvent(new Event("gosx:render"));
    }, editorHTML(JSON.stringify({ version: 1, blocks: [{ id: "soft-content", type: "paragraph", text: "Soft root" }] }), "soft-content"));
    await expect(page.locator("[data-content-editor]")).toHaveCount(2);
    await expect(page.locator("#soft-content")).toHaveValue(JSON.stringify({ version: 1, blocks: [{ id: "soft-content", type: "paragraph", text: "Soft root" }] }));
  });

  test("shows truthful offline, validation, auth, conflict, malformed, and network retry states", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/save",
      `<label for="title">Title</label><input id="title" name="title" value="Original title">`,
    ));
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");

    await page.evaluate(() => Object.defineProperty(window.navigator, "onLine", { configurable: true, value: false }));
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "offline");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "true");
    await expect(page.getByRole("alert")).toContainText("Offline");
    await expect(page.getByRole("button", { name: "Retry save" })).toBeVisible();

    await page.evaluate(() => Object.defineProperty(window.navigator, "onLine", { configurable: true, value: true }));
    await page.evaluate(() => {
      let attempt = 0;
      window.fetch = async () => {
        attempt += 1;
        if (attempt === 1) {
          return new Response(JSON.stringify({ ok: false, message: "Validation failed.", fieldErrors: { title: "Title is required." } }), {
            status: 422,
            headers: { "Content-Type": "application/json" },
          });
        }
        if (attempt === 2) {
          return new Response(JSON.stringify({ ok: false, message: "Sign-in required." }), {
            status: 401,
            headers: { "Content-Type": "application/json" },
          });
        }
        if (attempt === 3) {
          return new Response(JSON.stringify({ ok: false, message: "Another editor saved this content." }), {
            status: 409,
            headers: { "Content-Type": "application/json" },
          });
        }
        if (attempt === 4) {
          return new Response("{", {
            headers: { "Content-Type": "application/json" },
          });
        }
        if (attempt === 5) {
          return new Response("<p>Saved maybe</p>", {
            headers: { "Content-Type": "text/html" },
          });
        }
        if (attempt === 6) throw new Error("offline");
        return new Response(JSON.stringify({ ok: true, message: "Saved." }), {
          headers: { "Content-Type": "application/json" },
        });
      };
    });

    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "true");
    await expect(page.getByRole("alert")).toContainText("Validation failed");
    await expect(page.locator("[data-content-editor-save-errors]")).toBeVisible();
    await expect(page.locator("[data-content-editor-save-errors]")).toContainText("title: Title is required.");
    await expect(page.locator("#title")).toHaveAttribute("aria-invalid", "true");

    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "auth");
    await expect(page.getByRole("alert")).toContainText("Sign-in required");

    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "conflict");
    await expect(page.getByRole("alert")).toContainText("Another editor saved");

    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "unconfirmed");
    await expect(page.getByRole("alert")).toContainText("structured action response");

    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "unconfirmed");
    await expect(page.getByRole("alert")).toContainText("structured action response");

    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(page.getByRole("alert")).toContainText("Save failed");

    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "saved");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "false");
    await expect(page.locator("[data-content-editor-save-status]")).toHaveAttribute("role", "status");
    await expect(page.locator("[data-content-editor-save-label]")).toHaveText("Saved");
    await expect(page.getByRole("button", { name: "Retry save" })).toBeHidden();
  });

  test("accepts a GoSX 303 JSON redirect and reaches the new entity", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] })));
    await page.route("http://127.0.0.1:4173/saved", async (route) => {
      await route.fulfill({ contentType: "text/html", body: "<main>Saved entity</main>" });
    });
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");
    await page.evaluate(() => {
      window.fetch = async () => new Response(JSON.stringify({ ok: true, redirect: "/saved" }), {
        status: 303,
        headers: { "Content-Type": "application/json" },
      });
    });

    await Promise.all([
      page.waitForURL("http://127.0.0.1:4173/saved"),
      page.getByRole("button", { name: "Save page" }).click(),
    ]);
    await expect(page.locator("main")).toHaveText("Saved entity");
  });

  test("uses the form action for a normal submitter, not the document URL", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] })));
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");
    await page.evaluate(() => {
      const win = window as unknown as { saveURLs?: string[]; saveCSRF?: string[] };
      win.saveURLs = [];
      win.saveCSRF = [];
      window.fetch = async (input, init) => {
        win.saveURLs?.push(String(input));
        const headers = (init?.headers || {}) as Record<string, string>;
        win.saveCSRF?.push(headers["X-CSRF-Token"] || "");
        return new Response(JSON.stringify({ ok: true }), {
          headers: { "Content-Type": "application/json" },
        });
      };
    });

    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "saved");
    expect(await page.evaluate(() => (window as unknown as { saveURLs?: string[] }).saveURLs)).toEqual([
      "http://127.0.0.1:4173/save",
    ]);
    expect(await page.evaluate(() => (window as unknown as { saveCSRF?: string[] }).saveCSRF)).toEqual(["test-csrf"]);
    expect(page.url()).toBe("http://127.0.0.1:4173/editor");
  });

  test("fails visibly without a form CSRF token instead of claiming saved", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/save",
      "",
      null,
    ));
    await page.evaluate(() => {
      const win = window as unknown as { fetchCount?: number };
      win.fetchCount = 0;
      window.fetch = async () => {
        win.fetchCount = (win.fetchCount || 0) + 1;
        return new Response(JSON.stringify({ ok: true }), {
          headers: { "Content-Type": "application/json" },
        });
      };
    });
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(page.getByRole("alert")).toContainText("no CSRF token");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "true");
    expect(await page.evaluate(() => (window as unknown as { fetchCount?: number }).fetchCount)).toBe(0);
  });

  test("rejects a cross-origin enhanced target before attaching the form token", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "https://external.example/save",
    ));
    await page.evaluate(() => {
      const win = window as unknown as { fetchCount?: number };
      win.fetchCount = 0;
      window.fetch = async () => {
        win.fetchCount = (win.fetchCount || 0) + 1;
        return new Response(JSON.stringify({ ok: true }), {
          headers: { "Content-Type": "application/json" },
        });
      };
    });
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(page.getByRole("alert")).toContainText("must stay on this site");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "true");
    expect(await page.evaluate(() => (window as unknown as { fetchCount?: number }).fetchCount)).toBe(0);
  });

  test("scopes Ctrl/Cmd+S to the CMS form and submits its canonical save button", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/save",
      `<label for="title">Title</label><input id="title" name="title" value="Original title">`,
    ));
    await page.evaluate(() => {
      const win = window as unknown as { saveURLs?: string[] };
      win.saveURLs = [];
      window.fetch = async (input) => {
        win.saveURLs?.push(String(input));
        return new Response(JSON.stringify({ ok: true }), {
          headers: { "Content-Type": "application/json" },
        });
      };
    });
    await page.locator("#title").fill("Updated title");
    await page.keyboard.press(process.platform === "darwin" ? "Meta+S" : "Control+S");

    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "saved");
    expect(await page.evaluate(() => (window as unknown as { saveURLs?: string[] }).saveURLs)).toEqual([
      "http://127.0.0.1:4173/save",
    ]);
  });

  test("prevents a second enhanced submit from falling back to native navigation", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] })));
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");
    await page.evaluate(() => {
      const win = window as unknown as { saveCount?: number; releaseSave?: () => void };
      win.saveCount = 0;
      window.fetch = async () => {
        win.saveCount = (win.saveCount || 0) + 1;
        return new Promise<Response>((resolve) => {
          win.releaseSave = () => resolve(new Response(JSON.stringify({ ok: true }), {
            headers: { "Content-Type": "application/json" },
          }));
        });
      };
    });

    const save = page.getByRole("button", { name: "Save page" });
    await save.click();
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "saving");
    await expect(save).toBeDisabled();
    await save.click({ force: true }).catch(() => undefined);
    await expect.poll(() => page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(1);
    expect(page.url()).toBe("http://127.0.0.1:4173/editor");

    await page.evaluate(() => (window as unknown as { releaseSave?: () => void }).releaseSave?.());
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "saved");
  });

  test("does not navigate away or lose newer edits made during an in-flight save", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/save",
      `<label for="title">Title</label><input id="title" name="title" value="Original title"><label for="slug">Slug</label><input id="slug" name="slug" value="original-slug"><label for="metaTitle">Meta title</label><input id="metaTitle" name="metaTitle" value="Original meta"><label for="published">Published</label><input id="published" name="published" type="checkbox" checked>`,
    ));
    await page.evaluate(() => {
      const win = window as unknown as { releaseSave?: () => void };
      window.fetch = async () => new Promise<Response>((resolve) => {
        win.releaseSave = () => resolve(new Response(JSON.stringify({ ok: true, redirect: "/saved" }), {
          status: 303,
          headers: { "Content-Type": "application/json" },
        }));
      });
    });
    await page.locator("[data-content-block-id='a1'] textarea").fill("Draft one");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "saving");
    await page.locator("[data-content-block-id='a1'] textarea").fill("Draft two while saving");
    await page.locator("#title").fill("New title while saving");
    await page.locator("#slug").fill("new-slug-while-saving");
    await page.locator("#metaTitle").fill("New meta while saving");
    await page.locator("#published").uncheck();
    await page.evaluate(() => (window as unknown as { releaseSave?: () => void }).releaseSave?.());
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "dirty");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "true");
    await expect(page.locator("[data-content-block-id='a1'] textarea")).toHaveValue("Draft two while saving");
    await expect(page.locator("#title")).toHaveValue("New title while saving");
    await expect(page.locator("#slug")).toHaveValue("new-slug-while-saving");
    await expect(page.locator("#metaTitle")).toHaveValue("New meta while saving");
    await expect(page.locator("#published")).not.toBeChecked();
    await expect(page.locator("[data-content-editor-save-detail-text]")).toContainText("Newer edits remain local");
    expect(page.url()).toBe("http://127.0.0.1:4173/editor");

    await page.evaluate(() => {
      window.fetch = async () => new Response(JSON.stringify({ ok: true }), {
        headers: { "Content-Type": "application/json" },
      });
    });
    await page.getByRole("button", { name: "Retry save" }).click();
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-save-state", "saved");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-dirty", "false");
    await expect(page.locator("[data-content-block-id='a1'] textarea")).toHaveValue("Draft two while saving");
  });

  test("ignores a late response after managed navigation replaces the editor root", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/save",
    ));
    await page.route("http://127.0.0.1:4173/saved", async (route) => {
      await route.fulfill({ contentType: "text/html", body: "<main>Saved entity</main>" });
    });
    await page.evaluate(() => {
      const win = window as unknown as { releaseSave?: () => void };
      window.fetch = async () => new Promise<Response>((resolve) => {
        win.releaseSave = () => resolve(new Response(JSON.stringify({ ok: true, redirect: "/saved" }), {
          status: 303,
          headers: { "Content-Type": "application/json" },
        }));
      });
    });
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "saving");

    await page.evaluate(() => {
      window.confirm = () => true;
      document.dispatchEvent(new CustomEvent("gosx:before-navigate", { cancelable: true }));
      document.querySelector("[data-content-editor]")?.remove();
      const replacement = document.createElement("main");
      replacement.id = "replacement-page";
      replacement.textContent = "Replacement page";
      document.body.append(replacement);
    });
    await page.evaluate(() => (window as unknown as { releaseSave?: () => void }).releaseSave?.());
    await page.waitForTimeout(20);
    expect(page.url()).toBe("http://127.0.0.1:4173/editor");
    await expect(page.locator("#replacement-page")).toHaveText("Replacement page");
  });

  test("does not retry a create after it already created an entity with newer local edits", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/admin/pages/__actions/create",
      `<label for="title">Title</label><input id="title" name="title" value="Original title"><label for="slug">Slug</label><input id="slug" name="slug" value="original-slug">`,
    ));
    await page.route("http://127.0.0.1:4173/admin/pages/page_1", async (route) => {
      await route.fulfill({ contentType: "text/html", body: "<main>Created page</main>" });
    });
    await page.evaluate(() => {
      const win = window as unknown as { saveCount?: number; releaseSave?: () => void };
      win.saveCount = 0;
      window.fetch = async () => {
        win.saveCount = (win.saveCount || 0) + 1;
        return new Promise<Response>((resolve) => {
          win.releaseSave = () => resolve(new Response(JSON.stringify({ ok: true, redirect: "/admin/pages/page_1" }), {
            status: 303,
            headers: { "Content-Type": "application/json" },
          }));
        });
      };
    });
    await page.locator("[data-content-block-id='a1'] textarea").fill("Create body");
    await page.locator("#title").fill("Created title");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "saving");
    await expect(page.locator("#title")).toHaveAttribute("readonly", "");
    await expect(page.locator("#slug")).toHaveAttribute("readonly", "");
    await expect(page.getByRole("button", { name: "Save page" })).toBeDisabled();
    // User edits are locked for a pending create. Exercise the defensive
    // newer-signature branch with a script-level race without simulating an
    // interaction that the lock intentionally prevents.
    await page.evaluate(() => {
      (document.querySelector("#title") as HTMLInputElement).value = "Newer title while creating";
      (document.querySelector("#slug") as HTMLInputElement).value = "newer-slug-while-creating";
    });
    await page.evaluate(() => (window as unknown as { releaseSave?: () => void }).releaseSave?.());

    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "dirty");
    await expect(page.locator("#title")).not.toHaveAttribute("readonly");
    await expect(page.locator("#slug")).not.toHaveAttribute("readonly");
    await expect(page.locator("[data-content-editor-save-created-link]"))
      .toBeVisible();
    await expect(page.locator("[data-content-editor-save-created-link]"))
      .toHaveAttribute("href", "http://127.0.0.1:4173/admin/pages/page_1");
    await expect(page.getByRole("button", { name: "Retry save" })).toBeHidden();
    expect(await page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(1);

    await page.getByRole("button", { name: "Save page" }).click();
    await page.waitForTimeout(10);
    expect(await page.evaluate(() => (window as unknown as { saveCount?: number }).saveCount)).toBe(1);
    await expect(page.locator("#title")).toHaveValue("Newer title while creating");
    await expect(page.locator("#slug")).toHaveValue("newer-slug-while-creating");

    await Promise.all([
      page.waitForURL("http://127.0.0.1:4173/admin/pages/page_1"),
      page.locator("[data-content-editor-save-created-link]").click(),
    ]);
    await expect(page.locator("main")).toHaveText("Created page");
  });

  test("restores a create form after a failed pending response", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/admin/pages/__actions/create",
      `<label for="title">Title</label><input id="title" name="title" value="Original title">`,
    ));
    await page.evaluate(() => {
      const win = window as unknown as { releaseSave?: () => void };
      window.fetch = async () => new Promise<Response>((resolve) => {
        win.releaseSave = () => resolve(new Response(JSON.stringify({ ok: false, message: "Title is required." }), {
          status: 422,
          headers: { "Content-Type": "application/json" },
        }));
      });
    });
    await page.locator("[data-content-block-id='a1'] textarea").fill("Create body");
    await page.locator("#title").fill("Create title");
    await page.getByRole("button", { name: "Save page" }).click();
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "saving");
    await expect(page.locator("[data-content-editor]")).toHaveAttribute("data-content-editor-create-lock", "true");
    await expect(page.locator("#title")).toHaveAttribute("readonly", "");
    await expect(page.locator("[data-content-editor-search]")).toHaveAttribute("readonly", "");
    await expect(page.getByRole("button", { name: "Save page" })).toBeDisabled();

    await page.evaluate(() => (window as unknown as { releaseSave?: () => void }).releaseSave?.());
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(page.locator("[data-content-editor]")).not.toHaveAttribute("data-content-editor-create-lock");
    await expect(page.locator("#title")).not.toHaveAttribute("readonly");
    await expect(page.locator("[data-content-editor-search]")).not.toHaveAttribute("readonly");
    await expect(page.getByRole("button", { name: "Save page" })).toBeEnabled();
    await expect(page.getByRole("button", { name: "Retry save" })).toBeVisible();
  });

  test("does not treat a structured auth redirect as a saved entity", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] })));
    await page.locator("[data-content-block-id='a1'] textarea").fill("Changed");
    await page.evaluate(() => {
      window.fetch = async () => new Response(JSON.stringify({ ok: true, redirect: "/login" }), {
        status: 303,
        headers: { "Content-Type": "application/json" },
      });
    });
    await page.getByRole("button", { name: "Save page" }).click();

    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-save-state", "auth");
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-dirty", "true");
    expect(page.url()).toBe("http://127.0.0.1:4173/editor");
  });

  test("guards dirty managed links and history while allowing explicit discard links", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(
      JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] }),
      "/save",
      `<nav><a data-gosx-link="true" href="/preview">Preview</a><a data-gosx-link="true" href="/editor-settings">Editor</a><a data-gosx-link="true" href="/content">Content</a><a data-content-editor-discard="true" href="/discarded">Discard changes</a></nav>`,
    ));
    for (const target of ["/preview", "/editor-settings", "/content", "/discarded"]) {
      await page.route(`http://127.0.0.1:4173${target}`, async (route) => {
        await route.fulfill({ contentType: "text/html", body: `<main>${target}</main>` });
      });
    }
    await page.locator("[data-content-block-id='a1'] textarea").fill("Unsaved content");
    await page.evaluate(() => {
      window.confirm = () => false;
    });
    expect(await page.evaluate(() => {
      const event = new CustomEvent("gosx:before-navigate", { cancelable: true });
      document.dispatchEvent(event);
      return event.defaultPrevented;
    })).toBe(true);

    for (const label of ["Preview", "Editor", "Content"]) {
      await page.getByRole("link", { name: label }).click();
      expect(page.url()).toBe("http://127.0.0.1:4173/editor");
      await expect(page.locator("[data-content-editor]"))
        .toHaveAttribute("data-content-editor-dirty", "true");
    }

    await page.evaluate(() => {
      window.history.pushState({}, "", "/editor?history-step=2");
      document.dispatchEvent(new CustomEvent("gosx:navigate", { detail: { url: window.location.href } }));
    });
    await page.evaluate(() => window.history.back());
    await page.waitForTimeout(20);
    expect(page.url()).toBe("http://127.0.0.1:4173/editor?history-step=2");
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-dirty", "true");

    await Promise.all([
      page.waitForURL("http://127.0.0.1:4173/discarded"),
      page.getByRole("link", { name: "Discard changes" }).click(),
    ]);
    await expect(page.locator("main")).toHaveText("/discarded");
  });

  test("leaves destructive alternate submitters on the native form path", async ({ page }) => {
    await mountAtURL(page, enhancedEditorHTML(JSON.stringify({ version: 1, blocks: [{ id: "a1", type: "paragraph", text: "One" }] })));
    const prevented = await page.evaluate(() => new Promise<boolean>((resolve) => {
      const form = document.querySelector("form");
      form?.addEventListener("submit", (event) => {
        window.setTimeout(() => resolve(event.defaultPrevented), 0);
      }, { once: true });
      (document.querySelector("button[formaction]") as HTMLButtonElement)?.click();
    }));
    expect(prevented).toBe(false);
  });
});
