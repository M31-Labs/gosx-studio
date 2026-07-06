package studio

import "m31labs.dev/gosx"

const backendMediaReplaceAccept = "image/png,image/jpeg,image/webp,image/gif,image/x-icon,.ico,.woff,.woff2,.ttf,.otf"

type BackendMediaDetailPageProps struct {
	Class         string
	Heading       gosx.Node
	Preview       gosx.Node
	SaveAction    string
	ArchiveAction string
	RestoreAction string
	ReplaceAction string
	CSRFToken     string
	SaveStatus    BackendMediaDetailActionStatus
	ArchiveStatus BackendMediaDetailActionStatus
	RestoreStatus BackendMediaDetailActionStatus
	Replaced      bool
	HasError      bool
	Error         string
	Asset         BackendMediaDetailAsset
	Usage         []BackendMediaDetailUsage
}

type BackendMediaDetailActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendMediaDetailAsset struct {
	ID            string
	URL           string
	Alt           string
	Filename      string
	ContentType   string
	FocalX        string
	FocalY        string
	FocalXPercent string
	FocalYPercent string
	IsImage       bool
	CanArchive    bool
	CanRestore    bool
	Variants      []BackendMediaDetailVariant
}

type BackendMediaDetailVariant struct {
	URL        string
	Name       string
	Dimensions string
	Size       string
}

type BackendMediaDetailUsage struct {
	Href  string
	Title string
	Kind  string
}

func RenderBackendMediaDetailPage(props BackendMediaDetailPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-detail-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-media-detail-renderer", "gosx-studio"),
	),
		RenderBackendMediaDetailContent(props),
	)
}

func RenderBackendMediaDetailContent(props BackendMediaDetailPageProps) gosx.Node {
	return gosx.Fragment(
		props.Heading,
		RenderBackendMediaDetailForm(props),
		RenderBackendMediaReplaceForm(props),
	)
}

func RenderBackendMediaDetailForm(props BackendMediaDetailPageProps) gosx.Node {
	asset := props.Asset
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", props.SaveAction),
	),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", asset.ID))),
		gosx.Fragment(renderBackendMediaDetailStatuses(props)...),
		props.Preview,
		renderBackendMediaDetailInput("url", "URL", asset.URL, "field-row field-row--wide", renderBackendMediaDetailFieldError(props.SaveStatus, "url")),
		renderBackendMediaDetailInput("alt", "Alt text", asset.Alt, "", nil),
		renderBackendMediaDetailInput("filename", "Filename", asset.Filename, "", nil),
		renderBackendMediaDetailInput("contentType", "Content type", asset.ContentType, "", nil),
		renderBackendMediaDetailInput("focalX", "Focal X", asset.FocalX, "", renderBackendMediaDetailFieldError(props.SaveStatus, "focalX"),
			gosx.Attr("type", "number"),
			gosx.Attr("min", "0"),
			gosx.Attr("max", "1"),
			gosx.Attr("step", "0.01"),
			gosx.Attr("inputmode", "decimal"),
			gosx.Attr("data-media-focal-x", "true"),
		),
		renderBackendMediaDetailInput("focalY", "Focal Y", asset.FocalY, "", renderBackendMediaDetailFieldError(props.SaveStatus, "focalY"),
			gosx.Attr("type", "number"),
			gosx.Attr("min", "0"),
			gosx.Attr("max", "1"),
			gosx.Attr("step", "0.01"),
			gosx.Attr("inputmode", "decimal"),
			gosx.Attr("data-media-focal-y", "true"),
		),
		renderBackendMediaDetailFocalPreview(asset),
		renderBackendMediaDetailVariants(asset.Variants),
		renderBackendMediaDetailUsage(props.Usage),
		renderBackendMediaDetailButtons(props),
	)
}

