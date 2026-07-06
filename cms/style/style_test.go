package style

import (
	"strings"
	"testing"
)

func TestCompileCSSDeterministic(t *testing.T) {
	system := muddyLikeSystem()
	first, errs := CompileCSS(system)
	if len(errs) > 0 {
		t.Fatalf("compile first: %v", errs)
	}
	second, errs := CompileCSS(system)
	if len(errs) > 0 {
		t.Fatalf("compile second: %v", errs)
	}
	if first != second {
		t.Fatalf("expected deterministic css\nfirst: %s\nsecond: %s", first, second)
	}
	for _, want := range []string{
		`.gosx-style-muddy-noni{`,
		`--color-accent:#b86a4a;`,
		`--font-body:Space Grotesk;`,
		`.gosx-style-muddy-noni .gosx-style-template-studio{`,
		`--style-buttons:pill;`,
		`--style-spacing:balanced;`,
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("expected %q in css: %s", want, first)
		}
	}
}

func TestValidateRejectsInvalidScopeAndOption(t *testing.T) {
	system := muddyLikeSystem()
	system.Scopes = append(system.Scopes,
		StyleScope{Kind: "planet", TargetID: "mars", Values: map[string]Value{"buttons": String("pill")}},
		StyleScope{Kind: ScopeTemplate, TargetID: "studio", Values: map[string]Value{"buttons": String("triangle")}},
		StyleScope{Kind: ScopeBlock, Values: map[string]Value{"buttons": String("pill")}},
	)
	errs := Validate(system)
	if len(errs) != 3 {
		t.Fatalf("expected 3 validation errors, got %d: %v", len(errs), errs)
	}
	for _, want := range []string{"unsupported style scope", "choose one of the available options", "target id is required"} {
		if !strings.Contains(errs.Error(), want) {
			t.Fatalf("expected %q in errors: %v", want, errs)
		}
	}
}

func TestValidateRejectsUnknownControlWhenRecipesExist(t *testing.T) {
	system := muddyLikeSystem()
	system.Scopes = []StyleScope{{
		Kind:     ScopeBlock,
		TargetID: "hero",
		Values:   map[string]Value{"unbounded-css": String("surprise")},
	}}
	errs := Validate(system)
	if len(errs) != 1 || !strings.Contains(errs.Error(), "style control is not defined") {
		t.Fatalf("unexpected validation errors: %v", errs)
	}
}

func TestValidateContrast(t *testing.T) {
	system := muddyLikeSystem()
	system.Tokens.Colors["ink"] = "#f6f0e8"
	errs := Validate(system)
	if len(errs) == 0 || !strings.Contains(errs.Error(), "contrast") {
		t.Fatalf("expected contrast error, got %v", errs)
	}
}

func TestScopeClass(t *testing.T) {
	className := ScopeClass(StyleScope{Kind: ScopeBlock, TargetID: "Hero Lead"})
	if className != "gosx-style-block-hero-lead" {
		t.Fatalf("unexpected scope class %q", className)
	}
}

func TestRecipeViewsExposeControlsDefaultsAndVariants(t *testing.T) {
	system := muddyLikeSystem()
	system.Recipes[0].Controls = []StyleControl{{
		Key:     "spacing",
		Label:   "Spacing",
		Kind:    ControlSpacing,
		Default: String("balanced"),
		Options: []StyleOption{
			{Value: "airy", Label: "Airy"},
			{Value: "balanced", Label: "Balanced", CSS: "var(--space-md)"},
		},
	}}
	system.Recipes[0].Variants = []StyleVariant{{
		Key:    "airy",
		Label:  "Airy",
		Values: map[string]Value{"spacing": String("airy")},
	}}

	views := RecipeViews(system)
	if len(views) != 1 || views[0].Key != "theme" || views[0].Label != "Theme" {
		t.Fatalf("unexpected recipe views: %#v", views)
	}
	if len(views[0].Controls) != 1 {
		t.Fatalf("unexpected control views: %#v", views[0].Controls)
	}
	control := views[0].Controls[0]
	if control.Key != "spacing" || control.Kind != string(ControlSpacing) || !control.HasDefault || control.Default != "balanced" {
		t.Fatalf("unexpected control view: %#v", control)
	}
	if len(control.Options) != 2 || control.Options[0].Value != "airy" || control.Options[1].CSS != "var(--space-md)" {
		t.Fatalf("unexpected option views: %#v", control.Options)
	}
	if len(views[0].Variants) != 1 || views[0].Variants[0].Values["spacing"] != "airy" {
		t.Fatalf("unexpected variant views: %#v", views[0].Variants)
	}
}

func muddyLikeSystem() DesignSystem {
	return DesignSystem{
		ID: "Muddy Noni",
		Brand: Brand{
			Name: "Muddy Noni",
			Tone: "Warm and handmade",
		},
		Tokens: TokenSet{
			Colors: map[string]string{
				"canvas":       "#f6f0e8",
				"canvasSoft":   "#fdfaf6",
				"ink":          "#2a241e",
				"inkSoft":      "#4c4036",
				"muted":        "#76685c",
				"accent":       "#b86a4a",
				"accentStrong": "#965036",
				"moss":         "#2f5143",
				"sun":          "#e5b36a",
				"stone":        "#cbb39a",
				"danger":       "#9f3d2f",
			},
			Typography: map[string]string{
				"display": "Fraunces",
				"body":    "Space Grotesk",
				"mono":    "JetBrains Mono",
			},
			Spacing: map[string]string{
				"density": "balanced",
			},
			Radii: map[string]string{
				"image": "0",
			},
			Motion: map[string]string{
				"mode": "subtle",
			},
		},
		Recipes: []ComponentRecipe{{
			Key:   "theme",
			Label: "Theme",
			Controls: []StyleControl{
				{Key: "nav", Label: "Navigation", Kind: ControlChoice, Options: []StyleOption{{Value: "links"}, {Value: "tabs"}, {Value: "buttons"}}},
				{Key: "buttons", Label: "Buttons", Kind: ControlChoice, Options: []StyleOption{{Value: "pill"}, {Value: "square"}, {Value: "underlined"}}},
				{Key: "cards", Label: "Cards", Kind: ControlChoice, Options: []StyleOption{{Value: "quiet"}, {Value: "framed"}, {Value: "raised"}}},
				{Key: "spacing", Label: "Spacing", Kind: ControlSpacing, Options: []StyleOption{{Value: "compact"}, {Value: "balanced"}, {Value: "airy"}}},
				{Key: "images", Label: "Image frame", Kind: ControlCrop, Options: []StyleOption{{Value: "sharp"}, {Value: "soft"}, {Value: "round"}}},
				{Key: "motion", Label: "Motion", Kind: ControlMotion, Options: []StyleOption{{Value: "still"}, {Value: "subtle"}}},
			},
		}},
		Scopes: []StyleScope{{
			Kind:     ScopeTemplate,
			TargetID: "studio",
			Values: map[string]Value{
				"nav":     String("links"),
				"buttons": String("pill"),
				"cards":   String("quiet"),
				"spacing": String("balanced"),
				"images":  String("sharp"),
				"motion":  String("subtle"),
			},
		}},
		Breakpoints: []Breakpoint{
			{Key: "mobile", Label: "Mobile"},
			{Key: "desktop", Label: "Desktop", MinWidth: "64rem"},
		},
	}
}
