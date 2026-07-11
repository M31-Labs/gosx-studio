package panels

import (
	"fmt"
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/flows"
	"strings"
)

type FlowGraphPanelOptions struct {
	ID, Action, CSRFToken, OperationID, ExpectedHead string
	Flow                                             flows.Graph
}

func RenderFlowGraphPanel(o FlowGraphPanelOptions) gosx.Node {
	if strings.TrimSpace(o.ID) == "" {
		o.ID = "studio-flow-graph"
	}
	steps := []gosx.Node{}
	for i, s := range o.Flow.Steps {
		steps = append(steps, gosx.El("li", gosx.Attrs(gosx.Attr("data-studio-flow-step", s.ID), gosx.Attr("data-studio-flow-step-kind", string(s.Kind))), gosx.El("strong", nil, gosx.Text(fmt.Sprintf("%d. %s", i+1, s.Label))), gosx.El("span", nil, gosx.Text(string(s.Kind)))))
	}
	transitions := []gosx.Node{}
	for _, t := range o.Flow.Transitions {
		transitions = append(transitions, gosx.El("li", gosx.Attrs(gosx.Attr("data-studio-flow-transition", t.ID)), gosx.Text(t.FromStepID+" → "+t.ToStepID+" · "+string(t.Kind))))
	}
	children := []gosx.Node{hidden("gosx_studio_flow_operation", "upsert-flow"), hidden("gosx_studio_flow_operation_id", o.OperationID), hidden("gosx_studio_flow_expected_head", o.ExpectedHead), gosx.El("label", nil, gosx.Text("Flow name"), gosx.El("input", gosx.Attrs(gosx.Attr("name", "flow_name"), gosx.Attr("required", "required"), gosx.Attr("value", o.Flow.Name)))), gosx.El("label", nil, gosx.Text("Initial step"), gosx.El("input", gosx.Attrs(gosx.Attr("name", "flow_initial_step"), gosx.Attr("pattern", "[A-Za-z0-9_.-]+"), gosx.Attr("value", o.Flow.InitialStepID)))), gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Save flow"))}
	if o.CSRFToken != "" {
		children = append(children, hidden("csrf_token", o.CSRFToken))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("id", o.ID), gosx.Attr("class", "editor-panel studio-flow-graph"), gosx.Attr("data-studio-flow-graph-panel", "true")), gosx.El("header", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Flows")), gosx.El("h2", nil, gosx.Text("Step and transition map")), gosx.El("output", gosx.Attrs(gosx.Attr("aria-live", "polite")), gosx.Text(fmt.Sprintf("%d steps · %d transitions", len(o.Flow.Steps), len(o.Flow.Transitions))))), gosx.El("p", nil, gosx.Text("Success and error branches are explicit. Preview runs in its own scope and cannot modify the authoritative page canvas.")), gosx.El("ol", gosx.Attrs(gosx.Attr("aria-label", "Ordered flow steps")), gosx.Fragment(steps...)), gosx.El("ul", gosx.Attrs(gosx.Attr("aria-label", "Flow transitions")), gosx.Fragment(transitions...)), gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", o.Action), gosx.Attr("data-studio-flow-form", "true")), gosx.Fragment(children...)), gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-flow-preview-scope", "true"), gosx.Attr("aria-label", "Flow preview"), gosx.Attr("role", "region"))))
}
