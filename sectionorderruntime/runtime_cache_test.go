package sectionorderruntime

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerSupportsContentValidation(t *testing.T) {
	handler := Handler()
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/_gosx/studio/section-order-runtime.js", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", first.Code, http.StatusOK)
	}
	if first.Body.Len() == 0 {
		t.Fatal("handler body is empty")
	}
	if first.Header().Get("ETag") == "" {
		t.Fatal("handler ETag is empty")
	}
	if got := first.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
	if got := first.Header().Get("Last-Modified"); got != "" {
		t.Fatalf("Last-Modified = %q, want absent", got)
	}

	conditional := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/_gosx/studio/section-order-runtime.js", nil)
	request.Header.Set("If-None-Match", first.Header().Get("ETag"))
	handler.ServeHTTP(conditional, request)
	if conditional.Code != http.StatusNotModified {
		t.Fatalf("conditional status = %d, want %d", conditional.Code, http.StatusNotModified)
	}
	if conditional.Body.Len() != 0 {
		t.Fatalf("conditional body length = %d, want zero", conditional.Body.Len())
	}
}
