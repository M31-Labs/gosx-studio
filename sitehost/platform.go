package sitehost

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// platform.go is what a site does differently when a hosting platform runs
// it: the platform owns the address, the certificate, and the backups, so
// the site tells its owner that instead of walking them through DNS, and
// it lets the platform ask for status and a backup with a token nobody
// else has.

const (
	platformStatusPath = "/platform/status"
	platformExportPath = "/platform/export.zip"
)

// managed reports whether a platform runs this site.
func (h *Host) managed() bool { return strings.TrimSpace(h.opts.ManagedBy) != "" }

// applyManagedDomain records the domain the platform connected, so
// canonical links, the www redirect, and the sitemap use it. It runs at
// start, and again whenever the platform restarts the site with a change.
func (h *Host) applyManagedDomain() error {
	domain := normalizeDomain(h.opts.Domain)
	if domain == "" || !h.managed() {
		return nil
	}
	if h.Domain() == domain {
		return nil
	}
	if err := h.updateSettingsMetadata(func(m cmsstore.Metadata) { m[domainKey] = domain }); err != nil {
		return err
	}
	if strings.TrimSpace(h.settings().BaseURL) == "" || strings.HasPrefix(h.opts.BaseURL, "https://"+domain) {
		return h.setBaseURL("https://" + domain)
	}
	return nil
}

// renderManagedDomainPanel is the settings card when a platform owns the
// address.
func (h *Host) renderManagedDomainPanel() gosx.Node {
	host := h.opts.ManagedBy
	if domain := h.Domain(); domain != "" {
		return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your domain")),
			gosx.El("p", nil, gosx.El("strong", nil, gosx.Text(domain)), gosx.Text(" "),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", "published")), gosx.Text("HTTPS on"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Connected and kept secure by "+host+". To change it, ask them.")))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Your own domain")),
		gosx.El("p", nil, gosx.Text("Want yourbusiness.com instead of the address you started with? Tell "+host+" the name you bought; they connect it and HTTPS is taken care of. Your current address keeps working either way.")))
}

// renderManagedDomainPage answers /admin/domain for a managed site.
func (h *Host) renderManagedDomainPage(w http.ResponseWriter) {
	body := h.renderAdminShell("settings", "Your own domain",
		"On "+h.opts.ManagedBy+" your address, its certificate, and its backups are looked after for you.",
		adminStatus{}, h.renderManagedDomainPanel(),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/settings")), gosx.Text("Back to settings"))))
	h.writeDocument(w, http.StatusOK, h.adminMeta("Your own domain"), body)
}

// ---------- the platform's door ----------

// platformStatus is what the platform reads about a site.
type platformStatus struct {
	OK            bool   `json:"ok"`
	Title         string `json:"title"`
	SetupComplete bool   `json:"setupComplete"`
	Pages         int    `json:"pages"`
	Posts         int    `json:"posts"`
	Products      int    `json:"products"`
	Users         int    `json:"users"`
	Domain        string `json:"domain,omitempty"`
	LastBackup    string `json:"lastBackup,omitempty"`
	Mail          string `json:"mail,omitempty"`
	Features      string `json:"features,omitempty"`
}

func (h *Host) mountPlatform(mux *http.ServeMux) {
	mux.HandleFunc("GET "+platformStatusPath, h.handlePlatformStatus)
	mux.HandleFunc("GET "+platformExportPath, h.handlePlatformExport)
}

// platformCaller checks the bearer token. No token configured means no
// door at all.
func (h *Host) platformCaller(w http.ResponseWriter, r *http.Request) bool {
	token := strings.TrimSpace(h.opts.OperatorToken)
	offered := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer"))
	if token == "" || offered == "" || len(token) < 16 || subtle.ConstantTimeCompare([]byte(token), []byte(offered)) != 1 {
		if h.authFailures.blocked(remoteHost(r), timeNow()) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			return false
		}
		h.authFailures.record(remoteHost(r), timeNow())
		http.NotFound(w, r)
		return false
	}
	return true
}

func (h *Host) handlePlatformStatus(w http.ResponseWriter, r *http.Request) {
	if !h.platformCaller(w, r) {
		return
	}
	settings := h.settings()
	status := platformStatus{OK: true, Title: firstNonEmpty(settings.Title, h.opts.SiteTitle), SetupComplete: h.SetupComplete(), Domain: h.Domain(), Features: strings.Join(h.opts.Features, ",")}
	if pages, err := h.store.ListPages(cmsstore.PageFilter{}); err == nil {
		status.Pages = len(pages)
	}
	if posts, err := h.store.ListPosts(cmsstore.PostFilter{}); err == nil {
		status.Posts = len(posts)
	}
	status.Products = len(h.products.list())
	status.Users = h.users.count()
	if h.mailer != nil {
		status.Mail = h.mailer.Describe()
	}
	if backups := h.listBackups(); len(backups) > 0 {
		status.LastBackup = backups[0].Name
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(status)
}

func (h *Host) handlePlatformExport(w http.ResponseWriter, r *http.Request) {
	if !h.platformCaller(w, r) {
		return
	}
	h.handleAdminExport(w, r)
}
