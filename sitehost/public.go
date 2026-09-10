package sitehost

import (
	"net/http"
	"sort"
	"strings"

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

// publishedPage returns a page only when it is live. Draft work stays private
// until the owner publishes it.
func (h *Host) publishedPage(slug string) (cmsstore.Page, bool) {
	page, ok, err := h.store.PageBySlug(slug)
	if err != nil || !ok {
		return cmsstore.Page{}, false
	}
	if page.State.Publish != cmsstore.PublishStatePublished {
		return cmsstore.Page{}, false
	}
	return page, true
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

// navPages lists published pages for the site navigation, home first.
func (h *Host) navPages() []cmsstore.Page {
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return nil
	}
	live := make([]cmsstore.Page, 0, len(pages))
	for _, page := range pages {
		if page.State.Publish == cmsstore.PublishStatePublished {
			live = append(live, page)
		}
	}
	sort.SliceStable(live, func(i, j int) bool {
		if live[i].Slug == homeSlug {
			return true
		}
		if live[j].Slug == homeSlug {
			return false
		}
		return live[i].Title < live[j].Title
	})
	return live
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
