package sitehost

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func signStripe(payload []byte, secret string, at time.Time) string {
	t := strconv.FormatInt(at.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(t + "."))
	mac.Write(payload)
	return "t=" + t + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestStripeSignatureIsChecked(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	payload := []byte(`{"type":"checkout.session.completed"}`)
	if err := verifyStripeSignature(payload, signStripe(payload, "whsec_test", now), "whsec_test", now); err != nil {
		t.Fatalf("valid signature refused: %v", err)
	}
	if err := verifyStripeSignature([]byte(`{"type":"other"}`), signStripe(payload, "whsec_test", now), "whsec_test", now); err == nil {
		t.Fatal("a tampered payload passed")
	}
	if err := verifyStripeSignature(payload, signStripe(payload, "whsec_test", now.Add(-10*time.Minute)), "whsec_test", now); err == nil {
		t.Fatal("an old signature passed")
	}
	if err := verifyStripeSignature(payload, signStripe(payload, "whsec_test", now), "", now); err == nil {
		t.Fatal("no secret must refuse everything")
	}
}

func postWebhook(t *testing.T, handler http.Handler, payload, secret string, at time.Time) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, stripeWebhookPath, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", signStripe([]byte(payload), secret, at))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCheckoutFromCartToPaidOrder(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	var captured url.Values
	createStripeSession = func(ctx context.Context, key string, params url.Values) (stripeSession, error) {
		if key != "sk_test_abc" {
			t.Fatalf("wrong key %q", key)
		}
		captured = params
		return stripeSession{ID: "cs_test_123", URL: "https://checkout.stripe.com/c/pay/cs_test_123"}, nil
	}
	t.Cleanup(func() { createStripeSession = nil })

	host, handler := newTestHost(t)
	// Payments off: checkout just goes back to the cart.
	if rec := post(t, handler, checkoutPath, url.Values{}); rec.Header().Get("Location") != cartPath {
		t.Fatalf("checkout without payments = %q", rec.Header().Get("Location"))
	}

	postSettings(t, handler, map[string]string{
		"title": "Wildflower Bakery", "currency": "usd", "countVisitors": "on",
		"stripeSecretKey": "sk_test_abc", "stripeWebhookSecret": "whsec_xyz",
		"shippingFlat": "5.00", "shippingFreeOver": "50", "shipCountries": "US, CA", "stripeTax": "true", "allowPromos": "true",
	}, nil)
	if !host.checkoutReady() {
		t.Fatal("payments should be connected")
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "Connected (test mode)", "Settings shows the connection")
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "https://wildflower.example/stripe/webhook", "and the webhook address to paste into Stripe")

	id := createProduct(t, handler, "Sourdough loaf", "6.50")
	saveProduct(t, handler, id, map[string]string{"name": "Sourdough loaf", "slug": "sourdough-loaf", "price": "6.50", "imageCount": "0", "variantCount": "0", "trackStock": "1", "stock": "5", "ships": "1", "active": "1"})
	rec := post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"2"}})
	cookie := rec.Result().Cookies()[0]
	mustContain(t, getWithCookie(t, handler, "/cart", cookie).Body.String(), `formaction="/checkout"`, "the cart offers checkout")

	rec = postWithCookie(t, handler, checkoutPath, url.Values{}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "https://checkout.stripe.com/c/pay/cs_test_123" {
		t.Fatalf("checkout = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	for key, want := range map[string]string{
		"mode":                                                          "payment",
		"line_items[0][quantity]":                                       "2",
		"line_items[0][price_data][unit_amount]":                        "650",
		"line_items[0][price_data][currency]":                           "usd",
		"line_items[0][price_data][product_data][name]":                 "Sourdough loaf",
		"line_items[0][price_data][tax_behavior]":                       "exclusive",
		"shipping_address_collection[allowed_countries][0]":             "US",
		"shipping_address_collection[allowed_countries][1]":             "CA",
		"shipping_options[0][shipping_rate_data][fixed_amount][amount]": "500",
		"shipping_options[0][shipping_rate_data][display_name]":         "Shipping",
		"automatic_tax[enabled]":                                        "true",
		"allow_promotion_codes":                                         "true",
		"success_url":                                                   "https://wildflower.example/checkout/done?session_id={CHECKOUT_SESSION_ID}",
		"cancel_url":                                                    "https://wildflower.example/cart",
	} {
		if got := captured.Get(key); got != want {
			t.Fatalf("session param %s = %q, want %q", key, got, want)
		}
	}
	orders := host.orders.list()
	if len(orders) != 1 || orders[0].Status != "pending" || orders[0].Number != 1001 || orders[0].StripeSession != "cs_test_123" || orders[0].Total != 1800 {
		t.Fatalf("pending order: %+v", orders)
	}
	orderID := orders[0].ID
	if captured.Get("metadata[order]") != orderID {
		t.Fatal("the session must carry the order id")
	}
	// Pending orders stay out of the list.
	if strings.Contains(get(t, handler, "/admin/orders").Body.String(), "#1001") {
		t.Fatal("a pending order must not show as an order")
	}

	// Stripe says paid.
	event := `{"type":"checkout.session.completed","data":{"object":{"id":"cs_test_123","payment_intent":"pi_1","payment_status":"paid","amount_subtotal":1300,"amount_total":1800,"currency":"usd",
		"customer_details":{"email":"ana@example.com","name":"Ana Lopez"},
		"shipping_details":{"name":"Ana Lopez","address":{"line1":"1 Mill Lane","city":"Portland","state":"OR","postal_code":"97201","country":"US"}},
		"total_details":{"amount_shipping":500,"amount_tax":0,"amount_discount":0},"metadata":{"order":"` + orderID + `"}}}}`
	if rec := postWebhook(t, handler, event, "whsec_wrong", base); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad signature = %d", rec.Code)
	}
	if rec := postWebhook(t, handler, event, "whsec_xyz", base); rec.Code != http.StatusOK {
		t.Fatalf("webhook = %d %s", rec.Code, rec.Body.String())
	}
	order, _ := host.orders.get(orderID)
	if order.Status != "paid" || order.CustomerEmail != "ana@example.com" || order.ShipTo.City != "Portland" || order.StripePayment != "pi_1" || order.Shipping != 500 {
		t.Fatalf("paid order: %+v", order)
	}
	product, _ := host.products.get(id)
	if product.Stock != 3 {
		t.Fatalf("stock after sale = %d, want 3", product.Stock)
	}
	// A replayed event changes nothing.
	postWebhook(t, handler, event, "whsec_xyz", base)
	product, _ = host.products.get(id)
	if product.Stock != 3 {
		t.Fatal("a replayed webhook must not take stock twice")
	}

	// The thank-you page, and the cart is emptied.
	done := getWithCookie(t, handler, checkoutDonePath+"?session_id=cs_test_123", cookie)
	mustContain(t, done.Body.String(), "your order #1001 is in", "the buyer is thanked with the order number")
	mustContain(t, done.Body.String(), "2 × Sourdough loaf", "and sees what they bought")
	if c := done.Result().Cookies(); len(c) == 0 || c[0].MaxAge != -1 {
		t.Fatal("the cart must be cleared after paying")
	}

	// The owner sees it, sends it, and the dashboard counts it.
	mustContain(t, get(t, handler, "/admin").Body.String(), "Orders to send", "the dashboard counts orders")
	list := get(t, handler, "/admin/orders").Body.String()
	mustContain(t, list, "#1001", "the order is listed")
	mustContain(t, list, "Paid — to send", "as paid and unsent")
	mustContain(t, list, "Ana Lopez", "with the buyer")
	detail := get(t, handler, "/admin/orders/"+orderID).Body.String()
	mustContain(t, detail, "1 Mill Lane, Portland, OR, 97201, US", "with the address")
	mustContain(t, detail, "https://dashboard.stripe.com/payments/pi_1", "and a way to Stripe for refunds")
	mustContain(t, detail, "$18.00", "and the total")
	post(t, handler, "/admin/orders/"+orderID, url.Values{"action": {"fulfil"}})
	mustContain(t, get(t, handler, "/admin/orders").Body.String(), ">Sent<", "marking as sent shows in the list")
	post(t, handler, "/admin/orders/"+orderID, url.Values{"action": {"note"}, "note": {"Tracking RM123"}})
	mustContain(t, get(t, handler, "/admin/orders/"+orderID).Body.String(), "Tracking RM123", "notes are kept")
}

