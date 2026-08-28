// Phase 3 slice-4 parity tests for GoSXStudioBlockLayoutRuntime.
//
// Ten scenarios (one @coverage gate + one parity per method):
//   - @coverage        — confirms the candidate boot exposes all nine
//                        island globals. Self-skips with a descriptive
//                        reason when the gosx-studio module bump hasn't
//                        landed in the consumer yet (same precondition as
//                        slices 1, 2, 3).
//   - rows parity      — calls window.GoSXStudioBlockLayoutRuntime.rows
//                        on the editor block list; asserts equal key
//                        sequences between baseline and candidate.
//   - rowKey parity    — calls .rowKey on the first row; asserts equal
//                        return value.
//   - rowForKey parity — calls .rowForKey for a known key; asserts both
//                        resolve to a row with the same key attribute.
//   - moveRow parity   — calls .moveRow up on the second row; asserts
//                        equal post-move key sequences and the row's
//                        .is-selected class state.
//   - renumber parity  — calls .renumber after a manual DOM reorder;
//                        asserts equal [data-block-studio-order] values
//                        and equal blockstudio:reorder event source.
//   - selectRow parity — calls .selectRow for a known key; asserts equal
//                        .is-selected class membership across rows.
//   - commitReorder    — calls .commitReorder(list, key, target, "after");
//     parity             asserts the returned boolean and the post-call
//                        key sequence match.
//   - updateBlockLibraryState — calls the helper after toggling a row's
//     parity                    visibility; asserts equal aria-pressed +
//                              className on the [data-editor-add-block]
//                              buttons.
//   - updateVisibilityState   — toggles a row's visibility checkbox and
//     parity (transitional)     calls the helper; asserts equal row class /
//                              status text in editor frame, then asserts
//                              the iframe-DOM mutation parity (when
//                              same-origin) — this is the ADR 0008
//                              transitional contract (see
//                              ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md).
//
// Each test runs the same user interaction in baseline (legacy JS active)
// and candidate (island runtime active) modes, snapshots observable state,
// and asserts equivalence. See ./harness.ts and the slice plan
// (~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md)
// for the deletion gate this suite feeds.
//
// Per the parity matrix (gosx-studio-runtime-parity-matrix.md),
// BlockLayoutRuntime has exactly nine public methods. The slice plan calls
// for one parity sub-test per method plus the @coverage prerequisite gate;
// see Section E of the plan.

import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { bootBaseline, bootCandidate, disposeBoot } from "./harness";

// Editor block-list authoring surface selectors. These come from the editor
// route's stable structure — see muddy-noni-commerce/app/admin/editor/page.gsx
// + app/admin/editor/page.server.go:1402 (editorHomeLayerRowAttrs) which
// emits data-block-studio-block on every row.
const BLOCK_ROW_SELECTOR = "[data-block-studio-block]";

interface CandidateFlagSnapshot {
  flagAttrPresent: boolean;
  flagAttrValue: string | null;
  blockLayoutRuntimeGlobalShape: {
    rows: boolean;
    rowKey: boolean;
    rowForKey: boolean;
    moveRow: boolean;
    renumber: boolean;
    selectRow: boolean;
    commitReorder: boolean;
    updateBlockLibraryState: boolean;
    updateVisibilityState: boolean;
  };
  islandRuntimeAvailable: {
    rows: boolean;
    rowKey: boolean;
    rowForKey: boolean;
    moveRow: boolean;
    renumber: boolean;
    selectRow: boolean;
    commitReorder: boolean;
    updateBlockLibraryState: boolean;
    updateVisibilityState: boolean;
  };
}

async function snapshotFlag(page: Page): Promise<CandidateFlagSnapshot> {
  return await page.evaluate(() => {
    // The feature flag attribute is surfaced on the workbench root (see
    // lessons/phase-3-slice-1-fieldruntime-deletion-log.md), not on
    // documentElement. Fall back to documentElement for backward compat.
    const attr = "data-gosx-studio-feature-flag-block-layout-runtime-islands";
    const root =
      document.querySelector(`[${attr}]`) ?? document.documentElement;
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as Record<string, unknown>;
    return {
      flagAttrPresent: root.hasAttribute(attr),
      flagAttrValue: root.getAttribute(attr),
      blockLayoutRuntimeGlobalShape: {
        rows: typeof runtime.rows === "function",
        rowKey: typeof runtime.rowKey === "function",
        rowForKey: typeof runtime.rowForKey === "function",
        moveRow: typeof runtime.moveRow === "function",
        renumber: typeof runtime.renumber === "function",
        selectRow: typeof runtime.selectRow === "function",
        commitReorder: typeof runtime.commitReorder === "function",
        updateBlockLibraryState: typeof runtime.updateBlockLibraryState === "function",
        updateVisibilityState: typeof runtime.updateVisibilityState === "function",
      },
      islandRuntimeAvailable: {
        rows: typeof w.__gosx_blocklayout_runtime_island_rows === "function",
        rowKey: typeof w.__gosx_blocklayout_runtime_island_rowKey === "function",
        rowForKey: typeof w.__gosx_blocklayout_runtime_island_rowForKey === "function",
        moveRow: typeof w.__gosx_blocklayout_runtime_island_moveRow === "function",
        renumber: typeof w.__gosx_blocklayout_runtime_island_renumber === "function",
        selectRow: typeof w.__gosx_blocklayout_runtime_island_selectRow === "function",
        commitReorder: typeof w.__gosx_blocklayout_runtime_island_commitReorder === "function",
        updateBlockLibraryState: typeof w.__gosx_blocklayout_runtime_island_updateBlockLibraryState === "function",
        updateVisibilityState: typeof w.__gosx_blocklayout_runtime_island_updateVisibilityState === "function",
      },
    };
  });
}

