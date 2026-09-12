package sitehost

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/content"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// admin.go renders the back office the default host serves.
//
// Every list here ships an empty state with a call to action. Studio's own
// generic index renderer gates empty states behind host-supplied copy
// (backoffice/backend_resource_index.go:99), which is why an unconfigured
// install shows a headers-only table. The default host supplies that copy.

type adminStatus struct {
	Message string
	Error   bool
}

func (h *Host) mountAdmin(mux *http.ServeMux) {
	// Patterns are registered without a trailing slash because every link the
	// host renders is slash-free; the "{$}" variants keep a typed trailing
	// slash working instead of 404ing on the operator.
	mux.HandleFunc("GET /admin", h.handleAdminDashboard)
	mux.HandleFunc("GET /admin/{$}", h.handleAdminDashboard)
	mux.HandleFunc("GET /admin/pages", h.handleAdminPages)
	mux.HandleFunc("GET /admin/pages/{$}", h.handleAdminPages)
	mux.HandleFunc("POST /admin/pages", h.handleAdminCreatePage)
	mux.HandleFunc("POST /admin/pages/{$}", h.handleAdminCreatePage)
	mux.HandleFunc("GET /admin/pages/{id}", h.handleAdminPageDetail)
	mux.HandleFunc("POST /admin/pages/{id}", h.handleAdminSavePage)
	mux.HandleFunc("POST /admin/pages/{id}/publish", h.handleAdminPublishPage)
	mux.HandleFunc("POST /admin/pages/{id}/action", h.handleAdminPageAction)
	mux.HandleFunc("GET /admin/settings", h.handleAdminSettings)
	mux.HandleFunc("GET /admin/settings/{$}", h.handleAdminSettings)
	mux.HandleFunc("POST /admin/settings", h.handleAdminSaveSettings)
	mux.HandleFunc("POST /admin/settings/{$}", h.handleAdminSaveSettings)
	mux.HandleFunc("POST /admin/settings/test-mail", h.handleAdminTestMail)
}

func (h *Host) adminMeta(title string) PageMeta {
	settings := h.settings()
	meta := metaFromSettings(settings)
	meta.Title = title
	meta.Description = "Manage " + firstNonEmpty(settings.Title, h.opts.SiteTitle) + "."
	meta.AdminChrome = true
	meta.NoIndex = true
	return meta
}

func (h *Host) renderAdminShell(active, heading, lede string, status adminStatus, sections ...gosx.Node) gosx.Node {
	messagesLabel := "Messages"
	if unread := h.unreadMessages(); unread > 0 {
		messagesLabel = "Messages (" + itoa(unread) + ")"
	}
	navItems := []struct{ Key, Label, Href string }{
		{"dashboard", "Dashboard", "/admin"},
		{"pages", "Pages", "/admin/pages"},
		{"posts", "Blog", "/admin/posts"},
		{"messages", messagesLabel, "/admin/messages"},
		{"stats", "Visitors", "/admin/stats"},
		{"media", "Pictures", "/admin/media"},
		{"settings", "Settings", "/admin/settings"},
	}
	links := make([]gosx.Node, 0, len(navItems)+1)
	for _, item := range navItems {
		attrs := []any{gosx.Attr("href", item.Href)}
		if item.Key == active {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(item.Label)))
	}
	links = append(links, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/"), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("View site")))

	body := []gosx.Node{
		gosx.El("h1", nil, gosx.Text(heading)),
	}
	if strings.TrimSpace(lede) != "" {
		body = append(body, gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-lede")), gosx.Text(lede)))
	}
	if strings.TrimSpace(status.Message) != "" {
		state := "ok"
		if status.Error {
			state = "error"
		}
		body = append(body, gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "admin-status"),
			gosx.Attr("data-state", state),
			gosx.Attr("role", "status"),
		), gosx.Text(status.Message)))
	}
	body = append(body, sections...)

	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-shell")),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "admin-header")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-brand"), gosx.Attr("href", "/admin")),
				gosx.Text(firstNonEmpty(h.settings().Title, h.opts.SiteTitle))),
			gosx.El("nav", gosx.Attrs(gosx.Attr("class", "admin-nav"), gosx.Attr("aria-label", "Admin")),
				gosx.Fragment(links...)),
		),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "admin-main"), gosx.Attr("id", "main")),
			gosx.Fragment(body...)),
	)
}

