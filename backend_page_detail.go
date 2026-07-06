package studio

import "m31labs.dev/gosx"

type BackendPageDetailPageProps struct {
	Class                 string
	Heading               gosx.Node
	Preview               gosx.Node
	RevisionHistory       gosx.Node
	Media                 []BackendPageDetailMediaAsset
	SaveAction            string
	ArchiveAction         string
	RestoreAction         string
	CSRFToken             string
	SaveStatus            BackendPageDetailActionStatus
	ArchiveStatus         BackendPageDetailActionStatus
	RestoreStatus         BackendPageDetailActionStatus
	RestoreRevisionStatus BackendPageDetailActionStatus
	RevisionRestored      bool
	Page                  BackendPageDetailValues
}

type BackendPageDetailActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendPageDetailMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendPageDetailValues struct {
	ID               string
	Title            string
	Slug             string
	BodyFormatBlocks bool
	BodyFormatMDPP   bool
	Body             string
	MetaTitle        string
	MetaImageURL     string
	MetaDescription  string
	MetaImageAlt     string
	Published        bool
	CanArchive       bool
	CanRestore       bool
	PreviewHref      string
}

func RenderBackendPageDetailPage(props BackendPageDetailPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-detail-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-page-detail-renderer", "gosx-studio"),
	),
		RenderBackendPageDetailContent(props),
	)
}

func RenderBackendPageDetailContent(props BackendPageDetailPageProps) gosx.Node {
	return gosx.Fragment(
		props.Heading,
		RenderBackendPageDetailMediaDatalist(props.Media),
		RenderBackendPageDetailForm(props),
		props.RevisionHistory,
		RenderBackendPageDetailScripts(),
	)
}

func RenderBackendPageDetailMediaDatalist(media []BackendPageDetailMediaAsset) gosx.Node {
	nodes := make([]gosx.Node, 0, len(media))
	for _, asset := range media {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", asset.URL),
			gosx.Attr("label", asset.Filename),
			gosx.Attr("data-media-alt", asset.Alt),
		), gosx.Text(asset.Filename)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "page-media-urls")), gosx.Fragment(nodes...))
}

func RenderBackendPageDetailForm(props BackendPageDetailPageProps) gosx.Node {
	page := props.Page
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", props.SaveAction),
	),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", page.ID))),
		gosx.Fragment(renderBackendPageDetailStatuses(props)...),
		props.Preview,
		renderBackendPageDetailInput("title", "Title", page.Title, "", renderBackendPageDetailFieldError(props.SaveStatus, "title")),
		renderBackendPageDetailInput("slug", "Slug", page.Slug, "", nil),
		renderBackendPageDetailBodyFormat(page),
		renderBackendPageDetailContentEditor(page.Body),
		renderBackendPageDetailSEO(page),
		renderBackendPageDetailChecks(page),
		renderBackendPageDetailButtons(props),
	)
}

func RenderBackendPageDetailScripts() gosx.Node {
	return gosx.Fragment(
		gosx.El("script", gosx.Attrs(gosx.Attr("src", "/content-editor.js"), gosx.Attr("defer", true))),
		gosx.El("script", gosx.Attrs(gosx.Attr("src", "/media-picker.js"), gosx.Attr("defer", true))),
	)
}

func renderBackendPageDetailStatuses(props BackendPageDetailPageProps) []gosx.Node {
	nodes := []gosx.Node{}
	for _, status := range []BackendPageDetailActionStatus{props.SaveStatus, props.ArchiveStatus, props.RestoreStatus, props.RestoreRevisionStatus} {
		if status.Submitted {
			nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message)))
		}
	}
	if props.RevisionRestored {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--ok")), gosx.Text("Version restored.")))
	}
	return nodes
}

func renderBackendPageDetailInput(id, label, value, className string, errorNode *gosx.Node, extraAttrs ...any) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	attrs := []any{gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("value", value)}
	attrs = append(attrs, extraAttrs...)
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(attrs...)),
	}
	if errorNode != nil {
		children = append(children, *errorNode)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func renderBackendPageDetailTextarea(id, label, value, rows, className string) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)), gosx.Text(value)),
	)
}

func renderBackendPageDetailFieldError(status BackendPageDetailActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendPageDetailBodyFormat(page BackendPageDetailValues) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "bodyFormat")), gosx.Text("Body format")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", "bodyFormat"), gosx.Attr("name", "bodyFormat"), gosx.Attr("data-content-editor-format", "true")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "blocks"), gosx.Attr("selected", page.BodyFormatBlocks)), gosx.Text("GoSX blocks")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "mdpp"), gosx.Attr("selected", page.BodyFormatMDPP)), gosx.Text("MDPP")),
		),
	)
}

func renderBackendPageDetailContentEditor(body string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide content-editor"), gosx.Attr("data-content-editor", "true"), gosx.Attr("data-content-editor-media-list", "page-media-urls")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "body")), gosx.Text("Body")),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("class", "content-editor__source"), gosx.Attr("id", "body"), gosx.Attr("name", "body"), gosx.Attr("rows", "10"), gosx.Attr("data-content-editor-source", "true")), gosx.Text(body)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-editor__toolbar")),
			renderBackendPageDetailToolbarButton("heading", "Heading"),
			renderBackendPageDetailToolbarButton("paragraph", "Paragraph"),
			renderBackendPageDetailToolbarButton("image", "Image"),
			renderBackendPageDetailToolbarButton("gallery", "Gallery"),
			renderBackendPageDetailToolbarButton("button", "Button"),
			renderBackendPageDetailToolbarButton("product", "Product"),
			renderBackendPageDetailToolbarButton("quote", "Quote"),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-block-list"), gosx.Attr("data-content-editor-list", "true"))),
	)
}

func renderBackendPageDetailToolbarButton(kind, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "home-section-move"), gosx.Attr("data-content-add", kind)), gosx.Text(label))
}

func renderBackendPageDetailSEO(page BackendPageDetailValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendPageDetailInput("metaTitle", "Meta title", page.MetaTitle, "", nil),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "metaImageUrl")), gosx.Text("Meta image URL")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "metaImageUrl"),
				gosx.Attr("name", "metaImageUrl"),
				gosx.Attr("list", "page-media-urls"),
				gosx.Attr("value", page.MetaImageURL),
				gosx.Attr("data-media-alt-target", "metaImageAlt"),
			)),
		),
		renderBackendPageDetailTextarea("metaDescription", "Meta description", page.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendPageDetailInput("metaImageAlt", "Meta image alt text", page.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendPageDetailChecks(page BackendPageDetailValues) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", page.Published))), gosx.Text(" Published")),
	)
}

func renderBackendPageDetailButtons(props BackendPageDetailPageProps) gosx.Node {
	page := props.Page
	nodes := []gosx.Node{
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save page")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", page.PreviewHref), gosx.Attr("data-gosx-link", "true")), gosx.Text("Preview")),
	}
	if page.CanArchive {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.ArchiveAction),
			gosx.Attr("data-admin-confirm", "Archive this page? It will be unpublished and hidden from public routes."),
		), gosx.Text("Archive")))
	}
	if page.CanRestore {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.RestoreAction),
			gosx.Attr("data-admin-confirm", "Restore this page?"),
		), gosx.Text("Restore")))
	}
	nodes = append(nodes,
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", "/admin/pages"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Back to pages")),
	)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(nodes...))
}
