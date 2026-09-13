package sitehost

import (
	"net/http"
	"net/url"
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// staging.go gives the site a second address that shows everything as it
// would look if every unpublished change went live right now.
//
// The staging address (say staging.example.com) is the same site, read from
// the same store, with one difference: pages and posts render from their
// working records instead of their published snapshots. Nothing is copied.
// A key in the address, kept in a cookie afterwards, keeps strangers out,
// and search engines are told to stay away. "Publish everything waiting"
// then makes the live site match.

const (
	stagingHostKey   = "stagingHost"
	stagingKeyKey    = "stagingKey"
	stagingCookie    = "gosx_staging"
	stagingKeyParam  = "staging_key"
	stagingAdminPath = "/admin/staging"
)

func (h *Host) stagingHost() string { return normalizeDomain(h.settings().Metadata[stagingHostKey]) }

func (h *Host) stagingKey() string { return strings.TrimSpace(h.settings().Metadata[stagingKeyKey]) }

// stagingURL is the address the owner shares, key included.
func (h *Host) stagingURL() string {
	host := h.stagingHost()
	if host == "" {
		return ""
	}
	// The staging address answers on the same scheme and port as the site.
	scheme := h.publicScheme()
	if parsed, err := url.Parse(h.primaryOrigin()); err == nil && parsed.Host != "" {
		if parsed.Scheme == "https" {
			scheme = "https"
		}
		if port := parsed.Port(); port != "" {
			host += ":" + port
		}
	}
	return scheme + "://" + host + "/?" + stagingKeyParam + "=" + h.stagingKey()
}

// draftView is this host as staging sees it: the same stores, and the
// switch that makes livePage and livePost answer with the working record.
// Only public routes are ever served through it.
func (h *Host) draftView() *Host {
	view := &Host{
		store: h.store, opts: h.opts, messages: h.messages, authFailures: h.authFailures,
		mailer: h.mailer, media: h.media, stats: h.stats, forms: h.forms, products: h.products,
		orders: h.orders, bookings: h.bookings, carts: h.carts, users: h.users, auditLog: h.auditLog,
		metrics: h.metrics, collab: h.collab, draft: true,
	}
	// One install, one secret: the view must verify what the site signed.
	view.secret = h.installSecret()
	view.secretOnce.Do(func() {})
	return view
}

// isStagingRequest reports whether this request came in on the staging
// address.
func (h *Host) isStagingRequest(r *http.Request) bool {
	host := h.stagingHost()
	return host != "" && requestHost(r) == host
}

// stagingContent says whether a request on the staging address should be
// answered with drafts. Anything that writes, pays, or administers goes to
// the real site instead.
func stagingContent(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	path := r.URL.Path
	for _, prefix := range []string{adminPathPrefix, "/setup", "/preview", "/stripe", "/platform", checkoutPath, downloadPrefix, "/book", customerPath, cartPath, formSendPrefix} {
		if path == prefix || strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")+"/") {
			return false
		}
	}
	return true
}

// stagingGate routes staging-address requests to the draft view once the
// key has been shown, and everything else to the site as usual.
func (h *Host) stagingGate(next, drafts http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.isStagingRequest(r) {
			next.ServeHTTP(w, r)
			return
		}
		if !stagingContent(r) {
			if primary := h.primaryOrigin(); primary != "" {
				http.Redirect(w, r, primary+r.URL.RequestURI(), http.StatusTemporaryRedirect)
				return
			}
			next.ServeHTTP(w, r)
			return
		}
		key := h.stagingKey()
		if offered := r.URL.Query().Get(stagingKeyParam); offered != "" {
			if key != "" && secureEqual(offered, key) {
				http.SetCookie(w, &http.Cookie{Name: stagingCookie, Value: h.sign("staging:" + key), Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
					Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"), MaxAge: 30 * 24 * 3600})
				query := r.URL.Query()
				query.Del(stagingKeyParam)
				r.URL.RawQuery = query.Encode()
				http.Redirect(w, r, r.URL.RequestURI(), http.StatusSeeOther)
				return
			}
		}
		if cookie, err := r.Cookie(stagingCookie); err == nil && key != "" && secureEqual(cookie.Value, h.sign("staging:"+key)) {
			w.Header().Set("Cache-Control", "no-store")
			drafts.ServeHTTP(w, r)
			return
		}
		h.renderStagingGate(w)
	})
}

