// brand_header_logo.gsx — GoSXStudioBrandRuntime.updateHeaderLogo island.
//
// Phase 3 slice-3 burn-down of the JS function updateHeaderLogo in
// gosx-studio/assets/studio-engines.js:1264 (which itself is a thin
// delegate to assets/preview-runtime.js:1027). See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-3-brandruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// # Transitional iframe contract
//
// updateHeaderLogo historically reaches across the storefront preview iframe
// to mutate the .brand / .brand__logo DOM elements directly (set src, alt,
// hidden, --brand-logo-width / --brand-logo-offset-x / --brand-logo-offset-y
// CSS custom properties; toggle the .brand--with-logo class).
//
// Per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md,
// the iframe stays and cross-frame mutations route through $preview.* shared
// signals subscribed by the preview document. The long-term destination for
// this method is a $preview.brand.headerLogo signal subscribed by the
// preview-side header .gsx component. That subscriber ships in slice 6
// (GoSXStudioPreviewRuntime burn-down).
//
// Until slice 6 ships, this island keeps the same iframe contract by
// delegating to window.GoSXStudioPreviewRuntime.updateHeaderLogo. Ownership
// has moved to the brand island; the mechanism is identical. Slice 6 will
// flip the mechanism to a $preview.brand.headerLogo write without
// revisiting brand ownership.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_brand_runtime_island_updateHeaderLogo that calls
// previewRuntime().updateHeaderLogo(payload).
//
// During the additive shipping window (Section B of the slice plan), the
// BridgeShim only invokes this island when ShellConfig.FeatureFlags has
// "brand-runtime-islands"=true; otherwise the legacy updateHeaderLogo path
// in studio-engines.js runs (which also delegates to PreviewRuntime).

package brandruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BrandHeaderLogoProps lets the consumer pass an opaque selector
// identifying the preview-frame root the updateHeaderLogo island should
// scope to. Defaults to the editor document when unset; the actual
// cross-frame write happens via the legacy PreviewRuntime delegation
// (slice 6 will replace this with a $preview.brand.headerLogo shared
// signal write that doesn't need a DOM root).
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
