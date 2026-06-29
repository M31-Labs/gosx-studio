package studio

import "m31labs.dev/gosx"

type BackendResourceIndexProps struct {
	Kicker  string
	Title   string
	Summary string
	Empty   string
	Class   string
	Table   *BackendResourceTable
	Cards   []BackendResourceCard
}

type BackendResourceTable struct {
	Headers []string
	Rows    []BackendResourceTableRow
}

type BackendResourceTableRow struct {
	Cells  []BackendResourceTableCell
	Action BackendResourceLink
}

type BackendResourceTableCell struct {
	Node        *gosx.Node
	Text        string
	Primary     string
	Secondary   string
	StatusClass string
	Time        BackendResourceTime
	Link        BackendResourceLink
}

type BackendResourceCard struct {
	Title       string
	Status      string
	StatusClass string
	Body        string
	Email       BackendResourceLink
	Action      BackendResourceLink
	Time        BackendResourceTime
}

type BackendResourceLink struct {
	Href      string
	Label     string
	GOSXLink  bool
	ClassName string
}

type BackendResourceTime struct {
	Label   string
	Machine string
}

func RenderBackendResourceIndex(props BackendResourceIndexProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-resource-index-renderer", "gosx-studio"),
	),
		RenderBackendResourceIndexContent(props),
	)
}

func RenderBackendResourceIndexContent(props BackendResourceIndexProps) gosx.Node {
	return gosx.Fragment(
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
			gosx.El("h1", nil, gosx.Text(props.Title)),
			gosx.El("p", nil, gosx.Text(props.Summary)),
		),
		renderBackendResourceIndexBody(props),
	)
}

func renderBackendResourceIndexBody(props BackendResourceIndexProps) gosx.Node {
	if props.Table != nil {
		return renderBackendResourceTable(*props.Table, props.Empty)
	}
	return renderBackendResourceCards(props.Cards, props.Empty)
}

func renderBackendResourceTable(table BackendResourceTable, empty string) gosx.Node {
	children := []gosx.Node{gosx.El("table", gosx.Attrs(gosx.Attr("class", "data-table")),
		gosx.El("thead", nil,
			gosx.El("tr", nil, gosx.Fragment(renderBackendResourceTableHeaders(table.Headers)...)),
		),
		gosx.El("tbody", nil, gosx.Fragment(renderBackendResourceTableRows(table.Rows)...)),
	)}
	if len(table.Rows) == 0 && empty != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(empty)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
}

func renderBackendResourceTableHeaders(headers []string) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(headers))
	for _, header := range headers {
		nodes = append(nodes, gosx.El("th", nil, gosx.Text(header)))
	}
	return nodes
}

func renderBackendResourceTableRows(rows []BackendResourceTableRow) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(rows))
	for _, row := range rows {
		cells := make([]gosx.Node, 0, len(row.Cells)+1)
		for _, cell := range row.Cells {
			cells = append(cells, gosx.El("td", nil, renderBackendResourceTableCell(cell)))
		}
		cells = append(cells, gosx.El("td", nil, renderBackendResourceLink(row.Action)))
		nodes = append(nodes, gosx.El("tr", nil, gosx.Fragment(cells...)))
	}
	return nodes
}

func renderBackendResourceTableCell(cell BackendResourceTableCell) gosx.Node {
	if cell.Node != nil {
		return *cell.Node
	}
	if cell.Link.Href != "" || cell.Link.Label != "" {
		return renderBackendResourceLink(cell.Link)
	}
	if cell.Primary != "" || cell.Secondary != "" {
		nodes := []gosx.Node{}
		if cell.Primary != "" {
			nodes = append(nodes, gosx.El("strong", nil, gosx.Text(cell.Primary)))
		}
		if cell.Secondary != "" {
			nodes = append(nodes, gosx.El("span", nil, gosx.Text(cell.Secondary)))
		}
		return gosx.Fragment(nodes...)
	}
	if cell.StatusClass != "" {
		return gosx.El("span", gosx.Attrs(gosx.Attr("class", cell.StatusClass)), gosx.Text(cell.Text))
	}
	if cell.Time.Machine != "" || cell.Time.Label != "" {
		return renderBackendResourceTime(cell.Time, "")
	}
	return gosx.Text(cell.Text)
}

func renderBackendResourceCards(cards []BackendResourceCard, empty string) gosx.Node {
	children := make([]gosx.Node, 0, len(cards)+1)
	for _, card := range cards {
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("class", "panel")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
				gosx.El("h2", nil, gosx.Text(card.Title)),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", card.StatusClass)), gosx.Text(card.Status)),
			),
			gosx.El("p", nil, gosx.Text(card.Body)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")),
				renderBackendResourceLink(card.Email),
				renderBackendResourceLink(card.Action),
				renderBackendResourceTime(card.Time, "field-note"),
			),
		))
	}
	if len(cards) == 0 && empty != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(empty)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "message-list message-list--full")), gosx.Fragment(children...))
}

func renderBackendResourceLink(link BackendResourceLink) gosx.Node {
	attrs := []any{gosx.Attr("href", link.Href)}
	if link.ClassName != "" {
		attrs = append(attrs, gosx.Attr("class", link.ClassName))
	}
	if link.GOSXLink {
		attrs = append(attrs, gosx.Attr("data-gosx-link", "true"))
	}
	return gosx.El("a", gosx.Attrs(attrs...), gosx.Text(link.Label))
}

func renderBackendResourceTime(value BackendResourceTime, className string) gosx.Node {
	attrs := []any{
		gosx.Attr("datetime", value.Machine),
		gosx.Attr("data-viewer-time", "datetime"),
	}
	if className != "" {
		attrs = append([]any{gosx.Attr("class", className)}, attrs...)
	}
	return gosx.El("time", gosx.Attrs(attrs...), gosx.Text(value.Label))
}
