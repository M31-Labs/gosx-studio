package sitehost

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// checkout.go takes the money: Stripe Checkout, a signed webhook, and the
// orders it produces.
//
// Stripe's hosted Checkout page is the whole payment surface. The site never
// sees a card number; it builds a session from the cart with prices looked
// up fresh, sends the visitor to Stripe, and learns the outcome from a
// webhook whose signature it checks. An order exists from the moment the
// visitor leaves for Stripe (pending) and becomes real when the webhook says
// paid, so a refresh, a double click, or a replayed event cannot make two.

const (
	stripeWebhookSecretKey = "stripeWebhookSecret"
	shippingFlatKey        = "shippingFlat"
	shippingFreeOverKey    = "shippingFreeOver"
	shipCountriesKey       = "shipCountries"
	stripeTaxKey           = "stripeTax"
	allowPromosKey         = "allowPromos"
	stripeWebhookPath      = "/stripe/webhook"
	checkoutDonePath       = "/checkout/done"
	pendingOrderTTL        = 24 * time.Hour
	stripeAPI              = "https://api.stripe.com/v1/checkout/sessions"
)

// OrderLine is one thing in an order, priced as it was when bought.
type OrderLine struct {
	Product  string `json:"product"`
	Variant  string `json:"variant,omitempty"`
	Name     string `json:"name"`
	Qty      int    `json:"qty"`
	Unit     int64  `json:"unit"`
	Total    int64  `json:"total"`
	Kind     string `json:"kind,omitempty"`
	File     string `json:"file,omitempty"`     // digital: stored file
	FileName string `json:"fileName,omitempty"` // digital: the buyer's name for it
	Slot     string `json:"slot,omitempty"`     // bookings: start time, RFC 3339
}

// Address is where an order goes.
type Address struct {
	Name    string `json:"name,omitempty"`
	Line1   string `json:"line1,omitempty"`
	Line2   string `json:"line2,omitempty"`
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	Postal  string `json:"postal,omitempty"`
	Country string `json:"country,omitempty"`
}

// Order is one purchase, from pending through paid to fulfilled.
type Order struct {
	ID            string      `json:"id"`
	Number        int         `json:"number"`
	Status        string      `json:"status"` // pending, paid, fulfilled, cancelled
	Lines         []OrderLine `json:"lines"`
	Subtotal      int64       `json:"subtotal"`
	Shipping      int64       `json:"shipping"`
	Tax           int64       `json:"tax"`
	Total         int64       `json:"total"`
	Currency      string      `json:"currency"`
	CustomerName  string      `json:"customerName,omitempty"`
	CustomerEmail string      `json:"customerEmail,omitempty"`
	ShipTo        Address     `json:"shipTo,omitempty"`
	Ships         bool        `json:"ships"`
	StripeSession string      `json:"stripeSession,omitempty"`
	StripePayment string      `json:"stripePayment,omitempty"`
	// Subscriptions, when the order started one.
	StripeSubscription string     `json:"stripeSubscription,omitempty"`
	StripeCustomer     string     `json:"stripeCustomer,omitempty"`
	SubscriptionStatus string     `json:"subscriptionStatus,omitempty"` // active, cancelled, past_due
	Renewals           int        `json:"renewals,omitempty"`
	LastPaid           *time.Time `json:"lastPaid,omitempty"`
	CartEmail          string     `json:"cartEmail,omitempty"` // the visitor's own address, for a reminder
	Reminded           bool       `json:"reminded,omitempty"`
	Note               string     `json:"note,omitempty"`
	Created            time.Time  `json:"created"`
	Paid               *time.Time `json:"paid,omitempty"`
	Fulfilled          *time.Time `json:"fulfilled,omitempty"`
}

func (o Order) label() string { return "#" + strconv.Itoa(o.Number) }

// ---------- the order store ----------

type orderStore struct {
	mu     sync.Mutex
	path   string
	loaded bool
	orders []Order
	next   int
}

func newOrderStore(path string) *orderStore { return &orderStore{path: path} }

func (o Options) ordersPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "orders.json")
}

func (s *orderStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	s.next = 1001
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file struct {
		Next   int     `json:"next"`
		Orders []Order `json:"orders"`
	}
	if json.Unmarshal(raw, &file) == nil {
		s.orders = file.Orders
		if file.Next > s.next {
			s.next = file.Next
		}
	}
}

func (s *orderStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(struct {
		Next   int     `json:"next"`
		Orders []Order `json:"orders"`
	}{s.next, s.orders}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, s.path)
}

func (s *orderStore) list() []Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]Order, len(s.orders))
	copy(out, s.orders)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

