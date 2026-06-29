package studio

import (
	"os"
	"strings"
	"testing"
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
		"package studio",
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
