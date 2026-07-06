package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderPreviewSharePanelReady(t *testing.T) {
	view := map[string]any{
		"class":        "studio-preview-share",
		"kicker":       "Preview",
		"title":        "Share draft",
		"status":       "Expires in 7 days",
		"state":        "ready",
		"detail":       "Copy a signed preview for review without granting admin access.",
		"inputLabel":   "Signed preview URL",
		"copyLabel":    "Copy",
		"openLabel":    "Open preview",
		"href":         "https://example.test/?preview=token",
		"hasHref":      true,
		"resource":     "settings/site",
		"audience":     "reviewer",
		"expiresLabel": "Jun 28, 2026 10:00 UTC",
	}

	html := gosx.RenderHTML(RenderPreviewSharePanel(view, PreviewSharePanelOptions{}))

	for _, fragment := range []string{
		`<section class="studio-preview-share" data-studio-preview-share="true" data-studio-preview-share-state="ready" data-gosx-studio-preview-share-renderer="gosx-studio">`,
		`<div class="studio-preview-share__head"><div><p class="studio-preview-share__kicker">Preview</p><h2>Share draft</h2></div><output>Expires in 7 days</output></div>`,
		`<p class="studio-preview-share__detail">Copy a signed preview for review without granting admin access.</p>`,
		`<label class="studio-preview-share__field"><span>Signed preview URL</span><input type="url" readonly="readonly" value="https://example.test/?preview=token" data-studio-preview-url="true" aria-label="Signed preview URL" /></label>`,
		`<button type="button" data-studio-copy-target="[data-studio-preview-url]">Copy</button>`,
		`<a href="https://example.test/?preview=token" target="_blank" rel="noreferrer">Open preview</a>`,
		`<dl class="studio-preview-share__meta"><dt>Resource</dt><dd>settings/site</dd><dt>Audience</dt><dd>reviewer</dd><dt>Expires</dt><dd>Jun 28, 2026 10:00 UTC</dd></dl>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered preview share missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`studio-preview-share__empty`,
		"<form",
		"csrf_token",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered preview share must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderPreviewSharePanelDisabled(t *testing.T) {
	view := map[string]any{
		"class":       "studio-preview-share studio-preview-share--disabled",
		"kicker":      "Preview",
		"title":       "Share preview",
		"status":      "Unavailable",
		"state":       "disabled",
		"emptyTitle":  "Preview link unavailable",
		"emptyDetail": "Configure MUDDY_PREVIEW_SECRET or SESSION_SECRET before sharing drafts outside the editor.",
		"hasHref":     false,
	}

	html := gosx.RenderHTML(RenderPreviewSharePanel(view, PreviewSharePanelOptions{}))

	for _, fragment := range []string{
		`data-studio-preview-share="true"`,
		`data-studio-preview-share-state="disabled"`,
		`data-gosx-studio-preview-share-renderer="gosx-studio"`,
		`<div class="studio-preview-share__head"><div><p class="studio-preview-share__kicker">Preview</p><h2>Share preview</h2></div><output>Unavailable</output></div>`,
		`<article class="studio-preview-share__empty"><strong>Preview link unavailable</strong><p>Configure MUDDY_PREVIEW_SECRET or SESSION_SECRET before sharing drafts outside the editor.</p></article>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("disabled preview share missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`data-studio-preview-url="true"`,
		`data-studio-copy-target="[data-studio-preview-url]"`,
		`target="_blank"`,
		"<form",
		"csrf_token",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("disabled preview share must not include %q:\n%s", notWant, html)
		}
	}
}
