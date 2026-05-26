// block_rows.gsx — GoSXStudioBlockLayoutRuntime.{rows,rowKey,rowForKey} island.
//
// Phase 3 slice-4 burn-down of the three pure-helper JS functions in
// gosx-studio/assets/studio-engines.js:
//
//   - blockRows(list)        — line 1960; Array.from(list.querySelectorAll("[data-block-studio-block]")).
//   - blockRowKey(row)       — line 1964; row.getAttribute("data-block-studio-block") || "".
//   - blockRowForKey(root, k) — line 1968; root.querySelector('[data-block-studio-block="<k>"]').
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// These three are read-only DOM queries with no side effects, so a single
// mount-point marker covers all three islands. The runtime bundle's
// island_runtime.js publishes:
//   window.__gosx_blocklayout_runtime_island_rows
//   window.__gosx_blocklayout_runtime_island_rowKey
//   window.__gosx_blocklayout_runtime_island_rowForKey
// that the slice-4 BridgeShim delegates to when the
// "block-layout-runtime-islands" feature flag is on; otherwise the legacy
// blockRows / blockRowKey / blockRowForKey functions in studio-engines.js
// run.
//
// The block-list DOM contract these helpers query against
// (data-block-studio-block) is emitted by muddy-noni-commerce's
// editorHomeLayerRowAttrs in app/admin/editor/page.server.go:1407 —
// verified intact for slice 4.

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockRowsProps lets the consumer pass an opaque selector identifying the
// block-list root the rows / rowKey / rowForKey helpers should scope to.
// Defaults to the editor document when unset, matching the legacy call sites
// (most of which pass document or a containing form / list element).
type BlockRowsProps struct {
	RootSelector string
}

//gosx:island
func BlockRows(props BlockRowsProps) gosx.Node {
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
		data-gosx-studio-block-rows-island="true"
		data-gosx-studio-block-rows-root={root}
		data-gosx-studio-block-rows-mounted={mounted.Get()}
		hidden
	></div>
}
