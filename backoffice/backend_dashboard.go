package backoffice

import "m31labs.dev/gosx"

type BackendDashboardProps struct {
	Stats         []BackendDashboardStat
	Actions       []BackendDashboardCard
	Payment       BackendDashboardPayment
	Alerts        []BackendDashboardAlert
	Onboarding    []BackendDashboardChecklistItem
	Resources     []BackendDashboardResource
	Audit         []BackendDashboardTimelineEvent
	Webhooks      []BackendDashboardTimelineEvent
	Auth          BackendDashboardAuth
	Orders        []BackendDashboardOrder
	Contacts      []BackendDashboardContact
	ShowWorkbench bool
}

type BackendDashboardStat struct {
	Label string
	Value string
	Hint  string
}

type BackendDashboardCard struct {
	Label       string
	Body        string
	Href        string
	Status      string
	StatusClass string
}

type BackendDashboardPayment struct {
	Status            string
	StatusClass       string
	Source            string
	Mode              string
	SecretKey         string
	WebhookSecret     string
	PublishableKey    string
	Error             string
	UpdatedTime       BackendDashboardTime
	SaveActionPath    string
	TestActionPath    string
	CSRFToken         string
	SavePaymentAction BackendDashboardActionState
	TestPaymentAction BackendDashboardActionState
}

type BackendDashboardActionState struct {
	Submitted   bool
	OK          bool
	Message     string
	FieldErrors map[string]string
}

type BackendDashboardTime struct {
	Label   string
	Machine string
}

type BackendDashboardAlert struct {
	Kind  string
	Title string
	Body  string
	Href  string
}

type BackendDashboardChecklistItem struct {
	Label       string
	Href        string
	Status      string
	StatusClass string
}

type BackendDashboardResource struct {
	Label          string
	Route          string
	GeneratedLabel string
	CountLabel     string
}

type BackendDashboardTimelineEvent struct {
	Title       string
	Summary     string
	CreatedTime BackendDashboardTime
	Status      string
	StatusClass string
}

type BackendDashboardAuth struct {
	AdminCount    string
	GoogleEnabled string
}

type BackendDashboardOrder struct {
	ID          string
	ItemTitle   string
	StatusLabel string
	StatusClass string
	Total       string
}

type BackendDashboardContact struct {
	ID      string
	Name    string
	Email   string
	Message string
}

func RenderBackendDashboard(props BackendDashboardProps) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "admin-page"),
		gosx.Attr("data-gosx-studio-backend-dashboard-renderer", "gosx-studio"),
	),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("CMS")),
			gosx.El("h1", nil, gosx.Text("Studio operations")),
		),
		renderBackendDashboardStats(props.Stats),
		renderBackendDashboardOperate(props.Actions),
		renderBackendDashboardPayments(props.Payment),
		renderBackendDashboardAlerts(props.Alerts),
		renderBackendDashboardChecklist(props.Onboarding),
		renderBackendDashboardResources(props.Resources, props.ShowWorkbench),
		renderBackendDashboardTimeline("Recent changes", "/admin/search", "Find records", props.Audit, false),
		renderBackendDashboardTimeline("Stripe webhooks", "/admin/orders", "Orders", props.Webhooks, true),
		renderBackendDashboardIdentity(props.Auth),
		renderBackendDashboardAdminGrid(props.Orders, props.Contacts),
	)
}

func renderBackendDashboardStats(stats []BackendDashboardStat) gosx.Node {
	nodes := make([]gosx.Node, 0, len(stats))
	for _, stat := range stats {
		children := []gosx.Node{
			gosx.El("span", nil, gosx.Text(stat.Label)),
			gosx.El("strong", nil, gosx.Text(stat.Value)),
		}
		if stat.Hint != "" {
			children = append(children, gosx.El("small", nil, gosx.Text(stat.Hint)))
		}
		nodes = append(nodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "stat-card")), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "stat-grid")), gosx.Fragment(nodes...))
}

func renderBackendDashboardOperate(actions []BackendDashboardCard) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Operate")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/storefront"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Live preview")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "resource-grid")), gosx.Fragment(renderBackendDashboardCards(actions)...)),
	)
}

func renderBackendDashboardCards(cards []BackendDashboardCard) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(cards))
	for _, card := range cards {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(gosx.Attr("class", "resource-card"), gosx.Attr("href", card.Href), gosx.Attr("data-gosx-link", "true")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", card.StatusClass)), gosx.Text(card.Status)),
			gosx.El("strong", nil, gosx.Text(card.Label)),
			gosx.El("span", nil, gosx.Text(card.Body)),
		))
	}
	return nodes
}

