package shell

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderWorkbenchPageCanvasOwnsCanonicalFrameRouteViewportAndDiagnostics(t *testing.T) {
	html := gosx.RenderHTML(RenderWorkbenchPageCanvas(WorkbenchPageCanvasOptions{
		View: map[string]any{
			"selectionLabel": "Hero",
			"viewportKey":    "desktop",
			"viewports": []map[string]any{
				{"key": "desktop", "label": "Desktop", "width": "100%", "pressed": true},
				{"key": "mobile", "label": "Mobile", "width": "24rem"},
			},
		},
		URL:         "/?gosx-preview=1&amp;unsafe=escaped",
		Title:       "Noni <Home>",
		IFrameTitle: `Noni "Home" preview`,
		Routes: []WorkbenchPageCanvasRoute{
			{Key: "home", Label: "Home", URL: "/?gosx-preview=1", Active: true},
			{Key: "about", Label: "About", URL: "/about?gosx-preview=1"},
		},
	}))
	for _, fragment := range []string{
		`data-gosx-studio-page-canvas="true"`,
		`data-studio-preview-route="/?gosx-preview=1"`,
		`data-studio-preview-route="/about?gosx-preview=1"`,
		`data-studio-viewport="mobile"`,
		`data-studio-preview-diagnostic="true"`,
		`aria-live="polite"`,
		`data-studio-preview-frame="true"`,
		`title="Noni &#34;Home&#34; preview"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("page canvas missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "<form") {
		t.Fatalf("page canvas must never nest an authoring form:\n%s", html)
	}
}
