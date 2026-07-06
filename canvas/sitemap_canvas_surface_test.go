package canvas

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderSiteMapCanvasSurfaceComposesFullWASMCanvasDefault(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteMapCanvasSurface(siteMapCanvasTestView(), SiteMapCanvasSurfaceOptions{
		Mode:         SiteMapCanvasSurfaceCanvasDefault,
		AssetVersion: "dev 1",
		Canvas: SiteMapCanvasOptions{
			Class:       "studio-site-map-canvas host-canvas",
			CanvasClass: "host-canvas__surface",
		},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-site-map-canvas-renderer="gosx-studio"`,
		`data-gosx-surface-kind="canvas2d"`,
		`data-gosx-engine-component="CanvasBoard"`,
		`class="studio-site-map-canvas host-canvas"`,
		`class="host-canvas__surface"`,
		canvasSelectionBridgePath + `?v=dev+1`,
		`data-gosx-studio-canvas-selection-bridge="true"`,
		`data-studio-site-map-canvas-default="true"`,
		canvasInlineEditPath + `?v=dev+1`,
		`data-gosx-studio-canvas-inline-edit="true"`,
		canvasDefaultInlineInstallerPath + `?v=dev+1`,
		`data-gosx-studio-canvas-inline-edit-installer="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("canvas-default surface missing %q:\n%s", fragment, html)
		}
	}
	for _, forbidden := range []string{
		canvas2DPainterPath,
		canvasWASMFreeClientPath,
		canvasContextualPanelPath,
		`data-gosx-canvas-wasm-free="true"`,
		`component:home:hero`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("canvas-default surface must not include %q:\n%s", forbidden, html)
		}
	}
}

func TestRenderSiteMapCanvasSurfaceComposesWASMFreeSurface(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteMapCanvasSurface(siteMapCanvasTestView(), SiteMapCanvasSurfaceOptions{
		Mode:         SiteMapCanvasSurfaceWASMFree,
		AssetVersion: "dev",
		Canvas: SiteMapCanvasOptions{
			Class:       "studio-site-map-canvas host-canvas",
			CanvasClass: "host-canvas__surface",
		},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-site-map-canvas-renderer="gosx-studio"`,
		`data-gosx-canvas-wasm-free="true"`,
		`data-gosx-canvas-bundle="studio-site-map-canvas-board"`,
		canvas2DPainterPath + `?v=dev`,
		`data-gosx-studio-canvas2d-painter="true"`,
		`data-studio-site-map-canvas-default="true"`,
		canvasInlineEditPath + `?v=dev`,
		`data-gosx-studio-canvas-inline-edit="true"`,
		canvasContextualPanelPath + `?v=dev`,
		`data-gosx-studio-canvas-contextual-panel="true"`,
		canvasWASMFreeClientPath + `?v=dev`,
		`data-gosx-studio-canvas-wasm-free-client="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("wasm-free surface missing %q:\n%s", fragment, html)
		}
	}
	for _, forbidden := range []string{
		`data-gosx-surface-kind="canvas2d"`,
		`data-gosx-engine-component="CanvasBoard"`,
		canvasSelectionBridgePath,
		canvasDefaultInlineInstallerPath,
		`component:home:hero`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("wasm-free surface must not include %q:\n%s", forbidden, html)
		}
	}
}

func TestRenderSiteMapCanvasSurfaceOffRendersZeroBytes(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteMapCanvasSurface(siteMapCanvasTestView(), SiteMapCanvasSurfaceOptions{
		Mode: SiteMapCanvasSurfaceOff,
	}))
	if html != "" {
		t.Fatalf("off mode must render zero bytes, got %q", html)
	}
}
