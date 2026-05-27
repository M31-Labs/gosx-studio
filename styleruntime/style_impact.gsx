// style_impact.gsx — GoSXStudioStyleRuntime.{showImpact,restoreImpact} island.
//
// Phase 3 slice-5 island implementation of two impact-overlay JS functions in
// the legacy bundle (removed 2026-05-27):
//
//   - showStyleImpact(name, value, committed) — line 1785; looks up impact
//     metadata for the named control, calls
//     PreviewRuntime.applyStyleImpact(selector) for the count, writes the
//     editor-side impact-readout labels (label / summary / count / scope /
//     state), and toggles the impact panel's is-live / has-change classes.
//   - restoreStyleImpact()                    — line 1803; restores the last
//     committed impact via showStyleImpact (when present), or clears the
//     readouts to a "No active recipe" state and calls
//     PreviewRuntime.applyStyleImpact("") to remove overlay markers.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-5-styleruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// # Transitional iframe contract
//
// showImpact and restoreImpact are two of slice 5's three iframe-crossing
// methods (applyTheme is the third — see style_theme.gsx). The legacy
// implementation delegates the cross-frame overlay markup to
// window.GoSXStudioPreviewRuntime.applyStyleImpact — a thin wrapper that
// reaches into the preview document, injects overlay markers matching the
// supplied selector, and returns the count of affected nodes.
//
// Per
// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md,
// the iframe stays and cross-frame mutations route through $preview.* shared
// signals subscribed by the preview document. The long-term destination for
// these methods is a $preview.style.impact.selector signal subscribed by the
// preview-side overlay .gsx component. That subscriber ships in slice 6
// (GoSXStudioPreviewRuntime burn-down).
//
// Until slice 6 ships, this island keeps the same iframe contract by
// delegating to window.GoSXStudioPreviewRuntime.applyStyleImpact — exactly
// matching slice 3's transitional pattern for updateHeaderLogo and slice 4's
// for updateVisibilityState. Ownership has moved to the styleruntime island;
// the mechanism is identical. Slice 6 will flip the mechanism to a
// $preview.style.impact.* write without revisiting styleruntime ownership.
//
// The island is a mount-point marker — the consumer places it inside the
// Studio chrome and the runtime bundle's island_runtime.js publishes:
//   window.__gosx_style_runtime_island_showImpact
//   window.__gosx_style_runtime_island_restoreImpact
// that the slice-5 BridgeShim delegates to when the "style-runtime-islands"
// Post 2026-05-27 the legacy JS bundle is gone; the island path is the
// only path. See Phase 3 burn-down.
// delegate to PreviewRuntime).

package styleruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// StyleImpactProps lets the consumer pass an opaque selector identifying
// the editor-side root the showImpact / restoreImpact islands should scope
// to. Defaults to the editor document when unset; the actual cross-frame
// overlay write happens via the legacy PreviewRuntime delegation (slice 6
// will replace this with a $preview.style.impact.selector shared signal
// write that doesn't need a DOM root).
type StyleImpactProps struct {
	RootSelector string
}

//gosx:island
func StyleImpact(props StyleImpactProps) gosx.Node {
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
		data-gosx-studio-style-impact-island="true"
		data-gosx-studio-style-impact-root={root}
		data-gosx-studio-style-impact-mounted={mounted.Get()}
		hidden
	></div>
}
