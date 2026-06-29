package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendWorkbenchPopulated(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendWorkbench(BackendWorkbenchProps{
		Resources: []BackendWorkbenchResource{{
			Label:          "Products <live>",
			Description:    "One-off inventory.",
			Route:          "/admin/products",
			MutableLabel:   "Writable",
			GeneratedLabel: "Bespoke surface",
			CountLabel:     "2 records",
			Fields: []BackendWorkbenchField{
				{Label: "Title", Kind: "text"},
				{Label: "Price", Kind: "money"},
			},
			Actions: []BackendWorkbenchAction{
				{Label: "Create product", Kind: "form"},
				{Label: "Save product", Kind: "form"},
			},
		}},
		Tools: []BackendWorkbenchTool{{
			Kind:        "headless-api",
			Label:       "Headless GraphQL",
			Description: "Public content queries.",
			Route:       "/api/graphql",
		}},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-workbench-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">Workbench</p><h1>GoSX internal app surface</h1><p>Resources, fields, server actions, and API tools are defined in Go and rendered through GoSX.</p></section>`,
		`<section class="workbench-grid"><article class="panel resource-detail">`,
		`<h2>Products &lt;live&gt;</h2><p class="field-note">One-off inventory.</p>`,
		`<a class="button button--secondary" href="/admin/products" data-gosx-link="true">Open</a>`,
		`<div class="capability-row"><span class="status">Writable</span><span class="status">Bespoke surface</span><span class="status">2 records</span></div>`,
		`<h3>Fields</h3><ul class="field-list"><li><strong>Title</strong><span>text</span></li><li><strong>Price</strong><span>money</span></li></ul>`,
		`<h3>Actions</h3><ul class="field-list"><li><strong>Create product</strong><span>form</span></li><li><strong>Save product</strong><span>form</span></li></ul>`,
		`<section class="panel"><div class="panel__header"><h2>Runtime tools</h2></div><div class="tool-list"><article><span class="status">headless-api</span><strong>Headless GraphQL</strong><p>Public content queries.</p><span class="field-note">/api/graphql</span></article></div></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("workbench html missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<p class="empty">No direct actions.</p>`) {
		t.Fatalf("populated action list rendered empty state:\n%s", html)
	}
}

func TestRenderBackendWorkbenchEmptyish(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendWorkbench(BackendWorkbenchProps{
		Resources: []BackendWorkbenchResource{{
			Label:          "Contacts",
			Description:    "Inbound messages.",
			Route:          "/admin/contacts",
			MutableLabel:   "Read-only",
			GeneratedLabel: "Generated surface",
			CountLabel:     "0 records",
		}},
	}))

	for _, fragment := range []string{
		`data-gosx-studio-backend-workbench-renderer="gosx-studio"`,
		`<section class="workbench-grid"><article class="panel resource-detail">`,
		`<h2>Contacts</h2><p class="field-note">Inbound messages.</p>`,
		`<div class="capability-row"><span class="status">Read-only</span><span class="status">Generated surface</span><span class="status">0 records</span></div>`,
		`<h3>Fields</h3><ul class="field-list"></ul>`,
		`<h3>Actions</h3><p class="empty">No direct actions.</p>`,
		`<section class="panel"><div class="panel__header"><h2>Runtime tools</h2></div><div class="tool-list"></div></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty-ish workbench html missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendWorkbenchContentOmitsWrapper(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendWorkbenchContent(BackendWorkbenchProps{}))
	if !strings.Contains(html, `<section class="admin-heading">`) || !strings.Contains(html, `<section class="workbench-grid"></section>`) {
		t.Fatalf("content renderer missing expected sections:\n%s", html)
	}
	if strings.Contains(html, `data-gosx-studio-backend-workbench-renderer`) {
		t.Fatalf("content renderer should not render its own page wrapper:\n%s", html)
	}
}
