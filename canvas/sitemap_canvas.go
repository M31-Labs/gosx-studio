// Package canvas owns the page-canvas engine hosting and server-rendered
// surface markup: artboards, thumbnails, and the block-layout DOM contract.
//
// This package moved here from the studio root in Slice 4 of the package
// restructure (see .tiller/scratch/gosx-studio-restructure-spec-v0.1.md). Per
// the DAG in that spec's §1, canvas may import only core: the
// StudioEngineRuntime/BlockLayoutEngineRuntime engine-runtime interfaces stay
// consumer-side via structural typing (so canvas never imports hostruntime),
// and a handful of small map-reading/engine-capability helpers that are still
// unexported duplicates in not-yet-migrated root files are carried locally in
// support.go rather than importing the sitemap/shell packages those files are
// destined for (see support.go's doc comment).
package canvas

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/render/bundle2d"

	"m31labs.dev/gosx-studio/core"
)

// SiteMapCanvasOptions configures the opt-in Canvas2D site-map surface.
//
// This is a parallel, additive render path to RenderSiteMapBoard. It renders
// the same AuthoringSiteMapView page graph onto the merged Phase-2
// gosx.CanvasBoard primitive (surface kind "canvas2d") instead of the DOM
// board. It is feature-flagged and OFF by default: callers must set
// Enabled=true to mount the canvas surface. When disabled, the renderer emits
// only a stable, inert container hook and never emits a <canvas> element, so
// the existing DOM board stays the default editor experience.
type SiteMapCanvasOptions struct {
	// Enabled opts the host into the Canvas2D surface. When false (the
	// default) RenderSiteMapCanvasEngine emits a disabled placeholder only.
	Enabled bool

	// Class overrides the host-facing container CSS class. The renderer never
	// injects visible platform copy, so hosts style this surface freely.
	Class string

	// CanvasClass forwards to the underlying <canvas> element class.
	CanvasClass string

	// Background is the canvas CSS background color. Defaults to a neutral
	// dark board surface when blank.
	Background string

	// Width/Height set the canvas size in CSS pixels. Non-positive values fall
	// back to the CanvasBoard defaults (1280x720).
	Width  int
	Height int

	// ColumnStride/RowStride control the deterministic grid layout spacing in
	// world units. Non-positive values use sensible defaults.
	ColumnStride float64
	RowStride    float64

	// NodeWidth/NodeHeight size each rect node in world units. Non-positive
	// values use sensible defaults.
	NodeWidth  float64
	NodeHeight float64

	// OnPick is the Go handler name wired to CanvasBoard pick events. Defaults
	// to "handleSiteMapPick" when blank.
	OnPick string

	// Thumbnails maps a page card's workspace node key (e.g. "page:home") to a
	// per-page thumbnail image/texture URI. When a page card's node key is
	// present here with a non-empty value, the shared node builder emits an
	// ADDITIONAL Kind:"image" node (ID = key+":thumb", Src = the URI) at the
	// card's exact placement bounds. In the default path that node serializes
	// into gosx.CanvasBoard's data-gosx-engine-props payload and is consumed by
	// the CanvasBoard/WebGPU render path as an image/sprite node; the explicit
	// WASMFree compatibility path also carries it through the legacy inline
	// bundle. Pages with no entry (or an empty value) get no image node, and
	// entries keyed to non-page nodes are ignored. Empty/nil by default, so the
	// standard site-map render is unchanged.
	Thumbnails map[string]string

	// PageSurfaces maps a page card's workspace node key (e.g. "page:home") to
	// the host-supplied full editable HTML markup for that page. When a page
	// card's node key is present here with a non-empty value, the shared node
	// builder emits an ADDITIONAL Kind:"html" node (ID = the page node key;
	// Markup = the supplied HTML; PointerEvents = "auto") at the card's exact
	// placement bounds. In the default path that html node serializes into
	// gosx.CanvasBoard's data-gosx-engine-props payload and becomes a scene/html
	// DOM overlay in the CanvasBoard render bundle; the explicit WASMFree
	// compatibility path carries it through the legacy inline bundle. The html
	// node is appended AFTER thumbnail images so live surfaces win ordering at
	// near zoom. Pages with no entry (or an empty value) get no html node, and
	// entries keyed to non-page nodes are ignored. Empty/nil by default, so the
	// standard site-map render is unchanged.
	PageSurfaces map[string]string

	// ExtraNodes are appended verbatim to the deterministically-laid-out
	// site-map nodes before the bundle is computed, so a host can place
	// additional CanvasBoardNodes (e.g. a Kind:"html" surface) onto the same
	// board. In the default path they serialize into the same gosx.CanvasBoard
	// props payload as the site-map rects/labels/lines/images/html surfaces; in
	// the explicit WASMFree compatibility path they also flow into the legacy
	// inline RenderBundle. Empty by default, so the standard site-map render is
	// unchanged.
	ExtraNodes []gosx.CanvasBoardNode

	// PageArtboardsOnly renders the canvas as page artboards only: when true the
	// shared node builder emits rect + title label (+ route subtitle + thumbnail)
	// ONLY for workspace nodes whose kind is WorkspaceNodePage, skipping
	// component and resource nodes entirely, AND emits NO Kind:"line" nodes (the
	// link "spiderweb" between nodes is dropped). When false (the default) the
	// full authoring graph renders — all workspace nodes plus the connecting
	// lines — exactly as before, so existing callers are unchanged.
	PageArtboardsOnly bool

	// WASMFree opts the surface into the legacy WASM-free compatibility path.
	// When true the renderer emits a plain <canvas> that DELIBERATELY carries NO
	// data-gosx-surface-kind (and instead carries data-gosx-canvas-wasm-free),
	// plus the server-precomputed inline RenderBundle used by that JS painter.
	// When false (the default) the surface emits the real gosx.CanvasBoard marked
	// for the WebGPU backend.
	WASMFree bool
}

