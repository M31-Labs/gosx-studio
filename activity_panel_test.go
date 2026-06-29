package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderActivityPanelFull(t *testing.T) {
	view := activityPanelTestView(true, true)

	html := gosx.RenderHTML(RenderActivityPanel(view, ActivityPanelOptions{}))

	for _, fragment := range []string{
		`<section class="studio-activity-drawer" data-studio-activity-drawer="true" aria-label="Activity and readiness" data-gosx-studio-activity-panel-renderer="gosx-studio">`,
		`<div class="studio-activity-head"><div><p class="kicker">Activity</p><h2>Readiness</h2></div><output class="studio-readiness-score">2/3 ready</output><button type="button" data-studio-activity-toggle="true" aria-pressed="true">Hide</button></div>`,
		`<div class="studio-readiness-summary" aria-label="Readiness summary"><span><strong>2</strong>ready</span><span><strong>1</strong>watch</span><span><strong>0</strong>next</span></div>`,
		`<article class="studio-readiness-card studio-readiness-card--ready" data-readiness-key="content"><div class="studio-readiness-card__head"><div><strong>Content</strong><span>Homepage ready</span></div><output>Ready</output></div><p>All required sections are complete.</p><a href="#content" data-gosx-link="true">Review content</a></article>`,
		`<article class="studio-readiness-card studio-readiness-card--watch" data-readiness-key="media"><div class="studio-readiness-card__head"><div><strong>Media</strong><span>Alt text recommended</span></div><output>Watch</output></div></article>`,
		`<section class="studio-health" data-studio-health="true">`,
		`<div class="studio-health__summary" aria-label="Site health summary"><span><strong>1</strong>healthy</span><span><strong>1</strong>watch</span><span><strong>0</strong>next</span></div>`,
		`<article class="studio-health__card studio-health__card--watch" data-studio-health-check="seo"><div class="studio-health__card-head"><div><strong>SEO title</strong><span>Homepage</span></div><output>Watch</output></div><p class="studio-health__value">42 chars</p><p class="studio-health__detail">Consider a longer page title.</p><a href="#seo" data-gosx-link="true">Fix SEO</a></article>`,
		`<section class="studio-performance" data-studio-performance="true">`,
		`<div class="studio-performance__summary" aria-label="Performance summary"><span><strong>1</strong>ready</span><span><strong>0</strong>watch</span><span><strong>1</strong>next</span></div>`,
		`<article class="studio-performance__card studio-performance__card--ready" data-studio-performance-signal="lcp"><div class="studio-performance__card-head"><div><strong>LCP</strong><span>1.8s</span></div><output>Ready</output></div><p class="studio-performance__budget">&lt; 2.5s</p><p class="studio-performance__summary-text">Hero image is optimized.</p></article>`,
		`<section class="studio-comments"><div class="studio-comments__head"><div><p class="studio-comments__kicker">Discuss</p><h2>Comments</h2></div><output class="studio-comments__count">0 open</output></div><article class="studio-comments__empty"><strong>No comments</strong><p>Review notes will appear here.</p></article></section>`,
		`<section class="studio-proposals"><div class="studio-proposals__head"><div><p class="studio-proposals__kicker">Review</p><h2>Proposals</h2></div><output class="studio-proposals__count">0 pending</output></div><article class="studio-proposals__empty"><strong>No proposals</strong><p>Suggestions will appear here.</p></article></section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered activity panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		"<form",
		"csrf_token",
		`<p></p>`,
		`href=""`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered activity panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderActivityPanelMinimalOmitsOptionalPanels(t *testing.T) {
	view := activityPanelTestView(false, false)
	view["togglePressed"] = false
	readiness := workbenchViewMap(view, "readiness")
	readiness["items"] = []map[string]any{
		{
			"key":         "content",
			"class":       "studio-readiness-card studio-readiness-card--ready",
			"label":       "Content",
			"summary":     "Homepage ready",
			"statusLabel": "Ready",
			"hasHref":     false,
		},
	}

	html := gosx.RenderHTML(RenderActivityPanel(view, ActivityPanelOptions{}))

	for _, fragment := range []string{
		`data-gosx-studio-activity-panel-renderer="gosx-studio"`,
		`data-studio-activity-drawer="true"`,
		`data-studio-activity-toggle="true" aria-pressed="false"`,
		`<article class="studio-readiness-card studio-readiness-card--ready" data-readiness-key="content"><div class="studio-readiness-card__head"><div><strong>Content</strong><span>Homepage ready</span></div><output>Ready</output></div></article>`,
		`<section class="studio-comments">`,
		`<section class="studio-proposals">`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("minimal activity panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`data-studio-health="true"`,
		`data-studio-health-check=`,
		`data-studio-performance="true"`,
		`data-studio-performance-signal=`,
		`data-gosx-link="true"`,
		"<form",
		"csrf_token",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("minimal activity panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderActivityPanelStructuredCommentsAndProposals(t *testing.T) {
	view := activityPanelTestView(false, false)
	comments := workbenchViewMap(view, "comments")
	comments["countLabel"] = "1 open"
	comments["comments"] = []map[string]any{
		{
			"id":           "comment-hero-headline",
			"body":         "The lead is accurate, but it could feel warmer for a first-time shopper.",
			"actorID":      "agent-studio",
			"actorKind":    "agent",
			"targetLabel":  "hero / headline",
			"blockID":      "hero",
			"field":        "headline",
			"status":       "open",
			"statusLabel":  "Open",
			"canResolve":   true,
			"resolveEvent": "studio.resolveComment",
		},
	}
	proposals := workbenchViewMap(view, "proposals")
	proposals["countLabel"] = "1 pending"
	proposals["proposals"] = []map[string]any{
		{
			"id":             "proposal-home-lead",
			"title":          "Warm up the homepage lead",
			"summary":        "Make the hero copy more inviting before publish.",
			"actorID":        "agent-studio",
			"actorKind":      "agent",
			"status":         "pending",
			"statusLabel":    "Pending",
			"operationCount": 2,
			"reviewSummary":  "2 suggested copy operations need review.",
			"canAccept":      true,
			"canReject":      true,
			"acceptEvent":    "studio.acceptSuggestion",
			"rejectEvent":    "studio.rejectSuggestion",
			"items": []map[string]any{
				{
					"operationID": "op-headline",
					"kind":        "set_text",
					"summary":     "Update the hero headline with a warmer seasonal lead.",
				},
				{
					"operationID": "op-subhead",
					"kind":        "set_text",
					"summary":     "Tighten the supporting line around ready-to-ship favorites.",
				},
			},
		},
	}

	html := gosx.RenderHTML(RenderActivityPanel(view, ActivityPanelOptions{}))

	for _, fragment := range []string{
		`<section class="studio-comments"><div class="studio-comments__head"><div><p class="studio-comments__kicker">Discuss</p><h2>Comments</h2></div><output class="studio-comments__count">1 open</output></div><div class="studio-comments__list">`,
		`<article class="studio-comments__card studio-comments__card--open" data-studio-comment="comment-hero-headline" data-studio-comment-status="open" data-studio-comment-block="hero" data-studio-comment-field="headline">`,
		`<strong>hero / headline</strong><span>From agent-studio</span>`,
		`<p class="studio-comments__body">The lead is accurate, but it could feel warmer for a first-time shopper.</p>`,
		`<button type="button" class="studio-comments__resolve" data-studio-comment-action="resolve" data-studio-comment-id="comment-hero-headline" data-studio-comment-event="studio.resolveComment">Resolve</button>`,
		`<section class="studio-proposals"><div class="studio-proposals__head"><div><p class="studio-proposals__kicker">Review</p><h2>Proposals</h2></div><output class="studio-proposals__count">1 pending</output></div><div class="studio-proposals__list">`,
		`<article class="studio-proposals__card studio-proposals__card--pending" data-studio-proposal="proposal-home-lead" data-studio-proposal-status="pending">`,
		`<strong>Warm up the homepage lead</strong><span>2 operations from agent-studio</span>`,
		`<p class="studio-proposals__summary">Make the hero copy more inviting before publish.</p>`,
		`<p class="studio-proposals__review">2 suggested copy operations need review.</p>`,
		`<li data-studio-proposal-operation="op-headline" data-studio-proposal-operation-kind="set_text">Update the hero headline with a warmer seasonal lead.</li>`,
		`<button type="button" class="studio-proposals__accept" data-studio-proposal-action="accept" data-studio-proposal-id="proposal-home-lead" data-studio-proposal-event="studio.acceptSuggestion">Accept</button>`,
		`<button type="button" class="studio-proposals__reject" data-studio-proposal-action="reject" data-studio-proposal-id="proposal-home-lead" data-studio-proposal-event="studio.rejectSuggestion">Reject</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("structured activity panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`studio-comments__empty`,
		`studio-proposals__empty`,
		"<form",
		"csrf_token",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("structured activity panel must not include %q:\n%s", notWant, html)
		}
	}
}

