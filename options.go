package studio

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-cms/lifecycle"
	"m31labs.dev/gosx-cms/media"
)

type Options struct {
	Title         string
	PreviewURL    string
	SaveAction    string
	RestoreAction string
	BlockCatalog  []blockstudio.Definition
	Media         []media.Asset
	Revisions     []lifecycle.Revision
	Navigation    []Section
	Metrics       []Metric
	Left          []Panel
	Right         []Panel
	Modes         []Mode
	Viewports     []Viewport
	Canvas        CanvasSurface
	Actions       []Action
	Readiness     ShellReadiness
	Extras        map[string]any
}

type Action struct {
	Key     string
	Label   string
	Href    string
	Primary bool
}

type Panel struct {
	Key      string
	Label    string
	Summary  string
	Children []gosx.Node
}

type Section struct {
	Key     string
	Label   string
	Summary string
	Actions []Action
}

type Metric struct {
	Key   string
	Label string
	Value any
}

type Mode struct {
	Key    string
	Label  string
	Active bool
}

type Viewport struct {
	Key    string
	Label  string
	Width  string
	Active bool
}

type CanvasSurface struct {
	RouteLabel     string
	SelectionLabel string
	Zoom           string
	Focus          bool
}

type BlockSummary struct {
	Key        string
	Label      string
	Summary    string
	Kind       string
	Preview    string
	DefaultOn  bool
	Locked     bool
	Repeatable bool
	Icon       string
	FieldCount int
}

type MediaSummary struct {
	ID          string
	URL         string
	Alt         string
	Filename    string
	ContentType string
	Size        int64
	Archived    bool
}

type RevisionSummary struct {
	ID            string
	ResourceKind  string
	ResourceID    string
	ResourceTitle string
	Action        string
	Summary       string
	ChangeSummary string
	ChangeCount   int
	HasDiff       bool
	Created       string
}

type Shell struct {
	Title         string
	PreviewURL    string
	SaveAction    string
	RestoreAction string
	BlockCatalog  []blockstudio.Definition
	Blocks        []BlockSummary
	BlockCount    int
	Media         []media.Asset
	MediaLibrary  []MediaSummary
	MediaCount    int
	HasMedia      bool
	Revisions     []lifecycle.Revision
	RevisionLog   []RevisionSummary
	RevisionCount int
	HasRevisions  bool
	Navigation    []Section
	Metrics       []Metric
	Left          []Panel
	Right         []Panel
	Modes         []Mode
	Viewports     []Viewport
	Canvas        CanvasSurface
	Actions       []Action
	Readiness     ShellReadiness
	Extras        map[string]any
}

func View(shell Shell) map[string]any {
	view := map[string]any{
		"title":         shell.Title,
		"previewURL":    shell.PreviewURL,
		"saveAction":    shell.SaveAction,
		"restoreAction": shell.RestoreAction,
		"blockCatalog":  shellBlockViews(shell.Blocks),
		"blockCount":    shell.BlockCount,
		"media":         shellMediaViews(shell.MediaLibrary),
		"mediaCount":    shell.MediaCount,
		"hasMedia":      shell.HasMedia,
		"revisions":     shellRevisionViews(shell.RevisionLog),
		"revisionCount": shell.RevisionCount,
		"hasRevisions":  shell.HasRevisions,
		"navigation":    shellSectionViews(shell.Navigation),
		"metrics":       shellMetricViews(shell.Metrics),
		"modes":         shellModeViews(shell.Modes),
		"viewports":     shellViewportViews(shell.Viewports),
		"canvas":        shellCanvasView(shell.Canvas),
		"actions":       shellActionViews(shell.Actions),
		"leftPanels":    shellPanelViews(shell.Left),
		"rightPanels":   shellPanelViews(shell.Right),
		"readiness":     ShellReadinessView(shell.Readiness),
	}
	for key, value := range shell.Extras {
		if _, exists := view[key]; !exists {
			view[key] = value
		}
	}
	if shell.Extras != nil {
		view["extras"] = shellCloneExtras(shell.Extras)
	}
	return view
}

