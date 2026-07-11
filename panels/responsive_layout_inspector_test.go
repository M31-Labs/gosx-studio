package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
)

func TestResponsiveLayoutInspectorRendersSelectedTargetTypedControlsAndInheritance(t *testing.T) {
	node := RenderResponsiveLayoutInspector(ResponsiveLayoutInspectorOptions{
		ID: "layout", Action: "/authoring", CSRFToken: "token", DocumentRevision: 9,
		Target: authoring.OperationTarget{Route: "/", PageID: "page:home", ComponentKey: "home:hero"},
		Values: map[string]ResponsiveLayoutValue{
			"base/gap": {Present: true, Value: "var(--space-md)", TargetHead: "head-gap", UndoOperationID: "op-gap"},
		},
	})
	html := gosx.RenderHTML(node)
	for _, want := range []string{
		"data-studio-responsive-layout-inspector=\"true\"", "data-studio-selected-component=\"home:hero\"",
		"data-studio-layout-breakpoint=\"base\"", "data-studio-layout-breakpoint=\"tablet\"", "data-studio-layout-breakpoint=\"mobile\"",
		"grid-template-columns", "data-studio-layout-value=\"true\"", "data-gosx-studio-operation-value-source",
		"Inherited from Base", "Reset to inherited", "Undo layout", "head-gap",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(html, "textarea") || strings.Contains(html, "Selected content") {
		t.Fatal("layout forms must not own or mutate selected content")
	}
}

func TestResponsiveLayoutInspectorMobileInheritsTabletBeforeBase(t *testing.T) {
	control, ok := func() (authoring.LayoutControl, bool) {
		for _, candidate := range authoring.ResponsiveLayoutControls() {
			if candidate.Property == "gap" {
				return candidate, true
			}
		}
		return authoring.LayoutControl{}, false
	}()
	if !ok {
		t.Fatal("gap control missing")
	}
	value, inherited := responsiveLayoutEffective(map[string]ResponsiveLayoutValue{
		"base/gap":   {Present: true, Value: "var(--space-sm)"},
		"tablet/gap": {Present: true, Value: "var(--space-lg)"},
	}, authoring.StyleBreakpointMobile, control)
	if value != "var(--space-lg)" || inherited != authoring.StyleBreakpointTablet {
		t.Fatalf("mobile got %q from %q", value, inherited)
	}
}
