package studio

import (
	"fmt"
	"strings"
	"time"

	"m31labs.dev/gosx"
)

type PublishReview struct {
	Key                string
	ResourceKind       string
	ResourceID         string
	Title              string
	Summary            string
	Status             ReadinessStatus
	Approval           PublishApproval
	Schedule           PublishSchedule
	Checks             []PublishCheck
	Impacts            []PublishImpact
	PrimaryHref        string
	PrimaryActionLabel string
}

type PublishCheck struct {
	Key         string
	Label       string
	Scope       string
	Status      ReadinessStatus
	Summary     string
	Detail      string
	Href        string
	ActionLabel string
}

type PublishImpact struct {
	Key    string
	Label  string
	Scope  string
	Value  string
	Detail string
	Status ReadinessStatus
}

type PublishApproval struct {
	Required    bool
	Approved    bool
	Label       string
	Reviewer    string
	Summary     string
	Detail      string
	Status      ReadinessStatus
	Href        string
	ActionLabel string
}

type PublishSchedule struct {
	Enabled     bool
	Label       string
	PublishAt   time.Time
	UnpublishAt time.Time
	Timezone    string
	Summary     string
	Detail      string
	Status      ReadinessStatus
	Href        string
	ActionLabel string
}

type PublishReviewOptions struct {
	Class       string
	Kicker      string
	Title       string
	EmptyTitle  string
	EmptyDetail string
	ImpactTitle string
}

func NewPublishReview(checks ...PublishCheck) PublishReview {
	return NormalizePublishReview(PublishReview{Checks: checks})
}

func NewPublishCheck(key, label, scope string, status ReadinessStatus, summary, detail string) PublishCheck {
	return PublishCheck{
		Key:     key,
		Label:   label,
		Scope:   scope,
		Status:  status,
		Summary: summary,
		Detail:  detail,
	}
}

func (check PublishCheck) WithHref(href string) PublishCheck {
	check.Href = href
	return check
}

func (check PublishCheck) WithActionLabel(label string) PublishCheck {
	check.ActionLabel = label
	return check
}

func NewPublishImpact(key, label, scope, value, detail string, status ReadinessStatus) PublishImpact {
	return PublishImpact{
		Key:    key,
		Label:  label,
		Scope:  scope,
		Value:  value,
		Detail: detail,
		Status: status,
	}
}

func NormalizePublishReview(review PublishReview) PublishReview {
	review.Key = normalizeKey(review.Key)
	review.ResourceKind = strings.TrimSpace(review.ResourceKind)
	review.ResourceID = strings.TrimSpace(review.ResourceID)
	review.Title = strings.TrimSpace(review.Title)
	review.Summary = strings.TrimSpace(review.Summary)
	review.PrimaryHref = strings.TrimSpace(review.PrimaryHref)
	review.PrimaryActionLabel = strings.TrimSpace(review.PrimaryActionLabel)
	review.Approval = normalizePublishApproval(review.Approval)
	review.Schedule = normalizePublishSchedule(review.Schedule)
	review.Checks = normalizePublishChecks(review.Checks)
	review.Impacts = normalizePublishImpacts(review.Impacts)
	if review.Key == "" {
		review.Key = normalizeKey(firstNonEmpty(review.ResourceID, review.Title, review.ResourceKind, "publish-review"))
	}
	if review.ResourceKind == "" {
		review.ResourceKind = "Site"
	}
	if review.Title == "" {
		review.Title = "Publish review"
	}
	review.Status = normalizeReadinessStatus(review.Status)
	if review.Status == ReadinessWatch && (len(review.Checks) > 0 || hasPublishApproval(review.Approval) || hasPublishSchedule(review.Schedule)) {
		review.Status = derivedPublishStatus(review.Checks, review.Approval, review.Schedule)
	}
	if review.Summary == "" {
		ready, _, _, total := publishCheckCounts(review.Checks)
		review.Summary = fmt.Sprintf("%d/%d clear", ready, total)
	}
	if review.PrimaryActionLabel == "" {
		review.PrimaryActionLabel = readinessActionLabel(review.Status)
	}
	return review
}

func (review PublishReview) Counts() (ready, watch, next, total int) {
	return publishCheckCounts(normalizePublishChecks(review.Checks))
}

