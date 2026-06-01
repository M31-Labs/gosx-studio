package studio

import (
	"strings"

	"m31labs.dev/gosx"
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

type WorkbenchCommandPaletteOptions struct {
	Class       string
	Launcher    string
	Title       string
	SearchHint  string
	EmptyTitle  string
	EmptyDetail string
}

func RenderWorkbenchToolbar(view map[string]any, options WorkbenchToolbarOptions) gosx.Node {
	kicker := FirstNonEmpty(options.Kicker, workbenchViewString(view, "toolbarKicker"), "Website")
	title := FirstNonEmpty(options.Title, workbenchViewString(view, "toolbarTitle"), "Editor")
	summary := FirstNonEmpty(options.Summary, workbenchViewString(view, "selectionLabel"), workbenchViewString(view, "routeLabel"))
	children := []gosx.Node{
		renderWorkbenchToolbarTitle(
			FirstNonEmpty(options.TitleClass, "gosx-studio-toolbar__title"),
			kicker,
			title,
			summary,
		),
	}
	children = append(children, options.Controls...)
	children = appendWorkbenchNode(children, options.CommandPaletteNode)

	actions := []gosx.Node{}
	actions = appendWorkbenchNode(actions, options.SaveStatusNode)
	if !options.DisableHistoryControls {
		if workbenchNodeEmpty(options.HistoryControlsNode) {
			actions = append(actions, RenderWorkbenchHistoryControls(WorkbenchHistoryControlsOptions{}))
		} else {
			actions = append(actions, options.HistoryControlsNode)
		}
	}
	actions = append(actions, options.Actions...)
	if !options.DisablePreviewAction {
		if href := workbenchViewString(view, "previewHref"); href != "" {
			actions = append(actions, gosx.El("a", gosx.Attrs(
				gosx.Attr("class", FirstNonEmpty(options.PreviewLinkClass, "button button--secondary")),
				gosx.Attr("href", href),
				gosx.Attr("data-gosx-link", "true"),
				gosx.Attr("data-gosx-studio-preview-link", "true"),
			), gosx.Text(FirstNonEmpty(options.PreviewLabel, workbenchViewString(view, "previewLabel"), "Preview"))))
		}
	}
	if !options.DisableSaveButton {
		saveLabel := FirstNonEmpty(options.SaveButtonLabel, workbenchViewString(view, "saveLabel"), "Save changes")
		if saveLabel != "" {
			actions = append(actions, gosx.El("button", gosx.Attrs(
				gosx.Attr("class", FirstNonEmpty(options.SaveButtonClass, "button button--primary")),
				gosx.Attr("type", "submit"),
				gosx.Attr("data-editor-save-button", "true"),
				gosx.Attr("data-gosx-studio-save-button", "true"),
			), gosx.Text(saveLabel)))
		}
	}
	if len(actions) > 0 {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", FirstNonEmpty(options.ActionsClass, "gosx-studio-toolbar__actions")),
			gosx.Attr("data-gosx-studio-toolbar-actions", "true"),
		), gosx.Fragment(actions...)))
	}

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.Class, "gosx-studio-toolbar")),
		gosx.Attr("data-studio-toolbar", "true"),
		gosx.Attr("data-gosx-studio-toolbar", "true"),
		gosx.Attr("data-gosx-studio-toolbar-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func RenderWorkbenchModebar(view map[string]any, options WorkbenchModebarOptions) gosx.Node {
	modes := workbenchViewMapList(view, "modes")
	nodes := make([]gosx.Node, 0, len(modes))
	for _, mode := range modes {
		key := workbenchMapString(mode, "key")
		label := workbenchMapString(mode, "label")
		if key == "" || label == "" {
			continue
		}
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-mode-control", key),
			gosx.Attr("aria-pressed", FirstNonEmpty(workbenchMapString(mode, "pressed"), BoolAttr(workbenchMapBool(mode, "active")))),
		), gosx.Text(label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.Class, "gosx-studio-modebar")),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", FirstNonEmpty(options.Label, workbenchViewString(view, "modebarLabel"), "Editor mode")),
	), gosx.Fragment(nodes...))
}

func RenderWorkbenchMetricStrip(view map[string]any, options WorkbenchMetricStripOptions) gosx.Node {
	metrics := workbenchViewMapList(view, "metrics")
	nodes := make([]gosx.Node, 0, len(metrics))
	for _, metric := range metrics {
		key := workbenchMapString(metric, "key")
		label := workbenchMapString(metric, "label")
		if key == "" || label == "" {
			continue
		}
		nodes = append(nodes, gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-metric", key)),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(metric, "value"))),
			gosx.Text(" "),
			gosx.Text(label),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.Class, "gosx-studio-context-strip")),
		gosx.Attr("aria-label", FirstNonEmpty(options.Label, workbenchViewString(view, "metricLabel"), "Workspace details")),
	), gosx.Fragment(nodes...))
}

