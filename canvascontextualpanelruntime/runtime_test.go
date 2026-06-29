package canvascontextualpanelruntime

import (
	"strings"
	"testing"
)

func legacyMuddyGlobal(name string) string {
	return "__" + "muddy" + name
}

func TestCanvasContextualPanelScriptContract(t *testing.T) {
	body := string(CanvasContextualPanelScript())
	if body == "" {
		t.Fatal("CanvasContextualPanelScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasContextualPanelRuntime",
		"window.GoSXStudioCanvasInlineEditRuntime",
		"persistRepaintSafe",
		"render: render",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasContextualPanelScript() missing %q", check)
		}
	}
	for _, check := range []string{legacyMuddyGlobal("CanvasContextualPanel"), legacyMuddyGlobal("CanvasInlineEdit")} {
		if strings.Contains(body, check) {
			t.Fatalf("CanvasContextualPanelScript() should not contain %q", check)
		}
	}
}

func TestCanvasContextualPanelScriptNoHardcodedMuddyAuthoringRoute(t *testing.T) {
	body := string(CanvasContextualPanelScript())
	route := "/admin/editor/" + "__actions/authoring"
	if strings.Contains(body, route) {
		t.Fatalf("CanvasContextualPanelScript() must not hardcode the Muddy authoring route")
	}
}
