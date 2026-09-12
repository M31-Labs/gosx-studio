package sitehost

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/acme/autocert"
	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// domain.go connects the site to the owner's own domain name.
//
// Hosted builders do this in one screen: type the domain, add two DNS
// records, wait, done — and the certificate appears on its own. The default
// host does the same. The owner types the domain in the admin, follows the
// records shown, presses "Check", and — when the server was started with
// HTTPS on — the first visit over https fetches a Let's Encrypt certificate
// through autocert and renews it forever after. Nothing to install, no
// certificate files to manage, no cron.

const (
	domainKey    = "domain"    // the apex name, e.g. example.com
	domainWWWKey = "domainWWW" // "true" when the owner wants www.example.com as the address
)

// lookupHost resolves a name to addresses. Tests replace it.
var lookupHost = func(host string) ([]string, error) {
	return net.LookupHost(host)
}

var domainPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)

// normalizeDomain turns whatever the owner typed — with a scheme, a path,
// capitals, or a leading www. — into the bare apex name, or "" if it is not
// a domain.
func normalizeDomain(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "://") {
		if parsed, err := url.Parse(raw); err == nil {
			raw = parsed.Host
		}
	}
	raw = strings.TrimSuffix(strings.Split(raw, "/")[0], ".")
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	raw = strings.TrimPrefix(raw, "www.")
	if len(raw) > 253 || net.ParseIP(raw) != nil || !domainPattern.MatchString(raw) {
		return ""
	}
	return raw
}

func (o Options) certDir() string {
	if dir := strings.TrimSpace(o.CertDir); dir != "" {
		return dir
	}
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "certs")
}

// Domain is the owner's apex domain, or "" when none is connected.
func (h *Host) Domain() string {
	return normalizeDomain(h.settings().Metadata[domainKey])
}

func (h *Host) prefersWWW() bool {
	return h.settings().Metadata[domainWWWKey] == "true"
}

// canonicalHost is the address visitors should end up on; alternateHost the
// twin that redirects to it.
func (h *Host) canonicalHost() string {
	domain := h.Domain()
	if domain == "" {
		return ""
	}
	if h.prefersWWW() {
		return "www." + domain
	}
	return domain
}

func (h *Host) alternateHost() string {
	domain := h.Domain()
	if domain == "" {
		return ""
	}
	if h.prefersWWW() {
		return domain
	}
	return "www." + domain
}

func (h *Host) publicScheme() string {
	if h.opts.TLS {
		return "https"
	}
	return "http"
}

// HostPolicy is autocert's allow-list: the connected domain and its www
// twin, read live so connecting a domain needs no restart.
func (h *Host) HostPolicy(_ context.Context, host string) error {
	domain := h.Domain()
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if domain != "" && (host == domain || host == "www."+domain) {
		return nil
	}
	return errors.New("sitehost: no certificate for " + host + "; connect the domain in Settings first")
}

// TLSManager is the certificate manager for the HTTPS listener, or nil when
// the site runs without HTTPS.
func (h *Host) TLSManager() *autocert.Manager {
	if !h.opts.TLS {
		return nil
	}
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: h.HostPolicy,
		Email:      strings.TrimSpace(h.settings().Metadata["contactEmail"]),
	}
	if dir := h.opts.certDir(); dir != "" {
		manager.Cache = autocert.DirCache(dir)
	}
	return manager
}

