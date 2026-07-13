package studio

// This file is the deprecated compatibility facade for the symbols that used
// to live in sitemap_engine.go, sitemap_view.go, sitemap_board.go,
// sitemap_authoring_panels.go, sitemap_authoring_forms.go,
// site_navigator_panel.go, and site_navigator_island.gsx before Slice 5 of
// the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) extracted them into
// the m31labs.dev/gosx-studio/sitemap package: the visual site-map board
// (graph engine render, board/view projections, authoring panels/forms, and
// site navigator).
//
// Every exported type below is a type ALIAS (not a new type), so struct
// literals, embedding, method sets, and assignability are all unchanged for
// existing callers (muddy-noni-commerce, pajaritos-forest-school,
// gosx-cms/studio). Every exported free function is a thin forwarding
// wrapper. Methods on the aliased types are inherited automatically through
// the alias and do not need wrappers here. Every exported constant keeps its
// exact string value (rendered data-studio-*/data-gosx-studio-* attribute
// names and values are part of the rendered-page contract and must not
// change — spec risk R5).
//
// sitemap imports core and authoring (see the spec's §1 Import DAG):
// AuthoringSiteMapView consumes authoring.AuthoringSurface to build the
// site-map's view projection. sitemap never imports canvas, hostruntime, or
// the not-yet-extracted panels/shell territory it still borrows a few
// small helpers from (see sitemap/support.go's frozen, byte-identical
// copies — the same precedent canvas/support.go set in Slice 4).
//
// site_navigator_island.gsx moved with its package decl rewritten to
// `package sitemap`; it declares SiteNavigatorIsland via the gosx toolchain
// (`//gosx:island`), which is not a linkable Go symbol in either the old or
// new package (no .go file ever referenced it — only the source-contract
// test asserts its literal text), so there is no Go-level alias to add for
// it here.
//
// Deprecated: import m31labs.dev/gosx-studio/sitemap directly; this facade
// will be removed after the v0.6.x compatibility window (see spec §9).

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/sitemap"
)

// --- Site-map engine (sitemap/sitemap_engine.go) ---

// Deprecated: use sitemap.SiteMapEngineRuntime.
type SiteMapEngineRuntime = sitemap.SiteMapEngineRuntime

// Deprecated: use sitemap.SiteMapEngineHostOptions.
type SiteMapEngineHostOptions = sitemap.SiteMapEngineHostOptions

// Deprecated: use sitemap.SiteMapEngineOptions.
type SiteMapEngineOptions = sitemap.SiteMapEngineOptions

// Deprecated: use sitemap.SiteMapEngineSegments.
type SiteMapEngineSegments = sitemap.SiteMapEngineSegments

// Deprecated: use sitemap.SiteMapEngineRender.
type SiteMapEngineRender = sitemap.SiteMapEngineRender

// Deprecated: use sitemap.SiteMapEngineHostFromMap.
func SiteMapEngineHostFromMap(host map[string]any) SiteMapEngineHostOptions {
	return sitemap.SiteMapEngineHostFromMap(host)
}

// Deprecated: use sitemap.RenderSiteMapEngine.
func RenderSiteMapEngine(siteMapView map[string]any, options SiteMapEngineOptions) SiteMapEngineRender {
	return sitemap.RenderSiteMapEngine(siteMapView, options)
}

// Deprecated: use sitemap.RenderSiteMapEngineSegments.
func RenderSiteMapEngineSegments(siteMapView map[string]any, options SiteMapEngineOptions) SiteMapEngineSegments {
	return sitemap.RenderSiteMapEngineSegments(siteMapView, options)
}

// --- Site-map view projection (sitemap/sitemap_view.go) ---

// Deprecated: use sitemap.SiteMapViewOptions.
type SiteMapViewOptions = sitemap.SiteMapViewOptions

// Deprecated: use sitemap.SiteMapNodePosition.
type SiteMapNodePosition = sitemap.SiteMapNodePosition

// Deprecated: use sitemap.SiteMapAuthoringView.
func SiteMapAuthoringView(siteMap SiteMap, options SiteMapViewOptions) map[string]any {
	return sitemap.SiteMapAuthoringView(siteMap, options)
}

// Deprecated: use sitemap.AuthoringSiteMapView.
func AuthoringSiteMapView(surface AuthoringSurface, options SiteMapViewOptions) map[string]any {
	return sitemap.AuthoringSiteMapView(surface, options)
}

// --- Site-map board (sitemap/sitemap_board.go) ---

// Deprecated: use sitemap.SiteMapBoardOptions.
type SiteMapBoardOptions = sitemap.SiteMapBoardOptions

