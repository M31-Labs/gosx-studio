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

	actions := []gosx.Node{}
	actions = appendWorkbenchNode(actions, options.CommandPaletteNode)
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
