// workbench_toggles.gsx — GoSXStudioWorkbenchRuntime.{toggleFocus,toggleActivity}
// island.
//
// Phase 3 slice-7 island implementation of two toggle-related JS functions in
// the legacy bundle (removed 2026-05-27) (toggleWorkbenchRail is covered by
// workbench_rails.gsx since it's intrinsically tied to the rail-state model):
//
//   - toggleWorkbenchFocus(form) — line 489; flips data-studio-focus
//                                   between "true" and "false", re-syncs
//                                   the rail toggle aria-pressed states,
//                                   emits gosxstudio:workbench-focus-change,
//                                   refreshes the canvas.
//   - toggleWorkbenchActivity(form) — line 476 (via setWorkbenchActivity
//                                   at line 467); flips
//                                   data-studio-activity-state between
//                                   "open" and "collapsed", re-syncs
//                                   activity-toggle buttons (Show/Hide text
//                                   + aria-pressed), persists via
//                                   saveLayout, emits
//                                   gosxstudio:workbench-activity-change,
//                                   refreshes the canvas.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// Both methods are editor-chrome — no iframe crossing required.

package workbenchruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// WorkbenchTogglesProps lets the consumer pass an opaque selector
// identifying the editor-side root the toggles island should scope to.
// Defaults to the editor document when unset.
type WorkbenchTogglesProps struct {
	RootSelector string
}

//gosx:island
func WorkbenchToggles(props WorkbenchTogglesProps) gosx.Node {
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-workbench-toggles-island="true"
		data-gosx-studio-workbench-toggles-root={root}
		data-gosx-studio-workbench-toggles-mounted={mounted.Get()}
		hidden
	></div>
}
