package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderRevisionHistoryPanelPopulated(t *testing.T) {
	view := map[string]any{
		"class":       "panel studio-revision-history",
		"panelKey":    "versions",
		"headerClass": "panel__header",
		"title":       "Version history",
		"empty":       "No previous versions yet.",
		"hasItems":    true,
		"items": []map[string]any{
			{
				"key":                "rev_1",
				"title":              "Saved draft",
				"actionLabel":        "Saved",
				"createdMachine":     "2026-05-24T12:00:00Z",
				"createdLabel":       "May 24, 2026",
				"summary":            "Homepage copy updated.",
				"hasSummary":         true,
				"changeSummary":      "2 changes",
				"hasDiff":            true,
				"restoreAction":      "/admin/editor/__actions/restoreRevision",
				"confirm":            "Restore these editor settings? Current look and feel will be saved in history first.",
				"buttonLabel":        "Restore this version",
				"revisionInputName":  "revisionId",
				"revisionInputValue": "rev_1",
				"formID":             "websiteEditorForm",
				"changeItems": []map[string]any{
					{"kindLabel": "Changed", "path": "appearance.palette"},
					{"kindLabel": "Added", "path": "home.sections.hero"},
				},
			},
			{
				"key":                "rev_2",
				"title":              "Initial draft",
				"actionLabel":        "Created",
				"createdMachine":     "2026-05-23T12:00:00Z",
				"createdLabel":       "May 23, 2026",
				"hasSummary":         false,
				"hasDiff":            false,
				"restoreAction":      "/admin/editor/__actions/restoreRevision",
				"confirm":            "Restore these editor settings? Current look and feel will be saved in history first.",
				"buttonLabel":        "Restore this version",
				"revisionInputName":  "revisionId",
				"revisionInputValue": "rev_2",
				"formID":             "websiteEditorForm",
			},
		},
	}

	html := gosx.RenderHTML(RenderRevisionHistoryPanel(view, RevisionHistoryPanelOptions{}))

	for _, fragment := range []string{
		`<section class="panel studio-revision-history" data-panel-key="versions" data-studio-revision-history="true" data-gosx-studio-revision-history-renderer="gosx-studio">`,
		`<div class="panel__header"><h2>Version history</h2></div>`,
		`<p class="empty" hidden>No previous versions yet.</p>`,
		`<ul class="field-list field-list--stacked">`,
		`<li data-studio-revision="rev_1">`,
		`<strong>Saved draft</strong>`,
		`<span>Saved</span>`,
		`<time datetime="2026-05-24T12:00:00Z" data-viewer-time="datetime">May 24, 2026</time>`,
		`<p>Homepage copy updated.</p>`,
		`<p class="revision-diff-summary">2 changes</p>`,
		`<ul class="revision-diff-list"><li><span>Changed</span><code>appearance.palette</code></li><li><span>Added</span><code>home.sections.hero</code></li></ul>`,
		`<button class="button button--secondary" type="submit" form="websiteEditorForm" formaction="/admin/editor/__actions/restoreRevision" formmethod="post" formnovalidate="formnovalidate" name="revisionId" value="rev_1" data-admin-confirm="Restore these editor settings? Current look and feel will be saved in history first." data-studio-submit-action="restoreRevision" data-studio-field-action-formaction="/admin/editor/__actions/restoreRevision">Restore this version</button>`,
		`<li data-studio-revision="rev_2">`,
		`<p hidden></p><p class="revision-diff-summary" hidden></p><ul class="revision-diff-list" hidden></ul>`,
		`value="rev_2"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered revision history missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		"<form",
		"csrf_token",
		`<ul class="field-list field-list--stacked" hidden>`,
		// Regression guard for #20: the owner-reported staging bug rendered
		// "No previous versions yet." simultaneously with a populated,
		// working revision list. The empty-state <p> must ALWAYS carry
		// `hidden` when hasItems is true -- this exact (non-hidden) fragment
		// must never appear in populated output.
		`<p class="empty">No previous versions yet.</p>`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered revision history must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderRevisionHistoryPanelEmpty(t *testing.T) {
	view := map[string]any{
		"class":       "panel studio-revision-history",
		"panelKey":    "versions",
		"headerClass": "panel__header",
		"title":       "Version history",
		"empty":       "No previous versions yet.",
	}

	html := gosx.RenderHTML(RenderRevisionHistoryPanel(view, RevisionHistoryPanelOptions{}))

	for _, fragment := range []string{
		`data-studio-revision-history="true"`,
		`data-gosx-studio-revision-history-renderer="gosx-studio"`,
		`<p class="empty">No previous versions yet.</p>`,
		`<ul class="field-list field-list--stacked" hidden></ul>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty revision history missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`data-studio-revision=`,
		"<form",
		"csrf_token",
		`data-studio-submit-action="restoreRevision"`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty revision history must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderRevisionHistoryPanelStandaloneForms(t *testing.T) {
	view := map[string]any{
		"class":       "panel",
		"headerClass": "panel__header",
		"title":       "Version history",
		"empty":       "No previous versions yet.",
		"hasItems":    true,
		"csrfToken":   "csrf-token",
		"hiddenInputs": []map[string]string{
			{"name": "id", "value": "page_1"},
		},
		"items": []map[string]any{{
			"key":            "rev_1",
			"title":          "Saved draft",
			"actionLabel":    "Saved",
			"createdMachine": "2026-05-24T12:00:00Z",
			"createdLabel":   "May 24, 2026",
			"summary":        "Homepage copy updated.",
			"hasSummary":     true,
			"restoreAction":  "/admin/pages/page_1/__actions/restoreRevision",
			"confirm":        "Restore this page version? The current page will be saved in history first.",
			"buttonLabel":    "Restore this version",
			"hiddenInputs": []map[string]string{
				{"name": "revisionId", "value": "rev_1"},
			},
		}},
	}

	html := gosx.RenderHTML(RenderRevisionHistoryPanel(view, RevisionHistoryPanelOptions{StandaloneForms: true}))

	for _, fragment := range []string{
		`<section class="panel" data-panel-key="" data-studio-revision-history="true" data-gosx-studio-revision-history-renderer="gosx-studio">`,
		`<ul class="field-list field-list--stacked">`,
		`<li data-studio-revision="rev_1">`,
		`<form class="inline-form" method="post" action="/admin/pages/page_1/__actions/restoreRevision">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<input type="hidden" name="id" value="page_1" />`,
		`<input type="hidden" name="revisionId" value="rev_1" />`,
		`<button class="button button--secondary" type="submit" data-admin-confirm="Restore this page version? The current page will be saved in history first.">Restore this version</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("standalone revision history missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`form="websiteEditorForm"`,
		`formaction=`,
		`formmethod=`,
		`data-studio-submit-action="restoreRevision"`,
		// Regression guard for #20 (see TestRenderRevisionHistoryPanelPopulated):
		// the empty-state gating must hold for the StandaloneForms path too.
		`<p class="empty">No previous versions yet.</p>`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("standalone revision history must not include %q:\n%s", notWant, html)
		}
	}
}
