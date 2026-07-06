package studio

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx"
)

type MetricCardsOptions struct {
	Class     string
	CardClass string
}

type FlowStorageSummary struct {
	Kind          string
	Class         string
	Label         string
	StatusLabel   string
	Detail        string
	Path          string
	DocumentCount int
}

type FlowStoragePanelOptions struct {
	Class       string
	HeadClass   string
	KickerClass string
	MetaClass   string
	Kicker      string
}

type LifecycleActivityItem struct {
	Summary string
	Action  string
	Created string
}

type LifecycleControlsPanelOptions struct {
	Class             string
	FieldClass        string
	ButtonRowClass    string
	ActivityClass     string
	ActivityCardClass string
	Title             string
	PublishAtName     string
	ApproveAction     string
	ScheduleAction    string
	ProcessDueAction  string
	ApproveLabel      string
	ScheduleLabel     string
	ProcessDueLabel   string
	Activity          []LifecycleActivityItem
}

type StudioMapPanelOptions struct {
	Class       string
	GridClass   string
	PanelClass  string
	CardClass   string
	Title       string
	PanelsTitle string
}

type WorkbenchPreviewPanelOptions struct {
	Class        string
	Title        string
	URL          string
	IFrameTitle  string
	OpenLabel    string
	OpenClass    string
	FrameClass   string
	ToolbarClass string
	MetaClass    string
	StatusClass  string
}

type WorkbenchSectionNavItem struct {
	Key     string
	Label   string
	Target  string
	Href    string
	Summary string
	Class   string
	Active  bool
}

type WorkbenchSectionNavigatorOptions struct {
	Class        string
	HeadingClass string
	NavClass     string
	Kicker       string
	Title        string
	Label        string
	Mode         string
}

type WorkbenchSectionOptions struct {
	Key          string
	ID           string
	Class        string
	HeadingClass string
	Kicker       string
	Title        string
	Summary      string
	Children     []gosx.Node
}

type WorkbenchSummaryToolbarOptions struct {
	Class                  string
	ActionsClass           string
	Title                  string
	Summary                string
	SaveButtonClass        string
	SaveButtonLabel        string
	DisableHistoryControls bool
	CommandPaletteNode     gosx.Node
	SaveStatusNode         gosx.Node
	HistoryControlsNode    gosx.Node
	Controls               []gosx.Node
	Actions                []gosx.Node
}

func RenderMetricCards(metrics []Metric, options MetricCardsOptions) gosx.Node {
	cards := make([]gosx.Node, 0, len(metrics))
	for _, metric := range normalizeMetrics(metrics) {
		cards = append(cards, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.CardClass, "stat-card")),
			gosx.Attr("data-studio-metric", metric.Key),
		),
			gosx.El("span", nil, gosx.Text(metric.Label)),
			gosx.El("strong", nil, gosx.Text(fmtAny(metric.Value))),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.Class, "stat-grid"))), gosx.Fragment(cards...))
}

func RenderWorkbenchSectionNavigator(items []WorkbenchSectionNavItem, options WorkbenchSectionNavigatorOptions) gosx.Node {
	links := make([]gosx.Node, 0, len(items))
	for _, item := range normalizeWorkbenchSectionNavItems(items) {
		className := strings.TrimSpace(item.Class)
		if item.Active && !strings.Contains(" "+className+" ", " is-active ") {
			className = strings.TrimSpace(className + " is-active")
		}
		attrs := []any{
			gosx.Attr("href", firstNonEmpty(item.Href, "#"+item.Target)),
			gosx.Attr("data-studio-workbench-section-link", item.Key),
		}
		if className != "" {
			attrs = append(attrs, gosx.Attr("class", className))
		}
		children := []gosx.Node{gosx.El("span", nil, gosx.Text(item.Label))}
		if item.Summary != "" {
			children = append(children, gosx.El("small", nil, gosx.Text(item.Summary)))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Fragment(children...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.Class, "studio-workbench-nav")),
		gosx.Attr("data-studio-workbench-section-nav", "true"),
		gosx.Attr("data-studio-mode-panel", firstNonEmpty(options.Mode, "workspace")),
	),
		RenderPanelHeading(PanelHeadingOptions{
			Class:  options.HeadingClass,
			Kicker: firstNonEmpty(options.Kicker, "Workspace"),
			Title:  firstNonEmpty(options.Title, "Sections"),
		}),
		gosx.El("nav", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.NavClass, "studio-workbench-nav__list")),
			gosx.Attr("aria-label", firstNonEmpty(options.Label, "Workspace sections")),
		), gosx.Fragment(links...)),
	)
}

