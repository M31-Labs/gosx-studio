package canvas

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func decodeThumbnailPNG(t *testing.T, dataURI string) image.Image {
	t.Helper()
	raw := decodeThumbnailPNGRaw(t, dataURI)
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode thumbnail PNG: %v", err)
	}
	return img
}

func decodeThumbnailPNGRaw(t *testing.T, dataURI string) []byte {
	t.Helper()
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(dataURI, prefix) {
		t.Fatalf("data URI missing %q prefix: %q", prefix, dataURI)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURI, prefix))
	if err != nil {
		t.Fatalf("base64 decode data URI: %v", err)
	}
	if len(raw) < 8 || !bytes.Equal(raw[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatalf("decoded thumbnail is not a PNG payload: % x", raw[:minInt(len(raw), 8)])
	}
	return raw
}

func TestPageThumbnailDataURIHeroVariantRasterPNG(t *testing.T) {
	in := PageThumbnailInput{
		Title: "Home",
		Route: "/",
		Hero: &PageThumbnailHero{
			Headline: "Ceramic pieces with evidence of the <hand>",
			Subhead:  "Functional vessels & small sculptures",
			CTA:      "Shop available work",
		},
		Theme:  PageThumbnailTheme{Background: "#f6f0e8", Ink: "#2a241e", Accent: "#b86a4a"},
		Width:  200,
		Height: 72,
	}
	uri := PageThumbnailDataURI(in)
	raw := decodeThumbnailPNGRaw(t, uri)
	img := decodeThumbnailPNG(t, uri)

	if got := img.Bounds().Dx(); got != 200 {
		t.Fatalf("PNG width = %d, want 200", got)
	}
	if got := img.Bounds().Dy(); got != 72 {
		t.Fatalf("PNG height = %d, want 72", got)
	}
	assertNearRGBA(t, img.At(5, 5), color.RGBA{R: 0xb8, G: 0x6a, B: 0x4a, A: 0xff}, "accent header pixel")
	assertNearRGBA(t, img.At(5, 20), color.RGBA{R: 0xf6, G: 0xf0, B: 0xe8, A: 0xff}, "background pixel")
	assertNonUniform(t, img)
	if bytes.Contains(raw, []byte("Ceramic pieces")) || bytes.Contains(raw, []byte("<hand>")) {
		t.Fatalf("raster thumbnail must not expose raw strings in PNG bytes")
	}
}

func TestPageThumbnailDataURIHeroDefaultIncludesSubheadAndCTA(t *testing.T) {
	base := PageThumbnailInput{
		Title: "Home",
		Route: "/",
		Hero: &PageThumbnailHero{
			Headline: "Stoneware dinnerware",
			Subhead:  "Small batch table pieces",
			CTA:      "Shop work",
		},
		Theme: PageThumbnailTheme{Background: "#f6f0e8", Ink: "#2a241e", Accent: "#b86a4a"},
	}
	baseURI := PageThumbnailDataURI(base)
	baseImg := decodeThumbnailPNG(t, baseURI)

	changedSubhead := base
	changedSubhead.Hero = &PageThumbnailHero{
		Headline: base.Hero.Headline,
		Subhead:  "Wheel thrown serving pieces",
		CTA:      base.Hero.CTA,
	}
	subheadURI := PageThumbnailDataURI(changedSubhead)
	if subheadURI == baseURI {
		t.Fatalf("changing default hero subhead should alter thumbnail URI")
	}
	if !imagesDiffer(baseImg, decodeThumbnailPNG(t, subheadURI)) {
		t.Fatalf("changing default hero subhead should alter thumbnail pixels")
	}

	changedCTA := base
	changedCTA.Hero = &PageThumbnailHero{
		Headline: base.Hero.Headline,
		Subhead:  base.Hero.Subhead,
		CTA:      "View collection",
	}
	ctaURI := PageThumbnailDataURI(changedCTA)
	if ctaURI == baseURI {
		t.Fatalf("changing default hero CTA should alter thumbnail URI")
	}
	if !imagesDiffer(baseImg, decodeThumbnailPNG(t, ctaURI)) {
		t.Fatalf("changing default hero CTA should alter thumbnail pixels")
	}
}

func TestPageThumbnailDataURIPlainVariantRasterPNG(t *testing.T) {
	in := PageThumbnailInput{
		Title:  "Shop",
		Route:  "/shop",
		Theme:  PageThumbnailTheme{Background: "#fdfaf6", Ink: "#4c4036", Accent: "#965036"},
		Width:  200,
		Height: 72,
	}
	uri := PageThumbnailDataURI(in)
	img := decodeThumbnailPNG(t, uri)

	assertNearRGBA(t, img.At(5, 5), color.RGBA{R: 0x96, G: 0x50, B: 0x36, A: 0xff}, "accent header pixel")
	assertNearRGBA(t, img.At(5, 20), color.RGBA{R: 0xfd, G: 0xfa, B: 0xf6, A: 0xff}, "background pixel")
	assertNonUniform(t, img)
}

