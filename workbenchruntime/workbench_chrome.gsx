// workbench_chrome.gsx — GoSXStudioWorkbenchRuntime.{bindChrome,setMode,saveLayout} island.
//
// Phase 3 slice-7 burn-down of three workbench-chrome JS functions in
// gosx-studio/assets/studio-engines.js:
//
//   - bindWorkbenchChrome(root) — line 516; binds the workbench form's
//                                  delegated click handler (fans out to
//                                  setMode / syncViewport / syncZoom /
//                                  setStyleState / toggleRail / toggleFocus /
//                                  toggleActivity), wires saveLayoutSoon to
//                                  rail-width-change/commit events,
//                                  applies the persisted layout, and seeds
//                                  the workbench mode / viewport / zoom on
//                                  initial bind.
//   - setWorkbenchMode(form, mode, scroll) — line 343; writes
//                                  data-studio-mode on the form, toggles
//                                  panel hidden / aria-hidden / classes,
//                                  scrolls active panel into view, updates
//                                  [data-studio-mode-label] readouts, emits
//                                  gosxstudio:workbench-mode-change.
//   - saveWorkbenchLayout(form) — line 294; writes the form's
//                                  --studio-left-width / --studio-right-width
//                                  / data-studio-activity-state to
//                                  localStorage under
//                                  gosx-studio-editor-layout.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_workbench_runtime_island_bindChrome /
// window.__gosx_workbench_runtime_island_setMode /
// window.__gosx_workbench_runtime_island_saveLayout that the slice-7
// BridgeShim delegates to when the "workbench-runtime-islands" feature flag
// is on; otherwise the legacy bindWorkbenchChrome / setWorkbenchMode /
// saveWorkbenchLayout functions in studio-engines.js run.
//
// All three methods are editor-chrome — no iframe crossing required (this
// slice is independent of the slice 6 cross-frame-transport block).

package workbenchruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// WorkbenchChromeProps lets the consumer pass an opaque selector identifying
// the editor-side root the bindChrome island should scope to. Defaults to
// the editor document when unset.
type WorkbenchChromeProps struct {
	RootSelector string
}

//gosx:island
func WorkbenchChrome(props WorkbenchChromeProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a stable
	// mount event in the compiled output. It also marks the island as
	// having been hydrated on the client, which the parity tests assert in
	// candidate mode (see parity_e2e/workbenchruntime_test.ts).
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-workbench-chrome-island="true"
		data-gosx-studio-workbench-chrome-root={root}
		data-gosx-studio-workbench-chrome-mounted={mounted.Get()}
		hidden
	></div>
}
