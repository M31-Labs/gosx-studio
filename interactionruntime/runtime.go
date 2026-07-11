// Package interactionruntime provides the host-agnostic public behavior
// runtime for validated interaction graphs. It is inert during SSR.
package interactionruntime

import _ "embed"

//go:embed runtime.js
var runtimeJS []byte

func Bundle() []byte                   { return append([]byte(nil), runtimeJS...) }
func InteractionRuntimeScript() []byte { return Bundle() }