// hostRedirect sends the twin address to the canonical one, so a visitor
// who types www.example.com and one who types example.com share one site,
// one set of links, and one search listing.
func (h *Host) hostRedirect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			if alternate := h.alternateHost(); alternate != "" && requestHost(r) == alternate {
				target := h.publicScheme() + "://" + h.canonicalHost() + r.URL.RequestURI()
				http.Redirect(w, r, target, http.StatusMovedPermanently)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requestHost is the Host header without port, folded to lower case.
func requestHost(r *http.Request) string {
	host := strings.ToLower(strings.TrimSpace(r.Host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(host, ".")
}

// ---------- the admin screen ----------

func (h *Host) mountDomain(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/domain", h.handleAdminDomain)
	mux.HandleFunc("GET /admin/domain/{$}", h.handleAdminDomain)
	mux.HandleFunc("POST /admin/domain", h.handleAdminSaveDomain)
	mux.HandleFunc("POST /admin/domain/check", h.handleAdminCheckDomain)
	mux.HandleFunc("POST /admin/domain/remove", h.handleAdminRemoveDomain)
}

// serverIP is the address DNS records should point at: the one given at
// start-up, else the one the owner is using to reach the admin right now.
func (h *Host) serverIP(r *http.Request) string {
	if ip := strings.TrimSpace(h.opts.PublicIP); ip != "" {
		return ip
	}
	host := requestHost(r)
	if ip := net.ParseIP(host); ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified() {
		return host
	}
	return ""
}

type dnsCheck struct {
	Host       string
	Addresses  []string
	Found      bool
	PointsHere bool
}

func (h *Host) checkDNS(host, expected string) dnsCheck {
	result := dnsCheck{Host: host}
	addresses, err := lookupHost(host)
	if err != nil || len(addresses) == 0 {
		return result
	}
	result.Found = true
	result.Addresses = addresses
	for _, address := range addresses {
		if expected != "" && address == expected {
			result.PointsHere = true
		}
	}
	return result
}

func (h *Host) handleAdminDomain(w http.ResponseWriter, r *http.Request) {
	h.renderAdminDomain(w, r, adminStatus{Message: r.URL.Query().Get("status")}, nil)
}

func (h *Host) renderAdminDomain(w http.ResponseWriter, r *http.Request, status adminStatus, checks []dnsCheck) {
	domain := h.Domain()
	ip := h.serverIP(r)
	sections := []gosx.Node{}

	// Step 1: the name.
	nameAttrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "www"), gosx.Attr("value", "true")}
	if h.prefersWWW() {
		nameAttrs = append(nameAttrs, gosx.Attr("checked", "checked"))
	}
	sections = append(sections, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("1. Your domain")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/domain")),
			h.csrfField(),
			adminTextField("domain", "Domain name", domain, "The name you bought, such as yourbusiness.com. You buy one from a registrar like Namecheap, Cloudflare, or Google Domains if you don't have one yet."),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")),
				gosx.El("input", gosx.Attrs(nameAttrs...)),
				gosx.Text(" Use www. in the address (www.yourbusiness.com). Either way, both addresses work."),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Save domain")),
			),
		),
	))

	if domain == "" {
		body := h.renderAdminShell("settings", "Your own domain",
			"Use an address like yourbusiness.com instead of the one your server came with. Three steps, about ten minutes, plus waiting for the internet to notice.",
			status, sections...)
		h.writeDocument(w, http.StatusOK, h.adminMeta("Your own domain"), body)
		return
	}

	// Step 2: the records.
	ipCell := ip
	if ipCell == "" {
		ipCell = "this server's public IP address"
	}
	wwwTarget := domain
	records := gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-table--records")),
		gosx.El("thead", nil, gosx.El("tr", nil,
			gosx.El("th", nil, gosx.Text("Type")), gosx.El("th", nil, gosx.Text("Name")), gosx.El("th", nil, gosx.Text("Value")))),
		gosx.El("tbody", nil,
			gosx.El("tr", nil, gosx.El("td", nil, gosx.Text("A")), gosx.El("td", nil, gosx.Text("@")), gosx.El("td", nil, gosx.El("code", nil, gosx.Text(ipCell)))),
			gosx.El("tr", nil, gosx.El("td", nil, gosx.Text("CNAME")), gosx.El("td", nil, gosx.Text("www")), gosx.El("td", nil, gosx.El("code", nil, gosx.Text(wwwTarget)))),
		),
	)
	ipHint := gosx.Fragment()
	if ip == "" {
		ipHint = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
			gosx.Text("We couldn't tell this server's public address from here. Your hosting provider shows it on the server's page; you can also start the site with -public-ip so it appears above."))
	}
	sections = append(sections, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("2. Point "+domain+" at this server")),
		gosx.El("p", nil, gosx.Text("Sign in where you bought the domain, find the DNS settings, and add these two records. \"@\" means the domain itself; some registrars show it as blank.")),
		records,
		ipHint,
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Changes can take up to an hour to spread across the internet, occasionally longer.")),
	))

	// Step 3: check.
	checkNodes := []gosx.Node{
		gosx.El("h2", nil, gosx.Text("3. Check it")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/domain/check")),
			h.csrfField(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Check DNS now")),
		),
	}
	if len(checks) > 0 {
		rows := make([]gosx.Node, 0, len(checks))
		for _, check := range checks {
			state, label := "draft", "Not found yet — give it time, then check again"
			switch {
			case check.PointsHere:
				state, label = "published", "Points here ✓"
			case check.Found && ip != "":
				state, label = "offline", "Points at "+strings.Join(check.Addresses, ", ")+", not at "+ip
			case check.Found:
				state, label = "published", "Points at "+strings.Join(check.Addresses, ", ")
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.El("code", nil, gosx.Text(check.Host))),
				gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label))),
			))
		}
		checkNodes = append(checkNodes, gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")), gosx.El("tbody", nil, gosx.Fragment(rows...))))
	}
	sections = append(sections, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")), gosx.Fragment(checkNodes...)))

	// Step 4: HTTPS.
	var https gosx.Node
	if h.opts.TLS {
		https = gosx.Fragment(
			gosx.El("p", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", "published")), gosx.Text("HTTPS is on"))),
			gosx.El("p", nil, gosx.Text("A certificate is issued automatically by Let's Encrypt the first time someone opens https://"+h.canonicalHost()+" after the records above are live, and renewed for you before it expires.")),
		)
	} else {
		dataPath := h.opts.DataPath
		if abs, err := filepath.Abs(dataPath); err == nil {
			dataPath = abs
		}
		https = gosx.Fragment(
			gosx.El("p", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", "offline")), gosx.Text("HTTPS is off"))),
			gosx.El("p", nil, gosx.Text("Start the site with HTTPS on and it takes care of the certificate itself. Stop it, then run:")),
			gosx.El("pre", gosx.Attrs(gosx.Attr("class", "admin-code")), gosx.Text("gosx-site -https -data "+dataPath+" -admin-password YOUR_PASSWORD")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("It listens on ports 80 and 443, the standard web ports. In a container, publish both: -p 80:80 -p 443:443. Nothing else changes; your site data stays where it is.")),
		)
	}
	sections = append(sections, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("4. Secure it")), https))

	sections = append(sections, gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Disconnect")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Stops using "+domain+". Your site keeps working at the address it had before.")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/domain/remove")),
			h.csrfField(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Disconnect this domain")),
		),
	))

	body := h.renderAdminShell("settings", "Your own domain",
		"Your site's address is "+h.publicScheme()+"://"+h.canonicalHost()+" once the records below are live.",
		status, sections...)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Your own domain"), body)
}

