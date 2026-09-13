package sitehost

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestReadyMadeSectionsRoundTripFromEditorToPage(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	payload := `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"hero","variant":"split","fields":{"eyebrow":"Since 2014","headline":"Bread worth **the walk**","text":"Baked before dawn.","button":"See the menu","url":"/menu","image":"/uploads/loaf.png"}},
		{"kind":"section","style":"dark","align":"center","width":"wide","space":"roomy"},
		{"kind":"features","variant":"numbered","fields":{"heading":"Why us"},"items":[{"icon":"🥖","title":"Fresh","text":"Every morning."},{"icon":"","title":"Local","text":"Flour from the valley."},{"icon":"","title":"","text":""}]},
		{"kind":"faq","variant":"accordion","fields":{"heading":"Questions"},"items":[{"question":"Do you deliver?","answer":"Within the city, yes."},{"question":"Gluten free?","answer":"Saturdays."}]},
		{"kind":"pricing","fields":{"heading":"Plans"},"items":[{"name":"Weekly loaf","price":"$8","period":"a week","features":"One loaf\nPick up Saturday","button":"Start","url":"/contact","highlight":"yes"}]},
		{"kind":"section","style":"image","url":"/uploads/oven.jpg"},
		{"kind":"cta","variant":"band","fields":{"headline":"Come by","text":"We open at seven.","button":"Find us","url":"/contact"}},
		{"kind":"stats","items":[{"value":"12","label":"years"},{"value":"","label":""}]},
		{"kind":"hours","variant":"table","fields":{"heading":"Hours"},"items":[{"day":"Mon–Fri","time":"7–3"}]},
		{"kind":"map","variant":"compact","fields":{"address":"1 Mill Lane, Oakland"}},
		{"kind":"spacer","variant":"large"},
		{"kind":"testimonials","fields":{"heading":""},"items":[]},
		{"kind":"imagetext","variant":"left","fields":{"image":"/uploads/loaf.png","heading":"Our oven","text":"Wood fired."}},
		{"kind":"features","variant":"bogus","fields":{"heading":""},"items":[]}
	]}`
	rec := postJSON(t, handler, "/admin/api/pages/"+id, payload)
	mustContain(t, rec.Body.String(), `"ok":true`, "the save works")
	page, _, _ := host.Store().PageByID(id)
	keys := []string{}
	for _, block := range page.Body.Blocks {
		keys = append(keys, block.Key)
	}
	want := "hero section features faq pricing section cta stats hours map spacer imagetext"
	if strings.Join(keys, " ") != want {
		t.Fatalf("stored blocks = %q, want %q", strings.Join(keys, " "), want)
	}
	features := page.Body.Blocks[2]
	if features.Values["variant"].String != "numbered" || len(features.Values["items"].List) != 2 {
		t.Fatalf("features: %+v", features.Values)
	}
	if page.Body.Blocks[4].Values["items"].List[0].Object["highlight"].String != "yes" {
		t.Fatal("the highlight flag is kept")
	}

	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	for _, want := range []string{
		`class="site-hero site-hero--split"`, "Bread worth <strong>the walk</strong>", `class="button button--primary" href="/menu">See the menu</a>`, "site-hero__picture",
		`class="site-section site-section--dark site-section--align-center site-section--width-wide site-section--space-roomy"`,
		`class="site-features site-features--numbered"`, `<span class="site-features__icon">1</span>`, "Flour from the valley",
		`<details class="site-faq__item" open="open" name="site-faq-faq-3">`, "<summary class=\"site-faq__question\">Do you deliver?</summary>",
		"site-pricing__card site-pricing__card--highlight", `<ul class="site-pricing__features"><li>One loaf</li><li>Pick up Saturday</li></ul>`,
		`class="site-section site-section--image" style="--section-image: url(&#39;/uploads/oven.jpg&#39;)"`,
		`class="site-cta site-cta--band"`, `<strong class="site-stats__value">12</strong>`, `<span class="site-hours__day">Mon–Fri</span>`,
		`maps.google.com/maps?q=1+Mill+Lane%2C+Oakland`, `class="site-spacer site-spacer--large"`, `class="site-imagetext site-imagetext--left"`,
	} {
		if !strings.Contains(public, want) {
			t.Fatalf("public page lacks %q:\n%s", want, public)
		}
	}
	if strings.Count(public, "site-features__card") != 2 {
		t.Fatalf("empty cards are dropped: %d cards", strings.Count(public, "site-features__card"))
	}
	if strings.Contains(public, "site-testimonials") || strings.Contains(public, "data-field") || strings.Contains(public, "contenteditable") {
		t.Fatal("empty sections stay off the page and nothing editable leaks")
	}

	// The canvas shows the same sections, editable, with layout pickers.
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	for _, want := range []string{
		`data-composite="hero"`, `data-field="headline"`, `<select data-variant="true"`, `data-item-add="features"`, `data-item="true"`, `data-item-remove="true"`,
		`data-section-align="true"`, `data-section-width="true"`, `data-section-src="true"`, `data-server-kinds="section,image,hero,`, `data-add="testimonials"`,
		`type="checkbox" data-field="highlight" value="yes" checked="checked"`,
	} {
		if !strings.Contains(editor, want) {
			t.Fatalf("editor lacks %q", want)
		}
	}
}

func TestEditorFetchesFreshSectionsAndItemsFromTheServer(t *testing.T) {
	_, handler := newTestHost(t)
	rec := get(t, handler, "/admin/api/blocks/hero")
	if rec.Code != http.StatusOK {
		t.Fatalf("fresh hero = %d", rec.Code)
	}
	mustContain(t, rec.Body.String(), `data-composite="hero"`, "a fresh hero")
	mustContain(t, rec.Body.String(), "Say the one thing you want people to know", "with its starter copy")
	mustContain(t, get(t, handler, "/admin/api/blocks/section").Body.String(), `data-section-style="true"`, "a fresh section bar")
	item := get(t, handler, "/admin/api/blocks/features/item").Body.String()
	if !strings.HasPrefix(item, `<li class="site-features__card ed-item" data-item="true">`) || !strings.HasSuffix(item, "</li>") || strings.Count(item, `data-item="true"`) != 1 {
		t.Fatalf("one fresh card, exactly: %q", item)
	}
	if code := get(t, handler, "/admin/api/blocks/nope").Code; code != http.StatusNotFound {
		t.Fatal("unknown kinds are 404")
	}
	if code := get(t, handler, "/admin/api/blocks/spacer/item").Code; code != http.StatusNotFound {
		t.Fatal("kinds without items have no item endpoint")
	}
	pricing := get(t, handler, "/admin/api/blocks/pricing/item").Body.String()
	mustContain(t, pricing, `data-field="highlight"`, "a fresh plan has its flag")
	if strings.Contains(pricing, "checked") {
		t.Fatal("a fresh plan is not highlighted")
	}
}
