package sitehost

import (
	"net/http"
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// chrome.go is the frame around every page: the announcement bar, the
// header's menu with its folders and its button, and the footer's columns.
// Everything lives in the site settings and one pair of renderers draws it
// for the public site and the editor canvas alike.

const (
	announceTextKey  = "announceText"
	announceLinkKey  = "announceLink"
	announceOnKey    = "announceOn"
	headerStickyKey  = "headerSticky"
	menuButtonKey    = "menuButtonLabel"
	menuButtonURLKey = "menuButtonUrl"
	footerMenuKey    = "footerMenu"
	footerLinksKey   = "footerLinks"
	pageNavParentKey = "navParent"
)

// Chrome is the resolved frame.
type Chrome struct {
	AnnounceText string
	AnnounceLink string
	AnnounceOn   bool
	Sticky       bool
	MenuButton   string
	MenuButtonTo string
	FooterMenu   bool
	FooterLinks  [][2]string // label, href
}

func chromeFromSettings(settings cmsstore.SiteSettings) Chrome {
	m := settings.Metadata
	c := Chrome{
		AnnounceText: strings.TrimSpace(m[announceTextKey]),
		AnnounceLink: safeLinkHref(m[announceLinkKey]),
		AnnounceOn:   m[announceOnKey] == "true",
		Sticky:       m[headerStickyKey] == "true",
		MenuButton:   strings.TrimSpace(m[menuButtonKey]),
		MenuButtonTo: safeLinkHref(m[menuButtonURLKey]),
		FooterMenu:   m[footerMenuKey] == "true",
	}
	for _, line := range strings.Split(m[footerLinksKey], "\n") {
		label, href, ok := strings.Cut(line, "|")
		label = strings.TrimSpace(label)
		href = safeLinkHref(strings.TrimSpace(href))
		if ok && label != "" && href != "" {
			c.FooterLinks = append(c.FooterLinks, [2]string{label, href})
		}
	}
	return c
}

// PageNavParent is the page a page sits under in the menu, or "".
func PageNavParent(page cmsstore.Page) string {
	return strings.TrimSpace(page.Metadata[pageNavParentKey])
}

// navEntry is one top-level menu item with the pages grouped under it.
type navEntry struct {
	Page     cmsstore.Page
	Children []cmsstore.Page
}

// navTree groups the menu pages: a page whose parent is in the menu goes
// under it; a page whose parent is hidden or gone shows at the top level.
func (h *Host) navTree() []navEntry {
	pages := h.navPages()
	byID := map[string]bool{}
	for _, page := range pages {
		byID[page.ID] = true
	}
	entries := []navEntry{}
	index := map[string]int{}
	for _, page := range pages {
		if parent := PageNavParent(page); parent != "" && byID[parent] && parent != page.ID {
			continue
		}
		index[page.ID] = len(entries)
		entries = append(entries, navEntry{Page: page})
	}
	for _, page := range pages {
		parent := PageNavParent(page)
		if parent == "" || parent == page.ID || !byID[parent] {
			continue
		}
		if at, ok := index[parent]; ok {
			entries[at].Children = append(entries[at].Children, page)
		} else {
			// The parent is itself a child: keep the menu one level deep.
			index[page.ID] = len(entries)
			entries = append(entries, navEntry{Page: page})
		}
	}
	return entries
}

// menuParents are the pages another page may sit under: top-level pages
// that are not itself.
func (h *Host) menuParents(self cmsstore.Page) []cmsstore.Page {
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return nil
	}
	out := []cmsstore.Page{}
	for _, page := range h.orderPages(pages) {
		if page.ID == self.ID || page.Slug == homeSlug || PageArchived(page) || PageNavParent(page) != "" {
			continue
		}
		out = append(out, page)
	}
	return out
}

