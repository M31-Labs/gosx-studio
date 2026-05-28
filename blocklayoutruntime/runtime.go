// Package blocklayoutruntime is the Go-side surface for the Phase 3 slice-4
// burn-down of GoSXStudioBlockLayoutRuntime.
//
// The slice replaces the JavaScript-implemented BlockLayoutRuntime (nine
// public methods declared at the legacy bundle (removed 2026-05-27) —
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
// BlockLayoutRuntime declares nine slice-4 per-call methods (rows, rowKey,
// rowForKey, moveRow, renumber, selectRow, commitReorder,
// updateBlockLibraryState, updateVisibilityState) plus two v0.5.1
// listener-binding methods (bindLibrary, bindVisibility — see the
// "v0.5.1 follow-on" section in island_runtime.js for the rationale and
// the elm spore at ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
// for the bug they close) plus two v0.5.2 listener-binding methods
// (bindList, bindHandleDrag — see the trailing "v0.5.2 follow-on" section
// in island_runtime.js; these close the last two unmigrated handlers the
// elm spore surfaced: move-button click delegation, row click-to-select,
// initial renumber-on-mount, and the pointer-event drag-and-drop reorder
// chain). So the struct holds thirteen fields. With v0.5.2 the
// listener-binding surface of GoSXStudioBlockLayoutRuntime is fully on
// islands; the only legacy bind* helpers remaining in the deleted bundle
// snapshot are unrelated to this contract.
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
	BindLibrary             string
	BindVisibility          string
	BindList                string
	BindHandleDrag          string
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
	BindLibrary:             "__gosx_blocklayout_runtime_island_bindLibrary",
	BindVisibility:          "__gosx_blocklayout_runtime_island_bindVisibility",
	BindList:                "__gosx_blocklayout_runtime_island_bindList",
	BindHandleDrag:          "__gosx_blocklayout_runtime_island_bindHandleDrag",
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
// lived in the now-deleted legacy bundle.
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
// studio runtime asset pipeline serves. Order matters:
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
  function delegate(islandGlobal) {
    return function () {
      if (!flagEnabled()) return undefined;
      var island = window[islandGlobal];
      if (typeof island === "function") {
        return island.apply(null, arguments);
      }
      return undefined;
    };
  }
  // Install the runtime object. Pre-2026-05-27 the BridgeShim wrapped the
  // legacy bundle's nine block-layout binders (blockRows / blockRowKey /
  // blockRowForKey / moveBlockLayoutRow / renumberBlockLayoutList /
  // selectBlockLayoutRow / commitBlockLayoutReorder /
  // updateBlockLayoutLibraryState / updateBlockLayoutVisibilityState) as
  // fallback paths; with the legacy bundle deleted in Phase 3 Section E
  // those identifiers are undefined and the fallback branches were dead
  // code. v0.5.0 removes the dead branches — the island global is the only
  // path. The literal island-global names below MUST match
  // blocklayoutruntime.IslandGlobals in runtime.go — the test
  // TestBridgeShimDelegatesToIslandGlobals enforces this.
  window.GoSXStudioBlockLayoutRuntime = {
    rows: delegate("__gosx_blocklayout_runtime_island_rows"),
    rowKey: delegate("__gosx_blocklayout_runtime_island_rowKey"),
    rowForKey: delegate("__gosx_blocklayout_runtime_island_rowForKey"),
    moveRow: delegate("__gosx_blocklayout_runtime_island_moveRow"),
    renumber: delegate("__gosx_blocklayout_runtime_island_renumber"),
    selectRow: delegate("__gosx_blocklayout_runtime_island_selectRow"),
    commitReorder: delegate("__gosx_blocklayout_runtime_island_commitReorder"),
    updateBlockLibraryState: delegate("__gosx_blocklayout_runtime_island_updateBlockLibraryState"),
    updateVisibilityState: delegate("__gosx_blocklayout_runtime_island_updateVisibilityState"),
    // v0.5.1 follow-on: listener-binding methods. The slice-4 island
    // migration only covered the nine per-call methods above; the deleted
    // studio-engines.js bundle's bindBlockLayoutLibrary and
    // bindBlockLayoutVisibility helpers were the missing piece that made
    // add-block button clicks and visibility checkbox changes actually
    // do something. See the trailing "v0.5.1 follow-on" section in
    // island_runtime.js and elm's spore at
    // ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md.
    bindLibrary: delegate("__gosx_blocklayout_runtime_island_bindLibrary"),
    bindVisibility: delegate("__gosx_blocklayout_runtime_island_bindVisibility"),
    // v0.5.2 follow-on: the last two listener-binding methods. The deleted
    // studio-engines.js bundle also installed bindBlockLayoutList (per-list
    // click delegation for [data-block-studio-move] up/down buttons and
    // [data-block-studio-block] row click-to-select, plus initial
    // renumber-on-mount) and bindBlockLayoutHandleDrag (per-list pointer-
    // event drag-and-drop reorder over [data-block-studio-handle] grips).
    // After v0.5.1 the add-block / visibility checkboxes worked but the
    // move buttons, row clicks, and drag handles remained inert. v0.5.2
    // closes the burn-down. See the trailing "v0.5.2 follow-on" section
    // in island_runtime.js.
    bindList: delegate("__gosx_blocklayout_runtime_island_bindList"),
    bindHandleDrag: delegate("__gosx_blocklayout_runtime_island_bindHandleDrag")
  };
  // Auto-mount on document ready. Mirrors the v0.4.1 fieldruntime fix: the
  // deleted studio-engines.js bundle's engine-mount factory invoked
  // bindBlockLayoutList / bindBlockLayoutLibrary / bindBlockLayoutVisibility
  // for the block-layout engine at boot.
  //
  // v0.5.0 (the prior bridgemount restoration) only restored
  // updateBlockLibraryState — the observable that refreshes the library
  // button states (className / aria-pressed / label) against the current
  // row visibility on initial load. The two listener-binding helpers were
  // explicitly deferred to a follow-up because their migration to islands
  // hadn't shipped yet.
  //
  // v0.5.1 shipped those island migrations (see library_bind.gsx +
  // visibility_bind.gsx + the "v0.5.1 follow-on" section in
  // island_runtime.js); v0.5.2 ships the last two listener-binding islands
  // (see list_bind.gsx + handle_drag_bind.gsx + the trailing "v0.5.2 follow-
  // on" section in island_runtime.js). The auto-mount now invokes the
  // complete set of boot-time observables so add-block button clicks,
  // visibility checkbox changes, move-button clicks, row click-to-select,
  // initial renumber-on-mount, and pointer-event drag-and-drop reorder all
  // work in production — the v0.5.0 spore (Open questions 1 + 2) and the
  // last two unmigrated handlers are all closed.
  //
  // Call order matters:
  //   1. bindVisibility attaches change handlers AND fires
  //      updateVisibilityState(check) once at bind time for each checkbox —
  //      the initial-sync write also refreshes the library state.
  //   2. bindList walks every [data-block-studio-block]'s nearest list
  //      ancestor, attaches the move-button + row-click delegate, marks
  //      rows non-draggable (legacy HTML5 attribute would conflict with the
  //      pointer-event drag handler), and calls renumber(list, "engine-init")
  //      once per list so order inputs / button disabled states / the
  //      blockstudio:reorder event fire on first paint.
  //   3. bindHandleDrag walks every [data-block-studio-handle]'s nearest
  //      list ancestor and attaches the pointer-event drag chain.
  //   4. updateBlockLibraryState(document) explicitly so the library
  //      buttons reflect the post-sync row visibility even when there are
  //      zero checkboxes (in which case bindVisibility runs nothing).
  // bindLibrary is order-independent (it just attaches a click delegate to
  // the workbench form).
  //
  // Idempotent: a per-document marker attribute guards against double-mount
  // of the outer auto-mount IIFE; the bindLibrary / bindVisibility /
  // bindList / bindHandleDrag islands additionally guard themselves
  // per-form / per-checkbox / per-list via dataset markers so a host
  // calling them directly during SSR hydration also doesn't stack listeners.
  function autoMount() {
    try {
      var marker = "data-gosx-studio-blocklayout-runtime-auto-mounted";
      if (document.documentElement && document.documentElement.getAttribute(marker) === "true") return;
      if (document.documentElement) document.documentElement.setAttribute(marker, "true");
      window.GoSXStudioBlockLayoutRuntime.bindLibrary(document);
      window.GoSXStudioBlockLayoutRuntime.bindVisibility(document);
      window.GoSXStudioBlockLayoutRuntime.bindList(document);
      window.GoSXStudioBlockLayoutRuntime.bindHandleDrag(document);
      window.GoSXStudioBlockLayoutRuntime.updateBlockLibraryState(document);
    } catch (e) {
      // Auto-mount must never break the page; the host can always invoke
      // window.GoSXStudioBlockLayoutRuntime.bindLibrary /
      // .bindVisibility / .bindList / .bindHandleDrag /
      // .updateBlockLibraryState manually if it needs to recover.
    }
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", autoMount);
  } else {
    autoMount();
  }
})();
`
