package authoring

import (
	"errors"
	"testing"

	"m31labs.dev/gosx-studio/core"
)

func TestApplySetInteractionRevealOnScroll(t *testing.T) {
	var gotPage, gotComponent string
	var gotInteraction core.Interaction
	write := InteractionWriter(func(pageKey, componentKey string, interaction core.Interaction) error {
		gotPage, gotComponent, gotInteraction = pageKey, componentKey, interaction
		return nil
	})

	page := core.Page{Key: "home", Label: "Home", Route: "/"}
	component := core.Component{Key: "hero", Label: "Hero"}
	m := AuthoringMutationForSetInteraction(page, component, core.Interaction{
		Kind:   core.InteractionRevealOnScroll,
		Effect: core.InteractionEffectFadeUp,
		Once:   true,
	})

	result, err := ApplySetInteraction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotPage != "home" || gotComponent != "hero" {
		t.Fatalf("unexpected writer target: %q %q", gotPage, gotComponent)
	}
	if gotInteraction.Kind != core.InteractionRevealOnScroll {
		t.Fatalf("unexpected interaction kind: %q", gotInteraction.Kind)
	}
	if gotInteraction.Trigger != core.InteractionTriggerScroll {
		t.Fatalf("expected scroll trigger, got %q", gotInteraction.Trigger)
	}
	if !gotInteraction.Target.Stable() {
		t.Fatal("expected a stable target identity")
	}
	if gotInteraction.ReducedMotion != core.ReducedMotionImmediate {
		t.Fatalf("expected immediate reduced motion, got %q", gotInteraction.ReducedMotion)
	}
	if !result.RefreshPreview {
		t.Fatal("expected RefreshPreview so the canvas re-renders the interaction")
	}
}

func TestApplySetInteractionHoverFocusState(t *testing.T) {
	var gotInteraction core.Interaction
	write := InteractionWriter(func(pageKey, componentKey string, interaction core.Interaction) error {
		gotInteraction = interaction
		return nil
	})
	page := core.Page{Key: "home", Label: "Home", Route: "/"}
	component := core.Component{Key: "cta", Label: "CTA"}
	m := AuthoringMutationForSetInteraction(page, component, core.Interaction{
		Kind:   core.InteractionHoverFocusState,
		Effect: core.InteractionEffectLift,
	})

	result, err := ApplySetInteraction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotInteraction.Trigger != core.InteractionTriggerHover {
		t.Fatalf("expected hover trigger (which the CSS also keys :focus-visible off), got %q", gotInteraction.Trigger)
	}
}

func TestApplySetInteractionRejectsMissingTarget(t *testing.T) {
	called := false
	write := InteractionWriter(func(pageKey, componentKey string, interaction core.Interaction) error {
		called = true
		return nil
	})
	m := AuthoringMutation{Kind: AuthoringOperationSetInteraction, InteractionKind: core.InteractionRevealOnScroll}
	result, err := ApplySetInteraction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("write must not be called for a mutation with no target component")
	}
	if !result.HasErrors() {
		t.Fatal("expected field errors for a missing target")
	}
}

func TestApplySetInteractionRejectsUnsupportedKind(t *testing.T) {
	write := InteractionWriter(func(pageKey, componentKey string, interaction core.Interaction) error {
		t.Fatal("write must not be called for an unsupported kind")
		return nil
	})
	m := AuthoringMutation{
		Kind:            AuthoringOperationSetInteraction,
		PageKey:         "home",
		ComponentKey:    "hero",
		InteractionKind: "eval-script",
	}
	result, err := ApplySetInteraction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors[AuthoringFieldInteractionKind]; !ok {
		t.Fatalf("expected an interaction-kind field error, got %#v", result.FieldErrors)
	}
}

func TestApplySetInteractionRejectsMismatchedEffect(t *testing.T) {
	write := InteractionWriter(func(pageKey, componentKey string, interaction core.Interaction) error {
		t.Fatal("write must not be called for a mismatched effect")
		return nil
	})
	m := AuthoringMutation{
		Kind:              AuthoringOperationSetInteraction,
		PageKey:           "home",
		ComponentKey:      "hero",
		InteractionKind:   core.InteractionRevealOnScroll,
		InteractionEffect: core.InteractionEffectLift, // hover-only effect
	}
	result, err := ApplySetInteraction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors[AuthoringFieldInteractionEffect]; !ok {
		t.Fatalf("expected an interaction-effect field error, got %#v", result.FieldErrors)
	}
}

func TestApplySetInteractionPropagatesWriterError(t *testing.T) {
	write := InteractionWriter(func(pageKey, componentKey string, interaction core.Interaction) error {
		return errors.New("store unavailable")
	})
	page := core.Page{Key: "home", Route: "/"}
	component := core.Component{Key: "hero"}
	m := AuthoringMutationForSetInteraction(page, component, core.Interaction{Kind: core.InteractionRevealOnScroll})
	if _, err := ApplySetInteraction(m, write); err == nil {
		t.Fatal("expected the writer's error to propagate")
	}
}

func TestApplyRemoveInteraction(t *testing.T) {
	var gotPage, gotComponent, gotKey string
	write := InteractionRemoveWriter(func(pageKey, componentKey, interactionKey string) error {
		gotPage, gotComponent, gotKey = pageKey, componentKey, interactionKey
		return nil
	})
	page := core.Page{Key: "home", Route: "/"}
	component := core.Component{Key: "hero"}
	m := AuthoringMutationForRemoveInteraction(page, component, "hero:reveal-on-scroll")

	result, err := ApplyRemoveInteraction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotPage != "home" || gotComponent != "hero" || gotKey != "hero:reveal-on-scroll" {
		t.Fatalf("unexpected writer args: %q %q %q", gotPage, gotComponent, gotKey)
	}
}

func TestApplyRemoveInteractionRequiresKey(t *testing.T) {
	write := InteractionRemoveWriter(func(pageKey, componentKey, interactionKey string) error {
		t.Fatal("write must not be called without an interaction key")
		return nil
	})
	m := AuthoringMutation{Kind: AuthoringOperationRemoveInteraction, PageKey: "home", ComponentKey: "hero"}
	result, err := ApplyRemoveInteraction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors[AuthoringFieldInteractionKey]; !ok {
		t.Fatalf("expected an interaction-key field error, got %#v", result.FieldErrors)
	}
}

// --- AuthoringMutationFromForm/Validate integration for the new kinds ---

func TestAuthoringMutationValidateAcceptsInteractionOperationKinds(t *testing.T) {
	cases := []AuthoringMutation{
		{Kind: AuthoringOperationSetInteraction, PageKey: "home", ComponentKey: "hero", PageRoute: "/", InteractionKind: core.InteractionRevealOnScroll},
		{Kind: AuthoringOperationRemoveInteraction, PageKey: "home", ComponentKey: "hero", InteractionKey: "hero:reveal-on-scroll"},
	}
	for _, mutation := range cases {
		if validation := mutation.Validate(); !validation.OK() {
			t.Fatalf("expected %q to validate, got errors: %#v", mutation.Kind, validation.FieldErrors)
		}
	}
}

func TestAuthoringMutationValidateRejectsIncompleteInteractionOperations(t *testing.T) {
	cases := []AuthoringOperationKind{AuthoringOperationSetInteraction, AuthoringOperationRemoveInteraction}
	for _, kind := range cases {
		mutation := AuthoringMutation{Kind: kind}
		if validation := mutation.Validate(); validation.OK() {
			t.Fatalf("expected %q with no target fields to fail validation", kind)
		}
	}
}
