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
	Palette  Palette
	Fonts    FontPair
	Accent   string // hex, overrides Palette.Accent when set
	Buttons  string // ButtonShape key
	Spacing  string // SpacingScale key
	Headings string // HeadingScale key
	Width    string // PageWidth key
	// Ground and Ink are the owner's own colours when Palette is the custom
	// one; the rest of the palette is mixed from them.
	Ground, Ink string
	// FontHead and FontBody are the owner's own Google Fonts family names
	// when Fonts is the custom pairing.
	FontHead, FontBody string
	// CustomCSS is the owner's own stylesheet, already sanitised.
	CustomCSS string
	// Motion is how much the site's backdrops move: full, calm, or off.
	Motion string
}

var siteMotions = map[string]bool{"full": true, "calm": true, "off": true}

const (
	themePaletteKey  = "themePalette"
	themeFontsKey    = "themeFonts"
	themeAccentKey   = "themeAccent"
	themeButtonsKey  = "themeButtons"
	themeSpacingKey  = "themeSpacing"
	themeHeadingsKey = "themeHeadings"
	themeMotionKey   = "themeMotion"
	themeWidthKey    = "themeWidth"
	themeGroundKey   = "themeGround"
	themeInkKey      = "themeInk"
	customCSSKey     = "customCss"
	customPaletteKey = "custom"
	customFontsKey   = "custom"
	themeFontHeadKey = "themeFontHead"
	themeFontBodyKey = "themeFontBody"
	customCSSMax     = 20 << 10
)

// fontName keeps a Google Fonts family name to letters, digits, and
// spaces, so it can go into a stylesheet and a URL unescaped.
func fontName(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ':
			b.WriteRune(r)
		}
	}
	name := strings.Join(strings.Fields(b.String()), " ")
	if len(name) > 40 {
		name = name[:40]
	}
	return name
}

// customFonts builds a pairing from two Google Fonts family names. Either
// may be empty, in which case the body font stands in.
func customFonts(head, body string) FontPair {
	head, body = fontName(head), fontName(body)
	if head == "" && body == "" {
		return FontPairs()[0]
	}
	if head == "" {
		head = body
	}
	if body == "" {
		body = head
	}
	google := []string{strings.ReplaceAll(head, " ", "+") + ":wght@400;600;700"}
	if body != head {
		google = append(google, strings.ReplaceAll(body, " ", "+")+":wght@400;600")
	}
	return FontPair{
		Key: customFontsKey, Label: "Custom", Blurb: "Fonts you named.",
		Display: `"` + head + `", ui-sans-serif, system-ui, sans-serif`,
		Body:    `"` + body + `", ui-sans-serif, system-ui, sans-serif`,
		Google:  google,
	}
}

// HeadingScale is how big headings are next to the text.
type HeadingScale struct {
	Key, Label, Scale string
}

// HeadingScales are the sizes the owner can pick. The second is the default.
func HeadingScales() []HeadingScale {
	return []HeadingScale{
		{Key: "quiet", Label: "Quiet", Scale: "0.88"},
		{Key: "regular", Label: "Regular", Scale: "1"},
		{Key: "big", Label: "Big", Scale: "1.2"},
	}
}

func HeadingScaleByKey(key string) HeadingScale {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, scale := range HeadingScales() {
		if scale.Key == key {
			return scale
		}
	}
	return HeadingScales()[1]
}

// PageWidth is how wide a column of text runs.
type PageWidth struct {
	Key, Label, Measure string
}

// PageWidths are the widths the owner can pick. The second is the default.
func PageWidths() []PageWidth {
	return []PageWidth{
		{Key: "narrow", Label: "Narrow", Measure: "56ch"},
		{Key: "regular", Label: "Regular", Measure: "68ch"},
		{Key: "wide", Label: "Wide", Measure: "84ch"},
	}
}

func PageWidthByKey(key string) PageWidth {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, width := range PageWidths() {
		if width.Key == key {
			return width
		}
	}
	return PageWidths()[1]
}

// customPalette mixes a whole palette from two colours the owner chose.
// The mixes are left to the browser, which is exact and needs no maths.
func customPalette(ground, ink, accent string) Palette {
	scheme := "light"
	if luminance(ground) < 0.4 {
		scheme = "dark"
	}
	if accent == "" {
		accent = ink
	}
	return Palette{
		Key: customPaletteKey, Label: "Custom", Blurb: "Your own colours.", Scheme: scheme,
		Ground:  ground,
		Surface: "color-mix(in srgb, " + ink + " 6%, " + ground + ")",
		Ink:     ink,
		Muted:   "color-mix(in srgb, " + ink + " 62%, " + ground + ")",
		Rule:    "color-mix(in srgb, " + ink + " 14%, " + ground + ")",
		Accent:  accent,
	}
}

