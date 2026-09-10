package sitehost

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

func newTestHost(t *testing.T) (*Host, http.Handler) {
	t.Helper()
	host, err := Open(Options{
		DataPath:        filepath.Join(t.TempDir(), "site.json"),
		SiteTitle:       "Wildflower Bakery",
		SiteDescription: "Sourdough and pastries, baked every morning.",
		BaseURL:         "https://wildflower.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	return host, host.Handler()
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func post(t *testing.T, handler http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func mustContain(t *testing.T, body, want, why string) {
	t.Helper()
	if !strings.Contains(body, want) {
		t.Fatalf("%s: response does not contain %q", why, want)
	}
}

// TestFreshInstallServesAStarterSite is the whole thesis in one test: opening a
// data path with no configuration produces a working, published website.
func TestFreshInstallServesAStarterSite(t *testing.T) {
	_, handler := newTestHost(t)

	rec := get(t, handler, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	mustContain(t, body, "Welcome to Wildflower Bakery", "home page renders seeded content")
	mustContain(t, body, "<title>", "document has a title")
	mustContain(t, body, "site-nav", "home page renders site navigation")

	for _, path := range []string{"/about", "/contact"} {
		if code := get(t, handler, path).Code; code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, code)
		}
	}
}

// TestPublicHeadCarriesSEOAndShareTags covers the gap the audit found: Studio
// itself emits no <head> tags, so meta values were collected and never used.
func TestPublicHeadCarriesSEOAndShareTags(t *testing.T) {
	_, handler := newTestHost(t)
	body := get(t, handler, "/about").Body.String()

	mustContain(t, body, `<title>About — Wildflower Bakery</title>`, "page title composes page and site")
	mustContain(t, body, `<meta name="description"`, "description meta tag is emitted")
	mustContain(t, body, `<link rel="canonical" href="https://wildflower.example/about"`, "canonical URL is absolute")
	mustContain(t, body, `property="og:title"`, "Open Graph title is emitted")
	mustContain(t, body, `property="og:url" content="https://wildflower.example/about"`, "Open Graph URL is absolute")
	mustContain(t, body, `property="og:site_name" content="Wildflower Bakery"`, "Open Graph site name is emitted")
	mustContain(t, body, `name="twitter:card"`, "Twitter card is emitted")
}

func TestAdminPagesAreNotIndexed(t *testing.T) {
	_, handler := newTestHost(t)
	body := get(t, handler, "/admin").Body.String()
	mustContain(t, body, `content="noindex, nofollow"`, "admin pages must not be indexed")
}

// TestDraftPagesStayPrivate proves the draft/live split actually gates the
// public site rather than only decorating the back office.
func TestDraftPagesStayPrivate(t *testing.T) {
	host, handler := newTestHost(t)

	page, err := host.Store().CreatePage(cmsstore.PageInput{Title: "Secret menu", Slug: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if code := get(t, handler, "/secret").Code; code != http.StatusNotFound {
		t.Fatalf("unpublished page is public: GET /secret = %d, want 404", code)
	}

	if _, _, err := host.Store().PublishPage(page.ID); err != nil {
		t.Fatal(err)
	}
	if code := get(t, handler, "/secret").Code; code != http.StatusOK {
		t.Fatalf("published page is not public: GET /secret = %d, want 200", code)
	}
}

func TestNotFoundPageIsHelpfulAndNotIndexed(t *testing.T) {
	_, handler := newTestHost(t)
	rec := get(t, handler, "/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /nope = %d, want 404", rec.Code)
	}
	body := rec.Body.String()
	mustContain(t, body, "Go to the home page", "404 offers a way out")
	mustContain(t, body, `content="noindex, nofollow"`, "404 must not be indexed")
}

// TestCreateEditPublishRoundTrip walks the journey a site owner actually takes.
func TestCreateEditPublishRoundTrip(t *testing.T) {
	host, handler := newTestHost(t)

	rec := post(t, handler, "/admin/pages", url.Values{"title": {"Our services"}, "slug": {"Our Services"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create page = %d, want 303: %s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.HasPrefix(location, "/admin/pages/") {
		t.Fatalf("create redirected to %q, want a page detail URL", location)
	}
	id := strings.TrimPrefix(location, "/admin/pages/")

	page, ok, err := host.Store().PageByID(id)
	if err != nil || !ok {
		t.Fatalf("created page not found: %v", err)
	}
	if page.Slug != "our-services" {
		t.Fatalf("slug = %q, want %q (title should normalize)", page.Slug, "our-services")
	}

	// A new page is not live until it is published.
	if code := get(t, handler, "/our-services").Code; code != http.StatusNotFound {
		t.Fatalf("new page is public before publishing: got %d, want 404", code)
	}

	saveForm := url.Values{
		"title":           {"Our services"},
		"slug":            {"our-services"},
		"body":            {"# What we bake\n\nSourdough, focaccia, and seasonal tarts.\n\n> Best bread in town."},
		"metaDescription": {"Fresh bread and pastries baked daily."},
	}
	if code := post(t, handler, "/admin/pages/"+id, saveForm).Code; code != http.StatusSeeOther {
		t.Fatalf("save page = %d, want 303", code)
	}

	if code := post(t, handler, "/admin/pages/"+id+"/publish", url.Values{}).Code; code != http.StatusSeeOther {
		t.Fatalf("publish page = %d, want 303", code)
	}

	body := get(t, handler, "/our-services").Body.String()
	mustContain(t, body, "Sourdough, focaccia, and seasonal tarts.", "saved paragraph is live")
	mustContain(t, body, "What we bake", "saved heading is live")
	mustContain(t, body, "<blockquote>", "saved quote renders as a quote")
	mustContain(t, body, "Fresh bread and pastries baked daily.", "meta description reaches the head")
}

// TestDuplicateSlugIsRejected closes the audit's slug-uniqueness bug at the
// host boundary: two pages must never claim the same address.
func TestDuplicateSlugIsRejected(t *testing.T) {
	_, handler := newTestHost(t)

	rec := post(t, handler, "/admin/pages", url.Values{"title": {"About again"}, "slug": {"about"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("duplicate slug returned %d, want 200 with an error message", rec.Code)
	}
	mustContain(t, rec.Body.String(), "already uses the address /about",
		"duplicate slug explains the conflict in plain words")
}

func TestEmptySiteShowsAnEmptyStateWithACallToAction(t *testing.T) {
	host, err := Open(Options{
		DataPath:  filepath.Join(t.TempDir(), "empty.json"),
		SiteTitle: "Blank",
		SkipSeed:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, host.Handler(), "/admin/pages").Body.String()
	mustContain(t, body, "No pages yet", "empty page list says so")
	mustContain(t, body, "Create your first one below", "empty state tells the owner what to do")
	mustContain(t, body, "Create page", "empty state is next to a create form")
}

func TestSettingsChangeReachesThePublicHead(t *testing.T) {
	_, handler := newTestHost(t)

	form := url.Values{
		"title":       {"Wildflower Bakehouse"},
		"description": {"Small-batch bread in Oakland."},
		"baseURL":     {"https://bakehouse.example/"},
	}
	if code := post(t, handler, "/admin/settings", form).Code; code != http.StatusSeeOther {
		t.Fatalf("save settings = %d, want 303", code)
	}

	body := get(t, handler, "/about").Body.String()
	mustContain(t, body, "Wildflower Bakehouse", "new site name reaches the public page")
	mustContain(t, body, `property="og:url" content="https://bakehouse.example/about"`,
		"trailing slash is trimmed from the base URL")
}

// TestSiteSurvivesRestart is the durability claim the file store now supports.
func TestSiteSurvivesRestart(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "site.json")
	opts := Options{DataPath: dataPath, SiteTitle: "Persisted", BaseURL: "https://persisted.example"}

	first, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	rec := post(t, first.Handler(), "/admin/pages", url.Values{"title": {"Opening hours"}, "slug": {"hours"}})
	id := strings.TrimPrefix(rec.Header().Get("Location"), "/admin/pages/")
	if code := post(t, first.Handler(), "/admin/pages/"+id, url.Values{
		"title": {"Opening hours"}, "slug": {"hours"}, "body": {"Open Tuesday to Sunday, 7am to 2pm."},
	}).Code; code != http.StatusSeeOther {
		t.Fatal("save failed")
	}
	if code := post(t, first.Handler(), "/admin/pages/"+id+"/publish", url.Values{}).Code; code != http.StatusSeeOther {
		t.Fatal("publish failed")
	}

	second, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	rec = get(t, second.Handler(), "/hours")
	if rec.Code != http.StatusOK {
		t.Fatalf("published page lost across restart: GET /hours = %d, want 200", rec.Code)
	}
	mustContain(t, rec.Body.String(), "Open Tuesday to Sunday", "page content survived restart")
}

func TestSeedDoesNotOverwriteExistingContent(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "site.json")
	opts := Options{DataPath: dataPath, SiteTitle: "Once"}

	first, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := first.Store().ListPages(cmsstore.PageFilter{})

	second, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := second.Store().ListPages(cmsstore.PageFilter{})

	if len(after) != len(before) {
		t.Fatalf("reopening reseeded the site: %d pages before, %d after", len(before), len(after))
	}
}

func TestRuntimeAssetsAndHealthAreMounted(t *testing.T) {
	_, handler := newTestHost(t)

	if code := get(t, handler, "/healthz").Code; code != http.StatusOK {
		t.Fatalf("GET /healthz = %d, want 200", code)
	}
	rec := get(t, handler, publicStylesheetPath)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", publicStylesheetPath, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("stylesheet Content-Type = %q, want text/css", got)
	}
}

func TestNormalizeSlug(t *testing.T) {
	cases := map[string]string{
		"About Us":        "about-us",
		"  Our Services ": "our-services",
		"Café / Bar":      "caf-bar",
		"already-fine":    "already-fine",
		"///":             "",
		"Hours & Info":    "hours-info",
	}
	for input, want := range cases {
		if got := normalizeSlug(input); got != want {
			t.Errorf("normalizeSlug(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestBodyTextRoundTrip proves the editing form does not quietly lose content.
func TestBodyTextRoundTrip(t *testing.T) {
	source := "# A heading\n\nA paragraph.\n\n> A quote.\n\n[Contact us](/contact)"
	got := documentToText(textToDocument(source))
	if got != source {
		t.Fatalf("round trip changed the text:\n got: %q\nwant: %q", got, source)
	}
}

// TestHeadingLevelsSurviveTheRoundTrip guards the level that Studio's own
// renderer currently discards (cms/render/render.go hardcodes h2).
func TestHeadingLevelsSurviveTheRoundTrip(t *testing.T) {
	doc := textToDocument("# Two\n\n## Three\n\n### Four")
	levels := make([]string, 0, 3)
	for _, block := range doc.Blocks {
		levels = append(levels, block.Values["level"].String)
	}
	want := []string{"2", "3", "4"}
	for i, level := range levels {
		if level != want[i] {
			t.Fatalf("heading %d level = %q, want %q", i, level, want[i])
		}
	}
}

func TestAdminPasswordGuardsOnlyTheBackOffice(t *testing.T) {
	host, err := Open(Options{
		DataPath:      filepath.Join(t.TempDir(), "site.json"),
		SiteTitle:     "Guarded",
		AdminPassword: "correct horse",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()

	if code := get(t, handler, "/").Code; code != http.StatusOK {
		t.Fatalf("public site should stay open: GET / = %d, want 200", code)
	}
	if code := get(t, handler, "/admin").Code; code != http.StatusUnauthorized {
		t.Fatalf("admin without credentials = %d, want 401", code)
	}
	if code := get(t, handler, "/admin/pages").Code; code != http.StatusUnauthorized {
		t.Fatalf("admin subpath without credentials = %d, want 401", code)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.SetBasicAuth(AdminUser, "correct horse")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin with correct credentials = %d, want 200", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.SetBasicAuth(AdminUser, "wrong")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("admin with wrong password = %d, want 401", rec.Code)
	}
}

func TestNoAdminPasswordLeavesAdminOpen(t *testing.T) {
	_, handler := newTestHost(t)
	if code := get(t, handler, "/admin").Code; code != http.StatusOK {
		t.Fatalf("unguarded admin = %d, want 200", code)
	}
}
