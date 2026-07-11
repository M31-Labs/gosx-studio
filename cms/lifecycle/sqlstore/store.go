package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
)

type Clock func() time.Time

type Option func(*Store)

type Store struct {
	db  *sql.DB
	now Clock
}

func New(db *sql.DB, options ...Option) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("sqlstore requires a database")
	}
	store := &Store{db: db, now: time.Now}
	for _, option := range options {
		if option != nil {
			option(store)
		}
	}
	if store.now == nil {
		store.now = time.Now
	}
	return store, nil
}

func WithClock(clock Clock) Option {
	return func(store *Store) {
		store.now = clock
	}
}

func (s *Store) Migrate(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for _, statement := range schemaStatements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SaveRevision(input lifecycle.RevisionInput) (lifecycle.Revision, error) {
	return s.SaveRevisionContext(context.Background(), input)
}

func (s *Store) SaveRevisionContext(ctx context.Context, input lifecycle.RevisionInput) (lifecycle.Revision, error) {
	if strings.TrimSpace(input.ID) == "" {
		input.ID = newID("rev")
	}
	revision, err := lifecycle.NewRevision(input, s.currentTime(input.Created))
	if err != nil {
		return lifecycle.Revision{}, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO lifecycle_revisions (
			id, resource_kind, resource_id, resource_title, action, summary, snapshot, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, revision.ID, revision.ResourceKind, revision.ResourceID, revision.ResourceTitle, revision.Action, revision.Summary, string(revision.Snapshot), formatTime(revision.Created)); err != nil {
		return lifecycle.Revision{}, err
	}
	return lifecycle.CloneRevision(revision), nil
}

func (s *Store) ListRevisions(filter lifecycle.RevisionFilter) []lifecycle.Revision {
	revisions, err := s.ListRevisionsContext(context.Background(), filter)
	if err != nil {
		return nil
	}
	return revisions
}

func (s *Store) ListRevisionsContext(ctx context.Context, filter lifecycle.RevisionFilter) ([]lifecycle.Revision, error) {
	query := `SELECT id, resource_kind, resource_id, resource_title, action, summary, snapshot, created_at FROM lifecycle_revisions`
	where, args := revisionWhere(filter)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRevisions(rows)
}

func (s *Store) RevisionByID(resourceKind, resourceID, revisionID string) (lifecycle.Revision, bool) {
	revision, ok, err := s.RevisionByIDContext(context.Background(), resourceKind, resourceID, revisionID)
	if err != nil {
		return lifecycle.Revision{}, false
	}
	return revision, ok
}

// DeleteRevision physically removes one revision row. It is additive: no
// existing method/interface is affected by its presence, and RevisionStore
// (cms/store.RevisionStore) is unchanged. It exists so
// cms/lifecycle/engine's retention enforcement (spec §2, RetentionPolicy) has
// a real store to trim against -- Store structurally satisfies the engine's
// optional RevisionPruner capability without either package importing the
// other.
func (s *Store) DeleteRevision(resourceKind, resourceID, revisionID string) error {
	return s.DeleteRevisionContext(context.Background(), resourceKind, resourceID, revisionID)
}

func (s *Store) DeleteRevisionContext(ctx context.Context, resourceKind, resourceID, revisionID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM lifecycle_revisions WHERE id = ? AND resource_kind = ? AND resource_id = ?
	`, strings.TrimSpace(revisionID), strings.TrimSpace(resourceKind), strings.TrimSpace(resourceID))
	return err
}

func (s *Store) RevisionByIDContext(ctx context.Context, resourceKind, resourceID, revisionID string) (lifecycle.Revision, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, resource_kind, resource_id, resource_title, action, summary, snapshot, created_at
		FROM lifecycle_revisions
		WHERE id = ? AND resource_kind = ? AND resource_id = ?
	`, strings.TrimSpace(revisionID), strings.TrimSpace(resourceKind), strings.TrimSpace(resourceID))
	revision, err := scanRevision(row)
	if err == sql.ErrNoRows {
		return lifecycle.Revision{}, false, nil
	}
	if err != nil {
		return lifecycle.Revision{}, false, err
	}
	return revision, true, nil
}

func (s *Store) SavePublishDecision(ctx context.Context, input lifecycle.PublishDecisionInput) (lifecycle.PublishDecision, error) {
	decision, err := s.normalizeDecision(input)
	if err != nil {
		return lifecycle.PublishDecision{}, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO lifecycle_publish_decisions (
			id, resource_kind, resource_id, revision_id, status, actor_id, note, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, decision.ID, decision.ResourceKind, decision.ResourceID, decision.RevisionID, string(decision.Status), decision.ActorID, decision.Note, formatTime(decision.Created)); err != nil {
		return lifecycle.PublishDecision{}, err
	}
	return decision, nil
}

func (s *Store) LatestPublishDecision(ctx context.Context, resourceKind, resourceID string) (lifecycle.PublishDecision, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, resource_kind, resource_id, revision_id, status, actor_id, note, created_at
		FROM lifecycle_publish_decisions
		WHERE resource_kind = ? AND resource_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, strings.TrimSpace(resourceKind), strings.TrimSpace(resourceID))
	decision, err := scanDecision(row)
	if err == sql.ErrNoRows {
		return lifecycle.PublishDecision{}, false, nil
	}
	if err != nil {
		return lifecycle.PublishDecision{}, false, err
	}
	return decision, true, nil
}

func (s *Store) ListPublishDecisions(ctx context.Context, filter lifecycle.LedgerFilter) ([]lifecycle.PublishDecision, error) {
	query := `SELECT id, resource_kind, resource_id, revision_id, status, actor_id, note, created_at FROM lifecycle_publish_decisions`
	where, args := ledgerWhere(filter, false)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	query, args = applyLimit(query, args, filter.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []lifecycle.PublishDecision{}
	for rows.Next() {
		decision, err := scanDecision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, decision)
	}
	return out, rows.Err()
}

func (s *Store) SavePublishSchedule(ctx context.Context, input lifecycle.PublishScheduleInput) (lifecycle.PublishSchedule, error) {
	schedule, err := s.normalizeSchedule(input)
	if err != nil {
		return lifecycle.PublishSchedule{}, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO lifecycle_publish_schedules (
			id, resource_kind, resource_id, revision_id, action, state, due_at, timezone,
			actor_id, note, claim_token, claimed_at, completed_at, cancelled_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, schedule.ID, schedule.ResourceKind, schedule.ResourceID, schedule.RevisionID, string(schedule.Action), string(schedule.State),
		formatTime(schedule.DueAt), schedule.Timezone, schedule.ActorID, schedule.Note, schedule.ClaimToken,
		formatTimePtr(schedule.ClaimedAt), formatTimePtr(schedule.CompletedAt), formatTimePtr(schedule.CancelledAt),
		formatTime(schedule.Created), formatTime(schedule.Updated)); err != nil {
		return lifecycle.PublishSchedule{}, err
	}
	return schedule, nil
}

func (s *Store) ListPublishSchedules(ctx context.Context, filter lifecycle.ScheduleFilter) ([]lifecycle.PublishSchedule, error) {
	query := `SELECT id, resource_kind, resource_id, revision_id, action, state, due_at, timezone, actor_id, note, claim_token, claimed_at, completed_at, cancelled_at, created_at, updated_at FROM lifecycle_publish_schedules`
	where, args := scheduleWhere(filter)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY due_at ASC, created_at ASC, id ASC"
	query, args = applyLimit(query, args, filter.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSchedules(rows)
}

func (s *Store) CancelPublishSchedule(ctx context.Context, id, actorID, note string) (lifecycle.PublishSchedule, bool, error) {
	now := s.currentTime(time.Time{})
	result, err := s.db.ExecContext(ctx, `
		UPDATE lifecycle_publish_schedules
		SET state = ?, actor_id = ?, note = ?, cancelled_at = ?, updated_at = ?
		WHERE id = ? AND state NOT IN (?, ?)
	`, string(lifecycle.ScheduleCancelled), strings.TrimSpace(actorID), strings.TrimSpace(note), formatTime(now), formatTime(now),
		strings.TrimSpace(id), string(lifecycle.ScheduleCompleted), string(lifecycle.ScheduleCancelled))
	if err != nil {
		return lifecycle.PublishSchedule{}, false, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return lifecycle.PublishSchedule{}, false, nil
	}
	return s.scheduleByID(ctx, id)
}

func (s *Store) ClaimDueSchedules(ctx context.Context, dueAt time.Time, limit int, claimToken string) ([]lifecycle.PublishSchedule, error) {
	if dueAt.IsZero() {
		dueAt = s.currentTime(time.Time{})
	}
	if limit <= 0 {
		limit = 1
	}
	claimToken = strings.TrimSpace(claimToken)
	if claimToken == "" {
		claimToken = newID("claim")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM lifecycle_publish_schedules
		WHERE state = ? AND due_at <= ?
		ORDER BY due_at ASC, created_at ASC, id ASC
		LIMIT ?
	`, string(lifecycle.SchedulePending), formatTime(dueAt), limit)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	now := s.currentTime(time.Time{})
	claimed := []lifecycle.PublishSchedule{}
	for _, id := range ids {
		result, err := tx.ExecContext(ctx, `
			UPDATE lifecycle_publish_schedules
			SET state = ?, claim_token = ?, claimed_at = ?, updated_at = ?
			WHERE id = ? AND state = ?
		`, string(lifecycle.ScheduleClaimed), claimToken, formatTime(now), formatTime(now), id, string(lifecycle.SchedulePending))
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		if affected == 0 {
			continue
		}
		row := tx.QueryRowContext(ctx, `
			SELECT id, resource_kind, resource_id, revision_id, action, state, due_at, timezone, actor_id, note, claim_token, claimed_at, completed_at, cancelled_at, created_at, updated_at
			FROM lifecycle_publish_schedules
			WHERE id = ?
		`, id)
		schedule, err := scanSchedule(row)
		if err != nil {
			return nil, err
		}
		claimed = append(claimed, schedule)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s *Store) CompletePublishSchedule(ctx context.Context, id string) (lifecycle.PublishSchedule, bool, error) {
	now := s.currentTime(time.Time{})
	result, err := s.db.ExecContext(ctx, `
		UPDATE lifecycle_publish_schedules
		SET state = ?, completed_at = ?, updated_at = ?
		WHERE id = ? AND state = ?
	`, string(lifecycle.ScheduleCompleted), formatTime(now), formatTime(now), strings.TrimSpace(id), string(lifecycle.ScheduleClaimed))
	if err != nil {
		return lifecycle.PublishSchedule{}, false, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return lifecycle.PublishSchedule{}, false, nil
	}
	return s.scheduleByID(ctx, id)
}

func (s *Store) SavePublishNote(ctx context.Context, input lifecycle.PublishNoteInput) (lifecycle.PublishNote, error) {
	note, err := s.normalizeNote(input)
	if err != nil {
		return lifecycle.PublishNote{}, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO lifecycle_publish_notes (
			id, resource_kind, resource_id, revision_id, actor_id, body, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, note.ID, note.ResourceKind, note.ResourceID, note.RevisionID, note.ActorID, note.Body, formatTime(note.Created)); err != nil {
		return lifecycle.PublishNote{}, err
	}
	return note, nil
}

func (s *Store) ListPublishNotes(ctx context.Context, filter lifecycle.LedgerFilter) ([]lifecycle.PublishNote, error) {
	query := `SELECT id, resource_kind, resource_id, revision_id, actor_id, body, created_at FROM lifecycle_publish_notes`
	where, args := ledgerWhere(filter, false)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	query, args = applyLimit(query, args, filter.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []lifecycle.PublishNote{}
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, note)
	}
	return out, rows.Err()
}

func (s *Store) SaveAuditEvent(ctx context.Context, input lifecycle.AuditEventInput) (lifecycle.AuditEvent, error) {
	event, metadataJSON, err := s.normalizeAuditEvent(input)
	if err != nil {
		return lifecycle.AuditEvent{}, err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO lifecycle_audit_events (
			id, resource_kind, resource_id, revision_id, action, actor_id, summary, metadata_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, event.ID, event.ResourceKind, event.ResourceID, event.RevisionID, event.Action, event.ActorID, event.Summary, metadataJSON, formatTime(event.Created)); err != nil {
		return lifecycle.AuditEvent{}, err
	}
	return event, nil
}

func (s *Store) ListAuditEvents(ctx context.Context, filter lifecycle.LedgerFilter) ([]lifecycle.AuditEvent, error) {
	query := `SELECT id, resource_kind, resource_id, revision_id, action, actor_id, summary, metadata_json, created_at FROM lifecycle_audit_events`
	where, args := ledgerWhere(filter, true)
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	query, args = applyLimit(query, args, filter.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []lifecycle.AuditEvent{}
	for rows.Next() {
		event, err := scanAuditEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func (s *Store) scheduleByID(ctx context.Context, id string) (lifecycle.PublishSchedule, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, resource_kind, resource_id, revision_id, action, state, due_at, timezone, actor_id, note, claim_token, claimed_at, completed_at, cancelled_at, created_at, updated_at
		FROM lifecycle_publish_schedules
		WHERE id = ?
	`, strings.TrimSpace(id))
	schedule, err := scanSchedule(row)
	if err == sql.ErrNoRows {
		return lifecycle.PublishSchedule{}, false, nil
	}
	if err != nil {
		return lifecycle.PublishSchedule{}, false, err
	}
	return schedule, true, nil
}

func (s *Store) normalizeDecision(input lifecycle.PublishDecisionInput) (lifecycle.PublishDecision, error) {
	resourceKind, resourceID, err := resource(input.ResourceKind, input.ResourceID)
	if err != nil {
		return lifecycle.PublishDecision{}, err
	}
	status := input.Status
	if status == "" {
		status = lifecycle.DecisionPending
	}
	if !validDecisionStatus(status) {
		return lifecycle.PublishDecision{}, fmt.Errorf("unsupported publish decision status %q", status)
	}
	created := s.currentTime(input.Created)
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = newID("decision")
	}
	return lifecycle.PublishDecision{
		ID:           id,
		ResourceKind: resourceKind,
		ResourceID:   resourceID,
		RevisionID:   strings.TrimSpace(input.RevisionID),
		Status:       status,
		ActorID:      strings.TrimSpace(input.ActorID),
		Note:         strings.TrimSpace(input.Note),
		Created:      created,
	}, nil
}

func (s *Store) normalizeSchedule(input lifecycle.PublishScheduleInput) (lifecycle.PublishSchedule, error) {
	resourceKind, resourceID, err := resource(input.ResourceKind, input.ResourceID)
	if err != nil {
		return lifecycle.PublishSchedule{}, err
	}
	if input.DueAt.IsZero() {
		return lifecycle.PublishSchedule{}, fmt.Errorf("publish schedule requires a due time")
	}
	action := input.Action
	if action == "" {
		action = lifecycle.ScheduleActionPublish
	}
	if !validScheduleAction(action) {
		return lifecycle.PublishSchedule{}, fmt.Errorf("unsupported publish schedule action %q", action)
	}
	state := input.State
	if state == "" {
		state = lifecycle.SchedulePending
	}
	if !validScheduleState(state) {
		return lifecycle.PublishSchedule{}, fmt.Errorf("unsupported publish schedule state %q", state)
	}
	created := s.currentTime(input.Created)
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = newID("schedule")
	}
	return lifecycle.PublishSchedule{
		ID:           id,
		ResourceKind: resourceKind,
		ResourceID:   resourceID,
		RevisionID:   strings.TrimSpace(input.RevisionID),
		Action:       action,
		State:        state,
		DueAt:        input.DueAt.UTC(),
		Timezone:     strings.TrimSpace(input.Timezone),
		ActorID:      strings.TrimSpace(input.ActorID),
		Note:         strings.TrimSpace(input.Note),
		Created:      created,
		Updated:      created,
	}, nil
}

func (s *Store) normalizeNote(input lifecycle.PublishNoteInput) (lifecycle.PublishNote, error) {
	resourceKind, resourceID, err := resource(input.ResourceKind, input.ResourceID)
	if err != nil {
		return lifecycle.PublishNote{}, err
	}
	body := strings.TrimSpace(input.Body)
	if body == "" {
		return lifecycle.PublishNote{}, fmt.Errorf("publish note body is required")
	}
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = newID("note")
	}
	return lifecycle.PublishNote{
		ID:           id,
		ResourceKind: resourceKind,
		ResourceID:   resourceID,
		RevisionID:   strings.TrimSpace(input.RevisionID),
		ActorID:      strings.TrimSpace(input.ActorID),
		Body:         body,
		Created:      s.currentTime(input.Created),
	}, nil
}

