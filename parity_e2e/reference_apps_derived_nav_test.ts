import { expect, test, type Page, type APIRequestContext } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Live-proof e2e for the M5 AUTO-DERIVED NAVIGATION loop on the muddy website
// editor — proven end-to-end in a real headless Chromium against ONE running
// muddy server (so the editor DRAFT and the public storefront LIVE state share
// one in-process cms.Store across the editor/storefront split).
//
// M5 derives a content-aware site nav (cms.Store.EffectiveNavigation):
//   • Home (/) always;
//   • Shop (/shop) iff a published, non-archived Product exists;
//   • Gallery (/gallery) iff a published GalleryWork exists;
//   • Blog (/blog) iff a published BlogPost exists;
//   • About (/about); Contact (/contact);
//   • then each published custom Page as /pages/{slug}.
// A stored Settings.Navigation override (enabled, ordered) replaces derivation.
// The PUBLIC storefront (app/layout.gsx) renders this nav under
//   <nav class="nav" aria-label="Main"> … <a href> … </nav>
// (non-admin branch), escaped. The editor's "Navigation" panel
// (data-studio-navigation-panel) posts label / "Show in navigation" / position
// rows to the `navigation` editor action → store.SaveDraftSettings (a DRAFT).
// The existing M3 **Publish** button promotes draft→live.
//
// This file asserts, in order:
//   (1) DERIVED-NAV RENDER — a fresh GET of the public storefront "/" renders a
//       data-driven derived nav inside <nav … aria-label="Main">: the
//       always-present Home (/), About (/about), Contact (/contact), and —
//       because the seed publishes products / a gallery / a blog — Shop (/shop),
//       Gallery (/gallery), Blog (/blog).
//   (2) EDIT → PUBLISH → PUBLIC UPDATE — the editor Navigation panel relabels one
//       item ("About") to a UNIQUE per-run marker, the "Save navigation" submit is
//       triggered (real UI button click) writing the marker to the DRAFT, then the
//       REAL Publish button promotes draft→live (post-CSRF-fix). A fresh GET of "/"
//       then shows the UNIQUE marker label on the /about link — proving the edited
//       nav reached the live storefront end-to-end.
//
// Honesty discipline (matching the sibling canvas tests): the marker is freshly
// generated per run (timestamp + random nonce, HTML-safe chars only so it survives
// the server's html escaping byte-for-byte) so the post-publish storefront
// assertion cannot false-pass on stale/seed content; we also assert the marker is
// absent from the storefront BEFORE publish.
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness refreshes
// Muddy's ignored dist assets and boots the server against the gosx/gosx-studio
// under test.

const NAV_PANEL = "[data-studio-navigation-panel='true']";
const NAV_ROW = `${NAV_PANEL} [data-studio-navigation-row]`;
const NAVIGATION_ROUTE = "/admin/editor/__actions/navigation";
const PUBLISH_ROUTE = "/admin/editor/__actions/publish";
const PUBLISH_PANEL = "[data-studio-publish-panel='true']";
const PUBLISH_BUTTON = "[data-studio-submit-action='publish']";
const NAV_SAVE_BUTTON = `${NAV_PANEL} [data-studio-submit-action='navigation']`;

// Items the derived nav always carries (seed-independent).
const ALWAYS_NAV = [
  { label: "Home", href: "/" },
  { label: "About", href: "/about" },
  { label: "Contact", href: "/contact" },
];
// Items the SEED publishes content for (published products / gallery / blog).
const SEED_CONTENT_NAV = [
  { label: "Shop", href: "/shop" },
  { label: "Gallery", href: "/gallery" },
  { label: "Blog", href: "/blog" },
];

