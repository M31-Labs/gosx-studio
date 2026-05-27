// block_select.gsx — GoSXStudioBlockLayoutRuntime.selectRow island.
//
// Phase 3 slice-4 island implementation of the JS function selectBlockLayoutRow in
// the legacy bundle (removed 2026-05-27). It toggles the .is-selected
// class on every block-row to match the supplied key, scrollIntoView's the
// newly-selected row, and dispatches a blockstudio:select CustomEvent on
// document so other engines (selection commandbar, inspector form) can
// react.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// selectRow operates on editor-side DOM only — no iframe crossing. The
// blockstudio:select event fans out within the editor frame; the preview
// iframe is unaffected. The runtime bundle's island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_selectRow
// that the slice-4 BridgeShim delegates to when the
// "block-layout-runtime-islands" feature flag is on.
//
// During the additive shipping window (Section G of the slice plan), the
// BridgeShim only invokes this island when ShellConfig.FeatureFlags has
// Post 2026-05-27 the legacy JS bundle is gone; the island path is the
// only path. See Phase 3 burn-down.
// GoSXStudioSelectionRuntime.bind also calls into selectRow as part of its
// block-selection sub-behavior; the parity matrix's selection row
// (~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md)
// already documents this cross-runtime relationship.

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockSelectProps lets the consumer pass an opaque selector identifying
// the root element to scan for [data-block-studio-block] rows. Defaults to
// the editor document when unset, matching the legacy selectBlockLayoutRow
// call sites (most of which pass a containing list or form element).
type BlockSelectProps struct {
	RootSelector string
}

//gosx:island
func BlockSelect(props BlockSelectProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a stable
	// mount event in the compiled output. It also marks the island as
	// having been hydrated on the client, which the parity tests assert in
	// candidate mode (see parity_e2e/blocklayoutruntime_test.ts).
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-block-select-island="true"
		data-gosx-studio-block-select-root={root}
		data-gosx-studio-block-select-mounted={mounted.Get()}
		hidden
	></div>
}