func New(options Options) Shell {
	title := strings.TrimSpace(options.Title)
	if title == "" {
		title = "GoSX Studio"
	}
	previewURL := strings.TrimSpace(options.PreviewURL)
	if previewURL == "" {
		previewURL = "/"
	}
	actions := normalizeShellActions(options.Actions, options.SaveAction)
	navigation := normalizeShellSections(options.Navigation)
	metrics := NormalizeMetrics(options.Metrics)
	modes := NormalizeModes(options.Modes)
	viewports := NormalizeViewports(options.Viewports)
	canvas := normalizeShellCanvas(options.Canvas, title)
	blockCatalog := append([]blockstudio.Definition(nil), options.BlockCatalog...)
	mediaAssets := media.CloneAssets(options.Media)
	revisions := lifecycle.CloneRevisions(options.Revisions)
	return Shell{
		Title:         title,
		PreviewURL:    previewURL,
		SaveAction:    strings.TrimSpace(options.SaveAction),
		RestoreAction: strings.TrimSpace(options.RestoreAction),
		BlockCatalog:  blockCatalog,
		Blocks:        shellBlockSummaries(blockCatalog),
		BlockCount:    len(blockCatalog),
		Media:         mediaAssets,
		MediaLibrary:  shellMediaSummaries(mediaAssets),
		MediaCount:    len(mediaAssets),
		HasMedia:      len(mediaAssets) > 0,
		Revisions:     revisions,
		RevisionLog:   shellRevisionSummaries(revisions),
		RevisionCount: len(revisions),
		HasRevisions:  len(revisions) > 0,
		Navigation:    navigation,
		Metrics:       metrics,
		Left:          normalizeShellPanels(options.Left),
		Right:         normalizeShellPanels(options.Right),
		Modes:         modes,
		Viewports:     viewports,
		Canvas:        canvas,
		Actions:       actions,
		Readiness:     NormalizeShellReadiness(options.Readiness),
		Extras:        shellCloneExtras(options.Extras),
	}
}

func LinkAction(key, label, href string) Action {
	return Action{Key: key, Label: label, Href: href}
}

func PrimaryAction(key, label, href string) Action {
	return Action{Key: key, Label: label, Href: href, Primary: true}
}

func NewPanel(key, label, summary string, children ...gosx.Node) Panel {
	return Panel{Key: key, Label: label, Summary: summary, Children: children}
}

func NewSection(key, label string, actions ...Action) Section {
	return Section{Key: key, Label: label, Actions: actions}
}

func NewMetric(key, label string, value any) Metric {
	return Metric{Key: key, Label: label, Value: value}
}

func NewMode(key, label string, active bool) Mode {
	return Mode{Key: key, Label: label, Active: active}
}

func NewViewport(key, label, width string, active bool) Viewport {
	return Viewport{Key: key, Label: label, Width: width, Active: active}
}

func shellActionViews(actions []Action) []map[string]any {
	out := make([]map[string]any, 0, len(actions))
	for _, action := range actions {
		className := "button button--secondary"
		if action.Primary {
			className = "button button--primary"
		}
		out = append(out, map[string]any{
			"key":     action.Key,
			"label":   action.Label,
			"href":    action.Href,
			"primary": action.Primary,
			"class":   className,
		})
	}
	return out
}

func shellSectionViews(sections []Section) []map[string]any {
	out := make([]map[string]any, 0, len(sections))
	for _, section := range sections {
		out = append(out, map[string]any{
			"key":        section.Key,
			"label":      section.Label,
			"summary":    section.Summary,
			"hasSummary": section.Summary != "",
			"actions":    shellActionViews(section.Actions),
		})
	}
	return out
}

func shellMetricViews(metrics []Metric) []map[string]any {
	out := make([]map[string]any, 0, len(metrics))
	for _, metric := range metrics {
		out = append(out, map[string]any{
			"key":   metric.Key,
			"label": metric.Label,
			"value": metric.Value,
		})
	}
	return out
}

func shellModeViews(modes []Mode) []map[string]any {
	out := make([]map[string]any, 0, len(modes))
	for _, mode := range modes {
		out = append(out, map[string]any{
			"key":     mode.Key,
			"label":   mode.Label,
			"active":  mode.Active,
			"pressed": BoolAttr(mode.Active),
		})
	}
	return out
}

func shellViewportViews(viewports []Viewport) []map[string]any {
	out := make([]map[string]any, 0, len(viewports))
	for _, viewport := range viewports {
		out = append(out, map[string]any{
			"key":     viewport.Key,
			"label":   viewport.Label,
			"width":   viewport.Width,
			"active":  viewport.Active,
			"pressed": BoolAttr(viewport.Active),
		})
	}
	return out
}

func shellCanvasView(canvas CanvasSurface) map[string]any {
	return map[string]any{
		"routeLabel":     canvas.RouteLabel,
		"selectionLabel": canvas.SelectionLabel,
		"zoom":           canvas.Zoom,
		"focus":          canvas.Focus,
	}
}

