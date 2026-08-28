package core

import "testing"

func TestInteractionNormalizeAssignsTriggerAndDefaultKey(t *testing.T) {
	interaction := Interaction{
		Kind:   InteractionRevealOnScroll,
		Target: CanvasIdentity{Route: "/", BlockKey: "hero"},
	}.Normalize()

	if interaction.Trigger != InteractionTriggerScroll {
		t.Fatalf("expected scroll trigger, got %q", interaction.Trigger)
	}
	if interaction.Key != "hero:reveal-on-scroll" {
		t.Fatalf("expected derived key, got %q", interaction.Key)
	}
	if interaction.Effect != InteractionEffectFade {
		t.Fatalf("expected default reveal effect, got %q", interaction.Effect)
	}
	if interaction.DurationMS != DefaultInteractionDurationMS {
		t.Fatalf("expected default duration, got %d", interaction.DurationMS)
	}
	if interaction.ReducedMotion != ReducedMotionImmediate {
		t.Fatalf("reduced motion must always be immediate, got %q", interaction.ReducedMotion)
	}
}

func TestInteractionHoverFocusStateAlwaysUsesHoverTrigger(t *testing.T) {
	interaction := Interaction{
		Kind:   InteractionHoverFocusState,
		Target: CanvasIdentity{Route: "/", BlockKey: "cta"},
		Effect: InteractionEffectLift,
	}.Normalize()

	if interaction.Trigger != InteractionTriggerHover {
		t.Fatalf("expected hover trigger, got %q", interaction.Trigger)
	}
	if !interaction.Valid() {
		t.Fatalf("expected valid interaction, got errors: %v", interaction.Validate())
	}
}

func TestInteractionRejectsUnsupportedKind(t *testing.T) {
	interaction := Interaction{
		Kind:   "eval-script",
		Target: CanvasIdentity{Route: "/", BlockKey: "hero"},
	}

	errs := interaction.Validate()
	if errs["kind"] == "" {
		t.Fatalf("expected kind error, got %v", errs)
	}
}

func TestInteractionRejectsMissingTargetIdentity(t *testing.T) {
	interaction := Interaction{
		Kind: InteractionRevealOnScroll,
		// No Route/Field/BlockKey/NodeID/PageID: target cannot be stable.
	}

	errs := interaction.Validate()
	if errs["target"] == "" {
		t.Fatalf("expected target error, got %v", errs)
	}
	if interaction.ReadinessStatus() != ReadinessBlocked {
		t.Fatalf("expected blocked readiness, got %q", interaction.ReadinessStatus())
	}
}

func TestInteractionRejectsMismatchedEffect(t *testing.T) {
	interaction := Interaction{
		Kind:   InteractionRevealOnScroll,
		Target: CanvasIdentity{Route: "/", BlockKey: "hero"},
		Effect: InteractionEffectLift, // a hover-only effect
	}
	errs := interaction.Validate()
	if errs["effect"] == "" {
		t.Fatalf("expected effect error for mismatched kind, got %v", errs)
	}
}

func TestInteractionDiagnosticsReportBlockedForMissingTarget(t *testing.T) {
	interactions := []Interaction{
		{Kind: InteractionRevealOnScroll, Target: CanvasIdentity{Route: "/", BlockKey: "hero"}},
		{Kind: InteractionHoverFocusState}, // missing target
	}

	diagnostics := InteractionDiagnostics(interactions)
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diagnostics))
	}
	if diagnostics[0].Status != ReadinessReady {
		t.Fatalf("expected first interaction ready, got %q", diagnostics[0].Status)
	}
	if diagnostics[1].Status != ReadinessBlocked {
		t.Fatalf("expected second interaction blocked, got %q", diagnostics[1].Status)
	}
	if diagnostics[1].Reason == "" {
		t.Fatalf("expected an actionable reason for the blocked interaction")
	}

	blocked := BlockedInteractionDiagnostics(diagnostics)
	if len(blocked) != 1 || blocked[0].Key != "" {
		t.Fatalf("expected exactly the invalid interaction in the blocked list, got %+v", blocked)
	}
}

func TestInteractionRenderAttrsOmittedWhenInvalid(t *testing.T) {
	interaction := Interaction{Kind: InteractionRevealOnScroll} // no target
	if attrs := interaction.RenderAttrs(); attrs != nil {
		t.Fatalf("expected no render attrs for an invalid interaction, got %v", attrs)
	}
}

func TestInteractionRenderAttrsForValidInteraction(t *testing.T) {
	interaction := Interaction{
		Kind:   InteractionRevealOnScroll,
		Target: CanvasIdentity{Route: "/", BlockKey: "hero"},
		Effect: InteractionEffectFadeUp,
		Once:   true,
	}
	attrs := interaction.RenderAttrs()
	if attrs["data-gosx-interaction"] != "reveal-on-scroll" {
		t.Fatalf("unexpected kind attr: %v", attrs)
	}
	if attrs["data-gosx-interaction-effect"] != "fade-up" {
		t.Fatalf("unexpected effect attr: %v", attrs)
	}
	if attrs["data-gosx-interaction-once"] != "true" {
		t.Fatalf("unexpected once attr: %v", attrs)
	}
	if attrs["data-gosx-interaction-duration-ms"] == "" {
		t.Fatal("expected a duration attr")
	}
}

func TestInteractionLabelsAreHumanReadable(t *testing.T) {
	if InteractionKindLabel(InteractionRevealOnScroll) == "" {
		t.Fatal("expected a label for reveal-on-scroll")
	}
	if InteractionKindLabel(InteractionHoverFocusState) == "" {
		t.Fatal("expected a label for hover-focus-state")
	}
	if InteractionKindLabel("bogus") != "Unsupported interaction" {
		t.Fatalf("expected fallback label for unsupported kind, got %q", InteractionKindLabel("bogus"))
	}
	if InteractionEffectLabel(InteractionEffectFadeUp) == "" {
		t.Fatal("expected an effect label")
	}
}