func publishCheckCounts(checks []PublishCheck) (ready, watch, next, total int) {
	for _, check := range checks {
		switch check.Status {
		case ReadinessReady:
			ready++
		case ReadinessNext:
			next++
		default:
			watch++
		}
	}
	return ready, watch, next, len(checks)
}

func (review PublishReview) CountSummary() string {
	ready, _, _, total := review.Counts()
	return fmt.Sprintf("%d/%d clear", ready, total)
}

func PublishReviewView(review PublishReview) map[string]any {
	review = NormalizePublishReview(review)
	ready, watch, next, total := review.Counts()
	return map[string]any{
		"key":                review.Key,
		"resourceKind":       review.ResourceKind,
		"resourceID":         review.ResourceID,
		"title":              review.Title,
		"summary":            review.Summary,
		"status":             string(review.Status),
		"statusLabel":        readinessStatusLabel(review.Status),
		"readyCount":         ready,
		"watchCount":         watch,
		"nextCount":          next,
		"total":              total,
		"approval":           publishApprovalView(review.Approval),
		"hasApproval":        hasPublishApproval(review.Approval),
		"schedule":           publishScheduleView(review.Schedule),
		"hasSchedule":        hasPublishSchedule(review.Schedule),
		"checks":             publishCheckViews(review.Checks),
		"impacts":            publishImpactViews(review.Impacts),
		"primaryHref":        review.PrimaryHref,
		"hasPrimaryHref":     review.PrimaryHref != "",
		"primaryActionLabel": review.PrimaryActionLabel,
	}
}

