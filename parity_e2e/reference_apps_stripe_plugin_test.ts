import { expect, test, type Page, type APIRequestContext } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M6 STRIPE CHECKOUT PLUGIN loop on the muddy website
// editor — proven end-to-end in a real headless Chromium against ONE running
// muddy server (so the editor DRAFT and the public storefront LIVE state share
// one in-process cms.Store across the editor/storefront split). The muddy server
// is booted with MUDDY_MOCK_CHECKOUT=1 (set by the shared harness) so the
// checkout endpoint returns a MOCK session url with NO real Stripe call.
//
// M6 surfaces a compiled-in Stripe checkout studio.Plugin into the editor block
// palette. The plugin's dev-set checkout WIRING rides as a Locked control that
// gosx-studio filters out of the editor's editable views — so ONLY the
// editor-configurable controls (product + label) are offered. The Part-A
// "Checkout" editor panel (data-studio-checkout-panel) renders exactly those two
// configurable inputs (a product <select> populated from the published products +
// a label text input) and NO dev-wiring/endpoint field. Its "Save checkout"
// button is a <button type=submit form=websiteEditorForm formaction=checkout>
// (mirroring the M5 Navigation panel + the M3 Publish button — no nested <form>,
// which the browser parser drops) that posts the checkout* fields to the
// `checkout` editor action → store.SaveDraftSettings (a DRAFT). The existing M3
// **Publish** button promotes draft→live. The PUBLIC storefront home renders the
// checkout block from the LIVE SiteSettings.Checkout config, wrapping the
// existing POST /api/checkout endpoint.
//
// This file asserts, in order:
//   (1) SURFACING — the editor Checkout panel offers ONLY the configurable
//       controls: a product <select> (options from the published products) + a
//       label text input. There is NO editable action/endpoint/wiring input
//       (the locked dev wiring never reaches the editor).
//   (2) CONFIGURE → PUBLISH — the panel enables the block, picks a seeded
//       published product, and sets a UNIQUE per-run label; "Save checkout"
//       (real UI button click) writes the config to the DRAFT (response NOT 403,
//       2xx/3xx), then the REAL Publish button promotes draft→live (NOT 403).
//   (3) WORKING CHECKOUT — a fresh GET of the PUBLIC home "/" renders the
//       checkout block showing the UNIQUE label; filling the email + clicking the
//       label button initiates checkout — a POST /api/checkout returns 2xx with a
//       (mock) session: result.mock === true and a result.url. The marker label
//       is freshly generated per run so the assertion cannot false-pass on stale
//       seed content; we also assert it is absent from the storefront BEFORE
//       publish.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1 and a Muddy build rebuilt against the
// gosx/gosx-studio under test (`go run ./cmd/muddy-noni` via the shared harness).

const CHECKOUT_PANEL = "[data-studio-checkout-panel='true']";
const CHECKOUT_PRODUCT = `${CHECKOUT_PANEL} select[name='checkoutProductId']`;
const CHECKOUT_LABEL = `${CHECKOUT_PANEL} input[name='checkoutLabel']`;
const CHECKOUT_ENABLE = `${CHECKOUT_PANEL} input[name='checkoutEnabled']`;
const CHECKOUT_SAVE_BUTTON = `${CHECKOUT_PANEL} [data-studio-submit-action='checkout']`;
const CHECKOUT_ROUTE = "/admin/editor/__actions/checkout";
const PUBLISH_ROUTE = "/admin/editor/__actions/publish";
const PUBLISH_PANEL = "[data-studio-publish-panel='true']";
const PUBLISH_BUTTON = "[data-studio-submit-action='publish']";

const PUBLIC_CHECKOUT = "[data-public-checkout='true']";
const PUBLIC_CHECKOUT_FORM = "form[data-public-form='home-checkout']";

