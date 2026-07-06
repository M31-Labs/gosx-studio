package studio

import (
	"strconv"
	"strings"

	"m31labs.dev/gosx"
)

type WorkbenchOptions struct {
	Class                   string
	FormClass               string
	ToolbarClass            string
	ToolbarActionsClass     string
	StageClass              string
	LeftRailClass           string
	MainClass               string
	CanvasShellClass        string
	CanvasBarClass          string
	BoardClass              string
	FrameWrapClass          string
	RightRailClass          string
	ScriptsClass            string
	Action                  string
	Method                  string
	CSRFName                string
	CSRFToken               string
	Autosave                bool
	AutosaveURL             string
	AutosaveDelay           int
	FormAttrs               []FieldAttribute
	DisableClientActions    bool
	DisableCanvasStatus     bool
	DisableHistoryControls  bool
	DisableSelectionTools   bool
	ResizableRails          bool
	SaveButtonLabel         string
	ToolbarKicker           string
	ToolbarTitle            string
	ToolbarSummary          string
	Commands                []Command
	Insertions              []InsertOption
	SelectionCommands       []SelectionCommand
	Toolbar                 []gosx.Node
	ToolbarControls         []gosx.Node
	ToolbarActions          []gosx.Node
	Statuses                []gosx.Node
	Left                    []gosx.Node
	Main                    []gosx.Node
	CanvasBar               []gosx.Node
	Board                   []gosx.Node
	CanvasFooter            []gosx.Node
	Right                   []gosx.Node
	AfterForm               []gosx.Node
	Scripts                 []gosx.Node
	IncludeScripts          bool
	IncludeWorkbenchRuntime bool
	IncludeCommandRuntime   bool
	IncludeStateRuntime     bool
	IncludeCanvasRuntime    bool
	IncludeFlowRuntime      bool
}

func RenderWorkbench(shell Shell, options WorkbenchOptions) gosx.Node {
	formAttrs := workbenchFormAttrs(shell, options)
	formChildren := make([]gosx.Node, 0, 8)
	formChildren = append(formChildren, workbenchCSRFInput(options), gosx.Fragment(options.Statuses...))
	formChildren = append(formChildren, renderWorkbenchToolbar(shell, options))
	formChildren = append(formChildren, renderWorkbenchStage(shell, options))
	formChildren = append(formChildren, options.AfterForm...)

	children := []gosx.Node{
		gosx.El("form", gosx.Attrs(formAttrs...), gosx.Fragment(formChildren...)),
	}
	if scripts := renderWorkbenchScripts(options); len(scripts) > 0 {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.ScriptsClass, "gosx-studio-workbench__scripts")),
			gosx.Attr("data-gosx-studio-scripts", "true"),
		), gosx.Fragment(scripts...)))
	}

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.Class, "gosx-studio-workbench")),
		gosx.Attr("data-gosx-studio-workbench", "true"),
	), gosx.Fragment(children...))
}

func RenderStudioWorkbench(shell Shell, options WorkbenchOptions) gosx.Node {
	return RenderWorkbench(shell, options)
}

func workbenchFormAttrs(shell Shell, options WorkbenchOptions) []any {
	action := firstNonEmpty(options.Action, shell.SaveAction)
	method := firstNonEmpty(options.Method, "post")
	attrs := []any{
		gosx.Attr("class", firstNonEmpty(options.FormClass, "gosx-studio-workbench__form gosx-studio")),
		gosx.Attr("method", method),
		gosx.Attr("data-studio-workbench", "true"),
		gosx.Attr("data-editor-workbench", "true"),
		gosx.Attr("data-gosx-studio-state", "true"),
		gosx.Attr("data-studio-shell", normalizeKey(shell.Title)),
		gosx.Attr("data-studio-block-count", strconv.Itoa(shell.BlockCount)),
		gosx.Attr("data-studio-media-count", strconv.Itoa(shell.MediaCount)),
		gosx.Attr("data-studio-revision-count", strconv.Itoa(shell.RevisionCount)),
	}
	if action != "" {
		attrs = append(attrs, gosx.Attr("action", action))
	}
	if !options.DisableClientActions {
		attrs = append(attrs, gosx.Attr("data-gosx-studio-client", "true"))
	}
	if options.Autosave {
		delay := options.AutosaveDelay
		if delay <= 0 {
			delay = 1400
		}
		attrs = append(attrs,
			gosx.Attr("data-gosx-studio-autosave", "true"),
			gosx.Attr("data-gosx-studio-autosave-delay", strconv.Itoa(delay)),
			gosx.Attr("data-gosx-studio-autosave-url", firstNonEmpty(options.AutosaveURL, action)),
		)
	}
	return appendFieldAttributes(attrs, options.FormAttrs)
}

