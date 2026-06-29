package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendMediaLibraryRendersAssetGrid(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendMediaLibrary(BackendMediaLibraryProps{
		Kicker:  "CMS",
		Title:   "Media library",
		Summary: "Uploaded images, icons, and font files for storefront content and brand controls.",
		Empty:   "No media assets yet.",
		Assets: []BackendMediaAsset{
			{
				Href:        "/admin/media/image-1",
				URL:         "/media/cup.jpg",
				Alt:         "Cup <one>",
				Filename:    "cup.jpg",
				ContentType: "image/jpeg",
				Size:        "14 KB",
				StatusLabel: "Published",
				IsImage:     true,
				GOSXLink:    true,
			},
			{
				Href:        "/admin/media/font-1",
				Filename:    "brand.woff2",
				ContentType: "font/woff2",
				Size:        "8 KB",
				StatusLabel: "Archived",
				GOSXLink:    true,
			},
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-media-library-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Media library</h1><p>Uploaded images, icons, and font files for storefront content and brand controls.</p></section>`,
		`<section class="panel"><div class="panel__header"><h2>Assets</h2></div><div class="media-grid">`,
		`<a class="media-card" href="/admin/media/image-1" data-gosx-link="true"><img src="/media/cup.jpg" alt="Cup &lt;one&gt;" /><strong>cup.jpg</strong><span>14 KB</span><span>Published</span></a>`,
		`<a class="media-card" href="/admin/media/font-1" data-gosx-link="true"><span class="media-card__file">font/woff2</span><strong>brand.woff2</strong><span>8 KB</span><span>Archived</span></a>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("media library html missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<p class="empty">No media assets yet.</p>`) {
		t.Fatalf("populated media library rendered empty state:\n%s", html)
	}
}

func TestRenderBackendMediaLibraryContentAndEmptyState(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendMediaLibraryContent(BackendMediaLibraryProps{
		Kicker:     "CMS",
		Title:      "Media library",
		Summary:    "Summary",
		PanelTitle: "Library",
		Empty:      "Nothing uploaded.",
	}))

	for _, fragment := range []string{
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Media library</h1><p>Summary</p></section>`,
		`<section class="panel"><div class="panel__header"><h2>Library</h2></div><div class="media-grid"></div><p class="empty">Nothing uploaded.</p></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("media library content missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `data-gosx-studio-backend-media-library-renderer="gosx-studio"`) {
		t.Fatalf("content renderer should not render its own page wrapper:\n%s", html)
	}
}

func TestRenderBackendMediaLibrarySplitNodes(t *testing.T) {
	heading := gosx.RenderHTML(RenderBackendMediaLibraryHeading(BackendMediaLibraryProps{
		Kicker:  "CMS",
		Title:   "Media library",
		Summary: "Summary",
	}))
	assets := gosx.RenderHTML(RenderBackendMediaLibraryAssets(BackendMediaLibraryProps{}))

	if !strings.Contains(heading, `<section class="admin-heading">`) || strings.Contains(heading, `<section class="panel">`) {
		t.Fatalf("heading node should render only the admin heading:\n%s", heading)
	}
	if !strings.Contains(assets, `<section class="panel">`) || strings.Contains(assets, `<section class="admin-heading">`) {
		t.Fatalf("assets node should render only the media panel:\n%s", assets)
	}
	if !strings.Contains(assets, `<p class="empty">No media assets yet.</p>`) {
		t.Fatalf("assets node missing default empty state:\n%s", assets)
	}
}
