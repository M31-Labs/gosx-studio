package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"strings"
	"testing"
)

func TestRenderDirectEditPanelDurableIdentityAndCancel(t *testing.T) {
	node := RenderDirectEditPanel(DirectEditPanelOptions{Target: authoring.OperationTarget{Route: "/about", PageID: "page:about", Field: "pages.about.title"}, DurableHistory: true, DocumentRevision: 4, TargetHead: "head"})
	s := gosx.RenderHTML(node)
	for _, want := range []string{"data-gosx-studio-durable-history=\"true\"", "gosx_studio_expected_target_head", "data-gosx-studio-operation-cancel", "pages.about.title"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %s", want, s)
		}
	}
}
