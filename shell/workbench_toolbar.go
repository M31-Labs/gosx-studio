package shell

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type WorkbenchToolbarOptions struct {
	Class                  string
	ActionsClass           string
	TitleClass             string
	Kicker                 string
	Title                  string
	Summary                string
	PreviewLinkClass       string
	PreviewLabel           string
	SaveButtonClass        string
	SaveButtonLabel        string
	DisablePreviewAction   bool
	DisableSaveButton      bool
	DisableHistoryControls bool
	CommandPaletteNode     gosx.Node
	SaveStatusNode         gosx.Node
	HistoryControlsNode    gosx.Node
	Controls               []gosx.Node
	Actions                []gosx.Node
}

type WorkbenchSaveStatusOptions struct {
	Class           string
	StateClass      string
	DetailClass     string
	LastSavedClass  string
	DirtyCountClass string
	StateLabel      string
	DetailLabel     string
	LastSavedLabel  string
}

type WorkbenchHistoryControlsOptions struct {
	Class         string
	ButtonClass   string
	UndoLabel     string
	RedoLabel     string
	UndoTitle     string
	RedoTitle     string
	StatusClass   string
	StatusLabel   string
	IncludeStatus bool
}

type WorkbenchModebarOptions struct {
	Class string
	Label string
}

type WorkbenchMetricStripOptions struct {
	Class string
	Label string
}

type WorkbenchZoomControlsOptions struct {
	Class string
	Label string
}

type WorkbenchViewportControlsOptions struct {
	Class string
	Label string
}

type WorkbenchCanvasToolsOptions struct {
	Class string
	Label string
}

type WorkbenchCanvasBarOptions struct {
	Class            string
	TitleClass       string
	KickerClass      string
	Kicker           string
	RouteLabel       string
	BreadcrumbClass  string
	BreadcrumbLabel  string
	BreadcrumbRoot   string
	SelectionLabel   string
	SelectionDataKey string
	Controls         []gosx.Node
}

type WorkbenchCanvasStatusOptions struct {
	Class            string
	RouteLabel       string
	ViewportLabel    string
	SelectionLabel   string
	SelectionDataKey string
}

type WorkbenchCommandPaletteOptions struct {
	Class       string
	Launcher    string
	Title       string
	SearchHint  string
	EmptyTitle  string
	EmptyDetail string
}

