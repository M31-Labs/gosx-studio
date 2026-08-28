package backoffice

import "m31labs.dev/gosx"

type BackendBlogDetailPageProps struct {
	Class                 string
	Heading               gosx.Node
	Preview               gosx.Node
	RevisionHistory       gosx.Node
	Media                 []BackendBlogDetailMediaAsset
	Tags                  []BackendBlogDetailTag
	Products              []BackendBlogDetailRelation
	GalleryWorks          []BackendBlogDetailRelation
	SaveAction            string
	ArchiveAction         string
	RestoreAction         string
	CSRFToken             string
	SaveStatus            BackendBlogDetailActionStatus
	ArchiveStatus         BackendBlogDetailActionStatus
	RestoreStatus         BackendBlogDetailActionStatus
	RestoreRevisionStatus BackendBlogDetailActionStatus
	RevisionRestored      bool
	Post                  BackendBlogDetailValues
}

type BackendBlogDetailActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendBlogDetailMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendBlogDetailTag struct {
	Label string
	Slug  string
	Href  string
}

type BackendBlogDetailRelation struct {
	FieldName string
	Title     string
	Checked   bool
}

type BackendBlogDetailValues struct {
	ID                     string
	Title                  string
	Slug                   string
	AuthorName             string
	PublishedAtInput       string
	PublishedAtInputSource string
	CoverImage             string
	CoverImageAlt          string
	Excerpt                string
	BodyFormatBlocks       bool
	BodyFormatMDPP         bool
	Body                   string
	TagSummary             string
	MetaTitle              string
	MetaImageURL           string
	MetaDescription        string
	MetaImageAlt           string
	Published              bool
	Featured               bool
	CanArchive             bool
	CanRestore             bool
	PreviewHref            string
}

func RenderBackendBlogDetailPage(props BackendBlogDetailPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-detail-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-blog-detail-renderer", "gosx-studio"),
	),
		RenderBackendBlogDetailContent(props),
	)
}

func RenderBackendBlogDetailContent(props BackendBlogDetailPageProps) gosx.Node {
	return gosx.Fragment(
		props.Heading,
		RenderBackendBlogDetailMediaDatalist(props.Media),
		RenderBackendBlogDetailTagsDatalist(props.Tags),
		RenderBackendBlogDetailForm(props),
		props.RevisionHistory,
		RenderBackendBlogDetailScripts(),
	)
}

func RenderBackendBlogDetailMediaDatalist(media []BackendBlogDetailMediaAsset) gosx.Node {
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

func RenderBackendBlogDetailTagsDatalist(tags []BackendBlogDetailTag) gosx.Node {
	nodes := make([]gosx.Node, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(gosx.Attr("value", tag.Label)), gosx.Text(tag.Slug)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "blog-tags")), gosx.Fragment(nodes...))
}

func RenderBackendBlogDetailForm(props BackendBlogDetailPageProps) gosx.Node {
	post := props.Post
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", props.SaveAction),
		gosx.Attr("data-content-editor-enhanced-submit", "true"),
	),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", post.ID))),
		gosx.Fragment(renderBackendBlogDetailStatuses(props)...),
		renderBackendBlogDetailPrimaryButtons(props),
		props.Preview,
		renderBackendBlogDetailInput("title", "Title", post.Title, "", renderBackendBlogDetailFieldError(props.SaveStatus, "title")),
		renderBackendBlogDetailInput("slug", "Slug", post.Slug, "", nil),
		renderBackendBlogDetailInput("authorName", "Author", post.AuthorName, "", nil),
		renderBackendBlogDetailInput("publishedAt", "Publish date", post.PublishedAtInput, "", renderBackendBlogDetailFieldError(props.SaveStatus, "publishedAt"), gosx.Attr("type", "datetime-local"), gosx.Attr("data-viewer-datetime-input", "true"), gosx.Attr("data-viewer-time-source", post.PublishedAtInputSource)),
		renderBackendBlogDetailMediaInput("coverImage", "Cover image URL", post.CoverImage, "coverImageAlt"),
		renderBackendBlogDetailInput("coverImageAlt", "Cover image alt text", post.CoverImageAlt, "field-row field-row--wide", nil),
		renderBackendBlogDetailTextarea("excerpt", "Excerpt", post.Excerpt, "3", "field-row field-row--wide"),
		renderBackendBlogDetailBodyFormat(post),
		renderBackendBlogDetailContentEditor(post.Body),
		renderBackendBlogDetailInput("tags", "Tags", post.TagSummary, "field-row field-row--wide", nil, gosx.Attr("list", "blog-tags")),
		renderBackendBlogDetailCurrentTags(props.Tags),
		renderBackendBlogDetailSEO(post),
		renderBackendBlogDetailRelations("Related products", props.Products),
		renderBackendBlogDetailRelations("Related gallery works", props.GalleryWorks),
		renderBackendBlogDetailChecks(post),
		renderBackendBlogDetailSecondaryButtons(props),
	)
}

func RenderBackendBlogDetailScripts() gosx.Node {
	return renderBackendManagedStudioScripts()
}

