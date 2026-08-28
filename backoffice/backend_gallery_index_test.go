package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendGalleryIndexPagePreservesCreateFormContract(t *testing.T) {
	resourceIndex := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("resource index"))
	html := gosx.RenderHTML(RenderBackendGalleryIndexPage(BackendGalleryIndexPageProps{
		ResourceIndexContent: resourceIndex,
		CreateAction:         "/admin/gallery/__actions/create",
		CSRFToken:            "csrf-token",
		CreateStatus: BackendGalleryIndexActionStatus{
			Submitted:   true,
			Message:     "Please correct the highlighted fields.",
			FieldErrors: map[string]string{"title": "Title is required."},
		},
		Media: []BackendGalleryIndexMediaAsset{{
			URL:      "/media/vase.jpg",
			Filename: "vase.jpg",
			Alt:      "Stoneware vase",
		}},
		Work: BackendGalleryIndexValues{
			Title:           "Vase <Study>",
			Year:            "2026",
			Materials:       "Stoneware",
			Dimensions:      "8 x 6 in",
			ImagesText:      "/media/vase.jpg|Stoneware vase",
			Description:     "A study",
			MetaTitle:       "Vase meta",
			MetaImageURL:    "/media/vase.jpg",
			MetaDescription: "Meta description",
			MetaImageAlt:    "Meta alt",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-gallery-index-renderer="gosx-studio">`,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add gallery work</h2></div>`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<datalist id="gallery-media-urls"><option value="/media/vase.jpg" label="vase.jpg" data-media-alt="Stoneware vase">vase.jpg</option></datalist>`,
		`<form class="admin-form" method="post" action="/admin/gallery/__actions/create">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input id="title" name="title" value="Vase &lt;Study&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="year" name="year" value="2026" />`,
		`<input id="materials" name="materials" value="Stoneware" />`,
		`<input id="dimensions" name="dimensions" value="8 x 6 in" />`,
		`<textarea id="imagesText" name="imagesText" rows="4" data-media-lines-list="gallery-media-urls">/media/vase.jpg|Stoneware vase</textarea>`,
		`<textarea id="description" name="description" rows="4">A study</textarea>`,
		`<legend>SEO</legend>`,
		`<input id="metaTitle" name="metaTitle" value="Vase meta" />`,
		`<input id="metaImageUrl" name="metaImageUrl" list="gallery-media-urls" value="/media/vase.jpg" data-media-alt-target="metaImageAlt" />`,
		`<textarea id="metaDescription" name="metaDescription" rows="3">Meta description</textarea>`,
		`<input id="metaImageAlt" name="metaImageAlt" value="Meta alt" />`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<input type="checkbox" name="featured" /> Featured`,
		`<button class="button button--primary" type="submit">Create gallery work</button>`,
		`<script src="/_gosx/studio/media-runtime.js" data-gosx-script="managed" data-gosx-script-load="dom" data-gosx-script-loaded="pending" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("gallery index page missing %q:\n%s", fragment, html)
		}
	}

	assertFragmentOrder(t, html,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add gallery work</h2></div>`,
		`<datalist id="gallery-media-urls">`,
		`<form class="admin-form"`,
		`<script src="/_gosx/studio/media-runtime.js"`,
	)
}

func TestRenderBackendGalleryIndexPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendGalleryIndexPage(BackendGalleryIndexPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-gallery-index-renderer="gosx-studio">`,
		`<section class="panel"><div class="panel__header"><h2>Add gallery work</h2></div>`,
		`<datalist id="gallery-media-urls"></datalist>`,
		`<form class="admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input id="title" name="title" value="" />`,
		`<p class="form-error"></p>`,
		`<textarea id="imagesText" name="imagesText" rows="4" data-media-lines-list="gallery-media-urls"></textarea>`,
		`<textarea id="description" name="description" rows="4"></textarea>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="gallery-media-urls" value="" data-media-alt-target="metaImageAlt" />`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<input type="checkbox" name="featured" /> Featured`,
		`<button class="button button--primary" type="submit">Create gallery work</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty-safe gallery index page missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `form-status form-status--error`) {
		t.Fatalf("empty-safe gallery index page rendered unexpected status:\n%s", html)
	}
}
