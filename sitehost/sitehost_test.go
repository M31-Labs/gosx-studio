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
		SiteKind:        "food",
		Seed:            true,
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
	mustContain(t, body, "Wildflower Bakery", "home page renders seeded content")
	mustContain(t, body, "See the menu", "the food template's call to action is there")
	mustContain(t, body, "<title>", "document has a title")
	mustContain(t, body, "site-nav", "home page renders site navigation")

	for _, path := range []string{"/menu", "/visit", "/contact"} {
		if code := get(t, handler, path).Code; code != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, code)
		}
	}
}

// TestPublicHeadCarriesSEOAndShareTags covers the gap the audit found: Studio
// itself emits no <head> tags, so meta values were collected and never used.
func TestPublicHeadCarriesSEOAndShareTags(t *testing.T) {
	_, handler := newTestHost(t)
	body := get(t, handler, "/menu").Body.String()

	mustContain(t, body, `<title>Menu — Wildflower Bakery</title>`, "page title composes page and site")
	mustContain(t, body, `<meta name="description"`, "description meta tag is emitted")
	mustContain(t, body, `<link rel="canonical" href="https://wildflower.example/menu"`, "canonical URL is absolute")
	mustContain(t, body, `property="og:title"`, "Open Graph title is emitted")
	mustContain(t, body, `property="og:url" content="https://wildflower.example/menu"`, "Open Graph URL is absolute")
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
	if !strings.HasPrefix(location, "/admin/edit/") {
		t.Fatalf("create redirected to %q, want the editor", location)
	}
	id := strings.TrimPrefix(location, "/admin/edit/")

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

	rec := post(t, handler, "/admin/pages", url.Values{"title": {"Menu again"}, "slug": {"menu"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("duplicate slug returned %d, want 200 with an error message", rec.Code)
	}
	mustContain(t, rec.Body.String(), "already uses the address /menu",
		"duplicate slug explains the conflict in plain words")
}

func TestEmptySiteShowsAnEmptyStateWithACallToAction(t *testing.T) {
	host, err := Open(Options{
		DataPath:  filepath.Join(t.TempDir(), "empty.json"),
		SiteTitle: "Blank",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Finish setup without building any pages, which is the state an owner
	// reaches only by deleting everything.
	if _, err := host.Store().SaveSiteSettings(cmsstore.SiteSettingsInput{
		Title:    "Blank",
		Metadata: cmsstore.Metadata{setupCompleteKey: "true"},
	}); err != nil {
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

	body := get(t, handler, "/menu").Body.String()
	mustContain(t, body, "Wildflower Bakehouse", "new site name reaches the public page")
	mustContain(t, body, `property="og:url" content="https://bakehouse.example/menu"`,
		"trailing slash is trimmed from the base URL")
}

// TestSiteSurvivesRestart is the durability claim the file store now supports.
func TestSiteSurvivesRestart(t *testing.T) {
	dataPath := filepath.Join(t.TempDir(), "site.json")
	opts := Options{DataPath: dataPath, SiteTitle: "Persisted", BaseURL: "https://persisted.example", Seed: true}

	first, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	rec := post(t, first.Handler(), "/admin/pages", url.Values{"title": {"Opening hours"}, "slug": {"hours"}})
	id := strings.TrimPrefix(rec.Header().Get("Location"), "/admin/edit/")
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
	opts := Options{DataPath: dataPath, SiteTitle: "Once", Seed: true}

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
		Seed:          true,
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

// ---------- setup wizard ----------

func newUnbuiltHost(t *testing.T) (*Host, http.Handler) {
	t.Helper()
	host, err := Open(Options{
		DataPath:  filepath.Join(t.TempDir(), "new.json"),
		SiteTitle: "Untitled",
		BaseURL:   "https://new.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	return host, host.Handler()
}

func TestFreshSiteSendsTheOwnerToTheWizard(t *testing.T) {
	host, handler := newUnbuiltHost(t)
	if host.SetupComplete() {
		t.Fatal("a brand new site must not report setup as complete")
	}

	rec := get(t, handler, "/admin")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/setup" {
		t.Fatalf("admin on an unbuilt site = %d %q, want 303 to /setup", rec.Code, rec.Header().Get("Location"))
	}

	body := get(t, handler, "/setup").Body.String()
	mustContain(t, body, "build your website", "the wizard opens on step one")
	mustContain(t, body, "your business called", "step one asks for the name")
}

func TestVisitorsSeeComingSoonBeforeSetup(t *testing.T) {
	_, handler := newUnbuiltHost(t)
	rec := get(t, handler, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / on an unbuilt site = %d, want 200", rec.Code)
	}
	mustContain(t, rec.Body.String(), "Coming soon", "visitors get a holding page, not an error")
	mustContain(t, rec.Body.String(), `content="noindex, nofollow"`, "an unbuilt site must not be indexed")
}

func TestWizardWalksThreeStepsAndBuildsTheSite(t *testing.T) {
	host, handler := newUnbuiltHost(t)

	step1 := post(t, handler, "/setup", url.Values{
		"step": {"1"}, "siteTitle": {"Wildflower Bakery"}, "tagline": {"Sourdough, every morning."},
	})
	mustContain(t, step1.Body.String(), "Which sounds most like you?", "step one advances to step two")
	mustContain(t, step1.Body.String(), `value="Wildflower Bakery"`, "step two carries the name forward")

	step2 := post(t, handler, "/setup", url.Values{
		"step": {"2"}, "siteTitle": {"Wildflower Bakery"}, "tagline": {"Sourdough, every morning."},
		"kind": {"food"},
	})
	mustContain(t, step2.Body.String(), "How should people reach you?", "step two advances to step three")

	done := post(t, handler, "/setup", url.Values{
		"step": {"3"}, "siteTitle": {"Wildflower Bakery"}, "tagline": {"Sourdough, every morning."},
		"kind": {"food"}, "email": {"hello@wildflower.example"}, "location": {"42 Mill Lane"},
	})
	if done.Code != http.StatusSeeOther {
		t.Fatalf("finishing the wizard = %d, want 303: %s", done.Code, done.Body.String())
	}

	if !host.SetupComplete() {
		t.Fatal("finishing the wizard must mark setup complete")
	}

	home := get(t, handler, "/")
	if home.Code != http.StatusOK {
		t.Fatalf("home after setup = %d, want 200", home.Code)
	}
	mustContain(t, home.Body.String(), "Wildflower Bakery", "the site carries the name from step one")
	mustContain(t, home.Body.String(), "Sourdough, every morning.", "the tagline lands on the home page")

	// The food template's pages exist and are live.
	for _, path := range []string{"/menu", "/visit", "/contact"} {
		if code := get(t, handler, path).Code; code != http.StatusOK {
			t.Fatalf("GET %s after setup = %d, want 200", path, code)
		}
	}
	mustContain(t, get(t, handler, "/contact").Body.String(), "hello@wildflower.example",
		"the email from step three reaches the contact page")
	mustContain(t, get(t, handler, "/contact").Body.String(), "42 Mill Lane",
		"the location from step three reaches the contact page")
}

func TestWizardRefusesToBuildWithoutAName(t *testing.T) {
	_, handler := newUnbuiltHost(t)
	rec := post(t, handler, "/setup", url.Values{"step": {"1"}, "siteTitle": {"  "}})
	mustContain(t, rec.Body.String(), "Your site needs a name", "step one explains what is missing")
	mustContain(t, rec.Body.String(), "your business called", "step one stays on step one")
}

func TestWizardBackButtonKeepsAnswers(t *testing.T) {
	_, handler := newUnbuiltHost(t)
	rec := post(t, handler, "/setup", url.Values{
		"step": {"2"}, "back": {"1"}, "siteTitle": {"Half Typed"}, "tagline": {"Still thinking"},
	})
	mustContain(t, rec.Body.String(), `value="Half Typed"`, "going back keeps the name")
	mustContain(t, rec.Body.String(), `value="Still thinking"`, "going back keeps the tagline")
}

func TestEachSiteKindProducesItsOwnPages(t *testing.T) {
	expected := map[string][]string{
		"shop":      {"home", "shop", "about", "contact"},
		"services":  {"home", "services", "about", "contact"},
		"food":      {"home", "menu", "visit", "contact"},
		"portfolio": {"home", "work", "about", "contact"},
		"community": {"home", "whats-on", "about", "contact"},
		"simple":    {"home", "contact"},
	}
	for kind, want := range expected {
		pages := StarterSiteFor(SetupAnswers{SiteTitle: "Test", Kind: kind})
		if len(pages) != len(want) {
			t.Fatalf("%s produced %d pages, want %d", kind, len(pages), len(want))
		}
		for i, page := range pages {
			if page.Slug != want[i] {
				t.Errorf("%s page %d = %q, want %q", kind, i, page.Slug, want[i])
			}
			if len(page.Body.Blocks) == 0 {
				t.Errorf("%s page %q has no content", kind, page.Slug)
			}
		}
	}
}

// ---------- the editing canvas ----------

func firstPageID(t *testing.T, host *Host, slug string) string {
	t.Helper()
	page, ok, err := host.Store().PageBySlug(slug)
	if err != nil || !ok {
		t.Fatalf("page %q not found", slug)
	}
	return page.ID
}

func TestEditorRendersTheRealPageNotASchemaCard(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	body := get(t, handler, "/admin/edit/"+id).Body.String()

	// The real page content is on the canvas, editable in place.
	mustContain(t, body, `contenteditable="true"`, "text is editable in place")
	mustContain(t, body, `data-block="heading"`, "blocks carry their kind")
	mustContain(t, body, "Menu", "the page's own content is on the canvas")
	mustContain(t, body, `class="site-article"`, "the canvas uses the public site's own markup")

	// And the metadata card the old Studio canvas drew is not.
	for _, unwanted := range []string{"CanvasBoard HTML surface", "Not configured", "data-studio-page-surface"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("editor canvas leaked internal machinery: %q", unwanted)
		}
	}
}

func TestEditorSaveRewritesTheDocument(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	payload := `{"title":"Our menu","slug":"menu","description":"Everything we bake.","blocks":[
		{"kind":"heading","text":"Breakfast","level":"3"},
		{"kind":"paragraph","text":"Served until eleven."},
		{"kind":"quote","text":"The croissants are unreal."},
		{"kind":"button","text":"Book a table","url":"/contact"}
	]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+id, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("editor save = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("editor save was not accepted: %s", rec.Body.String())
	}

	page, _, _ := host.Store().PageByID(id)
	if page.Title != "Our menu" {
		t.Fatalf("title = %q, want %q", page.Title, "Our menu")
	}
	if len(page.Body.Blocks) != 4 {
		t.Fatalf("saved %d blocks, want 4", len(page.Body.Blocks))
	}

	// Publish and confirm the edit reaches a visitor with the right heading level.
	if _, _, err := host.Store().PublishPage(id); err != nil {
		t.Fatal(err)
	}
	live := get(t, handler, "/menu").Body.String()
	mustContain(t, live, "<h3>Breakfast</h3>", "heading level chosen in the editor survives to the page")
	mustContain(t, live, "Served until eleven.", "paragraph reaches the page")
	mustContain(t, live, "<blockquote>", "quote reaches the page")
	mustContain(t, live, `href="/contact"`, "button target reaches the page")
}

func TestEditorDropsEmptyBlocks(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	payload := `{"title":"Menu","slug":"menu","blocks":[
		{"kind":"paragraph","text":"Kept."},
		{"kind":"paragraph","text":"   "},
		{"kind":"heading","text":"","level":"2"},
		{"kind":"image","url":""}
	]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+id, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	page, _, _ := host.Store().PageByID(id)
	if len(page.Body.Blocks) != 1 {
		t.Fatalf("kept %d blocks, want 1 — empty blocks should not persist", len(page.Body.Blocks))
	}
}

func TestEditorPublishGoesLive(t *testing.T) {
	host, handler := newTestHost(t)
	page, err := host.Store().CreatePage(cmsstore.PageInput{Title: "Specials", Slug: "specials"})
	if err != nil {
		t.Fatal(err)
	}
	if code := get(t, handler, "/specials").Code; code != http.StatusNotFound {
		t.Fatal("new page should not be public yet")
	}

	rec := post(t, handler, "/admin/api/pages/"+page.ID+"/publish", url.Values{})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("publish = %d: %s", rec.Code, rec.Body.String())
	}
	if code := get(t, handler, "/specials").Code; code != http.StatusOK {
		t.Fatal("page should be live after publishing from the editor")
	}
}

func TestEditorRejectsADuplicateAddress(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	payload := `{"title":"Menu","slug":"visit","blocks":[{"kind":"paragraph","text":"Hi."}]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+id, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, `"ok":true`) {
		t.Fatal("editor accepted a slug another page already uses")
	}
	mustContain(t, body, "already uses /visit", "the clash is explained in plain words")
}

func TestEditorScriptIsServed(t *testing.T) {
	_, handler := newTestHost(t)
	rec := get(t, handler, editorScriptPath)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", editorScriptPath, rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/javascript") {
		t.Fatalf("editor script Content-Type = %q", got)
	}
	mustContain(t, rec.Body.String(), "data-editor", "the served script is the editor runtime")
}

// TestEditingALivePageDoesNotTakeItOffline is the bug this model exists to
// prevent: the store keeps one record per page and saving a draft moves that
// record out of the published state, so serving the record directly would drop
// a live page off the internet on the first keystroke.
func TestEditingALivePageDoesNotTakeItOffline(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	if code := get(t, handler, "/menu").Code; code != http.StatusOK {
		t.Fatal("menu should start live")
	}

	payload := `{"title":"Menu","slug":"menu","blocks":[{"kind":"paragraph","text":"A draft nobody should see yet."}]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+id, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	rec := get(t, handler, "/menu")
	if rec.Code != http.StatusOK {
		t.Fatalf("editing a live page took it offline: GET /menu = %d, want 200", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "A draft nobody should see yet.") {
		t.Fatal("unpublished draft text is visible to the public")
	}
	mustContain(t, rec.Body.String(), "Menu", "visitors still see the published version")

	// The menu link must also survive the edit.
	mustContain(t, get(t, handler, "/").Body.String(), `href="/menu"`,
		"a page being edited stays in the site menu")

	// Publishing releases the draft.
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	mustContain(t, get(t, handler, "/menu").Body.String(), "A draft nobody should see yet.",
		"publishing makes the draft live")
}

func TestNeverPublishedPageStaysPrivateWhileEdited(t *testing.T) {
	host, handler := newTestHost(t)
	page, err := host.Store().CreatePage(cmsstore.PageInput{Title: "Draft only", Slug: "draft-only"})
	if err != nil {
		t.Fatal(err)
	}
	payload := `{"title":"Draft only","slug":"draft-only","blocks":[{"kind":"paragraph","text":"Secret."}]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/pages/"+page.ID, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if code := get(t, handler, "/draft-only").Code; code != http.StatusNotFound {
		t.Fatal("a page that was never published must stay private")
	}
}
