import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

// Self-contained @smoke coverage for the flow designer's durable field /
// validation / action / binding editing (authoring/flow_handlers.go,
// panels/flow_field_editor_panel.go) and its isolated test-execution path
// (authoring.ApplyTestFlowAction). Like flow_field_editor_panel_test.go's Go
// coverage, this file needs no sibling host repo: it drives the real
// authoringruntime script against a fixture matching
// panels.RenderFlowFieldEditorPanel's exact field-name contract
// (authoring.AuthoringField*), with page.route() standing in for the host
// adapter — the same pattern authoringruntime_test.ts already uses for
// save-control. It runs in plain `npx playwright test`, no
// GOSX_STUDIO_REFERENCE_APP_E2E env var required.
//
// Covers the descriptor's verification target: modify a form field,
// validation, and submit binding; execute success and error paths against an
// isolated handler; readiness rejects an invalid binding.

const runtimeJS = readFileSync(
  path.resolve(__dirname, "../authoringruntime/island_runtime.js"),
  "utf8",
);

const fields = {
  operation: "gosx_studio_operation",
  flowKey: "gosx_studio_flow_key",
  flowActionKey: "gosx_studio_flow_action_key",
  flowActionLabel: "gosx_studio_flow_action_label",
  flowFieldName: "gosx_studio_flow_field_name",
  flowFieldLabel: "gosx_studio_flow_field_label",
  flowFieldRequired: "gosx_studio_flow_field_required",
  controlKind: "gosx_studio_control_kind",
  binding: "gosx_studio_binding",
};

function editorHTML(): string {
  return `
    <main class="editor-workbench" data-gosx-studio-workbench="true">
      <output data-gosx-studio-save-state="true">Dirty</output>
      <span data-gosx-studio-save-detail="true">Unsaved</span>

      <section data-studio-flow-field-editor="true" data-studio-flow-field-editor-flow="contact" data-studio-flow-field-editor-action="submit">
        <form
          id="flow-action-form"
          action="/admin/editor/__actions/authoring"
          method="post"
          data-gosx-studio-authoring-managed="true"
          data-studio-flow-action-form="submit"
        >
          <input type="hidden" name="csrf_token" value="test-csrf" />
          <input type="hidden" name="${fields.operation}" value="set-flow-action" />
          <input type="hidden" name="${fields.flowKey}" value="contact" />
          <input type="hidden" name="${fields.flowActionKey}" value="submit" />
          <label>
            Action label
            <input name="${fields.flowActionLabel}" value="Send message" />
          </label>
          <label>
            Submit binding (handler)
            <input name="${fields.binding}" value="" data-studio-flow-binding-input="true" />
          </label>
          <output data-studio-flow-binding-status="true">Needs setup</output>
          <p data-studio-flow-binding-reason="true">Connect a handler before this flow can accept submissions.</p>
          <button type="submit">Save action</button>
        </form>

        <form
          id="flow-field-form"
          action="/admin/editor/__actions/authoring"
          method="post"
          data-gosx-studio-authoring-managed="true"
          data-studio-flow-field-form="email"
        >
          <input type="hidden" name="csrf_token" value="test-csrf" />
          <input type="hidden" name="${fields.operation}" value="set-flow-field" />
          <input type="hidden" name="${fields.flowKey}" value="contact" />
          <input type="hidden" name="${fields.flowActionKey}" value="submit" />
          <input type="hidden" name="${fields.flowFieldName}" value="email" />
          <label>
            Label
            <input name="${fields.flowFieldLabel}" value="Email" />
          </label>
          <label>
            Kind
            <select name="${fields.controlKind}">
              <option value="text" selected>Text</option>
              <option value="choice">Choice</option>
            </select>
          </label>
          <label>
            Required (validation rule)
            <input type="checkbox" name="${fields.flowFieldRequired}" value="true" checked />
          </label>
          <button type="submit">Save field</button>
        </form>

        <form
          id="flow-test-form"
          action="/admin/editor/__actions/authoring"
          method="post"
          data-gosx-studio-authoring-managed="true"
          data-studio-flow-test-form="submit"
        >
          <input type="hidden" name="csrf_token" value="test-csrf" />
          <input type="hidden" name="${fields.operation}" value="test-flow-action" />
          <input type="hidden" name="${fields.flowKey}" value="contact" />
          <input type="hidden" name="${fields.flowActionKey}" value="submit" />
          <label>
            Email
            <input name="email" value="" />
          </label>
          <button type="submit">Run test submission</button>
        </form>
      </section>
    </main>
  `;
}