func RenderWorkbenchToolbar(view map[string]any, options WorkbenchToolbarOptions) gosx.Node {
	kicker := core.FirstNonEmpty(options.Kicker, core.WorkbenchViewString(view, "toolbarKicker"), "Website")
	title := core.FirstNonEmpty(options.Title, core.WorkbenchViewString(view, "toolbarTitle"), "Editor")
	summary := core.FirstNonEmpty(options.Summary, core.WorkbenchViewString(view, "selectionLabel"), core.WorkbenchViewString(view, "routeLabel"))
	children := []gosx.Node{
		renderWorkbenchToolbarTitle(core.FirstNonEmpty(options.TitleClass, "gosx-studio-toolbar__title"), kicker,
			title,
			summary,
		),
	}
	children = append(children, options.Controls...)
	children = appendWorkbenchNode(children, options.CommandPaletteNode)

	actions := []gosx.Node{}
	actions = appendWorkbenchNode(actions, options.SaveStatusNode)
	if !options.DisableHistoryControls {
		if core.WorkbenchNodeEmpty(options.HistoryControlsNode) {
			actions = append(actions, RenderWorkbenchHistoryControls(WorkbenchHistoryControlsOptions{}))
		} else {
			actions = append(actions, options.HistoryControlsNode)
		}
	}
	actions = append(actions, options.Actions...)
	if !options.DisablePreviewAction {
		if href := core.WorkbenchViewString(view, "previewHref"); href != "" {
			actions = append(actions, gosx.El("a", gosx.Attrs(
				gosx.Attr("class", core.FirstNonEmpty(options.PreviewLinkClass, "button button--secondary")),
				gosx.Attr("href", href),
				gosx.Attr("data-gosx-link", "true"),
				gosx.Attr("data-gosx-studio-preview-link", "true"),
			), gosx.Text(core.FirstNonEmpty(options.PreviewLabel, core.WorkbenchViewString(view, "previewLabel"), "Preview"))))
		}
	}
	if !options.DisableSaveButton {
		saveLabel := core.FirstNonEmpty(options.SaveButtonLabel, core.WorkbenchViewString(view, "saveLabel"), "Save changes")
		if saveLabel != "" {
			actions = append(actions, gosx.El("button", gosx.Attrs(
				gosx.Attr("class", core.FirstNonEmpty(options.SaveButtonClass, "button button--primary")),
				gosx.Attr("type", "submit"),
				// This button submits the SHARED workbench form — every
				// panel's fields, including any hidden behind a
				// data-studio-mode-panel the operator isn't currently
				// viewing, are "listed" constraint-validation candidates of
				// that one form. Native HTML5 validation runs BEFORE the
				// browser dispatches the form's "submit" event; if any
				// hidden control is invalid, the browser can't focus it to
				// show a validation bubble and instead silently cancels the
				// submission with no submit event at all (Chrome logs "An
				// invalid form control with name='<x>' is not focusable" and
				// stops there) — the primary Save button would otherwise be
				// a no-op click with no visible error whenever an unrelated
				// hidden panel happens to carry an invalid value. Every
				// other action button that submits this same shared form
				// (Publish/Discard/Schedule in publish_panel.go, Restore in
				// revision_history_panel.go, Save in navigation_panel.go /
				// checkout_panel.go) already carries this exemption for the
				// same reason; the primary Save button must too.
				gosx.Attr("formnovalidate", "formnovalidate"),
				gosx.Attr("data-editor-save-button", "true"),
				gosx.Attr("data-gosx-studio-save-button", "true"),
			), gosx.Text(saveLabel)))
		}
	}
	if len(actions) > 0 {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.ActionsClass, "gosx-studio-toolbar__actions")),
			gosx.Attr("data-gosx-studio-toolbar-actions", "true"),
		), gosx.Fragment(actions...)))
	}

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "gosx-studio-toolbar")),
		gosx.Attr("data-studio-toolbar", "true"),
		gosx.Attr("data-gosx-studio-toolbar", "true"),
		gosx.Attr("data-gosx-studio-toolbar-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func RenderWorkbenchModebar(view map[string]any, options WorkbenchModebarOptions) gosx.Node {
	modes := core.WorkbenchViewMapList(view, "modes")
	nodes := make([]gosx.Node, 0, len(modes))
	for _, mode := range modes {
		key := core.WorkbenchViewString(mode, "key")
		label := core.WorkbenchViewString(mode, "label")
		if key == "" || label == "" {
			continue
		}
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-mode-control", key),
			gosx.Attr("aria-pressed", core.FirstNonEmpty(core.WorkbenchViewString(mode, "pressed"), core.BoolAttr(core.WorkbenchViewBool(mode, "active")))),
		), gosx.Text(label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "gosx-studio-modebar")),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", core.FirstNonEmpty(options.Label, core.WorkbenchViewString(view, "modebarLabel"), "Editor mode")),
	), gosx.Fragment(nodes...))
}

func RenderWorkbenchMetricStrip(view map[string]any, options WorkbenchMetricStripOptions) gosx.Node {
	metrics := core.WorkbenchViewMapList(view, "metrics")
	nodes := make([]gosx.Node, 0, len(metrics))
	for _, metric := range metrics {
		key := core.WorkbenchViewString(metric, "key")
		label := core.WorkbenchViewString(metric, "label")
		if key == "" || label == "" {
			continue
		}
		nodes = append(nodes, gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-metric", key)),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(metric, "value"))),
			gosx.Text(" "),
			gosx.Text(label),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "gosx-studio-context-strip")),
		gosx.Attr("aria-label", core.FirstNonEmpty(options.Label, core.WorkbenchViewString(view, "metricLabel"), "Workspace details")),
	), gosx.Fragment(nodes...))
}

