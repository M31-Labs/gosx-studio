package studio

import (
	"os"
	"strings"
	"testing"
)

func TestFlowDesignerIslandSourceContract(t *testing.T) {
	source, err := os.ReadFile("flow_designer_island.gsx")
	if err != nil {
		t.Fatalf("read flow designer island source: %v", err)
	}
	text := string(source)

	for _, want := range []string{
		"//gosx:island",
		"func FlowDesignerIsland(props FlowDesignerProps) gosx.Node",
		"selected := signal.New(props.DefaultFlowKey)",
		`dirtyFlow := signal.New("")`,
		"markDirty := func() { dirtyFlow.Set(selected.Get()) }",
		"flow.Readiness.CanPublish",
		"<Each of={flow.Steps} as=\"step\">",
		"<Each of={flow.Actions} as=\"action\">",
		"form={props.FormID}",
		"formaction={props.PublishAction}",
		"data-gosx-studio-flow-designer-panel-renderer=\"gosx-studio\"",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("flow designer island source missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"props any",
		"map[string]any",
		"props gosx.Node",
		"props []gosx.Node",
		"data.flowDesigner",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("flow designer island source must use typed serializable props, found %q", forbidden)
		}
	}
}
