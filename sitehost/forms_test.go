package sitehost

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func createForm(t *testing.T, handler http.Handler, name, template string) string {
	t.Helper()
	rec := post(t, handler, "/admin/forms", url.Values{"name": {name}, "template": {template}})
	location := rec.Header().Get("Location")
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(location, "/admin/forms/") {
		t.Fatalf("create form = %d %q", rec.Code, location)
	}
	return strings.TrimPrefix(strings.Split(location, "?")[0], "/admin/forms/")
}

func TestFormBuilderFromTemplateToSubmission(t *testing.T) {
	host, handler := newTestHost(t)
	mustContain(t, get(t, handler, "/admin/forms").Body.String(), "No forms of your own yet", "the list starts empty")

	id := createForm(t, handler, "Newsletter", "newsletter")
	builder := get(t, handler, "/admin/forms/"+id).Body.String()
	mustContain(t, builder, `value="Your email"`, "the template's questions are there")
	mustContain(t, builder, `value="Send me occasional news and offers"`, "including the tick box")
	mustContain(t, builder, `name="count" value="2"`, "the builder knows how many rows it has")

	// Add a question, then label it.
	post(t, handler, "/admin/forms/"+id, url.Values{
		"count": {"2"}, "label_0": {"Your email"}, "kind_0": {"email"}, "required_0": {"1"},
		"label_1": {"Send me occasional news and offers"}, "kind_1": {"checkbox"}, "required_1": {"1"},
		"name": {"Newsletter"}, "button": {"Sign me up"}, "thanks": {"You're on the list."}, "add": {"1"},
	})
	post(t, handler, "/admin/forms/"+id, url.Values{
		"count": {"3"}, "label_0": {"Your email"}, "kind_0": {"email"}, "required_0": {"1"},
		"label_1": {"Send me occasional news and offers"}, "kind_1": {"checkbox"}, "required_1": {"1"},
		"label_2": {"Your name"}, "kind_2": {"text"},
		"name": {"Newsletter"}, "button": {"Sign me up"}, "thanks": {"You're on the list."}, "save": {"1"},
	})
	form, _ := host.forms.get(id)
	if len(form.Fields) != 3 || form.Fields[2].Key != "your-name" || form.Fields[2].Required {
		t.Fatalf("fields after edit: %+v", form.Fields)
	}

	// Put it on a page.
	pageID := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+pageID, `{"title":"Menu","slug":"menu","blocks":[{"kind":"paragraph","text":"Join us"},{"kind":"form","form":"form:`+id+`"}]}`)
	editor := get(t, handler, "/admin/edit/"+pageID).Body.String()
	mustContain(t, editor, `data-form-select="true"`, "the canvas offers the form picker")
	mustContain(t, editor, `<option value="form:`+id+`" selected="selected">Newsletter</option>`, "with this form chosen")
	mustContain(t, editor, `data-forms-presets="true"`, "and the presets for new blocks")
	post(t, handler, "/admin/api/pages/"+pageID+"/publish", url.Values{})

	public := get(t, handler, "/menu").Body.String()
	mustContain(t, public, `action="/forms/`+id+`/send"`, "the public form posts to its own address")
	mustContain(t, public, `name="f_your-email" required="required"`, "required email field")
	mustContain(t, public, `<input name="f_send-me-occasional-news-and-offers" required="required" type="checkbox" value="yes"`, "the tick box")
	mustContain(t, public, `>Sign me up</button>`, "the owner's button text")

	// A missing tick is explained; a good submission lands in Messages.
	rec := post(t, handler, "/forms/"+id+"/send", url.Values{"page": {"/menu"}, "f_your-email": {"ana@example.com"}})
	mustContain(t, rec.Header().Get("Location"), "why=tick", "an unticked required box is refused")
	mustContain(t, get(t, handler, rec.Header().Get("Location")).Body.String(), "Please tick", "and the page says so")

	rec = post(t, handler, "/forms/"+id+"/send", url.Values{"page": {"/menu"}, "f_your-email": {"ana@example.com"}, "f_send-me-occasional-news-and-offers": {"yes"}, "f_your-name": {"Ana"}})
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "sent=1&form="+id) {
		t.Fatalf("submit = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	mustContain(t, get(t, handler, rec.Header().Get("Location")).Body.String(), "re on the list.", "the thank-you shows")

	inbox := get(t, handler, "/admin/messages").Body.String()
	mustContain(t, inbox, `class="admin-badge">Newsletter</span>`, "the inbox says which form")
	mustContain(t, inbox, "<dt>Your email</dt><dd>ana@example.com</dd>", "and shows the answers")
	mustContain(t, inbox, "<dt>Your name</dt><dd>Ana</dd>", "all of them")

	export := get(t, handler, "/admin/forms/"+id+"/export.csv")
	if !strings.HasPrefix(export.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("export content type = %q", export.Header().Get("Content-Type"))
	}
	lines := strings.Split(strings.TrimSpace(export.Body.String()), "\n")
	if len(lines) != 2 || lines[0] != "Received,Your email,Send me occasional news and offers,Your name" || !strings.HasSuffix(lines[1], ",ana@example.com,yes,Ana") {
		t.Fatalf("csv = %q", export.Body.String())
	}
	mustContain(t, get(t, handler, "/admin/forms").Body.String(), "1 submission", "the list counts submissions")

	if code := post(t, handler, "/forms/nope/send", url.Values{}).Code; code != http.StatusNotFound {
		t.Fatalf("unknown form = %d", code)
	}
	// Deleting keeps the submissions.
	post(t, handler, "/admin/forms/"+id+"/delete", url.Values{})
	if _, ok := host.forms.get(id); ok {
		t.Fatal("form still there after delete")
	}
	mustContain(t, get(t, handler, "/admin/messages").Body.String(), "ana@example.com", "submissions survive the form")
}

func TestFormWebhookDeliversJSON(t *testing.T) {
	allowInsecureWebhooks, webhookWait = true, true
	t.Cleanup(func() { allowInsecureWebhooks, webhookWait = false, false })
	var got map[string]any
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	host, handler := newTestHost(t)
	id := createForm(t, handler, "Bookings", "booking")
	form, _ := host.forms.get(id)
	form.WebhookURL = target.URL + "/hook"
	if _, err := host.forms.put(form); err != nil {
		t.Fatal(err)
	}
	post(t, handler, "/forms/"+id+"/send", url.Values{"page": {"/"}, "f_your-name": {"Ana"}, "f_your-email": {"ana@example.com"}, "f_date": {"2026-10-01"}, "f_how-many-people": {"2"}})
	if got == nil || got["name"] != "Ana" {
		t.Fatalf("webhook payload = %v", got)
	}
	fields, _ := got["fields"].([]any)
	if len(fields) != 6 {
		t.Fatalf("webhook fields = %v", got["fields"])
	}
	mustContain(t, get(t, handler, "/admin/forms/"+id).Body.String(), "Last delivery worked", "the builder reports the delivery")

	// A choice outside the list is refused.
	rec := post(t, handler, "/forms/"+id+"/send", url.Values{"page": {"/"}, "f_your-name": {"Ana"}, "f_your-email": {"ana@example.com"}, "f_date": {"2026-10-01"}, "f_how-many-people": {"99"}})
	mustContain(t, rec.Header().Get("Location"), "why=choice", "an invented choice is refused")
}

func TestContactFormStillWorksBesideCustomForms(t *testing.T) {
	_, handler := newTestHost(t)
	createForm(t, handler, "Quotes", "blank")
	rec := post(t, handler, "/contact/send", url.Values{"page": {"/contact"}, "name": {"Sam"}, "email": {"sam@example.com"}, "message": {"Hello there"}})
	if !strings.Contains(rec.Header().Get("Location"), "sent=1") {
		t.Fatalf("contact send = %q", rec.Header().Get("Location"))
	}
	inbox := get(t, handler, "/admin/messages").Body.String()
	mustContain(t, inbox, "Hello there", "contact messages still arrive")
	if strings.Contains(inbox, `class="admin-badge">Quotes`) {
		t.Fatal("a contact message must not be labelled with a custom form")
	}
	// A page's contact form ignores another form's state.
	public := get(t, handler, "/contact?sent=1&form=other").Body.String()
	if strings.Contains(public, "your message is on its way") {
		t.Fatal("the contact form must not show another form's thank-you")
	}
}