func renderBackendBlogDetailStatuses(props BackendBlogDetailPageProps) []gosx.Node {
	nodes := []gosx.Node{}
	for _, status := range []BackendBlogDetailActionStatus{props.SaveStatus, props.ArchiveStatus, props.RestoreStatus, props.RestoreRevisionStatus} {
		if status.Submitted {
			nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message)))
		}
	}
	if props.RevisionRestored {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--ok")), gosx.Text("Version restored.")))
	}
	return nodes
}

func renderBackendBlogDetailInput(id, label, value, className string, errorNode *gosx.Node, extraAttrs ...any) gosx.Node {
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

func renderBackendBlogDetailTextarea(id, label, value, rows, className string) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)), gosx.Text(value)),
	)
}

func renderBackendBlogDetailFieldError(status BackendBlogDetailActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendBlogDetailMediaInput(id, label, value, altTarget string) gosx.Node {
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

func renderBackendBlogDetailBodyFormat(post BackendBlogDetailValues) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "bodyFormat")), gosx.Text("Body format")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", "bodyFormat"), gosx.Attr("name", "bodyFormat"), gosx.Attr("data-content-editor-format", "true")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "blocks"), gosx.Attr("selected", post.BodyFormatBlocks)), gosx.Text("GoSX blocks")),
			gosx.El("option", gosx.Attrs(gosx.Attr("value", "mdpp"), gosx.Attr("selected", post.BodyFormatMDPP)), gosx.Text("MDPP")),
		),
	)
}

func renderBackendBlogDetailContentEditor(body string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide content-editor"), gosx.Attr("data-content-editor", "true"), gosx.Attr("data-content-editor-media-list", "blog-media-urls")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "body")), gosx.Text("Body")),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("class", "content-editor__source"), gosx.Attr("id", "body"), gosx.Attr("name", "body"), gosx.Attr("rows", "10"), gosx.Attr("data-content-editor-source", "true")), gosx.Text(body)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-editor__toolbar")),
			renderBackendBlogDetailToolbarButton("heading", "Heading"),
			renderBackendBlogDetailToolbarButton("paragraph", "Paragraph"),
			renderBackendBlogDetailToolbarButton("image", "Image"),
			renderBackendBlogDetailToolbarButton("gallery", "Gallery"),
			renderBackendBlogDetailToolbarButton("button", "Button"),
			renderBackendBlogDetailToolbarButton("product", "Product"),
			renderBackendBlogDetailToolbarButton("quote", "Quote"),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "content-block-list"), gosx.Attr("data-content-editor-list", "true"))),
	)
}

func renderBackendBlogDetailToolbarButton(kind, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "home-section-move"), gosx.Attr("data-content-add", kind)), gosx.Text(label))
}

func renderBackendBlogDetailCurrentTags(tags []BackendBlogDetailTag) gosx.Node {
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

func renderBackendBlogDetailSEO(post BackendBlogDetailValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendBlogDetailInput("metaTitle", "Meta title", post.MetaTitle, "", nil),
		renderBackendBlogDetailMediaInput("metaImageUrl", "Meta image URL", post.MetaImageURL, "metaImageAlt"),
		renderBackendBlogDetailTextarea("metaDescription", "Meta description", post.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendBlogDetailInput("metaImageAlt", "Meta image alt text", post.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendBlogDetailRelations(label string, values []BackendBlogDetailRelation) gosx.Node {
	if len(values) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(values))
	for _, value := range values {
		nodes = append(nodes, gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", value.FieldName), gosx.Attr("checked", value.Checked))),
			gosx.Text(" "+value.Title),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "radio-row")),
		gosx.El("legend", nil, gosx.Text(label)),
		gosx.Fragment(nodes...),
	)
}

func renderBackendBlogDetailChecks(post BackendBlogDetailValues) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", post.Published))), gosx.Text(" Published")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "featured"), gosx.Attr("checked", post.Featured))), gosx.Text(" Featured")),
	)
}

func renderBackendBlogDetailPrimaryButtons(props BackendBlogDetailPageProps) gosx.Node {
	post := props.Post
	primary := []gosx.Node{
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save post")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", post.PreviewHref), gosx.Attr("data-gosx-link", "true")), gosx.Text("Preview")),
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "button-row admin-form__primary-actions"),
		gosx.Attr("data-content-editor-primary-actions", "true"),
	), gosx.Fragment(primary...))
}

func renderBackendBlogDetailSecondaryButtons(props BackendBlogDetailPageProps) gosx.Node {
	post := props.Post
	secondary := []gosx.Node{}
	if post.CanArchive {
		secondary = append(secondary, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.ArchiveAction),
			gosx.Attr("data-admin-confirm", "Archive this post? It will be unpublished and hidden from public blog routes."),
		), gosx.Text("Archive")))
	}
	if post.CanRestore {
		secondary = append(secondary, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.RestoreAction),
			gosx.Attr("data-admin-confirm", "Restore this post?"),
		), gosx.Text("Restore")))
	}
	secondary = append(secondary, gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", "/admin/blog"), gosx.Attr("data-gosx-link", "true"), gosx.Attr("data-content-editor-discard", "true")), gosx.Text("Back to blog")))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "button-row admin-form__secondary-actions admin-form__danger-actions"),
		gosx.Attr("data-content-editor-secondary-actions", "true"),
		gosx.Attr("data-content-editor-danger-actions", "true"),
	), gosx.Fragment(secondary...))
}
