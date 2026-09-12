package sitehost

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
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

// forms.go is the form builder: the owner names a form, adds fields, and
// drops it on any page. Submissions land in Messages, go out by email and
// (optionally) to a webhook, and export as a spreadsheet.
//
// The built-in contact form stays exactly as it was; owner-built forms sit
// beside it and are referenced from a page by "form:<id>" in the same flow
// block Studio already has. The field vocabulary — short answer, long
// answer, email, phone, date, choice, tick box — is the one a person who has
// used any form tool expects, and nothing here needs JavaScript to build.

const (
	formSendPrefix = "/forms/"
	formRefPrefix  = "form:"
	formMaxFields  = 20
	formValueMax   = 2000
)

// FormField is one question on a form.
type FormField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

// Form is one owner-built form.
type Form struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Fields      []FormField    `json:"fields"`
	Button      string         `json:"button"`
	Thanks      string         `json:"thanks"`
	NotifyEmail string         `json:"notifyEmail,omitempty"`
	WebhookURL  string         `json:"webhookUrl,omitempty"`
	Created     time.Time      `json:"created"`
	Updated     time.Time      `json:"updated"`
	LastWebhook *webhookResult `json:"lastWebhook,omitempty"`
}

type webhookResult struct {
	At    time.Time `json:"at"`
	OK    bool      `json:"ok"`
	Error string    `json:"error,omitempty"`
}

// FieldValue is one answer in a submission, in the form's order.
type FieldValue struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type fieldKind struct{ Key, Label string }

func fieldKinds() []fieldKind {
	return []fieldKind{
		{"text", "Short answer"}, {"textarea", "Long answer"}, {"email", "Email address"},
		{"phone", "Phone number"}, {"date", "Date"}, {"select", "Choice from a list"}, {"checkbox", "Tick box"},
	}
}

func normalizeFieldKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	for _, known := range fieldKinds() {
		if known.Key == kind {
			return kind
		}
	}
	return "text"
}

func (f Form) ref() string { return formRefPrefix + f.ID }

func (f Form) anchor() string { return "form-" + f.ID }

// normalizeForm tidies labels, keys, kinds, and options, and drops fields
// with no label.
func normalizeForm(form Form) Form {
	form.Name = strings.TrimSpace(form.Name)
	if form.Name == "" {
		form.Name = "Form"
	}
	form.Button = firstNonEmpty(strings.TrimSpace(form.Button), "Send")
	form.Thanks = firstNonEmpty(strings.TrimSpace(form.Thanks), "Thanks — we've got it.")
	form.NotifyEmail = strings.TrimSpace(form.NotifyEmail)
	form.WebhookURL = strings.TrimSpace(form.WebhookURL)
	fields := make([]FormField, 0, len(form.Fields))
	seen := map[string]bool{}
	for index, field := range form.Fields {
		field.Label = strings.TrimSpace(field.Label)
		if field.Label == "" {
			continue
		}
		field.Kind = normalizeFieldKind(field.Kind)
		key := normalizeSlug(field.Label)
		if key == "" {
			key = "field"
		}
		if seen[key] {
			key = key + "-" + strconv.Itoa(index+1)
		}
		seen[key] = true
		field.Key = key
		options := make([]string, 0, len(field.Options))
		for _, option := range field.Options {
			if option = strings.TrimSpace(option); option != "" {
				options = append(options, option)
			}
		}
		if field.Kind == "select" {
			field.Options = options
		} else {
			field.Options = nil
		}
		fields = append(fields, field)
		if len(fields) == formMaxFields {
			break
		}
	}
	form.Fields = fields
	return form
}

// ---------- templates ----------

type formTemplate struct {
	Key, Label, Blurb string
	Form              Form
}

