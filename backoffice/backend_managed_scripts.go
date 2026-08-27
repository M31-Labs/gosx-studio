package backoffice

import (
	"net/url"
	"runtime/debug"
	"strings"

	"m31labs.dev/gosx"
)

const (
	backendContentEditorRuntimePath = "/_gosx/studio/content-editor.js"
	backendMediaRuntimePath         = "/_gosx/studio/media-runtime.js"
	studioModulePath                = "m31labs.dev/gosx-studio"
)

// renderBackendManagedStudioScripts is shared by the Page and Blog index/detail
// renderers. Those renderers are mounted below the application's body shell,
// so their script nodes are body-local rather than direct children of <body>.
// GoSX's navigation collector still finds them recursively, but replaceBody
// only skips direct managed-script children. Marking the server-rendered tag as
// pending makes the loader ignore that inert clone when it looks for an already
// loaded URL, allowing it to create the real DOM-loaded script in <head>.
//
// The marker is intentionally only "pending" on the server-rendered body tag;
// the GoSX loader owns the live tag and changes its marker to "true" after the
// browser has executed it. Keeping one tag per runtime here preserves the
// normal cache/idempotency path on revisits.
func renderBackendManagedStudioScripts() gosx.Node {
	return renderBackendManagedStudioScriptsForBuildInfo(readBuildInfo())
}

func renderBackendManagedStudioScriptsForBuildInfo(info *debug.BuildInfo) gosx.Node {
	version := managedStudioReleaseVersion(info)
	return gosx.Fragment(
		renderBackendManagedStudioScriptForVersion(backendContentEditorRuntimePath, version),
		renderBackendManagedStudioScriptForVersion(backendMediaRuntimePath, version),
	)
}

func renderBackendManagedStudioScript(src string) gosx.Node {
	return renderBackendManagedStudioScriptForVersion(src, managedStudioReleaseVersion(readBuildInfo()))
}

func renderBackendManagedStudioScriptForVersion(src, version string) gosx.Node {
	return gosx.El("script", gosx.Attrs(
		gosx.Attr("src", managedStudioAssetHref(src, version)),
		gosx.Attr("data-gosx-script", "managed"),
		gosx.Attr("data-gosx-script-load", "dom"),
		gosx.Attr("data-gosx-script-loaded", "pending"),
		gosx.Attr("defer", true),
	))
}

func readBuildInfo() *debug.BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	return info
}

// managedStudioReleaseVersion returns the exact released module version for
// this package. A replaced module, a development build, or incomplete build
// metadata must not turn an unrelated version into a cache key: the caller
// will keep the historical unversioned URL in those cases.
func managedStudioReleaseVersion(info *debug.BuildInfo) string {
	if info == nil {
		return ""
	}
	if info.Main.Path == studioModulePath {
		return validStudioModuleVersion(info.Main)
	}
	for _, dependency := range info.Deps {
		if dependency == nil || dependency.Path != studioModulePath {
			continue
		}
		return validStudioModuleVersion(*dependency)
	}
	return ""
}

func validStudioModuleVersion(module debug.Module) string {
	if module.Path != studioModulePath || module.Replace != nil {
		return ""
	}
	version := module.Version
	if version == "" || version == "(devel)" || !strings.HasPrefix(version, "v") {
		return ""
	}
	for _, character := range version {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			strings.ContainsRune(".+-~", character) {
			continue
		}
		return ""
	}
	return version
}

func managedStudioAssetHref(path, version string) string {
	if version == "" {
		return path
	}
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "v=" + url.QueryEscape(version)
}
