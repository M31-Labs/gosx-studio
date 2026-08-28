// Package collabruntime owns Studio's authenticated collaboration browser
// runtime. Hosts provide only the fixed WebSocket endpoint through rendered
// markup; identity and capabilities arrive from the server hello frame.
package collabruntime

import (
	"embed"
	"net/http"
)

//go:embed island_runtime.js
var runtimeFS embed.FS

func Script() []byte { b, _ := runtimeFS.ReadFile("island_runtime.js"); return b }
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(Script())
	})
}
