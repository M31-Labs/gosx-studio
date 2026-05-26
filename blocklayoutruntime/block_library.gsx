// block_library.gsx — GoSXStudioBlockLayoutRuntime.updateBlockLibraryState island.
//
// Phase 3 slice-4 burn-down of the JS function updateBlockLayoutLibraryState
// in gosx-studio/assets/studio-engines.js:1972. For every
// [data-editor-add-block] button in the supplied root, the helper resolves
// the corresponding block-row in document via blockRowForKey, reads the
// row's [data-editor-block-visible] checkbox state, and writes the button's:
//   - className     (toggling button--ghost is-active vs button--secondary).
//   - aria-pressed  ("true" when the block is on the page, "false" when not).
//   - <small> label ("On page" vs "Add").
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// updateBlockLibraryState operates on editor-side DOM only — no iframe
// crossing. It runs every time updateVisibilityState changes a row's
// visibility (so library buttons stay in sync) and every time the library
// bindBlockLayoutLibrary click handler activates a block.
//
// The runtime bundle's island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_updateBlockLibraryState
// that the slice-4 BridgeShim delegates to when the
// "block-layout-runtime-islands" feature flag is on. During the additive
// shipping window (Section G of the slice plan), the BridgeShim only
// invokes this island when ShellConfig.FeatureFlags has
// "block-layout-runtime-islands"=true; otherwise the legacy
// updateBlockLayoutLibraryState function in studio-engines.js runs.

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockLibraryProps lets the consumer pass an opaque selector identifying
// the root element to scan for [data-editor-add-block] buttons. Defaults to
// the editor document when unset, matching the legacy
// updateBlockLayoutLibraryState(document) call sites.
type BlockLibraryProps struct {
	RootSelector string
}

//gosx:island
func BlockLibrary(props BlockLibraryProps) gosx.Node {
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
		data-gosx-studio-block-library-island="true"
		data-gosx-studio-block-library-root={root}
		data-gosx-studio-block-library-mounted={mounted.Get()}
		hidden
	></div>
}
