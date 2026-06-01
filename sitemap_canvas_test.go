package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

// siteMapCanvasTestView builds the same AuthoringSiteMapView shape that
// sitemap_board_test.go exercises, so the Canvas2D derivation is validated
// against the exact data the DOM board renders from.
func siteMapCanvasTestView() map[string]any {
	return AuthoringSiteMapView(NoCodeAuthoringSurface(SiteMap{
		Pages: []Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Status:        "Editable",
			Editable:      true,
			Selected:      true,
			Components: []Component{{
				Key:           "hero",
				TemplateKey:   "hero",
				Label:         "Hero",
				GoSXComponent: "HomeHero",
				Source:        ComponentSourceHost,
				Binding:       "home.section.hero",
				Status:        "Visible",
				Editable:      true,
				Visible:       true,
			}},
		}},
	}), SiteMapViewOptions{})
}

func TestSiteMapCanvasNodesDeriveFromWorkspaceLayers(t *testing.T) {
	nodes := siteMapCanvasNodes(siteMapCanvasTestView())

	// The known view yields three workspace nodes:
	//   layer "home":      page:home, component:home:hero
	//   layer "resources": resource:home_section_hero
	// plus two workspace links (contains + binds). Each workspace node maps to
	// one rect (id == node key) and one label; each link maps to one line.
	rects := map[string]gosx.CanvasBoardNode{}
	labels := 0
	lines := 0
	for _, node := range nodes {
		switch node.Kind {
		case "rect":
			rects[node.ID] = node
		case "label":
			labels++
		case "line":
			lines++
		default:
			t.Fatalf("unexpected canvas node kind %q", node.Kind)
		}
	}

	// Resource node keys run the binding through workspaceToken, which
	// collapses non-alphanumeric runs to single dashes: "home.section.hero"
	// becomes "home-section-hero".
	wantRectIDs := []string{
		"page:home",
		"component:home:hero",
		"resource:home-section-hero",
	}
	for _, id := range wantRectIDs {
		rect, ok := rects[id]
		if !ok {
			t.Fatalf("expected rect node for workspace node %q; got rects %v", id, rectKeys(rects))
		}
		if rect.Width <= 0 || rect.Height <= 0 {
			t.Fatalf("rect %q must have positive size, got %vx%v", id, rect.Width, rect.Height)
		}
		if strings.TrimSpace(rect.Color) == "" {
			t.Fatalf("rect %q must carry a per-group color", id)
		}
	}
	if len(rects) != len(wantRectIDs) {
		t.Fatalf("expected %d rect nodes, got %d (%v)", len(wantRectIDs), len(rects), rectKeys(rects))
	}
	if labels != len(wantRectIDs) {
		t.Fatalf("expected one label per workspace node (%d), got %d", len(wantRectIDs), labels)
	}
	if lines != 2 {
		t.Fatalf("expected two link lines (contains + binds), got %d", lines)
	}

	// The page node and component node live in the same layer, so they must
	// not collide on the deterministic grid.
	page := rects["page:home"]
	component := rects["component:home:hero"]
	if page.X == component.X && page.Y == component.Y {
		t.Fatalf("grid layout collided page and component nodes at (%v,%v)", page.X, page.Y)
	}

	// The hero label must be present so the board carries human-readable text.
	if !canvasNodesContainText(nodes, "Hero") {
		t.Fatalf("expected a label node carrying the Hero title; got %#v", nodes)
	}
}

func TestRenderSiteMapCanvasEngineEmitsCanvas2DSurface(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{
		Enabled: true,
		Class:   "host-canvas-board",
	}))

	for _, fragment := range []string{
		`data-studio-site-map-canvas-engine="true"`,
		`data-gosx-surface-kind="canvas2d"`,
		`data-gosx-engine-component="CanvasBoard"`,
		`class="host-canvas-board"`,
		`data-gosx-onpick="handleSiteMapPick"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered site-map canvas engine missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("site-map canvas engine must not inject visible platform copy:\n%s", html)
	}
}

func TestRenderSiteMapCanvasEngineDisabledByDefault(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{}))

	if strings.Contains(html, `data-gosx-surface-kind="canvas2d"`) {
		t.Fatalf("canvas engine must be OFF by default and not emit a canvas2d surface:\n%s", html)
	}
	if !strings.Contains(html, `data-studio-site-map-canvas-engine="disabled"`) {
		t.Fatalf("disabled canvas engine should emit a stable disabled hook:\n%s", html)
	}
}

func rectKeys(rects map[string]gosx.CanvasBoardNode) []string {
	keys := make([]string, 0, len(rects))
	for key := range rects {
		keys = append(keys, key)
	}
	return keys
}

func canvasNodesContainText(nodes []gosx.CanvasBoardNode, text string) bool {
	for _, node := range nodes {
		if strings.TrimSpace(node.Text) == text {
			return true
		}
	}
	return false
}
