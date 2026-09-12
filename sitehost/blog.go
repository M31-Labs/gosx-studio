package sitehost

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	"m31labs.dev/gosx-studio/cms/render"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// blog.go is the blog: posts written on the same canvas as pages, an index
// at /blog, one address per post, category pages, an RSS feed, and a
// "publish at" date.
//
// Studio's store has had posts all along — cms/store's Post, with tags and a
// lifecycle — and no host ever gave them a surface. The default host does,
// without adding a second content model: a post is a Post, categories are
// its Tags, and scheduling is one metadata key read at request time. Nothing
// runs in the background; a scheduled post simply becomes live when the
// clock passes its date.

const (
	blogPath      = "/blog"
	feedPath      = "/feed.xml"
	blogTitleKey  = "blogTitle"
	blogPageSize  = 10
	excerptLength = 160
)

// timeNow is the clock for scheduling. Tests set it.
var timeNow = time.Now

// reservedSlugs are addresses the host serves itself; a page cannot take one.
var reservedSlugs = map[string]bool{
	"blog": true, "feed.xml": true, "admin": true, "setup": true, "uploads": true,
	"sitemap.xml": true, "robots.txt": true, "healthz": true, "contact-send": true,
}

func reservedSlugMessage(slug string) string {
	if reservedSlugs[slug] {
		return "The address /" + slug + " is reserved. Pick a different one."
	}
	return ""
}

func (h *Host) mountBlog(mux *http.ServeMux) {
	mux.HandleFunc("GET "+blogPath, h.handleBlogIndex)
	mux.HandleFunc("GET "+blogPath+"/{$}", h.handleBlogIndex)
	mux.HandleFunc("GET "+blogPath+"/category/{tag}", h.handleBlogCategory)
	mux.HandleFunc("GET "+blogPath+"/{slug}", h.handleBlogPost)
	mux.HandleFunc("GET "+feedPath, h.handleFeed)

	mux.HandleFunc("GET /admin/posts", h.handleAdminPosts)
	mux.HandleFunc("GET /admin/posts/{$}", h.handleAdminPosts)
	mux.HandleFunc("POST /admin/posts", h.handleAdminCreatePost)
	mux.HandleFunc("POST /admin/posts/{$}", h.handleAdminCreatePost)
	mux.HandleFunc("POST /admin/posts/{id}/action", h.handleAdminPostAction)
	mux.HandleFunc("GET /admin/edit/post/{id}", h.handlePostEditor)
	mux.HandleFunc("POST /admin/api/posts/{id}", h.handlePostSave)
	mux.HandleFunc("POST /admin/api/posts/{id}/publish", h.handlePostPublish)
}

// ---------- flags, dates, liveness ----------

func postFlag(post cmsstore.Post, key string) bool {
	return strings.TrimSpace(post.Metadata[key]) == "true"
}

// PostOffline reports a post the owner has taken off the site.
func PostOffline(post cmsstore.Post) bool { return postFlag(post, pageOfflineKey) }

// PostArchived reports a post the owner has put away.
func PostArchived(post cmsstore.Post) bool { return postFlag(post, pageArchivedKey) }

var errPostNotFound = errors.New("post not found")

func postInput(post cmsstore.Post, metadata cmsstore.Metadata) cmsstore.PostInput {
	return cmsstore.PostInput{
		Slug: post.Slug, Title: post.Title, Excerpt: post.Excerpt, Author: post.Author,
		Tags: post.Tags, Body: post.Body, State: post.State, Metadata: metadata,
	}
}

func (h *Host) setPostFlag(id, key string, on bool) (cmsstore.Post, error) {
	post, ok, err := h.store.PostByID(id)
	if err != nil {
		return cmsstore.Post{}, err
	}
	if !ok {
		return cmsstore.Post{}, errPostNotFound
	}
	metadata := cloneMetadata(post.Metadata)
	if on {
		metadata[key] = "true"
	} else {
		delete(metadata, key)
	}
	return h.store.UpdatePost(post.ID, postInput(post, metadata))
}