func TestFreeShippingOverThresholdAndStripeFailure(t *testing.T) {
	var captured url.Values
	createStripeSession = func(ctx context.Context, key string, params url.Values) (stripeSession, error) {
		captured = params
		if params.Get("line_items[0][quantity]") == "9" {
			return stripeSession{}, context.DeadlineExceeded
		}
		return stripeSession{ID: "cs_2", URL: "https://checkout.stripe.com/c/pay/cs_2"}, nil
	}
	t.Cleanup(func() { createStripeSession = nil })
	host, handler := newTestHost(t)
	postSettings(t, handler, map[string]string{"title": "W", "currency": "usd", "countVisitors": "on", "stripeSecretKey": "sk_test_abc", "shippingFlat": "5.00", "shippingFreeOver": "50"}, nil)
	id := createProduct(t, handler, "Cake", "30")
	saveProduct(t, handler, id, map[string]string{"name": "Cake", "slug": "cake", "price": "30", "imageCount": "0", "variantCount": "0", "ships": "1", "active": "1"})
	cookie := post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"2"}}).Result().Cookies()[0]
	postWithCookie(t, handler, checkoutPath, url.Values{}, cookie)
	if captured.Get("shipping_options[0][shipping_rate_data][fixed_amount][amount]") != "0" || captured.Get("shipping_options[0][shipping_rate_data][display_name]") != "Free shipping" {
		t.Fatalf("free shipping over 50: %v", captured)
	}
	if captured.Get("automatic_tax[enabled]") != "" {
		t.Fatal("tax must stay off unless switched on")
	}

	cookie = post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"7"}}).Result().Cookies()[0]
	_ = cookie
	nine := post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"9"}}).Result().Cookies()[0]
	rec := postWithCookie(t, handler, checkoutPath, url.Values{}, nine)
	mustContain(t, rec.Header().Get("Location"), "checkout=failed", "a Stripe failure sends the visitor back with a note")
	mustContain(t, getWithCookie(t, handler, rec.Header().Get("Location"), nine).Body.String(), "couldn", "which the cart page shows")
	for _, order := range host.orders.list() {
		if order.Status == "pending" && strings.Contains(order.Note, "") && order.Lines[0].Qty == 9 {
			t.Fatal("a failed checkout must not leave a pending order")
		}
	}
}
