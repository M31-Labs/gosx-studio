package sitehost

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// Mail goes out in the background; tests wait for it briefly.
func awaitMails(t *testing.T, mailer *fakeMailer, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(mailer.snapshot()) >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %d mails, got %d", n, len(mailer.snapshot()))
}

func (f *fakeMailer) snapshot() []Mail {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Mail, len(f.sent))
	copy(out, f.sent)
	return out
}

// postProductWithFile saves the product editor with a file attached.
func postProductWithFile(t *testing.T, handler http.Handler, id string, fields map[string]string, fileName string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("_csrf", csrfToken(t, handler))
	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}
	if fileName != "" {
		part, _ := writer.CreateFormFile("newFile", fileName)
		_, _ = part.Write(data)
	}
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/admin/shop/"+id, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestDigitalProductsEarnDownloadLinks(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	var captured url.Values
	createStripeSession = func(ctx context.Context, key string, params url.Values) (stripeSession, error) {
		captured = params
		return stripeSession{ID: "cs_dl", URL: "https://checkout.stripe.com/c/pay/cs_dl"}, nil
	}
	t.Cleanup(func() { createStripeSession = nil })
	mailer := &fakeMailer{}
	host, err := Open(Options{DataPath: t.TempDir() + "/site.json", SiteTitle: "Wildflower", SiteKind: "shop", Seed: true, NoBackups: true, BaseURL: "https://wildflower.example", Mailer: mailer})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	postSettings(t, handler, map[string]string{"title": "Wildflower", "currency": "usd", "countVisitors": "on", "stripeSecretKey": "sk_test_abc", "stripeWebhookSecret": "whsec_xyz"}, nil)
	if err := host.updateSettingsMetadata(func(m cmsstore.Metadata) { m["contactEmail"] = "owner@example.com" }); err != nil {
		t.Fatal(err)
	}

	id := createProduct(t, handler, "Sourdough recipe book", "9.00")
	rec := postProductWithFile(t, handler, id, map[string]string{"name": "Sourdough recipe book", "slug": "recipe-book", "price": "9.00", "kind": "digital", "imageCount": "0", "variantCount": "0", "active": "1", "save": "1"}, "recipes.pdf", []byte("%PDF-1.4 fake"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("save digital = %d %s", rec.Code, rec.Body.String())
	}
	product, _ := host.products.get(id)
	if product.Kind != kindDigital || product.File == "" || product.FileName != "recipes.pdf" || product.Ships {
		t.Fatalf("digital product: %+v", product)
	}
	mustContain(t, get(t, handler, "/admin/shop/"+id).Body.String(), "Current file: <code>recipes.pdf</code>", "the editor shows the file")
	page := get(t, handler, "/shop/recipe-book").Body.String()
	mustContain(t, page, `>Buy</button>`, "a download says Buy")
	if strings.Contains(page, `name="qty" value="1" min`) {
		t.Fatal("no quantity picker for a download")
	}

	// Buying: no shipping, one copy, and a link once paid.
	cookie := post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"3"}}).Result().Cookies()[0]
	mustContain(t, getWithCookie(t, handler, "/cart", cookie).Body.String(), `name="qty_0" value="1"`, "one copy of a download")
	postWithCookie(t, handler, checkoutPath, url.Values{}, cookie)
	if captured.Get("mode") != "payment" || captured.Get("shipping_options[0][shipping_rate_data][type]") != "" {
		t.Fatalf("download checkout params: %v", captured)
	}
	orderID := captured.Get("metadata[order]")
	event := `{"type":"checkout.session.completed","data":{"object":{"id":"cs_dl","payment_intent":"pi_dl","payment_status":"paid","amount_total":900,"currency":"usd","customer_details":{"email":"ana@example.com","name":"Ana"},"metadata":{"order":"` + orderID + `"}}}}`
	postWebhook(t, handler, event, "whsec_xyz", base)
	done := get(t, handler, checkoutDonePath+"?session_id=cs_dl").Body.String()
	mustContain(t, done, "Your downloads", "the thank-you page offers the download")
	start := strings.Index(done, "/download/")
	link := done[start:]
	link = link[:strings.Index(link, `"`)]
	dl := get(t, handler, link)
	if dl.Code != http.StatusOK || dl.Header().Get("Content-Disposition") != `attachment; filename="recipes.pdf"` || dl.Body.String() != "%PDF-1.4 fake" {
		t.Fatalf("download = %d %q %q", dl.Code, dl.Header().Get("Content-Disposition"), dl.Body.String())
	}
	if code := get(t, handler, link[:len(link)-3]+"xyz").Code; code != http.StatusNotFound {
		t.Fatal("a tampered link must fail")
	}
	timeNow = func() time.Time { return base.Add(8 * 24 * time.Hour) }
	if code := get(t, handler, link).Code; code != http.StatusNotFound {
		t.Fatal("an expired link must fail")
	}
	timeNow = func() time.Time { return base }
	// The buyer's receipt carries the link; the owner gets the order mail.
	awaitMails(t, mailer, 2)
	var receipt, ownerMail bool
	for _, m := range mailer.snapshot() {
		if m.To == "ana@example.com" && strings.Contains(m.Text, "/download/") {
			receipt = true
		}
		if strings.Contains(m.Subject, "Order #") {
			ownerMail = true
		}
	}
	if !receipt || !ownerMail {
		t.Fatalf("mails: receipt=%v owner=%v (%d sent)", receipt, ownerMail, len(mailer.sent))
	}
	mustContain(t, get(t, handler, "/admin/orders/"+orderID).Body.String(), "Downloads the buyer received", "the order shows the downloads")
}

