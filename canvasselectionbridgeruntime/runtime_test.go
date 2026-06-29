package canvasselectionbridgeruntime

import (
	"strings"
	"testing"
)

func TestCanvasSelectionBridgeScriptContract(t *testing.T) {
	body := string(CanvasSelectionBridgeScript())
	if body == "" {
		t.Fatal("CanvasSelectionBridgeScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasSelectionBridgeRuntime",
		"$surface.event.selectedID",
		"$surface.event.selectedIDs",
		"__gosx_get_shared_signal",
		"GoSXStudioSiteMapRuntime",
		"setState",
		"selectedNode",
		"selectedNodes",
		"data-studio-site-map-board",
		"data-studio-site-map-selected-node",
		"data-gosx-studio-canvas-selection-bridge-bound",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasSelectionBridgeScript() missing %q", check)
		}
	}
}

func TestCanvasSelectionBridgeScriptNoHardcodedMuddyAuthoringRoute(t *testing.T) {
	body := string(CanvasSelectionBridgeScript())
	route := "/admin/editor/" + "__actions/authoring"
	if strings.Contains(body, route) {
		t.Fatalf("CanvasSelectionBridgeScript() must not hardcode the Muddy authoring route")
	}
}
