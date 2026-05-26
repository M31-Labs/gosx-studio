// Package blocklayoutruntime is the Go-side surface for the Phase 3 slice-4
// burn-down of GoSXStudioBlockLayoutRuntime.
//
// The slice replaces the JavaScript-implemented BlockLayoutRuntime (nine
// public methods declared at gosx-studio/assets/studio-engines.js:2353 —
// rows, rowKey, rowForKey, moveRow, renumber, selectRow, commitReorder,
// updateBlockLibraryState, updateVisibilityState) with .gsx-authored islands
// living in this package's *.gsx files. Each island publishes itself onto a
// well-known window global (see BridgeShim below); the JS shim emitted by
// BridgeShim delegates window.GoSXStudioBlockLayoutRuntime calls to those
// globals when the FeatureFlagKey flag is on in studio.ShellConfig.FeatureFlags.
//
// Both implementations ship additively for at least seven days of CI green
// (see ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
// Section G). After the deletion window closes, the legacy JS implementation
// (blockRows / blockRowKey / blockRowForKey / moveBlockLayoutRow /
// renumberBlockLayoutList / selectBlockLayoutRow / commitBlockLayoutReorder /
// updateBlockLayoutLibraryState / updateBlockLayoutVisibilityState) is removed
// and the shim collapses to a thin pass-through.
//
// # Method fan-out
//
// The nine methods divide into three groups:
//
//  1. Pure helpers (read-only DOM queries):
//
//   - rows(list) — Array.from(list.querySelectorAll("[data-block-studio-block]")).
//   - rowKey(row) — row.getAttribute("data-block-studio-block") || "".
//   - rowForKey(root, key) — root.querySelector('[data-block-studio-block="<key>"]').
//
//  2. Editor-side mutations (block-list ordering / visibility / library state):
//
//   - moveRow(list, row, "up"|"down") — DOM swap + renumber + selectRow.
//   - renumber(list, source) — re-indexes order inputs + data-block-studio-index +
//     toggles up/down disabled states + dispatches blockstudio:reorder /
//     input / change events.
//   - selectRow(root, key) — toggles .is-selected class + scrollIntoView +
//     dispatches blockstudio:select CustomEvent.
//   - commitReorder(list, key, targetKey, "before"|"after") — DOM insertBefore +
//     renumber + selectRow; returns false on no-op.
//   - updateBlockLibraryState(root) — syncs every [data-editor-add-block]
//     button's className / aria-pressed / label text against the
//     corresponding row's visibility checkbox.
//
//  3. Transitional iframe-crossing (1 method):
//
//   - updateVisibilityState(check) — mutates the editor-side row's classes /
//     status text / pill className, then calls
//     window.GoSXStudioPreviewRuntime.setBlockVisibility(key, visible) to
//     update the storefront preview iframe, then updates library state.
//     Per ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
//     this is the transitional pattern — ownership has moved to the block-
//     layout island, the mechanism (iframe DOM mutation via PreviewRuntime)
//     stays until slice 6 collapses it to a $preview.block.<key>.visible
//     shared-signal write subscribed by the preview document. The slice 4
//     island keeps the same iframe contract by delegating to the legacy
//     PreviewRuntime function — slice 6 will swap the mechanism without
//     revisiting block-layout ownership.
//
// Each method is exercised by its own test() call in
// parity_e2e/blocklayoutruntime_test.ts so the parity claim is not collapsed
// to a single boolean.
package blocklayoutruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// FeatureFlagKey is the studio.ShellConfig.FeatureFlags key consumers set to
// activate the .gsx-island BlockLayoutRuntime path. Default value (omitted
// from a host's flag map) keeps the legacy JS implementation active.
//
// Phase 3 naming convention: "<contract>-runtime-islands". Slices 1, 2, 3,
// 5..7 each declare their own constant of the same shape.
const FeatureFlagKey = "block-layout-runtime-islands"

