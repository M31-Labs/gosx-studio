package operationruntime

import (
	"strings"
	"testing"
)

func TestScriptContainsDurableOperationContract(t *testing.T) {
	s := string(Script())
	for _, want := range []string{"data-gosx-studio-durable-history", "credentials", "gosx_studio_expected_target_head", "gosx_studio_style_property", "gosx_studio_component_key", "data-studio-operation-value", "targetHeads", "kind === \"set-field\"", "updateCursors", "documentRevision", "targetHead", "value", "undo", "redo"} {
		if !strings.Contains(s, want) {
			t.Fatalf("script missing %q", want)
		}
	}
}