const (
	siteMapCanvasDefaultColumnStride = 280.0
	siteMapCanvasDefaultRowStride    = 120.0
	siteMapCanvasDefaultNodeWidth    = 200.0
	siteMapCanvasDefaultNodeHeight   = 72.0
	siteMapCanvasDefaultBackground   = "#0f1720"
	siteMapCanvasLabelInsetX         = 14.0
	siteMapCanvasLabelInsetY         = 28.0
	// siteMapCanvasRouteInsetY stacks the page route subtitle a row below the
	// title (title at +28, route at +48), staying inside the default 72-tall rect
	// so CanvasBoard/render-bundle consumers can associate it to the card by
	// world-point containment and render it as the muted subtitle.
	siteMapCanvasRouteInsetY   = 48.0
	siteMapCanvasDefaultOnPick = "handleSiteMapPick"
)

// siteMapCanvasLayout captures the resolved grid spacing for node placement.
type siteMapCanvasLayout struct {
	columnStride float64
	rowStride    float64
	nodeWidth    float64
	nodeHeight   float64
}

func siteMapCanvasLayoutFrom(options SiteMapCanvasOptions) siteMapCanvasLayout {
	return siteMapCanvasLayout{
		columnStride: positiveOr(options.ColumnStride, siteMapCanvasDefaultColumnStride),
		rowStride:    positiveOr(options.RowStride, siteMapCanvasDefaultRowStride),
		nodeWidth:    positiveOr(options.NodeWidth, siteMapCanvasDefaultNodeWidth),
		nodeHeight:   positiveOr(options.NodeHeight, siteMapCanvasDefaultNodeHeight),
	}
}

// siteMapCanvasNodePlacement records the resolved rect bounds of a workspace
// node so links can be drawn between node centers.
type siteMapCanvasNodePlacement struct {
	x      float64
	y      float64
	width  float64
	height float64
}

func (p siteMapCanvasNodePlacement) centerX() float64 { return p.x + p.width/2 }
func (p siteMapCanvasNodePlacement) centerY() float64 { return p.y + p.height/2 }

