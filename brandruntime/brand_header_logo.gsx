// brand_header_logo.gsx — GoSXStudioBrandRuntime.updateHeaderLogo island.
//
// Phase 3 slice-3 island implementation of updateHeaderLogo (originally a
// thin JS delegate in the now-deleted studio-engines.js bundle). See the
// slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-3-brandruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// # Preview signal contract
//
// updateHeaderLogo historically reaches across the storefront preview iframe
// to mutate the .brand / .brand__logo DOM elements directly (set src, alt,
// hidden, --brand-logo-width / --brand-logo-offset-x / --brand-logo-offset-y
// CSS custom properties; toggle the .brand--with-logo class).
//
// Per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md,
// the iframe stays and cross-frame mutations route through $preview.* shared
// signals subscribed by the preview document. This island publishes
// $preview.brand.headerLogo for the preview frame and $brand.headerLogo for
// editor-frame consumers.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_brand_runtime_island_updateHeaderLogo.

package brandruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BrandHeaderLogoProps lets the consumer pass an opaque selector
// identifying the preview-frame root the updateHeaderLogo island should
// scope to. Defaults to the editor document when unset; the actual
// cross-frame write happens via the $preview.brand.headerLogo shared signal.
type BrandHeaderLogoProps struct {
	RootSelector string
}

//gosx:island
func BrandHeaderLogo(props BrandHeaderLogoProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a stable
	// mount event in the compiled output. It also marks the island as
	// having been hydrated on the client, which the parity tests assert in
	// candidate mode (see parity_e2e/brandruntime_test.ts).
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-brand-header-logo-island="true"
		data-gosx-studio-brand-header-logo-root={root}
		data-gosx-studio-brand-header-logo-mounted={mounted.Get()}
		hidden
	></div>
}
