// Package sectionorderruntime embeds the shared PageCanvas section ordering
// browser runtime.
package sectionorderruntime

import (
	"embed"
	"net/http"
)

//go:embed section_order_runtime.js
var runtimeFS embed.FS

// Script returns the shared section-order JavaScript bundle.
func Script() []byte {
	b, _ := runtimeFS.ReadFile("section_order_runtime.js")
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
