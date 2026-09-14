package sitehost

import (
	"net/http"
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// setup.go is the first-run wizard.
//
// Three questions, one screen each, then a real website. Answers carry forward
// in hidden fields, so the wizard needs no session, no cookie, and no
// JavaScript — a person can back up, change an answer, and continue.
//
// The site is not seeded until the wizard finishes. A generic starter site that
// appears before the owner has said anything about their business is a thing
// they have to undo rather than something they can build on.

const setupCompleteKey = "setupComplete"

// SetupComplete reports whether the wizard has been finished for this site.
func (h *Host) SetupComplete() bool {
	settings, ok, err := h.store.SiteSettings()
	if err != nil || !ok {
		return false
	}
	return strings.TrimSpace(settings.Metadata[setupCompleteKey]) == "true"
}

func (h *Host) mountSetup(mux *http.ServeMux) {
	mux.HandleFunc("GET /setup", h.handleSetupStep)
	mux.HandleFunc("GET /setup/{$}", h.handleSetupStep)
	mux.HandleFunc("POST /setup", h.handleSetupStep)
	mux.HandleFunc("POST /setup/{$}", h.handleSetupStep)
}

func (h *Host) handleSetupStep(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	answers := answersFromForm(r)
	step := strings.TrimSpace(r.FormValue("step"))
	if step == "" {
		step = "1"
	}

	// "back" moves without validating, so a half-typed answer is never lost.
	if r.Method == http.MethodPost && r.FormValue("back") != "" {
		h.renderSetup(w, previousStep(step), answers, "")
		return
	}
	if r.Method != http.MethodPost {
		if h.SetupComplete() {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
		h.renderSetup(w, "1", answers, "")
		return
	}

	switch step {
	case "1":
		if answers.SiteTitle == "" {
			h.renderSetup(w, "1", answers, "Your site needs a name before we can build it.")
			return
		}
		h.renderSetup(w, "2", answers, "")
	case "2":
		if answers.Kind == "" {
			h.renderSetup(w, "2", answers, "Pick whichever is closest. You can change everything afterwards.")
			return
		}
		h.renderSetup(w, "3", answers, "")
	case "3":
		h.renderSetup(w, "4", answers, "")
	case "4":
		if answers.Email != "" && !strings.Contains(answers.Email, "@") {
			h.renderSetup(w, "4", answers, "That email address doesn't look right.")
			return
		}
		h.renderSetup(w, "5", answers, "")
	case "5":
		// A look is always chosen: the first card is preselected, and an
		// unknown key falls back to the kind's default.
		answers.Template = templateFor(answers).Key
		h.renderSetup(w, "6", answers, "")
	case "6":
		if err := h.CompleteSetup(answers); err != nil {
			h.renderSetup(w, "6", answers, "Something went wrong building your site. Try again.")
			return
		}
		http.Redirect(w, r, "/admin?welcome=1", http.StatusSeeOther)
	default:
		h.renderSetup(w, "1", answers, "")
	}
}

func previousStep(step string) string {
	switch step {
	case "6":
		return "5"
	case "5":
		return "4"
	case "4":
		return "3"
	case "3":
		return "2"
	case "2":
		return "1"
	default:
		return "1"
	}
}

// CompleteSetup writes the answers and builds the starter site.
func (h *Host) CompleteSetup(answers SetupAnswers) error {
	answers = answers.trimmed()
	if answers.BaseURL == "" {
		answers.BaseURL = h.opts.BaseURL
	}

	metadata := cmsstore.Metadata{setupCompleteKey: "true"}
	if answers.Email != "" {
		metadata["contactEmail"] = answers.Email
	}
	if answers.Phone != "" {
		metadata["contactPhone"] = answers.Phone
	}
	if answers.Location != "" {
		metadata["contactLocation"] = answers.Location
	}
	if answers.Kind != "" {
		metadata["siteKind"] = SiteKindByKey(answers.Kind).Key
	}
	if answers.Tagline != "" {
		metadata["tagline"] = answers.Tagline
	}
	for network, link := range answers.Social {
		if safe := safeLinkHref(link); safe != "" {
			metadata[socialKey(network)] = safe
		}
	}
	// A header button that does the one thing most visitors came for.
	switch {
	case answers.Phone != "":
		metadata[menuButtonKey] = "Call us"
		metadata[menuButtonURLKey] = "tel:" + strings.ReplaceAll(answers.Phone, " ", "")
	case answers.Email != "":
		metadata[menuButtonKey] = "Email us"
		metadata[menuButtonURLKey] = "mailto:" + answers.Email
	}
	metadata[footerMenuKey] = "true"
	// The chosen starting point sets the Look; the owner changes any of it
	// later from the editor.
	template := templateFor(answers)
	metadata["siteTemplate"] = template.Key
	metadata[themePaletteKey] = template.Palette
	metadata[themeFontsKey] = template.Fonts
	metadata[themeButtonsKey] = template.Buttons
	metadata[themeSpacingKey] = template.Spacing

	if _, err := h.store.SaveSiteSettings(cmsstore.SiteSettingsInput{
		Title:       answers.SiteTitle,
		Description: answers.Tagline,
		BaseURL:     answers.BaseURL,
		Locale:      "en",
		Metadata:    metadata,
	}); err != nil {
		return err
	}
	if _, _, err := h.store.PublishSiteSettings(); err != nil {
		return err
	}

	existing, err := h.store.ListPages(cmsstore.PageFilter{})
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	for _, starter := range StarterSiteFor(answers) {
		pageMetadata := cmsstore.Metadata{"metaDescription": starter.Description}
		if starter.HideFromMenu {
			pageMetadata[pageNavHiddenKey] = "true"
		}
		page, err := h.store.CreatePage(cmsstore.PageInput{
			Slug:        starter.Slug,
			Title:       starter.Title,
			Description: starter.Description,
			Body:        starter.Body,
			Metadata:    pageMetadata,
		})
		if err != nil {
			return err
		}
		if starter.Publish {
			if _, _, err := h.store.PublishPage(page.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ---------- rendering ----------

func (h *Host) renderSetup(w http.ResponseWriter, step string, answers SetupAnswers, problem string) {
	meta := PageMeta{
		Title:     "Set up your site",
		SiteTitle: firstNonEmpty(answers.SiteTitle, h.opts.SiteTitle),
		NoIndex:   true,
	}

	var panel gosx.Node
	var extras gosx.Node = gosx.Fragment()
	switch step {
	case "2":
		panel = setupStepTwo(answers)
	case "3":
		panel = setupStepOffers(answers)
	case "4":
		panel = setupStepWhere(answers)
	case "5":
		panel = setupStepLook(answers)
		if fonts := fontsPreviewURL(); fonts != "" {
			extras = gosx.El("link", gosx.Attrs(gosx.Attr("rel", "stylesheet"), gosx.Attr("href", fonts)))
		}
	case "6":
		panel = setupStepPages(answers)
	default:
		panel = setupStepOne(answers)
	}

	body := gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz")),
		extras,
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-card")),
			setupProgress(step),
			problemNote(problem),
			gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/setup"), gosx.Attr("class", "wz-form")),
				panel,
			),
		),
	)
	h.writeDocument(w, http.StatusOK, meta, body)
}

func problemNote(problem string) gosx.Node {
	if strings.TrimSpace(problem) == "" {
		return gosx.Fragment()
	}
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-problem"), gosx.Attr("role", "alert")), gosx.Text(problem))
}

func setupProgress(step string) gosx.Node {
	steps := []struct{ Key, Label string }{
		{"1", "Your business"},
		{"2", "What you do"},
		{"3", "What you offer"},
		{"4", "Where and when"},
		{"5", "Your look"},
		{"6", "Your pages"},
	}
	dots := make([]gosx.Node, 0, len(steps))
	for _, item := range steps {
		attrs := []any{gosx.Attr("class", "wz-step")}
		if item.Key == step {
			attrs = append(attrs, gosx.Attr("data-state", "current"), gosx.Attr("aria-current", "step"))
		} else if item.Key < step {
			attrs = append(attrs, gosx.Attr("data-state", "done"))
		}
		dots = append(dots, gosx.El("li", gosx.Attrs(attrs...), gosx.Text(item.Label)))
	}
	return gosx.El("ol", gosx.Attrs(gosx.Attr("class", "wz-progress"), gosx.Attr("aria-label", "Setup progress")),
		gosx.Fragment(dots...))
}

func hidden(name, value string) gosx.Node {
	if strings.TrimSpace(value) == "" {
		return gosx.Fragment()
	}
	return gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", name), gosx.Attr("value", value)))
}

func setupStepOne(answers SetupAnswers) gosx.Node {
	return gosx.Fragment(
		gosx.El("h1", nil, gosx.Text("Let's build your website")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")),
			gosx.Text("A few questions, and you'll have a real site: pages written from your answers, live if you want it. Nothing here is permanent.")),
		wizardField("siteTitle", "What's your business called?", answers.SiteTitle, "Wildflower Bakery", true),
		wizardField("tagline", "What do you do, in one line?", answers.Tagline,
			"Sourdough and pastries, baked every morning in Oakland", false),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-hint")),
			gosx.Text("This line shows up in Google results and when someone shares your site.")),
		wizardArea("description", "Tell us a little more (optional)", answers.Description, "Who you are, what you do, and who it's for. Two or three sentences; they go on your About page and your home page."),
		carry(answers, "siteTitle", "tagline", "description"),
		hidden("step", "1"),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Next")),
		),
	)
}

func setupStepTwo(answers SetupAnswers) gosx.Node {
	cards := make([]gosx.Node, 0, 6)
	for _, kind := range SiteKinds() {
		inputAttrs := []any{
			gosx.Attr("type", "radio"),
			gosx.Attr("name", "kind"),
			gosx.Attr("id", "kind-"+kind.Key),
			gosx.Attr("value", kind.Key),
		}
		if answers.Kind == kind.Key {
			inputAttrs = append(inputAttrs, gosx.Attr("checked", "checked"))
		}
		cards = append(cards, gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-choice"), gosx.Attr("for", "kind-"+kind.Key)),
			gosx.El("input", gosx.Attrs(inputAttrs...)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-choice__label")), gosx.Text(kind.Label)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-choice__blurb")), gosx.Text(kind.Blurb)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-choice__eg")), gosx.Text(kind.Examples)),
		))
	}

	return gosx.Fragment(
		gosx.El("h1", nil, gosx.Text("Which sounds most like you?")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")),
			gosx.Text("This decides which pages we start you with. Pick the closest one — you can add, rename, and delete pages afterwards.")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-choices")), gosx.Fragment(cards...)),
		carry(answers, "kind"),
		hidden("step", "2"),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn wz-btn--ghost"), gosx.Attr("type", "submit"), gosx.Attr("name", "back"), gosx.Attr("value", "1")), gosx.Text("Back")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Next")),
		),
	)
}

