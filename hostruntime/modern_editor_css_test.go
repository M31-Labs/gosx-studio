package hostruntime

import (
	"strings"
	"testing"
)

func TestModernContentEditorStylesheetContract(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		"Modern shared content editor primitives",
		`:where(.gosx-studio, .gosx-studio-workbench, .editor-workbench, .admin-page)`,
		`.content-editor__toolbar`,
		`[data-content-editor-toolbar]`,
		`.content-block:is(`,
		`[data-content-block-selected="true"]`,
		`[data-content-editor-selected="true"]`,
		`content: "Drop before";`,
		`content: "Drop after";`,
		`[data-content-drag-handle]`,
		`[data-content-editor-keyboard-help]`,
		`[data-content-editor-announcement]`,
		`[data-content-editor-drag-preview="true"]`,
		`[data-content-editor-drop-before]`,
		`[data-content-editor-drop-after]`,
		`[data-content-editor-action]`,
		`touch-action: none;`,
		`grid-template-columns: minmax(0, 1fr) minmax(0, 0.45fr);`,
		`@media (pointer: coarse)`,
		`@media (prefers-reduced-motion: reduce)`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("modern content editor stylesheet missing %q", want)
		}
	}
}

func TestModernContentEditorStylesAvoidKnownOverflowAndMotionRegressions(t *testing.T) {
	css := string(Stylesheet())
	for _, forbidden := range []string{
		`minmax(12rem, 0.45fr)`,
		`minmax(12rem,.45fr)`,
		`.content-block {
  touch-action: none;`,
	} {
		if strings.Contains(css, forbidden) {
			t.Fatalf("modern content editor stylesheet must not contain known regression %q", forbidden)
		}
	}
	for _, want := range []string{
		`.content-block__body {
    grid-template-columns: minmax(0, 1fr);`,
		`transition: none;
    transform: none;`,
		`min-width: calc(var(--space-md) + var(--space-sm) + var(--space-chip-y));`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("modern content editor stylesheet missing responsive/reduced-motion guard %q", want)
		}
	}
}

func TestModernMediaPickerStylesheetContract(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		"Shared media picker and upload drop zone primitives",
		`:where(.media-picker, [data-media-picker])`,
		`.media-picker__toolbar`,
		`.media-picker__preview`,
		`.media-picker__filename`,
		`[data-media-picker-filename]`,
		`.media-picker__clear`,
		`[data-media-picker-clear]`,
		`.media-picker__drop-zone`,
		`[data-media-upload-drop-zone]`,
		`[data-media-upload]`,
		`[data-media-upload-status]`,
		`[data-media-upload-preview]`,
		`[data-media-upload-clear]`,
		`[data-studio-media-upload-drop-zone]`,
		`[data-media-drop-active="true"]`,
		`.media-picker__error`,
		`[data-media-upload-error]`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("modern media picker stylesheet missing %q", want)
		}
	}
}

func TestModernSectionOrderStylesheetContract(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`[data-gosx-studio-section-order="true"]`,
		`.studio-page-canvas__section-order-header`,
		`.studio-page-canvas__section-order-title`,
		`[data-gosx-studio-section-order-status="true"]`,
		`[data-gosx-studio-section-order-state="saving"]`,
		`[data-gosx-studio-section-order-state="saved"]`,
		`[data-gosx-studio-section-order-state="error"]`,
		`[data-gosx-studio-section-order-list="true"]`,
		`[data-gosx-studio-section-order-item]`,
		`[data-gosx-studio-section-order-dragging="true"]`,
		`[data-gosx-studio-section-order-drop="before"]`,
		`[data-gosx-studio-section-order-drop="after"]`,
		`[data-gosx-studio-section-order-handle="true"]`,
		`[data-gosx-studio-section-order-move]`,
		`[data-gosx-studio-section-order-history]`,
		`[data-gosx-studio-section-order-overlay="true"]`,
		`[data-gosx-studio-section-order-preview-handle="true"]`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("modern section order stylesheet missing %q", want)
		}
	}
}

func TestModernSectionOrderUsesItsOwnIntrinsicTrack(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`.studio-page-canvas:has(> [data-gosx-studio-section-order="true"])`,
		`grid-template-rows: auto auto minmax(0, 1fr) auto auto;`,
		`> [data-gosx-studio-section-order="true"] {`,
		`grid-row: 2;`,
		`> .studio-page-canvas__stage {`,
		`grid-row: 3;`,
		`> .studio-page-canvas__status {`,
		`grid-row: 4;`,
		`> .studio-page-canvas__diagnostic {`,
		`grid-row: 5;`,
		`.studio-page-canvas__section-order-guidance`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("section order layout contract missing %q", want)
		}
	}
	if !strings.Contains(css, `grid-template-rows: auto minmax(0, 1fr) auto auto;`) {
		t.Fatal("page canvas must retain the four-row layout for canvases without SectionOrder")
	}
}