func TestSubscriptionsCheckOutInSubscriptionModeAndFollowStripe(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	var captured url.Values
	createStripeSession = func(ctx context.Context, key string, params url.Values) (stripeSession, error) {
		captured = params
		return stripeSession{ID: "cs_sub", URL: "https://checkout.stripe.com/c/pay/cs_sub"}, nil
	}
	t.Cleanup(func() { createStripeSession = nil })
	host, handler := newTestHost(t)
	postSettings(t, handler, map[string]string{"title": "W", "currency": "usd", "countVisitors": "on", "stripeSecretKey": "sk_test_abc", "stripeWebhookSecret": "whsec_xyz"}, nil)
	id := createProduct(t, handler, "Bread club", "25")
	saveProduct(t, handler, id, map[string]string{"name": "Bread club", "slug": "bread-club", "price": "25", "kind": "subscription", "interval": "month", "imageCount": "0", "variantCount": "0", "active": "1"})
	product, _ := host.products.get(id)
	if product.Kind != kindSubscription || product.Interval != "month" {
		t.Fatalf("subscription product: %+v", product)
	}
	mustContain(t, get(t, handler, "/shop/bread-club").Body.String(), "$25.00 / month", "the price says how often")
	mustContain(t, get(t, handler, "/shop/bread-club").Body.String(), ">Subscribe</button>", "and the button says Subscribe")

	cookie := post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"1"}}).Result().Cookies()[0]
	postWithCookie(t, handler, checkoutPath, url.Values{}, cookie)
	if captured.Get("mode") != "subscription" || captured.Get("line_items[0][price_data][recurring][interval]") != "month" {
		t.Fatalf("subscription params: %v", captured)
	}
	orderID := captured.Get("metadata[order]")
	postWebhook(t, handler, `{"type":"checkout.session.completed","data":{"object":{"id":"cs_sub","mode":"subscription","subscription":"sub_1","customer":"cus_1","payment_status":"paid","amount_total":2500,"currency":"usd","customer_details":{"email":"ana@example.com","name":"Ana"},"metadata":{"order":"`+orderID+`"}}}}`, "whsec_xyz", base)
	order, _ := host.orders.get(orderID)
	if order.Status != "paid" || order.StripeSubscription != "sub_1" || order.SubscriptionStatus != "active" || order.StripeCustomer != "cus_1" {
		t.Fatalf("subscription order: %+v", order)
	}
	mustContain(t, get(t, handler, "/admin/orders/"+orderID).Body.String(), "Active subscription", "the order shows the subscription")

	postWebhook(t, handler, `{"type":"invoice.paid","data":{"object":{"id":"in_2","subscription":"sub_1","billing_reason":"subscription_cycle"}}}`, "whsec_xyz", base)
	order, _ = host.orders.get(orderID)
	if order.Renewals != 1 {
		t.Fatalf("renewals = %d", order.Renewals)
	}
	postWebhook(t, handler, `{"type":"customer.subscription.deleted","data":{"object":{"id":"sub_1","status":"canceled"}}}`, "whsec_xyz", base)
	order, _ = host.orders.get(orderID)
	if order.SubscriptionStatus != "cancelled" {
		t.Fatalf("after cancel: %+v", order)
	}
	mustContain(t, get(t, handler, "/admin/orders/"+orderID).Body.String(), "Cancelled subscription", "and its cancellation")
}

