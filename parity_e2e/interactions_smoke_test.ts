import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import path from "node:path";

// Self-contained @smoke coverage for the no-code interaction primitives
// (core/interactions.go, panels/interactions_panel.go,
// interactionruntime/island_runtime.js, hostruntime/assets/studio.css's
// [data-gosx-interaction=...] rules). Unlike the @reference-apps suites this
// file needs no sibling host repo: it drives the real runtime script and
// real stylesheet against a hand-authored fixture that matches
// core.Interaction.RenderAttrs()'s attribute contract exactly, so it runs in
// plain `npx playwright test` with no GOSX_STUDIO_REFERENCE_APP_E2E env var.
//
// Covers the descriptor's verification target: one reveal-on-scroll and one
// hover-focus-state interaction; reduced motion; keyboard equivalence to
// hover. Publish/readiness coverage for interactions lives in the Go suite
// (core/interactions_test.go, cms/studio/interaction_readiness_test.go) since
// that is pure durable-operation/readiness logic with no browser surface.

const interactionRuntimeJS = readFileSync(
  path.resolve(__dirname, "../interactionruntime/island_runtime.js"),
  "utf8",
);
const studioCSS = readFileSync(
  path.resolve(__dirname, "../hostruntime/assets/studio.css"),
  "utf8",
);

// The stylesheet is embedded inline in the SAME markup passed to
// page.setContent (rather than injected afterward via addStyleTag) so the
// element's *first* style resolution already has opacity:0 — CSS
// Transitions never run on an element's first style calculation, only on a
// later change from a real "before-change style". Adding the stylesheet
// after the element already exists (with the UA-default opacity:1) would
// itself count as a style change and spuriously animate opacity 1 -> 0,
// which is a test-harness artifact, not real page-load behavior.
function interactionsFixtureHTML(): string {
  return `
    <style>${studioCSS}</style>
    <main>
      <button
        id="hover-target"
        data-gosx-interaction="hover-focus-state"
        data-gosx-interaction-effect="lift"
        data-gosx-interaction-duration-ms="150"
        data-gosx-interaction-delay-ms="0"
        data-gosx-interaction-once="false"
      >Hover or focus me</button>
      <p
        id="reveal-target"
        data-gosx-interaction="reveal-on-scroll"
        data-gosx-interaction-effect="fade-up"
        data-gosx-interaction-duration-ms="250"
        data-gosx-interaction-delay-ms="0"
        data-gosx-interaction-once="true"
        style="margin-top: 1600px;"
      >Reveal me on scroll</p>
      <div style="height: 400px;"></div>
    </main>
  `;
}

test.describe("@smoke GoSXStudio no-code interactions", () => {
  test("reveal-on-scroll stays keyboard/AT-reachable before reveal and reveals once scrolled into view", async ({ page }) => {
    await page.setContent(interactionsFixtureHTML());
    await page.addScriptTag({ content: interactionRuntimeJS });

    const reveal = page.locator("#reveal-target");
    // Before it intersects, the element must still be present, not
    // display:none/visibility:hidden — an interaction state may dim it
    // (opacity) but must never remove it from the accessibility tree.
    await expect(reveal).not.toHaveAttribute("data-gosx-interaction-revealed", "true");
    const beforeReveal = await reveal.evaluate((el) => {
      const style = getComputedStyle(el);
      return { display: style.display, visibility: style.visibility, opacity: style.opacity };
    });
    expect(beforeReveal.display).not.toBe("none");
    expect(beforeReveal.visibility).not.toBe("hidden");
    expect(Number(beforeReveal.opacity)).toBeLessThan(1);

    await reveal.scrollIntoViewIfNeeded();
    // The runtime's rootMargin ("0px 0px -10% 0px") intentionally requires
    // clearing a bit past the raw viewport edge before it counts as
    // intersecting — scrollIntoViewIfNeeded stops at the nearest edge, so
    // nudge a little further, the way a real visitor's scroll would.
    await page.mouse.wheel(0, 300);
    await expect(reveal).toHaveAttribute("data-gosx-interaction-revealed", "true", { timeout: 5_000 });
    await expect
      .poll(async () => Number(await reveal.evaluate((el) => getComputedStyle(el).opacity)))
      .toBe(1);
  });

  test("reduced motion reveals content immediately with no animation, regardless of scroll", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.setContent(interactionsFixtureHTML());
    await page.addScriptTag({ content: interactionRuntimeJS });

    const reveal = page.locator("#reveal-target");
    // No scrollIntoViewIfNeeded() call here: reduced motion must reveal
    // immediately, without ever needing the visitor to scroll.
    await expect(reveal).toHaveAttribute("data-gosx-interaction-revealed", "true");
    const opacity = await reveal.evaluate((el) => getComputedStyle(el).opacity);
    expect(Number(opacity)).toBe(1);
    const transition = await reveal.evaluate((el) => getComputedStyle(el).transitionDuration);
    expect(transition.split(",").every((value) => value.trim() === "0s")).toBe(true);
  });

  test("hover-focus-state gives keyboard focus the same visual treatment as mouse hover", async ({ page }) => {
    await page.setContent(interactionsFixtureHTML());
    await page.addScriptTag({ content: interactionRuntimeJS });

    const hoverTarget = page.locator("#hover-target");
    const baseline = await hoverTarget.evaluate((el) => getComputedStyle(el).transform);

    await hoverTarget.hover();
    const hoveredTransform = await hoverTarget.evaluate((el) => getComputedStyle(el).transform);
    expect(hoveredTransform).not.toBe(baseline);
    await page.mouse.move(0, 0);

    // Reach the same element by keyboard only. Click the (non-focusable)
    // reveal paragraph first to blur without landing focus back on the
    // button itself, then Tab forward — the button is the sole focusable
    // element in this fixture, so one Tab press reaches it — and confirm
    // focus alone reproduces the identical hover effect.
    await page.locator("#reveal-target").click();
    await page.keyboard.press("Tab");
    const focusedTag = await page.evaluate(() => document.activeElement?.id ?? "");
    expect(focusedTag).toBe("hover-target");
    const focusedTransform = await hoverTarget.evaluate((el) => getComputedStyle(el).transform);
    expect(focusedTransform).toBe(hoveredTransform);
  });

  test("reduced motion disables the hover-focus-state transform effect too", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.setContent(interactionsFixtureHTML());
    await page.addScriptTag({ content: interactionRuntimeJS });

    const hoverTarget = page.locator("#hover-target");
    const baseline = await hoverTarget.evaluate((el) => getComputedStyle(el).transform);
    await hoverTarget.hover();
    const hoveredTransform = await hoverTarget.evaluate((el) => getComputedStyle(el).transform);
    expect(hoveredTransform).toBe(baseline);
  });
});
