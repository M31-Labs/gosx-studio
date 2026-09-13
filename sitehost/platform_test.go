package sitehost

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedSitesLeaveDomainsToThePlatform(t *testing.T) {
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Wildflower", SiteKind: "food", Seed: true, NoBackups: true,
		ManagedBy: "GoSX Platform", Domain: "Wildflower.Example", OperatorToken: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	if host.Domain() != "wildflower.example" || host.settings().BaseURL != "https://wildflower.example" {
		t.Fatalf("domain=%q base=%q", host.Domain(), host.settings().BaseURL)
	}
	page := get(t, handler, "/admin/domain").Body.String()
	mustContain(t, page, "Connected and kept secure by GoSX Platform", "the domain page defers to the platform")
	if strings.Contains(page, "A record") || strings.Contains(page, `name="domain"`) {
		t.Fatal("no DNS instructions or domain form on a managed site")
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "kept secure by GoSX Platform", "so does the settings card")
	rec := post(t, handler, "/admin/domain", url.Values{"domain": {"other.example"}})
	if rec.Code != http.StatusSeeOther || host.Domain() != "wildflower.example" {
		t.Fatal("the owner cannot change a platform-connected domain")
	}
	// The www twin redirects to the connected name, as it would on its own.
	req := httptest.NewRequest(http.MethodGet, "http://www.wildflower.example/menu", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "http://wildflower.example/menu" {
		t.Fatalf("www = %d %q", rec.Code, rec.Header().Get("Location"))
	}

	// The platform's door: closed without the token, open with it.
	for _, offered := range []string{"", "Bearer wrong", "Bearer 0123456789abcdef0123456789abcde"} {
		req := httptest.NewRequest(http.MethodGet, platformStatusPath, nil)
		if offered != "" {
			req.Header.Set("Authorization", offered)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status with %q = %d", offered, rec.Code)
		}
	}
	req = httptest.NewRequest(http.MethodGet, platformStatusPath, nil)
	req.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d %s", rec.Code, rec.Body.String())
	}
	mustContain(t, rec.Body.String(), `"title":"Wildflower"`, "status names the site")
	mustContain(t, rec.Body.String(), `"setupComplete":true`, "and says setup is done")
	mustContain(t, rec.Body.String(), `"domain":"wildflower.example"`, "and the domain")
	req = httptest.NewRequest(http.MethodGet, platformExportPath, nil)
	req.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" || rec.Body.Len() < 100 {
		t.Fatalf("export = %d %q %d bytes", rec.Code, rec.Header().Get("Content-Type"), rec.Body.Len())
	}

	// Before the owner has run the wizard, the platform can still ask.
	fresh, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "New", NoBackups: true, ManagedBy: "GoSX Platform", OperatorToken: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	freshHandler := fresh.Handler()
	req = httptest.NewRequest(http.MethodGet, platformStatusPath, nil)
	req.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef")
	rec = httptest.NewRecorder()
	freshHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"setupComplete":false`) {
		t.Fatalf("status before setup = %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, platformExportPath, nil)
	req.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef")
	rec = httptest.NewRecorder()
	freshHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("export before setup = %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	req = httptest.NewRequest(http.MethodGet, platformStatusPath, nil)
	rec = httptest.NewRecorder()
	freshHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status before setup without token = %d", rec.Code)
	}

	// A site on its own has no door at all.
	_, plain := newTestHost(t)
	req = httptest.NewRequest(http.MethodGet, platformStatusPath, nil)
	req.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef")
	rec = httptest.NewRecorder()
	plain.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unmanaged status = %d", rec.Code)
	}
	mustContain(t, get(t, plain, "/admin/domain").Body.String(), `name="domain"`, "and keeps its own domain form")
}
