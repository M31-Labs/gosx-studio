package authoring

import (
	"errors"
	"testing"

	"m31labs.dev/gosx-studio/core"
)

// --- ApplySetSharedField tests ---

func TestApplySetSharedFieldWritesDefinitionAndFansOutByConstruction(t *testing.T) {
	var gotDefinitionKey, gotControlKey, gotValue string
	write := SharedFieldWriter(func(definitionKey, controlKey, value string) error {
		gotDefinitionKey, gotControlKey, gotValue = definitionKey, controlKey, value
		return nil
	})

	m := AuthoringMutationForSharedField("testimonial", core.Control{Key: "quote", Value: "Updated quote"})
	result, err := ApplySetSharedField(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotDefinitionKey != "testimonial" || gotControlKey != "quote" || gotValue != "Updated quote" {
		t.Fatalf("unexpected writer args: definition=%q control=%q value=%q", gotDefinitionKey, gotControlKey, gotValue)
	}
	if !result.RefreshPreview {
		t.Fatal("expected RefreshPreview so both instances re-render")
	}
	if len(result.Changes) != 1 || result.Changes[0].Kind != "shared-field" {
		t.Fatalf("unexpected changes: %#v", result.Changes)
	}
}

func TestApplySetSharedFieldRejectsMissingDefinition(t *testing.T) {
	called := false
	write := SharedFieldWriter(func(definitionKey, controlKey, value string) error {
		called = true
		return nil
	})
	m := AuthoringMutation{Kind: AuthoringOperationSetSharedField, ControlKey: "quote", Value: "x"}
	result, err := ApplySetSharedField(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("write must not be called without a definition key")
	}
	if _, ok := result.FieldErrors[AuthoringFieldComponentTemplateKey]; !ok {
		t.Fatalf("expected a field error for the missing definition key, got %#v", result.FieldErrors)
	}
}

func TestApplySetSharedFieldPropagatesWriterError(t *testing.T) {
	write := SharedFieldWriter(func(definitionKey, controlKey, value string) error {
		return errors.New("store unavailable")
	})
	m := AuthoringMutationForSharedField("testimonial", core.Control{Key: "quote", Value: "x"})
	if _, err := ApplySetSharedField(m, write); err == nil {
		t.Fatal("expected the writer's error to propagate")
	}
}

// --- ApplyOverrideInstance tests ---

func TestApplyOverrideInstanceWritesOnlyThatInstance(t *testing.T) {
	var gotPage, gotComponent, gotControl, gotValue string
	write := InstanceOverrideWriter(func(pageKey, componentKey, controlKey, value string) error {
		gotPage, gotComponent, gotControl, gotValue = pageKey, componentKey, controlKey, value
		return nil
	})
	page := core.Page{Key: "home", Label: "Home", Route: "/", GoSXComponent: "HomePage"}
	component := core.Component{Key: "testimonial-1", Label: "Testimonial", GoSXComponent: "Testimonial", DefinitionKey: "testimonial"}
	m := AuthoringMutationForInstanceOverride(page, component, core.Control{Key: "quote", Value: "Only mine"})

	result, err := ApplyOverrideInstance(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasErrors() {
		t.Fatalf("unexpected field errors: %#v", result.FieldErrors)
	}
	if gotPage != "home" || gotComponent != "testimonial-1" || gotControl != "quote" || gotValue != "Only mine" {
		t.Fatalf("unexpected writer args: %q %q %q %q", gotPage, gotComponent, gotControl, gotValue)
	}
}

func TestApplyOverrideInstanceRequiresPageComponentAndControl(t *testing.T) {
	write := InstanceOverrideWriter(func(pageKey, componentKey, controlKey, value string) error {
		t.Fatal("write must not be called")
		return nil
	})
	if _, err := ApplyOverrideInstance(AuthoringMutation{Kind: AuthoringOperationOverrideInstance}, write); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result, err := ApplyOverrideInstance(AuthoringMutation{Kind: AuthoringOperationOverrideInstance, PageKey: "home", ComponentKey: "testimonial-1"}, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors[AuthoringFieldControlKey]; !ok {
		t.Fatalf("expected a control-key field error, got %#v", result.FieldErrors)
	}
}

// --- ApplyDetachInstance / ApplyRestoreInstance tests ---

func TestApplyDetachInstanceCallsWriterAndReportsIndependence(t *testing.T) {
	var gotPage, gotComponent string
	write := InstanceDetachWriter(func(pageKey, componentKey string) error {
		gotPage, gotComponent = pageKey, componentKey
		return nil
	})
	page := core.Page{Key: "home", Label: "Home", Route: "/", GoSXComponent: "HomePage"}
	component := core.Component{Key: "testimonial-1", Label: "Testimonial", GoSXComponent: "Testimonial", DefinitionKey: "testimonial"}
	m := AuthoringMutationForInstanceDetach(page, component)

	result, err := ApplyDetachInstance(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != "home" || gotComponent != "testimonial-1" {
		t.Fatalf("unexpected writer args: %q %q", gotPage, gotComponent)
	}
	if len(result.Changes) != 1 || result.Changes[0].Kind != "instance-detach" {
		t.Fatalf("unexpected changes: %#v", result.Changes)
	}
}

func TestApplyRestoreInstanceCallsWriterAndReportsReattachment(t *testing.T) {
	var gotPage, gotComponent string
	write := InstanceRestoreWriter(func(pageKey, componentKey string) error {
		gotPage, gotComponent = pageKey, componentKey
		return nil
	})
	page := core.Page{Key: "home", Label: "Home", Route: "/", GoSXComponent: "HomePage"}
	component := core.Component{Key: "testimonial-1", Label: "Testimonial", GoSXComponent: "Testimonial", DefinitionKey: "testimonial", Detached: true}
	m := AuthoringMutationForInstanceRestore(page, component)

	result, err := ApplyRestoreInstance(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPage != "home" || gotComponent != "testimonial-1" {
		t.Fatalf("unexpected writer args: %q %q", gotPage, gotComponent)
	}
	if len(result.Changes) != 1 || result.Changes[0].Kind != "instance-restore" {
		t.Fatalf("unexpected changes: %#v", result.Changes)
	}
}

func TestApplyDetachAndRestoreInstanceRequirePageAndComponent(t *testing.T) {
	detachWrite := InstanceDetachWriter(func(pageKey, componentKey string) error {
		t.Fatal("detach write must not be called")
		return nil
	})
	if result, err := ApplyDetachInstance(AuthoringMutation{Kind: AuthoringOperationDetachInstance}, detachWrite); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, ok := result.FieldErrors[AuthoringFieldComponentKey]; !ok {
		t.Fatalf("expected a component-key field error, got %#v", result.FieldErrors)
	}

	restoreWrite := InstanceRestoreWriter(func(pageKey, componentKey string) error {
		t.Fatal("restore write must not be called")
		return nil
	})
	if result, err := ApplyRestoreInstance(AuthoringMutation{Kind: AuthoringOperationRestoreInstance}, restoreWrite); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if _, ok := result.FieldErrors[AuthoringFieldComponentKey]; !ok {
		t.Fatalf("expected a component-key field error, got %#v", result.FieldErrors)
	}
}

// --- AuthoringMutationFromForm/Validate integration for the new kinds ---

func TestAuthoringMutationValidateAcceptsNewInstanceOperationKinds(t *testing.T) {
	cases := []AuthoringMutation{
		{Kind: AuthoringOperationSetSharedField, ComponentTemplateKey: "testimonial", ControlKey: "quote"},
		{Kind: AuthoringOperationOverrideInstance, PageKey: "home", ComponentKey: "testimonial-1", ControlKey: "quote"},
		{Kind: AuthoringOperationDetachInstance, PageKey: "home", ComponentKey: "testimonial-1"},
		{Kind: AuthoringOperationRestoreInstance, PageKey: "home", ComponentKey: "testimonial-1"},
	}
	for _, mutation := range cases {
		if validation := mutation.Validate(); !validation.OK() {
			t.Fatalf("expected %q to validate, got errors: %#v", mutation.Kind, validation.FieldErrors)
		}
	}
}

func TestAuthoringMutationValidateRejectsIncompleteInstanceOperations(t *testing.T) {
	cases := []AuthoringOperationKind{
		AuthoringOperationSetSharedField,
		AuthoringOperationOverrideInstance,
		AuthoringOperationDetachInstance,
		AuthoringOperationRestoreInstance,
	}
	for _, kind := range cases {
		mutation := AuthoringMutation{Kind: kind}
		if validation := mutation.Validate(); validation.OK() {
			t.Fatalf("expected %q with no target fields to fail validation", kind)
		}
	}
}
