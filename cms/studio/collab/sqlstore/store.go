package sqlstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/studio/collab"
)

const schemaVersion = 1

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("studio collaboration database path is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// A single writer connection makes transaction ordering explicit and also
	// keeps :memory: databases coherent in tests.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) DB() *sql.DB  { return s.db }

func (s *Store) SeedHeads(ctx context.Context, resource collab.ResourceKey, heads []collab.SeedHead) error {
	resource = resource.Normalize()
	if err := resource.Validate(); err != nil {
		return err
	}
	if len(heads) == 0 {
		return nil
	}
	normalized := make([]collab.SeedHead, 0, len(heads))
	for _, head := range heads {
		head.Target = head.Target.Normalize()
		kind := authoring.OperationSetField
		if head.Target.Property != "" {
			kind = authoring.OperationSetStyle
		}
		request := authoring.OperationRequest{ID: "seed-validation", Kind: kind, Target: head.Target}
		if protocolErr := (collab.OperationSubmit{Request: request}).Validate(); protocolErr != nil {
			return protocolErr
		}
		normalized = append(normalized, head)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	key := resource.StableKey()
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO studio_resources(resource_key,tenant_id,project_id,resource_kind,resource_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, key, resource.TenantID, resource.ProjectID, resource.Kind, resource.ID, timestamp(now), timestamp(now)); err != nil {
		return err
	}
	for _, head := range normalized {
		kind := authoring.OperationSetField
		if head.Target.Property != "" {
			kind = authoring.OperationSetStyle
		}
		targetKey := head.Target.Key(kind)
		var existingHead, existingValue string
		var present int
		scanErr := tx.QueryRowContext(ctx, `SELECT head_operation_id,value_present,value FROM studio_field_heads WHERE resource_key=? AND target_key=?`, key, targetKey).Scan(&existingHead, &present, &existingValue)
		if scanErr == nil {
			if present != boolInt(head.Value.Present) || existingValue != head.Value.Value {
				return fmt.Errorf("collaboration seed conflict for %s", head.Target.String())
			}
			continue
		}
		if !errors.Is(scanErr, sql.ErrNoRows) {
			return scanErr
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO studio_field_heads(resource_key,target_key,head_operation_id,sequence,value_present,value,updated_at) VALUES(?,?,?,?,?,?,?)`, key, targetKey, "", 0, boolInt(head.Value.Present), head.Value.Value, timestamp(now)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS studio_schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS studio_resources (
			resource_key TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, project_id TEXT NOT NULL,
			resource_kind TEXT NOT NULL, resource_id TEXT NOT NULL, accepted_sequence INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS studio_field_heads (
			resource_key TEXT NOT NULL, target_key TEXT NOT NULL, head_operation_id TEXT NOT NULL,
			sequence INTEGER NOT NULL, value_present INTEGER NOT NULL, value TEXT NOT NULL,
			updated_at TEXT NOT NULL, PRIMARY KEY(resource_key,target_key),
			FOREIGN KEY(resource_key) REFERENCES studio_resources(resource_key) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS studio_operation_attempts (
			attempt_id INTEGER PRIMARY KEY AUTOINCREMENT, resource_key TEXT NOT NULL,
			operation_id TEXT NOT NULL, actor_id TEXT NOT NULL, accepted INTEGER NOT NULL,
			accepted_sequence INTEGER, error_code TEXT NOT NULL DEFAULT '', request_json BLOB NOT NULL,
			record_json BLOB, current_head TEXT NOT NULL DEFAULT '', current_value_present INTEGER NOT NULL DEFAULT 0,
			current_value TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL,
			FOREIGN KEY(resource_key) REFERENCES studio_resources(resource_key) ON DELETE CASCADE)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS studio_operation_accepted_id ON studio_operation_attempts(resource_key,operation_id) WHERE accepted=1`,
		`CREATE INDEX IF NOT EXISTS studio_operation_sequence ON studio_operation_attempts(resource_key,accepted_sequence) WHERE accepted=1`,
		`CREATE TABLE IF NOT EXISTS studio_projection_outbox (
			outbox_id INTEGER PRIMARY KEY AUTOINCREMENT, resource_key TEXT NOT NULL,
			accepted_sequence INTEGER NOT NULL, operation_id TEXT NOT NULL, payload BLOB NOT NULL,
			created_at TEXT NOT NULL, projected_at TEXT,
			UNIQUE(resource_key,accepted_sequence),
			FOREIGN KEY(resource_key) REFERENCES studio_resources(resource_key) ON DELETE CASCADE)`,
		`CREATE TABLE IF NOT EXISTS studio_field_head_repairs (
			repair_id INTEGER PRIMARY KEY AUTOINCREMENT, resource_key TEXT NOT NULL,
			target_key TEXT NOT NULL, target_json BLOB NOT NULL, actor_id TEXT NOT NULL,
			reason TEXT NOT NULL, old_head TEXT NOT NULL, old_value_present INTEGER NOT NULL,
			old_value TEXT NOT NULL, new_head TEXT NOT NULL, new_value_present INTEGER NOT NULL,
			new_value TEXT NOT NULL, created_at TEXT NOT NULL,
			FOREIGN KEY(resource_key) REFERENCES studio_resources(resource_key) ON DELETE CASCADE)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate collaboration store: %w", err)
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO studio_schema_migrations(version,applied_at) VALUES(?,?)`, schemaVersion, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) Apply(ctx context.Context, command collab.ApplyCommand) (collab.OperationAck, *collab.ProtocolError) {
	resource := command.Resource.Normalize()
	principal := command.Principal.Clone()
	request := command.Request.Normalize()
	now := command.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := resource.Validate(); err != nil {
		return collab.OperationAck{}, collab.NewProtocolError(collab.ErrorInvalidRequest, err.Error())
	}
	if err := principal.Validate(); err != nil {
		return collab.OperationAck{}, collab.NewProtocolError(collab.ErrorForbidden, err.Error())
	}
	if protocolErr := (collab.OperationSubmit{Request: request}).Validate(); protocolErr != nil {
		return collab.OperationAck{}, protocolErr
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	defer tx.Rollback()
	key := resource.StableKey()
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO studio_resources(resource_key,tenant_id,project_id,resource_kind,resource_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, key, resource.TenantID, resource.ProjectID, resource.Kind, resource.ID, timestamp(now), timestamp(now)); err != nil {
		return collab.OperationAck{}, storeError(err)
	}

	if ack, existing, altered, err := existingAccepted(ctx, tx, key, principal.ActorID, request); err != nil {
		return collab.OperationAck{}, storeError(err)
	} else if existing {
		if altered {
			protocolErr := collab.NewProtocolError(collab.ErrorOperationIDReuse, "operation id was already accepted with a different payload")
			protocolErr.OperationID = request.ID
			if err := insertRejected(ctx, tx, key, principal.ActorID, request, protocolErr, authoring.OperationValue{}, now); err != nil {
				return collab.OperationAck{}, storeError(err)
			}
			if err := tx.Commit(); err != nil {
				return collab.OperationAck{}, storeError(err)
			}
			return collab.OperationAck{}, protocolErr
		}
		ack.Idempotent = true
		if err := tx.Commit(); err != nil {
			return collab.OperationAck{}, storeError(err)
		}
		return ack, nil
	}

	effective, history, desiredHistoryValue, protocolErr := resolveHistory(ctx, tx, key, principal, request)
	if protocolErr != nil {
		if protocolErr.Code == collab.ErrorStoreUnavailable {
			return collab.OperationAck{}, protocolErr
		}
		protocolErr.OperationID = request.ID
		if err := insertRejected(ctx, tx, key, principal.ActorID, request, protocolErr, authoring.OperationValue{}, now); err != nil {
			return collab.OperationAck{}, storeError(err)
		}
		if err := tx.Commit(); err != nil {
			return collab.OperationAck{}, storeError(err)
		}
		return collab.OperationAck{}, protocolErr
	}
	requiredCapability := collab.CapabilityAuthor
	switch effective.Kind {
	case authoring.OperationSetStyle, authoring.OperationResetStyle, authoring.OperationSetInteraction, authoring.OperationRemoveInteraction:
		// Style and interaction operations are both visual/behavioral
		// treatment, not content or structural configuration -- they share
		// style's CapabilityDesign gate. Instance ops (shared field,
		// override, detach/restore) and flow ops (field, action) stay
		// CapabilityAuthor, the same gate SetField already uses, because
		// they edit content/structure, not visual treatment.
		requiredCapability = collab.CapabilityDesign
	}
	if !principal.Can(requiredCapability) {
		protocolErr := collab.NewProtocolError(collab.ErrorForbidden, "principal lacks the capability required for this operation")
		protocolErr.OperationID = request.ID
		if err := insertRejected(ctx, tx, key, principal.ActorID, request, protocolErr, authoring.OperationValue{}, now); err != nil {
			return collab.OperationAck{}, storeError(err)
		}
		if err := tx.Commit(); err != nil {
			return collab.OperationAck{}, storeError(err)
		}
		return collab.OperationAck{}, protocolErr
	}
	targetKey := effective.Target.Key(effective.Kind)
	currentHead, currentValue, err := readHead(ctx, tx, key, targetKey)
	if err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	if effective.ExpectedTargetHead != currentHead {
		protocolErr := collab.NewProtocolError(collab.ErrorStaleField, "field head changed; recover using the returned value and head")
		protocolErr.OperationID, protocolErr.CurrentHead = request.ID, currentHead
		valueCopy := currentValue
		protocolErr.CurrentValue = &valueCopy
		submitted := authoring.PresentValue(effective.Value)
		if authoring.ClearsTarget(effective.Kind) {
			submitted = authoring.OperationValue{}
		}
		if desiredHistoryValue != nil {
			submitted = *desiredHistoryValue
		}
		protocolErr.SubmittedValue = &submitted
		if err := insertRejected(ctx, tx, key, principal.ActorID, request, protocolErr, currentValue, now); err != nil {
			return collab.OperationAck{}, storeError(err)
		}
		if err := tx.Commit(); err != nil {
			return collab.OperationAck{}, storeError(err)
		}
		return collab.OperationAck{}, protocolErr
	}

	var sequence uint64
	if err := tx.QueryRowContext(ctx, `SELECT accepted_sequence FROM studio_resources WHERE resource_key=?`, key).Scan(&sequence); err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	sequence++
	after := authoring.PresentValue(effective.Value)
	if authoring.ClearsTarget(effective.Kind) {
		after = authoring.OperationValue{}
	}
	if desiredHistoryValue != nil {
		after = *desiredHistoryValue
	}
	record := authoring.OperationRecord{
		SchemaVersion: authoring.OperationSchemaVersion, ID: request.ID,
		ActorID: principal.ActorID, ActorLabel: principal.DisplayName, Kind: request.Kind,
		Target: effective.Target, TargetKey: targetKey, Before: currentValue, After: after,
		DocumentRevision: sequence, PreviousTargetHead: currentHead, TargetHead: request.ID, Created: now,
	}.Normalize()
	if request.Kind == authoring.OperationUndo {
		record.UndoOf = history.ID
	}
	if request.Kind == authoring.OperationRedo {
		record.RedoOf = history.ID
	}
	requestJSON, _ := json.Marshal(request)
	recordJSON, _ := json.Marshal(record)
	if _, err := tx.ExecContext(ctx, `INSERT INTO studio_operation_attempts(resource_key,operation_id,actor_id,accepted,accepted_sequence,request_json,record_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, key, request.ID, principal.ActorID, 1, sequence, requestJSON, recordJSON, timestamp(now)); err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO studio_field_heads(resource_key,target_key,head_operation_id,sequence,value_present,value,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(resource_key,target_key) DO UPDATE SET head_operation_id=excluded.head_operation_id,sequence=excluded.sequence,value_present=excluded.value_present,value=excluded.value,updated_at=excluded.updated_at`, key, targetKey, request.ID, sequence, boolInt(after.Present), after.Value, timestamp(now)); err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE studio_resources SET accepted_sequence=?,updated_at=? WHERE resource_key=?`, sequence, timestamp(now), key); err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO studio_projection_outbox(resource_key,accepted_sequence,operation_id,payload,created_at) VALUES(?,?,?,?,?)`, key, sequence, request.ID, recordJSON, timestamp(now)); err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	if err := tx.Commit(); err != nil {
		return collab.OperationAck{}, storeError(err)
	}
	return collab.OperationAck{Record: record, Sequence: sequence}, nil
}

func existingAccepted(ctx context.Context, tx *sql.Tx, key, actorID string, request authoring.OperationRequest) (collab.OperationAck, bool, bool, error) {
	var requestJSON, recordJSON []byte
	var sequence uint64
	var acceptedActor string
	err := tx.QueryRowContext(ctx, `SELECT request_json,record_json,accepted_sequence FROM studio_operation_attempts WHERE resource_key=? AND operation_id=? AND accepted=1`, key, request.ID).Scan(&requestJSON, &recordJSON, &sequence)
	if errors.Is(err, sql.ErrNoRows) {
		return collab.OperationAck{}, false, false, nil
	}
	if err != nil {
		return collab.OperationAck{}, false, false, err
	}
	var oldRequest authoring.OperationRequest
	var record authoring.OperationRecord
	if err := json.Unmarshal(requestJSON, &oldRequest); err != nil {
		return collab.OperationAck{}, false, false, err
	}
	if err := json.Unmarshal(recordJSON, &record); err != nil {
		return collab.OperationAck{}, false, false, err
	}
	acceptedActor = record.ActorID
	return collab.OperationAck{Record: record, Sequence: sequence}, true, acceptedActor != actorID || !authoring.RequestsEqual(oldRequest, request), nil
}

func resolveHistory(ctx context.Context, tx *sql.Tx, key string, principal collab.Principal, request authoring.OperationRequest) (authoring.OperationRequest, authoring.OperationRecord, *authoring.OperationValue, *collab.ProtocolError) {
	if request.Kind != authoring.OperationUndo && request.Kind != authoring.OperationRedo {
		return request, authoring.OperationRecord{}, nil, nil
	}
	loadRecord := func(id string) (authoring.OperationRecord, error) {
		var raw []byte
		err := tx.QueryRowContext(ctx, `SELECT record_json FROM studio_operation_attempts WHERE resource_key=? AND operation_id=? AND accepted=1`, key, id).Scan(&raw)
		if err != nil {
			return authoring.OperationRecord{}, err
		}
		var record authoring.OperationRecord
		if err := json.Unmarshal(raw, &record); err != nil {
			return authoring.OperationRecord{}, err
		}
		return record, nil
	}
	history, err := loadRecord(request.HistoryOperationID)
	if errors.Is(err, sql.ErrNoRows) {
		return authoring.OperationRequest{}, authoring.OperationRecord{}, nil, collab.NewProtocolError(collab.ErrorHistoryNotFound, "history operation was not found")
	}
	if err != nil {
		return authoring.OperationRequest{}, authoring.OperationRecord{}, nil, storeError(err)
	}
	if history.ActorID != principal.ActorID {
		return authoring.OperationRequest{}, history, nil, collab.NewProtocolError(collab.ErrorHistoryActor, "an actor may only undo or redo their own operation")
	}
	requiredTargetHead := history.TargetHead
	if request.Kind == authoring.OperationRedo {
		if history.Kind != authoring.OperationUndo || history.UndoOf == "" {
			return authoring.OperationRequest{}, history, nil, collab.NewProtocolError(collab.ErrorInvalidRequest, "redo must reference an accepted undo operation")
		}
		original, err := loadRecord(history.UndoOf)
		if errors.Is(err, sql.ErrNoRows) {
			return authoring.OperationRequest{}, history, nil, collab.NewProtocolError(collab.ErrorHistoryNotFound, "original operation for redo was not found")
		}
		if err != nil {
			return authoring.OperationRequest{}, history, nil, storeError(err)
		}
		if original.ActorID != principal.ActorID {
			return authoring.OperationRequest{}, original, nil, collab.NewProtocolError(collab.ErrorHistoryActor, "an actor may only redo their own operation")
		}
		history = original
	}
	effective := request
	effective.Target = history.Target
	// Undo/redo is the inverse of a specific current head, not permission to
	// replay an arbitrary historical value over intervening same-field work.
	effective.ExpectedTargetHead = requiredTargetHead
	value := history.Before
	if request.Kind == authoring.OperationRedo {
		value = history.After
	}
	effective.Value = value.Value
	// history.Target's own shape (not history.Kind, which is "undo" for a
	// redo's history record) identifies which durable family this replay
	// belongs to -- the same classification InverseRequest/CanonicalRequest
	// use, so a target can never be replayed as a different family than the
	// one that originally wrote it.
	effective.Kind = authoring.SetKindForTarget(history.Target, value.Present)
	return effective, history, &value, nil
}

func readHead(ctx context.Context, tx *sql.Tx, key, targetKey string) (string, authoring.OperationValue, error) {
	var head, value string
	var present int
	err := tx.QueryRowContext(ctx, `SELECT head_operation_id,value_present,value FROM studio_field_heads WHERE resource_key=? AND target_key=?`, key, targetKey).Scan(&head, &present, &value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", authoring.OperationValue{}, nil
	}
	return head, authoring.OperationValue{Present: present != 0, Value: value}, err
}

func insertRejected(ctx context.Context, tx *sql.Tx, key, actor string, request authoring.OperationRequest, protocolErr *collab.ProtocolError, current authoring.OperationValue, now time.Time) error {
	raw, _ := json.Marshal(request)
	_, err := tx.ExecContext(ctx, `INSERT INTO studio_operation_attempts(resource_key,operation_id,actor_id,accepted,error_code,request_json,current_head,current_value_present,current_value,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, key, request.ID, actor, 0, protocolErr.Code, raw, protocolErr.CurrentHead, boolInt(current.Present), current.Value, timestamp(now))
	return err
}

func (s *Store) Tail(ctx context.Context, resource collab.ResourceKey, after uint64, limit int) ([]collab.OperationAck, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT accepted_sequence,record_json FROM studio_operation_attempts WHERE resource_key=? AND accepted=1 AND accepted_sequence>? ORDER BY accepted_sequence LIMIT ?`, resource.Normalize().StableKey(), after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []collab.OperationAck
	for rows.Next() {
		var ack collab.OperationAck
		var raw []byte
		if err := rows.Scan(&ack.Sequence, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &ack.Record); err != nil {
			return nil, err
		}
		result = append(result, ack)
	}
	return result, rows.Err()
}

func (s *Store) Attempts(ctx context.Context, resource collab.ResourceKey) ([]collab.Attempt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT attempt_id,operation_id,actor_id,accepted,COALESCE(accepted_sequence,0),error_code,request_json,current_head,current_value_present,current_value,created_at FROM studio_operation_attempts WHERE resource_key=? ORDER BY attempt_id`, resource.Normalize().StableKey())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []collab.Attempt
	for rows.Next() {
		var a collab.Attempt
		var accepted, present int
		var raw, created string
		if err := rows.Scan(&a.ID, &a.OperationID, &a.ActorID, &accepted, &a.Sequence, &a.ErrorCode, &raw, &a.CurrentHead, &present, &a.CurrentValue.Value, &created); err != nil {
			return nil, err
		}
		a.Resource = resource.Normalize()
		a.Accepted = accepted != 0
		a.CurrentValue.Present = present != 0
		if err := json.Unmarshal([]byte(raw), &a.Request); err != nil {
			return nil, err
		}
		a.Created, _ = time.Parse(time.RFC3339Nano, created)
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *Store) PendingOutbox(ctx context.Context, resource collab.ResourceKey, limit int) ([]collab.OutboxEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT outbox_id,accepted_sequence,operation_id,payload,created_at FROM studio_projection_outbox WHERE resource_key=? AND projected_at IS NULL ORDER BY accepted_sequence LIMIT ?`, resource.Normalize().StableKey(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []collab.OutboxEntry
	for rows.Next() {
		var e collab.OutboxEntry
		var created string
		if err := rows.Scan(&e.ID, &e.Sequence, &e.OperationID, &e.Payload, &created); err != nil {
			return nil, err
		}
		e.Resource = resource.Normalize()
		e.Created, _ = time.Parse(time.RFC3339Nano, created)
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *Store) MarkProjected(ctx context.Context, resource collab.ResourceKey, id int64, at time.Time) error {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	result, err := s.db.ExecContext(ctx, `UPDATE studio_projection_outbox SET projected_at=? WHERE resource_key=? AND outbox_id=? AND projected_at IS NULL`, timestamp(at.UTC()), resource.Normalize().StableKey(), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected != 1 {
		return fmt.Errorf("projection outbox entry not pending")
	}
	return nil
}

// RepairFieldHead force-overwrites one target's studio_field_heads row to
// cmd.NewHead/cmd.NewValue, bypassing every Apply invariant (expected-head
// match, actor ownership, operation-id idempotency) on purpose -- see
// collab.RepairFieldHeadCommand's doc comment for why. It never touches
// studio_operation_attempts, studio_projection_outbox, or
// studio_resources.accepted_sequence: a repair is explicitly outside the
// accepted-operation ledger, not a new operation in it. Every call (success
// or a no-op repairing an already-healthy head) writes exactly one
// studio_field_head_repairs audit row.
func (s *Store) RepairFieldHead(ctx context.Context, cmd collab.RepairFieldHeadCommand) (collab.RepairFieldHeadResult, *collab.ProtocolError) {
	resource := cmd.Resource.Normalize()
	principal := cmd.Principal.Clone()
	target := cmd.Target.Normalize()
	reason := strings.TrimSpace(cmd.Reason)
	newHead := strings.TrimSpace(cmd.NewHead)
	newValue := cmd.NewValue
	now := cmd.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if err := resource.Validate(); err != nil {
		return collab.RepairFieldHeadResult{}, collab.NewProtocolError(collab.ErrorInvalidRequest, err.Error())
	}
	if err := principal.Validate(); err != nil {
		return collab.RepairFieldHeadResult{}, collab.NewProtocolError(collab.ErrorForbidden, err.Error())
	}
	if reason == "" {
		return collab.RepairFieldHeadResult{}, collab.NewProtocolError(collab.ErrorInvalidRequest, "a repair reason is required")
	}
	// Key's kind parameter is unused (target key hashing depends only on the
	// normalized target's own fields) -- see authoring.OperationTarget.Key.
	targetKey := target.Key("")
	if !principal.Can(collab.CapabilityRepair) {
		protocolErr := collab.NewProtocolError(collab.ErrorForbidden, "principal lacks the field-head repair capability")
		return collab.RepairFieldHeadResult{}, protocolErr
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return collab.RepairFieldHeadResult{}, storeError(err)
	}
	defer tx.Rollback()
	key := resource.StableKey()
	if _, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO studio_resources(resource_key,tenant_id,project_id,resource_kind,resource_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, key, resource.TenantID, resource.ProjectID, resource.Kind, resource.ID, timestamp(now), timestamp(now)); err != nil {
		return collab.RepairFieldHeadResult{}, storeError(err)
	}

	var oldHead, oldValue string
	var oldPresentInt, sequence int
	scanErr := tx.QueryRowContext(ctx, `SELECT head_operation_id,value_present,value,sequence FROM studio_field_heads WHERE resource_key=? AND target_key=?`, key, targetKey).Scan(&oldHead, &oldPresentInt, &oldValue, &sequence)
	if scanErr != nil {
		if !errors.Is(scanErr, sql.ErrNoRows) {
			return collab.RepairFieldHeadResult{}, storeError(scanErr)
		}
		oldHead, oldPresentInt, oldValue, sequence = "", 0, "", 0
	}
	oldValuePresent := oldPresentInt != 0

	if _, err := tx.ExecContext(ctx, `INSERT INTO studio_field_heads(resource_key,target_key,head_operation_id,sequence,value_present,value,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(resource_key,target_key) DO UPDATE SET head_operation_id=excluded.head_operation_id,value_present=excluded.value_present,value=excluded.value,updated_at=excluded.updated_at`, key, targetKey, newHead, sequence, boolInt(newValue.Present), newValue.Value, timestamp(now)); err != nil {
		return collab.RepairFieldHeadResult{}, storeError(err)
	}

	targetJSON, _ := json.Marshal(target)
	if _, err := tx.ExecContext(ctx, `INSERT INTO studio_field_head_repairs(resource_key,target_key,target_json,actor_id,reason,old_head,old_value_present,old_value,new_head,new_value_present,new_value,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		key, targetKey, targetJSON, principal.ActorID, reason, oldHead, boolInt(oldValuePresent), oldValue, newHead, boolInt(newValue.Present), newValue.Value, timestamp(now)); err != nil {
		return collab.RepairFieldHeadResult{}, storeError(err)
	}
	if err := tx.Commit(); err != nil {
		return collab.RepairFieldHeadResult{}, storeError(err)
	}
	return collab.RepairFieldHeadResult{
		Resource: resource, TargetKey: targetKey, Target: target,
		ActorID: principal.ActorID, Reason: reason,
		OldHead: oldHead, OldValue: authoring.OperationValue{Present: oldValuePresent, Value: oldValue},
		NewHead: newHead, NewValue: newValue, Created: now,
	}, nil
}

// RepairHistory returns resource's durable RepairFieldHead audit trail,
// oldest first.
func (s *Store) RepairHistory(ctx context.Context, resource collab.ResourceKey, limit int) ([]collab.RepairRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	resource = resource.Normalize()
	rows, err := s.db.QueryContext(ctx, `SELECT repair_id,target_key,target_json,actor_id,reason,old_head,old_value_present,old_value,new_head,new_value_present,new_value,created_at FROM studio_field_head_repairs WHERE resource_key=? ORDER BY repair_id LIMIT ?`, resource.StableKey(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []collab.RepairRecord
	for rows.Next() {
		var r collab.RepairRecord
		var targetJSON []byte
		var oldPresent, newPresent int
		var created string
		if err := rows.Scan(&r.ID, &r.TargetKey, &targetJSON, &r.ActorID, &r.Reason, &r.OldHead, &oldPresent, &r.OldValue.Value, &r.NewHead, &newPresent, &r.NewValue.Value, &created); err != nil {
			return nil, err
		}
		r.Resource = resource
		r.OldValue.Present = oldPresent != 0
		r.NewValue.Present = newPresent != 0
		_ = json.Unmarshal(targetJSON, &r.Target)
		r.Created, _ = time.Parse(time.RFC3339Nano, created)
		result = append(result, r)
	}
	return result, rows.Err()
}

func storeError(err error) *collab.ProtocolError {
	e := collab.NewProtocolError(collab.ErrorStoreUnavailable, err.Error())
	e.Retryable = true
	return e
}
func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// StateHash is useful for reopen/convergence proofs without exposing content.
func (s *Store) StateHash(ctx context.Context, resource collab.ResourceKey) (string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT target_key,head_operation_id,sequence,value_present,value FROM studio_field_heads WHERE resource_key=? ORDER BY target_key`, resource.Normalize().StableKey())
	if err != nil {
		return "", err
	}
	defer rows.Close()
	h := sha256.New()
	for rows.Next() {
		var target, head, value string
		var seq uint64
		var present int
		if err := rows.Scan(&target, &head, &seq, &present, &value); err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\x00%s\x00%d\x00%d\x00%s\x00", target, head, seq, present, value)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