func (s *orderStore) get(id string) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, order := range s.orders {
		if order.ID == id {
			return order, true
		}
	}
	return Order{}, false
}

func (s *orderStore) bySession(session string) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, order := range s.orders {
		if session != "" && order.StripeSession == session {
			return order, true
		}
	}
	return Order{}, false
}

var errOrderNotFound = errors.New("order not found")

// create adds a pending order and gives it the next number. Pending orders
// nobody paid for are dropped after a day so they never clutter the list.
func (s *orderStore) create(order Order) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	now := timeNow().UTC()
	kept := s.orders[:0]
	for _, existing := range s.orders {
		if existing.Status == "pending" && now.Sub(existing.Created) > pendingOrderTTL {
			continue
		}
		kept = append(kept, existing)
	}
	s.orders = kept
	order.ID = "o" + randomHex(5)
	order.Number = s.next
	s.next++
	order.Status = "pending"
	order.Created = now
	s.orders = append(s.orders, order)
	return order, s.saveLocked()
}

// update replaces an order, keeping its number and creation time.
func (s *orderStore) update(order Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index, existing := range s.orders {
		if existing.ID == order.ID {
			order.Number, order.Created = existing.Number, existing.Created
			s.orders[index] = order
			return s.saveLocked()
		}
	}
	return errOrderNotFound
}

// ---------- settings ----------

type paymentSettings struct {
	SecretKey     string
	WebhookSecret string
	ShippingFlat  int64
	FreeOver      int64
	Countries     []string
	Tax           bool
	Promos        bool
}

func (h *Host) payments() paymentSettings {
	meta := h.settings().Metadata
	flat, _ := strconv.ParseInt(strings.TrimSpace(meta[shippingFlatKey]), 10, 64)
	free, _ := strconv.ParseInt(strings.TrimSpace(meta[shippingFreeOverKey]), 10, 64)
	return paymentSettings{
		SecretKey:     strings.TrimSpace(meta[stripeSecretKey]),
		WebhookSecret: strings.TrimSpace(meta[stripeWebhookSecretKey]),
		ShippingFlat:  flat,
		FreeOver:      free,
		Countries:     parseCountries(firstNonEmpty(meta[shipCountriesKey], defaultCountries(normalizeCurrency(meta[currencyKey])))),
		Tax:           meta[stripeTaxKey] == "true",
		Promos:        meta[allowPromosKey] != "false",
	}
}

func defaultCountries(currency string) string {
	switch currency {
	case "gbp":
		return "GB"
	case "eur":
		return "DE, FR, ES, IT, NL, BE, AT, IE, PT, FI"
	case "cad":
		return "CA"
	case "aud":
		return "AU"
	case "nzd":
		return "NZ"
	case "jpy":
		return "JP"
	case "chf":
		return "CH"
	case "sek":
		return "SE"
	case "dkk":
		return "DK"
	case "nok":
		return "NO"
	case "pln":
		return "PL"
	case "inr":
		return "IN"
	case "brl":
		return "BR"
	case "mxn":
		return "MX"
	case "sgd":
		return "SG"
	case "hkd":
		return "HK"
	case "zar":
		return "ZA"
	case "krw":
		return "KR"
	default:
		return "US"
	}
}

