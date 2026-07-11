package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderPublishPanelFull(t *testing.T) {
	view := publishPanelTestView()

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{
		PreviewShareNode:    gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-preview-share", "true")), gosx.Text("Preview node")),
		ActivityPanelNode:   gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-activity-drawer", "true")), gosx.Text("Activity node")),
		RevisionHistoryNode: gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-revision-history", "true")), gosx.Text("History node")),
	}))

	for _, fragment := range []string{
		`<section class="editor-panel editor-panel--publish-studio studio-publish-panel" data-studio-publish-panel="true" data-studio-mode-panel="publish" data-studio-panel="publish" data-studio-engine-source="gosx" data-studio-publish-status="draft" data-gosx-studio-publish-panel-renderer="gosx-studio">`,
		`<header class="studio-publish-panel__head"><div><p class="kicker">Publish</p><h2>Release center</h2><p>Ready for owner review.</p></div><output>2/3 clear</output></header>`,
		`<p class="studio-publish-panel__draft-status" data-studio-publish-draft-status="true" data-studio-has-draft="true">Unpublished changes ready</p>`,
		`<a class="button button--secondary" href="/preview" data-gosx-link="true">Open preview</a>`,
		`<button class="button button--primary" type="submit" form="websiteEditorForm" formaction="/publish" formmethod="post" formnovalidate="formnovalidate" data-admin-confirm="Publish this draft?" data-studio-submit-action="publish" data-studio-field-action-formaction="/publish">Publish</button>`,
		`<button class="button button--ghost" type="submit" form="websiteEditorForm" formaction="/discard" formmethod="post" formnovalidate="formnovalidate" data-admin-confirm="Discard unpublished changes?" data-studio-submit-action="discard" data-studio-field-action-formaction="/discard">Discard changes</button>`,
		`<section class="studio-publish-panel__pending" aria-label="Unpublished changes"><header><h3>What will change</h3><p>2 unpublished changes</p></header><ul class="revision-diff-list"><li><span>Changed</span><code>site.title</code></li><li><span>Added</span><code>home.sections.hero</code></li></ul></section>`,
		`<section class="studio-publish-panel__preview-share" data-studio-publish-preview-share="true">`,
		`<div class="studio-publish-panel__preview-share-slot" data-studio-publish-preview-share-slot="true"><section data-studio-preview-share="true">Preview node</section></div>`,
		`<section class="studio-publish-panel__environments" aria-label="Publishing environments" data-studio-publish-environments="true">`,
		`<article class="studio-publish-environment studio-publish-environment--ready" data-studio-publish-environment="staging" data-studio-publish-environment-state="ready"><div><strong>Staging</strong><output>Ready</output></div><a class="studio-publish-environment__value" href="https://staging.example.test" target="_blank" rel="noreferrer" data-gosx-link="true" data-studio-publish-environment-value="true">https://staging.example.test</a><p>Review the release candidate.</p></article>`,
		`<article class="studio-publish-environment studio-publish-environment--tbd" data-studio-publish-environment="production" data-studio-publish-environment-state="tbd"><div><strong>Production</strong><output>TBD</output></div><span class="studio-publish-environment__value" data-studio-publish-environment-value="true">TBD</span><p>Production domain is not ready yet.</p></article>`,
		`<input id="publishAt" name="lifecyclePublishAt" type="datetime-local" value="2026-06-29T10:30" form="websiteEditorForm" data-studio-field-source="lifecycle.schedule.publishAt" data-studio-field-editable="lifecycle" />`,
		`<button class="button button--secondary" type="submit" form="websiteEditorForm" formaction="/schedule" formmethod="post" formnovalidate="formnovalidate" data-admin-confirm="Schedule this draft?" data-studio-submit-action="schedule" data-studio-field-action-formaction="/schedule">Schedule</button>`,
		`<div class="studio-publish-panel__decision" aria-label="Release decision"><article class="studio-publish-review__decision studio-publish-review__decision--next">`,
		`<a href="#approval" data-gosx-link="true">Request approval</a>`,
		`<article class="studio-publish-review__decision studio-publish-review__decision--ready"><header class="studio-publish-review__decision-head"><strong>Publish timing</strong><output>Ready</output></header><p class="studio-publish-review__decision-summary">Manual publish</p><p class="studio-publish-review__detail">No future publish time is set.</p></article>`,
		`<div class="studio-publish-review__summary" aria-label="Publish review summary"><span><strong>2</strong>clear</span><span><strong>1</strong>watch</span><span><strong>0</strong>next</span></div>`,
		`<div class="studio-publish-panel__checks" aria-label="Publish checks">`,
		`<article class="studio-publish-review__card studio-publish-review__card--watch" data-studio-publish-check="media"><header class="studio-publish-review__card-head"><div><strong>Media alt text</strong><span>Media</span></div><output>Watch</output></header><p class="studio-publish-review__check-summary">Needs review</p><p class="studio-publish-review__detail">One image needs alt text.</p><a href="/admin/media" data-gosx-link="true">Review media</a></article>`,
		`<section class="studio-publish-panel__impacts" aria-label="Release impact"><h3>Release impact</h3><div class="studio-publish-review__impact-list"><article class="studio-publish-review__impact studio-publish-review__impact--ready" data-studio-publish-impact="sections"><div><strong>Home sections</strong><span>Homepage</span></div><output>5 enabled</output><p>Homepage structure changes.</p></article></div></section>`,
		`<section class="studio-publish-panel__collaboration" data-studio-publish-activity="true">`,
		`<div class="studio-publish-panel__activity-slot" data-studio-publish-activity-slot="true"><section data-studio-activity-drawer="true">Activity node</section></div>`,
		`<section class="studio-publish-panel__history" data-studio-publish-history="true">`,
		`<div class="studio-publish-panel__history-slot" data-studio-publish-history-slot="true"><section data-studio-revision-history="true">History node</section></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered publish panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		"<form",
		"csrf_token",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered publish panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderPublishPanelMinimalOmitsOptionalSections(t *testing.T) {
	view := publishPanelTestView()
	view["status"] = "clean"
	view["hasDraft"] = false
	view["hasPublishAction"] = false
	view["hasDiscardAction"] = false
	view["hasScheduleAction"] = false
	view["hasPendingChanges"] = false
	view["hasApproval"] = false
	view["hasSchedule"] = false
	view["hasChecks"] = false
	view["hasImpacts"] = false
	view["hasEnvironments"] = false
	view["checks"] = nil
	view["impacts"] = nil
	view["environments"] = nil

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	for _, fragment := range []string{
		`data-gosx-studio-publish-panel-renderer="gosx-studio"`,
		`data-studio-publish-status="clean"`,
		`<p class="studio-publish-panel__draft-status" data-studio-publish-draft-status="true" data-studio-has-draft="false" hidden>Unpublished changes ready</p>`,
		`<div class="studio-publish-panel__actions" aria-label="Publish actions"><a class="button button--secondary" href="/preview" data-gosx-link="true">Open preview</a></div>`,
		`<div class="studio-publish-panel__decision" aria-label="Release decision"></div>`,
		`<div class="studio-publish-panel__preview-share-slot" data-studio-publish-preview-share-slot="true"></div>`,
		`<div class="studio-publish-panel__activity-slot" data-studio-publish-activity-slot="true"></div>`,
		`<div class="studio-publish-panel__history-slot" data-studio-publish-history-slot="true"></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("minimal publish panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`data-studio-submit-action="publish"`,
		`data-studio-submit-action="discard"`,
		`data-studio-submit-action="schedule"`,
		`class="studio-publish-panel__pending"`,
		`data-studio-publish-environments="true"`,
		`data-studio-publish-environment=`,
		`data-studio-publish-check=`,
		`data-studio-publish-impact=`,
		"<form",
		"csrf_token",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("minimal publish panel must not include %q:\n%s", notWant, html)
		}
	}
}