func cloneMetadata(in cmsstore.Metadata) cmsstore.Metadata {
	out := cmsstore.Metadata{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

// postScheduled reports a post with a publish waiting for its date.
func postScheduled(post cmsstore.Post) (time.Time, bool) {
	return scheduledFor(post.Metadata)
}

// postDate is the date shown on a post: the chosen one, else when it was
// published, else when it was written.
func postDate(post cmsstore.Post) time.Time {
	if at, ok := PostPublishAt(post); ok {
		return at
	}
	if post.State.PublishedAt != nil && !post.State.PublishedAt.IsZero() {
		return *post.State.PublishedAt
	}
	return post.Created
}

// livePost returns the version of one post a visitor should see, by the
// same ledger rule as livePage.
func (h *Host) livePost(post cmsstore.Post) (cmsstore.Post, bool) {
	if PostOffline(post) || PostArchived(post) {
		return cmsstore.Post{}, false
	}
	if post.State.Publish == cmsstore.PublishStatePublished {
		return post, true
	}
	revision, ok := h.latestPublished(cmsstore.ResourceKindPost, post.ID, cmsstore.ActionPostPublished)
	if !ok {
		return cmsstore.Post{}, false
	}
	var snapshot cmsstore.Post
	if err := json.Unmarshal(revision.Snapshot, &snapshot); err != nil {
		return cmsstore.Post{}, false
	}
	return snapshot, true
}

// livePosts is every post a visitor can read, newest first.
func (h *Host) livePosts() []cmsstore.Post {
	h.PublishDue()
	posts, err := h.store.ListPosts(cmsstore.PostFilter{})
	if err != nil {
		return nil
	}
	live := make([]cmsstore.Post, 0, len(posts))
	for _, post := range posts {
		if published, ok := h.livePost(post); ok {
			live = append(live, published)
		}
	}
	sortPostsNewestFirst(live)
	return live
}

func sortPostsNewestFirst(posts []cmsstore.Post) {
	sort.SliceStable(posts, func(i, j int) bool {
		return postDate(posts[i]).After(postDate(posts[j]))
	})
}

func (h *Host) livePostBySlug(slug string) (cmsstore.Post, bool) {
	for _, post := range h.livePosts() {
		if post.Slug == slug {
			return post, true
		}
	}
	return cmsstore.Post{}, false
}

// blogInMenu reports whether the site menu shows a Blog link: it does the
// moment a first post is live, and never before.
func (h *Host) blogInMenu() bool {
	return len(h.livePosts()) > 0
}

func (h *Host) blogTitle() string {
	return firstNonEmpty(strings.TrimSpace(h.settings().Metadata[blogTitleKey]), "Blog")
}

func postPath(slug string) string { return blogPath + "/" + slug }

func categorySlug(tag string) string { return normalizeSlug(tag) }

func categoryPath(tag string) string { return blogPath + "/category/" + categorySlug(tag) }

// postCategory is one category with how many live posts carry it.
type postCategory struct {
	Name  string
	Slug  string
	Count int
}

func postCategories(posts []cmsstore.Post) []postCategory {
	counts := map[string]*postCategory{}
	for _, post := range posts {
		for _, tag := range post.Tags {
			slug := categorySlug(tag)
			if slug == "" {
				continue
			}
			if entry, ok := counts[slug]; ok {
				entry.Count++
			} else {
				counts[slug] = &postCategory{Name: tag, Slug: slug, Count: 1}
			}
		}
	}
	out := make([]postCategory, 0, len(counts))
	for _, entry := range counts {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// postExcerpt is the owner's summary, or the first paragraph shortened.
func postExcerpt(post cmsstore.Post) string {
	if excerpt := strings.TrimSpace(post.Excerpt); excerpt != "" {
		return excerpt
	}
	for _, instance := range post.Body.Blocks {
		if !instance.Enabled || instance.Key != content.BlockParagraph {
			continue
		}
		if text := blockPlainText(instance); text != "" {
			return shorten(text, excerptLength)
		}
	}
	return ""
}

func shorten(text string, limit int) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= limit {
		return text
	}
	cut := strings.LastIndex(text[:limit], " ")
	if cut < limit/2 {
		cut = limit
	}
	return strings.TrimRight(text[:cut], ",;:.") + "…"
}

func formatPostDate(at time.Time) string {
	return at.In(time.Local).Format("2 January 2006")
}

// ---------- public: index, category, post, feed ----------

func (h *Host) handleBlogIndex(w http.ResponseWriter, r *http.Request) {
	h.serveBlogList(w, r, h.livePosts(), "", "")
}

func (h *Host) handleBlogCategory(w http.ResponseWriter, r *http.Request) {
	wanted := normalizeSlug(r.PathValue("tag"))
	all := h.livePosts()
	name := ""
	matching := make([]cmsstore.Post, 0, len(all))
	for _, post := range all {
		for _, tag := range post.Tags {
			if categorySlug(tag) == wanted {
				matching = append(matching, post)
				if name == "" {
					name = tag
				}
				break
			}
		}
	}
	if len(matching) == 0 {
		h.servePublicNotFound(w, h.settings(), strings.TrimPrefix(categoryPath(wanted), "/"))
		return
	}
	h.serveBlogList(w, r, matching, name, wanted)
}

func (h *Host) serveBlogList(w http.ResponseWriter, r *http.Request, posts []cmsstore.Post, category, categoryKey string) {
	settings := h.settings()
	title := h.blogTitle()
	path := blogPath
	if category != "" {
		title = category
		path = blogPath + "/category/" + categoryKey
	}

	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 1 {
			page = n
		}
	}
	start := (page - 1) * blogPageSize
	if start > len(posts) {
		start = len(posts)
	}
	end := start + blogPageSize
	if end > len(posts) {
		end = len(posts)
	}

	meta := metaFromSettings(settings)
	meta.Title = title
	meta.Description = firstNonEmpty(settings.Description, "Posts from "+firstNonEmpty(settings.Title, h.opts.SiteTitle)+".")
	if category != "" {
		meta.Description = "Posts about " + category + " from " + firstNonEmpty(settings.Title, h.opts.SiteTitle) + "."
	}
	meta.CanonicalPath = path
	if page > 1 {
		meta.CanonicalPath = path + "?page=" + strconv.Itoa(page)
	}
	meta.Kind = "website"
	meta.Feed = len(h.livePosts()) > 0
	meta.HeadCode = h.headCode()
	meta.Consent = h.consentRequired()
	brand := brandFromSettings(settings)
	meta.ImageURL = brand.LogoURL
	meta.ImageAlt = settings.Title

	status := http.StatusOK
	var body gosx.Node
	if len(posts) == 0 {
		status = http.StatusNotFound
		meta.NoIndex = true
		body = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("Nothing posted yet. Check back soon."))
	} else {
		body = gosx.Fragment(
			h.renderCategoryNav(categoryKey),
			renderPostList(posts[start:end]),
			renderPager(path, page, end < len(posts)),
		)
	}

	var consent gosx.Node = gosx.Fragment()
	if meta.HeadCode != "" && meta.Consent {
		consent = renderConsentBanner()
	}
	heading := title
	if category != "" {
		heading = "Posts about " + category
	}
	shell := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-shell")),
		h.renderPublicNav(settings, "blog"),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")),
			gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-article site-blog")),
				gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text(heading)),
				body,
			),
		),
		h.renderPublicFooter(settings),
		consent,
	)
	h.writeDocument(w, status, meta, shell)
}

