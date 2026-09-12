package sitehost

import (
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
)

// sectionStyles are the backgrounds a section break can choose.
var sectionStyles = map[string]bool{"plain": true, "tinted": true, "accent": true}

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
		style  string
		blocks []gosx.Node
	}
	sections := []section{{style: "plain"}}
	current := &sections[0]

	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue // kept in the document, not shown
		}
		if instance.Key == blockSection {
			sections = append(sections, section{style: normalizeSectionStyle(instance.Values["style"].String)})
			current = &sections[len(sections)-1]
			continue
		}
		if node, ok := h.renderBlock(instance, hooks); ok {
			current.blocks = append(current.blocks, node)
		}
	}

	out := make([]gosx.Node, 0, len(sections))
	for _, s := range sections {
		if len(s.blocks) == 0 {
			continue
		}
		out = append(out, gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-section site-section--"+s.style)),
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
		return gosx.El("h"+content.NormalizeHeadingLevel(instance.Values["level"].String), nil, renderInline(text)), true
	case content.BlockParagraph:
		if text == "" {
			return gosx.Fragment(), false
		}
		return gosx.El("p", nil, renderInline(text)), true
	case content.BlockQuote:
		if text == "" {
			return gosx.Fragment(), false
		}
		return gosx.El("blockquote", nil, renderInline(text)), true
	case content.BlockButton:
		label := strings.TrimSpace(instance.Values["label"].String)
		href := safeLinkHref(instance.Values["href"].String)
		if label == "" || href == "" {
			return gosx.Fragment(), false
		}
		return gosx.El("a", gosx.Attrs(append([]any{gosx.Attr("class", "button button--primary")}, linkAttrs(href)...)...), gosx.Text(label)), true
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