func formTemplates() []formTemplate {
	return []formTemplate{
		{Key: "contact", Label: "Contact", Blurb: "Name, email, message.", Form: Form{Name: "Contact", Button: "Send message", Thanks: "Thanks — your message is on its way.", Fields: []FormField{
			{Label: "Your name", Kind: "text", Required: true}, {Label: "Your email", Kind: "email", Required: true}, {Label: "Message", Kind: "textarea", Required: true}}}},
		{Key: "newsletter", Label: "Newsletter sign-up", Blurb: "An email address and a tick box. Export the list any time.", Form: Form{Name: "Newsletter", Button: "Sign me up", Thanks: "You're on the list.", Fields: []FormField{
			{Label: "Your email", Kind: "email", Required: true}, {Label: "Send me occasional news and offers", Kind: "checkbox", Required: true}}}},
		{Key: "booking", Label: "Booking request", Blurb: "Name, contact details, a date, and what they'd like.", Form: Form{Name: "Booking request", Button: "Request a booking", Thanks: "Thanks — we'll confirm by email.", Fields: []FormField{
			{Label: "Your name", Kind: "text", Required: true}, {Label: "Your email", Kind: "email", Required: true}, {Label: "Phone number", Kind: "phone"},
			{Label: "Date", Kind: "date", Required: true}, {Label: "How many people", Kind: "select", Options: []string{"1", "2", "3", "4", "5", "6 or more"}}, {Label: "Anything we should know?", Kind: "textarea"}}}},
		{Key: "blank", Label: "Start from scratch", Blurb: "Just a name and an email field to begin with.", Form: Form{Name: "New form", Button: "Send", Fields: []FormField{
			{Label: "Your name", Kind: "text", Required: true}, {Label: "Your email", Kind: "email", Required: true}}}},
	}
}

func formTemplateByKey(key string) formTemplate {
	for _, template := range formTemplates() {
		if template.Key == key {
			return template
		}
	}
	return formTemplates()[0]
}

// ---------- the store ----------

type formStore struct {
	mu     sync.Mutex
	path   string
	loaded bool
	forms  []Form
}

func newFormStore(path string) *formStore { return &formStore{path: path} }

func (o Options) formsPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "forms.json")
}

func (s *formStore) loadLocked() {
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
		Forms []Form `json:"forms"`
	}
	if json.Unmarshal(raw, &file) == nil {
		s.forms = file.Forms
	}
}

func (s *formStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(struct {
		Forms []Form `json:"forms"`
	}{s.forms}, "", "  ")
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

func (s *formStore) list() []Form {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]Form, len(s.forms))
	copy(out, s.forms)
	return out
}

func (s *formStore) get(id string) (Form, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, form := range s.forms {
		if form.ID == id {
			return form, true
		}
	}
	return Form{}, false
}

var errFormNotFound = errors.New("form not found")

// put inserts or replaces a form.
func (s *formStore) put(form Form) (Form, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	now := timeNow().UTC()
	form = normalizeForm(form)
	form.Updated = now
	if form.ID == "" {
		form.ID = "f" + randomHex(4)
		form.Created = now
		s.forms = append(s.forms, form)
		return form, s.saveLocked()
	}
	for index, existing := range s.forms {
		if existing.ID == form.ID {
			form.Created = existing.Created
			form.LastWebhook = existing.LastWebhook
			s.forms[index] = form
			return form, s.saveLocked()
		}
	}
	return Form{}, errFormNotFound
}

func (s *formStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index, existing := range s.forms {
		if existing.ID == id {
			s.forms = append(s.forms[:index], s.forms[index+1:]...)
			return s.saveLocked()
		}
	}
	return errFormNotFound
}

func (s *formStore) recordWebhook(id string, result webhookResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index := range s.forms {
		if s.forms[index].ID == id {
			s.forms[index].LastWebhook = &result
			_ = s.saveLocked()
			return
		}
	}
}

// formByRef resolves a flow block's reference to an owner-built form.
func (h *Host) formByRef(ref string) (Form, bool) {
	ref = strings.TrimSpace(ref)
	if !strings.HasPrefix(ref, formRefPrefix) {
		return Form{}, false
	}
	return h.forms.get(strings.TrimPrefix(ref, formRefPrefix))
}

// ---------- routes ----------

