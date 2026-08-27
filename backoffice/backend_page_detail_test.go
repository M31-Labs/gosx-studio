package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendPageDetailPagePreservesFormContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendPageDetailPage(BackendPageDetailPageProps{
		Heading: gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")), gosx.Text("heading")),
		Preview: gosx.El("div", gosx.Attrs(gosx.Attr("class", "edit-preview edit-preview--text")), gosx.Text("preview")),
		RevisionHistory: gosx.El("section", gosx.Attrs(
			gosx.Attr("class", "panel"),
			gosx.Attr("data-studio-revision-history", "true"),
		), gosx.Text("history")),
		Media: []BackendPageDetailMediaAsset{{
			URL:      "/media/noni.jpg",
			Filename: "noni.jpg",
			Alt:      "Noni alt",
		}},
		SaveAction:    "/admin/pages/page_1/__actions/save",
		ArchiveAction: "/admin/pages/page_1/__actions/archive",
		RestoreAction: "/admin/pages/page_1/__actions/restore",
		CSRFToken:     "csrf-token",
		SaveStatus: BackendPageDetailActionStatus{
			Submitted: true,
			Message:   "Please correct the highlighted fields.",
			FieldErrors: map[string]string{
				"title": "Title is required.",
			},
		},
		ArchiveStatus:         BackendPageDetailActionStatus{Submitted: true, Message: "Archive failed."},
		RestoreStatus:         BackendPageDetailActionStatus{Submitted: true, Message: "Restore failed."},
		RestoreRevisionStatus: BackendPageDetailActionStatus{Submitted: true, Message: "Restore revision failed."},
		RevisionRestored:      true,
		Page: BackendPageDetailValues{
			ID:               "page_1",
			Title:            "About <Noni>",
			Slug:             "about-noni",
			BodyFormatBlocks: true,
			Body:             `[{"type":"paragraph","text":"Body"}]`,
			MetaTitle:        "About meta",
			MetaImageURL:     "/media/noni.jpg",
			MetaDescription:  "Meta description",
			MetaImageAlt:     "Meta alt",
			Published:        true,
			CanArchive:       true,
			CanRestore:       true,
			PreviewHref:      "/about-noni?preview=1",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-page-detail-renderer="gosx-studio">`,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="page-media-urls"><option value="/media/noni.jpg" label="noni.jpg" data-media-alt="Noni alt">noni.jpg</option></datalist>`,
		`<form class="panel admin-form" method="post" action="/admin/pages/page_1/__actions/save">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input type="hidden" name="id" value="page_1" />`,
		`<p class="form-status form-status--error">Please correct the highlighted fields.</p>`,
		`<p class="form-status form-status--error">Archive failed.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<p class="form-status form-status--error">Restore revision failed.</p>`,
		`<p class="form-status form-status--ok">Version restored.</p>`,
		`<div class="edit-preview edit-preview--text">preview</div>`,
		`<input id="title" name="title" value="About &lt;Noni&gt;" />`,
		`<p class="form-error">Title is required.</p>`,
		`<input id="slug" name="slug" value="about-noni" />`,
		`<select id="bodyFormat" name="bodyFormat" data-content-editor-format="true"><option value="blocks" selected>GoSX blocks</option><option value="mdpp">MDPP</option></select>`,
		`<div class="field-row field-row--wide content-editor" data-content-editor="true" data-content-editor-media-list="page-media-urls">`,
		`<textarea class="content-editor__source" id="body" name="body" rows="10" data-content-editor-source="true">[{&#34;type&#34;:&#34;paragraph&#34;,&#34;text&#34;:&#34;Body&#34;}]</textarea>`,
		`<button type="button" class="home-section-move" data-content-add="heading">Heading</button>`,
		`<button type="button" class="home-section-move" data-content-add="product">Product</button>`,
		`<button type="button" class="home-section-move" data-content-add="quote">Quote</button>`,
		`<div class="content-block-list" data-content-editor-list="true"></div>`,
		`<legend>SEO</legend>`,
		`<input id="metaTitle" name="metaTitle" value="About meta" />`,
		`<input id="metaImageUrl" name="metaImageUrl" list="page-media-urls" value="/media/noni.jpg" data-media-alt-target="metaImageAlt" />`,
		`<textarea id="metaDescription" name="metaDescription" rows="3">Meta description</textarea>`,
		`<input id="metaImageAlt" name="metaImageAlt" value="Meta alt" />`,
		`<input type="checkbox" name="published" checked /> Published`,
		`<button class="button button--primary" type="submit">Save page</button>`,
		`<a class="button button--secondary" href="/about-noni?preview=1" data-gosx-link="true">Preview</a>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/pages/page_1/__actions/archive" data-admin-confirm="Archive this page? It will be unpublished and hidden from public routes.">Archive</button>`,
		`<button class="button button--secondary" type="submit" formaction="/admin/pages/page_1/__actions/restore" data-admin-confirm="Restore this page?">Restore</button>`,
		`<a class="button button--secondary" href="/admin/pages" data-gosx-link="true">Back to pages</a>`,
		`<section class="panel" data-studio-revision-history="true">history</section>`,
		`<script src="/content-editor.js" defer></script><script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("page detail renderer missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`<section class="admin-heading">heading</section>`,
		`<datalist id="page-media-urls">`,
		`<form class="panel admin-form"`,
		`<div class="edit-preview edit-preview--text">preview</div>`,
		`<button class="button button--primary" type="submit">Save page</button>`,
		`<a class="button button--secondary" href="/about-noni?preview=1"`,
		`formaction="/admin/pages/page_1/__actions/archive"`,
		`formaction="/admin/pages/page_1/__actions/restore"`,
		`<a class="button button--secondary" href="/admin/pages"`,
		`<section class="panel" data-studio-revision-history="true">history</section>`,
		`<script src="/content-editor.js"`,
		`<script src="/media-picker.js"`,
	)
}

func TestRenderBackendPageDetailPageDefaultsAreEmptySafe(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendPageDetailPage(BackendPageDetailPageProps{}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-detail-renderer="gosx-studio" data-gosx-studio-backend-page-detail-renderer="gosx-studio">`,
		`<datalist id="page-media-urls"></datalist>`,
		`<form class="panel admin-form" method="post" action="">`,
		`<input type="hidden" name="csrf_token" value="" />`,
		`<input type="hidden" name="id" value="" />`,
		`<input id="title" name="title" value="" />`,
		`<p class="form-error"></p>`,
		`<input id="slug" name="slug" value="" />`,
		`<select id="bodyFormat" name="bodyFormat" data-content-editor-format="true"><option value="blocks">GoSX blocks</option><option value="mdpp">MDPP</option></select>`,
		`<textarea class="content-editor__source" id="body" name="body" rows="10" data-content-editor-source="true"></textarea>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="page-media-urls" value="" data-media-alt-target="metaImageAlt" />`,
		`<div class="check-row"><label><input type="checkbox" name="published" /> Published</label></div>`,
		`<button class="button button--primary" type="submit">Save page</button>`,
		`<a class="button button--secondary" href="" data-gosx-link="true">Preview</a>`,
		`<a class="button button--secondary" href="/admin/pages" data-gosx-link="true">Back to pages</a>`,
		`<script src="/content-editor.js" defer></script><script src="/media-picker.js" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty page detail renderer missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{
		`formaction=""`,
		`form-status`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty page detail renderer should not contain %q:\n%s", notWant, html)
		}
	}
}

func TestRenderBackendPageDetailPageSupportsDraftWorkflowAffordances(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendPageDetailPage(BackendPageDetailPageProps{
		SaveLabel:      "Save draft",
		PublishedLabel: "Mark page as ready",
		PublishedHelp:  "Saved page changes stay in draft until they are published from the Release Center.",
		CancelHref:     "/admin/pages",
		CancelLabel:    "Cancel",
		PublishHref:    "/admin/editor?mode=publish",
		PublishLabel:   "Publish changes",
		PublishNotice:  "Publishing happens from the Release Center.",
		Page: BackendPageDetailValues{
			ID:          "page_1",
			PreviewHref: "/about-noni?preview=1",
		},
	}))

	for _, fragment := range []string{
		`<div class="check-row"><label><input type="checkbox" name="published" /> Mark page as ready</label><small class="field-help">Saved page changes stay in draft until they are published from the Release Center.</small></div>`,
		`<button class="button button--primary" type="submit">Save draft</button>`,
		`<a class="button button--secondary" href="/about-noni?preview=1" data-gosx-link="true">Preview</a>`,
		`<a class="button button--secondary" href="/admin/pages" data-gosx-link="true">Cancel</a>`,
		`<a class="button button--primary" href="/admin/editor?mode=publish" data-gosx-link="true">Publish changes</a>`,
		`<small class="field-help">Publishing happens from the Release Center.</small>`,
		`<a class="button button--secondary" href="/admin/pages" data-gosx-link="true">Back to pages</a>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("draft workflow renderer missing %q:\n%s", fragment, html)
		}
	}

	for _, notWant := range []string{
		`<button class="button button--primary" type="submit">Save page</button>`,
		` /> Published</label>`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("draft workflow renderer should not contain %q:\n%s", notWant, html)
		}
	}

	assertOrder(t, html,
		`<button class="button button--primary" type="submit">Save draft</button>`,
		`<a class="button button--secondary" href="/about-noni?preview=1"`,
		`<a class="button button--secondary" href="/admin/pages" data-gosx-link="true">Cancel</a>`,
		`<a class="button button--primary" href="/admin/editor?mode=publish"`,
		`Publishing happens from the Release Center.`,
		`<a class="button button--secondary" href="/admin/pages" data-gosx-link="true">Back to pages</a>`,
	)
}
