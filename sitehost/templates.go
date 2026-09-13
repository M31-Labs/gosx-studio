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
	Kind      string
	Template  string // Template key; empty means the kind's default
	Email     string
	Phone     string
	Location  string
	BaseURL   string
}

func (a SetupAnswers) trimmed() SetupAnswers {
	a.SiteTitle = strings.TrimSpace(a.SiteTitle)
	a.Tagline = strings.TrimSpace(a.Tagline)
	a.Kind = strings.TrimSpace(a.Kind)
	a.Template = strings.ToLower(strings.TrimSpace(a.Template))
	a.Email = strings.TrimSpace(a.Email)
	a.Phone = strings.TrimSpace(a.Phone)
	a.Location = strings.TrimSpace(a.Location)
	a.BaseURL = strings.TrimRight(strings.TrimSpace(a.BaseURL), "/")
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

	hero := func(order int, button, target, second, secondTarget string) blockstudio.BlockInstance {
		fields := map[string]string{"eyebrow": "Welcome", "headline": tagline, "text": "A sentence or two on what you do and who it's for. The rest of the page can do the explaining.", "button": button, "url": target}
		if second != "" {
			fields["button2"] = second
			fields["url2"] = secondTarget
		}
		return comp(order, "hero", "center", fields, nil)
	}
	cta := func(order int, headline, text, button, target string) blockstudio.BlockInstance {
		return comp(order, "cta", "band", map[string]string{"headline": headline, "text": text, "button": button, "url": target}, nil)
	}

	var pages []StarterPage
	switch SiteKindByKey(answers.Kind).Key {
	case "shop":
		home.Body = document(
			hero(0, "See what's in stock", "/shop", "", ""),
			comp(1, "features", "plain", map[string]string{"heading": "Why people buy from us"}, []map[string]string{
				{"icon": "✦", "title": "Made with care", "text": "Materials, process, the person behind it — whatever a customer would want to know before they spend money."},
				{"icon": "✦", "title": "Made to last", "text": "A sentence on quality, and what happens if something isn't right."},
				{"icon": "✦", "title": "Made nearby", "text": "Where it's made, how it ships, and how long that takes."}}),
			comp(2, "products", "three", map[string]string{"heading": "From the shop"}, nil),
			cta(3, "Something you can't see here?", "Get in touch and we'll sort it out.", "Get in touch", "/contact"),
		)
		pages = []StarterPage{
			home,
			// No "Shop" page: /shop belongs to the products in Shop and joins
			// the menu the moment the first one goes on sale.
			{Slug: "about", Title: "About", Description: "The story behind " + name + ".", Publish: true, Body: document(
				heading(0, 2, "About "+name),
				para(1, "Who you are and how you got here. People buy from people, so write this the way you'd tell a customer standing in front of you."),
				quote(2, "Swap this for something a real customer said about your work."),
			)},
		}
	case "services":
		home.Body = document(
			hero(0, "See what I do", "/services", "Get in touch", "/contact"),
			comp(1, "features", "cards", map[string]string{"heading": "How I can help"}, []map[string]string{
				{"icon": "1", "title": "The first service", "text": "Say what the client gets, not how you do it."},
				{"icon": "2", "title": "The second service", "text": "One or two sentences is plenty."},
				{"icon": "3", "title": "The third service", "text": "If you can name a price or a range, do."}}),
			comp(2, "testimonials", "grid", map[string]string{"heading": "What clients say"}, nil),
			cta(3, "Tell me about the job", "I reply within a working day.", "Get in touch", "/contact"),
		)
		pages = []StarterPage{
			home,
			{Slug: "services", Title: "Services", Description: "What " + name + " offers.", Publish: true, Body: document(
				heading(0, 2, "What I do"),
				para(1, "Describe each service in a sentence or two. Say what the client gets, not how you do it."),
				heading(2, 3, "How it works"),
				para(3, "Walk through what happens after someone gets in touch. Removing the mystery is the most persuasive thing on most service websites."),
				para(4, "If you can name a price or a range, do. It saves everyone time."),
			)},
			{Slug: "about", Title: "About", Description: "About " + name + ".", Publish: true, Body: document(
				heading(0, 2, "About"),
				para(1, "Your background, in plain words. Enough for someone to decide they'd be comfortable working with you."),
			)},
		}
	case "food":
		home.Body = document(
			hero(0, "See the menu", "/menu", "Find us", "/visit"),
			comp(1, "hours", "inline", map[string]string{"heading": "Opening hours", "note": ""}, nil),
			cta(2, "Come and find us", firstNonEmpty(answers.Location, "Add your address here, plus a line about parking, the nearest stop, or the door that's easy to miss."), "How to find us", "/visit"),
		)
		pages = []StarterPage{
			home,
			{Slug: "menu", Title: "Menu", Description: "What's on at " + name + ".", Publish: true, Body: document(
				heading(0, 2, "Menu"),
				heading(1, 3, "Mornings"),
				para(2, "List what you serve and what it costs. Keep it short — people scan menus, they don't read them."),
				heading(3, 3, "Afternoons"),
				para(4, "Add a line about anything you're known for, and note what you can do for allergies."),
			)},
			{Slug: "visit", Title: "Visit", Description: "Opening hours and directions for " + name + ".", Publish: true, Body: document(
				comp(0, "hours", "table", map[string]string{"heading": "When we're open"}, nil),
				comp(1, "map", "wide", map[string]string{"address": firstNonEmpty(answers.Location, "Your address"), "note": "How to find the door, where to park, the nearest stop."}, nil),
			)},
		}
	case "portfolio":
		home.Body = document(
			hero(0, "See the work", "/work", "", ""),
			comp(1, "features", "plain", map[string]string{"heading": "What I do"}, []map[string]string{
				{"icon": "✦", "title": "The first thing", "text": "A line on it."},
				{"icon": "✦", "title": "The second thing", "text": "A line on it."},
				{"icon": "✦", "title": "The third thing", "text": "A line on it."}}),
			cta(2, "Working on something?", "Tell me about it.", "Get in touch", "/contact"),
		)
		pages = []StarterPage{
			home,
			{Slug: "work", Title: "Work", Description: "Selected work by " + name + ".", Publish: true, Body: document(
				heading(0, 2, "Selected work"),
				para(1, "Show six to ten pieces, not everything. For each one, a line on what it was and what you did."),
				para(2, "Put your strongest piece first. Most people never scroll to the bottom."),
			)},
			{Slug: "about", Title: "About", Description: "About " + name + ".", Publish: true, Body: document(
				heading(0, 2, "About"),
				para(1, "What you do, who you've done it for, and what you'd like to do next."),
			)},
		}
	case "community":
		home.Body = document(
			hero(0, "What's coming up", "/whats-on", "", ""),
			comp(1, "features", "numbered", map[string]string{"heading": "New here?"}, []map[string]string{
				{"icon": "", "title": "Come along", "text": "When to arrive and where to go."},
				{"icon": "", "title": "Say hello", "text": "Who to look for when you get there."},
				{"icon": "", "title": "Join in", "text": "What happens, and whether you need to bring anything."}}),
			cta(2, "Questions before you come?", "We're happy to help.", "Get in touch", "/contact"),
		)
		pages = []StarterPage{
			home,
			{Slug: "whats-on", Title: "What's on", Description: "Upcoming events at " + name + ".", Publish: true, Body: document(
				heading(0, 2, "What's coming up"),
				para(1, "List the next few things with dates and times. Keep past events off this page."),
			)},
			{Slug: "about", Title: "About", Description: "About " + name + ".", Publish: true, Body: document(
				heading(0, 2, "Who we are"),
				para(1, "What the group is for, who runs it, and how long you've been going."),
				quote(2, "A line from a member about what this place means to them."),
			)},
		}
	default:
		home.Body = document(
			hero(0, "Get in touch", "/contact", "", ""),
			comp(1, "features", "plain", map[string]string{"heading": "What you'll find here"}, []map[string]string{
				{"icon": "✦", "title": "One thing", "text": "A line on it."},
				{"icon": "✦", "title": "Another", "text": "A line on it."},
				{"icon": "✦", "title": "And a third", "text": "A line on it."}}),
			cta(2, "Ready when you are", "One line on what happens when they get in touch.", "Get in touch", "/contact"),
		)
		pages = []StarterPage{home}
	}

	pages[0].Body = bandHome(pages[0].Body, templateFor(answers).Band)
	return append(pages, contactPage(name, answers))
}

func contactPage(name string, answers SetupAnswers) StarterPage {
	body := document(
		heading(0, 2, "Get in touch"),
		para(1, contactLine(answers)),
	)
	if answers.Location != "" {
		body.Blocks = append(body.Blocks, heading(2, 3, "Where to find us"), para(3, answers.Location))
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
		Publish:     true,
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