func (h *Host) mountForms(mux *http.ServeMux) {
	mux.HandleFunc("POST "+formSendPrefix+"{id}/send", h.handleFormSend)
	mux.HandleFunc("GET /admin/forms", h.handleAdminForms)
	mux.HandleFunc("GET /admin/forms/{$}", h.handleAdminForms)
	mux.HandleFunc("POST /admin/forms", h.handleAdminCreateForm)
	mux.HandleFunc("POST /admin/forms/{$}", h.handleAdminCreateForm)
	mux.HandleFunc("GET /admin/forms/{id}", h.handleAdminFormBuilder)
	mux.HandleFunc("POST /admin/forms/{id}", h.handleAdminFormAction)
	mux.HandleFunc("POST /admin/forms/{id}/delete", h.handleAdminFormDelete)
	mux.HandleFunc("GET /admin/forms/{id}/export.csv", h.handleAdminFormExport)
}

// ---------- the public form ----------

func (h *Host) renderCustomForm(form Form, pagePath string, state formState) gosx.Node {
	if state.Form != form.ID {
		state = formState{}
	}
	if state.Sent {
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-form site-form--sent"), gosx.Attr("id", form.anchor()), gosx.Attr("role", "status")),
			gosx.El("h3", nil, gosx.Text(form.Thanks)),
		)
	}
	nodes := []gosx.Node{}
	if state.Error != "" {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "alert")), gosx.Text(state.Error)))
	}
	nodes = append(nodes, gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "page"), gosx.Attr("value", pagePath))))
	for _, field := range form.Fields {
		nodes = append(nodes, renderFormFieldInput(field, "", false))
	}
	nodes = append(nodes,
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-form__hp"), gosx.Attr("aria-hidden", "true")),
			gosx.El("label", nil, gosx.Text("Leave this field empty"),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "website"), gosx.Attr("tabindex", "-1"), gosx.Attr("autocomplete", "off")))),
		),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("type", "submit")), gosx.Text(form.Button)),
	)
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "site-form"), gosx.Attr("id", form.anchor()),
		gosx.Attr("method", "post"), gosx.Attr("action", formSendPrefix+form.ID+"/send"),
	), gosx.Fragment(nodes...))
}

// renderFormFieldInput draws one question. disabled makes the canvas preview.
func renderFormFieldInput(field FormField, value string, disabled bool) gosx.Node {
	name := "f_" + field.Key
	common := []any{gosx.Attr("name", name)}
	if field.Required && !disabled {
		common = append(common, gosx.Attr("required", "required"))
	}
	if disabled {
		common = append(common, gosx.Attr("disabled", "disabled"))
	}
	label := field.Label
	if field.Required {
		label += " *"
	}
	switch field.Kind {
	case "textarea":
		return gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")), gosx.El("span", nil, gosx.Text(label)),
			gosx.El("textarea", gosx.Attrs(append(common, gosx.Attr("rows", "5"), gosx.Attr("maxlength", strconv.Itoa(formValueMax)))...), gosx.Text(value)))
	case "select":
		options := make([]gosx.Node, 0, len(field.Options)+1)
		options = append(options, gosx.El("option", gosx.Attrs(gosx.Attr("value", "")), gosx.Text("Choose…")))
		for _, option := range field.Options {
			attrs := []any{gosx.Attr("value", option)}
			if option == value {
				attrs = append(attrs, gosx.Attr("selected", "selected"))
			}
			options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(option)))
		}
		return gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")), gosx.El("span", nil, gosx.Text(label)),
			gosx.El("select", gosx.Attrs(common...), gosx.Fragment(options...)))
	case "checkbox":
		attrs := append(common, gosx.Attr("type", "checkbox"), gosx.Attr("value", "yes"))
		if value == "yes" {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		return gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__check")),
			gosx.El("input", gosx.Attrs(attrs...)), gosx.El("span", nil, gosx.Text(label)))
	}
	inputType := map[string]string{"email": "email", "phone": "tel", "date": "date"}[field.Kind]
	if inputType == "" {
		inputType = "text"
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")), gosx.El("span", nil, gosx.Text(label)),
		gosx.El("input", gosx.Attrs(append(common, gosx.Attr("type", inputType), gosx.Attr("value", value), gosx.Attr("maxlength", strconv.Itoa(formValueMax)))...)))
}