test.describe("@reference-apps muddy M5 auto-derived navigation: derived render + edit→publish→public update", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni: the public storefront renders the content-derived nav, and a Navigation-panel relabel + Publish promotes the new label to the live nav", async ({ page, request }) => {
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
    const marker = `Nav ${Date.now()} ${nonce}`;

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      // ── (1) DERIVED-NAV RENDER — storefront "/" shows the content-derived nav ──
      const storefrontSeed = await fetchStorefront(request, server.baseURL);
      const seedNav = extractMainNav(storefrontSeed);
      expect(
        seedNav,
        `the public storefront must render the main nav <nav … aria-label="Main">; html-head=${storefrontSeed.slice(0, 240)}`,
      ).not.toBeNull();
      const seedNavHTML = seedNav as string;

      // Always-present derived items.
      for (const { label, href } of ALWAYS_NAV) {
        expect(
          navHasLink(seedNavHTML, href, label),
          `derived nav must always contain ${label} (href ${href}); nav=${seedNavHTML}`,
        ).toBe(true);
      }
      // Seed publishes products / gallery / blog → content-aware items present.
      for (const { label, href } of SEED_CONTENT_NAV) {
        expect(
          navHasLink(seedNavHTML, href, label),
          `the seed publishes content, so the derived nav must contain ${label} (href ${href}); nav=${seedNavHTML}`,
        ).toBe(true);
      }
      // The marker must not yet exist anywhere on the storefront (pre-edit baseline).
      expect(
        storefrontSeed.includes(marker),
        "the per-run nav marker must NOT exist on the storefront before any edit",
      ).toBe(false);

      // ── (2) EDIT → PUBLISH → PUBLIC UPDATE ────────────────────────────────────
      await openEditorNavigationPanel(page, server.baseURL);

      // The Navigation panel renders its editable rows (label inputs are present +
      // editable): assert the real UI is there and that the /about row is editable.
      const aboutLabelInput = await navRowLabelInputForHref(page, "/about");
      expect(aboutLabelInput, "the Navigation panel must render an editable row for the /about nav item").not.toBeNull();
      const labelLocator = page.locator(`#${aboutLabelInput}`);
      await labelLocator.click();
      await page.keyboard.press("ControlOrMeta+A");
      await page.keyboard.type(marker, { delay: 8 });
      expect(await labelLocator.inputValue(), "the marker must land in the About label input before save").toBe(marker);

      // Drive the nav edit through the REAL UI: the "Save navigation" button is a
      // <button type=submit form=websiteEditorForm formaction=navigation> (mirroring
      // the Publish button) — no nested <form>, so the studio STATE runtime re-sends
      // the workbench form (carrying the typed marker in the /about label + the
      // workbench csrf_token) as a multipart fetch to the navigation editor action.
      // Click it and await the navigation action's POST response.
      const navSave = await clickSaveNavigation(page);

      // ── (2a) NAV SAVE landed (real button reached the navigation action) ──────
      // The real "Save navigation" click must hit the navigation editor action with
      // a 2xx/3xx (NOT 403/4xx): proves the formaction button submits the workbench
      // form to the navigation action (not the workbench save action, and not blocked
      // by CSRF).
      expect(
        navSave.status,
        `the real Save-navigation click must NOT 403 (workbench csrf_token carries through the formaction submit); url=${navSave.url} status=${navSave.status} body=${navSave.body.slice(0, 200)}`,
      ).not.toBe(403);
      expect(
        navSave.status >= 200 && navSave.status < 400,
        `the real Save-navigation click must reach the navigation action with 2xx/3xx (not a 4xx/wrong-action); url=${navSave.url} status=${navSave.status} body=${navSave.body.slice(0, 200)}`,
      ).toBe(true);

      // After save the editor draft must carry the new label (draft-aware editor
      // re-render). Reload so the panel re-renders from EffectiveNavigationForEditor.
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      await waitForNavigationPanel(page);
      const draftLabel = await navRowLabelValueForHref(page, "/about");
      expect(
        draftLabel,
        `after the nav save + reload the editor panel must re-serve the DRAFT label for /about; got=${JSON.stringify(draftLabel)}`,
      ).toBe(marker);

      // The DRAFT must NOT yet have reached the public storefront (draft isolation).
      const storefrontDuringDraft = await fetchStorefront(request, server.baseURL);
      expect(
        storefrontDuringDraft.includes(marker),
        "the relabeled nav item is only a DRAFT before publish — it must NOT appear on the live storefront yet",
      ).toBe(false);

      // ── (2b) PUBLISH — the REAL Publish button promotes draft → live ──────────
      const publishStatus = await triggerPublish(page, server.baseURL);
      expect(
        [200, 303].includes(publishStatus),
        `the real Publish button click must be accepted/redirected (post-CSRF-fix); got status ${publishStatus}`,
      ).toBe(true);

      // ── (2c) PUBLIC UPDATE — the storefront nav now carries the marker label ──
      const storefrontAfterPublish = await fetchStorefront(request, server.baseURL);
      const liveNav = extractMainNav(storefrontAfterPublish);
      expect(liveNav, "the storefront must still render the main nav after publish").not.toBeNull();
      const liveNavHTML = liveNav as string;
      expect(
        navHasLink(liveNavHTML, "/about", marker),
        `after Publish the LIVE storefront nav must show the relabeled /about item as the unique marker; nav=${liveNavHTML}`,
      ).toBe(true);
      // The other derived items survive the override (full row set posted).
      expect(navHasLink(liveNavHTML, "/", "Home"), `Home must survive the nav override; nav=${liveNavHTML}`).toBe(true);
      expect(navHasLink(liveNavHTML, "/contact", "Contact"), `Contact must survive the nav override; nav=${liveNavHTML}`).toBe(true);

      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|navigation|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// openEditorNavigationPanel boots the editor route and waits for the Navigation
// panel (data-studio-navigation-panel) + its editable rows to render.
async function openEditorNavigationPanel(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("[data-studio-workbench='true']").first(), "the editor workbench must render").toBeAttached({ timeout: 30_000 });
  await waitForNavigationPanel(page);
}

