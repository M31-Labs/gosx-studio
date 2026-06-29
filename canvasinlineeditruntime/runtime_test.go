package canvasinlineeditruntime

import (
	"strings"
	"testing"
)

func legacyMuddyGlobal(name string) string {
	return "__" + "muddy" + name
}

func TestCanvasInlineEditScriptContract(t *testing.T) {
	body := string(CanvasInlineEditScript())
	if body == "" {
		t.Fatal("CanvasInlineEditScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasInlineEditRuntime",
		"GoSXStudioInlineEditRuntime",
		"__gosxStudioCanvasInlineEditPendingInstall",
		"return false",
		"persistRepaintSafe",
		"data-gosx-canvas-bundle",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasInlineEditScript() missing %q", check)
		}
	}
	for _, check := range []string{"window." + legacyMuddyGlobal("CanvasInlineEdit"), legacyMuddyGlobal("CanvasInlineEdit")} {
		if strings.Contains(body, check) {
			t.Fatalf("CanvasInlineEditScript() should not contain %q", check)
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
		"GoSXStudioCanvasInlineEditRuntime",
		"data-studio-site-map-canvas-default",
		"data-gosx-canvas-html",
		"installed === false",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasDefaultInlineInstallerScript() missing %q", check)
		}
	}
	if forbidden := legacyMuddyGlobal("CanvasDefaultInlineInstaller"); strings.Contains(body, forbidden) {
		t.Fatalf("CanvasDefaultInlineInstallerScript() should not contain %q", forbidden)
	}
}
