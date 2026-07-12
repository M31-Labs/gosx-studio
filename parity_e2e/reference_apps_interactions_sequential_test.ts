import { expect, test } from "@playwright/test";
import { clickAuthoringPanel, gotoEditor, revealModeIfPresent, startMuddy } from "./reference_apps_harness";

// HANDOFF-21 regression probe: "interaction fails after an edit" (owner
// report on Noni staging). Every existing authoring-managed-form e2e test
// (reference_apps_interactions_test.ts included) drives clicks through
// clickEditorActionButton/clickAuthoringPanel, whose default `reloadAfter`
// behavior forces an explicit page.goto(page.url()) reload after EVERY
// click -- which re-executes every <script> on the page and re-runs every
// runtime's DOMContentLoaded bind() from scratch. That default masks any
// bug where a SECOND authoring-managed save on the SAME page load (no
// reload in between -- the real end-user's browser behavior, since
// authoringruntime/island_runtime.js's document-level submit listener
// intercepts the click via fetch/AJAX, not a real navigation) silently
// stops working. This test forces `reloadAfter: false` for both saves to
// match the real end-user's browser behavior and checks whether the SECOND
// interaction attach (a different interaction kind, same component, no
// reload) actually reaches the server and persists.
const HERO_INTERACTIONS_PANEL = "#studio-interactions-home-hero";
const REVEAL_FORM = `${HERO_INTERACTIONS_PANEL} article[data-studio-interaction-card='reveal-on-scroll'] form[data-studio-interaction-row='reveal-on-scroll']`;
const HOVER_FORM = `${HERO_INTERACTIONS_PANEL} article[data-studio-interaction-card='hover-focus-state'] form[data-studio-interaction-row='hover-focus-state']`;

test.describe("@reference-apps Muddy/Noni interactions: sequential same-page saves", () => {
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("attaching a second interaction on the same page load (no reload) still succeeds", async ({ page, request }) => {
    page.on("dialog", (dialog) => { void dialog.accept(); });
    const server = await startMuddy(request);
    try {
      await gotoEditor(page, server.baseURL);
      await revealModeIfPresent(page, "advanced");

      const revealForm = page.locator(REVEAL_FORM);
      await expect(revealForm, "the hero reveal-on-scroll interaction row must render").toBeAttached({ timeout: 15_000 });
      const firstResponse = await clickAuthoringPanel(page, REVEAL_FORM, { reloadAfter: false });
      expect(firstResponse.status(), "attaching the first interaction must not fail server-side").toBe(200);
      const firstBody = await firstResponse.json();
      expect(firstBody?.data?.message ?? firstBody?.message ?? "", "first save should report success").toContain("saved");

      // NO page reload here -- this is the exact gap: a real user stays on
      // the same page and immediately attaches a second interaction.
      const hoverForm = page.locator(HOVER_FORM);
      await expect(hoverForm, "the hero hover-focus-state interaction row must still be attached").toBeAttached({ timeout: 5_000 });
      const secondResponse = await clickAuthoringPanel(page, HOVER_FORM, { reloadAfter: false, noWaitAfter: false });
      expect(secondResponse.status(), "attaching the SECOND interaction (same page, no reload) must not fail server-side").toBe(200);
      const secondBody = await secondResponse.json();
      expect(secondBody?.data?.message ?? secondBody?.message ?? "", "second save should report success").toContain("saved");

      // Confirm both interactions actually persisted (not just that a
      // request was sent) by reloading and reading the panel's own
      // rendered/attached state back from the server.
      await page.goto(page.url(), { waitUntil: "networkidle" });
      await revealModeIfPresent(page, "advanced");
      await expect(page.locator(REVEAL_FORM), "reveal-on-scroll must be attached after reload").toHaveAttribute("data-studio-interaction-attached", "true");
      await expect(page.locator(HOVER_FORM), "hover-focus-state must be attached after reload").toHaveAttribute("data-studio-interaction-attached", "true");
    } finally {
      await server.stop();
    }
  });
});
