package studio

import "m31labs.dev/gosx"

type HistoryControlsOptions struct {
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

func RenderHistoryControls(options HistoryControlsOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio-history-controls")
	buttonClass := firstNonEmpty(options.ButtonClass, "button button--secondary")
	undoTitle := firstNonEmpty(options.UndoTitle, "Undo last change")
	redoTitle := firstNonEmpty(options.RedoTitle, "Redo last undone change")
	children := []gosx.Node{
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", buttonClass),
			gosx.Attr("type", "button"),
			gosx.Attr("data-gosx-studio-history-undo", "true"),
			gosx.Attr("aria-label", undoTitle),
			gosx.Attr("title", undoTitle),
			gosx.Attr("disabled", "disabled"),
		), gosx.Text(firstNonEmpty(options.UndoLabel, "Undo"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", buttonClass),
			gosx.Attr("type", "button"),
			gosx.Attr("data-gosx-studio-history-redo", "true"),
			gosx.Attr("aria-label", redoTitle),
			gosx.Attr("title", redoTitle),
			gosx.Attr("disabled", "disabled"),
		), gosx.Text(firstNonEmpty(options.RedoLabel, "Redo"))),
	}
	if options.IncludeStatus {
		children = append(children, gosx.El("output", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.StatusClass, className+"__status")),
			gosx.Attr("data-gosx-studio-history-status", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(firstNonEmpty(options.StatusLabel, "No local edits"))))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-history-controls", "true"),
	), gosx.Fragment(children...))
}
