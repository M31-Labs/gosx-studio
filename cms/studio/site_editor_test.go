package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderStudioToolbarComposesSharedChrome(t *testing.T) {
	html := gosx.RenderHTML(RenderStudioToolbar(StudioToolbarOptions{
		Class:        "studio-toolbar",
		ActionsClass: "studio-toolbar__actions",
		Kicker:       "Homepage",
		Title:        "Canvas",
		Summary:      "4 blocks",
		Controls: []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("data-studio-mode-control", "structure")), gosx.Text("Structure")),
		},
		Actions: []gosx.Node{
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Save")),
		},
	}))
	for _, want := range []string{
		`data-gosx-studio-toolbar="true"`,
		`data-studio-toolbar="true"`,
		`class="studio-toolbar__actions"`,
		`data-gosx-studio-toolbar-actions="true"`,
		`Homepage`,
		`Canvas`,
		`data-studio-mode-control="structure"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in toolbar markup: %s", want, html)
		}
	}
}

func TestRenderPreviewFrameExposesPreviewHooks(t *testing.T) {
	html := gosx.RenderHTML(RenderPreviewFrame(PreviewFrameOptions{
		ShellClass:   "editor-preview-shell",
		ToolbarClass: "storefront-frame-toolbar",
		MetaClass:    "studio-preview-toolbar__flow",
		FrameClass:   "preview-frame",
		StatusClass:  "studio-preview-toolbar__status",
		OpenClass:    "button button--secondary",
		Kicker:       "Previewing",
		Title:        "Public site",
		URL:          "/contact?flow=schedule-tour",
		IFrameTitle:  "Public site preview",
		StatusLabel:  "Ready",
		OpenLabel:    "Open route",
		OpenNewTab:   true,
		DynamicTitle: true,
		DynamicRoute: true,
	}))
	for _, want := range []string{
		`data-gosx-studio-preview="true"`,
		`data-gosx-studio-preview-url="/contact?flow=schedule-tour"`,
		`data-gosx-studio-preview-state="ready"`,
		`data-studio-preview-toolbar="true"`,
		`data-studio-preview-frame="true"`,
		`data-studio-preview-src="/contact?flow=schedule-tour"`,
		`data-studio-preview-status="true"`,
		`aria-live="polite"`,
		`data-studio-selected-flow-label="true"`,
		`data-studio-selected-flow-route="true"`,
		`data-studio-open-preview="true"`,
		`target="_blank"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in preview markup: %s", want, html)
		}
	}
}
