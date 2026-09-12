package sitehost

import (
	"bytes"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

func TestLockedBlocksStayPutForEditors(t *testing.T) {
	host, handler := newGuardedHost(t)
	rec := post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}, "serverPassword": {"server-secret"}})
	owner := cookieNamed(rec, sessionCookie)
	body := postAs(t, handler, owner, peoplePath, url.Values{"action": {"invite"}, "email": {"sam@example.com"}, "role": {"editor"}}).Body.String()
	link := body[strings.Index(body, "https://wildflower.example/admin/join/"):]
	token := strings.TrimPrefix(link[:strings.Index(link, "<")], "https://wildflower.example/admin/join/")
	editor := cookieNamed(post(t, handler, joinPrefix+token, url.Values{"name": {"Sam"}, "password": {"another long password"}}), sessionCookie)

	id := firstPageID(t, host, "menu")
	// The owner locks the opening line.
	postJSONAs(t, handler, owner, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"heading","text":"Our brand promise","level":"2","locked":"true"},
		{"kind":"paragraph","text":"Editors may change this."}
	]}`)
	page, _, _ := host.Store().PageByID(id)
	if page.Body.Blocks[0].Values[lockedKey].String != "true" || page.Body.Blocks[1].Values[lockedKey].String != "" {
		t.Fatalf("lock stored wrong: %+v", page.Body.Blocks)
	}
	ownerView := getWithCookie(t, handler, "/admin/edit/"+id, owner).Body.String()
	mustContain(t, ownerView, `data-can-lock="true"`, "owners can lock")
	mustContain(t, ownerView, `data-locked="true"`, "the block is marked")
	editorView := getWithCookie(t, handler, "/admin/edit/"+id, editor).Body.String()
	mustContain(t, editorView, `data-can-lock="false"`, "editors cannot lock")
	mustContain(t, editorView, `data-locked="true"`, "but see the lock")

	// The editor changes the free paragraph: fine.
	rec = postJSONAs(t, handler, editor, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"heading","text":"Our brand promise","level":"2","locked":"true"},
		{"kind":"paragraph","text":"Changed by Sam."}
	]}`)
	mustContain(t, rec.Body.String(), `"ok":true`, "editing around a lock is fine")
	// The editor changes the locked heading: refused, nothing saved.
	rec = postJSONAs(t, handler, editor, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"heading","text":"Our NEW promise","level":"2","locked":"true"},
		{"kind":"paragraph","text":"Changed by Sam."}
	]}`)
	mustContain(t, rec.Body.String(), "locked sections that only an admin can change", "changing a locked block is refused")
	// Dropping it is refused too.
	rec = postJSONAs(t, handler, editor, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Alone"}]}`)
	mustContain(t, rec.Body.String(), "locked sections", "removing a locked block is refused")
	page, _, _ = host.Store().PageByID(id)
	if page.Body.Blocks[0].Values["text"].String != "Our brand promise" || page.Body.Blocks[1].Values["text"].String != "Changed by Sam." {
		t.Fatalf("page after refused saves: %+v", page.Body.Blocks)
	}
	// The owner may change or unlock it.
	rec = postJSONAs(t, handler, owner, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"heading","text":"Our NEW promise","level":"2"},{"kind":"paragraph","text":"Changed by Sam."}]}`)
	mustContain(t, rec.Body.String(), `"ok":true`, "owners change locked blocks")
}

func TestRequestsAreLoggedWithIdsAndCounted(t *testing.T) {
	var log bytes.Buffer
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "W", SiteKind: "food", Seed: true, NoBackups: true, LogRequests: true, LogWriter: &log})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	rec := get(t, handler, "/menu")
	if id := rec.Header().Get(requestIDHeader); len(id) != 16 {
		t.Fatalf("request id = %q", id)
	}
	req, _ := http.NewRequest(http.MethodGet, "/nope", nil)
	req.Header.Set(requestIDHeader, "trace-abc-123")
	rec2 := getWithCookie(t, handler, "/nope", &http.Cookie{Name: "x", Value: "y"})
	_ = rec2
	rec3 := get(t, handler, "/admin/metrics")
	mustContain(t, rec3.Body.String(), `gosx_site_requests_total{status="2xx",area="public"} 1`, "public hits are counted")
	mustContain(t, rec3.Body.String(), `gosx_site_requests_total{status="4xx",area="public"} 1`, "and misses")
	mustContain(t, rec3.Body.String(), `gosx_site_content{kind="pages"}`, "content gauges are exported")
	lines := strings.Split(strings.TrimSpace(log.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("log lines = %d: %s", len(lines), log.String())
	}
	mustContain(t, lines[0], `"path":"/menu"`, "the log names the path")
	mustContain(t, lines[0], `"status":200`, "and the status")
	mustContain(t, lines[0], `"area":"public"`, "and the area")
	mustContain(t, lines[0], `"id":"`, "and the request id")
	if strings.Contains(log.String(), "/_gosx/") {
		t.Fatal("assets are not logged")
	}
}

