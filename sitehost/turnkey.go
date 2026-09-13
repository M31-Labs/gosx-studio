package sitehost

import (
	"net/http"
	"net/url"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// turnkey.go is the part of the setup wizard that makes the first visit
// enough: what you offer, where and when to find you, and which pages to
// build. Every answer lands on a page or in a setting, so the owner opens
// the editor to a site that already says the right things, not to a blank.

// Offer is one thing the business sells or does.
type Offer struct {
	Name, Text, Price string
}

// HoursDay is one row of opening hours.
type HoursDay struct {
	Day, Open, Close string
	Closed           bool
}

var weekDays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

func dayKey(day string) string { return strings.ToLower(day[:3]) }

// defaultHours is a plausible week the owner corrects rather than types.
func defaultHours() [7]HoursDay {
	var week [7]HoursDay
	for i, day := range weekDays {
		switch i {
		case 5:
			week[i] = HoursDay{Day: day, Open: "10:00", Close: "16:00"}
		case 6:
			week[i] = HoursDay{Day: day, Closed: true}
		default:
			week[i] = HoursDay{Day: day, Open: "9:00", Close: "17:00"}
		}
	}
	return week
}

// wizardPages are the optional pages, in the order the wizard lists them.
var wizardPages = []string{"main", "about", "pricing", "gallery", "faq", "privacy"}

// mainPageFor is the kind's own page: the menu, the services, the work.
func mainPageFor(kind string) (slug, title string) {
	switch SiteKindByKey(kind).Key {
	case "food":
		return "menu", "Menu"
	case "services":
		return "services", "Services"
	case "portfolio":
		return "work", "Work"
	case "community":
		return "whats-on", "What's on"
	}
	return "", ""
}

// offerPrompt is how the wizard asks for the three offers, per kind.
func offerPrompt(kind string) (heading, lede, namePlaceholder, textPlaceholder string) {
	switch SiteKindByKey(kind).Key {
	case "food":
		return "Three things people come for", "Dishes, drinks, or specialities. They go on your menu page and your home page.", "Sourdough loaf", "Baked before dawn, sold until it's gone"
	case "services":
		return "Your three main services", "Say what the client gets, not how you do it. A price or a range saves everyone time.", "Kitchen fitting", "From first drawing to the last tile"
	case "shop":
		return "Three things you sell", "Your best sellers, or the three you'd show a friend first.", "Hand-thrown mug", "Stoneware, dishwasher safe, made in the studio"
	case "portfolio":
		return "Three kinds of work you do", "The work you want more of.", "Brand identity", "Logo, type, and colour for small businesses"
	case "community":
		return "Three things you do", "What a newcomer would join in with.", "Tuesday run", "5k around the park, all paces welcome"
	}
	return "Three things you offer", "The three things a visitor should know you do.", "The first thing", "One line on it"
}

// defaultPages are ticked when the owner reaches the pages step.
func defaultPages(answers SetupAnswers) map[string]bool {
	kind := SiteKindByKey(answers.Kind).Key
	// Only pages that read well from day one are ticked: a photos page with
	// no photos, or questions with placeholder answers, wait for the owner.
	pages := map[string]bool{"about": true, "privacy": true, "faq": false, "gallery": false}
	if slug, _ := mainPageFor(kind); slug != "" {
		pages["main"] = true
	}
	pages["pricing"] = answers.hasPrices() && kind != "food"
	return pages
}

func (a SetupAnswers) hasOffers() bool {
	for _, offer := range a.Offers {
		if strings.TrimSpace(offer.Name) != "" {
			return true
		}
	}
	return false
}

func (a SetupAnswers) hasPrices() bool {
	for _, offer := range a.Offers {
		if strings.TrimSpace(offer.Name) != "" && strings.TrimSpace(offer.Price) != "" {
			return true
		}
	}
	return false
}

// offers are the filled-in ones, in order.
func (a SetupAnswers) offers() []Offer {
	out := []Offer{}
	for _, offer := range a.Offers {
		if strings.TrimSpace(offer.Name) != "" {
			out = append(out, Offer{Name: strings.TrimSpace(offer.Name), Text: strings.TrimSpace(offer.Text), Price: strings.TrimSpace(offer.Price)})
		}
	}
	return out
}

// hoursItems are the rows for an hours section, or nil when the business
// has none.
func (a SetupAnswers) hoursItems() []map[string]string {
	if a.NoHours || !a.HoursSet {
		return nil
	}
	items := []map[string]string{}
	for _, day := range a.Hours {
		if day.Day == "" {
			continue
		}
		when := "Closed"
		if !day.Closed && (strings.TrimSpace(day.Open) != "" || strings.TrimSpace(day.Close) != "") {
			when = strings.TrimSpace(strings.TrimSpace(day.Open) + " – " + strings.TrimSpace(day.Close))
			when = strings.Trim(when, "– ")
		}
		items = append(items, map[string]string{"day": day.Day, "time": when})
	}
	return items
}

// pages answers "which optional pages", with the kind's defaults when the
// owner never reached that step (seeds, the agent API).
func (a SetupAnswers) pages() map[string]bool {
	if !a.PagesSet {
		return defaultPages(a)
	}
	out := map[string]bool{}
	for key, on := range a.Pages {
		out[key] = on
	}
	return out
}

// ---------- form round trip ----------

// formValues is every answer as form fields, so a step can carry the rest
// of them as hidden inputs without naming each one.
func (a SetupAnswers) formValues() url.Values {
	v := url.Values{}
	set := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			v.Set(key, value)
		}
	}
	set("siteTitle", a.SiteTitle)
	set("tagline", a.Tagline)
	set("description", a.Description)
	set("kind", a.Kind)
	set("template", a.Template)
	set("email", a.Email)
	set("phone", a.Phone)
	set("location", a.Location)
	for i, offer := range a.Offers {
		n := itoa(i + 1)
		set("offer"+n+"Name", offer.Name)
		set("offer"+n+"Text", offer.Text)
		set("offer"+n+"Price", offer.Price)
	}
	if a.HoursSet {
		v.Set("hoursSet", "1")
		for _, day := range a.Hours {
			if day.Day == "" {
				continue
			}
			key := dayKey(day.Day)
			set("hours"+key+"Open", day.Open)
			set("hours"+key+"Close", day.Close)
			if day.Closed {
				v.Set("hours"+key+"Closed", "1")
			}
		}
	}
	if a.NoHours {
		v.Set("noHours", "1")
	}
	for network, link := range a.Social {
		set("social"+network, link)
	}
	if a.PagesSet {
		v.Set("pagesSet", "1")
		for key, on := range a.Pages {
			if on {
				v.Set("page"+key, "1")
			}
		}
		if a.Publish {
			v.Set("publish", "1")
		}
	}
	return v
}

