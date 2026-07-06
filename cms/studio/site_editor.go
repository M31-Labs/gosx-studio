package studio

import "m31labs.dev/gosx"

type StudioToolbarOptions struct {
	Class        string
	ActionsClass string
	Kicker       string
	Title        string
	Summary      string
	Controls     []gosx.Node
	Actions      []gosx.Node
}

type PreviewFrameOptions struct {
	ShellClass   string
	ToolbarClass string
	MetaClass    string
	FrameClass   string
	StatusClass  string
	OpenClass    string
	Kicker       string
	Title        string
	URL          string
	IFrameTitle  string
	StatusLabel  string
	OpenLabel    string
	OpenNewTab   bool
	DynamicTitle bool
	DynamicRoute bool
	Controls     []gosx.Node
	Actions      []gosx.Node
}

func RenderStudioToolbar(options StudioToolbarOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio-toolbar")
	actionsClass := firstNonEmpty(options.ActionsClass, "gosx-studio-toolbar__actions")
	title := firstNonEmpty(options.Title, "Studio")

	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-studio-toolbar__title")),
			optionalText("p", "kicker", options.Kicker),
			gosx.El("strong", nil, gosx.Text(title)),
			optionalText("span", "", options.Summary),
		),
	}
	children = append(children, options.Controls...)
	if len(options.Actions) > 0 {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", actionsClass),
			gosx.Attr("data-gosx-studio-toolbar-actions", "true"),
		), gosx.Fragment(options.Actions...)))
	}

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-toolbar", "true"),
		gosx.Attr("data-gosx-studio-toolbar", "true"),
	), gosx.Fragment(children...))
}

func RenderPreviewFrame(options PreviewFrameOptions) gosx.Node {
	shellClass := firstNonEmpty(options.ShellClass, "gosx-studio-preview")
	toolbarClass := firstNonEmpty(options.ToolbarClass, "gosx-studio-preview__toolbar")
	metaClass := firstNonEmpty(options.MetaClass, "gosx-studio-preview__meta")
	frameClass := firstNonEmpty(options.FrameClass, "gosx-studio-preview__frame")
	statusClass := firstNonEmpty(options.StatusClass, "gosx-studio-preview__status")
	url := firstNonEmpty(options.URL, "/")
	title := firstNonEmpty(options.Title, "Preview")
	iframeTitle := firstNonEmpty(options.IFrameTitle, title)

	toolbarChildren := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", metaClass)),
			optionalText("span", "", options.Kicker),
			previewTitle(title, options.DynamicTitle),
			previewRoute(url, options.DynamicRoute),
		),
	}
	if options.StatusLabel != "" {
		toolbarChildren = append(toolbarChildren, gosx.El("output", gosx.Attrs(
			gosx.Attr("class", statusClass),
			gosx.Attr("data-studio-preview-status", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(options.StatusLabel)))
	}
	toolbarChildren = append(toolbarChildren, options.Controls...)
	toolbarChildren = append(toolbarChildren, options.Actions...)
	if options.OpenLabel != "" {
		toolbarChildren = append(toolbarChildren, previewOpenLink(options))
	}

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", shellClass),
		gosx.Attr("data-gosx-studio-preview", "true"),
		gosx.Attr("data-gosx-studio-preview-url", url),
		gosx.Attr("data-gosx-studio-preview-state", "ready"),
	),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", toolbarClass),
			gosx.Attr("data-studio-preview-toolbar", "true"),
		), gosx.Fragment(toolbarChildren...)),
		gosx.El("iframe", gosx.Attrs(
			gosx.Attr("class", frameClass),
			gosx.Attr("src", url),
			gosx.Attr("title", iframeTitle),
			gosx.Attr("data-studio-preview-frame", "true"),
			gosx.Attr("data-studio-preview-src", url),
		)),
	)
}

func optionalText(tag, className, text string) gosx.Node {
	if text == "" {
		return gosx.Fragment()
	}
	attrs := []any{}
	if className != "" {
		attrs = append(attrs, gosx.Attr("class", className))
	}
	return gosx.El(tag, gosx.Attrs(attrs...), gosx.Text(text))
}

func previewTitle(title string, dynamic bool) gosx.Node {
	attrs := []any{}
	if dynamic {
		attrs = append(attrs, gosx.Attr("data-studio-selected-flow-label", "true"))
	}
	return gosx.El("strong", gosx.Attrs(attrs...), gosx.Text(title))
}

func previewRoute(url string, dynamic bool) gosx.Node {
	attrs := []any{}
	if dynamic {
		attrs = append(attrs, gosx.Attr("data-studio-selected-flow-route", "true"))
	}
	return gosx.El("code", gosx.Attrs(attrs...), gosx.Text(url))
}

func previewOpenLink(options PreviewFrameOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", firstNonEmpty(options.OpenClass, "button")),
		gosx.Attr("href", firstNonEmpty(options.URL, "/")),
		gosx.Attr("data-studio-open-preview", "true"),
	}
	if options.OpenNewTab {
		attrs = append(attrs, gosx.Attr("target", "_blank"), gosx.Attr("rel", "noreferrer"))
	}
	return gosx.El("a", gosx.Attrs(attrs...), gosx.Text(options.OpenLabel))
}
