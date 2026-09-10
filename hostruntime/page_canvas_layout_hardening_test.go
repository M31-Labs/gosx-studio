package hostruntime

import (
	"strings"
	"testing"
)

const sectionOrderPageCanvasSelector = `.studio-page-canvas:has(> [data-gosx-studio-section-order="true"])`

func stylesheetRule(css, selector string) string {
	start := strings.Index(css, selector)
	if start < 0 {
		return ""
	}
	openOffset := strings.Index(css[start:], "{")
	if openOffset < 0 {
		return ""
	}
	closeOffset := strings.Index(css[start+openOffset+1:], "}")
	if closeOffset < 0 {
		return ""
	}
	return css[start : start+openOffset+closeOffset+2]
}

func TestPageCanvasSectionOrderLayoutReservesPrimaryPreview(t *testing.T) {
	css := string(Stylesheet())
	canvasRule := stylesheetRule(css, sectionOrderPageCanvasSelector)
	if canvasRule == "" {
		t.Fatalf("SectionOrder PageCanvas layout rule missing %q", sectionOrderPageCanvasSelector)
	}
	for _, want := range []string{
		`grid-template-rows:
    auto
    minmax(8rem, clamp(8rem, 25vh, 14rem))
    minmax(clamp(15rem, calc(25vh + 3rem), 16rem), 1fr)
    auto
    auto;`,
		`overflow: auto;`,
		`scrollbar-gutter: stable;`,
	} {
		if !strings.Contains(canvasRule, want) {
			t.Fatalf("SectionOrder PageCanvas layout contract missing %q", want)
		}
	}

	laneSelector := sectionOrderPageCanvasSelector + `
  > [data-gosx-studio-section-order="true"]`
	laneRule := stylesheetRule(css, laneSelector)
	if laneRule == "" {
		t.Fatalf("SectionOrder lane selector missing %q", laneSelector)
	}
	for _, want := range []string{
		`grid-row: 2;`,
		`min-height: 0;`,
		`overflow: auto;`,
		`overscroll-behavior: contain;`,
		`scrollbar-gutter: stable;`,
	} {
		if !strings.Contains(laneRule, want) {
			t.Fatalf("SectionOrder lane must remain an internal scrollport with %q", want)
		}
	}

	for index, selector := range []string{
		sectionOrderPageCanvasSelector + `
  > .studio-page-canvas__stage {`,
		sectionOrderPageCanvasSelector + `
  > .studio-page-canvas__status {`,
		sectionOrderPageCanvasSelector + `
  > .studio-page-canvas__diagnostic {`,
	} {
		rowRule := stylesheetRule(css, strings.TrimSuffix(selector, " {"))
		if rowRule == "" || !strings.Contains(rowRule, `grid-row: `+string(rune('3'+index))+`;`) {
			t.Fatalf("SectionOrder PageCanvas row selector missing %q", selector)
		}
	}
}

func TestPageCanvasSectionOrderPhoneFloorAndCanonicalLayout(t *testing.T) {
	css := string(Stylesheet())
	if !strings.Contains(css, `minmax(clamp(10rem, 20vh, 12rem), 1fr)`) {
		t.Fatal("phone SectionOrder PageCanvas must reserve a responsive preview floor")
	}
	if !strings.Contains(css, `grid-template-rows: auto minmax(0, 1fr) auto auto;`) {
		t.Fatal("PageCanvas must retain its canonical four-row layout when SectionOrder is absent")
	}

	// The preview floor is intentionally expressed by the optional row track;
	// do not turn the hardening slice into a global minimum for every PageCanvas
	// or another editor surface.
	if strings.Contains(css, `.studio-page-canvas {
  min-block-size: clamp(`) {
		t.Fatal("preview floor must not become a global PageCanvas minimum")
	}
}