test.describe("@reference-apps muddy M6 Stripe checkout plugin: surface → configure → publish → working checkout", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni: the editor surfaces only the configurable checkout controls, configures + publishes them, and the public home then runs a (mock) checkout", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));
    // The publish button carries data-admin-confirm; auto-accept any confirm so it
    // can never stall the test.
    page.on("dialog", (dialog) => { void dialog.accept(); });

    // Freshly generated per run so the post-publish assertion cannot false-pass on
    // stale/seed content. HTML-safe chars only (letters/digits/spaces) so it
    // survives html escaping in the storefront markup byte-for-byte.
    const nonce = Math.random().toString(36).slice(2, 10);
    const marker = `Buy ${Date.now()} ${nonce}`;

    // The harness boots muddy with MUDDY_MOCK_CHECKOUT=1 already set (base env),
    // so /api/checkout returns a mock session with no real Stripe call.
    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      // The marker must not yet exist anywhere on the storefront (pre-edit baseline).
      const storefrontSeed = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontSeed.includes(marker),
        "the per-run checkout marker must NOT exist on the storefront before any edit",
      ).toBe(false);

      await openEditorCheckoutPanel(page, server.baseURL);

      // ── (1) SURFACING — only product + label are offered; no dev wiring ───────
      // The Checkout panel renders the two editor-configurable controls: a product
      // <select> (options from the published products) and a label text input.
      const productSelect = page.locator(CHECKOUT_PRODUCT).first();
      const labelInput = page.locator(CHECKOUT_LABEL).first();
      const enableToggle = page.locator(CHECKOUT_ENABLE).first();
      await expect(productSelect, "the Checkout panel must offer a product <select> control").toBeAttached({ timeout: 30_000 });
      await expect(labelInput, "the Checkout panel must offer a label text control").toBeAttached({ timeout: 30_000 });
      await expect(enableToggle, "the Checkout panel must offer an enable toggle").toBeAttached({ timeout: 30_000 });

      // The product control is populated from the published products (the seed
      // publishes several products): at least one real option with a non-empty id.
      const productOptions = await page.locator(`${CHECKOUT_PRODUCT} option`).evaluateAll((nodes) =>
        nodes.map((node) => ({ value: (node as HTMLOptionElement).value, label: (node as HTMLElement).textContent?.trim() ?? "" })),
      );
      const realOptions = productOptions.filter((opt) => opt.value.length > 0);
      expect(
        realOptions.length,
        `the Checkout product <select> must be populated from the published products; options=${JSON.stringify(productOptions)}`,
      ).toBeGreaterThan(0);

      // There is NO editable action/endpoint/wiring control anywhere in the panel —
      // the locked dev wiring never reaches the editor.
      const wiringFieldCount = await page.locator(
        `${CHECKOUT_PANEL} input[name='action'], ${CHECKOUT_PANEL} input[name='endpoint'], ${CHECKOUT_PANEL} input[name='checkoutAction'], ${CHECKOUT_PANEL} input[name='checkoutEndpoint'], ${CHECKOUT_PANEL} select[name='action'], ${CHECKOUT_PANEL} [data-studio-control-key='action']`,
      ).count();
      expect(
        wiringFieldCount,
        "the editor must NOT surface the locked checkout action/endpoint wiring as an editable field",
      ).toBe(0);

      // ── (2) CONFIGURE — enable + pick a published product + set unique label ──
      const chosen = realOptions[0];
      if (!(await enableToggle.isChecked())) {
        await enableToggle.check();
      }
      await productSelect.selectOption(chosen.value);
      await labelInput.click();
      await page.keyboard.press("ControlOrMeta+A");
      await page.keyboard.type(marker, { delay: 8 });
      expect(await labelInput.inputValue(), "the marker must land in the checkout label input before save").toBe(marker);

      // Drive the save through the REAL UI: the "Save checkout" button is a
      // <button type=submit form=websiteEditorForm formaction=checkout> (mirroring
      // the Navigation Save + Publish buttons) — no nested <form>, so the studio
      // STATE runtime re-sends the workbench form (carrying the chosen product +
      // typed label + the workbench csrf_token) as a multipart fetch to the
      // checkout editor action. Click it and await the checkout action's POST.
      const checkoutSave = await clickSaveCheckout(page);
      expect(
        checkoutSave.status,
        `the real Save-checkout click must NOT 403 (workbench csrf_token carries through the formaction submit); url=${checkoutSave.url} status=${checkoutSave.status} body=${checkoutSave.body.slice(0, 200)}`,
      ).not.toBe(403);
      expect(
        checkoutSave.status >= 200 && checkoutSave.status < 400,
        `the real Save-checkout click must reach the checkout action with 2xx/3xx; url=${checkoutSave.url} status=${checkoutSave.status} body=${checkoutSave.body.slice(0, 200)}`,
      ).toBe(true);

      // The DRAFT must NOT yet have reached the public storefront (draft isolation).
      const storefrontDuringDraft = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontDuringDraft.includes(marker),
        "the configured checkout block is only a DRAFT before publish — its label must NOT appear on the live storefront yet",
      ).toBe(false);

      // ── (2b) PUBLISH — the REAL Publish button promotes draft → live ──────────
      const publishStatus = await triggerPublish(page);
      expect(
        publishStatus,
        `the real Publish button click must NOT 403 (gosx reads the multipart csrf_token); got ${publishStatus}`,
      ).not.toBe(403);
      expect(
        [200, 303].includes(publishStatus),
        `the real Publish button click must be accepted/redirected (post-CSRF-fix); got status ${publishStatus}`,
      ).toBe(true);

      // ── (3) WORKING CHECKOUT — public home renders the block + runs (mock) checkout ──
      const storefrontAfterPublish = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontAfterPublish.includes(marker),
        `after Publish the LIVE storefront home must render the checkout block with the unique label "${marker}"; html=${sliceCheckout(storefrontAfterPublish)}`,
      ).toBe(true);

      // Load the public home in the browser, fill the email, click the label
      // button, and capture the POST /api/checkout response. With
      // MUDDY_MOCK_CHECKOUT=1 the endpoint returns a mock session (mock:true + url)
      // and the public-site.js client navigates to result.url on success.
      const checkout = await runPublicCheckout(page, server.baseURL, marker);
      expect(
        checkout.status >= 200 && checkout.status < 300,
        `the public checkout must POST /api/checkout and get a 2xx (mock) session; status=${checkout.status} body=${JSON.stringify(checkout.body)}`,
      ).toBe(true);
      expect(
        checkout.body && typeof checkout.body.url === "string" && checkout.body.url.length > 0,
        `the (mock) checkout response must carry a session url; body=${JSON.stringify(checkout.body)}`,
      ).toBe(true);
      expect(
        checkout.body?.mock,
        `MUDDY_MOCK_CHECKOUT=1 must produce a mock session (mock:true); body=${JSON.stringify(checkout.body)}`,
      ).toBe(true);

      const clientErrors = consoleErrors.filter(
        (line) => /canvas|sitemap|site-map|painter|wasm-free|checkout|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line) && !/mudrelics/i.test(line),
      );
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// openEditorCheckoutPanel boots the editor route and waits for the Checkout panel
// (data-studio-checkout-panel) to render.
async function openEditorCheckoutPanel(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("[data-studio-workbench='true']").first(), "the editor workbench must render").toBeAttached({ timeout: 30_000 });
  await revealModeIfPresent(page, "checkout");
  await expect(page.locator(CHECKOUT_PANEL).first(), "the editor Checkout panel must render").toBeAttached({ timeout: 30_000 });
}

