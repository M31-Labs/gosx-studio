package studio

// This file is the deprecated compatibility facade for the symbols that used
// to live in sitemap_canvas.go, sitemap_canvas_surface.go, page_surface.go,
// page_thumbnail.go, block_layout_engine.go, block_layout_contract.go, and
// engine_hosts.go before Slice 4 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) extracted them into
// the m31labs.dev/gosx-studio/canvas package: the page-canvas engine hosting
// and server-rendered surface markup (artboards, thumbnails, block-layout DOM
// contract).
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
// canvas imports only core (see the spec's §1 Import DAG): the
// StudioEngineRuntime/BlockLayoutEngineRuntime engine-runtime interfaces stay
// consumer-side via structural typing, so canvas never imports hostruntime,
// even though sitemap_canvas_surface.go renders <script src=...> tags
// pointing at hostruntime-owned runtime paths (canvas carries its own frozen,
// byte-identical copies of those six path constants — see canvas/support.go).
//
// Deprecated: import m31labs.dev/gosx-studio/canvas directly; this facade
// will be removed after the v0.6.x compatibility window (see spec §9).

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/canvas"
)

// --- Site-map Canvas2D engine (canvas/sitemap_canvas.go) ---

// Deprecated: use canvas.SiteMapCanvasOptions.
type SiteMapCanvasOptions = canvas.SiteMapCanvasOptions

// Deprecated: use canvas.RenderSiteMapCanvasEngine.
func RenderSiteMapCanvasEngine(siteMapView map[string]any, options SiteMapCanvasOptions) gosx.Node {
	return canvas.RenderSiteMapCanvasEngine(siteMapView, options)
}

// --- Site-map canvas surface composition (canvas/sitemap_canvas_surface.go) ---

// Deprecated: use canvas.SiteMapCanvasSurfaceMode.
type SiteMapCanvasSurfaceMode = canvas.SiteMapCanvasSurfaceMode

const (
	// Deprecated: use canvas.SiteMapCanvasSurfaceOff.
	SiteMapCanvasSurfaceOff = canvas.SiteMapCanvasSurfaceOff
	// Deprecated: use canvas.SiteMapCanvasSurfaceCoRender.
	SiteMapCanvasSurfaceCoRender = canvas.SiteMapCanvasSurfaceCoRender
	// Deprecated: use canvas.SiteMapCanvasSurfaceCanvasDefault.
	SiteMapCanvasSurfaceCanvasDefault = canvas.SiteMapCanvasSurfaceCanvasDefault
	// Deprecated: use canvas.SiteMapCanvasSurfaceWASMFree.
	SiteMapCanvasSurfaceWASMFree = canvas.SiteMapCanvasSurfaceWASMFree
)

// Deprecated: use canvas.SiteMapCanvasSurfaceOptions.
type SiteMapCanvasSurfaceOptions = canvas.SiteMapCanvasSurfaceOptions

// Deprecated: use canvas.RenderSiteMapCanvasSurface.
func RenderSiteMapCanvasSurface(siteMapView map[string]any, options SiteMapCanvasSurfaceOptions) gosx.Node {
	return canvas.RenderSiteMapCanvasSurface(siteMapView, options)
}

// --- Page surface markup (canvas/page_surface.go) ---

// Deprecated: use canvas.PageSurfaceTheme.
type PageSurfaceTheme = canvas.PageSurfaceTheme

// Deprecated: use canvas.HomePageSurfaceInput.
type HomePageSurfaceInput = canvas.HomePageSurfaceInput

// Deprecated: use canvas.PageArtboardSurfaceInput.
type PageArtboardSurfaceInput = canvas.PageArtboardSurfaceInput

// Deprecated: use canvas.PageArtboardComponent.
type PageArtboardComponent = canvas.PageArtboardComponent

// Deprecated: use canvas.RenderHomePageSurfaceMarkup.
func RenderHomePageSurfaceMarkup(input HomePageSurfaceInput) string {
	return canvas.RenderHomePageSurfaceMarkup(input)
}

// Deprecated: use canvas.RenderPageArtboardSurfaceMarkup.
func RenderPageArtboardSurfaceMarkup(input PageArtboardSurfaceInput) string {
	return canvas.RenderPageArtboardSurfaceMarkup(input)
}

// --- Page thumbnail rasterization (canvas/page_thumbnail.go) ---

// Deprecated: use canvas.PageThumbnailTheme.
type PageThumbnailTheme = canvas.PageThumbnailTheme

// Deprecated: use canvas.PageThumbnailHero.
type PageThumbnailHero = canvas.PageThumbnailHero

// Deprecated: use canvas.PageThumbnailInput.
type PageThumbnailInput = canvas.PageThumbnailInput

// Deprecated: use canvas.PageThumbnailDataURI.
func PageThumbnailDataURI(in PageThumbnailInput) string { return canvas.PageThumbnailDataURI(in) }

