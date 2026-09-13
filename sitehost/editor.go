package sitehost

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// editor.go is the direct-manipulation editing surface.
//
// The canvas renders the real page — the same markup a visitor sees — with
// editing affordances layered on top. Text is edited in place, sections are
// added and moved by direct manipulation, and nothing reloads. The document in
// the store is the source of truth on the server; the canvas DOM is the source
// of truth in the browser, and the client serializes it back on save.
//
// This is deliberately not the schema card the existing Studio canvas draws
// (canvas/page_surface.go), which shows Route/Component/Source/Status metadata
// and five editable fields. An owner has to recognize their own website.

func (h *Host) mountEditor(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/edit/{id}", h.handleEditor)
	mux.HandleFunc("POST /admin/api/pages/{id}", h.handleEditorSave)
	mux.HandleFunc("POST /admin/api/pages/{id}/publish", h.handleEditorPublish)
	mux.HandleFunc("POST /admin/api/theme", h.handleThemeSave)
	h.mountBlocks(mux)
	mux.Handle("GET "+editorScriptPath, editorScriptHandler())
}

// ---------- the editor shell ----------

// editorSubject is what the canvas edits: a page or a post. The two share
// every editing affordance and differ only in their sidebar fields and where
// their saves go.
type editorSubject struct {
	Kind        string // "page" or "post"
	Noun        string // "page" or "post", for copy
	ID          string
	Title       string
	Slug        string
	Description string
	Body        blockstudio.Document
	Live        bool
	Scheduled   time.Time // a post published ahead of its chosen date
	BackHref    string
	BackLabel   string
	ViewHref    string
	SaveURL     string
	PublishURL  string
	Page        *cmsstore.Page
	Post        *cmsstore.Post
	Checks      []string // what to fix before publishing
	CanDesign   bool     // may change the site-wide Look
	MustRequest bool     // approval is on and this person is an editor
	Review      reviewState
	ReviewURL   string
	PreviewURL  string // where the client asks for a share link
}

func (h *Host) pageSubject(page cmsstore.Page) editorSubject {
	scheduled, _ := scheduledFor(page.Metadata)
	return editorSubject{
		Scheduled:   scheduled,
		Checks:      h.readinessChecks("page", pageMetaValue(page, "metaDescription", page.Description), page.Body),
		Kind:        "page",
		Noun:        "page",
		ID:          page.ID,
		Title:       page.Title,
		Slug:        page.Slug,
		Description: pageMetaValue(page, "metaDescription", page.Description),
		Body:        page.Body,
		Live:        h.isLive(page),
		BackHref:    "/admin/pages",
		BackLabel:   "← Pages",
		ViewHref:    publicPath(page.Slug),
		SaveURL:     "/admin/api/pages/" + page.ID,
		PublishURL:  "/admin/api/pages/" + page.ID + "/publish",
		ReviewURL:   "/admin/api/pages/" + page.ID + "/review",
		PreviewURL:  "/admin/api/pages/" + page.ID + "/preview-link",
		Review:      reviewStateOf(page.Metadata),
		Page:        &page,
	}
}

func (h *Host) handleEditor(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		h.writeAdminNotFound(w, "page")
		return
	}
	subject := h.pageSubject(page)
	subject.CanDesign = h.roleAtLeast(r, roleAdmin)
	subject.MustRequest = h.mustRequestReview(r)
	h.renderEditor(w, subject)
}

func (h *Host) renderEditor(w http.ResponseWriter, subject editorSubject) {
	settings := h.settings()

	body := gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "ed"),
		gosx.Attr("data-editor", "true"),
		gosx.Attr("data-kind", subject.Kind),
		gosx.Attr("data-page-id", subject.ID),
		gosx.Attr("data-page-slug", subject.Slug),
		gosx.Attr("data-save-url", subject.SaveURL),
		gosx.Attr("data-publish-url", subject.PublishURL),
		gosx.Attr("data-review-url", subject.ReviewURL),
		gosx.Attr("data-preview-url", subject.PreviewURL),
		gosx.Attr("data-must-request", boolAttr(subject.MustRequest)),
		gosx.Attr("data-can-lock", boolAttr(subject.CanDesign)),
		gosx.Attr("data-server-kinds", "section,image,button,columns,gallery,"+compositeKeys()),
	),
		h.renderEditorToolbar(subject),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-body")),
			h.renderEditorSidebar(subject),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-stage")),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-frame"), gosx.Attr("data-frame", "true")),
					h.renderEditableCanvas(settings, subject),
				),
			),
		),
		renderInsertMenu(),
		gosx.El("script", gosx.Attrs(gosx.Attr("type", "application/json"), gosx.Attr("data-forms-presets", "true")), gosx.RawHTML(h.formPresetsJSON())),
		gosx.El("script", gosx.Attrs(gosx.Attr("type", "application/json"), gosx.Attr("data-products-presets", "true")), gosx.RawHTML(h.productPresetsJSON())),
		gosx.El("script", gosx.Attrs(gosx.Attr("src", editorScriptPath), gosx.Attr("defer", "defer"))),
	)

	meta := h.adminMeta("Editing " + subject.Title)
	h.writeDocument(w, http.StatusOK, meta, body)
}

func (h *Host) renderEditorToolbar(subject editorSubject) gosx.Node {
	live := subject.Live
	statusText := subjectChip(subject)
	publishText := publishLabel(live || !subject.Scheduled.IsZero())
	if subject.MustRequest {
		publishText = "Request review"
	}
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "ed-bar")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-bar__left")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "ed-back"), gosx.Attr("href", subject.BackHref), gosx.Attr("aria-label", "Back to all "+subject.Noun+"s")),
				gosx.Text(subject.BackLabel)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-page-name")), gosx.Text(subject.Title)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-chip"), gosx.Attr("data-live", boolAttr(live))), gosx.Text(statusText)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-bar__right")),
			renderPeople(),
			gosx.El("span", gosx.Attrs(
				gosx.Attr("class", "ed-save"),
				gosx.Attr("data-save-status", "idle"),
				gosx.Attr("role", "status"),
				gosx.Attr("aria-live", "polite"),
			), gosx.Text("All changes saved")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", "ed-btn ed-btn--ghost"),
				gosx.Attr("type", "button"),
				gosx.Attr("data-undo", "true"),
				gosx.Attr("title", "Undo (Ctrl+Z)"),
			), gosx.Text("Undo")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", "ed-btn ed-btn--ghost"),
				gosx.Attr("type", "button"),
				gosx.Attr("data-redo", "true"),
				gosx.Attr("title", "Redo (Ctrl+Y)"),
			), gosx.Text("Redo")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-device"), gosx.Attr("role", "group"), gosx.Attr("aria-label", "Preview as")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "ed-device__btn"), gosx.Attr("type", "button"), gosx.Attr("data-device", "desktop"), gosx.Attr("aria-pressed", "true"), gosx.Attr("title", "See it on a computer")), gosx.Text("Desktop")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "ed-device__btn"), gosx.Attr("type", "button"), gosx.Attr("data-device", "phone"), gosx.Attr("aria-pressed", "false"), gosx.Attr("title", "See it on a phone")), gosx.Text("Phone")),
			),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "ed-btn ed-btn--ghost"),
				gosx.Attr("href", historyHref(subject.Kind, subject.ID)),
				gosx.Attr("title", "Every save and publish, with restore"),
			), gosx.Text("History")),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "ed-btn ed-btn--ghost"),
				gosx.Attr("href", subject.ViewHref),
				gosx.Attr("target", "_blank"),
				gosx.Attr("rel", "noopener"),
			), gosx.Text("View")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", "ed-btn ed-btn--primary"),
				gosx.Attr("type", "button"),
				gosx.Attr("data-publish", "true"),
			), gosx.Text(publishText)),
		),
	)
}

// alignAttrs carries a text block's alignment onto the canvas.
func alignAttrs(instance blockstudio.BlockInstance) []any {
	if align := normalizeChoice(instance.Values["align"].String, "", textAligns); align != "" {
		return []any{gosx.Attr("data-align", align), gosx.Attr("class", "site-align-"+align)}
	}
	return []any{}
}

