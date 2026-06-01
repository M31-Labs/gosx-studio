package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderSiteMapEngineSegmentsOwnsFrameChrome(t *testing.T) {
	view := map[string]any{
		"kicker":              "Site map",
		"title":               "Pages",
		"summary":             "Editable pages composed from sections.",
		"pageCountLabel":      "2 pages",
		"componentCountLabel": "5 sections",
		"controlCountLabel":   "4 fields",
		"selectedPageLabel":   "Home /",
		"sources": []map[string]any{
			{"key": "host", "label": "Site", "countLabel": "3 sections"},
			{"key": "cms", "label": "Content", "countLabel": "2 sections"},
		},
	}

	segments := RenderSiteMapEngineSegments(view, SiteMapEngineOptions{})
	html := strings.Join([]string{
		gosx.RenderHTML(segments.RootOpen),
		gosx.RenderHTML(segments.Header),
		gosx.RenderHTML(segments.SourceLegend),
		gosx.RenderHTML(segments.RootClose),
	}, "")

	for _, fragment := range []string{
		`class="studio-site-map-engine"`,
		`data-studio-site-map-panel="true"`,
		`data-studio-mode-panel="home"`,
		`data-studio-panel="site-map"`,
		`data-gosx-studio-site-map-engine-renderer="gosx-studio"`,
		`<p class="kicker">Site map</p>`,
		`<h2>Pages</h2>`,
		`<output>2 pages</output>`,
		`<output>Home /</output>`,
		`data-gosx-studio-site-map-source-legend-renderer="gosx-studio"`,
		`data-studio-site-map-source-kind="host"`,
		`<strong>Content</strong>`,
		`</section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("site-map engine segments missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("site-map engine frame must not inject visible platform copy:\n%s", html)
	}
}
