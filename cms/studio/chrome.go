package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type ModebarOptions struct {
	Class string
	Label string
}

type MetricStripOptions struct {
	Class string
	Label string
}

type ViewportSwitcherOptions struct {
	Class string
	Label string
}

type CanvasToolsOptions struct {
	Class          string
	Label          string
	LeftCollapsed  bool
	RightCollapsed bool
	ActivityClosed bool
	FocusActive    bool
	HideActivity   bool
	HideFocus      bool
	LeftLabel      string
	RightLabel     string
	ActivityLabel  string
	FocusLabel     string
}

type ZoomLevel struct {
	Key    string
	Label  string
	Active bool
}

type ZoomControlsOptions struct {
	Class  string
	Label  string
	Active string
	Levels []ZoomLevel
}

type InsertOption struct {
	Key             string
	Label           string
	Summary         string
	Target          string
	ButtonLabel     string
	ButtonClass     string
	ButtonBaseClass string
}

type InsertShelfOptions struct {
	Class  string
	Kicker string
	Title  string
}

type SelectionCommand struct {
	Key   string
	Label string
}

type SelectionCommandOptions struct {
	Class          string
	Kicker         string
	SelectionLabel string
	StatusLabel    string
	FieldLabel     string
	Commands       []SelectionCommand
}

type CanvasStatusOptions struct {
	Class          string
	RouteLabel     string
	ViewportLabel  string
	SelectionLabel string
}

func RenderModebar(modes []Mode, options ModebarOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__modebar")
	label := firstNonEmpty(options.Label, "Editor mode")
	modes = normalizeModes(modes)
	nodes := make([]gosx.Node, 0, len(modes))
	for _, mode := range modes {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-mode-control", mode.Key),
			gosx.Attr("aria-pressed", boolAttr(mode.Active)),
		), gosx.Text(mode.Label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", label),
	), gosx.Fragment(nodes...))
}

func RenderMetricStrip(metrics []Metric, options MetricStripOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__metrics")
	label := firstNonEmpty(options.Label, "Workspace metrics")
	metrics = normalizeMetrics(metrics)
	nodes := make([]gosx.Node, 0, len(metrics))
	for _, metric := range metrics {
		nodes = append(nodes, gosx.El("span", gosx.Attrs(
			gosx.Attr("data-studio-metric", metric.Key),
		),
			gosx.El("strong", nil, gosx.Text(fmtAny(metric.Value))),
			gosx.Text(" "+metric.Label),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("aria-label", label),
	), gosx.Fragment(nodes...))
}

func RenderViewportSwitcher(viewports []Viewport, options ViewportSwitcherOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__viewports")
	label := firstNonEmpty(options.Label, "Preview viewport")
	viewports = normalizeViewports(viewports)
	nodes := make([]gosx.Node, 0, len(viewports))
	for _, viewport := range viewports {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-viewport", viewport.Key),
			gosx.Attr("data-studio-viewport-width", viewport.Width),
			gosx.Attr("aria-pressed", boolAttr(viewport.Active)),
		), gosx.Text(viewport.Label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", label),
	), gosx.Fragment(nodes...))
}

func RenderCanvasTools(options CanvasToolsOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__canvas-tools")
	label := firstNonEmpty(options.Label, "Canvas tools")
	nodes := []gosx.Node{
		canvasToolButton("data-studio-rail-toggle", "left", firstNonEmpty(options.LeftLabel, "Layers"), !options.LeftCollapsed),
		canvasToolButton("data-studio-rail-toggle", "right", firstNonEmpty(options.RightLabel, "Inspector"), !options.RightCollapsed),
	}
	if !options.HideActivity {
		nodes = append(nodes, canvasToolButton("data-studio-activity-toggle", "true", firstNonEmpty(options.ActivityLabel, "Activity"), !options.ActivityClosed))
	}
	if !options.HideFocus {
		nodes = append(nodes, canvasToolButton("data-studio-focus-toggle", "true", firstNonEmpty(options.FocusLabel, "Focus"), options.FocusActive))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", label),
	), gosx.Fragment(nodes...))
}

func RenderZoomControls(options ZoomControlsOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__zoom")
	label := firstNonEmpty(options.Label, "Canvas zoom")
	levels := normalizeZoomLevels(options.Levels, options.Active)
	nodes := make([]gosx.Node, 0, len(levels))
	for _, level := range levels {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-zoom", level.Key),
			gosx.Attr("aria-pressed", boolAttr(level.Active)),
		), gosx.Text(level.Label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", label),
	), gosx.Fragment(nodes...))
}

