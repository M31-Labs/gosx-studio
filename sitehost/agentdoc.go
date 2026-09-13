package sitehost

import (
	"net/http"
	"strconv"
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// agentdoc.go is the read side of the agent API: a stored document back
// into the same block shape the editor sends, pages as Markdown for agents
// that read rather than edit, the llms.txt pair, and the schema and OpenAPI
// documents that tell an agent what it can build.

func socialNetworkKeys() []string {
	keys := []string{}
	for _, network := range SocialNetworks() {
		keys = append(keys, network.Key)
	}
	return keys
}

// documentPayload is the inverse of payloadDocument: every enabled block
// as the editor (and the agent API) would send it.
func (h *Host) documentPayload(doc blockstudio.Document) []editorBlockPayload {
	out := make([]editorBlockPayload, 0, len(doc.Blocks))
	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue
		}
		out = append(out, h.blockPayload(instance))
	}
	return out
}

func (h *Host) blockPayload(instance blockstudio.BlockInstance) editorBlockPayload {
	v := instance.Values
	str := func(key string) string { return strings.TrimSpace(v[key].String) }
	kind := editorKind(instance.Key)
	payload := editorBlockPayload{Kind: kind}
	switch instance.Key {
	case content.BlockHeading:
		payload.Text, payload.Level, payload.Align = str("text"), str("level"), str("align")
	case content.BlockQuote:
		payload.Text, payload.Align = str("text"), str("align")
	case content.BlockButton:
		payload.Text, payload.URL, payload.Look, payload.Align = str("label"), str("href"), str("look"), str("align")
	case content.BlockImage:
		payload.URL, payload.Alt, payload.Size, payload.Shape, payload.Caption, payload.Link = str("url"), str("alt"), str("size"), str("shape"), str("caption"), str("link")
	case content.BlockFlow:
		payload.Form = firstNonEmpty(str("flowKey"), contactFlowKey)
	case content.BlockProduct:
		payload.Product = str("productRef")
	case blockList:
		payload.Text = str("text")
	case blockDivider:
	case blockSection:
		options := sectionOptionsOf(instance)
		payload.Style, payload.Align, payload.Width, payload.Space, payload.URL, payload.Anchor = options.Style, options.Align, options.Width, options.Space, options.Image, options.Anchor
	case blockVideo:
		payload.URL, payload.Text = str("url"), str("text")
	case blockColumns:
		payload.Text, payload.Text2, payload.Text3, payload.Count = str("text"), str("text2"), str("text3"), str("count")
	case content.BlockGallery:
		for _, image := range galleryImages(instance) {
			payload.Images = append(payload.Images, editorImagePayload{URL: image[0], Alt: image[1]})
		}
		payload.Style = str("style")
	default:
		if spec, ok := compositeByKey(instance.Key); ok {
			payload.Variant = spec.variant(str("variant"))
			if len(spec.Fields) > 0 {
				payload.Fields = map[string]string{}
				for _, field := range spec.Fields {
					payload.Fields[field.Key] = str(field.Key)
				}
			}
			if spec.Item != nil {
				payload.Items = compositeItems(spec, instance)
			}
		} else {
			payload.Text, payload.Align = str("text"), str("align")
		}
	}
	payload.Spacing = normalizeChoice(str(spacingKey), "", blockSpacings)
	if str(phoneKey) == phoneHide {
		payload.Phone = phoneHide
	}
	if str(lockedKey) == "true" {
		payload.Locked = "true"
	}
	return payload
}

// ---------- Markdown ----------

