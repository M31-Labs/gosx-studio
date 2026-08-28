package panels

import (
	"m31labs.dev/gosx"
	"strings"
	"testing"
)

func TestCollaborationPanelRendersAccessibleLiveSurface(t *testing.T) {
	html := gosx.RenderHTML(RenderCollaborationPanel(CollaborationPanelOptions{Endpoint: "/admin/editor/collab", Resource: "site:main"}))
	for _, want := range []string{
		`data-gosx-studio-collaboration="true"`,
		`data-studio-collab-endpoint="/admin/editor/collab"`,
		`data-studio-collab-facepile`,
		`aria-live="polite"`,
		`role="alert"`,
		`data-studio-collab-reconnect`,
		// handoff-4 (punch #7 -- presence to top chrome): a stable id and
		// detail marker so the toolbar's collaboration summary
		// (shell.RenderWorkbenchCollaborationSummary) can point
		// aria-controls at this section and reveal it as the detail view.
		`id="studio-collaboration-panel"`,
		`data-studio-collab-detail="true"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("panel missing %q: %s", want, html)
		}
	}
}

// TestCollaborationPanelCollapsesByDefaultAndExpandsOnRequest guards
// handoff-4 (punch #7 -- presence to top chrome): the full collaboration
// panel is the "detail view" the toolbar's compact summary reveals on
// click, so it renders [hidden] unless the host explicitly opts into
// always-expanded via Expanded: true.
func TestCollaborationPanelCollapsesByDefaultAndExpandsOnRequest(t *testing.T) {
	sectionTag := func(html string) string {
		end := strings.Index(html, ">")
		if end < 0 {
			return html
		}
		return html[:end+1]
	}
	collapsed := gosx.RenderHTML(RenderCollaborationPanel(CollaborationPanelOptions{Endpoint: "/admin/editor/collab", Resource: "site:main"}))
	if !strings.Contains(sectionTag(collapsed), "hidden") {
		t.Fatalf("expected the collaboration panel's own <section> tag to be hidden by default: %s", sectionTag(collapsed))
	}
	expanded := gosx.RenderHTML(RenderCollaborationPanel(CollaborationPanelOptions{Endpoint: "/admin/editor/collab", Resource: "site:main", Expanded: true}))
	if strings.Contains(sectionTag(expanded), "hidden") {
		t.Fatalf("expected Expanded: true to omit hidden from the <section> tag: %s", sectionTag(expanded))
	}
}
