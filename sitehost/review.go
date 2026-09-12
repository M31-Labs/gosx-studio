package sitehost

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// review.go is approval before publishing, and preview links for drafts.
//
// With "publishing needs approval" on, editors write and request review;
// an admin looks at the draft, then approves and publishes it or sends it
// back with a note. The request lives on the page or post itself, as
// metadata, so nothing new has to be stored or kept in sync. A preview link
// is a signed token — the library's own preview-link format, signed with
// the install secret — that shows the current draft to anyone who has the
// link, for three days, without an account.

const (
	reviewRequiredKey  = "reviewRequired"
	reviewRequestedKey = "reviewRequested"
	reviewByKey        = "reviewBy"
	reviewAtKey        = "reviewAt"
	reviewNoteKey      = "reviewNote"
	reviewFeedbackKey  = "reviewFeedback"
	reviewPath         = "/admin/review"
	previewPrefix      = "/preview/"
	previewTTL         = 72 * time.Hour
)

// reviewRequired is the site-wide switch.
func (h *Host) reviewRequired() bool {
	return h.settings().Metadata[reviewRequiredKey] == "true"
}

// mustRequestReview reports whether this person has to ask before
// publishing: approval is on, and they are an editor.
func (h *Host) mustRequestReview(r *http.Request) bool {
	return h.reviewRequired() && h.users.count() > 0 && !h.roleAtLeast(r, roleAdmin)
}

// reviewState is what a page or post carries about its review.
type reviewState struct {
	Requested bool
	By        string
	At        time.Time
	Note      string
	Feedback  string
}

func reviewStateOf(metadata cmsstore.Metadata) reviewState {
	state := reviewState{
		Requested: metadata[reviewRequestedKey] == "true",
		By:        strings.TrimSpace(metadata[reviewByKey]),
		Note:      strings.TrimSpace(metadata[reviewNoteKey]),
		Feedback:  strings.TrimSpace(metadata[reviewFeedbackKey]),
	}
	if at, err := time.Parse(time.RFC3339, strings.TrimSpace(metadata[reviewAtKey])); err == nil {
		state.At = at
	}
	return state
}

func setReview(metadata cmsstore.Metadata, requested bool, by, note, feedback string) {
	if requested {
		metadata[reviewRequestedKey] = "true"
		metadata[reviewByKey] = by
		metadata[reviewAtKey] = timeNow().UTC().Format(time.RFC3339)
		if note = strings.TrimSpace(note); note != "" {
			metadata[reviewNoteKey] = note
		} else {
			delete(metadata, reviewNoteKey)
		}
		delete(metadata, reviewFeedbackKey)
		return
	}
	for _, key := range []string{reviewRequestedKey, reviewByKey, reviewAtKey, reviewNoteKey} {
		delete(metadata, key)
	}
	if feedback = strings.TrimSpace(feedback); feedback != "" {
		metadata[reviewFeedbackKey] = feedback
	} else {
		delete(metadata, reviewFeedbackKey)
	}
}

func (h *Host) mountReview(mux *http.ServeMux) {
	mux.HandleFunc("GET "+reviewPath, h.handleReviewQueue)
	mux.HandleFunc("GET "+reviewPath+"/{$}", h.handleReviewQueue)
	mux.HandleFunc("GET "+reviewPath+"/{kind}/{id}", h.handleReviewLook)
	mux.HandleFunc("POST "+reviewPath+"/{kind}/{id}", h.handleReviewDecision)
	mux.HandleFunc("POST /admin/api/pages/{id}/review", h.handlePageReviewRequest)
	mux.HandleFunc("POST /admin/api/posts/{id}/review", h.handlePostReviewRequest)
	mux.HandleFunc("POST /admin/api/pages/{id}/preview-link", h.handlePagePreviewLink)
	mux.HandleFunc("POST /admin/api/posts/{id}/preview-link", h.handlePostPreviewLink)
	mux.HandleFunc("GET "+previewPrefix+"{token}", h.handlePreview)
}

// ---------- settings ----------

func renderReviewField(settings cmsstore.SiteSettings) gosx.Node {
	attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "reviewRequired"), gosx.Attr("value", "true")}
	if settings.Metadata[reviewRequiredKey] == "true" {
		attrs = append(attrs, gosx.Attr("checked", "checked"))
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead"), gosx.Attr("id", "team")), gosx.Text("Team")),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")),
			gosx.El("input", gosx.Attrs(attrs...)),
			gosx.Text(" Publishing needs approval: editors request review, and an admin approves before anything goes live"),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Admins and owners always publish directly. Requests wait under Review.")),
	)
}

