package sitehost

import (
	"regexp"
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// theme.go is the site-wide Look: one palette, one font pairing, one accent.
//
// A non-technical owner does not want six colour pickers and a type scale.
// They want to say "warm" or "bold", pick a font that feels like them, and
// maybe nudge the accent to match a logo. So the Look is a small set of
// curated palettes and pairings plus one free colour, stored on the site's
// settings and rendered as CSS custom properties. Every page — and the editor
// canvas — reads the same variables, so a change lands everywhere at once.

// Palette is one curated colour scheme.
type Palette struct {
	Key     string
	Label   string
	Blurb   string
	Scheme  string // "light" or "dark"
	Ground  string
	Surface string
	Ink     string
	Muted   string
	Rule    string
	Accent  string
}

// FontPair is one curated heading/body pairing.
type FontPair struct {
	Key     string
	Label   string
	Blurb   string
	Display string   // CSS font-family stack for headings
	Body    string   // CSS font-family stack for running text
	Google  []string // Google Fonts family specs, empty for system stacks
}

// Theme is the resolved Look for a site.
type Theme struct {
	Palette Palette
	Fonts   FontPair
	Accent  string // hex, overrides Palette.Accent when set
}

const (
	themePaletteKey = "themePalette"
	themeFontsKey   = "themeFonts"
	themeAccentKey  = "themeAccent"
)

// Palettes are the looks the owner can pick from. The first is the default.
func Palettes() []Palette {
	return []Palette{
		{Key: "fresh", Label: "Fresh", Blurb: "Clean white with a deep green accent.", Scheme: "light",
			Ground: "#ffffff", Surface: "#f6f8f7", Ink: "#16201d", Muted: "#566762", Rule: "#dce3e0", Accent: "#0e6b59"},
		{Key: "warm", Label: "Warm", Blurb: "Cream and clay. Bakeries, makers, anything handmade.", Scheme: "light",
			Ground: "#fffaf3", Surface: "#f7efe4", Ink: "#2a2119", Muted: "#6b5d4f", Rule: "#e6dccd", Accent: "#b4531e"},
		{Key: "cool", Label: "Cool", Blurb: "Soft grey-blue. Calm and professional.", Scheme: "light",
			Ground: "#f7f9fc", Surface: "#eef2f8", Ink: "#121a2b", Muted: "#55627a", Rule: "#d9e0ec", Accent: "#2457c5"},
		{Key: "bold", Label: "Bold", Blurb: "Black on white with one loud colour.", Scheme: "light",
			Ground: "#ffffff", Surface: "#f4f4f6", Ink: "#111111", Muted: "#555555", Rule: "#e2e2e6", Accent: "#d6244a"},
		{Key: "night", Label: "Night", Blurb: "Dark green-black. Bars, studios, evenings.", Scheme: "dark",
			Ground: "#0e1413", Surface: "#151e1c", Ink: "#e6edea", Muted: "#97a8a1", Rule: "#24302d", Accent: "#4cbba0"},
		{Key: "slate", Label: "Slate", Blurb: "Charcoal with a gold accent.", Scheme: "dark",
			Ground: "#14161b", Surface: "#1c1f26", Ink: "#e9ebf0", Muted: "#9aa1ad", Rule: "#2a2e38", Accent: "#f2b84b"},
	}
}

// FontPairs are the type pairings the owner can pick from. The first is the
// default and loads nothing from the network.
func FontPairs() []FontPair {
	return []FontPair{
		{Key: "clean", Label: "Clean", Blurb: "Your visitor's own system font. Fast, familiar, no downloads.",
			Display: `ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif`,
			Body:    `ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif`},
		{Key: "classic", Label: "Classic", Blurb: "Serif headings, easy sans text. Established and trustworthy.",
			Display: `"Playfair Display", Georgia, "Times New Roman", serif`,
			Body:    `"Source Sans 3", ui-sans-serif, system-ui, sans-serif`,
			Google:  []string{"Playfair+Display:wght@600;700", "Source+Sans+3:wght@400;600"}},
		{Key: "editorial", Label: "Editorial", Blurb: "Bookish and warm. Writers, studios, considered brands.",
			Display: `"Newsreader", Georgia, serif`,
			Body:    `"IBM Plex Sans", ui-sans-serif, system-ui, sans-serif`,
			Google:  []string{"Newsreader:wght@500;600", "IBM+Plex+Sans:wght@400;500;600"}},
		{Key: "modern", Label: "Modern", Blurb: "Geometric and confident. Tech, design, anything new.",
			Display: `"Manrope", ui-sans-serif, system-ui, sans-serif`,
			Body:    `"Manrope", ui-sans-serif, system-ui, sans-serif`,
			Google:  []string{"Manrope:wght@400;500;700"}},
		{Key: "friendly", Label: "Friendly", Blurb: "Rounded and open. Schools, community, family businesses.",
			Display: `"Nunito", ui-sans-serif, system-ui, sans-serif`,
			Body:    `"Nunito", ui-sans-serif, system-ui, sans-serif`,
			Google:  []string{"Nunito:wght@400;600;800"}},
	}
}

// PaletteByKey returns the named palette, defaulting to the first.
func PaletteByKey(key string) Palette {
	key = strings.ToLower(strings.TrimSpace(key))
	all := Palettes()
	for _, palette := range all {
		if palette.Key == key {
			return palette
		}
	}
	return all[0]
}

// FontPairByKey returns the named pairing, defaulting to the first.
func FontPairByKey(key string) FontPair {
	key = strings.ToLower(strings.TrimSpace(key))
	all := FontPairs()
	for _, pair := range all {
		if pair.Key == key {
			return pair
		}
	}
	return all[0]
}

var hexColour = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// NormalizeAccent accepts "#abc" or "#aabbcc" and nothing else, so a value
// pasted into the colour field can never reach a stylesheet unescaped.
func NormalizeAccent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if !hexColour.MatchString(value) {
		return ""
	}
	return value
}

