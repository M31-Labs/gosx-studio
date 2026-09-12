package sitehost

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// createPost goes through the admin form, the way an owner does, and returns
// the new post's ID from the editor redirect.
func createPost(t *testing.T, handler http.Handler, title string) string {
	t.Helper()
	rec := post(t, handler, "/admin/posts", url.Values{"title": {title}})
	location := rec.Header().Get("Location")
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(location, "/admin/edit/post/") {
		t.Fatalf("create post = %d %q", rec.Code, location)
	}
	return strings.TrimPrefix(location, "/admin/edit/post/")
}

func savePost(t *testing.T, handler http.Handler, id, payload string) string {
	t.Helper()
	rec := postJSON(t, handler, "/admin/api/posts/"+id, payload)
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("save post: %s", rec.Body.String())
	}
	return rec.Body.String()
}

func publishPost(t *testing.T, handler http.Handler, id string) string {
	t.Helper()
	rec := post(t, handler, "/admin/api/posts/"+id+"/publish", url.Values{})
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("publish post: %s", rec.Body.String())
	}
	return rec.Body.String()
}

func TestBlogFromFirstPostToFeed(t *testing.T) {
	_, handler := newTestHost(t)

	// Before any post: no blog in the menu, nothing at /blog.
	if strings.Contains(siteNav(t, handler), `href="/blog"`) {
		t.Fatal("the menu must not show a Blog link before a post is live")
	}
	if code := get(t, handler, "/blog").Code; code != http.StatusNotFound {
		t.Fatalf("/blog with no posts = %d, want 404", code)
	}
	mustContain(t, get(t, handler, "/admin/posts").Body.String(), "No posts yet", "the posts list starts empty")

	id := createPost(t, handler, "Spring menu")
	editor := get(t, handler, "/admin/edit/post/"+id).Body.String()
	mustContain(t, editor, `data-save-url="/admin/api/posts/`+id+`"`, "the post editor saves to the posts API")
	mustContain(t, editor, `data-publish-url="/admin/api/posts/`+id+`/publish"`, "and publishes there")
	mustContain(t, editor, "This post", "the sidebar is about the post")
	mustContain(t, editor, `id="pageTags"`, "categories are editable")
	mustContain(t, editor, `id="pagePublishAt"`, "the publish date is editable")
	mustContain(t, editor, `type="datetime-local"`, "as a date picker")
	mustContain(t, editor, "← Blog", "back goes to the blog list")
	mustContain(t, editor, `class="site-post-meta"`, "the canvas previews the meta line")

	savePost(t, handler, id, `{"title":"Spring menu","slug":"spring-menu","excerpt":"","tags":"News, Menu","author":"Ana","publishAt":"","blocks":[
		{"kind":"paragraph","text":"Asparagus is **back** and so are we."},
		{"kind":"paragraph","text":"More soon."}
	]}`)

	// Saved, not published: still invisible.
	if code := get(t, handler, "/blog/spring-menu").Code; code != http.StatusNotFound {
		t.Fatalf("draft post = %d, want 404", code)
	}
	mustContain(t, get(t, handler, "/admin/posts").Body.String(), "Not published", "the list shows it as a draft")

	body := publishPost(t, handler, id)
	mustContain(t, body, `"live":true`, "publishing makes it live")
	mustContain(t, body, "your post is live", "with a message for the owner")

	// The index.
	index := get(t, handler, "/blog")
	if index.Code != http.StatusOK {
		t.Fatalf("/blog = %d", index.Code)
	}
	mustContain(t, index.Body.String(), `<a href="/blog/spring-menu">Spring menu</a>`, "the index links the post")
	mustContain(t, index.Body.String(), "Asparagus is back and so are we", "the excerpt comes from the first paragraph, markers stripped")
	mustContain(t, index.Body.String(), `href="/blog/category/news"`, "the category nav lists News")
	mustContain(t, index.Body.String(), "News (1)", "with a count")
	mustContain(t, index.Body.String(), `rel="alternate" type="application/rss+xml"`, "the feed is advertised")

	// The post.
	page := get(t, handler, "/blog/spring-menu").Body.String()
	mustContain(t, page, "Asparagus is <strong>back</strong>", "the body renders with formatting")
	mustContain(t, page, "by Ana", "the author shows")
	mustContain(t, page, `in <a href="/blog/category/news">News</a>, <a href="/blog/category/menu">Menu</a>`, "categories link")
	mustContain(t, page, `"@type":"BlogPosting"`, "structured data describes the post")
	mustContain(t, page, `"author":{"@type":"Person","name":"Ana"}`, "with its author")
	mustContain(t, page, `<link rel="canonical" href="https://wildflower.example/blog/spring-menu"`, "canonical points at the post")
	mustContain(t, page, `href="/blog">← All posts`, "the post links back")

	// The menu, the category page, the feed, the sitemap.
	mustContain(t, siteNav(t, handler), `href="/blog">Blog</a>`, "a live post puts Blog in the menu")
	mustContain(t, get(t, handler, "/blog/category/menu").Body.String(), "Posts about Menu", "category pages filter")
	if code := get(t, handler, "/blog/category/nothing").Code; code != http.StatusNotFound {
		t.Fatalf("unknown category = %d", code)
	}
	feed := get(t, handler, "/feed.xml")
	if feed.Code != http.StatusOK || !strings.HasPrefix(feed.Header().Get("Content-Type"), "application/rss+xml") {
		t.Fatalf("feed = %d %s", feed.Code, feed.Header().Get("Content-Type"))
	}
	mustContain(t, feed.Body.String(), `<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">`, "the feed is RSS 2.0")
	mustContain(t, feed.Body.String(), `<link>https://wildflower.example/blog/spring-menu</link>`, "with an absolute link")
	mustContain(t, feed.Body.String(), `<category>News</category>`, "and categories")
	mustContain(t, feed.Body.String(), `<atom:link href="https://wildflower.example/feed.xml" rel="self"`, "and a self link")
	sitemap := get(t, handler, "/sitemap.xml").Body.String()
	mustContain(t, sitemap, "<loc>https://wildflower.example/blog</loc>", "the sitemap lists the blog")
	mustContain(t, sitemap, "<loc>https://wildflower.example/blog/spring-menu</loc>", "and the post")

	// Editing after publishing keeps the published version live.
	savePost(t, handler, id, `{"title":"Spring menu","slug":"spring-menu","tags":"News","blocks":[{"kind":"paragraph","text":"Half-written edit"}]}`)
	mustContain(t, get(t, handler, "/blog/spring-menu").Body.String(), "Asparagus", "a draft edit leaves the live post alone")
	mustContain(t, get(t, handler, "/admin/posts").Body.String(), ">Live<", "and the list still says live")
}

