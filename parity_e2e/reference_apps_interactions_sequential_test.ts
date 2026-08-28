import { expect, test, type Locator, type Page } from "@playwright/test";
import { openInteractionsTargetDisclosure, revealModeIfPresent, startMuddyCollaboration } from "./reference_apps_harness";

// HANDOFF-21 regression probe: "interaction fails after an edit" (owner
// report on Noni staging). This test forces two saves on the SAME page load
// (no reload in between — the real end-user's browser behavior) and checks
// whether the SECOND interaction attach (a different interaction kind, same
// component, no reload) actually reaches the server and persists.
//
// Wave 5 (handoff-31, "one product path"): the interactions panel's
// attach/remove forms now dispatch through the durable operation protocol
// instead of a plain managed-AJAX POST, so this now drives
// startMuddyCollaboration + the MUDDY_COLLAB_TEST_AUTH sign-in route and
// commits through the gosxstudio:operation-committed/-error event contract —
// see reference_apps_interactions_test.ts's header comment for the full
// rationale (collabruntime's permissions() gate disables every
// [data-gosx-studio-operation-kind] button until a real, authenticated
// websocket handshake grants capabilities).
const HERO_INTERACTIONS_PANEL = "#studio-interactions-home-hero";
const REVEAL_FORM = `${HERO_INTERACTIONS_PANEL} article[data-studio-interaction-card='reveal-on-scroll'] form[data-studio-interaction-row='reveal-on-scroll']`;
const HOVER_FORM = `${HERO_INTERACTIONS_PANEL} article[data-studio-interaction-card='hover-focus-state'] form[data-studio-interaction-row='hover-focus-state']`;
const SET_INTERACTION_BUTTON = "[data-gosx-studio-operation-kind='set-interaction']";
const COLLAB_PANEL = "[data-gosx-studio-collaboration='true']";

test.describe("@reference-apps Muddy/Noni interactions: sequential same-page saves", () => {
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("attaching a second interaction on the same page load (no reload) still succeeds", async ({ page, request }) => {
    page.on("dialog", (dialog) => { void dialog.accept(); });
    const server = await startMuddyCollaboration(request);
    try {
      await page.goto(`${server.baseURL}/__test/collaboration/signin/designer`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const collabPanel = page.locator(COLLAB_PANEL).first();
      await expect(
        collabPanel,
        "the collaboration runtime must connect before durable operation buttons enable",
      ).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });

      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(collabPanel).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      // The interactions inspector is the Home-mode contextual inspector
      // (declutter slice: workbenchruntime's Bundle() now actually binds, so
      // "advanced" would hide it) — "home" is also the default, so this is a
      // harmless no-op click, but explicit for clarity/robustness against a
      // future default-mode change.
      await revealModeIfPresent(page, "home");

      const revealForm = page.locator(REVEAL_FORM);
      await expect(revealForm, "the hero reveal-on-scroll interaction row must render").toBeAttached({ timeout: 15_000 });
      // Wave 3A (inspector-presentation): open the shared "+ Add interaction"
      // picker (home:hero starts unattached, so its whole card is nested
      // inside it) and the target's own per-target disclosure before the
      // human step of clicking the row's own submit button.
      await openInteractionsTargetDisclosure(page, HERO_INTERACTIONS_PANEL);
      await commitInteraction(page, revealForm, SET_INTERACTION_BUTTON, "first (reveal-on-scroll) attach");

      // NO page reload here -- this is the exact gap: a real user stays on
      // the same page and immediately attaches a second interaction. The
      // first save's in-place fragment refresh may have re-rendered the
      // target's card (now attached, so it has moved out of the "+ Add
      // interaction" picker and its own disclosure defaults back to closed
      // again) — re-run the same disclosure-opening step against the live
      // DOM before the second click.
      const hoverForm = page.locator(HOVER_FORM);
      await expect(hoverForm, "the hero hover-focus-state interaction row must still be attached").toBeAttached({ timeout: 5_000 });
      await openInteractionsTargetDisclosure(page, HERO_INTERACTIONS_PANEL);
      await commitInteraction(page, hoverForm, SET_INTERACTION_BUTTON, "second (hover-focus-state) attach, same page, no reload");

      // Confirm both interactions actually persisted (not just that a
      // request was sent) by reloading and reading the panel's own
      // rendered/attached state back from the server.
      // domcontentloaded, not networkidle: the interaction-heavy editor page
      // keeps background requests trickling and never reliably reaches idle
      // on slow runners — the attribute assertions below are the real gate.
      await page.goto(page.url(), { waitUntil: "domcontentloaded" });
      await revealModeIfPresent(page, "home");
      await expect(page.locator(REVEAL_FORM), "reveal-on-scroll must be attached after reload").toHaveAttribute("data-studio-interaction-attached", "true");
      await expect(page.locator(HOVER_FORM), "hover-focus-state must be attached after reload").toHaveAttribute("data-studio-interaction-attached", "true");
    } finally {
      await server.stop();
    }
  });
});

// commitInteraction clicks the given operation-kind button inside form and
// waits for the durable gosxstudio:operation-committed/-error event pair,
// failing loudly (with a labeled message) instead of silently timing out.
async function commitInteraction(page: Page, form: Locator, buttonSelector: string, label: string) {
  const outcome = page.evaluate(
    () => new Promise<{ kind: "committed" | "error"; detail: unknown }>((resolve) => {
      document.addEventListener("gosxstudio:operation-committed", (event) => resolve({ kind: "committed", detail: (event as CustomEvent).detail }), { once: true });
      document.addEventListener("gosxstudio:operation-error", (event) => resolve({ kind: "error", detail: (event as CustomEvent).detail }), { once: true });
    }),
  );
  await form.locator(buttonSelector).click();
  const result = await outcome;
  expect(result.kind, `${label} must succeed; got ${JSON.stringify(result)}`).toBe("committed");
}
