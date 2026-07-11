package flowruntime

import _ "embed"

//go:embed runtime.js
var js []byte

func Bundle() []byte { return append([]byte(nil), js...) }