// ---------- dashboard ----------

func (h *Host) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	pages, _ := h.store.ListPages(cmsstore.PageFilter{})
	published, drafts := 0, 0
	for _, page := range pages {
		if page.State.Publish == cmsstore.PublishStatePublished {
			published++
		} else {
			drafts++
		}
	}

	stats := gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-stats")),
		adminStat(published, "Live pages"),
		adminStat(drafts, "Unpublished pages"),
		adminStat(len(h.livePosts()), "Live posts"),
		adminStat(h.unreadMessages(), "New messages"),
		adminStat(h.stats.summary(timeNow(), 7).WeekVisitors, "Visitors this week"),
	)

	next := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("What to do next")),
		gosx.El("ol", nil,
			gosx.El("li", nil, gosx.Text("Edit your home page text, then publish it.")),
			gosx.El("li", nil, gosx.Text("Add your site name and description in Settings so search results and shared links read well.")),
			gosx.El("li", nil, gosx.Text("Add the pages your visitors need, such as services, pricing, or opening hours.")),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", h.homeEditHref())), gosx.Text("Edit your home page")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", h.homeEditHref()+"#look")), gosx.Text("Change the look")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/pages")), gosx.Text("All pages")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/posts")), gosx.Text("Write a post")),
		),
	)

	body := h.renderAdminShell("dashboard", "Your site",
		"Everything you publish here appears on your public website.",
		adminStatus{}, stats, next)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Your site"), body)
}

func adminStat(value int, label string) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-stat")),
		gosx.El("strong", nil, gosx.Text(itoa(value))),
		gosx.El("span", nil, gosx.Text(label)),
	)
}

// ---------- pages ----------

func (h *Host) sortedPages() []cmsstore.Page {
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return nil
	}
	return h.orderPages(pages)
}

func (h *Host) handleAdminPages(w http.ResponseWriter, r *http.Request) {
	h.renderAdminPages(w, r, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderAdminPages(w http.ResponseWriter, r *http.Request, status adminStatus) {
	pages := h.sortedPages()
	active := make([]cmsstore.Page, 0, len(pages))
	archived := make([]cmsstore.Page, 0, 2)
	for _, page := range pages {
		if PageArchived(page) {
			archived = append(archived, page)
		} else {
			active = append(active, page)
		}
	}

	var listing gosx.Node
	if len(active) == 0 {
		// The empty state Studio's generic index renderer never ships.
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No pages yet")),
			gosx.El("p", nil, gosx.Text("Pages are what visitors read on your website — a home page, an about page, a contact page. Create your first one below.")),
		)
	} else {
		rows := make([]gosx.Node, 0, len(active))
		for index, page := range active {
			rows = append(rows, h.renderPageRow(page, index == 0, index == len(active)-1))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your pages")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
				gosx.Text("The order here is the order of your site menu. Your home page always comes first.")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-table--pages")),
				gosx.El("thead", nil, gosx.El("tr", nil,
					gosx.El("th", nil, gosx.Text("Page")),
					gosx.El("th", nil, gosx.Text("Address")),
					gosx.El("th", nil, gosx.Text("Status")),
					gosx.El("th", nil, gosx.Text("")),
				)),
				gosx.El("tbody", nil, gosx.Fragment(rows...)),
			),
		)
	}

	var archivedPanel gosx.Node = gosx.Fragment()
	if len(archived) > 0 {
		rows := make([]gosx.Node, 0, len(archived))
		for _, page := range archived {
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.Text(page.Title)),
				gosx.El("td", nil, gosx.Text(publicPath(page.Slug))),
				gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", "archived")), gosx.Text("Archived"))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")), h.pageActionButton(page.ID, "restore", "Restore")),
			))
		}
		archivedPanel = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Archived")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Not on your site and not in the menu, but nothing is deleted.")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-table--pages")),
				gosx.El("tbody", nil, gosx.Fragment(rows...))),
		)
	}

	create := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Add a page")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/pages")),
			h.csrfField(),
			adminTextField("title", "Page name", "", "Shown as the heading and in your site menu."),
			adminTextField("slug", "Web address", "", "Letters and dashes only. \"about-us\" becomes yoursite.com/about-us."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Create page")),
			),
		),
	)

	body := h.renderAdminShell("pages", "Pages",
		"Create, edit, and publish the pages on your website.",
		status, listing, create, archivedPanel)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Pages"), body)
}

