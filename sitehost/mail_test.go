package sitehost

import (
	"context"

	"encoding/json"
	"io"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeMailer struct {
	mu   sync.Mutex
	sent []Mail
	fail error
}

func (f *fakeMailer) Send(_ context.Context, mail Mail) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail != nil {
		return f.fail
	}
	f.sent = append(f.sent, mail)
	return nil
}
func (f *fakeMailer) Describe() string { return "a fake transport" }
func (f *fakeMailer) count() int       { f.mu.Lock(); defer f.mu.Unlock(); return len(f.sent) }

func newMailHost(t *testing.T) (*Host, http.Handler, *fakeMailer) {
	t.Helper()
	mailer := &fakeMailer{}
	host, err := Open(Options{
		DataPath: filepath.Join(t.TempDir(), "s.json"), SiteTitle: "Wildflower Bakery", SiteKind: "food",
		Seed: true, BaseURL: "https://wildflower.example", Mailer: mailer,
	})
	if err != nil {
		t.Fatal(err)
	}
	// The wizard would have set this; Seed does not ask.
	if err := host.updateSettingsMetadata(func(m cmsstore.Metadata) { m["contactEmail"] = "owner@wildflower.example" }); err != nil {
		t.Fatal(err)
	}
	return host, host.Handler(), mailer
}

func TestNewMessageEmailsTheOwner(t *testing.T) {
	_, handler, mailer := newMailHost(t)
	sendMessage(t, handler, url.Values{"name": {"Priya"}, "email": {"priya@example.com"}, "message": {"Wedding cake?"}})

	deadline := time.Now().Add(3 * time.Second)
	for mailer.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if mailer.count() != 1 {
		t.Fatalf("expected one notification, got %d", mailer.count())
	}
	mail := mailer.sent[0]
	if mail.To != "owner@wildflower.example" {
		t.Fatalf("notification went to %q", mail.To)
	}
	if mail.ReplyTo != "priya@example.com" {
		t.Fatalf("reply-to = %q, want the sender so the owner can just hit reply", mail.ReplyTo)
	}
	mustContain(t, mail.Subject, "Priya", "subject names the sender")
	mustContain(t, mail.Subject, "Wildflower Bakery", "subject names the site")
	mustContain(t, mail.Text, "Wedding cake?", "body carries the message")
	mustContain(t, mail.Text, "https://wildflower.example/admin/messages", "body links to the inbox")
}

func TestNotifyAddressOverrideFromSettings(t *testing.T) {
	host, handler, mailer := newMailHost(t)
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "notifyEmail": "team@wildflower.example"}, nil)
	if host.notifyAddress() != "team@wildflower.example" {
		t.Fatalf("notify address = %q", host.notifyAddress())
	}
	sendMessage(t, handler, url.Values{"name": {"A"}, "email": {"a@example.com"}, "message": {"hi"}})
	deadline := time.Now().Add(3 * time.Second)
	for mailer.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if mailer.count() != 1 || mailer.sent[0].To != "team@wildflower.example" {
		t.Fatalf("override not used: %+v", mailer.sent)
	}
}

func TestTestEmailButtonReportsOutcome(t *testing.T) {
	_, handler, mailer := newMailHost(t)
	rec := post(t, handler, "/admin/settings/test-mail", url.Values{})
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "Test+email+sent+to+owner%40wildflower.example") {
		t.Fatalf("test mail redirect = %q", loc)
	}
	if mailer.count() != 1 {
		t.Fatal("test email was not sent")
	}
	settings := get(t, handler, "/admin/settings").Body.String()
	mustContain(t, settings, "Working. Last email sent", "settings shows delivery is working")

	mailer.fail = io.ErrUnexpectedEOF
	rec = post(t, handler, "/admin/settings/test-mail", url.Values{})
	if loc := rec.Header().Get("Location"); !strings.Contains(loc, "didn%27t+send") {
		t.Fatalf("failure not reported: %q", loc)
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "Last attempt failed", "settings shows the last failure")
}