func (h *Host) renderCategoryNav(activeKey string) gosx.Node {
	categories := postCategories(h.livePosts())
	if len(categories) == 0 {
		return gosx.Fragment()
	}
	links := make([]gosx.Node, 0, len(categories)+1)
	allAttrs := []any{gosx.Attr("href", blogPath)}
	if activeKey == "" {
		allAttrs = append(allAttrs, gosx.Attr("aria-current", "page"))
	}
	links = append(links, gosx.El("a", gosx.Attrs(allAttrs...), gosx.Text("All posts")))
	for _, category := range categories {
		attrs := []any{gosx.Attr("href", blogPath+"/category/"+category.Slug)}
		if category.Slug == activeKey {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(category.Name+" ("+strconv.Itoa(category.Count)+")")))
	}
	return gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-categories"), gosx.Attr("aria-label", "Categories")), gosx.Fragment(links...))
}

func renderPostList(posts []cmsstore.Post) gosx.Node {
	items := make([]gosx.Node, 0, len(posts))
	for _, post := range posts {
		card := []gosx.Node{
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "site-post-card__title")),
				gosx.El("a", gosx.Attrs(gosx.Attr("href", postPath(post.Slug))), gosx.Text(post.Title))),
			renderPostMeta(post),
		}
		if excerpt := postExcerpt(post); excerpt != "" {
			card = append(card, gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-card__excerpt")), gosx.Text(excerpt)))
		}
		items = append(items, gosx.El("li", gosx.Attrs(gosx.Attr("class", "site-post-card")), gosx.Fragment(card...)))
	}
	return gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-posts")), gosx.Fragment(items...))
}

