package studio

import "m31labs.dev/gosx"

type BackendProductDetailPageProps struct {
	Class         string
	Heading       gosx.Node
	Preview       gosx.Node
	Media         []BackendProductDetailMediaAsset
	SaveAction    string
	ArchiveAction string
	RestoreAction string
	CSRFToken     string
	SaveStatus    BackendProductDetailActionStatus
	ArchiveStatus BackendProductDetailActionStatus
	RestoreStatus BackendProductDetailActionStatus
	Product       BackendProductDetailValues
	Categories    []BackendProductDetailCategory
}

type BackendProductDetailActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendProductDetailMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendProductDetailValues struct {
	ID               string
	Title            string
	Slug             string
	PriceCents       string
	Currency         string
	Description      string
	ImagesText       string
	MetaTitle        string
	MetaImageURL     string
	MetaDescription  string
	MetaImageAlt     string
	Height           string
	Width            string
	Depth            string
	DimensionUnits   string
	WeightValue      string
	WeightUnits      string
	IsAvailable      bool
	IsReserved       bool
	IsSold           bool
	Published        bool
	Featured         bool
	RequiresShipping bool
	ShippingNote     string
	CanArchive       bool
	CanRestore       bool
	PreviewHref      string
	Gallery          []BackendProductDetailImage
}

type BackendProductDetailImage struct {
	URL string
	Alt string
}

type BackendProductDetailCategory struct {
	FieldName string
	Title     string
	Checked   bool
}

func RenderBackendProductDetailPage(props BackendProductDetailPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-detail-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-product-detail-renderer", "gosx-studio"),
	),
		RenderBackendProductDetailContent(props),
	)
}

func RenderBackendProductDetailContent(props BackendProductDetailPageProps) gosx.Node {
	return gosx.Fragment(
		props.Heading,
		RenderBackendProductDetailMediaDatalist(props.Media),
		RenderBackendProductDetailForm(props),
		RenderBackendProductDetailScripts(),
	)
}

func RenderBackendProductDetailMediaDatalist(media []BackendProductDetailMediaAsset) gosx.Node {
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

func RenderBackendProductDetailForm(props BackendProductDetailPageProps) gosx.Node {
	product := props.Product
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", props.SaveAction),
	),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", product.ID))),
		gosx.Fragment(renderBackendProductDetailStatuses(props)...),
		props.Preview,
		renderBackendProductDetailGallery(product.Gallery),
		renderBackendProductDetailInput("title", "Title", product.Title, "", renderBackendProductDetailFieldError(props.SaveStatus, "title")),
		renderBackendProductDetailInput("slug", "Slug", product.Slug, "", nil),
		renderBackendProductDetailInput("price", "Price in cents", product.PriceCents, "", renderBackendProductDetailFieldError(props.SaveStatus, "price"), gosx.Attr("type", "number"), gosx.Attr("min", "0")),
		renderBackendProductDetailInput("currency", "Currency", product.Currency, "", nil),
		renderBackendProductDetailTextarea("description", "Description", product.Description, "5", "field-row field-row--wide"),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "imagesText")), gosx.Text("Images")),
			gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "imagesText"), gosx.Attr("name", "imagesText"), gosx.Attr("rows", "5"), gosx.Attr("data-media-lines-list", "product-media-urls")), gosx.Text(product.ImagesText)),
		),
		renderBackendProductDetailSEO(product),
		renderBackendProductDetailInput("height", "Height", product.Height, "", nil),
		renderBackendProductDetailInput("width", "Width", product.Width, "", nil),
		renderBackendProductDetailInput("depth", "Depth", product.Depth, "", nil),
		renderBackendProductDetailInput("dimensionUnits", "Units", product.DimensionUnits, "", nil),
		renderBackendProductDetailInput("weightValue", "Weight", product.WeightValue, "", nil),
		renderBackendProductDetailInput("weightUnits", "Weight units", product.WeightUnits, "", nil),
		renderBackendProductDetailAvailability(product),
		renderBackendProductDetailCategories(props.Categories),
		renderBackendProductDetailChecks(product),
		renderBackendProductDetailTextarea("shippingNote", "Shipping note", product.ShippingNote, "3", "field-row field-row--wide"),
		renderBackendProductDetailButtons(props),
	)
}

