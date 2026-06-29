package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type SiteMapCanvasSurfaceMode string

const (
	SiteMapCanvasSurfaceOff           SiteMapCanvasSurfaceMode = "off"
	SiteMapCanvasSurfaceCoRender      SiteMapCanvasSurfaceMode = "co-render"
	SiteMapCanvasSurfaceCanvasDefault SiteMapCanvasSurfaceMode = "canvas-default"
	SiteMapCanvasSurfaceWASMFree      SiteMapCanvasSurfaceMode = "wasm-free"
)

type SiteMapCanvasSurfaceOptions struct {
	Mode         SiteMapCanvasSurfaceMode
	Canvas       SiteMapCanvasOptions
	AssetVersion string
}

func RenderSiteMapCanvasSurface(siteMapView map[string]any, options SiteMapCanvasSurfaceOptions) gosx.Node {
	mode := normalizeSiteMapCanvasSurfaceMode(options.Mode)
	if mode == SiteMapCanvasSurfaceOff {
		return gosx.Fragment()
	}

	canvasOptions := options.Canvas
	canvasOptions.Enabled = true
	canvasOptions.PageArtboardsOnly = true
	canvasOptions.Class = FirstNonEmpty(canvasOptions.Class, "studio-site-map-canvas")
	canvasOptions.WASMFree = mode == SiteMapCanvasSurfaceWASMFree

	canvas := RenderSiteMapCanvasEngine(siteMapView, canvasOptions)
	if mode == SiteMapCanvasSurfaceWASMFree {
		return gosx.Fragment(canvas,
			siteMapCanvasSurfaceScript(Canvas2DPainterPath, options.AssetVersion,
				gosx.Attr("data-gosx-studio-canvas2d-painter", "true"),
				gosx.Attr("data-studio-site-map-canvas-default", "true"),
			),
			siteMapCanvasSurfaceScript(CanvasInlineEditPath, options.AssetVersion,
				gosx.Attr("data-gosx-studio-canvas-inline-edit", "true"),
			),
			siteMapCanvasSurfaceScript(CanvasContextualPanelPath, options.AssetVersion,
				gosx.Attr("data-gosx-studio-canvas-contextual-panel", "true"),
			),
			siteMapCanvasSurfaceScript(CanvasWASMFreeClientPath, options.AssetVersion,
				gosx.Attr("data-gosx-studio-canvas-wasm-free-client", "true"),
			),
		)
	}

	bridgeAttrs := []any{gosx.Attr("data-gosx-studio-canvas-selection-bridge", "true")}
	if mode == SiteMapCanvasSurfaceCanvasDefault {
		bridgeAttrs = append(bridgeAttrs, gosx.Attr("data-studio-site-map-canvas-default", "true"))
	}
	return gosx.Fragment(canvas,
		siteMapCanvasSurfaceScript(CanvasSelectionBridgePath, options.AssetVersion, bridgeAttrs...),
		siteMapCanvasSurfaceScript(CanvasInlineEditPath, options.AssetVersion,
			gosx.Attr("data-gosx-studio-canvas-inline-edit", "true"),
		),
		siteMapCanvasSurfaceScript(CanvasDefaultInlineInstallerPath, options.AssetVersion,
			gosx.Attr("data-gosx-studio-canvas-inline-edit-installer", "true"),
		),
	)
}

func normalizeSiteMapCanvasSurfaceMode(mode SiteMapCanvasSurfaceMode) SiteMapCanvasSurfaceMode {
	switch SiteMapCanvasSurfaceMode(strings.TrimSpace(string(mode))) {
	case SiteMapCanvasSurfaceOff:
		return SiteMapCanvasSurfaceOff
	case SiteMapCanvasSurfaceCanvasDefault:
		return SiteMapCanvasSurfaceCanvasDefault
	case SiteMapCanvasSurfaceWASMFree:
		return SiteMapCanvasSurfaceWASMFree
	default:
		return SiteMapCanvasSurfaceCoRender
	}
}

func siteMapCanvasSurfaceScript(path string, version string, attrs ...any) gosx.Node {
	all := []any{
		gosx.Attr("src", AssetHref(path, version)),
		gosx.Attr("defer", true),
	}
	all = append(all, attrs...)
	return gosx.El("script", gosx.Attrs(all...))
}