// answersFromForm reads every step's fields; the ones a step does not show
// arrive as hidden inputs.
func answersFromForm(r *http.Request) SetupAnswers {
	get := func(key string) string { return strings.TrimSpace(r.FormValue(key)) }
	answers := SetupAnswers{
		SiteTitle: get("siteTitle"), Tagline: get("tagline"), Description: get("description"), Kind: get("kind"), Template: get("template"),
		Email: get("email"), Phone: get("phone"), Location: get("location"), Social: map[string]string{}, Pages: map[string]bool{},
	}
	for i := range answers.Offers {
		n := itoa(i + 1)
		answers.Offers[i] = Offer{Name: get("offer" + n + "Name"), Text: get("offer" + n + "Text"), Price: get("offer" + n + "Price")}
	}
	answers.HoursSet = get("hoursSet") == "1"
	answers.NoHours = get("noHours") == "1"
	if answers.HoursSet {
		for i, day := range weekDays {
			key := dayKey(day)
			answers.Hours[i] = HoursDay{Day: day, Open: get("hours" + key + "Open"), Close: get("hours" + key + "Close"), Closed: get("hours"+key+"Closed") == "1"}
		}
	}
	for _, network := range []string{"instagram", "facebook", "tiktok", "youtube", "x", "linkedin"} {
		if link := safeLinkHref(get("social" + network)); link != "" {
			answers.Social[network] = link
		}
	}
	answers.PagesSet = get("pagesSet") == "1"
	if answers.PagesSet {
		for _, key := range wizardPages {
			answers.Pages[key] = get("page"+key) == "1"
		}
		answers.Publish = get("publish") == "1"
	} else {
		answers.Publish = true
	}
	return answers.trimmed()
}

