// Package interactionruntime exposes the browser-side reveal-on-scroll
// runtime that ships with the rendered (preview and published) page. It is
// deliberately separate from operationruntime: that package is the Studio
// editor's own durable-mutation bridge, while this one only ever observes
// scroll position on the site a visitor sees, using the fixed
// data-gosx-interaction* attribute contract core.Interaction.RenderAttrs
// produces. hover-focus-state needs no script at all — see studio.css.
package interactionruntime

import (
	"embed"
	"net/http"
)

//go:embed island_runtime.js
var runtimeFS embed.FS

// Script returns the runtime's JS source.
func Script() []byte { b, _ := runtimeFS.ReadFile("island_runtime.js"); return b }

// Handler serves Script() as a same-origin JS asset.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(Script())
	})
}
