// handle_drag_bind.gsx — GoSXStudioBlockLayoutRuntime.bindHandleDrag island.
//
// v0.5.2 follow-up to the v0.5.1 bindLibrary/bindVisibility migration.
// Restores the per-list pointer-event drag-and-drop chain that the deleted
// bindBlockLayoutHandleDrag in studio-engines.js (removed 2026-05-27) used
// to attach to the block-row list element so each row's
// [data-block-studio-handle] grip lets the user reorder rows by dragging.
//
// IMPORTANT: the legacy implementation uses pointer events (pointerdown /
// pointermove / pointerup / pointercancel) — NOT HTML5 native drag events
// (dragstart / dragover / drop / dragend) — and the handle selector is
// [data-block-studio-handle] (NOT [data-block-studio-drag-handle]). The
// v0.5.2 implementation preserves the legacy event model exactly so the
// post-deletion observable behavior is identical to the pre-deletion
// behavior. The pointer-event model has two advantages over HTML5 drag:
//   1. It works on touch + stylus inputs without separate touchstart code.
//   2. setPointerCapture keeps events flowing to the grip even when the
//      cursor leaves it during a drag, eliminating the dragleave race.
//
// On pointerdown the handler captures the row + key + clientY into a drag
// state; on pointermove it locates the nearest other row by midpoint Y and
// insertBefore-swaps the dragged row above or below it; on pointerup /
// pointercancel it removes the .is-dragging class, calls renumber(list,
// "engine-list"), and reselects the dragged row.
//
// See:
//   - elm's spore at ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
//   - the slice plan at ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
//   - the deletion log at ~/.hyphae/spaces/m31labs-gosx/lessons/phase-3-slice-4-blocklayoutruntime-deletion-log.md
//
// The runtime bundle's island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_bindHandleDrag
// that the v0.5.2 BridgeShim delegates to. The island is idempotent per-list
// via the data-gosx-studio-block-handle-drag-island-bound="true" marker on
// the bound list root (consistent with the other island bound-attrs across
// this package — sibling marker to gosxStudioBlockListIslandBound /
// gosxStudioBlockLibraryIslandBound / gosxStudioBlockVisibilityIslandBound).

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockHandleDragBindProps lets the consumer pass an opaque selector
// identifying the block-row list element the bindHandleDrag island should
// attach the pointer-event chain to. Defaults to the editor document when
// unset; the island internally walks every [data-block-studio-handle]'s
// nearest list ancestor.
type BlockHandleDragBindProps struct {
	RootSelector string
}

//gosx:island
func BlockHandleDragBind(props BlockHandleDragBindProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a stable
	// mount event in the compiled output. It also marks the island as having
	// been hydrated on the client, which the parity tests assert in candidate
	// mode (see parity_e2e/blocklayoutruntime_test.ts).
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-block-handle-drag-bind-island="true"
		data-gosx-studio-block-handle-drag-bind-root={root}
		data-gosx-studio-block-handle-drag-bind-mounted={mounted.Get()}
		hidden
	></div>
}
