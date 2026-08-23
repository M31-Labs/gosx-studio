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

// TestRenderPublishPanelChangeSetGrouped proves the Release Center prefers
// the richer, typed change-set (adversarial-review punch #4:
// cms/lifecycle/engine Engine.PendingDiff surfaced through
// cms/studio.PublishChangeSetView) over the legacy flat pendingChanges list
// once a host supplies one: changes group by scope in the fixed Content /
// Design / Layout / Components / Media / Interactions / Flows order, each
// row shows label + before -> after + actor + when, and the legacy flat
// "revision-diff-list" rendering (kindLabel/path only) is NOT also rendered.
func TestRenderPublishPanelChangeSetGrouped(t *testing.T) {
	view := publishPanelTestView()
	view["hasChangeSet"] = true
	view["hasChanges"] = true
	view["changeSummaryLabel"] = "2 changes ready to publish."
	view["changeGroups"] = []map[string]any{
		{
			"scopeKey":   "content",
			"scopeLabel": "Content",
			"count":      1,
			"changes": []map[string]any{
				{
					"label": "site.title", "kind": "changed", "kindLabel": "Changed",
					"hasBefore": true, "before": "Old title",
					"hasAfter": true, "after": "New title",
					"hasActor": true, "actorLabel": "Jane Doe",
					"hasWhen": true, "whenLabel": "Jul 10, 2026 3:04 PM", "whenMachine": "2026-07-10T15:04:00Z",
				},
			},
		},
		{
			"scopeKey":   "style",
			"scopeLabel": "Design",
			"count":      1,
			"changes": []map[string]any{
				{
					"label": "hero heading shadow", "kind": "added", "kindLabel": "Added",
					"hasBefore": false, "before": "",
					"hasAfter": true, "after": "0 1px 2px #000",
					"hasActor": false, "actorLabel": "",
					"hasWhen": false, "whenLabel": "", "whenMachine": "",
				},
			},
		},
	}

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	for _, fragment := range []string{
		`data-studio-publish-changeset="true"`,
		`data-studio-publish-changeset-has-changes="true"`,
		`<h3>What will change</h3><p data-studio-publish-changeset-empty="false">2 changes ready to publish.</p>`,
		`<div class="studio-publish-panel__changeset-groups" data-studio-publish-changeset-groups="true">`,
		`data-studio-publish-changeset-group="content"`,
		`data-studio-publish-changeset-group="style"`,
		`<h4>Design<span class="studio-publish-panel__changeset-count">1</span></h4>`,
		`data-studio-publish-changeset-row="true" data-studio-publish-changeset-kind="changed"`,
		`<code class="revision-diff-row__label">site.title</code>`,
		`<span class="revision-diff-row__value revision-diff-row__value--before">Old title</span>`,
		`<span class="revision-diff-row__value revision-diff-row__value--after">New title</span>`,
		`data-studio-publish-changeset-actor="true"`,
		`Jane Doe`,
		`datetime="2026-07-10T15:04:00Z" data-viewer-time="datetime"`,
		`data-studio-publish-changeset-kind="added"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("grouped change-set publish panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`class="revision-diff-list"><li>`, // legacy flat list must not also render
		`hero heading shadow" hidden`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("grouped change-set publish panel must not include legacy flat fragment %q:\n%s", notWant, html)
		}
	}
	// The added row (no prior value) must not fabricate a before value or a
	// visible arrow between before/after.
	addedRowStart := strings.Index(html, `data-studio-publish-changeset-kind="added"`)
	if addedRowStart < 0 {
		t.Fatalf("expected an added-kind row in html:\n%s", html)
	}
	addedRowHTML := html[addedRowStart:]
	if !strings.Contains(addedRowHTML[:400], `revision-diff-row__value--before" hidden`) {
		t.Fatalf("expected the added row's before span to be hidden:\n%s", addedRowHTML[:400])
	}
}

