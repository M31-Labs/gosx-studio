package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendResourceIndexOrderTable(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendResourceIndex(BackendResourceIndexProps{
		Kicker:  "CMS",
		Title:   "Orders",
		Summary: "Checkout sessions, payment status, and fulfillment totals.",
		Empty:   "No orders yet.",
		Table: &BackendResourceTable{
			Headers: []string{"Order", "Customer", "Status", "Total", "Created", ""},
			Rows: []BackendResourceTableRow{{
				Cells: []BackendResourceTableCell{
					{Primary: "Mug <Special>", Secondary: "order-1"},
					{Text: "ada@example.test"},
					{Text: "paid", StatusClass: "status status--ready"},
					{Text: "$120.00"},
					{Time: BackendResourceTime{Label: "Jun 28, 2026 4:30 PM", Machine: "2026-06-28T16:30:00Z"}},
				},
				Action: BackendResourceLink{Href: "/admin/orders/order-1", Label: "Open", GOSXLink: true},
			}},
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Orders</h1><p>Checkout sessions, payment status, and fulfillment totals.</p></section>`,
		`<section class="panel"><table class="data-table">`,
		`<thead><tr><th>Order</th><th>Customer</th><th>Status</th><th>Total</th><th>Created</th><th></th></tr></thead>`,
		`<td><strong>Mug &lt;Special&gt;</strong><span>order-1</span></td>`,
		`<td>ada@example.test</td>`,
		`<td><span class="status status--ready">paid</span></td>`,
		`<td><time datetime="2026-06-28T16:30:00Z" data-viewer-time="datetime">Jun 28, 2026 4:30 PM</time></td>`,
		`<td><a href="/admin/orders/order-1" data-gosx-link="true">Open</a></td>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered order resource index missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<p class="empty">No orders yet.</p>`) {
		t.Fatalf("populated order resource index rendered empty state:\n%s", html)
	}
}

func TestRenderBackendResourceIndexContactCards(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendResourceIndex(BackendResourceIndexProps{
		Kicker:  "CMS",
		Title:   "Contacts",
		Summary: "Studio inquiries captured by the Go-native contact action and API.",
		Empty:   "No contact submissions yet.",
		Cards: []BackendResourceCard{{
			Title:       "Ada",
			Status:      "new",
			StatusClass: "status status--new",
			Body:        "Is <this> available?",
			Email:       BackendResourceLink{Href: "mailto:ada@example.test", Label: "ada@example.test"},
			Action:      BackendResourceLink{Href: "/admin/contacts/contact-1", Label: "Open", GOSXLink: true},
			Time:        BackendResourceTime{Label: "Jun 28, 2026 4:35 PM", Machine: "2026-06-28T16:35:00Z"},
		}},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-backend-resource-index-renderer="gosx-studio"`,
		`<section class="message-list message-list--full">`,
		`<article class="panel"><div class="panel__header"><h2>Ada</h2><span class="status status--new">new</span></div>`,
		`<p>Is &lt;this&gt; available?</p>`,
		`<div class="button-row"><a href="mailto:ada@example.test">ada@example.test</a><a href="/admin/contacts/contact-1" data-gosx-link="true">Open</a><time class="field-note" datetime="2026-06-28T16:35:00Z" data-viewer-time="datetime">Jun 28, 2026 4:35 PM</time></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered contact resource index missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendResourceIndexEmptyStates(t *testing.T) {
	tableHTML := gosx.RenderHTML(RenderBackendResourceIndex(BackendResourceIndexProps{
		Kicker: "CMS", Title: "Orders", Summary: "Summary", Empty: "No orders yet.",
		Table: &BackendResourceTable{Headers: []string{"Order", ""}},
	}))
	if !strings.Contains(tableHTML, `<p class="empty">No orders yet.</p>`) {
		t.Fatalf("empty table index missing empty state:\n%s", tableHTML)
	}

	cardHTML := gosx.RenderHTML(RenderBackendResourceIndex(BackendResourceIndexProps{
		Kicker: "CMS", Title: "Contacts", Summary: "Summary", Empty: "No contact submissions yet.",
	}))
	if !strings.Contains(cardHTML, `<p class="empty">No contact submissions yet.</p>`) {
		t.Fatalf("empty card index missing empty state:\n%s", cardHTML)
	}
}

func TestRenderBackendResourceIndexContentOmitsPageWrapper(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendResourceIndexContent(BackendResourceIndexProps{
		Kicker:  "CMS",
		Title:   "Categories",
		Summary: "Storefront taxonomy for grouping products and shaping headless commerce queries.",
		Empty:   "No categories yet.",
		Table: &BackendResourceTable{
			Headers: []string{"Category", "Slug", "Status", "Updated", ""},
		},
	}))

	for _, fragment := range []string{
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Categories</h1><p>Storefront taxonomy for grouping products and shaping headless commerce queries.</p></section>`,
		`<section class="panel"><table class="data-table">`,
		`<thead><tr><th>Category</th><th>Slug</th><th>Status</th><th>Updated</th><th></th></tr></thead>`,
		`<p class="empty">No categories yet.</p>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("resource index content missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `data-gosx-studio-backend-resource-index-renderer="gosx-studio"`) {
		t.Fatalf("resource index content should not render its own page wrapper:\n%s", html)
	}
}
