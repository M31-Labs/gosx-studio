package canvas

import (
	"strings"
	"testing"
)

func TestRenderHomePageSurfaceMarkupEscapesAndFallsBack(t *testing.T) {
	markup := RenderHomePageSurfaceMarkup(HomePageSurfaceInput{
		Key:          `page:"home"`,
		PaletteClass: `theme <clay>`,
		CSS:          `.theme{color:red}`,
		Tagline:      `<slow craft>`,
		Headline:     " ",
		Subhead:      `cups & bowls`,
		CTALabel:     "",
		CTAURL:       `/shop?kind="cups"&q=<new>`,
	})

	for _, want := range []string{
		`class="theme &lt;clay&gt;"`,
		`data-gosx-html-key="page:&#34;home&#34;"`,
		`data-studio-page-surface="home"`,
		`<style>.theme{color:red}</style>`,
		`data-studio-field="tagline"`,
		`data-studio-field="home.hero.headline"`,
		`data-studio-field="home.hero.subhead"`,
		`data-studio-field="home.hero.ctaLabel"`,
		`data-studio-editable="text"`,
		`contenteditable="true"`,
		`&lt;slow craft&gt;`,
		`Ceramic pieces with evidence of the hand`,
		`cups &amp; bowls`,
		`Shop available work`,
		`href="/shop?kind=&#34;cups&#34;&amp;q=&lt;new&gt;"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("home surface missing %q\nmarkup: %s", want, markup)
		}
	}
	if strings.Contains(markup, `<slow craft>`) {
		t.Fatalf("home surface did not escape tagline: %s", markup)
	}
}

func TestRenderPageArtboardSurfaceMarkupEditableAndComponents(t *testing.T) {
	markup := RenderPageArtboardSurfaceMarkup(PageArtboardSurfaceInput{
		Key:           `page:"about"`,
		PageKey:       `about&team`,
		Title:         `<About Us>`,
		Route:         `/about?draft="1"`,
		Group:         "content",
		Status:        "draft",
		Component:     `AboutPage`,
		Binding:       `cms.pages.about`,
		EditableTitle: true,
		PaletteClass:  `theme <clay>`,
		CSS:           `.theme{background:white}`,
		Components: []PageArtboardComponent{
			{Label: `<Hero>`, Detail: `ready / cms`},
			{},
		},
	})

	for _, want := range []string{
		`class="theme &lt;clay&gt;"`,
		`data-gosx-html-key="page:&#34;about&#34;"`,
		`data-studio-page-surface="about&amp;team"`,
		`<style>.theme{background:white}</style>`,
		`/about?draft=&#34;1&#34; / content / draft`,
		`data-studio-field="pages.about&amp;team.title"`,
		`data-studio-field-label="Title"`,
		`data-studio-editable="text"`,
		`contenteditable="true"`,
		`data-studio-operation="update-page"`,
		`data-studio-page-key="about&amp;team"`,
		`data-studio-page-route="/about?draft=&#34;1&#34;"`,
		`&lt;About Us&gt;`,
		`AboutPage`,
		`cms.pages.about`,
		`&lt;Hero&gt;`,
		`ready / cms`,
		`Component`,
		`Not configured`,
		`CanvasBoard HTML surface`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("page artboard surface missing %q\nmarkup: %s", want, markup)
		}
	}
	if strings.Contains(markup, `<About Us>`) || strings.Contains(markup, `<Hero>`) {
		t.Fatalf("page artboard surface did not escape text: %s", markup)
	}
}

// TestRenderHomePageSurfaceMarkupOmitsDurableAttrsByDefault guards handoff-31:
// the durable opt-in is additive-only — a caller that never sets
// DurableHeadlineField (every pre-existing caller) must render byte-for-byte
// the same headline element it always did.
func TestRenderHomePageSurfaceMarkupOmitsDurableAttrsByDefault(t *testing.T) {
	markup := RenderHomePageSurfaceMarkup(HomePageSurfaceInput{Headline: "Hi"})
	if strings.Contains(markup, "data-gosx-studio-durable-history") {
		t.Fatalf("headline must not carry durable attrs when DurableHeadlineField is blank: %s", markup)
	}
}

// TestRenderHomePageSurfaceMarkupHeadlineDurableOptIn proves the headline
// surface renders the durable dispatch attributes inlineeditruntime's
// island_runtime.js reads (DURABLE_ATTR/DURABLE_FIELD_ATTR/etc.) when a host
// supplies DurableHeadlineField, and that the durable Field literal need not
// match the canvas binding string (see the struct doc comment: "hero.headline"
// vs "home.hero.headline").
func TestRenderHomePageSurfaceMarkupHeadlineDurableOptIn(t *testing.T) {
	markup := RenderHomePageSurfaceMarkup(HomePageSurfaceInput{
		Headline:                 "Hi",
		DurableHeadlineField:     "hero.headline",
		DurableHeadlinePageID:    "page:home",
		DurableHeadlineRoute:     "/",
		DurableHeadlineComponent: "home:hero",
	})
	for _, want := range []string{
		`data-studio-field="home.hero.headline"`,
		`data-gosx-studio-durable-history="true"`,
		`data-gosx-studio-durable-field="hero.headline"`,
		`data-gosx-studio-durable-page-id="page:home"`,
		`data-gosx-studio-durable-route="/"`,
		`data-gosx-studio-durable-component="home:hero"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing %q in:\n%s", want, markup)
		}
	}
}

// TestRenderPageArtboardSurfaceMarkupTitleDurableOptIn proves an editable
// page title surface renders the SAME durable dispatch attributes when a
// host supplies DurableTitleField (e.g. the about page, which already has a
// durable "pages.about.title" set-field ledger case) — additive alongside
// the pre-existing data-studio-operation="update-page" fallback attributes.
func TestRenderPageArtboardSurfaceMarkupTitleDurableOptIn(t *testing.T) {
	markup := RenderPageArtboardSurfaceMarkup(PageArtboardSurfaceInput{
		Key: "page:about", PageKey: "about", Title: "About", Route: "/about", EditableTitle: true,
		DurableTitleField: "pages.about.title", DurableTitlePageID: "page:about", DurableTitleComponent: "about:content",
	})
	for _, want := range []string{
		`data-studio-field="pages.about.title"`,
		`data-studio-operation="update-page"`,
		`data-gosx-studio-durable-history="true"`,
		`data-gosx-studio-durable-field="pages.about.title"`,
		`data-gosx-studio-durable-page-id="page:about"`,
		`data-gosx-studio-durable-route="/about"`,
		`data-gosx-studio-durable-component="about:content"`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("missing %q in:\n%s", want, markup)
		}
	}
}

func TestRenderPageArtboardSurfaceMarkupPlainTitleFallbacks(t *testing.T) {
	markup := RenderPageArtboardSurfaceMarkup(PageArtboardSurfaceInput{
		Key:     "page:blank",
		PageKey: "blank",
	})

	for _, want := range []string{
		`data-studio-page-surface="blank"`,
		`>blank</h1>`,
		`>/blank</p>`,
		`>Not configured</p>`,
		`CanvasBoard HTML surface`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("plain page artboard surface missing %q\nmarkup: %s", want, markup)
		}
	}
	for _, notWant := range []string{
		`data-studio-operation="update-page"`,
		`contenteditable="true"`,
		`data-studio-editable="text"`,
	} {
		if strings.Contains(markup, notWant) {
			t.Fatalf("plain page artboard surface unexpectedly contained %q\nmarkup: %s", notWant, markup)
		}
	}
}
