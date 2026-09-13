package sitehost

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
)

// bookings.go is a product with a calendar: a haircut, a table, an hour of
// consulting.
//
// The owner says which days and hours they take bookings and how long a
// slot is; the product page shows a date and the open times; a visitor
// picks one and either pays through the usual checkout or, for a free
// booking, just leaves a name and email. Every booking is a row the owner
// can see by day and cancel, and a cancelled slot opens again at once.

// BookingRules is when a service can be booked.
type BookingRules struct {
	Days        []int  `json:"days,omitempty"` // 0 Sunday … 6 Saturday
	Start       string `json:"start,omitempty"`
	End         string `json:"end,omitempty"`
	SlotMinutes int    `json:"slotMinutes,omitempty"`
	LeadHours   int    `json:"leadHours,omitempty"`
	WeeksAhead  int    `json:"weeksAhead,omitempty"`
}

func normalizeBookingRules(rules BookingRules) BookingRules {
	if len(rules.Days) == 0 {
		rules.Days = []int{1, 2, 3, 4, 5}
	}
	days := []int{}
	seen := map[int]bool{}
	for _, day := range rules.Days {
		if day >= 0 && day <= 6 && !seen[day] {
			seen[day] = true
			days = append(days, day)
		}
	}
	sort.Ints(days)
	rules.Days = days
	if _, err := time.Parse("15:04", rules.Start); err != nil {
		rules.Start = "09:00"
	}
	if _, err := time.Parse("15:04", rules.End); err != nil {
		rules.End = "17:00"
	}
	if rules.SlotMinutes < 5 || rules.SlotMinutes > 24*60 {
		rules.SlotMinutes = 60
	}
	if rules.LeadHours < 0 || rules.LeadHours > 24*30 {
		rules.LeadHours = 24
	}
	if rules.WeeksAhead < 1 || rules.WeeksAhead > 52 {
		rules.WeeksAhead = 8
	}
	return rules
}

