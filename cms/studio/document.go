package studio

import (
	"fmt"
	"strings"
	"time"
)

const (
	StudioDocumentNodePage    = "page"
	StudioDocumentNodeContent = "content"
	StudioDocumentNodeStyle   = "style"
	StudioDocumentNodeFlow    = "flow"
	StudioDocumentNodeRelease = "release"
)

type StudioDocument struct {
	Key      string
	Label    string
	Summary  string
	Status   ReadinessStatus
	Pages    []StudioPage
	Content  []StudioContent
	Styles   []StudioStyle
	Flows    []StudioFlow
	Releases []StudioRelease
	Edges    []StudioDocumentEdge
}

type StudioPage struct {
	Key         string
	Label       string
	Path        string
	Summary     string
	Status      ReadinessStatus
	ParentKey   string
	ContentKeys []string
	StyleKeys   []string
	FlowKeys    []string
	ReleaseKeys []string
	Href        string
	View        StudioDocumentNodeView
	Metrics     []Metric
	Tags        []string
}

type StudioContent struct {
	Key     string
	Label   string
	Kind    string
	Summary string
	Status  ReadinessStatus
	Href    string
	View    StudioDocumentNodeView
	Metrics []Metric
	Tags    []string
}

type StudioStyle struct {
	Key     string
	Label   string
	Scope   string
	Summary string
	Status  ReadinessStatus
	Tokens  map[string]string
	Href    string
	View    StudioDocumentNodeView
	Metrics []Metric
	Tags    []string
}

type StudioFlow struct {
	Key        string
	Label      string
	Trigger    string
	Route      string
	Summary    string
	Status     ReadinessStatus
	StepCount  int
	Executable bool
	Href       string
	View       StudioDocumentNodeView
	Metrics    []Metric
	Tags       []string
}

type StudioRelease struct {
	Key         string
	Label       string
	Summary     string
	Status      ReadinessStatus
	ScheduledAt time.Time
	Href        string
	View        StudioDocumentNodeView
	Metrics     []Metric
	Tags        []string
}

type StudioDocumentEdge struct {
	Key   string
	From  string
	To    string
	Kind  string
	Label string
}

type StudioDocumentNodeView struct {
	X        float64
	Y        float64
	Width    float64
	Height   float64
	Selected bool
}

type StudioDocumentViewMaps struct {
	Pages    map[string]StudioPage
	Content  map[string]StudioContent
	Styles   map[string]StudioStyle
	Flows    map[string]StudioFlow
	Releases map[string]StudioRelease
	Edges    map[string]StudioDocumentEdge
	Nodes    map[string]StudioDocumentNode
}

type StudioDocumentNode struct {
	Key     string
	Kind    string
	Label   string
	Summary string
	Status  ReadinessStatus
	Href    string
	View    StudioDocumentNodeView
	Metrics []Metric
	Tags    []string
}

func NormalizeStudioDocument(document StudioDocument) StudioDocument {
	document.Key = normalizeKey(document.Key)
	document.Label = strings.TrimSpace(document.Label)
	document.Summary = strings.TrimSpace(document.Summary)
	document.Status = normalizeReadinessStatus(document.Status)
	if document.Key == "" {
		document.Key = normalizeKey(firstNonEmpty(document.Label, "studio-document"))
	}
	if document.Label == "" {
		document.Label = "Studio document"
	}

	document.Pages = normalizeStudioPages(document.Pages)
	document.Content = normalizeStudioContent(document.Content)
	document.Styles = normalizeStudioStyles(document.Styles)
	document.Flows = normalizeStudioFlows(document.Flows)
	document.Releases = normalizeStudioReleases(document.Releases)

	maps := document.ViewMaps()
	document.Edges = normalizeStudioDocumentEdges(append(inferredStudioDocumentEdges(document), document.Edges...), maps.Nodes)
	return document
}