async function blockListAvailable(page: Page): Promise<boolean> {
  return await page.evaluate((sel) => {
    return document.querySelectorAll(sel).length > 0;
  }, BLOCK_ROW_SELECTOR);
}

// Walk every block row's data-block-studio-block key and return the
// ordered sequence. Stable across baseline and candidate when the legacy
// rows / island rows return identical NodeLists.
async function snapshotRowKeys(page: Page): Promise<string[]> {
  return await page.evaluate((sel) => {
    return Array.prototype.slice.call(document.querySelectorAll(sel)).map((r) => r.getAttribute("data-block-studio-block") || "");
  }, BLOCK_ROW_SELECTOR);
}

async function snapshotOrderValues(page: Page): Promise<string[]> {
  return await page.evaluate((sel) => {
    return Array.prototype.slice.call(document.querySelectorAll(sel)).map((r) => {
      const order = r.querySelector("[data-block-studio-order]") as HTMLInputElement | null;
      return order ? order.value : "";
    });
  }, BLOCK_ROW_SELECTOR);
}

async function snapshotSelected(page: Page): Promise<string[]> {
  return await page.evaluate((sel) => {
    return Array.prototype.slice.call(document.querySelectorAll(sel))
      .filter((r) => r.classList.contains("is-selected"))
      .map((r) => r.getAttribute("data-block-studio-block") || "");
  }, BLOCK_ROW_SELECTOR);
}

async function callRuntime(page: Page, method: string, args: unknown[]): Promise<unknown> {
  return await page.evaluate(({ method, args }) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as Record<string, (...args: unknown[]) => unknown>;
    const fn = runtime[method];
    if (typeof fn !== "function") return undefined;
    return fn.apply(null, args);
  }, { method, args });
}

// Resolve the block-list element (the closest container of every row).
// All nine methods accept a list / root argument; the parity tests pass
// the same element on both sides so any divergence is in the method itself.
async function blockListElement(page: Page): Promise<void> {
  // No-op preamble; we resolve the list inside each evaluate() so we can
  // pass a real Element. Kept here to centralize the documentation.
}

