package sitehost

import (
	"encoding/json"
	"net/http"
	"strings"

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
	mux.Handle("GET "+editorScriptPath, editorScriptHandler())
}

// ---------- the editor shell ----------

func (h *Host) handleEditor(w http.ResponseWriter, r *http.Request) {
	page, ok, err := h.store.PageByID(r.PathValue("id"))
	if err != nil || !ok {
		h.writeAdminNotFound(w, "page")
		return
	}

	settings := h.settings()
	live := page.State.Publish == cmsstore.PublishStatePublished

	body := gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "ed"),
		gosx.Attr("data-editor", "true"),
		gosx.Attr("data-page-id", page.ID),
		gosx.Attr("data-page-slug", page.Slug),
	),
		h.renderEditorToolbar(page, live),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-body")),
			h.renderEditorSidebar(page),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-stage")),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-frame"), gosx.Attr("data-frame", "true")),
					h.renderEditableCanvas(settings, page),
				),
			),
		),
		renderInsertMenu(),
		gosx.El("script", gosx.Attrs(gosx.Attr("src", editorScriptPath), gosx.Attr("defer", "defer"))),
	)

	meta := h.adminMeta("Editing " + page.Title)
	h.writeDocument(w, http.StatusOK, meta, body)
}

func (h *Host) renderEditorToolbar(page cmsstore.Page, live bool) gosx.Node {
	statusText := "Not published yet"
	if live {
		statusText = "Live"
	}
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "ed-bar")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-bar__left")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "ed-back"), gosx.Attr("href", "/admin/pages"), gosx.Attr("aria-label", "Back to all pages")),
				gosx.Text("← Pages")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-page-name")), gosx.Text(page.Title)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-chip"), gosx.Attr("data-live", boolAttr(live))), gosx.Text(statusText)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-bar__right")),
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
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "ed-btn ed-btn--ghost"),
				gosx.Attr("href", publicPath(page.Slug)),
				gosx.Attr("target", "_blank"),
				gosx.Attr("rel", "noopener"),
			), gosx.Text("View")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", "ed-btn ed-btn--primary"),
				gosx.Attr("type", "button"),
				gosx.Attr("data-publish", "true"),
			), gosx.Text(publishLabel(live))),
		),
	)
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

func (h *Host) renderEditorSidebar(page cmsstore.Page) gosx.Node {
	return gosx.El("aside", gosx.Attrs(gosx.Attr("class", "ed-side")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block")),
			gosx.El("h2", nil, gosx.Text("Add to this page")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint")),
				gosx.Text("Click a section on the page to edit it. Use these to add something new at the end.")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-add-grid")),
				addButton("heading", "Heading", "A section title"),
				addButton("paragraph", "Text", "A paragraph"),
				addButton("quote", "Quote", "A customer's words"),
				addButton("button", "Button", "Sends people somewhere"),
				addButton("image", "Image", "A picture"),
			),
		),
		h.renderLookSection(),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block")),
			gosx.El("h2", nil, gosx.Text("This page")),
			editorField("pageTitle", "Page name", page.Title, "Shown as the heading and in your menu."),
			editorField("pageSlug", "Web address", page.Slug, addressHint(page.Slug)),
			editorField("pageDescription", "Description for search results", pageMetaValue(page, "metaDescription", page.Description),
				"One or two sentences. Also used when someone shares the link."),
		),
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

func renderInsertMenu() gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-menu"), gosx.Attr("data-insert-menu", "true"), gosx.Attr("hidden", "hidden")),
		gosx.Fragment(
			menuItem("heading", "Heading"),
			menuItem("paragraph", "Text"),
			menuItem("quote", "Quote"),
			menuItem("button", "Button"),
			menuItem("image", "Image"),
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
func (h *Host) renderEditableCanvas(settings cmsstore.SiteSettings, page cmsstore.Page) gosx.Node {
	blocks := make([]gosx.Node, 0, len(page.Body.Blocks)+1)
	for index, instance := range page.Body.Blocks {
		blocks = append(blocks, renderEditableBlock(index, instance))
	}

	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "gosx-site gosx-site--public ed-canvas")),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "site-header")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-brand")), gosx.Text(firstNonEmpty(settings.Title, h.opts.SiteTitle))),
			gosx.El("nav", gosx.Attrs(gosx.Attr("class", "site-nav"), gosx.Attr("aria-label", "Site")),
				gosx.Fragment(h.editorNavLinks(page.Slug)...)),
		),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main")),
			gosx.El("article", gosx.Attrs(gosx.Attr("class", "site-article"), gosx.Attr("data-blocks", "true")),
				gosx.El("h1", gosx.Attrs(
					gosx.Attr("class", "site-title"),
					gosx.Attr("data-page-title", "true"),
					gosx.Attr("contenteditable", "true"),
					gosx.Attr("spellcheck", "true"),
				), gosx.Text(page.Title)),
				gosx.Fragment(blocks...),
			),
		),
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
func renderEditableBlock(index int, instance blockstudio.BlockInstance) gosx.Node {
	kind := editorKind(instance.Key)
	inner := renderBlockInner(kind, instance)

	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "ed-block"),
		gosx.Attr("data-block", kind),
		gosx.Attr("data-index", itoa(index)),
		gosx.Attr("tabindex", "0"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-block__tools"), gosx.Attr("contenteditable", "false")),
			toolButton("grab", "⠿", "Drag to move"),
			toolButton("up", "↑", "Move up"),
			toolButton("down", "↓", "Move down"),
			toolButton("duplicate", "⧉", "Make a copy"),
			toolButton("delete", "✕", "Delete"),
		),
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

