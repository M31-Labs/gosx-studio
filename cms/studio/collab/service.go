package collab

import (
	"context"
	"fmt"
	"time"

	"m31labs.dev/gosx-studio/authoring"
)

// Service binds a host-owned resource to the durable ledger. Client frames
// contain only OperationRequest, so tenant/project/resource cannot be swapped
// by a browser. The host resolves Principal from trusted connection metadata.
type Service struct {
	resource ResourceKey
	store    OperationStore
	presence *PresenceRegistry
	clock    func() time.Time
}

func NewService(resource ResourceKey, store OperationStore) (*Service, error) {
	resource = resource.Normalize()
	if err := resource.Validate(); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf("collaboration operation store is required")
	}
	return &Service{resource: resource, store: store, presence: NewPresenceRegistry(), clock: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *Service) Resource() ResourceKey { return s.resource }
func (s *Service) Seed(ctx context.Context, heads ...SeedHead) error {
	return s.store.SeedHeads(ctx, s.resource, heads)
}
func (s *Service) Submit(ctx context.Context, principal Principal, request authoring.OperationRequest) (OperationAck, *ProtocolError) {
	return s.store.Apply(ctx, ApplyCommand{Resource: s.resource, Principal: principal.Clone(), Request: request, Now: s.clock()})
}
func (s *Service) Resume(ctx context.Context, after uint64, limit int) ([]OperationAck, error) {
	return s.store.Tail(ctx, s.resource, after, limit)
}
func (s *Service) Presence() *PresenceRegistry { return s.presence }