// pageMarkdown renders a page for a reader that wants text: an agent, a
// crawler, a search tool. Inline marks are already Markdown.
func (h *Host) pageMarkdown(page cmsstore.Page, base string) string {
	var b strings.Builder
	b.WriteString("# " + strings.TrimSpace(page.Title) + "\n\n")
	if description := strings.TrimSpace(pageMetaValue(page, "metaDescription", page.Description)); description != "" {
		b.WriteString("> " + description + "\n\n")
	}
	b.WriteString(h.documentMarkdown(page.Body, base))
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func (h *Host) postMarkdown(post cmsstore.Post, base string) string {
	var b strings.Builder
	b.WriteString("# " + strings.TrimSpace(post.Title) + "\n\n")
	meta := []string{}
	if post.Author != "" {
		meta = append(meta, "By "+post.Author)
	}
	if post.State.PublishedAt != nil {
		meta = append(meta, post.State.PublishedAt.Format("2 January 2006"))
	}
	if len(post.Tags) > 0 {
		meta = append(meta, "Tags: "+strings.Join(post.Tags, ", "))
	}
	if len(meta) > 0 {
		b.WriteString("_" + strings.Join(meta, " · ") + "_\n\n")
	}
	if post.Excerpt != "" {
		b.WriteString("> " + strings.TrimSpace(post.Excerpt) + "\n\n")
	}
	b.WriteString(h.documentMarkdown(post.Body, base))
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func (h *Host) documentMarkdown(doc blockstudio.Document, base string) string {
	var b strings.Builder
	abs := func(href string) string {
		if strings.HasPrefix(href, "/") && base != "" {
			return base + href
		}
		return href
	}
	para := func(text string) {
		if text = strings.TrimSpace(text); text != "" {
			b.WriteString(text + "\n\n")
		}
	}
	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue
		}
		v := instance.Values
		str := func(key string) string { return strings.TrimSpace(v[key].String) }
		switch instance.Key {
		case content.BlockHeading:
			level := "##"
			if content.NormalizeHeadingLevel(str("level")) == "3" {
				level = "###"
			}
			para(level + " " + str("text"))
		case content.BlockQuote:
			if text := str("text"); text != "" {
				para("> " + strings.ReplaceAll(text, "\n", "\n> "))
			}
		case content.BlockButton:
			para("[" + str("label") + "](" + abs(str("href")) + ")")
		case content.BlockImage:
			line := "![" + str("alt") + "](" + abs(str("url")) + ")"
			if caption := str("caption"); caption != "" {
				line += "\n_" + caption + "_"
			}
			para(line)
		case content.BlockFlow:
			name := "Contact form"
			if form, ok := h.formByRef(str("flowKey")); ok {
				name = form.Name + " form"
			}
			para("_" + name + " (fill it in on the page)_")
		case content.BlockProduct:
			if product, ok := h.productByRef(str("productRef")); ok {
				para("**" + product.Name + "** — " + formatMoney(product.Price, h.currency()) + " · [" + product.Name + "](" + abs(shopPath+"/"+product.Slug) + ")")
			}
		case blockList:
			lines := []string{}
			for _, line := range strings.Split(str("text"), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					lines = append(lines, "- "+line)
				}
			}
			para(strings.Join(lines, "\n"))
		case blockDivider:
			para("---")
		case blockSection:
			if anchor := sectionOptionsOf(instance).Anchor; anchor != "" {
				para("<a id=\"" + anchor + "\"></a>")
			}
		case blockVideo:
			para("[" + firstNonEmpty(str("text"), "Watch the video") + "](" + str("url") + ")")
		case blockColumns:
			for _, key := range []string{"text", "text2", "text3"} {
				para(str(key))
			}
		case content.BlockGallery:
			lines := []string{}
			for _, image := range galleryImages(instance) {
				lines = append(lines, "!["+image[1]+"]("+abs(image[0])+")")
			}
			para(strings.Join(lines, "\n"))
		default:
			spec, ok := compositeByKey(instance.Key)
			if !ok {
				para(str("text"))
				continue
			}
			b.WriteString(h.compositeMarkdown(spec, instance, abs))
		}
	}
	return b.String()
}

func (h *Host) compositeMarkdown(spec compositeSpec, instance blockstudio.BlockInstance, abs func(string) string) string {
	var b strings.Builder
	v := instance.Values
	str := func(key string) string { return strings.TrimSpace(v[key].String) }
	if spec.Live {
		switch spec.Key {
		case "posts":
			b.WriteString("## " + firstNonEmpty(str("heading"), "Latest posts") + "\n\n")
			for i, post := range h.livePosts() {
				if i >= 3 {
					break
				}
				b.WriteString("- [" + post.Title + "](" + abs(postPath(post.Slug)) + ")\n")
			}
			b.WriteString("\n")
		case "products":
			b.WriteString("## " + firstNonEmpty(str("heading"), "From the shop") + "\n\n")
			for i, product := range h.products.list() {
				if i >= 6 || !product.Active {
					continue
				}
				b.WriteString("- [" + product.Name + "](" + abs(shopPath+"/"+product.Slug) + ") — " + formatMoney(product.Price, h.currency()) + "\n")
			}
			b.WriteString("\n")
		}
		return b.String()
	}
	// The first text field that reads as a title becomes the heading; every
	// other text field is a paragraph; links and pictures keep their shape.
	for _, field := range spec.Fields {
		value := str(field.Key)
		if value == "" {
			continue
		}
		switch {
		case field.Key == "headline" || field.Key == "heading" || field.Key == "title":
			b.WriteString("## " + value + "\n\n")
		case field.Key == "eyebrow":
			b.WriteString("_" + value + "_\n\n")
		case field.Kind == partImage:
			b.WriteString("![](" + abs(value) + ")\n\n")
		case field.Kind == partURL:
			if label := firstNonEmpty(str("button"), str("button2"), "Link"); strings.HasSuffix(field.Key, "url") || field.Key == "url" {
				b.WriteString("[" + label + "](" + abs(value) + ")\n\n")
			}
		case field.Kind == partFlag:
		case field.Key == "button" || field.Key == "button2":
			// rendered with its url above
		default:
			b.WriteString(value + "\n\n")
		}
	}
	for _, item := range compositeItems(spec, instance) {
		parts := []string{}
		for _, field := range spec.Item {
			value := strings.TrimSpace(item[field.Key])
			if value == "" || field.Kind == partFlag {
				continue
			}
			switch field.Kind {
			case partImage:
				parts = append(parts, "!["+"]("+abs(value)+")")
			case partURL:
				parts = append(parts, "<"+abs(value)+">")
			default:
				if field.Key == "title" || field.Key == "name" || field.Key == "question" || field.Key == "day" || field.Key == "value" {
					parts = append(parts, "**"+value+"**")
				} else {
					parts = append(parts, strings.ReplaceAll(value, "\n", " / "))
				}
			}
		}
		if len(parts) > 0 {
			b.WriteString("- " + strings.Join(parts, " — ") + "\n")
		}
	}
	if spec.Item != nil {
		b.WriteString("\n")
	}
	return b.String()
}

