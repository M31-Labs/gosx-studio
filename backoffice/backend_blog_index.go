package backoffice

import "m31labs.dev/gosx"

type BackendBlogIndexPageProps struct {
	Class                string
	ResourceIndexContent gosx.Node
	Media                []BackendBlogIndexMediaAsset
	Tags                 []BackendBlogIndexTag
	Products             []BackendBlogIndexRelation
	GalleryWorks         []BackendBlogIndexRelation
	CreateAction         string
	CSRFToken            string
	CreateStatus         BackendBlogIndexActionStatus
	Post                 BackendBlogIndexValues
}

type BackendBlogIndexActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendBlogIndexMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendBlogIndexTag struct {
	Label string
	Slug  string
	Href  string
}

type BackendBlogIndexRelation struct {
	FieldName string
	Title     string
}

type BackendBlogIndexValues struct {
	Title           string
	Slug            string
	AuthorName      string
	PublishedAt     string
	CoverImage      string
	CoverImageAlt   string
	Excerpt         string
	Body            string
	Tags            string
	MetaTitle       string
	MetaImageURL    string
	MetaDescription string
	MetaImageAlt    string
}

func RenderBackendBlogIndexPage(props BackendBlogIndexPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-resource-index-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-blog-index-renderer", "gosx-studio"),
	),
		RenderBackendBlogIndexContent(props),
	)
}

func RenderBackendBlogIndexContent(props BackendBlogIndexPageProps) gosx.Node {
	return gosx.Fragment(
		props.ResourceIndexContent,
		RenderBackendBlogCreatePanel(props),
		RenderBackendBlogIndexScripts(),
	)
}

func RenderBackendBlogCreatePanel(props BackendBlogIndexPageProps) gosx.Node {
	post := props.Post
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Add post")),
		),
		renderBackendBlogIndexStatus(props.CreateStatus),
		RenderBackendBlogIndexMediaDatalist(props.Media),
		RenderBackendBlogIndexTagsDatalist(props.Tags),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", props.CreateAction),
			gosx.Attr("data-content-editor-enhanced-submit", "true"),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			renderBackendBlogIndexButtons(),
			renderBackendBlogIndexInput("title", "Title", post.Title, "", renderBackendBlogIndexFieldError(props.CreateStatus, "title")),
			renderBackendBlogIndexInput("slug", "Slug", post.Slug, "", nil),
			renderBackendBlogIndexInput("authorName", "Author", post.AuthorName, "", nil),
			renderBackendBlogIndexInput("publishedAt", "Publish date", post.PublishedAt, "", renderBackendBlogIndexFieldError(props.CreateStatus, "publishedAt"), gosx.Attr("type", "datetime-local"), gosx.Attr("data-viewer-datetime-input", "true")),
			renderBackendBlogIndexMediaInput("coverImage", "Cover image URL", post.CoverImage, "coverImageAlt"),
			renderBackendBlogIndexInput("coverImageAlt", "Cover image alt text", post.CoverImageAlt, "field-row field-row--wide", nil),
			renderBackendBlogIndexTextarea("excerpt", "Excerpt", post.Excerpt, "3", "field-row field-row--wide"),
			renderBackendBlogIndexBodyFormat(),
			renderBackendBlogIndexContentEditor(post.Body),
			renderBackendBlogIndexInput("tags", "Tags", post.Tags, "field-row field-row--wide", nil, gosx.Attr("list", "blog-tags")),
			renderBackendBlogIndexCurrentTags(props.Tags),
			renderBackendBlogIndexSEO(post),
			renderBackendBlogIndexRelations("Related products", props.Products),
			renderBackendBlogIndexRelations("Related gallery works", props.GalleryWorks),
			renderBackendBlogIndexChecks(),
		),
	)
}

func RenderBackendBlogIndexMediaDatalist(media []BackendBlogIndexMediaAsset) gosx.Node {
	nodes := make([]gosx.Node, 0, len(media))
	for _, asset := range media {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", asset.URL),
			gosx.Attr("label", asset.Filename),
			gosx.Attr("data-media-alt", asset.Alt),
		), gosx.Text(asset.Filename)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "blog-media-urls")), gosx.Fragment(nodes...))
}

