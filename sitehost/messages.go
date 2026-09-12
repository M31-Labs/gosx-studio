package sitehost

import (
	"encoding/json"
	"errors"
	"net"
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
	"m31labs.dev/gosx-studio/cms/render"
)

// messages.go is the contact form and the inbox behind it.
//
// The audit's sharpest business finding was that a site owner could not
// create a form, could not place one, and could not say where submissions
// went — and that no email of any kind existed, so a lead would sit unseen.
// The default host ships one form that every site needs, places it on the
// contact page the wizard builds, and lands every message in an inbox the
// owner sees the moment they open the admin area. No mail server required.
//
// The form is Studio's own "flow" block with flowKey "contact", rendered
// through cms/render's Flow hook, so a page's document stays in Studio's
// vocabulary and any future host can render the same block its own way.

const (
	contactFlowKey     = "contact"
	contactSendPath    = "/contact/send"
	contactFormAnchor  = "contact-form"
	messagesRatePer    = 5
	messagesRateWindow = 10 * time.Minute
	messageMaxLen      = 4000
)

// Message is one thing a visitor sent through the contact form.
type Message struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Body     string    `json:"body"`
	Page     string    `json:"page,omitempty"`
	Received time.Time `json:"received"`
	Read     bool      `json:"read"`
	// Owner-built forms: which one, and every answer in order.
	Form     string       `json:"form,omitempty"`
	FormName string       `json:"formName,omitempty"`
	Fields   []FieldValue `json:"fields,omitempty"`
}

// messageStore keeps messages in one JSON file beside the site data. Volumes
// here are small — a busy small business gets a few a day — so the whole
// list is rewritten on every change under a mutex, which is the simplest
// thing that cannot lose a message.
type messageStore struct {
	mu       sync.Mutex
	path     string
	messages []Message
	loaded   bool
	// rate limiting: sends per remote host within the window
	hits map[string][]time.Time
}

func newMessageStore(path string) *messageStore {
	return &messageStore{path: path, hits: map[string][]time.Time{}}
}

func (s *messageStore) loadLocked() error {
	if s.loaded {
		return nil
	}
	s.loaded = true
	if s.path == "" {
		return nil
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.messages)
}

func (s *messageStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.messages, "", "  ")
	if err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temp, s.path)
}

func (s *messageStore) add(message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	message.ID = "msg_" + strconv.FormatInt(message.Received.UnixNano(), 36)
	s.messages = append(s.messages, message)
	return s.saveLocked()
}

// list returns messages newest first.
func (s *messageStore) list() ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return nil, err
	}
	out := append([]Message(nil), s.messages...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Received.After(out[j].Received) })
	return out, nil
}

func (s *messageStore) unread() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return 0
	}
	count := 0
	for _, message := range s.messages {
		if !message.Read {
			count++
		}
	}
	return count
}

func (s *messageStore) markRead(id string, read bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadLocked(); err != nil {
		return err
	}
	for index := range s.messages {
		if s.messages[index].ID == id {
			s.messages[index].Read = read
			return s.saveLocked()
		}
	}
	return errors.New("message not found")
}

// allow reports whether one more send from this host fits the rate limit.
// It is deliberately small: enough to stop a script from filling the inbox,
// not a substitute for a real abuse system behind a proxy.
func (s *messageStore) allow(host string, now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := now.Add(-messagesRateWindow)
	kept := s.hits[host][:0]
	for _, hit := range s.hits[host] {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= messagesRatePer {
		s.hits[host] = kept
		return false
	}
	s.hits[host] = append(kept, now)
	return true
}

// messagesPath is where the inbox is stored: beside the site data.
func (o Options) messagesPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "messages.json")
}

func (h *Host) mountMessages(mux *http.ServeMux) {
	mux.HandleFunc("POST "+contactSendPath, h.handleContactSend)
	mux.HandleFunc("GET /admin/messages", h.handleAdminMessages)
	mux.HandleFunc("GET /admin/messages/{$}", h.handleAdminMessages)
	mux.HandleFunc("POST /admin/messages/{id}/read", h.handleAdminMessageRead)
}

// ---------- the public form ----------

// formState is what the form shows above itself after a submit.
type formState struct {
	Sent  bool
	Error string
	Form  string // which form the state is about; "" is the contact form
}

func formStateFromQuery(r *http.Request) formState {
	state := formStateBase(r)
	state.Form = strings.TrimSpace(r.URL.Query().Get("form"))
	return state
}

