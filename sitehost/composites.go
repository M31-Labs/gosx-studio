package sitehost

import (
	"net/http"
	"net/url"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
)

// composites.go is the set of ready-made sections an owner drops onto a
// page: a hero, feature cards, testimonials, pricing, questions and
// answers, a call to action, numbers, people, opening hours, a picture
// beside text, a map, and empty space.
//
// One table describes each of them. The same renderer draws the public
// page and the editing canvas, so what the owner edits is exactly what a
// visitor sees; the editing canvas just adds contenteditable, a few inline
// inputs, and add/remove buttons for repeated items. Nothing here needs a
// property panel.

// partKind says how a field is edited and stored.
type partKind string

const (
	partText  partKind = "text"  // inline-formatted text, edited in place
	partURL   partKind = "url"   // a link, edited in a small input
	partImage partKind = "image" // a picture, uploaded or picked
	partFlag  partKind = "flag"  // yes or no, a checkbox
)

type partSpec struct {
	Key, Label string
	Kind       partKind
	Default    string
}

type variantSpec struct{ Key, Label string }

type compositeSpec struct {
	Key, Label, Blurb string
	Fields            []partSpec
	Item              []partSpec // fields of one repeated item; nil when there are none
	ItemName          string     // "card", "quote", …
	ItemDefaults      []map[string]string
	MaxItems          int
	Variants          []variantSpec
	// Live sections draw from the site itself (posts, products) rather
	// than from fields, so they are never empty by the owner's doing.
	Live bool
}

func (s compositeSpec) variant(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, candidate := range s.Variants {
		if candidate.Key == value {
			return value
		}
	}
	if len(s.Variants) > 0 {
		return s.Variants[0].Key
	}
	return ""
}

func (s compositeSpec) field(key string) (partSpec, bool) {
	for _, field := range s.Fields {
		if field.Key == key {
			return field, true
		}
	}
	return partSpec{}, false
}

