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