// carry emits every answer as a hidden input except the fields this step
// shows for real.
func carry(answers SetupAnswers, shown ...string) gosx.Node {
	skip := map[string]bool{}
	for _, name := range shown {
		skip[name] = true
	}
	nodes := []gosx.Node{}
	values := answers.formValues()
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sortStrings(keys)
	for _, key := range keys {
		if skip[key] || skip[prefixOf(key)] {
			continue
		}
		nodes = append(nodes, hidden(key, values.Get(key)))
	}
	return gosx.Fragment(nodes...)
}

// prefixOf groups offerN*, hoursXxx*, socialXxx, pageXxx fields so a step
// can skip a whole family by its prefix.
func prefixOf(key string) string {
	for _, prefix := range []string{"offer", "hours", "social", "page", "noHours", "pagesSet", "publish"} {
		if strings.HasPrefix(key, prefix) {
			return prefix
		}
	}
	return key
}

// ---------- the new steps ----------

func wizardArea(name, label, value, placeholder string) gosx.Node {
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-field"), gosx.Attr("for", "wz-"+name)),
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", "wz-"+name), gosx.Attr("name", name), gosx.Attr("rows", "3"), gosx.Attr("placeholder", placeholder)), gosx.Text(value)))
}

func wizardCheck(name, label, hint string, checked bool) gosx.Node {
	attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", name), gosx.Attr("value", "1"), gosx.Attr("id", "wz-"+name)}
	if checked {
		attrs = append(attrs, gosx.Attr("checked", "checked"))
	}
	children := []gosx.Node{gosx.El("input", gosx.Attrs(attrs...)), gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-check__label")), gosx.Text(label))}
	if hint != "" {
		children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-check__hint")), gosx.Text(hint)))
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-check"), gosx.Attr("for", "wz-"+name)), gosx.Fragment(children...))
}

func setupStepOffers(answers SetupAnswers) gosx.Node {
	heading, lede, namePlaceholder, textPlaceholder := offerPrompt(answers.Kind)
	rows := make([]gosx.Node, 0, 3)
	for i, offer := range answers.Offers {
		n := itoa(i + 1)
		rows = append(rows, gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "wz-offer")),
			gosx.El("legend", nil, gosx.Text(itoa(i+1)+".")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "offer"+n+"Name"), gosx.Attr("value", offer.Name), gosx.Attr("placeholder", namePlaceholder), gosx.Attr("aria-label", "Name of the "+ordinal(i+1)+" offer"))),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "offer"+n+"Text"), gosx.Attr("value", offer.Text), gosx.Attr("placeholder", textPlaceholder), gosx.Attr("aria-label", "One line about it"))),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "offer"+n+"Price"), gosx.Attr("value", offer.Price), gosx.Attr("placeholder", "Price (optional)"), gosx.Attr("aria-label", "Price, optional"), gosx.Attr("class", "wz-offer__price")))))
	}
	return gosx.Fragment(
		gosx.El("h1", nil, gosx.Text(heading)),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")), gosx.Text(lede+" Leave any of them blank; you can always add more in the editor.")),
		gosx.Fragment(rows...),
		carry(answers, "offer"),
		hidden("step", "3"),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn wz-btn--ghost"), gosx.Attr("type", "submit"), gosx.Attr("name", "back"), gosx.Attr("value", "2")), gosx.Text("Back")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Next"))),
	)
}

func ordinal(n int) string {
	switch n {
	case 1:
		return "first"
	case 2:
		return "second"
	}
	return "third"
}