func (h *Host) handleFormSend(w http.ResponseWriter, r *http.Request) {
	form, ok := h.forms.get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/?sent=error&form="+form.ID, http.StatusSeeOther)
		return
	}
	back := safeReturnPath(r.PostFormValue("page"))
	fail := func(why, which string) {
		http.Redirect(w, r, back+"?sent=error&form="+form.ID+"&why="+why+"&which="+url.QueryEscape(which)+"#"+form.anchor(), http.StatusSeeOther)
	}
	if strings.TrimSpace(r.PostFormValue("website")) != "" {
		http.Redirect(w, r, back+"?sent=1&form="+form.ID+"#"+form.anchor(), http.StatusSeeOther)
		return
	}

	values := make([]FieldValue, 0, len(form.Fields))
	name, email := "", ""
	var body strings.Builder
	for _, field := range form.Fields {
		value := strings.TrimSpace(r.PostFormValue("f_" + field.Key))
		if len(value) > formValueMax {
			value = value[:formValueMax]
		}
		switch field.Kind {
		case "checkbox":
			if value != "" {
				value = "yes"
			}
			if field.Required && value == "" {
				fail("tick", field.Label)
				return
			}
		case "select":
			if value != "" && !containsString(field.Options, value) {
				fail("choice", field.Label)
				return
			}
		case "email":
			if value != "" {
				if _, err := mail.ParseAddress(value); err != nil || strings.ContainsAny(value, " <>") {
					fail("email", field.Label)
					return
				}
				if email == "" {
					email = value
				}
			}
		case "text":
			if name == "" && strings.Contains(strings.ToLower(field.Label), "name") {
				name = value
			}
		}
		if field.Required && value == "" {
			fail("field", field.Label)
			return
		}
		values = append(values, FieldValue{Key: field.Key, Label: field.Label, Value: value})
		if value != "" {
			body.WriteString(field.Label + ": " + value + "\n")
		}
	}

	now := timeNow().UTC()
	if !h.messages.allow(remoteHost(r), now) {
		fail("rate", "")
		return
	}
	message := Message{
		Name: firstNonEmpty(name, email, "Someone"), Email: email, Body: strings.TrimSpace(body.String()),
		Page: back, Received: now, Form: form.ID, FormName: form.Name, Fields: values,
	}
	if err := h.messages.add(message); err != nil {
		fail("save", "")
		return
	}
	h.notify(h.newFormMail(form, message, h.absoluteBase(r)))
	h.deliverWebhook(form, message, h.absoluteBase(r))
	http.Redirect(w, r, back+"?sent=1&form="+form.ID+"#"+form.anchor(), http.StatusSeeOther)
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func (h *Host) newFormMail(form Form, message Message, base string) Mail {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	var b strings.Builder
	b.WriteString("Someone filled in the “" + form.Name + "” form on your website.\n\n")
	b.WriteString(message.Body + "\n\n")
	if message.Page != "" {
		b.WriteString("Page: " + base + message.Page + "\n")
	}
	b.WriteString("See all submissions at " + base + "/admin/messages\n")
	m := Mail{
		To:      firstNonEmpty(form.NotifyEmail, h.notifyAddress()),
		Subject: form.Name + ": new submission from " + message.Name + " — " + siteTitle,
		Text:    b.String(),
	}
	if message.Email != "" {
		m.ReplyTo = message.Email
	}
	return m
}

// allowInsecureWebhooks lets tests point a webhook at a plain http server.
var allowInsecureWebhooks = false

// webhookClient posts submissions. Tests replace it.
var webhookClient = &http.Client{Timeout: 10 * time.Second}

