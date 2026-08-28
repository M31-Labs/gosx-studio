package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendPageIndexPagePreservesCreateFormContract(t *testing.T) {
	resourceIndex := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("resource index"))
	html := gosx.RenderHTML(RenderBackendPageIndexPage(BackendPageIndexPageProps{
		ResourceIndexContent: resourceIndex,
		CreateAction:         "/admin/pages/__actions/create",
		CSRFToken:            "csrf-token",
		CreateStatus: BackendPageIndexActionStatus{
			Submitted:   true,
			Message:     "Please correct the highlighted fields.",
			FieldErrors: map[string]string{"title": "Title is required."},
		},
		Media: []BackendPageIndexMediaAsset{{
			URL:      "/media/noni.jpg",
			Filename: "noni.jpg",
			Alt:      "Noni fruit",
		}},
		Page: BackendPageIndexValues{
			Title:           "About <Noni>",
			Slug:            "about-noni",
			Body:            `[{"type":"paragraph","text":"Body"}]`,
			MetaTitle:       "About meta",
			MetaImageURL:    "/media/noni.jpg",
			MetaDescription: "Meta description",
			MetaImageAlt:    "Meta alt",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-page-index-renderer="gosx-studio">`,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add page</h2></div>`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<datalist id="page-media-urls"><option value="/media/noni.jpg" label="noni.jpg" data-media-alt="Noni fruit">noni.jpg</option></datalist>`,
		`<form class="admin-form" method="post" action="/admin/pages/__actions/create" data-content-editor-enhanced-submit="true">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<div class="button-row admin-form__primary-actions" data-content-editor-primary-actions="true">`,
		`<input id="title" name="title" value="About &lt;Noni&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="slug" name="slug" value="about-noni" />`,
		`<select id="bodyFormat" name="bodyFormat" data-content-editor-format="true"><option value="blocks" selected="selected">GoSX blocks</option><option value="mdpp">MDPP</option></select>`,
		`<div class="field-row field-row--wide content-editor" data-content-editor="true" data-content-editor-media-list="page-media-urls">`,
		`<textarea class="content-editor__source" id="body" name="body" rows="8" data-content-editor-source="true">[{&#34;type&#34;:&#34;paragraph&#34;,&#34;text&#34;:&#34;Body&#34;}]</textarea>`,
		`<button type="button" class="home-section-move" data-content-add="heading">Heading</button>`,
		`<button type="button" class="home-section-move" data-content-add="quote">Quote</button>`,
		`<div class="content-block-list" data-content-editor-list="true"></div>`,
		`<legend>SEO</legend>`,
		`<input id="metaTitle" name="metaTitle" value="About meta" />`,
		`<input id="metaImageUrl" name="metaImageUrl" list="page-media-urls" value="/media/noni.jpg" data-media-alt-target="metaImageAlt" />`,
		`<textarea id="metaDescription" name="metaDescription" rows="3">Meta description</textarea>`,
		`<input id="metaImageAlt" name="metaImageAlt" value="Meta alt" />`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<button class="button button--primary" type="submit">Create page</button>`,
		`<script src="/_gosx/studio/content-editor.js" data-gosx-script="managed" data-gosx-script-load="dom" data-gosx-script-loaded="pending" defer></script><script src="/_gosx/studio/media-runtime.js" data-gosx-script="managed" data-gosx-script-load="dom" data-gosx-script-loaded="pending" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("page index page missing %q:\n%s", fragment, html)
		}
	}
	if got := strings.Count(html, `>Create page<`); got != 1 {
		t.Fatalf("page index renderer must expose one canonical Create page action, got %d:\n%s", got, html)
	}

	assertFragmentOrder(t, html,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add page</h2></div>`,
		`<datalist id="page-media-urls">`,
		`<form class="admin-form"`,
		`<div class="button-row admin-form__primary-actions" data-content-editor-primary-actions="true">`,
		`<input id="title" name="title"`,
		`<script src="/_gosx/studio/content-editor.js"`,
		`<script src="/_gosx/studio/media-runtime.js"`,
	)
}

func TestRenderBackendPageIndexPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendPageIndexPage(BackendPageIndexPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-page-index-renderer="gosx-studio">`,
		`<section class="panel"><div class="panel__header"><h2>Add page</h2></div>`,
		`<datalist id="page-media-urls"></datalist>`,
		`<form class="admin-form" method="post" action="" data-content-editor-enhanced-submit="true">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<div class="button-row admin-form__primary-actions" data-content-editor-primary-actions="true">`,
		`<input id="title" name="title" value="" />`,
		`<p class="form-error"></p>`,
		`<input id="slug" name="slug" value="" />`,
		`<textarea class="content-editor__source" id="body" name="body" rows="8" data-content-editor-source="true"></textarea>`,
		`<option value="blocks" selected="selected">GoSX blocks</option>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="page-media-urls" value="" data-media-alt-target="metaImageAlt" />`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<button class="button button--primary" type="submit">Create page</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty-safe page index page missing %q:\n%s", fragment, html)
		}
	}
	if got := strings.Count(html, `>Create page<`); got != 1 {
		t.Fatalf("empty page index renderer must expose one canonical Create page action, got %d:\n%s", got, html)
	}
	if strings.Contains(html, `form-status form-status--error`) {
		t.Fatalf("empty-safe page index page rendered unexpected status:\n%s", html)
	}
}
