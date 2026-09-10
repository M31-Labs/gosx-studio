package sitehost

import (
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/hostruntime"
)

// PageMeta is everything the document shell needs to build a complete <head>.
//
// Studio historically emitted no <head> tags at all. Meta title, description,
// and share image were collected by four back-office forms and consumed by
// nothing, because "hosts supply routes" put the document shell outside
// Studio's boundary. The default host owns the shell, so every site it serves
// gets titles, descriptions, canonical URLs, and share cards without anyone
// writing Go.
type PageMeta struct {
	Title       string
	Description string
	CanonicalPath string
	ImageURL    string
	ImageAlt    string
	SiteTitle   string
	BaseURL     string
	Kind        string // "website" or "article"
	AdminChrome bool
	NoIndex     bool
}

func metaFromSettings(settings cmsstore.SiteSettings) PageMeta {
	return PageMeta{
		SiteTitle: strings.TrimSpace(settings.Title),
		BaseURL:   strings.TrimSpace(settings.BaseURL),
	}
}

// documentTitle composes "Page — Site", collapsing to one part when the page
// title is empty or already equals the site title.
func (m PageMeta) documentTitle() string {
	title := strings.TrimSpace(m.Title)
	site := strings.TrimSpace(m.SiteTitle)
	switch {
	case title == "" && site == "":
		return "Untitled site"
	case title == "":
		return site
	case site == "" || strings.EqualFold(title, site):
		return title
	default:
		return title + " — " + site
	}
}

func (m PageMeta) canonicalURL() string {
	base := strings.TrimRight(strings.TrimSpace(m.BaseURL), "/")
	path := strings.TrimSpace(m.CanonicalPath)
	if base == "" || path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func (m PageMeta) shareImageURL() string {
	image := strings.TrimSpace(m.ImageURL)
	if image == "" {
		return ""
	}
	if strings.HasPrefix(image, "http://") || strings.HasPrefix(image, "https://") {
		return image
	}
	base := strings.TrimRight(strings.TrimSpace(m.BaseURL), "/")
	if base == "" {
		return image
	}
	if !strings.HasPrefix(image, "/") {
		image = "/" + image
	}
	return base + image
}

func metaTag(name, content string) gosx.Node {
	if strings.TrimSpace(content) == "" {
		return gosx.Fragment()
	}
	return gosx.El("meta", gosx.Attrs(gosx.Attr("name", name), gosx.Attr("content", content)))
}

func propertyTag(property, content string) gosx.Node {
	if strings.TrimSpace(content) == "" {
		return gosx.Fragment()
	}
	return gosx.El("meta", gosx.Attrs(gosx.Attr("property", property), gosx.Attr("content", content)))
}

// RenderHead builds the document head for one page.
func RenderHead(meta PageMeta) gosx.Node {
	kind := strings.TrimSpace(meta.Kind)
	if kind == "" {
		kind = "website"
	}
	title := meta.documentTitle()
	canonical := meta.canonicalURL()
	image := meta.shareImageURL()

	nodes := []gosx.Node{
		gosx.El("meta", gosx.Attrs(gosx.Attr("charset", "utf-8"))),
		gosx.El("meta", gosx.Attrs(
			gosx.Attr("name", "viewport"),
			gosx.Attr("content", "width=device-width, initial-scale=1"),
		)),
		gosx.El("title", nil, gosx.Text(title)),
		metaTag("description", meta.Description),
	}

	if canonical != "" {
		nodes = append(nodes, gosx.El("link", gosx.Attrs(
			gosx.Attr("rel", "canonical"),
			gosx.Attr("href", canonical),
		)))
	}
	if meta.NoIndex {
		nodes = append(nodes, metaTag("robots", "noindex, nofollow"))
	}

	nodes = append(nodes,
		propertyTag("og:type", kind),
		propertyTag("og:title", title),
		propertyTag("og:description", meta.Description),
		propertyTag("og:url", canonical),
		propertyTag("og:site_name", meta.SiteTitle),
		propertyTag("og:image", image),
	)
	if image != "" {
		nodes = append(nodes,
			propertyTag("og:image:alt", meta.ImageAlt),
			metaTag("twitter:card", "summary_large_image"),
		)
	} else {
		nodes = append(nodes, metaTag("twitter:card", "summary"))
	}
	nodes = append(nodes,
		metaTag("twitter:title", title),
		metaTag("twitter:description", meta.Description),
		metaTag("twitter:image", image),
	)

	nodes = append(nodes, gosx.El("link", gosx.Attrs(
		gosx.Attr("rel", "stylesheet"),
		gosx.Attr("href", publicStylesheetPath),
	)))
	if meta.AdminChrome {
		nodes = append(nodes, gosx.El("link", gosx.Attrs(
			gosx.Attr("rel", "stylesheet"),
			gosx.Attr("href", hostruntime.StylesheetPath),
		)))
	}

	return gosx.El("head", nil, gosx.Fragment(nodes...))
}

// RenderDocument wraps rendered body content in a complete HTML document.
func RenderDocument(meta PageMeta, body gosx.Node) string {
	lang := "en"
	html := gosx.El("html", gosx.Attrs(gosx.Attr("lang", lang)),
		RenderHead(meta),
		gosx.El("body", gosx.Attrs(gosx.Attr("class", bodyClass(meta))), body),
	)
	return "<!doctype html>" + gosx.RenderHTML(html)
}

func bodyClass(meta PageMeta) string {
	if meta.AdminChrome {
		return "gosx-site gosx-site--admin"
	}
	return "gosx-site gosx-site--public"
}
