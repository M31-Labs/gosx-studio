// island_runtime.js — companion JS for the blocklayoutruntime .gsx islands.
//
// The .gsx islands (block_rows.gsx, block_reorder.gsx, block_select.gsx,
// block_library.gsx, block_visibility.gsx) are mount-point markers in the
// editor DOM. This script publishes the nine
// window.__gosx_blocklayout_runtime_island_<method> globals that
// blocklayoutruntime.BridgeShim delegates to when the
// "block-layout-runtime-islands" feature flag is on. It is emitted into the
// studio runtime bundle by blocklayoutruntime.IslandRuntimeJS() (see
// runtime.go) and runs before BridgeShim() at bundle init time.
//
// The nine functions below replace window.GoSXStudioBlockLayoutRuntime.{
// rows, rowKey, rowForKey, moveRow, renumber, selectRow, commitReorder,
// updateBlockLibraryState, updateVisibilityState } while preserving exact
// observable behavior of the legacy implementations (blockRows /
// blockRowKey / blockRowForKey / moveBlockLayoutRow /
// renumberBlockLayoutList / selectBlockLayoutRow / commitBlockLayoutReorder
// / updateBlockLayoutLibraryState / updateBlockLayoutVisibilityState at
// assets/studio-engines.js:1960–2110).
//
// # Method fan-out (group 1: pure helpers)
//
// rows(list)        — list.querySelectorAll("[data-block-studio-block]"),
//                     returned as an Array (legacy uses Array.prototype.slice).
// rowKey(row)       — row.getAttribute("data-block-studio-block") || "".
// rowForKey(root, k) — root.querySelector('[data-block-studio-block="<k>"]'),
//                     with the key escaped for attribute-value quoting.
//
// # Method fan-out (group 2: editor-side mutations)
//
// moveRow(list, row, "up"|"down") — DOM insertBefore neighbor swap, then
//                                   renumber(list, "engine-buttons") +
//                                   selectRow(list, rowKey(row)).
// renumber(list, source)          — re-indexes [data-block-studio-order]
//                                   inputs to 1-based numbers, sets
//                                   data-block-studio-index, refreshes
//                                   move-button disabled state, fires
//                                   blockstudio:reorder + input + change
//                                   events on list.
// selectRow(root, key)            — toggles .is-selected on every row,
//                                   scrollIntoView's the new row, fires
//                                   blockstudio:select on document.
// commitReorder(list, key, target, position) — guards no-op (returns false);
//                                              insertBefore key-row before/after
//                                              target-row, then renumber +
//                                              selectRow; returns true.
// updateBlockLibraryState(root)   — for every [data-editor-add-block] in
//                                   root, sync className / aria-pressed /
//                                   <small> label to row visibility checkbox.
//
// # Method fan-out (group 3: iframe-crossing transitional)
//
// updateVisibilityState(check)    — mutates the row's classes / status text /
//                                   pill className, delegates the cross-frame
//                                   write to GoSXStudioPreviewRuntime
//                                   .setBlockVisibility(key, visible) (per
//                                   ADR 0008 transitional pattern), then
//                                   refreshes library state.
//
// # Idempotency
//
// These nine methods are stateless pure-ish (rows / rowKey / rowForKey are
// pure; the mutation methods don't bind listeners — they perform a single
// imperative DOM update per call), so no idempotency-guard dataset keys are
// required. Compare to bindBrandLogo which attaches event listeners and
// needs gosxStudioBrandLogoIslandBound; selectBlockLayoutRow / moveBlockLayoutRow
// can be called arbitrarily many times without leaking listeners.

;(function () {
  if (typeof window === "undefined") return;
  var doc = window.document;
  if (!doc) return;

  // attrValue mirrors the legacy attrValue helper at studio-engines.js:2158 —
  // escapes backslashes and double quotes so an arbitrary key string can be
  // embedded inside a CSS attribute selector double-quoted value. This is
  // load-bearing for rowForKey: a key like he"ro would break the selector
  // without escaping.
  function attrValue(value) {
    return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
  }

  // ===== Group 1: pure helpers =====

  // rows(list) — mirrors blockRows at studio-engines.js:1960.
  // Returns an Array (legacy uses Array.prototype.slice.call so the result is
  // a real Array, not a NodeList). Callers iterate with .forEach / .find /
  // .filter; preserving the Array contract keeps those callers working.
  function rowsIsland(list) {
    if (!list || !list.querySelectorAll) return [];
    return Array.prototype.slice.call(list.querySelectorAll("[data-block-studio-block]"));
  }

  window.__gosx_blocklayout_runtime_island_rows = rowsIsland;

  // rowKey(row) — mirrors blockRowKey at studio-engines.js:1964.
  // Reads data-block-studio-block off the row. Returns "" when row is null
  // or the attribute is missing — callers rely on this to no-op safely on
  // empty results from rowForKey lookups.
  function rowKeyIsland(row) {
    return row ? row.getAttribute("data-block-studio-block") || "" : "";
  }

  window.__gosx_blocklayout_runtime_island_rowKey = rowKeyIsland;
})();
