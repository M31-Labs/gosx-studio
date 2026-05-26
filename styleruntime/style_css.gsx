// style_css.gsx — GoSXStudioStyleRuntime.{bindCSS,bindFonts} island.
//
// Phase 3 slice-5 burn-down of two CSS-injection JS functions in
// gosx-studio/assets/studio-engines.js:
//
//   - bindCSS(root)   — line 1570; binds the [data-editor-css] textarea so
//                       input dispatches PreviewRuntime.applyCSS(value).
//   - bindFonts(root) — line 1602; binds the three [data-editor-font-name] /
//                       [data-editor-font-url] slot pairs (display / body /
//                       mono) so input/change recomputes the combined
//                       @font-face + :root font-variable CSS and dispatches
//                       PreviewRuntime.applyFonts(css).
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-5-styleruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// Both methods are editor-side listener-binders that delegate the actual
// cross-frame mutation to PreviewRuntime.applyCSS / applyFonts. Per the
// parity-matrix split, applyCSS / applyFonts themselves are slice 6
// (GoSXStudioPreviewRuntime) burn-down targets — slice 5 only owns the
// editor-side controls that trigger them. The runtime bundle's
// island_runtime.js publishes:
//   window.__gosx_style_runtime_island_bindCSS
//   window.__gosx_style_runtime_island_bindFonts
// that the slice-5 BridgeShim delegates to when the
// "style-runtime-islands" feature flag is on; otherwise the legacy bindCSS /
// bindFonts functions in studio-engines.js run.

package styleruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// StyleCSSProps lets the consumer pass an opaque selector identifying the
// editor-side root the bindCSS / bindFonts islands should scope to.
// Defaults to the editor document when unset.
type StyleCSSProps struct {
	RootSelector string
}

//gosx:island
func StyleCSS(props StyleCSSProps) gosx.Node {
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
		data-gosx-studio-style-css-island="true"
		data-gosx-studio-style-css-root={root}
		data-gosx-studio-style-css-mounted={mounted.Get()}
		hidden
	></div>
}
