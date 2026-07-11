package authoring

import (
	"context"
	"strings"

	"m31labs.dev/gosx-studio/core"
)

// This file extends the flow designer (panels/flow_designer.go) with durable
// editing of one action's fields/validation/binding, plus an isolated
// test-execution path. It follows instance_handlers.go's shape: Studio
// re-validates the mutation and delegates the actual write to a
// host-supplied closure. Flow field/action storage stays entirely host-owned
// (typically cms/flows.Document via cms/studio.SaveConfiguredFlowDraft) —
// this file only ever talks in terms of core.FlowField/core.ControlKind, not
// any particular host's document shape.

// FlowFieldWriter persists one field (add or update) on a flow action,
// identified by flowKey+actionKey. field.Required doubles as that field's
// validation rule (core.FlowAction.ValidatePayload).
type FlowFieldWriter func(flowKey, actionKey string, field core.FlowField) error

// FlowFieldRemoveWriter removes one field from a flow action.
type FlowFieldRemoveWriter func(flowKey, actionKey, fieldName string) error

// FlowActionWriter updates a flow action's label and/or its submit binding
// (the connected handler ref). An empty handlerRef is a legal, in-progress
// state — readiness (not this writer) blocks publish until one is connected.
type FlowActionWriter func(flowKey, actionKey, label, handlerRef string) error

// FlowTestResult reports one isolated test-execution outcome: either the
// handler ran and reports Success, or it reports field-level problems without
// ever reaching a production side effect (email/webhook/order).
type FlowTestResult struct {
	Success     bool
	Message     string
	FieldErrors map[string]string
}

// FlowTestHandler executes one flow action against submitted test values in
// isolation — no email/webhook/order side effects — used by the flow
// designer's "Test" affordance so an operator can verify field validation and
// the submit binding before publish. Hosts implement it (typically a thin
// dry-run wrapper around the real cms/flows.Handler); Studio never calls a
// production handler directly from a durable mutation.
type FlowTestHandler func(ctx context.Context, flowKey, actionKey string, values map[string]string) (FlowTestResult, error)

// flowFieldControlKinds is the allow-list of core.ControlKind values that
// make sense as a flow submission field. Rich text / media / color / a
// nested flow reference / a 3D scene are legitimate content control kinds
// elsewhere in Studio but are not sensible primitive form-field inputs, so
// they are excluded here even though core.NormalizeControlKind accepts them.
var flowFieldControlKinds = map[core.ControlKind]struct{}{
	core.ControlText:   {},
	core.ControlChoice: {},
	core.ControlToggle: {},
	core.ControlNumber: {},
	core.ControlLink:   {},
}

// FlowFieldControlKinds returns the allow-listed field kinds a flow
// submission field may use, in a stable display order.
func FlowFieldControlKinds() []core.ControlKind {
	return []core.ControlKind{core.ControlText, core.ControlChoice, core.ControlToggle, core.ControlNumber, core.ControlLink}
}

func isFlowFieldControlKind(kind core.ControlKind) bool {
	_, ok := flowFieldControlKinds[kind]
	return ok
}

// ApplySetFlowField is the host-agnostic body for the set-flow-field
// authoring operation.
func ApplySetFlowField(m AuthoringMutation, write FlowFieldWriter) (AuthoringMutationResult, error) {
	m = m.Normalize()
	if errs := validateFlowFieldMutation(m); len(errs) > 0 {
		return AuthoringMutationResult{
			Message:     "Choose a supported field kind for this flow action.",
			FieldErrors: errs,
			Values:      m.FormValues(),
		}, nil
	}
	field := core.FlowField{
		Name:     m.FlowFieldName,
		Label:    m.FlowFieldLabel,
		Kind:     m.ControlKind,
		Required: m.FlowFieldRequired,
	}
	if strings.TrimSpace(field.Label) == "" {
		field.Label = field.Name
	}
	if err := write(m.FlowKey, m.FlowActionKey, field); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Field saved.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:     "flow-field:" + m.FlowKey + ":" + m.FlowActionKey + ":" + field.Name,
			Label:   field.Label,
			Summary: "Updated the " + field.Label + " field on " + m.FlowActionKey + ".",
			Kind:    "flow-field",
			PageKey: m.FlowKey,
		}},
	}, nil
}

// ApplyRemoveFlowField is the host-agnostic body for the remove-flow-field
// authoring operation.
func ApplyRemoveFlowField(m AuthoringMutation, write FlowFieldRemoveWriter) (AuthoringMutationResult, error) {
	m = m.Normalize()
	errs := map[string]string{}
	requireFlowAction(&AuthoringValidation{FieldErrors: errs}, m)
	if m.FlowFieldName == "" {
		errs[AuthoringFieldFlowFieldName] = "Choose a field."
	}
	if len(errs) > 0 {
		return AuthoringMutationResult{Message: "Choose a field to remove.", FieldErrors: errs, Values: m.FormValues()}, nil
	}
	if err := write(m.FlowKey, m.FlowActionKey, m.FlowFieldName); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Field removed.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:     "remove-flow-field:" + m.FlowKey + ":" + m.FlowActionKey + ":" + m.FlowFieldName,
			Label:   "Remove field",
			Summary: "Removed " + m.FlowFieldName + " from " + m.FlowActionKey + ".",
			Kind:    "flow-field-removed",
			PageKey: m.FlowKey,
		}},
	}, nil
}

