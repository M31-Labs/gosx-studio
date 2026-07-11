package studio

import (
	"testing"

	"m31labs.dev/gosx-studio/core"
)

func TestBindingPublishChecksRejectsAnInvalidFlowBinding(t *testing.T) {
	unconnected := core.Flow{Key: "contact", Label: "Contact"} // no HandlerRef
	resolved, reason := unconnected.BindingResolved()
	if resolved {
		t.Fatal("expected an unconnected flow binding to be unresolved")
	}

	diagnostics := []core.BindingDiagnostic{
		{PageKey: "home", PageLabel: "Home", ComponentKey: "contact-form", ComponentLabel: "Contact form", Binding: "flow.contact", Resolved: resolved, Reason: reason},
	}
	checks := BindingPublishChecks(diagnostics)
	if len(checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(checks))
	}
	if checks[0].Status != ReadinessNext {
		t.Fatalf("expected an invalid binding to block publish (ReadinessNext), got %q", checks[0].Status)
	}
	if checks[0].Detail == "" {
		t.Fatal("expected an actionable detail for the broken binding")
	}
}

func TestBindingPublishChecksAcceptsAResolvedFlowBinding(t *testing.T) {
	connected := core.Flow{
		Key: "contact", Label: "Contact", HandlerRef: "handlers.email.contact", CanExecute: true,
		Steps:   []core.FlowStep{{Key: "message", Label: "Message"}},
		Actions: []core.FlowAction{{Key: "submit", Label: "Send", HandlerRef: "handlers.email.contact"}},
	}
	resolved, reason := connected.BindingResolved()
	if !resolved {
		t.Fatalf("expected a connected, executable flow binding to resolve; reason=%q", reason)
	}

	diagnostics := []core.BindingDiagnostic{
		{PageKey: "home", ComponentKey: "contact-form", Binding: "flow.contact", Resolved: resolved, Reason: reason},
	}
	checks := BindingPublishChecks(diagnostics)
	if len(checks) != 1 || checks[0].Status != ReadinessReady {
		t.Fatalf("expected a ready check, got %#v", checks)
	}
}

func TestInteractionPublishChecksRejectsAnInvalidTarget(t *testing.T) {
	diagnostics := core.InteractionDiagnostics([]core.Interaction{
		{Kind: core.InteractionRevealOnScroll, Target: core.CanvasIdentity{Route: "/", BlockKey: "hero"}},
		{Kind: core.InteractionHoverFocusState}, // missing target identity
	})
	checks := InteractionPublishChecks(diagnostics)
	if len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(checks))
	}
	if checks[0].Status != ReadinessReady {
		t.Fatalf("expected the well-formed interaction to be ready, got %q", checks[0].Status)
	}
	if checks[1].Status != ReadinessNext {
		t.Fatalf("expected the interaction with no target to block publish (ReadinessNext), got %q", checks[1].Status)
	}
	if checks[1].Detail == "" {
		t.Fatal("expected an actionable detail for the blocked interaction")
	}
}