func renderBackendDashboardPayments(payment BackendDashboardPayment) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel"), gosx.Attr("id", "payments")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Payments")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", payment.StatusClass)), gosx.Text(payment.Status)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "payment-admin-grid")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "stat-grid")),
				renderBackendDashboardPaymentStat("Source", payment.Source),
				renderBackendDashboardPaymentStat("Mode", payment.Mode),
				renderBackendDashboardPaymentStat("Secret key", payment.SecretKey),
				renderBackendDashboardPaymentStat("Webhook secret", payment.WebhookSecret),
				renderBackendDashboardPaymentStat("Publishable key", payment.PublishableKey),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "payment-form-stack")),
				renderBackendDashboardSavePaymentForm(payment),
				renderBackendDashboardTestPaymentForm(payment),
			),
		),
	)
}

func renderBackendDashboardPaymentStat(label, value string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "stat-card")),
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("strong", nil, gosx.Text(value)),
	)
}

func renderBackendDashboardSavePaymentForm(payment BackendDashboardPayment) gosx.Node {
	children := []gosx.Node{
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", payment.CSRFToken))),
	}
	children = append(children, renderBackendDashboardActionStatus(payment.SavePaymentAction)...)
	if payment.Error != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(payment.Error)))
	}
	children = append(children,
		renderBackendDashboardSecretField("stripeSecretKey", "Stripe secret key", payment.SecretKey, payment.SavePaymentAction.FieldErrors["stripeSecretKey"]),
		renderBackendDashboardSecretField("stripeWebhookSecret", "Webhook signing secret", payment.WebhookSecret, payment.SavePaymentAction.FieldErrors["stripeWebhookSecret"]),
		renderBackendDashboardSecretField("stripePublishableKey", "Publishable key", payment.PublishableKey, payment.SavePaymentAction.FieldErrors["stripePublishableKey"]),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
			gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "clearStripeSecretKey"))), gosx.Text(" Clear secret key")),
			gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "clearStripeWebhookSecret"))), gosx.Text(" Clear webhook secret")),
			gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "clearStripePublishableKey"))), gosx.Text(" Clear publishable key")),
		),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save payment keys")),
	)
	if payment.UpdatedTime.Machine != "" {
		children = append(children, gosx.El("time", gosx.Attrs(
			gosx.Attr("class", "field-note"),
			gosx.Attr("datetime", payment.UpdatedTime.Machine),
			gosx.Attr("data-viewer-time", "datetime"),
		), gosx.Text(payment.UpdatedTime.Label)))
	}
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "admin-form admin-form--single payment-secret-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", payment.SaveActionPath),
		gosx.Attr("data-payment-secret-form", "true"),
	), gosx.Fragment(children...))
}

func renderBackendDashboardTestPaymentForm(payment BackendDashboardPayment) gosx.Node {
	children := []gosx.Node{
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", payment.CSRFToken))),
	}
	children = append(children, renderBackendDashboardActionStatus(payment.TestPaymentAction)...)
	children = append(children, gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("type", "submit")), gosx.Text("Test connection")))
	return gosx.El("form", gosx.Attrs(gosx.Attr("class", "payment-test-form"), gosx.Attr("method", "post"), gosx.Attr("action", payment.TestActionPath)), gosx.Fragment(children...))
}

func renderBackendDashboardActionStatus(state BackendDashboardActionState) []gosx.Node {
	if state.Message == "" {
		return nil
	}
	if state.OK {
		return []gosx.Node{gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--ok")), gosx.Text(state.Message))}
	}
	if state.Submitted {
		return []gosx.Node{gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(state.Message))}
	}
	return nil
}

func renderBackendDashboardSecretField(id, label, placeholder, errorText string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("type", "password"), gosx.Attr("autocomplete", "off"), gosx.Attr("placeholder", placeholder))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-error")), gosx.Text(errorText)),
	)
}

func renderBackendDashboardAlerts(alerts []BackendDashboardAlert) gosx.Node {
	if len(alerts) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(alerts))
	for _, alert := range alerts {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(gosx.Attr("class", "alert-item"), gosx.Attr("href", alert.Href), gosx.Attr("data-gosx-link", "true")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(alert.Kind)),
			gosx.El("strong", nil, gosx.Text(alert.Title)),
			gosx.El("span", nil, gosx.Text(alert.Body)),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Needs attention")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/orders"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Orders")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "alert-list")), gosx.Fragment(nodes...)),
	)
}

func renderBackendDashboardChecklist(items []BackendDashboardChecklistItem) gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(gosx.Attr("class", "resource-card"), gosx.Attr("href", item.Href), gosx.Attr("data-gosx-link", "true")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", item.StatusClass)), gosx.Text(item.Status)),
			gosx.El("strong", nil, gosx.Text(item.Label)),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Launch checklist")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/settings"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Settings")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "resource-grid")), gosx.Fragment(nodes...)),
	)
}

