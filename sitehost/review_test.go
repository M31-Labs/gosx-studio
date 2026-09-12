package sitehost

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

func TestEditorsRequestReviewAndAdminsDecide(t *testing.T) {
	host, handler := newGuardedHost(t)
	rec := post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}, "serverPassword": {"server-secret"}})
	owner := cookieNamed(rec, sessionCookie)
	// Invite an editor.
	body := postAs(t, handler, owner, peoplePath, url.Values{"action": {"invite"}, "email": {"sam@example.com"}, "role": {"editor"}}).Body.String()
	link := body[strings.Index(body, "https://wildflower.example/admin/join/"):]
	token := strings.TrimPrefix(link[:strings.Index(link, "<")], "https://wildflower.example/admin/join/")
	editor := cookieNamed(post(t, handler, joinPrefix+token, url.Values{"name": {"Sam"}, "password": {"another long password"}}), sessionCookie)

	id := firstPageID(t, host, "menu")
	// Approval off: the editor publishes directly.
	mustContain(t, getWithCookie(t, handler, "/admin/edit/"+id, editor).Body.String(), `data-must-request="false"`, "no approval needed by default")

	// Turn approval on (the Settings form's switch writes this key).
	if err := host.updateSettingsMetadata(func(m cmsstore.Metadata) { m[reviewRequiredKey] = "true" }); err != nil {
		t.Fatal(err)
	}
	mustContain(t, getWithCookie(t, handler, "/admin/settings", owner).Body.String(), `name="reviewRequired" value="true" checked="checked"`, "the switch shows on")

	// The editor now sees Request review, and publishing is refused.
	editorPage := getWithCookie(t, handler, "/admin/edit/"+id, editor).Body.String()
	mustContain(t, editorPage, `data-must-request="true"`, "the editor must request review")
	mustContain(t, editorPage, ">Request review<", "the button says so")
	mustContain(t, editorPage, "Make a preview link", "and can share a preview")
	rec = postWithCookie(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{"_csrf": {csrfWith(t, handler, editor)}}, editor)
	mustContain(t, rec.Body.String(), "needs an admin to approve", "publishing is refused for editors")
	mustContain(t, getWithCookie(t, handler, "/admin/edit/"+id, owner).Body.String(), `data-must-request="false"`, "the owner still publishes directly")

	// The editor saves a change and asks for a review.
	postJSONAs(t, handler, editor, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"New autumn menu"}]}`)
	rec = postJSONAs(t, handler, editor, "/admin/api/pages/"+id+"/review", `{"note":"Please check the prices"}`)
	mustContain(t, rec.Body.String(), "Sent for review", "the request is acknowledged")
	if strings.Contains(get(t, handler, "/menu").Body.String(), "New autumn menu") {
		t.Fatal("a request must not publish")
	}
	mustContain(t, getWithCookie(t, handler, "/admin/edit/"+id, editor).Body.String(), "Waiting for review", "the editor sees the state")
	mustContain(t, getWithCookie(t, handler, "/admin", owner).Body.String(), "Review (1)", "the nav counts the request")
	queue := getWithCookie(t, handler, reviewPath, owner).Body.String()
	mustContain(t, queue, "Menu", "the queue lists the page")
	mustContain(t, queue, "Sam", "and who asked")
	mustContain(t, queue, "Please check the prices", "and their note")
	look := getWithCookie(t, handler, reviewPath+"/page/"+id, owner).Body.String()
	mustContain(t, look, "New autumn menu", "the admin sees the draft")
	mustContain(t, look, "Sam asked for a review", "with the banner")

	// Send it back, then approve a second request.
	postAs(t, handler, owner, reviewPath+"/page/"+id, url.Values{"decision": {"sendback"}, "note": {"Prices look wrong, check the croissant"}})
	mustContain(t, getWithCookie(t, handler, "/admin/edit/"+id, editor).Body.String(), "Prices look wrong", "the editor sees the feedback")
	if len(host.reviewQueue()) != 0 {
		t.Fatal("sending back clears the queue")
	}
	postJSONAs(t, handler, editor, "/admin/api/pages/"+id+"/review", `{}`)
	rec = postAs(t, handler, owner, reviewPath+"/page/"+id, url.Values{"decision": {"approve"}})
	mustContain(t, rec.Header().Get("Location"), "Approved", "approval is confirmed")
	mustContain(t, get(t, handler, "/menu").Body.String(), "New autumn menu", "and the change is live")
	if len(host.reviewQueue()) != 0 {
		t.Fatal("approval clears the queue")
	}
	mustContain(t, getWithCookie(t, handler, "/admin/activity", owner).Body.String(), "Approved and published", "the decision is logged")
	if code := postWithCookie(t, handler, reviewPath+"/page/"+id, url.Values{"decision": {"approve"}, "_csrf": {csrfWith(t, handler, editor)}}, editor).Code; code != http.StatusForbidden {
		t.Fatalf("editors deciding = %d", code)
	}
}

func TestPreviewLinksShowTheDraftToAnyone(t *testing.T) {
	base := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Secret winter menu"}]}`)
	rec := post(t, handler, "/admin/api/pages/"+id+"/preview-link", url.Values{})
	body := rec.Body.String()
	mustContain(t, body, `"ok":true`, "a link is made")
	mustContain(t, body, "https://wildflower.example/preview/v1.", "in the library's token format")
	start := strings.Index(body, "/preview/")
	path := body[start:]
	path = path[:strings.Index(path, `"`)]

	// Anyone with the link sees the draft, which is not live.
	if strings.Contains(get(t, handler, "/menu").Body.String(), "Secret winter menu") {
		t.Fatal("the draft must not be live")
	}
	preview := get(t, handler, path)
	if preview.Code != http.StatusOK {
		t.Fatalf("preview = %d", preview.Code)
	}
	mustContain(t, preview.Body.String(), "Secret winter menu", "the preview shows the draft")
	mustContain(t, preview.Body.String(), "Preview of a draft", "with a banner")
	mustContain(t, preview.Body.String(), `content="noindex, nofollow"`, "and stays out of search")
	if strings.Contains(preview.Body.String(), statsScriptPath) {
		t.Fatal("a preview is not a visit")
	}
	// Tampered or expired links fail.
	if code := get(t, handler, path[:len(path)-2]+"xx").Code; code != http.StatusNotFound {
		t.Fatalf("tampered token = %d", code)
	}
	timeNow = func() time.Time { return base.Add(80 * time.Hour) }
	if code := get(t, handler, path).Code; code != http.StatusNotFound {
		t.Fatalf("expired token = %d", code)
	}
}

func postJSONAs(t *testing.T, handler http.Handler, session *http.Cookie, path, payload string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfWith(t, handler, session))
	req.AddCookie(session)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
