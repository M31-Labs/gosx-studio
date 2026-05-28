// visibility_bind.gsx — GoSXStudioBlockLayoutRuntime.bindVisibility island.
//
// v0.5.1 follow-up to the v0.5.0 bridgemount restoration. Restores the
// per-checkbox change handler that the deleted bindBlockLayoutVisibility in
// studio-engines.js (removed 2026-05-27) used to attach to every
// [data-editor-block-visible] checkbox: on change, the helper calls
// updateBlockLayoutVisibilityState(checkbox) (now
// window.GoSXStudioBlockLayoutRuntime.updateVisibilityState(checkbox))
// to toggle the row's hidden class, status text, pill className, and
// publish the $preview.block.<key>.visible shared signal that the
// previewruntime subscriber observes.
//
// The slice-4 island runtime exports updateVisibilityState as a per-call
// method but never owned the change-listener attachment. The legacy
// bindBlockLayoutVisibility lived in studio-engines.js and was deleted
// 2026-05-27; checkboxes can be ticked but nothing happens until this
// island wires them.
//
// The legacy implementation also called updateBlockLayoutVisibilityState(check)
// once at bind time to sync the initial state (which then dispatched the
// $preview write through the legacy PreviewRuntime). The v0.5.1 island
// preserves this initial-sync behavior so the storefront preview's visible
// state matches the editor checkbox state on first paint without requiring
// the user to toggle each checkbox manually.
//
// See:
//   - elm's spore at ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
//     (Open question 2 — block layout visibility checkbox change handlers).
//   - the slice plan at ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
//   - the deletion log at ~/.hyphae/spaces/m31labs-gosx/lessons/phase-3-slice-4-blocklayoutruntime-deletion-log.md
//     ("Idempotency" table, gosxStudioBlockVisibilityBound row).
//
// The runtime bundle's island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_bindVisibility
// that the v0.5.1 BridgeShim delegates to. The island is idempotent
// per-checkbox via the data-gosx-studio-block-visibility-island-bound="true"
// marker on each bound checkbox (consistent with the legacy
// gosxStudioBlockVisibilityBound dataset key the deleted code used).

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockVisibilityBindProps lets the consumer pass an opaque selector
// identifying the editor root the bindVisibility island should scope its
// per-checkbox attachment to. Defaults to the editor document when unset,
// matching the legacy bindBlockLayoutVisibility(document) call site.
type BlockVisibilityBindProps struct {
	RootSelector string
}

//gosx:island
func BlockVisibilityBind(props BlockVisibilityBindProps) gosx.Node {
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
		data-gosx-studio-block-visibility-bind-island="true"
		data-gosx-studio-block-visibility-bind-root={root}
		data-gosx-studio-block-visibility-bind-mounted={mounted.Get()}
		hidden
	></div>
}
