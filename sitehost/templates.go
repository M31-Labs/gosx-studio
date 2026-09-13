package sitehost

import (
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
)

// templates.go turns one answer — "what kind of business is this?" — into a
// real starting website.
//
// A generic starter site is only slightly better than an empty one: the
// operator still has to imagine every page from scratch. A bakery that lands on
// a menu page, and a consultant who lands on a services page, both start from
// something they only need to correct rather than invent.

// SiteKind is a starting point, not a category the product enforces later.
type SiteKind struct {
	Key      string
	Label    string
	Blurb    string
	Examples string
}

// SiteKinds are the starting points the setup wizard offers.
func SiteKinds() []SiteKind {
	return []SiteKind{
		{
			Key:      "shop",
			Label:    "I sell things",
			Blurb:    "A shop window for what you make or resell.",
			Examples: "Makers, boutiques, online stores",
		},
		{
			Key:      "services",
			Label:    "I offer a service",
			Blurb:    "Explain what you do and let people book or enquire.",
			Examples: "Consultants, trades, studios, coaches",
		},
		{
			Key:      "food",
			Label:    "I run a place people visit",
			Blurb:    "Menu, hours, and directions, front and centre.",
			Examples: "Cafés, restaurants, bars, shops",
		},
		{
			Key:      "portfolio",
			Label:    "I show my work",
			Blurb:    "Let the work do the talking, with a way to get in touch.",
			Examples: "Designers, photographers, writers, artists",
		},
		{
			Key:      "community",
			Label:    "I run a group or organisation",
			Blurb:    "What you're for, what's coming up, and how to join in.",
			Examples: "Schools, clubs, nonprofits, congregations",
		},
		{
			Key:      "simple",
			Label:    "Something else",
			Blurb:    "A clean home page and a contact page. Add the rest yourself.",
			Examples: "Anything that doesn't fit a box",
		},
	}
}

// SiteKindByKey returns the chosen starting point, defaulting to the simple one.
func SiteKindByKey(key string) SiteKind {
	key = strings.TrimSpace(strings.ToLower(key))
	kinds := SiteKinds()
	for _, kind := range kinds {
		if kind.Key == key {
			return kind
		}
	}
	return kinds[len(kinds)-1]
}

// SetupAnswers is everything the wizard collects.
type SetupAnswers struct {
	SiteTitle string
	Tagline   string
	// Description is two or three sentences on the business, for the About
	// page and the hero.
	Description string
	Kind        string
	Template    string // Template key; empty means the kind's default
	Email       string
	Phone       string
	Location    string
	BaseURL     string
	// Offers are the three things the business does or sells.
	Offers [3]Offer
	// Hours are the opening hours; HoursSet says the owner saw that step,
	// NoHours that they have none.
	Hours    [7]HoursDay
	HoursSet bool
	NoHours  bool
	// Social maps a network key (instagram, facebook, …) to its link.
	Social map[string]string
	// Pages are the optional pages to build; PagesSet says the owner chose
	// (otherwise the kind's defaults apply). Publish makes them live at once.
	Pages    map[string]bool
	PagesSet bool
	Publish  bool
}

func (a SetupAnswers) trimmed() SetupAnswers {
	a.SiteTitle = strings.TrimSpace(a.SiteTitle)
	a.Tagline = strings.TrimSpace(a.Tagline)
	a.Description = strings.TrimSpace(a.Description)
	a.Kind = strings.TrimSpace(a.Kind)
	a.Template = strings.ToLower(strings.TrimSpace(a.Template))
	a.Email = strings.TrimSpace(a.Email)
	a.Phone = strings.TrimSpace(a.Phone)
	a.Location = strings.TrimSpace(a.Location)
	a.BaseURL = strings.TrimRight(strings.TrimSpace(a.BaseURL), "/")
	for i := range a.Offers {
		a.Offers[i] = Offer{Name: strings.TrimSpace(a.Offers[i].Name), Text: strings.TrimSpace(a.Offers[i].Text), Price: strings.TrimSpace(a.Offers[i].Price)}
	}
	if !a.PagesSet {
		a.Publish = true
	}
	return a
}

