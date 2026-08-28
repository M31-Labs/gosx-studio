package sitemap

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

func TestRenderSiteNavigatorPanelUsesTypedProps(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteNavigatorPanel(SiteNavigatorProps{
		Mode:     "site",
		Kicker:   "Website",
		Title:    "Site areas",
		Label:    "Site areas",
		Empty:    "No pages.",
		HasItems: true,
		Items: []SiteNavigatorItem{{
			Key:     "products",
			Label:   "Products",
			Href:    "/admin/products",
			Group:   "store",
			Class:   "is-active",
			Summary: "Manage product catalog",
		}},
	}, SiteNavigatorPanelOptions{}))

	for _, fragment := range []string{
		`<section class="studio-nav-panel" data-studio-site-navigator-panel="true" data-studio-mode-panel="site" data-studio-engine-source="gosx" data-studio-site-filter="all" data-gosx-studio-site-navigator-renderer="gosx-studio" data-studio-rail-group="site-navigator">`,
		`<div class="studio-page-filter" role="toolbar" aria-label="Site area filter">`,
		`<a href="/admin/products" class="is-active" data-gosx-link="true" data-studio-site-page="products" data-studio-site-group="store" title="Manage product catalog"><span>Products</span></a>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("site navigator panel render missing %q:\n%s", fragment, html)
		}
	}
	if !strings.Contains(html, `<button type="button" aria-pressed>All</button>`) &&
		!strings.Contains(html, `<button type="button" aria-pressed="true">All</button>`) {
		t.Fatalf("site navigator panel render missing pressed All filter:\n%s", html)
	}
}

func TestRenderSiteNavigatorNewPage(t *testing.T) {
	view := map[string]any{
		"newPageLabel": "New page",
		"newPageItems": []map[string]any{{
			"key":     "landing",
			"label":   "Landing page",
			"formID":  "studioSiteMapIntentForm-landing",
			"summary": "Hero, story, and a direct next step.",
		}},
	}

	html := gosx.RenderHTML(RenderSiteNavigatorNewPage(view, SiteNavigatorPanelOptions{}))

	for _, fragment := range []string{
		`<details class="studio-new-page" data-studio-new-page="true" data-gosx-studio-site-navigator-renderer="gosx-studio">`,
		`<summary>+ New page</summary>`,
		`<div class="studio-new-page__options">`,
		`<button type="submit" form="studioSiteMapIntentForm-landing" class="button button--secondary studio-new-page-option" data-editor-add-block="page-landing" data-studio-new-page-blueprint="landing" title="Hero, story, and a direct next step.">New page: Landing page</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered new-page disclosure missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderSiteNavigatorNewPageEmpty(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteNavigatorNewPage(map[string]any{}, SiteNavigatorPanelOptions{}))
	if strings.Contains(html, "studio-new-page") {
		t.Fatalf("new-page disclosure should not render with no blueprint choices:\n%s", html)
	}
}

func TestRenderSiteNavigatorPanelIncludesNewPageWhenProvided(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteNavigatorPanel(SiteNavigatorProps{
		Mode:         "home",
		Kicker:       "Website",
		Title:        "Site areas",
		Label:        "Site areas",
		Empty:        "No pages.",
		HasItems:     true,
		NewPageLabel: "New page",
		NewPageItems: []SiteNavigatorNewPageItem{{
			Key:    "landing",
			Label:  "Landing page",
			FormID: "studioSiteMapIntentForm-landing",
		}},
		Items: []SiteNavigatorItem{{Key: "home", Label: "Home page", Href: "/"}},
	}, SiteNavigatorPanelOptions{}))

	for _, fragment := range []string{
		`data-studio-new-page="true"`,
		`New page: Landing page`,
		`data-editor-add-block="page-landing"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("site navigator panel missing new-page disclosure %q:\n%s", fragment, html)
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