// wantsMarkdown is true for a reader that asked for text rather than HTML.
func wantsMarkdown(r *http.Request) bool {
	if format := strings.ToLower(r.URL.Query().Get("format")); format == "md" || format == "markdown" {
		return true
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	if !strings.Contains(accept, "text/markdown") {
		return false
	}
	// Prefer Markdown only when it is listed before HTML, or HTML is absent.
	html := strings.Index(accept, "text/html")
	return html < 0 || strings.Index(accept, "text/markdown") < html
}

func writeMarkdown(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Vary", "Accept")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write([]byte(body))
}

// ---------- llms.txt ----------

const (
	llmsPath     = "/llms.txt"
	llmsFullPath = "/llms-full.txt"
)

func (h *Host) mountAgentPublic(mux *http.ServeMux) {
	mux.HandleFunc("GET "+llmsPath, h.handleLLMs)
	mux.HandleFunc("GET "+llmsFullPath, h.handleLLMsFull)
}

// handleLLMs is the site's front door for agents: what it is, where every
// page lives, and how to work with it.
func (h *Host) handleLLMs(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	base := h.absoluteBase(r)
	m := settings.Metadata
	var b strings.Builder
	b.WriteString("# " + firstNonEmpty(settings.Title, h.opts.SiteTitle) + "\n\n")
	if description := firstNonEmpty(strings.TrimSpace(m["tagline"]), strings.TrimSpace(settings.Description)); description != "" {
		b.WriteString("> " + description + "\n\n")
	}
	if kind := SiteKindByKey(m["siteKind"]); kind.Key != "" && kind.Key != "other" {
		b.WriteString("Kind of business: " + kind.Label + ".\n\n")
	}
	contact := []string{}
	if email := strings.TrimSpace(m["contactEmail"]); email != "" {
		contact = append(contact, "Email: "+email)
	}
	if phone := strings.TrimSpace(m["contactPhone"]); phone != "" {
		contact = append(contact, "Phone: "+phone)
	}
	if location := strings.TrimSpace(m["contactLocation"]); location != "" {
		contact = append(contact, "Address: "+location)
	}
	if len(contact) > 0 {
		b.WriteString("## Contact\n\n")
		for _, line := range contact {
			b.WriteString("- " + line + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Pages\n\n")
	for _, page := range h.livePages() {
		line := "- [" + page.Title + "](" + base + publicPath(page.Slug) + ")"
		if description := strings.TrimSpace(pageMetaValue(page, "metaDescription", page.Description)); description != "" {
			line += ": " + description
		} else if lead := firstParagraph(page.Body); lead != "" {
			line += ": " + lead
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")
	if h.featureOn(FeatureBlog) {
		if posts := h.livePosts(); len(posts) > 0 {
			b.WriteString("## Posts\n\n")
			for i, post := range posts {
				if i >= 20 {
					break
				}
				line := "- [" + post.Title + "](" + base + postPath(post.Slug) + ")"
				if post.Excerpt != "" {
					line += ": " + strings.TrimSpace(post.Excerpt)
				}
				b.WriteString(line + "\n")
			}
			b.WriteString("\n")
		}
	}
	if h.featureOn(FeatureShop) {
		active := []Product{}
		for _, product := range h.products.list() {
			if product.Active {
				active = append(active, product)
			}
		}
		if len(active) > 0 {
			b.WriteString("## Shop\n\n")
			for i, product := range active {
				if i >= 30 {
					break
				}
				b.WriteString("- [" + product.Name + "](" + base + shopPath + "/" + product.Slug + "): " + formatMoney(product.Price, h.currency()) + "\n")
			}
			b.WriteString("\n")
		}
	}
	b.WriteString("## For agents\n\n")
	b.WriteString("- Every page is also Markdown: add `?format=md` or send `Accept: text/markdown`.\n")
	b.WriteString("- All pages as one Markdown file: " + base + llmsFullPath + "\n")
	b.WriteString("- Agent API (read and edit with a key from the owner): " + base + agentAPIPrefix + " — OpenAPI at " + base + agentAPIPrefix + "/openapi.json\n")
	b.WriteString("- MCP endpoint: " + base + agentPathPrefix + "/mcp (Streamable HTTP, bearer key)\n")
	writeMarkdown(w, b.String())
}

func (h *Host) handleLLMsFull(w http.ResponseWriter, r *http.Request) {
	base := h.absoluteBase(r)
	var b strings.Builder
	settings := h.settings()
	b.WriteString("# " + firstNonEmpty(settings.Title, h.opts.SiteTitle) + "\n\n")
	for _, page := range h.livePages() {
		b.WriteString("<!-- " + base + publicPath(page.Slug) + " -->\n")
		b.WriteString(h.pageMarkdown(page, base))
		b.WriteString("\n---\n\n")
	}
	if h.featureOn(FeatureBlog) {
		for i, post := range h.livePosts() {
			if i >= 50 {
				break
			}
			b.WriteString("<!-- " + base + postPath(post.Slug) + " -->\n")
			b.WriteString(h.postMarkdown(post, base))
			b.WriteString("\n---\n\n")
		}
	}
	writeMarkdown(w, strings.TrimRight(b.String(), "\n-")+"\n")
}

// firstParagraph is the first plain sentence of a page, for summaries.
func firstParagraph(doc blockstudio.Document) string {
	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue
		}
		text := ""
		switch instance.Key {
		case content.BlockParagraph:
			text = instance.Values["text"].String
		default:
			if spec, ok := compositeByKey(instance.Key); ok && !spec.Live {
				text = firstNonEmpty(instance.Values["text"].String, instance.Values["headline"].String)
			}
		}
		if text = strings.TrimSpace(inlineToPlain(text)); text != "" {
			if len(text) > 160 {
				text = text[:157] + "…"
			}
			return text
		}
	}
	return ""
}

// ---------- schema ----------

type schemaField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label,omitempty"`
	Type     string   `json:"type"` // text, url, image, flag, choice
	Default  string   `json:"default,omitempty"`
	Choices  []string `json:"choices,omitempty"`
	Optional bool     `json:"optional,omitempty"`
}

type schemaKind struct {
	Kind     string        `json:"kind"`
	Label    string        `json:"label"`
	Blurb    string        `json:"blurb,omitempty"`
	Fields   []schemaField `json:"fields,omitempty"`
	Item     *schemaItem   `json:"items,omitempty"`
	Variants []variantSpec `json:"variants,omitempty"`
	Live     bool          `json:"live,omitempty"`
	Example  any           `json:"example,omitempty"`
}

type schemaItem struct {
	Name     string        `json:"name"`
	Fields   []schemaField `json:"fields"`
	MaxItems int           `json:"maxItems,omitempty"`
}

func choiceKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sortStrings(keys)
	return keys
}

func sortStrings(keys []string) {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
}

func partType(kind partKind) string {
	switch kind {
	case partURL:
		return "url"
	case partImage:
		return "image"
	case partFlag:
		return "flag"
	}
	return "text"
}

// agentSchema is what an agent needs to compose a page: every block kind,
// its fields, its repeated items, and every choice that has a fixed set.
func (h *Host) agentSchema() map[string]any {
	kinds := []schemaKind{}
	simple := func(kind, label, blurb string, fields ...schemaField) {
		kinds = append(kinds, schemaKind{Kind: kind, Label: label, Blurb: blurb, Fields: fields})
	}
	alignField := schemaField{Key: "align", Label: "Alignment", Type: "choice", Choices: append([]string{""}, choiceKeys(textAligns)...), Optional: true}
	simple("heading", "Heading", "A section title.", schemaField{Key: "text", Label: "Text", Type: "text"}, schemaField{Key: "level", Label: "Level", Type: "choice", Choices: []string{"2", "3"}, Default: "2", Optional: true}, alignField)
	simple("paragraph", "Text", "A paragraph. Inline marks: **bold**, _italic_, [label](href), and line breaks.", schemaField{Key: "text", Label: "Text", Type: "text"}, alignField)
	simple("quote", "Quote", "A customer's words.", schemaField{Key: "text", Label: "Text", Type: "text"}, alignField)
	simple("list", "List", "Bullet points, one per line.", schemaField{Key: "text", Label: "Lines", Type: "text"})
	simple("button", "Button", "Sends people somewhere.", schemaField{Key: "text", Label: "Label", Type: "text"}, schemaField{Key: "url", Label: "Link", Type: "url"}, schemaField{Key: "look", Label: "Look", Type: "choice", Choices: choiceKeys(buttonLooks), Default: "primary", Optional: true}, alignField)
	simple("image", "Picture", "One picture with optional caption and link.", schemaField{Key: "url", Label: "Picture", Type: "image"}, schemaField{Key: "alt", Label: "Description", Type: "text", Optional: true}, schemaField{Key: "size", Type: "choice", Choices: choiceKeys(imageSizes), Default: "full", Optional: true}, schemaField{Key: "shape", Type: "choice", Choices: choiceKeys(imageShapes), Default: "natural", Optional: true}, schemaField{Key: "caption", Type: "text", Optional: true}, schemaField{Key: "link", Type: "url", Optional: true})
	simple("gallery", "Gallery", "Several pictures; send them in \"images\" as [{url, alt}].", schemaField{Key: "images", Label: "Pictures", Type: "list of {url, alt}"}, schemaField{Key: "style", Type: "choice", Choices: choiceKeys(galleryStyles), Default: "grid", Optional: true})
	simple("video", "Video", "A YouTube or Vimeo link.", schemaField{Key: "url", Label: "Link", Type: "url"}, schemaField{Key: "text", Label: "Caption", Type: "text", Optional: true})
	simple("columns", "Columns", "Two or three columns of text.", schemaField{Key: "text", Label: "First column", Type: "text"}, schemaField{Key: "text2", Label: "Second column", Type: "text"}, schemaField{Key: "text3", Label: "Third column", Type: "text", Optional: true}, schemaField{Key: "count", Type: "choice", Choices: choiceKeys(columnCounts), Default: "2", Optional: true})
	simple("divider", "Divider", "A thin line.")
	simple("section", "Section break", "Starts a band that styles every block after it until the next break.", schemaField{Key: "style", Type: "choice", Choices: choiceKeys(sectionStyles), Default: "plain"}, schemaField{Key: "align", Type: "choice", Choices: choiceKeys(sectionAligns), Default: "left", Optional: true}, schemaField{Key: "width", Type: "choice", Choices: choiceKeys(sectionWidths), Default: "normal", Optional: true}, schemaField{Key: "space", Type: "choice", Choices: choiceKeys(sectionSpaces), Default: "normal", Optional: true}, schemaField{Key: "url", Label: "Background picture (style image)", Type: "image", Optional: true}, schemaField{Key: "anchor", Label: "Jump-to name", Type: "text", Optional: true})
	simple("form", "Form", "A form; \"form\" is an id from GET /agent/v1/forms (\"contact\" is built in).", schemaField{Key: "form", Label: "Form id", Type: "text", Default: "contact"})
	simple("product", "Product", "One product from the shop; \"product\" is its id or address.", schemaField{Key: "product", Label: "Product id", Type: "text"})
	for _, spec := range composites {
		kind := schemaKind{Kind: spec.Key, Label: spec.Label, Blurb: spec.Blurb, Variants: spec.Variants, Live: spec.Live}
		for _, field := range spec.Fields {
			kind.Fields = append(kind.Fields, schemaField{Key: field.Key, Label: field.Label, Type: partType(field.Kind), Default: field.Default, Optional: true})
		}
		if spec.Item != nil {
			item := &schemaItem{Name: spec.ItemName, MaxItems: spec.MaxItems}
			for _, field := range spec.Item {
				item.Fields = append(item.Fields, schemaField{Key: field.Key, Label: field.Label, Type: partType(field.Kind), Default: field.Default, Optional: true})
			}
			kind.Item = item
		}
		if !spec.Live {
			example := map[string]any{"kind": spec.Key}
			if len(spec.Fields) > 0 {
				fields := map[string]string{}
				for _, field := range spec.Fields {
					fields[field.Key] = field.Default
				}
				example["fields"] = fields
			}
			if spec.Item != nil && len(spec.ItemDefaults) > 0 {
				example["items"] = spec.ItemDefaults
			}
			if len(spec.Variants) > 0 {
				example["variant"] = spec.Variants[0].Key
			}
			kind.Example = example
		}
		kinds = append(kinds, kind)
	}
	templates := []map[string]string{}
	for _, template := range PageTemplates() {
		templates = append(templates, map[string]string{"key": template.Key, "label": template.Label, "blurb": template.Blurb})
	}
	siteKinds := []map[string]string{}
	for _, kind := range SiteKinds() {
		siteKinds = append(siteKinds, map[string]string{"key": kind.Key, "label": kind.Label, "blurb": kind.Blurb})
	}
	starters := []map[string]string{}
	for _, template := range Templates() {
		starters = append(starters, map[string]string{"key": template.Key, "label": template.Label, "kind": template.Kind, "blurb": template.Blurb})
	}
	palettes := []map[string]string{}
	for _, palette := range Palettes() {
		palettes = append(palettes, map[string]string{"key": palette.Key, "label": palette.Label, "blurb": palette.Blurb, "scheme": palette.Scheme})
	}
	fonts := []map[string]string{}
	for _, pair := range FontPairs() {
		fonts = append(fonts, map[string]string{"key": pair.Key, "label": pair.Label, "blurb": pair.Blurb})
	}
	keysOf := func(items ...string) []string { return items }
	buttons, spacing, headings, widths := []string{}, []string{}, []string{}, []string{}
	for _, shape := range ButtonShapes() {
		buttons = append(buttons, shape.Key)
	}
	for _, scale := range SpacingScales() {
		spacing = append(spacing, scale.Key)
	}
	for _, scale := range HeadingScales() {
		headings = append(headings, scale.Key)
	}
	for _, width := range PageWidths() {
		widths = append(widths, width.Key)
	}
	return map[string]any{
		"version":    agentAPIVersion,
		"model":      "A page is an ordered list of blocks. A \"section\" block starts a styled band that holds every block after it until the next section. Ready-made sections have named \"fields\" and, when they repeat, \"items\". Text accepts inline marks: **bold**, _italic_, [label](href), and newlines.",
		"blockKinds": kinds,
		"common": map[string]any{
			"spacing": schemaField{Key: "spacing", Label: "Room around the block", Type: "choice", Choices: append([]string{""}, choiceKeys(blockSpacings)...), Optional: true},
			"phone":   schemaField{Key: "phone", Label: "Hide on phones", Type: "choice", Choices: keysOf("", phoneHide), Optional: true},
			"locked":  schemaField{Key: "locked", Label: "Only admins may change it", Type: "choice", Choices: keysOf("", "true"), Optional: true},
		},
		"pageTemplates": templates,
		"siteKinds":     siteKinds,
		"starters":      starters,
		"look": map[string]any{
			"palettes": palettes, "fonts": fonts, "buttons": buttons, "spacing": spacing, "headings": headings, "widths": widths,
			"custom": "palette \"custom\" uses ground and ink (hex colours); fonts \"custom\" uses fontHead and fontBody (Google Fonts family names); accent is a hex colour on any palette.",
		},
		"pageActions": []string{"offline", "online", "archive", "restore", "hide", "show", "up", "down"},
		"scopes":      agentScopeBlurbs,
	}
}

func (h *Host) handleAgentSchema(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, h.agentSchema())
}

// ---------- OpenAPI ----------

func (h *Host) handleAgentOpenAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, h.openAPI(r))
}