// StarterSiteFor builds the pages a new site begins with. Every line of copy is
// written to be replaced: it says what belongs there, in the owner's voice,
// rather than sitting there as "Lorem ipsum" or an empty box.
func StarterSiteFor(answers SetupAnswers) []StarterPage {
	answers = answers.trimmed()
	name := firstNonEmpty(answers.SiteTitle, "My site")
	tagline := firstNonEmpty(answers.Tagline, "A short line about what you do and who it's for.")

	home := StarterPage{
		Slug:        "home",
		Title:       name,
		Description: tagline,
		Publish:     true,
	}

	pages := answers.pages()
	kind := SiteKindByKey(answers.Kind).Key
	mainSlug, _ := mainPageFor(kind)
	mainTarget := "/contact"
	if mainSlug != "" && pages["main"] {
		mainTarget = "/" + mainSlug
	}
	hero := func(order int, button, target, second, secondTarget string) blockstudio.BlockInstance {
		if target == "/"+mainSlug && mainSlug != "" && !pages["main"] {
			target, second, secondTarget = "/contact", "", ""
			button = "Get in touch"
		}
		fields := map[string]string{"eyebrow": "Welcome", "headline": tagline, "text": firstNonEmpty(answers.Description, "A sentence or two on what you do and who it's for. The rest of the page can do the explaining."), "button": button, "url": target}
		if second != "" {
			fields["button2"] = second
			fields["url2"] = secondTarget
		}
		return comp(order, "hero", "center", fields, nil)
	}
	cta := func(order int, headline, text, button, target string) blockstudio.BlockInstance {
		return comp(order, "cta", "band", map[string]string{"headline": headline, "text": text, "button": button, "url": target}, nil)
	}

	var starters, extras []StarterPage
	switch kind {
	case "shop":
		home.Body = document(
			hero(0, "See what's in stock", "/shop", "", ""),
			comp(1, "features", "plain", map[string]string{"heading": "Why people buy from us"}, answers.featureItems([]map[string]string{
				{"icon": "✦", "title": "Made with care", "text": "Materials, process, the person behind it — whatever a customer would want to know before they spend money."},
				{"icon": "✦", "title": "Made to last", "text": "A sentence on quality, and what happens if something isn't right."},
				{"icon": "✦", "title": "Made nearby", "text": "Where it's made, how it ships, and how long that takes."}})),
			comp(2, "products", "three", map[string]string{"heading": "From the shop"}, nil),
			cta(3, "Something you can't see here?", "Get in touch and we'll sort it out.", "Get in touch", "/contact"),
		)
		// No "Shop" page: /shop belongs to the products in Shop and joins
		// the menu the moment the first one goes on sale.
		starters = []StarterPage{home}
	case "services":
		home.Body = document(
			hero(0, "See what I do", "/services", "Get in touch", "/contact"),
			comp(1, "features", "cards", map[string]string{"heading": "How I can help"}, answers.featureItems([]map[string]string{
				{"icon": "1", "title": "The first service", "text": "Say what the client gets, not how you do it."},
				{"icon": "2", "title": "The second service", "text": "One or two sentences is plenty."},
				{"icon": "3", "title": "The third service", "text": "If you can name a price or a range, do."}})),
			comp(2, "testimonials", "grid", map[string]string{"heading": "What clients say"}, nil),
			cta(3, "Tell me about the job", "I reply within a working day.", "Get in touch", "/contact"),
		)
		starters = []StarterPage{home}
	case "food":
		foodBlocks := []blockstudio.BlockInstance{hero(0, "See the menu", "/menu", "Find us", "/visit")}
		if !answers.NoHours {
			foodBlocks = append(foodBlocks, comp(1, "hours", "inline", map[string]string{"heading": "Opening hours", "note": ""}, answers.hoursItems()))
		}
		if answers.hasOffers() {
			foodBlocks = append(foodBlocks, comp(len(foodBlocks), "features", "plain", map[string]string{"heading": "What people come for"}, answers.featureItems(nil)))
		}
		foodBlocks = append(foodBlocks, cta(len(foodBlocks), "Come and find us", firstNonEmpty(answers.Location, "Add your address here, plus a line about parking, the nearest stop, or the door that's easy to miss."), "How to find us", "/visit"))
		home.Body = document(foodBlocks...)
		visitBlocks := []blockstudio.BlockInstance{}
		if !answers.NoHours {
			visitBlocks = append(visitBlocks, comp(0, "hours", "table", map[string]string{"heading": "When we're open"}, answers.hoursItems()))
		}
		visitBlocks = append(visitBlocks, comp(len(visitBlocks), "map", "wide", map[string]string{"address": firstNonEmpty(answers.Location, "Your address"), "note": "How to find the door, where to park, the nearest stop."}, nil))
		starters = []StarterPage{home}
		extras = append(extras, StarterPage{Slug: "visit", Title: "Visit", Description: "Opening hours and directions for " + name + ".", Publish: answers.Publish, Body: document(visitBlocks...)})
	case "portfolio":
		home.Body = document(
			hero(0, "See the work", "/work", "", ""),
			comp(1, "features", "plain", map[string]string{"heading": "What I do"}, answers.featureItems([]map[string]string{
				{"icon": "✦", "title": "The first thing", "text": "A line on it."},
				{"icon": "✦", "title": "The second thing", "text": "A line on it."},
				{"icon": "✦", "title": "The third thing", "text": "A line on it."}})),
			cta(2, "Working on something?", "Tell me about it.", "Get in touch", "/contact"),
		)
		starters = []StarterPage{home}
	case "community":
		home.Body = document(
			hero(0, "What's coming up", "/whats-on", "", ""),
			comp(1, "features", "numbered", map[string]string{"heading": "New here?"}, answers.featureItems([]map[string]string{
				{"icon": "", "title": "Come along", "text": "When to arrive and where to go."},
				{"icon": "", "title": "Say hello", "text": "Who to look for when you get there."},
				{"icon": "", "title": "Join in", "text": "What happens, and whether you need to bring anything."}})),
			cta(2, "Questions before you come?", "We're happy to help.", "Get in touch", "/contact"),
		)
		starters = []StarterPage{home}
	default:
		home.Body = document(
			hero(0, "Get in touch", "/contact", "", ""),
			comp(1, "features", "plain", map[string]string{"heading": "What you'll find here"}, answers.featureItems([]map[string]string{
				{"icon": "✦", "title": "One thing", "text": "A line on it."},
				{"icon": "✦", "title": "Another", "text": "A line on it."},
				{"icon": "✦", "title": "And a third", "text": "A line on it."}})),
			cta(2, "Ready when you are", "One line on what happens when they get in touch.", "Get in touch", "/contact"),
		)
		starters = []StarterPage{home}
	}

	starters[0].Body = bandHome(starters[0].Body, templateFor(answers).Band)
	starters[0].Publish = answers.Publish
	starters = append(starters, turnkeyPages(name, answers, kind, extras)...)
	starters = append(starters, contactPage(name, answers))
	if pages["privacy"] {
		starters = append(starters, privacyPage(name, answers, kind == "shop"))
	}
	_ = mainTarget
	return starters
}

