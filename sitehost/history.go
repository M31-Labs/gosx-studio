package sitehost

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/lifecycle"
	"m31labs.dev/gosx-studio/cms/render"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// history.go is version history: every save and every publish of a page or
// post, with a preview of each and a one-click restore.
//
// The revision ledger already records all of it; Studio's lifecycle store
// has done so since before this host existed. What was missing was a screen
// an owner could read. Restoring makes the old version the current draft —
// the live page is untouched until the owner publishes — so the worst a
// wrong click can do is give them another version to restore.

const historyPageSize = 40

func (h *Host) mountHistory(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/history/{kind}/{id}", h.handleHistory)
	mux.HandleFunc("GET /admin/history/{kind}/{id}/{rev}", h.handleHistoryPreview)
	mux.HandleFunc("POST /admin/history/{kind}/{id}/{rev}/restore", h.handleHistoryRestore)
}

// historySubject is the page or post whose history is shown.
type historySubject struct {
	Kind         string // "page" or "post"
	ResourceKind string
	ID           string
	Title        string
	Slug         string
	EditorHref   string
	ListHref     string
	Body         blockstudio.Document
}

func (h *Host) historySubject(kind, id string) (historySubject, bool) {
	switch kind {
	case "page":
		page, ok, err := h.store.PageByID(id)
		if err != nil || !ok {
			return historySubject{}, false
		}
		return historySubject{Kind: "page", ResourceKind: cmsstore.ResourceKindPage, ID: page.ID, Title: page.Title, Slug: page.Slug,
			EditorHref: "/admin/edit/" + page.ID, ListHref: "/admin/pages", Body: page.Body}, true
	case "post":
		post, ok, err := h.store.PostByID(id)
		if err != nil || !ok {
			return historySubject{}, false
		}
		return historySubject{Kind: "post", ResourceKind: cmsstore.ResourceKindPost, ID: post.ID, Title: post.Title, Slug: post.Slug,
			EditorHref: "/admin/edit/post/" + post.ID, ListHref: "/admin/posts", Body: post.Body}, true
	}
	return historySubject{}, false
}

func historyHref(kind, id string) string { return "/admin/history/" + kind + "/" + id }

// revisionLabel is the owner's word for a ledger action.
func revisionLabel(action string) (state, label string) {
	switch {
	case strings.HasSuffix(action, ".published"):
		return "published", "Published"
	case strings.HasSuffix(action, ".restored"):
		return "scheduled", "Restored an earlier version"
	default:
		return "draft", "Saved"
	}
}

func formatWhen(at time.Time) string {
	return at.In(time.Local).Format("2 January 2006, 15:04")
}

func (h *Host) revisionsNewestFirst(subject historySubject) []lifecycle.Revision {
	return newestFirst(h.store.ListRevisions(lifecycle.Filter{ResourceKind: subject.ResourceKind, ResourceID: subject.ID}))
}

// newestFirst orders revisions by when they were made, latest first,
// whatever order the store handed them back in.
func newestFirst(revisions []lifecycle.Revision) []lifecycle.Revision {
	out := make([]lifecycle.Revision, len(revisions))
	copy(out, revisions)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

// latestPublished is the newest "published" revision of a resource.
func (h *Host) latestPublished(resourceKind, resourceID, action string) (lifecycle.Revision, bool) {
	for _, revision := range newestFirst(h.store.ListRevisions(lifecycle.Filter{ResourceKind: resourceKind, ResourceID: resourceID})) {
		if revision.Action == action && len(revision.Snapshot) > 0 {
			return revision, true
		}
	}
	return lifecycle.Revision{}, false
}

func (h *Host) handleHistory(w http.ResponseWriter, r *http.Request) {
	subject, ok := h.historySubject(r.PathValue("kind"), r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, r.PathValue("kind"))
		return
	}
	revisions := h.revisionsNewestFirst(subject)
	page := 1
	if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 1 {
		page = n
	}
	start := (page - 1) * historyPageSize
	if start > len(revisions) {
		start = len(revisions)
	}
	end := start + historyPageSize
	if end > len(revisions) {
		end = len(revisions)
	}

	var listing gosx.Node
	if len(revisions) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("No versions yet")),
			gosx.El("p", nil, gosx.Text("Every save and every publish will appear here, and any of them can be brought back.")),
		)
	} else {
		rows := make([]gosx.Node, 0, end-start)
		for index, revision := range revisions[start:end] {
			state, label := revisionLabel(revision.Action)
			current := gosx.Fragment()
			if page == 1 && index == 0 {
				current = gosx.Fragment(gosx.Text(" "), gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge")), gosx.Text("current")))
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.Text(formatWhen(revision.Created))),
				gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label)), current),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
					gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", historyHref(subject.Kind, subject.ID)+"/"+revision.ID)), gosx.Text("Look")),
					h.restoreButton(subject, revision.ID, "Restore"),
				),
			))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Versions of “"+subject.Title+"”")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")),
				gosx.Text("Newest first. Restoring makes that version your current draft; what visitors see doesn't change until you publish.")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil,
					gosx.El("th", nil, gosx.Text("When")), gosx.El("th", nil, gosx.Text("What")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...)),
			),
			renderPager(historyHref(subject.Kind, subject.ID), page, end < len(revisions)),
		)
	}

	back := gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", subject.EditorHref)), gosx.Text("Back to the editor")),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", subject.ListHref)), gosx.Text("All "+subject.Kind+"s")),
	)
	body := h.renderAdminShell(subject.Kind+"s", "Version history",
		"Everything this "+subject.Kind+" has been. Nothing is ever lost.",
		adminStatus{Message: r.URL.Query().Get("status")}, listing, back)
	h.writeDocument(w, http.StatusOK, h.adminMeta("History of "+subject.Title), body)
}