// --- Block-layout engine (canvas/block_layout_engine.go) ---

// Deprecated: use canvas.BlockLayoutEngineRuntime.
type BlockLayoutEngineRuntime = canvas.BlockLayoutEngineRuntime

// Deprecated: use canvas.BlockLayoutEngineHostOptions.
type BlockLayoutEngineHostOptions = canvas.BlockLayoutEngineHostOptions

// Deprecated: use canvas.BlockLayoutEngineOptions.
type BlockLayoutEngineOptions = canvas.BlockLayoutEngineOptions

// Deprecated: use canvas.BlockLayoutEngineSegments.
type BlockLayoutEngineSegments = canvas.BlockLayoutEngineSegments

// Deprecated: use canvas.BlockLayoutEngineHostFromMap.
func BlockLayoutEngineHostFromMap(host map[string]any) BlockLayoutEngineHostOptions {
	return canvas.BlockLayoutEngineHostFromMap(host)
}

// Deprecated: use canvas.RenderBlockLayoutEngineSegments.
func RenderBlockLayoutEngineSegments(view map[string]any, options BlockLayoutEngineOptions) BlockLayoutEngineSegments {
	return canvas.RenderBlockLayoutEngineSegments(view, options)
}

// Deprecated: use canvas.RenderBlockLayoutEngineHost.
func RenderBlockLayoutEngineHost(options BlockLayoutEngineOptions) gosx.Node {
	return canvas.RenderBlockLayoutEngineHost(options)
}

// Deprecated: use canvas.RenderBlockLayoutEngine.
func RenderBlockLayoutEngine(view map[string]any, options BlockLayoutEngineOptions) gosx.Node {
	return canvas.RenderBlockLayoutEngine(view, options)
}

// --- Block-layout runtime attribute contract (canvas/block_layout_contract.go) ---

const (
	// Deprecated: use canvas.BlockLayoutRowAttr.
	BlockLayoutRowAttr = canvas.BlockLayoutRowAttr
	// Deprecated: use canvas.BlockLayoutOrderAttr.
	BlockLayoutOrderAttr = canvas.BlockLayoutOrderAttr
	// Deprecated: use canvas.BlockLayoutIndexAttr.
	BlockLayoutIndexAttr = canvas.BlockLayoutIndexAttr
	// Deprecated: use canvas.BlockLayoutHandleAttr.
	BlockLayoutHandleAttr = canvas.BlockLayoutHandleAttr
	// Deprecated: use canvas.BlockLayoutMoveAttr.
	BlockLayoutMoveAttr = canvas.BlockLayoutMoveAttr
	// Deprecated: use canvas.BlockLayoutAddBlockAttr.
	BlockLayoutAddBlockAttr = canvas.BlockLayoutAddBlockAttr
	// Deprecated: use canvas.BlockLayoutVisibleAttr.
	BlockLayoutVisibleAttr = canvas.BlockLayoutVisibleAttr
	// Deprecated: use canvas.BlockLayoutPillAttr.
	BlockLayoutPillAttr = canvas.BlockLayoutPillAttr
	// Deprecated: use canvas.BlockLayoutSelectedClass.
	BlockLayoutSelectedClass = canvas.BlockLayoutSelectedClass
	// Deprecated: use canvas.BlockLayoutReorderEvent.
	BlockLayoutReorderEvent = canvas.BlockLayoutReorderEvent
	// Deprecated: use canvas.BlockLayoutSelectEvent.
	BlockLayoutSelectEvent = canvas.BlockLayoutSelectEvent
)

// --- Studio engine hosts (canvas/engine_hosts.go) ---

// Deprecated: use canvas.StudioEngineHostsOptions.
type StudioEngineHostsOptions = canvas.StudioEngineHostsOptions

// Deprecated: use canvas.StudioEngineRuntime.
type StudioEngineRuntime = canvas.StudioEngineRuntime

// Deprecated: use canvas.RenderStudioEngineHosts.
func RenderStudioEngineHosts(hosts []map[string]any, options StudioEngineHostsOptions) gosx.Node {
	return canvas.RenderStudioEngineHosts(hosts, options)
}

// --- Unexported shims for not-yet-moved root files ---
//
// block_layout_engine.go also carried an unexported "open tag" render helper
// that three still-root-resident panel files (brand_panel.go,
// home_layers_panel.go, advanced_panel.go — panels package territory, Slice
// 6) call by its original lowercase name. canvas exports the equivalent
// implementation as RenderBlockLayoutEngineOpenTag (see that package's doc
// comment); this var keeps those call sites compiling unchanged, and will
// disappear once those files move to their own subpackage in a later slice.
var renderBlockLayoutEngineOpenTag = canvas.RenderBlockLayoutEngineOpenTag