func (h *Host) handleAdminCreatePage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAdminPages(w, r, adminStatus{Message: "We couldn't read that form. Try again.", Error: true})
		return
	}
	title := strings.TrimSpace(r.PostFormValue("title"))
	slug := normalizeSlug(firstNonEmpty(r.PostFormValue("slug"), title))
	if title == "" {
		h.renderAdminPages(w, r, adminStatus{Message: "Give the page a name before creating it.", Error: true})
		return
	}
	if slug == "" {
		h.renderAdminPages(w, r, adminStatus{Message: "Give the page a web address, such as \"about-us\".", Error: true})
		return
	}
	if message := reservedSlugMessage(slug); message != "" {
		h.renderAdminPages(w, r, adminStatus{Message: message, Error: true})
		return
	}
	if _, exists, _ := h.store.PageBySlug(slug); exists {
		h.renderAdminPages(w, r, adminStatus{
			Message: "Another page already uses the address /" + slug + ". Pick a different one.",
			Error:   true,
		})
		return
	}

	page, err := h.store.CreatePage(cmsstore.PageInput{
		Slug:  slug,
		Title: title,
		Body:  document(block(0, content.BlockParagraph, values("text", "Write your page here."))),
	})
	if err != nil {
		h.renderAdminPages(w, r, adminStatus{Message: "We couldn't create that page. Try again.", Error: true})
		return
	}
	http.Redirect(w, r, "/admin/edit/"+page.ID, http.StatusSeeOther)
}

// handleAdminPageDetail keeps the old form URL working by sending it to the
// editor. One page, one place to edit it.
func (h *Host) handleAdminPageDetail(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		h.writeAdminNotFound(w, "page")
		return
	}
	http.Redirect(w, r, "/admin/edit/"+page.ID, http.StatusSeeOther)
}

func (h *Host) renderAdminPageDetail(w http.ResponseWriter, page cmsstore.Page, status adminStatus) {
	live := page.State.Publish == cmsstore.PublishStatePublished
	stateLine := "This page is not on your website yet."
	if live {
		stateLine = "This page is live at " + publicPath(page.Slug) + "."
	}

	form := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Page content")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/pages/"+page.ID)),
			h.csrfField(),
			adminTextField("title", "Page name", page.Title, ""),
			adminTextField("slug", "Web address", page.Slug, "Changing this changes the page's link. Old links will stop working."),
			adminTextareaField("body", "Page text", documentToText(page.Body),
				"One block per line. Start a line with \"# \" for a heading or \"> \" for a quote."),
			adminTextField("metaTitle", "Title shown in Google", page.Metadata["metaTitle"],
				"Leave blank to use the page name."),
			adminTextField("metaDescription", "Description shown in Google", page.Metadata["metaDescription"],
				"One or two sentences. This also appears when someone shares the link."),
			adminTextField("metaImageUrl", "Image shown when shared", page.Metadata["metaImageUrl"],
				"A link to an image. Used by social apps when someone posts your page."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Save")),
				gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/pages")), gosx.Text("Back to pages")),
			),
		),
	)

	publishLabel := "Publish this page"
	if live {
		publishLabel = "Publish your changes"
	}
	publish := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Publishing")),
		gosx.El("p", nil, gosx.Text(stateLine)),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/pages/"+page.ID+"/publish")),
			h.csrfField(),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text(publishLabel)),
				viewLink(live, page.Slug),
			),
		),
	)

	body := h.renderAdminShell("pages", page.Title, stateLine, status, form, publish)
	h.writeDocument(w, http.StatusOK, h.adminMeta(page.Title), body)
}