var composites = []compositeSpec{
	{Key: "hero", Label: "Hero", Blurb: "A big opening: headline, a line, and a button",
		Fields: []partSpec{
			{Key: "eyebrow", Label: "Small line above", Kind: partText, Default: "Welcome"},
			{Key: "headline", Label: "Headline", Kind: partText, Default: "Say the one thing you want people to know"},
			{Key: "text", Label: "Text", Kind: partText, Default: "A sentence or two that explains what you do and who it's for. Keep it short; the rest of the page can do the explaining."},
			{Key: "button", Label: "Button", Kind: partText, Default: "Get in touch"},
			{Key: "url", Label: "Button link", Kind: partURL, Default: "/contact"},
			{Key: "button2", Label: "Second button", Kind: partText},
			{Key: "url2", Label: "Second button link", Kind: partURL},
			{Key: "image", Label: "Picture", Kind: partImage},
		},
		Variants: []variantSpec{{"center", "Centred"}, {"left", "Text on the left"}, {"split", "Text beside the picture"}, {"cover", "Text over the picture"}}},
	{Key: "features", Label: "Feature cards", Blurb: "Three things you offer, side by side",
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "What we do"}, {Key: "intro", Label: "Intro", Kind: partText}},
		Item:     []partSpec{{Key: "icon", Label: "Icon or emoji", Kind: partText}, {Key: "title", Label: "Title", Kind: partText}, {Key: "text", Label: "Text", Kind: partText}},
		ItemName: "card", MaxItems: 6,
		ItemDefaults: []map[string]string{
			{"icon": "✦", "title": "The first thing", "text": "One or two sentences on what this is and why it matters to the customer."},
			{"icon": "✦", "title": "The second thing", "text": "Say what they get, not how you do it."},
			{"icon": "✦", "title": "The third thing", "text": "Three is plenty. Cut anything that doesn't earn its place."},
		},
		Variants: []variantSpec{{"cards", "Cards"}, {"plain", "Plain"}, {"numbered", "Numbered"}}},
	{Key: "testimonials", Label: "Testimonials", Blurb: "What customers say, in their words",
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "Kind words"}},
		Item:     []partSpec{{Key: "quote", Label: "Quote", Kind: partText}, {Key: "name", Label: "Name", Kind: partText}, {Key: "role", Label: "Who they are", Kind: partText}, {Key: "image", Label: "Photo", Kind: partImage}},
		ItemName: "quote", MaxItems: 6,
		ItemDefaults: []map[string]string{
			{"quote": "Swap this for something a real customer said. Short and specific beats long and glowing.", "name": "A happy customer", "role": "Regular since 2021"},
			{"quote": "Another voice. Two or three of these are more convincing than ten.", "name": "Someone else", "role": "Local business"},
		},
		Variants: []variantSpec{{"grid", "Side by side"}, {"single", "One big quote"}}},
	{Key: "pricing", Label: "Pricing", Blurb: "Plans or packages with a price each",
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "Simple pricing"}, {Key: "intro", Label: "Intro", Kind: partText}},
		Item:     []partSpec{{Key: "name", Label: "Name", Kind: partText}, {Key: "price", Label: "Price", Kind: partText}, {Key: "period", Label: "Per", Kind: partText}, {Key: "blurb", Label: "One line", Kind: partText}, {Key: "features", Label: "What's included, one per line", Kind: partText}, {Key: "button", Label: "Button", Kind: partText}, {Key: "url", Label: "Button link", Kind: partURL}, {Key: "highlight", Label: "Highlight this one", Kind: partFlag}},
		ItemName: "plan", MaxItems: 4,
		ItemDefaults: []map[string]string{
			{"name": "Starter", "price": "$20", "period": "per month", "blurb": "For getting going.", "features": "The first thing\nThe second thing", "button": "Choose Starter", "url": "/contact"},
			{"name": "Standard", "price": "$45", "period": "per month", "blurb": "The one most people pick.", "features": "Everything in Starter\nThe third thing\nPriority replies", "button": "Choose Standard", "url": "/contact", "highlight": "yes"},
			{"name": "Pro", "price": "$90", "period": "per month", "blurb": "For the busiest.", "features": "Everything in Standard\nSomething extra", "button": "Choose Pro", "url": "/contact"},
		},
		Variants: []variantSpec{{"cards", "Cards"}, {"simple", "Simple list"}}},
	{Key: "faq", Label: "Questions and answers", Blurb: "The questions people always ask",
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "Questions people ask"}},
		Item:     []partSpec{{Key: "question", Label: "Question", Kind: partText}, {Key: "answer", Label: "Answer", Kind: partText}},
		ItemName: "question", MaxItems: 12,
		ItemDefaults: []map[string]string{
			{"question": "How does it work?", "answer": "Answer the question the way you would in person: short, warm, and specific."},
			{"question": "How much does it cost?", "answer": "If you can say a number or a range, say it. It saves everyone a phone call."},
			{"question": "Where are you?", "answer": "Address, parking, the door that's easy to miss."},
		},
		Variants: []variantSpec{{"accordion", "Open one at a time"}, {"list", "All open"}}},
	{Key: "cta", Label: "Call to action", Blurb: "A band that asks people to do the thing",
		Fields: []partSpec{
			{Key: "headline", Label: "Headline", Kind: partText, Default: "Ready when you are"},
			{Key: "text", Label: "Text", Kind: partText, Default: "One line on what happens when they get in touch."},
			{Key: "button", Label: "Button", Kind: partText, Default: "Get in touch"},
			{Key: "url", Label: "Button link", Kind: partURL, Default: "/contact"},
		},
		Variants: []variantSpec{{"band", "Coloured band"}, {"plain", "Plain"}}},
	{Key: "stats", Label: "Numbers", Blurb: "A few big numbers that tell the story",
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText}},
		Item:     []partSpec{{Key: "value", Label: "Number", Kind: partText}, {Key: "label", Label: "What it counts", Kind: partText}},
		ItemName: "number", MaxItems: 6,
		ItemDefaults: []map[string]string{{"value": "12", "label": "years in business"}, {"value": "2,400", "label": "happy customers"}, {"value": "4.9★", "label": "average review"}},
		Variants:     []variantSpec{{"row", "In a row"}, {"cards", "Cards"}}},
	{Key: "team", Label: "People", Blurb: "The faces behind the business",
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "Who you'll meet"}, {Key: "intro", Label: "Intro", Kind: partText}},
		Item:     []partSpec{{Key: "image", Label: "Photo", Kind: partImage}, {Key: "name", Label: "Name", Kind: partText}, {Key: "role", Label: "Role", Kind: partText}, {Key: "bio", Label: "A line about them", Kind: partText}},
		ItemName: "person", MaxItems: 12,
		ItemDefaults: []map[string]string{{"name": "Your name", "role": "Founder", "bio": "A line about what you do here and what you love about it."}, {"name": "A colleague", "role": "What they do", "bio": "A line about them."}},
		Variants:     []variantSpec{{"grid", "Grid"}, {"list", "List"}}},
	{Key: "hours", Label: "Opening hours", Blurb: "When people can find you",
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "Opening hours"}, {Key: "note", Label: "A note", Kind: partText, Default: "Closed on public holidays."}},
		Item:     []partSpec{{Key: "day", Label: "Day", Kind: partText}, {Key: "time", Label: "Hours", Kind: partText}},
		ItemName: "day", MaxItems: 8,
		ItemDefaults: []map[string]string{{"day": "Monday to Friday", "time": "8am – 5pm"}, {"day": "Saturday", "time": "9am – 2pm"}, {"day": "Sunday", "time": "Closed"}},
		Variants:     []variantSpec{{"table", "Table"}, {"inline", "In a line"}}},
	{Key: "imagetext", Label: "Picture and text", Blurb: "A picture beside a heading, text, and a button",
		Fields: []partSpec{
			{Key: "image", Label: "Picture", Kind: partImage},
			{Key: "heading", Label: "Heading", Kind: partText, Default: "A heading for this part"},
			{Key: "text", Label: "Text", Kind: partText, Default: "A paragraph beside the picture. Say what the picture shows and why it matters."},
			{Key: "button", Label: "Button", Kind: partText},
			{Key: "url", Label: "Button link", Kind: partURL},
		},
		Variants: []variantSpec{{"right", "Picture on the right"}, {"left", "Picture on the left"}}},
	{Key: "map", Label: "Map", Blurb: "A map of your address",
		Fields:   []partSpec{{Key: "address", Label: "Address", Kind: partText, Default: "1 High Street, Your Town"}, {Key: "note", Label: "How to find the door", Kind: partText}},
		Variants: []variantSpec{{"wide", "Wide"}, {"compact", "Compact"}}},
	{Key: "spacer", Label: "Space", Blurb: "Empty room between things",
		Variants: []variantSpec{{"medium", "Medium"}, {"small", "Small"}, {"large", "Large"}}},
	{Key: "posts", Label: "Latest posts", Blurb: "Your newest blog posts, kept up to date by themselves", Live: true,
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "From the blog"}},
		Variants: []variantSpec{{"three", "The latest three"}, {"six", "The latest six"}, {"one", "Just the newest"}}},
	{Key: "products", Label: "From the shop", Blurb: "A few products, straight from your shop", Live: true,
		Fields:   []partSpec{{Key: "heading", Label: "Heading", Kind: partText, Default: "From the shop"}},
		Variants: []variantSpec{{"three", "Three products"}, {"six", "Six products"}, {"all", "Everything on sale"}}},
}

