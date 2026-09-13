package sitehost

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// paidOrderFor puts one paid order under an email through the normal
// checkout and webhook path.
func paidOrderFor(t *testing.T, host *Host, handler http.Handler, productID, session, email string, at time.Time) string {
	t.Helper()
	cookie := post(t, handler, "/cart/add", url.Values{"product": {productID}, "qty": {"1"}}).Result().Cookies()[0]
	postWithCookie(t, handler, checkoutPath, url.Values{}, cookie)
	var orderID string
	for _, order := range host.orders.list() {
		if order.StripeSession == session {
			orderID = order.ID
		}
	}
	if orderID == "" {
		t.Fatalf("no pending order for %s", session)
	}
	postWebhook(t, handler, `{"type":"checkout.session.completed","data":{"object":{"id":"`+session+`","mode":"subscription","subscription":"sub_`+session+`","customer":"cus_`+session+`","payment_status":"paid","amount_total":2500,"currency":"usd","customer_details":{"email":"`+email+`","name":"Ana"},"metadata":{"order":"`+orderID+`"}}}}`, "whsec_xyz", at)
	return orderID
}

func TestCustomersSignInByEmailLinkAndSeeTheirOrders(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	sessions := []string{"cs_a", "cs_b"}
	createStripeSession = func(ctx context.Context, key string, params url.Values) (stripeSession, error) {
		id := sessions[0]
		sessions = sessions[1:]
		return stripeSession{ID: id, URL: "https://checkout.stripe.com/c/pay/" + id}, nil
	}
	t.Cleanup(func() { createStripeSession = nil })
	var portalCustomer string
	createStripePortal = func(ctx context.Context, secretKey, customer, returnURL string) (string, error) {
		portalCustomer = customer
		return "https://billing.stripe.com/p/session/xyz", nil
	}
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
	id := createProduct(t, handler, "Bread club", "25")
	saveProduct(t, handler, id, map[string]string{"name": "Bread club", "slug": "bread-club", "price": "25", "kind": "subscription", "interval": "month", "imageCount": "0", "variantCount": "0", "active": "1"})
	mine := paidOrderFor(t, host, handler, id, "cs_a", "Ana@Example.com", base)
	paidOrderFor(t, host, handler, id, "cs_b", "someone-else@example.com", base)
	awaitMails(t, mailer, 4) // two receipts, two owner mails

	mustContain(t, get(t, handler, "/").Body.String(), `href="/account">Your orders</a>`, "the footer links to the orders page when the shop is on")
	page := get(t, handler, "/account").Body.String()
	mustContain(t, page, "Send me a link", "the orders page asks for an email")
	if strings.Contains(page, csrfMetaName) {
		t.Fatal("no CSRF token on a public page")
	}

	// Asking for a link: same answer for a stranger and a buyer.
	rec := post(t, handler, "/account", url.Values{"email": {"nobody@example.com"}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/account?sent=1" {
		t.Fatalf("stranger = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	time.Sleep(50 * time.Millisecond)
	if mailer.count() != 4 {
		t.Fatalf("a stranger must not receive mail, got %d", mailer.count())
	}
	rec = post(t, handler, "/account", url.Values{"email": {"ana@example.com"}})
	if rec.Header().Get("Location") != "/account?sent=1" {
		t.Fatalf("buyer = %q", rec.Header().Get("Location"))
	}
	awaitMails(t, mailer, 5)
	linkMail := mailer.snapshot()[4]
	if linkMail.To != "ana@example.com" || !strings.Contains(linkMail.Text, "https://wildflower.example/account/in/") {
		t.Fatalf("link mail: %+v", linkMail)
	}
	link := linkMail.Text[strings.Index(linkMail.Text, "/account/in/"):]
	link = strings.Fields(link)[0]

	// The link signs in and shows only Ana's orders.
	rec = get(t, handler, link)
	cookie := cookieNamed(rec, customerCookie)
	if rec.Code != http.StatusSeeOther || cookie == nil || !cookie.HttpOnly {
		t.Fatalf("enter = %d cookie=%v", rec.Code, cookie)
	}
	orders := getWithCookie(t, handler, "/account", cookie).Body.String()
	mustContain(t, orders, "Signed in as ana@example.com", "the page names the customer")
	mine1, _ := host.orders.get(mine)
	mustContain(t, orders, "Order "+mine1.label(), "and lists her order")
	mustContain(t, orders, "Subscription active", "with its subscription state")
	mustContain(t, orders, "Manage subscription", "and a way to manage it")
	if strings.Count(orders, "site-post-card__title") != 1 {
		t.Fatal("another customer's order must not show")
	}

	// Managing the subscription opens Stripe's portal for that customer.
	rec = postWithCookie(t, handler, "/account/portal", url.Values{"order": {mine}}, cookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "https://billing.stripe.com/p/session/xyz" || portalCustomer != "cus_cs_a" {
		t.Fatalf("portal = %d %q customer=%q", rec.Code, rec.Header().Get("Location"), portalCustomer)
	}
	var other string
	for _, order := range host.orders.list() {
		if order.ID != mine && order.Status == "paid" {
			other = order.ID
		}
	}
	portalCustomer = ""
	rec = postWithCookie(t, handler, "/account/portal", url.Values{"order": {other}}, cookie)
	if rec.Header().Get("Location") != "/account" || portalCustomer != "" {
		t.Fatal("someone else's subscription must not open")
	}

	// Links expire; sign-in cookies expire later.
	timeNow = func() time.Time { return base.Add(31 * time.Minute) }
	rec = get(t, handler, link)
	if rec.Header().Get("Location") != "/account?sent=expired" {
		t.Fatalf("stale link = %q", rec.Header().Get("Location"))
	}
	mustContain(t, getWithCookie(t, handler, "/account", cookie).Body.String(), "Signed in as", "the session outlives the link")
	timeNow = func() time.Time { return base.Add(31 * 24 * time.Hour) }
	mustContain(t, getWithCookie(t, handler, "/account", cookie).Body.String(), "Send me a link", "and ends after a month")
	timeNow = func() time.Time { return base }
	rec = postWithCookie(t, handler, "/account/out", url.Values{}, cookie)
	if out := cookieNamed(rec, customerCookie); out == nil || out.MaxAge != -1 {
		t.Fatal("sign out must clear the cookie")
	}
}

func TestSavedCartsGetOneReminderWithARestoreLink(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	mailer := &fakeMailer{}
	host, err := Open(Options{DataPath: t.TempDir() + "/site.json", SiteTitle: "Wildflower", SiteKind: "shop", Seed: true, NoBackups: true, BaseURL: "https://wildflower.example", Mailer: mailer})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	postSettings(t, handler, map[string]string{"title": "Wildflower", "currency": "usd", "countVisitors": "on", "stripeSecretKey": "sk_test_abc"}, nil)
	id := createProduct(t, handler, "Sourdough loaf", "6.50")
	saveProduct(t, handler, id, map[string]string{"name": "Sourdough loaf", "slug": "sourdough", "price": "6.50", "imageCount": "0", "variantCount": "0", "active": "1"})

	cookie := post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"2"}}).Result().Cookies()[0]
	mustContain(t, getWithCookie(t, handler, "/cart", cookie).Body.String(), "Email me a link to this cart", "the cart offers to save itself")
	rec := postWithCookie(t, handler, "/cart/email", url.Values{"email": {"ana@example.com"}}, cookie)
	if rec.Header().Get("Location") != "/cart?saved=1" {
		t.Fatalf("save = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	mustContain(t, getWithCookie(t, handler, "/cart?saved=1", cookie).Body.String(), "one reminder", "and says one reminder will come")
	if carts := host.carts.list(); len(carts) != 1 || carts[0].Email != "ana@example.com" || len(carts[0].Lines) != 1 || carts[0].Lines[0].Qty != 2 {
		t.Fatalf("saved carts: %+v", carts)
	}
	// Saving again replaces, never duplicates.
	postWithCookie(t, handler, "/cart/email", url.Values{"email": {"ana@example.com"}}, cookie)
	if len(host.carts.list()) != 1 {
		t.Fatal("one cart per email")
	}

	// Too early: nothing. After two hours: one mail, then never again.
	host.remindCarts()
	if mailer.count() != 0 {
		t.Fatal("no reminder within two hours")
	}
	timeNow = func() time.Time { return base.Add(3 * time.Hour) }
	host.remindCarts()
	awaitMails(t, mailer, 1)
	reminder := mailer.snapshot()[0]
	if reminder.To != "ana@example.com" || !strings.Contains(reminder.Text, "2 × Sourdough loaf") || !strings.Contains(reminder.Text, "https://wildflower.example/cart/restore/") {
		t.Fatalf("reminder: %+v", reminder)
	}
	host.remindCarts()
	time.Sleep(50 * time.Millisecond)
	if mailer.count() != 1 {
		t.Fatalf("only one reminder, got %d", mailer.count())
	}

	// The link puts the cart back for a fresh browser.
	link := reminder.Text[strings.Index(reminder.Text, "/cart/restore/"):]
	link = strings.Fields(link)[0]
	rec = get(t, handler, link)
	restored := cookieNamed(rec, cartCookie)
	if rec.Code != http.StatusSeeOther || restored == nil {
		t.Fatalf("restore = %d cookie=%v", rec.Code, restored)
	}
	mustContain(t, getWithCookie(t, handler, "/cart", restored).Body.String(), `name="qty_0" value="2"`, "the restored cart has the loaves")
	if rec := get(t, handler, link[:len(link)-2]+"zz"); cookieNamed(rec, cartCookie) != nil {
		t.Fatal("a tampered restore link must not set a cart")
	}

	// A bad address is refused; a bot field is honoured.
	rec = postWithCookie(t, handler, "/cart/email", url.Values{"email": {"not-an-email"}}, cookie)
	if rec.Header().Get("Location") != "/cart?saved=no" {
		t.Fatalf("bad email = %q", rec.Header().Get("Location"))
	}
}

func TestSavedCartsAreForgottenAfterAPurchase(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	createStripeSession = func(ctx context.Context, key string, params url.Values) (stripeSession, error) {
		return stripeSession{ID: "cs_buy", URL: "https://checkout.stripe.com/c/pay/cs_buy"}, nil
	}
	t.Cleanup(func() { createStripeSession = nil })
	mailer := &fakeMailer{}
	host, err := Open(Options{DataPath: t.TempDir() + "/site.json", SiteTitle: "Wildflower", SiteKind: "shop", Seed: true, NoBackups: true, BaseURL: "https://wildflower.example", Mailer: mailer})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	postSettings(t, handler, map[string]string{"title": "Wildflower", "currency": "usd", "countVisitors": "on", "stripeSecretKey": "sk_test_abc", "stripeWebhookSecret": "whsec_xyz"}, nil)
	id := createProduct(t, handler, "Sourdough loaf", "6.50")
	saveProduct(t, handler, id, map[string]string{"name": "Sourdough loaf", "slug": "sourdough", "price": "6.50", "imageCount": "0", "variantCount": "0", "active": "1"})
	cookie := post(t, handler, "/cart/add", url.Values{"product": {id}, "qty": {"1"}}).Result().Cookies()[0]
	postWithCookie(t, handler, "/cart/email", url.Values{"email": {"ana@example.com"}}, cookie)
	postWithCookie(t, handler, checkoutPath, url.Values{}, cookie)
	var orderID string
	for _, order := range host.orders.list() {
		orderID = order.ID
	}
	postWebhook(t, handler, `{"type":"checkout.session.completed","data":{"object":{"id":"cs_buy","payment_intent":"pi_1","payment_status":"paid","amount_total":650,"currency":"usd","customer_details":{"email":"ana@example.com","name":"Ana"},"metadata":{"order":"`+orderID+`"}}}}`, "whsec_xyz", base)
	if len(host.carts.list()) != 0 {
		t.Fatal("a purchase forgets the saved cart")
	}
	timeNow = func() time.Time { return base.Add(3 * time.Hour) }
	host.remindCarts()
	time.Sleep(50 * time.Millisecond)
	for _, m := range mailer.snapshot() {
		if strings.Contains(m.Subject, "Still thinking") {
			t.Fatal("no reminder after a purchase")
		}
	}
}
