package sqlstore_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	"m31labs.dev/gosx-studio/cms/lifecycle/sqlstore"
)

func TestStoreMigratesAndRoundTripsRevisions(t *testing.T) {
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	store := openStore(t, func() time.Time { return base })
	ctx := context.Background()

	first, err := store.SaveRevisionContext(ctx, lifecycle.RevisionInput{
		ID:            "rev-1",
		ResourceKind:  "page",
		ResourceID:    "home",
		ResourceTitle: "Home",
		Action:        "page.draft_saved",
		Summary:       "First pass",
		Snapshot:      map[string]string{"title": "Home"},
		Created:       base,
	})
	if err != nil {
		t.Fatalf("save first revision: %v", err)
	}
	_, err = store.SaveRevisionContext(ctx, lifecycle.RevisionInput{
		ID:           "rev-2",
		ResourceKind: "page",
		ResourceID:   "home",
		Action:       "page.draft_saved",
		Snapshot:     map[string]string{"title": "Home updated"},
		Created:      base.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("save second revision: %v", err)
	}

	latest, err := store.ListRevisionsContext(ctx, lifecycle.RevisionFilter{ResourceKind: "page", ResourceID: "home", Limit: 1})
	if err != nil {
		t.Fatalf("list revisions: %v", err)
	}
	if len(latest) != 1 || latest[0].ID != "rev-2" {
		t.Fatalf("expected latest revision rev-2, got %#v", latest)
	}

	found, ok, err := store.RevisionByIDContext(ctx, "page", "home", first.ID)
	if err != nil {
		t.Fatalf("find revision: %v", err)
	}
	if !ok {
		t.Fatalf("expected revision %q to exist", first.ID)
	}
	var snapshot map[string]string
	if err := json.Unmarshal(found.Snapshot, &snapshot); err != nil {
		t.Fatalf("decode snapshot: %v", err)
	}
	if snapshot["title"] != "Home" {
		t.Fatalf("unexpected snapshot title %q", snapshot["title"])
	}
}

func TestStorePersistsDecisionsNotesAndAudit(t *testing.T) {
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	store := openStore(t, func() time.Time { return base })
	ctx := context.Background()

	if _, err := store.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{
		ID:           "decision-1",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-1",
		Status:       lifecycle.DecisionPending,
		ActorID:      "editor",
		Note:         "Ready for review",
		Created:      base,
	}); err != nil {
		t.Fatalf("save pending decision: %v", err)
	}
	if _, err := store.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{
		ID:           "decision-2",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-1",
		Status:       lifecycle.DecisionApproved,
		ActorID:      "owner",
		Note:         "Approved",
		Created:      base.Add(time.Minute),
	}); err != nil {
		t.Fatalf("save approved decision: %v", err)
	}

	latest, ok, err := store.LatestPublishDecision(ctx, "page", "home")
	if err != nil {
		t.Fatalf("latest decision: %v", err)
	}
	if !ok || latest.Status != lifecycle.DecisionApproved || latest.ActorID != "owner" {
		t.Fatalf("unexpected latest decision: %#v", latest)
	}

	decisions, err := store.ListPublishDecisions(ctx, lifecycle.LedgerFilter{ResourceKind: "page", ResourceID: "home"})
	if err != nil {
		t.Fatalf("list decisions: %v", err)
	}
	if len(decisions) != 2 || decisions[0].ID != "decision-2" {
		t.Fatalf("unexpected decisions: %#v", decisions)
	}

	note, err := store.SavePublishNote(ctx, lifecycle.PublishNoteInput{
		ID:           "note-1",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-1",
		ActorID:      "owner",
		Body:         "Publish before service hours.",
		Created:      base.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("save publish note: %v", err)
	}
	notes, err := store.ListPublishNotes(ctx, lifecycle.LedgerFilter{ResourceKind: "page", ResourceID: "home"})
	if err != nil {
		t.Fatalf("list publish notes: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != note.ID {
		t.Fatalf("unexpected notes: %#v", notes)
	}

	event, err := store.SaveAuditEvent(ctx, lifecycle.AuditEventInput{
		ID:           "audit-1",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-1",
		Action:       "publish.completed",
		ActorID:      "publisher",
		Summary:      "Published homepage",
		Metadata:     map[string]string{"source": "test"},
		Created:      base.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("save audit event: %v", err)
	}
	events, err := store.ListAuditEvents(ctx, lifecycle.LedgerFilter{Action: "publish.completed"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 1 || events[0].ID != event.ID || events[0].Metadata["source"] != "test" {
		t.Fatalf("unexpected audit events: %#v", events)
	}
}

func TestStoreClaimsCompletesAndCancelsSchedules(t *testing.T) {
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	now := base
	store := openStore(t, func() time.Time { return now })
	ctx := context.Background()

	if _, err := store.SavePublishSchedule(ctx, lifecycle.PublishScheduleInput{
		ID:           "schedule-due",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-1",
		DueAt:        base.Add(-time.Minute),
		ActorID:      "editor",
	}); err != nil {
		t.Fatalf("save due schedule: %v", err)
	}
	if _, err := store.SavePublishSchedule(ctx, lifecycle.PublishScheduleInput{
		ID:           "schedule-future",
		ResourceKind: "page",
		ResourceID:   "home",
		RevisionID:   "rev-2",
		Action:       lifecycle.ScheduleActionUnpublish,
		DueAt:        base.Add(time.Hour),
		ActorID:      "editor",
	}); err != nil {
		t.Fatalf("save future schedule: %v", err)
	}

	claimed, err := store.ClaimDueSchedules(ctx, base, 10, "worker-1")
	if err != nil {
		t.Fatalf("claim due schedules: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != "schedule-due" {
		t.Fatalf("unexpected claimed schedules: %#v", claimed)
	}
	if claimed[0].State != lifecycle.ScheduleClaimed || claimed[0].ClaimToken != "worker-1" || claimed[0].ClaimedAt == nil {
		t.Fatalf("schedule was not marked claimed: %#v", claimed[0])
	}

	claimedAgain, err := store.ClaimDueSchedules(ctx, base, 10, "worker-2")
	if err != nil {
		t.Fatalf("claim due schedules again: %v", err)
	}
	if len(claimedAgain) != 0 {
		t.Fatalf("expected no second claim, got %#v", claimedAgain)
	}

	now = base.Add(2 * time.Minute)
	completed, ok, err := store.CompletePublishSchedule(ctx, "schedule-due")
	if err != nil {
		t.Fatalf("complete schedule: %v", err)
	}
	if !ok || completed.State != lifecycle.ScheduleCompleted || completed.CompletedAt == nil {
		t.Fatalf("schedule was not completed: %#v", completed)
	}

	cancelled, ok, err := store.CancelPublishSchedule(ctx, "schedule-future", "owner", "Hold for review")
	if err != nil {
		t.Fatalf("cancel future schedule: %v", err)
	}
	if !ok || cancelled.State != lifecycle.ScheduleCancelled || cancelled.CancelledAt == nil {
		t.Fatalf("future schedule was not cancelled: %#v", cancelled)
	}

	cancelledSchedules, err := store.ListPublishSchedules(ctx, lifecycle.ScheduleFilter{State: lifecycle.ScheduleCancelled})
	if err != nil {
		t.Fatalf("list cancelled schedules: %v", err)
	}
	if len(cancelledSchedules) != 1 || cancelledSchedules[0].ID != "schedule-future" {
		t.Fatalf("unexpected cancelled schedules: %#v", cancelledSchedules)
	}
}

func TestStoreRejectsIncompleteLedgerInputs(t *testing.T) {
	base := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	store := openStore(t, func() time.Time { return base })
	ctx := context.Background()

	if _, err := store.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{ResourceKind: "page"}); err == nil {
		t.Fatalf("expected decision without resource id to fail")
	}
	if _, err := store.SavePublishDecision(ctx, lifecycle.PublishDecisionInput{
		ResourceKind: "page",
		ResourceID:   "home",
		Status:       lifecycle.DecisionStatus("deferred"),
	}); err == nil {
		t.Fatalf("expected unsupported decision status to fail")
	}
	if _, err := store.SavePublishSchedule(ctx, lifecycle.PublishScheduleInput{ResourceKind: "page", ResourceID: "home"}); err == nil {
		t.Fatalf("expected schedule without due time to fail")
	}
	if _, err := store.SavePublishSchedule(ctx, lifecycle.PublishScheduleInput{
		ResourceKind: "page",
		ResourceID:   "home",
		Action:       lifecycle.ScheduleAction("archive"),
		DueAt:        base,
	}); err == nil {
		t.Fatalf("expected unsupported schedule action to fail")
	}
	if _, err := store.SavePublishNote(ctx, lifecycle.PublishNoteInput{ResourceKind: "page", ResourceID: "home", Body: "   "}); err == nil {
		t.Fatalf("expected empty note body to fail")
	}
	if _, err := store.SaveAuditEvent(ctx, lifecycle.AuditEventInput{}); err == nil {
		t.Fatalf("expected audit event without action to fail")
	}
}

func TestOpenCreatesParentDirectoryAndMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "lifecycle.db")
	store, closeStore, err := sqlstore.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := closeStore(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if _, err := store.SavePublishDecision(context.Background(), lifecycle.PublishDecisionInput{
		ResourceKind: "page",
		ResourceID:   "home",
		Status:       lifecycle.DecisionApproved,
	}); err != nil {
		t.Fatalf("save decision after open: %v", err)
	}
}

func openStore(t *testing.T, clock func() time.Time) *sqlstore.Store {
	t.Helper()
	store, closeStore, err := sqlstore.Open(":memory:", sqlstore.WithClock(clock))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := closeStore(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	return store
}
