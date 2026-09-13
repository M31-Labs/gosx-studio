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

	links := make([]gosx.Node, 0, 8)
	for _, page := range h.navPages() {
		if page.Slug == homeSlug {
			continue
		}
		attrs := []any{gosx.Attr("href", publicPath(page.Slug))}
		if inEditor {
			attrs = []any{gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}
		}
		if page.Slug == activeSlug {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(page.Title)))
	}
	if h.shopInMenu() {
		attrs := []any{gosx.Attr("href", shopPath)}
		if inEditor {
			attrs = []any{gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}
		}
		if activeSlug == "shop" {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(h.shopTitle())))
	}
	if h.blogInMenu() {
		attrs := []any{gosx.Attr("href", blogPath)}
		if inEditor {
			attrs = []any{gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}
		}
		if activeSlug == "blog" {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(h.blogTitle())))
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

	return gosx.El("header", gosx.Attrs(
		gosx.Attr("class", "site-header site-header--"+brand.HeaderLayout),
		gosx.Attr("data-header-layout", brand.HeaderLayout),
	),
		gosx.El("a", gosx.Attrs(brandAttrs...), mark),
		gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-nav"), gosx.Attr("aria-label", "Site")),
			gosx.Fragment(links...)),
	)
}

// renderSiteFooter draws the footer: the owner's text, social links, contact
// details, and the site name with the year.
func (h *Host) renderSiteFooter(settings cmsstore.SiteSettings, brand Brand) gosx.Node {
	siteTitle := firstNonEmpty(settings.Title, h.opts.SiteTitle)
	nodes := []gosx.Node{}

	if brand.FooterText != "" {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__text")), gosx.Text(brand.FooterText)))
	}
	if h.shopInMenu() {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__text")),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", customerPath)), gosx.Text("Your orders"))))
	}

	social := make([]gosx.Node, 0, 6)
	for _, network := range SocialNetworks() {
		if link, ok := brand.Social[network.Key]; ok {
			social = append(social, gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "site-social__link"),
				gosx.Attr("href", link),
				gosx.Attr("rel", "me noopener"),
				gosx.Attr("target", "_blank"),
			), gosx.Text(network.Label)))
		}
	}
	if len(social) > 0 {
		nodes = append(nodes, gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-social"), gosx.Attr("aria-label", "Find us elsewhere")),
			gosx.Fragment(social...)))
	}

	contact := make([]gosx.Node, 0, 2)
	if brand.Email != "" {
		contact = append(contact, gosx.El("a", gosx.Attrs(gosx.Attr("href", "mailto:"+brand.Email)), gosx.Text(brand.Email)))
	}
	if brand.Phone != "" {
		contact = append(contact, gosx.El("a", gosx.Attrs(gosx.Attr("href", "tel:"+strings.ReplaceAll(brand.Phone, " ", ""))), gosx.Text(brand.Phone)))
	}
	if len(contact) > 0 {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__contact")), gosx.Fragment(contact...)))
	}

	nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-footer__meta")),
		gosx.Text("© "+time.Now().Format("2006")+" "+siteTitle)))

	return gosx.El("footer", gosx.Attrs(gosx.Attr("class", "site-footer")), gosx.Fragment(nodes...))
}
