package hostruntime

import (
	"strings"
	"testing"
)

// TestEnterpriseEditorPolishStylesheetContract locks the small, reusable
// quality surface owned by this slice. Browser tests prove behavior and
// geometry; this test makes sure the served stylesheet cannot silently lose
// the accessibility and dense-media primitives those fixtures exercise.
func TestEnterpriseEditorPolishStylesheetContract(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		"@media (forced-colors: active)",
		"forced-color-adjust: auto;",
		"@media (prefers-reduced-motion: reduce)",
		"grid-template-columns: max-content minmax(0, 1fr) max-content;",
		"[data-content-editor-save-state=\"conflict\"]",
		".content-editor__save-errors",
		".content-editor__save-conflict-link",
		"grid-column: 1 / -1;",
		".media-picker__asset[data-media-asset-selected=\"true\"]",
		".media-list-editor",
		".media-list-editor__items",
		".media-list-editor__item",
		".media-list-editor__fields",
		"overflow-wrap: anywhere;",
		"grid-template-columns: minmax(0, 1fr);",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("enterprise editor polish stylesheet missing %q", want)
		}
	}
}

func TestEnterpriseEditorPolishControlsUseAccessibleTargetSizes(t *testing.T) {
	css := string(Stylesheet())
	for _, want := range []string{
		"min-width: calc(var(--space-md) + var(--space-sm) + var(--space-chip-y));",
		"min-height: calc(var(--space-md) + var(--space-sm) + var(--space-chip-y));",
		"outline: 2px solid Highlight;",
		"outline-offset: 2px;",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("enterprise editor polish stylesheet missing accessible control guard %q", want)
		}
	}
}

func TestEnterpriseEditorPolishHeaderControlsKeepMinimumAfterToolbarShrink(t *testing.T) {
	css := string(Stylesheet())
	headerControlSelector := `.content-block__header
  > :where(`
	baseStart := strings.Index(css, headerControlSelector)
	if baseStart < 0 {
		t.Fatalf("scoped content-block header control selector missing")
	}
	shrinkStart := strings.Index(css, `.content-block__header
  > *,`)
	if shrinkStart < 0 || baseStart <= shrinkStart {
		t.Fatalf("scoped header minimum must follow the generic min-width reset")
	}
	baseEnd := strings.Index(css[baseStart:], "}")
	if baseEnd < 0 {
		t.Fatal("scoped header minimum rule is not closed")
	}
	baseRule := css[baseStart : baseStart+baseEnd]
	for _, want := range []string{
		`.content-block__drag-handle`,
		`.content-block__move`,
		`[data-content-drag-handle]`,
		`[data-content-editor-action]`,
		`min-width: var(--space-md);`,
	} {
		if !strings.Contains(baseRule, want) {
			t.Fatalf("scoped header minimum rule missing %q", want)
		}
	}

	coarseStart := strings.LastIndex(css, "@media (pointer: coarse) {")
	if coarseStart < 0 {
		t.Fatal("content editor coarse-pointer media query missing")
	}
	coarseHeaderSelector := `.content-block__header
    > :where(`
	coarseHeaderStart := strings.Index(css[coarseStart:], coarseHeaderSelector)
	if coarseHeaderStart < 0 {
		t.Fatal("coarse-pointer scoped header control selector missing")
	}
	coarseHeaderStart += coarseStart
	coarseGenericStart := strings.Index(css[coarseStart:], `[data-content-editor-action],`)
	if coarseGenericStart < 0 || coarseHeaderStart <= coarseStart+coarseGenericStart {
		t.Fatal("coarse scoped header minimum must follow the generic coarse control rule")
	}
	coarseEnd := strings.Index(css[coarseHeaderStart:], "}")
	if coarseEnd < 0 {
		t.Fatal("coarse-pointer scoped header minimum rule is not closed")
	}
	coarseRule := css[coarseHeaderStart : coarseHeaderStart+coarseEnd]
	for _, want := range []string{
		"min-width: calc(var(--space-md) + var(--space-sm) + var(--space-chip-y));",
		"min-height: calc(var(--space-md) + var(--space-sm) + var(--space-chip-y));",
	} {
		if !strings.Contains(coarseRule, want) {
			t.Fatalf("coarse-pointer scoped header minimum must retain the enlarged target expression %q", want)
		}
	}
}

func TestEnterpriseEditorPolishAdvancedGroupsHaveRevealAndSelectedLabelSelectors(t *testing.T) {
	css := string(Stylesheet())
	groups := []string{"flows", "tools", "schema", "schedule", "typography", "settings"}
	for _, group := range groups {
		reveal := `.studio-advanced-panel__group-input[value="` + group + `"]:checked ~ [data-studio-advanced-group-slot="` + group + `"]`
		selectedLabel := `.studio-advanced-panel__group-input[value="` + group + `"]:checked ~ .studio-advanced-panel__groups [data-studio-advanced-group-label="` + group + `"]`
		for _, want := range []string{reveal, selectedLabel} {
			if !strings.Contains(css, want) {
				t.Fatalf("Advanced group %q selector missing %q", group, want)
			}
		}
	}
}

func TestEnterpriseEditorPolishReducedMotionDisablesSharedSurfaceTransitions(t *testing.T) {
	css := string(Stylesheet())
	marker := "/* Enterprise editor quality guards."
	markerStart := strings.Index(css, marker)
	if markerStart < 0 {
		t.Fatal("enterprise editor quality guard marker missing")
	}
	start := strings.Index(css[markerStart:], "@media (prefers-reduced-motion: reduce)")
	if start < 0 {
		t.Fatal("reduced-motion media query missing")
	}
	block := css[markerStart+start:]
	for _, want := range []string{
		".content-block",
		".media-picker__asset",
		".media-picker__drop-zone",
		".button",
		"transition: none;",
		"transform: none;",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("reduced-motion quality guard missing %q", want)
		}
	}
}
