// Package mediaruntime embeds the shared Studio media picker browser runtime.
package mediaruntime

import (
	"embed"
	"net/http"

	"m31labs.dev/gosx-studio/internal/runtimeasset"
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
	return runtimeasset.Handler("media-runtime.js", Script(), "text/javascript; charset=utf-8")
}
