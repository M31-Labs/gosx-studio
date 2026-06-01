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

func TestRenderSiteMapEngineComposesVisibleSurfaceAndExternalForms(t *testing.T) {
	view := map[string]any{
		"kicker":              "Site map",
		"title":               "Pages",
		"summary":             "Editable pages composed from sections.",
		"pageCountLabel":      "2 pages",
		"componentCountLabel": "5 sections",
		"controlCountLabel":   "4 fields",
		"selectedPageLabel":   "Home /",
		"hasPages":            true,
		"hasMetadataPage":     true,
		"metadataPage": map[string]any{
			"key":                "home",
			"groupLabel":         "Site",
			"statusLabel":        "Editable",
			"authoringPageLabel": "Home",
			"authoringPageRoute": "/",
		},
		"sources": []map[string]any{
			{"key": "host", "label": "Site", "countLabel": "3 sections"},
		},
		"compositionIntents": []map[string]any{
			{
				"key":         "create-page:landing",
				"kind":        "create-page",
				"formID":      "studioSiteMapIntentForm-create-page-landing",
				"label":       "Landing page",
				"summary":     "Create a reusable landing page.",
				"actionLabel": "Create page",
			},
		},
		"hasCompositionIntents": true,
	}

	engine := RenderSiteMapEngine(view, SiteMapEngineOptions{
		EngineHost: SiteMapEngineHostFromMap(map[string]any{
			"key":          "site-map",
			"name":         SiteMapEngineName,
			"mountId":      "gosx-studio-site-map-engine",
			"class":        "studio-site-map-engine-host",
			"capabilities": "canvas pointer keyboard text-input animation",
		}),
		AuthoringForms: SiteMapAuthoringFormsOptions{
			Action:    "/admin/editor/__actions/authoring",
			CSRFToken: "csrf-token",
		},
	})
	surfaceHTML := gosx.RenderHTML(engine.Surface)
	formsHTML := gosx.RenderHTML(engine.Forms)

	for _, fragment := range []string{
		`data-gosx-studio-site-map-engine-renderer="gosx-studio"`,
		`data-gosx-studio-authoring-panel-renderer="gosx-studio"`,
		`data-gosx-studio-site-map-source-legend-renderer="gosx-studio"`,
		`id="gosx-studio-site-map-engine"`,
		`data-gosx-engine="GoSXStudioSiteMap"`,
		`data-studio-site-map-engine-surface="true"`,
		`data-gosx-studio-site-map-board-renderer="gosx-studio"`,
		`</section>`,
	} {
		if !strings.Contains(surfaceHTML, fragment) {
			t.Fatalf("site-map engine surface missing %q:\n%s", fragment, surfaceHTML)
		}
	}
	for _, fragment := range []string{
		`data-gosx-studio-authoring-forms-renderer="gosx-studio"`,
		`data-studio-composition-intent-forms="true"`,
		`id="studioSiteMapIntentForm-create-page-landing"`,
		`action="/admin/editor/__actions/authoring"`,
		`name="csrf_token" value="csrf-token"`,
	} {
		if !strings.Contains(formsHTML, fragment) {
			t.Fatalf("site-map engine forms missing %q:\n%s", fragment, formsHTML)
		}
	}
	if strings.Contains(surfaceHTML, "<form") {
		t.Fatalf("site-map engine visible surface must keep managed forms outside the workbench form:\n%s", surfaceHTML)
	}
	if strings.Contains(surfaceHTML, "GoSX Studio") || strings.Contains(formsHTML, "GoSX Studio") {
		t.Fatalf("site-map engine render must not inject visible platform copy:\n%s\n%s", surfaceHTML, formsHTML)
	}
}
