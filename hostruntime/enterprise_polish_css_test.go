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