// clickSaveCheckout clicks the REAL "Save checkout" button — a
// <button type=submit form=websiteEditorForm formaction=checkout> that mirrors
// the Navigation Save + Publish buttons. The studio STATE runtime intercepts the
// workbench form submit and re-sends it as a multipart fetch honoring the
// submitter's formaction (the checkout editor action), carrying the live
// workbench form fields (the chosen product + typed label + the workbench
// csrf_token). We await that POST response so the assertion proves the click
// really reached the checkout action. Returns the action's url + status + body.
async function clickSaveCheckout(page: Page): Promise<{ url: string; status: number; body: string }> {
  const button = page.locator(CHECKOUT_SAVE_BUTTON).first();
  await expect(
    button,
    "the real 'Save checkout' button (form=websiteEditorForm formaction=checkout) must be present + visible",
  ).toBeVisible({ timeout: 30_000 });

  const responsePromise = page.waitForResponse(
    (response) => response.url().includes(CHECKOUT_ROUTE) && response.request().method() === "POST",
    { timeout: 30_000 },
  );
  await button.click();
  const response = await responsePromise;
  const status = response.status();
  const body = status >= 300 && status < 400 ? "" : await response.text();
  return { url: response.url(), status, body };
}

// triggerPublish reveals the publish panel + clicks the REAL Publish button
// (form="websiteEditorForm" + formaction=publish), which the studio STATE runtime
// re-sends as a multipart fetch whose csrf_token the gosx guard now reads. Returns
// the publish action POST status, then reloads the editor.
async function triggerPublish(page: Page): Promise<number> {
  await revealModeIfPresent(page, "publish");
  const button = page.locator(PUBLISH_BUTTON).first();
  await expect(page.locator(PUBLISH_PANEL).first(), "the publish panel must render").toBeAttached({ timeout: 30_000 });
  await expect(button, "the real Publish button must be present + visible to promote the draft").toBeVisible({ timeout: 30_000 });

  const responsePromise = page.waitForResponse(
    (response) => response.url().includes(PUBLISH_ROUTE) && response.request().method() === "POST",
    { timeout: 30_000 },
  );
  await button.click();
  const response = await responsePromise;
  const status = response.status();
  await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
  return status;
}

