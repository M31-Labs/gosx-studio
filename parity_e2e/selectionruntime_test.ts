// Phase 3 slice-2 parity tests for GoSXStudioSelectionRuntime.
//
// Six scenarios:
//   - @coverage         — confirms the candidate boot exposes the island
//                         global. Self-skips with a descriptive reason when
//                         the gosx-studio module bump hasn't landed in the
//                         consumer yet (same precondition as slice 1).
//   - block selection   — clicks a block row, asserts data-studio-selection
//                         and data-studio-selection-kind="block".
//   - workspace target  — clicks a workspace target link, asserts
//                         data-studio-workspace-selection and
//                         data-studio-selection-kind="workspace".
//   - field focus       — focuses a field-source input, asserts
//                         data-studio-field-selection.
//   - commandbar action — clicks a data-studio-selection-action button,
//                         asserts the row's .is-selected class state matches.
//   - style scope       — asserts data-studio-style-scope-block/field
//                         readouts and data-studio-style-state attribute
//                         stay in sync between modes after selection.
//
// Each test runs the same user interaction in baseline (legacy JS active)
// and candidate (island runtime active) modes, snapshots observable state,
// and asserts equivalence. See ./harness.ts and the slice plan
// (~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-2-selectionruntime.md)
// for the deletion gate this suite feeds.
//
// Per the parity matrix (gosx-studio-runtime-parity-matrix.md), the slice
// plan author MUST enumerate each sub-binding as its own sub-test rather
// than collapse the bind() fan-out to a single assertion. Five sub-test
// calls below cover block / workspace / field-focus / commandbar / style-
// scope. The @coverage test is the harness's prerequisite gate, not a
// sub-behavior.

import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { bootBaseline, bootCandidate, disposeBoot } from "./harness";

// Selector for the workbench form root the legacy bindSelectionSurface
// resolves via editorWorkbench(document). Stable contract from the editor
// route — the muddy-noni-commerce editor places this on its top-level form.
const WORKBENCH_SELECTOR = "[data-studio-workbench]";

// Block row selector — every block row in the editor inspector carries
// data-block-studio-block as its key. The first row is the simplest
// fixture for a click-to-select scenario.
const BLOCK_ROW_SELECTOR = "[data-block-studio-block]";

// Workspace target selectors — links the editor's advanced-mode workspace
// uses to switch between editable surfaces.
const WORKSPACE_TARGET_SELECTOR = "[data-studio-panel-link], [data-studio-site-page]";

// Field-source selector — inspector input that fires the field-focus
// sub-behavior on focusin.
const FIELD_SOURCE_SELECTOR = "[data-studio-field-source]";

// Selection-action button selector — selection commandbar buttons. The
// "reveal" action is the safest target for a click parity assertion
// because it doesn't depend on PreviewRuntime (slice 6) being island-ified.
const SELECTION_ACTION_REVEAL_SELECTOR = '[data-studio-selection-action="reveal"]';

interface CandidateFlagSnapshot {
  flagAttrPresent: boolean;
  flagAttrValue: string | null;
  selectionRuntimeGlobalShape: { bind: boolean };
  islandRuntimeAvailable: { bind: boolean };
}

async function snapshotFlag(page: Page): Promise<CandidateFlagSnapshot> {
  return await page.evaluate(() => {
    // The feature flag attribute is surfaced on the workbench root (see
    // lessons/phase-3-slice-1-fieldruntime-deletion-log.md), not on
    // documentElement. Fall back to documentElement for backward compat.
    const attr = "data-gosx-studio-feature-flag-selection-runtime-islands";
    const root =
      document.querySelector(`[${attr}]`) ?? document.documentElement;
    const w = window as unknown as Record<string, unknown>;
    const runtime = (w.GoSXStudioSelectionRuntime ?? {}) as Record<string, unknown>;
    return {
      flagAttrPresent: root.hasAttribute(attr),
      flagAttrValue: root.getAttribute(attr),
      selectionRuntimeGlobalShape: {
        bind: typeof runtime.bind === "function",
      },
      islandRuntimeAvailable: {
        bind: typeof w.__gosx_selection_runtime_island_bind === "function",
      },
    };
  });
}

interface SelectionSnapshot {
  selection: string | null;
  workspaceSelection: string | null;
  selectionKind: string | null;
  selectionLabel: string;
  selectedRowKey: string | null;
}

async function snapshotSelection(page: Page): Promise<SelectionSnapshot> {
  return await page.evaluate((sel) => {
    const form = document.querySelector(sel.workbench);
    const selected = form ? form.querySelector("[data-block-studio-block].is-selected") : null;
    const label = form ? form.querySelector("[data-studio-selection-label]") : null;
    return {
      selection: form ? form.getAttribute("data-studio-selection") : null,
      workspaceSelection: form ? form.getAttribute("data-studio-workspace-selection") : null,
      selectionKind: form ? form.getAttribute("data-studio-selection-kind") : null,
      selectionLabel: label?.textContent?.trim() ?? "",
      selectedRowKey: selected ? selected.getAttribute("data-block-studio-block") : null,
    };
  }, { workbench: WORKBENCH_SELECTOR });
}