func contactPage(name string, answers SetupAnswers) StarterPage {
	body := document(
		heading(0, 2, "Get in touch"),
		para(1, contactLine(answers)),
	)
	if answers.Location != "" {
		body.Blocks = append(body.Blocks, heading(len(body.Blocks), 3, "Where to find us"), para(len(body.Blocks)+1, answers.Location))
		if SiteKindByKey(answers.Kind).Key != "food" {
			body.Blocks = append(body.Blocks, comp(len(body.Blocks), "map", "compact", map[string]string{"address": answers.Location}, nil))
		}
	}
	if items := answers.hoursItems(); len(items) > 0 && SiteKindByKey(answers.Kind).Key != "food" {
		body.Blocks = append(body.Blocks, comp(len(body.Blocks), "hours", "table", map[string]string{"heading": "When we're open", "note": ""}, items))
	}
	// A working form, on the contact page, from the first minute. Messages
	// land in the admin inbox; no mail server is needed.
	body.Blocks = append(body.Blocks,
		heading(len(body.Blocks), 3, "Send us a message"),
		block(len(body.Blocks)+1, content.BlockFlow, values("flowKey", contactFlowKey)),
	)
	return StarterPage{
		Slug:        "contact",
		Title:       "Contact",
		Description: "How to reach " + name + ".",
		Publish:     answers.Publish,
		Body:        body,
	}
}

func contactLine(answers SetupAnswers) string {
	parts := []string{}
	if answers.Email != "" {
		parts = append(parts, "Email "+answers.Email)
	}
	if answers.Phone != "" {
		parts = append(parts, "call "+answers.Phone)
	}
	switch len(parts) {
	case 0:
		return "Add the best way to reach you — an email address, a phone number, or both."
	case 1:
		return parts[0] + "."
	default:
		return parts[0] + " or " + parts[1] + "."
	}
}

// ---- small block builders, so the templates above read like content ----

func heading(order, level int, value string) blockstudio.BlockInstance {
	return block(order, content.BlockHeading, values("text", value, "level", itoa(level)))
}

func para(order int, value string) blockstudio.BlockInstance {
	return block(order, content.BlockParagraph, values("text", value))
}

func quote(order int, value string) blockstudio.BlockInstance {
	return block(order, content.BlockQuote, values("text", value))
}

func button(order int, label, target string) blockstudio.BlockInstance {
	return block(order, content.BlockButton, values("label", label, "href", target))
}