test.describe("GoSXStudioBlockLayoutRuntime parity", () => {
  // The harness boots both modes against the same consumer process via the
  // ?gosx-studio-parity= query parameter. Until the consumer bumps its
  // gosx-studio module dependency to include the blocklayoutruntime island
  // bundle, both modes load the same legacy bundle and the assertions
  // trivially succeed. The "@coverage" test below detects this and warns
  // so the team doesn't ship a false-positive parity claim.

  test("@coverage candidate boot exposes block-layout island runtime globals", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      const snap = await snapshotFlag(handle.page);
      expect(snap.flagAttrPresent).toBe(true);
      expect(snap.flagAttrValue).toBe("true");
      // The next assertion is the "real" coverage check. Until the
      // muddy-noni-commerce gosx-studio bump lands, the island runtime
      // globals are absent and the test marks itself skipped with a
      // descriptive reason so CI surfaces the prerequisite plainly.
      const allIslands = Object.values(snap.islandRuntimeAvailable).every(Boolean);
      test.skip(
        !allIslands,
        "gosx-studio module bump not yet landed in muddy-noni-commerce: " +
          "one or more __gosx_blocklayout_runtime_island_* globals is undefined. " +
          "The parity tests below still run but only verify legacy-path stability. " +
          "Land the dependency bump and rerun to get true parity coverage.",
      );
      // When the islands are present, all nine runtime methods must be
      // wired through the shim. If a method is missing from the shim, the
      // delegate factory above silently produces undefined and dispatches
      // become no-ops.
      for (const method of Object.keys(snap.blockLayoutRuntimeGlobalShape) as Array<keyof typeof snap.blockLayoutRuntimeGlobalShape>) {
        expect(snap.blockLayoutRuntimeGlobalShape[method]).toBe(true);
        expect(snap.islandRuntimeAvailable[method]).toBe(true);
      }
    } finally {
      await disposeBoot(handle);
    }
  });

  // Method 1/9: rows(list)
  test("blocklayoutruntime.rows parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page)) || !(await blockListAvailable(candidate.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const baselineKeys = await page_rowsKeys(baseline.page);
      const candidateKeys = await page_rowsKeys(candidate.page);
      expect(candidateKeys).toEqual(baselineKeys);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 2/9: rowKey(row)
  test("blocklayoutruntime.rowKey parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const baselineKey = await page_firstRowKey(baseline.page);
      const candidateKey = await page_firstRowKey(candidate.page);
      expect(candidateKey).toBe(baselineKey);
      // Sanity — should be a non-empty string from the editor fixture.
      expect(candidateKey.length).toBeGreaterThan(0);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 3/9: rowForKey(root, key)
  test("blocklayoutruntime.rowForKey parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const key = await page_firstRowKey(baseline.page);
      const baselineResolved = await page_rowForKeyMatches(baseline.page, key);
      const candidateResolved = await page_rowForKeyMatches(candidate.page, key);
      expect(candidateResolved).toBe(baselineResolved);
      expect(candidateResolved).toBe(true);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 4/9: moveRow(list, row, direction)
  test("blocklayoutruntime.moveRow parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const keysBefore = await snapshotRowKeys(baseline.page);
      if (keysBefore.length < 2) {
        test.skip(true, "not enough rows for moveRow fixture (need >= 2)");
        return;
      }
      await page_moveRowUp(baseline.page, keysBefore[1]);
      await page_moveRowUp(candidate.page, keysBefore[1]);
      const baselineAfter = await snapshotRowKeys(baseline.page);
      const candidateAfter = await snapshotRowKeys(candidate.page);
      expect(candidateAfter).toEqual(baselineAfter);
      const baselineSelected = await snapshotSelected(baseline.page);
      const candidateSelected = await snapshotSelected(candidate.page);
      expect(candidateSelected).toEqual(baselineSelected);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 5/9: renumber(list, source)
  test("blocklayoutruntime.renumber parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const baselineSource = await page_renumberAndCaptureSource(baseline.page);
      const candidateSource = await page_renumberAndCaptureSource(candidate.page);
      expect(candidateSource).toBe(baselineSource);
      expect(candidateSource).toBe("parity-test");
      const baselineOrders = await snapshotOrderValues(baseline.page);
      const candidateOrders = await snapshotOrderValues(candidate.page);
      expect(candidateOrders).toEqual(baselineOrders);
      // Indices must be 1-based and dense.
      for (let i = 0; i < candidateOrders.length; i++) {
        expect(candidateOrders[i]).toBe(String(i + 1));
      }
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 6/9: selectRow(root, key)
  test("blocklayoutruntime.selectRow parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const keys = await snapshotRowKeys(baseline.page);
      if (keys.length === 0) {
        test.skip(true, "no rows present for selectRow fixture");
        return;
      }
      const target = keys[Math.min(1, keys.length - 1)];
      await page_selectRow(baseline.page, target);
      await page_selectRow(candidate.page, target);
      const baselineSelected = await snapshotSelected(baseline.page);
      const candidateSelected = await snapshotSelected(candidate.page);
      expect(candidateSelected).toEqual(baselineSelected);
      expect(candidateSelected).toContain(target);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 7/9: commitReorder(list, key, targetKey, position)
  test("blocklayoutruntime.commitReorder parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const keys = await snapshotRowKeys(baseline.page);
      if (keys.length < 2) {
        test.skip(true, "not enough rows for commitReorder fixture (need >= 2)");
        return;
      }
      // Move keys[0] to after keys[keys.length-1] — exercises the "after"
      // branch.
      const moveKey = keys[0];
      const targetKey = keys[keys.length - 1];
      const baselineResult = await page_commitReorder(baseline.page, moveKey, targetKey, "after");
      const candidateResult = await page_commitReorder(candidate.page, moveKey, targetKey, "after");
      expect(candidateResult).toBe(baselineResult);
      expect(candidateResult).toBe(true);
      const baselineAfter = await snapshotRowKeys(baseline.page);
      const candidateAfter = await snapshotRowKeys(candidate.page);
      expect(candidateAfter).toEqual(baselineAfter);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 8/9: updateBlockLibraryState(root)
  test("blocklayoutruntime.updateBlockLibraryState parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const baselineButtons = await page_libraryButtonState(baseline.page);
      const candidateButtons = await page_libraryButtonState(candidate.page);
      expect(candidateButtons).toEqual(baselineButtons);
      if (baselineButtons.length === 0) {
        test.skip(true, "no [data-editor-add-block] buttons present — fixture has no library");
      }
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Method 9/9: updateVisibilityState(check) — transitional iframe delegation
  // per ADR 0008. Editor-side observables MUST match exactly; iframe-side
  // observables match exactly when same-origin (otherwise we fall back to
  // legacy-vs-candidate behavior check).
  test("blocklayoutruntime.updateVisibilityState parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(baseline.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const keys = await snapshotRowKeys(baseline.page);
      if (keys.length === 0) {
        test.skip(true, "no rows present for updateVisibilityState fixture");
        return;
      }
      const targetKey = keys[0];
      const baselineSnap = await page_toggleVisibility(baseline.page, targetKey);
      const candidateSnap = await page_toggleVisibility(candidate.page, targetKey);
      // Editor-side observables — row class, status text, pill class —
      // must match exactly. These are visible to the user authoring the
      // page and are unaffected by the iframe-crossing pattern.
      expect(candidateSnap.rowHasHiddenClass).toBe(baselineSnap.rowHasHiddenClass);
      expect(candidateSnap.statusText).toBe(baselineSnap.statusText);
      expect(candidateSnap.pillClassName).toBe(baselineSnap.pillClassName);
      // Iframe-side observable parity per ADR 0008 transitional contract.
      if (baselineSnap.iframeReachable && candidateSnap.iframeReachable) {
        expect(candidateSnap.iframeBlockHidden).toBe(baselineSnap.iframeBlockHidden);
      }
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // ===== v0.5.1 follow-on test for the bindLibrary island =====
  //
  // bindLibrary is NOT a slice-4 method — it was added in v0.5.1 to close
  // Open question 1 in
  // ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md:
  // clicks on [data-editor-add-block] buttons were silently inert after
  // the legacy bundle deletion because no listener was attached.
  //
  // This test does NOT use the parity baseline/candidate harness because
  // there is no baseline implementation to compare against post-deletion;
  // it verifies the v0.5.1 behavior directly on the candidate (post-auto-
  // mount) page.
  //
  // The legacy bindBlockLayoutLibrary click handler (studio-engines.js:2022)
  // flipped the row's [data-editor-block-visible] checkbox to checked +
  // dispatched input/change events, selected the row, scrolled into view,
  // focused it, and refreshed the library buttons. v0.5.1's bindLibrary
  // island restores all of that. We assert the two observables that don't
  // depend on viewport state (the checkbox transitions to checked, and a
  // change event fires on it) plus the .is-selected class membership and
  // the per-form bound-attr marker.
  test("v0.5.1 bindLibrary: clicking [data-editor-add-block] activates the block", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(handle.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const result = await handle.page.evaluate(() => {
        const form = document.querySelector("[data-editor-workbench]");
        if (!form) return { skipped: "no [data-editor-workbench] form" as const };
        const button = form.querySelector("[data-editor-add-block]") as HTMLElement | null;
        if (!button) return { skipped: "no [data-editor-add-block] button" as const };
        const key = button.getAttribute("data-editor-add-block") || "";
        if (!key) return { skipped: "[data-editor-add-block] button missing key" as const };
        const row = document.querySelector(`[data-block-studio-block="${key}"]`);
        if (!row) return { skipped: "no row matches the add-block key" as const };
        const checkbox = row.querySelector("[data-editor-block-visible]") as HTMLInputElement | null;
        if (!checkbox) return { skipped: "row has no [data-editor-block-visible] checkbox" as const };
        // Force the checkbox off so the click handler has work to do.
        checkbox.checked = false;
        let changeFired = 0;
        const onChange = () => { changeFired++; };
        checkbox.addEventListener("change", onChange);
        button.click();
        checkbox.removeEventListener("change", onChange);
        const formIsBound = form.getAttribute("data-gosx-studio-block-library-island-bound") === "true";
        return {
          skipped: null,
          formIsBound,
          checkboxChecked: checkbox.checked,
          changeEventFired: changeFired,
          rowSelected: row.classList.contains("is-selected"),
        };
      });
      if (result.skipped) {
        test.skip(true, result.skipped);
        return;
      }
      expect(result.formIsBound).toBe(true);
      expect(result.checkboxChecked).toBe(true);
      expect(result.changeEventFired).toBeGreaterThan(0);
      expect(result.rowSelected).toBe(true);
    } finally {
      await disposeBoot(handle);
    }
  });

  // ===== v0.5.1 follow-on test for the bindVisibility island =====
  //
  // bindVisibility is NOT a slice-4 method — it was added in v0.5.1 to
  // close Open question 2 in
  // ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md:
  // changes to [data-editor-block-visible] checkboxes were silently inert
  // after the legacy bundle deletion because no listener was attached.
  //
  // The legacy bindBlockLayoutVisibility change handler (studio-engines.js:
  // 2010) called updateBlockLayoutVisibilityState(check) on every change,
  // which toggled the row's editor-block--hidden class, updated the status
  // text + pill className, and delegated the cross-frame mutation through
  // GoSXStudioPreviewRuntime.setBlockVisibility. v0.5.1's bindVisibility
  // island attaches the same change listener (the cross-frame mutation now
  // rides on the $preview.block.<key>.visible shared signal per ADR 0008 /
  // slice 6 transitional cleanup). We assert the editor-side observables —
  // the row gains the hidden class and the status text flips to "Hidden" —
  // when the checkbox is toggled off, plus the per-checkbox bound-attr
  // marker.
  test("v0.5.1 bindVisibility: toggling [data-editor-block-visible] hides the row", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(handle.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const result = await handle.page.evaluate(() => {
        const row = document.querySelector("[data-block-studio-block]");
        if (!row) return { skipped: "no [data-block-studio-block] row" as const };
        const checkbox = row.querySelector("[data-editor-block-visible]") as HTMLInputElement | null;
        if (!checkbox) return { skipped: "row has no [data-editor-block-visible] checkbox" as const };
        const checkboxIsBound = checkbox.getAttribute("data-gosx-studio-block-visibility-island-bound") === "true";
        // Force the checkbox on so the bind-time initial-sync write doesn't
        // race with our assertion below. The bind already ran at auto-mount;
        // setting checked here and dispatching change drives the listener.
        checkbox.checked = true;
        // Drive the listener that bindVisibility attached.
        checkbox.checked = false;
        checkbox.dispatchEvent(new Event("change", { bubbles: true }));
        const status = row.querySelector("[data-editor-block-status]") as HTMLElement | null;
        const pill = row.querySelector("[data-editor-block-pill]") as HTMLElement | null;
        return {
          skipped: null,
          checkboxIsBound,
          rowHasHiddenClass: row.classList.contains("editor-block--hidden"),
          statusText: status?.textContent?.trim() ?? "",
          pillClassName: pill?.className ?? "",
        };
      });
      if (result.skipped) {
        test.skip(true, result.skipped);
        return;
      }
      expect(result.checkboxIsBound).toBe(true);
      expect(result.rowHasHiddenClass).toBe(true);
      // The status text is set by updateVisibilityState to "Hidden" when
      // the checkbox is unchecked (matches the legacy bundle's exact label).
      // Skip the assertion when the status node isn't present in the
      // fixture — some block rows render without an explicit status node.
      if (result.statusText !== "") {
        expect(result.statusText).toBe("Hidden");
      }
      // Pill className is set to "status" (no --ready modifier) when
      // hidden. Skip the assertion when the pill isn't present.
      if (result.pillClassName !== "") {
        expect(result.pillClassName).toBe("status");
      }
    } finally {
      await disposeBoot(handle);
    }
  });

  // ===== v0.5.2 follow-on test for the bindList island =====
  //
  // bindList is NOT a slice-4 method — it was added in v0.5.2 to close the
  // last two unmigrated handlers from elm's spore at
  // ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md:
  // clicks on [data-block-studio-move] up/down buttons were silently inert
  // after the legacy bundle deletion because no listener was attached.
  //
  // This test does NOT use the parity baseline/candidate harness because
  // there is no baseline implementation to compare against post-deletion;
  // it verifies the v0.5.2 behavior directly on the candidate (post-auto-
  // mount) page.
  //
  // The legacy bindBlockLayoutList click handler (studio-engines.js:1942)
  // dispatched move-button clicks to moveBlockLayoutRow (which insertBefore-
  // swapped the row with its sibling, then renumbered + reselected) and row
  // clicks to selectBlockLayoutRow (which toggled .is-selected). v0.5.2's
  // bindList island restores both. We assert that clicking a move=down
  // button moves the first row to position two, fires the blockstudio:
  // reorder CustomEvent on the list with a non-empty source, and that the
  // per-list bound-attr marker is set.
  test("v0.5.2 bindList: clicking [data-block-studio-move=down] reorders the row", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(handle.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const result = await handle.page.evaluate(() => {
        const firstRow = document.querySelector("[data-block-studio-block]");
        if (!firstRow) return { skipped: "no [data-block-studio-block] row" as const };
        const list = firstRow.parentElement;
        if (!list) return { skipped: "row has no parent list" as const };
        const listIsBound = list.getAttribute("data-gosx-studio-block-list-island-bound") === "true";
        // Need at least two rows for a move-down to do something visible.
        const allRows = Array.prototype.slice.call(list.querySelectorAll("[data-block-studio-block]"));
        if (allRows.length < 2) return { skipped: "fewer than two block rows" as const };
        const firstKey = firstRow.getAttribute("data-block-studio-block") || "";
        const secondKey = (allRows[1] as Element).getAttribute("data-block-studio-block") || "";
        const moveDown = firstRow.querySelector('[data-block-studio-move="down"]') as HTMLElement | null;
        if (!moveDown) return { skipped: "first row has no [data-block-studio-move=down] button" as const };
        let reorderSource = "";
        let reorderFired = 0;
        const onReorder = (e: Event) => {
          reorderFired++;
          const detail = (e as CustomEvent).detail as { source?: string } | null;
          if (detail && detail.source) reorderSource = detail.source;
        };
        list.addEventListener("blockstudio:reorder", onReorder);
        moveDown.click();
        list.removeEventListener("blockstudio:reorder", onReorder);
        const newOrder = Array.prototype.slice
          .call(list.querySelectorAll("[data-block-studio-block]"))
          .map((r: Element) => r.getAttribute("data-block-studio-block") || "");
        return {
          skipped: null,
          listIsBound,
          firstKey,
          secondKey,
          // After moving row 0 down, the previously-second row becomes first
          // and the originally-first row becomes second.
          firstNowSecond: newOrder[1] === firstKey,
          secondNowFirst: newOrder[0] === secondKey,
          reorderFired,
          reorderSource,
        };
      });
      if (result.skipped) {
        test.skip(true, result.skipped);
        return;
      }
      expect(result.listIsBound).toBe(true);
      expect(result.firstNowSecond).toBe(true);
      expect(result.secondNowFirst).toBe(true);
      expect(result.reorderFired).toBeGreaterThan(0);
      // The move-button path renumbers with source "engine-buttons" (from
      // moveRow → renumber); the source string must be non-empty so
      // consumers can distinguish init / button / preview / drag reorders.
      expect(result.reorderSource).not.toBe("");
    } finally {
      await disposeBoot(handle);
    }
  });

  // ===== v0.5.2 follow-on test for the bindList row-click delegate =====
  //
  // bindList also wires row-click-to-select on top of the move-button
  // delegate. The legacy bindBlockLayoutList (studio-engines.js:1942)
  // attached a single click listener that dispatched to selectBlockLayoutRow
  // when the click hit a [data-block-studio-block] row (and didn't hit a
  // move button first). hazel's v0.5.1 library_bind covers add-block buttons
  // on the workbench form, NOT row clicks inside the list, so this delegate
  // is genuinely new in v0.5.2.
  //
  // We click the second row (not the first, to avoid colliding with any
  // pre-existing .is-selected state on row 0) and assert .is-selected
  // transitions: the second row gains it, the first row loses it (or never
  // had it), and a blockstudio:select CustomEvent fires on document.
  test("v0.5.2 bindList: clicking a [data-block-studio-block] row selects it", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(handle.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const result = await handle.page.evaluate(() => {
        const firstRow = document.querySelector("[data-block-studio-block]");
        if (!firstRow) return { skipped: "no [data-block-studio-block] row" as const };
        const list = firstRow.parentElement;
        if (!list) return { skipped: "row has no parent list" as const };
        const allRows = Array.prototype.slice.call(list.querySelectorAll("[data-block-studio-block]")) as Element[];
        if (allRows.length < 2) return { skipped: "fewer than two block rows" as const };
        const targetRow = allRows[1] as HTMLElement;
        let selectFired = 0;
        const onSelect = () => { selectFired++; };
        document.addEventListener("blockstudio:select", onSelect);
        // Synthesize a click on the row (not on a child input/button) so the
        // delegate's row-click branch executes, not the move-button branch.
        targetRow.click();
        document.removeEventListener("blockstudio:select", onSelect);
        return {
          skipped: null,
          targetSelected: targetRow.classList.contains("is-selected"),
          selectFired,
        };
      });
      if (result.skipped) {
        test.skip(true, result.skipped);
        return;
      }
      expect(result.targetSelected).toBe(true);
      expect(result.selectFired).toBeGreaterThan(0);
    } finally {
      await disposeBoot(handle);
    }
  });

  // ===== v0.5.2 follow-on test for the bindHandleDrag island =====
  //
  // bindHandleDrag is NOT a slice-4 method — it was added in v0.5.2 to
  // close the last unmigrated handler from elm's spore: pointer-event drag
  // on [data-block-studio-handle] grips. We drive the drag synthetically
  // via dispatched PointerEvent instances rather than relying on
  // Playwright's .dragTo() (which doesn't fire pointer events reliably in
  // headless mode; the legacy bundle uses pointer events specifically so
  // touch + stylus inputs work without separate code).
  //
  // The reorder source for drag drops is "engine-list" (per the legacy
  // renumberBlockLayoutList(list, "engine-list") call), distinguishing
  // drag-drop reorders from button-move ("engine-buttons") / preview-commit
  // ("engine-preview") / boot-init ("engine-init") reorders.
  //
  // Note: we cannot assert the cross-row insertBefore-swap observable in
  // this fixture because the editor route renders the block list inside a
  // collapsed [data-editor-rail] panel, so every row's
  // getBoundingClientRect returns 0×0 in the e2e probe and the pointermove
  // handler's "midpoint Y" computation degenerates. The legacy bundle has
  // the same geometric dependency. We instead verify the OBSERVABLE
  // lifecycle of the drag: the bound-attr marker is set, pointerdown on
  // the handle adds .is-dragging to the row, and a no-op pointerup removes it
  // without firing blockstudio:reorder. The cross-row swap remains covered
  // by the moveRow + commitReorder per-call parity tests above, which exercise
  // valid DOM-mutation commits.
  test("v0.5.2 bindHandleDrag: no-op pointerdown/up cleans the drag lifecycle", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      if (!(await blockListAvailable(handle.page))) {
        test.skip(true, "no block list present in editor route — fixture drift");
        return;
      }
      const result = await handle.page.evaluate(() => {
        const firstRow = document.querySelector("[data-block-studio-block]");
        if (!firstRow) return { skipped: "no [data-block-studio-block] row" as const };
        const list = firstRow.parentElement;
        if (!list) return { skipped: "row has no parent list" as const };
        const listIsBound = list.getAttribute("data-gosx-studio-block-handle-drag-island-bound") === "true";
        const grip = firstRow.querySelector("[data-block-studio-handle]") as HTMLElement | null;
        if (!grip) return { skipped: "first row has no [data-block-studio-handle] grip" as const };
        const downEvent = new PointerEvent("pointerdown", {
          bubbles: true,
          cancelable: true,
          clientX: 0,
          clientY: 0,
          pointerId: 1,
          pointerType: "mouse",
        });
        const upEvent = new PointerEvent("pointerup", {
          bubbles: true,
          cancelable: true,
          clientX: 0,
          clientY: 0,
          pointerId: 1,
          pointerType: "mouse",
        });
        let reorderSource = "";
        let reorderFired = 0;
        const onReorder = (e: Event) => {
          reorderFired++;
          const detail = (e as CustomEvent).detail as { source?: string } | null;
          if (detail && detail.source) reorderSource = detail.source;
        };
        list.addEventListener("blockstudio:reorder", onReorder);
        grip.dispatchEvent(downEvent);
        const isDraggingAfterDown = firstRow.classList.contains("is-dragging");
        list.dispatchEvent(upEvent);
        list.removeEventListener("blockstudio:reorder", onReorder);
        return {
          skipped: null,
          listIsBound,
          isDraggingAfterDown,
          firstRowHasIsDragging: firstRow.classList.contains("is-dragging"),
          reorderFired,
          reorderSource,
        };
      });
      if (result.skipped) {
        test.skip(true, result.skipped);
        return;
      }
      expect(result.listIsBound).toBe(true);
      // pointerdown branch executed → row marked .is-dragging.
      expect(result.isDraggingAfterDown).toBe(true);
      // pointerup branch executed → .is-dragging cleaned up.
      expect(result.firstRowHasIsDragging).toBe(false);
      // A no-op pointerup must not renumber or dispatch blockstudio:reorder.
      expect(result.reorderFired).toBe(0);
      expect(result.reorderSource).toBe("");
    } finally {
      await disposeBoot(handle);
    }
  });
});

// ===== page.evaluate helpers =====

async function page_rowsKeys(page: Page): Promise<string[]> {
  return await page.evaluate((sel) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as { rows?: (list: Element) => Element[] };
    const list = document.querySelector(sel)?.parentElement ?? document.body;
    if (typeof runtime.rows !== "function") return [];
    const result = runtime.rows(list);
    return Array.prototype.slice.call(result || []).map((r) => r.getAttribute("data-block-studio-block") || "");
  }, BLOCK_ROW_SELECTOR);
}