func compositeByKey(key string) (compositeSpec, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, spec := range composites {
		if spec.Key == key {
			return spec, true
		}
	}
	return compositeSpec{}, false
}

// compositeKeys is the comma-separated list the editor script reads.
func compositeKeys() string {
	keys := make([]string, 0, len(composites))
	for _, spec := range composites {
		keys = append(keys, spec.Key)
	}
	return strings.Join(keys, ",")
}

// ---------- storage ----------

// freshComposite is the block an owner gets from the sidebar.
func freshComposite(spec compositeSpec, order int) blockstudio.BlockInstance {
	values := blockstudio.Values{}
	for _, field := range spec.Fields {
		if field.Default != "" {
			values[field.Key] = text(field.Default)
		}
	}
	if len(spec.Variants) > 0 {
		values["variant"] = text(spec.Variants[0].Key)
	}
	if spec.Item != nil {
		values["items"] = itemsValue(spec, spec.ItemDefaults)
	}
	return blockstudio.BlockInstance{ID: spec.Key + "-" + itoa(order), Key: spec.Key, Enabled: true, Order: order, Values: values}
}

func itemsValue(spec compositeSpec, items []map[string]string) blockstudio.Value {
	list := make([]blockstudio.Value, 0, len(items))
	for _, item := range items {
		object := map[string]blockstudio.Value{}
		empty := true
		for _, field := range spec.Item {
			value := strings.TrimSpace(item[field.Key])
			if value != "" {
				empty = false
			}
			object[field.Key] = text(value)
		}
		if !empty {
			list = append(list, blockstudio.Value{Object: object})
		}
		if spec.MaxItems > 0 && len(list) >= spec.MaxItems {
			break
		}
	}
	return blockstudio.Value{List: list}
}

func compositeItems(spec compositeSpec, instance blockstudio.BlockInstance) []map[string]string {
	out := []map[string]string{}
	for _, entry := range instance.Values["items"].List {
		item := map[string]string{}
		for _, field := range spec.Item {
			item[field.Key] = strings.TrimSpace(entry.Object[field.Key].String)
		}
		out = append(out, item)
	}
	return out
}

// compositeFromPayload turns what the editor sent into a stored block, or
// nothing when the block is empty.
func compositeFromPayload(spec compositeSpec, incoming editorBlockPayload, order int) (blockstudio.BlockInstance, bool) {
	values := blockstudio.Values{}
	filled := (len(spec.Fields) == 0 && spec.Item == nil) || spec.Live
	for _, field := range spec.Fields {
		value := strings.TrimSpace(incoming.Fields[field.Key])
		if field.Kind == partFlag {
			if value != "" && value != "no" && value != "false" {
				value = "yes"
			} else {
				value = ""
			}
		}
		if value != "" {
			filled = true
		}
		values[field.Key] = text(value)
	}
	if spec.Item != nil {
		items := itemsValue(spec, incoming.Items)
		values["items"] = items
		if len(items.List) > 0 {
			filled = true
		}
	}
	if !filled {
		return blockstudio.BlockInstance{}, false
	}
	values["variant"] = text(spec.variant(incoming.Variant))
	return blockstudio.BlockInstance{ID: spec.Key + "-" + itoa(order), Key: spec.Key, Enabled: true, Order: order, Values: values}, true
}

// ---------- rendering, once for both places ----------

// canvas carries the mode through the renderer.
type canvas struct {
	h        *Host
	editable bool
}

func (c canvas) attrs(spec partSpec, extra ...any) []any {
	if !c.editable {
		return extra
	}
	return append(extra, gosx.Attr("data-field", spec.Key), gosx.Attr("data-text", "true"), gosx.Attr("contenteditable", "true"), gosx.Attr("spellcheck", "true"), gosx.Attr("data-placeholder", spec.Label))
}

// textNode is a text field: the element with its content, or nothing on
// the public page when it is empty.
func (c canvas) textNode(tag, class string, spec partSpec, value string) gosx.Node {
	value = strings.TrimSpace(value)
	if value == "" && !c.editable {
		return gosx.Fragment()
	}
	attrs := []any{}
	if class != "" {
		attrs = append(attrs, gosx.Attr("class", class))
	}
	return gosx.El(tag, gosx.Attrs(c.attrs(spec, attrs...)...), renderInline(value))
}