// TestRenderPublishPanelChangeSetEmptyState proves a host-supplied but empty
// change-set (Supplied=true, zero changes) renders the honest "nothing to
// publish" copy instead of an empty list or the legacy flat section.
func TestRenderPublishPanelChangeSetEmptyState(t *testing.T) {
	view := publishPanelTestView()
	view["hasChangeSet"] = true
	view["hasChanges"] = false
	view["changeSummaryLabel"] = "Nothing to publish — the site matches your draft."
	view["changeGroups"] = []map[string]any{}

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	for _, fragment := range []string{
		`data-studio-publish-changeset="true"`,
		`data-studio-publish-changeset-has-changes="false"`,
		`<p data-studio-publish-changeset-empty="true">Nothing to publish — the site matches your draft.</p>`,
		`<div class="studio-publish-panel__changeset-groups" data-studio-publish-changeset-groups="true" hidden></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty change-set publish panel missing %q:\n%s", fragment, html)
		}
	}
}

// TestRenderPublishPanelChangeSetAbsentWhenNotSupplied proves the section is
// OMITTED (not rendered blank/fake) when the host has not supplied a
// change-set at all -- graceful degradation to the pre-existing legacy flat
// list is covered by TestRenderPublishPanelFull, and full absence (neither
// legacy nor typed data) is covered by TestRenderPublishPanelMinimalOmitsOptionalSections.
func TestRenderPublishPanelChangeSetAbsentWhenNotSupplied(t *testing.T) {
	view := publishPanelTestView()
	view["hasPendingChanges"] = false
	// hasChangeSet intentionally left unset (host has not wired the source).

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	for _, notWant := range []string{
		`data-studio-publish-changeset`,
		`class="studio-publish-panel__pending"`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("publish panel must omit the change-set section entirely when unsupplied, found %q:\n%s", notWant, html)
		}
	}
}

// TestRenderPublishPanelActionsRenderAfterSummaryAndHeadline proves the
// decision-legibility reorder: the Publish/Discard actions block used to
// render before everything else in the panel (a bare button with no
// context above it). It must now render AFTER the ready/watch/next summary
// triple and the one-line decision headline, so a non-technical owner reads
// "why" before they reach the button.
func TestRenderPublishPanelActionsRenderAfterSummaryAndHeadline(t *testing.T) {
	view := publishPanelTestView()
	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	assertPublishOrder(t, html,
		`<div class="studio-publish-review__summary" aria-label="Publish review summary">`,
		`<p class="studio-publish-panel__decision-headline" data-studio-publish-decision-headline="true">`,
		`<div class="studio-publish-panel__actions" aria-label="Publish actions"`,
		`<div class="studio-publish-panel__checks" aria-label="Publish checks">`,
	)
}

// TestRenderPublishPanelDecisionHeadlineAllReady proves the one-line,
// plain-English headline reads "All N checks ready." when every check is
// Ready, and that Publish is never disabled even though the headline is
// upbeat (publishing always stays the owner's call).
func TestRenderPublishPanelDecisionHeadlineAllReady(t *testing.T) {
	view := publishPanelTestView()
	view["checks"] = []map[string]any{
		{"key": "media", "class": "studio-publish-review__card studio-publish-review__card--ready", "label": "Media alt text", "scope": "Media", "status": "ready", "statusLabel": "Ready", "summary": "All clear", "detail": "Every image has alt text."},
		{"key": "seo", "class": "studio-publish-review__card studio-publish-review__card--ready", "label": "SEO", "scope": "Search", "status": "ready", "statusLabel": "Ready", "summary": "All clear", "detail": "Meta description set."},
	}

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	if !strings.Contains(html, `<p class="studio-publish-panel__decision-headline" data-studio-publish-decision-headline="true">All 2 checks ready.</p>`) {
		t.Fatalf("publish panel missing all-ready decision headline:\n%s", html)
	}
	if strings.Contains(html, `data-studio-publish-attention`) {
		t.Fatalf("publish panel must not carry an attention marker when every check is ready:\n%s", html)
	}
	if strings.Contains(html, `data-studio-publish-attention-note`) {
		t.Fatalf("publish panel must not render an attention note when every check is ready:\n%s", html)
	}
	if !strings.Contains(html, `<button class="button button--primary" type="submit" form="websiteEditorForm" formaction="/publish" formmethod="post" formnovalidate="formnovalidate" data-admin-confirm="Publish this draft?" data-studio-submit-action="publish" data-studio-field-action-formaction="/publish">Publish</button>`) {
		t.Fatalf("publish button must render exactly, with no disabled attribute:\n%s", html)
	}
}

// TestRenderPublishPanelDecisionHeadlineNeedsAttention proves the headline
// reads "M of N checks need attention." (or, singular, "needs attention")
// when one or more checks aren't Ready, that the actions container and a
// visible note both carry the same count, and that Publish is still not
// disabled -- the attention note is information, not a gate.
func TestRenderPublishPanelDecisionHeadlineNeedsAttention(t *testing.T) {
	view := publishPanelTestView()
	view["checks"] = []map[string]any{
		{"key": "media", "class": "studio-publish-review__card studio-publish-review__card--watch", "label": "Media alt text", "scope": "Media", "status": "watch", "statusLabel": "Watch", "summary": "Needs review", "detail": "One image needs alt text."},
		{"key": "seo", "class": "studio-publish-review__card studio-publish-review__card--ready", "label": "SEO", "scope": "Search", "status": "ready", "statusLabel": "Ready", "summary": "All clear", "detail": "Meta description set."},
	}

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	for _, fragment := range []string{
		`<p class="studio-publish-panel__decision-headline" data-studio-publish-decision-headline="true">1 of 2 checks need attention.</p>`,
		`data-studio-publish-attention="1"`,
		`<p class="studio-publish-panel__attention-note" data-studio-publish-attention-note="true">1 item needs attention — you can still publish.</p>`,
		`data-studio-submit-action="publish"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("publish panel missing %q:\n%s", fragment, html)
		}
	}
	if !strings.Contains(html, `<button class="button button--primary" type="submit" form="websiteEditorForm" formaction="/publish" formmethod="post" formnovalidate="formnovalidate" data-admin-confirm="Publish this draft?" data-studio-submit-action="publish" data-studio-field-action-formaction="/publish">Publish</button>`) {
		t.Fatalf("Publish must never be disabled by attention checks:\n%s", html)
	}

	// The attention note must render inside the actions container, after the
	// Publish button, so an owner reads the button first and the caveat
	// immediately beside it.
	assertPublishOrder(t, html,
		`data-studio-submit-action="publish"`,
		`data-studio-publish-attention-note="true"`,
	)
}