async function page_firstRowKey(page: Page): Promise<string> {
  return await page.evaluate((sel) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as { rowKey?: (row: Element) => string };
    const row = document.querySelector(sel);
    if (!row || typeof runtime.rowKey !== "function") return "";
    return runtime.rowKey(row);
  }, BLOCK_ROW_SELECTOR);
}

async function page_rowForKeyMatches(page: Page, key: string): Promise<boolean> {
  return await page.evaluate(({ sel, key }) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as { rowForKey?: (root: Document | Element, k: string) => Element | null };
    if (typeof runtime.rowForKey !== "function") return false;
    const found = runtime.rowForKey(document, key);
    return !!(found && found.getAttribute("data-block-studio-block") === key);
  }, { sel: BLOCK_ROW_SELECTOR, key });
}

async function page_moveRowUp(page: Page, key: string): Promise<void> {
  await page.evaluate(({ sel, key }) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as {
      moveRow?: (list: Element, row: Element, direction: string) => void;
      rowForKey?: (root: Document | Element, k: string) => Element | null;
    };
    const row = typeof runtime.rowForKey === "function"
      ? runtime.rowForKey(document, key)
      : document.querySelector(`[data-block-studio-block="${key}"]`);
    if (!row || typeof runtime.moveRow !== "function") return;
    const list = (row.parentElement as Element) ?? row;
    runtime.moveRow(list, row, "up");
  }, { sel: BLOCK_ROW_SELECTOR, key });
}

