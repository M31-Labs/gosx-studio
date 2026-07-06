package studio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
)

type LifecycleReviewQuery struct {
	ResourceKind  string
	ResourceID    string
	ScheduleLimit int
	Now           time.Time
}

type LifecycleReviewState struct {
	Decision    lifecycle.PublishDecision
	HasDecision bool
	Schedules   []lifecycle.PublishSchedule
	Now         time.Time
}

type LifecycleApprovalOptions struct {
	Required     bool
	Label        string
	Reviewer     string
	Href         string
	ActionLabel  string
	DefaultActor string
	EmptyDetail  string
	Location     *time.Location
}

type LifecycleScheduleOptions struct {
	Label         string
	Href          string
	ActionLabel   string
	ManualSummary string
	ManualDetail  string
	Location      *time.Location
}

type LifecyclePublishAtOptions struct {
	Location  *time.Location
	Now       time.Time
	PastGrace time.Duration
}

func LoadLifecycleReviewState(ctx context.Context, store lifecycle.LedgerStore, query LifecycleReviewQuery) (LifecycleReviewState, error) {
	state := LifecycleReviewState{Now: lifecycleReviewNow(query.Now)}
	if store == nil {
		return state, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	resourceKind := strings.TrimSpace(query.ResourceKind)
	resourceID := strings.TrimSpace(query.ResourceID)
	if resourceKind == "" || resourceID == "" {
		return state, nil
	}
	decision, ok, err := store.LatestPublishDecision(ctx, resourceKind, resourceID)
	if err != nil {
		return state, err
	}
	state.Decision = decision
	state.HasDecision = ok
	limit := query.ScheduleLimit
	if limit <= 0 {
		limit = 5
	}
	schedules, err := store.ListPublishSchedules(ctx, lifecycle.ScheduleFilter{
		ResourceKind: resourceKind,
		ResourceID:   resourceID,
		State:        lifecycle.SchedulePending,
		Limit:        limit,
	})
	if err != nil {
		return state, err
	}
	state.Schedules = append([]lifecycle.PublishSchedule(nil), schedules...)
	return state, nil
}

func LifecycleDraftStateFromRevisions(count int) lifecycle.DraftState {
	if count > 0 {
		return lifecycle.DraftStatePreview
	}
	return lifecycle.DraftStateDraft
}

func LifecyclePublishStateFromDecision(state LifecycleReviewState) lifecycle.PublishState {
	if state.HasDecision && state.Decision.Status == lifecycle.DecisionApproved {
		return lifecycle.PublishStatePublished
	}
	return lifecycle.PublishStateDraft
}

func LifecyclePublishApproval(state LifecycleReviewState, options LifecycleApprovalOptions) PublishApproval {
	approval := PublishApproval{
		Required:    options.Required,
		Label:       firstNonEmpty(options.Label, "Owner approval"),
		Reviewer:    strings.TrimSpace(options.Reviewer),
		Summary:     "Approval pending",
		Detail:      firstNonEmpty(options.EmptyDetail, "Publish decisions are persisted in the lifecycle ledger."),
		Status:      ReadinessWatch,
		Href:        strings.TrimSpace(options.Href),
		ActionLabel: strings.TrimSpace(options.ActionLabel),
	}
	if !state.HasDecision {
		return approval
	}
	actor := firstNonEmpty(state.Decision.ActorID, options.DefaultActor, "studio")
	when := LifecycleTimeLabel(state.Decision.Created, options.Location)
	note := strings.TrimSpace(state.Decision.Note)
	switch state.Decision.Status {
	case lifecycle.DecisionApproved:
		approval.Approved = true
		approval.Summary = "Approved by " + actor
		approval.Detail = firstNonEmpty(note, "Approved "+when+".")
		approval.Status = ReadinessReady
	case lifecycle.DecisionRejected:
		approval.Summary = "Rejected by " + actor
		approval.Detail = firstNonEmpty(note, "Rejected "+when+".")
		approval.Status = ReadinessNext
	case lifecycle.DecisionChangesRequested:
		approval.Summary = "Changes requested by " + actor
		approval.Detail = firstNonEmpty(note, "Changes requested "+when+".")
		approval.Status = ReadinessWatch
	default:
		approval.Summary = "Review pending by " + actor
		approval.Detail = firstNonEmpty(note, "Review opened "+when+".")
	}
	return approval
}

func LifecyclePublishSchedule(state LifecycleReviewState, options LifecycleScheduleOptions) PublishSchedule {
	schedule := PublishSchedule{
		Label:       firstNonEmpty(options.Label, "Publish timing"),
		Summary:     firstNonEmpty(options.ManualSummary, "Manual publish"),
		Detail:      firstNonEmpty(options.ManualDetail, "No future publish time is set; this draft goes live only through the explicit publish action."),
		Status:      ReadinessReady,
		Href:        strings.TrimSpace(options.Href),
		ActionLabel: strings.TrimSpace(options.ActionLabel),
	}
	pending, ok := NextPendingPublishSchedule(state)
	if !ok {
		return schedule
	}
	schedule.Enabled = true
	schedule.PublishAt = pending.DueAt
	schedule.Timezone = pending.Timezone
	schedule.Summary = "Scheduled for " + LifecycleTimeLabel(pending.DueAt, options.Location)
	schedule.Detail = firstNonEmpty(strings.TrimSpace(pending.Note), "A pending publish schedule is stored in the lifecycle ledger.")
	schedule.Status = ReadinessReady
	return schedule
}

func NextPendingPublishSchedule(state LifecycleReviewState) (lifecycle.PublishSchedule, bool) {
	now := lifecycleReviewNow(state.Now)
	for _, schedule := range state.Schedules {
		if !publishScheduleCandidate(schedule) {
			continue
		}
		if !schedule.DueAt.IsZero() && schedule.DueAt.Before(now) {
			continue
		}
		return schedule, true
	}
	for _, schedule := range state.Schedules {
		if publishScheduleCandidate(schedule) {
			return schedule, true
		}
	}
	return lifecycle.PublishSchedule{}, false
}

func LifecycleScheduleInputValue(state LifecycleReviewState, location *time.Location) string {
	schedule, ok := NextPendingPublishSchedule(state)
	if !ok || schedule.DueAt.IsZero() {
		return ""
	}
	if location == nil {
		location = time.Local
	}
	return schedule.DueAt.In(location).Format("2006-01-02T15:04")
}

func LifecycleScheduleHelp(state LifecycleReviewState, location *time.Location) string {
	schedule, ok := NextPendingPublishSchedule(state)
	if !ok {
		return "Set a future publish time before using Schedule."
	}
	return "Pending publish: " + LifecycleTimeLabel(schedule.DueAt, location) + "."
}

func ParseLifecyclePublishAt(value string, options LifecyclePublishAtOptions) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("choose a publish time")
	}
	location := options.Location
	if location == nil {
		location = time.Local
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	pastGrace := options.PastGrace
	if pastGrace == 0 {
		pastGrace = time.Minute
	}
	for _, layout := range []string{"2006-01-02T15:04", time.RFC3339, time.RFC3339Nano} {
		if parsed, err := time.ParseInLocation(layout, value, location); err == nil {
			if parsed.Before(now.Add(-pastGrace)) {
				return time.Time{}, fmt.Errorf("choose a future publish time")
			}
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("use YYYY-MM-DD HH:MM")
}

func LifecycleTimeLabel(value time.Time, location *time.Location) string {
	if value.IsZero() {
		return "now"
	}
	if location == nil {
		location = time.Local
	}
	return value.In(location).Format("Jan 2, 2006 3:04 PM")
}

func publishScheduleCandidate(schedule lifecycle.PublishSchedule) bool {
	if schedule.State != lifecycle.SchedulePending {
		return false
	}
	return schedule.Action == "" || schedule.Action == lifecycle.ScheduleActionPublish
}

func lifecycleReviewNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
