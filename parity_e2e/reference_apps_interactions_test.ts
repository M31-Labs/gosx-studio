import { expect, test, type Page } from "@playwright/test";
import { openInteractionsTargetDisclosure, revealModeIfPresent, startMuddyCollaboration } from "./reference_apps_harness";

// Descriptor 05/06 independent review P1 fix + P2b executing proof.
//
// P1 finding: internal/cms/interactions.go kept a single Interactions list;
// app/page.server.go / app/about/page.server.go read
// Store.InteractionRenderAttrs unconditionally on the PUBLIC render, so an
// authored no-code interaction (core/interactions.go, descriptor
// HANDOFF-INTERACTIONS-06) went live the instant an editor saved it — no
// draft/live isolation at all, unlike every other durable content field
// (hero headline, hero image, component styles), which already only ever go
// live through an explicit publish.
//
// The fix split Store.InteractionRenderAttrs into a preview-gated read: the
// DRAFT Interactions set for the editor's own preview render, and a NEW
// published LiveInteractions set for every real public request; publishing
// (PublishSettingsDraftAsActor) now promotes draft -> live in the same
// transaction as the rest of a publish.
//
// This file is the browser-level executing proof for that fix, driving the
// REAL Noni admin editor's interactions inspector panel (panels/
// interactions_panel.go, app/admin/editor/interaction_authoring.server.go)
// end to end against one running server: attach a reveal-on-scroll
// interaction to the home hero section; prove the public "/" render carries
// nothing before publish while the preview render already does; publish
// through the real publish panel; prove the public render now carries the
// core.Interaction.RenderAttrs() data attributes AND ships the
// interactionruntime script; and prove prefers-reduced-motion reveals the
// hero section immediately, with no scroll needed — the exact runtime
// contract interactions_smoke_test.ts already proves against a synthetic
// fixture, now proven end-to-end against the real host.
//
// Wave 5 (handoff-31, "one product path"): the interactions panel's
// attach/remove forms now dispatch through the durable operation protocol
// (operationruntime/island_runtime.js's [data-gosx-studio-durable-history])
// instead of a plain managed-AJAX POST. collabruntime/island_runtime.js's
// global permissions() gate disables every [data-gosx-studio-operation-kind]
// button — including these — until its websocket handshake completes and
// the server grants real capabilities, and that handshake needs a real
// authenticated session (MUDDY_MOCK_AUTH only covers plain HTTP action
// handlers). This file now uses startMuddyCollaboration + the
// MUDDY_COLLAB_TEST_AUTH sign-in route (the same mechanism
// reference_apps_collaboration_direct_edit_form_test.ts and
// reference_apps_publish_changeset_test.ts already prove out) instead of
// the plain startMuddy + clickAuthoringPanel HTTP-response wait, and commits
// through the SAME gosxstudio:operation-committed/-error event contract
// those files use.
const HERO_SELECTOR = "[data-gosx-component='home:hero']";
const HERO_INTERACTIONS_PANEL = "#studio-interactions-home-hero";
const REVEAL_FORM = `${HERO_INTERACTIONS_PANEL} article[data-studio-interaction-card='reveal-on-scroll'] form[data-studio-interaction-row='reveal-on-scroll']`;
const SET_INTERACTION_BUTTON = "[data-gosx-studio-operation-kind='set-interaction']";
const COLLAB_PANEL = "[data-gosx-studio-collaboration='true']";
const PUBLISH_BUTTON = "[data-studio-submit-action='publish']";
const PUBLISH_ROUTE = "/admin/editor/__actions/publish";

