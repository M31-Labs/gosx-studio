package sitehost

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	"m31labs.dev/gosx-studio/cms/render"
)

func inlineHTML(text string) string { return gosx.RenderHTML(gosx.El("p", nil, renderInline(text))) }

func TestInlineMarkersRenderAsTags(t *testing.T) {
	cases := map[string]string{
		"plain":                             "<p>plain</p>",
		"**bold** and _italic_":             "<p><strong>bold</strong> and <em>italic</em></p>",
		"see [our menu](/menu) today":       `<p>see <a href="/menu">our menu</a> today</p>`,
		"[out](https://x.example/p)":        `<p><a href="https://x.example/p" rel="noopener">out</a></p>`,
		"[mail](mailto:a@b.example)":        `<p><a href="mailto:a@b.example">mail</a></p>`,
		"line one\nline two":                "<p>line one<br />line two</p>",
		"**[bold link](/x)**":               `<p><strong><a href="/x">bold link</a></strong></p>`,
		"snake_case_name stays":             "<p>snake_case_name stays</p>",
		"lonely ** stars":                   "<p>lonely ** stars</p>",
		"[not a link":                       "<p>[not a link</p>",
		"[x](javascript:alert(1))":          "<p>[x](javascript:alert(1))</p>",
		"[x](data:text/html,hi)":            "<p>[x](data:text/html,hi)</p>",
		"<script>alert(1)</script>":         "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>",
		"**<b>x</b>**":                      "<p><strong>&lt;b&gt;x&lt;/b&gt;</strong></p>",
		"[a](/p\" onclick=\"x)":             "<p>[a](/p&#34; onclick=&#34;x)</p>",
	}
	for in, want := range cases {
		if got := inlineHTML(in); got != want {
			t.Errorf("renderInline(%q)\n got %s\nwant %s", in, got, want)
		}
	}
}

func TestInlineToPlainStripsMarkers(t *testing.T) {
	if got := inlineToPlain("**Fresh** _bread_ at [our shop](/shop)"); got != "Fresh bread at our shop" {
		t.Fatalf("inlineToPlain = %q", got)
	}
}

func TestRenderBodyGroupsSectionsAndNewKinds(t *testing.T) {
	host, _ := newTestHost(t)
	doc := document(
		heading(0, 2, "Top"),
		para(1, "Intro with **bold**."),
		block(2, blockSection, values("style", "tinted")),
		block(3, blockList, values("text", "One\n_Two_\n\nThree")),
		block(4, blockDivider, values()),
		block(5, blockSection, values("style", "accent")),
		block(6, content.BlockQuote, values("text", "Loved it")),
		block(7, blockSection, values("style", "nonsense")),
	)
	html := gosx.RenderHTML(host.renderBody(doc, render.Hooks{}))
	mustContain(t, html, `<section class="site-section site-section--plain"><div class="site-section__inner"><h2>Top</h2><p>Intro with <strong>bold</strong>.</p></div></section>`, "leading blocks form a plain section")
	mustContain(t, html, `<section class="site-section site-section--tinted"><div class="site-section__inner"><ul class="site-list"><li>One</li><li><em>Two</em></li><li>Three</li></ul><hr class="site-divider" /></div></section>`, "a section break starts a tinted section with a list and divider")
	mustContain(t, html, `<section class="site-section site-section--accent"><div class="site-section__inner"><blockquote>Loved it</blockquote></div></section>`, "an accent section")
	if strings.Count(html, "<section") != 3 {
		t.Fatalf("an empty trailing section must not render: %s", html)
	}
	// Disabled blocks stay out.
	hidden := document(block(0, content.BlockParagraph, values("text", "shown")))
	hidden.Blocks = append(hidden.Blocks, blockstudio.BlockInstance{ID: "x", Key: content.BlockParagraph, Enabled: false, Values: values("text", "hidden")})
	if out := gosx.RenderHTML(host.renderBody(hidden, render.Hooks{})); strings.Contains(out, "hidden") {
		t.Fatal("a disabled block rendered")
	}
}

func TestEditorRoundTripsFormattingListsAndSections(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	payload := `{"title":"Menu","slug":"menu","blocks":[
		{"kind":"paragraph","text":"Try our **sourdough** and _rye_, or [book a table](/contact)."},
		{"kind":"section","style":"tinted"},
		{"kind":"list","text":"Baguette\nFocaccia"},
		{"kind":"divider"},
		{"kind":"section","style":"bogus"}
	]}`
	if rec := postJSON(t, handler, "/admin/api/pages/"+id, payload); !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Fatalf("save: %s", rec.Body.String())
	}
	page, _, _ := host.Store().PageByID(id)
	if len(page.Body.Blocks) != 5 {
		t.Fatalf("stored %d blocks", len(page.Body.Blocks))
	}
	if page.Body.Blocks[4].Values["style"].String != "plain" {
		t.Fatalf("bogus section style should normalize to plain, got %q", page.Body.Blocks[4].Values["style"].String)
	}

	// The canvas shows real formatting inside editable blocks.
	canvas := get(t, handler, "/admin/edit/"+id).Body.String()
	mustContain(t, canvas, `<strong>sourdough</strong>`, "bold shows on the canvas")
	mustContain(t, canvas, `<a href="/contact">book a table</a>`, "links show on the canvas")
	mustContain(t, canvas, `data-list="true"`, "the list is an editable list")
	mustContain(t, canvas, `<li>Focaccia</li>`, "list items render")
	mustContain(t, canvas, `data-section="tinted"`, "the section bar shows its style")
	mustContain(t, canvas, `<option value="tinted" selected="selected">`, "the style select reflects it")

	post(t, handler, "/admin/api/pages/"+id+"/publish", map[string][]string{})
	live := get(t, handler, "/menu").Body.String()
	mustContain(t, live, `<em>rye</em>`, "italic reaches visitors")
	mustContain(t, live, `site-section--tinted`, "the section background reaches visitors")
	mustContain(t, live, `<ul class="site-list"><li>Baguette</li>`, "the list reaches visitors")
}

func TestEditorOffersTheNewBlocks(t *testing.T) {
	host, handler := newTestHost(t)
	body := get(t, handler, "/admin/edit/"+firstPageID(t, host, "menu")).Body.String()
	for _, kind := range []string{"list", "divider", "section"} {
		mustContain(t, body, `data-add="`+kind+`"`, kind+" is in the sidebar")
		mustContain(t, body, `data-insert="`+kind+`"`, kind+" is in the insert menu")
	}
}
