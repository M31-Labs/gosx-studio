// Package sectionorderruntime embeds the shared PageCanvas section ordering
// browser runtime.
package sectionorderruntime

import (
	"embed"
	"net/http"

	"m31labs.dev/gosx-studio/internal/runtimeasset"
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
	return runtimeasset.Handler("section-order-runtime.js", Script(), "text/javascript; charset=utf-8")
}
