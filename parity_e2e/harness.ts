// Shared parity-test harness for the gosx-studio runtime burn-down.
//
// Used by every Phase 3 slice (1 of 7 = FieldRuntime). The slice-2..7 authors
// reuse the same `bootBaseline` / `bootCandidate` / `assertEditorRoute` API.
// Keep this file small and intention-revealing — if it sprawls, the next
// slice author won't reuse the pattern and the matrix loses cohesion.
//
// ## Architecture
//
// Phase 3 ships each runtime contract additively: the legacy JS implementation
// in `assets/studio-engines.js` keeps living, and the new `.gsx`-authored
// island ships beside it. A bridge shim in studio-engines.js decides which
// path runs based on a feature flag (`field-runtime-islands`,
// `selection-runtime-islands`, etc.) declared in
// `studio.ShellConfig.FeatureFlags`.
//
// The original Phase 3 plan called for `bootBaseline()` / `bootCandidate()`
// to spawn two separate consumer processes pinned to different gosx-studio
// versions. We collapse that to a single consumer process and switch modes
// via a query parameter (`?gosx-studio-parity=baseline|candidate`) that the
// editor route maps onto the feature flag. Rationale: both implementations
// already coexist additively (that is the whole point of Section C of every
// slice plan). The flag is the slice-shipping mechanism. The harness should
// drive that same mechanism rather than maintain a parallel binary version-
// pinning machinery. See lessons/phase-3-slice-1-fieldruntime-harness-shape.md
// for the trade-off discussion.
//
// ## Consumer contract
//
// The consumer (muddy-noni-commerce editor route) must:
//
//   1. Honor `?gosx-studio-parity=candidate` by enabling the relevant
//      runtime-islands feature flag for that request.
//   2. Honor `?gosx-studio-parity=baseline` by force-disabling the flag.
//   3. Default (no param) to the production-default value of the flag.
//
// Without this contract, both modes return the same DOM and the parity
// assertion is meaningless. Each slice's editor patch must add the
// query-param handling for its contract's flag.
//
// ## Per-test usage
//
// ```ts
// import { bootBaseline, bootCandidate, disposeBoot } from "./harness";
//
// test("fieldruntime.bind parity", async ({ browser }) => {
//   const baseline = await bootBaseline(browser);
//   const candidate = await bootCandidate(browser);
//   try {
//     await runScenario(baseline.page);
//     await runScenario(candidate.page);
//     expect(await snapshot(candidate.page)).toEqual(await snapshot(baseline.page));
//   } finally {
//     await disposeBoot(baseline);
//     await disposeBoot(candidate);
//   }
// });
// ```

import type { Browser, BrowserContext, Page } from "@playwright/test";

/**
 * One side of a parity comparison: a browser context + page, isolated from
 * the other side's cookies / storage / network state.
 */
export interface BootHandle {
  readonly mode: ParityMode;
  readonly context: BrowserContext;
  readonly page: Page;
}

export type ParityMode = "baseline" | "candidate";

/** Path the editor route is mounted at in muddy-noni-commerce. */
export const EDITOR_ROUTE = "/admin/editor";

/** Query parameter the editor route reads to flip the parity flag. */
export const PARITY_QUERY_PARAM = "gosx-studio-parity";

/**
 * Resolve the parity URL for a given mode. Slice tests use this when they
 * need to navigate to a non-default route while staying in the harness mode.
 */
export function parityURL(mode: ParityMode, route: string = EDITOR_ROUTE): string {
  const separator = route.includes("?") ? "&" : "?";
  return `${route}${separator}${PARITY_QUERY_PARAM}=${mode}`;
}

/** Boot the editor route with the legacy JS runtime active (feature flag off). */
export async function bootBaseline(browser: Browser): Promise<BootHandle> {
  return bootMode(browser, "baseline");
}

/** Boot the editor route with the island runtime active (feature flag on). */
export async function bootCandidate(browser: Browser): Promise<BootHandle> {
  return bootMode(browser, "candidate");
}

/** Tear down a boot handle and close its underlying browser context. */
export async function disposeBoot(handle: BootHandle | null | undefined): Promise<void> {
  if (!handle) return;
  await handle.context.close().catch(() => {
    // Best-effort cleanup. Test teardown errors must not mask scenario
    // failures.
  });
}

/**
 * Navigate `page` to the editor route in the given mode and wait for the
 * canvas + inspector mounts to be present. Throws if mounts never appear.
 *
 * Reused inside `bootMode` and exported so individual scenarios can re-assert
 * after a navigation.
 */
export async function assertEditorRoute(page: Page, mode: ParityMode): Promise<void> {
  await page.goto(parityURL(mode), { waitUntil: "domcontentloaded" });

  // Canvas + inspector are present in both modes; their selectors come from
  // the editor route's stable structure (see
  // muddy-noni-commerce/app/admin/editor/page.gsx).
  await page.locator("[data-studio-canvas]").first().waitFor({ state: "attached" });
  await page.locator("[data-studio-inspector], [data-studio-mode-panel]")
    .first()
    .waitFor({ state: "attached" });

  // The studio-engines bundle exposes its runtime globals on `window`; if
  // any of them are missing the bundle didn't execute and the parity test
  // would silently pass against a broken bundle. Fail loudly here instead.
  await page.waitForFunction(() => {
    const w = window as unknown as Record<string, unknown>;
    return Boolean(w.GoSXStudioEngineRuntime && w.GoSXStudioFieldRuntime);
  }, undefined, { timeout: 10_000 });
}

async function bootMode(browser: Browser, mode: ParityMode): Promise<BootHandle> {
  const context = await browser.newContext();
  const page = await context.newPage();
  await assertEditorRoute(page, mode);
  return { mode, context, page };
}
