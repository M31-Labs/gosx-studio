import { expect, type Page, type Request, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../contenteditorruntime/content_editor.js"),
  "utf8",
);

const editorURL = "http://127.0.0.1:4173/editor";
const saveURL = "http://127.0.0.1:4173/save";
const archiveURL = "http://127.0.0.1:4173/archive";

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function bodySource(text = "Original body"): string {
  return JSON.stringify({
    version: 1,
    blocks: [{ id: "a", type: "paragraph", text }],
  });
}

type FixtureOptions = {
  archive?: boolean;
  duplicateSubmitter?: boolean;
};

function editorFixtureHTML(key: string, options: FixtureOptions = {}): string {
  const archive = options.archive === true;
  const duplicateSubmitter = options.duplicateSubmitter === true;
  const ordinaryIntent = duplicateSubmitter
    ? '<input type="hidden" name="intent" value="ordinary">'
    : "";
  const submitterName = duplicateSubmitter ? ' name="intent" value="save"' : "";
  const archiveButton = archive
    ? '<button type="submit" formaction="/archive" data-admin-confirm="Archive?">Archive</button>'
    : "";

  return `
    <main>
      <form id="form-${escapeHTML(key)}" action="/save" method="post" data-content-editor-enhanced-submit="true">
        <input type="hidden" name="csrf_token" value="test-csrf">
        <input type="hidden" name="id" value="page-1">
        ${ordinaryIntent}
        <label for="${escapeHTML(key)}-title">Title</label>
        <input id="${escapeHTML(key)}-title" name="title" value="Original title">
        <label for="${escapeHTML(key)}-slug">Slug</label>
        <input id="${escapeHTML(key)}-slug" name="slug" value="page-one">
        <label for="${escapeHTML(key)}-meta-description">Meta description</label>
        <input id="${escapeHTML(key)}-meta-description" name="metaDescription" value="Original description">
        <select name="bodyFormat" data-content-editor-format="true">
          <option value="blocks" selected>GoSX blocks</option>
          <option value="mdpp">MDPP</option>
        </select>
        <div class="content-editor" data-content-editor="true" data-content-editor-media-list="${escapeHTML(key)}-media-list">
          <textarea id="${escapeHTML(key)}-body" name="body" data-content-editor-source="true">${escapeHTML(bodySource())}</textarea>
          <div class="content-editor__toolbar">
            <button type="button" data-content-add="paragraph">Paragraph</button>
          </div>
          <div class="content-block-list" data-content-editor-list="true"></div>
        </div>
        <datalist id="${escapeHTML(key)}-media-list"></datalist>
        <button type="submit"${submitterName}>Save ${escapeHTML(key)}</button>
        ${archiveButton}
      </form>
    </main>`;
}

async function mountAtURL(page: Page, html: string): Promise<void> {
  await page.route(editorURL, async (route) => {
    await route.fulfill({ contentType: "text/html", body: html });
  });
  await page.goto(editorURL);
  await page.addScriptTag({ content: runtimeJS });
}

function multipartValues(request: Request, fieldName: string): string[] {
  const contentType = request.headers()["content-type"] || "";
  const match = contentType.match(/boundary=(?:"([^"]+)"|([^;]+))/i);
  const boundary = match?.[1] || match?.[2];
  if (!boundary) return [];

  const body = request.postDataBuffer()?.toString("utf8") || "";
  return body.split(`--${boundary}`).slice(1).flatMap((part) => {
    const separator = part.indexOf("\r\n\r\n");
    if (separator < 0) return [];
    const headers = part.slice(0, separator);
    const name = headers.match(/(?:^|\r\n)Content-Disposition:[^\r\n]*\bname="([^"]+)"/i)?.[1];
    if (name !== fieldName) return [];
    return [part.slice(separator + 4).replace(/\r\n$/, "")];
  });
}

function jsonResponse(status: number, body: Record<string, unknown>) {
  return {
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  };
}