// linkNode is a button or link whose label is a text field and whose
// address is a url field.
func (c canvas) linkNode(class string, label partSpec, labelValue string, link partSpec, href string) gosx.Node {
	labelValue = strings.TrimSpace(labelValue)
	href = strings.TrimSpace(href)
	if c.editable {
		return gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-link-field")),
			gosx.El("a", gosx.Attrs(c.attrs(label, gosx.Attr("class", class), gosx.Attr("href", "#"))...), renderInline(labelValue)),
			gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("data-field", link.Key), gosx.Attr("list", "site-links"), gosx.Attr("value", href), gosx.Attr("placeholder", "/contact"), gosx.Attr("aria-label", link.Label), gosx.Attr("contenteditable", "false"))))
	}
	safe := safeLinkHref(href)
	if labelValue == "" || safe == "" {
		return gosx.Fragment()
	}
	return gosx.El("a", gosx.Attrs(append([]any{gosx.Attr("class", class)}, linkAttrs(safe)...)...), renderInline(labelValue))
}

// imageNode is a picture field: the image with a picker on the canvas.
func (c canvas) imageNode(class string, spec partSpec, rawURL, alt string) gosx.Node {
	rawURL = strings.TrimSpace(rawURL)
	if c.editable {
		return gosx.El("figure", gosx.Attrs(gosx.Attr("class", "ed-figure ed-image-field "+class), gosx.Attr("contenteditable", "false"), gosx.Attr("data-image-field", spec.Key)),
			imagePreview(rawURL),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-image-field__controls")),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-upload")),
					gosx.El("input", gosx.Attrs(gosx.Attr("type", "file"), gosx.Attr("accept", "image/png,image/jpeg,image/gif,image/webp"), gosx.Attr("data-upload", "true"), gosx.Attr("aria-label", "Upload a picture"))),
					gosx.El("span", nil, gosx.Text("Upload"))),
				gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-library-btn"), gosx.Attr("data-library", "true")), gosx.Text("Choose")),
				gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("data-src", "true"), gosx.Attr("data-field", spec.Key), gosx.Attr("value", rawURL), gosx.Attr("placeholder", "or paste a link"), gosx.Attr("aria-label", spec.Label)))))
	}
	if rawURL == "" {
		return gosx.Fragment()
	}
	attrs, ok := c.h.imageAttrs(rawURL, alt, "(min-width: 900px) 50vw, 100vw")
	if !ok {
		return gosx.Fragment()
	}
	return gosx.El("figure", gosx.Attrs(gosx.Attr("class", class)), gosx.El("img", gosx.Attrs(attrs...)))
}

func (c canvas) flagNode(spec partSpec, value string) gosx.Node {
	if !c.editable {
		return gosx.Fragment()
	}
	attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("data-field", spec.Key), gosx.Attr("value", "yes")}
	if value == "yes" {
		attrs = append(attrs, gosx.Attr("checked", "checked"))
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-flag"), gosx.Attr("contenteditable", "false")), gosx.El("input", gosx.Attrs(attrs...)), gosx.Text(" "+spec.Label))
}

// itemWrap gives a repeated item its remove button on the canvas.
func (c canvas) itemWrap(tag, class string, body ...gosx.Node) gosx.Node {
	if !c.editable {
		return gosx.El(tag, gosx.Attrs(gosx.Attr("class", class)), gosx.Fragment(body...))
	}
	return gosx.El(tag, gosx.Attrs(gosx.Attr("class", class+" ed-item"), gosx.Attr("data-item", "true")),
		itemTools(),
		gosx.Fragment(body...))
}

// itemTools are the grab handle, the arrows, and the remove button that a
// repeated item shows when hovered or focused.
func itemTools() gosx.Node {
	button := func(attr, glyph, label string) gosx.Node {
		return gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-item__tool"), gosx.Attr(attr, "true"), gosx.Attr("title", label), gosx.Attr("aria-label", label)), gosx.Text(glyph))
	}
	return gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-item__tools"), gosx.Attr("contenteditable", "false")),
		button("data-item-grab", "⠿", "Drag to reorder"),
		button("data-item-up", "↑", "Move up"),
		button("data-item-down", "↓", "Move down"),
		button("data-item-remove", "✕", "Remove"))
}

// itemsWrap holds the repeated items, and the add button on the canvas.
func (c canvas) itemsWrap(spec compositeSpec, tag, class string, items []gosx.Node) gosx.Node {
	if !c.editable {
		return gosx.El(tag, gosx.Attrs(gosx.Attr("class", class)), gosx.Fragment(items...))
	}
	return gosx.Fragment(
		gosx.El(tag, gosx.Attrs(gosx.Attr("class", class), gosx.Attr("data-items", "true")), gosx.Fragment(items...)),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-item-add"), gosx.Attr("data-item-add", spec.Key), gosx.Attr("contenteditable", "false")), gosx.Text("+ Add a "+spec.ItemName)))
}

func (c canvas) variantPicker(spec compositeSpec, current string) gosx.Node {
	if !c.editable || len(spec.Variants) < 2 {
		return gosx.Fragment()
	}
	options := make([]gosx.Node, 0, len(spec.Variants))
	for _, variant := range spec.Variants {
		attrs := []any{gosx.Attr("value", variant.Key)}
		if variant.Key == current {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(variant.Label)))
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-variant"), gosx.Attr("contenteditable", "false")),
		gosx.El("span", nil, gosx.Text("Layout")),
		gosx.El("select", gosx.Attrs(gosx.Attr("data-variant", "true"), gosx.Attr("aria-label", "Layout")), gosx.Fragment(options...)))
}