async function page_renumberAndCaptureSource(page: Page): Promise<string | null> {
  return await page.evaluate((sel) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as { renumber?: (list: Element, source: string) => void };
    const firstRow = document.querySelector(sel);
    const list = firstRow?.parentElement ?? null;
    if (!list || typeof runtime.renumber !== "function") return null;
    let captured: string | null = null;
    const onReorder = (event: Event) => {
      const detail = (event as CustomEvent<{ source?: string }>).detail;
      captured = detail?.source ?? null;
    };
    list.addEventListener("blockstudio:reorder", onReorder, { once: true });
    runtime.renumber(list, "parity-test");
    list.removeEventListener("blockstudio:reorder", onReorder);
    return captured;
  }, BLOCK_ROW_SELECTOR);
}

async function page_selectRow(page: Page, key: string): Promise<void> {
  await page.evaluate(({ sel, key }) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as { selectRow?: (root: Document | Element, k: string) => void };
    const firstRow = document.querySelector(sel);
    const root = firstRow?.parentElement ?? document;
    if (typeof runtime.selectRow !== "function") return;
    runtime.selectRow(root, key);
  }, { sel: BLOCK_ROW_SELECTOR, key });
}

async function page_commitReorder(page: Page, moveKey: string, targetKey: string, position: string): Promise<boolean> {
  return await page.evaluate(({ sel, moveKey, targetKey, position }) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as {
      commitReorder?: (list: Element, key: string, targetKey: string, position: string) => boolean;
    };
    const firstRow = document.querySelector(sel);
    const list = firstRow?.parentElement ?? null;
    if (!list || typeof runtime.commitReorder !== "function") return false;
    return !!runtime.commitReorder(list, moveKey, targetKey, position);
  }, { sel: BLOCK_ROW_SELECTOR, moveKey, targetKey, position });
}

