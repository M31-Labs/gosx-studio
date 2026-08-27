package backoffice

import "m31labs.dev/gosx"

const (
	backendContentEditorRuntimePath = "/_gosx/studio/content-editor.js"
	backendMediaRuntimePath         = "/_gosx/studio/media-runtime.js"
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
	return gosx.Fragment(
		renderBackendManagedStudioScript(backendContentEditorRuntimePath),
		renderBackendManagedStudioScript(backendMediaRuntimePath),
	)
}

func renderBackendManagedStudioScript(src string) gosx.Node {
	return gosx.El("script", gosx.Attrs(
		gosx.Attr("src", src),
		gosx.Attr("data-gosx-script", "managed"),
		gosx.Attr("data-gosx-script-load", "dom"),
		gosx.Attr("data-gosx-script-loaded", "pending"),
		gosx.Attr("defer", true),
	))
}
