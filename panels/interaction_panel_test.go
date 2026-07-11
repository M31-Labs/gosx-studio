package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
	"strings"
	"testing"
)

func TestInteractionPanelAccessibleFiniteEditor(t *testing.T) {
	g := core.InteractionGraph{ID: "graph_a", Event: core.InteractionHover, Actions: []core.InteractionAction{{ID: "a", Kind: core.InteractionShow}}}
	html := gosx.RenderHTML(RenderInteractionPanel(InteractionPanelOptions{Action: "/interactions", CSRFToken: "csrf", SelectedGraphID: g.ID, Graphs: []core.InteractionGraph{g}, OperationIDs: map[string]string{"upsert": "op"}, Heads: map[string]string{"interaction:graph_a": "head"}}))
	for _, want := range []string{`data-studio-interaction-panel="true"`, `aria-label="Configured interactions"`, `Stable interaction ID`, `viewport-enter`, `toggle-class`, `play-media`, `Only without reduced motion`, `Hover behaviors also run from keyboard focus`, `name="csrf_token" value="csrf"`} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "javascript:") || strings.Contains(html, "textarea") {
		t.Fatal("panel exposes arbitrary code input")
	}
}
