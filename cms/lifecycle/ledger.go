package lifecycle

import (
	"context"
	"time"
)

type DecisionStatus string

const (
	DecisionPending          DecisionStatus = "pending"
	DecisionApproved         DecisionStatus = "approved"
	DecisionRejected         DecisionStatus = "rejected"
	DecisionChangesRequested DecisionStatus = "changes_requested"
)

type ScheduleAction string

const (
	ScheduleActionPublish   ScheduleAction = "publish"
	ScheduleActionUnpublish ScheduleAction = "unpublish"
)

type ScheduleState string

const (
	SchedulePending   ScheduleState = "pending"
	ScheduleClaimed   ScheduleState = "claimed"
	ScheduleCompleted ScheduleState = "completed"
	ScheduleCancelled ScheduleState = "cancelled"
)

type PublishDecision struct {
	ID           string         `json:"id"`
	ResourceKind string         `json:"resourceKind"`
	ResourceID   string         `json:"resourceId"`
	RevisionID   string         `json:"revisionId,omitempty"`
	Status       DecisionStatus `json:"status"`
	ActorID      string         `json:"actorId,omitempty"`
	Note         string         `json:"note,omitempty"`
	Created      time.Time      `json:"created"`
}

type PublishDecisionInput struct {
	ID           string
	ResourceKind string
	ResourceID   string
	RevisionID   string
	Status       DecisionStatus
	ActorID      string
	Note         string
	Created      time.Time
}

type PublishSchedule struct {
	ID           string         `json:"id"`
	ResourceKind string         `json:"resourceKind"`
	ResourceID   string         `json:"resourceId"`
	RevisionID   string         `json:"revisionId,omitempty"`
	Action       ScheduleAction `json:"action"`
	State        ScheduleState  `json:"state"`
	DueAt        time.Time      `json:"dueAt"`
	Timezone     string         `json:"timezone,omitempty"`
	ActorID      string         `json:"actorId,omitempty"`
	Note         string         `json:"note,omitempty"`
	ClaimToken   string         `json:"claimToken,omitempty"`
	ClaimedAt    *time.Time     `json:"claimedAt,omitempty"`
	CompletedAt  *time.Time     `json:"completedAt,omitempty"`
	CancelledAt  *time.Time     `json:"cancelledAt,omitempty"`
	Created      time.Time      `json:"created"`
	Updated      time.Time      `json:"updated"`
}

type PublishScheduleInput struct {
	ID           string
	ResourceKind string
	ResourceID   string
	RevisionID   string
	Action       ScheduleAction
	State        ScheduleState
	DueAt        time.Time
	Timezone     string
	ActorID      string
	Note         string
	Created      time.Time
}

type PublishNote struct {
	ID           string    `json:"id"`
	ResourceKind string    `json:"resourceKind"`
	ResourceID   string    `json:"resourceId"`
	RevisionID   string    `json:"revisionId,omitempty"`
	ActorID      string    `json:"actorId,omitempty"`
	Body         string    `json:"body"`
	Created      time.Time `json:"created"`
}

type PublishNoteInput struct {
	ID           string
	ResourceKind string
	ResourceID   string
	RevisionID   string
	ActorID      string
	Body         string
	Created      time.Time
}

type AuditEvent struct {
	ID           string            `json:"id"`
	ResourceKind string            `json:"resourceKind,omitempty"`
	ResourceID   string            `json:"resourceId,omitempty"`
	RevisionID   string            `json:"revisionId,omitempty"`
	Action       string            `json:"action"`
	ActorID      string            `json:"actorId,omitempty"`
	Summary      string            `json:"summary,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Created      time.Time         `json:"created"`
}

type AuditEventInput struct {
	ID           string
	ResourceKind string
	ResourceID   string
	RevisionID   string
	Action       string
	ActorID      string
	Summary      string
	Metadata     map[string]string
	Created      time.Time
}

type LedgerFilter struct {
	ResourceKind string
	ResourceID   string
	RevisionID   string
	ActorID      string
	Action       string
	Limit        int
}

type ScheduleFilter struct {
	ResourceKind string
	ResourceID   string
	RevisionID   string
	State        ScheduleState
	DueBefore    time.Time
	Limit        int
}

type PublishDecisionStore interface {
	SavePublishDecision(context.Context, PublishDecisionInput) (PublishDecision, error)
	LatestPublishDecision(context.Context, string, string) (PublishDecision, bool, error)
	ListPublishDecisions(context.Context, LedgerFilter) ([]PublishDecision, error)
}

type PublishScheduleStore interface {
	SavePublishSchedule(context.Context, PublishScheduleInput) (PublishSchedule, error)
	ListPublishSchedules(context.Context, ScheduleFilter) ([]PublishSchedule, error)
	CancelPublishSchedule(context.Context, string, string, string) (PublishSchedule, bool, error)
	ClaimDueSchedules(context.Context, time.Time, int, string) ([]PublishSchedule, error)
	CompletePublishSchedule(context.Context, string) (PublishSchedule, bool, error)
}

type PublishNoteStore interface {
	SavePublishNote(context.Context, PublishNoteInput) (PublishNote, error)
	ListPublishNotes(context.Context, LedgerFilter) ([]PublishNote, error)
}

type AuditStore interface {
	SaveAuditEvent(context.Context, AuditEventInput) (AuditEvent, error)
	ListAuditEvents(context.Context, LedgerFilter) ([]AuditEvent, error)
}

type LedgerStore interface {
	PublishDecisionStore
	PublishScheduleStore
	PublishNoteStore
	AuditStore
}
