package sitehost

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// privacy.go is the owner's side of data protection: what the site keeps
// about visitors, for how long, and how to export or delete it.

const (
	messageRetentionKey = "messageRetentionDays"
	privacySlug         = "privacy"
)

func (h *Host) messageRetention() int {
	days, _ := strconv.Atoi(strings.TrimSpace(h.settings().Metadata[messageRetentionKey]))
	if days < 0 {
		days = 0
	}
	return days
}

func (h *Host) mountPrivacy(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/messages/export.csv", h.handleMessagesExport)
	mux.HandleFunc("POST /admin/messages/{id}/delete", h.handleMessageDelete)
	mux.HandleFunc("POST /admin/messages/prune", h.handleMessagesPrune)
	mux.HandleFunc("POST /admin/privacy-page", h.handleAddPrivacyPage)
	mux.HandleFunc("GET /admin/metrics", h.handleMetrics)
}

func (h *Host) handleMessagesExport(w http.ResponseWriter, r *http.Request) {
	messages, _ := h.messages.list()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="messages.csv"`)
	w.Header().Set("Cache-Control", "no-store")
	writer := csv.NewWriter(w)
	_ = writer.Write([]string{"Received", "Form", "Name", "Email", "Message", "Page"})
	for index := len(messages) - 1; index >= 0; index-- {
		m := messages[index]
		_ = writer.Write([]string{m.Received.UTC().Format(time.RFC3339), firstNonEmpty(m.FormName, "Contact"), csvSafe(m.Name), csvSafe(m.Email), csvSafe(m.Body), m.Page})
	}
	writer.Flush()
}

func (h *Host) handleMessageDelete(w http.ResponseWriter, r *http.Request) {
	if err := h.messages.remove(r.PathValue("id")); err != nil {
		http.Redirect(w, r, "/admin/messages?status="+queryEscape("We couldn't find that message."), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "message.deleted", "Deleted a message")
	http.Redirect(w, r, "/admin/messages?status="+queryEscape("Deleted."), http.StatusSeeOther)
}

func (h *Host) handleMessagesPrune(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	days, _ := strconv.Atoi(strings.TrimSpace(r.PostFormValue("days")))
	if days < 1 {
		http.Redirect(w, r, "/admin/messages?status="+queryEscape("Say how many days to keep."), http.StatusSeeOther)
		return
	}
	removed := h.messages.pruneOlderThan(timeNow().Add(-time.Duration(days) * 24 * time.Hour))
	h.auditContent(r, "message.pruned", "Deleted "+plural(removed, "message")+" older than "+plural(days, "day"))
	http.Redirect(w, r, "/admin/messages?status="+queryEscape("Deleted "+plural(removed, "message")+" older than "+plural(days, "day")+"."), http.StatusSeeOther)
}

// pruneIfDue applies the retention setting once a day.
func (h *Host) pruneIfDue() {
	days := h.messageRetention()
	if days == 0 {
		return
	}
	h.retention.mu.Lock()
	if !h.retention.lastRun.IsZero() && timeNow().Sub(h.retention.lastRun) < 24*time.Hour {
		h.retention.mu.Unlock()
		return
	}
	h.retention.lastRun = timeNow()
	h.retention.mu.Unlock()
	h.messages.pruneOlderThan(timeNow().Add(-time.Duration(days) * 24 * time.Hour))
}

func privacyStarter(siteTitle string, sells bool) []string {
	lines := []string{
		siteTitle + " keeps as little about you as it can. This page says what it does keep, and why.",
		"**When you send us a message** through a form on this site, we keep what you typed, including your name and email address, so we can reply. You can ask us to delete it at any time.",
		"**Visitor counts.** We count how many people visit and which pages they read. This uses no cookies and stores nothing that identifies you.",
		"**Cookies.** This site sets a cookie only when it has to: to remember what's in your cart, or to keep you signed in if you have an account. It does not use advertising or tracking cookies.",
	}
	if sells {
		lines = append(lines, "**When you buy something**, payment is handled by Stripe on Stripe's own secure pages. We never see your card details. We keep your order, your name, and your delivery address so we can send your order and answer questions about it.")
	}
	lines = append(lines, "**Questions or requests** about your data: get in touch through the contact page, and we'll answer as soon as we can.")
	return lines
}

func (h *Host) handleAddPrivacyPage(w http.ResponseWriter, r *http.Request) {
	if page, ok, _ := h.store.PageBySlug(privacySlug); ok {
		http.Redirect(w, r, "/admin/edit/"+page.ID, http.StatusSeeOther)
		return
	}
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	blocks := []blockstudio.BlockInstance{}
	for index, line := range privacyStarter(siteTitle, len(h.activeProducts()) > 0) {
		blocks = append(blocks, block(index, content.BlockParagraph, values("text", line)))
	}
	page, err := h.store.CreatePage(cmsstore.PageInput{
		Slug: privacySlug, Title: "Privacy", Description: "What " + siteTitle + " keeps about visitors, and why.",
		Body: document(blocks...), Metadata: cmsstore.Metadata{"metaDescription": "What " + siteTitle + " keeps about visitors, and why.", pageNavHiddenKey: "true"},
	})
	if err != nil {
		http.Redirect(w, r, "/admin/settings?status="+queryEscape("We couldn't add the page. Try again."), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "page.created", "Added a privacy page")
	http.Redirect(w, r, "/admin/edit/"+page.ID+"?status="+queryEscape("Here's a privacy page in plain words. Read it, change anything that isn't true for you, then publish it."), http.StatusSeeOther)
}

// renderPrivacyFields is the Settings panel.
func (h *Host) renderPrivacyFields(settings cmsstore.SiteSettings) gosx.Node {
	var pageNote gosx.Node
	if page, ok, _ := h.store.PageBySlug(privacySlug); ok {
		pageNote = gosx.El("p", nil, gosx.Text("You have a privacy page. "), gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/edit/"+page.ID)), gosx.Text("Edit it")))
	} else {
		pageNote = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("No privacy page yet. The panel below writes a plain-language draft for you to check and publish."))
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead"), gosx.Attr("id", "privacy")), gosx.Text("Privacy")),
		adminTextField("messageRetentionDays", "Delete messages after", strings.TrimSpace(settings.Metadata[messageRetentionKey]), "Days. Leave blank to keep messages until you delete them. Messages and form submissions can also be exported or deleted one by one under Messages."),
		pageNote,
	)
}

func applyPrivacyFields(r *http.Request, metadata cmsstore.Metadata) string {
	raw := strings.TrimSpace(r.PostFormValue("messageRetentionDays"))
	if raw == "" {
		delete(metadata, messageRetentionKey)
		return ""
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 || days > 3650 {
		return "\"Delete messages after\" needs a number of days between 1 and 3650, or nothing."
	}
	metadata[messageRetentionKey] = strconv.Itoa(days)
	return ""
}

// renderPrivacyPanel is the button that adds the starter page; it is its
// own form because the settings form is already a form.
func (h *Host) renderPrivacyPanel() gosx.Node {
	if page, ok, _ := h.store.PageBySlug(privacySlug); ok {
		return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Privacy page")),
			gosx.El("p", nil, gosx.Text("Your privacy page lives at "+publicPath(page.Slug)+". Link to it from your footer text so visitors can find it.")),
			gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/edit/"+page.ID)), gosx.Text("Edit the privacy page"))))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Privacy page")),
		gosx.El("p", nil, gosx.Text("Most countries expect a page that says what a website keeps about its visitors. This writes one in plain words — messages, visitor counts, cookies, and orders if you sell — as a draft for you to check.")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/privacy-page")),
			h.csrfField(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Write a privacy page for me"))))
}
