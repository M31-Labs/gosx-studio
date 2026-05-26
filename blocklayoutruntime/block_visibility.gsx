// block_visibility.gsx — GoSXStudioBlockLayoutRuntime.updateVisibilityState island.
//
// Phase 3 slice-4 burn-down of the JS function updateBlockLayoutVisibilityState
// in gosx-studio/assets/studio-engines.js:1994. Given a [data-editor-block-visible]
// checkbox, the helper:
//   1. Resolves the enclosing [data-block-studio-block] row.
//   2. Toggles the row's .editor-block--hidden class against !check.checked.
//   3. Updates [data-editor-block-status] / [data-editor-block-pill] text +
//      className ("Visible" / "Hidden" + status--ready / status).
//   4. Calls window.GoSXStudioPreviewRuntime.setBlockVisibility(key, visible)
//      to mutate the storefront preview iframe — see transitional contract
//      below.
//   5. Calls updateBlockLayoutLibraryState(document) so the block-library
//      buttons stay in sync.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// # Transitional iframe contract
//
// updateVisibilityState is the only slice-4 method that crosses the iframe.
// The legacy implementation delegates the cross-frame mutation to
// window.GoSXStudioPreviewRuntime.setBlockVisibility — a thin wrapper that
// reaches into the preview document and toggles the matching
// [data-block-key="<key>"] element's hidden / data-block-visible / .is-hidden
// classes.
//
// Per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md,
// the iframe stays and cross-frame mutations route through $preview.* shared
// signals subscribed by the preview document. The long-term destination for
// this method is a $preview.block.<key>.visible signal subscribed by the
// preview-side block.gsx component. That subscriber ships in slice 6
// (GoSXStudioPreviewRuntime burn-down).
//
// Until slice 6 ships, this island keeps the same iframe contract by
// delegating to window.GoSXStudioPreviewRuntime.setBlockVisibility — exactly
// matching slice 3's transitional pattern for updateHeaderLogo (see
// brandruntime/brand_header_logo.gsx). Ownership has moved to the block-
// layout island; the mechanism is identical. Slice 6 will flip the mechanism
// to a $preview.block.<key>.visible write without revisiting block-layout
// ownership.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_blocklayout_runtime_island_updateVisibilityState that calls
// previewRuntime().setBlockVisibility(key, visible).
//
// During the additive shipping window (Section G of the slice plan), the
// BridgeShim only invokes this island when ShellConfig.FeatureFlags has
// "block-layout-runtime-islands"=true; otherwise the legacy
// updateBlockLayoutVisibilityState path in studio-engines.js runs (which
// also delegates to PreviewRuntime).

package blocklayoutruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BlockVisibilityProps lets the consumer pass an opaque selector identifying
// the editor-side root the updateVisibilityState island should scope to.
// Defaults to the editor document when unset; the actual cross-frame write
// happens via the legacy PreviewRuntime delegation (slice 6 will replace
// this with a $preview.block.<key>.visible shared signal write that doesn't
// need a DOM root).
type BlockVisibilityProps struct {
	RootSelector string
}

//gosx:island
func BlockVisibility(props BlockVisibilityProps) gosx.Node {
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
		data-gosx-studio-block-visibility-island="true"
		data-gosx-studio-block-visibility-root={root}
		data-gosx-studio-block-visibility-mounted={mounted.Get()}
		hidden
	></div>
}
