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

// TestRenderDirectEditPanelCarriesSelectionScopingMarker guards wave 3B fix
// #2 (dedupe "Selected content"): hostruntime/assets/workbench_runtime.js's
// scopeDirectEditPanels() keys off data-studio-direct-edit-panel="true" (the
// generic marker, since a host may give each panel's own <form id> caller-
// supplied and non-predictable) plus the already-existing
// data-studio-target-component attribute (the ComponentKey) to hide every
// direct-edit panel except the one(s) matching the live canvas selection.
// Both attributes must be present on every rendered instance regardless of
// which target it is, or the runtime guard has nothing to find.
func TestRenderDirectEditPanelCarriesSelectionScopingMarker(t *testing.T) {
	node := RenderDirectEditPanel(DirectEditPanelOptions{Target: authoring.OperationTarget{Route: "/contact", PageID: "page:contact", ComponentKey: "contact:contact-links", Field: "contact.email"}})
	s := gosx.RenderHTML(node)
	for _, want := range []string{`data-studio-direct-edit-panel="true"`, `data-studio-target-component="contact:contact-links"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in %s", want, s)
		}
	}
}