func TestPageThumbnailCacheKeyStructuredDelimiterCollisionRegression(t *testing.T) {
	plain := normalizePageThumbnailInput(PageThumbnailInput{
		Title: "Title|with|pipes",
		Route: "/route|hero|Headline|Subhead|CTA",
	})
	hero := normalizePageThumbnailInput(PageThumbnailInput{
		Title: "Title|with|pipes",
		Route: "/route",
		Hero: &PageThumbnailHero{
			Headline: "Headline",
			Subhead:  "Subhead",
			CTA:      "CTA",
		},
	})
	if gotPlain, gotHero := pageThumbnailCacheKey(plain), pageThumbnailCacheKey(hero); gotPlain == gotHero {
		t.Fatalf("structured cache keys collided:\nplain: %s\nhero: %s", gotPlain, gotHero)
	}

	plainURI := PageThumbnailDataURI(plain)
	heroURI := PageThumbnailDataURI(hero)
	if plainURI == heroURI {
		t.Fatalf("delimiter-like plain content should not share cached thumbnail with hero content")
	}
}

func TestPageThumbnailDataURIDefaultsAndDeterministicChanges(t *testing.T) {
	base := PageThumbnailInput{
		Title:  `A & B "quote"`,
		Route:  "/a?x=1&y=2",
		Width:  -1,
		Height: 0,
	}
	uri1 := PageThumbnailDataURI(base)
	uri2 := PageThumbnailDataURI(base)
	if uri1 != uri2 {
		t.Fatalf("same thumbnail input should produce deterministic cached URI")
	}

	img := decodeThumbnailPNG(t, uri1)
	if got := img.Bounds().Dx(); got != pageThumbnailDefaultWidth {
		t.Fatalf("default width = %d, want %d", got, pageThumbnailDefaultWidth)
	}
	if got := img.Bounds().Dy(); got != pageThumbnailDefaultHeight {
		t.Fatalf("default height = %d, want %d", got, pageThumbnailDefaultHeight)
	}
	assertNearRGBA(t, img.At(5, 5), color.RGBA{R: 0xb8, G: 0x6a, B: 0x4a, A: 0xff}, "default accent header pixel")

	changedContent := base
	changedContent.Title = "Different page"
	if got := PageThumbnailDataURI(changedContent); got == uri1 {
		t.Fatalf("content change should alter thumbnail URI")
	}

	changedTheme := base
	changedTheme.Theme.Accent = "#225588"
	if got := PageThumbnailDataURI(changedTheme); got == uri1 {
		t.Fatalf("theme change should alter thumbnail URI")
	}
}

func TestPageThumbnailDataURINonASCIIContentSensitive(t *testing.T) {
	base := PageThumbnailInput{
		Title: "Café “Minka”",
		Route: "/céramique",
		Hero: &PageThumbnailHero{
			Headline: "Café “Minka”",
			Subhead:  "Pièces émaillées",
			CTA:      "Réserver",
		},
		Theme: PageThumbnailTheme{Background: "#f6f0e8", Ink: "#2a241e", Accent: "#b86a4a"},
	}
	uri1 := PageThumbnailDataURI(base)
	uri2 := PageThumbnailDataURI(base)
	if uri1 != uri2 {
		t.Fatalf("non-ASCII thumbnail generation should be deterministic")
	}
	img1 := decodeThumbnailPNG(t, uri1)

	changed := base
	changed.Hero = &PageThumbnailHero{
		Headline: "Cafe \"Minka\"",
		Subhead:  "Pieces emaillees",
		CTA:      "Reserver",
	}
	changedURI := PageThumbnailDataURI(changed)
	if changedURI == uri1 {
		t.Fatalf("non-ASCII content change should alter thumbnail URI")
	}
	if !imagesDiffer(img1, decodeThumbnailPNG(t, changedURI)) {
		t.Fatalf("non-ASCII content change should alter thumbnail pixels")
	}
}

func assertNonUniform(t *testing.T, img image.Image) {
	t.Helper()
	b := img.Bounds()
	first := color.RGBAModel.Convert(img.At(b.Min.X, b.Min.Y)).(color.RGBA)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if color.RGBAModel.Convert(img.At(x, y)).(color.RGBA) != first {
				return
			}
		}
	}
	t.Fatalf("thumbnail image is uniform color %v", first)
}

func imagesDiffer(a, b image.Image) bool {
	if !a.Bounds().Eq(b.Bounds()) {
		return true
	}
	bounds := a.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if color.RGBAModel.Convert(a.At(x, y)).(color.RGBA) != color.RGBAModel.Convert(b.At(x, y)).(color.RGBA) {
				return true
			}
		}
	}
	return false
}

func assertNearRGBA(t *testing.T, got color.Color, want color.RGBA, label string) {
	t.Helper()
	rgba := color.RGBAModel.Convert(got).(color.RGBA)
	const tolerance = 2
	if absInt(int(rgba.R)-int(want.R)) > tolerance ||
		absInt(int(rgba.G)-int(want.G)) > tolerance ||
		absInt(int(rgba.B)-int(want.B)) > tolerance ||
		absInt(int(rgba.A)-int(want.A)) > tolerance {
		t.Fatalf("%s = %#v, want near %#v", label, rgba, want)
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
