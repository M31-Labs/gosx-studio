package authoring

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