func viewLink(live bool, slug string) gosx.Node {
	if !live {
		return gosx.Fragment()
	}
	return gosx.El("a", gosx.Attrs(
		gosx.Attr("class", "admin-secondary"),
		gosx.Attr("href", publicPath(slug)),
		gosx.Attr("target", "_blank"),
		gosx.Attr("rel", "noopener"),
	), gosx.Text("View this page"))
}

func (h *Host) handleAdminSavePage(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		h.writeAdminNotFound(w, "page")
		return
	}
	if err := r.ParseForm(); err != nil {
		h.renderAdminPageDetail(w, page, adminStatus{Message: "We couldn't read that form. Try again.", Error: true})
		return
	}

	title := strings.TrimSpace(r.PostFormValue("title"))
	if title == "" {
		h.renderAdminPageDetail(w, page, adminStatus{Message: "Give the page a name before saving.", Error: true})
		return
	}
	slug := normalizeSlug(firstNonEmpty(r.PostFormValue("slug"), title))
	if other, exists, _ := h.store.PageBySlug(slug); exists && other.ID != page.ID {
		h.renderAdminPageDetail(w, page, adminStatus{
			Message: "Another page already uses the address /" + slug + ". Pick a different one.",
			Error:   true,
		})
		return
	}

	metadata := cmsstore.Metadata{}
	for key, value := range page.Metadata {
		metadata[key] = value
	}
	for _, key := range []string{"metaTitle", "metaDescription", "metaImageUrl"} {
		if value := strings.TrimSpace(r.PostFormValue(key)); value != "" {
			metadata[key] = value
		} else {
			delete(metadata, key)
		}
	}

	input := cmsstore.PageInput{
		Slug:        slug,
		Title:       title,
		Description: strings.TrimSpace(r.PostFormValue("metaDescription")),
		Body:        textToDocument(r.PostFormValue("body")),
		Metadata:    metadata,
		State:       page.State,
	}
	if _, _, err := h.store.PreviewPage(page.ID, input); err != nil {
		h.renderAdminPageDetail(w, page, adminStatus{Message: "We couldn't save that page. Try again.", Error: true})
		return
	}
	if slug != page.Slug && page.Slug != homeSlug {
		_ = h.recordRedirect(publicPath(page.Slug), publicPath(slug))
	}
	http.Redirect(w, r, "/admin/pages/"+page.ID+"?status="+queryEscape("Saved. Publish when you're ready for visitors to see it."), http.StatusSeeOther)
}

func (h *Host) handleAdminPublishPage(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		h.writeAdminNotFound(w, "page")
		return
	}
	if _, _, err := h.store.PublishPage(page.ID); err != nil {
		h.renderAdminPageDetail(w, page, adminStatus{Message: "We couldn't publish that page. Try again.", Error: true})
		return
	}
	http.Redirect(w, r, "/admin/pages/"+page.ID+"?status="+queryEscape("Published. Your page is live at "+publicPath(page.Slug)+"."), http.StatusSeeOther)
}

// ---------- settings ----------