// renderPostMeta is the date, author, and categories line under a title.
func renderPostMeta(post cmsstore.Post) gosx.Node {
	parts := []gosx.Node{gosx.El("time", gosx.Attrs(gosx.Attr("datetime", postDate(post).UTC().Format(time.RFC3339))), gosx.Text(formatPostDate(postDate(post))))}
	if author := strings.TrimSpace(post.Author); author != "" {
		parts = append(parts, gosx.Text(" · by "+author))
	}
	if len(post.Tags) > 0 {
		parts = append(parts, gosx.Text(" · in "))
		for index, tag := range post.Tags {
			if index > 0 {
				parts = append(parts, gosx.Text(", "))
			}
			parts = append(parts, gosx.El("a", gosx.Attrs(gosx.Attr("href", categoryPath(tag))), gosx.Text(tag)))
		}
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Fragment(parts...))
}

func renderPager(path string, page int, more bool) gosx.Node {
	links := make([]gosx.Node, 0, 2)
	if page > 1 {
		href := path
		if page > 2 {
			href = path + "?page=" + strconv.Itoa(page-1)
		}
		links = append(links, gosx.El("a", gosx.Attrs(gosx.Attr("href", href), gosx.Attr("rel", "prev")), gosx.Text("← Newer posts")))
	}
	if more {
		links = append(links, gosx.El("a", gosx.Attrs(gosx.Attr("href", path+"?page="+strconv.Itoa(page+1)), gosx.Attr("rel", "next")), gosx.Text("Older posts →")))
	}
	if len(links) == 0 {
		return gosx.Fragment()
	}
	return gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-pager"), gosx.Attr("aria-label", "More posts")), gosx.Fragment(links...))
}

func (h *Host) handleBlogPost(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	slug := strings.TrimSpace(r.PathValue("slug"))
	post, ok := h.livePostBySlug(slug)
	if !ok {
		if target, moved := h.resolveRedirect(postPath(slug)); moved {
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		h.servePublicNotFound(w, settings, strings.TrimPrefix(postPath(slug), "/"))
		return
	}

	brand := brandFromSettings(settings)
	meta := metaFromSettings(settings)
	meta.Title = post.Title
	meta.Description = firstNonEmpty(postExcerpt(post), settings.Description)
	meta.CanonicalPath = postPath(post.Slug)
	meta.Kind = "article"
	meta.Feed = true
	meta.ImageURL = firstNonEmpty(firstImageURL(post.Body), brand.LogoURL)
	meta.ImageAlt = firstNonEmpty(post.Title, settings.Title)
	meta.JSONLD = h.articleData(settings, brand, post, h.absoluteBase(r))
	meta.HeadCode = h.headCode()
	meta.Consent = h.consentRequired()

	var consent gosx.Node = gosx.Fragment()
	if meta.HeadCode != "" && meta.Consent {
		consent = renderConsentBanner()
	}
	body := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-shell")),
		h.renderPublicNav(settings, "blog"),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")),
			gosx.El("article", gosx.Attrs(gosx.Attr("class", "site-article site-post")),
				gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text(post.Title)),
				renderPostMeta(post),
				h.renderBody(post.Body, render.Hooks{Flow: h.flowHook(postPath(post.Slug), formStateFromQuery(r)), Image: h.imageHook()}),
				gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-post-nav"), gosx.Attr("aria-label", "Blog")),
					gosx.El("a", gosx.Attrs(gosx.Attr("href", blogPath)), gosx.Text("← All posts"))),
			),
		),
		h.renderPublicFooter(settings),
		consent,
	)
	h.writeDocument(w, http.StatusOK, meta, body)
}

// firstImageURL is the first picture in a document, for share cards.
func firstImageURL(doc blockstudio.Document) string {
	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue
		}
		switch instance.Key {
		case content.BlockImage:
			if url := strings.TrimSpace(instance.Values["url"].String); url != "" {
				return url
			}
		case content.BlockGallery:
			if images := galleryImages(instance); len(images) > 0 {
				return images[0][0]
			}
		}
	}
	return ""
}

