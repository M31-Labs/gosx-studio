package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendProductIndexPagePreservesCreateFormContract(t *testing.T) {
	resourceIndex := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("resource index"))
	html := gosx.RenderHTML(RenderBackendProductIndexPage(BackendProductIndexPageProps{
		ResourceIndexContent: resourceIndex,
		CreateAction:         "/admin/products/__actions/create",
		CSRFToken:            "csrf-token",
		CreateStatus: BackendProductIndexActionStatus{
			Submitted:   true,
			Message:     "Please correct the highlighted fields.",
			FieldErrors: map[string]string{"title": "Title is required.", "price": "Enter a price in cents."},
		},
		Media: []BackendProductIndexMediaAsset{{
			URL:      "/media/mug.jpg",
			Filename: "mug.jpg",
			Alt:      "Mug alt",
		}},
		Categories: []BackendProductIndexCategory{{
			FieldName: "category_cups",
			Title:     "Cups",
		}},
		Product: BackendProductIndexValues{
			Title:           "Mug <Special>",
			Price:           "4800",
			ImagesText:      "/media/mug.jpg|Mug alt",
			Description:     "Stoneware mug",
			MetaTitle:       "Mug meta",
			MetaImageURL:    "/media/mug.jpg",
			MetaDescription: "Meta description",
			MetaImageAlt:    "Meta alt",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-product-index-renderer="gosx-studio">`,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add product</h2></div>`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<datalist id="product-media-urls"><option value="/media/mug.jpg" label="mug.jpg" data-media-alt="Mug alt">mug.jpg</option></datalist>`,
		`<form class="admin-form" method="post" action="/admin/products/__actions/create">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input id="title" name="title" value="Mug &lt;Special&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="price" name="price" value="4800" type="number" min="0" />`,
		`<p class="form-error">Enter a price in cents.</p>`,
		`<textarea id="imagesText" name="imagesText" rows="4" data-media-lines-list="product-media-urls">/media/mug.jpg|Mug alt</textarea>`,
		`<textarea id="description" name="description" rows="4">Stoneware mug</textarea>`,
		`<legend>SEO</legend>`,
		`<input id="metaTitle" name="metaTitle" value="Mug meta" />`,
		`<input id="metaImageUrl" name="metaImageUrl" list="product-media-urls" value="/media/mug.jpg" data-media-alt-target="metaImageAlt" />`,
		`<textarea id="metaDescription" name="metaDescription" rows="3">Meta description</textarea>`,
		`<input id="metaImageAlt" name="metaImageAlt" value="Meta alt" />`,
		`<legend>Categories</legend>`,
		`<input type="checkbox" name="category_cups" /> Cups`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<input type="checkbox" name="featured" /> Featured`,
		`<input type="checkbox" name="requiresShipping" checked="checked" /> Requires shipping`,
		`<button class="button button--primary" type="submit">Create product</button>`,
		`<script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("product index page missing %q:\n%s", fragment, html)
		}
	}

	assertFragmentOrder(t, html,
		`<section class="admin-heading">resource index</section>`,
		`<section class="panel"><div class="panel__header"><h2>Add product</h2></div>`,
		`<datalist id="product-media-urls">`,
		`<form class="admin-form"`,
		`<script src="/media-picker.js"`,
	)
}

func TestRenderBackendProductIndexPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendProductIndexPage(BackendProductIndexPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-product-index-renderer="gosx-studio">`,
		`<section class="panel"><div class="panel__header"><h2>Add product</h2></div>`,
		`<datalist id="product-media-urls"></datalist>`,
		`<form class="admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input id="title" name="title" value="" />`,
		`<p class="form-error"></p>`,
		`<input id="price" name="price" value="" type="number" min="0" />`,
		`<textarea id="imagesText" name="imagesText" rows="4" data-media-lines-list="product-media-urls"></textarea>`,
		`<textarea id="description" name="description" rows="4"></textarea>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="product-media-urls" value="" data-media-alt-target="metaImageAlt" />`,
		`<input type="checkbox" name="published" checked="checked" /> Published`,
		`<input type="checkbox" name="featured" /> Featured`,
		`<input type="checkbox" name="requiresShipping" checked="checked" /> Requires shipping`,
		`<button class="button button--primary" type="submit">Create product</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty-safe product index page missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`form-status form-status--error`,
		`<legend>Categories</legend>`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty-safe product index page rendered unexpected %q:\n%s", notWant, html)
		}
	}
}
