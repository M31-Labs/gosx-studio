// workbench_rails.gsx — GoSXStudioWorkbenchRuntime.{bindRailResizers,
// currentRailWidth,setRailWidth,toggleRail} island.
//
// Phase 3 slice-7 burn-down of four rail-related JS functions in
// gosx-studio/assets/studio-engines.js:
//
//   - bindWorkbenchRailResizers(root) — line 194; binds the
//                                       [data-studio-resizer] pointer drag
//                                       + ArrowLeft/ArrowRight keyboard
//                                       nudges on the left/right rail
//                                       handles.
//   - currentWorkbenchRailWidth(form, side, handle) — line 165; pure read
//                                       returning the form's
//                                       --studio-{left,right}-width custom
//                                       property as an integer (or the
//                                       sidebar's bounding-rect fallback,
//                                       or the handle's bounds fallback).
//                                       Slice 7 implements this as a
//                                       derivation from form state.
//   - setWorkbenchRailWidth(form, side, width, handle, committed) — line 185;
//                                       clamps width to bounds, writes
//                                       --studio-{left,right}-width on the
//                                       form, updates the handle's
//                                       aria-valuenow, emits
//                                       gosxstudio:rail-width-change or
//                                       gosxstudio:rail-width-commit.
//   - toggleWorkbenchRail(form, side) — line 480; flips data-studio-{left,
//                                       right} between "open" and
//                                       "collapsed", sets data-studio-focus
//                                       to "false", re-syncs the rail
//                                       toggle buttons, emits
//                                       gosxstudio:workbench-rail-change.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// All four methods are editor-chrome — no iframe crossing required.

package workbenchruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// WorkbenchRailsProps lets the consumer pass an opaque selector identifying
// the editor-side root the rail-resizer island should scope to. Defaults
// to the editor document when unset.
type WorkbenchRailsProps struct {
	RootSelector string
}

//gosx:island
func WorkbenchRails(props WorkbenchRailsProps) gosx.Node {
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-workbench-rails-island="true"
		data-gosx-studio-workbench-rails-root={root}
		data-gosx-studio-workbench-rails-mounted={mounted.Get()}
		hidden
	></div>
}
