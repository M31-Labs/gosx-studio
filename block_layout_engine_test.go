package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBlockLayoutEngineSegmentsOwnsFrameChrome(t *testing.T) {
	view := map[string]any{
		"kicker":     "Home",
		"title":      "Sections",
		"countLabel": "2 sections",
	}

	segments := RenderBlockLayoutEngineSegments(view, BlockLayoutEngineOptions{})
	html := strings.Join([]string{
		gosx.RenderHTML(segments.RootOpen),
		gosx.RenderHTML(segments.Header),
		gosx.RenderHTML(segments.EngineHost),
		gosx.RenderHTML(segments.LayersOpen),
		gosx.RenderHTML(segments.LayersClose),
		gosx.RenderHTML(segments.LibraryOpen),
		gosx.RenderHTML(segments.LibraryClose),
		gosx.RenderHTML(segments.RootClose),
	}, "")

	for _, fragment := range []string{
		`class="studio-block-layout-engine"`,
		`data-studio-block-layout-panel="true"`,
		`data-studio-mode-panel="home"`,
		`data-studio-panel="block-layout"`,
		`data-studio-engine-source="gosx"`,
		`data-gosx-studio-block-layout-engine-renderer="gosx-studio"`,
		`<header class="studio-block-layout-engine__chrome">`,
		`<p class="kicker">Home</p>`,
		`<h2>Sections</h2>`,
		`class="studio-block-layout-engine__state" data-studio-block-layout-state="true">2 sections</output>`,
		`id="gosx-studio-block-layout-engine"`,
		`class="studio-block-layout-engine-host"`,
		`data-gosx-engine="GoSXStudioBlockLayout"`,
		`data-gosx-engine-kind="surface"`,
		`data-gosx-studio-engine="block-layout"`,
		`data-studio-engine-role="block-layout"`,
		`data-studio-block-layout-engine-surface="true"`,
		`data-gosx-engine-capabilities="pointer keyboard text-input animation"`,
		`class="studio-block-layout-engine__layers" data-studio-block-layout-layers="true"`,
		`class="studio-block-layout-engine__library" data-studio-block-layout-library="true"`,
		`</section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("block-layout engine segments missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("block-layout engine frame must not inject visible platform copy:\n%s", html)
	}
}

func TestBlockLayoutEngineHostFromMapNormalizesCapabilities(t *testing.T) {
	host := BlockLayoutEngineHostFromMap(map[string]any{
		"key":          " blocks ",
		"name":         "LayoutEngine",
		"mountId":      "layout-engine",
		"class":        "layout-host",
		"capabilities": " pointer, keyboard pointer text-input ",
	})
	segments := RenderBlockLayoutEngineSegments(nil, BlockLayoutEngineOptions{
		EngineHost:   host,
		EngineSource: "host-app",
		Kicker:       "Landing",
		Title:        "Rows",
		CountLabel:   "3 rows",
	})
	html := gosx.RenderHTML(gosx.Fragment(segments.RootOpen, segments.Header, segments.EngineHost, segments.RootClose))

	for _, fragment := range []string{
		`data-studio-engine-source="host-app"`,
		`<p class="kicker">Landing</p>`,
		`<h2>Rows</h2>`,
		`>3 rows</output>`,
		`id="layout-engine"`,
		`class="layout-host"`,
		`data-gosx-engine="LayoutEngine"`,
		`data-gosx-studio-engine="blocks"`,
		`data-studio-engine-role="blocks"`,
		`data-gosx-engine-capabilities="pointer keyboard text-input"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("block-layout engine host missing %q:\n%s", fragment, html)
		}
	}
}
