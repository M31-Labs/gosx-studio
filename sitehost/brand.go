package sitehost

import (
	"html"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// brand.go is the identity around the content: logo, favicon, header layout,
// footer text, social links. All of it lives in the site settings' metadata,
// and one pair of renderers draws it for the public site and the editor
// canvas, so the canvas is never a sketch of the site — it is the site.

const (
	brandLogoKey         = "logoUrl"
	brandFaviconKey      = "faviconUrl"
	brandHeaderLayoutKey = "headerLayout"
	brandFooterTextKey   = "footerText"
)

// SocialNetwork is one place a business can be found.
type SocialNetwork struct {
	Key         string
	Label       string
	Placeholder string
}

// SocialNetworks are the links the footer can carry, in display order.
func SocialNetworks() []SocialNetwork {
	return []SocialNetwork{
		{Key: "instagram", Label: "Instagram", Placeholder: "https://instagram.com/yourbusiness"},
		{Key: "facebook", Label: "Facebook", Placeholder: "https://facebook.com/yourbusiness"},
		{Key: "tiktok", Label: "TikTok", Placeholder: "https://tiktok.com/@yourbusiness"},
		{Key: "youtube", Label: "YouTube", Placeholder: "https://youtube.com/@yourbusiness"},
		{Key: "x", Label: "X", Placeholder: "https://x.com/yourbusiness"},
		{Key: "linkedin", Label: "LinkedIn", Placeholder: "https://linkedin.com/company/yourbusiness"},
	}
}

func socialKey(network string) string {
	return "social" + strings.ToUpper(network[:1]) + network[1:]
}

// Brand is the resolved identity for a site.
type Brand struct {
	LogoURL      string
	FaviconURL   string
	HeaderLayout string // "left" (default) or "centered"
	FooterText   string
	Social       map[string]string
	Email        string
	Phone        string
}

func brandFromSettings(settings cmsstore.SiteSettings) Brand {
	brand := Brand{
		LogoURL:      strings.TrimSpace(settings.Metadata[brandLogoKey]),
		FaviconURL:   strings.TrimSpace(settings.Metadata[brandFaviconKey]),
		HeaderLayout: normalizeHeaderLayout(settings.Metadata[brandHeaderLayoutKey]),
		FooterText:   strings.TrimSpace(settings.Metadata[brandFooterTextKey]),
		Social:       map[string]string{},
		Email:        strings.TrimSpace(settings.Metadata["contactEmail"]),
		Phone:        strings.TrimSpace(settings.Metadata["contactPhone"]),
	}
	for _, network := range SocialNetworks() {
		if link := normalizeSocialLink(settings.Metadata[socialKey(network.Key)]); link != "" {
			brand.Social[network.Key] = link
		}
	}
	return brand
}

func (h *Host) brand() Brand {
	return brandFromSettings(h.settings())
}

func normalizeHeaderLayout(value string) string {
	if strings.TrimSpace(strings.ToLower(value)) == "centered" {
		return "centered"
	}
	return "left"
}

// normalizeSocialLink accepts an https or http URL, or a bare host like
// "instagram.com/name" which it prefixes with https. Anything else — a
// javascript: URL pasted by mistake or on purpose — becomes empty.
func normalizeSocialLink(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return ""
	}
	return parsed.String()
}