// articleData is the BlogPosting structured data search engines read.
func (h *Host) articleData(settings cmsstore.SiteSettings, brand Brand, post cmsstore.Post, base string) []map[string]any {
	siteTitle := firstNonEmpty(settings.Title, h.opts.SiteTitle)
	publisher := map[string]any{"@type": "Organization", "name": siteTitle}
	if brand.LogoURL != "" {
		publisher["logo"] = map[string]any{"@type": "ImageObject", "url": absoluteURL(base, brand.LogoURL)}
	}
	article := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         post.Title,
		"url":              base + postPath(post.Slug),
		"mainEntityOfPage": base + postPath(post.Slug),
		"datePublished":    postDate(post).UTC().Format(time.RFC3339),
		"publisher":        publisher,
	}
	if !post.Updated.IsZero() {
		article["dateModified"] = post.Updated.UTC().Format(time.RFC3339)
	}
	if excerpt := postExcerpt(post); excerpt != "" {
		article["description"] = excerpt
	}
	if author := strings.TrimSpace(post.Author); author != "" {
		article["author"] = map[string]any{"@type": "Person", "name": author}
	} else {
		article["author"] = map[string]any{"@type": "Organization", "name": siteTitle}
	}
	if len(post.Tags) > 0 {
		article["keywords"] = strings.Join(post.Tags, ", ")
	}
	if image := firstImageURL(post.Body); image != "" {
		article["image"] = absoluteURL(base, image)
	}
	return []map[string]any{article}
}

// ---------- RSS ----------

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string      `xml:"title"`
	Link        string      `xml:"link"`
	Description string      `xml:"description"`
	AtomLink    rssAtomLink `xml:"atom:link"`
	Items       []rssItem   `xml:"item"`
}

type rssAtomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	GUID        rssGUID  `xml:"guid"`
	PubDate     string   `xml:"pubDate"`
	Description string   `xml:"description,omitempty"`
	Categories  []string `xml:"category,omitempty"`
}

type rssGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr"`
}

func (h *Host) handleFeed(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	base := h.absoluteBase(r)
	siteTitle := firstNonEmpty(settings.Title, h.opts.SiteTitle)
	feed := rssFeed{Version: "2.0", AtomNS: "http://www.w3.org/2005/Atom", Channel: rssChannel{
		Title:       siteTitle + " — " + h.blogTitle(),
		Link:        base + blogPath,
		Description: firstNonEmpty(settings.Description, "Posts from "+siteTitle+"."),
		AtomLink:    rssAtomLink{Href: base + feedPath, Rel: "self", Type: "application/rss+xml"},
	}}
	for _, post := range h.livePosts() {
		feed.Channel.Items = append(feed.Channel.Items, rssItem{
			Title:       post.Title,
			Link:        base + postPath(post.Slug),
			GUID:        rssGUID{Value: base + postPath(post.Slug), IsPermaLink: "true"},
			PubDate:     postDate(post).UTC().Format(time.RFC1123Z),
			Description: postExcerpt(post),
			Categories:  post.Tags,
		})
	}
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(feed)
}

// ---------- admin: the posts list ----------

func (h *Host) allPosts() []cmsstore.Post {
	posts, err := h.store.ListPosts(cmsstore.PostFilter{})
	if err != nil {
		return nil
	}
	sortPostsNewestFirst(posts)
	return posts
}

func (h *Host) handleAdminPosts(w http.ResponseWriter, r *http.Request) {
	h.renderAdminPosts(w, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderAdminPosts(w http.ResponseWriter, status adminStatus) {
	posts := h.allPosts()
	active := make([]cmsstore.Post, 0, len(posts))
	archived := make([]cmsstore.Post, 0, 2)
	for _, post := range posts {
		if PostArchived(post) {
			archived = append(archived, post)
		} else {
			active = append(active, post)
		}
	}

	var listing gosx.Node
	if len(active) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No posts yet")),
			gosx.El("p", nil, gosx.Text("A blog is the easiest way to keep your site fresh: news, tips, what's new this month. Your first post goes live at "+blogPath+" and adds a Blog link to your menu.")),
		)
	} else {
		rows := make([]gosx.Node, 0, len(active))
		for _, post := range active {
			rows = append(rows, h.renderPostRow(post))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your posts")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
				gosx.Text("Newest first, the way visitors see them at "+blogPath+".")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-table--pages")),
				gosx.El("thead", nil, gosx.El("tr", nil,
					gosx.El("th", nil, gosx.Text("Post")),
					gosx.El("th", nil, gosx.Text("Date")),
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
		for _, post := range archived {
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.Text(post.Title)),
				gosx.El("td", nil, gosx.Text(formatPostDate(postDate(post)))),
				gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", "archived")), gosx.Text("Archived"))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")), h.postActionButton(post.ID, "restore", "Restore")),
			))
		}
		archivedPanel = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Archived")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Not on your site, but nothing is deleted.")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-table--pages")),
				gosx.El("tbody", nil, gosx.Fragment(rows...))),
		)
	}

	create := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Write a post")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/posts")),
			h.csrfField(),
			adminTextField("title", "Title", "", "You can change it while you write."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Start writing")),
			),
		),
	)

	body := h.renderAdminShell("posts", "Blog",
		"Write posts, choose when they go live, and sort them into categories.",
		status, listing, create, archivedPanel)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Blog"), body)
}

