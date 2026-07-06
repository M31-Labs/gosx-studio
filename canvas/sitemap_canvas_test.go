package canvas

import (
	"encoding/json"
	"html"
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

type siteMapCanvasBoardPayload struct {
	Nodes []gosx.CanvasBoardNode `json:"nodes"`
}

// siteMapCanvasTestView hand-builds the same "workspaceLayers"/"workspaceLinks"
// map[string]any shape that m31labs.dev/gosx-studio (root package)'s
// AuthoringSiteMapView(NoCodeAuthoringSurface(...), SiteMapViewOptions{})
// produces for one home page with a bound "hero" component (the sitemap
// package's real view-building pipeline, Slice 5 territory, and the
// underlying SiteMap fixture used before this slice moved canvas out — see
// sitemap_board_test.go for the DOM board's equivalent fixture). canvas may
// import only core (spec .tiller/scratch/gosx-studio-restructure-spec-v0.1.md
// §1 Import DAG), so this fixture is a literal reproduction of that pipeline's
// output shape rather than a call through authoring/sitemap. It carries only
// the fields RenderSiteMapCanvasEngine/siteMapCanvasNodes actually read (key,
// kind, group, route, label on nodes; key, fromNodeKey, toNodeKey on links) —
// verified against the real pipeline's JSON output for this exact fixture
// before the move (2 layers: "home" with page:home + component:home:hero,
// "resources" with resource:home-section-hero; 2 links: contains + binds).
func siteMapCanvasTestView() map[string]any {
	return map[string]any{
		"workspaceLayers": []map[string]any{
			{
				"key":   "home",
				"label": "Home",
				"nodes": []map[string]any{
					{
						"key":   "page:home",
						"kind":  "page",
						"group": "site",
						"route": "/",
						"label": "Home",
					},
					{
						"key":   "component:home:hero",
						"kind":  "component",
						"group": "site",
						"route": "/",
						"label": "Hero",
					},
				},
			},
			{
				"key":   "resources",
				"label": "Resources",
				"nodes": []map[string]any{
					{
						"key":   "resource:home-section-hero",
						"kind":  "resource",
						"group": "site",
						"route": "",
						"label": "Hero content",
					},
				},
			},
		},
		"workspaceLinks": []map[string]any{
			{
				"key":         "contains:home:hero",
				"kind":        "contains",
				"fromNodeKey": "page:home",
				"toNodeKey":   "component:home:hero",
			},
			{
				"key":         "binds:home:hero:home-section-hero",
				"kind":        "binds",
				"fromNodeKey": "component:home:hero",
				"toNodeKey":   "resource:home-section-hero",
			},
		},
	}
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
// CanvasBoard/render-bundle path can associate it to the card by world-point
// containment and stack it as the muted subtitle). Non-page nodes
// (component/resource) get NO route label.
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
	// Nodes are emitted +Y-up (flipBoardNodesYUp), so "below" means a SMALLER Y.
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
	if !(route.Y < title.Y) {
		t.Fatalf("route subtitle (y=%v) must sit below the title (y=%v) — smaller Y is lower in +Y-up", route.Y, title.Y)
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

// TestSiteMapCanvasNodesPageArtboardsOnly locks M9-1: with
// options.PageArtboardsOnly=true the shared node builder emits rect + title
// label (+ thumbnail/route subtitle) ONLY for nodes whose kind is
// WorkspaceNodePage, skipping component and resource nodes entirely, AND emits
// ZERO Kind:"line" nodes (no spiderweb). With the flag OFF (default) behavior is
// unchanged: all workspace nodes AND the link lines are present.
func TestSiteMapCanvasNodesPageArtboardsOnly(t *testing.T) {
	view := siteMapCanvasTestView()

	// Flag ON: only the page artboard survives.
	on := siteMapCanvasNodesWith(view, SiteMapCanvasOptions{PageArtboardsOnly: true})
	rectsOn := map[string]gosx.CanvasBoardNode{}
	linesOn := 0
	for _, node := range on {
		switch node.Kind {
		case "rect":
			rectsOn[node.ID] = node
		case "line":
			linesOn++
		}
	}
	if linesOn != 0 {
		t.Fatalf("PageArtboardsOnly must emit zero line nodes (no spiderweb), got %d", linesOn)
	}
	if _, ok := rectsOn["page:home"]; !ok {
		t.Fatalf("PageArtboardsOnly must keep the page rect; got rects %v", rectKeys(rectsOn))
	}
	for _, id := range []string{"component:home:hero", "resource:home-section-hero"} {
		if _, ok := rectsOn[id]; ok {
			t.Fatalf("PageArtboardsOnly must skip non-page node %q; got rects %v", id, rectKeys(rectsOn))
		}
	}
	if len(rectsOn) != 1 {
		t.Fatalf("PageArtboardsOnly must emit exactly one (page) rect, got %d (%v)", len(rectsOn), rectKeys(rectsOn))
	}
	// The page's title + route subtitle labels still compose (no component/resource labels).
	if !canvasNodesContainText(on, "Home") {
		t.Fatalf("PageArtboardsOnly must keep the page title label; got %#v", on)
	}
	if canvasNodesContainText(on, "Hero") {
		t.Fatalf("PageArtboardsOnly must drop the component (Hero) label; got %#v", on)
	}

	// Flag OFF (default): byte-identical to the un-optioned builder.
	off := siteMapCanvasNodesWith(view, SiteMapCanvasOptions{})
	base := siteMapCanvasNodes(view)
	if len(off) != len(base) {
		t.Fatalf("flag OFF must match default builder: got %d nodes, want %d", len(off), len(base))
	}
	for i := range base {
		if off[i] != base[i] {
			t.Fatalf("flag OFF node[%d] diverged from default builder:\n got %#v\nwant %#v", i, off[i], base[i])
		}
	}
	// Sanity: the default still carries the spiderweb (lines) and non-page nodes.
	rectsOff := 0
	linesOff := 0
	for _, node := range off {
		switch node.Kind {
		case "rect":
			rectsOff++
		case "line":
			linesOff++
		}
	}
	if rectsOff != 3 || linesOff != 2 {
		t.Fatalf("flag OFF must keep all 3 rects + 2 lines, got %d rects / %d lines", rectsOff, linesOff)
	}
}

func pointInRect(px, py float64, rect gosx.CanvasBoardNode) bool {
	return px >= rect.X && px <= rect.X+rect.Width &&
		py >= rect.Y && py <= rect.Y+rect.Height
}

// TestSiteMapCanvasNodesEmitThumbnailImageForPageCards locks M8-1: when a page
// card's node key is present in options.Thumbnails with a non-empty value, the
// shared node builder emits an ADDITIONAL "image"-kind node whose Src is the
// thumbnail data URI and whose bounds (X/Y/Width/Height) exactly match the
// page's rect placement. Pages without a thumbnail entry get no image node, and
// thumbnails keyed to non-page nodes are ignored (only page cards get an image).
func TestSiteMapCanvasNodesEmitThumbnailImageForPageCards(t *testing.T) {
	const homeThumb = "data:image/svg+xml;base64,AAAA"
	nodes := siteMapCanvasNodesWith(siteMapCanvasTestView(), SiteMapCanvasOptions{
		Thumbnails: map[string]string{
			// A real page card → must yield an image node at the card bounds.
			"page:home": homeThumb,
			// A non-page node → must be ignored (no image node for it).
			"resource:home-section-hero": "data:image/svg+xml;base64,BBBB",
			// A page that does not exist in this view → no node, no crash.
			"page:missing": "data:image/svg+xml;base64,CCCC",
			// An empty value for a real page → treated as no thumbnail.
			"component:home:hero": "",
		},
	})

	rects := map[string]gosx.CanvasBoardNode{}
	images := map[string]gosx.CanvasBoardNode{}
	for _, node := range nodes {
		switch node.Kind {
		case "rect":
			rects[node.ID] = node
		case "image":
			images[node.ID] = node
		}
	}

	// Exactly one image node, for the single page card carrying a thumbnail.
	if len(images) != 1 {
		t.Fatalf("expected exactly one thumbnail image node, got %d: %#v", len(images), images)
	}

	pageRect, ok := rects["page:home"]
	if !ok {
		t.Fatalf("expected a rect for page:home; got %v", rectKeys(rects))
	}
	thumb, ok := images["page:home:thumb"]
	if !ok {
		t.Fatalf("expected a thumbnail image node id %q; got images %v", "page:home:thumb", rectKeys(images))
	}
	if thumb.Src != homeThumb {
		t.Fatalf("thumbnail Src = %q, want %q", thumb.Src, homeThumb)
	}
	// The image node must occupy the exact card bounds so CanvasBoard/WebGPU can
	// render it as a full-card image overlay.
	if thumb.X != pageRect.X || thumb.Y != pageRect.Y ||
		thumb.Width != pageRect.Width || thumb.Height != pageRect.Height {
		t.Fatalf("thumbnail bounds (%v,%v,%v,%v) must equal page rect bounds (%v,%v,%v,%v)",
			thumb.X, thumb.Y, thumb.Width, thumb.Height,
			pageRect.X, pageRect.Y, pageRect.Width, pageRect.Height)
	}

	// The non-page node with a thumbnail entry must NOT get an image node.
	if _, exists := images["resource:home-section-hero:thumb"]; exists {
		t.Fatalf("non-page node must not receive a thumbnail image node: %#v", images)
	}
}

// TestSiteMapCanvasNodesEmitPageSurfaceForPageCards locks M10-1: when a page
// card's node key is present in options.PageSurfaces with a non-empty value, the
// shared node builder emits an ADDITIONAL "html"-kind node whose ID is the page
// node key (the CanvasBoard surface key), whose Markup is the supplied editable
// HTML, whose PointerEvents is "auto", and whose bounds (X/Y/Width/Height)
// exactly match the page's rect placement. Pages without a PageSurfaces entry
// get no html node, and an empty/nil PageSurfaces map leaves the render
// unchanged.
func TestSiteMapCanvasNodesEmitPageSurfaceForPageCards(t *testing.T) {
	const homeSurface = `<h1 data-gosx-html-key="page:home">hi</h1>`
	nodes := siteMapCanvasNodesWith(siteMapCanvasTestView(), SiteMapCanvasOptions{
		PageSurfaces: map[string]string{
			// A real page card → must yield an html node at the card bounds.
			"page:home": homeSurface,
		},
	})

	rects := map[string]gosx.CanvasBoardNode{}
	htmls := map[string]gosx.CanvasBoardNode{}
	for _, node := range nodes {
		switch node.Kind {
		case "rect":
			rects[node.ID] = node
		case "html":
			htmls[node.ID] = node
		}
	}

	// Exactly one html node, for the single page card carrying a surface.
	if len(htmls) != 1 {
		t.Fatalf("expected exactly one html surface node, got %d: %#v", len(htmls), htmls)
	}

	pageRect, ok := rects["page:home"]
	if !ok {
		t.Fatalf("expected a rect for page:home; got %v", rectKeys(rects))
	}
	// The html node's ID must be the page node key, which CanvasBoard uses as the
	// surface key.
	surface, ok := htmls["page:home"]
	if !ok {
		t.Fatalf("expected an html surface node id %q; got htmls %v", "page:home", rectKeys(htmls))
	}
	if surface.Markup != homeSurface {
		t.Fatalf("surface Markup = %q, want %q", surface.Markup, homeSurface)
	}
	if surface.PointerEvents != "auto" {
		t.Fatalf("surface PointerEvents = %q, want %q", surface.PointerEvents, "auto")
	}
	// The html node must occupy the exact card bounds so the CanvasBoard render
	// bundle can expose it as a full-card live surface.
	if surface.X != pageRect.X || surface.Y != pageRect.Y ||
		surface.Width != pageRect.Width || surface.Height != pageRect.Height {
		t.Fatalf("surface bounds (%v,%v,%v,%v) must equal page rect bounds (%v,%v,%v,%v)",
			surface.X, surface.Y, surface.Width, surface.Height,
			pageRect.X, pageRect.Y, pageRect.Width, pageRect.Height)
	}

	// A page card with NO PageSurfaces entry must NOT get an html node: the test
	// view has exactly one page (home), so once it carries the only entry, any
	// other page key absent from the map must be silently skipped. Re-derive with
	// the home entry pointed at a non-existent page to prove absence yields none.
	absent := siteMapCanvasNodesWith(siteMapCanvasTestView(), SiteMapCanvasOptions{
		PageSurfaces: map[string]string{"page:missing": homeSurface},
	})
	for _, node := range absent {
		if node.Kind == "html" {
			t.Fatalf("page with no PageSurfaces entry must not receive an html node: %#v", node)
		}
	}

	// Empty/nil PageSurfaces (the default) → no html nodes at all (unchanged).
	for _, node := range siteMapCanvasNodesWith(siteMapCanvasTestView(), SiteMapCanvasOptions{}) {
		if node.Kind == "html" {
			t.Fatalf("default (nil PageSurfaces) must not emit any html node: %#v", node)
		}
	}
}

// TestRenderSiteMapCanvasEngineSerializesPageContentNodes locks the default
// CanvasBoard/WebGPU contract: page thumbnails and editable page surfaces must
// be present in the rendered data-gosx-engine-props nodes payload, not only in
// the legacy WASMFree inline bundle.
func TestRenderSiteMapCanvasEngineSerializesPageContentNodes(t *testing.T) {
	const (
		homeThumb   = "data:image/png;base64,HOME"
		homeSurface = `<main data-gosx-html-key="page:home"><button>Save</button></main>`
	)
	htmlOut := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{
		Enabled: true,
		Thumbnails: map[string]string{
			"page:home": homeThumb,
		},
		PageSurfaces: map[string]string{
			"page:home": homeSurface,
		},
	}))

	for _, fragment := range []string{
		`data-gosx-canvas-backend="webgpu"`,
		`data-gosx-engine-component="CanvasBoard"`,
	} {
		if !strings.Contains(htmlOut, fragment) {
			t.Fatalf("rendered default CanvasBoard missing %q:\n%s", fragment, htmlOut)
		}
	}
	if strings.Contains(htmlOut, "data-gosx-canvas-bundle") {
		t.Fatalf("default CanvasBoard path must not emit the WASMFree inline bundle:\n%s", htmlOut)
	}

	payload := parseRenderedCanvasBoardPayload(t, htmlOut)
	rect := findCanvasNode(payload.Nodes, "rect", "page:home")
	if rect == nil {
		t.Fatalf("rendered CanvasBoard props missing page rect node; nodes=%#v", payload.Nodes)
	}
	thumb := findCanvasNode(payload.Nodes, "image", "page:home:thumb")
	if thumb == nil {
		t.Fatalf("rendered CanvasBoard props missing thumbnail image node; nodes=%#v", payload.Nodes)
	}
	if thumb.Src != homeThumb {
		t.Fatalf("thumbnail Src = %q, want %q", thumb.Src, homeThumb)
	}
	assertCanvasNodeBoundsEqual(t, "thumbnail", *thumb, *rect)

	surface := findCanvasNode(payload.Nodes, "html", "page:home")
	if surface == nil {
		t.Fatalf("rendered CanvasBoard props missing page html surface node; nodes=%#v", payload.Nodes)
	}
	if surface.Markup != homeSurface {
		t.Fatalf("surface Markup = %q, want %q", surface.Markup, homeSurface)
	}
	if surface.PointerEvents != "auto" {
		t.Fatalf("surface PointerEvents = %q, want %q", surface.PointerEvents, "auto")
	}
	assertCanvasNodeBoundsEqual(t, "html surface", *surface, *rect)
}

func TestRenderSiteMapCanvasEngineEmitsCanvas2DSurface(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{
		Enabled: true,
		Class:   "host-canvas-board",
	}))

	for _, fragment := range []string{
		`data-studio-site-map-canvas-engine="true"`,
		`data-gosx-surface-kind="canvas2d"`,
		`data-gosx-canvas-backend="webgpu"`,
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
	if strings.Contains(html, "data-gosx-canvas-bundle") {
		t.Fatalf("default enabled site-map canvas engine must not emit the WASM-free inline bundle:\n%s", html)
	}
}

