package studio

import (
	"errors"
	"testing"
)

// --- ApplySetStyle tests ---

func TestApplySetStyleRejectsUnsupportedProperty(t *testing.T) {
	called := false
	write := StyleDraftWriter(func(componentKey, property, value, breakpoint, state string) error {
		called = true
		return nil
	})

	m := AuthoringMutation{
		Kind:          AuthoringOperationSetStyle,
		PageKey:       "home",
		ComponentKey:  "hero",
		StyleProperty: "behavior",
		StyleValue:    "url(x)",
	}
	result, err := ApplySetStyle(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("write must NOT be called for an unsupported property")
	}
	if _, ok := result.FieldErrors[AuthoringFieldStyleProperty]; !ok {
		t.Fatalf("unsupported property must produce a field error, got: %v", result.FieldErrors)
	}
}

func TestApplySetStyleRejectsDangerousValue(t *testing.T) {
	called := false
	write := StyleDraftWriter(func(componentKey, property, value, breakpoint, state string) error {
		called = true
		return nil
	})

	m := AuthoringMutation{
		Kind:          AuthoringOperationSetStyle,
		PageKey:       "home",
		ComponentKey:  "hero",
		StyleProperty: "color",
		StyleValue:    "red; position:fixed",
	}
	result, err := ApplySetStyle(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("write must NOT be called for a dangerous value")
	}
	if _, ok := result.FieldErrors[AuthoringFieldStyleValue]; !ok {
		t.Fatalf("dangerous value must produce a field error, got: %v", result.FieldErrors)
	}
}

func TestApplySetStyleRejectsMissingComponentKey(t *testing.T) {
	called := false
	write := StyleDraftWriter(func(componentKey, property, value, breakpoint, state string) error {
		called = true
		return nil
	})

	m := AuthoringMutation{
		Kind:          AuthoringOperationSetStyle,
		PageKey:       "home",
		ComponentKey:  "",
		StyleProperty: "color",
		StyleValue:    "#ff0000",
	}
	result, err := ApplySetStyle(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("write must NOT be called when componentKey is missing")
	}
	if _, ok := result.FieldErrors[AuthoringFieldComponentKey]; !ok {
		t.Fatalf("missing componentKey must produce a field error, got: %v", result.FieldErrors)
	}
}

func TestApplySetStyleCallsWriterWithCorrectArgs(t *testing.T) {
	var gotKey, gotProp, gotVal, gotBP, gotState string
	write := StyleDraftWriter(func(componentKey, property, value, breakpoint, state string) error {
		gotKey = componentKey
		gotProp = property
		gotVal = value
		gotBP = breakpoint
		gotState = state
		return nil
	})

	page, component := Page{Key: "home", Label: "Home", Route: "/"}, Component{Key: "hero", Label: "Hero"}
	m := AuthoringMutationForStyle(page, component, "padding-block", "96px", StyleBreakpointTablet, StyleStateHover)
	result, err := ApplySetStyle(m, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FieldErrors) > 0 {
		t.Fatalf("valid set-style must not error: %v", result.FieldErrors)
	}
	if !result.RefreshPreview {
		t.Fatal("result must have RefreshPreview=true")
	}
	if gotKey != "hero" || gotProp != "padding-block" || gotVal != "96px" ||
		gotBP != StyleBreakpointTablet || gotState != StyleStateHover {
		t.Fatalf("writer called with wrong args: key=%q prop=%q val=%q bp=%q state=%q",
			gotKey, gotProp, gotVal, gotBP, gotState)
	}
}

func TestApplySetStylePropagatesWriterError(t *testing.T) {
	writeErr := errors.New("storage failure")
	write := StyleDraftWriter(func(componentKey, property, value, breakpoint, state string) error {
		return writeErr
	})

	page, component := Page{Key: "home"}, Component{Key: "hero"}
	m := AuthoringMutationForStyle(page, component, "color", "#000", "", "")
	_, err := ApplySetStyle(m, write)
	if err == nil {
		t.Fatal("writer error must bubble up")
	}
}

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

// --- ApplySaveAppearance tests ---