// postStatus is the one-word state an owner sees in the list and the editor.
func (h *Host) postStatus(post cmsstore.Post) (state, label string) {
	switch {
	case PostOffline(post):
		return "offline", "Offline"
	case PostArchived(post):
		return "archived", "Archived"
	}
	if at, scheduled := postScheduled(post); scheduled {
		return "scheduled", "Scheduled for " + formatPostDate(at)
	}
	if _, live := h.livePost(post); live {
		return "published", "Live"
	}
	return "draft", "Not published"
}

func (h *Host) renderPostRow(post cmsstore.Post) gosx.Node {
	state, label := h.postStatus(post)
	actions := []gosx.Node{}
	if PostOffline(post) {
		actions = append(actions, h.postActionButton(post.ID, "online", "Put back online"))
	} else if state == "published" || state == "scheduled" {
		actions = append(actions, h.postActionButton(post.ID, "offline", "Take offline"))
	}
	actions = append(actions, h.postActionButton(post.ID, "archive", "Archive"))

	return gosx.El("tr", nil,
		gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/edit/post/"+post.ID)), gosx.Text(post.Title))),
		gosx.El("td", nil, gosx.Text(formatPostDate(postDate(post)))),
		gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label))),
		gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")), gosx.Fragment(actions...)),
	)
}

func (h *Host) postActionButton(id, action, label string) gosx.Node {
	return gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/posts/"+id+"/action"), gosx.Attr("class", "admin-inline-form")),
		h.csrfField(),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "action"), gosx.Attr("value", action))),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", action)), gosx.Text(label)),
	)
}

func (h *Host) handleAdminCreatePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAdminPosts(w, adminStatus{Message: "We couldn't read that form. Try again.", Error: true})
		return
	}
	title := strings.TrimSpace(r.PostFormValue("title"))
	if title == "" {
		h.renderAdminPosts(w, adminStatus{Message: "Give the post a title before you start.", Error: true})
		return
	}
	slug := h.freePostSlug(normalizeSlug(title), "")
	if slug == "" {
		slug = h.freePostSlug("post", "")
	}
	post, err := h.store.CreatePost(cmsstore.PostInput{
		Slug:  slug,
		Title: title,
		Body:  document(block(0, content.BlockParagraph, values("text", "Start writing your post here."))),
	})
	if err != nil {
		h.renderAdminPosts(w, adminStatus{Message: "We couldn't create that post. Try again.", Error: true})
		return
	}
	http.Redirect(w, r, "/admin/edit/post/"+post.ID, http.StatusSeeOther)
}

// freePostSlug returns slug, or slug-2, slug-3… until one is unused by any
// post other than exceptID.
func (h *Host) freePostSlug(slug, exceptID string) string {
	if slug == "" {
		return ""
	}
	candidate := slug
	for n := 2; n < 1000; n++ {
		other, exists, _ := h.store.PostBySlug(candidate)
		if !exists || other.ID == exceptID {
			return candidate
		}
		candidate = slug + "-" + strconv.Itoa(n)
	}
	return ""
}

