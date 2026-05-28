// list_bind.gsx — GoSXStudioBlockLayoutRuntime.bindList island.
//
// v0.5.2 follow-up to the v0.5.1 bindLibrary/bindVisibility migration.
// Restores the per-form click delegation that the deleted bindBlockLayoutList
// in studio-engines.js (removed 2026-05-27) used to attach to the block-row
// list element:
//
//   1. Click delegate on [data-block-studio-move] (up/down move buttons) →
//      calls moveRow(list, row, "up"|"down") which insertBefores the row
//      with its sibling, then renumbers + reselects.
//   2. Click delegate on [data-block-studio-block] rows → row click-to-select
//      via selectRow(list, key). Distinct from the v0.5.1 bindLibrary
//      delegate which scopes to the workbench form for add-block buttons —
//      this one scopes to the list itself and handles row clicks.
//   3. Initial draggable=false on every row so the legacy HTML5 native drag
//      attribute doesn't conflict with the pointer-event drag handler.
//   4. Initial renumberBlockLayoutList(list, "engine-init") at mount time to
//      reset order inputs / button disabled states / fire the blockstudio:
//      reorder event on first paint (matches legacy boot-time renumber pass).
//   5. Calls bindHandleDrag(list) so each row's [data-block-studio-handle]
//      grip wires up the pointer-event drag-and-drop reorder chain.
//
// The slice-4 island runtime (Phase 3 slice 4, shipped 2026-05-26) owned the
// nine per-call methods on GoSXStudioBlockLayoutRuntime but not the
// listener-binding entry points. v0.5.0 restored auto-mount;
// v0.5.1 (hazel) restored bindLibrary + bindVisibility; v0.5.2 (this slice)
// restores bindList + bindHandleDrag — the last two unmigrated handlers.
// After this slice the block-layout-runtime listener-binding surface is
// fully on islands.
//
// See:
//   - elm's spore at ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
//   - the slice plan at ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
//   - the deletion log at ~/.hyphae/spaces/m31labs-gosx/lessons/phase-3-slice-4-blocklayoutruntime-deletion-log.md
//     ("Idempotency" table, gosxStudioBlockLayoutBound row — the dataset
//     key that guarded the legacy bindBlockLayoutList).
//
// The runtime bundle's island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_bindList
// that the v0.5.2 BridgeShim delegates to. The island is idempotent per-list
// via the data-gosx-studio-block-list-island-bound="true" marker on the
// bound list root (consistent with the other island bound-attrs across this
// package — sibling marker to gosxStudioBlockLibraryIslandBound /
// gosxStudioBlockVisibilityIslandBound).

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockListBindProps lets the consumer pass an opaque selector identifying
// the block-row list element the bindList island should attach the click
// delegate to. Defaults to the editor document when unset; the island
// internally walks every [data-block-studio-block]'s nearest list ancestor.
type BlockListBindProps struct {
	RootSelector string
}

//gosx:island
func BlockListBind(props BlockListBindProps) gosx.Node {
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
		data-gosx-studio-block-list-bind-island="true"
		data-gosx-studio-block-list-bind-root={root}
		data-gosx-studio-block-list-bind-mounted={mounted.Get()}
		hidden
	></div>
}
