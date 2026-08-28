package hostruntime

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAssetHandlerUsesContentValidatorsForMutableURLs(t *testing.T) {
	const contentType = "text/javascript; charset=utf-8"
	oldPayload := []byte("runtime-v1")
	oldHandler := AssetHandler("runtime.js", oldPayload, contentType)
	oldPayload[0] = 'X'

	first := serveAssetRequest(oldHandler, httptest.NewRequest(http.MethodGet, "/_gosx/studio/runtime.js", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusOK)
	}
	if got := first.Body.Bytes(); !bytes.Equal(got, []byte("runtime-v1")) {
		t.Fatalf("first body = %q, want original snapshot", got)
	}
	assertAssetHeaders(t, first, contentType)
	oldETag := first.Header().Get("ETag")
	if oldETag == "" || !strings.HasPrefix(oldETag, `W/"`) {
		t.Fatalf("old ETag = %q, want weak validator", oldETag)
	}
	if got := first.Header().Get("Last-Modified"); got != "" {
		t.Fatalf("Last-Modified = %q, want absent", got)
	}

	newHandler := AssetHandler("runtime.js", []byte("runtime-v2"), contentType)
	changedRequest := httptest.NewRequest(http.MethodGet, "/_gosx/studio/runtime.js", nil)
	changedRequest.Header.Set("If-Modified-Since", time.Now().UTC().Add(24*time.Hour).Format(http.TimeFormat))
	changedRequest.Header.Set("If-None-Match", oldETag)
	changed := serveAssetRequest(newHandler, changedRequest)
	if changed.Code != http.StatusOK {
		t.Fatalf("changed status = %d, want %d", changed.Code, http.StatusOK)
	}
	if got := changed.Body.String(); got != "runtime-v2" {
		t.Fatalf("changed body = %q, want new payload", got)
	}
	newETag := changed.Header().Get("ETag")
	if newETag == "" || newETag == oldETag {
		t.Fatalf("new ETag = %q, old ETag = %q; want changed validator", newETag, oldETag)
	}
	if got := changed.Header().Get("Last-Modified"); got != "" {
		t.Fatalf("changed Last-Modified = %q, want absent", got)
	}

	for name, modifiedSince := range map[string]time.Time{
		"old-fixed-modtime": time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		"future":            time.Now().UTC().Add(24 * time.Hour),
	} {
		t.Run("ims-only-"+name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/_gosx/studio/runtime.js", nil)
			req.Header.Set("If-Modified-Since", modifiedSince.Format(http.TimeFormat))
			rec := serveAssetRequest(newHandler, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Body.String(); got != "runtime-v2" {
				t.Fatalf("body = %q, want new payload", got)
			}
		})
	}

	for name, match := range map[string]string{
		"weak": newETag,
		"list": `"stale", ` + newETag + `, "also-stale"`,
		"star": "*",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/_gosx/studio/runtime.js", nil)
			req.Header.Set("If-None-Match", match)
			rec := serveAssetRequest(newHandler, req)
			if rec.Code != http.StatusNotModified {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotModified)
			}
			if rec.Body.Len() != 0 {
				t.Fatalf("304 body length = %d, want zero", rec.Body.Len())
			}
			if got := rec.Header().Get("ETag"); got != newETag {
				t.Fatalf("304 ETag = %q, want %q", got, newETag)
			}
			if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
				t.Fatalf("304 Cache-Control = %q, want no-cache", got)
			}
		})
	}

	head := httptest.NewRecorder()
	newHandler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/_gosx/studio/runtime.js", nil))
	if head.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want %d", head.Code, http.StatusOK)
	}
	if head.Body.Len() != 0 {
		t.Fatalf("HEAD body length = %d, want zero", head.Body.Len())
	}
	if got := head.Header().Get("ETag"); got != newETag {
		t.Fatalf("HEAD ETag = %q, want %q", got, newETag)
	}

	weakIfRange := httptest.NewRecorder()
	rangeRequest := httptest.NewRequest(http.MethodGet, "/_gosx/studio/runtime.js", nil)
	rangeRequest.Header.Set("Range", "bytes=0-3")
	rangeRequest.Header.Set("If-Range", newETag)
	newHandler.ServeHTTP(weakIfRange, rangeRequest)
	if weakIfRange.Code != http.StatusOK {
		t.Fatalf("weak If-Range status = %d, want %d", weakIfRange.Code, http.StatusOK)
	}
	if got := weakIfRange.Body.String(); got != "runtime-v2" {
		t.Fatalf("weak If-Range body = %q, want full payload", got)
	}
}

func TestAssetHandlerPreservesRangeSemantics(t *testing.T) {
	handler := AssetHandler("runtime.js", []byte("0123456789"), "text/javascript; charset=utf-8")
	req := httptest.NewRequest(http.MethodGet, "/_gosx/studio/runtime.js", nil)
	req.Header.Set("Range", "bytes=2-5")
	rec := serveAssetRequest(handler, req)
	if rec.Code != http.StatusPartialContent {
		t.Fatalf("range status = %d, want %d", rec.Code, http.StatusPartialContent)
	}
	if got := rec.Body.String(); got != "2345" {
		t.Fatalf("range body = %q, want 2345", got)
	}
	if got := rec.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Fatalf("Content-Range = %q, want bytes 2-5/10", got)
	}
}

func TestAssetHandlerETagMatchesAcrossAliases(t *testing.T) {
	payload := []byte("shared-runtime")
	canonical := serveAssetRequest(
		AssetHandler("canonical.js", payload, "text/javascript; charset=utf-8"),
		httptest.NewRequest(http.MethodGet, "/canonical.js", nil),
	)
	alias := serveAssetRequest(
		AssetHandler("legacy.js", payload, "text/javascript; charset=utf-8"),
		httptest.NewRequest(http.MethodGet, "/legacy.js", nil),
	)
	if canonical.Code != http.StatusOK || alias.Code != http.StatusOK {
		t.Fatalf("canonical/alias statuses = %d/%d, want 200/200", canonical.Code, alias.Code)
	}
	if !bytes.Equal(canonical.Body.Bytes(), alias.Body.Bytes()) {
		t.Fatalf("canonical and alias bodies differ")
	}
	if got, want := alias.Header().Get("ETag"), canonical.Header().Get("ETag"); got != want {
		t.Fatalf("alias ETag = %q, canonical ETag = %q", got, want)
	}
}

func TestAssetHandlerConcurrentRequests(t *testing.T) {
	payload := []byte("concurrent-runtime-payload")
	handler := AssetHandler("runtime.js", payload, "text/javascript; charset=utf-8")
	const workers = 16
	const requestsPerWorker = 20

	var wg sync.WaitGroup
	errCh := make(chan error, workers*requestsPerWorker)
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for request := 0; request < requestsPerWorker; request++ {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/runtime-%d-%d.js", worker, request), nil))
				if rec.Code != http.StatusOK {
					errCh <- fmt.Errorf("request %d/%d status = %d", worker, request, rec.Code)
					continue
				}
				if !bytes.Equal(rec.Body.Bytes(), payload) {
					errCh <- fmt.Errorf("request %d/%d body changed", worker, request)
				}
			}
		}(worker)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Error(err)
	}
}

func serveAssetRequest(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func assertAssetHeaders(t *testing.T, rec *httptest.ResponseRecorder, contentType string) {
	t.Helper()
	if got := rec.Header().Get("Content-Type"); got != contentType {
		t.Fatalf("Content-Type = %q, want %q", got, contentType)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}
