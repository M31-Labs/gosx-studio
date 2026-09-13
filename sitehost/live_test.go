package sitehost

import (
	"net/url"
	"strings"
	"testing"
)

func TestLiveSectionsFollowTheSite(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"posts","variant":"three","fields":{"heading":"News"}},{"kind":"products","variant":"three","fields":{"heading":"Buy"}}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	if strings.Contains(public, "site-posts") || strings.Contains(public, "site-products") {
		t.Fatal("with no posts or products, the live sections stay off the page")
	}
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, "Your newest posts appear here", "the canvas explains the empty section")
	mustContain(t, editor, `data-composite="products"`, "and keeps the block")

	postID := createPost(t, handler, "Harvest supper")
	savePost(t, handler, postID, `{"title":"Harvest supper","slug":"harvest-supper","blocks":[{"kind":"paragraph","text":"Book a seat."}]}`)
	post(t, handler, "/admin/api/posts/"+postID+"/publish", url.Values{})
	pid := createProduct(t, handler, "Sourdough loaf", "6.50")
	saveProduct(t, handler, pid, map[string]string{"name": "Sourdough loaf", "slug": "sourdough", "price": "6.50", "imageCount": "0", "variantCount": "0", "active": "1"})
	public = get(t, handler, "/menu").Body.String()
	mustContain(t, public, `<h2 class="site-posts__heading">News</h2><ul class="site-posts"><li class="site-post-card"><h2 class="site-post-card__title"><a href="/blog/harvest-supper">Harvest supper</a>`, "the latest post shows up by itself")
	mustContain(t, public, `<h2 class="site-products__heading">Buy</h2><ul class="site-products"><li class="site-product-card">`, "so does the product")
	mustContain(t, public, "Sourdough loaf", "by name")
}

func TestStartersAreBuiltFromSections(t *testing.T) {
	for _, kind := range []string{"shop", "services", "food", "portfolio", "community", "simple"} {
		pages := StarterSiteFor(SetupAnswers{SiteTitle: "Test", Tagline: "A tagline to keep", Kind: kind, Location: "1 High Street"})
		home := pages[0].Body.Blocks
		if home[0].Key != "hero" || home[0].Values["headline"].String != "A tagline to keep" {
			t.Fatalf("%s home opens with a hero carrying the tagline: %+v", kind, home[0].Values)
		}
		if home[len(home)-1].Key != "cta" {
			t.Fatalf("%s home ends with a call to action", kind)
		}
	}
	_, handler := newTestHost(t)
	home := get(t, handler, "/").Body.String()
	mustContain(t, home, `class="site-hero site-hero--center"`, "a fresh food site opens with a hero")
	mustContain(t, home, `class="site-hours site-hours--inline"`, "then its hours")
	mustContain(t, home, `class="site-cta site-cta--band"`, "then a call to action")
	mustContain(t, get(t, handler, "/visit").Body.String(), "maps.google.com/maps?q=", "and its visit page has a map")
}

func TestAHeroCarriesThePageTitle(t *testing.T) {
	host, handler := newTestHost(t)
	home := get(t, handler, "/").Body.String()
	mustContain(t, home, `<h1 class="site-title site-title--quiet">Wildflower Bakery</h1>`, "the title stays for search engines but out of sight")
	mustContain(t, get(t, handler, "/menu").Body.String(), `<h1 class="site-title">Menu</h1>`, "pages without a hero keep their title")
	id := firstPageID(t, host, "home")
	mustContain(t, get(t, handler, "/admin/edit/"+id).Body.String(), `class="site-title ed-title--quiet"`, "the canvas shows the name small")
}
