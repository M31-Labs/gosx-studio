package authoring

import (
	"context"
	"errors"
	"testing"

	"m31labs.dev/gosx-studio/core"
)

func TestApplySetFlowFieldWritesFieldWithValidationRule(t *testing.T) {
	var gotFlow, gotAction string
	var gotField core.FlowField
	write := FlowFieldWriter(func(flowKey, actionKey string, field core.FlowField) error {
		gotFlow, gotAction, gotField = flowKey, actionKey, field
		return nil
	})

	m := AuthoringMutationForSetFlowField("contact", "submit", core.FlowField{
		Name:     "email",
		Label:    "Email",
		Kind:     core.ControlText,
		Required: true,
	})
	result, err := ApplySetFlowField(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotFlow != "contact" || gotAction != "submit" {
		t.Fatalf("unexpected writer target: %q %q", gotFlow, gotAction)
	}
	if gotField.Name != "email" || !gotField.Required {
		t.Fatalf("unexpected field: %#v", gotField)
	}
}

func TestApplySetFlowFieldRejectsUnsupportedKind(t *testing.T) {
	write := FlowFieldWriter(func(flowKey, actionKey string, field core.FlowField) error {
		t.Fatal("write must not be called for an unsupported field kind")
		return nil
	})
	m := AuthoringMutationForSetFlowField("contact", "submit", core.FlowField{
		Name: "hero-image", Label: "Hero image", Kind: core.ControlMedia,
	})
	result, err := ApplySetFlowField(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors[AuthoringFieldControlKind]; !ok {
		t.Fatalf("expected a control-kind field error, got %#v", result.FieldErrors)
	}
}

func TestApplySetFlowFieldRequiresFlowAndAction(t *testing.T) {
	write := FlowFieldWriter(func(flowKey, actionKey string, field core.FlowField) error {
		t.Fatal("write must not be called without a flow/action target")
		return nil
	})
	m := AuthoringMutation{Kind: AuthoringOperationSetFlowField, FlowFieldName: "email"}
	result, err := ApplySetFlowField(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors[AuthoringFieldFlowKey]; !ok {
		t.Fatalf("expected a flow-key field error, got %#v", result.FieldErrors)
	}
	if _, ok := result.FieldErrors[AuthoringFieldFlowActionKey]; !ok {
		t.Fatalf("expected a flow-action-key field error, got %#v", result.FieldErrors)
	}
}

func TestApplyRemoveFlowField(t *testing.T) {
	var gotFlow, gotAction, gotName string
	write := FlowFieldRemoveWriter(func(flowKey, actionKey, fieldName string) error {
		gotFlow, gotAction, gotName = flowKey, actionKey, fieldName
		return nil
	})
	m := AuthoringMutationForRemoveFlowField("contact", "submit", "notes")
	result, err := ApplyRemoveFlowField(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotFlow != "contact" || gotAction != "submit" || gotName != "notes" {
		t.Fatalf("unexpected writer args: %q %q %q", gotFlow, gotAction, gotName)
	}
}

func TestApplySetFlowActionSavesLabelAndBinding(t *testing.T) {
	var gotFlow, gotAction, gotLabel, gotHandlerRef string
	write := FlowActionWriter(func(flowKey, actionKey, label, handlerRef string) error {
		gotFlow, gotAction, gotLabel, gotHandlerRef = flowKey, actionKey, label, handlerRef
		return nil
	})
	m := AuthoringMutationForSetFlowAction("contact", "submit", "Send message", "handlers.email.contact")
	result, err := ApplySetFlowAction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotFlow != "contact" || gotAction != "submit" || gotLabel != "Send message" || gotHandlerRef != "handlers.email.contact" {
		t.Fatalf("unexpected writer args: %q %q %q %q", gotFlow, gotAction, gotLabel, gotHandlerRef)
	}
}

func TestApplySetFlowActionAllowsEmptyBindingButWarns(t *testing.T) {
	write := FlowActionWriter(func(flowKey, actionKey, label, handlerRef string) error { return nil })
	m := AuthoringMutationForSetFlowAction("contact", "submit", "Send message", "")
	result, err := ApplySetFlowAction(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("an empty binding must not be a blocking mutation error: %#v", result.FieldErrors)
	}
	if result.Message == "" {
		t.Fatal("expected a message noting the handler still needs connecting")
	}
}

// --- ApplyTestFlowAction: isolated handler success and error paths ---

func contactTestAction() core.FlowAction {
	return core.FlowAction{
		Key:        "submit",
		Label:      "Send message",
		HandlerRef: "handlers.email.contact",
		Fields: []core.FlowField{
			{Name: "email", Label: "Email", Kind: core.ControlText, Required: true},
			{Name: "message", Label: "Message", Kind: core.ControlText, Required: true},
		},
	}
}

func TestApplyTestFlowActionSuccessPath(t *testing.T) {
	fakeHandler := FlowTestHandler(func(ctx context.Context, flowKey, actionKey string, values map[string]string) (FlowTestResult, error) {
		if values["email"] == "" {
			t.Fatal("expected validated values to reach the isolated handler")
		}
		return FlowTestResult{Success: true, Message: "Delivered to the sandbox inbox."}, nil
	})
	m := AuthoringMutationForTestFlowAction("contact", "submit", map[string]string{
		"email": "operator@example.com", "message": "Hello",
	})
	result, err := ApplyTestFlowAction(context.Background(), m, contactTestAction(), fakeHandler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors on success: %#v", result.FieldErrors)
	}
	if result.Message != "Delivered to the sandbox inbox." {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestApplyTestFlowActionValidationErrorPathNeverCallsHandler(t *testing.T) {
	fakeHandler := FlowTestHandler(func(ctx context.Context, flowKey, actionKey string, values map[string]string) (FlowTestResult, error) {
		t.Fatal("handler must not be called when field validation fails")
		return FlowTestResult{}, nil
	})
	m := AuthoringMutationForTestFlowAction("contact", "submit", map[string]string{
		"email": "", "message": "Hello",
	})
	result, err := ApplyTestFlowAction(context.Background(), m, contactTestAction(), fakeHandler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors["email"]; !ok {
		t.Fatalf("expected a required-field error for email, got %#v", result.FieldErrors)
	}
}

func TestApplyTestFlowActionHandlerFailurePathReportsWithoutError(t *testing.T) {
	fakeHandler := FlowTestHandler(func(ctx context.Context, flowKey, actionKey string, values map[string]string) (FlowTestResult, error) {
		return FlowTestResult{Success: false, Message: "The sandbox webhook rejected the payload."}, nil
	})
	m := AuthoringMutationForTestFlowAction("contact", "submit", map[string]string{
		"email": "operator@example.com", "message": "Hello",
	})
	result, err := ApplyTestFlowAction(context.Background(), m, contactTestAction(), fakeHandler)
	if err != nil {
		t.Fatalf("expected a handler-reported failure, not a Go error: %v", err)
	}
	if result.Message != "The sandbox webhook rejected the payload." {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestApplyTestFlowActionPropagatesInfrastructureError(t *testing.T) {
	fakeHandler := FlowTestHandler(func(ctx context.Context, flowKey, actionKey string, values map[string]string) (FlowTestResult, error) {
		return FlowTestResult{}, errors.New("sandbox unavailable")
	})
	m := AuthoringMutationForTestFlowAction("contact", "submit", map[string]string{
		"email": "operator@example.com", "message": "Hello",
	})
	if _, err := ApplyTestFlowAction(context.Background(), m, contactTestAction(), fakeHandler); err == nil {
		t.Fatal("expected an infrastructure error to propagate")
	}
}

// --- AuthoringMutationFromForm/Validate integration for the new kinds ---

func TestAuthoringMutationValidateAcceptsFlowDesignerOperationKinds(t *testing.T) {
	cases := []AuthoringMutation{
		{Kind: AuthoringOperationSetFlowField, FlowKey: "contact", FlowActionKey: "submit", FlowFieldName: "email"},
		{Kind: AuthoringOperationRemoveFlowField, FlowKey: "contact", FlowActionKey: "submit", FlowFieldName: "email"},
		{Kind: AuthoringOperationSetFlowAction, FlowKey: "contact", FlowActionKey: "submit"},
		{Kind: AuthoringOperationTestFlowAction, FlowKey: "contact", FlowActionKey: "submit"},
	}
	for _, mutation := range cases {
		if validation := mutation.Validate(); !validation.OK() {
			t.Fatalf("expected %q to validate, got errors: %#v", mutation.Kind, validation.FieldErrors)
		}
	}
}

func TestAuthoringMutationValidateRejectsIncompleteFlowDesignerOperations(t *testing.T) {
	cases := []AuthoringOperationKind{
		AuthoringOperationSetFlowField,
		AuthoringOperationRemoveFlowField,
		AuthoringOperationSetFlowAction,
		AuthoringOperationTestFlowAction,
	}
	for _, kind := range cases {
		mutation := AuthoringMutation{Kind: kind}
		if validation := mutation.Validate(); validation.OK() {
			t.Fatalf("expected %q with no target fields to fail validation", kind)
		}
	}
}
