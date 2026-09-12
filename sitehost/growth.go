package sitehost

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
	"time"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// growth.go is how a site gets found and keeps its links working.
//
// The audit found that Studio emitted no sitemap, no robots rules, no
// structured data, and no redirects — renaming a page silently killed every
// inbound link to it — and that an owner could not even paste an analytics
// snippet. All of that is here, and the two pieces that touch a visitor's
// privacy (third-party code, consent) are explicit choices in Settings.

const (
	sitemapPath       = "/sitemap.xml"
	robotsPath        = "/robots.txt"
	consentScriptPath = "/_gosx/site/consent.js"
	redirectsKey      = "redirects"
	headCodeKey       = "headCode"
	consentKey        = "cookieConsent" // "" (required when code exists) or "off"
	redirectMaxHops   = 5
)

func (h *Host) mountGrowth(mux *http.ServeMux) {
	mux.HandleFunc("GET "+sitemapPath, h.handleSitemap)
	mux.HandleFunc("GET "+robotsPath, h.handleRobots)
	mux.Handle("GET "+consentScriptPath, consentScriptHandler())
}

// absoluteBase is the origin absolute links are built on: the configured
// website address, or the request's own host before one is set.
func (h *Host) absoluteBase(r *http.Request) string {
	if base := strings.TrimRight(strings.TrimSpace(h.settings().BaseURL), "/"); base != "" {
		return base
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// ---------- sitemap.xml and robots.txt ----------

type sitemapURL struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod,omitempty"`
}

type sitemapSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (h *Host) handleSitemap(w http.ResponseWriter, r *http.Request) {
	base := h.absoluteBase(r)
	set := sitemapSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	if h.SetupComplete() {
		for _, page := range h.livePages() {
			entry := sitemapURL{Loc: base + publicPath(page.Slug)}
			if !page.Updated.IsZero() {
				entry.LastMod = page.Updated.UTC().Format("2006-01-02")
			}
			set.URLs = append(set.URLs, entry)
		}
		if posts := h.livePosts(); len(posts) > 0 {
			set.URLs = append(set.URLs, sitemapURL{Loc: base + blogPath})
			for _, post := range posts {
				entry := sitemapURL{Loc: base + postPath(post.Slug)}
				if !post.Updated.IsZero() {
					entry.LastMod = post.Updated.UTC().Format("2006-01-02")
				}
				set.URLs = append(set.URLs, entry)
			}
		}
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(set)
}

func (h *Host) handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if !h.SetupComplete() {
		_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
		return
	}
	_, _ = w.Write([]byte("User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /setup\nSitemap: " + h.absoluteBase(r) + sitemapPath + "\n"))
}

// ---------- redirects ----------

// redirects is the old-path → new-path table, stored as JSON in settings.
func (h *Host) redirects() map[string]string {
	out := map[string]string{}
	raw := strings.TrimSpace(h.settings().Metadata[redirectsKey])
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func (h *Host) setRedirects(table map[string]string) error {
	return h.updateSettingsMetadata(func(metadata cmsstore.Metadata) {
		if len(table) == 0 {
			delete(metadata, redirectsKey)
			return
		}
		data, err := json.Marshal(table)
		if err == nil {
			metadata[redirectsKey] = string(data)
		}
	})
}

func cleanRedirectPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") || strings.ContainsAny(value, " \r\n\t") {
		return ""
	}
	if index := strings.IndexAny(value, "?#"); index >= 0 {
		value = value[:index]
	}
	if len(value) > 1 {
		value = strings.TrimRight(value, "/")
	}
	return value
}

// recordRedirect remembers that a page moved, so the old address keeps
// working. Existing entries that pointed at the old address are re-pointed
// at the new one, so a page renamed twice never leaves a chain behind.
func (h *Host) recordRedirect(from, to string) error {
	from, to = cleanRedirectPath(from), cleanRedirectPath(to)
	if from == "" || to == "" || from == to || from == "/" {
		return nil
	}
	table := h.redirects()
	for key, target := range table {
		if target == from {
			table[key] = to
		}
	}
	delete(table, to) // the new address is a real page now, never a redirect
	table[from] = to
	return h.setRedirects(table)
}

// resolveRedirect follows the table from a path that is not a live page.
func (h *Host) resolveRedirect(path string) (string, bool) {
	table := h.redirects()
	current := cleanRedirectPath(path)
	seen := map[string]bool{}
	found := false
	for hop := 0; hop < redirectMaxHops; hop++ {
		next, ok := table[current]
		if !ok || seen[next] || next == current {
			break
		}
		seen[current] = true
		current, found = next, true
	}
	return current, found
}

// parseRedirectLines reads the Settings textarea: one "/old /new" (or
// "/old -> /new") per line.
func parseRedirectLines(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.ReplaceAll(line, "->", " "))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		from, to := cleanRedirectPath(fields[0]), cleanRedirectPath(fields[1])
		if from == "" || to == "" || from == to || from == "/" {
			continue
		}
		out[from] = to
	}
	return out
}