func renderBlockInner(kind string, instance blockstudio.BlockInstance) gosx.Node {
	value := instance.Values["text"].String
	switch kind {
	case "heading":
		level := content.NormalizeHeadingLevel(instance.Values["level"].String)
		return gosx.El("h"+level, gosx.Attrs(
			gosx.Attr("data-text", "true"),
			gosx.Attr("data-level", level),
			gosx.Attr("contenteditable", "true"),
			gosx.Attr("spellcheck", "true"),
		), gosx.Text(value))
	case "quote":
		return gosx.El("blockquote", gosx.Attrs(
			gosx.Attr("data-text", "true"),
			gosx.Attr("contenteditable", "true"),
			gosx.Attr("spellcheck", "true"),
		), gosx.Text(value))
	case "button":
		return gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-button-row")),
			gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "site-button"),
				gosx.Attr("data-text", "true"),
				gosx.Attr("contenteditable", "true"),
				gosx.Attr("href", "#"),
			), gosx.Text(instance.Values["label"].String)),
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
	case "image":
		url := instance.Values["url"].String
		return gosx.El("figure", gosx.Attrs(gosx.Attr("class", "ed-figure")),
			imagePreview(url),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-upload"), gosx.Attr("contenteditable", "false")),
				gosx.El("input", gosx.Attrs(
					gosx.Attr("type", "file"),
					gosx.Attr("accept", "image/png,image/jpeg,image/gif,image/webp"),
					gosx.Attr("data-upload", "true"),
					gosx.Attr("aria-label", "Upload a picture"),
				)),
				gosx.El("span", nil, gosx.Text("Upload a picture")),
			),
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
	default:
		return gosx.El("p", gosx.Attrs(
			gosx.Attr("data-text", "true"),
			gosx.Attr("contenteditable", "true"),
			gosx.Attr("spellcheck", "true"),
		), gosx.Text(value))
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
	default:
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
	default:
		return content.BlockParagraph
	}
}

// ---------- the save API ----------

type editorBlockPayload struct {
	Kind  string `json:"kind"`
	Text  string `json:"text"`
	Level string `json:"level,omitempty"`
	URL   string `json:"url,omitempty"`
	Alt   string `json:"alt,omitempty"`
}

