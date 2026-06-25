package studio

import (
	"strings"
	"testing"
)

func TestRenderComponentStylesCSSEmptyIsBlank(t *testing.T) {
	if got := RenderComponentStylesCSS(nil); got != "" {
		t.Fatalf("no overrides must render empty CSS, got %q", got)
	}
	if got := RenderComponentStylesCSS(StyleOverrides{}); got != "" {
		t.Fatalf("empty overrides must render empty CSS, got %q", got)
	}
}

func TestRenderComponentStylesCSSScopedRules(t *testing.T) {
	css := RenderComponentStylesCSS(StyleOverrides{
		"hero": {
			"base/default": {"padding-block": "96px", "text-align": "center"},
			"mobile/hover": {"color": "#b5613f"},
		},
	})
	for _, want := range []string{
		`[data-gosx-component="hero"]{`,
		"padding-block:96px;",
		"text-align:center;",
		"@media (max-width:47.99rem)",
		`[data-gosx-component="hero"]:hover{color:#b5613f;}`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("CSS missing %q\n%s", want, css)
		}
	}
}

func TestRenderComponentStylesCSSUsesGrammarVocabulary(t *testing.T) {
	// tablet breakpoint + focus state, asserting the renderer speaks the same
	// StyleBreakpoint*/StyleState* vocabulary as the set-style grammar.
	css := RenderComponentStylesCSS(StyleOverrides{
		"cta": {StyleBreakpointTablet + "/" + StyleStateFocus: {"opacity": "0.8"}},
	})
	if !strings.Contains(css, "@media (max-width:63.99rem)") || !strings.Contains(css, `[data-gosx-component="cta"]:focus{opacity:0.8;}`) {
		t.Fatalf("tablet/focus override rendered wrong:\n%s", css)
	}
}

func TestParseSectionFieldBinding(t *testing.T) {
	cases := []struct {
		binding string
		key     string
		field   string
		ok      bool
	}{
		{"home.section.pagehead.eyebrow", "pagehead", "eyebrow", true},
		{"section.collection.category", "collection", "category", true},
		{"home.hero.headline", "", "", false},
		{"tagline", "", "", false},
		{"home.section.pagehead", "", "", false},
	}
	for _, tc := range cases {
		key, field, ok := ParseSectionFieldBinding(tc.binding)
		if ok != tc.ok || key != tc.key || field != tc.field {
			t.Fatalf("ParseSectionFieldBinding(%q) = (%q,%q,%v), want (%q,%q,%v)", tc.binding, key, field, ok, tc.key, tc.field, tc.ok)
		}
	}
	// Round-trip through the inverse builder.
	key, field, ok := ParseSectionFieldBinding(SectionFieldBinding("postlist", "tag"))
	if !ok || key != "postlist" || field != "tag" {
		t.Fatalf("SectionFieldBinding round-trip failed: (%q,%q,%v)", key, field, ok)
	}
}

func TestStyleFormIDPart(t *testing.T) {
	if got := StyleFormIDPart("hero copy_2!"); got != "hero-copy-2-" {
		t.Fatalf("StyleFormIDPart sanitization wrong: %q", got)
	}
}
