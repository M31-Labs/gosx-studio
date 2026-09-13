package sitehost

import (
	"net/url"
	"strings"
	"testing"
)

func TestHeaderAndFooterDesigner(t *testing.T) {
	host, handler := newTestHost(t)
	postSettings(t, handler, map[string]string{
		"title": "Wildflower Bakery", "countVisitors": "on",
		"announceText": "Closed Monday 14 September", "announceLink": "/visit", "announceOn": "true",
		"menuButtonLabel": "Book a table", "menuButtonUrl": "/contact", "headerSticky": "true",
		"footerMenu": "true", "footerLinks": "Privacy | /privacy\nGift cards | https://gifts.example\nbroken line\n | /nowhere",
		"socialInstagram": "https://instagram.com/wildflower", "footerText": "42 Mill Lane, Oakland",
	}, nil)
	home := get(t, handler, "/").Body.String()
	for _, want := range []string{
		`<div class="site-announce" role="region" aria-label="Announcement"><a class="site-announce__link" href="/visit">Closed Monday 14 September</a></div>`,
		`class="site-header site-header--left site-header--sticky"`,
		`<a href="/contact" class="site-nav__cta button button--primary">Book a table</a>`,
		`<h2 class="site-footer__head">Pages</h2>`, `<li><a href="/menu">Menu</a></li>`, `<li><a href="/contact">Contact</a></li>`,
		`<h2 class="site-footer__head">More</h2>`, `<a href="/privacy">Privacy</a>`, `href="https://gifts.example"`,
		`<a class="site-social__link" href="https://instagram.com/wildflower" rel="me noopener" target="_blank" aria-label="Instagram" title="Instagram"><svg`,
		`<p class="site-footer__text">42 Mill Lane, Oakland</p>`,
	} {
		if !strings.Contains(home, want) {
			t.Fatalf("home lacks %q:\n%s", want, home)
		}
	}
	if strings.Contains(home, "broken line") || strings.Contains(home, "/nowhere") {
		t.Fatal("malformed footer links are dropped")
	}

	// Switching the bar off keeps the text for later.
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "countVisitors": "on", "announceText": "Closed Monday 14 September"}, nil)
	home = get(t, handler, "/").Body.String()
	if strings.Contains(home, "site-announce") {
		t.Fatal("the bar is off")
	}
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), `value="Closed Monday 14 September"`, "but the text is kept")

	// Folders: a page sits under another and becomes a drop-down.
	menu := firstPageID(t, host, "menu")
	post(t, handler, "/admin/pages", url.Values{"title": {"Wine list"}, "slug": {"wine"}})
	wine, _, _ := host.store.PageBySlug("wine")
	editor := get(t, handler, "/admin/edit/"+wine.ID).Body.String()
	mustContain(t, editor, `<select id="pageParent" data-meta="parent">`, "the editor offers a parent")
	mustContain(t, editor, `<option value="`+menu+`">Menu</option>`, "such as Menu")
	if strings.Contains(editor, `>Wine list</option>`) {
		t.Fatal("a page cannot sit under itself")
	}
	postJSON(t, handler, "/admin/api/pages/"+wine.ID, `{"title":"Wine list","slug":"wine","description":"x","navParent":"`+menu+`","blocks":[{"kind":"paragraph","text":"Reds and whites."}]}`)
	post(t, handler, "/admin/api/pages/"+wine.ID+"/publish", url.Values{})
	home = get(t, handler, "/").Body.String()
	mustContain(t, home, `<div class="site-nav__group"><a href="/menu" class="site-nav__parent" aria-haspopup="true">Menu</a><div class="site-nav__menu"><a href="/wine">Wine list</a></div></div>`, "the menu has a drop-down")
	mustContain(t, get(t, handler, "/wine").Body.String(), `class="site-nav__group" data-open="true"`, "which is open on the child page")
	mustContain(t, get(t, handler, "/admin/pages").Body.String(), "↳ Wine list", "the pages list shows the nesting")
	// A child cannot become a parent, and hiding the parent hides the child.
	post(t, handler, "/admin/pages", url.Values{"title": {"Rosé"}, "slug": {"rose"}})
	rose, _, _ := host.store.PageBySlug("rose")
	postJSON(t, handler, "/admin/api/pages/"+rose.ID, `{"title":"Rosé","slug":"rose","description":"x","navParent":"`+wine.ID+`","blocks":[{"kind":"paragraph","text":"Pink."}]}`)
	updated, _, _ := host.store.PageByID(rose.ID)
	if PageNavParent(updated) != "" {
		t.Fatal("menus stay one level deep")
	}
	post(t, handler, "/admin/pages/"+menu+"/action", url.Values{"action": {"hide"}})
	home = get(t, handler, "/").Body.String()
	if strings.Contains(home, `href="/wine"`) && !strings.Contains(home, "site-footer") {
		t.Fatal("a hidden parent takes its children out of the menu")
	}
	if strings.Contains(home, "site-nav__group") {
		t.Fatal("no drop-down without its parent")
	}
}
