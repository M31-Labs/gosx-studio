// Package contenteditorruntime embeds the shared Page CMS content editor
// browser runtime.
package contenteditorruntime

import (
	"embed"
	"net/http"
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(Script())
	})
}
