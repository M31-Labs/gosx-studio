package collab

import (
	"fmt"
	"sync"

	"m31labs.dev/gosx-admin/blockstudio"
)

type MemoryStore struct {
	mu     sync.RWMutex
	drafts map[string]blockstudio.Document
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{drafts: map[string]blockstudio.Document{}}
}

func (s *MemoryStore) LoadDraft(resource Resource) (blockstudio.Document, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.drafts[resourceKey(resource)]
	if !ok {
		return blockstudio.Document{}, false, nil
	}
	return cloneDocument(doc), true, nil
}

func (s *MemoryStore) SaveDraft(resource Resource, doc blockstudio.Document) error {
	resource = normalizeResource(resource)
	if resource.Kind == "" || resource.ID == "" {
		return fmt.Errorf("studio collab memory store requires resource kind and id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.drafts == nil {
		s.drafts = map[string]blockstudio.Document{}
	}
	s.drafts[resourceKey(resource)] = cloneDocument(doc)
	return nil
}