func TestMessagesCanBeExportedDeletedAndExpired(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	host, handler := newTestHost(t)
	post(t, handler, "/contact/send", url.Values{"page": {"/contact"}, "name": {"Old Sam"}, "email": {"old@example.com"}, "message": {"From long ago"}})
	messages, _ := host.messages.list()
	old := messages[0]
	old.Received = base.Add(-400 * 24 * time.Hour)
	host.messages.remove(old.ID)
	host.messages.add(old)
	post(t, handler, "/contact/send", url.Values{"page": {"/contact"}, "name": {"New Ana"}, "email": {"ana@example.com"}, "message": {"=SUM(1) fresh"}})

	export := get(t, handler, "/admin/messages/export.csv")
	mustContain(t, export.Header().Get("Content-Type"), "text/csv", "messages export as CSV")
	mustContain(t, export.Body.String(), "Received,Form,Name,Email,Message,Page", "with a header")
	mustContain(t, export.Body.String(), "'=SUM(1) fresh", "and formula-safe cells")
	mustContain(t, export.Body.String(), "From long ago", "and every message")

	// Prune by age from the inbox.
	rec := post(t, handler, "/admin/messages/prune", url.Values{"days": {"365"}})
	mustContain(t, rec.Header().Get("Location"), "Deleted+1+message", "the old one goes")
	inbox := get(t, handler, "/admin/messages").Body.String()
	if strings.Contains(inbox, "From long ago") {
		t.Fatal("pruned message still shown")
	}
	mustContain(t, inbox, "fresh", "the fresh one stays")
	// Delete one.
	messages, _ = host.messages.list()
	post(t, handler, "/admin/messages/"+messages[0].ID+"/delete", url.Values{})
	mustContain(t, get(t, handler, "/admin/messages").Body.String(), "No messages yet", "deleted for good")

	// Retention applies on its own once set.
	post(t, handler, "/contact/send", url.Values{"page": {"/contact"}, "name": {"Sam"}, "email": {"sam@example.com"}, "message": {"Keep me 10 days"}})
	if err := host.updateSettingsMetadata(func(m cmsstore.Metadata) { m[messageRetentionKey] = "10" }); err != nil {
		t.Fatal(err)
	}
	timeNow = func() time.Time { return base.Add(11 * 24 * time.Hour) }
	get(t, handler, "/")
	if strings.Contains(get(t, handler, "/admin/messages").Body.String(), "Keep me 10 days") {
		t.Fatal("retention should have deleted the message")
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), `name="messageRetentionDays" value="10"`, "the setting shows")

	// A privacy page in plain words.
	rec = post(t, handler, "/admin/privacy-page", url.Values{})
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/admin/edit/") {
		t.Fatalf("privacy page = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	page, ok, _ := host.Store().PageBySlug(privacySlug)
	if !ok || !PageNavHidden(page) || len(page.Body.Blocks) < 4 {
		t.Fatalf("privacy page: %+v %v", page, ok)
	}
	mustContain(t, page.Body.Blocks[0].Values["text"].String, "keeps as little about you as it can", "it is written for people")
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "Edit the privacy page", "Settings links to it once it exists")
	if rec := post(t, handler, "/admin/privacy-page", url.Values{}); !strings.Contains(rec.Header().Get("Location"), "/admin/edit/"+page.ID) {
		t.Fatal("a second click opens the existing page")
	}
}
