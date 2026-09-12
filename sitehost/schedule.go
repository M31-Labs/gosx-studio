package sitehost

import (
	"strings"
	"sync"
	"time"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// schedule.go is "publish at": the owner picks a date, presses Publish, and
// the change goes live then — not before.
//
// One rule for pages and posts alike. Pressing Publish with a future date
// marks the draft as pending instead of publishing it, so whatever was live
// stays live. When the date arrives the pending draft is published exactly
// as if the owner had pressed the button: a real publish, a real revision in
// the ledger. Nothing runs in the background; the check happens on the next
// visit, which is the only moment it could matter.

const (
	publishAtKey      = "publishAt"      // RFC 3339, UTC
	publishPendingKey = "publishPending" // "true" while a scheduled publish waits
	dueCheckInterval  = 10 * time.Second
)

func publishAtOf(metadata cmsstore.Metadata) (time.Time, bool) {
	raw := strings.TrimSpace(metadata[publishAtKey])
	if raw == "" {
		return time.Time{}, false
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return at, true
}

// scheduledFor reports a pending publish whose date has not arrived.
func scheduledFor(metadata cmsstore.Metadata) (time.Time, bool) {
	if metadata[publishPendingKey] != "true" {
		return time.Time{}, false
	}
	at, ok := publishAtOf(metadata)
	if !ok || !at.After(timeNow()) {
		return time.Time{}, false
	}
	return at, true
}

// PagePublishAt is the date the owner chose for a page, if any.
func PagePublishAt(page cmsstore.Page) (time.Time, bool) { return publishAtOf(page.Metadata) }

// PostPublishAt is the date the owner chose for a post, if any.
func PostPublishAt(post cmsstore.Post) (time.Time, bool) { return publishAtOf(post.Metadata) }

// publishPage is the one way a page gets published: now, or pending until
// its date. It also lifts "offline", because Publish means "show this".
func (h *Host) publishPage(page cmsstore.Page) editorSaveResult {
	if PageOffline(page) {
		updated, err := h.setPageFlag(page.ID, pageOfflineKey, false)
		if err != nil {
			return editorSaveResult{Message: "We couldn't publish that. Try again."}
		}
		page = updated
	}
	if at, ok := publishAtOf(page.Metadata); ok && at.After(timeNow()) {
		if _, err := h.setPageFlag(page.ID, publishPendingKey, true); err != nil {
			return editorSaveResult{Message: "We couldn't schedule that. Try again."}
		}
		return editorSaveResult{OK: true, Slug: page.Slug, Chip: "Scheduled", Message: "Scheduled — it goes live on " + formatPostDate(at)}
	}
	if page.Metadata[publishPendingKey] == "true" {
		if _, err := h.setPageFlag(page.ID, publishPendingKey, false); err != nil {
			return editorSaveResult{Message: "We couldn't publish that. Try again."}
		}
	}
	if _, _, err := h.store.PublishPage(page.ID); err != nil {
		return editorSaveResult{Message: "We couldn't publish that. Try again."}
	}
	return editorSaveResult{OK: true, Live: true, Slug: page.Slug, Chip: "Live", Message: "Published — your page is live"}
}

func (h *Host) publishPost(post cmsstore.Post) editorSaveResult {
	if PostOffline(post) {
		updated, err := h.setPostFlag(post.ID, pageOfflineKey, false)
		if err != nil {
			return editorSaveResult{Message: "We couldn't publish that. Try again."}
		}
		post = updated
	}
	if at, ok := publishAtOf(post.Metadata); ok && at.After(timeNow()) {
		if _, err := h.setPostFlag(post.ID, publishPendingKey, true); err != nil {
			return editorSaveResult{Message: "We couldn't schedule that. Try again."}
		}
		return editorSaveResult{OK: true, Slug: post.Slug, Chip: "Scheduled", Message: "Scheduled — it goes live on " + formatPostDate(at)}
	}
	if post.Metadata[publishPendingKey] == "true" {
		if _, err := h.setPostFlag(post.ID, publishPendingKey, false); err != nil {
			return editorSaveResult{Message: "We couldn't publish that. Try again."}
		}
	}
	if _, _, err := h.store.PublishPost(post.ID); err != nil {
		return editorSaveResult{Message: "We couldn't publish that. Try again."}
	}
	return editorSaveResult{OK: true, Live: true, Slug: post.Slug, Chip: "Live", Message: "Published — your post is live"}
}

type dueChecker struct {
	mu      sync.Mutex
	lastRun time.Time
}

// PublishDue publishes every pending page and post whose date has passed.
// It is called on the way into every public render and throttled, so a
// scheduled change appears within seconds of its time without a scheduler.
func (h *Host) PublishDue() {
	h.due.mu.Lock()
	now := timeNow()
	if !h.due.lastRun.IsZero() && now.Sub(h.due.lastRun) < dueCheckInterval && now.After(h.due.lastRun) {
		h.due.mu.Unlock()
		return
	}
	h.due.lastRun = now
	h.due.mu.Unlock()

	if pages, err := h.store.ListPages(cmsstore.PageFilter{}); err == nil {
		for _, page := range pages {
			if page.Metadata[publishPendingKey] != "true" {
				continue
			}
			if at, ok := publishAtOf(page.Metadata); ok && !at.After(now) {
				if _, err := h.setPageFlag(page.ID, publishPendingKey, false); err == nil {
					_, _, _ = h.store.PublishPage(page.ID)
				}
			}
		}
	}
	if posts, err := h.store.ListPosts(cmsstore.PostFilter{}); err == nil {
		for _, post := range posts {
			if post.Metadata[publishPendingKey] != "true" {
				continue
			}
			if at, ok := publishAtOf(post.Metadata); ok && !at.After(now) {
				if _, err := h.setPostFlag(post.ID, publishPendingKey, false); err == nil {
					_, _, _ = h.store.PublishPost(post.ID)
				}
			}
		}
	}
}