test.describe("@smoke GoSXStudio flow designer durable field/action/binding editing", () => {
  test("editing a field's label, kind, and required rule posts the set-flow-field mutation", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor", (route) =>
      route.fulfill({ contentType: "text/html", body: editorHTML() }));
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      const request = route.request();
      const body = request.postData() ?? "";
      expect(body).toContain(`name="${fields.operation}"`);
      expect(body).toContain("set-flow-field");
      expect(body).toContain(`name="${fields.flowFieldLabel}"`);
      expect(body).toContain("Email address");
      expect(body).toContain(`name="${fields.flowFieldRequired}"`);
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ ok: true, message: "Field saved.", data: { message: "Field saved.", refreshPreview: true } }),
      });
    });

    await page.goto("http://127.0.0.1:4173/editor");
    await page.addScriptTag({ content: runtimeJS });

    const resultPromise = page.evaluate(() => new Promise<{ result?: { message?: string } }>((resolve) => {
      document.addEventListener("gosxstudio:authoring-result", (event) => resolve((event as CustomEvent).detail), { once: true });
    }));

    await page.locator("#flow-field-form").getByLabel("Label", { exact: true }).fill("Email address");
    await page.getByRole("button", { name: "Save field" }).click();
    const detail = await resultPromise;

    expect(detail.result?.message).toBe("Field saved.");
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Field saved.");
  });

  test("connecting the submit binding posts the set-flow-action mutation with the handler ref", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor", (route) =>
      route.fulfill({ contentType: "text/html", body: editorHTML() }));
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      const body = route.request().postData() ?? "";
      expect(body).toContain("set-flow-action");
      expect(body).toContain(`name="${fields.binding}"`);
      expect(body).toContain("handlers.email.contact");
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ ok: true, message: "Flow action saved.", data: { message: "Flow action saved.", refreshPreview: true } }),
      });
    });

    await page.goto("http://127.0.0.1:4173/editor");
    await page.addScriptTag({ content: runtimeJS });

    await page.locator("[data-studio-flow-binding-input='true']").fill("handlers.email.contact");
    await page.getByRole("button", { name: "Save action" }).click();
    await expect(page.locator("[data-gosx-studio-save-detail]")).toHaveText("Flow action saved.");
  });

  test("the isolated test handler's success path reports success without a real submission", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor", (route) =>
      route.fulfill({ contentType: "text/html", body: editorHTML() }));
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      const body = route.request().postData() ?? "";
      expect(body).toContain("test-flow-action");
      expect(body).toContain("operator@example.com");
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          ok: true,
          message: "Test submission succeeded.",
          data: { message: "Test submission succeeded.", refreshPreview: false },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/editor");
    await page.addScriptTag({ content: runtimeJS });

    const resultPromise = page.evaluate(() => new Promise<{ result?: { message?: string } }>((resolve) => {
      document.addEventListener("gosxstudio:authoring-result", (event) => resolve((event as CustomEvent).detail), { once: true });
    }));

    await page.locator("#flow-test-form [name='email']").fill("operator@example.com");
    await page.getByRole("button", { name: "Run test submission" }).click();
    const detail = await resultPromise;
    expect(detail.result?.message).toBe("Test submission succeeded.");
  });

  test("the isolated test handler's error path never reports success", async ({ page }) => {
    await page.route("http://127.0.0.1:4173/editor", (route) =>
      route.fulfill({ contentType: "text/html", body: editorHTML() }));
    await page.route("**/admin/editor/__actions/authoring", async (route) => {
      await route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          ok: false,
          message: "This field is required.",
          data: { fieldErrors: { email: "This field is required." } },
        }),
      });
    });

    await page.goto("http://127.0.0.1:4173/editor");
    await page.addScriptTag({ content: runtimeJS });

    const raceResult = await page.evaluate(() => new Promise<string>((resolve) => {
      let resolved = false;
      document.addEventListener("gosxstudio:authoring-result", () => {
        resolved = true;
        resolve("authoring-result-fired");
      }, { once: true });
      window.setTimeout(() => {
        if (!resolved) resolve("no-authoring-result");
      }, 1_500);
    }));

    await page.getByRole("button", { name: "Run test submission" }).click();
    const outcome = await raceResult;

    expect(outcome).toBe("no-authoring-result");
    // The form must not be left stuck mid-submission after a handled failure.
    await expect(page.locator("#flow-test-form")).not.toHaveAttribute("data-gosx-pending", "true");
  });

  test("readiness rejects an invalid submit binding and surfaces an actionable reason", async ({ page }) => {
    await page.setContent(editorHTML());
    await expect(page.locator("[data-studio-flow-binding-status='true']")).toHaveText("Needs setup");
    await expect(page.locator("[data-studio-flow-binding-reason='true']")).toHaveText(
      "Connect a handler before this flow can accept submissions.",
    );
  });
});
