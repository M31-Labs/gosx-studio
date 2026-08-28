package hostruntime

import (
	"strings"
	"testing"
)

const galleryRendererScope = `[data-gosx-studio-backend-gallery-index-renderer="gosx-studio"]`

func galleryResponsiveCSS(t *testing.T) string {
	t.Helper()
	css := string(Stylesheet())
	start := strings.Index(css, galleryRendererScope)
	if start < 0 {
		t.Fatal("Gallery renderer scope is missing")
	}
	endOffset := strings.Index(css[start:], "\n.admin-page .table-product img")
	if endOffset < 0 {
		t.Fatal("Gallery renderer scope is not bounded before shared table-product styles")
	}
	return css[start : start+endOffset]
}

func TestGalleryResponsivePolishContainsLabeledStackedWorkCards(t *testing.T) {
	css := galleryResponsiveCSS(t)

	for _, want := range []string{
		`.data-table`,
		`.table-product`,
		`.table-product > div`,
		`min-width: 0;`,
		`max-width: 100%;`,
		`overflow-wrap: anywhere;`,
		`display: block;`,
		`tbody`,
		`tr`,
		`td`,
		`grid-template-columns: minmax(0, 1fr);`,
		`grid-template-columns: minmax(5.5rem, auto) minmax(0, 1fr);`,
		`align-items: start;`,
		`content: attr(data-label);`,
		`clip-path: inset(50%);`,
		`td:last-child`,
		`a:hover`,
		`a:focus-visible`,
		`a:active`,
		`outline: 2px solid var(--color-accent);`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("Gallery responsive stylesheet missing %q", want)
		}
	}
}

func TestGalleryResponsivePolishDoesNotHardcodeColumnLabelsOrHideOverflow(t *testing.T) {
	css := galleryResponsiveCSS(t)

	for _, forbidden := range []string{
		`td:nth-child`,
		`content: "Work"`,
		`content: "Year"`,
		`content: "Visibility"`,
		`content: "Updated"`,
		`.table-product > div {\n    overflow: hidden;`,
	} {
		if strings.Contains(css, forbidden) {
			t.Fatalf("Gallery responsive polish must derive labels and contain text without masking it; found %q", forbidden)
		}
	}
}

func TestGalleryResponsivePolishStacksWorkThumbnailAboveCopy(t *testing.T) {
	css := galleryResponsiveCSS(t)
	selector := `.data-table
    .table-product {`
	start := strings.LastIndex(css, selector)
	if start < 0 {
		t.Fatalf("Gallery phone Work product selector is missing %q", selector)
	}
	if !strings.Contains(css[start:], `grid-template-columns: minmax(0, 1fr);`) {
		t.Fatal("Gallery phone Work product must stack its thumbnail above full-width copy")
	}
}

func TestGalleryResponsivePolishLeavesSharedDesktopTableContractIntact(t *testing.T) {
	css := string(Stylesheet())
	if !strings.Contains(css, `.admin-page .data-table {
  width: 100%;
  border-collapse: collapse;
}`) {
		t.Fatal("shared native data-table desktop contract is missing")
	}
	if !strings.Contains(css, galleryRendererScope) {
		t.Fatal("Gallery responsive rules are not scoped to the Gallery renderer")
	}
}