func RenderWorkbenchZoomControls(view map[string]any, options WorkbenchZoomControlsOptions) gosx.Node {
	current := core.FirstNonEmpty(core.WorkbenchViewString(view, "zoom"), "fit")
	levels := core.WorkbenchViewMapList(view, "zoomLevels")
	for _, level := range levels {
		if core.WorkbenchViewBool(level, "pressed") {
			if key := core.WorkbenchViewString(level, "key"); key != "" {
				current = key
				break
			}
		}
	}
	nodes := make([]gosx.Node, 0, len(levels))
	for _, level := range levels {
		key := core.WorkbenchViewString(level, "key")
		label := core.WorkbenchViewString(level, "label")
		if key == "" || label == "" {
			continue
		}
		pressed := core.WorkbenchViewBool(level, "pressed")
		if key == current {
			pressed = true
		}
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-zoom", key),
			gosx.Attr("aria-pressed", core.BoolAttr(pressed)),
		), gosx.Text(label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-zoombar")),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", core.FirstNonEmpty(options.Label, "Canvas zoom")),
		gosx.Attr("data-studio-zoom-island", "true"),
		gosx.Attr("data-studio-zoom-current", current),
		gosx.Attr("data-gosx-studio-zoom-controls-renderer", "gosx-studio"),
	), gosx.Fragment(nodes...))
}

func RenderWorkbenchViewportControls(view map[string]any, options WorkbenchViewportControlsOptions) gosx.Node {
	current := core.FirstNonEmpty(core.WorkbenchViewString(view, "viewportKey"), "desktop")
	viewports := core.WorkbenchViewMapList(view, "viewports")
	nodes := make([]gosx.Node, 0, len(viewports))
	for _, viewport := range viewports {
		key := core.WorkbenchViewString(viewport, "key")
		label := core.WorkbenchViewString(viewport, "label")
		if key == "" || label == "" {
			continue
		}
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-viewport", key),
			gosx.Attr("data-studio-viewport-width", core.WorkbenchViewString(viewport, "width")),
			gosx.Attr("aria-pressed", core.BoolAttr(key == current)),
		), gosx.Text(label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-viewport-switcher")),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", core.FirstNonEmpty(options.Label, "Preview viewport")),
		gosx.Attr("data-studio-viewport-current", current),
		gosx.Attr("data-gosx-studio-viewport-controls-renderer", "gosx-studio"),
	), gosx.Fragment(nodes...))
}

func RenderWorkbenchCanvasTools(view map[string]any, options WorkbenchCanvasToolsOptions) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-canvas-tools")),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", core.FirstNonEmpty(options.Label, "Canvas tools")),
		gosx.Attr("data-gosx-studio-canvas-tools-renderer", "gosx-studio"),
	),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-rail-toggle", "left"),
			gosx.Attr("aria-pressed", "true"),
		), gosx.Text(core.FirstNonEmpty(core.WorkbenchViewString(view, "leftRailLabel"), "Layers"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-rail-toggle", "right"),
			gosx.Attr("aria-pressed", "true"),
		), gosx.Text(core.FirstNonEmpty(core.WorkbenchViewString(view, "rightRailLabel"), "Inspector"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-activity-toggle", "true"),
			gosx.Attr("aria-pressed", "true"),
		), gosx.Text(core.FirstNonEmpty(core.WorkbenchViewString(view, "activityLabel"), "Activity"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-focus-toggle", "true"),
			gosx.Attr("aria-pressed", "false"),
		), gosx.Text(core.FirstNonEmpty(core.WorkbenchViewString(view, "focusLabel"), "Focus"))),
	)
}

