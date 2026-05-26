// style_theme.gsx — GoSXStudioStyleRuntime.{bindTheme,applyTheme} island.
//
// Phase 3 slice-5 burn-down of two theme-related JS functions in
// gosx-studio/assets/studio-engines.js:
//
//   - bindTheme(root)  — line 1421; binds theme kit / template / palette /
//                        image-ratio / custom-template / style-class /
//                        color-token controls; on input/change dispatches
//                        updateTheme + applies kit/preset side effects.
//   - updateTheme()    — line 1396; reads form state, calls
//                        PreviewRuntime.applyTheme(payload), updates theme
//                        swatches / kit cards / template cards / custom-
//                        template builder, then syncStyleControlButtons.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-5-styleruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// # Transitional iframe contract
//
// applyTheme is one of slice 5's three iframe-crossing methods. The legacy
// implementation delegates the cross-frame mutation to
// window.GoSXStudioPreviewRuntime.applyTheme — a thin wrapper that reaches
// into the preview document and toggles theme classes / tokens / palette
// variables.
//
// Per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md,
// the iframe stays and cross-frame mutations route through $preview.* shared
// signals subscribed by the preview document. The long-term destination for
// this method is a $preview.theme.* family of signals subscribed by the
// preview-side root .gsx component. That subscriber ships in slice 6
// (GoSXStudioPreviewRuntime burn-down).
//
// Until slice 6 ships, this island keeps the same iframe contract by
// delegating to window.GoSXStudioPreviewRuntime.applyTheme — exactly
// matching slice 3's transitional pattern for updateHeaderLogo and slice 4's
// for updateVisibilityState. Ownership has moved to the styleruntime island;
// the mechanism is identical. Slice 6 will flip the mechanism to a
// $preview.theme.* write without revisiting styleruntime ownership.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_style_runtime_island_bindTheme +
// window.__gosx_style_runtime_island_applyTheme that the slice-5 BridgeShim
// delegates to when the "style-runtime-islands" feature flag is on;
// otherwise the legacy bindTheme / updateTheme functions in studio-engines.js
// run.

package styleruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// StyleThemeProps lets the consumer pass an opaque selector identifying the
// editor-side root the bindTheme island should scope to. Defaults to the
// editor document when unset.
type StyleThemeProps struct {
	RootSelector string
}

//gosx:island
func StyleTheme(props StyleThemeProps) gosx.Node {
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
		data-gosx-studio-style-theme-island="true"
		data-gosx-studio-style-theme-root={root}
		data-gosx-studio-style-theme-mounted={mounted.Get()}
		hidden
	></div>
}
