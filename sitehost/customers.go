package sitehost

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
)

// customers.go is the buyer's side of the shop: their orders, their
// downloads, their subscription — and the cart they walked away from.
//
// There are no customer passwords. A buyer types their email, gets a link
// that signs them in for a month, and sees every paid order under that
// address. A subscription is managed on Stripe's own billing portal. A
// visitor can also ask to be emailed their cart; if they haven't bought
// within a couple of hours, one reminder goes out with a link that puts
// the cart back. One, never more.

const (
	customerPath     = "/account"
	customerCookie   = "gosx_customer"
	customerLinkTTL  = 30 * time.Minute
	customerLoginTTL = 30 * 24 * time.Hour
	cartReminderWait = 2 * time.Hour
	stripePortalAPI  = "https://api.stripe.com/v1/billing_portal/sessions"
)

// createStripePortal opens Stripe's billing portal for a customer. Tests
// replace it.
var createStripePortal = func(ctx context.Context, secretKey, customer, returnURL string) (string, error) {
	form := url.Values{}
	form.Set("customer", customer)
	form.Set("return_url", returnURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stripePortalAPI, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var session struct {
		URL string `json:"url"`
	}
	if res.StatusCode >= 300 || json.Unmarshal(body, &session) != nil || session.URL == "" {
		return "", errors.New("stripe: could not open the billing portal")
	}
	return session.URL, nil
}

// ---------- signed tokens ----------

func (h *Host) customerToken(email string, ttl time.Duration) string {
	expiry := strconv.FormatInt(timeNow().Add(ttl).Unix(), 10)
	value := email + "|" + expiry
	return base64.RawURLEncoding.EncodeToString([]byte(value + "|" + h.sign("customer:"+value)))
}

func (h *Host) parseCustomerToken(token string) (string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", false
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || !tokensEqual(parts[2], h.sign("customer:"+parts[0]+"|"+parts[1])) {
		return "", false
	}
	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || timeNow().Unix() > expiry {
		return "", false
	}
	return normalizeEmail(parts[0]), true
}

func (h *Host) customerEmail(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(customerCookie)
	if err != nil {
		return "", false
	}
	return h.parseCustomerToken(cookie.Value)
}

func setCustomerCookie(w http.ResponseWriter, r *http.Request, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: customerCookie, Value: value, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"), MaxAge: maxAge})
}

// ---------- saved carts ----------

type savedCart struct {
	ID       string     `json:"id"`
	Email    string     `json:"email"`
	Lines    []cartLine `json:"lines"`
	Created  time.Time  `json:"created"`
	Reminded bool       `json:"reminded"`
}

type cartStore struct {
	mu     sync.Mutex
	path   string
	loaded bool
	carts  []savedCart
}

func newCartStore(path string) *cartStore { return &cartStore{path: path} }

func (o Options) cartsPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "carts.json")
}

func (s *cartStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file struct {
		Carts []savedCart `json:"carts"`
	}
	if json.Unmarshal(raw, &file) == nil {
		s.carts = file.Carts
	}
}

func (s *cartStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	// Anything older than a month is nobody's cart any more.
	cutoff := timeNow().UTC().AddDate(0, -1, 0)
	kept := s.carts[:0]
	for _, cart := range s.carts {
		if cart.Created.After(cutoff) {
			kept = append(kept, cart)
		}
	}
	s.carts = kept
	raw, err := json.MarshalIndent(struct {
		Carts []savedCart `json:"carts"`
	}{s.carts}, "", "  ")
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

// save keeps one cart per email, the newest.
func (s *cartStore) save(email string, lines []cartLine) (savedCart, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	kept := s.carts[:0]
	for _, cart := range s.carts {
		if cart.Email != email {
			kept = append(kept, cart)
		}
	}
	s.carts = kept
	cart := savedCart{ID: "c" + randomHex(5), Email: email, Lines: lines, Created: timeNow().UTC()}
	s.carts = append(s.carts, cart)
	return cart, s.saveLocked()
}

func (s *cartStore) get(id string) (savedCart, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, cart := range s.carts {
		if cart.ID == id {
			return cart, true
		}
	}
	return savedCart{}, false
}

func (s *cartStore) list() []savedCart {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]savedCart, len(s.carts))
	copy(out, s.carts)
	return out
}