func RenderBackendBlogIndexTagsDatalist(tags []BackendBlogIndexTag) gosx.Node {
	nodes := make([]gosx.Node, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(gosx.Attr("value", tag.Label)), gosx.Text(tag.Slug)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "blog-tags")), gosx.Fragment(nodes...))
}

func RenderBackendBlogIndexScripts() gosx.Node {
	return renderBackendManagedStudioScripts()
}

func renderBackendBlogIndexStatus(status BackendBlogIndexActionStatus) gosx.Node {
	if !status.Submitted {
		return gosx.Fragment()
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message))
}

func renderBackendBlogIndexInput(id, label, value, className string, errorNode *gosx.Node, extraAttrs ...any) gosx.Node {
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

func renderBackendBlogIndexTextarea(id, label, value, rows, className string) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)), gosx.Text(value)),
	)
}

func renderBackendBlogIndexFieldError(status BackendBlogIndexActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendBlogIndexMediaInput(id, label, value, altTarget string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("id", id),
			gosx.Attr("name", id),
			gosx.Attr("list", "blog-media-urls"),
			gosx.Attr("value", value),
			gosx.Attr("data-media-alt-target", altTarget),
		)),
	)
}

func renderBackendBlogIndexBodyFormat() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "bodyFormat")), gosx.Text("Body format")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", "bodyFormat"), gosx.Attr("name", "bodyFormat"), gosx.Attr("data-content-editor-format", "true")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "blocks"), gosx.Attr("selected", "selected")), gosx.Text("GoSX blocks")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "mdpp")), gosx.Text("MDPP")),
		),
	)
}

func renderBackendBlogIndexContentEditor(body string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide content-editor"), gosx.Attr("data-content-editor", "true"), gosx.Attr("data-content-editor-media-list", "blog-media-urls")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "body")), gosx.Text("Body")),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("class", "content-editor__source"), gosx.Attr("id", "body"), gosx.Attr("name", "body"), gosx.Attr("rows", "9"), gosx.Attr("data-content-editor-source", "true")), gosx.Text(body)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-editor__toolbar")),
			renderBackendBlogIndexToolbarButton("heading", "Heading"),
			renderBackendBlogIndexToolbarButton("paragraph", "Paragraph"),
			renderBackendBlogIndexToolbarButton("image", "Image"),
			renderBackendBlogIndexToolbarButton("gallery", "Gallery"),
			renderBackendBlogIndexToolbarButton("button", "Button"),
			renderBackendBlogIndexToolbarButton("product", "Product"),
			renderBackendBlogIndexToolbarButton("quote", "Quote"),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-block-list"), gosx.Attr("data-content-editor-list", "true"))),
	)
}

func renderBackendBlogIndexToolbarButton(kind, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "home-section-move"), gosx.Attr("data-content-add", kind)), gosx.Text(label))
}

func renderBackendBlogIndexCurrentTags(tags []BackendBlogIndexTag) gosx.Node {
	if len(tags) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(gosx.Attr("href", tag.Href)), gosx.Text(tag.Label)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", nil, gosx.Text("Current tags")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "tag-list")), gosx.Fragment(nodes...)),
	)
}

func renderBackendBlogIndexSEO(post BackendBlogIndexValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendBlogIndexInput("metaTitle", "Meta title", post.MetaTitle, "", nil),
		renderBackendBlogIndexMediaInput("metaImageUrl", "Meta image URL", post.MetaImageURL, "metaImageAlt"),
		renderBackendBlogIndexTextarea("metaDescription", "Meta description", post.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendBlogIndexInput("metaImageAlt", "Meta image alt text", post.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendBlogIndexRelations(label string, values []BackendBlogIndexRelation) gosx.Node {
	if len(values) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(values))
	for _, value := range values {
		nodes = append(nodes, gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", value.FieldName))),
			gosx.Text(" "+value.Title),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "radio-row")),
		gosx.El("legend", nil, gosx.Text(label)),
		gosx.Fragment(nodes...),
	)
}

func renderBackendBlogIndexChecks() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", "checked"))), gosx.Text(" Published")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "featured"))), gosx.Text(" Featured")),
	)
}

func renderBackendBlogIndexButtons() gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "button-row admin-form__primary-actions"),
		gosx.Attr("data-content-editor-primary-actions", "true"),
	), gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "button button--primary"),
		gosx.Attr("type", "submit"),
	), gosx.Text("Create post")))
}
