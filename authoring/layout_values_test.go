package authoring

import "testing"

func TestResponsiveLayoutSchemaCoversBoxStackFlexGrid(t *testing.T) {
	for _, property := range []string{"display", "flex-direction", "grid-template-columns", "align-items", "justify-content", "gap", "padding", "width", "max-width"} {
		if !IsResponsiveLayoutProperty(property) {
			t.Fatalf("missing typed layout property %q", property)
		}
	}
}

func TestValidateLayoutValueUsesExactPerPropertyVocabulary(t *testing.T) {
	accepted := map[string]string{
		"display": "grid", "grid-template-columns": "repeat(3,minmax(0,1fr))",
		"gap": "var(--space-md)", "padding": "0", "width": "100%", "max-width": "48rem",
	}
	for property, value := range accepted {
		if reason, ok := ValidateLayoutValue(property, value); !ok {
			t.Fatalf("%s=%q rejected: %s", property, value, reason)
		}
	}
	for property, value := range map[string]string{
		"gap": "-1rem", "padding": "calc(100vw + 2rem)", "width": "120vw",
		"display": "contents", "grid-template-columns": "repeat(99,1fr)",
	} {
		if _, ok := ValidateLayoutValue(property, value); ok {
			t.Fatalf("unsafe/arbitrary %s=%q accepted", property, value)
		}
	}
}

func TestResponsiveLayoutControlsReturnsDefensiveCopy(t *testing.T) {
	first := ResponsiveLayoutControls()
	first[0].Options[0].Value = "poison"
	second := ResponsiveLayoutControls()
	if second[0].Options[0].Value == "poison" {
		t.Fatal("caller mutated canonical layout vocabulary")
	}
}