func RenderBackendProductDetailScripts() gosx.Node {
	return gosx.El("script", gosx.Attrs(gosx.Attr("src", "/media-picker.js"), gosx.Attr("defer", true)))
}

func renderBackendProductDetailStatuses(props BackendProductDetailPageProps) []gosx.Node {
	nodes := []gosx.Node{}
	for _, status := range []BackendProductDetailActionStatus{props.SaveStatus, props.ArchiveStatus, props.RestoreStatus} {
		if status.Submitted {
			nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message)))
		}
	}
	return nodes
}

func renderBackendProductDetailGallery(images []BackendProductDetailImage) gosx.Node {
	if len(images) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(images))
	for _, image := range images {
		nodes = append(nodes, gosx.El("img", gosx.Attrs(gosx.Attr("src", image.URL), gosx.Attr("alt", image.Alt))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "media-strip media-strip--compact")), gosx.Fragment(nodes...))
}

func renderBackendProductDetailInput(id, label, value, className string, errorNode *gosx.Node, extraAttrs ...any) gosx.Node {
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

func renderBackendProductDetailTextarea(id, label, value, rows, className string) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)), gosx.Text(value)),
	)
}

func renderBackendProductDetailFieldError(status BackendProductDetailActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendProductDetailSEO(product BackendProductDetailValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendProductDetailInput("metaTitle", "Meta title", product.MetaTitle, "", nil),
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
		renderBackendProductDetailTextarea("metaDescription", "Meta description", product.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendProductDetailInput("metaImageAlt", "Meta image alt text", product.MetaImageAlt, "field-row field-row--wide", nil),
	)
}

func renderBackendProductDetailAvailability(product BackendProductDetailValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "radio-row")),
		gosx.El("legend", nil, gosx.Text("Availability")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "radio"), gosx.Attr("name", "availability"), gosx.Attr("value", "available"), gosx.Attr("checked", product.IsAvailable))), gosx.Text(" Available")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "radio"), gosx.Attr("name", "availability"), gosx.Attr("value", "reserved"), gosx.Attr("checked", product.IsReserved))), gosx.Text(" Reserved")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "radio"), gosx.Attr("name", "availability"), gosx.Attr("value", "sold"), gosx.Attr("checked", product.IsSold))), gosx.Text(" Sold")),
	)
}

func renderBackendProductDetailCategories(categories []BackendProductDetailCategory) gosx.Node {
	if len(categories) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(categories))
	for _, category := range categories {
		nodes = append(nodes, gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", category.FieldName), gosx.Attr("checked", category.Checked))),
			gosx.Text(" "+category.Title),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "radio-row")),
		gosx.El("legend", nil, gosx.Text("Categories")),
		gosx.Fragment(nodes...),
	)
}

func renderBackendProductDetailChecks(product BackendProductDetailValues) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "published"), gosx.Attr("checked", product.Published))), gosx.Text(" Published")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "featured"), gosx.Attr("checked", product.Featured))), gosx.Text(" Featured")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "requiresShipping"), gosx.Attr("checked", product.RequiresShipping))), gosx.Text(" Requires shipping")),
	)
}

func renderBackendProductDetailButtons(props BackendProductDetailPageProps) gosx.Node {
	product := props.Product
	nodes := []gosx.Node{
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save product")),
	}
	if product.CanArchive {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.ArchiveAction),
			gosx.Attr("data-admin-confirm", "Archive this product? It will be hidden from the storefront."),
		), gosx.Text("Archive")))
	}
	if product.CanRestore {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.RestoreAction),
			gosx.Attr("data-admin-confirm", "Restore this product to the admin catalog?"),
		), gosx.Text("Restore")))
	}
	nodes = append(nodes,
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", product.PreviewHref), gosx.Attr("data-gosx-link", "true")), gosx.Text("Preview")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", "/admin/products"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Back to products")),
	)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(nodes...))
}
