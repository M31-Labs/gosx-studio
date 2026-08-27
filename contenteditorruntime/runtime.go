// Package contenteditorruntime embeds the shared Page CMS content editor
// browser runtime.
package contenteditorruntime

import (
	"embed"
	"net/http"

	"m31labs.dev/gosx-studio/internal/runtimeasset"
)

//go:embed content_editor.js
var runtimeFS embed.FS

// Script returns the shared content-editor JavaScript bundle.
func Script() []byte {
	b, _ := runtimeFS.ReadFile("content_editor.js")
	return b
}

// Handler serves Script as a browser JavaScript asset.
func Handler() http.Handler {
	return runtimeasset.Handler("content-editor.js", Script(), "text/javascript; charset=utf-8")
}
