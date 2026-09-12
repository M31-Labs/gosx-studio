package sitehost

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// rawPost sends a form post with no token and no browser-supplied origin
// signal — what a request forged from another site looks like.
func rawPost(t *testing.T, handler http.Handler, path string, form url.Values, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestAdminPostsRequireTheToken(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	// No token: refused, and nothing changed.
	rec := rawPost(t, handler, "/admin/pages/"+id+"/action", url.Values{"action": {"archive"}}, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("forged admin post = %d, want 403", rec.Code)
	}
	if page, _, _ := host.Store().PageByID(id); PageArchived(page) {
		t.Fatal("a forged post archived the page")
	}
	mustContain(t, rec.Body.String(), "reload the page", "the refusal tells a real person what to do")

	// Wrong token: refused.
	rec = rawPost(t, handler, "/admin/pages/"+id+"/action", url.Values{"action": {"archive"}, "_csrf": {"nope"}}, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong token = %d, want 403", rec.Code)
	}

	// Right token: accepted.
	rec = rawPost(t, handler, "/admin/pages/"+id+"/action", url.Values{"action": {"archive"}, "_csrf": {csrfToken(t, handler)}}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("token post = %d, want 303", rec.Code)
	}
}

func TestCrossSiteOriginIsRefusedEvenWithAToken(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	rec := rawPost(t, handler, "/admin/pages/"+id+"/action",
		url.Values{"action": {"archive"}, "_csrf": {csrfToken(t, handler)}},
		map[string]string{"Sec-Fetch-Site": "cross-site"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site post = %d, want 403", rec.Code)
	}
}

func TestAPIPostsRequireTheHeaderToken(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+id+"/publish", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("API post without header = %d, want 403", rec.Code)
	}
	mustContain(t, rec.Body.String(), `"ok":false`, "the API refusal is JSON the editor can show")
	if rec := postJSON(t, handler, "/admin/api/pages/"+id+"/publish", ""); rec.Code != http.StatusOK {
		t.Fatalf("API post with header = %d, want 200", rec.Code)
	}
}

func TestPublicEndpointsNeedNoToken(t *testing.T) {
	_, handler := newTestHost(t)
	// The contact form is for the public; the wizard runs before there is an owner.
	rec := rawPost(t, handler, contactSendPath, url.Values{"page": {"/contact"}, "name": {"A"}, "email": {"a@example.com"}, "message": {"hi"}}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("contact form without token = %d, want 303", rec.Code)
	}
	unbuilt, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "n.json")})
	if err != nil {
		t.Fatal(err)
	}
	if rec := rawPost(t, unbuilt.Handler(), "/setup", url.Values{"step": {"1"}, "siteTitle": {"X"}}, nil); rec.Code != http.StatusOK {
		t.Fatalf("wizard without token = %d, want 200", rec.Code)
	}
}

func TestTokenIsOnAdminPagesAndNeverOnPublicOnes(t *testing.T) {
	_, handler := newTestHost(t)
	mustContain(t, get(t, handler, "/admin").Body.String(), `name="csrf-token"`, "admin pages expose the token to their own scripts")
	for _, path := range []string{"/", "/menu", "/contact", "/nope"} {
		if strings.Contains(get(t, handler, path).Body.String(), "csrf-token") {
			t.Fatalf("public page %s leaks the CSRF token", path)
		}
	}
}

func TestTokenIsStableAcrossRestartsWhenGuarded(t *testing.T) {
	opts := Options{DataPath: filepath.Join(t.TempDir(), "s.json"), AdminPassword: "pw", Seed: true}
	first, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	if first.csrfToken() != second.csrfToken() {
		t.Fatal("a deploy must not invalidate every open admin tab")
	}
	if first.csrfToken() == "" || len(first.csrfToken()) < 32 {
		t.Fatal("token too short")
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	_, handler := newTestHost(t)
	for _, path := range []string{"/", "/admin", "/menu", publicStylesheetPath} {
		rec := get(t, handler, path)
		csp := rec.Header().Get("Content-Security-Policy")
		if !strings.Contains(csp, "script-src 'self'") || !strings.Contains(csp, "frame-ancestors 'none'") {
			t.Fatalf("%s: weak or missing CSP: %q", path, csp)
		}
		if rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("X-Frame-Options") != "DENY" {
			t.Fatalf("%s: hardening headers missing", path)
		}
	}
}

func TestRepeatedFailedSignInsLockOut(t *testing.T) {
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "g.json"), AdminPassword: "right", Seed: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	attempt := func(password string) int {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.RemoteAddr = "203.0.113.9:4444"
		req.SetBasicAuth(AdminUser, password)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}
	for i := 0; i < authFailLimit; i++ {
		if code := attempt("wrong"); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401", i, code)
		}
	}
	if code := attempt("right"); code != http.StatusTooManyRequests {
		t.Fatalf("after %d failures the right password should still be locked out, got %d", authFailLimit, code)
	}
	// Another address is unaffected.
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.RemoteAddr = "198.51.100.7:1234"
	req.SetBasicAuth(AdminUser, "right")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("a different address was locked out too: %d", rec.Code)
	}
}
