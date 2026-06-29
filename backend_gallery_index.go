package studio

import "m31labs.dev/gosx"

type BackendGalleryIndexPageProps struct {
	Class                string
	ResourceIndexContent gosx.Node
	Media                []BackendGalleryIndexMediaAsset
	CreateAction         string
	CSRFToken            string
	CreateStatus         BackendGalleryIndexActionStatus
	Work                 BackendGalleryIndexValues
}

type BackendGalleryIndexActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendGalleryIndexMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendGalleryIndexValues struct {
	Title           string
	Year            string
	Materials       string
	Dimensions      string
	ImagesText      string
	Description     string
	MetaTitle       string
	MetaImageURL    string
	MetaDescription string
	MetaImageAlt    string
}

func RenderBackendGalleryIndexPage(props BackendGalleryIndexPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-resource-index-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-gallery-index-renderer", "gosx-studio"),
	),
		RenderBackendGalleryIndexContent(props),
	)
}

func RenderBackendGalleryIndexContent(props BackendGalleryIndexPageProps) gosx.Node {
	return gosx.Fragment(
		props.ResourceIndexContent,
		RenderBackendGalleryCreatePanel(props),
		RenderBackendGalleryIndexScripts(),
	)
}

func RenderBackendGalleryCreatePanel(props BackendGalleryIndexPageProps) gosx.Node {
	work := props.Work
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Add gallery work")),
		),
		renderBackendGalleryIndexStatus(props.CreateStatus),
		RenderBackendGalleryIndexMediaDatalist(props.Media),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", props.CreateAction),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			renderBackendGalleryIndexInput("title", "Title", work.Title, "", renderBackendGalleryIndexFieldError(props.CreateStatus, "title")),
			renderBackendGalleryIndexInput("year", "Year", work.Year, "", nil),
			renderBackendGalleryIndexInput("materials", "Materials", work.Materials, "", nil),
			renderBackendGalleryIndexInput("dimensions", "Dimensions", work.Dimensions, "", nil),
			renderBackendGalleryIndexTextarea("imagesText", "Images", work.ImagesText, "4", "field-row field-row--wide", gosx.Attr("data-media-lines-list", "gallery-media-urls")),
			renderBackendGalleryIndexTextarea("description", "Description", work.Description, "4", "field-row field-row--wide"),
			renderBackendGalleryIndexSEO(work),
			renderBackendGalleryIndexChecks(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Create gallery work")),
		),
	)
}

func RenderBackendGalleryIndexMediaDatalist(media []BackendGalleryIndexMediaAsset) gosx.Node {
	nodes := make([]gosx.Node, 0, len(media))
	for _, asset := range media {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", asset.URL),
			gosx.Attr("label", asset.Filename),
			gosx.Attr("data-media-alt", asset.Alt),
		), gosx.Text(asset.Filename)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "gallery-media-urls")), gosx.Fragment(nodes...))
}

func RenderBackendGalleryIndexScripts() gosx.Node {
	return gosx.El("script", gosx.Attrs(gosx.Attr("src", "/media-picker.js"), gosx.Attr("defer", true)))
}

func renderBackendGalleryIndexStatus(status BackendGalleryIndexActionStatus) gosx.Node {
	if !status.Submitted {
		return gosx.Fragment()
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message))
}

func renderBackendGalleryIndexInput(id, label, value, className string, errorNode *gosx.Node) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("value", value))),
	}
	if errorNode != nil {
		children = append(children, *errorNode)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func renderBackendGalleryIndexTextarea(id, label, value, rows, className string, extraAttrs ...any) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	attrs := []any{gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)}
	attrs = append(attrs, extraAttrs...)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(attrs...), gosx.Text(value)),
	)
}

func renderBackendGalleryIndexFieldError(status BackendGalleryIndexActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendGalleryIndexSEO(work BackendGalleryIndexValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendGalleryIndexInput("metaTitle", "Meta title", work.MetaTitle, "", nil),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "metaImageUrl")), gosx.Text("Meta image URL")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "metaImageUrl"),
				gosx.Attr("name", "metaImageUrl"),
				gosx.Attr("list", "gallery-media-urls"),
				gosx.Attr("value", work.MetaImageURL),
				gosx.Attr("data-media-alt-target", "metaImageAlt"),
			)),
		),
		renderBackendGalleryIndexTextarea("metaDescription", "Meta description", work.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendGalleryIndexInput("metaImageAlt", "Meta image alt text", work.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendGalleryIndexChecks() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", "checked"))), gosx.Text(" Published")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "featured"))), gosx.Text(" Featured")),
	)
}
