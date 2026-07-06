package studio

import (
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

func TestRenderPreviewSharePanelReady(t *testing.T) {
	now := time.Date(2026, 5, 17, 20, 0, 0, 0, time.UTC)
	link := lifecycle.PreviewLink{
		ResourceKind: "settings",
		ResourceID:   "site",
		Route:        "/",
		Audience:     "client",
		Created:      now,
		Expires:      now.Add(72 * time.Hour),
	}
	html := gosx.RenderHTML(RenderPreviewSharePanel(link, "https://example.test/?preview_token=abc", PreviewShareOptions{Now: now}))
	for _, want := range []string{
		`data-studio-preview-share="true"`,
		`data-studio-preview-share-state="ready"`,
		`Share preview`,
		`data-studio-preview-url="true"`,
		`data-studio-copy-target="[data-studio-preview-url]"`,
		`settings/site`,
		`client`,
		`3d left`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in preview share html: %s", want, html)
		}
	}
}

func TestRenderPreviewSharePanelDisabled(t *testing.T) {
	html := gosx.RenderHTML(RenderPreviewSharePanel(lifecycle.PreviewLink{}, "", PreviewShareOptions{
		Class:      "studio-preview-share",
		EmptyTitle: "Missing secret",
	}))
	for _, want := range []string{
		`class="studio-preview-share studio-preview-share--disabled"`,
		`data-studio-preview-share-state="disabled"`,
		`Missing secret`,
		`Configure a preview signing secret`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in disabled preview share html: %s", want, html)
		}
	}
}