// renderAnnouncement is the bar above the header, when it is on.
func renderAnnouncement(c Chrome, inEditor bool) gosx.Node {
	if !c.AnnounceOn || c.AnnounceText == "" {
		return gosx.Fragment()
	}
	var inner gosx.Node = gosx.Text(c.AnnounceText)
	if c.AnnounceLink != "" {
		attrs := append([]any{gosx.Attr("class", "site-announce__link")}, linkAttrs(c.AnnounceLink)...)
		if inEditor {
			attrs = []any{gosx.Attr("class", "site-announce__link"), gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}
		}
		inner = gosx.El("a", gosx.Attrs(attrs...), gosx.Text(c.AnnounceText))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-announce"), gosx.Attr("role", "region"), gosx.Attr("aria-label", "Announcement")), inner)
}

// socialIcons are simple marks for the networks the footer can carry.
var socialIcons = map[string]string{
	"instagram": `<path d="M12 7.3a4.7 4.7 0 1 0 0 9.4 4.7 4.7 0 0 0 0-9.4zm0 7.7a3 3 0 1 1 0-6 3 3 0 0 1 0 6zm5.3-8.2a1.1 1.1 0 1 1-2.2 0 1.1 1.1 0 0 1 2.2 0zM12 3.8c2.5 0 2.8 0 3.8.1 2.5.1 3.7 1.3 3.8 3.8.1 1 .1 1.3.1 3.8s0 2.8-.1 3.8c-.1 2.5-1.3 3.7-3.8 3.8-1 .1-1.3.1-3.8.1s-2.8 0-3.8-.1c-2.5-.1-3.7-1.3-3.8-3.8-.1-1-.1-1.3-.1-3.8s0-2.8.1-3.8c.1-2.5 1.3-3.7 3.8-3.8 1-.1 1.3-.1 3.8-.1zM12 2C9.3 2 9 2 7.9 2.1 4.4 2.2 2.2 4.4 2.1 7.9 2 9 2 9.3 2 12s0 3 .1 4.1c.1 3.5 2.3 5.7 5.8 5.8C9 22 9.3 22 12 22s3 0 4.1-.1c3.5-.1 5.7-2.3 5.8-5.8.1-1.1.1-1.4.1-4.1s0-3-.1-4.1c-.1-3.5-2.3-5.7-5.8-5.8C15 2 14.7 2 12 2z"/>`,
	"facebook":  `<path d="M13.5 22v-8h2.7l.4-3.2h-3.1V8.8c0-.9.3-1.6 1.6-1.6h1.7V4.4c-.3 0-1.3-.1-2.5-.1-2.5 0-4.1 1.5-4.1 4.2v2.3H7.4V14h2.8v8h3.3z"/>`,
	"tiktok":    `<path d="M16.6 5.8a4.3 4.3 0 0 1-1-2.8h-3.1v12.4a2.6 2.6 0 1 1-1.8-2.5V9.7a5.7 5.7 0 1 0 4.9 5.7V9.3a7.4 7.4 0 0 0 4.3 1.4V7.6a4.3 4.3 0 0 1-3.3-1.8z"/>`,
	"youtube":   `<path d="M21.6 7.2c-.2-.9-.9-1.6-1.8-1.8C18.2 5 12 5 12 5s-6.2 0-7.8.4c-.9.2-1.6.9-1.8 1.8C2 8.8 2 12 2 12s0 3.2.4 4.8c.2.9.9 1.6 1.8 1.8 1.6.4 7.8.4 7.8.4s6.2 0 7.8-.4c.9-.2 1.6-.9 1.8-1.8.4-1.6.4-4.8.4-4.8s0-3.2-.4-4.8zM10 15V9l5.2 3L10 15z"/>`,
	"x":         `<path d="M17.8 3h3l-6.7 7.7L22 21h-6.2l-4.8-6.3L5.4 21h-3l7.2-8.2L2 3h6.3l4.4 5.8L17.8 3zm-1.1 16.2h1.7L7.4 4.7H5.6l11.1 14.5z"/>`,
	"linkedin":  `<path d="M6.9 20.5H3.4V9h3.5v11.5zM5.2 7.5a2 2 0 1 1 0-4.1 2 2 0 0 1 0 4.1zM20.6 20.5h-3.5v-5.6c0-1.3 0-3-1.9-3s-2.1 1.4-2.1 2.9v5.7H9.6V9h3.4v1.6c.5-.9 1.6-1.8 3.3-1.8 3.6 0 4.2 2.3 4.2 5.4v6.3z"/>`,
}

// renderSocialIcons draws the footer's social links as marks with labels
// for screen readers.
func renderSocialIcons(brand Brand, inEditor bool) gosx.Node {
	links := make([]gosx.Node, 0, 6)
	for _, network := range SocialNetworks() {
		link, ok := brand.Social[network.Key]
		if !ok {
			continue
		}
		attrs := []any{gosx.Attr("class", "site-social__link"), gosx.Attr("href", link), gosx.Attr("rel", "me noopener"), gosx.Attr("target", "_blank"), gosx.Attr("aria-label", network.Label), gosx.Attr("title", network.Label)}
		if inEditor {
			attrs = []any{gosx.Attr("class", "site-social__link"), gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1"), gosx.Attr("aria-label", network.Label)}
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...),
			gosx.El("svg", gosx.Attrs(gosx.Attr("viewBox", "0 0 24 24"), gosx.Attr("width", "20"), gosx.Attr("height", "20"), gosx.Attr("fill", "currentColor"), gosx.Attr("aria-hidden", "true")), gosx.RawHTML(socialIcons[network.Key])),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-social__name")), gosx.Text(network.Label))))
	}
	if len(links) == 0 {
		return gosx.Fragment()
	}
	return gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-social"), gosx.Attr("aria-label", "Find us elsewhere")), gosx.Fragment(links...))
}

// ---------- settings ----------

func (h *Host) renderChromeFields(settings cmsstore.SiteSettings) gosx.Node {
	c := chromeFromSettings(settings)
	check := func(name, label, hint string, on bool) gosx.Node {
		attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", name), gosx.Attr("value", "true")}
		if on {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")), gosx.El("input", gosx.Attrs(attrs...)), gosx.Text(" "+label)),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(hint)))
	}
	links := make([]string, 0, len(c.FooterLinks))
	for _, link := range c.FooterLinks {
		links = append(links, link[0]+" | "+link[1])
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Announcement bar")),
		adminTextField("announceText", "Announcement", c.AnnounceText, "A line across the top of every page: an offer, a holiday closure, a new opening time."),
		adminTextField("announceLink", "Where it links (optional)", c.AnnounceLink, "A page on your site, or a full address."),
		check("announceOn", "Show the announcement bar", "Turn it off without losing the text.", c.AnnounceOn),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Menu")),
		adminTextField("menuButtonLabel", "Menu button", c.MenuButton, "A standout button at the end of the menu, such as \"Book a table\" or \"Get a quote\". Leave empty for none."),
		adminTextField("menuButtonUrl", "Where the button goes", c.MenuButtonTo, "/contact, or a full address."),
		check("headerSticky", "Keep the header at the top while scrolling", "Handy for long pages.", c.Sticky),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("To group pages under one menu item, open a page in the editor and choose what it sits under.")),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Footer columns")),
		check("footerMenu", "Repeat the menu in the footer", "Every page from the menu, listed at the bottom of each page.", c.FooterMenu),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "footerLinks")), gosx.Text("Extra footer links")),
			gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "footerLinks"), gosx.Attr("name", "footerLinks"), gosx.Attr("rows", "4"), gosx.Attr("placeholder", "Privacy | /privacy\nGift cards | https://…")), gosx.Text(strings.Join(links, "\n"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("One per line: the words, a | sign, then the address."))),
	)
}

