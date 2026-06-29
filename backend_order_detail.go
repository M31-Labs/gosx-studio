package studio

import "m31labs.dev/gosx"

type BackendOrderDetailProps struct {
	Kicker            string
	Title             string
	Summary           string
	Class             string
	SummaryPanel      BackendOrderSummary
	Actions           BackendOrderActions
	Items             []BackendOrderItem
	ShippingAddress   []BackendOrderSpecRow
	Notes             BackendOrderNotes
	PaymentReferences BackendOrderPaymentReferences
	Timeline          []BackendOrderTimelineEvent
	Audit             []BackendOrderAuditEvent
	Webhooks          []BackendOrderWebhookEvent
}

type BackendOrderSummary struct {
	CustomerName      string
	Email             string
	Phone             string
	Subtotal          string
	Shipping          string
	Total             string
	StatusLabel       string
	StatusClass       string
	FulfillmentLabel  string
	FulfillmentClass  string
	TrackingReference string
	Created           BackendOrderTime
	Updated           BackendOrderTime
}

type BackendOrderItem struct {
	Title    string
	Quantity string
	Price    string
	Href     string
}

type BackendOrderActions struct {
	OrderID            string
	CSRFToken          string
	BackHref           string
	Paths              BackendOrderActionPaths
	CanMarkPaid        bool
	CanCancel          bool
	CanRefund          bool
	RefundNote         string
	FulfillmentStatus  string
	TrackingReference  string
	FulfillmentNote    string
	FulfillmentOptions []BackendOrderFulfillmentOption
}

type BackendOrderActionPaths struct {
	MarkPaid        string
	Cancel          string
	MarkRefund      string
	SaveFulfillment string
	SaveNotes       string
}

type BackendOrderFulfillmentOption struct {
	Value    string
	Label    string
	Selected bool
}

type BackendOrderNotes struct {
	OrderID   string
	CSRFToken string
	Action    string
	Notes     string
}

type BackendOrderSpecRow struct {
	Label string
	Value string
}

type BackendOrderPaymentReferences struct {
	Provider string
	Session  string
	Payment  string
}

type BackendOrderTimelineEvent struct {
	Label   string
	Detail  string
	Created BackendOrderTime
}

type BackendOrderAuditEvent struct {
	Action  string
	Summary string
	Created BackendOrderTime
}

type BackendOrderWebhookEvent struct {
	Type        string
	Status      string
	StatusClass string
	Summary     string
	Created     BackendOrderTime
}

type BackendOrderTime struct {
	Label   string
	Machine string
}

func RenderBackendOrderDetail(props BackendOrderDetailProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-order-detail-renderer", "gosx-studio"),
	),
		RenderBackendOrderDetailContent(props),
	)
}

func RenderBackendOrderDetailContent(props BackendOrderDetailProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendOrderDetailHeading(props),
		RenderBackendOrderDetailSummary(props.SummaryPanel),
		RenderBackendOrderDetailItems(props.Items),
		RenderBackendOrderDetailShippingAddress(props.ShippingAddress),
		RenderBackendOrderDetailPaymentReferences(props.PaymentReferences),
		RenderBackendOrderDetailTimeline(props.Timeline),
		RenderBackendOrderDetailAudit(props.Audit),
		RenderBackendOrderDetailWebhooks(props.Webhooks),
	)
}

func RenderBackendOrderDetailPage(props BackendOrderDetailProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-order-detail-renderer", "gosx-studio"),
	),
		RenderBackendOrderDetailPageContent(props),
	)
}

func RenderBackendOrderDetailPageContent(props BackendOrderDetailProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendOrderDetailHeading(props),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-grid")),
			RenderBackendOrderDetailSummary(props.SummaryPanel),
			RenderBackendOrderDetailActions(props.Actions),
		),
		RenderBackendOrderDetailItems(props.Items),
		RenderBackendOrderDetailShippingAddress(props.ShippingAddress),
		RenderBackendOrderDetailNotes(props.Notes),
		RenderBackendOrderDetailPaymentReferences(props.PaymentReferences),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-grid")),
			RenderBackendOrderDetailTimeline(props.Timeline),
			RenderBackendOrderDetailAudit(props.Audit),
			RenderBackendOrderDetailWebhooks(props.Webhooks),
		),
	)
}

