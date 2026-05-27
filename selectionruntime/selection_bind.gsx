// selection_bind.gsx — GoSXStudioSelectionRuntime.bind island.
//
// Phase 3 slice-2 island implementation of the JS function bindSelectionSurface in
// the legacy bundle (removed 2026-05-27). See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-2-selectionruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_selection_runtime_island_bind that scans the DOM and wires
// up the five sub-behaviors the legacy bindSelectionSurface covers:
//
//   1. Block selection         — data-block-studio-block click/focus.
//   2. Workspace target select — data-studio-panel-link, data-studio-site-page.
//   3. Field focus             — data-studio-field-source, studio:field-select.
//   4. Selection commandbar    — data-studio-selection-action: reveal /
//                                content / style / inline-text /
//                                previous-field / next-field /
//                                toggle-visibility.
//   5. Style-scope readouts    — data-studio-style-scope-*,
//                                data-studio-style-state,
//                                data-studio-style-breakpoint.
//
// All five operate on editor-side DOM only — no iframe crossing required
// for selection state. The island writes plain Bridge signals
// ($selection.block, $selection.workspaceTarget, $selection.fieldFocus,
// $selection.styleScope) for the island's own bookkeeping; the legacy
// applyTextUpdate / requestInlineEdit / cycleField fan-outs into the
// preview iframe still go through GoSXStudioPreviewRuntime (slice 6).
//
// During the additive shipping window (Section B of the slice plan), the
// BridgeShim only invokes this island when ShellConfig.FeatureFlags has
// Post 2026-05-27 the legacy JS bundle is gone; the island path is the
// only path. See Phase 3 burn-down.

package selectionruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// SelectionBindProps lets the consumer pass an opaque selector identifying
// the workbench-form root the selection runtime should bind under. Defaults
// to the editor workbench form when unset, matching the legacy
// bindSelectionSurface(document) call site (which resolves the form via
// editorWorkbench(document)).
type SelectionBindProps struct {
	RootSelector string
}

//gosx:island
func SelectionBind(props SelectionBindProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a stable
	// mount event in the compiled output. It also marks the island as
	// having been hydrated on the client, which the parity tests assert in
	// candidate mode (see parity_e2e/selectionruntime_test.ts).
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-selection-bind-island="true"
		data-gosx-studio-selection-bind-root={root}
		data-gosx-studio-selection-bind-mounted={mounted.Get()}
		hidden
	></div>
}