func setupStepWhere(answers SetupAnswers) gosx.Node {
	hours := answers.Hours
	if !answers.HoursSet {
		hours = defaultHours()
	}
	rows := make([]gosx.Node, 0, 7)
	for _, day := range hours {
		key := dayKey(day.Day)
		closedAttrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", "hours"+key+"Closed"), gosx.Attr("value", "1"), gosx.Attr("aria-label", "Closed on "+day.Day)}
		if day.Closed {
			closedAttrs = append(closedAttrs, gosx.Attr("checked", "checked"))
		}
		rows = append(rows, gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-hours__row")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-hours__day")), gosx.Text(day.Day)),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "hours"+key+"Open"), gosx.Attr("value", day.Open), gosx.Attr("placeholder", "9:00"), gosx.Attr("aria-label", day.Day+" opens at"), gosx.Attr("inputmode", "numeric"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-hours__to")), gosx.Text("to")),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "hours"+key+"Close"), gosx.Attr("value", day.Close), gosx.Attr("placeholder", "17:00"), gosx.Attr("aria-label", day.Day+" closes at"), gosx.Attr("inputmode", "numeric"))),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-hours__closed")), gosx.El("input", gosx.Attrs(closedAttrs...)), gosx.Text(" Closed"))))
	}
	return gosx.Fragment(
		gosx.El("h1", nil, gosx.Text("Where and when can people find you?")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")), gosx.Text("This fills your contact page, your footer, and what Google shows next to your name. Skip anything you'd rather not share.")),
		wizardField("email", "Email address", answers.Email, "hello@wildflower.com", false),
		wizardField("phone", "Phone number", answers.Phone, "0161 496 0000", false),
		wizardField("location", "Where you are", answers.Location, "42 Mill Lane, Oakland", false),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "wz-sub")), gosx.Text("Opening hours")),
		wizardCheck("noHours", "We don't have opening hours", "Online only, by appointment, or it just doesn't apply.", answers.NoHours),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-hours"), gosx.Attr("data-hours", "true")), gosx.Fragment(rows...)),
		hidden("hoursSet", "1"),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "wz-sub")), gosx.Text("Where else you are")),
		wizardField("socialinstagram", "Instagram", answers.Social["instagram"], "https://instagram.com/yourbusiness", false),
		wizardField("socialfacebook", "Facebook", answers.Social["facebook"], "https://facebook.com/yourbusiness", false),
		carry(answers, "email", "phone", "location", "hours", "noHours", "social"),
		hidden("step", "4"),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn wz-btn--ghost"), gosx.Attr("type", "submit"), gosx.Attr("name", "back"), gosx.Attr("value", "3")), gosx.Text("Back")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Next"))),
	)
}

func setupStepPages(answers SetupAnswers) gosx.Node {
	pages := answers.pages()
	mainSlug, mainTitle := mainPageFor(answers.Kind)
	name := firstNonEmpty(answers.SiteTitle, "your business")
	items := []gosx.Node{
		wizardCheck("pagehome", "Home", "Always. Your tagline, what you offer, and how to reach you.", true),
	}
	if mainSlug != "" {
		hint := "Built from what you offer."
		if !answers.hasOffers() {
			hint = "With room to list what you offer."
		}
		items = append(items, wizardCheck("pagemain", mainTitle, hint, pages["main"]))
	}
	items = append(items,
		wizardCheck("pageabout", "About", "Who you are, from your description.", pages["about"]),
	)
	if answers.hasPrices() {
		items = append(items, wizardCheck("pagepricing", "Pricing", "The offers you priced, as plans.", pages["pricing"]))
	}
	items = append(items,
		wizardCheck("pagegallery", "Photos", "A gallery page ready for your pictures.", pages["gallery"]),
		wizardCheck("pagefaq", "Questions and answers", "The three questions people always ask, ready to answer.", pages["faq"]),
		wizardCheck("pagecontact", "Contact", "Always. Email, phone, address, hours, a map, and a form that works.", true),
		wizardCheck("pageprivacy", "Privacy policy", "A plain-words policy for "+name+", kept out of the menu.", pages["privacy"]),
	)
	return gosx.Fragment(
		gosx.El("h1", nil, gosx.Text("Which pages do you want?")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")), gosx.Text("We've ticked what a business like yours usually needs. Each page comes filled in from your answers; you can add, rename, or delete any of them later.")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-pages")), gosx.Fragment(items...)),
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "wz-sub")), gosx.Text("Go live?")),
		wizardCheck("publish", "Publish everything now", "Your site is live the moment it's built. Untick to keep it all as drafts until you've had a look.", !answers.PagesSet || answers.Publish),
		hidden("pagesSet", "1"),
		carry(answers, "page", "pagesSet", "publish"),
		hidden("step", "6"),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn wz-btn--ghost"), gosx.Attr("type", "submit"), gosx.Attr("name", "back"), gosx.Attr("value", "5")), gosx.Text("Back")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Build my site"))),
	)
}

