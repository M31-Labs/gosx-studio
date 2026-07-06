package studio

import "m31labs.dev/gosx"

type BackendDetailProps struct {
	Kicker  string
	Title   string
	Summary string
	Class   string
	Preview BackendDetailPreview
}

type BackendDetailPreview struct {
	Class    string
	TextOnly bool
	Lead     *gosx.Node
	Image    BackendDetailPreviewImage
	Primary  string
	Statuses []BackendDetailStatus
	Times    []BackendDetailTime
	Extra    *gosx.Node
}

type BackendDetailPreviewImage struct {
	Src string
	Alt string
}

type BackendDetailStatus struct {
	Label     string
	ClassName string
}

type BackendDetailTime struct {
	Label          string
	Machine        string
	ClassName      string
	ViewerTimeMode string
}

func RenderBackendDetail(props BackendDetailProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-detail-renderer", "gosx-studio"),
	),
		RenderBackendDetailContent(props),
	)
}

func RenderBackendDetailContent(props BackendDetailProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendDetailHeading(props),
		RenderBackendDetailPreview(props.Preview),
	)
}

func RenderBackendDetailHeading(props BackendDetailProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
		gosx.El("h1", nil, gosx.Text(props.Title)),
		gosx.El("p", nil, gosx.Text(props.Summary)),
	)
}

func RenderBackendDetailPreview(preview BackendDetailPreview) gosx.Node {
	className := preview.Class
	if className == "" {
		className = "edit-preview"
		if preview.TextOnly {
			className = "edit-preview edit-preview--text"
		}
	}
	children := make([]gosx.Node, 0, 2)
	if preview.Lead != nil && !preview.TextOnly {
		children = append(children, *preview.Lead)
	} else if preview.Image.Src != "" && !preview.TextOnly {
		children = append(children, gosx.El("img", gosx.Attrs(
			gosx.Attr("src", preview.Image.Src),
			gosx.Attr("alt", preview.Image.Alt),
		)))
	}
	children = append(children, gosx.El("div", nil, renderBackendDetailPreviewBody(preview)))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func renderBackendDetailPreviewBody(preview BackendDetailPreview) gosx.Node {
	children := make([]gosx.Node, 0, len(preview.Statuses)+len(preview.Times)+2)
	for _, status := range preview.Statuses {
		className := status.ClassName
		if className == "" {
			className = "status"
		}
		children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", className)), gosx.Text(status.Label)))
	}
	if preview.Primary != "" {
		children = append(children, gosx.El("strong", nil, gosx.Text(preview.Primary)))
	}
	for _, value := range preview.Times {
		children = append(children, renderBackendDetailTime(value))
	}
	if preview.Extra != nil {
		children = append(children, *preview.Extra)
	}
	return gosx.Fragment(children...)
}

func renderBackendDetailTime(value BackendDetailTime) gosx.Node {
	viewerTimeMode := value.ViewerTimeMode
	if viewerTimeMode == "" {
		viewerTimeMode = "datetime"
	}
	attrs := []any{
		gosx.Attr("datetime", value.Machine),
		gosx.Attr("data-viewer-time", viewerTimeMode),
	}
	if value.ClassName != "" {
		attrs = append([]any{gosx.Attr("class", value.ClassName)}, attrs...)
	}
	return gosx.El("time", gosx.Attrs(attrs...), gosx.Text(value.Label))
}