func (h *Host) handleAdminSaveDomain(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAdminDomain(w, r, adminStatus{Message: "We couldn't read that form. Try again.", Error: true}, nil)
		return
	}
	raw := strings.TrimSpace(r.PostFormValue("domain"))
	domain := normalizeDomain(raw)
	if domain == "" {
		message := "Type the domain name on its own, such as yourbusiness.com."
		if raw == "" {
			message = "Type your domain name first."
		}
		h.renderAdminDomain(w, r, adminStatus{Message: message, Error: true}, nil)
		return
	}
	www := r.PostFormValue("www") == "true"
	if err := h.updateSettingsMetadata(func(m cmsstore.Metadata) {
		m[domainKey] = domain
		if www {
			m[domainWWWKey] = "true"
		} else {
			delete(m, domainWWWKey)
		}
	}); err != nil {
		h.renderAdminDomain(w, r, adminStatus{Message: "We couldn't save that. Try again.", Error: true}, nil)
		return
	}
	// The site's public address follows the domain, so links, the sitemap,
	// and the feed all use it.
	if err := h.setBaseURL(h.publicScheme() + "://" + h.canonicalHost()); err != nil {
		h.renderAdminDomain(w, r, adminStatus{Message: "We couldn't save that. Try again.", Error: true}, nil)
		return
	}
	http.Redirect(w, r, "/admin/domain?status="+queryEscape("Saved. Now add the DNS records in step 2."), http.StatusSeeOther)
}

func (h *Host) setBaseURL(baseURL string) error {
	settings := h.settings()
	metadata := cmsstore.Metadata{}
	for key, value := range settings.Metadata {
		metadata[key] = value
	}
	_, err := h.store.SaveSiteSettings(cmsstore.SiteSettingsInput{
		Title: settings.Title, Description: settings.Description, BaseURL: baseURL,
		Locale: settings.Locale, State: settings.State, Metadata: metadata,
	})
	return err
}

func (h *Host) handleAdminCheckDomain(w http.ResponseWriter, r *http.Request) {
	domain := h.Domain()
	if domain == "" {
		http.Redirect(w, r, "/admin/domain", http.StatusSeeOther)
		return
	}
	ip := h.serverIP(r)
	checks := []dnsCheck{h.checkDNS(domain, ip), h.checkDNS("www."+domain, ip)}
	status := adminStatus{Message: "Checked just now."}
	if checks[0].PointsHere || (ip == "" && checks[0].Found) {
		status.Message = "Checked just now: " + domain + " reaches this server."
	}
	h.renderAdminDomain(w, r, status, checks)
}

func (h *Host) handleAdminRemoveDomain(w http.ResponseWriter, r *http.Request) {
	previous := h.publicScheme() + "://" + h.canonicalHost()
	if err := h.updateSettingsMetadata(func(m cmsstore.Metadata) {
		delete(m, domainKey)
		delete(m, domainWWWKey)
	}); err != nil {
		h.renderAdminDomain(w, r, adminStatus{Message: "We couldn't do that. Try again.", Error: true}, nil)
		return
	}
	if strings.TrimRight(h.settings().BaseURL, "/") == previous {
		_ = h.setBaseURL("")
	}
	http.Redirect(w, r, "/admin/domain?status="+queryEscape("Disconnected."), http.StatusSeeOther)
}

// renderDomainPanel is the short card on the Settings page.
func (h *Host) renderDomainPanel() gosx.Node {
	domain := h.Domain()
	if domain == "" {
		return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your own domain")),
			gosx.El("p", nil, gosx.Text("Not connected yet. Use an address like yourbusiness.com, with HTTPS taken care of for you.")),
			gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "/admin/domain")), gosx.Text("Connect a domain"))),
		)
	}
	state, label := "offline", "HTTPS off"
	if h.opts.TLS {
		state, label = "published", "HTTPS on"
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Your own domain")),
		gosx.El("p", nil, gosx.El("strong", nil, gosx.Text(h.canonicalHost())), gosx.Text(" "),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label))),
		gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/domain")), gosx.Text("Manage domain"))),
	)
}