// renderComposite draws one ready-made section for the page or the canvas.
func (h *Host) renderComposite(spec compositeSpec, instance blockstudio.BlockInstance, editable bool) gosx.Node {
	c := canvas{h: h, editable: editable}
	get := func(key string) string { return strings.TrimSpace(instance.Values[key].String) }
	f := func(key string) partSpec { field, _ := spec.field(key); return field }
	variant := spec.variant(get("variant"))
	shown := variant
	// A split or cover hero without a picture reads as a plain one.
	if spec.Key == "hero" && get("image") == "" && !editable && (variant == "split" || variant == "cover") {
		shown = "left"
	}
	class := "site-" + spec.Key + " site-" + spec.Key + "--" + shown
	rootAttrs := []any{gosx.Attr("class", class)}
	if editable {
		rootAttrs = append(rootAttrs, gosx.Attr("data-composite", spec.Key), gosx.Attr("data-variant-value", variant))
	}
	items := compositeItems(spec, instance)
	itemField := func(key string) partSpec {
		for _, field := range spec.Item {
			if field.Key == key {
				return field
			}
		}
		return partSpec{Key: key, Label: key, Kind: partText}
	}
	var body []gosx.Node

	switch spec.Key {
	case "hero":
		buttons := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-hero__actions")),
			c.linkNode("button button--primary", f("button"), get("button"), f("url"), get("url")),
			c.linkNode("button button--ghost", f("button2"), get("button2"), f("url2"), get("url2")))
		copyNode := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-hero__copy")),
			c.textNode("p", "site-hero__eyebrow", f("eyebrow"), get("eyebrow")),
			c.textNode("h2", "site-hero__headline", f("headline"), get("headline")),
			c.textNode("p", "site-hero__text", f("text"), get("text")),
			buttons)
		picture := c.imageNode("site-hero__picture", f("image"), get("image"), inlineToPlain(get("headline")))
		if variant == "cover" && !editable && get("image") != "" {
			rootAttrs = append(rootAttrs, gosx.Attr("style", "--hero-image: url('"+cssURL(get("image"))+"')"))
			picture = gosx.Fragment()
		}
		body = []gosx.Node{copyNode, picture}
	case "features":
		cards := make([]gosx.Node, 0, len(items))
		for index, item := range items {
			icon := c.textNode("span", "site-features__icon", itemField("icon"), item["icon"])
			if variant == "numbered" && !editable {
				icon = gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-features__icon")), gosx.Text(itoa(index+1)))
			}
			cards = append(cards, c.itemWrap("li", "site-features__card", icon,
				c.textNode("h3", "site-features__title", itemField("title"), item["title"]),
				c.textNode("p", "site-features__text", itemField("text"), item["text"])))
		}
		body = []gosx.Node{c.textNode("h2", "site-features__heading", f("heading"), get("heading")), c.textNode("p", "site-features__intro", f("intro"), get("intro")), c.itemsWrap(spec, "ul", "site-features__grid", cards)}
	case "testimonials":
		cards := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			cards = append(cards, c.itemWrap("li", "site-testimonials__card",
				c.textNode("blockquote", "site-testimonials__quote", itemField("quote"), item["quote"]),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-testimonials__who")),
					c.imageNode("site-testimonials__photo", itemField("image"), item["image"], inlineToPlain(item["name"])),
					gosx.El("div", nil,
						c.textNode("span", "site-testimonials__name", itemField("name"), item["name"]),
						c.textNode("span", "site-testimonials__role", itemField("role"), item["role"])))))
		}
		body = []gosx.Node{c.textNode("h2", "site-testimonials__heading", f("heading"), get("heading")), c.itemsWrap(spec, "ul", "site-testimonials__grid", cards)}
	case "pricing":
		cards := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			cardClass := "site-pricing__card"
			if item["highlight"] == "yes" {
				cardClass += " site-pricing__card--highlight"
			}
			lines := make([]gosx.Node, 0, 4)
			if editable {
				lines = append(lines, c.textNode("div", "site-pricing__features", itemField("features"), item["features"]))
			} else {
				for _, line := range strings.Split(item["features"], "\n") {
					if line = strings.TrimSpace(line); line != "" {
						lines = append(lines, gosx.El("li", nil, renderInline(line)))
					}
				}
				if len(lines) > 0 {
					lines = []gosx.Node{gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-pricing__features")), gosx.Fragment(lines...))}
				}
			}
			cards = append(cards, c.itemWrap("li", cardClass,
				c.textNode("h3", "site-pricing__name", itemField("name"), item["name"]),
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-pricing__price")),
					c.textNode("span", "site-pricing__amount", itemField("price"), item["price"]), gosx.Text(" "),
					c.textNode("span", "site-pricing__period", itemField("period"), item["period"])),
				c.textNode("p", "site-pricing__blurb", itemField("blurb"), item["blurb"]),
				gosx.Fragment(lines...),
				c.linkNode("button button--primary", itemField("button"), item["button"], itemField("url"), item["url"]),
				c.flagNode(itemField("highlight"), item["highlight"])))
		}
		body = []gosx.Node{c.textNode("h2", "site-pricing__heading", f("heading"), get("heading")), c.textNode("p", "site-pricing__intro", f("intro"), get("intro")), c.itemsWrap(spec, "ul", "site-pricing__grid", cards)}
	case "faq":
		entries := make([]gosx.Node, 0, len(items))
		for index, item := range items {
			if editable {
				entries = append(entries, c.itemWrap("div", "site-faq__item",
					c.textNode("h3", "site-faq__question", itemField("question"), item["question"]),
					c.textNode("p", "site-faq__answer", itemField("answer"), item["answer"])))
				continue
			}
			attrs := []any{gosx.Attr("class", "site-faq__item")}
			if variant == "list" || index == 0 {
				attrs = append(attrs, gosx.Attr("open", "open"))
			}
			if variant == "accordion" {
				attrs = append(attrs, gosx.Attr("name", "site-faq-"+instance.ID))
			}
			entries = append(entries, gosx.El("details", gosx.Attrs(attrs...),
				gosx.El("summary", gosx.Attrs(gosx.Attr("class", "site-faq__question")), renderInline(item["question"])),
				gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-faq__answer")), renderInline(item["answer"]))))
		}
		body = []gosx.Node{c.textNode("h2", "site-faq__heading", f("heading"), get("heading")), c.itemsWrap(spec, "div", "site-faq__list", entries)}
	case "cta":
		body = []gosx.Node{
			c.textNode("h2", "site-cta__headline", f("headline"), get("headline")),
			c.textNode("p", "site-cta__text", f("text"), get("text")),
			c.linkNode("button button--primary", f("button"), get("button"), f("url"), get("url"))}
	case "stats":
		cells := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			cells = append(cells, c.itemWrap("li", "site-stats__item",
				c.textNode("strong", "site-stats__value", itemField("value"), item["value"]),
				c.textNode("span", "site-stats__label", itemField("label"), item["label"])))
		}
		body = []gosx.Node{c.textNode("h2", "site-stats__heading", f("heading"), get("heading")), c.itemsWrap(spec, "ul", "site-stats__row", cells)}
	case "team":
		people := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			people = append(people, c.itemWrap("li", "site-team__person",
				c.imageNode("site-team__photo", itemField("image"), item["image"], inlineToPlain(item["name"])),
				c.textNode("h3", "site-team__name", itemField("name"), item["name"]),
				c.textNode("p", "site-team__role", itemField("role"), item["role"]),
				c.textNode("p", "site-team__bio", itemField("bio"), item["bio"])))
		}
		body = []gosx.Node{c.textNode("h2", "site-team__heading", f("heading"), get("heading")), c.textNode("p", "site-team__intro", f("intro"), get("intro")), c.itemsWrap(spec, "ul", "site-team__grid", people)}
	case "hours":
		rows := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			rows = append(rows, c.itemWrap("div", "site-hours__row",
				c.textNode("span", "site-hours__day", itemField("day"), item["day"]),
				c.textNode("span", "site-hours__time", itemField("time"), item["time"])))
		}
		body = []gosx.Node{c.textNode("h2", "site-hours__heading", f("heading"), get("heading")), c.itemsWrap(spec, "div", "site-hours__table", rows), c.textNode("p", "site-hours__note", f("note"), get("note"))}
	case "imagetext":
		body = []gosx.Node{
			c.imageNode("site-imagetext__picture", f("image"), get("image"), inlineToPlain(get("heading"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-imagetext__copy")),
				c.textNode("h2", "site-imagetext__heading", f("heading"), get("heading")),
				c.textNode("p", "site-imagetext__text", f("text"), get("text")),
				c.linkNode("button button--primary", f("button"), get("button"), f("url"), get("url")))}
	case "map":
		address := get("address")
		frame := gosx.Fragment()
		if address != "" {
			frame = gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-map__frame")),
				gosx.El("iframe", gosx.Attrs(gosx.Attr("src", "https://maps.google.com/maps?q="+url.QueryEscape(address)+"&z=15&output=embed"), gosx.Attr("title", "Map of "+inlineToPlain(address)), gosx.Attr("loading", "lazy"), gosx.Attr("referrerpolicy", "no-referrer-when-downgrade"))))
		}
		body = []gosx.Node{c.textNode("p", "site-map__address", f("address"), address), frame, c.textNode("p", "site-map__note", f("note"), get("note"))}
	case "spacer":
		if editable {
			body = []gosx.Node{gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-spacer__label"), gosx.Attr("contenteditable", "false")), gosx.Text("Space"))}
		}
		rootAttrs = append(rootAttrs, gosx.Attr("aria-hidden", "true"))
	case "posts":
		count := map[string]int{"one": 1, "three": 3, "six": 6}[variant]
		posts := h.livePosts()
		if len(posts) > count {
			posts = posts[:count]
		}
		var list gosx.Node = gosx.Fragment()
		switch {
		case len(posts) > 0:
			list = renderPostList(posts)
		case editable:
			list = gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-live-hint"), gosx.Attr("contenteditable", "false")), gosx.Text("Your newest posts appear here once you publish one under Blog."))
		default:
			return gosx.Fragment()
		}
		if editable {
			list = gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-live"), gosx.Attr("contenteditable", "false")), list)
		}
		body = []gosx.Node{c.textNode("h2", "site-posts__heading", f("heading"), get("heading")), list}
	case "products":
		count := map[string]int{"three": 3, "six": 6, "all": 1 << 20}[variant]
		products := h.activeProducts()
		if len(products) > count {
			products = products[:count]
		}
		var grid gosx.Node = gosx.Fragment()
		switch {
		case len(products) > 0:
			grid = h.renderProductGrid(products)
		case editable:
			grid = gosx.El("p", gosx.Attrs(gosx.Attr("class", "ed-live-hint"), gosx.Attr("contenteditable", "false")), gosx.Text("Products on sale appear here. Add one under Shop."))
		default:
			return gosx.Fragment()
		}
		if editable {
			grid = gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-live"), gosx.Attr("contenteditable", "false")), grid)
		}
		body = []gosx.Node{c.textNode("h2", "site-products__heading", f("heading"), get("heading")), grid}
	}
	if editable {
		body = append([]gosx.Node{c.variantPicker(spec, variant)}, body...)
	}
	return gosx.El("div", gosx.Attrs(rootAttrs...), gosx.Fragment(body...))
}

