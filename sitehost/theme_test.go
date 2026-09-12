package sitehost

import (
	"net/http"
	"strings"
	"testing"
)

func TestPresetsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range Palettes() {
		if seen[p.Key] {
			t.Fatalf("duplicate palette key %q", p.Key)
		}
		seen[p.Key] = true
		for name, value := range map[string]string{"ground": p.Ground, "surface": p.Surface, "ink": p.Ink, "muted": p.Muted, "rule": p.Rule, "accent": p.Accent} {
			if NormalizeAccent(value) == "" {
				t.Fatalf("palette %q %s = %q is not a hex colour", p.Key, name, value)
			}
		}
		if p.Scheme != "light" && p.Scheme != "dark" {
			t.Fatalf("palette %q scheme = %q", p.Key, p.Scheme)
		}
	}
	seen = map[string]bool{}
	for _, f := range FontPairs() {
		if seen[f.Key] {
			t.Fatalf("duplicate font key %q", f.Key)
		}
		seen[f.Key] = true
		if f.Display == "" || f.Body == "" {
			t.Fatalf("font pair %q has an empty stack", f.Key)
		}
	}
	// The defaults must load nothing from the network.
	if DefaultTheme().GoogleFontsURL() != "" {
		t.Fatal("the default font pairing must be system-only")
	}
}

func TestNormalizeAccentRejectsAnythingButHex(t *testing.T) {
	cases := map[string]string{
		"#abc": "#abc", "#AABBCC": "#aabbcc", " #0e6b59 ": "#0e6b59",
		"red": "", "#ab": "", "#12345": "", "url(x)": "", "#aabbcc; }": "", "": "",
	}
	for in, want := range cases {
		if got := NormalizeAccent(in); got != want {
			t.Errorf("NormalizeAccent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestThemeCSSIsScopedAndComplete(t *testing.T) {
	theme := Theme{Palette: PaletteByKey("warm"), Fonts: FontPairByKey("classic"), Accent: "#123456"}
	css := theme.CSS()
	for _, want := range []string{
		".gosx-site--public{", "color-scheme:light", "--site-ground:#fffaf3", "--site-accent:#123456",
		`--site-font-display:"Playfair Display"`, "--site-font-body:",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("theme css missing %q: %s", want, css)
		}
	}
	if strings.Contains(css, ":root") {
		t.Fatal("theme css must be scoped to the public root class, not :root, so the editor chrome keeps its own look")
	}
}

func TestSaveThemeRoundTripsAndPreservesSettings(t *testing.T) {
	host, handler := newTestHost(t)

	saved, err := host.SaveTheme("night", "editorial", "#ff8800")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Palette.Key != "night" || saved.Fonts.Key != "editorial" || saved.Accent != "#ff8800" {
		t.Fatalf("saved theme = %+v", saved)
	}

	// Site title, description, and the setup marker all survive.
	settings := host.settings()
	if settings.Title != "Wildflower Bakery" {
		t.Fatalf("saving the look changed the site title to %q", settings.Title)
	}
	if !host.SetupComplete() {
		t.Fatal("saving the look wiped the setup marker")
	}

	// The public page carries it.
	body := get(t, handler, "/").Body.String()
	mustContain(t, body, `<style data-site-theme="true">`, "public head carries the theme style")
	mustContain(t, body, "--site-ground:#0e1413", "the chosen palette reaches visitors")
	mustContain(t, body, "--site-accent:#ff8800", "the custom accent reaches visitors")
	mustContain(t, body, "fonts.googleapis.com/css2?family=Newsreader", "the pairing's fonts are linked")

	// Choosing the palette's own accent drops the override.
	saved, err = host.SaveTheme("night", "clean", "#4cbba0")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Accent != "" {
		t.Fatalf("accent equal to the palette's should not be stored as an override, got %q", saved.Accent)
	}
	if strings.Contains(get(t, handler, "/").Body.String(), "fonts.googleapis.com") {
		t.Fatal("the system pairing must not link web fonts")
	}
}

func TestUnknownThemeKeysFallBackToDefaults(t *testing.T) {
	host, _ := newTestHost(t)
	saved, err := host.SaveTheme("nope", "nope", "javascript:alert(1)")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Palette.Key != Palettes()[0].Key || saved.Fonts.Key != FontPairs()[0].Key || saved.Accent != "" {
		t.Fatalf("unknown keys must fall back to defaults, got %+v", saved)
	}
}

func TestEditorCarriesTheLookPicker(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	body := get(t, handler, "/admin/edit/"+id).Body.String()

	mustContain(t, body, `data-look="true"`, "editor sidebar has the Look section")
	mustContain(t, body, `data-look-palette="warm"`, "palettes are offered")
	mustContain(t, body, `data-look-fonts="classic"`, "font pairings are offered")
	mustContain(t, body, `data-look-accent="true"`, "an accent picker is offered")
	mustContain(t, body, `data-look-presets="true"`, "presets are embedded for live preview")
	mustContain(t, body, `"fontsUrl"`, "presets carry each pairing's font stylesheet")
}

func TestThemeAPISavesAndReturnsCSS(t *testing.T) {
	host, handler := newTestHost(t)
	rec := postJSON(t, handler, "/admin/api/theme", `{"palette":"bold","fonts":"modern","accent":"#00aa00"}`)

	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `"ok":true`) {
		t.Fatalf("theme save = %d: %s", rec.Code, body)
	}
	mustContain(t, body, "--site-accent:#00aa00", "the response carries the new css for the live style tag")
	if theme := host.theme(); theme.Palette.Key != "bold" || theme.Fonts.Key != "modern" {
		t.Fatalf("theme not persisted: %+v", theme)
	}
}
