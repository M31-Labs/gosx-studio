package canvaswasmfreeruntime

import (
	"strings"
	"testing"
)

func legacyMuddyGlobal(name string) string {
	return "__" + "muddy" + name
}

func TestCanvas2DPainterScriptContract(t *testing.T) {
	body := string(Canvas2DPainterScript())
	if body == "" {
		t.Fatal("Canvas2DPainterScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvas2DPainterRuntime",
		"paint: paintCanvasBundle",
		"screenTransform: canvasBoardScreenTransform",
		"renderCanvasBoardHTML",
		"renderCanvasBoardThumbnails",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("Canvas2DPainterScript() missing %q", check)
		}
	}
	if forbidden := legacyMuddyGlobal("Canvas2DPainter"); strings.Contains(body, forbidden) {
		t.Fatalf("Canvas2DPainterScript() should not contain %q", forbidden)
	}
}

func TestCanvasWASMFreeClientScriptContract(t *testing.T) {
	body := string(CanvasWASMFreeClientScript())
	if body == "" {
		t.Fatal("CanvasWASMFreeClientScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasWASMFreeClientRuntime",
		"window.GoSXStudioCanvas2DPainterRuntime",
		"window.GoSXStudioCanvasInlineEditRuntime",
		"window.GoSXStudioCanvasContextualPanelRuntime",
		"canvas.__gosxStudioCanvasWASMFree",
		"canvas.GoSXStudioCanvasWasmFree",
		"data-gosx-canvas-wasm-free-client",
		"mountAll: mountAll",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasWASMFreeClientScript() missing %q", check)
		}
	}
	for _, check := range []string{
		legacyMuddyGlobal("CanvasWasmFreeClient"),
		legacyMuddyGlobal("Canvas2DPainter"),
		legacyMuddyGlobal("CanvasInlineEdit"),
		legacyMuddyGlobal("CanvasContextualPanel"),
		legacyMuddyGlobal("CanvasWasmFree"),
	} {
		if strings.Contains(body, check) {
			t.Fatalf("CanvasWASMFreeClientScript() should not contain %q", check)
		}
	}
}

func TestCanvasWASMFreeClientScriptNoHardcodedMuddyAuthoringRoute(t *testing.T) {
	body := string(CanvasWASMFreeClientScript())
	route := "/admin/editor/" + "__actions/authoring"
	if strings.Contains(body, route) {
		t.Fatalf("CanvasWASMFreeClientScript() must not hardcode the Muddy authoring route")
	}
}
