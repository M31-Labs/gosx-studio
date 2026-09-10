package sitehost

import (
	"net/http"
	"sort"
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
	mux.HandleFunc("GET /admin/settings", h.handleAdminSettings)
	mux.HandleFunc("GET /admin/settings/{$}", h.handleAdminSettings)
	mux.HandleFunc("POST /admin/settings", h.handleAdminSaveSettings)
	mux.HandleFunc("POST /admin/settings/{$}", h.handleAdminSaveSettings)
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
	navItems := []struct{ Key, Label, Href string }{
		{"dashboard", "Dashboard", "/admin"},
		{"pages", "Pages", "/admin/pages"},
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
		adminStat(len(h.store.ListRevisions(revisionFilterAll())), "Saved versions"),
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
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/pages")), gosx.Text("All pages")),
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
	sort.SliceStable(pages, func(i, j int) bool {
		return pages[i].Slug == homeSlug && pages[j].Slug != homeSlug
	})
	return pages
}

func (h *Host) handleAdminPages(w http.ResponseWriter, r *http.Request) {
	h.renderAdminPages(w, r, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderAdminPages(w http.ResponseWriter, r *http.Request, status adminStatus) {
	pages := h.sortedPages()

	var listing gosx.Node
	if len(pages) == 0 {
		// The empty state Studio's generic index renderer never ships.
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No pages yet")),
			gosx.El("p", nil, gosx.Text("Pages are what visitors read on your website — a home page, an about page, a contact page. Create your first one below.")),
		)
	} else {
		rows := make([]gosx.Node, 0, len(pages))
		for _, page := range pages {
			state, stateLabel := "draft", "Not published"
			if page.State.Publish == cmsstore.PublishStatePublished {
				state, stateLabel = "published", "Live"
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/edit/"+page.ID)), gosx.Text(page.Title))),
				gosx.El("td", nil, gosx.Text(publicPath(page.Slug))),
				gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(stateLabel))),
			))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your pages")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil,
					gosx.El("th", nil, gosx.Text("Page")),
					gosx.El("th", nil, gosx.Text("Address")),
					gosx.El("th", nil, gosx.Text("Status")),
				)),
				gosx.El("tbody", nil, gosx.Fragment(rows...)),
			),
		)
	}

	create := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Add a page")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/pages")),
			adminTextField("title", "Page name", "", "Shown as the heading and in your site menu."),
			adminTextField("slug", "Web address", "", "Letters and dashes only. \"about-us\" becomes yoursite.com/about-us."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Create page")),
			),
		),
	)

	body := h.renderAdminShell("pages", "Pages",
		"Create, edit, and publish the pages on your website.",
		status, listing, create)
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
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/settings")),
			adminTextField("title", "Site name", settings.Title, "Shown in the browser tab, your site menu, and search results."),
			adminTextField("description", "Site description", settings.Description,
				"One or two sentences about your business. Search engines show this under your site name."),
			adminTextField("baseURL", "Website address", settings.BaseURL,
				"For example https://yourbusiness.com. Needed so shared links and search results point at the right place."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Save settings")),
			),
		),
	)
	body := h.renderAdminShell("settings", "Settings",
		"These details appear in search results and when someone shares a link to your site.",
		status, form)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Settings"), body)
}

func (h *Host) handleAdminSaveSettings(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAdminSettings(w, adminStatus{Message: "We couldn't read that form. Try again.", Error: true})
		return
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
