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

// SeedHead records one existing host draft value before the first accepted
// collaboration operation for that target. It creates no actor attempt,
// accepted sequence, or projection outbox entry.
type SeedHead struct {
	Target authoring.OperationTarget
	Value  authoring.OperationValue
}

type OperationStore interface {
	SeedHeads(context.Context, ResourceKey, []SeedHead) error
	Apply(context.Context, ApplyCommand) (OperationAck, *ProtocolError)
	Tail(context.Context, ResourceKey, uint64, int) ([]OperationAck, error)
	Attempts(context.Context, ResourceKey) ([]Attempt, error)
	PendingOutbox(context.Context, ResourceKey, int) ([]OutboxEntry, error)
	MarkProjected(context.Context, ResourceKey, int64, time.Time) error
	// RepairFieldHead force-overwrites one target's field head to a
	// caller-supplied known-good revision, bypassing Apply's optimistic
	// concurrency envelope entirely. See RepairFieldHeadCommand.
	RepairFieldHead(context.Context, RepairFieldHeadCommand) (RepairFieldHeadResult, *ProtocolError)
	// RepairHistory returns the durable audit trail RepairFieldHead writes,
	// oldest first, for one resource.
	RepairHistory(context.Context, ResourceKey, int) ([]RepairRecord, error)
	Close() error
}

// RepairFieldHeadCommand force-overwrites one target's field-head slot
// (studio_field_heads) to a caller-supplied known-good revision, bypassing
// Apply's optimistic-concurrency envelope (expected-head match, actor-scoped
// Undo/Redo) entirely.
//
// It exists for exactly one recovery situation: a host quarantined an
// operation its projection could not apply (see the poison-outbox handoff
// report), leaving that target's ledger field head advanced past a value the
// host's real draft store never actually received. Apply's normal machinery
// cannot recover that target -- there is no legitimate operation to Undo,
// because it is the PROJECTION that diverged from the ledger, not a
// legitimately-reversible edit. Every future Submit for that target keeps
// failing ErrorStaleField against a head the host's draft can never present,
// until something resets the ledger's belief about the field back to what
// the host's draft actually holds.
//
// RepairFieldHead is the host-callable primitive that performs that reset.
// It is deliberately NOT part of the ordinary Apply path: it requires
// CapabilityRepair (never granted to a browser-facing collaboration
// principal -- see CapabilityRepair's doc comment), and every call -- success
// or no-op -- writes a durable RepairRecord (RepairHistory) so a repair is
// always attributable to an actor and a reason, never silent.
type RepairFieldHeadCommand struct {
	Resource  ResourceKey
	Principal Principal
	Target    authoring.OperationTarget
	// NewHead is the operation id RepairFieldHead sets as this target's
	// field head. It does not need to reference a real accepted operation
	// (the whole point is recovering a target the ledger's own history
	// cannot explain) -- hosts typically pass the id of the last operation
	// whose projection is known-good, or a synthetic marker of their own.
	NewHead string
	// NewValue is the known-good current value RepairFieldHead writes
	// alongside NewHead -- typically read back from the host's own draft
	// store after manual recovery.
	NewValue authoring.OperationValue
	// Reason is required: a short operator-facing note (e.g. "poison outbox
	// recovery, ticket #123") persisted verbatim in the audit record.
	Reason string
	Now    time.Time
}

// RepairFieldHeadResult reports one RepairFieldHead call's before/after
// state. It mirrors OperationAck's shape (what changed, at what target) but
// carries no DocumentRevision/Sequence -- a repair is explicitly outside the
// accepted-operation sequence hosts replay through Tail/PendingOutbox.
type RepairFieldHeadResult struct {
	Resource  ResourceKey
	TargetKey string
	Target    authoring.OperationTarget
	ActorID   string
	Reason    string
	OldHead   string
	OldValue  authoring.OperationValue
	NewHead   string
	NewValue  authoring.OperationValue
	Created   time.Time
}

// RepairRecord is one durable, queryable audit entry a RepairFieldHead call
// wrote: Old/NewHead+Value are exactly what changed, ActorID+Reason are who
// did it and why. Every RepairFieldHead call writes exactly one of these,
// including a call that repairs an already-healthy head to its own current
// value -- a safe, idempotent no-op on studio_field_heads that still leaves
// an audit trail of the attempt.
type RepairRecord struct {
	ID        int64
	Resource  ResourceKey
	TargetKey string
	Target    authoring.OperationTarget
	ActorID   string
	Reason    string
	OldHead   string
	OldValue  authoring.OperationValue
	NewHead   string
	NewValue  authoring.OperationValue
	Created   time.Time
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
func (p *PresenceRegistry) SetSelection(connectionID string, selection SelectionState) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.entries[connectionID]
	if !ok {
		return false
	}
	entry.Selection = selection.Normalize()
	p.entries[connectionID] = entry
	return true
}
func (p *PresenceRegistry) SetCursor(connectionID string, cursor CursorState) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	entry, ok := p.entries[connectionID]
	if !ok {
		return false
	}
	entry.Cursor = cursor.Normalize()
	p.entries[connectionID] = entry
	return true
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