func workbenchCSRFInput(options WorkbenchOptions) gosx.Node {
	if strings.TrimSpace(options.CSRFToken) == "" {
		return gosx.Fragment()
	}
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("type", "hidden"),
		gosx.Attr("name", firstNonEmpty(options.CSRFName, "csrf_token")),
		gosx.Attr("value", options.CSRFToken),
	))
}

func renderWorkbenchToolbar(shell Shell, options WorkbenchOptions) gosx.Node {
	if len(options.Toolbar) > 0 {
		return gosx.Fragment(options.Toolbar...)
	}
	controls := []gosx.Node{
		RenderModebar(shell.Modes, ModebarOptions{Class: "studio-modebar", Label: "Editor mode"}),
		RenderMetricStrip(shell.Metrics, MetricStripOptions{Class: "studio-context-strip", Label: "Workspace details"}),
	}
	if len(options.Commands) > 0 {
		controls = append(controls, RenderCommandPalette(CommandPaletteOptions{
			Class:      "studio-command-palette",
			Launcher:   "Commands",
			Title:      firstNonEmpty(options.ToolbarTitle, shell.Title) + " commands",
			SearchHint: "Search modes, routes, blocks, styling, and publish actions",
			Commands:   options.Commands,
		}))
	}
	controls = append(controls, options.ToolbarControls...)

	actions := []gosx.Node{
		RenderSaveStatus(SaveStatusOptions{
			Class:           "editor-save-status",
			StateClass:      "editor-save-state",
			DetailClass:     "editor-save-detail",
			LastSavedClass:  "editor-save-time",
			DirtyCountClass: "editor-save-count",
		}),
	}
	if !options.DisableHistoryControls {
		actions = append(actions, RenderHistoryControls(HistoryControlsOptions{}))
	}
	actions = append(actions, renderWorkbenchActionLinks(shell.Actions)...)
	actions = append(actions, options.ToolbarActions...)
	if firstNonEmpty(options.Action, shell.SaveAction) != "" {
		actions = append(actions, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--primary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("data-gosx-studio-save-button", "true"),
		), gosx.Text(firstNonEmpty(options.SaveButtonLabel, "Save changes"))))
	}

	return RenderStudioToolbar(StudioToolbarOptions{
		Class:        firstNonEmpty(options.ToolbarClass, "editor-toolbar"),
		ActionsClass: firstNonEmpty(options.ToolbarActionsClass, "button-row"),
		Kicker:       firstNonEmpty(options.ToolbarKicker, shell.Canvas.RouteLabel),
		Title:        firstNonEmpty(options.ToolbarTitle, shell.Title),
		Summary:      options.ToolbarSummary,
		Controls:     controls,
		Actions:      actions,
	})
}

func renderWorkbenchActionLinks(actions []Action) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		if normalizeKey(action.Key) == "save" || strings.TrimSpace(action.Href) == "" {
			continue
		}
		className := "button button--secondary"
		if action.Primary {
			className = "button button--primary"
		}
		nodes = append(nodes, gosx.El("a", gosx.Attrs(
			gosx.Attr("class", className),
			gosx.Attr("href", action.Href),
			gosx.Attr("data-action-key", action.Key),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(action.Label)))
	}
	return nodes
}