func TestApplySaveAppearanceFiltersToAllowedKeys(t *testing.T) {
	var gotForm map[string]string
	write := AppearanceWriter(func(rawForm map[string]string) (map[string]string, error) {
		gotForm = rawForm
		return nil, nil
	})

	allowedKeys := []string{"colorCanvas", "colorAccent", "displayFont"}
	m := AuthoringMutation{
		Kind: AuthoringOperationSaveAppearance,
		RawForm: map[string]string{
			"colorCanvas":   "#f6f0e8",
			"colorAccent":   "#b86a4a",
			"displayFont":   "Fraunces",
			"unknownField":  "should-be-filtered",
			"anotherSecret": "also-filtered",
		},
	}
	result, err := ApplySaveAppearance(m, allowedKeys, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.FieldErrors) > 0 {
		t.Fatalf("valid appearance save must not error: %v", result.FieldErrors)
	}
	if !result.RefreshPreview {
		t.Fatal("result must have RefreshPreview=true")
	}
	// Allowed keys present.
	if gotForm["colorCanvas"] != "#f6f0e8" || gotForm["colorAccent"] != "#b86a4a" || gotForm["displayFont"] != "Fraunces" {
		t.Fatalf("allowed keys must be forwarded: %v", gotForm)
	}
	// Non-allowed keys must be absent.
	if _, ok := gotForm["unknownField"]; ok {
		t.Fatal("unknownField must be filtered out")
	}
	if _, ok := gotForm["anotherSecret"]; ok {
		t.Fatal("anotherSecret must be filtered out")
	}
}

func TestApplySaveAppearanceEmptyFilteredFormReturnsError(t *testing.T) {
	called := false
	write := AppearanceWriter(func(rawForm map[string]string) (map[string]string, error) {
		called = true
		return nil, nil
	})

	allowedKeys := []string{"colorCanvas", "colorAccent"}
	m := AuthoringMutation{
		Kind:    AuthoringOperationSaveAppearance,
		RawForm: map[string]string{"unknownField": "x"},
	}
	result, err := ApplySaveAppearance(m, allowedKeys, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("write must NOT be called when filtered form is empty")
	}
	if _, ok := result.FieldErrors[AuthoringFieldOperation]; !ok {
		t.Fatalf("empty filtered form must produce operation field error, got: %v", result.FieldErrors)
	}
}

func TestApplySaveAppearanceForwardsWriterFieldErrors(t *testing.T) {
	write := AppearanceWriter(func(rawForm map[string]string) (map[string]string, error) {
		return map[string]string{"colorAccent": "invalid color value"}, nil
	})

	allowedKeys := []string{"colorAccent"}
	m := AuthoringMutation{
		Kind:    AuthoringOperationSaveAppearance,
		RawForm: map[string]string{"colorAccent": "notacolor"},
	}
	result, err := ApplySaveAppearance(m, allowedKeys, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.FieldErrors["colorAccent"]; !ok {
		t.Fatalf("writer field errors must be forwarded, got: %v", result.FieldErrors)
	}
}

func TestApplySaveAppearancePropagatesWriterError(t *testing.T) {
	write := AppearanceWriter(func(rawForm map[string]string) (map[string]string, error) {
		return nil, errors.New("storage failure")
	})

	allowedKeys := []string{"colorAccent"}
	m := AuthoringMutation{
		Kind:    AuthoringOperationSaveAppearance,
		RawForm: map[string]string{"colorAccent": "#ff0000"},
	}
	_, err := ApplySaveAppearance(m, allowedKeys, write)
	if err == nil {
		t.Fatal("writer system error must bubble up")
	}
}

func TestApplySaveAppearanceOnlyAllowedKeysPresent(t *testing.T) {
	var gotForm map[string]string
	write := AppearanceWriter(func(rawForm map[string]string) (map[string]string, error) {
		gotForm = rawForm
		return nil, nil
	})

	allowedKeys := []string{"colorCanvas"}
	m := AuthoringMutation{
		Kind: AuthoringOperationSaveAppearance,
		// RawForm contains only one allowed key (colorCanvas); colorAccent is absent.
		RawForm: map[string]string{
			"colorCanvas": "#ffffff",
		},
	}
	_, err := ApplySaveAppearance(m, allowedKeys, write)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotForm) != 1 || gotForm["colorCanvas"] != "#ffffff" {
		t.Fatalf("only present allowed keys must be forwarded, got: %v", gotForm)
	}
}
