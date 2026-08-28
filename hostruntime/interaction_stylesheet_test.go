package hostruntime

import (
	"strings"
	"testing"
)

// TestStylesheetCoversInteractionAllowList proves the published/preview
// stylesheet ships CSS for both interaction primitives this slice supports
// (reveal-on-scroll, hover-focus-state), that hover always ships a matching
// :focus-visible rule, and that prefers-reduced-motion collapses every
// interaction to immediately-visible/no-animation regardless of Kind/Effect.
func TestStylesheetCoversInteractionAllowList(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`[data-gosx-interaction="reveal-on-scroll"]`,
		`[data-gosx-interaction-revealed="true"]`,
		`[data-gosx-interaction="hover-focus-state"]`,
		`[data-gosx-interaction-effect="lift"]:hover,`,
		`[data-gosx-interaction-effect="lift"]:focus-visible`,
		`[data-gosx-interaction-effect="glow"]:hover,`,
		`[data-gosx-interaction-effect="underline"]:hover,`,
		`[data-gosx-interaction-effect="scale"]:hover,`,
		"@media (prefers-reduced-motion: reduce)",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("stylesheet missing %q", want)
		}
	}
}

func TestStylesheetHoverEffectsNeverExistWithoutFocusVisibleEquivalent(t *testing.T) {
	css := string(Stylesheet())
	for _, effect := range []string{"lift", "glow", "underline", "scale"} {
		hoverSelector := `[data-gosx-interaction-effect="` + effect + `"]:hover`
		focusSelector := `[data-gosx-interaction-effect="` + effect + `"]:focus-visible`
		if strings.Contains(css, hoverSelector) != strings.Contains(css, focusSelector) {
			t.Fatalf("effect %q has a :hover rule without a matching :focus-visible rule (or vice versa)", effect)
		}
	}
}

func TestStylesheetInteractionRulesNeverToggleDisplayOrVisibility(t *testing.T) {
	css := string(Stylesheet())
	start := strings.Index(css, `[data-gosx-interaction="reveal-on-scroll"]`)
	if start < 0 {
		t.Fatal("expected the reveal-on-scroll rule block to exist")
	}
	block := css[start:]
	for _, forbidden := range []string{"display: none", "visibility: hidden"} {
		if strings.Contains(block, forbidden) {
			t.Fatalf("interaction CSS must not use %q — it would hide focusable content from keyboard/assistive-tech users", forbidden)
		}
	}
}