func TestBookingsOfferSlotsAndFillThem(t *testing.T) {
	// A Tuesday at 08:00 local.
	base := time.Date(2026, 9, 15, 8, 0, 0, 0, time.Local)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	mailer := &fakeMailer{}
	host, err := Open(Options{DataPath: t.TempDir() + "/site.json", SiteTitle: "Wildflower", SiteKind: "services", Seed: true, NoBackups: true, BaseURL: "https://wildflower.example", Mailer: mailer})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	if err := host.updateSettingsMetadata(func(m cmsstore.Metadata) { m["contactEmail"] = "owner@example.com" }); err != nil {
		t.Fatal(err)
	}
	id := createProduct(t, handler, "Bread class", "0")
	saveProduct(t, handler, id, map[string]string{"name": "Bread class", "slug": "bread-class", "price": "0", "kind": "booking", "imageCount": "0", "variantCount": "0", "active": "1",
		"bookDay": "2", "bookStart": "10:00", "bookEnd": "12:00", "bookSlot": "60", "bookLead": "1", "bookWeeks": "4"})
	product, _ := host.products.get(id)
	if product.Kind != kindBooking || product.Booking.SlotMinutes != 60 || len(product.Booking.Days) != 1 || product.Booking.Days[0] != 2 {
		t.Fatalf("booking product: %+v", product.Booking)
	}
	// Today (Tuesday) from 10:00: two slots; Wednesday: none; tomorrow's Tuesday next week too.
	slots := host.availableSlots(product, base)
	if len(slots) != 2 || slots[0].Format("15:04") != "10:00" || slots[1].Format("15:04") != "11:00" {
		t.Fatalf("slots today = %v", slots)
	}
	if len(host.availableSlots(product, base.Add(24*time.Hour))) != 0 {
		t.Fatal("Wednesday must have no slots")
	}
	page := get(t, handler, "/shop/bread-class?date="+base.Format("2006-01-02")).Body.String()
	mustContain(t, page, `value="`+slots[0].UTC().Format(time.RFC3339)+`"`, "the page offers the first slot")
	mustContain(t, page, ">Book</button>", "a free class is booked directly")
	mustContain(t, page, `name="email"`, "with a name and email")

	// Book the 10:00 for free.
	rec := post(t, handler, "/book/"+id, url.Values{"slot": {slots[0].UTC().Format(time.RFC3339)}, "name": {"Ana"}, "email": {"ana@example.com"}})
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(rec.Header().Get("Location"), "/book/done?id=") {
		t.Fatalf("free booking = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	mustContain(t, get(t, handler, rec.Header().Get("Location")).Body.String(), "you&#39;re booked for Bread class", "the visitor is thanked")
	if remaining := host.availableSlots(product, base); len(remaining) != 1 || remaining[0].Format("15:04") != "11:00" {
		t.Fatalf("after booking: %v", remaining)
	}
	// The same slot again is refused.
	rec = post(t, handler, "/book/"+id, url.Values{"slot": {slots[0].UTC().Format(time.RFC3339)}, "name": {"Sam"}, "email": {"sam@example.com"}})
	mustContain(t, rec.Header().Get("Location"), "problem=slot", "a taken slot is refused")
	awaitMails(t, mailer, 2)

	// The admin sees it by day, and cancelling frees the time.
	bookings := get(t, handler, "/admin/bookings").Body.String()
	mustContain(t, bookings, "Tuesday 15 September", "grouped by day")
	mustContain(t, bookings, "10:00–11:00", "with the time")
	mustContain(t, bookings, "Ana ana@example.com", "and who")
	booking := host.bookings.list()[0]
	post(t, handler, "/admin/bookings/"+booking.ID, url.Values{"action": {"cancel"}})
	if remaining := host.availableSlots(product, base); len(remaining) != 2 {
		t.Fatal("cancelling must free the slot")
	}

	// A paid class goes through the cart and is booked when paid.
	var captured url.Values
	createStripeSession = func(ctx context.Context, key string, params url.Values) (stripeSession, error) {
		captured = params
		return stripeSession{ID: "cs_book", URL: "https://checkout.stripe.com/c/pay/cs_book"}, nil
	}
	t.Cleanup(func() { createStripeSession = nil })
	postSettings(t, handler, map[string]string{"title": "Wildflower", "currency": "usd", "countVisitors": "on", "stripeSecretKey": "sk_test_abc", "stripeWebhookSecret": "whsec_xyz"}, nil)
	saveProduct(t, handler, id, map[string]string{"name": "Bread class", "slug": "bread-class", "price": "40", "kind": "booking", "imageCount": "0", "variantCount": "0", "active": "1",
		"bookDay": "2", "bookStart": "10:00", "bookEnd": "12:00", "bookSlot": "60", "bookLead": "1", "bookWeeks": "4"})
	slot := slots[1].UTC().Format(time.RFC3339)
	rec = post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"1"}, "slot": {slot}})
	cookie := rec.Result().Cookies()[0]
	cart := getWithCookie(t, handler, "/cart", cookie).Body.String()
	mustContain(t, cart, "Bread class — "+slotLabel(slot), "the cart names the time")
	postWithCookie(t, handler, checkoutPath, url.Values{}, cookie)
	if !strings.Contains(captured.Get("line_items[0][price_data][product_data][name]"), slotLabel(slot)) {
		t.Fatalf("checkout line: %v", captured)
	}
	orderID := captured.Get("metadata[order]")
	postWebhook(t, handler, `{"type":"checkout.session.completed","data":{"object":{"id":"cs_book","payment_intent":"pi_b","payment_status":"paid","amount_total":4000,"currency":"usd","customer_details":{"email":"sam@example.com","name":"Sam"},"metadata":{"order":"`+orderID+`"}}}}`, "whsec_xyz", base)
	live := 0
	for _, b := range host.bookings.list() {
		if b.Status == "booked" && b.Order == orderID && b.CustomerName == "Sam" {
			live++
		}
	}
	if live != 1 {
		t.Fatalf("paid booking rows = %d", live)
	}
	mustContain(t, get(t, handler, checkoutDonePath+"?session_id=cs_book").Body.String(), "Your booking", "the thank-you confirms the booking")
	if len(host.availableSlots(product, base)) != 1 {
		t.Fatal("the paid slot is taken")
	}
	_ = strconv.Itoa
}