// deliverWebhook posts the submission as JSON, in the background, and
// remembers whether it worked so the builder page can say so.
func (h *Host) deliverWebhook(form Form, message Message, base string) {
	target := strings.TrimSpace(form.WebhookURL)
	if target == "" {
		return
	}
	parsed, err := url.Parse(target)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(allowInsecureWebhooks && parsed.Scheme == "http")) {
		h.forms.recordWebhook(form.ID, webhookResult{At: timeNow().UTC(), Error: "The webhook address must start with https://"})
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"form":     map[string]string{"id": form.ID, "name": form.Name},
		"received": message.Received.Format(time.RFC3339),
		"page":     base + message.Page,
		"name":     message.Name,
		"email":    message.Email,
		"fields":   message.Fields,
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
		if err != nil {
			h.forms.recordWebhook(form.ID, webhookResult{At: timeNow().UTC(), Error: err.Error()})
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "gosx-site")
		res, err := webhookClient.Do(req)
		if err != nil {
			h.forms.recordWebhook(form.ID, webhookResult{At: timeNow().UTC(), Error: "Couldn't reach it: " + err.Error()})
			return
		}
		res.Body.Close()
		if res.StatusCode >= 300 {
			h.forms.recordWebhook(form.ID, webhookResult{At: timeNow().UTC(), Error: "It answered " + res.Status})
			return
		}
		h.forms.recordWebhook(form.ID, webhookResult{At: timeNow().UTC(), OK: true})
	}()
	if webhookWait {
		<-done
	}
}

// webhookWait makes delivery synchronous. Tests set it.
var webhookWait = false

// ---------- admin: the list ----------

func (h *Host) submissionsByForm() map[string]int {
	counts := map[string]int{}
	messages, _ := h.messages.list()
	for _, message := range messages {
		if message.Form != "" {
			counts[message.Form]++
		}
	}
	return counts
}

func (h *Host) handleAdminForms(w http.ResponseWriter, r *http.Request) {
	h.renderAdminForms(w, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderAdminForms(w http.ResponseWriter, status adminStatus) {
	forms := h.forms.list()
	counts := h.submissionsByForm()

	var listing gosx.Node
	if len(forms) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No forms of your own yet")),
			gosx.El("p", nil, gosx.Text("Your contact page already has a contact form. Build another for sign-ups, bookings, quotes, or anything else, then add it to a page with the Form block.")),
		)
	} else {
		rows := make([]gosx.Node, 0, len(forms))
		for _, form := range forms {
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/forms/"+form.ID)), gosx.Text(form.Name))),
				gosx.El("td", nil, gosx.Text(plural(len(form.Fields), "field"))),
				gosx.El("td", nil, gosx.Text(plural(counts[form.ID], "submission"))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
					gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", "/admin/forms/"+form.ID)), gosx.Text("Edit")),
					gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", "/admin/forms/"+form.ID+"/export.csv")), gosx.Text("Export CSV")),
				),
			))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your forms")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Add any of these to a page with the Form block in the editor. Submissions arrive in Messages.")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Form")), gosx.El("th", nil, gosx.Text("Fields")), gosx.El("th", nil, gosx.Text("Submissions")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...)),
			),
		)
	}

	choices := make([]gosx.Node, 0, 4)
	for index, template := range formTemplates() {
		attrs := []any{gosx.Attr("type", "radio"), gosx.Attr("name", "template"), gosx.Attr("value", template.Key), gosx.Attr("id", "template-"+template.Key)}
		if index == 0 {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		choices = append(choices, gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-choice"), gosx.Attr("for", "template-"+template.Key)),
			gosx.El("input", gosx.Attrs(attrs...)),
			gosx.El("strong", nil, gosx.Text(template.Label)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(template.Blurb)),
		))
	}
	create := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Build a form")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/forms")),
			h.csrfField(),
			adminTextField("name", "What is it for?", "", "For example \"Newsletter\", \"Table bookings\", or \"Quote request\". You can rename it later."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-choices")), gosx.Fragment(choices...)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Create form")),
			),
		),
	)

	body := h.renderAdminShell("forms", "Forms",
		"Sign-ups, bookings, quotes: build a form, put it on a page, and read what comes in.",
		status, listing, create)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Forms"), body)
}

