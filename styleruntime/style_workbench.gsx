// style_workbench.gsx — GoSXStudioStyleRuntime.bindWorkbench island.
//
// Phase 3 slice-5 island implementation of the JS function bindStyleWorkbench in
// the legacy bundle (removed 2026-05-27). Given an editor workbench root,
// the helper:
//   1. Resolves the [data-editor-workbench] form (or the root itself when it
//      matches).
//   2. Guards re-entry via [data-gosx-studio-style-workbench-bound].
//   3. Binds click listeners that dispatch setStyleControlValue /
//      resetStyleControlValue when style-control or reset buttons are
//      activated.
//   4. Binds pointerover / pointerout / focusin / focusout listeners that
//      preview showStyleImpact on hover / focus and restoreStyleImpact on
//      leave / blur.
//   5. Binds input / change listeners that re-sync style buttons + impact
//      panel when theme/template/custom-class controls change.
//   6. Registers a bindStudioFrameLoad("data-editor-style-impact-frame-bound")
//      so the impact restores on iframe reload.
//   7. Runs syncStyleControlButtons(form) + restoreStyleImpact() on bind.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-5-styleruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// bindWorkbench operates on editor-side DOM only — the cross-frame impact
// preview is delegated to the showImpact / restoreImpact helpers (which are
// the three iframe-crossing methods covered by style_impact.gsx). The
// runtime bundle's island_runtime.js publishes:
//   window.__gosx_style_runtime_island_bindWorkbench
// that the slice-5 BridgeShim delegates to directly. Post 2026-05-27 the
// legacy JS bundle is gone; the island path is the only path. See Phase 3
// burn-down.

package styleruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// StyleWorkbenchProps lets the consumer pass an opaque selector identifying
// the editor-workbench root the bindWorkbench island should scope to.
// Defaults to the editor document when unset.
type StyleWorkbenchProps struct {
	RootSelector string
}

//gosx:island
func StyleWorkbench(props StyleWorkbenchProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a stable
	// mount event in the compiled output. It also marks the island as
	// having been hydrated on the client, which the parity tests assert in
	// candidate mode (see parity_e2e/styleruntime_test.ts).
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-style-workbench-island="true"
		data-gosx-studio-style-workbench-root={root}
		data-gosx-studio-style-workbench-mounted={mounted.Get()}
		hidden
	></div>
}
