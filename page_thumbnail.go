package studio

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// PageThumbnailTheme carries the three literal hex colors a thumbnail paints
// with. They come from the active palette (internal/theme) resolved to hex
// upstream; blank fields fall back to neutral defaults so a thumbnail always
// renders.
type PageThumbnailTheme struct {
	Background string // page card surface color (e.g. "#f6f0e8")
	Ink        string // primary text color (e.g. "#2a241e")
	Accent     string // header bar + CTA color (e.g. "#b86a4a")
}

// PageThumbnailHero is the optional hero content for the home page thumbnail.
// When present the generator renders the headline/subhead/CTA snippet instead of
// the plain title+route variant.
type PageThumbnailHero struct {
	Headline string
	Subhead  string
	CTA      string
}

// PageThumbnailInput describes one page's thumbnail. Title/Route identify the
// page (and are the plain variant's content); Hero, when non-nil, switches to
// the hero variant. Theme supplies the literal colors. Width/Height size the PNG
// to match the page card's placement bounds so the painter can paint it as a
// full-card overlay; non-positive values fall back to a sensible default.
type PageThumbnailInput struct {
	Title  string
	Route  string
	Hero   *PageThumbnailHero
	Theme  PageThumbnailTheme
	Width  int
	Height int
}

const (
	pageThumbnailDefaultWidth  = 200
	pageThumbnailDefaultHeight = 72
	pageThumbnailDataPrefix    = "data:image/png;base64,"

	// Neutral fallbacks mirroring the theme package's "clay" default so a blank
	// theme still produces a readable, themed-looking card.
	pageThumbnailDefaultBackground = "#f6f0e8"
	pageThumbnailDefaultInk        = "#2a241e"
	pageThumbnailDefaultAccent     = "#b86a4a"
)

var pageThumbnailCache sync.Map

type pageThumbnailFaces struct {
	Headline           font.Face
	Body               font.Face
	CTA                font.Face
	HeadlineLineHeight int
	BodyLineHeight     int
	CTALineHeight      int
}

