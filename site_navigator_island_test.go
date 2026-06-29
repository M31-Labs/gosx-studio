package studio

import (
	"os"
	"strings"
	"testing"
)

func TestSiteNavigatorIslandSourceContract(t *testing.T) {
	source, err := os.ReadFile("site_navigator_island.gsx")
	if err != nil {
		t.Fatalf("read site navigator island source: %v", err)
	}
	text := string(source)
	typeSource, err := os.ReadFile("site_navigator_panel.go")
	if err != nil {
		t.Fatalf("read site navigator prop source: %v", err)
	}
	typeText := string(typeSource)

	for _, want := range []string{
		"package studio",
		"//gosx:island",
		"func SiteNavigatorIsland(props SiteNavigatorProps) gosx.Node",
		`filter := signal.New("all")`,
		`showSite := func() { filter.Set("site") }`,
		`showContent := func() { filter.Set("content") }`,
		`showStore := func() { filter.Set("store") }`,
		`showTools := func() { filter.Set("tools") }`,
		`data-studio-site-navigator-panel="true"`,
		`data-studio-mode-panel={props.Mode}`,
		`data-studio-engine-source="gosx"`,
		`data-studio-site-filter={filter.Get()}`,
		`data-gosx-studio-site-navigator-renderer="gosx-studio"`,
		`<Each of={props.Items} as="item">`,
		`data-gosx-link="true"`,
		`data-studio-site-page={item.Key}`,
		`data-studio-site-group={item.Group}`,
		`title={item.Summary}`,
		`hidden={filter.Get() != "all" && item.Group != filter.Get()}`,
		`hidden={props.HasItems}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("site navigator island source missing %q", want)
		}
	}
	for _, want := range []string{
		"type SiteNavigatorProps struct",
		"Mode     string",
		"Kicker   string",
		"Title    string",
		"Label    string",
		"Empty    string",
		"HasItems bool",
		"Items    []SiteNavigatorItem",
		"type SiteNavigatorItem struct",
		"Key     string",
		"Group   string",
		"Summary string",
	} {
		if !strings.Contains(typeText, want) {
			t.Fatalf("site navigator prop source missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"props any",
		"props gosx.Node",
		"map[string]any",
		"data.siteNavigator",
		"data.siteNavigatorHeader",
		"data.siteNavigatorList",
		"props.mode",
		"item.group",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("site navigator island source should use serializable typed props, found %q", forbidden)
		}
	}
}