func publishPanelTestView() map[string]any {
	return map[string]any{
		"status":             "draft",
		"kicker":             "Publish",
		"panelTitle":         "Release center",
		"summary":            "Ready for owner review.",
		"countLabel":         "2/3 clear",
		"hasDraft":           true,
		"draftStatusLabel":   "Unpublished changes ready",
		"previewHref":        "/preview",
		"formID":             "websiteEditorForm",
		"publishAction":      "/publish",
		"hasPublishAction":   true,
		"discardAction":      "/discard",
		"hasDiscardAction":   true,
		"hasPendingChanges":  true,
		"draftSummary":       "2 unpublished changes",
		"scheduleInputID":    "publishAt",
		"scheduleInputName":  "lifecyclePublishAt",
		"scheduleInputValue": "2026-06-29T10:30",
		"scheduleHelp":       "Leave blank to publish manually.",
		"scheduleAction":     "/schedule",
		"hasScheduleAction":  true,
		"hasApproval":        true,
		"hasSchedule":        true,
		"readyCountLabel":    "2",
		"watchCountLabel":    "1",
		"nextCountLabel":     "0",
		"hasChecks":          true,
		"hasImpacts":         true,
		"hasEnvironments":    true,
		"pendingChanges":     []map[string]any{{"kindLabel": "Changed", "path": "site.title"}, {"kindLabel": "Added", "path": "home.sections.hero"}},
		"approval":           publishPanelDecisionTestView("studio-publish-review__decision studio-publish-review__decision--next", "Owner approval", "Approval pending", "Noni should review the preview.", "Next", true),
		"schedule":           publishPanelDecisionTestView("studio-publish-review__decision studio-publish-review__decision--ready", "Publish timing", "Manual publish", "No future publish time is set.", "Ready", false),
		"checks":             []map[string]any{{"key": "media", "class": "studio-publish-review__card studio-publish-review__card--watch", "label": "Media alt text", "scope": "Media", "statusLabel": "Watch", "summary": "Needs review", "detail": "One image needs alt text.", "hasHref": true, "href": "/admin/media", "actionLabel": "Review media"}},
		"impacts":            []map[string]any{{"key": "sections", "class": "studio-publish-review__impact studio-publish-review__impact--ready", "label": "Home sections", "scope": "Homepage", "value": "5 enabled", "detail": "Homepage structure changes."}},
		"environments":       []map[string]any{{"key": "staging", "label": "Staging", "url": "https://staging.example.test", "state": "ready", "stateLabel": "Ready", "hasHref": true, "detail": "Review the release candidate."}, {"key": "production", "label": "Production", "value": "TBD", "state": "tbd", "stateLabel": "TBD", "detail": "Production domain is not ready yet."}},
	}
}

func publishPanelDecisionTestView(className, label, summary, detail, statusLabel string, href bool) map[string]any {
	return map[string]any{
		"class":       className,
		"label":       label,
		"summary":     summary,
		"detail":      detail,
		"statusLabel": statusLabel,
		"hasHref":     href,
		"href":        "#approval",
		"actionLabel": "Request approval",
	}
}
