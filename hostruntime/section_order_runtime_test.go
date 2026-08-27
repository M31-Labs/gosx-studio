package hostruntime

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSectionOrderRuntimePathAndHandler(t *testing.T) {
	if SectionOrderRuntimePath != RuntimeRoot+"/section-order-runtime.js" {
		t.Fatalf("section order runtime path = %q", SectionOrderRuntimePath)
	}
	if len(SectionOrderRuntimeScript()) == 0 {
		t.Fatalf("section order runtime script is empty")
	}

	rec := httptest.NewRecorder()
	SectionOrderRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", SectionOrderRuntimePath, nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Fatalf("content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "GoSXStudioSectionOrderRuntime") {
		t.Fatalf("handler body missing shared runtime global")
	}
}
