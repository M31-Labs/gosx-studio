package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	studiocollab "m31labs.dev/gosx-studio/cms/studio/collab"
)

func TestRenderActivityPanelIncludesReadinessAndProposalShell(t *testing.T) {
	readiness := NewReadiness(
		NewReadinessItem("shell", "Shell", ReadinessReady, "Mounted", "Canvas is ready."),
		NewReadinessItem("collab", "Collaboration", ReadinessWatch, "Proposal review", "Wire rooms."),
	)
	html := gosx.RenderHTML(RenderActivityPanel(readiness, emptyProposalSnapshot(), ActivityOptions{}))
	for _, want := range []string{
		`data-studio-activity-drawer="true"`,
		`class="studio-readiness-score"`,
		`1/2 ready`,
		`data-readiness-key="shell"`,
		`data-studio-comments="true"`,
		`No comments`,
		`data-studio-proposals="true"`,
		`No proposals`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in activity html: %s", want, html)
		}
	}
}

func TestRenderActivityPanelCollapsedAndCustomClass(t *testing.T) {
	html := gosx.RenderHTML(RenderActivityPanel(Readiness{}, emptyProposalSnapshot(), ActivityOptions{
		Class:          "studio-drawer",
		ReadinessLabel: "Launch",
		Collapsed:      true,
	}))
	for _, want := range []string{
		`class="studio-drawer"`,
		`aria-pressed="false"`,
		`Show`,
		`Launch`,
		`0/0 ready`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in collapsed activity html: %s", want, html)
		}
	}
}

func TestRenderActivityPanelIncludesExtraPanels(t *testing.T) {
	html := gosx.RenderHTML(RenderActivityPanel(Readiness{}, emptyProposalSnapshot(), ActivityOptions{
		ExtraPanels: []gosx.Node{
			RenderPerformancePanel([]PerformanceSignal{
				NewPerformanceSignal("frame", "Frame batching", "rAF", "single frame", ReadinessReady, ""),
			}, PerformancePanelOptions{Class: "studio-performance"}),
		},
	}))
	for _, want := range []string{
		`data-studio-performance="true"`,
		`Frame batching`,
		`data-studio-comments="true"`,
		`data-studio-proposals="true"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in activity html: %s", want, html)
		}
	}
}

func emptyProposalSnapshot() studiocollab.Snapshot {
	return studiocollab.Snapshot{Resource: studiocollab.Resource{Kind: "page", ID: "home"}}
}
