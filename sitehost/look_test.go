package sitehost

import (
	"net/url"
	"strings"
	"testing"
)

func TestLookGoesDeeper(t *testing.T) {
	host, handler := newTestHost(t)
	rec := postJSON(t, handler, "/admin/api/theme", `{"palette":"custom","fonts":"classic","accent":"#c05621","buttons":"pill","spacing":"airy","headings":"big","width":"wide","ground":"#101418","ink":"#f4f1ea"}`)
	mustContain(t, rec.Body.String(), `"ok":true`, "the look saves")
	theme := host.theme()
	if theme.Palette.Key != "custom" || theme.Palette.Scheme != "dark" || theme.Ground != "#101418" || theme.Ink != "#f4f1ea" || theme.Headings != "big" || theme.Width != "wide" {
		t.Fatalf("theme: %+v", theme)
	}
	css := theme.CSS()
	for _, want := range []string{"--site-ground:#101418;", "--site-ink:#f4f1ea;", "--site-surface:color-mix(in srgb, #f4f1ea 6%, #101418);", "--site-accent:#c05621;", "--site-heading-scale:1.2;", "--site-measure:84ch;", "color-scheme:dark;"} {
		if !strings.Contains(css, want) {
			t.Fatalf("css lacks %q: %s", want, css)
		}
	}
	mustContain(t, get(t, handler, "/").Body.String(), "--site-measure:84ch", "visitors get it")
	editor := get(t, handler, "/admin/edit/"+firstPageID(t, host, "menu")).Body.String()
	mustContain(t, editor, `data-look="true" id="look" data-custom="true"`, "the Look panel opens on custom")
	mustContain(t, editor, `data-look-ground="true" value="#101418"`, "with the owner's colours")
	mustContain(t, editor, `id="lookHeadings-big" value="big" data-look-headings="big" checked="checked"`, "and heading size")

	// Back to a preset: the custom colours are forgotten, a bad accent ignored.
	postJSON(t, handler, "/admin/api/theme", `{"palette":"fresh","fonts":"clean","accent":"red","headings":"nonsense","width":"narrow"}`)
	theme = host.theme()
	if theme.Palette.Key != "fresh" || theme.Ground != "" || theme.Headings != "regular" || theme.Width != "narrow" || theme.Accent != "" {
		t.Fatalf("after preset: %+v", theme)
	}

	// Owner CSS: kept, sanitised, on the page and the canvas.
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "countVisitors": "on",
		"customCss": ".site-title { letter-spacing: -0.02em }</style><script>alert(1)</script> @import url(evil); .x { background: url(javascript:alert(1)) }"}, nil)
	home := get(t, handler, "/").Body.String()
	mustContain(t, home, `<style data-site-custom="true">.site-title { letter-spacing: -0.02em }`, "the owner's CSS is on the page")
	if strings.Contains(home, "<script>alert") || strings.Contains(home, "@import") || strings.Contains(home, "javascript:") {
		t.Fatal("dangerous bits are stripped")
	}
	mustContain(t, editor, "", "")
	mustContain(t, get(t, handler, "/admin/edit/"+firstPageID(t, host, "menu")).Body.String(), `data-site-custom="true"`, "and on the canvas")
	mustContain(t, get(t, handler, "/admin/settings").Body.String(), `letter-spacing: -0.02em`, "and in the settings field")
}

func TestPicturesHaveSizeShapeCaptionAndLink(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[{"kind":"image","url":"/uploads/loaf.png","alt":"A loaf","size":"medium","shape":"round","caption":"Fresh **every** morning","link":"/visit"},{"kind":"image","url":"/uploads/oven.png","alt":"","size":"nonsense","shape":"nonsense"}]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	mustContain(t, public, `<figure class="site-figure site-figure--medium site-figure--crop-round"><a href="/visit"><img src="/uploads/loaf.png" alt="A loaf"`, "size, shape, and link")
	mustContain(t, public, `<figcaption>Fresh <strong>every</strong> morning</figcaption>`, "and a caption")
	mustContain(t, public, `<figure class="site-figure site-figure--full site-figure--crop-natural"><img src="/uploads/oven.png"`, "unknown choices fall back")
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, editor, `<select data-img-size="true" aria-label="Size"><option value="full">Full width</option><option value="wide">Wider than the text</option><option value="medium" selected="selected">`, "the canvas shows the size")
	mustContain(t, editor, `data-caption="true" value="Fresh **every** morning"`, "and the caption")
	fresh := get(t, handler, "/admin/api/blocks/image").Body.String()
	mustContain(t, fresh, `data-img-shape="true"`, "a fresh picture block comes from the server")
	mustContain(t, fresh, "No picture yet", "empty")
}

func TestOwnersCanNameTheirOwnFonts(t *testing.T) {
	host, handler := newTestHost(t)
	mustContain(t, postJSON(t, handler, "/admin/api/theme", `{"palette":"fresh","fonts":"custom","fontHead":"Playfair Display","fontBody":"Inter; drop table","headings":"regular","width":"regular"}`).Body.String(), `"ok":true`, "saves")
	theme := host.theme()
	if theme.Fonts.Key != "custom" || theme.FontHead != "Playfair Display" || theme.FontBody != "Inter drop table" {
		t.Fatalf("theme fonts: %+v head=%q body=%q", theme.Fonts, theme.FontHead, theme.FontBody)
	}
	home := get(t, handler, "/").Body.String()
	mustContain(t, home, `--site-font-display:"Playfair Display", ui-sans-serif, system-ui, sans-serif;`, "headings use the named font")
	mustContain(t, home, `https://fonts.googleapis.com/css2?family=Playfair+Display:wght@400;600;700&amp;family=Inter+drop+table:wght@400;600&amp;display=swap`, "and the fonts are loaded from Google")
	postJSON(t, handler, "/admin/api/theme", `{"palette":"fresh","fonts":"custom","fontHead":"","fontBody":""}`)
	if host.theme().Fonts.Key != "clean" {
		t.Fatal("custom with no names falls back to the first pairing")
	}
	postJSON(t, handler, "/admin/api/theme", `{"palette":"fresh","fonts":"modern","fontHead":"Lobster"}`)
	if theme := host.theme(); theme.Fonts.Key != "modern" || theme.FontHead != "" {
		t.Fatalf("a preset forgets the custom names: %+v", theme)
	}
}