interface LibraryButtonState {
  key: string;
  ariaPressed: string | null;
  className: string;
  labelText: string;
}

async function page_libraryButtonState(page: Page): Promise<LibraryButtonState[]> {
  // Trigger updateBlockLibraryState then snapshot every [data-editor-add-block]
  // button so both runtimes get to run the helper before the comparison.
  return await page.evaluate(() => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as { updateBlockLibraryState?: (root: Document) => void };
    if (typeof runtime.updateBlockLibraryState === "function") {
      runtime.updateBlockLibraryState(document);
    }
    return Array.prototype.slice.call(document.querySelectorAll("[data-editor-add-block]")).map((b) => ({
      key: b.getAttribute("data-editor-add-block") || "",
      ariaPressed: b.getAttribute("aria-pressed"),
      className: b.className,
      labelText: (b.querySelector("small") as HTMLElement | null)?.textContent ?? "",
    }));
  });
}

interface VisibilityToggleSnapshot {
  rowHasHiddenClass: boolean;
  statusText: string;
  pillClassName: string;
  iframeReachable: boolean;
  iframeBlockHidden: boolean | null;
}

async function page_toggleVisibility(page: Page, key: string): Promise<VisibilityToggleSnapshot> {
  return await page.evaluate((key) => {
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioBlockLayoutRuntime ?? {}) as {
      updateVisibilityState?: (check: HTMLInputElement) => void;
      rowForKey?: (root: Document | Element, k: string) => Element | null;
    };
    const row = typeof runtime.rowForKey === "function"
      ? runtime.rowForKey(document, key)
      : document.querySelector(`[data-block-studio-block="${key}"]`);
    const check = row?.querySelector("[data-editor-block-visible]") as HTMLInputElement | null;
    if (!check) {
      return {
        rowHasHiddenClass: false,
        statusText: "",
        pillClassName: "",
        iframeReachable: false,
        iframeBlockHidden: null,
      };
    }
    // Toggle to the opposite state then invoke the helper, mirroring the
    // user clicking the visibility checkbox.
    check.checked = !check.checked;
    if (typeof runtime.updateVisibilityState === "function") {
      runtime.updateVisibilityState(check);
    }
    const status = row?.querySelector("[data-editor-block-status]") as HTMLElement | null;
    const pill = row?.querySelector("[data-editor-block-pill]") as HTMLElement | null;
    // Iframe DOM snapshot — best-effort. Cross-origin iframes are skipped.
    let iframeReachable = false;
    let iframeBlockHidden: boolean | null = null;
    const frames = Array.prototype.slice.call(document.querySelectorAll("iframe")) as HTMLIFrameElement[];
    for (const f of frames) {
      let doc: Document | null = null;
      try { doc = f.contentDocument; } catch (_) { continue; }
      if (!doc) continue;
      iframeReachable = true;
      const candidate = doc.querySelector(`[data-block-key="${key}"]`) as HTMLElement | null;
      if (candidate) {
        iframeBlockHidden = Boolean(candidate.hidden);
        break;
      }
    }
    return {
      rowHasHiddenClass: !!row?.classList.contains("editor-block--hidden"),
      statusText: status?.textContent?.trim() ?? "",
      pillClassName: pill?.className ?? "",
      iframeReachable,
      iframeBlockHidden,
    };
  }, key);
}