func TestNoTransportMeansNoEmailAndAnHonestSettingsPanel(t *testing.T) {
	_, handler := newTestHost(t)
	body := get(t, handler, "/admin/settings").Body.String()
	mustContain(t, body, "No email service is set up", "settings says email is not configured")
	mustContain(t, body, "GOSX_SITE_MAIL", "settings says how to configure it")
	// A message still lands in the inbox.
	sendMessage(t, handler, url.Values{"name": {"A"}, "email": {"a@example.com"}, "message": {"hi"}})
	mustContain(t, get(t, handler, "/admin/messages").Body.String(), "a@example.com", "inbox still works without email")
}

func TestParseMailURL(t *testing.T) {
	cases := map[string]string{
		"smtp://u:p@smtp.example.com:587?from=hi@example.com": "SMTP via smtp.example.com",
		"smtps://u:p@smtp.example.com?from=hi@example.com":    "SMTP via smtp.example.com",
		"resend://re_123?from=hi@example.com":                 "Resend",
		"postmark://server-token?from=hi@example.com":         "Postmark",
	}
	for raw, want := range cases {
		mailer, err := ParseMailURL(raw)
		if err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		mustContain(t, mailer.Describe(), want, raw)
	}
	for _, bad := range []string{"smtp://smtp.example.com", "ftp://x?from=a@b", "resend://?from=a@b"} {
		if _, err := ParseMailURL(bad); err == nil {
			t.Fatalf("%q should be rejected", bad)
		}
	}
	if mailer, err := ParseMailURL(""); mailer != nil || err != nil {
		t.Fatal("empty means no email, not an error")
	}
}

func TestHTTPMailersSpeakTheirAPIs(t *testing.T) {
	var got map[string]any
	var auth, token string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		token = r.Header.Get("X-Postmark-Server-Token")
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer server.Close()

	resend := &httpMailer{name: "Resend", endpoint: server.URL, key: "re_k", from: "hi@example.com", shape: "resend", client: server.Client()}
	if err := resend.Send(context.Background(), Mail{To: "o@example.com", Subject: "S", Text: "T", ReplyTo: "r@example.com"}); err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer re_k" || got["reply_to"] != "r@example.com" || got["from"] != "hi@example.com" {
		t.Fatalf("resend payload wrong: auth=%q body=%v", auth, got)
	}

	postmark := &httpMailer{name: "Postmark", endpoint: server.URL, key: "pm_t", from: "hi@example.com", shape: "postmark", client: server.Client()}
	if err := postmark.Send(context.Background(), Mail{To: "o@example.com", Subject: "S", Text: "T"}); err != nil {
		t.Fatal(err)
	}
	if token != "pm_t" || got["TextBody"] != "T" || got["MessageStream"] != "outbound" {
		t.Fatalf("postmark payload wrong: token=%q body=%v", token, got)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"bad from"}`))
	}))
	defer failing.Close()
	bad := &httpMailer{name: "Resend", endpoint: failing.URL, key: "k", from: "x", shape: "resend", client: failing.Client()}
	if err := bad.Send(context.Background(), Mail{To: "o@example.com"}); err == nil || !strings.Contains(err.Error(), "422") {
		t.Fatalf("API errors must surface with the status: %v", err)
	}
}

func TestSMTPMessageFormatting(t *testing.T) {
	text := formatMessage("hi@example.com", Mail{To: "o@example.com", Subject: "Line\r\nInjected: x", Text: "a\nb", ReplyTo: "r@example.com"})
	mustContain(t, text, "Subject: Line  Injected: x\r\n", "header newlines are stripped so nothing can be injected")
	mustContain(t, text, "Reply-To: r@example.com", "reply-to is set")
	mustContain(t, text, "\r\n\r\na\r\nb", "body follows the blank line with CRLF endings")
}
