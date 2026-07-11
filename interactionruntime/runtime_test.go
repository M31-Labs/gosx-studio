package interactionruntime

import (
	"strings"
	"testing"
)

func TestRuntimeSecurityAccessibilityAndSSRContracts(t *testing.T) {
	js := string(Bundle())
	for _, want := range []string{`typeof window==="undefined"`, `prefers-reduced-motion: reduce`, `keydown`, `e.key==="Enter"||e.key===" "`, `mouseenter`, `focusin`, `IntersectionObserver`, `getElementById`, `Promise.resolve`, `safeNav`} {
		if !strings.Contains(js, want) {
			t.Fatalf("runtime missing %q", want)
		}
	}
	for _, bad := range []string{"eval(", "new Function", "innerHTML", "document.querySelector(a.target"} {
		if strings.Contains(js, bad) {
			t.Fatalf("runtime contains %q", bad)
		}
	}
}
