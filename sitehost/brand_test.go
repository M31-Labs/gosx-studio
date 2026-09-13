package sitehost

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// postSettings posts the Settings form as a browser would: multipart, with
// the token as a field and optional image files.
func postSettings(t *testing.T, handler http.Handler, fields map[string]string, files map[string][]byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("_csrf", csrfToken(t, handler))
	for key, value := range fields {
		_ = writer.WriteField(key, value)
	}
	for field, data := range files {
		part, err := writer.CreateFormFile(field, field+".png")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write(data)
	}
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/admin/settings", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestFreshSiteGetsAGeneratedFavicon(t *testing.T) {
	_, handler := newTestHost(t)
	body := get(t, handler, "/").Body.String()
	mustContain(t, body, `<link rel="icon" href="data:image/svg+xml,`, "a site with no icon gets a generated one")
	mustContain(t, body, `type="image/svg+xml"`, "the generated icon declares its type")
	// The letter mark is the site's initial in the accent colour.
	mustContain(t, body, "%3EW%3C", "the letter mark is the site's initial")
}

func TestLogoAndFaviconUploadThroughSettings(t *testing.T) {
	host, handler := newTestHost(t)
	rec := postSettings(t, handler, map[string]string{
		"title": "Wildflower Bakery", "description": "Bread.", "baseURL": "https://wildflower.example",
		"headerLayout": "centered", "footerText": "42 Mill Lane, Oakland — open Tue to Sun",
		"socialInstagram": "instagram.com/wildflower", "socialX": "javascript:alert(1)",
	}, map[string][]byte{"logo": tinyPNG(t), "favicon": tinyPNG(t)})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("settings save = %d: %s", rec.Code, rec.Body.String())
	}

	brand := host.brand()
	if !strings.HasPrefix(brand.LogoURL, "/uploads/") || !strings.HasPrefix(brand.FaviconURL, "/uploads/") {
		t.Fatalf("uploads not stored: %+v", brand)
	}
	if brand.HeaderLayout != "centered" {
		t.Fatalf("header layout = %q", brand.HeaderLayout)
	}
	if brand.Social["instagram"] != "https://instagram.com/wildflower" {
		t.Fatalf("bare social host should be prefixed with https, got %q", brand.Social["instagram"])
	}
	if _, ok := brand.Social["x"]; ok {
		t.Fatal("a javascript: link was accepted as a social link")
	}

	home := get(t, handler, "/").Body.String()
	mustContain(t, home, `<img class="site-logo" src="`+brand.LogoURL+`" alt="Wildflower Bakery"`, "the logo replaces the site name in the header")
	mustContain(t, home, `class="site-header site-header--centered"`, "the header layout applies")
	mustContain(t, home, `<link rel="icon" href="`+brand.FaviconURL+`"`, "the uploaded favicon is linked")
	mustContain(t, home, "42 Mill Lane, Oakland", "footer text renders")
	mustContain(t, home, `href="https://instagram.com/wildflower" rel="me noopener"`, "social links open safely")
	mustContain(t, home, `Instagram</span></a>`, "social links are labelled")
	mustContain(t, home, `property="og:image" content="https://wildflower.example`+brand.LogoURL+`"`, "the logo is the share image when a page has none")
	if !host.SetupComplete() {
		t.Fatal("saving brand settings wiped the setup marker")
	}

	// The editor canvas shows the same header and footer.
	canvas := get(t, handler, "/admin/edit/"+firstPageID(t, host, "menu")).Body.String()
	mustContain(t, canvas, `class="site-logo"`, "the canvas shows the logo")
	mustContain(t, canvas, "42 Mill Lane, Oakland", "the canvas shows the footer")

	// Remove the logo; the name comes back.
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "remove_logo": "1"}, nil)
	if host.brand().LogoURL != "" {
		t.Fatal("remove logo did not clear it")
	}
	mustContain(t, get(t, handler, "/").Body.String(), `class="site-brand" href="/">Wildflower Bakery<`, "the site name returns when the logo is removed")
}

func TestFooterCarriesWizardContactDetails(t *testing.T) {
	host, handler := newUnbuiltHost(t)
	post(t, handler, "/setup", map[string][]string{
		"step": {"4"}, "siteTitle": {"Corner Shop"}, "kind": {"shop"},
		"email": {"hi@corner.example"}, "phone": {"0161 496 0000"},
	})
	if !host.SetupComplete() {
		t.Fatal("setup did not complete")
	}
	footer := get(t, handler, "/").Body.String()
	mustContain(t, footer, `href="mailto:hi@corner.example"`, "email from setup is in the footer")
	mustContain(t, footer, `href="tel:01614960000"`, "phone from setup is in the footer, dialable")
	mustContain(t, footer, "Corner Shop", "the site name closes the footer")
}

func TestSettingsRefusesANonImageLogoInPlainWords(t *testing.T) {
	host, handler := newTestHost(t)
	rec := postSettings(t, handler, map[string]string{"title": "X"}, map[string][]byte{"logo": []byte("not a picture")})
	if rec.Code != http.StatusOK {
		t.Fatalf("bad logo = %d, want 200 with an error message", rec.Code)
	}
	mustContain(t, rec.Body.String(), "logo doesn", "the refusal names the field")
	if host.brand().LogoURL != "" {
		t.Fatal("a non-image was stored as the logo")
	}
}
