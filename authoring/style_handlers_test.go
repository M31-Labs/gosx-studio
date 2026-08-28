package authoring

import (
	"errors"
	"testing"

	"m31labs.dev/gosx-studio/core"
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

	page, component := core.Page{Key: "home", Label: "Home", Route: "/"}, core.Component{Key: "hero", Label: "Hero"}
	m := AuthoringMutationForStyle(page, component, "padding-block", "var(--space-xl)", StyleBreakpointTablet, StyleStateHover)
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
	if gotKey != "hero" || gotProp != "padding-block" || gotVal != "var(--space-xl)" ||
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

	page, component := core.Page{Key: "home"}, core.Component{Key: "hero"}
	m := AuthoringMutationForStyle(page, component, "color", "#000", "", "")
	_, err := ApplySetStyle(m, write)
	if err == nil {
		t.Fatal("writer error must bubble up")
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