func applyChromeFields(r *http.Request, metadata cmsstore.Metadata) {
	set := func(key, value string) {
		if value = strings.TrimSpace(value); value != "" {
			metadata[key] = value
		} else {
			delete(metadata, key)
		}
	}
	set(announceTextKey, r.PostFormValue("announceText"))
	set(announceLinkKey, safeLinkHref(r.PostFormValue("announceLink")))
	set(menuButtonKey, r.PostFormValue("menuButtonLabel"))
	set(menuButtonURLKey, safeLinkHref(r.PostFormValue("menuButtonUrl")))
	lines := []string{}
	for _, line := range strings.Split(r.PostFormValue("footerLinks"), "\n") {
		if label, href, ok := strings.Cut(line, "|"); ok && strings.TrimSpace(label) != "" && safeLinkHref(strings.TrimSpace(href)) != "" {
			lines = append(lines, strings.TrimSpace(label)+" | "+safeLinkHref(strings.TrimSpace(href)))
		}
	}
	set(footerLinksKey, strings.Join(lines, "\n"))
	for key, field := range map[string]string{announceOnKey: "announceOn", headerStickyKey: "headerSticky", footerMenuKey: "footerMenu"} {
		if r.PostFormValue(field) == "true" {
			metadata[key] = "true"
		} else {
			delete(metadata, key)
		}
	}
}
