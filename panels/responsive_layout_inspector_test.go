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

// TestResponsiveLayoutInspectorStyleStateAttributeDoesNotCollideWithFormStateRuntime
// guards against regressing the attribute-name collision fixed alongside this
// test: hostruntime/assets/state_runtime.js's initAll() selects
// document.querySelectorAll("[data-gosx-studio-state]") expecting only the
// workbench's own <form data-gosx-studio-state="true"> (shell/workbench_frame.go).
// This panel's style-state buttons must use a distinct attribute name
// (data-gosx-studio-style-state) so state_runtime.js never matches them and
// calls initForm/formSignature on a non-form element (whose .elements is
// undefined, throwing "forEach called on null or undefined").
func TestResponsiveLayoutInspectorStyleStateAttributeDoesNotCollideWithFormStateRuntime(t *testing.T) {
	node := RenderResponsiveLayoutInspector(ResponsiveLayoutInspectorOptions{
		ID: "layout", Action: "/authoring", CSRFToken: "token", DocumentRevision: 9,
		Target: authoring.OperationTarget{Route: "/", PageID: "page:home", ComponentKey: "home:hero"},
		Values: map[string]ResponsiveLayoutValue{
			"base/gap": {Present: true, Value: "var(--space-md)", TargetHead: "head-gap", UndoOperationID: "op-gap"},
		},
	})
	html := gosx.RenderHTML(node)
	if !strings.Contains(html, "data-gosx-studio-style-state=") {
		t.Fatal("expected style-state buttons to carry the non-colliding data-gosx-studio-style-state attribute")
	}
	if strings.Contains(html, "data-gosx-studio-state=") {
		t.Fatal("data-gosx-studio-state must remain reserved for the form-state runtime's own <form> element; the layout inspector must not emit it")
	}
}

// TestResponsiveLayoutInspectorCollapsesControlsByDefaultWithSummary guards
// wave 3A fix (c): the breakpoint controls (base/tablet/mobile, each with
// several property selects) must sit behind one closed-by-default
// disclosure, with the heading and a one-line override-count summary staying
// outside it so a collapsed panel is still scannable.
func TestResponsiveLayoutInspectorCollapsesControlsByDefaultWithSummary(t *testing.T) {
	node := RenderResponsiveLayoutInspector(ResponsiveLayoutInspectorOptions{
		ID: "layout", Action: "/authoring", CSRFToken: "token", DocumentRevision: 9,
		Target: authoring.OperationTarget{Route: "/", PageID: "page:home", ComponentKey: "home:hero"},
		Values: map[string]ResponsiveLayoutValue{
			"base/gap":            {Present: true, Value: "var(--space-md)"},
			"base/flex-direction": {Present: true, Value: "row"},
		},
	})
	html := gosx.RenderHTML(node)
	for _, want := range []string{
		`<header class="studio-responsive-layout__head"><h3>Responsive layout</h3>`,
		`data-studio-layout-summary="true">home:hero · 2 overrides</span>`,
		`<details class="studio-responsive-layout__disclosure">`,
		`<summary>Edit layout</summary>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, `<details class="studio-responsive-layout__disclosure" open`) {
		t.Fatal("the layout-controls disclosure must be closed by default")
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

// TestResponsiveLayoutInspectorHiddenOptionControlsInitialVisibility guards the
// multi-target composition contract: a host rendering more than one target's
// inspector marks every non-default instance Hidden so a user without working
// JS never sees two phone-only panels at once; scopeResponsiveLayoutInspectors
// (hostruntime/assets/workbench_runtime.js) toggles [hidden] on selection.
func TestResponsiveLayoutInspectorHiddenOptionControlsInitialVisibility(t *testing.T) {
	options := ResponsiveLayoutInspectorOptions{
		ID: "layout", Action: "/authoring",
		Target: authoring.OperationTarget{Route: "/about", PageID: "page:about", ComponentKey: "about:content"},
	}
	visible := gosx.RenderHTML(RenderResponsiveLayoutInspector(options))
	if strings.Contains(visible, "hidden=\"hidden\"") {
		t.Fatal("default (non-Hidden) inspector must render without a hidden attribute")
	}
	options.Hidden = true
	hiddenHTML := gosx.RenderHTML(RenderResponsiveLayoutInspector(options))
	section := hiddenHTML[:strings.Index(hiddenHTML, ">")+1]
	if !strings.Contains(section, "hidden=\"hidden\"") {
		t.Fatalf("Hidden inspector must carry the hidden attribute on its outer section, got %q", section)
	}
	if !strings.Contains(section, "data-studio-responsive-layout-inspector=\"true\"") {
		t.Fatal("hidden attribute must live on the [data-studio-responsive-layout-inspector] element the runtime toggles")
	}
}
