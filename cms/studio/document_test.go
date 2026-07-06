package studio

import "testing"

func TestNormalizeStudioDocumentBuildsCanonicalViewMaps(t *testing.T) {
	document := NormalizeStudioDocument(StudioDocument{
		Label: "  Commerce site  ",
		Pages: []StudioPage{
			{
				Key:         " Home Page ",
				Label:       " Home ",
				Path:        " / ",
				Status:      ReadinessReady,
				ContentKeys: []string{"Hero Copy", "hero-copy", " "},
				StyleKeys:   []string{"Theme"},
				FlowKeys:    []string{"Checkout"},
				ReleaseKeys: []string{"Spring Launch"},
				Tags:        []string{"public", "public", " "},
			},
			{Key: "home-page", Label: "Duplicate"},
		},
		Content: []StudioContent{{Key: "Hero Copy", Label: " Hero copy ", Kind: "Rich Text", Status: ReadinessNext}},
		Styles:  []StudioStyle{{Key: "Theme", Label: "Theme", Tokens: map[string]string{" color.primary ": " #111 ", "empty": " "}}},
		Flows:   []StudioFlow{{Key: "Checkout", Label: "Checkout", StepCount: -1, Executable: true}},
		Releases: []StudioRelease{{
			Key:    "Spring Launch",
			Label:  "Spring launch",
			Status: ReadinessReady,
		}},
		Edges: []StudioDocumentEdge{
			{From: "Theme", To: "missing", Kind: "style"},
			{From: "Checkout", To: "Spring Launch", Kind: "release", Label: "Included"},
		},
	})

	if document.Key != "commerce-site" || document.Label != "Commerce site" {
		t.Fatalf("unexpected document identity: %#v", document)
	}
	if len(document.Pages) != 1 {
		t.Fatalf("expected duplicate pages to be removed, got %#v", document.Pages)
	}
	page := document.Pages[0]
	if page.Key != "home-page" || page.Path != "/" || page.ContentKeys[0] != "hero-copy" || len(page.ContentKeys) != 1 {
		t.Fatalf("unexpected normalized page: %#v", page)
	}
	if len(page.Tags) != 1 || page.Tags[0] != "public" {
		t.Fatalf("unexpected normalized tags: %#v", page.Tags)
	}
	if document.Content[0].Kind != "rich-text" || document.Flows[0].StepCount != 0 {
		t.Fatalf("unexpected content or flow normalization: %#v %#v", document.Content[0], document.Flows[0])
	}
	if got := document.Styles[0].Tokens["color.primary"]; got != "#111" || len(document.Styles[0].Tokens) != 1 {
		t.Fatalf("unexpected normalized style tokens: %#v", document.Styles[0].Tokens)
	}

	maps := document.ViewMaps()
	if maps.Pages["home-page"].Label != "Home" || maps.Content["hero-copy"].Label != "Hero copy" {
		t.Fatalf("view maps did not expose typed records: %#v", maps)
	}
	if maps.Nodes["theme"].Kind != StudioDocumentNodeStyle || maps.Nodes["spring-launch"].Kind != StudioDocumentNodeRelease {
		t.Fatalf("view maps did not expose canonical nodes: %#v", maps.Nodes)
	}
	if _, ok := maps.Edges["theme-style-missing"]; ok {
		t.Fatalf("expected edge to missing node to be dropped: %#v", maps.Edges)
	}
	if maps.Edges["checkout-release-spring-launch"].Label != "Included" {
		t.Fatalf("expected explicit valid edge in view map: %#v", maps.Edges)
	}
}

func TestStudioDocumentSiteCanvasIncludesInferredAndExplicitEdges(t *testing.T) {
	document := StudioDocument{
		Label:   "Site model",
		Summary: "Pages, content, style, flows, and releases",
		Pages: []StudioPage{{
			Key:         "home",
			Label:       "Home",
			Path:        "/",
			Status:      ReadinessReady,
			ContentKeys: []string{"hero"},
			StyleKeys:   []string{"theme"},
			FlowKeys:    []string{"checkout"},
			ReleaseKeys: []string{"launch"},
			View:        StudioDocumentNodeView{X: 12, Y: 34, Width: 300, Height: 140, Selected: true},
			Metrics:     []Metric{NewMetric("sections", "sections", 6)},
		}},
		Content:  []StudioContent{{Key: "hero", Label: "Hero copy", Summary: "Landing content"}},
		Styles:   []StudioStyle{{Key: "theme", Label: "Theme"}},
		Flows:    []StudioFlow{{Key: "checkout", Label: "Checkout"}},
		Releases: []StudioRelease{{Key: "launch", Label: "Launch"}},
		Edges:    []StudioDocumentEdge{{From: "checkout", To: "launch", Kind: "release", Label: "Gates"}},
	}

	nodes, edges := document.SiteCanvas()
	if len(nodes) != 5 {
		t.Fatalf("expected all document concepts as canvas nodes, got %#v", nodes)
	}
	if nodes[0].Key != "home" || nodes[0].Kind != "page" || nodes[0].Status != "Ready" || !nodes[0].Selected {
		t.Fatalf("unexpected page canvas node: %#v", nodes[0])
	}
	if nodes[0].Href != "/" || nodes[0].X != 12 || nodes[0].Width != 300 || len(nodes[0].Metrics) != 1 {
		t.Fatalf("expected page view, href, and metrics to map to canvas node: %#v", nodes[0])
	}

	wantEdges := map[string]string{
		"home-content-hero":       "content",
		"theme-style-home":        "style",
		"home-flow-checkout":      "flow",
		"home-release-launch":     "release",
		"checkout-release-launch": "release",
	}
	if len(edges) != len(wantEdges) {
		t.Fatalf("unexpected canvas edges: %#v", edges)
	}
	for _, edge := range edges {
		if wantEdges[edge.Key] != edge.Kind {
			t.Fatalf("unexpected canvas edge: %#v in %#v", edge, edges)
		}
	}

	options := document.SiteCanvasOptions()
	if options.Title != "Site model" || options.Summary != "Pages, content, style, flows, and releases" || len(options.Nodes) != 5 || len(options.Edges) != len(wantEdges) {
		t.Fatalf("unexpected canvas options: %#v", options)
	}
}