func applyReviewField(r *http.Request, metadata cmsstore.Metadata) {
	if r.PostFormValue("reviewRequired") == "true" {
		metadata[reviewRequiredKey] = "true"
	} else {
		delete(metadata, reviewRequiredKey)
	}
}

// ---------- requesting ----------

type reviewPayload struct {
	Note string `json:"note"`
}

func (h *Host) handlePageReviewRequest(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, editorSaveResult{Message: "We couldn't find that page."})
		return
	}
	var payload reviewPayload
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	user, _ := h.currentUser(r)
	metadata := cloneMetadata(page.Metadata)
	setReview(metadata, true, firstNonEmpty(user.Name, "an editor"), payload.Note, "")
	if _, err := h.store.UpdatePage(page.ID, cmsstore.PageInput{Slug: page.Slug, Title: page.Title, Description: page.Description, Body: page.Body, State: page.State, Metadata: metadata}); err != nil {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "We couldn't send that for review. Try again."})
		return
	}
	h.auditContent(r, "review.requested", "Asked for a review of “"+page.Title+"”")
	writeJSON(w, http.StatusOK, editorSaveResult{OK: true, Chip: "Waiting for review", Message: "Sent for review — an admin will take a look", Checks: h.readinessChecks("page", pageMetaValue(page, "metaDescription", page.Description), page.Body)})
}

func (h *Host) handlePostReviewRequest(w http.ResponseWriter, r *http.Request) {
	post, ok, err := h.store.PostByID(r.PathValue("id"))
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, editorSaveResult{Message: "We couldn't find that post."})
		return
	}
	var payload reviewPayload
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	user, _ := h.currentUser(r)
	metadata := cloneMetadata(post.Metadata)
	setReview(metadata, true, firstNonEmpty(user.Name, "an editor"), payload.Note, "")
	if _, err := h.store.UpdatePost(post.ID, postInput(post, metadata)); err != nil {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "We couldn't send that for review. Try again."})
		return
	}
	h.auditContent(r, "review.requested", "Asked for a review of “"+post.Title+"”")
	writeJSON(w, http.StatusOK, editorSaveResult{OK: true, Chip: "Waiting for review", Message: "Sent for review — an admin will take a look", Checks: h.readinessChecks("post", "", post.Body)})
}

// ---------- the queue ----------

type reviewItem struct {
	Kind   string
	ID     string
	Title  string
	State  reviewState
	Editor string
}

func (h *Host) reviewQueue() []reviewItem {
	items := []reviewItem{}
	if pages, err := h.store.ListPages(cmsstore.PageFilter{}); err == nil {
		for _, page := range pages {
			if state := reviewStateOf(page.Metadata); state.Requested {
				items = append(items, reviewItem{Kind: "page", ID: page.ID, Title: page.Title, State: state, Editor: "/admin/edit/" + page.ID})
			}
		}
	}
	if posts, err := h.store.ListPosts(cmsstore.PostFilter{}); err == nil {
		for _, post := range posts {
			if state := reviewStateOf(post.Metadata); state.Requested {
				items = append(items, reviewItem{Kind: "post", ID: post.ID, Title: post.Title, State: state, Editor: "/admin/edit/post/" + post.ID})
			}
		}
	}
	return items
}

func (h *Host) handleReviewQueue(w http.ResponseWriter, r *http.Request) {
	items := h.reviewQueue()
	canDecide := h.roleAtLeast(r, roleAdmin)
	var listing gosx.Node
	if len(items) == 0 {
		text := "When an editor asks for a review, it appears here for an admin to look at, approve, or send back."
		if !h.reviewRequired() {
			text = "Approval is off, so everyone publishes directly. Turn it on under Settings → Team to have editors ask first."
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("Nothing waiting")), gosx.El("p", nil, gosx.Text(text)))
	} else {
		rows := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			actions := []gosx.Node{gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", reviewPath+"/"+item.Kind+"/"+item.ID)), gosx.Text("Look"))}
			if canDecide {
				actions = append(actions, h.reviewButton(item, "approve", "Approve and publish"))
			}
			note := gosx.Fragment()
			if item.State.Note != "" {
				note = gosx.El("br", nil)
				note = gosx.Fragment(gosx.El("br", nil), gosx.El("small", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("“"+item.State.Note+"”")))
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", item.Editor)), gosx.Text(item.Title)), gosx.Text(" "), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge")), gosx.Text(item.Kind)), note),
				gosx.El("td", nil, gosx.Text(item.State.By)),
				gosx.El("td", nil, gosx.Text(formatWhen(item.State.At))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")), gosx.Fragment(actions...)),
			))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Waiting for a decision")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("What")), gosx.El("th", nil, gosx.Text("Who")), gosx.El("th", nil, gosx.Text("When")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...))))
	}
	body := h.renderAdminShell("review", "Review", "Changes waiting for an admin's approval before they go live.", adminStatus{Message: r.URL.Query().Get("status")}, listing)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Review"), body)
}

