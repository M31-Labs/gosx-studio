package studio

import (
	"strings"
	"testing"
)

func TestStylesheetCarriesReusableBackendEditorSurface(t *testing.T) {
	css := string(Stylesheet())

	for _, check := range []string{
		`--font-display: "Fraunces", Georgia, serif;`,
		`--font-body: "Space Grotesk", "Avenir Next", sans-serif;`,
		`--font-mono: "JetBrains Mono", monospace;`,
		`--color-canvas: #f6f0e8;`,
		`--color-panel: #fffaf2;`,
		`--color-accent: #b86a4a;`,
		`--space-xs: clamp(0.75rem, 0.7rem + 0.25vw, 0.9rem);`,
		`.gosx-studio-workbench`,
		`.editor-workbench`,
		`.studio-canvas-bar`,
		`.studio-canvas-status`,
		`.studio-zoombar`,
		`.studio-left-rail`,
		`.editor-sidebar`,
		`.studio-site-map-engine`,
		`.studio-site-map-board`,
		`.studio-site-map-workspace__layers`,
		`.studio-home-inspector`,
		`.studio-look-panel`,
		`.studio-brand-panel`,
		`.studio-publish-panel`,
		`.studio-advanced-panel`,
		`.studio-flow-designer`,
		`.studio-activity-drawer`,
		`.studio-preview-share`,
		`.studio-health`,
		`.studio-performance`,
		`[data-gosx-studio-authoring-selected="true"]`,
		`[data-gosx-studio-site-map-board-renderer="gosx-studio"]`,
		`[data-gosx-studio-site-map-engine-renderer="gosx-studio"]`,
		`[data-gosx-studio-frame-renderer="gosx-studio"]`,
		`[data-gosx-studio-rail-resizer-renderer="gosx-studio"]`,
	} {
		if !strings.Contains(css, check) {
			t.Fatalf("stylesheet missing reusable editor fragment %q", check)
		}
	}
}

func TestStylesheetUsesHiddenAttributeValuesWithoutHidingFalse(t *testing.T) {
	css := string(Stylesheet())

	for _, check := range []string{
		`[hidden=""]`,
		`[hidden="true"]`,
		`display: none !important;`,
	} {
		if !strings.Contains(css, check) {
			t.Fatalf("stylesheet missing hidden fragment %q", check)
		}
	}

	for _, line := range strings.Split(css, "\n") {
		if strings.TrimSpace(line) == "[hidden] {" {
			t.Fatal(`stylesheet must not hide GoSX hidden="false" panels`)
		}
	}
}

func TestStylesheetSupportsBackendPanelCSSOnlyControls(t *testing.T) {
	css := string(Stylesheet())

	for _, check := range []string{
		`.studio-flow-designer__input`,
		`.studio-flow-designer__editors .studio-flow-editor`,
		`.studio-flow-designer__input:checked ~ .studio-flow-designer__editors .studio-flow-editor`,
		`.studio-flow-designer__input:checked ~ .studio-flow-card`,
		`.studio-brand-media-picker__filter-input`,
		`#studioBrandMediaFilterReady:checked ~ .studio-brand-media-picker__grid [data-studio-media-filter-group]:not([data-studio-media-filter-group="ready"])`,
		`#studioBrandMediaFilterMissingAlt:checked ~ .studio-brand-media-picker__grid [data-studio-media-filter-group]:not([data-studio-media-filter-group="missing-alt"])`,
		`#studioBrandMediaFilterAll:checked ~ .studio-brand-media-picker__filters [data-studio-media-filter-option="all"]`,
	} {
		if !strings.Contains(css, check) {
			t.Fatalf("stylesheet missing CSS-only backend panel control selector %q", check)
		}
	}
}

func TestStylesheetFlowRadioSelectionShowsOnlyCheckedEditor(t *testing.T) {
	css := string(Stylesheet())
	defaultHide := `.studio-flow-designer__editors .studio-flow-editor`
	checkedShow := `.studio-flow-designer__input:checked ~ .studio-flow-designer__editors .studio-flow-editor`
	defaultHideIndex := strings.Index(css, defaultHide)
	checkedShowIndex := strings.Index(css, checkedShow)
	if defaultHideIndex < 0 {
		t.Fatalf("stylesheet missing default flow editor hide selector %q", defaultHide)
	}
	if checkedShowIndex < 0 {
		t.Fatalf("stylesheet missing checked flow editor show selector %q", checkedShow)
	}
	if defaultHideIndex > checkedShowIndex {
		t.Fatalf("flow editor default hide selector must appear before checked show selector")
	}
}
