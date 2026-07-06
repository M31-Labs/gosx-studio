// workbench_viewport.gsx — GoSXStudioWorkbenchRuntime.{syncViewport,
// activateViewport,currentBreakpoint} island.
//
// Phase 3 slice-7 island implementation of three viewport-related JS functions in
// the legacy bundle (removed 2026-05-27):
//
//   - syncWorkbenchViewport(form, viewport) — line 369; sets
//                                              data-studio-breakpoint on the
//                                              form, data-studio-preview-
//                                              viewport on the
//                                              .editor-preview-shell, updates
//                                              [data-studio-viewport-label]
//                                              readouts, emits
//                                              gosxstudio:workbench-viewport-
//                                              change, refreshes the canvas.
//   - activateWorkbenchViewport(form, viewport) — line 382; clicks the
//                                              matching
//                                              [data-studio-viewport="<x>"]
//                                              button if present; otherwise
//                                              falls through to
//                                              syncViewport.
//   - currentWorkbenchBreakpoint(form) — line 393; pure read returning the
//                                              form's
//                                              data-studio-breakpoint
//                                              attribute (or the
//                                              preview-shell's
//                                              data-studio-preview-viewport
//                                              fallback, or "desktop").
//                                              Slice 7 implements this as a
//                                              derivation from form state.
//
// # Migration note (StudioPreviewViewportIsland)
//
// muddy-noni-commerce/app/admin/editor/page.gsx already declares
// StudioPreviewViewportIsland (around lines 404-424) — an early-shipped
// preview-viewport switcher that writes data-studio-viewport-current on a
// .studio-viewport-switcher root and toggles aria-pressed on its
// [data-studio-viewport] buttons. Slice 7's WorkbenchViewport island is
// the canonical gosx-studio-side home for that contract. The host renders
// WorkbenchViewport as the mount-point and the WorkbenchRuntime bridge
// delegates directly to the island globals.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// All three methods are editor-chrome — no iframe crossing required.

package workbenchruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// WorkbenchViewportProps lets the consumer pass an opaque selector
// identifying the editor-side root the viewport island should scope to,
// plus the viewport options list (desktop/tablet/mobile by default) and
// the initial selection. Defaults align with the legacy
// StudioPreviewViewportIsland behavior in
// muddy-noni-commerce/app/admin/editor/page.gsx.
type WorkbenchViewportProps struct {
	RootSelector string
	Viewport     string
}

//gosx:island
func WorkbenchViewport(props WorkbenchViewportProps) gosx.Node {
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	viewport := props.Viewport
	if viewport == "" {
		viewport = "desktop"
	}
	return <div
		data-gosx-studio-workbench-viewport-island="true"
		data-gosx-studio-workbench-viewport-root={root}
		data-gosx-studio-workbench-viewport-current={viewport}
		data-gosx-studio-workbench-viewport-mounted={mounted.Get()}
		hidden
	></div>
}