func (s *cartStore) markReminded(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index := range s.carts {
		if s.carts[index].ID == id {
			s.carts[index].Reminded = true
		}
	}
	_ = s.saveLocked()
}

func (s *cartStore) remove(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	kept := s.carts[:0]
	for _, cart := range s.carts {
		if cart.Email != email {
			kept = append(kept, cart)
		}
	}
	s.carts = kept
	_ = s.saveLocked()
}

// ---------- routes ----------

func (h *Host) mountCustomers(mux *http.ServeMux) {
	mux.HandleFunc("GET "+customerPath, h.handleAccountPage)
	mux.HandleFunc("GET "+customerPath+"/{$}", h.handleAccountPage)
	mux.HandleFunc("POST "+customerPath, h.handleAccountLink)
	mux.HandleFunc("GET "+customerPath+"/in/{token}", h.handleAccountEnter)
	mux.HandleFunc("POST "+customerPath+"/out", h.handleAccountOut)
	mux.HandleFunc("POST "+customerPath+"/portal", h.handleAccountPortal)
	mux.HandleFunc("POST "+cartPath+"/email", h.handleCartEmail)
	mux.HandleFunc("GET "+cartPath+"/restore/{token}", h.handleCartRestore)
}

// ordersFor lists paid orders under an email, newest first.
func (h *Host) ordersFor(email string) []Order {
	out := []Order{}
	for _, order := range h.orders.list() {
		if normalizeEmail(order.CustomerEmail) == email && (order.Status == "paid" || order.Status == "fulfilled") {
			out = append(out, order)
		}
	}
	return out
}

func (h *Host) handleAccountPage(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	meta := metaFromSettings(settings)
	meta.Title = "Your orders"
	meta.NoIndex = true
	email, signedIn := h.customerEmail(r)
	var body gosx.Node
	if !signedIn {
		note := gosx.Fragment()
		switch r.URL.Query().Get("sent") {
		case "1":
			note = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-notice"), gosx.Attr("role", "status")), gosx.Text("If we have orders for that address, a sign-in link is on its way. It works for 30 minutes."))
		case "nomail":
			note = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "alert")), gosx.Text("This site can't send email yet, so we can't send a sign-in link. Get in touch through the contact page instead."))
		case "expired":
			note = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "alert")), gosx.Text("That link has expired. Ask for a new one."))
		}
		body = gosx.Fragment(note,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("Type the email you used when you ordered and we'll send you a link — no password needed.")),
			gosx.El("form", gosx.Attrs(gosx.Attr("class", "site-form"), gosx.Attr("method", "post"), gosx.Attr("action", customerPath)),
				formField("email", "Your email", "email", "", 200),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-form__hp"), gosx.Attr("aria-hidden", "true")),
					gosx.El("label", nil, gosx.Text("Leave this field empty"), gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "website"), gosx.Attr("tabindex", "-1"), gosx.Attr("autocomplete", "off"))))),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("type", "submit")), gosx.Text("Send me a link"))),
		)
	} else {
		orders := h.ordersFor(email)
		base := h.absoluteBase(r)
		items := make([]gosx.Node, 0, len(orders))
		for _, order := range orders {
			lines := make([]gosx.Node, 0, len(order.Lines))
			for _, line := range order.Lines {
				lines = append(lines, gosx.El("li", nil, gosx.Text(strconv.Itoa(line.Qty)+" × "+line.Name)))
			}
			status := map[string]string{"paid": "Paid", "fulfilled": "Sent"}[order.Status]
			if order.Ships && order.Status == "paid" {
				status = "Paid — being prepared"
			}
			extras := []gosx.Node{renderDownloads(h.downloadsFor(order, base)), renderBookingConfirmation(order)}
			if order.StripeSubscription != "" {
				label := map[string]string{"active": "Subscription active", "cancelled": "Subscription cancelled", "past_due": "Subscription payment overdue"}[order.SubscriptionStatus]
				manage := gosx.Fragment()
				if order.StripeCustomer != "" && h.checkoutReady() {
					manage = gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", customerPath+"/portal"), gosx.Attr("class", "site-inline-form")),
						gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "order"), gosx.Attr("value", order.ID))),
						gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-cart__update"), gosx.Attr("type", "submit")), gosx.Text("Manage subscription")))
				}
				extras = append(extras, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Text(firstNonEmpty(label, "Subscription")+" ")), manage)
			}
			items = append(items, gosx.El("li", gosx.Attrs(gosx.Attr("class", "site-post-card")),
				gosx.El("h2", gosx.Attrs(gosx.Attr("class", "site-post-card__title")), gosx.Text("Order "+order.label())),
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Text(formatWhen(order.Created)+" · "+formatMoney(order.Total, order.Currency)+" · "+status)),
				gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-list")), gosx.Fragment(lines...)),
				gosx.Fragment(extras...),
			))
		}
		var list gosx.Node = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("No orders under "+email+" yet."))
		if len(items) > 0 {
			list = gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-posts")), gosx.Fragment(items...))
		}
		body = gosx.Fragment(
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Text("Signed in as "+email+". "),
				gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", customerPath+"/out"), gosx.Attr("class", "site-inline-form")),
					gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-cart__remove"), gosx.Attr("type", "submit")), gosx.Text("Sign out")))),
			list,
		)
	}
	meta, shell := h.publicShell(r, "account", meta, gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-article")),
		gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text("Your orders")), body))
	h.writeDocument(w, http.StatusOK, meta, shell)
}

