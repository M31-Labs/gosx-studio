package shell

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/canvas"
	"m31labs.dev/gosx-studio/core"
	"m31labs.dev/gosx-studio/hostruntime"
	"m31labs.dev/gosx-studio/panels"
	"m31labs.dev/gosx-studio/sitemap"
)

// This file is the Slice 8 drift guard for the frozen-copy debt the earlier
// (parallel) slices had to carry. The restructure DAG keeps canvas/sitemap/
// panels as leaf/peer packages that may not import hostruntime, shell, or each
// other, so a handful of small rendered-page-contract values and one widget
// renderer still live as byte-identical duplicates in
// canvas/support.go, sitemap/support.go, and panels/inspector.go. shell is the
// ONLY package that may import both sides, so these tests assert the copies
// stay identical to their canonical sources — drift now breaks the build
// instead of silently diverging (spec §4 Slice 8, risks R5/R10; the workbench
// map helpers themselves were consolidated into core this slice, so they no
// longer need a guard).

// TestInspectorWidgetFrozenCopyMatchesPanels asserts sitemap's frozen
// RenderInspectorControl (sitemap/support.go, needed by the site-map
// editable-control panel) renders byte-identically to the canonical
// panels.RenderInspectorControl (panels/inspector.go). sitemap may not import
// panels (peer), hence the frozen copy.
func TestInspectorWidgetFrozenCopyMatchesPanels(t *testing.T) {
	cases := []map[string]any{
		{"kind": string(core.ControlText), "value": "Hello", "actionLabel": "Save field",
			"formInputs": []map[string]string{{"name": "pageKey", "value": "home"}}},
		{"kind": string(core.ControlRichText), "value": "Body copy"},
		{"kind": string(core.ControlNumber), "value": "12"},
		{"kind": string(core.ControlLink), "value": "https://example.com"},
		{"kind": string(core.ControlColor), "value": "#ff0000"},
		{"kind": string(core.ControlToggle), "value": "true"},
		{"kind": string(core.ControlChoice), "value": "b",
			"options": []map[string]string{{"value": "a", "label": "A"}, {"value": "b", "label": "B"}}},
		{"kind": string(core.ControlMedia), "value": "asset-1"},
		{"kind": "flow", "value": "checkout"},
	}
	for i, control := range cases {
		canonical := gosx.RenderHTML(panels.RenderInspectorControl(control, "form-"+control["kind"].(string), "value"))
		frozen := gosx.RenderHTML(sitemap.RenderInspectorControl(control, "form-"+control["kind"].(string), "value"))
		if canonical != frozen {
			t.Fatalf("case %d (kind=%v): sitemap frozen RenderInspectorControl diverged from panels canonical\n panels:  %s\n sitemap: %s", i, control["kind"], canonical, frozen)
		}
	}
}

// TestCanvasRuntimePathsMatchHostruntime asserts canvas's frozen /_gosx/studio/*
// path constants (canvas/support.go) still equal hostruntime's canonical public
// path contract. canvas may not import hostruntime (peer), so it carries the
// literals; RenderSiteMapCanvasSurface renders <script src> tags from them.
// Rendering both surface modes emits all six paths, so we assert the canonical
// hostruntime.AssetHref(path, version) appears in the output — this guards the
// path constants AND the frozen assetHref helper in one shot.
func TestCanvasRuntimePathsMatchHostruntime(t *testing.T) {
	const version = "dev"
	var html strings.Builder
	for _, mode := range []canvas.SiteMapCanvasSurfaceMode{
		canvas.SiteMapCanvasSurfaceWASMFree,
		canvas.SiteMapCanvasSurfaceCanvasDefault,
	} {
		html.WriteString(gosx.RenderHTML(canvas.RenderSiteMapCanvasSurface(map[string]any{}, canvas.SiteMapCanvasSurfaceOptions{
			Mode:         mode,
			AssetVersion: version,
		})))
	}
	rendered := html.String()
	for _, path := range []string{
		hostruntime.CanvasSelectionBridgePath,
		hostruntime.CanvasInlineEditPath,
		hostruntime.Canvas2DPainterPath,
		hostruntime.CanvasWASMFreeClientPath,
		hostruntime.CanvasContextualPanelPath,
		hostruntime.CanvasDefaultInlineInstallerPath,
	} {
		want := hostruntime.AssetHref(path, version)
		if !strings.Contains(rendered, want) {
			t.Fatalf("canvas surface output missing canonical hostruntime path %q; the frozen copy in canvas/support.go has drifted from hostruntime/paths.go", want)
		}
	}
}

// TestBlockLayoutDefaultEngineNameMatchesShell asserts canvas's frozen
// blockLayoutEngineDefaultName (canvas/support.go) equals the canonical
// shell.BlockLayoutEngineName engine-name registry constant. canvas may not
// import shell.
func TestBlockLayoutDefaultEngineNameMatchesShell(t *testing.T) {
	rendered := gosx.RenderHTML(canvas.RenderBlockLayoutEngineHost(canvas.BlockLayoutEngineOptions{}))
	want := `data-gosx-engine="` + BlockLayoutEngineName + `"`
	if !strings.Contains(rendered, want) {
		t.Fatalf("canvas block-layout default engine name drifted from shell.BlockLayoutEngineName (%q); rendered: %s", BlockLayoutEngineName, rendered)
	}
}

// TestSiteMapDefaultEngineNameMatchesShell asserts sitemap's frozen
// siteMapEngineDefaultName (sitemap/support.go) equals the canonical
// shell.SiteMapEngineName engine-name registry constant. sitemap may not import
// shell.
func TestSiteMapDefaultEngineNameMatchesShell(t *testing.T) {
	render := sitemap.RenderSiteMapEngine(map[string]any{}, sitemap.SiteMapEngineOptions{})
	rendered := gosx.RenderHTML(render.Surface)
	want := `data-gosx-engine="` + SiteMapEngineName + `"`
	if !strings.Contains(rendered, want) {
		t.Fatalf("sitemap default engine name drifted from shell.SiteMapEngineName (%q); rendered: %s", SiteMapEngineName, rendered)
	}
}