func cssURL(raw string) string {
	return strings.NewReplacer("'", "%27", "\\", "%5C", "\n", "", ")", "%29", "(", "%28").Replace(strings.TrimSpace(raw))
}

// leadsWithHero reports whether a page opens with a hero, whose headline
// then does the title's job on the page.
func leadsWithHero(doc blockstudio.Document) bool {
	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue
		}
		return instance.Key == "hero"
	}
	return false
}

// ---------- sections ----------

// Section options beyond the background.
var (
	sectionAligns = map[string]bool{"left": true, "center": true}
	sectionWidths = map[string]bool{"normal": true, "narrow": true, "wide": true, "full": true}
	sectionSpaces = map[string]bool{"normal": true, "compact": true, "roomy": true}
)

func normalizeChoice(value, fallback string, allowed map[string]bool) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if allowed[value] {
		return value
	}
	return fallback
}

// sectionOptions is everything a section break carries.
type sectionOptions struct {
	Style, Align, Width, Space, Image string
	// Anchor is the name a link can jump to: /#pricing.
	Anchor string
}

func sectionOptionsOf(instance blockstudio.BlockInstance) sectionOptions {
	get := func(key string) string { return instance.Values[key].String }
	return sectionOptions{
		Style:  normalizeSectionStyle(get("style")),
		Align:  normalizeChoice(get("align"), "left", sectionAligns),
		Width:  normalizeChoice(get("width"), "normal", sectionWidths),
		Space:  normalizeChoice(get("space"), "normal", sectionSpaces),
		Image:  strings.TrimSpace(get("image")),
		Anchor: normalizeSlug(get("anchor")),
	}
}

