package studio

import "m31labs.dev/gosx"

const backendMediaLibraryUploadAccept = "image/png,image/jpeg,image/webp,image/gif,image/x-icon,.ico,.woff,.woff2,.ttf,.otf"

type BackendMediaLibraryProps struct {
	Kicker     string
	Title      string
	Summary    string
	Empty      string
	Class      string
	PanelTitle string
	Assets     []BackendMediaAsset
}

type BackendMediaAsset struct {
	Href        string
	URL         string
	Alt         string
	Filename    string
	ContentType string
	Size        string
	StatusLabel string
	IsImage     bool
	GOSXLink    bool
}

type BackendMediaLibraryPageProps struct {
	Class        string
	Library      BackendMediaLibraryProps
	UploadAction string
	CreateAction string
	CSRFToken    string
	Uploaded     bool
	HasError     bool
	Error        string
	CreateStatus BackendMediaLibraryActionStatus
	CreateValues BackendMediaLibraryRemoteAssetValues
}

type BackendMediaLibraryActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendMediaLibraryRemoteAssetValues struct {
	URL      string
	Alt      string
	Filename string
}

func RenderBackendMediaLibrary(props BackendMediaLibraryProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-media-library-renderer", "gosx-studio"),
	),
		RenderBackendMediaLibraryContent(props),
	)
}

func RenderBackendMediaLibraryPage(props BackendMediaLibraryPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-media-library-renderer", "gosx-studio"),
	),
		RenderBackendMediaLibraryPageContent(props),
	)
}

func RenderBackendMediaLibraryPageContent(props BackendMediaLibraryPageProps) gosx.Node {
	children := []gosx.Node{
		RenderBackendMediaLibraryHeading(props.Library),
	}
	if props.Uploaded {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--ok")), gosx.Text("Upload saved to the media library.")))
	}
	if props.HasError {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(props.Error)))
	}
	children = append(children,
		RenderBackendMediaLibraryAssets(props.Library),
		RenderBackendMediaLibraryActions(props),
	)
	return gosx.Fragment(children...)
}

func RenderBackendMediaLibraryActions(props BackendMediaLibraryPageProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-grid")),
		RenderBackendMediaLibraryUploadForm(props),
		RenderBackendMediaLibraryRemoteAssetForm(props),
	)
}

func RenderBackendMediaLibraryUploadForm(props BackendMediaLibraryPageProps) gosx.Node {
	action := props.UploadAction
	if action == "" {
		action = "/admin/media/upload"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Upload file")),
		),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form admin-form--single"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", action),
			gosx.Attr("enctype", "multipart/form-data"),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", "file")), gosx.Text("File")),
				gosx.El("input", gosx.Attrs(gosx.Attr("id", "file"), gosx.Attr("name", "file"), gosx.Attr("type", "file"), gosx.Attr("accept", backendMediaLibraryUploadAccept))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", "uploadAlt")), gosx.Text("Alt text")),
				gosx.El("input", gosx.Attrs(gosx.Attr("id", "uploadAlt"), gosx.Attr("name", "alt"))),
			),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Upload file")),
		),
	)
}

func RenderBackendMediaLibraryRemoteAssetForm(props BackendMediaLibraryPageProps) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Add remote asset")),
		),
		renderBackendMediaLibraryCreateStatus(props.CreateStatus),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form admin-form--single"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", props.CreateAction),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			renderBackendMediaLibraryInput("url", "URL", props.CreateValues.URL, renderBackendMediaLibraryFieldError(props.CreateStatus, "url")),
			renderBackendMediaLibraryInput("alt", "Alt text", props.CreateValues.Alt, nil),
			renderBackendMediaLibraryInput("filename", "Filename", props.CreateValues.Filename, nil),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Add asset")),
		),
	)
}

func RenderBackendMediaLibraryContent(props BackendMediaLibraryProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendMediaLibraryHeading(props),
		RenderBackendMediaLibraryAssets(props),
	)
}

func RenderBackendMediaLibraryHeading(props BackendMediaLibraryProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
		gosx.El("h1", nil, gosx.Text(props.Title)),
		gosx.El("p", nil, gosx.Text(props.Summary)),
	)
}

func RenderBackendMediaLibraryAssets(props BackendMediaLibraryProps) gosx.Node {
	panelTitle := props.PanelTitle
	if panelTitle == "" {
		panelTitle = "Assets"
	}
	empty := props.Empty
	if empty == "" {
		empty = "No media assets yet."
	}
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text(panelTitle)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "media-grid")),
			gosx.Fragment(renderBackendMediaAssetCards(props.Assets)...),
		),
	}
	if len(props.Assets) == 0 {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(empty)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
}

func renderBackendMediaAssetCards(assets []BackendMediaAsset) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(assets))
	for _, asset := range assets {
		attrs := []any{
			gosx.Attr("class", "media-card"),
			gosx.Attr("href", asset.Href),
		}
		if asset.GOSXLink {
			attrs = append(attrs, gosx.Attr("data-gosx-link", "true"))
		}
		nodes = append(nodes, gosx.El("a", gosx.Attrs(attrs...),
			renderBackendMediaAssetPreview(asset),
			gosx.El("strong", nil, gosx.Text(asset.Filename)),
			gosx.El("span", nil, gosx.Text(asset.Size)),
			gosx.El("span", nil, gosx.Text(asset.StatusLabel)),
		))
	}
	return nodes
}

func renderBackendMediaAssetPreview(asset BackendMediaAsset) gosx.Node {
	if asset.IsImage {
		return gosx.El("img", gosx.Attrs(
			gosx.Attr("src", asset.URL),
			gosx.Attr("alt", asset.Alt),
		))
	}
	return gosx.El("span", gosx.Attrs(gosx.Attr("class", "media-card__file")), gosx.Text(asset.ContentType))
}

func renderBackendMediaLibraryCreateStatus(status BackendMediaLibraryActionStatus) gosx.Node {
	if !status.Submitted {
		return gosx.Fragment()
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message))
}

func renderBackendMediaLibraryInput(id, label, value string, errorNode *gosx.Node) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("value", value))),
	}
	if errorNode != nil {
		children = append(children, *errorNode)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")), gosx.Fragment(children...))
}

func renderBackendMediaLibraryFieldError(status BackendMediaLibraryActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}
