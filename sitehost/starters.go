package sitehost

import (
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
)

// starters.go is the template gallery: a starting point is a kind of
// business plus a look plus a shape for the home page.
//
// Squarespace-class tools open with a wall of designs. Most of the
// difference between those designs is colour, type, corner shape, spacing,
// and whether the home page has a coloured band — exactly the dials this
// host already has. So a template here is a named bundle of those dials over
// the starter pages for a kind of business, and the wizard shows the three
// that suit what the owner said they do.

// Template is one starting point in the gallery.
type Template struct {
	Key     string
	Label   string
	Blurb   string
	Kind    string // SiteKind key
	Palette string
	Fonts   string
	Buttons string
	Spacing string
	Band    string // "", "tinted", or "accent": a background band on the home page
}

// Templates are every starting point, grouped by kind. The first of each
// kind is that kind's default.
func Templates() []Template {
	return []Template{
		// I sell things
		{Key: "market", Label: "Market", Blurb: "Warm and friendly, like a good stall.", Kind: "shop", Palette: "warm", Fonts: "friendly", Buttons: "rounded", Spacing: "regular", Band: "tinted"},
		{Key: "gallery", Label: "Gallery", Blurb: "Quiet and spacious, so the things you make stand out.", Kind: "shop", Palette: "slate", Fonts: "editorial", Buttons: "square", Spacing: "airy"},
		{Key: "bold-goods", Label: "Bold goods", Blurb: "Big colour, big buttons, no hesitation.", Kind: "shop", Palette: "bold", Fonts: "modern", Buttons: "pill", Spacing: "compact", Band: "accent"},
		// I offer a service
		{Key: "consult", Label: "Consult", Blurb: "Clean and trustworthy. Lets your words do the work.", Kind: "services", Palette: "cool", Fonts: "clean", Buttons: "square", Spacing: "regular"},
		{Key: "studio", Label: "Studio", Blurb: "Dark and confident, for creative work.", Kind: "services", Palette: "night", Fonts: "modern", Buttons: "rounded", Spacing: "airy", Band: "tinted"},
		{Key: "trade", Label: "Trade", Blurb: "Friendly and direct, with a clear way to get in touch.", Kind: "services", Palette: "fresh", Fonts: "friendly", Buttons: "pill", Spacing: "regular", Band: "accent"},
		// I run a place people visit
		{Key: "bakery", Label: "Bakery", Blurb: "Warm tones and a classic face. Menu first.", Kind: "food", Palette: "warm", Fonts: "classic", Buttons: "rounded", Spacing: "regular", Band: "tinted"},
		{Key: "bistro", Label: "Bistro", Blurb: "Dark, elegant, unhurried.", Kind: "food", Palette: "night", Fonts: "editorial", Buttons: "square", Spacing: "airy"},
		{Key: "corner-cafe", Label: "Corner café", Blurb: "Bright and cheerful, easy to read on a phone in the queue.", Kind: "food", Palette: "fresh", Fonts: "friendly", Buttons: "pill", Spacing: "regular", Band: "accent"},
		// I show my work
		{Key: "monograph", Label: "Monograph", Blurb: "Lots of room. The work is the point.", Kind: "portfolio", Palette: "slate", Fonts: "editorial", Buttons: "square", Spacing: "airy"},
		{Key: "bright", Label: "Bright", Blurb: "A strong accent and modern type.", Kind: "portfolio", Palette: "bold", Fonts: "modern", Buttons: "rounded", Spacing: "regular", Band: "accent"},
		{Key: "quiet", Label: "Quiet", Blurb: "Cool, calm, classic.", Kind: "portfolio", Palette: "cool", Fonts: "classic", Buttons: "square", Spacing: "airy", Band: "tinted"},
		// I run a group or organisation
		{Key: "gather", Label: "Gather", Blurb: "Fresh and welcoming. Easy for newcomers.", Kind: "community", Palette: "fresh", Fonts: "friendly", Buttons: "rounded", Spacing: "regular", Band: "tinted"},
		{Key: "chapel", Label: "Chapel", Blurb: "Classic type and a steady, traditional feel.", Kind: "community", Palette: "slate", Fonts: "classic", Buttons: "square", Spacing: "regular"},
		{Key: "rally", Label: "Rally", Blurb: "Bold and energetic, built to get people moving.", Kind: "community", Palette: "bold", Fonts: "modern", Buttons: "pill", Spacing: "compact", Band: "accent"},
		// Something else
		{Key: "plain", Label: "Plain", Blurb: "Clean and neutral. Nothing to undo later.", Kind: "simple", Palette: "fresh", Fonts: "clean", Buttons: "square", Spacing: "regular"},
		{Key: "ink", Label: "Ink", Blurb: "Dark with editorial type, for people who write.", Kind: "simple", Palette: "night", Fonts: "editorial", Buttons: "rounded", Spacing: "airy"},
		{Key: "sunny", Label: "Sunny", Blurb: "Warm, round, and friendly.", Kind: "simple", Palette: "warm", Fonts: "friendly", Buttons: "pill", Spacing: "regular", Band: "tinted"},
	}
}

// TemplatesForKind lists the starting points that suit one kind of site.
func TemplatesForKind(kind string) []Template {
	kind = SiteKindByKey(kind).Key
	out := make([]Template, 0, 3)
	for _, template := range Templates() {
		if template.Kind == kind {
			out = append(out, template)
		}
	}
	return out
}

// TemplateByKey finds a template by its key.
func TemplateByKey(key string) (Template, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, template := range Templates() {
		if template.Key == key {
			return template, true
		}
	}
	return Template{}, false
}

// templateFor resolves the owner's choice, falling back to the kind's
// default when the choice is missing or belongs to another kind.
func templateFor(answers SetupAnswers) Template {
	kind := SiteKindByKey(answers.Kind).Key
	if template, ok := TemplateByKey(answers.Template); ok && template.Kind == kind {
		return template
	}
	return TemplatesForKind(kind)[0]
}

// theme is the Look this template starts the site with.
func (t Template) theme() Theme {
	return Theme{Palette: PaletteByKey(t.Palette), Fonts: FontPairByKey(t.Fonts), Buttons: ButtonShapeByKey(t.Buttons).Key, Spacing: SpacingScaleByKey(t.Spacing).Key}
}

// bandHome puts the second half of the home page — the part after the
// opening line and its button — on a coloured band.
func bandHome(doc blockstudio.Document, band string) blockstudio.Document {
	if band == "" || len(doc.Blocks) < 3 {
		return doc
	}
	blocks := make([]blockstudio.BlockInstance, 0, len(doc.Blocks)+1)
	for index, instance := range doc.Blocks {
		if index == 2 {
			blocks = append(blocks, block(len(blocks), blockSection, values("style", normalizeSectionStyle(band))))
		}
		blocks = append(blocks, block(len(blocks), instance.Key, instance.Values))
	}
	doc.Blocks = blocks
	return doc
}
