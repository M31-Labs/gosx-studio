package sitehost

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeDomain(t *testing.T) {
	cases := map[string]string{
		"example.com":                     "example.com",
		"  HTTPS://WWW.Example.com/menu ": "example.com",
		"www.shop.example.co.uk":          "shop.example.co.uk",
		"example.com:8080":                "example.com",
		"example.com.":                    "example.com",
		"localhost":                       "",
		"203.0.113.5":                     "",
		"not a domain":                    "",
		"-bad.com":                        "",
		"":                                "",
		"ex_ample.com":                    "",
	}
	for in, want := range cases {
		if got := normalizeDomain(in); got != want {
			t.Errorf("normalizeDomain(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConnectDomainGuidesTheOwner(t *testing.T) {
	previousLookup := lookupHost
	lookupHost = func(host string) ([]string, error) {
		switch host {
		case "bluedoor.example":
			return []string{"203.0.113.5"}, nil
		}
		return nil, errors.New("no such host")
	}
	t.Cleanup(func() { lookupHost = previousLookup })

	host, err := Open(Options{
		DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Blue Door", SiteKind: "food", Seed: true,
		BaseURL: "http://203.0.113.5:8080", PublicIP: "203.0.113.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()

	page := get(t, handler, "/admin/domain").Body.String()
	mustContain(t, page, "1. Your domain", "the screen starts with the name")
	if strings.Contains(page, "2. Point") {
		t.Fatal("no records before a domain is typed")
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "Connect a domain", "Settings links to it")
	mustContain(t, get(t, handler, "/admin").Body.String(), `href="/admin/domain"`, "so does the dashboard")

	rec := post(t, handler, "/admin/domain", url.Values{"domain": {"https://WWW.BlueDoor.example/"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("save = %d %s", rec.Code, rec.Body.String())
	}
	if host.Domain() != "bluedoor.example" || host.canonicalHost() != "bluedoor.example" {
		t.Fatalf("domain = %q canonical = %q", host.Domain(), host.canonicalHost())
	}
	if base := host.settings().BaseURL; base != "http://bluedoor.example" {
		t.Fatalf("base URL follows the domain: %q", base)
	}

	page = get(t, handler, "/admin/domain").Body.String()
	mustContain(t, page, "2. Point bluedoor.example at this server", "step two names the domain")
	mustContain(t, page, "<td>A</td><td>@</td><td><code>203.0.113.5</code></td>", "the A record shows the server IP")
	mustContain(t, page, "<td>CNAME</td><td>www</td><td><code>bluedoor.example</code></td>", "the www record points at the apex")
	mustContain(t, page, "HTTPS is off", "step four says HTTPS is off")
	mustContain(t, page, "gosx-site -https -data ", "and shows the exact command to turn it on")
	mustContain(t, get(t, handler, "/").Body.String(), `<link rel="canonical" href="http://bluedoor.example/"`, "share links use the new address")
	mustContain(t, get(t, handler, "/sitemap.xml").Body.String(), "<loc>http://bluedoor.example/</loc>", "and so does the sitemap")

	check := post(t, handler, "/admin/domain/check", url.Values{}).Body.String()
	mustContain(t, check, "Checked just now: bluedoor.example reaches this server.", "the check reports success")
	mustContain(t, check, "Points here ✓", "the apex points here")
	mustContain(t, check, "Not found yet", "www is not set up yet")

	// The twin address redirects to the canonical one.
	req := httptest.NewRequest(http.MethodGet, "http://www.bluedoor.example/menu?x=1", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusMovedPermanently || res.Header().Get("Location") != "http://bluedoor.example/menu?x=1" {
		t.Fatalf("www redirect = %d %q", res.Code, res.Header().Get("Location"))
	}
	req = httptest.NewRequest(http.MethodGet, "http://bluedoor.example/menu", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("canonical host = %d", res.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "http://203.0.113.5:8080/menu", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("the old address keeps working: %d", res.Code)
	}

	// Preferring www flips the redirect.
	post(t, handler, "/admin/domain", url.Values{"domain": {"bluedoor.example"}, "www": {"true"}})
	if host.canonicalHost() != "www.bluedoor.example" || host.settings().BaseURL != "http://www.bluedoor.example" {
		t.Fatalf("www preference: canonical=%q base=%q", host.canonicalHost(), host.settings().BaseURL)
	}
	req = httptest.NewRequest(http.MethodGet, "http://bluedoor.example/", nil)
	res = httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Header().Get("Location") != "http://www.bluedoor.example/" {
		t.Fatalf("apex → www redirect = %q", res.Header().Get("Location"))
	}

	// Certificates are only ever requested for the connected domain.
	if err := host.HostPolicy(context.Background(), "bluedoor.example"); err != nil {
		t.Fatal(err)
	}
	if err := host.HostPolicy(context.Background(), "WWW.bluedoor.example."); err != nil {
		t.Fatal(err)
	}
	if err := host.HostPolicy(context.Background(), "evil.example"); err == nil {
		t.Fatal("a stranger's host must be refused")
	}

	// Disconnecting goes back to the address the server started with.
	post(t, handler, "/admin/domain/remove", url.Values{})
	if host.Domain() != "" || host.settings().BaseURL != "http://203.0.113.5:8080" {
		t.Fatalf("after disconnect: domain=%q base=%q", host.Domain(), host.settings().BaseURL)
	}

	// Typing something that is not a domain is explained, not saved.
	mustContain(t, post(t, handler, "/admin/domain", url.Values{"domain": {"my shop"}}).Body.String(), "Type the domain name on its own", "bad input is explained")
	if host.Domain() != "" {
		t.Fatal("bad input must not be saved")
	}
}

func TestHTTPSModeSaysSoAndUsesHTTPSAddresses(t *testing.T) {
	dir := t.TempDir()
	host, err := Open(Options{DataPath: filepath.Join(dir, "site.json"), SiteTitle: "Blue Door", SiteKind: "food", Seed: true, TLS: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	post(t, handler, "/admin/domain", url.Values{"domain": {"bluedoor.example"}})
	if base := host.settings().BaseURL; base != "https://bluedoor.example" {
		t.Fatalf("base = %q", base)
	}
	page := get(t, handler, "/admin/domain").Body.String()
	mustContain(t, page, "HTTPS is on", "step four confirms HTTPS")
	mustContain(t, page, "https://bluedoor.example", "and the secure address")
	manager := host.TLSManager()
	if manager == nil || manager.Cache == nil {
		t.Fatal("HTTPS mode has a certificate manager with a cache")
	}
	if got := host.Options().certDir(); got != filepath.Join(dir, "certs") {
		t.Fatalf("cert dir = %q", got)
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "HTTPS on", "Settings shows the state")

	// Without HTTPS there is no manager at all.
	plain, _ := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "x", SiteKind: "food", Seed: true})
	if plain.TLSManager() != nil {
		t.Fatal("no manager without -https")
	}
}
