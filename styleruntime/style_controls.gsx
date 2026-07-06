// style_controls.gsx — GoSXStudioStyleRuntime.{syncControlButtons,setControlValue,resetControlValue} island.
//
// Phase 3 slice-5 island implementation of three style-control JS functions in
// the legacy bundle (removed 2026-05-27):
//
//   - syncStyleControlButtons(root) — line 1848; for every
//     [data-studio-style-control], [data-studio-style-readout],
//     [data-studio-style-reset] in root, sync aria-pressed / .is-selected /
//     textContent against the live form values and kit defaults.
//   - setStyleControlValue(name, value) — line 1821; toggles the implicit
//     custom-template radio when the control belongs to a custom-* slot,
//     sets the named control's value (radio / input / select), dispatches
//     input/change events, then runs syncStyleControlButtons +
//     showStyleImpact(name, value, true).
//   - resetStyleControlValue(name) — line 1842; looks up the kit default for
//     name (kitDefaultForControl) and calls setStyleControlValue(name,
//     default).
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-5-styleruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// These three operate on editor-side DOM only — they mutate the workbench
// form's control values, button class/aria states, and readout labels.
// setControlValue does call into showImpact (the iframe-crossing method in
// style_impact.gsx), but the call site stays editor-side; the cross-frame
// overlay is handled by showImpact's transitional delegation. The runtime
// bundle's island_runtime.js publishes:
//   window.__gosx_style_runtime_island_syncControlButtons
//   window.__gosx_style_runtime_island_setControlValue
//   window.__gosx_style_runtime_island_resetControlValue
// that the slice-5 BridgeShim delegates to directly. Post 2026-05-27 the
// legacy JS bundle is gone; the island path is the only path. See Phase 3
// burn-down.

package styleruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// StyleControlsProps lets the consumer pass an opaque selector identifying
// the editor-side root the syncControlButtons / setControlValue /
// resetControlValue islands should scope to. Defaults to the editor
// document when unset.
type StyleControlsProps struct {
	RootSelector string
}

//gosx:island
func StyleControls(props StyleControlsProps) gosx.Node {
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
		data-gosx-studio-style-controls-island="true"
		data-gosx-studio-style-controls-root={root}
		data-gosx-studio-style-controls-mounted={mounted.Get()}
		hidden
	></div>
}
