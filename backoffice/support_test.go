package backoffice

import (
	"strings"
	"testing"
)

// assertOrder is a frozen, byte-identical copy of the unexported test helper
// of the same name defined in the root package's backend_editor_page_test.go
// (backend_editor_page.go/backend_editor_workbench.go are shell territory
// per the restructure spec — .tiller/scratch/gosx-studio-restructure-spec-v0.1.md,
// Slice 8 — and never move into backoffice, so their test file cannot be
// imported here). backend_blog_detail_test.go, backend_category_detail_test.go,
// backend_gallery_detail_test.go, backend_media_detail_test.go,
// backend_page_detail_test.go, and backend_product_detail_test.go call this
// bare unexported name by its original (pre-restructure) name; this copy
// keeps them compiling unchanged until a later slice gives ordered-fragment
// assertion a single shared home (mirrors panels/support.go's identical
// treatment of shared production helpers, and Slice 6's frozen-copy
// treatment of shared test fixtures for backend_editor_workbench_test.go).
func assertOrder(t *testing.T, html string, fragments ...string) {
	t.Helper()
	last := -1
	for _, fragment := range fragments {
		next := strings.Index(html, fragment)
		if next < 0 {
			t.Fatalf("missing ordered fragment %q:\n%s", fragment, html)
		}
		if next <= last {
			t.Fatalf("fragment %q rendered out of order:\n%s", fragment, html)
		}
		last = next
	}
}