// IslandGlobals lists the window globals each BlockLayout island publishes
// itself at when it mounts. The shim returned by BridgeShim looks these up
// at call time so unmounted islands fall back to the legacy implementation.
//
// Naming convention: window.__gosx_blocklayout_runtime_island_<methodName>.
// BlockLayoutRuntime declares nine public methods (rows, rowKey, rowForKey,
// moveRow, renumber, selectRow, commitReorder, updateBlockLibraryState,
// updateVisibilityState), so the struct holds nine fields.
var IslandGlobals = struct {
	Rows                    string
	RowKey                  string
	RowForKey               string
	MoveRow                 string
	Renumber                string
	SelectRow               string
	CommitReorder           string
	UpdateBlockLibraryState string
	UpdateVisibilityState   string
}{
	Rows:                    "__gosx_blocklayout_runtime_island_rows",
	RowKey:                  "__gosx_blocklayout_runtime_island_rowKey",
	RowForKey:               "__gosx_blocklayout_runtime_island_rowForKey",
	MoveRow:                 "__gosx_blocklayout_runtime_island_moveRow",
	Renumber:                "__gosx_blocklayout_runtime_island_renumber",
	SelectRow:               "__gosx_blocklayout_runtime_island_selectRow",
	CommitReorder:           "__gosx_blocklayout_runtime_island_commitReorder",
	UpdateBlockLibraryState: "__gosx_blocklayout_runtime_island_updateBlockLibraryState",
	UpdateVisibilityState:   "__gosx_blocklayout_runtime_island_updateVisibilityState",
}

// BridgeShim returns the JavaScript snippet that gosx-studio's runtime
// bundle appends to gate window.GoSXStudioBlockLayoutRuntime through the
// FeatureFlagKey flag.
//
// When the host has marked FeatureFlagKey true (read by the consumer and
// surfaced to the page via the data-gosx-studio-feature-flag-<flag>
// attribute on <html> or any element with that attribute), the shim
// dispatches each method to its corresponding IslandGlobals entry on
// window. When the flag is off or the island global is missing (island
// never mounted), the shim falls back to the legacy JS implementation that
// already lives in studio-engines.js.
//
// The shim itself never imports the legacy implementation directly; it
// references the existing in-bundle functions by name (blockRows /
// blockRowKey / blockRowForKey / moveBlockLayoutRow / renumberBlockLayoutList
// / selectBlockLayoutRow / commitBlockLayoutReorder /
// updateBlockLayoutLibraryState / updateBlockLayoutVisibilityState) so the
// bundle remains a single self-contained IIFE. After Section G deletes the
// legacy functions, the shim is rewritten to remove the fallback branch.
func BridgeShim() []byte {
	return []byte(bridgeShimJS)
}

// IslandRuntimeJS returns the JS that publishes the
// window.__gosx_blocklayout_runtime_island_* globals BridgeShim delegates to.
// Sourced from island_runtime.js (embedded). Companion to the .gsx islands
// in this package: the .gsx files are the mount-point markers in the editor
// DOM; this script is the actual behavior layer for slice 4, sized so the
// slice ships independently of the inspector form being signal-driven
// (which is a separate workstream).
func IslandRuntimeJS() []byte {
	return islandRuntimeJS
}

// Bundle returns the IslandRuntimeJS + BridgeShim concatenation that the
// studio runtime asset pipeline appends to studio-engines.js. Order matters:
// the island runtime publishes globals; the shim consults them.
func Bundle() []byte {
	island := IslandRuntimeJS()
	shim := BridgeShim()
	out := make([]byte, 0, len(island)+1+len(shim))
	out = append(out, island...)
	out = append(out, '\n')
	out = append(out, shim...)
	return out
}

