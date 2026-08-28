package studio

import (
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx"
)

func TestPublishReviewView(t *testing.T) {
	review := PublishReview{
		Key:          " homepage ",
		ResourceKind: "Page",
		ResourceID:   "home",
		Title:        "Homepage publish",
		Approval: PublishApproval{
			Required: true,
			Approved: true,
			Reviewer: "Owner",
		},
		Schedule: PublishSchedule{
			Enabled:   true,
			PublishAt: time.Date(2026, 7, 1, 16, 30, 0, 0, time.UTC),
			Timezone:  "America/Los_Angeles",
		},
		Checks: []PublishCheck{
			NewPublishCheck("approval", "Owner approval", "Governance", ReadinessReady, "Approved", "The owner approved the release."),
			NewPublishCheck("forms", "Flow handlers", "Forms", ReadinessWatch, "1 flow needs review", "Confirm lead capture before release.").WithHref("/admin/flows"),
			{Key: "skip"},
		},
		Impacts: []PublishImpact{
			NewPublishImpact("copy", "Copy changes", "Homepage", "3 fields", "Hero and guarantee copy changed.", ReadinessReady),
		},
		PrimaryHref: "/admin/publish",
	}
	view := PublishReviewView(review)
	if view["summary"] != "1/2 clear" || view["readyCount"] != 1 || view["watchCount"] != 1 || view["nextCount"] != 0 || view["total"] != 2 {
		t.Fatalf("unexpected view counts: %#v", view)
	}
	if view["status"] != "watch" || view["primaryActionLabel"] != "Open" || view["hasPrimaryHref"] != true {
		t.Fatalf("unexpected review view: %#v", view)
	}
	if view["hasApproval"] != true || view["hasSchedule"] != true {
		t.Fatalf("expected approval and schedule views: %#v", view)
	}
	approval := view["approval"].(map[string]any)
	if approval["approved"] != true || approval["summary"] != "Owner" {
		t.Fatalf("unexpected approval view: %#v", approval)
	}
	schedule := view["schedule"].(map[string]any)
	if schedule["summary"] != "Jul 1, 2026 9:30 AM PDT" || schedule["publishAt"] != "2026-07-01T16:30:00Z" {
		t.Fatalf("unexpected schedule view: %#v", schedule)
	}
	checks := view["checks"].([]map[string]any)
	if checks[1]["key"] != "forms" || checks[1]["statusLabel"] != "Watch" || checks[1]["hasHref"] != true {
		t.Fatalf("unexpected check view: %#v", checks[1])
	}
	impacts := view["impacts"].([]map[string]any)
	if impacts[0]["key"] != "copy" || impacts[0]["value"] != "3 fields" {
		t.Fatalf("unexpected impact view: %#v", impacts[0])
	}
}

func TestRenderPublishReviewPanel(t *testing.T) {
	review := PublishReview{
		Key:          "homepage",
		ResourceKind: "Page",
		Title:        "Homepage publish",
		Approval: PublishApproval{
			Required: true,
			Summary:  "Needs approval",
			Href:     "/admin/review",
		},
		Schedule: PublishSchedule{
			Enabled: true,
			Summary: "Publish time required",
		},
		Checks: []PublishCheck{
			NewPublishCheck("copy", "Required copy", "Content", ReadinessReady, "Ready to publish", "Title, tagline, and hero fields are filled."),
			NewPublishCheck("approval", "Owner approval", "Governance", ReadinessNext, "Needs approval", "Collect approval before the public release."),
		},
		Impacts: []PublishImpact{
			NewPublishImpact("navigation", "Navigation", "Site shell", "Updated", "Header and footer links remain stable.", ReadinessReady),
		},
		PrimaryHref:        "/admin/publish",
		PrimaryActionLabel: "Open release",
	}
	html := gosx.RenderHTML(RenderPublishReviewPanel(review, PublishReviewOptions{Class: "studio-publish-review"}))
	for _, want := range []string{
		`data-studio-publish-review="homepage"`,
		`data-studio-publish-status="next"`,
		`class="studio-publish-review"`,
		`1/2 clear`,
		`data-studio-publish-approval="true"`,
		`data-studio-publish-schedule="true"`,
		`data-studio-publish-check="copy"`,
		`Required copy`,
		`Owner approval`,
		`Needs approval`,
		`data-studio-publish-impact="navigation"`,
		`Open release`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in publish review html: %s", want, html)
		}
	}
}

func TestRenderPublishReviewPanelEmpty(t *testing.T) {
	html := gosx.RenderHTML(RenderPublishReviewPanel(PublishReview{}, PublishReviewOptions{EmptyTitle: "No release checks"}))
	for _, want := range []string{
		`data-studio-publish-review="publish-review"`,
		`No release checks`,
		`Register content, approval, flow, and deployment checks`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in empty publish review html: %s", want, html)
		}
	}
}

