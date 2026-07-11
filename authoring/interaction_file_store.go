package authoring

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type FileInteractionStateStore struct {
	Path string
	mu   sync.Mutex
}

func (s *FileInteractionStateStore) LoadInteractionState(context.Context) (InteractionState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}
func (s *FileInteractionStateStore) SaveInteractionState(_ context.Context, state InteractionState, expected uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, e := s.load()
	if e != nil {
		return e
	}
	if current.Revision != expected {
		return ErrInteractionConflict
	}
	path, e := filepath.Abs(s.Path)
	if e != nil || s.Path == "" {
		return coreInvalid()
	}
	if e = os.MkdirAll(filepath.Dir(path), 0750); e != nil {
		return e
	}
	data, e := json.MarshalIndent(normalizeInteractionState(state), "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".interaction-state-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	if closeErr := f.Close(); e == nil {
		e = closeErr
	}
	if e != nil {
		return e
	}
	return os.Rename(name, path)
}
func (s *FileInteractionStateStore) load() (InteractionState, error) {
	if s == nil || s.Path == "" {
		return InteractionState{}, coreInvalid()
	}
	data, e := os.ReadFile(s.Path)
	if errors.Is(e, os.ErrNotExist) {
		return normalizeInteractionState(InteractionState{}), nil
	}
	if e != nil {
		return InteractionState{}, e
	}
	var state InteractionState
	if e = json.Unmarshal(data, &state); e != nil {
		return InteractionState{}, e
	}
	return normalizeInteractionState(state), nil
}
func coreInvalid() error { return errors.New("invalid interaction state path") }

var _ InteractionStateStore = (*FileInteractionStateStore)(nil)