func activityPanelTestView(hasHealth, hasPerformance bool) map[string]any {
	return map[string]any{
		"class":         "studio-activity-drawer",
		"kicker":        "Activity",
		"title":         "Readiness",
		"score":         "2/3 ready",
		"toggleLabel":   "Hide",
		"togglePressed": true,
		"readiness": map[string]any{
			"readyCount": "2",
			"watchCount": "1",
			"nextCount":  "0",
			"items": []map[string]any{
				{
					"key":         "content",
					"class":       "studio-readiness-card studio-readiness-card--ready",
					"label":       "Content",
					"summary":     "Homepage ready",
					"statusLabel": "Ready",
					"detail":      "All required sections are complete.",
					"hasHref":     true,
					"href":        "#content",
					"actionLabel": "Review content",
				},
				{
					"key":         "media",
					"class":       "studio-readiness-card studio-readiness-card--watch",
					"label":       "Media",
					"summary":     "Alt text recommended",
					"statusLabel": "Watch",
					"hasHref":     false,
				},
			},
		},
		"hasHealth": hasHealth,
		"health": map[string]any{
			"class":      "studio-health",
			"kicker":     "Health",
			"title":      "Site health",
			"summary":    "1/2 healthy",
			"readyCount": "1",
			"watchCount": "1",
			"nextCount":  "0",
			"checks": []map[string]any{
				{
					"key":         "seo",
					"class":       "studio-health__card studio-health__card--watch",
					"label":       "SEO title",
					"scope":       "Homepage",
					"statusLabel": "Watch",
					"value":       "42 chars",
					"detail":      "Consider a longer page title.",
					"hasHref":     true,
					"href":        "#seo",
					"actionLabel": "Fix SEO",
				},
			},
		},
		"hasPerformance": hasPerformance,
		"performance": map[string]any{
			"class":      "studio-performance",
			"kicker":     "Performance",
			"title":      "Smoothness",
			"summary":    "1/2 ready",
			"readyCount": "1",
			"watchCount": "0",
			"nextCount":  "1",
			"signals": []map[string]any{
				{
					"key":         "lcp",
					"class":       "studio-performance__card studio-performance__card--ready",
					"label":       "LCP",
					"value":       "1.8s",
					"statusLabel": "Ready",
					"budget":      "< 2.5s",
					"summary":     "Hero image is optimized.",
				},
			},
		},
		"comments": activityEmptyPanelTestView("studio-comments", "Discuss", "Comments", "0 open", "No comments", "Review notes will appear here."),
		"proposals": activityEmptyPanelTestView(
			"studio-proposals",
			"Review",
			"Proposals",
			"0 pending",
			"No proposals",
			"Suggestions will appear here.",
		),
	}
}

func activityEmptyPanelTestView(className, kicker, title, countLabel, emptyTitle, emptyDetail string) map[string]any {
	return map[string]any{
		"class":       className,
		"headClass":   className + "__head",
		"kickerClass": className + "__kicker",
		"countClass":  className + "__count",
		"emptyClass":  className + "__empty",
		"kicker":      kicker,
		"title":       title,
		"countLabel":  countLabel,
		"emptyTitle":  emptyTitle,
		"emptyDetail": emptyDetail,
	}
}
