package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderCheckoutPanelPopulated(t *testing.T) {
	view := map[string]any{
		"class":          "panel editor-panel editor-panel--checkout studio-checkout-panel",
		"panelKey":       "checkout",
		"headerClass":    "panel__header",
		"kicker":         "Checkout",
		"title":          "Stripe checkout",
		"help":           "Turn on the checkout block, pick a published product, and set the button label. Changes are saved to the draft and go live when you publish.",
		"empty":          "Publish a product to offer it through checkout.",
		"enabledName":    "checkoutEnabled",
		"enabledValue":   "on",
		"enabled":        true,
		"enabledLabel":   "Enable checkout block",
		"productName":    "checkoutProductId",
		"productID":      "checkoutProductId",
		"productLabel":   "Product",
		"labelName":      "checkoutLabel",
		"labelID":        "checkoutLabel",
		"labelFieldText": "Button label",
		"label":          "Reserve this piece",
		"labelHelp":      "The call-to-action shown on the public checkout button.",
		"action":         "/admin/editor/__actions/checkout",
		"buttonLabel":    "Save checkout",
		"formID":         "websiteEditorForm",
		"hasProducts":    true,
		"products": []map[string]any{
			{"value": "prod-1", "label": "Handled vase", "selected": false},
			{"value": "prod-2", "label": "Studio mug", "selected": true},
		},
	}

	html := gosx.RenderHTML(RenderCheckoutPanel(view, CheckoutPanelOptions{}))

	for _, fragment := range []string{
		`<section class="panel editor-panel editor-panel--checkout studio-checkout-panel" data-panel-key="checkout" data-studio-checkout-panel="true" data-gosx-studio-checkout-panel-renderer="gosx-studio">`,
		`<div class="panel__header">`,
		`<p class="kicker">Checkout</p>`,
		`<h2>Stripe checkout</h2>`,
		`<p class="field-help">Turn on the checkout block, pick a published product, and set the button label. Changes are saved to the draft and go live when you publish.</p>`,
		`<div class="inline-form studio-checkout-panel__form">`,
		`<input name="checkoutEnabled" type="checkbox" value="on" checked />`,
		`<span>Enable checkout block</span>`,
		`<p class="empty" hidden>Publish a product to offer it through checkout.</p>`,
		`<div class="studio-checkout-panel__field"><label for="checkoutProductId">Product</label><select id="checkoutProductId" name="checkoutProductId" data-studio-checkout-product="true">`,
		`<option value="prod-1">Handled vase</option>`,
		`<option value="prod-2" selected>Studio mug</option>`,
		`<label for="checkoutLabel">Button label</label>`,
		`<input id="checkoutLabel" name="checkoutLabel" type="text" value="Reserve this piece" />`,
		`<small class="field-help">The call-to-action shown on the public checkout button.</small>`,
		`<button class="button button--primary" type="submit" form="websiteEditorForm" formaction="/admin/editor/__actions/checkout" formmethod="post" data-studio-submit-action="checkout" data-studio-field-action-formaction="/admin/editor/__actions/checkout">Save checkout</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered checkout panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		"<form",
		"csrf_token",
		`name="endpoint"`,
		`name="action"`,
		`name="checkoutEndpoint"`,
		`<div class="studio-checkout-panel__field" hidden><label for="checkoutProductId">Product</label>`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered checkout panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderCheckoutPanelEmptyProducts(t *testing.T) {
	view := map[string]any{
		"class":          "panel editor-panel editor-panel--checkout studio-checkout-panel",
		"panelKey":       "checkout",
		"headerClass":    "panel__header",
		"kicker":         "Checkout",
		"title":          "Stripe checkout",
		"help":           "Turn on the checkout block, pick a published product, and set the button label. Changes are saved to the draft and go live when you publish.",
		"empty":          "Publish a product to offer it through checkout.",
		"enabledName":    "checkoutEnabled",
		"enabledValue":   "on",
		"enabledLabel":   "Enable checkout block",
		"productName":    "checkoutProductId",
		"productID":      "checkoutProductId",
		"productLabel":   "Product",
		"labelName":      "checkoutLabel",
		"labelID":        "checkoutLabel",
		"labelFieldText": "Button label",
		"labelHelp":      "The call-to-action shown on the public checkout button.",
		"action":         "/admin/editor/__actions/checkout",
		"buttonLabel":    "Save checkout",
		"formID":         "websiteEditorForm",
	}

	html := gosx.RenderHTML(RenderCheckoutPanel(view, CheckoutPanelOptions{}))

	for _, fragment := range []string{
		`data-studio-checkout-panel="true"`,
		`data-gosx-studio-checkout-panel-renderer="gosx-studio"`,
		`<input name="checkoutEnabled" type="checkbox" value="on" />`,
		`<p class="empty">Publish a product to offer it through checkout.</p>`,
		`<div class="studio-checkout-panel__field" hidden><label for="checkoutProductId">Product</label><select id="checkoutProductId" name="checkoutProductId" data-studio-checkout-product="true"></select></div>`,
		`<button class="button button--primary" type="submit" form="websiteEditorForm" formaction="/admin/editor/__actions/checkout" formmethod="post" data-studio-submit-action="checkout" data-studio-field-action-formaction="/admin/editor/__actions/checkout">Save checkout</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty checkout panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		"<form",
		"csrf_token",
		`<option`,
		`name="endpoint"`,
		`name="action"`,
		`name="checkoutEndpoint"`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty checkout panel must not include %q:\n%s", notWant, html)
		}
	}
}
