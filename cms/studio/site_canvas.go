package studio

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"m31labs.dev/gosx"
)

type SiteCanvasNode struct {
	Key        string
	Kind       string
	Label      string
	Summary    string
	Status     string
	Href       string
	X          float64
	Y          float64
	Width      float64
	Height     float64
	Selected   bool
	XInputName string
	YInputName string
	Metrics    []Metric
	Tags       []string
}

type SiteCanvasEdge struct {
	Key   string
	From  string
	To    string
	Kind  string
	Label string
}

type SiteCanvasOptions struct {
	Class                string
	ToolbarClass         string
	ControlsClass        string
	ViewportClass        string
	SurfaceClass         string
	EdgesClass           string
	NodesClass           string
	NodeClass            string
	Label                string
	Kicker               string
	Title                string
	Summary              string
	PositionInputPrefix  string
	Zoom                 float64
	PanX                 float64
	PanY                 float64
	KeyboardNudge        float64
	PersistNodePositions bool
	Nodes                []SiteCanvasNode
	Edges                []SiteCanvasEdge
	Controls             []gosx.Node
}

func RenderSiteCanvas(options SiteCanvasOptions) gosx.Node {
	nodes := normalizeSiteCanvasNodes(options.Nodes)
	nodes = attachSiteCanvasPositionInputs(nodes, options)
	edges := normalizeSiteCanvasEdges(options.Edges, nodes)
	zoom := options.Zoom
	if zoom <= 0 {
		zoom = 1
	}
	width, height := siteCanvasBounds(nodes)
	keyboardNudge := options.KeyboardNudge
	if keyboardNudge <= 0 {
		keyboardNudge = 8
	}
	className := firstNonEmpty(options.Class, "gosx-studio-site-canvas")
	toolbarClass := firstNonEmpty(options.ToolbarClass, "gosx-studio-site-canvas__toolbar")
	controlsClass := firstNonEmpty(options.ControlsClass, "gosx-studio-site-canvas__controls")
	viewportClass := firstNonEmpty(options.ViewportClass, "gosx-studio-site-canvas__viewport")
	surfaceClass := firstNonEmpty(options.SurfaceClass, "gosx-studio-site-canvas__surface")
	edgesClass := firstNonEmpty(options.EdgesClass, "gosx-studio-site-canvas__edges")
	nodesClass := firstNonEmpty(options.NodesClass, "gosx-studio-site-canvas__nodes")
	label := firstNonEmpty(options.Label, options.Title, "Site canvas")

	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-site-canvas", "true"),
		gosx.Attr("data-gosx-studio-canvas-zoom", fmtFloat(zoom)),
		gosx.Attr("data-gosx-studio-canvas-pan-x", fmtFloat(options.PanX)),
		gosx.Attr("data-gosx-studio-canvas-pan-y", fmtFloat(options.PanY)),
		gosx.Attr("data-gosx-studio-canvas-keyboard-nudge", fmtFloat(keyboardNudge)),
		gosx.Attr("tabindex", "0"),
		gosx.Attr("aria-label", label),
	),
		renderSiteCanvasToolbar(toolbarClass, controlsClass, options),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", viewportClass),
			gosx.Attr("data-gosx-studio-canvas-viewport", "true"),
		),
			gosx.El("div", gosx.Attrs(
				gosx.Attr("class", surfaceClass),
				gosx.Attr("data-gosx-studio-canvas-surface", "true"),
				gosx.Attr("style", canvasSurfaceStyle(options.PanX, options.PanY, zoom, width, height)),
			),
				renderSiteCanvasEdges(edgesClass, width, height, nodes, edges),
				renderSiteCanvasNodes(nodesClass, options.NodeClass, nodes),
			),
		),
	)
}

type SiteCanvasPosition struct {
	Key string
	X   float64
	Y   float64
}

type SiteCanvasPositionFormOptions struct {
	NamePrefix string
}

func SiteCanvasPositionInputName(prefix, nodeKey, axis string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = "siteCanvas"
	}
	axis = strings.ToUpper(strings.TrimSpace(axis))
	if axis != "Y" {
		axis = "X"
	}
	return prefix + FieldNamePrefix(nodeKey) + axis
}