func renderWorkbenchStage(shell Shell, options WorkbenchOptions) gosx.Node {
	children := []gosx.Node{
		renderWorkbenchRail("aside", firstNonEmpty(options.LeftRailClass, "studio-left-rail"), "left", options.Left, shell.Left),
	}
	if options.ResizableRails {
		children = append(children, renderWorkbenchRailResizer("left"))
	}
	children = append(children, renderWorkbenchMain(shell, options))
	if options.ResizableRails {
		children = append(children, renderWorkbenchRailResizer("right"))
	}
	children = append(children, renderWorkbenchRail("aside", firstNonEmpty(options.RightRailClass, "editor-sidebar"), "right", options.Right, shell.Right))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.StageClass, "editor-stage")),
		gosx.Attr("data-studio-layout", "true"),
	), gosx.Fragment(children...))
}

func renderWorkbenchRailResizer(side string) gosx.Node {
	min := "256"
	max := "448"
	value := "320"
	label := "Resize layers rail"
	if side == "right" {
		min = "320"
		max = "544"
		value = "416"
		label = "Resize inspector rail"
	}
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "studio-rail-resizer studio-rail-resizer--"+side),
		gosx.Attr("type", "button"),
		gosx.Attr("role", "separator"),
		gosx.Attr("aria-orientation", "vertical"),
		gosx.Attr("aria-label", label),
		gosx.Attr("aria-valuemin", min),
		gosx.Attr("aria-valuemax", max),
		gosx.Attr("aria-valuenow", value),
		gosx.Attr("data-studio-resizer", side),
		gosx.Attr("data-studio-rail-min", min),
		gosx.Attr("data-studio-rail-max", max),
		gosx.Attr("data-studio-rail-default", value),
	))
}

func renderWorkbenchRail(tag, className, side string, nodes []gosx.Node, panels []Panel) gosx.Node {
	children := nodes
	if len(children) == 0 {
		children = renderWorkbenchPanels(panels)
	}
	return gosx.El(tag, gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-sidebar", side),
	), gosx.Fragment(children...))
}

func renderWorkbenchPanels(panels []Panel) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(panels))
	for _, panel := range panels {
		children := []gosx.Node{gosx.El("h2", nil, gosx.Text(panel.Label))}
		if panel.Summary != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(panel.Summary)))
		}
		children = append(children, panel.Children...)
		nodes = append(nodes, gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "gosx-studio-workbench__panel"),
			gosx.Attr("data-panel-key", panel.Key),
		), gosx.Fragment(children...)))
	}
	return nodes
}

func renderWorkbenchMain(shell Shell, options WorkbenchOptions) gosx.Node {
	children := options.Main
	if len(children) == 0 {
		children = []gosx.Node{renderWorkbenchCanvasShell(shell, options)}
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.MainClass, "editor-canvas")),
		gosx.Attr("aria-label", firstNonEmpty(shell.Canvas.RouteLabel, shell.Title, "Studio canvas")),
		gosx.Attr("data-panel-key", normalizeKey(firstNonEmpty(shell.Canvas.RouteLabel, "canvas"))),
	), gosx.Fragment(children...))
}

func renderWorkbenchCanvasShell(shell Shell, options WorkbenchOptions) gosx.Node {
	children := []gosx.Node{
		renderWorkbenchCanvasBar(shell, options),
		renderWorkbenchBoard(shell, options),
	}
	if !options.DisableCanvasStatus {
		children = append(children, RenderCanvasStatus(CanvasStatusOptions{
			Class:          "studio-canvas-status",
			RouteLabel:     shell.Canvas.RouteLabel,
			ViewportLabel:  activeWorkbenchViewportLabel(shell.Viewports),
			SelectionLabel: shell.Canvas.SelectionLabel,
		}))
	}
	children = append(children, options.CanvasFooter...)
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.CanvasShellClass, "studio-canvas-shell")),
		gosx.Attr("data-studio-canvas", "true"),
		gosx.Attr("data-studio-canvas-zoom", firstNonEmpty(shell.Canvas.Zoom, "fit")),
	), gosx.Fragment(children...))
}

