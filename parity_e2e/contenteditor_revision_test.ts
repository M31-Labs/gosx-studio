import { expect, type Page, type Request, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../contenteditorruntime/content_editor.js"),
  "utf8",
);

const editorURL = "http://127.0.0.1:4173/editor";
const saveURL = "http://127.0.0.1:4173/save";
const restoreURL = "http://127.0.0.1:4173/restoreRevision";

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
  revisionToken?: string | null;
  revisionRequired?: boolean;
  siblingRestore?: boolean;
  requiredTitle?: boolean;
  preRenderedRecoveryLinks?: boolean;
};

function editorFixtureHTML(key: string, options: FixtureOptions = {}): string {
  const hasRevision = options.revisionToken !== null;
  const revisionRequired = options.revisionRequired === true ? " required" : "";
  const requiredTitle = options.requiredTitle === true ? " required" : "";
  const revision = hasRevision
    ? `<input type="hidden" name="expectedRevision" data-content-editor-revision="true" value="${escapeHTML(options.revisionToken ?? "rev-1")}"${revisionRequired}>`
    : "";
  const siblingRestore = options.siblingRestore
    ? `<form id="${escapeHTML(key)}-history" method="post" action="/restoreRevision"><input type="hidden" name="csrf_token" value="test-csrf"><button type="submit" data-studio-submit-action="restoreRevision">Restore ${escapeHTML(key)}</button></form>`
    : "";
  const preRenderedRecoveryLinks = options.preRenderedRecoveryLinks
    ? `<div class="content-editor__save-status" data-content-editor-save-status="true" role="status" aria-live="polite" aria-atomic="true">
          <span class="content-editor__save-label" data-content-editor-save-label="true">Ready</span>
          <span class="content-editor__save-detail" data-content-editor-save-detail-text="true">Ready</span>
          <button type="button" class="content-editor__save-retry" data-content-editor-save-retry="true" hidden>Retry save</button>
          <div class="content-editor__save-errors" data-content-editor-save-errors="true" aria-label="Save errors" id="${escapeHTML(key)}-save-errors" tabindex="-1" hidden></div>
          <a class="content-editor__save-created-link" data-content-editor-save-created-link="true" data-content-editor-discard="true" tabindex="-1" hidden>Open saved item</a>
          <a class="content-editor__save-conflict-link" data-content-editor-save-conflict-link="true" target="_blank" rel="noopener" tabindex="-1" hidden>Open current version in new tab</a>
        </div>`
    : "";

  return `
    <main data-gosx-studio-backend-detail-renderer="gosx-studio">
      <form id="form-${escapeHTML(key)}" action="/save" method="post" data-content-editor-enhanced-submit="true">
        <input type="hidden" name="csrf_token" value="test-csrf">
        <input type="hidden" name="id" value="page-1">
        ${revision}
        <label for="${escapeHTML(key)}-title">Title</label>
        <input id="${escapeHTML(key)}-title" name="title" value="Original title"${requiredTitle}>
        <select name="bodyFormat" data-content-editor-format="true">
          <option value="blocks" selected>GoSX blocks</option>
          <option value="mdpp">MDPP</option>
        </select>
        <div class="content-editor" data-content-editor="true" data-content-editor-media-list="${escapeHTML(key)}-media-list">
          ${preRenderedRecoveryLinks}
          <textarea id="${escapeHTML(key)}-body" name="body" data-content-editor-source="true">${escapeHTML(bodySource())}</textarea>
          <div class="content-editor__toolbar">
            <button type="button" data-content-add="paragraph">Paragraph</button>
          </div>
          <div class="content-block-list" data-content-editor-list="true"></div>
        </div>
        <datalist id="${escapeHTML(key)}-media-list"></datalist>
        <button type="submit">Save ${escapeHTML(key)}</button>
      </form>
      ${siblingRestore}
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

test.describe("@revision content-editor save CAS boundary", () => {
  test("rotates only the marked token and keeps newer edits across two saves", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("rotate", { siblingRestore: true }));

    let requestCount = 0;
    let restoreRequests = 0;
    const tokenBodies: string[][] = [];
    let releaseFirst: (() => void) | undefined;
    await page.route(saveURL, async (route) => {
      requestCount += 1;
      tokenBodies.push(multipartValues(route.request(), "expectedRevision"));
      if (requestCount === 1) {
        await new Promise<void>((resolve) => {
          releaseFirst = resolve;
        });
        await route.fulfill(jsonResponse(200, {
          ok: true,
          message: "First edits saved.",
          values: { expectedRevision: "rev-2", title: "server title", body: "server body" },
        }));
        return;
      }
      await route.fulfill(jsonResponse(200, {
        ok: true,
        message: "Saved.",
        values: { expectedRevision: "rev-3", title: "another server title" },
      }));
    });
    await page.route(restoreURL, async (route) => {
      restoreRequests += 1;
      await route.abort();
    });

    const root = page.locator("#form-rotate [data-content-editor]");
    const title = page.locator("#rotate-title");
    const token = page.locator("[data-content-editor-revision='true']");
    const save = page.getByRole("button", { name: "Save rotate" });
    const restore = page.getByRole("button", { name: "Restore rotate" });

    await title.fill("Submitted title");
    await save.click();
    await expect.poll(() => requestCount).toBe(1);
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saving");
    await expect(restore).toBeDisabled();
    await restore.click({ force: true }).catch(() => undefined);
    expect(restoreRequests).toBe(0);

    // Existing-record fields remain editable while the request is pending.
    // This edit must stay local and make the first response leave the editor
    // dirty, while the returned token becomes the next request's token.
    await title.fill("Newer local title");
    releaseFirst?.();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "dirty");
    await expect(token).toHaveValue("rev-2");
    await expect(title).toHaveValue("Newer local title");
    await expect(restore).toBeEnabled();

    await save.click();
    await expect.poll(() => requestCount).toBe(2);
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saved");
    await expect(root).toHaveAttribute("data-content-editor-dirty", "false");
    await expect(token).toHaveValue("rev-3");
    await expect(title).toHaveValue("Newer local title");
    expect(tokenBodies).toEqual([["rev-1"], ["rev-2"]]);
  });

  test("preserves a stale token and draft on guarded conflict with a safe current-version link", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("conflict"));

    let requestToken: string[] = [];
    await page.route(saveURL, async (route) => {
      requestToken = multipartValues(route.request(), "expectedRevision");
      await route.fulfill(jsonResponse(409, {
        ok: false,
        conflict: true,
        message: "A newer version exists.",
        fieldErrors: { expectedRevision: "Refresh the current version before reconciling." },
        values: { expectedRevision: "server-rev-9" },
      }));
    });

    const root = page.locator("#form-conflict [data-content-editor]");
    const title = page.locator("#conflict-title");
    const token = page.locator("[data-content-editor-revision='true']");
    await title.fill("Draft that must remain local");
    await page.getByRole("button", { name: "Save conflict" }).click();

    await expect(root).toHaveAttribute("data-content-editor-save-state", "conflict");
    await expect(token).toHaveValue("rev-1");
    await expect(title).toHaveValue("Draft that must remain local");
    expect(requestToken).toEqual(["rev-1"]);
    await expect(root.locator("[data-content-editor-save-detail-text]")).toContainText("Compare and reconcile");
    await expect(root.locator("[data-content-editor-save-errors]")).toContainText("Refresh the current version before reconciling.");
    await expect(root.locator("[data-content-editor-save-errors]")).not.toContainText("expectedRevision:");

    const currentVersion = root.locator("[data-content-editor-save-conflict-link]");
    await expect(currentVersion).toBeVisible();
    await expect(currentVersion).toHaveAttribute("href", editorURL);
    await expect(currentVersion).toHaveAttribute("target", "_blank");
    await expect(currentVersion).toHaveAttribute("rel", "noopener");
    await expect(currentVersion).toHaveAttribute("tabindex", "0");
    await expect(root.locator("[data-content-editor-save-retry]")).toBeHidden();
  });

  test("normalizes pre-rendered recovery links without replacing their safe attributes", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("pre-rendered", { preRenderedRecoveryLinks: true }));

    const root = page.locator("#form-pre-rendered [data-content-editor]");
    const status = root.locator("[data-content-editor-save-status]");
    const createdLink = root.locator("[data-content-editor-save-created-link]");
    const conflictLink = root.locator("[data-content-editor-save-conflict-link]");
    await expect(createdLink).toHaveAttribute("tabindex", "0");
    await expect(createdLink).toHaveAttribute("data-content-editor-discard", "true");
    await expect(createdLink).toBeHidden();
    await expect(conflictLink).toHaveAttribute("tabindex", "0");
    await expect(conflictLink).toHaveAttribute("target", "_blank");
    await expect(conflictLink).toHaveAttribute("rel", "noopener");
    await expect(conflictLink).toBeHidden();
    expect(await status.locator(":scope > a").evaluateAll((nodes) => nodes.map((node) =>
      node.getAttribute("data-content-editor-save-created-link") === "true" ? "created" : "conflict")))
      .toEqual(["created", "conflict"]);

    await page.route(saveURL, async (route) => {
      await route.fulfill(jsonResponse(409, {
        ok: false,
        conflict: true,
        message: "A newer version exists.",
        values: { expectedRevision: "server-rev-9" },
      }));
    });
    await page.getByRole("button", { name: "Save pre-rendered" }).click();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "conflict");
    await expect(conflictLink).toBeVisible();
    await expect(conflictLink).toHaveAttribute("href", editorURL);
    await expect(conflictLink).toHaveAttribute("target", "_blank");
    await expect(conflictLink).toHaveAttribute("rel", "noopener");
    await expect(conflictLink).toHaveAttribute("tabindex", "0");
  });

  test("keeps unmarked hosts and user fields unchanged when a response has token-shaped values", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("legacy", { revisionToken: null }));

    await page.route(saveURL, async (route) => {
      await route.fulfill(jsonResponse(200, {
        ok: true,
        values: { expectedRevision: "must-not-be-created", title: "must-not-overwrite" },
      }));
    });

    const root = page.locator("#form-legacy [data-content-editor]");
    const title = page.locator("#legacy-title");
    await title.fill("Local title");
    await page.getByRole("button", { name: "Save legacy" }).click();

    await expect(root).toHaveAttribute("data-content-editor-save-state", "saved");
    await expect(title).toHaveValue("Local title");
    await expect(page.locator("[data-content-editor-revision='true']")).toHaveCount(0);
    await expect(root).toHaveAttribute("data-content-editor-dirty", "false");
  });

  test("keeps the old token and local edits when a guarded success omits next revision", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("missing", { revisionRequired: true }));

    let requestCount = 0;
    await page.route(saveURL, async (route) => {
      requestCount += 1;
      await route.fulfill(jsonResponse(200, { ok: true, message: "Accepted without token." }));
    });

    const root = page.locator("#form-missing [data-content-editor]");
    const title = page.locator("#missing-title");
    const token = page.locator("[data-content-editor-revision='true']");
    await title.fill("Local edit after submit");
    await page.getByRole("button", { name: "Save missing" }).click();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "unconfirmed");
    await expect(token).toHaveValue("rev-1");
    await expect(title).toHaveValue("Local edit after submit");
    await expect(root.locator("[data-content-editor-save-detail-text]")).toContainText("Earlier edits were accepted");
    await expect(root.locator("[data-content-editor-save-detail-text]")).toContainText("compare");
    const currentVersion = root.locator("[data-content-editor-save-conflict-link]");
    await expect(currentVersion).toBeVisible();
    await expect(currentVersion).toHaveAttribute("href", editorURL);
    await expect(currentVersion).toHaveAttribute("target", "_blank");
    await expect(currentVersion).toHaveAttribute("rel", "noopener");
    await expect(currentVersion).toHaveAttribute("tabindex", "0");
    await expect(root.locator("[data-content-editor-save-retry]")).toBeHidden();
    expect(requestCount).toBe(1);
  });

  test("Ctrl+S reports native validity and focuses the first invalid control", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("validity", { revisionToken: null, requiredTitle: true }));

    let requestCount = 0;
    await page.route(saveURL, async (route) => {
      requestCount += 1;
      await route.fulfill(jsonResponse(200, { ok: true }));
    });

    const root = page.locator("#form-validity [data-content-editor]");
    const title = page.locator("#validity-title");
    await title.fill("");
    await page.keyboard.press("Control+S");

    await expect(title).toBeFocused();
    expect(requestCount).toBe(0);
    await expect(root).toHaveAttribute("data-content-editor-dirty", "true");
  });

  test("focuses the save-error summary when only a hidden source field is invalid", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("source-error", { revisionToken: null }));
    await page.route(saveURL, async (route) => {
      await route.fulfill(jsonResponse(422, {
        ok: false,
        message: "Source needs attention.",
        fieldErrors: { body: "Body source is invalid." },
      }));
    });

    const root = page.locator("#form-source-error [data-content-editor]");
    const errors = root.locator("[data-content-editor-save-errors]");
    await page.getByRole("button", { name: "Save source-error" }).click();

    await expect(root).toHaveAttribute("data-content-editor-save-state", "failed");
    await expect(errors).toBeVisible();
    await expect(errors).toBeFocused();
    await expect(page.locator("#source-error-body")).toHaveAttribute("aria-invalid", "true");
  });

  test("explains that leaving during a pending save does not cancel the sent request", async ({ page }) => {
    await mountAtURL(page, editorFixtureHTML("navigation", { revisionToken: null }));

    let releaseSave: (() => void) | undefined;
    await page.route(saveURL, async (route) => {
      await new Promise<void>((resolve) => {
        releaseSave = resolve;
      });
      await route.fulfill(jsonResponse(200, { ok: true }));
    });

    const root = page.locator("#form-navigation [data-content-editor]");
    await page.locator("#navigation-title").fill("Pending navigation draft");
    await page.getByRole("button", { name: "Save navigation" }).click();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saving");

    const navigation = await page.evaluate(() => {
      const messages: string[] = [];
      window.confirm = (message?: string) => {
        messages.push(String(message ?? ""));
        return false;
      };
      const event = new CustomEvent("gosx:before-navigate", { cancelable: true });
      const followup = new CustomEvent("gosx:before-navigate", { cancelable: true });
      document.dispatchEvent(event);
      document.dispatchEvent(followup);
      return {
        prevented: event.defaultPrevented,
        followupPrevented: followup.defaultPrevented,
        messages,
      };
    });
    expect(navigation.prevented).toBe(true);
    expect(navigation.followupPrevented).toBe(true);
    expect(navigation.messages).toHaveLength(2);
    expect(navigation.messages[0]).toContain("will not cancel the save already sent");

    releaseSave?.();
    await expect(root).toHaveAttribute("data-content-editor-save-state", "saved");

    await page.locator("#navigation-title").fill("Dirty after pending save");
    const confirmed = await page.evaluate(() => {
      let calls = 0;
      window.confirm = () => {
        calls += 1;
        return true;
      };
      const accepted = new CustomEvent("gosx:before-navigate", { cancelable: true });
      const followup = new CustomEvent("gosx:before-navigate", { cancelable: true });
      document.dispatchEvent(accepted);
      document.dispatchEvent(followup);
      return { acceptedPrevented: accepted.defaultPrevented, followupPrevented: followup.defaultPrevented, calls };
    });
    expect(confirmed.acceptedPrevented).toBe(false);
    expect(confirmed.followupPrevented).toBe(false);
    expect(confirmed.calls).toBe(1);
  });
});
