package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderSaveStatusExposesStateTelemetryHooks(t *testing.T) {
	html := gosx.RenderHTML(RenderSaveStatus(SaveStatusOptions{
		Class:           "studio-save-status",
		StateClass:      "studio-save-state",
		DetailClass:     "studio-save-detail",
		LastSavedClass:  "studio-save-time",
		DirtyCountClass: "studio-save-count",
	}))

	for _, want := range []string{
		`data-gosx-studio-save-status="true"`,
		`data-gosx-studio-save-state="true"`,
		`data-gosx-studio-save-detail="true"`,
		`data-gosx-studio-dirty-count="true"`,
		`data-gosx-studio-last-saved="true"`,
		`class="studio-save-state"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in save status markup: %s", want, html)
		}
	}
}
