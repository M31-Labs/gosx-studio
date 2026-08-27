package hostruntime

import (
	"net/http"

	"m31labs.dev/gosx-studio/sectionorderruntime"
)

// SectionOrderRuntimePath is the canonical Studio-owned browser asset for
// PageCanvas section ordering.
const SectionOrderRuntimePath = RuntimeRoot + "/section-order-runtime.js"

// SectionOrderRuntimeScript returns the PageCanvas section-order runtime.
func SectionOrderRuntimeScript() []byte { return sectionorderruntime.Script() }

// SectionOrderRuntimeHandler serves the PageCanvas section-order runtime.
func SectionOrderRuntimeHandler() http.Handler {
	return ScriptHandler("section-order-runtime.js", SectionOrderRuntimeScript())
}
