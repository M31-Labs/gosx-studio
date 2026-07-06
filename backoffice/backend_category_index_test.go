package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendCategoryIndexPagePreservesCreateFormContract(t *testing.T) {
	resourceIndex := RenderBackendResourceIndexContent(BackendResourceIndexProps{
		Kicker:  "CMS",
		Title:   "Categories",
		Summary: "Storefront taxonomy for grouping products and shaping headless commerce queries.",
		Empty:   "No categories yet.",
		Table: &BackendResourceTable{
			Headers: []string{"Category", "Slug", "Status", "Updated", ""},
			Rows: []BackendResourceTableRow{{
				Cells: []BackendResourceTableCell{
					{Primary: "Tea <Special>"},
					{Text: "tea-special"},
					{Text: "Active"},
					{Time: BackendResourceTime{Label: "Jun 28, 2026 4:45 PM", Machine: "2026-06-28T16:45:00Z"}},
				},
				Action: BackendResourceLink{Href: "/admin/categories/category-1", Label: "Edit", GOSXLink: true},
			}},
		},
	})
	html := gosx.RenderHTML(RenderBackendCategoryIndexPage(BackendCategoryIndexPageProps{
		ResourceIndexContent: resourceIndex,
		CreateAction:         "/admin/categories/__actions/create",
		CSRFToken:            "csrf-token",
		CreateStatus: BackendCategoryIndexActionStatus{
			Submitted:   true,
			Message:     "Please correct the highlighted fields.",
			FieldErrors: map[string]string{"title": "Title is required."},
		},
		Category: BackendCategoryIndexValues{
			Title: "Tea <Draft>",
			Slug:  "tea-draft",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-resource-index-renderer="gosx-studio" data-gosx-studio-backend-category-index-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Categories</h1><p>Storefront taxonomy for grouping products and shaping headless commerce queries.</p></section>`,
		`<section class="panel"><table class="data-table">`,
		`<td><strong>Tea &lt;Special&gt;</strong></td>`,
		`<td><a href="/admin/categories/category-1" data-gosx-link="true">Edit</a></td>`,
		`<section class="panel"><div class="panel__header"><h2>Add category</h2></div>`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<form class="admin-form" method="post" action="/admin/categories/__actions/create">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<label for="title">Title</label><input id="title" name="title" value="Tea &lt;Draft&gt;" /><p class="form-error">Title is required.</p>`,
		`<label for="slug">Slug</label><input id="slug" name="slug" value="tea-draft" />`,
		`<button class="button button--primary" type="submit">Create category</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("category index page missing %q:\n%s", fragment, html)
		}
	}

	assertCategoryIndexOrder(t, html,
		`<section class="admin-heading">`,
		`<section class="panel"><table class="data-table">`,
		`<section class="panel"><div class="panel__header"><h2>Add category</h2></div>`,
		`<form class="admin-form"`,
	)
}

func TestRenderBackendCategoryCreatePanelDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendCategoryCreatePanel(BackendCategoryIndexPageProps{}))

	for _, fragment := range []string{
		`<section class="panel"><div class="panel__header"><h2>Add category</h2></div>`,
		`<form class="admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input id="title" name="title" value="" />`,
		`<p class="form-error"></p>`,
		`<input id="slug" name="slug" value="" />`,
		`<button class="button button--primary" type="submit">Create category</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("category create panel missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `form-status`) {
		t.Fatalf("unsubmitted category create panel rendered status:\n%s", html)
	}
}

func assertCategoryIndexOrder(t *testing.T, html string, fragments ...string) {
	t.Helper()
	last := -1
	for _, fragment := range fragments {
		next := strings.Index(html, fragment)
		if next == -1 {
			t.Fatalf("category index html missing ordered fragment %q:\n%s", fragment, html)
		}
		if next < last {
			t.Fatalf("category index fragment %q rendered out of order:\n%s", fragment, html)
		}
		last = next
	}
}