func (h *Host) handleAdminCreateForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAdminForms(w, adminStatus{Message: "We couldn't read that form. Try again.", Error: true})
		return
	}
	template := formTemplateByKey(r.PostFormValue("template"))
	form := template.Form
	if name := strings.TrimSpace(r.PostFormValue("name")); name != "" {
		form.Name = name
	}
	saved, err := h.forms.put(form)
	if err != nil {
		h.renderAdminForms(w, adminStatus{Message: "We couldn't create that form. Try again.", Error: true})
		return
	}
	http.Redirect(w, r, "/admin/forms/"+saved.ID+"?status="+queryEscape("Created. Change the questions below, then add the form to a page."), http.StatusSeeOther)
}

// ---------- admin: the builder ----------

func (h *Host) handleAdminFormBuilder(w http.ResponseWriter, r *http.Request) {
	form, ok := h.forms.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "form")
		return
	}
	h.renderFormBuilder(w, form, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderFormBuilder(w http.ResponseWriter, form Form, status adminStatus) {
	rows := make([]gosx.Node, 0, len(form.Fields)+1)
	for index, field := range form.Fields {
		n := strconv.Itoa(index)
		kinds := make([]gosx.Node, 0, 7)
		for _, kind := range fieldKinds() {
			attrs := []any{gosx.Attr("value", kind.Key)}
			if kind.Key == field.Kind {
				attrs = append(attrs, gosx.Attr("selected", "selected"))
			}
			kinds = append(kinds, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(kind.Label)))
		}
		requiredAttrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "required_"+n), gosx.Attr("value", "1"), gosx.Attr("aria-label", "Required")}
		if field.Required {
			requiredAttrs = append(requiredAttrs, gosx.Attr("checked", "checked"))
		}
		rows = append(rows, gosx.El("tr", gosx.Attrs(gosx.Attr("class", "admin-field-row")),
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "label_"+n), gosx.Attr("value", field.Label), gosx.Attr("aria-label", "Question"), gosx.Attr("maxlength", "120")))),
			gosx.El("td", nil, gosx.El("select", gosx.Attrs(gosx.Attr("name", "kind_"+n), gosx.Attr("aria-label", "Answer type")), gosx.Fragment(kinds...))),
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "options_"+n), gosx.Attr("value", strings.Join(field.Options, ", ")), gosx.Attr("placeholder", "For a list: Small, Medium, Large"), gosx.Attr("aria-label", "Choices")))),
			gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-field-row__required")), gosx.El("input", gosx.Attrs(requiredAttrs...))),
			gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("name", "remove"), gosx.Attr("value", n), gosx.Attr("aria-label", "Remove this question")), gosx.Text("✕"))),
		))
	}

	webhookNote := gosx.Fragment()
	if form.LastWebhook != nil {
		state, text := "published", "Last delivery worked ("+formatWhen(form.LastWebhook.At)+")."
		if !form.LastWebhook.OK {
			state, text = "offline", "Last delivery failed ("+formatWhen(form.LastWebhook.At)+"): "+form.LastWebhook.Error
		}
		webhookNote = gosx.El("p", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(text)))
	}

	builder := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Questions")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/forms/"+form.ID)),
			h.csrfField(),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "count"), gosx.Attr("value", strconv.Itoa(len(form.Fields))))),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-fields")),
				gosx.El("thead", nil, gosx.El("tr", nil,
					gosx.El("th", nil, gosx.Text("Question")), gosx.El("th", nil, gosx.Text("Answer type")), gosx.El("th", nil, gosx.Text("Choices")), gosx.El("th", nil, gosx.Text("Required")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...)),
			),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit"), gosx.Attr("name", "add"), gosx.Attr("value", "1")), gosx.Text("Add a question")),
			),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Details")),
			adminTextField("name", "Form name", form.Name, "How it appears in your lists. Visitors don't see it."),
			adminTextField("button", "Button text", form.Button, "What the send button says."),
			adminTextField("thanks", "Thank-you message", form.Thanks, "Shown after someone sends the form."),
			adminTextField("notifyEmail", "Email new submissions to", form.NotifyEmail, "Leave blank to use the address in Settings."),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Send submissions somewhere else (optional)")),
			adminTextField("webhookUrl", "Webhook address", form.WebhookURL, "For tools like Zapier, Make, or your own system: each submission is posted there as JSON. Must start with https://."),
			webhookNote,
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit"), gosx.Attr("name", "save"), gosx.Attr("value", "1")), gosx.Text("Save form")),
			),
		),
	)

	counts := h.submissionsByForm()
	use := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Use it")),
		gosx.El("p", nil, gosx.Text("Open a page in the editor, add a Form block, and choose “"+form.Name+"”.")),
		gosx.El("p", nil, gosx.Text(plural(counts[form.ID], "submission")+" so far.")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/messages")), gosx.Text("Read submissions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/forms/"+form.ID+"/export.csv")), gosx.Text("Export as a spreadsheet (CSV)")),
		),
	)
	remove := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Delete")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Removes the form from every page it's on. Submissions you already received stay in Messages.")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/forms/"+form.ID+"/delete")),
			h.csrfField(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Delete this form")),
		),
	)

	body := h.renderAdminShell("forms", form.Name, "Change the questions, then save. The form updates everywhere it's used.", status, builder, use, remove)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Form: "+form.Name), body)
}