func RenderBackendOrderDetailHeading(props BackendOrderDetailProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
		gosx.El("h1", nil, gosx.Text(props.Title)),
		gosx.El("p", nil, gosx.Text(props.Summary)),
	)
}

func RenderBackendOrderDetailSummary(summary BackendOrderSummary) gosx.Node {
	statusClass := summary.StatusClass
	if statusClass == "" {
		statusClass = "status"
	}
	fulfillmentClass := summary.FulfillmentClass
	if fulfillmentClass == "" {
		fulfillmentClass = "status"
	}
	rows := []gosx.Node{
		renderBackendOrderSpecText("Customer", summary.CustomerName),
		renderBackendOrderSpecLink("Email", "mailto:"+summary.Email, summary.Email),
	}
	if summary.Phone != "" {
		rows = append(rows, renderBackendOrderSpecText("Phone", summary.Phone))
	}
	rows = append(rows,
		renderBackendOrderSpecText("Subtotal", summary.Subtotal),
		renderBackendOrderSpecText("Shipping", summary.Shipping),
		renderBackendOrderSpecStrong("Total", summary.Total),
		renderBackendOrderSpecNode("Fulfillment", gosx.El("span", gosx.Attrs(gosx.Attr("class", fulfillmentClass)), gosx.Text(summary.FulfillmentLabel))),
	)
	if summary.TrackingReference != "" {
		rows = append(rows, renderBackendOrderSpecText("Tracking", summary.TrackingReference))
	}
	rows = append(rows,
		renderBackendOrderSpecTime("Created", summary.Created),
		renderBackendOrderSpecTime("Updated", summary.Updated),
	)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Summary")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", statusClass)), gosx.Text(summary.StatusLabel)),
		),
		gosx.El("dl", gosx.Attrs(gosx.Attr("class", "spec-list")), gosx.Fragment(rows...)),
	)
}

func RenderBackendOrderDetailActions(actions BackendOrderActions) gosx.Node {
	backHref := actions.BackHref
	if backHref == "" {
		backHref = "/admin/orders"
	}
	actionButtons := []gosx.Node{}
	if actions.CanMarkPaid {
		actionButtons = append(actionButtons, renderBackendOrderInlineActionForm(actions.Paths.MarkPaid, actions.CSRFToken, actions.OrderID, "button button--primary", "Mark this order paid and mark the item sold?", "Mark paid"))
	}
	if actions.CanCancel {
		actionButtons = append(actionButtons, renderBackendOrderInlineActionForm(actions.Paths.Cancel, actions.CSRFToken, actions.OrderID, "button button--secondary", "Cancel this order and release reserved inventory?", "Cancel and release"))
	}
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", backHref), gosx.Attr("data-gosx-link", "true")), gosx.Text("Back to orders")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(actionButtons...)),
	}
	if actions.CanRefund {
		children = append(children, renderBackendOrderRefundForm(actions))
	}
	children = append(children, renderBackendOrderFulfillmentForm(actions))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
}

func RenderBackendOrderDetailNotes(notes BackendOrderNotes) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Notes")),
		),
		gosx.El("form", gosx.Attrs(
			gosx.Attr("class", "admin-form admin-form--single"),
			gosx.Attr("method", "post"),
			gosx.Attr("action", notes.Action),
		),
			renderBackendOrderHiddenInputs(notes.CSRFToken, notes.OrderID),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", "notes")), gosx.Text("Internal notes")),
				gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "notes"), gosx.Attr("name", "notes"), gosx.Attr("rows", "5")), gosx.Text(notes.Notes)),
			),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save notes")),
		),
	)
}