const bridgeShimJS = `;(function () {
  // Phase 3 slice-4 BlockLayoutRuntime island bridge.
  // See gosx-studio/blocklayoutruntime/runtime.go for the contract.
  // Feature flag: ` + FeatureFlagKey + `
  if (typeof window === "undefined") return;
  // The consumer surfaces ShellConfig.FeatureFlags via a
  // data-gosx-studio-feature-flag-<key>="true" attribute on the document
  // root (or any ancestor of the editor mount). This keeps the flag
  // decision out of cookies / query params at the bundle level so SSR and
  // CSR observations agree. The attribute name below MUST be the literal
  // "data-gosx-studio-feature-flag-block-layout-runtime-islands" so a grep
  // over the bundle finds it; the test
  // TestEngineRuntimeIncludesBlockLayoutRuntimeIslandBundle enforces this.
  var FLAG_ATTR = "data-gosx-studio-feature-flag-block-layout-runtime-islands";
  function flagEnabled() {
    try {
      if (document.documentElement && document.documentElement.getAttribute(FLAG_ATTR) === "true") return true;
      var marked = document.querySelector("[" + FLAG_ATTR + "='true']");
      return !!marked;
    } catch (e) {
      return false;
    }
  }
  function delegate(islandGlobal, legacyFn) {
    return function () {
      if (flagEnabled()) {
        var island = window[islandGlobal];
        if (typeof island === "function") {
          return island.apply(null, arguments);
        }
      }
      if (typeof legacyFn === "function") {
        return legacyFn.apply(null, arguments);
      }
      return undefined;
    };
  }
  // Replace the runtime object emitted by studio-engines.js with a shim
  // that consults the feature flag on every call. The legacy functions
  // (blockRows / blockRowKey / blockRowForKey / moveBlockLayoutRow /
  // renumberBlockLayoutList / selectBlockLayoutRow / commitBlockLayoutReorder /
  // updateBlockLayoutLibraryState / updateBlockLayoutVisibilityState) remain
  // defined in the bundle and are passed in as the fallback path. The
  // literal island-global names below MUST match
  // blocklayoutruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  var legacy = window.GoSXStudioBlockLayoutRuntime || {};
  window.GoSXStudioBlockLayoutRuntime = {
    rows: delegate(
      "__gosx_blocklayout_runtime_island_rows",
      typeof blockRows === "function" ? blockRows : legacy.rows
    ),
    rowKey: delegate(
      "__gosx_blocklayout_runtime_island_rowKey",
      typeof blockRowKey === "function" ? blockRowKey : legacy.rowKey
    ),
    rowForKey: delegate(
      "__gosx_blocklayout_runtime_island_rowForKey",
      typeof blockRowForKey === "function" ? blockRowForKey : legacy.rowForKey
    ),
    moveRow: delegate(
      "__gosx_blocklayout_runtime_island_moveRow",
      typeof moveBlockLayoutRow === "function" ? moveBlockLayoutRow : legacy.moveRow
    ),
    renumber: delegate(
      "__gosx_blocklayout_runtime_island_renumber",
      typeof renumberBlockLayoutList === "function" ? renumberBlockLayoutList : legacy.renumber
    ),
    selectRow: delegate(
      "__gosx_blocklayout_runtime_island_selectRow",
      typeof selectBlockLayoutRow === "function" ? selectBlockLayoutRow : legacy.selectRow
    ),
    commitReorder: delegate(
      "__gosx_blocklayout_runtime_island_commitReorder",
      typeof commitBlockLayoutReorder === "function" ? commitBlockLayoutReorder : legacy.commitReorder
    ),
    updateBlockLibraryState: delegate(
      "__gosx_blocklayout_runtime_island_updateBlockLibraryState",
      typeof updateBlockLayoutLibraryState === "function" ? updateBlockLayoutLibraryState : legacy.updateBlockLibraryState
    ),
    updateVisibilityState: delegate(
      "__gosx_blocklayout_runtime_island_updateVisibilityState",
      typeof updateBlockLayoutVisibilityState === "function" ? updateBlockLayoutVisibilityState : legacy.updateVisibilityState
    )
  };
})();
`