func RenderWorkbenchCanvasBar(view map[string]any, options WorkbenchCanvasBarOptions) gosx.Node {
	routeLabel := core.FirstNonEmpty(options.RouteLabel, core.WorkbenchViewString(view, "routeLabel"), "Preview")
	selectionLabel := core.FirstNonEmpty(options.SelectionLabel, core.WorkbenchViewString(view, "selectionLabel"), "No selection")
	titleAttrs := []any{}
	if strings.TrimSpace(options.TitleClass) != "" {
		titleAttrs = append(titleAttrs, gosx.Attr("class", strings.TrimSpace(options.TitleClass)))
	}
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(titleAttrs...),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", core.FirstNonEmpty(options.KickerClass, "kicker"))), gosx.Text(core.FirstNonEmpty(options.Kicker, "Canvas"))),
			gosx.El("strong", nil, gosx.Text(routeLabel)),
		),
		gosx.El("nav", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.BreadcrumbClass, "studio-breadcrumbs")),
			gosx.Attr("aria-label", core.FirstNonEmpty(options.BreadcrumbLabel, "Canvas selection")),
		),
			gosx.El("span", nil, gosx.Text(core.FirstNonEmpty(options.BreadcrumbRoot, "Site"))),
			gosx.El("span", nil, gosx.Text(routeLabel)),
			gosx.El("output", gosx.Attrs(gosx.Attr(core.FirstNonEmpty(options.SelectionDataKey, "data-studio-selection-label"), "true")), gosx.Text(selectionLabel)),
		),
	}
	children = append(children, options.Controls...)
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-canvas-bar")),
		gosx.Attr("data-gosx-studio-canvas-bar-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func RenderWorkbenchCanvasStatus(view map[string]any, options WorkbenchCanvasStatusOptions) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-canvas-status")),
		gosx.Attr("data-gosx-studio-canvas-status-renderer", "gosx-studio"),
	),
		gosx.El("span", nil, gosx.Text(core.FirstNonEmpty(options.RouteLabel, core.WorkbenchViewString(view, "routeLabel"), "Preview"))),
		gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-viewport-label", "true")), gosx.Text(core.FirstNonEmpty(options.ViewportLabel, core.WorkbenchViewString(view, "viewportLabel")))),
		gosx.El("output", gosx.Attrs(gosx.Attr(core.FirstNonEmpty(options.SelectionDataKey, "data-studio-selection-label"), "true")), gosx.Text(core.FirstNonEmpty(options.SelectionLabel, core.WorkbenchViewString(view, "selectionLabel"), "No selection"))),
	)
}