// DefaultTheme is what a site looks like before the owner touches the Look.
func DefaultTheme() Theme {
	return Theme{Palette: Palettes()[0], Fonts: FontPairs()[0]}
}

// ThemeFromSettings reads the Look off the site's settings metadata.
func ThemeFromSettings(settings cmsstore.SiteSettings) Theme {
	return Theme{
		Palette: PaletteByKey(settings.Metadata[themePaletteKey]),
		Fonts:   FontPairByKey(settings.Metadata[themeFontsKey]),
		Accent:  NormalizeAccent(settings.Metadata[themeAccentKey]),
	}
}

// theme resolves the running site's Look.
func (h *Host) theme() Theme {
	return ThemeFromSettings(h.settings())
}

// EffectiveAccent is the accent actually in use.
func (t Theme) EffectiveAccent() string {
	if t.Accent != "" {
		return t.Accent
	}
	return t.Palette.Accent
}

// GoogleFontsURL is the stylesheet that loads this theme's web fonts, or empty
// when the pairing is system-only.
func (t Theme) GoogleFontsURL() string {
	if len(t.Fonts.Google) == 0 {
		return ""
	}
	parts := make([]string, 0, len(t.Fonts.Google))
	for _, family := range t.Fonts.Google {
		parts = append(parts, "family="+family)
	}
	return "https://fonts.googleapis.com/css2?" + strings.Join(parts, "&") + "&display=swap"
}

// CSS renders the theme as custom-property overrides scoped to the public
// site's root class. Scoping to the class rather than :root is what lets the
// editor show the owner their real theme on the canvas while the editor's own
// chrome around it keeps its neutral look.
func (t Theme) CSS() string {
	p := t.Palette
	var b strings.Builder
	b.WriteString(".gosx-site--public{")
	b.WriteString("color-scheme:" + p.Scheme + ";")
	b.WriteString("--site-ground:" + p.Ground + ";")
	b.WriteString("--site-surface:" + p.Surface + ";")
	b.WriteString("--site-ink:" + p.Ink + ";")
	b.WriteString("--site-muted:" + p.Muted + ";")
	b.WriteString("--site-rule:" + p.Rule + ";")
	b.WriteString("--site-accent:" + t.EffectiveAccent() + ";")
	b.WriteString("--site-font-display:" + t.Fonts.Display + ";")
	b.WriteString("--site-font-body:" + t.Fonts.Body + ";")
	b.WriteString("}")
	return b.String()
}

// RenderThemeHead emits the font link (when any) and the theme style. It must
// come after the base stylesheet so its :root-equivalent values win.
func RenderThemeHead(t Theme) gosx.Node {
	nodes := make([]gosx.Node, 0, 3)
	if fonts := t.GoogleFontsURL(); fonts != "" {
		nodes = append(nodes,
			gosx.El("link", gosx.Attrs(gosx.Attr("rel", "preconnect"), gosx.Attr("href", "https://fonts.gstatic.com"), gosx.Attr("crossorigin", "anonymous"))),
			gosx.El("link", gosx.Attrs(gosx.Attr("rel", "stylesheet"), gosx.Attr("href", fonts))),
		)
	}
	nodes = append(nodes, gosx.El("style", gosx.Attrs(gosx.Attr("data-site-theme", "true")), gosx.RawHTML(t.CSS())))
	return gosx.Fragment(nodes...)
}

// themeView is what the editor's Look controls and JS need to know.
type themeView struct {
	PaletteKey string
	FontsKey   string
	Accent     string
}

func (t Theme) view() themeView {
	return themeView{PaletteKey: t.Palette.Key, FontsKey: t.Fonts.Key, Accent: t.EffectiveAccent()}
}

// SaveTheme writes the Look to the site's settings, preserving every other
// setting and metadata key, and publishes it so visitors see it at once.
func (h *Host) SaveTheme(paletteKey, fontsKey, accent string) (Theme, error) {
	current := h.settings()
	metadata := cmsstore.Metadata{}
	for key, value := range current.Metadata {
		metadata[key] = value
	}
	palette := PaletteByKey(paletteKey)
	fonts := FontPairByKey(fontsKey)
	metadata[themePaletteKey] = palette.Key
	metadata[themeFontsKey] = fonts.Key
	if normalized := NormalizeAccent(accent); normalized != "" && normalized != palette.Accent {
		metadata[themeAccentKey] = normalized
	} else {
		delete(metadata, themeAccentKey)
	}

	input := cmsstore.SiteSettingsInput{
		Title:       current.Title,
		Description: current.Description,
		BaseURL:     current.BaseURL,
		Locale:      firstNonEmpty(current.Locale, "en"),
		Metadata:    metadata,
	}
	if _, err := h.store.SaveSiteSettings(input); err != nil {
		return Theme{}, err
	}
	if _, _, err := h.store.PublishSiteSettings(); err != nil {
		return Theme{}, err
	}
	return h.theme(), nil
}

// fontsPreviewURL is the Google Fonts stylesheet that loads every pairing at
// once, so the Look picker can show each option in its own face.
func fontsPreviewURL() string {
	parts := make([]string, 0, 8)
	for _, pair := range FontPairs() {
		for _, family := range pair.Google {
			parts = append(parts, "family="+family)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "https://fonts.googleapis.com/css2?" + strings.Join(parts, "&") + "&display=swap"
}