func SiteCanvasPositionInputNames(prefix, nodeKey string) (string, string) {
	return SiteCanvasPositionInputName(prefix, nodeKey, "x"), SiteCanvasPositionInputName(prefix, nodeKey, "y")
}

func SiteCanvasPositionsFromForm(form map[string]string, nodes []SiteCanvasNode, options SiteCanvasPositionFormOptions) (map[string]SiteCanvasPosition, error) {
	positions := map[string]SiteCanvasPosition{}
	if len(form) == 0 || len(nodes) == 0 {
		return positions, nil
	}
	nodes = normalizeSiteCanvasNodes(nodes)
	for _, node := range nodes {
		xName := firstNonEmpty(node.XInputName, SiteCanvasPositionInputName(options.NamePrefix, node.Key, "x"))
		yName := firstNonEmpty(node.YInputName, SiteCanvasPositionInputName(options.NamePrefix, node.Key, "y"))
		xValue, hasX := form[xName]
		yValue, hasY := form[yName]
		if !hasX && !hasY {
			continue
		}
		position := SiteCanvasPosition{Key: node.Key, X: node.X, Y: node.Y}
		if hasX {
			x, err := strconv.ParseFloat(strings.TrimSpace(xValue), 64)
			if err != nil {
				return positions, fmt.Errorf("invalid site canvas x position for %s: %w", node.Key, err)
			}
			position.X = x
		}
		if hasY {
			y, err := strconv.ParseFloat(strings.TrimSpace(yValue), 64)
			if err != nil {
				return positions, fmt.Errorf("invalid site canvas y position for %s: %w", node.Key, err)
			}
			position.Y = y
		}
		positions[node.Key] = position
	}
	return positions, nil
}

func ApplySiteCanvasPositions(nodes []SiteCanvasNode, positions map[string]SiteCanvasPosition) []SiteCanvasNode {
	if len(nodes) == 0 || len(positions) == 0 {
		return nodes
	}
	out := make([]SiteCanvasNode, 0, len(nodes))
	for _, node := range nodes {
		key := normalizeKey(firstNonEmpty(node.Key, node.Label))
		if position, ok := positions[key]; ok {
			node.X = position.X
			node.Y = position.Y
		}
		out = append(out, node)
	}
	return out
}

func renderSiteCanvasToolbar(className, controlsClass string, options SiteCanvasOptions) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", nil,
			optionalText("p", "kicker", options.Kicker),
			gosx.El("strong", nil, gosx.Text(firstNonEmpty(options.Title, "Site canvas"))),
			optionalText("span", "", options.Summary),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", controlsClass),
			gosx.Attr("role", "toolbar"),
			gosx.Attr("aria-label", "Site canvas controls"),
		),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-canvas-zoom-out", "true")), gosx.Text("-")),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-canvas-reset", "true")), gosx.Text("Fit")),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-canvas-zoom-in", "true")), gosx.Text("+")),
		),
	}
	children = append(children, options.Controls...)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func renderSiteCanvasEdges(className string, width, height float64, nodes []SiteCanvasNode, edges []SiteCanvasEdge) gosx.Node {
	byKey := map[string]SiteCanvasNode{}
	for _, node := range nodes {
		byKey[node.Key] = node
	}
	edgeNodes := make([]gosx.Node, 0, len(edges))
	for _, edge := range edges {
		from := byKey[edge.From]
		to := byKey[edge.To]
		x1, y1 := from.X+from.Width, from.Y+(from.Height/2)
		x2, y2 := to.X, to.Y+(to.Height/2)
		if to.X < from.X {
			x1, x2 = from.X, to.X+to.Width
		}
		edgeNodes = append(edgeNodes, gosx.El("path", gosx.Attrs(
			gosx.Attr("data-gosx-studio-canvas-edge", edge.Key),
			gosx.Attr("data-gosx-studio-canvas-edge-kind", edge.Kind),
			gosx.Attr("data-gosx-studio-canvas-edge-from", edge.From),
			gosx.Attr("data-gosx-studio-canvas-edge-to", edge.To),
			gosx.Attr("aria-label", firstNonEmpty(edge.Label, edge.From+" to "+edge.To)),
			gosx.Attr("d", fmt.Sprintf("M %s %s C %s %s, %s %s, %s %s", fmtFloat(x1), fmtFloat(y1), fmtFloat(x1+80), fmtFloat(y1), fmtFloat(x2-80), fmtFloat(y2), fmtFloat(x2), fmtFloat(y2))),
		)))
	}
	return gosx.El("svg", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-canvas-edges", "true"),
		gosx.Attr("viewBox", fmt.Sprintf("0 0 %s %s", fmtFloat(width), fmtFloat(height))),
		gosx.Attr("aria-hidden", "true"),
	), gosx.Fragment(edgeNodes...))
}

