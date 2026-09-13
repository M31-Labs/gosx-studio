package sitehost

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// onHost sends a GET the way a browser on another address would.
func onHost(t *testing.T, handler http.Handler, rawURL string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, rawURL, nil)
	for _, cookie := range cookies {
		if cookie != nil {
			req.AddCookie(cookie)
		}
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestStagingAddressShowsDraftsToKeyHolders(t *testing.T) {
	host, handler := newGuardedHost(t)
	rec := post(t, handler, loginPath, url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "password": {"correct horse battery"}, "serverPassword": {"server-secret"}})
	session := cookieNamed(rec, sessionCookie)

	// Turning it on.
	page := getWithCookie(t, handler, stagingAdminPath, session).Body.String()
	mustContain(t, page, "Turn on staging", "the page offers to set up a staging address")
	rec = postAs(t, handler, session, stagingAdminPath, url.Values{"stagingHost": {"wildflower.example"}})
	mustContain(t, rec.Header().Get("Location"), "different+from+your+main", "the main address cannot be the staging address")
	rec = postAs(t, handler, session, stagingAdminPath, url.Values{"stagingHost": {"Staging.Wildflower.example"}})
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "Staging+is+on") {
		t.Fatalf("enable = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	key := host.stagingKey()
	if host.stagingHost() != "staging.wildflower.example" || len(key) < 20 {
		t.Fatalf("staging settings: host=%q key=%q", host.stagingHost(), key)
	}
	page = getWithCookie(t, handler, stagingAdminPath, session).Body.String()
	mustContain(t, page, "https://staging.wildflower.example/?staging_key="+key, "the page shows the link to share")
	if err := host.HostPolicy(context.Background(), "staging.wildflower.example"); err != nil {
		t.Fatalf("staging host must get a certificate: %v", err)
	}

	// A change saved but not published, and a page never published.
	id := firstPageID(t, host, "menu")
	postJSONAs(t, handler, session, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Autumn tasting menu"}]}`)
	rec = postAs(t, handler, session, "/admin/pages", url.Values{"title": {"Workshops"}, "slug": {"workshops"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create page = %d %s", rec.Code, rec.Body.String())
	}
	live := get(t, handler, "/menu").Body.String()
	if strings.Contains(live, "Autumn tasting menu") || strings.Contains(live, "Workshops") {
		t.Fatal("visitors must not see drafts")
	}

	// The staging address: a gate without the key, drafts with it.
	rec = onHost(t, handler, "https://staging.wildflower.example/menu")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("staging without key = %d", rec.Code)
	}
	mustContain(t, rec.Body.String(), "staging copy of Wildflower", "the gate says what this is")
	if onHost(t, handler, "https://staging.wildflower.example/menu?staging_key=wrong").Code != http.StatusForbidden {
		t.Fatal("a wrong key must not open staging")
	}
	rec = onHost(t, handler, "https://staging.wildflower.example/menu?staging_key="+key)
	cookie := cookieNamed(rec, stagingCookie)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/menu" || cookie == nil || !cookie.HttpOnly || !cookie.Secure {
		t.Fatalf("staging with key = %d %q cookie=%v", rec.Code, rec.Header().Get("Location"), cookie)
	}
	rec = onHost(t, handler, "https://staging.wildflower.example/menu", cookie)
	staged := rec.Body.String()
	if rec.Code != http.StatusOK {
		t.Fatalf("staging with cookie = %d", rec.Code)
	}
	mustContain(t, staged, "Autumn tasting menu", "staging shows the unpublished change")
	mustContain(t, staged, "Staging — showing unpublished changes", "under a banner")
	mustContain(t, staged, `href="https://wildflower.example"`, "that links to the live site")
	mustContain(t, staged, `name="robots" content="noindex`, "and asks search engines to stay away")
	mustContain(t, staged, `href="/workshops"`, "the menu includes the never-published page")
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("staging pages must not be cached")
	}
	mustContain(t, onHost(t, handler, "https://staging.wildflower.example/workshops", cookie).Body.String(), "Workshops", "and the page itself opens")
	mustContain(t, onHost(t, handler, "https://staging.wildflower.example/robots.txt", cookie).Body.String(), "Disallow: /\n", "robots.txt closes staging")
	if strings.Contains(staged, "stats.js") {
		t.Fatal("no visitor beacon on staging")
	}

	// Anything that is not reading content goes to the real site.
	rec = onHost(t, handler, "https://staging.wildflower.example/admin/pages", cookie)
	if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "https://wildflower.example/admin/pages" {
		t.Fatalf("staging admin = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	req := httptest.NewRequest(http.MethodPost, "https://staging.wildflower.example/contact/send", strings.NewReader("x=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("staging POST = %d", rec.Code)
	}

	// A new key shuts old links and cookies out; turning staging off ends it.
	postAs(t, handler, session, stagingAdminPath+"/key", url.Values{})
	if host.stagingKey() == key {
		t.Fatal("the key must change")
	}
	if onHost(t, handler, "https://staging.wildflower.example/menu", cookie).Code != http.StatusForbidden {
		t.Fatal("an old cookie must stop working")
	}
	postAs(t, handler, session, stagingAdminPath+"/remove", url.Values{})
	rec = onHost(t, handler, "https://staging.wildflower.example/menu", cookie)
	if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "Autumn tasting menu") {
		t.Fatalf("with staging off the address is just the site: %d", rec.Code)
	}
	if err := host.HostPolicy(context.Background(), "staging.wildflower.example"); err == nil {
		t.Fatal("no certificate once staging is off")
	}
	mustContain(t, getWithCookie(t, handler, "/admin/activity", session).Body.String(), "Turned on staging at staging.wildflower.example", "the activity log records it")
}

func TestPublishEverythingWaiting(t *testing.T) {
	host, handler := newTestHost(t)
	mustContain(t, get(t, handler, "/admin").Body.String(), `href="/admin/staging">Staging</a>`, "Staging is in the admin menu")
	mustContain(t, get(t, handler, stagingAdminPath).Body.String(), "Nothing waiting", "a fresh site has nothing waiting")

	menu := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+menu, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Autumn tasting menu"}]}`)
	postID := createPost(t, handler, "Harvest supper")
	savePost(t, handler, postID, `{"title":"Harvest supper","slug":"harvest-supper","blocks":[{"kind":"paragraph","text":"Book a seat."}]}`)
	// One deliberately offline page must not be swept up.
	visit := firstPageID(t, host, "visit")
	postJSON(t, handler, "/admin/api/pages/"+visit, `{"title":"Visit","slug":"visit","description":"x","blocks":[{"kind":"paragraph","text":"New hours"}]}`)
	post(t, handler, "/admin/pages/"+visit+"/action", url.Values{"action": {"offline"}})

	page := get(t, handler, stagingAdminPath).Body.String()
	mustContain(t, page, "Waiting to go live (2)", "two changes wait")
	mustContain(t, page, ">Menu</a>", "the changed page")
	mustContain(t, page, ">Harvest supper</a>", "and the new post")
	mustContain(t, page, "Never published", "with its state")
	if strings.Contains(page, ">Visit</a>") {
		t.Fatal("an offline page is not waiting")
	}
	mustContain(t, get(t, handler, "/admin").Body.String(), "Staging (2)</a>", "the menu counts them")

	// One at a time, or all at once.
	rec := post(t, handler, stagingAdminPath+"/publish/page/"+menu, url.Values{})
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "Published") {
		t.Fatalf("publish one = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	mustContain(t, get(t, handler, "/menu").Body.String(), "Autumn tasting menu", "the page is live")
	rec = post(t, handler, stagingAdminPath+"/publish", url.Values{})
	if !strings.Contains(rec.Header().Get("Location"), "Published+1+change") {
		t.Fatalf("publish all = %q", rec.Header().Get("Location"))
	}
	mustContain(t, get(t, handler, "/blog/harvest-supper").Body.String(), "Book a seat.", "the post is live")
	mustContain(t, get(t, handler, stagingAdminPath).Body.String(), "Nothing waiting", "and nothing waits")
	if strings.Contains(get(t, handler, "/visit").Body.String(), "New hours") {
		t.Fatal("the offline page stays offline")
	}
	mustContain(t, get(t, handler, "/admin/activity").Body.String(), "from staging", "the activity log says how it was published")
}
