package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendBlogIndexPagePreservesCreateFormContract(t *testing.T) {
	resourceIndex := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("resource index"))
	html := gosx.RenderHTML(RenderBackendBlogIndexPage(BackendBlogIndexPageProps{
		ResourceIndexContent: resourceIndex,
		CreateAction:         "/admin/blog/__actions/create",
		CSRFToken:            "csrf-token",
		CreateStatus: BackendBlogIndexActionStatus{
			Submitted:   true,
			Message:     "Please correct the highlighted fields.",
			FieldErrors: map[string]string{"title": "Title is required.", "publishedAt": "Use a valid publish date."},
		},
		Media: []BackendBlogIndexMediaAsset{{
			URL:      "/media/noni.jpg",
			Filename: "noni.jpg",
			Alt:      "Noni fruit",
		}},
		Tags: []BackendBlogIndexTag{{
			Label: "Care",
			Slug:  "care",
			Href:  "/blog?tag=care",
		}},
		Products: []BackendBlogIndexRelation{{
			FieldName: "product_prod_1",
			Title:     "Mug",
		}},
		GalleryWorks: []BackendBlogIndexRelation{{
			FieldName: "gallery_work_1",
			Title:     "Wall piece",
		}},
		Post: BackendBlogIndexValues{
			Title:           "Care <Guide>",
			Slug:            "care-guide",
			AuthorName:      "Noni",
			PublishedAt:     "2026-06-29T09:00",
			CoverImage:      "/media/noni.jpg",
			CoverImageAlt:   "Noni fruit",
			Excerpt:         "Care notes",
			Body:            `[{"type":"paragraph","text":"Body"}]`,
			Tags:            "Care",
			MetaTitle:       "Care meta",
			MetaImageURL:    "/media/noni.jpg",
			MetaDescription: "Meta description",
			MetaImageAlt:    "Meta alt",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-blog-index-renderer="gosx-studio">`,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add post</h2></div>`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<datalist id="blog-media-urls"><option value="/media/noni.jpg" label="noni.jpg" data-media-alt="Noni fruit">noni.jpg</option></datalist>`,
		`<datalist id="blog-tags"><option value="Care">care</option></datalist>`,
		`<form class="admin-form" method="post" action="/admin/blog/__actions/create">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input id="title" name="title" value="Care &lt;Guide&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="slug" name="slug" value="care-guide" />`,
		`<input id="authorName" name="authorName" value="Noni" />`,
		`<input id="publishedAt" name="publishedAt" value="2026-06-29T09:00" type="datetime-local" data-viewer-datetime-input="true" />`,
		`<p class="form-error">Use a valid publish date.</p>`,
		`<input id="coverImage" name="coverImage" list="blog-media-urls" value="/media/noni.jpg" data-media-alt-target="coverImageAlt" />`,
		`<input id="coverImageAlt" name="coverImageAlt" value="Noni fruit" />`,
		`<textarea id="excerpt" name="excerpt" rows="3">Care notes</textarea>`,
		`<select id="bodyFormat" name="bodyFormat" data-content-editor-format="true"><option value="blocks" selected="selected">GoSX blocks</option><option value="mdpp">MDPP</option></select>`,
		`<div class="field-row field-row--wide content-editor" data-content-editor="true" data-content-editor-media-list="blog-media-urls">`,
		`<textarea class="content-editor__source" id="body" name="body" rows="9" data-content-editor-source="true">[{&#34;type&#34;:&#34;paragraph&#34;,&#34;text&#34;:&#34;Body&#34;}]</textarea>`,
		`<button type="button" class="home-section-move" data-content-add="heading">Heading</button>`,
		`<button type="button" class="home-section-move" data-content-add="quote">Quote</button>`,
		`<div class="content-block-list" data-content-editor-list="true"></div>`,
		`<input id="tags" name="tags" value="Care" list="blog-tags" />`,
		`<div class="tag-list"><a href="/blog?tag=care">Care</a></div>`,
		`<legend>SEO</legend>`,
		`<input id="metaTitle" name="metaTitle" value="Care meta" />`,
		`<input id="metaImageUrl" name="metaImageUrl" list="blog-media-urls" value="/media/noni.jpg" data-media-alt-target="metaImageAlt" />`,
		`<textarea id="metaDescription" name="metaDescription" rows="3">Meta description</textarea>`,
		`<input id="metaImageAlt" name="metaImageAlt" value="Meta alt" />`,
		`<legend>Related products</legend>`,
		`<input type="checkbox" name="product_prod_1" /> Mug`,
		`<legend>Related gallery works</legend>`,
		`<input type="checkbox" name="gallery_work_1" /> Wall piece`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<input type="checkbox" name="featured" /> Featured`,
		`<button class="button button--primary" type="submit">Create post</button>`,
		`<script src="/content-editor.js" defer></script><script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("blog index page missing %q:\n%s", fragment, html)
		}
	}

	assertFragmentOrder(t, html,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add post</h2></div>`,
		`<datalist id="blog-media-urls">`,
		`<datalist id="blog-tags">`,
		`<form class="admin-form"`,
		`<script src="/content-editor.js"`,
		`<script src="/media-picker.js"`,
	)
}

func TestRenderBackendBlogIndexPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendBlogIndexPage(BackendBlogIndexPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-blog-index-renderer="gosx-studio">`,
		`<section class="panel"><div class="panel__header"><h2>Add post</h2></div>`,
		`<datalist id="blog-media-urls"></datalist>`,
		`<datalist id="blog-tags"></datalist>`,
		`<form class="admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input id="title" name="title" value="" />`,
		`<p class="form-error"></p>`,
		`<textarea class="content-editor__source" id="body" name="body" rows="9" data-content-editor-source="true"></textarea>`,
		`<option value="blocks" selected="selected">GoSX blocks</option>`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<input type="checkbox" name="featured" /> Featured`,
		`<button class="button button--primary" type="submit">Create post</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty-safe blog index page missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`form-status form-status--error`,
		`<legend>Related products</legend>`,
		`<legend>Related gallery works</legend>`,
		`<div class="tag-list">`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty-safe blog index page rendered unexpected %q:\n%s", notWant, html)
		}
	}
}

func assertFragmentOrder(t *testing.T, html string, fragments ...string) {
	t.Helper()
	offset := 0
	for _, fragment := range fragments {
		index := strings.Index(html[offset:], fragment)
		if index < 0 {
			t.Fatalf("fragment %q not found after offset %d:\n%s", fragment, offset, html)
		}
		offset += index + len(fragment)
	}
}
