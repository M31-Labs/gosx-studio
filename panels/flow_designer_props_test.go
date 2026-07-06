package panels

import (
	"os"
	"strings"
	"testing"
)

func TestFlowDesignerPropsSourceContract(t *testing.T) {
	source, err := os.ReadFile("flow_designer.go")
	if err != nil {
		t.Fatalf("read flow designer prop source: %v", err)
	}
	text := string(source)

	for _, want := range []string{
		"type FlowDesignerProps struct",
		"FormID         string",
		"PublishAction  string",
		"DefaultFlowKey string",
		"Flows          []FlowDesignerFlow",
		"type FlowDesignerFlow struct",
		"HandlerInputName   string",
		"HandlerSource      string",
		"Readiness          FlowDesignerReadiness",
		"ReadinessChecks    []FlowDesignerCheck",
		"Nodes              []FlowDesignerNode",
		"Steps              []FlowDesignerStep",
		"Actions            []FlowDesignerAction",
		"type FlowDesignerReadiness struct",
		"CanPublish  bool",
		"BadgeClass  string",
		"type FlowDesignerCheck struct",
		"TabIndex    string",
		"type FlowDesignerNode struct",
		"KindLabel   string",
		"type FlowDesignerStep struct",
		"LabelInputName string",
		"BodySource     string",
		"type FlowDesignerAction struct",
		"FieldCount int",
		"Fields     []FlowDesignerField",
		"type FlowDesignerField struct",
		"RequiredLabel string",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("flow designer prop source missing %q", want)
		}
	}

	for _, forbidden := range []string{
		" any",
		"gosx.Node",
		"map[string]any",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("flow designer props must stay serializable typed data, found %q", forbidden)
		}
	}
}
