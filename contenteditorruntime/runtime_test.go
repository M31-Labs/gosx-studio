package contenteditorruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScriptExposesSharedContentEditorRuntimeContract(t *testing.T) {
	script := string(Script())
	for _, check := range []string{
		"GoSXStudioContentEditorRuntime",
		"existingRuntime.init(document)",
		"document.currentScript",
		`getAttribute("data-gosx-script") !== "managed"`,
		`setAttribute("data-gosx-script-loaded", "true")`,
		"data-content-drag-handle",
		"data-content-editor-drop-before",
		"data-content-editor-drop-after",
		"data-content-editor-dirty",
		"data-content-editor-source-state",
		"data-content-editor-keyboard-dragging",
		"data-content-editor-keyboard-preview-index",
		"data-content-editor-search-controls",
		"data-content-editor-search-empty",
		"data-content-editor-collapse",
		"data-content-editor-touch-fallback",
		"data-content-editor-selected",
		"focusin",
		"isInteractiveTarget",
		"syncSelection",
		"focusMutationTarget",
		"revealSelectedBlock",
		"editorDOMID",
		"domIDPart",
		"aria-controls",
		"data-content-editor-enhanced-submit",
		"data-content-editor-save-state",
		"data-content-editor-save-status",
		"data-content-editor-save-detail-text",
		"data-content-editor-save-retry",
		"data-content-editor-save-errors",
		"data-content-editor-save-created-link",
		"data-content-editor-save-conflict-link",
		`tabindex: "0"`,
		`createdLink.setAttribute("tabindex", "0")`,
		`conflictLink.setAttribute("tabindex", "0")`,
		"setCreateFormLocked",
		"data-content-editor-create-lock",
		"data-content-editor-create-locked",
		"data-content-editor-revision",
		"markedRevisionFieldNames",
		"applyRevisionToken",
		"revisionRecoveryRequired",
		"pendingSubmitLockRecords",
		"data-content-editor-submit-lock",
		"requestSubmit",
		"focusValidationError",
		"reportValidity",
		"Compare and reconcile",
		"editorStillActive",
		"gosxstudio:content-editor-save-state",
		"payload.ok === true",
		"response.status >= 200 && response.status < 400",
		"function formCSRFToken(formData)",
		`"X-CSRF-Token": csrfToken`,
		"duplicateBlockIDs",
		"duplicate-id",
		"data-content-editor-source-diagnostic",
		"data-content-editor-source-error",
		"Duplicate block IDs detected",
		"Save target must stay on this site",
		"This form has no CSRF token",
		"gosxstudio:content-editor-navigate",
		"initialFormSignature",
		"hasAttribute(\"formaction\")",
		"fieldErrors",
		"createCompleted",
		"previewKeyboardMove",
		"isSaveShortcut",
		"gosx:before-navigate",
		"guardEditorHistoryNavigation",
		"flow",
		"level",
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("content editor runtime missing %q", check)
		}
	}
	for _, forbidden := range []string{
		"innerHTML",
		"/shop",
		"cloud-slip-bowl",
		"orderFromDOM",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("content editor runtime contains forbidden legacy fragment %q", forbidden)
		}
	}
}

func TestHandlerServesJavaScript(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/_gosx/studio/content-editor.js", nil))

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Fatalf("content type = %q", ct)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("nosniff = %q", got)
	}
	if !strings.Contains(rec.Body.String(), "GoSXStudioContentEditorRuntime") {
		t.Fatalf("handler body missing runtime global")
	}
}
