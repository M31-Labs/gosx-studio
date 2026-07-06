package panels

import (
	"os"
	"strings"
	"testing"
)

func TestBrandMediaPickerIslandSourceContract(t *testing.T) {
	source, err := os.ReadFile("brand_media_picker_island.gsx")
	if err != nil {
		t.Fatalf("read island source: %v", err)
	}
	text := string(source)
	typeSource, err := os.ReadFile("brand_media_picker.go")
	if err != nil {
		t.Fatalf("read island prop source: %v", err)
	}
	typeText := string(typeSource)

	for _, want := range []string{
		"package panels",
		"//gosx:island",
		"func BrandMediaPickerIsland(props BrandMediaPickerProps) gosx.Node",
		`filter := signal.New("all")`,
		`showReady := func() { filter.Set("ready") }`,
		`showMissingAlt := func() { filter.Set("missing-alt") }`,
		`data-studio-media-filter={filter.Get()}`,
		`hidden={filter.Get() != "all" && asset.FilterGroup != filter.Get()}`,
		`form={props.FormID}`,
		`formaction={props.SaveAction}`,
		`name={props.LogoPickName}`,
		`name={props.FaviconPickName}`,
		`value={asset.URL}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("brand media picker island source missing %q", want)
		}
	}
	for _, want := range []string{
		"type BrandMediaPickerProps struct",
		"Kicker          string",
		"FilterLabel     string",
		"LogoPickName    string",
		"FaviconPickName string",
		"HasAssets       bool",
		"Assets          []BrandMediaPickerAssetItem",
		"type BrandMediaPickerAssetItem struct",
		"FilterGroup string",
		"ActionLabel string",
	} {
		if !strings.Contains(typeText, want) {
			t.Fatalf("brand media picker prop source missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"props gosx.Node",
		"props any",
		"data.mediaPicker",
		"asset.filterGroup",
		"props.logoPickName",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("brand media picker island source should use serializable typed props, found %q", forbidden)
		}
	}
}