func (document StudioDocument) ViewMaps() StudioDocumentViewMaps {
	document.Pages = normalizeStudioPages(document.Pages)
	document.Content = normalizeStudioContent(document.Content)
	document.Styles = normalizeStudioStyles(document.Styles)
	document.Flows = normalizeStudioFlows(document.Flows)
	document.Releases = normalizeStudioReleases(document.Releases)

	maps := StudioDocumentViewMaps{
		Pages:    map[string]StudioPage{},
		Content:  map[string]StudioContent{},
		Styles:   map[string]StudioStyle{},
		Flows:    map[string]StudioFlow{},
		Releases: map[string]StudioRelease{},
		Edges:    map[string]StudioDocumentEdge{},
		Nodes:    map[string]StudioDocumentNode{},
	}
	for _, page := range document.Pages {
		maps.Pages[page.Key] = page
		maps.Nodes[page.Key] = StudioDocumentNode{Key: page.Key, Kind: StudioDocumentNodePage, Label: page.Label, Summary: page.Summary, Status: page.Status, Href: firstNonEmpty(page.Href, page.Path), View: page.View, Metrics: page.Metrics, Tags: page.Tags}
	}
	for _, content := range document.Content {
		maps.Content[content.Key] = content
		maps.Nodes[content.Key] = StudioDocumentNode{Key: content.Key, Kind: StudioDocumentNodeContent, Label: content.Label, Summary: content.Summary, Status: content.Status, Href: content.Href, View: content.View, Metrics: content.Metrics, Tags: content.Tags}
	}
	for _, style := range document.Styles {
		maps.Styles[style.Key] = style
		maps.Nodes[style.Key] = StudioDocumentNode{Key: style.Key, Kind: StudioDocumentNodeStyle, Label: style.Label, Summary: style.Summary, Status: style.Status, Href: style.Href, View: style.View, Metrics: style.Metrics, Tags: style.Tags}
	}
	for _, flow := range document.Flows {
		maps.Flows[flow.Key] = flow
		maps.Nodes[flow.Key] = StudioDocumentNode{Key: flow.Key, Kind: StudioDocumentNodeFlow, Label: flow.Label, Summary: flow.Summary, Status: flow.Status, Href: firstNonEmpty(flow.Href, flow.Route), View: flow.View, Metrics: flow.Metrics, Tags: flow.Tags}
	}
	for _, release := range document.Releases {
		maps.Releases[release.Key] = release
		maps.Nodes[release.Key] = StudioDocumentNode{Key: release.Key, Kind: StudioDocumentNodeRelease, Label: release.Label, Summary: release.Summary, Status: release.Status, Href: release.Href, View: release.View, Metrics: release.Metrics, Tags: release.Tags}
	}
	for _, edge := range normalizeStudioDocumentEdges(document.Edges, maps.Nodes) {
		maps.Edges[edge.Key] = edge
	}
	return maps
}

func (document StudioDocument) SiteCanvas() ([]SiteCanvasNode, []SiteCanvasEdge) {
	document = NormalizeStudioDocument(document)
	nodes := make([]SiteCanvasNode, 0, len(document.Pages)+len(document.Content)+len(document.Styles)+len(document.Flows)+len(document.Releases))
	for _, node := range document.ViewMaps().orderedNodes(document) {
		nodes = append(nodes, siteCanvasNodeFromStudioNode(node, len(nodes)))
	}
	edges := make([]SiteCanvasEdge, 0, len(document.Edges))
	for _, edge := range document.Edges {
		edges = append(edges, SiteCanvasEdge{
			Key:   edge.Key,
			From:  edge.From,
			To:    edge.To,
			Kind:  edge.Kind,
			Label: edge.Label,
		})
	}
	return normalizeSiteCanvasNodes(nodes), normalizeSiteCanvasEdges(edges, nodes)
}

func (document StudioDocument) SiteCanvasOptions() SiteCanvasOptions {
	document = NormalizeStudioDocument(document)
	nodes, edges := document.SiteCanvas()
	return SiteCanvasOptions{
		Kicker:  "Document",
		Title:   document.Label,
		Summary: document.Summary,
		Nodes:   nodes,
		Edges:   edges,
	}
}

func (maps StudioDocumentViewMaps) orderedNodes(document StudioDocument) []StudioDocumentNode {
	nodes := make([]StudioDocumentNode, 0, len(maps.Nodes))
	for _, page := range document.Pages {
		nodes = append(nodes, maps.Nodes[page.Key])
	}
	for _, content := range document.Content {
		nodes = append(nodes, maps.Nodes[content.Key])
	}
	for _, style := range document.Styles {
		nodes = append(nodes, maps.Nodes[style.Key])
	}
	for _, flow := range document.Flows {
		nodes = append(nodes, maps.Nodes[flow.Key])
	}
	for _, release := range document.Releases {
		nodes = append(nodes, maps.Nodes[release.Key])
	}
	return nodes
}