func (h *Host) handleAdminPostAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/admin/posts?status="+queryEscape("We couldn't read that. Try again."), http.StatusSeeOther)
		return
	}
	message, err := h.postAction(r.PathValue("id"), strings.TrimSpace(r.PostFormValue("action")))
	if err != nil {
		if errors.Is(err, errPostNotFound) {
			h.writeAdminNotFound(w, "post")
			return
		}
		http.Redirect(w, r, "/admin/posts?status="+queryEscape("That didn't work. Try again."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/posts?status="+queryEscape(message), http.StatusSeeOther)
}

func (h *Host) postAction(id, action string) (string, error) {
	switch action {
	case "offline":
		post, err := h.setPostFlag(id, pageOfflineKey, true)
		return "“" + post.Title + "” is off your site. Put it back online whenever you like.", err
	case "online":
		post, err := h.setPostFlag(id, pageOfflineKey, false)
		return "“" + post.Title + "” is back online.", err
	case "archive":
		post, err := h.setPostFlag(id, pageArchivedKey, true)
		return "“" + post.Title + "” is archived. Restore it from the list below whenever you like.", err
	case "restore":
		post, err := h.setPostFlag(id, pageArchivedKey, false)
		return "“" + post.Title + "” is back.", err
	}
	return "", errors.New("unknown action")
}

// ---------- admin: the post editor ----------

func (h *Host) postSubject(post cmsstore.Post) editorSubject {
	_, live := h.livePost(post)
	subject := editorSubject{
		Kind:       "post",
		Noun:       "post",
		ID:         post.ID,
		Title:      post.Title,
		Slug:       post.Slug,
		Body:       post.Body,
		Live:       live,
		BackHref:   "/admin/posts",
		BackLabel:  "← Blog",
		ViewHref:   postPath(post.Slug),
		SaveURL:    "/admin/api/posts/" + post.ID,
		PublishURL: "/admin/api/posts/" + post.ID + "/publish",
		Post:       &post,
	}
	if at, scheduled := postScheduled(post); scheduled {
		subject.Scheduled = at
	}
	subject.Checks = h.readinessChecks("post", "", post.Body)
	return subject
}

func (h *Host) handlePostEditor(w http.ResponseWriter, r *http.Request) {
	post, ok, err := h.store.PostByID(r.PathValue("id"))
	if err != nil || !ok {
		h.writeAdminNotFound(w, "post")
		return
	}
	h.renderEditor(w, h.postSubject(post))
}

type postSavePayload struct {
	editorSavePayload
	Excerpt string `json:"excerpt"`
	Author  string `json:"author"`
	Tags    string `json:"tags"`
}

func splitTags(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (h *Host) handlePostSave(w http.ResponseWriter, r *http.Request) {
	post, ok, err := h.store.PostByID(r.PathValue("id"))
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, editorSaveResult{Message: "We couldn't find that post."})
		return
	}
	var payload postSavePayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, editorSaveResult{Message: "We couldn't read that change. Try again."})
		return
	}
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "Give the post a title."})
		return
	}
	slug := normalizeSlug(firstNonEmpty(payload.Slug, title))
	if slug == "" {
		slug = post.Slug
	}
	if other, exists, _ := h.store.PostBySlug(slug); exists && other.ID != post.ID {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "Another post already uses " + postPath(slug) + "."})
		return
	}

	metadata := cloneMetadata(post.Metadata)
	if message := applyPublishAt(metadata, payload.PublishAt); message != "" {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: message})
		return
	}

	input := cmsstore.PostInput{
		Slug:     slug,
		Title:    title,
		Excerpt:  strings.TrimSpace(payload.Excerpt),
		Author:   strings.TrimSpace(payload.Author),
		Tags:     splitTags(payload.Tags),
		Body:     payloadDocument(payload.Blocks),
		Metadata: metadata,
		State:    post.State,
	}
	if _, _, err := h.store.PreviewPost(post.ID, input); err != nil {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "We couldn't save that. Try again."})
		return
	}
	if slug != post.Slug {
		_ = h.recordRedirect(postPath(post.Slug), postPath(slug))
	}
	updated, _, _ := h.store.PostByID(post.ID)
	_, live := h.livePost(updated)
	writeJSON(w, http.StatusOK, editorSaveResult{OK: true, Slug: slug, Live: live, Checks: h.readinessChecks("post", "", updated.Body)})
}

func (h *Host) handlePostPublish(w http.ResponseWriter, r *http.Request) {
	post, ok, err := h.store.PostByID(r.PathValue("id"))
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, editorSaveResult{Message: "We couldn't find that post."})
		return
	}
	result := h.publishPost(post)
	result.Checks = h.readinessChecks("post", "", post.Body)
	writeJSON(w, http.StatusOK, result)
}