func wizardField(name, label, value, placeholder string, autofocus bool) gosx.Node {
	attrs := []any{
		gosx.Attr("type", "text"),
		gosx.Attr("id", "wz-"+name),
		gosx.Attr("name", name),
		gosx.Attr("value", value),
		gosx.Attr("placeholder", placeholder),
		gosx.Attr("autocomplete", "off"),
	}
	if autofocus {
		attrs = append(attrs, gosx.Attr("autofocus", "autofocus"))
	}
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-field"), gosx.Attr("for", "wz-"+name)),
		gosx.El("span", nil, gosx.Text(label)),
		gosx.El("input", gosx.Attrs(attrs...)),
	)
}

// requireSetup sends an unconfigured site to the wizard instead of showing a
// half-built admin area or an empty public page.
func (h *Host) requireSetup(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		exempt := strings.HasPrefix(path, "/setup") ||
			strings.HasPrefix(path, "/_gosx/") ||
			strings.HasPrefix(path, uploadsURLPrefix) ||
			strings.HasPrefix(path, "/platform/") ||
			strings.HasPrefix(path, agentPathPrefix+"/") ||
			path == sitemapPath || path == robotsPath ||
			path == "/healthz"
		if exempt || h.SetupComplete() {
			next.ServeHTTP(w, r)
			return
		}
		if isAdminPath(path) {
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}
		h.renderUnbuiltSite(w)
	})
}