// choiceSelect is a small select for a block option.
func choiceSelect(attr, label, current string, options [][2]string) gosx.Node {
	nodes := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		attrs := []any{gosx.Attr("value", option[0])}
		if option[0] == current {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		nodes = append(nodes, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(option[1])))
	}
	return gosx.El("select", gosx.Attrs(gosx.Attr(attr, "true"), gosx.Attr("aria-label", label)), gosx.Fragment(nodes...))
}

// canvasTitleClass shows the page name small when a hero leads, so the
// owner sees what visitors see and still has the name to hand.
func canvasTitleClass(doc blockstudio.Document) string {
	if leadsWithHero(doc) {
		return "site-title ed-title--quiet"
	}
	return "site-title"
}

// renderSectionAdds lists the ready-made sections in the sidebar.
func renderSectionAdds() gosx.Node {
	nodes := make([]gosx.Node, 0, len(composites))
	for _, spec := range composites {
		nodes = append(nodes, addButton(spec.Key, spec.Label, spec.Blurb))
	}
	return gosx.Fragment(nodes...)
}

func publishLabel(live bool) string {
	if live {
		return "Publish changes"
	}
	return "Publish"
}

func boolAttr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func (h *Host) renderEditorSidebar(subject editorSubject) gosx.Node {
	var lookSection gosx.Node = gosx.Fragment()
	if subject.CanDesign {
		lookSection = h.renderLookSection()
	}
	return gosx.El("aside", gosx.Attrs(gosx.Attr("class", "ed-side")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block")),
			gosx.El("h2", nil, gosx.Text("Add to this "+subject.Noun)),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint")),
				gosx.Text("Click a section on the "+subject.Noun+" to edit it. Use these to add something new at the end.")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-add-grid")), renderSectionAdds()),
			gosx.El("h3", gosx.Attrs(gosx.Attr("class", "ed-side__sub")), gosx.Text("Or one piece at a time")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-add-grid")),
				addButton("heading", "Heading", "A section title"),
				addButton("paragraph", "Text", "A paragraph"),
				addButton("quote", "Quote", "A customer's words"),
				addButton("button", "Button", "Sends people somewhere"),
				addButton("image", "Image", "A picture"),
				addButton("gallery", "Gallery", "A grid of pictures"),
				addButton("video", "Video", "YouTube or Vimeo"),
				addButton("columns", "Two columns", "Text side by side"),
				addButton("list", "List", "Bullet points"),
				addButton("divider", "Divider", "A thin line"),
				addButton("section", "Section", "A new background band"),
				addButton("form", "Form", "Contact, sign-up, booking"),
				addButton("product", "Product", "Something from your shop"),
			),
		),
		lookSection,
		h.renderPresetAdds(),
		h.renderSubjectFields(subject),
		renderReviewNotes(subject),
		renderChecks(subject.Checks),
		renderShareBlock(subject),
	)
}

// renderReviewNotes tells an editor where their draft stands.
func renderReviewNotes(subject editorSubject) gosx.Node {
	switch {
	case subject.Review.Requested:
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block")),
			gosx.El("h2", nil, gosx.Text("Review")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-check ed-check--ok")), gosx.Text("Sent for review by "+subject.Review.By+". An admin will approve it or send it back. You can keep editing meanwhile.")))
	case subject.Review.Feedback != "":
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block")),
			gosx.El("h2", nil, gosx.Text("Sent back")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-check")), gosx.Text(subject.Review.Feedback)))
	}
	return gosx.Fragment()
}

// renderShareBlock makes preview links for people without an account.
func renderShareBlock(subject editorSubject) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block"), gosx.Attr("data-share", "true")),
		gosx.El("h2", nil, gosx.Text("Share a preview")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint")), gosx.Text("A link that shows this draft, as it is right now, to anyone who has it. It works for three days.")),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "ed-library-btn"), gosx.Attr("type", "button"), gosx.Attr("data-share-make", "true")), gosx.Text("Make a preview link")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-share__result"), gosx.Attr("data-share-result", "true"), gosx.Attr("hidden", "hidden")),
			gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("readonly", "readonly"), gosx.Attr("data-share-link", "true"), gosx.Attr("aria-label", "Preview link"))),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "ed-library-btn"), gosx.Attr("type", "button"), gosx.Attr("data-share-copy", "true")), gosx.Text("Copy")),
			gosx.El("small", gosx.Attrs(gosx.Attr("data-share-expires", "true"))),
		),
	)
}

// renderChecks is the "before you publish" block; the script refreshes it
// after every save.
func renderChecks(checks []string) gosx.Node {
	items := make([]gosx.Node, 0, len(checks)+1)
	if len(checks) == 0 {
		items = append(items, gosx.El("li", gosx.Attrs(gosx.Attr("class", "ed-check ed-check--ok")), gosx.Text("Looks good. Nothing to fix.")))
	}
	for _, check := range checks {
		items = append(items, gosx.El("li", gosx.Attrs(gosx.Attr("class", "ed-check")), gosx.Text(check)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block"), gosx.Attr("data-checks", "true")),
		gosx.El("h2", nil, gosx.Text("Before you publish")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "ed-checks"), gosx.Attr("data-checks-list", "true")), gosx.Fragment(items...)),
	)
}

// renderSubjectFields is the "This page" or "This post" block of the sidebar.
func (h *Host) renderSubjectFields(subject editorSubject) gosx.Node {
	if subject.Post != nil {
		post := subject.Post
		publishAt := ""
		if at, ok := PostPublishAt(*post); ok {
			publishAt = at.UTC().Format(time.RFC3339)
		}
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block")),
			gosx.El("h2", nil, gosx.Text("This post")),
			editorField("pageSlug", "Web address", post.Slug, "yoursite.com"+postPath(post.Slug)),
			editorField("pageExcerpt", "Summary", post.Excerpt, "One or two sentences, shown in the post list and search results. Leave it blank to use your first paragraph."),
			editorField("pageTags", "Categories", strings.Join(post.Tags, ", "), "Separate with commas, such as \"News, Recipes\"."),
			editorField("pageAuthor", "Written by", post.Author, "Optional. Shown under the title."),
			editorDateField("pagePublishAt", "Publish date", publishAt, "Leave it blank to go live as soon as you publish. Pick a future date to schedule it."),
		)
	}
	publishAt := ""
	if subject.Page != nil {
		if at, ok := PagePublishAt(*subject.Page); ok {
			publishAt = at.UTC().Format(time.RFC3339)
		}
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block")),
		gosx.El("h2", nil, gosx.Text("This page")),
		editorField("pageTitle", "Page name", subject.Title, "Shown as the heading and in your menu."),
		editorField("pageSlug", "Web address", subject.Slug, addressHint(subject.Slug)),
		editorField("pageDescription", "Description for search results", subject.Description,
			"One or two sentences. Also used when someone shares the link."),
		h.renderParentField(subject),
		editorDateField("pagePublishAt", "Publish date", publishAt, "Leave it blank to go live when you press Publish. Pick a future date, then press Publish, to schedule the change."),
	)
}

// renderParentField lets a page sit under another one in the menu.
func (h *Host) renderParentField(subject editorSubject) gosx.Node {
	if subject.Page == nil || subject.Page.Slug == homeSlug {
		return gosx.Fragment()
	}
	current := PageNavParent(*subject.Page)
	options := []gosx.Node{gosx.El("option", gosx.Attrs(gosx.Attr("value", "")), gosx.Text("On its own"))}
	for _, parent := range h.menuParents(*subject.Page) {
		attrs := []any{gosx.Attr("value", parent.ID)}
		if parent.ID == current {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(parent.Title)))
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-field"), gosx.Attr("for", "pageParent")),
		gosx.El("span", nil, gosx.Text("In the menu")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", "pageParent"), gosx.Attr("data-meta", "parent")), gosx.Fragment(options...)),
		gosx.El("small", nil, gosx.Text("Sit this page under another one to make a drop-down.")))
}