// luminance is a rough brightness of a hex colour, 0 dark to 1 light.
func luminance(hex string) float64 {
	hex = strings.TrimPrefix(NormalizeAccent(hex), "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return 1
	}
	channel := func(from int) float64 {
		var value int
		for _, r := range hex[from : from+2] {
			value *= 16
			switch {
			case r >= '0' && r <= '9':
				value += int(r - '0')
			case r >= 'a' && r <= 'f':
				value += int(r-'a') + 10
			}
		}
		return float64(value) / 255
	}
	return 0.2126*channel(0) + 0.7152*channel(2) + 0.0722*channel(4)
}

// sanitizeCustomCSS keeps the owner's stylesheet from breaking out of its
// style element or pulling in other people's code. It is not a parser;
// it removes the few things that matter and caps the size.
func sanitizeCustomCSS(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > customCSSMax {
		raw = raw[:customCSSMax]
	}
	lower := strings.ToLower(raw)
	for _, banned := range []string{"</style", "<script", "@import", "expression(", "javascript:", "behavior:", "-moz-binding"} {
		for {
			at := strings.Index(lower, banned)
			if at < 0 {
				break
			}
			raw = raw[:at] + raw[at+len(banned):]
			lower = lower[:at] + lower[at+len(banned):]
		}
	}
	return strings.TrimSpace(raw)
}

// ButtonShape is how corners on buttons, pictures, and fields look.
type ButtonShape struct {
	Key    string
	Label  string
	Radius string // CSS length
}

// ButtonShapes are the corner styles. The first is the default.
func ButtonShapes() []ButtonShape {
	return []ButtonShape{
		{Key: "square", Label: "Square", Radius: "2px"},
		{Key: "rounded", Label: "Rounded", Radius: "8px"},
		{Key: "pill", Label: "Pill", Radius: "999px"},
	}
}

func ButtonShapeByKey(key string) ButtonShape {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, shape := range ButtonShapes() {
		if shape.Key == key {
			return shape
		}
	}
	return ButtonShapes()[0]
}

// SpacingScale is how much air the site has between things.
type SpacingScale struct {
	Key   string
	Label string
	Scale string // multiplier as CSS number
}

// SpacingScales are the spacing choices. The second is the default.
func SpacingScales() []SpacingScale {
	return []SpacingScale{
		{Key: "compact", Label: "Compact", Scale: "0.8"},
		{Key: "regular", Label: "Regular", Scale: "1"},
		{Key: "airy", Label: "Airy", Scale: "1.35"},
	}
}

func SpacingScaleByKey(key string) SpacingScale {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, scale := range SpacingScales() {
		if scale.Key == key {
			return scale
		}
	}
	return SpacingScales()[1]
}

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
	m := settings.Metadata
	theme := Theme{
		Palette:   PaletteByKey(m[themePaletteKey]),
		Fonts:     FontPairByKey(m[themeFontsKey]),
		Accent:    NormalizeAccent(m[themeAccentKey]),
		Buttons:   ButtonShapeByKey(m[themeButtonsKey]).Key,
		Spacing:   SpacingScaleByKey(m[themeSpacingKey]).Key,
		Headings:  HeadingScaleByKey(m[themeHeadingsKey]).Key,
		Width:     PageWidthByKey(m[themeWidthKey]).Key,
		Ground:    NormalizeAccent(m[themeGroundKey]),
		Ink:       NormalizeAccent(m[themeInkKey]),
		FontHead:  fontName(m[themeFontHeadKey]),
		FontBody:  fontName(m[themeFontBodyKey]),
		CustomCSS: sanitizeCustomCSS(m[customCSSKey]),
		Motion:    normalizeChoice(m[themeMotionKey], "full", siteMotions),
	}
	if strings.EqualFold(strings.TrimSpace(m[themePaletteKey]), customPaletteKey) && theme.Ground != "" && theme.Ink != "" {
		theme.Palette = customPalette(theme.Ground, theme.Ink, theme.Accent)
	}
	if strings.EqualFold(strings.TrimSpace(m[themeFontsKey]), customFontsKey) && (theme.FontHead != "" || theme.FontBody != "") {
		theme.Fonts = customFonts(theme.FontHead, theme.FontBody)
	}
	return theme
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
	b.WriteString("--site-radius:" + ButtonShapeByKey(t.Buttons).Radius + ";")
	b.WriteString("--site-space:" + SpacingScaleByKey(t.Spacing).Scale + ";")
	b.WriteString("--site-heading-scale:" + HeadingScaleByKey(t.Headings).Scale + ";")
	b.WriteString("--site-measure:" + PageWidthByKey(t.Width).Measure + ";")
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
	if t.CustomCSS != "" {
		nodes = append(nodes, gosx.El("style", gosx.Attrs(gosx.Attr("data-site-custom", "true")), gosx.RawHTML(t.CustomCSS)))
	}
	return gosx.Fragment(nodes...)
}