// FaviconHref is the icon the browser tab shows. With no upload it is a
// generated letter mark in the site's accent colour, so a fresh site never
// shows the browser's blank-page icon.
func (b Brand) FaviconHref(theme Theme, siteTitle string) string {
	if b.FaviconURL != "" {
		return b.FaviconURL
	}
	letter := "•"
	if r, size := utf8.DecodeRuneInString(strings.TrimSpace(siteTitle)); size > 0 && unicode.IsLetter(r) || unicode.IsDigit(r) {
		letter = string(unicode.ToUpper(r))
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64">` +
		`<rect width="64" height="64" rx="12" fill="` + theme.EffectiveAccent() + `"/>` +
		`<text x="32" y="43" text-anchor="middle" font-family="system-ui,sans-serif" font-size="36" font-weight="700" fill="#ffffff">` +
		html.EscapeString(letter) + `</text></svg>`
	return "data:image/svg+xml," + url.PathEscape(svg)
}

// ---------- shared renderers ----------

// renderSiteHeader draws the header for the public site and the editor
// canvas. In the editor, links are inert so a click edits rather than
// navigates.
func (h *Host) renderSiteHeader(settings cmsstore.SiteSettings, brand Brand, activeSlug string, inEditor bool) gosx.Node {
	siteTitle := firstNonEmpty(settings.Title, h.opts.SiteTitle)
	chrome := chromeFromSettings(settings)
	link := func(href, label, slug string, extra ...any) gosx.Node {
		attrs := append([]any{gosx.Attr("href", href)}, extra...)
		if inEditor {
			attrs = append([]any{gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}, extra...)
		}
		if slug != "" && slug == activeSlug {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		return gosx.El("a", gosx.Attrs(attrs...), gosx.Text(label))
	}

	links := make([]gosx.Node, 0, 8)
	for _, entry := range h.navTree() {
		if entry.Page.Slug == homeSlug {
			continue
		}
		if len(entry.Children) == 0 {
			links = append(links, link(publicPath(entry.Page.Slug), entry.Page.Title, entry.Page.Slug))
			continue
		}
		children := make([]gosx.Node, 0, len(entry.Children))
		for _, child := range entry.Children {
			children = append(children, link(publicPath(child.Slug), child.Title, child.Slug))
		}
		open := entry.Page.Slug == activeSlug
		for _, child := range entry.Children {
			open = open || child.Slug == activeSlug
		}
		groupAttrs := []any{gosx.Attr("class", "site-nav__group")}
		if open {
			groupAttrs = append(groupAttrs, gosx.Attr("data-open", "true"))
		}
		links = append(links, gosx.El("div", gosx.Attrs(groupAttrs...),
			link(publicPath(entry.Page.Slug), entry.Page.Title, entry.Page.Slug, gosx.Attr("class", "site-nav__parent"), gosx.Attr("aria-haspopup", "true")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-nav__menu")), gosx.Fragment(children...))))
	}
	if h.shopInMenu() {
		links = append(links, link(shopPath, h.shopTitle(), "shop"))
	}
	if h.blogInMenu() {
		links = append(links, link(blogPath, h.blogTitle(), "blog"))
	}
	if chrome.MenuButton != "" && chrome.MenuButtonTo != "" {
		links = append(links, link(chrome.MenuButtonTo, chrome.MenuButton, "", gosx.Attr("class", "site-nav__cta button button--primary")))
	}

	var mark gosx.Node
	if brand.LogoURL != "" {
		mark = gosx.El("img", gosx.Attrs(
			gosx.Attr("class", "site-logo"),
			gosx.Attr("src", brand.LogoURL),
			gosx.Attr("alt", siteTitle),
		))
	} else {
		mark = gosx.Text(siteTitle)
	}
	brandAttrs := []any{gosx.Attr("class", "site-brand"), gosx.Attr("href", "/")}
	if inEditor {
		brandAttrs = []any{gosx.Attr("class", "site-brand"), gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}
	}
	headerClass := "site-header site-header--" + brand.HeaderLayout
	if chrome.Sticky {
		headerClass += " site-header--sticky"
	}

	return gosx.Fragment(
		renderAnnouncement(chrome, inEditor),
		gosx.El("header", gosx.Attrs(
			gosx.Attr("class", headerClass),
			gosx.Attr("data-header-layout", brand.HeaderLayout),
		),
			gosx.El("a", gosx.Attrs(brandAttrs...), mark),
			gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-nav"), gosx.Attr("aria-label", "Site")),
				gosx.Fragment(links...)),
		),
	)
}

// renderSiteFooter draws the footer: the owner's text, social links, contact
// details, and the site name with the year.
func (h *Host) renderSiteFooter(settings cmsstore.SiteSettings, brand Brand) gosx.Node {
	return h.renderSiteFooterIn(settings, brand, false)
}

// renderSiteFooterIn draws the footer for the site or the canvas: the
// owner's text and contact details, the menu and extra links when asked
// for, social marks, and the site name with the year.
func (h *Host) renderSiteFooterIn(settings cmsstore.SiteSettings, brand Brand, inEditor bool) gosx.Node {
	siteTitle := firstNonEmpty(settings.Title, h.opts.SiteTitle)
	chrome := chromeFromSettings(settings)
	inert := func(href string) []any {
		if inEditor {
			return []any{gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}
		}
		return []any{gosx.Attr("href", href)}
	}

	about := []gosx.Node{}
	if brand.FooterText != "" {
		about = append(about, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__text")), gosx.Text(brand.FooterText)))
	}
	contact := make([]gosx.Node, 0, 2)
	if brand.Email != "" {
		contact = append(contact, gosx.El("a", gosx.Attrs(inert("mailto:"+brand.Email)...), gosx.Text(brand.Email)))
	}
	if brand.Phone != "" {
		contact = append(contact, gosx.El("a", gosx.Attrs(inert("tel:"+strings.ReplaceAll(brand.Phone, " ", ""))...), gosx.Text(brand.Phone)))
	}
	if len(contact) > 0 {
		about = append(about, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__contact")), gosx.Fragment(contact...)))
	}
	if h.shopInMenu() {
		about = append(about, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__text")),
			gosx.El("a", gosx.Attrs(inert(customerPath)...), gosx.Text("Your orders"))))
	}

	columns := []gosx.Node{}
	if len(about) > 0 {
		columns = append(columns, gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-footer__col site-footer__col--about")), gosx.Fragment(about...)))
	}
	if chrome.FooterMenu {
		items := []gosx.Node{}
		for _, entry := range h.navTree() {
			items = append(items, gosx.El("li", nil, gosx.El("a", gosx.Attrs(inert(publicPath(entry.Page.Slug))...), gosx.Text(entry.Page.Title))))
			for _, child := range entry.Children {
				items = append(items, gosx.El("li", gosx.Attrs(gosx.Attr("class", "site-footer__sub")), gosx.El("a", gosx.Attrs(inert(publicPath(child.Slug))...), gosx.Text(child.Title))))
			}
		}
		if h.shopInMenu() {
			items = append(items, gosx.El("li", nil, gosx.El("a", gosx.Attrs(inert(shopPath)...), gosx.Text(h.shopTitle()))))
		}
		if h.blogInMenu() {
			items = append(items, gosx.El("li", nil, gosx.El("a", gosx.Attrs(inert(blogPath)...), gosx.Text(h.blogTitle()))))
		}
		if len(items) > 0 {
			columns = append(columns, gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-footer__col"), gosx.Attr("aria-label", "Pages")),
				gosx.El("h2", gosx.Attrs(gosx.Attr("class", "site-footer__head")), gosx.Text("Pages")),
				gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-footer__links")), gosx.Fragment(items...))))
		}
	}
	if len(chrome.FooterLinks) > 0 {
		items := make([]gosx.Node, 0, len(chrome.FooterLinks))
		for _, link := range chrome.FooterLinks {
			attrs := append([]any{}, linkAttrs(link[1])...)
			if inEditor {
				attrs = inert(link[1])
			}
			items = append(items, gosx.El("li", nil, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(link[0]))))
		}
		columns = append(columns, gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-footer__col"), gosx.Attr("aria-label", "More")),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "site-footer__head")), gosx.Text("More")),
			gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-footer__links")), gosx.Fragment(items...))))
	}
	if social := renderSocialIcons(brand, inEditor); len(brand.Social) > 0 {
		columns = append(columns, gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-footer__col")),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "site-footer__head")), gosx.Text("Find us")), social))
	}

	nodes := []gosx.Node{}
	if len(columns) > 0 {
		nodes = append(nodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-footer__grid")), gosx.Fragment(columns...)))
	}
	nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__meta")),
		gosx.Text("© "+time.Now().Format("2006")+" "+siteTitle)))
	return gosx.El("footer", gosx.Attrs(gosx.Attr("class", "site-footer")), gosx.Fragment(nodes...))
}
