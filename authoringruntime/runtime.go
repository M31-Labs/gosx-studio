// Package authoringruntime owns the browser-side feedback loop for successful
// Studio authoring mutations.
package authoringruntime

import (
	_ "embed"
)

//go:embed island_runtime.js
var islandRuntimeJS []byte

// IslandRuntimeJS returns the embedded authoring-result runtime.
func IslandRuntimeJS() []byte {
	return islandRuntimeJS
}

// Bundle returns the complete authoring-result runtime served by the Studio
// engine asset.
func Bundle() []byte {
	return IslandRuntimeJS()
}