func RenderWorkbenchCommandPalette(view map[string]any, options WorkbenchCommandPaletteOptions) gosx.Node {
	commands := workbenchViewMapList(view, "commands")
	commandNodes := make([]gosx.Node, 0, len(commands))
	for _, command := range commands {
		key := workbenchMapString(command, "key")
		label := workbenchMapString(command, "label")
		if key == "" || label == "" {
			continue
		}
		children := []gosx.Node{
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__group")), gosx.Text(workbenchMapString(command, "group"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__label")), gosx.Text(label)),
		}
		if workbenchMapBool(command, "hasSummary") {
			children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-command-item__summary")), gosx.Text(workbenchMapString(command, "summary"))))
		}
		if workbenchMapBool(command, "hasShortcut") {
			children = append(children, gosx.El("kbd", nil, gosx.Text(workbenchMapString(command, "shortcut"))))
		}
		commandNodes = append(commandNodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", FirstNonEmpty(workbenchMapString(command, "class"), "studio-command-item")),
			gosx.Attr("type", "button"),
			gosx.Attr("role", "option"),
			gosx.Attr("data-studio-command", key),
			gosx.Attr("data-studio-command-kind", workbenchMapString(command, "kind")),
			gosx.Attr("data-studio-command-target", workbenchMapString(command, "target")),
			gosx.Attr("data-studio-command-search-text", workbenchMapString(command, "searchText")),
			gosx.Attr("data-studio-command-href", workbenchMapString(command, "href")),
			gosx.Attr("data-studio-command-shortcut", workbenchMapString(command, "shortcut")),
		), gosx.Fragment(children...)))
	}

	title := FirstNonEmpty(options.Title, workbenchViewString(view, "commandTitle"), "Editor tools")
	listID := "studio-command-list"
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.Class, "studio-command-palette")),
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
			gosx.El("span", nil, gosx.Text(FirstNonEmpty(options.Launcher, workbenchViewString(view, "commandLauncher"), "Find"))),
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
						gosx.Attr("placeholder", FirstNonEmpty(options.SearchHint, workbenchViewString(view, "commandSearchHint"), "Find editing areas, sections, previews, and advanced tools")),
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
					gosx.El("strong", nil, gosx.Text(FirstNonEmpty(options.EmptyTitle, workbenchViewString(view, "commandEmptyTitle"), "No commands"))),
					gosx.El("span", nil, gosx.Text(FirstNonEmpty(options.EmptyDetail, workbenchViewString(view, "commandEmptyDetail"), "Try a different search."))),
				),
			),
		),
	)
}

func RenderWorkbenchSaveStatus(options WorkbenchSaveStatusOptions) gosx.Node {
	className := FirstNonEmpty(options.Class, "gosx-studio-save-status")
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-save-status", "true"),
	),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", FirstNonEmpty(options.StateClass, className+"__state")),
			gosx.Attr("data-gosx-studio-save-state", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(FirstNonEmpty(options.StateLabel, "Saved"))),
		gosx.El("span", gosx.Attrs(
			gosx.Attr("class", FirstNonEmpty(options.DetailClass, className+"__detail")),
			gosx.Attr("data-gosx-studio-save-detail", "true"),
		), gosx.Text(FirstNonEmpty(options.DetailLabel, "Ready"))),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", FirstNonEmpty(options.DirtyCountClass, className+"__dirty-count")),
			gosx.Attr("data-gosx-studio-dirty-count", "true"),
			gosx.Attr("hidden", "hidden"),
		), gosx.Text("0 changes")),
		gosx.El("time", gosx.Attrs(
			gosx.Attr("class", FirstNonEmpty(options.LastSavedClass, className+"__last-saved")),
			gosx.Attr("data-gosx-studio-last-saved", "true"),
			gosx.Attr("data-gosx-studio-last-saved-empty", FirstNonEmpty(options.LastSavedLabel, "Not saved this session")),
		), gosx.Text(FirstNonEmpty(options.LastSavedLabel, "Not saved this session"))),
	)
}

func RenderWorkbenchHistoryControls(options WorkbenchHistoryControlsOptions) gosx.Node {
	className := FirstNonEmpty(options.Class, "gosx-studio-history-controls")
	buttonClass := FirstNonEmpty(options.ButtonClass, "button button--secondary")
	undoTitle := FirstNonEmpty(options.UndoTitle, "Undo last change")
	redoTitle := FirstNonEmpty(options.RedoTitle, "Redo last undone change")
	children := []gosx.Node{
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", buttonClass),
			gosx.Attr("type", "button"),
			gosx.Attr("data-gosx-studio-history-undo", "true"),
			gosx.Attr("aria-label", undoTitle),
			gosx.Attr("title", undoTitle),
			gosx.Attr("disabled", "disabled"),
		), gosx.Text(FirstNonEmpty(options.UndoLabel, "Undo"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", buttonClass),
			gosx.Attr("type", "button"),
			gosx.Attr("data-gosx-studio-history-redo", "true"),
			gosx.Attr("aria-label", redoTitle),
			gosx.Attr("title", redoTitle),
			gosx.Attr("disabled", "disabled"),
		), gosx.Text(FirstNonEmpty(options.RedoLabel, "Redo"))),
	}
	if options.IncludeStatus {
		children = append(children, gosx.El("output", gosx.Attrs(
			gosx.Attr("class", FirstNonEmpty(options.StatusClass, className+"__status")),
			gosx.Attr("data-gosx-studio-history-status", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(FirstNonEmpty(options.StatusLabel, "No local edits"))))
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
		children = append(children, gosx.El("span", nil, gosx.Text(summary)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func appendWorkbenchNode(nodes []gosx.Node, node gosx.Node) []gosx.Node {
	if workbenchNodeEmpty(node) {
		return nodes
	}
	return append(nodes, node)
}

func workbenchNodeEmpty(node gosx.Node) bool {
	return gosx.RenderHTML(node) == "<></>"
}

func workbenchViewString(view map[string]any, key string) string {
	if view == nil {
		return ""
	}
	value, ok := view[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(FmtAny(typed))
	}
}

func workbenchViewMapList(view map[string]any, key string) []map[string]any {
	if view == nil {
		return nil
	}
	switch typed := view[key].(type) {
	case []map[string]any:
		return typed
	case []map[string]string:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			next := make(map[string]any, len(item))
			for key, value := range item {
				next[key] = value
			}
			out = append(out, next)
		}
		return out
	default:
		return nil
	}
}

func workbenchMapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(FmtAny(typed))
	}
}

func workbenchMapBool(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	switch typed := values[key].(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}