// themeView is what the editor's Look controls and JS need to know.
type themeView struct {
	PaletteKey         string
	FontsKey           string
	Accent             string
	ButtonsKey         string
	SpacingKey         string
	HeadingsKey        string
	WidthKey           string
	Motion             string
	Ground, Ink        string
	FontHead, FontBody string
}

func (t Theme) view() themeView {
	return themeView{
		PaletteKey: t.Palette.Key, FontsKey: t.Fonts.Key, Accent: t.EffectiveAccent(),
		ButtonsKey: ButtonShapeByKey(t.Buttons).Key, SpacingKey: SpacingScaleByKey(t.Spacing).Key,
		HeadingsKey: HeadingScaleByKey(t.Headings).Key, WidthKey: PageWidthByKey(t.Width).Key, Motion: firstNonEmpty(t.Motion, "full"),
		Ground: firstNonEmpty(t.Ground, "#ffffff"), Ink: firstNonEmpty(t.Ink, "#1a1a1a"),
		FontHead: t.FontHead, FontBody: t.FontBody,
	}
}

// ThemeChoice is everything the Look panel can set at once.
type ThemeChoice struct {
	Palette, Fonts, Accent, Buttons, Spacing, Headings, Width, Ground, Ink string
	FontHead, FontBody                                                     string
	Motion                                                                 string
}

// SaveTheme writes the Look to the site's settings, preserving every other
// setting and metadata key, and publishes it so visitors see it at once.
func (h *Host) SaveTheme(choice ThemeChoice) (Theme, error) {
	current := h.settings()
	metadata := cmsstore.Metadata{}
	for key, value := range current.Metadata {
		metadata[key] = value
	}
	palette := PaletteByKey(choice.Palette)
	fonts := FontPairByKey(choice.Fonts)
	ground, ink := NormalizeAccent(choice.Ground), NormalizeAccent(choice.Ink)
	if strings.EqualFold(strings.TrimSpace(choice.Palette), customPaletteKey) && ground != "" && ink != "" {
		metadata[themePaletteKey] = customPaletteKey
		metadata[themeGroundKey] = ground
		metadata[themeInkKey] = ink
		palette = customPalette(ground, ink, NormalizeAccent(choice.Accent))
	} else {
		metadata[themePaletteKey] = palette.Key
		delete(metadata, themeGroundKey)
		delete(metadata, themeInkKey)
	}
	head, body := fontName(choice.FontHead), fontName(choice.FontBody)
	if strings.EqualFold(strings.TrimSpace(choice.Fonts), customFontsKey) && (head != "" || body != "") {
		metadata[themeFontsKey] = customFontsKey
		metadata[themeFontHeadKey] = head
		metadata[themeFontBodyKey] = body
	} else {
		metadata[themeFontsKey] = fonts.Key
		delete(metadata, themeFontHeadKey)
		delete(metadata, themeFontBodyKey)
	}
	metadata[themeButtonsKey] = ButtonShapeByKey(choice.Buttons).Key
	metadata[themeSpacingKey] = SpacingScaleByKey(choice.Spacing).Key
	metadata[themeHeadingsKey] = HeadingScaleByKey(choice.Headings).Key
	metadata[themeWidthKey] = PageWidthByKey(choice.Width).Key
	metadata[themeMotionKey] = normalizeChoice(choice.Motion, "full", siteMotions)
	if normalized := NormalizeAccent(choice.Accent); normalized != "" && (normalized != palette.Accent || palette.Key == customPaletteKey) {
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
