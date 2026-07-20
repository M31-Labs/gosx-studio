import { expect, test, type Page, type APIRequestContext } from "@playwright/test";
import { revealModeIfPresent, startMuddyCanvasHTMLSurface } from "./reference_apps_harness";

// Edit-revert regression e2e (Noni staging incident, handoff-24-edit-revert):
// the owner reported "some edits saved, some were auto-reverted." Root cause
// #1 (app/layout.server.go's layoutNavFromStore): the editor's own preview —
// BOTH the workbench page-canvas iframe's "/" and "Contact" tabs AND the
// public storefront route hit with the editor's own ?gosx-preview=1 query
// parameter — rendered store.EffectiveNavigation() (LIVE) unconditionally,
// never store.EffectiveNavigationForEditor() (draft-aware), even though the
// Navigation panel's own "Save navigation" writes to the DRAFT (the SAME
// draft the editor's Navigation panel re-renders from — see
// reference_apps_derived_nav_test.ts's step (2), which proves the panel
// itself goes draft-aware after save). So an operator who relabeled a nav
// item, saved, then checked the canvas preview (or the public storefront
// while still signed in) saw the OLD label — a save that was written
// correctly but LOOKED reverted, purely from a preview-rendering gap, not
// a data-loss bug (staging's cms.json quarantine store was confirmed EMPTY —
// no operation was ever accepted-then-unprojected).
//
// This file proves the fix at the http-response level (no editor DOM
// involved for the assertions) against the SAME running muddy-noni-commerce
// server the other reference-apps e2e specs use:
//   (1) baseline: neither the live storefront nor the ?gosx-preview=1
//       preview view carries the marker yet.
//   (2) after the real "Save navigation" UI submit (DRAFT only, no Publish):
//       the ?gosx-preview=1 preview view (this is what the editor's own
//       canvas-preview iframe/tab requests) shows the NEW draft label; the
//       real public view (no gosx-preview param) still shows the OLD label
//       — proving the draft genuinely persisted (not a revert) and that the
//       public/live surface honestly stays on the published value until
//       Publish (no premature exposure of an unpublished label either).
//
// Requires GOSX_STUDIO_REFERENCE_APP_E2E=1. The shared harness boots a real
// muddy-noni-commerce dev server with MUDDY_MOCK_AUTH=1 (so a bare
// ?gosx-preview=1 request is treated as the authorized editor's own preview
// — see app/page.server.go's authorizedDraftPreview).

const NAV_PANEL = "[data-studio-navigation-panel='true']";
const NAV_ROW = `${NAV_PANEL} [data-studio-navigation-row]`;
const NAVIGATION_ROUTE = "/admin/editor/__actions/navigation";
const NAV_SAVE_BUTTON = `${NAV_PANEL} [data-studio-submit-action='navigation']`;

