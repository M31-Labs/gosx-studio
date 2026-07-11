package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

func TestInteractionsPanelRendersRevealAndHoverFocusRows(t *testing.T) {
	reveal := core.Interaction{
		Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFadeUp,
		Target: core.CanvasIdentity{Route: "/", BlockKey: "hero"}, Once: true,
	}.Normalize()
	revealDiagnostics := core.InteractionDiagnostics([]core.Interaction{reveal})

	node := RenderInteractionsPanel(InteractionsPanelOptions{
		ID: "interactions", Action: "/authoring", CSRFToken: "token",
		PageKey: "home", PageRoute: "/", ComponentKey: "hero",
		Rows: []InteractionRow{
			InteractionRowFromInteraction(core.InteractionRevealOnScroll, reveal, revealDiagnostics[0]),
			InteractionRowFromInteraction(core.InteractionHoverFocusState, core.Interaction{}, core.InteractionDiagnostic{}),
		},
	})
	html := gosx.RenderHTML(node)

	for _, want := range []string{
		"data-studio-interactions-inspector=\"true\"",
		"data-studio-interaction-card=\"reveal-on-scroll\"",
		"data-studio-interaction-card=\"hover-focus-state\"",
		"data-studio-interaction-attached=\"true\"",
		"data-studio-interaction-attached=\"false\"",
		"gosx_studio_interaction_kind",
		"gosx_studio_interaction_effect",
		"gosx_studio_interaction_duration_ms",
		"gosx_studio_interaction_delay_ms",
		"gosx_studio_interaction_once",
		"Reveal once",
		"Reduced motion: content shows immediately, no animation plays.",
		"Remove", // remove form only for the attached reveal row
		"data-gosx-studio-authoring-managed=\"true\"",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
}

func TestInteractionsPanelSurfacesBlockedReasonForInvalidTarget(t *testing.T) {
	invalid := core.Interaction{Kind: core.InteractionHoverFocusState} // no target identity
	diagnostics := core.InteractionDiagnostics([]core.Interaction{invalid})

	node := RenderInteractionsPanel(InteractionsPanelOptions{
		Rows: []InteractionRow{InteractionRowFromInteraction(core.InteractionHoverFocusState, invalid, diagnostics[0])},
	})
	html := gosx.RenderHTML(node)
	if !strings.Contains(html, "data-studio-interaction-reason=\"true\"") {
		t.Fatalf("expected a blocked-reason element, got:\n%s", html)
	}
	if !strings.Contains(html, "data-studio-interaction-status=\"blocked\"") {
		t.Fatalf("expected the row to carry blocked status, got:\n%s", html)
	}
}

func TestInteractionsPanelHoverRowHasNoOnceControl(t *testing.T) {
	hover := core.Interaction{Kind: core.InteractionHoverFocusState, Target: core.CanvasIdentity{Route: "/", BlockKey: "cta"}}.Normalize()
	diagnostics := core.InteractionDiagnostics([]core.Interaction{hover})
	node := RenderInteractionsPanel(InteractionsPanelOptions{
		Rows: []InteractionRow{InteractionRowFromInteraction(core.InteractionHoverFocusState, hover, diagnostics[0])},
	})
	html := gosx.RenderHTML(node)
	if strings.Contains(html, "Reveal once") {
		t.Fatal("hover-focus-state row must not offer a scroll-only 'reveal once' control")
	}
}
