package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/flows"
	"strings"
	"testing"
)

func TestFlowGraphPanelAccessibleMap(t *testing.T) {
	g := flows.Graph{ID: "f", Name: "Flow", InitialStepID: "a", Steps: []flows.FlowStep{{ID: "a", Label: "Start", Kind: flows.StepForm}, {ID: "b", Label: "Done", Kind: flows.StepSuccess}}, Transitions: []flows.FlowTransition{{ID: "t", FromStepID: "a", ToStepID: "b", Kind: flows.TransitionSuccess}}}
	h := gosx.RenderHTML(RenderFlowGraphPanel(FlowGraphPanelOptions{Action: "/flows", CSRFToken: "csrf", OperationID: "op", ExpectedHead: "head", Flow: g}))
	for _, w := range []string{`aria-label="Ordered flow steps"`, `aria-label="Flow transitions"`, `a → b · success`, `data-gosx-flow-preview-scope="true"`, `authoritative page canvas`, `name="csrf_token" value="csrf"`} {
		if !strings.Contains(h, w) {
			t.Fatalf("missing %q", w)
		}
	}
}
