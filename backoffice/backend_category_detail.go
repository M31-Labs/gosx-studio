package backoffice

import "m31labs.dev/gosx"

type BackendCategoryDetailPageProps struct {
	Class         string
	Heading       gosx.Node
	Preview       gosx.Node
	SaveAction    string
	ArchiveAction string
	RestoreAction string
	CSRFToken     string
	SaveStatus    BackendCategoryDetailActionStatus
	ArchiveStatus BackendCategoryDetailActionStatus
	RestoreStatus BackendCategoryDetailActionStatus
	Category      BackendCategoryDetailValues
}

type BackendCategoryDetailActionStatus struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendCategoryDetailValues struct {
	ID         string
	Title      string
	Slug       string
	CanArchive bool
	CanRestore bool
}

func RenderBackendCategoryDetailPage(props BackendCategoryDetailPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-detail-renderer", "gosx-studio"),
		gosx.Attr("data-gosx-studio-backend-category-detail-renderer", "gosx-studio"),
	),
		RenderBackendCategoryDetailContent(props),
	)
}

func RenderBackendCategoryDetailContent(props BackendCategoryDetailPageProps) gosx.Node {
	return gosx.Fragment(
		props.Heading,
		RenderBackendCategoryDetailForm(props),
	)
}

func RenderBackendCategoryDetailForm(props BackendCategoryDetailPageProps) gosx.Node {
	category := props.Category
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", props.SaveAction),
	),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", category.ID))),
		gosx.Fragment(renderBackendCategoryDetailStatuses(props)...),
		props.Preview,
		renderBackendCategoryDetailInput("title", "Title", category.Title, renderBackendCategoryDetailFieldError(props.SaveStatus, "title")),
		renderBackendCategoryDetailInput("slug", "Slug", category.Slug, nil),
		renderBackendCategoryDetailButtons(props),
	)
}

func renderBackendCategoryDetailStatuses(props BackendCategoryDetailPageProps) []gosx.Node {
	nodes := []gosx.Node{}
	for _, status := range []BackendCategoryDetailActionStatus{props.SaveStatus, props.ArchiveStatus, props.RestoreStatus} {
		if status.Submitted {
			nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(status.Message)))
		}
	}
	return nodes
}

func renderBackendCategoryDetailInput(id, label, value string, errorNode *gosx.Node) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("value", value))),
	}
	if errorNode != nil {
		children = append(children, *errorNode)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")), gosx.Fragment(children...))
}

func renderBackendCategoryDetailFieldError(status BackendCategoryDetailActionStatus, name string) *gosx.Node {
	message := ""
	if status.FieldErrors != nil {
		message = status.FieldErrors[name]
	}
	node := gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(message))
	return &node
}

func renderBackendCategoryDetailButtons(props BackendCategoryDetailPageProps) gosx.Node {
	category := props.Category
	nodes := []gosx.Node{
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save category")),
	}
	if category.CanArchive {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.ArchiveAction),
			gosx.Attr("data-admin-confirm", "Archive this category? Category links will be removed from public browsing."),
		), gosx.Text("Archive")))
	}
	if category.CanRestore {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", props.RestoreAction),
			gosx.Attr("data-admin-confirm", "Restore this category?"),
		), gosx.Text("Restore")))
	}
	nodes = append(nodes, gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", "/admin/categories"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Back to categories")))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(nodes...))
}
