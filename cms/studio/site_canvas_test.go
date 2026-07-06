package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderSiteCanvasMapsWebsiteObjects(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteCanvas(SiteCanvasOptions{
		Class:         "studio-site-map",
		ControlsClass: "studio-site-map__controls",
		NodeClass:     "studio-site-map__node",
		Kicker:        "Spatial authoring",
		Title:         "Website map",
		Summary:       "Pages, flows, content, and style",
		Nodes: []SiteCanvasNode{
			{Key: "home", Kind: "page", Label: "Home", Summary: "Landing page", Href: "/", X: 120, Y: 120, Selected: true, Metrics: []Metric{NewMetric("sections", "sections", 6)}},
			{Key: "checkout", Kind: "flow", Label: "Checkout", Summary: "Purchase path", Href: "/shop", X: 480, Y: 180},
			{Key: "theme", Kind: "style", Label: "Theme", Summary: "Palette and type", X: 120, Y: 360},
		},
		Edges: []SiteCanvasEdge{
			{From: "home", To: "checkout", Kind: "journey", Label: "Buy path"},
			{From: "theme", To: "home", Kind: "style"},
		},
	}))

	for _, want := range []string{
		`data-gosx-studio-site-canvas="true"`,
		`data-gosx-studio-canvas-viewport="true"`,
		`data-gosx-studio-canvas-surface="true"`,
		`data-gosx-studio-canvas-keyboard-nudge="8"`,
		`data-gosx-studio-canvas-node="home"`,
		`data-gosx-studio-canvas-node-kind="style"`,
		`data-gosx-studio-canvas-edge-from="home"`,
		`data-gosx-studio-canvas-edge-to="checkout"`,
		`class="studio-site-map__controls"`,
		`studio-site-map__node--page`,
		`studio-site-map__node-kind`,
		`studio-site-map__node-metrics`,
		`aria-pressed="true"`,
		`Website map`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in site canvas markup: %s", want, html)
		}
	}
}

func TestRenderSiteCanvasCanPersistNodePositions(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteCanvas(SiteCanvasOptions{
		PositionInputPrefix:  "pajaritosCanvas",
		PersistNodePositions: true,
		Nodes: []SiteCanvasNode{
			{Key: "home", Kind: "page", Label: "Home", X: 80, Y: 120},
			{Key: "flow-schedule-tour", Kind: "flow", Label: "Schedule tour", X: 400, Y: 160},
		},
	}))

	for _, want := range []string{
		`name="pajaritosCanvasHomeX" value="80"`,
		`name="pajaritosCanvasHomeY" value="120"`,
		`name="pajaritosCanvasFlowScheduleTourX" value="400"`,
		`data-gosx-studio-canvas-node-position="true"`,
		`data-gosx-studio-canvas-node-position-key="flow-schedule-tour"`,
		`data-gosx-studio-canvas-node-position-axis="y"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in persisted site canvas markup: %s", want, html)
		}
	}
}

func TestSiteCanvasPositionsFromForm(t *testing.T) {
	nodes := []SiteCanvasNode{
		{Key: "home", Label: "Home", X: 80, Y: 120},
		{Key: "flow-schedule-tour", Label: "Schedule tour", X: 400, Y: 160},
	}
	positions, err := SiteCanvasPositionsFromForm(map[string]string{
		SiteCanvasPositionInputName("pajaritosCanvas", "home", "x"):               "144.5",
		SiteCanvasPositionInputName("pajaritosCanvas", "home", "y"):               "180",
		SiteCanvasPositionInputName("pajaritosCanvas", "flow-schedule-tour", "x"): "520",
	}, nodes, SiteCanvasPositionFormOptions{NamePrefix: "pajaritosCanvas"})
	if err != nil {
		t.Fatalf("positions from form: %v", err)
	}
	if positions["home"].X != 144.5 || positions["home"].Y != 180 {
		t.Fatalf("unexpected home position: %#v", positions["home"])
	}
	if positions["flow-schedule-tour"].X != 520 || positions["flow-schedule-tour"].Y != 160 {
		t.Fatalf("unexpected partial flow position: %#v", positions["flow-schedule-tour"])
	}

	applied := ApplySiteCanvasPositions(nodes, positions)
	if applied[0].X != 144.5 || applied[1].X != 520 {
		t.Fatalf("expected positions applied to nodes: %#v", applied)
	}
}

func TestSiteCanvasPositionsFromFormRejectsInvalidNumbers(t *testing.T) {
	_, err := SiteCanvasPositionsFromForm(map[string]string{
		SiteCanvasPositionInputName("", "home", "x"): "nope",
	}, []SiteCanvasNode{{Key: "home", Label: "Home"}}, SiteCanvasPositionFormOptions{})
	if err == nil {
		t.Fatal("expected invalid position error")
	}
}
