package studio

import "m31labs.dev/gosx"

type BackendPageIndexPageProps struct {
	Class                string
	ResourceIndexContent gosx.Node
	Media                []BackendPageIndexMediaAsset
	CreateAction         string
	CSRFToken            string
	CreateStatus         BackendPageIndexActionStatus
	Page                 BackendPageIndexValues
}

type BackendPageIndexActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendPageIndexMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendPageIndexValues struct {
	Title           string
	Slug            string
	Body            string
	MetaTitle       string
	MetaImageURL    string
	MetaDescription string
	MetaImageAlt    string
}

func RenderBackendPageIndexPage(props BackendPageIndexPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-resource-index-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-page-index-renderer", "gosx-studio"),
	),
		RenderBackendPageIndexContent(props),
	)
}

func RenderBackendPageIndexContent(props BackendPageIndexPageProps) gosx.Node {
	return gosx.Fragment(
		props.ResourceIndexContent,
		RenderBackendPageCreatePanel(props),
		RenderBackendPageIndexScripts(),
	)
}

func RenderBackendPageCreatePanel(props BackendPageIndexPageProps) gosx.Node {
	page := props.Page
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Add page")),
		),
		renderBackendPageIndexStatus(props.CreateStatus),
		RenderBackendPageIndexMediaDatalist(props.Media),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", props.CreateAction),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			renderBackendPageIndexInput("title", "Title", page.Title, "", renderBackendPageIndexFieldError(props.CreateStatus, "title")),
			renderBackendPageIndexInput("slug", "Slug", page.Slug, "", nil),
			renderBackendPageIndexBodyFormat(),
			renderBackendPageIndexContentEditor(page.Body),
			renderBackendPageIndexSEO(page),
			renderBackendPageIndexChecks(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Create page")),
		),
	)
}

func RenderBackendPageIndexMediaDatalist(media []BackendPageIndexMediaAsset) gosx.Node {
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

func RenderBackendPageIndexScripts() gosx.Node {
	return gosx.Fragment(
		gosx.El("script", gosx.Attrs(gosx.Attr("src", "/content-editor.js"), gosx.Attr("defer", true))),
		gosx.El("script", gosx.Attrs(gosx.Attr("src", "/media-picker.js"), gosx.Attr("defer", true))),
	)
}

func renderBackendPageIndexStatus(status BackendPageIndexActionStatus) gosx.Node {
	if !status.Submitted {
		return gosx.Fragment()
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message))
}

func renderBackendPageIndexInput(id, label, value, className string, errorNode *gosx.Node) gosx.Node {
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

func renderBackendPageIndexTextarea(id, label, value, rows, className string) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)), gosx.Text(value)),
	)
}

func renderBackendPageIndexFieldError(status BackendPageIndexActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendPageIndexBodyFormat() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "bodyFormat")), gosx.Text("Body format")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", "bodyFormat"), gosx.Attr("name", "bodyFormat"), gosx.Attr("data-content-editor-format", "true")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "blocks"), gosx.Attr("selected", "selected")), gosx.Text("GoSX blocks")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "mdpp")), gosx.Text("MDPP")),
		),
	)
}

func renderBackendPageIndexContentEditor(body string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide content-editor"), gosx.Attr("data-content-editor", "true"), gosx.Attr("data-content-editor-media-list", "page-media-urls")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "body")), gosx.Text("Body")),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("class", "content-editor__source"), gosx.Attr("id", "body"), gosx.Attr("name", "body"), gosx.Attr("rows", "8"), gosx.Attr("data-content-editor-source", "true")), gosx.Text(body)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-editor__toolbar")),
			renderBackendPageIndexToolbarButton("heading", "Heading"),
			renderBackendPageIndexToolbarButton("paragraph", "Paragraph"),
			renderBackendPageIndexToolbarButton("image", "Image"),
			renderBackendPageIndexToolbarButton("gallery", "Gallery"),
			renderBackendPageIndexToolbarButton("button", "Button"),
			renderBackendPageIndexToolbarButton("product", "Product"),
			renderBackendPageIndexToolbarButton("quote", "Quote"),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-block-list"), gosx.Attr("data-content-editor-list", "true"))),
	)
}

func renderBackendPageIndexToolbarButton(kind, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "home-section-move"), gosx.Attr("data-content-add", kind)), gosx.Text(label))
}

func renderBackendPageIndexSEO(page BackendPageIndexValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendPageIndexInput("metaTitle", "Meta title", page.MetaTitle, "", nil),
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
		renderBackendPageIndexTextarea("metaDescription", "Meta description", page.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendPageIndexInput("metaImageAlt", "Meta image alt text", page.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendPageIndexChecks() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", "checked"))), gosx.Text(" Published")),
	)
}
