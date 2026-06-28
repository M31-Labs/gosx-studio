package canvasinlineeditruntime

import (
	"strings"
	"testing"
)

func TestCanvasInlineEditScriptContract(t *testing.T) {
	body := string(CanvasInlineEditScript())
	if body == "" {
		t.Fatal("CanvasInlineEditScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasInlineEditRuntime",
		"window.__muddyCanvasInlineEdit",
		"GoSXStudioInlineEditRuntime",
		"persistRepaintSafe",
		"data-gosx-canvas-bundle",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasInlineEditScript() missing %q", check)
		}
	}
}

func TestCanvasInlineEditScriptNoHardcodedMuddyAuthoringRoute(t *testing.T) {
	body := string(CanvasInlineEditScript())
	if strings.Contains(body, "/admin/editor/__actions/authoring") {
		t.Fatalf("CanvasInlineEditScript() must not hardcode the Muddy authoring route")
	}
}

func TestCanvasDefaultInlineInstallerScriptContract(t *testing.T) {
	body := string(CanvasDefaultInlineInstallerScript())
	if body == "" {
		t.Fatal("CanvasDefaultInlineInstallerScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasDefaultInlineInstallerRuntime",
		"window.__muddyCanvasDefaultInlineInstaller",
		"GoSXStudioCanvasInlineEditRuntime",
		"data-studio-site-map-canvas-default",
		"data-gosx-canvas-html",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasDefaultInlineInstallerScript() missing %q", check)
		}
	}
}
