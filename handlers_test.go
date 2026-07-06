package studio

import (
	"errors"
	"testing"
)

// --- ApplySaveSectionField tests ---

func TestApplySaveSectionFieldCallsWriterCorrectly(t *testing.T) {
	var gotSection, gotField, gotValue string
	write := SectionFieldWriter(func(sectionKey, field, value string) error {
		gotSection = sectionKey
		gotField = field
		gotValue = value
		return nil
	})

	m := AuthoringMutation{
		Kind:    AuthoringOperationSaveControl,
		Binding: "home.section.pagehead.title",
		Value:   "Forest school",
	}
	result, err := ApplySaveSectionField(m, write, func(string) string { return "Title" })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FieldErrors) > 0 {
		t.Fatalf("valid section-field must not error: %v", result.FieldErrors)
	}
	if !result.RefreshPreview {
		t.Fatal("result must have RefreshPreview=true")
	}
	if gotSection != "pagehead" || gotField != "title" || gotValue != "Forest school" {
		t.Fatalf("writer called with wrong args: section=%q field=%q value=%q", gotSection, gotField, gotValue)
	}
	if result.Message != "Title saved." {
		t.Fatalf("section-field label must come from labelForField, got Message %q", result.Message)
	}
	if len(result.Changes) == 0 || result.Changes[0].Label != "Title" {
		t.Fatalf("change label must come from labelForField, got %+v", result.Changes)
	}
}

func TestApplySaveSectionFieldRejectsNonSectionBinding(t *testing.T) {
	called := false
	write := SectionFieldWriter(func(sectionKey, field, value string) error {
		called = true
		return nil
	})

	m := AuthoringMutation{
		Kind:    AuthoringOperationSaveControl,
		Binding: "home.hero.headline",
		Value:   "ignored",
	}
	result, err := ApplySaveSectionField(m, write, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("write must NOT be called for a non-section binding")
	}
	if _, ok := result.FieldErrors[AuthoringFieldBinding]; !ok {
		t.Fatalf("non-section binding must produce a field error, got: %v", result.FieldErrors)
	}
}

func TestApplySaveSectionFieldPropagatesWriterError(t *testing.T) {
	write := SectionFieldWriter(func(sectionKey, field, value string) error {
		return errors.New("db error")
	})

	m := AuthoringMutation{
		Kind:    AuthoringOperationSaveControl,
		Binding: "home.section.pagehead.title",
		Value:   "test",
	}
	// Writer errors for section-field are treated as non-fatal validation failures
	// (the host returns an error string in the field errors), not system errors.
	result, err := ApplySaveSectionField(m, write, nil)
	if err != nil {
		t.Fatalf("section-field writer errors must be surfaced as validation failures, not system errors: %v", err)
	}
	if len(result.FieldErrors) == 0 {
		t.Fatal("expected field errors when writer fails")
	}
}
