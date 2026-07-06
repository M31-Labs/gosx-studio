package lifecycle

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const DefaultRevisionLimit = 500

type DraftState string
type PublishState string

const (
	DraftStateDraft    DraftState = "draft"
	DraftStatePreview  DraftState = "preview"
	DraftStateRollback DraftState = "rolled_back"

	PublishStateDraft     PublishState = "draft"
	PublishStatePublished PublishState = "published"
)

type Revision struct {
	ID            string          `json:"id"`
	ResourceKind  string          `json:"resourceKind"`
	ResourceID    string          `json:"resourceId"`
	ResourceTitle string          `json:"resourceTitle,omitempty"`
	Action        string          `json:"action"`
	Summary       string          `json:"summary,omitempty"`
	Snapshot      json.RawMessage `json:"snapshot"`
	Created       time.Time       `json:"created"`
}

type RevisionInput struct {
	ID            string
	ResourceKind  string
	ResourceID    string
	ResourceTitle string
	Action        string
	Summary       string
	Snapshot      any
	Created       time.Time
}

type Filter struct {
	ResourceKind string
	ResourceID   string
	Limit        int
}

type RevisionFilter = Filter

type RevisionStore interface {
	ListRevisions(Filter) []Revision
}

func NewRevision(input RevisionInput, now time.Time) (Revision, error) {
	resourceKind := strings.TrimSpace(input.ResourceKind)
	resourceID := strings.TrimSpace(input.ResourceID)
	action := strings.TrimSpace(input.Action)
	if resourceKind == "" || resourceID == "" || action == "" {
		return Revision{}, fmt.Errorf("revision requires resource kind, resource id, and action")
	}
	if input.Snapshot == nil {
		return Revision{}, fmt.Errorf("revision requires a snapshot")
	}
	raw, err := Snapshot(input.Snapshot)
	if err != nil {
		return Revision{}, err
	}
	if input.Created.IsZero() {
		input.Created = now
	}
	return Revision{
		ID:            strings.TrimSpace(input.ID),
		ResourceKind:  resourceKind,
		ResourceID:    resourceID,
		ResourceTitle: strings.TrimSpace(input.ResourceTitle),
		Action:        action,
		Summary:       strings.TrimSpace(input.Summary),
		Snapshot:      raw,
		Created:       input.Created,
	}, nil
}

func Snapshot(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, fmt.Errorf("snapshot value is nil")
	}
	if raw, ok := value.(json.RawMessage); ok {
		if len(raw) == 0 {
			return nil, fmt.Errorf("snapshot value is empty")
		}
		return append(json.RawMessage(nil), raw...), nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("snapshot value is not serializable: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("snapshot value is empty")
	}
	return append(json.RawMessage(nil), data...), nil
}

func FilterRevisions(revisions []Revision, filter Filter) []Revision {
	out := make([]Revision, 0, len(revisions))
	for _, revision := range revisions {
		if strings.TrimSpace(filter.ResourceKind) != "" && revision.ResourceKind != filter.ResourceKind {
			continue
		}
		if strings.TrimSpace(filter.ResourceID) != "" && revision.ResourceID != filter.ResourceID {
			continue
		}
		out = append(out, revision)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Created.After(out[j].Created)
	})
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return CloneRevisions(out)
}

func ListRevisions(revisions []Revision, filter RevisionFilter) []Revision {
	return FilterRevisions(revisions, filter)
}

func FindRevision(revisions []Revision, resourceKind, resourceID, revisionID string) (Revision, bool) {
	resourceKind = strings.TrimSpace(resourceKind)
	resourceID = strings.TrimSpace(resourceID)
	revisionID = strings.TrimSpace(revisionID)
	for _, revision := range revisions {
		if revision.ID == revisionID && revision.ResourceKind == resourceKind && revision.ResourceID == resourceID {
			return CloneRevision(revision), true
		}
	}
	return Revision{}, false
}

func DecodeSnapshot[T any](revision Revision) (T, error) {
	var out T
	if err := json.Unmarshal(revision.Snapshot, &out); err != nil {
		return out, err
	}
	return out, nil
}

func NormalizeRevision(revision Revision, now time.Time, newID func(string) string) Revision {
	revision.ResourceKind = strings.TrimSpace(revision.ResourceKind)
	revision.ResourceID = strings.TrimSpace(revision.ResourceID)
	revision.ResourceTitle = strings.TrimSpace(revision.ResourceTitle)
	revision.Action = strings.TrimSpace(revision.Action)
	revision.Summary = strings.TrimSpace(revision.Summary)
	revision.Snapshot = append(json.RawMessage(nil), revision.Snapshot...)
	if strings.TrimSpace(revision.ID) == "" && newID != nil {
		revision.ID = strings.TrimSpace(newID("rev"))
	}
	if revision.Created.IsZero() {
		revision.Created = now
	}
	return revision
}

func TrimRevisions(revisions []Revision, limit int) []Revision {
	if limit <= 0 {
		limit = DefaultRevisionLimit
	}
	if len(revisions) <= limit {
		return revisions
	}
	return revisions[len(revisions)-limit:]
}

func CloneRevision(revision Revision) Revision {
	revision.Snapshot = append(json.RawMessage(nil), revision.Snapshot...)
	return revision
}

func CloneRevisions(revisions []Revision) []Revision {
	out := make([]Revision, len(revisions))
	for i, revision := range revisions {
		out[i] = CloneRevision(revision)
	}
	return out
}

func ActionLabel(action string) string {
	action = strings.TrimSpace(action)
	for _, prefix := range []string{"page.", "blog.", "settings."} {
		action = strings.TrimPrefix(action, prefix)
	}
	action = strings.ReplaceAll(action, "_", " ")
	action = strings.ReplaceAll(action, ".", " ")
	if action == "" {
		return "saved"
	}
	return action
}