func TestNormalizePublishReviewDefaults(t *testing.T) {
	review := NormalizePublishReview(PublishReview{
		Approval: PublishApproval{Required: true},
		Schedule: PublishSchedule{Enabled: true},
		Checks: []PublishCheck{
			{Label: "  Approval  ", Scope: " ", Summary: " ", Status: "unknown"},
			{Key: "skip"},
		},
		Impacts: []PublishImpact{
			{Label: "  Routes  ", Scope: " ", Value: " ", Status: "unknown"},
			{Key: "skip"},
		},
	})
	if review.Key != "publish-review" || review.ResourceKind != "Site" || review.Title != "Publish review" {
		t.Fatalf("unexpected review defaults: %#v", review)
	}
	if len(review.Checks) != 1 || review.Checks[0].Key != "approval" || review.Checks[0].Summary != "Watch" || review.Checks[0].ActionLabel != "Open" {
		t.Fatalf("unexpected checks: %#v", review.Checks)
	}
	if len(review.Impacts) != 1 || review.Impacts[0].Key != "routes" || review.Impacts[0].Value != "Tracked" {
		t.Fatalf("unexpected impacts: %#v", review.Impacts)
	}
	if review.Approval.Status != ReadinessNext || review.Approval.Summary != "Approval required" {
		t.Fatalf("unexpected approval defaults: %#v", review.Approval)
	}
	if review.Schedule.Status != ReadinessNext || review.Schedule.Summary != "Publish time required" {
		t.Fatalf("unexpected schedule defaults: %#v", review.Schedule)
	}
}

// TestPublishChangeSetViewNotSupplied covers the "host hasn't wired
// Engine.PendingDiff yet" case: Supplied=false must produce ONLY
// hasChangeSet=false so panels.RenderPublishPanel omits the section rather
// than rendering a fake/empty one.
func TestPublishChangeSetViewNotSupplied(t *testing.T) {
	view := PublishChangeSetView(PublishChangeSet{}, nil)
	if len(view) != 1 || view["hasChangeSet"] != false {
		t.Fatalf("expected only hasChangeSet=false for an unsupplied change-set, got %#v", view)
	}
}

// TestPublishChangeSetViewEmptySupplied covers the honest "nothing to
// publish" state: the host DID ask (Supplied=true) and the answer was zero
// changes, which is different from "did not ask" and must render.
func TestPublishChangeSetViewEmptySupplied(t *testing.T) {
	view := PublishChangeSetView(PublishChangeSet{Supplied: true}, nil)
	if view["hasChangeSet"] != true || view["hasChanges"] != false || view["changeCount"] != 0 {
		t.Fatalf("unexpected empty-supplied view: %#v", view)
	}
	if view["changeSummaryLabel"] != "Nothing to publish — the site matches your draft." {
		t.Fatalf("unexpected empty-state summary: %#v", view["changeSummaryLabel"])
	}
	groups, ok := view["changeGroups"].([]map[string]any)
	if !ok || len(groups) != 0 {
		t.Fatalf("expected zero groups for an empty change-set, got %#v", view["changeGroups"])
	}
}

// TestPublishChangeSetViewGroupsByScopeInCanonicalOrder proves the grouped
// section reads Content, Design, Layout, Components, Media, Interactions,
// Flows in that fixed order regardless of the input slice's order, skips any
// scope with zero changes, and buckets an unrecognized scope into a trailing
// "Other" group instead of dropping it.
func TestPublishChangeSetViewGroupsByScopeInCanonicalOrder(t *testing.T) {
	when := time.Date(2026, 7, 10, 15, 4, 0, 0, time.UTC)
	view := PublishChangeSetView(PublishChangeSet{
		Supplied: true,
		Changes: []PublishChange{
			{Scope: PublishChangeScopeFlows, Label: "flows.checkout.route", Before: "/old", BeforeSet: true, After: "/new", AfterSet: true, ActorLabel: "Jane Doe", When: when},
			{Scope: PublishChangeScopeContent, Label: "site.title", Before: "Old title", BeforeSet: true, After: "New title", AfterSet: true, ActorLabel: "Jane Doe", When: when},
			{Scope: PublishChangeScope("weird"), Label: "unrecognized.field", After: "value", AfterSet: true},
		},
	}, nil)
	if view["hasChangeSet"] != true || view["hasChanges"] != true || view["changeCount"] != 3 {
		t.Fatalf("unexpected populated view: %#v", view)
	}
	groups, ok := view["changeGroups"].([]map[string]any)
	if !ok || len(groups) != 3 {
		t.Fatalf("expected 3 groups (content, flows, other), got %#v", view["changeGroups"])
	}
	if groups[0]["scopeKey"] != "content" || groups[0]["scopeLabel"] != "Content" || groups[0]["count"] != 1 {
		t.Fatalf("expected Content group first (canonical order before Flows), got %#v", groups[0])
	}
	if groups[1]["scopeKey"] != "flows" || groups[1]["scopeLabel"] != "Flows" {
		t.Fatalf("expected Flows group second, got %#v", groups[1])
	}
	if groups[2]["scopeKey"] != "other" || groups[2]["scopeLabel"] != "Other" {
		t.Fatalf("expected an unrecognized scope to land in a trailing Other group, not be dropped, got %#v", groups[2])
	}
}