func RenderBackendOrderDetailItems(items []BackendOrderItem) gosx.Node {
	rows := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		rows = append(rows, gosx.El("tr", nil,
			gosx.El("td", nil, gosx.El("strong", nil, gosx.Text(item.Title))),
			gosx.El("td", nil, gosx.Text(item.Quantity)),
			gosx.El("td", nil, gosx.Text(item.Price)),
			gosx.El("td", nil, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", item.Href),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text("Product"))),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Items")),
		),
		gosx.El("table", gosx.Attrs(gosx.Attr("class", "data-table")),
			gosx.El("thead", nil, gosx.El("tr", nil,
				gosx.El("th", nil, gosx.Text("Item")),
				gosx.El("th", nil, gosx.Text("Quantity")),
				gosx.El("th", nil, gosx.Text("Price")),
				gosx.El("th", nil),
			)),
			gosx.El("tbody", nil, gosx.Fragment(rows...)),
		),
	)
}

func RenderBackendOrderDetailShippingAddress(rows []BackendOrderSpecRow) gosx.Node {
	if len(rows) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(rows))
	for _, row := range rows {
		nodes = append(nodes, renderBackendOrderSpecText(row.Label, row.Value))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Shipping address")),
		),
		gosx.El("dl", gosx.Attrs(gosx.Attr("class", "spec-list")), gosx.Fragment(nodes...)),
	)
}

func RenderBackendOrderDetailPaymentReferences(refs BackendOrderPaymentReferences) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Payment references")),
		),
		gosx.El("dl", gosx.Attrs(gosx.Attr("class", "spec-list")),
			renderBackendOrderSpecText("Provider", refs.Provider),
			renderBackendOrderSpecText("Session", refs.Session),
			renderBackendOrderSpecText("Payment", refs.Payment),
		),
	)
}

func RenderBackendOrderDetailTimeline(events []BackendOrderTimelineEvent) gosx.Node {
	items := make([]gosx.Node, 0, len(events))
	for _, event := range events {
		nodes := []gosx.Node{
			gosx.El("strong", nil, gosx.Text(event.Label)),
			renderBackendOrderTime(event.Created),
		}
		if event.Detail != "" {
			nodes = append(nodes, gosx.El("p", nil, gosx.Text(event.Detail)))
		}
		items = append(items, gosx.El("li", nil, gosx.Fragment(nodes...)))
	}
	return renderBackendOrderHistoryPanel("Timeline", items, "No timeline events yet.")
}

func RenderBackendOrderDetailAudit(events []BackendOrderAuditEvent) gosx.Node {
	items := make([]gosx.Node, 0, len(events))
	for _, event := range events {
		nodes := []gosx.Node{
			gosx.El("strong", nil, gosx.Text(event.Action)),
			renderBackendOrderTime(event.Created),
		}
		if event.Summary != "" {
			nodes = append(nodes, gosx.El("p", nil, gosx.Text(event.Summary)))
		}
		items = append(items, gosx.El("li", nil, gosx.Fragment(nodes...)))
	}
	return renderBackendOrderHistoryPanel("Audit", items, "No audit events yet.")
}

func RenderBackendOrderDetailWebhooks(events []BackendOrderWebhookEvent) gosx.Node {
	items := make([]gosx.Node, 0, len(events))
	for _, event := range events {
		statusClass := event.StatusClass
		if statusClass == "" {
			statusClass = "status"
		}
		nodes := []gosx.Node{
			gosx.El("strong", nil, gosx.Text(event.Type)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", statusClass)), gosx.Text(event.Status)),
			renderBackendOrderTime(event.Created),
		}
		if event.Summary != "" {
			nodes = append(nodes, gosx.El("p", nil, gosx.Text(event.Summary)))
		}
		items = append(items, gosx.El("li", nil, gosx.Fragment(nodes...)))
	}
	return renderBackendOrderHistoryPanel("Webhook events", items, "No Stripe webhook events for this order yet.")
}

func renderBackendOrderInlineActionForm(action, csrfToken, orderID, buttonClass, confirm, label string) gosx.Node {
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "inline-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", action),
	),
		renderBackendOrderHiddenInputs(csrfToken, orderID),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", buttonClass),
			gosx.Attr("type", "submit"),
			gosx.Attr("data-admin-confirm", confirm),
		), gosx.Text(label)),
	)
}