// RenderSiteMapCanvasEngine renders the AuthoringSiteMapView page graph onto
// the Canvas2D gosx.CanvasBoard primitive as an opt-in alternative to the DOM
// site-map board. It is OFF by default: when options.Enabled is false it emits
// only an inert container hook and no canvas. The existing RenderSiteMapBoard
// DOM path is unaffected by this renderer.
func RenderSiteMapCanvasEngine(siteMapView map[string]any, options SiteMapCanvasOptions) gosx.Node {
	if !options.Enabled {
		return gosx.El("section", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-site-map-canvas")),
			gosx.Attr("data-studio-site-map-canvas-engine", "disabled"),
			gosx.Attr("hidden", true),
			gosx.Attr("aria-hidden", "true"),
		))
	}

	// Build the board nodes once and feed them to either the real CanvasBoard or
	// the explicit WASM-free compatibility bundle.
	nodes := siteMapCanvasNodesWith(siteMapView, options)
	if len(options.ExtraNodes) > 0 {
		// Appended last so host-supplied nodes (e.g. a Kind:"html" surface)
		// paint on top of the site-map graph and serialize through the same
		// bundle path.
		nodes = append(nodes, options.ExtraNodes...)
	}
	background := core.FirstNonEmpty(options.Background, siteMapCanvasDefaultBackground)

	// In the WASM-free path we emit a plain <canvas> that carries NO
	// data-gosx-surface-kind, so the gosx client bootstrap never WASM-hydrates it.
	// Otherwise we emit the real gosx.CanvasBoard and mark it for WebGPU.
	var board gosx.Node
	if options.WASMFree {
		board = siteMapCanvasWASMFreeCanvas(background, options.Width, options.Height, options.CanvasClass)
	} else {
		board = gosx.CanvasBoard(gosx.CanvasBoardProps{
			ID:         "studio-site-map-canvas-board",
			Width:      options.Width,
			Height:     options.Height,
			Background: background,
			Backend:    gosx.CanvasBoardBackendWebGPU,
			Zoom:       1.0,
			Nodes:      nodes,
			OnPick:     core.FirstNonEmpty(options.OnPick, siteMapCanvasDefaultOnPick),
			ClassName:  options.CanvasClass,
		})
	}

	// gosx.El takes ...any (AttrList + Node children), so children carry as []any.
	sectionAttrs := []any{
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-site-map-canvas")),
		gosx.Attr("data-studio-site-map-canvas-engine", "true"),
		gosx.Attr("data-gosx-studio-site-map-canvas-renderer", "gosx-studio"),
		gosx.Attr("aria-label", "Site map canvas surface"),
	}
	if options.WASMFree {
		// Mark the surface section so the host (and e2e) can confirm the
		// WASM-free render path owns this board.
		sectionAttrs = append(sectionAttrs, gosx.Attr("data-gosx-canvas-wasm-free", "true"))
	}
	children := []any{
		gosx.Attrs(sectionAttrs...),
		board,
	}
	if options.WASMFree {
		if bundleScript, ok := siteMapCanvasBundleScript(nodes, background, options.Width, options.Height); ok {
			// The explicit WASM-free client reads this server-side bundle after
			// locating the preceding canvas.
			children = append(children, bundleScript)
		}
	}

	return gosx.El("section", children...)
}

// siteMapCanvasWASMFreeCanvas emits the WASM-free site-map <canvas>. It carries
// the SAME id and size the WASM gosx.CanvasBoard would, but DELIBERATELY OMITS
// data-gosx-surface-kind (and the other data-gosx-engine-* / data-gosx-canvas2d
// attributes the WASM path uses for hydration). That omission is the
// decoupling: the gosx client bootstrap discovers canvas surfaces via
// `[data-gosx-surface-kind]:not([data-gosx-engine-bytecode])`
// (client/js/bootstrap-src/26b-feature-engines-prefix.js), so a canvas without
// that attribute is invisible to the WASM hydration path. Instead it carries
// data-gosx-canvas-wasm-free="true", the marker the Studio-owned WASM-free
// client keys on to own painting + interaction. Width/Height default to the
// same 1280x720 the WASM board uses so the inline bundle's framebuffer matches.
func siteMapCanvasWASMFreeCanvas(background string, width, height int, canvasClass string) gosx.Node {
	w, h := width, height
	if w == 0 && h == 0 {
		w, h = 1280, 720
	}
	attrs := []any{
		gosx.Attr("id", "studio-site-map-canvas-board"),
		gosx.Attr("data-gosx-canvas-wasm-free", "true"),
		gosx.Attr("width", intToString(w)),
		gosx.Attr("height", intToString(h)),
		// A focusable canvas so arrow-key navigation targets it (the WASM-free
		// client calls canvas.focus() on pointerdown and listens for keydown).
		gosx.Attr("tabindex", "0"),
	}
	if background != "" {
		// Match the WASM CanvasBoard's inline background so the element shows the
		// board surface color before the first JS paint.
		attrs = append(attrs, gosx.Attr("style", "background:"+background))
	}
	if canvasClass != "" {
		attrs = append(attrs, gosx.Attr("class", canvasClass))
	}
	return gosx.El("canvas", gosx.Attrs(attrs...))
}

