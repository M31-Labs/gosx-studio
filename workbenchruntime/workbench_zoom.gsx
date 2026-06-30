// workbench_zoom.gsx — GoSXStudioWorkbenchRuntime.{syncZoom,activateZoom}
// island.
//
// Phase 3 slice-7 island implementation of two zoom-related JS functions in
// the legacy bundle (removed 2026-05-27):
//
//   - syncWorkbenchZoom(form, zoom) — line 421; sets data-studio-canvas-zoom
//                                      on the [data-studio-canvas] element,
//                                      emits gosxstudio:workbench-zoom-change,
//                                      refreshes the canvas via a resize
//                                      event.
//   - activateWorkbenchZoom(form, zoom) — line 430; clicks the matching
//                                      [data-studio-zoom="<x>"] button if
//                                      present; otherwise falls through to
//                                      syncZoom.
//
// # Migration note (StudioZoomControlsIsland)
//
// muddy-noni-commerce/app/admin/editor/page.gsx already declares
// StudioZoomControlsIsland (around lines 322-341) — an early-shipped
// zoom-toolbar switcher that writes data-studio-zoom-current on a
// .studio-zoombar root and toggles aria-pressed on its [data-studio-zoom]
// buttons. Slice 7's WorkbenchZoom island is the canonical gosx-studio-side
// home for that contract. The host renders WorkbenchZoom as the mount-point
// and the WorkbenchRuntime bridge delegates directly to the island globals.
//
// See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// Both methods are editor-chrome — no iframe crossing required.

package workbenchruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// WorkbenchZoomProps lets the consumer pass an opaque selector identifying
// the editor-side root the zoom island should scope to, plus the initial
// zoom level. Defaults align with the legacy StudioZoomControlsIsland
// behavior in muddy-noni-commerce/app/admin/editor/page.gsx.
type WorkbenchZoomProps struct {
	RootSelector string
	Zoom         string
}

//gosx:island
func WorkbenchZoom(props WorkbenchZoomProps) gosx.Node {
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	zoom := props.Zoom
	if zoom == "" {
		zoom = "fit"
	}
	return <div
		data-gosx-studio-workbench-zoom-island="true"
		data-gosx-studio-workbench-zoom-root={root}
		data-gosx-studio-workbench-zoom-current={zoom}
		data-gosx-studio-workbench-zoom-mounted={mounted.Get()}
		hidden
	></div>
}