func parseCountries(raw string) []string {
	out := make([]string, 0, 8)
	seen := map[string]bool{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		code := strings.ToUpper(strings.TrimSpace(part))
		if len(code) != 2 || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	return out
}

func (h *Host) renderPaymentFields(settings cmsstore.SiteSettings, base string) gosx.Node {
	pay := h.payments()
	currency := normalizeCurrency(settings.Metadata[currencyKey])
	keyHint := "Starts with sk_live_ or sk_test_. Find it under Developers → API keys in your Stripe dashboard. Stored on this server only."
	state, label := "draft", "Not connected"
	if pay.SecretKey != "" {
		state, label = "published", "Connected"
		if strings.HasPrefix(pay.SecretKey, "sk_test_") {
			label = "Connected (test mode)"
		}
	}
	check := func(name string, on bool, text string) gosx.Node {
		attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", name), gosx.Attr("value", "true")}
		if on {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		return gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")), gosx.El("input", gosx.Attrs(attrs...)), gosx.Text(" "+text))
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead"), gosx.Attr("id", "payments")), gosx.Text("Payments"),
			gosx.Text(" "), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
			gosx.Text("Payments go through Stripe. Visitors pay on Stripe's secure page and come back here; card details never touch this site. Create a free Stripe account at stripe.com, then paste two keys.")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "stripeSecretKey")), gosx.Text("Stripe secret key")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "password"), gosx.Attr("id", "stripeSecretKey"), gosx.Attr("name", "stripeSecretKey"), gosx.Attr("value", pay.SecretKey), gosx.Attr("autocomplete", "off"))),
			gosx.El("small", nil, gosx.Text(keyHint)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "stripeWebhookSecret")), gosx.Text("Stripe webhook signing secret")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "password"), gosx.Attr("id", "stripeWebhookSecret"), gosx.Attr("name", "stripeWebhookSecret"), gosx.Attr("value", pay.WebhookSecret), gosx.Attr("autocomplete", "off"))),
			gosx.El("small", nil, gosx.Text("In Stripe, under Developers → Webhooks, add an endpoint for "), gosx.El("code", nil, gosx.Text(base+stripeWebhookPath)),
				gosx.Text(" that sends the event checkout.session.completed, then paste its signing secret (starts with whsec_). This is how the site learns an order was paid.")),
		),
		adminTextField("shippingFlat", "Shipping charge", moneyInput(pay.ShippingFlat, currency), "A flat amount added to every order that needs shipping, in "+strings.ToUpper(currency)+". Leave blank for free shipping."),
		adminTextField("shippingFreeOver", "Free shipping over", moneyInput(pay.FreeOver, currency), "Orders at or above this amount ship free. Leave blank to always charge."),
		adminTextField("shipCountries", "Ship to", strings.Join(pay.Countries, ", "), "Two-letter country codes, separated by commas. Stripe asks buyers for an address in one of these."),
		check("stripeTax", pay.Tax, "Work out sales tax or VAT automatically (needs Stripe Tax turned on in your Stripe account)"),
		check("allowPromos", pay.Promos, "Let buyers enter discount codes (create the codes as coupons in Stripe)"),
	)
}

func applyPaymentFields(r *http.Request, metadata cmsstore.Metadata, currency string) string {
	set := func(key, value string) {
		if value = strings.TrimSpace(value); value != "" {
			metadata[key] = value
		} else {
			delete(metadata, key)
		}
	}
	secret := strings.TrimSpace(r.PostFormValue("stripeSecretKey"))
	if secret != "" && !strings.HasPrefix(secret, "sk_") && !strings.HasPrefix(secret, "rk_") {
		return "That doesn't look like a Stripe secret key. It starts with sk_live_ or sk_test_."
	}
	set(stripeSecretKey, secret)
	set(stripeWebhookSecretKey, r.PostFormValue("stripeWebhookSecret"))
	flat, err := parseMoney(r.PostFormValue("shippingFlat"), currency)
	if err != nil {
		return "The shipping charge doesn't look like an amount."
	}
	free, err := parseMoney(r.PostFormValue("shippingFreeOver"), currency)
	if err != nil {
		return "The free-shipping amount doesn't look like an amount."
	}
	set(shippingFlatKey, strconv.FormatInt(flat, 10))
	set(shippingFreeOverKey, strconv.FormatInt(free, 10))
	if flat == 0 {
		delete(metadata, shippingFlatKey)
	}
	if free == 0 {
		delete(metadata, shippingFreeOverKey)
	}
	set(shipCountriesKey, strings.Join(parseCountries(r.PostFormValue("shipCountries")), ", "))
	if r.PostFormValue("stripeTax") == "true" {
		metadata[stripeTaxKey] = "true"
	} else {
		delete(metadata, stripeTaxKey)
	}
	if r.PostFormValue("allowPromos") == "true" {
		delete(metadata, allowPromosKey)
	} else {
		metadata[allowPromosKey] = "false"
	}
	return ""
}

// ---------- Stripe ----------

// stripeSession is what Checkout gives back: where to send the visitor.
type stripeSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// createStripeSession posts to Stripe. Tests replace it.
var createStripeSession = func(ctx context.Context, secretKey string, params url.Values) (stripeSession, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stripeAPI, strings.NewReader(params.Encode()))
	if err != nil {
		return stripeSession{}, err
	}
	req.Header.Set("Authorization", "Bearer "+secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", "2024-06-20")
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return stripeSession{}, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		var failure struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &failure)
		return stripeSession{}, errors.New("stripe: " + firstNonEmpty(failure.Error.Message, res.Status))
	}
	var session stripeSession
	if err := json.Unmarshal(body, &session); err != nil || session.URL == "" {
		return stripeSession{}, errors.New("stripe: unexpected reply")
	}
	return session, nil
}