func (h *Host) handleAdminFormAction(w http.ResponseWriter, r *http.Request) {
	form, ok := h.forms.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "form")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderFormBuilder(w, form, adminStatus{Message: "We couldn't read that. Try again.", Error: true})
		return
	}
	count, _ := strconv.Atoi(r.PostFormValue("count"))
	if count < 0 || count > formMaxFields {
		count = 0
	}
	fields := make([]FormField, 0, count+1)
	for index := 0; index < count; index++ {
		n := strconv.Itoa(index)
		if r.PostFormValue("remove") == n {
			continue
		}
		fields = append(fields, FormField{
			Label:    r.PostFormValue("label_" + n),
			Kind:     r.PostFormValue("kind_" + n),
			Required: r.PostFormValue("required_"+n) == "1",
			Options:  strings.Split(r.PostFormValue("options_"+n), ","),
		})
	}
	message := "Saved."
	if r.PostFormValue("add") == "1" {
		if len(fields) >= formMaxFields {
			h.renderFormBuilder(w, form, adminStatus{Message: "That's the most questions a form can have.", Error: true})
			return
		}
		fields = append(fields, FormField{Label: "New question", Kind: "text"})
		message = "Added a question. Give it a label."
	} else if r.PostFormValue("remove") != "" {
		message = "Removed."
	}
	form.Name = r.PostFormValue("name")
	form.Button = r.PostFormValue("button")
	form.Thanks = r.PostFormValue("thanks")
	form.NotifyEmail = r.PostFormValue("notifyEmail")
	form.WebhookURL = r.PostFormValue("webhookUrl")
	form.Fields = fields
	if hook := strings.TrimSpace(form.WebhookURL); hook != "" {
		if parsed, err := url.Parse(hook); err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(allowInsecureWebhooks && parsed.Scheme == "http")) {
			h.renderFormBuilder(w, form, adminStatus{Message: "The webhook address must start with https://.", Error: true})
			return
		}
	}
	saved, err := h.forms.put(form)
	if err != nil {
		h.renderFormBuilder(w, form, adminStatus{Message: "We couldn't save that. Try again.", Error: true})
		return
	}
	if len(saved.Fields) == 0 {
		message = "Saved, but the form has no questions yet. Add one."
	}
	http.Redirect(w, r, "/admin/forms/"+saved.ID+"?status="+queryEscape(message), http.StatusSeeOther)
}