interface FieldFocusSnapshot {
  fieldSelection: string | null;
  fieldEditable: string | null;
  activeRowPresent: boolean;
}

async function snapshotFieldFocus(page: Page): Promise<FieldFocusSnapshot> {
  return await page.evaluate((sel) => {
    const form = document.querySelector(sel.workbench);
    const active = form ? form.querySelector(".is-studio-field-active") : null;
    return {
      fieldSelection: form ? form.getAttribute("data-studio-field-selection") : null,
      fieldEditable: form ? form.getAttribute("data-studio-field-editable") : null,
      activeRowPresent: active !== null,
    };
  }, { workbench: WORKBENCH_SELECTOR });
}

interface StyleScopeSnapshot {
  styleState: string | null;
  styleBreakpoint: string | null;
  scopeBlockReadout: string;
  scopeFieldReadout: string;
}

async function snapshotStyleScope(page: Page): Promise<StyleScopeSnapshot> {
  return await page.evaluate((sel) => {
    const form = document.querySelector(sel.workbench);
    const block = form ? form.querySelector("[data-studio-style-scope-block]") : null;
    const field = form ? form.querySelector("[data-studio-style-scope-field]") : null;
    return {
      styleState: form ? form.getAttribute("data-studio-style-state") : null,
      styleBreakpoint: form ? form.getAttribute("data-studio-style-breakpoint") : null,
      scopeBlockReadout: block?.textContent?.trim() ?? "",
      scopeFieldReadout: field?.textContent?.trim() ?? "",
    };
  }, { workbench: WORKBENCH_SELECTOR });
}

test.describe("GoSXStudioSelectionRuntime parity", () => {
  // The harness boots both modes against the same consumer process via the
  // ?gosx-studio-parity= query parameter. Until the consumer bumps its
  // gosx-studio module dependency to include the selectionruntime island
  // bundle, both modes load the same legacy bundle and the assertions
  // trivially succeed. The "@coverage" test below detects this and warns
  // so the team doesn't ship a false-positive parity claim.

  test("@coverage candidate boot exposes selection island runtime global", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      const snap = await snapshotFlag(handle.page);
      expect(snap.flagAttrPresent).toBe(true);
      expect(snap.flagAttrValue).toBe("true");
      // The next assertion is the "real" coverage check. Until the
      // muddy-noni-commerce gosx-studio bump lands, the island runtime
      // global is absent and the test marks itself skipped with a
      // descriptive reason so CI surfaces the prerequisite plainly.
      test.skip(
        !snap.islandRuntimeAvailable.bind,
        "gosx-studio module bump not yet landed in muddy-noni-commerce: " +
          "__gosx_selection_runtime_island_bind is undefined. The parity tests below " +
          "still run but only verify legacy-path stability. Land the dependency " +
          "bump and rerun to get true parity coverage.",
      );
      expect(snap.islandRuntimeAvailable.bind).toBe(true);
      expect(snap.selectionRuntimeGlobalShape.bind).toBe(true);
    } finally {
      await disposeBoot(handle);
    }
  });

  // Sub-behavior 1 of 5: block selection.
  test("selectionruntime.bind block-selection parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      await clickFirstBlockRow(baseline.page);
      await clickFirstBlockRow(candidate.page);

      const baselineSnap = await snapshotSelection(baseline.page);
      const candidateSnap = await snapshotSelection(candidate.page);

      // Both modes must agree on the selected key and selection kind.
      expect(candidateSnap.selection).toBe(baselineSnap.selection);
      expect(candidateSnap.selectionKind).toBe(baselineSnap.selectionKind);
      expect(candidateSnap.selectedRowKey).toBe(baselineSnap.selectedRowKey);

      // When a block row was actually clicked, kind should be "block".
      if (baselineSnap.selectedRowKey) {
        expect(baselineSnap.selectionKind).toBe("block");
        expect(candidateSnap.selectionKind).toBe("block");
      }
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Sub-behavior 2 of 5: workspace target selection.
  test("selectionruntime.bind workspace-target parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      await clickFirstWorkspaceTarget(baseline.page);
      await clickFirstWorkspaceTarget(candidate.page);

      const baselineSnap = await snapshotSelection(baseline.page);
      const candidateSnap = await snapshotSelection(candidate.page);

      // Both modes must agree on whether workspace selection landed.
      expect(candidateSnap.workspaceSelection).toBe(baselineSnap.workspaceSelection);
      expect(candidateSnap.selectionKind).toBe(baselineSnap.selectionKind);

      // When a workspace target was actually clicked, kind should be
      // "workspace" and workspaceSelection should be a non-empty scope.
      if (baselineSnap.workspaceSelection) {
        expect(baselineSnap.selectionKind).toBe("workspace");
        expect(candidateSnap.selectionKind).toBe("workspace");
        expect(candidateSnap.workspaceSelection).toBe(baselineSnap.workspaceSelection);
      }
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Sub-behavior 3 of 5: field focus.
  test("selectionruntime.bind field-focus parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      await focusFirstFieldSource(baseline.page);
      await focusFirstFieldSource(candidate.page);

      const baselineSnap = await snapshotFieldFocus(baseline.page);
      const candidateSnap = await snapshotFieldFocus(candidate.page);

      expect(candidateSnap.fieldSelection).toBe(baselineSnap.fieldSelection);
      expect(candidateSnap.fieldEditable).toBe(baselineSnap.fieldEditable);
      expect(candidateSnap.activeRowPresent).toBe(baselineSnap.activeRowPresent);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Sub-behavior 4 of 5: selection commandbar action.
  test("selectionruntime.bind commandbar-action parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      // First select a block so the commandbar has something to act on.
      await clickFirstBlockRow(baseline.page);
      await clickFirstBlockRow(candidate.page);

      await clickSelectionAction(baseline.page, "reveal");
      await clickSelectionAction(candidate.page, "reveal");

      // After "reveal", the selected row in both modes should still be set
      // (reveal is non-destructive — it scrolls + selects but doesn't clear).
      const baselineSnap = await snapshotSelection(baseline.page);
      const candidateSnap = await snapshotSelection(candidate.page);
      expect(candidateSnap.selectedRowKey).toBe(baselineSnap.selectedRowKey);
      expect(candidateSnap.selection).toBe(baselineSnap.selection);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });

  // Sub-behavior 5 of 5: style-scope readouts.
  test("selectionruntime.bind style-scope-readout parity", async ({ browser }) => {
    const baseline = await bootBaseline(browser);
    const candidate = await bootCandidate(browser);
    try {
      // Select a block so the style scope has a label to render.
      await clickFirstBlockRow(baseline.page);
      await clickFirstBlockRow(candidate.page);

      const baselineSnap = await snapshotStyleScope(baseline.page);
      const candidateSnap = await snapshotStyleScope(candidate.page);

      // Style state attribute should resolve to one of the known values in
      // both modes; the legacy implementation normalizes any unknown value
      // to "default", so candidate must do the same.
      expect(candidateSnap.styleState).toBe(baselineSnap.styleState);
      expect(candidateSnap.scopeBlockReadout).toBe(baselineSnap.scopeBlockReadout);
      expect(candidateSnap.scopeFieldReadout).toBe(baselineSnap.scopeFieldReadout);
      // The breakpoint readout is derived from the workbench viewport; both
      // modes should derive the same value because they share the
      // WorkbenchRuntime (slice 7 hasn't burned that down yet).
      expect(candidateSnap.styleBreakpoint).toBe(baselineSnap.styleBreakpoint);
    } finally {
      await disposeBoot(baseline);
      await disposeBoot(candidate);
    }
  });
});