func (o sectionOptions) values() blockstudio.Values {
	return values("style", o.Style, "align", o.Align, "width", o.Width, "space", o.Space, "image", o.Image, "anchor", o.Anchor)
}

// classes are what the public page hangs its CSS on.
func (o sectionOptions) classes() string {
	out := "site-section site-section--" + o.Style
	if o.Align != "left" {
		out += " site-section--align-" + o.Align
	}
	if o.Width != "normal" {
		out += " site-section--width-" + o.Width
	}
	if o.Space != "normal" {
		out += " site-section--space-" + o.Space
	}
	return out
}

// renderSectionBar is a section break on the canvas: a labelled rule with
// the choices for everything beneath it.
func renderSectionBar(o sectionOptions) gosx.Node {
	choice := func(attr, label, current string, options [][2]string) gosx.Node {
		nodes := make([]gosx.Node, 0, len(options))
		for _, option := range options {
			attrs := []any{gosx.Attr("value", option[0])}
			if option[0] == current {
				attrs = append(attrs, gosx.Attr("selected", "selected"))
			}
			nodes = append(nodes, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(option[1])))
		}
		return gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-section-bar__style")),
			gosx.El("span", nil, gosx.Text(label)),
			gosx.El("select", gosx.Attrs(gosx.Attr(attr, "true"), gosx.Attr("aria-label", label)), gosx.Fragment(nodes...)))
	}
	picture := gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-section-bar__image"), gosx.Attr("data-section-image", "true")),
		gosx.El("figure", gosx.Attrs(gosx.Attr("class", "ed-figure ed-image-field ed-figure--small"), gosx.Attr("contenteditable", "false")),
			imagePreview(o.Image),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-image-field__controls")),
				gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-upload")),
					gosx.El("input", gosx.Attrs(gosx.Attr("type", "file"), gosx.Attr("accept", "image/png,image/jpeg,image/gif,image/webp"), gosx.Attr("data-upload", "true"), gosx.Attr("aria-label", "Upload a background picture"))),
					gosx.El("span", nil, gosx.Text("Upload"))),
				gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("class", "ed-library-btn"), gosx.Attr("data-library", "true")), gosx.Text("Choose")),
				gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("data-src", "true"), gosx.Attr("data-section-src", "true"), gosx.Attr("value", o.Image), gosx.Attr("placeholder", "or paste a link"), gosx.Attr("aria-label", "Background picture link"))))))
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-section-bar"), gosx.Attr("data-section", o.Style), gosx.Attr("contenteditable", "false")),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "ed-section-bar__label")), gosx.Text("New section")),
		choice("data-section-style", "Background", o.Style, [][2]string{{"plain", "Plain"}, {"tinted", "Tinted"}, {"accent", "Accent colour"}, {"dark", "Dark"}, {"image", "Picture"}}),
		choice("data-section-align", "Text", o.Align, [][2]string{{"left", "Left"}, {"center", "Centred"}}),
		choice("data-section-width", "Width", o.Width, [][2]string{{"narrow", "Narrow"}, {"normal", "Normal"}, {"wide", "Wide"}, {"full", "Edge to edge"}}),
		choice("data-section-space", "Space", o.Space, [][2]string{{"compact", "Compact"}, {"normal", "Normal"}, {"roomy", "Roomy"}}),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "ed-section-bar__style")),
			gosx.El("span", nil, gosx.Text("Jump-to name")),
			gosx.El("input", gosx.Attrs(gosx.Attr("class", "ed-inline-input"), gosx.Attr("type", "text"), gosx.Attr("data-section-anchor", "true"), gosx.Attr("value", o.Anchor), gosx.Attr("placeholder", "pricing"), gosx.Attr("aria-label", "Jump-to name"), gosx.Attr("title", "Buttons can then link to #pricing")))),
		picture,
	)
}