func shellPanelViews(panels []Panel) []map[string]any {
	out := make([]map[string]any, 0, len(panels))
	for _, panel := range panels {
		out = append(out, map[string]any{
			"key":        panel.Key,
			"label":      panel.Label,
			"summary":    panel.Summary,
			"hasSummary": panel.Summary != "",
		})
	}
	return out
}

func shellBlockViews(blocks []BlockSummary) []map[string]any {
	out := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, map[string]any{
			"key":        block.Key,
			"label":      block.Label,
			"summary":    block.Summary,
			"kind":       block.Kind,
			"preview":    block.Preview,
			"defaultOn":  block.DefaultOn,
			"locked":     block.Locked,
			"repeatable": block.Repeatable,
			"icon":       block.Icon,
			"fieldCount": block.FieldCount,
		})
	}
	return out
}

func shellMediaViews(media []MediaSummary) []map[string]any {
	out := make([]map[string]any, 0, len(media))
	for _, asset := range media {
		out = append(out, map[string]any{
			"id":          asset.ID,
			"url":         asset.URL,
			"alt":         asset.Alt,
			"filename":    asset.Filename,
			"contentType": asset.ContentType,
			"size":        asset.Size,
			"archived":    asset.Archived,
		})
	}
	return out
}

func shellRevisionViews(revisions []RevisionSummary) []map[string]any {
	out := make([]map[string]any, 0, len(revisions))
	for _, revision := range revisions {
		out = append(out, map[string]any{
			"id":            revision.ID,
			"resourceKind":  revision.ResourceKind,
			"resourceID":    revision.ResourceID,
			"resourceTitle": revision.ResourceTitle,
			"action":        revision.Action,
			"summary":       revision.Summary,
			"changeSummary": revision.ChangeSummary,
			"changeCount":   revision.ChangeCount,
			"hasDiff":       revision.HasDiff,
			"created":       revision.Created,
		})
	}
	return out
}

func Render(shell Shell) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-studio")),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "gosx-studio__toolbar")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-studio__identity")),
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "gosx-studio__kicker")), gosx.Text("Studio")),
				gosx.El("h1", nil, gosx.Text(shell.Title)),
			),
			shellRenderModebar(shell.Modes),
			shellRenderMetricStrip(shell.Metrics),
			shellRenderActions(shell.Actions),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-studio__layout")),
			shellRenderPanelColumn("gosx-studio__left", shell.Left),
			gosx.El("main", gosx.Attrs(gosx.Attr("class", "gosx-studio__canvas")),
				shellRenderCanvasBar(shell),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-studio__frame")),
					gosx.El("iframe", gosx.Attrs(
						gosx.Attr("src", shell.PreviewURL),
						gosx.Attr("title", shell.Title+" preview"),
						gosx.Attr("data-studio-preview", "true"),
					)),
				),
				shellRenderReadiness(shell.Readiness),
			),
			shellRenderPanelColumn("gosx-studio__right", shell.Right),
		),
	)
}

func shellRenderModebar(modes []Mode) gosx.Node {
	nodes := make([]gosx.Node, 0, len(modes))
	for _, mode := range modes {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-mode-control", mode.Key),
			gosx.Attr("aria-pressed", BoolAttr(mode.Active)),
		), gosx.Text(mode.Label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "gosx-studio__modebar"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Editor mode"),
	), gosx.Fragment(nodes...))
}

func shellRenderMetricStrip(metrics []Metric) gosx.Node {
	nodes := make([]gosx.Node, 0, len(metrics))
	for _, metric := range metrics {
		nodes = append(nodes, gosx.El("span", gosx.Attrs(
			gosx.Attr("data-studio-metric", metric.Key),
		), gosx.Text(metric.Label+": "+strings.TrimSpace(strings.ReplaceAll(FmtAny(metric.Value), "\n", " ")))))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "gosx-studio__metrics"),
		gosx.Attr("aria-label", "Workspace metrics"),
	), gosx.Fragment(nodes...))
}

func shellRenderCanvasBar(shell Shell) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-studio__canvas-bar")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "gosx-studio__kicker")), gosx.Text("Canvas")),
			gosx.El("strong", nil, gosx.Text(shell.Canvas.RouteLabel)),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("data-studio-selection-label", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(shell.Canvas.SelectionLabel)),
		shellRenderViewports(shell.Viewports),
	)
}

func shellRenderViewports(viewports []Viewport) gosx.Node {
	nodes := make([]gosx.Node, 0, len(viewports))
	for _, viewport := range viewports {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-viewport", viewport.Key),
			gosx.Attr("aria-pressed", BoolAttr(viewport.Active)),
		), gosx.Text(viewport.Label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "gosx-studio__viewports"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Preview viewport"),
	), gosx.Fragment(nodes...))
}