func siteCanvasNodeFromStudioNode(node StudioDocumentNode, index int) SiteCanvasNode {
	view := node.View
	if view.X == 0 && view.Y == 0 {
		view.X = 120 + float64(index%3)*320
		view.Y = 120 + float64(index/3)*220
	}
	return SiteCanvasNode{
		Key:      node.Key,
		Kind:     node.Kind,
		Label:    node.Label,
		Summary:  node.Summary,
		Status:   readinessStatusLabel(node.Status),
		Href:     node.Href,
		X:        view.X,
		Y:        view.Y,
		Width:    view.Width,
		Height:   view.Height,
		Selected: view.Selected,
		Metrics:  node.Metrics,
		Tags:     node.Tags,
	}
}

func normalizeStudioPages(pages []StudioPage) []StudioPage {
	out := make([]StudioPage, 0, len(pages))
	seen := map[string]bool{}
	for index, page := range pages {
		page.Key = normalizeKey(firstNonEmpty(page.Key, page.Label, page.Path, fmt.Sprintf("page-%d", index+1)))
		if page.Key == "" || seen[page.Key] {
			continue
		}
		seen[page.Key] = true
		page.Label = strings.TrimSpace(firstNonEmpty(page.Label, page.Key))
		page.Path = strings.TrimSpace(page.Path)
		page.Summary = strings.TrimSpace(page.Summary)
		page.Status = normalizeReadinessStatus(page.Status)
		page.ParentKey = normalizeKey(page.ParentKey)
		page.ContentKeys = normalizeStudioDocumentKeys(page.ContentKeys)
		page.StyleKeys = normalizeStudioDocumentKeys(page.StyleKeys)
		page.FlowKeys = normalizeStudioDocumentKeys(page.FlowKeys)
		page.ReleaseKeys = normalizeStudioDocumentKeys(page.ReleaseKeys)
		page.Href = strings.TrimSpace(page.Href)
		page.Metrics = normalizeMetrics(page.Metrics)
		page.Tags = normalizeStudioDocumentTags(page.Tags)
		out = append(out, page)
	}
	return out
}

func normalizeStudioContent(items []StudioContent) []StudioContent {
	out := make([]StudioContent, 0, len(items))
	seen := map[string]bool{}
	for index, item := range items {
		item.Key = normalizeKey(firstNonEmpty(item.Key, item.Label, fmt.Sprintf("content-%d", index+1)))
		if item.Key == "" || seen[item.Key] {
			continue
		}
		seen[item.Key] = true
		item.Label = strings.TrimSpace(firstNonEmpty(item.Label, item.Key))
		item.Kind = normalizeKey(firstNonEmpty(item.Kind, "content"))
		item.Summary = strings.TrimSpace(item.Summary)
		item.Status = normalizeReadinessStatus(item.Status)
		item.Href = strings.TrimSpace(item.Href)
		item.Metrics = normalizeMetrics(item.Metrics)
		item.Tags = normalizeStudioDocumentTags(item.Tags)
		out = append(out, item)
	}
	return out
}

func normalizeStudioStyles(styles []StudioStyle) []StudioStyle {
	out := make([]StudioStyle, 0, len(styles))
	seen := map[string]bool{}
	for index, style := range styles {
		style.Key = normalizeKey(firstNonEmpty(style.Key, style.Label, fmt.Sprintf("style-%d", index+1)))
		if style.Key == "" || seen[style.Key] {
			continue
		}
		seen[style.Key] = true
		style.Label = strings.TrimSpace(firstNonEmpty(style.Label, style.Key))
		style.Scope = normalizeKey(firstNonEmpty(style.Scope, "site"))
		style.Summary = strings.TrimSpace(style.Summary)
		style.Status = normalizeReadinessStatus(style.Status)
		style.Href = strings.TrimSpace(style.Href)
		style.Tokens = normalizeStudioStyleTokens(style.Tokens)
		style.Metrics = normalizeMetrics(style.Metrics)
		style.Tags = normalizeStudioDocumentTags(style.Tags)
		out = append(out, style)
	}
	return out
}