func renderSiteCanvasNodes(className, nodeClass string, nodes []SiteCanvasNode) gosx.Node {
	children := make([]gosx.Node, 0, len(nodes))
	for _, node := range nodes {
		children = append(children, renderSiteCanvasNode(nodeClass, node))
		children = append(children, renderSiteCanvasPositionInputs(node)...)
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-canvas-nodes", "true"),
	), gosx.Fragment(children...))
}

func renderSiteCanvasPositionInputs(node SiteCanvasNode) []gosx.Node {
	fields := []gosx.Node{}
	if strings.TrimSpace(node.XInputName) != "" {
		fields = append(fields, renderSiteCanvasPositionInput(node, "x", node.XInputName, node.X))
	}
	if strings.TrimSpace(node.YInputName) != "" {
		fields = append(fields, renderSiteCanvasPositionInput(node, "y", node.YInputName, node.Y))
	}
	return fields
}

func renderSiteCanvasPositionInput(node SiteCanvasNode, axis, name string, value float64) gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("type", "hidden"),
		gosx.Attr("name", name),
		gosx.Attr("value", fmtFloat(value)),
		gosx.Attr("data-gosx-studio-canvas-node-position", "true"),
		gosx.Attr("data-gosx-studio-canvas-node-position-key", node.Key),
		gosx.Attr("data-gosx-studio-canvas-node-position-axis", axis),
	))
}

func renderSiteCanvasNode(className string, node SiteCanvasNode) gosx.Node {
	kind := firstNonEmpty(node.Kind, "surface")
	baseClass := firstNonEmpty(className, "gosx-studio-site-canvas__node")
	baseToken := firstClass(baseClass)
	classes := strings.TrimSpace(baseClass + " " + baseToken + "--" + normalizeKey(kind))
	if node.Selected {
		classes += " is-selected"
	}
	children := []gosx.Node{
		gosx.El("span", gosx.Attrs(gosx.Attr("class", baseToken+"-kind")), gosx.Text(kind)),
		gosx.El("strong", nil, gosx.Text(node.Label)),
		optionalText("span", baseToken+"-summary", node.Summary),
		optionalText("output", baseToken+"-status", node.Status),
	}
	if len(node.Metrics) > 0 {
		metrics := make([]gosx.Node, 0, len(node.Metrics))
		for _, metric := range normalizeMetrics(node.Metrics) {
			metrics = append(metrics, gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(fmtAny(metric.Value))),
				gosx.Text(" "+metric.Label),
			))
		}
		children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", baseToken+"-metrics")), gosx.Fragment(metrics...)))
	}
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("class", classes),
		gosx.Attr("type", "button"),
		gosx.Attr("data-gosx-studio-canvas-node", node.Key),
		gosx.Attr("data-gosx-studio-canvas-node-kind", kind),
		gosx.Attr("data-gosx-studio-canvas-node-label", node.Label),
		gosx.Attr("data-gosx-studio-canvas-node-href", node.Href),
		gosx.Attr("aria-pressed", boolAttr(node.Selected)),
		gosx.Attr("style", canvasNodeStyle(node)),
	), gosx.Fragment(children...))
}

func firstClass(className string) string {
	fields := strings.Fields(className)
	if len(fields) == 0 {
		return "gosx-studio-site-canvas__node"
	}
	return fields[0]
}

