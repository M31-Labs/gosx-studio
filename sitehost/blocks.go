package sitehost

import (
	"net/url"
	"regexp"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	"m31labs.dev/gosx-studio/cms/render"
)

// blocks.go renders a page's blocks for visitors.
//
// Studio's cms/render draws the block kinds its catalog knows and treats text
// as plain. The default host's editor produces inline formatting, lists,
// dividers, and section breaks that the catalog does not have, so the host
// renders its documents itself and calls back into cms/render's hooks for the
// two kinds that already have host-owned rendering: pictures and the contact
// form. Anything it does not recognise falls through to cms/render, so a
// document written by a different Studio host still shows.

// Block kinds this host adds beyond cms/content's catalog.
const (
	blockList    = "list"
	blockDivider = "divider"
	blockSection = "section"
	blockVideo   = "video"
	blockColumns = "columns"
)

const gallerySizes = "(max-width: 720px) 50vw, 360px"

// A block can be kept off small screens: a wide table of prices, a video
// that would swamp a phone. The value lives on the block itself.
const (
	phoneKey  = "phone"
	phoneHide = "hide"
)

var (
	youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{6,20}$`)
	vimeoID   = regexp.MustCompile(`^[0-9]{5,15}$`)
)

// videoEmbedURL turns a YouTube or Vimeo link a person would paste into the
// privacy-enhanced player address, or reports that it is neither.
func videoEmbedURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	host := strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
	path := strings.Trim(parsed.Path, "/")
	switch host {
	case "youtube.com", "m.youtube.com", "youtube-nocookie.com":
		id := parsed.Query().Get("v")
		if id == "" {
			parts := strings.Split(path, "/")
			if len(parts) == 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "live") {
				id = parts[1]
			}
		}
		if youtubeID.MatchString(id) {
			return "https://www.youtube-nocookie.com/embed/" + id, true
		}
	case "youtu.be":
		if youtubeID.MatchString(path) {
			return "https://www.youtube-nocookie.com/embed/" + path, true
		}
	case "vimeo.com", "player.vimeo.com":
		parts := strings.Split(path, "/")
		id := parts[len(parts)-1]
		if vimeoID.MatchString(id) {
			return "https://player.vimeo.com/video/" + id, true
		}
	}
	return "", false
}

// galleryImages reads the images list in cms/content's own shape.
func galleryImages(instance blockstudio.BlockInstance) [][2]string {
	out := make([][2]string, 0, len(instance.Values["images"].List))
	for _, item := range instance.Values["images"].List {
		urlValue := item.Object["url"]
		url := strings.TrimSpace(urlValue.String)
		if urlValue.Media != nil && strings.TrimSpace(urlValue.Media.URL) != "" {
			url = strings.TrimSpace(urlValue.Media.URL)
		}
		if url == "" {
			continue
		}
		out = append(out, [2]string{url, strings.TrimSpace(item.Object["alt"].String)})
	}
	return out
}

func galleryValue(images [][2]string) blockstudio.Value {
	list := make([]blockstudio.Value, 0, len(images))
	for _, image := range images {
		if strings.TrimSpace(image[0]) == "" {
			continue
		}
		list = append(list, blockstudio.Value{Object: map[string]blockstudio.Value{
			"url": text(strings.TrimSpace(image[0])),
			"alt": text(strings.TrimSpace(image[1])),
		}})
	}
	return blockstudio.Value{List: list}
}

// sectionStyles are the backgrounds a section break can choose.
var sectionStyles = map[string]bool{"plain": true, "tinted": true, "accent": true, "dark": true, "image": true}

func normalizeSectionStyle(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if sectionStyles[value] {
		return value
	}
	return "plain"
}

// renderBody renders a document as sections of blocks.
func (h *Host) renderBody(doc blockstudio.Document, hooks render.Hooks) gosx.Node {
	type section struct {
		options sectionOptions
		blocks  []gosx.Node
	}
	sections := []section{{options: sectionOptions{Style: "plain", Align: "left", Width: "normal", Space: "normal"}}}
	current := &sections[0]

	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue // kept in the document, not shown
		}
		if instance.Key == blockSection {
			sections = append(sections, section{options: sectionOptionsOf(instance)})
			current = &sections[len(sections)-1]
			continue
		}
		if node, ok := h.renderBlock(instance, hooks); ok {
			if instance.Values[phoneKey].String == phoneHide {
				node = gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-no-phone")), node)
			}
			current.blocks = append(current.blocks, node)
		}
	}

	out := make([]gosx.Node, 0, len(sections))
	for _, s := range sections {
		if len(s.blocks) == 0 {
			continue
		}
		attrs := []any{gosx.Attr("class", s.options.classes())}
		if s.options.Anchor != "" {
			attrs = append(attrs, gosx.Attr("id", s.options.Anchor))
		}
		if s.options.Style == "image" && s.options.Image != "" {
			attrs = append(attrs, gosx.Attr("style", "--section-image: url('"+cssURL(s.options.Image)+"')"))
		}
		out = append(out, gosx.El("section", gosx.Attrs(attrs...),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-section__inner")), gosx.Fragment(s.blocks...))))
	}
	return gosx.Fragment(out...)
}

func (h *Host) renderBlock(instance blockstudio.BlockInstance, hooks render.Hooks) (gosx.Node, bool) {
	text := strings.TrimSpace(instance.Values["text"].String)
	switch instance.Key {
	case content.BlockHeading:
		if text == "" {
			return gosx.Fragment(), false
		}
		return gosx.El("h"+content.NormalizeHeadingLevel(instance.Values["level"].String), gosx.Attrs(alignClass(instance)...), renderInline(text)), true
	case content.BlockParagraph:
		if text == "" {
			return gosx.Fragment(), false
		}
		return gosx.El("p", gosx.Attrs(alignClass(instance)...), renderInline(text)), true
	case content.BlockQuote:
		if text == "" {
			return gosx.Fragment(), false
		}
		return gosx.El("blockquote", gosx.Attrs(alignClass(instance)...), renderInline(text)), true
	case content.BlockButton:
		label := strings.TrimSpace(instance.Values["label"].String)
		href := safeLinkHref(instance.Values["href"].String)
		if label == "" || href == "" {
			return gosx.Fragment(), false
		}
		look := normalizeChoice(instance.Values["look"].String, "primary", buttonLooks)
		button := gosx.El("a", gosx.Attrs(append([]any{gosx.Attr("class", "button button--"+look)}, linkAttrs(href)...)...), gosx.Text(label))
		if align := normalizeChoice(instance.Values["align"].String, "", textAligns); align != "" {
			return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-button-row site-align-"+align)), button), true
		}
		return button, true
	case blockList:
		items := make([]gosx.Node, 0, 8)
		for _, line := range strings.Split(text, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				items = append(items, gosx.El("li", nil, renderInline(line)))
			}
		}
		if len(items) == 0 {
			return gosx.Fragment(), false
		}
		return gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-list")), gosx.Fragment(items...)), true
	case blockDivider:
		return gosx.El("hr", gosx.Attrs(gosx.Attr("class", "site-divider"))), true
	case blockVideo:
		embed, ok := videoEmbedURL(instance.Values["url"].String)
		if !ok {
			return gosx.Fragment(), false
		}
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-video")),
			gosx.El("iframe", gosx.Attrs(
				gosx.Attr("src", embed),
				gosx.Attr("title", firstNonEmpty(text, "Video")),
				gosx.Attr("loading", "lazy"),
				gosx.Attr("allow", "accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"),
				gosx.Attr("allowfullscreen", "allowfullscreen"),
				gosx.Attr("referrerpolicy", "strict-origin-when-cross-origin"),
			)),
		), true
	case blockColumns:
		left := strings.TrimSpace(instance.Values["text"].String)
		right := strings.TrimSpace(instance.Values["text2"].String)
		third := strings.TrimSpace(instance.Values["text3"].String)
		count := normalizeChoice(instance.Values["count"].String, "2", columnCounts)
		if left == "" && right == "" && third == "" {
			return gosx.Fragment(), false
		}
		cols := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns__col")), renderInline(left)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns__col")), renderInline(right)),
		}
		if count == "3" {
			cols = append(cols, gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns__col")), renderInline(third)))
		}
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-columns site-columns--"+count)), gosx.Fragment(cols...)), true
	case content.BlockGallery:
		images := galleryImages(instance)
		if len(images) == 0 {
			return gosx.Fragment(), false
		}
		items := make([]gosx.Node, 0, len(images))
		for _, image := range images {
			attrs, ok := h.imageAttrs(image[0], image[1], gallerySizes)
			if !ok {
				continue
			}
			items = append(items, gosx.El("figure", gosx.Attrs(gosx.Attr("class", "site-gallery__item")), gosx.El("img", gosx.Attrs(attrs...))))
		}
		if len(items) == 0 {
			return gosx.Fragment(), false
		}
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-gallery")), gosx.Fragment(items...)), true
	}

	if spec, ok := compositeByKey(instance.Key); ok {
		return h.renderComposite(spec, instance, false), true
	}
	if instance.Key == content.BlockImage {
		return h.renderPicture(instance)
	}

	// Pictures and the contact form: cms/render's hooks, exactly as before.
	view := content.ViewBlocksFromDocument(blockstudio.Document{Version: 1, Kind: "body", Blocks: []blockstudio.BlockInstance{instance}})
	if len(view) == 0 {
		return gosx.Fragment(), false
	}
	node, ok := render.RenderBlock(view[0], hooks)
	return node, ok
}

// blockPlainText is a block's text with formatting markers removed.
func blockPlainText(instance blockstudio.BlockInstance) string {
	return inlineToPlain(strings.TrimSpace(instance.Values["text"].String))
}

// Picture options: how big and what shape.
var (
	imageSizes  = map[string]bool{"full": true, "wide": true, "medium": true, "small": true}
	imageShapes = map[string]bool{"natural": true, "wide": true, "square": true, "round": true}
)

// renderPicture draws a picture block with its size, shape, caption, and
// link, or nothing when there is no picture.
func (h *Host) renderPicture(instance blockstudio.BlockInstance) (gosx.Node, bool) {
	alt := strings.TrimSpace(instance.Values["alt"].String)
	attrs, ok := h.imageAttrs(instance.Values["url"].String, alt, mediaSizes)
	if !ok {
		return gosx.Fragment(), false
	}
	size := normalizeChoice(instance.Values["size"].String, "full", imageSizes)
	shape := normalizeChoice(instance.Values["shape"].String, "natural", imageShapes)
	var picture gosx.Node = gosx.El("img", gosx.Attrs(attrs...))
	if href := safeLinkHref(instance.Values["link"].String); href != "" {
		picture = gosx.El("a", gosx.Attrs(linkAttrs(href)...), picture)
	}
	nodes := []gosx.Node{picture}
	if caption := strings.TrimSpace(instance.Values["caption"].String); caption != "" {
		nodes = append(nodes, gosx.El("figcaption", nil, renderInline(caption)))
	}
	return gosx.El("figure", gosx.Attrs(gosx.Attr("class", "site-figure site-figure--"+size+" site-figure--crop-"+shape)), gosx.Fragment(nodes...)), true
}

// Text alignment, button looks, and column counts an owner can pick.
var (
	textAligns   = map[string]bool{"center": true, "right": true}
	buttonLooks  = map[string]bool{"primary": true, "ghost": true, "link": true}
	columnCounts = map[string]bool{"2": true, "3": true}
)

func alignClass(instance blockstudio.BlockInstance) []any {
	if align := normalizeChoice(instance.Values["align"].String, "", textAligns); align != "" {
		return []any{gosx.Attr("class", "site-align-"+align)}
	}
	return nil
}
