// Package mediaruntime embeds the shared Studio media picker browser runtime.
package mediaruntime

import (
	"embed"
	"net/http"
)

//go:embed media_runtime.js
var runtimeFS embed.FS

// Script returns the shared media picker JavaScript bundle.
func Script() []byte {
	b, _ := runtimeFS.ReadFile("media_runtime.js")
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
