package studio

import "testing"

func styleTestPageComponent() (Page, Component) {
	page := Page{Key: "home", Label: "Home", Route: "/"}
	component := Component{Key: "hero", Label: "Hero", Binding: "blocks.hero"}
	return page, component
}

func TestAuthoringMutationForStyleSetsFields(t *testing.T) {
	page, component := styleTestPageComponent()
	m := AuthoringMutationForStyle(page, component, "Padding-Top", " 96px ", "", "")
	if m.Kind != AuthoringOperationSetStyle {
		t.Fatalf("kind = %q, want set-style", m.Kind)
	}
	if m.ComponentKey != "hero" || m.PageKey != "home" {
		t.Fatalf("target not set: page=%q component=%q", m.PageKey, m.ComponentKey)
	}
	if m.StyleProperty != "padding-top" {
		t.Fatalf("property must be normalized to lowercase, got %q", m.StyleProperty)
	}
	if m.StyleValue != "96px" {
		t.Fatalf("value must be trimmed, got %q", m.StyleValue)
	}
	if m.Breakpoint != StyleBreakpointBase || m.State != StyleStateDefault {
		t.Fatalf("empty breakpoint/state must default to base/default, got %q/%q", m.Breakpoint, m.State)
	}
}

func TestSetStyleRoundTripsThroughForm(t *testing.T) {
	page, component := styleTestPageComponent()
	original := AuthoringMutationForStyle(page, component, "background-color", "#f4ece1", StyleBreakpointMobile, StyleStateHover)

	rebuilt, validation := AuthoringMutationFromForm(original.FormValues())
	if !validation.OK() {
		t.Fatalf("round-trip must validate, errors: %v", validation.FieldErrors)
	}
	if rebuilt.Kind != AuthoringOperationSetStyle ||
		rebuilt.StyleProperty != "background-color" ||
		rebuilt.StyleValue != "#f4ece1" ||
		rebuilt.Breakpoint != StyleBreakpointMobile ||
		rebuilt.State != StyleStateHover ||
		rebuilt.ComponentKey != "hero" {
		t.Fatalf("round-trip lost fields: %+v", rebuilt)
	}
}

func TestSetStyleValidMutationPasses(t *testing.T) {
	page, component := styleTestPageComponent()
	m := AuthoringMutationForStyle(page, component, "text-align", "center", StyleBreakpointBase, StyleStateDefault)
	if v := m.Validate(); !v.OK() {
		t.Fatalf("a well-formed set-style must validate, errors: %v", v.FieldErrors)
	}
}

func TestSetStyleRejectsUnknownProperty(t *testing.T) {
	page, component := styleTestPageComponent()
	m := AuthoringMutationForStyle(page, component, "behavior", "url(x)", "", "")
	v := m.Validate()
	if _, ok := v.FieldErrors[AuthoringFieldStyleProperty]; !ok {
		t.Fatalf("unsupported property must be rejected, errors: %v", v.FieldErrors)
	}
}

func TestSetStyleRejectsDangerousValues(t *testing.T) {
	page, component := styleTestPageComponent()
	for _, bad := range []string{
		"red; position:fixed",    // declaration injection
		"red} body{display:none", // selector breakout
		"javascript:alert(1)",
		"url(javascript:alert(1))",
		"expression(alert(1))",
		"</style><script>",
	} {
		m := AuthoringMutationForStyle(page, component, "color", bad, "", "")
		v := m.Validate()
		if _, ok := v.FieldErrors[AuthoringFieldStyleValue]; !ok {
			t.Fatalf("dangerous value %q must be rejected, errors: %v", bad, v.FieldErrors)
		}
	}
}

func TestSetStyleRejectsEmptyValue(t *testing.T) {
	page, component := styleTestPageComponent()
	m := AuthoringMutationForStyle(page, component, "color", "   ", "", "")
	if _, ok := m.Validate().FieldErrors[AuthoringFieldStyleValue]; !ok {
		t.Fatal("empty value must be rejected")
	}
}

func TestSetStyleRejectsUnknownBreakpointAndState(t *testing.T) {
	page, component := styleTestPageComponent()
	m := AuthoringMutationForStyle(page, component, "color", "#000", "ultrawide", "pressed")
	v := m.Validate()
	if _, ok := v.FieldErrors[AuthoringFieldBreakpoint]; !ok {
		t.Fatalf("unknown breakpoint must be rejected, errors: %v", v.FieldErrors)
	}
	if _, ok := v.FieldErrors[AuthoringFieldState]; !ok {
		t.Fatalf("unknown state must be rejected, errors: %v", v.FieldErrors)
	}
}

func TestSetStyleRequiresPageComponent(t *testing.T) {
	m := AuthoringMutation{Kind: AuthoringOperationSetStyle, StyleProperty: "color", StyleValue: "#000"}
	v := m.Validate()
	if _, ok := v.FieldErrors[AuthoringFieldPageKey]; !ok {
		t.Fatalf("set-style must require a page, errors: %v", v.FieldErrors)
	}
	if _, ok := v.FieldErrors[AuthoringFieldComponentKey]; !ok {
		t.Fatalf("set-style must require a component, errors: %v", v.FieldErrors)
	}
}

func TestSetStyleOperationKindRecognized(t *testing.T) {
	if got := normalizeAuthoringOperationKind("set-style"); got != AuthoringOperationSetStyle {
		t.Fatalf("set-style must be a recognized op kind, got %q", got)
	}
}

func TestSupportedStylePropertiesSortedAndCoversDesign(t *testing.T) {
	props := SupportedStyleProperties()
	if len(props) == 0 {
		t.Fatal("supported properties must be non-empty")
	}
	for i := 1; i < len(props); i++ {
		if props[i-1] >= props[i] {
			t.Fatalf("supported properties must be sorted+unique, got %q then %q", props[i-1], props[i])
		}
	}
	// The GoSX Studio inspector Style section drives exactly these:
	for _, required := range []string{"text-align", "background-color", "color", "padding-block", "font-size"} {
		if !IsSupportedStyleProperty(required) {
			t.Fatalf("inspector requires %q to be a supported style property", required)
		}
	}
}

// Non-style mutations must be unaffected: their FormValues carry no style fields.
func TestNonStyleMutationOmitsStyleFields(t *testing.T) {
	page, component := styleTestPageComponent()
	control := Control{Key: "title", Kind: ControlText, Binding: "blocks.hero.title", Value: "Hello"}
	values := AuthoringMutationForControl(page, component, control).FormValues()
	for _, field := range []string{AuthoringFieldStyleProperty, AuthoringFieldStyleValue, AuthoringFieldBreakpoint, AuthoringFieldState} {
		if _, present := values[field]; present {
			t.Fatalf("non-style mutation must not emit %q", field)
		}
	}
}