// primaryHost is the host name of the real site, or "" on a laptop.
func (h *Host) primaryHost() string {
	origin := h.primaryOrigin()
	if origin == "" {
		return ""
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return normalizeDomain(parsed.Hostname())
}

// primaryOrigin is where the real site lives, or "" on a laptop.
func (h *Host) primaryOrigin() string {
	if base := strings.TrimRight(strings.TrimSpace(h.settings().BaseURL), "/"); base != "" {
		return base
	}
	if h.Domain() != "" {
		return h.publicScheme() + "://" + h.canonicalHost()
	}
	return ""
}

func (h *Host) renderStagingGate(w http.ResponseWriter) {
	settings := h.settings()
	meta := metaFromSettings(settings)
	meta.Title = "Staging"
	meta.NoIndex = true
	body := gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-article")),
			gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text("This is the staging copy of "+firstNonEmpty(settings.Title, h.opts.SiteTitle))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("It shows changes that aren't published yet, so it's only for the team. Open the link the site owner shared with you — it carries the key.")),
			gosx.El("form", gosx.Attrs(gosx.Attr("class", "site-form"), gosx.Attr("method", "get"), gosx.Attr("action", "/")),
				formField(stagingKeyParam, "Staging key", "text", "", 80),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("type", "submit")), gosx.Text("Open staging")))))
	h.writeDocument(w, http.StatusForbidden, meta, body)
}

// stagingBanner tops every draft page so nobody mistakes it for the site.
func (h *Host) stagingBanner() gosx.Node {
	live := gosx.Fragment()
	if primary := h.primaryOrigin(); primary != "" {
		live = gosx.El("a", gosx.Attrs(gosx.Attr("href", primary)), gosx.Text("See the live site"))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-preview-banner"), gosx.Attr("role", "status")),
		gosx.El("span", nil, gosx.Text("Staging — showing unpublished changes. Visitors don't see this.")), live)
}

// ---------- what is waiting ----------

// waitingItem is a page or post whose working record differs from what
// visitors see.
type waitingItem struct {
	Kind, ID, Title, Slug, Editor, State string
	Page                                 cmsstore.Page
	Post                                 cmsstore.Post
	Checks                               []string
}

// waiting lists every unpublished change, pages first. Offline and archived
// records are the owner's deliberate choice and stay out.
func (h *Host) waiting() []waitingItem {
	items := []waitingItem{}
	if pages, err := h.store.ListPages(cmsstore.PageFilter{}); err == nil {
		for _, page := range h.orderPages(pages) {
			if PageOffline(page) || PageArchived(page) || page.State.Publish == cmsstore.PublishStatePublished || page.Metadata[publishPendingKey] == "true" {
				continue
			}
			state := "Changed"
			if !h.isLive(page) {
				state = "Never published"
			}
			items = append(items, waitingItem{Kind: "page", ID: page.ID, Title: page.Title, Slug: page.Slug, Editor: "/admin/edit/" + page.ID, State: state, Page: page, Checks: h.readinessChecks("page", page.Description, page.Body)})
		}
	}
	if posts, err := h.store.ListPosts(cmsstore.PostFilter{}); err == nil {
		sortPostsNewestFirst(posts)
		for _, post := range posts {
			if PostOffline(post) || PostArchived(post) || post.State.Publish == cmsstore.PublishStatePublished || post.Metadata[publishPendingKey] == "true" {
				continue
			}
			state := "Changed"
			if _, live := h.livePost(post); !live {
				state = "Never published"
			}
			items = append(items, waitingItem{Kind: "post", ID: post.ID, Title: post.Title, Slug: post.Slug, Editor: "/admin/edit/post/" + post.ID, State: state, Post: post, Checks: h.readinessChecks("post", "", post.Body)})
		}
	}
	return items
}

