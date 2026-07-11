package authoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileAssetStateStore persists asset records atomically. Its compare-and-swap
// revision prevents two hosts from silently replacing each other's metadata.
type FileAssetStateStore struct {
	Path string
	mu   sync.Mutex
}

func (s *FileAssetStateStore) LoadAssetState(context.Context) (AssetState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *FileAssetStateStore) SaveAssetState(_ context.Context, state AssetState, expected uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.load()
	if err != nil {
		return err
	}
	if current.Revision != expected {
		return ErrAssetConflict
	}
	path, err := s.safePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(normalizeAssetState(state), "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".asset-state-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

func (s *FileAssetStateStore) load() (AssetState, error) {
	path, err := s.safePath()
	if err != nil {
		return AssetState{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return normalizeAssetState(AssetState{}), nil
	}
	if err != nil {
		return AssetState{}, err
	}
	var state AssetState
	if err := json.Unmarshal(data, &state); err != nil {
		return AssetState{}, err
	}
	return normalizeAssetState(state), nil
}

func (s *FileAssetStateStore) safePath() (string, error) {
	if s == nil || s.Path == "" {
		return "", fmt.Errorf("%w: asset state path is required", ErrAssetInvalid)
	}
	path, err := filepath.Abs(s.Path)
	if err != nil || filepath.Base(path) == "." {
		return "", fmt.Errorf("%w: invalid asset state path", ErrAssetInvalid)
	}
	return path, nil
}

var _ AssetStateStore = (*FileAssetStateStore)(nil)