func RenderWorkbenchSection(options WorkbenchSectionOptions) gosx.Node {
	key := normalizeKey(firstNonEmpty(options.Key, options.Title, options.ID))
	id := strings.TrimSpace(options.ID)
	if id == "" && key != "" {
		id = "studio-section-" + key
	}
	children := make([]gosx.Node, 0, len(options.Children)+1)
	if strings.TrimSpace(options.Title) != "" || strings.TrimSpace(options.Summary) != "" || strings.TrimSpace(options.Kicker) != "" {
		headingChildren := []gosx.Node{}
		if strings.TrimSpace(options.Kicker) != "" {
			headingChildren = append(headingChildren, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(options.Kicker)))
		}
		if strings.TrimSpace(options.Title) != "" {
			headingChildren = append(headingChildren, gosx.El("h2", nil, gosx.Text(options.Title)))
		}
		if strings.TrimSpace(options.Summary) != "" {
			headingChildren = append(headingChildren, gosx.El("p", nil, gosx.Text(options.Summary)))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.HeadingClass, "studio-workbench-section__head"))), gosx.Fragment(headingChildren...)))
	}
	children = append(children, options.Children...)
	attrs := []any{
		gosx.Attr("class", firstNonEmpty(options.Class, "studio-workbench-section")),
		gosx.Attr("data-studio-workbench-section", key),
	}
	if id != "" {
		attrs = append(attrs, gosx.Attr("id", id))
	}
	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func normalizeWorkbenchSectionNavItems(items []WorkbenchSectionNavItem) []WorkbenchSectionNavItem {
	out := make([]WorkbenchSectionNavItem, 0, len(items))
	for _, item := range items {
		item.Key = normalizeKey(firstNonEmpty(item.Key, item.Target, item.Label))
		item.Label = strings.TrimSpace(item.Label)
		item.Target = strings.Trim(strings.TrimSpace(item.Target), "#")
		item.Href = strings.TrimSpace(item.Href)
		item.Summary = strings.TrimSpace(item.Summary)
		item.Class = strings.TrimSpace(item.Class)
		if item.Label == "" || item.Key == "" {
			continue
		}
		if item.Target == "" && strings.HasPrefix(item.Href, "#") {
			item.Target = strings.TrimPrefix(item.Href, "#")
		}
		if item.Target == "" {
			item.Target = "studio-section-" + item.Key
		}
		out = append(out, item)
	}
	return out
}

func RenderFlowStoragePanel(summary FlowStorageSummary, options FlowStoragePanelOptions) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.HeadClass, "studio-storage__head"))),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.KickerClass, "studio-storage__kicker"))), gosx.Text(firstNonEmpty(options.Kicker, "Storage"))),
				gosx.El("h2", nil, gosx.Text(summary.Label)),
			),
			gosx.El("output", nil, gosx.Text(summary.StatusLabel)),
		),
		gosx.El("p", nil, gosx.Text(summary.Detail)),
	}
	meta := []gosx.Node{
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(fmtAny(summary.DocumentCount))),
			gosx.Text(" flow documents"),
		),
	}
	if path := strings.TrimSpace(summary.Path); path != "" {
		meta = append(meta, gosx.El("code", nil, gosx.Text(path)))
	}
	children = append(children, gosx.El("div", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.MetaClass, "studio-storage__meta")),
		gosx.Attr("aria-label", "Flow storage summary"),
	), gosx.Fragment(meta...)))
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.Class, summary.Class, "studio-panel studio-storage")),
		gosx.Attr("data-studio-flow-storage", summary.Kind),
	), gosx.Fragment(children...))
}

func RenderLifecycleControlsPanel(state LifecycleReviewState, options LifecycleControlsPanelOptions) gosx.Node {
	if strings.TrimSpace(options.ApproveAction) == "" && strings.TrimSpace(options.ScheduleAction) == "" && strings.TrimSpace(options.ProcessDueAction) == "" {
		return gosx.Fragment()
	}
	buttons := []gosx.Node{}
	if strings.TrimSpace(options.ApproveAction) != "" {
		buttons = append(buttons, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", options.ApproveAction),
		), gosx.Text(firstNonEmpty(options.ApproveLabel, "Approve publish"))))
	}
	if strings.TrimSpace(options.ScheduleAction) != "" {
		buttons = append(buttons, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--primary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", options.ScheduleAction),
		), gosx.Text(firstNonEmpty(options.ScheduleLabel, "Schedule publish"))))
	}
	if strings.TrimSpace(options.ProcessDueAction) != "" {
		buttons = append(buttons, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", options.ProcessDueAction),
		), gosx.Text(firstNonEmpty(options.ProcessDueLabel, "Run due publishes"))))
	}
	children := []gosx.Node{
		gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, "Lifecycle"))),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.FieldClass, "field"))),
			gosx.El("span", nil, gosx.Text("Publish time")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "datetime-local"),
				gosx.Attr("name", firstNonEmpty(options.PublishAtName, "publishAt")),
				gosx.Attr("value", LifecycleScheduleInputValue(state, nil)),
			)),
			gosx.El("small", nil, gosx.Text(LifecycleScheduleHelp(state, nil))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.ButtonRowClass, "button-row"))), gosx.Fragment(buttons...)),
	}
	if len(options.Activity) > 0 {
		events := make([]gosx.Node, 0, len(options.Activity))
		for _, event := range options.Activity {
			events = append(events, gosx.El("article", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.ActivityCardClass, "list-card"))),
				gosx.El("strong", nil, gosx.Text(event.Summary)),
				gosx.El("span", nil, gosx.Text(strings.TrimSpace(event.Action+" / "+event.Created))),
			))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.ActivityClass, "stack"))), gosx.Fragment(events...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.Class, "studio-panel"))), gosx.Fragment(children...))
}

