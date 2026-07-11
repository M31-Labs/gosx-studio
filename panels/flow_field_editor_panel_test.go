package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
)

func TestRenderFlowFieldEditorPanelExposesFieldActionBindingAndTestForms(t *testing.T) {
	node := RenderFlowFieldEditorPanel(FlowFieldEditorOptions{
		ID: "flow-fields", Action: "/authoring", CSRFToken: "token",
		FlowKey: "contact", FlowLabel: "Contact", ActionKey: "submit", ActionLabel: "Send message",
		Binding: "handlers.email.contact", BindingStatusLabel: "Connected",
		Fields: []FlowFieldRow{
			{Name: "email", Label: "Email", Kind: "text", Required: true},
		},
		TestValues: []FlowTestFieldValue{
			{Name: "email", Label: "Email", Value: "operator@example.com"},
		},
		TestMessage: "Test submission succeeded.",
	})
	html := gosx.RenderHTML(node)

	for _, want := range []string{
		"data-studio-flow-field-editor=\"true\"",
		"data-studio-flow-field-editor-flow=\"contact\"",
		"data-studio-flow-field-editor-action=\"submit\"",
		string(authoring.AuthoringOperationSetFlowAction),
		string(authoring.AuthoringOperationSetFlowField),
		string(authoring.AuthoringOperationRemoveFlowField),
		string(authoring.AuthoringOperationTestFlowAction),
		"data-studio-flow-binding-input=\"true\"",
		"handlers.email.contact",
		"data-studio-flow-field-row=\"email\"",
		"data-studio-flow-add-field-form=\"true\"",
		"data-studio-flow-test-form=\"submit\"",
		"Test submission succeeded.",
		"csrf_token",
		"data-gosx-studio-authoring-managed=\"true\"",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("flow field editor panel missing %q in:\n%s", want, html)
		}
	}

	if strings.Count(html, "<form") < 4 {
		t.Fatalf("expected at least 4 self-contained forms (action, field save, field remove, add-field, test), got:\n%s", html)
	}
}

func TestRenderFlowFieldEditorPanelSurfacesTestFieldErrors(t *testing.T) {
	node := RenderFlowFieldEditorPanel(FlowFieldEditorOptions{
		FlowKey: "contact", ActionKey: "submit",
		TestValues: []FlowTestFieldValue{
			{Name: "email", Label: "Email", Value: "", Error: "This field is required."},
		},
	})
	html := gosx.RenderHTML(node)
	if !strings.Contains(html, "data-studio-flow-test-field-error=\"email\"") {
		t.Fatalf("expected a field-level test error marker, got:\n%s", html)
	}
	if !strings.Contains(html, "This field is required.") {
		t.Fatal("expected the field error message to render")
	}
}

func TestRenderFlowFieldEditorPanelBindingReasonRendersWhenUnresolved(t *testing.T) {
	node := RenderFlowFieldEditorPanel(FlowFieldEditorOptions{
		FlowKey: "contact", ActionKey: "submit",
		BindingStatusLabel: "Needs setup",
		BindingReason:      "Connect a handler before this flow can accept submissions.",
	})
	html := gosx.RenderHTML(node)
	if !strings.Contains(html, "data-studio-flow-binding-reason=\"true\"") {
		t.Fatal("expected a binding-reason element when the binding is unresolved")
	}
}
