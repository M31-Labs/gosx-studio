package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendGalleryDetailPagePreservesFormContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendGalleryDetailPage(BackendGalleryDetailPageProps{
		Heading: gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("heading")),
		Preview: gosx.El("div", gosx.Attrs(gosx.Attr("class", "edit-preview")), gosx.Text("preview")),
		Media: []BackendGalleryDetailMediaAsset{{
			URL:      "/media/vase.jpg",
			Filename: "vase.jpg",
			Alt:      "Vase alt",
		}},
		SaveAction:    "/admin/gallery/work_1/__actions/save",
		ArchiveAction: "/admin/gallery/work_1/__actions/archive",
		RestoreAction: "/admin/gallery/work_1/__actions/restore",
		CSRFToken:     "csrf-token",
		SaveStatus: BackendGalleryDetailActionStatus{
			Submitted: true,
			Message:   "Please correct the highlighted fields.",
			FieldErrors: map[string]string{
				"title": "Title is required.",
			},
		},
		ArchiveStatus: BackendGalleryDetailActionStatus{Submitted: true, Message: "Archive failed."},
		RestoreStatus: BackendGalleryDetailActionStatus{Submitted: true, Message: "Restore failed."},
		Work: BackendGalleryDetailValues{
			ID:              "work_1",
			Title:           "Vase <Study>",
			Slug:            "vase-study",
			Year:            "2026",
			Materials:       "Clay",
			Dimensions:      "10 in",
			ImagesText:      "/media/vase.jpg|Vase alt",
			Description:     "Stoneware study",
			MetaTitle:       "Vase meta",
			MetaImageURL:    "/media/vase.jpg",
			MetaDescription: "Meta description",
			MetaImageAlt:    "Meta alt",
			Published:       true,
			Featured:        true,
			CanArchive:      true,
			PreviewHref:     "/gallery/vase-study?preview=1",
			Images: []BackendGalleryDetailImage{{
				URL: "/media/vase.jpg",
				Alt: "Vase alt",
			}},
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-gallery-detail-renderer="gosx-studio">`,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="gallery-media-urls"><option value="/media/vase.jpg" label="vase.jpg" data-media-alt="Vase alt">vase.jpg</option></datalist>`,
		`<form class="panel admin-form" method="post" action="/admin/gallery/work_1/__actions/save">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input type="hidden" name="id" value="work_1" />`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<p class="form-status form-status--error">Archive failed.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<div class="edit-preview">preview</div>`,
		`<div class="media-strip media-strip--compact"><img src="/media/vase.jpg" alt="Vase alt" /></div>`,
		`<input id="title" name="title" value="Vase &lt;Study&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="slug" name="slug" value="vase-study" />`,
		`<input id="year" name="year" value="2026" />`,
		`<input id="materials" name="materials" value="Clay" />`,
		`<input id="dimensions" name="dimensions" value="10 in" />`,
		`<textarea id="imagesText" name="imagesText" rows="5" data-media-lines-list="gallery-media-urls">/media/vase.jpg|Vase alt</textarea>`,
		`<textarea id="description" name="description" rows="5">Stoneware study</textarea>`,
		`<legend>SEO</legend>`,
		`<input id="metaTitle" name="metaTitle" value="Vase meta" />`,
		`<input id="metaImageUrl" name="metaImageUrl" list="gallery-media-urls" value="/media/vase.jpg" data-media-alt-target="metaImageAlt" />`,
		`<textarea id="metaDescription" name="metaDescription" rows="3">Meta description</textarea>`,
		`<input id="metaImageAlt" name="metaImageAlt" value="Meta alt" />`,
		`<input type="checkbox" name="published" checked /> Published`,
		`<input type="checkbox" name="featured" checked /> Featured`,
		`<button class="button button--primary" type="submit">Save gallery work</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/gallery/work_1/__actions/archive" data-admin-confirm="Archive this gallery work? It will be hidden from public gallery pages.">Archive</button>`,
		`<a class="button button--secondary" href="/gallery/vase-study?preview=1" data-gosx-link="true">Preview</a>`,
		`<a class="button button--secondary" href="/admin/gallery" data-gosx-link="true">Back to gallery</a>`,
		`<script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("gallery detail renderer missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="gallery-media-urls">`,
		`<form class="panel admin-form"`,
		`<div class="edit-preview">preview</div>`,
		`<div class="media-strip media-strip--compact">`,
		`<button class="button button--primary"`,
		`<script src="/media-picker.js"`,
	)
}

func TestRenderBackendGalleryDetailPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendGalleryDetailPage(BackendGalleryDetailPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-gallery-detail-renderer="gosx-studio">`,
		`<datalist id="gallery-media-urls"></datalist>`,
		`<form class="panel admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input type="hidden" name="id" value="" />`,
		`<textarea id="imagesText" name="imagesText" rows="5" data-media-lines-list="gallery-media-urls"></textarea>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="gallery-media-urls" value="" data-media-alt-target="metaImageAlt" />`,
		`<div class="check-row"><label><input type="checkbox" name="published" /> Published</label><label><input type="checkbox" name="featured" /> Featured</label></div>`,
		`<button class="button button--primary" type="submit">Save gallery work</button>`,
		`<a class="button button--secondary" href="" data-gosx-link="true">Preview</a>`,
		`<script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty gallery detail renderer missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{
		`media-strip--compact`,
		`formaction=""`,
		`form-status`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty gallery detail renderer should not contain %q:\n%s", notWant, html)
		}
	}
}