func formatRedirectLines(table map[string]string) string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+" -> "+table[key])
	}
	return strings.Join(lines, "\n")
}

// ---------- structured data ----------

// structuredData describes the site and the page in schema.org terms, from
// what the owner already told the wizard. Search engines read it; nobody has
// to write it.
func (h *Host) structuredData(settings cmsstore.SiteSettings, brand Brand, page cmsstore.Page, base string) []map[string]any {
	siteTitle := firstNonEmpty(settings.Title, h.opts.SiteTitle)
	pageURL := base + publicPath(page.Slug)

	orgType := "Organization"
	switch settings.Metadata["siteKind"] {
	case "food", "shop", "services":
		orgType = "LocalBusiness"
	}
	org := map[string]any{
		"@context": "https://schema.org",
		"@type":    orgType,
		"name":     siteTitle,
		"url":      base + "/",
	}
	if settings.Description != "" {
		org["description"] = settings.Description
	}
	if brand.LogoURL != "" {
		org["logo"] = absoluteURL(base, brand.LogoURL)
	}
	if brand.Email != "" {
		org["email"] = brand.Email
	}
	if brand.Phone != "" {
		org["telephone"] = brand.Phone
	}
	if location := strings.TrimSpace(settings.Metadata["contactLocation"]); location != "" {
		org["address"] = location
	}
	if len(brand.Social) > 0 {
		same := make([]string, 0, len(brand.Social))
		for _, network := range SocialNetworks() {
			if link, ok := brand.Social[network.Key]; ok {
				same = append(same, link)
			}
		}
		org["sameAs"] = same
	}

	site := map[string]any{
		"@context": "https://schema.org",
		"@type":    "WebSite",
		"name":     siteTitle,
		"url":      base + "/",
	}

	webpage := map[string]any{
		"@context": "https://schema.org",
		"@type":    "WebPage",
		"name":     page.Title,
		"url":      pageURL,
	}
	if description := firstNonEmpty(page.Metadata["metaDescription"], page.Description); description != "" {
		webpage["description"] = description
	}
	if !page.Updated.IsZero() {
		webpage["dateModified"] = page.Updated.UTC().Format(time.RFC3339)
	}
	return []map[string]any{org, site, webpage}
}

func absoluteURL(base, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

// renderJSONLD emits the data block. A JSON-LD script is data, not code:
// browsers never execute it, so the Content Security Policy does not apply.
func renderJSONLD(items []map[string]any) gosx.Node {
	if len(items) == 0 {
		return gosx.Fragment()
	}
	data, err := json.Marshal(items)
	if err != nil {
		return gosx.Fragment()
	}
	safe := strings.ReplaceAll(string(data), "</", "<\\/")
	return gosx.El("script", gosx.Attrs(gosx.Attr("type", "application/ld+json")), gosx.RawHTML(safe))
}

// ---------- third-party code and consent ----------

func (h *Host) headCode() string {
	return strings.TrimSpace(h.settings().Metadata[headCodeKey])
}

func (h *Host) consentRequired() bool {
	return h.settings().Metadata[consentKey] != "off"
}

// renderHeadCode places the owner's pasted code. With consent required it
// waits in a template until the visitor accepts; the consent script moves it
// into the head. Without, it is inserted directly.
func renderHeadCode(code string, consent bool) gosx.Node {
	if code == "" {
		return gosx.Fragment()
	}
	if !consent {
		return gosx.RawHTML(code)
	}
	return gosx.Fragment(
		gosx.El("template", gosx.Attrs(gosx.Attr("id", "site-head-code")), gosx.RawHTML(code)),
		gosx.El("script", gosx.Attrs(gosx.Attr("src", consentScriptPath), gosx.Attr("defer", "defer"))),
	)
}

func renderConsentBanner() gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "site-consent"), gosx.Attr("data-consent", "true"),
		gosx.Attr("role", "region"), gosx.Attr("aria-label", "Cookies"), gosx.Attr("hidden", "hidden"),
	),
		gosx.El("p", nil, gosx.Text("This site uses cookies from other services to understand how it's used. You can say no and everything still works.")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-consent__actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "site-button"), gosx.Attr("data-consent-accept", "true")), gosx.Text("That's fine")),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "site-consent__decline"), gosx.Attr("data-consent-decline", "true")), gosx.Text("No thanks")),
		),
	)
}

// publicCSP is the policy for visitor-facing pages. Once the owner has
// pasted code from another service, scripts from that service (external and
// inline) have to be allowed for it to work; the admin area keeps the strict
// policy regardless.
func (h *Host) publicCSP() string {
	if h.headCode() == "" {
		return contentSecurityPolicy
	}
	policy := strings.Replace(contentSecurityPolicy, "script-src 'self';", "script-src 'self' 'unsafe-inline' https:;", 1)
	return strings.Replace(policy, "connect-src 'self';", "connect-src 'self' https:;", 1)
}
