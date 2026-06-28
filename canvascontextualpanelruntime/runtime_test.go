package canvascontextualpanelruntime

import (
	"strings"
	"testing"
)

func TestCanvasContextualPanelScriptContract(t *testing.T) {
	body := string(CanvasContextualPanelScript())
	if body == "" {
		t.Fatal("CanvasContextualPanelScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasContextualPanelRuntime",
		"window.__muddyCanvasContextualPanel",
		"window.GoSXStudioCanvasInlineEditRuntime",
		"window.__muddyCanvasInlineEdit",
		"persistRepaintSafe",
		"render: render",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasContextualPanelScript() missing %q", check)
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
