package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendDashboardPopulated(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendDashboard(BackendDashboardProps{
		Stats: []BackendDashboardStat{{Label: "Orders", Value: "12", Hint: "3 pending"}},
		Actions: []BackendDashboardCard{{
			Label: "Editor", Body: "studio template", Href: "/admin/editor", Status: "editable", StatusClass: "status status--ready",
		}},
		Payment: BackendDashboardPayment{
			Status:         "Ready",
			StatusClass:    "status status--ready",
			Source:         "store",
			Mode:           "test",
			SecretKey:      "sk_test_...",
			WebhookSecret:  "whsec_...",
			PublishableKey: "pk_test_...",
			UpdatedTime:    BackendDashboardTime{Label: "Jun 28, 2026 4:30 PM UTC", Machine: "2026-06-28T16:30:00Z"},
			SaveActionPath: "/admin/__actions/savePayment",
			TestActionPath: "/admin/__actions/testPayment",
			CSRFToken:      "csrf-token",
			SavePaymentAction: BackendDashboardActionState{
				Submitted:   true,
				OK:          false,
				Message:     "Please correct the highlighted fields.",
				FieldErrors: map[string]string{"stripeSecretKey": "Use a Stripe secret key."},
			},
			TestPaymentAction: BackendDashboardActionState{Submitted: true, OK: true, Message: "Stripe test mode verified."},
		},
		Alerts: []BackendDashboardAlert{{Kind: "order", Title: "Pending order", Body: "Order needs review", Href: "/admin/orders"}},
		Onboarding: []BackendDashboardChecklistItem{{
			Label: "Stripe checkout", Href: "/admin#payments", Status: "complete", StatusClass: "status status--complete",
		}},
		Resources: []BackendDashboardResource{{
			Label: "Products", Route: "/admin/products", GeneratedLabel: "Generated", CountLabel: "4 records",
		}},
		Audit: []BackendDashboardTimelineEvent{{
			Title: "Updated product", Summary: "Studio mug changed", CreatedTime: BackendDashboardTime{Label: "Now", Machine: "2026-06-28T16:31:00Z"},
		}},
		Webhooks: []BackendDashboardTimelineEvent{{
			Title: "checkout.session.completed", Status: "received", StatusClass: "status status--ready", Summary: "evt_123", CreatedTime: BackendDashboardTime{Label: "Now", Machine: "2026-06-28T16:32:00Z"},
		}},
		Auth: BackendDashboardAuth{AdminCount: "2", GoogleEnabled: "true"},
		Orders: []BackendDashboardOrder{{
			ID: "order-1", ItemTitle: "Handled vase", StatusLabel: "paid", StatusClass: "status status--ready", Total: "$120.00",
		}},
		Contacts: []BackendDashboardContact{{
			ID: "contact-1", Name: "Ada", Email: "ada@example.test", Message: "Is this available?",
		}},
		ShowWorkbench: true,
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-dashboard-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Studio operations</h1></section>`,
		`<div class="stat-card"><span>Orders</span><strong>12</strong><small>3 pending</small></div>`,
		`<h2>Operate</h2><a href="/admin/storefront" data-gosx-link="true">Live preview</a>`,
		`<a class="resource-card" href="/admin/editor" data-gosx-link="true"><span class="status status--ready">editable</span><strong>Editor</strong><span>studio template</span></a>`,
		`<section class="panel" id="payments">`,
		`<span class="status status--ready">Ready</span>`,
		`<strong>sk_test_...</strong>`,
		`<form class="admin-form admin-form--single payment-secret-form" method="post" action="/admin/__actions/savePayment" data-payment-secret-form="true">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<input id="stripeSecretKey" name="stripeSecretKey" type="password" autocomplete="off" placeholder="sk_test_..." />`,
		`<p class="form-error">Use a Stripe secret key.</p>`,
		`<label><input type="checkbox" name="clearStripeSecretKey" /> Clear secret key</label>`,
		`<time class="field-note" datetime="2026-06-28T16:30:00Z" data-viewer-time="datetime">Jun 28, 2026 4:30 PM UTC</time>`,
		`<form class="payment-test-form" method="post" action="/admin/__actions/testPayment">`,
		`<p class="form-status form-status--ok">Stripe test mode verified.</p>`,
		`<h2>Needs attention</h2>`,
		`<a class="alert-item" href="/admin/orders" data-gosx-link="true"><span class="status">order</span><strong>Pending order</strong><span>Order needs review</span></a>`,
		`<h2>Launch checklist</h2>`,
		`<strong>Stripe checkout</strong>`,
		`<h2>Resources</h2><a href="/admin/search" data-gosx-link="true">Search</a><a href="/admin/workbench" data-gosx-link="true">Open workbench</a>`,
		`<span class="status">Generated</span><strong>Products</strong><span>4 records</span>`,
		`<h2>Recent changes</h2>`,
		`<strong>Updated product</strong><time datetime="2026-06-28T16:31:00Z" data-viewer-time="datetime">Now</time><p>Studio mug changed</p>`,
		`<h2>Stripe webhooks</h2>`,
		`<strong>checkout.session.completed</strong><span class="status status--ready">received</span>`,
		`<h2>Identity</h2>`,
		`<span>Admins</span><strong>2</strong>`,
		`<td><a href="/admin/orders/order-1" data-gosx-link="true">Handled vase</a></td>`,
		`<article><strong>Ada</strong><span>ada@example.test</span><p>Is this available?</p><a href="/admin/contacts/contact-1" data-gosx-link="true">Open</a></article>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered backend dashboard missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendDashboardEmptyStates(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendDashboard(BackendDashboardProps{
		Payment: BackendDashboardPayment{
			Status:         "Needs key",
			StatusClass:    "status status--request",
			Source:         "missing",
			Mode:           "missing",
			SecretKey:      "Not set",
			WebhookSecret:  "Not set",
			PublishableKey: "Not set",
			SaveActionPath: "/admin/__actions/savePayment",
			TestActionPath: "/admin/__actions/testPayment",
		},
		Auth: BackendDashboardAuth{AdminCount: "0", GoogleEnabled: "false"},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-backend-dashboard-renderer="gosx-studio"`,
		`<section class="panel" id="payments">`,
		`<p class="empty">No orders yet.</p>`,
		`<p class="empty">No contact submissions yet.</p>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty backend dashboard missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`Needs attention`,
		`Recent changes`,
		`Stripe webhooks`,
		`Open workbench`,
		`form-status form-status--ok`,
		`form-status form-status--error`,
		`class="field-note"`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty backend dashboard must not include %q:\n%s", notWant, html)
		}
	}
}