// Booking is one reserved slot.
type Booking struct {
	ID            string    `json:"id"`
	Product       string    `json:"product"`
	ProductName   string    `json:"productName"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	CustomerName  string    `json:"customerName,omitempty"`
	CustomerEmail string    `json:"customerEmail,omitempty"`
	Order         string    `json:"order,omitempty"`
	Status        string    `json:"status"` // booked, cancelled
	Note          string    `json:"note,omitempty"`
	Created       time.Time `json:"created"`
}

type bookingStore struct {
	mu       sync.Mutex
	path     string
	loaded   bool
	bookings []Booking
}

func newBookingStore(path string) *bookingStore { return &bookingStore{path: path} }

func (o Options) bookingsPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "bookings.json")
}

func (s *bookingStore) loadLocked() {
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
		Bookings []Booking `json:"bookings"`
	}
	if json.Unmarshal(raw, &file) == nil {
		s.bookings = file.Bookings
	}
}

func (s *bookingStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(struct {
		Bookings []Booking `json:"bookings"`
	}{s.bookings}, "", "  ")
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

func (s *bookingStore) list() []Booking {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]Booking, len(s.bookings))
	copy(out, s.bookings)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	return out
}

func (s *bookingStore) add(booking Booking) (Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	booking.ID = "b" + randomHex(4)
	booking.Status = "booked"
	booking.Created = timeNow().UTC()
	s.bookings = append(s.bookings, booking)
	return booking, s.saveLocked()
}

func (s *bookingStore) update(booking Booking) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index, existing := range s.bookings {
		if existing.ID == booking.ID {
			s.bookings[index] = booking
			return s.saveLocked()
		}
	}
	return errors.New("booking not found")
}

func (s *bookingStore) get(id string) (Booking, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, booking := range s.bookings {
		if booking.ID == id {
			return booking, true
		}
	}
	return Booking{}, false
}

// taken reports whether a product already has a live booking at start.
func (s *bookingStore) taken(product string, start time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, booking := range s.bookings {
		if booking.Product == product && booking.Status == "booked" && booking.Start.Equal(start) {
			return true
		}
	}
	return false
}

// ---------- slots ----------

func slotLabel(rfc string) string {
	at, err := time.Parse(time.RFC3339, rfc)
	if err != nil {
		return rfc
	}
	return at.In(time.Local).Format("Mon 2 Jan, 15:04")
}

// availableSlots lists the open start times on one day, in local time.
func (h *Host) availableSlots(product Product, day time.Time) []time.Time {
	rules := normalizeBookingRules(product.Booking)
	day = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.Local)
	allowed := false
	for _, weekday := range rules.Days {
		if int(day.Weekday()) == weekday {
			allowed = true
		}
	}
	if !allowed {
		return nil
	}
	start, _ := time.Parse("15:04", rules.Start)
	end, _ := time.Parse("15:04", rules.End)
	first := day.Add(time.Duration(start.Hour())*time.Hour + time.Duration(start.Minute())*time.Minute)
	last := day.Add(time.Duration(end.Hour())*time.Hour + time.Duration(end.Minute())*time.Minute)
	earliest := timeNow().Add(time.Duration(rules.LeadHours) * time.Hour)
	latest := timeNow().Add(time.Duration(rules.WeeksAhead) * 7 * 24 * time.Hour)
	slots := []time.Time{}
	for at := first; at.Add(time.Duration(rules.SlotMinutes)*time.Minute).Compare(last) <= 0; at = at.Add(time.Duration(rules.SlotMinutes) * time.Minute) {
		if at.Before(earliest) || at.After(latest) || h.bookings.taken(product.ID, at.UTC()) {
			continue
		}
		slots = append(slots, at)
	}
	return slots
}

// slotOK reports whether a start time is one the product offers right now.
func (h *Host) slotOK(product Product, start time.Time) bool {
	for _, slot := range h.availableSlots(product, start.In(time.Local)) {
		if slot.Equal(start) {
			return true
		}
	}
	return false
}

// ---------- the product page ----------

func (h *Host) renderBookingForm(product Product, bookingDay *time.Time) gosx.Node {
	rules := normalizeBookingRules(product.Booking)
	currency := h.currency()
	today := timeNow().In(time.Local)
	min := today.Format("2006-01-02")
	max := today.Add(time.Duration(rules.WeeksAhead) * 7 * 24 * time.Hour).Format("2006-01-02")
	chosen := ""
	var slotNodes []gosx.Node
	if bookingDay != nil {
		day := *bookingDay
		{
			chosen = day.Format("2006-01-02")
			slots := h.availableSlots(product, day)
			if len(slots) == 0 {
				slotNodes = append(slotNodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Text("Nothing free on "+day.Format("Monday 2 January")+". Try another day.")))
			}
			for index, slot := range slots {
				id := "slot-" + strconv.Itoa(index)
				attrs := []any{gosx.Attr("type", "radio"), gosx.Attr("name", "slot"), gosx.Attr("id", id), gosx.Attr("value", slot.UTC().Format(time.RFC3339)), gosx.Attr("required", "required")}
				if index == 0 {
					attrs = append(attrs, gosx.Attr("checked", "checked"))
				}
				slotNodes = append(slotNodes, gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-slot"), gosx.Attr("for", id)),
					gosx.El("input", gosx.Attrs(attrs...)), gosx.Text(slot.Format("15:04"))))
			}
		}
	}
	dayField := gosx.El("form", gosx.Attrs(gosx.Attr("class", "site-form site-buy site-book__day"), gosx.Attr("method", "get"), gosx.Attr("action", product.path())),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")),
			gosx.El("span", nil, gosx.Text("Pick a day")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "date"), gosx.Attr("name", "date"), gosx.Attr("value", chosen), gosx.Attr("min", min), gosx.Attr("max", max), gosx.Attr("required", "required")))),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-cart__update"), gosx.Attr("type", "submit")), gosx.Text("See times")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Text(bookingHours(rules)+". Times are "+time.Local.String()+".")),
	)
	if len(slotNodes) == 0 || chosen == "" {
		return gosx.Fragment(dayField)
	}
	button := "Book and pay " + formatMoney(product.Price, currency)
	action := cartPath + "/add"
	extra := []gosx.Node{}
	if product.Price == 0 {
		button = "Book"
		action = "/book/" + product.ID
		extra = append(extra,
			formField("name", "Your name", "text", "", 120),
			formField("email", "Your email", "email", "", 200),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-form__hp"), gosx.Attr("aria-hidden", "true")),
				gosx.El("label", nil, gosx.Text("Leave this field empty"),
					gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "website"), gosx.Attr("tabindex", "-1"), gosx.Attr("autocomplete", "off"))))),
		)
	}
	return gosx.Fragment(dayField,
		gosx.El("form", gosx.Attrs(gosx.Attr("class", "site-form site-buy site-book__slots"), gosx.Attr("method", "post"), gosx.Attr("action", action)),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "product"), gosx.Attr("value", product.ID))),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "qty"), gosx.Attr("value", "1"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-slots"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Times")), gosx.Fragment(slotNodes...)),
			gosx.Fragment(extra...),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("type", "submit")), gosx.Text(button)),
		))
}

func bookingHours(rules BookingRules) string {
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	days := make([]string, 0, len(rules.Days))
	for _, day := range rules.Days {
		days = append(days, names[day])
	}
	return strings.Join(days, ", ") + " " + rules.Start + "–" + rules.End + ", " + strconv.Itoa(rules.SlotMinutes) + " minutes each"
}

// ---------- booking without payment ----------

func (h *Host) mountBookings(mux *http.ServeMux) {
	mux.HandleFunc("POST /book/{id}", h.handleFreeBooking)
	mux.HandleFunc("GET /book/done", h.handleBookingDone)
	mux.HandleFunc("GET /admin/bookings", h.handleAdminBookings)
	mux.HandleFunc("POST /admin/bookings/{id}", h.handleAdminBookingAction)
}

func (h *Host) handleFreeBooking(w http.ResponseWriter, r *http.Request) {
	product, ok := h.products.get(r.PathValue("id"))
	if !ok || !product.Active || product.Kind != kindBooking || product.Price > 0 {
		http.Redirect(w, r, shopPath, http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	if strings.TrimSpace(r.PostFormValue("website")) != "" {
		http.Redirect(w, r, product.path(), http.StatusSeeOther)
		return
	}
	start, err := time.Parse(time.RFC3339, r.PostFormValue("slot"))
	name := strings.TrimSpace(r.PostFormValue("name"))
	email := strings.TrimSpace(r.PostFormValue("email"))
	if err != nil || !h.slotOK(product, start) || name == "" {
		http.Redirect(w, r, product.path()+"?date="+start.In(time.Local).Format("2006-01-02")+"&problem=slot", http.StatusSeeOther)
		return
	}
	if _, err := mail.ParseAddress(email); err != nil {
		http.Redirect(w, r, product.path()+"?date="+start.In(time.Local).Format("2006-01-02")+"&problem=email", http.StatusSeeOther)
		return
	}
	if !h.messages.allow(remoteHost(r), timeNow()) {
		http.Redirect(w, r, product.path()+"?problem=rate", http.StatusSeeOther)
		return
	}
	rules := normalizeBookingRules(product.Booking)
	booking, err := h.bookings.add(Booking{Product: product.ID, ProductName: product.Name, Start: start.UTC(), End: start.UTC().Add(time.Duration(rules.SlotMinutes) * time.Minute), CustomerName: name, CustomerEmail: email})
	if err != nil {
		http.Redirect(w, r, product.path()+"?problem=save", http.StatusSeeOther)
		return
	}
	h.notify(h.newBookingMail(booking, h.absoluteBase(r)))
	h.notify(Mail{To: email, Subject: "Booked: " + product.Name + ", " + slotLabel(start.UTC().Format(time.RFC3339)), ReplyTo: h.notifyAddress(),
		Text: "Hi " + name + ",\n\nYou're booked for " + product.Name + " on " + slotLabel(start.UTC().Format(time.RFC3339)) + ".\n\nReply to this email if you need to change it.\n"})
	http.Redirect(w, r, "/book/done?id="+booking.ID, http.StatusSeeOther)
}

func (h *Host) handleBookingDone(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	booking, ok := h.bookings.get(strings.TrimSpace(r.URL.Query().Get("id")))
	meta := metaFromSettings(settings)
	meta.Title = "Booked"
	meta.NoIndex = true
	text := "Thanks — you're booked. We've sent the details by email."
	if ok {
		text = "Thanks, " + booking.CustomerName + " — you're booked for " + booking.ProductName + " on " + slotLabel(booking.Start.Format(time.RFC3339)) + ". We've sent the details to " + booking.CustomerEmail + "."
	}
	meta, shell := h.publicShell(r, "shop", meta, gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-article")),
		gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text("You're booked")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text(text)),
		gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("href", "/")), gosx.Text("Back to the site"))),
	))
	h.writeDocument(w, http.StatusOK, meta, shell)
}

// confirmBooking turns a paid order line into a booking.
func (h *Host) confirmBooking(order Order, line OrderLine) {
	start, err := time.Parse(time.RFC3339, line.Slot)
	if err != nil {
		return
	}
	product, _ := h.products.get(line.Product)
	rules := normalizeBookingRules(product.Booking)
	booking, err := h.bookings.add(Booking{Product: line.Product, ProductName: firstNonEmpty(product.Name, line.Name), Start: start.UTC(), End: start.UTC().Add(time.Duration(rules.SlotMinutes) * time.Minute),
		CustomerName: order.CustomerName, CustomerEmail: order.CustomerEmail, Order: order.ID})
	if err == nil {
		h.notify(h.newBookingMail(booking, h.absoluteBaseFromSettings()))
	}
}

func (h *Host) newBookingMail(booking Booking, base string) Mail {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	return Mail{To: h.notifyAddress(), Subject: "New booking: " + booking.ProductName + ", " + slotLabel(booking.Start.Format(time.RFC3339)) + " — " + siteTitle,
		Text:    strings.TrimSpace(booking.CustomerName+" "+booking.CustomerEmail) + " booked " + booking.ProductName + " for " + slotLabel(booking.Start.Format(time.RFC3339)) + ".\n\nSee all bookings at " + base + "/admin/bookings\n",
		ReplyTo: booking.CustomerEmail}
}

func renderBookingConfirmation(order Order) gosx.Node {
	items := []gosx.Node{}
	for _, line := range order.Lines {
		if line.Slot != "" {
			items = append(items, gosx.El("li", nil, gosx.Text("Booked: "+line.Name)))
		}
	}
	if len(items) == 0 {
		return gosx.Fragment()
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-downloads")),
		gosx.El("h3", nil, gosx.Text("Your booking")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-list")), gosx.Fragment(items...)))
}

// ---------- admin ----------

func (h *Host) handleAdminBookings(w http.ResponseWriter, r *http.Request) {
	all := h.bookings.list()
	now := timeNow()
	upcoming := make([]Booking, 0, len(all))
	past := 0
	for _, booking := range all {
		if booking.Status == "booked" && booking.Start.After(now.Add(-24*time.Hour)) {
			upcoming = append(upcoming, booking)
		} else {
			past++
		}
	}
	var listing gosx.Node
	if len(upcoming) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("Nothing booked yet")),
			gosx.El("p", nil, gosx.Text("Make a product a bookable service under Shop, set its days and hours, and bookings appear here by day.")))
	} else {
		rows := make([]gosx.Node, 0, len(upcoming))
		lastDay := ""
		for _, booking := range upcoming {
			day := booking.Start.In(time.Local).Format("Monday 2 January")
			if day != lastDay {
				rows = append(rows, gosx.El("tr", gosx.Attrs(gosx.Attr("class", "admin-day")), gosx.El("th", gosx.Attrs(gosx.Attr("colspan", "4")), gosx.Text(day))))
				lastDay = day
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.Text(booking.Start.In(time.Local).Format("15:04")+"–"+booking.End.In(time.Local).Format("15:04"))),
				gosx.El("td", nil, gosx.Text(booking.ProductName)),
				gosx.El("td", nil, gosx.Text(strings.TrimSpace(booking.CustomerName+" "+booking.CustomerEmail))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
					gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/bookings/"+booking.ID), gosx.Attr("class", "admin-inline-form")),
						h.csrfField(), hidden("action", "cancel"),
						gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit")), gosx.Text("Cancel")))),
			))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")), gosx.El("tbody", nil, gosx.Fragment(rows...))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(plural(past, "past or cancelled booking")+" not shown.")))
	}
	body := h.renderAdminShell("shop", "Bookings", "Who's coming when. Cancelling frees the time for someone else.", adminStatus{Message: r.URL.Query().Get("status")}, listing)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Bookings"), body)
}

func (h *Host) handleAdminBookingAction(w http.ResponseWriter, r *http.Request) {
	booking, ok := h.bookings.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "booking")
		return
	}
	_ = r.ParseForm()
	if r.PostFormValue("action") == "cancel" {
		booking.Status = "cancelled"
		_ = h.bookings.update(booking)
		h.auditContent(r, "booking.cancelled", "Cancelled "+booking.ProductName+" for "+booking.CustomerName)
	}
	http.Redirect(w, r, "/admin/bookings?status="+queryEscape("Cancelled. The time is free again."), http.StatusSeeOther)
}

// ---------- the editor's rules ----------

func renderBookingRuleFields(product Product) gosx.Node {
	rules := normalizeBookingRules(product.Booking)
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	days := make([]gosx.Node, 0, 7)
	for index, name := range names {
		attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "bookDay"), gosx.Attr("value", strconv.Itoa(index))}
		for _, day := range rules.Days {
			if day == index {
				attrs = append(attrs, gosx.Attr("checked", "checked"))
			}
		}
		days = append(days, gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-radio")), gosx.El("input", gosx.Attrs(attrs...)), gosx.Text(" "+name)))
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("When it can be booked (bookable services)")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field admin-radios")), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-radios__label")), gosx.Text("Days")), gosx.Fragment(days...)),
		adminTextField("bookStart", "From", rules.Start, "24-hour time, such as 09:00."),
		adminTextField("bookEnd", "Until", rules.End, ""),
		adminTextField("bookSlot", "Each booking lasts (minutes)", strconv.Itoa(rules.SlotMinutes), ""),
		adminTextField("bookLead", "Book at least this many hours ahead", strconv.Itoa(rules.LeadHours), ""),
		adminTextField("bookWeeks", "Book up to this many weeks ahead", strconv.Itoa(rules.WeeksAhead), ""),
	)
}

func bookingRulesFromForm(r *http.Request) BookingRules {
	rules := BookingRules{Start: r.PostFormValue("bookStart"), End: r.PostFormValue("bookEnd")}
	for _, raw := range r.PostForm["bookDay"] {
		if day, err := strconv.Atoi(raw); err == nil {
			rules.Days = append(rules.Days, day)
		}
	}
	rules.SlotMinutes, _ = strconv.Atoi(strings.TrimSpace(r.PostFormValue("bookSlot")))
	rules.LeadHours, _ = strconv.Atoi(strings.TrimSpace(r.PostFormValue("bookLead")))
	rules.WeeksAhead, _ = strconv.Atoi(strings.TrimSpace(r.PostFormValue("bookWeeks")))
	return normalizeBookingRules(rules)
}
