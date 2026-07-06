package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/cms/store/memory"
)

type Option func(*options)

type options struct {
	memoryOptions []memory.Option
}

type Snapshot struct {
	Settings    cmsstore.SiteSettings `json:"settings"`
	HasSettings bool                  `json:"hasSettings"`
	Pages       []cmsstore.Page       `json:"pages"`
	Posts       []cmsstore.Post       `json:"posts"`
	Revisions   []lifecycle.Revision  `json:"revisions"`
}

type Store struct {
	mu    sync.Mutex
	path  string
	store *memory.Store
}

var _ cmsstore.Store = (*Store)(nil)

func New(path string, seed memory.Seed, opts ...Option) (*Store, error) {
	config := applyOptions(opts)
	store, err := newStore(path, seed, config)
	if err != nil {
		return nil, err
	}
	if err := store.persistLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

func Open(path string, opts ...Option) (*Store, error) {
	config := applyOptions(opts)
	seed, err := load(path)
	if err != nil {
		return nil, err
	}
	return newStore(path, seed, config)
}

func WithClock(clock memory.Clock) Option {
	return func(opts *options) {
		opts.memoryOptions = append(opts.memoryOptions, memory.WithClock(clock))
	}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Snapshot() memory.Seed {
	return s.store.Snapshot()
}

func (s *Store) ListPages(filter cmsstore.PageFilter) ([]cmsstore.Page, error) {
	return s.store.ListPages(filter)
}

func (s *Store) PageByID(id string) (cmsstore.Page, bool, error) {
	return s.store.PageByID(id)
}

func (s *Store) PageBySlug(slug string) (cmsstore.Page, bool, error) {
	return s.store.PageBySlug(slug)
}

func (s *Store) CreatePage(input cmsstore.PageInput) (cmsstore.Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, err := s.store.CreatePage(input)
	if err != nil {
		return cmsstore.Page{}, err
	}
	return page, s.persistLocked()
}

func (s *Store) UpdatePage(id string, input cmsstore.PageInput) (cmsstore.Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	page, err := s.store.UpdatePage(id, input)
	if err != nil {
		return cmsstore.Page{}, err
	}
	return page, s.persistLocked()
}

func (s *Store) ListPosts(filter cmsstore.PostFilter) ([]cmsstore.Post, error) {
	return s.store.ListPosts(filter)
}

func (s *Store) PostByID(id string) (cmsstore.Post, bool, error) {
	return s.store.PostByID(id)
}

func (s *Store) PostBySlug(slug string) (cmsstore.Post, bool, error) {
	return s.store.PostBySlug(slug)
}

func (s *Store) CreatePost(input cmsstore.PostInput) (cmsstore.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	post, err := s.store.CreatePost(input)
	if err != nil {
		return cmsstore.Post{}, err
	}
	return post, s.persistLocked()
}

func (s *Store) UpdatePost(id string, input cmsstore.PostInput) (cmsstore.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	post, err := s.store.UpdatePost(id, input)
	if err != nil {
		return cmsstore.Post{}, err
	}
	return post, s.persistLocked()
}

func (s *Store) SiteSettings() (cmsstore.SiteSettings, bool, error) {
	return s.store.SiteSettings()
}

func (s *Store) SaveSiteSettings(input cmsstore.SiteSettingsInput) (cmsstore.SiteSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	settings, err := s.store.SaveSiteSettings(input)
	if err != nil {
		return cmsstore.SiteSettings{}, err
	}
	return settings, s.persistLocked()
}

func (s *Store) ListRevisions(filter lifecycle.RevisionFilter) []lifecycle.Revision {
	return s.store.ListRevisions(filter)
}

func (s *Store) RevisionByID(resourceKind, resourceID, revisionID string) (lifecycle.Revision, bool) {
	return s.store.RevisionByID(resourceKind, resourceID, revisionID)
}

func (s *Store) SaveRevision(input lifecycle.RevisionInput) (lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	revision, err := s.store.SaveRevision(input)
	if err != nil {
		return lifecycle.Revision{}, err
	}
	return revision, s.persistLocked()
}

func (s *Store) persistLocked() error {
	return save(s.path, s.store.Snapshot())
}

func newStore(path string, seed memory.Seed, config options) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("file store path is required")
	}
	return &Store{
		path:  path,
		store: memory.New(seed, config.memoryOptions...),
	}, nil
}

func applyOptions(opts []Option) options {
	var config options
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return config
}

func load(path string) (memory.Seed, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return memory.Seed{}, fmt.Errorf("file store path is required")
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return memory.Seed{}, nil
	}
	if err != nil {
		return memory.Seed{}, fmt.Errorf("open file store snapshot: %w", err)
	}
	defer file.Close()

	var snapshot Snapshot
	if err := json.NewDecoder(file).Decode(&snapshot); err != nil {
		if errors.Is(err, io.EOF) {
			return memory.Seed{}, nil
		}
		return memory.Seed{}, fmt.Errorf("decode file store snapshot: %w", err)
	}
	return seedFromSnapshot(snapshot), nil
}

func save(path string, seed memory.Seed) error {
	snapshot := snapshotFromSeed(seed)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create file store directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create file store temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode file store snapshot: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync file store temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close file store temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace file store snapshot: %w", err)
	}
	cleanup = false
	return syncDir(dir)
}

func syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open file store directory: %w", err)
	}
	defer handle.Close()
	if err := handle.Sync(); err != nil {
		return fmt.Errorf("sync file store directory: %w", err)
	}
	return nil
}

func snapshotFromSeed(seed memory.Seed) Snapshot {
	return Snapshot{
		Settings:    cmsstore.CloneSiteSettings(seed.Settings),
		HasSettings: seed.HasSettings,
		Pages:       cmsstore.ClonePages(seed.Pages),
		Posts:       cmsstore.ClonePosts(seed.Posts),
		Revisions:   lifecycle.CloneRevisions(seed.Revisions),
	}
}

func seedFromSnapshot(snapshot Snapshot) memory.Seed {
	return memory.Seed{
		Settings:    cmsstore.CloneSiteSettings(snapshot.Settings),
		HasSettings: snapshot.HasSettings,
		Pages:       cmsstore.ClonePages(snapshot.Pages),
		Posts:       cmsstore.ClonePosts(snapshot.Posts),
		Revisions:   lifecycle.CloneRevisions(snapshot.Revisions),
	}
}
