package sitehost

import (
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
)

// pagetemplates.go is what "Add a page" starts from. A blank page is one
// option; the others are whole layouts built from the ready-made sections,
// with copy written to be replaced.

// PageTemplate is one starting layout.
type PageTemplate struct {
	Key, Label, Blurb string
	build             func(title string) blockstudio.Document
}

// PageTemplates are the layouts, blank first.
func PageTemplates() []PageTemplate {
	return []PageTemplate{
		{Key: "blank", Label: "Blank", Blurb: "One line of text to replace.", build: func(string) blockstudio.Document {
			return document(block(0, content.BlockParagraph, values("text", "Write your page here.")))
		}},
		{Key: "landing", Label: "Landing page", Blurb: "Hero, three features, numbers, testimonials, a call to action.", build: func(title string) blockstudio.Document {
			return document(
				comp(0, "hero", "center", map[string]string{"eyebrow": title, "headline": "The one thing you want people to know", "text": "A sentence or two on what you do and who it's for.", "button": "Get in touch", "url": "/contact"}, nil),
				comp(1, "features", "cards", map[string]string{"heading": "What you get"}, nil),
				sectionBreak(2, "tinted", "center", "normal", "normal"),
				comp(3, "stats", "row", nil, nil),
				sectionBreak(4, "plain", "left", "normal", "normal"),
				comp(5, "testimonials", "grid", map[string]string{"heading": "Kind words"}, nil),
				comp(6, "cta", "band", map[string]string{"headline": "Ready when you are", "text": "One line on what happens when they get in touch.", "button": "Get in touch", "url": "/contact"}, nil),
			)
		}},
		{Key: "about", Label: "About", Blurb: "Your story, the people, and what you stand for.", build: func(title string) blockstudio.Document {
			return document(
				comp(0, "imagetext", "right", map[string]string{"heading": "How it started", "text": "Who you are and how you got here. People buy from people, so write this the way you'd tell a customer standing in front of you."}, nil),
				comp(1, "features", "plain", map[string]string{"heading": "What we care about"}, []map[string]string{{"icon": "1", "title": "The first value", "text": "What it means in practice."}, {"icon": "2", "title": "The second", "text": "What it means in practice."}, {"icon": "3", "title": "The third", "text": "What it means in practice."}}),
				sectionBreak(2, "tinted", "left", "normal", "normal"),
				comp(3, "team", "grid", map[string]string{"heading": "Who you'll meet"}, nil),
			)
		}},
		{Key: "services", Label: "Services", Blurb: "What you offer, how it works, and pricing.", build: func(title string) blockstudio.Document {
			return document(
				comp(0, "hero", "left", map[string]string{"eyebrow": "Services", "headline": "What we do", "text": "Describe each service in a sentence or two. Say what the client gets, not how you do it.", "button": "Ask about a job", "url": "/contact"}, nil),
				comp(1, "features", "cards", map[string]string{"heading": "How we can help"}, nil),
				sectionBreak(2, "tinted", "left", "normal", "normal"),
				comp(3, "faq", "accordion", map[string]string{"heading": "How it works"}, []map[string]string{{"question": "What happens first?", "answer": "Walk through what happens after someone gets in touch. Removing the mystery is the most persuasive thing on most service websites."}, {"question": "How long does it take?", "answer": "Be honest about timings."}, {"question": "What does it cost?", "answer": "If you can name a price or a range, do. It saves everyone time."}}),
				sectionBreak(4, "plain", "left", "normal", "normal"),
				comp(5, "pricing", "cards", map[string]string{"heading": "Pricing"}, nil),
				comp(6, "cta", "band", map[string]string{"headline": "Tell us about the job", "text": "We reply within a working day.", "button": "Get a quote", "url": "/contact"}, nil),
			)
		}},
		{Key: "pricing", Label: "Pricing", Blurb: "Plans side by side, with questions answered.", build: func(title string) blockstudio.Document {
			return document(
				comp(0, "pricing", "cards", map[string]string{"heading": "Simple pricing", "intro": "Pick the one that fits. Change any time."}, nil),
				sectionBreak(1, "tinted", "left", "normal", "normal"),
				comp(2, "faq", "accordion", map[string]string{"heading": "Questions about pricing"}, []map[string]string{{"question": "Can I change plans later?", "answer": "Yes, whenever you like."}, {"question": "Is there a contract?", "answer": "No. Monthly, cancel any time."}}),
			)
		}},
		{Key: "faq", Label: "Questions and answers", Blurb: "The questions people always ask, answered once.", build: func(title string) blockstudio.Document {
			return document(
				comp(0, "faq", "accordion", map[string]string{"heading": title}, nil),
				comp(1, "cta", "plain", map[string]string{"headline": "Still wondering?", "text": "Ask us anything.", "button": "Get in touch", "url": "/contact"}, nil),
			)
		}},
		{Key: "team", Label: "People", Blurb: "The faces behind the business.", build: func(title string) blockstudio.Document {
			return document(
				comp(0, "team", "grid", map[string]string{"heading": title, "intro": "A line on who you are as a group."}, nil),
				sectionBreak(1, "tinted", "center", "normal", "normal"),
				comp(2, "cta", "plain", map[string]string{"headline": "Come and work with us", "text": "Say what you're looking for.", "button": "Get in touch", "url": "/contact"}, nil),
			)
		}},
		{Key: "visit", Label: "Visit us", Blurb: "Opening hours, a map, and how to find the door.", build: func(title string) blockstudio.Document {
			return document(
				comp(0, "hours", "table", map[string]string{"heading": "When we're open"}, nil),
				comp(1, "map", "wide", map[string]string{"address": "Your address", "note": "How to find the door, where to park, the nearest stop."}, nil),
				comp(2, "cta", "plain", map[string]string{"headline": "Questions before you come?", "text": "We're happy to help.", "button": "Get in touch", "url": "/contact"}, nil),
			)
		}},
		{Key: "menu", Label: "Menu or price list", Blurb: "Headings and lines for what you serve or sell.", build: func(title string) blockstudio.Document {
			return document(
				heading(0, 2, title),
				para(1, "A line about what you're known for, and what you can do for allergies."),
				heading(2, 3, "Mornings"),
				block(3, blockList, values("text", "Sourdough loaf — $6\nCinnamon bun — $4\nFlat white — $4")),
				heading(4, 3, "Afternoons"),
				block(5, blockList, values("text", "Sandwich of the day — $9\nSoup and bread — $8")),
				comp(6, "hours", "inline", map[string]string{"heading": "When we're open"}, nil),
			)
		}},
	}
}

// PageTemplateByKey finds a layout; blank when unknown.
func PageTemplateByKey(key string) PageTemplate {
	key = strings.ToLower(strings.TrimSpace(key))
	all := PageTemplates()
	for _, template := range all {
		if template.Key == key {
			return template
		}
	}
	return all[0]
}

// comp builds one ready-made section with the given field values and
// items, falling back to the section's starter items.
func comp(order int, key, variant string, fields map[string]string, items []map[string]string) blockstudio.BlockInstance {
	spec, ok := compositeByKey(key)
	if !ok {
		return para(order, "")
	}
	instance := freshComposite(spec, order)
	for field, value := range fields {
		instance.Values[field] = text(value)
	}
	if variant != "" {
		instance.Values["variant"] = text(spec.variant(variant))
	}
	if items != nil {
		instance.Values["items"] = itemsValue(spec, items)
	}
	return instance
}

func sectionBreak(order int, style, align, width, space string) blockstudio.BlockInstance {
	return block(order, blockSection, sectionOptions{Style: normalizeSectionStyle(style), Align: align, Width: width, Space: space}.values())
}
