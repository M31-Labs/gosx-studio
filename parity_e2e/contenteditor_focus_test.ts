import { expect, type Page, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../contenteditorruntime/content_editor.js"),
  "utf8",
);

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function editorHTML(value: string, id = "focus", blocksID = "focus-blocks"): string {
  return `
    <form id="${id}-form">
      <select id="${id}-format" name="${id}Format" data-content-editor-format="true">
        <option value="blocks" selected>GoSX blocks</option>
        <option value="mdpp">MDPP</option>
      </select>
      <div class="content-editor" data-content-editor="true" data-content-editor-media-list="${blocksID}">
        <textarea id="${id}-source" name="${id}Body" data-content-editor-source="true">${escapeHTML(value)}</textarea>
        <div class="content-editor__toolbar">
          <button type="button" data-content-add="paragraph">Paragraph</button>
          <button type="button" data-content-add="heading">Heading</button>
          <button type="button" data-content-add="image">Image</button>
        </div>
        <div class="content-block-list" data-content-editor-list="true"></div>
      </div>
      <datalist id="${blocksID}"></datalist>
    </form>
  `;
}

async function mount(page: Page, html: string): Promise<void> {
  await page.setContent(html);
  await page.addScriptTag({ content: runtimeJS });
  await expect(page.locator("[data-content-editor]").first())
    .toHaveAttribute("data-content-editor-source-state", "ready");
}

function sourceFor(blocks: unknown[], extra: Record<string, unknown> = {}): string {
  return JSON.stringify({ version: 1, ...extra, blocks });
}

