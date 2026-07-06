// island_runtime.js — companion JS for the blocklayoutruntime .gsx islands.
//
// The .gsx islands (block_rows.gsx, block_reorder.gsx, block_select.gsx,
// block_library.gsx, block_visibility.gsx) are mount-point markers in the
// editor DOM. This script publishes the nine
// window.__gosx_blocklayout_runtime_island_<method> globals that
// blocklayoutruntime.BridgeShim delegates to directly. It is emitted into
// the studio runtime bundle by blocklayoutruntime.IslandRuntimeJS() (see
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

  // updateBlockLibraryState(root) — mirrors updateBlockLayoutLibraryState at
  // studio-engines.js:1972. For every [data-editor-add-block] button in the
  // supplied root (defaulting to document when root has no querySelectorAll),
  // resolve the row in document by key, read its visibility checkbox, and
  // update the button's className / aria-pressed / <small> label.
  function updateBlockLibraryStateIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    Array.prototype.forEach.call(scope.querySelectorAll("[data-editor-add-block]"), function (button) {
      var key = button.getAttribute("data-editor-add-block");
      var row = rowForKeyIsland(doc, key);
      var checkbox = row && row.querySelector("[data-editor-block-visible]");
      var enabled = !!(checkbox && checkbox.checked);
      var base = button.getAttribute("data-editor-button-base") || "button";
      button.className = base + " " + (enabled ? "button--ghost is-active" : "button--secondary");
      button.setAttribute("aria-pressed", enabled ? "true" : "false");
      var label = button.querySelector("small");
      if (label) label.textContent = enabled ? "On page" : "Add";
    });
  }

  window.__gosx_blocklayout_runtime_island_updateBlockLibraryState = updateBlockLibraryStateIsland;

  // ===== Group 3: iframe-crossing — now via $preview.* signals =====

  // Set a shared signal value through the WASM bridge. The cross-frame
  // relay (per ADR 0009 + Bridge.EnableCrossFrameRelay) routes writes
  // whose name starts with "$preview." to the storefront iframe's
  // Bridge, where slice 6's preview_subscriber observes them and applies
  // the matching DOM mutation locally. Before the bridge boots, the call
  // is a no-op.
  function writeSharedSignal(name, value) {
    try {
      var setter = window.__gosx_set_shared_signal_json;
      if (typeof setter === "function") {
        setter(name, JSON.stringify(value));
      }
    } catch (error) {
      // Shared-signal channel not ready yet.
    }
  }

  // updateVisibilityState(check) — slice 6 transitional cleanup (Section G.3).
  //
  // The previous shape delegated the cross-frame block-hide to
  // window.GoSXStudioPreviewRuntime.setBlockVisibility. Slice 6 swaps the
  // delivery mechanism: instead of reach-across-frame DOM mutation, we
  // publish to $preview.block.<key>.visible and let the cross-frame
  // relay (ADR 0009) deliver the write to the iframe's subscriber where
  // the [data-studio-block-key="<key>"] node's display style gets
  // toggled. Editor-side row class / status text / library-state updates
  // are unchanged — those are editor-frame-only.
  function updateVisibilityStateIsland(check) {
    var row = check && check.closest ? check.closest("[data-block-studio-block]") : null;
    if (!row) return;
    var visible = !!check.checked;
    var key = rowKeyIsland(row);
    var statuses = row.querySelectorAll("[data-editor-block-status], [data-editor-block-pill]");
    row.classList.toggle("editor-block--hidden", !visible);
    Array.prototype.forEach.call(statuses, function (status) {
      status.textContent = visible ? "Visible" : "Hidden";
    });
    var pill = row.querySelector("[data-editor-block-pill]");
    if (pill) pill.className = visible ? "status status--ready" : "status";
    if (key) {
      writeSharedSignal("$preview.block." + key + ".visible", visible);
    }
    updateBlockLibraryStateIsland(doc);
  }

  window.__gosx_blocklayout_runtime_island_updateVisibilityState = updateVisibilityStateIsland;

  // ===== v0.5.1 follow-on: listener-binding islands =====
  //
  // The legacy studio-engines.js bundle (deleted 2026-05-27) installed two
  // listener-binding helpers at boot — bindBlockLayoutLibrary and
  // bindBlockLayoutVisibility — that the slice-4 island migration did not
  // cover (slice 4 only owned the nine per-call methods on
  // GoSXStudioBlockLayoutRuntime). After v0.5.0's bridgemount restoration
  // the per-call methods are reachable, but add-block button clicks and
  // visibility checkbox changes have no listener attached so the buttons /
  // checkboxes render in the correct visual state and silently do nothing
  // when interacted with.
  //
  // v0.5.1 closes the gap by adding two new listener-binding islands
  // (window.GoSXStudioBlockLayoutRuntime.bindLibrary /
  // window.GoSXStudioBlockLayoutRuntime.bindVisibility) and calling both
  // at boot from the BridgeShim auto-mount alongside updateBlockLibraryState.
  // See:
  //   - inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
  //     (Open questions 1 + 2).
  //   - the legacy bindings at
  //     git -C ~/work/gosx-studio show 32dac98:assets/studio-engines.js
  //     lines 2010-2034.

  // editorWorkbenchIsland mirrors the legacy editorWorkbench helper at
  // studio-engines.js:69. Used by bindLibrary to scope the click delegate
  // to the workbench form so clicks outside it (e.g., in nav chrome) are
  // ignored.
  function editorWorkbenchIsland(root) {
    if (root && root.matches && root.matches("[data-editor-workbench]")) return root;
    var scope = root && root.querySelector ? root : doc;
    return scope.querySelector ? scope.querySelector("[data-editor-workbench]") : null;
  }

  // bindLibrary(root) — mirrors the legacy bindBlockLayoutLibrary at
  // studio-engines.js:2022. Resolves the editor workbench form under root,
  // attaches a click delegate that activates the block named by every
  // [data-editor-add-block] button: flips the row's visibility checkbox to
  // checked + dispatches input/change events (so the visibility island
  // fires updateVisibilityState), calls selectRow(form, key), scrolls the
  // row into view, focuses it, and refreshes the library button states.
  //
  // Idempotency: guarded per-form via the
  // data-gosx-studio-block-library-island-bound dataset key (renamed from
  // the legacy gosxStudioBlockLibraryBound to make the post-deletion
  // ownership obvious). A second call against the same form returns
  // silently — listeners do not stack.
  //
  // The handler closes over the form scope and uses event.target.closest
  // so dynamically-inserted add-block buttons (e.g., when a host re-renders
  // the library) work without re-binding — only the form needs to be in
  // the DOM at bind time.
  function bindLibraryIsland(root) {
    var form = editorWorkbenchIsland(root);
    if (!form || form.dataset.gosxStudioBlockLibraryIslandBound === "true") return;
    form.dataset.gosxStudioBlockLibraryIslandBound = "true";
    form.addEventListener("click", function (event) {
      var button = event.target && event.target.closest
        ? event.target.closest("[data-editor-add-block]")
        : null;
      if (!button || !form.contains(button)) return;
      var key = button.getAttribute("data-editor-add-block");
      var row = rowForKeyIsland(form, key);
      if (!row) return;
      var checkbox = row.querySelector("[data-editor-block-visible]");
      if (checkbox && !checkbox.checked) {
        checkbox.checked = true;
        checkbox.dispatchEvent(new Event("input", { bubbles: true }));
        checkbox.dispatchEvent(new Event("change", { bubbles: true }));
      }
      selectRowIsland(form, key);
      var reduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
      if (row.scrollIntoView) {
        row.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });
      }
      window.setTimeout(function () {
        if (row.focus) row.focus();
      }, 180);
      updateBlockLibraryStateIsland(doc);
    });
    updateBlockLibraryStateIsland(doc);
  }

  window.__gosx_blocklayout_runtime_island_bindLibrary = bindLibraryIsland;

  // bindVisibility(root) — mirrors the legacy bindBlockLayoutVisibility at
  // studio-engines.js:2010. For every [data-editor-block-visible] checkbox
  // in scope, attaches a change listener that calls updateVisibilityState
  // (the row hide-class / status text / pill class / library refresh +
  // $preview.block.<key>.visible publish per ADR 0008 / slice 6 transitional
  // cleanup). Also fires updateVisibilityState(check) once at bind time to
  // sync the storefront preview's initial state to the checkbox state.
  //
  // Idempotency: guarded per-checkbox via the
  // data-gosx-studio-block-visibility-island-bound dataset key (renamed
  // from the legacy gosxStudioBlockVisibilityBound to make the post-deletion
  // ownership obvious). A second call against the same checkbox returns
  // silently — listeners do not stack and the initial-sync write does not
  // re-fire.
  function bindVisibilityIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    Array.prototype.forEach.call(scope.querySelectorAll("[data-editor-block-visible]"), function (check) {
      if (check.dataset.gosxStudioBlockVisibilityIslandBound === "true") return;
      check.dataset.gosxStudioBlockVisibilityIslandBound = "true";
      check.addEventListener("change", function () {
        updateVisibilityStateIsland(check);
      });
      updateVisibilityStateIsland(check);
    });
  }

  window.__gosx_blocklayout_runtime_island_bindVisibility = bindVisibilityIsland;

  // ===== v0.5.2 follow-on: bindList + bindHandleDrag islands =====
  //
  // The deleted studio-engines.js bundle (removed 2026-05-27) installed two
  // further listener-binding helpers — bindBlockLayoutList (the per-list
  // click delegate for move buttons + row click-to-select, plus initial
  // boot-time renumber and rows-non-draggable flag) and bindBlockLayoutHandleDrag
  // (the per-list pointer-event drag-and-drop reorder chain for
  // [data-block-studio-handle] grips). The slice-4 island migration owned
  // only the nine per-call methods on GoSXStudioBlockLayoutRuntime; the
  // v0.5.1 follow-on covered bindLibrary + bindVisibility. v0.5.2 closes
  // the burn-down by migrating these last two.
  //
  // After v0.5.2 the block-layout-runtime listener-binding surface is fully
  // on islands. The only legacy bind* helpers remaining in the deleted
  // bundle snapshot (32dac98:assets/studio-engines.js) belong to other
  // contracts that already have their own island packages
  // (workbenchruntime / fieldruntime / brandruntime / styleruntime /
  // selectionruntime).
  //
  // See:
  //   - inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
  //   - the legacy bindings at
  //     git -C ~/work/gosx-studio show 32dac98:assets/studio-engines.js
  //     lines 1942-1958 (bindBlockLayoutList) and 2112-2141 (bindBlockLayoutHandleDrag).

  // listForRowIsland walks from a [data-block-studio-block] row to its
  // nearest list ancestor — the parent element that contains the row
  // siblings the delegate should swap with. The legacy bindBlockLayoutList
  // was called from the engine-mount factory with the list element directly,
  // so it didn't need this helper; v0.5.2 instead walks the document for
  // every row and binds each row's parent list once, idempotently. This
  // matches how the BridgeShim's auto-mount passes `document` as scope.
  function listForRowIsland(row) {
    return row && row.parentElement ? row.parentElement : null;
  }

  // bindListIsland(root) — mirrors the legacy bindBlockLayoutList at
  // studio-engines.js:1942. For every block-row list (the parent of every
  // [data-block-studio-block] row), attaches a single click delegate that
  // dispatches to:
  //   - moveRow(list, row, "up"|"down") when the click hits a
  //     [data-block-studio-move] button inside the list.
  //   - selectRow(list, rowKey(row)) when the click hits a
  //     [data-block-studio-block] row inside the list.
  // Also sets each row's .draggable = false (so the HTML5 native drag
  // attribute does not interfere with the pointer-event drag handler that
  // bindHandleDrag attaches) and calls renumber(list, "engine-init") once
  // so [data-block-studio-order] inputs, [data-block-studio-index] attrs,
  // and the move-button disabled states are correct on first paint —
  // matching the legacy boot-time renumberBlockLayoutList(list, "engine-init")
  // call at the tail of bindBlockLayoutList.
  //
  // Idempotency: guarded per-list via the
  // data-gosx-studio-block-list-island-bound dataset key (renamed from the
  // legacy gosxStudioBlockLayoutBound to make the post-deletion ownership
  // obvious). A second call against the same list returns silently —
  // listeners do not stack and the initial renumber does not re-fire.
  function bindListIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    var seen = [];
    Array.prototype.forEach.call(scope.querySelectorAll("[data-block-studio-block]"), function (row) {
      var list = listForRowIsland(row);
      if (!list) return;
      // De-dup lists within a single bindList call (one bind per parent).
      for (var i = 0; i < seen.length; i++) {
        if (seen[i] === list) return;
      }
      seen.push(list);
      if (list.dataset.gosxStudioBlockListIslandBound === "true") return;
      list.dataset.gosxStudioBlockListIslandBound = "true";
      // Disable the HTML5 native drag attribute on every row so the
      // pointer-event drag handler (bindHandleDrag) owns drag interactions
      // unambiguously. The legacy bundle did this at bind time too.
      Array.prototype.forEach.call(list.querySelectorAll("[data-block-studio-block]"), function (r) {
        r.draggable = false;
      });
      list.addEventListener("click", function (event) {
        var move = event.target && event.target.closest
          ? event.target.closest("[data-block-studio-move]")
          : null;
        if (move && list.contains(move)) {
          event.preventDefault();
          var moveRow = move.closest("[data-block-studio-block]");
          moveRowIsland(list, moveRow, move.getAttribute("data-block-studio-move"));
          return;
        }
        var clickedRow = event.target && event.target.closest
          ? event.target.closest("[data-block-studio-block]")
          : null;
        if (clickedRow && list.contains(clickedRow)) {
          selectRowIsland(list, rowKeyIsland(clickedRow));
        }
      });
      // Initial renumber pass — preserves the legacy boot-time call
      // renumberBlockLayoutList(list, "engine-init"). Source string is
      // surfaced on the blockstudio:reorder CustomEvent so consumers can
      // distinguish init from engine-buttons / engine-preview / engine-list.
      renumberIsland(list, "engine-init");
    });
  }

  window.__gosx_blocklayout_runtime_island_bindList = bindListIsland;

  // closestRowByYIsland mirrors the legacy closestBlockLayoutRowByY at
  // studio-engines.js:2143. Used by the pointermove handler to locate the
  // nearest non-dragging row by midpoint Y so the dragged row can
  // insertBefore-swap above or below it. Pure read-only DOM query; returns
  // null when no candidate row exists.
  function closestRowByYIsland(list, y, dragging) {
    var best = null;
    var bestDistance = Number.POSITIVE_INFINITY;
    Array.prototype.forEach.call(list.querySelectorAll("[data-block-studio-block]"), function (row) {
      if (row === dragging) return;
      var rect = row.getBoundingClientRect();
      var distance = Math.abs(y - (rect.top + rect.height / 2));
      if (distance < bestDistance) {
        best = row;
        bestDistance = distance;
      }
    });
    return best;
  }

  // bindHandleDragIsland(root) — mirrors the legacy bindBlockLayoutHandleDrag
  // at studio-engines.js:2112. For every block-row list (the parent of every
  // [data-block-studio-handle] grip), attaches the pointer-event drag chain:
  //
  //   - pointerdown on [data-block-studio-handle]: capture the row + key +
  //     clientY into a per-list drag state; mark the row .is-dragging;
  //     setPointerCapture on the handle so subsequent pointermove /
  //     pointerup events stream to the handle even when the cursor leaves it.
  //   - pointermove (any target): locate the nearest other row by midpoint
  //     Y; insertBefore-swap the dragged row above or below it based on
  //     whether the cursor is in the top or bottom half of that row.
  //   - pointerup / pointercancel: remove .is-dragging; call
  //     renumber(list, "engine-list") so [data-block-studio-order] /
  //     [data-block-studio-index] reflect the final ordering and the
  //     blockstudio:reorder event fires; selectRow(list, drag.key) so the
  //     dropped row is the active selection.
  //
  // IMPORTANT: the legacy implementation uses pointer events (not HTML5
  // native drag) and the handle selector is [data-block-studio-handle]
  // (not [data-block-studio-drag-handle]). This island preserves both
  // choices exactly so behavior parity holds against the deleted legacy
  // snapshot.
  //
  // Idempotency: guarded per-list via the
  // data-gosx-studio-block-handle-drag-island-bound dataset key. A second
  // call against the same list returns silently — listeners do not stack
  // and the per-list drag state is preserved.
  function bindHandleDragIsland(root) {
    var scope = root && root.querySelectorAll ? root : doc;
    var seen = [];
    Array.prototype.forEach.call(scope.querySelectorAll("[data-block-studio-handle]"), function (handle) {
      var row = handle.closest ? handle.closest("[data-block-studio-block]") : null;
      var list = row ? listForRowIsland(row) : null;
      if (!list) return;
      // De-dup lists within a single bindHandleDrag call (one bind per parent).
      for (var i = 0; i < seen.length; i++) {
        if (seen[i] === list) return;
      }
      seen.push(list);
      if (list.dataset.gosxStudioBlockHandleDragIslandBound === "true") return;
      list.dataset.gosxStudioBlockHandleDragIslandBound = "true";
      var drag = null;
      list.addEventListener("pointerdown", function (event) {
        var pointerHandle = event.target && event.target.closest
          ? event.target.closest("[data-block-studio-handle]")
          : null;
        if (!pointerHandle || !list.contains(pointerHandle)) return;
        var pointerRow = pointerHandle.closest("[data-block-studio-block]");
        if (!pointerRow) return;
        event.preventDefault();
        drag = { row: pointerRow, key: rowKeyIsland(pointerRow), y: event.clientY };
        pointerRow.classList.add("is-dragging");
        if (pointerHandle.setPointerCapture) {
          try { pointerHandle.setPointerCapture(event.pointerId); } catch (e) { /* ignore */ }
        }
      });
      list.addEventListener("pointermove", function (event) {
        if (!drag) return;
        event.preventDefault();
        var target = closestRowByYIsland(list, event.clientY, drag.row);
        if (!target) return;
        var rect = target.getBoundingClientRect();
        list.insertBefore(drag.row, event.clientY > rect.top + rect.height / 2 ? target.nextElementSibling : target);
      });
      function finish() {
        if (!drag) return;
        drag.row.classList.remove("is-dragging");
        // Reorder source "engine-list" preserves the legacy
        // renumberBlockLayoutList(list, "engine-list") call so consumers
        // subscribing to blockstudio:reorder can distinguish drag-drop
        // reorders from button-move / preview-commit reorders.
        renumberIsland(list, "engine-list");
        selectRowIsland(list, drag.key);
        drag = null;
      }
      list.addEventListener("pointerup", finish);
      list.addEventListener("pointercancel", finish);
    });
  }

  window.__gosx_blocklayout_runtime_island_bindHandleDrag = bindHandleDragIsland;
})();