// ---------- page builders ----------

func (a SetupAnswers) featureItems(fallback []map[string]string) []map[string]string {
	offers := a.offers()
	if len(offers) == 0 {
		return fallback
	}
	items := make([]map[string]string, 0, len(offers))
	for i, offer := range offers {
		text := offer.Text
		if offer.Price != "" && !a.pages()["pricing"] {
			text = strings.TrimSpace(text + " — " + offer.Price)
			text = strings.TrimPrefix(text, "— ")
		}
		items = append(items, map[string]string{"icon": itoa(i + 1), "title": offer.Name, "text": text})
	}
	return items
}

func (a SetupAnswers) pricingItems(cardStyle bool) []map[string]string {
	items := []map[string]string{}
	for _, offer := range a.offers() {
		item := map[string]string{"name": offer.Name, "price": offer.Price, "blurb": offer.Text}
		if cardStyle {
			item["button"] = "Get in touch"
			item["url"] = "/contact"
		}
		items = append(items, item)
	}
	return items
}

// faqFor is three questions a business of this kind is always asked. The
// answers say what to write, in the owner's voice.
func faqFor(kind string, answers SetupAnswers) []map[string]string {
	switch SiteKindByKey(kind).Key {
	case "food":
		return []map[string]string{
			{"question": "Do you take bookings?", "answer": "Say whether people can book, how (phone, email, a link), and for how many."},
			{"question": "Can you cater for allergies?", "answer": "List what you can do, and ask people to mention allergies when they order."},
			{"question": "Is there parking nearby?", "answer": firstNonEmpty(answers.Location, "Say where the nearest parking and public transport are.")},
		}
	case "services":
		return []map[string]string{
			{"question": "How much does it cost?", "answer": "Give a range or a starting price. It saves everyone a call."},
			{"question": "How soon can you start?", "answer": "Your usual lead time, and what happens first after someone gets in touch."},
			{"question": "Which areas do you cover?", "answer": firstNonEmpty(answers.Location, "The towns or the radius you work in.")},
		}
	case "shop":
		return []map[string]string{
			{"question": "How long does delivery take?", "answer": "Where you ship, how long it takes, and what it costs."},
			{"question": "Can I return something?", "answer": "Your returns window and how to start one."},
			{"question": "Do you make things to order?", "answer": "Whether you take custom requests and how to ask."},
		}
	case "community":
		return []map[string]string{
			{"question": "Do I need to be a member?", "answer": "Whether newcomers can just turn up, and what membership involves."},
			{"question": "Is there a cost?", "answer": "Any fee, and what it covers."},
			{"question": "What should I bring?", "answer": "Anything a first-timer needs on the day."},
		}
	}
	return []map[string]string{
		{"question": "What exactly do you do?", "answer": "Two sentences, in plain words."},
		{"question": "How do I get started?", "answer": "The first step, and what happens after it."},
		{"question": "How can I reach you?", "answer": firstNonEmpty(contactLine(answers), "The best way to get hold of you.")},
	}
}