func TestModernAdminFormSectionsStackAtPhoneWidths(t *testing.T) {
	css := string(Stylesheet())
	if !strings.Contains(css, ".admin-page .form-section {\n    grid-template-columns: minmax(0, 1fr);") {
		t.Fatal("admin form sections must use one column at phone widths")
	}
}

func TestModernSectionOrderControlsStayCompactAtPhoneWidths(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`[data-gosx-studio-section-order-list="true"] {`,
		`[data-gosx-studio-section-order-item] {`,
		`grid-template-columns: auto minmax(0, 1fr) auto auto;`,
		`width: auto;`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("phone SectionOrder layout contract missing %q", want)
		}
	}
}

func TestModernContentEditorSaveStatusStylesExposeTruthfulStates(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`.content-editor__save-status`,
		`[data-content-editor-save-status]`,
		`[data-content-editor-save-label]`,
		`[data-content-editor-save-detail-text]`,
		`[data-content-editor-save-retry]`,
		`data-content-editor-save-state="idle"`,
		`data-content-editor-save-state="saving"`,
		`data-content-editor-save-state="saved"`,
		`data-content-editor-save-state="dirty"`,
		`data-content-editor-save-state="offline"`,
		`data-content-editor-save-state="failed"`,
		`data-content-editor-save-state="unconfirmed"`,
		`data-content-editor-save-state="auth"`,
		`data-content-editor-save-state="conflict"`,
		`data-gosx-studio-save-state-value="saved"`,
		`@media (pointer: coarse)`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("content editor save status contract missing %q", want)
		}
	}
}

func TestModernContentEditorHistoryAndSearchStylesStayCompact(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`.content-editor__history`,
		`[data-content-editor-history]`,
		`[data-content-editor-action]`,
		`.content-editor__status`,
		`[data-content-editor-status]`,
		`.content-editor__search-controls`,
		`[data-content-editor-search-controls]`,
		`.content-editor__search-input`,
		`[data-content-editor-search]`,
		`.content-editor__search-clear`,
		`[data-content-editor-search-clear]`,
		`.content-editor__search-count`,
		`[data-content-editor-search-count]`,
		`grid-template-columns: minmax(0, 1fr) auto auto;`,
		`grid-template-columns: minmax(0, 1fr) minmax(9rem, min(32%, 20rem));`,
		`grid-column: 1;`,
		`grid-column: 2;`,
		`grid-column: 1 / -1;`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("content editor history/search stylesheet contract missing %q", want)
		}
	}
}

func TestModernCMSAuthoringLayoutStylesStayScopedAndResponsive(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`data-gosx-studio-backend-page-detail-renderer="gosx-studio"`,
		`data-gosx-studio-backend-blog-detail-renderer="gosx-studio"`,
		`data-gosx-studio-backend-page-index-renderer="gosx-studio"`,
		`data-gosx-studio-backend-blog-index-renderer="gosx-studio"`,
		`.admin-form[data-content-editor-enhanced-submit]`,
		`.admin-form__primary-actions`,
		`position: sticky;`,
		`.admin-form__secondary-actions`,
		`data-content-editor-danger-actions="true"`,
		`flex-direction: row;`,
		`grid-template-columns: repeat(3, minmax(0, 1fr));`,
		`:has(> [data-content-editor-format])`,
		`.form-error:empty`,
		`@media (min-width: 821px)`,
		`row-gap: 0.5rem;`,
		`background: var(--color-accent-strong);`,
		`color: var(--color-canvas-soft);`,
		`.site-shell--admin:has(.admin-page[data-gosx-studio-backend-page-detail-renderer="gosx-studio"]) .site-main`,
		`padding-top: var(--space-sm);`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("CMS authoring stylesheet contract missing %q", want)
		}
	}
}

func TestModernCMSPrimaryActionsUseStrongContrastAtEveryWidth(t *testing.T) {
	css := string(Stylesheet())
	selector := `.admin-page .admin-form[data-content-editor-enhanced-submit] > .admin-form__primary-actions > .button--primary`
	start := strings.Index(css, selector)
	if start < 0 {
		t.Fatalf("CMS primary action contrast selector missing")
	}
	end := strings.Index(css[start:], "}")
	if end < 0 {
		t.Fatalf("CMS primary action contrast rule is not closed")
	}
	rule := css[start : start+end]
	for _, want := range []string{
		`border-color: var(--color-accent-strong);`,
		`background: var(--color-accent-strong);`,
		`color: var(--color-canvas-soft);`,
	} {
		if !strings.Contains(rule, want) {
			t.Fatalf("CMS primary action contrast rule missing %q", want)
		}
	}
	if cssSelectorIsInsideAtRule(css, start, "@media (min-width: 821px)") {
		t.Fatal("CMS primary action contrast must apply outside the desktop-only media query")
	}
}