// intToString renders a non-negative int as a decimal string without pulling in
// strconv at this seam (the file is otherwise import-light).
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// siteMapCanvasBundleScript computes the Canvas2D RenderBundle for the board
// nodes SERVER-SIDE (plain Go, no syscall/js) via gosx's bundle2d API and wraps
// the painter-ready JSON in a <script type="application/json"
// data-gosx-canvas-bundle> element. The WASM-free client locates the surface,
// reads this sibling script's textContent, JSON.parses it, and paints — no WASM,
// no extra round-trip.
//
// Width/Height are resolved to the same defaults gosx.CanvasBoard applies (both
// zero -> 1280x720), and the camera matches the authored Zoom=1, Pan=(0,0), so
// the explicit WASM-free path gets painter-shaped server bundle JSON from the
// same dimensions and camera. The JSON is embedded raw: gosx/bundle2d marshals
// via encoding/json, which escapes <, >, & as </>/&, so the payload
// can never contain a literal </script> sequence. Returns ok=false (and no
// script) only if marshaling fails, leaving the default CanvasBoard path
// untouched.
func siteMapCanvasBundleScript(nodes []gosx.CanvasBoardNode, background string, width, height int) (gosx.Node, bool) {
	w, h := width, height
	if w == 0 && h == 0 {
		w, h = 1280, 720
	}
	bundle := bundle2d.ComputeCanvasBundleWithBackground(nodes, background, w, h, 1.0, 0, 0)
	payload, err := bundle2d.MarshalCanvasBundle(bundle)
	if err != nil {
		return gosx.Node{}, false
	}
	script := gosx.El("script", gosx.Attrs(
		gosx.Attr("type", "application/json"),
		gosx.Attr("data-gosx-canvas-bundle", "studio-site-map-canvas-board"),
	), gosx.RawHTML(payload))
	return script, true
}

// siteMapCanvasNodes builds CanvasBoardNodes from an AuthoringSiteMapView.
//
// Layout strategy: AuthoringSiteMapView does not expose numeric per-node
// coordinates — it only carries an SVG viewBox plus SVG path strings for the
// DOM board. So node positions are derived deterministically from the
// workspace graph structure: each "workspaceLayers" entry is a column
// (x = layerIndex * columnStride) and each node within a layer is a row
// (y = nodeIndex * rowStride). Every workspace node becomes one "rect"
// (id == node key, color keyed by page group) plus one "label" node carrying
// its title. Every "workspaceLinks" entry becomes one "line" node drawn from
// the center of its from-rect to the center of its to-rect, when both
// endpoints resolve to placed nodes.
func siteMapCanvasNodes(siteMapView map[string]any) []gosx.CanvasBoardNode {
	return siteMapCanvasNodesWith(siteMapView, SiteMapCanvasOptions{})
}