test.describe("@reference-apps muddy edit-revert regression: navigation draft must preview before Publish", () => {
  test.describe.configure({ timeout: 300_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("Muddy/Noni: a saved-but-unpublished nav relabel shows the DRAFT on the editor's own preview view, and the OLD label on the real public view", async ({ page, request }) => {
    const consoleErrors: string[] = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => consoleErrors.push(`pageerror: ${error.message}`));
    page.on("dialog", (dialog) => { void dialog.accept(); });

    // Freshly generated per run so the assertions cannot false-pass on stale
    // content. HTML-safe chars only so it survives html escaping byte-for-byte.
    const nonce = Math.random().toString(36).slice(2, 10);
    const marker = `PreviewNav ${Date.now()} ${nonce}`;

    const server = await startMuddyCanvasHTMLSurface(request);
    try {
      // ── (1) BASELINE — marker exists nowhere yet ──────────────────────────
      const liveBefore = await fetchHTML(request, server.baseURL, "/");
      const previewBefore = await fetchHTML(request, server.baseURL, "/?gosx-preview=1");
      expect(liveBefore.includes(marker), "marker must not exist on the live storefront before any edit").toBe(false);
      expect(previewBefore.includes(marker), "marker must not exist on the preview view before any edit").toBe(false);

      // ── (2) EDIT — real UI Navigation panel relabel, DRAFT save only ─────
      await openEditorNavigationPanel(page, server.baseURL);
      const aboutLabelInput = await navRowLabelInputForHref(page, "/about");
      expect(aboutLabelInput, "the Navigation panel must render an editable row for the /about nav item").not.toBeNull();
      const labelLocator = page.locator(`#${aboutLabelInput}`);
      await labelLocator.click();
      await page.keyboard.press("ControlOrMeta+A");
      await page.keyboard.type(marker, { delay: 8 });
      expect(await labelLocator.inputValue(), "the marker must land in the About label input before save").toBe(marker);

      const navSave = await clickSaveNavigation(page);
      expect(
        navSave.status >= 200 && navSave.status < 400,
        `the real Save-navigation click must reach the navigation action with 2xx/3xx; url=${navSave.url} status=${navSave.status} body=${navSave.body.slice(0, 200)}`,
      ).toBe(true);

      // ── (3) DRAFT-PREVIEW PARITY — the fix under test ─────────────────────
      // The editor's OWN preview view (?gosx-preview=1 — what the workbench
      // page-canvas iframe/tabs request; see app/admin/editor/page.server.go's
      // augmentPreviewURLForCrossFrameRelay) must show the just-saved DRAFT
      // label. Before the fix, this read store.EffectiveNavigation() (LIVE)
      // unconditionally and kept showing "About" — the save LOOKED reverted.
      const previewAfterSave = await fetchHTML(request, server.baseURL, "/?gosx-preview=1");
      expect(
        navHasLink(extractMainNav(previewAfterSave) ?? previewAfterSave, "/about", marker),
        `the editor's own preview view must reflect the unpublished draft nav label; preview html (nav slice)=${(extractMainNav(previewAfterSave) ?? previewAfterSave).slice(0, 400)}`,
      ).toBe(true);
      // The Contact tab (one of only three routes the workbench page-canvas
      // preview offers — Home/About/Contact) must ALSO be draft-aware: the nav
      // renders on every page via the shared layout, so any route's preview
      // request must carry the SAME draft-aware nav, not just "/".
      const contactPreviewAfterSave = await fetchHTML(request, server.baseURL, "/contact?gosx-preview=1");
      expect(
        navHasLink(extractMainNav(contactPreviewAfterSave) ?? contactPreviewAfterSave, "/about", marker),
        "the Contact route's preview view must also reflect the unpublished draft nav label",
      ).toBe(true);

      // ── (4) LIVE HONESTY — the real public view stays on the OLD label ───
      // until Publish (this is NOT a revert bug in the other direction: an
      // unpublished draft edit must never leak to a real visitor).
      const liveAfterSave = await fetchHTML(request, server.baseURL, "/");
      expect(
        liveAfterSave.includes(marker),
        "the real public storefront (no gosx-preview param) must NOT show the unpublished draft nav label",
      ).toBe(false);
      expect(
        navHasLink(extractMainNav(liveAfterSave) ?? "", "/about", "About"),
        "the real public storefront must keep showing the live (unedited) About label until Publish",
      ).toBe(true);

      const clientErrors = consoleErrors.filter((line) => /canvas|sitemap|site-map|painter|wasm-free|navigation|GoSXStudioSiteMapRuntime/i.test(line) && !/relay\.js/i.test(line));
      expect(clientErrors, `unexpected WASM-free client console errors: ${JSON.stringify(clientErrors)}`).toEqual([]);
    } finally {
      await server.stop();
    }
  });
});

// openEditorNavigationPanel boots the editor route and waits for the
// Navigation panel (data-studio-navigation-panel) + its editable rows to
// render (mirrors reference_apps_derived_nav_test.ts's helper of the same
// shape).
async function openEditorNavigationPanel(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
  await expect(page.locator("[data-studio-workbench='true']").first(), "the editor workbench must render").toBeAttached({ timeout: 30_000 });
  await revealModeIfPresent(page, "home");
  await expect(page.locator(NAV_PANEL).first(), "the editor Navigation panel must render").toBeAttached({ timeout: 30_000 });
  await expect(page.locator(NAV_ROW).first(), "the Navigation panel must render at least one editable nav row").toBeAttached({ timeout: 30_000 });
}

// navRowLabelInputForHref finds the label <input id> for the nav row whose
// hidden href input equals the given href.
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

// clickSaveNavigation clicks the real "Save navigation" button and awaits the
// navigation editor action's POST response (draft-only — no Publish).
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
  const body = status >= 300 && status < 400 ? "" : await response.text();
  return { url: response.url(), status, body };
}

// fetchHTML GETs the given path from the SAME running server via a
// server-side request (no editor DOM involved) and returns its HTML.
async function fetchHTML(request: APIRequestContext, baseURL: string, path: string): Promise<string> {
  const response = await request.get(`${baseURL}${path}`, { failOnStatusCode: false, timeout: 60_000 });
  return response.text();
}

// extractMainNav returns the inner HTML of the storefront's
// <nav … aria-label="Main"> … </nav>, or null if absent (mirrors
// reference_apps_derived_nav_test.ts's helper of the same name).
function extractMainNav(html: string): string | null {
  const open = /<nav\b[^>]*aria-label="Main"[^>]*>/i.exec(html);
  if (!open) return null;
  const start = open.index + open[0].length;
  const end = html.indexOf("</nav>", start);
  if (end < 0) return null;
  return html.slice(start, end);
}

// navHasLink reports whether the given nav HTML contains an <a href="…">
// whose (html-escaped) text equals the expected label.
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

// decodeEntities reverses the minimal html escaping the gsx renderer applies.
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