func RenderPublishReviewPanel(review PublishReview, options PublishReviewOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "studio-publish-review")
	review = NormalizePublishReview(review)
	ready, watch, next, _ := review.Counts()
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__kicker")), gosx.Text(firstNonEmpty(options.Kicker, review.ResourceKind))),
				gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, review.Title))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", className+"__count")), gosx.Text(review.CountSummary())),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__summary-text")), gosx.Text(review.Summary)),
	}
	if hasPublishApproval(review.Approval) || hasPublishSchedule(review.Schedule) {
		children = append(children, renderPublishDecision(className, review.Approval, review.Schedule))
	}
	if len(review.Checks) == 0 {
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("class", className+"__empty")),
			gosx.El("strong", nil, gosx.Text(firstNonEmpty(options.EmptyTitle, "No publish checks"))),
			gosx.El("p", nil, gosx.Text(firstNonEmpty(options.EmptyDetail, "Register content, approval, flow, and deployment checks before publish."))),
		))
	} else {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", className+"__summary"),
			gosx.Attr("aria-label", "Publish review summary"),
		),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(ready))), gosx.Text(" clear")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(watch))), gosx.Text(" watch")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(next))), gosx.Text(" next")),
		))
		items := make([]gosx.Node, 0, len(review.Checks))
		for _, check := range review.Checks {
			items = append(items, renderPublishCheck(className, check))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(items...)))
	}
	if len(review.Impacts) > 0 {
		children = append(children, renderPublishImpacts(className, firstNonEmpty(options.ImpactTitle, "Publish impact"), review.Impacts))
	}
	if review.PrimaryHref != "" {
		children = append(children, gosx.El("a", gosx.Attrs(
			gosx.Attr("class", className+"__primary"),
			gosx.Attr("href", review.PrimaryHref),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(review.PrimaryActionLabel)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-publish-review", review.Key),
		gosx.Attr("data-studio-publish-status", string(review.Status)),
	), gosx.Fragment(children...))
}

func normalizePublishChecks(checks []PublishCheck) []PublishCheck {
	out := make([]PublishCheck, 0, len(checks))
	for _, check := range checks {
		check.Key = normalizeKey(check.Key)
		check.Label = strings.TrimSpace(check.Label)
		check.Scope = strings.TrimSpace(check.Scope)
		check.Status = normalizeReadinessStatus(check.Status)
		check.Summary = strings.TrimSpace(check.Summary)
		check.Detail = strings.TrimSpace(check.Detail)
		check.Href = strings.TrimSpace(check.Href)
		check.ActionLabel = strings.TrimSpace(check.ActionLabel)
		if check.Key == "" {
			check.Key = normalizeKey(check.Label)
		}
		if check.Scope == "" {
			check.Scope = "Site"
		}
		if check.Summary == "" {
			check.Summary = readinessStatusLabel(check.Status)
		}
		if check.ActionLabel == "" {
			check.ActionLabel = readinessActionLabel(check.Status)
		}
		if check.Key == "" || check.Label == "" {
			continue
		}
		out = append(out, check)
	}
	return out
}

func normalizePublishImpacts(impacts []PublishImpact) []PublishImpact {
	out := make([]PublishImpact, 0, len(impacts))
	for _, impact := range impacts {
		impact.Key = normalizeKey(impact.Key)
		impact.Label = strings.TrimSpace(impact.Label)
		impact.Scope = strings.TrimSpace(impact.Scope)
		impact.Value = strings.TrimSpace(impact.Value)
		impact.Detail = strings.TrimSpace(impact.Detail)
		impact.Status = normalizeReadinessStatus(impact.Status)
		if impact.Key == "" {
			impact.Key = normalizeKey(impact.Label)
		}
		if impact.Scope == "" {
			impact.Scope = "Site"
		}
		if impact.Value == "" {
			impact.Value = "Tracked"
		}
		if impact.Key == "" || impact.Label == "" {
			continue
		}
		out = append(out, impact)
	}
	return out
}

func normalizePublishApproval(approval PublishApproval) PublishApproval {
	approval.Label = strings.TrimSpace(approval.Label)
	approval.Reviewer = strings.TrimSpace(approval.Reviewer)
	approval.Summary = strings.TrimSpace(approval.Summary)
	approval.Detail = strings.TrimSpace(approval.Detail)
	approval.Href = strings.TrimSpace(approval.Href)
	approval.ActionLabel = strings.TrimSpace(approval.ActionLabel)
	if approval.Label == "" {
		approval.Label = "Approval"
	}
	approval.Status = normalizeReadinessStatus(approval.Status)
	if hasPublishApproval(approval) {
		switch {
		case approval.Approved:
			approval.Status = ReadinessReady
		case approval.Required && approval.Status == ReadinessWatch:
			approval.Status = ReadinessNext
		}
		if approval.Summary == "" {
			if approval.Approved {
				approval.Summary = firstNonEmpty(approval.Reviewer, "Approved")
			} else if approval.Required {
				approval.Summary = "Approval required"
			} else {
				approval.Summary = "Optional approval"
			}
		}
		if approval.Detail == "" {
			if approval.Approved {
				approval.Detail = "Release approval is recorded."
			} else if approval.Required {
				approval.Detail = "Collect approval before publishing this draft."
			}
		}
		if approval.ActionLabel == "" {
			approval.ActionLabel = readinessActionLabel(approval.Status)
		}
	}
	return approval
}

func normalizePublishSchedule(schedule PublishSchedule) PublishSchedule {
	schedule.Label = strings.TrimSpace(schedule.Label)
	schedule.Timezone = strings.TrimSpace(schedule.Timezone)
	schedule.Summary = strings.TrimSpace(schedule.Summary)
	schedule.Detail = strings.TrimSpace(schedule.Detail)
	schedule.Href = strings.TrimSpace(schedule.Href)
	schedule.ActionLabel = strings.TrimSpace(schedule.ActionLabel)
	if schedule.Label == "" {
		schedule.Label = "Schedule"
	}
	schedule.Status = normalizeReadinessStatus(schedule.Status)
	if hasPublishSchedule(schedule) {
		if schedule.Enabled {
			if schedule.PublishAt.IsZero() && schedule.Status == ReadinessWatch {
				schedule.Status = ReadinessNext
			} else if !schedule.PublishAt.IsZero() {
				schedule.Status = ReadinessReady
			}
		}
		if schedule.Summary == "" {
			if schedule.Enabled && !schedule.PublishAt.IsZero() {
				schedule.Summary = publishTimeLabel(schedule.PublishAt, schedule.Timezone)
			} else if schedule.Enabled {
				schedule.Summary = "Publish time required"
			} else {
				schedule.Summary = "Manual publish"
			}
		}
		if schedule.Detail == "" {
			if schedule.Enabled {
				schedule.Detail = "This draft is prepared for scheduled publishing."
			} else {
				schedule.Detail = "Publishing runs only after an explicit publish action."
			}
		}
		if schedule.ActionLabel == "" {
			schedule.ActionLabel = readinessActionLabel(schedule.Status)
		}
	}
	return schedule
}

func hasPublishApproval(approval PublishApproval) bool {
	return approval.Required ||
		approval.Approved ||
		strings.TrimSpace(approval.Reviewer) != "" ||
		strings.TrimSpace(approval.Summary) != "" ||
		strings.TrimSpace(approval.Detail) != "" ||
		strings.TrimSpace(approval.Href) != ""
}

func hasPublishSchedule(schedule PublishSchedule) bool {
	return schedule.Enabled ||
		!schedule.PublishAt.IsZero() ||
		!schedule.UnpublishAt.IsZero() ||
		strings.TrimSpace(schedule.Summary) != "" ||
		strings.TrimSpace(schedule.Detail) != "" ||
		strings.TrimSpace(schedule.Href) != ""
}

func derivedPublishStatus(checks []PublishCheck, approval PublishApproval, schedule PublishSchedule) ReadinessStatus {
	status := ReadinessReady
	for _, check := range checks {
		status = highestReadinessStatus(status, normalizeReadinessStatus(check.Status))
		if status == ReadinessNext {
			return status
		}
	}
	if hasPublishApproval(approval) {
		status = highestReadinessStatus(status, normalizeReadinessStatus(approval.Status))
	}
	if status == ReadinessNext {
		return status
	}
	if hasPublishSchedule(schedule) {
		status = highestReadinessStatus(status, normalizeReadinessStatus(schedule.Status))
	}
	return status
}

func highestReadinessStatus(current, candidate ReadinessStatus) ReadinessStatus {
	current = normalizeReadinessStatus(current)
	candidate = normalizeReadinessStatus(candidate)
	if current == ReadinessNext || candidate == ReadinessNext {
		return ReadinessNext
	}
	if current == ReadinessWatch || candidate == ReadinessWatch {
		return ReadinessWatch
	}
	return ReadinessReady
}

func publishCheckViews(checks []PublishCheck) []map[string]any {
	out := make([]map[string]any, 0, len(checks))
	for _, check := range checks {
		status := normalizeReadinessStatus(check.Status)
		out = append(out, map[string]any{
			"key":         check.Key,
			"label":       check.Label,
			"scope":       check.Scope,
			"status":      string(status),
			"statusLabel": readinessStatusLabel(status),
			"class":       "studio-publish-review__card studio-publish-review__card--" + string(status),
			"summary":     check.Summary,
			"detail":      check.Detail,
			"href":        check.Href,
			"hasHref":     check.Href != "",
			"actionLabel": firstNonEmpty(check.ActionLabel, readinessActionLabel(status)),
		})
	}
	return out
}

func publishApprovalView(approval PublishApproval) map[string]any {
	approval = normalizePublishApproval(approval)
	status := normalizeReadinessStatus(approval.Status)
	return map[string]any{
		"required":    approval.Required,
		"approved":    approval.Approved,
		"label":       approval.Label,
		"reviewer":    approval.Reviewer,
		"summary":     approval.Summary,
		"detail":      approval.Detail,
		"status":      string(status),
		"statusLabel": readinessStatusLabel(status),
		"class":       "studio-publish-review__decision-card studio-publish-review__decision-card--" + string(status),
		"href":        approval.Href,
		"hasHref":     approval.Href != "",
		"actionLabel": firstNonEmpty(approval.ActionLabel, readinessActionLabel(status)),
	}
}

func publishScheduleView(schedule PublishSchedule) map[string]any {
	schedule = normalizePublishSchedule(schedule)
	status := normalizeReadinessStatus(schedule.Status)
	publishAt := ""
	unpublishAt := ""
	if !schedule.PublishAt.IsZero() {
		publishAt = schedule.PublishAt.Format(time.RFC3339)
	}
	if !schedule.UnpublishAt.IsZero() {
		unpublishAt = schedule.UnpublishAt.Format(time.RFC3339)
	}
	return map[string]any{
		"enabled":     schedule.Enabled,
		"label":       schedule.Label,
		"publishAt":   publishAt,
		"unpublishAt": unpublishAt,
		"timezone":    schedule.Timezone,
		"summary":     schedule.Summary,
		"detail":      schedule.Detail,
		"status":      string(status),
		"statusLabel": readinessStatusLabel(status),
		"class":       "studio-publish-review__decision-card studio-publish-review__decision-card--" + string(status),
		"href":        schedule.Href,
		"hasHref":     schedule.Href != "",
		"actionLabel": firstNonEmpty(schedule.ActionLabel, readinessActionLabel(status)),
	}
}

func publishImpactViews(impacts []PublishImpact) []map[string]any {
	out := make([]map[string]any, 0, len(impacts))
	for _, impact := range impacts {
		status := normalizeReadinessStatus(impact.Status)
		out = append(out, map[string]any{
			"key":         impact.Key,
			"label":       impact.Label,
			"scope":       impact.Scope,
			"value":       impact.Value,
			"detail":      impact.Detail,
			"status":      string(status),
			"statusLabel": readinessStatusLabel(status),
			"class":       "studio-publish-review__impact studio-publish-review__impact--" + string(status),
		})
	}
	return out
}

func renderPublishCheck(className string, check PublishCheck) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__card-head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(check.Label)),
				gosx.El("span", nil, gosx.Text(check.Scope)),
			),
			gosx.El("output", nil, gosx.Text(readinessStatusLabel(check.Status))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__check-summary")), gosx.Text(check.Summary)),
	}
	if check.Detail != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__detail")), gosx.Text(check.Detail)))
	}
	if check.Href != "" {
		children = append(children, gosx.El("a", gosx.Attrs(
			gosx.Attr("href", check.Href),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(check.ActionLabel)))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", className+"__card "+className+"__card--"+string(check.Status)),
		gosx.Attr("data-studio-publish-check", check.Key),
	), gosx.Fragment(children...))
}