func newPageThumbnailFaces() (pageThumbnailFaces, error) {
	ttf, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return pageThumbnailFaces{}, err
	}
	headline, err := opentype.NewFace(ttf, &opentype.FaceOptions{Size: 9.5, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return pageThumbnailFaces{}, err
	}
	body, err := opentype.NewFace(ttf, &opentype.FaceOptions{Size: 7.2, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		_ = headline.Close()
		return pageThumbnailFaces{}, err
	}
	cta, err := opentype.NewFace(ttf, &opentype.FaceOptions{Size: 7.2, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		_ = headline.Close()
		_ = body.Close()
		return pageThumbnailFaces{}, err
	}
	return pageThumbnailFaces{
		Headline:           headline,
		Body:               body,
		CTA:                cta,
		HeadlineLineHeight: maxInt(1, headline.Metrics().Height.Ceil()),
		BodyLineHeight:     maxInt(1, body.Metrics().Height.Ceil()),
		CTALineHeight:      maxInt(1, cta.Metrics().Height.Ceil()),
	}, nil
}

func (f pageThumbnailFaces) Close() {
	if f.Headline != nil {
		_ = f.Headline.Close()
	}
	if f.Body != nil {
		_ = f.Body.Close()
	}
	if f.CTA != nil {
		_ = f.CTA.Close()
	}
}

// PageThumbnailDataURI builds a deterministic server-side raster PNG thumbnail
// for one page and returns it as a "data:image/png;base64,..." URL.
//
// Two variants:
//   - hero: when in.Hero != nil, paints the headline, subhead, and a CTA block
//     in the accent color - the real home page snapshot.
//   - plain: otherwise, paints the page title and its route.
//
// Both carry an accent-colored header bar at the top so the card reads as a
// themed page at a glance. The renderer uses only Go image primitives and the
// bundled Go Regular TrueType font from x/image; it has no browser, WASM,
// network, or external asset dependency. Results are cached by normalized
// content/theme/size so repeated thumbnails avoid repeated PNG encoding.
func PageThumbnailDataURI(in PageThumbnailInput) string {
	n := normalizePageThumbnailInput(in)
	key := pageThumbnailCacheKey(n)
	if cached, ok := pageThumbnailCache.Load(key); ok {
		return cached.(string)
	}

	img := image.NewRGBA(image.Rect(0, 0, n.Width, n.Height))
	bg := parseThumbnailColor(n.Theme.Background, pageThumbnailDefaultBackground)
	ink := parseThumbnailColor(n.Theme.Ink, pageThumbnailDefaultInk)
	accent := parseThumbnailColor(n.Theme.Accent, pageThumbnailDefaultAccent)

	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	fillRect(img, image.Rect(0, 0, n.Width, minInt(8, n.Height)), accent)

	contentTop := minInt(8, n.Height)
	content := image.Rect(10, contentTop+8, maxInt(10, n.Width-10), maxInt(contentTop+8, n.Height-8))
	faces, err := newPageThumbnailFaces()
	if err != nil {
		return ""
	}
	defer faces.Close()
	if n.Hero != nil {
		drawPageThumbnailHero(img, content, *n.Hero, ink, accent, faces)
	} else {
		drawPageThumbnailPlain(img, content, n.Title, n.Route, ink, faces)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	uri := pageThumbnailDataPrefix + base64.StdEncoding.EncodeToString(buf.Bytes())
	actual, _ := pageThumbnailCache.LoadOrStore(key, uri)
	return actual.(string)
}

func normalizePageThumbnailInput(in PageThumbnailInput) PageThumbnailInput {
	out := in
	if out.Width <= 0 {
		out.Width = pageThumbnailDefaultWidth
	}
	if out.Height <= 0 {
		out.Height = pageThumbnailDefaultHeight
	}
	out.Title = strings.TrimSpace(out.Title)
	out.Route = strings.TrimSpace(out.Route)
	out.Theme.Background = firstNonBlank(out.Theme.Background, pageThumbnailDefaultBackground)
	out.Theme.Ink = firstNonBlank(out.Theme.Ink, pageThumbnailDefaultInk)
	out.Theme.Accent = firstNonBlank(out.Theme.Accent, pageThumbnailDefaultAccent)
	if out.Hero != nil {
		hero := &PageThumbnailHero{
			Headline: strings.TrimSpace(out.Hero.Headline),
			Subhead:  strings.TrimSpace(out.Hero.Subhead),
			CTA:      strings.TrimSpace(out.Hero.CTA),
		}
		out.Hero = hero
	}
	return out
}

func pageThumbnailCacheKey(in PageThumbnailInput) string {
	payload := struct {
		Version int                `json:"version"`
		Width   int                `json:"width"`
		Height  int                `json:"height"`
		Title   string             `json:"title"`
		Route   string             `json:"route"`
		Theme   PageThumbnailTheme `json:"theme"`
		Hero    *PageThumbnailHero `json:"hero,omitempty"`
	}{
		Version: 2,
		Width:   in.Width,
		Height:  in.Height,
		Title:   in.Title,
		Route:   in.Route,
		Theme:   in.Theme,
		Hero:    in.Hero,
	}
	key, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(key)
}

func drawPageThumbnailHero(img *image.RGBA, bounds image.Rectangle, hero PageThumbnailHero, ink, accent color.RGBA, faces pageThumbnailFaces) {
	ctaText := ""
	ctaH := 0
	ctaY := bounds.Max.Y
	if hero.CTA != "" && bounds.Dy() >= 22 {
		ctaText = truncateText(hero.CTA, maxInt(0, bounds.Dx()-18), faces.CTA)
		if ctaText != "" {
			ctaH = minInt(15, maxInt(12, faces.CTALineHeight+4))
			ctaY = bounds.Max.Y - ctaH
			ctaW := minInt(bounds.Dx(), font.MeasureString(faces.CTA, ctaText).Ceil()+16)
			fillRect(img, image.Rect(bounds.Min.X, ctaY, bounds.Min.X+ctaW, ctaY+ctaH), accent)
			drawText(img, ctaText, bounds.Min.X+8, ctaY+maxInt(1, (ctaH-faces.CTALineHeight)/2), color.RGBA{R: 255, G: 255, B: 255, A: 255}, faces.CTA)
		}
	}

	textBottom := bounds.Max.Y
	if ctaText != "" {
		textBottom = ctaY - 3
	}
	y := bounds.Min.Y
	if hero.Headline != "" {
		y = drawClippedText(img, hero.Headline, bounds.Min.X, y, bounds.Max.X-bounds.Min.X, ink, faces.Headline, faces.HeadlineLineHeight)
		y += 1
	}
	if hero.Subhead != "" && y+faces.BodyLineHeight <= textBottom {
		drawClippedText(img, hero.Subhead, bounds.Min.X, y, bounds.Max.X-bounds.Min.X, mutedColor(ink, 190), faces.Body, faces.BodyLineHeight)
	}
}

func drawPageThumbnailPlain(img *image.RGBA, bounds image.Rectangle, title, route string, ink color.RGBA, faces pageThumbnailFaces) {
	y := bounds.Min.Y + 2
	if title != "" {
		y = drawClippedText(img, title, bounds.Min.X, y, bounds.Max.X-bounds.Min.X, ink, faces.Headline, faces.HeadlineLineHeight)
		y += 3
	}
	if route != "" && y+faces.BodyLineHeight <= bounds.Max.Y {
		drawClippedText(img, route, bounds.Min.X, y, bounds.Max.X-bounds.Min.X, mutedColor(ink, 175), faces.Body, faces.BodyLineHeight)
	}
}

func drawClippedText(img *image.RGBA, text string, x, topY, maxWidth int, c color.RGBA, face font.Face, lineHeight int) int {
	if maxWidth <= 0 {
		return topY
	}
	drawText(img, truncateText(text, maxWidth, face), x, topY, c, face)
	return topY + lineHeight
}

func drawText(img *image.RGBA, text string, x, topY int, c color.RGBA, face font.Face) {
	if text == "" {
		return
	}
	baselineY := topY + face.Metrics().Ascent.Ceil()
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, baselineY),
	}
	d.DrawString(text)
}

func truncateText(text string, maxWidth int, face font.Face) string {
	text = strings.TrimSpace(text)
	if text == "" || maxWidth <= 0 {
		return ""
	}
	if font.MeasureString(face, text).Ceil() <= maxWidth {
		return text
	}
	runes := []rune(text)
	const ellipsis = "..."
	for len(runes) > 0 {
		candidate := string(runes) + ellipsis
		if font.MeasureString(face, candidate).Ceil() <= maxWidth {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	if font.MeasureString(face, ellipsis).Ceil() <= maxWidth {
		return ellipsis
	}
	return ""
}

func fillRect(img *image.RGBA, r image.Rectangle, c color.RGBA) {
	r = r.Intersect(img.Bounds())
	if r.Empty() {
		return
	}
	draw.Draw(img, r, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func parseThumbnailColor(value, fallback string) color.RGBA {
	v := strings.TrimSpace(value)
	if v == "" {
		v = fallback
	}
	if c, ok := parseHexColor(v); ok {
		return c
	}
	c, _ := parseHexColor(fallback)
	return c
}

func parseHexColor(value string) (color.RGBA, bool) {
	v := strings.TrimPrefix(strings.TrimSpace(value), "#")
	switch len(v) {
	case 3:
		r, okR := parseHexNibble(v[0])
		g, okG := parseHexNibble(v[1])
		b, okB := parseHexNibble(v[2])
		if okR && okG && okB {
			return color.RGBA{R: r * 17, G: g * 17, B: b * 17, A: 255}, true
		}
	case 6:
		r, okR := parseHexByte(v[0:2])
		g, okG := parseHexByte(v[2:4])
		b, okB := parseHexByte(v[4:6])
		if okR && okG && okB {
			return color.RGBA{R: r, G: g, B: b, A: 255}, true
		}
	}
	return color.RGBA{}, false
}

func parseHexByte(value string) (uint8, bool) {
	n, err := strconv.ParseUint(value, 16, 8)
	if err != nil {
		return 0, false
	}
	return uint8(n), true
}

func parseHexNibble(value byte) (uint8, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func mutedColor(c color.RGBA, alpha uint8) color.RGBA {
	bg := parseThumbnailColor(pageThumbnailDefaultBackground, pageThumbnailDefaultBackground)
	a := int(alpha)
	return color.RGBA{
		R: uint8((int(c.R)*a + int(bg.R)*(255-a)) / 255),
		G: uint8((int(c.G)*a + int(bg.G)*(255-a)) / 255),
		B: uint8((int(c.B)*a + int(bg.B)*(255-a)) / 255),
		A: 255,
	}
}

// firstNonBlank returns the first non-empty (trimmed) value, or "".
func firstNonBlank(values ...string) string {
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