func RenderInsertShelf(options []InsertOption, renderOptions InsertShelfOptions) gosx.Node {
	className := firstNonEmpty(renderOptions.Class, "gosx-studio__insert-shelf")
	kicker := firstNonEmpty(renderOptions.Kicker, "Add block")
	title := firstNonEmpty(renderOptions.Title, "Insert")
	insertions := normalizeInsertOptions(options)
	buttons := make([]gosx.Node, 0, len(insertions))
	for _, option := range insertions {
		baseClass := firstNonEmpty(option.ButtonBaseClass, option.ButtonClass, "button")
		classAttr := firstNonEmpty(option.ButtonClass, baseClass+" button--secondary")
		target := firstNonEmpty(option.Target, option.Key)
		buttons = append(buttons, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", classAttr),
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-insert-block", target),
			gosx.Attr("data-editor-add-block", target),
			gosx.Attr("data-editor-button-base", baseClass),
		),
			gosx.El("span", nil, gosx.Text(option.Label)),
			gosx.El("small", nil, gosx.Text(firstNonEmpty(option.ButtonLabel, "Add"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-insert-shelf", "true"),
	),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(kicker)),
			gosx.El("strong", nil, gosx.Text(title)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-insert-list")), gosx.Fragment(buttons...)),
	)
}

func RenderSelectionCommandbar(options SelectionCommandOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__selection-commandbar")
	kicker := firstNonEmpty(options.Kicker, "Selection")
	selectionLabel := firstNonEmpty(options.SelectionLabel, "No selection")
	statusLabel := firstNonEmpty(options.StatusLabel, "Visible")
	fieldLabel := firstNonEmpty(options.FieldLabel, "Block")
	commands := normalizeSelectionCommands(options.Commands)
	commandNodes := make([]gosx.Node, 0, len(commands))
	for _, command := range commands {
		commandNodes = append(commandNodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-selection-action", command.Key),
		), gosx.Text(command.Label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-selection-commandbar", "true"),
	),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(kicker)),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(selectionLabel)),
		),
		gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "studio-selection-state"),
			gosx.Attr("data-studio-selection-status", "true"),
		), gosx.Text(statusLabel)),
		gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "studio-selection-state studio-selection-field"),
			gosx.Attr("data-studio-field-selection-label", "true"),
		), gosx.Text(fieldLabel)),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-selection-actions"),
			gosx.Attr("role", "toolbar"),
			gosx.Attr("aria-label", "Selected block actions"),
		), gosx.Fragment(commandNodes...)),
	)
}

func RenderCanvasStatus(options CanvasStatusOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__canvas-status")
	route := firstNonEmpty(options.RouteLabel, "Preview")
	viewport := firstNonEmpty(options.ViewportLabel, "Desktop")
	selection := firstNonEmpty(options.SelectionLabel, "No selection")
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("span", nil, gosx.Text("Page / "+route)),
		gosx.El("span", nil,
			gosx.Text("Viewport / "),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-viewport-label", "true")), gosx.Text(viewport)),
		),
		gosx.El("span", nil,
			gosx.Text("Selection / "),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(selection)),
		),
	)
}

func DefaultSelectionCommands() []SelectionCommand {
	return []SelectionCommand{
		{Key: "reveal", Label: "Layer"},
		{Key: "previous-field", Label: "Prev"},
		{Key: "next-field", Label: "Next"},
		{Key: "inline-text", Label: "Edit"},
		{Key: "content", Label: "Content"},
		{Key: "style", Label: "Style"},
		{Key: "toggle-visibility", Label: "Hide"},
	}
}

func DefaultZoomLevels(active string) []ZoomLevel {
	active = normalizeKey(firstNonEmpty(active, "fit"))
	levels := []ZoomLevel{
		{Key: "fit", Label: "Fit"},
		{Key: "75", Label: "75%"},
		{Key: "100", Label: "100%"},
		{Key: "125", Label: "125%"},
	}
	for index := range levels {
		levels[index].Active = levels[index].Key == active
	}
	return levels
}

func canvasToolButton(attr, value, label string, pressed bool) gosx.Node {
	attrs := []any{
		gosx.Attr("type", "button"),
		gosx.Attr("aria-pressed", boolAttr(pressed)),
	}
	if attr != "" {
		attrs = append(attrs, gosx.Attr(attr, value))
	}
	return gosx.El("button", gosx.Attrs(attrs...), gosx.Text(label))
}

func normalizeZoomLevels(levels []ZoomLevel, active string) []ZoomLevel {
	if len(levels) == 0 {
		return DefaultZoomLevels(active)
	}
	active = normalizeKey(active)
	out := make([]ZoomLevel, 0, len(levels))
	hasActive := false
	for _, level := range levels {
		level.Key = normalizeKey(level.Key)
		level.Label = strings.TrimSpace(level.Label)
		if level.Key == "" || level.Label == "" {
			continue
		}
		if active != "" {
			level.Active = level.Key == active
		}
		if level.Active {
			if hasActive {
				level.Active = false
			} else {
				hasActive = true
			}
		}
		out = append(out, level)
	}
	if len(out) > 0 && !hasActive {
		out[0].Active = true
	}
	return out
}

func normalizeInsertOptions(options []InsertOption) []InsertOption {
	out := make([]InsertOption, 0, len(options))
	for _, option := range options {
		option.Key = normalizeKey(option.Key)
		option.Label = strings.TrimSpace(option.Label)
		option.Summary = strings.TrimSpace(option.Summary)
		option.Target = normalizeKey(firstNonEmpty(option.Target, option.Key))
		option.ButtonLabel = strings.TrimSpace(option.ButtonLabel)
		option.ButtonClass = strings.TrimSpace(option.ButtonClass)
		option.ButtonBaseClass = strings.TrimSpace(option.ButtonBaseClass)
		if option.Key == "" || option.Label == "" || option.Target == "" {
			continue
		}
		out = append(out, option)
	}
	return out
}

func normalizeSelectionCommands(commands []SelectionCommand) []SelectionCommand {
	if len(commands) == 0 {
		commands = DefaultSelectionCommands()
	}
	out := make([]SelectionCommand, 0, len(commands))
	for _, command := range commands {
		command.Key = normalizeKey(command.Key)
		command.Label = strings.TrimSpace(command.Label)
		if command.Key == "" || command.Label == "" {
			continue
		}
		out = append(out, command)
	}
	return out
}