func TestModernWorkbenchToolbarKeepsSelectionLabelDistinctAtLaptopWidths(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		`.gosx-studio-toolbar__title`,
		`.gosx-studio-toolbar__title [data-studio-selection-label]`,
		`border-inline-start: 1px solid var(--color-line-strong);`,
		`@media (max-width: 1360px)`,
		`flex-wrap: wrap;`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("workbench toolbar responsive contract missing %q", want)
		}
	}
}

func TestModernWorkbenchModebarWrapsOnDesktopSurfaces(t *testing.T) {
	css := string(Stylesheet())
	toolbarSelector := `.admin-page.editor-page[data-gosx-studio-backend-editor-renderer="gosx-studio"] .editor-toolbar`
	modebarSelector := `.admin-page.editor-page[data-gosx-studio-backend-editor-renderer="gosx-studio"] .studio-modebar`
	media := strings.Index(css, "@media (min-width: 821px)")
	if media < 0 {
		t.Fatalf("desktop workbench toolbar media query missing")
	}
	blockEnd := strings.Index(css[media:], "@media (max-width: 1200px)")
	if blockEnd < 0 {
		t.Fatalf("desktop workbench toolbar media query boundary missing")
	}
	block := css[media : media+blockEnd]
	for _, want := range []string{
		toolbarSelector,
		modebarSelector,
		`grid-template-columns: minmax(0, max-content) minmax(0, 1fr) auto auto;`,
		`grid-template-areas: "title modes command actions";`,
		`display: flex;`,
		`width: 100%;`,
		`flex-wrap: wrap;`,
		`justify-self: stretch;`,
		`overflow: visible;`,
		`scrollbar-gutter: auto;`,
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("desktop workbench modebar contract missing %q", want)
		}
	}
}

func TestModernWorkbenchModebarGetsFullLaptopRow(t *testing.T) {
	css := string(Stylesheet())
	mediaQuery := "@media (min-width: 821px) and (max-width: 1360px)"
	media := strings.Index(css, mediaQuery)
	if media < 0 {
		t.Fatalf("bounded laptop workbench toolbar media query missing")
	}
	blockEnd := strings.Index(css[media:], "@media (max-width: 1200px)")
	if blockEnd < 0 {
		t.Fatalf("bounded laptop workbench toolbar media query boundary missing")
	}
	block := css[media : media+blockEnd]
	for _, want := range []string{
		`grid-template-columns: minmax(0, 1fr) auto auto auto;`,
		`grid-template-rows: auto auto;`,
		`"title metrics command actions"`,
		`"modes modes modes modes"`,
		`.studio-modebar`,
		`grid-area: modes;`,
		`.studio-context-strip`,
		`grid-area: metrics;`,
		`.studio-command-palette`,
		`grid-area: command;`,
		`.editor-toolbar > .button-row`,
		`grid-area: actions;`,
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("bounded laptop workbench toolbar contract missing %q", want)
		}
	}
}

func TestModernEditorStylesUseTokensAndNoImportantOverrides(t *testing.T) {
	css := string(Stylesheet())
	modern := modernEditorStyleBlock(t, css)
	for _, want := range []string{
		`var(--color-accent-strong)`,
		`var(--font-mono)`,
		`var(--text-xs)`,
		`var(--space-xs)`,
		`var(--duration-fast) var(--ease-out)`,
	} {
		if !strings.Contains(modern, want) {
			t.Fatalf("modern editor stylesheet block missing tokenized value %q", want)
		}
	}
	if strings.Contains(modern, "!important") {
		t.Fatal("modern editor stylesheet block must not add !important overrides")
	}
}

func modernEditorStyleBlock(t *testing.T, css string) string {
	t.Helper()
	start := strings.Index(css, "/* Modern shared content editor primitives.")
	if start < 0 {
		t.Fatal("modern content editor stylesheet block not found")
	}
	end := strings.Index(css[start:], "@media (max-width: 820px) {\n  .admin-page")
	if end < 0 {
		t.Fatal("end of modern editor stylesheet block not found")
	}
	return css[start : start+end]
}

func cssSelectorIsInsideAtRule(css string, selectorStart int, atRule string) bool {
	for searchFrom := 0; searchFrom < selectorStart; {
		relativeStart := strings.Index(css[searchFrom:selectorStart], atRule)
		if relativeStart < 0 {
			return false
		}
		atRuleStart := searchFrom + relativeStart
		openRelative := strings.Index(css[atRuleStart:], "{")
		if openRelative < 0 {
			return false
		}
		open := atRuleStart + openRelative
		depth := 0
		for i := open; i < len(css); i++ {
			switch css[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					if selectorStart > open && selectorStart < i {
						return true
					}
					searchFrom = i + 1
					goto nextAtRule
				}
			}
		}
		return false

	nextAtRule:
	}
	return false
}
