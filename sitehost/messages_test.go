package sitehost

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func sendMessage(t *testing.T, handler http.Handler, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	if form.Get("page") == "" {
		form.Set("page", "/contact")
	}
	return post(t, handler, contactSendPath, form)
}

func TestStarterContactPageHasAWorkingForm(t *testing.T) {
	_, handler := newTestHost(t)
	body := get(t, handler, "/contact").Body.String()
	mustContain(t, body, `<form class="site-form" id="contact-form" method="post" action="/contact/send"`, "the contact page carries the form")
	mustContain(t, body, `name="email"`, "the form asks for an email")
	mustContain(t, body, `name="message"`, "the form asks for a message")
	mustContain(t, body, `name="website"`, "the form has a honeypot")
	mustContain(t, body, `value="/contact"`, "the form knows which page to return to")
}

func TestVisitorMessageLandsInTheInbox(t *testing.T) {
	host, handler := newTestHost(t)

	rec := sendMessage(t, handler, url.Values{
		"name": {"Priya"}, "email": {"priya@example.com"}, "message": {"Do you do wedding cakes?"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("send = %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/contact?sent=1#contact-form" {
		t.Fatalf("send redirected to %q", loc)
	}
	thanks := get(t, handler, "/contact?sent=1").Body.String()
	mustContain(t, thanks, "your message is on its way", "the visitor sees a thank-you")

	// The owner sees it everywhere it matters.
	mustContain(t, get(t, handler, "/admin").Body.String(), "Messages (1)", "the admin nav shows the unread count")
	inbox := get(t, handler, "/admin/messages").Body.String()
	mustContain(t, inbox, "Priya", "the inbox shows the sender")
	mustContain(t, inbox, "Do you do wedding cakes?", "the inbox shows the message")
	mustContain(t, inbox, `href="mailto:priya@example.com`, "the owner can reply by email")
	mustContain(t, inbox, `data-state="unread"`, "a new message is marked unread")

	// It survives a restart.
	reopened, err := Open(host.Options())
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, get(t, reopened.Handler(), "/admin/messages").Body.String(), "wedding cakes", "messages persist on disk")

	// Marking it read clears the count.
	messages, _ := host.messages.list()
	post(t, handler, "/admin/messages/"+messages[0].ID+"/read", url.Values{"read": {"1"}})
	if strings.Contains(get(t, handler, "/admin").Body.String(), "Messages (1)") {
		t.Fatal("marking read did not clear the unread count")
	}
}

func TestContactFormRejectsBadInputInPlainWords(t *testing.T) {
	host, handler := newTestHost(t)

	rec := sendMessage(t, handler, url.Values{"name": {"Sam"}, "email": {"not-an-email"}, "message": {"Hi"}})
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "why=email") {
		t.Fatalf("bad email redirected to %q", loc)
	}
	mustContain(t, get(t, handler, "/contact?sent=error&why=email").Body.String(), "Check it and try again", "the visitor is told what to fix")

	rec = sendMessage(t, handler, url.Values{"name": {"Sam"}, "email": {"sam@example.com"}, "message": {"   "}})
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "why=message") {
		t.Fatalf("empty message redirected to %q", loc)
	}

	if messages, _ := host.messages.list(); len(messages) != 0 {
		t.Fatalf("rejected submissions were stored: %d", len(messages))
	}
}

func TestHoneypotSubmissionsAreSilentlyDropped(t *testing.T) {
	host, handler := newTestHost(t)
	rec := sendMessage(t, handler, url.Values{
		"name": {"Bot"}, "email": {"bot@example.com"}, "message": {"BUY NOW"}, "website": {"http://spam.example"},
	})
	if loc := rec.Header().Get("Location"); loc != "/contact?sent=1#contact-form" {
		t.Fatalf("a bot should be told it succeeded, got %q", loc)
	}
	if messages, _ := host.messages.list(); len(messages) != 0 {
		t.Fatal("a honeypot submission was stored")
	}
}

func TestContactFormRateLimits(t *testing.T) {
	host, handler := newTestHost(t)
	for i := 0; i < messagesRatePer; i++ {
		sendMessage(t, handler, url.Values{"name": {"A"}, "email": {"a@example.com"}, "message": {"m"}})
	}
	rec := sendMessage(t, handler, url.Values{"name": {"A"}, "email": {"a@example.com"}, "message": {"one too many"}})
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "why=rate") {
		t.Fatalf("sixth send in a row was not rate limited: %q", loc)
	}
	if messages, _ := host.messages.list(); len(messages) != messagesRatePer {
		t.Fatalf("stored %d messages, want %d", len(messages), messagesRatePer)
	}
}

func TestReturnPathStaysOnThisSite(t *testing.T) {
	_, handler := newTestHost(t)
	for _, evil := range []string{"https://evil.example/", "//evil.example", "/contact\r\nSet-Cookie: x=y"} {
		rec := sendMessage(t, handler, url.Values{"page": {evil}, "name": {"A"}, "email": {"a@example.com"}, "message": {"m"}})
		if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/?sent=") && !strings.HasPrefix(loc, "/contact?sent=") {
			t.Fatalf("return path %q escaped to %q", evil, loc)
		}
	}
}

func TestEditorOffersAndSavesTheContactForm(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	body := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, body, `data-add="form"`, "the sidebar offers a contact form")
	mustContain(t, body, `data-insert="form"`, "the insert menu offers a contact form")

	payload := `{"title":"Menu","slug":"menu","blocks":[{"kind":"paragraph","text":"Ask us anything."},{"kind":"form"}]}`
	postJSON(t, handler, "/admin/api/pages/"+id, payload)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})

	live := get(t, handler, "/menu").Body.String()
	mustContain(t, live, `action="/contact/send"`, "a form placed in the editor works for visitors")
	mustContain(t, live, `value="/menu"`, "the form returns to the page it lives on")
	canvas := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, canvas, `data-block="form"`, "the form shows on the canvas as a block")
	mustContain(t, canvas, `data-form-select="true"`, "the canvas preview lets the owner pick a form")
}