// waitForNavigationPanel waits for the Navigation panel + at least one editable
// row to be present (the panel only renders rows when nav items exist).
async function waitForNavigationPanel(page: Page) {
  // The Navigation panel is homed to the default "Home" mode (editing the
  // storefront menu is routine owner work, so it must not require the
  // support-only Advanced tab) — reveal Home explicitly in case an earlier
  // step left another mode active.
  await revealModeIfPresent(page, "home");
  await expect(page.locator(NAV_PANEL).first(), "the editor Navigation panel must render").toBeAttached({ timeout: 30_000 });
  await expect(page.locator(NAV_ROW).first(), "the Navigation panel must render at least one editable nav row").toBeAttached({ timeout: 30_000 });
}

// navRowLabelInputForHref finds the label <input id> for the nav row whose hidden
// href input equals the given href, so the test edits the right row by content
// (not by fragile positional index).
async function navRowLabelInputForHref(page: Page, href: string): Promise<string | null> {
  return page.evaluate((wantHref) => {
    const rows = Array.from(document.querySelectorAll("[data-studio-navigation-panel='true'] [data-studio-navigation-row]"));
    for (const row of rows) {
      const hidden = row.querySelector("input[type='hidden'][name^='navigationHref']") as HTMLInputElement | null;
      if (hidden && hidden.value === wantHref) {
        const label = row.querySelector("input[name^='navigationLabel']") as HTMLInputElement | null;
        return label ? label.id : null;
      }
    }
    return null;
  }, href);
}

// navRowLabelValueForHref reads the current label input value for the nav row
// matching the given href (used to confirm the draft re-renders the new label).
async function navRowLabelValueForHref(page: Page, href: string): Promise<string | null> {
  return page.evaluate((wantHref) => {
    const rows = Array.from(document.querySelectorAll("[data-studio-navigation-panel='true'] [data-studio-navigation-row]"));
    for (const row of rows) {
      const hidden = row.querySelector("input[type='hidden'][name^='navigationHref']") as HTMLInputElement | null;
      if (hidden && hidden.value === wantHref) {
        const label = row.querySelector("input[name^='navigationLabel']") as HTMLInputElement | null;
        return label ? label.value : null;
      }
    }
    return null;
  }, href);
}

