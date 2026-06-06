package studio

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/render/bundle2d"
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
	// per-page thumbnail data URI (the real themed page content, generated
	// upstream by muddy as an SVG-<foreignObject> data URI). When a page card's
	// node key is present here with a non-empty value, the shared node builder
	// emits an ADDITIONAL Kind:"image" node (ID = key+":thumb", Src = the data
	// URI) at the card's exact placement bounds, so the painter can paint it as
	// a full-card overlay at medium zoom. Pages with no entry (or an empty
	// value) get no image node, and entries keyed to non-page nodes are ignored.
	// Empty/nil by default, so the standard site-map render is unchanged.
	Thumbnails map[string]string

	// PageSurfaces maps a page card's workspace node key (e.g. "page:home") to
	// the host-supplied full editable HTML markup for that page. When a page
	// card's node key is present here with a non-empty value, the shared node
	// builder emits an ADDITIONAL Kind:"html" node (ID = the page node key, which
	// is the painter mount key; Markup = the supplied HTML; PointerEvents =
	// "auto") at the card's exact placement bounds, so the painter can mount it
	// as a full-card LIVE editable surface at near zoom. The html node is appended
	// AFTER the thumbnail images so live surfaces paint last (on top). Pages with
	// no entry (or an empty value) get no html node, and entries keyed to non-page
	// nodes are ignored. Empty/nil by default, so the standard site-map render is
	// unchanged.
	PageSurfaces map[string]string

	// ExtraNodes are appended verbatim to the deterministically-laid-out
	// site-map nodes before the bundle is computed, so a host can place
	// additional CanvasBoardNodes (e.g. a Kind:"html" surface) onto the same
	// board. They paint AND serialize through the identical bundle path as the
	// site-map rects/labels/lines, so an "html" node here flows into the inline
	// RenderBundle's html records (and thus the WASM-free overlay). Empty by
	// default, so the standard site-map render is unchanged.
	ExtraNodes []gosx.CanvasBoardNode

	// PageArtboardsOnly renders the canvas as page artboards only: when true the
	// shared node builder emits rect + title label (+ route subtitle + thumbnail)
	// ONLY for workspace nodes whose kind is WorkspaceNodePage, skipping
	// component and resource nodes entirely, AND emits NO Kind:"line" nodes (the
	// link "spiderweb" between nodes is dropped). When false (the default) the
	// full authoring graph renders — all workspace nodes plus the connecting
	// lines — exactly as before, so existing callers are unchanged.
	PageArtboardsOnly bool

	// WASMFree opts the surface into the WASM-free render path. When true the
	// renderer emits a plain <canvas> that DELIBERATELY carries NO
	// data-gosx-surface-kind (and instead carries data-gosx-canvas-wasm-free),
	// so the gosx client bootstrap's canvas discovery
	// ([data-gosx-surface-kind]:not([data-gosx-engine-bytecode])) never tries to
	// WASM-hydrate it. The SAME server-precomputed inline RenderBundle is still
	// emitted next to the canvas, so a WASM-free client (gosx-studio's host ships
	// a ported JS painter + interaction loop) can paint and drive it with no WASM.
	// When false (the default) the surface emits the WASM gosx.CanvasBoard exactly
	// as before, so the existing co-render / canvas-default modes are unchanged.
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
	// so the muddy painter associates it to the card by world-point containment
	// and renders it as the muted subtitle.
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
			gosx.Attr("class", FirstNonEmpty(options.Class, "studio-site-map-canvas")),
			gosx.Attr("data-studio-site-map-canvas-engine", "disabled"),
			gosx.Attr("hidden", true),
			gosx.Attr("aria-hidden", "true"),
		))
	}

	// Build the board nodes once and feed BOTH the (WASM) CanvasBoard primitive
	// and the server-precomputed bundle below, so the inline bundle is exactly
	// what the WASM path would produce for this surface.
	nodes := siteMapCanvasNodesWith(siteMapView, options)
	if len(options.ExtraNodes) > 0 {
		// Appended last so host-supplied nodes (e.g. a Kind:"html" surface)
		// paint on top of the site-map graph and serialize through the same
		// bundle path.
		nodes = append(nodes, options.ExtraNodes...)
	}
	background := FirstNonEmpty(options.Background, siteMapCanvasDefaultBackground)

	// In the WASM-free path we emit a plain <canvas> that carries NO
	// data-gosx-surface-kind, so the gosx client bootstrap never WASM-hydrates it;
	// otherwise we emit the WASM gosx.CanvasBoard exactly as before. Either way the
	// same inline RenderBundle (computed below) ships next to the canvas.
	var board gosx.Node
	if options.WASMFree {
		board = siteMapCanvasWASMFreeCanvas(background, options.Width, options.Height, options.CanvasClass)
	} else {
		board = gosx.CanvasBoard(gosx.CanvasBoardProps{
			ID:         "studio-site-map-canvas-board",
			Width:      options.Width,
			Height:     options.Height,
			Background: background,
			Zoom:       1.0,
			Nodes:      nodes,
			OnPick:     FirstNonEmpty(options.OnPick, siteMapCanvasDefaultOnPick),
			ClassName:  options.CanvasClass,
		})
	}

	// gosx.El takes ...any (AttrList + Node children), so children carry as []any.
	sectionAttrs := []any{
		gosx.Attr("class", FirstNonEmpty(options.Class, "studio-site-map-canvas")),
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
	if bundleScript, ok := siteMapCanvasBundleScript(nodes, background, options.Width, options.Height); ok {
		// Additive: the WASM path still hydrates `board` from data-gosx-engine-props.
		// This script ADDS the same bundle precomputed server-side (plain Go, no
		// WASM) so a WASM-free client can paint it directly. Placed after the canvas
		// so a client can resolve the surface, then read its sibling bundle.
		children = append(children, bundleScript)
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
// data-gosx-canvas-wasm-free="true", the marker the muddy WASM-free client keys
// on to own painting + interaction. Width/Height default to the same 1280x720
// the WASM board uses so the inline bundle's framebuffer matches.
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
// Width/Height are resolved to the SAME defaults gosx.CanvasBoard applies (both
// zero → 1280x720) and the camera matches the board's authored Zoom=1, Pan=(0,0),
// so the inline bundle is byte-identical to what the WASM CanvasBoardAdapter
// would render for this surface. The JSON is embedded raw: gosx/bundle2d marshals
// via encoding/json, which escapes <, >, & as </>/&, so the payload
// can never contain a literal </script> sequence. Returns ok=false (and no
// script) only if marshaling fails, leaving the WASM path untouched.
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
	layers := workbenchViewMapList(siteMapView, "workspaceLayers")
	placements := map[string]siteMapCanvasNodePlacement{}

	rects := make([]gosx.CanvasBoardNode, 0)
	thumbs := make([]gosx.CanvasBoardNode, 0)
	surfaces := make([]gosx.CanvasBoardNode, 0)
	labels := make([]gosx.CanvasBoardNode, 0)

	for layerIndex, layer := range layers {
		nodes := siteMapMapList(layer, "nodes")
		x := float64(layerIndex) * layout.columnStride
		for nodeIndex, node := range nodes {
			key := workbenchMapString(node, "key")
			if key == "" {
				continue
			}
			// PageArtboardsOnly renders page artboards only: skip component and
			// resource nodes entirely so the board shows just page cards. Their
			// placements are never recorded, but that's safe because the link
			// "spiderweb" loop below is also skipped under this flag, so there are
			// no dangling endpoints to resolve.
			if options.PageArtboardsOnly && workbenchMapString(node, "kind") != string(WorkspaceNodePage) {
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
				Color:  siteMapCanvasGroupColor(workbenchMapString(node, "group")),
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
			// bounds, so the muddy painter associates it to the card by
			// world-point containment and renders it as the subtitle. The
			// authoring view exposes "route" only on page nodes (components and
			// resources leave it empty), so the kind+route gate keeps the route
			// label off non-page cards.
			if workbenchMapString(node, "kind") == string(WorkspaceNodePage) {
				if route := workbenchMapString(node, "route"); route != "" {
					labels = append(labels, gosx.CanvasBoardNode{
						ID:    key + ":route",
						Kind:  "label",
						X:     placement.x + siteMapCanvasLabelInsetX,
						Y:     placement.y + siteMapCanvasRouteInsetY,
						Color: siteMapCanvasSubtitleColor,
						Text:  route,
					})
				}

				// When a per-page thumbnail data URI is supplied for this page
				// card's node key, emit an additional "image" node at the card's
				// exact placement bounds. It serializes through the same bundle
				// path as the rects/labels (adapter case "image","sprite" → a
				// sprite), so the WASM-free painter can show it as a full-card
				// overlay. Keyed by the page node key; only page cards are
				// eligible, and only when the value is non-empty.
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
				// exact placement bounds. Its ID is the page node key (the painter
				// mount key) and PointerEvents is "auto" so the painter can mount it
				// as a full-card LIVE editable surface. Keyed by the page node key;
				// only page cards are eligible, and only when the value is non-empty.
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
		for _, link := range workbenchViewMapList(siteMapView, "workspaceLinks") {
			from, okFrom := placements[workbenchMapString(link, "fromNodeKey")]
			to, okTo := placements[workbenchMapString(link, "toNodeKey")]
			if !okFrom || !okTo {
				continue
			}
			lines = append(lines, gosx.CanvasBoardNode{
				ID:    workbenchMapString(link, "key"),
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
	return out
}

func siteMapCanvasNodeText(node map[string]any) string {
	return FirstNonEmpty(
		workbenchMapString(node, "label"),
		workbenchMapString(node, "key"),
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
	case string(PageGroupSite):
		return "#2563eb"
	case string(PageGroupCommerce):
		return "#7c3aed"
	case string(PageGroupContent):
		return "#0d9488"
	case string(PageGroupFlows):
		return "#d97706"
	case string(PageGroupUtility):
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
