package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendOrderDetailRendersReadOnlyPanels(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendOrderDetail(BackendOrderDetailProps{
		Kicker:  "CMS",
		Title:   "Order",
		Summary: "order_123",
		SummaryPanel: BackendOrderSummary{
			CustomerName:      "Ari <Blue>",
			Email:             "ari@example.com",
			Phone:             "555-0100",
			Subtotal:          "$120.00",
			Shipping:          "$8.00",
			Total:             "$128.00",
			StatusLabel:       "Paid",
			StatusClass:       "status status--paid",
			FulfillmentLabel:  "Shipped",
			FulfillmentClass:  "status status--shipped",
			TrackingReference: "TRACK123",
			Created:           BackendOrderTime{Label: "Jun 27, 2026 8:00 AM", Machine: "2026-06-27T08:00:00Z"},
			Updated:           BackendOrderTime{Label: "Jun 28, 2026 4:45 PM", Machine: "2026-06-28T16:45:00Z"},
		},
		Items: []BackendOrderItem{{
			Title:    "Handled bowl",
			Quantity: "2",
			Price:    "$60.00",
			Href:     "/admin/products/prod_1",
		}},
		ShippingAddress: []BackendOrderSpecRow{
			{Label: "Name", Value: "Ari Blue"},
			{Label: "City", Value: "Portland"},
		},
		PaymentReferences: BackendOrderPaymentReferences{
			Provider: "stripe",
			Session:  "cs_test",
			Payment:  "pi_test",
		},
		Timeline: []BackendOrderTimelineEvent{{
			Label:   "Paid",
			Detail:  "Stripe checkout completed.",
			Created: BackendOrderTime{Label: "Jun 27, 2026 8:15 AM", Machine: "2026-06-27T08:15:00Z"},
		}},
		Audit: []BackendOrderAuditEvent{{
			Action:  "order.mark_paid",
			Summary: "Marked paid by admin.",
			Created: BackendOrderTime{Label: "Jun 27, 2026 8:16 AM", Machine: "2026-06-27T08:16:00Z"},
		}},
		Webhooks: []BackendOrderWebhookEvent{{
			Type:        "checkout.session.completed",
			Status:      "processed",
			StatusClass: "status status--processed",
			Summary:     "Payment received.",
			Created:     BackendOrderTime{Label: "Jun 27, 2026 8:17 AM", Machine: "2026-06-27T08:17:00Z"},
		}},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-order-detail-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Order</h1><p>order_123</p></section>`,
		`<div class="panel"><div class="panel__header"><h2>Summary</h2><span class="status status--paid">Paid</span></div><dl class="spec-list"><div><dt>Customer</dt><dd>Ari &lt;Blue&gt;</dd></div><div><dt>Email</dt><dd><a href="mailto:ari@example.com">ari@example.com</a></dd></div><div><dt>Phone</dt><dd>555-0100</dd></div><div><dt>Subtotal</dt><dd>$120.00</dd></div><div><dt>Shipping</dt><dd>$8.00</dd></div><div><dt>Total</dt><dd><strong>$128.00</strong></dd></div><div><dt>Fulfillment</dt><dd><span class="status status--shipped">Shipped</span></dd></div><div><dt>Tracking</dt><dd>TRACK123</dd></div><div><dt>Created</dt><dd><time datetime="2026-06-27T08:00:00Z" data-viewer-time="datetime">Jun 27, 2026 8:00 AM</time></dd></div><div><dt>Updated</dt><dd><time datetime="2026-06-28T16:45:00Z" data-viewer-time="datetime">Jun 28, 2026 4:45 PM</time></dd></div></dl></div>`,
		`<section class="panel"><div class="panel__header"><h2>Items</h2></div><table class="data-table"><thead><tr><th>Item</th><th>Quantity</th><th>Price</th><th></th></tr></thead><tbody><tr><td><strong>Handled bowl</strong></td><td>2</td><td>$60.00</td><td><a href="/admin/products/prod_1" data-gosx-link="true">Product</a></td></tr></tbody></table></section>`,
		`<section class="panel"><div class="panel__header"><h2>Shipping address</h2></div><dl class="spec-list"><div><dt>Name</dt><dd>Ari Blue</dd></div><div><dt>City</dt><dd>Portland</dd></div></dl></section>`,
		`<section class="panel"><div class="panel__header"><h2>Payment references</h2></div><dl class="spec-list"><div><dt>Provider</dt><dd>stripe</dd></div><div><dt>Session</dt><dd>cs_test</dd></div><div><dt>Payment</dt><dd>pi_test</dd></div></dl></section>`,
		`<div class="panel"><div class="panel__header"><h2>Timeline</h2></div><ul class="field-list field-list--stacked"><li><strong>Paid</strong><time datetime="2026-06-27T08:15:00Z" data-viewer-time="datetime">Jun 27, 2026 8:15 AM</time><p>Stripe checkout completed.</p></li></ul></div>`,
		`<div class="panel"><div class="panel__header"><h2>Audit</h2></div><ul class="field-list field-list--stacked"><li><strong>order.mark_paid</strong><time datetime="2026-06-27T08:16:00Z" data-viewer-time="datetime">Jun 27, 2026 8:16 AM</time><p>Marked paid by admin.</p></li></ul></div>`,
		`<div class="panel"><div class="panel__header"><h2>Webhook events</h2></div><ul class="field-list field-list--stacked"><li><strong>checkout.session.completed</strong><span class="status status--processed">processed</span><time datetime="2026-06-27T08:17:00Z" data-viewer-time="datetime">Jun 27, 2026 8:17 AM</time><p>Payment received.</p></li></ul></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("order detail html missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendOrderDetailSplitNodesAndEmptyStates(t *testing.T) {
	heading := gosx.RenderHTML(RenderBackendOrderDetailHeading(BackendOrderDetailProps{
		Kicker:  "CMS",
		Title:   "Order",
		Summary: "order_123",
	}))
	summary := gosx.RenderHTML(RenderBackendOrderDetailSummary(BackendOrderSummary{}))
	items := gosx.RenderHTML(RenderBackendOrderDetailItems(nil))
	shipping := gosx.RenderHTML(RenderBackendOrderDetailShippingAddress(nil))
	payment := gosx.RenderHTML(RenderBackendOrderDetailPaymentReferences(BackendOrderPaymentReferences{}))
	timeline := gosx.RenderHTML(RenderBackendOrderDetailTimeline(nil))
	audit := gosx.RenderHTML(RenderBackendOrderDetailAudit(nil))
	webhooks := gosx.RenderHTML(RenderBackendOrderDetailWebhooks(nil))
	content := gosx.RenderHTML(RenderBackendOrderDetailContent(BackendOrderDetailProps{}))

	for _, check := range []struct {
		name string
		html string
		want string
	}{
		{name: "heading", html: heading, want: `<section class="admin-heading"><p class="kicker">CMS</p><h1>Order</h1><p>order_123</p></section>`},
		{name: "summary", html: summary, want: `<div class="panel"><div class="panel__header"><h2>Summary</h2><span class="status"></span></div>`},
		{name: "items", html: items, want: `<section class="panel"><div class="panel__header"><h2>Items</h2></div><table class="data-table">`},
		{name: "payment", html: payment, want: `<h2>Payment references</h2>`},
		{name: "timeline", html: timeline, want: `<p class="empty">No timeline events yet.</p>`},
		{name: "audit", html: audit, want: `<p class="empty">No audit events yet.</p>`},
		{name: "webhooks", html: webhooks, want: `<p class="empty">No Stripe webhook events for this order yet.</p>`},
	} {
		if !strings.Contains(check.html, check.want) {
			t.Fatalf("%s node missing %q:\n%s", check.name, check.want, check.html)
		}
	}
	if strings.Contains(shipping, `Shipping address`) {
		t.Fatalf("empty shipping address should render no section:\n%s", shipping)
	}
	if strings.Contains(content, `data-gosx-studio-backend-order-detail-renderer="gosx-studio"`) {
		t.Fatalf("content renderer should not render its own page wrapper:\n%s", content)
	}
	if strings.Contains(summary, `<dt>Phone</dt>`) || strings.Contains(summary, `<dt>Tracking</dt>`) {
		t.Fatalf("summary should omit empty optional phone/tracking rows:\n%s", summary)
	}
}

func TestRenderBackendOrderDetailActionsAndNotesForms(t *testing.T) {
	actions := BackendOrderActions{
		OrderID:   "order_123",
		CSRFToken: "csrf-token",
		Paths: BackendOrderActionPaths{
			MarkPaid:        "/mark-paid",
			Cancel:          "/cancel",
			MarkRefund:      "/refund",
			SaveFulfillment: "/fulfillment",
			SaveNotes:       "/notes",
		},
		CanMarkPaid:       true,
		CanCancel:         true,
		CanRefund:         true,
		RefundNote:        "Refund <reason>",
		TrackingReference: "TRACK123",
		FulfillmentNote:   "Packed carefully",
		FulfillmentOptions: []BackendOrderFulfillmentOption{
			{Value: "pending", Label: "Pending"},
			{Value: "shipped", Label: "Shipped", Selected: true},
		},
	}
	actionsHTML := gosx.RenderHTML(RenderBackendOrderDetailActions(actions))
	notesHTML := gosx.RenderHTML(RenderBackendOrderDetailNotes(BackendOrderNotes{
		OrderID:   actions.OrderID,
		CSRFToken: actions.CSRFToken,
		Action:    actions.Paths.SaveNotes,
		Notes:     "Customer prefers pickup",
	}))

	for _, fragment := range []string{
		`<div class="panel"><div class="panel__header"><h2>Actions</h2><a href="/admin/orders" data-gosx-link="true">Back to orders</a></div>`,
		`<form class="inline-form" method="post" action="/mark-paid"><input type="hidden" name="csrf_token" value="csrf-token" /><input type="hidden" name="id" value="order_123" /><button class="button button--primary" type="submit" data-admin-confirm="Mark this order paid and mark the item sold?">Mark paid</button></form>`,
		`<form class="inline-form" method="post" action="/cancel"><input type="hidden" name="csrf_token" value="csrf-token" /><input type="hidden" name="id" value="order_123" /><button class="button button--secondary" type="submit" data-admin-confirm="Cancel this order and release reserved inventory?">Cancel and release</button></form>`,
		`<form class="admin-form admin-form--single" method="post" action="/refund"><input type="hidden" name="csrf_token" value="csrf-token" /><input type="hidden" name="id" value="order_123" /><div class="field-row"><label for="refundNote">Refund note</label><textarea id="refundNote" name="refundNote" rows="3">Refund &lt;reason&gt;</textarea></div><button class="button button--secondary" type="submit" data-admin-confirm="Record this order as refunded?">Record refund</button></form>`,
		`<form class="admin-form admin-form--single" method="post" action="/fulfillment"><input type="hidden" name="csrf_token" value="csrf-token" /><input type="hidden" name="id" value="order_123" /><div class="field-row"><label for="fulfillmentStatus">Fulfillment</label><select id="fulfillmentStatus" name="fulfillmentStatus"><option value="pending">Pending</option><option value="shipped" selected>Shipped</option></select></div><div class="field-row"><label for="trackingReference">Tracking/reference</label><input id="trackingReference" name="trackingReference" value="TRACK123" /></div><div class="field-row"><label for="fulfillmentNote">Fulfillment note</label><textarea id="fulfillmentNote" name="fulfillmentNote" rows="3">Packed carefully</textarea></div><button class="button button--secondary" type="submit">Save fulfillment</button></form>`,
	} {
		if !strings.Contains(actionsHTML, fragment) {
			t.Fatalf("actions html missing %q:\n%s", fragment, actionsHTML)
		}
	}
	for _, fragment := range []string{
		`<section class="panel"><div class="panel__header"><h2>Notes</h2></div><form class="admin-form admin-form--single" method="post" action="/notes">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" /><input type="hidden" name="id" value="order_123" />`,
		`<textarea id="notes" name="notes" rows="5">Customer prefers pickup</textarea>`,
		`<button class="button button--primary" type="submit">Save notes</button>`,
	} {
		if !strings.Contains(notesHTML, fragment) {
			t.Fatalf("notes html missing %q:\n%s", fragment, notesHTML)
		}
	}
}

func TestRenderBackendOrderDetailPageOwnsActionsAndNotes(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendOrderDetailPage(BackendOrderDetailProps{
		Kicker:  "CMS",
		Title:   "Order",
		Summary: "order_123",
		SummaryPanel: BackendOrderSummary{
			CustomerName:     "Ari Blue",
			Email:            "ari@example.com",
			Subtotal:         "$120.00",
			Shipping:         "$8.00",
			Total:            "$128.00",
			StatusLabel:      "Reserved",
			FulfillmentLabel: "Pending",
		},
		Actions: BackendOrderActions{
			OrderID:   "order_123",
			CSRFToken: "csrf-token",
			Paths: BackendOrderActionPaths{
				SaveFulfillment: "/fulfillment",
				SaveNotes:       "/notes",
			},
			FulfillmentOptions: []BackendOrderFulfillmentOption{{Value: "pending", Label: "Pending", Selected: true}},
		},
		Notes: BackendOrderNotes{
			OrderID:   "order_123",
			CSRFToken: "csrf-token",
			Action:    "/notes",
			Notes:     "Internal note",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-order-detail-renderer="gosx-studio">`,
		`<section class="admin-grid"><div class="panel"><div class="panel__header"><h2>Summary</h2>`,
		`<h2>Actions</h2><a href="/admin/orders" data-gosx-link="true">Back to orders</a>`,
		`<h2>Items</h2>`,
		`<h2>Notes</h2>`,
		`<h2>Payment references</h2>`,
		`<section class="admin-grid"><div class="panel"><div class="panel__header"><h2>Timeline</h2>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("page html missing %q:\n%s", fragment, html)
		}
	}
	for _, hidden := range []string{
		`Mark paid`,
		`Cancel and release`,
		`Record refund`,
		`refundNote`,
	} {
		if strings.Contains(html, hidden) {
			t.Fatalf("page html should hide unavailable action %q:\n%s", hidden, html)
		}
	}
}