func (h *Host) handleAccountLink(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if strings.TrimSpace(r.PostFormValue("website")) != "" {
		http.Redirect(w, r, customerPath+"?sent=1", http.StatusSeeOther)
		return
	}
	email := normalizeEmail(r.PostFormValue("email"))
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, customerPath, http.StatusSeeOther)
		return
	}
	if h.mailer == nil {
		http.Redirect(w, r, customerPath+"?sent=nomail", http.StatusSeeOther)
		return
	}
	if !h.messages.allow(remoteHost(r), timeNow()) {
		http.Redirect(w, r, customerPath+"?sent=1", http.StatusSeeOther)
		return
	}
	// The answer is the same whether or not orders exist, so the form
	// cannot be used to learn who has bought.
	if len(h.ordersFor(email)) > 0 {
		link := h.absoluteBase(r) + customerPath + "/in/" + h.customerToken(email, customerLinkTTL)
		siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
		h.notify(Mail{To: email, Subject: "Your orders at " + siteTitle, Text: "Open this link to see your orders at " + siteTitle + ":\n\n" + link + "\n\nIt works for 30 minutes. If you didn't ask for it, ignore this email.\n", ReplyTo: h.notifyAddress()})
	}
	http.Redirect(w, r, customerPath+"?sent=1", http.StatusSeeOther)
}

func (h *Host) handleAccountEnter(w http.ResponseWriter, r *http.Request) {
	email, ok := h.parseCustomerToken(r.PathValue("token"))
	if !ok {
		http.Redirect(w, r, customerPath+"?sent=expired", http.StatusSeeOther)
		return
	}
	setCustomerCookie(w, r, h.customerToken(email, customerLoginTTL), int(customerLoginTTL/time.Second))
	http.Redirect(w, r, customerPath, http.StatusSeeOther)
}

func (h *Host) handleAccountOut(w http.ResponseWriter, r *http.Request) {
	setCustomerCookie(w, r, "", -1)
	http.Redirect(w, r, customerPath, http.StatusSeeOther)
}

