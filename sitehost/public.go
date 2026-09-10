package sitehost

import (
	"encoding/json"
	"net/http"
	"sort"
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
	// Home first; everything else keeps the order it was created in, which is
	// the order the starter template meant them to read.
	sort.SliceStable(live, func(i, j int) bool {
		return live[i].Slug == homeSlug && live[j].Slug != homeSlug
	})
	return live
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
	meta.ImageURL = page.Metadata["metaImageUrl"]
	meta.ImageAlt = page.Metadata["metaImageAlt"]
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
				render.Document(page.Body, render.Hooks{}),
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

// navPages lists the pages that appear in the site menu.
func (h *Host) navPages() []cmsstore.Page {
	return h.livePages()
}

func (h *Host) renderPublicNav(settings cmsstore.SiteSettings, activeSlug string) gosx.Node {
	links := make([]gosx.Node, 0, 8)
	for _, page := range h.navPages() {
		if page.Slug == homeSlug {
			continue
		}
		attrs := []any{gosx.Attr("href", publicPath(page.Slug))}
		if page.Slug == activeSlug {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(page.Title)))
	}

	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "site-header")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "site-brand"), gosx.Attr("href", "/")),
			gosx.Text(firstNonEmpty(settings.Title, h.opts.SiteTitle))),
		gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-nav"), gosx.Attr("aria-label", "Site")),
			gosx.Fragment(links...)),
	)
}

func (h *Host) renderPublicFooter(settings cmsstore.SiteSettings) gosx.Node {
	return gosx.El("footer", gosx.Attrs(gosx.Attr("class", "site-footer")),
		gosx.El("p", nil, gosx.Text(firstNonEmpty(settings.Title, h.opts.SiteTitle))),
	)
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