func (h *Host) reviewButton(item reviewItem, decision, label string) gosx.Node {
	return gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", reviewPath+"/"+item.Kind+"/"+item.ID), gosx.Attr("class", "admin-inline-form")),
		h.csrfField(), hidden("decision", decision),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", decision)), gosx.Text(label)),
	)
}

// draftOf is the working draft of a page or post: title, body, and how to
// render its meta line.
func (h *Host) draftOf(kind, id string) (title string, body blockstudio.Document, metadata cmsstore.Metadata, extra gosx.Node, ok bool) {
	switch kind {
	case "page":
		page, found, err := h.store.PageByID(id)
		if err != nil || !found {
			return "", blockstudio.Document{}, nil, gosx.Fragment(), false
		}
		return page.Title, page.Body, page.Metadata, gosx.Fragment(), true
	case "post":
		post, found, err := h.store.PostByID(id)
		if err != nil || !found {
			return "", blockstudio.Document{}, nil, gosx.Fragment(), false
		}
		return post.Title, post.Body, post.Metadata, renderPostMeta(post), true
	}
	return "", blockstudio.Document{}, nil, gosx.Fragment(), false
}

// renderDraft shows a draft as visitors would see it, under a banner.
func (h *Host) renderDraft(w http.ResponseWriter, kind string, title string, body blockstudio.Document, extra gosx.Node, banner gosx.Node) {
	settings := h.settings()
	meta := metaFromSettings(settings)
	meta.Title = "Preview: " + title
	meta.NoIndex = true
	shell := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-shell")),
		banner,
		h.renderPublicNav(settings, ""),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")),
			gosx.El("article", gosx.Attrs(gosx.Attr("class", "site-article")),
				gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text(title)),
				extra,
				h.renderBody(body, h.hooksFor("#", formState{})),
			),
		),
		h.renderPublicFooter(settings),
	)
	h.writeDocument(w, http.StatusOK, meta, shell)
}

func (h *Host) handleReviewLook(w http.ResponseWriter, r *http.Request) {
	kind, id := r.PathValue("kind"), r.PathValue("id")
	title, body, metadata, extra, ok := h.draftOf(kind, id)
	if !ok {
		h.writeAdminNotFound(w, kind)
		return
	}
	state := reviewStateOf(metadata)
	var actions gosx.Node = gosx.Fragment()
	if h.roleAtLeast(r, roleAdmin) {
		item := reviewItem{Kind: kind, ID: id}
		actions = gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-preview-banner__actions")),
			h.reviewButton(item, "approve", "Approve and publish"),
			gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", reviewPath+"/"+kind+"/"+id), gosx.Attr("class", "admin-inline-form site-preview-banner__sendback")),
				h.csrfField(), hidden("decision", "sendback"),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "note"), gosx.Attr("placeholder", "What to change"), gosx.Attr("aria-label", "Note for the editor"))),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", "sendback")), gosx.Text("Send back"))),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", reviewPath)), gosx.Text("Back to review")),
		)
	}
	who := "This draft is waiting for review."
	if state.By != "" {
		who = state.By + " asked for a review on " + formatWhen(state.At) + "."
	}
	banner := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-preview-banner"), gosx.Attr("role", "status")),
		gosx.El("span", nil, gosx.Text(who+" This is the draft, not what visitors see.")), actions)
	h.renderDraft(w, kind, title, body, extra, banner)
}