// TestRenderSiteMapCanvasEngineWASMFreeOmitsSurfaceKind locks the WASM-free
// decoupling: with WASMFree=true the engine emits a host-rendered <canvas> that
// carries data-gosx-canvas-wasm-free but DELIBERATELY OMITS data-gosx-surface-kind
// (and the data-gosx-engine-* hydration attributes), so the gosx client
// bootstrap's canvas discovery never WASM-hydrates it. The SAME server-precomputed
// inline RenderBundle must still ship next to the canvas so the Studio-owned
// WASM-free client can paint it.
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
		`data-gosx-canvas-backend="webgpu"`,
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
	if strings.Contains(html, `<canvas`) {
		t.Fatalf("disabled canvas engine must stay inert and emit no canvas:\n%s", html)
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

func parseRenderedCanvasBoardPayload(t *testing.T, htmlOut string) siteMapCanvasBoardPayload {
	t.Helper()
	propsJSON := extractRenderedAttr(htmlOut, "data-gosx-engine-props")
	if propsJSON == "" {
		t.Fatalf("data-gosx-engine-props missing or empty:\n%s", htmlOut)
	}
	var payload siteMapCanvasBoardPayload
	if err := json.Unmarshal([]byte(propsJSON), &payload); err != nil {
		t.Fatalf("data-gosx-engine-props JSON did not parse: %v\nJSON: %s", err, propsJSON)
	}
	return payload
}

func extractRenderedAttr(htmlOut, name string) string {
	prefix := name + `="`
	i := strings.Index(htmlOut, prefix)
	if i < 0 {
		return ""
	}
	start := i + len(prefix)
	end := strings.Index(htmlOut[start:], `"`)
	if end < 0 {
		return ""
	}
	return html.UnescapeString(htmlOut[start : start+end])
}

func findCanvasNode(nodes []gosx.CanvasBoardNode, kind, id string) *gosx.CanvasBoardNode {
	for i := range nodes {
		if nodes[i].Kind == kind && nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func assertCanvasNodeBoundsEqual(t *testing.T, name string, got, want gosx.CanvasBoardNode) {
	t.Helper()
	if got.X != want.X || got.Y != want.Y || got.Width != want.Width || got.Height != want.Height {
		t.Fatalf("%s bounds (%v,%v,%v,%v) must equal page rect bounds (%v,%v,%v,%v)",
			name, got.X, got.Y, got.Width, got.Height,
			want.X, want.Y, want.Width, want.Height)
	}
}
