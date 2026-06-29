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

func TestRenderBlockLayoutEngineWrapsSlotsInsideRoot(t *testing.T) {
	view := map[string]any{
		"kicker":     "Home",
		"title":      "Sections",
		"countLabel": "2 sections",
	}

	html := gosx.RenderHTML(RenderBlockLayoutEngine(view, BlockLayoutEngineOptions{
		LayersNode:  gosx.El("div", gosx.Attrs(gosx.Attr("data-test-layers-slot", "true")), gosx.Text("layers")),
		LibraryNode: gosx.El("aside", gosx.Attrs(gosx.Attr("data-test-library-slot", "true")), gosx.Text("library")),
	}))

	for _, fragment := range []string{
		`<section class="studio-block-layout-engine"`,
		`data-studio-block-layout-panel="true"`,
		`data-studio-engine-source="gosx"`,
		`data-gosx-studio-block-layout-engine-renderer="gosx-studio"`,
		`<header class="studio-block-layout-engine__chrome">`,
		`<h2>Sections</h2>`,
		`id="gosx-studio-block-layout-engine"`,
		`data-studio-block-layout-engine-surface="true"`,
		`class="studio-block-layout-engine__layers" data-studio-block-layout-layers="true"><div data-test-layers-slot="true">layers</div></div>`,
		`class="studio-block-layout-engine__library" data-studio-block-layout-library="true"><aside data-test-library-slot="true">library</aside></div>`,
		`</section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("block-layout engine missing %q:\n%s", fragment, html)
		}
	}
	if strings.Index(html, `<section`) > strings.Index(html, `<header`) ||
		strings.Index(html, `<header`) > strings.Index(html, `id="gosx-studio-block-layout-engine"`) ||
		strings.Index(html, `id="gosx-studio-block-layout-engine"`) > strings.Index(html, `data-test-layers-slot`) ||
		strings.Index(html, `data-test-layers-slot`) > strings.Index(html, `data-test-library-slot`) {
		t.Fatalf("block-layout engine did not preserve root/header/host/layers/library order:\n%s", html)
	}
}

func TestRenderBlockLayoutEngineUsesExplicitEngineHostNode(t *testing.T) {
	t.Run("empty explicit host falls back to default", func(t *testing.T) {
		html := gosx.RenderHTML(RenderBlockLayoutEngine(map[string]any{"countLabel": "1 section"}, BlockLayoutEngineOptions{
			EngineHostNode: gosx.Fragment(),
			LayersNode:     gosx.El("div", gosx.Attrs(gosx.Attr("data-test-layers-slot", "true"))),
			LibraryNode:    gosx.El("div", gosx.Attrs(gosx.Attr("data-test-library-slot", "true"))),
		}))

		for _, fragment := range []string{
			`data-gosx-studio-block-layout-engine-renderer="gosx-studio"`,
			`id="gosx-studio-block-layout-engine"`,
			`data-gosx-engine="GoSXStudioBlockLayout"`,
			`data-studio-block-layout-layers="true"`,
			`data-test-layers-slot="true"`,
			`data-studio-block-layout-library="true"`,
			`data-test-library-slot="true"`,
		} {
			if !strings.Contains(html, fragment) {
				t.Fatalf("block-layout engine fallback missing %q:\n%s", fragment, html)
			}
		}
		if strings.Contains(html, `data-route-rendered-engine-host="true"`) {
			t.Fatalf("block-layout engine should not render an empty explicit host:\n%s", html)
		}
	})

	t.Run("non-empty explicit host overrides default", func(t *testing.T) {
		html := gosx.RenderHTML(RenderBlockLayoutEngine(map[string]any{"countLabel": "1 section"}, BlockLayoutEngineOptions{
			EngineHostNode: gosx.El("surface", gosx.Attrs(
				gosx.Attr("id", "route-block-layout-host"),
				gosx.Attr("data-route-rendered-engine-host", "true"),
			)),
			LayersNode:  gosx.El("div", gosx.Attrs(gosx.Attr("data-test-layers-slot", "true"))),
			LibraryNode: gosx.El("div", gosx.Attrs(gosx.Attr("data-test-library-slot", "true"))),
		}))

		for _, fragment := range []string{
			`data-gosx-studio-block-layout-engine-renderer="gosx-studio"`,
			`<surface id="route-block-layout-host" data-route-rendered-engine-host="true"></surface>`,
			`data-studio-block-layout-layers="true"`,
			`data-test-layers-slot="true"`,
			`data-studio-block-layout-library="true"`,
			`data-test-library-slot="true"`,
		} {
			if !strings.Contains(html, fragment) {
				t.Fatalf("block-layout engine override missing %q:\n%s", fragment, html)
			}
		}
		for _, blocked := range []string{
			`id="gosx-studio-block-layout-engine"`,
			`data-gosx-engine="GoSXStudioBlockLayout"`,
		} {
			if strings.Contains(html, blocked) {
				t.Fatalf("block-layout engine should use explicit host node instead of fallback %q:\n%s", blocked, html)
			}
		}
	})
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

func TestRenderBlockLayoutEngineHostOwnsRuntimeHostContract(t *testing.T) {
	host := BlockLayoutEngineHostFromMap(map[string]any{
		"key":          " blocks ",
		"name":         "LayoutEngine",
		"mountId":      "layout-engine",
		"class":        "layout-host",
		"capabilities": " pointer keyboard text-input ",
	})
	html := gosx.RenderHTML(RenderBlockLayoutEngineHost(BlockLayoutEngineOptions{
		EngineHost:   host,
		EngineSource: "host-app",
	}))

	for _, fragment := range []string{
		`id="layout-engine"`,
		`class="layout-host"`,
		`data-gosx-engine="LayoutEngine"`,
		`data-gosx-engine-kind="surface"`,
		`data-gosx-studio-engine="blocks"`,
		`data-studio-engine-role="blocks"`,
		`data-studio-engine-source="host-app"`,
		`data-studio-block-layout-engine-surface="true"`,
		`data-gosx-engine-capabilities="pointer keyboard text-input"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("block-layout engine host missing %q:\n%s", fragment, html)
		}
	}
}