func (h *Host) handleReviewDecision(w http.ResponseWriter, r *http.Request) {
	if !h.roleAtLeast(r, roleAdmin) {
		http.Error(w, "Only an admin can decide a review.", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	kind, id := r.PathValue("kind"), r.PathValue("id")
	decision := r.PostFormValue("decision")
	note := strings.TrimSpace(r.PostFormValue("note"))
	var title string
	switch kind {
	case "page":
		page, ok, err := h.store.PageByID(id)
		if err != nil || !ok {
			h.writeAdminNotFound(w, "page")
			return
		}
		title = page.Title
		metadata := cloneMetadata(page.Metadata)
		setReview(metadata, false, "", "", map[bool]string{true: note, false: ""}[decision == "sendback"])
		updated, err := h.store.UpdatePage(page.ID, cmsstore.PageInput{Slug: page.Slug, Title: page.Title, Description: page.Description, Body: page.Body, State: page.State, Metadata: metadata})
		if err != nil {
			http.Redirect(w, r, reviewPath+"?status="+queryEscape("We couldn't save that decision."), http.StatusSeeOther)
			return
		}
		if decision == "approve" {
			if result := h.publishPage(updated); !result.OK {
				http.Redirect(w, r, reviewPath+"?status="+queryEscape(result.Message), http.StatusSeeOther)
				return
			}
		}
	case "post":
		post, ok, err := h.store.PostByID(id)
		if err != nil || !ok {
			h.writeAdminNotFound(w, "post")
			return
		}
		title = post.Title
		metadata := cloneMetadata(post.Metadata)
		setReview(metadata, false, "", "", map[bool]string{true: note, false: ""}[decision == "sendback"])
		updated, err := h.store.UpdatePost(post.ID, postInput(post, metadata))
		if err != nil {
			http.Redirect(w, r, reviewPath+"?status="+queryEscape("We couldn't save that decision."), http.StatusSeeOther)
			return
		}
		if decision == "approve" {
			if result := h.publishPost(updated); !result.OK {
				http.Redirect(w, r, reviewPath+"?status="+queryEscape(result.Message), http.StatusSeeOther)
				return
			}
		}
	default:
		h.writeAdminNotFound(w, kind)
		return
	}
	if decision == "approve" {
		h.auditContent(r, "review.approved", "Approved and published “"+title+"”")
		http.Redirect(w, r, reviewPath+"?status="+queryEscape("Approved: “"+title+"” is live."), http.StatusSeeOther)
		return
	}
	h.auditContent(r, "review.sentback", "Sent “"+title+"” back"+map[bool]string{true: ": " + note, false: ""}[note != ""])
	http.Redirect(w, r, reviewPath+"?status="+queryEscape("Sent “"+title+"” back to its editor."), http.StatusSeeOther)
}

// ---------- preview links ----------

type previewLinkResult struct {
	OK      bool   `json:"ok"`
	URL     string `json:"url,omitempty"`
	Expires string `json:"expires,omitempty"`
	Message string `json:"message,omitempty"`
}

func (h *Host) issuePreviewLink(w http.ResponseWriter, r *http.Request, kind, id string) {
	user, _ := h.currentUser(r)
	link, err := lifecycle.NewPreviewLink(lifecycle.PreviewLinkInput{
		ResourceKind: kind, ResourceID: id, TTL: previewTTL, Issuer: firstNonEmpty(user.Email, "owner"), Nonce: randomHex(6),
	}, timeNow())
	if err != nil {
		writeJSON(w, http.StatusOK, previewLinkResult{Message: "We couldn't make a link. Try again."})
		return
	}
	token, err := lifecycle.SignPreviewLink(link, h.installSecret())
	if err != nil {
		writeJSON(w, http.StatusOK, previewLinkResult{Message: "We couldn't make a link. Try again."})
		return
	}
	writeJSON(w, http.StatusOK, previewLinkResult{OK: true, URL: h.absoluteBase(r) + previewPrefix + token, Expires: formatWhen(link.Expires)})
}

func (h *Host) handlePagePreviewLink(w http.ResponseWriter, r *http.Request) {
	if _, ok, err := h.store.PageByID(r.PathValue("id")); err != nil || !ok {
		writeJSON(w, http.StatusNotFound, previewLinkResult{Message: "We couldn't find that page."})
		return
	}
	h.issuePreviewLink(w, r, "page", r.PathValue("id"))
}

func (h *Host) handlePostPreviewLink(w http.ResponseWriter, r *http.Request) {
	if _, ok, err := h.store.PostByID(r.PathValue("id")); err != nil || !ok {
		writeJSON(w, http.StatusNotFound, previewLinkResult{Message: "We couldn't find that post."})
		return
	}
	h.issuePreviewLink(w, r, "post", r.PathValue("id"))
}

// handlePreview shows a draft to anyone holding a valid link.
func (h *Host) handlePreview(w http.ResponseWriter, r *http.Request) {
	link, err := lifecycle.VerifyPreviewLink(r.PathValue("token"), h.installSecret(), timeNow())
	if err != nil {
		h.servePublicNotFound(w, h.settings(), "preview")
		return
	}
	title, body, _, extra, ok := h.draftOf(link.ResourceKind, link.ResourceID)
	if !ok {
		h.servePublicNotFound(w, h.settings(), "preview")
		return
	}
	banner := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-preview-banner"), gosx.Attr("role", "status")),
		gosx.El("span", nil, gosx.Text("Preview of a draft — not published. This link stops working on "+formatWhen(link.Expires)+".")))
	h.renderDraft(w, link.ResourceKind, title, body, extra, banner)
}
