package core

import "testing"

func reusableDefinitionFixture() ComponentDefinition {
	return ComponentDefinition{
		ID: "def-hero", Key: "shared-hero", Version: 2, TemplateKey: "hero", Label: "Shared hero",
		Controls: []Control{{Key: "headline", Label: "Headline", Kind: ControlRichText, Value: "Original"}},
		Styles:   DefinitionStyles{"base/default": {"background-color": "var(--color-canvas)"}},
		Layout:   DefinitionLayout{"base/default": {"display": "grid"}},
		Binding:  ResourceBinding{Key: "hero-copy", Label: "Hero copy", Binding: "pages.home.hero"},
	}
}

func TestResolveComponentInstanceFollowsDefinitionAndExplicitOverrides(t *testing.T) {
	definition := reusableDefinitionFixture()
	instance := ComponentInstance{
		ID: "inst-home-hero", DefinitionID: definition.ID, DefinitionVersion: definition.Version, PageKey: "home", Region: "main", Position: 0,
		Overrides: map[string]ExplicitOverride{
			"control:headline":               {Present: true, Value: "Instance headline"},
			"style:mobile/default:font-size": {Present: true, Value: "var(--text-xl)"},
			"layout:mobile/default:display":  {Present: true, Value: "block"},
			"binding":                        {Present: true, Value: "pages.home.promo"},
		},
	}
	if got := instance.CanvasIdentity(); got != "instance:inst-home-hero" {
		t.Fatalf("canvas identity must remain instance-based, got %q", got)
	}
	if err := instance.Validate(map[string]ComponentDefinition{definition.ID: definition}, []string{"main", "footer"}); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveComponentInstance(instance, map[string]ComponentDefinition{definition.ID: definition})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Controls[0].Value != "Instance headline" || resolved.Styles["mobile/default"]["font-size"] != "var(--text-xl)" || resolved.Layout["mobile/default"]["display"] != "block" || resolved.Binding.Binding != "pages.home.promo" {
		t.Fatalf("explicit overrides were not materialized: %#v", resolved)
	}
	if definition.Controls[0].Value != "Original" || definition.Styles["mobile/default"] != nil {
		t.Fatalf("resolution must not mutate the reusable definition: %#v", definition)
	}
}

func TestDetachedInstanceUsesExactMaterialization(t *testing.T) {
	original := reusableDefinitionFixture().Normalize()
	instance := ComponentInstance{ID: "inst-1", DefinitionID: original.ID, DefinitionVersion: original.Version, PageKey: "home", Region: "main", Detached: true, Materialized: &original}
	updated := original.Normalize()
	updated.Version++
	updated.Controls[0].Value = "Definition changed"
	resolved, err := ResolveComponentInstance(instance, map[string]ComponentDefinition{updated.ID: updated})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Controls[0].Value != "Original" || resolved.Version != original.Version {
		t.Fatalf("detached instance must preserve its materialized definition: %#v", resolved)
	}
	if err := instance.Validate(map[string]ComponentDefinition{}, []string{"main"}); err != nil {
		t.Fatalf("detached materialization must remain valid without the source definition: %v", err)
	}
}

func TestReusableValidationRejectsInvalidReferencesAndImplicitOverrides(t *testing.T) {
	definition := reusableDefinitionFixture()
	if err := definition.Validate([]ComponentTemplate{{Key: "cta"}}); err == nil {
		t.Fatal("unknown template must be rejected")
	}
	instance := ComponentInstance{ID: "inst-1", DefinitionID: definition.ID, PageKey: "home", Region: "unknown", Overrides: map[string]ExplicitOverride{"control:headline": {Value: "implicit"}}}
	if err := instance.Validate(map[string]ComponentDefinition{definition.ID: definition}, []string{"main"}); err == nil {
		t.Fatal("invalid region and Present=false override must be rejected")
	}
	for _, invalid := range []string{"", "control:", "style:base:color", "style:base/default:COLOR", "raw-html"} {
		if ValidInstanceOverrideKey(invalid) {
			t.Fatalf("invalid override key accepted: %q", invalid)
		}
	}
}