// TestRenderPublishPanelChecksGroupAttentionFirst proves
// renderPublishPanelChecks groups status-first: every non-Ready check
// renders under a "Needs attention" heading BEFORE a "Ready" heading, so a
// single red/yellow check doesn't get lost among a wall of green cards in
// plain source order. Order within each group is stable.
func TestRenderPublishPanelChecksGroupAttentionFirst(t *testing.T) {
	view := publishPanelTestView()
	view["checks"] = []map[string]any{
		{"key": "seo", "class": "studio-publish-review__card studio-publish-review__card--ready", "label": "SEO", "scope": "Search", "status": "ready", "statusLabel": "Ready", "summary": "All clear", "detail": "Meta description set."},
		{"key": "media", "class": "studio-publish-review__card studio-publish-review__card--watch", "label": "Media alt text", "scope": "Media", "status": "watch", "statusLabel": "Watch", "summary": "Needs review", "detail": "One image needs alt text."},
		{"key": "links", "class": "studio-publish-review__card studio-publish-review__card--next", "label": "Broken links", "scope": "Content", "status": "next", "statusLabel": "Next", "summary": "Fix before publish", "detail": "One link 404s."},
	}

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	for _, fragment := range []string{
		`<div class="studio-publish-panel__checks-group studio-publish-panel__checks-group--attention" data-studio-publish-checks-group="attention"><h3 class="studio-publish-panel__checks-heading">Needs attention</h3>`,
		`<div class="studio-publish-panel__checks-group studio-publish-panel__checks-group--ready" data-studio-publish-checks-group="ready"><h3 class="studio-publish-panel__checks-heading">Ready</h3>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("publish panel missing %q:\n%s", fragment, html)
		}
	}

	// Attention-first: the "Needs attention" group (and both non-ready
	// checks within it, in their original source order) must render before
	// the "Ready" group.
	assertPublishOrder(t, html,
		`data-studio-publish-checks-group="attention"`,
		`data-studio-publish-check="media"`,
		`data-studio-publish-check="links"`,
		`data-studio-publish-checks-group="ready"`,
		`data-studio-publish-check="seo"`,
	)
}

// TestRenderPublishPanelChecksAllReadyOmitsGroupHeadings proves the happy
// path -- every check Ready -- renders with no "Needs attention"/"Ready"
// group headings at all (no noise for the common case): the check cards
// render directly, matching the pre-grouping markup.
func TestRenderPublishPanelChecksAllReadyOmitsGroupHeadings(t *testing.T) {
	view := publishPanelTestView()
	view["checks"] = []map[string]any{
		{"key": "media", "class": "studio-publish-review__card studio-publish-review__card--ready", "label": "Media alt text", "scope": "Media", "status": "ready", "statusLabel": "Ready", "summary": "All clear", "detail": "Every image has alt text."},
	}

	html := gosx.RenderHTML(RenderPublishPanel(view, PublishPanelOptions{}))

	for _, notWant := range []string{
		"studio-publish-panel__checks-group",
		"studio-publish-panel__checks-heading",
		"data-studio-publish-checks-group",
		"Needs attention",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("all-ready publish panel must not render group headings, found %q:\n%s", notWant, html)
		}
	}
	if !strings.Contains(html, `<div class="studio-publish-panel__checks" aria-label="Publish checks"><article class="studio-publish-review__card studio-publish-review__card--ready" data-studio-publish-check="media">`) {
		t.Fatalf("all-ready publish panel must render the check card directly under the checks wrapper:\n%s", html)
	}
}

// assertPublishOrder fails the test unless every fragment appears in html,
// each strictly after the previous one (matching shell.assertOrder's
// contract, duplicated here since panels has no shared test helper
// package).
func assertPublishOrder(t *testing.T, html string, fragments ...string) {
	t.Helper()
	last := -1
	for _, fragment := range fragments {
		next := strings.Index(html, fragment)
		if next < 0 {
			t.Fatalf("missing ordered fragment %q:\n%s", fragment, html)
		}
		if next <= last {
			t.Fatalf("fragment %q rendered out of order:\n%s", fragment, html)
		}
		last = next
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
