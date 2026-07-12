package operationruntime

import (
	"strings"
	"testing"
)

func TestScriptContainsDurableOperationContract(t *testing.T) {
	s := string(Script())
	for _, want := range []string{"data-gosx-studio-durable-history", "credentials", "gosx_studio_expected_target_head", "gosx_studio_style_property", "gosx_studio_component_key", "data-studio-operation-value", "targetHeads", "kind === \"set-field\"", "preview-select", "editor-operation", "bridgeSelection", "updateCursors", "updateHistoryButtons", "data.value", "documentRevision", "targetHead", "value", "undo", "redo"} {
		if !strings.Contains(s, want) {
			t.Fatalf("script missing %q", want)
		}
	}
	for _, want := range []string{"data-gosx-studio-operation-value-source", "data-studio-layout-control", "data-studio-layout-value", "Inherited after reload"} {
		if !strings.Contains(s, want) {
			t.Fatalf("script missing responsive layout contract %q", want)
		}
	}
	for _, forbidden := range []string{
		`(kind === "set-field" || kind === "set-style"`,
		`(kind === "set-field" || kind === "reset-style"`,
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("style operations must not replace content value or content Undo history: found %q", forbidden)
		}
	}
	// The style-state button attribute (panels/responsive_layout_inspector.go)
	// must stay data-gosx-studio-style-state, never data-gosx-studio-state --
	// the latter is reserved for the form-state runtime's own <form> element
	// (hostruntime/assets/state_runtime.js's initAll() selector) and a
	// collision there throws "forEach called on null or undefined" on every
	// editor load once the layout inspector renders (real, pre-existing bug,
	// fixed alongside this test).
	if !strings.Contains(s, `getAttribute("data-gosx-studio-style-state")`) {
		t.Fatal(`script must read the non-colliding data-gosx-studio-style-state attribute`)
	}
	if strings.Contains(s, `getAttribute("data-gosx-studio-state")`) {
		t.Fatal(`script must not read data-gosx-studio-state -- that attribute is reserved for the form-state runtime`)
	}
}

func TestRuntimePrefersAuthenticatedCollaborationTransport(t *testing.T) {
	script := string(Script())
	for _, want := range []string{
		`function collaborationRequest(payload, target, extra)`,
		`typeof collaboration.available === "function"`,
		`collaboration.submit(request).then(collaborationResponse)`,
		`return submit(form, payload, target, extra).then(function (response)`,
		`request.expectedTargetHead = collaboration.expectedHead(request)`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("runtime missing collaboration bridge %q", want)
		}
	}
	for _, forbidden := range []string{`actorId:`, `capabilities:`} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("operation request contains client authority %q", forbidden)
		}
	}
}

// TestRuntimeDispatchesInstanceInteractionAndFlowDurableFamilies locks the
// client dispatch contract for the six instance/interaction/flow durable
// operation kinds (authoring/operations.go) to the exact literal Field
// values and JSON settings keys those Go types expect -- see
// authoring.FieldInstancesSharedField etc. and
// authoring.InteractionSettings/FlowFieldSettings/FlowActionSettings' json
// tags. A drift here would silently desync the browser-built
// OperationRequest from what OperationRequest.Validate accepts.
func TestRuntimeDispatchesInstanceInteractionAndFlowDurableFamilies(t *testing.T) {
	script := string(Script())
	for _, want := range []string{
		`"set-shared-field": "instances.shared-field"`,
		`"override-instance": "instances.override"`,
		`"detach-instance": "instances.attachment"`,
		`"restore-instance": "instances.attachment"`,
		`"set-interaction": "interactions.entry"`,
		`"remove-interaction": "interactions.entry"`,
		`"set-flow-field": "flows.field"`,
		`"remove-flow-field": "flows.field"`,
		`"set-flow-action": "flows.action"`,
		`function durableValue(kind, payload)`,
		`function durableControlKey(kind, payload)`,
		`gosx_studio_control_key: extra.controlKey || ""`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("runtime missing durable family dispatch %q", want)
		}
	}
}