func (s *Store) normalizeAuditEvent(input lifecycle.AuditEventInput) (lifecycle.AuditEvent, string, error) {
	action := strings.TrimSpace(input.Action)
	if action == "" {
		return lifecycle.AuditEvent{}, "", fmt.Errorf("audit event action is required")
	}
	id := strings.TrimSpace(input.ID)
	if id == "" {
		id = newID("audit")
	}
	metadata := cloneMetadata(input.Metadata)
	data, err := json.Marshal(metadata)
	if err != nil {
		return lifecycle.AuditEvent{}, "", err
	}
	event := lifecycle.AuditEvent{
		ID:           id,
		ResourceKind: strings.TrimSpace(input.ResourceKind),
		ResourceID:   strings.TrimSpace(input.ResourceID),
		RevisionID:   strings.TrimSpace(input.RevisionID),
		Action:       action,
		ActorID:      strings.TrimSpace(input.ActorID),
		Summary:      strings.TrimSpace(input.Summary),
		Metadata:     metadata,
		Created:      s.currentTime(input.Created),
	}
	return event, string(data), nil
}

func (s *Store) currentTime(input time.Time) time.Time {
	if !input.IsZero() {
		return input.UTC()
	}
	return s.now().UTC()
}

func resource(kind, id string) (string, string, error) {
	kind = strings.TrimSpace(kind)
	id = strings.TrimSpace(id)
	if kind == "" || id == "" {
		return "", "", fmt.Errorf("resource kind and id are required")
	}
	return kind, id, nil
}