func (h *Host) mountCheckout(mux *http.ServeMux) {
	mux.HandleFunc("POST "+checkoutPath, h.handleCheckout)
	mux.HandleFunc("GET "+checkoutDonePath, h.handleCheckoutDone)
	mux.HandleFunc("POST "+stripeWebhookPath, h.handleStripeWebhook)
	mux.HandleFunc("GET /admin/orders", h.handleAdminOrders)
	mux.HandleFunc("GET /admin/orders/{$}", h.handleAdminOrders)
	mux.HandleFunc("GET /admin/orders/{id}", h.handleAdminOrder)
	mux.HandleFunc("POST /admin/orders/{id}", h.handleAdminOrderAction)
}

// shippingFor is the shipping charge for a cart under the owner's rules.
func (pay paymentSettings) shippingFor(subtotal int64, ships bool) int64 {
	if !ships || pay.ShippingFlat == 0 {
		return 0
	}
	if pay.FreeOver > 0 && subtotal >= pay.FreeOver {
		return 0
	}
	return pay.ShippingFlat
}

func (h *Host) handleCheckout(w http.ResponseWriter, r *http.Request) {
	pay := h.payments()
	if pay.SecretKey == "" {
		http.Redirect(w, r, cartPath, http.StatusSeeOther)
		return
	}
	resolved, subtotal := h.resolveCart(readCart(r))
	if len(resolved) == 0 {
		http.Redirect(w, r, cartPath, http.StatusSeeOther)
		return
	}
	currency := h.currency()
	base := h.absoluteBase(r)

	order := Order{Currency: currency, Subtotal: subtotal}
	mode := "payment"
	for _, line := range resolved {
		if line.Product.Kind == kindSubscription {
			mode = "subscription"
		}
	}
	params := url.Values{}
	params.Set("mode", mode)
	params.Set("success_url", base+checkoutDonePath+"?session_id={CHECKOUT_SESSION_ID}")
	params.Set("cancel_url", base+cartPath)
	for index, line := range resolved {
		prefix := "line_items[" + strconv.Itoa(index) + "]"
		params.Set(prefix+"[quantity]", strconv.Itoa(line.Line.Qty))
		params.Set(prefix+"[price_data][currency]", currency)
		params.Set(prefix+"[price_data][unit_amount]", strconv.FormatInt(line.Unit, 10))
		params.Set(prefix+"[price_data][product_data][name]", line.Name)
		if line.Product.Kind == kindSubscription {
			params.Set(prefix+"[price_data][recurring][interval]", normalizeInterval(line.Product.Interval))
		}
		if len(line.Product.Images) > 0 {
			if image := absoluteURL(base, line.Product.Images[0].URL); strings.HasPrefix(image, "https://") {
				params.Set(prefix+"[price_data][product_data][images][0]", image)
			}
		}
		if pay.Tax {
			params.Set(prefix+"[price_data][tax_behavior]", "exclusive")
		}
		order.Lines = append(order.Lines, OrderLine{Product: line.Product.ID, Variant: line.Line.Variant, Name: line.Name, Qty: line.Line.Qty, Unit: line.Unit, Total: line.Total,
			Kind: normalizeKind(line.Product.Kind), File: line.Product.File, FileName: line.Product.FileName, Slot: line.Line.Slot})
		order.Ships = order.Ships || line.Product.Ships
	}
	order.Shipping = pay.shippingFor(subtotal, order.Ships)
	order.Total = subtotal + order.Shipping
	if order.Ships {
		for index, country := range pay.Countries {
			params.Set("shipping_address_collection[allowed_countries]["+strconv.Itoa(index)+"]", country)
		}
		name := "Shipping"
		if order.Shipping == 0 {
			name = "Free shipping"
		}
		params.Set("shipping_options[0][shipping_rate_data][type]", "fixed_amount")
		params.Set("shipping_options[0][shipping_rate_data][display_name]", name)
		params.Set("shipping_options[0][shipping_rate_data][fixed_amount][amount]", strconv.FormatInt(order.Shipping, 10))
		params.Set("shipping_options[0][shipping_rate_data][fixed_amount][currency]", currency)
		if pay.Tax {
			params.Set("shipping_options[0][shipping_rate_data][tax_behavior]", "exclusive")
		}
	}
	if pay.Tax {
		params.Set("automatic_tax[enabled]", "true")
	}
	if pay.Promos {
		params.Set("allow_promotion_codes", "true")
	}

	created, err := h.orders.create(order)
	if err != nil {
		http.Redirect(w, r, cartPath+"?checkout=failed", http.StatusSeeOther)
		return
	}
	params.Set("client_reference_id", created.ID)
	params.Set("metadata[order]", created.ID)
	params.Set("metadata[site]", firstNonEmpty(h.settings().Title, h.opts.SiteTitle))

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	session, err := createStripeSession(ctx, pay.SecretKey, params)
	if err != nil {
		// Nobody paid and nobody will: the pending order expires with the
		// abandoned ones rather than showing up as a cancelled sale.
		created.Note = "Stripe refused the checkout: " + err.Error()
		_ = h.orders.update(created)
		http.Redirect(w, r, cartPath+"?checkout=failed", http.StatusSeeOther)
		return
	}
	created.StripeSession = session.ID
	_ = h.orders.update(created)
	http.Redirect(w, r, session.URL, http.StatusSeeOther)
}

