// workbench_style_state.gsx — GoSXStudioWorkbenchRuntime.setStyleState
// island.
//
// Phase 3 slice-7 burn-down of one style-state JS function in
// gosx-studio/assets/studio-engines.js:
//
//   - setWorkbenchStyleState(form, state) — line 405; sets
//                                            data-studio-style-state on the
//                                            form, toggles aria-pressed on
//                                            [data-studio-style-state]
//                                            buttons, writes
//                                            data-style-state /
//                                            data-style-breakpoint /
//                                            data-style-valid on
//                                            [data-studio-style-scope]
//                                            wrappers, emits
//                                            gosxstudio:workbench-style-state-
//                                            change, refreshes the canvas.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// This method is editor-chrome — no iframe crossing required. Note that
// data-style-breakpoint depends on currentWorkbenchBreakpoint, which slice
// 7 implements as a signal-derivation from data-studio-breakpoint on the
// form (see the slice plan and workbench_viewport.gsx).

package workbenchruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// WorkbenchStyleStateProps lets the consumer pass an opaque selector
// identifying the editor-side root the style-state island should scope to.
// Defaults to the editor document when unset.
type WorkbenchStyleStateProps struct {
	RootSelector string
}

//gosx:island
func WorkbenchStyleState(props WorkbenchStyleStateProps) gosx.Node {
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-workbench-style-state-island="true"
		data-gosx-studio-workbench-style-state-root={root}
		data-gosx-studio-workbench-style-state-mounted={mounted.Get()}
		hidden
	></div>
}
