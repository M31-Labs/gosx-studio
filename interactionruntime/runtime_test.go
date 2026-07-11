package interactionruntime

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScriptContainsRevealOnScrollContract(t *testing.T) {
	s := string(Script())
	for _, want := range []string{
		"data-gosx-interaction",
		"reveal-on-scroll",
		"data-gosx-interaction-revealed",
		"data-gosx-interaction-once",
		"IntersectionObserver",
		"prefers-reduced-motion: reduce",
		"GoSXStudioInteractionRuntime",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("script missing %q", want)
		}
	}
}

func TestScriptFailsOpenToVisibleUnderReducedMotionAndMissingObserver(t *testing.T) {
	s := string(Script())
	// The reduced-motion/no-observer branch must call reveal(...) directly
	// rather than merely skip binding, so content is never left hidden.
	if !strings.Contains(s, "forEach.call(elements, reveal)") {
		t.Fatal("expected the reduced-motion/no-observer path to reveal every matched element immediately")
	}
}

func TestScriptNeverUsesEvalOrArbitrarySelectors(t *testing.T) {
	s := string(Script())
	for _, forbidden := range []string{"eval(", "new Function(", "innerHTML", "querySelector(root"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("script must not use %q", forbidden)
		}
	}
}

func TestScriptNeverTogglesVisibilityOrDisplay(t *testing.T) {
	s := string(Script())
	// Only opacity/transform (driven by CSS, not this script) may animate;
	// this script must only ever set the revealed data-attribute, never
	// touch style.display/style.visibility, which would remove a focusable
	// descendant from the tab order.
	for _, forbidden := range []string{"style.display", "style.visibility", ".hidden = true"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("script must not use %q (would hide focusable content from keyboard users)", forbidden)
		}
	}
}

func TestHandlerServesScriptAsJavaScript(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/interaction-runtime.js", nil)
	rec := httptest.NewRecorder()
	Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("expected a javascript content type, got %q", ct)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected a non-empty response body")
	}
}