// runPublicCheckout loads the PUBLIC home in the browser, asserts the checkout
// block + its label button render, fills the email, and clicks the label button.
// The public-site.js client intercepts the home-checkout submit, POSTs JSON to
// /api/checkout, and navigates to result.url on success — we capture that POST
// response (status + parsed JSON body) before the navigation resolves it.
async function runPublicCheckout(
  page: Page,
  baseURL: string,
  label: string,
): Promise<{ status: number; body: { url?: string; mock?: boolean; orderId?: string } | null }> {
  await page.goto(`${baseURL}/`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator(PUBLIC_CHECKOUT).first(), "the public home must render the checkout block").toBeAttached({ timeout: 30_000 });

  const form = page.locator(PUBLIC_CHECKOUT_FORM).first();
  await expect(form, "the public home must render the home-checkout form").toBeAttached({ timeout: 30_000 });

  // The submit button carries the configured unique label — proves the public
  // block is driven by the published config, not stale markup.
  const submit = form.locator("button[type='submit']").first();
  await expect(submit, "the checkout button must render the configured unique label").toHaveText(label, { timeout: 30_000 });

  await form.locator("input[name='email']").fill("buyer@example.com");

  const responsePromise = page.waitForResponse(
    (response) => response.url().includes("/api/checkout") && response.request().method() === "POST",
    { timeout: 30_000 },
  );
  await submit.click();
  const response = await responsePromise;
  const status = response.status();
  const body = (await response.json().catch(() => null)) as { url?: string; mock?: boolean; orderId?: string } | null;
  return { status, body };
}

// fetchStorefront GETs the PUBLIC storefront "/" from the SAME running server via a
// server-side request (no editor DOM involved) and returns its HTML. The
// storefront renders the checkout block from store.Settings().Checkout (LIVE),
// the ground truth for the config + draft isolation.
async function fetchStorefront(request: APIRequestContext, baseURL: string): Promise<string> {
  const response = await request.get(`${baseURL}/`, { failOnStatusCode: false, timeout: 60_000 });
  return response.text();
}

// sliceCheckout returns the checkout-section slice of the storefront html (for
// readable failure messages) or a head slice when the block is absent.
function sliceCheckout(html: string): string {
  const idx = html.indexOf("data-public-checkout");
  if (idx < 0) return html.slice(0, 240);
  return html.slice(idx, idx + 400);
}
