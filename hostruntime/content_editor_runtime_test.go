package hostruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContentEditorRuntimePathAndHandler(t *testing.T) {
	if ContentEditorRuntimePath != RuntimeRoot+"/content-editor.js" {
		t.Fatalf("content editor runtime path = %q", ContentEditorRuntimePath)
	}
	if len(ContentEditorRuntimeScript()) == 0 {
		t.Fatalf("content editor runtime script is empty")
	}

	rec := httptest.NewRecorder()
	ContentEditorRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", ContentEditorRuntimePath, nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Fatalf("content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "GoSXStudioContentEditorRuntime") {
		t.Fatalf("handler body missing shared runtime global")
	}
}