func RenderBackendMediaReplaceForm(props BackendMediaDetailPageProps) gosx.Node {
	action := props.ReplaceAction
	if action == "" {
		action = "/admin/media/replace"
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Replace file")),
		),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form admin-form--single"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", action),
			gosx.Attr("enctype", "multipart/form-data"),
			gosx.Attr("data-admin-confirm", "Replace this media file? Existing references will use the new file."),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", props.Asset.ID))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", "replaceFile")), gosx.Text("File")),
				gosx.El("input", gosx.Attrs(gosx.Attr("id", "replaceFile"), gosx.Attr("name", "file"), gosx.Attr("type", "file"), gosx.Attr("accept", backendMediaReplaceAccept))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", "replaceAlt")), gosx.Text("Alt text")),
				gosx.El("input", gosx.Attrs(gosx.Attr("id", "replaceAlt"), gosx.Attr("name", "alt"), gosx.Attr("value", props.Asset.Alt))),
			),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("type", "submit")), gosx.Text("Replace file")),
		),
	)
}

func renderBackendMediaDetailStatuses(props BackendMediaDetailPageProps) []gosx.Node {
	nodes := []gosx.Node{}
	for _, status := range []BackendMediaDetailActionStatus{props.SaveStatus, props.ArchiveStatus, props.RestoreStatus} {
		if status.Submitted {
			nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message)))
		}
	}
	if props.Replaced {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--ok")), gosx.Text("Media file replaced.")))
	}
	if props.HasError {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(props.Error)))
	}
	return nodes
}

func renderBackendMediaDetailInput(id, label, value, className string, errorNode *gosx.Node, extraAttrs ...any) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	attrs := []any{gosx.Attr("id", id), gosx.Attr("name", id)}
	attrs = append(attrs, extraAttrs...)
	attrs = append(attrs, gosx.Attr("value", value))
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(attrs...)),
	}
	if errorNode != nil {
		children = append(children, *errorNode)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func renderBackendMediaDetailFieldError(status BackendMediaDetailActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendMediaDetailFocalPreview(asset BackendMediaDetailAsset) gosx.Node {
	if !asset.IsImage {
		return gosx.Fragment()
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", nil, gosx.Text("Crop preview")),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "media-focal-preview"),
			gosx.Attr("type", "button"),
			gosx.Attr("aria-label", "Set focal point"),
			gosx.Attr("data-media-focal-preview", "true"),
		),
			gosx.El("img", gosx.Attrs(gosx.Attr("src", asset.URL), gosx.Attr("alt", asset.Alt))),
			gosx.El("span", gosx.Attrs(
				gosx.Attr("style", "left:"+asset.FocalXPercent+"; top:"+asset.FocalYPercent),
				gosx.Attr("data-media-focal-marker", "true"),
			)),
		),
	)
}

func renderBackendMediaDetailVariants(variants []BackendMediaDetailVariant) gosx.Node {
	if len(variants) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(variants))
	for _, variant := range variants {
		nodes = append(nodes, gosx.El("li", nil,
			gosx.El("a", gosx.Attrs(gosx.Attr("href", variant.URL)), gosx.Text(variant.Name)),
			gosx.El("span", nil, gosx.Text(variant.Dimensions+" "+variant.Size)),
		))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", nil, gosx.Text("Variants")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list")), gosx.Fragment(nodes...)),
	)
}

func renderBackendMediaDetailUsage(usage []BackendMediaDetailUsage) gosx.Node {
	if len(usage) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(usage))
	for _, value := range usage {
		nodes = append(nodes, gosx.El("li", nil,
			gosx.El("a", gosx.Attrs(gosx.Attr("href", value.Href), gosx.Attr("data-gosx-link", "true")), gosx.Text(value.Title)),
			gosx.El("span", nil, gosx.Text(value.Kind)),
		))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", nil, gosx.Text("Used by")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list")), gosx.Fragment(nodes...)),
	)
}

func renderBackendMediaDetailButtons(props BackendMediaDetailPageProps) gosx.Node {
	asset := props.Asset
	nodes := []gosx.Node{
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save media")),
	}
	if asset.CanArchive {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.ArchiveAction),
			gosx.Attr("data-admin-confirm", "Archive this media asset? Existing content may still reference this URL."),
		), gosx.Text("Archive")))
	}
	if asset.CanRestore {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.RestoreAction),
			gosx.Attr("data-admin-confirm", "Restore this media asset?"),
		), gosx.Text("Restore")))
	}
	nodes = append(nodes, gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", "/admin/media"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Back to media")))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(nodes...))
}