func formStateBase(r *http.Request) formState {
	which := strings.TrimSpace(r.URL.Query().Get("which"))
	switch r.URL.Query().Get("sent") {
	case "1":
		return formState{Sent: true}
	case "error":
		switch r.URL.Query().Get("why") {
		case "field":
			return formState{Error: "Please fill in “" + which + "”."}
		case "tick":
			return formState{Error: "Please tick “" + which + "” to continue."}
		case "choice":
			return formState{Error: "Pick one of the choices for “" + which + "”."}
		case "email":
			return formState{Error: "That email address doesn't look right. Check it and try again."}
		case "message":
			return formState{Error: "Add a message before sending."}
		case "name":
			return formState{Error: "Add your name so we know who to reply to."}
		case "rate":
			return formState{Error: "You've sent a few messages in a row. Give it a few minutes and try again."}
		default:
			return formState{Error: "Something went wrong sending that. Try again."}
		}
	}
	return formState{}
}

// flowHook renders Studio's "flow" block. Only the contact flow is known
// here; anything else renders nothing rather than a placeholder a visitor
// would see.
func (h *Host) flowHook(pagePath string, state formState) render.Hook {
	return func(ctx render.Context) (gosx.Node, bool) {
		if form, ok := h.formByRef(ctx.Ref); ok {
			return h.renderCustomForm(form, pagePath, state), true
		}
		if strings.TrimSpace(ctx.Ref) != contactFlowKey {
			return gosx.Fragment(), true
		}
		if state.Form != "" {
			state = formState{}
		}
		return renderContactForm(pagePath, state), true
	}
}

func renderContactForm(pagePath string, state formState) gosx.Node {
	if state.Sent {
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-form site-form--sent"), gosx.Attr("id", contactFormAnchor), gosx.Attr("role", "status")),
			gosx.El("h3", nil, gosx.Text("Thanks — your message is on its way")),
			gosx.El("p", nil, gosx.Text("We'll get back to you as soon as we can.")),
		)
	}
	nodes := []gosx.Node{}
	if state.Error != "" {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "alert")), gosx.Text(state.Error)))
	}
	nodes = append(nodes,
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "page"), gosx.Attr("value", pagePath))),
		formField("name", "Your name", "text", "", 120),
		formField("email", "Your email", "email", "", 200),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")),
			gosx.El("span", nil, gosx.Text("Message")),
			gosx.El("textarea", gosx.Attrs(
				gosx.Attr("name", "message"), gosx.Attr("required", "required"),
				gosx.Attr("rows", "5"), gosx.Attr("maxlength", strconv.Itoa(messageMaxLen)),
			)),
		),
		// The honeypot: a field people never see and bots fill in. A
		// submission with anything in it is accepted and thrown away.
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-form__hp"), gosx.Attr("aria-hidden", "true")),
			gosx.El("label", nil, gosx.Text("Leave this field empty"),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "website"), gosx.Attr("tabindex", "-1"), gosx.Attr("autocomplete", "off")))),
		),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("type", "submit")), gosx.Text("Send message")),
	)
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "site-form"), gosx.Attr("id", contactFormAnchor),
		gosx.Attr("method", "post"), gosx.Attr("action", contactSendPath),
	), gosx.Fragment(nodes...))
}

func formField(name, label, kind, value string, max int) gosx.Node {
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")),
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", kind), gosx.Attr("name", name), gosx.Attr("value", value),
			gosx.Attr("required", "required"), gosx.Attr("maxlength", strconv.Itoa(max)),
		)),
	)
}

// safeReturnPath keeps the post-submit redirect on this site.
func safeReturnPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, "\r\n") {
		return "/"
	}
	if index := strings.IndexAny(value, "?#"); index >= 0 {
		value = value[:index]
	}
	return value
}

func remoteHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *Host) handleContactSend(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/?sent=error", http.StatusSeeOther)
		return
	}
	back := safeReturnPath(r.PostFormValue("page"))
	fail := func(why string) {
		http.Redirect(w, r, back+"?sent=error&why="+why+"#"+contactFormAnchor, http.StatusSeeOther)
	}

	// Honeypot filled: pretend it worked, store nothing.
	if strings.TrimSpace(r.PostFormValue("website")) != "" {
		http.Redirect(w, r, back+"?sent=1#"+contactFormAnchor, http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.PostFormValue("name"))
	email := strings.TrimSpace(r.PostFormValue("email"))
	body := strings.TrimSpace(r.PostFormValue("message"))
	switch {
	case name == "":
		fail("name")
		return
	case body == "":
		fail("message")
		return
	}
	if _, err := mail.ParseAddress(email); err != nil || strings.ContainsAny(email, " <>") {
		fail("email")
		return
	}
	if len(body) > messageMaxLen {
		body = body[:messageMaxLen]
	}
	now := time.Now().UTC()
	if !h.messages.allow(remoteHost(r), now) {
		fail("rate")
		return
	}
	message := Message{Name: name, Email: email, Body: body, Page: back, Received: now}
	if err := h.messages.add(message); err != nil {
		fail("save")
		return
	}
	h.notify(h.newMessageMail(message, h.absoluteBase(r)))
	http.Redirect(w, r, back+"?sent=1#"+contactFormAnchor, http.StatusSeeOther)
}

// ---------- the inbox ----------

func (h *Host) unreadMessages() int {
	if h.messages == nil {
		return 0
	}
	return h.messages.unread()
}

func (h *Host) handleAdminMessages(w http.ResponseWriter, r *http.Request) {
	messages, err := h.messages.list()
	status := adminStatus{Message: r.URL.Query().Get("status")}
	if err != nil {
		status = adminStatus{Message: "We couldn't read your messages. Try again.", Error: true}
	}

	var listing gosx.Node
	if len(messages) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No messages yet")),
			gosx.El("p", nil, gosx.Text("When someone fills in the contact form on your site, their message lands here. There's a form on your contact page already.")),
		)
	} else {
		items := make([]gosx.Node, 0, len(messages))
		for _, message := range messages {
			items = append(items, h.renderMessage(message))
		}
		listing = gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-messages")), gosx.Fragment(items...))
	}

	body := h.renderAdminShell("messages", "Messages",
		"Everything visitors have sent through your forms, newest first.",
		status, listing)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Messages"), body)
}

func (h *Host) renderMessage(message Message) gosx.Node {
	state := "unread"
	toggleLabel, toggleValue := "Mark as read", "1"
	if message.Read {
		state = "read"
		toggleLabel, toggleValue = "Mark as unread", "0"
	}
	when := message.Received.Local().Format("Mon 2 Jan, 3:04 PM")
	var body gosx.Node = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-message__body")), gosx.Text(message.Body))
	if len(message.Fields) > 0 {
		rows := make([]gosx.Node, 0, len(message.Fields))
		for _, field := range message.Fields {
			rows = append(rows, gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-message__field")),
				gosx.El("dt", nil, gosx.Text(field.Label)), gosx.El("dd", nil, gosx.Text(firstNonEmpty(field.Value, "—")))))
		}
		body = gosx.El("dl", gosx.Attrs(gosx.Attr("class", "admin-message__fields")), gosx.Fragment(rows...))
	}
	var formBadge gosx.Node = gosx.Fragment()
	if message.FormName != "" {
		formBadge = gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge")), gosx.Text(message.FormName))
	}
	var emailLink gosx.Node = gosx.Fragment()
	if message.Email != "" {
		emailLink = gosx.El("a", gosx.Attrs(gosx.Attr("href", "mailto:"+message.Email)), gosx.Text(message.Email))
	}
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", "admin-message"), gosx.Attr("data-state", state)),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "admin-message__head")),
			gosx.El("strong", nil, gosx.Text(message.Name)),
			formBadge,
			emailLink,
			gosx.El("time", gosx.Attrs(gosx.Attr("datetime", message.Received.Format(time.RFC3339))), gosx.Text(when)),
		),
		body,
		gosx.El("footer", gosx.Attrs(gosx.Attr("class", "admin-message__actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "mailto:"+message.Email+"?subject="+replySubject(message))), gosx.Text("Reply by email")),
			gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/messages/"+message.ID+"/read"), gosx.Attr("class", "admin-inline-form")),
				h.csrfField(),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "read"), gosx.Attr("value", toggleValue))),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text(toggleLabel)),
			),
		),
	)
}

func replySubject(message Message) string {
	return "Re:%20your%20message"
}

func (h *Host) handleAdminMessageRead(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	read := r.PostFormValue("read") != "0"
	if err := h.messages.markRead(r.PathValue("id"), read); err != nil {
		http.Redirect(w, r, "/admin/messages?status="+queryEscape("We couldn't find that message."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/messages", http.StatusSeeOther)
}
