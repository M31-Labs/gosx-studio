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
	for _, forbidden := range []string{
		`(kind === "set-field" || kind === "set-style"`,
		`(kind === "set-field" || kind === "reset-style"`,
	} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("style operations must not replace content value or content Undo history: found %q", forbidden)
		}
	}
}
