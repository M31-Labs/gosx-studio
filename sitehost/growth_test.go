package sitehost

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func TestSitemapListsLivePagesOnly(t *testing.T) {
	host, handler := newTestHost(t)
	act(t, handler, firstPageID(t, host, "visit"), "offline")

	rec := get(t, handler, "/sitemap.xml")
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/xml") {
		t.Fatalf("sitemap = %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	mustContain(t, body, "<loc>https://wildflower.example/</loc>", "home is in the sitemap with the site's own address")
	mustContain(t, body, "<loc>https://wildflower.example/menu</loc>", "live pages are listed")
	mustContain(t, body, "<lastmod>", "entries carry a last-modified date")
	if strings.Contains(body, "/visit</loc>") {
		t.Fatal("an offline page is in the sitemap")
	}
}

func TestRobotsPointsAtTheSitemapAndKeepsAdminOut(t *testing.T) {
	_, handler := newTestHost(t)
	body := get(t, handler, "/robots.txt").Body.String()
	mustContain(t, body, "Disallow: /admin", "robots keeps crawlers out of the admin area")
	mustContain(t, body, "Sitemap: https://wildflower.example/sitemap.xml", "robots points at the sitemap")

	unbuilt, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "n.json")})
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, get(t, unbuilt.Handler(), "/robots.txt").Body.String(), "Disallow: /\n", "an unbuilt site asks not to be indexed")
}

func TestRenamingAPageKeepsTheOldAddressWorking(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	rec := postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"our-menu","blocks":[{"kind":"paragraph","text":"Bread."}]}`)
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("rename failed: %s", rec.Body.String())
	}
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})

	old := get(t, handler, "/menu")
	if old.Code != http.StatusMovedPermanently || old.Header().Get("Location") != "/our-menu" {
		t.Fatalf("old address = %d → %q, want 301 → /our-menu", old.Code, old.Header().Get("Location"))
	}
	if code := get(t, handler, "/our-menu").Code; code != http.StatusOK {
		t.Fatal("new address is not live")
	}

	// Rename again: the first address must jump straight to the newest one.
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"food","blocks":[{"kind":"paragraph","text":"Bread."}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	if loc := get(t, handler, "/menu").Header().Get("Location"); loc != "/food" {
		t.Fatalf("chain not collapsed: /menu → %q, want /food", loc)
	}
	if loc := get(t, handler, "/our-menu").Header().Get("Location"); loc != "/food" {
		t.Fatalf("/our-menu → %q, want /food", loc)
	}
}

func TestManualRedirectsFromSettings(t *testing.T) {
	_, handler := newTestHost(t)
	rec := postSettings(t, handler, map[string]string{
		"title": "Wildflower Bakery", "redirects": "/old-shop -> /menu\n/loop -> /loop\n# comment\nnot a path\n/promo /contact",
	}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("settings = %d", rec.Code)
	}
	if loc := get(t, handler, "/old-shop").Header().Get("Location"); loc != "/menu" {
		t.Fatalf("/old-shop → %q", loc)
	}
	if loc := get(t, handler, "/promo").Header().Get("Location"); loc != "/contact" {
		t.Fatalf("/promo → %q", loc)
	}
	if code := get(t, handler, "/loop").Code; code != http.StatusNotFound {
		t.Fatalf("a self-loop must be ignored, got %d", code)
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), "/old-shop -&gt; /menu", "the table shows in Settings")
}

func TestStructuredDataDescribesTheBusinessAndPage(t *testing.T) {
	host, handler := newTestHost(t)
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "socialInstagram": "https://instagram.com/wf"}, nil)
	_ = host
	body := get(t, handler, "/menu").Body.String()
	mustContain(t, body, `<script type="application/ld+json">`, "public pages carry structured data")
	mustContain(t, body, `"@type":"LocalBusiness"`, "a food business is a LocalBusiness")
	mustContain(t, body, `"name":"Wildflower Bakery"`, "the business name is there")
	mustContain(t, body, `"sameAs":["https://instagram.com/wf"]`, "social profiles are linked")
	mustContain(t, body, `"@type":"WebPage"`, "the page is described")
	mustContain(t, body, `"url":"https://wildflower.example/menu"`, "the page URL is absolute")
	if strings.Contains(get(t, handler, "/admin").Body.String(), "application/ld+json") {
		t.Fatal("admin pages must not carry structured data")
	}
}

func TestThirdPartyCodeIsGatedByConsentAndRelaxesOnlyThePublicPolicy(t *testing.T) {
	_, handler := newTestHost(t)

	// Nothing pasted: strict policy, no banner, no template.
	home := get(t, handler, "/")
	if !strings.Contains(home.Header().Get("Content-Security-Policy"), "script-src 'self';") {
		t.Fatal("public CSP should be strict with no third-party code")
	}
	if strings.Contains(home.Body.String(), "site-head-code") {
		t.Fatal("no code, no template")
	}

	snippet := `<script async src="https://www.googletagmanager.com/gtag/js?id=G-1"></script><script>window.dataLayer=[];</script>`
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "headCode": snippet, "cookieConsent": "required"}, nil)

	home = get(t, handler, "/")
	csp := home.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "script-src 'self' 'unsafe-inline' https:") {
		t.Fatalf("public CSP should allow the pasted code: %q", csp)
	}
	body := home.Body.String()
	mustContain(t, body, `<template id="site-head-code">`, "with consent required the code waits in a template")
	mustContain(t, body, `googletagmanager.com`, "the pasted code is present")
	mustContain(t, body, `src="`+consentScriptURL+`"`, "the consent script is loaded")
	mustContain(t, body, `data-consent="true"`, "the banner is rendered")
	if !strings.Contains(get(t, handler, "/admin").Header().Get("Content-Security-Policy"), "script-src 'self';") {
		t.Fatal("the admin area must keep the strict script policy")
	}
	if rec := get(t, handler, "/_gosx/site/consent.js"); rec.Code != http.StatusOK {
		t.Fatal("consent script not served")
	}

	// Consent switched off: injected directly, no banner.
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "headCode": snippet}, nil)
	body = get(t, handler, "/").Body.String()
	if strings.Contains(body, "site-head-code") || strings.Contains(body, "data-consent") {
		t.Fatal("with consent off the code should be injected directly")
	}
	mustContain(t, body, `<script async src="https://www.googletagmanager.com/gtag/js?id=G-1"></script>`, "the code is injected verbatim")
}

func TestRedirectHelpersAreSafe(t *testing.T) {
	if cleanRedirectPath("https://evil.example/") != "" || cleanRedirectPath("//evil") != "" || cleanRedirectPath("/ok?x=1#f") != "/ok" {
		t.Fatal("redirect paths must stay on this site")
	}
	table := parseRedirectLines("/a -> /b\n/b -> /c")
	if table["/a"] != "/b" || table["/b"] != "/c" {
		t.Fatalf("parse = %v", table)
	}
}

func TestRelaxedPolicyHasNoDuplicateDirectives(t *testing.T) {
	_, handler := newTestHost(t)
	postSettings(t, handler, map[string]string{"title": "X", "headCode": "<script>1</script>"}, nil)
	csp := get(t, handler, "/").Header().Get("Content-Security-Policy")
	if strings.Count(csp, "connect-src") != 1 || strings.Count(csp, "script-src") != 1 {
		t.Fatalf("duplicate directives in relaxed policy: %q", csp)
	}
	mustContain(t, csp, "connect-src 'self' https:", "pasted services may call home")
}
