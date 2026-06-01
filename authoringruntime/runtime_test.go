package authoringruntime

import (
	"strings"
	"testing"
)

func TestIslandRuntimeJSOwnsAuthoringResultFeedback(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		"GoSXStudioAuthoringRuntime",
		"gosx:form:result",
		"gosxstudio:authoring-result",
		"data-gosx-studio-authoring-managed",
		"data-gosx-studio-authoring-selected",
		"data-gosx-studio-authoring-state",
		"data-gosx-studio-preview-url",
		"data-gosx-studio-canvas-node",
		"gosxstudio:canvas-select",
		"data-studio-site-map-component",
		"GoSXStudioPreviewRuntime",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing authoring feedback fragment %q", fragment)
		}
	}
}

func TestBundleReturnsAuthoringRuntime(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	if !strings.Contains(bundle, "window.GoSXStudioAuthoringRuntime") {
		t.Fatalf("Bundle() missing runtime global:\n%s", bundle)
	}
}