func shellRenderReadiness(readiness ShellReadiness) gosx.Node {
	readiness = NormalizeShellReadiness(readiness)
	if len(readiness.Items) == 0 {
		return gosx.Fragment()
	}
	items := make([]gosx.Node, 0, len(readiness.Items))
	for _, item := range readiness.Items {
		children := []gosx.Node{
			gosx.El("strong", nil, gosx.Text(item.Label)),
			gosx.El("span", nil, gosx.Text(ShellReadinessStatusLabel(item.Status))),
		}
		if item.Summary != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(item.Summary)))
		}
		items = append(items, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", "gosx-studio__readiness-item gosx-studio__readiness-item--"+string(item.Status)),
			gosx.Attr("data-readiness-key", item.Key),
		), gosx.Fragment(children...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "gosx-studio__activity"),
		gosx.Attr("aria-label", "Readiness"),
	), gosx.Fragment(items...))
}

func shellRenderActions(actions []Action) gosx.Node {
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		className := "button button--secondary"
		if action.Primary {
			className = "button button--primary"
		}
		nodes = append(nodes, gosx.El("a", gosx.Attrs(
			gosx.Attr("class", className),
			gosx.Attr("href", action.Href),
			gosx.Attr("data-action-key", action.Key),
		), gosx.Text(action.Label)))
	}
	return gosx.El("nav", gosx.Attrs(gosx.Attr("class", "gosx-studio__actions")), gosx.Fragment(nodes...))
}

func shellRenderPanelColumn(className string, panels []Panel) gosx.Node {
	nodes := make([]gosx.Node, 0, len(panels))
	for _, panel := range panels {
		children := []gosx.Node{gosx.El("h2", nil, gosx.Text(panel.Label))}
		if panel.Summary != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(panel.Summary)))
		}
		children = append(children, panel.Children...)
		nodes = append(nodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "gosx-studio__panel"),
			gosx.Attr("data-panel-key", panel.Key),
		), gosx.Fragment(children...)))
	}
	return gosx.El("aside", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(nodes...))
}

func normalizeShellActions(actions []Action, saveAction string) []Action {
	out := make([]Action, 0, len(actions)+1)
	for _, action := range actions {
		action.Key = NormalizeKey(action.Key)
		action.Label = strings.TrimSpace(action.Label)
		action.Href = strings.TrimSpace(action.Href)
		if action.Key == "" || action.Label == "" {
			continue
		}
		out = append(out, action)
	}
	if strings.TrimSpace(saveAction) != "" && !shellHasAction(out, "save") {
		out = append([]Action{{Key: "save", Label: "Save", Href: strings.TrimSpace(saveAction), Primary: true}}, out...)
	}
	return out
}

func normalizeShellSections(sections []Section) []Section {
	out := make([]Section, 0, len(sections))
	for _, section := range sections {
		section.Key = NormalizeKey(section.Key)
		section.Label = strings.TrimSpace(section.Label)
		section.Summary = strings.TrimSpace(section.Summary)
		section.Actions = normalizeShellActions(section.Actions, "")
		if section.Key == "" || section.Label == "" {
			continue
		}
		out = append(out, section)
	}
	return out
}

func NormalizeMetrics(metrics []Metric) []Metric {
	out := make([]Metric, 0, len(metrics))
	for _, metric := range metrics {
		metric.Key = NormalizeKey(metric.Key)
		metric.Label = strings.TrimSpace(metric.Label)
		if metric.Key == "" || metric.Label == "" {
			continue
		}
		out = append(out, metric)
	}
	return out
}

func NormalizeModes(modes []Mode) []Mode {
	if len(modes) == 0 {
		modes = []Mode{
			{Key: "structure", Label: "Structure", Active: true},
			{Key: "content", Label: "Content"},
			{Key: "style", Label: "Style"},
			{Key: "preview", Label: "Preview"},
		}
	}
	out := make([]Mode, 0, len(modes))
	hasActive := false
	for _, mode := range modes {
		mode.Key = NormalizeKey(mode.Key)
		mode.Label = strings.TrimSpace(mode.Label)
		if mode.Key == "" || mode.Label == "" {
			continue
		}
		if mode.Active {
			if hasActive {
				mode.Active = false
			} else {
				hasActive = true
			}
		}
		out = append(out, mode)
	}
	if len(out) > 0 && !hasActive {
		out[0].Active = true
	}
	return out
}