// clickSaveNavigation clicks the REAL "Save navigation" button — a
// <button type=submit form=websiteEditorForm formaction=navigation> that mirrors
// the Publish button. The studio STATE runtime intercepts the workbench form
// submit and re-sends it as a multipart fetch honoring the submitter's
// formaction (the navigation editor action), carrying the live workbench form
// fields (the typed marker in the /about label + the workbench csrf_token). We
// await that POST response so the assertion proves the click really reached the
// navigation action. Returns the action's url + status + body.
async function clickSaveNavigation(page: Page): Promise<{ url: string; status: number; body: string }> {
  const button = page.locator(NAV_SAVE_BUTTON).first();
  await expect(
    button,
    "the real 'Save navigation' button (form=websiteEditorForm formaction=navigation) must be present + visible",
  ).toBeVisible({ timeout: 30_000 });

  const responsePromise = page.waitForResponse(
    (response) => response.url().includes(NAVIGATION_ROUTE) && response.request().method() === "POST",
    { timeout: 30_000 },
  );
  await button.click();
  const response = await responsePromise;
  const status = response.status();
  // Redirect responses (the navigation action answers 303 → editor) have no
  // readable body; only read the body for non-redirect statuses so a successful
  // redirect doesn't throw.
  const body = status >= 300 && status < 400 ? "" : await response.text();
  return { url: response.url(), status, body };
}

// triggerPublish reveals the publish panel + clicks the REAL Publish button
// (form="websiteEditorForm" + formaction=publish), which the studio STATE runtime
// re-sends as a multipart fetch whose csrf_token the gosx guard now reads. Returns
// the publish action POST status, then reloads the editor.
async function triggerPublish(page: Page, baseURL: string): Promise<number> {
  void baseURL;
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
  expect(
    status,
    `the native Publish click must NOT 403 (gosx now reads the multipart csrf_token); got ${status}`,
  ).not.toBe(403);
  await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
  return status;
}

// fetchStorefront GETs the PUBLIC storefront "/" from the SAME running server via a
// server-side request (no editor DOM involved) and returns its HTML. The
// storefront renders store.EffectiveNavigation() (LIVE), the ground truth for the
// derived nav + draft isolation.
async function fetchStorefront(request: APIRequestContext, baseURL: string): Promise<string> {
  const response = await request.get(`${baseURL}/`, { failOnStatusCode: false, timeout: 60_000 });
  return response.text();
}

// extractMainNav returns the inner HTML of the public storefront's
// <nav … aria-label="Main"> … </nav>, or null if absent. The derived nav links
// live inside this element (layout.gsx non-admin branch), so all link assertions
// are scoped to it — proving the nav is the data-driven main nav, not some other
// markup that happens to contain a matching href.
function extractMainNav(html: string): string | null {
  // Match the <nav …> opening tag that carries aria-label="Main", then capture up
  // to the next </nav>. The storefront main nav has no nested <nav>, so a
  // non-greedy capture to the first closing tag is exact.
  const open = /<nav\b[^>]*aria-label="Main"[^>]*>/i.exec(html);
  if (!open) return null;
  const start = open.index + open[0].length;
  const end = html.indexOf("</nav>", start);
  if (end < 0) return null;
  return html.slice(start, end);
}

// navHasLink reports whether the given nav HTML contains an <a href="…"> whose
// (html-escaped) text equals the expected label — i.e. a real, content-matched
// nav link, not merely a substring coincidence.
function navHasLink(navHTML: string, href: string, label: string): boolean {
  const anchorRe = /<a\b[^>]*href="([^"]*)"[^>]*>([\s\S]*?)<\/a>/gi;
  let match: RegExpExecArray | null;
  while ((match = anchorRe.exec(navHTML)) !== null) {
    const gotHref = decodeEntities(match[1]).trim();
    const gotText = decodeEntities(match[2]).replace(/<[^>]*>/g, "").trim();
    if (gotHref === href && gotText === label) return true;
  }
  return false;
}

// decodeEntities reverses the minimal html escaping the gsx renderer applies to
// attribute values + text (&amp; &lt; &gt; &quot; &#34; &#39;) so a marker with
// safe chars round-trips byte-for-byte when compared.
function decodeEntities(value: string): string {
  return value
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#34;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&apos;/g, "'");
}
