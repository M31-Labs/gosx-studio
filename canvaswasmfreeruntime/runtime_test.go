package canvaswasmfreeruntime

import (
	"strings"
	"testing"
)

func TestCanvas2DPainterScriptContract(t *testing.T) {
	body := string(Canvas2DPainterScript())
	if body == "" {
		t.Fatal("Canvas2DPainterScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvas2DPainterRuntime",
		"window.__muddyCanvas2DPainter",
		"paint: paintCanvasBundle",
		"screenTransform: canvasBoardScreenTransform",
		"renderCanvasBoardHTML",
		"renderCanvasBoardThumbnails",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("Canvas2DPainterScript() missing %q", check)
		}
	}
}

func TestCanvasWASMFreeClientScriptContract(t *testing.T) {
	body := string(CanvasWASMFreeClientScript())
	if body == "" {
		t.Fatal("CanvasWASMFreeClientScript() must return a non-empty script")
	}
	for _, check := range []string{
		"window.GoSXStudioCanvasWASMFreeClientRuntime",
		"window.__muddyCanvasWasmFreeClient",
		"window.GoSXStudioCanvas2DPainterRuntime",
		"window.__muddyCanvas2DPainter",
		"window.GoSXStudioCanvasInlineEditRuntime",
		"window.__muddyCanvasInlineEdit",
		"window.GoSXStudioCanvasContextualPanelRuntime",
		"canvas.__gosxStudioCanvasWASMFree",
		"canvas.__muddyCanvasWasmFree",
		"data-gosx-canvas-wasm-free-client",
		"mountAll: mountAll",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("CanvasWASMFreeClientScript() missing %q", check)
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
