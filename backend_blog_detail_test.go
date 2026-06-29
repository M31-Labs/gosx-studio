package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendBlogDetailPagePreservesFormContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendBlogDetailPage(BackendBlogDetailPageProps{
		Heading: gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("heading")),
		Preview: gosx.El("div", gosx.Attrs(gosx.Attr("class", "edit-preview")), gosx.Text("preview")),
		RevisionHistory: gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "panel"),
			gosx.Attr("data-studio-revision-history", "true"),
		), gosx.Text("history")),
		Media: []BackendBlogDetailMediaAsset{{
			URL:      "/media/noni.jpg",
			Filename: "noni.jpg",
			Alt:      "Noni alt",
		}},
		Tags: []BackendBlogDetailTag{{
			Label: "Care",
			Slug:  "care",
			Href:  "/blog?tag=care",
		}},
		Products: []BackendBlogDetailRelation{{
			FieldName: "product_prod_1",
			Title:     "Mug",
			Checked:   true,
		}},
		GalleryWorks: []BackendBlogDetailRelation{{
			FieldName: "gallery_work_1",
			Title:     "Wall piece",
			Checked:   true,
		}},
		SaveAction:    "/admin/blog/post_1/__actions/save",
		ArchiveAction: "/admin/blog/post_1/__actions/archive",
		RestoreAction: "/admin/blog/post_1/__actions/restore",
		CSRFToken:     "csrf-token",
		SaveStatus: BackendBlogDetailActionStatus{
			Submitted: true,
			Message:   "Please correct the highlighted fields.",
			FieldErrors: map[string]string{
				"title":       "Title is required.",
				"publishedAt": "Use a valid publish date.",
			},
		},
		ArchiveStatus:         BackendBlogDetailActionStatus{Submitted: true, Message: "Archive failed."},
		RestoreStatus:         BackendBlogDetailActionStatus{Submitted: true, Message: "Restore failed."},
		RestoreRevisionStatus: BackendBlogDetailActionStatus{Submitted: true, Message: "Restore revision failed."},
		RevisionRestored:      true,
		Post: BackendBlogDetailValues{
			ID:                     "post_1",
			Title:                  "Care <Guide>",
			Slug:                   "care-guide",
			AuthorName:             "Noni",
			PublishedAtInput:       "2026-06-29T09:00",
			PublishedAtInputSource: "2026-06-29T09:00:00Z",
			CoverImage:             "/media/noni.jpg",
			CoverImageAlt:          "Noni fruit",
			Excerpt:                "Care notes",
			BodyFormatBlocks:       true,
			Body:                   `[{"type":"paragraph","text":"Body"}]`,
			TagSummary:             "Care",
			MetaTitle:              "Care meta",
			MetaImageURL:           "/media/noni.jpg",
			MetaDescription:        "Meta description",
			MetaImageAlt:           "Meta alt",
			Published:              true,
			Featured:               true,
			CanArchive:             true,
			PreviewHref:            "/blog/care-guide?preview=1",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-blog-detail-renderer="gosx-studio">`,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="blog-media-urls"><option value="/media/noni.jpg" label="noni.jpg" data-media-alt="Noni alt">noni.jpg</option></datalist>`,
		`<datalist id="blog-tags"><option value="Care">care</option></datalist>`,
		`<form class="panel admin-form" method="post" action="/admin/blog/post_1/__actions/save">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input type="hidden" name="id" value="post_1" />`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<p class="form-status form-status--error">Archive failed.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<p class="form-status form-status--error">Restore revision failed.</p>`,
		`<p class="form-status form-status--ok">Version restored.</p>`,
		`<div class="edit-preview">preview</div>`,
		`<input id="title" name="title" value="Care &lt;Guide&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="slug" name="slug" value="care-guide" />`,
		`<input id="authorName" name="authorName" value="Noni" />`,
		`<input id="publishedAt" name="publishedAt" value="2026-06-29T09:00" type="datetime-local" data-viewer-datetime-input="true" data-viewer-time-source="2026-06-29T09:00:00Z" />`,
		`<p class="form-error">Use a valid publish date.</p>`,
		`<input id="coverImage" name="coverImage" list="blog-media-urls" value="/media/noni.jpg" data-media-alt-target="coverImageAlt" />`,
		`<input id="coverImageAlt" name="coverImageAlt" value="Noni fruit" />`,
		`<textarea id="excerpt" name="excerpt" rows="3">Care notes</textarea>`,
		`<select id="bodyFormat" name="bodyFormat" data-content-editor-format="true"><option value="blocks" selected>GoSX blocks</option><option value="mdpp">MDPP</option></select>`,
		`<div class="field-row field-row--wide content-editor" data-content-editor="true" data-content-editor-media-list="blog-media-urls">`,
		`<textarea class="content-editor__source" id="body" name="body" rows="10" data-content-editor-source="true">[{&#34;type&#34;:&#34;paragraph&#34;,&#34;text&#34;:&#34;Body&#34;}]</textarea>`,
		`<button type="button" class="home-section-move" data-content-add="heading">Heading</button>`,
		`<button type="button" class="home-section-move" data-content-add="product">Product</button>`,
		`<div class="content-block-list" data-content-editor-list="true"></div>`,
		`<input id="tags" name="tags" value="Care" list="blog-tags" />`,
		`<div class="tag-list"><a href="/blog?tag=care">Care</a></div>`,
		`<legend>SEO</legend>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="blog-media-urls" value="/media/noni.jpg" data-media-alt-target="metaImageAlt" />`,
		`<textarea id="metaDescription" name="metaDescription" rows="3">Meta description</textarea>`,
		`<input id="metaImageAlt" name="metaImageAlt" value="Meta alt" />`,
		`<legend>Related products</legend>`,
		`<input type="checkbox" name="product_prod_1" checked /> Mug`,
		`<legend>Related gallery works</legend>`,
		`<input type="checkbox" name="gallery_work_1" checked /> Wall piece`,
		`<input type="checkbox" name="published" checked /> Published`,
		`<input type="checkbox" name="featured" checked /> Featured`,
		`<button class="button button--primary" type="submit">Save post</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/blog/post_1/__actions/archive" data-admin-confirm="Archive this post? It will be unpublished and hidden from public blog routes.">Archive</button>`,
		`<a class="button button--secondary" href="/blog/care-guide?preview=1" data-gosx-link="true">Preview</a>`,
		`<a class="button button--secondary" href="/admin/blog" data-gosx-link="true">Back to blog</a>`,
		`<section class="panel" data-studio-revision-history="true">history</section>`,
		`<script src="/content-editor.js" defer></script><script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("blog detail renderer missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="blog-media-urls">`,
		`<datalist id="blog-tags">`,
		`<form class="panel admin-form"`,
		`<div class="edit-preview">preview</div>`,
		`<section class="panel" data-studio-revision-history="true">history</section>`,
		`<script src="/content-editor.js"`,
		`<script src="/media-picker.js"`,
	)
}

func TestRenderBackendBlogDetailPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendBlogDetailPage(BackendBlogDetailPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-blog-detail-renderer="gosx-studio">`,
		`<datalist id="blog-media-urls"></datalist>`,
		`<datalist id="blog-tags"></datalist>`,
		`<form class="panel admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input type="hidden" name="id" value="" />`,
		`<input id="publishedAt" name="publishedAt" value="" type="datetime-local" data-viewer-datetime-input="true" data-viewer-time-source="" />`,
		`<select id="bodyFormat" name="bodyFormat" data-content-editor-format="true"><option value="blocks">GoSX blocks</option><option value="mdpp">MDPP</option></select>`,
		`<textarea class="content-editor__source" id="body" name="body" rows="10" data-content-editor-source="true"></textarea>`,
		`<input id="tags" name="tags" value="" list="blog-tags" />`,
		`<div class="check-row"><label><input type="checkbox" name="published" /> Published</label><label><input type="checkbox" name="featured" /> Featured</label></div>`,
		`<button class="button button--primary" type="submit">Save post</button>`,
		`<a class="button button--secondary" href="" data-gosx-link="true">Preview</a>`,
		`<script src="/content-editor.js" defer></script><script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty blog detail renderer missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{
		`<legend>Related products</legend>`,
		`<legend>Related gallery works</legend>`,
		`<div class="tag-list">`,
		`formaction=""`,
		`form-status`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty blog detail renderer should not contain %q:\n%s", notWant, html)
		}
	}
}