func (h *Host) handleAdminSettings(w http.ResponseWriter, r *http.Request) {
	h.renderAdminSettings(w, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderAdminSettings(w http.ResponseWriter, status adminStatus) {
	settings := h.settings()
	form := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Site details")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/settings"), gosx.Attr("enctype", "multipart/form-data")),
			h.csrfField(),
			adminTextField("title", "Site name", settings.Title, "Shown in the browser tab, your site menu, and search results."),
			adminTextField("description", "Site description", settings.Description,
				"One or two sentences about your business. Search engines show this under your site name."),
			adminTextField("baseURL", "Website address", settings.BaseURL,
				"For example https://yourbusiness.com. Needed so shared links and search results point at the right place."),
			h.renderBrandFields(settings),
			h.renderMailFields(settings),
			h.renderGrowthFields(settings),
			renderStatsField(settings),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Save settings")),
			),
		),
	)
	body := h.renderAdminShell("settings", "Settings",
		"These details appear in search results and when someone shares a link to your site.",
		status, form, h.renderMailTestPanel())
	h.writeDocument(w, http.StatusOK, h.adminMeta("Settings"), body)
}

func (h *Host) handleAdminSaveSettings(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		if err := r.ParseForm(); err != nil {
			h.renderAdminSettings(w, adminStatus{Message: "We couldn't read that form. Try again.", Error: true})
			return
		}
	}
	title := strings.TrimSpace(r.PostFormValue("title"))
	if title == "" {
		h.renderAdminSettings(w, adminStatus{Message: "Give your site a name before saving.", Error: true})
		return
	}
	// Carry existing metadata forward. It holds the setup marker and the
	// contact details the wizard collected; dropping it would send a
	// configured site back to the "not set up yet" screen.
	metadata := cmsstore.Metadata{}
	if current, ok, err := h.store.SiteSettings(); err == nil && ok {
		for key, value := range current.Metadata {
			metadata[key] = value
		}
	}
	if problem := h.applyBrandFields(r, metadata); problem != "" {
		h.renderAdminSettings(w, adminStatus{Message: problem, Error: true})
		return
	}
	applyGrowthFields(r, metadata)
	applyStatsField(r, metadata)
	if address := strings.TrimSpace(r.PostFormValue("notifyEmail")); address != "" {
		metadata[notifyEmailKey] = address
	} else {
		delete(metadata, notifyEmailKey)
	}
	input := cmsstore.SiteSettingsInput{
		Title:       title,
		Description: strings.TrimSpace(r.PostFormValue("description")),
		BaseURL:     strings.TrimRight(strings.TrimSpace(r.PostFormValue("baseURL")), "/"),
		Locale:      "en",
		Metadata:    metadata,
	}
	if _, err := h.store.SaveSiteSettings(input); err != nil {
		h.renderAdminSettings(w, adminStatus{Message: "We couldn't save those settings. Try again.", Error: true})
		return
	}
	if _, _, err := h.store.PublishSiteSettings(); err != nil {
		h.renderAdminSettings(w, adminStatus{Message: "We couldn't save those settings. Try again.", Error: true})
		return
	}
	http.Redirect(w, r, "/admin/settings?status="+queryEscape("Settings saved."), http.StatusSeeOther)
}

func (h *Host) writeAdminNotFound(w http.ResponseWriter, kind string) {
	body := h.renderAdminShell("pages", "We couldn't find that "+kind,
		"It may have been deleted, or the link may be out of date.",
		adminStatus{},
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "/admin/pages")), gosx.Text("Back to pages")),
		),
	)
	h.writeDocument(w, http.StatusNotFound, h.adminMeta("Not found"), body)
}

// ---------- form helpers ----------

func adminTextField(name, label, value, hint string) gosx.Node {
	nodes := []gosx.Node{
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "text"),
			gosx.Attr("id", "field-"+name),
			gosx.Attr("name", name),
			gosx.Attr("value", value),
		)),
	}
	if strings.TrimSpace(hint) != "" {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(hint)))
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-field"), gosx.Attr("for", "field-"+name)),
		gosx.Fragment(nodes...))
}

func adminTextareaField(name, label, value, hint string) gosx.Node {
	nodes := []gosx.Node{
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(
			gosx.Attr("id", "field-"+name),
			gosx.Attr("name", name),
			gosx.Attr("rows", "14"),
		), gosx.Text(value)),
	}
	if strings.TrimSpace(hint) != "" {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(hint)))
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-field"), gosx.Attr("for", "field-"+name)),
		gosx.Fragment(nodes...))
}

