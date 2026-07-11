package panels

import (
	"os"
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestHomeLayerSelectionIslandSourceContract(t *testing.T) {
	source, err := os.ReadFile("home_layer_selection_island.gsx")
	if err != nil {
		t.Fatalf("read island source: %v", err)
	}
	text := string(source)
	typeSource, err := os.ReadFile("home_layer_selection.go")
	if err != nil {
		t.Fatalf("read island prop source: %v", err)
	}
	typeText := string(typeSource)

	for _, want := range []string{
		"package panels",
		"//gosx:island",
		"func HomeLayerSelectionIsland(props HomeLayerSelectionProps) gosx.Node",
		"selected := signal.New(props.DefaultSelectedKey)",
		"selectedLabel := signal.New(props.DefaultSelectedLabel)",
		`data-studio-home-layer-selected={selected.Get()}`,
		`<Each of={props.Items} as="layer">`,
		`data-studio-home-layer-pick={layer.Key}`,
		`onClick={func() { selected.Set(layer.Key); selectedLabel.Set(layer.Label) }}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("home layer selection island source missing %q", want)
		}
	}
	for _, want := range []string{
		"type HomeLayerSelectionProps struct",
		"DefaultSelectedKey   string",
		"DefaultSelectedLabel string",
		"Items                []HomeLayerSelectionItem",
	} {
		if !strings.Contains(typeText, want) {
			t.Fatalf("home layer selection prop source missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"props gosx.Node",
		"props any",
		"data.homeLayers",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("home layer selection island source should use serializable typed props, found %q", forbidden)
		}
	}
}

func TestRenderHomeLayerSelectionUsesTypedProps(t *testing.T) {
	html := gosx.RenderHTML(RenderHomeLayerSelection(HomeLayerSelectionProps{
		DefaultSelectedKey:   "hero",
		DefaultSelectedLabel: "Hero",
		Items: []HomeLayerSelectionItem{
			{Key: "hero", Label: "Hero"},
			{Key: "featured", Label: "Featured"},
		},
	}))

	for _, fragment := range []string{
		`<div class="studio-home-layer-picker" data-studio-home-layer-picker="true" data-studio-home-layer-selected="hero">`,
		`<output data-studio-selection-label="true" aria-live="polite">Hero</output>`,
		`<div class="studio-home-layer-picker__list" role="toolbar" aria-label="Select home section">`,
		`<button type="button" data-studio-home-layer-pick="hero" aria-pressed="true">Hero</button>`,
		`<button type="button" data-studio-home-layer-pick="featured" aria-pressed="false">Featured</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("home layer selection render missing %q:\n%s", fragment, html)
		}
	}
}
