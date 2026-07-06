package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderHistoryControls(t *testing.T) {
	html := gosx.RenderHTML(RenderHistoryControls(HistoryControlsOptions{IncludeStatus: true}))
	for _, want := range []string{
		`data-gosx-studio-history-controls="true"`,
		`data-gosx-studio-history-undo="true"`,
		`data-gosx-studio-history-redo="true"`,
		`data-gosx-studio-history-status="true"`,
		`disabled="disabled"`,
		`Undo`,
		`Redo`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in history controls: %s", want, html)
		}
	}
}