func TestScheduledPostGoesLiveOnItsDate(t *testing.T) {
	base := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })

	_, handler := newTestHost(t)
	id := createPost(t, handler, "Autumn hours")
	at := base.Add(2 * time.Hour).Format(time.RFC3339)
	savePost(t, handler, id, `{"title":"Autumn hours","slug":"autumn-hours","publishAt":"`+at+`","blocks":[{"kind":"paragraph","text":"We open later."}]}`)
	body := publishPost(t, handler, id)
	mustContain(t, body, `"live":false`, "a scheduled post is not live yet")
	mustContain(t, body, "Scheduled — it goes live on", "and the owner is told when")

	if code := get(t, handler, "/blog/autumn-hours").Code; code != http.StatusNotFound {
		t.Fatalf("scheduled post = %d, want 404", code)
	}
	if strings.Contains(get(t, handler, "/feed.xml").Body.String(), "Autumn hours") {
		t.Fatal("a scheduled post must not be in the feed")
	}
	mustContain(t, get(t, handler, "/admin/posts").Body.String(), "Scheduled for 12 September 2026", "the list shows the schedule")
	mustContain(t, get(t, handler, "/admin/edit/post/"+id).Body.String(), "Scheduled for 12 September 2026", "so does the editor")

	timeNow = func() time.Time { return base.Add(3 * time.Hour) }
	if code := get(t, handler, "/blog/autumn-hours").Code; code != http.StatusOK {
		t.Fatalf("post after its date = %d, want 200", code)
	}
	mustContain(t, get(t, handler, "/blog").Body.String(), "12 September 2026", "the index shows the chosen date")
	mustContain(t, get(t, handler, "/feed.xml").Body.String(), "Autumn hours", "and the feed carries it")
}

func TestRenamedPostKeepsItsOldAddress(t *testing.T) {
	_, handler := newTestHost(t)
	id := createPost(t, handler, "Old name")
	savePost(t, handler, id, `{"title":"Old name","slug":"old-name","blocks":[{"kind":"paragraph","text":"Hello"}]}`)
	publishPost(t, handler, id)
	savePost(t, handler, id, `{"title":"New name","slug":"new-name","blocks":[{"kind":"paragraph","text":"Hello"}]}`)
	publishPost(t, handler, id)

	rec := get(t, handler, "/blog/old-name")
	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "/blog/new-name" {
		t.Fatalf("old address = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if code := get(t, handler, "/blog/new-name").Code; code != http.StatusOK {
		t.Fatalf("new address = %d", code)
	}
}

func TestPostActionsAndReservedSlugs(t *testing.T) {
	_, handler := newTestHost(t)
	id := createPost(t, handler, "Notes")
	savePost(t, handler, id, `{"title":"Notes","slug":"notes","blocks":[{"kind":"paragraph","text":"Hi"}]}`)
	publishPost(t, handler, id)

	post(t, handler, "/admin/posts/"+id+"/action", url.Values{"action": {"offline"}})
	if code := get(t, handler, "/blog/notes").Code; code != http.StatusNotFound {
		t.Fatal("an offline post must not be served")
	}
	mustContain(t, get(t, handler, "/admin/posts").Body.String(), ">Offline<", "the list says offline")
	post(t, handler, "/admin/posts/"+id+"/action", url.Values{"action": {"online"}})
	if code := get(t, handler, "/blog/notes").Code; code != http.StatusOK {
		t.Fatal("putting a post back online must serve it again")
	}
	post(t, handler, "/admin/posts/"+id+"/action", url.Values{"action": {"archive"}})
	list := get(t, handler, "/admin/posts").Body.String()
	mustContain(t, list, ">Archived<", "the list shows the archive")
	mustContain(t, list, "No posts yet", "and the active list is empty")
	if code := get(t, handler, "/blog/notes").Code; code != http.StatusNotFound {
		t.Fatal("an archived post must not be served")
	}
	post(t, handler, "/admin/posts/"+id+"/action", url.Values{"action": {"restore"}})
	if code := get(t, handler, "/blog/notes").Code; code != http.StatusOK {
		t.Fatal("a restored post is served again")
	}

	// A second post with the same title gets its own address.
	second := createPost(t, handler, "Notes")
	if second == id {
		t.Fatal("second post reused the first id")
	}
	mustContain(t, get(t, handler, "/admin/edit/post/"+second).Body.String(), `value="notes-2"`, "the slug is made unique")

	// Pages cannot take the blog's address.
	rec := post(t, handler, "/admin/pages", url.Values{"title": {"Blog"}, "slug": {"blog"}})
	mustContain(t, rec.Body.String(), "reserved", "a page cannot be called /blog")
}