func (h *Host) handleAccountPortal(w http.ResponseWriter, r *http.Request) {
	email, ok := h.customerEmail(r)
	if !ok {
		http.Redirect(w, r, customerPath, http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	order, found := h.orders.get(r.PostFormValue("order"))
	if !found || normalizeEmail(order.CustomerEmail) != email || order.StripeCustomer == "" || !h.checkoutReady() {
		http.Redirect(w, r, customerPath, http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	portal, err := createStripePortal(ctx, h.payments().SecretKey, order.StripeCustomer, h.absoluteBase(r)+customerPath)
	if err != nil {
		http.Redirect(w, r, customerPath, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, portal, http.StatusSeeOther)
}

// ---------- cart reminders ----------

func (h *Host) handleCartEmail(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	email := normalizeEmail(r.PostFormValue("email"))
	resolved, _ := h.resolveCart(readCart(r))
	if _, err := mail.ParseAddress(email); err != nil || len(resolved) == 0 {
		http.Redirect(w, r, cartPath+"?saved=no", http.StatusSeeOther)
		return
	}
	if !h.messages.allow(remoteHost(r), timeNow()) {
		http.Redirect(w, r, cartPath, http.StatusSeeOther)
		return
	}
	if _, err := h.carts.save(email, linesOf(resolved)); err != nil {
		http.Redirect(w, r, cartPath+"?saved=no", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, cartPath+"?saved=1", http.StatusSeeOther)
}

func (h *Host) cartRestoreToken(id string) string {
	value := id + "|" + strconv.FormatInt(timeNow().Add(30*24*time.Hour).Unix(), 10)
	return base64.RawURLEncoding.EncodeToString([]byte(value + "|" + h.sign("cart:"+value)))
}

func (h *Host) handleCartRestore(w http.ResponseWriter, r *http.Request) {
	raw, err := base64.RawURLEncoding.DecodeString(r.PathValue("token"))
	if err != nil {
		http.Redirect(w, r, cartPath, http.StatusSeeOther)
		return
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 || !tokensEqual(parts[2], h.sign("cart:"+parts[0]+"|"+parts[1])) {
		http.Redirect(w, r, cartPath, http.StatusSeeOther)
		return
	}
	if expiry, err := strconv.ParseInt(parts[1], 10, 64); err != nil || timeNow().Unix() > expiry {
		http.Redirect(w, r, cartPath, http.StatusSeeOther)
		return
	}
	if cart, ok := h.carts.get(parts[0]); ok {
		writeCart(w, r, cart.Lines)
	}
	http.Redirect(w, r, cartPath, http.StatusSeeOther)
}

// remindCarts sends one reminder for each cart left for a couple of hours
// by someone who asked to be emailed it and has not bought since.
func (h *Host) remindCarts() {
	if h.mailer == nil {
		return
	}
	now := timeNow()
	for _, cart := range h.carts.list() {
		if cart.Reminded || now.Sub(cart.Created) < cartReminderWait {
			continue
		}
		bought := false
		for _, order := range h.ordersFor(cart.Email) {
			if order.Created.After(cart.Created) {
				bought = true
			}
		}
		h.carts.markReminded(cart.ID)
		if bought {
			continue
		}
		resolved, subtotal := h.resolveCart(cart.Lines)
		if len(resolved) == 0 {
			continue
		}
		siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
		var b strings.Builder
		b.WriteString("You left these in your cart at " + siteTitle + ":\n\n")
		for _, line := range resolved {
			b.WriteString(strconv.Itoa(line.Line.Qty) + " × " + line.Name + " — " + formatMoney(line.Total, h.currency()) + "\n")
		}
		b.WriteString("\nSubtotal " + formatMoney(subtotal, h.currency()) + ". Pick up where you left off:\n" + h.absoluteBaseFromSettings() + cartPath + "/restore/" + h.cartRestoreToken(cart.ID) + "\n\nThis is the only reminder we'll send.\n")
		h.notify(Mail{To: cart.Email, Subject: "Still thinking it over? Your cart at " + siteTitle, Text: b.String(), ReplyTo: h.notifyAddress()})
	}
}

// renderCartEmailForm is the "email me my cart" field on the cart page.
func renderCartEmailForm(saved string) gosx.Node {
	note := gosx.Fragment()
	switch saved {
	case "1":
		note = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Text("Saved. If you don't finish today we'll send one reminder with a link back to it."))
	case "no":
		note = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error")), gosx.Text("That email doesn't look right."))
	}
	return gosx.El("form", gosx.Attrs(gosx.Attr("class", "site-cart__save"), gosx.Attr("method", "post"), gosx.Attr("action", cartPath+"/email")),
		note,
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")),
			gosx.El("span", nil, gosx.Text("Not ready? Email me a link to this cart")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "email"), gosx.Attr("name", "email"), gosx.Attr("maxlength", "200"), gosx.Attr("placeholder", "you@example.com")))),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-cart__update"), gosx.Attr("type", "submit")), gosx.Text("Save my cart")),
	)
}
