package sitehost

import (
	"encoding/json"
	"net/http"
	"strings"

	"m31labs.dev/gosx-studio/cms/lifecycle"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/render"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

const homeSlug = "home"

func (h *Host) mountPublic(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", h.handlePublicHome)
	mux.HandleFunc("GET /{slug}", h.handlePublicPage)
}

func (h *Host) handlePublicHome(w http.ResponseWriter, r *http.Request) {
	h.servePublicSlug(w, r, homeSlug)
}

func (h *Host) handlePublicPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		slug = homeSlug
	}
	h.servePublicSlug(w, r, slug)
}

// livePage returns the version of one page a visitor should see.
//
// The store keeps a single record per page, and saving a draft moves that
// record out of the published state and clears PublishedAt. Serving the record
// directly would mean the first keystroke in the editor takes a live page off
// the internet. So "is this live?" is answered by the revision ledger, which
// remembers every publish, rather than by the working record.
func (h *Host) livePage(page cmsstore.Page) (cmsstore.Page, bool) {
	// The owner's own switches are read from the working record, which is
	// where they are set; a published snapshot in the ledger predates them.
	if PageOffline(page) || PageArchived(page) {
		return cmsstore.Page{}, false
	}
	if page.State.Publish == cmsstore.PublishStatePublished {
		return page, true
	}
	return h.lastPublishedSnapshot(page.ID)
}

// lastPublishedSnapshot decodes the newest "page.published" revision.
func (h *Host) lastPublishedSnapshot(pageID string) (cmsstore.Page, bool) {
	revisions := h.store.ListRevisions(lifecycle.Filter{
		ResourceKind: cmsstore.ResourceKindPage,
		ResourceID:   pageID,
	})
	for index := len(revisions) - 1; index >= 0; index-- {
		revision := revisions[index]
		if revision.Action != cmsstore.ActionPagePublished || len(revision.Snapshot) == 0 {
			continue
		}
		var page cmsstore.Page
		if err := json.Unmarshal(revision.Snapshot, &page); err != nil {
			continue
		}
		return page, true
	}
	return cmsstore.Page{}, false
}

// livePages is every page as a visitor currently sees it, home first.
func (h *Host) livePages() []cmsstore.Page {
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return nil
	}
	live := make([]cmsstore.Page, 0, len(pages))
	for _, page := range pages {
		if published, ok := h.livePage(page); ok {
			live = append(live, published)
		}
	}
	return h.orderPages(live)
}

// publishedPage resolves a slug against the live versions, so renaming a page
// in a draft leaves the old address working until the owner publishes.
func (h *Host) publishedPage(slug string) (cmsstore.Page, bool) {
	for _, page := range h.livePages() {
		if page.Slug == slug {
			return page, true
		}
	}
	return cmsstore.Page{}, false
}

func (h *Host) servePublicSlug(w http.ResponseWriter, r *http.Request, slug string) {
	settings := h.settings()
	page, ok := h.publishedPage(slug)
	if !ok {
		h.servePublicNotFound(w, settings, slug)
		return
	}

	meta := metaFromSettings(settings)
	meta.Title = page.Title
	meta.Description = firstNonEmpty(page.Description, settings.Description)
	meta.CanonicalPath = publicPath(page.Slug)
	meta.Kind = "article"
	if page.Slug == homeSlug {
		meta.Kind = "website"
	}
	// A page with no share image of its own falls back to the logo, so a
	// shared link never renders as a naked URL once a logo exists.
	brand := brandFromSettings(settings)
	meta.ImageURL = firstNonEmpty(page.Metadata["metaImageUrl"], brand.LogoURL)
	meta.ImageAlt = firstNonEmpty(page.Metadata["metaImageAlt"], settings.Title)
	if title := strings.TrimSpace(page.Metadata["metaTitle"]); title != "" {
		meta.Title = title
	}
	if description := strings.TrimSpace(page.Metadata["metaDescription"]); description != "" {
		meta.Description = description
	}

	body := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-shell")),
		h.renderPublicNav(settings, page.Slug),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")),
			gosx.El("article", gosx.Attrs(gosx.Attr("class", "site-article")),
				gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text(page.Title)),
				render.Document(page.Body, render.Hooks{Flow: h.flowHook(publicPath(page.Slug), formStateFromQuery(r))}),
			),
		),
		h.renderPublicFooter(settings),
	)
	h.writeDocument(w, http.StatusOK, meta, body)
}

func (h *Host) servePublicNotFound(w http.ResponseWriter, settings cmsstore.SiteSettings, slug string) {
	meta := metaFromSettings(settings)
	meta.Title = "Page not found"
	meta.Description = "That page does not exist on this site."
	meta.NoIndex = true

	body := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-shell")),
		h.renderPublicNav(settings, ""),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")),
			gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text("We couldn't find that page")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")),
				gosx.Text("There is no published page at /"+slug+" yet. Check the address, or head back to the home page.")),
			gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("href", "/")), gosx.Text("Go to the home page"))),
		),
		h.renderPublicFooter(settings),
	)
	h.writeDocument(w, http.StatusNotFound, meta, body)
}

// navPages lists the pages that appear in the site menu: live, and not
// hidden from it. A hidden page is still served at its address.
func (h *Host) navPages() []cmsstore.Page {
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return nil
	}
	hidden := map[string]bool{}
	for _, page := range pages {
		if PageNavHidden(page) {
			hidden[page.ID] = true
		}
	}
	out := make([]cmsstore.Page, 0, len(pages))
	for _, page := range h.livePages() {
		if !hidden[page.ID] {
			out = append(out, page)
		}
	}
	return out
}

func (h *Host) renderPublicNav(settings cmsstore.SiteSettings, activeSlug string) gosx.Node {
	return h.renderSiteHeader(settings, brandFromSettings(settings), activeSlug, false)
}

func (h *Host) renderPublicFooter(settings cmsstore.SiteSettings) gosx.Node {
	return h.renderSiteFooter(settings, brandFromSettings(settings))
}

func publicPath(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" || slug == homeSlug {
		return "/"
	}
	return "/" + slug
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