func (h *Host) restoreButton(subject historySubject, revisionID, label string) gosx.Node {
	return gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", historyHref(subject.Kind, subject.ID)+"/"+revisionID+"/restore"), gosx.Attr("class", "admin-inline-form")),
		h.csrfField(),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("data-action", "restore")), gosx.Text(label)),
	)
}

// handleHistoryPreview shows one version as visitors would have seen it.
func (h *Host) handleHistoryPreview(w http.ResponseWriter, r *http.Request) {
	subject, ok := h.historySubject(r.PathValue("kind"), r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, r.PathValue("kind"))
		return
	}
	revision, found := h.store.RevisionByID(subject.ResourceKind, subject.ID, r.PathValue("rev"))
	if !found {
		h.writeAdminNotFound(w, "version")
		return
	}
	var title string
	var body blockstudio.Document
	var metaLine gosx.Node = gosx.Fragment()
	switch subject.Kind {
	case "page":
		page, err := lifecycle.DecodeSnapshot[cmsstore.Page](revision)
		if err != nil {
			h.writeAdminNotFound(w, "version")
			return
		}
		title, body = page.Title, page.Body
	default:
		post, err := lifecycle.DecodeSnapshot[cmsstore.Post](revision)
		if err != nil {
			h.writeAdminNotFound(w, "version")
			return
		}
		title, body = post.Title, post.Body
		metaLine = renderPostMeta(post)
	}

	settings := h.settings()
	_, label := revisionLabel(revision.Action)
	meta := metaFromSettings(settings)
	meta.Title = "Version from " + formatWhen(revision.Created)
	meta.NoIndex = true
	banner := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-preview-banner"), gosx.Attr("role", "status")),
		gosx.El("span", nil, gosx.Text(label+" on "+formatWhen(revision.Created)+". This is how the "+subject.Kind+" looked then.")),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-preview-banner__actions")),
			h.restoreButton(subject, revision.ID, "Restore this version"),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", historyHref(subject.Kind, subject.ID))), gosx.Text("Back to history")),
		),
	)
	shell := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-shell")),
		banner,
		h.renderPublicNav(settings, subject.Slug),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")),
			gosx.El("article", gosx.Attrs(gosx.Attr("class", "site-article")),
				gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text(title)),
				metaLine,
				h.renderBody(body, render.Hooks{Flow: h.flowHook("#", formState{}), Image: h.imageHook()}),
			),
		),
		h.renderPublicFooter(settings),
	)
	h.writeDocument(w, http.StatusOK, meta, shell)
}

func (h *Host) handleHistoryRestore(w http.ResponseWriter, r *http.Request) {
	subject, ok := h.historySubject(r.PathValue("kind"), r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, r.PathValue("kind"))
		return
	}
	revision, found := h.store.RevisionByID(subject.ResourceKind, subject.ID, r.PathValue("rev"))
	if !found {
		h.writeAdminNotFound(w, "version")
		return
	}
	var err error
	switch subject.Kind {
	case "page":
		_, _, err = h.store.RestorePageRevision(subject.ID, revision.ID)
	default:
		_, _, err = h.store.RestorePostRevision(subject.ID, revision.ID)
	}
	if err != nil {
		http.Redirect(w, r, historyHref(subject.Kind, subject.ID)+"?status="+queryEscape("We couldn't restore that version. Try again."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, historyHref(subject.Kind, subject.ID)+"?status="+queryEscape(
		"Restored the version from "+formatWhen(revision.Created)+". It's your current draft now; open the editor and publish when you're happy with it."), http.StatusSeeOther)
}