// Deprecated: use sitemap.RenderSiteMapBoard.
func RenderSiteMapBoard(siteMapView map[string]any, options SiteMapBoardOptions) gosx.Node {
	return sitemap.RenderSiteMapBoard(siteMapView, options)
}

// --- Site-map authoring panels (sitemap/sitemap_authoring_panels.go) ---

// Deprecated: use sitemap.SiteMapAuthoringPanelsOptions.
type SiteMapAuthoringPanelsOptions = sitemap.SiteMapAuthoringPanelsOptions

// Deprecated: use sitemap.RenderSiteMapAuthoringPanels.
func RenderSiteMapAuthoringPanels(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	return sitemap.RenderSiteMapAuthoringPanels(siteMapView, options)
}

// --- Site-map authoring forms (sitemap/sitemap_authoring_forms.go) ---

// Deprecated: use sitemap.SiteMapAuthoringFormsOptions.
type SiteMapAuthoringFormsOptions = sitemap.SiteMapAuthoringFormsOptions

// Deprecated: use sitemap.RenderSiteMapAuthoringForms.
func RenderSiteMapAuthoringForms(siteMapView map[string]any, options SiteMapAuthoringFormsOptions) gosx.Node {
	return sitemap.RenderSiteMapAuthoringForms(siteMapView, options)
}

// --- Site navigator panel (sitemap/site_navigator_panel.go) ---

// Deprecated: use sitemap.SiteNavigatorItem.
type SiteNavigatorItem = sitemap.SiteNavigatorItem

// Deprecated: use sitemap.SiteNavigatorProps.
type SiteNavigatorProps = sitemap.SiteNavigatorProps

// Deprecated: use sitemap.SiteNavigatorNewPageItem.
type SiteNavigatorNewPageItem = sitemap.SiteNavigatorNewPageItem

// Deprecated: use sitemap.SiteNavigatorPanelOptions.
type SiteNavigatorPanelOptions = sitemap.SiteNavigatorPanelOptions

// Deprecated: use sitemap.SiteNavigatorPropsFromMap.
func SiteNavigatorPropsFromMap(view map[string]any) SiteNavigatorProps {
	return sitemap.SiteNavigatorPropsFromMap(view)
}

// Deprecated: use sitemap.RenderSiteNavigatorPanel.
func RenderSiteNavigatorPanel(props SiteNavigatorProps, options SiteNavigatorPanelOptions) gosx.Node {
	return sitemap.RenderSiteNavigatorPanel(props, options)
}

// Deprecated: use sitemap.RenderSiteNavigatorNewPage.
func RenderSiteNavigatorNewPage(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	return sitemap.RenderSiteNavigatorNewPage(view, options)
}

// Deprecated: use sitemap.RenderSiteNavigatorHeader.
func RenderSiteNavigatorHeader(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	return sitemap.RenderSiteNavigatorHeader(view, options)
}

// Deprecated: use sitemap.RenderSiteNavigatorList.
func RenderSiteNavigatorList(view map[string]any, options SiteNavigatorPanelOptions) gosx.Node {
	return sitemap.RenderSiteNavigatorList(view, options)
}

// --- Unexported shims for not-yet-moved root files ---
//
// sitemap_authoring_panels.go and sitemap_engine.go also carried unexported
// helpers that root-only files (not moved until later slices) call by their
// original lowercase names: plugin_test.go
// (authoringSiteMapComponentTemplateViews — stays at root pending a later
// cleanup slice, see spec §1 core row) and control_locked_test.go
// (authoringSiteMapControlViews, authoringSiteMapEditableControlView,
// authoringSiteMapStaticControlViews — same). The sitemap package exports the
// equivalent implementations under capitalized names (matching the
// canvas/block_layout_engine.go precedent from Slice 4); these vars keep
// those call sites compiling unchanged, and will disappear once those files
// move to their own subpackages in later slices.
//
// The siteMapEngineCapabilityNames shim (formerly here for
// canvas_test_fixtures_test.go) was removed in Slice 8 when that test moved to
// the shell package; shell references sitemap.SiteMapEngineCapabilityNames
// directly. Likewise siteMapHiddenInputs/siteMapInputViews/siteMapMapList
// (formerly shimmed for inspector.go/inspector_fields.go) were removed in
// Slice 6 when those files moved to panels.
var (
	authoringSiteMapComponentTemplateViews = sitemap.AuthoringSiteMapComponentTemplateViews
	authoringSiteMapControlViews           = sitemap.AuthoringSiteMapControlViews
	authoringSiteMapEditableControlView    = sitemap.AuthoringSiteMapEditableControlView
	authoringSiteMapStaticControlViews     = sitemap.AuthoringSiteMapStaticControlViews
)