// applyNavParent records which page this one sits under, or clears it.
// Only a top-level page can be a parent, so menus stay one level deep.
func (h *Host) applyNavParent(page cmsstore.Page, metadata cmsstore.Metadata, parentID string) {
	parentID = strings.TrimSpace(parentID)
	if parentID == "" || parentID == page.ID {
		delete(metadata, pageNavParentKey)
		return
	}
	parent, ok, _ := h.store.PageByID(parentID)
	if !ok || parent.Slug == homeSlug || PageNavParent(parent) != "" || PageArchived(parent) {
		delete(metadata, pageNavParentKey)
		return
	}
	metadata[pageNavParentKey] = parentID
}

// editorDateField is a date-and-time picker. The stored value is UTC; the
// client shows and reads it in the owner's own time zone.
func editorDateField(id, label, iso, hint string) gosx.Node {
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-field"), gosx.Attr("for", id)),
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "datetime-local"),
			gosx.Attr("id", id),
			gosx.Attr("data-meta", strings.TrimPrefix(id, "page")),
			gosx.Attr("data-iso", iso),
		)),
		gosx.El("small", nil, gosx.Text(hint)),
	)
}

// addressHint shows the address a visitor actually types. The home page is
// stored under the "home" slug but served at the site root.
func addressHint(slug string) string {
	if publicPath(slug) == "/" {
		return "This is your home page, served at /"
	}
	return "yoursite.com" + publicPath(slug)
}

func pageMetaValue(page cmsstore.Page, key, fallback string) string {
	if value := strings.TrimSpace(page.Metadata[key]); value != "" {
		return value
	}
	return fallback
}

func addButton(kind, label, hint string) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "ed-add"),
		gosx.Attr("type", "button"),
		gosx.Attr("data-add", kind),
		gosx.Attr("title", hint),
	),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-add__label")), gosx.Text(label)),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-add__hint")), gosx.Text(hint)),
	)
}

func editorField(id, label, value, hint string) gosx.Node {
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-field"), gosx.Attr("for", id)),
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "text"),
			gosx.Attr("id", id),
			gosx.Attr("data-meta", strings.TrimPrefix(id, "page")),
			gosx.Attr("value", value),
		)),
		gosx.El("small", nil, gosx.Text(hint)),
	)
}

func renderSectionMenuItems() gosx.Node {
	nodes := make([]gosx.Node, 0, len(composites))
	for _, spec := range composites {
		nodes = append(nodes, menuItem(spec.Key, spec.Label))
	}
	return gosx.Fragment(nodes...)
}

func renderInsertMenu() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-menu"), gosx.Attr("data-insert-menu", "true"), gosx.Attr("hidden", "hidden")),
		gosx.Fragment(
			menuItem("heading", "Heading"),
			menuItem("paragraph", "Text"),
			menuItem("quote", "Quote"),
			menuItem("button", "Button"),
			menuItem("image", "Image"),
			menuItem("gallery", "Gallery"),
			menuItem("video", "Video"),
			menuItem("columns", "Two columns"),
			renderSectionMenuItems(),
			menuItem("list", "List"),
			menuItem("divider", "Divider"),
			menuItem("section", "Section"),
			menuItem("form", "Form"),
			menuItem("product", "Product"),
		),
	)
}

func menuItem(kind, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "ed-menu__item"),
		gosx.Attr("type", "button"),
		gosx.Attr("data-insert", kind),
	), gosx.Text(label))
}

// ---------- the editable canvas ----------

// renderEditableCanvas renders the real page inside the editor, one wrapper per
// block carrying the block's kind so the client can serialize the DOM back into
// a document without a second source of truth.
func (h *Host) renderEditableCanvas(settings cmsstore.SiteSettings, subject editorSubject) gosx.Node {
	blocks := make([]gosx.Node, 0, len(subject.Body.Blocks)+1)
	for index, instance := range subject.Body.Blocks {
		blocks = append(blocks, h.renderEditableBlock(index, instance))
	}

	activeSlug := subject.Slug
	var metaLine gosx.Node = gosx.Fragment()
	if subject.Post != nil {
		activeSlug = "blog"
		metaLine = gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "ed-post-meta"),
			gosx.Attr("contenteditable", "false"),
			gosx.Attr("title", "Set the date, author, and categories in the sidebar"),
			gosx.Attr("data-default-date", formatPostDate(postDate(*subject.Post))),
		), renderPostMeta(*subject.Post))
	}

	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-site gosx-site--public ed-canvas")),
		h.renderSiteHeader(settings, brandFromSettings(settings), activeSlug, true),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main")),
			gosx.El("article", gosx.Attrs(gosx.Attr("class", "site-article"), gosx.Attr("data-blocks", "true")),
				gosx.El("h1", gosx.Attrs(
					gosx.Attr("class", canvasTitleClass(subject.Body)),
					gosx.Attr("data-page-title", "true"),
					gosx.Attr("contenteditable", "true"),
					gosx.Attr("spellcheck", "true"),
				), gosx.Text(subject.Title)),
				metaLine,
				gosx.Fragment(blocks...),
			),
		),
		h.renderSiteFooterIn(settings, brandFromSettings(settings), true),
	)
}

func (h *Host) editorNavLinks(activeSlug string) []gosx.Node {
	links := make([]gosx.Node, 0, 6)
	for _, page := range h.navPages() {
		if page.Slug == homeSlug {
			continue
		}
		attrs := []any{gosx.Attr("href", "#"), gosx.Attr("tabindex", "-1")}
		if page.Slug == activeSlug {
			attrs = append(attrs, gosx.Attr("aria-current", "page"))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Text(page.Title)))
	}
	return links
}

// renderEditableBlock wraps one block in the editing chrome: a hover toolbar,
// an insert point, and a contenteditable region for text kinds.
func (h *Host) renderEditableBlock(index int, instance blockstudio.BlockInstance) gosx.Node {
	kind := editorKind(instance.Key)
	inner := h.renderBlockInner(kind, instance)

	hiddenOnPhone := instance.Values[phoneKey].String == phoneHide
	locked := instance.Values[lockedKey].String == "true"
	spacing := normalizeChoice(instance.Values[spacingKey].String, "", blockSpacings)
	class := "ed-block"
	if spacing != "" {
		class += " site-space--" + spacing
	}
	attrs := []any{
		gosx.Attr("class", class),
		gosx.Attr("data-block", kind),
		gosx.Attr("data-index", itoa(index)),
		gosx.Attr("tabindex", "0"),
	}
	if hiddenOnPhone {
		attrs = append(attrs, gosx.Attr("data-phone", phoneHide))
	}
	if locked {
		attrs = append(attrs, gosx.Attr("data-locked", "true"))
	}
	if spacing != "" {
		attrs = append(attrs, gosx.Attr("data-spacing", spacing))
	}
	lockTool := toolButton("lock", "🔒", "Lock: only admins can change this")
	if locked {
		lockTool = gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "ed-tool"), gosx.Attr("type", "button"), gosx.Attr("data-tool", "lock"),
			gosx.Attr("title", "Unlock for editors"), gosx.Attr("aria-label", "Unlock for editors"), gosx.Attr("aria-pressed", "true"),
		), gosx.Text("🔒"))
	}
	phoneTool := toolButton("phone", "📱", "Hide on phones")
	if hiddenOnPhone {
		phoneTool = gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "ed-tool"), gosx.Attr("type", "button"), gosx.Attr("data-tool", "phone"),
			gosx.Attr("title", "Show on phones again"), gosx.Attr("aria-label", "Show on phones again"), gosx.Attr("aria-pressed", "true"),
		), gosx.Text("📱"))
	}
	return gosx.El("div", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-block__tools"), gosx.Attr("contenteditable", "false")),
			toolButton("grab", "⠿", "Drag to move"),
			toolButton("up", "↑", "Move up"),
			toolButton("down", "↓", "Move down"),
			toolButton("duplicate", "⧉", "Make a copy"),
			spacingTool(spacing),
			phoneTool,
			lockTool,
			toolButton("preset", "★", "Save as a preset"),
			toolButton("delete", "✕", "Delete"),
		),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-block__badge"), gosx.Attr("contenteditable", "false")), gosx.Text("Hidden on phones")),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-block__lock"), gosx.Attr("contenteditable", "false")), gosx.Text("Locked")),
		levelPicker(kind, instance),
		inner,
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "ed-insert"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-insert-at", itoa(index)),
			gosx.Attr("contenteditable", "false"),
			gosx.Attr("aria-label", "Add a section here"),
		), gosx.Text("+")),
	)
}