// homeEditHref opens the home page in the editor, falling back to the page list
// on a site that has no home page.
func (h *Host) homeEditHref() string {
	if page, ok, err := h.store.PageBySlug(homeSlug); err == nil && ok {
		return "/admin/edit/" + page.ID
	}
	return "/admin/pages"
}

// renderPageRow is one page in the admin list with its management actions.
func (h *Host) renderPageRow(page cmsstore.Page, first, last bool) gosx.Node {
	state, stateLabel := "draft", "Not published"
	switch {
	case PageOffline(page):
		state, stateLabel = "offline", "Offline"
	case h.isLive(page):
		state, stateLabel = "published", "Live"
	}
	isHome := page.Slug == homeSlug

	actions := []gosx.Node{}
	if !isHome {
		if !first {
			actions = append(actions, h.pageActionButton(page.ID, "up", "↑"))
		}
		if !last {
			actions = append(actions, h.pageActionButton(page.ID, "down", "↓"))
		}
		if PageNavHidden(page) {
			actions = append(actions, h.pageActionButton(page.ID, "show", "Show in menu"))
		} else {
			actions = append(actions, h.pageActionButton(page.ID, "hide", "Hide from menu"))
		}
		if PageOffline(page) {
			actions = append(actions, h.pageActionButton(page.ID, "online", "Put back online"))
		} else if h.isLive(page) {
			actions = append(actions, h.pageActionButton(page.ID, "offline", "Take offline"))
		}
		actions = append(actions, h.pageActionButton(page.ID, "archive", "Archive"))
	}

	name := gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/edit/"+page.ID)), gosx.Text(page.Title))
	if PageNavHidden(page) {
		name = gosx.Fragment(name, gosx.Text(" "), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge")), gosx.Text("not in menu")))
	}
	return gosx.El("tr", nil,
		gosx.El("td", nil, name),
		gosx.El("td", nil, gosx.Text(publicPath(page.Slug))),
		gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(stateLabel))),
		gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")), gosx.Fragment(actions...)),
	)
}

// isLive reports whether visitors can currently see a page.
func (h *Host) isLive(page cmsstore.Page) bool {
	_, ok := h.livePage(page)
	return ok
}

func (h *Host) pageActionButton(id, action, label string) gosx.Node {
	return gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/pages/"+id+"/action"), gosx.Attr("class", "admin-inline-form")),
		h.csrfField(),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "action"), gosx.Attr("value", action))),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", action)), gosx.Text(label)),
	)
}

func (h *Host) handleAdminPageAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/pages?status="+queryEscape("We couldn't read that. Try again."), http.StatusSeeOther)
		return
	}
	message, err := h.pageAction(r.PathValue("id"), strings.TrimSpace(r.PostFormValue("action")))
	if err != nil {
		if errors.Is(err, errPageNotFound) {
			h.writeAdminNotFound(w, "page")
			return
		}
		http.Redirect(w, r, "/admin/pages?status="+queryEscape("That didn't work. Try again."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/pages?status="+queryEscape(message), http.StatusSeeOther)
}

// ---------- brand fields on the Settings page ----------

func (h *Host) renderBrandFields(settings cmsstore.SiteSettings) gosx.Node {
	brand := brandFromSettings(settings)

	layout := func(key, label, hint string) gosx.Node {
		attrs := []any{gosx.Attr("type", "radio"), gosx.Attr("name", "headerLayout"), gosx.Attr("value", key), gosx.Attr("id", "layout-"+key)}
		if brand.HeaderLayout == key {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		return gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-choice"), gosx.Attr("for", "layout-"+key)),
			gosx.El("input", gosx.Attrs(attrs...)),
			gosx.El("span", nil, gosx.Text(label)),
			gosx.El("small", nil, gosx.Text(hint)),
		)
	}

	social := make([]gosx.Node, 0, 6)
	for _, network := range SocialNetworks() {
		social = append(social, adminTextField(socialKey(network.Key), network.Label, brand.Social[network.Key], network.Placeholder))
	}

	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Brand")),
		imageField("logo", "Logo", brand.LogoURL, "Shown in the header instead of your site name. PNG or JPEG, ideally on a transparent background."),
		imageField("favicon", "Browser tab icon", brand.FaviconURL, "The small square icon in the browser tab. Leave empty to use a letter mark in your accent colour."),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
			gosx.El("span", nil, gosx.Text("Header layout")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-choices")),
				layout("left", "Left", "Name or logo on the left, menu on the right."),
				layout("centered", "Centered", "Name or logo centred, menu beneath it."),
			),
		),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Footer")),
		adminTextField("footerText", "Footer text", brand.FooterText, "A line at the bottom of every page — an address, opening hours, or a tagline."),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Your email and phone from setup appear in the footer automatically.")),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Find us elsewhere")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Paste the address of each profile you want linked from the footer. Leave the rest empty.")),
		gosx.Fragment(social...),
	)
}

