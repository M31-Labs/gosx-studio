package backoffice

import "m31labs.dev/gosx"

type BackendProductIndexPageProps struct {
	Class                string
	ResourceIndexContent gosx.Node
	Media                []BackendProductIndexMediaAsset
	Categories           []BackendProductIndexCategory
	CreateAction         string
	CSRFToken            string
	CreateStatus         BackendProductIndexActionStatus
	Product              BackendProductIndexValues
}

type BackendProductIndexActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendProductIndexMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendProductIndexCategory struct {
	FieldName string
	Title     string
}

type BackendProductIndexValues struct {
	Title           string
	Price           string
	ImagesText      string
	Description     string
	MetaTitle       string
	MetaImageURL    string
	MetaDescription string
	MetaImageAlt    string
}

func RenderBackendProductIndexPage(props BackendProductIndexPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-resource-index-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-product-index-renderer", "gosx-studio"),
	),
		RenderBackendProductIndexContent(props),
	)
}

func RenderBackendProductIndexContent(props BackendProductIndexPageProps) gosx.Node {
	return gosx.Fragment(
		props.ResourceIndexContent,
		RenderBackendProductCreatePanel(props),
		RenderBackendProductIndexScripts(),
	)
}

func RenderBackendProductCreatePanel(props BackendProductIndexPageProps) gosx.Node {
	product := props.Product
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Add product")),
		),
		renderBackendProductIndexStatus(props.CreateStatus),
		RenderBackendProductIndexMediaDatalist(props.Media),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", props.CreateAction),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			renderBackendProductIndexInput("title", "Title", product.Title, "", renderBackendProductIndexFieldError(props.CreateStatus, "title")),
			renderBackendProductIndexInput("price", "Price in cents", product.Price, "", renderBackendProductIndexFieldError(props.CreateStatus, "price"), gosx.Attr("type", "number"), gosx.Attr("min", "0")),
			renderBackendProductIndexTextarea("imagesText", "Images", product.ImagesText, "4", "field-row field-row--wide", gosx.Attr("data-media-lines-list", "product-media-urls")),
			renderBackendProductIndexTextarea("description", "Description", product.Description, "4", "field-row field-row--wide"),
			renderBackendProductIndexSEO(product),
			renderBackendProductIndexCategories(props.Categories),
			renderBackendProductIndexChecks(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Create product")),
		),
	)
}

func RenderBackendProductIndexMediaDatalist(media []BackendProductIndexMediaAsset) gosx.Node {
	nodes := make([]gosx.Node, 0, len(media))
	for _, asset := range media {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", asset.URL),
			gosx.Attr("label", asset.Filename),
			gosx.Attr("data-media-alt", asset.Alt),
		), gosx.Text(asset.Filename)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "product-media-urls")), gosx.Fragment(nodes...))
}

func RenderBackendProductIndexScripts() gosx.Node {
	return renderBackendManagedStudioScript(backendMediaRuntimePath)
}

func renderBackendProductIndexStatus(status BackendProductIndexActionStatus) gosx.Node {
	if !status.Submitted {
		return gosx.Fragment()
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message))
}

func renderBackendProductIndexInput(id, label, value, className string, errorNode *gosx.Node, extraAttrs ...any) gosx.Node {
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

func renderBackendProductIndexTextarea(id, label, value, rows, className string, extraAttrs ...any) gosx.Node {
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

func renderBackendProductIndexFieldError(status BackendProductIndexActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendProductIndexSEO(product BackendProductIndexValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendProductIndexInput("metaTitle", "Meta title", product.MetaTitle, "", nil),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "metaImageUrl")), gosx.Text("Meta image URL")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "metaImageUrl"),
				gosx.Attr("name", "metaImageUrl"),
				gosx.Attr("list", "product-media-urls"),
				gosx.Attr("value", product.MetaImageURL),
				gosx.Attr("data-media-alt-target", "metaImageAlt"),
			)),
		),
		renderBackendProductIndexTextarea("metaDescription", "Meta description", product.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendProductIndexInput("metaImageAlt", "Meta image alt text", product.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendProductIndexCategories(categories []BackendProductIndexCategory) gosx.Node {
	if len(categories) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(categories))
	for _, category := range categories {
		nodes = append(nodes, gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", category.FieldName))),
			gosx.Text(" "+category.Title),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "radio-row")),
		gosx.El("legend", nil, gosx.Text("Categories")),
		gosx.Fragment(nodes...),
	)
}

func renderBackendProductIndexChecks() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", "checked"))), gosx.Text(" Published")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "featured"))), gosx.Text(" Featured")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "requiresShipping"), gosx.Attr("checked", "checked"))), gosx.Text(" Requires shipping")),
	)
}
