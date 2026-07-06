package studio

import "m31labs.dev/gosx"

type SaveStatusOptions struct {
	Class           string
	StateClass      string
	DetailClass     string
	LastSavedClass  string
	DirtyCountClass string
	StateLabel      string
	DetailLabel     string
	LastSavedLabel  string
}

func RenderSaveStatus(options SaveStatusOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio-save-status")
	stateClass := firstNonEmpty(options.StateClass, "gosx-studio-save-status__state")
	detailClass := firstNonEmpty(options.DetailClass, "gosx-studio-save-status__detail")
	lastSavedClass := firstNonEmpty(options.LastSavedClass, "gosx-studio-save-status__last-saved")
	dirtyCountClass := firstNonEmpty(options.DirtyCountClass, "gosx-studio-save-status__dirty-count")
	stateLabel := firstNonEmpty(options.StateLabel, "Saved")
	detailLabel := firstNonEmpty(options.DetailLabel, "Ready")
	lastSavedLabel := firstNonEmpty(options.LastSavedLabel, "Not saved this session")

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-save-status", "true"),
	),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", stateClass),
			gosx.Attr("data-gosx-studio-save-state", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(stateLabel)),
		gosx.El("span", gosx.Attrs(
			gosx.Attr("class", detailClass),
			gosx.Attr("data-gosx-studio-save-detail", "true"),
		), gosx.Text(detailLabel)),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", dirtyCountClass),
			gosx.Attr("data-gosx-studio-dirty-count", "true"),
			gosx.Attr("hidden", "hidden"),
		), gosx.Text("0 changes")),
		gosx.El("time", gosx.Attrs(
			gosx.Attr("class", lastSavedClass),
			gosx.Attr("data-gosx-studio-last-saved", "true"),
			gosx.Attr("data-gosx-studio-last-saved-empty", lastSavedLabel),
		), gosx.Text(lastSavedLabel)),
	)
}
