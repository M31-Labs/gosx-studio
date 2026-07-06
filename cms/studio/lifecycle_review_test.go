package studio

import (
	"context"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	lifecyclesql "m31labs.dev/gosx-studio/cms/lifecycle/sqlstore"
)

func TestLifecycleReviewStateLoadsLedgerData(t *testing.T) {
	ledger, closeLedger, err := lifecyclesql.Open(":memory:")
	if err != nil {
		t.Fatalf("open ledger: %v", err)
	}
	t.Cleanup(func() {
		if err := closeLedger(); err != nil {
			t.Fatalf("close ledger: %v", err)
		}
	})
	ctx := context.Background()
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	if _, err := ledger.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{
		ID:           "decision-1",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-1",
		Status:       lifecycle.DecisionApproved,
		ActorID:      "owner",
		Note:         "Approved for release.",
		Created:      base,
	}); err != nil {
		t.Fatalf("save decision: %v", err)
	}
	if _, err := ledger.SavePublishSchedule(ctx, lifecycle.PublishScheduleInput{
		ID:           "schedule-1",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-1",
		Action:       lifecycle.ScheduleActionPublish,
		DueAt:        base.Add(time.Hour),
		ActorID:      "owner",
		Note:         "Publish after review.",
		Created:      base,
	}); err != nil {
		t.Fatalf("save schedule: %v", err)
	}

	state, err := LoadLifecycleReviewState(ctx, ledger, LifecycleReviewQuery{
		ResourceKind:  "page",
		ResourceID:    "home",
		ScheduleLimit: 3,
		Now:           base,
	})
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if !state.HasDecision || state.Decision.Status != lifecycle.DecisionApproved {
		t.Fatalf("unexpected decision state: %#v", state)
	}
	if len(state.Schedules) != 1 || state.Schedules[0].ID != "schedule-1" {
		t.Fatalf("unexpected schedule state: %#v", state.Schedules)
	}
}

func TestLifecyclePublishApprovalMapsDecisionStatuses(t *testing.T) {
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	approval := LifecyclePublishApproval(LifecycleReviewState{
		Decision: lifecycle.PublishDecision{
			Status:  lifecycle.DecisionApproved,
			ActorID: "owner@example.com",
			Note:    "Looks ready.",
			Created: base,
		},
		HasDecision: true,
	}, LifecycleApprovalOptions{Required: true, Reviewer: "Owner"})
	if !approval.Approved || approval.Status != ReadinessReady || approval.Summary != "Approved by owner@example.com" || approval.Detail != "Looks ready." {
		t.Fatalf("unexpected approved state: %#v", approval)
	}

	rejected := LifecyclePublishApproval(LifecycleReviewState{
		Decision:    lifecycle.PublishDecision{Status: lifecycle.DecisionRejected, ActorID: "owner", Created: base},
		HasDecision: true,
	}, LifecycleApprovalOptions{Required: true})
	if rejected.Approved || rejected.Status != ReadinessNext || rejected.Summary != "Rejected by owner" {
		t.Fatalf("unexpected rejected state: %#v", rejected)
	}

	pending := LifecyclePublishApproval(LifecycleReviewState{}, LifecycleApprovalOptions{Required: true})
	if pending.Approved || pending.Summary != "Approval pending" || pending.Detail == "" {
		t.Fatalf("unexpected pending state: %#v", pending)
	}
}

func TestLifecyclePublishScheduleChoosesNextFuturePublish(t *testing.T) {
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	state := LifecycleReviewState{
		Now: base,
		Schedules: []lifecycle.PublishSchedule{
			{ID: "past", Action: lifecycle.ScheduleActionPublish, State: lifecycle.SchedulePending, DueAt: base.Add(-time.Hour)},
			{ID: "unpublish", Action: lifecycle.ScheduleActionUnpublish, State: lifecycle.SchedulePending, DueAt: base.Add(30 * time.Minute)},
			{ID: "future", Action: lifecycle.ScheduleActionPublish, State: lifecycle.SchedulePending, DueAt: base.Add(time.Hour), Note: "After staff review."},
		},
	}
	pending, ok := NextPendingPublishSchedule(state)
	if !ok || pending.ID != "future" {
		t.Fatalf("unexpected pending schedule %#v ok=%v", pending, ok)
	}
	schedule := LifecyclePublishSchedule(state, LifecycleScheduleOptions{})
	if !schedule.Enabled || schedule.PublishAt != base.Add(time.Hour) || schedule.Detail != "After staff review." {
		t.Fatalf("unexpected publish schedule: %#v", schedule)
	}
	inputValue := LifecycleScheduleInputValue(state, time.UTC)
	if inputValue != "2026-05-18T13:00" {
		t.Fatalf("unexpected schedule input value %q", inputValue)
	}
	if help := LifecycleScheduleHelp(state, time.UTC); help != "Pending publish: May 18, 2026 1:00 PM." {
		t.Fatalf("unexpected schedule help %q", help)
	}
}

func TestParseLifecyclePublishAt(t *testing.T) {
	location := time.FixedZone("PDT", -7*60*60)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, location)
	parsed, err := ParseLifecyclePublishAt("2026-05-18T13:30", LifecyclePublishAtOptions{Now: now, Location: location})
	if err != nil {
		t.Fatalf("parse datetime-local publish at: %v", err)
	}
	if want := time.Date(2026, 5, 18, 20, 30, 0, 0, time.UTC); !parsed.Equal(want) {
		t.Fatalf("unexpected parsed publish time %s want %s", parsed, want)
	}
	parsed, err = ParseLifecyclePublishAt("2026-05-18T20:45:00Z", LifecyclePublishAtOptions{Now: now, Location: location})
	if err != nil {
		t.Fatalf("parse RFC3339 publish at: %v", err)
	}
	if want := time.Date(2026, 5, 18, 20, 45, 0, 0, time.UTC); !parsed.Equal(want) {
		t.Fatalf("unexpected RFC3339 publish time %s want %s", parsed, want)
	}
	if _, err := ParseLifecyclePublishAt("", LifecyclePublishAtOptions{Now: now, Location: location}); err == nil || err.Error() != "choose a publish time" {
		t.Fatalf("expected empty publish time error, got %v", err)
	}
	if _, err := ParseLifecyclePublishAt("not-a-date", LifecyclePublishAtOptions{Now: now, Location: location}); err == nil || err.Error() != "use YYYY-MM-DD HH:MM" {
		t.Fatalf("expected invalid publish time error, got %v", err)
	}
	if _, err := ParseLifecyclePublishAt("2026-05-18T11:58", LifecyclePublishAtOptions{Now: now, Location: location, PastGrace: time.Minute}); err == nil || err.Error() != "choose a future publish time" {
		t.Fatalf("expected past publish time error, got %v", err)
	}
}

func TestLifecycleStateDefaults(t *testing.T) {
	if LifecycleDraftStateFromRevisions(0) != lifecycle.DraftStateDraft {
		t.Fatalf("expected draft state with no revisions")
	}
	if LifecycleDraftStateFromRevisions(1) != lifecycle.DraftStatePreview {
		t.Fatalf("expected preview state with revisions")
	}
	if LifecyclePublishStateFromDecision(LifecycleReviewState{}) != lifecycle.PublishStateDraft {
		t.Fatalf("expected draft publish state with no decision")
	}
	if LifecyclePublishStateFromDecision(LifecycleReviewState{
		Decision:    lifecycle.PublishDecision{Status: lifecycle.DecisionApproved},
		HasDecision: true,
	}) != lifecycle.PublishStatePublished {
		t.Fatalf("expected published state with approved decision")
	}
}
