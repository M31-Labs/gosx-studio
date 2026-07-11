package panels

import (
	"m31labs.dev/gosx"
	"strings"
	"testing"
)

func TestCollaborationPanelRendersAccessibleLiveSurface(t *testing.T) {
	html := gosx.RenderHTML(RenderCollaborationPanel(CollaborationPanelOptions{Endpoint: "/admin/editor/collab", Resource: "site:main"}))
	for _, want := range []string{`data-gosx-studio-collaboration="true"`, `data-studio-collab-endpoint="/admin/editor/collab"`, `data-studio-collab-facepile`, `aria-live="polite"`, `role="alert"`, `data-studio-collab-reconnect`} {
		if !strings.Contains(html, want) {
			t.Fatalf("panel missing %q: %s", want, html)
		}
	}
}
