package mediaruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScriptOwnsSharedMediaPickerAndUploadContracts(t *testing.T) {
	script := string(Script())
	for _, want := range []string{
		"window.GoSXStudioMediaRuntime",
		"existingRuntime.init(document)",
		"document.currentScript",
		`getAttribute("data-gosx-script") !== "managed"`,
		`setAttribute("data-gosx-script-loaded", "true")`,
		"mediaDragType",
		"application/x-gosx-studio-media",
		"data-media-upload-drop-zone",
		"data-media-upload-error",
		"data-media-drop-active",
		"data-media-picker-status",
		"data-media-asset-selected",
		"data-media-item-id",
		"data-media-list-action",
		"aria-controls",
		"aria-expanded",
		"aria-current",
		"aria-live",
		"stopPropagation",
		"gosxstudio:content-editor-render",
		"new MutationObserver",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("media runtime missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"innerHTML",
		"insertAdjacentHTML",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("media runtime should avoid unsafe fragment %q", forbidden)
		}
	}
}

func TestHandlerServesJavaScript(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/_gosx/studio/media-runtime.js", nil))
	if rec.Code != 200 {
		t.Fatalf("handler status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Fatalf("handler content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "GoSXStudioMediaRuntime") {
		t.Fatalf("handler body missing runtime global: %s", rec.Body.String())
	}
}
