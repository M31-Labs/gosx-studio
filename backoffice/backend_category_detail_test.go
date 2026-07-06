package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendCategoryDetailPagePreservesFormContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendCategoryDetailPage(BackendCategoryDetailPageProps{
		Heading:       gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("heading")),
		Preview:       gosx.El("div", gosx.Attrs(gosx.Attr("class", "edit-preview edit-preview--text")), gosx.Text("preview")),
		SaveAction:    "/admin/categories/cat_1/__actions/save",
		ArchiveAction: "/admin/categories/cat_1/__actions/archive",
		RestoreAction: "/admin/categories/cat_1/__actions/restore",
		CSRFToken:     "csrf-token",
		SaveStatus: BackendCategoryDetailActionStatus{
			Submitted: true,
			Message:   "Please correct the highlighted fields.",
			FieldErrors: map[string]string{
				"title": "Title is required.",
			},
		},
		ArchiveStatus: BackendCategoryDetailActionStatus{Submitted: true, Message: "Archive failed."},
		RestoreStatus: BackendCategoryDetailActionStatus{Submitted: true, Message: "Restore failed."},
		Category: BackendCategoryDetailValues{
			ID:         "cat_1",
			Title:      "Tea <Special>",
			Slug:       "tea-special",
			CanArchive: true,
			CanRestore: true,
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-category-detail-renderer="gosx-studio">`,
		`<section class="admin-heading">heading</section>`,
		`<form class="panel admin-form" method="post" action="/admin/categories/cat_1/__actions/save">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input type="hidden" name="id" value="cat_1" />`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<p class="form-status form-status--error">Archive failed.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<div class="edit-preview edit-preview--text">preview</div>`,
		`<label for="title">Title</label><input id="title" name="title" value="Tea &lt;Special&gt;" /><p class="form-error">Title is required.</p>`,
		`<label for="slug">Slug</label><input id="slug" name="slug" value="tea-special" />`,
		`<button class="button button--primary" type="submit">Save category</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/categories/cat_1/__actions/archive" data-admin-confirm="Archive this category? Category links will be removed from public browsing.">Archive</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/categories/cat_1/__actions/restore" data-admin-confirm="Restore this category?">Restore</button>`,
		`<a class="button button--secondary" href="/admin/categories" data-gosx-link="true">Back to categories</a>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("category detail renderer missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`<section class="admin-heading">heading</section>`,
		`<form class="panel admin-form"`,
		`<div class="edit-preview edit-preview--text">preview</div>`,
		`<input id="title"`,
		`<input id="slug"`,
		`<button class="button button--primary" type="submit">Save category</button>`,
		`formaction="/admin/categories/cat_1/__actions/archive"`,
		`formaction="/admin/categories/cat_1/__actions/restore"`,
		`<a class="button button--secondary" href="/admin/categories"`,
	)
}

func TestRenderBackendCategoryDetailPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendCategoryDetailPage(BackendCategoryDetailPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-category-detail-renderer="gosx-studio">`,
		`<form class="panel admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input type="hidden" name="id" value="" />`,
		`<input id="title" name="title" value="" />`,
		`<p class="form-error"></p>`,
		`<input id="slug" name="slug" value="" />`,
		`<button class="button button--primary" type="submit">Save category</button>`,
		`<a class="button button--secondary" href="/admin/categories" data-gosx-link="true">Back to categories</a>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty category detail renderer missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{
		`formaction=""`,
		`form-status`,
		`data-admin-confirm`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty category detail renderer should not contain %q:\n%s", notWant, html)
		}
	}
}
