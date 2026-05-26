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

  // rowForKey(root, key) — mirrors blockRowForKey at studio-engines.js:1968.
  // Selects the row whose data-block-studio-block matches the supplied key,
  // scoped to root. attrValue() escape mirrors the legacy attrValue helper so
  // keys with backslashes / double quotes (rare in practice but allowed by
  // the data-attribute spec) don't break the selector.
  function rowForKeyIsland(root, key) {
    if (!root || !root.querySelector) return null;
    return root.querySelector('[data-block-studio-block="' + attrValue(key) + '"]');
  }

  window.__gosx_blocklayout_runtime_island_rowForKey = rowForKeyIsland;

  // ===== Group 2: editor-side mutations =====

  // moveRow(list, row, direction) — mirrors moveBlockLayoutRow at
  // studio-engines.js:2072. direction = "up" or "down". DOM neighbor swap
  // via insertBefore, then renumber(list, "engine-buttons") +
  // selectRow(list, rowKey(row)) to refresh order inputs / button states /
  // selection class. No-op when row is null or sibling is missing (legacy
  // behavior — moving the first row up or the last row down is a silent
  // no-op so callers can dispatch unconditionally).
  function moveRowIsland(list, row, direction) {
    if (!row || !list) return;
    if (direction === "up" && row.previousElementSibling) {
      list.insertBefore(row, row.previousElementSibling);
    }
    if (direction === "down" && row.nextElementSibling) {
      list.insertBefore(row.nextElementSibling, row);
    }
    renumberIsland(list, "engine-buttons");
    selectRowIsland(list, rowKeyIsland(row));
  }

  window.__gosx_blocklayout_runtime_island_moveRow = moveRowIsland;

  // updateBlockMoveButtonsIsland — internal helper mirroring
  // updateBlockMoveButtons at studio-engines.js:2062. Walks every row and
  // sets the row's [data-block-studio-move="up"|"down"] button.disabled
  // based on the row's position. Used by renumberIsland; not a public
  // method.
  function updateBlockMoveButtonsIsland(list) {
    var all = rowsIsland(list);
    all.forEach(function (row, index) {
      var up = row.querySelector('[data-block-studio-move="up"]');
      var down = row.querySelector('[data-block-studio-move="down"]');
      if (up) up.disabled = index === 0;
      if (down) down.disabled = index === all.length - 1;
    });
  }

  // renumber(list, source) — mirrors renumberBlockLayoutList at
  // studio-engines.js:2047. Default source mirrors the legacy
  // "block-layout-engine" string used when callers pass null/undefined.
  // The order input takes a 1-based string; data-block-studio-index is
  // 0-based — matching the legacy convention so the form posts the right
  // order on submit and the engine's index queries see the right ordinal.
  function renumberIsland(list, source) {
    if (!list) return;
    rowsIsland(list).forEach(function (row, index) {
      var input = row.querySelector("[data-block-studio-order]");
      if (input) input.value = String(index + 1);
      row.setAttribute("data-block-studio-index", String(index));
    });
    updateBlockMoveButtonsIsland(list);
    list.dispatchEvent(new CustomEvent("blockstudio:reorder", {
      bubbles: true,
      detail: { source: source || "block-layout-engine" }
    }));
    list.dispatchEvent(new Event("input", { bubbles: true }));
    list.dispatchEvent(new Event("change", { bubbles: true }));
  }

  window.__gosx_blocklayout_runtime_island_renumber = renumberIsland;

  // selectRow(root, key) — mirrors selectBlockLayoutRow at
  // studio-engines.js:2099. Toggles .is-selected on every row to match the
  // supplied key, scrollIntoView's the newly-selected row (block: "nearest"
  // so it doesn't bounce when already in view), and dispatches a
  // blockstudio:select CustomEvent on document so the selection commandbar
  // / inspector form can react. No-op when key is empty — callers rely on
  // this to skip the dispatch on initial bind when nothing is selected yet.
  function selectRowIsland(root, key) {
    if (!key) return;
    rowsIsland(root).forEach(function (row) {
      row.classList.toggle("is-selected", rowKeyIsland(row) === key);
    });
    var row = rowForKeyIsland(root, key);
    if (row && row.scrollIntoView) row.scrollIntoView({ block: "nearest", behavior: "smooth" });
    doc.dispatchEvent(new CustomEvent("blockstudio:select", {
      bubbles: true,
      detail: { key: key }
    }));
  }

  window.__gosx_blocklayout_runtime_island_selectRow = selectRowIsland;

  // commitReorder(list, key, targetKey, position) — mirrors
  // commitBlockLayoutReorder at studio-engines.js:2084. position="before"|
  // "after". Returns false on no-op (missing key/target or key===target);
  // otherwise true. Caller-facing return value matters — the
  // canvas-preview-drop handler in preview-runtime.js branches on the
  // return to decide whether to revert its visual hint.
  function commitReorderIsland(list, key, targetKey, position) {
    if (!key || !targetKey || key === targetKey) return false;
    var row = rowForKeyIsland(list, key);
    var target = rowForKeyIsland(list, targetKey);
    if (!row || !target) return false;
    if (position === "after") {
      list.insertBefore(row, target.nextElementSibling);
    } else {
      list.insertBefore(row, target);
    }
    renumberIsland(list, "engine-preview");
    selectRowIsland(list, key);
    return true;
  }

  window.__gosx_blocklayout_runtime_island_commitReorder = commitReorderIsland;
})();