// ---------- the editor's block service ----------

// mountBlocks serves freshly rendered blocks and items to the canvas, so
// the browser never has to know how a section looks.
func (h *Host) mountBlocks(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/api/blocks/{kind}", h.handleBlockFresh)
	mux.HandleFunc("GET /admin/api/blocks/{kind}/item", h.handleBlockItem)
}

func (h *Host) handleBlockFresh(w http.ResponseWriter, r *http.Request) {
	kind := r.PathValue("kind")
	var inner gosx.Node
	switch {
	case kind == "section":
		inner = renderSectionBar(sectionOptions{Style: "plain", Align: "left", Width: "normal", Space: "normal"})
	case kind == "image":
		inner = h.renderBlockInner("image", blockstudio.BlockInstance{Key: "image", Values: blockstudio.Values{}})
	case kind == "button":
		inner = h.renderBlockInner("button", blockstudio.BlockInstance{Key: "button", Values: values("label", "Get in touch", "href", "/contact", "look", "primary")})
	case kind == "columns":
		inner = h.renderBlockInner("columns", blockstudio.BlockInstance{Key: "columns", Values: blockstudio.Values{}})
	case kind == "gallery":
		inner = h.renderBlockInner("gallery", blockstudio.BlockInstance{Key: "gallery", Values: blockstudio.Values{}})
	default:
		spec, ok := compositeByKey(kind)
		if !ok {
			http.NotFound(w, r)
			return
		}
		inner = h.renderComposite(spec, freshComposite(spec, 0), true)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(gosx.RenderHTML(inner)))
}

func (h *Host) handleBlockItem(w http.ResponseWriter, r *http.Request) {
	spec, ok := compositeByKey(r.PathValue("kind"))
	if !ok || spec.Item == nil {
		http.NotFound(w, r)
		return
	}
	// One empty item, rendered inside a throwaway block so the markup
	// matches its siblings exactly.
	blank := map[string]string{}
	for _, field := range spec.Item {
		blank[field.Key] = ""
	}
	instance := freshComposite(spec, 0)
	instance.Values["items"] = blockstudio.Value{List: []blockstudio.Value{{Object: objectOf(blank)}}}
	html := gosx.RenderHTML(h.renderComposite(spec, instance, true))
	start := strings.Index(html, `data-item="true"`)
	if start < 0 {
		http.NotFound(w, r)
		return
	}
	// Walk back to the opening tag, then forward to the matching close.
	open := strings.LastIndex(html[:start], "<")
	item := html[open:]
	item = item[:closingIndex(item)]
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(item))
}

func objectOf(item map[string]string) map[string]blockstudio.Value {
	object := map[string]blockstudio.Value{}
	for key, value := range item {
		object[key] = text(value)
	}
	return object
}

// closingIndex finds the end of the first element in html by counting
// tags; the markup is ours and always balanced.
func closingIndex(html string) int {
	depth := 0
	i := 0
	for i < len(html) {
		if html[i] != '<' {
			i++
			continue
		}
		end := strings.IndexByte(html[i:], '>')
		if end < 0 {
			return len(html)
		}
		tag := html[i : i+end+1]
		switch {
		case strings.HasPrefix(tag, "</"):
			depth--
			if depth == 0 {
				return i + end + 1
			}
		case strings.HasSuffix(tag, "/>"), strings.HasPrefix(tag, "<input"), strings.HasPrefix(tag, "<img"), strings.HasPrefix(tag, "<br"):
			// void
		default:
			depth++
		}
		i += end + 1
	}
	return len(html)
}