// renderUnbuiltSite is what a visitor sees before the owner finishes setup.
func (h *Host) renderUnbuiltSite(w http.ResponseWriter) {
	meta := PageMeta{Title: "Coming soon", SiteTitle: h.opts.SiteTitle, NoIndex: true}
	body := gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-card wz-card--quiet")),
			gosx.El("h1", nil, gosx.Text("Coming soon")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")),
				gosx.Text("This site hasn't been set up yet.")),
			gosx.El("p", nil,
				gosx.El("a", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("href", "/setup")), gosx.Text("Set it up"))),
		),
	)
	h.writeDocument(w, http.StatusOK, meta, body)
}

// setupStepLook is the template gallery: the three starting points that
// suit what the owner does, each drawn from its own palette and type.
func setupStepLook(answers SetupAnswers) gosx.Node {
	templates := TemplatesForKind(answers.Kind)
	chosen := templateFor(answers).Key
	cards := make([]gosx.Node, 0, len(templates))
	for _, template := range templates {
		palette := PaletteByKey(template.Palette)
		fonts := FontPairByKey(template.Fonts)
		inputAttrs := []any{
			gosx.Attr("type", "radio"), gosx.Attr("name", "template"),
			gosx.Attr("id", "template-"+template.Key), gosx.Attr("value", template.Key),
		}
		if template.Key == chosen {
			inputAttrs = append(inputAttrs, gosx.Attr("checked", "checked"))
		}
		radius := ButtonShapeByKey(template.Buttons).Radius
		cards = append(cards, gosx.El("label", gosx.Attrs(gosx.Attr("class", "wz-choice wz-choice--look"), gosx.Attr("for", "template-"+template.Key)),
			gosx.El("input", gosx.Attrs(inputAttrs...)),
			// A miniature of the look: its ground, a heading in its display
			// face, a line of body text, and a button in its accent and shape.
			// The card also carries the backdrop a site of this kind starts
			// with, drawn in this palette and seeded by the site's name, so the
			// owner sees the real thing before choosing.
			gosx.El("span", gosx.Attrs(append([]any{gosx.Attr("class", "wz-look"), gosx.Attr("aria-hidden", "true"),
				gosx.Attr("style", "display:block;padding:14px;border-radius:3px;background:"+palette.Ground+";color:"+palette.Ink+";border:1px solid "+palette.Rule+";--site-accent:"+palette.Accent+";--site-ground:"+palette.Ground+";--site-ink:"+palette.Ink)},
				fxAttrs(fxDefaultFor(answers.Kind), "slow", "subtle", fxSeedFor(firstNonEmpty(answers.SiteTitle, template.Key)))...)...),
				gosx.El("span", gosx.Attrs(gosx.Attr("style", "display:block;font-family:"+fonts.Display+";font-weight:700;font-size:18px;line-height:1.1;margin-bottom:6px")), gosx.Text(firstNonEmpty(answers.SiteTitle, "Your name here"))),
				gosx.El("span", gosx.Attrs(gosx.Attr("style", "display:block;font-family:"+fonts.Body+";font-size:11.5px;color:"+palette.Muted+";margin-bottom:10px")), gosx.Text("A short line about what you do.")),
				gosx.El("span", gosx.Attrs(gosx.Attr("style", "display:inline-block;padding:5px 11px;font-size:11px;font-family:"+fonts.Body+";background:"+palette.Accent+";color:"+palette.Ground+";border-radius:"+radius)), gosx.Text("Get in touch")),
			),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-choice__label")), gosx.Text(template.Label)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "wz-choice__blurb")), gosx.Text(template.Blurb)),
		))
	}
	return gosx.Fragment(
		gosx.El("h1", nil, gosx.Text("Pick a look")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "wz-lede")),
			gosx.Text("Three starting points that suit what you do. Colours, fonts, and shapes can all be changed later, from the editor, while you watch.")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-choices")), gosx.Fragment(cards...)),
		carry(answers, "template"),
		hidden("step", "5"),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "wz-actions")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn wz-btn--ghost"), gosx.Attr("type", "submit"), gosx.Attr("name", "back"), gosx.Attr("value", "4")), gosx.Text("Back")),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "wz-btn"), gosx.Attr("type", "submit")), gosx.Text("Next")),
		),
	)
}