func validDecisionStatus(status lifecycle.DecisionStatus) bool {
	switch status {
	case lifecycle.DecisionPending, lifecycle.DecisionApproved, lifecycle.DecisionRejected, lifecycle.DecisionChangesRequested:
		return true
	default:
		return false
	}
}

func validScheduleAction(action lifecycle.ScheduleAction) bool {
	switch action {
	case lifecycle.ScheduleActionPublish, lifecycle.ScheduleActionUnpublish:
		return true
	default:
		return false
	}
}

func validScheduleState(state lifecycle.ScheduleState) bool {
	switch state {
	case lifecycle.SchedulePending, lifecycle.ScheduleClaimed, lifecycle.ScheduleCompleted, lifecycle.ScheduleCancelled:
		return true
	default:
		return false
	}
}

func revisionWhere(filter lifecycle.RevisionFilter) ([]string, []any) {
	where := []string{}
	args := []any{}
	if value := strings.TrimSpace(filter.ResourceKind); value != "" {
		where = append(where, "resource_kind = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.ResourceID); value != "" {
		where = append(where, "resource_id = ?")
		args = append(args, value)
	}
	return where, args
}

func ledgerWhere(filter lifecycle.LedgerFilter, includeAction bool) ([]string, []any) {
	where := []string{}
	args := []any{}
	if value := strings.TrimSpace(filter.ResourceKind); value != "" {
		where = append(where, "resource_kind = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.ResourceID); value != "" {
		where = append(where, "resource_id = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.RevisionID); value != "" {
		where = append(where, "revision_id = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.ActorID); value != "" {
		where = append(where, "actor_id = ?")
		args = append(args, value)
	}
	if includeAction {
		if value := strings.TrimSpace(filter.Action); value != "" {
			where = append(where, "action = ?")
			args = append(args, value)
		}
	}
	return where, args
}

func scheduleWhere(filter lifecycle.ScheduleFilter) ([]string, []any) {
	where := []string{}
	args := []any{}
	if value := strings.TrimSpace(filter.ResourceKind); value != "" {
		where = append(where, "resource_kind = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.ResourceID); value != "" {
		where = append(where, "resource_id = ?")
		args = append(args, value)
	}
	if value := strings.TrimSpace(filter.RevisionID); value != "" {
		where = append(where, "revision_id = ?")
		args = append(args, value)
	}
	if filter.State != "" {
		where = append(where, "state = ?")
		args = append(args, string(filter.State))
	}
	if !filter.DueBefore.IsZero() {
		where = append(where, "due_at <= ?")
		args = append(args, formatTime(filter.DueBefore))
	}
	return where, args
}

func applyLimit(query string, args []any, limit int) (string, []any) {
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	return query, args
}

func scanRevisions(rows *sql.Rows) ([]lifecycle.Revision, error) {
	out := []lifecycle.Revision{}
	for rows.Next() {
		revision, err := scanRevision(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, revision)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRevision(row rowScanner) (lifecycle.Revision, error) {
	var revision lifecycle.Revision
	var snapshot string
	var created string
	if err := row.Scan(&revision.ID, &revision.ResourceKind, &revision.ResourceID, &revision.ResourceTitle, &revision.Action, &revision.Summary, &snapshot, &created); err != nil {
		return lifecycle.Revision{}, err
	}
	createdAt, err := parseTime(created)
	if err != nil {
		return lifecycle.Revision{}, err
	}
	revision.Created = createdAt
	revision.Snapshot = append([]byte(nil), snapshot...)
	return revision, nil
}

func scanDecision(row rowScanner) (lifecycle.PublishDecision, error) {
	var decision lifecycle.PublishDecision
	var status string
	var created string
	if err := row.Scan(&decision.ID, &decision.ResourceKind, &decision.ResourceID, &decision.RevisionID, &status, &decision.ActorID, &decision.Note, &created); err != nil {
		return lifecycle.PublishDecision{}, err
	}
	createdAt, err := parseTime(created)
	if err != nil {
		return lifecycle.PublishDecision{}, err
	}
	decision.Status = lifecycle.DecisionStatus(status)
	decision.Created = createdAt
	return decision, nil
}

func scanSchedules(rows *sql.Rows) ([]lifecycle.PublishSchedule, error) {
	out := []lifecycle.PublishSchedule{}
	for rows.Next() {
		schedule, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, schedule)
	}
	return out, rows.Err()
}

func scanSchedule(row rowScanner) (lifecycle.PublishSchedule, error) {
	var schedule lifecycle.PublishSchedule
	var action string
	var state string
	var dueAt string
	var claimedAt, completedAt, cancelledAt sql.NullString
	var created string
	var updated string
	if err := row.Scan(
		&schedule.ID, &schedule.ResourceKind, &schedule.ResourceID, &schedule.RevisionID, &action, &state, &dueAt, &schedule.Timezone,
		&schedule.ActorID, &schedule.Note, &schedule.ClaimToken, &claimedAt, &completedAt, &cancelledAt, &created, &updated,
	); err != nil {
		return lifecycle.PublishSchedule{}, err
	}
	parsedDueAt, err := parseTime(dueAt)
	if err != nil {
		return lifecycle.PublishSchedule{}, err
	}
	createdAt, err := parseTime(created)
	if err != nil {
		return lifecycle.PublishSchedule{}, err
	}
	updatedAt, err := parseTime(updated)
	if err != nil {
		return lifecycle.PublishSchedule{}, err
	}
	schedule.Action = lifecycle.ScheduleAction(action)
	schedule.State = lifecycle.ScheduleState(state)
	schedule.DueAt = parsedDueAt
	schedule.Created = createdAt
	schedule.Updated = updatedAt
	schedule.ClaimedAt = parseTimePtr(claimedAt)
	schedule.CompletedAt = parseTimePtr(completedAt)
	schedule.CancelledAt = parseTimePtr(cancelledAt)
	return schedule, nil
}

func scanNote(row rowScanner) (lifecycle.PublishNote, error) {
	var note lifecycle.PublishNote
	var created string
	if err := row.Scan(&note.ID, &note.ResourceKind, &note.ResourceID, &note.RevisionID, &note.ActorID, &note.Body, &created); err != nil {
		return lifecycle.PublishNote{}, err
	}
	createdAt, err := parseTime(created)
	if err != nil {
		return lifecycle.PublishNote{}, err
	}
	note.Created = createdAt
	return note, nil
}

func scanAuditEvent(row rowScanner) (lifecycle.AuditEvent, error) {
	var event lifecycle.AuditEvent
	var metadataJSON string
	var created string
	if err := row.Scan(&event.ID, &event.ResourceKind, &event.ResourceID, &event.RevisionID, &event.Action, &event.ActorID, &event.Summary, &metadataJSON, &created); err != nil {
		return lifecycle.AuditEvent{}, err
	}
	createdAt, err := parseTime(created)
	if err != nil {
		return lifecycle.AuditEvent{}, err
	}
	event.Created = createdAt
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &event.Metadata); err != nil {
			return lifecycle.AuditEvent{}, err
		}
	}
	return event, nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatTimePtr(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return formatTime(*value)
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}

func parseTimePtr(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed, err := parseTime(value.String)
	if err != nil || parsed.IsZero() {
		return nil
	}
	return &parsed
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func newID(prefix string) string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return strings.TrimSpace(prefix) + "_" + hex.EncodeToString(raw[:])
}

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS lifecycle_revisions (
		id TEXT PRIMARY KEY,
		resource_kind TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		resource_title TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		summary TEXT NOT NULL DEFAULT '',
		snapshot TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS lifecycle_revisions_resource_idx ON lifecycle_revisions (resource_kind, resource_id, created_at DESC)`,
	`CREATE TABLE IF NOT EXISTS lifecycle_publish_decisions (
		id TEXT PRIMARY KEY,
		resource_kind TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		revision_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		actor_id TEXT NOT NULL DEFAULT '',
		note TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS lifecycle_publish_decisions_resource_idx ON lifecycle_publish_decisions (resource_kind, resource_id, created_at DESC)`,
	`CREATE TABLE IF NOT EXISTS lifecycle_publish_schedules (
		id TEXT PRIMARY KEY,
		resource_kind TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		revision_id TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		state TEXT NOT NULL,
		due_at TEXT NOT NULL,
		timezone TEXT NOT NULL DEFAULT '',
		actor_id TEXT NOT NULL DEFAULT '',
		note TEXT NOT NULL DEFAULT '',
		claim_token TEXT NOT NULL DEFAULT '',
		claimed_at TEXT,
		completed_at TEXT,
		cancelled_at TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS lifecycle_publish_schedules_due_idx ON lifecycle_publish_schedules (state, due_at, created_at)`,
	`CREATE TABLE IF NOT EXISTS lifecycle_publish_notes (
		id TEXT PRIMARY KEY,
		resource_kind TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		revision_id TEXT NOT NULL DEFAULT '',
		actor_id TEXT NOT NULL DEFAULT '',
		body TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS lifecycle_publish_notes_resource_idx ON lifecycle_publish_notes (resource_kind, resource_id, created_at DESC)`,
	`CREATE TABLE IF NOT EXISTS lifecycle_audit_events (
		id TEXT PRIMARY KEY,
		resource_kind TEXT NOT NULL DEFAULT '',
		resource_id TEXT NOT NULL DEFAULT '',
		revision_id TEXT NOT NULL DEFAULT '',
		action TEXT NOT NULL,
		actor_id TEXT NOT NULL DEFAULT '',
		summary TEXT NOT NULL DEFAULT '',
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS lifecycle_audit_events_resource_idx ON lifecycle_audit_events (resource_kind, resource_id, created_at DESC)`,
}