func renderBackendDashboardResources(resources []BackendDashboardResource, showWorkbench bool) gosx.Node {
	header := []gosx.Node{
		gosx.El("h2", nil, gosx.Text("Resources")),
		gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/search"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Search")),
	}
	if showWorkbench {
		header = append(header, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/workbench"), gosx.Attr("data-gosx-link", "true")), gosx.Text("Open workbench")))
	}
	nodes := make([]gosx.Node, 0, len(resources))
	for _, resource := range resources {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(gosx.Attr("class", "resource-card"), gosx.Attr("href", resource.Route), gosx.Attr("data-gosx-link", "true")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text(resource.GeneratedLabel)),
			gosx.El("strong", nil, gosx.Text(resource.Label)),
			gosx.El("span", nil, gosx.Text(resource.CountLabel)),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")), gosx.Fragment(header...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "resource-grid")), gosx.Fragment(nodes...)),
	)
}

func renderBackendDashboardTimeline(title, href, linkLabel string, events []BackendDashboardTimelineEvent, showStatus bool) gosx.Node {
	if len(events) == 0 {
		return gosx.Fragment()
	}
	nodes := make([]gosx.Node, 0, len(events))
	for _, event := range events {
		children := []gosx.Node{gosx.El("strong", nil, gosx.Text(event.Title))}
		if showStatus {
			children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", event.StatusClass)), gosx.Text(event.Status)))
		}
		children = append(children, gosx.El("time", gosx.Attrs(gosx.Attr("datetime", event.CreatedTime.Machine), gosx.Attr("data-viewer-time", "datetime")), gosx.Text(event.CreatedTime.Label)))
		if event.Summary != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(event.Summary)))
		}
		nodes = append(nodes, gosx.El("li", nil, gosx.Fragment(children...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text(title)),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", href), gosx.Attr("data-gosx-link", "true")), gosx.Text(linkLabel)),
		),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list field-list--stacked")), gosx.Fragment(nodes...)),
	)
}

func renderBackendDashboardIdentity(auth BackendDashboardAuth) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Identity")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "status")), gosx.Text("Exact allowlist")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "stat-grid")),
			renderBackendDashboardPaymentStat("Admins", auth.AdminCount),
			renderBackendDashboardPaymentStat("Google OAuth", auth.GoogleEnabled),
		),
	)
}

func renderBackendDashboardAdminGrid(orders []BackendDashboardOrder, contacts []BackendDashboardContact) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-grid")),
		renderBackendDashboardOrders(orders),
		renderBackendDashboardContacts(contacts),
	)
}

func renderBackendDashboardOrders(orders []BackendDashboardOrder) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Recent orders")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/orders"), gosx.Attr("data-gosx-link", "true")), gosx.Text("View all")),
		),
	}
	if len(orders) == 0 {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text("No orders yet.")))
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
	}
	rows := make([]gosx.Node, 0, len(orders))
	for _, order := range orders {
		rows = append(rows, gosx.El("tr", nil,
			gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/orders/"+order.ID), gosx.Attr("data-gosx-link", "true")), gosx.Text(order.ItemTitle))),
			gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", order.StatusClass)), gosx.Text(order.StatusLabel))),
			gosx.El("td", nil, gosx.Text(order.Total)),
		))
	}
	children = append(children, gosx.El("table", gosx.Attrs(gosx.Attr("class", "data-table")),
		gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Order")), gosx.El("th", nil, gosx.Text("Status")), gosx.El("th", nil, gosx.Text("Total")))),
		gosx.El("tbody", nil, gosx.Fragment(rows...)),
	))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
}

func renderBackendDashboardContacts(contacts []BackendDashboardContact) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text("Messages")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/contacts"), gosx.Attr("data-gosx-link", "true")), gosx.Text("View all")),
		),
	}
	if len(contacts) == 0 {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text("No contact submissions yet.")))
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
	}
	articles := make([]gosx.Node, 0, len(contacts))
	for _, contact := range contacts {
		articles = append(articles, gosx.El("article", nil,
			gosx.El("strong", nil, gosx.Text(contact.Name)),
			gosx.El("span", nil, gosx.Text(contact.Email)),
			gosx.El("p", nil, gosx.Text(contact.Message)),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/contacts/"+contact.ID), gosx.Attr("data-gosx-link", "true")), gosx.Text("Open")),
		))
	}
	children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", "message-list")), gosx.Fragment(articles...)))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
}