func (h *Host) handleAdminFormDelete(w http.ResponseWriter, r *http.Request) {
	form, ok := h.forms.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "form")
		return
	}
	if err := h.forms.remove(form.ID); err != nil {
		http.Redirect(w, r, "/admin/forms?status="+queryEscape("We couldn't delete that. Try again."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/forms?status="+queryEscape("Deleted “"+form.Name+"”."), http.StatusSeeOther)
}

// handleAdminFormExport is the spreadsheet: one row per submission, one
// column per question, in the form's current order.
func (h *Host) handleAdminFormExport(w http.ResponseWriter, r *http.Request) {
	form, ok := h.forms.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "form")
		return
	}
	messages, _ := h.messages.list()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+normalizeSlug(form.Name)+`-submissions.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	writer := csv.NewWriter(w)
	header := []string{"Received"}
	for _, field := range form.Fields {
		header = append(header, field.Label)
	}
	_ = writer.Write(header)
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message.Form != form.ID {
			continue
		}
		byKey := map[string]string{}
		for _, value := range message.Fields {
			byKey[value.Key] = value.Value
		}
		row := []string{message.Received.UTC().Format(time.RFC3339)}
		for _, field := range form.Fields {
			row = append(row, csvSafe(byKey[field.Key]))
		}
		_ = writer.Write(row)
	}
	writer.Flush()
}

// csvSafe keeps a cell from being read as a formula by a spreadsheet.
func csvSafe(value string) string {
	if value != "" && strings.ContainsAny(value[:1], "=+-@\t\r") {
		return "'" + value
	}
	return value
}

// ---------- the editor's form picker ----------

type formPreset struct {
	Ref    string      `json:"ref"`
	Name   string      `json:"name"`
	Button string      `json:"button"`
	Fields []FormField `json:"fields"`
	Edit   string      `json:"edit"`
}

// formPresets lists every form the editor can place: the contact form and
// the owner's own.
func (h *Host) formPresets() []formPreset {
	contact := formTemplateByKey("contact").Form
	presets := []formPreset{{Ref: contactFlowKey, Name: "Contact form", Button: contact.Button, Fields: contact.Fields, Edit: "/admin/messages"}}
	for _, form := range h.forms.list() {
		presets = append(presets, formPreset{Ref: form.ref(), Name: form.Name, Button: form.Button, Fields: form.Fields, Edit: "/admin/forms/" + form.ID})
	}
	return presets
}

func (h *Host) formPresetsJSON() string {
	data, err := json.Marshal(h.formPresets())
	if err != nil {
		return "[]"
	}
	return strings.ReplaceAll(string(data), "</", "<\\/")
}

// renderFormPreview is a form on the canvas: its questions, disabled, with a
// picker to swap in another form and a link to change the questions.
func (h *Host) renderFormPreview(ref string) gosx.Node {
	ref = strings.TrimSpace(ref)
	presets := h.formPresets()
	var chosen formPreset
	found := false
	for _, preset := range presets {
		if preset.Ref == ref {
			chosen, found = preset, true
			break
		}
	}
	if !found {
		chosen = presets[0]
	}
	options := make([]gosx.Node, 0, len(presets))
	for _, preset := range presets {
		attrs := []any{gosx.Attr("value", preset.Ref)}
		if preset.Ref == chosen.Ref {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(preset.Name)))
	}
	fields := make([]gosx.Node, 0, len(chosen.Fields)+1)
	for _, field := range chosen.Fields {
		fields = append(fields, renderFormFieldInput(field, "", true))
	}
	fields = append(fields, gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("aria-hidden", "true")), gosx.Text(chosen.Button)))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-form"), gosx.Attr("contenteditable", "false")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-form ed-form-preview"), gosx.Attr("data-form-fields", "true")), gosx.Fragment(fields...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-form__bar")),
			gosx.El("label", nil, gosx.Text("Which form: "),
				gosx.El("select", gosx.Attrs(gosx.Attr("class", "ed-inline-select"), gosx.Attr("data-form-select", "true"), gosx.Attr("aria-label", "Which form")), gosx.Fragment(options...))),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", chosen.Edit), gosx.Attr("data-form-edit", "true"), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("Change the questions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/forms"), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("Build a new form")),
		),
	)
}

// formRefForPayload keeps only references the site actually has.
func (h *Host) formRefForPayload(ref string) string {
	if _, ok := h.formByRef(ref); ok {
		return strings.TrimSpace(ref)
	}
	return contactFlowKey
}