func RenderStudioMapPanel(shell Shell, options StudioMapPanelOptions) gosx.Node {
	navItems := make([]gosx.Node, 0, len(shell.Navigation))
	for _, section := range shell.Navigation {
		navItems = append(navItems, gosx.El("article", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.CardClass, "list-card"))),
			gosx.El("strong", nil, gosx.Text(section.Label)),
			gosx.El("span", nil, gosx.Text(section.Summary)),
		))
	}
	panelItems := make([]gosx.Node, 0, len(shell.Left)+len(shell.Right))
	for _, panel := range append(append([]Panel{}, shell.Left...), shell.Right...) {
		panelItems = append(panelItems, gosx.El("article", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.CardClass, "list-card"))),
			gosx.El("strong", nil, gosx.Text(panel.Label)),
			gosx.El("span", nil, gosx.Text(panel.Summary)),
		))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.GridClass, options.Class, "studio-grid"))),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.PanelClass, "studio-panel"))),
			gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, "Studio map"))),
			gosx.Fragment(navItems...),
		),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.PanelClass, "studio-panel"))),
			gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.PanelsTitle, "Panels"))),
			gosx.Fragment(panelItems...),
		),
	)
}

func RenderWorkbenchPreviewPanel(shell Shell, options WorkbenchPreviewPanelOptions) gosx.Node {
	url := firstNonEmpty(options.URL, shell.PreviewURL, "/")
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.Class, "studio-panel studio-preview-panel"))),
		gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, "Preview"))),
		RenderPreviewFrame(PreviewFrameOptions{
			ShellClass:   "studio-preview-frame-shell",
			ToolbarClass: firstNonEmpty(options.ToolbarClass, "studio-preview-toolbar"),
			MetaClass:    firstNonEmpty(options.MetaClass, "studio-preview-toolbar__flow"),
			FrameClass:   firstNonEmpty(options.FrameClass, "preview-frame"),
			StatusClass:  firstNonEmpty(options.StatusClass, "studio-preview-toolbar__status"),
			OpenClass:    firstNonEmpty(options.OpenClass, "button button--secondary"),
			Kicker:       "Previewing",
			Title:        "Public site",
			URL:          url,
			IFrameTitle:  firstNonEmpty(options.IFrameTitle, "Public site preview"),
			StatusLabel:  "Ready",
			OpenLabel:    firstNonEmpty(options.OpenLabel, "Open route"),
			OpenNewTab:   true,
			DynamicTitle: true,
			DynamicRoute: true,
		}),
	)
}

func RenderWorkbenchSummaryToolbar(shell Shell, options WorkbenchSummaryToolbarOptions) gosx.Node {
	actions := []gosx.Node{}
	if !isZeroNode(options.CommandPaletteNode) {
		actions = append(actions, options.CommandPaletteNode)
	}
	if !isZeroNode(options.SaveStatusNode) {
		actions = append(actions, options.SaveStatusNode)
	}
	if !options.DisableHistoryControls {
		if !isZeroNode(options.HistoryControlsNode) {
			actions = append(actions, options.HistoryControlsNode)
		} else {
			actions = append(actions, RenderHistoryControls(HistoryControlsOptions{}))
		}
	}
	actions = append(actions, options.Actions...)
	if firstNonEmpty(options.SaveButtonLabel, "Save checkpoint") != "" {
		actions = append(actions, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.SaveButtonClass, "button button--primary")),
			gosx.Attr("type", "submit"),
			gosx.Attr("data-gosx-studio-save-button", "true"),
		), gosx.Text(firstNonEmpty(options.SaveButtonLabel, "Save checkpoint"))))
	}
	return RenderStudioToolbar(StudioToolbarOptions{
		Class:        firstNonEmpty(options.Class, "studio-toolbar"),
		ActionsClass: firstNonEmpty(options.ActionsClass, "studio-toolbar__actions"),
		Title:        firstNonEmpty(options.Title, "Authoring surfaces"),
		Summary:      firstNonEmpty(options.Summary, fmt.Sprintf("%d reusable blocks", shell.BlockCount)),
		Controls:     options.Controls,
		Actions:      actions,
	})
}

func MetricValue(metrics []Metric, key string) string {
	for _, metric := range normalizeMetrics(metrics) {
		if metric.Key == normalizeKey(key) {
			return fmtAny(metric.Value)
		}
	}
	return "0"
}
