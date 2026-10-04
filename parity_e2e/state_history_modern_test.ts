import { expect, Page, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../hostruntime/assets/state_runtime.js"),
  "utf8",
);
const authoringRuntimeJS = readFileSync(
  path.resolve(__dirname, "../authoringruntime/island_runtime.js"), "utf8",
);

function editorHTML(): string {
  return `
    <button id="outside" type="button">Outside focus</button>
    <form id="editor-a" action="http://127.0.0.1:4173/save-a" method="post" data-gosx-studio-state="true" data-gosx-studio-client="true">
      <output data-gosx-studio-save-state="true">Saved</output>
      <span data-gosx-studio-save-detail="true">Ready</span>
      <input type="hidden" name="csrf_token" value="state-csrf-a" />
      <label>Title A <input name="title" value="Alpha" /></label>
      <button id="focus-a" type="button">Focus A</button>
      <button type="submit" formaction="http://127.0.0.1:4173/delete-a" name="danger" value="1">Delete A</button>
      <button type="submit" data-gosx-studio-save-button="true">Save A</button>
      <button type="button" data-gosx-studio-history-undo>Undo A</button>
      <button type="button" data-gosx-studio-history-redo>Redo A</button>
    </form>
    <form id="editor-b" action="http://127.0.0.1:4173/save-b" method="post" data-gosx-studio-state="true" data-gosx-studio-client="true">
      <output data-gosx-studio-save-state="true">Saved</output>
      <span data-gosx-studio-save-detail="true">Ready</span>
      <input type="hidden" name="csrf_token" value="state-csrf-b" />
      <label>Title B <input name="title" value="Beta" /></label>
      <button id="focus-b" type="button">Focus B</button>
      <button type="submit" formaction="http://127.0.0.1:4173/delete-b" name="danger" value="1">Delete B</button>
      <button type="submit" data-gosx-studio-save-button="true">Save B</button>
      <button type="button" data-gosx-studio-history-undo>Undo B</button>
      <button type="button" data-gosx-studio-history-redo>Redo B</button>
    </form>
  `;
}

async function mount(page: Page) {
  await page.route("http://127.0.0.1:4173/editor**", (route) =>
    route.fulfill({ contentType: "text/html", body: editorHTML() }));
  await page.goto("http://127.0.0.1:4173/editor");
  await page.addScriptTag({ content: runtimeJS });
}

function nextActionResult(page: Page) {
  return page.evaluate(() => new Promise<Record<string, unknown>>((resolve) => {
    document.addEventListener("gosxstudio:action-result", (event) => {
      resolve((event as CustomEvent).detail as Record<string, unknown>);
    }, { once: true });
  }));
}

