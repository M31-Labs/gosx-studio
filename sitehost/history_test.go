package sitehost

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestVersionHistoryPreviewAndRestore(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"What we bake.","blocks":[{"kind":"paragraph","text":"First version"}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"What we bake.","blocks":[{"kind":"paragraph","text":"Second version, still a draft"}]}`)

	mustContain(t, get(t, handler, "/admin/edit/"+id).Body.String(), `href="/admin/history/page/`+id+`"`, "the editor links to history")
	history := get(t, handler, "/admin/history/page/"+id).Body.String()
	mustContain(t, history, "Versions of “Menu”", "history names the page")
	mustContain(t, history, ">Published<", "publishes are listed")
	mustContain(t, history, ">Saved<", "saves are listed")
	mustContain(t, history, ">current<", "the newest is marked current")

	// Find the published revision and look at it.
	revisions := host.revisionsNewestFirst(historySubject{Kind: "page", ResourceKind: "page", ID: id})
	var publishedID string
	for _, revision := range revisions {
		if strings.HasSuffix(revision.Action, ".published") {
			publishedID = revision.ID
			break
		}
	}
	if publishedID == "" {
		t.Fatal("no published revision found")
	}
	preview := get(t, handler, "/admin/history/page/"+id+"/"+publishedID)
	mustContain(t, preview.Body.String(), "First version", "the preview shows that version's text")
	mustContain(t, preview.Body.String(), "This is how the page looked then", "with a banner")
	mustContain(t, preview.Body.String(), `content="noindex, nofollow"`, "and is not for search engines")
	if strings.Contains(preview.Body.String(), statsScriptPath) {
		t.Fatal("a preview is not a visit")
	}

	// Restore: the draft goes back, the live page is untouched until publish.
	rec := post(t, handler, "/admin/history/page/"+id+"/"+publishedID+"/restore", url.Values{})
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "Restored+the+version") {
		t.Fatalf("restore = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	mustContain(t, get(t, handler, "/admin/edit/"+id).Body.String(), "First version", "the editor now holds the restored version")
	mustContain(t, get(t, handler, "/menu").Body.String(), "First version", "visitors still see the published one")
	mustContain(t, get(t, handler, "/admin/history/page/"+id).Body.String(), "Restored an earlier version", "the restore is itself a version")

	if code := get(t, handler, "/admin/history/page/nope").Code; code != http.StatusNotFound {
		t.Fatalf("unknown page history = %d", code)
	}
}

func TestScheduledPageChangeKeepsTheOldVersionLiveUntilItsTime(t *testing.T) {
	base := time.Date(2026, 9, 12, 9, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return base }
	t.Cleanup(func() { timeNow = time.Now })
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")

	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"paragraph","text":"Summer menu"}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	mustContain(t, get(t, handler, "/menu").Body.String(), "Summer menu", "the summer menu is live")

	at := base.Add(48 * time.Hour).Format(time.RFC3339)
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","publishAt":"`+at+`","blocks":[{"kind":"paragraph","text":"Autumn menu"}]}`)
	body := post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{}).Body.String()
	mustContain(t, body, `"live":false`, "publishing with a future date does not go live")
	mustContain(t, body, "Scheduled — it goes live on 14 September 2026", "the owner is told when")
	mustContain(t, get(t, handler, "/menu").Body.String(), "Summer menu", "the summer menu stays live meanwhile")
	mustContain(t, get(t, handler, "/admin/pages").Body.String(), "Scheduled for 14 September 2026", "the pages list shows it")
	mustContain(t, get(t, handler, "/admin/edit/"+id).Body.String(), "Scheduled for 14 September 2026", "so does the editor chip")

	timeNow = func() time.Time { return base.Add(49 * time.Hour) }
	mustContain(t, get(t, handler, "/menu").Body.String(), "Autumn menu", "at its time the change goes live on the next visit")
	page, _, _ := host.Store().PageByID(id)
	if page.Metadata[publishPendingKey] != "" {
		t.Fatal("the pending flag is cleared once published")
	}
	mustContain(t, get(t, handler, "/admin/pages").Body.String(), ">Live<", "and the list says live")

	// Clearing the date cancels a pending schedule.
	later := base.Add(96 * time.Hour).Format(time.RFC3339)
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","publishAt":"`+later+`","blocks":[{"kind":"paragraph","text":"Winter menu"}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","publishAt":"","blocks":[{"kind":"paragraph","text":"Winter menu"}]}`)
	page, _, _ = host.Store().PageByID(id)
	if page.Metadata[publishPendingKey] != "" || page.Metadata[publishAtKey] != "" {
		t.Fatalf("clearing the date must cancel the schedule: %+v", page.Metadata)
	}
}

func TestReadinessChecksBeforePublish(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	body := postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"","blocks":[
		{"kind":"image","url":"/uploads/x.png","alt":""},
		{"kind":"gallery","images":[{"url":"/uploads/a.png","alt":""},{"url":"/uploads/b.png","alt":"fine"}]},
		{"kind":"button","text":"Book","url":"/bookings"},
		{"kind":"paragraph","text":"See [our story](/about) and [the menu](/menu) and [the blog](/blog)."}
	]}`).Body.String()
	mustContain(t, body, "No description for search results yet", "a missing description is flagged")
	mustContain(t, body, "2 pictures have no description", "pictures without alt text are counted")
	mustContain(t, body, "A link points at /bookings, which doesn't exist", "a broken button is flagged")
	mustContain(t, body, "A link points at /about, which doesn't exist", "a broken inline link is flagged")
	if strings.Contains(body, "/menu, which") || strings.Contains(body, "/blog, which") {
		t.Fatal("links to real pages must not be flagged")
	}
	mustContain(t, get(t, handler, "/admin/edit/"+id).Body.String(), `data-checks-list="true"`, "the editor shows the list")
	mustContain(t, get(t, handler, "/admin/edit/"+id).Body.String(), "2 pictures have no description", "with the current findings")

	body = postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"Our menu.","blocks":[{"kind":"paragraph","text":"All good"}]}`).Body.String()
	mustContain(t, body, `"checks":[]`, "a clean page has nothing to fix")
	mustContain(t, postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"Our menu.","blocks":[]}`).Body.String(), "The page is empty", "an empty page is flagged")
}

func TestBlocksCanBeHiddenOnPhones(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"paragraph","text":"Everyone sees this"},
		{"kind":"paragraph","text":"Only on big screens","phone":"hide"}
	]}`)
	page, _, _ := host.Store().PageByID(id)
	if page.Body.Blocks[1].Values[phoneKey].String != phoneHide || page.Body.Blocks[0].Values[phoneKey].String != "" {
		t.Fatalf("phone flag stored wrong: %+v", page.Body.Blocks)
	}
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `data-phone="hide"`, "the canvas marks the block")
	mustContain(t, editor, `data-tool="phone" title="Show on phones again" aria-label="Show on phones again" aria-pressed="true"`, "its tool reads as pressed")
	mustContain(t, editor, `data-device="phone"`, "the toolbar has a phone preview")
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	live := get(t, handler, "/menu").Body.String()
	mustContain(t, live, `<div class="site-no-phone"><p>Only on big screens</p></div>`, "the hidden block is wrapped for visitors")
	if strings.Contains(live, `<div class="site-no-phone"><p>Everyone`) {
		t.Fatal("a normal block must not be wrapped")
	}
	css := get(t, handler, publicStylesheetPath).Body.String()
	mustContain(t, css, "@container site (max-width: 640px)", "phone rules follow the site's own width")
	mustContain(t, css, ".site-no-phone { display: none; }", "and hide the block there")
}
