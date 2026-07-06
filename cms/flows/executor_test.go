package flows

import (
	"context"
	"errors"
	"testing"
	"time"

	"m31labs.dev/gosx-admin/workbench"
)

func TestResolveExecutableFlowPrefersPublication(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	document := DocumentFromDefinition(ScheduleTour("draft.submit"), DocumentOptions{ID: "tour-flow"})
	published := document
	published.Actions[0].HandlerRef = "tour.submit"
	publication, err := NewPublication(published, "publisher", "rev_1", now)
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemoryStore(document)
	if err := store.SaveFlowPublication(publication); err != nil {
		t.Fatal(err)
	}

	flow, ok := ResolveExecutableFlow(store, store, "tour-flow", true)
	if !ok {
		t.Fatal("expected executable flow")
	}
	if flow.Source != ExecutableSourcePublication || flow.RevisionID != "rev_1" {
		t.Fatalf("expected publication source, got %#v", flow)
	}
	if flow.Document.Actions[0].HandlerRef != "tour.submit" {
		t.Fatalf("expected published document handler, got %#v", flow.Document.Actions[0])
	}
}

func TestResolveExecutableFlowDocumentFallbackIsExplicit(t *testing.T) {
	store := NewMemoryStore(DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"}))
	if _, ok := ResolveExecutableFlow(store, store, "tour-flow", false); ok {
		t.Fatal("expected unpublished document to require explicit fallback")
	}
	flow, ok := ResolveExecutableFlow(store, store, "tour-flow", true)
	if !ok {
		t.Fatal("expected document fallback")
	}
	if flow.Source != ExecutableSourceDocument || flow.Document.Key != FlowKeyScheduleTour {
		t.Fatalf("unexpected fallback flow: %#v", flow)
	}
}

func TestResolveExecutableFlowDraftFallbackIsExplicit(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	document := DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"})
	draft := NewDraft(document, "author", "rev_base", now)
	draft.Document.Actions[0].HandlerRef = "tour.draft"
	store := NewMemoryStore()
	if err := store.SaveFlowDraft(draft); err != nil {
		t.Fatal(err)
	}

	if _, ok := ResolveExecutableFlowVersion(store, store, nil, "tour-flow", false, false); ok {
		t.Fatal("expected draft to require explicit fallback")
	}
	flow, ok := ResolveExecutableFlowVersion(store, store, nil, "tour-flow", true, false)
	if !ok {
		t.Fatal("expected draft fallback")
	}
	if flow.Source != ExecutableSourceDraft || flow.RevisionID != "rev_base" || flow.Document.Actions[0].HandlerRef != "tour.draft" {
		t.Fatalf("unexpected draft fallback: %#v", flow)
	}
}

func TestValidateActionPayloadReturnsRequiredFieldErrors(t *testing.T) {
	action := DocumentAction{
		Key:        "submit",
		Label:      "Submit",
		HandlerRef: "contact.submit",
		Fields: []workbench.Field{
			{Name: "name", Label: "Name", Required: true},
			{Name: "email", Label: "Email", Required: true},
			{Name: "notes", Label: "Notes"},
		},
	}
	errs := ValidateActionPayload(action, map[string]string{"name": "Ada", "email": " "})
	if errs["email"] == "" || errs["name"] != "" || errs["notes"] != "" {
		t.Fatalf("unexpected validation errors: %#v", errs)
	}
}

func TestValidateActionPayloadChecksEmailFieldShape(t *testing.T) {
	action := DocumentAction{
		Key:        "submit",
		Label:      "Submit",
		HandlerRef: "contact.submit",
		Fields: []workbench.Field{
			{Name: "email", Label: "Email", Required: true},
			{Name: "guardianEmail", Label: "Guardian email"},
		},
	}
	errs := ValidateActionPayload(action, map[string]string{"email": "ada", "guardianEmail": "family"})
	if errs["email"] == "" || errs["guardianEmail"] == "" {
		t.Fatalf("expected email validation errors, got %#v", errs)
	}
	errs = ValidateActionPayload(action, map[string]string{"email": "ada@example.com", "guardianEmail": "family@example.com"})
	if len(errs) != 0 {
		t.Fatalf("expected valid email fields, got %#v", errs)
	}
}

func TestFindDocumentActionNormalizesLookup(t *testing.T) {
	document := DocumentFromDefinition(CheckoutHandoff("checkout.continue"), DocumentOptions{ID: "checkout-flow"})
	action, ok := FindDocumentAction(document, " Continue ")
	if !ok {
		t.Fatal("expected action lookup")
	}
	if action.Key != "continue" || action.HandlerRef != "checkout.continue" {
		t.Fatalf("unexpected action: %#v", action)
	}
}

func TestDocumentCanExecuteRequiresValidDocumentAndHandlerRefs(t *testing.T) {
	ready := DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"})
	missingHandler := DocumentFromDefinition(ScheduleTour(""), DocumentOptions{ID: "tour-flow"})
	noActions := DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"})
	noActions.Actions = nil
	if !DocumentCanExecute(ready) {
		t.Fatalf("expected ready document to execute: %#v", ready)
	}
	if DocumentCanExecute(missingHandler) {
		t.Fatalf("expected missing handler to be non-executable")
	}
	if DocumentCanExecute(noActions) {
		t.Fatalf("expected document with no actions to be non-executable")
	}
	if got := ExecutableDocumentCount([]Document{ready, missingHandler, noActions}); got != 1 {
		t.Fatalf("expected one executable document, got %d", got)
	}
}

func TestExecuteFlowReturnsUnknownHandler(t *testing.T) {
	store := NewMemoryStore(DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"}))
	result, err := ExecuteFlow(context.Background(), Executor{Documents: store, Publications: store, Registry: NewRegistry()}, Submission{
		DocumentID:          "tour-flow",
		ActionKey:           "submit",
		UseDocumentFallback: true,
		Values: map[string]string{
			"guardianName": "Ada Lovelace",
			"email":        "ada@example.com",
		},
	})
	if !errors.Is(err, ErrHandlerNotFound) {
		t.Fatalf("expected unknown handler error, got result=%#v err=%v", result, err)
	}
	if result.HandlerRef != "tour.submit" {
		t.Fatalf("expected handler ref in result, got %#v", result)
	}
}

func TestExecuteFlowDispatchesHandler(t *testing.T) {
	store := NewMemoryStore(DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"}))
	registry := NewRegistry()
	var got Submission
	registry.Register("tour.submit", func(ctx context.Context, submission Submission) (any, error) {
		got = submission
		return map[string]any{"queued": true, "email": submission.Values["email"]}, nil
	})

	result, err := ExecuteFlow(context.Background(), Executor{Documents: store, Publications: store, Registry: registry}, Submission{
		FlowKey:             FlowKeyScheduleTour,
		ActionKey:           "submit",
		UseDocumentFallback: true,
		Values: map[string]string{
			"guardianName": "Ada Lovelace",
			"email":        "ada@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.FlowKey != FlowKeyScheduleTour || got.ActionKey != "submit" || got.Values["email"] != "ada@example.com" {
		t.Fatalf("unexpected handler submission: %#v", got)
	}
	output, ok := result.Output.(map[string]any)
	if !ok || output["queued"] != true || output["email"] != "ada@example.com" {
		t.Fatalf("unexpected handler output: %#v", result.Output)
	}
	if len(result.FieldErrors) != 0 || result.Flow.Document.Key != FlowKeyScheduleTour || result.Action.Key != "submit" {
		t.Fatalf("unexpected execution result: %#v", result)
	}
}

func TestExecuteFlowValidationSkipsHandler(t *testing.T) {
	store := NewMemoryStore(DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"}))
	registry := NewRegistry()
	called := false
	registry.Register("tour.submit", func(ctx context.Context, submission Submission) (any, error) {
		called = true
		return nil, nil
	})

	result, err := ExecuteFlow(context.Background(), Executor{Documents: store, Publications: store, Registry: registry}, Submission{
		DocumentID:          "tour-flow",
		ActionKey:           "submit",
		UseDocumentFallback: true,
		Values:              map[string]string{"guardianName": "Ada Lovelace"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("expected validation failure to skip handler")
	}
	if result.FieldErrors["email"] == "" {
		t.Fatalf("expected required email field error, got %#v", result.FieldErrors)
	}
}
