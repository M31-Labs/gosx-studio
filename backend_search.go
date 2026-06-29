package studio

import "m31labs.dev/gosx"

type BackendSearchProps struct {
	Kicker   string
	Title    string
	Summary  string
	Query    string
	Empty    string
	Class    string
	Searched bool
	Results  []BackendSearchResult
}

type BackendSearchResult struct {
	Href     string
	Kind     string
	Status   string
	Title    string
	Detail   string
	GOSXLink bool
}

func RenderBackendSearch(props BackendSearchProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-search-renderer", "gosx-studio"),
	),
		RenderBackendSearchContent(props),
	)
}

func RenderBackendSearchContent(props BackendSearchProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendSearchHeading(props),
		RenderBackendSearchResults(props),
	)
}

func RenderBackendSearchHeading(props BackendSearchProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
		gosx.El("h1", nil, gosx.Text(props.Title)),
		gosx.El("p", nil, gosx.Text(props.Summary)),
	)
}

func RenderBackendSearchResults(props BackendSearchProps) gosx.Node {
	if !props.Searched {
		return gosx.Fragment()
	}
	empty := props.Empty
	if empty == "" {
		empty = "No matching records."
	}
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Results")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(props.Query)),
		),
	}
	if len(props.Results) > 0 {
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", "resource-grid")),
			gosx.Fragment(renderBackendSearchResultCards(props.Results)...),
		))
	} else {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(empty)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
}

func renderBackendSearchResultCards(results []BackendSearchResult) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(results))
	for _, result := range results {
		attrs := []any{
			gosx.Attr("class", "resource-card"),
			gosx.Attr("href", result.Href),
		}
		if result.GOSXLink {
			attrs = append(attrs, gosx.Attr("data-gosx-link", "true"))
		}
		status := result.Status
		if status == "" {
			status = result.Kind
		}
		nodes = append(nodes, gosx.El("a", gosx.Attrs(attrs...),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(status)),
			gosx.El("strong", nil, gosx.Text(result.Title)),
			gosx.El("span", nil, gosx.Text(result.Detail)),
		))
	}
	return nodes
}
