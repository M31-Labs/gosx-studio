package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendSearchRendersResults(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendSearch(BackendSearchProps{
		Kicker:   "CMS",
		Title:    "Search",
		Summary:  "Find products, media, content, orders, and messages.",
		Query:    "cup <blue>",
		Searched: true,
		Results: []BackendSearchResult{
			{
				Href:     "/admin/products/product-1",
				Kind:     "Product",
				Title:    "Blue <Cup>",
				Detail:   "blue-cup",
				GOSXLink: true,
			},
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-search-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Search</h1><p>Find products, media, content, orders, and messages.</p></section>`,
		`<section class="panel"><div class="panel__header"><h2>Results</h2><span class="status">cup &lt;blue&gt;</span></div><div class="resource-grid">`,
		`<a class="resource-card" href="/admin/products/product-1" data-gosx-link="true"><span class="status">Product</span><strong>Blue &lt;Cup&gt;</strong><span>blue-cup</span></a>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("search html missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<p class="empty">No matching records.</p>`) {
		t.Fatalf("populated search rendered empty state:\n%s", html)
	}
}

func TestRenderBackendSearchContentAndEmptyState(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendSearchContent(BackendSearchProps{
		Kicker:   "CMS",
		Title:    "Search",
		Summary:  "Summary",
		Query:    "missing",
		Searched: true,
		Empty:    "Nothing found.",
	}))

	for _, fragment := range []string{
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Search</h1><p>Summary</p></section>`,
		`<section class="panel"><div class="panel__header"><h2>Results</h2><span class="status">missing</span></div><p class="empty">Nothing found.</p></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("search content missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `data-gosx-studio-backend-search-renderer="gosx-studio"`) {
		t.Fatalf("content renderer should not render its own page wrapper:\n%s", html)
	}
}

func TestRenderBackendSearchSplitNodes(t *testing.T) {
	heading := gosx.RenderHTML(RenderBackendSearchHeading(BackendSearchProps{
		Kicker:  "CMS",
		Title:   "Search",
		Summary: "Summary",
	}))
	unsearched := gosx.RenderHTML(RenderBackendSearchResults(BackendSearchProps{}))
	results := gosx.RenderHTML(RenderBackendSearchResults(BackendSearchProps{Searched: true}))

	if !strings.Contains(heading, `<section class="admin-heading">`) || strings.Contains(heading, `<section class="panel">`) {
		t.Fatalf("heading node should render only the admin heading:\n%s", heading)
	}
	if strings.Contains(unsearched, `<section class="panel">`) {
		t.Fatalf("unsearched results node should not render a panel:\n%s", unsearched)
	}
	if !strings.Contains(results, `<p class="empty">No matching records.</p>`) {
		t.Fatalf("results node missing default empty state:\n%s", results)
	}
}
