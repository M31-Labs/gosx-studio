package studio

import (
	"m31labs.dev/gosx"
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
}

const (
	siteMapCanvasDefaultColumnStride = 280.0
	siteMapCanvasDefaultRowStride    = 120.0
	siteMapCanvasDefaultNodeWidth    = 200.0
	siteMapCanvasDefaultNodeHeight   = 72.0
	siteMapCanvasDefaultBackground   = "#0f1720"
	siteMapCanvasLabelInsetX         = 14.0
	siteMapCanvasLabelInsetY         = 28.0
	siteMapCanvasDefaultOnPick       = "handleSiteMapPick"
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

	board := gosx.CanvasBoard(gosx.CanvasBoardProps{
		ID:         "studio-site-map-canvas-board",
		Width:      options.Width,
		Height:     options.Height,
		Background: FirstNonEmpty(options.Background, siteMapCanvasDefaultBackground),
		Zoom:       1.0,
		Nodes:      siteMapCanvasNodes(siteMapView),
		OnPick:     FirstNonEmpty(options.OnPick, siteMapCanvasDefaultOnPick),
		ClassName:  options.CanvasClass,
	})

	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.Class, "studio-site-map-canvas")),
		gosx.Attr("data-studio-site-map-canvas-engine", "true"),
		gosx.Attr("data-gosx-studio-site-map-canvas-renderer", "gosx-studio"),
		gosx.Attr("aria-label", "Site map canvas surface"),
	), board)
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
	return siteMapCanvasNodesWith(siteMapView, siteMapCanvasLayoutFrom(SiteMapCanvasOptions{}))
}

func siteMapCanvasNodesWith(siteMapView map[string]any, layout siteMapCanvasLayout) []gosx.CanvasBoardNode {
	layers := workbenchViewMapList(siteMapView, "workspaceLayers")
	placements := map[string]siteMapCanvasNodePlacement{}

	rects := make([]gosx.CanvasBoardNode, 0)
	labels := make([]gosx.CanvasBoardNode, 0)

	for layerIndex, layer := range layers {
		nodes := siteMapMapList(layer, "nodes")
		x := float64(layerIndex) * layout.columnStride
		for nodeIndex, node := range nodes {
			key := workbenchMapString(node, "key")
			if key == "" {
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
		}
	}

	lines := make([]gosx.CanvasBoardNode, 0)
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

	// Draw links first so rects/labels paint on top of connectors.
	out := make([]gosx.CanvasBoardNode, 0, len(lines)+len(rects)+len(labels))
	out = append(out, lines...)
	out = append(out, rects...)
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
	siteMapCanvasLinkColor  = "#475569"
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