// turnkeyPages are the optional pages built from the answers, in menu
// order, after the kind's own home page.
func turnkeyPages(name string, answers SetupAnswers, kind string, extras []StarterPage) []StarterPage {
	pages := answers.pages()
	out := []StarterPage{}
	mainSlug, mainTitle := mainPageFor(kind)
	if mainSlug != "" && pages["main"] {
		out = append(out, mainPage(name, mainSlug, mainTitle, kind, answers))
	}
	out = append(out, extras...)
	if pages["about"] {
		about := document(
			heading(0, 2, "About "+name),
			para(1, firstNonEmpty(answers.Description, "Who you are and how you got here. People buy from people, so write this the way you'd tell a customer standing in front of you.")),
			para(2, "What you care about, what you're known for, and what someone can expect when they get in touch."),
			quote(3, "Swap this for something a real customer said about you."),
		)
		out = append(out, StarterPage{Slug: "about", Title: "About", Description: "The story behind " + name + ".", Publish: answers.Publish, Body: about})
	}
	if pages["pricing"] && answers.hasPrices() {
		out = append(out, StarterPage{Slug: "pricing", Title: "Pricing", Description: "What " + name + " charges.", Publish: answers.Publish, Body: document(
			comp(0, "pricing", "cards", map[string]string{"heading": "Pricing", "intro": "Straight answers, no surprises."}, answers.pricingItems(true)),
			comp(1, "cta", "band", map[string]string{"headline": "Not sure which is right?", "text": "Tell us what you need and we'll point you the right way.", "button": "Get in touch", "url": "/contact"}, nil),
		)})
	}
	if pages["gallery"] {
		gallery := blockstudio.BlockInstance{ID: content.BlockGallery + "-1", Key: content.BlockGallery, Enabled: true, Order: 1, Values: blockstudio.Values{"images": galleryValue(nil), "style": text("grid")}}
		out = append(out, StarterPage{Slug: "photos", Title: "Photos", Description: "Pictures from " + name + ".", Publish: answers.Publish, Body: document(
			heading(0, 2, "Photos"),
			gallery,
			para(2, "Add your pictures above: the ones that show what it's like to be here, not the stock ones."),
		)})
	}
	if pages["faq"] {
		out = append(out, StarterPage{Slug: "questions", Title: "Questions", Description: "Questions people ask " + name + ".", Publish: answers.Publish, Body: document(
			comp(0, "faq", "accordion", map[string]string{"heading": "Questions people ask"}, faqFor(kind, answers)),
			comp(1, "cta", "band", map[string]string{"headline": "Something else?", "text": "Ask us anything.", "button": "Get in touch", "url": "/contact"}, nil),
		)})
	}
	return out
}

func mainPage(name, slug, title, kind string, answers SetupAnswers) StarterPage {
	var body blockstudio.Document
	switch kind {
	case "food":
		if answers.hasOffers() {
			intro := "What we make, and what it costs."
			if !answers.hasPrices() {
				intro = "What we make."
			}
			body = document(
				comp(0, "pricing", "simple", map[string]string{"heading": "Menu", "intro": intro}, answers.pricingItems(false)),
				para(1, "Add the rest of the menu here. Keep it short: people scan menus, they don't read them. Note what you can do for allergies."),
			)
		} else {
			body = document(
				heading(0, 2, "Menu"),
				heading(1, 3, "Mornings"),
				para(2, "List what you serve and what it costs. Keep it short — people scan menus, they don't read them."),
				heading(3, 3, "Afternoons"),
				para(4, "Add a line about anything you're known for, and note what you can do for allergies."),
			)
		}
	case "services":
		body = document(
			comp(0, "features", "cards", map[string]string{"heading": "What I do", "intro": firstNonEmpty(answers.Description, "")}, answers.featureItems([]map[string]string{
				{"icon": "1", "title": "The first service", "text": "Say what the client gets, not how you do it."},
				{"icon": "2", "title": "The second service", "text": "One or two sentences is plenty."},
				{"icon": "3", "title": "The third service", "text": "If you can name a price or a range, do."}})),
			heading(1, 3, "How it works"),
			para(2, "Walk through what happens after someone gets in touch. Removing the mystery is the most persuasive thing on most service websites."),
			comp(3, "cta", "band", map[string]string{"headline": "Tell me about the job", "text": "I reply within a working day.", "button": "Get in touch", "url": "/contact"}, nil),
		)
	case "portfolio":
		body = document(
			heading(0, 2, "Selected work"),
			para(1, "Show six to ten pieces, not everything. For each one, a line on what it was and what you did."),
			comp(2, "features", "plain", map[string]string{"heading": "What I do"}, answers.featureItems([]map[string]string{
				{"icon": "✦", "title": "The first thing", "text": "A line on it."},
				{"icon": "✦", "title": "The second thing", "text": "A line on it."},
				{"icon": "✦", "title": "The third thing", "text": "A line on it."}})),
			para(3, "Put your strongest piece first. Most people never scroll to the bottom."),
		)
	case "community":
		body = document(
			heading(0, 2, "What's coming up"),
			para(1, "List the next few things with dates and times. Keep past events off this page."),
			comp(2, "features", "numbered", map[string]string{"heading": "What we do"}, answers.featureItems([]map[string]string{
				{"icon": "", "title": "Come along", "text": "When to arrive and where to go."},
				{"icon": "", "title": "Say hello", "text": "Who to look for when you get there."},
				{"icon": "", "title": "Join in", "text": "What happens, and whether you need to bring anything."}})),
		)
	}
	return StarterPage{Slug: slug, Title: title, Description: title + " at " + name + ".", Publish: answers.Publish, Body: body}
}

