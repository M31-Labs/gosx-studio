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

// TestRenderFlowFieldEditorPanelCollapsesIntoOneCompactCardWithDisclosure
// guards wave 3B's ADVANCED-discipline fix: the action-binding/field-rows/
// add-field/test forms must sit behind a closed-by-default <details>, with
// only a compact label + field-count summary visible outside it — the same
// disclosure shape wave 3A already proved for
// panels/interactions_panel.go's renderInteractionsTargetCard.
func TestRenderFlowFieldEditorPanelCollapsesIntoOneCompactCardWithDisclosure(t *testing.T) {
	node := RenderFlowFieldEditorPanel(FlowFieldEditorOptions{
		FlowKey: "contact", FlowLabel: "Contact", ActionKey: "submit", ActionLabel: "Send message",
		Fields: []FlowFieldRow{
			{Name: "email", Label: "Email", Kind: "text", Required: true},
			{Name: "name", Label: "Name", Kind: "text"},
		},
	})
	html := gosx.RenderHTML(node)
	for _, want := range []string{
		`class="studio-flow-field-editor"`,
		`data-studio-flow-field-editor-label="true"`,
		"Send message · Contact",
		`data-studio-flow-field-editor-summary="true"`,
		"2 fields · 1 required",
		`<details class="studio-flow-field-editor__editor">`,
		"Edit fields, validation, and binding",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("flow field editor compact card missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, `<details class="studio-flow-field-editor__editor" open`) {
		t.Fatalf("flow field editor disclosure must be closed by default:\n%s", html)
	}
}

// TestRenderFlowFieldEditorInspectorRendersOneHeaderAndCollapsedCardsPerAction
// guards the dedupe half of the same fix: a host composing every flow
// action's editor together (muddy-noni-commerce's editorFlowFieldEditorPanels
// loops every flow's every action) must get exactly one "Fields, validation,
// and binding" header, not one per action — the magnolia-3 re-score's "4
// duplicate 'Fields, validation, and binding' panels" ADVANCED finding.
func TestRenderFlowFieldEditorInspectorRendersOneHeaderAndCollapsedCardsPerAction(t *testing.T) {
	html := gosx.RenderHTML(RenderFlowFieldEditorInspector(FlowFieldEditorInspectorOptions{
		Actions: []FlowFieldEditorOptions{
			{FlowKey: "contact", FlowLabel: "Contact", ActionKey: "submit", ActionLabel: "Send message", Fields: []FlowFieldRow{{Name: "email", Label: "Email", Kind: "text"}}},
			{FlowKey: "newsletter", FlowLabel: "Newsletter", ActionKey: "submit", ActionLabel: "Subscribe", Fields: []FlowFieldRow{{Name: "email", Label: "Email", Kind: "text"}}},
			{FlowKey: "purchase-request", FlowLabel: "Purchase request", ActionKey: "submit", ActionLabel: "Send request", Fields: []FlowFieldRow{{Name: "email", Label: "Email", Kind: "text"}}},
			{FlowKey: "checkout-handoff", FlowLabel: "Checkout handoff", ActionKey: "continue", ActionLabel: "Continue", Fields: []FlowFieldRow{{Name: "email", Label: "Email", Kind: "text"}}},
		},
	}))
	if got := strings.Count(html, "Fields, validation, and binding"); got != 1 {
		t.Fatalf("expected exactly one \"Fields, validation, and binding\" header, got %d:\n%s", got, html)
	}
	if got := strings.Count(html, `class="studio-flow-field-editor"`); got != 4 {
		t.Fatalf("expected 4 compact action cards, got %d:\n%s", got, html)
	}
	if strings.Contains(html, `<details class="studio-flow-field-editor__editor" open`) {
		t.Fatalf("no action's disclosure should be pre-expanded:\n%s", html)
	}
	for _, want := range []string{"Send message · Contact", "Subscribe · Newsletter", "Send request · Purchase request", "Continue · Checkout handoff"} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing action label %q in:\n%s", want, html)
		}
	}
}

// TestRenderFlowFieldEditorInspectorEmptyStateWhenNoActions guards the
// no-crash empty path (mirrors RenderInteractionsInspector's own empty
// state).
func TestRenderFlowFieldEditorInspectorEmptyStateWhenNoActions(t *testing.T) {
	html := gosx.RenderHTML(RenderFlowFieldEditorInspector(FlowFieldEditorInspectorOptions{}))
	if !strings.Contains(html, "No form flow actions yet.") {
		t.Fatalf("expected an empty-state message, got:\n%s", html)
	}
}