func normalizeSiteCanvasNodes(nodes []SiteCanvasNode) []SiteCanvasNode {
	out := make([]SiteCanvasNode, 0, len(nodes))
	for index, node := range nodes {
		node.Key = normalizeKey(firstNonEmpty(node.Key, node.Label, fmt.Sprintf("node-%d", index+1)))
		node.Kind = normalizeKey(firstNonEmpty(node.Kind, "surface"))
		node.Label = strings.TrimSpace(firstNonEmpty(node.Label, node.Key))
		node.Summary = strings.TrimSpace(node.Summary)
		node.Status = strings.TrimSpace(node.Status)
		node.Href = strings.TrimSpace(node.Href)
		if node.Width <= 0 {
			node.Width = 240
		}
		if node.Height <= 0 {
			node.Height = 132
		}
		out = append(out, node)
	}
	return out
}

func attachSiteCanvasPositionInputs(nodes []SiteCanvasNode, options SiteCanvasOptions) []SiteCanvasNode {
	if !options.PersistNodePositions {
		return nodes
	}
	out := make([]SiteCanvasNode, 0, len(nodes))
	for _, node := range nodes {
		if strings.TrimSpace(node.XInputName) == "" || strings.TrimSpace(node.YInputName) == "" {
			xName, yName := SiteCanvasPositionInputNames(options.PositionInputPrefix, node.Key)
			if strings.TrimSpace(node.XInputName) == "" {
				node.XInputName = xName
			}
			if strings.TrimSpace(node.YInputName) == "" {
				node.YInputName = yName
			}
		}
		out = append(out, node)
	}
	return out
}

func normalizeSiteCanvasEdges(edges []SiteCanvasEdge, nodes []SiteCanvasNode) []SiteCanvasEdge {
	exists := map[string]bool{}
	for _, node := range nodes {
		exists[node.Key] = true
	}
	out := make([]SiteCanvasEdge, 0, len(edges))
	for index, edge := range edges {
		edge.From = normalizeKey(edge.From)
		edge.To = normalizeKey(edge.To)
		if !exists[edge.From] || !exists[edge.To] {
			continue
		}
		edge.Kind = normalizeKey(firstNonEmpty(edge.Kind, "link"))
		edge.Key = normalizeKey(firstNonEmpty(edge.Key, edge.From+"-"+edge.To, fmt.Sprintf("edge-%d", index+1)))
		edge.Label = strings.TrimSpace(edge.Label)
		out = append(out, edge)
	}
	return out
}

func siteCanvasBounds(nodes []SiteCanvasNode) (float64, float64) {
	width, height := 960.0, 640.0
	for _, node := range nodes {
		width = math.Max(width, node.X+node.Width+160)
		height = math.Max(height, node.Y+node.Height+160)
	}
	return width, height
}

func canvasSurfaceStyle(panX, panY, zoom, width, height float64) string {
	return fmt.Sprintf("--gosx-studio-canvas-pan-x:%spx;--gosx-studio-canvas-pan-y:%spx;--gosx-studio-canvas-zoom:%s;--gosx-studio-canvas-width:%spx;--gosx-studio-canvas-height:%spx;transform:translate3d(var(--gosx-studio-canvas-pan-x),var(--gosx-studio-canvas-pan-y),0) scale(var(--gosx-studio-canvas-zoom));width:var(--gosx-studio-canvas-width);height:var(--gosx-studio-canvas-height);", fmtFloat(panX), fmtFloat(panY), fmtFloat(zoom), fmtFloat(width), fmtFloat(height))
}

func canvasNodeStyle(node SiteCanvasNode) string {
	return fmt.Sprintf("--gosx-studio-node-x:%spx;--gosx-studio-node-y:%spx;--gosx-studio-node-width:%spx;--gosx-studio-node-height:%spx;left:var(--gosx-studio-node-x);top:var(--gosx-studio-node-y);width:var(--gosx-studio-node-width);min-height:var(--gosx-studio-node-height);", fmtFloat(node.X), fmtFloat(node.Y), fmtFloat(node.Width), fmtFloat(node.Height))
}

func fmtFloat(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.001 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.3f", value)
}