// spacingTool is the room-around-this-block choice in the tools.
func spacingTool(current string) gosx.Node {
	options := make([]gosx.Node, 0, 4)
	for _, option := range [][2]string{{"", "Normal space"}, {"tight", "Tight"}, {"roomy", "Roomy"}, {"extra", "Extra room"}} {
		attrs := []any{gosx.Attr("value", option[0])}
		if option[0] == current {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(option[1])))
	}
	return gosx.El("select", gosx.Attrs(gosx.Attr("class", "ed-tool ed-tool--select"), gosx.Attr("data-tool-spacing", "true"), gosx.Attr("title", "Space around this"), gosx.Attr("aria-label", "Space around this")), gosx.Fragment(options...))
}

func toolButton(action, glyph, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("class", "ed-tool"),
		gosx.Attr("type", "button"),
		gosx.Attr("data-tool", action),
		gosx.Attr("title", label),
		gosx.Attr("aria-label", label),
	), gosx.Text(glyph))
}

// levelPicker lets a heading change rank without a properties panel.
func levelPicker(kind string, instance blockstudio.BlockInstance) gosx.Node {
	if kind != "heading" {
		return gosx.Fragment()
	}
	level := content.NormalizeHeadingLevel(instance.Values["level"].String)
	options := make([]gosx.Node, 0, 3)
	for _, candidate := range []string{"2", "3", "4"} {
		attrs := []any{
			gosx.Attr("class", "ed-level"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-level", candidate),
			gosx.Attr("aria-label", "Heading level "+candidate),
		}
		if candidate == level {
			attrs = append(attrs, gosx.Attr("aria-pressed", "true"))
		}
		options = append(options, gosx.El("button", gosx.Attrs(attrs...), gosx.Text("H"+candidate)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-levels"), gosx.Attr("contenteditable", "false")),
		gosx.Fragment(options...))
}

func (h *Host) renderBlockInner(kind string, instance blockstudio.BlockInstance) gosx.Node {
	value := instance.Values["text"].String
	switch kind {
	case "heading":
		level := content.NormalizeHeadingLevel(instance.Values["level"].String)
		return gosx.El("h"+level, gosx.Attrs(append(alignAttrs(instance),
			gosx.Attr("data-text", "true"),
			gosx.Attr("data-level", level),
			gosx.Attr("contenteditable", "true"),
			gosx.Attr("spellcheck", "true"),
		)...), renderInline(value))
	case "quote":
		return gosx.El("blockquote", gosx.Attrs(append(alignAttrs(instance),
			gosx.Attr("data-text", "true"),
			gosx.Attr("contenteditable", "true"),
			gosx.Attr("spellcheck", "true"),
		)...), renderInline(value))
	case "button":
		look := normalizeChoice(instance.Values["look"].String, "primary", buttonLooks)
		rowAttrs := append(alignAttrs(instance), gosx.Attr("class", "ed-button-row"))
		return gosx.El("span", gosx.Attrs(rowAttrs...),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "site-button button--"+look),
				gosx.Attr("data-text", "true"),
				gosx.Attr("contenteditable", "true"),
				gosx.Attr("href", "#"),
			), gosx.Text(instance.Values["label"].String)),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-variant"), gosx.Attr("contenteditable", "false")), gosx.El("span", nil, gosx.Text("Style")),
				choiceSelect("data-button-look", "Button style", look, [][2]string{{"primary", "Solid"}, {"ghost", "Outline"}, {"link", "Just a link"}})),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("class", "ed-inline-input"),
				gosx.Attr("type", "text"),
				gosx.Attr("data-href", "true"),
				gosx.Attr("value", instance.Values["href"].String),
				gosx.Attr("placeholder", "/contact"),
				gosx.Attr("aria-label", "Where this button goes"),
				gosx.Attr("contenteditable", "false"),
			)),
		)
	case "form":
		return h.renderFormPreview(instance.Values["flowKey"].String)
	case "product":
		return h.renderProductPreview(instance.Values["productRef"].String)
	case "image":
		url := instance.Values["url"].String
		size := normalizeChoice(instance.Values["size"].String, "full", imageSizes)
		shape := normalizeChoice(instance.Values["shape"].String, "natural", imageShapes)
		pick := func(attr, label, current string, options [][2]string) gosx.Node {
			nodes := make([]gosx.Node, 0, len(options))
			for _, option := range options {
				attrs := []any{gosx.Attr("value", option[0])}
				if option[0] == current {
					attrs = append(attrs, gosx.Attr("selected", "selected"))
				}
				nodes = append(nodes, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(option[1])))
			}
			return gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-variant")), gosx.El("span", nil, gosx.Text(label)),
				gosx.El("select", gosx.Attrs(gosx.Attr(attr, "true"), gosx.Attr("aria-label", label)), gosx.Fragment(nodes...)))
		}
		return gosx.El("figure", gosx.Attrs(gosx.Attr("class", "ed-figure site-figure site-figure--"+size+" site-figure--crop-"+shape), gosx.Attr("data-figure", "true")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-figure__options"), gosx.Attr("contenteditable", "false")),
				pick("data-img-size", "Size", size, [][2]string{{"full", "Full width"}, {"wide", "Wider than the text"}, {"medium", "Medium"}, {"small", "Small"}}),
				pick("data-img-shape", "Shape", shape, [][2]string{{"natural", "As it is"}, {"wide", "Wide crop"}, {"square", "Square"}, {"round", "Round"}})),
			imagePreview(url),
			gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input ed-caption"), gosx.Attr("type", "text"), gosx.Attr("data-caption", "true"), gosx.Attr("value", instance.Values["caption"].String), gosx.Attr("placeholder", "A caption under the picture (optional)"), gosx.Attr("aria-label", "Caption"), gosx.Attr("contenteditable", "false"))),
			gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("data-link", "true"), gosx.Attr("value", instance.Values["link"].String), gosx.Attr("placeholder", "Where a click goes (optional), such as /menu"), gosx.Attr("aria-label", "Picture link"), gosx.Attr("contenteditable", "false"))),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-upload"), gosx.Attr("contenteditable", "false")),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("type", "file"),
					gosx.Attr("accept", "image/png,image/jpeg,image/gif,image/webp"),
					gosx.Attr("data-upload", "true"),
					gosx.Attr("aria-label", "Upload a picture"),
				)),
				gosx.El("span", nil, gosx.Text("Upload a picture")),
			),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"), gosx.Attr("class", "ed-library-btn"),
				gosx.Attr("data-library", "true"), gosx.Attr("contenteditable", "false"),
			), gosx.Text("Choose from your pictures")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("class", "ed-inline-input"),
				gosx.Attr("type", "text"),
				gosx.Attr("data-src", "true"),
				gosx.Attr("value", url),
				gosx.Attr("placeholder", "or paste a link to one"),
				gosx.Attr("aria-label", "Image link"),
				gosx.Attr("contenteditable", "false"),
			)),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("class", "ed-inline-input"),
				gosx.Attr("type", "text"),
				gosx.Attr("data-alt", "true"),
				gosx.Attr("value", instance.Values["alt"].String),
				gosx.Attr("placeholder", "Describe the picture for people who can't see it"),
				gosx.Attr("aria-label", "Image description"),
				gosx.Attr("contenteditable", "false"),
			)),
		)
	case "list":
		items := make([]gosx.Node, 0, 6)
		for _, line := range strings.Split(value, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				items = append(items, gosx.El("li", nil, renderInline(line)))
			}
		}
		if len(items) == 0 {
			items = append(items, gosx.El("li", nil, gosx.Text("First point")))
		}
		return gosx.El("ul", gosx.Attrs(
			gosx.Attr("class", "site-list"),
			gosx.Attr("data-text", "true"),
			gosx.Attr("data-list", "true"),
			gosx.Attr("contenteditable", "true"),
			gosx.Attr("spellcheck", "true"),
		), gosx.Fragment(items...))
	case "divider":
		return gosx.El("hr", gosx.Attrs(gosx.Attr("class", "site-divider"), gosx.Attr("contenteditable", "false")))
	case "section":
		return renderSectionBar(sectionOptionsOf(instance))
	case "video":
		return renderVideoEditor(instance.Values["url"].String)
	case "columns":
		count := normalizeChoice(instance.Values["count"].String, "2", columnCounts)
		cols := []gosx.Node{
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-variant ed-columns__count"), gosx.Attr("contenteditable", "false")), gosx.El("span", nil, gosx.Text("Columns")),
				choiceSelect("data-column-count", "How many columns", count, [][2]string{{"2", "Two"}, {"3", "Three"}})),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns__col"), gosx.Attr("data-text", "true"), gosx.Attr("data-col", "1"), gosx.Attr("contenteditable", "true"), gosx.Attr("spellcheck", "true")),
				renderInline(firstNonEmpty(strings.TrimSpace(instance.Values["text"].String), "First column"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns__col"), gosx.Attr("data-text", "true"), gosx.Attr("data-col", "2"), gosx.Attr("contenteditable", "true"), gosx.Attr("spellcheck", "true")),
				renderInline(firstNonEmpty(strings.TrimSpace(instance.Values["text2"].String), "Second column"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns__col site-columns__col--third"), gosx.Attr("data-text", "true"), gosx.Attr("data-col", "3"), gosx.Attr("contenteditable", "true"), gosx.Attr("spellcheck", "true")),
				renderInline(firstNonEmpty(strings.TrimSpace(instance.Values["text3"].String), "Third column"))),
		}
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns site-columns--"+count+" ed-columns"), gosx.Attr("data-columns", count)), gosx.Fragment(cols...))
	case "gallery":
		return renderGalleryEditor(galleryImages(instance), normalizeChoice(instance.Values["style"].String, "grid", galleryStyles))
	default:
		if spec, ok := compositeByKey(kind); ok {
			return h.renderComposite(spec, instance, true)
		}
		return gosx.El("p", gosx.Attrs(append(alignAttrs(instance),
			gosx.Attr("data-text", "true"),
			gosx.Attr("contenteditable", "true"),
			gosx.Attr("spellcheck", "true"),
		)...), renderInline(value))
	}
}

