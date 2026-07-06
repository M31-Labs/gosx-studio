package flows

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type FileStore struct {
	*MemoryStore

	mu   sync.Mutex
	path string
}

type fileStoreData struct {
	Version      int           `json:"version"`
	Documents    []Document    `json:"documents,omitempty"`
	Drafts       []Draft       `json:"drafts,omitempty"`
	Publications []Publication `json:"publications,omitempty"`
}

var (
	_ DocumentStore    = (*FileStore)(nil)
	_ DraftStore       = (*FileStore)(nil)
	_ PublicationStore = (*FileStore)(nil)
)

func NewFileStore(path string, documents ...Document) (*FileStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("flow file store path is required")
	}
	store := &FileStore{
		MemoryStore: NewMemoryStore(documents...),
		path:        path,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileStore) SaveFlowDocument(document Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.MemoryStore.SaveFlowDocument(document); err != nil {
		return err
	}
	return s.persist()
}

func (s *FileStore) SaveFlowDraft(draft Draft) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.MemoryStore.SaveFlowDraft(draft); err != nil {
		return err
	}
	return s.persist()
}

func (s *FileStore) SaveFlowPublication(publication Publication) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.MemoryStore.SaveFlowPublication(publication); err != nil {
		return err
	}
	return s.persist()
}

func (s *FileStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}
	var snapshot fileStoreData
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("load flow file store: %w", err)
	}
	for _, document := range snapshot.Documents {
		if err := s.MemoryStore.SaveFlowDocument(document); err != nil {
			return err
		}
	}
	for _, draft := range snapshot.Drafts {
		if err := s.MemoryStore.SaveFlowDraft(draft); err != nil {
			return err
		}
	}
	for _, publication := range snapshot.Publications {
		if err := s.MemoryStore.SaveFlowPublication(publication); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStore) persist() error {
	snapshot := s.snapshot()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWriteFile(s.path, data, 0o644)
}

func (s *FileStore) snapshot() fileStoreData {
	s.MemoryStore.mu.RLock()
	defer s.MemoryStore.mu.RUnlock()
	return fileStoreData{
		Version:      1,
		Documents:    CloneDocuments(s.MemoryStore.documents),
		Drafts:       cloneDrafts(s.MemoryStore.drafts),
		Publications: clonePublications(s.MemoryStore.publications),
	}
}

func cloneDrafts(drafts map[string]Draft) []Draft {
	out := make([]Draft, 0, len(drafts))
	for _, draft := range drafts {
		draft.Document = CloneDocument(draft.Document)
		out = append(out, draft)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return documentResourceID(out[i].Document) < documentResourceID(out[j].Document)
	})
	return out
}

func clonePublications(publications map[string]Publication) []Publication {
	out := make([]Publication, 0, len(publications))
	for _, publication := range publications {
		publication.Document = CloneDocument(publication.Document)
		out = append(out, publication)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return documentResourceID(out[i].Document) < documentResourceID(out[j].Document)
	})
	return out
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Chmod(perm); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	removeTemp = false
	return nil
}