type editorSavePayload struct {
	Title       string               `json:"title"`
	Slug        string               `json:"slug"`
	Description string               `json:"description"`
	Blocks      []editorBlockPayload `json:"blocks"`
}

type editorSaveResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Slug    string `json:"slug,omitempty"`
	Live    bool   `json:"live"`
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

	input := cmsstore.PageInput{
		Slug:        slug,
		Title:       title,
		Description: strings.TrimSpace(payload.Description),
		Body:        payloadDocument(payload.Blocks),
		Metadata:    metadata,
		State:       page.State,
	}
	if _, _, err := h.store.PreviewPage(page.ID, input); err != nil {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "We couldn't save that. Try again."})
		return
	}
	writeJSON(w, http.StatusOK, editorSaveResult{
		OK:   true,
		Slug: slug,
		Live: page.State.Publish == cmsstore.PublishStatePublished,
	})
}

func payloadDocument(blocks []editorBlockPayload) blockstudio.Document {
	instances := make([]blockstudio.BlockInstance, 0, len(blocks))
	order := 0
	for _, incoming := range blocks {
		kind := strings.TrimSpace(incoming.Kind)
		value := strings.TrimSpace(incoming.Text)
		switch kind {
		case "heading":
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockHeading,
				values("text", value, "level", content.NormalizeHeadingLevel(incoming.Level))))
		case "quote":
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockQuote, values("text", value)))
		case "button":
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockButton,
				values("label", value, "href", firstNonEmpty(strings.TrimSpace(incoming.URL), "/"))))
		case "image":
			url := strings.TrimSpace(incoming.URL)
			if url == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockImage,
				values("url", url, "alt", strings.TrimSpace(incoming.Alt))))
		default:
			if value == "" {
				continue
			}
			instances = append(instances, block(order, content.BlockParagraph, values("text", value)))
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
	// Publishing is the owner saying "show this": it also clears an earlier
	// "take offline", which would otherwise silently keep the page hidden.
	if PageOffline(page) {
		if _, err := h.setPageFlag(page.ID, pageOfflineKey, false); err != nil {
			writeJSON(w, http.StatusOK, editorSaveResult{Message: "We couldn't publish that. Try again."})
			return
		}
	}
	if _, _, err := h.store.PublishPage(page.ID); err != nil {
		writeJSON(w, http.StatusOK, editorSaveResult{Message: "We couldn't publish that. Try again."})
		return
	}
	writeJSON(w, http.StatusOK, editorSaveResult{
		OK:      true,
		Live:    true,
		Slug:    page.Slug,
		Message: "Published",
	})
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

	presets := lookPresetsJSON()

	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-side__block"), gosx.Attr("data-look", "true"), gosx.Attr("id", "look")),
		gosx.El("h2", nil, gosx.Text("Look")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-hint")),
			gosx.Text("Changes here apply to your whole site, and you can see them on the page as you pick.")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Colours")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-swatches"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Colour palette")),
				gosx.Fragment(swatches...)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-look-group")),
			gosx.El("span", nil, gosx.Text("Fonts")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-fonts"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", "Font pairing")),
				gosx.Fragment(fonts...)),
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
	data, err := json.Marshal(map[string]any{"palettes": palettes, "fonts": fonts})
	if err != nil {
		return "{}"
	}
	// A closing script tag inside the JSON would end the element early.
	return strings.ReplaceAll(string(data), "</", "<\\/")
}

type themeSavePayload struct {
	Palette string `json:"palette"`
	Fonts   string `json:"fonts"`
	Accent  string `json:"accent"`
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
	theme, err := h.SaveTheme(payload.Palette, payload.Fonts, payload.Accent)
	if err != nil {
		writeJSON(w, http.StatusOK, themeSaveResult{Message: "We couldn't save the look. Try again."})
		return
	}
	writeJSON(w, http.StatusOK, themeSaveResult{OK: true, CSS: theme.CSS(), FontsURL: theme.GoogleFontsURL()})
}