// publishWaiting publishes every waiting item and reports how many went
// live and what was skipped. Records with a future publish date keep it.
func (h *Host) publishWaiting(r *http.Request) (published int, failed []string) {
	for _, item := range h.waiting() {
		var result editorSaveResult
		if item.Kind == "page" {
			result = h.publishPage(item.Page)
		} else {
			result = h.publishPost(item.Post)
		}
		if result.OK {
			published++
			h.auditContent(r, "publish", "Published “"+item.Title+"” from staging")
		} else {
			failed = append(failed, item.Title)
		}
	}
	return published, failed
}

// ---------- the admin screen ----------

func (h *Host) mountStaging(mux *http.ServeMux) {
	mux.HandleFunc("GET "+stagingAdminPath, h.handleAdminStaging)
	mux.HandleFunc("GET "+stagingAdminPath+"/{$}", h.handleAdminStaging)
	mux.HandleFunc("POST "+stagingAdminPath, h.handleAdminSaveStaging)
	mux.HandleFunc("POST "+stagingAdminPath+"/publish", h.handleAdminPublishWaiting)
	mux.HandleFunc("POST "+stagingAdminPath+"/publish/{kind}/{id}", h.handleAdminPublishOne)
	mux.HandleFunc("POST "+stagingAdminPath+"/key", h.handleAdminNewStagingKey)
	mux.HandleFunc("POST "+stagingAdminPath+"/remove", h.handleAdminRemoveStaging)
}

func (h *Host) handleAdminStaging(w http.ResponseWriter, r *http.Request) {
	items := h.waiting()
	status := adminStatus{Message: r.URL.Query().Get("status")}
	if r.URL.Query().Get("error") != "" {
		status = adminStatus{Message: r.URL.Query().Get("error"), Error: true}
	}

	var queue gosx.Node
	if len(items) == 0 {
		queue = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("Nothing waiting")),
			gosx.El("p", nil, gosx.Text("Every page and post is published. Changes you save without publishing will show up here.")))
	} else {
		rows := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			checks := gosx.Fragment()
			if len(item.Checks) > 0 {
				checks = gosx.Fragment(gosx.El("br", nil), gosx.El("small", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(strings.Join(item.Checks, " · "))))
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", item.Editor)), gosx.Text(item.Title)), gosx.Text(" "),
					gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge")), gosx.Text(item.Kind)), checks),
				gosx.El("td", nil, gosx.Text(item.State)),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
					gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", stagingAdminPath+"/publish/"+item.Kind+"/"+item.ID), gosx.Attr("class", "admin-inline-form")),
						h.csrfField(),
						gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", "publish")), gosx.Text("Publish")))),
			))
		}
		queue = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Waiting to go live ("+itoa(len(items))+")")),
			gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", stagingAdminPath+"/publish")),
				h.csrfField(),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit"), gosx.Attr("data-action", "publish-all")), gosx.Text("Publish everything waiting")),
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Each one is published the same way as the Publish button in the editor. Anything with a future date stays scheduled."))),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("What")), gosx.El("th", nil, gosx.Text("State")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...))))
	}

	var address gosx.Node
	if host := h.stagingHost(); host == "" {
		address = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("A staging address")),
			gosx.El("p", nil, gosx.Text("A second address, such as staging.yourbusiness.com, shows the site with every unpublished change in place — the whole thing, menus and all — so you can walk through it before visitors do. Only people with the key can open it.")),
			gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", stagingAdminPath), gosx.Attr("class", "admin-form")),
				h.csrfField(),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
					gosx.El("label", gosx.Attrs(gosx.Attr("for", "stagingHost")), gosx.Text("Staging address")),
					gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("id", "stagingHost"), gosx.Attr("name", "stagingHost"), gosx.Attr("placeholder", "staging.yourbusiness.com"), gosx.Attr("autocomplete", "off"))),
					gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Point it at this server the same way as your domain: an A record with the same address. HTTPS is taken care of."))),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Turn on staging"))))
	} else {
		link := h.stagingURL()
		address = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your staging address")),
			gosx.El("p", nil, gosx.El("strong", nil, gosx.Text(host)), gosx.Text(" shows the site with every unpublished change in place. Share this link with the team; it carries the key.")),
			gosx.El("p", nil, gosx.El("code", gosx.Attrs(gosx.Attr("class", "admin-code"), gosx.Attr("id", "staging-link")), gosx.Text(link))),
			gosx.El("p", nil,
				gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", link), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("Open staging")),
				gosx.Text(" "),
				gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", stagingAdminPath+"/key"), gosx.Attr("class", "admin-inline-form")),
					h.csrfField(),
					gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("New key"))),
				gosx.Text(" "),
				gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", stagingAdminPath+"/remove"), gosx.Attr("class", "admin-inline-form")),
					h.csrfField(),
					gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Turn off staging")))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("A new key makes every old link stop working. Search engines are told to ignore the staging address.")))
	}

	body := h.renderAdminShell("staging", "Staging", "See every unpublished change together, then put them all live at once.", status, queue, address)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Staging"), body)
}