func renderPublishDecision(className string, approval PublishApproval, schedule PublishSchedule) gosx.Node {
	items := []gosx.Node{}
	if hasPublishApproval(approval) {
		items = append(items, renderPublishDecisionCard(className, "approval", approval.Label, approval.Summary, approval.Detail, approval.Status, approval.Href, approval.ActionLabel))
	}
	if hasPublishSchedule(schedule) {
		items = append(items, renderPublishDecisionCard(className, "schedule", schedule.Label, schedule.Summary, schedule.Detail, schedule.Status, schedule.Href, schedule.ActionLabel))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className+"__decision"),
		gosx.Attr("aria-label", "Publish decision"),
	), gosx.Fragment(items...))
}

func renderPublishDecisionCard(className, key, label, summary, detail string, status ReadinessStatus, href, actionLabel string) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__decision-head")),
			gosx.El("strong", nil, gosx.Text(label)),
			gosx.El("output", nil, gosx.Text(readinessStatusLabel(status))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__decision-summary")), gosx.Text(summary)),
	}
	if detail != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__detail")), gosx.Text(detail)))
	}
	if href != "" {
		children = append(children, gosx.El("a", gosx.Attrs(
			gosx.Attr("href", href),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(actionLabel)))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", className+"__decision-card "+className+"__decision-card--"+string(status)),
		gosx.Attr("data-studio-publish-"+key, "true"),
	), gosx.Fragment(children...))
}

func renderPublishImpacts(className, title string, impacts []PublishImpact) gosx.Node {
	items := make([]gosx.Node, 0, len(impacts))
	for _, impact := range impacts {
		children := []gosx.Node{
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(impact.Label)),
				gosx.El("span", nil, gosx.Text(impact.Scope)),
			),
			gosx.El("output", nil, gosx.Text(impact.Value)),
		}
		if impact.Detail != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(impact.Detail)))
		}
		items = append(items, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", className+"__impact "+className+"__impact--"+string(impact.Status)),
			gosx.Attr("data-studio-publish-impact", impact.Key),
		), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__impacts")),
		gosx.El("h3", nil, gosx.Text(title)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__impact-list")), gosx.Fragment(items...)),
	)
}

func publishTimeLabel(value time.Time, timezone string) string {
	if value.IsZero() {
		return ""
	}
	if timezone != "" {
		if location, err := time.LoadLocation(timezone); err == nil {
			value = value.In(location)
		}
	}
	return value.Format("Jan 2, 2006 3:04 PM MST")
}
