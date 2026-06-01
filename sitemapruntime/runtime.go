package sitemapruntime

import _ "embed"

//go:embed island_runtime.js
var islandRuntimeJS []byte

// Bundle returns the Studio-owned site-map board interaction runtime.
// The visible board can stay host-rendered while this package owns the
// control semantics for filters, detail tabs, palette focus, and selection.
func Bundle() []byte {
	return islandRuntimeJS
}
