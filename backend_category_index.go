package studio

import "m31labs.dev/gosx"

type BackendCategoryIndexPageProps struct {
	Class                string
	ResourceIndexContent gosx.Node
	CreateAction         string
	CSRFToken            string
	CreateStatus         BackendCategoryIndexActionStatus
	Category             BackendCategoryIndexValues
}

type BackendCategoryIndexActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendCategoryIndexValues struct {
	Title string
	Slug  string
}

func RenderBackendCategoryIndexPage(props BackendCategoryIndexPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-resource-index-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-category-index-renderer", "gosx-studio"),
	),
		RenderBackendCategoryIndexContent(props),
	)
}

func RenderBackendCategoryIndexContent(props BackendCategoryIndexPageProps) gosx.Node {
	return gosx.Fragment(
		props.ResourceIndexContent,
		RenderBackendCategoryCreatePanel(props),
	)
}

func RenderBackendCategoryCreatePanel(props BackendCategoryIndexPageProps) gosx.Node {
	category := props.Category
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Add category")),
		),
		renderBackendCategoryIndexStatus(props.CreateStatus),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", props.CreateAction),
		),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
			renderBackendCategoryIndexInput("title", "Title", category.Title, renderBackendCategoryIndexFieldError(props.CreateStatus, "title")),
			renderBackendCategoryIndexInput("slug", "Slug", category.Slug, nil),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Create category")),
		),
	)
}

func renderBackendCategoryIndexStatus(status BackendCategoryIndexActionStatus) gosx.Node {
	if !status.Submitted {
		return gosx.Fragment()
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message))
}

func renderBackendCategoryIndexInput(id, label, value string, errorNode *gosx.Node) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("value", value))),
	}
	if errorNode != nil {
		children = append(children, *errorNode)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")), gosx.Fragment(children...))
}

func renderBackendCategoryIndexFieldError(status BackendCategoryIndexActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}