async function clickFirstBlockRow(page: Page): Promise<void> {
  const row = page.locator(BLOCK_ROW_SELECTOR).first();
  if ((await row.count()) === 0) {
    test.skip(true, "no block rows present in editor route — fixture drift");
    return;
  }
  // Block rows live inside an inspector panel that may be CSS-hidden when
  // its parent mode isn't active. The runtime binding we're exercising
  // listens for click events regardless of layout visibility — drive the
  // click through the underlying DOM API so the test is independent of
  // editor-panel-visibility drift.
  await row.evaluate((el) => (el as HTMLElement).click());
  // Allow rAF-throttled style-scope update to flush.
  await page.waitForTimeout(50);
}

async function clickFirstWorkspaceTarget(page: Page): Promise<void> {
  const target = page.locator(WORKSPACE_TARGET_SELECTOR).first();
  if ((await target.count()) === 0) {
    test.skip(true, "no workspace targets present in editor route — fixture drift");
    return;
  }
  // Workspace target links live inside the workspace nav which may be CSS-
  // hidden when collapsed. See clickFirstBlockRow above for rationale.
  await target.evaluate((el) => (el as HTMLElement).click());
  await page.waitForTimeout(50);
}

async function focusFirstFieldSource(page: Page): Promise<void> {
  const source = page.locator(FIELD_SOURCE_SELECTOR).first();
  if ((await source.count()) === 0) {
    test.skip(true, "no field-source inputs present in editor route — fixture drift");
    return;
  }
  // focusin is what bindSelectionSurface listens for on field-source inputs.
  // Fire focus through the DOM API so we don't depend on the source being
  // visible in the current inspector layout.
  await source.evaluate((el) => {
    const node = el as HTMLElement;
    node.focus();
    node.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
  });
  await page.waitForTimeout(50);
}

async function clickSelectionAction(page: Page, action: string): Promise<void> {
  const button = page.locator(`[data-studio-selection-action="${action}"]`).first();
  if ((await button.count()) === 0) {
    test.skip(true, `no selection-action button for "${action}" — fixture drift`);
    return;
  }
  // Selection commandbar buttons live in a panel that's only visible when
  // something is selected; drive the click through the DOM API so the
  // runtime binding is exercised regardless of panel visibility state.
  await button.evaluate((el) => (el as HTMLElement).click());
  await page.waitForTimeout(50);
}
