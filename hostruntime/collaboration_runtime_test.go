package hostruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEngineBundleIncludesCollaborationRuntime(t *testing.T) {
	bundle := string(EngineRuntimeScript())
	for _, want := range []string{"GoSXStudioCollaborationRuntime", "studio.sync.resume", "studio.selection.submit", "studio.cursor.submit"} {
		if !strings.Contains(bundle, want) {
			t.Fatalf("engine bundle missing %q", want)
		}
	}
}
func TestCollaborationRuntimeHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	CollaborationRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/_gosx/studio/collaboration.js", nil))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "GoSXStudioCollaborationRuntime") {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
