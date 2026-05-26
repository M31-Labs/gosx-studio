// block_reorder.gsx — GoSXStudioBlockLayoutRuntime.{moveRow,renumber,commitReorder} island.
//
// Phase 3 slice-4 burn-down of the three reorder JS functions in
// gosx-studio/assets/studio-engines.js:
//
//   - moveBlockLayoutRow(list, row, direction) — line 2072; DOM insertBefore +
//     renumberBlockLayoutList(list, "engine-buttons") + selectBlockLayoutRow.
//   - renumberBlockLayoutList(list, source)    — line 2047; re-indexes
//     [data-block-studio-order] inputs + data-block-studio-index attrs;
//     toggles [data-block-studio-move=up|down] disabled state via
//     updateBlockMoveButtons; dispatches blockstudio:reorder CustomEvent
//     plus input / change events.
//   - commitBlockLayoutReorder(list, key, targetKey, position) — line 2084;
//     guards no-op (key === targetKey or either missing) and returns false;
//     otherwise insertBefore + renumber(list, "engine-preview") + selectRow.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// These three operate on editor-side DOM only — they mutate the block-list
// element and dispatch native DOM events the form / engine listens for. No
// iframe crossing required; the preview-side block-visibility fan-out is
// the separate updateVisibilityState method (see block_visibility.gsx).
// The runtime bundle's island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_moveRow
//   window.__gosx_blocklayout_runtime_island_renumber
//   window.__gosx_blocklayout_runtime_island_commitReorder
// that the slice-4 BridgeShim delegates to when the
// "block-layout-runtime-islands" feature flag is on.
//
// During the additive shipping window (Section G of the slice plan), the
// BridgeShim only invokes these islands when ShellConfig.FeatureFlags has
// "block-layout-runtime-islands"=true; otherwise the legacy
// moveBlockLayoutRow / renumberBlockLayoutList / commitBlockLayoutReorder
// functions in studio-engines.js run.

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockReorderProps lets the consumer pass an opaque selector identifying
// the block-list element the reorder helpers should scope to. Defaults to
// the editor's block-list when unset.
type BlockReorderProps struct {
	RootSelector string
}

//gosx:island
func BlockReorder(props BlockReorderProps) gosx.Node {
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
		data-gosx-studio-block-reorder-island="true"
		data-gosx-studio-block-reorder-root={root}
		data-gosx-studio-block-reorder-mounted={mounted.Get()}
		hidden
	></div>
}