func siteMapCanvasNodesWith(siteMapView map[string]any, options SiteMapCanvasOptions) []gosx.CanvasBoardNode {
	layout := siteMapCanvasLayoutFrom(options)
	layers := core.WorkbenchViewMapList(siteMapView, "workspaceLayers")
	placements := map[string]siteMapCanvasNodePlacement{}

	rects := make([]gosx.CanvasBoardNode, 0)
	thumbs := make([]gosx.CanvasBoardNode, 0)
	surfaces := make([]gosx.CanvasBoardNode, 0)
	labels := make([]gosx.CanvasBoardNode, 0)

	for layerIndex, layer := range layers {
		nodes := core.WorkbenchViewMapList(layer, "nodes")
		x := float64(layerIndex) * layout.columnStride
		for nodeIndex, node := range nodes {
			key := core.WorkbenchViewString(node, "key")
			if key == "" {
				continue
			}
			// PageArtboardsOnly renders page artboards only: skip component and
			// resource nodes entirely so the board shows just page cards. Their
			// placements are never recorded, but that's safe because the link
			// "spiderweb" loop below is also skipped under this flag, so there are
			// no dangling endpoints to resolve.
			if options.PageArtboardsOnly && core.WorkbenchViewString(node, "kind") != string(core.WorkspaceNodePage) {
				continue
			}
			if _, seen := placements[key]; seen {
				// A node may appear in more than one layer view (e.g. shared
				// resources). Place it once, at its first occurrence.
				continue
			}
			y := float64(nodeIndex) * layout.rowStride
			placement := siteMapCanvasNodePlacement{
				x:      x,
				y:      y,
				width:  layout.nodeWidth,
				height: layout.nodeHeight,
			}
			placements[key] = placement

			rects = append(rects, gosx.CanvasBoardNode{
				ID:     key,
				Kind:   "rect",
				X:      placement.x,
				Y:      placement.y,
				Width:  placement.width,
				Height: placement.height,
				Color:  siteMapCanvasGroupColor(core.WorkbenchViewString(node, "group")),
			})
			labels = append(labels, gosx.CanvasBoardNode{
				ID:    key + ":label",
				Kind:  "label",
				X:     placement.x + siteMapCanvasLabelInsetX,
				Y:     placement.y + siteMapCanvasLabelInsetY,
				Color: siteMapCanvasLabelColor,
				Text:  siteMapCanvasNodeText(node),
			})

			// For a PAGE node that carries a route, stack the route as a muted
			// subtitle label a row below the title but still inside the rect
			// bounds, so CanvasBoard/render-bundle consumers associate it to the
			// card by world-point containment and render it as the subtitle. The
			// authoring view exposes "route" only on page nodes (components and
			// resources leave it empty), so the kind+route gate keeps the route
			// label off non-page cards.
			if core.WorkbenchViewString(node, "kind") == string(core.WorkspaceNodePage) {
				if route := core.WorkbenchViewString(node, "route"); route != "" {
					labels = append(labels, gosx.CanvasBoardNode{
						ID:    key + ":route",
						Kind:  "label",
						X:     placement.x + siteMapCanvasLabelInsetX,
						Y:     placement.y + siteMapCanvasRouteInsetY,
						Color: siteMapCanvasSubtitleColor,
						Text:  route,
					})
				}

				// When a per-page thumbnail URI is supplied for this page card's
				// node key, emit an additional "image" node at the card's exact
				// placement bounds. It serializes with the rects/labels into the
				// default CanvasBoard props payload and is carried through the
				// legacy WASMFree inline bundle when that compatibility path is
				// explicitly selected.
				if dataURI := options.Thumbnails[key]; dataURI != "" {
					thumbs = append(thumbs, gosx.CanvasBoardNode{
						ID:     key + ":thumb",
						Kind:   "image",
						Src:    dataURI,
						X:      placement.x,
						Y:      placement.y,
						Width:  placement.width,
						Height: placement.height,
					})
				}

				// When a per-page full editable HTML surface is supplied for this
				// page card's node key, emit an additional "html" node at the card's
				// exact placement bounds. Its ID is the page node key and
				// PointerEvents is "auto" so the CanvasBoard render bundle can expose
				// it as a scene/html DOM overlay. Keyed by the page node key; only
				// page cards are eligible, and only when the value is non-empty.
				if markup := options.PageSurfaces[key]; markup != "" {
					surfaces = append(surfaces, gosx.CanvasBoardNode{
						ID:            key,
						Kind:          "html",
						Markup:        markup,
						X:             placement.x,
						Y:             placement.y,
						Width:         placement.width,
						Height:        placement.height,
						PointerEvents: "auto",
					})
				}
			}
		}
	}

	lines := make([]gosx.CanvasBoardNode, 0)
	// PageArtboardsOnly drops the link "spiderweb": with page artboards only there
	// are no inter-node connectors to draw (and non-page endpoints have no
	// placements anyway). When the flag is off this emits the full link graph.
	if !options.PageArtboardsOnly {
		for _, link := range core.WorkbenchViewMapList(siteMapView, "workspaceLinks") {
			from, okFrom := placements[core.WorkbenchViewString(link, "fromNodeKey")]
			to, okTo := placements[core.WorkbenchViewString(link, "toNodeKey")]
			if !okFrom || !okTo {
				continue
			}
			lines = append(lines, gosx.CanvasBoardNode{
				ID:    core.WorkbenchViewString(link, "key"),
				Kind:  "line",
				X1:    from.centerX(),
				Y1:    from.centerY(),
				X2:    to.centerX(),
				Y2:    to.centerY(),
				Color: siteMapCanvasLinkColor,
			})
		}
	}

	// Draw links first so rects paint on top of connectors; thumbnail images
	// paint over their card's rect fill; live html surfaces paint over the
	// thumbnails (so the editable surface wins at near zoom); labels paint last so
	// card text stays legible above the overlays. Ordering is deterministic.
	out := make([]gosx.CanvasBoardNode, 0, len(lines)+len(rects)+len(thumbs)+len(surfaces)+len(labels))
	out = append(out, lines...)
	out = append(out, rects...)
	out = append(out, thumbs...)
	out = append(out, surfaces...)
	out = append(out, labels...)
	return flipBoardNodesYUp(out)
}