// ApplySetFlowAction is the host-agnostic body for the set-flow-action
// authoring operation. mutation.Binding carries the submit binding (handler
// ref); an empty binding is accepted here (readiness blocks publish, not this
// mutation) so an operator can build the form before connecting a handler.
func ApplySetFlowAction(m AuthoringMutation, write FlowActionWriter) (AuthoringMutationResult, error) {
	m = m.Normalize()
	errs := map[string]string{}
	requireFlowAction(&AuthoringValidation{FieldErrors: errs}, m)
	if len(errs) > 0 {
		return AuthoringMutationResult{Message: "Choose a flow action.", FieldErrors: errs, Values: m.FormValues()}, nil
	}
	if err := write(m.FlowKey, m.FlowActionKey, m.FlowActionLabel, m.Binding); err != nil {
		return AuthoringMutationResult{}, err
	}
	message := "Flow action saved."
	if strings.TrimSpace(m.Binding) == "" {
		message = "Flow action saved. Connect a handler before publishing."
	}
	return AuthoringMutationResult{
		Message:        message,
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:     "flow-action:" + m.FlowKey + ":" + m.FlowActionKey,
			Label:   "Flow action",
			Summary: "Updated " + m.FlowActionKey + "'s label and submit binding.",
			Kind:    "flow-action",
			PageKey: m.FlowKey,
			Binding: m.Binding,
		}},
	}, nil
}

// ApplyTestFlowAction is the isolated test-execution path: it validates the
// submitted test values against action's own field rules (the same rules a
// real submission would use — core.FlowAction.ValidatePayload) and, only if
// those pass, calls run. run must never reach a production side effect; both
// its success and its intentional-failure result are surfaced to the
// operator without a Go error, exactly like a real submission would surface a
// field or handler failure.
func ApplyTestFlowAction(ctx context.Context, m AuthoringMutation, action core.FlowAction, run FlowTestHandler) (AuthoringMutationResult, error) {
	m = m.Normalize()
	errs := map[string]string{}
	requireFlowAction(&AuthoringValidation{FieldErrors: errs}, m)
	if len(errs) > 0 {
		return AuthoringMutationResult{Message: "Choose a flow action to test.", FieldErrors: errs, Values: m.FormValues()}, nil
	}
	values := m.RawForm
	if values == nil {
		values = map[string]string{}
	}
	if fieldErrors := action.ValidatePayload(values); len(fieldErrors) > 0 {
		return AuthoringMutationResult{
			Message:     "Fix the highlighted fields before testing this flow.",
			FieldErrors: fieldErrors,
			Values:      m.FormValues(),
		}, nil
	}
	if run == nil {
		return AuthoringMutationResult{Message: "No test handler is configured for this flow."}, nil
	}
	result, err := run(ctx, m.FlowKey, m.FlowActionKey, values)
	if err != nil {
		return AuthoringMutationResult{}, err
	}
	if !result.Success {
		return AuthoringMutationResult{
			Message:     firstNonEmpty(result.Message, "The test submission did not succeed."),
			FieldErrors: result.FieldErrors,
			Values:      m.FormValues(),
		}, nil
	}
	return AuthoringMutationResult{
		Message: firstNonEmpty(result.Message, "Test submission succeeded."),
		Values:  m.FormValues(),
		Changes: []AuthoringChange{{
			Key:     "flow-test:" + m.FlowKey + ":" + m.FlowActionKey,
			Label:   "Test submission",
			Summary: "Ran " + m.FlowActionKey + " against test values in isolation.",
			Kind:    "flow-test",
			PageKey: m.FlowKey,
		}},
	}, nil
}

// validateFlowFieldMutation validates the set-flow-field mutation shape:
// flow+action target, a non-empty field name, and a field kind in the
// flow-field allow-list.
func validateFlowFieldMutation(m AuthoringMutation) map[string]string {
	errs := map[string]string{}
	requireFlowAction(&AuthoringValidation{FieldErrors: errs}, m)
	if m.FlowFieldName == "" {
		errs[AuthoringFieldFlowFieldName] = "Choose a field name."
	}
	if m.ControlKind != "" && !isFlowFieldControlKind(m.ControlKind) {
		errs[AuthoringFieldControlKind] = "Choose a supported field kind."
	}
	return errs
}

// validateSetFlowFieldMutation is the AuthoringMutation.Validate() case for
// set-flow-field.
func validateSetFlowFieldMutation(validation *AuthoringValidation, mutation AuthoringMutation) {
	for field, message := range validateFlowFieldMutation(mutation) {
		validation.AddFieldError(field, message)
	}
}
