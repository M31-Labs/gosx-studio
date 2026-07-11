package authoring

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type FileFlowStateStore struct {
	Path string
	mu   sync.Mutex
}

func (s *FileFlowStateStore) LoadFlowState(context.Context) (FlowState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}
func (s *FileFlowStateStore) SaveFlowState(_ context.Context, state FlowState, expected uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, e := s.load()
	if e != nil {
		return e
	}
	if cur.Revision != expected {
		return ErrFlowConflict
	}
	p, e := filepath.Abs(s.Path)
	if e != nil || s.Path == "" {
		return errors.New("invalid flow state path")
	}
	if e = os.MkdirAll(filepath.Dir(p), 0750); e != nil {
		return e
	}
	data, e := json.MarshalIndent(normalizeFlowState(state), "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(p), ".flow-state-*")
	if e != nil {
		return e
	}
	n := f.Name()
	defer os.Remove(n)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(data)
	}
	if e == nil {
		e = f.Sync()
	}
	if ce := f.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(n, p)
}
func (s *FileFlowStateStore) load() (FlowState, error) {
	if s == nil || s.Path == "" {
		return FlowState{}, errors.New("invalid flow state path")
	}
	data, e := os.ReadFile(s.Path)
	if errors.Is(e, os.ErrNotExist) {
		return normalizeFlowState(FlowState{}), nil
	}
	if e != nil {
		return FlowState{}, e
	}
	var state FlowState
	if e = json.Unmarshal(data, &state); e != nil {
		return FlowState{}, e
	}
	return normalizeFlowState(state), nil
}

var _ FlowStateStore = (*FileFlowStateStore)(nil)
