package sitehost

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

func fullAnswers() SetupAnswers {
	answers := SetupAnswers{
		SiteTitle: "Mill Lane Bakery", Tagline: "Bread worth the walk", Description: "A family bakery on Mill Lane since 2014.", Kind: "food",
		Email: "ana@example.com", Phone: "0161 496 0000", Location: "1 Mill Lane, Oakland",
		Social:   map[string]string{"instagram": "https://instagram.com/milllane"},
		HoursSet: true, PagesSet: true, Publish: true,
		Pages: map[string]bool{"main": true, "about": true, "pricing": true, "gallery": true, "faq": true, "privacy": true},
	}
	answers.Offers = [3]Offer{{Name: "Sourdough loaf", Text: "Baked before dawn", Price: "$8"}, {Name: "Cinnamon knot", Text: "Gone by ten", Price: "$4"}, {}}
	answers.Hours = defaultHours()
	return answers
}

func TestTheTurnkeyBuildFillsEveryPageFromTheAnswers(t *testing.T) {
	answers := fullAnswers()
	pages := StarterSiteFor(answers)
	slugs := []string{}
	for _, page := range pages {
		slugs = append(slugs, page.Slug)
	}
	if got := strings.Join(slugs, " "); got != "home menu visit about pricing photos questions contact privacy" {
		t.Fatalf("pages = %q", got)
	}
	find := func(slug string) StarterPage {
		for _, page := range pages {
			if page.Slug == slug {
				return page
			}
		}
		t.Fatalf("no page %s", slug)
		return StarterPage{}
	}
	menu := find("menu")
	if menu.Body.Blocks[0].Key != "pricing" || menu.Body.Blocks[0].Values["items"].List[1].Object["name"].String != "Cinnamon knot" {
		t.Fatalf("the menu lists the offers as a simple price list: %+v", menu.Body.Blocks[0].Values)
	}
	if find("privacy").HideFromMenu != true || find("privacy").Title != "Privacy" {
		t.Fatal("the privacy page is hidden and matches the settings' one")
	}
	if !find("photos").Publish || find("photos").Body.Blocks[1].Key != "gallery" {
		t.Fatal("the photos page carries an empty gallery to fill")
	}
	faq := find("questions").Body.Blocks[0]
	if faq.Key != "faq" || len(faq.Values["items"].List) != 3 || !strings.Contains(faq.Values["items"].List[2].Object["answer"].String, "1 Mill Lane") {
		t.Fatalf("the questions page asks what a bakery is asked: %+v", faq.Values)
	}
	// Pricing is not ticked by default for food: the menu carries the prices.
	if defaultPages(answers)["pricing"] {
		t.Fatal("a food business gets its prices on the menu unless it asks for a pricing page")
	}
	answers.Kind = "services"
	answers.Pages["gallery"], answers.Pages["faq"] = false, false
	pages = StarterSiteFor(answers)
	slugs = slugs[:0]
	for _, page := range pages {
		slugs = append(slugs, page.Slug)
	}
	if got := strings.Join(slugs, " "); got != "home services about pricing contact privacy" {
		t.Fatalf("services pages = %q", got)
	}
	pricing := find("pricing")
	if pricing.Body.Blocks[0].Values["items"].List[0].Object["price"].String != "$8" || pricing.Body.Blocks[0].Values["items"].List[0].Object["button"].String != "Get in touch" {
		t.Fatalf("pricing plans come from the priced offers: %+v", pricing.Body.Blocks[0].Values)
	}
	contact := find("contact")
	keys := []string{}
	for _, block := range contact.Body.Blocks {
		keys = append(keys, block.Key)
	}
	if !strings.Contains(strings.Join(keys, " "), "map") || !strings.Contains(strings.Join(keys, " "), "hours") {
		t.Fatalf("a services contact page carries the map and the hours: %v", keys)
	}
}