func RenderWorkbenchCommandPalette(view map[string]any, options WorkbenchCommandPaletteOptions) gosx.Node {
	commands := core.WorkbenchViewMapList(view, "commands")
	commandNodes := make([]gosx.Node, 0, len(commands))
	for _, command := range commands {
		key := core.WorkbenchViewString(command, "key")
		label := core.WorkbenchViewString(command, "label")
		if key == "" || label == "" {
			continue
		}
		children := []gosx.Node{
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__group")), gosx.Text(core.WorkbenchViewString(command, "group"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__label")), gosx.Text(label)),
		}
		if core.WorkbenchViewBool(command, "hasSummary") {
			children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__summary")), gosx.Text(core.WorkbenchViewString(command, "summary"))))
		}
		if core.WorkbenchViewBool(command, "hasShortcut") {
			children = append(children, gosx.El("kbd", nil, gosx.Text(core.WorkbenchViewString(command, "shortcut"))))
		}
		commandNodes = append(commandNodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(core.WorkbenchViewString(command, "class"), "studio-command-item")),
			gosx.Attr("type", "button"),
			gosx.Attr("role", "option"),
			gosx.Attr("data-studio-command", key),
			gosx.Attr("data-studio-command-kind", core.WorkbenchViewString(command, "kind")),
			gosx.Attr("data-studio-command-target", core.WorkbenchViewString(command, "target")),
			gosx.Attr("data-studio-command-search-text", core.WorkbenchViewString(command, "searchText")),
			gosx.Attr("data-studio-command-href", core.WorkbenchViewString(command, "href")),
			gosx.Attr("data-studio-command-shortcut", core.WorkbenchViewString(command, "shortcut")),
		), gosx.Fragment(children...)))
	}

	title := core.FirstNonEmpty(options.Title, core.WorkbenchViewString(view, "commandTitle"), "Editor tools")
	listID := "studio-command-list"
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "studio-command-palette")),
		gosx.Attr("data-studio-command-palette", "true"),
		gosx.Attr("data-studio-command-state", "closed"),
		gosx.Attr("data-gosx-studio-command-renderer", "gosx-studio"),
	),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "studio-command-launcher"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-command-open", "true"),
			gosx.Attr("aria-haspopup", "dialog"),
			gosx.Attr("aria-expanded", "false"),
		),
			gosx.El("span", nil, gosx.Text(core.FirstNonEmpty(options.Launcher, core.WorkbenchViewString(view, "commandLauncher"), "Find"))),
			gosx.El("kbd", nil, gosx.Text("Ctrl K")),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-command-overlay"),
			gosx.Attr("data-studio-command-overlay", "true"),
			gosx.Attr("hidden", "hidden"),
		),
			gosx.El("section", gosx.Attrs(
				gosx.Attr("class", "studio-command-dialog"),
				gosx.Attr("role", "dialog"),
				gosx.Attr("aria-modal", "true"),
				gosx.Attr("aria-label", title),
			),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-command-head")),
					gosx.El("div", nil,
						gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Quick actions")),
						gosx.El("h2", nil, gosx.Text(title)),
					),
					gosx.El("button", gosx.Attrs(
						gosx.Attr("type", "button"),
						gosx.Attr("data-studio-command-close", "true"),
						gosx.Attr("aria-label", "Close command palette"),
					), gosx.Text("Close")),
				),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-command-search")),
					gosx.El("span", nil, gosx.Text("Search")),
					gosx.El("input", gosx.Attrs(
						gosx.Attr("type", "search"),
						gosx.Attr("role", "combobox"),
						gosx.Attr("aria-autocomplete", "list"),
						gosx.Attr("aria-expanded", "false"),
						gosx.Attr("aria-controls", listID),
						gosx.Attr("data-studio-command-search", "true"),
						gosx.Attr("placeholder", core.FirstNonEmpty(options.SearchHint, core.WorkbenchViewString(view, "commandSearchHint"), "Find editing areas, sections, previews, and advanced tools")),
						gosx.Attr("autocomplete", "off"),
					)),
				),
				gosx.El("div", gosx.Attrs(
					gosx.Attr("id", listID),
					gosx.Attr("class", "studio-command-list"),
					gosx.Attr("role", "listbox"),
					gosx.Attr("data-studio-command-list", "true"),
				), gosx.Fragment(commandNodes...)),
				gosx.El("p", gosx.Attrs(
					gosx.Attr("class", "studio-command-empty"),
					gosx.Attr("data-studio-command-empty", "true"),
					gosx.Attr("hidden", "hidden"),
				),
					gosx.El("strong", nil, gosx.Text(core.FirstNonEmpty(options.EmptyTitle, core.WorkbenchViewString(view, "commandEmptyTitle"), "No commands"))),
					gosx.El("span", nil, gosx.Text(core.FirstNonEmpty(options.EmptyDetail, core.WorkbenchViewString(view, "commandEmptyDetail"), "Try a different search."))),
				),
			),
		),
	)
}