test.describe("@smoke modern state runtime history shortcuts and save feedback", () => {
  for (const mode of ["replace", "inner"]) {
    for (const edit of ["addition", "removal", "multi-select", "empty multi-select", "reorder"]) {
      test(`fragment refresh preserves unrelated ${edit} (${mode})`, async ({ page }) => {
        await page.route("http://127.0.0.1:4173/editor", (route) => route.fulfill({
          contentType: "text/html",
          body: `<form data-gosx-studio-state="true" data-gosx-studio-workbench="true">
            <section id="unrelated"><input name="shared" value="Local baseline">
              <input name="removed" value="Keep me">
              <select name="tags" multiple><option value="a" selected>A</option><option value="b">B</option></select>
              <select name="empty" multiple><option value="b">B</option></select>
            </section>
            <section id="contact"><input name="shared" value="Old contact"><input name="server-removed" value="Old field"><input type="checkbox" name="contact"></section>
            <span data-gosx-studio-save-detail="true">Ready</span>
          </form><a href="/leave">Leave editor</a>`,
        }));
        await page.route("http://127.0.0.1:4173/fragment", (route) => route.fulfill({
          contentType: "text/html",
          body: `<section id="contact"><input name="shared" value="Server contact"><input type="checkbox" name="contact" checked><input name="server-added" value="Saved addition"></section>`,
        }));
        await page.goto("http://127.0.0.1:4173/editor");
        await page.addScriptTag({ content: runtimeJS });
        await page.addScriptTag({ content: authoringRuntimeJS });
        await page.evaluate((edit) => {
          const section = document.querySelector("#unrelated")!;
          (window as any).__originalFields = [...section.children];
          if (edit === "addition") section.insertAdjacentHTML("afterbegin", '<input name="new-field" value="Unsaved addition">');
          if (edit === "removal") section.querySelector('[name="removed"]')!.remove();
          if (edit === "multi-select") (section.querySelector('[name="tags"]') as HTMLSelectElement).options[1].selected = true;
          if (edit === "empty multi-select") (section.querySelector('[name="empty"]') as HTMLSelectElement).options[0].selected = true;
          if (edit === "reorder") section.appendChild(section.firstElementChild!);
          section.dispatchEvent(new Event("change", { bubbles: true }));
        }, edit);
        await expect(page.locator("form")).toHaveAttribute("data-gosx-studio-save-state", "dirty");
        await page.evaluate(async (mode) => {
          await (window as any).GoSXStudioAuthoringRuntime.handleResult({ ok: true, data: {
            message: "Contact saved.", fragmentURL: "/fragment",
            fragments: [{ selector: "#contact", mode }], changes: [],
          } });
        }, mode);
        await expect(page.locator('#contact [name="contact"]')).toBeChecked();
        await expect(page.locator('#contact [name="shared"]')).toHaveValue("Server contact");
        await expect(page.locator('[name="server-removed"]')).toHaveCount(0);
        await expect(page.locator("form")).toHaveAttribute("data-gosx-studio-save-state", "dirty");
        expect(await page.evaluate(() => {
          const event = new Event("beforeunload", { cancelable: true });
          window.dispatchEvent(event);
          return event.defaultPrevented;
        })).toBe(true);
        await page.evaluate(() => {
          (window as any).__discardPrompts = 0;
          window.confirm = () => { (window as any).__discardPrompts++; return false; };
        });
        await page.getByRole("link", { name: "Leave editor" }).click();
        expect(await page.evaluate(() => (window as any).__discardPrompts)).toBe(1);
        await expect(page).toHaveURL("http://127.0.0.1:4173/editor");
        await page.evaluate(() => {
          const section = document.querySelector("#unrelated")!;
          section.replaceChildren(...(window as any).__originalFields);
          (section.querySelector('[name="tags"]') as HTMLSelectElement).options[1].selected = false;
          (section.querySelector('[name="empty"]') as HTMLSelectElement).options[0].selected = false;
          section.dispatchEvent(new Event("change", { bubbles: true }));
        });
        await expect(page.locator("form")).toHaveAttribute("data-gosx-studio-save-state", "saved");
      });
    }
  }

  test("client save acknowledges refreshed fields while preserving edits made in flight", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor", (route) => route.fulfill({
      contentType: "text/html",
      body: `<form action="/save" method="post" data-gosx-studio-state="true" data-gosx-studio-client="true">
        <input name="csrf_token" value="own-form-token" type="hidden">
        <input name="title" value="Original">
        <section id="contact"><input type="checkbox" name="contact"></section>
        <button>Save</button>
      </form>`,
    }));
    await page.route("http://127.0.0.1:4173/fragment", (route) => route.fulfill({
      contentType: "text/html", body: '<section id="contact"><input type="checkbox" name="contact" checked></section>',
    }));
    await page.goto("http://127.0.0.1:4173/editor");
    await page.addScriptTag({ content: runtimeJS });
    await page.addScriptTag({ content: authoringRuntimeJS });
    await page.evaluate(() => {
      const originalFetch = window.fetch;
      window.fetch = async (url, init) => {
        if (init?.method !== "POST") return originalFetch(url, init);
        await new Promise<void>((resolve) => { (window as any).__releaseSave = resolve; });
        return new Response(JSON.stringify({ ok: true, data: {
          fragmentURL: "/fragment", fragments: [{ selector: "#contact" }], changes: [],
        } }), { headers: { "Content-Type": "application/json" } });
      };
    });
    await page.locator('[name="title"]').fill("Submitted");
    const result = nextActionResult(page);
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await expect(page.locator("form")).toHaveAttribute("data-gosx-studio-save-state", "saving");
    await page.locator('[name="title"]').fill("New local draft");
    await page.evaluate(() => {
      document.querySelector("form")!.insertAdjacentHTML("afterbegin", '<input name="new-field" value="New local field">');
      (window as any).__releaseSave();
    });
    await expect(result).resolves.toMatchObject({ ok: true });
    await expect(page.locator('[name="contact"]')).toBeChecked();
    await expect(page.locator('[name="title"]')).toHaveValue("New local draft");
    await expect(page.locator("form")).toHaveAttribute("data-gosx-studio-save-state", "dirty");
    expect(await page.evaluate(() => {
      const event = new Event("beforeunload", { cancelable: true });
      window.dispatchEvent(event);
      return event.defaultPrevented;
    })).toBe(true);
    await page.locator('[name="title"]').fill("Submitted");
    await page.locator('[name="new-field"]').evaluate((field) => {
      const form = field.closest("form")!;
      field.remove();
      form.dispatchEvent(new Event("change", { bubbles: true }));
    });
    await expect(page.locator("form")).toHaveAttribute("data-gosx-studio-save-state", "saved");
  });

  for (const client of [false, true]) {
    for (const override of ["local GET", "foreign GET", "foreign POST", ...(client ? [] : ["base-relative POST", "invalid GET"])]) {
      test(`session token stays private for ${client ? "client" : "native"} ${override} submitter`, async ({ page }) => {
        const target = override.startsWith("foreign") || override === "base-relative POST" ? "https://foreign.example/collect" : "http://127.0.0.1:4173/collect";
        const method = override.endsWith("GET") ? "get" : "post";
        await page.route("http://127.0.0.1:4173/editor", (route) => route.fulfill({
          contentType: "text/html",
          body: `${override === "base-relative POST" ? '<base href="https://foreign.example/">' : ""}
          <main data-gosx-studio-backend-editor-renderer="gosx-studio" data-gosx-studio-csrf-token="private-session-token">
            <form method="post" action="http://127.0.0.1:4173/save" target="result" data-gosx-studio-state="true" data-gosx-studio-client="${client}">
              <input name="title" value="Draft">
              <input name="action" value="edit" type="hidden">
              <button id="override" formmethod="${override === "invalid GET" ? "put" : method}" formaction="${override === "base-relative POST" ? "/collect" : target}">Override</button>
              <button id="save">Save</button>
            </form><iframe name="result"></iframe></main>`,
        }));
        await page.goto("http://127.0.0.1:4173/editor");
        await page.addScriptTag({ content: runtimeJS });
        await expect(page.locator('[name="csrf_token"]')).toHaveValue("private-session-token");
        if (client) {
          await page.evaluate(() => {
            (window as any).__requests = [];
            window.fetch = async (url, init) => {
              (window as any).__requests.push({ url: String(url), body: init?.body ? [...(init.body as FormData)] : [], headers: [...new Headers(init?.headers)] });
              return new Response('{"ok":true}', { headers: { "Content-Type": "application/json" } });
            };
          });
          const result = nextActionResult(page);
          await page.locator("#override").click();
          await expect(result).resolves.toMatchObject({ ok: method === "get" });
          const requests = await page.evaluate(() => (window as any).__requests);
          expect(requests).toHaveLength(method === "get" ? 1 : 0);
          expect(JSON.stringify(requests)).not.toMatch(/csrf_token|private-session-token|x-csrf-token/i);
          const saved = nextActionResult(page);
          await page.locator("#save").click();
          await expect(saved).resolves.toMatchObject({ ok: true });
          expect(await page.evaluate(() => (window as any).__requests.at(-1).body)).toContainEqual(["csrf_token", "private-session-token"]);
        } else {
          await page.route(`${target}**`, (route) => route.fulfill({ contentType: "text/html", body: "Submitted" }));
          const submitted = page.waitForRequest((request) => request.url().startsWith(target));
          await page.locator("#override").click();
          const request = await submitted;
          expect(request.method()).toBe(method.toUpperCase());
          expect(request.url() + (request.postData() || "")).not.toMatch(/csrf_token|private-session-token/);
          await page.route("http://127.0.0.1:4173/save", (route) => route.fulfill({ contentType: "text/html", body: "Saved" }));
          const saved = page.waitForRequest("http://127.0.0.1:4173/save");
          await page.locator("#save").click();
          expect((await saved).postData()).toContain("csrf_token=private-session-token");
        }
      });
    }
  }

  test("server editor session populates only its same-origin mutation forms", async ({ page }) => {
    const markup = editorHTML().replace('<input type="hidden" name="csrf_token" value="state-csrf-a" />', "");
    await page.route("http://127.0.0.1:4173/editor**", (route) => route.fulfill({
      contentType: "text/html",
      body: `<div data-gosx-studio-backend-editor-renderer="gosx-studio" data-gosx-studio-csrf-token="server-editor-token">
        ${markup}
        <form id="preview-get" method="get" action="/preview"></form>
        <form id="foreign-post" method="post" action="https://foreign.example/save"></form>
      </div><form id="outside-post" method="post" action="/outside"></form>`,
    }));
    await page.goto("http://127.0.0.1:4173/editor");
    await page.addScriptTag({ content: runtimeJS });
    await expect(page.locator("#editor-a input[name='csrf_token']")).toHaveValue("server-editor-token");
    await expect(page.locator("#editor-b input[name='csrf_token']")).toHaveValue("state-csrf-b");
    for (const id of ["preview-get", "foreign-post", "outside-post"]) {
      await expect(page.locator(`#${id} input[name='csrf_token']`)).toHaveCount(0);
    }
    await page.evaluate(() => {
      window.fetch = async (_url: RequestInfo | URL, init?: RequestInit) => {
        const body = init?.body as FormData;
        (window as unknown as { __editorSessionRequest: unknown }).__editorSessionRequest = {
          token: body.get("csrf_token"),
          header: new Headers(init?.headers).get("X-CSRF-Token"),
        };
        return new Response(JSON.stringify({ ok: true, message: "Saved." }), {
          headers: { "Content-Type": "application/json" },
        });
      };
    });
    await page.getByLabel("Title A").fill("First editor save");
    const result = nextActionResult(page);
    await page.locator("#editor-a [data-gosx-studio-save-button]").click();
    await expect(result).resolves.toMatchObject({ ok: true });
    expect(await page.evaluate(() => (window as unknown as { __editorSessionRequest: unknown }).__editorSessionRequest)).toEqual({
      token: "server-editor-token", header: "server-editor-token",
    });
  });

  test("native input undo is not canceled by editor history shortcuts", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      const seen: boolean[] = [];
      document.addEventListener("keydown", (event) => {
        if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "z") {
          window.setTimeout(() => seen.push(event.defaultPrevented), 0);
        }
      });
      (window as unknown as { __seenUndoPrevented: boolean[] }).__seenUndoPrevented = seen;
    });

    await page.getByLabel("Title A").click();
    await page.keyboard.type(" typed");
    await expect(page.getByLabel("Title A")).toHaveValue("Alpha typed");
    await page.keyboard.press("Control+Z");

    await expect(page.getByLabel("Title A")).not.toHaveValue("Alpha typed");
    await expect.poll(() => page.evaluate(() =>
      (window as unknown as { __seenUndoPrevented: boolean[] }).__seenUndoPrevented.at(-1))).toBe(false);
  });

  test("editor history undo and redo stay scoped to the active form", async ({ page }) => {
    await mount(page);

    await page.getByLabel("Title A").fill("Alpha revised");
    await page.waitForTimeout(450);
    await page.locator("#focus-a").focus();
    await page.keyboard.press("Control+Z");
    await expect(page.getByLabel("Title A")).toHaveValue("Alpha");
    await page.keyboard.press("Control+Y");
    await expect(page.getByLabel("Title A")).toHaveValue("Alpha revised");

    await page.getByLabel("Title B").fill("Beta revised");
    await page.waitForTimeout(450);
    await page.locator("#focus-b").focus();
    await page.keyboard.press("Control+Z");
    await expect(page.getByLabel("Title B")).toHaveValue("Beta");
    await expect(page.getByLabel("Title A")).toHaveValue("Alpha revised");
  });

  test("two forms do not both consume shortcuts, and outside focus does not save", async ({ page }) => {
    const hits: string[] = [];
    await page.route("**/save-a", async (route) => {
      hits.push("save-a");
      await route.fulfill({ contentType: "application/json", body: JSON.stringify({ ok: true, message: "Saved A." }) });
    });
    await page.route("**/save-b", async (route) => {
      hits.push("save-b");
      await route.fulfill({ contentType: "application/json", body: JSON.stringify({ ok: true, message: "Saved B." }) });
    });
    await page.route("**/delete-*", async (route) => {
      hits.push("delete");
      await route.fulfill({ status: 500, body: "delete should not be reached" });
    });
    await mount(page);

    const resultPromise = nextActionResult(page);
    await page.getByLabel("Title B").focus();
    await page.keyboard.press("Control+S");
    const result = await resultPromise;

    expect(result.ok).toBe(true);
    expect(result.result).toMatchObject({ ok: true, message: "Saved B." });
    expect(hits).toEqual(["save-b"]);
    await expect(page.locator("#editor-b [data-gosx-studio-save-state='true']")).toHaveText("Saved");

    await page.locator("#outside").focus();
    await page.keyboard.press("Control+S");
    await page.waitForTimeout(150);
    expect(hits).toEqual(["save-b"]);
  });

  test("client actions require structured GoSX JSON success, including HTTP 303 redirects", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      (window as unknown as { __seenFetchHeaders: Record<string, string>[] }).__seenFetchHeaders = [];
      window.fetch = async (_url: RequestInfo | URL, init?: RequestInit) => {
        const headers = init?.headers as Record<string, string>;
        (window as unknown as { __seenFetchHeaders: Record<string, string>[] }).__seenFetchHeaders.push(headers);
        return {
          ok: false,
          status: 303,
          redirected: false,
          url: "http://127.0.0.1:4173/save-b",
          headers: { get: () => "application/json" },
          json: async () => ({ ok: true, redirect: "/editor?saved=1" }),
        } as unknown as Response;
      };
    });

    const resultPromise = nextActionResult(page);
    await page.getByLabel("Title B").focus();
    await page.keyboard.press("Control+S");
    const result = await resultPromise;

    expect(result.ok).toBe(true);
    expect(result.status).toBe(303);
    expect(result.redirected).toBe(false);
    expect(result.result).toMatchObject({ ok: true, redirect: "/editor?saved=1" });
    const seenHeaders = await page.evaluate(() =>
      (window as unknown as { __seenFetchHeaders: Record<string, string>[] }).__seenFetchHeaders);
    expect(seenHeaders[0]).toMatchObject({
      Accept: "application/json",
      "X-Requested-With": "XMLHttpRequest",
    });
    await expect(page.locator("#editor-b")).toHaveAttribute("data-gosx-studio-save-state", "saved");
  });

  test("2xx HTML without a structured action result does not claim saved", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      window.fetch = async () => new Response("<html>Saved maybe</html>", {
        status: 200,
        headers: { "Content-Type": "text/html" },
      });
    });
    await page.getByLabel("Title B").fill("Beta revised");
    const resultPromise = nextActionResult(page);
    await page.locator("#editor-b [data-gosx-studio-save-button='true']").click();
    const result = await resultPromise;

    expect(result.ok).toBe(false);
    expect(String(result.error)).toContain("structured success response");
    await expect(page.locator("#editor-b")).toHaveAttribute("data-gosx-studio-save-state", "error");
    await expect(page.locator("#editor-b")).toHaveAttribute("data-studio-dirty-state", "dirty");
    await expect(page.getByLabel("Title B")).toHaveValue("Beta revised");
  });

  test("structured auth redirects do not claim a successful save", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      window.fetch = async () => new Response(JSON.stringify({ ok: true, redirect: "/login" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    await page.getByLabel("Title B").fill("Auth redirect draft");
    const resultPromise = nextActionResult(page);
    await page.locator("#editor-b [data-gosx-studio-save-button='true']").click();
    const result = await resultPromise;
    expect(result.ok).toBe(false);
    expect(String(result.error)).toContain("sign-in");
    await expect(page.locator("#editor-b")).toHaveAttribute("data-gosx-studio-save-state", "error");
    await expect(page.locator("#editor-b")).toHaveAttribute("data-studio-dirty-state", "dirty");
  });

  test("autosave also requires structured JSON success", async ({ page }) => {
    await mount(page);
    await page.locator("#editor-a").evaluate((form) => {
      form.setAttribute("data-gosx-studio-autosave", "true");
      form.setAttribute("data-gosx-studio-autosave-delay", "250");
    });
    await page.evaluate(() => {
      window.fetch = async () => new Response("<html>Saved maybe</html>", {
        status: 200,
        headers: { "Content-Type": "text/html" },
      });
    });
    await page.getByLabel("Title A").fill("Alpha autosave");
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "error", { timeout: 2000 });
    await expect(page.getByLabel("Title A")).toHaveValue("Alpha autosave");
  });

  test("gosxstudio command events still drive form history", async ({ page }) => {
    await mount(page);
    await page.getByLabel("Title A").fill("Command edit");
    await page.waitForTimeout(450);

    const prevented = await page.locator("#editor-a").evaluate((form) => {
      const event = new CustomEvent("gosxstudio:command", {
        bubbles: true,
        cancelable: true,
        detail: { kind: "history", target: "undo" },
      });
      form.dispatchEvent(event);
      return event.defaultPrevented;
    });

    expect(prevented).toBe(true);
    await expect(page.getByLabel("Title A")).toHaveValue("Alpha");
  });

  test("client save failures, redirects, retry, and edits during save keep dirty state truthful", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      const win = window as unknown as {
        __saveBodies: string[];
        __releaseSlowSave?: () => void;
      };
      let attempt = 0;
      win.__saveBodies = [];
      function response(init: { ok?: boolean; status?: number; redirected?: boolean; url?: string; type?: string; body?: unknown }): Response {
        return {
          ok: init.ok ?? true,
          status: init.status ?? 200,
          redirected: init.redirected ?? false,
          url: init.url ?? "http://127.0.0.1:4173/save-a",
          headers: { get: () => init.type ?? "application/json" },
          json: async () => init.body,
        } as unknown as Response;
      }
      window.fetch = async (_url: RequestInfo | URL, init?: RequestInit) => {
        attempt += 1;
        const body = init?.body as FormData | undefined;
        win.__saveBodies.push(body ? Array.from(body.entries()).map(([key, value]) => `${key}=${value}`).join("&") : "");
        if (attempt === 1) {
          return response({ body: { ok: false, message: "Validation failed.", data: { fieldErrors: { title: "Required" } } } });
        }
        if (attempt === 2) {
          return response({ redirected: true, url: "http://127.0.0.1:4173/login", type: "text/html", body: null });
        }
        if (attempt === 3) {
          return new Promise<Response>((resolve) => {
            win.__releaseSlowSave = () => resolve(response({ body: { ok: true, message: "Saved." } }));
          });
        }
        return response({ body: { ok: true, message: "Saved." } });
      };
    });

    await page.getByLabel("Title A").fill("Draft one");
    let resultPromise = nextActionResult(page);
    await page.keyboard.press("Control+S");
    let result = await resultPromise;
    expect(result.ok).toBe(false);
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "error");
    await expect(page.getByLabel("Title A")).toHaveValue("Draft one");

    resultPromise = nextActionResult(page);
    await page.keyboard.press("Control+S");
    result = await resultPromise;
    expect(result.ok).toBe(false);
    expect(String(result.error)).toContain("sign-in");
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "error");
    await expect(page.getByLabel("Title A")).toHaveValue("Draft one");

    resultPromise = nextActionResult(page);
    await page.keyboard.press("Control+S");
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "saving");
    await page.getByLabel("Title A").fill("Draft two while saving");
    await page.evaluate(() => {
      (window as unknown as { __releaseSlowSave?: () => void }).__releaseSlowSave?.();
    });
    result = await resultPromise;
    expect(result.ok).toBe(true);
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "dirty");
    await expect(page.getByLabel("Title A")).toHaveValue("Draft two while saving");

    resultPromise = nextActionResult(page);
    await page.keyboard.press("Control+S");
    result = await resultPromise;
    expect(result.ok).toBe(true);
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "saved");
    const bodies = await page.evaluate(() =>
      (window as unknown as { __saveBodies: string[] }).__saveBodies);
    expect(bodies.at(-1)).toContain("Draft two while saving");
  });

  test("client actions and autosave send the owning form CSRF token", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      const win = window as unknown as {
        __csrfRequests: Array<{ url: string; header: string | null; body: string | null }>;
      };
      win.__csrfRequests = [];
      window.fetch = async (url: RequestInfo | URL, init?: RequestInit) => {
        const headers = new Headers(init?.headers);
        const body = init?.body as FormData | undefined;
        win.__csrfRequests.push({
          url: String(url),
          header: headers.get("X-CSRF-Token"),
          body: body && typeof body.get === "function" ? String(body.get("csrf_token") || "") : null,
        });
        return new Response(JSON.stringify({ ok: true, message: "Saved." }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      };
    });

    let resultPromise = nextActionResult(page);
    await page.getByLabel("Title A").fill("Alpha saved");
    await page.locator("#editor-a [data-gosx-studio-save-button='true']").click();
    await expect(resultPromise).resolves.toMatchObject({ ok: true });

    await page.locator("#editor-b").evaluate((form) => {
      form.setAttribute("data-gosx-studio-autosave", "true");
      form.setAttribute("data-gosx-studio-autosave-delay", "250");
    });
    await page.getByLabel("Title B").fill("Beta autosaved");
    await expect.poll(() => page.evaluate(() =>
      (window as unknown as { __csrfRequests: unknown[] }).__csrfRequests.length)).toBe(2);
    await expect(page.locator("#editor-b")).toHaveAttribute("data-gosx-studio-save-state", "saved", { timeout: 2000 });

    const requests = await page.evaluate(() =>
      (window as unknown as { __csrfRequests: Array<{ url: string; header: string | null; body: string | null }> }).__csrfRequests);
    expect(requests).toEqual([
      { url: "http://127.0.0.1:4173/save-a", header: "state-csrf-a", body: "state-csrf-a" },
      { url: "http://127.0.0.1:4173/save-b", header: "state-csrf-b", body: "state-csrf-b" },
    ]);
  });

  test("missing or cross-origin CSRF-protected client targets fail visibly without a request", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      const form = document.querySelector("#editor-a") as HTMLFormElement;
      form.querySelector("input[name='csrf_token']")?.remove();
      const win = window as unknown as { __fetchCount: number };
      win.__fetchCount = 0;
      window.fetch = async () => {
        win.__fetchCount += 1;
        return new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      };
    });

    await page.getByLabel("Title A").fill("Missing token draft");
    let resultPromise = nextActionResult(page);
    await page.locator("#editor-a [data-gosx-studio-save-button='true']").click();
    let result = await resultPromise;
    expect(result.ok).toBe(false);
    expect(String(result.error)).toContain("CSRF token");
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "error");
    await expect(page.locator("#editor-a [data-gosx-studio-save-detail='true']")).toContainText("CSRF token");
    await expect(page.locator("#editor-a")).toHaveAttribute("data-studio-dirty-state", "dirty");

    await page.locator("#editor-a").evaluate((form) => {
      form.insertAdjacentHTML("afterbegin", '<input type="hidden" name="csrf_token" value="current-token">');
      form.setAttribute("action", "http://127.0.0.1:4174/external-save");
    });
    resultPromise = nextActionResult(page);
    await page.locator("#editor-a [data-gosx-studio-save-button='true']").click();
    result = await resultPromise;
    expect(result.ok).toBe(false);
    expect(String(result.error)).toContain("this site");
    await expect(page.locator("#editor-a [data-gosx-studio-save-detail='true']")).toContainText("this site");
    expect(await page.evaluate(() => (window as unknown as { __fetchCount: number }).__fetchCount)).toBe(0);
    await expect(page.getByLabel("Title A")).toHaveValue("Missing token draft");
  });

  test("pending enhanced saves guard newer edits from links and unload, then clean saves navigate", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      const win = window as unknown as {
        __releasePendingSave?: () => void;
        __confirmCalls: number;
      };
      win.__confirmCalls = 0;
      window.confirm = () => {
        win.__confirmCalls += 1;
        return false;
      };
      const leave = document.createElement("a");
      leave.id = "leave-editor";
      leave.href = "/next-page";
      leave.textContent = "Leave editor";
      document.body.appendChild(leave);
      window.fetch = async () => new Promise<Response>((resolve) => {
        win.__releasePendingSave = () => resolve(new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }));
      });
    });

    await page.getByLabel("Title A").fill("Draft sent");
    const pendingResult = nextActionResult(page);
    await page.locator("#editor-a [data-gosx-studio-save-button='true']").click();
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "saving");
    await page.getByLabel("Title A").fill("Draft newer while saving");

    const guarded = await page.evaluate(() => {
      const link = document.getElementById("leave-editor") as HTMLAnchorElement;
      const click = new MouseEvent("click", { bubbles: true, cancelable: true });
      link.dispatchEvent(click);
      const unload = new Event("beforeunload", { cancelable: true });
      window.dispatchEvent(unload);
      return { clickPrevented: click.defaultPrevented, unloadPrevented: unload.defaultPrevented };
    });
    expect(guarded).toEqual({ clickPrevented: true, unloadPrevented: true });
    expect(await page.evaluate(() => (window as unknown as { __confirmCalls: number }).__confirmCalls)).toBe(1);

    await page.evaluate(() => {
      (window as unknown as { __releasePendingSave?: () => void }).__releasePendingSave?.();
    });
    await expect(pendingResult).resolves.toMatchObject({ ok: true });
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "dirty");
    await expect(page.getByLabel("Title A")).toHaveValue("Draft newer while saving");

    await page.evaluate(() => {
      window.fetch = async () => new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    const cleanResult = nextActionResult(page);
    await page.locator("#editor-a [data-gosx-studio-save-button='true']").click();
    await expect(cleanResult).resolves.toMatchObject({ ok: true });
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "saved");
    const cleanNavigation = await page.evaluate(() => {
      const link = document.getElementById("leave-editor") as HTMLAnchorElement;
      const click = new MouseEvent("click", { bubbles: true, cancelable: true });
      link.dispatchEvent(click);
      const unload = new Event("beforeunload", { cancelable: true });
      window.dispatchEvent(unload);
      return { clickPrevented: click.defaultPrevented, unloadPrevented: unload.defaultPrevented };
    });
    expect(cleanNavigation).toEqual({ clickPrevented: false, unloadPrevented: false });
  });

  test("native form submits bypass enhanced pending-save guards without a spurious unload prompt", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      const form = document.querySelector("#editor-b") as HTMLFormElement;
      form.removeAttribute("data-gosx-studio-client");
      const leave = document.createElement("a");
      leave.id = "native-leave";
      leave.href = "/native-next";
      leave.textContent = "Native next";
      document.body.appendChild(leave);
      leave.addEventListener("click", (event) => event.preventDefault());
      const win = window as unknown as { __confirmCalls: number };
      win.__confirmCalls = 0;
      window.confirm = () => {
        win.__confirmCalls += 1;
        return false;
      };
    });
    await page.getByLabel("Title B").fill("Native draft");
    const submitResult = await page.evaluate(() => {
      const form = document.querySelector("#editor-b") as HTMLFormElement;
      const submitter = form.querySelector("[data-gosx-studio-save-button='true']") as HTMLButtonElement;
      const event = new SubmitEvent("submit", { bubbles: true, cancelable: true, submitter });
      const dispatched = form.dispatchEvent(event);
      const link = document.getElementById("native-leave") as HTMLAnchorElement;
      let stateGuardPrevented = false;
      document.addEventListener("click", (clickEvent) => {
        if (clickEvent.target === link) stateGuardPrevented = clickEvent.defaultPrevented;
      }, true);
      const click = new MouseEvent("click", { bubbles: true, cancelable: true });
      link.dispatchEvent(click);
      const unload = new Event("beforeunload", { cancelable: true });
      window.dispatchEvent(unload);
      return { dispatched, submitPrevented: event.defaultPrevented, clickPrevented: stateGuardPrevented, unloadPrevented: unload.defaultPrevented };
    });
    expect(submitResult).toEqual({ dispatched: true, submitPrevented: false, clickPrevented: false, unloadPrevented: false });
    expect(await page.evaluate(() => (window as unknown as { __confirmCalls: number }).__confirmCalls)).toBe(0);
  });

  test("capture popstate guard cancels dirty history back before GoSX bubble listeners", async ({ page }) => {
    await mount(page);
    await page.getByLabel("Title A").fill("History draft");
    const originalURL = await page.evaluate(() => {
      const win = window as unknown as { __popstateBubbleHits: number; __confirmCalls: number };
      win.__popstateBubbleHits = 0;
      win.__confirmCalls = 0;
      window.confirm = () => {
        win.__confirmCalls += 1;
        return false;
      };
      window.addEventListener("popstate", () => {
        win.__popstateBubbleHits += 1;
      });
      const original = window.location.href;
      window.history.pushState({ page: "one" }, "", "/editor?history=one");
      window.history.pushState({ page: "two" }, "", "/editor?history=two");
      window.history.back();
      return original;
    });
    await expect.poll(() => page.evaluate(() => window.location.href)).toBe(originalURL);
    await expect.poll(() => page.evaluate(() => (window as unknown as { __confirmCalls: number }).__confirmCalls)).toBe(1);
    expect(await page.evaluate(() => (window as unknown as { __popstateBubbleHits: number }).__popstateBubbleHits)).toBe(0);
  });

  test("late client responses cannot update a detached form after a soft-root replacement", async ({ page }) => {
    await mount(page);
    await page.evaluate(() => {
      const win = window as unknown as {
        __releaseDetachedSave?: () => void;
        __lateActionResults: number;
      };
      win.__lateActionResults = 0;
      document.addEventListener("gosxstudio:action-result", () => {
        win.__lateActionResults += 1;
      });
      window.fetch = async () => new Promise<Response>((resolve) => {
        win.__releaseDetachedSave = () => resolve(new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }));
      });
    });

    await page.getByLabel("Title A").fill("Detached draft");
    const resultPromise = nextActionResult(page);
    await page.locator("#editor-a [data-gosx-studio-save-button='true']").click();
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "saving");
    await page.evaluate(() => {
      const oldForm = document.getElementById("editor-a");
      oldForm?.remove();
      const replacement = document.createElement("form");
      replacement.id = "editor-a";
      replacement.action = "/save-a";
      replacement.method = "post";
      replacement.setAttribute("data-gosx-studio-state", "true");
      replacement.innerHTML = "<output data-gosx-studio-save-state=\"true\">Saved</output><span data-gosx-studio-save-detail=\"true\">Ready</span><input type=\"hidden\" name=\"csrf_token\" value=\"replacement-token\"><label>Title A <input name=\"title\" value=\"Replacement\"></label>";
      document.body.appendChild(replacement);
      document.dispatchEvent(new Event("gosx:render"));
      (window as unknown as { __releaseDetachedSave?: () => void }).__releaseDetachedSave?.();
    });
    await page.waitForTimeout(50);
    await expect(page.locator("#editor-a")).toHaveAttribute("data-gosx-studio-save-state", "saved");
    await expect(page.getByLabel("Title A")).toHaveValue("Replacement");
    expect(await page.evaluate(() => (window as unknown as { __lateActionResults: number }).__lateActionResults)).toBe(0);
    expect(await Promise.race([
      resultPromise.then(() => "unexpected"),
      new Promise<string>((resolve) => setTimeout(() => resolve("no-result"), 50)),
    ])).toBe("no-result");
  });
});