// privacyPage is the plain-words policy, out of the menu but linked from
// the footer.
func privacyPage(name string, answers SetupAnswers, sells bool) StarterPage {
	blocks := []blockstudio.BlockInstance{}
	for i, line := range privacyStarter(name, sells) {
		blocks = append(blocks, para(i, line))
	}
	return StarterPage{Slug: privacySlug, Title: "Privacy", Description: "What " + name + " keeps about visitors, and why.", Publish: answers.Publish, HideFromMenu: true, Body: document(blocks...)}
}

// ---------- after the build: the welcome ----------

// renderWelcomePanel is what the owner sees the first time the dashboard
// opens: what was built, whether it is live, and the next three things.
func (h *Host) renderWelcomePanel(r *http.Request) gosx.Node {
	settings := h.settings()
	base := h.absoluteBase(r)
	pages, _ := h.store.ListPages(cmsstore.PageFilter{})
	rows := make([]gosx.Node, 0, len(pages))
	live := 0
	for _, page := range pages {
		state := "Draft"
		if h.isLive(page) {
			state = "Live"
			live++
		}
		rows = append(rows, gosx.El("li", gosx.Attrs(gosx.Attr("class", "admin-welcome__page")),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-welcome__state"), gosx.Attr("data-live", boolAttr(state == "Live"))), gosx.Text(state)),
			gosx.El("strong", nil, gosx.Text(page.Title)),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", "/admin/edit/"+page.ID)), gosx.Text("Edit")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", publicPath(page.Slug)), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("View"))))
	}
	lede := "Your site is live at " + base + ". Everything below is on it now; change anything and publish again."
	if live == 0 {
		lede = "Your pages are built and saved as drafts. Nothing is public yet: open a page, have a look, and press Publish when you're happy."
	}
	steps := []gosx.Node{
		gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", h.homeEditHref())), gosx.Text("Add your photos")), gosx.Text(" — the home page and the Photos page have room for them. Real pictures of your place do more than any words.")),
		gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/pages")), gosx.Text("Read each page once")), gosx.Text(" — every line is written to be replaced with your own; the grey hints tell you what goes where.")),
		gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/domain")), gosx.Text("Connect your domain")), gosx.Text(" — so people find you at your own address.")),
	}
	if h.featureOn(FeatureTeam) {
		steps = append(steps, gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", peoplePath)), gosx.Text("Invite someone to help")), gosx.Text(" — editors can change pages; only you can publish.")))
	}
	if h.featureOn(FeatureBlog) {
		steps = append(steps, gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/posts")), gosx.Text("Write a first post")), gosx.Text(" — news, an opening, a season: the blog appears in the menu with its first post.")))
	}
	if h.featureOn(FeatureShop) {
		steps = append(steps, gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/shop")), gosx.Text("Add something to sell")), gosx.Text(" — the shop joins the menu with its first product.")))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel admin-welcome"), gosx.Attr("data-welcome", "true")),
		gosx.El("h2", nil, gosx.Text("Your site is ready, "+firstNonEmpty(settings.Title, "friend"))),
		gosx.El("p", nil, gosx.Text(lede)),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "admin-welcome__pages")), gosx.Fragment(rows...)),
		gosx.El("h3", nil, gosx.Text("The next few things")),
		gosx.El("ol", gosx.Attrs(gosx.Attr("class", "admin-list")), gosx.Fragment(steps...)),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Press Ctrl+K in the editor to do anything by typing: add a section, publish, switch page.")))
}
