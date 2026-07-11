package flowruntime

import (
	"strings"
	"testing"
)

func TestRuntimeContracts(t *testing.T) {
	s := string(Bundle())
	for _, w := range []string{`typeof window==="undefined"`, `data-gosx-flow-preview-scope`, `data-authoritative-page-canvas`, `aria-invalid`, `data-flow-announcer`, `prefers-reduced-motion: reduce`, `checkValidity`, `safe(t.destination)`} {
		if !strings.Contains(s, w) {
			t.Fatalf("missing %q", w)
		}
	}
	for _, bad := range []string{"eval(", "new Function", "innerHTML", "new RegExp"} {
		if strings.Contains(s, bad) {
			t.Fatalf("unsafe %q", bad)
		}
	}
}