func normalizeStudioFlows(flows []StudioFlow) []StudioFlow {
	out := make([]StudioFlow, 0, len(flows))
	seen := map[string]bool{}
	for index, flow := range flows {
		flow.Key = normalizeKey(firstNonEmpty(flow.Key, flow.Label, flow.Route, fmt.Sprintf("flow-%d", index+1)))
		if flow.Key == "" || seen[flow.Key] {
			continue
		}
		seen[flow.Key] = true
		flow.Label = strings.TrimSpace(firstNonEmpty(flow.Label, flow.Key))
		flow.Trigger = normalizeKey(flow.Trigger)
		flow.Route = strings.TrimSpace(flow.Route)
		flow.Summary = strings.TrimSpace(flow.Summary)
		flow.Status = normalizeReadinessStatus(flow.Status)
		if flow.StepCount < 0 {
			flow.StepCount = 0
		}
		flow.Href = strings.TrimSpace(flow.Href)
		flow.Metrics = normalizeMetrics(flow.Metrics)
		flow.Tags = normalizeStudioDocumentTags(flow.Tags)
		out = append(out, flow)
	}
	return out
}

func normalizeStudioReleases(releases []StudioRelease) []StudioRelease {
	out := make([]StudioRelease, 0, len(releases))
	seen := map[string]bool{}
	for index, release := range releases {
		release.Key = normalizeKey(firstNonEmpty(release.Key, release.Label, fmt.Sprintf("release-%d", index+1)))
		if release.Key == "" || seen[release.Key] {
			continue
		}
		seen[release.Key] = true
		release.Label = strings.TrimSpace(firstNonEmpty(release.Label, release.Key))
		release.Summary = strings.TrimSpace(release.Summary)
		release.Status = normalizeReadinessStatus(release.Status)
		release.Href = strings.TrimSpace(release.Href)
		release.Metrics = normalizeMetrics(release.Metrics)
		release.Tags = normalizeStudioDocumentTags(release.Tags)
		out = append(out, release)
	}
	return out
}

func inferredStudioDocumentEdges(document StudioDocument) []StudioDocumentEdge {
	edges := []StudioDocumentEdge{}
	for _, page := range document.Pages {
		if page.ParentKey != "" {
			edges = append(edges, StudioDocumentEdge{From: page.ParentKey, To: page.Key, Kind: "child", Label: "Child page"})
		}
		for _, key := range page.ContentKeys {
			edges = append(edges, StudioDocumentEdge{From: page.Key, To: key, Kind: "content", Label: "Uses content"})
		}
		for _, key := range page.StyleKeys {
			edges = append(edges, StudioDocumentEdge{From: key, To: page.Key, Kind: "style", Label: "Styles page"})
		}
		for _, key := range page.FlowKeys {
			edges = append(edges, StudioDocumentEdge{From: page.Key, To: key, Kind: "flow", Label: "Starts flow"})
		}
		for _, key := range page.ReleaseKeys {
			edges = append(edges, StudioDocumentEdge{From: page.Key, To: key, Kind: "release", Label: "Ships in release"})
		}
	}
	return edges
}

func normalizeStudioDocumentEdges(edges []StudioDocumentEdge, nodes map[string]StudioDocumentNode) []StudioDocumentEdge {
	out := make([]StudioDocumentEdge, 0, len(edges))
	seen := map[string]bool{}
	for index, edge := range edges {
		edge.From = normalizeKey(edge.From)
		edge.To = normalizeKey(edge.To)
		if edge.From == "" || edge.To == "" {
			continue
		}
		if _, ok := nodes[edge.From]; !ok {
			continue
		}
		if _, ok := nodes[edge.To]; !ok {
			continue
		}
		edge.Kind = normalizeKey(firstNonEmpty(edge.Kind, "link"))
		edge.Key = normalizeKey(firstNonEmpty(edge.Key, edge.From+"-"+edge.Kind+"-"+edge.To, fmt.Sprintf("edge-%d", index+1)))
		if edge.Key == "" || seen[edge.Key] {
			continue
		}
		seen[edge.Key] = true
		edge.Label = strings.TrimSpace(edge.Label)
		out = append(out, edge)
	}
	return out
}

func normalizeStudioDocumentKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	seen := map[string]bool{}
	for _, key := range keys {
		key = normalizeKey(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

func normalizeStudioDocumentTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

func normalizeStudioStyleTokens(tokens map[string]string) map[string]string {
	if len(tokens) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range tokens {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
