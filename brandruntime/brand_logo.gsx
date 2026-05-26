// brand_logo.gsx — GoSXStudioBrandRuntime.bindLogo island.
//
// Phase 3 slice-3 burn-down of the JS function bindBrandLogo in
// gosx-studio/assets/studio-engines.js:1269. See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-3-brandruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_brand_runtime_island_bindLogo that scans the DOM and wires
// up the editor-side brand-logo authoring surface (URL / alt / width / X /
// Y inputs, snap toggle + grid size, drag handle with keyboard nudges,
// reset button, readout). It operates on editor-side DOM only — the
// $brand.logo signal it writes is the island's own bookkeeping.
//
// The companion island for the iframe-crossing brand mutation
// (updateHeaderLogo) lives in brand_header_logo.gsx. bindLogo calls into
// updateHeaderLogo on every input change, so the two islands ship together.
//
// During the additive shipping window (Section B of the slice plan), the
// BridgeShim only invokes this island when ShellConfig.FeatureFlags has
// "brand-runtime-islands"=true; otherwise the legacy bindBrandLogo path in
// studio-engines.js runs.

package brandruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// BrandLogoProps lets the consumer pass an opaque selector identifying the
// authoring-surface root the bindLogo island should scope to. Defaults to
// the editor document when unset, matching the legacy bindBrandLogo(document)
// call site.
type BrandLogoProps struct {
	RootSelector string
}

//gosx:island
func BrandLogo(props BrandLogoProps) gosx.Node {
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
		data-gosx-studio-brand-logo-island="true"
		data-gosx-studio-brand-logo-root={root}
		data-gosx-studio-brand-logo-mounted={mounted.Get()}
		hidden
	></div>
}
