import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../operationruntime/island_runtime.js"),
  "utf8",
);

function multipartField(body: string, name: string): string {
  const escapedName = name.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return new RegExp(`name="${escapedName}"\\r?\\n\\r?\\n([\\s\\S]*?)\\r?\\n--`).exec(body)?.[1] ?? "";
}

test.describe("@smoke GoSXStudioOperationRuntime request ordering", () => {
  test("uses the first committed target head in a queued same-target second POST", async ({ page }) => {
    const posts: { value: string; expectedHead: string; expectedRevision: string }[] = [];
    await page.route("http://127.0.0.1:4173/operation-same-target", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <form
            id="operation-form"
            action="/admin/editor/__actions/operation"
            method="post"
            data-gosx-studio-durable-history="true"
            data-studio-target-route="/"
            data-studio-target-page-id="page:home"
            data-studio-target-field="home.hero.title"
            data-studio-target-component="hero"
            data-studio-target-head=""
            data-studio-document-revision="0"
          >
            <input type="hidden" name="csrf_token" value="csrf-operation" />
            <input type="hidden" name="gosx_studio_expected_revision" value="0" />
            <input type="hidden" name="gosx_studio_expected_target_head" value="" />
          </form>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      const body = route.request().postData() ?? "";
      const index = posts.length;
      posts.push({
        value: multipartField(body, "gosx_studio_value"),
        expectedHead: multipartField(body, "gosx_studio_expected_target_head"),
        expectedRevision: multipartField(body, "gosx_studio_expected_revision"),
      });
      if (index === 0) await new Promise((resolve) => setTimeout(resolve, 100));
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: index + 1,
            targetHead: index === 0 ? "head-one" : "head-two",
            operationID: index === 0 ? "op-one" : "op-two",
            value: multipartField(body, "gosx_studio_value"),
          },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/operation-same-target");
    await page.addScriptTag({ content: runtimeJS });

    const snapshot = await page.evaluate(async () => {
      const form = document.querySelector("#operation-form") as HTMLFormElement;
      const runtime = (window as unknown as {
        GoSXStudioOperationRuntime: {
          create: (form: HTMLFormElement) => {
            commit: (kind: string, value: string, extra?: Record<string, string>) => Promise<Response>;
          };
        };
      }).GoSXStudioOperationRuntime.create(form);
      const commits: string[] = [];
      form.addEventListener("gosxstudio:operation-committed", (event) => {
        const detail = (event as CustomEvent).detail as { body?: { data?: { targetHead?: string } } };
        commits.push(detail.body?.data?.targetHead ?? "");
      });

      const first = runtime.commit("set-field", "First");
      const second = runtime.commit("set-field", "Second");
      await Promise.all([first, second]);

      return {
        revision: form.getAttribute("data-studio-document-revision"),
        targetHead: form.getAttribute("data-studio-target-head"),
        expectedHead: (form.querySelector("[name='gosx_studio_expected_target_head']") as HTMLInputElement).value,
        commits,
      };
    });

    expect(posts).toEqual([
      { value: "First", expectedHead: "", expectedRevision: "0" },
      { value: "Second", expectedHead: "head-one", expectedRevision: "1" },
    ]);
    expect(snapshot.commits).toEqual(["head-one", "head-two"]);
    expect(snapshot.revision).toBe("2");
    expect(snapshot.targetHead).toBe("head-two");
    expect(snapshot.expectedHead).toBe("head-two");
  });

  test("keeps distinct out-of-order targets committed without downgrading document revision", async ({ page }) => {
    const posts: { field: string; value: string; expectedRevision: string }[] = [];
    await page.route("http://127.0.0.1:4173/operation-targets", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <form
            id="operation-form"
            action="/admin/editor/__actions/operation"
            method="post"
            data-gosx-studio-durable-history="true"
            data-studio-target-route="/"
            data-studio-target-page-id="page:home"
            data-studio-target-field="home.hero.title"
            data-studio-target-component="hero"
            data-studio-target-head=""
            data-studio-document-revision="0"
          >
            <input type="hidden" name="csrf_token" value="csrf-operation" />
            <input type="hidden" name="gosx_studio_expected_revision" value="0" />
            <input type="hidden" name="gosx_studio_expected_target_head" value="" />
          </form>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      const body = route.request().postData() ?? "";
      const field = multipartField(body, "gosx_studio_binding");
      const value = multipartField(body, "gosx_studio_value");
      posts.push({
        field,
        value,
        expectedRevision: multipartField(body, "gosx_studio_expected_revision"),
      });
      if (field === "home.hero.title") {
        await new Promise((resolve) => setTimeout(resolve, 150));
      }
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: field === "home.hero.title" ? 1 : 2,
            targetHead: field === "home.hero.title" ? "head-title" : "head-subtitle",
            operationID: field === "home.hero.title" ? "op-title" : "op-subtitle",
            value,
          },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/operation-targets");
    await page.addScriptTag({ content: runtimeJS });

    const snapshot = await page.evaluate(async () => {
      const form = document.querySelector("#operation-form") as HTMLFormElement;
      const runtime = (window as unknown as {
        GoSXStudioOperationRuntime: {
          create: (form: HTMLFormElement) => {
            commit: (kind: string, value: string, extra?: Record<string, string>) => Promise<Response>;
          };
        };
      }).GoSXStudioOperationRuntime.create(form);
      const commits: { kind?: string; revision?: number; head?: string }[] = [];
      form.addEventListener("gosxstudio:operation-committed", (event) => {
        const detail = (event as CustomEvent).detail as {
          kind?: string;
          body?: { data?: { documentRevision?: number; targetHead?: string } };
        };
        commits.push({
          kind: detail.kind,
          revision: detail.body?.data?.documentRevision,
          head: detail.body?.data?.targetHead,
        });
      });

      const title = runtime.commit("set-field", "Title", { field: "home.hero.title" });
      const subtitle = runtime.commit("set-field", "Subtitle", { field: "home.hero.subtitle" });
      await Promise.all([title, subtitle]);

      return {
        revision: form.getAttribute("data-studio-document-revision"),
        expectedRevision: (form.querySelector("[name='gosx_studio_expected_revision']") as HTMLInputElement).value,
        commits,
      };
    });

    expect(posts.map((post) => post.field).sort()).toEqual(["home.hero.subtitle", "home.hero.title"]);
    expect(posts.every((post) => post.expectedRevision === "0")).toBe(true);
    expect(snapshot.commits).toEqual([
      { kind: "set-field", revision: 2, head: "head-subtitle" },
      { kind: "set-field", revision: 1, head: "head-title" },
    ]);
    expect(snapshot.revision).toBe("2");
    expect(snapshot.expectedRevision).toBe("2");
  });

  test("emits structured non-ok operation errors and recovers safe cursors", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/operation-conflict", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <form
            id="operation-form"
            action="/admin/editor/__actions/operation"
            method="post"
            data-gosx-studio-durable-history="true"
            data-studio-target-route="/"
            data-studio-target-page-id="page:home"
            data-studio-target-field="home.hero.title"
            data-studio-target-component="hero"
            data-studio-target-head="old-head"
            data-studio-document-revision="3"
          >
            <input type="hidden" name="csrf_token" value="csrf-operation" />
            <input type="hidden" name="gosx_studio_expected_revision" value="3" />
            <input type="hidden" name="gosx_studio_expected_target_head" value="old-head" />
          </form>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      await route.fulfill({
        status: 409,
        contentType: "application/json",
        body: JSON.stringify({
          ok: false,
          message: "Conflict detected.",
          data: {
            documentRevision: 4,
            targetHead: "server-head",
            operationID: "server-op",
          },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/operation-conflict");
    await page.addScriptTag({ content: runtimeJS });

    const snapshot = await page.evaluate(async () => {
      const form = document.querySelector("#operation-form") as HTMLFormElement;
      const runtime = (window as unknown as {
        GoSXStudioOperationRuntime: {
          create: (form: HTMLFormElement) => {
            commit: (kind: string, value: string, extra?: Record<string, string>) => Promise<Response>;
          };
        };
      }).GoSXStudioOperationRuntime.create(form);
      const error = new Promise<{
        status?: number;
        body?: { message?: string; data?: { targetHead?: string } };
      }>((resolve) => {
        form.addEventListener("gosxstudio:operation-error", (event) => {
          resolve((event as CustomEvent).detail);
        }, { once: true });
      });
      await runtime.commit("set-field", "Conflict").catch(() => undefined);
      const detail = await error;
      return {
        revision: form.getAttribute("data-studio-document-revision"),
        targetHead: form.getAttribute("data-studio-target-head"),
        expectedRevision: (form.querySelector("[name='gosx_studio_expected_revision']") as HTMLInputElement).value,
        expectedHead: (form.querySelector("[name='gosx_studio_expected_target_head']") as HTMLInputElement).value,
        detail,
      };
    });

    expect(snapshot.revision).toBe("4");
    expect(snapshot.targetHead).toBe("server-head");
    expect(snapshot.expectedRevision).toBe("4");
    expect(snapshot.expectedHead).toBe("server-head");
    expect(snapshot.detail.status).toBe(409);
    expect(snapshot.detail.body?.message).toBe("Conflict detected.");
    expect(snapshot.detail.body?.data?.targetHead).toBe("server-head");
  });

  test("rejects queued same-target edits after a conflict until the user retries", async ({ page }) => {
    const posts: string[] = [];
    await page.route("http://127.0.0.1:4173/operation-conflict-queue", async (route) => {
      await route.fulfill({
        contentType: "text/html",
        body: `
          <form
            id="operation-form"
            action="/admin/editor/__actions/operation"
            method="post"
            data-gosx-studio-durable-history="true"
            data-studio-target-route="/"
            data-studio-target-page-id="page:home"
            data-studio-target-field="home.hero.title"
            data-studio-target-component="hero"
            data-studio-target-head="old-head"
            data-studio-document-revision="3"
          >
            <input type="hidden" name="csrf_token" value="csrf-operation" />
            <input type="hidden" name="gosx_studio_expected_revision" value="3" />
            <input type="hidden" name="gosx_studio_expected_target_head" value="old-head" />
          </form>
        `,
      });
    });
    await page.route("**/admin/editor/__actions/operation", async (route) => {
      const body = route.request().postData() ?? "";
      posts.push(multipartField(body, "gosx_studio_value"));
      if (posts.length === 1) {
        await route.fulfill({
          status: 409,
          contentType: "application/json",
          body: JSON.stringify({
            ok: false,
            message: "Conflict detected.",
            data: { documentRevision: 4, targetHead: "server-head" },
          }),
        });
        return;
      }
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documentRevision: 5,
            targetHead: "retry-head",
            operationID: "retry-op",
            value: multipartField(body, "gosx_studio_value"),
          },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/operation-conflict-queue");
    await page.addScriptTag({ content: runtimeJS });

    const snapshot = await page.evaluate(async () => {
      const form = document.querySelector("#operation-form") as HTMLFormElement;
      const runtime = (window as unknown as {
        GoSXStudioOperationRuntime: {
          create: (form: HTMLFormElement) => {
            commit: (kind: string, value: string, extra?: Record<string, string>) => Promise<Response>;
          };
        };
      }).GoSXStudioOperationRuntime.create(form);
      const errors: { status?: number; message?: string }[] = [];
      const commits: string[] = [];
      form.addEventListener("gosxstudio:operation-error", (event) => {
        const detail = (event as CustomEvent).detail as { status?: number; body?: { message?: string } };
        errors.push({ status: detail.status, message: detail.body?.message });
      });
      form.addEventListener("gosxstudio:operation-committed", (event) => {
        const detail = (event as CustomEvent).detail as { body?: { data?: { targetHead?: string } } };
        commits.push(detail.body?.data?.targetHead ?? "");
      });

      const first = runtime.commit("set-field", "First").catch((error) => error);
      const second = runtime.commit("set-field", "Second").catch((error) => error);
      const results = await Promise.all([first, second]);
      const third = await runtime.commit("set-field", "Retry");

      return {
        resultNames: results.map((result) => result instanceof Error ? result.message : "ok"),
        thirdStatus: third.status,
        errors,
        commits,
        revision: form.getAttribute("data-studio-document-revision"),
        targetHead: form.getAttribute("data-studio-target-head"),
      };
    });

    expect(posts).toEqual(["First", "Retry"]);
    expect(snapshot.resultNames).toEqual(["Operation failed (409)", "Operation failed (409)"]);
    expect(snapshot.errors).toEqual([
      { status: 409, message: "Conflict detected." },
      { status: 409, message: "Conflict detected." },
    ]);
    expect(snapshot.thirdStatus).toBe(200);
    expect(snapshot.commits).toEqual(["retry-head"]);
    expect(snapshot.revision).toBe("5");
    expect(snapshot.targetHead).toBe("retry-head");
  });
});