func (h *Host) openAPI(r *http.Request) map[string]any {
	ref := func(name string) map[string]any { return map[string]any{"$ref": "#/components/schemas/" + name} }
	jsonBody := func(schema any) map[string]any {
		return map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
	}
	response := func(description string, schema any) map[string]any {
		if schema == nil {
			return map[string]any{"description": description}
		}
		return map[string]any{"description": description, "content": map[string]any{"application/json": map[string]any{"schema": schema}}}
	}
	op := func(summary, scope string, body any, ok any, extra ...string) map[string]any {
		description := summary
		if len(extra) > 0 {
			description = extra[0]
		}
		out := map[string]any{"summary": summary, "description": description, "responses": map[string]any{"200": response("OK", ok), "4XX": response("Refused; see error.code and error.message", ref("Error"))}}
		if scope != "" {
			out["security"] = []map[string]any{{"agentKey": []string{}}}
			out["x-scope"] = scope
		}
		if body != nil {
			out["requestBody"] = jsonBody(body)
		}
		return out
	}
	idParam := []map[string]any{{"name": "id", "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": "The id, or the address (slug)."}}
	withID := func(o map[string]any) map[string]any { o["parameters"] = idParam; return o }
	str := map[string]any{"type": "string"}
	boolean := map[string]any{"type": "boolean"}
	block := ref("Block")
	blocks := map[string]any{"type": "array", "items": block}
	pageWrite := map[string]any{"type": "object", "properties": map[string]any{
		"title": str, "slug": str, "description": str, "navParent": map[string]any{"type": "string", "description": "Id of the menu page this one sits under, or empty."}, "template": map[string]any{"type": "string", "description": "A page template key from the schema (create only)."},
		"blocks": blocks, "publish": boolean,
		"replace": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"index": map[string]any{"type": "integer"}, "block": block}}},
		"remove":  map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
		"insert":  map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"index": map[string]any{"type": "integer"}, "block": block}}},
		"append":  blocks, "move": map[string]any{"type": "object", "properties": map[string]any{"from": map[string]any{"type": "integer"}, "to": map[string]any{"type": "integer"}}},
	}}
	paths := map[string]any{
		agentAPIPrefix:                          map[string]any{"get": op("What this API is and where things are", "", nil, ref("Index"))},
		agentAPIPrefix + "/schema":              map[string]any{"get": op("Every block kind, its fields, items, and choices", "", nil, map[string]any{"type": "object"})},
		agentAPIPrefix + "/setup":               map[string]any{"post": op("Build a site that has not been set up yet", scopeSettings, ref("Setup"), ref("Site"))},
		agentAPIPrefix + "/site":                map[string]any{"get": op("The site: name, contact, header, footer, Look, pages, menu", scopeRead, nil, ref("Site")), "patch": op("Change the site's name, description, contact, header, footer, social links, or custom CSS", scopeSettings, ref("SitePatch"), ref("Site"))},
		agentAPIPrefix + "/pages":               map[string]any{"get": op("List pages", scopeRead, nil, map[string]any{"type": "object", "properties": map[string]any{"pages": map[string]any{"type": "array", "items": ref("PageRow")}}}), "post": op("Create a page from a template or from blocks", scopeWrite, pageWrite, ref("Page"))},
		agentAPIPrefix + "/pages/{id}":          map[string]any{"get": withID(op("Read a page with its blocks", scopeRead, nil, ref("Page"))), "put": withID(op("Replace a page's blocks (and optionally its title, address, description)", scopeWrite, pageWrite, ref("Page"))), "patch": withID(op("Change part of a page: title, address, description, or block edits by index", scopeWrite, pageWrite, ref("Page")))},
		agentAPIPrefix + "/pages/{id}/publish":  map[string]any{"post": withID(op("Publish the page's saved draft", scopePublish, nil, ref("Page")))},
		agentAPIPrefix + "/pages/{id}/actions":  map[string]any{"post": withID(op("offline, online, archive, restore, hide, show, up, down", scopeWrite, map[string]any{"type": "object", "properties": map[string]any{"action": str}}, map[string]any{"type": "object"}))},
		agentAPIPrefix + "/pages/{id}/markdown": map[string]any{"get": withID(op("The page as Markdown", scopeRead, nil, nil))},
		agentAPIPrefix + "/posts":               map[string]any{"get": op("List blog posts", scopeRead, nil, map[string]any{"type": "object"}), "post": op("Create a post", scopeWrite, ref("PostWrite"), ref("Post"))},
		agentAPIPrefix + "/posts/{id}":          map[string]any{"get": withID(op("Read a post", scopeRead, nil, ref("Post"))), "put": withID(op("Change a post", scopeWrite, ref("PostWrite"), ref("Post")))},
		agentAPIPrefix + "/posts/{id}/publish":  map[string]any{"post": withID(op("Publish a post", scopePublish, nil, ref("Post")))},
		agentAPIPrefix + "/products":            map[string]any{"get": op("List products", scopeRead, nil, map[string]any{"type": "object"}), "post": op("Create a product", scopeWrite, ref("ProductWrite"), ref("Product"))},
		agentAPIPrefix + "/products/{id}":       map[string]any{"get": withID(op("Read a product", scopeRead, nil, ref("Product"))), "put": withID(op("Change a product", scopeWrite, ref("ProductWrite"), ref("Product"))), "delete": withID(op("Remove a product", scopeWrite, nil, map[string]any{"type": "object"}))},
		agentAPIPrefix + "/media":               map[string]any{"get": op("List uploaded pictures", scopeRead, nil, map[string]any{"type": "object"}), "post": op("Upload a picture: JSON {data: base64} or {url}, a multipart “file”, or a raw image body", scopeWrite, ref("MediaUpload"), map[string]any{"type": "object", "properties": map[string]any{"url": str, "absoluteUrl": str}})},
		agentAPIPrefix + "/look":                map[string]any{"get": op("The Look: palette, fonts, accent, buttons, spacing, headings, width", scopeRead, nil, ref("Look")), "put": op("Change any part of the Look (send only the keys to change)", scopeSettings, ref("Look"), ref("Look"))},
		agentAPIPrefix + "/presets":             map[string]any{"get": op("Saved section presets", scopeRead, nil, map[string]any{"type": "object"}), "post": op("Save a block as a preset", scopeWrite, map[string]any{"type": "object", "properties": map[string]any{"name": str, "block": block}}, map[string]any{"type": "object"})},
		agentAPIPrefix + "/presets/{id}":        map[string]any{"delete": withID(op("Forget a preset", scopeWrite, nil, map[string]any{"type": "object"}))},
		agentAPIPrefix + "/forms":               map[string]any{"get": op("Forms a page can hold", scopeRead, nil, map[string]any{"type": "object"})},
		agentAPIPrefix + "/messages":            map[string]any{"get": op("Messages visitors sent (newest first; ?limit=)", scopeRead, nil, map[string]any{"type": "object"})},
		agentAPIPrefix + "/stats":               map[string]any{"get": op("Visitor counts (?days=)", scopeRead, nil, map[string]any{"type": "object"})},
		agentAPIPrefix + "/activity":            map[string]any{"get": op("Recent changes by people and agents", scopeRead, nil, map[string]any{"type": "object"})},
		agentPathPrefix + "/mcp":                map[string]any{"post": op("Model Context Protocol (Streamable HTTP): the same operations as tools", scopeRead, map[string]any{"type": "object"}, map[string]any{"type": "object"})},
	}
	fieldProps := map[string]any{"type": "object", "additionalProperties": str}
	schemas := map[string]any{
		"Error": map[string]any{"type": "object", "properties": map[string]any{"error": map[string]any{"type": "object", "properties": map[string]any{"code": str, "message": str}}}},
		"Index": map[string]any{"type": "object"},
		"Block": map[string]any{"type": "object", "description": "One block, in the shape GET /agent/v1/schema describes. \"kind\" is required; simple blocks use text/url/alt/…; ready-made sections use fields, items, and variant.", "required": []string{"kind"}, "properties": map[string]any{
			"kind": str, "text": str, "text2": str, "text3": str, "level": str, "url": str, "alt": str, "style": str, "form": str, "product": str, "align": str, "look": str, "count": str, "size": str, "shape": str, "caption": str, "link": str,
			"width": str, "space": str, "anchor": str, "images": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"url": str, "alt": str}}},
			"fields": fieldProps, "items": map[string]any{"type": "array", "items": fieldProps}, "variant": str, "spacing": str, "phone": str, "locked": str,
		}},
		"PageRow": map[string]any{"type": "object", "properties": map[string]any{"id": str, "title": str, "slug": str, "url": str, "editUrl": str, "status": map[string]any{"type": "string", "enum": []string{"published", "draft"}}, "live": boolean, "hiddenFromMenu": boolean, "offline": boolean, "archived": boolean, "navParent": str, "updated": str, "blockCount": map[string]any{"type": "integer"}}},
		"Page":    map[string]any{"allOf": []any{ref("PageRow"), map[string]any{"type": "object", "properties": map[string]any{"description": str, "blocks": blocks, "checks": map[string]any{"type": "array", "items": str}}}}},
		"Setup":   map[string]any{"type": "object", "required": []string{"title", "kind"}, "properties": map[string]any{"title": str, "tagline": str, "kind": map[string]any{"type": "string", "description": "A site kind key from the schema."}, "template": str, "email": str, "phone": str, "location": str}},
		"Site":    map[string]any{"type": "object"},
		"SitePatch": map[string]any{"type": "object", "properties": map[string]any{"title": str, "tagline": str, "description": str, "kind": str, "contact": map[string]any{"type": "object", "properties": map[string]any{"email": str, "phone": str, "location": str}},
			"header": map[string]any{"type": "object", "properties": map[string]any{"announceText": str, "announceLink": str, "announceOn": boolean, "sticky": boolean, "menuButton": str, "menuButtonTo": str}},
			"footer": map[string]any{"type": "object", "properties": map[string]any{"menu": boolean, "links": map[string]any{"type": "array", "items": map[string]any{"type": "array", "items": str, "minItems": 2, "maxItems": 2}}}},
			"social": map[string]any{"type": "object", "additionalProperties": str, "description": "instagram, facebook, tiktok, youtube, x, linkedin"}, "customCss": str}},
		"PostWrite":    map[string]any{"type": "object", "properties": map[string]any{"title": str, "slug": str, "excerpt": str, "author": str, "tags": map[string]any{"type": "array", "items": str}, "blocks": blocks, "publish": boolean}},
		"Post":         map[string]any{"type": "object"},
		"ProductWrite": map[string]any{"type": "object", "properties": map[string]any{"name": str, "slug": str, "description": str, "price": map[string]any{"type": "string", "description": "A decimal such as 12.50, in the site's currency."}, "compare": str, "images": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"url": str, "alt": str}}}, "stock": map[string]any{"type": "integer"}, "trackStock": boolean, "ships": boolean, "active": boolean, "kind": map[string]any{"type": "string", "enum": []string{"physical", "digital", "subscription", "booking"}}}},
		"Product":      map[string]any{"type": "object"},
		"MediaUpload":  map[string]any{"type": "object", "properties": map[string]any{"data": map[string]any{"type": "string", "description": "Base64 image bytes, with or without a data: prefix."}, "url": map[string]any{"type": "string", "description": "Or an http(s) address to fetch and keep."}}},
		"Look":         map[string]any{"type": "object", "properties": map[string]any{"palette": str, "fonts": str, "accent": str, "buttons": str, "spacing": str, "headings": str, "width": str, "ground": str, "ink": str, "fontHead": str, "fontBody": str}},
	}
	return map[string]any{
		"openapi":  "3.1.0",
		"info":     map[string]any{"title": firstNonEmpty(h.settings().Title, h.opts.SiteTitle) + " — agent API", "version": agentAPIVersion, "description": "Read and change this website the way its editor does. Every write is a draft until published. Scopes: read, write, publish, settings."},
		"servers":  []map[string]any{{"url": h.absoluteBase(r)}},
		"security": []map[string]any{{"agentKey": []string{}}},
		"components": map[string]any{
			"securitySchemes": map[string]any{"agentKey": map[string]any{"type": "http", "scheme": "bearer", "description": "An agent key (gsk_…) an admin created under Agents."}},
			"schemas":         schemas,
		},
		"paths": paths,
	}
}

func itoaInt(n int) string { return strconv.Itoa(n) }