// imageField is a file input with a preview of the current image and a way
// to remove it. Uploads go through the same content-addressed store the
// editor uses.
func imageField(name, label, current, hint string) gosx.Node {
	nodes := []gosx.Node{gosx.El("span", nil, gosx.Text(label))}
	if current != "" {
		nodes = append(nodes,
			gosx.El("img", gosx.Attrs(gosx.Attr("class", "admin-image-preview"), gosx.Attr("src", current), gosx.Attr("alt", ""))),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "remove_"+name), gosx.Attr("value", "1"))),
				gosx.Text(" Remove this image"),
			),
		)
	}
	nodes = append(nodes,
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "file"), gosx.Attr("id", "field-"+name), gosx.Attr("name", name),
			gosx.Attr("accept", "image/png,image/jpeg,image/gif,image/webp"),
		)),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text(hint)),
	)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field admin-field--image")), gosx.Fragment(nodes...))
}

// applyBrandFields reads the Brand and Footer sections of a Settings post
// into metadata. It returns a message when an upload was refused.
func (h *Host) applyBrandFields(r *http.Request, metadata cmsstore.Metadata) string {
	for _, image := range []struct{ field, key, label string }{
		{"logo", brandLogoKey, "logo"},
		{"favicon", brandFaviconKey, "tab icon"},
	} {
		if r.PostFormValue("remove_"+image.field) == "1" {
			delete(metadata, image.key)
		}
		file, _, err := r.FormFile(image.field)
		if err != nil {
			continue // no file chosen
		}
		url, err := h.storeUpload(file)
		file.Close()
		switch {
		case errors.Is(err, errUploadNotImage):
			return "The " + image.label + " doesn't look like a picture. PNG, JPEG, GIF, and WebP work."
		case errors.Is(err, errUploadTooLarge):
			return "The " + image.label + " is too big. Pictures up to 10 MB work best."
		case err != nil:
			return "We couldn't save the " + image.label + ". Try again."
		}
		metadata[image.key] = url
	}

	metadata[brandHeaderLayoutKey] = normalizeHeaderLayout(r.PostFormValue("headerLayout"))
	if text := strings.TrimSpace(r.PostFormValue("footerText")); text != "" {
		metadata[brandFooterTextKey] = text
	} else {
		delete(metadata, brandFooterTextKey)
	}
	for _, network := range SocialNetworks() {
		key := socialKey(network.Key)
		if link := normalizeSocialLink(r.PostFormValue(key)); link != "" {
			metadata[key] = link
		} else {
			delete(metadata, key)
		}
	}
	return ""
}

// ---------- search engines and other services on the Settings page ----------

