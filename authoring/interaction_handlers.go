package authoring

import (
	"strings"

	"m31labs.dev/gosx-studio/core"
)

// This file is the host-agnostic body for the no-code interaction schema
// (core/interactions.go): set-interaction and remove-interaction. It follows
// the same shape as instance_handlers.go — Studio re-validates the mutation
// against core's strict allow-lists and delegates the actual draft write to a
// host-supplied writer closure, so hosts never have to duplicate the
// kind/trigger/effect/target validation.

// InteractionWriter persists one interaction (create or update) attached to a
// placed component, identified by pageKey+componentKey. The interaction is
// already normalized and validated by the time write is called.
type InteractionWriter func(pageKey, componentKey string, interaction core.Interaction) error

// InteractionRemoveWriter detaches one interaction, identified by its key,
// from a placed component.
type InteractionRemoveWriter func(pageKey, componentKey, interactionKey string) error

// ApplySetInteraction is the host-agnostic body for the set-interaction
// authoring operation. It builds a core.Interaction from the mutation,
// rejects anything outside core's allow-lists before ever calling write, and
// reports every field problem at once so the panel can highlight them
// together.
func ApplySetInteraction(m AuthoringMutation, write InteractionWriter) (AuthoringMutationResult, error) {
	m = m.Normalize()
	interaction := interactionFromMutation(m)
	if errs := interaction.Validate(); len(errs) > 0 {
		return AuthoringMutationResult{
			Message:     "Choose a supported interaction and attach it to an identified element.",
			FieldErrors: interactionFieldErrors(errs),
			Values:      m.FormValues(),
		}, nil
	}
	if strings.TrimSpace(m.PageKey) == "" || strings.TrimSpace(m.ComponentKey) == "" {
		return AuthoringMutationResult{
			Message:     "Choose a component to attach this interaction to.",
			FieldErrors: map[string]string{AuthoringFieldComponentKey: "Choose a component."},
			Values:      m.FormValues(),
		}, nil
	}
	if err := write(m.PageKey, m.ComponentKey, interaction); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        core.InteractionKindLabel(interaction.Kind) + " saved.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "interaction:" + m.ComponentKey + ":" + interaction.Key,
			Label:     core.InteractionKindLabel(interaction.Kind),
			Summary:   "Attached " + core.InteractionKindLabel(interaction.Kind) + " with the " + core.InteractionEffectLabel(interaction.Effect) + " effect.",
			Kind:      "interaction",
			PageKey:   m.PageKey,
			Component: m.ComponentKey,
		}},
	}, nil
}

// ApplyRemoveInteraction is the host-agnostic body for the remove-interaction
// authoring operation.
func ApplyRemoveInteraction(m AuthoringMutation, write InteractionRemoveWriter) (AuthoringMutationResult, error) {
	m = m.Normalize()
	if strings.TrimSpace(m.PageKey) == "" || strings.TrimSpace(m.ComponentKey) == "" {
		return AuthoringMutationResult{
			Message:     "Choose a component to remove an interaction from.",
			FieldErrors: map[string]string{AuthoringFieldComponentKey: "Choose a component."},
			Values:      m.FormValues(),
		}, nil
	}
	interactionKey := strings.TrimSpace(m.InteractionKey)
	if interactionKey == "" {
		return AuthoringMutationResult{
			Message:     "Choose an interaction to remove.",
			FieldErrors: map[string]string{AuthoringFieldInteractionKey: "Choose an interaction."},
			Values:      m.FormValues(),
		}, nil
	}
	if err := write(m.PageKey, m.ComponentKey, interactionKey); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Interaction removed.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "remove-interaction:" + m.ComponentKey + ":" + interactionKey,
			Label:     "Remove interaction",
			Summary:   "This element no longer has that interaction.",
			Kind:      "interaction-removed",
			PageKey:   m.PageKey,
			Component: m.ComponentKey,
		}},
	}, nil
}

// interactionFromMutation builds the candidate core.Interaction the mutation
// describes. Target reuses the mutation's page route/page key/component key —
// the same address style operations already target — mapped onto
// core.CanvasIdentity's Route/PageID/BlockKey.
func interactionFromMutation(m AuthoringMutation) core.Interaction {
	return core.Interaction{
		Key:  m.InteractionKey,
		Kind: m.InteractionKind,
		Target: core.CanvasIdentity{
			Route:    m.PageRoute,
			PageID:   m.PageKey,
			BlockKey: m.ComponentKey,
		},
		Effect:     m.InteractionEffect,
		DurationMS: m.InteractionDurationMS,
		DelayMS:    m.InteractionDelayMS,
		Once:       m.InteractionOnce,
	}.Normalize()
}

// validateSetInteractionMutation is the AuthoringMutation.Validate() case for
// set-interaction: it requires a real page+component target and delegates the
// interaction-shape checks to core.Interaction.Validate so the allow-lists
// live in exactly one place.
func validateSetInteractionMutation(validation *AuthoringValidation, mutation AuthoringMutation) {
	requirePageComponent(validation, mutation)
	interaction := interactionFromMutation(mutation)
	for field, message := range interactionFieldErrors(interaction.Validate()) {
		validation.AddFieldError(field, message)
	}
}

// interactionFieldErrors maps core.Interaction.Validate()'s field keys onto
// the authoring form fields the interactions panel renders.
func interactionFieldErrors(errs map[string]string) map[string]string {
	out := make(map[string]string, len(errs))
	for field, message := range errs {
		switch field {
		case "kind":
			out[AuthoringFieldInteractionKind] = message
		case "target":
			out[AuthoringFieldComponentKey] = message
		case "effect":
			out[AuthoringFieldInteractionEffect] = message
		case "durationMS":
			out[AuthoringFieldInteractionDurationMS] = message
		case "delayMS":
			out[AuthoringFieldInteractionDelayMS] = message
		case "key":
			out[AuthoringFieldInteractionKey] = message
		default:
			out[field] = message
		}
	}
	return out
}