func imagePreview(url string) gosx.Node {
	if strings.TrimSpace(url) == "" {
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-image-empty"), gosx.Attr("data-img", "true")),
			gosx.Text("No picture yet — upload one, or paste a link"))
	}
	return gosx.El("img", gosx.Attrs(gosx.Attr("data-img", "true"), gosx.Attr("src", url), gosx.Attr("alt", "")))
}

// editorKind maps a stored block key to the kinds this editor can edit.
func editorKind(key string) string {
	switch key {
	case content.BlockHeading:
		return "heading"
	case content.BlockQuote:
		return "quote"
	case content.BlockButton:
		return "button"
	case content.BlockImage:
		return "image"
	case content.BlockFlow:
		return "form"
	case blockList:
		return "list"
	case blockDivider:
		return "divider"
	case blockSection:
		return "section"
	case blockVideo:
		return "video"
	case blockColumns:
		return "columns"
	case content.BlockGallery:
		return "gallery"
	case content.BlockProduct:
		return "product"
	default:
		if _, ok := compositeByKey(key); ok {
			return key
		}
		return "paragraph"
	}
}

func storeKey(kind string) string {
	switch kind {
	case "heading":
		return content.BlockHeading
	case "quote":
		return content.BlockQuote
	case "button":
		return content.BlockButton
	case "image":
		return content.BlockImage
	case "form":
		return content.BlockFlow
	case "list":
		return blockList
	case "divider":
		return blockDivider
	case "section":
		return blockSection
	case "video":
		return blockVideo
	case "columns":
		return blockColumns
	case "gallery":
		return content.BlockGallery
	case "product":
		return content.BlockProduct
	default:
		return content.BlockParagraph
	}
}

// ---------- the save API ----------

type editorImagePayload struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type editorBlockPayload struct {
	Kind    string               `json:"kind"`
	Text    string               `json:"text"`
	Text2   string               `json:"text2,omitempty"`
	Level   string               `json:"level,omitempty"`
	URL     string               `json:"url,omitempty"`
	Alt     string               `json:"alt,omitempty"`
	Style   string               `json:"style,omitempty"`
	Form    string               `json:"form,omitempty"`
	Product string               `json:"product,omitempty"`
	Phone   string               `json:"phone,omitempty"`
	Locked  string               `json:"locked,omitempty"`
	Spacing string               `json:"spacing,omitempty"`
	Images  []editorImagePayload `json:"images,omitempty"`
	// Ready-made sections: their named fields, repeated items, and layout.
	Fields  map[string]string   `json:"fields,omitempty"`
	Items   []map[string]string `json:"items,omitempty"`
	Variant string              `json:"variant,omitempty"`
	// Text blocks: where they sit. Buttons: how they look. Columns: how many.
	Align string `json:"align,omitempty"`
	Look  string `json:"look,omitempty"`
	Count string `json:"count,omitempty"`
	Text3 string `json:"text3,omitempty"`
	// Pictures: how big, what shape, a caption, a link.
	Size    string `json:"size,omitempty"`
	Shape   string `json:"shape,omitempty"`
	Caption string `json:"caption,omitempty"`
	Link    string `json:"link,omitempty"`
	// Section breaks: everything beyond the background (Align is shared).
	Width  string `json:"width,omitempty"`
	Anchor string `json:"anchor,omitempty"`
	Space  string `json:"space,omitempty"`
}

type editorSavePayload struct {
	Title       string               `json:"title"`
	Slug        string               `json:"slug"`
	Description string               `json:"description"`
	PublishAt   string               `json:"publishAt"`
	NavParent   string               `json:"navParent"`
	Blocks      []editorBlockPayload `json:"blocks"`
}

// applyPublishAt stores the owner's date in UTC, or clears it — and with
// it any publish that was waiting for it.
func applyPublishAt(metadata cmsstore.Metadata, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		delete(metadata, publishAtKey)
		delete(metadata, publishPendingKey)
		return ""
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return "That publish date doesn't look right. Pick it again."
	}
	metadata[publishAtKey] = at.UTC().Format(time.RFC3339)
	return ""
}

