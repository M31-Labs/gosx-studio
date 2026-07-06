package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendMediaDetailPagePreservesFormContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendMediaDetailPage(BackendMediaDetailPageProps{
		Heading:       gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("heading")),
		Preview:       gosx.El("div", gosx.Attrs(gosx.Attr("class", "edit-preview")), gosx.Text("preview")),
		SaveAction:    "/admin/media/media_1/__actions/save",
		ArchiveAction: "/admin/media/media_1/__actions/archive",
		RestoreAction: "/admin/media/media_1/__actions/restore",
		ReplaceAction: "/admin/media/replace",
		CSRFToken:     "csrf-token",
		SaveStatus: BackendMediaDetailActionStatus{
			Submitted: true,
			Message:   "Please correct the highlighted fields.",
			FieldErrors: map[string]string{
				"url":    "URL is required.",
				"focalX": "Use a value from 0 to 1.",
				"focalY": "Use a value from 0 to 1.",
			},
		},
		ArchiveStatus: BackendMediaDetailActionStatus{Submitted: true, Message: "Archive failed."},
		RestoreStatus: BackendMediaDetailActionStatus{Submitted: true, Message: "Restore failed."},
		Replaced:      true,
		HasError:      true,
		Error:         "Upload failed.",
		Asset: BackendMediaDetailAsset{
			ID:            "media_1",
			URL:           "/media/cup.jpg",
			Alt:           "Cup <blue>",
			Filename:      "cup.jpg",
			ContentType:   "image/jpeg",
			FocalX:        "0.25",
			FocalY:        "0.75",
			FocalXPercent: "25%",
			FocalYPercent: "75%",
			IsImage:       true,
			CanArchive:    true,
			CanRestore:    true,
			Variants: []BackendMediaDetailVariant{{
				URL:        "/media/cup-thumb.jpg",
				Name:       "thumb",
				Dimensions: "320 x 240",
				Size:       "8 KB",
			}},
		},
		Usage: []BackendMediaDetailUsage{{
			Href:  "/admin/products/prod_1",
			Title: "Blue Cup",
			Kind:  "Product",
		}},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-media-detail-renderer="gosx-studio">`,
		`<section class="admin-heading">heading</section>`,
		`<form class="panel admin-form" method="post" action="/admin/media/media_1/__actions/save">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input type="hidden" name="id" value="media_1" />`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<p class="form-status form-status--error">Archive failed.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<p class="form-status form-status--ok">Media file replaced.</p>`,
		`<p class="form-status form-status--error">Upload failed.</p>`,
		`<div class="edit-preview">preview</div>`,
		`<input id="url" name="url" value="/media/cup.jpg" />`,
		`<p class="form-error">URL is required.</p>`,
		`<input id="alt" name="alt" value="Cup &lt;blue&gt;" />`,
		`<input id="filename" name="filename" value="cup.jpg" />`,
		`<input id="contentType" name="contentType" value="image/jpeg" />`,
		`<input id="focalX" name="focalX" type="number" min="0" max="1" step="0.01" inputmode="decimal" data-media-focal-x="true" value="0.25" />`,
		`<p class="form-error">Use a value from 0 to 1.</p>`,
		`<input id="focalY" name="focalY" type="number" min="0" max="1" step="0.01" inputmode="decimal" data-media-focal-y="true" value="0.75" />`,
		`<button class="media-focal-preview" type="button" aria-label="Set focal point" data-media-focal-preview="true"><img src="/media/cup.jpg" alt="Cup &lt;blue&gt;" /><span style="left:25%; top:75%" data-media-focal-marker="true"></span></button>`,
		`<label>Variants</label><ul class="field-list"><li><a href="/media/cup-thumb.jpg">thumb</a><span>320 x 240 8 KB</span></li></ul>`,
		`<label>Used by</label><ul class="field-list"><li><a href="/admin/products/prod_1" data-gosx-link="true">Blue Cup</a><span>Product</span></li></ul>`,
		`<button class="button button--primary" type="submit">Save media</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/media/media_1/__actions/archive" data-admin-confirm="Archive this media asset? Existing content may still reference this URL.">Archive</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/media/media_1/__actions/restore" data-admin-confirm="Restore this media asset?">Restore</button>`,
		`<a class="button button--secondary" href="/admin/media" data-gosx-link="true">Back to media</a>`,
		`<section class="panel"><div class="panel__header"><h2>Replace file</h2></div><form class="admin-form admin-form--single" method="post" action="/admin/media/replace" enctype="multipart/form-data" data-admin-confirm="Replace this media file? Existing references will use the new file.">`,
		`<input id="replaceFile" name="file" type="file" accept="image/png,image/jpeg,image/webp,image/gif,image/x-icon,.ico,.woff,.woff2,.ttf,.otf" />`,
		`<input id="replaceAlt" name="alt" value="Cup &lt;blue&gt;" />`,
		`<button class="button button--secondary" type="submit">Replace file</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("media detail renderer missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`<section class="admin-heading">heading</section>`,
		`<form class="panel admin-form"`,
		`<div class="edit-preview">preview</div>`,
		`<input id="url"`,
		`data-media-focal-preview="true"`,
		`<label>Variants</label>`,
		`<label>Used by</label>`,
		`<button class="button button--primary" type="submit">Save media</button>`,
		`formaction="/admin/media/media_1/__actions/archive"`,
		`formaction="/admin/media/media_1/__actions/restore"`,
		`<a class="button button--secondary" href="/admin/media"`,
		`<section class="panel"><div class="panel__header"><h2>Replace file</h2></div>`,
	)
}

func TestRenderBackendMediaDetailPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendMediaDetailPage(BackendMediaDetailPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-media-detail-renderer="gosx-studio">`,
		`<form class="panel admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input type="hidden" name="id" value="" />`,
		`<input id="url" name="url" value="" />`,
		`<input id="focalX" name="focalX" type="number" min="0" max="1" step="0.01" inputmode="decimal" data-media-focal-x="true" value="" />`,
		`<input id="focalY" name="focalY" type="number" min="0" max="1" step="0.01" inputmode="decimal" data-media-focal-y="true" value="" />`,
		`<button class="button button--primary" type="submit">Save media</button>`,
		`<a class="button button--secondary" href="/admin/media" data-gosx-link="true">Back to media</a>`,
		`<form class="admin-form admin-form--single" method="post" action="/admin/media/replace" enctype="multipart/form-data" data-admin-confirm="Replace this media file? Existing references will use the new file.">`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty media detail renderer missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{
		`media-focal-preview`,
		`<label>Variants</label>`,
		`<label>Used by</label>`,
		`formaction=""`,
		`form-status`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty media detail renderer should not contain %q:\n%s", notWant, html)
		}
	}
}

func TestRenderBackendMediaDetailFormOmitsImageOnlyAndOptionalListsForNonImage(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendMediaDetailForm(BackendMediaDetailPageProps{
		Asset: BackendMediaDetailAsset{
			ID:          "font_1",
			Filename:    "brand.woff2",
			ContentType: "font/woff2",
			IsImage:     false,
		},
	}))

	for _, fragment := range []string{
		`<input type="hidden" name="id" value="font_1" />`,
		`<input id="filename" name="filename" value="brand.woff2" />`,
		`<input id="contentType" name="contentType" value="font/woff2" />`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("non-image media detail form missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`media-focal-preview`,
		`<label>Variants</label>`,
		`<label>Used by</label>`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("non-image media detail form should not contain %q:\n%s", notWant, html)
		}
	}
}