func renderBackendOrderRefundForm(actions BackendOrderActions) gosx.Node {
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "admin-form admin-form--single"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", actions.Paths.MarkRefund),
	),
		renderBackendOrderHiddenInputs(actions.CSRFToken, actions.OrderID),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "refundNote")), gosx.Text("Refund note")),
			gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "refundNote"), gosx.Attr("name", "refundNote"), gosx.Attr("rows", "3")), gosx.Text(actions.RefundNote)),
		),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("data-admin-confirm", "Record this order as refunded?"),
		), gosx.Text("Record refund")),
	)
}

func renderBackendOrderFulfillmentForm(actions BackendOrderActions) gosx.Node {
	options := make([]gosx.Node, 0, len(actions.FulfillmentOptions))
	for _, option := range actions.FulfillmentOptions {
		selected := option.Selected
		if !selected && actions.FulfillmentStatus != "" {
			selected = option.Value == actions.FulfillmentStatus
		}
		options = append(options, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", option.Value),
			gosx.Attr("selected", selected),
		), gosx.Text(option.Label)))
	}
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "admin-form admin-form--single"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", actions.Paths.SaveFulfillment),
	),
		renderBackendOrderHiddenInputs(actions.CSRFToken, actions.OrderID),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "fulfillmentStatus")), gosx.Text("Fulfillment")),
			gosx.El("select", gosx.Attrs(gosx.Attr("id", "fulfillmentStatus"), gosx.Attr("name", "fulfillmentStatus")), gosx.Fragment(options...)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "trackingReference")), gosx.Text("Tracking/reference")),
			gosx.El("input", gosx.Attrs(gosx.Attr("id", "trackingReference"), gosx.Attr("name", "trackingReference"), gosx.Attr("value", actions.TrackingReference))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "fulfillmentNote")), gosx.Text("Fulfillment note")),
			gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "fulfillmentNote"), gosx.Attr("name", "fulfillmentNote"), gosx.Attr("rows", "3")), gosx.Text(actions.FulfillmentNote)),
		),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("type", "submit")), gosx.Text("Save fulfillment")),
	)
}

func renderBackendOrderHiddenInputs(csrfToken, orderID string) gosx.Node {
	return gosx.Fragment(
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", csrfToken))),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "id"), gosx.Attr("value", orderID))),
	)
}

func renderBackendOrderHistoryPanel(title string, items []gosx.Node, empty string) gosx.Node {
	content := gosx.Node(gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list field-list--stacked")), gosx.Fragment(items...)))
	if len(items) == 0 {
		content = gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(empty))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text(title)),
		),
		content,
	)
}

func renderBackendOrderSpecText(label, value string) gosx.Node {
	return renderBackendOrderSpecNode(label, gosx.Text(value))
}

func renderBackendOrderSpecStrong(label, value string) gosx.Node {
	return renderBackendOrderSpecNode(label, gosx.El("strong", nil, gosx.Text(value)))
}

func renderBackendOrderSpecLink(label, href, value string) gosx.Node {
	return renderBackendOrderSpecNode(label, gosx.El("a", gosx.Attrs(gosx.Attr("href", href)), gosx.Text(value)))
}

func renderBackendOrderSpecTime(label string, value BackendOrderTime) gosx.Node {
	return renderBackendOrderSpecNode(label, renderBackendOrderTime(value))
}

func renderBackendOrderSpecNode(label string, value gosx.Node) gosx.Node {
	return gosx.El("div", nil,
		gosx.El("dt", nil, gosx.Text(label)),
		gosx.El("dd", nil, value),
	)
}

func renderBackendOrderTime(value BackendOrderTime) gosx.Node {
	return gosx.El("time", gosx.Attrs(
		gosx.Attr("datetime", value.Machine),
		gosx.Attr("data-viewer-time", "datetime"),
	), gosx.Text(value.Label))
}
