package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendStorefrontPreviewRendersShell(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendStorefrontPreview(BackendStorefrontPreviewProps{
		Kicker:       "Storefront",
		Title:        "Live preview",
		RefreshHref:  "/admin/storefront?path=%2Fshop",
		PublicHref:   "/shop",
		PathInput:    "/shop",
		RouteSummary: "Shop",
		PreviewPath:  "/shop",
		Routes: []BackendStorefrontPreviewRoute{
			{
				Label:    "Home",
				Href:     "/admin/storefront?path=%2F",
				Class:    "storefront-tab",
				GOSXLink: true,
			},
			{
				Label:    "Shop",
				Href:     "/admin/storefront?path=%2Fshop",
				Class:    "storefront-tab storefront-tab--active",
				GOSXLink: true,
			},
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-storefront-preview-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">Storefront</p><h1>Live preview</h1></section>`,
		`<section class="panel storefront-preview"><div class="storefront-preview__toolbar"><nav class="storefront-tabs" aria-label="Storefront preview routes">`,
		`<a class="storefront-tab" href="/admin/storefront?path=%2F" data-gosx-link="true">Home</a>`,
		`<a class="storefront-tab storefront-tab--active" href="/admin/storefront?path=%2Fshop" data-gosx-link="true">Shop</a>`,
		`<div class="button-row"><a class="button button--secondary" href="/admin/storefront?path=%2Fshop" data-gosx-link="true">Refresh</a><a class="button button--primary" href="/shop" target="_blank" rel="noreferrer">Open public</a></div>`,
		`<form class="storefront-preview__form" method="get" action="/admin/storefront"><label for="storefront-path">Path</label><input id="storefront-path" name="path" value="/shop" class="storefront-preview__path" /><button class="button button--secondary" type="submit">Load</button></form>`,
		`<div class="storefront-frame-toolbar"><span>Shop</span><code>/shop</code></div><iframe class="storefront-frame" title="Storefront preview" src="/shop"></iframe>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("storefront preview html missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendStorefrontPreviewContentAndSummary(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendStorefrontPreviewContent(BackendStorefrontPreviewProps{
		Kicker:       "Storefront",
		Title:        "Live preview",
		Summary:      "Preview published routes.",
		RefreshHref:  "/admin/storefront?path=%2F",
		PublicHref:   "/",
		PathInput:    "/",
		RouteSummary: "Home",
		PreviewPath:  "/",
	}))

	for _, fragment := range []string{
		`<section class="admin-heading"><p class="kicker">Storefront</p><h1>Live preview</h1><p>Preview published routes.</p></section>`,
		`<nav class="storefront-tabs" aria-label="Storefront preview routes"></nav>`,
		`<span>Home</span><code>/</code>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("storefront preview content missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `data-gosx-studio-backend-storefront-preview-renderer="gosx-studio"`) {
		t.Fatalf("content renderer should not render its own page wrapper:\n%s", html)
	}
}

func TestRenderBackendStorefrontPreviewSplitNodes(t *testing.T) {
	heading := gosx.RenderHTML(RenderBackendStorefrontPreviewHeading(BackendStorefrontPreviewProps{
		Kicker: "Storefront",
		Title:  "Live preview",
	}))
	panel := gosx.RenderHTML(RenderBackendStorefrontPreviewPanel(BackendStorefrontPreviewProps{}))

	if !strings.Contains(heading, `<section class="admin-heading">`) || strings.Contains(heading, `<section class="panel storefront-preview">`) {
		t.Fatalf("heading node should render only the admin heading:\n%s", heading)
	}
	if !strings.Contains(panel, `<section class="panel storefront-preview">`) || strings.Contains(panel, `<section class="admin-heading">`) {
		t.Fatalf("panel node should render only the storefront preview panel:\n%s", panel)
	}
}