func renderWorkbenchCanvasBar(shell Shell, options WorkbenchOptions) gosx.Node {
	if len(options.CanvasBar) > 0 {
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.CanvasBarClass, "studio-canvas-bar"))), gosx.Fragment(options.CanvasBar...))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.CanvasBarClass, "studio-canvas-bar"))),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Canvas")),
			gosx.El("strong", nil, gosx.Text(firstNonEmpty(shell.Canvas.RouteLabel, shell.Title, "Preview"))),
		),
		gosx.El("nav", gosx.Attrs(
			gosx.Attr("class", "studio-breadcrumbs"),
			gosx.Attr("aria-label", "Canvas selection"),
		),
			gosx.El("span", nil, gosx.Text("Site")),
			gosx.El("span", nil, gosx.Text(firstNonEmpty(shell.Canvas.RouteLabel, "Preview"))),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(firstNonEmpty(shell.Canvas.SelectionLabel, "No selection"))),
		),
		RenderCanvasTools(CanvasToolsOptions{Class: "studio-canvas-tools", FocusActive: shell.Canvas.Focus}),
		RenderZoomControls(ZoomControlsOptions{Class: "studio-zoombar", Active: shell.Canvas.Zoom}),
	)
}

func renderWorkbenchBoard(shell Shell, options WorkbenchOptions) gosx.Node {
	children := options.Board
	if len(children) == 0 {
		if len(options.Insertions) > 0 {
			children = append(children, RenderInsertShelf(options.Insertions, InsertShelfOptions{Class: "studio-insert-shelf"}))
		}
		if !options.DisableSelectionTools {
			children = append(children, RenderSelectionCommandbar(SelectionCommandOptions{
				Class:          "studio-selection-commandbar",
				SelectionLabel: shell.Canvas.SelectionLabel,
				Commands:       options.SelectionCommands,
			}))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.FrameWrapClass, "studio-frame-wrap"))),
			RenderPreviewFrame(PreviewFrameOptions{
				ShellClass:   "editor-preview-shell",
				ToolbarClass: "storefront-frame-toolbar",
				MetaClass:    "studio-preview-toolbar__flow",
				FrameClass:   "storefront-frame editor-preview-frame",
				Title:        firstNonEmpty(shell.Canvas.RouteLabel, shell.Title, "Preview"),
				URL:          shell.PreviewURL,
				IFrameTitle:  firstNonEmpty(shell.Title, "Studio") + " preview",
				Controls: []gosx.Node{
					RenderViewportSwitcher(shell.Viewports, ViewportSwitcherOptions{Class: "studio-viewport-switcher", Label: "Preview viewport"}),
					gosx.El("output", gosx.Attrs(
						gosx.Attr("class", "studio-selection-readout"),
						gosx.Attr("data-studio-selection-label", "true"),
						gosx.Attr("aria-live", "polite"),
					), gosx.Text(firstNonEmpty(shell.Canvas.SelectionLabel, "No selection"))),
				},
			}),
		))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.BoardClass, "studio-canvas-board"))), gosx.Fragment(children...))
}

func renderWorkbenchScripts(options WorkbenchOptions) []gosx.Node {
	scripts := make([]gosx.Node, 0, 5+len(options.Scripts))
	if options.IncludeScripts || options.IncludeWorkbenchRuntime {
		scripts = append(scripts, RenderWorkbenchScript())
	}
	if options.IncludeScripts || options.IncludeCommandRuntime {
		scripts = append(scripts, RenderCommandPaletteScript())
	}
	if options.IncludeScripts || options.IncludeStateRuntime {
		scripts = append(scripts, RenderStudioStateScript())
	}
	if options.IncludeScripts || options.IncludeCanvasRuntime {
		scripts = append(scripts, RenderSiteCanvasScript())
	}
	if options.IncludeScripts || options.IncludeFlowRuntime {
		scripts = append(scripts, RenderFlowEditorScript())
	}
	scripts = append(scripts, options.Scripts...)
	return scripts
}

func activeWorkbenchViewportLabel(viewports []Viewport) string {
	for _, viewport := range normalizeViewports(viewports) {
		if viewport.Active {
			return viewport.Label
		}
	}
	return "Desktop"
}
