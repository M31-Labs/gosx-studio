package hostruntime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHostRuntimeEngineExportsSharedRuntimeGlobals(t *testing.T) {
	script := string(EngineRuntimeScript())
	for _, global := range []string{
		"window.GoSXStudioFieldRuntime =",
		"window.GoSXStudioSelectionRuntime =",
		"window.GoSXStudioWorkbenchRuntime =",
	} {
		if !strings.Contains(script, global) {
			t.Fatalf("engine runtime missing exported global %q", global)
		}
	}
}

func TestHostRuntimeWorkbenchPreservesSharedChromeContract(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, marker := range []string{
		"window.GoSXStudioWorkbenchRuntime",
		"form[data-studio-workbench], form[data-editor-workbench]",
		"[data-studio-mode-control]",
		"[data-studio-viewport]",
		"[data-studio-zoom]",
		"[data-studio-rail-toggle]",
		"[data-studio-focus-toggle]",
		"[data-studio-activity-toggle]",
		"[data-studio-preview-frame]",
		`emit(form, "gosxstudio:mode-change"`,
		`emit(form, "gosxstudio:viewport-change"`,
		`emit(form, "gosxstudio:zoom-change"`,
		`emit(form, "gosxstudio:rail-change"`,
		`emit(form, "gosxstudio:focus-change"`,
		`emit(form, "gosxstudio:activity-change"`,
		`emit(form, "gosxstudio:workbench-layout"`,
	} {
		if !strings.Contains(script, marker) {
			t.Fatalf("host workbench runtime missing shared chrome marker %q", marker)
		}
	}
}

func TestHostRuntimeWorkbenchKeepsPreviewOwnershipAtDelegationBoundary(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, marker := range []string{
		"window.GoSXStudioFieldRuntime",
		"window.GoSXStudioInlineEditRuntime",
		"window.GoSXStudioSelectionRuntime",
		"window.__gosx_preview_runtime_island_postPatch",
		"window.__gosx_preview_runtime_island_bindFrames",
		"gosxstudio:field-preview-patch-resolve",
	} {
		if !strings.Contains(script, marker) {
			t.Fatalf("host workbench runtime missing delegated preview marker %q", marker)
		}
	}
	for _, forbidden := range []string{
		`emit(form, "gosxstudio:preview-patch"`,
		"function previewDockForFrame(",
		"function applyPreviewPatch(",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("host workbench runtime must not reclaim CMS preview implementation %q", forbidden)
		}
	}
}

func TestAssetHrefPreservesOptionalReleaseIdentity(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "empty", version: "", want: SectionOrderRuntimePath},
		{name: "whitespace", version: "  ", want: SectionOrderRuntimePath},
		{name: "escaped", version: "release 1&beta", want: SectionOrderRuntimePath + "?v=release+1%26beta"},
		{name: "first-release", version: "release-1", want: SectionOrderRuntimePath + "?v=release-1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := AssetHref(SectionOrderRuntimePath, test.version); got != test.want {
				t.Fatalf("AssetHref(%q) = %q, want %q", test.version, got, test.want)
			}
		})
	}
	if first, second := AssetHref(SectionOrderRuntimePath, "release-1"), AssetHref(SectionOrderRuntimePath, "release-2"); first == second {
		t.Fatalf("new release version must change the asset URL: %q", first)
	}
}

func TestMountedSectionOrderRuntimePreservesCacheValidatorsAcrossReleaseURLs(t *testing.T) {
	var mounter recordingMounter
	MountRuntimes(&mounter)

	var handler http.Handler
	for index, pattern := range mounter.patterns {
		if pattern == "GET "+SectionOrderRuntimePath {
			handler = mounter.handlers[index]
			break
		}
	}
	if handler == nil {
		t.Fatalf("mounted section-order runtime handler is missing")
	}

	canonical := serveAssetRequest(handler, httptest.NewRequest(http.MethodGet, SectionOrderRuntimePath, nil))
	versioned := serveAssetRequest(handler, httptest.NewRequest(http.MethodGet, AssetHref(SectionOrderRuntimePath, "release 1"), nil))
	if canonical.Code != http.StatusOK || versioned.Code != http.StatusOK {
		t.Fatalf("canonical/versioned statuses = %d/%d, want 200/200", canonical.Code, versioned.Code)
	}
	if canonical.Body.Len() == 0 || canonical.Body.String() != versioned.Body.String() {
		t.Fatalf("canonical/versioned mounted asset representations differ")
	}
	canonicalETag := canonical.Header().Get("ETag")
	if canonicalETag == "" || versioned.Header().Get("ETag") != canonicalETag {
		t.Fatalf("canonical/versioned ETags = %q/%q, want one stable validator", canonicalETag, versioned.Header().Get("ETag"))
	}
	if got := versioned.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("versioned Cache-Control = %q, want no-cache", got)
	}

	conditional := httptest.NewRequest(http.MethodGet, AssetHref(SectionOrderRuntimePath, "release 1"), nil)
	conditional.Header.Set("If-None-Match", canonicalETag)
	notModified := serveAssetRequest(handler, conditional)
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("versioned conditional status = %d, want 304", notModified.Code)
	}
	if notModified.Body.Len() != 0 {
		t.Fatalf("versioned conditional body length = %d, want zero", notModified.Body.Len())
	}
}