func RenderWorkbenchSaveStatus(options WorkbenchSaveStatusOptions) gosx.Node {
	className := core.FirstNonEmpty(options.Class, "gosx-studio-save-status")
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-save-status", "true"),
	),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.StateClass, className+"__state")),
			gosx.Attr("data-gosx-studio-save-state", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(core.FirstNonEmpty(options.StateLabel, "Saved"))),
		gosx.El("span", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.DetailClass, className+"__detail")),
			gosx.Attr("data-gosx-studio-save-detail", "true"),
		), gosx.Text(core.FirstNonEmpty(options.DetailLabel, "Ready"))),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.DirtyCountClass, className+"__dirty-count")),
			gosx.Attr("data-gosx-studio-dirty-count", "true"),
			gosx.Attr("hidden", "hidden"),
		), gosx.Text("0 changes")),
		gosx.El("time", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.LastSavedClass, className+"__last-saved")),
			gosx.Attr("data-gosx-studio-last-saved", "true"),
			gosx.Attr("data-gosx-studio-last-saved-empty", core.FirstNonEmpty(options.LastSavedLabel, "Not saved this session")),
		), gosx.Text(core.FirstNonEmpty(options.LastSavedLabel, "Not saved this session"))),
	)
}

func RenderWorkbenchHistoryControls(options WorkbenchHistoryControlsOptions) gosx.Node {
	className := core.FirstNonEmpty(options.Class, "gosx-studio-history-controls")
	buttonClass := core.FirstNonEmpty(options.ButtonClass, "button button--secondary")
	undoTitle := core.FirstNonEmpty(options.UndoTitle, "Undo last change")
	redoTitle := core.FirstNonEmpty(options.RedoTitle, "Redo last undone change")
	children := []gosx.Node{
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", buttonClass),
			gosx.Attr("type", "button"),
			gosx.Attr("data-gosx-studio-history-undo", "true"),
			gosx.Attr("aria-label", undoTitle),
			gosx.Attr("title", undoTitle),
			gosx.Attr("disabled", "disabled"),
		), gosx.Text(core.FirstNonEmpty(options.UndoLabel, "Undo"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", buttonClass),
			gosx.Attr("type", "button"),
			gosx.Attr("data-gosx-studio-history-redo", "true"),
			gosx.Attr("aria-label", redoTitle),
			gosx.Attr("title", redoTitle),
			gosx.Attr("disabled", "disabled"),
		), gosx.Text(core.FirstNonEmpty(options.RedoLabel, "Redo"))),
	}
	if options.IncludeStatus {
		children = append(children, gosx.El("output", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.StatusClass, className+"__status")),
			gosx.Attr("data-gosx-studio-history-status", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(core.FirstNonEmpty(options.StatusLabel, "No local edits"))))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-history-controls", "true"),
	), gosx.Fragment(children...))
}

func renderWorkbenchToolbarTitle(className, kicker, title, summary string) gosx.Node {
	children := []gosx.Node{}
	if kicker != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(kicker)))
	}
	children = append(children, gosx.El("strong", nil, gosx.Text(title)))
	if summary != "" {
		// This subtitle is fed by selectionLabel (falling back to routeLabel) at
		// server-render time (see RenderWorkbenchToolbar above), so it MUST carry
		// the same data-studio-selection-label hook the OTHER selection readouts
		// use (RenderWorkbenchCanvasBar, RenderWorkbenchCanvasStatus,
		// shell/workbench_page_canvas.go). Without it, workbench_runtime.js's
		// setReadout("[data-studio-selection-label]", ...) can never find this
		// node, so it stays frozen at whatever string was baked in on initial
		// load (almost always "No selection") no matter how many times the user
		// clicks the canvas — this is the toolbar-title-never-updates defect
		// (HANDOFF-19): every OTHER selection surface (the page-canvas readout,
		// the right-rail inspector) updates correctly on click; only this most
		// prominent, top-of-page label was silently exempt because it is a bare
		// <span> with no data attribute at all.
		children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(summary)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func appendWorkbenchNode(nodes []gosx.Node, node gosx.Node) []gosx.Node {
	if core.WorkbenchNodeEmpty(node) {
		return nodes
	}
	return append(nodes, node)
}

// workbenchNodeEmpty/workbenchViewString/workbenchViewMapList/
// workbenchMapString/workbenchMapBool moved to core as the exported
// WorkbenchNodeEmpty/WorkbenchViewString/WorkbenchViewMapList/WorkbenchViewBool
// helpers in Slice 8 (see core/workbench_view.go). Callers in this package now
// reference core.Workbench* directly.
