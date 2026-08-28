package hostruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMediaRuntimePublicPathAndHandler(t *testing.T) {
	if MediaRuntimePath != "/_gosx/studio/media-runtime.js" {
		t.Fatalf("media runtime path = %q", MediaRuntimePath)
	}
	script := string(MediaRuntimeScript())
	for _, want := range []string{
		"GoSXStudioMediaRuntime",
		"application/x-gosx-studio-media",
		"data-media-upload-drop-zone",
		"gosxstudio:content-editor-render",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("media runtime script missing %q", want)
		}
	}

	rec := httptest.NewRecorder()
	MediaRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", MediaRuntimePath, nil))
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