test.describe("@reference-apps Muddy/Noni no-code interaction draft/live isolation", () => {
  test.describe.configure({ timeout: 180_000 });
  test.skip(process.env.GOSX_STUDIO_REFERENCE_APP_E2E !== "1", "set GOSX_STUDIO_REFERENCE_APP_E2E=1 to boot sibling reference apps");

  test("an authored hero reveal interaction stays draft-only until publish, then reaches the public page with runtime attrs + script and honors reduced motion", async ({ page, request }) => {
    // The Publish button (and any other admin action button) carries
    // data-admin-confirm; auto-accept so window.confirm never stalls the test.
    page.on("dialog", (dialog) => { void dialog.accept(); });

    const server = await startMuddyCollaboration(request);
    try {
      await page.goto(`${server.baseURL}/__test/collaboration/signin/admin`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const collabPanel = page.locator(COLLAB_PANEL).first();
      await expect(
        collabPanel,
        "the collaboration runtime must connect before durable operation buttons enable",
      ).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });

      // Baseline: nothing attached yet, the public hero section carries no
      // interaction kind.
      await page.goto(`${server.baseURL}/`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const heroBaseline = page.locator(HERO_SELECTOR).first();
      await expect(heroBaseline, "the home hero section must render").toBeAttached();
      expect(await heroBaseline.getAttribute("data-gosx-interaction") || "", "expected no interaction attached at all yet").toBe("");

      // Author the reveal-on-scroll interaction via the REAL editor panel —
      // the same interactions inspector row `interactions_panel_test.go`
      // (gosx-studio) and `interaction_authoring_test.go` (Noni) cover at
      // the Go level; this drives the actual browser durable-operation
      // commit (not a raw runtime.submit() call).
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await expect(collabPanel).toHaveAttribute("data-studio-collab-state", "connected", { timeout: 20_000 });
      // The interactions inspector is the Home-mode contextual inspector
      // (declutter slice: workbenchruntime's Bundle() now actually binds
      // mode tabs, so "advanced" would hide it instead of revealing it).
      await revealModeIfPresent(page, "home");
      const revealForm = page.locator(REVEAL_FORM);
      await expect(revealForm, "the hero reveal-on-scroll interaction row must render").toBeAttached({ timeout: 15_000 });
      // Wave 3A (inspector-presentation): the target's controls now sit
      // behind a closed-by-default disclosure, and with nothing attached yet
      // the target itself is nested inside the shared "+ Add interaction"
      // picker — open both before the human step of clicking the row's own
      // submit button.
      await openInteractionsTargetDisclosure(page, HERO_INTERACTIONS_PANEL);
      await commitInteraction(page, revealForm, SET_INTERACTION_BUTTON);

      // Draft/live isolation — the descriptor 05/06 review P1 fix's core
      // assertion: the public home page STILL carries nothing, while the
      // editor's own preview render (?gosx-preview=1) DOES show it.
      await page.goto(`${server.baseURL}/`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const heroPreDraft = page.locator(HERO_SELECTOR).first();
      expect(
        await heroPreDraft.getAttribute("data-gosx-interaction") || "",
        "the draft interaction must NOT appear on the public page before publish",
      ).toBe("");

      await page.goto(`${server.baseURL}/?gosx-preview=1`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const heroPreview = page.locator(HERO_SELECTOR).first();
      await expect(
        heroPreview,
        "the preview render must show the newly-attached draft interaction",
      ).toHaveAttribute("data-gosx-interaction", "reveal-on-scroll");

      // Publish through the REAL publish panel (the SAME PUBLISH_BUTTON/
      // PUBLISH_ROUTE reference_apps_canvas_draft_publish_test.ts already
      // drives for hero-headline draft/live isolation).
      await page.goto(`${server.baseURL}/admin/editor`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      await revealModeIfPresent(page, "publish");
      const publishButton = page.locator(PUBLISH_BUTTON).first();
      await expect(publishButton, "the Publish button must be present").toBeVisible({ timeout: 15_000 });
      const publishResponsePromise = page.waitForResponse(
        (candidate) => candidate.url().includes(PUBLISH_ROUTE) && candidate.request().method() === "POST",
        { timeout: 30_000 },
      );
      await publishButton.click();
      const publishResponse = await publishResponsePromise;
      expect(publishResponse.status(), "publish must not fail server-side (e.g. CSRF/403)").not.toBe(403);
      expect([200, 303]).toContain(publishResponse.status());

      // The interaction is now live: the public page carries the SAME
      // core.Interaction.RenderAttrs() data attributes the preview render did.
      await page.goto(`${server.baseURL}/`, { waitUntil: "domcontentloaded", timeout: 60_000 });
      const heroLive = page.locator(HERO_SELECTOR).first();
      await expect(
        heroLive,
        "the published interaction must now be live on the public page",
      ).toHaveAttribute("data-gosx-interaction", "reveal-on-scroll");
      expect(await heroLive.getAttribute("data-gosx-interaction-effect"), "the live render must carry a real effect value").toBeTruthy();

      // The published page ships the interactionruntime script
      // (m31labs.dev/gosx-studio/interactionruntime/island_runtime.js,
      // inlined by cmd/muddy-noni/main.go for every publicClientPath
      // request) — the same script interactions_smoke_test.ts drives
      // directly against a synthetic fixture.
      const html = await page.content();
      expect(
        html,
        "the public page must ship the reveal-on-scroll interaction runtime script",
      ).toContain("data-gosx-interaction-revealed");

      // Reduced motion reveals the hero section immediately, with no scroll
      // needed at all (island_runtime.js's bind(): reducedMotion() fails
      // open to fully revealed content, bypassing IntersectionObserver
      // entirely) — the same contract interactions_smoke_test.ts proves in
      // isolation, now proven against the real published page.
      await page.emulateMedia({ reducedMotion: "reduce" });
      await page.reload({ waitUntil: "domcontentloaded", timeout: 60_000 });
      const heroReducedMotion = page.locator(HERO_SELECTOR).first();
      await expect(
        heroReducedMotion,
        "reduced motion must reveal the hero section immediately, with no scroll needed",
      ).toHaveAttribute("data-gosx-interaction-revealed", "true", { timeout: 5_000 });
    } finally {
      await server.stop();
    }
  });
});

// commitInteraction clicks the given operation-kind button inside form and
// waits for the durable gosxstudio:operation-committed/-error event pair,
// failing loudly if the commit was rejected instead of silently timing out.
async function commitInteraction(page: Page, form: import("@playwright/test").Locator, buttonSelector: string) {
  const outcome = page.evaluate(
    () => new Promise<{ kind: "committed" | "error"; detail: unknown }>((resolve) => {
      document.addEventListener("gosxstudio:operation-committed", (event) => resolve({ kind: "committed", detail: (event as CustomEvent).detail }), { once: true });
      document.addEventListener("gosxstudio:operation-error", (event) => resolve({ kind: "error", detail: (event as CustomEvent).detail }), { once: true });
    }),
  );
  await form.locator(buttonSelector).click();
  const result = await outcome;
  expect(result.kind, `the durable interaction commit must succeed; got ${JSON.stringify(result)}`).toBe("committed");
}
