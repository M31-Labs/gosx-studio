package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/hostruntime"
)

func TestBackofficeContentEditorScriptsUseCanonicalStudioRuntimePaths(t *testing.T) {
	for name, node := range map[string]gosx.Node{
		"page-detail": RenderBackendPageDetailScripts(),
		"page-index":  RenderBackendPageIndexScripts(),
		"blog-detail": RenderBackendBlogDetailScripts(),
		"blog-index":  RenderBackendBlogIndexScripts(),
	} {
		html := gosx.RenderHTML(node)
		if !strings.Contains(html, `src="`+hostruntime.ContentEditorRuntimePath+`"`) {
			t.Fatalf("%s scripts missing canonical content editor path %q: %s", name, hostruntime.ContentEditorRuntimePath, html)
		}
		if !strings.Contains(html, `src="`+hostruntime.MediaRuntimePath+`"`) {
			t.Fatalf("%s scripts missing canonical media runtime path %q: %s", name, hostruntime.MediaRuntimePath, html)
		}
		if strings.Contains(html, `src="/content-editor.js"`) || strings.Contains(html, `src="/media-picker.js"`) {
			t.Fatalf("%s scripts still reference legacy root runtime paths: %s", name, html)
		}
		for _, marker := range []string{
			`data-gosx-script="managed"`,
			`data-gosx-script-load="dom"`,
			`data-gosx-script-loaded="pending"`,
		} {
			if !strings.Contains(html, marker) {
				t.Fatalf("%s scripts missing GoSX managed-script marker %q: %s", name, marker, html)
			}
		}
		for _, src := range []string{hostruntime.ContentEditorRuntimePath, hostruntime.MediaRuntimePath} {
			if got := strings.Count(html, `src="`+src+`"`); got != 1 {
				t.Fatalf("%s scripts must emit exactly one %q tag, got %d: %s", name, src, got, html)
			}
		}
	}
}

func TestBackofficeMediaOnlyScriptsUseCanonicalStudioRuntimePath(t *testing.T) {
	for name, node := range map[string]gosx.Node{
		"gallery-detail": RenderBackendGalleryDetailScripts(),
		"gallery-index":  RenderBackendGalleryIndexScripts(),
		"product-detail": RenderBackendProductDetailScripts(),
		"product-index":  RenderBackendProductIndexScripts(),
		"settings":       RenderBackendSettingsScripts(),
		"media-library":  RenderBackendMediaLibraryScripts(),
	} {
		html := gosx.RenderHTML(node)
		if !strings.Contains(html, `src="`+hostruntime.MediaRuntimePath+`"`) {
			t.Fatalf("%s scripts missing canonical media runtime path %q: %s", name, hostruntime.MediaRuntimePath, html)
		}
		if strings.Contains(html, `src="/media-picker.js"`) {
			t.Fatalf("%s scripts still reference legacy media picker path: %s", name, html)
		}
		for _, marker := range []string{
			`data-gosx-script="managed"`,
			`data-gosx-script-load="dom"`,
			`data-gosx-script-loaded="pending"`,
		} {
			if !strings.Contains(html, marker) {
				t.Fatalf("%s scripts missing GoSX managed-script marker %q: %s", name, marker, html)
			}
		}
		if got := strings.Count(html, `src="`+hostruntime.MediaRuntimePath+`"`); got != 1 {
			t.Fatalf("%s scripts must emit exactly one media runtime tag, got %d: %s", name, got, html)
		}
	}
}