func NormalizeViewports(viewports []Viewport) []Viewport {
	if len(viewports) == 0 {
		viewports = []Viewport{
			{Key: "desktop", Label: "Desktop", Width: "100%", Active: true},
			{Key: "tablet", Label: "Tablet", Width: "48rem"},
			{Key: "mobile", Label: "Mobile", Width: "24rem"},
		}
	}
	out := make([]Viewport, 0, len(viewports))
	hasActive := false
	for _, viewport := range viewports {
		viewport.Key = NormalizeKey(viewport.Key)
		viewport.Label = strings.TrimSpace(viewport.Label)
		viewport.Width = strings.TrimSpace(viewport.Width)
		if viewport.Key == "" || viewport.Label == "" {
			continue
		}
		if viewport.Active {
			if hasActive {
				viewport.Active = false
			} else {
				hasActive = true
			}
		}
		out = append(out, viewport)
	}
	if len(out) > 0 && !hasActive {
		out[0].Active = true
	}
	return out
}

func normalizeShellCanvas(canvas CanvasSurface, fallbackTitle string) CanvasSurface {
	canvas.RouteLabel = strings.TrimSpace(canvas.RouteLabel)
	canvas.SelectionLabel = strings.TrimSpace(canvas.SelectionLabel)
	canvas.Zoom = NormalizeKey(canvas.Zoom)
	if canvas.RouteLabel == "" {
		canvas.RouteLabel = FirstNonEmpty(fallbackTitle, "Preview")
	}
	if canvas.SelectionLabel == "" {
		canvas.SelectionLabel = "No selection"
	}
	if canvas.Zoom == "" {
		canvas.Zoom = "fit"
	}
	return canvas
}

func normalizeShellPanels(panels []Panel) []Panel {
	out := make([]Panel, 0, len(panels))
	for _, panel := range panels {
		panel.Key = NormalizeKey(panel.Key)
		panel.Label = strings.TrimSpace(panel.Label)
		panel.Summary = strings.TrimSpace(panel.Summary)
		if panel.Key == "" || panel.Label == "" {
			continue
		}
		out = append(out, panel)
	}
	return out
}

func shellBlockSummaries(blocks []blockstudio.Definition) []BlockSummary {
	out := make([]BlockSummary, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, BlockSummary{
			Key:        block.Key,
			Label:      block.Label,
			Summary:    block.Summary,
			Kind:       block.Kind,
			Preview:    block.Preview,
			DefaultOn:  block.DefaultOn,
			Locked:     block.Locked,
			Repeatable: block.Repeatable,
			Icon:       block.Icon,
			FieldCount: len(block.Fields),
		})
	}
	return out
}

func shellMediaSummaries(assets []media.Asset) []MediaSummary {
	out := make([]MediaSummary, 0, len(assets))
	for _, asset := range assets {
		out = append(out, MediaSummary{
			ID:          asset.ID,
			URL:         asset.URL,
			Alt:         asset.Alt,
			Filename:    asset.Filename,
			ContentType: asset.ContentType,
			Size:        asset.Size,
			Archived:    asset.ArchivedAt != nil,
		})
	}
	return out
}

func shellRevisionSummaries(revisions []lifecycle.Revision) []RevisionSummary {
	out := make([]RevisionSummary, 0, len(revisions))
	for index, revision := range revisions {
		created := ""
		if !revision.Created.IsZero() {
			created = revision.Created.Format("2006-01-02T15:04:05Z07:00")
		}
		summary := RevisionSummary{
			ID:            revision.ID,
			ResourceKind:  revision.ResourceKind,
			ResourceID:    revision.ResourceID,
			ResourceTitle: revision.ResourceTitle,
			Action:        revision.Action,
			Summary:       revision.Summary,
			Created:       created,
		}
		if index+1 < len(revisions) {
			if diff, err := lifecycle.DiffRevisions(revisions[index+1], revision); err == nil {
				summary.ChangeSummary = diff.Summary
				summary.ChangeCount = len(diff.Changes)
				summary.HasDiff = true
			}
		}
		out = append(out, summary)
	}
	return out
}

func shellCloneExtras(extras map[string]any) map[string]any {
	if extras == nil {
		return nil
	}
	out := make(map[string]any, len(extras))
	for key, value := range extras {
		out[key] = value
	}
	return out
}

func shellHasAction(actions []Action, key string) bool {
	key = NormalizeKey(key)
	for _, action := range actions {
		if NormalizeKey(action.Key) == key {
			return true
		}
	}
	return false
}