// verifyStripeSignature checks the Stripe-Signature header against the
// payload: t=<unix>,v1=<hex hmac sha256 of "<t>.<payload>">.
func verifyStripeSignature(payload []byte, header, secret string, now time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return errors.New("no webhook secret configured")
	}
	var timestamp string
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return errors.New("malformed signature")
	}
	unix, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("malformed timestamp")
	}
	if delta := now.Unix() - unix; delta > 300 || delta < -300 {
		return errors.New("signature too old")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, signature := range signatures {
		if hmac.Equal([]byte(signature), []byte(expected)) {
			return nil
		}
	}
	return errors.New("signature mismatch")
}

// stripeEvent is the slice of a Checkout event this site reads.
type stripeEvent struct {
	Type string `json:"type"`
	Data struct {
		Object struct {
			ID              string `json:"id"`
			PaymentIntent   string `json:"payment_intent"`
			PaymentStatus   string `json:"payment_status"`
			AmountSubtotal  int64  `json:"amount_subtotal"`
			AmountTotal     int64  `json:"amount_total"`
			Currency        string `json:"currency"`
			CustomerDetails struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"customer_details"`
			ShippingDetails struct {
				Name    string `json:"name"`
				Address struct {
					Line1      string `json:"line1"`
					Line2      string `json:"line2"`
					City       string `json:"city"`
					State      string `json:"state"`
					PostalCode string `json:"postal_code"`
					Country    string `json:"country"`
				} `json:"address"`
			} `json:"shipping_details"`
			TotalDetails struct {
				AmountShipping int64 `json:"amount_shipping"`
				AmountTax      int64 `json:"amount_tax"`
				AmountDiscount int64 `json:"amount_discount"`
			} `json:"total_details"`
			Metadata map[string]string `json:"metadata"`
			// Present on sessions in subscription mode, and on the
			// subscription and invoice objects Stripe sends later.
			Mode          string `json:"mode"`
			Subscription  string `json:"subscription"`
			Customer      string `json:"customer"`
			Status        string `json:"status"`
			BillingReason string `json:"billing_reason"`
		} `json:"object"`
	} `json:"data"`
}

func (h *Host) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)
		return
	}
	if err := verifyStripeSignature(payload, r.Header.Get("Stripe-Signature"), h.payments().WebhookSecret, timeNow()); err != nil {
		http.Error(w, "bad signature", http.StatusBadRequest)
		return
	}
	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "bad event", http.StatusBadRequest)
		return
	}
	switch event.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		if event.Data.Object.PaymentStatus == "unpaid" {
			break // an async method; the succeeded event follows
		}
		h.markPaid(event, h.absoluteBase(r))
	case "customer.subscription.deleted":
		h.subscriptionChanged(event.Data.Object.ID, "cancelled")
	case "customer.subscription.updated":
		switch event.Data.Object.Status {
		case "canceled", "unpaid", "incomplete_expired":
			h.subscriptionChanged(event.Data.Object.ID, "cancelled")
		case "past_due":
			h.subscriptionChanged(event.Data.Object.ID, "past_due")
		case "active", "trialing":
			h.subscriptionChanged(event.Data.Object.ID, "active")
		}
	case "invoice.paid":
		if event.Data.Object.BillingReason == "subscription_cycle" && event.Data.Object.Subscription != "" {
			h.subscriptionRenewed(event.Data.Object.Subscription)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// subscriptionChanged records what Stripe says about a subscription's life.
func (h *Host) subscriptionChanged(subscription, status string) {
	for _, order := range h.orders.list() {
		if order.StripeSubscription == subscription && subscription != "" {
			order.SubscriptionStatus = status
			_ = h.orders.update(order)
			return
		}
	}
}

func (h *Host) subscriptionRenewed(subscription string) {
	for _, order := range h.orders.list() {
		if order.StripeSubscription == subscription {
			now := timeNow().UTC()
			order.Renewals++
			order.LastPaid = &now
			order.SubscriptionStatus = "active"
			_ = h.orders.update(order)
			return
		}
	}
}

// markPaid turns a pending order into a paid one, exactly once.
func (h *Host) markPaid(event stripeEvent, base string) {
	object := event.Data.Object
	order, ok := h.orders.get(object.Metadata["order"])
	if !ok {
		order, ok = h.orders.bySession(object.ID)
	}
	if !ok || order.Status != "pending" {
		return
	}
	now := timeNow().UTC()
	order.Status = "paid"
	order.Paid = &now
	order.StripeSession = firstNonEmpty(object.ID, order.StripeSession)
	order.StripePayment = object.PaymentIntent
	order.CustomerEmail = object.CustomerDetails.Email
	order.CustomerName = object.CustomerDetails.Name
	order.ShipTo = Address{
		Name: object.ShippingDetails.Name, Line1: object.ShippingDetails.Address.Line1, Line2: object.ShippingDetails.Address.Line2,
		City: object.ShippingDetails.Address.City, State: object.ShippingDetails.Address.State, Postal: object.ShippingDetails.Address.PostalCode, Country: object.ShippingDetails.Address.Country,
	}
	if object.AmountTotal > 0 {
		order.Total = object.AmountTotal
		order.Shipping = object.TotalDetails.AmountShipping
		order.Tax = object.TotalDetails.AmountTax
		if object.AmountSubtotal > 0 {
			order.Subtotal = object.AmountSubtotal
		}
	}
	if object.Currency != "" {
		order.Currency = normalizeCurrency(object.Currency)
	}
	if object.Subscription != "" {
		order.StripeSubscription = object.Subscription
		order.SubscriptionStatus = "active"
		order.LastPaid = &now
	}
	if object.Customer != "" {
		order.StripeCustomer = object.Customer
	}
	if err := h.orders.update(order); err != nil {
		return
	}
	for _, line := range order.Lines {
		_ = h.products.adjustStock(line.Product, line.Variant, -line.Qty)
		if line.Slot != "" {
			h.confirmBooking(order, line)
		}
	}
	h.notify(h.newOrderMail(order, base))
	if order.CustomerEmail != "" {
		h.notify(h.newReceiptMail(order, base))
	}
}

func (h *Host) newOrderMail(order Order, base string) Mail {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	var b strings.Builder
	b.WriteString("New order " + order.label() + " on " + siteTitle + ".\n\n")
	for _, line := range order.Lines {
		b.WriteString(strconv.Itoa(line.Qty) + " × " + line.Name + " — " + formatMoney(line.Total, order.Currency) + "\n")
	}
	b.WriteString("\nTotal: " + formatMoney(order.Total, order.Currency) + "\n")
	if order.CustomerName != "" || order.CustomerEmail != "" {
		b.WriteString("Buyer: " + strings.TrimSpace(order.CustomerName+" "+order.CustomerEmail) + "\n")
	}
	if order.Ships {
		b.WriteString("Ship to: " + order.ShipTo.oneLine() + "\n")
	}
	b.WriteString("\nSee it at " + base + "/admin/orders/" + order.ID + "\n")
	mail := Mail{To: h.notifyAddress(), Subject: "Order " + order.label() + " — " + siteTitle, Text: b.String()}
	if order.CustomerEmail != "" {
		mail.ReplyTo = order.CustomerEmail
	}
	return mail
}

func (a Address) oneLine() string {
	parts := []string{}
	for _, part := range []string{a.Name, a.Line1, a.Line2, a.City, a.State, a.Postal, a.Country} {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

func (h *Host) handleCheckoutDone(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	writeCart(w, r, nil)
	order, ok := h.orders.bySession(strings.TrimSpace(r.URL.Query().Get("session_id")))
	meta := metaFromSettings(settings)
	meta.Title = "Thank you"
	meta.NoIndex = true
	var body gosx.Node
	switch {
	case ok && order.Status != "pending":
		lines := make([]gosx.Node, 0, len(order.Lines))
		for _, line := range order.Lines {
			lines = append(lines, gosx.El("li", nil, gosx.Text(strconv.Itoa(line.Qty)+" × "+line.Name)))
		}
		body = gosx.Fragment(
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("Thanks — your order "+order.label()+" is in. A receipt is on its way to "+firstNonEmpty(order.CustomerEmail, "your email")+".")),
			gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-list")), gosx.Fragment(lines...)),
			gosx.El("p", nil, gosx.Text("Total paid: "+formatMoney(order.Total, order.Currency))),
			renderDownloads(h.downloadsFor(order, h.absoluteBase(r))),
			renderBookingConfirmation(order),
		)
	case ok:
		body = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("Thanks — your payment is going through. You'll get a receipt by email in a moment."))
	default:
		body = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("Thanks for your order. A receipt is on its way by email."))
	}
	meta, shell := h.publicShell(r, "shop", meta, gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-article")),
		gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text("Thank you")),
		body,
		gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("href", "/")), gosx.Text("Back to the site"))),
	))
	h.writeDocument(w, http.StatusOK, meta, shell)
}

// ---------- admin: orders ----------

func (h *Host) openOrders() int {
	count := 0
	for _, order := range h.orders.list() {
		if order.Status == "paid" && order.Ships {
			count++
		}
	}
	return count
}

func (h *Host) handleAdminOrders(w http.ResponseWriter, r *http.Request) {
	orders := h.orders.list()
	rows := make([]gosx.Node, 0, len(orders))
	for _, order := range orders {
		if order.Status == "pending" {
			continue
		}
		state, label := "draft", "Cancelled"
		switch order.Status {
		case "paid":
			state, label = "scheduled", "Paid"
			if order.Ships {
				label = "Paid — to send"
			}
		case "fulfilled":
			state, label = "published", "Sent"
		}
		rows = append(rows, gosx.El("tr", nil,
			gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/orders/"+order.ID)), gosx.Text(order.label()))),
			gosx.El("td", nil, gosx.Text(formatWhen(order.Created))),
			gosx.El("td", nil, gosx.Text(firstNonEmpty(order.CustomerName, order.CustomerEmail, "—"))),
			gosx.El("td", nil, gosx.Text(formatMoney(order.Total, order.Currency))),
			gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label))),
		))
	}
	var listing gosx.Node
	if len(rows) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No orders yet")),
			gosx.El("p", nil, gosx.Text("When someone pays, the order lands here with what they bought and where to send it.")),
		)
	} else {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Order")), gosx.El("th", nil, gosx.Text("When")), gosx.El("th", nil, gosx.Text("Buyer")), gosx.El("th", nil, gosx.Text("Total")), gosx.El("th", nil, gosx.Text("Status")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...))))
	}
	body := h.renderAdminShell("shop", "Orders", "Everything that's been paid for, newest first.", adminStatus{Message: r.URL.Query().Get("status")}, listing)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Orders"), body)
}

func (h *Host) handleAdminOrder(w http.ResponseWriter, r *http.Request) {
	order, ok := h.orders.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "order")
		return
	}
	h.renderOrder(w, order, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderOrder(w http.ResponseWriter, order Order, status adminStatus) {
	lines := make([]gosx.Node, 0, len(order.Lines))
	for _, line := range order.Lines {
		lines = append(lines, gosx.El("tr", nil,
			gosx.El("td", nil, gosx.Text(line.Name)),
			gosx.El("td", nil, gosx.Text(strconv.Itoa(line.Qty))),
			gosx.El("td", nil, gosx.Text(formatMoney(line.Unit, order.Currency))),
			gosx.El("td", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.Text(formatMoney(line.Total, order.Currency))),
		))
	}
	totals := []gosx.Node{
		gosx.El("tr", nil, gosx.El("td", gosx.Attrs(gosx.Attr("colspan", "3")), gosx.Text("Items")), gosx.El("td", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.Text(formatMoney(order.Subtotal, order.Currency)))),
	}
	if order.Shipping > 0 || order.Ships {
		totals = append(totals, gosx.El("tr", nil, gosx.El("td", gosx.Attrs(gosx.Attr("colspan", "3")), gosx.Text("Shipping")), gosx.El("td", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.Text(formatMoney(order.Shipping, order.Currency)))))
	}
	if order.Tax > 0 {
		totals = append(totals, gosx.El("tr", nil, gosx.El("td", gosx.Attrs(gosx.Attr("colspan", "3")), gosx.Text("Tax")), gosx.El("td", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.Text(formatMoney(order.Tax, order.Currency)))))
	}
	totals = append(totals, gosx.El("tr", nil, gosx.El("td", gosx.Attrs(gosx.Attr("colspan", "3")), gosx.El("strong", nil, gosx.Text("Total paid"))), gosx.El("td", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.El("strong", nil, gosx.Text(formatMoney(order.Total, order.Currency))))))

	statusLabel := map[string]string{"pending": "Waiting for payment", "paid": "Paid", "fulfilled": "Sent", "cancelled": "Cancelled"}[order.Status]
	var actions []gosx.Node
	if order.Status == "paid" {
		actions = append(actions, h.orderActionButton(order.ID, "fulfil", "Mark as sent", "admin-button"))
	}
	if order.Status == "fulfilled" {
		actions = append(actions, h.orderActionButton(order.ID, "unfulfil", "Mark as not sent", "admin-secondary"))
	}
	if order.StripePayment != "" {
		actions = append(actions, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "https://dashboard.stripe.com/payments/"+order.StripePayment), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("Open in Stripe (refunds, receipt)")))
	}
	if order.CustomerEmail != "" {
		actions = append(actions, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "mailto:"+order.CustomerEmail+"?subject=Your%20order%20"+url.QueryEscape(order.label()))), gosx.Text("Email the buyer")))
	}

	details := []gosx.Node{
		gosx.El("p", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", map[string]string{"paid": "scheduled", "fulfilled": "published", "cancelled": "offline"}[order.Status])), gosx.Text(statusLabel)),
			gosx.Text("  Placed "+formatWhen(order.Created))),
	}
	if order.CustomerName != "" || order.CustomerEmail != "" {
		details = append(details, gosx.El("p", nil, gosx.El("strong", nil, gosx.Text("Buyer: ")), gosx.Text(strings.TrimSpace(order.CustomerName+" "+order.CustomerEmail))))
	}
	if order.Ships {
		details = append(details, gosx.El("p", nil, gosx.El("strong", nil, gosx.Text("Send to: ")), gosx.Text(firstNonEmpty(order.ShipTo.oneLine(), "no address given"))))
	}
	if order.StripeSubscription != "" {
		label := map[string]string{"active": "Active subscription", "cancelled": "Cancelled subscription", "past_due": "Subscription payment overdue"}[order.SubscriptionStatus]
		renewals := ""
		if order.Renewals > 0 {
			renewals = " · renewed " + plural(order.Renewals, "time")
		}
		details = append(details, gosx.El("p", nil, gosx.El("strong", nil, gosx.Text(firstNonEmpty(label, "Subscription")+": ")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "https://dashboard.stripe.com/subscriptions/"+order.StripeSubscription), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("manage in Stripe")), gosx.Text(renewals)))
	}
	if links := h.downloadsFor(order, h.absoluteBaseFromSettings()); len(links) > 0 {
		items := make([]gosx.Node, 0, len(links))
		for _, link := range links {
			items = append(items, gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", link.URL)), gosx.Text(link.Name))))
		}
		details = append(details, gosx.El("p", nil, gosx.El("strong", nil, gosx.Text("Downloads the buyer received: "))), gosx.El("ul", nil, gosx.Fragment(items...)))
	}
	for _, line := range order.Lines {
		if line.Slot != "" {
			details = append(details, gosx.El("p", nil, gosx.El("strong", nil, gosx.Text("Booked: ")), gosx.Text(line.Name)))
		}
	}
	if order.Note != "" {
		details = append(details, gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(order.Note)))
	}
	panel := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.Fragment(details...),
		gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
			gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Item")), gosx.El("th", nil, gosx.Text("Qty")), gosx.El("th", nil, gosx.Text("Each")), gosx.El("th", gosx.Attrs(gosx.Attr("class", "stats-count")), gosx.Text("Total")))),
			gosx.El("tbody", nil, gosx.Fragment(lines...)),
			gosx.El("tfoot", nil, gosx.Fragment(totals...)),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")), gosx.Fragment(actions...)),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/orders/"+order.ID)),
			h.csrfField(),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "action"), gosx.Attr("value", "note"))),
			adminTextareaField("note", "Your notes", order.Note, "Tracking numbers, what was substituted, anything to remember. Only you see this."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Save note"))),
		),
	)
	back := gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")), gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/orders")), gosx.Text("All orders")))
	body := h.renderAdminShell("shop", "Order "+order.label(), "", status, panel, back)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Order "+order.label()), body)
}

func (h *Host) orderActionButton(id, action, label, class string) gosx.Node {
	return gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/orders/"+id), gosx.Attr("class", "admin-inline-form")),
		h.csrfField(),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "action"), gosx.Attr("value", action))),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", class), gosx.Attr("type", "submit")), gosx.Text(label)),
	)
}

func (h *Host) handleAdminOrderAction(w http.ResponseWriter, r *http.Request) {
	order, ok := h.orders.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "order")
		return
	}
	_ = r.ParseForm()
	message := "Saved."
	now := timeNow().UTC()
	switch r.PostFormValue("action") {
	case "fulfil":
		if order.Status == "paid" {
			order.Status = "fulfilled"
			order.Fulfilled = &now
			message = "Marked as sent."
		}
	case "unfulfil":
		if order.Status == "fulfilled" {
			order.Status = "paid"
			order.Fulfilled = nil
			message = "Marked as not sent."
		}
	case "note":
		order.Note = strings.TrimSpace(r.PostFormValue("note"))
		message = "Note saved."
	}
	if err := h.orders.update(order); err != nil {
		h.renderOrder(w, order, adminStatus{Message: "We couldn't save that. Try again.", Error: true})
		return
	}
	http.Redirect(w, r, "/admin/orders/"+order.ID+"?status="+queryEscape(message), http.StatusSeeOther)
}
