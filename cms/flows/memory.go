package flows

import (
	"strings"
	"sync"
)

type MemoryStore struct {
	mu           sync.RWMutex
	documents    []Document
	drafts       map[string]Draft
	publications map[string]Publication
}

var (
	_ DocumentStore    = (*MemoryStore)(nil)
	_ DraftStore       = (*MemoryStore)(nil)
	_ PublicationStore = (*MemoryStore)(nil)
)

func NewMemoryStore(documents ...Document) *MemoryStore {
	store := &MemoryStore{
		documents:    make([]Document, 0, len(documents)),
		drafts:       map[string]Draft{},
		publications: map[string]Publication{},
	}
	for _, document := range documents {
		_ = store.SaveFlowDocument(document)
	}
	return store
}

func (s *MemoryStore) ListFlowDocuments() []Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return CloneDocuments(s.documents)
}

func (s *MemoryStore) GetFlowDocument(id string) (Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id = normalizeKey(firstNonEmpty(id, strings.TrimSpace(id)))
	for _, document := range s.documents {
		if documentIDMatches(document, id) {
			return CloneDocument(document), true
		}
	}
	return Document{}, false
}

func (s *MemoryStore) SaveFlowDocument(document Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	document = NormalizeDocument(document, WithUnknownStepBlocks())
	for index, existing := range s.documents {
		if documentIdentity(existing) == documentIdentity(document) || existing.Key == document.Key {
			s.documents[index] = CloneDocument(document)
			return nil
		}
	}
	s.documents = append(s.documents, CloneDocument(document))
	return nil
}

func (s *MemoryStore) GetFlowDraft(documentID string) (Draft, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := normalizeKey(documentID)
	draft, ok := s.drafts[key]
	if !ok {
		for _, candidate := range s.drafts {
			if documentIDMatches(candidate.Document, key) {
				draft = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return Draft{}, false
	}
	draft.Document = CloneDocument(draft.Document)
	return draft, true
}

func (s *MemoryStore) SaveFlowDraft(draft Draft) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft.Document = NormalizeDocument(draft.Document, WithUnknownStepBlocks())
	key := normalizeKey(documentResourceID(draft.Document))
	if key == "" {
		key = normalizeKey(draft.Document.Key)
	}
	draft.Document = CloneDocument(draft.Document)
	s.drafts[key] = draft
	return nil
}

func (s *MemoryStore) GetFlowPublication(documentID string) (Publication, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := normalizeKey(documentID)
	publication, ok := s.publications[key]
	if !ok {
		for _, candidate := range s.publications {
			if documentIDMatches(candidate.Document, key) {
				publication = candidate
				ok = true
				break
			}
		}
	}
	if !ok {
		return Publication{}, false
	}
	publication.Document = CloneDocument(publication.Document)
	return publication, true
}

func (s *MemoryStore) SaveFlowPublication(publication Publication) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	publication.Document = NormalizeDocument(publication.Document, WithUnknownStepBlocks())
	key := normalizeKey(documentResourceID(publication.Document))
	if key == "" {
		key = normalizeKey(publication.Document.Key)
	}
	publication.Document = CloneDocument(publication.Document)
	s.publications[key] = publication
	return nil
}

func documentIDMatches(document Document, id string) bool {
	return normalizeKey(document.ID) == id || normalizeKey(document.Key) == id || normalizeKey(documentResourceID(document)) == id
}

func documentIdentity(document Document) string {
	if strings.TrimSpace(document.ID) != "" {
		return normalizeKey(document.ID)
	}
	return normalizeKey(document.Key)
}
