package sectionorderruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScriptContainsSectionOrderContracts(t *testing.T) {
	script := string(Script())
	for _, want := range []string{
		"GoSXStudioSectionOrderRuntime",
		"data-gosx-studio-section-order",
		"data-gosx-studio-section-key",
		"data-gosx-studio-section-order-preview-handle",
		"data-gosx-studio-section-order-move",
		"data-gosx-studio-section-order-history",
		"home.sections.order",
		"set-field",
		"GoSXStudioOperationRuntime",
		"JSON.stringify(nextOrder)",
		"pointercancel",
		"Escape",
		"sameSet(order, nextOrder)",
		"syncOrderBindings",
		"createDetachedForm",
		"copyCSRFToken",
		"gosx_studio_operation_id",
		"listen(frame, \"load\"",
		"data-gosx-studio-section-order-preview-optional",
		"Some optional sections are hidden",
		"pending",
		"historyShortcut",
		"stopImmediatePropagation",
		"editableShortcutTarget",
		"data-studio-target-component",
		"expectedTargetValue",
		"committedOrder",
		"your reorder remains unsaved",
		"gosx:navigate",
		"gosx:render",
		"removeEventListener",
		"dispose",
		"isConnected",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("section order runtime missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"orderFromDOM",
		"cloud-slip-bowl",
		"/shop",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("section order runtime contains forbidden host-specific fragment %q", forbidden)
		}
	}
}

func TestHandlerServesJavaScript(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/_gosx/studio/section-order-runtime.js", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Fatalf("content type = %q", ct)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("nosniff = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "GoSXStudioSectionOrderRuntime") {
		t.Fatalf("handler body missing runtime global")
	}
}
