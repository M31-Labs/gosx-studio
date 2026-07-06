package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendDetailRendersTextPreview(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendDetail(BackendDetailProps{
		Kicker:  "CMS",
		Title:   "Edit category",
		Summary: "Tea <Special>",
		Preview: BackendDetailPreview{
			TextOnly: true,
			Primary:  "/shop/category/tea-special",
			Statuses: []BackendDetailStatus{{Label: "Active"}},
			Times: []BackendDetailTime{{
				Label:   "Jun 28, 2026 4:45 PM",
				Machine: "2026-06-28T16:45:00Z",
			}},
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Edit category</h1><p>Tea &lt;Special&gt;</p></section>`,
		`<div class="edit-preview edit-preview--text"><div><span class="status">Active</span><strong>/shop/category/tea-special</strong><time datetime="2026-06-28T16:45:00Z" data-viewer-time="datetime">Jun 28, 2026 4:45 PM</time></div></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("detail html missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendDetailRendersImagePreviewAndExtraNode(t *testing.T) {
	extra := gosx.El("span", gosx.Attrs(gosx.Attr("class", "field-note")), gosx.Text("Draft preview"))
	html := gosx.RenderHTML(RenderBackendDetailPreview(BackendDetailPreview{
		Image:   BackendDetailPreviewImage{Src: "/media/cup.jpg", Alt: "Cup <blue>"},
		Primary: "Blue cup",
		Statuses: []BackendDetailStatus{
			{Label: "Draft", ClassName: "status status--muted"},
		},
		Times: []BackendDetailTime{{
			Label:          "Jun 28, 2026",
			Machine:        "2026-06-28",
			ClassName:      "field-note",
			ViewerTimeMode: "date",
		}},
		Extra: &extra,
	}))

	for _, fragment := range []string{
		`<div class="edit-preview"><img src="/media/cup.jpg" alt="Cup &lt;blue&gt;" /><div>`,
		`<span class="status status--muted">Draft</span><strong>Blue cup</strong>`,
		`<time class="field-note" datetime="2026-06-28" data-viewer-time="date">Jun 28, 2026</time>`,
		`<span class="field-note">Draft preview</span>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("detail preview html missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendDetailPreviewRendersCustomLeadNode(t *testing.T) {
	lead := gosx.El("span", gosx.Attrs(gosx.Attr("class", "media-card__file")), gosx.Text("font/woff2"))
	html := gosx.RenderHTML(RenderBackendDetailPreview(BackendDetailPreview{
		Lead:    &lead,
		Primary: "brand.woff2",
		Statuses: []BackendDetailStatus{
			{Label: "Active"},
			{Label: "font/woff2"},
		},
	}))

	want := `<div class="edit-preview"><span class="media-card__file">font/woff2</span><div><span class="status">Active</span><span class="status">font/woff2</span><strong>brand.woff2</strong></div></div>`
	if !strings.Contains(html, want) {
		t.Fatalf("detail preview html missing custom lead node %q:\n%s", want, html)
	}
	if strings.Contains(html, `<img`) {
		t.Fatalf("custom lead preview should not render an image:\n%s", html)
	}
}

func TestRenderBackendDetailPreviewTextOnlySuppressesLeadAndImage(t *testing.T) {
	lead := gosx.El("span", gosx.Attrs(gosx.Attr("class", "media-card__file")), gosx.Text("font/woff2"))
	html := gosx.RenderHTML(RenderBackendDetailPreview(BackendDetailPreview{
		TextOnly: true,
		Lead:     &lead,
		Image:    BackendDetailPreviewImage{Src: "/media/cup.jpg", Alt: "Cup"},
		Primary:  "Text preview",
	}))

	want := `<div class="edit-preview edit-preview--text"><div><strong>Text preview</strong></div></div>`
	if !strings.Contains(html, want) {
		t.Fatalf("text-only preview html changed; missing %q:\n%s", want, html)
	}
	if strings.Contains(html, `media-card__file`) || strings.Contains(html, `<img`) {
		t.Fatalf("text-only preview should suppress lead and image:\n%s", html)
	}
}

func TestRenderBackendDetailSplitNodes(t *testing.T) {
	heading := gosx.RenderHTML(RenderBackendDetailHeading(BackendDetailProps{
		Kicker:  "CMS",
		Title:   "Edit",
		Summary: "Summary",
	}))
	content := gosx.RenderHTML(RenderBackendDetailContent(BackendDetailProps{
		Kicker: "CMS",
		Title:  "Edit",
		Preview: BackendDetailPreview{
			TextOnly: true,
			Primary:  "Preview",
		},
	}))

	if !strings.Contains(heading, `<section class="admin-heading">`) || strings.Contains(heading, `edit-preview`) {
		t.Fatalf("heading node should render only the admin heading:\n%s", heading)
	}
	if !strings.Contains(content, `<section class="admin-heading">`) || !strings.Contains(content, `<div class="edit-preview edit-preview--text">`) {
		t.Fatalf("content node should render heading and preview:\n%s", content)
	}
	if strings.Contains(content, `data-gosx-studio-backend-detail-renderer="gosx-studio"`) {
		t.Fatalf("content renderer should not render its own page wrapper:\n%s", content)
	}
}
