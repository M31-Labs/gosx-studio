package sitehost

import (
	"strconv"
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
)

// checks.go is the "before you publish" list: the handful of things a
// visitor or a search engine would notice that an owner would not.
//
// None of them block publishing. They are shown beside the page, refreshed
// on every save, and written so the fix is obvious.

// readinessChecks reports what is worth fixing on one page or post.
func (h *Host) readinessChecks(kind, description string, body blockstudio.Document) []string {
	out := []string{}
	missingAlt := 0
	blocks := 0
	broken := map[string]bool{}
	var brokenList []string
	note := func(path string) {
		if !broken[path] {
			broken[path] = true
			brokenList = append(brokenList, path)
		}
	}

	for _, instance := range body.Blocks {
		if !instance.Enabled {
			continue
		}
		blocks++
		switch instance.Key {
		case content.BlockImage:
			if strings.TrimSpace(instance.Values["url"].String) != "" && strings.TrimSpace(instance.Values["alt"].String) == "" {
				missingAlt++
			}
		case content.BlockGallery:
			for _, image := range galleryImages(instance) {
				if image[1] == "" {
					missingAlt++
				}
			}
		case content.BlockButton:
			if href := strings.TrimSpace(instance.Values["href"].String); !h.internalLinkExists(href) {
				note(href)
			}
		}
		for _, key := range []string{"text", "text2"} {
			for _, href := range inlineLinks(instance.Values[key].String) {
				if !h.internalLinkExists(href) {
					note(href)
				}
			}
		}
	}

	if blocks == 0 {
		out = append(out, "The "+kind+" is empty. Add a heading and some text.")
	}
	if kind == "page" && strings.TrimSpace(description) == "" {
		out = append(out, "No description for search results yet. Add one or two sentences in the sidebar.")
	}
	switch missingAlt {
	case 0:
	case 1:
		out = append(out, "1 picture has no description. Describe it for people who can't see it and for search engines.")
	default:
		out = append(out, strconv.Itoa(missingAlt)+" pictures have no description. Describe them for people who can't see them and for search engines.")
	}
	for i, path := range brokenList {
		if i == 3 {
			out = append(out, "…and "+strconv.Itoa(len(brokenList)-3)+" more links that go nowhere.")
			break
		}
		out = append(out, "A link points at "+path+", which doesn't exist on your site.")
	}
	return out
}

// inlineLinks lists the addresses inside a block's inline markers.
func inlineLinks(text string) []string {
	var out []string
	for i := 0; i < len(text); i++ {
		if text[i] != '[' {
			continue
		}
		_, href, consumed := parseLink(text[i:])
		if consumed == 0 {
			continue
		}
		out = append(out, href)
		i += consumed - 1
	}
	return out
}

// internalLinkExists is true for anything external and for site-relative
// paths that lead somewhere: a page, a post, the blog, a picture, an anchor.
func (h *Host) internalLinkExists(href string) bool {
	href = strings.TrimSpace(href)
	if href == "" || !strings.HasPrefix(href, "/") || strings.HasPrefix(href, "//") {
		return true
	}
	path := href
	if at := strings.IndexAny(path, "?#"); at >= 0 {
		path = path[:at]
	}
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return true
	}
	if strings.HasPrefix(path, uploadsURLPrefix) || path == blogPath || path == feedPath || path == sitemapPath || strings.HasPrefix(path, blogPath+"/category/") {
		return true
	}
	if strings.HasPrefix(path, blogPath+"/") {
		_, exists, _ := h.store.PostBySlug(strings.TrimPrefix(path, blogPath+"/"))
		return exists
	}
	if _, moved := h.resolveRedirect(path); moved {
		return true
	}
	slug := strings.TrimPrefix(path, "/")
	if strings.Contains(slug, "/") {
		return false
	}
	_, exists, _ := h.store.PageBySlug(slug)
	return exists
}