test.describe("@smoke shared content editor focus continuity", () => {
  test("keeps native textarea focus, caret, typing, and undo after a real click", async ({ page }) => {
    await mount(page, editorHTML(sourceFor([{ id: "a", type: "paragraph", text: "Alpha" }])));

    const textarea = page.locator("[data-content-block-id='a'] textarea");
    await textarea.evaluate((node) => {
      (window as unknown as { __focusProbe?: Element }).__focusProbe = node;
    });
    await textarea.click();
    await expect(textarea).toBeFocused();
    await expect.poll(() => page.evaluate(() => {
      const browserWindow = window as unknown as { __focusProbe?: Element };
      return document.activeElement === browserWindow.__focusProbe;
    })).toBe(true);

    await page.keyboard.press("End");
    await page.keyboard.type("!");
    await expect(textarea).toHaveValue("Alpha!");
    expect(JSON.parse(await page.locator("#focus-source").inputValue()).blocks[0].text).toBe("Alpha!");

    await page.keyboard.press("Control+Z");
    await expect(textarea).toHaveValue("Alpha");
    expect(JSON.parse(await page.locator("#focus-source").inputValue()).blocks[0].text).toBe("Alpha");
  });

  test("tracks selection on click and keyboard focus without rerendering controls or dirtying draft", async ({ page }) => {
    await mount(page, editorHTML(sourceFor([
      { id: "a", type: "paragraph", text: "Alpha" },
      { id: "b", type: "paragraph", text: "Beta" },
    ])));

    const sourceBefore = await page.locator("#focus-source").inputValue();
    const firstTextarea = page.locator("[data-content-block-id='a'] textarea");
    await firstTextarea.click();
    await expect(page.locator("[data-content-block-id='a']"))
      .toHaveAttribute("data-content-editor-selected", "true");
    await expect(page.locator("[data-content-block-id='b']"))
      .toHaveAttribute("data-content-editor-selected", "false");

    const secondRow = page.locator("[data-content-block-id='b']");
    await page.keyboard.press("Tab");
    await expect(secondRow).toBeFocused();
    await expect(secondRow).toHaveAttribute("data-content-editor-selected", "true");
    await expect(page.locator("[data-content-block-id='a']"))
      .toHaveAttribute("data-content-editor-selected", "false");
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-dirty", "false");
    await expect(page.locator("[data-content-editor]"))
      .toHaveAttribute("data-content-editor-history-can-undo", "false");
    expect(await page.locator("#focus-source").inputValue()).toBe(sourceBefore);
  });

  test("moves focus to undo when removing the final block", async ({ page }) => {
    await mount(page, editorHTML(sourceFor([{ id: "only", type: "paragraph", text: "Only" }])));

    await page.locator("[data-content-block-id='only'] [data-content-editor-action='delete']").click();
    await expect(page.locator("[data-content-block-id]"))
      .toHaveCount(0);
    const undo = page.locator("[data-content-editor-action='undo']");
    await expect(undo).toBeEnabled();
    await expect(undo).toBeFocused();
    expect(JSON.parse(await page.locator("#focus-source").inputValue()).blocks).toEqual([]);
  });

  test("keeps focus on an enabled editor control when undo restores an empty snapshot", async ({ page }) => {
    await mount(page, editorHTML(sourceFor([])));

    await page.locator("[data-content-add='paragraph']").click();
    await expect(page.locator("[data-content-editor-selected='true'] [data-content-drag-handle]")).toBeFocused();

    const undo = page.locator("[data-content-editor-action='undo']");
    await expect(undo).toBeEnabled();
    await undo.click();
    await expect(page.locator("[data-content-block-id]")).toHaveCount(0);
    await expect.poll(() => page.evaluate(() => {
      const active = document.activeElement;
      return !!active
        && (active.matches("[data-content-editor-action='redo'], [data-content-add]")
          || active.matches("[data-content-editor-action='undo']"))
        && !(active as HTMLButtonElement).disabled
        && !(active as HTMLElement).hidden;
    })).toBe(true);

    const redo = page.locator("[data-content-editor-action='redo']");
    await expect(redo).toBeEnabled();
    await redo.click();
    await expect(page.locator("[data-content-block-id]")).toHaveCount(1);
    await expect(page.locator("[data-content-editor-selected='true'] [data-content-drag-handle]")).toBeFocused();
  });

  test("keeps focus valid when redo removes the final block after undo restores it", async ({ page }) => {
    await mount(page, editorHTML(sourceFor([{ id: "only", type: "paragraph", text: "Only" }])));

    const root = page.locator("[data-content-editor]");
    await root.locator("[data-content-block-id='only'] [data-content-editor-action='delete']").click();
    await expect(root.locator("[data-content-block-id]")).toHaveCount(0);

    const undo = root.locator("[data-content-editor-action='undo']");
    await expect(undo).toBeVisible();
    await expect(undo).toBeEnabled();
    await expect(undo).toBeFocused();

    await undo.click();
    await expect(root.locator("[data-content-block-id='only']")).toHaveCount(1);
    await expect(root.locator("[data-content-editor-selected='true'] [data-content-drag-handle]")).toBeFocused();

    const redo = root.locator("[data-content-editor-action='redo']");
    await expect(redo).toBeEnabled();
    await redo.click();
    await expect(root.locator("[data-content-block-id]")).toHaveCount(0);
    await expect(undo).toBeVisible();
    await expect(undo).toBeEnabled();
    await expect(undo).toBeFocused();
    expect(JSON.parse(await page.locator("#focus-source").inputValue()).blocks).toEqual([]);
  });

  test("keeps history focus scoped to the editor instance being restored", async ({ page }) => {
    await mount(page, editorHTML(sourceFor([]), "first", "media-first") + editorHTML(
      sourceFor([{ id: "second-only", type: "paragraph", text: "Second" }]),
      "second",
      "media-second",
    ));

    const first = page.locator("[data-content-editor]").nth(0);
    const second = page.locator("[data-content-editor]").nth(1);
    await first.locator("[data-content-add='paragraph']").click();
    await expect(first.locator("[data-content-editor-selected='true'] [data-content-drag-handle]")).toBeFocused();

    await first.locator("[data-content-editor-action='undo']").click();
    await expect(first.locator("[data-content-block-id]")).toHaveCount(0);
    await expect.poll(() => page.evaluate(() => {
      const active = document.activeElement;
      const editors = Array.from(document.querySelectorAll("[data-content-editor]"));
      return !!active
        && !!editors[0]?.contains(active)
        && active.matches("[data-content-editor-action='undo'], [data-content-editor-action='redo'], [data-content-add]")
        && !(active as HTMLButtonElement).disabled
        && !(active as HTMLElement).hidden;
    })).toBe(true);
    await expect(second.locator("[data-content-block-id='second-only']")).toHaveCount(1);
    await expect(second).toHaveAttribute("data-content-editor-history-can-undo", "false");
    await expect(second).toHaveAttribute("data-content-editor-history-can-redo", "false");
  });

  test("reveals and focuses selected results for filtered insert, duplicate, and remove", async ({ page }) => {
    await mount(page, editorHTML(sourceFor([
      { id: "alpha", type: "paragraph", text: "Alpha" },
      { id: "beta", type: "heading", text: "Beta", level: "2" },
    ])));

    const root = page.locator("[data-content-editor]");
    const search = root.locator("[data-content-editor-search]");
    await search.fill("Alpha");
    await expect(page.locator("[data-content-block-id]"))
      .toHaveCount(1);
    await root.locator("[data-content-add='paragraph']").click();
    await expect(search).toHaveValue("");
    await expect(page.locator("[data-content-block-id]"))
      .toHaveCount(3);
    const inserted = page.locator("[data-content-editor-selected='true']");
    await expect(inserted).toHaveCount(1);
    await expect(inserted.locator("[data-content-drag-handle]")).toBeFocused();

    await search.fill("Beta");
    const beta = page.locator("[data-content-block-id='beta']");
    await beta.locator("[data-content-editor-action='duplicate']").click();
    await expect(search).toHaveValue("Beta");
    const duplicate = page.locator("[data-content-editor-selected='true']");
    await expect(duplicate).toHaveCount(1);
    await expect(duplicate.locator("[data-content-drag-handle]")).toBeFocused();

    await duplicate.locator("[data-content-editor-action='delete']").click();
    await expect(search).toHaveValue("");

    await mount(page, editorHTML(sourceFor([
      { id: "alpha", type: "paragraph", text: "Alpha" },
      { id: "beta", type: "heading", text: "Beta", level: "2" },
    ])));
    const removeSearch = page.locator("[data-content-editor-search]");
    await removeSearch.fill("Beta");
    await page.locator("[data-content-block-id='beta'] [data-content-editor-action='delete']").click();
    await expect(removeSearch).toHaveValue("");
    await expect(page.locator("[data-content-block-id='alpha']"))
      .toBeVisible();
    await expect(page.locator("[data-content-editor-selected='true'] [data-content-drag-handle]"))
      .toBeFocused();
  });

  test("retains type-select focus and restores block focus through undo and redo", async ({ page }) => {
    const source = sourceFor([{ id: "a", type: "paragraph", text: "Alpha", extra: { keep: true } }]);
    await mount(page, editorHTML(source));

    const row = page.locator("[data-content-block-id='a']");
    const type = row.locator("select[aria-label='Block type']");
    await type.focus();
    await type.press("Home");
    await type.press("Enter");
    await expect(type).toBeFocused();
    let parsed = JSON.parse(await page.locator("#focus-source").inputValue());
    expect(parsed.blocks[0]).toMatchObject({ id: "a", type: "heading", text: "Alpha", extra: { keep: true } });

    await page.locator("[data-content-editor-action='undo']").click();
    await expect(page.locator("[data-content-block-id='a']"))
      .toHaveAttribute("data-content-block-type", "paragraph");
    await expect(page.locator("[data-content-block-id='a'] [data-content-drag-handle]")).toBeFocused();

    await page.locator("[data-content-editor-action='redo']").click();
    await expect(page.locator("[data-content-block-id='a']"))
      .toHaveAttribute("data-content-block-type", "heading");
    await expect(page.locator("[data-content-block-id='a'] [data-content-drag-handle]")).toBeFocused();
    parsed = JSON.parse(await page.locator("#focus-source").inputValue());
    expect(parsed.blocks[0]).toMatchObject({ id: "a", type: "heading", text: "Alpha", extra: { keep: true } });
  });

  test("uses collision-safe per-editor IDs for collapse and media alt targets", async ({ page }) => {
    const collidingIDs = sourceFor([
      { id: "a/b", type: "image", url: "/media/a.jpg", alt: "A" },
      { id: "a?b", type: "image", url: "/media/b.jpg", alt: "B" },
    ]);
    const sameID = sourceFor([{ id: "a/b", type: "image", url: "/media/shared.jpg", alt: "Shared" }]);
    await mount(page, editorHTML(collidingIDs, "first", "media-first") + editorHTML(sameID, "second", "media-second"));

    const ids = await page.locator("[data-content-editor]").evaluateAll((editors) => editors.flatMap((editor) => Array.from(editor.querySelectorAll("[data-content-block-id]")).map((row) => {
      const collapse = row.querySelector("[data-content-editor-collapse]");
      const media = row.querySelector("input[data-media-alt-target]");
      return {
        body: collapse?.getAttribute("aria-controls") ?? "",
        alt: media?.getAttribute("data-media-alt-target") ?? "",
      };
    })));
    expect(new Set(ids.map((item) => item.body)).size).toBe(ids.length);
    expect(new Set(ids.map((item) => item.alt)).size).toBe(ids.length);
    for (const item of ids) {
      expect(item.body).not.toBe("");
      expect(item.alt).toMatch(/^#/);
      expect(await page.locator(`#${item.body}`).count()).toBe(1);
      expect(await page.locator(`#${item.alt.slice(1)}`).count()).toBe(1);
    }
  });
});