func (h *Host) renderGrowthFields(settings cmsstore.SiteSettings) gosx.Node {
	consentOff := settings.Metadata[consentKey] == "off"
	consentAttrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "cookieConsent"), gosx.Attr("value", "required")}
	if !consentOff {
		consentAttrs = append(consentAttrs, gosx.Attr("checked", "checked"))
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Search engines")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
			gosx.Text("Your site publishes a sitemap at "+sitemapPath+" and rules at "+robotsPath+" automatically. Submit the sitemap to Google Search Console to be found faster.")),
		adminTextareaField("redirects", "Redirects", formatRedirectLines(h.redirects()),
			"One per line: /old-address -> /new-address. When you rename a page, its old address is added here for you so links people already have keep working."),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Code from other services")),
		adminTextareaField("headCode", "Paste code here", h.headCode(),
			"Analytics, chat widgets, or pixels from other services usually give you a snippet to paste. It goes into every page of your public site. Pasting code here relaxes the site's script policy to let it run."),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")),
			gosx.El("input", gosx.Attrs(consentAttrs...)),
			gosx.Text(" Ask visitors before running it (shows a cookie notice; recommended in the EU and UK)"),
		),
	)
}

func applyGrowthFields(r *http.Request, metadata cmsstore.Metadata) {
	table := parseRedirectLines(r.PostFormValue("redirects"))
	if len(table) == 0 {
		delete(metadata, redirectsKey)
	} else if data, err := json.Marshal(table); err == nil {
		metadata[redirectsKey] = string(data)
	}
	if code := strings.TrimSpace(r.PostFormValue("headCode")); code != "" {
		metadata[headCodeKey] = code
	} else {
		delete(metadata, headCodeKey)
	}
	if r.PostFormValue("cookieConsent") == "required" {
		delete(metadata, consentKey)
	} else {
		metadata[consentKey] = "off"
	}
}

// ---------- email on the Settings page ----------

func (h *Host) renderMailFields(settings cmsstore.SiteSettings) gosx.Node {
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Email")),
		adminTextField("notifyEmail", "Send new messages to", settings.Metadata[notifyEmailKey],
			"Leave empty to use the contact email from setup ("+firstNonEmpty(settings.Metadata["contactEmail"], "not set")+")."),
	)
}

// renderMailTestPanel shows whether email works and lets the owner prove it.
func (h *Host) renderMailTestPanel() gosx.Node {
	if h.mailer == nil {
		return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Email delivery")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
				gosx.Text("No email service is set up, so new messages wait in your inbox here until you look. To be emailed about them, start the site with -mail (or GOSX_SITE_MAIL) set to an smtp://, resend://, or postmark:// address.")),
		)
	}
	last, lastErr, attempts, sent := h.mailStatus.snapshot()
	state := "Not tried yet."
	stateClass := ""
	switch {
	case attempts == 0:
	case lastErr != "":
		state = "Last attempt failed: " + lastErr
		stateClass = "error"
	default:
		state = "Working. Last email sent " + last.Local().Format("Mon 2 Jan, 3:04 PM") + "."
	}
	_ = sent
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Email delivery")),
		gosx.El("p", nil, gosx.Text(h.mailer.Describe()+".")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-status"), gosx.Attr("data-state", stateClass)), gosx.Text(state)),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/settings/test-mail")),
			h.csrfField(),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Send a test email to "+firstNonEmpty(h.notifyAddress(), "(no address set)"))),
			),
		),
	)
}

func (h *Host) handleAdminTestMail(w http.ResponseWriter, r *http.Request) {
	to := h.notifyAddress()
	if to == "" {
		http.Redirect(w, r, "/admin/settings?status="+queryEscape("Add an email address first, then send the test."), http.StatusSeeOther)
		return
	}
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	err := h.sendNow(Mail{
		To:      to,
		Subject: "Test email from " + siteTitle,
		Text:    "If you're reading this, email from your website works. New messages from your contact form will arrive the same way.\n",
	})
	if err != nil {
		http.Redirect(w, r, "/admin/settings?status="+queryEscape("The test email didn't send: "+err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/settings?status="+queryEscape("Test email sent to "+to+". Check your inbox (and spam folder)."), http.StatusSeeOther)
}
