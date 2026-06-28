package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderSiteNavigatorHeaderAndList(t *testing.T) {
	view := map[string]any{
		"kicker":   "Website",
		"title":    "Site areas",
		"label":    "Site areas",
		"hasItems": true,
		"items": []map[string]any{{
			"key":     "products",
			"label":   "Products",
			"href":    "/admin/products",
			"summary": "Manage product catalog",
			"group":   "store",
			"class":   "is-active",
		}},
	}

	html := gosx.RenderHTML(gosx.Fragment(
		RenderSiteNavigatorHeader(view, SiteNavigatorPanelOptions{}),
		RenderSiteNavigatorList(view, SiteNavigatorPanelOptions{}),
	))

	for _, fragment := range []string{
		`<div class="studio-panel-heading" data-gosx-studio-site-navigator-renderer="gosx-studio"><p class="kicker">Website</p><h2>Site areas</h2></div>`,
		`<nav class="studio-page-list" aria-label="Site areas" data-gosx-studio-site-navigator-renderer="gosx-studio">`,
		`<a href="/admin/products" class="is-active" data-gosx-link="true" data-studio-site-page="products" data-studio-site-group="store" title="Manage product catalog"><span>Products</span></a>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("site navigator render missing %q:\n%s", fragment, html)
		}
	}
	for _, forbidden := range []string{
		"GoSX Studio",
		"studio-page-list__summary",
		"studio-page-list__group",
		`hidden=`,
	} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("site navigator render should not include %q:\n%s", forbidden, html)
		}
	}
}

func TestRenderSiteNavigatorListEmpty(t *testing.T) {
	view := map[string]any{
		"label": "Site areas",
		"empty": "No site areas are available.",
	}

	html := gosx.RenderHTML(RenderSiteNavigatorList(view, SiteNavigatorPanelOptions{}))

	for _, fragment := range []string{
		`data-gosx-studio-site-navigator-renderer="gosx-studio"`,
		`<p class="empty">No site areas are available.</p>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty site navigator render missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<a `) {
		t.Fatalf("empty site navigator should not render anchors:\n%s", html)
	}
}