// TestPublishChangeRowViewHonorsBeforeAfterSetHonesty proves a row never
// fabricates a before/after value when the underlying engine.DraftChange
// left BeforeSet/AfterSet false (a field with no prior value, or one this
// edit cleared) -- mirroring engine.OperationChangeSet's own documented
// honesty discipline -- and that kind classification (added/removed/changed)
// follows BeforeSet/AfterSet, with actor/when rendered only when supplied.
func TestPublishChangeRowViewHonorsBeforeAfterSetHonesty(t *testing.T) {
	when := time.Date(2026, 7, 10, 15, 4, 0, 0, time.UTC)
	view := PublishChangeSetView(PublishChangeSet{
		Supplied: true,
		Changes: []PublishChange{
			// changed: both sides set.
			{Scope: PublishChangeScopeStyle, Label: "hero heading color", Before: "#111", BeforeSet: true, After: "#fff", AfterSet: true, ActorLabel: "Jane Doe", When: when},
			// added: no prior value.
			{Scope: PublishChangeScopeStyle, Label: "hero heading shadow", After: "0 1px 2px #000", AfterSet: true},
			// removed/cleared: no next value -- Before must NOT be faked as "" masquerading as a real empty string; After must be absent, not "".
			{Scope: PublishChangeScopeStyle, Label: "hero heading outline", Before: "1px solid red", BeforeSet: true},
			// no actor/when supplied at all.
		},
	}, time.UTC)
	groups := view["changeGroups"].([]map[string]any)
	if len(groups) != 1 || groups[0]["scopeKey"] != "style" || groups[0]["scopeLabel"] != "Design" {
		t.Fatalf("expected one Design group, got %#v", groups)
	}
	rows := groups[0]["changes"].([]map[string]any)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d: %#v", len(rows), rows)
	}

	changed := rows[0]
	if changed["kind"] != "changed" || changed["kindLabel"] != "Changed" {
		t.Fatalf("expected a both-sides-set row to classify as changed, got %#v", changed)
	}
	if changed["hasBefore"] != true || changed["before"] != "#111" || changed["hasAfter"] != true || changed["after"] != "#fff" {
		t.Fatalf("unexpected before/after on changed row: %#v", changed)
	}
	if changed["hasActor"] != true || changed["actorLabel"] != "Jane Doe" {
		t.Fatalf("expected actor attribution on changed row: %#v", changed)
	}
	if changed["hasWhen"] != true || changed["whenLabel"] != "Jul 10, 2026 3:04 PM" || changed["whenMachine"] != "2026-07-10T15:04:00Z" {
		t.Fatalf("unexpected when formatting on changed row: %#v", changed)
	}

	added := rows[1]
	if added["kind"] != "added" || added["kindLabel"] != "Added" {
		t.Fatalf("expected a no-prior-value row to classify as added, got %#v", added)
	}
	if added["hasBefore"] != false || added["before"] != "" {
		t.Fatalf("expected an added row to have no fabricated before value, got %#v", added)
	}
	if added["hasActor"] != false || added["actorLabel"] != "" {
		t.Fatalf("expected no fabricated actor label when none was supplied, got %#v", added)
	}
	if added["hasWhen"] != false || added["whenLabel"] != "" || added["whenMachine"] != "" {
		t.Fatalf("expected no fabricated when value when none was supplied, got %#v", added)
	}

	removed := rows[2]
	if removed["kind"] != "removed" || removed["kindLabel"] != "Removed" {
		t.Fatalf("expected a no-next-value row to classify as removed, got %#v", removed)
	}
	if removed["hasAfter"] != false || removed["after"] != "" {
		t.Fatalf("expected a removed row to have no fabricated after value, got %#v", removed)
	}
	if removed["hasBefore"] != true || removed["before"] != "1px solid red" {
		t.Fatalf("expected the removed row to keep its real before value, got %#v", removed)
	}
}

// TestPublishChangeSetViewDropsBlankLabelChanges mirrors
// normalizePublishChecks/normalizePublishImpacts' existing discipline: a
// change with no label is a degenerate/incomplete entry and is silently
// dropped rather than rendered as a blank row.
func TestPublishChangeSetViewDropsBlankLabelChanges(t *testing.T) {
	view := PublishChangeSetView(PublishChangeSet{
		Supplied: true,
		Changes: []PublishChange{
			{Scope: PublishChangeScopeContent, Label: "  ", After: "x", AfterSet: true},
			{Scope: PublishChangeScopeContent, Label: "site.title", After: "New title", AfterSet: true},
		},
	}, nil)
	if view["changeCount"] != 1 {
		t.Fatalf("expected the blank-label change to be dropped, got changeCount=%v", view["changeCount"])
	}
}