func TestDraftsStayDraftsWhenTheOwnerSaysSo(t *testing.T) {
	host, handler := newUnbuiltHost(t)
	form := url.Values{"step": {"6"}, "siteTitle": {"Quiet Start"}, "kind": {"services"}, "pagesSet": {"1"}, "pagemain": {"1"}, "pageabout": {"1"}}
	rec := post(t, handler, "/setup", form)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("finish: %d", rec.Code)
	}
	if !host.SetupComplete() {
		t.Fatal("setup completes")
	}
	pages, _ := host.store.ListPages(pageFilterAll())
	if len(pages) != 4 { // home, services, about, contact
		t.Fatalf("pages: %d", len(pages))
	}
	for _, page := range pages {
		if host.isLive(page) {
			t.Fatalf("%s went live without publish", page.Slug)
		}
	}
	if code := get(t, handler, "/").Code; code == http.StatusOK {
		t.Fatal("nothing is public yet")
	}
	welcome := getWithCookieOrPlain(t, handler, "/admin?welcome=1")
	mustContain(t, welcome, "Your pages are built and saved as drafts", "the welcome says nothing is live yet")
	mustContain(t, welcome, `data-live="false">Draft</span><strong>Services</strong>`, "and lists each page as a draft")
}

func TestTheWelcomePanelShowsWhatWasBuilt(t *testing.T) {
	host, handler := newTestHost(t)
	body := get(t, handler, "/admin?welcome=1").Body.String()
	mustContain(t, body, `data-welcome="true"`, "the first dashboard visit has the welcome")
	mustContain(t, body, "Your site is live at https://wildflower.example", "with the live address")
	mustContain(t, body, `data-live="true">Live</span><strong>Menu</strong>`, "and every page with its state")
	mustContain(t, body, `href="/admin/edit/`+firstPageID(t, host, "menu")+`">Edit</a>`, "and a way to edit each")
	mustContain(t, body, "Connect your domain", "and the next steps")
	if strings.Contains(get(t, handler, "/admin").Body.String(), `data-welcome="true"`) {
		t.Fatal("the welcome only shows when asked")
	}
}

func TestTheWizardKeepsAnswersWhenGoingBack(t *testing.T) {
	_, handler := newUnbuiltHost(t)
	back := post(t, handler, "/setup", url.Values{"step": {"4"}, "back": {"3"}, "siteTitle": {"Corner Shop"}, "kind": {"shop"}, "offer1Name": {"Mugs"}, "email": {"hi@corner.example"}, "socialinstagram": {"https://instagram.com/corner"}})
	body := back.Body.String()
	mustContain(t, body, "Three things you sell", "back returns to the offers, worded for a shop")
	mustContain(t, body, `value="Mugs"`, "with the offer still there")
	mustContain(t, body, `name="email" value="hi@corner.example"`, "and the email carried as a hidden field")
	mustContain(t, body, `name="socialinstagram" value="https://instagram.com/corner"`, "and the social link")
	pages := post(t, handler, "/setup", url.Values{"step": {"5"}, "siteTitle": {"Corner Shop"}, "kind": {"shop"}, "offer1Name": {"Mugs"}, "offer1Price": {"$12"}})
	mustContain(t, pages.Body.String(), `id="wz-pagepricing" checked="checked"`, "a priced offer ticks the pricing page for a shop")
	mustContain(t, pages.Body.String(), `id="wz-pagegallery"`, "photos are offered")
	if strings.Contains(pages.Body.String(), `id="wz-pagegallery" checked`) {
		t.Fatal("but not ticked until the owner has photos")
	}
	if strings.Contains(pages.Body.String(), `id="wz-pagemain"`) {
		t.Fatal("a shop has no main page of its own; /shop is the shop")
	}
}

func TestSocialLinksAndTheHeaderButtonLandOnTheSite(t *testing.T) {
	host, handler := newUnbuiltHost(t)
	post(t, handler, "/setup", url.Values{"step": {"6"}, "siteTitle": {"Corner Shop"}, "kind": {"shop"}, "phone": {"0161 496 0000"}, "socialinstagram": {"https://instagram.com/corner"}, "socialfacebook": {"javascript:alert(1)"}})
	home := get(t, handler, "/").Body.String()
	mustContain(t, home, `href="https://instagram.com/corner"`, "the Instagram link is in the footer")
	if strings.Contains(home, "javascript:") {
		t.Fatal("an unsafe link never lands")
	}
	mustContain(t, home, `href="tel:01614960000" class="site-nav__cta`, "the header button calls when there is a phone")
	if host.settings().Metadata[footerMenuKey] != "true" {
		t.Fatal("the footer menu is on")
	}
}

// pageFilterAll and getWithCookieOrPlain keep the tests above readable.
func pageFilterAll() cmsstore.PageFilter { return cmsstore.PageFilter{} }

func getWithCookieOrPlain(t *testing.T, handler http.Handler, path string) string {
	t.Helper()
	return get(t, handler, path).Body.String()
}
