// library_bind.gsx — GoSXStudioBlockLayoutRuntime.bindLibrary island.
//
// v0.5.1 follow-up to the v0.5.0 bridgemount restoration. Restores the
// per-form click delegation that the deleted bindBlockLayoutLibrary in
// studio-engines.js (removed 2026-05-27) used to attach to the workbench
// form: every click on a [data-editor-add-block] button activates the block
// it names by flipping the row's [data-editor-block-visible] checkbox to
// checked, dispatching input + change events (so the visibility island and
// any other change listeners fire), selecting the row, scrolling it into
// view, focusing it, and refreshing the library button states.
//
// The slice-4 island runtime never owned this binding — only the per-call
// methods on GoSXStudioBlockLayoutRuntime (rows / rowKey / rowForKey /
// moveRow / renumber / selectRow / commitReorder / updateBlockLibraryState /
// updateVisibilityState). The legacy bindBlockLayoutLibrary lived in
// studio-engines.js and called those methods on click. When the legacy
// bundle was deleted, the delegation went with it; add-block buttons render
// in the correct visual state (updateBlockLibraryState runs at boot via
// v0.5.0 auto-mount) but clicks are inert until this island wires them.
//
// See:
//   - elm's spore at ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
//     (Open question 1 — block layout library click handlers).
//   - the slice plan at ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
//   - the deletion log at ~/.hyphae/spaces/m31labs-gosx/lessons/phase-3-slice-4-blocklayoutruntime-deletion-log.md
//     ("Idempotency" table, gosxStudioBlockLibraryBound row).
//
// The runtime bundle's island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_bindLibrary
// that the v0.5.1 BridgeShim delegates to. The island is idempotent via the
// data-gosx-studio-block-library-island-bound="true" marker on the bound
// workbench form (consistent with the other island bound-attrs across this
// package).

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockLibraryBindProps lets the consumer pass an opaque selector identifying
// the editor workbench root the bindLibrary island should attach the click
// delegate to. Defaults to the editor document when unset; the island
// internally resolves the workbench form via [data-editor-workbench].
type BlockLibraryBindProps struct {
	RootSelector string
}

//gosx:island
func BlockLibraryBind(props BlockLibraryBindProps) gosx.Node {
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
		data-gosx-studio-block-library-bind-island="true"
		data-gosx-studio-block-library-bind-root={root}
		data-gosx-studio-block-library-bind-mounted={mounted.Get()}
		hidden
	></div>
}
