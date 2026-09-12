package sitehost

import (
	"errors"
	"strings"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// pages.go is page management: taking a page offline, archiving it, keeping
// it out of the menu, and choosing the menu order.
//
// The content store knows two states, draft and published, and has no delete.
// Rather than teach it new states — and every host that already depends on
// it — the default host keeps these as flags in the page's own metadata and
// the menu order in the site settings. Nothing is ever destroyed: an archived
// page is one click from coming back, with its history intact.

const (
	pageOfflineKey   = "offline"
	pageArchivedKey  = "archived"
	pageNavHiddenKey = "navHidden"
	navOrderKey      = "navOrder"
)

func pageFlag(page cmsstore.Page, key string) bool {
	return strings.TrimSpace(page.Metadata[key]) == "true"
}

// PageOffline reports a page the owner has taken off the public site.
func PageOffline(page cmsstore.Page) bool { return pageFlag(page, pageOfflineKey) }

// PageArchived reports a page the owner has put away.
func PageArchived(page cmsstore.Page) bool { return pageFlag(page, pageArchivedKey) }

// PageNavHidden reports a page kept out of the site menu.
func PageNavHidden(page cmsstore.Page) bool { return pageFlag(page, pageNavHiddenKey) }

// setPageFlag rewrites one metadata flag through UpdatePage, which changes
// nothing about the page's draft/publish lifecycle.
func (h *Host) setPageFlag(id, key string, on bool) (cmsstore.Page, error) {
	page, ok, err := h.store.PageByID(id)
	if err != nil {
		return cmsstore.Page{}, err
	}
	if !ok {
		return cmsstore.Page{}, errPageNotFound
	}
	metadata := cmsstore.Metadata{}
	for k, v := range page.Metadata {
		metadata[k] = v
	}
	if on {
		metadata[key] = "true"
	} else {
		delete(metadata, key)
	}
	return h.store.UpdatePage(page.ID, cmsstore.PageInput{
		Slug:        page.Slug,
		Title:       page.Title,
		Description: page.Description,
		Body:        page.Body,
		State:       page.State,
		Metadata:    metadata,
	})
}

var errPageNotFound = errors.New("page not found")

// ---------- menu order ----------

// navOrder is the owner's chosen order of page IDs, from the site settings.
func (h *Host) navOrder() []string {
	raw := strings.TrimSpace(h.settings().Metadata[navOrderKey])
	if raw == "" {
		return nil
	}
	out := make([]string, 0, 8)
	for _, id := range strings.Split(raw, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}

// orderPages sorts pages for the menu and the admin list: home first, then
// the owner's chosen order, then anything not yet ordered in creation order.
func (h *Host) orderPages(pages []cmsstore.Page) []cmsstore.Page {
	rank := map[string]int{}
	for index, id := range h.navOrder() {
		rank[id] = index
	}
	out := append([]cmsstore.Page(nil), pages...)
	key := func(page cmsstore.Page) (int, int) {
		if page.Slug == homeSlug {
			return 0, 0
		}
		if r, ok := rank[page.ID]; ok {
			return 1, r
		}
		return 2, 0
	}
	// Insertion sort keeps it stable and the lists are small.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			ga, ra := key(out[j-1])
			gb, rb := key(out[j])
			if ga > gb || (ga == gb && ra > rb) {
				out[j-1], out[j] = out[j], out[j-1]
			} else {
				break
			}
		}
	}
	return out
}

// movePage shifts one page a step earlier or later in the menu.
func (h *Host) movePage(id string, delta int) error {
	pages, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return err
	}
	for _, page := range pages {
		if page.ID == id && page.Slug == homeSlug {
			return nil // home is always first; asking to move it is a no-op
		}
	}
	ordered := h.orderPages(pages)
	ids := make([]string, 0, len(ordered))
	for _, page := range ordered {
		if page.Slug == homeSlug {
			continue // home is always first and never moves
		}
		ids = append(ids, page.ID)
	}
	index := -1
	for i, candidate := range ids {
		if candidate == id {
			index = i
		}
	}
	if index < 0 {
		return errPageNotFound
	}
	target := index + delta
	if target < 0 || target >= len(ids) {
		return nil // already at the edge; nothing to do
	}
	ids[index], ids[target] = ids[target], ids[index]
	return h.updateSettingsMetadata(func(metadata cmsstore.Metadata) {
		metadata[navOrderKey] = strings.Join(ids, ",")
	})
}

// updateSettingsMetadata rewrites the site settings' metadata in place and
// publishes them, preserving every other field. SaveTheme uses the same shape.
func (h *Host) updateSettingsMetadata(mutate func(cmsstore.Metadata)) error {
	current := h.settings()
	metadata := cmsstore.Metadata{}
	for key, value := range current.Metadata {
		metadata[key] = value
	}
	mutate(metadata)
	input := cmsstore.SiteSettingsInput{
		Title:       current.Title,
		Description: current.Description,
		BaseURL:     current.BaseURL,
		Locale:      firstNonEmpty(current.Locale, "en"),
		Metadata:    metadata,
	}
	if _, err := h.store.SaveSiteSettings(input); err != nil {
		return err
	}
	_, _, err := h.store.PublishSiteSettings()
	return err
}

// ---------- actions ----------

// pageAction applies one management verb and returns the message the owner
// should read afterwards.
func (h *Host) pageAction(id, action string) (string, error) {
	switch action {
	case "offline":
		page, err := h.setPageFlag(id, pageOfflineKey, true)
		return "\"" + page.Title + "\" is offline. Visitors will see \"page not found\" until you put it back.", err
	case "online":
		page, err := h.setPageFlag(id, pageOfflineKey, false)
		return "\"" + page.Title + "\" is back online.", err
	case "archive":
		if _, err := h.setPageFlag(id, pageArchivedKey, true); err != nil {
			return "", err
		}
		page, err := h.setPageFlag(id, pageOfflineKey, true)
		return "\"" + page.Title + "\" is archived. Nothing is deleted — restore it any time.", err
	case "restore":
		if _, err := h.setPageFlag(id, pageArchivedKey, false); err != nil {
			return "", err
		}
		page, err := h.setPageFlag(id, pageOfflineKey, false)
		return "\"" + page.Title + "\" is restored and back online.", err
	case "hide":
		page, err := h.setPageFlag(id, pageNavHiddenKey, true)
		return "\"" + page.Title + "\" is hidden from the menu. People can still reach it by its address.", err
	case "show":
		page, err := h.setPageFlag(id, pageNavHiddenKey, false)
		return "\"" + page.Title + "\" is back in the menu.", err
	case "up":
		return "Menu order updated.", h.movePage(id, -1)
	case "down":
		return "Menu order updated.", h.movePage(id, +1)
	default:
		return "", errors.New("unknown action")
	}
}
