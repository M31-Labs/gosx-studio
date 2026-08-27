package backoffice

import (
	"net/url"
	"runtime/debug"
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestManagedStudioReleaseVersionUsesOnlyReleasedStudioMetadata(t *testing.T) {
	released := "v0.6.2-0.20260827190001-8a4c9b8de356"
	releasedWithBuildMetadata := released + "+incompatible"

	tests := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{name: "nil metadata", info: nil},
		{name: "missing metadata", info: &debug.BuildInfo{}},
		{
			name: "unrelated dependency",
			info: &debug.BuildInfo{Deps: []*debug.Module{{Path: "example.com/other", Version: "v9.9.9"}}},
		},
		{
			name: "released dependency",
			info: &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: released}}},
			want: released,
		},
		{
			name: "released main module",
			info: &debug.BuildInfo{Main: debug.Module{Path: studioModulePath, Version: released}},
			want: released,
		},
		{
			name: "development dependency",
			info: &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: "(devel)"}}},
		},
		{
			name: "development main module",
			info: &debug.BuildInfo{Main: debug.Module{Path: studioModulePath, Version: "(devel)"}},
		},
		{
			name: "local dependency replacement",
			info: &debug.BuildInfo{Deps: []*debug.Module{{
				Path:    studioModulePath,
				Version: released,
				Replace: &debug.Module{Path: "/workspace/gosx-studio"},
			}}},
		},
		{
			name: "local main replacement",
			info: &debug.BuildInfo{Main: debug.Module{
				Path:    studioModulePath,
				Version: released,
				Replace: &debug.Module{Path: "/workspace/gosx-studio"},
			}},
		},
		{
			name: "invalid version characters",
			info: &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: "v0.6.2?cache=bad"}}},
		},
		{
			name: "version with whitespace",
			info: &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: " v0.6.2"}}},
		},
		{
			name: "build metadata is escaped",
			info: &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: releasedWithBuildMetadata}}},
			want: releasedWithBuildMetadata,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := managedStudioReleaseVersion(test.info); got != test.want {
				t.Fatalf("managedStudioReleaseVersion() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestManagedStudioReleaseVersionChangesWithPinnedVersion(t *testing.T) {
	first := &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: "v0.6.2-0.20260827190001-8a4c9b8de356"}}}
	second := &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: "v0.6.2-0.20260827205410-ae0c95bc7d47"}}}

	firstVersion := managedStudioReleaseVersion(first)
	secondVersion := managedStudioReleaseVersion(second)
	if firstVersion == "" || secondVersion == "" {
		t.Fatalf("release metadata unexpectedly fell back: first=%q second=%q", firstVersion, secondVersion)
	}
	if firstVersion == secondVersion {
		t.Fatalf("different pinned versions produced the same cache key: %q", firstVersion)
	}
}

func TestRenderManagedStudioScriptsUsesEscapedReleaseVersion(t *testing.T) {
	released := "v0.6.2-0.20260827190001-8a4c9b8de356+incompatible"
	info := &debug.BuildInfo{Deps: []*debug.Module{{Path: studioModulePath, Version: released}}}
	html := gosx.RenderHTML(renderBackendManagedStudioScriptsForBuildInfo(info))
	encoded := url.QueryEscape(released)

	for _, path := range []string{backendContentEditorRuntimePath, backendMediaRuntimePath} {
		want := `src="` + path + `?v=` + encoded + `"`
		if got := strings.Count(html, want); got != 1 {
			t.Fatalf("rendered %q count = %d, want 1 (%s): %s", path, got, want, html)
		}
	}
	if strings.Contains(html, `src="`+backendContentEditorRuntimePath+`"`) || strings.Contains(html, `src="`+backendMediaRuntimePath+`"`) {
		t.Fatalf("released managed scripts unexpectedly rendered an unversioned URL: %s", html)
	}
	for _, marker := range []string{
		`data-gosx-script="managed"`,
		`data-gosx-script-load="dom"`,
		`data-gosx-script-loaded="pending"`,
		`defer`,
	} {
		if got := strings.Count(html, marker); got != 2 {
			t.Fatalf("marker %q count = %d, want 2: %s", marker, got, html)
		}
	}
}

func TestRenderManagedStudioScriptPreservesMediaSingletonAndFallback(t *testing.T) {
	released := "v0.6.2-0.20260827190001-8a4c9b8de356"
	versioned := gosx.RenderHTML(renderBackendManagedStudioScriptForVersion(backendMediaRuntimePath, released))
	if got := strings.Count(versioned, `src="`+backendMediaRuntimePath+`?v=`+url.QueryEscape(released)+`"`); got != 1 {
		t.Fatalf("versioned media script count = %d, want 1: %s", got, versioned)
	}
	if strings.Contains(versioned, backendContentEditorRuntimePath) {
		t.Fatalf("media-only script unexpectedly rendered content editor runtime: %s", versioned)
	}

	fallback := gosx.RenderHTML(renderBackendManagedStudioScriptsForBuildInfo(nil))
	for _, path := range []string{backendContentEditorRuntimePath, backendMediaRuntimePath} {
		if got := strings.Count(fallback, `src="`+path+`"`); got != 1 {
			t.Fatalf("fallback %q count = %d, want 1: %s", path, got, fallback)
		}
		if strings.Contains(fallback, `src="`+path+`?v=`) {
			t.Fatalf("fallback %q unexpectedly has a version query: %s", path, fallback)
		}
	}
}

func TestManagedStudioAssetHrefPreservesExistingQuery(t *testing.T) {
	if got := managedStudioAssetHref("/_gosx/studio/runtime.js?debug=true", "v1.2.3+meta"); got != "/_gosx/studio/runtime.js?debug=true&v=v1.2.3%2Bmeta" {
		t.Fatalf("managedStudioAssetHref() = %q", got)
	}
}
