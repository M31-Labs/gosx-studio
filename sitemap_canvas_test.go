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
	// one rect (id == node key) and one title label; each link maps to one line.
	// Each PAGE node with a route additionally maps to one route subtitle label,
	// so the only routed page (page:home → "/") contributes a second label.
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
	// One title label per workspace node, plus one route subtitle label for the
	// single routed page node (page:home → "/").
	const routedPageCards = 1
	wantLabels := len(wantRectIDs) + routedPageCards
	if labels != wantLabels {
		t.Fatalf("expected one title label per workspace node plus %d route subtitle label(s) (%d), got %d", routedPageCards, wantLabels, labels)
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

// TestSiteMapCanvasNodesEmitRouteSubtitleForPageCards locks M7-3: each PAGE node
// that carries a route gets an ADDITIONAL label node whose Text is the route and
// whose point sits below the title label but inside the page rect's bounds (so the
// muddy painter associates it to the card by world-point containment and stacks it
// as the muted subtitle). Non-page nodes (component/resource) get NO route label.
func TestSiteMapCanvasNodesEmitRouteSubtitleForPageCards(t *testing.T) {
	nodes := siteMapCanvasNodes(siteMapCanvasTestView())

	// Collect rects by ID and group labels by which rect bounds contain their point.
	rects := map[string]gosx.CanvasBoardNode{}
	var labels []gosx.CanvasBoardNode
	for _, node := range nodes {
		switch node.Kind {
		case "rect":
			rects[node.ID] = node
		case "label":
			labels = append(labels, node)
		}
	}

	pageRect, ok := rects["page:home"]
	if !ok {
		t.Fatalf("expected a rect for page:home; got %v", rectKeys(rects))
	}

	// Labels whose point falls inside the page:home rect bounds.
	var inPage []gosx.CanvasBoardNode
	for _, label := range labels {
		if pointInRect(label.X, label.Y, pageRect) {
			inPage = append(inPage, label)
		}
	}
	if len(inPage) != 2 {
		t.Fatalf("expected page:home card to carry a title + route subtitle label (2), got %d: %#v", len(inPage), inPage)
	}

	// The title ("Home") and the route subtitle ("/") must both be present, with
	// the route label strictly below the title and still inside the rect bounds.
	var title, route *gosx.CanvasBoardNode
	for i := range inPage {
		switch strings.TrimSpace(inPage[i].Text) {
		case "Home":
			title = &inPage[i]
		case "/":
			route = &inPage[i]
		}
	}
	if title == nil {
		t.Fatalf("expected the page title label %q inside the card; got %#v", "Home", inPage)
	}
	if route == nil {
		t.Fatalf("expected the route subtitle label %q inside the card; got %#v", "/", inPage)
	}
	if !(route.Y > title.Y) {
		t.Fatalf("route subtitle (y=%v) must sit below the title (y=%v)", route.Y, title.Y)
	}
	if !pointInRect(route.X, route.Y, pageRect) {
		t.Fatalf("route subtitle point (%v,%v) must be inside page rect bounds (%v,%v,%v,%v)",
			route.X, route.Y, pageRect.X, pageRect.Y, pageRect.Width, pageRect.Height)
	}
	if route.ID == title.ID {
		t.Fatalf("route subtitle label must carry a distinct ID from the title (%q)", title.ID)
	}

	// Non-page nodes (component, resource) must NOT receive a route subtitle: no
	// label node may carry a route-looking Text inside their bounds beyond the
	// single title label.
	for _, id := range []string{"component:home:hero", "resource:home-section-hero"} {
		rect, ok := rects[id]
		if !ok {
			t.Fatalf("expected a rect for %q; got %v", id, rectKeys(rects))
		}
		count := 0
		for _, label := range labels {
			if pointInRect(label.X, label.Y, rect) {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("non-page node %q must carry exactly one (title) label, got %d", id, count)
		}
	}
}

func pointInRect(px, py float64, rect gosx.CanvasBoardNode) bool {
	return px >= rect.X && px <= rect.X+rect.Width &&
		py >= rect.Y && py <= rect.Y+rect.Height
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

// TestRenderSiteMapCanvasEngineWASMFreeOmitsSurfaceKind locks the WASM-free
// decoupling: with WASMFree=true the engine emits a muddy-owned <canvas> that
// carries data-gosx-canvas-wasm-free but DELIBERATELY OMITS data-gosx-surface-kind
// (and the data-gosx-engine-* hydration attributes), so the gosx client
// bootstrap's canvas discovery never WASM-hydrates it. The SAME server-precomputed
// inline RenderBundle must still ship next to the canvas so a WASM-free client can
// paint it.
func TestRenderSiteMapCanvasEngineWASMFreeOmitsSurfaceKind(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{
		Enabled:  true,
		WASMFree: true,
		Class:    "host-canvas-board",
	}))

	// The decoupling markers must be present.
	for _, fragment := range []string{
		`data-studio-site-map-canvas-engine="true"`,
		`data-gosx-canvas-wasm-free="true"`,
		`id="studio-site-map-canvas-board"`,
		// The inline bundle still ships for the WASM-free painter.
		`data-gosx-canvas-bundle="studio-site-map-canvas-board"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("wasm-free site-map canvas engine missing %q:\n%s", fragment, html)
		}
	}
	// The WASM hydration attributes must be ABSENT (the gosx bootstrap discovers
	// `[data-gosx-surface-kind]:not([data-gosx-engine-bytecode])`; omitting the
	// surface-kind is what keeps this canvas out of the WASM hydration path).
	for _, forbidden := range []string{
		`data-gosx-surface-kind="canvas2d"`,
		`data-gosx-engine-component="CanvasBoard"`,
		`data-gosx-canvas2d="1"`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("wasm-free site-map canvas engine must NOT carry %q (so the gosx bootstrap skips it):\n%s", forbidden, html)
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