type editorSaveResult struct {
	OK      bool     `json:"ok"`
	Message string   `json:"message,omitempty"`
	Slug    string   `json:"slug,omitempty"`
	Live    bool     `json:"live"`
	Chip    string   `json:"chip,omitempty"`
	Checks  []string `json:"checks"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Host) handleEditorSave(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, editorSaveResult{Message: "We couldn't find that page."})
		return
	}

	var payload editorSavePayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, editorSaveResult{Message: "We couldn't read that change. Try again."})
		return
	}

	title := strings.TrimSpace(payload.Title)
	if title == "" {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "Give the page a name."})
		return
	}
	slug := normalizeSlug(firstNonEmpty(payload.Slug, title))
	if message := reservedSlugMessage(slug); message != "" && page.Slug != slug {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: message})
		return
	}
	if other, exists, _ := h.store.PageBySlug(slug); exists && other.ID != page.ID {
		writeJSON(w, http.StatusOK, editorSaveResult{
			Message: "Another page already uses /" + slug + ".",
		})
		return
	}

	metadata := cmsstore.Metadata{}
	for key, value := range page.Metadata {
		metadata[key] = value
	}
	if description := strings.TrimSpace(payload.Description); description != "" {
		metadata["metaDescription"] = description
	} else {
		delete(metadata, "metaDescription")
	}
	if message := applyPublishAt(metadata, payload.PublishAt); message != "" {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: message})
		return
	}
	h.applyNavParent(page, metadata, payload.NavParent)

	body := h.payloadDocument(payload.Blocks)
	if !h.roleAtLeast(r, roleAdmin) && !lockedBlocksUnchanged(page.Body, body) {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "This page has locked sections that only an admin can change. Your other edits are still here; undo the change to the locked part and save again."})
		return
	}
	input := cmsstore.PageInput{
		Slug:        slug,
		Title:       title,
		Description: strings.TrimSpace(payload.Description),
		Body:        body,
		Metadata:    metadata,
		State:       page.State,
	}
	if _, _, err := h.store.PreviewPage(page.ID, input); err != nil {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "We couldn't save that. Try again."})
		return
	}
	if slug != page.Slug && page.Slug != homeSlug {
		_ = h.recordRedirect(publicPath(page.Slug), publicPath(slug))
	}
	h.notifyChanged(r, "page", page.ID)
	writeJSON(w, http.StatusOK, editorSaveResult{
		OK:     true,
		Slug:   slug,
		Live:   h.isLive(page),
		Checks: h.readinessChecks("page", strings.TrimSpace(payload.Description), input.Body),
	})
}

func (h *Host) payloadDocument(blocks []editorBlockPayload) blockstudio.Document {
	instances := make([]blockstudio.BlockInstance, 0, len(blocks))
	order := 0
	for _, incoming := range blocks {
		kind := strings.TrimSpace(incoming.Kind)
		value := strings.TrimSpace(incoming.Text)
		before := len(instances)
		switch kind {
		case "heading":
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockHeading,
				values("text", value, "level", content.NormalizeHeadingLevel(incoming.Level), "align", normalizeChoice(incoming.Align, "", textAligns))))
		case "quote":
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockQuote, values("text", value, "align", normalizeChoice(incoming.Align, "", textAligns))))
		case "button":
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockButton,
				values("label", value, "href", firstNonEmpty(strings.TrimSpace(incoming.URL), "/"), "look", normalizeChoice(incoming.Look, "primary", buttonLooks), "align", normalizeChoice(incoming.Align, "", textAligns))))
		case "image":
			url := strings.TrimSpace(incoming.URL)
			if url == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockImage,
				values("url", url, "alt", strings.TrimSpace(incoming.Alt), "size", normalizeChoice(incoming.Size, "full", imageSizes), "shape", normalizeChoice(incoming.Shape, "natural", imageShapes), "caption", strings.TrimSpace(incoming.Caption), "link", strings.TrimSpace(incoming.Link))))
		case "form":
			instances = append(instances, block(order, content.BlockFlow, values("flowKey", h.formRefForPayload(incoming.Form))))
		case "product":
			product, ok := h.productByRef(incoming.Product)
			if !ok {
				continue
			}
			instances = append(instances, block(order, content.BlockProduct, values("productRef", product.ref())))
		case "list":
			if value == "" {
				continue
			}
			instances = append(instances, block(order, blockList, values("text", value)))
		case "divider":
			instances = append(instances, block(order, blockDivider, values()))
		case "section":
			options := sectionOptions{Style: normalizeSectionStyle(incoming.Style), Align: normalizeChoice(incoming.Align, "left", sectionAligns), Width: normalizeChoice(incoming.Width, "normal", sectionWidths), Space: normalizeChoice(incoming.Space, "normal", sectionSpaces), Image: strings.TrimSpace(incoming.URL), Anchor: normalizeSlug(incoming.Anchor)}
			instances = append(instances, block(order, blockSection, options.values()))
		case "video":
			url := strings.TrimSpace(incoming.URL)
			if url == "" {
				continue
			}
			instances = append(instances, block(order, blockVideo, values("url", url, "text", value)))
		case "columns":
			right := strings.TrimSpace(incoming.Text2)
			third := strings.TrimSpace(incoming.Text3)
			if value == "" && right == "" && third == "" {
				continue
			}
			instances = append(instances, block(order, blockColumns, values("text", value, "text2", right, "text3", third, "count", normalizeChoice(incoming.Count, "2", columnCounts))))
		case "gallery":
			images := make([][2]string, 0, len(incoming.Images))
			for _, image := range incoming.Images {
				images = append(images, [2]string{image.URL, image.Alt})
			}
			galleryList := galleryValue(images)
			if len(galleryList.List) == 0 {
				continue
			}
			instances = append(instances, blockstudio.BlockInstance{
				ID: content.BlockGallery + "-" + itoa(order), Key: content.BlockGallery, Enabled: true, Order: order,
				Values: blockstudio.Values{"images": galleryList, "style": text(normalizeChoice(incoming.Style, "grid", galleryStyles))},
			})
		default:
			if spec, ok := compositeByKey(kind); ok {
				if instance, ok := compositeFromPayload(spec, incoming, order); ok {
					instances = append(instances, instance)
				}
				break
			}
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockParagraph, values("text", value, "align", normalizeChoice(incoming.Align, "", textAligns))))
		}
		if len(instances) > before {
			if spacing := normalizeChoice(incoming.Spacing, "", blockSpacings); spacing != "" {
				instances[len(instances)-1].Values[spacingKey] = text(spacing)
			}
		}
		if len(instances) > before && strings.TrimSpace(incoming.Phone) == phoneHide {
			instances[len(instances)-1].Values[phoneKey] = text(phoneHide)
		}
		if len(instances) > before && strings.TrimSpace(incoming.Locked) == "true" {
			instances[len(instances)-1].Values[lockedKey] = text("true")
		}
		order++
	}
	return document(instances...)
}

func (h *Host) handleEditorPublish(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		writeJSON(w, http.StatusNotFound, editorSaveResult{Message: "We couldn't find that page."})
		return
	}
	if h.mustRequestReview(r) {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "This site needs an admin to approve changes. Use Request review."})
		return
	}
	page = h.clearReviewOnPage(page)
	result := h.publishPage(page)
	result.Checks = h.readinessChecks("page", pageMetaValue(page, "metaDescription", page.Description), page.Body)
	if result.OK {
		h.auditContent(r, "page.published", firstNonEmpty(result.Message, "Published")+": “"+page.Title+"”")
		h.notifyChanged(r, "page", page.ID)
	}
	writeJSON(w, http.StatusOK, result)
}

// ---------- the Look ----------

// renderLookSection is the site-wide theme picker. It lives in the editor
// rather than on a settings page because the point of choosing a colour is
// seeing the page change under your cursor.
func (h *Host) renderLookSection() gosx.Node {
	theme := h.theme()
	view := theme.view()

	swatches := make([]gosx.Node, 0, 6)
	for _, palette := range Palettes() {
		inputAttrs := []any{
			gosx.Attr("type", "radio"), gosx.Attr("name", "lookPalette"),
			gosx.Attr("id", "look-palette-"+palette.Key), gosx.Attr("value", palette.Key),
			gosx.Attr("data-look-palette", palette.Key),
		}
		if palette.Key == view.PaletteKey {
			inputAttrs = append(inputAttrs, gosx.Attr("checked", "checked"))
		}
		swatches = append(swatches, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "ed-swatch"), gosx.Attr("for", "look-palette-"+palette.Key), gosx.Attr("title", palette.Blurb),
		),
			gosx.El("input", gosx.Attrs(inputAttrs...)),
			gosx.El("span", gosx.Attrs(
				gosx.Attr("class", "ed-swatch__chip"),
				gosx.Attr("style", "background:"+palette.Ground+";border:1px solid "+palette.Rule),
			), gosx.El("i", gosx.Attrs(gosx.Attr("style", "background:"+palette.Accent)))),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-swatch__name")), gosx.Text(palette.Label)),
		))
	}

	customAttrs := []any{gosx.Attr("type", "radio"), gosx.Attr("name", "lookPalette"), gosx.Attr("id", "look-palette-custom"), gosx.Attr("value", customPaletteKey), gosx.Attr("data-look-palette", customPaletteKey)}
	if view.PaletteKey == customPaletteKey {
		customAttrs = append(customAttrs, gosx.Attr("checked", "checked"))
	}
	swatches = append(swatches, gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-swatch"), gosx.Attr("for", "look-palette-custom"), gosx.Attr("title", "Pick your own background and text colours")),
		gosx.El("input", gosx.Attrs(customAttrs...)),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-swatch__chip ed-swatch__chip--custom"), gosx.Attr("style", "background:"+view.Ground+";border:1px solid "+view.Ink)), gosx.El("i", gosx.Attrs(gosx.Attr("style", "background:"+view.Ink)))),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-swatch__name")), gosx.Text("Custom"))))
	custom := gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-custom-colours"), gosx.Attr("data-look-custom", "true")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "color"), gosx.Attr("data-look-ground", "true"), gosx.Attr("value", view.Ground), gosx.Attr("aria-label", "Background colour"))), gosx.Text(" Background")),
		gosx.El("label", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "color"), gosx.Attr("data-look-ink", "true"), gosx.Attr("value", view.Ink), gosx.Attr("aria-label", "Text colour"))), gosx.Text(" Text")),
		gosx.El("small", nil, gosx.Text("Everything else is mixed from these two and the accent.")))

	fonts := make([]gosx.Node, 0, 5)
	for _, pair := range FontPairs() {
		inputAttrs := []any{
			gosx.Attr("type", "radio"), gosx.Attr("name", "lookFonts"),
			gosx.Attr("id", "look-fonts-"+pair.Key), gosx.Attr("value", pair.Key),
			gosx.Attr("data-look-fonts", pair.Key),
		}
		if pair.Key == view.FontsKey {
			inputAttrs = append(inputAttrs, gosx.Attr("checked", "checked"))
		}
		fonts = append(fonts, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "ed-font"), gosx.Attr("for", "look-fonts-"+pair.Key), gosx.Attr("title", pair.Blurb),
		),
			gosx.El("input", gosx.Attrs(inputAttrs...)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-font__sample"), gosx.Attr("style", "font-family:"+pair.Display)), gosx.Text("Aa")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-font__name")), gosx.Text(pair.Label)),
		))
	}

	customFontAttrs := []any{gosx.Attr("type", "radio"), gosx.Attr("name", "lookFonts"), gosx.Attr("id", "look-fonts-custom"), gosx.Attr("value", customFontsKey), gosx.Attr("data-look-fonts", customFontsKey)}
	if view.FontsKey == customFontsKey {
		customFontAttrs = append(customFontAttrs, gosx.Attr("checked", "checked"))
	}
	fonts = append(fonts, gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-font"), gosx.Attr("for", "look-fonts-custom"), gosx.Attr("title", "Name any Google Font")),
		gosx.El("input", gosx.Attrs(customFontAttrs...)),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-font__sample")), gosx.Text("Aa")),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-font__name")), gosx.Text("Custom"))))
	customFontFields := gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-custom-fonts"), gosx.Attr("data-look-custom-fonts", "true")),
		gosx.El("label", nil, gosx.El("span", nil, gosx.Text("Headings")), gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("data-look-font-head", "true"), gosx.Attr("value", view.FontHead), gosx.Attr("placeholder", "Playfair Display"), gosx.Attr("aria-label", "Heading font")))),
		gosx.El("label", nil, gosx.El("span", nil, gosx.Text("Text")), gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("data-look-font-body", "true"), gosx.Attr("value", view.FontBody), gosx.Attr("placeholder", "Inter"), gosx.Attr("aria-label", "Text font")))),
		gosx.El("small", nil, gosx.Text("Any name from fonts.google.com, spelled as it is there.")))

	shapes := make([]gosx.Node, 0, 3)
	for _, shape := range ButtonShapes() {
		inputAttrs := []any{
			gosx.Attr("type", "radio"), gosx.Attr("name", "lookButtons"),
			gosx.Attr("id", "look-buttons-"+shape.Key), gosx.Attr("value", shape.Key),
			gosx.Attr("data-look-buttons", shape.Key),
		}
		if shape.Key == view.ButtonsKey {
			inputAttrs = append(inputAttrs, gosx.Attr("checked", "checked"))
		}
		shapes = append(shapes, gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-shape"), gosx.Attr("for", "look-buttons-"+shape.Key)),
			gosx.El("input", gosx.Attrs(inputAttrs...)),
			gosx.El("i", gosx.Attrs(gosx.Attr("class", "ed-shape__sample"), gosx.Attr("style", "border-radius:"+shape.Radius))),
			gosx.El("span", nil, gosx.Text(shape.Label)),
		))
	}
	spacing := make([]gosx.Node, 0, 3)
	for _, scale := range SpacingScales() {
		inputAttrs := []any{
			gosx.Attr("type", "radio"), gosx.Attr("name", "lookSpacing"),
			gosx.Attr("id", "look-spacing-"+scale.Key), gosx.Attr("value", scale.Key),
			gosx.Attr("data-look-spacing", scale.Key),
		}
		if scale.Key == view.SpacingKey {
			inputAttrs = append(inputAttrs, gosx.Attr("checked", "checked"))
		}
		spacing = append(spacing, gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-shape"), gosx.Attr("for", "look-spacing-"+scale.Key)),
			gosx.El("input", gosx.Attrs(inputAttrs...)),
			gosx.El("i", gosx.Attrs(gosx.Attr("class", "ed-shape__sample ed-shape__sample--space"), gosx.Attr("data-scale", scale.Key))),
			gosx.El("span", nil, gosx.Text(scale.Label)),
		))
	}

	radios := func(name, attr, current string, items [][2]string) gosx.Node {
		nodes := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			inputAttrs := []any{gosx.Attr("type", "radio"), gosx.Attr("name", name), gosx.Attr("id", name+"-"+item[0]), gosx.Attr("value", item[0]), gosx.Attr(attr, item[0])}
			if item[0] == current {
				inputAttrs = append(inputAttrs, gosx.Attr("checked", "checked"))
			}
			nodes = append(nodes, gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-shape ed-shape--text"), gosx.Attr("for", name+"-"+item[0])),
				gosx.El("input", gosx.Attrs(inputAttrs...)), gosx.El("span", nil, gosx.Text(item[1]))))
		}
		return gosx.Fragment(nodes...)
	}
	headingItems := [][2]string{}
	for _, scale := range HeadingScales() {
		headingItems = append(headingItems, [2]string{scale.Key, scale.Label})
	}
	widthItems := [][2]string{}
	for _, width := range PageWidths() {
		widthItems = append(widthItems, [2]string{width.Key, width.Label})
	}
	presets := lookPresetsJSON()

	lookAttrs := []any{gosx.Attr("class", "ed-side__block"), gosx.Attr("data-look", "true"), gosx.Attr("id", "look")}
	if view.PaletteKey == customPaletteKey {
		lookAttrs = append(lookAttrs, gosx.Attr("data-custom", "true"))
	}
	if view.FontsKey == customFontsKey {
		lookAttrs = append(lookAttrs, gosx.Attr("data-custom-fonts", "true"))
	}
	return gosx.El("div", gosx.Attrs(lookAttrs...),
		gosx.El("h2", nil, gosx.Text("Look")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint")),
			gosx.Text("Changes here apply to your whole site, and you can see them on the page as you pick.")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Colours")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-swatches"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Colour palette")),
				gosx.Fragment(swatches...)),
			custom,
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Fonts")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-fonts"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Font pairing")),
				gosx.Fragment(fonts...)),
			customFontFields,
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Accent colour")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-accent")),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("type", "color"), gosx.Attr("id", "lookAccent"),
					gosx.Attr("data-look-accent", "true"), gosx.Attr("value", view.Accent),
					gosx.Attr("aria-label", "Accent colour"),
				)),
				gosx.El("small", nil, gosx.Text("Buttons and links.")),
				gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-look-accent-reset", "true")), gosx.Text("Use the palette's")),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Corners")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-shapes"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Button shape")),
				gosx.Fragment(shapes...)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Spacing")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-shapes"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Spacing")),
				gosx.Fragment(spacing...)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Headings")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-shapes"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Heading size")),
				radios("lookHeadings", "data-look-headings", view.HeadingsKey, headingItems)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Text width")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-shapes"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Text width")),
				radios("lookWidth", "data-look-width", view.WidthKey, widthItems)),
		),
		gosx.El("link", gosx.Attrs(gosx.Attr("rel", "stylesheet"), gosx.Attr("href", fontsPreviewURL()))),
		gosx.El("script", gosx.Attrs(gosx.Attr("type", "application/json"), gosx.Attr("data-look-presets", "true")),
			gosx.RawHTML(presets)),
	)
}

type lookPresetPalette struct {
	Key     string `json:"key"`
	Scheme  string `json:"scheme"`
	Ground  string `json:"ground"`
	Surface string `json:"surface"`
	Ink     string `json:"ink"`
	Muted   string `json:"muted"`
	Rule    string `json:"rule"`
	Accent  string `json:"accent"`
}

type lookPresetShape struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type lookPresetFonts struct {
	Key     string `json:"key"`
	Display string `json:"display"`
	Body    string `json:"body"`
	Fonts   string `json:"fontsUrl"`
}

// lookPresetsJSON hands the picker every preset's values so a click can
// restyle the canvas before the save round-trip returns.
func lookPresetsJSON() string {
	palettes := make([]lookPresetPalette, 0, 6)
	for _, p := range Palettes() {
		palettes = append(palettes, lookPresetPalette{p.Key, p.Scheme, p.Ground, p.Surface, p.Ink, p.Muted, p.Rule, p.Accent})
	}
	fonts := make([]lookPresetFonts, 0, 5)
	for _, f := range FontPairs() {
		fonts = append(fonts, lookPresetFonts{f.Key, f.Display, f.Body, (Theme{Fonts: f}).GoogleFontsURL()})
	}
	shapes := make([]lookPresetShape, 0, 3)
	for _, shape := range ButtonShapes() {
		shapes = append(shapes, lookPresetShape{shape.Key, shape.Radius})
	}
	spacing := make([]lookPresetShape, 0, 3)
	for _, scale := range SpacingScales() {
		spacing = append(spacing, lookPresetShape{scale.Key, scale.Scale})
	}
	headings := make([]lookPresetShape, 0, 3)
	for _, scale := range HeadingScales() {
		headings = append(headings, lookPresetShape{scale.Key, scale.Scale})
	}
	widths := make([]lookPresetShape, 0, 3)
	for _, width := range PageWidths() {
		widths = append(widths, lookPresetShape{width.Key, width.Measure})
	}
	data, err := json.Marshal(map[string]any{"palettes": palettes, "fonts": fonts, "buttons": shapes, "spacing": spacing, "headings": headings, "widths": widths})
	if err != nil {
		return "{}"
	}
	// A closing script tag inside the JSON would end the element early.
	return strings.ReplaceAll(string(data), "</", "<\\/")
}

type themeSavePayload struct {
	Palette  string `json:"palette"`
	Fonts    string `json:"fonts"`
	Accent   string `json:"accent"`
	Buttons  string `json:"buttons"`
	Spacing  string `json:"spacing"`
	Headings string `json:"headings"`
	Width    string `json:"width"`
	Ground   string `json:"ground"`
	Ink      string `json:"ink"`
	FontHead string `json:"fontHead"`
	FontBody string `json:"fontBody"`
}

type themeSaveResult struct {
	OK       bool   `json:"ok"`
	Message  string `json:"message,omitempty"`
	CSS      string `json:"css,omitempty"`
	FontsURL string `json:"fontsUrl,omitempty"`
}

func (h *Host) handleThemeSave(w http.ResponseWriter, r *http.Request) {
	var payload themeSavePayload
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, themeSaveResult{Message: "We couldn't read that change. Try again."})
		return
	}
	theme, err := h.SaveTheme(ThemeChoice{Palette: payload.Palette, Fonts: payload.Fonts, Accent: payload.Accent, Buttons: payload.Buttons, Spacing: payload.Spacing, Headings: payload.Headings, Width: payload.Width, Ground: payload.Ground, Ink: payload.Ink, FontHead: payload.FontHead, FontBody: payload.FontBody})
	if err != nil {
		writeJSON(w, http.StatusOK, themeSaveResult{Message: "We couldn't save the look. Try again."})
		return
	}
	writeJSON(w, http.StatusOK, themeSaveResult{OK: true, CSS: theme.CSS(), FontsURL: theme.GoogleFontsURL()})
}

// renderVideoEditor shows the video on the canvas with its address beneath,
// or a friendly empty state until an address is pasted.
func renderVideoEditor(rawURL string) gosx.Node {
	rawURL = strings.TrimSpace(rawURL)
	var preview gosx.Node
	if embed, ok := videoEmbedURL(rawURL); ok {
		preview = gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-video"), gosx.Attr("data-video-preview", "true")),
			gosx.El("iframe", gosx.Attrs(gosx.Attr("src", embed), gosx.Attr("title", "Video"), gosx.Attr("loading", "lazy"), gosx.Attr("allowfullscreen", "allowfullscreen"))))
	} else {
		preview = gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-image-empty"), gosx.Attr("data-video-preview", "true")),
			gosx.Text("Paste a YouTube or Vimeo link below"))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-video")),
		preview,
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("data-video-url", "true"),
			gosx.Attr("value", rawURL), gosx.Attr("placeholder", "https://youtube.com/watch?v=…"),
			gosx.Attr("aria-label", "Video link"), gosx.Attr("contenteditable", "false"),
		)),
	)
}

// renderGalleryEditor is the gallery on the canvas: its pictures with a
// remove control each, plus upload and library controls.
func renderGalleryEditor(images [][2]string, style string) gosx.Node {
	items := make([]gosx.Node, 0, len(images))
	for _, image := range images {
		items = append(items, galleryEditorItem(image[0], image[1]))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-gallery"), gosx.Attr("data-picker-target", "gallery"), gosx.Attr("contenteditable", "false")),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-variant")), gosx.El("span", nil, gosx.Text("Layout")),
			choiceSelect("data-gallery-style", "Gallery layout", style, [][2]string{{"grid", "Grid"}, {"strip", "Strip, one row"}, {"masonry", "Masonry"}, {"big", "One big at a time"}})),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-gallery site-gallery--"+style+" ed-gallery__grid"), gosx.Attr("data-gallery-items", "true")), gosx.Fragment(items...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-gallery__controls")),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-upload")),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "file"), gosx.Attr("multiple", "multiple"), gosx.Attr("accept", "image/png,image/jpeg,image/gif,image/webp"), gosx.Attr("data-upload", "true"), gosx.Attr("aria-label", "Add pictures"))),
				gosx.El("span", nil, gosx.Text("Add pictures")),
			),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-library-btn"), gosx.Attr("data-library", "true")), gosx.Text("Choose from your pictures")),
		),
	)
}

func galleryEditorItem(url, alt string) gosx.Node {
	return gosx.El("figure", gosx.Attrs(gosx.Attr("class", "site-gallery__item ed-gallery__item ed-item"), gosx.Attr("data-item", "true")),
		itemTools(),
		gosx.El("img", gosx.Attrs(gosx.Attr("src", url), gosx.Attr("alt", ""), gosx.Attr("data-gimg", "true"), gosx.Attr("loading", "lazy"))),
		gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("data-galt", "true"), gosx.Attr("value", alt), gosx.Attr("placeholder", "Describe this picture"), gosx.Attr("aria-label", "Picture description"))),
	)
}

// clearReviewOnPage drops a pending request when an admin publishes directly.
func (h *Host) clearReviewOnPage(page cmsstore.Page) cmsstore.Page {
	if !reviewStateOf(page.Metadata).Requested && reviewStateOf(page.Metadata).Feedback == "" {
		return page
	}
	metadata := cloneMetadata(page.Metadata)
	setReview(metadata, false, "", "", "")
	updated, err := h.store.UpdatePage(page.ID, cmsstore.PageInput{Slug: page.Slug, Title: page.Title, Description: page.Description, Body: page.Body, State: page.State, Metadata: metadata})
	if err != nil {
		return page
	}
	return updated
}

// A locked block belongs to the admins: editors see it, cannot change it,
// and cannot remove it.
const lockedKey = "locked"

// lockedBlocksUnchanged is true when the incoming document keeps every
// locked block exactly as stored, in the same order, and adds none.
func lockedBlocksUnchanged(stored, incoming blockstudio.Document) bool {
	pick := func(doc blockstudio.Document) []string {
		out := []string{}
		for _, instance := range doc.Blocks {
			if instance.Values[lockedKey].String != "true" {
				continue
			}
			raw, _ := json.Marshal(struct {
				Key    string
				Values blockstudio.Values
			}{instance.Key, instance.Values})
			out = append(out, string(raw))
		}
		return out
	}
	before, after := pick(stored), pick(incoming)
	if len(before) != len(after) {
		return false
	}
	for index := range before {
		if before[index] != after[index] {
			return false
		}
	}
	return true
}
