package collab

import (
	"context"
	"sort"
	"sync"
	"time"

	"m31labs.dev/gosx-studio/authoring"
)

type ApplyCommand struct {
	Resource  ResourceKey
	Principal Principal
	Request   authoring.OperationRequest
	Now       time.Time
}

type Attempt struct {
	ID           int64
	Resource     ResourceKey
	OperationID  string
	ActorID      string
	Accepted     bool
	Sequence     uint64
	ErrorCode    ErrorCode
	Request      authoring.OperationRequest
	CurrentHead  string
	CurrentValue authoring.OperationValue
	Created      time.Time
}

type OutboxEntry struct {
	ID          int64
	Resource    ResourceKey
	Sequence    uint64
	OperationID string
	Payload     []byte
	Created     time.Time
	Projected   *time.Time
}

type OperationStore interface {
	Apply(context.Context, ApplyCommand) (OperationAck, *ProtocolError)
	Tail(context.Context, ResourceKey, uint64, int) ([]OperationAck, error)
	Attempts(context.Context, ResourceKey) ([]Attempt, error)
	PendingOutbox(context.Context, ResourceKey, int) ([]OutboxEntry, error)
	MarkProjected(context.Context, ResourceKey, int64, time.Time) error
	Close() error
}

// PresenceRegistry is intentionally process-local and connection-keyed. A
// restart starts empty; actor identity never determines connection lifetime.
type PresenceRegistry struct {
	mu      sync.RWMutex
	entries map[string]PresenceEntry
}
type PresenceEntry struct {
	ConnectionID string
	Principal    Principal
	Selection    SelectionState
	Cursor       CursorState
}

func NewPresenceRegistry() *PresenceRegistry {
	return &PresenceRegistry{entries: map[string]PresenceEntry{}}
}
func (p *PresenceRegistry) Join(connectionID string, principal Principal) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.entries[connectionID] = PresenceEntry{ConnectionID: connectionID, Principal: principal.Clone()}
}
func (p *PresenceRegistry) Leave(connectionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.entries, connectionID)
}
func (p *PresenceRegistry) Entries() []PresenceEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]PresenceEntry, 0, len(p.entries))
	for _, e := range p.entries {
		e.Principal = e.Principal.Clone()
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ConnectionID < out[j].ConnectionID })
	return out
}
