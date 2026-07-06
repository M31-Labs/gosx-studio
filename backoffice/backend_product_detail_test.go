package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendProductDetailPagePreservesFormContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendProductDetailPage(BackendProductDetailPageProps{
		Heading: gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("heading")),
		Preview: gosx.El("div", gosx.Attrs(gosx.Attr("class", "edit-preview")), gosx.Text("preview")),
		Media: []BackendProductDetailMediaAsset{{
			URL:      "/media/mug.jpg",
			Filename: "mug.jpg",
			Alt:      "Mug alt",
		}},
		SaveAction:    "/admin/products/prod_1/__actions/save",
		ArchiveAction: "/admin/products/prod_1/__actions/archive",
		RestoreAction: "/admin/products/prod_1/__actions/restore",
		CSRFToken:     "csrf-token",
		SaveStatus: BackendProductDetailActionStatus{
			Submitted: true,
			Message:   "Please correct the highlighted fields.",
			FieldErrors: map[string]string{
				"title": "Title is required.",
				"price": "Enter a price in cents.",
			},
		},
		ArchiveStatus: BackendProductDetailActionStatus{Submitted: true, Message: "Archive failed."},
		RestoreStatus: BackendProductDetailActionStatus{Submitted: true, Message: "Restore failed."},
		Product: BackendProductDetailValues{
			ID:               "prod_1",
			Title:            "Mug <Special>",
			Slug:             "mug-special",
			PriceCents:       "4800",
			Currency:         "USD",
			Description:      "Stoneware mug",
			ImagesText:       "/media/mug.jpg|Mug alt",
			MetaTitle:        "Mug meta",
			MetaImageURL:     "/media/mug.jpg",
			MetaDescription:  "Meta description",
			MetaImageAlt:     "Meta alt",
			Height:           "4",
			Width:            "3",
			Depth:            "3",
			DimensionUnits:   "in",
			WeightValue:      "1.2",
			WeightUnits:      "lb",
			IsAvailable:      true,
			Published:        true,
			Featured:         true,
			RequiresShipping: true,
			ShippingNote:     "Ships next week.",
			CanArchive:       true,
			PreviewHref:      "/shop/mug-special?preview=1",
			Gallery: []BackendProductDetailImage{{
				URL: "/media/mug.jpg",
				Alt: "Mug alt",
			}},
		},
		Categories: []BackendProductDetailCategory{{
			FieldName: "category_cups",
			Title:     "Cups",
			Checked:   true,
		}},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-product-detail-renderer="gosx-studio">`,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="product-media-urls"><option value="/media/mug.jpg" label="mug.jpg" data-media-alt="Mug alt">mug.jpg</option></datalist>`,
		`<form class="panel admin-form" method="post" action="/admin/products/prod_1/__actions/save">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input type="hidden" name="id" value="prod_1" />`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<p class="form-status form-status--error">Archive failed.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<div class="edit-preview">preview</div>`,
		`<div class="media-strip media-strip--compact"><img src="/media/mug.jpg" alt="Mug alt" /></div>`,
		`<input id="title" name="title" value="Mug &lt;Special&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="slug" name="slug" value="mug-special" />`,
		`<input id="price" name="price" value="4800" type="number" min="0" />`,
		`<p class="form-error">Enter a price in cents.</p>`,
		`<input id="currency" name="currency" value="USD" />`,
		`<textarea id="description" name="description" rows="5">Stoneware mug</textarea>`,
		`<textarea id="imagesText" name="imagesText" rows="5" data-media-lines-list="product-media-urls">/media/mug.jpg|Mug alt</textarea>`,
		`<legend>SEO</legend>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="product-media-urls" value="/media/mug.jpg" data-media-alt-target="metaImageAlt" />`,
		`<input id="height" name="height" value="4" />`,
		`<input id="width" name="width" value="3" />`,
		`<input id="depth" name="depth" value="3" />`,
		`<input id="dimensionUnits" name="dimensionUnits" value="in" />`,
		`<input id="weightValue" name="weightValue" value="1.2" />`,
		`<input id="weightUnits" name="weightUnits" value="lb" />`,
		`<legend>Availability</legend>`,
		`<input type="radio" name="availability" value="available" checked /> Available`,
		`<legend>Categories</legend>`,
		`<input type="checkbox" name="category_cups" checked /> Cups`,
		`<input type="checkbox" name="published" checked /> Published`,
		`<input type="checkbox" name="featured" checked /> Featured`,
		`<input type="checkbox" name="requiresShipping" checked /> Requires shipping`,
		`<textarea id="shippingNote" name="shippingNote" rows="3">Ships next week.</textarea>`,
		`<button class="button button--primary" type="submit">Save product</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/products/prod_1/__actions/archive" data-admin-confirm="Archive this product? It will be hidden from the storefront.">Archive</button>`,
		`<a class="button button--secondary" href="/shop/mug-special?preview=1" data-gosx-link="true">Preview</a>`,
		`<a class="button button--secondary" href="/admin/products" data-gosx-link="true">Back to products</a>`,
		`<script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("product detail renderer missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="product-media-urls">`,
		`<form class="panel admin-form"`,
		`<div class="edit-preview">preview</div>`,
		`<div class="media-strip media-strip--compact">`,
		`<button class="button button--primary"`,
		`<script src="/media-picker.js"`,
	)
}

func TestRenderBackendProductDetailPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendProductDetailPage(BackendProductDetailPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-product-detail-renderer="gosx-studio">`,
		`<datalist id="product-media-urls"></datalist>`,
		`<form class="panel admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input type="hidden" name="id" value="" />`,
		`<textarea id="imagesText" name="imagesText" rows="5" data-media-lines-list="product-media-urls"></textarea>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="product-media-urls" value="" data-media-alt-target="metaImageAlt" />`,
		`<legend>Availability</legend>`,
		`<div class="check-row"><label><input type="checkbox" name="published" /> Published</label><label><input type="checkbox" name="featured" /> Featured</label><label><input type="checkbox" name="requiresShipping" /> Requires shipping</label></div>`,
		`<button class="button button--primary" type="submit">Save product</button>`,
		`<a class="button button--secondary" href="" data-gosx-link="true">Preview</a>`,
		`<script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty product detail renderer missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{
		`media-strip--compact`,
		`<legend>Categories</legend>`,
		`formaction=""`,
		`form-status`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty product detail renderer should not contain %q:\n%s", notWant, html)
		}
	}
}