test.describe("@adversarial content-editor save boundary", () => {
  test("blocks Archive while Save is pending, performs one mutation, and restores controls", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("pending", { archive: true }));

    let saveRequests = 0;
    let archiveRequests = 0;
    let releaseSave: (() => void) | undefined;
    await page.route(saveURL, async (route) => {
      saveRequests += 1;
      await new Promise<void>((resolve) => {
        releaseSave = resolve;
      });
      await route.fulfill(jsonResponse(200, { ok: true, message: "Saved." }));
    });
    await page.route(archiveURL, async (route) => {
      archiveRequests += 1;
      // Keep a bad native submission from replacing the fixture document; the
      // request count is the mutation proof for this isolated test.
      await route.abort();
    });

    const root = page.locator("#form-pending [data-content-editor]");
    const save = page.getByRole("button", { name: "Save pending" });
    const archive = page.getByRole("button", { name: "Archive" });
    await page.locator("#pending-title").fill("Changed while Save is pending");
    await save.click();
    await expect.poll(() => saveRequests).toBe(1);
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saving");

    // Use the real native submit button. A pending enhanced Save must own the
    // form until it settles, so Archive cannot issue a competing mutation.
    await archive.click({ force: true }).catch(() => undefined);
    await page.waitForTimeout(100);
    expect(archiveRequests).toBe(0);

    releaseSave?.();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saved");
    await expect(save).toBeEnabled();
    await expect(archive).toBeEnabled();
    expect(saveRequests).toBe(1);
    expect(archiveRequests).toBe(0);
    expect(page.url()).toBe(editorURL);
  });

  test("includes both ordinary and submitter values for a duplicate-name button", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("payload", { duplicateSubmitter: true }));

    let intentValues: string[] = [];
    await page.route(saveURL, async (route) => {
      intentValues = multipartValues(route.request(), "intent");
      await route.fulfill(jsonResponse(200, { ok: true, message: "Saved." }));
    });

    // Real button activation supplies SubmitEvent.submitter and exercises the
    // browser's multipart FormData construction at the request boundary.
    await page.getByRole("button", { name: "Save payload" }).click();
    await expect.poll(() => intentValues).toEqual(["ordinary", "save"]);
  });

  test("focuses the first visible 422 field only when no newer typing occurred", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("focus"));

    let requestCount = 0;
    let releaseSecond: (() => void) | undefined;
    await page.route(saveURL, async (route) => {
      requestCount += 1;
      if (requestCount === 1) {
        await route.fulfill(jsonResponse(422, {
          ok: false,
          message: "Please correct the highlighted fields.",
          fieldErrors: { title: "Title is required." },
        }));
        return;
      }
      await new Promise<void>((resolve) => {
        releaseSecond = resolve;
      });
      await route.fulfill(jsonResponse(422, {
        ok: false,
        message: "Please correct the highlighted fields.",
        fieldErrors: { title: "Title is still invalid." },
      }));
    });

    const root = page.locator("#form-focus [data-content-editor]");
    const title = page.locator("#focus-title");
    const slug = page.locator("#focus-slug");
    await title.fill("");
    await page.getByRole("button", { name: "Save focus" }).click();
    await expect(title).toHaveAttribute("aria-invalid", "true");
    await expect(page.locator("[data-content-editor-save-errors]"))
      .toContainText("title: Title is required.");
    await expect(page.locator("[data-content-editor-save-errors]")).toBeVisible();
    await expect.soft(title, "without newer edits, focus the first visible invalid field")
      .toBeFocused();

    await title.fill("Fixed title");
    await slug.focus();
    await page.keyboard.press("Control+S");
    await expect.poll(() => requestCount).toBe(2);
    // Keep typing while the second response is pending. Its validation result
    // must not steal the active typing target.
    await page.keyboard.press("End");
    await page.keyboard.type("-typed");
    releaseSecond?.();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(slug).toBeFocused();
  });
});