func (h *Host) handleAdminSaveStaging(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	host := normalizeDomain(r.PostFormValue("stagingHost"))
	if host == "" || host == h.Domain() || host == "www."+h.Domain() || host == h.primaryHost() {
		http.Redirect(w, r, stagingAdminPath+"?error="+queryEscape("Use a name like staging.yourbusiness.com — different from your main address."), http.StatusSeeOther)
		return
	}
	if err := h.updateSettingsMetadata(func(m cmsstore.Metadata) {
		m[stagingHostKey] = host
		if strings.TrimSpace(m[stagingKeyKey]) == "" {
			m[stagingKeyKey] = randomHex(12)
		}
	}); err != nil {
		http.Redirect(w, r, stagingAdminPath+"?error="+queryEscape("We couldn't save that. Try again."), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "staging.enabled", "Turned on staging at "+host)
	http.Redirect(w, r, stagingAdminPath+"?status="+queryEscape("Staging is on at "+host+"."), http.StatusSeeOther)
}

func (h *Host) handleAdminNewStagingKey(w http.ResponseWriter, r *http.Request) {
	if err := h.updateSettingsMetadata(func(m cmsstore.Metadata) { m[stagingKeyKey] = randomHex(12) }); err != nil {
		http.Redirect(w, r, stagingAdminPath+"?error="+queryEscape("We couldn't do that. Try again."), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "staging.key", "Made a new staging key")
	http.Redirect(w, r, stagingAdminPath+"?status="+queryEscape("New key made. Old staging links no longer work."), http.StatusSeeOther)
}

func (h *Host) handleAdminRemoveStaging(w http.ResponseWriter, r *http.Request) {
	if err := h.updateSettingsMetadata(func(m cmsstore.Metadata) { delete(m, stagingHostKey); delete(m, stagingKeyKey) }); err != nil {
		http.Redirect(w, r, stagingAdminPath+"?error="+queryEscape("We couldn't do that. Try again."), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "staging.disabled", "Turned off staging")
	http.Redirect(w, r, stagingAdminPath+"?status="+queryEscape("Staging is off."), http.StatusSeeOther)
}

func (h *Host) handleAdminPublishWaiting(w http.ResponseWriter, r *http.Request) {
	published, failed := h.publishWaiting(r)
	message := "Published " + plural(published, "change") + ". The live site is up to date."
	if published == 0 {
		message = "Nothing was waiting."
	}
	if len(failed) > 0 {
		http.Redirect(w, r, stagingAdminPath+"?error="+queryEscape("Published "+itoa(published)+", but not: "+strings.Join(failed, ", ")+". Open each one to see why."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, stagingAdminPath+"?status="+queryEscape(message), http.StatusSeeOther)
}

func (h *Host) handleAdminPublishOne(w http.ResponseWriter, r *http.Request) {
	kind, id := r.PathValue("kind"), r.PathValue("id")
	var result editorSaveResult
	var title string
	switch kind {
	case "page":
		page, ok, _ := h.store.PageByID(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		title, result = page.Title, h.publishPage(page)
	case "post":
		post, ok, _ := h.store.PostByID(id)
		if !ok {
			http.NotFound(w, r)
			return
		}
		title, result = post.Title, h.publishPost(post)
	default:
		http.NotFound(w, r)
		return
	}
	if !result.OK {
		http.Redirect(w, r, stagingAdminPath+"?error="+queryEscape(result.Message), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "publish", "Published “"+title+"” from staging")
	http.Redirect(w, r, stagingAdminPath+"?status="+queryEscape(result.Message), http.StatusSeeOther)
}