// flipBoardNodesYUp converts the board's top-down AUTHORING (row 0 at Y=0, the
// route subtitle "below" the title at a larger Y, +Y-down screen convention)
// into the +Y-up world space the GoSX renderers, click hit-testing, and the DOM
// overlays all use. It mirrors every node's Y around the board's vertical extent
// so the rendered board reads top-down (title above route, row 0 at the top) and
// — crucially — every consumer agrees, with NO renderer-side projection flip
// (which would desync render from picking/overlays). The board's bounding box is
// preserved (mirror around maxY keeps nodes in [0, maxY]), so the camera/pan are
// unaffected. Empty input is returned unchanged.
func flipBoardNodesYUp(nodes []gosx.CanvasBoardNode) []gosx.CanvasBoardNode {
	maxY := 0.0
	for _, n := range nodes {
		bottom := n.Y + n.Height // rect / image / html bottom edge
		switch n.Kind {
		case "label":
			bottom = n.Y
		case "line":
			bottom = n.Y1
			if n.Y2 > bottom {
				bottom = n.Y2
			}
		}
		if bottom > maxY {
			maxY = bottom
		}
	}
	for i := range nodes {
		n := &nodes[i]
		switch n.Kind {
		case "label":
			n.Y = maxY - n.Y
		case "line":
			n.Y1 = maxY - n.Y1
			n.Y2 = maxY - n.Y2
		default:
			n.Y = maxY - n.Y - n.Height
		}
	}
	return nodes
}

func siteMapCanvasNodeText(node map[string]any) string {
	return core.FirstNonEmpty(
		core.WorkbenchViewString(node, "label"),
		core.WorkbenchViewString(node, "key"),
	)
}

const (
	siteMapCanvasLabelColor = "#e2e8f0"
	// siteMapCanvasSubtitleColor is the dimmer fill for the page route subtitle,
	// so the route reads as muted secondary text under the card title.
	siteMapCanvasSubtitleColor = "#94a3b8"
	siteMapCanvasLinkColor     = "#475569"
)

// siteMapCanvasGroupColor maps a page group key to a stable rect fill color so
// the board reads at a glance. Unknown/empty groups (e.g. resource nodes) get
// a neutral slate.
func siteMapCanvasGroupColor(group string) string {
	switch group {
	case string(core.PageGroupSite):
		return "#2563eb"
	case string(core.PageGroupCommerce):
		return "#7c3aed"
	case string(core.PageGroupContent):
		return "#0d9488"
	case string(core.PageGroupFlows):
		return "#d97706"
	case string(core.PageGroupUtility):
		return "#475569"
	default:
		return "#334155"
	}
}

func positiveOr(value, fallback float64) float64 {
	if value > 0 {
		return value
	}
	return fallback
}
