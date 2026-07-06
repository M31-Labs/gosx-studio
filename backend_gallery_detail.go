package studio

import "m31labs.dev/gosx"

type BackendGalleryDetailPageProps struct {
	Class         string
	Heading       gosx.Node
	Preview       gosx.Node
	Media         []BackendGalleryDetailMediaAsset
	SaveAction    string
	ArchiveAction string
	RestoreAction string
	CSRFToken     string
	SaveStatus    BackendGalleryDetailActionStatus
	ArchiveStatus BackendGalleryDetailActionStatus
	RestoreStatus BackendGalleryDetailActionStatus
	Work          BackendGalleryDetailValues
}

type BackendGalleryDetailActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendGalleryDetailMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendGalleryDetailValues struct {
	ID              string
	Title           string
	Slug            string
	Year            string
	Materials       string
	Dimensions      string
	ImagesText      string
	Description     string
	MetaTitle       string
	MetaImageURL    string
	MetaDescription string
	MetaImageAlt    string
	Published       bool
	Featured        bool
	CanArchive      bool
	CanRestore      bool
	PreviewHref     string
	Images          []BackendGalleryDetailImage
}

type BackendGalleryDetailImage struct {
	URL string
	Alt string
}

func RenderBackendGalleryDetailPage(props BackendGalleryDetailPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-detail-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-gallery-detail-renderer", "gosx-studio"),
	),
		RenderBackendGalleryDetailContent(props),
	)
}

func RenderBackendGalleryDetailContent(props BackendGalleryDetailPageProps) gosx.Node {
	return gosx.Fragment(
		props.Heading,
		RenderBackendGalleryDetailMediaDatalist(props.Media),
		RenderBackendGalleryDetailForm(props),
		RenderBackendGalleryDetailScripts(),
	)
}

func RenderBackendGalleryDetailMediaDatalist(media []BackendGalleryDetailMediaAsset) gosx.Node {
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

func RenderBackendGalleryDetailForm(props BackendGalleryDetailPageProps) gosx.Node {
	work := props.Work
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", props.SaveAction),
	),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", work.ID))),
		gosx.Fragment(renderBackendGalleryDetailStatuses(props)...),
		props.Preview,
		renderBackendGalleryDetailImages(work.Images),
		renderBackendGalleryDetailInput("title", "Title", work.Title, "", renderBackendGalleryDetailFieldError(props.SaveStatus, "title")),
		renderBackendGalleryDetailInput("slug", "Slug", work.Slug, "", nil),
		renderBackendGalleryDetailInput("year", "Year", work.Year, "", nil),
		renderBackendGalleryDetailInput("materials", "Materials", work.Materials, "", nil),
		renderBackendGalleryDetailInput("dimensions", "Dimensions", work.Dimensions, "", nil),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "imagesText")), gosx.Text("Images")),
			gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "imagesText"), gosx.Attr("name", "imagesText"), gosx.Attr("rows", "5"), gosx.Attr("data-media-lines-list", "gallery-media-urls")), gosx.Text(work.ImagesText)),
		),
		renderBackendGalleryDetailTextarea("description", "Description", work.Description, "5", "field-row field-row--wide"),
		renderBackendGalleryDetailSEO(work),
		renderBackendGalleryDetailChecks(work),
		renderBackendGalleryDetailButtons(props),
	)
}

func RenderBackendGalleryDetailScripts() gosx.Node {
	return gosx.El("script", gosx.Attrs(gosx.Attr("src", "/media-picker.js"), gosx.Attr("defer", true)))
}

func renderBackendGalleryDetailStatuses(props BackendGalleryDetailPageProps) []gosx.Node {
	nodes := []gosx.Node{}
	for _, status := range []BackendGalleryDetailActionStatus{props.SaveStatus, props.ArchiveStatus, props.RestoreStatus} {
		if status.Submitted {
			nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message)))
		}
	}
	return nodes
}

func renderBackendGalleryDetailImages(images []BackendGalleryDetailImage) gosx.Node {
	if len(images) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(images))
	for _, image := range images {
		nodes = append(nodes, gosx.El("img", gosx.Attrs(gosx.Attr("src", image.URL), gosx.Attr("alt", image.Alt))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "media-strip media-strip--compact")), gosx.Fragment(nodes...))
}

func renderBackendGalleryDetailInput(id, label, value, className string, errorNode *gosx.Node) gosx.Node {
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

func renderBackendGalleryDetailTextarea(id, label, value, rows, className string) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)), gosx.Text(value)),
	)
}

func renderBackendGalleryDetailFieldError(status BackendGalleryDetailActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendGalleryDetailSEO(work BackendGalleryDetailValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendGalleryDetailInput("metaTitle", "Meta title", work.MetaTitle, "", nil),
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
		renderBackendGalleryDetailTextarea("metaDescription", "Meta description", work.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendGalleryDetailInput("metaImageAlt", "Meta image alt text", work.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendGalleryDetailChecks(work BackendGalleryDetailValues) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", work.Published))), gosx.Text(" Published")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "featured"), gosx.Attr("checked", work.Featured))), gosx.Text(" Featured")),
	)
}

func renderBackendGalleryDetailButtons(props BackendGalleryDetailPageProps) gosx.Node {
	work := props.Work
	nodes := []gosx.Node{
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save gallery work")),
	}
	if work.CanArchive {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.ArchiveAction),
			gosx.Attr("data-admin-confirm", "Archive this gallery work? It will be hidden from public gallery pages."),
		), gosx.Text("Archive")))
	}
	if work.CanRestore {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.RestoreAction),
			gosx.Attr("data-admin-confirm", "Restore this gallery work to the admin catalog?"),
		), gosx.Text("Restore")))
	}
	nodes = append(nodes,
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", work.PreviewHref), gosx.Attr("data-gosx-link", "true")), gosx.Text("Preview")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", "/admin/gallery"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Back to gallery")),
	)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(nodes...))
}
